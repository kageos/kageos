package widget

import (
	"encoding/json"
	"testing"
)

// 测试 select 组件的 creatable 参数
type SelectTestStruct struct {
	Status   string `json:"status" widget:"name:状态;type:select;options:待处理,处理中,已完成;creatable:true"`
	Priority string `json:"priority" widget:"name:优先级;type:select;options:低,中,高;creatable:false"`
}

func TestSelectCreatable(t *testing.T) {
	testStruct := &SelectTestStruct{}
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

	// 打印 JSON 结构
	jsonData, _ := json.MarshalIndent(fields, "", "  ")
	t.Logf("Select 组件 creatable 测试:\n%s", string(jsonData))

	// 验证 status 字段（支持创建）
	var statusField *Field
	for _, field := range fields {
		if field.Code == "status" {
			statusField = field
			break
		}
	}

	if statusField == nil {
		t.Fatal("未找到 status 字段")
	}

	statusConfig, ok := statusField.Widget.Config.(*Select)
	if !ok {
		t.Fatal("无法转换配置为 Select")
	}

	if !statusConfig.Creatable {
		t.Errorf("status字段应该支持创建，但creatable=%v", statusConfig.Creatable)
	}

	// 验证 priority 字段（不支持创建）
	var priorityField *Field
	for _, field := range fields {
		if field.Code == "priority" {
			priorityField = field
			break
		}
	}

	if priorityField == nil {
		t.Fatal("未找到 priority 字段")
	}

	priorityConfig, ok := priorityField.Widget.Config.(*Select)
	if !ok {
		t.Fatal("无法转换配置为 Select")
	}

	if priorityConfig.Creatable {
		t.Errorf("priority字段不应该支持创建，但creatable=%v", priorityConfig.Creatable)
	}

	t.Logf("✅ Select 组件 creatable 测试通过")
	t.Logf("  - Status: creatable=%v", statusConfig.Creatable)
	t.Logf("  - Priority: creatable=%v", priorityConfig.Creatable)
}











