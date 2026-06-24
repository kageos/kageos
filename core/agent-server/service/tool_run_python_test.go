package service

import (
	"strings"
	"testing"
)

func TestResolvePythonInputFiles(t *testing.T) {
	t.Run("explicit input_files wins", func(t *testing.T) {
		got := resolvePythonInputFiles(
			"kageos/workspace/current.csv",
			"kageos/workspace/attached.csv",
			map[string]interface{}{"bucket": "kageos", "key": "workspace/chat/old.csv"},
		)
		if got != "kageos/workspace/current.csv" {
			t.Fatalf("unexpected input files: %q", got)
		}
	})

	t.Run("attached current message files are used before args fallback", func(t *testing.T) {
		got := resolvePythonInputFiles(
			"",
			"kageos/workspace/attached.csv",
			map[string]interface{}{"refs": "kageos/workspace/history.csv"},
		)
		if got != "kageos/workspace/attached.csv" {
			t.Fatalf("unexpected input files: %q", got)
		}
	})

	t.Run("args refs fallback supports follow up messages", func(t *testing.T) {
		got := resolvePythonInputFiles(
			"",
			"",
			map[string]interface{}{"refs": "kageos/workspace/chat/report.csv"},
		)
		if got != "kageos/workspace/chat/report.csv" {
			t.Fatalf("unexpected input files: %q", got)
		}
	})

	t.Run("args bucket key fallback matches uploaded file references", func(t *testing.T) {
		got := resolvePythonInputFiles(
			"",
			"",
			map[string]interface{}{
				"bucket": "kageos",
				"key":    "/workspace/chat/2026/05/26/190f96cc-4d1b-4529-937f-9753689c3f75.csv",
			},
		)
		want := "kageos/workspace/chat/2026/05/26/190f96cc-4d1b-4529-937f-9753689c3f75.csv"
		if got != want {
			t.Fatalf("unexpected input files: %q, want %q", got, want)
		}
	})
}

func TestNormalizeRunPythonInputFileRefsValue(t *testing.T) {
	got := normalizeRunPythonInputFileRefsValue([]interface{}{
		"kageos/workspace/a.csv",
		" ",
		"kageos/workspace/b.csv",
		123,
	})
	want := "kageos/workspace/a.csv,kageos/workspace/b.csv"
	if got != want {
		t.Fatalf("unexpected refs: %q, want %q", got, want)
	}

	got = normalizeRunPythonInputFileRefsValue(map[string]interface{}{
		"refs": "kageos/workspace/nested.csv",
	})
	want = "kageos/workspace/nested.csv"
	if got != want {
		t.Fatalf("unexpected nested refs: %q, want %q", got, want)
	}
}

func TestRunPythonWorkspaceRoot(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "workspace root", path: "/alice/demo", want: "/alice/demo"},
		{name: "nested function", path: "/alice/demo/tools/export.form", want: "/alice/demo"},
		{name: "trim spaces and slashes", path: " alice/demo/tools/export.form/ ", want: "/alice/demo"},
		{name: "empty", path: "", want: ""},
		{name: "missing app", path: "/alice", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runPythonWorkspaceRoot(tt.path); got != tt.want {
				t.Fatalf("runPythonWorkspaceRoot(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestBuildPythonModelGuidanceForCommonFileMistakes(t *testing.T) {
	guidance := buildPythonModelGuidance(map[string]interface{}{
		"status": "失败",
		"output": `执行错误: list index out of range
Traceback:
IndexError: list index out of range`,
	})
	if !strings.Contains(guidance, "【输入文件】") {
		t.Fatalf("expected input file guidance, got %q", guidance)
	}

	guidance = buildPythonModelGuidance(map[string]interface{}{
		"status": "失败",
		"output": "ValueError: kageos_entry 返回了不支持的字段: error",
	})
	if !strings.Contains(guidance, "【返回协议】") {
		t.Fatalf("expected return protocol guidance, got %q", guidance)
	}

	guidance = buildPythonModelGuidance(map[string]interface{}{
		"status": "失败",
		"output": "RuntimeError: python_code 必须定义函数 kageos_entry(args, output_dir)",
	})
	for _, want := range []string{
		"【入口协议】",
		"def kageos_entry(args, output_dir):",
		"print 只做日志",
		"/system/prompt/case_catalog/form/python_output",
	} {
		if !strings.Contains(guidance, want) {
			t.Fatalf("expected missing entry guidance to contain %q, got %q", want, guidance)
		}
	}

	guidance = buildPythonModelGuidance(map[string]interface{}{
		"status": "失败",
		"output": "requests.exceptions.ConnectionError: Failed to establish a new connection",
	})
	if !strings.Contains(guidance, "【文件读取】") {
		t.Fatalf("expected file reading guidance, got %q", guidance)
	}
}

func TestFileOutputInstructionsAvoidManualDownloadLinks(t *testing.T) {
	runPythonDesc := (&RunPythonTool{}).Definition().Description
	for _, content := range []string{
		runPythonDesc,
		(&RunFormSubmitTool{}).Definition().Description,
		filesInstruction,
	} {
		if !strings.Contains(content, "文件组件") {
			t.Fatalf("expected file component guidance in %q", content)
		}
		if !strings.Contains(content, "不要") || !strings.Contains(content, "下载") {
			t.Fatalf("expected guidance to forbid manual download wording in %q", content)
		}
	}
}
