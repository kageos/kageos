package widget

import (
	"encoding/json"
	"testing"
)

// 测试 multiselect 组件的数据结构
type MultiSelectTestStruct struct {
	Tags       []string `json:"tags" widget:"name:标签;type:multiselect;options:前端,后端,全栈,DevOps,测试;default:前端,后端"`
	Categories []string `json:"categories" widget:"name:分类;type:multiselect;options:技术,产品,设计,运营;placeholder:请选择分类;max_count:3"`
	Skills     []string `json:"skills" widget:"name:技能;type:multiselect;options:Go,Python,Java;creatable:true"` // 支持创建
	Languages  []string `json:"languages" widget:"name:语言;type:multiselect;options:中文,英文,日文;creatable:false"`   // 不支持创建
}

func TestMultiSelect(t *testing.T) {
	t.Run("解析multiselect组件", func(t *testing.T) {
		testStruct := &MultiSelectTestStruct{}
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
		t.Logf("MultiSelect 组件 JSON:\n%s", string(jsonData))

		// 验证 tags 字段
		var tagsField *Field
		for _, field := range fields {
			if field.Code == "tags" {
				tagsField = field
				break
			}
		}

		if tagsField == nil {
			t.Fatal("未找到 tags 字段")
		}

		if tagsField.Widget.Type != TypeMultiSelect {
			t.Errorf("tags字段类型错误，期望%s，实际%s", TypeMultiSelect, tagsField.Widget.Type)
		}

		if tagsField.Data.Type != DataTypeStrings {
			t.Errorf("tags字段数据类型错误，期望%s，实际%s", DataTypeStrings, tagsField.Data.Type)
		}

		// 验证配置
		config, ok := tagsField.Widget.Config.(*MultiSelect)
		if !ok {
			t.Fatal("无法转换配置为 MultiSelect")
		}

		if len(config.Options) != 5 {
			t.Errorf("选项数量错误，期望5，实际%d", len(config.Options))
		}

		expectedOptions := []string{"前端", "后端", "全栈", "DevOps", "测试"}
		for i, expected := range expectedOptions {
			if i < len(config.Options) && config.Options[i] != expected {
				t.Errorf("选项[%d]错误，期望%s，实际%s", i, expected, config.Options[i])
			}
		}

		if len(config.Default) != 2 {
			t.Errorf("默认值数量错误，期望2，实际%d", len(config.Default))
		}

		// 验证 categories 字段
		var categoriesField *Field
		for _, field := range fields {
			if field.Code == "categories" {
				categoriesField = field
				break
			}
		}

		if categoriesField == nil {
			t.Fatal("未找到 categories 字段")
		}

		catConfig, ok := categoriesField.Widget.Config.(*MultiSelect)
		if !ok {
			t.Fatal("无法转换配置为 MultiSelect")
		}

		if catConfig.Placeholder != "请选择分类" {
			t.Errorf("占位符错误，期望'请选择分类'，实际'%s'", catConfig.Placeholder)
		}

		// 验证 skills 字段（支持创建）
		var skillsField *Field
		for _, field := range fields {
			if field.Code == "skills" {
				skillsField = field
				break
			}
		}

		if skillsField == nil {
			t.Fatal("未找到 skills 字段")
		}

		skillsConfig, ok := skillsField.Widget.Config.(*MultiSelect)
		if !ok {
			t.Fatal("无法转换配置为 MultiSelect")
		}

		if !skillsConfig.Creatable {
			t.Errorf("skills字段应该支持创建，但creatable=%v", skillsConfig.Creatable)
		}

		// 验证 languages 字段（不支持创建）
		var languagesField *Field
		for _, field := range fields {
			if field.Code == "languages" {
				languagesField = field
				break
			}
		}

		if languagesField == nil {
			t.Fatal("未找到 languages 字段")
		}

		langConfig, ok := languagesField.Widget.Config.(*MultiSelect)
		if !ok {
			t.Fatal("无法转换配置为 MultiSelect")
		}

		if langConfig.Creatable {
			t.Errorf("languages字段不应该支持创建，但creatable=%v", langConfig.Creatable)
		}

		t.Logf("✅ MultiSelect 组件测试通过")
		t.Logf("  - Tags: %d个选项，%d个默认值", len(config.Options), len(config.Default))
		t.Logf("  - Categories: 占位符='%s', 最大选择数=%d", catConfig.Placeholder, catConfig.MaxCount)
		t.Logf("  - Skills: creatable=%v", skillsConfig.Creatable)
		t.Logf("  - Languages: creatable=%v", langConfig.Creatable)
	})
}
