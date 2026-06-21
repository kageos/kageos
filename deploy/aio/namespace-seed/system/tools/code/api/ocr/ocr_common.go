// ocr_common.go：tesseract OCR 公共逻辑，供 ImageToText、PDFToText 等接口复用

package ocr

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
)

// RunTesseractOnFile 对单个文件执行 tesseract 识别，输出到 stdout，返回识别文本
// imagePath 为本地路径，lang 为 tesseract -l 参数（如 chi_sim、eng、chi_sim+eng）
func RunTesseractOnFile(ctx *app.Context, imagePath string, lang string) (string, error) {
	if lang == "" {
		lang = "chi_sim+eng"
	}
	args := []string{imagePath, "stdout", "-l", lang}
	cmd := exec.Command("tesseract", args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		logger.Warnf(ctx, "[tesseract] 识别失败 %s: %v, output: %s", imagePath, err, text)
		return "", fmt.Errorf("tesseract 执行失败: %w", err)
	}
	return text, nil
}

// pdfExtractText 用 pdftotext 直接提取 PDF 内嵌文本（有文本层的 PDF）。无文本层或扫描版会返回很少或空。
// 依赖 poppler-utils（pdftotext）。
func pdfExtractText(ctx *app.Context, pdfPath string) (string, error) {
	cmd := exec.Command("pdftotext", pdfPath, "-")
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		logger.Warnf(ctx, "[tesseract] pdftotext 失败 %s: %v", pdfPath, err)
		return "", fmt.Errorf("pdftotext 执行失败: %w", err)
	}
	return text, nil
}

// minExtractedTextLen 直接提取的文本若超过该长度则认为「有文本」，不再走 OCR
const minExtractedTextLen = 50

// pdfToImages 使用 pdftoppm 将 PDF 转为 PNG 图片，返回按页码排序的图片路径列表
// 依赖系统安装 poppler-utils（pdftoppm）。outputDir 为输出目录，prefix 为文件名前缀，生成 prefix-1.png, prefix-2.png ...
func pdfToImages(ctx *app.Context, pdfPath string, outputDir string, prefix string) ([]string, error) {
	cmd := exec.Command("pdftoppm", "-png", "-r", "200", pdfPath, filepath.Join(outputDir, prefix))
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warnf(ctx, "[tesseract] pdftoppm 失败 %s: %v, output: %s", pdfPath, err, string(out))
		return nil, fmt.Errorf("pdftoppm 执行失败（请安装 poppler-utils）: %w", err)
	}
	// 列出 prefix-*.png 并按页码排序
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}
	var paths []string
	pfx := prefix + "-"
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, pfx) || !strings.HasSuffix(name, ".png") {
			continue
		}
		paths = append(paths, filepath.Join(outputDir, name))
	}
	sort.Slice(paths, func(i, j int) bool {
		ni := strings.TrimSuffix(filepath.Base(paths[i]), ".png")
		nj := strings.TrimSuffix(filepath.Base(paths[j]), ".png")
		pi, _ := strconv.Atoi(strings.TrimPrefix(ni, pfx))
		pj, _ := strconv.Atoi(strings.TrimPrefix(nj, pfx))
		return pi < pj
	})
	return paths, nil
}
