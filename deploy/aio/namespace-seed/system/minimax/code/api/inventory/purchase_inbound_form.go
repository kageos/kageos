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

// 采购入库请求结构体
type PurchaseInboundReq struct {
	SupplierID    int     `json:"supplier_id" widget:"name:供应商;type:select" validate:"required" callback:"OnSelectFuzzy"`
	SupplierName  string  `json:"supplier_name" gorm:"-" widget:"-"`
	ProductID     int     `json:"product_id" widget:"name:商品;type:select" validate:"required" callback:"OnSelectFuzzy"`
	ProductName   string  `json:"product_name" gorm:"-" widget:"-"`
	Quantity      int     `json:"quantity" widget:"name:采购数量;type:integer;min:1;step:1" validate:"required,gte=1"`
	PurchasePrice float64 `json:"purchase_price" widget:"name:采购单价;type:float;min:0;precision:2;step:0.01;unit:元" validate:"required,gte=0"`
	Warehouse     string  `json:"warehouse" widget:"name:入库仓库;type:select;options:A仓,B仓,C仓;options_colors:67C23A,409EFF,E6A23C" validate:"required"`
	InboundTime   string  `json:"inbound_time" widget:"name:入库时间;type:datetime;format:YYYY-MM-DD HH:mm:ss;render_default:CURRENT_TIMESTAMP"`
}

// 采购入库响应结构体
type PurchaseInboundResp struct {
	InboundNo    string `json:"inbound_no" widget:"name:入库单号;type:input"`
	Result       string `json:"result" widget:"name:结果;type:input"`
	CurrentStock int    `json:"current_stock" widget:"name:当前库存;type:integer"`
}

// 生成入库单号 CG + 日期 + 序号
func generateInboundNo(db *gorm.DB) string {
	dateStr := time.Now().Format("20060102")
	var count int64
	db.Model(&PurchaseInbound{}).Where("inbound_no LIKE ?", "CG"+dateStr+"%").Count(&count)
	seq := count + 1
	return fmt.Sprintf("CG%s%03d", dateStr, seq)
}

func PurchaseInboundForm(ctx *app.Context, resp response.Response) error {
	var req PurchaseInboundReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()

	// 事务处理
	var resultResp PurchaseInboundResp
	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. 获取供应商名称
		var supplier Supplier
		if err := tx.First(&supplier, req.SupplierID).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[PurchaseInbound] 获取供应商失败, req: %+v, err: %v", req, err)
			return fmt.Errorf("[系统错误] 供应商不存在")
		}

		// 2. 获取商品信息
		var product Product
		if err := tx.First(&product, req.ProductID).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[PurchaseInbound] 获取商品失败, req: %+v, err: %v", req, err)
			return fmt.Errorf("[系统错误] 商品不存在")
		}

		// 3. 生成入库单号
		inboundNo := generateInboundNo(tx)

		// 4. 计算入库时间
		inboundTime := time.Now()
		if req.InboundTime != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", req.InboundTime); err == nil {
				inboundTime = t
			}
		}

		// 5. 创建采购入库记录
		purchaseRecord := PurchaseInbound{
			InboundNo:     inboundNo,
			InboundTime:   types.Time(inboundTime),
			SupplierName:  supplier.SupplierName,
			ProductName:   product.ProductName,
			Quantity:      req.Quantity,
			PurchasePrice: req.PurchasePrice,
			Amount:        float64(req.Quantity) * req.PurchasePrice,
			Warehouse:     req.Warehouse,
			CreatedBy:     ctx.GetRequestUser(),
		}
		if err := tx.Create(&purchaseRecord).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[PurchaseInbound] 创建入库记录失败, req: %+v, err: %v", req, err)
			return fmt.Errorf("[系统错误] 创建入库记录失败")
		}

		// 6. 更新或创建库存台账
		var ledger InventoryLedger
		err := tx.Where("product_name = ?", product.ProductName).First(&ledger).Error
		if err == gorm.ErrRecordNotFound {
			// 新增库存记录
			stockStatus := "正常"
			if req.Quantity < product.SafeStock {
				stockStatus = "预警"
			}
			ledger = InventoryLedger{
				ProductName: product.ProductName,
				Category:    product.Category,
				Unit:        product.Unit,
				StockQty:    req.Quantity,
				SafeStock:   product.SafeStock,
				StockStatus: stockStatus,
				CreatedBy:   ctx.GetRequestUser(),
			}
			if err := tx.Create(&ledger).Error; err != nil {
				logger.Errorf(ctx, "[系统错误]-[PurchaseInbound] 创建库存记录失败, req: %+v, err: %v", req, err)
				return fmt.Errorf("[系统错误] 创建库存记录失败")
			}
		} else if err != nil {
			logger.Errorf(ctx, "[系统错误]-[PurchaseInbound] 查询库存记录失败, req: %+v, err: %v", req, err)
			return fmt.Errorf("[系统错误] 查询库存记录失败")
		} else {
			// 更新库存数量
			newQty := ledger.StockQty + req.Quantity
			stockStatus := "正常"
			if newQty < product.SafeStock {
				stockStatus = "预警"
			}
			if err := tx.Model(&ledger).Updates(map[string]interface{}{
				"stock_qty":    newQty,
				"stock_status": stockStatus,
			}).Error; err != nil {
				logger.Errorf(ctx, "[系统错误]-[PurchaseInbound] 更新库存失败, req: %+v, err: %v", req, err)
				return fmt.Errorf("[系统错误] 更新库存失败")
			}
		}

		// 7. 返回结果
		resultResp.InboundNo = inboundNo
		resultResp.Result = "入库成功"
		resultResp.CurrentStock = ledger.StockQty + req.Quantity
		return nil
	})

	if err != nil {
		return fmt.Errorf("%v", err)
	}

	return resp.Form(&resultResp).Build()
}

// 采购入库下拉模糊搜索 - 供应商
func purchaseInboundSupplierFuzzy(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	var suppliers []Supplier
	query := db.Model(&Supplier{}).Where("status = ?", "正常")
	if req.IsByValue() {
		query = query.Where("id = ?", req.GetValue())
	} else if req.IsByValues() {
		query = query.Where("id in ?", req.GetValues())
	} else {
		query = query.Where("supplier_name LIKE ?", "%"+req.Keyword()+"%")
	}
	query.Limit(20).Find(&suppliers)

	items := make([]*callback.SelectFuzzyItem, 0, len(suppliers))
	for _, s := range suppliers {
		items = append(items, &callback.SelectFuzzyItem{
			Value: s.ID, Label: s.SupplierName,
			DisplayInfo: map[string]interface{}{"联系人": s.Contact, "手机": s.Phone},
		})
	}
	return &callback.OnSelectFuzzyResp{Items: items}, nil
}

// 采购入库下拉模糊搜索 - 商品
func purchaseInboundProductFuzzy(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
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

var PurchaseInboundFormTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "采购入库",
		Request:  &PurchaseInboundReq{},
		Response: &PurchaseInboundResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"supplier_id": purchaseInboundSupplierFuzzy,
			"product_id":  purchaseInboundProductFuzzy,
		},
	},
}

func init() {
	packageContext.POST("purchase_inbound.form", PurchaseInboundForm, PurchaseInboundFormTemplate)
}
