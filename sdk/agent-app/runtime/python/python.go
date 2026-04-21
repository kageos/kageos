package python

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type PythonArtifact struct {
	Path        string `json:"path"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

const DefaultEntryFunctionName = "agentos_entry"

type ExecutionResult struct {
	OK          bool             `json:"ok"`
	Stdout      string           `json:"stdout,omitempty"`
	Stderr      string           `json:"stderr,omitempty"`
	Data        interface{}      `json:"data,omitempty"`
	OutputFiles []PythonArtifact `json:"output_files,omitempty"`
	Warnings    []string         `json:"warnings,omitempty"`
	Error       string           `json:"error,omitempty"`
	WorkDir     string           `json:"work_dir,omitempty"`
	OutputDir   string           `json:"output_dir,omitempty"`
	DurationMs  int64            `json:"duration_ms,omitempty"`
}

type executionManifest struct {
	OK          bool             `json:"ok"`
	Data        json.RawMessage  `json:"data,omitempty"`
	OutputFiles []PythonArtifact `json:"output_files,omitempty"`
	Warnings    []string         `json:"warnings,omitempty"`
	Error       string           `json:"error,omitempty"`
}

// Executor Python 代码执行器（Builder 模式）
type Executor struct {
	code       string
	request    interface{} // 请求结构体（会序列化为 JSON）
	packages   []string
	timeout    time.Duration
	workDir    string
	outputDir  string
	pythonPath string

	// managedTempDirs 由 MkdirTemp 创建的工作目录，由 Close 负责 RemoveAll（WithWorkDir 指定的目录不会加入此列表）
	managedTempDirs []string
	closed          bool
}

// NewExecutor 创建新的 Python 执行器
// code: Python 代码字符串
func NewExecutor(code string) *Executor {
	return &Executor{
		code:       code,
		request:    nil,
		packages:   []string{},
		timeout:    5 * time.Minute, // 默认超时 5 分钟
		pythonPath: "",              // 自动检测
	}
}

// WithRequest 设置请求结构体（会序列化为 JSON 传递给 Python）
// req: 请求结构体（必须是可 JSON 序列化的类型）
//
// 示例:
//
//	type Request struct {
//	    Name string `json:"name"`
//	    Age  int    `json:"age"`
//	}
//
//	req := Request{Name: "Alice", Age: 30}
//	executor := python.NewExecutor(code).WithRequest(req)
//	defer executor.Close()
func (e *Executor) WithRequest(req interface{}) *Executor {
	e.request = req
	return e
}

// WithPackages 设置需要安装的 Python 包
// packages: 包名列表（例如: []string{"pandas", "numpy"}）
func (e *Executor) WithPackages(packages ...string) *Executor {
	e.packages = append(e.packages, packages...)
	return e
}

// WithTimeout 设置执行超时时间
// timeout: 超时时间（例如: 2 * time.Minute）
func (e *Executor) WithTimeout(timeout time.Duration) *Executor {
	e.timeout = timeout
	return e
}

// WithWorkDir 设置工作目录
// workDir: 工作目录路径（如果为空，则使用临时目录）
func (e *Executor) WithWorkDir(workDir string) *Executor {
	e.workDir = workDir
	return e
}

// WithOutputDir 设置产物输出目录。
// 若传相对路径，则会在实际工作目录下解析；为空时默认使用 "<workDir>/output"。
func (e *Executor) WithOutputDir(outputDir string) *Executor {
	e.outputDir = outputDir
	return e
}

// WithPythonPath 设置 Python 解释器路径
// pythonPath: Python 解释器路径（如果为空，则自动检测）
func (e *Executor) WithPythonPath(pythonPath string) *Executor {
	e.pythonPath = pythonPath
	return e
}

// Close 释放本 Executor 在默认临时工作目录模式下创建的目录（os.MkdirTemp）。
//
// 未调用 WithWorkDir 时，Execute/ExecuteJSON 会在系统临时目录下创建工作区；用完后必须调用 Close，否则会泄漏磁盘。
// 推荐在创建 Executor 后立即：defer executor.Close()。
//
// WithWorkDir 指向的目录不会被删除，仅删除 SDK 内部创建的临时目录。
// Close 可安全多次调用（幂等）。Close 之后不得再调用 Execute / ExecuteJSON。
//
// 同一 Executor 在未 Close 前多次 Execute（且未使用 WithWorkDir）时，每次都会新建临时目录；一次 Close 会删除记录的全部此类目录。
func (e *Executor) Close() error {
	if e.closed {
		return nil
	}
	e.closed = true
	var errs []error
	for _, dir := range e.managedTempDirs {
		if err := os.RemoveAll(dir); err != nil {
			errs = append(errs, fmt.Errorf("python executor: remove temp dir %q: %w", dir, err))
		}
	}
	e.managedTempDirs = nil
	return errors.Join(errs...)
}

// Execute 执行 Python 代码，返回原始输出
// ctx: 上下文（用于超时控制）
// 返回: 执行输出和错误
//
// 使用默认临时工作目录（未 WithWorkDir）时，须在适当时机调用 Close() 释放磁盘（推荐创建 Executor 后 defer Close()）。
func (e *Executor) Execute(ctx context.Context) ([]byte, error) {
	result, err := e.ExecuteResult(ctx)
	if result == nil {
		return nil, err
	}
	return []byte(combineStdoutStderr(result.Stdout, result.Stderr)), err
}

// ExecuteJSON 执行 Python 代码，自动解析 agentos_entry 返回的 data 字段到 result
// ctx: 上下文（用于超时控制）
// result: 结果结构体指针（必须是可 JSON 反序列化的类型）
// 返回: 错误
//
// 示例:
//
//	var result struct {
//	    Sum int `json:"sum"`
//	}
//	err := executor.ExecuteJSON(ctx, &result)
//
// 使用默认临时工作目录时须配合 Close()，见 Execute。
func (e *Executor) ExecuteJSON(ctx context.Context, result interface{}) error {
	_, err := e.ExecuteJSONWithResult(ctx, result)
	return err
}

// ExecuteJSONWithResult 执行 Python 代码，解析结构化结果并返回完整执行信息。
func (e *Executor) ExecuteJSONWithResult(ctx context.Context, result interface{}) (*ExecutionResult, error) {
	execResult, err := e.ExecuteResult(ctx)
	if err != nil {
		return execResult, err
	}

	if execResult == nil || execResult.Data == nil {
		return execResult, fmt.Errorf("Python 执行结果中没有结构化 data")
	}

	dataBytes, marshalErr := json.Marshal(execResult.Data)
	if marshalErr != nil {
		return execResult, fmt.Errorf("序列化 Python data 失败: %w", marshalErr)
	}
	if err := json.Unmarshal(dataBytes, result); err != nil {
		return execResult, fmt.Errorf("解析 Python data 失败: %w, data: %s", err, string(dataBytes))
	}
	return execResult, nil
}

// extractJSONFromOutput 从输出中提取 JSON（使用标记 <python-out>...</python-out>）
func (e *Executor) extractJSONFromOutput(output string) (string, error) {
	startMarker := "<python-out>"
	endMarker := "</python-out>"

	startIdx := strings.Index(output, startMarker)
	if startIdx == -1 {
		return "", fmt.Errorf("未找到开始标记 %s", startMarker)
	}

	endIdx := strings.Index(output, endMarker)
	if endIdx == -1 {
		return "", fmt.Errorf("未找到结束标记 %s", endMarker)
	}

	if endIdx <= startIdx {
		return "", fmt.Errorf("结束标记在开始标记之前")
	}

	// 提取 JSON 字符串（去除标记和换行）
	jsonStart := startIdx + len(startMarker)
	jsonEnd := endIdx
	jsonStr := strings.TrimSpace(output[jsonStart:jsonEnd])

	if jsonStr == "" {
		return "", fmt.Errorf("标记之间没有内容")
	}

	return jsonStr, nil
}

func combineStdoutStderr(stdout string, stderr string) string {
	stdout = strings.TrimRight(stdout, "\n")
	stderr = strings.TrimRight(stderr, "\n")
	switch {
	case stdout == "":
		return stderr
	case stderr == "":
		return stdout
	default:
		return stdout + "\n" + stderr
	}
}

// CombinedOutput 返回 stdout/stderr 合并后的文本。
func (r *ExecutionResult) CombinedOutput() string {
	if r == nil {
		return ""
	}
	return combineStdoutStderr(r.Stdout, r.Stderr)
}

// FormattedData 返回 data 的格式化 JSON 字符串，便于直接展示。
func (r *ExecutionResult) FormattedData() string {
	if r == nil || r.Data == nil {
		return ""
	}
	b, err := json.MarshalIndent(r.Data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", r.Data)
	}
	return string(b)
}

// OutputFilePaths 校验并返回 output_files 的绝对路径列表。
// 若设置了 OutputDir，则所有输出文件都必须位于该目录内。
func (r *ExecutionResult) OutputFilePaths() ([]string, error) {
	if r == nil || len(r.OutputFiles) == 0 {
		return nil, nil
	}

	outputDir := strings.TrimSpace(r.OutputDir)
	if outputDir != "" {
		outputDir = filepath.Clean(outputDir)
	}

	paths := make([]string, 0, len(r.OutputFiles))
	for _, file := range r.OutputFiles {
		path := strings.TrimSpace(file.Path)
		if path == "" {
			return nil, fmt.Errorf("python output_files.path 不能为空")
		}
		if !filepath.IsAbs(path) {
			return nil, fmt.Errorf("python output_files.path 必须是绝对路径: %s", path)
		}

		cleanPath := filepath.Clean(path)
		if outputDir != "" {
			rel, err := filepath.Rel(outputDir, cleanPath)
			if err != nil {
				return nil, fmt.Errorf("校验 python 输出路径失败 %s: %w", cleanPath, err)
			}
			if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				return nil, fmt.Errorf("python 输出文件必须位于 output_dir 内: %s", cleanPath)
			}
		}

		info, err := os.Stat(cleanPath)
		if err != nil {
			return nil, fmt.Errorf("python 输出文件不存在: %s: %w", cleanPath, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("python 输出路径不是普通文件: %s", cleanPath)
		}

		paths = append(paths, cleanPath)
	}

	return paths, nil
}

func (e *Executor) resolveOutputDir(workDir string) (string, error) {
	outputDir := strings.TrimSpace(e.outputDir)
	if outputDir == "" {
		outputDir = filepath.Join(workDir, "output")
	} else if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(workDir, outputDir)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", err
	}
	return outputDir, nil
}

func (e *Executor) readExecutionManifest(resultPath string) (*executionManifest, error) {
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return nil, err
	}
	var manifest executionManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// createWorkDir 创建工作目录
func (e *Executor) createWorkDir() (string, error) {
	if e.workDir != "" {
		// 确保目录存在
		if err := os.MkdirAll(e.workDir, 0755); err != nil {
			return "", err
		}
		return e.workDir, nil
	}

	// 创建临时目录
	return os.MkdirTemp("", "python-exec-*")
}

// installPackages 安装 Python 包
// 优化策略：三步检查机制，快速跳过已安装的包
// 1. 环境变量快速检查（O(1) 查找，< 0.001 秒）
// 2. 导入检查（< 0.1 秒）
// 3. pip 安装（仅在未安装时执行）
func (e *Executor) installPackages(ctx context.Context, workDir, pythonPath string) error {
	// 获取已安装包列表（从环境变量，一次性读取）
	installedPackages := e.getInstalledPackages()

	for _, pkg := range e.packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}

		// 提取包名（去除版本号，如 "pandas==1.5.0" -> "pandas"）
		// 支持格式：pandas, pandas==1.5.0, pandas>=1.5.0, pandas~=1.5.0
		pkgName := e.extractPackageName(pkg)

		// 第一步：环境变量快速检查（最快，O(1) 查找）
		if e.isPackageInstalled(pkgName, installedPackages) {
			logger.Infof(ctx, "[Python] 包 %s 已安装（环境变量检查），跳过", pkgName)
			continue
		}

		// 第二步：尝试导入包（快速验证，< 0.1 秒）
		// 处理包名映射（例如：Pillow -> PIL, opencv-python -> cv2）
		importName := e.mapPackageToImport(pkgName)
		if e.canImportPackage(ctx, pythonPath, importName, workDir) {
			logger.Infof(ctx, "[Python] 包 %s 已安装（导入检查），跳过", pkgName)
			continue
		}

		// 第三步：包未安装，执行安装
		logger.Infof(ctx, "[Python] 安装包: %s", pkg)
		cmd := exec.CommandContext(ctx, pythonPath, "-m", "pip", "install", "--quiet", "--break-system-packages", pkg)
		cmd.Dir = workDir

		if err := cmd.Run(); err != nil {
			logger.Warnf(ctx, "[Python] 安装包失败: %s, 错误: %v", pkg, err)
			// 不返回错误，继续安装其他包
		} else {
			logger.Infof(ctx, "[Python] 包 %s 安装成功", pkgName)
		}
	}
	return nil
}

// getInstalledPackages 从环境变量获取已安装包列表（一次性读取，避免重复解析）
// 优化：支持从环境变量或文件读取，确保在容器环境中可用
func (e *Executor) getInstalledPackages() map[string]bool {
	installedPackages := make(map[string]bool)

	// 第一步：从环境变量读取（优先）
	envPackages := os.Getenv("PYTHON_INSTALLED_PACKAGES")

	// 第二步：如果环境变量不存在，尝试从文件读取（备用方案）
	if envPackages == "" {
		if data, err := os.ReadFile("/etc/python-installed-packages.txt"); err == nil {
			envPackages = strings.TrimSpace(string(data))
		}
	}

	if envPackages == "" {
		return installedPackages
	}

	// 解析逗号分隔的包列表
	packages := strings.Split(envPackages, ",")
	for _, pkg := range packages {
		pkg = strings.TrimSpace(strings.ToLower(pkg))
		if pkg != "" {
			installedPackages[pkg] = true
		}
	}

	return installedPackages
}

// isPackageInstalled 检查包是否在已安装列表中（快速检查，O(1) 查找）
func (e *Executor) isPackageInstalled(pkgName string, installedPackages map[string]bool) bool {
	return installedPackages[strings.ToLower(pkgName)]
}

// canImportPackage 尝试导入包，检查是否已安装
func (e *Executor) canImportPackage(ctx context.Context, pythonPath, importName, workDir string) bool {
	// 尝试导入
	checkCmd := exec.CommandContext(ctx, pythonPath, "-c", fmt.Sprintf("import %s", importName))
	checkCmd.Dir = workDir
	// 忽略检查命令的输出和错误
	checkCmd.Stdout = nil
	checkCmd.Stderr = nil

	// 设置超时（避免卡住）
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	checkCmd = exec.CommandContext(checkCtx, pythonPath, "-c", fmt.Sprintf("import %s", importName))
	checkCmd.Dir = workDir
	checkCmd.Stdout = nil
	checkCmd.Stderr = nil

	return checkCmd.Run() == nil
}

// mapPackageToImport 将包名映射到导入名
// 例如：Pillow -> PIL, opencv-python -> cv2, python-docx -> docx
func (e *Executor) mapPackageToImport(pkgName string) string {
	packageMap := map[string]string{
		// 图像处理
		"pillow":                 "PIL",
		"opencv-python":          "cv2",
		"opencv-python-headless": "cv2",
		// 文档处理
		"python-docx": "docx",
		"python-pptx": "pptx",
		"py-pdf2":     "PyPDF2",
		"pypdf2":      "PyPDF2",
		"pdfplumber":  "pdfplumber",
		"reportlab":   "reportlab",
		"openpyxl":    "openpyxl",
		"xlsxwriter":  "xlsxwriter",
		"xlrd":        "xlrd",
		"xlwt":        "xlwt",
		// OCR（光学字符识别）
		"easyocr":      "easyocr",
		"paddleocr":    "paddleocr", // 如果使用 Python 3.11 可以安装
		"paddlepaddle": "paddle",    // 如果使用 Python 3.11 可以安装
		// 数据科学
		"pandas":  "pandas",
		"numpy":   "numpy",
		"scipy":   "scipy",
		"pymysql": "pymysql",
		// 数据可视化
		"matplotlib": "matplotlib",
		"seaborn":    "seaborn",
		"plotly":     "plotly",
		"pyecharts":  "pyecharts",
		"wordcloud":  "wordcloud",
		// NLP
		"jieba":   "jieba",
		"snownlp": "snownlp",
		// HTTP
		"requests": "requests",
		"aiohttp":  "aiohttp",
		// 其他常用包
		"beautifulsoup4":  "bs4",
		"scikit-learn":    "sklearn",
		"pyyaml":          "yaml",
		"qrcode":          "qrcode",
		"python-barcode":  "barcode",
		"toml":            "toml",
		"tabulate":        "tabulate",
		"arrow":           "arrow",
		"dateutil":        "dateutil",
		"python-dateutil": "dateutil",
		"lxml":            "lxml",
		"cryptography":    "cryptography",
	}

	if importName, ok := packageMap[strings.ToLower(pkgName)]; ok {
		return importName
	}

	// 默认使用包名（去除连字符，转换为下划线）
	return strings.ReplaceAll(pkgName, "-", "_")
}

// extractPackageName 从包规范中提取包名
// 支持格式：pandas, pandas==1.5.0, pandas>=1.5.0, pandas~=1.5.0, pandas[extra]
func (e *Executor) extractPackageName(pkgSpec string) string {
	// 去除版本号部分
	// 支持的操作符：==, >=, <=, >, <, ~=, !=
	operators := []string{"==", ">=", "<=", "~=", "!=", ">", "<"}

	pkgName := pkgSpec
	for _, op := range operators {
		if idx := strings.Index(pkgName, op); idx != -1 {
			pkgName = pkgName[:idx]
			break
		}
	}

	// 去除 extras 部分（如 pandas[excel] -> pandas）
	if idx := strings.Index(pkgName, "["); idx != -1 {
		pkgName = pkgName[:idx]
	}

	// 去除首尾空白
	pkgName = strings.TrimSpace(pkgName)

	return pkgName
}

// buildWrapperScript 构建 Python 包装脚本
func (e *Executor) buildWrapperScript() (string, error) {
	userCode, err := json.Marshal(e.code)
	if err != nil {
		return "", err
	}
	wrapper := `import sys
import json
import traceback
import os

# 解析 JSON 请求结构体
request = {}
if len(sys.argv) > 1:
    try:
        request = json.loads(sys.argv[1])
    except Exception as e:
        print(f"请求解析错误: {e}", file=sys.stderr)
        sys.exit(1)

ENTRY_FUNCTION_NAME = "` + DefaultEntryFunctionName + `"
output_dir = os.environ.get("AGENTOS_OUTPUT_DIR", "")
result_path = os.environ.get("AGENTOS_RESULT_PATH", "")

def output_file(path, name=None, description=None):
    if path is None:
        raise ValueError("output_file path 不能为空")
    abs_path = os.path.abspath(path)
    item = {"path": abs_path}
    if name:
        item["name"] = str(name)
    if description:
        item["description"] = str(description)
    return item

def _normalize_output_file(item):
    if isinstance(item, str):
        item = {"path": item}
    if not isinstance(item, dict):
        raise TypeError("output_files 的每一项必须是 dict 或 path 字符串")
    path = item.get("path")
    if not isinstance(path, str) or not path.strip():
        raise ValueError("output_files[*].path 必须是非空字符串")
    normalized = {"path": os.path.abspath(path)}
    name = item.get("name")
    if name not in (None, ""):
        normalized["name"] = str(name)
    description = item.get("description")
    if description not in (None, ""):
        normalized["description"] = str(description)
    return normalized

def _normalize_result(result):
    if result is None:
        result = {}
    if not isinstance(result, dict):
        raise TypeError(f"{ENTRY_FUNCTION_NAME}(args, output_dir) 必须返回 dict")
    allowed_keys = {"data", "output_files", "warnings"}
    unknown_keys = sorted(set(result.keys()) - allowed_keys)
    if unknown_keys:
        raise ValueError(f"{ENTRY_FUNCTION_NAME} 返回了不支持的字段: {', '.join(unknown_keys)}")
    output_files_raw = result.get("output_files") or []
    if not isinstance(output_files_raw, list):
        raise TypeError("output_files 必须是 list")
    warnings_raw = result.get("warnings") or []
    if not isinstance(warnings_raw, list):
        raise TypeError("warnings 必须是 list")
    return {
        "data": result.get("data"),
        "output_files": [_normalize_output_file(item) for item in output_files_raw],
        "warnings": [str(item) for item in warnings_raw],
    }

def _write_manifest(ok, normalized_result=None, error_message=""):
    payload = {
        "ok": bool(ok),
        "data": None,
        "output_files": [],
        "warnings": [],
        "error": str(error_message or ""),
    }
    if normalized_result is not None:
        payload["data"] = normalized_result.get("data")
        payload["output_files"] = normalized_result.get("output_files", [])
        payload["warnings"] = normalized_result.get("warnings", [])
    if not result_path:
        return
    result_dir = os.path.dirname(result_path)
    if result_dir:
        os.makedirs(result_dir, exist_ok=True)
    with open(result_path, "w", encoding="utf-8") as fh:
        json.dump(payload, fh, ensure_ascii=False, default=str)

# 为了兼容性，提供 true/false/null 的别名
true = True
false = False
null = None

# 辅助函数：自动将字典/列表转换为 pandas DataFrame
# 使用方式：df = to_dataframe(data) 或 df = to_dataframe(data, auto=True)
def to_dataframe(data, auto=False):
    """
    将字典或列表转换为 pandas DataFrame
    
    参数:
        data: 要转换的数据（字典或列表）
        auto: 是否自动检测并转换（默认 False，需要显式调用）
    
    返回:
        pandas DataFrame 或原数据（如果转换失败）
    
    示例:
        # 字典（每行一个字典）
        df = to_dataframe([{"a": 1, "b": 2}, {"a": 3, "b": 4}])
        
        # 字典（每列一个列表）
        df = to_dataframe({"a": [1, 3], "b": [2, 4]})
        
        # 自动转换（如果 data 是字典或列表）
        df = to_dataframe(data, auto=True)
    """
    try:
        import pandas as pd
        if isinstance(data, dict):
            # 字典：尝试转换为 DataFrame
            # 如果字典的值是列表（每列一个列表），直接转换
            if all(isinstance(v, list) for v in data.values()):
                return pd.DataFrame(data)
            # 如果字典的值是标量（单行数据），包装成列表
            elif all(not isinstance(v, (list, dict)) for v in data.values()):
                return pd.DataFrame([data])
            # 其他情况：尝试直接转换
            else:
                return pd.DataFrame(data)
        elif isinstance(data, list):
            # 列表：如果元素是字典，转换为 DataFrame
            if len(data) > 0 and isinstance(data[0], dict):
                return pd.DataFrame(data)
            # 其他情况：尝试直接转换
            else:
                return pd.DataFrame(data)
        else:
            return data  # 不是字典或列表，返回原值
    except ImportError:
        # pandas 未安装，返回原值
        return data
    except Exception:
        # 转换失败，返回原值
        return data

# 自动转换：如果请求中的字段名是常见的 DataFrame 变量名，且值是字典/列表，自动转换
# 常见的 DataFrame 变量名
dataframe_var_names = ['df', 'data', 'dataframe', 'df_data', 'table', 'table_data']
try:
    import pandas as pd
    if isinstance(request, dict):
        for key, value in request.items():
            # 如果变量名是常见的 DataFrame 变量名，且值是字典或列表
            if key.lower() in dataframe_var_names and isinstance(value, (dict, list)):
                try:
                    # 尝试转换为 DataFrame
                    if isinstance(value, dict):
                        if all(isinstance(v, list) for v in value.values()):
                            globals()[key] = pd.DataFrame(value)
                        elif all(not isinstance(v, (list, dict)) for v in value.values()):
                            globals()[key] = pd.DataFrame([value])
                        else:
                            globals()[key] = pd.DataFrame(value)
                    elif isinstance(value, list) and len(value) > 0 and isinstance(value[0], dict):
                        globals()[key] = pd.DataFrame(value)
                except Exception:
                    pass  # 转换失败，保持原值
except ImportError:
    pass  # pandas 未安装，跳过自动转换

# 执行用户代码
user_code = ` + string(userCode) + `
try:
    compiled = compile(user_code, "<agentos-user-code>", "exec")
    exec(compiled, globals(), globals())
    entry = globals().get(ENTRY_FUNCTION_NAME)
    if not callable(entry):
        raise RuntimeError(f"python_code 必须定义函数 {ENTRY_FUNCTION_NAME}(args, output_dir)")
    normalized_result = _normalize_result(entry(request, output_dir))
    _write_manifest(True, normalized_result)
except Exception as e:
    _write_manifest(False, None, e)
    print(f"执行错误: {e}", file=sys.stderr)
    traceback.print_exc()
    sys.exit(1)
`

	return wrapper, nil
}

// detectPythonPath 检测 Python 解释器路径
func (e *Executor) detectPythonPath() string {
	// 1. 如果已设置，直接使用
	if e.pythonPath != "" {
		return e.pythonPath
	}

	// 2. 检查环境变量
	if path := os.Getenv("PYTHON_PATH"); path != "" {
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
	}

	// 3. 尝试多个可能的 Python 路径
	possiblePaths := []string{
		"/usr/bin/python3",
		"/usr/local/bin/python3",
		"python3",
		"python",
	}

	for _, path := range possiblePaths {
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
	}

	// 4. 默认返回 python3（假设在 PATH 中）
	return "python3"
}
