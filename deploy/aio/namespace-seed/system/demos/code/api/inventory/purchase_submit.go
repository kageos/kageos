package inventory

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
)

type PurchaseItemLine struct {
	ProductID int     `json:"product_id" widget:"name:商品;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Quantity  int     `json:"quantity" widget:"name:数量;type:integer;min:1" validate:"required,min=1"`
	UnitPrice float64 `json:"unit_price" widget:"name:单价;type:float;precision:2;unit:元" validate:"required,min=0"`
}

type PurchaseSubmitReq struct {
	SupplierID   int                `json:"supplier_id" widget:"name:供应商;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Items        []PurchaseItemLine `json:"items" widget:"name:商品明细;type:table" validate:"required,min=1"`
	PurchaseTime types.Time         `json:"purchase_time" widget:"name:入库时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`
	Remark       string             `json:"remark" widget:"name:备注;type:text_area"`
}

type PurchaseSubmitResp struct {
	OrderNo     string  `json:"order_no" widget:"name:入库单号;type:input"`
	Result      string  `json:"result" widget:"name:处理结果;type:input"`
	TotalAmount float64 `json:"total_amount" widget:"name:采购总额;type:float;precision:2;unit:元"`
}

var PurchaseSubmitTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "采购入库",
		Request:  &PurchaseSubmitReq{},
		Response: &PurchaseSubmitResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"supplier_id": onSelectFuzzySupplier,
			"product_id":  onSelectFuzzyProductForPurchase,
		},
	},
}

func PurchaseSubmit(ctx *app.Context, resp response.Response) error {
	var req PurchaseSubmitReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()

	// 生成入库单号
	now := time.Now()
	dateStr := now.Format("20060102")
	var count int64
	db.Model(&PurchaseOrder{}).Where("order_no LIKE ?", "PO"+dateStr+"%").Count(&count)
	orderNo := fmt.Sprintf("PO%s%03d", dateStr, count+1)

	// 查询供应商
	var supplier Supplier
	if err := db.First(&supplier, req.SupplierID).Error; err != nil {
		return fmt.Errorf("供应商不存在")
	}

	// 计算总额并构建明细
	var totalAmount float64
	var itemsDesc string
	for i, item := range req.Items {
		var product Product
		if err := db.First(&product, item.ProductID).Error; err != nil {
			return fmt.Errorf("商品ID %d 不存在", item.ProductID)
		}
		totalAmount += float64(item.Quantity) * item.UnitPrice
		if i > 0 {
			itemsDesc += "，"
		}
		itemsDesc += fmt.Sprintf("%s×%d@%.2f元", product.ProductName, item.Quantity, item.UnitPrice)
	}

	// 创建采购入库单
	order := PurchaseOrder{
		OrderNo:       orderNo,
		SupplierID:    req.SupplierID,
		SupplierName:  supplier.SupplierName,
		PurchaseItems: itemsDesc,
		TotalAmount:   totalAmount,
		PurchaseTime:  req.PurchaseTime,
		Remark:        req.Remark,
		Status:        "已入库",
	}
	if err := db.Create(&order).Error; err != nil {
		return fmt.Errorf("[系统错误]-[PurchaseSubmit] 创建入库单失败, req: %+v, err: %w", req, err)
	}

	// 更新库存并生成流水
	for _, item := range req.Items {
		var product Product
		if err := db.First(&product, item.ProductID).Error; err != nil {
			continue
		}
		beforeStock := product.CurrentStock
		afterStock := beforeStock + item.Quantity

		// 更新库存
		if err := db.Model(&Product{}).Where("id = ?", product.ID).Update("current_stock", afterStock).Error; err != nil {
			return fmt.Errorf("[系统错误]-[PurchaseSubmit] 更新库存失败, product_id: %d, err: %w", product.ID, err)
		}

		// 生成流水号
		var txCount int64
		db.Model(&InventoryTransaction{}).Where("transaction_no LIKE ?", "FL"+dateStr+"%").Count(&txCount)
		txNo := fmt.Sprintf("FL%s%03d", dateStr, txCount+1)

		// 创建库存流水
		tx := InventoryTransaction{
			TransactionNo:   txNo,
			ProductID:       product.ID,
			ProductName:     product.ProductName,
			ProductCode:     product.ProductCode,
			TransactionType: "采购入库",
			Quantity:        item.Quantity,
			BeforeStock:     beforeStock,
			AfterStock:      afterStock,
			RelatedOrderNo:  orderNo,
			TransactionTime: req.PurchaseTime,
		}
		if err := db.Create(&tx).Error; err != nil {
			return fmt.Errorf("[系统错误]-[PurchaseSubmit] 创建库存流水失败, product_id: %d, err: %w", product.ID, err)
		}
	}

	return resp.Form(&PurchaseSubmitResp{
		OrderNo:     orderNo,
		Result:      "入库成功",
		TotalAmount: totalAmount,
	}).Build()
}

func onSelectFuzzyProductForPurchase(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
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
				"采购单价": p.PurchasePrice,
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
	packageContext.POST("purchase_submit.form", PurchaseSubmit, PurchaseSubmitTemplate)
}
