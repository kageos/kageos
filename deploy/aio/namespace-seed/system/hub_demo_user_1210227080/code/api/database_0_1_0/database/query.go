package database

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type QueryReq struct {
	InputFiles     string `json:"input_files" widget:"name:上传 SQLite 数据库;type:files;accept:.db,.sqlite,.sqlite3,application/vnd.sqlite3,*/*;max_size:1000MB;max_count:10" validate:"required"`
	SQL            string `json:"sql" widget:"name:SQL 查询;type:text_area;placeholder:例如 SELECT * FROM table_name LIMIT 20;" validate:"required"`
	OutputFormat   string `json:"output_format" widget:"name:输出格式;type:select;options:表格,CSV,JSON;options_colors:409EFF,67C23A,E6A23C;render_default:表格" validate:"required,oneof=表格 CSV JSON"`
	OutputFileName string `json:"output_file_name" widget:"name:输出文件名;type:input;placeholder:可选，仅单库时生效，例如 query.csv"`
}

type QueryResp struct {
	ResultText  string `json:"result_text" widget:"name:查询结果预览;type:text_area"`
	OutputFiles string `json:"output_files" widget:"name:查询结果文件;type:files"`
	Summary     string `json:"summary" widget:"name:处理说明;type:text_area"`
}

func Query(ctx *app.Context, resp response.Response) error {
	var req QueryReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoQuery(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoQuery(ctx *app.Context, req *QueryReq) (*QueryResp, error) {
	if err := ensureSQLite(); err != nil {
		return nil, err
	}
	sql := strings.TrimSpace(req.SQL)
	if sql == "" {
		return nil, fmt.Errorf("SQL 查询不能为空")
	}
	if strings.HasPrefix(sql, ".") {
		return nil, fmt.Errorf("只读查询不支持 sqlite3 点命令，请使用 SELECT/PRAGMA/WITH 等 SQL")
	}

	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)
	if len(inputFiles) == 0 {
		return nil, fmt.Errorf("没有找到输入数据库")
	}

	outputDir := fs.GetTraceOutputDir()
	outputExt := sqliteQueryExt(req.OutputFormat)
	var outputPaths []string
	var previews []string
	var infos []string
	for _, dbPath := range inputFiles {
		if dbPath == "" {
			continue
		}
		outputName := sqliteSafeBase(filepath.Base(dbPath), "query") + "_query." + outputExt
		if len(inputFiles) == 1 && strings.TrimSpace(req.OutputFileName) != "" {
			outputName = sqliteEnsureExt(req.OutputFileName, outputExt)
		}
		outputPath := filepath.Join(outputDir, outputName)
		out, err := runSQLiteQuery(dbPath, sql, req.OutputFormat)
		if err != nil {
			logger.Warnf(ctx, "[SQLite/Query] 查询失败 %s: %v", filepath.Base(dbPath), err)
			infos = append(infos, fmt.Sprintf("失败 %s: %v", filepath.Base(dbPath), err))
			continue
		}
		if err := os.WriteFile(outputPath, out, 0644); err != nil {
			return nil, fmt.Errorf("写入查询结果失败: %w", err)
		}
		outputPaths = append(outputPaths, outputPath)
		previews = append(previews, fmt.Sprintf("## %s\n%s", filepath.Base(dbPath), truncateSQLiteText(string(out), 80000)))
		infos = append(infos, fmt.Sprintf("成功 %s -> %s", filepath.Base(dbPath), outputName))
	}
	if len(outputPaths) == 0 {
		return nil, fmt.Errorf("没有成功执行的查询\n%s", strings.Join(infos, "\n"))
	}
	return &QueryResp{
		ResultText:  strings.Join(previews, "\n\n"),
		OutputFiles: fs.ResponseFiles(outputPaths),
		Summary:     fmt.Sprintf("SQLite 查询完成\n输出格式: %s\n输出文件数: %d\n\n详情:\n%s", req.OutputFormat, len(outputPaths), strings.Join(infos, "\n")),
	}, nil
}

func runSQLiteQuery(dbPath, sql, outputFormat string) ([]byte, error) {
	args := []string{"-readonly"}
	switch outputFormat {
	case "CSV":
		args = append(args, "-header", "-csv")
	case "JSON":
		args = append(args, "-json")
	default:
		args = append(args, "-header", "-column")
	}
	args = append(args, dbPath, sql)
	out, err := exec.Command("sqlite3", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sqlite3 查询失败: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func sqliteQueryExt(outputFormat string) string {
	switch outputFormat {
	case "CSV":
		return "csv"
	case "JSON":
		return "json"
	default:
		return "txt"
	}
}

var QueryTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "SQLite 只读查询",
		Desc:     `对上传的 SQLite 数据库执行只读 SQL 查询，支持表格、CSV、JSON 输出。使用 sqlite3 -readonly，不经过 shell，适合查看 CSV 入库后的数据或分析已有 .db 文件。`,
		Tags:     []string{"SQLite", "SQL", "查询", "CSV", "JSON", "数据库"},
		Request:  &QueryReq{},
		Response: &QueryResp{},
	},
}

func init() {
	packageContext.POST("query.form", Query, QueryTemplate)
}
