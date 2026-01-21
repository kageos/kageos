package python

import (
	"context"
	"testing"
	"time"
)

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
			code: `
result = a + b
print(f"结果: {result}")
`,
			args: map[string]interface{}{
				"a": 10,
				"b": 20,
			},
			wantErr: false,
		},
		{
			name: "JSON 输出",
			code: `
import json
result = {"sum": a + b, "product": a * b}
print(json.dumps(result))
`,
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
			code: `
import json
result = {"sum": a + b, "product": a * b}
print(json.dumps(result))
`,
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
	code := `
import json
import pandas as pd

df = pd.DataFrame([{"a": 1, "b": 2}])
result = {"rows": len(df), "columns": df.columns.tolist()}
print(json.dumps(result))
`

	executor := NewExecutor(code).
		WithPackages("pandas").
		WithTimeout(2 * time.Minute)

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
	code := `
import json
result = {"message": f"Hello, {name}!", "count": count}
print(json.dumps(result))
`

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

	err := executor.ExecuteJSON(ctx, &result)
	if err != nil {
		t.Fatalf("ExecuteJSON() error = %v", err)
	}

	if result.Message != "Hello, World!" || result.Count != 42 {
		t.Errorf("结果不正确: message=%s, count=%d", result.Message, result.Count)
	}

	t.Logf("结果: %+v", result)
}
