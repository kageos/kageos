// create_spreadsheet.go：根据表格数据创建 Excel，路由 POST /document/excel/create_spreadsheet.form

package table

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/xuri/excelize/v2"
)

// CreateSpreadsheetReq 根据数据创建 Excel 请求
// 表格数据支持两种格式：1）单表 [[行1],[行2],...]；2）多表 {"表名1":[[...]], "表名2":[[...]]}
type CreateSpreadsheetReq struct {
	TableData string `json:"table_data" widget:"name:表格数据(JSON);type:text_area;placeholder:单表 [[\"姓名\",\"年龄\"],[\"张三\",18]] 或多表 {\"销售\":[[...]],\"库存\":[[...]]}" validate:"required"`
	SheetName string `json:"sheet_name" widget:"name:单表时的工作表名;type:input;render_default:Sheet1"`
	FileName  string `json:"file_name" widget:"name:输出文件名;type:input;render_default:表格.xlsx"`
}

// CreateSpreadsheetResp 创建 Excel 响应
type CreateSpreadsheetResp struct {
	OutputFile string `json:"output_file" widget:"name:输出文件;type:files"`
	Message    string `json:"message" widget:"name:说明;type:text_area"`
}

// CreateSpreadsheet 入口
func CreateSpreadsheet(ctx *app.Context, resp response.Response) error {
	var req CreateSpreadsheetReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoCreateSpreadsheet(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoCreateSpreadsheet 解析 JSON 表格数据，用 excelize 生成 xlsx（支持多 Sheet）
// table_data 可为：1）二维数组 [[...]] 单表；2）对象 {"表名":[[...]], ...} 多表
func DoCreateSpreadsheet(ctx *app.Context, req *CreateSpreadsheetReq) (*CreateSpreadsheetResp, error) {
	fileName := strings.TrimSpace(req.FileName)
	if fileName == "" {
		fileName = "表格.xlsx"
	}
	if !strings.HasSuffix(strings.ToLower(fileName), ".xlsx") {
		fileName = fileName + ".xlsx"
	}
	sheetName := strings.TrimSpace(req.SheetName)
	if sheetName == "" {
		sheetName = "Sheet1"
	}

	data := strings.TrimSpace(req.TableData)
	if data == "" {
		return nil, fmt.Errorf("表格数据不能为空")
	}
	raw := []byte(data)

	// 先尝试解析为多表：{"表名": [[...]], ...}
	var sheetMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &sheetMap); err == nil && len(sheetMap) > 0 {
		// 判定为对象：键为表名，值为二维数组（再解析一次）
		names := make([]string, 0, len(sheetMap))
		for k := range sheetMap {
			names = append(names, k)
		}
		sort.Strings(names)
		f := excelize.NewFile()
		defer f.Close()
		first := true
		for _, name := range names {
			var rows [][]interface{}
			if err := json.Unmarshal(sheetMap[name], &rows); err != nil {
				return nil, fmt.Errorf("工作表「%s」的数据不是合法二维数组: %w", name, err)
			}
			if first {
				if f.GetSheetName(0) != name {
					_ = f.SetSheetName(f.GetSheetName(0), name)
				}
				setSheetData(ctx, f, name, rows)
				first = false
			} else {
				_, _ = f.NewSheet(name)
				setSheetData(ctx, f, name, rows)
			}
		}
		outputDir := ctx.GetFS().GetTraceOutputDir()
		outPath := filepath.Join(outputDir, fileName)
		if err := f.SaveAs(outPath); err != nil {
			return nil, fmt.Errorf("保存 xlsx 失败: %w", err)
		}
		fs := ctx.GetFS()
		outputFiles := fs.ResponseFiles([]string{outPath})
		return &CreateSpreadsheetResp{
			OutputFile: outputFiles,
			Message:    fmt.Sprintf("已创建 %d 个工作表：%s，文件 %s", len(names), strings.Join(names, "、"), fileName),
		}, nil
	}

	// 单表：二维数组
	var rows [][]interface{}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("表格数据不是合法 JSON（应为二维数组或 {\"表名\":[[...]]}）: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("表格数据不能为空")
	}
	f := excelize.NewFile()
	defer f.Close()
	if f.GetSheetName(0) != sheetName {
		_ = f.SetSheetName(f.GetSheetName(0), sheetName)
	}
	setSheetData(ctx, f, sheetName, rows)
	outputDir := ctx.GetFS().GetTraceOutputDir()
	outPath := filepath.Join(outputDir, fileName)
	if err := f.SaveAs(outPath); err != nil {
		return nil, fmt.Errorf("保存 xlsx 失败: %w", err)
	}
	fs := ctx.GetFS()
	outputFiles := fs.ResponseFiles([]string{outPath})
	return &CreateSpreadsheetResp{
		OutputFile: outputFiles,
		Message:    fmt.Sprintf("已创建 %d 行数据，工作表「%s」，文件 %s", len(rows), sheetName, fileName),
	}, nil
}

func setSheetData(ctx *app.Context, f *excelize.File, sheetName string, rows [][]interface{}) {
	for i, row := range rows {
		for j, cell := range row {
			cellRef, err := excelize.CoordinatesToCellName(j+1, i+1)
			if err != nil {
				logger.Warnf(ctx, "[excel/CreateSpreadsheet] 单元格 %d,%d 跳过: %v", i+1, j+1, err)
				continue
			}
			var v interface{}
			switch c := cell.(type) {
			case string:
				v = c
			case float64:
				v = c
			case int:
				v = float64(c)
			case int64:
				v = float64(c)
			case bool:
				v = c
			case nil:
				v = ""
			default:
				v = fmt.Sprint(cell)
			}
			_ = f.SetCellValue(sheetName, cellRef, v)
		}
	}
}

var CreateSpreadsheetTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "根据数据创建 Excel",
		Desc:     `根据 JSON 数据生成 xlsx 文件，支持多工作表。单表：[["姓名","年龄"],["张三",18]]；多表：{"销售":[[...]], "库存":[[...]]}，键为表名。适合对话或检索结果导出为表格、报表生成。`,
		Tags:     []string{"Excel", "xlsx", "创建Excel", "JSON转Excel", "json转excel", "根据数据创建表格", "多工作表", "报表", "导出表格"},
		Request:  &CreateSpreadsheetReq{},
		Response: &CreateSpreadsheetResp{},
	},
}

func init() {
	packageContext.POST("create_spreadsheet.form", CreateSpreadsheet, CreateSpreadsheetTemplate)
}
