package python

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)


// Executor Python 代码执行器（Builder 模式）
type Executor struct {
	code       string
	request    interface{} // 请求结构体（会序列化为 JSON）
	packages   []string
	timeout    time.Duration
	workDir    string
	pythonPath string
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

// WithPythonPath 设置 Python 解释器路径
// pythonPath: Python 解释器路径（如果为空，则自动检测）
func (e *Executor) WithPythonPath(pythonPath string) *Executor {
	e.pythonPath = pythonPath
	return e
}

// Execute 执行 Python 代码，返回原始输出
// ctx: 上下文（用于超时控制）
// 返回: 执行输出和错误
func (e *Executor) Execute(ctx context.Context) ([]byte, error) {
	// 创建带超时的上下文
	if e.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.timeout)
		defer cancel()
	}

	// 1. 检测 Python 路径
	pythonPath := e.detectPythonPath()
	if pythonPath == "" {
		return nil, fmt.Errorf("未找到 Python 解释器，请确保已安装 Python 3")
	}

	// 2. 创建工作目录
	workDir, err := e.createWorkDir()
	if err != nil {
		return nil, fmt.Errorf("创建工作目录失败: %w", err)
	}
	defer func() {
		// 清理临时目录（如果是临时创建的）
		if e.workDir == "" {
			os.RemoveAll(workDir)
		}
	}()

	// 3. 安装依赖包（如果需要）
	if len(e.packages) > 0 {
		if err := e.installPackages(ctx, workDir, pythonPath); err != nil {
			logger.Warnf(ctx, "[Python] 安装包失败: %v", err)
			// 不阻止执行，继续运行
		}
	}

	// 4. 生成 Python 包装脚本
	wrapperScript := e.buildWrapperScript()
	scriptPath := filepath.Join(workDir, "script.py")
	if err := os.WriteFile(scriptPath, []byte(wrapperScript), 0644); err != nil {
		return nil, fmt.Errorf("写入 Python 脚本失败: %w", err)
	}

	// 5. 构建执行命令
	// 将请求结构体序列化为 JSON
	var requestJSON []byte
	if e.request != nil {
		var err error
		requestJSON, err = json.Marshal(e.request)
		if err != nil {
			return nil, fmt.Errorf("序列化请求结构体失败: %w", err)
		}
	} else {
		// 如果没有请求，传递空对象
		requestJSON = []byte("{}")
	}

	cmd := exec.CommandContext(ctx, pythonPath, scriptPath, string(requestJSON))
	cmd.Dir = workDir

	// 6. 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("执行 Python 脚本失败: %w, 输出: %s", err, string(output))
	}

	return output, nil
}

// ExecuteJSON 执行 Python 代码，自动解析 JSON 输出到 result
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
func (e *Executor) ExecuteJSON(ctx context.Context, result interface{}) error {
	output, err := e.Execute(ctx)
	if err != nil {
		return err
	}

	// 尝试解析 JSON 输出
	outputStr := string(output)

	// 从标记中提取 JSON
	jsonStr, err := e.extractJSONFromOutput(outputStr)
	if err != nil {
		return fmt.Errorf("Python 输出中未找到 JSON 数据: %w, 输出: %s", err, outputStr)
	}

	if err := json.Unmarshal([]byte(jsonStr), result); err != nil {
		return fmt.Errorf("解析 Python JSON 输出失败: %w, JSON 字符串: %s", err, jsonStr)
	}

	return nil
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
		"py-pdf2":     "PyPDF2",
		"pypdf2":      "PyPDF2",
		"pdfplumber":  "pdfplumber",
		"openpyxl":    "openpyxl",
		// OCR（光学字符识别）
		"easyocr":      "easyocr",
		"paddleocr":    "paddleocr", // 如果使用 Python 3.11 可以安装
		"paddlepaddle": "paddle",    // 如果使用 Python 3.11 可以安装
		// 数据科学
		"pandas": "pandas",
		"numpy":  "numpy",
		"scipy":  "scipy",
		// 数据可视化
		"matplotlib": "matplotlib",
		"seaborn":    "seaborn",
		// NLP
		"jieba": "jieba",
		// HTTP
		"requests": "requests",
		// 其他常用包
		"beautifulsoup4": "bs4",
		"scikit-learn":   "sklearn",
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
func (e *Executor) buildWrapperScript() string {
	wrapper := `import sys
import json
import traceback

# 解析 JSON 请求结构体
request = {}
if len(sys.argv) > 1:
    try:
        request = json.loads(sys.argv[1])
    except Exception as e:
        print(f"请求解析错误: {e}", file=sys.stderr)
        sys.exit(1)

# 辅助函数：输出 JSON（带标记，便于 Go 端解析）
def output_json(data):
    """输出 JSON 数据，带标记以便 Go 端解析"""
    json_str = json.dumps(data, ensure_ascii=False)
    print("<python-out>")
    print(json_str)
    print("</python-out>")

# 将请求字段注入到全局命名空间（方便直接使用）
# 注意：JSON 的 true/false/null 会被 json.loads() 自动转换为 Python 的 True/False/None
if isinstance(request, dict):
    for key, value in request.items():
        globals()[key] = value

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
try:
`
	// 添加用户代码，每行缩进 4 个空格
	lines := strings.Split(e.code, "\n")
	for _, line := range lines {
		wrapper += "    " + line + "\n"
	}

	wrapper += `except Exception as e:
    print(f"执行错误: {e}", file=sys.stderr)
    traceback.print_exc()
    sys.exit(1)
`

	return wrapper
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
