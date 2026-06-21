package document

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type DocumentToMarkdownReq struct {
	InputFiles        string `json:"input_files" widget:"name:上传文件;type:files;accept:.pdf,.doc,.docx,.ppt,.pptx,.pptm,.odp,.xlsx,.xls,.csv,.txt,.md,.markdown,.json,.html,.htm,image/*;max_size:100MB;max_count:20" validate:"required"`
	FileName          string `json:"file_name" widget:"name:输出文件名;type:input;render_default:documents_markdown"`
	SheetName         string `json:"sheet_name" widget:"name:指定工作表(Excel可选);type:input;placeholder:留空则输出全部工作表"`
	OCRLanguage       string `json:"ocr_language" widget:"name:OCR语言;type:select;options:chi_sim+eng,chi_sim,eng;render_default:chi_sim+eng"`
	IncludeFileTitles bool   `json:"include_file_titles" widget:"name:按文件插入标题;type:switch;render_default:true"`
}

type DocumentToMarkdownResp struct {
	MarkdownText string `json:"markdown_text" widget:"name:Markdown内容;type:text_area"`
	OutputFile   string `json:"output_file" widget:"name:Markdown文件;type:files"`
	Summary      string `json:"summary" widget:"name:说明;type:text_area"`
}

func DocumentToMarkdown(ctx *app.Context, resp response.Response) error {
	var req DocumentToMarkdownReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoDocumentToMarkdown(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoDocumentToMarkdown(ctx *app.Context, req *DocumentToMarkdownReq) (*DocumentToMarkdownResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	var markdownParts []string
	var summaryParts []string
	includeTitles := req.IncludeFileTitles || len(files) > 1

	for _, file := range files {
		if file == "" {
			summaryParts = append(summaryParts, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}
		result, err := extractLocalFileToMarkdown(ctx, file, filepath.Base(file), req.SheetName, req.OCRLanguage)
		if err != nil {
			summaryParts = append(summaryParts, fmt.Sprintf("失败 %s: %v", filepath.Base(file), err))
			continue
		}
		content := strings.TrimSpace(result.Markdown)
		if content == "" {
			summaryParts = append(summaryParts, fmt.Sprintf("跳过 %s: 未提取到可用内容", filepath.Base(file)))
			continue
		}
		if includeTitles {
			content = fmt.Sprintf("## %s\n\n%s", filepath.Base(file), content)
		}
		markdownParts = append(markdownParts, content)
		summaryParts = append(summaryParts, fmt.Sprintf("成功 %s: %s", filepath.Base(file), result.Summary))
	}

	if len(markdownParts) == 0 {
		return nil, fmt.Errorf("所有文件都未成功提取 Markdown")
	}

	markdownText := strings.TrimSpace(strings.Join(markdownParts, "\n\n"))
	outputDir := fs.GetTraceOutputDir()
	baseName := sanitizeMarkdownFileName(req.FileName)
	if baseName == "" {
		baseName = "documents_markdown"
	}
	outputPath := filepath.Join(outputDir, baseName+".md")
	if err := os.WriteFile(outputPath, []byte(markdownText), 0644); err != nil {
		return nil, fmt.Errorf("写入 Markdown 文件失败: %w", err)
	}

	outputFiles := fs.ResponseFiles([]string{outputPath})

	return &DocumentToMarkdownResp{
		MarkdownText: markdownText,
		OutputFile:   outputFiles,
		Summary:      strings.Join(summaryParts, "\n"),
	}, nil
}

var DocumentToMarkdownTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "文档转 Markdown",
		Desc:     `将 PDF、Word、PPT、Excel、HTML、JSON、CSV、图片等文件统一提取为 Markdown。适合作为工作台读取资料、后续对比、结构化抽取、报告生成的标准入口。`,
		Tags:     []string{"文档处理", "Markdown", "PDF", "Word", "PPT", "Excel", "OCR", "资料抽取"},
		Request:  &DocumentToMarkdownReq{},
		Response: &DocumentToMarkdownResp{},
	},
}

func init() {
	packageContext.POST("to_markdown.form", DocumentToMarkdown, DocumentToMarkdownTemplate)
}
