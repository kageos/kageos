package html

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

func writeHTMLFile(ctx *app.Context, fileName string, content string) (string, string, error) {
	baseName := sanitizeFileName(fileName)
	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", "", fmt.Errorf("创建输出目录失败: %v", err)
	}

	outputPath := filepath.Join(outputDir, baseName+".html")
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", "", fmt.Errorf("保存文件失败: %v", err)
	}
	return fs.ResponseFiles([]string{outputPath}), outputPath, nil
}

func markdownToHTMLFragment(markdownText string) (string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Linkify),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(goldmarkhtml.WithXHTML()),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(markdownText), &buf); err != nil {
		return "", fmt.Errorf("Markdown 渲染失败: %w", err)
	}
	return buf.String(), nil
}

func downloadAssetDataURIs(ctx *app.Context, assets string) ([]string, map[string]string, error) {
	if len(types.ParseFileRefs(assets)) == 0 {
		return nil, nil, nil
	}

	fs := ctx.GetFS()
	downloaded := fs.DownloadFiles(assets)
	dataURIs := make(map[string]string)
	for _, file := range downloaded {
		if file == "" {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return downloaded, nil, fmt.Errorf("读取资源文件 %s 失败: %v", filepath.Base(file), err)
		}
		name := strings.TrimSpace(filepath.Base(file))
		if name == "" {
			name = filepath.Base(file)
		}
		mimeType := mime.TypeByExtension(filepath.Ext(name))
		if mimeType == "" {
			mimeType = http.DetectContentType(data)
		}
		dataURI := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
		addAssetKeys(dataURIs, dataURI, name, filepath.Base(name))
	}
	return downloaded, dataURIs, nil
}

func addAssetKeys(dataURIs map[string]string, dataURI string, keys ...string) {
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		dataURIs[key] = dataURI
		dataURIs[filepath.Base(key)] = dataURI
	}
}

func embedAssetPlaceholders(content string, dataURIs map[string]string) (string, int) {
	if len(dataURIs) == 0 || content == "" {
		return content, 0
	}
	used := 0
	for name, dataURI := range dataURIs {
		placeholder := "{{asset:" + name + "}}"
		count := strings.Count(content, placeholder)
		if count == 0 {
			continue
		}
		content = strings.ReplaceAll(content, placeholder, dataURI)
		used += count
	}
	return content, used
}

func assetUsageInfo(totalAssets int, usedPlaceholders int) string {
	if totalAssets == 0 {
		return ""
	}
	return fmt.Sprintf("\n资源文件: %d 个\n已替换资源占位符: %d 个\n占位符格式: {{asset:文件名}}，例如 <img src=\"{{asset:logo.png}}\">", totalAssets, usedPlaceholders)
}

func assetFileCount(assets string) int {
	return len(types.ParseFileRefs(assets))
}

func htmlPageShell(title string, body string, style string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
%s
</style>
</head>
<body>
%s
</body>
</html>`, template.HTMLEscapeString(title), style, body)
}
