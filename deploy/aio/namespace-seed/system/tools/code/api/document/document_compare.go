package document

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type DocumentCompareReq struct {
	InputFileA  string `json:"input_file_a" widget:"name:文件A;type:files;accept:.pdf,.doc,.docx,.ppt,.pptx,.pptm,.odp,.xlsx,.xls,.csv,.txt,.md,.markdown,.json,.html,.htm,image/*;max_size:100MB;max_count:1" validate:"required"`
	InputFileB  string `json:"input_file_b" widget:"name:文件B;type:files;accept:.pdf,.doc,.docx,.ppt,.pptx,.pptm,.odp,.xlsx,.xls,.csv,.txt,.md,.markdown,.json,.html,.htm,image/*;max_size:100MB;max_count:1" validate:"required"`
	FileName    string `json:"file_name" widget:"name:输出文件名前缀;type:input;render_default:document_compare"`
	SheetName   string `json:"sheet_name" widget:"name:指定工作表(Excel可选);type:input;placeholder:留空则输出全部工作表"`
	OCRLanguage string `json:"ocr_language" widget:"name:OCR语言;type:select;options:chi_sim+eng,chi_sim,eng;render_default:chi_sim+eng"`
}

type DocumentCompareResp struct {
	OutputFile string `json:"output_file" widget:"name:输出文件;type:files"`
	Summary    string `json:"summary" widget:"name:对比摘要;type:text_area"`
	Stats      string `json:"stats" widget:"name:统计信息;type:text_area"`
}

type documentDiffLine struct {
	Type     string
	LineA    int
	LineB    int
	ContentA string
	ContentB string
}

func DocumentCompare(ctx *app.Context, resp response.Response) error {
	var req DocumentCompareReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoDocumentCompare(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoDocumentCompare(ctx *app.Context, req *DocumentCompareReq) (*DocumentCompareResp, error) {
	fs := ctx.GetFS()
	filesA := fs.DownloadFiles(req.InputFileA)
	defer fs.RemoveFiles(filesA)
	filesB := fs.DownloadFiles(req.InputFileB)
	defer fs.RemoveFiles(filesB)

	fileA, err := firstDownloadedFile(filesA)
	if err != nil {
		return nil, err
	}
	fileB, err := firstDownloadedFile(filesB)
	if err != nil {
		return nil, err
	}

	fileAName := filepath.Base(fileA)
	fileBName := filepath.Base(fileB)

	resultA, err := extractLocalFileToMarkdown(ctx, fileA, fileAName, req.SheetName, req.OCRLanguage)
	if err != nil {
		return nil, fmt.Errorf("提取文件A失败: %w", err)
	}
	resultB, err := extractLocalFileToMarkdown(ctx, fileB, fileBName, req.SheetName, req.OCRLanguage)
	if err != nil {
		return nil, fmt.Errorf("提取文件B失败: %w", err)
	}

	leftMarkdown := normalizeMarkdownText(resultA.Markdown)
	rightMarkdown := normalizeMarkdownText(resultB.Markdown)
	leftLines := strings.Split(leftMarkdown, "\n")
	rightLines := strings.Split(rightMarkdown, "\n")
	diffs := computeDocumentDiff(leftLines, rightLines)

	added := 0
	removed := 0
	equal := 0
	for _, diff := range diffs {
		switch diff.Type {
		case "insert":
			added++
		case "delete":
			removed++
		default:
			equal++
		}
	}

	outputDir := fs.GetTraceOutputDir()
	baseName := sanitizeMarkdownFileName(req.FileName)
	if baseName == "" {
		baseName = "document_compare"
	}

	leftPath := filepath.Join(outputDir, baseName+"_a.md")
	rightPath := filepath.Join(outputDir, baseName+"_b.md")
	diffPath := filepath.Join(outputDir, baseName+"_diff.html")

	if err := os.WriteFile(leftPath, []byte(leftMarkdown), 0644); err != nil {
		return nil, fmt.Errorf("写入文件A标准化文本失败: %w", err)
	}
	if err := os.WriteFile(rightPath, []byte(rightMarkdown), 0644); err != nil {
		return nil, fmt.Errorf("写入文件B标准化文本失败: %w", err)
	}
	diffHTML := buildDocumentDiffHTML(fileAName, fileBName, diffs)
	if err := os.WriteFile(diffPath, []byte(diffHTML), 0644); err != nil {
		return nil, fmt.Errorf("写入 Diff HTML 失败: %w", err)
	}

	outputFiles := fs.ResponseFiles([]string{leftPath, rightPath, diffPath})

	summary := strings.Join([]string{
		fmt.Sprintf("文件A: %s (%s)", fileAName, resultA.Summary),
		fmt.Sprintf("文件B: %s (%s)", fileBName, resultB.Summary),
	}, "\n")
	stats := strings.Join([]string{
		fmt.Sprintf("A 文本行数: %d", len(leftLines)),
		fmt.Sprintf("B 文本行数: %d", len(rightLines)),
		fmt.Sprintf("相同行: %d", equal),
		fmt.Sprintf("新增行: %d", added),
		fmt.Sprintf("删除行: %d", removed),
	}, "\n")

	return &DocumentCompareResp{
		OutputFile: outputFiles,
		Summary:    summary,
		Stats:      stats,
	}, nil
}

func firstDownloadedFile(files []string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("没有找到输入文件")
	}
	file := files[0]
	if file == "" {
		return "", fmt.Errorf("文件本地路径为空")
	}
	return file, nil
}

func computeDocumentDiff(a []string, b []string) []documentDiffLine {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	var stack []documentDiffLine
	i, j := m, n
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			stack = append(stack, documentDiffLine{Type: "equal", LineA: i, LineB: j, ContentA: a[i-1], ContentB: b[j-1]})
			i--
			j--
			continue
		}
		if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			stack = append(stack, documentDiffLine{Type: "insert", LineB: j, ContentB: b[j-1]})
			j--
			continue
		}
		stack = append(stack, documentDiffLine{Type: "delete", LineA: i, ContentA: a[i-1]})
		i--
	}

	var result []documentDiffLine
	for k := len(stack) - 1; k >= 0; k-- {
		result = append(result, stack[k])
	}
	return result
}

func buildDocumentDiffHTML(titleA string, titleB string, diffs []documentDiffLine) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html><html lang="zh-CN"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0">`)
	sb.WriteString(`<title>Document Compare</title><style>`)
	sb.WriteString(`*{box-sizing:border-box}body{margin:0;padding:24px;background:#f8fafc;color:#0f172a;font:13px/1.7 ui-monospace,SFMono-Regular,Menlo,"Noto Sans SC",monospace}`)
	sb.WriteString(`.container{max-width:1440px;margin:0 auto}.header{margin-bottom:16px}.header h1{margin:0 0 8px;font-size:20px}.header p{margin:0;color:#475569}`)
	sb.WriteString(`table{width:100%;border-collapse:collapse;background:#fff;border:1px solid #e2e8f0;box-shadow:0 6px 24px rgba(15,23,42,.08)}th,td{border-bottom:1px solid #e2e8f0;vertical-align:top}`)
	sb.WriteString(`th{padding:10px 14px;background:#e2e8f0;text-align:left;font-size:13px}td{padding:0 12px;white-space:pre-wrap;word-break:break-word}.ln{width:56px;text-align:right;color:#94a3b8;border-right:1px solid #e2e8f0;padding-right:12px}`)
	sb.WriteString(`.add{background:#dcfce7}.del{background:#fee2e2}.code{min-width:260px}.tag-add::before{content:"+";color:#16a34a;font-weight:700;margin-right:6px}.tag-del::before{content:"-";color:#dc2626;font-weight:700;margin-right:6px}`)
	sb.WriteString(`</style></head><body><div class="container"><div class="header">`)
	sb.WriteString(fmt.Sprintf(`<h1>%s vs %s</h1>`, template.HTMLEscapeString(titleA), template.HTMLEscapeString(titleB)))
	sb.WriteString(`<p>先将文档标准化为 Markdown，再做逐行对比。</p></div><table><tr><th class="ln">#</th><th>`)
	sb.WriteString(template.HTMLEscapeString(titleA))
	sb.WriteString(`</th><th class="ln">#</th><th>`)
	sb.WriteString(template.HTMLEscapeString(titleB))
	sb.WriteString(`</th></tr>`)
	for _, diff := range diffs {
		switch diff.Type {
		case "equal":
			sb.WriteString(fmt.Sprintf(`<tr><td class="ln">%d</td><td class="code">%s</td><td class="ln">%d</td><td class="code">%s</td></tr>`,
				diff.LineA, template.HTMLEscapeString(diff.ContentA), diff.LineB, template.HTMLEscapeString(diff.ContentB)))
		case "delete":
			sb.WriteString(fmt.Sprintf(`<tr class="del"><td class="ln">%d</td><td class="code tag-del">%s</td><td class="ln"></td><td class="code"></td></tr>`,
				diff.LineA, template.HTMLEscapeString(diff.ContentA)))
		case "insert":
			sb.WriteString(fmt.Sprintf(`<tr class="add"><td class="ln"></td><td class="code"></td><td class="ln">%d</td><td class="code tag-add">%s</td></tr>`,
				diff.LineB, template.HTMLEscapeString(diff.ContentB)))
		}
	}
	sb.WriteString(`</table></div></body></html>`)
	return sb.String()
}

var DocumentCompareTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "文档对比",
		Desc:     `将两个文档统一标准化为 Markdown 后进行逐行对比，输出标准化文本和可直接访问的 HTML Diff 页面。适合合同、方案、会议纪要、报表版本比对。`,
		Tags:     []string{"文档处理", "文档对比", "Diff", "Markdown", "PDF", "Word", "Excel"},
		Request:  &DocumentCompareReq{},
		Response: &DocumentCompareResp{},
	},
}

func init() {
	packageContext.POST("compare.form", DocumentCompare, DocumentCompareTemplate)
}
