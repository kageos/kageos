package json

import (
	"bytes"
	"encoding/csv"
	stdjson "encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type QueryTransformReq struct {
	InputJSON      string `json:"input_json" widget:"name:JSON 文本;type:text_area;placeholder:可直接粘贴 JSON；如果上传了文件，此项可留空"`
	InputFiles     string `json:"input_files" widget:"name:上传 JSON 文件;type:files;accept:.json,application/json,text/*,*/*;max_size:200MB;max_count:20"`
	Operation      string `json:"operation" widget:"name:操作;type:select;options:格式化,压缩,jq查询,转CSV;options_colors:409EFF,909399,67C23A,E6A23C;render_default:格式化" validate:"required,oneof=格式化 压缩 jq查询 转CSV"`
	JQFilter       string `json:"jq_filter" widget:"name:jq 查询表达式;type:input;placeholder:jq查询时使用，例如 .items[] | {id,name}"`
	OutputFileName string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，仅单输入时生效，例如 result.json"`
}

type QueryTransformResp struct {
	OutputText  string `json:"output_text" widget:"name:输出预览;type:text_area"`
	OutputFiles string `json:"output_files" widget:"name:输出文件;type:files"`
	Summary     string `json:"summary" widget:"name:处理说明;type:text_area"`
}

func QueryTransform(ctx *app.Context, resp response.Response) error {
	var req QueryTransformReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoQueryTransform(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoQueryTransform(ctx *app.Context, req *QueryTransformReq) (*QueryTransformResp, error) {
	fs := ctx.GetFS()
	inputs, downloaded, err := loadJSONInputs(ctx, req.InputJSON, req.InputFiles)
	if err != nil {
		return nil, err
	}
	defer fs.RemoveFiles(downloaded)
	if len(inputs) == 0 {
		return nil, fmt.Errorf("请提供 JSON 文本或上传 JSON 文件")
	}

	outputDir := fs.GetTraceOutputDir()
	var outputPaths []string
	var previews []string
	var infos []string
	for i, input := range inputs {
		outputExt := "json"
		var output []byte
		switch req.Operation {
		case "压缩":
			output, err = compactJSON(input.Data)
		case "jq查询":
			output, err = runJQQuery(ctx, input.Data, req.JQFilter, outputDir, i)
		case "转CSV":
			outputExt = "csv"
			output, err = jsonToCSV(input.Data)
		default:
			output, err = prettyJSON(input.Data)
		}
		if err != nil {
			infos = append(infos, fmt.Sprintf("失败 %s: %v", input.Name, err))
			continue
		}

		outputName := jsonOutputName(input.Name, req.OutputFileName, req.Operation, outputExt, len(inputs) == 1)
		outputPath := filepath.Join(outputDir, outputName)
		if err := os.WriteFile(outputPath, output, 0644); err != nil {
			return nil, fmt.Errorf("写入输出文件失败: %w", err)
		}
		outputPaths = append(outputPaths, outputPath)
		previews = append(previews, fmt.Sprintf("## %s\n%s", input.Name, truncateRunes(string(output), 80000)))
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", input.Name, outputName))
	}
	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功处理的 JSON\n%s", strings.Join(infos, "\n"))
	}
	return &QueryTransformResp{
		OutputText:  strings.Join(previews, "\n\n"),
		OutputFiles: fs.ResponseFiles(outputPaths),
		Summary:     fmt.Sprintf("JSON 处理完成\n操作: %s\n输入数: %d\n输出文件数: %d\n\n详情:\n%s", req.Operation, len(inputs), len(outputPaths), strings.Join(infos, "\n")),
	}, nil
}

type jsonInput struct {
	Name string
	Data []byte
}

func loadJSONInputs(ctx *app.Context, text string, fileRefs string) ([]jsonInput, []string, error) {
	var inputs []jsonInput
	var downloaded []string
	if strings.TrimSpace(fileRefs) != "" {
		fs := ctx.GetFS()
		downloaded = fs.DownloadFiles(fileRefs)
		for _, path := range downloaded {
			if path == "" {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, downloaded, fmt.Errorf("读取 JSON 文件失败 %s: %w", filepath.Base(path), err)
			}
			inputs = append(inputs, jsonInput{Name: filepath.Base(path), Data: data})
		}
	}
	if strings.TrimSpace(text) != "" {
		inputs = append(inputs, jsonInput{Name: "input.json", Data: []byte(text)})
	}
	return inputs, downloaded, nil
}

func prettyJSON(data []byte) ([]byte, error) {
	var out bytes.Buffer
	if err := stdjson.Indent(&out, bytes.TrimSpace(data), "", "  "); err != nil {
		return nil, fmt.Errorf("JSON 校验失败: %w", err)
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

func compactJSON(data []byte) ([]byte, error) {
	var out bytes.Buffer
	if err := stdjson.Compact(&out, bytes.TrimSpace(data)); err != nil {
		return nil, fmt.Errorf("JSON 校验失败: %w", err)
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

func runJQQuery(ctx *app.Context, data []byte, filter string, outputDir string, index int) ([]byte, error) {
	if _, err := exec.LookPath("jq"); err != nil {
		return nil, fmt.Errorf("未找到 jq，请确认运行环境已安装 jq")
	}
	filter = strings.TrimSpace(filter)
	if filter == "" {
		filter = "."
	}
	inputPath := filepath.Join(outputDir, fmt.Sprintf("jq_input_%03d.json", index+1))
	if err := os.WriteFile(inputPath, data, 0644); err != nil {
		return nil, err
	}
	defer os.Remove(inputPath)
	out, err := exec.Command("jq", filter, inputPath).CombinedOutput()
	if err != nil {
		logger.Warnf(ctx, "[JSON/QueryTransform] jq 执行失败: %v, output: %s", err, string(out))
		return nil, fmt.Errorf("jq 查询失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func jsonToCSV(data []byte) ([]byte, error) {
	var raw interface{}
	if err := stdjson.Unmarshal(bytes.TrimSpace(data), &raw); err != nil {
		return nil, fmt.Errorf("JSON 校验失败: %w", err)
	}
	var rows []map[string]interface{}
	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			if row, ok := item.(map[string]interface{}); ok {
				rows = append(rows, row)
			}
		}
	case map[string]interface{}:
		rows = append(rows, v)
	default:
		return nil, fmt.Errorf("转 CSV 需要 JSON 对象或对象数组")
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("没有可转换的对象行")
	}
	colSet := map[string]bool{}
	for _, row := range rows {
		for key := range row {
			colSet[key] = true
		}
	}
	columns := make([]string, 0, len(colSet))
	for key := range colSet {
		columns = append(columns, key)
	}
	sort.Strings(columns)
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(columns)
	for _, row := range rows {
		record := make([]string, 0, len(columns))
		for _, col := range columns {
			record = append(record, jsonCellString(row[col]))
		}
		_ = w.Write(record)
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func jsonCellString(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case float64, bool:
		return fmt.Sprintf("%v", v)
	default:
		b, _ := stdjson.Marshal(v)
		return string(b)
	}
}

func jsonOutputName(inputName, customName, operation, ext string, single bool) string {
	if single && strings.TrimSpace(customName) != "" {
		return ensureExt(sanitizeBase(customName, "result"), ext)
	}
	base := sanitizeBase(inputName, "result")
	suffix := map[string]string{"格式化": "_pretty", "压缩": "_compact", "jq查询": "_jq", "转CSV": "_csv"}[operation]
	if suffix == "" {
		suffix = "_out"
	}
	return base + suffix + "." + ext
}

func sanitizeBase(name, fallback string) string {
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

func ensureExt(base, ext string) string {
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return base + "." + strings.TrimPrefix(ext, ".")
}

func truncateRunes(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max]) + "\n...（输出过长，已截断；完整内容见输出文件）"
}

var QueryTransformTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "JSON 查询转换",
		Desc:     `校验、格式化、压缩 JSON，支持 jq 表达式查询和 JSON 对象数组转 CSV。适合处理接口返回、配置文件、日志片段和临时结构化数据。`,
		Tags:     []string{"JSON", "jq", "CSV", "格式化", "压缩", "结构化数据"},
		Request:  &QueryTransformReq{},
		Response: &QueryTransformResp{},
	},
}

func init() {
	packageContext.POST("query_transform.form", QueryTransform, QueryTransformTemplate)
}
