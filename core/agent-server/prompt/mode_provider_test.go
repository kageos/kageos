package prompt

import (
	"strings"
	"testing"
)

func TestModeProviderAppendsAllInOnePromptForWriteModes(t *testing.T) {
	for _, code := range []string{"dev", "agent", "modify"} {
		provider := GetModeProvider(code)
		if provider == nil {
			t.Fatalf("mode provider %q is nil", code)
		}
		got := provider.SystemPrompt(nil)
		for _, want := range []string{
			"### 全部 widget 白名单",
			"## 二十、Agent-App SDK README 全文",
			"# Agent-App SDK 使用说明",
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("mode %q system prompt missing %q", code, want)
			}
		}
	}
}

func TestWriteModesAppendAllInOnePromptEvenWithoutConfigField(t *testing.T) {
	for _, code := range []string{"dev", "agent", "modify"} {
		got := modeSystemPromptAppendFiles(code, nil)
		if len(got) != 1 || got[0] != allInOneSystemPromptPath {
			t.Fatalf("mode %q append files = %#v, want all-in-one fallback", code, got)
		}
	}
	if got := modeSystemPromptAppendFiles("execute", nil); len(got) != 0 {
		t.Fatalf("execute append files = %#v, want none", got)
	}
}

func TestAllInOnePromptSeedDocReadable(t *testing.T) {
	_, content := GetPromptDocContent(nil, allInOneSystemPromptPath)
	for _, want := range []string{
		"### 全部 widget 白名单",
		"## 二十、Agent-App SDK README 全文",
		"# Agent-App SDK 使用说明",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("all-in-one prompt doc missing %q", want)
		}
	}
}

func TestAllInOnePromptDoesNotTeachDuplicateStatusRequestField(t *testing.T) {
	_, content := GetPromptDocContent(nil, allInOneSystemPromptPath)
	for _, want := range []string{
		`json:"status_filter"`,
		"Request 字段 json/form code 不能和 Model 任意字段重复",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("all-in-one prompt missing %q", want)
		}
	}
	if strings.Contains(content, `Status   string `+"`"+`json:"status" form:"status"`) ||
		strings.Contains(content, `Status                    string `+"`"+`json:"status" form:"status"`) {
		t.Fatalf("all-in-one prompt still contains duplicate status Request example")
	}
}
