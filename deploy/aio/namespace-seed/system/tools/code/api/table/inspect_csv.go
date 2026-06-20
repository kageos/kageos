package table

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

type InspectCSVReq struct {
	InputFiles   string `json:"input_files" widget:"name:上传 CSV/TSV 文件;type:files;accept:.csv,.tsv,text/csv,text/tab-separated-values,text/*,*/*;max_size:500MB;max_count:20" validate:"required"`
	Delimiter    string `json:"delimiter" widget:"name:分隔符;type:select;options:自动,逗号,制表符,分号,竖线;options_colors:409EFF,909399,67C23A,E6A23C,909399;render_default:自动" validate:"required,oneof=自动 逗号 制表符 分号 竖线"`
	HasHeader    bool   `json:"has_header" widget:"name:首行是表头;type:switch;render_default:true"`
	MaxRows      int    `json:"max_rows" widget:"name:最多采样行数;type:integer;min:10;max:100000;render_default:1000" validate:"min=0,max=100000"`
	OutputReport bool   `json:"output_report" widget:"name:输出体检报告文件;type:switch;render_default:true"`
}

type InspectCSVResp struct {
	ReportText string `json:"report_text" widget:"name:CSV 体检报告;type:text_area"`
	OutputFile string `json:"output_file" widget:"name:体检报告;type:files"`
	Summary    string `json:"summary" widget:"name:处理说明;type:text_area"`
}

func InspectCSV(ctx *app.Context, resp response.Response) error {
	var req InspectCSVReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoInspectCSV(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoInspectCSV(ctx *app.Context, req *InspectCSVReq) (*InspectCSVResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	maxRows := req.MaxRows
	if maxRows <= 0 {
		maxRows = 1000
	}

	var blocks []string
	var summaries []string
	for _, path := range inputFiles {
		if path == "" {
			continue
		}
		block, summary, err := inspectOneCSV(path, req.Delimiter, req.HasHeader, maxRows)
		if err != nil {
			summaries = append(summaries, fmt.Sprintf("失败 %s: %v", filepath.Base(path), err))
			continue
		}
		blocks = append(blocks, block)
		summaries = append(summaries, summary)
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("没有成功读取的 CSV 文件\n%s", strings.Join(summaries, "\n"))
	}

	report := strings.TrimSpace(strings.Join(blocks, "\n\n"))
	var outputFile string
	if req.OutputReport {
		outputPath := filepath.Join(fs.GetTraceOutputDir(), "csv_inspection_report.md")
		if err := os.WriteFile(outputPath, []byte(report), 0644); err != nil {
			return nil, fmt.Errorf("写入体检报告失败: %w", err)
		}
		outputFile = fs.ResponseFiles([]string{outputPath})
	}
	return &InspectCSVResp{
		ReportText: report,
		OutputFile: outputFile,
		Summary:    "CSV 体检完成\n" + strings.Join(summaries, "\n"),
	}, nil
}

func inspectOneCSV(path, delimiterOption string, hasHeader bool, maxRows int) (string, string, error) {
	sample := readSample(path, 65536)
	delimiter := delimiterRune(delimiterOption, sample)
	rows, truncated, err := readCSVRows(path, delimiter, maxRows)
	if err != nil {
		return "", "", err
	}
	if len(rows) == 0 {
		return "", "", fmt.Errorf("文件为空或没有可读行")
	}

	headers := rows[0]
	dataRows := rows
	if hasHeader {
		headers = normalizeHeaders(rows[0])
		dataRows = rows[1:]
	} else {
		headers = generatedHeaders(maxRecordWidth(rows))
	}
	types, fillCounts := inferColumns(headers, dataRows)
	stat, _ := os.Stat(path)

	var b strings.Builder
	b.WriteString("## ")
	b.WriteString(filepath.Base(path))
	b.WriteString("\n")
	if stat != nil {
		b.WriteString(fmt.Sprintf("- 文件大小: %d bytes\n", stat.Size()))
	}
	b.WriteString(fmt.Sprintf("- 分隔符: %s\n", delimiterLabel(delimiter)))
	b.WriteString(fmt.Sprintf("- 采样行数: %d\n", len(rows)))
	b.WriteString(fmt.Sprintf("- 数据行数(采样内): %d\n", len(dataRows)))
	b.WriteString(fmt.Sprintf("- 列数: %d\n", len(headers)))
	if truncated {
		b.WriteString("- 注意: 文件行数超过采样上限，报告基于采样数据\n")
	}
	b.WriteString("\n### 字段\n")
	for i, header := range headers {
		fill := 0
		if i < len(fillCounts) {
			fill = fillCounts[i]
		}
		typ := "empty"
		if i < len(types) && types[i] != "" {
			typ = types[i]
		}
		b.WriteString(fmt.Sprintf("- `%s`: %s，非空 %d/%d\n", header, typ, fill, len(dataRows)))
	}
	if len(dataRows) > 0 {
		b.WriteString("\n### 数据预览\n")
		b.WriteString(markdownTable(headers, dataRows, 20))
	}
	return b.String(), fmt.Sprintf("成功 %s: %d 列，采样 %d 行", filepath.Base(path), len(headers), len(rows)), nil
}

func maxRecordWidth(rows [][]string) int {
	width := 0
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	return width
}

func inferColumns(headers []string, rows [][]string) ([]string, []int) {
	types := make([]string, len(headers))
	fillCounts := make([]int, len(headers))
	for _, row := range rows {
		for i := range headers {
			value := ""
			if i < len(row) {
				value = row[i]
			}
			if strings.TrimSpace(value) != "" {
				fillCounts[i]++
			}
			types[i] = mergeCellType(types[i], inferCellType(value))
		}
	}
	return types, fillCounts
}

var InspectCSVTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "CSV 结构体检",
		Desc:     `读取 CSV/TSV 文件，自动或手动识别分隔符，输出列名、采样行数、字段类型、非空数量和前 20 行预览。适合导入数据前快速判断结构质量。`,
		Tags:     []string{"CSV", "TSV", "表格", "体检", "字段类型", "数据预览"},
		Request:  &InspectCSVReq{},
		Response: &InspectCSVResp{},
	},
}

func init() {
	packageContext.POST("inspect.form", InspectCSV, InspectCSVTemplate)
}
