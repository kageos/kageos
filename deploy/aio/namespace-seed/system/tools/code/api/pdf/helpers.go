package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func ensureCommand(commandName, packageName string) error {
	if _, err := exec.LookPath(commandName); err != nil {
		return fmt.Errorf("未找到 %s，请确认运行环境已安装 %s", commandName, packageName)
	}
	return nil
}

func outputFileName(originalName, localPath, suffix, ext string, seen map[string]int) string {
	name := strings.TrimSpace(originalName)
	if name == "" {
		name = filepath.Base(localPath)
	}
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	base = sanitizeFileName(base, "document")
	ext = strings.TrimPrefix(strings.TrimSpace(ext), ".")
	if ext == "" {
		ext = "txt"
	}
	candidate := base + suffix + "." + ext
	if seen == nil {
		return candidate
	}
	if _, ok := seen[candidate]; !ok {
		seen[candidate] = 1
		return candidate
	}
	baseWithSuffix := base + suffix
	for {
		seen[candidate]++
		next := fmt.Sprintf("%s_%d.%s", baseWithSuffix, seen[candidate], ext)
		if _, ok := seen[next]; !ok {
			seen[next] = 1
			return next
		}
	}
}

func outputStem(originalName, localPath, suffix string, seen map[string]int) string {
	name := outputFileName(originalName, localPath, suffix, "pdf", seen)
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func sanitizeFileName(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	name = replacer.Replace(name)
	name = strings.TrimSpace(name)
	if name == "" {
		return fallback
	}
	return name
}

func appendPageArgs(args []string, firstPage, lastPage int) ([]string, error) {
	if firstPage < 0 || lastPage < 0 {
		return nil, fmt.Errorf("页码不能为负数")
	}
	if firstPage > 0 {
		args = append(args, "-f", strconv.Itoa(firstPage))
	}
	if lastPage > 0 {
		if firstPage > 0 && lastPage < firstPage {
			return nil, fmt.Errorf("结束页不能小于开始页")
		}
		args = append(args, "-l", strconv.Itoa(lastPage))
	}
	return args, nil
}

func normalizeDPI(dpi int) int {
	if dpi <= 0 {
		return 150
	}
	if dpi < 36 {
		return 36
	}
	if dpi > 600 {
		return 600
	}
	return dpi
}

func normalizeImageFormat(format string) (flag string, ext string) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jpeg", "jpg":
		return "-jpeg", "jpg"
	case "tiff", "tif":
		return "-tiff", "tif"
	default:
		return "-png", "png"
	}
}

func collectOutputFiles(outputDir, prefix string, allowedExts map[string]bool) ([]string, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if allowedExts != nil {
			ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
			if !allowedExts[ext] {
				continue
			}
		}
		paths = append(paths, filepath.Join(outputDir, name))
	}
	sort.Strings(paths)
	return paths, nil
}

func readTextPreview(path string, limit int) string {
	if limit <= 0 {
		limit = 120000
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("读取文本预览失败: %v", err)
	}
	text := string(data)
	runes := []rune(text)
	if len(runes) <= limit {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(string(runes[:limit])) + "\n\n...（文本过长，已截断；完整内容请查看输出 txt 文件）"
}
