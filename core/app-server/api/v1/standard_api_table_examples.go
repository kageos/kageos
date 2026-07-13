package v1

import (
	"fmt"
	"time"

	"github.com/kageos/kageos-sdk/agent-app/widget"
)

// generateExampleRows 根据字段类型生成示例数据行
// 返回：多行示例数据，每行是一个字段值的数组
// username: 当前用户名，用于创建用户字段的默认值
func generateExampleRows(fields []*widget.Field, username string) [][]interface{} {
	// 找出需要生成多行的字段（select/multiselect 和 bool/switch）
	var maxRows int = 1 // 最大行数

	for _, field := range fields {
		rowCount := 1
		switch field.Widget.Type {
		case widget.TypeSelect, widget.TypeMultiSelect:
			// 从 widget config 中获取选项
			if configMap, ok := field.Widget.Config.(map[string]interface{}); ok {
				// Options 是 []string 类型
				if options, ok := configMap["options"].([]interface{}); ok && len(options) > 0 {
					rowCount = len(options)
				} else if optionsStr, ok := configMap["options"].([]string); ok && len(optionsStr) > 0 {
					rowCount = len(optionsStr)
				}
			}
		case widget.TypeSwitch:
			// bool 类型：显示"是"和"否"两行
			rowCount = 2
		}

		if rowCount > maxRows {
			maxRows = rowCount
		}
	}

	// 生成示例数据行
	rows := make([][]interface{}, maxRows)
	for rowIndex := 0; rowIndex < maxRows; rowIndex++ {
		row := make([]interface{}, len(fields))
		for colIndex, field := range fields {
			row[colIndex] = generateExampleValueForRow(field, rowIndex, maxRows, username)
		}
		rows[rowIndex] = row
	}

	return rows
}

// generateExampleValueForRow 为指定行生成字段的示例值
// username: 当前用户名，用于创建用户字段的默认值
func generateExampleValueForRow(field *widget.Field, rowIndex int, maxRows int, username string) interface{} {
	dataType := "string"
	if field.Data != nil {
		dataType = field.Data.Type
	}

	widgetType := field.Widget.Type
	config, ok := field.Widget.Config.(map[string]interface{})

	// 处理系统字段
	switch field.Code {
	case "created_at", "updated_at":
		// 创建时间/更新时间：使用当前时间，格式为 2006-01-02 15:04:05
		now := time.Now()
		return now.Format("2006-01-02 15:04:05")
	case "created_by", "updated_by":
		// 创建用户/更新用户：使用当前用户名（必须获取到，否则返回空字符串）
		if username != "" {
			return username
		}
		return "" // 如果获取不到用户名，返回空字符串（前端会处理）
	}

	switch widgetType {
	case widget.TypeSelect, widget.TypeMultiSelect:
		// 获取所有选项，按行索引返回对应选项
		// Options 是 []string 类型，直接返回字符串
		if ok {
			// 尝试 []interface{} 类型（JSON 反序列化后）
			if options, ok := config["options"].([]interface{}); ok && len(options) > 0 {
				if rowIndex < len(options) {
					// 每个选项是字符串
					if optionStr, ok := options[rowIndex].(string); ok {
						return optionStr
					}
				}
				// 如果行数超过选项数，返回最后一个选项
				if len(options) > 0 {
					if optionStr, ok := options[len(options)-1].(string); ok {
						return optionStr
					}
				}
			} else if optionsStr, ok := config["options"].([]string); ok && len(optionsStr) > 0 {
				// 尝试 []string 类型（直接类型断言）
				if rowIndex < len(optionsStr) {
					return optionsStr[rowIndex]
				}
				// 如果行数超过选项数，返回最后一个选项
				return optionsStr[len(optionsStr)-1]
			}
		}
		// 没有选项，返回默认值
		return fmt.Sprintf("选项%d", rowIndex+1)

	case widget.TypeSwitch:
		// bool 类型：第一行显示"是"，第二行显示"否"
		if rowIndex == 0 {
			return "是"
		}
		return "否"

	case widget.TypeInteger, widget.TypeFloat:
		// 数字类型：使用默认值或示例数字
		if ok {
			if defaultVal, ok := config["default"]; ok {
				return defaultVal
			}
		}
		return 123

	case widget.TypeTextArea:
		// 多行文本：使用默认值或示例文本
		if ok {
			if defaultVal, ok := config["default"]; ok {
				return defaultVal
			}
		}
		return fmt.Sprintf("示例文本%d", rowIndex+1)

	case widget.TypeFiles:
		// 文件类型：显示为空
		return ""

	case widget.TypeUser:
		// 用户类型：如果是创建用户/更新用户字段，使用当前用户名；否则使用默认值
		if field.Code == "created_by" || field.Code == "updated_by" {
			if username != "" {
				return username
			}
			return "" // 如果获取不到用户名，返回空字符串（前端会处理）
		}
		if ok {
			if defaultVal, ok := config["default"]; ok {
				return defaultVal
			}
		}
		return "" // 默认值为空字符串

	default:
		// 其他类型：根据 dataType 判断
		if dataType == widget.DataTypeInt || dataType == widget.DataTypeFloat {
			if ok {
				if defaultVal, ok := config["default"]; ok {
					return defaultVal
				}
			}
			return 123
		}
		// 文本类型：使用默认值或示例文本
		if ok {
			if defaultVal, ok := config["default"]; ok {
				return defaultVal
			}
		}
		return fmt.Sprintf("示例文本%d", rowIndex+1)
	}
}
