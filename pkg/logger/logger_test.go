package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultLogFilenameUsesTempDirDuringGoTest(t *testing.T) {
	t.Setenv("KAGEOS_LOG_FILE", "")

	got := defaultLogFilename()
	if !strings.HasPrefix(got, filepath.Join(os.TempDir(), "kageos-test-logs")+string(os.PathSeparator)) {
		t.Fatalf("defaultLogFilename() = %q, want temp test log path", got)
	}
	if strings.Contains(got, string(filepath.Separator)+"core"+string(filepath.Separator)) ||
		strings.Contains(got, string(filepath.Separator)+"pkg"+string(filepath.Separator)) {
		t.Fatalf("defaultLogFilename() = %q, should not point inside source package dirs", got)
	}
}

func TestDefaultLogFilenameHonorsExplicitEnv(t *testing.T) {
	want := filepath.Join(t.TempDir(), "explicit.log")
	t.Setenv("KAGEOS_LOG_FILE", want)

	if got := defaultLogFilename(); got != want {
		t.Fatalf("defaultLogFilename() = %q, want %q", got, want)
	}
}
