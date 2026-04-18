package service

import (
	"context"
	"encoding/json"
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

const officialPythonFormPath = "/system/official/python/execute.form"

type RunOfficialPythonTool struct{}

type runOfficialPythonArgs struct {
	PythonCode     string                 `json:"python_code" schema_desc:"完整 Python 源码" schema_required:"true"`
	Args           map[string]interface{} `json:"args" schema_desc:"注入脚本的对象参数（推荐）"`
	ArgsJSON       string                 `json:"args_json" schema_desc:"注入脚本的 JSON 对象字符串（兼容旧调用方）"`
	TimeoutSeconds *int                   `json:"timeout_seconds" schema_desc:"超时秒数"`
}

var runOfficialPythonToolDef = toolDefinition[runOfficialPythonArgs](
	"run_official_python",
	runOfficialPythonPreinstallDoc+`

**执行环境：** Python 跑在 **应用运行时容器内**（Podman 等业务容器，**不是宿主机**）。本工具调用官方路径 **/system/official/python/execute.form**，由 **官方应用** 对应容器执行；脚本在 **临时目录** 中运行，不把工作区源码树当作工作目录。

**固定入口协议：**
- python_code **必须定义**：def agentos_entry(args, output_dir): ...
- 第一个参数 args 为传入的对象参数；第二个参数 output_dir 为受控输出目录
- 返回值 **必须是 dict**，仅允许：
  - data: JSON 可序列化结果
  - output_files: 输出文件列表，每项至少含 path
  - warnings: 警告字符串列表
- print(...) 只用于日志，不作为主结果协议

**输出结果：** 官方执行端会解析 agentos_entry 的返回值；若返回里有 **output_files**，Go 侧会负责校验、上传并构造成最终 types.Files，工作台自动展示可下载附件。

**输出文件约束：**
- 只能声明写在 output_dir 里的最终文件
- 不要返回随机路径、临时缓存路径、相对路径拼猜出来的文件
- 每个输出文件项建议形如：{"path": "/abs/path/in/output_dir/report.xlsx", "name": "report.xlsx"}

**若你需要把字段、权限、命名规则固化为应用接口：** 请用 **read_doc** 读取内置示例文档 **/system/prompt/case_catalog/form/python_output**（含 PRD 与完整 Go 示例），再按文档配合 **agent-app SDK** 在用户应用内新增 Form：**pythonRuntime.NewExecutor** → **defer executor.Close()**（默认临时目录）→ Go 用 **filepath.Abs** 得到 **绝对路径**（如 GetTraceOutputDir 下文件）经请求传给 Python → Python **直接写入该路径**（如 savefig，勿用相对路径互传，Go/Python **cwd 不同**）→ 用 **OutputFilePaths + ResponseFiles** 下发附件。Go 与 Python 为**同机子进程**，非网络隔离。

**参数：** 推荐用 args 直接传对象参数；args_json 仅兼容旧调用方。timeout_seconds 默认 120、上限 300。

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
	content, isError, data := runOfficialPythonTool(ctx, args)
	return toolResultWithData(content, isError, data)
}

// runOfficialPythonTool 调用系统官方 Python 执行 Form
func runOfficialPythonTool(ctx context.Context, args runOfficialPythonArgs) (string, bool, map[string]interface{}) {
	code := strings.TrimSpace(args.PythonCode)
	if code == "" {
		return "run_official_python 需传 python_code。", true, nil
	}
	body := map[string]interface{}{
		"python_code":          code,
		"collect_output_files": true,
	}
	if len(args.Args) > 0 {
		body["args"] = args.Args
		if b, err := json.Marshal(args.Args); err == nil {
			body["args_json"] = string(b)
		}
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
		return "run_official_python 调用失败: " + err.Error() + "\n\n【给模型】可检查 python_code 是否过长、args/args_json 是否为合法对象；网络或权限问题可稍后重试。", true, nil
	}
	out := make(map[string]interface{}, len(result)+1)
	for k, v := range result {
		out[k] = v
	}
	if g := buildOfficialPythonModelGuidance(result); g != "" {
		out["_model_guidance"] = g
	}
	content, isError := formatJSONResult(out)
	return content, isError, out
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
			appendLine("【结构化结果解析失败】请确认 python_code 定义了 agentos_entry(args, output_dir)，并返回 {\"data\": ...} 这种合法 dict，而不是靠 print 输出 JSON。")
		}
		if strings.Contains(jr, "输出不是JSON格式") || strings.Contains(jr, "不是JSON格式") {
			appendLine("【降级·正常】当前为纯文本日志输出（print）。若只需报告说明，可直接使用；若你需要程序取字段，请让 agentos_entry 返回 {\"data\": {...}}。")
		}
		if strings.Contains(jr, "标记内无 JSON") {
			appendLine("【data 为空】请确保 agentos_entry 返回的 dict 中包含 data；若本意只是打印日志，可继续使用 print。")
		}
		if jr == "" && out != "" && !strings.Contains(out, "<python-out>") {
			appendLine("【提示】当前没有结构化 data，先以 output 为准；若需要上层稳定解析，请改为让 agentos_entry 返回 {\"data\": ...}。")
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
