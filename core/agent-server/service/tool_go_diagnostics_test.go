package service

import (
	"strings"
	"testing"
)

func TestAppendGoFileDiagnosticsAddsNonBlockingIssues(t *testing.T) {
	msg := appendGoFileDiagnostics("已落盘: bad.go。", "/u/app/nps", "bad.go", `package nps

type NpsRecord struct {
	Attachment string `+"`json:\"attachment\" widget:\"name:附件;type:file;readonly:true\"`"+`
}
`)
	for _, want := range []string{"已落盘", "自动代码诊断", "非阻断问题", "widget_type", "widget_tag"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected %q in %q", want, msg)
		}
	}
}

func TestAppendGoFileDiagnosticsCleanFileReportsClean(t *testing.T) {
	msg := appendGoFileDiagnostics("已落盘: ok.go。", "/u/app/nps", "ok.go", `package nps

type NpsRecord struct {
	Title string `+"`json:\"title\" widget:\"name:标题;type:input\"`"+`
}
`)
	if !strings.Contains(msg, "未发现当前文件") {
		t.Fatalf("expected clean diagnostics in %q", msg)
	}
}

func TestAppendGoFileDiagnosticsSkipsNonGoFile(t *testing.T) {
	msg := appendGoFileDiagnostics("done", "/u/app/nps", "note.md", "bad")
	if msg != "done" {
		t.Fatalf("expected unchanged message, got %q", msg)
	}
}

func TestAppendGoFileDiagnosticsDoesNotRunCrossFileOnSelectFuzzyCheck(t *testing.T) {
	msg := appendGoFileDiagnostics("已落盘: nps_record_list.go。", "/u/app/nps", "nps_record_list.go", `package nps

import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"

var T = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"questionnaire_id": nil,
		},
	},
}
`)
	if strings.Contains(msg, "OnSelectFuzzyMap key") {
		t.Fatalf("file-local diagnostics should not report cross-file OnSelectFuzzyMap warning: %q", msg)
	}
}

func TestAppendGoFileDiagnosticsWarnsImportShim(t *testing.T) {
	msg := appendGoFileDiagnostics("已落盘: nps_types.go。", "/u/app/nps", "nps_types.go", `package nps

import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"

// 此文件用于导出 types 包，供其他文件使用
var _ = types.Time{}
`)
	for _, want := range []string{"go_file_structure", "import shim", "Go import 是文件级"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected %q in %q", want, msg)
		}
	}
}

func TestAppendGoFileDiagnosticsWarnsRootAgentAppImport(t *testing.T) {
	msg := appendGoFileDiagnostics("已落盘: nps_distribution.go。", "/u/app/nps", "nps_distribution.go", `package nps

import agentapp "github.com/ai-agent-os/ai-agent-os/sdk/agent-app"

var _ = agentapp.ChartTypePie
`)
	for _, want := range []string{"sdk_import", "不要导入 sdk/agent-app 根包"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected %q in %q", want, msg)
		}
	}
}

func TestAppendGoFileDiagnosticsWarnsSelectWithoutOptionsOrCallback(t *testing.T) {
	msg := appendGoFileDiagnostics("已落盘: nps_model.go。", "/u/app/nps", "nps_model.go", `package nps

type NpsRecord struct {
	QuestionnaireID int `+"`json:\"questionnaire_id\" widget:\"name:问卷;type:select\"`"+`
}
`)
	for _, want := range []string{"widget_select", "必须有静态 options", "纯存储外键不要写成 select"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected %q in %q", want, msg)
		}
	}
}
