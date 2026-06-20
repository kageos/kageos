package inventory

import (
	"errors"
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/statistics"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 销售出库（Form） ================

type SaleOutboundReq struct {
	ProductName string  `json:"product_name" widget:"name:商品名称;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Qty         int     `json:"qty" widget:"name:数量;type:integer" validate:"required,min=1"`
	Price       float64 `json:"price" widget:"name:单价;type:float;precision:2;step:0.01;unit:元" validate:"required"`
}

type SaleOutboundResp struct {
	OrderNo      string     `json:"order_no" widget:"name:销售单号;type:input"`
	TotalAmount  float64    `json:"total_amount" widget:"name:实付金额;type:float;precision:2;unit:元"`
	OutboundTime types.Time `json:"outbound_time" widget:"name:出库时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	Result       string     `json:"result" widget:"name:销售结果;type:input"`
}

func SaleOutbound(ctx *app.Context, resp response.Response) error {
	var req SaleOutboundReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()

	var product Product
	if err := db.Where("name = ?", req.ProductName).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品 %s 不存在，请先创建商品", req.ProductName)
		}
		return fmt.Errorf("[系统错误]-[SaleOutbound] 查询商品失败, req: %+v, err: %w", req, err)
	}

	if product.StockQty < req.Qty {
		return fmt.Errorf("商品 %s 库存不足（当前库存 %d，需要出库 %d），无法出库", req.ProductName, product.StockQty, req.Qty)
	}

	totalAmount := float64(req.Qty) * req.Price
	now := time.Now()
	orderNo := fmt.Sprintf("XS%s%04d", now.Format("20060102"), product.ID)

	record := SalesRecord{
		OrderNo:      orderNo,
		ProductName:  req.ProductName,
		Qty:          req.Qty,
		Price:        req.Price,
		TotalAmount:  totalAmount,
		OutboundTime: types.Time(now),
		CreatedBy:    ctx.GetRequestUser(),
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		if err := tx.Model(&Product{}).Where("id = ?", product.ID).
			Update("stock_qty", gorm.Expr("stock_qty - ?", req.Qty)).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[SaleOutbound] 销售出库事务失败, req: %+v, err: %v", req, err)
		return fmt.Errorf("[系统错误]-[SaleOutbound] 销售出库失败, req: %+v, err: %w", req, err)
	}

	return resp.Form(&SaleOutboundResp{
		OrderNo:      orderNo,
		TotalAmount:  totalAmount,
		OutboundTime: types.Time(now),
		Result:       "出库成功",
	}).Build()
}

var SaleOutboundTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "销售出库",
		Request:  &SaleOutboundReq{},
		Response: &SaleOutboundResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"product_name": onSelectFuzzyProductNameForSale,
		},
	},
}

func onSelectFuzzyProductNameForSale(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	var products []Product
	if req.IsByValue() {
		db.Where("name = ?", req.GetValue()).Limit(1).Find(&products)
	} else if req.IsByValues() {
		db.Where("name in ?", req.GetValues()).Find(&products)
	} else {
		db.Where("name LIKE ?", "%"+req.Keyword()+"%").Limit(20).Find(&products)
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(products))
	for _, p := range products {
		items = append(items, &callback.SelectFuzzyItem{
			Value: p.Name,
			Label: p.Name,
			DisplayInfo: map[string]interface{}{
				"分类": p.Category,
				"单价": p.Price,
				"库存": p.StockQty,
				"单位": p.Unit,
			},
		})
	}
	return &callback.OnSelectFuzzyResp{
		Items: items,
		Statistics: map[string]interface{}{
			"选中商品": statistics.Value("分类"),
			"单价":   statistics.Value("单价"),
			"库存":   statistics.Value("库存"),
		},
	}, nil
}

func init() {
	packageContext.POST("sale_outbound.form", SaleOutbound, SaleOutboundTemplate)
}
