package table

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	pythonRuntime "github.com/kageos/kageos/sdk/agent-app/runtime/python"
)

type TableCleanMergeQueryReq struct {
	InputFiles        string `json:"input_files" widget:"name:上传表格文件;type:files;accept:.csv,.tsv,.xlsx,.xls,.json;max_size:100MB;max_count:20" validate:"required"`
	Operation         string `json:"operation" widget:"name:处理方式;type:select;options:自动清洗,纵向合并,横向关联,筛选排序,分组统计;render_default:自动清洗" validate:"required,oneof=自动清洗 纵向合并 横向关联 筛选排序 分组统计"`
	SheetName         string `json:"sheet_name" widget:"name:指定工作表(Excel可选);type:input;placeholder:留空默认首个工作表"`
	JoinKeys          string `json:"join_keys" widget:"name:关联主键;type:input;placeholder:多个列用逗号分隔，例如：订单号,门店ID;desc:横向关联模式显示并生效" validate:"required_if=Operation 横向关联"`
	JoinType          string `json:"join_type" widget:"name:关联方式;type:select;options:left,inner,right,outer;render_default:left;desc:横向关联模式显示并生效" validate:"required_if=Operation 横向关联,omitempty,oneof=left inner right outer"`
	FilterExpression  string `json:"filter_expression" widget:"name:筛选表达式;type:text_area;placeholder:例如：金额 > 1000 and 状态 == '已完成'"`
	GroupByColumns    string `json:"group_by_columns" widget:"name:分组列;type:input;placeholder:多个列用逗号分隔;desc:分组统计模式显示并生效" validate:"required_if=Operation 分组统计"`
	AggregationsJSON  string `json:"aggregations_json" widget:"name:聚合配置(JSON);type:text_area;placeholder:{\"销售额\":\"sum\",\"订单号\":\"count\"};desc:分组统计模式显示并生效" validate:"required_if=Operation 分组统计"`
	SelectColumns     string `json:"select_columns" widget:"name:保留列;type:input;placeholder:多个列用逗号分隔"`
	RenameColumnsJSON string `json:"rename_columns_json" widget:"name:重命名列(JSON);type:text_area;placeholder:{\"old_name\":\"new_name\"}"`
	SortBy            string `json:"sort_by" widget:"name:排序列;type:input;placeholder:多个列用逗号分隔"`
	SortDescending    bool   `json:"sort_descending" widget:"name:倒序排序;type:switch;render_default:false"`
	Limit             int    `json:"limit" widget:"name:结果行数限制;type:integer;render_default:0;placeholder:0 表示不限制"`
	RemoveDuplicates  bool   `json:"remove_duplicates" widget:"name:去重;type:switch;render_default:true"`
	DeduplicateBy     string `json:"deduplicate_by" widget:"name:按指定列去重;type:input;placeholder:多个列用逗号分隔"`
	TrimWhitespace    bool   `json:"trim_whitespace" widget:"name:清理文本首尾空格;type:switch;render_default:true"`
	FillNAValue       string `json:"fill_na_value" widget:"name:空值填充值;type:input;placeholder:留空则不填充"`
	OutputFormat      string `json:"output_format" widget:"name:输出格式;type:select;options:xlsx,csv,json;render_default:xlsx" validate:"required"`
	FileName          string `json:"file_name" widget:"name:输出文件名;type:input;render_default:table_result"`
	PreviewRows       int    `json:"preview_rows" widget:"name:预览行数;type:integer;render_default:20"`
}

type TableCleanMergeQueryResp struct {
	OutputFile      string `json:"output_file" widget:"name:输出文件;type:files"`
	PreviewMarkdown string `json:"preview_markdown" widget:"name:结果预览;type:text_area"`
	Summary         string `json:"summary" widget:"name:处理说明;type:text_area"`
	Statistics      string `json:"statistics" widget:"name:统计信息;type:text_area"`
}

func TableCleanMergeQuery(ctx *app.Context, resp response.Response) error {
	var req TableCleanMergeQueryReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoTableCleanMergeQuery(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

func DoTableCleanMergeQuery(ctx *app.Context, req *TableCleanMergeQueryReq) (*TableCleanMergeQueryResp, error) {
	fs := ctx.GetFS()
	inputFiles := fs.DownloadFiles(req.InputFiles)
	defer fs.RemoveFiles(inputFiles)

	files := inputFiles
	if len(files) == 0 {
		return nil, fmt.Errorf("没有找到输入文件")
	}

	outputDir := fs.GetTraceOutputDir()
	baseName := sanitizeOutputBaseName(req.FileName)
	outputExt := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(req.OutputFormat)), ".")
	if outputExt == "" {
		outputExt = "xlsx"
	}
	outputPath := filepath.Join(outputDir, baseName+"."+outputExt)

	requestPayload := struct {
		InputFiles        []map[string]string `json:"input_files"`
		Operation         string              `json:"operation"`
		SheetName         string              `json:"sheet_name"`
		JoinKeys          string              `json:"join_keys"`
		JoinType          string              `json:"join_type"`
		FilterExpression  string              `json:"filter_expression"`
		GroupByColumns    string              `json:"group_by_columns"`
		AggregationsJSON  string              `json:"aggregations_json"`
		SelectColumns     string              `json:"select_columns"`
		RenameColumnsJSON string              `json:"rename_columns_json"`
		SortBy            string              `json:"sort_by"`
		SortDescending    bool                `json:"sort_descending"`
		Limit             int                 `json:"limit"`
		RemoveDuplicates  bool                `json:"remove_duplicates"`
		DeduplicateBy     string              `json:"deduplicate_by"`
		TrimWhitespace    bool                `json:"trim_whitespace"`
		FillNAValue       string              `json:"fill_na_value"`
		OutputFormat      string              `json:"output_format"`
		OutputPath        string              `json:"output_path"`
		PreviewRows       int                 `json:"preview_rows"`
	}{}

	for _, file := range files {
		if file == "" {
			continue
		}
		requestPayload.InputFiles = append(requestPayload.InputFiles, map[string]string{
			"name": filepath.Base(file),
			"path": file,
		})
	}
	requestPayload.Operation = req.Operation
	requestPayload.SheetName = req.SheetName
	requestPayload.JoinKeys = req.JoinKeys
	requestPayload.JoinType = req.JoinType
	requestPayload.FilterExpression = req.FilterExpression
	requestPayload.GroupByColumns = req.GroupByColumns
	requestPayload.AggregationsJSON = req.AggregationsJSON
	requestPayload.SelectColumns = req.SelectColumns
	requestPayload.RenameColumnsJSON = req.RenameColumnsJSON
	requestPayload.SortBy = req.SortBy
	requestPayload.SortDescending = req.SortDescending
	requestPayload.Limit = req.Limit
	requestPayload.RemoveDuplicates = req.RemoveDuplicates
	requestPayload.DeduplicateBy = req.DeduplicateBy
	requestPayload.TrimWhitespace = req.TrimWhitespace
	requestPayload.FillNAValue = req.FillNAValue
	requestPayload.OutputFormat = outputExt
	requestPayload.OutputPath = outputPath
	requestPayload.PreviewRows = req.PreviewRows

	executor := pythonRuntime.NewExecutor(buildTableCleanMergeQueryPythonCode()).
		WithRequest(requestPayload).
		WithPackages("pandas", "openpyxl").
		WithOutputDir(outputDir).
		WithTimeout(5 * time.Minute)
	defer executor.Close()

	var result struct {
		PreviewMarkdown string `json:"preview_markdown"`
		Summary         string `json:"summary"`
		Statistics      string `json:"statistics"`
	}
	execResult, err := executor.ExecuteJSONWithResult(ctx, &result)
	if err != nil {
		return nil, err
	}

	var outputFiles string
	outputPaths, err := execResult.OutputFilePaths()
	if err != nil {
		return nil, err
	}
	if len(outputPaths) > 0 {
		outputFiles = fs.ResponseFiles(outputPaths)
	}

	return &TableCleanMergeQueryResp{
		OutputFile:      outputFiles,
		PreviewMarkdown: result.PreviewMarkdown,
		Summary:         result.Summary,
		Statistics:      result.Statistics,
	}, nil
}

func sanitizeOutputBaseName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "table_result"
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
	name = replacer.Replace(name)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if name == "" {
		return "table_result"
	}
	return name
}

func buildTableCleanMergeQueryPythonCode() string {
	return `import json
import os
import traceback
import pandas as pd

def split_csv_list(value):
    if not value:
        return []
    return [item.strip() for item in str(value).split(",") if item.strip()]

def read_csv_with_fallback(path, sep=","):
    last_error = None
    for encoding in ["utf-8-sig", "utf-8", "gb18030", "gbk"]:
        try:
            return pd.read_csv(path, encoding=encoding, sep=sep)
        except Exception as err:
            last_error = err
    raise last_error

def read_input_table(file_info, sheet_name):
    path = file_info["path"]
    ext = os.path.splitext(path)[1].lower()
    if ext == ".csv":
        return read_csv_with_fallback(path, ",")
    if ext == ".tsv":
        return read_csv_with_fallback(path, "\t")
    if ext in [".xlsx", ".xls"]:
        if sheet_name:
            return pd.read_excel(path, sheet_name=sheet_name)
        return pd.read_excel(path)
    if ext == ".json":
        with open(path, "r", encoding="utf-8") as f:
            raw = json.load(f)
        if isinstance(raw, list):
            return pd.DataFrame(raw)
        if isinstance(raw, dict):
            if len(raw) == 1:
                only_value = list(raw.values())[0]
                if isinstance(only_value, list):
                    return pd.DataFrame(only_value)
            return pd.json_normalize(raw)
    raise ValueError(f"不支持的表格文件类型: {ext}")

def normalize_dataframe(df, trim_whitespace, fill_na_value, remove_duplicates, deduplicate_by):
    df = df.copy()
    df.columns = [str(col).strip() for col in df.columns]
    df = df.dropna(how="all")
    if trim_whitespace:
        object_columns = df.select_dtypes(include=["object"]).columns
        for col in object_columns:
            df[col] = df[col].map(lambda v: v.strip() if isinstance(v, str) else v)
    if fill_na_value != "":
        df = df.fillna(fill_na_value)
    if remove_duplicates:
        subset = split_csv_list(deduplicate_by)
        subset = [col for col in subset if col in df.columns]
        df = df.drop_duplicates(subset=subset or None)
    return df

def dataframe_to_markdown(df, max_rows):
    if df.empty:
        return "(空结果)"
    preview = df.head(max_rows if max_rows and max_rows > 0 else 20).copy()
    columns = [str(col) for col in preview.columns]
    lines = []
    lines.append("| " + " | ".join([col.replace("|", "\\|") for col in columns]) + " |")
    lines.append("| " + " | ".join(["---"] * len(columns)) + " |")
    for _, row in preview.iterrows():
        values = []
        for col in preview.columns:
            value = row[col]
            if pd.isna(value):
                text = ""
            else:
                text = str(value)
            text = text.replace("|", "\\|").replace("\n", "<br>")
            values.append(text)
        lines.append("| " + " | ".join(values) + " |")
    return "\n".join(lines)

def kageos_entry(args, output_dir):
    input_files = args.get("input_files") or []
    if not input_files:
        raise ValueError("没有输入文件")

    operation = (args.get("operation") or "自动清洗").strip()
    sheet_name = (args.get("sheet_name") or "").strip()
    join_keys = split_csv_list(args.get("join_keys"))
    join_type = (args.get("join_type") or "left").strip() or "left"
    filter_expression = (args.get("filter_expression") or "").strip()
    group_by_columns = split_csv_list(args.get("group_by_columns"))
    aggregations_json = (args.get("aggregations_json") or "").strip()
    select_columns = split_csv_list(args.get("select_columns"))
    rename_columns_json = (args.get("rename_columns_json") or "").strip()
    sort_by = split_csv_list(args.get("sort_by"))
    sort_descending = bool(args.get("sort_descending"))
    limit = int(args.get("limit") or 0)
    remove_duplicates = bool(args.get("remove_duplicates"))
    deduplicate_by = args.get("deduplicate_by") or ""
    trim_whitespace = bool(args.get("trim_whitespace"))
    fill_na_value = "" if args.get("fill_na_value") is None else str(args.get("fill_na_value"))
    output_format = (args.get("output_format") or "xlsx").strip().lower()
    output_path = args.get("output_path") or os.path.join(output_dir, "table_result.xlsx")
    preview_rows = int(args.get("preview_rows") or 20)

    frames = []
    summaries = []
    for file_info in input_files:
        df = read_input_table(file_info, sheet_name)
        df = normalize_dataframe(df, trim_whitespace, fill_na_value, remove_duplicates, deduplicate_by)
        frames.append(df)
        summaries.append(f"{file_info['name']}: {len(df)} 行 x {len(df.columns)} 列")

    if not frames:
        raise ValueError("没有可处理的数据表")

    if operation in ["自动清洗", "纵向合并"]:
        result_df = pd.concat(frames, ignore_index=True, sort=False) if len(frames) > 1 else frames[0].copy()
    elif operation == "横向关联":
        if len(frames) < 2:
            raise ValueError("横向关联至少需要两个文件")
        if not join_keys:
            raise ValueError("横向关联必须提供 join_keys")
        result_df = frames[0].copy()
        for idx, df in enumerate(frames[1:], start=2):
            available_keys = [key for key in join_keys if key in result_df.columns and key in df.columns]
            if not available_keys:
                raise ValueError(f"第 {idx} 个表缺少可用关联键")
            result_df = result_df.merge(df, on=available_keys, how=join_type, suffixes=("", f"_{idx}"))
    elif operation in ["筛选排序", "分组统计"]:
        result_df = pd.concat(frames, ignore_index=True, sort=False) if len(frames) > 1 else frames[0].copy()
    else:
        raise ValueError(f"不支持的处理方式: {operation}")

    if rename_columns_json:
        rename_map = json.loads(rename_columns_json)
        result_df = result_df.rename(columns=rename_map)

    if filter_expression:
        result_df = result_df.query(filter_expression, engine="python")

    if operation == "分组统计":
        if not group_by_columns:
            raise ValueError("分组统计必须提供 group_by_columns")
        if not aggregations_json:
            raise ValueError("分组统计必须提供 aggregations_json")
        agg_map = json.loads(aggregations_json)
        result_df = result_df.groupby(group_by_columns, dropna=False).agg(agg_map).reset_index()

    if select_columns:
        keep_columns = [col for col in select_columns if col in result_df.columns]
        if not keep_columns:
            raise ValueError("指定的保留列都不存在")
        result_df = result_df[keep_columns]

    if sort_by:
        available_sort_columns = [col for col in sort_by if col in result_df.columns]
        if available_sort_columns:
            result_df = result_df.sort_values(by=available_sort_columns, ascending=not sort_descending)

    if limit > 0:
        result_df = result_df.head(limit)

    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    generated = True
    if output_format == "xlsx":
        result_df.to_excel(output_path, index=False)
    elif output_format == "csv":
        result_df.to_csv(output_path, index=False, encoding="utf-8-sig")
    elif output_format == "json":
        result_df.to_json(output_path, orient="records", force_ascii=False, indent=2)
    else:
        raise ValueError(f"不支持的输出格式: {output_format}")

    statistics_lines = [
        f"输入文件数: {len(frames)}",
        f"结果行数: {len(result_df)}",
        f"结果列数: {len(result_df.columns)}",
        f"结果列: {', '.join([str(col) for col in result_df.columns]) if len(result_df.columns) > 0 else '(无)'}",
    ]
    summary_lines = [f"处理方式: {operation}"] + summaries

    return {
        "data": {
            "preview_markdown": dataframe_to_markdown(result_df, preview_rows),
            "summary": "\n".join(summary_lines),
            "statistics": "\n".join(statistics_lines)
        },
        "output_files": [{"path": output_path, "name": os.path.basename(output_path)}]
    }`
}

var TableCleanMergeQueryTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "表格清洗合并查询",
		Desc:     `对 CSV、Excel、JSON 表格做统一清洗、纵向合并、横向关联、筛选排序、分组统计，并导出结果文件。适合报表整合、数据对账、临时分析和工作台查询结果再加工。`,
		Tags:     []string{"表格处理", "数据清洗", "合并", "关联", "查询", "筛选", "统计", "Excel", "CSV", "JSON"},
		Request:  &TableCleanMergeQueryReq{},
		Response: &TableCleanMergeQueryResp{},
	},
}

func init() {
	packageContext.POST("table_clean_merge_query.form", TableCleanMergeQuery, TableCleanMergeQueryTemplate)
}
