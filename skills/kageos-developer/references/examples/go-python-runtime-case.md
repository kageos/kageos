# Go 调 Python Runtime 最佳实践案例

适用：业务 Form 需要 Python 生态能力，例如 pandas/openpyxl 数据清洗、jieba/NLP、matplotlib 图表图片、PDF/图片处理、模型推理前后处理，且这些能力需要沉淀成稳定的工作台函数。

## 先判断用哪条路

- 一次性脚本、临时分析、快速转换：优先用安装自带的 `/system/tools/runtime/python.form`。
- 需要固定字段、命名规则、响应结构、权限、业务含义、定时执行或长期复用：在目标业务 app 中写 Form，用 Go 调 `pythonRuntime.NewExecutor`。

## 固定协议

Python 代码必须定义：

```python
def kageos_entry(args, output_dir):
    return {
        "data": {...},
        "output_files": [{"path": "...", "name": "..."}],
        "warnings": []
    }
```

规则：

- `args` 来自 Go 的 `WithRequest(...)`，必须是 JSON 可序列化对象。
- `output_dir` 是受控输出目录；输出文件必须写到这里，且用绝对路径返回。
- 返回值只使用 `data`、`output_files`、`warnings`。
- 错误用 `raise ValueError(...)` 或让 Go 包装系统错误；不要靠 print JSON 给 Go 解析。

## 纯结构化结果模式

```go
import (
	"fmt"
	"time"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	pythonRuntime "github.com/kageos/kageos-sdk/agent-app/runtime/python"
)

type AnalyzeReq struct {
	Text string `json:"text" widget:"name:文本;type:text_area" validate:"required"`
}

type AnalyzeResp struct {
	WordCount int      `json:"word_count" widget:"name:词数;type:integer"`
	Keywords  []string `json:"keywords" widget:"name:关键词;type:text_area"`
}

func Analyze(ctx *app.Context, resp response.Response) error {
	var req AnalyzeReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	executor := pythonRuntime.NewExecutor(analyzePythonCode()).
		WithRequest(req).
		WithTimeout(30 * time.Second)
	defer func() { _ = executor.Close() }()

	var result AnalyzeResp
	if err := executor.ExecuteJSON(ctx, &result); err != nil {
		return fmt.Errorf("[系统错误]-[Analyze] Python 执行失败: %w", err)
	}
	return resp.Form(&result).Build()
}

func analyzePythonCode() string {
	return `def kageos_entry(args, output_dir):
    text = args["text"]
    words = [w for w in text.split() if w.strip()]
    return {"data": {"word_count": len(words), "keywords": words[:10]}}`
}
```

## 文件输入模式

平台 `type:files` 字段传到 Go 里的是文件引用字符串，不是本地路径。业务 app 需要读上传文件时，先下载成运行时本地临时文件，再把本地绝对路径传给 Python。涉及输入文件路径正规化的 Go 文件需要引入 `path/filepath`。

```go
type ProcessFileReq struct {
	InputFiles string `json:"input_files" widget:"name:输入文件;type:files;max_count:20" validate:"required"`
}

func PrepareInputFiles(ctx *app.Context, req *ProcessFileReq) ([]string, func(), error) {
	fs := ctx.GetFS()
	localPaths := fs.DownloadFiles(req.InputFiles)
	cleanup := func() { fs.RemoveFiles(localPaths) }
	if len(localPaths) == 0 {
		cleanup()
		return nil, func() {}, fmt.Errorf("没有找到输入文件")
	}

	inputPaths := make([]string, 0, len(localPaths))
	for _, path := range localPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			cleanup()
			return nil, func() {}, err
		}
		inputPaths = append(inputPaths, absPath)
	}
	return inputPaths, cleanup, nil
}
```

把 `inputPaths` 放进 `WithRequest(...)`：

```go
inputPaths, cleanup, err := PrepareInputFiles(ctx, &req)
if err != nil {
	return err
}
defer cleanup()

executor := pythonRuntime.NewExecutor(processPythonCode()).
	WithRequest(map[string]any{"input_files": inputPaths}).
	WithTimeout(60 * time.Second)
defer func() { _ = executor.Close() }()
```

对应 Python 只读本地路径：

```python
def kageos_entry(args, output_dir):
    input_files = args.get("input_files") or []
    if not input_files:
        raise ValueError("没有找到输入文件")
    with open(input_files[0], "r", encoding="utf-8", errors="ignore") as f:
        text = f.read()
    return {"data": {"chars": len(text)}}
```

如果只是一次性处理，优先调用 `/system/tools/runtime/python.form`：它的顶层 `input_files` 会自动下载为容器本地路径并注入 `args["input_files"]`。不要把 bucket/object_key 文件引用当本地路径读，也不要在 Python 里猜对象存储 URL。

## 输出文件模式

涉及输出文件的 Go 文件需要引入 `path/filepath`。

```go
func GenerateImage(ctx *app.Context, resp response.Response) error {
	var req GenerateImageReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	outputPath := filepath.Join(outputDir, "result.png")
	outputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return err
	}

	executor := pythonRuntime.NewExecutor(generateImagePythonCode()).
		WithRequest(map[string]any{"title": req.Title, "output_path": outputPath}).
		WithOutputDir(outputDir).
		WithTimeout(45 * time.Second)
	defer func() { _ = executor.Close() }()

	var result struct {
		Summary string `json:"summary"`
	}
	execResult, err := executor.ExecuteJSONWithResult(ctx, &result)
	if err != nil {
		return fmt.Errorf("[系统错误]-[GenerateImage] Python 执行失败: %w", err)
	}
	paths, err := execResult.OutputFilePaths()
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("Python 未声明输出文件")
	}

	return resp.Form(&GenerateImageResp{
		OutputFile: fs.ResponseFiles(paths),
		Summary:    result.Summary,
	}).Build()
}
```

对应 Python：

```python
def kageos_entry(args, output_dir):
    import os
    output_path = args["output_path"]
    if not os.path.isabs(output_path):
        raise ValueError("output_path 必须是绝对路径")
    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    with open(output_path, "w", encoding="utf-8") as f:
        f.write(args.get("title", "result"))
    return {
        "data": {"summary": "文件已生成"},
        "output_files": [{"path": output_path, "name": os.path.basename(output_path)}]
    }
```

## 输入输出一体化模式

处理上传文件并返回新文件时，输入文件走 `DownloadFiles`，输出文件写到 `GetTraceOutputDir()`，Go 传给 Python 的路径都使用绝对路径。

```go
type ProcessFileResp struct {
	OutputFiles string `json:"output_files" widget:"name:输出文件;type:files"`
	Summary     string `json:"summary" widget:"name:摘要;type:text_area"`
}

func ProcessFile(ctx *app.Context, resp response.Response) error {
	var req ProcessFileReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	fs := ctx.GetFS()
	inputPaths := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputPaths)
	if len(inputPaths) == 0 {
		return fmt.Errorf("没有找到输入文件")
	}
	for i, path := range inputPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		inputPaths[i] = absPath
	}

	outputDir := fs.GetTraceOutputDir()
	outputPath, err := filepath.Abs(filepath.Join(outputDir, "result.txt"))
	if err != nil {
		return err
	}

	executor := pythonRuntime.NewExecutor(processPythonCode()).
		WithRequest(map[string]any{
			"input_files": inputPaths,
			"output_path": outputPath,
		}).
		WithOutputDir(outputDir).
		WithTimeout(60 * time.Second)
	defer func() { _ = executor.Close() }()

	var result struct {
		Summary string `json:"summary"`
	}
	execResult, err := executor.ExecuteJSONWithResult(ctx, &result)
	if err != nil {
		return fmt.Errorf("[系统错误]-[ProcessFile] Python 执行失败: %w", err)
	}
	paths, err := execResult.OutputFilePaths()
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("Python 未声明输出文件")
	}

	return resp.Form(&ProcessFileResp{
		OutputFiles: fs.ResponseFiles(paths),
		Summary:     result.Summary,
	}).Build()
}
```

对应 Python：

```python
def kageos_entry(args, output_dir):
    import os

    input_files = args.get("input_files") or []
    output_path = args["output_path"]
    if not input_files:
        raise ValueError("没有找到输入文件")
    if not os.path.isabs(output_path):
        raise ValueError("output_path 必须是绝对路径")

    blocks = []
    for path in input_files:
        with open(path, "r", encoding="utf-8", errors="ignore") as f:
            blocks.append(f.read())

    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    with open(output_path, "w", encoding="utf-8") as f:
        f.write("\n\n".join(blocks))

    return {
        "data": {"summary": f"处理输入文件 {len(input_files)} 个"},
        "output_files": [{"path": output_path, "name": os.path.basename(output_path)}]
    }
```

## 关键注意事项

- 创建 executor 后立即 `defer executor.Close()`，避免默认临时目录泄漏。
- Go 和 Python 是同机子进程，但 cwd 可能不同；Go 传给 Python 的文件路径使用绝对路径。
- 输入文件字段给 Go 的是平台文件引用，不是本地路径；业务 app 里必须先 `ctx.GetFS().DownloadFiles(...)`，用完 `defer fs.RemoveFiles(inputFiles)`。
- 输出文件用 `WithOutputDir` 限定目录，再用 `OutputFilePaths()` 校验，最后 `ctx.GetFS().ResponseFiles(...)` 下发。
- 大文件不要 base64 放进 `data`；产物走 `output_files`。
- `WithPackages` 可安装包，但长期依赖最好进运行镜像或系统工具能力，避免每次执行安装带来不稳定。
- `/system/tools/runtime/python.form` 的顶层 `input_files` 支持用户上传附件或上一步 `output_files` 返回的文件引用；Python 里读取注入的 `args["input_files"]` 本地路径。
