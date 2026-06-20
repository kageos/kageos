package image

import (
	"fmt"
	"path/filepath"
	"strings"
)

func outputFileName(originalName, localPath, suffix, ext string, seen map[string]int) string {
	name := strings.TrimSpace(originalName)
	if name == "" {
		name = filepath.Base(localPath)
	}
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	base = sanitizeFileName(base, "image")
	ext = strings.TrimPrefix(strings.TrimSpace(ext), ".")
	if ext == "" {
		ext = strings.TrimPrefix(filepath.Ext(localPath), ".")
	}
	if ext == "" {
		ext = "png"
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

func clampQuality(q int, fallback int) int {
	if q <= 0 {
		q = fallback
	}
	if q < 1 {
		return 1
	}
	if q > 100 {
		return 100
	}
	return q
}

func normalizeOutputFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jpg":
		return "jpeg"
	case "jpeg", "png", "webp":
		return strings.ToLower(strings.TrimSpace(format))
	default:
		return "jpeg"
	}
}
