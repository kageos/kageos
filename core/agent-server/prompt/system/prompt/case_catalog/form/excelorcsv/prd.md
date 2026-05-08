# 案例：Excel/CSV 工具（单 Form）

## 一、项目概要

- **类型**：单 Form，多个 POST，无 Table。
- **路由**：office_excel_to_csv.form、office_excel_to_json.form、office_excel_fill_column.form、office_csv_to_excel.form 等；路由组 `/form/excelorcsv`。
- **适合参考**：files 上传、多 POST 同目录、excelize、GetFS、响应 text_area 或 table。

---

## 二、旧版 PRD 要点（仅实现参考，不作为 app.plan 输出格式）

以 **Excel 转 JSON**（office_excel_to_json.form）为例，其余 POST 类似：请求为 files + 可选参数，响应为 text_area 或 table。

### Excel 转 JSON（office_excel_to_json.form，POST）

**请求**（表单字段五列：字段 | 类型 | 必填 | 默认值 | 说明）

| 字段           | 类型     | 必填 | 默认值 | 说明 |
|----------------|----------|------|--------|------|
| 上传 Excel 文件 | 文件上传 | ✓   | —      | .xlsx/.xls，最大 50MB，1 个 |
| 工作表名称     | 文本输入 | ✗   | —      | 留空则第一个工作表 |
| 使用第一行作为键名 | 开关   | ✗   | true   | — |
| 跳过空行       | 开关     | ✗   | true   | — |

**响应**

| 字段           | 类型     | 说明 |
|----------------|----------|------|
| JSON 文本内容  | 多行文本 | 转换后的 JSON |
| 转换统计       | 多行文本 | 行数、列数等统计 |

**说明**：office_csv_to_excel、office_excel_to_csv、office_excel_fill_column 等均为「请求 files + 可选参数 → 响应 text_area 或 files」，结构类似。

---

## 三、文件与路由

| 文件                     | 说明           | 注册路由                    |
|--------------------------|----------------|-----------------------------|
| office_excel_to_csv.go   | Excel 转 CSV   | POST office_excel_to_csv.form 等 |
| office_excel_to_json.go  | Excel 转 JSON  | POST office_excel_to_json.form、office_excel_extract_column.form |
| office_excel_fill_column.go | 填列        | POST office_excel_fill_column.form |
| office_csv_to_excel.go   | CSV 转 Excel   | POST office_csv_to_excel.form、office_csv_text_to_excel.form |

---

## 四、说明

代码随本案例一起提供；read_doc 本案例路径（如 `/system/prompt/case_catalog/form/excelorcsv`）即获得 PRD 与代码，无需再调用 read_go_file。


---

## 代码实现

以下为本案目录下 Go 源码，供 read_doc 时一并参考。

### office_csv_to_excel.go

```go
//<文件名>office_csv_to_excel.go</文件名>

package excelorcsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
	"github.com/xuri/excelize/v2"
)

// CsvToExcelReq CSV转Excel请求结构体
type CsvToExcelReq struct {
	// 框架标签：widget:"type:files;accept:.csv;max_size:50MB;max_count:10" - 文件上传组件，支持多文件上传
	InputFiles string `json:"input_files" widget:"name:上传CSV文件;type:files;accept:.csv;max_size:50MB;max_count:10" validate:"required"`
}

// CsvToExcelResp CSV转Excel响应结构体
type CsvToExcelResp struct {
	// 转换后的文件列表
	OutputFiles string `json:"output_files" widget:"name:转换后的Excel文件;type:files"`

	// 转换统计信息
	ConvertStats string `json:"convert_stats" widget:"name:转换统计;type:text_area"`
}

// CsvToExcel CSV转Excel入口（SDK 注册用）：解析请求 → 调 DoCsvToExcel → 写响应
func CsvToExcel(ctx *app.Context, resp response.Response) error {
	var req CsvToExcelReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoCsvToExcel(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoCsvToExcel CSV转Excel业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoCsvToExcel(ctx *app.Context, req *CsvToExcelReq) (*CsvToExcelResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	outputFilePaths := make([]string, 0)
	successCount := 0
	failCount := 0
	var errors []string

	for _, file := range inputFiles {
		if file == "" {
			logger.Warnf(ctx, "[CsvToExcel] 文件 %s 没有本地路径，跳过", filepath.Base(file))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", filepath.Base(file)))
			continue
		}

		outputPath, err := csvToExcel(ctx, file)
		if err != nil {
			logger.Errorf(ctx, "[CsvToExcel] 转换CSV失败 %s: %v", filepath.Base(file), err)
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: %v", filepath.Base(file), err))
			continue
		}

		outputFilePaths = append(outputFilePaths, outputPath)
		successCount++
	}

	var outputFiles string
	if len(outputFilePaths) > 0 {
		outputFiles = fs.ResponseFiles(outputFilePaths)
	}

	stats := fmt.Sprintf("转换完成！\n成功: %d 个\n失败: %d 个", successCount, failCount)
	if len(errors) > 0 {
		stats += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	return &CsvToExcelResp{
		OutputFiles:  outputFiles,
		ConvertStats: stats,
	}, nil
}

// csvToExcel 转换CSV为Excel
func csvToExcel(ctx *app.Context, inputPath string) (string, error) {
	// 打开CSV文件
	csvFile, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("打开CSV文件失败: %v", err)
	}
	defer csvFile.Close()

	// 检测并处理UTF-8 BOM
	// 读取前3个字节检查是否是BOM
	bom := make([]byte, 3)
	n, err := csvFile.Read(bom)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("读取CSV文件失败: %v", err)
	}

	// 如果是BOM，文件指针已经在第3个字节之后；否则重置文件指针到开头
	if n == 3 && bom[0] == 0xEF && bom[1] == 0xBB && bom[2] == 0xBF {
		// 是UTF-8 BOM，文件指针已经在第3个字节之后，直接创建reader
		logger.Infof(ctx, "[csvToExcel] 检测到UTF-8 BOM，已跳过")
	} else {
		// 不是BOM，重置文件指针到开头
		_, err = csvFile.Seek(0, 0)
		if err != nil {
			return "", fmt.Errorf("重置文件指针失败: %v", err)
		}
	}

	// 创建CSV读取器（在BOM检测之后创建，确保文件指针位置正确）
	reader := csv.NewReader(csvFile)

	// 生成输出文件路径
	outputPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".xlsx"

	// 创建Excel文件
	excelFile := excelize.NewFile()
	defer excelFile.Close()

	// 删除默认的Sheet1，创建新的工作表（使用文件名作为工作表名称）
	sheetName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	// Excel工作表名称限制：最大31个字符，不能包含某些特殊字符
	if len(sheetName) > 31 {
		sheetName = sheetName[:31]
	}
	// 替换不允许的字符
	sheetName = strings.ReplaceAll(sheetName, ":", "")
	sheetName = strings.ReplaceAll(sheetName, "/", "")
	sheetName = strings.ReplaceAll(sheetName, "\\", "")
	sheetName = strings.ReplaceAll(sheetName, "?", "")
	sheetName = strings.ReplaceAll(sheetName, "*", "")
	sheetName = strings.ReplaceAll(sheetName, "[", "")
	sheetName = strings.ReplaceAll(sheetName, "]", "")
	if sheetName == "" {
		sheetName = "Sheet1"
	}

	// 删除默认工作表
	excelFile.DeleteSheet("Sheet1")

	// 创建新工作表
	_, err = excelFile.NewSheet(sheetName)
	if err != nil {
		return "", fmt.Errorf("创建工作表失败: %v", err)
	}

	// 设置活动工作表
	excelFile.SetActiveSheet(0)

	// 读取CSV数据并写入Excel
	rowIndex := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("读取CSV行失败: %v", err)
		}

		// 写入Excel行
		for colIndex, cellValue := range record {
			cellName, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex)
			if err != nil {
				return "", fmt.Errorf("转换单元格坐标失败: %v", err)
			}
			err = excelFile.SetCellValue(sheetName, cellName, cellValue)
			if err != nil {
				return "", fmt.Errorf("写入单元格失败: %v", err)
			}
		}
		rowIndex++
	}

	// 保存Excel文件
	err = excelFile.SaveAs(outputPath)
	if err != nil {
		return "", fmt.Errorf("保存Excel文件失败: %v", err)
	}

	logger.Infof(ctx, "[csvToExcel] 转换成功: %s -> %s (工作表: %s, 行数: %d)", inputPath, outputPath, sheetName, rowIndex-1)
	return outputPath, nil
}

// CsvToExcelTemplate CSV转Excel配置
var CsvToExcelTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "CSV转Excel",
		Desc:     `支持将CSV文件转换为Excel格式（.xlsx）。自动检测并处理UTF-8 BOM，确保中文内容正确读取。转换后的Excel文件使用文件名作为工作表名称。应用场景：数据导入、格式转换、数据分析等。`,
		Tags:     []string{"Office工具", "格式转换", "CSV", "Excel"},
		Request:  &CsvToExcelReq{},
		Response: &CsvToExcelResp{},
	},
}

// CsvTextToExcelReq CSV文本转Excel请求结构体
type CsvTextToExcelReq struct {
	// 框架标签：widget:"type:text_area" - 多行文本区域组件
	CsvText string `json:"csv_text" widget:"name:CSV文本内容;type:text_area" validate:"required"`

	// 框架标签：widget:"type:input" - 文本输入组件
	SheetName string `json:"sheet_name" widget:"name:工作表名称;type:input;render_default:Sheet1"`
}

// CsvTextToExcelResp CSV文本转Excel响应结构体
type CsvTextToExcelResp struct {
	// 转换后的文件列表
	OutputFiles string `json:"output_files" widget:"name:转换后的Excel文件;type:files"`

	// 转换统计信息
	ConvertStats string `json:"convert_stats" widget:"name:转换统计;type:text_area"`
}

// CsvTextToExcel CSV文本转Excel入口（SDK 注册用）：解析请求 → 调 DoCsvTextToExcel → 写响应
func CsvTextToExcel(ctx *app.Context, resp response.Response) error {
	var req CsvTextToExcelReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoCsvTextToExcel(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoCsvTextToExcel CSV文本转Excel业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoCsvTextToExcel(ctx *app.Context, req *CsvTextToExcelReq) (*CsvTextToExcelResp, error) {
	fs := ctx.GetFS()

	outputPath, err := csvTextToExcel(ctx, req.CsvText, req.SheetName)
	if err != nil {
		logger.Errorf(ctx, "[CsvTextToExcel] 转换CSV文本失败: %v", err)
		return &CsvTextToExcelResp{OutputFiles: nil, ConvertStats: fmt.Sprintf("转换失败: %v", err)}, nil
	}

	outputFiles := fs.ResponseFiles([]string{outputPath})

	stats := fmt.Sprintf("转换完成！\nCSV文本已成功转换为Excel文件。")
	return &CsvTextToExcelResp{OutputFiles: outputFiles, ConvertStats: stats}, nil
}

// csvTextToExcel 转换CSV文本为Excel
func csvTextToExcel(ctx *app.Context, csvText string, sheetName string) (string, error) {
	// 处理工作表名称
	if sheetName == "" {
		sheetName = "Sheet1"
	}
	// Excel工作表名称限制：最大31个字符，不能包含某些特殊字符
	if len(sheetName) > 31 {
		sheetName = sheetName[:31]
	}
	// 替换不允许的字符
	sheetName = strings.ReplaceAll(sheetName, ":", "")
	sheetName = strings.ReplaceAll(sheetName, "/", "")
	sheetName = strings.ReplaceAll(sheetName, "\\", "")
	sheetName = strings.ReplaceAll(sheetName, "?", "")
	sheetName = strings.ReplaceAll(sheetName, "*", "")
	sheetName = strings.ReplaceAll(sheetName, "[", "")
	sheetName = strings.ReplaceAll(sheetName, "]", "")
	if sheetName == "" {
		sheetName = "Sheet1"
	}

	// 检测并处理UTF-8 BOM
	csvContent := csvText
	if len(csvText) >= 3 && csvText[0] == '\xEF' && csvText[1] == '\xBB' && csvText[2] == '\xBF' {
		// 是UTF-8 BOM，跳过前3个字符
		csvContent = csvText[3:]
		logger.Infof(ctx, "[csvTextToExcel] 检测到UTF-8 BOM，已跳过")
	}

	// 创建CSV读取器
	reader := csv.NewReader(strings.NewReader(csvContent))

	// 生成输出文件路径（GetTraceOutputDir 已经基于 TraceId 返回唯一目录，只需添加唯一文件名）
	outputPath := filepath.Join(ctx.GetFS().GetTraceOutputDir(), fmt.Sprintf("csv_text_%d.xlsx", time.Now().UnixNano()))

	// 创建输出目录
	err := os.MkdirAll(filepath.Dir(outputPath), 0755)
	if err != nil {
		return "", fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 创建Excel文件
	excelFile := excelize.NewFile()
	defer excelFile.Close()

	// 删除默认的Sheet1
	excelFile.DeleteSheet("Sheet1")

	// 创建新工作表
	_, err = excelFile.NewSheet(sheetName)
	if err != nil {
		return "", fmt.Errorf("创建工作表失败: %v", err)
	}

	// 设置活动工作表
	excelFile.SetActiveSheet(0)

	// 读取CSV数据并写入Excel
	rowIndex := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("读取CSV行失败: %v", err)
		}

		// 写入Excel行
		for colIndex, cellValue := range record {
			cellName, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex)
			if err != nil {
				return "", fmt.Errorf("转换单元格坐标失败: %v", err)
			}
			err = excelFile.SetCellValue(sheetName, cellName, cellValue)
			if err != nil {
				return "", fmt.Errorf("写入单元格失败: %v", err)
			}
		}
		rowIndex++
	}

	// 保存Excel文件
	err = excelFile.SaveAs(outputPath)
	if err != nil {
		return "", fmt.Errorf("保存Excel文件失败: %v", err)
	}

	logger.Infof(ctx, "[csvTextToExcel] 转换成功: CSV文本 -> %s (工作表: %s, 行数: %d)", outputPath, sheetName, rowIndex-1)
	return outputPath, nil
}

func init() {
	// 💡 packageContext 是在当前目录下系统自动创建的变量，直接用即可，无需定义
	// CsvToExcelRouterGroup 这个要用固定的格式，文件名称的小写开头的驼峰+RouterGroup
	// 注册Form函数 - CSV转Excel
	packageContext.POST("office_csv_to_excel.form", CsvToExcel, CsvToExcelTemplate)

	// 注册Form函数 - CSV文本转Excel
	packageContext.POST("office_csv_text_to_excel.form", CsvTextToExcel, CsvTextToExcelTemplate)
}

// CsvTextToExcelTemplate CSV文本转Excel配置
var CsvTextToExcelTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "CSV文本转Excel",
		Desc:     `支持将CSV文本内容转换为Excel格式（.xlsx）。自动检测并处理UTF-8 BOM，确保中文内容正确读取。可以自定义工作表名称。应用场景：数据导入、格式转换、数据分析等。`,
		Tags:     []string{"Office工具", "格式转换", "CSV", "Excel"},
		Request:  &CsvTextToExcelReq{},
		Response: &CsvTextToExcelResp{},
	},
}
```

### office_excel_fill_column.go

```go
//<文件名>office_excel_fill_column.go</文件名>

package excelorcsv

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
	"github.com/xuri/excelize/v2"
)

// ExcelFillColumnItem 列填充配置项
type ExcelFillColumnItem struct {
	// 框架标签：widget:"type:input" - 表头列名（使用第一行的列名，如：lottery_id、数量、状态等）
	Column string `json:"column" widget:"name:表头列名;type:input;placeholder:例如: lottery_id、数量、状态（使用第一行的列名）" validate:"required"`

	// 框架标签：widget:"type:input" - 要填充的值
	Value string `json:"value" widget:"name:填充值;type:input;placeholder:例如: 1、文本、2024-01-01" validate:"required"`

	// 框架标签：widget:"type:number" - 填充的行数
	RowCount int `json:"row_count" widget:"name:填充行数;type:number;render_default:1;placeholder:从第2行开始填充（第1行通常是表头）" validate:"required,min=1"`
}

// ExcelFillColumnReq Excel列值填充请求结构体
type ExcelFillColumnReq struct {
	// 框架标签：widget:"type:files;accept:.xlsx,.xls;max_size:50MB;max_count:1" - 文件上传组件
	InputFiles string `json:"input_files" widget:"name:上传Excel文件;type:files;accept:.xlsx,.xls;max_size:50MB;max_count:1" validate:"required"`

	// 框架标签：widget:"type:input" - 工作表名称（可选，默认第一个工作表）
	SheetName string `json:"sheet_name" widget:"name:工作表名称（可选）;type:input;placeholder:留空则使用第一个工作表"`

	// 框架标签：widget:"type:table" - 表格组件，用于配置多个列的填充规则
	FillItems []ExcelFillColumnItem `json:"fill_items" widget:"name:列填充配置;type:table" validate:"required,min=1"`

	// 框架标签：widget:"type:number" - 起始行号（默认2，即从第2行开始填充，第1行通常是表头）
	StartRow int `json:"start_row" widget:"name:起始行号;type:number;render_default:2;placeholder:默认从第2行开始填充（第1行通常是表头）"`
}

// ExcelFillColumnResp Excel列值填充响应结构体
type ExcelFillColumnResp struct {
	// 填充后的文件列表
	OutputFiles string `json:"output_files" widget:"name:填充后的Excel文件;type:files"`

	// 填充统计信息
	FillStats string `json:"fill_stats" widget:"name:填充统计;type:text_area"`
}

// ExcelFillColumn Excel列值填充入口（SDK 注册用）：解析请求 → 调 DoExcelFillColumn → 写响应
func ExcelFillColumn(ctx *app.Context, resp response.Response) error {
	var req ExcelFillColumnReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoExcelFillColumn(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoExcelFillColumn Excel列值填充业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoExcelFillColumn(ctx *app.Context, req *ExcelFillColumnReq) (*ExcelFillColumnResp, error) {
	fs := ctx.GetFS()

	startRow := req.StartRow
	if startRow <= 0 {
		startRow = 2
	}

	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	if len(inputFiles) == 0 {
		return &ExcelFillColumnResp{OutputFiles: nil, FillStats: "错误: 没有找到输入文件"}, nil
	}

	file := inputFiles[0]
	if file == "" {
		return &ExcelFillColumnResp{OutputFiles: nil, FillStats: fmt.Sprintf("错误: 文件 %s 没有本地路径", filepath.Base(file))}, nil
	}

	outputPath, stats, err := excelFillColumn(ctx, file, req.SheetName, req.FillItems, startRow)
	if err != nil {
		logger.Errorf(ctx, "[ExcelFillColumn] 填充列值失败 %s: %v", filepath.Base(file), err)
		return &ExcelFillColumnResp{OutputFiles: nil, FillStats: fmt.Sprintf("填充失败: %v", err)}, nil
	}

	outputFiles := fs.ResponseFiles([]string{outputPath})

	return &ExcelFillColumnResp{OutputFiles: outputFiles, FillStats: stats}, nil
}

// excelFillColumn 填充Excel列值
func excelFillColumn(ctx *app.Context, inputPath string, sheetName string, fillItems []ExcelFillColumnItem, startRow int) (string, string, error) {
	// 打开Excel文件
	f, err := excelize.OpenFile(inputPath)
	if err != nil {
		return "", "", fmt.Errorf("打开Excel文件失败: %v", err)
	}
	defer f.Close()

	// 获取所有工作表名称
	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return "", "", fmt.Errorf("Excel文件没有工作表")
	}

	// 确定要使用的工作表
	targetSheet := sheetName
	if targetSheet == "" {
		targetSheet = sheetList[0]
	} else {
		// 检查指定的工作表是否存在
		found := false
		for _, s := range sheetList {
			if s == targetSheet {
				found = true
				break
			}
		}
		if !found {
			return "", "", fmt.Errorf("工作表 %s 不存在，可用工作表: %v", targetSheet, sheetList)
		}
	}

	// 读取第一行作为表头
	rows, err := f.GetRows(targetSheet)
	if err != nil {
		return "", "", fmt.Errorf("读取工作表 %s 失败: %v", targetSheet, err)
	}

	if len(rows) == 0 {
		return "", "", fmt.Errorf("工作表 %s 为空，无法获取表头", targetSheet)
	}

	// 获取表头（第一行）
	headerRow := rows[0]
	if len(headerRow) == 0 {
		return "", "", fmt.Errorf("工作表 %s 的第一行为空，无法获取表头", targetSheet)
	}

	// 构建表头列名到列索引的映射（不区分大小写，去除空格）
	columnNameMap := make(map[string]int)
	for i, header := range headerRow {
		header = strings.TrimSpace(header)
		if header != "" {
			// 使用小写作为key，支持不区分大小写匹配
			columnNameMap[strings.ToLower(header)] = i
		}
	}

	// 解析并填充每个列
	var fillStats []string
	totalFilled := 0

	for _, item := range fillItems {
		// 根据表头列名查找列索引
		columnName := strings.TrimSpace(item.Column)
		if columnName == "" {
			return "", "", fmt.Errorf("列名不能为空")
		}

		colIndex, found := columnNameMap[strings.ToLower(columnName)]
		if !found {
			// 如果找不到，列出所有可用的列名
			availableColumns := make([]string, 0)
			for _, h := range headerRow {
				if strings.TrimSpace(h) != "" {
					availableColumns = append(availableColumns, strings.TrimSpace(h))
				}
			}
			return "", "", fmt.Errorf("找不到列名 '%s'，可用的列名: %v", columnName, availableColumns)
		}

		// 填充列值
		filledCount := 0
		for i := 0; i < item.RowCount; i++ {
			rowNum := startRow + i
			cellName, err := excelize.CoordinatesToCellName(colIndex+1, rowNum)
			if err != nil {
				return "", "", fmt.Errorf("转换单元格坐标失败 (列:%d, 行:%d): %v", colIndex+1, rowNum, err)
			}

			// 设置单元格值
			err = f.SetCellValue(targetSheet, cellName, item.Value)
			if err != nil {
				return "", "", fmt.Errorf("设置单元格值失败 (%s): %v", cellName, err)
			}
			filledCount++
		}

		totalFilled += filledCount
		fillStats = append(fillStats, fmt.Sprintf("列 %s: 填充 %d 行，值: %s", columnName, filledCount, item.Value))
	}

	// 生成输出文件路径
	outputPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + "_filled_" + fmt.Sprintf("%d", time.Now().UnixNano()) + ".xlsx"

	// 保存Excel文件
	err = f.SaveAs(outputPath)
	if err != nil {
		return "", "", fmt.Errorf("保存Excel文件失败: %v", err)
	}

	// 构建统计信息
	stats := fmt.Sprintf("填充完成！\n工作表: %s\n起始行: %d\n总填充单元格数: %d\n\n填充详情:\n%s",
		targetSheet, startRow, totalFilled, strings.Join(fillStats, "\n"))

	logger.Infof(ctx, "[excelFillColumn] 填充成功: %s -> %s (工作表: %s, 填充单元格数: %d)", inputPath, outputPath, targetSheet, totalFilled)
	return outputPath, stats, nil
}

// ExcelFillColumnTemplate Excel列值填充配置
var ExcelFillColumnTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Excel列值填充",
		Desc:     `支持批量填充Excel指定列的值。使用表头列名（第一行的列名，如：lottery_id、数量、状态等）来指定要填充的列，支持一次配置多个列的填充规则，自动从指定起始行开始填充。应用场景：批量数据填充、数据初始化、模板填充等。`,
		Tags:     []string{"Office工具", "Excel", "数据填充"},
		Request:  &ExcelFillColumnReq{},
		Response: &ExcelFillColumnResp{},
	},
}

func init() {
	// 💡 packageContext 是在当前目录下系统自动创建的变量，直接用即可，无需定义
	// 注册Form函数 - Excel列值填充
	packageContext.POST("office_excel_fill_column.form", ExcelFillColumn, ExcelFillColumnTemplate)
}
```

### office_excel_to_csv.go

```go
//<文件名>office_excel_to_csv.go</文件名>

package excelorcsv

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
	"github.com/xuri/excelize/v2"
)

// ExcelToCsvReq Excel转CSV请求结构体
type ExcelToCsvReq struct {
	// 框架标签：widget:"type:files;accept:.xlsx,.xls;max_size:50MB;max_count:10" - 文件上传组件，支持多文件上传
	InputFiles string `json:"input_files" widget:"name:上传Excel文件;type:files;accept:.xlsx,.xls;max_size:50MB;max_count:10" validate:"required"`
}

// ExcelToCsvResp Excel转CSV响应结构体
type ExcelToCsvResp struct {
	// 转换后的文件列表
	OutputFiles string `json:"output_files" widget:"name:转换后的CSV文件;type:files"`

	// 转换统计信息
	ConvertStats string `json:"convert_stats" widget:"name:转换统计;type:text_area"`
}

// ExcelToCsv Excel转CSV入口（SDK 注册用）：解析请求 → 调 DoExcelToCsv → 写响应
func ExcelToCsv(ctx *app.Context, resp response.Response) error {
	var req ExcelToCsvReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoExcelToCsv(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoExcelToCsv Excel转CSV业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoExcelToCsv(ctx *app.Context, req *ExcelToCsvReq) (*ExcelToCsvResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	outputFilePaths := make([]string, 0)
	successCount := 0
	failCount := 0
	var errors []string

	for _, file := range inputFiles {
		if file == "" {
			logger.Warnf(ctx, "[ExcelToCsv] 文件 %s 没有本地路径，跳过", filepath.Base(file))
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: 本地路径为空", filepath.Base(file)))
			continue
		}

		outputPath, err := excelToCsv(ctx, file)
		if err != nil {
			logger.Errorf(ctx, "[ExcelToCsv] 转换Excel失败 %s: %v", filepath.Base(file), err)
			failCount++
			errors = append(errors, fmt.Sprintf("文件 %s: %v", filepath.Base(file), err))
			continue
		}

		outputFilePaths = append(outputFilePaths, outputPath)
		successCount++
	}

	var outputFiles string
	if len(outputFilePaths) > 0 {
		outputFiles = fs.ResponseFiles(outputFilePaths)
	}

	stats := fmt.Sprintf("转换完成！\n成功: %d 个\n失败: %d 个", successCount, failCount)
	if len(errors) > 0 {
		stats += "\n\n失败详情:\n" + strings.Join(errors, "\n")
	}

	return &ExcelToCsvResp{
		OutputFiles:  outputFiles,
		ConvertStats: stats,
	}, nil
}

// excelToCsv 转换Excel为CSV
func excelToCsv(ctx *app.Context, inputPath string) (string, error) {
	// 打开Excel文件
	f, err := excelize.OpenFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("打开Excel文件失败: %v", err)
	}
	defer f.Close()

	// 获取所有工作表名称
	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return "", fmt.Errorf("Excel文件没有工作表")
	}

	// 生成输出文件路径（使用第一个工作表名称，如果没有则使用文件名）
	outputPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".csv"

	// 创建CSV文件（带UTF-8 BOM）
	csvFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("创建CSV文件失败: %v", err)
	}
	defer csvFile.Close()

	// 写入UTF-8 BOM（\xEF\xBB\xBF）
	_, err = csvFile.WriteString("\xEF\xBB\xBF")
	if err != nil {
		return "", fmt.Errorf("写入BOM失败: %v", err)
	}

	// 创建CSV写入器
	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// 处理第一个工作表（如果有多个工作表，只处理第一个）
	sheetName := sheetList[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return "", fmt.Errorf("读取工作表 %s 失败: %v", sheetName, err)
	}

	// 写入所有行
	for _, row := range rows {
		// 处理空行：如果行中所有单元格都为空，则跳过
		isEmpty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			continue
		}

		// 写入CSV行
		err = writer.Write(row)
		if err != nil {
			return "", fmt.Errorf("写入CSV行失败: %v", err)
		}
	}

	logger.Infof(ctx, "[excelToCsv] 转换成功: %s -> %s (工作表: %s, 行数: %d)", inputPath, outputPath, sheetName, len(rows))
	return outputPath, nil
}

// ExcelToCsvTemplate Excel转CSV配置
var ExcelToCsvTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Excel转CSV",
		Desc:     `支持将Excel文件（.xlsx、.xls）转换为CSV格式。转换后的CSV文件使用UTF-8编码并包含BOM，确保在Excel中打开时中文显示正常。如果Excel文件包含多个工作表，只转换第一个工作表。应用场景：数据导出、格式转换、数据迁移等。`,
		Tags:     []string{"Office工具", "格式转换", "Excel", "CSV"},
		Request:  &ExcelToCsvReq{},
		Response: &ExcelToCsvResp{},
	},
}

// ExcelToCsvTextReq Excel转CSV文本请求结构体
type ExcelToCsvTextReq struct {
	// 框架标签：widget:"type:files;accept:.xlsx,.xls;max_size:50MB;max_count:1" - 文件上传组件
	InputFiles string `json:"input_files" widget:"name:上传Excel文件;type:files;accept:.xlsx,.xls;max_size:50MB;max_count:1" validate:"required"`
}

// ExcelToCsvTextResp Excel转CSV文本响应结构体
type ExcelToCsvTextResp struct {
	// CSV文本内容
	CsvText string `json:"csv_text" widget:"name:CSV文本内容;type:text_area"`

	// 转换统计信息
	ConvertStats string `json:"convert_stats" widget:"name:转换统计;type:text_area"`
}

// ExcelToCsvText Excel转CSV文本入口（SDK 注册用）：解析请求 → 调 DoExcelToCsvText → 写响应
func ExcelToCsvText(ctx *app.Context, resp response.Response) error {
	var req ExcelToCsvTextReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoExcelToCsvText(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoExcelToCsvText Excel转CSV文本业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoExcelToCsvText(ctx *app.Context, req *ExcelToCsvTextReq) (*ExcelToCsvTextResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	if len(inputFiles) == 0 {
		return &ExcelToCsvTextResp{CsvText: "", ConvertStats: "错误: 没有找到输入文件"}, nil
	}

	file := inputFiles[0]
	if file == "" {
		return &ExcelToCsvTextResp{CsvText: "", ConvertStats: fmt.Sprintf("错误: 文件 %s 没有本地路径", filepath.Base(file))}, nil
	}

	csvText, err := excelToCsvText(ctx, file)
	if err != nil {
		logger.Errorf(ctx, "[ExcelToCsvText] 转换Excel失败 %s: %v", filepath.Base(file), err)
		return &ExcelToCsvTextResp{CsvText: "", ConvertStats: fmt.Sprintf("转换失败: %v", err)}, nil
	}

	stats := fmt.Sprintf("转换完成！\n文件: %s\n行数: %d", filepath.Base(file), strings.Count(csvText, "\n")+1)
	return &ExcelToCsvTextResp{CsvText: csvText, ConvertStats: stats}, nil
}

// excelToCsvText 转换Excel为CSV文本
func excelToCsvText(ctx *app.Context, inputPath string) (string, error) {
	// 打开Excel文件
	f, err := excelize.OpenFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("打开Excel文件失败: %v", err)
	}
	defer f.Close()

	// 获取所有工作表名称
	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return "", fmt.Errorf("Excel文件没有工作表")
	}

	// 处理第一个工作表（如果有多个工作表，只处理第一个）
	sheetName := sheetList[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return "", fmt.Errorf("读取工作表 %s 失败: %v", sheetName, err)
	}

	// 构建CSV文本内容（带UTF-8 BOM）
	var csvBuilder strings.Builder
	// 写入UTF-8 BOM（\xEF\xBB\xBF）
	csvBuilder.WriteString("\xEF\xBB\xBF")

	// 创建CSV写入器（写入到内存）
	writer := csv.NewWriter(&csvBuilder)
	defer writer.Flush()

	// 写入所有行
	rowCount := 0
	for _, row := range rows {
		// 处理空行：如果行中所有单元格都为空，则跳过
		isEmpty := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			continue
		}

		// 写入CSV行
		err = writer.Write(row)
		if err != nil {
			return "", fmt.Errorf("写入CSV行失败: %v", err)
		}
		rowCount++
	}

	// 🔥 重要：在获取字符串之前，必须先刷新缓冲区，确保所有数据都写入到 strings.Builder
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("刷新CSV写入器失败: %v", err)
	}

	csvText := csvBuilder.String()
	logger.Infof(ctx, "[excelToCsvText] 转换成功: %s -> CSV文本 (工作表: %s, 行数: %d)", inputPath, sheetName, rowCount)
	return csvText, nil
}

// ExcelToCsvTextTemplate Excel转CSV文本配置
var ExcelToCsvTextTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Excel转CSV文本",
		Desc:     `支持将Excel文件（.xlsx、.xls）转换为CSV文本内容。转换后的CSV文本使用UTF-8编码并包含BOM，确保在Excel中打开时中文显示正常。如果Excel文件包含多个工作表，只转换第一个工作表。应用场景：数据导出、格式转换、数据迁移等。`,
		Tags:     []string{"Office工具", "格式转换", "Excel", "CSV"},
		Request:  &ExcelToCsvTextReq{},
		Response: &ExcelToCsvTextResp{},
	},
}

func init() {
	// 💡 packageContext 是在当前目录下系统自动创建的变量，直接用即可，无需定义
	packageContext.POST("office_excel_to_csv.form", ExcelToCsv, ExcelToCsvTemplate)

	// 注册Form函数 - Excel转CSV文本
	packageContext.POST("office_excel_to_csv_text.form", ExcelToCsvText, ExcelToCsvTextTemplate)
}
```

### office_excel_to_json.go

```go
//<文件名>office_excel_to_json.go</文件名>

package excelorcsv

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
	"github.com/xuri/excelize/v2"
)

// ExcelToJsonReq Excel转JSON请求结构体
type ExcelToJsonReq struct {
	// 框架标签：widget:"type:files;accept:.xlsx,.xls;max_size:50MB;max_count:1" - 文件上传组件
	InputFiles string `json:"input_files" widget:"name:上传Excel文件;type:files;accept:.xlsx,.xls;max_size:50MB;max_count:1" validate:"required"`

	// 框架标签：widget:"type:input" - 工作表名称（可选，默认第一个工作表）
	SheetName string `json:"sheet_name" widget:"name:工作表名称（可选）;type:input;placeholder:留空则使用第一个工作表"`

	// 框架标签：widget:"type:switch" - 是否使用第一行作为键名
	UseFirstRowAsKeys bool `json:"use_first_row_as_keys" widget:"name:使用第一行作为键名;type:switch"`

	// 框架标签：widget:"type:switch" - 是否跳过空行
	SkipEmptyRows bool `json:"skip_empty_rows" widget:"name:跳过空行;type:switch"`
}

// ExcelToJsonResp Excel转JSON响应结构体
type ExcelToJsonResp struct {
	// JSON文本内容
	JsonText string `json:"json_text" widget:"name:JSON文本内容;type:text_area"`

	// 转换统计信息
	ConvertStats string `json:"convert_stats" widget:"name:转换统计;type:text_area"`
}

// ExcelToJson Excel转JSON入口（SDK 注册用）：解析请求 → 调 DoExcelToJson → 写响应
func ExcelToJson(ctx *app.Context, resp response.Response) error {
	var req ExcelToJsonReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoExcelToJson(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoExcelToJson Excel转JSON业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoExcelToJson(ctx *app.Context, req *ExcelToJsonReq) (*ExcelToJsonResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	if len(inputFiles) == 0 {
		return &ExcelToJsonResp{JsonText: "", ConvertStats: "错误: 没有找到输入文件"}, nil
	}

	file := inputFiles[0]
	if file == "" {
		return &ExcelToJsonResp{JsonText: "", ConvertStats: fmt.Sprintf("错误: 文件 %s 没有本地路径", filepath.Base(file))}, nil
	}

	jsonText, stats, err := excelToJson(ctx, file, req.SheetName, req.UseFirstRowAsKeys, req.SkipEmptyRows)
	if err != nil {
		logger.Errorf(ctx, "[ExcelToJson] 转换Excel失败 %s: %v", filepath.Base(file), err)
		return &ExcelToJsonResp{JsonText: "", ConvertStats: fmt.Sprintf("转换失败: %v", err)}, nil
	}

	return &ExcelToJsonResp{JsonText: jsonText, ConvertStats: stats}, nil
}

// excelToJson 转换Excel为JSON
func excelToJson(ctx *app.Context, inputPath string, sheetName string, useFirstRowAsKeys bool, skipEmptyRows bool) (string, string, error) {
	// 打开Excel文件
	f, err := excelize.OpenFile(inputPath)
	if err != nil {
		return "", "", fmt.Errorf("打开Excel文件失败: %v", err)
	}
	defer f.Close()

	// 获取所有工作表名称
	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return "", "", fmt.Errorf("Excel文件没有工作表")
	}

	// 确定要使用的工作表
	targetSheet := sheetName
	if targetSheet == "" {
		targetSheet = sheetList[0]
	} else {
		// 检查指定的工作表是否存在
		found := false
		for _, s := range sheetList {
			if s == targetSheet {
				found = true
				break
			}
		}
		if !found {
			return "", "", fmt.Errorf("工作表 %s 不存在，可用工作表: %v", targetSheet, sheetList)
		}
	}

	// 读取工作表数据
	rows, err := f.GetRows(targetSheet)
	if err != nil {
		return "", "", fmt.Errorf("读取工作表 %s 失败: %v", targetSheet, err)
	}

	if len(rows) == 0 {
		return "", "", fmt.Errorf("工作表 %s 为空", targetSheet)
	}

	// 处理数据
	var result []map[string]interface{}
	var keys []string

	// 如果使用第一行作为键名
	if useFirstRowAsKeys && len(rows) > 0 {
		keys = rows[0]
		// 清理键名（去除空格）
		for i, key := range keys {
			keys[i] = strings.TrimSpace(key)
			if keys[i] == "" {
				keys[i] = fmt.Sprintf("column_%d", i+1)
			}
		}
		rows = rows[1:] // 跳过第一行
	}

	// 处理每一行数据
	rowCount := 0
	for _, row := range rows {
		// 检查是否为空行
		if skipEmptyRows {
			isEmpty := true
			for _, cell := range row {
				if strings.TrimSpace(cell) != "" {
					isEmpty = false
					break
				}
			}
			if isEmpty {
				continue
			}
		}

		// 构建行数据
		rowData := make(map[string]interface{})
		if useFirstRowAsKeys {
			// 使用第一行作为键名
			for i, cell := range row {
				key := keys[i]
				if key == "" {
					key = fmt.Sprintf("column_%d", i+1)
				}
				rowData[key] = cell
			}
		} else {
			// 使用列索引作为键名
			for i, cell := range row {
				rowData[fmt.Sprintf("column_%d", i+1)] = cell
			}
		}

		result = append(result, rowData)
		rowCount++
	}

	// 转换为JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("转换为JSON失败: %v", err)
	}

	jsonText := string(jsonBytes)
	stats := fmt.Sprintf("转换完成！\n工作表: %s\n总行数: %d\n有效行数: %d", targetSheet, len(rows), rowCount)

	logger.Infof(ctx, "[excelToJson] 转换成功: %s -> JSON (工作表: %s, 行数: %d)", inputPath, targetSheet, rowCount)
	return jsonText, stats, nil
}

// ExcelExtractColumnReq Excel提取指定列请求结构体
type ExcelExtractColumnReq struct {
	// 框架标签：widget:"type:files;accept:.xlsx,.xls;max_size:50MB;max_count:1" - 文件上传组件
	InputFiles string `json:"input_files" widget:"name:上传Excel文件;type:files;accept:.xlsx,.xls;max_size:50MB;max_count:1" validate:"required"`

	// 框架标签：widget:"type:input" - 工作表名称（可选，默认第一个工作表）
	SheetName string `json:"sheet_name" widget:"name:工作表名称（可选）;type:input;placeholder:留空则使用第一个工作表"`

	// 框架标签：widget:"type:input" - 列名或列索引（支持列名如username，或列索引如A、B、1、2）
	Column string `json:"column" widget:"name:列名或列索引;type:input;placeholder:例如: username、A、B、1、2（支持第一行的列名）" validate:"required"`

	// 框架标签：widget:"type:switch" - 是否跳过空行
	SkipEmptyRows bool `json:"skip_empty_rows" widget:"name:跳过空行;type:switch"`

	// 框架标签：widget:"type:switch" - 是否跳过第一行（表头）
	SkipFirstRow bool `json:"skip_first_row" widget:"name:跳过第一行（表头）;type:switch"`
}

// ExcelExtractColumnResp Excel提取指定列响应结构体
type ExcelExtractColumnResp struct {
	// JSON字符串数组
	JsonArray string `json:"json_array" widget:"name:JSON字符串数组;type:text_area"`

	// 提取统计信息
	ExtractStats string `json:"extract_stats" widget:"name:提取统计;type:text_area"`
}

// ExcelExtractColumn Excel提取指定列入口（SDK 注册用）：解析请求 → 调 DoExcelExtractColumn → 写响应
func ExcelExtractColumn(ctx *app.Context, resp response.Response) error {
	var req ExcelExtractColumnReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoExcelExtractColumn(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoExcelExtractColumn Excel提取指定列业务逻辑：(ctx, req) → (res, err)，便于单测与复用
func DoExcelExtractColumn(ctx *app.Context, req *ExcelExtractColumnReq) (*ExcelExtractColumnResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	if len(inputFiles) == 0 {
		return &ExcelExtractColumnResp{JsonArray: "", ExtractStats: "错误: 没有找到输入文件"}, nil
	}

	file := inputFiles[0]
	if file == "" {
		return &ExcelExtractColumnResp{JsonArray: "", ExtractStats: fmt.Sprintf("错误: 文件 %s 没有本地路径", filepath.Base(file))}, nil
	}

	jsonArray, stats, err := excelExtractColumn(ctx, file, req.SheetName, req.Column, req.SkipEmptyRows, req.SkipFirstRow)
	if err != nil {
		logger.Errorf(ctx, "[ExcelExtractColumn] 提取列失败 %s: %v", filepath.Base(file), err)
		return &ExcelExtractColumnResp{JsonArray: "", ExtractStats: fmt.Sprintf("提取失败: %v", err)}, nil
	}

	return &ExcelExtractColumnResp{JsonArray: jsonArray, ExtractStats: stats}, nil
}

// excelExtractColumn 从Excel提取指定列
func excelExtractColumn(ctx *app.Context, inputPath string, sheetName string, column string, skipEmptyRows bool, skipFirstRow bool) (string, string, error) {
	// 打开Excel文件
	f, err := excelize.OpenFile(inputPath)
	if err != nil {
		return "", "", fmt.Errorf("打开Excel文件失败: %v", err)
	}
	defer f.Close()

	// 获取所有工作表名称
	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return "", "", fmt.Errorf("Excel文件没有工作表")
	}

	// 确定要使用的工作表
	targetSheet := sheetName
	if targetSheet == "" {
		targetSheet = sheetList[0]
	} else {
		// 检查指定的工作表是否存在
		found := false
		for _, s := range sheetList {
			if s == targetSheet {
				found = true
				break
			}
		}
		if !found {
			return "", "", fmt.Errorf("工作表 %s 不存在，可用工作表: %v", targetSheet, sheetList)
		}
	}

	// 获取工作表的所有行数据（用于确定行数和查找列名）
	rows, err := f.GetRows(targetSheet)
	if err != nil {
		return "", "", fmt.Errorf("读取工作表 %s 失败: %v", targetSheet, err)
	}

	if len(rows) == 0 {
		return "", "", fmt.Errorf("工作表 %s 为空", targetSheet)
	}

	// 解析列索引：先尝试通过第一行的列名查找，如果找不到再尝试解析为字母列名或数字索引
	var colIndex int
	if len(rows) > 0 {
		// 尝试在第一行中查找匹配的列名（不区分大小写）
		firstRow := rows[0]
		found := false
		columnLower := strings.ToLower(strings.TrimSpace(column))
		for i, cellValue := range firstRow {
			if strings.ToLower(strings.TrimSpace(cellValue)) == columnLower {
				colIndex = i
				found = true
				logger.Infof(ctx, "[excelExtractColumn] 通过列名 '%s' 找到列索引: %d", column, colIndex+1)
				break
			}
		}
		if !found {
			// 如果第一行找不到，尝试解析为字母列名或数字索引
			parsedIndex, err := parseColumnIndex(column)
			if err != nil {
				return "", "", fmt.Errorf("无法找到列 '%s'（第一行中不存在，且无法解析为列索引）: %v", column, err)
			}
			colIndex = parsedIndex
			logger.Infof(ctx, "[excelExtractColumn] 通过列索引解析找到列: %s -> %d", column, colIndex+1)
		}
	} else {
		// 如果没有第一行，直接尝试解析为字母列名或数字索引
		parsedIndex, err := parseColumnIndex(column)
		if err != nil {
			return "", "", fmt.Errorf("解析列索引失败: %v", err)
		}
		colIndex = parsedIndex
	}

	// 获取最大行数
	maxRow := len(rows)
	logger.Infof(ctx, "[excelExtractColumn] 工作表 %s 共有 %d 行数据", targetSheet, maxRow)

	// 提取列数据
	var columnData []string
	startRow := 1 // Excel行号从1开始
	if skipFirstRow {
		startRow = 2 // 跳过第一行（表头）
	}

	validCount := 0

	// 遍历每一行，使用GetCellValue直接获取指定列的单元格值
	for rowIndex := startRow; rowIndex <= maxRow; rowIndex++ {
		// 将列索引和行号转换为单元格名称（如 A1, B2）
		cellName, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex)
		if err != nil {
			logger.Warnf(ctx, "[excelExtractColumn] 转换单元格坐标失败 (列:%d, 行:%d): %v", colIndex+1, rowIndex, err)
			continue
		}

		// 直接获取单元格值
		cellValue, err := f.GetCellValue(targetSheet, cellName)
		if err != nil {
			logger.Warnf(ctx, "[excelExtractColumn] 获取单元格值失败 (%s): %v", cellName, err)
			cellValue = "" // 如果获取失败，使用空字符串
		}

		// 检查是否跳过空行
		if skipEmptyRows && strings.TrimSpace(cellValue) == "" {
			continue
		}

		columnData = append(columnData, cellValue)
		validCount++
	}

	// 转换为JSON字符串数组
	jsonBytes, err := json.Marshal(columnData)
	if err != nil {
		return "", "", fmt.Errorf("转换为JSON失败: %v", err)
	}

	jsonArray := string(jsonBytes)
	totalRows := maxRow - startRow + 1
	stats := fmt.Sprintf("提取完成！\n工作表: %s\n列: %s (索引: %d)\n总行数: %d\n有效行数: %d", targetSheet, column, colIndex+1, totalRows, validCount)

	logger.Infof(ctx, "[excelExtractColumn] 提取成功: %s -> 列 %s (工作表: %s, 有效行数: %d)", inputPath, column, targetSheet, validCount)
	return jsonArray, stats, nil
}

// parseColumnIndex 解析列索引（支持A、B、C...或AA、AB...或1、2、3...）
func parseColumnIndex(column string) (int, error) {
	column = strings.TrimSpace(strings.ToUpper(column))

	// 如果是字母（如A、B、C...或AA、AB...），转换为数字索引
	if len(column) > 0 && column[0] >= 'A' && column[0] <= 'Z' {
		// 验证所有字符都是字母
		for _, c := range column {
			if c < 'A' || c > 'Z' {
				return 0, fmt.Errorf("无效的列格式: %s，字母列只能包含A-Z", column)
			}
		}

		// 将字母列转换为数字索引（Excel列索引：A=1, B=2, ..., Z=26, AA=27, AB=28, ...）
		// 然后转换为0-based索引
		colIndex := 0
		for i := 0; i < len(column); i++ {
			colIndex = colIndex*26 + int(column[i]-'A'+1)
		}
		return colIndex - 1, nil
	}

	// 如果是数字，转换为索引（1-based转0-based）
	var colNum int
	_, err := fmt.Sscanf(column, "%d", &colNum)
	if err != nil {
		return 0, fmt.Errorf("无效的列格式: %s，请使用字母（A-Z、AA-ZZ...）或数字（1、2、3...）", column)
	}

	if colNum < 1 {
		return 0, fmt.Errorf("列索引必须大于0")
	}

	return colNum - 1, nil
}

// ExcelToJsonTemplate Excel转JSON配置
var ExcelToJsonTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Excel转JSON",
		Desc:     `支持将Excel文件（.xlsx、.xls）转换为JSON格式。可以选择使用第一行作为键名，支持跳过空行，可以指定工作表名称。应用场景：数据导出、格式转换、API数据准备等。`,
		Tags:     []string{"Office工具", "格式转换", "Excel", "JSON"},
		Request:  &ExcelToJsonReq{},
		Response: &ExcelToJsonResp{},
	},
}

// ExcelExtractColumnTemplate Excel提取指定列配置
var ExcelExtractColumnTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "Excel提取指定列",
		Desc:     `支持从Excel文件（.xlsx、.xls）中提取指定列的数据，返回JSON字符串数组。可以指定第一行的列名（如username、email等）、列字母（A、B、C...）或列索引（1、2、3...），支持跳过空行和表头。应用场景：数据提取、批量处理、数据筛选等。`,
		Tags:     []string{"Office工具", "数据提取", "Excel", "JSON"},
		Request:  &ExcelExtractColumnReq{},
		Response: &ExcelExtractColumnResp{},
	},
}

func init() {
	// 💡 packageContext 是在当前目录下系统自动创建的变量，直接用即可，无需定义
	// 注册Form函数 - Excel转JSON
	packageContext.POST("office_excel_to_json.form", ExcelToJson, ExcelToJsonTemplate)

	// 注册Form函数 - Excel提取指定列
	packageContext.POST("office_excel_extract_column.form", ExcelExtractColumn, ExcelExtractColumnTemplate)
}
```
