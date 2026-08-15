package cashier

import (
	"fmt"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/statistics"
)

func onSelectFuzzyProduct(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db, err := cashierDB(ctx)
	if err != nil {
		return nil, err
	}
	queryDB := db.Model(&Product{}).Where("status = ?", productStatusListed)
	if req.IsByValue() {
		queryDB = queryDB.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		queryDB = queryDB.Where("id IN ?", req.GetValues())
	} else {
		keyword := strings.TrimSpace(req.Keyword())
		if keyword != "" {
			like := "%" + keyword + "%"
			queryDB = queryDB.Where("(product_name LIKE ? OR product_code LIKE ? OR category LIKE ?)", like, like, like)
		}
		queryDB = queryDB.Order("product_name ASC").Limit(20)
	}

	var products []Product
	if err := queryDB.Find(&products).Error; err != nil {
		return nil, err
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(products))
	for _, p := range products {
		label := p.ProductName
		if p.ProductCode != "" {
			label = fmt.Sprintf("%s（%s）", p.ProductName, p.ProductCode)
		}
		discount := normalizeDiscount(p.Discount)
		items = append(items, &callback.SelectFuzzyItem{
			Value: p.ID,
			Label: label,
			DisplayInfo: map[string]interface{}{
				"商品名称": p.ProductName,
				"商品分类": p.Category,
				"单价":   p.SalePrice,
				"折扣":   discount,
				"折后单价": discountedUnitPrice(p),
				"库存":   p.StockQuantity,
				"单位":   p.Unit,
			},
		})
	}
	return &callback.OnSelectFuzzyResp{
		Items:         items,
		MaxSelections: 1,
		Statistics: map[string]interface{}{
			"商品原价总额(元)": statistics.Sum("单价 * quantity"),
			"折后应收(元)":   statistics.Sum("折后单价 * quantity"),
			"优惠金额(元)":   statistics.Sum("单价 * quantity - 折后单价 * quantity"),
			"商品种类数":     statistics.Count("单价"),
			"商品总数量":     statistics.Sum("quantity"),
		},
	}, nil
}
