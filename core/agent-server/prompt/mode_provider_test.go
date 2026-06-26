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
		!strings.Contains(got, "### `app_operator` 应用执行") ||
		!strings.Contains(got, "## 工作台背景与能力地图") ||
		!strings.Contains(got, "## 角色速查与选择") ||
		!strings.Contains(got, "## 执行阶段与上下文交接") ||
		!strings.Contains(got, "Kageos 工作台不是普通聊天窗口") ||
		!strings.Contains(got, "能力地图") ||
		!strings.Contains(got, "角色文档不是主路由入口") ||
		!strings.Contains(got, "`automation_operator` 自动执行配置") ||
		!strings.Contains(got, "定时函数、定时会话") ||
		!strings.Contains(got, "Form 默认调度") ||
		!strings.Contains(got, "固定函数和固定参数用定时函数") ||
		!strings.Contains(got, "`reviewer` 代码审查分析师") ||
		!strings.Contains(got, "可以沿用当前角色继续推进") ||
		!strings.Contains(got, "主执行目录/绑定目录") ||
		!strings.Contains(got, "其他空间函数或连接器函数完整路径") ||
		!strings.Contains(got, "使用当前软件完成事情") ||
		!strings.Contains(got, "不要因为用户说“创建、处理、生成、提交、更新、整理、查看”等动词就默认写 PRD 或开发") ||
		!strings.Contains(got, "`tables.fields` 是模型字段，`tables.search_fields` 是查询请求字段") {
		t.Fatalf("dev prompt should include generated role routing:\n%s", got)
	}
	if strings.Contains(got, "每次收到用户需求后，必须先调用 `change_role`") ||
		strings.Contains(got, "如果仍适合当前角色，也通过 `change_role` 明确沿用当前角色") {
		t.Fatalf("dev prompt should not force ritual change_role calls when the current role still matches:\n%s", got)
	}
	if strings.Index(got, "### `app_operator` 应用执行") > strings.Index(got, "### `product_manager` 产品经理") {
		t.Fatalf("dev prompt should present app_operator before product_manager:\n%s", got)
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
