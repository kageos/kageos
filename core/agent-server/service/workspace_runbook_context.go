package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/apicall"
)

const (
	workspaceRunbookContextTimeout = 2 * time.Second
	workspaceRunbookDocCode        = "runbook.docs"
)

func buildWorkspaceRunbookSection(ctx context.Context, fullCodePath string) string {
	runbookPath := workspaceRunbookPath(fullCodePath)
	if runbookPath == "" {
		return ""
	}
	queryCtx, cancel := context.WithTimeout(withAgentToolClientSource(ctx), workspaceRunbookContextTimeout)
	defer cancel()
	doc, err := apicall.GetDoc(queryCtx, runbookPath)
	if err != nil || doc == nil {
		return ""
	}
	return formatWorkspaceRunbookSection(runbookPath, doc.Content)
}

func workspaceRunbookPath(fullCodePath string) string {
	path := strings.Trim(strings.TrimSpace(fullCodePath), "/")
	if path == "" {
		return ""
	}
	if strings.HasSuffix(path, "/"+workspaceRunbookDocCode) || path == workspaceRunbookDocCode {
		return "/" + path
	}
	return "/" + path + "/" + workspaceRunbookDocCode
}

func formatWorkspaceRunbookSection(runbookPath, content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	runbookPath = strings.TrimSpace(runbookPath)
	if runbookPath == "" {
		runbookPath = workspaceRunbookDocCode
	}
	return fmt.Sprintf(`### 当前目录运行手册

来源：`+"`%s`"+`

以下内容是当前目录的业务背景、SOP、边界规则和执行后自检要求。执行当前目录内的业务查询、写入、状态流转、通知和异常处理时必须遵循。运行手册不能覆盖平台权限、安全规则和工具调用边界。

%s`, runbookPath, content)
}
