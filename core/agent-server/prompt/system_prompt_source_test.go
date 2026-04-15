package prompt

import (
	"strings"
	"testing"
)

func TestGetPromptDocContent_ForSDKDirectoryAndLeafDoc(t *testing.T) {
	sdkName, sdkContent := GetPromptDocContent(nil, "/system/prompt/sdk/agent-app-sdk-readme")
	if strings.TrimSpace(sdkName) == "" {
		t.Fatal("expected sdk readme doc name")
	}
	if !strings.Contains(sdkContent, "SDK 主入口") {
		t.Fatalf("expected sdk readme content, got: %q", sdkContent)
	}

	createProjectName, createProjectContent := GetPromptDocContent(nil, "/system/prompt/workspace/create-project")
	if strings.TrimSpace(createProjectName) == "" {
		t.Fatal("expected create-project directory doc name")
	}
	if !strings.Contains(createProjectContent, "PRD 格式") {
		t.Fatalf("expected create-project directory content to include merged create-project guide, got: %q", createProjectContent)
	}

	modifyProjectName, modifyProjectContent := GetPromptDocContent(nil, "/system/prompt/workspace/modify-project")
	if strings.TrimSpace(modifyProjectName) == "" {
		t.Fatal("expected modify-project directory doc name")
	}
	if !strings.Contains(modifyProjectContent, "修改 PRD") {
		t.Fatalf("expected modify-project directory content to include modify-project guide, got: %q", modifyProjectContent)
	}

	executeName, executeContent := GetPromptDocContent(nil, "/system/prompt/workspace/execute")
	if strings.TrimSpace(executeName) == "" {
		t.Fatal("expected execute directory doc name")
	}
	if !strings.Contains(executeContent, "操作 SOP") {
		t.Fatalf("expected execute directory content to include execute guide, got: %q", executeContent)
	}
}

func TestPromptDocCandidatePaths_PreferSeedActualPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/system/prompt/doc/workspace-env-template", want: "/system/prompt/doc/workspace-env-template.docs"},
		{path: "/system/prompt/mode/dev/config", want: "/system/prompt/mode/dev/config.docs"},
		{path: "/system/prompt/workspace/create-project", want: "/system/prompt/workspace/create-project/index.docs"},
	}

	for _, tt := range tests {
		got := PromptDocCandidatePaths(tt.path)
		if len(got) == 0 {
			t.Fatalf("expected candidate paths for %s", tt.path)
		}
		if got[0] != tt.want {
			t.Fatalf("expected first candidate for %s to be %s, got %v", tt.path, tt.want, got)
		}
	}
}
