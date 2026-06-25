package query

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PaginatedTable 分页结果结构体
type PaginatedTable[T any] struct {
	Items       T     `json:"items" runner:"widget:table;type:array;code:items"` // 分页数据
	CurrentPage int   `json:"current_page" runner:"search_cond"`                 // 当前页码
	TotalCount  int64 `json:"total_count" runner:"search_cond"`                  // 总数据量
	TotalPages  int   `json:"total_pages" runner:"search_cond"`                  // 总页数
	PageSize    int   `json:"page_size" runner:"search_cond"`                    // 每页数量
}

// PageSortReq 只负责分页和排序，不承载搜索协议。
// 业务筛选字段应该显式写在业务 Request struct 中，并在 Handler 里手写 Where。
type PageSortReq struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Sorts    string `json:"sorts" form:"sorts"` // 结构化排序 JSON 数组
}

// SortItem 是前端可传入的结构化排序项。
type SortItem struct {
	Field string `json:"field" form:"field"`
	Order string `json:"order" form:"order"` // asc/desc
}

// GetLimit 获取分页大小，支持默认值
func (i *PageSortReq) GetLimit(defaultSize ...int) int {
	if i.PageSize <= 0 {
		if len(defaultSize) > 0 {
			return defaultSize[0]
		}
		return 20
	}
	return i.PageSize
}

// GetPage 获取规范化后的当前页码
func (i *PageSortReq) GetPage() int {
	if i.Page < 1 {
		return 1
	}
	return i.Page
}

// GetOffset 获取分页偏移量
func (i *PageSortReq) GetOffset() int {
	return (i.GetPage() - 1) * i.GetLimit()
}

// GetOrder 获取可传给 GORM Order 的安全排序 SQL。
//
// 前端只传排序意图（如 sorts=[{"field":"created_at","order":"desc"}]），这里统一校验字段名并转换成
// `created_at` DESC, `name` ASC。业务代码不要从前端接收裸 SQL order by。
func (i *PageSortReq) GetOrder() string {
	return buildSortsFromJSON(i.Sorts)
}

// SafeColumn 检查列名是否安全（防SQL注入）
func SafeColumn(column string) bool {
	for _, c := range column {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// SafeColumnName 为列名添加反引号，防止关键字冲突
func SafeColumnName(column string) string {
	if !SafeColumn(column) {
		return column // 如果列名不安全，直接返回（会被后续验证拦截）
	}
	return "`" + column + "`"
}

func buildSortsFromJSON(sorts string) string {
	if !strings.HasPrefix(strings.TrimSpace(sorts), "[") {
		return ""
	}
	var items []SortItem
	if err := json.Unmarshal([]byte(sorts), &items); err != nil {
		return ""
	}
	return buildSortsFromItems(items)
}

func buildSortsFromItems(items []SortItem) string {
	if len(items) == 0 {
		return ""
	}

	sortFields := make([]string, 0, len(items))
	for _, item := range items {
		field := strings.TrimSpace(item.Field)
		if field == "" || !SafeColumn(field) {
			return ""
		}
		order := normalizeSortOrder(item.Order)
		if order == "" {
			return ""
		}
		sortFields = append(sortFields, fmt.Sprintf("%s %s", SafeColumnName(field), order))
	}
	return strings.Join(sortFields, ", ")
}

func normalizeSortOrder(order string) string {
	switch strings.TrimSpace(order) {
	case "asc":
		return "ASC"
	case "desc":
		return "DESC"
	default:
		return ""
	}
}
