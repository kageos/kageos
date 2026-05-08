package prompt

import (
	"strings"
	"testing"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/functionschema"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
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

func timeNowForTest() time.Time {
	return time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
}
