package cashier

import (
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CheckoutItemLine struct {
	ProductID int `json:"product_id" widget:"name:商品;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Quantity  int `json:"quantity" widget:"name:数量;type:integer;min:1;render_default:1" validate:"required,min=1"`
}

type CheckoutReq struct {
	Items          []CheckoutItemLine `json:"items" widget:"name:商品清单;type:table" validate:"required,min=1"`
	PaymentMethod  string             `json:"payment_method" widget:"name:支付方式;type:select;options:现金,微信,支付宝,银行卡,其他;options_colors:67C23A,409EFF,00B8A9,E6A23C,909399;render_default:现金" validate:"required"`
	ReceivedAmount float64            `json:"received_amount" widget:"name:实收金额;type:float;precision:2;unit:元;placeholder:留空按应收金额结算" validate:"min=0"`
	Remark         string             `json:"remark" widget:"name:备注;type:text_area"`
}

type CheckoutResp struct {
	OrderNo        string  `json:"order_no" widget:"name:订单号;type:input"`
	Result         string  `json:"result" widget:"name:支付结果;type:input"`
	TotalAmount    float64 `json:"total_amount" widget:"name:应收金额;type:float;precision:2;unit:元"`
	DiscountAmount float64 `json:"discount_amount" widget:"name:优惠金额;type:float;precision:2;unit:元"`
	PaidAmount     float64 `json:"paid_amount" widget:"name:实收金额;type:float;precision:2;unit:元"`
	ChangeAmount   float64 `json:"change_amount" widget:"name:找零金额;type:float;precision:2;unit:元"`
	ItemsDesc      string  `json:"items_desc" widget:"name:消费明细;type:text_area"`
}

var CheckoutTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "收银结账",
		Desc:         "选择已上架商品和数量，提交后生成支付记录并扣减库存。实收金额留空时按应收金额结算。",
		Tags:         []string{"收银", "结账", "库存扣减"},
		Request:      &CheckoutReq{},
		Response:     &CheckoutResp{},
		CreateTables: []interface{}{&Product{}, &Payment{}, &PaymentItem{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"product_id": onSelectFuzzyProduct,
		},
	},
}

func Checkout(ctx *app.Context, resp response.Response) error {
	db, err := cashierDB(ctx)
	if err != nil {
		return err
	}
	var req CheckoutReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	req.PaymentMethod = normalizePaymentMethod(req.PaymentMethod)
	req.Remark = strings.TrimSpace(req.Remark)
	if !isValidPaymentMethod(req.PaymentMethod) {
		return fmt.Errorf("支付方式只能是：现金、微信、支付宝、银行卡、其他")
	}
	if len(req.Items) == 0 {
		return fmt.Errorf("请至少选择 1 个商品")
	}

	var result CheckoutResp
	if err := db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		paymentTime := types.Time(now)
		orderNo := generateOrderNo(now)

		type preparedItem struct {
			product  Product
			quantity int
			amount   float64
		}
		prepared := make([]preparedItem, 0, len(req.Items))
		var totalAmount float64
		var discountAmount float64
		var descParts []string

		for _, item := range req.Items {
			if item.ProductID <= 0 {
				return fmt.Errorf("商品不能为空")
			}
			if item.Quantity <= 0 {
				return fmt.Errorf("商品数量必须大于 0")
			}
			var product Product
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&product, item.ProductID).Error; err != nil {
				return fmt.Errorf("商品不存在或已删除")
			}
			if product.Status != productStatusListed {
				return fmt.Errorf("商品[%s]已下架，不能收银", product.ProductName)
			}
			if product.StockQuantity < item.Quantity {
				return fmt.Errorf("商品[%s]库存不足，当前库存 %d，需要 %d", product.ProductName, product.StockQuantity, item.Quantity)
			}
			discount := normalizeDiscount(product.Discount)
			unitPrice := discountedUnitPrice(product)
			originalLineAmount := roundMoney(product.SalePrice * float64(item.Quantity))
			lineAmount := roundMoney(unitPrice * float64(item.Quantity))
			lineDiscountAmount := roundMoney(originalLineAmount - lineAmount)
			totalAmount = roundMoney(totalAmount + lineAmount)
			discountAmount = roundMoney(discountAmount + lineDiscountAmount)
			prepared = append(prepared, preparedItem{product: product, quantity: item.Quantity, amount: lineAmount})
			if discount < 10 {
				descParts = append(descParts, fmt.Sprintf("%s×%d@%.2f元(%.1f折)", product.ProductName, item.Quantity, product.SalePrice, discount))
			} else {
				descParts = append(descParts, fmt.Sprintf("%s×%d@%.2f元", product.ProductName, item.Quantity, product.SalePrice))
			}
		}

		paidAmount := roundMoney(req.ReceivedAmount)
		if paidAmount <= 0 {
			paidAmount = totalAmount
		}
		if paidAmount < totalAmount {
			return fmt.Errorf("实收金额不足，应收 %.2f 元，实收 %.2f 元", totalAmount, paidAmount)
		}
		changeAmount := roundMoney(paidAmount - totalAmount)
		itemsDesc := strings.Join(descParts, "，")
		payment := Payment{
			OrderNo:        orderNo,
			ItemsDesc:      itemsDesc,
			TotalAmount:    totalAmount,
			DiscountAmount: discountAmount,
			PaidAmount:     paidAmount,
			ChangeAmount:   changeAmount,
			PaymentMethod:  req.PaymentMethod,
			PaymentStatus:  paymentStatusSuccess,
			PaymentTime:    paymentTime,
			Cashier:        ctx.GetRequestUser(),
			Remark:         req.Remark,
		}
		if err := tx.Create(&payment).Error; err != nil {
			return fmt.Errorf("[系统错误]-[Checkout] 创建支付记录失败: %w", err)
		}

		items := make([]PaymentItem, 0, len(prepared))
		for _, preparedItem := range prepared {
			update := tx.Model(&Product{}).
				Where("id = ? AND stock_quantity >= ?", preparedItem.product.ID, preparedItem.quantity).
				Updates(map[string]interface{}{"stock_quantity": gorm.Expr("stock_quantity - ?", preparedItem.quantity)})
			if update.Error != nil {
				return fmt.Errorf("[系统错误]-[Checkout] 扣减库存失败，商品 %s: %w", preparedItem.product.ProductName, update.Error)
			}
			if update.RowsAffected != 1 {
				return fmt.Errorf("商品[%s]库存不足，请刷新后重试", preparedItem.product.ProductName)
			}
			items = append(items, PaymentItem{
				PaymentID:       payment.ID,
				OrderNo:         payment.OrderNo,
				ProductID:       preparedItem.product.ID,
				ProductName:     preparedItem.product.ProductName,
				ProductCategory: preparedItem.product.Category,
				Quantity:        preparedItem.quantity,
				UnitPrice:       preparedItem.product.SalePrice,
				Discount:        normalizeDiscount(preparedItem.product.Discount),
				DiscountAmount:  roundMoney(preparedItem.product.SalePrice*float64(preparedItem.quantity) - preparedItem.amount),
				LineAmount:      preparedItem.amount,
				PaymentTime:     paymentTime,
			})
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return fmt.Errorf("[系统错误]-[Checkout] 创建支付明细失败: %w", err)
			}
		}
		result = CheckoutResp{
			OrderNo:        payment.OrderNo,
			Result:         paymentStatusSuccess,
			TotalAmount:    totalAmount,
			DiscountAmount: discountAmount,
			PaidAmount:     paidAmount,
			ChangeAmount:   changeAmount,
			ItemsDesc:      itemsDesc,
		}
		return nil
	}); err != nil {
		return err
	}

	return resp.Form(&result).Build()
}

type SalesTrendReq struct {
	StartTime types.Time `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   types.Time `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

func SalesTrendChart(ctx *app.Context, resp response.Response) error {
	db, err := cashierDB(ctx)
	if err != nil {
		return err
	}
	var req SalesTrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	startTime := req.StartTime.Time()
	endTime := req.EndTime.Time()
	if startTime.IsZero() {
		startTime = time.Now().AddDate(0, 0, -30)
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}

	type dailyStat struct {
		Date       string  `json:"date"`
		Amount     float64 `json:"amount"`
		OrderCount int     `json:"order_count"`
	}
	var stats []dailyStat
	if err := db.Model(&Payment{}).
		Select("DATE(payment_time) AS date, SUM(paid_amount) AS amount, COUNT(*) AS order_count").
		Where("payment_status = ? AND payment_time BETWEEN ? AND ?", paymentStatusSuccess, startTime, endTime).
		Group("DATE(payment_time)").
		Order("date ASC").
		Scan(&stats).Error; err != nil {
		return err
	}

	xAxis := make([]string, 0, len(stats))
	amountData := make([]interface{}, 0, len(stats))
	countData := make([]interface{}, 0, len(stats))
	var totalAmount float64
	var totalOrders int
	for _, stat := range stats {
		xAxis = append(xAxis, stat.Date)
		amountData = append(amountData, roundMoney(stat.Amount))
		countData = append(countData, stat.OrderCount)
		totalAmount = roundMoney(totalAmount + stat.Amount)
		totalOrders += stat.OrderCount
	}
	return resp.Chart(&chart.LineChart{
		Title: "销售趋势",
		XAxis: xAxis,
		Series: []chart.ChartSeries{
			{Name: "销售额", Data: amountData},
			{Name: "订单数", Data: countData},
		},
		Metadata: map[string]interface{}{
			"统计开始": startTime.Format(types.TimeLayout),
			"统计结束": endTime.Format(types.TimeLayout),
			"总销售额": totalAmount,
			"订单数":  totalOrders,
		},
	}).Build()
}

func init() {
	packageContext.POST("checkout.form", Checkout, CheckoutTemplate)
	packageContext.GET("sales_trend.chart", SalesTrendChart, &app.ChartTemplate{
		BaseConfig: app.BaseConfig{
			Name:         "销售趋势",
			Desc:         "基于支付记录按日期统计销售额和订单数。",
			Request:      &SalesTrendReq{},
			Response:     &chart.LineChart{},
			CreateTables: []interface{}{&Payment{}},
		},
		ChartType: app.ChartTypeLine,
	})
}
