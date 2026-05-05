package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
)

type ReadDocTool struct{}

type readDocArgs struct {
	Directory    string `json:"directory" schema_desc:"文档路径，支持逗号分隔多路径" schema_required:"true"`
	FullCodePath string `json:"full_code_path" schema_ignore:"true"`
}

var readDocToolDef = toolDefinition[readDocArgs](
	"read_doc",
	"读取文档内容。传 directory 定位文档（单路径如 /system/prompt/sdk/agent-app-sdk-readme，或多路径逗号分隔如 /system/prompt/a,/system/prompt/b）。当 directory 指向文档目录时，会返回该目录下全部文档。系统消息中会列出可读文档的 directory 及名称。",
)

func (t *ReadDocTool) Definition() dto.ToolDef {
	return readDocToolDef
}

func (t *ReadDocTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[readDocArgs](call.Args)
	if err != nil {
		return toolResult("read_doc 参数解析失败: "+err.Error(), true)
	}
	content, isError := runReadDocTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

// runReadDocTool 读取文档（directory 唯一定位，内置或工作区；支持逗号分隔多路径）
func runReadDocTool(ctx context.Context, args readDocArgs, currentFullCodePath string) (string, bool) {
	dirArg := resolveDirectoryArg(args.Directory, args.FullCodePath, currentFullCodePath)
	if dirArg == "" {
		return "read_doc 需传 directory。", true
	}
	paths := splitDirectoryPaths(dirArg)
	if len(paths) == 0 {
		return "read_doc 需传 directory。", true
	}

	var sb strings.Builder
	var hasError bool
	for i, fullCodePath := range paths {
		if fullCodePath == "" {
			continue
		}
		if !strings.HasPrefix(fullCodePath, "/") {
			fullCodePath = "/" + fullCodePath
		}
		if prompt.IsLegacyWorkspacePromptPath(fullCodePath) {
			if i > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString("## " + fullCodePath + "\n\n旧 `/system/prompt/workspace/*` SOP 路径已下线。请使用 `read_skill` 读取匹配的 Skill，再按 skill 的 required_docs 读取 SDK、平台总览或案例文档。")
			hasError = true
			continue
		}

		if prompt.IsPromptDocPath(fullCodePath) {
			docName, content := prompt.GetPromptDocContent(ctx, fullCodePath)
			if content == "" {
				if i > 0 {
					sb.WriteString("\n\n")
				}
				sb.WriteString(fmt.Sprintf("## %s\n\n未找到：directory=%s。请使用系统消息中列出的 directory。", fullCodePath, fullCodePath))
				hasError = true
				continue
			}
			if docName == "" {
				docName = fullCodePath
			}
			if i > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(fmt.Sprintf("## %s\n\n%s", docName, content))
			continue
		}

		doc, err := apicall.GetDoc(ctx, fullCodePath)
		if err != nil {
			if i > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(fmt.Sprintf("## %s\n\n获取文档失败: %v", fullCodePath, err))
			hasError = true
			continue
		}
		if doc == nil || doc.Content == "" {
			if i > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(fmt.Sprintf("## %s\n\n文档《%s》无正文内容。", fullCodePath, fullCodePath))
			continue
		}
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("## %s\n\n%s", doc.Name, doc.Content))
	}
	return sb.String(), hasError
}
