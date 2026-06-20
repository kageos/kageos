package inventory

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
)

type SalesItemLine struct {
	ProductID int     `json:"product_id" widget:"name:商品;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Quantity  int     `json:"quantity" widget:"name:数量;type:integer;min:1" validate:"required,min=1"`
	UnitPrice float64 `json:"unit_price" widget:"name:单价;type:float;precision:2;unit:元" validate:"required,min=0"`
}

type SalesSubmitReq struct {
	CustomerID int             `json:"customer_id" widget:"name:客户;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Items      []SalesItemLine `json:"items" widget:"name:商品明细;type:table" validate:"required,min=1"`
	SalesTime  types.Time      `json:"sales_time" widget:"name:出库时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`
	Remark     string          `json:"remark" widget:"name:备注;type:text_area"`
}

type SalesSubmitResp struct {
	OrderNo     string  `json:"order_no" widget:"name:出库单号;type:input"`
	Result      string  `json:"result" widget:"name:处理结果;type:input"`
	TotalAmount float64 `json:"total_amount" widget:"name:销售总额;type:float;precision:2;unit:元"`
}

var SalesSubmitTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "销售出库",
		Request:  &SalesSubmitReq{},
		Response: &SalesSubmitResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"customer_id": onSelectFuzzyCustomer,
			"product_id":  onSelectFuzzyProductForSales,
		},
	},
}

func SalesSubmit(ctx *app.Context, resp response.Response) error {
	var req SalesSubmitReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()

	// 生成出库单号
	now := time.Now()
	dateStr := now.Format("20060102")
	var count int64
	db.Model(&SalesOrder{}).Where("order_no LIKE ?", "SO"+dateStr+"%").Count(&count)
	orderNo := fmt.Sprintf("SO%s%03d", dateStr, count+1)

	// 查询客户
	var customer Customer
	if err := db.First(&customer, req.CustomerID).Error; err != nil {
		return fmt.Errorf("客户不存在")
	}

	// 校验库存、计算总额并构建明细
	var totalAmount float64
	var itemsDesc string
	type stockCheck struct {
		product  Product
		quantity int
	}
	var checks []stockCheck

	for i, item := range req.Items {
		var product Product
		if err := db.First(&product, item.ProductID).Error; err != nil {
			return fmt.Errorf("商品ID %d 不存在", item.ProductID)
		}
		if product.CurrentStock < item.Quantity {
			return fmt.Errorf("商品[%s]库存不足，当前库存 %d，需要 %d", product.ProductName, product.CurrentStock, item.Quantity)
		}
		checks = append(checks, stockCheck{product: product, quantity: item.Quantity})
		totalAmount += float64(item.Quantity) * item.UnitPrice
		if i > 0 {
			itemsDesc += "，"
		}
		itemsDesc += fmt.Sprintf("%s×%d@%.2f元", product.ProductName, item.Quantity, item.UnitPrice)
	}

	// 创建销售出库单
	order := SalesOrder{
		OrderNo:      orderNo,
		CustomerID:   req.CustomerID,
		CustomerName: customer.CustomerName,
		SalesItems:   itemsDesc,
		TotalAmount:  totalAmount,
		SalesTime:    req.SalesTime,
		Remark:       req.Remark,
		Status:       "已出库",
	}
	if err := db.Create(&order).Error; err != nil {
		return fmt.Errorf("[系统错误]-[SalesSubmit] 创建出库单失败, req: %+v, err: %w", req, err)
	}

	// 扣减库存并生成流水
	for _, sc := range checks {
		beforeStock := sc.product.CurrentStock
		afterStock := beforeStock - sc.quantity

		if err := db.Model(&Product{}).Where("id = ?", sc.product.ID).Update("current_stock", afterStock).Error; err != nil {
			return fmt.Errorf("[系统错误]-[SalesSubmit] 更新库存失败, product_id: %d, err: %w", sc.product.ID, err)
		}

		var txCount int64
		db.Model(&InventoryTransaction{}).Where("transaction_no LIKE ?", "FL"+dateStr+"%").Count(&txCount)
		txNo := fmt.Sprintf("FL%s%03d", dateStr, txCount+1)

		tx := InventoryTransaction{
			TransactionNo:   txNo,
			ProductID:       sc.product.ID,
			ProductName:     sc.product.ProductName,
			ProductCode:     sc.product.ProductCode,
			TransactionType: "销售出库",
			Quantity:        -sc.quantity,
			BeforeStock:     beforeStock,
			AfterStock:      afterStock,
			RelatedOrderNo:  orderNo,
			TransactionTime: req.SalesTime,
		}
		if err := db.Create(&tx).Error; err != nil {
			return fmt.Errorf("[系统错误]-[SalesSubmit] 创建库存流水失败, product_id: %d, err: %w", sc.product.ID, err)
		}
	}

	return resp.Form(&SalesSubmitResp{
		OrderNo:     orderNo,
		Result:      "出库成功",
		TotalAmount: totalAmount,
	}).Build()
}

func onSelectFuzzyProductForSales(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	var products []Product
	queryDB := db.Model(&Product{}).Where("status = ?", "正常")
	if req.IsByValue() {
		queryDB = queryDB.Where("id = ?", req.GetValue())
	} else if req.IsByValues() {
		queryDB = queryDB.Where("id IN ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		if keyword != "" {
			queryDB = queryDB.Where("product_name LIKE ? OR product_code LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}
	}
	if err := queryDB.Limit(20).Find(&products).Error; err != nil {
		return nil, err
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(products))
	for _, p := range products {
		items = append(items, &callback.SelectFuzzyItem{
			Value: p.ID,
			Label: fmt.Sprintf("%s(%s)", p.ProductName, p.ProductCode),
			DisplayInfo: map[string]interface{}{
				"销售单价": p.SalePrice,
				"当前库存": p.CurrentStock,
				"单位":   p.Unit,
			},
		})
	}
	return &callback.OnSelectFuzzyResp{
		Items:         items,
		MaxSelections: 1,
	}, nil
}

func init() {
	packageContext.POST("sales_submit.form", SalesSubmit, SalesSubmitTemplate)
}
