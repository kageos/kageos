package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/pkg/sourcepolicy"
)

func TestWriteGoFileMissingRequiredArgsMakesNoPersistenceExplicit(t *testing.T) {
	msg, isErr := runWriteGoFileTool(context.Background(), writeGoFileArgs{Content: "package demo"}, "/user/app/demo")
	if !isErr {
		t.Fatalf("expected missing file_name to fail")
	}
	for _, want := range []string{"缺少参数 file_name", "本次未落盘", "不要继续假设该文件已创建"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected %q in %q", want, msg)
		}
	}

	msg, isErr = runWriteGoFileTool(context.Background(), writeGoFileArgs{FileName: "demo.go"}, "/user/app/demo")
	if !isErr {
		t.Fatalf("expected missing content to fail")
	}
	for _, want := range []string{"缺少参数 content", "本次未落盘", "不要继续假设该文件已创建"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected %q in %q", want, msg)
		}
	}
}

func TestWriteGoFileRejectsTestFileName(t *testing.T) {
	for _, fileName := range []string{"demo_test.go", "demo_test", "demo_test.go.go"} {
		t.Run(fileName, func(t *testing.T) {
			msg, isErr := runWriteGoFileTool(context.Background(), writeGoFileArgs{
				FileName: fileName,
				Content:  "package demo",
			}, "/user/app/demo")
			if !isErr {
				t.Fatalf("expected _test.go file_name to fail")
			}
			for _, want := range []string{"不允许写入 _test.go", "API 注册", "本次未落盘"} {
				if !strings.Contains(msg, want) {
					t.Fatalf("expected %q in %q", want, msg)
				}
			}
		})
	}
}

func TestWriteGoFileRejectsSQLiteDriverImport(t *testing.T) {
	msg, isErr := runWriteGoFileTool(context.Background(), writeGoFileArgs{
		FileName: "importer.go",
		Content: `package demo

import "gorm.io/driver/sqlite"

var _ = sqlite.Open
`,
	}, "/user/app/demo")
	if !isErr {
		t.Fatal("expected sqlite driver import to fail")
	}
	for _, want := range []string{"源码规范校验失败", "本次未落盘", "KageOS SDK 已全局注册", `sql.Open("sqlite3", path)`} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected %q in %q", want, msg)
		}
	}
}

func TestWriteGoFileSourcePolicyAllowsAppDBPassedToExternalPackage(t *testing.T) {
	source := `package demo

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
	third "github.com/acme/blackbox"
)

func Handle(ctx *app.Context) error {
	db := ctx.GetGormDB()
	return third.Use(db)
}
`
	if err := sourcepolicy.ValidateAppGoSource("db_leak.go", source); err != nil {
		t.Fatalf("ValidateAppGoSource() error = %v", err)
	}
}
