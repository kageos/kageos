package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// runOfficialPythonPreinstallDoc 与 deploy/base/images/app-base/Dockerfile 中 apt/python3-* 与 pip3 install 预装保持一致；改镜像时请同步更新本文案
const runOfficialPythonPreinstallDoc = `**生产镜像已预装、可直接 import 的第三方库（对应 deploy/base/images/app-base/Dockerfile）：**
- 数据与图表：pandas、numpy、scipy、matplotlib、seaborn
- 网络与表格：requests、openpyxl
- 图像：PIL（Pillow，如 from PIL import Image）
- 文档与 PDF：docx（python-docx）、PyPDF2、pdfplumber
- 中文分词：jieba
- 另有 **Python 标准库**（json、re、collections、datetime、itertools、math、random 等）

**若 import 报错：** 优先改用上面列表或标准库；需要新依赖时请管理员更新 Dockerfile / 官方 requirements.txt 并重打镜像。不可在本工具参数里指定 pip 包。
**环境差异：** 本地非 Docker 运行时以本机 python 为准，可能与镜像不一致。`

const officialPythonFormPath = "/system/official/python/execute"

type RunOfficialPythonTool struct{}

type runOfficialPythonArgs struct {
	PythonCode     string `json:"python_code" schema_desc:"完整 Python 源码" schema_required:"true"`
	ArgsJSON       string `json:"args_json" schema_desc:"注入脚本的 JSON 对象字符串"`
	TimeoutSeconds *int   `json:"timeout_seconds" schema_desc:"超时秒数"`
}

var runOfficialPythonToolDef = toolDefinition[runOfficialPythonArgs](
	"run_official_python",
	runOfficialPythonPreinstallDoc+`

**执行环境：** Python 跑在 **应用运行时容器内**（Podman 等业务容器，**不是宿主机**）。本工具调用官方路径 **/system/official/python/execute**，由 **官方应用** 对应容器执行；脚本在 **临时目录** 中运行，不把工作区源码树当作工作目录。

**无法输出文件到工作台供用户下载：** 本工具只能返回文本/JSON（output/json_result），**不能**把 Python 生成的 PNG/Excel 等变成工作台可下载附件。

**若需要「处理后的文件给用户下载」：** 请先用 **read_doc** 读取内置示例文档 **/builtin/doc/case_catalog/form/python_output**（含 PRD 与完整 Go 示例），再按文档配合 **agent-app SDK** 在用户应用内新增 Form：**pythonRuntime.NewExecutor** → **defer executor.Close()**（默认临时目录）→ Go 用 **filepath.Abs** 得到 **绝对路径**（如 GetTraceOutputDir 下文件）经请求传给 Python → Python **直接写入该路径**（如 savefig，勿用相对路径互传，Go/Python **cwd 不同**）→ 响应 **types.Files**（ResponseFiles 使用同一绝对路径）。Go 与 Python 为**同机子进程**，非网络隔离。

**两种输出方式（二选一或组合）：**
1. **结构化结果（推荐）**：脚本末尾调用 output_json(字典或列表)，键用双引号、值为 JSON 可序列化类型。返回里 json_result 为格式化后的 JSON，便于你后续取字段。
2. **纯文本/报表**：用 print(...)。返回里以 output 为准；json_result 会提示「非 JSON」，属正常降级，不要误判为失败。

**如何读返回：** status=成功 时，有结构化数据优先看 json_result；无则读 output。若 json_result 含「JSON解析失败」而 output 里已有 <python-out> 片段，以 output 内 JSON 为准或修正脚本后重试。

**参数：** args_json 为 JSON 对象字符串，字段注入脚本全局命名空间。timeout_seconds 默认 120、上限 300。

返回中可能含 _model_guidance：面向你的纠错/降级说明，请优先阅读。`,
)

func (t *RunOfficialPythonTool) Definition() dto.ToolDef {
	return runOfficialPythonToolDef
}

func (t *RunOfficialPythonTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[runOfficialPythonArgs](call.Args)
	if err != nil {
		return toolResult("run_official_python 参数解析失败: "+err.Error(), true)
	}
	content, isError := runOfficialPythonTool(ctx, args)
	return toolResult(content, isError)
}

// runOfficialPythonTool 调用系统官方 Python 执行 Form
func runOfficialPythonTool(ctx context.Context, args runOfficialPythonArgs) (string, bool) {
	code := strings.TrimSpace(args.PythonCode)
	if code == "" {
		return "run_official_python 需传 python_code。", true
	}
	body := map[string]interface{}{
		"python_code": code,
	}
	if argsJSON := strings.TrimSpace(args.ArgsJSON); argsJSON != "" {
		body["args_json"] = argsJSON
	}
	timeoutSec := 120
	if args.TimeoutSeconds != nil && *args.TimeoutSeconds > 0 {
		timeoutSec = *args.TimeoutSeconds
	}
	if timeoutSec > 300 {
		timeoutSec = 300
	}
	body["timeout_seconds"] = timeoutSec

	result, err := apicall.FormSubmit(ctx, officialPythonFormPath, body)
	if err != nil {
		logger.Errorf(ctx, "[RunOfficialPython] FormSubmit 失败: %v", err)
		return "run_official_python 调用失败: " + err.Error() + "\n\n【给模型】可检查 python_code 是否过长、args_json 是否为合法 JSON 对象字符串；网络或权限问题可稍后重试。", true
	}
	out := make(map[string]interface{}, len(result)+1)
	for k, v := range result {
		out[k] = v
	}
	if g := buildOfficialPythonModelGuidance(result); g != "" {
		out["_model_guidance"] = g
	}
	return formatJSONResult(out)
}

func officialPythonFormPayload(m map[string]interface{}) map[string]interface{} {
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

func officialPythonAnyToString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func buildOfficialPythonModelGuidance(raw map[string]interface{}) string {
	p := officialPythonFormPayload(raw)
	if p == nil {
		return ""
	}
	status := strings.TrimSpace(officialPythonAnyToString(p["status"]))
	out := officialPythonAnyToString(p["output"])
	jr := officialPythonAnyToString(p["json_result"])
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
			appendLine("【依赖】ModuleNotFoundError：请优先使用工具说明里已列出的预装库（pandas、numpy、jieba、requests、openpyxl、matplotlib…）或仅用标准库；若必须新库，请管理员更新 deploy/base/images/app-base/Dockerfile 或官方 requirements.txt 并重打镜像。")
		}
		if strings.Contains(out, "SyntaxError") || strings.Contains(out, "IndentationError") {
			appendLine("【语法】请检查引号、缩进、括号是否匹配；字符串内换行需用三引号或 \\n。")
		}
		if strings.Contains(lowOut, "timeout") || strings.Contains(out, "deadline exceeded") || strings.Contains(out, "context deadline") {
			appendLine("【超时】可适当增大 timeout_seconds（最大 300），或拆分计算、减少数据量。")
		}
	case "成功":
		if strings.Contains(jr, "JSON解析失败") {
			appendLine("【json_result 解析失败】执行已成功；结构化内容可能在 output 的 <python-out>...</python-out> 内。请以 output 为准，或改为 output_json(合法 dict/list)，避免在标记内输出非 JSON 文本。")
		}
		if strings.Contains(jr, "输出不是JSON格式") || strings.Contains(jr, "不是JSON格式") {
			appendLine("【降级·正常】当前为纯文本输出（print）。若用户只需要报告/说明，无需改代码；若你需要程序取字段，请让脚本改用 output_json({...})。")
		}
		if strings.Contains(jr, "标记内无 JSON") {
			appendLine("【output_json 空内容】请确保 output_json 传入非空 dict/list；若本意是纯文本请改用 print。")
		}
		if jr == "" && out != "" && !strings.Contains(out, "<python-out>") {
			appendLine("【提示】未使用 output_json 时 json_result 常为空，以 output 为准即可。")
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
