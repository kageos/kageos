package document

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
)

func findLibreOfficeBin() (string, error) {
	for _, bin := range []string{"libreoffice", "soffice"} {
		if _, err := exec.LookPath(bin); err == nil {
			return bin, nil
		}
	}
	return "", fmt.Errorf("未找到 libreoffice 或 soffice，请确保运行环境已安装 LibreOffice")
}

func convertOfficeToPDF(ctx *app.Context, inputPath, originalName, outputDir, suffix string, seen map[string]int) (string, string, error) {
	bin, err := findLibreOfficeBin()
	if err != nil {
		return "", "", err
	}
	tempDir, err := os.MkdirTemp(outputDir, "libreoffice_pdf_*")
	if err != nil {
		return "", "", fmt.Errorf("创建 LibreOffice 临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	cmd := exec.Command(bin, "--headless", "--convert-to", "pdf", "--outdir", tempDir, inputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[document/convertOfficeToPDF] 转换失败 %s: %v, output: %s", filepath.Base(inputPath), err, string(out))
		return "", "", fmt.Errorf("LibreOffice 转 PDF 失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	generatedPDF, err := firstPDFInDir(tempDir)
	if err != nil {
		return "", "", fmt.Errorf("LibreOffice 执行完成但未找到 PDF 输出: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	outputName := documentOutputFileName(originalName, suffix, "pdf", seen)
	outputPath := filepath.Join(outputDir, outputName)
	if err := copyLocalFile(generatedPDF, outputPath); err != nil {
		return "", "", fmt.Errorf("复制 PDF 输出失败: %w", err)
	}
	return outputPath, outputName, nil
}

func firstPDFInDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
			return filepath.Join(dir, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("未找到 PDF 文件")
}

func documentOutputFileName(originalName, suffix, ext string, seen map[string]int) string {
	name := strings.TrimSpace(originalName)
	if name == "" {
		name = "document"
	}
	base := sanitizeMarkdownFileName(strings.TrimSuffix(filepath.Base(name), filepath.Ext(name)))
	if base == "" {
		base = "document"
	}
	ext = strings.TrimPrefix(strings.TrimSpace(ext), ".")
	if ext == "" {
		ext = "pdf"
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

func copyLocalFile(src, dst string) error {
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
