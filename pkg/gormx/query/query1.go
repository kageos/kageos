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
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Sorts    string `json:"sorts" form:"sorts"` //category:asc,price:desc

	Keyword string `json:"keyword" form:"keyword"`
	// 查询条件
	Eq       []string `form:"eq" json:"eq"`             // 格式：field:value
	Like     []string `form:"like" json:"like"`         // 格式：field:value
	In       []string `form:"in" json:"in"`             // 格式：field:value
	Contains []string `form:"contains" json:"contains"` // 格式：field:value1,value2（用于多选场景，使用 FIND_IN_SET）
	Gt       []string `form:"gt" json:"gt"`             // 格式：field:value
	Gte      []string `form:"gte" json:"gte"`           // 格式：field:value
	Lt       []string `form:"lt" json:"lt"`             // 格式：field:value
	Lte      []string `form:"lte" json:"lte"`           // 格式：field:value
	// 否定查询条件
	NotEq   []string `form:"not_eq" json:"not_eq"`     // 格式：field:value
	NotLike []string `form:"not_like" json:"not_like"` // 格式：field:value
	NotIn   []string `form:"not_in" json:"not_in"`     // 格式：field:value
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
	pairs := splitCommaOutsideParens(input)

	for _, pair := range pairs {
		parts := strings.SplitN(pair, ":", 2)
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

func splitCommaOutsideParens(input string) []string {
	parts := make([]string, 0, 2)
	start := 0
	depth := 0
	var quote rune

	for i, r := range input {
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, input[start:i])
				start = i + 1
			}
		}
	}

	parts = append(parts, input[start:])
	return parts
}

// parseInValues 解析IN查询的字段和值
// 支持两种格式：
// 1. 单个字段：field:value1,value2
// 2. 多个字段：field1:value1,value2,field2:value3,value4（使用逗号分隔多个字段，与 in 操作符一致）
// 注意：通过查找 "field:" 模式来识别字段边界，避免与值中的逗号混淆
func parseInValues(input string) (map[string][]string, error) {
	if input == "" {
		return nil, nil
	}

	result := make(map[string][]string)

	// 🔥 向后兼容：如果包含分号，说明是多个字段（旧格式）
	// 格式：field1:value1,value2;field2:value3,value4
	if strings.Contains(input, ";") {
		parts := strings.Split(input, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			// 解析单个字段部分
			fieldResult, err := parseSingleFieldInValues(part)
			if err != nil {
				return nil, err
			}
			// 合并到结果中
			for field, values := range fieldResult {
				result[field] = append(result[field], values...)
			}
		}
		return result, nil
	}

	// 🔥 智能解析：通过查找 "field:" 模式来分割多个字段（与 in 操作符一致）
	// 格式：field1:value1,value2,field2:value3,value4
	// 通过查找冒号前的内容是否为有效字段名来识别字段边界
	// 但是，如果只有一个字段，直接使用 parseSingleFieldInValues 更简单高效
	parts := strings.Split(input, ",")

	// 🔥 先检查是否只有一个字段（格式：field:value1,value2）
	// 如果第一个部分包含冒号，且冒号前是有效字段名，可能是单个字段
	if len(parts) > 0 {
		firstPart := strings.TrimSpace(parts[0])
		if strings.Contains(firstPart, ":") {
			colonIndex := strings.Index(firstPart, ":")
			field := strings.TrimSpace(firstPart[:colonIndex])
			// 如果第一个部分是有效的字段名，检查后面是否有其他字段
			if SafeColumn(field) {
				// 检查后续部分是否包含其他字段（通过查找 "field:" 模式）
				hasOtherFields := false
				for i := 1; i < len(parts); i++ {
					part := strings.TrimSpace(parts[i])
					if strings.Contains(part, ":") {
						partColonIndex := strings.Index(part, ":")
						partField := strings.TrimSpace(part[:partColonIndex])
						if SafeColumn(partField) {
							hasOtherFields = true
							break
						}
					}
				}
				// 如果没有其他字段，直接使用 parseSingleFieldInValues
				if !hasOtherFields {
					return parseSingleFieldInValues(input)
				}
			}
		}
	}

	// 🔥 多个字段的情况：通过查找 "field:" 模式来分割
	var currentField string
	var currentValues []string

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 检查是否包含冒号（可能是新字段的开始）
		if strings.Contains(part, ":") {
			// 如果之前有字段，先保存它
			if currentField != "" && len(currentValues) > 0 {
				result[currentField] = append(result[currentField], currentValues...)
				currentValues = []string{}
			}

			// 解析新字段
			colonIndex := strings.Index(part, ":")
			field := strings.TrimSpace(part[:colonIndex])
			value := strings.TrimSpace(part[colonIndex+1:])

			// 验证字段名是否有效（简单检查：只包含字母、数字、下划线）
			if SafeColumn(field) {
				currentField = field
				if value != "" {
					currentValues = []string{value}
				} else {
					currentValues = []string{}
				}
			} else {
				// 如果不是有效字段名，可能是值的一部分
				if currentField != "" {
					currentValues = append(currentValues, part)
				} else {
					// 如果没有当前字段，可能是单个字段格式，尝试解析
					return parseSingleFieldInValues(input)
				}
			}
		} else {
			// 没有冒号，应该是当前字段的值
			if currentField != "" {
				currentValues = append(currentValues, part)
			} else {
				// 如果没有当前字段，可能是单个字段格式，尝试解析
				if i == 0 {
					// 第一个部分没有冒号，可能是单个字段格式，回退到 parseSingleFieldInValues
					return parseSingleFieldInValues(input)
				}
				return nil, fmt.Errorf("参数格式错误：%s，无法识别字段名", part)
			}
		}
	}

	// 保存最后一个字段
	if currentField != "" && len(currentValues) > 0 {
		result[currentField] = append(result[currentField], currentValues...)
	}

	// 如果成功解析出多个字段，返回结果
	if len(result) > 0 {
		return result, nil
	}

	// 否则，按单个字段格式解析
	return parseSingleFieldInValues(input)
}

// parseSingleFieldInValues 解析单个字段的 IN 值
func parseSingleFieldInValues(input string) (map[string][]string, error) {
	result := make(map[string][]string)

	// 查找第一个冒号的位置
	colonIndex := strings.Index(input, ":")
	if colonIndex == -1 {
		return nil, fmt.Errorf("参数格式错误：%s，应为 field:value1,value2 格式", input)
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

	if operator == "contains" {
		// 🔥 contains 操作符：用于多选场景，使用 MySQL 的 FIND_IN_SET 函数
		// 格式：field:value1,value2（逗号分隔的多个值）
		// 生成 SQL: FIND_IN_SET('value1', field) OR FIND_IN_SET('value2', field)
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
			if len(values) == 0 {
				continue
			}
			// 🔥 使用 SQLite 兼容的方式实现 FIND_IN_SET 功能
			// SQLite 不支持 FIND_IN_SET，使用 LIKE 和边界检查来实现相同功能
			// 原理：在字段值前后加上逗号，然后检查 ',value,' 是否存在于 ',field_value,'
			// 例如：',紧急,' LIKE '%,紧急,%' OR ',重要,' LIKE '%,重要,%'
			// 这样可以精确匹配逗号分隔的值，避免误匹配（如 "高优先级" 不会匹配 "高"）
			var conditions []string
			var args []interface{}
			for _, value := range values {
				value = strings.TrimSpace(value)
				if value != "" {
					// SQLite 兼容方式：使用 LIKE 和边界检查
					// (',' || field || ',' LIKE '%,' || ? || ',%')
					// 或者使用 instr 函数：instr(',' || field || ',', ',' || ? || ',') > 0
					// 使用 instr 更高效
					conditions = append(conditions, "instr(',' || "+SafeColumnName(field)+" || ',', ',' || ? || ',') > 0")
					args = append(args, value)
				}
			}
			if len(conditions) > 0 {
				query := "(" + strings.Join(conditions, " OR ") + ")"
				*db = (*db).Where(query, args...)
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

	// 验证并构建CONTAINS查询条件（用于多选场景）
	if err := validateAndBuildCondition(db, pageInfo.Contains, "contains", config); err != nil {
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

	// 构建CONTAINS查询条件（用于多选场景）
	if err := validateAndBuildCondition(db, pageInfo.Contains, "contains", nil); err != nil {
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
