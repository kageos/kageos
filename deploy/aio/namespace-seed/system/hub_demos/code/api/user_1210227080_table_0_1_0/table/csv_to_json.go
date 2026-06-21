package table

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type CSVToJSONReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传 CSV/TSV 文件;type:files;accept:.csv,.tsv,text/csv,text/tab-separated-values,text/*,*/*;max_size:500MB;max_count:20" validate:"required"`
	Delimiter      string `json:"delimiter" widget:"name:分隔符;type:select;options:自动,逗号,制表符,分号,竖线;options_colors:409EFF,909399,67C23A,E6A23C,909399;render_default:自动" validate:"required,oneof=自动 逗号 制表符 分号 竖线"`
	HasHeader      bool   `json:"has_header" widget:"name:首行是表头;type:switch;render_default:true"`
	OutputMode     string `json:"output_mode" widget:"name:输出模式;type:select;options:对象数组,二维数组;options_colors:409EFF,67C23A;render_default:对象数组" validate:"required,oneof=对象数组 二维数组"`
	MaxRows        int    `json:"max_rows" widget:"name:最多转换数据行;type:integer;min:0;max:200000;render_default:50000;placeholder:0 表示使用默认上限" validate:"min=0,max=200000"`
	OutputFileName string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，仅单文件时生效，例如 data.json"`
}

type CSVToJSONResp struct {
	OutputFiles string `json:"output_files" widget:"name:JSON 文件;type:files"`
	PreviewText string `json:"preview_text" widget:"name:JSON 预览;type:text_area"`
	ConvertInfo string `json:"convert_info" widget:"name:转换信息;type:text_area"`
}

func CSVToJSON(ctx *app.Context, resp response.Response) error {
	var req CSVToJSONReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoCSVToJSON(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoCSVToJSON(ctx *app.Context, req *CSVToJSONReq) (*CSVToJSONResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	maxRows := req.MaxRows
	if maxRows <= 0 {
		maxRows = 50000
	}
	outputDir := fs.GetTraceOutputDir()
	var outputPaths []string
	var previews []string
	var infos []string
	for _, path := range inputFiles {
		if path == "" {
			continue
		}
		outputName := csvOutputName(filepath.Base(path), req.OutputFileName, "_json", "json", len(inputFiles) == 1)
		outputPath := filepath.Join(outputDir, outputName)
		preview, count, truncated, err := convertCSVToJSON(path, outputPath, req.Delimiter, req.HasHeader, req.OutputMode, maxRows)
		if err != nil {
			infos = append(infos, fmt.Sprintf("失败 %s: %v", filepath.Base(path), err))
			continue
		}
		outputPaths = append(outputPaths, outputPath)
		previews = append(previews, fmt.Sprintf("## %s\n%s", filepath.Base(path), preview))
		info := fmt.Sprintf("成功 %s -> %s，转换 %d 行", filepath.Base(path), outputName, count)
		if truncated {
			info += "（达到上限，已截断）"
		}
		infos = append(infos, info)
	}
	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功转换的 CSV 文件\n%s", strings.Join(infos, "\n"))
	}
	return &CSVToJSONResp{
		OutputFiles: fs.ResponseFiles(outputPaths),
		PreviewText: strings.Join(previews, "\n\n"),
		ConvertInfo: fmt.Sprintf("CSV 转 JSON 完成\n输出模式: %s\n输出文件数: %d\n\n详情:\n%s",
			req.OutputMode, len(outputPaths), strings.Join(infos, "\n")),
	}, nil
}

func convertCSVToJSON(inputPath, outputPath, delimiterOption string, hasHeader bool, outputMode string, maxRows int) (string, int, bool, error) {
	delimiter := delimiterRune(delimiterOption, readSample(inputPath, 65536))
	rows, truncated, err := readCSVRows(inputPath, delimiter, maxRows+1)
	if err != nil {
		return "", 0, false, err
	}
	if len(rows) == 0 {
		return "", 0, false, fmt.Errorf("文件为空或没有可读行")
	}
	if truncated && len(rows) > maxRows {
		rows = rows[:maxRows]
	}

	var payload interface{}
	convertedRows := len(rows)
	if outputMode == "二维数组" {
		payload = rows
	} else {
		headers := rows[0]
		dataRows := rows
		if hasHeader {
			headers = normalizeHeaders(rows[0])
			dataRows = rows[1:]
		} else {
			headers = generatedHeaders(maxRecordWidth(rows))
		}
		objects := make([]map[string]string, 0, len(dataRows))
		for _, row := range dataRows {
			obj := make(map[string]string, len(headers))
			for i, header := range headers {
				value := ""
				if i < len(row) {
					value = row[i]
				}
				obj[header] = value
			}
			objects = append(objects, obj)
		}
		payload = objects
		convertedRows = len(objects)
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", 0, false, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return "", 0, false, err
	}
	preview := string(data)
	if len([]rune(preview)) > 80000 {
		preview = string([]rune(preview)[:80000]) + "\n...（预览已截断，完整内容见输出文件）"
	}
	return preview, convertedRows, truncated, nil
}

var CSVToJSONTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "CSV 转 JSON",
		Desc:     `把 CSV/TSV 转成 JSON 对象数组或二维数组，支持自动识别分隔符、首行表头和转换行数上限。适合把上传表格转换为代码更容易处理的结构化输入。`,
		Tags:     []string{"CSV", "TSV", "JSON", "转换", "结构化数据"},
		Request:  &CSVToJSONReq{},
		Response: &CSVToJSONResp{},
	},
}

func init() {
	packageContext.POST("to_json.form", CSVToJSON, CSVToJSONTemplate)
}
