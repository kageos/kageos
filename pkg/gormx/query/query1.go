package query

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// PaginatedTable 分页结果结构体
type PaginatedTable[T any] struct {
	Items       T     `json:"items" runner:"widget:table;type:array;code:items"` // 分页数据
	CurrentPage int   `json:"current_page" runner:"search_cond"`                 // 当前页码
	TotalCount  int64 `json:"total_count" runner:"search_cond"`                  // 总数据量
	TotalPages  int   `json:"total_pages" runner:"search_cond"`                  // 总页数
	PageSize    int   `json:"page_size" runner:"search_cond"`                    // 每页数量
}

// SearchFilterPageReq 分页参数结构体
type SearchFilterPageReq struct {
	Page     int    `json:"page" form:"page" runner:"search_cond;code:page"`
	PageSize int    `json:"page_size" form:"page_size" runner:"search_cond;code:page_size"`
	Sorts    string `json:"sorts" form:"sorts" runner:"search_cond;code:sorts"` //category:asc,price:desc

	Keyword string `json:"keyword" form:"keyword" runner:"search_cond;keyword"`
	// 查询条件
	Eq   []string `form:"eq" json:"eq" runner:"search_cond;code:like"`     // 格式：field:value
	Like []string `form:"like" json:"like" runner:"search_cond;code:like"` // 格式：field:value
	In   []string `form:"in" json:"in" runner:"search_cond;code:in"`       // 格式：field:value
	Gt   []string `form:"gt" json:"gt" runner:"search_cond;code:gt"`       // 格式：field:value
	Gte  []string `form:"gte" json:"gte" runner:"search_cond;code:gte"`    // 格式：field:value
	Lt   []string `form:"lt" json:"lt" runner:"search_cond;code:lt"`       // 格式：field:value
	Lte  []string `form:"lte" json:"lte" runner:"search_cond;code:lte"`    // 格式：field:value
	// 否定查询条件
	NotEq   []string `form:"not_eq" json:"not_eq" runner:"search_cond;code:not_eq"`       // 格式：field:value
	NotLike []string `form:"not_like" json:"not_like" runner:"search_cond;code:not_like"` // 格式：field:value
	NotIn   []string `form:"not_in" json:"not_in" runner:"search_cond;code:not_in"`       // 格式：field:value
}

// normalizeSortField 标准化排序字段格式
func normalizeSortField(sort string) string {
	sort = strings.TrimSpace(sort)

	// 如果已经包含 :asc 或 :desc，直接返回
	if strings.Contains(sort, ":asc") || strings.Contains(sort, ":desc") {
		return sort
	}

	// 处理减号前缀格式
	if strings.HasPrefix(sort, "-") {
		return strings.ReplaceAll(sort, "-", "") + ":desc"
	}

	// 默认添加 :asc
	return sort + ":asc"
}

func (r *SearchFilterPageReq) WithSorts(sorts string) *SearchFilterPageReq {
	if sorts == "" {
		return r
	}

	// 解析现有的排序条件
	var existingFields []string
	var existingMap = make(map[string]string)

	if r.Sorts != "" {
		existingSorts := strings.Split(r.Sorts, ",")
		for _, sort := range existingSorts {
			normalized := normalizeSortField(sort)
			parts := strings.Split(normalized, ":")
			if len(parts) == 2 {
				field := parts[0]
				existingMap[field] = normalized
				existingFields = append(existingFields, field)
			}
		}
	}

	// 处理新的排序条件，只添加不存在的字段
	var newFields []string
	for _, sort := range strings.Split(sorts, ",") {
		normalized := normalizeSortField(sort)
		parts := strings.Split(normalized, ":")
		if len(parts) == 2 {
			field := parts[0]

			// 检查字段是否已存在
			found := false
			for _, ef := range existingFields {
				if ef == field {
					found = true
					break
				}
			}

			// 只有不存在的字段才添加
			if !found {
				existingMap[field] = normalized
				newFields = append(newFields, field)
			}
		}
	}

	// 重建排序列表，保持现有字段的顺序，然后添加新字段
	var result []string

	// 先添加现有字段（保持原有顺序）
	for _, field := range existingFields {
		if sort, exists := existingMap[field]; exists {
			result = append(result, sort)
		}
	}

	// 再添加新字段
	for _, field := range newFields {
		if sort, exists := existingMap[field]; exists {
			result = append(result, sort)
		}
	}

	r.Sorts = strings.Join(result, ",")
	return r
}

// QueryConfig 查询配置
type QueryConfig struct {
	Fields    map[string][]string // 字段名 -> 允许的操作符列表（白名单）
	Blacklist map[string]struct{} // 不允许查询的字段（黑名单）
}

// NewQueryConfig 创建查询配置
func NewQueryConfig() *QueryConfig {
	return &QueryConfig{
		Fields:    make(map[string][]string),
		Blacklist: make(map[string]struct{}),
	}
}

// AllowField 允许字段查询
func (c *QueryConfig) AllowField(field string, operators ...string) {
	c.Fields[field] = operators
}

// DenyField 禁止字段查询
func (c *QueryConfig) DenyField(field string) {
	c.Blacklist[field] = struct{}{}
}

// GetLimit 获取分页大小，支持默认值
func (i *SearchFilterPageReq) GetLimit(defaultSize ...int) int {
	if i.PageSize <= 0 {
		if len(defaultSize) > 0 {
			return defaultSize[0]
		}
		return 20
	}
	return i.PageSize
}

// GetOffset 获取分页偏移量
func (i *SearchFilterPageReq) GetOffset() int {
	page := i.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * i.GetLimit()
	return offset
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

// ParseSortFields 解析排序字段字符串
func ParseSortFields(sortStr string) ([]string, error) {
	if sortStr == "" {
		return nil, nil
	}

	parts := strings.Split(sortStr, ",")
	var sortFields []string

	for _, part := range parts {
		fieldOrder := strings.Split(part, ":")
		if len(fieldOrder) != 2 {
			return nil, fmt.Errorf("排序字段格式错误：%s，应为 field:order 格式", part)
		}

		field := strings.TrimSpace(fieldOrder[0])
		order := strings.TrimSpace(fieldOrder[1])

		if !SafeColumn(field) {
			return nil, fmt.Errorf("无效的排序字段名：%s", field)
		}

		order = strings.ToUpper(order)
		if order != "ASC" && order != "DESC" {
			return nil, fmt.Errorf("无效的排序方向：%s", order)
		}

		sortFields = append(sortFields, fmt.Sprintf("%s %s", SafeColumnName(field), order))
	}

	return sortFields, nil
}

// GetSorts 获取排序SQL
func (i *SearchFilterPageReq) GetSorts() string {
	sortFields, err := ParseSortFields(i.Sorts)
	if err != nil || len(sortFields) == 0 {
		return ""
	}
	return strings.Join(sortFields, ", ")
}

// parseFieldValues 解析字段和值
func parseFieldValues(input string) (map[string]string, error) {
	if input == "" {
		return nil, nil
	}

	result := make(map[string]string)
	pairs := strings.Split(input, ",")

	for _, pair := range pairs {
		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("参数格式错误：%s，应为 field:value 格式", pair)
		}

		field := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if !SafeColumn(field) {
			return nil, fmt.Errorf("无效的字段名：%s", field)
		}

		result[field] = value
	}

	return result, nil
}

// parseInValues 解析IN查询的字段和值
func parseInValues(input string) (map[string][]string, error) {
	if input == "" {
		return nil, nil
	}

	result := make(map[string][]string)

	// 查找第一个冒号的位置
	colonIndex := strings.Index(input, ":")
	if colonIndex == -1 {
		return nil, fmt.Errorf("参数格式错误：%s，应为 field:value1,value2,value3 格式", input)
	}
	// 提取字段名
	field := strings.TrimSpace(input[:colonIndex])
	if !SafeColumn(field) {
		return nil, fmt.Errorf("无效的字段名：%s", field)
	}
	// 提取值部分
	valuesPart := strings.TrimSpace(input[colonIndex+1:])
	if valuesPart == "" {
		return nil, fmt.Errorf("参数格式错误：%s，值不能为空", input)
	}

	// 按逗号分割值
	values := strings.Split(valuesPart, ",")
	for _, value := range values {
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue != "" {
			result[field] = append(result[field], trimmedValue)
		}
	}

	return result, nil
}

// validateField 验证字段
func validateField(field, operator string, config *QueryConfig) error {
	// 如果配置为 nil，只进行基本的安全检查
	if config == nil {
		if !SafeColumn(field) {
			return fmt.Errorf("无效的字段名：%s", field)
		}
		return nil
	}

	// 检查字段是否在黑名单中
	if _, ok := config.Blacklist[field]; ok {
		return fmt.Errorf("字段 %s 被禁止查询", field)
	}

	// 如果配置了白名单，则检查字段是否在白名单中
	if len(config.Fields) > 0 {
		allowedOperators, ok := config.Fields[field]
		if !ok {
			return fmt.Errorf("不允许查询字段: %s", field)
		}

		// 检查操作符是否允许
		if !contains(allowedOperators, operator) {
			return fmt.Errorf("字段 %s 不支持 %s 操作符", field, operator)
		}
	}

	return nil
}

// validateAndBuildCondition 验证并构建查询条件
func validateAndBuildCondition(db **gorm.DB, inputs []string, operator string, config *QueryConfig) error {
	if len(inputs) == 0 {
		return nil
	}

	if operator == "in" {
		// 合并所有输入的条件
		allConditions := make(map[string][]string)
		for _, input := range inputs {
			conditions, err := parseInValues(input)
			if err != nil {
				return err
			}
			// 合并相同字段的值
			for field, values := range conditions {
				if err := validateField(field, operator, config); err != nil {
					return err
				}
				allConditions[field] = append(allConditions[field], values...)
			}
		}
		// 构建最终的查询条件
		for field, values := range allConditions {
			// 尝试将值转换为适当的类型
			convertedValues := make([]interface{}, len(values))
			hasBool := false

			for i, value := range values {
				// 尝试转换为数字
				if numValue, err := strconv.ParseInt(value, 10, 64); err == nil {
					convertedValues[i] = numValue
				} else if boolValue, err := strconv.ParseBool(value); err == nil {
					// 尝试转换为布尔值
					convertedValues[i] = boolValue
					hasBool = true
				} else {
					// 保持为字符串
					convertedValues[i] = value
				}
			}

			// 如果包含布尔值，使用布尔值查询
			if hasBool {
				*db = (*db).Where(SafeColumnName(field)+" IN ?", convertedValues)
			} else {
				*db = (*db).Where(SafeColumnName(field)+" IN ?", convertedValues)
			}
		}
		return nil
	}

	if operator == "not_in" {
		// 合并所有输入的条件
		allConditions := make(map[string][]string)
		for _, input := range inputs {
			conditions, err := parseInValues(input)
			if err != nil {
				return err
			}
			// 合并相同字段的值
			for field, values := range conditions {
				if err := validateField(field, operator, config); err != nil {
					return err
				}
				allConditions[field] = append(allConditions[field], values...)
			}
		}
		// 构建最终的查询条件
		for field, values := range allConditions {
			// 尝试将值转换为适当的类型
			convertedValues := make([]interface{}, len(values))
			hasBool := false

			for i, value := range values {
				// 尝试转换为数字
				if numValue, err := strconv.ParseInt(value, 10, 64); err == nil {
					convertedValues[i] = numValue
				} else if boolValue, err := strconv.ParseBool(value); err == nil {
					// 尝试转换为布尔值
					convertedValues[i] = boolValue
					hasBool = true
				} else {
					// 保持为字符串
					convertedValues[i] = value
				}
			}

			// 如果包含布尔值，使用布尔值查询
			if hasBool {
				*db = (*db).Where(SafeColumnName(field)+" NOT IN ?", convertedValues)
			} else {
				*db = (*db).Where(SafeColumnName(field)+" NOT IN ?", convertedValues)
			}
		}
		return nil
	}

	// 处理其他操作符
	for _, input := range inputs {
		conditions, err := parseFieldValues(input)
		if err != nil {
			return err
		}

		for field, value := range conditions {
			if err := validateField(field, operator, config); err != nil {
				return err
			}

			// 对于 like 和 not_like 操作符，始终使用字符串比较
			if operator == "like" || operator == "not_like" {
				// 使用字符串比较
				switch operator {
				case "like":
					*db = (*db).Where(SafeColumnName(field)+" LIKE ?", "%"+value+"%")
				case "not_like":
					*db = (*db).Where(SafeColumnName(field)+" NOT LIKE ?", "%"+value+"%")
				}
			} else {
				// 尝试将值转换为数字
				numValue, err := strconv.ParseInt(value, 10, 64)
				if err == nil {
					// 如果是数字，使用数字比较
					switch operator {
					case "eq":
						*db = (*db).Where(SafeColumnName(field)+" = ?", numValue)
					case "not_eq":
						*db = (*db).Where(SafeColumnName(field)+" != ?", numValue)
					case "gt":
						*db = (*db).Where(SafeColumnName(field)+" > ?", numValue)
					case "gte":
						*db = (*db).Where(SafeColumnName(field)+" >= ?", numValue)
					case "lt":
						*db = (*db).Where(SafeColumnName(field)+" < ?", numValue)
					case "lte":
						*db = (*db).Where(SafeColumnName(field)+" <= ?", numValue)
					}
				} else {
					// 尝试将值转换为布尔值
					boolValue, err := strconv.ParseBool(value)
					if err == nil {
						// 如果是布尔值，使用布尔比较
						switch operator {
						case "eq":
							*db = (*db).Where(SafeColumnName(field)+" = ?", boolValue)
						case "not_eq":
							*db = (*db).Where(SafeColumnName(field)+" != ?", boolValue)
						}
					} else {
						// 如果不是布尔值，使用字符串比较
						switch operator {
						case "eq":
							*db = (*db).Where(SafeColumnName(field)+" = ?", value)
						case "not_eq":
							*db = (*db).Where(SafeColumnName(field)+" != ?", value)
						case "gt":
							*db = (*db).Where(SafeColumnName(field)+" > ?", value)
						case "gte":
							*db = (*db).Where(SafeColumnName(field)+" >= ?", value)
						case "lt":
							*db = (*db).Where(SafeColumnName(field)+" < ?", value)
						case "lte":
							*db = (*db).Where(SafeColumnName(field)+" <= ?", value)
						}
					}
				}
			}
		}
	}

	return nil
}

// AutoPaginateTable 自动分页查询
func AutoPaginateTable[T any](
	ctx context.Context,
	db *gorm.DB,
	model interface{},
	data T,
	pageInfo *SearchFilterPageReq,
	configs ...*QueryConfig,
) (*PaginatedTable[T], error) {
	if pageInfo == nil {
		pageInfo = new(SearchFilterPageReq)
	}

	// 修复：克隆数据库连接，避免污染原始连接
	dbClone := db.Session(&gorm.Session{})

	// 构建查询条件到克隆的连接
	if err := buildWhereConditions(&dbClone, pageInfo, configs...); err != nil {
		return nil, err
	}

	// 获取分页大小
	pageSize := pageInfo.GetLimit()
	offset := pageInfo.GetOffset()

	// 查询总数
	var totalCount int64
	if err := dbClone.Model(model).Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("分页查询统计总数失败: %w", err)
	}

	// 应用排序条件
	sortStr := pageInfo.GetSorts()
	if sortStr != "" {
		dbClone = dbClone.Order(sortStr)
	}

	// 查询当前页数据
	if err := dbClone.Offset(offset).Limit(pageSize).Find(data).Error; err != nil {
		return nil, fmt.Errorf("分页查询数据失败: %w", err)
	}

	// 计算总页数
	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	return &PaginatedTable[T]{
		Items:       data,
		CurrentPage: pageInfo.Page,
		TotalCount:  totalCount,
		TotalPages:  totalPages,
		PageSize:    pageSize,
	}, nil
}

// ApplySearchConditions 应用搜索条件到GORM查询（公开方法）
// 这个方法可以被其他库调用，用于在任何GORM查询中应用搜索条件
//
// 使用示例：
//
//	db, err := query.ApplySearchConditions(db, pageInfo)
//	if err != nil {
//	    return err
//	}
//
// 支持的搜索操作符：
//   - eq: 精确匹配
//   - like: 模糊匹配
//   - in: 包含查询
//   - gt/gte: 大于/大于等于
//   - lt/lte: 小于/小于等于
//   - not_eq: 不等于
//   - not_like: 否定模糊匹配
//   - not_in: 否定包含查询
func ApplySearchConditions(db *gorm.DB, pageInfo *SearchFilterPageReq, configs ...*QueryConfig) (*gorm.DB, error) {
	if pageInfo == nil {
		return db, nil
	}

	// 修复：克隆数据库连接，避免污染原始连接
	// 因为buildWhereConditions会直接修改传入的db指针，所以需要先克隆
	dbClone := db.Session(&gorm.Session{})

	// 应用搜索条件到克隆的连接
	var dbPtr *gorm.DB = dbClone
	err := buildWhereConditions(&dbPtr, pageInfo, configs...)
	if err != nil {
		return db, err
	}

	// 再次克隆，确保返回的连接完全独立
	finalDB := dbPtr.Session(&gorm.Session{})
	return finalDB, nil
}

// SimplePaginate 简单分页查询（公开方法）
// 这是一个便捷方法，适用于不需要复杂配置的场景
//
// 使用示例：
//
//	var products []Product
//	result, err := query.SimplePaginate(db, &Product{}, &products, pageInfo)
//	if err != nil {
//	    return err
//	}
//
// 参数说明：
//   - db: GORM数据库连接
//   - model: 模型实例，用于获取表信息
//   - dest: 查询结果存储的切片指针
//   - pageInfo: 分页和搜索参数
func SimplePaginate(db *gorm.DB, model interface{}, dest interface{}, pageInfo *SearchFilterPageReq) (*PaginatedTable[interface{}], error) {
	if pageInfo == nil {
		pageInfo = &SearchFilterPageReq{PageSize: 20}
	}

	// 应用搜索条件
	dbWithConditions, err := ApplySearchConditions(db, pageInfo)
	if err != nil {
		return nil, fmt.Errorf("应用搜索条件失败: %w", err)
	}

	// 获取分页参数
	pageSize := pageInfo.GetLimit()
	offset := pageInfo.GetOffset()

	// 查询总数
	var totalCount int64
	if err := dbWithConditions.Model(model).Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("查询总数失败: %w", err)
	}

	// 应用排序和分页
	if pageInfo.GetSorts() != "" {
		dbWithConditions = dbWithConditions.Order(pageInfo.GetSorts())
	}

	if err := dbWithConditions.Offset(offset).Limit(pageSize).Find(dest).Error; err != nil {
		return nil, fmt.Errorf("分页查询数据失败: %w", err)
	}

	// 计算总页数
	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	return &PaginatedTable[interface{}]{
		Items:       dest,
		CurrentPage: pageInfo.Page,
		TotalCount:  totalCount,
		TotalPages:  totalPages,
		PageSize:    pageSize,
	}, nil
}

// buildWhereConditions 构建查询条件
func buildWhereConditions(db **gorm.DB, pageInfo *SearchFilterPageReq, configs ...*QueryConfig) error {
	// 如果没有配置，直接构建查询条件
	if len(configs) == 0 {
		return buildWhereConditionsWithoutConfig(db, pageInfo)
	}

	// 合并所有配置
	config := mergeConfigs(configs...)

	// 验证并构建等于条件
	if err := validateAndBuildCondition(db, pageInfo.Eq, "eq", config); err != nil {
		return err
	}

	// 验证并构建模糊匹配条件
	if err := validateAndBuildCondition(db, pageInfo.Like, "like", config); err != nil {
		return err
	}

	// 验证并构建IN查询条件
	if err := validateAndBuildCondition(db, pageInfo.In, "in", config); err != nil {
		return err
	}

	// 验证并构建大于条件
	if err := validateAndBuildCondition(db, pageInfo.Gt, "gt", config); err != nil {
		return err
	}

	// 验证并构建大于等于条件
	if err := validateAndBuildCondition(db, pageInfo.Gte, "gte", config); err != nil {
		return err
	}

	// 验证并构建小于条件
	if err := validateAndBuildCondition(db, pageInfo.Lt, "lt", config); err != nil {
		return err
	}

	// 验证并构建小于等于条件
	if err := validateAndBuildCondition(db, pageInfo.Lte, "lte", config); err != nil {
		return err
	}

	// 验证并构建不等于条件
	if err := validateAndBuildCondition(db, pageInfo.NotEq, "not_eq", config); err != nil {
		return err
	}

	// 验证并构建不模糊匹配条件
	if err := validateAndBuildCondition(db, pageInfo.NotLike, "not_like", config); err != nil {
		return err
	}

	// 验证并构建NOT IN查询条件
	if err := validateAndBuildCondition(db, pageInfo.NotIn, "not_in", config); err != nil {
		return err
	}

	return nil
}

// buildWhereConditionsWithoutConfig 无配置构建查询条件
func buildWhereConditionsWithoutConfig(db **gorm.DB, pageInfo *SearchFilterPageReq) error {
	// 构建等于条件
	if err := validateAndBuildCondition(db, pageInfo.Eq, "eq", nil); err != nil {
		return err
	}

	// 构建模糊匹配条件
	if err := validateAndBuildCondition(db, pageInfo.Like, "like", nil); err != nil {
		return err
	}

	// 构建IN查询条件
	if err := validateAndBuildCondition(db, pageInfo.In, "in", nil); err != nil {
		return err
	}

	// 构建大于条件
	if err := validateAndBuildCondition(db, pageInfo.Gt, "gt", nil); err != nil {
		return err
	}

	// 构建大于等于条件
	if err := validateAndBuildCondition(db, pageInfo.Gte, "gte", nil); err != nil {
		return err
	}

	// 构建小于条件
	if err := validateAndBuildCondition(db, pageInfo.Lt, "lt", nil); err != nil {
		return err
	}

	// 构建小于等于条件
	if err := validateAndBuildCondition(db, pageInfo.Lte, "lte", nil); err != nil {
		return err
	}

	// 验证并构建不等于条件
	if err := validateAndBuildCondition(db, pageInfo.NotEq, "not_eq", nil); err != nil {
		return err
	}

	// 验证并构建不模糊匹配条件
	if err := validateAndBuildCondition(db, pageInfo.NotLike, "not_like", nil); err != nil {
		return err
	}

	// 验证并构建NOT IN查询条件
	if err := validateAndBuildCondition(db, pageInfo.NotIn, "not_in", nil); err != nil {
		return err
	}

	return nil
}

// mergeConfigs 合并多个配置
func mergeConfigs(configs ...*QueryConfig) *QueryConfig {
	merged := NewQueryConfig()

	for _, config := range configs {
		if config == nil {
			continue
		}

		// 合并白名单
		for field, operators := range config.Fields {
			if existing, ok := merged.Fields[field]; ok {
				existing = append(existing, operators...)
				existing = removeDuplicates(existing)
				merged.Fields[field] = existing
			} else {
				merged.Fields[field] = operators
			}
		}

		// 合并黑名单
		for field := range config.Blacklist {
			merged.Blacklist[field] = struct{}{}
		}
	}

	return merged
}

// removeDuplicates 去除切片中的重复元素
func removeDuplicates(slice []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)

	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}

	return result
}

// contains 检查切片是否包含指定值
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
