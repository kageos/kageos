package cashier

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

const (
	productStatusListed   = "上架"
	productStatusUnlisted = "下架"

	paymentStatusSuccess = "支付成功"

	paymentMethodCash   = "现金"
	paymentMethodWechat = "微信"
	paymentMethodAlipay = "支付宝"
	paymentMethodCard   = "银行卡"
	paymentMethodOther  = "其他"
)

type Product struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:商品ID;type:ID" hide:"create,update"`
	ProductCode   string         `json:"product_code" gorm:"column:product_code;type:varchar(80);index" widget:"name:商品编码;type:input;placeholder:条码或内部编码"`
	ProductName   string         `json:"product_name" gorm:"column:product_name;type:varchar(160);index" widget:"name:商品名称;type:input" validate:"required"`
	Category      string         `json:"category" gorm:"column:category;type:varchar(40);index" widget:"name:商品分类;type:select;options:饮料,零食,日用品,其他;options_colors:409EFF,67C23A,E6A23C,909399;render_default:其他" validate:"required,oneof=饮料 零食 日用品 其他"`
	Unit          string         `json:"unit" gorm:"column:unit;type:varchar(20)" widget:"name:单位;type:select;options:件,瓶,包,盒,斤,其他;options_colors:909399,409EFF,67C23A,E6A23C,F56C6C,9C27B0;render_default:件" validate:"required,oneof=件 瓶 包 盒 斤 其他"`
	SalePrice     float64        `json:"sale_price" gorm:"column:sale_price" widget:"name:销售单价;type:float;precision:2;unit:元" validate:"required,min=0"`
	Discount      float64        `json:"discount" gorm:"column:discount" widget:"name:折扣;type:float;precision:1;unit:折;min:0;max:10;render_default:10;placeholder:10表示不打折，8.5表示八五折" validate:"min=0,max=10"`
	StockQuantity int            `json:"stock_quantity" gorm:"column:stock_quantity" widget:"name:库存数量;type:integer;min:0" validate:"required,min=0"`
	Status        string         `json:"status" gorm:"column:status;type:varchar(20);index" widget:"name:上架状态;type:select;options:上架,下架;options_colors:67C23A,909399;render_default:上架" validate:"required,oneof=上架 下架"`
	CreatedAt     types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt     types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index;column:deleted_at" widget:"-"`
}

func (Product) TableName() string {
	return "cashier_product"
}

type Payment struct {
	ID             int        `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:支付ID;type:ID" hide:"create,update"`
	OrderNo        string     `json:"order_no" gorm:"column:order_no;type:varchar(64);uniqueIndex" widget:"name:订单号;type:input" hide:"create,update"`
	ItemsDesc      string     `json:"items_desc" gorm:"column:items_desc;type:text" widget:"name:消费明细;type:text_area" hide:"create,update"`
	TotalAmount    float64    `json:"total_amount" gorm:"column:total_amount" widget:"name:应收金额;type:float;precision:2;unit:元" hide:"create,update"`
	DiscountAmount float64    `json:"discount_amount" gorm:"column:discount_amount" widget:"name:优惠金额;type:float;precision:2;unit:元" hide:"create,update"`
	PaidAmount     float64    `json:"paid_amount" gorm:"column:paid_amount" widget:"name:实收金额;type:float;precision:2;unit:元" hide:"create,update"`
	ChangeAmount   float64    `json:"change_amount" gorm:"column:change_amount" widget:"name:找零金额;type:float;precision:2;unit:元" hide:"create,update"`
	PaymentMethod  string     `json:"payment_method" gorm:"column:payment_method;type:varchar(30);index" widget:"name:支付方式;type:select;options:现金,微信,支付宝,银行卡,其他;options_colors:67C23A,409EFF,00B8A9,E6A23C,909399" hide:"create,update"`
	PaymentStatus  string     `json:"payment_status" gorm:"column:payment_status;type:varchar(30);index" widget:"name:支付状态;type:select;options:支付成功;options_colors:67C23A" hide:"create,update"`
	PaymentTime    types.Time `json:"payment_time" gorm:"column:payment_time;type:datetime;index" widget:"name:支付时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	Cashier        string     `json:"cashier" gorm:"column:cashier;type:varchar(120);index" widget:"name:收银员;type:user" hide:"create,update"`
	Remark         string     `json:"remark" gorm:"column:remark;type:text" widget:"name:备注;type:text_area" hide:"create,update"`
	CreatedAt      types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
}

func (Payment) TableName() string {
	return "cashier_payment"
}

type PaymentItem struct {
	ID              int        `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"-"`
	PaymentID       int        `json:"payment_id" gorm:"column:payment_id;index" widget:"-"`
	OrderNo         string     `json:"order_no" gorm:"column:order_no;type:varchar(64);index" widget:"-"`
	ProductID       int        `json:"product_id" gorm:"column:product_id;index" widget:"-"`
	ProductName     string     `json:"product_name" gorm:"column:product_name;type:varchar(160);index" widget:"-"`
	ProductCategory string     `json:"product_category" gorm:"column:product_category;type:varchar(40);index" widget:"-"`
	Quantity        int        `json:"quantity" gorm:"column:quantity" widget:"-"`
	UnitPrice       float64    `json:"unit_price" gorm:"column:unit_price" widget:"-"`
	Discount        float64    `json:"discount" gorm:"column:discount" widget:"-"`
	DiscountAmount  float64    `json:"discount_amount" gorm:"column:discount_amount" widget:"-"`
	LineAmount      float64    `json:"line_amount" gorm:"column:line_amount" widget:"-"`
	PaymentTime     types.Time `json:"payment_time" gorm:"column:payment_time;type:datetime;index" widget:"-"`
	CreatedAt       types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"-"`
}

func (PaymentItem) TableName() string {
	return "cashier_payment_item"
}

func cashierDB(ctx *app.Context) (*gorm.DB, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}
	return db, nil
}

func normalizeProduct(row *Product) {
	row.ProductCode = strings.TrimSpace(row.ProductCode)
	row.ProductName = strings.TrimSpace(row.ProductName)
	row.Category = strings.TrimSpace(row.Category)
	row.Unit = strings.TrimSpace(row.Unit)
	row.Status = strings.TrimSpace(row.Status)
	if row.Category == "" {
		row.Category = "其他"
	}
	if row.Unit == "" {
		row.Unit = "件"
	}
	if row.Status == "" {
		row.Status = productStatusListed
	}
	row.SalePrice = roundMoney(row.SalePrice)
	row.Discount = normalizeDiscount(row.Discount)
}

func validateProduct(row Product) error {
	if row.ProductName == "" {
		return fmt.Errorf("商品名称不能为空")
	}
	if !isValidCategory(row.Category) {
		return fmt.Errorf("商品分类只能是：饮料、零食、日用品、其他")
	}
	if !isValidUnit(row.Unit) {
		return fmt.Errorf("单位只能是：件、瓶、包、盒、斤、其他")
	}
	if row.SalePrice < 0 {
		return fmt.Errorf("销售单价不能小于 0")
	}
	if row.Discount < 0 || row.Discount > 10 {
		return fmt.Errorf("折扣必须在 0 到 10 之间，10 表示不打折，8.5 表示八五折")
	}
	if row.StockQuantity < 0 {
		return fmt.Errorf("库存数量不能小于 0")
	}
	if !isValidProductStatus(row.Status) {
		return fmt.Errorf("上架状态只能是：上架、下架")
	}
	return nil
}

func isValidCategory(category string) bool {
	switch category {
	case "饮料", "零食", "日用品", "其他":
		return true
	default:
		return false
	}
}

func isValidUnit(unit string) bool {
	switch unit {
	case "件", "瓶", "包", "盒", "斤", "其他":
		return true
	default:
		return false
	}
}

func isValidProductStatus(status string) bool {
	return status == productStatusListed || status == productStatusUnlisted
}

func normalizePaymentMethod(method string) string {
	method = strings.TrimSpace(method)
	if method == "" {
		return paymentMethodCash
	}
	return method
}

func isValidPaymentMethod(method string) bool {
	switch method {
	case paymentMethodCash, paymentMethodWechat, paymentMethodAlipay, paymentMethodCard, paymentMethodOther:
		return true
	default:
		return false
	}
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func normalizeDiscount(value float64) float64 {
	if value <= 0 {
		return 10
	}
	if value > 10 {
		return 10
	}
	return roundMoney(value)
}

func discountedUnitPrice(product Product) float64 {
	return roundMoney(product.SalePrice * normalizeDiscount(product.Discount) / 10)
}

func generateOrderNo(now time.Time) string {
	return fmt.Sprintf("ORD%s%06d", now.Format("20060102150405"), now.Nanosecond()/1000%1000000)
}
