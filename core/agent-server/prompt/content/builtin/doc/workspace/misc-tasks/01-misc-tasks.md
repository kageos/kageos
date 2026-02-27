# 杂活 / 通用任务

当用户提的是**零散、单次、跨能力**的需求（不限定在当前项目）时，按本文档执行。典型如：画折线图、图片转格式、视频处理、解析 Excel、文件压缩等。

核心原则：**先看环境再搜，按「用途」选类型（form/table/chart），没有则问再建**。

---

## 函数类型说明（form / table / chart）

选 `search_tools` 的 template_type、以及选执行工具（run_form_submit / run_table_xxx / run_chart_query）时，按**用途和交互方式**判断，不要只看关键词。

| 类型 | 用途 | 交互方式 | 典型场景 | 调用方式 |
|------|------|----------|----------|----------|
| **form** | **干一件事、算一个结果**：一次提交、一次返回，不管理「一堆记录」。 | 用户填参数（输入框、选择、上传文件等）→ 提交 → 得到一份结果（文件、文本、图片等）。 | 图片/视频转格式、解析 Excel、根据内容生成文档、**根据数据生成一张图**（如「帮我画个折线图」且用户给了数据或要你算数据再出图）、调用某个 API、压缩文件。 | `run_form_submit(full_code_path, body)` |
| **table** | **管理一批记录**：增删改查「行数据」，有 id、多行、可筛选。 | 查列表（run_table_search）、新增行（run_table_create）、改行（run_table_update）、删行。数据持久在库里。 | 问卷列表、工单、客户表、订单、任何「要查列表 / 加一条 / 改一条 / 删一条」的业务。 | `run_table_search` / `run_table_create` / `run_table_update` |
| **chart** | **把已有数据画成图表展示**：数据通常来自某张 table 或已配置的数据源，用户是「看图表」「查图表」。 | 传查询条件 → 返回图表配置或数据（用于前端渲染折线/饼图等）。**不**是「用户给一串数据请你生成一张图」那种。 | 销售趋势图、占比饼图、某张业务表的统计图表；用户说「看一下 XX 的图表」「查一下 XX 数据图」时。 | `run_chart_query(full_code_path, body)` |

**如何区分「画折线图」**：

- 用户说「帮我画个折线图」并**给了数据**（或要你根据条件算数据再出图）→ 属于「干一件事、出一张图」→ 搜 **form**，用 `run_form_submit` 调「根据数据画图」类表单。
- 用户说「看一下销售数据的折线图」「查 XX 的图表」→ 属于「查已有数据/图表的展示」→ 搜 **chart**，用 `run_chart_query`。

**杂活类（转格式、解析、生成文档、根据数据出图）绝大多数是 form**；只有明确「查/管一批记录」用 table、「查已有图表」用 chart。

---

## 执行流程

### 0. 先看当前环境/目录

根据环境信息或 `read_dir()` 看当前目录下是否已有与任务相关的能力（如画图、转换相关）。有则直接选用执行；没有则进入下一步。

### 1. 再 search_tools

按关键词**搜索可用工具**（内置工具 + **system 用户下**已注册的表单/表格/图表函数）。根据用户意图提取关键词，调用 `search_tools(keyword, template_type, limit)`。

- **keyword**（必填）：多关键词用 `|` 分隔（OR 语义），如 `折线图|chart|画图`、`图片|png|转换`。
- **template_type**（可选，建议传）：按函数类型过滤。**绝大部分杂活、画图、转格式、解析文件等都由 form 完成**，应传 `template_type=form` 缩小范围、避免混入大量 table/chart。仅当用户明确是「表格增删改查」或「图表数据查询」时才传 `table` 或 `chart`；不传则返回全部类型。

### 2. 有则直接执行

若返回中有**已注册函数**，根据类型选用工具直接执行（form 用 `run_form_submit`、table 用 `run_table_search` 等），无需询问用户。

### 3. 无则先问再建

若未匹配到可用函数：
- 禁止直接创建。
- 向用户说明「当前没有现成能力，需要走创建项目流程」。
- 用户同意后，转入创建项目流程：read_doc 创建项目文档 + SDK → 出 PRD → 确认 → write_go_file → build_workspace → 执行。

---

## files 组件传参（易错）

**表单（run_form_submit）的 request 与表格（run_table_create、run_table_update）的 model** 里凡是 `widget.type === "files"` 的字段，传参时须为**对象**（不是数组），内含 `files` 数组：

**正确**：
```json
{
  "input_files": {
    "files": [
      { "name": "xxx", "source_name": "原始文件名.mp4", "storage": "minio", "url": "https://...", "server_url": "http://...", "size": 12345, "is_uploaded": true }
    ],
    "widget_type": "files",
    "data_type": "struct"
  },
  "output_format": "mp4"
}
```

**错误**（直接传数组，会报 unmarshal 错误）：
```json
{
  "input_files": [ { "name": "xxx", "url": "..." } ],
  "output_format": "mp4"
}
```

---

## search_tools 返回格式

本工具用于**搜索可用工具**，返回包含：
- **【内置工具】**：匹配到的内置工具，每条为「名称：描述」。
- **【已注册函数】**：仅限 **system 用户下**已注册的表单/表格/图表函数；含一句统一调用方式说明；每条为 name、full_code_path、已使用 N 次（若有）、description、type、request（JSON）。不返回 response，减少冗余。

从已注册函数中取 full_code_path 作为 `run_form_submit` 的第一个参数，按 request 各字段的 code 构造 body。
