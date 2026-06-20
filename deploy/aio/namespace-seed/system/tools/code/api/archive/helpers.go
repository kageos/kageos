package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func archiveKind(path string) string {
	name := strings.ToLower(filepath.Base(path))
	switch {
	case strings.HasSuffix(name, ".zip"):
		return "zip"
	case strings.HasSuffix(name, ".tar"), strings.HasSuffix(name, ".tar.gz"), strings.HasSuffix(name, ".tgz"),
		strings.HasSuffix(name, ".tar.xz"), strings.HasSuffix(name, ".txz"), strings.HasSuffix(name, ".tar.bz2"), strings.HasSuffix(name, ".tbz2"):
		return "tar"
	default:
		return ""
	}
}

func ensureTar() error {
	if _, err := exec.LookPath("tar"); err != nil {
		return fmt.Errorf("未找到 tar，请确认运行环境已安装 tar")
	}
	return nil
}

func archiveBaseName(path string) string {
	name := filepath.Base(path)
	lower := strings.ToLower(name)
	for _, suffix := range []string{".tar.gz", ".tar.xz", ".tar.bz2", ".tgz", ".txz", ".tbz2", ".zip", ".tar"} {
		if strings.HasSuffix(lower, suffix) {
			return sanitizeArchiveName(name[:len(name)-len(suffix)], "archive")
		}
	}
	return sanitizeArchiveName(strings.TrimSuffix(name, filepath.Ext(name)), "archive")
}

func sanitizeArchiveName(name, fallback string) string {
	name = strings.TrimSpace(name)
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

func flattenEntryName(prefix, entryName string, seen map[string]int) string {
	entryName = strings.ReplaceAll(entryName, "\\", "/")
	entryName = strings.TrimPrefix(entryName, "/")
	entryName = strings.TrimSpace(entryName)
	if entryName == "" {
		entryName = "file"
	}
	parts := strings.Split(entryName, "/")
	for i, part := range parts {
		parts[i] = sanitizeArchiveName(part, "file")
	}
	name := strings.Join(parts, "__")
	if prefix != "" {
		name = prefix + "__" + name
	}
	return uniqueArchiveName(name, seen)
}

func uniqueArchiveName(name string, seen map[string]int) string {
	if seen == nil {
		return name
	}
	if _, ok := seen[name]; !ok {
		seen[name] = 1
		return name
	}
	base := strings.TrimSuffix(name, filepath.Ext(name))
	ext := filepath.Ext(name)
	for {
		seen[name]++
		candidate := fmt.Sprintf("%s_%d%s", base, seen[name], ext)
		if _, ok := seen[candidate]; !ok {
			seen[candidate] = 1
			return candidate
		}
	}
}

func listZipEntries(path string, maxEntries int) ([]string, int, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, 0, err
	}
	defer reader.Close()
	total := 0
	var lines []string
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		total++
		if maxEntries <= 0 || len(lines) < maxEntries {
			lines = append(lines, fmt.Sprintf("%10d  %s", entry.UncompressedSize64, entry.Name))
		}
	}
	return lines, total, nil
}

func listTarEntries(path string, maxEntries int) ([]string, int, error) {
	if err := ensureTar(); err != nil {
		return nil, 0, err
	}
	out, err := exec.Command("tar", "-tf", path).CombinedOutput()
	if err != nil {
		return nil, 0, fmt.Errorf("tar 查看失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	rawLines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var lines []string
	total := 0
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasSuffix(line, "/") {
			continue
		}
		total++
		if maxEntries <= 0 || len(lines) < maxEntries {
			lines = append(lines, line)
		}
	}
	return lines, total, nil
}

func copyRegularFile(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	output, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer output.Close()
	_, err = io.Copy(output, input)
	return err
}
