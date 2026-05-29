package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/logger"
)

// runPythonPreinstallDoc 与 deploy/base/images/app-base/Dockerfile 中 apt/python3-* 与 pip3 install 预装保持一致；改镜像时请同步更新本文案
const runPythonPreinstallDoc = `**生产镜像已预装、可直接 import 的第三方库（对应 deploy/base/images/app-base/Dockerfile）：**
- 数据与图表：pandas、numpy、scipy、matplotlib、seaborn、plotly、pyecharts
- 数据展示与日期：tabulate、arrow、dateutil（python-dateutil）
- 网络与网页解析：requests、aiohttp、bs4（beautifulsoup4）、lxml
- 在线媒体：yt_dlp（yt-dlp CLI 也可直接调用）
- 表格与 Office：openpyxl、xlsxwriter、xlrd、xlwt、pptx（python-pptx）
- 图像、OCR 与码图：PIL（Pillow，如 from PIL import Image）、pytesseract、qrcode、barcode（python-barcode）
- 文档与 PDF：docx（python-docx）、PyPDF2、pdfplumber、reportlab
- 中文处理：jieba、snownlp、wordcloud
- 数据库：pymysql
- 配置与安全：yaml（PyYAML）、toml、cryptography
- 另有 **Python 标准库**（json、re、collections、datetime、itertools、math、random 等）

**若 import 报错：** 优先改用上面列表或标准库；需要新依赖时请管理员更新 Dockerfile / 基础镜像 requirements.txt 并重打镜像。不可在本工具参数里指定 pip 包。
**环境差异：** 本地非 Docker 运行时以本机 python 为准，可能与镜像不一致。`

const runPythonFormPath = "/system/tools/runtime/python.form"

type RunPythonTool struct{}

type runPythonArgs struct {
	PythonCode     string                 `json:"python_code" schema_desc:"完整 Python 源码" schema_required:"true"`
	Args           map[string]interface{} `json:"args" schema_desc:"注入脚本的对象参数（推荐）"`
	InputFiles     string                 `json:"input_files" schema_desc:"可选文件引用字符串，格式 bucket/object_key，多文件用英文逗号分隔；不传时自动使用当前用户消息上传的附件；支持直接填入上一步 output_files 返回的路径，实现 output -> input 文件流转"`
	TimeoutSeconds *int                   `json:"timeout_seconds" schema_desc:"超时秒数"`
}

var runPythonToolDef = toolDefinition[runPythonArgs](
	"run_python",
	runPythonPreinstallDoc+`

**执行环境：** Python 跑在 **应用运行时容器内**（Podman 等业务容器，**不是宿主机**）。本工具调用工具库路径 **/system/tools/runtime/python.form**，由 **system/tools** 应用对应容器执行；脚本在 **临时目录** 中运行，不把工作区源码树当作工作目录。

**固定入口协议：**
- python_code **必须定义**：def kageos_entry(args, output_dir): ...
- 第一个参数 args 为传入的对象参数；第二个参数 output_dir 为受控输出目录
- 若本轮用户上传了附件，系统会在执行前自动下载到容器本地，并注入 args["input_files"]：本地文件路径列表。单文件取 args["input_files"][0]。Python 代码应直接 open 本地路径，不要 requests.get 文件引用或猜 URL；不要把文件引用数组再塞进 args["input_files"]。
- 若继续处理历史消息里的上传文件，请把 <files> 里的 refs 原样传给本工具顶层 input_files 参数；不要拆成 bucket/key 放进 args，也不要自行拼 COS URL。
- input_files 也可以直接传入上一步工具返回的 output_files 文件引用（bucket/object_key，如 kageos/system/tools/runtime/python.form/.../result.csv）。平台会像处理用户上传附件一样自动从 COS/对象存储下载到容器本地，并注入 args["input_files"] 本地路径列表；这用于多步骤流水线，避免手动下载再上传。
- 返回值 **必须是 dict**，仅允许：
  - data: JSON 可序列化结果
  - output_files: 输出文件列表，每项至少含 path
  - warnings: 警告字符串列表
- print(...) 只用于日志，不作为主结果协议

**Python 代码书写规范（非常重要）：**
- python_code 会按原文传给执行端，平台不做 BOM、控制字符、缩进的隐式修复；请直接输出干净的 UTF-8 源码，并从 def kageos_entry(args, output_dir): 开始。
- 使用 4 个空格缩进，不要使用 Tab；不要混用空格和 Tab。
- 不要把 ANSI 颜色控制符、终端转义字符或 NUL 等不可见控制字符写进 python_code；这类字符会导致 SyntaxError 或 IndentationError。
- 优先生成短脚本、少嵌套脚本。Excel/CSV 分析优先用 pandas；只有确实要精细 Excel 样式时才用 openpyxl，避免逐单元格大段样式代码。
- import 语句放在文件开头，或至少放在 kageos_entry 函数体开头；不要先使用名字再在后面 import。
- 避免长的多层嵌套块，尤其是 for 里再套 if/else、try、with。能用 pandas 向量化、groupby、assign、map、apply、to_dict('records')、zip、列表推导式解决的，就不要写多层块。
- 返回值里的 data 必须保持 JSON 可序列化：dict key 只能是字符串；不要把 tuple、Timestamp、numpy 标量、集合直接塞进 data。pandas 聚合后优先用 as_index=False 或 reset_index()，最终用 to_dict('records') 返回。
- 构造复杂返回值时，先把中间结果赋给变量，再 return；不要在 return 里塞很长的多层字典/列表字面量。
- 删除 DataFrame 列前先确认列存在；不确定时优先显式选择需要的列，而不是 drop 一组临时列。
- 输出图片、Excel、PDF 等文件时，统一写到 output_dir，再在 output_files 里声明绝对路径。
- 如果上一轮出现 SyntaxError 或 IndentationError，不要局部修补旧长脚本；请重新生成一份更短、更扁平、缩进完整的 python_code。

**输出结果：** 工具库执行端会解析 kageos_entry 的返回值；若返回里有 **output_files**，Go 侧会负责校验、上传并构造成最终 string，工作台自动展示文件组件（预览、打开、下载等能力由组件提供）。最终回复只需说明已生成/已处理，不要手写“下载文件：xxx”、Markdown 下载链接或伪 URL。

**文件流转能力（重要）：**
- output_files 返回的文件引用可以直接作为下一次 run_python 的 input_files 参数。
- 多个文件引用仍用英文逗号分隔。
- 下一步 Python 代码里不要读取 bucket/object_key 字符串本身；应读取平台注入的本地路径 args["input_files"][0] / args["input_files"][i]。
- 示例流水线：步骤1 清洗 Excel -> output_files 返回 CSV；步骤2 input_files 填该 CSV 路径并读取 pd.read_csv(args["input_files"][0]) -> 输出 XLSX；步骤3 input_files 填该 XLSX 路径 -> 生成图表图片。

**输出文件约束：**
- 只能声明写在 output_dir 里的最终文件
- 不要返回随机路径、临时缓存路径、相对路径拼猜出来的文件
- 每个输出文件项建议形如：{"path": "/abs/path/in/output_dir/report.xlsx", "name": "report.xlsx"}

**若你需要把字段、权限、命名规则固化为应用接口：** 请用 **read_doc** 读取内置示例文档 **/system/prompt/case_catalog/form/python_output**（含 PRD 与完整 Go 示例），再按文档配合 **agent-app SDK** 在用户应用内新增 Form：**pythonRuntime.NewExecutor** → **defer executor.Close()**（默认临时目录）→ Go 用 **filepath.Abs** 得到 **绝对路径**（如 GetTraceOutputDir 下文件）经请求传给 Python → Python **直接写入该路径**（如 savefig，勿用相对路径互传，Go/Python **cwd 不同**）→ 用 **OutputFilePaths + ResponseFiles** 下发附件。Go 与 Python 为**同机子进程**，非网络隔离。

**参数：** 使用 args 传对象参数；timeout_seconds 默认 120、上限 300。

返回中可能含 _model_guidance：面向你的纠错/降级说明，请优先阅读。`,
)

func (t *RunPythonTool) Definition() dto.ToolDef {
	return runPythonToolDef
}

func (t *RunPythonTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[runPythonArgs](call.Args)
	if err != nil {
		return toolResult("run_python 参数解析失败: "+err.Error(), true)
	}
	content, isError, data := runPythonTool(ctx, args, call.Files)
	return toolResultWithDataAndMetadata(content, isError, data, metadataForDisplayFileFields("output_files"))
}

// runPythonTool 调用 system/tools 的 Python 执行 Form
func runPythonTool(ctx context.Context, args runPythonArgs, attachedFiles string) (string, bool, map[string]interface{}) {
	// python_code 必须按原文透传。历史上尝试清理 BOM/控制字符/缩进会让真实错误更难定位，
	// 也可能改变 Python 源码语义；这里仅判空，不做任何隐式修复。
	code := args.PythonCode
	if strings.TrimSpace(code) == "" {
		return "run_python 需传 python_code。", true, nil
	}
	body := map[string]interface{}{
		"python_code":          code,
		"collect_output_files": true,
	}
	if len(args.Args) > 0 {
		body["args"] = args.Args
	}
	if inputFiles := resolvePythonInputFiles(args.InputFiles, attachedFiles, args.Args); inputFiles != "" {
		body["input_files"] = inputFiles
	}
	timeoutSec := 120
	if args.TimeoutSeconds != nil && *args.TimeoutSeconds > 0 {
		timeoutSec = *args.TimeoutSeconds
	}
	if timeoutSec > 300 {
		timeoutSec = 300
	}
	body["timeout_seconds"] = timeoutSec

	result, err := apicall.FormSubmit(ctx, runPythonFormPath, body)
	if err != nil {
		logger.Errorf(ctx, "[RunPython] FormSubmit 失败: %v", err)
		return "run_python 调用失败: " + err.Error() + "\n\n【给模型】可检查 python_code 是否过长、args 是否为合法对象；网络或权限问题可稍后重试。", true, nil
	}
	out := make(map[string]interface{}, len(result)+1)
	for k, v := range result {
		out[k] = v
	}
	if g := buildPythonModelGuidance(result); g != "" {
		out["_model_guidance"] = g
	}
	content, isError := formatJSONResult(out)
	return content, isError, out
}

func resolvePythonInputFiles(explicit string, attached string, args map[string]interface{}) string {
	if s := strings.TrimSpace(explicit); s != "" {
		return s
	}
	if s := strings.TrimSpace(attached); s != "" {
		return s
	}
	return inputFileRefsFromRunPythonArgs(args)
}

func inputFileRefsFromRunPythonArgs(args map[string]interface{}) string {
	if len(args) == 0 {
		return ""
	}
	for _, key := range []string{"input_files", "files", "refs"} {
		if refs := normalizeRunPythonInputFileRefsValue(args[key]); refs != "" {
			return refs
		}
	}
	bucket := strings.TrimSpace(fmt.Sprint(args["bucket"]))
	key := strings.TrimSpace(fmt.Sprint(args["key"]))
	if bucket == "" || bucket == "<nil>" || key == "" || key == "<nil>" {
		return ""
	}
	return strings.TrimRight(bucket, "/") + "/" + strings.TrimLeft(key, "/")
}

func normalizeRunPythonInputFileRefsValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []string:
		return strings.Join(cleanRunPythonStringParts(v), ",")
	case []interface{}:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(cleanRunPythonStringParts(parts), ",")
	case map[string]interface{}:
		return normalizeRunPythonInputFileRefsValue(v["refs"])
	default:
		return ""
	}
}

func cleanRunPythonStringParts(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func pythonFormPayload(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	if _, ok := m["output"]; ok {
		return m
	}
	for _, key := range []string{"data", "result"} {
		inner, ok := m[key].(map[string]interface{})
		if !ok {
			continue
		}
		if _, ok2 := inner["output"]; ok2 {
			return inner
		}
	}
	return m
}

func pythonAnyToString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func buildPythonModelGuidance(raw map[string]interface{}) string {
	p := pythonFormPayload(raw)
	if p == nil {
		return ""
	}
	status := strings.TrimSpace(pythonAnyToString(p["status"]))
	out := pythonAnyToString(p["output"])
	jr := pythonAnyToString(p["json_result"])
	lowOut := strings.ToLower(out)

	var lines []string
	appendLine := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		for _, ex := range lines {
			if ex == s {
				return
			}
		}
		lines = append(lines, s)
	}

	switch status {
	case "失败":
		appendLine("【状态为失败】请阅读 output 中的 traceback/错误信息，修正 python_code 后重试。")
		if strings.Contains(out, "ModuleNotFoundError") || strings.Contains(out, "No module named") {
			appendLine("【依赖】ModuleNotFoundError：请优先使用工具说明里已列出的预装库（pandas、numpy、jieba、snownlp、requests、openpyxl、xlsxwriter、python-pptx、matplotlib、plotly、pyecharts、bs4、tabulate、arrow、wordcloud、pytesseract、yt_dlp、PyYAML…）或仅用标准库；若必须新库，请管理员更新 deploy/base/images/app-base/Dockerfile 或基础镜像 requirements.txt 并重打镜像。")
		}
		if strings.Contains(out, "SyntaxError") || strings.Contains(out, "IndentationError") {
			appendLine("【语法】请检查引号、缩进、括号是否匹配；字符串内换行需用三引号或 \\n。")
			appendLine("【重写建议】遇到 SyntaxError/IndentationError 时，不要局部修补旧长脚本；请重新生成一份更短、更扁平、统一 4 空格缩进的完整 python_code。")
			appendLine("【缩进策略】优先减少 for/if/else 多层嵌套；能改成 pandas API、zip、列表推导式或先算中间变量再 return 的，就不要继续堆块。")
		}
		if strings.Contains(out, "UnboundLocalError") {
			appendLine("【作用域】请检查变量是否先使用后赋值；import 语句请放到文件顶部或函数体开头。")
		}
		if strings.Contains(out, "返回了不支持的字段") {
			appendLine("【返回协议】kageos_entry 只能返回 data、output_files、warnings；错误说明请放进 data.error，或直接 raise ValueError(...)。")
		}
		if strings.Contains(out, "IndexError: list index out of range") {
			appendLine("【输入文件】如果代码读取 args[\"input_files\"][0]，请确认工具顶层 input_files 已传入文件引用，或本轮消息确实带有上传附件；不要把 bucket/key 当成本地文件路径读取。")
		}
		if strings.Contains(out, "requests.exceptions.ConnectionError") || strings.Contains(out, "Failed to establish a new connection") {
			appendLine("【文件读取】不要 requests.get 对象存储 URL 或猜 COS 域名；应把文件引用传给工具顶层 input_files，然后读取注入的 args[\"input_files\"] 本地路径。")
		}
		if strings.Contains(out, "keys must be str, int, float, bool or None, not tuple") {
			appendLine("【JSON 序列化】data 中的 dict key 不能是 tuple。pandas groupby/agg 后请先 reset_index() 或 as_index=False，再用 to_dict('records')。")
		}
		if strings.Contains(out, "not found in axis") {
			appendLine("【列处理】删除列前先确认列存在；不确定时优先显式选择要保留的列，而不是 drop 一组临时列。")
		}
		if strings.Contains(lowOut, "timeout") || strings.Contains(out, "deadline exceeded") || strings.Contains(out, "context deadline") {
			appendLine("【超时】可适当增大 timeout_seconds（最大 300），或拆分计算、减少数据量。")
		}
	case "成功":
		if strings.Contains(jr, "JSON解析失败") {
			appendLine("【结构化结果解析失败】请确认 python_code 定义了 kageos_entry(args, output_dir)，并返回 {\"data\": ...} 这种合法 dict，而不是靠 print 输出 JSON。")
		}
		if strings.Contains(jr, "输出不是JSON格式") || strings.Contains(jr, "不是JSON格式") {
			appendLine("【降级·正常】当前为纯文本日志输出（print）。若只需报告说明，可直接使用；若你需要程序取字段，请让 kageos_entry 返回 {\"data\": {...}}。")
		}
		if strings.Contains(jr, "标记内无 JSON") {
			appendLine("【data 为空】请确保 kageos_entry 返回的 dict 中包含 data；若本意只是打印日志，可继续使用 print。")
		}
		if jr == "" && out != "" && !strings.Contains(out, "<python-out>") {
			appendLine("【提示】当前没有结构化 data，先以 output 为准；若需要上层稳定解析，请改为让 kageos_entry 返回 {\"data\": ...}。")
		}
	default:
		if status != "" {
			appendLine("【状态】status=" + status + "：请结合 output、json_result 判断。")
		}
	}

	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}
