package python

import (
	"bytes"
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

type pythonExecutionWorkspace struct {
	pythonPath string
	workDir    string
	outputDir  string
	resultPath string
	scriptPath string
}

type pythonRunOutput struct {
	stdout     string
	stderr     string
	durationMs int64
	runErr     error
}

// ExecuteResult 执行 Python 代码，返回结构化结果与输出文件声明。
//
// Python 代码必须定义固定入口函数:
//
//	def agentos_entry(args, output_dir): ...
//
// 其中:
// - args: Go 侧 WithRequest 传入的对象（dict）
// - output_dir: 受控输出目录，需将最终文件写到此目录
//
// 返回值必须为 dict，支持字段:
// - data: 任意 JSON 可序列化对象
// - output_files: []{path,name?,description?}
// - warnings: []string
func (e *Executor) ExecuteResult(ctx context.Context) (*ExecutionResult, error) {
	if err := e.ensureExecutable(); err != nil {
		return nil, err
	}

	ctx, cancel := e.executionContext(ctx)
	defer cancel()

	workspace, err := e.prepareExecutionWorkspace(ctx)
	if err != nil {
		return nil, err
	}

	output, err := e.runExecutionCommand(ctx, workspace)
	if err != nil {
		return nil, err
	}

	result := newExecutionResult(workspace, output)
	e.applyExecutionPayload(result, workspace.resultPath, output.runErr)
	return finishExecutionResult(result, output.runErr)
}

func (e *Executor) ensureExecutable() error {
	if e.closed {
		return fmt.Errorf("python executor: already closed")
	}
	return nil
}

func (e *Executor) executionContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if e.timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, e.timeout)
}

func (e *Executor) prepareExecutionWorkspace(ctx context.Context) (*pythonExecutionWorkspace, error) {
	pythonPath := e.detectPythonPath()
	if pythonPath == "" {
		return nil, fmt.Errorf("未找到 Python 解释器，请确保已安装 Python 3")
	}

	workDir, err := e.createExecutionWorkDir()
	if err != nil {
		return nil, err
	}

	if len(e.packages) > 0 {
		if err := e.installPackages(ctx, workDir, pythonPath); err != nil {
			logger.Warnf(ctx, "[Python] 安装包失败: %v", err)
		}
	}

	outputDir, err := e.resolveOutputDir(workDir)
	if err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	scriptPath, err := e.writeWrapperScript(workDir)
	if err != nil {
		return nil, err
	}

	return &pythonExecutionWorkspace{
		pythonPath: pythonPath,
		workDir:    workDir,
		outputDir:  outputDir,
		resultPath: filepath.Join(workDir, ".agentos", "result.json"),
		scriptPath: scriptPath,
	}, nil
}

func (e *Executor) createExecutionWorkDir() (string, error) {
	workDir, err := e.createWorkDir()
	if err != nil {
		return "", fmt.Errorf("创建工作目录失败: %w", err)
	}
	if e.workDir == "" {
		e.managedTempDirs = append(e.managedTempDirs, workDir)
	}
	return workDir, nil
}

func (e *Executor) writeWrapperScript(workDir string) (string, error) {
	wrapperScript, err := e.buildWrapperScript()
	if err != nil {
		return "", fmt.Errorf("生成 Python 包装脚本失败: %w", err)
	}
	scriptPath := filepath.Join(workDir, "script.py")
	if err := os.WriteFile(scriptPath, []byte(wrapperScript), 0644); err != nil {
		return "", fmt.Errorf("写入 Python 脚本失败: %w", err)
	}
	return scriptPath, nil
}

func (e *Executor) runExecutionCommand(ctx context.Context, workspace *pythonExecutionWorkspace) (*pythonRunOutput, error) {
	requestJSON, err := e.executionRequestJSON()
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, workspace.pythonPath, workspace.scriptPath, string(requestJSON))
	cmd.Dir = workspace.workDir
	cmd.Env = append(os.Environ(),
		"AGENTOS_OUTPUT_DIR="+workspace.outputDir,
		"AGENTOS_RESULT_PATH="+workspace.resultPath,
	)

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	start := time.Now()
	runErr := cmd.Run()
	return &pythonRunOutput{
		stdout:     stdoutBuf.String(),
		stderr:     stderrBuf.String(),
		durationMs: time.Since(start).Milliseconds(),
		runErr:     runErr,
	}, nil
}

func (e *Executor) executionRequestJSON() ([]byte, error) {
	if e.request == nil {
		return []byte("{}"), nil
	}

	requestJSON, err := json.Marshal(e.request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求结构体失败: %w", err)
	}
	return requestJSON, nil
}

func newExecutionResult(workspace *pythonExecutionWorkspace, output *pythonRunOutput) *ExecutionResult {
	return &ExecutionResult{
		Stdout:     output.stdout,
		Stderr:     output.stderr,
		WorkDir:    workspace.workDir,
		OutputDir:  workspace.outputDir,
		DurationMs: output.durationMs,
	}
}

func (e *Executor) applyExecutionPayload(result *ExecutionResult, resultPath string, runErr error) {
	if manifest, err := e.readExecutionManifest(resultPath); err == nil && manifest != nil {
		applyManifestToResult(result, manifest)
		return
	}
	e.applyFallbackJSON(result, runErr)
}

func applyManifestToResult(result *ExecutionResult, manifest *executionManifest) {
	result.OK = manifest.OK
	result.OutputFiles = manifest.OutputFiles
	result.Warnings = manifest.Warnings
	result.Error = manifest.Error
	if len(manifest.Data) == 0 {
		return
	}

	var data interface{}
	if err := json.Unmarshal(manifest.Data, &data); err == nil {
		result.Data = data
	}
}

func (e *Executor) applyFallbackJSON(result *ExecutionResult, runErr error) {
	fallbackJSON, err := e.extractJSONFromOutput(result.Stdout)
	if err != nil {
		return
	}

	var data interface{}
	if json.Unmarshal([]byte(fallbackJSON), &data) == nil {
		result.Data = data
		result.OK = runErr == nil
	}
}

func finishExecutionResult(result *ExecutionResult, runErr error) (*ExecutionResult, error) {
	if runErr != nil {
		if result.Error == "" {
			if strings.TrimSpace(result.Stderr) != "" {
				result.Error = strings.TrimSpace(result.Stderr)
			} else {
				result.Error = runErr.Error()
			}
		}
		return result, fmt.Errorf("执行 Python 脚本失败: %w", runErr)
	}
	if result.Error == "" {
		result.OK = true
	}
	return result, nil
}
