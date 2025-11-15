package widget

import (
	"encoding/json"
	"testing"
)

// 测试数据类型推断
type DataTypeTestStruct struct {
	// 字符串类型
	Name string `json:"name" widget:"name:名称;type:input"`

	// 整数类型
	Age int `json:"age" widget:"name:年龄;type:number"`

	// 浮点数类型
	Price float64 `json:"price" widget:"name:价格;type:float"`

	// 布尔类型
	IsVip bool `json:"is_vip" widget:"name:是否VIP;type:switch"`

	// 字符串数组 - 应该推断为 []string
	Tags []string `json:"tags" widget:"name:标签;type:multiselect"`

	// 整数数组 - 应该推断为 []int
	Pids []int `json:"pids" widget:"name:进程ID列表;type:multiselect"`

	// 浮点数数组 - 应该推断为 []float
	Scores []float64 `json:"scores" widget:"name:分数列表;type:multiselect"`

	// 单个字符串选择 - 应该推断为 string
	Status string `json:"status" widget:"name:状态;type:select"`

	// 单个整数选择 - 应该推断为 int
	Priority int `json:"priority" widget:"name:优先级;type:select"`
}

func TestDataTypeInference(t *testing.T) {
	testStruct := &DataTypeTestStruct{}
	result, err := ParseModelWithType(testStruct)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 转换为 Field 结构
	var fields []*Field
	for _, tag := range result.Tags {
		field := ConvertTagsToField(tag)
		fields = append(fields, field)
	}

	// 打印 JSON 查看数据类型
	jsonData, _ := json.MarshalIndent(fields, "", "  ")
	t.Logf("数据类型推断结果:\n%s", string(jsonData))

	// 验证各个字段的数据类型
	testCases := []struct {
		fieldCode    string
		expectedType string
		description  string
	}{
		{"name", DataTypeString, "字符串字段"},
		{"age", DataTypeInt, "整数字段"},
		{"price", DataTypeFloat, "浮点数字段"},
		{"is_vip", DataTypeBool, "布尔字段"},
		{"tags", DataTypeStrings, "字符串数组（[]string）"},
		{"pids", DataTypeInts, "整数数组（[]int）"},
		{"scores", DataTypeFloats, "浮点数数组（[]float64）"},
		{"status", DataTypeString, "单个字符串选择"},
		{"priority", DataTypeInt, "单个整数选择"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			var field *Field
			for _, f := range fields {
				if f.Code == tc.fieldCode {
					field = f
					break
				}
			}

			if field == nil {
				t.Fatalf("未找到字段: %s", tc.fieldCode)
			}

			if field.Data.Type != tc.expectedType {
				t.Errorf("字段 %s 数据类型错误，期望 %s，实际 %s",
					tc.fieldCode, tc.expectedType, field.Data.Type)
			} else {
				t.Logf("✅ %s: %s → %s", tc.fieldCode, tc.description, field.Data.Type)
			}
		})
	}
}









