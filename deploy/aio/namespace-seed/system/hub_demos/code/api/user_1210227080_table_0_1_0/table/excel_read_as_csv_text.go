// excel_read_as_csv_text.go：读取 Excel 为 CSV 文本，路由 POST /document/excel/excel_read_as_csv_text.form
// 不生成文件，直接返回 CSV 格式的文本，便于大模型或后续处理。

package table

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/xuri/excelize/v2"
)

// ExcelReadAsCSVTextReq 读取 Excel 为 CSV 文本请求
type ExcelReadAsCSVTextReq struct {
	InputFiles string `json:"input_files" widget:"name:上传Excel文件;type:files;accept:.xlsx,.xls;max_size:50MB;max_count:10" validate:"required"`
	SheetName  string `json:"sheet_name" widget:"name:指定工作表(可选);type:input;placeholder:留空则输出所有工作表"`
}

// ExcelReadAsCSVTextResp 读取 Excel 为 CSV 文本响应
type ExcelReadAsCSVTextResp struct {
	CSVText string `json:"csv_text" widget:"name:CSV文本;type:text_area"`
	Summary string `json:"summary" widget:"name:说明;type:text_area"`
}

// ExcelReadAsCSVText 入口
func ExcelReadAsCSVText(ctx *app.Context, resp response.Response) error {
	var req ExcelReadAsCSVTextReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoExcelReadAsCSVText(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoExcelReadAsCSVText 读取 xlsx 所有（或指定）工作表，输出为 CSV 文本
func DoExcelReadAsCSVText(ctx *app.Context, req *ExcelReadAsCSVTextReq) (*ExcelReadAsCSVTextResp, error) {
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
			logger.Warnf(ctx, "[excel/ReadAsCSVText] 文件 %s 无本地路径，跳过", filepath.Base(file))
			summaryParts = append(summaryParts, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		f, err := excelize.OpenFile(file)
		if err != nil {
			logger.Errorf(ctx, "[excel/ReadAsCSVText] 打开失败 %s: %v", filepath.Base(file), err)
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
			csvContent := rowsToCSVText(rows)
			if len(exportSheets) > 1 {
				outParts = append(outParts, fmt.Sprintf("## %s - %s\n\n%s", filepath.Base(file), sheetName, csvContent))
			} else {
				outParts = append(outParts, csvContent)
			}
		}
		f.Close()
		summaryParts = append(summaryParts, fmt.Sprintf("已读 %s，%d 个工作表", filepath.Base(file), len(exportSheets)))
	}

	csvText := strings.TrimSpace(strings.Join(outParts, "\n\n"))
	if csvText == "" {
		csvText = "(无数据)"
	}
	return &ExcelReadAsCSVTextResp{
		CSVText: csvText,
		Summary: strings.Join(summaryParts, "\n"),
	}, nil
}

// rowsToCSVText 将 [][]string 转为 CSV 文本（无 BOM）
func rowsToCSVText(rows [][]string) string {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.UseCRLF = false
	for _, row := range rows {
		_ = w.Write(row)
	}
	w.Flush()
	return strings.TrimRight(buf.String(), "\n")
}

var ExcelReadAsCSVTextTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "读取 Excel 为 CSV 文本",
		Desc:     `上传 xlsx 或 xls，直接返回 CSV 格式文本（不生成文件）。可指定单个工作表，不指定则输出所有工作表；多表时以「## 文件名 - 表名」分段。适合给大模型看表内容、或快速复制粘贴到别处。`,
		Tags:     []string{"Excel", "xlsx", "xls", "CSV", "读取Excel", "Excel转CSV文本", "excel转csv文本", "CSV文本", "多工作表", "不生成文件"},
		Request:  &ExcelReadAsCSVTextReq{},
		Response: &ExcelReadAsCSVTextResp{},
	},
}

func init() {
	packageContext.POST("excel_read_as_csv_text.form", ExcelReadAsCSVText, ExcelReadAsCSVTextTemplate)
}
