package service

import (
	"context"
	"strings"
	"testing"
)

func TestNormalizeAndValidateWorkspaceRoleHandoffPacketRepairsUnsafeFields(t *testing.T) {
	fullPRD := `{"kind":"agent_app_prd","project":{"name":"投票系统","code":"vote"},"tables":[{"name":"投票主题"}],"forms":[{"name":"提交投票"}],"charts":[{"name":"结果统计"}]}`
	packet := &workspaceRoleHandoffPacket{
		TargetRole:         WorkspaceRoleQAEngineer,
		ExecuteDirectory:   "/system/x_world",
		TargetAppDirectory: "/system/x_world/vote",
		TaskContext: []string{
			"build 已通过，进入测试",
			fullPRD,
		},
		KeyInformation: []string{
			fullPRD,
			"重点验证投票提交",
		},
		References: []string{
			"当前目录",
			"/system/x_world/vote",
			fullPRD,
		},
		BuildDiagnostics: &workspaceBuildDiagnostics{
			Status:       "error",
			Categories:   []string{"schema_validation"},
			RequiredDocs: []string{"/system/prompt/sdk/reference/build-validation"},
		},
	}

	normalizeAndValidateWorkspaceRoleHandoffPacket(packet)

	if packet.Validation.Status != "warning" {
		t.Fatalf("expected warning validation after repairs, got %#v", packet.Validation)
	}
	if packet.Version != workspaceRoleHandoffPacketVersion {
		t.Fatalf("version should be repaired, got %q", packet.Version)
	}
	if packet.ExecuteDirectory != "/system/x_world/vote" {
		t.Fatalf("QA execute_directory should be narrowed to app dir, got %q", packet.ExecuteDirectory)
	}
	if packet.BuildDiagnostics != nil {
		t.Fatalf("non-build-engineer packet should not keep build diagnostics: %#v", packet.BuildDiagnostics)
	}
	if containsWorkspaceRoleString(packet.References, "当前目录") || strings.Contains(strings.Join(packet.References, "；"), `"project"`) {
		t.Fatalf("references should remove placeholder/full JSON entries: %#v", packet.References)
	}
	if strings.Contains(strings.Join(packet.TaskContext, "；"), `"project"`) ||
		strings.Contains(strings.Join(packet.KeyInformation, "；"), `"project"`) {
		t.Fatalf("task/key info should not inline full artifact JSON: %#v %#v", packet.TaskContext, packet.KeyInformation)
	}
	for _, want := range []string{"execute_directory", "build_diagnostics", "疑似完整产物 JSON"} {
		if !strings.Contains(strings.Join(append(packet.Validation.Warnings, packet.Validation.Repaired...), "；"), want) {
			t.Fatalf("validation should mention %q, got %#v", want, packet.Validation)
		}
	}
}

func TestChangeRoleRejectsPlaceholderExecuteDirectoryAfterNormalization(t *testing.T) {
	res := (&ChangeRoleTool{}).Execute(context.Background(), ToolCall{
		FullCodePath: "/system/x_world/vote",
		Args: map[string]interface{}{
			"target_role":       WorkspaceRoleQAEngineer,
			"execute_directory": "当前目录",
			"task_context":      []interface{}{"build 已通过，进入测试"},
		},
	})
	if !res.IsError {
		t.Fatalf("expected placeholder execute directory to fail, got %#v", res)
	}
	if !strings.Contains(res.Content, "交接协议校验失败") || !strings.Contains(res.Content, "execute_directory") {
		t.Fatalf("error should mention handoff packet validation and execute_directory, got %q", res.Content)
	}
}

func TestBuildFailureHandoffPacketRequiresDiagnostics(t *testing.T) {
	packet := &workspaceRoleHandoffPacket{
		Version:          workspaceRoleHandoffPacketVersion,
		TargetRole:       WorkspaceRoleBuildEngineer,
		ArtifactKind:     workspaceBuildFailureKind,
		ExecuteDirectory: "/system/x_world/vote",
		TaskContext:      []string{"build_workspace 失败，进入构建修复"},
	}

	normalizeAndValidateWorkspaceRoleHandoffPacket(packet)

	if packet.Validation.Status != "error" {
		t.Fatalf("build failure packet without diagnostics should be invalid, got %#v", packet.Validation)
	}
	if !strings.Contains(strings.Join(packet.Validation.Errors, "；"), "build_diagnostics") {
		t.Fatalf("validation errors should mention build_diagnostics, got %#v", packet.Validation.Errors)
	}
}
