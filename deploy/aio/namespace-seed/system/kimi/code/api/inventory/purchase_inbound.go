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

// ================ 采购入库（Form） ================

type PurchaseInboundReq struct {
	ProductName  string  `json:"product_name" widget:"name:商品名称;type:select" validate:"required" callback:"OnSelectFuzzy"`
	SupplierName string  `json:"supplier_name" widget:"name:供应商名称;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Qty          int     `json:"qty" widget:"name:数量;type:integer" validate:"required,min=1"`
	Price        float64 `json:"price" widget:"name:单价;type:float;precision:2;step:0.01;unit:元" validate:"required"`
}

type PurchaseInboundResp struct {
	OrderNo     string     `json:"order_no" widget:"name:采购单号;type:input"`
	TotalAmount float64    `json:"total_amount" widget:"name:总价;type:float;precision:2;unit:元"`
	InboundTime types.Time `json:"inbound_time" widget:"name:入库时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	Result      string     `json:"result" widget:"name:采购结果;type:input"`
}

func PurchaseInbound(ctx *app.Context, resp response.Response) error {
	var req PurchaseInboundReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()

	var product Product
	if err := db.Where("name = ?", req.ProductName).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("商品 %s 不存在，请先创建商品", req.ProductName)
		}
		return fmt.Errorf("[系统错误]-[PurchaseInbound] 查询商品失败, req: %+v, err: %w", req, err)
	}

	var supplier Supplier
	if err := db.Where("name = ?", req.SupplierName).First(&supplier).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("供应商 %s 不存在，请先创建供应商", req.SupplierName)
		}
		return fmt.Errorf("[系统错误]-[PurchaseInbound] 查询供应商失败, req: %+v, err: %w", req, err)
	}

	totalAmount := float64(req.Qty) * req.Price
	now := time.Now()
	orderNo := fmt.Sprintf("CG%s%04d", now.Format("20060102"), product.ID)

	record := PurchaseRecord{
		OrderNo:      orderNo,
		ProductName:  req.ProductName,
		SupplierName: req.SupplierName,
		Qty:          req.Qty,
		Price:        req.Price,
		TotalAmount:  totalAmount,
		InboundTime:  types.Time(now),
		CreatedBy:    ctx.GetRequestUser(),
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		if err := tx.Model(&Product{}).Where("id = ?", product.ID).
			Update("stock_qty", gorm.Expr("stock_qty + ?", req.Qty)).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[PurchaseInbound] 采购入库事务失败, req: %+v, err: %v", req, err)
		return fmt.Errorf("[系统错误]-[PurchaseInbound] 采购入库失败, req: %+v, err: %w", req, err)
	}

	return resp.Form(&PurchaseInboundResp{
		OrderNo:     orderNo,
		TotalAmount: totalAmount,
		InboundTime: types.Time(now),
		Result:      "入库成功",
	}).Build()
}

var PurchaseInboundTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "采购入库",
		Request:  &PurchaseInboundReq{},
		Response: &PurchaseInboundResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"product_name":  onSelectFuzzyProductName,
			"supplier_name": onSelectFuzzySupplierName,
		},
	},
}

func onSelectFuzzyProductName(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
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

func onSelectFuzzySupplierName(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	var suppliers []Supplier
	if req.IsByValue() {
		db.Where("name = ?", req.GetValue()).Limit(1).Find(&suppliers)
	} else if req.IsByValues() {
		db.Where("name in ?", req.GetValues()).Find(&suppliers)
	} else {
		db.Where("name LIKE ?", "%"+req.Keyword()+"%").Limit(20).Find(&suppliers)
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(suppliers))
	for _, s := range suppliers {
		items = append(items, &callback.SelectFuzzyItem{
			Value: s.Name,
			Label: s.Name,
			DisplayInfo: map[string]interface{}{
				"联系人": s.Contact,
				"电话":  s.Phone,
			},
		})
	}
	return &callback.OnSelectFuzzyResp{
		Items: items,
		Statistics: map[string]interface{}{
			"联系人": statistics.Value("联系人"),
			"电话":  statistics.Value("电话"),
		},
	}, nil
}

func init() {
	packageContext.POST("purchase_inbound.form", PurchaseInbound, PurchaseInboundTemplate)
}
