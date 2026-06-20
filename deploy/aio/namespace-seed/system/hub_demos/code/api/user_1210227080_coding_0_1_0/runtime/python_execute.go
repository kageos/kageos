//<文件名>python_execute.go</文件名>

package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	pythonRuntime "github.com/kageos/kageos/sdk/agent-app/runtime/python"
)

// FlexibleJSONObject 兼容表单 text_area 提交的 JSON 字符串和工具调用提交的 JSON 对象。
// 底层保持 string，避免 text_area schema 与 Go 类型漂移。
type FlexibleJSONObject string

func (a *FlexibleJSONObject) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*a = ""
		return nil
	}

	if data[0] == '"' {
		var raw string
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			*a = ""
			return nil
		}
		data = []byte(raw)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("args 必须是 JSON 对象或 JSON 对象字符串: %w", err)
	}
	if parsed == nil {
		parsed = make(map[string]interface{})
	}
	*a = FlexibleJSONObject(string(data))
	return nil
}

func (a FlexibleJSONObject) Map() map[string]interface{} {
	raw := strings.TrimSpace(string(a))
	if raw == "" {
		return map[string]interface{}{}
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil || parsed == nil {
		return map[string]interface{}{}
	}
	return parsed
}

// PythonExecuteReq Python代码执行请求结构体
type PythonExecuteReq struct {
	// 框架标签：widget:"type:text_area;placeholder:请输入Python代码..." - 代码输入框
	PythonCode string `json:"python_code" widget:"name:Python代码;type:text_area;placeholder:必须定义 def kageos_entry(args, output_dir): ...；统一 4 空格缩进；返回 {\"data\": ..., \"output_files\": [...]}；输出文件必须写入 output_dir；不要手动覆盖 matplotlib 中文字体" validate:"required"`

	// 结构化参数对象（推荐）
	Args FlexibleJSONObject `json:"args" widget:"name:参数对象;type:text_area;placeholder:仅支持对象，例如: {\"message\":\"hello\",\"file_name\":\"result.txt\"}（推荐）"`

	// 框架标签：widget:"type:text_area;placeholder:JSON格式的参数..." - JSON格式的参数（可选）
	ArgsJSON string `json:"args_json" widget:"name:参数(JSON格式);type:text_area;placeholder:仅支持 JSON 对象，例如: {\"message\":\"hello\",\"file_name\":\"result.txt\"}（可选，优先级高于参数对象）"`

	// 输入文件引用，格式 bucket/object_key，多文件逗号分隔。执行前会下载并注入为 args.input_files 本地路径列表。
	InputFiles string `json:"input_files" widget:"name:输入文件;type:files;max_count:100"`

	// 框架标签：widget:"type:input;placeholder:例如: pandas,numpy" - Python包列表（可选）
	Packages string `json:"packages" widget:"name:Python包列表;type:input;placeholder:例如: pandas,openpyxl,matplotlib（多个包用逗号分隔，可选）"`

	// 框架标签：widget:"type:integer;placeholder:300" - 超时时间（秒）
	TimeoutSeconds int `json:"timeout_seconds" widget:"name:超时时间(秒);type:integer;render_default:300;placeholder:请输入超时时间（秒）"`

	// 是否收集并发布 output_files（默认开启）
	CollectOutputFiles bool `json:"collect_output_files" widget:"name:收集输出文件;type:switch;render_default:true"`
}

// PythonExecuteResp Python代码执行响应结构体
type PythonExecuteResp struct {
	// 执行结果输出
	Output string `json:"output" widget:"name:执行结果;type:text_area"`

	// 执行状态
	Status string `json:"status" widget:"name:执行状态;type:text"`

	// JSON解析结果（如果输出是JSON格式）
	JSONResult string `json:"json_result" data:"format:json" widget:"name:JSON解析结果;type:text_area"`

	// 输出文件
	OutputFiles string `json:"output_files,omitempty" widget:"name:输出文件;type:files"`
}

// PythonExecute Python代码执行函数
func PythonExecute(ctx *app.Context, resp response.Response) error {
	var req PythonExecuteReq
	err := ctx.ShouldBindValidate(&req)
	if err != nil {
		return err
	}

	// 1. 解析参数（固定为对象）
	requestData := make(map[string]interface{})
	for k, v := range req.Args.Map() {
		requestData[k] = v
	}
	if strings.TrimSpace(req.ArgsJSON) != "" {
		var parsed interface{}
		err := json.Unmarshal([]byte(req.ArgsJSON), &parsed)
		if err != nil {
			logger.Warnf(ctx, "[PythonExecute] 解析参数JSON失败: %v", err)
		} else if parsedMap, ok := parsed.(map[string]interface{}); ok {
			requestData = parsedMap
		} else {
			logger.Warnf(ctx, "[PythonExecute] args_json 不是对象，忽略")
		}
	}

	inputFileRefs := strings.TrimSpace(req.InputFiles)
	if inputFileRefs == "" {
		inputFileRefs = inputFileRefsFromArgs(requestData)
	}
	var downloadedInputFiles []string
	if inputFileRefs != "" {
		fs := ctx.GetFS()
		downloadedInputFiles = fs.DownloadFiles(inputFileRefs)
		defer fs.RemoveFiles(downloadedInputFiles)
		if len(downloadedInputFiles) == 0 {
			return resp.Form(&PythonExecuteResp{
				Output:     "输入文件下载失败：未能把文件引用下载到容器本地路径。请确认 input_files 为 bucket/object_key 字符串，且文件已上传完成。",
				Status:     "失败",
				JSONResult: "（无结构化结果）",
			}).Build()
		}
		requestData["input_files"] = downloadedInputFiles
	}

	// 2. 解析包列表（如果有）
	var packages []string
	if req.Packages != "" {
		// 按逗号分割包名
		packageList := splitPackages(req.Packages)
		packages = append(packages, packageList...)
	}

	// 3. 设置超时时间（默认5分钟）
	timeout := 5 * time.Minute
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}

	// 4. 创建Python执行器
	executor := pythonRuntime.NewExecutor(normalizePythonCode(req.PythonCode)).
		WithRequest(requestData).
		WithTimeout(timeout)
	defer func() {
		if closeErr := executor.Close(); closeErr != nil {
			logger.Warnf(ctx, "[PythonExecute] executor.Close: %v", closeErr)
		}
	}()

	// 添加包列表（如果有）
	if len(packages) > 0 {
		executor = executor.WithPackages(packages...)
	}

	// 5. 执行Python代码
	execResult, err := executor.ExecuteResult(ctx)

	outputStr := ""
	status := "成功"
	jsonResult := ""
	var outputFiles string

	if execResult != nil {
		outputStr = execResult.CombinedOutput()
		jsonResult = execResult.FormattedData()
	}

	if err != nil {
		logger.Errorf(ctx, "[PythonExecute] 执行Python代码失败: %v, output: %s", err, outputStr)
		status = "失败"
		if strings.TrimSpace(outputStr) != "" {
			outputStr = fmt.Sprintf("执行错误: %v\n\n输出:\n%s", err, outputStr)
		} else {
			outputStr = fmt.Sprintf("执行错误: %v", err)
		}
	} else if execResult != nil {
		logger.Infof(ctx, "[PythonExecute] 执行Python代码成功")
		collectOutputFiles := req.CollectOutputFiles || len(execResult.OutputFiles) > 0
		if collectOutputFiles {
			paths, pathErr := execResult.OutputFilePaths()
			if pathErr != nil {
				logger.Errorf(ctx, "[PythonExecute] 校验 output_files 失败: %v", pathErr)
				status = "失败"
				if outputStr != "" {
					outputStr = outputStr + "\n\n输出文件错误:\n" + pathErr.Error()
				} else {
					outputStr = "输出文件错误: " + pathErr.Error()
				}
			} else if len(paths) > 0 {
				outputFiles = ctx.GetFS().ResponseFiles(paths)
			}
		}
	}

	// 6. 如果输出为空，显示提示信息
	if outputStr == "" {
		outputStr = "（脚本执行完成，无输出）"
	}
	if jsonResult == "" {
		jsonResult = "（无结构化结果）"
	}

	// 7. 构建响应
	return resp.Form(&PythonExecuteResp{
		Output:      outputStr,
		Status:      status,
		JSONResult:  jsonResult,
		OutputFiles: outputFiles,
	}).Build()
}

// splitPackages 分割包名字符串
func splitPackages(packagesStr string) []string {
	var packages []string
	// 按逗号分割
	parts := strings.Split(packagesStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			packages = append(packages, part)
		}
	}
	return packages
}

func inputFileRefsFromArgs(args map[string]interface{}) string {
	for _, key := range []string{"input_files", "files"} {
		if refs := normalizeInputFileRefsValue(args[key]); refs != "" {
			return refs
		}
	}
	return ""
}

func normalizeInputFileRefsValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []string:
		return strings.Join(cleanStringParts(v), ",")
	case []interface{}:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(cleanStringParts(parts), ",")
	default:
		return ""
	}
}

func cleanStringParts(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func normalizePythonCode(code string) string {
	return strings.TrimPrefix(code, "\ufeff")
}

// PythonExecuteTemplate Python代码执行配置
var PythonExecuteTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Python代码执行",
		Desc:     `执行 Python 脚本代码。脚本必须定义固定入口 def kageos_entry(args, output_dir): ...，统一使用 4 空格缩进，不要混用 Tab；返回 dict，支持 data/output_files/warnings 三个字段；args 仅支持对象；若传入 input_files，系统会先下载为容器本地路径并注入 args.input_files；若返回 output_files，文件必须写入 output_dir，系统会自动上传并在工作台展示。生成图表时不要手动覆盖 matplotlib 中文字体，镜像已配置中文字体；若出现 SyntaxError/IndentationError，应重写一份更短、更扁平的完整脚本，不要局部修补旧长脚本。`,
		Tags:     []string{"脚本执行", "Python", "工具"},
		Request:  &PythonExecuteReq{},
		Response: &PythonExecuteResp{},
	},
}

func init() {
	// 注册Form函数 - Python代码执行
	packageContext.POST("python.form", PythonExecute, PythonExecuteTemplate)
}
