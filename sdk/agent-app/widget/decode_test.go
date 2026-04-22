package widget

import (
	"encoding/json"
	"strings"
	"testing"

	apptypes "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// 测试用的嵌套结构体
type OrderItem struct {
	ID       int     `json:"id" widget:"name:商品ID;type:ID"`
	Name     string  `json:"name" widget:"name:商品名称;type:input"`
	Price    float64 `json:"price" widget:"name:价格;type:float"`
	Quantity int     `json:"quantity" widget:"name:数量;type:number"`
}

type OrderDetail struct {
	Address string `json:"address" widget:"name:收货地址;type:input"`
	Phone   string `json:"phone" widget:"name:联系电话;type:input"`
	Note    string `json:"note" widget:"name:备注;type:text_area"`
}

type Order struct {
	ID     int          `json:"id" widget:"name:订单ID;type:ID"`
	Title  string       `json:"title" widget:"name:订单标题;type:input"`
	Status string       `json:"status" widget:"name:订单状态;type:select;options:待发货,已发货,已收货"`
	Items  []OrderItem  `json:"items" widget:"name:订单项;type:table"`  // 明确指定为table
	Detail *OrderDetail `json:"detail" widget:"name:订单详情;type:form"` // 明确指定为form
	Remark string       `json:"remark" widget:"name:备注;type:text_area"`
}

// 测试没有明确指定 type 的结构体
type OrderNoWidget struct {
	ID     int          `json:"id" widget:"name:订单ID;type:ID"`
	Items  []OrderItem  `json:"items" widget:"name:订单项"`   // 没有指定type，不会递归
	Detail *OrderDetail `json:"detail" widget:"name:订单详情"` // 没有指定type，不会递归
}

// 测试多层嵌套
type NestedLevel3 struct {
	Field1 string `json:"field1" widget:"name:字段1;type:input"`
	Field2 string `json:"field2" widget:"name:字段2;type:input"`
}

type NestedLevel2 struct {
	Name   string       `json:"name" widget:"name:名称;type:input"`
	Level3 NestedLevel3 `json:"level3" widget:"name:第三层;type:form"`
}

type NestedLevel1 struct {
	Title  string         `json:"title" widget:"name:标题;type:input"`
	Level2 []NestedLevel2 `json:"level2" widget:"name:第二层;type:table"`
}

type OmitEmptyFieldSample struct {
	OutputFiles string `json:"output_files,omitempty" widget:"name:输出文件;type:files"`
	TraceID     string `json:",omitempty" widget:"name:追踪ID;type:input"`
}

type DateTimeFieldSample struct {
	CreatedAt apptypes.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" search:"gte,lte"`
}

type RenderDefaultFieldSample struct {
	Priority string `json:"priority" widget:"name:优先级;type:select;options:低,中,高;render_default:中"`
}

type VoteOptionItemSample struct {
	Content string `json:"content" widget:"name:选项内容;type:input"`
	Sort    int    `json:"sort" widget:"name:排序;type:number"`
}

type DisplayCreateTableFieldSample struct {
	Options []VoteOptionItemSample `json:"options" widget:"name:投票选项;type:table" display:"scenes:create"`
}

type EmptyDisplayScenesFieldSample struct {
	Title string `json:"title" widget:"name:标题;type:input" display:"scenes:"`
}

func TestDecodeForm(t *testing.T) {
	t.Run("基础Form解析-包含table和form嵌套", func(t *testing.T) {
		order := &Order{}
		requestFields, _, err := DecodeForm(nil, order, nil)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}

		// 打印完整的JSON结构
		jsonData, _ := json.MarshalIndent(requestFields, "", "  ")
		t.Logf("解析结果:\n%s", string(jsonData))

		// 验证基础字段数量（6个字段）
		if len(requestFields) != 6 {
			t.Errorf("期望6个字段，实际得到%d个", len(requestFields))
		}

		// 验证 Items 字段（table类型）
		var itemsField *Field
		for _, field := range requestFields {
			if field.Code == "items" {
				itemsField = field
				break
			}
		}

		if itemsField == nil {
			t.Fatal("未找到items字段")
		}

		if itemsField.Widget.Type != TypeTable {
			t.Errorf("items字段widget类型错误，期望%s，实际%s", TypeTable, itemsField.Widget.Type)
		}

		if itemsField.Data.Type != DataTypeStructs {
			t.Errorf("items字段data类型错误，期望%s，实际%s", DataTypeStructs, itemsField.Data.Type)
		}

		// 验证 Items 的子字段（应该有4个）
		if len(itemsField.Children) != 4 {
			t.Errorf("items应该有4个子字段，实际有%d个", len(itemsField.Children))
		}

		// 验证子字段名称
		childNames := make(map[string]bool)
		for _, child := range itemsField.Children {
			childNames[child.Code] = true
			t.Logf("Items子字段: code=%s, name=%s, type=%s", child.Code, child.Name, child.Widget.Type)
		}

		expectedChildren := []string{"id", "name", "price", "quantity"}
		for _, expected := range expectedChildren {
			if !childNames[expected] {
				t.Errorf("items缺少子字段: %s", expected)
			}
		}

		// 验证 Detail 字段（form类型）
		var detailField *Field
		for _, field := range requestFields {
			if field.Code == "detail" {
				detailField = field
				break
			}
		}

		if detailField == nil {
			t.Fatal("未找到detail字段")
		}

		if detailField.Widget.Type != TypeForm {
			t.Errorf("detail字段widget类型错误，期望%s，实际%s", TypeForm, detailField.Widget.Type)
		}

		if detailField.Data.Type != DataTypeStruct {
			t.Errorf("detail字段data类型错误，期望%s，实际%s", DataTypeStruct, detailField.Data.Type)
		}

		// 验证 Detail 的子字段（应该有3个）
		if len(detailField.Children) != 3 {
			t.Errorf("detail应该有3个子字段，实际有%d个", len(detailField.Children))
		}

		// 验证子字段名称
		detailChildNames := make(map[string]bool)
		for _, child := range detailField.Children {
			detailChildNames[child.Code] = true
			t.Logf("Detail子字段: code=%s, name=%s, type=%s", child.Code, child.Name, child.Widget.Type)
		}

		expectedDetailChildren := []string{"address", "phone", "note"}
		for _, expected := range expectedDetailChildren {
			if !detailChildNames[expected] {
				t.Errorf("detail缺少子字段: %s", expected)
			}
		}
	})

	t.Run("json tag option不会污染字段code", func(t *testing.T) {
		fields, _, err := DecodeForm(nil, &OmitEmptyFieldSample{}, nil)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}

		if len(fields) != 2 {
			t.Fatalf("期望2个字段，实际得到%d个", len(fields))
		}

		if fields[0].Code != "output_files" {
			t.Fatalf("fields[0].Code = %q, want %q", fields[0].Code, "output_files")
		}

		if fields[1].Code != "TraceID" {
			t.Fatalf("fields[1].Code = %q, want %q", fields[1].Code, "TraceID")
		}
	})

	t.Run("datetime字段按字符串协议输出schema", func(t *testing.T) {
		fields, _, err := DecodeForm(nil, &DateTimeFieldSample{}, nil)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if len(fields) != 1 {
			t.Fatalf("期望1个字段，实际得到%d个", len(fields))
		}
		field := fields[0]
		if field.Widget.Type != TypeDatetime {
			t.Fatalf("widget type = %q, want %q", field.Widget.Type, TypeDatetime)
		}
		if field.Data.Type != DataTypeString {
			t.Fatalf("data type = %q, want %q", field.Data.Type, DataTypeString)
		}
		config, ok := field.Widget.Config.(*DateTime)
		if !ok {
			t.Fatalf("config type = %T, want *DateTime", field.Widget.Config)
		}
		if config.Format != "YYYY-MM-DD HH:mm:ss" {
			t.Fatalf("format = %q", config.Format)
		}
	})

	t.Run("render_default只输出新schema", func(t *testing.T) {
		fields, _, err := DecodeForm(nil, &RenderDefaultFieldSample{}, nil)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if len(fields) != 1 {
			t.Fatalf("期望1个字段，实际得到%d个", len(fields))
		}

		priorityConfig, ok := fields[0].Widget.Config.(*Select)
		if !ok {
			t.Fatalf("priority config type = %T, want *Select", fields[0].Widget.Config)
		}
		if priorityConfig.RenderDefault != "中" {
			t.Fatalf("priority render_default = %q", priorityConfig.RenderDefault)
		}
		data, err := json.Marshal(fields)
		if err != nil {
			t.Fatalf("marshal fields: %v", err)
		}
		if string(data) == "" || !strings.Contains(string(data), `"render_default"`) {
			t.Fatalf("schema should contain render_default, got %s", string(data))
		}
		if strings.Contains(string(data), `"default"`) {
			t.Fatalf("schema should not emit legacy default, got %s", string(data))
		}
	})

	t.Run("table字段保留create display scenes", func(t *testing.T) {
		fields, _, err := DecodeForm(nil, &DisplayCreateTableFieldSample{}, nil)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if len(fields) != 1 {
			t.Fatalf("期望1个字段，实际得到%d个", len(fields))
		}
		if fields[0].Display == nil {
			t.Fatal("Display should not be nil")
		}
		if got, want := strings.Join(fields[0].Display.Scenes, ","), "create"; got != want {
			t.Fatalf("display scenes = %q, want %q", got, want)
		}
		if len(fields[0].Children) != 2 {
			t.Fatalf("table children = %d, want 2", len(fields[0].Children))
		}
	})

	t.Run("空display scenes按未配置处理", func(t *testing.T) {
		fields, _, err := DecodeForm(nil, &EmptyDisplayScenesFieldSample{}, nil)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}
		if len(fields) != 1 {
			t.Fatalf("期望1个字段，实际得到%d个", len(fields))
		}
		if fields[0].Display != nil {
			t.Fatalf("Display = %#v, want nil", fields[0].Display)
		}
	})

	t.Run("没有明确指定type-不进行递归", func(t *testing.T) {
		order := &OrderNoWidget{}
		requestFields, _, err := DecodeForm(nil, order, nil)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}

		// 打印完整的JSON结构
		jsonData, _ := json.MarshalIndent(requestFields, "", "  ")
		t.Logf("解析结果:\n%s", string(jsonData))

		// 验证 Items 字段
		var itemsField *Field
		for _, field := range requestFields {
			if field.Code == "items" {
				itemsField = field
				break
			}
		}

		if itemsField == nil {
			t.Fatal("未找到items字段")
		}

		// 没有指定type，不应该有Children
		if len(itemsField.Children) != 0 {
			t.Errorf("items字段没有指定type:table，不应该有Children，但实际有%d个", len(itemsField.Children))
		}

		// 验证 Detail 字段
		var detailField *Field
		for _, field := range requestFields {
			if field.Code == "detail" {
				detailField = field
				break
			}
		}

		if detailField == nil {
			t.Fatal("未找到detail字段")
		}

		// 没有指定type，不应该有Children
		if len(detailField.Children) != 0 {
			t.Errorf("detail字段没有指定type:form，不应该有Children，但实际有%d个", len(detailField.Children))
		}
	})

	t.Run("多层嵌套解析", func(t *testing.T) {
		nested := &NestedLevel1{}
		requestFields, _, err := DecodeForm(nil, nested, nil)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}

		// 打印完整的JSON结构
		jsonData, _ := json.MarshalIndent(requestFields, "", "  ")
		t.Logf("多层嵌套解析结果:\n%s", string(jsonData))

		// 查找 level2 字段
		var level2Field *Field
		for _, field := range requestFields {
			if field.Code == "level2" {
				level2Field = field
				break
			}
		}

		if level2Field == nil {
			t.Fatal("未找到level2字段")
		}

		// 验证第二层
		if level2Field.Widget.Type != TypeTable {
			t.Errorf("level2应该是table类型，实际是%s", level2Field.Widget.Type)
		}

		if len(level2Field.Children) != 2 {
			t.Errorf("level2应该有2个子字段，实际有%d个", len(level2Field.Children))
		}

		// 查找第三层
		var level3Field *Field
		for _, child := range level2Field.Children {
			if child.Code == "level3" {
				level3Field = child
				break
			}
		}

		if level3Field == nil {
			t.Fatal("未找到level3字段")
		}

		// 验证第三层
		if level3Field.Widget.Type != TypeForm {
			t.Errorf("level3应该是form类型，实际是%s", level3Field.Widget.Type)
		}

		if len(level3Field.Children) != 2 {
			t.Errorf("level3应该有2个子字段，实际有%d个", len(level3Field.Children))
		}

		// 验证第三层的字段
		level3ChildNames := make(map[string]bool)
		for _, child := range level3Field.Children {
			level3ChildNames[child.Code] = true
			t.Logf("Level3子字段: code=%s, name=%s, type=%s", child.Code, child.Name, child.Widget.Type)
		}

		if !level3ChildNames["field1"] || !level3ChildNames["field2"] {
			t.Error("level3缺少子字段")
		}
	})
}

func TestDecodeTable(t *testing.T) {
	t.Run("Table解析", func(t *testing.T) {
		// 这里测试的是表格的列定义
		tableModel := &OrderItem{}
		_, responseFields, err := DecodeTable(map[string][]string{}, nil, tableModel)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}

		// 打印完整的JSON结构
		jsonData, _ := json.MarshalIndent(responseFields, "", "  ")
		t.Logf("Table列定义:\n%s", string(jsonData))

		// 验证字段数量（4个字段）
		if len(responseFields) != 4 {
			t.Errorf("期望4个字段，实际得到%d个", len(responseFields))
		}

		// 验证字段名称
		fieldNames := make(map[string]bool)
		for _, field := range responseFields {
			fieldNames[field.Code] = true
			t.Logf("Table字段: code=%s, name=%s, type=%s, dataType=%s",
				field.Code, field.Name, field.Widget.Type, field.Data.Type)
		}

		expectedFields := []string{"id", "name", "price", "quantity"}
		for _, expected := range expectedFields {
			if !fieldNames[expected] {
				t.Errorf("缺少字段: %s", expected)
			}
		}
	})
}

// 测试收银台结构体（用于验证 callback 标签在嵌套结构体中的解析）
type CashierProductQuantity struct {
	ProductID int `json:"product_id" widget:"name:商品;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Quantity  int `json:"quantity" widget:"name:数量;type:number" validate:"required,min=1"`
}

type CashierDeskReq struct {
	ProductQuantities []CashierProductQuantity `json:"product_quantities" widget:"name:商品清单;type:table" validate:"required,min=1"`
	MemberID          int                      `json:"member_id" widget:"name:会员卡;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Remarks           string                   `json:"remarks" widget:"name:备注;type:text_area"`
}

func TestCashierDeskReqCallbackParsing(t *testing.T) {
	t.Run("测试嵌套结构体中 callback 标签的解析", func(t *testing.T) {
		req := &CashierDeskReq{}
		result, err := ParseModelWithType(req)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}

		// 转换为 Field 结构
		var fields []*Field
		for _, tag := range result.Tags {
			field := ConvertTagsToField(tag)
			fields = append(fields, field)
		}

		// 打印 JSON 查看结果
		jsonData, _ := json.MarshalIndent(fields, "", "  ")
		t.Logf("解析结果:\n%s", string(jsonData))

		// 1. 验证 product_quantities 字段（table 类型）
		var productQuantitiesField *Field
		for _, field := range fields {
			if field.Code == "product_quantities" {
				productQuantitiesField = field
				break
			}
		}

		if productQuantitiesField == nil {
			t.Fatal("未找到 product_quantities 字段")
		}

		if productQuantitiesField.Widget.Type != "table" {
			t.Errorf("期望 widget.type 为 table，实际为 %s", productQuantitiesField.Widget.Type)
		}

		if len(productQuantitiesField.Children) == 0 {
			t.Fatal("product_quantities 字段应该包含子字段")
		}

		// 2. 验证嵌套的 product_id 字段的 callback
		var productIDField *Field
		for _, child := range productQuantitiesField.Children {
			if child.Code == "product_id" {
				productIDField = child
				break
			}
		}

		if productIDField == nil {
			t.Fatal("未找到嵌套的 product_id 字段")
		}

		// 验证 callback 标签是否被正确解析
		if len(productIDField.Callbacks) == 0 {
			t.Error("product_id 字段的 callback 标签未被解析")
		} else {
			if productIDField.Callbacks[0] != "OnSelectFuzzy" {
				t.Errorf("期望 callback 为 OnSelectFuzzy，实际为 %v", productIDField.Callbacks)
			} else {
				t.Logf("✅ product_id 字段的 callback 标签解析成功: %v", productIDField.Callbacks)
			}
		}

		// 3. 验证顶层的 member_id 字段的 callback
		var memberIDField *Field
		for _, field := range fields {
			if field.Code == "member_id" {
				memberIDField = field
				break
			}
		}

		if memberIDField == nil {
			t.Fatal("未找到 member_id 字段")
		}

		if len(memberIDField.Callbacks) == 0 {
			t.Error("member_id 字段的 callback 标签未被解析")
		} else {
			if memberIDField.Callbacks[0] != "OnSelectFuzzy" {
				t.Errorf("期望 callback 为 OnSelectFuzzy，实际为 %v", memberIDField.Callbacks)
			} else {
				t.Logf("✅ member_id 字段的 callback 标签解析成功: %v", memberIDField.Callbacks)
			}
		}

		// 4. 验证 quantity 字段没有 callback
		var quantityField *Field
		for _, child := range productQuantitiesField.Children {
			if child.Code == "quantity" {
				quantityField = child
				break
			}
		}

		if quantityField == nil {
			t.Fatal("未找到 quantity 字段")
		}

		if len(quantityField.Callbacks) > 0 {
			t.Errorf("quantity 字段不应该有 callback，但实际有: %v", quantityField.Callbacks)
		} else {
			t.Logf("✅ quantity 字段没有 callback（符合预期）")
		}
	})
}

func TestParseModelWithType(t *testing.T) {
	t.Run("解析基础模型", func(t *testing.T) {
		order := &Order{}
		result, err := ParseModelWithType(order)
		if err != nil {
			t.Fatalf("解析失败: %v", err)
		}

		t.Logf("解析到%d个字段", len(result.Tags))
		t.Logf("结构体类型: %s", result.Type.String())

		// 打印完整的 JSON 结构
		jsonData, _ := json.MarshalIndent(result.Tags, "", "  ")
		t.Logf("完整的 FieldTags JSON 结构:\n%s", string(jsonData))

		// 转换为 Field 结构并打印
		var fields []*Field
		for _, tag := range result.Tags {
			field := ConvertTagsToField(tag)
			fields = append(fields, field)
		}

		fieldsJSON, _ := json.MarshalIndent(fields, "", "  ")
		t.Logf("\n\n转换后的 Field JSON 结构:\n%s", string(fieldsJSON))

		for _, tag := range result.Tags {
			t.Logf("字段: %s, json=%s, widget=%s, type=%s, children=%d",
				tag.FieldName, tag.Json, tag.Widget, tag.Type.String(), len(tag.Children))

			// 如果有子节点，打印子节点信息
			if len(tag.Children) > 0 {
				for _, child := range tag.Children {
					t.Logf("  -> 子字段: %s, json=%s, widget=%s",
						child.FieldName, child.Json, child.Widget)
				}
			}
		}
	})
}
