package inventory

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// 销售出库请求结构体
type SalesOutboundReq struct {
	CustomerID   int     `json:"customer_id" widget:"name:客户;type:select" validate:"required" callback:"OnSelectFuzzy"`
	CustomerName string  `json:"customer_name" gorm:"-" widget:"-"`
	ProductID    int     `json:"product_id" widget:"name:商品;type:select" validate:"required" callback:"OnSelectFuzzy"`
	ProductName  string  `json:"product_name" gorm:"-" widget:"-"`
	Quantity     int     `json:"quantity" widget:"name:销售数量;type:integer;min:1;step:1" validate:"required,gte=1"`
	SalesPrice   float64 `json:"sales_price" widget:"name:销售单价;type:float;min:0;precision:2;step:0.01;unit:元" validate:"required,gte=0"`
	Warehouse    string  `json:"warehouse" widget:"name:出库仓库;type:select;options:A仓,B仓,C仓;options_colors:67C23A,409EFF,E6A23C" validate:"required"`
	OutboundTime string  `json:"outbound_time" widget:"name:出库时间;type:datetime;format:YYYY-MM-DD HH:mm:ss;render_default:CURRENT_TIMESTAMP"`
}

// 销售出库响应结构体
type SalesOutboundResp struct {
	OutboundNo   string  `json:"outbound_no" widget:"name:出库单号;type:input"`
	Result       string  `json:"result" widget:"name:结果;type:input"`
	CurrentStock int     `json:"current_stock" widget:"name:当前库存;type:integer"`
	Profit       float64 `json:"profit" widget:"name:毛利;type:float;precision:2;unit:元"`
}

// 生成出库单号 XS + 日期 + 序号
func generateOutboundNo(db *gorm.DB) string {
	dateStr := time.Now().Format("20060102")
	var count int64
	db.Model(&SalesOutbound{}).Where("outbound_no LIKE ?", "XS"+dateStr+"%").Count(&count)
	seq := count + 1
	return fmt.Sprintf("XS%s%03d", dateStr, seq)
}

func SalesOutboundForm(ctx *app.Context, resp response.Response) error {
	var req SalesOutboundReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()

	// 事务处理
	var resultResp SalesOutboundResp
	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. 获取客户名称
		var customer Customer
		if err := tx.First(&customer, req.CustomerID).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[SalesOutbound] 获取客户失败, req: %+v, err: %v", req, err)
			return fmt.Errorf("[系统错误] 客户不存在")
		}

		// 2. 获取商品信息（含采购单价用于计算毛利）
		var product Product
		if err := tx.First(&product, req.ProductID).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[SalesOutbound] 获取商品失败, req: %+v, err: %v", req, err)
			return fmt.Errorf("[系统错误] 商品不存在")
		}

		// 3. 检查库存是否充足
		var ledger InventoryLedger
		if err := tx.Where("product_name = ?", product.ProductName).First(&ledger).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[SalesOutbound] 库存记录不存在, req: %+v, err: %v", req, err)
			return fmt.Errorf("[系统错误] 库存不足，无法出库")
		}
		if ledger.StockQty < req.Quantity {
			return fmt.Errorf("库存不足：当前库存 %d，需要出库 %d", ledger.StockQty, req.Quantity)
		}

		// 4. 生成出库单号
		outboundNo := generateOutboundNo(tx)

		// 5. 计算出库时间
		outboundTime := time.Now()
		if req.OutboundTime != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", req.OutboundTime); err == nil {
				outboundTime = t
			}
		}

		// 6. 计算销售额和毛利
		// 毛利 = (销售单价 - 采购单价) × 销售数量
		salesAmount := float64(req.Quantity) * req.SalesPrice
		profit := (req.SalesPrice - product.PurchasePrice) * float64(req.Quantity)

		// 7. 创建销售出库记录
		salesRecord := SalesOutbound{
			OutboundNo:   outboundNo,
			OutboundTime: types.Time(outboundTime),
			CustomerName: customer.CustomerName,
			ProductName:  product.ProductName,
			Quantity:     req.Quantity,
			SalesPrice:   req.SalesPrice,
			SalesAmount:  salesAmount,
			Profit:       profit,
			Warehouse:    req.Warehouse,
			CreatedBy:    ctx.GetRequestUser(),
		}
		if err := tx.Create(&salesRecord).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[SalesOutbound] 创建出库记录失败, req: %+v, err: %v", req, err)
			return fmt.Errorf("[系统错误] 创建出库记录失败")
		}

		// 8. 扣减库存
		newQty := ledger.StockQty - req.Quantity
		stockStatus := "正常"
		if newQty < product.SafeStock {
			stockStatus = "预警"
		}
		if err := tx.Model(&ledger).Updates(map[string]interface{}{
			"stock_qty":    newQty,
			"stock_status": stockStatus,
		}).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[SalesOutbound] 扣减库存失败, req: %+v, err: %v", req, err)
			return fmt.Errorf("[系统错误] 扣减库存失败")
		}

		// 9. 返回结果
		resultResp.OutboundNo = outboundNo
		resultResp.Result = "出库成功"
		resultResp.CurrentStock = newQty
		resultResp.Profit = profit
		return nil
	})

	if err != nil {
		return fmt.Errorf("%v", err)
	}

	return resp.Form(&resultResp).Build()
}

// 销售出库下拉模糊搜索 - 客户
func salesOutboundCustomerFuzzy(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	var customers []Customer
	query := db.Model(&Customer{}).Where("status = ?", "正常")
	if req.IsByValue() {
		query = query.Where("id = ?", req.GetValue())
	} else if req.IsByValues() {
		query = query.Where("id in ?", req.GetValues())
	} else {
		query = query.Where("customer_name LIKE ?", "%"+req.Keyword()+"%")
	}
	query.Limit(20).Find(&customers)

	items := make([]*callback.SelectFuzzyItem, 0, len(customers))
	for _, c := range customers {
		items = append(items, &callback.SelectFuzzyItem{
			Value: c.ID, Label: c.CustomerName,
			DisplayInfo: map[string]interface{}{"联系人": c.Contact, "手机": c.Phone},
		})
	}
	return &callback.OnSelectFuzzyResp{Items: items}, nil
}

// 销售出库下拉模糊搜索 - 商品
func salesOutboundProductFuzzy(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	var products []Product
	query := db.Model(&Product{}).Where("listing_status = ?", "上架")
	if req.IsByValue() {
		query = query.Where("id = ?", req.GetValue())
	} else if req.IsByValues() {
		query = query.Where("id in ?", req.GetValues())
	} else {
		query = query.Where("product_name LIKE ?", "%"+req.Keyword()+"%")
	}
	query.Limit(20).Find(&products)

	items := make([]*callback.SelectFuzzyItem, 0, len(products))
	for _, p := range products {
		items = append(items, &callback.SelectFuzzyItem{
			Value: p.ID, Label: p.ProductName,
			DisplayInfo: map[string]interface{}{"商品分类": p.Category, "单位": p.Unit, "采购单价": p.PurchasePrice, "零售单价": p.RetailPrice},
		})
	}
	return &callback.OnSelectFuzzyResp{Items: items}, nil
}

var SalesOutboundFormTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "销售出库",
		Request:  &SalesOutboundReq{},
		Response: &SalesOutboundResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"customer_id": salesOutboundCustomerFuzzy,
			"product_id":  salesOutboundProductFuzzy,
		},
	},
}

func init() {
	packageContext.POST("sales_outbound.form", SalesOutboundForm, SalesOutboundFormTemplate)
}
