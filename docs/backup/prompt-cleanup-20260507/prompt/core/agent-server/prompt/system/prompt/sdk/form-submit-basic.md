# SDK Form 提交任务包

本文档用于单 Form 场景：一次性提交、文件处理、生成转换、导入、发送、触发动作等。它是一个闭环任务包：说明什么时候用 Form、前端长什么样、目录/路由怎么命名、最小代码结构、文件处理、结果展示、验证方式和常见错误。字段组件细节统一读取 `/system/prompt/sdk/widget-system`。

## 什么时候用 Form

用户需求包含以下特征时，优先 Form：

- 用户填写一组参数，提交后执行一次动作。
- 上传文件后转换、解析、压缩、提取、生成结果。
- 提交一个业务入口，例如报名、投票、报修、复杂派单、收银结算。
- 触发一次副作用，例如发送通知、导入数据、生成报告。
- 不需要分页列表，不需要长期维护一批记录。

Form 前端形态：Element Plus `el-form` 风格的表单界面。前端按 Request 结构体字段渲染输入控件，用户提交后，后端执行一次处理，并把 Response 渲染成结果区、文件、链接或提示。

不要把长期记录管理、分页搜索列表、统计趋势图写成单 Form。长期记录用 Table，统计图表用 Chart。

## 最小目录和路由

单 Form 通常一个业务目录 + 一个 `.form` 函数：

```text
/用户/应用/pdf
  extract_text.form
```

路由命名：

- 动作用动词或动宾短语：`extract_text.form`、`image_resize.form`、`excel_import.form`。
- 路由最后一段必须 `.form`。
- 注册使用 `packageContext.POST("xxx.form", Handler, XxxFormTemplate)`。

## 最小结构

一个基础 Form 文件通常包含：

1. Request：前端表单字段，使用 widget 和 validate 描述输入。
2. Response：提交后的结果字段，使用 widget 描述展示。
3. FormTemplate：声明 Request、Response、名称、描述和可选 OnSelectFuzzy。
4. Handler：绑定并校验 Request，调用业务函数，`resp.Form(res).Build()`。
5. `init()`：注册 `.form` 路由。

示例骨架：

```go
package pdf

import (
    "fmt"
    "strings"

    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
    "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
)

type PdfExtractTextReq struct {
    InputFiles string `json:"input_files" widget:"name:上传PDF文件;type:files;accept:.pdf;max_size:100MB;max_count:10" validate:"required"`
}

type PdfExtractTextResp struct {
    ExtractedText string `json:"extracted_text" widget:"name:提取文本;type:text_area"`
    Summary       string `json:"summary" widget:"name:处理结果;type:text_area"`
}

var PdfExtractTextTemplate = &app.FormTemplate{
    BaseConfig: app.BaseConfig{
        Name:     "PDF文本提取",
        Desc:     "上传 PDF 文件并提取文本内容。",
        Tags:     []string{"PDF处理", "文件处理"},
        Request:  &PdfExtractTextReq{},
        Response: &PdfExtractTextResp{},
    },
}

func PdfExtractText(ctx *app.Context, resp response.Response) error {
    var req PdfExtractTextReq
    if err := ctx.ShouldBindValidate(&req); err != nil {
        return err
    }
    res, err := DoPdfExtractText(ctx, &req)
    if err != nil {
        return err
    }
    return resp.Form(res).Build()
}

func DoPdfExtractText(ctx *app.Context, req *PdfExtractTextReq) (*PdfExtractTextResp, error) {
    fs := ctx.GetFS()
    inputFiles := fs.DownloadFiles(req.InputFiles)
    defer fs.RemoveFiles(inputFiles)

    if len(inputFiles) == 0 {
        return nil, fmt.Errorf("没有找到输入文件")
    }

    names := make([]string, 0, len(inputFiles))
    for _, file := range inputFiles {
        names = append(names, file)
    }

    return &PdfExtractTextResp{
        ExtractedText: "这里返回实际提取结果",
        Summary:       fmt.Sprintf("已处理 %d 个文件：%s", len(inputFiles), strings.Join(names, "、")),
    }, nil
}

func init() {
    packageContext.POST("extract_text.form", PdfExtractText, PdfExtractTextTemplate)
}
```

实际项目以仓库现有 SDK API 为准；写代码前优先读匹配案例，例如 `/system/prompt/case_catalog/form/pdf`、`/system/prompt/case_catalog/form/excelorcsv`、`/system/prompt/case_catalog/form/images`。

## Request 字段规则

Form 的 Request 就是前端表单：

- 文本输入用 `input` 或 `text_area`。
- 枚举选择用 `select` / `radio`，要写 `options`；静态 `select` / `multiselect` 的 `options_colors` 只用不带 `#` 的 6 位十六进制 `RRGGBB`。
- 多选用 `multiselect` / `checkbox`。
- 文件上传用 `files`，Go 字段通常是 `string`，提交后通过 `ctx.GetFS().DownloadFiles(...)` 下载。
- 用户、部门、关联业务对象选择不确定时，读 `/system/prompt/sdk/widget-system`，必要时实现 OnSelectFuzzy。
- 后端自动生成字段不要放在 Request；只放用户需要填写的字段。
- 当前请求上下文能确定的字段不要让用户填写，例如出价人、投票人、评价提交人、操作人、收银员等；在 Handler 中用 `ctx.GetRequestUser()` 赋值，必要时用 `ctx.GetRequestUserDept()` 记录部门。

不要在 Request 里嵌入分页结构，也不要把 Table 的 Model 原样当成 Form Request。

## 单 Form 必备 SDK 能力

单 Form 虽然不负责列表搜索，但仍然要把以下能力规划清楚，否则前端不会出现动态选择、文件结果或跳转入口。

### 动态选择和 OnSelectFuzzy

Form 里选择数据库对象时，不要写死超长 `options`。使用 `select` / `multiselect` + `callback:"OnSelectFuzzy"`，并在 `FormTemplate.BaseConfig.OnSelectFuzzyMap` 注册回调。

```go
type EvaluationSubmitReq struct {
    ObjectID int `json:"object_id" widget:"name:评价对象;type:select" validate:"required" callback:"OnSelectFuzzy"`
    Score    int `json:"score" widget:"name:评分;type:slider;min:1;max:5" validate:"required,min=1,max=5"`
}

var EvaluationSubmitTemplate = &app.FormTemplate{
    BaseConfig: app.BaseConfig{
        Name:     "提交评价",
        Request:  &EvaluationSubmitReq{},
        Response: &EvaluationSubmitResp{},
        OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
            "object_id": onSelectFuzzyEvaluationObject,
        },
    },
}
```

联动字段用 `depend_on`，依赖字段必须排在被联动字段上方。例如先选投票主题，再选投票选项。

### link 跳转

Form 提交后经常要跳回 Table、跳到 Chart 或跳到另一个 Form。Response 中用 `link` 字段，不手拼 URL。

```go
type EvaluationSubmitResp struct {
    Message    string `json:"message" widget:"name:提交结果;type:text"`
    RecordLink string `json:"record_link" widget:"name:查看评价记录;type:link;target:_blank"`
}

func DoEvaluationSubmit(ctx *app.Context, req *EvaluationSubmitReq) (*EvaluationSubmitResp, error) {
    recordLink, _ := ctx.BuildFunctionUrlWithText(
        "evaluation_record_list.table",
        EvaluationRecord{ObjectID: req.ObjectID},
        "查看评价记录",
    )
    return &EvaluationSubmitResp{Message: "提交成功", RecordLink: recordLink}, nil
}
```

跳 Table 时，参数会按目标 Table 的 Request 字段转成筛选条件；跳 Form 或 Chart 时，参数会按目标 Request 转成表单参数。

### 文件上传、下载和返回文件

Form 文件链路必须按 SDK 文件协议写完整：

1. Request 文件字段使用 `type:files`，Go 类型必须是 `string`。
2. 通过 `ctx.GetFS().DownloadFiles(req.InputFiles)` 把前端上传的文件引用下载成本地路径。
3. 用 `defer fs.RemoveFiles(inputFiles)` 清理下载目录。
4. 输出文件写到 `fs.GetTraceOutputDir()`，不要写到输入文件目录。
5. 用 `fs.ResponseFiles([]string{outputPath})` 或 `fs.ResponseDirFiles(outputDir)` 上传输出文件并得到 files 字段值。
6. Response 文件字段也使用 `type:files`，Go 类型也是 `string`。

```go
type ExcelConvertReq struct {
    InputFiles string `json:"input_files" widget:"name:上传Excel;type:files;accept:.xlsx,.xls;max_size:50MB;max_count:1" validate:"required"`
}

type ExcelConvertResp struct {
    OutputFiles string `json:"output_files" widget:"name:转换结果;type:files"`
    Summary     string `json:"summary" widget:"name:处理摘要;type:text_area"`
}

func DoExcelConvert(ctx *app.Context, req *ExcelConvertReq) (*ExcelConvertResp, error) {
    fs := ctx.GetFS()
    inputFiles := fs.DownloadFiles(req.InputFiles)
    defer fs.RemoveFiles(inputFiles)
    if len(inputFiles) == 0 {
        return nil, fmt.Errorf("没有找到输入文件")
    }

    outputDir := fs.GetTraceOutputDir()
    outputPath := filepath.Join(outputDir, "result.xlsx")
    // 在 outputPath 写入结果文件

    return &ExcelConvertResp{
        OutputFiles: fs.ResponseFiles([]string{outputPath}),
        Summary:     "转换完成",
    }, nil
}
```

规则：

- `files` 字段值是 `bucket/object_key` 字符串，多文件用英文逗号分隔；不要把字段写成 `[]string`。
- `accept`、`max_size`、`max_count` 要根据业务限制写清楚。
- 如果处理过程直接复用输入文件，也要复制到输出目录后再 `ResponseFiles`，不能返回即将被 `RemoveFiles` 删除的输入路径。
- 如果 Python 运行时产生文件，Python 返回 `output_files` 后，Go 侧仍要用 SDK 文件系统上传并放入 Response 的 files 字段。

### GORM 查询和后置关联填充边界

单 Form 可以读取数据库并返回一个结果，但它不负责分页列表。如果 Form 需要展示关联对象名称或统计摘要：

- 少量关联对象可以在业务查询中 `Preload`。
- 多个关联或统计值先批量查询，再后置关联填充到 Response 字段。
- 如果用户需要反复查看、搜索、分页这些结果，应拆 Table，而不是在 Form Response 中拼一个大列表。

### hide、筛选字段和 Form 的边界

Form Request/Response 通常不需要 `hide:"create,update"`，因为它不是 Table 列表。只有当同一个结构体也作为 Table Model 使用时，才需要严格写 `hide`。

- `hide:"create,update"`：只给 Table 列表字段、只读计算字段、link 字段使用。
- 筛选字段：单 Form 不使用；Table 筛选字段写在 Request 中，并嵌入 `query.PageSortReq`。
- 如果用户要求“提交后能查看历史记录”，不要把历史列表塞进 Form Response，应新增 Table，并在 Table Request 中规划自定义搜索参数。

### Form 写入 Table 数据

Form 写入长期记录时，要按目标 Table 的字段规则建模：

- Form Request 只放用户填写字段，不放 ID、创建时间、更新时间等系统字段。
- Table Model 中系统字段、计算字段、关联展示字段要用 `hide:"create,update"` 控制展示。
- 写入后如需展示记录详情，用 Response link 跳到对应 Table。
- 跨表写入必须用事务，避免只写入一半数据。

## Response 字段规则

Form 的 Response 是提交结果，不是 Chart，也不是分页表格：

- 普通文本结果用 `text` / `text_area`。
- 返回文件时使用 `files` 字段，先用 SDK 文件系统生成或上传输出文件，再把文件标识放进 Response。
- 需要跳转到 Table 或 Chart 时，用 `link` 字段和 `ctx.BuildFunctionUrlWithText`。
- 只返回成功/失败提示时，也要用明确字段，例如 `success`、`message`、`detail`。

如果用户要求可视化趋势、占比、看板，不要把 ECharts 数据塞进 Form Response，应拆 Chart。

## 文件处理规则

文件处理是最常见的单 Form：

1. Request 使用 `files` 组件。
2. Handler 中 `ctx.ShouldBindValidate(&req)`。
3. 使用 `ctx.GetFS().DownloadFiles(req.InputFiles)` 获取本地路径。
4. 处理完成后清理输入临时文件。
5. 如需输出文件，写入 `ctx.GetFS().GetTraceOutputDir()`，再用 `ResponseFiles` 或 `ResponseDirFiles` 上传并在 Response 中用 `files` 展示。
6. 处理失败要返回清晰业务错误；系统级错误要带足日志上下文。

文件处理不要默认建 Table。只有用户明确要求保存处理历史、查看导入记录、二次管理结果时，才加 Table。

## PRD 中怎么描述 Form

创建类 PRD 不要只写“创建一个 Form”。应写清楚：

```text
函数类型判断：
- PDF 文本提取是一次性文件处理动作，不需要维护长期记录，因此选择 `extract_text.form`。
- 前端会渲染为 Element 表单，用户上传 PDF 文件并提交。
- 后端下载上传文件、执行提取逻辑，最后在结果区展示提取文本和处理统计。
```

“表单字段”部分只列用户需要填写的输入字段；“返回结果”部分列提交后展示的字段、文件或链接。

PRD 必须额外写：

- 落地目录和函数清单：例如“确认后创建 `/用户/应用/pdf_tools`，生成 `extract_text.form`”，并说明前端是 Element 表单。
- 示例数据：至少给一条用户输入样例和一条提交后的返回结果样例，例如“输入文件：合同.pdf；处理模式：提取正文；返回：提取文本、页数、下载链接”。
- 确认后创建内容：在确认语前写清“确认后我将创建目录：xxx，并生成：xxx.form”。

## 验证

写完后必须：

1. `build_workspace`。
2. 使用 `run_form_submit` 构造最小输入，验证 Form 能提交并返回结果。
3. 如果有文件上传，使用真实或案例文件验证下载、处理、返回文件链路。
4. 如果有 OnSelectFuzzy，使用 `run_on_select_fuzzy` 验证搜索和回显。

验证失败时先读错误，再修复，不要直接总结完成。

## 推荐案例

- Excel/CSV 转换：`/system/prompt/case_catalog/form/excelorcsv`
- PDF 处理：`/system/prompt/case_catalog/form/pdf`
- 图片处理：`/system/prompt/case_catalog/form/images`
- NLP 文本处理：`/system/prompt/case_catalog/form/nlp`

## 常见错误

- 把单次文件转换写成 Table。
- 路由不是 `.form`，或 `.form` 路由注册了 TableTemplate。
- Request 放入 ID、创建时间、更新时间等系统展示字段。
- Response 返回 chart 结构，导致前端不能按 Chart 渲染。
- 文件上传字段没有 `accept`、`max_size`、`max_count`。
- 下载输入文件后没有清理临时文件。
- 写完没有 `run_form_submit`。
