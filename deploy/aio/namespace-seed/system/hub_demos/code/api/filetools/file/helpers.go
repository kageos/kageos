package file

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func sanitizeOutputBaseName(name, fallback string) string {
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
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if name == "" {
		return fallback
	}
	return name
}

func sanitizeFileName(name, fallback string) string {
	base := sanitizeOutputBaseName(name, fallback)
	ext := strings.TrimSpace(filepath.Ext(name))
	return base + ext
}

func uniqueName(name string, seen map[string]int) string {
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
		if _, exists := seen[candidate]; !exists {
			seen[candidate] = 1
			return candidate
		}
	}
}

func flattenZipEntryName(zipBase, entryName string, seen map[string]int) string {
	entryName = strings.ReplaceAll(entryName, "\\", "/")
	entryName = strings.TrimPrefix(entryName, "/")
	entryName = strings.TrimSpace(entryName)
	if entryName == "" {
		entryName = "file"
	}
	parts := strings.Split(entryName, "/")
	// 工作台文件展示和上传都更适合稳定的平铺文件名，这里把 ZIP 内层级压平成单层名称。
	for i, part := range parts {
		parts[i] = sanitizeOutputBaseName(part, "file")
	}
	flattened := strings.Join(parts, "__")
	if flattened == "" {
		flattened = "file"
	}
	if zipBase != "" {
		flattened = zipBase + "__" + flattened
	}
	return uniqueName(flattened, seen)
}

func copyFile(src, dst string) error {
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
