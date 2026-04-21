package widget

import (
	"encoding/json"
	"fmt"
	"testing"

	apptypes "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// ExampleDecodeTable 完整的DecodeTable使用示例（使用MVP简化后的widget组件）
func ExampleDecodeTable() {
	// 模拟CrmTicketSearchReq结构体
	type CrmTicketSearchReq struct {
		SelfOnly string `json:"self_only" widget:"name:只看我的;type:switch"`
	}

	// 模拟CrmTicket结构体（适配MVP简化后的widget组件）
	// 含截止时间、剩余时间：剩余时间为只读计算字段，在 GORM AfterFind 中根据当前时间与截止时间计算，仅用于列表/详情展示。
	type CrmTicket struct {
		ID            int           `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" display:"scenes:list"`                                                        // 前端仅在列表展示，不进入新增/编辑表单。
		CreatedAt     apptypes.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" display:"scenes:list"` // 前端仅在列表展示，不进入新增/编辑表单。
		UpdatedAt     apptypes.Time `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" display:"scenes:list"` // 前端仅在列表展示，不进入新增/编辑表单。
		DeletedAt     string        `json:"deleted_at" gorm:"column:deleted_at" widget:"-"`                                                                                                     // 隐藏字段
		Title         string        `json:"title" gorm:"column:title" widget:"name:工单标题;type:input" search:"like" validate:"required,min=2,max=200"`
		Description   string        `json:"description" gorm:"column:description" widget:"name:问题描述;type:text_area" validate:"required,min=10"`
		Priority      string        `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;render_default:中" validate:"required,oneof=低 中 高"`
		Status        string        `json:"status" gorm:"column:status" widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;render_default:待处理" validate:"required,oneof=待处理 处理中 已完成 已关闭"`
		Phone         string        `json:"phone" gorm:"column:phone" widget:"name:联系电话;type:input" validate:"required,min=11,max=20"`
		CreateBy      string        `json:"create_by" gorm:"column:create_by" widget:"name:创建用户;type:user" display:"scenes:list"` // 前端仅在列表展示，不进入新增/编辑表单。
		Deadline      apptypes.Time `json:"deadline" gorm:"column:deadline;type:datetime" widget:"name:截止时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
		RemainingTime string        `json:"remaining_time" gorm:"-" widget:"name:剩余时间;type:input" display:"scenes:list"` // 前端仅在列表展示，不进入新增/编辑表单；AfterFind 计算
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

	t.Run("DateTime Widget JSON", func(t *testing.T) {
		datetimeWidget := &DateTime{
			Format:   "YYYY-MM-DD HH:mm:ss",
			Disabled: true,
		}
		datetimeJSON, _ := json.Marshal(datetimeWidget)
		t.Logf("DateTime widget JSON: %s", string(datetimeJSON))

		expected := `{"format":"YYYY-MM-DD HH:mm:ss","disabled":true}`
		if string(datetimeJSON) != expected {
			t.Errorf("Expected datetime JSON '%s', got '%s'", expected, string(datetimeJSON))
		}
	})

	t.Run("Select Widget JSON", func(t *testing.T) {
		// 测试Select组件的JSON序列化
		selectWidget := &Select{
			Options:       []string{"低", "中", "高"},
			Placeholder:   "请选择优先级",
			RenderDefault: "中",
			Creatable:     false,
		}
		selectJSON, _ := json.Marshal(selectWidget)
		t.Logf("Select widget JSON: %s", string(selectJSON))

		expected := `{"options":["低","中","高"],"placeholder":"请选择优先级","render_default":"中","creatable":false}`
		if string(selectJSON) != expected {
			t.Errorf("Expected select JSON '%s', got '%s'", expected, string(selectJSON))
		}
	})
}
