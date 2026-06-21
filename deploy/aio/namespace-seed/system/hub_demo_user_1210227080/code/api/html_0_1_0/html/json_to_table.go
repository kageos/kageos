package html

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type JSONToTableReq struct {
	InputJSON string `json:"input_json" widget:"name:JSON 数据;type:text_area;placeholder:输入 JSON 数组，例如 [{\"name\":\"张三\",\"age\":20}]" validate:"required"`
	Title     string `json:"title" widget:"name:表格标题;type:input;placeholder:可选"`
	FileName  string `json:"file_name" widget:"name:文件名;type:input;placeholder:输入文件名（无需后缀），例如 sales-report" validate:"required"`
	Theme     string `json:"theme" widget:"name:主题风格;type:select;options:简洁白,深色,蓝色商务;render_default:简洁白" validate:"required"`
}

type JSONToTableResp struct {
	OutputFile string `json:"output_file" widget:"name:HTML 文件;type:files"`
	Info       string `json:"info" widget:"name:信息;type:text"`
}

func JSONToTable(ctx *app.Context, resp response.Response) error {
	var req JSONToTableReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	var rawData interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(req.InputJSON)), &rawData); err != nil {
		return fmt.Errorf("JSON 解析失败: %v", err)
	}

	var rows []map[string]interface{}
	switch v := rawData.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				rows = append(rows, m)
			}
		}
	case map[string]interface{}:
		rows = append(rows, v)
	default:
		return fmt.Errorf("JSON 必须是数组或对象，当前类型不支持")
	}

	if len(rows) == 0 {
		return fmt.Errorf("没有可展示的数据行")
	}

	colSet := make(map[string]bool)
	for _, row := range rows {
		for k := range row {
			colSet[k] = true
		}
	}
	var columns []string
	for k := range colSet {
		columns = append(columns, k)
	}
	sort.Strings(columns)

	title := req.Title
	if title == "" {
		title = "数据表格"
	}

	baseName := sanitizeFileName(req.FileName)

	themeCSS := getTableThemeCSS(req.Theme)
	htmlContent := buildTableHTML(title, columns, rows, themeCSS)

	fs := ctx.GetFS()
	outputDir := fs.GetTraceOutputDir()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	outputPath := filepath.Join(outputDir, baseName+".html")
	if err := os.WriteFile(outputPath, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}

	outputFiles := fs.ResponseFiles([]string{outputPath})

	return resp.Form(&JSONToTableResp{
		OutputFile: outputFiles,
		Info:       fmt.Sprintf("共 %d 行 × %d 列", len(rows), len(columns)),
	}).Build()
}

func cellToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return template.HTMLEscapeString(val)
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		b, _ := json.Marshal(val)
		return template.HTMLEscapeString(string(b))
	}
}

func buildTableHTML(title string, columns []string, rows []map[string]interface{}, themeCSS string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
%s
</style>
</head>
<body>
<div class="container">
<h1 class="title">%s</h1>
<div class="toolbar">
<input type="text" id="searchInput" placeholder="搜索..." class="search-input">
<span class="row-count" id="rowCount">共 %d 条</span>
</div>
<div class="table-wrap">
<table id="dataTable">
<thead><tr>`, template.HTMLEscapeString(title), themeCSS, template.HTMLEscapeString(title), len(rows)))

	for _, col := range columns {
		sb.WriteString(fmt.Sprintf(`<th onclick="sortTable(this)" data-col="%s">%s <span class="sort-icon">⇅</span></th>`, template.HTMLEscapeString(col), template.HTMLEscapeString(col)))
	}
	sb.WriteString(`</tr></thead><tbody>`)

	for _, row := range rows {
		sb.WriteString("<tr>")
		for _, col := range columns {
			sb.WriteString(fmt.Sprintf("<td>%s</td>", cellToString(row[col])))
		}
		sb.WriteString("</tr>")
	}

	sb.WriteString(`</tbody></table></div></div>`)
	sb.WriteString(tableScript)
	sb.WriteString(`</body></html>`)

	return sb.String()
}

const tableScript = `
<script>
const table = document.getElementById('dataTable');
const searchInput = document.getElementById('searchInput');
const rowCount = document.getElementById('rowCount');
const tbody = table.querySelector('tbody');
let sortDir = {};

searchInput.addEventListener('input', function() {
  const q = this.value.toLowerCase();
  let visible = 0;
  for (const row of tbody.rows) {
    const match = row.textContent.toLowerCase().includes(q);
    row.style.display = match ? '' : 'none';
    if (match) visible++;
  }
  rowCount.textContent = q ? visible + ' / ' + tbody.rows.length + ' 条' : '共 ' + tbody.rows.length + ' 条';
});

function sortTable(th) {
  const col = th.cellIndex;
  const dir = sortDir[col] === 'asc' ? 'desc' : 'asc';
  sortDir[col] = dir;
  const rows = Array.from(tbody.rows);
  rows.sort((a, b) => {
    let va = a.cells[col].textContent.trim();
    let vb = b.cells[col].textContent.trim();
    const na = parseFloat(va), nb = parseFloat(vb);
    if (!isNaN(na) && !isNaN(nb)) return dir === 'asc' ? na - nb : nb - na;
    return dir === 'asc' ? va.localeCompare(vb, 'zh') : vb.localeCompare(va, 'zh');
  });
  rows.forEach(r => tbody.appendChild(r));
  for (const s of table.querySelectorAll('.sort-icon')) s.textContent = '⇅';
  th.querySelector('.sort-icon').textContent = dir === 'asc' ? '↑' : '↓';
}
</script>`

func getTableThemeCSS(theme string) string {
	switch theme {
	case "深色":
		return `* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans SC", sans-serif; background: #1a1a2e; color: #eee; padding: 2rem; }
.container { max-width: 1200px; margin: 0 auto; }
.title { font-size: 1.5rem; margin-bottom: 1rem; color: #e0e0e0; }
.toolbar { display: flex; align-items: center; gap: 1rem; margin-bottom: 1rem; }
.search-input { padding: 8px 14px; border: 1px solid #444; border-radius: 6px; font-size: 14px; width: 260px; background: #16213e; color: #eee; outline: none; }
.search-input:focus { border-color: #0f3460; }
.row-count { color: #888; font-size: 13px; }
.table-wrap { overflow-x: auto; border-radius: 8px; box-shadow: 0 2px 12px rgba(0,0,0,.4); }
table { width: 100%; border-collapse: collapse; font-size: 14px; }
th { background: #16213e; color: #ccc; padding: 10px 14px; text-align: left; cursor: pointer; user-select: none; white-space: nowrap; border-bottom: 2px solid #0f3460; }
th:hover { background: #0f3460; }
td { padding: 9px 14px; border-bottom: 1px solid #2a2a4a; }
tr:hover td { background: #16213e; }
.sort-icon { font-size: 12px; color: #666; }`
	case "蓝色商务":
		return `* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans SC", sans-serif; background: #f0f4f8; color: #333; padding: 2rem; }
.container { max-width: 1200px; margin: 0 auto; }
.title { font-size: 1.5rem; margin-bottom: 1rem; color: #1a365d; }
.toolbar { display: flex; align-items: center; gap: 1rem; margin-bottom: 1rem; }
.search-input { padding: 8px 14px; border: 1px solid #cbd5e0; border-radius: 6px; font-size: 14px; width: 260px; outline: none; }
.search-input:focus { border-color: #3182ce; box-shadow: 0 0 0 2px rgba(49,130,206,.2); }
.row-count { color: #718096; font-size: 13px; }
.table-wrap { overflow-x: auto; border-radius: 8px; box-shadow: 0 1px 8px rgba(0,0,0,.1); }
table { width: 100%; border-collapse: collapse; font-size: 14px; background: #fff; }
th { background: #2b6cb0; color: #fff; padding: 10px 14px; text-align: left; cursor: pointer; user-select: none; white-space: nowrap; }
th:hover { background: #2c5282; }
td { padding: 9px 14px; border-bottom: 1px solid #e2e8f0; }
tr:nth-child(even) td { background: #f7fafc; }
tr:hover td { background: #ebf8ff; }
.sort-icon { font-size: 12px; color: rgba(255,255,255,.6); }`
	default:
		return `* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans SC", sans-serif; background: #fafafa; color: #333; padding: 2rem; }
.container { max-width: 1200px; margin: 0 auto; }
.title { font-size: 1.5rem; margin-bottom: 1rem; color: #222; }
.toolbar { display: flex; align-items: center; gap: 1rem; margin-bottom: 1rem; }
.search-input { padding: 8px 14px; border: 1px solid #ddd; border-radius: 6px; font-size: 14px; width: 260px; outline: none; }
.search-input:focus { border-color: #999; }
.row-count { color: #999; font-size: 13px; }
.table-wrap { overflow-x: auto; border-radius: 8px; box-shadow: 0 1px 6px rgba(0,0,0,.08); }
table { width: 100%; border-collapse: collapse; font-size: 14px; background: #fff; }
th { background: #f5f5f5; color: #555; padding: 10px 14px; text-align: left; cursor: pointer; user-select: none; white-space: nowrap; border-bottom: 2px solid #e0e0e0; }
th:hover { background: #eee; }
td { padding: 9px 14px; border-bottom: 1px solid #f0f0f0; }
tr:hover td { background: #f9f9f9; }
.sort-icon { font-size: 12px; color: #ccc; }`
	}
}

var JSONToTableTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "JSON 转交互式表格",
		Desc:     `将 JSON 数组数据转换为可直接访问的交互式 HTML 表格网页。支持列排序、实时搜索过滤、多种主题风格。无需任何外部依赖，生成的页面可直接在浏览器中打开。常用于数据展示、报表分享、数据审查等场景。`,
		Tags:     []string{"JSON", "表格", "HTML表格", "数据展示", "数据可视化", "报表", "网页"},
		Request:  &JSONToTableReq{},
		Response: &JSONToTableResp{},
	},
}

func init() {
	packageContext.POST("json_to_table.form", JSONToTable, JSONToTableTemplate)
}
