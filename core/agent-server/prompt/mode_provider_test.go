package prompt

import (
	"strings"
	"testing"
)

func TestModeProviderOnlyRegistersDev(t *testing.T) {
	if provider := GetModeProvider("dev"); provider == nil {
		t.Fatal("dev mode provider is nil")
	}
	for _, code := range []string{"qa", "modify", "execute", "agent"} {
		if provider := GetModeProvider(code); provider != nil {
			t.Fatalf("mode provider %q should be unavailable", code)
		}
	}
}

func TestDevModePromptIsShortAndDoesNotAppendAllInOne(t *testing.T) {
	provider := GetModeProvider("dev")
	if provider == nil {
		t.Fatal("dev mode provider is nil")
	}
	got := provider.SystemPrompt(nil)
	if strings.Contains(got, "{{WORKSPACE_ROLE_ROUTING}}") {
		t.Fatalf("dev prompt should expand role routing marker:\n%s", got)
	}
	if !strings.Contains(got, "### `product_manager` 产品经理") ||
		!strings.Contains(got, "### `app_operator` 应用操作员") ||
		!strings.Contains(got, "`tables.fields` 是模型字段，`tables.search_fields` 是查询请求字段") {
		t.Fatalf("dev prompt should include generated role routing:\n%s", got)
	}
	if strings.Contains(got, "全家桶") || strings.Contains(got, "## 二十、Agent-App SDK README 全文") {
		t.Fatalf("dev prompt should not append all-in-one prompt:\n%s", got)
	}
	removedTerm := "sk" + "ill"
	if strings.Contains(strings.ToLower(got), removedTerm) || strings.Contains(got, "read_"+removedTerm) || strings.Contains(got, "search_"+removedTerm+"s") {
		t.Fatalf("dev prompt should not mention removed doc tools:\n%s", got)
	}
}

func TestModeSystemPromptAppendFilesUsesOnlyConfig(t *testing.T) {
	if got := modeSystemPromptAppendFiles("dev", nil); len(got) != 0 {
		t.Fatalf("dev append files = %#v, want none", got)
	}
	configured := []string{"/system/prompt/sdk/agent-app-sdk-readme"}
	got := modeSystemPromptAppendFiles("dev", configured)
	if len(got) != 1 || got[0] != configured[0] {
		t.Fatalf("append files = %#v, want configured only", got)
	}
}

func TestDevModeSystemPromptDocExpandsRoleRouting(t *testing.T) {
	_, got := GetPromptDocContent(nil, "/system/prompt/mode/dev/system_prompt")
	if strings.Contains(got, "{{WORKSPACE_ROLE_ROUTING}}") {
		t.Fatalf("dev system prompt doc should expand role routing marker:\n%s", got)
	}
	if !strings.Contains(got, "### `app_developer` 应用开发工程师") {
		t.Fatalf("dev system prompt doc should include generated role routing:\n%s", got)
	}
}
