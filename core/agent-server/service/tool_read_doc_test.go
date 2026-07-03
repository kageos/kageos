package service

import (
	"context"
	"strings"
	"testing"
)

func TestRunReadDocToolSuggestsPromptDocChildren(t *testing.T) {
	content, isError := runReadDocTool(context.Background(), readDocArgs{
		Directory: "/system/prompt/roles",
	}, "/system/demo")
	if !isError {
		t.Fatalf("read_doc should fail for non-leaf prompt path, got content=%q", content)
	}
	for _, want := range []string{
		"目录前缀本身不一定是文档",
		"/system/prompt/roles/router",
		"/system/prompt/roles/app-developer",
		"search",
		"可读的目录",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("read_doc missing helpful hint %q in:\n%s", want, content)
		}
	}
}
