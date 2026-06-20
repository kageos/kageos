package database

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func ensureSQLite() error {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return fmt.Errorf("未找到 sqlite3，请确认运行环境已安装 sqlite3")
	}
	return nil
}

var sqliteIdentRE = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func sanitizeSQLiteIdentifier(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	name = sqliteIdentRE.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")
	if name == "" {
		name = fallback
	}
	if name[0] >= '0' && name[0] <= '9' {
		name = "_" + name
	}
	return name
}

func quoteSQLiteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func quoteSQLiteString(value string) string {
	return `'` + strings.ReplaceAll(value, `'`, `''`) + `'`
}

func sqliteSafeBase(name, fallback string) string {
	name = strings.TrimSuffix(filepath.Base(strings.TrimSpace(name)), filepath.Ext(name))
	if name == "" {
		name = fallback
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)
	if name == "" {
		return fallback
	}
	return name
}

func sqliteEnsureExt(name, ext string) string {
	base := sqliteSafeBase(name, "database")
	return base + "." + strings.TrimPrefix(ext, ".")
}

func truncateSQLiteText(text string, max int) string {
	if max <= 0 {
		max = 80000
	}
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max]) + "\n...（预览已截断，完整内容见输出文件）"
}
