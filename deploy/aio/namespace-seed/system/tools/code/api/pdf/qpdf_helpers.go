package pdf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func ensureQPDF() error {
	if _, err := exec.LookPath("qpdf"); err != nil {
		return fmt.Errorf("未找到 qpdf，请确认运行环境已安装 qpdf")
	}
	return nil
}

func outputPDFName(originalName, localPath, suffix string, seen map[string]int) string {
	name := strings.TrimSpace(originalName)
	if name == "" {
		name = filepath.Base(localPath)
	}
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	base = sanitizeFileName(base, "document")
	candidate := base + suffix + ".pdf"
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
		next := fmt.Sprintf("%s_%d.pdf", baseWithSuffix, seen[candidate])
		if _, ok := seen[next]; !ok {
			seen[next] = 1
			return next
		}
	}
}

func explicitPDFName(name, fallback string) string {
	name = sanitizeFileName(strings.TrimSuffix(strings.TrimSpace(name), filepath.Ext(name)), fallback)
	if name == "" {
		name = fallback
	}
	return name + ".pdf"
}

func outputPDFStem(originalName, localPath, suffix string, seen map[string]int) string {
	return strings.TrimSuffix(outputPDFName(originalName, localPath, suffix, seen), ".pdf")
}

func normalizePageRange(pageRange string) (string, error) {
	pageRange = strings.TrimSpace(pageRange)
	if pageRange == "" {
		return "", fmt.Errorf("页码范围不能为空")
	}
	parts := strings.Split(pageRange, ",")
	for _, part := range parts {
		if err := validatePageRangePart(strings.TrimSpace(part)); err != nil {
			return "", err
		}
	}
	return strings.Join(parts, ","), nil
}

func validatePageRangePart(part string) error {
	if part == "" {
		return fmt.Errorf("页码范围包含空片段")
	}
	if strings.Count(part, "-") == 0 {
		return validatePageEndpoint(part)
	}
	if strings.Count(part, "-") != 1 {
		return fmt.Errorf("页码范围格式错误: %s", part)
	}
	pair := strings.Split(part, "-")
	if pair[0] != "" {
		if err := validatePageEndpoint(pair[0]); err != nil {
			return err
		}
	}
	if pair[1] != "" {
		if err := validatePageEndpoint(pair[1]); err != nil {
			return err
		}
	}
	if pair[0] == "" && pair[1] == "" {
		return fmt.Errorf("页码范围格式错误: %s", part)
	}
	return nil
}

func validatePageEndpoint(endpoint string) error {
	if endpoint == "z" || endpoint == "Z" {
		return nil
	}
	if !regexp.MustCompile(`^\d+$`).MatchString(endpoint) {
		return fmt.Errorf("页码必须是正整数或 z: %s", endpoint)
	}
	return validatePositiveNumber(endpoint, "页码")
}

func validatePositiveNumber(value, label string) error {
	if strings.TrimLeft(value, "0") == "" {
		return fmt.Errorf("%s必须大于 0: %s", label, value)
	}
	return nil
}

func splitPDFOutputPaths(pattern string) ([]string, error) {
	dir := filepath.Dir(pattern)
	base := filepath.Base(pattern)
	prefix := strings.Split(base, "%")[0]
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.EqualFold(filepath.Ext(name), ".pdf") {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func languageCode(option string) string {
	switch strings.TrimSpace(option) {
	case "中文":
		return "chi_sim"
	case "英文":
		return "eng"
	default:
		return "chi_sim+eng"
	}
}
