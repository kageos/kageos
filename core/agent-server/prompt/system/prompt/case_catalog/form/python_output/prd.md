# 案例：Python 容器内产物输出（单 Form）

## 一、项目概要

- **类型**：单 Form，POST，无 Table。
- **路由**：`sandbox_file_out_demo.form`；路由组 `/form/python_output`。
- **适合参考**：`pythonRuntime.NewExecutor`、**`defer executor.Close()`**（默认临时工作目录须释放）、固定入口 `agentos_entry(args, output_dir)`、`ExecuteJSONWithResult`；Go 计算 **`filepath.Abs` 输出绝对路径** 经请求传给 Python，Python **直接写入该路径**（如 `savefig`），再通过 `output_files` 声明该文件，Go 再 **`OutputFilePaths()` + `ResponseFiles(...)`** 下发给用户下载。勿用**相对路径**在 Go/Python 之间互传（**双方进程 cwd 不同**）。
- **运行关系**：Go 调用 Python 为**同一运行时环境内**拉起子进程（同机、非网络隔离），Python 只要写 Go 给出的绝对路径，Go 即可读同一文件。
- **与 `run_official_python` 区别**：官方工具链路现在支持输入附件自动下载为 `args.input_files` 本地路径列表，也支持 `output_files` 下发附件；但本类 Form 仍适合需要**固化字段、沉淀业务接口、控制响应结构/权限/命名规则**的场景（跑在**应用运行时容器内**，非宿主机）。

---

## 二、PRD 要点（表格格式）

### 容器内 Python 生成 PNG（sandbox_file_out_demo.form，POST）

**请求**（表单字段五列：字段 | 类型 | 必填 | 默认值 | 说明）

| 字段     | 类型     | 必填 | 默认值 | 说明           |
|----------|----------|------|--------|----------------|
| 图片标题 | 文本输入 | ✓   | —      | 将绘制在图上，并用于生成文件名（非法字符会替换为 `_`） |

**响应**

| 字段     | 类型     | 说明 |
|----------|----------|------|
| 生成的 PNG | 文件     | 可下载；由 matplotlib 在容器内生成 |
| 说明     | 多行文本 | 固定说明文案 |
| 状态     | 文本     | 成功 |

**业务规则简述**

1. Go：`GetTraceOutputDir()` → `MkdirAll` → 拼文件名 → **`filepath.Abs`** 得到 **`image_output_path`**，随 `WithRequest` 传给 Python。
2. Python：`matplotlib` 非交互后端（Agg），在 `agentos_entry(args, output_dir)` 中校验绝对路径后 **`plt.savefig(image_output_path)`**，返回 `{"data": {...}, "output_files": [...]}`（**不再经 base64 传图**）。
3. Go：`ExecuteJSONWithResult` 后用 **`OutputFilePaths()`** 校验 `output_files`，再 `ResponseFiles(...)`；**`defer executor.Close()`** 释放 Python 临时工作区。
4. 依赖：生产镜像预装 matplotlib 等（见 `deploy/base/images/app-base/Dockerfile`）。

---

## 三、文件与路由

| 文件                     | 说明                         | 注册路由                          |
|--------------------------|------------------------------|-----------------------------------|
| init_.go                 | 包路由组 `/form/python_output` | —                                 |
| sandbox_file_out_demo.go | 演示 PNG 产出与 Files 响应    | POST sandbox_file_out_demo.form   |

---

## 四、说明

代码随本案例一起提供；read_doc 本案例路径 **`/system/prompt/case_catalog/form/python_output`**（同步 case_catalog 后）即获得 PRD 与代码，无需再调用 read_go_file。


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### sandbox_file_out_demo.go

```go
//<文件名>sandbox_file_out_demo.go</文件名>
//
// 标准模式说明：
// 1) run_official_python 现在支持输入附件自动下载为 args.input_files，也支持 output_files 下发附件；
//    若你要把字段、权限、命名规则固化到应用接口，仍建议在 **本应用** 做 Form。
// 2) 若既要 Python 处理能力又要可下载文件：在 **本应用** Form 里调 pythonRuntime.NewExecutor；
//    执行发生在 **应用运行时容器内**（工作空间容器，非宿主机）。Go 与 Python **同机**（子进程），非网络隔离。
// 3) Go 将 **绝对路径**（filepath.Abs + GetTraceOutputDir）通过请求字段传给 Python，Python 在固定入口
//    agentos_entry(args, output_dir) 中直接 savefig 到该路径，并返回 output_files 声明；
//    Go 再用 ExecuteJSONWithResult + OutputFilePaths() 校验产物，再 ResponseFiles 下发。
//    勿用相对路径互传（Go/Python 的 cwd 不同）。须 defer executor.Close() 释放默认临时目录。
// 大模型请先 read_doc /system/prompt/case_catalog/form/python_output。

package python_output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	pythonRuntime "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/runtime/python"
)

// SandboxFileOutDemoReq 请求：用户只传标题，由 Python 生成一张说明图
type SandboxFileOutDemoReq struct {
	Title string `json:"title" widget:"name:图片标题;type:input;placeholder:例如：季度销售摘要" validate:"required"`
}

// SandboxFileOutDemoResp 响应：可下载 PNG + 说明文案
type SandboxFileOutDemoResp struct {
	OutputPNG string `json:"output_png" widget:"name:生成的 PNG;type:files"`
	Info      string       `json:"info" widget:"name:说明;type:text_area"`
	Status    string       `json:"status" widget:"name:状态;type:text"`
}

// SandboxFileOutDemo Form：同机 Python 按绝对路径落盘 → Go ResponseFiles
func SandboxFileOutDemo(ctx *app.Context, resp response.Response) error {
	var req SandboxFileOutDemoReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	baseName := sanitizeDemoFileName(req.Title)
	outPath := filepath.Join(outputDir, baseName+"_sandbox_demo.png")
	outPath, err := filepath.Abs(outPath)
	if err != nil {
		return fmt.Errorf("解析输出绝对路径失败: %w", err)
	}

	type pyReq struct {
		Title           string `json:"title"`
		ImageOutputPath string `json:"image_output_path"`
	}
	pythonCode := buildSandboxDemoPython()

	executor := pythonRuntime.NewExecutor(pythonCode).
		WithRequest(pyReq{Title: strings.TrimSpace(req.Title), ImageOutputPath: outPath}).
		WithOutputDir(outputDir).
		WithTimeout(45 * time.Second)
	defer func() { _ = executor.Close() }()

	var result struct {
		Info string `json:"info"`
	}
	execResult, err := executor.ExecuteJSONWithResult(ctx, &result)
	if err != nil {
		logger.Errorf(ctx, "[SandboxFileOutDemo] Python 执行失败: %v", err)
		return fmt.Errorf("Python 生成失败: %w", err)
	}

	outputPaths, err := execResult.OutputFilePaths()
	if err != nil {
		return fmt.Errorf("输出文件校验失败: %w", err)
	}
	if len(outputPaths) == 0 {
		return fmt.Errorf("Python 未声明输出文件")
	}

	files := fs.ResponseFiles(outputPaths)

	return resp.Form(&SandboxFileOutDemoResp{
		OutputPNG: files,
		Info:      result.Info,
		Status:    "成功",
	}).Build()
}

func buildSandboxDemoPython() string {
	return `import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import os

plt.rcParams['axes.unicode_minus'] = False

def agentos_entry(args, output_dir):
    title = args["title"]
    image_output_path = args["image_output_path"]
    if not image_output_path or not os.path.isabs(image_output_path):
        raise ValueError("image_output_path 必须为非空的绝对路径")

    fig, ax = plt.subplots(figsize=(6, 3.2), dpi=120)
    ax.set_xlim(0, 1)
    ax.set_ylim(0, 1)
    ax.axis('off')
    ax.text(0.5, 0.62, title, ha='center', va='center', fontsize=15, fontweight='bold')
    ax.text(0.5, 0.38, '模式：Go 传绝对路径 → Python savefig；同机子进程；defer Close() 释放临时目录', ha='center', va='center', fontsize=9, linespacing=1.4)
    ax.text(0.5, 0.12, 'read_doc: /system/prompt/case_catalog/form/python_output', ha='center', va='center', fontsize=8, color='#666')

    out_dir = os.path.dirname(image_output_path)
    if out_dir:
        os.makedirs(out_dir, exist_ok=True)
    plt.savefig(image_output_path, format='png', bbox_inches='tight', pad_inches=0.2)
    plt.close(fig)

    return {
        "data": {
            "info": "本图由 matplotlib 在本应用容器内写入 Go 指定的绝对路径；Go 用 ResponseFiles 下发。勿用相对路径在 Go/Python 间互传 cwd 不同。详阅 read_doc /system/prompt/case_catalog/form/python_output。"
        },
        "output_files": [
            {"path": image_output_path, "name": os.path.basename(image_output_path)}
        ]
    }
`
}

func sanitizeDemoFileName(name string) string {
	if name == "" {
		return "output"
	}
	for _, ch := range []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"} {
		name = strings.ReplaceAll(name, ch, "_")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "output"
	}
	return name
}

var sandboxFileOutDemoTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Python 容器内产物输出（示例）",
		Desc:     `演示：同机 Python 按 **绝对路径** 落盘并声明 output_files，Go 用 **OutputFilePaths + ResponseFiles** 下发附件。须 defer executor.Close()。大模型请先 read_doc /system/prompt/case_catalog/form/python_output。`,
		Tags:     []string{"Python", "示例", "文件输出", "matplotlib"},
		Request:  &SandboxFileOutDemoReq{},
		Response: &SandboxFileOutDemoResp{},
	},
}

func init() {
	packageContext.POST("sandbox_file_out_demo.form", SandboxFileOutDemo, sandboxFileOutDemoTemplate)
}
```
