package service

import (
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
)

func TestResolveSourceFileWriteTargetRejectsTestFiles(t *testing.T) {
	t.Parallel()

	appPaths := newRuntimeAppPaths(t.TempDir(), "luobei", "demo")
	for _, fileName := range []string{"ticket_test.go", "ticket_test"} {
		t.Run(fileName, func(t *testing.T) {
			t.Parallel()

			_, _, err := resolveSourceFileWriteTarget(appPaths, &dto.SourceFileWrite{
				DirectoryPath: "tickets",
				FileName:      fileName,
				SourceCode:    "package tickets\n",
			})
			if err == nil {
				t.Fatalf("expected _test.go source file to fail")
			}
			if !strings.Contains(err.Error(), "_test.go") || !strings.Contains(err.Error(), "API 注册") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
