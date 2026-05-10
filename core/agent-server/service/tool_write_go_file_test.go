package service

import (
	"context"
	"strings"
	"testing"
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
