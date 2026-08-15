package cashier

import (
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/pkg/gormx/query"
)

type ProductListReq struct {
	Keyword           string `json:"keyword" form:"keyword" widget:"name:关键词;type:input;placeholder:商品名称、编码"`
	Category          string `json:"category" form:"category" widget:"name:商品分类;type:select;options:全部,饮料,零食,日用品,其他;options_colors:909399,409EFF,67C23A,E6A23C,909399;render_default:全部"`
	Status            string `json:"status" form:"status" widget:"name:上架状态;type:select;options:全部,上架,下架;options_colors:909399,67C23A,909399;render_default:全部"`
	query.PageSortReq `widget:"-"`
}

func ProductList(ctx *app.Context, resp response.Response) error {
	db, err := cashierDB(ctx)
	if err != nil {
		return err
	}
	var req ProductListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&Product{})
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		queryDB = queryDB.Where("(product_name LIKE ? OR product_code LIKE ?)", like, like)
	}
	if category := strings.TrimSpace(req.Category); category != "" && category != "全部" {
		queryDB = queryDB.Where("category = ?", category)
	}
	if status := strings.TrimSpace(req.Status); status != "" && status != "全部" {
		queryDB = queryDB.Where("status = ?", status)
	}

	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("updated_at DESC, id DESC")
	}
	var rows []Product
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&rows).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{Items: rows, TotalCount: total, PageInfo: &req.PageSortReq}).Build()
}

var ProductListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "商品管理",
		Desc:         "维护收银可售商品、价格、库存和上架状态。下架或删除后的商品不会出现在收银台商品选择里。",
		Tags:         []string{"收银", "商品", "库存"},
		Request:      &ProductListReq{},
		CreateTables: []interface{}{&Product{}},
	},
	AutoCrudTable: &Product{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db, err := cashierDB(ctx)
		if err != nil {
			return nil, err
		}
		var row Product
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		normalizeProduct(&row)
		if err := validateProduct(row); err != nil {
			return nil, err
		}
		if err := db.Create(&row).Error; err != nil {
			return nil, fmt.Errorf("[系统错误]-[ProductAdd] 创建商品失败: %w", err)
		}
		return &callback.OnTableAddRowResp{Data: row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db, err := cashierDB(ctx)
		if err != nil {
			return nil, err
		}
		var existing Product
		if err := db.First(&existing, req.GetId()).Error; err != nil {
			return nil, fmt.Errorf("商品不存在")
		}
		var changed Product
		if err := req.BindChangedFields(&changed); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}

		candidate := existing
		if req.IsFieldUpdated("product_code") {
			candidate.ProductCode = changed.ProductCode
		}
		if req.IsFieldUpdated("product_name") {
			candidate.ProductName = changed.ProductName
		}
		if req.IsFieldUpdated("category") {
			candidate.Category = changed.Category
		}
		if req.IsFieldUpdated("unit") {
			candidate.Unit = changed.Unit
		}
		if req.IsFieldUpdated("sale_price") {
			candidate.SalePrice = changed.SalePrice
		}
		if req.IsFieldUpdated("discount") {
			candidate.Discount = changed.Discount
		}
		if req.IsFieldUpdated("stock_quantity") {
			candidate.StockQuantity = changed.StockQuantity
		}
		if req.IsFieldUpdated("status") {
			candidate.Status = changed.Status
		}
		normalizeProduct(&candidate)
		if err := validateProduct(candidate); err != nil {
			return nil, err
		}

		updates := req.ChangedFields()
		if req.IsFieldUpdated("product_code") {
			updates["product_code"] = candidate.ProductCode
		}
		if req.IsFieldUpdated("product_name") {
			updates["product_name"] = candidate.ProductName
		}
		if req.IsFieldUpdated("category") {
			updates["category"] = candidate.Category
		}
		if req.IsFieldUpdated("unit") {
			updates["unit"] = candidate.Unit
		}
		if req.IsFieldUpdated("sale_price") {
			updates["sale_price"] = candidate.SalePrice
		}
		if req.IsFieldUpdated("discount") {
			updates["discount"] = candidate.Discount
		}
		if req.IsFieldUpdated("stock_quantity") {
			updates["stock_quantity"] = candidate.StockQuantity
		}
		if req.IsFieldUpdated("status") {
			updates["status"] = candidate.Status
		}
		if err := db.Model(&Product{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("[系统错误]-[ProductUpdate] 更新商品失败: %w", err)
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db, err := cashierDB(ctx)
		if err != nil {
			return nil, err
		}
		if err := db.Model(&Product{}).Where("id in ?", req.GetIds()).Updates(map[string]interface{}{
			"deleted_at": time.Now(),
		}).Error; err != nil {
			return nil, fmt.Errorf("[系统错误]-[ProductDelete] 删除商品失败: %w", err)
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func init() {
	packageContext.GET("products.table", ProductList, ProductListTemplate)
}
