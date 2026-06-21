// excel_to_markdown.go：Excel 转为 Markdown 表格，路由 POST /document/excel/excel_to_markdown.form
// 读取 xlsx 所有工作表，输出为 Markdown 表格文本。

package table

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/xuri/excelize/v2"
)

// ExcelToMarkdownReq Excel 转 Markdown 请求
type ExcelToMarkdownReq struct {
	InputFiles string `json:"input_files" widget:"name:上传Excel文件;type:files;accept:.xlsx,.xls;max_size:50MB;max_count:10" validate:"required"`
	SheetName  string `json:"sheet_name" widget:"name:指定工作表(可选);type:input;placeholder:留空则输出所有工作表"`
}

// ExcelToMarkdownResp Excel 转 Markdown 响应
type ExcelToMarkdownResp struct {
	MarkdownText string `json:"markdown_text" widget:"name:Markdown表格;type:text_area"`
	Summary      string `json:"summary" widget:"name:说明;type:text_area"`
}

// ExcelToMarkdown 入口
func ExcelToMarkdown(ctx *app.Context, resp response.Response) error {
	var req ExcelToMarkdownReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoExcelToMarkdown(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoExcelToMarkdown 读取 xlsx 所有（或指定）工作表，输出为 Markdown 表格文本
func DoExcelToMarkdown(ctx *app.Context, req *ExcelToMarkdownReq) (*ExcelToMarkdownResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	wantSheet := strings.TrimSpace(req.SheetName)
	var outParts []string
	var summaryParts []string

	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[excel/ToMarkdown] 文件 %s 无本地路径，跳过", filepath.Base(file))
			summaryParts = append(summaryParts, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		f, err := excelize.OpenFile(file)
		if err != nil {
			logger.Errorf(ctx, "[excel/ToMarkdown] 打开失败 %s: %v", filepath.Base(file), err)
			summaryParts = append(summaryParts, fmt.Sprintf("失败 %s: %v", filepath.Base(file), err))
			continue
		}

		sheetList := f.GetSheetList()
		if len(sheetList) == 0 {
			f.Close()
			summaryParts = append(summaryParts, fmt.Sprintf("跳过 %s: 无工作表", filepath.Base(file)))
			continue
		}

		exportSheets := sheetList
		if wantSheet != "" {
			exportSheets = nil
			for _, s := range sheetList {
				if s == wantSheet {
					exportSheets = []string{s}
					break
				}
			}
			if len(exportSheets) == 0 {
				f.Close()
				summaryParts = append(summaryParts, fmt.Sprintf("跳过 %s: 未找到工作表「%s」", filepath.Base(file), wantSheet))
				continue
			}
		}

		for _, sheetName := range exportSheets {
			rows, err := f.GetRows(sheetName)
			if err != nil {
				summaryParts = append(summaryParts, fmt.Sprintf("  %s / %s 读取失败: %v", filepath.Base(file), sheetName, err))
				continue
			}
			md := rowsToMarkdownTable(rows)
			if len(exportSheets) > 1 || len(files) > 1 {
				outParts = append(outParts, fmt.Sprintf("### %s — %s\n\n%s", filepath.Base(file), sheetName, md))
			} else {
				outParts = append(outParts, md)
			}
		}
		f.Close()
		summaryParts = append(summaryParts, fmt.Sprintf("已转 %s，%d 个工作表", filepath.Base(file), len(exportSheets)))
	}

	markdownText := strings.TrimSpace(strings.Join(outParts, "\n\n"))
	if markdownText == "" {
		markdownText = "(无数据)"
	}
	return &ExcelToMarkdownResp{
		MarkdownText: markdownText,
		Summary:      strings.Join(summaryParts, "\n"),
	}, nil
}

// rowsToMarkdownTable 将 [][]string 转为 Markdown 表格（首行为表头，第二行为分隔 |---|）
func rowsToMarkdownTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	// 表头
	header := rows[0]
	b.WriteString("| ")
	b.WriteString(strings.Join(escapeMarkdownCells(header), " | "))
	b.WriteString(" |\n")
	// 分隔行
	sep := make([]string, len(header))
	for i := range sep {
		sep[i] = "---"
	}
	b.WriteString("| ")
	b.WriteString(strings.Join(sep, " | "))
	b.WriteString(" |\n")
	// 数据行
	for _, row := range rows[1:] {
		// 补齐列数
		for len(row) < len(header) {
			row = append(row, "")
		}
		if len(row) > len(header) {
			row = row[:len(header)]
		}
		b.WriteString("| ")
		b.WriteString(strings.Join(escapeMarkdownCells(row), " | "))
		b.WriteString(" |\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// escapeMarkdownCells 对单元格中的 | 和换行做转义，便于 Markdown 表格正确显示
func escapeMarkdownCells(cells []string) []string {
	out := make([]string, len(cells))
	for i, c := range cells {
		c = strings.ReplaceAll(c, "|", "\\|")
		c = strings.ReplaceAll(c, "\n", " ")
		c = strings.TrimSpace(c)
		out[i] = c
	}
	return out
}

var ExcelToMarkdownTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Excel 转 Markdown 表格",
		Desc:     `上传 xlsx 或 xls，将所有工作表转为 Markdown 表格文本（首行为表头）。可指定单个工作表，不指定则输出全部。适合写文档、放 README、或给大模型做结构化输入。`,
		Tags:     []string{"Excel", "xlsx", "xls", "Markdown", "Excel转Markdown", "excel转markdown", "表格转Markdown", "多工作表", "写文档", "README"},
		Request:  &ExcelToMarkdownReq{},
		Response: &ExcelToMarkdownResp{},
	},
}

func init() {
	packageContext.POST("excel_to_markdown.form", ExcelToMarkdown, ExcelToMarkdownTemplate)
}
