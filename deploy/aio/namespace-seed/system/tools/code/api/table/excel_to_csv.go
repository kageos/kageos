// excel_to_csv.go：Excel 转 CSV（Go 实现），路由 POST /document/excel/excel_to_csv.form
// 使用 excelize 读取 xlsx，所有 sheet 分别导出为 CSV，支持 UTF-8 BOM。

package table

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/xuri/excelize/v2"
)

// UTF-8 BOM，用于 utf-8-sig，便于 Excel 正确识别中文
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// ExcelToCSVReq Excel 转 CSV 请求
type ExcelToCSVReq struct {
	InputFiles string `json:"input_files" widget:"name:上传Excel文件;type:files;accept:.xlsx,.xls;max_size:50MB;max_count:10" validate:"required"`
	Encoding   string `json:"encoding" widget:"name:输出编码;type:select;render_default:utf-8-sig;options:utf-8-sig,utf-8"`
}

// ExcelToCSVResp Excel 转 CSV 响应
type ExcelToCSVResp struct {
	OutputFiles string `json:"output_files" widget:"name:转换后的CSV文件;type:files"`
	ConvertInfo string `json:"convert_info" widget:"name:转换信息;type:text_area"`
}

// ExcelToCSV 入口
func ExcelToCSV(ctx *app.Context, resp response.Response) error {
	var req ExcelToCSVReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoExcelToCSV(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoExcelToCSV 使用 excelize 读取 xlsx，一个 Excel 文件导出一个 CSV（多 sheet 合并，首列「工作表」区分来源），支持 BOM
func DoExcelToCSV(ctx *app.Context, req *ExcelToCSVReq) (*ExcelToCSVResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	encoding := strings.TrimSpace(strings.ToLower(req.Encoding))
	if encoding == "" {
		encoding = "utf-8-sig"
	}
	if encoding != "utf-8-sig" && encoding != "utf-8" {
		encoding = "utf-8-sig"
	}
	useBOM := encoding == "utf-8-sig"

	outputDir := fs.GetTraceOutputDir()
	var outputPaths []string
	var convertInfos []string

	for _, file := range files {
		if file == "" {
			logger.Warnf(ctx, "[excel/ExcelToCSV] 文件 %s 无本地路径，跳过", filepath.Base(file))
			convertInfos = append(convertInfos, fmt.Sprintf("跳过 %s: 无本地路径", filepath.Base(file)))
			continue
		}

		f, err := excelize.OpenFile(file)
		if err != nil {
			logger.Errorf(ctx, "[excel/ExcelToCSV] 打开失败 %s: %v", filepath.Base(file), err)
			convertInfos = append(convertInfos, fmt.Sprintf("失败 %s: %v", filepath.Base(file), err))
			continue
		}

		baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		baseName = sanitizeFileName(baseName)
		sheetList := f.GetSheetList()
		if len(sheetList) == 0 {
			f.Close()
			convertInfos = append(convertInfos, fmt.Sprintf("跳过 %s: 无工作表", filepath.Base(file)))
			continue
		}

		merged := mergeSheetsToCSVRows(f, sheetList)
		csvPath := filepath.Join(outputDir, baseName+".csv")
		if err := writeCSV(csvPath, merged, useBOM); err != nil {
			f.Close()
			return nil, fmt.Errorf("写入 CSV 失败: %w", err)
		}
		f.Close()
		outputPaths = append(outputPaths, csvPath)
		convertInfos = append(convertInfos, fmt.Sprintf("成功 %s -> 1 个 CSV（%d 个工作表已合并）", filepath.Base(file), len(sheetList)))
	}

	var outputFiles string
	if len(outputPaths) > 0 {
		outputFiles = fs.ResponseFiles(outputPaths)
	}

	return &ExcelToCSVResp{
		OutputFiles: outputFiles,
		ConvertInfo: strings.Join(convertInfos, "\n"),
	}, nil
}

// mergeSheetsToCSVRows 将多 sheet 合并为一张表：首列「工作表」，后面为各 sheet 数据；列数按所有 sheet 最大列数对齐
func mergeSheetsToCSVRows(f *excelize.File, sheetList []string) [][]string {
	const sheetCol = "工作表"
	maxCols := 0
	for _, sheetName := range sheetList {
		rows, _ := f.GetRows(sheetName)
		for _, r := range rows {
			if len(r) > maxCols {
				maxCols = len(r)
			}
		}
	}
	if maxCols == 0 {
		return [][]string{{sheetCol}}
	}
	dataCols := maxCols

	var allRows [][]string
	for si, sheetName := range sheetList {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			continue
		}
		for ri, row := range rows {
			padded := padRow(row, dataCols)
			line := make([]string, 0, dataCols+1)
			line = append(line, sheetName)
			line = append(line, padded...)
			if si == 0 && ri == 0 {
				allRows = append(allRows, append([]string{sheetCol}, padded...))
			} else {
				allRows = append(allRows, line)
			}
		}
	}
	if len(allRows) == 0 {
		return [][]string{{sheetCol}}
	}
	return allRows
}

func padRow(row []string, width int) []string {
	if width <= 0 {
		return row
	}
	if len(row) >= width {
		return row[:width]
	}
	out := make([]string, width)
	copy(out, row)
	return out
}

// writeCSV 将 [][]string 写入 CSV 文件，可选写入 UTF-8 BOM
func writeCSV(path string, rows [][]string, withBOM bool) error {
	fd, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fd.Close()

	if withBOM {
		if _, err := fd.Write(utf8BOM); err != nil {
			return err
		}
	}

	w := csv.NewWriter(fd)
	w.UseCRLF = false
	for _, row := range rows {
		_ = w.Write(row)
	}
	w.Flush()
	return w.Error()
}

// sanitizeFileName 去掉或替换文件名非法字符
func sanitizeFileName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			continue
		default:
			if utf8.ValidRune(r) {
				b.WriteRune(r)
			}
		}
	}
	return strings.TrimSpace(b.String())
}

var ExcelToCSVTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Excel 转 CSV",
		Desc:     `上传 xlsx 或 xls，一个 Excel 导出一个 CSV；多工作表会合并到同一 CSV，首列「工作表」区分来源，与工作簿等价。支持 UTF-8 带 BOM（utf-8-sig）。适合表格转纯文本、导入数据库或做数据分析。`,
		Tags:     []string{"Excel", "xlsx", "xls", "CSV", "Excel转CSV", "excel转CSV", "表格转CSV", "导出CSV", "多工作表", "UTF-8", "BOM", "中文"},
		Request:  &ExcelToCSVReq{},
		Response: &ExcelToCSVResp{},
	},
}

func init() {
	packageContext.POST("excel_to_csv.form", ExcelToCSV, ExcelToCSVTemplate)
}
