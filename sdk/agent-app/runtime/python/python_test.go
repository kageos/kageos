package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func entryCode(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		body = "return {}"
	}
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = ""
			continue
		}
		lines[i] = "    " + line
	}
	return "def " + DefaultEntryFunctionName + "(args, output_dir):\n" + strings.Join(lines, "\n") + "\n"
}

func TestExecutor_Execute(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		code    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "简单计算",
			code: entryCode(`
result = args["a"] + args["b"]
print(f"结果: {result}")
return {"data": {"sum": result}}
`),
			args: map[string]interface{}{
				"a": 10,
				"b": 20,
			},
			wantErr: false,
		},
		{
			name: "JSON 输出",
			code: entryCode(`
import json
result = {"sum": args["a"] + args["b"], "product": args["a"] * args["b"]}
print(json.dumps(result))
return {"data": result}
`),
			args: map[string]interface{}{
				"a": 10,
				"b": 20,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(tt.code).
				WithRequest(tt.args).
				WithTimeout(30 * time.Second)
			defer func() { _ = executor.Close() }()

			output, err := executor.Execute(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				t.Logf("输出: %s", string(output))
			}
		})
	}
}

func TestExecutor_ExecuteJSON(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		code    string
		args    map[string]interface{}
		wantErr bool
	}{
		{
			name: "解析 JSON 结果",
			code: entryCode(`
return {
    "data": {
        "sum": args["a"] + args["b"],
        "product": args["a"] * args["b"],
    },
}
`),
			args: map[string]interface{}{
				"a": 10,
				"b": 20,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result struct {
				Sum     int `json:"sum"`
				Product int `json:"product"`
			}

			executor := NewExecutor(tt.code).
				WithRequest(tt.args).
				WithTimeout(30 * time.Second)
			defer func() { _ = executor.Close() }()

			err := executor.ExecuteJSON(ctx, &result)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				t.Logf("结果: %+v", result)
				if result.Sum != 30 || result.Product != 200 {
					t.Errorf("结果不正确: sum=%d, product=%d", result.Sum, result.Product)
				}
			}
		})
	}
}

func TestExecutor_WithPackages(t *testing.T) {
	ctx := context.Background()

	// 测试安装包（如果系统有 pandas）
	code := entryCode(`
import pandas as pd

df = pd.DataFrame([{"a": 1, "b": 2}])
result = {"rows": len(df), "columns": df.columns.tolist()}
return {"data": result}
`)

	executor := NewExecutor(code).
		WithPackages("pandas").
		WithTimeout(2 * time.Minute)
	defer func() { _ = executor.Close() }()

	output, err := executor.Execute(ctx)
	if err != nil {
		t.Logf("执行失败（可能没有 pandas）: %v", err)
		t.Logf("输出: %s", string(output))
		return
	}

	t.Logf("输出: %s", string(output))
}

func TestExecutor_BuilderPattern(t *testing.T) {
	ctx := context.Background()

	// 测试 Builder 模式的链式调用
	code := entryCode(`
result = {"message": f"Hello, {args['name']}!", "count": args["count"]}
return {"data": result}
`)

	var result struct {
		Message string `json:"message"`
		Count   int    `json:"count"`
	}

	// 使用结构化的请求
	request := map[string]interface{}{
		"name":  "World",
		"count": 42,
	}

	executor := NewExecutor(code).
		WithRequest(request).
		WithTimeout(30 * time.Second)
	defer func() { _ = executor.Close() }()

	err := executor.ExecuteJSON(ctx, &result)
	if err != nil {
		t.Fatalf("ExecuteJSON() error = %v", err)
	}

	if result.Message != "Hello, World!" || result.Count != 42 {
		t.Errorf("结果不正确: message=%s, count=%d", result.Message, result.Count)
	}

	t.Logf("结果: %+v", result)
}

func TestExecutor_CloseKeepsWithWorkDir(t *testing.T) {
	ctx := context.Background()
	workDir, err := os.MkdirTemp("", "python-exec-workdir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workDir)

	code := entryCode(`
return {"data": {"ok": true}}
`)
	var result struct {
		OK bool `json:"ok"`
	}
	ex := NewExecutor(code).WithWorkDir(workDir).WithTimeout(30 * time.Second)
	defer func() { _ = ex.Close() }()

	if err := ex.ExecuteJSON(ctx, &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatal("unexpected result")
	}
	if _, err := os.Stat(workDir); err != nil {
		t.Fatalf("WithWorkDir 目录应在 Close 后仍存在: %v", err)
	}
}

func TestExecutor_CloseIdempotent(t *testing.T) {
	ctx := context.Background()
	code := entryCode(`
return {"data": {"a": 1}}
`)
	var parsed struct {
		A int `json:"a"`
	}
	ex := NewExecutor(code).WithTimeout(30 * time.Second)
	if err := ex.ExecuteJSON(ctx, &parsed); err != nil {
		t.Fatal(err)
	}
	if err := ex.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ex.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExecutor_ExecuteAfterClose(t *testing.T) {
	ctx := context.Background()
	ex := NewExecutor(entryCode(`
return {"data": {"x": 1}}
`)).WithTimeout(5 * time.Second)
	_ = ex.Close()
	if _, err := ex.Execute(ctx); err == nil {
		t.Fatal("Close 后 Execute 应返回错误")
	}
}

func TestExecutor_ExecuteResult(t *testing.T) {
	ctx := context.Background()
	ex := NewExecutor(entryCode(`
print("done")
return {
    "data": {
        "sum": args["a"] + args["b"],
        "product": args["a"] * args["b"],
    },
}
`)).WithRequest(map[string]interface{}{
		"a": 6,
		"b": 7,
	}).WithTimeout(30 * time.Second)
	defer func() { _ = ex.Close() }()

	result, err := ex.ExecuteResult(ctx)
	if err != nil {
		t.Fatalf("ExecuteResult() error = %v", err)
	}
	if !result.OK {
		t.Fatalf("expected result.OK=true, got false: %+v", result)
	}
	if result.OutputDir == "" {
		t.Fatal("OutputDir 不能为空")
	}
	if _, err := os.Stat(result.OutputDir); err != nil {
		t.Fatalf("OutputDir 不存在: %v", err)
	}
	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("result.Data 类型错误: %T", result.Data)
	}
	if int(data["sum"].(float64)) != 13 || int(data["product"].(float64)) != 42 {
		t.Fatalf("unexpected data: %+v", data)
	}
}

func TestExecutor_ExecuteResultOutputFiles(t *testing.T) {
	ctx := context.Background()
	ex := NewExecutor(entryCode(`
import os
report_path = os.path.join(output_dir, "report.txt")
with open(report_path, "w", encoding="utf-8") as fh:
    fh.write("hello output file")
return {
    "data": {"file_count": 1},
    "output_files": [
        output_file(report_path, name="report.txt", description="测试产物"),
    ],
}
`)).WithTimeout(30 * time.Second)
	defer func() { _ = ex.Close() }()

	result, err := ex.ExecuteResult(ctx)
	if err != nil {
		t.Fatalf("ExecuteResult() error = %v", err)
	}
	if len(result.OutputFiles) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(result.OutputFiles))
	}
	artifact := result.OutputFiles[0]
	if artifact.Name != "report.txt" {
		t.Fatalf("unexpected artifact name: %+v", artifact)
	}
	if !filepath.IsAbs(artifact.Path) {
		t.Fatalf("artifact path 应为绝对路径: %s", artifact.Path)
	}
	content, err := os.ReadFile(artifact.Path)
	if err != nil {
		t.Fatalf("读取产物失败: %v", err)
	}
	if string(content) != "hello output file" {
		t.Fatalf("unexpected artifact content: %s", string(content))
	}
}

func TestExecutor_ExecuteResultRequiresEntryFunction(t *testing.T) {
	ctx := context.Background()
	ex := NewExecutor(`
print("hello")
`).WithTimeout(30 * time.Second)
	defer func() { _ = ex.Close() }()

	result, err := ex.ExecuteResult(ctx)
	if err == nil {
		t.Fatal("missing entry function should fail")
	}
	if result == nil || !strings.Contains(result.Error, DefaultEntryFunctionName) {
		t.Fatalf("unexpected result: %+v, err=%v", result, err)
	}
}

func TestExecutor_ExecuteResultRejectsInvalidReturnShape(t *testing.T) {
	ctx := context.Background()
	ex := NewExecutor(entryCode(`
return ["not allowed"]
`)).WithTimeout(30 * time.Second)
	defer func() { _ = ex.Close() }()

	result, err := ex.ExecuteResult(ctx)
	if err == nil {
		t.Fatal("invalid return shape should fail")
	}
	if result == nil || !strings.Contains(result.Error, "必须返回 dict") {
		t.Fatalf("unexpected result: %+v, err=%v", result, err)
	}
}
