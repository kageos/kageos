package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
)

// runPythonPreinstallDoc 与 deploy/base/images/app-base/Dockerfile 中 apt/python3-* 与 pip3 install 预装保持一致；改镜像时请同步更新本文案
const runPythonPreinstallDoc = `**可直接 import 的第三方库：**
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

**若 import 报错：** 优先使用可用库或标准库，也可以用 packages 声明额外 PyPI 包。`

type RunPythonTool struct{}

type runPythonArgs struct {
	PythonCode     string                 `json:"python_code" schema_desc:"完整 Python 源码" schema_required:"true"`
	Args           map[string]interface{} `json:"args" schema_desc:"注入脚本的对象参数（推荐）"`
	InputFiles     string                 `json:"input_files" schema_desc:"可选文件引用字符串，格式 bucket/object_key，多文件用英文逗号分隔；不传时自动使用当前用户消息上传的附件；支持直接填入上一步 output_files 返回的路径，实现 output -> input 文件流转"`
	Packages       string                 `json:"packages" schema_desc:"可选额外 pip 包，逗号分隔；填写 PyPI 安装名而不是 import 名，仅支持简单包名或版本约束，如 openai,scikit-learn==1.5.0,zxing-cpp；每次需要该依赖时均应声明"`
	TimeoutSeconds *int                   `json:"timeout_seconds" schema_desc:"超时秒数"`
}

var runPythonToolDef = toolDefinition[runPythonArgs](
	"run_python",
	runPythonPreinstallDoc+`

处理数据、计算或生成文件。只需提供脚本及输入参数。

**脚本契约：**
- 定义 def kageos_entry(args, output_dir):，使用 4 空格缩进。
- args 是对象参数。输入附件的可读路径由 args["input_files"] 提供，直接 open 或交给 pandas 读取。
- input_files 接收上传附件或上一步 output_files 的文件引用，多文件以英文逗号分隔；不传则使用本轮附件。文件引用原样传递，不自行拼下载地址。
- 输出文件写入 output_dir；返回 dict，仅包含 data（JSON 可序列化数据）、output_files（每项含 path，可选 name）、warnings（字符串列表）。
- print 只做日志，不作为主结果。data 不包含 Timestamp、numpy 标量、tuple 字典键等不能直接转 JSON 的对象。
- packages 填 PyPI 安装名，可带版本约束，如 openai,scikit-learn==1.5.0；例如 zxing-cpp 对应 import zxingcpp。只接收简单包名、extras 和版本约束。
- timeout_seconds 默认 120，最大 300。
- 图表使用默认字体；需要时设置 axes.unicode_minus=False。

**结果与文件：**
输出文件会由工作台展示文件组件。不要手写“下载文件：xxx”、Markdown 下载链接或伪 URL。output_files 返回的文件引用可以直接传给下一次 input_files。
执行失败时根据脚本错误类型和行号修正输入或代码。
`,
)

func (t *RunPythonTool) Definition() dto.ToolDef {
	return runPythonToolDef
}

func (t *RunPythonTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[runPythonArgs](call.Args)
	if err != nil {
		return toolResult("run_python 参数解析失败: "+err.Error(), true)
	}
	content, isError, data := runPythonTool(ctx, args, call.Files, call.FullCodePath)
	return toolResultWithDataAndMetadata(content, isError, data, metadataForDisplayFileFields("output_files"))
}

// runPythonTool 调用当前工作区应用内置的私有 Python runtime。
func runPythonTool(ctx context.Context, args runPythonArgs, attachedFiles string, currentFullCodePath string) (string, bool, map[string]interface{}) {
	// python_code 必须按原文透传。历史上尝试清理 BOM/控制字符/缩进会让真实错误更难定位，
	// 也可能改变 Python 源码语义；这里仅判空，不做任何隐式修复。
	code := args.PythonCode
	if strings.TrimSpace(code) == "" {
		return "run_python 需传 python_code。", true, nil
	}
	workspaceRoot := runPythonWorkspaceRoot(currentFullCodePath)
	if workspaceRoot == "" {
		return "run_python 需要当前工作区上下文（full_code_path 至少包含 /user/app）。", true, nil
	}

	body := &dto.RunPythonRuntimeReq{
		PythonCode:         code,
		CollectOutputFiles: true,
	}
	if len(args.Args) > 0 {
		body.Args = args.Args
	}
	if inputFiles := resolvePythonInputFiles(args.InputFiles, attachedFiles, args.Args); inputFiles != "" {
		body.InputFiles = inputFiles
	}
	if packages, err := normalizeRunPythonPackages(args.Packages); err != nil {
		return "run_python packages 参数无效: " + err.Error(), true, nil
	} else if packages != "" {
		body.Packages = packages
	}
	timeoutSec := 120
	if args.TimeoutSeconds != nil && *args.TimeoutSeconds > 0 {
		timeoutSec = *args.TimeoutSeconds
	}
	if timeoutSec > 300 {
		timeoutSec = 300
	}
	body.TimeoutSeconds = timeoutSec

	runtimeCtx := withRunPythonRuntimeSource(ctx)
	result, err := apicall.RunWorkspacePython(runtimeCtx, workspaceRoot, body)
	if err != nil {
		logger.Errorf(ctx, "[RunPython] RunWorkspacePython 失败: %v", err)
		return publicToolBackendError(ctx, "run_python", err), true, nil
	}
	out := publicPythonResult(result, workspaceRoot)
	if g := buildPythonModelGuidance(out); g != "" {
		out["_model_guidance"] = g
	}
	content, _ := formatJSONResult(out)
	isError := out["status"] == "失败"
	return content, isError, out
}

func withRunPythonRuntimeSource(ctx context.Context) context.Context {
	ctx = withAgentToolClientSource(ctx)
	return contextx.WithSourceInfo(ctx, contextx.SourceTypeAgentTool, contextx.GetSourceRef(ctx))
}

var runPythonPackageSpecPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*(\[[A-Za-z0-9._-]+\])?((===|==|!=|>=|<=|~=|>|<)[A-Za-z0-9][A-Za-z0-9.*_+!~-]*)?$`)

func normalizeRunPythonPackages(packages string) (string, error) {
	packages = strings.TrimSpace(packages)
	if packages == "" {
		return "", nil
	}
	if len(packages) > 512 {
		return "", fmt.Errorf("总长度不能超过 512 个字符")
	}
	parts := strings.Split(packages, ",")
	if len(parts) > 10 {
		return "", fmt.Errorf("最多一次声明 10 个包")
	}

	out := make([]string, 0, len(parts))
	seen := make(map[string]bool)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if err := validateRunPythonPackageSpec(part); err != nil {
			return "", err
		}
		key := strings.ToLower(part)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, part)
	}
	return strings.Join(out, ","), nil
}

func validateRunPythonPackageSpec(pkg string) error {
	if strings.HasPrefix(pkg, "-") {
		return fmt.Errorf("包 %q 不能是 pip 命令行参数", pkg)
	}
	if strings.ContainsAny(pkg, " \t\r\n") {
		return fmt.Errorf("包 %q 不能包含空白字符", pkg)
	}
	if strings.ContainsAny(pkg, `/\`) || strings.Contains(pkg, "://") || strings.Contains(pkg, "@") {
		return fmt.Errorf("包 %q 不能是 URL、本地路径或直接引用", pkg)
	}
	if !runPythonPackageSpecPattern.MatchString(pkg) {
		return fmt.Errorf("包 %q 不是支持的简单包名或版本约束", pkg)
	}
	return nil
}

func runPythonWorkspaceRoot(currentFullCodePath string) string {
	trimmed := strings.Trim(strings.TrimSpace(currentFullCodePath), "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	return "/" + parts[0] + "/" + parts[1]
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
			appendLine("【依赖】ModuleNotFoundError：请优先使用工具说明里已列出的预装库（pandas、numpy、jieba、snownlp、requests、openpyxl、xlsxwriter、python-pptx、matplotlib、plotly、pyecharts、bs4、tabulate、arrow、wordcloud、pytesseract、yt_dlp、PyYAML…）或仅用标准库；临时新库可在 packages 参数中声明简单 PyPI 包名/版本约束。注意 packages 填 pip 安装名，不一定等于 import 名，例如 packages: zxing-cpp 对应 import zxingcpp。")
		}
		if strings.Contains(out, "安装 Python 包") || strings.Contains(lowOut, "no matching distribution found") || strings.Contains(lowOut, "could not find a version") {
			appendLine("【依赖安装】packages 会执行 pip install；请确认填写的是 PyPI 安装名而不是 import 名，必要时换用预装库或标准库。例如二维码/条码识别应填 packages: zxing-cpp，代码里再 import zxingcpp。")
		}
		if strings.Contains(out, "SyntaxError") || strings.Contains(out, "IndentationError") {
			appendLine("【语法】请检查引号、缩进、括号是否匹配；字符串内换行需用三引号或 \\n。")
			appendLine("【重写建议】遇到 SyntaxError/IndentationError 时，不要局部修补旧长脚本；请重新生成一份更短、更扁平、统一 4 空格缩进的完整 python_code。")
			appendLine("【缩进策略】优先减少 for/if/else 多层嵌套；能改成 pandas API、zip、列表推导式或先算中间变量再 return 的，就不要继续堆块。")
		}
		if strings.Contains(out, "必须定义函数 kageos_entry") || strings.Contains(out, "python_code 必须定义函数") {
			appendLine("【入口协议】run_python 不是普通 Python REPL。请重写完整 python_code，从 def kageos_entry(args, output_dir): 开始；返回 dict 只包含 data、output_files、warnings，例如 {\"data\": {...}, \"warnings\": [], \"output_files\": []}。print 只做日志，不作为主结果。")
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

func publicPythonResult(raw map[string]interface{}, workspace string) map[string]interface{} {
	p := pythonFormPayload(raw)
	out := make(map[string]interface{})
	for _, key := range []string{"status", "json_result", "output_files"} {
		if value, ok := p[key]; ok {
			out[key] = value
		}
	}
	if output, ok := p["output"]; ok {
		out["output"] = publicDiagnosticText(pythonAnyToString(output), workspace)
	}
	if out["status"] != "成功" && out["status"] != "失败" {
		out["status"] = "失败"
		out["output"] = "execution_result_unavailable：未收到有效执行结果。"
	}
	return out
}
