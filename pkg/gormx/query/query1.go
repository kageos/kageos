package query

import (
	"encoding/json"
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

	// 分号分隔多个字段。
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
			// 切换字段前先保存当前字段。
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
