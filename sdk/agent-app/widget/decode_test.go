package widget

import (
	"testing"
)

// 测试用的结构体
type TestCrmTicket struct {
	ID          int    `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" permission:"read"`
	Title       string `json:"title" gorm:"column:title" widget:"name:工单标题;type:input" search:"like" validate:"required,min=2,max=200"`
	Description string `json:"description" gorm:"column:description" widget:"name:问题描述;type:text_area" validate:"required,min=10"`
	Priority    string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;default:中" validate:"required,oneof=低,中,高"`
	Status      string `json:"status" gorm:"column:status" widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;default:待处理" validate:"required,oneof=待处理,处理中,已完成,已关闭"`
	Phone       string `json:"phone" gorm:"column:phone" widget:"name:联系电话;type:input" validate:"required,min=11,max=20"`
	CreateBy    string `json:"create_by" gorm:"column:create_by" widget:"name:创建用户;type:user" permission:"read"`
	DeletedAt   string `json:"deleted_at" gorm:"column:deleted_at" widget:"-"` // 隐藏字段
}

type TestSearchReq struct {
	SelfOnly string `json:"self_only" widget:"name:只看我的;type:switch;default:false"`
}

// 新增一个测试结构体来测试default在widget中的情况
type TestStructWithDefault struct {
	Priority string `json:"priority" widget:"name:优先级;type:select;options:低,中,高;default:中"`
	Status   string `json:"status"   widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;default:待处理"`
}

func TestParseModel(t *testing.T) {
	// 测试解析结构体
	ticket := &TestCrmTicket{}
	tags, err := ParseModel(ticket)
	if err != nil {
		t.Fatalf("ParseModel failed: %v", err)
	}

	if len(tags) == 0 {
		t.Fatal("Expected at least one field, got none")
	}

	// 检查第一个字段的解析结果
	titleTag := tags[1] // Title字段
	if titleTag.Json != "title" {
		t.Errorf("Expected json tag 'title', got '%s'", titleTag.Json)
	}

	if titleTag.WidgetParsed["name"] != "工单标题" {
		t.Errorf("Expected widget name '工单标题', got '%s'", titleTag.WidgetParsed["name"])
	}

	if titleTag.WidgetParsed["type"] != "input" {
		t.Errorf("Expected widget type 'input', got '%s'", titleTag.WidgetParsed["type"])
	}

	t.Logf("Parsed %d fields successfully", len(tags))
	for i, tag := range tags {
		t.Logf("Field %d: %+v", i, tag)
	}
}

func TestConvertTagsToField(t *testing.T) {
	// 测试将标签转换为Field结构体
	tags := &FieldTags{
		Json:         "title",
		Widget:       "name:工单标题;type:input;desc:工单标题的详细说明;default:默认标题",
		Search:       "like",
		Validate:     "required,min=2,max=200",
		Permission:   "",
		Data:         "",
		WidgetParsed: make(map[string]string),
		DataParsed:   make(map[string]string),
	}

	// 解析widget标签
	parseTagValue(tags.Widget, tags.WidgetParsed)

	field := ConvertTagsToField(tags)

	if field.Code != "title" {
		t.Errorf("Expected field code 'title', got '%s'", field.Code)
	}

	if field.Desc != "工单标题的详细说明" {
		t.Errorf("Expected field desc '工单标题的详细说明', got '%s'", field.Desc)
	}

	if field.Widget.Type != "input" {
		t.Errorf("Expected widget type 'input', got '%s'", field.Widget.Type)
	}

	// 注意：DefaultValue 字段已删除，默认值现在在 widget.Config 中处理
	// 可以通过检查 widget.Config 来验证默认值是否正确设置

	if field.Search != "like" {
		t.Errorf("Expected search 'like', got '%s'", field.Search)
	}

	if field.Validation != "required,min=2,max=200" {
		t.Errorf("Expected validation 'required,min=2,max=200', got '%s'", field.Validation)
	}

	t.Logf("Converted field: %+v", field)
}

func TestDecodeTable(t *testing.T) {
	// 测试DecodeTable功能
	req := &TestSearchReq{}
	tableModel := &TestCrmTicket{}

	requestFields, responseFields, err := DecodeTable(req, tableModel)
	if err != nil {
		t.Fatalf("DecodeTable failed: %v", err)
	}

	t.Logf("Request fields: %d", len(requestFields))
	for i, field := range requestFields {
		t.Logf("Request Field %d: %+v", i, field)
	}

	t.Logf("Response fields: %d", len(responseFields))
	for i, field := range responseFields {
		t.Logf("Response Field %d: %+v", i, field)
	}

	// 检查是否有response字段
	if len(responseFields) == 0 {
		t.Fatal("Expected at least one response field, got none")
	}

	// 检查是否有request字段
	if len(requestFields) == 0 {
		t.Fatal("Expected at least one request field, got none")
	}
}

func TestDebugDefaultInWidget(t *testing.T) {
	// 测试包含default的新结构体
	testStruct := &TestStructWithDefault{}

	// 解析模型
	tags, err := ParseModel(testStruct)
	if err != nil {
		t.Fatalf("ParseModel failed: %v", err)
	}

	// 找到priority字段
	var priorityTags *FieldTags
	for _, tag := range tags {
		if tag.FieldName == "Priority" {
			priorityTags = tag
			break
		}
	}

	if priorityTags == nil {
		t.Fatal("Priority field not found")
	}

	t.Logf("Priority field tags:")
	t.Logf("  WidgetParsed: %+v", priorityTags.WidgetParsed)

	// 转换为Field
	field := ConvertTagsToField(priorityTags)

	// 测试简化后的Select组件
	if selectConfig, ok := field.Widget.Config.(*Select); ok {
		t.Logf("Select widget config:")
		t.Logf("  Options: %v", selectConfig.Options)
		t.Logf("  Placeholder: %s", selectConfig.Placeholder)
		t.Logf("  Default: %s", selectConfig.Default)
	} else {
		t.Logf("Widget config type: %T, value: %v", field.Widget.Config, field.Widget.Config)
	}
}

func TestMVPWidgets(t *testing.T) {
	// 测试MVP简化后的widget组件
	t.Run("User Widget", func(t *testing.T) {
		userParsed := map[string]string{
			"name": "创建用户",
			"type": "user",
		}

		userWidget := newUser(userParsed)
		config := userWidget.Config()

		t.Logf("User widget: %T, %+v", config, config)

		// User组件现在为空结构体
		if _, ok := config.(*User); !ok {
			t.Errorf("Expected *User type, got %T", config)
		}
	})

	t.Run("ID Widget", func(t *testing.T) {
		idParsed := map[string]string{
			"name": "ID",
			"type": "ID",
		}

		idWidget := newID(idParsed)
		config := idWidget.Config()

		t.Logf("ID widget: %T, %+v", config, config)

		// ID组件现在为空结构体
		if _, ok := config.(*ID); !ok {
			t.Errorf("Expected *ID type, got %T", config)
		}
	})

	t.Run("Switch Widget", func(t *testing.T) {
		switchParsed := map[string]string{
			"name": "开关",
			"type": "switch",
		}

		switchWidget := newSwitch(switchParsed)
		config := switchWidget.Config()

		t.Logf("Switch widget: %T, %+v", config, config)

		// Switch组件现在为空结构体
		if _, ok := config.(*Switch); !ok {
			t.Errorf("Expected *Switch type, got %T", config)
		}
	})

	t.Run("Timestamp Widget", func(t *testing.T) {
		timestampParsed := map[string]string{
			"name":     "创建时间",
			"type":     "timestamp",
			"format":   "YYYY-MM-DD HH:mm:ss",
			"disabled": "true",
		}

		timestampWidget := newTimestamp(timestampParsed)
		config := timestampWidget.Config()

		t.Logf("Timestamp widget: %T, %+v", config, config)

		// Timestamp组件保留Format和Disabled
		if ts, ok := config.(*Timestamp); ok {
			if ts.Format != "YYYY-MM-DD HH:mm:ss" {
				t.Errorf("Expected format 'YYYY-MM-DD HH:mm:ss', got '%s'", ts.Format)
			}
			if !ts.Disabled {
				t.Errorf("Expected disabled=true, got %v", ts.Disabled)
			}
		} else {
			t.Errorf("Expected *Timestamp type, got %T", config)
		}
	})
}
