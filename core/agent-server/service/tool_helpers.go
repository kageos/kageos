package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
)

const agentToolClientSource = "agent"

// decodeToolArgs 将 tool args 反序列化到强类型结构体；未知字段保持忽略，避免对旧调用方过于敏感。
func decodeToolArgs[T any](args map[string]interface{}) (T, error) {
	var out T
	if len(args) == 0 {
		return out, nil
	}
	data, err := json.Marshal(args)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	return out, nil
}

// formatJSONResult 将 map 序列化为可读字符串
func formatJSONResult(m map[string]interface{}) (string, bool) {
	if m == nil {
		return "{}", false
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", m), false
	}
	return string(b), false
}

// ToToolArgs 将 interface{} 转为 map[string]interface{}，供 CallTool 使用
// JSON 反序列化后，object→map[string]interface{}；nil/null/缺省→nil，按空 map 处理
func ToToolArgs(v interface{}) map[string]interface{} {
	if v == nil {
		return map[string]interface{}{}
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

// splitFileNames 将 file_name 按逗号拆成多个文件名（如 "a.go,b.go" -> ["a.go","b.go"]），单文件返回单元素
func splitFileNames(fileName string) []string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return nil
	}
	parts := strings.Split(fileName, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// splitDirectoryPaths 将 directory 按逗号拆成多个路径并去重（如 "/a,/b,/a" -> ["/a","/b"]），保持顺序
func splitDirectoryPaths(directory string) []string {
	directory = strings.TrimSpace(directory)
	if directory == "" {
		return nil
	}
	parts := strings.Split(directory, ",")
	seen := make(map[string]bool)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

// resolveDirectoryArg 兼容 directory/full_code_path 两种传法；都未传时回退到默认路径。
func resolveDirectoryArg(directory string, fullCodePath string, defaultPath string) string {
	if s := strings.TrimSpace(directory); s != "" {
		return s
	}
	if s := strings.TrimSpace(fullCodePath); s != "" {
		return s
	}
	return defaultPath
}

func normalizeAbsoluteToolPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func resolveFullCodePathArg(fullCodePath string, defaultPath string) string {
	if path := normalizeAbsoluteToolPath(fullCodePath); path != "" {
		return path
	}
	return normalizeAbsoluteToolPath(defaultPath)
}

func withAgentToolClientSource(ctx context.Context) context.Context {
	return contextx.WithClientSource(ctx, agentToolClientSource)
}

func workspaceFileLineCount(file dto.WorkspaceContextFile) int {
	if file.LineCount > 0 {
		return file.LineCount
	}
	if file.Content == "" {
		return 0
	}
	lines := strings.Split(file.Content, "\n")
	lineCount := len(lines)
	if lineCount > 0 && lines[lineCount-1] == "" {
		lineCount--
	}
	return lineCount
}
