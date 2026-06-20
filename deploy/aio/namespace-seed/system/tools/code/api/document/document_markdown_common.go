package document

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/xuri/excelize/v2"
)

type markdownExtractResult struct {
	Markdown string
	Summary  string
}

func extractLocalFileToMarkdown(ctx *app.Context, localPath string, displayName string, sheetName string, ocrLanguage string) (*markdownExtractResult, error) {
	ext := strings.ToLower(filepath.Ext(displayName))
	switch ext {
	case ".md", ".markdown":
		text, err := readUTF8LikeTextFile(localPath)
		if err != nil {
			return nil, err
		}
		return &markdownExtractResult{
			Markdown: normalizeMarkdownText(text),
			Summary:  "直接读取 Markdown 文本",
		}, nil
	case ".txt":
		text, err := readUTF8LikeTextFile(localPath)
		if err != nil {
			return nil, err
		}
		return &markdownExtractResult{
			Markdown: normalizeMarkdownText(text),
			Summary:  "直接读取纯文本",
		}, nil
	case ".csv":
		md, err := csvFileToMarkdown(localPath)
		if err != nil {
			return nil, err
		}
		return &markdownExtractResult{
			Markdown: md,
			Summary:  "CSV 转 Markdown 表格",
		}, nil
	case ".json":
		md, summary, err := jsonFileToMarkdown(localPath)
		if err != nil {
			return nil, err
		}
		return &markdownExtractResult{
			Markdown: md,
			Summary:  summary,
		}, nil
	case ".html", ".htm":
		md, err := convertFileToMarkdownWithPandoc(ctx, localPath, "html")
		if err != nil {
			return nil, err
		}
		return &markdownExtractResult{
			Markdown: normalizeMarkdownText(md),
			Summary:  "HTML 转 Markdown",
		}, nil
	case ".docx", ".odt", ".rtf":
		md, err := convertFileToMarkdownWithPandoc(ctx, localPath, "")
		if err != nil {
			return nil, err
		}
		return &markdownExtractResult{
			Markdown: normalizeMarkdownText(md),
			Summary:  "Pandoc 转 Markdown",
		}, nil
	case ".doc", ".ppt", ".pptx", ".pptm", ".odp":
		md, err := convertOfficeFileViaLibreOfficePDF(ctx, localPath, displayName, ocrLanguage)
		if err != nil {
			return nil, err
		}
		return &markdownExtractResult{
			Markdown: normalizeMarkdownText(md),
			Summary:  "LibreOffice 转 PDF 后提取文本",
		}, nil
	case ".xlsx", ".xls":
		md, err := excelFileToMarkdown(localPath, sheetName)
		if err != nil {
			return nil, err
		}
		return &markdownExtractResult{
			Markdown: md,
			Summary:  "Excel 转 Markdown 表格",
		}, nil
	case ".pdf":
		md, summary, err := pdfFileToMarkdown(ctx, localPath, displayName, ocrLanguage)
		if err != nil {
			return nil, err
		}
		return &markdownExtractResult{
			Markdown: normalizeMarkdownText(md),
			Summary:  summary,
		}, nil
	case ".png", ".jpg", ".jpeg", ".bmp", ".gif", ".tif", ".tiff", ".webp":
		md, err := imageFileToMarkdown(ctx, localPath, ocrLanguage)
		if err != nil {
			return nil, err
		}
		return &markdownExtractResult{
			Markdown: normalizeMarkdownText(md),
			Summary:  "图片 OCR 识别",
		}, nil
	default:
		text, err := readUTF8LikeTextFile(localPath)
		if err != nil {
			return nil, fmt.Errorf("暂不支持的文件类型 %s", ext)
		}
		return &markdownExtractResult{
			Markdown: normalizeMarkdownText(text),
			Summary:  "按纯文本读取",
		}, nil
	}
}

func readUTF8LikeTextFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("文件不是 UTF-8 文本")
	}
	return string(data), nil
}

func csvFileToMarkdown(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开 CSV 失败: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("解析 CSV 失败: %w", err)
	}
	return rowsToMarkdownTable(rows), nil
}

func jsonFileToMarkdown(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("读取 JSON 失败: %w", err)
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err == nil && len(rows) > 0 {
		return recordsToMarkdownTable(rows), "JSON 数组转 Markdown 表格", nil
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil && len(obj) > 0 {
		pretty, _ := json.MarshalIndent(obj, "", "  ")
		return "```json\n" + string(pretty) + "\n```", "JSON 对象按代码块输出", nil
	}

	text := strings.TrimSpace(string(data))
	if text == "" {
		text = "{}"
	}
	return "```json\n" + text + "\n```", "原样输出 JSON 文本", nil
}

func convertFileToMarkdownWithPandoc(ctx *app.Context, inputPath string, inputFormat string) (string, error) {
	if _, err := exec.LookPath("pandoc"); err != nil {
		return "", fmt.Errorf("未找到 pandoc，请确保运行环境已安装 Pandoc")
	}
	args := []string{inputPath}
	if inputFormat != "" {
		args = append(args, "-f", inputFormat)
	}
	args = append(args, "-t", "gfm")
	cmd := exec.Command("pandoc", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[document/convertFileToMarkdownWithPandoc] 转换失败: %v, output: %s", err, string(out))
		return "", fmt.Errorf("Pandoc 转 Markdown 失败: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func convertOfficeFileViaLibreOfficePDF(ctx *app.Context, inputPath string, displayName string, ocrLanguage string) (string, error) {
	bin := "libreoffice"
	if _, err := exec.LookPath(bin); err != nil {
		bin = "soffice"
		if _, err2 := exec.LookPath(bin); err2 != nil {
			return "", fmt.Errorf("未找到 libreoffice 或 soffice，请确保运行环境已安装 LibreOffice")
		}
	}

	outputDir := ctx.GetFS().GetTraceOutputDir()
	cmd := exec.Command(bin, "--headless", "--convert-to", "pdf", "--outdir", outputDir, inputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[document/convertOfficeFileViaLibreOfficePDF] 转换失败: %v, output: %s", err, string(out))
		return "", fmt.Errorf("LibreOffice 转 PDF 失败: %v", err)
	}

	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	pdfPath := filepath.Join(outputDir, baseName+".pdf")
	defer os.Remove(pdfPath)

	md, _, err := pdfFileToMarkdown(ctx, pdfPath, displayName+".pdf", ocrLanguage)
	if err != nil {
		return "", err
	}
	return md, nil
}

func excelFileToMarkdown(path string, sheetName string) (string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", fmt.Errorf("打开 Excel 失败: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return "", fmt.Errorf("Excel 没有工作表")
	}
	targetSheets := sheets
	if strings.TrimSpace(sheetName) != "" {
		targetSheets = nil
		for _, name := range sheets {
			if name == sheetName {
				targetSheets = []string{name}
				break
			}
		}
		if len(targetSheets) == 0 {
			return "", fmt.Errorf("未找到指定工作表 %s", sheetName)
		}
	}

	var parts []string
	for _, name := range targetSheets {
		rows, err := f.GetRows(name)
		if err != nil {
			return "", fmt.Errorf("读取工作表 %s 失败: %w", name, err)
		}
		table := rowsToMarkdownTable(rows)
		if len(targetSheets) > 1 {
			parts = append(parts, "### "+name+"\n\n"+table)
		} else {
			parts = append(parts, table)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n")), nil
}

func pdfFileToMarkdown(ctx *app.Context, pdfPath string, displayName string, ocrLanguage string) (string, string, error) {
	directText, err := extractPDFTextWithPoppler(ctx, pdfPath)
	if err == nil && len(strings.TrimSpace(directText)) >= 50 {
		return directText, "PDF 直接提取文本", nil
	}

	outputDir := ctx.GetFS().GetTraceOutputDir()
	prefix := sanitizeMarkdownFileName(strings.TrimSuffix(filepath.Base(displayName), filepath.Ext(displayName))) + "_ocr"
	imagePaths, err := renderPDFToPNG(ctx, pdfPath, outputDir, prefix)
	if err != nil {
		return "", "", err
	}
	defer func() {
		for _, path := range imagePaths {
			_ = os.Remove(path)
		}
	}()

	var pageTexts []string
	for idx, imagePath := range imagePaths {
		text, err := runTesseractOCR(ctx, imagePath, ocrLanguage)
		if err != nil {
			logger.Warnf(ctx, "[document/pdfFileToMarkdown] OCR 失败 %s 第%d页: %v", displayName, idx+1, err)
			continue
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		pageTexts = append(pageTexts, fmt.Sprintf("### 第 %d 页\n\n%s", idx+1, strings.TrimSpace(text)))
	}
	if len(pageTexts) == 0 {
		return "", "", fmt.Errorf("PDF 未提取到可用文本")
	}
	return strings.Join(pageTexts, "\n\n"), "PDF 转图片后 OCR 识别", nil
}

func imageFileToMarkdown(ctx *app.Context, imagePath string, ocrLanguage string) (string, error) {
	text, err := runTesseractOCR(ctx, imagePath, ocrLanguage)
	if err != nil {
		return "", err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("图片未识别到可用文本")
	}
	return text, nil
}

func extractPDFTextWithPoppler(ctx *app.Context, pdfPath string) (string, error) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return "", fmt.Errorf("未找到 pdftotext，请确保运行环境已安装 poppler-utils")
	}
	cmd := exec.Command("pdftotext", pdfPath, "-")
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warnf(ctx, "[document/extractPDFTextWithPoppler] 提取失败: %v", err)
		return "", fmt.Errorf("pdftotext 执行失败: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func renderPDFToPNG(ctx *app.Context, pdfPath string, outputDir string, prefix string) ([]string, error) {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return nil, fmt.Errorf("未找到 pdftoppm，请确保运行环境已安装 poppler-utils")
	}
	cmd := exec.Command("pdftoppm", "-png", "-r", "200", pdfPath, filepath.Join(outputDir, prefix))
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorf(ctx, "[document/renderPDFToPNG] 转图片失败: %v, output: %s", err, string(out))
		return nil, fmt.Errorf("pdftoppm 执行失败: %w", err)
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}
	var paths []string
	prefixWithDash := prefix + "-"
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, prefixWithDash) && strings.HasSuffix(strings.ToLower(name), ".png") {
			paths = append(paths, filepath.Join(outputDir, name))
		}
	}
	sort.Slice(paths, func(i, j int) bool {
		leftName := strings.TrimSuffix(filepath.Base(paths[i]), ".png")
		rightName := strings.TrimSuffix(filepath.Base(paths[j]), ".png")
		leftPage, leftErr := strconv.Atoi(strings.TrimPrefix(leftName, prefixWithDash))
		rightPage, rightErr := strconv.Atoi(strings.TrimPrefix(rightName, prefixWithDash))
		if leftErr == nil && rightErr == nil {
			return leftPage < rightPage
		}
		return paths[i] < paths[j]
	})
	if len(paths) == 0 {
		return nil, fmt.Errorf("PDF 转图片后未生成输出文件")
	}
	return paths, nil
}

func runTesseractOCR(ctx *app.Context, inputPath string, ocrLanguage string) (string, error) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		return "", fmt.Errorf("未找到 tesseract，请确保运行环境已安装 Tesseract")
	}
	lang := strings.TrimSpace(ocrLanguage)
	if lang == "" {
		lang = "chi_sim+eng"
	}
	cmd := exec.Command("tesseract", inputPath, "stdout", "-l", lang)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warnf(ctx, "[document/runTesseractOCR] OCR 失败: %v, output: %s", err, string(out))
		return "", fmt.Errorf("tesseract 执行失败: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func rowsToMarkdownTable(rows [][]string) string {
	if len(rows) == 0 {
		return "(空表)"
	}
	header := append([]string(nil), rows[0]...)
	if len(header) == 0 {
		header = []string{"列1"}
	}
	var builder strings.Builder
	builder.WriteString("| ")
	builder.WriteString(strings.Join(escapeMarkdownCells(header), " | "))
	builder.WriteString(" |\n")
	builder.WriteString("| ")
	builder.WriteString(strings.Join(repeatStrings("---", len(header)), " | "))
	builder.WriteString(" |\n")
	for _, row := range rows[1:] {
		padded := append([]string(nil), row...)
		for len(padded) < len(header) {
			padded = append(padded, "")
		}
		if len(padded) > len(header) {
			padded = padded[:len(header)]
		}
		builder.WriteString("| ")
		builder.WriteString(strings.Join(escapeMarkdownCells(padded), " | "))
		builder.WriteString(" |\n")
	}
	return strings.TrimRight(builder.String(), "\n")
}

func recordsToMarkdownTable(rows []map[string]interface{}) string {
	columnSet := make(map[string]struct{})
	for _, row := range rows {
		for key := range row {
			columnSet[key] = struct{}{}
		}
	}
	columns := make([]string, 0, len(columnSet))
	for key := range columnSet {
		columns = append(columns, key)
	}
	sort.Strings(columns)
	if len(columns) == 0 {
		return "(空数据)"
	}
	tableRows := make([][]string, 0, len(rows)+1)
	tableRows = append(tableRows, columns)
	for _, row := range rows {
		line := make([]string, 0, len(columns))
		for _, col := range columns {
			line = append(line, fmt.Sprint(row[col]))
		}
		tableRows = append(tableRows, line)
	}
	return rowsToMarkdownTable(tableRows)
}

func escapeMarkdownCells(cells []string) []string {
	out := make([]string, len(cells))
	for i, cell := range cells {
		cell = strings.ReplaceAll(cell, "|", "\\|")
		cell = strings.ReplaceAll(cell, "\r\n", "\n")
		cell = strings.ReplaceAll(cell, "\n", "<br>")
		out[i] = strings.TrimSpace(cell)
	}
	return out
}

func repeatStrings(value string, count int) []string {
	out := make([]string, count)
	for i := 0; i < count; i++ {
		out[i] = value
	}
	return out
}

func normalizeMarkdownText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	var normalized []string
	blankCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount > 2 {
				continue
			}
		} else {
			blankCount = 0
		}
		normalized = append(normalized, line)
	}
	return strings.TrimSpace(strings.Join(normalized, "\n"))
}

func sanitizeMarkdownFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "document"
	}
	if ext := filepath.Ext(name); ext != "" {
		name = strings.TrimSuffix(name, ext)
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
		" ", "_",
	)
	return replacer.Replace(name)
}
