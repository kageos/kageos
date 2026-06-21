package functionschema

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kageos/kageos-sdk/agent-app/widget"
)

func TestTableSceneSelectors(t *testing.T) {
	schema := NewTable(nil, []*widget.Field{
		testField("id", widget.TypeID, SceneCreate, SceneUpdate),
		testField("title", widget.TypeInput),
		testField("remaining_time", "", SceneCreate, SceneUpdate),
		testField("create_note", "", SceneList, SceneUpdate),
	}, []string{"OnTableAddRow", "OnTableAddRow", ""})

	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}

	listFields := TableListFields(raw)
	if got, want := fieldCodes(listFields), []string{"id", "title", "remaining_time"}; !sameStrings(got, want) {
		t.Fatalf("list fields = %v, want %v", got, want)
	}

	createFields := TableCreateFields(raw)
	if got, want := fieldCodes(createFields), []string{"title", "create_note"}; !sameStrings(got, want) {
		t.Fatalf("create fields = %v, want %v", got, want)
	}

	updateFields := TableUpdateFields(raw)
	if got, want := fieldCodes(updateFields), []string{"title"}; !sameStrings(got, want) {
		t.Fatalf("update fields = %v, want %v", got, want)
	}

	if got, want := Callbacks(raw), []string{"OnTableAddRow"}; !sameStrings(got, want) {
		t.Fatalf("callbacks = %v, want %v", got, want)
	}
}

func TestValidateRejectsInvalidHideScenes(t *testing.T) {
	tests := []struct {
		name   string
		scenes []string
	}{
		{name: "empty", scenes: []string{}},
		{name: "unknown", scenes: []string{"detail"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := NewTable(nil, []*widget.Field{{
				Code: "title",
				Hide: &widget.FieldHide{
					Scenes: tt.scenes,
				},
			}}, nil)

			if err := Validate(schema); err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
		})
	}
}

func TestValidateRejectsUnsupportedWidgetType(t *testing.T) {
	schema := NewForm([]*widget.Field{testField("record_date", "date")}, nil, nil)

	if err := Validate(schema); err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}

func TestValidateAggregatesFieldErrors(t *testing.T) {
	schema := NewForm([]*widget.Field{
		testField("record_date", "date"),
		testField("title", widget.TypeInput, "detail"),
	}, nil, nil)

	err := Validate(schema)
	if err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
	for _, want := range []string{
		`field "record_date" has unsupported widget type: date`,
		`field "title" has unsupported hide scene: detail`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Validate() error = %v, want substring %q", err, want)
		}
	}
}

func TestListSceneForContainerWidgetIsIgnored(t *testing.T) {
	schema := NewTable(nil, []*widget.Field{{
		Code: "options",
		Hide: &widget.FieldHide{
			Scenes: []string{SceneList},
		},
	}}, nil)
	schema.Table.Fields[0].Widget.Type = widget.TypeTable

	if err := Validate(schema); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
	if got := TableListFields(mustMarshalSchema(t, schema)); len(got) != 0 {
		t.Fatalf("TableListFields length = %d, want 0", len(got))
	}
}

func mustMarshalSchema(t *testing.T, schema *FunctionSchema) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	return raw
}

func fieldCodes(fields []*widget.Field) []string {
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		if field != nil {
			result = append(result, field.Code)
		}
	}
	return result
}

func testField(code string, widgetType string, scenes ...string) *widget.Field {
	field := &widget.Field{Code: code}
	field.Widget.Type = widgetType
	if scenes != nil {
		field.Hide = &widget.FieldHide{Scenes: scenes}
	}
	return field
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
