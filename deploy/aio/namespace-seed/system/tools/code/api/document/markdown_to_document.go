package document

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type MarkdownToDocumentReq struct {
	MarkdownText   string `json:"markdown_text" widget:"name:Markdown 内容;type:text_area;placeholder:可直接输入 Markdown；如果上传了 Markdown 文件，此项可留空"`
	InputFiles     string `json:"input_files" widget:"name:上传 Markdown/文本文件;type:files;accept:.md,.markdown,.txt,text/*;max_size:50MB;max_count:20"`
	OutputFormat   string `json:"output_format" widget:"name:输出格式;type:select;options:docx,html,pdf;options_colors:409EFF,67C23A,E6A23C;render_default:docx" validate:"required,oneof=docx html pdf"`
	Title          string `json:"title" widget:"name:文档标题;type:input;placeholder:可选，会写入文档元数据"`
	OutputFileName string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，仅单输入时生效，例如 report.docx"`
}

type MarkdownToDocumentResp struct {
	OutputFiles string `json:"output_files" widget:"name:输出文档;type:files"`
	ConvertInfo string `json:"convert_info" widget:"name:转换信息;type:text_area"`
}

func MarkdownToDocument(ctx *app.Context, resp response.Response) error {
	var req MarkdownToDocumentReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoMarkdownToDocument(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoMarkdownToDocument(ctx *app.Context, req *MarkdownToDocumentReq) (*MarkdownToDocumentResp, error) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		return nil, fmt.Errorf("未找到 pandoc，请确认运行环境已安装 Pandoc")
	}
	if req.OutputFormat == "pdf" {
		if _, err := exec.LookPath("libreoffice"); err != nil {
			return nil, fmt.Errorf("输出 PDF 需要 LibreOffice，请确认运行环境已安装 libreoffice")
		}
	}

	fs := ctx.GetFS()
	inputs, downloaded, err := loadMarkdownInputs(ctx, req.MarkdownText, req.InputFiles)
	if err != nil {
		return nil, err
	}
	defer fs.RemoveFiles(downloaded)
	if len(inputs) == 0 {
		return nil, fmt.Errorf("请提供 Markdown 内容或上传 Markdown 文件")
	}

	outputDir := fs.GetTraceOutputDir()
	var outputPaths []string
	var infos []string
	for i, input := range inputs {
		outputName := markdownOutputName(input.Name, req.OutputFileName, req.OutputFormat, len(inputs) == 1)
		outputPath := filepath.Join(outputDir, outputName)
		if req.OutputFormat == "pdf" {
			if err := markdownToPDFViaHTML(ctx, input.Path, outputDir, outputPath, req.Title, i); err != nil {
				infos = append(infos, fmt.Sprintf("失败 %s: %v", input.Name, err))
				continue
			}
		} else {
			if err := runPandocToFile(ctx, input.Path, outputPath, req.OutputFormat, req.Title); err != nil {
				infos = append(infos, fmt.Sprintf("失败 %s: %v", input.Name, err))
				continue
			}
		}
		outputPaths = append(outputPaths, outputPath)
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", input.Name, outputName))
	}
	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功转换的文档\n%s", strings.Join(infos, "\n"))
	}
	return &MarkdownToDocumentResp{
		OutputFiles: fs.ResponseFiles(outputPaths),
		ConvertInfo: fmt.Sprintf("Markdown 转文档完成\n输出格式: %s\n输入数: %d\n输出文件数: %d\n\n详情:\n%s",
			req.OutputFormat, len(inputs), len(outputPaths), strings.Join(infos, "\n")),
	}, nil
}

type markdownInput struct {
	Name string
	Path string
}

func loadMarkdownInputs(ctx *app.Context, text string, fileRefs string) ([]markdownInput, []string, error) {
	fs := ctx.GetFS()
	var downloaded []string
	var inputs []markdownInput
	if strings.TrimSpace(fileRefs) != "" {
		downloaded = fs.DownloadFiles(fileRefs)
		for _, path := range downloaded {
			if path != "" {
				inputs = append(inputs, markdownInput{Name: filepath.Base(path), Path: path})
			}
		}
	}
	if strings.TrimSpace(text) != "" {
		outputDir := fs.GetTraceOutputDir()
		inputPath := filepath.Join(outputDir, "markdown_input.md")
		if err := os.WriteFile(inputPath, []byte(text), 0644); err != nil {
			return nil, downloaded, fmt.Errorf("写入 Markdown 临时文件失败: %w", err)
		}
		inputs = append(inputs, markdownInput{Name: "markdown_input.md", Path: inputPath})
	}
	return inputs, downloaded, nil
}

func runPandocToFile(ctx *app.Context, inputPath, outputPath, format, title string) error {
	args := []string{inputPath, "-o", outputPath, "--standalone", "--from", "gfm"}
	if strings.TrimSpace(title) != "" {
		args = append(args, "--metadata", "title="+title)
	}
	out, err := exec.Command("pandoc", args...).CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[MarkdownToDocument] pandoc 失败: %v, output: %s", err, string(out))
		return fmt.Errorf("pandoc 转 %s 失败: %v\n%s", format, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func markdownToPDFViaHTML(ctx *app.Context, inputPath, outputDir, outputPath, title string, index int) error {
	tempHTML := filepath.Join(outputDir, fmt.Sprintf("markdown_pdf_%03d.html", index+1))
	if err := runPandocToFile(ctx, inputPath, tempHTML, "html", title); err != nil {
		return err
	}
	defer os.Remove(tempHTML)
	out, err := exec.Command("libreoffice", "--headless", "--convert-to", "pdf", "--outdir", outputDir, tempHTML).CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[MarkdownToDocument] libreoffice PDF 失败: %v, output: %s", err, string(out))
		return fmt.Errorf("LibreOffice 转 PDF 失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	generated := filepath.Join(outputDir, strings.TrimSuffix(filepath.Base(tempHTML), filepath.Ext(tempHTML))+".pdf")
	if generated != outputPath {
		_ = os.Remove(outputPath)
		if err := os.Rename(generated, outputPath); err != nil {
			return fmt.Errorf("移动 PDF 输出失败: %w", err)
		}
	}
	return nil
}

func markdownOutputName(inputName, customName, ext string, single bool) string {
	if single && strings.TrimSpace(customName) != "" {
		return pandocEnsureExt(customName, ext)
	}
	base := pandocSafeBase(inputName, "document")
	return base + "." + ext
}

func pandocSafeBase(name, fallback string) string {
	name = strings.TrimSuffix(filepath.Base(strings.TrimSpace(name)), filepath.Ext(name))
	if name == "" {
		name = fallback
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	name = replacer.Replace(name)
	if strings.TrimSpace(name) == "" {
		return fallback
	}
	return name
}

func pandocEnsureExt(name, ext string) string {
	base := pandocSafeBase(name, "document")
	return base + "." + strings.TrimPrefix(ext, ".")
}

var MarkdownToDocumentTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Markdown 转文档",
		Desc:     `将 Markdown/GFM 文本或文件转换为 docx、html 或 pdf。PDF 会先由 Pandoc 转 HTML，再用 LibreOffice 转 PDF，避免依赖 LaTeX。适合报告、说明书、会议纪要和交付文档生成。`,
		Tags:     []string{"Markdown", "Pandoc", "docx", "html", "pdf", "文档转换"},
		Request:  &MarkdownToDocumentReq{},
		Response: &MarkdownToDocumentResp{},
	},
}

func init() {
	packageContext.POST("markdown_to_document.form", MarkdownToDocument, MarkdownToDocumentTemplate)
}
