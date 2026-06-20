package inventory

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// PurchaseInStockReq 采购入库请求
type PurchaseInStockReq struct {
	OrderNo string `json:"order_no" widget:"name:采购单号;type:select;placeholder:请选择待入库的采购单" validate:"required" callback:"OnSelectFuzzy"`
}

// PurchaseInStockResp 采购入库响应
type PurchaseInStockResp struct {
	Result    string `json:"result" widget:"name:入库结果;type:input"`
	ItemCount int    `json:"item_count" widget:"name:入库商品数;type:integer"`
}

var PurchaseInStockTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "采购入库",
		Request:  &PurchaseInStockReq{},
		Response: &PurchaseInStockResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"order_no": onSelectFuzzyPendingPurchaseOrder,
		},
	},
}

func onSelectFuzzyPendingPurchaseOrder(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	var orders []PurchaseOrder
	queryDB := db.Model(&PurchaseOrder{}).Where("status = ?", "待入库")
	if req.IsByValue() {
		queryDB = queryDB.Where("order_no = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		queryDB = queryDB.Where("order_no in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		if keyword != "" {
			queryDB = queryDB.Where("order_no LIKE ? OR supplier LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
		}
		queryDB = queryDB.Limit(20)
	}
	queryDB.Find(&orders)

	items := make([]*callback.SelectFuzzyItem, 0, len(orders))
	for _, o := range orders {
		items = append(items, &callback.SelectFuzzyItem{
			Value: o.OrderNo,
			Label: o.OrderNo,
			DisplayInfo: map[string]interface{}{
				"供应商":  o.Supplier,
				"采购金额": o.TotalAmount,
				"采购日期": o.OrderDate.Time().Format("2006-01-02 15:04"),
			},
		})
	}
	return &callback.OnSelectFuzzyResp{
		Items:         items,
		MaxSelections: 1,
	}, nil
}

// PurchaseInStock 采购入库处理
func PurchaseInStock(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	var req PurchaseInStockReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	var result string
	var itemCount int

	err := db.Transaction(func(tx *gorm.DB) error {
		var po PurchaseOrder
		if err := tx.Where("order_no = ?", req.OrderNo).First(&po).Error; err != nil {
			result = fmt.Sprintf("采购单 %s 不存在", req.OrderNo)
			return fmt.Errorf("采购单 %s 不存在", req.OrderNo)
		}
		if po.Status != "待入库" {
			result = fmt.Sprintf("采购单 %s 当前状态为%s，非待入库状态", req.OrderNo, po.Status)
			return fmt.Errorf("采购单状态异常: %s", po.Status)
		}

		// 解析商品明细: "商品名称×数量@单价"
		lines := strings.Split(strings.TrimSpace(po.ItemsDetail), "\n")
		type itemInfo struct {
			name     string
			quantity int
			price    float64
		}
		var items []itemInfo
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// 格式: 商品名称×数量@单价
			atIdx := strings.LastIndex(line, "@")
			if atIdx < 0 {
				result = fmt.Sprintf("商品明细格式错误：%s", line)
				return fmt.Errorf("格式错误: %s", line)
			}
			nameAndQty := strings.TrimSpace(line[:atIdx])
			priceStr := strings.TrimSpace(line[atIdx+1:])

			xIdx := strings.LastIndex(nameAndQty, "×")
			if xIdx < 0 {
				result = fmt.Sprintf("商品明细格式错误（缺少×分隔）：%s", line)
				return fmt.Errorf("格式错误: %s", line)
			}
			name := strings.TrimSpace(nameAndQty[:xIdx])
			qtyStr := strings.TrimSpace(nameAndQty[xIdx+len("×"):])

			qty, err := strconv.Atoi(qtyStr)
			if err != nil || qty <= 0 {
				result = fmt.Sprintf("商品数量格式错误：%s", line)
				return fmt.Errorf("数量错误: %s, err: %w", line, err)
			}
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil || price < 0 {
				result = fmt.Sprintf("商品单价格式错误：%s", line)
				return fmt.Errorf("单价错误: %s, err: %w", line, err)
			}
			items = append(items, itemInfo{name: name, quantity: qty, price: price})
		}

		if len(items) == 0 {
			result = "商品明细为空"
			return fmt.Errorf("商品明细为空")
		}

		now := types.Time(time.Now())
		username := ctx.GetRequestUser()
		for _, item := range items {
			var product Product
			if err := tx.Where("name = ?", item.name).First(&product).Error; err != nil {
				result = fmt.Sprintf("商品 %s 不存在", item.name)
				return fmt.Errorf("商品 %s 不存在: %w", item.name, err)
			}
			// 增加库存
			product.Stock += item.quantity
			if err := tx.Model(&product).Where("id = ?", product.ID).Update("stock", product.Stock).Error; err != nil {
				result = fmt.Sprintf("更新商品 %s 库存失败", item.name)
				return fmt.Errorf("更新库存失败: %w", err)
			}

			// 生成库存流水
			flow := InventoryFlow{
				Product:     product.Name,
				ChangeType:  "采购入库",
				ChangeQty:   item.quantity,
				AfterStock:  product.Stock,
				RefOrderNo:  po.OrderNo,
				OperateTime: now,
				CreatedBy:   username,
				UpdatedBy:   username,
			}
			if err := tx.Create(&flow).Error; err != nil {
				result = fmt.Sprintf("生成商品 %s 库存流水失败", item.name)
				return fmt.Errorf("生成流水失败: %w", err)
			}
		}

		// 更新采购单状态
		if err := tx.Model(&po).Updates(map[string]interface{}{
			"status":     "已入库",
			"updated_by": username,
		}).Error; err != nil {
			result = "更新采购单状态失败"
			return fmt.Errorf("更新采购单状态失败: %w", err)
		}

		itemCount = len(items)
		result = "入库成功"
		return nil
	})

	if err != nil {
		return resp.Form(&PurchaseInStockResp{
			Result:    result,
			ItemCount: itemCount,
		}).Build()
	}

	return resp.Form(&PurchaseInStockResp{
		Result:    result,
		ItemCount: itemCount,
	}).Build()
}

func init() {
	packageContext.POST("purchase_instock.form", PurchaseInStock, PurchaseInStockTemplate)
}
