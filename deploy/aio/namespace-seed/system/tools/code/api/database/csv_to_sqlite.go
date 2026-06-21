package database

import (
	stdcsv "encoding/csv"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type CSVToSQLiteReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传 CSV/TSV 文件;type:files;accept:.csv,.tsv,text/csv,text/tab-separated-values,text/*,*/*;max_size:500MB;max_count:20" validate:"required"`
	Delimiter      string `json:"delimiter" widget:"name:分隔符;type:select;options:自动,逗号,制表符,分号,竖线;options_colors:409EFF,909399,67C23A,E6A23C,909399;render_default:自动" validate:"required,oneof=自动 逗号 制表符 分号 竖线"`
	HasHeader      bool   `json:"has_header" widget:"name:首行是表头;type:switch;render_default:true"`
	TableName      string `json:"table_name" widget:"name:表名;type:input;placeholder:可选，默认使用文件名"`
	MaxRows        int    `json:"max_rows" widget:"name:最多导入数据行;type:integer;min:0;max:100000;render_default:20000;placeholder:0 表示使用默认上限" validate:"min=0,max=100000"`
	OutputFileName string `json:"output_file_name" widget:"name:输出数据库文件名;type:input;placeholder:可选，仅单文件时生效，例如 data.db"`
}

type CSVToSQLiteResp struct {
	OutputFiles string `json:"output_files" widget:"name:SQLite 数据库;type:files"`
	ImportInfo  string `json:"import_info" widget:"name:导入信息;type:text_area"`
}

func CSVToSQLite(ctx *app.Context, resp response.Response) error {
	var req CSVToSQLiteReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoCSVToSQLite(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoCSVToSQLite(ctx *app.Context, req *CSVToSQLiteReq) (*CSVToSQLiteResp, error) {
	if err := ensureSQLite(); err != nil {
		return nil, err
	}
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	maxRows := req.MaxRows
	if maxRows <= 0 {
		maxRows = 20000
	}
	outputDir := fs.GetTraceOutputDir()
	var outputPaths []string
	var infos []string
	for _, path := range inputFiles {
		if path == "" {
			continue
		}
		outputName := sqliteEnsureExt(sqliteSafeBase(filepath.Base(path), "data"), "db")
		if len(inputFiles) == 1 && strings.TrimSpace(req.OutputFileName) != "" {
			outputName = sqliteEnsureExt(req.OutputFileName, "db")
		}
		outputPath := filepath.Join(outputDir, outputName)
		tableName := strings.TrimSpace(req.TableName)
		if tableName == "" || len(inputFiles) > 1 {
			tableName = sqliteSafeBase(filepath.Base(path), "data")
		}
		count, truncated, err := importCSVToSQLite(path, outputPath, req.Delimiter, req.HasHeader, tableName, maxRows)
		if err != nil {
			infos = append(infos, fmt.Sprintf("失败 %s: %v", filepath.Base(path), err))
			continue
		}
		outputPaths = append(outputPaths, outputPath)
		info := fmt.Sprintf("成功 %s -> %s，表 %s，导入 %d 行", filepath.Base(path), outputName, sanitizeSQLiteIdentifier(tableName, "data"), count)
		if truncated {
			info += "（达到上限，已截断）"
		}
		infos = append(infos, info)
	}
	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功生成的 SQLite 数据库\n%s", strings.Join(infos, "\n"))
	}
	return &CSVToSQLiteResp{
		OutputFiles: fs.ResponseFiles(outputPaths),
		ImportInfo:  "CSV 入库完成\n" + strings.Join(infos, "\n"),
	}, nil
}

func importCSVToSQLite(inputPath, outputPath, delimiterOption string, hasHeader bool, tableName string, maxRows int) (int, bool, error) {
	delimiter := sqliteDelimiterRune(delimiterOption, sqliteReadSample(inputPath, 65536))
	rows, truncated, err := sqliteReadCSVRows(inputPath, delimiter, maxRows+1)
	if err != nil {
		return 0, false, err
	}
	if len(rows) == 0 {
		return 0, false, fmt.Errorf("文件为空或没有可读行")
	}
	if truncated && len(rows) > maxRows {
		rows = rows[:maxRows]
	}

	width := maxSQLiteRecordWidth(rows)
	headers := sqliteGeneratedHeaders(width)
	dataRows := rows
	if hasHeader {
		headers = normalizeSQLiteHeaders(rows[0], width)
		dataRows = rows[1:]
	}

	table := sanitizeSQLiteIdentifier(tableName, "data")
	var script strings.Builder
	script.WriteString("PRAGMA journal_mode=OFF;\n")
	script.WriteString("PRAGMA synchronous=OFF;\n")
	script.WriteString("BEGIN;\n")
	script.WriteString("DROP TABLE IF EXISTS ")
	script.WriteString(quoteSQLiteIdent(table))
	script.WriteString(";\nCREATE TABLE ")
	script.WriteString(quoteSQLiteIdent(table))
	script.WriteString(" (")
	for i, header := range headers {
		if i > 0 {
			script.WriteString(", ")
		}
		script.WriteString(quoteSQLiteIdent(header))
		script.WriteString(" TEXT")
	}
	script.WriteString(");\n")
	for _, row := range dataRows {
		script.WriteString("INSERT INTO ")
		script.WriteString(quoteSQLiteIdent(table))
		script.WriteString(" VALUES (")
		for i := range headers {
			if i > 0 {
				script.WriteString(", ")
			}
			value := ""
			if i < len(row) {
				value = row[i]
			}
			script.WriteString(quoteSQLiteString(value))
		}
		script.WriteString(");\n")
	}
	script.WriteString("COMMIT;\n")

	_ = os.Remove(outputPath)
	cmd := exec.Command("sqlite3", outputPath)
	cmd.Stdin = strings.NewReader(script.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, false, fmt.Errorf("sqlite3 导入失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	return len(dataRows), truncated, nil
}

func sqliteDelimiterRune(option string, sample []byte) rune {
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
		best := ','
		bestScore := -1
		for _, candidate := range []rune{',', '\t', ';', '|'} {
			score := strings.Count(string(sample), string(candidate))
			if score > bestScore {
				best = candidate
				bestScore = score
			}
		}
		return best
	}
}

func sqliteReadSample(path string, max int) []byte {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	buf := make([]byte, max)
	n, _ := file.Read(buf)
	return buf[:n]
}

func sqliteReadCSVRows(path string, delimiter rune, maxRows int) ([][]string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	reader := stdcsv.NewReader(file)
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
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
		if len(record) > 0 {
			record[0] = strings.TrimPrefix(record[0], "\ufeff")
		}
		rows = append(rows, record)
	}
	return rows, truncated, nil
}

func maxSQLiteRecordWidth(rows [][]string) int {
	width := 0
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	return width
}

func sqliteGeneratedHeaders(width int) []string {
	headers := make([]string, width)
	for i := range headers {
		headers[i] = fmt.Sprintf("column_%d", i+1)
	}
	return headers
}

func normalizeSQLiteHeaders(row []string, width int) []string {
	headers := make([]string, width)
	seen := map[string]int{}
	for i := range headers {
		header := ""
		if i < len(row) {
			header = row[i]
		}
		header = sanitizeSQLiteIdentifier(header, fmt.Sprintf("column_%d", i+1))
		seen[header]++
		if seen[header] > 1 {
			header = fmt.Sprintf("%s_%d", header, seen[header])
		}
		headers[i] = header
	}
	return headers
}

var CSVToSQLiteTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "CSV 转 SQLite",
		Desc:     `把上传的 CSV/TSV 转成 SQLite 数据库，所有字段按 TEXT 入库，自动生成安全表名和列名。适合把临时表格数据变成可 SQL 查询的数据文件。`,
		Tags:     []string{"SQLite", "CSV", "TSV", "数据库", "入库", "SQL"},
		Request:  &CSVToSQLiteReq{},
		Response: &CSVToSQLiteResp{},
	},
}

func init() {
	packageContext.POST("csv_to_sqlite.form", CSVToSQLite, CSVToSQLiteTemplate)
}
