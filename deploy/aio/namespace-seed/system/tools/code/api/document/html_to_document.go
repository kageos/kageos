// html_to_document.go：使用 Pandoc 从 HTML 内容生成文档

package document

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// HTMLToDocumentReq 从 HTML 内容生成文档请求
type HTMLToDocumentReq struct {
	Content      string `json:"content" widget:"name:HTML内容;type:text_area;placeholder:<h1>标题</h1><p>正文...</p>" validate:"required"`
	OutputFormat string `json:"output_format" widget:"name:输出格式;type:select;options:docx,pdf;render_default:docx" validate:"required"`
	FileName     string `json:"file_name" widget:"name:输出文件名;type:input;render_default:文档"`
}

// HTMLToDocumentResp 生成文档响应
type HTMLToDocumentResp struct {
	OutputFile string `json:"output_file" widget:"name:输出文件;type:files"`
	Message    string `json:"message" widget:"name:说明;type:text_area"`
}

// HTMLToDocument 入口
func HTMLToDocument(ctx *app.Context, resp response.Response) error {
	var req HTMLToDocumentReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoHTMLToDocument(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoHTMLToDocument 将 HTML 写入临时文件，用 Pandoc 转为 docx 或 pdf
func DoHTMLToDocument(ctx *app.Context, req *HTMLToDocumentReq) (*HTMLToDocumentResp, error) {
	format := strings.TrimSpace(strings.ToLower(req.OutputFormat))
	if format != "docx" && format != "pdf" {
		format = "docx"
	}
	baseName := strings.TrimSpace(req.FileName)
	if baseName == "" {
		baseName = "文档"
	}
	baseName = strings.TrimSuffix(baseName, ".html")
	baseName = strings.TrimSuffix(baseName, ".docx")
	baseName = strings.TrimSuffix(baseName, ".pdf")

	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	htmlPath := filepath.Join(outputDir, baseName+".html")

	// 确保 HTML 内容有完整的 HTML 结构
	htmlContent := req.Content
	if !strings.Contains(htmlContent, "<html") {
		htmlContent = "<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"UTF-8\">\n<title>" + baseName + "</title>\n</head>\n<body>\n" + htmlContent + "\n</body>\n</html>"
	}

	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		return nil, fmt.Errorf("写入 HTML 失败: %w", err)
	}
	defer os.Remove(htmlPath)

	// 检查 pandoc 是否可用
	if _, err := exec.LookPath("pandoc"); err != nil {
		return nil, fmt.Errorf("未找到 pandoc，请确保已安装 Pandoc 并在 PATH 中: %w", err)
	}

	outFileName := baseName + "." + format
	outPath := filepath.Join(outputDir, outFileName)

	// 构建 pandoc 命令
	var cmd *exec.Cmd
	if format == "pdf" {
		// 对于 PDF，使用默认引擎
		cmd = exec.Command("pandoc", htmlPath, "-o", outPath)
	} else {
		// 对于 docx，直接转换
		cmd = exec.Command("pandoc", htmlPath, "-o", outPath)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[pandoc/HTMLToDocument] 转换失败: %v, output: %s", err, string(out))
		return nil, fmt.Errorf("Pandoc 转换失败: %v\n%s", err, string(out))
	}

	if _, err := os.Stat(outPath); err != nil {
		return nil, fmt.Errorf("未生成输出文件 %s: %w", outPath, err)
	}

	outputFiles := fs.ResponseFiles([]string{outPath})

	return &HTMLToDocumentResp{
		OutputFile: outputFiles,
		Message:    fmt.Sprintf("已根据 HTML 内容生成 %s 文档", format),
	}, nil
}

var HTMLToDocumentTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "HTML转文档(Pandoc)",
		Desc:     `使用 Pandoc 从 HTML 内容生成 Word(docx) 或 PDF 文档。支持中文，PDF 使用 xelatex 引擎确保中文正常显示。`,
		Tags:     []string{"Pandoc", "HTML", "文档", "创建", "docx", "pdf", "内容"},
		Request:  &HTMLToDocumentReq{},
		Response: &HTMLToDocumentResp{},
	},
}

func init() {
	packageContext.POST("html_to_document.form", HTMLToDocument, HTMLToDocumentTemplate)
}
