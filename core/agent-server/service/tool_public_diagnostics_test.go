package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
)

func TestPublicBackendErrorsDoNotLeakOrSuggestRepeatingWrites(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code string
	}{
		{errors.New("HTTP 500 SELECT password FROM accounts at /srv/namespace/system/app/code/cmd/app"), "service_error"},
		{fmt.Errorf("private transport: %w", context.DeadlineExceeded), "operation_timeout"},
		{context.Canceled, "operation_canceled"},
		{os.ErrPermission, "permission_denied"},
	} {
		got := publicToolBackendError(context.Background(), "create", tc.err)
		if !strings.Contains(got, tc.code) {
			t.Fatalf("wrong category: %s", got)
		}
		for _, bad := range []string{"SELECT", "password", "/srv", "namespace", "HTTP", "private", "重试"} {
			if strings.Contains(got, bad) {
				t.Fatalf("unexpected %s: %s", bad, got)
			}
		}
	}
}

func TestPublicLogPreservesBusinessMessageWithoutStorageOrSQL(t *testing.T) {
	got := formatPublicAppLog(&dto.ReadAppLogResp{
		ResolvedVersion: "v3", LogFile: "/srv/workplace/logs/app.log",
		Content: "订单金额必须大于零\nSELECT * FROM private_accounts\nconnect nats://private:4222\nFile \"/tmp/runtime/script.py\", line 12\nValueError: amount must be positive\nC:\\private\\runtime\\main.go:20\n/system/test11/orders.table",
	}, "/system/test11")
	for _, bad := range []string{"/srv", "private", "SELECT", "nats", "/tmp", "script.py", "main.go", "LogFile"} {
		if strings.Contains(got, bad) {
			t.Fatalf("leaked %s: %s", bad, got)
		}
	}
	for _, want := range []string{"订单金额必须大于零", "ValueError", "line 12", "/system/test11/orders.table", "v3"} {
		if !strings.Contains(got, want) {
			t.Fatalf("lost %s: %s", want, got)
		}
	}
}

func TestPythonPublicContractAndResult(t *testing.T) {
	def := (&RunPythonTool{}).Definition()
	schema := formatStructuredToolData(def.InputSchema)
	guidance := buildPythonModelGuidance(map[string]interface{}{"status": "失败", "output": "ModuleNotFoundError: No module named foo"})
	for _, bad := range []string{"Podman", "Dockerfile", "/_runtime", "基础镜像", "pythonRuntime.NewExecutor", "容器", "read_file"} {
		if strings.Contains(def.Description+schema+guidance, bad) {
			t.Fatalf("Python contract exposes %s", bad)
		}
	}
	raw := map[string]interface{}{"data": map[string]interface{}{
		"status": "失败", "output": "File \"/srv/private/worker.py\", line 7\nKeyError: 'customer'",
		"json_result": "{\"count\":3}", "output_files": "bucket/result.csv", "internal_trace": "private trace",
	}, "runtime_host": "private"}
	out := publicPythonResult(raw, "/system/test11")
	got := formatStructuredToolData(out)
	for _, bad := range []string{"runtime_host", "internal_trace", "/srv", "worker.py"} {
		if strings.Contains(got, bad) {
			t.Fatalf("result leaked %s: %s", bad, got)
		}
	}
	if out["status"] != "失败" || out["output_files"] != "bucket/result.csv" || out["json_result"] != "{\"count\":3}" || !strings.Contains(got, "KeyError") {
		t.Fatalf("public result lost: %s", got)
	}
	if !strings.Contains(raw["data"].(map[string]interface{})["output"].(string), "/srv") {
		t.Fatal("mutated original diagnostics")
	}
	if publicPythonResult(nil, "")["status"] != "失败" {
		t.Fatal("missing result must not be success")
	}
}
