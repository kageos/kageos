package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/logger"
)

type ReadAppLogTool struct{}

type readAppLogArgs struct {
	Directory    string `json:"directory" schema_desc:"日志所属目录，不传则当前目录"`
	FullCodePath string `json:"full_code_path" schema_ignore:"true"`
	Version      string `json:"version" schema_desc:"日志版本，如 v48"`
	Keyword      string `json:"keyword" schema_desc:"关键词过滤"`
	Lines        *int   `json:"lines" schema_desc:"返回行数"`
	ContextLines *int   `json:"context_lines" schema_desc:"命中上下文行数"`
	MaxMatches   *int   `json:"max_matches" schema_desc:"最大命中数"`
	IgnoreCase   *bool  `json:"ignore_case" schema_desc:"是否忽略大小写"`
}

var readAppLogToolDef = toolDefinition[readAppLogArgs](
	"read_app_log",
	"读取应用日志（workspace/logs），用于排查 bug、报错、超时、异常行为等运行问题。默认读取当前版本日志；可传 version 指定历史版本（如 v48）。支持按关键词过滤（keyword），并返回命中上下文。参数：directory（可选，不传则当前目录）、version（可选，默认当前版本）、lines（可选，默认 200，最大 1000）、keyword（可选）、context_lines（可选，默认 2，最大 5）、max_matches（可选，默认 50，最大 200）、ignore_case（可选，默认 false）。",
)

func (t *ReadAppLogTool) Definition() dto.ToolDef {
	return readAppLogToolDef
}

func (t *ReadAppLogTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[readAppLogArgs](call.Args)
	if err != nil {
		return toolResult("read_app_log 参数解析失败: "+err.Error(), true)
	}
	content, isError := runReadAppLogTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

// runReadAppLogTool 读取应用日志（支持 version、关键词检索）
func runReadAppLogTool(ctx context.Context, args readAppLogArgs, currentFullCodePath string) (string, bool) {
	targetPath := resolveDirectoryArg(args.Directory, args.FullCodePath, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	}
	if targetPath != "" && !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	req := &dto.ReadAppLogReq{
		FullCodePath: targetPath,
		Version:      strings.TrimSpace(args.Version),
		Keyword:      args.Keyword,
		ContextLines: 0,
		MaxMatches:   0,
		IgnoreCase:   false,
	}
	if args.Lines != nil {
		req.Lines = *args.Lines
	}
	if args.ContextLines != nil {
		req.ContextLines = *args.ContextLines
	}
	if args.MaxMatches != nil {
		req.MaxMatches = *args.MaxMatches
	}
	if args.IgnoreCase != nil {
		req.IgnoreCase = *args.IgnoreCase
	}
	resp, err := apicall.ReadAppLog(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[ReadAppLog] ReadAppLog 失败: %v", err)
		return "read_app_log 调用失败: " + err.Error(), true
	}
	if !resp.Success {
		return "read_app_log: " + resp.Message, true
	}
	msg := fmt.Sprintf("日志读取成功：版本=%s，文件=%s，总行数=%d，返回行数=%d，命中数=%d，截断=%t",
		resp.ResolvedVersion, resp.LogFile, resp.TotalLines, resp.ReturnedLines, resp.MatchCount, resp.Truncated)
	if resp.Content != "" {
		msg += "\n\n" + resp.Content
	}
	return msg, false
}
