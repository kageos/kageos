package widget

import (
	"encoding/json"
	"fmt"
	"testing"
)

// ExampleDecodeTable 完整的DecodeTable使用示例（使用MVP简化后的widget组件）
func ExampleDecodeTable() {
	// 模拟CrmTicketSearchReq结构体
	type CrmTicketSearchReq struct {
		SelfOnly string `json:"self_only" widget:"name:只看我的;type:switch"`
	}

	// 模拟CrmTicket结构体（适配MVP简化后的widget组件）
	type CrmTicket struct {
		ID          int    `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" permission:"read"`
		CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime:milli;column:created_at" widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" permission:"read"`
		UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime:milli;column:updated_at" widget:"name:更新时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" permission:"read"`
		DeletedAt   string `json:"deleted_at" gorm:"column:deleted_at" widget:"-"` // 隐藏字段
		Title       string `json:"title" gorm:"column:title" widget:"name:工单标题;type:input" search:"like" validate:"required,min=2,max=200"`
		Description string `json:"description" gorm:"column:description" widget:"name:问题描述;type:text_area" validate:"required,min=10"`
		Priority    string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;default:中" validate:"required,oneof=低 中 高"`
		Status      string `json:"status" gorm:"column:status" widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;default:待处理" validate:"required,oneof=待处理 处理中 已完成 已关闭"`
		Phone       string `json:"phone" gorm:"column:phone" widget:"name:联系电话;type:input" validate:"required,min=11,max=20"`
		CreateBy    string `json:"create_by" gorm:"column:create_by" widget:"name:创建用户;type:user" permission:"read"`
	}

	// 调用DecodeTable
	request := &CrmTicketSearchReq{}
	tableModel := &CrmTicket{}

	requestFields, responseFields, err := DecodeTable(map[string][]string{}, request, tableModel)
	if err != nil {
		fmt.Printf("DecodeTable error: %v\n", err)
		return
	}

	// 打印结果
	fmt.Printf("=== Request Fields (%d) ===\n", len(requestFields))
	for i, field := range requestFields {
		data, _ := json.MarshalIndent(field, "  ", "  ")
		fmt.Printf("Field %d:\n%s\n", i, string(data))
	}

	fmt.Printf("\n=== Response Fields (%d) ===\n", len(responseFields))
	for i, field := range responseFields {
		data, _ := json.MarshalIndent(field, "  ", "  ")
		fmt.Printf("Field %d:\n%s\n", i, string(data))
	}
}

func TestExampleDecodeTable(t *testing.T) {
	ExampleDecodeTable()
}

// TestMVPWidgetJSON 测试MVP简化后widget组件的JSON序列化结果
func TestMVPWidgetJSON(t *testing.T) {
	t.Run("Empty Widgets JSON", func(t *testing.T) {
		// 测试空结构体组件的JSON序列化
		userWidget := &User{}
		userJSON, _ := json.Marshal(userWidget)
		t.Logf("User widget JSON: %s", string(userJSON))

		idWidget := &ID{}
		idJSON, _ := json.Marshal(idWidget)
		t.Logf("ID widget JSON: %s", string(idJSON))

		switchWidget := &Switch{}
		switchJSON, _ := json.Marshal(switchWidget)
		t.Logf("Switch widget JSON: %s", string(switchJSON))

		// 验证空结构体序列化为空JSON对象
		if string(userJSON) != "{}" {
			t.Errorf("Expected user widget JSON to be '{}', got '%s'", string(userJSON))
		}
	})

	t.Run("Timestamp Widget JSON", func(t *testing.T) {
		// 测试Timestamp组件的JSON序列化
		timestampWidget := &Timestamp{
			Format:   "YYYY-MM-DD HH:mm:ss",
			Disabled: true,
		}
		timestampJSON, _ := json.Marshal(timestampWidget)
		t.Logf("Timestamp widget JSON: %s", string(timestampJSON))

		expected := `{"format":"YYYY-MM-DD HH:mm:ss","disabled":true}`
		if string(timestampJSON) != expected {
			t.Errorf("Expected timestamp JSON '%s', got '%s'", expected, string(timestampJSON))
		}
	})

	t.Run("Select Widget JSON", func(t *testing.T) {
		// 测试Select组件的JSON序列化
		selectWidget := &Select{
			Options:     []string{"低", "中", "高"},
			Placeholder: "请选择优先级",
			Default:     "中",
			Creatable:   false,
		}
		selectJSON, _ := json.Marshal(selectWidget)
		t.Logf("Select widget JSON: %s", string(selectJSON))

		expected := `{"options":["低","中","高"],"placeholder":"请选择优先级","default":"中","creatable":false}`
		if string(selectJSON) != expected {
			t.Errorf("Expected select JSON '%s', got '%s'", expected, string(selectJSON))
		}
	})
}
