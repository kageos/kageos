package table

import (
	stdcsv "encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func delimiterRune(option string, sample []byte) rune {
	switch strings.TrimSpace(option) {
	case "制表符":
		return '\t'
	case "分号":
		return ';'
	case "竖线":
		return '|'
	case "逗号":
		return ','
	default:
		return detectDelimiter(sample)
	}
}

func detectDelimiter(sample []byte) rune {
	candidates := []rune{',', '\t', ';', '|'}
	best := ','
	bestScore := -1
	lines := strings.Split(string(sample), "\n")
	if len(lines) > 20 {
		lines = lines[:20]
	}
	for _, candidate := range candidates {
		score := 0
		nonZero := 0
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			count := strings.Count(line, string(candidate))
			if count > 0 {
				nonZero++
				score += count
			}
		}
		score += nonZero * 10
		if score > bestScore {
			best = candidate
			bestScore = score
		}
	}
	return best
}

func newCSVReader(path string, delimiter rune) (*stdcsv.Reader, *os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	reader := stdcsv.NewReader(file)
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	return reader, file, nil
}

func readSample(path string, max int) []byte {
	if max <= 0 {
		max = 65536
	}
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	buf := make([]byte, max)
	n, _ := file.Read(buf)
	return buf[:n]
}

func normalizeRecord(record []string) []string {
	out := make([]string, len(record))
	for i, cell := range record {
		if i == 0 {
			cell = strings.TrimPrefix(cell, "\ufeff")
		}
		out[i] = strings.TrimSpace(cell)
	}
	return out
}

func readCSVRows(path string, delimiter rune, maxRows int) ([][]string, bool, error) {
	reader, file, err := newCSVReader(path, delimiter)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	var rows [][]string
	truncated := false
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return rows, truncated, err
		}
		if maxRows > 0 && len(rows) >= maxRows {
			truncated = true
			break
		}
		rows = append(rows, normalizeRecord(record))
	}
	return rows, truncated, nil
}

func csvOutputName(inputName, customName, suffix, ext string, single bool) string {
	if single && strings.TrimSpace(customName) != "" {
		return csvEnsureExt(customName, ext)
	}
	base := csvSafeBase(inputName, "data")
	return base + suffix + "." + strings.TrimPrefix(ext, ".")
}

func csvSafeBase(name, fallback string) string {
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

func csvEnsureExt(name, ext string) string {
	base := csvSafeBase(name, "data")
	return base + "." + strings.TrimPrefix(ext, ".")
}

func delimiterLabel(delimiter rune) string {
	switch delimiter {
	case '\t':
		return "制表符"
	case ';':
		return "分号"
	case '|':
		return "竖线"
	default:
		return "逗号"
	}
}

func markdownTable(headers []string, rows [][]string, maxRows int) string {
	if len(headers) == 0 {
		return ""
	}
	if maxRows <= 0 || maxRows > len(rows) {
		maxRows = len(rows)
	}
	var b strings.Builder
	b.WriteString("| ")
	for _, header := range headers {
		b.WriteString(escapeMarkdownCell(header))
		b.WriteString(" | ")
	}
	b.WriteString("\n| ")
	for range headers {
		b.WriteString("--- | ")
	}
	for i := 0; i < maxRows; i++ {
		b.WriteString("\n| ")
		for col := range headers {
			cell := ""
			if col < len(rows[i]) {
				cell = rows[i][col]
			}
			b.WriteString(escapeMarkdownCell(cell))
			b.WriteString(" | ")
		}
	}
	return b.String()
}

func escapeMarkdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	if len([]rune(value)) > 80 {
		return string([]rune(value)[:80]) + "..."
	}
	return value
}

func inferCellType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "empty"
	}
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return "integer"
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return "float"
	}
	lower := strings.ToLower(value)
	if lower == "true" || lower == "false" || lower == "yes" || lower == "no" {
		return "boolean"
	}
	for _, layout := range []string{"2006-01-02", "2006-01-02 15:04:05", time.RFC3339, "2006/01/02", "01/02/2006"} {
		if _, err := time.Parse(layout, value); err == nil {
			return "date"
		}
	}
	return "text"
}

func mergeCellType(current, next string) string {
	if current == "" || current == "empty" {
		return next
	}
	if next == "" || next == "empty" || current == next {
		return current
	}
	if (current == "integer" && next == "float") || (current == "float" && next == "integer") {
		return "float"
	}
	return "text"
}

func generatedHeaders(width int) []string {
	headers := make([]string, width)
	for i := range headers {
		headers[i] = fmt.Sprintf("column_%d", i+1)
	}
	return headers
}

func normalizeHeaders(row []string) []string {
	seen := map[string]int{}
	headers := make([]string, len(row))
	for i, header := range row {
		header = strings.TrimSpace(header)
		if header == "" {
			header = fmt.Sprintf("column_%d", i+1)
		}
		if seen[header] == 0 {
			seen[header] = 1
			headers[i] = header
			continue
		}
		seen[header]++
		headers[i] = fmt.Sprintf("%s_%d", header, seen[header])
	}
	return headers
}
