package prompt

import (
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos-sdk/agent-app/widget"
	"github.com/kageos/kageos/pkg/functionschema"
)

func TestBuildInitGoContentUsesPackageRelativeRouterGroup(t *testing.T) {
	t.Parallel()

	content := BuildInitGoContent("/luobei/demo/ticket_system/order", "订单", "订单目录")
	if !strings.Contains(content, "package order") {
		t.Fatalf("unexpected package declaration: %s", content)
	}
	if !strings.Contains(content, `RouterGroup: "/ticket_system/order"`) {
		t.Fatalf("unexpected router group: %s", content)
	}
	if strings.Contains(content, `RouterGroup: "/luobei/demo/ticket_system/order"`) {
		t.Fatalf("router group should not include user/app prefix: %s", content)
	}
}

func TestBuildWorkspaceEnvDataIncludesFunctionRequestSummary(t *testing.T) {
	t.Parallel()

	data := BuildWorkspaceEnvData(&WorkspaceEnvInput{
		Children: []WorkspaceEnvNode{
			{
				Name:         "读取 PDF 信息",
				Code:         "inspect",
				Type:         "function",
				FullCodePath: "/system/tools/pdf/inspect.form",
				TemplateType: "form",
				Schema: functionschema.NewForm(
					[]*widget.Field{
						{
							Code:       "input_files",
							Name:       "上传 PDF 文件",
							Validation: "required",
							Widget: struct {
								Type   string      `json:"type"`
								Config interface{} `json:"config,omitempty"`
							}{Type: "files"},
						},
					},
					nil,
					nil,
				),
			},
		},
	}, "pdf", "/system/tools/pdf", timeNowForTest())

	if !strings.Contains(data.FunctionsSection, "Schema 摘要") {
		t.Fatalf("expected field summary in functions section: %s", data.FunctionsSection)
	}
	if !strings.Contains(data.FunctionsSection, "input_files") {
		t.Fatalf("expected input_files field in functions section: %s", data.FunctionsSection)
	}
	if !strings.Contains(data.FunctionsSection, "【必填】") {
		t.Fatalf("expected required marker in functions section: %s", data.FunctionsSection)
	}
}

func TestWorkspaceEnvBlockIncludesDirectoryIntentHint(t *testing.T) {
	t.Parallel()

	data := BuildWorkspaceEnvData(&WorkspaceEnvInput{
		DirName:      "投票系统",
		DirCode:      "vote",
		FullCodePath: "/system/x_world/vote",
	}, "vote", "/system/x_world/vote", timeNowForTest())
	got := BuildWorkspaceEnvBlock(data, true, "vote", "/system/x_world/vote")
	for _, want := range []string{
		"### 当前目录语义",
		"选择角色前必须先结合当前目录",
		"使用这个软件完成业务结果",
		"不要先写 PRD 或进入开发",
		"新增或改变软件能力",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("workspace env block should contain %q, got:\n%s", want, got)
		}
	}
}

func TestWorkspaceEnvBlockIncludesScheduledTasksSection(t *testing.T) {
	t.Parallel()

	data := BuildWorkspaceEnvData(&WorkspaceEnvInput{
		DirName:               "黄金盯盘助手",
		DirCode:               "gold_watch",
		FullCodePath:          "/system/democase/gold_watch",
		ScheduledTasksSection: "### 当前目录自动执行摘要\n- id=25；类型=Agent 任务；标题=黄金盯盘日报",
	}, "黄金盯盘助手", "/system/democase/gold_watch", timeNowForTest())
	got := BuildWorkspaceEnvBlock(data, true, "黄金盯盘助手", "/system/democase/gold_watch")
	if !strings.Contains(got, "### 当前目录自动执行摘要") ||
		!strings.Contains(got, "标题=黄金盯盘日报") {
		t.Fatalf("workspace env block should include scheduled task summary, got:\n%s", got)
	}
}

func TestWorkspaceEnvBlockPlacesRunbookBeforeDirectoryIntent(t *testing.T) {
	t.Parallel()

	data := BuildWorkspaceEnvData(&WorkspaceEnvInput{
		DirName:                 "跨境物流跟踪",
		DirCode:                 "logistics_tracking",
		FullCodePath:            "/system/cross_border/logistics_tracking",
		DirectoryRunbookSection: "### 当前目录运行手册\n\n来源：`/system/cross_border/logistics_tracking/runbook.docs`\n\n收到物流通知后先核对订单号。",
		ScheduledTasksSection:   "### 当前目录自动执行摘要\n- 当前目录没有已配置的函数任务或 Agent 任务。",
	}, "跨境物流跟踪", "/system/cross_border/logistics_tracking", timeNowForTest())
	got := BuildWorkspaceEnvBlock(data, true, "跨境物流跟踪", "/system/cross_border/logistics_tracking")

	runbookIdx := strings.Index(got, "### 当前目录运行手册")
	intentIdx := strings.Index(got, "### 当前目录语义")
	scheduledIdx := strings.Index(got, "### 当前目录自动执行摘要")
	if runbookIdx < 0 {
		t.Fatalf("workspace env block should include runbook section, got:\n%s", got)
	}
	if intentIdx < 0 || runbookIdx > intentIdx {
		t.Fatalf("runbook section should be before directory intent section, got:\n%s", got)
	}
	if scheduledIdx < 0 || runbookIdx > scheduledIdx {
		t.Fatalf("runbook section should be before scheduled task summary, got:\n%s", got)
	}
	if !strings.Contains(got, "收到物流通知后先核对订单号。") {
		t.Fatalf("workspace env block should include runbook content, got:\n%s", got)
	}
}

func timeNowForTest() time.Time {
	return time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
}
