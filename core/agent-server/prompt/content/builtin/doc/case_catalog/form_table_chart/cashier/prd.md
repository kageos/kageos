# 案例：收银台（Table + Form + Chart）

## 一、项目概要

- **类型**：多 Table（商品、会员、支付记录）+ 收银台 Form（请求里 table 子项 + 会员 select）+ 多个统计 Form（销售趋势、分类销售、客单价等折线图）。
- **路由**：POST `cashier_desk.form`，GET 多个 list.table + 多个 statistics.chart；路由组 `/form_table_chart/cashier`。
- **适合参考**：FormTemplate 请求中 table 子组件、OnSelectFuzzy、主从表、统计/图表。

### 图形化展示

**模块与数据流（纯文本示意，任意环境可见）**

```
  [商品表] ──┐
            ├──► [收银台 Form] ──► [支付记录表] ──┬──► 销售趋势（折线图）
  [会员表] ──┘    POST cashier_desk.form          ├──► 分类销售（饼图）
                                                 └──► 平均订单金额（仪表盘）
```

**图表类型一览**

| 图表         | 类型     | 路由名 |
|--------------|----------|--------|
| 销售趋势     | line 折线图 | cashier_sales_trend_statistics |
| 分类销售     | pie 饼图   | cashier_category_sales_statistics |
| 平均订单金额 | gauge 仪表盘 | cashier_average_order_amount_statistics |

**图表效果简述**（供大模型理解每个图长什么样，写其他项目 PRD 时可照此用文本描述）

- **销售趋势（折线图）**：横轴为日期（按日聚合），纵轴两条线——销售额(元)、订单数；下方 Metadata 展示总销售额、总订单数、统计天数、平均日销售额。
- **分类销售（饼图）**：四类扇区——饮料/零食/日用品/其他，数值为各类销售额；Metadata 展示总销售额、总订单数。
- **平均订单金额（仪表盘）**：单值仪表盘，中心显示平均订单金额（元），有 min/max 刻度；Metadata 展示总订单数、总销售额、平均/最高/最低订单金额。

---

## 二、PRD 要点（表格格式）

### 1. 商品表（cashier_product_list）

**表单字段（新增/编辑）**

| 字段     | 类型     | 必填 | 默认值 | 说明 |
|----------|----------|------|--------|------|
| 商品名称 | 文本输入 | ✓   | —      | 2–50 字 |
| 商品分类 | 下拉选择 | ✓   | —      | 饮料/零食/日用品/其他 |
| 售价     | 浮点数   | ✓   | —      | 元，>0 |
| 库存     | 数字输入 | ✓   | —      | 件，≥0 |
| 折扣率   | 浮点数   | ✗   | 0.9    | 0–1 |
| 状态     | 下拉选择 | ✓   | 上架   | 上架/下架 |

**列表模式**

| ID | 创建时间 | 更新时间 | 创建人 | 商品名称 | 商品分类 | 售价 | 库存 | 折扣率 | 状态 |
|----|----------|----------|--------|----------|----------|------|------|--------|------|
| 1 | 2025-01-15 10:00 | 2025-01-15 10:00 | — | 可口可乐 | 饮料 | 3.50 | 100 | 0.9 | 上架 |
| 2 | 2025-01-15 10:05 | 2025-01-16 09:00 | — | 薯片 | 零食 | 8.00 | 50 | 0.85 | 上架 |

---

### 2. 会员表（cashier_member_list）

**表单字段（新增/编辑）**

| 字段     | 类型     | 必填 | 默认值 | 说明 |
|----------|----------|------|--------|------|
| 会员卡号 | 文本输入 | ✓   | —      | 6–20 字，唯一 |
| 客户姓名 | 文本输入 | ✓   | —      | 2–20 字 |
| 余额     | 浮点数   | ✗   | —      | 元，≥0 |
| 状态     | 下拉选择 | ✓   | 正常   | 正常/冻结 |

**列表模式**

| ID | 创建时间 | 更新时间 | 创建人 | 会员卡号 | 客户姓名 | 余额 | 状态 |
|----|----------|----------|--------|----------|----------|------|------|
| 1 | 2025-01-10 09:00 | 2025-01-10 09:00 | — | M001 | 张三 | 200.00 | 正常 |
| 2 | 2025-01-12 14:00 | 2025-01-12 14:00 | — | M002 | 李四 | 0 | 正常 |

---

### 3. 支付记录表（cashier_payment_record_list）

仅列表查询（记录由收银台 Form 产生）。

**列表模式**

| 创建时间 | 订单号 | 会员卡号 | 会员姓名 | 消费明细 | 商品总额 | 折扣金额 | 实付金额 | 状态 |
|----------|--------|----------|----------|----------|----------|----------|----------|------|
| 2025-01-20 11:30 | ORD202501200001 | M001 | 张三 | 可口可乐×2,薯片×1 | 15.00 | 1.50 | 13.50 | 支付成功 |

---

### 4. 收银台 Form（cashier_desk.form，POST）

**请求**（表单字段五列：字段 | 类型 | 必填 | 默认值 | 说明）

| 字段       | 类型     | 必填 | 默认值 | 说明 |
|------------|----------|------|--------|------|
| 商品清单   | 表格     | ✓   | —      | type:table，至少 1 行；每行：商品（OnSelectFuzzy）、数量（≥1） |
| 会员卡     | 下拉选择 | ✓   | —      | OnSelectFuzzy 从会员表选 |
| 备注       | 多行文本 | ✗   | —      | 可选 |

**响应**

| 字段       | 类型     | 说明 |
|------------|----------|------|
| 支付结果   | 多行文本 | 支付成功/失败说明 |
| 订单号     | 文本     | 订单号 |
| 商品总额   | 浮点数   | 打折前金额 |
| 折扣金额   | 浮点数   | 折扣金额 |
| 实付金额   | 浮点数   | 打折后金额 |
| 商品清单   | 表格     | 商品名称、单价、数量、小计、折扣率、折扣后金额 |
| 会员信息   | 表单     | 会员卡号、姓名、余额等 |

**说明**：请求中「商品清单」为 table 子组件，每行选商品 + 填数量；按会员折扣率计算实付金额；提交后写支付记录、扣减库存、扣减会员余额（若使用余额）。

---

### 5. 统计/图表（cashier_statistics）

多个 GET 图表接口，共用同一请求结构；支持时间范围、支付状态筛选，响应为 Chart 数据（折线图/饼图/仪表盘）。

**请求（各图表通用）**

| 字段     | 类型     | 必填 | 说明 |
|----------|----------|------|------|
| 开始时间 | 时间戳   | ✗   | 默认 30 天前 |
| 结束时间 | 时间戳   | ✗   | 默认当前时间 |
| 支付状态 | 下拉选择 | ✗   | 支付成功/已退款，不选则全部 |

**图表列表**

| 路由名 | 图表类型 | 说明 |
|--------|----------|------|
| cashier_sales_trend_statistics | line 折线图 | 按日汇总：X 轴日期，双系列「销售额(元)」「订单数」；Metadata：总销售额、总订单数、统计天数、平均日销售额 |
| cashier_category_sales_statistics | pie 饼图 | 按商品分类汇总销售额占比（饮料/零食/日用品/其他）；Metadata：总销售额、总订单数 |
| cashier_average_order_amount_statistics | gauge 仪表盘 | 平均订单金额仪表盘；Series.Config 含 min/max、detail.formatter（如 ¥{value}）；Metadata：总订单数、总销售额、平均/最高/最低订单金额 |

**实现要点**：请求体用同一结构体（如 `CashierSalesStatisticsReq`）+ widget 标签；处理函数内按时间/状态筛选支付记录与明细，聚合后组装 `types.Chart`（ChartType、Title、XAxis、Series、Metadata），`resp.Chart(chart).Build()` 返回；多个图表用 `packageContext.GET(路由名.chart, Handler, ChartTemplate)` 注册。

---

## 三、文件与路由

| 文件                         | 说明           | 注册路由                    |
|------------------------------|----------------|-----------------------------|
| cashier_desk.go              | 收银台 Form    | POST cashier_desk.form      |
| cashier_product_list.go      | 商品列表       | GET cashier_product_list.table  |
| cashier_member_list.go       | 会员列表       | GET cashier_member_list.table   |
| cashier_payment_record_list.go | 支付记录列表 | GET cashier_payment_record_list.table |
| cashier_statistics.go        | 统计/图表      | GET 多个 xxx_statistics.chart   |

---

代码随本案例一起提供；read_doc 本案例路径（如 `/builtin/doc/case_catalog/form_table_chart/cashier`）即获得 PRD 与代码，无需再调用 read_go_file。


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### cashier_desk.go

```go
package cashier

import (
	"errors"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/formatter"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/statistics"
	"gorm.io/gorm"
)

// CashierCartItem 购物车商品（响应结构）
type CashierCartItem struct {
	ProductID     int     `json:"product_id" widget:"name:商品ID;type:ID" permission:"read"`
	ProductName   string  `json:"product_name" widget:"name:商品名称;type:input" permission:"read"`
	Price         float64 `json:"price" widget:"name:单价;type:float" permission:"read"`
	Quantity      int     `json:"quantity" widget:"name:数量;type:number" permission:"read"`
	TotalPrice    float64 `json:"total_price" widget:"name:小计;type:float" permission:"read"`
	DiscountRate  float64 `json:"discount_rate" widget:"name:折扣率;type:float" permission:"read"`
	DiscountPrice float64 `json:"discount_price" widget:"name:折扣后金额;type:float" permission:"read"`
}

// CashierProductQuantity 商品数量结构体（用于购物车）
type CashierProductQuantity struct {
	ProductID int `json:"product_id" widget:"name:商品;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Quantity  int `json:"quantity" widget:"name:数量;type:number;default:1" validate:"required,min=1"`
}

// CashierDeskReq 收银台请求
type CashierDeskReq struct {
	MemberID          int                      `json:"member_id" widget:"name:会员卡;type:select" validate:"required" callback:"OnSelectFuzzy"`
	ProductQuantities []CashierProductQuantity `json:"product_quantities" widget:"name:商品清单;type:table" validate:"required,min=1"`
	Remarks           string                   `json:"remarks" widget:"name:备注;type:text_area"`
}

// CashierDeskResp 收银台响应
type CashierDeskResp struct {
	PaymentResult  string            `json:"payment_result" widget:"name:支付结果;type:text_area" permission:"read"`
	OrderNumber    string            `json:"order_number" widget:"name:订单号;type:input" permission:"read"`
	TotalAmount    float64           `json:"total_amount" widget:"name:商品总额;type:float" permission:"read"`
	DiscountAmount float64           `json:"discount_amount" widget:"name:折扣金额;type:float" permission:"read"`
	FinalAmount    float64           `json:"final_amount" widget:"name:实付金额;type:float" permission:"read"`
	ProductList    []CashierCartItem `json:"product_list" widget:"name:商品清单;type:table" permission:"read"`
	MemberInfo     *CashierMember    `json:"member_info" widget:"name:会员信息;type:form" permission:"read"`
}

// roundMoney 金额精度处理，保留2位小数
func roundMoney(amount float64) float64 {
	return float64(int64(amount*100+0.5)) / 100
}

// CashierDesk 收银台入口（SDK 注册用）：解析请求 → 调 DoCashierDesk → 写响应
func CashierDesk(ctx *app.Context, resp response.Response) error {
	var req CashierDeskReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoCashierDesk(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoCashierDesk 收银台业务逻辑：(ctx, req) → (res, err)，便于单测与复用。
// 仅需智能体介入的错误加 [系统错误] 前缀（由 SDK 区分）；此类错误打日志时须带足上下文（req/model %+v）便于排查。
func DoCashierDesk(ctx *app.Context, req *CashierDeskReq) (*CashierDeskResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoCashierDesk] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoCashierDesk]： 数据库连接失败, req: %+v", req)
	}

	member, err := validateMember(db, req.MemberID)
	if err != nil {
		return nil, fmt.Errorf("%s", err.Error())
	}

	productList, totalAmount, finalAmount, err := validateAndCalculateProducts(db, req.ProductQuantities)
	if err != nil {
		return nil, fmt.Errorf("%s", err.Error())
	}

	discountAmount := roundMoney(totalAmount - finalAmount)

	if member.Balance < finalAmount {
		return nil, fmt.Errorf("余额不足，需要支付 ¥%.2f，当前余额 ¥%.2f，缺少 ¥%.2f，请充值后重试",
			finalAmount, member.Balance, finalAmount-member.Balance)
	}

	orderNumber, err := processPaymentTransaction(db, member, *req, productList, totalAmount, discountAmount, finalAmount)
	if err != nil {
		// [系统错误] 需智能体介入；打足上下文（member、req）便于根据参数排查
		logger.Errorf(ctx, "[系统错误]-[DoCashierDesk] 支付事务失败, member: %+v, req: %+v, err: %v", member, req, err)
		return nil, fmt.Errorf("[系统错误]-[DoCashierDesk]： 支付事务失败, req: %+v, member_id: %d, final_amount: %.2f, err: %v", req, member.ID, finalAmount, err)
	}

	updatedMember, err := validateMember(db, req.MemberID)
	if err != nil {
		// [系统错误] 需智能体介入；打足上下文便于排查
		logger.Errorf(ctx, "[系统错误]-[DoCashierDesk] 重新查询会员信息失败, req: %+v, err: %v", req, err)
		updatedMember = member
	}

	return &CashierDeskResp{
		OrderNumber:    orderNumber,
		ProductList:    productList,
		TotalAmount:    totalAmount,
		DiscountAmount: discountAmount,
		FinalAmount:    finalAmount,
		MemberInfo:     &updatedMember,
		PaymentResult:  fmt.Sprintf("支付成功！订单号：%s，实付金额：¥%.2f", orderNumber, finalAmount),
	}, nil
}

// validateMember 验证会员卡
func validateMember(db *gorm.DB, memberID int) (CashierMember, error) {
	var member CashierMember

	if memberID == 0 {
		return member, fmt.Errorf("请选择会员卡")
	}

	if err := db.Where("id = ?", memberID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return member, fmt.Errorf("会员卡ID %d 不存在，请重新选择会员卡", memberID)
		}
		return member, fmt.Errorf("查询会员信息失败 - 会员ID: %d, 错误: %v", memberID, err)
	}

	if member.Status != "正常" {
		return member, fmt.Errorf("会员卡 %s (ID: %d) 状态异常：%s，无法消费。请联系客服处理",
			member.CardNumber, member.ID, member.Status)
	}

	return member, nil
}

// validateAndCalculateProducts 验证商品并计算总额（返回打折前总额和打折后总额）
func validateAndCalculateProducts(db *gorm.DB, quantities []CashierProductQuantity) ([]CashierCartItem, float64, float64, error) {
	var totalAmount float64
	var finalAmount float64
	productList := make([]CashierCartItem, 0)

	if len(quantities) == 0 {
		return productList, totalAmount, finalAmount, fmt.Errorf("请至少选择一件商品")
	}

	for _, pq := range quantities {
		var product CashierProduct
		if err := db.Where("id = ?", pq.ProductID).First(&product).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return productList, totalAmount, finalAmount, fmt.Errorf("商品ID %d 不存在，请重新选择商品", pq.ProductID)
			}
			return productList, totalAmount, finalAmount, fmt.Errorf("查询商品信息失败 - 商品ID: %d, 错误: %v", pq.ProductID, err)
		}

		if product.Status != "上架" {
			return productList, totalAmount, finalAmount, fmt.Errorf("商品 %s (ID: %d) 状态为 %s，无法购买",
				product.Name, product.ID, product.Status)
		}

		if product.Stock < pq.Quantity {
			return productList, totalAmount, finalAmount, fmt.Errorf("商品 %s 库存不足，需要 %d 件，当前库存 %d 件",
				product.Name, pq.Quantity, product.Stock)
		}

		discountRate := product.DiscountRate
		if discountRate == 0 {
			discountRate = 0.9
		}

		itemTotal := roundMoney(product.Price * float64(pq.Quantity))
		itemDiscountPrice := roundMoney(itemTotal * discountRate)

		item := CashierCartItem{
			ProductID:     pq.ProductID,
			ProductName:   product.Name,
			Price:         product.Price,
			Quantity:      pq.Quantity,
			TotalPrice:    itemTotal,
			DiscountRate:  discountRate,
			DiscountPrice: itemDiscountPrice,
		}
		productList = append(productList, item)
		totalAmount += itemTotal
		finalAmount += itemDiscountPrice
	}

	return productList, totalAmount, finalAmount, nil
}

// processPaymentTransaction 处理支付事务
func processPaymentTransaction(db *gorm.DB, member CashierMember, req CashierDeskReq, productList []CashierCartItem, totalAmount, discountAmount, finalAmount float64) (string, error) {
	var orderNumber string

	err := db.Transaction(func(tx *gorm.DB) error {
		orderNumber = fmt.Sprintf("ORD%s%d", time.Now().Format("20060102"), time.Now().UnixNano())

		if err := tx.Model(&member).Update("balance", gorm.Expr("balance - ?", finalAmount)).Error; err != nil {
			return fmt.Errorf("扣减会员余额失败 - 会员ID: %d, 卡号: %s, 消费金额: %.2f, 错误: %v",
				member.ID, member.CardNumber, finalAmount, err)
		}

		for _, item := range productList {
			if err := tx.Model(&CashierProduct{}).Where("id = ?", item.ProductID).
				Update("stock", gorm.Expr("stock - ?", item.Quantity)).Error; err != nil {
				return fmt.Errorf("扣减商品库存失败 - 商品ID: %d, 扣减数量: %d, 错误: %v",
					item.ProductID, item.Quantity, err)
			}
		}

		// 使用 formatter 将商品清单格式化为 CSV文本，前端可以直接渲染成表格展示非常方便
		var productDetails string
		if len(productList) > 0 {
			tf := formatter.NewTableFormatter().
				SetFields("ProductName", "Quantity", "Price", "TotalPrice").
				SetFieldName("ProductName", "商品名称").
				SetFieldName("Quantity", "数量").
				SetFieldName("Price", "单价").
				SetFieldName("TotalPrice", "小计")
			if csv, err := tf.ToCSV(productList); err == nil {
				productDetails = csv
			}
		}

		paymentRecord := CashierPaymentRecord{
			OrderNumber:    orderNumber,
			CardNumber:     member.CardNumber,
			MemberName:     member.CustomerName,
			ProductDetails: productDetails,
			TotalAmount:    totalAmount,
			DiscountAmount: discountAmount,
			FinalAmount:    finalAmount,
			Status:         "支付成功",
		}
		if err := tx.Create(&paymentRecord).Error; err != nil {
			return fmt.Errorf("创建支付记录失败 - 订单号: %s, 支付金额: %.2f, 错误: %v",
				orderNumber, finalAmount, err)
		}

		for _, item := range productList {
			var product CashierProduct
			if err := tx.Where("id = ?", item.ProductID).First(&product).Error; err != nil {
				return fmt.Errorf("查询商品信息失败 - 商品ID: %d, 错误: %v", item.ProductID, err)
			}

			recordItem := CashierPaymentRecordItem{
				OrderNumber:   orderNumber,
				ProductID:     item.ProductID,
				ProductName:   item.ProductName,
				Category:      product.Category,
				Price:         item.Price,
				Quantity:      item.Quantity,
				TotalPrice:    item.TotalPrice,
				DiscountRate:  item.DiscountRate,
				DiscountPrice: item.DiscountPrice,
			}
			if err := tx.Create(&recordItem).Error; err != nil {
				return fmt.Errorf("创建支付记录明细失败 - 订单号: %s, 商品ID: %d, 错误: %v",
					orderNumber, item.ProductID, err)
			}
		}

		return nil
	})

	if err != nil {
		return orderNumber, err
	}

	return orderNumber, nil
}

// CashierDeskTemplate 收银台配置
var CashierDeskTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "收银台",
		Desc:         "商品搜索选择、数量输入、会员卡支付一体化收银系统。支持商品快速搜索、会员9折优惠、余额支付、库存管理等功能。应用场景：小商店收银、便利店结算、会员消费等。",
		Tags:         []string{"收银系统", "商品管理", "会员服务"},
		Request:      &CashierDeskReq{},
		Response:     &CashierDeskResp{},
		CreateTables: []interface{}{&CashierProduct{}, &CashierMember{}, &CashierPaymentRecord{}, &CashierPaymentRecordItem{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"product_id": onSelectFuzzyProduct,
			"member_id":  onSelectFuzzyMember,
		},
	},
}

// onSelectFuzzyProduct 商品选择的模糊搜索回调
func onSelectFuzzyProduct(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}
	var currentFormData CashierDeskReq
	err := req.BindCurrentFormData(&currentFormData) //可以获取当前前端表单用户已经输入的表单数据，此时可以获取到用户已经填写的会员信息等等，假如有需要根据会员做处理的可以用这个
	if err != nil {
		return nil, err
	}

	var products []CashierProduct

	db = db.Model(&CashierProduct{}).
		Where("status = ?", "上架").
		Where("stock > ?", 0)

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		db = db.Where("name LIKE ? OR category LIKE ?", "%"+req.Keyword()+"%", "%"+req.Keyword()+"%").Limit(20)
	}

	db.Find(&products)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, p := range products {
		discountRate := p.DiscountRate
		if discountRate == 0 {
			discountRate = 0.9
		}
		discountPercent := int(discountRate * 100)

		items = append(items, &callback.SelectFuzzyItem{
			Value: p.ID,
			Label: fmt.Sprintf("%s - ¥%.2f (库存:%d, %d折)", p.Name, p.Price, p.Stock, discountPercent),
			DisplayInfo: map[string]interface{}{
				"商品名称": p.Name,
				"价格":   p.Price,
				"库存":   p.Stock,
				"分类":   p.Category,
				"折扣率":  discountRate,
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		MaxSelections: 0,
		Items:         items,
		Statistics: map[string]interface{}{
			"商品原价总额(元)":  statistics.Sum("价格 * quantity"),
			"会员折扣后价格(元)": statistics.Sum("价格 * quantity * 折扣率"),
			"优惠金额(元)":    statistics.Sum("价格 * quantity * (1 - 折扣率)"),
			"商品种类数":      statistics.Count("价格"),
			"商品总数量(件)":   statistics.Sum("quantity"),
			"折扣说明":       "每个商品可设置不同折扣率（如0.9表示9折，0.8表示8折）",
			"配送说明":       "满99元包邮，不满99元运费10元",
		},
	}, nil
}

// onSelectFuzzyMember 会员选择的模糊搜索回调
func onSelectFuzzyMember(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}

	var members []CashierMember

	db = db.Model(&CashierMember{}).
		Where("status = ?", "正常")

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		db = db.Where("card_number LIKE ? OR customer_name LIKE ?", "%"+req.Keyword()+"%", "%"+req.Keyword()+"%").Limit(20)
	}

	db.Find(&members)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, m := range members {
		items = append(items, &callback.SelectFuzzyItem{
			Value: m.ID,
			Label: fmt.Sprintf("%s - %s (余额:¥%.2f)", m.CardNumber, m.CustomerName, m.Balance),
			DisplayInfo: map[string]interface{}{
				"卡号":   m.CardNumber,
				"客户姓名": m.CustomerName,
				"余额":   m.Balance,
				"状态":   m.Status,
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		MaxSelections: 1,
		Items:         items,
		Statistics: map[string]interface{}{
			"当前余额": statistics.Value("余额"),
			"会员卡号": statistics.Value("卡号"),
			"客户姓名": statistics.Value("客户姓名"),
			"会员状态": statistics.Value("状态"),
		},
	}, nil
}

func init() {
	packageContext.POST("cashier_desk.form", CashierDesk, CashierDeskTemplate)
}
```

### cashier_member_list.go

```go
package cashier

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"gorm.io/gorm"
)

// CashierMember 会员信息表
type CashierMember struct {
	ID           int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:会员ID;type:ID" permission:"read" search:"eq"`
	CreatedAt    int64          `json:"created_at" gorm:"autoCreateTime:milli;column:created_at" widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`
	UpdatedAt    int64          `json:"updated_at" gorm:"autoUpdateTime:milli;column:updated_at" widget:"name:更新时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	CardNumber   string         `json:"card_number" gorm:"column:card_number;comment:会员卡号;uniqueIndex" widget:"name:会员卡号;type:input" search:"like" validate:"required,min=6,max=20"`
	CustomerName string         `json:"customer_name" gorm:"column:customer_name;comment:客户姓名" widget:"name:客户姓名;type:input" search:"like" validate:"required,min=2,max=20"`
	Balance      float64        `json:"balance" gorm:"column:balance;comment:余额(元)" widget:"name:余额;type:float" search:"gte,lte" validate:"gte=0"`
	Status       string         `json:"status" gorm:"column:status;comment:状态" widget:"name:状态;type:select;options:正常,冻结;options_colors:success,danger;default:正常" search:"in" validate:"required"`
}

func (CashierMember) TableName() string {
	return "cashier_member"
}

// CashierMemberListReq 会员列表请求
type CashierMemberListReq struct {
	query.SearchFilterPageReq `widget:"-"`
}

// CashierMemberList 会员管理
func CashierMemberList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req CashierMemberListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	var members []CashierMember
	return resp.Table(&members).AutoSearchFilterPaged(db, &CashierMember{}, &req.SearchFilterPageReq).Build()
}

// CashierMemberListTemplate 会员管理配置
var CashierMemberListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "会员管理",
		Desc:         "会员信息的增删改查管理，包括会员卡号、客户姓名、账户余额、状态等信息",
		Tags:         []string{"收银系统", "会员管理"},
		Request:      &CashierMemberListReq{},
		Response:     query.PaginatedTable[[]CashierMember]{},
		CreateTables: []interface{}{&CashierMember{}},
	},
	AutoCrudTable: &CashierMember{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row CashierMember
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		err := db.Create(&row).Error
		if err != nil {
			logger.Errorf(ctx, "Create cashier_member err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()

		var updateFields CashierMember
		if err := req.BindUpdates(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}

		updates := req.GetUpdates()
		err := db.Model(&CashierMember{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "Update cashier_member err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&CashierMember{}).Delete(&CashierMember{}, "id in ?", req.GetIds()).Error
		if err != nil {
			logger.Errorf(ctx, "Delete cashier_member err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func init() {
	packageContext.GET("cashier_member_list.table", CashierMemberList, CashierMemberListTemplate)
}
```

### cashier_payment_record_list.go

```go
package cashier

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"gorm.io/gorm"
)

// CashierPaymentRecord 支付记录表（订单主表）
type CashierPaymentRecord struct {
	ID             int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" permission:"read" search:"eq"`
	CreatedAt      int64          `json:"created_at" gorm:"autoCreateTime:milli;column:created_at" widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`
	UpdatedAt      int64          `json:"updated_at" gorm:"autoUpdateTime:milli;column:updated_at" widget:"name:更新时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	OrderNumber    string         `json:"order_number" gorm:"column:order_number;comment:订单号" widget:"name:订单号;type:input" search:"like"`
	CardNumber     string         `json:"card_number" gorm:"column:card_number;comment:会员卡号" widget:"name:会员卡号;type:input" search:"like"`
	MemberName     string         `json:"member_name" gorm:"column:member_name;comment:会员姓名" widget:"name:会员姓名;type:input" search:"like"`
	ProductDetails string         `json:"product_details" gorm:"column:product_details;type:text;comment:消费明细" widget:"name:消费明细;type:text;format:csv"`
	TotalAmount    float64        `json:"total_amount" gorm:"column:total_amount;comment:商品总额(打折前)" widget:"name:商品总额;type:float" search:"gte,lte"`
	DiscountAmount float64        `json:"discount_amount" gorm:"column:discount_amount;comment:折扣金额" widget:"name:折扣金额;type:float" search:"gte,lte"`
	FinalAmount    float64        `json:"final_amount" gorm:"column:final_amount;comment:实付金额(打折后)" widget:"name:实付金额;type:float" search:"gte,lte"`
	Status         string         `json:"status" gorm:"column:status;comment:状态" widget:"name:状态;type:select;options:支付成功,已退款;options_colors:success,danger" search:"in"`
}

func (CashierPaymentRecord) TableName() string {
	return "cashier_payment_record"
}

// CashierPaymentRecordItem 支付记录明细表（每个商品一条记录）
type CashierPaymentRecordItem struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" permission:"read" search:"eq"`
	CreatedAt     int64          `json:"created_at" gorm:"autoCreateTime:milli;column:created_at" widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`
	UpdatedAt     int64          `json:"updated_at" gorm:"autoUpdateTime:milli;column:updated_at" widget:"name:更新时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	OrderNumber   string         `json:"order_number" gorm:"column:order_number;comment:订单号;index" widget:"name:订单号;type:input" search:"like"`
	ProductID     int            `json:"product_id" gorm:"column:product_id;comment:商品ID;index" widget:"name:商品;type:select" search:"eq" callback:"OnSelectFuzzy"`
	ProductName   string         `json:"product_name" gorm:"column:product_name;comment:商品名称" widget:"name:商品名称;type:input" search:"like"`
	Category      string         `json:"category" gorm:"column:category;comment:商品分类" widget:"name:商品分类;type:input" search:"in"`
	Price         float64        `json:"price" gorm:"column:price;comment:单价" widget:"name:单价;type:float" search:"gte,lte"`
	Quantity      int            `json:"quantity" gorm:"column:quantity;comment:数量" widget:"name:数量;type:number" search:"gte,lte"`
	TotalPrice    float64        `json:"total_price" gorm:"column:total_price;comment:小计(打折前)" widget:"name:小计;type:float" search:"gte,lte"`
	DiscountRate  float64        `json:"discount_rate" gorm:"column:discount_rate;comment:折扣率" widget:"name:折扣率;type:float" search:"gte,lte"`
	DiscountPrice float64        `json:"discount_price" gorm:"column:discount_price;comment:折扣后金额(打折后)" widget:"name:折扣后金额;type:float" search:"gte,lte"`
}

func (CashierPaymentRecordItem) TableName() string {
	return "cashier_payment_record_item"
}

// CashierPaymentRecordListReq 支付记录列表请求
type CashierPaymentRecordListReq struct {
	query.SearchFilterPageReq `widget:"-"`
}

// CashierPaymentRecordList 支付记录列表
func CashierPaymentRecordList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req CashierPaymentRecordListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	var records []CashierPaymentRecord
	return resp.Table(&records).AutoSearchFilterPaged(db, &CashierPaymentRecord{}, &req.SearchFilterPageReq).Build()
}

// CashierPaymentRecordListTemplate 支付记录配置
var CashierPaymentRecordListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "支付记录",
		Desc:         "收银支付记录查询管理，包括订单号、会员信息、消费明细、实付金额等信息",
		Tags:         []string{"收银系统", "支付记录"},
		Request:      &CashierPaymentRecordListReq{},
		Response:     query.PaginatedTable[[]CashierPaymentRecord]{},
		CreateTables: []interface{}{&CashierPaymentRecord{}, &CashierPaymentRecordItem{}},
	},
	AutoCrudTable: &CashierPaymentRecord{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row CashierPaymentRecord
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		err := db.Create(&row).Error
		if err != nil {
			logger.Errorf(ctx, "Create cashier_payment_record err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()

		var updateFields CashierPaymentRecord
		if err := req.BindUpdates(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}

		updates := req.GetUpdates()
		err := db.Model(&CashierPaymentRecord{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "Update cashier_payment_record err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&CashierPaymentRecord{}).Delete(&CashierPaymentRecord{}, "id in ?", req.GetIds()).Error
		if err != nil {
			logger.Errorf(ctx, "Delete cashier_payment_record err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func init() {
	packageContext.GET("cashier_payment_record_list.table", CashierPaymentRecordList, CashierPaymentRecordListTemplate)
}
```

### cashier_product_list.go

```go
package cashier

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"gorm.io/gorm"
)

// CashierProduct 商品信息表
type CashierProduct struct {
	ID           int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:商品ID;type:ID" permission:"read" search:"eq"`
	CreatedAt    int64          `json:"created_at" gorm:"autoCreateTime:milli;column:created_at" widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`
	UpdatedAt    int64          `json:"updated_at" gorm:"autoUpdateTime:milli;column:updated_at" widget:"name:更新时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte" permission:"read"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	Name         string         `json:"name" gorm:"column:name;comment:商品名称" widget:"name:商品名称;type:input" search:"like" validate:"required,min=2,max=50"`
	Category     string         `json:"category" gorm:"column:category;comment:商品分类" widget:"name:商品分类;type:select;options:饮料,零食,日用品,其他;options_colors:info,primary,success,warning" search:"in" validate:"required"`
	Price        float64        `json:"price" gorm:"column:price;comment:售价(元)" widget:"name:售价;type:float" search:"gte,lte" validate:"required,gt=0"`
	Stock        int            `json:"stock" gorm:"column:stock;comment:库存(件)" widget:"name:库存;type:number" search:"gte,lte" validate:"required,gte=0"`
	DiscountRate float64        `json:"discount_rate" gorm:"column:discount_rate;comment:折扣率;default:0.9" widget:"name:折扣率;type:float;default:0.9" search:"gte,lte" validate:"gte=0,lte=1"`
	Status       string         `json:"status" gorm:"column:status;comment:状态" widget:"name:状态;type:select;options:上架,下架;options_colors:success,danger;default:上架" search:"in" validate:"required"`
}

func (CashierProduct) TableName() string {
	return "cashier_product"
}

// CashierProductListReq 商品列表请求
type CashierProductListReq struct {
	query.SearchFilterPageReq `widget:"-"`
}

// CashierProductList 商品管理
func CashierProductList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req CashierProductListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	var products []CashierProduct
	return resp.Table(&products).AutoSearchFilterPaged(db, &CashierProduct{}, &req.SearchFilterPageReq).Build()
}

// CashierProductListTemplate 商品管理配置
var CashierProductListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "商品管理",
		Desc:         "商品信息的增删改查管理，包括商品名称、分类、价格、库存、状态等信息",
		Tags:         []string{"收银系统", "商品管理"},
		Request:      &CashierProductListReq{},
		Response:     query.PaginatedTable[[]CashierProduct]{},
		CreateTables: []interface{}{&CashierProduct{}},
	},
	AutoCrudTable: &CashierProduct{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row CashierProduct
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		err := db.Create(&row).Error
		if err != nil {
			logger.Errorf(ctx, "Create cashier_product err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()

		var updateFields CashierProduct
		if err := req.BindUpdates(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}

		updates := req.GetUpdates()
		err := db.Model(&CashierProduct{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "Update cashier_product err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&CashierProduct{}).Delete(&CashierProduct{}, "id in ?", req.GetIds()).Error
		if err != nil {
			logger.Errorf(ctx, "Delete cashier_product err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func init() {
	packageContext.GET("cashier_product_list.table", CashierProductList, CashierProductListTemplate)
}
```

### cashier_statistics.go

```go
package cashier

import (
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
	"gorm.io/gorm"
)

// CashierSalesStatisticsReq 销售统计请求参数
type CashierSalesStatisticsReq struct {
	StartTime int64  `json:"start_time" form:"start_time" widget:"name:开始时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   int64  `json:"end_time" form:"end_time" widget:"name:结束时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss"`
	Status    string `json:"status" form:"status" widget:"name:支付状态;type:select;options:支付成功,已退款;options_colors:success,danger"`
}

// CashierGetDateFormatSQL 根据数据库类型返回对应的日期格式化 SQL 表达式
func CashierGetDateFormatSQL(db *gorm.DB) (dateFormatExpr, groupByExpr string) {
	dbType := db.Dialector.Name()
	switch dbType {
	case "mysql":
		dateFormatExpr = "DATE_FORMAT(FROM_UNIXTIME(created_at/1000), '%Y-%m-%d')"
	case "sqlite":
		dateFormatExpr = "strftime('%Y-%m-%d', created_at/1000, 'unixepoch')"
	default:
		dateFormatExpr = "strftime('%Y-%m-%d', created_at/1000, 'unixepoch')"
	}
	return dateFormatExpr, dateFormatExpr
}

func cashierMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// CashierSalesTrendStatistics 销售额趋势统计（折线图）
func CashierSalesTrendStatistics(ctx *app.Context, resp response.Response) error {
	var req CashierSalesStatisticsReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "CashierSalesTrendStatistics ShouldBind err: %v", err)
		return err
	}

	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	if req.StartTime == 0 {
		req.StartTime = time.Now().AddDate(0, 0, -30).UnixMilli()
	}
	if req.EndTime == 0 {
		req.EndTime = time.Now().UnixMilli()
	}

	baseQuery := db.Model(&CashierPaymentRecord{}).
		Where("created_at >= ?", req.StartTime).
		Where("created_at <= ?", req.EndTime)

	if req.Status != "" {
		baseQuery = baseQuery.Where("status = ?", req.Status)
	}

	var totalAmount float64
	var totalCount int64
	var stats struct {
		TotalAmount float64 `gorm:"column:total_amount"`
		TotalCount  int64   `gorm:"column:total_count"`
	}
	baseQuery.Select("COALESCE(SUM(final_amount), 0) as total_amount, COUNT(*) as total_count").
		Scan(&stats)
	totalAmount = stats.TotalAmount
	totalCount = stats.TotalCount

	var trendStats []struct {
		Date   string  `gorm:"column:date"`
		Amount float64 `gorm:"column:amount"`
		Count  int64   `gorm:"column:count"`
	}

	trendQuery := db.Model(&CashierPaymentRecord{}).
		Where("created_at >= ?", req.StartTime).
		Where("created_at <= ?", req.EndTime)

	if req.Status != "" {
		trendQuery = trendQuery.Where("status = ?", req.Status)
	}

	dateFormatExpr, groupByExpr := CashierGetDateFormatSQL(db)

	err := trendQuery.
		Select(fmt.Sprintf("%s as date, COALESCE(SUM(final_amount), 0) as amount, COUNT(*) as count", dateFormatExpr)).
		Group(groupByExpr).
		Order("date ASC").
		Scan(&trendStats).Error
	if err != nil {
		logger.Errorf(ctx, "CashierSalesTrendStatistics Scan err: %v", err)
		sql := fmt.Sprintf("SELECT %s as date, COALESCE(SUM(final_amount), 0) as amount, COUNT(*) as count FROM cashier_payment_record WHERE created_at >= ? AND created_at <= ?", dateFormatExpr)
		args := []interface{}{req.StartTime, req.EndTime}
		if req.Status != "" {
			sql += " AND status = ?"
			args = append(args, req.Status)
		}
		sql += fmt.Sprintf(" GROUP BY %s ORDER BY date ASC", groupByExpr)

		err2 := db.Raw(sql, args...).Scan(&trendStats).Error
		if err2 != nil {
			logger.Errorf(ctx, "CashierSalesTrendStatistics Raw SQL err: %v", err2)
			trendStats = []struct {
				Date   string  `gorm:"column:date"`
				Amount float64 `gorm:"column:amount"`
				Count  int64   `gorm:"column:count"`
			}{}
		}
	}

	dateLabels := make([]string, 0, len(trendStats))
	salesData := make([]interface{}, 0, len(trendStats))
	orderData := make([]interface{}, 0, len(trendStats))
	for _, stat := range trendStats {
		dateLabels = append(dateLabels, stat.Date)
		salesData = append(salesData, roundMoney(stat.Amount))
		orderData = append(orderData, stat.Count)
	}

	chart := &types.LineChart{
		Title: "销售额趋势统计",
		XAxis: dateLabels,
		// Series：数据系列，每项为一条折线，Name 为图例名，Data 与 XAxis 一一对应
		Series: []types.ChartSeries{
			{Name: "销售额(元)", Data: salesData},
			{Name: "订单数", Data: orderData},
		},
		Metadata: map[string]interface{}{
			"总销售额":   roundMoney(totalAmount),
			"总订单数":   totalCount,
			"统计天数":   len(dateLabels),
			"平均日销售额": roundMoney(totalAmount / float64(cashierMax(len(dateLabels), 1))),
			"数据更新时间": time.Now().Format("2006-01-02 15:04:05"),
		},
	}
	return resp.Chart(chart).Build()
}

// CashierSalesBarStatistics 每日销售额柱状图统计
func CashierSalesBarStatistics(ctx *app.Context, resp response.Response) error {
	var req CashierSalesStatisticsReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "CashierSalesBarStatistics ShouldBind err: %v", err)
		return err
	}

	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	if req.StartTime == 0 {
		req.StartTime = time.Now().AddDate(0, 0, -30).UnixMilli()
	}
	if req.EndTime == 0 {
		req.EndTime = time.Now().UnixMilli()
	}

	var trendStats []struct {
		Date   string  `gorm:"column:date"`
		Amount float64 `gorm:"column:amount"`
		Count  int64   `gorm:"column:count"`
	}

	trendQuery := db.Model(&CashierPaymentRecord{}).
		Where("created_at >= ?", req.StartTime).
		Where("created_at <= ?", req.EndTime)

	if req.Status != "" {
		trendQuery = trendQuery.Where("status = ?", req.Status)
	}

	dateFormatExpr, groupByExpr := CashierGetDateFormatSQL(db)
	err := trendQuery.
		Select(fmt.Sprintf("%s as date, COALESCE(SUM(final_amount), 0) as amount, COUNT(*) as count", dateFormatExpr)).
		Group(groupByExpr).
		Order("date ASC").
		Scan(&trendStats).Error
	if err != nil {
		logger.Errorf(ctx, "CashierSalesBarStatistics Scan err: %v", err)
		trendStats = nil
	}

	dateLabels := make([]string, 0, len(trendStats))
	salesData := make([]interface{}, 0, len(trendStats))
	orderData := make([]interface{}, 0, len(trendStats))
	var totalAmount float64
	var totalCount int64
	for _, stat := range trendStats {
		dateLabels = append(dateLabels, stat.Date)
		salesData = append(salesData, roundMoney(stat.Amount))
		orderData = append(orderData, stat.Count)
		totalAmount += stat.Amount
		totalCount += stat.Count
	}

	chart := &types.BarChart{
		Title: "每日销售额柱状图",
		XAxis: dateLabels,
		// Series：数据系列，每项为一组柱子，Name 为图例名，Data 与 XAxis 一一对应
		Series: []types.ChartSeries{
			{Name: "销售额(元)", Data: salesData},
			{Name: "订单数", Data: orderData},
		},
		Metadata: map[string]interface{}{
			"总销售额":   roundMoney(totalAmount),
			"总订单数":   totalCount,
			"统计天数":   len(dateLabels),
			"平均日销售额": roundMoney(totalAmount / float64(cashierMax(len(dateLabels), 1))),
			"数据更新时间": time.Now().Format("2006-01-02 15:04:05"),
		},
	}
	return resp.Chart(chart).Build()
}

// CashierCategorySalesStatistics 商品分类销售额统计（饼图）
func CashierCategorySalesStatistics(ctx *app.Context, resp response.Response) error {
	var req CashierSalesStatisticsReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "CashierCategorySalesStatistics ShouldBind err: %v", err)
		return err
	}

	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	if req.StartTime == 0 {
		req.StartTime = time.Now().AddDate(0, 0, -30).UnixMilli()
	}
	if req.EndTime == 0 {
		req.EndTime = time.Now().UnixMilli()
	}

	queryDB := db.Model(&CashierPaymentRecordItem{}).
		Joins("INNER JOIN cashier_payment_record ON cashier_payment_record_item.order_number = cashier_payment_record.order_number").
		Where("cashier_payment_record.created_at >= ?", req.StartTime).
		Where("cashier_payment_record.created_at <= ?", req.EndTime).
		Where("cashier_payment_record.status = ?", "支付成功")

	if req.Status != "" {
		queryDB = queryDB.Where("cashier_payment_record.status = ?", req.Status)
	}

	var categoryStats []struct {
		Category string  `gorm:"column:category"`
		Amount   float64 `gorm:"column:amount"`
	}

	err := queryDB.
		Select("cashier_payment_record_item.category, COALESCE(SUM(cashier_payment_record_item.discount_price), 0) as amount").
		Group("cashier_payment_record_item.category").
		Scan(&categoryStats).Error

	if err != nil {
		logger.Errorf(ctx, "CashierCategorySalesStatistics 统计分类销售额失败: %v", err)
		categoryStats = []struct {
			Category string  `gorm:"column:category"`
			Amount   float64 `gorm:"column:amount"`
		}{}
	}

	var totalStats struct {
		TotalAmount float64 `gorm:"column:total_amount"`
		TotalCount  int64   `gorm:"column:total_count"`
	}

	baseQuery := db.Model(&CashierPaymentRecord{}).
		Where("created_at >= ?", req.StartTime).
		Where("created_at <= ?", req.EndTime).
		Where("status = ?", "支付成功")

	if req.Status != "" {
		baseQuery = baseQuery.Where("status = ?", req.Status)
	}

	baseQuery.
		Select("COALESCE(SUM(final_amount), 0) as total_amount, COUNT(*) as total_count").
		Scan(&totalStats)

	categorySales := make(map[string]float64)
	for _, stat := range categoryStats {
		categorySales[stat.Category] = stat.Amount
	}

	pieData := make([]interface{}, 0)
	categoryLabels := []string{"饮料", "零食", "日用品", "其他"}
	for _, category := range categoryLabels {
		amount := categorySales[category]
		if amount > 0 {
			pieData = append(pieData, map[string]interface{}{
				"name":  category,
				"value": roundMoney(amount),
			})
		}
	}

	if len(pieData) == 0 {
		pieData = append(pieData, map[string]interface{}{
			"name":  "暂无数据",
			"value": 0,
		})
	}

	chart := &types.PieChart{
		Title: "商品分类销售额统计",
		// Series：饼图一般一条系列，Data 为 []{name, value} 表示各扇区
		Series: []types.ChartSeries{
			{Name: "销售额", Data: pieData},
		},
		Metadata: map[string]interface{}{
			"总销售额":   roundMoney(totalStats.TotalAmount),
			"总订单数":   totalStats.TotalCount,
			"数据更新时间": time.Now().Format("2006-01-02 15:04:05"),
		},
	}
	return resp.Chart(chart).Build()
}

// CashierAverageOrderAmountStatistics 平均订单金额统计（仪表盘）
func CashierAverageOrderAmountStatistics(ctx *app.Context, resp response.Response) error {
	var req CashierSalesStatisticsReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "CashierAverageOrderAmountStatistics ShouldBind err: %v", err)
		return err
	}

	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	if req.StartTime == 0 {
		req.StartTime = time.Now().AddDate(0, 0, -30).UnixMilli()
	}
	if req.EndTime == 0 {
		req.EndTime = time.Now().UnixMilli()
	}

	baseQuery := db.Model(&CashierPaymentRecord{}).
		Where("created_at >= ?", req.StartTime).
		Where("created_at <= ?", req.EndTime).
		Where("status = ?", "支付成功")

	if req.Status != "" {
		baseQuery = baseQuery.Where("status = ?", req.Status)
	}

	var stats struct {
		TotalCount  int64   `gorm:"column:total_count"`
		TotalAmount float64 `gorm:"column:total_amount"`
		MaxAmount   float64 `gorm:"column:max_amount"`
		MinAmount   float64 `gorm:"column:min_amount"`
	}
	baseQuery.
		Select("COUNT(*) as total_count, COALESCE(SUM(final_amount), 0) as total_amount, COALESCE(MAX(final_amount), 0) as max_amount, COALESCE(MIN(final_amount), 0) as min_amount").
		Scan(&stats)

	var avgAmount float64
	if stats.TotalCount > 0 {
		avgAmount = roundMoney(stats.TotalAmount / float64(stats.TotalCount))
	}

	maxValue := roundMoney(stats.MaxAmount * 1.5)
	if maxValue < 100 {
		maxValue = 100
	}

	chart := &types.GaugeChart{
		Title: "平均订单金额统计",
		// Series：仪表盘一般一条系列，Data 为单值，Config 可配 min/max/detail 等
		Series: []types.ChartSeries{
			{
				Name:   "平均订单金额",
				Data:   []interface{}{avgAmount},
				Config: map[string]interface{}{
					"min": 0,
					"max": maxValue,
					"detail": map[string]interface{}{
						"formatter":  "¥{value}",
						"fontSize":   20,
						"color":      "#1f2937",
						"fontWeight": "bold",
					},
					"axisLabel": map[string]interface{}{
						"formatter": "¥{value}",
					},
				},
			},
		},
		Metadata: map[string]interface{}{
			"总订单数":   stats.TotalCount,
			"总销售额":   roundMoney(stats.TotalAmount),
			"平均订单金额": fmt.Sprintf("¥%.2f", avgAmount),
			"最高订单金额": fmt.Sprintf("¥%.2f", roundMoney(stats.MaxAmount)),
			"最低订单金额": fmt.Sprintf("¥%.2f", roundMoney(stats.MinAmount)),
			"数据更新时间": time.Now().Format("2006-01-02 15:04:05"),
		},
	}
	return resp.Chart(chart).Build()
}

// CashierSalesTrendStatisticsTemplate 销售额趋势统计图表模板
var CashierSalesTrendStatisticsTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "销售额趋势统计",
		Tags:     []string{"BI", "销售分析"},
		Desc:     "展示销售额和订单数的时间趋势（折线图）",
		Request:  &CashierSalesStatisticsReq{},
		Response: &types.LineChart{},
	},
	ChartType: app.ChartTypeLine,
}

// CashierSalesBarStatisticsTemplate 每日销售额柱状图统计图表模板
var CashierSalesBarStatisticsTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "每日销售额柱状图",
		Tags:     []string{"BI", "销售分析"},
		Desc:     "按日期展示每日销售额和订单数（柱状图）",
		Request:  &CashierSalesStatisticsReq{},
		Response: &types.BarChart{},
	},
	ChartType: app.ChartTypeBar,
}

// CashierCategorySalesStatisticsTemplate 商品分类销售额统计图表模板
var CashierCategorySalesStatisticsTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "商品分类销售额统计",
		Tags:     []string{"BI", "销售分析"},
		Desc:     "展示各商品分类的销售额占比（饼图）",
		Request:  &CashierSalesStatisticsReq{},
		Response: &types.PieChart{},
	},
	ChartType: app.ChartTypePie,
}

// CashierAverageOrderAmountStatisticsTemplate 平均订单金额统计图表模板
var CashierAverageOrderAmountStatisticsTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "平均订单金额统计",
		Tags:     []string{"BI", "经营分析"},
		Desc:     "展示平均订单金额、总销售额、最高/最低订单金额等关键指标（仪表盘）",
		Request:  &CashierSalesStatisticsReq{},
		Response: &types.GaugeChart{},
	},
	ChartType: app.ChartTypeGauge,
}

func init() {
	// 销售额趋势统计：按日期展示销售额、订单数折线图，支持时间范围、支付状态筛选
	packageContext.GET("cashier_sales_trend_statistics.chart", CashierSalesTrendStatistics, CashierSalesTrendStatisticsTemplate)
	// 每日销售额柱状图：按日期展示销售额、订单数柱状图，支持时间范围、支付状态筛选
	packageContext.GET("cashier_sales_bar_statistics.chart", CashierSalesBarStatistics, CashierSalesBarStatisticsTemplate)
	// 商品分类销售额统计：各分类销售额占比饼图，支持时间范围、支付状态筛选
	packageContext.GET("cashier_category_sales_statistics.chart", CashierCategorySalesStatistics, CashierCategorySalesStatisticsTemplate)
	// 平均订单金额统计：平均订单金额、总销售额等指标仪表盘，支持时间范围、支付状态筛选
	packageContext.GET("cashier_average_order_amount_statistics.chart", CashierAverageOrderAmountStatistics, CashierAverageOrderAmountStatisticsTemplate)
}
```
