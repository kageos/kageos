package app

import (
	"strings"
	"testing"

	"github.com/kageos/kageos/sdk/agent-app/response"
)

type compileTestReq struct {
	Title string `json:"title" widget:"name:标题;type:input"`
}

type compileTestTableModel struct {
	ID    int    `json:"id" widget:"name:ID;type:ID" hide:"create,update"`
	Title string `json:"title" widget:"name:标题;type:input"`
}

func (compileTestTableModel) TableName() string {
	return "compile_test_table"
}

type compileTestOtherTableModel struct {
	Name string `json:"name" widget:"name:名称;type:input"`
}

type compileTestTableReq struct {
	Keyword string `json:"keyword" widget:"name:关键词;type:input"`
}

type compileTestUnsupportedWidgetReq struct {
	RecordDate string `json:"record_date" widget:"name:日期;type:date"`
}

type compileTestAggregateReq struct {
	InputFiles []string `json:"input_files" widget:"name:输入文件;type:files;max_count:-1"`
}

type compileTestAggregateResp struct {
	Status int `json:"status" widget:"name:状态;type:select"`
}

func TestCompileAndValidateRejectsRouteSuffixMismatch(t *testing.T) {
	t.Parallel()

	testApp := newCompileTestApp("/demo/create.table", &FormTemplate{
		BaseConfig: BaseConfig{
			Request: compileTestReq{},
		},
	})

	err := testApp.CompileAndValidate()
	if err == nil {
		t.Fatal("CompileAndValidate() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "must end with .form") {
		t.Fatalf("CompileAndValidate() error = %v, want route suffix error", err)
	}
}

func TestCompileAndValidateAggregatesRouteAndSchemaErrors(t *testing.T) {
	t.Parallel()

	testApp := newCompileTestApp("/demo/create.table", &FormTemplate{
		BaseConfig: BaseConfig{
			Request:  compileTestAggregateReq{},
			Response: compileTestAggregateResp{},
		},
	})

	err := testApp.CompileAndValidate()
	if err == nil {
		t.Fatal("CompileAndValidate() error = nil, want error")
	}
	for _, want := range []string{
		"must end with .form",
		"files widget uses comma-separated file refs and requires string Go type",
		`widget tag "max_count" must be >= 0`,
		`widget "select" requires options or OnSelectFuzzyMap entry`,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("CompileAndValidate() error = %v, want substring %q", err, want)
		}
	}
}

func TestCompileAndValidateRejectsUnsupportedWidgetType(t *testing.T) {
	t.Parallel()

	testApp := newCompileTestApp("/demo/create.form", &FormTemplate{
		BaseConfig: BaseConfig{
			Request: compileTestUnsupportedWidgetReq{},
		},
	})

	err := testApp.CompileAndValidate()
	if err == nil {
		t.Fatal("CompileAndValidate() error = nil, want error")
	}
	if !strings.Contains(err.Error(), `unsupported widget type "date"`) {
		t.Fatalf("CompileAndValidate() error = %v, want unsupported widget type error", err)
	}
}

func TestCompileAndValidateAcceptsValidForm(t *testing.T) {
	t.Parallel()

	testApp := newCompileTestApp("/demo/create.form", &FormTemplate{
		BaseConfig: BaseConfig{
			Request: compileTestReq{},
		},
	})

	if err := testApp.CompileAndValidate(); err != nil {
		t.Fatalf("CompileAndValidate() error = %v, want nil", err)
	}
}

func TestTableTemplateFallsBackToFirstCreateTableWhenAutoCrudMissing(t *testing.T) {
	t.Parallel()

	testApp := newCompileTestApp("/demo/list.table", &TableTemplate{
		BaseConfig: BaseConfig{
			Request:      compileTestTableReq{},
			CreateTables: []interface{}{nil, &compileTestTableModel{}},
		},
	})

	if err := testApp.CompileAndValidate(); err != nil {
		t.Fatalf("CompileAndValidate() error = %v, want nil", err)
	}

	apis, _, err := testApp.getApis()
	if err != nil {
		t.Fatalf("getApis() error = %v, want nil", err)
	}
	if len(apis) != 1 || apis[0].Schema == nil || apis[0].Schema.Table == nil {
		t.Fatalf("getApis() schema = %#v, want one table schema", apis)
	}
	if len(apis[0].Schema.Table.Fields) == 0 || apis[0].Schema.Table.Fields[0].Code != "id" {
		t.Fatalf("table fields = %#v, want fields from compileTestTableModel fallback", apis[0].Schema.Table.Fields)
	}
}

func TestTableTemplateExplicitAutoCrudTableWinsOverCreateTablesFallback(t *testing.T) {
	t.Parallel()

	template := &TableTemplate{
		BaseConfig: BaseConfig{
			CreateTables: []interface{}{&compileTestTableModel{}},
		},
		AutoCrudTable: &compileTestOtherTableModel{},
	}

	if got := template.EffectiveAutoCrudTable(); got != template.AutoCrudTable {
		t.Fatalf("EffectiveAutoCrudTable() = %#v, want explicit AutoCrudTable", got)
	}
}

func newCompileTestApp(route string, template Templater) *App {
	return &App{
		routerInfo: map[string]*routerInfo{
			routerKey(route): {
				Router:     route,
				Method:     "POST",
				HandleFunc: func(ctx *Context, resp response.Response) error { return nil },
				Template:   template,
			},
		},
	}
}
