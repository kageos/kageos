package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
)

func TestBuildWorkspaceSuccessResultExposesTestInteraction(t *testing.T) {
	got := buildWorkspaceSuccessResult("/liubeiluo/nps", &dto.UpdateAppResp{
		User:       "liubeiluo",
		App:        "nps",
		OldVersion: "v3",
		NewVersion: "v4",
		Warnings:   []string{"metadata sync delayed"},
	})
	if got.Kind != workspaceBuildArtifactKind || got.WorkspacePath != "/liubeiluo/nps" || got.App != "nps" {
		t.Fatalf("unexpected build result: %#v", got)
	}
	if got.Interaction == nil || got.Interaction.Status != "pending_test" || got.Interaction.TargetRoleOnConfirm != WorkspaceRoleQAEngineer {
		t.Fatalf("unexpected build interaction: %#v", got.Interaction)
	}
	if !containsWorkspaceRoleString(got.Interaction.AllowedActions, "start_test") ||
		!containsWorkspaceRoleString(got.Interaction.AllowedActions, "continue_development") {
		t.Fatalf("interaction should expose test handoff actions: %#v", got.Interaction)
	}
	if len(got.Warnings) != 1 || got.Warnings[0] != "metadata sync delayed" {
		t.Fatalf("warnings not copied: %#v", got.Warnings)
	}
}

func TestBuildWorkspaceToolSchemaExposesStructuredResult(t *testing.T) {
	def := (&BuildWorkspaceTool{}).Definition()
	props, ok := def.OutputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("output schema properties missing: %#v", def.OutputSchema)
	}
	data, ok := props["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("output schema should expose data: %#v", props)
	}
	dataProps, ok := data["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("data schema properties missing: %#v", data)
	}
	for _, name := range []string{"kind", "workspace_path", "app", "interaction"} {
		if _, ok := dataProps[name]; !ok {
			t.Fatalf("build_workspace data schema should expose %q", name)
		}
	}
}

func TestWorkspaceBuildErrorHints(t *testing.T) {
	errText := `app startup failed: SDK schema compile failed:
router /auction/auction_item_list.table table schema decode failed: field Status widget tag "options_colors" contains invalid color "default"
failed to validate table request fields: table request field "session_id" conflicts with table model field "session_id"
failed to validate table callbacks: OnSelectFuzzyMap field "item_id" must use select or multiselect widget
field CurrentPrice (current_price): unsupported widget tag "readonly" for widget "float"
field Attachments (attachments): unsupported widget type "file"
field AvgScore (avg_score): number widget requires integer Go type, got float64
field SupplierName (supplier_name): widget "select" requires options or OnSelectFuzzyMap entry
field CreatedBy (created_by): audit field "created_by" hide tag must be "create,update", got ""
namespace/auction/evaluation_object_list.go:62:9: req.Status undefined (type EvaluationObjectListReq has no field or method Status)
namespace/auction/auction_statistics.go:94:34: d.After undefined (type func() time.Time has no field or method After)
namespace/auction/auction_item_list.go:115:14: req.GetPage undefined
namespace/auction/auction_statistics.go:49: undefined: chart.ComboChart
namespace/auction/auction_statistics.go:204:20: undefined: types.EmptyRequest
namespace/auction/publish_item.go:81:21: undefined: app.Time
namespace/auction/nps_statistics.go:81:20: cannot use res (variable of type *NpsStatisticsResp) as chart.Charter value in argument to resp.Chart: *NpsStatisticsResp does not implement chart.Charter (missing method GetChartType)
namespace/auction/auction_bid_list.go:15:6: AuctionBidRecord redeclared in this block`

	enriched := enrichWorkspaceBuildError(errText)
	for _, want := range []string{
		"下一步要求",
		"不确定 SDK schema、widget、callback、审计字段或 API 写法时",
		"options_colors 只支持不带 # 的 6 位十六进制 RRGGBB",
		"widget 的 type 和配置 key 必须来自 SDK 主文档组件速查和运行时白名单",
		"整数用 type:number",
		"Table 列表使用 query.PageSortReq",
		"调整 Table Request 字段后",
		"OnSelectFuzzyMap 的 key 必须对应 schema",
		"select/multiselect 字段必须有选项来源",
		"Table 列表先用 req.PageSortReq.GetOrder/GetOffset/GetLimit",
		"types.Time 做格式化或比较时先调用 Time() 方法",
		"未确认的 SDK API chart.ComboChart",
		"未确认的 SDK API types.EmptyRequest",
		"未确认的 SDK API app.Time",
		"Chart Handler 必须把 SDK chart 包里的具体图表对象传给 resp.Chart",
		"同一个目录里的模型、函数和方法只能定义一次",
		"系统审计字段必须按 SDK 规范写完整 tag",
	} {
		if !strings.Contains(enriched, want) {
			t.Fatalf("expected build error hint %q in:\n%s", want, enriched)
		}
	}
	if got := strings.Count(enriched, "options_colors 只支持"); got != 1 {
		t.Fatalf("expected options_colors hint once, got %d in:\n%s", got, enriched)
	}

	diagnostics := buildWorkspaceDiagnostics(errText, "/liubeiluo/auction")
	for _, want := range []string{
		"schema_validation",
		"audit_field",
		"select_options",
		"widget",
		"table_request",
		"pagination_response",
		"sdk_api_or_go_symbol",
		"chart_response",
		"duplicate_definition",
	} {
		if !containsWorkspaceRoleString(diagnostics.Categories, want) {
			t.Fatalf("expected diagnostics category %q, got %#v", want, diagnostics.Categories)
		}
	}
	if !containsWorkspaceRoleString(diagnostics.Routers, "/auction/auction_item_list.table") {
		t.Fatalf("expected router in diagnostics, got %#v", diagnostics.Routers)
	}
	if !containsWorkspaceRoleString(diagnostics.Files, "namespace/auction/auction_statistics.go:49") {
		t.Fatalf("expected file reference in diagnostics, got %#v", diagnostics.Files)
	}
	if !workspaceBuildDiagnosticsHasFieldIssue(diagnostics, "CreatedBy", "created_by") {
		t.Fatalf("expected CreatedBy field issue, got %#v", diagnostics.FieldIssues)
	}
	if !containsWorkspaceRoleString(diagnostics.SDKSymbols, "chart.ComboChart") ||
		!containsWorkspaceRoleString(diagnostics.SDKSymbols, "types.EmptyRequest") ||
		!containsWorkspaceRoleString(diagnostics.SDKSymbols, "app.Time") {
		t.Fatalf("expected sdk symbols in diagnostics, got %#v", diagnostics.SDKSymbols)
	}
	if !containsWorkspaceRoleString(diagnostics.RequiredDocs, "/system/prompt/sdk/reference/build-validation") ||
		!containsWorkspaceRoleString(diagnostics.RequiredDocs, "/system/prompt/sdk/agent-app-sdk-readme") {
		t.Fatalf("expected required docs, got %#v", diagnostics.RequiredDocs)
	}
	if !strings.Contains(strings.Join(diagnostics.RepairPolicy, "；"), "不要直接整文件重写") {
		t.Fatalf("expected repair policy, got %#v", diagnostics.RepairPolicy)
	}
}

func TestWorkspaceBuildErrorHintsNoop(t *testing.T) {
	const errText = "unexpected build failure"
	got := enrichWorkspaceBuildError(errText)
	for _, want := range []string{
		errText,
		"下一步要求",
		"不要想当然改 tag 或反复整文件重写",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in:\n%s", want, got)
		}
	}
}

func TestWorkspaceBuildErrorMentionsScope(t *testing.T) {
	const errText = "app startup failed: SDK schema compile failed: router /evaluation/evaluation_record_list.table table schema decode failed"
	got := enrichWorkspaceBuildError(errText, "/liubeiluo/test4")
	for _, want := range []string{
		"本次编译范围: /liubeiluo/test4",
		"router /xxx 是该工作空间内相对路由",
		errText,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in:\n%s", want, got)
		}
	}
}

func TestBuildWorkspaceToolReturnsStructuredDataOnLocalError(t *testing.T) {
	got := (&BuildWorkspaceTool{}).Execute(context.Background(), ToolCall{})
	if !got.IsError {
		t.Fatalf("expected local path error, got %#v", got)
	}
	data, ok := got.Data.(buildWorkspaceResultData)
	if !ok {
		t.Fatalf("expected structured build data on error, got %#v", got.Data)
	}
	if data.Kind != workspaceBuildFailureKind || data.Status != "error" || data.BuildDiagnostics == nil {
		t.Fatalf("unexpected build failure data: %#v", data)
	}
	if data.BuildDiagnostics.Status != "error" || !strings.Contains(data.BuildDiagnostics.ErrorSummary, "无法获取当前工作目录") {
		t.Fatalf("unexpected diagnostics: %#v", data.BuildDiagnostics)
	}
}

func workspaceBuildDiagnosticsHasFieldIssue(diagnostics *workspaceBuildDiagnostics, field string, jsonName string) bool {
	if diagnostics == nil {
		return false
	}
	for _, issue := range diagnostics.FieldIssues {
		if issue.Field == field && issue.JSONName == jsonName {
			return true
		}
	}
	return false
}
