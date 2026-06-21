package database

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type InspectReq struct {
	InputFiles    string `json:"input_files" widget:"name:上传 SQLite 数据库;type:files;accept:.db,.sqlite,.sqlite3,application/vnd.sqlite3,*/*;max_size:1000MB;max_count:10" validate:"required"`
	IncludeCounts bool   `json:"include_counts" widget:"name:统计每张表行数;type:switch;render_default:true"`
	OutputReport  bool   `json:"output_report" widget:"name:输出结构报告文件;type:switch;render_default:true"`
}

type InspectResp struct {
	SchemaText string `json:"schema_text" widget:"name:数据库结构;type:text_area"`
	OutputFile string `json:"output_file" widget:"name:结构报告;type:files"`
	Summary    string `json:"summary" widget:"name:处理说明;type:text_area"`
}

func Inspect(ctx *app.Context, resp response.Response) error {
	var req InspectReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoInspect(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoInspect(ctx *app.Context, req *InspectReq) (*InspectResp, error) {
	if err := ensureSQLite(); err != nil {
		return nil, err
	}
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入数据库")
	}

	var blocks []string
	var summaries []string
	for _, dbPath := range inputFiles {
		if dbPath == "" {
			continue
		}
		block, summary, err := inspectSQLiteDB(dbPath, req.IncludeCounts)
		if err != nil {
			summaries = append(summaries, fmt.Sprintf("失败 %s: %v", filepath.Base(dbPath), err))
			continue
		}
		blocks = append(blocks, block)
		summaries = append(summaries, summary)
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("没有成功读取的 SQLite 数据库\n%s", strings.Join(summaries, "\n"))
	}
	text := strings.TrimSpace(strings.Join(blocks, "\n\n"))
	var outputFile string
	if req.OutputReport {
		outputPath := filepath.Join(fs.GetTraceOutputDir(), "sqlite_schema_report.txt")
		if err := os.WriteFile(outputPath, []byte(text), 0644); err != nil {
			return nil, fmt.Errorf("写入结构报告失败: %w", err)
		}
		outputFile = fs.ResponseFiles([]string{outputPath})
	}
	return &InspectResp{
		SchemaText: text,
		OutputFile: outputFile,
		Summary:    "SQLite 结构读取完成\n" + strings.Join(summaries, "\n"),
	}, nil
}

func inspectSQLiteDB(dbPath string, includeCounts bool) (string, string, error) {
	schemaSQL := `SELECT type, name, tbl_name, sql FROM sqlite_master WHERE type IN ('table','view','index','trigger') AND name NOT LIKE 'sqlite_%' ORDER BY type, name;`
	out, err := exec.Command("sqlite3", "-readonly", "-header", "-column", dbPath, schemaSQL).CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("读取结构失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	var b strings.Builder
	b.WriteString("## ")
	b.WriteString(filepath.Base(dbPath))
	b.WriteString("\n\n### 对象结构\n")
	b.WriteString(strings.TrimSpace(string(out)))
	if includeCounts {
		counts, err := sqliteTableCounts(dbPath)
		if err != nil {
			b.WriteString("\n\n### 表行数\n统计失败: ")
			b.WriteString(err.Error())
		} else if strings.TrimSpace(counts) != "" {
			b.WriteString("\n\n### 表行数\n")
			b.WriteString(counts)
		}
	}
	return b.String(), fmt.Sprintf("成功 %s", filepath.Base(dbPath)), nil
}

func sqliteTableCounts(dbPath string) (string, error) {
	out, err := exec.Command("sqlite3", "-readonly", "-noheader", "-list", dbPath, `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name;`).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("读取表列表失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	var lines []string
	for _, table := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		table = strings.TrimSpace(table)
		if table == "" {
			continue
		}
		countOut, err := exec.Command("sqlite3", "-readonly", "-noheader", dbPath, "SELECT COUNT(*) FROM "+quoteSQLiteIdent(table)+";").CombinedOutput()
		if err != nil {
			lines = append(lines, fmt.Sprintf("%s: 统计失败: %v", table, err))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", table, strings.TrimSpace(string(countOut))))
	}
	return strings.Join(lines, "\n"), nil
}

var InspectTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "查看 SQLite 结构",
		Desc:     `读取 SQLite 数据库中的表、视图、索引、触发器定义，并可统计每张表行数。适合查询前快速了解上传 .db 文件结构。`,
		Tags:     []string{"SQLite", "Schema", "数据库结构", "表", "行数"},
		Request:  &InspectReq{},
		Response: &InspectResp{},
	},
}

func init() {
	packageContext.POST("inspect.form", Inspect, InspectTemplate)
}
