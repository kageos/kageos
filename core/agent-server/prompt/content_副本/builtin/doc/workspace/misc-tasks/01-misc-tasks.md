# 杂活 / 通用任务

当用户提的是**零散、单次、跨能力**的需求（不限定在当前项目、不一定是「做一个系统」）时，按本文档执行。典型如：图片转格式、视频处理、解析 Excel、文件压缩、格式转换等——**先搜现成能力，有则直接用，没有再问用户是否同意创建**。

---

## 一、什么是杂活/通用任务

| 特征 | 说明 |
|------|------|
| **典型说法** | 「帮我把图片转成 PNG」「处理一下这个视频」「有没有解析 Excel 的」「转个格式」「帮我压缩一下文件」 |
| **与其它任务区别** | 不是「做一个 XXX 系统」（创建项目）、不是「改当前项目里的 XXX」（修改项目）、不是「查当前项目下的列表/提交当前项目下的表单」（操作项目）。用户要的是**某一种能力**，可能平台里已有、也可能需要新建。 |
| **核心原则** | **先搜后用，没有则问再建**：先 search_tools 找是否已有现成函数，有则直接调用执行；没有则**不得直接创建**，须先问用户是否同意创建，同意后再走创建项目流程。 |

---

## 二、执行流程（必须按此顺序）

1. **先 search_tools**  
   根据用户意图提取关键词，调用 **search_tools(keyword)**。keyword 支持**多关键词用竖线 | 分隔**（OR 语义），如：`图片|png|转换`、`视频|video|流媒体`、`Excel|解析|csv`。

2. **有则直接执行**  
   若返回结果中有匹配的**已注册函数**（【已注册函数】列表），根据类型选用工具直接执行：  
   - **form**：用 **run_form_submit(full_code_path, body)**，body 按该函数返回的 **request** 结构填写（用户上传的文件放入对应字段，如 input_files）。  
   - **table**：用 run_table_search / run_table_create / run_table_update（见 execute 文档）。  
   - **chart**：用 run_chart_query。  
   **无需询问用户**，直接执行并给出结果即可。

3. **无则先问再建**  
   若 **search_tools** 未匹配到可用函数（或只有内置工具说明、没有可直接调用的已注册函数）：  
   - **禁止直接创建**。  
   - 须先向用户说明：「当前没有现成能力，需要创建新目录并走创建项目流程（先出 PRD，您确认后再写代码）。」  
   - **明确询问**：「是否同意我创建新目录并生成该能力？」（或类似表述。）  
   - **只有用户明确同意后**，才可：创建目录（如需）→ **read_doc("/builtin/doc/workspace/create-project")** 与 SDK → 出 PRD → 用户确认 PRD → 再 write_go_file / add_functions、build_workspace → 用 run_form_submit 等调用新能力。

---

## 三、search_tools 用法

| 参数 | 必填 | 说明 |
|------|------|------|
| **keyword** | ✓ | 搜索关键词。支持多关键词用竖线 \| 分隔（OR），如 `视频\|video\|流媒体`。 |
| template_type | 否 | 限定类型：form / table / chart，空表示全部。 |
| limit | 否 | 最多返回条数，默认 20。 |

**返回**：  
- 【内置工具】：匹配到的内置工具，每条为「名称：描述」。  
- 【已注册函数】：每条包含 **名称**、**full_code_path**、**description**、**type**，以及 **request**、**response** 的 JSON（便于构造 run_form_submit 的 body）。  

根据返回的 full_code_path 和 request 结构，直接调用 run_form_submit 等即可。

### 3.1 返回示例（search_tools 实际返回格式）

例如调用 `search_tools("视频|转换")` 后，可能返回如下（节选一条已注册函数作示例）：

```
【内置工具】
- write_doc：在指定目录下创建或更新一篇文档。必填：name（显示名称）、code（英文标识）、content（正文）。可选：directory（父目录，不传则当前工作目录）、format（默认 markdown）。
- search_tools：按关键词搜索「可用能力」：包括内置工具（读文件、写代码、执行表单等）和平台内已注册的表单/表格/图表函数。用于在不确定是否有现成能力时先搜索，再决定直接调用或创建。支持多关键词：用竖线 | 分隔，如 视频|video|流媒体 表示命中任一关键词即可。

【已注册函数】
1. 视频格式转换
   full_code_path: /luobei/demos/form/videos/convert.form
   description: 支持将视频转换为MP4、WebM、AVI、MKV等多种格式，支持批量处理。使用 FFmpeg，应用场景：视频格式统一、兼容性转换等。
   type: form（调用时 form 用 run_form_submit，table 用 run_table_search/run_table_create/run_table_update，chart 用 run_chart_query）
   request: [
     {
       "code": "input_files",
       "data": { "type": "struct" },
       "field_name": "InputFiles",
       "name": "上传视频文件",
       "validation": "required",
       "widget": {
         "config": { "accept": "video/*", "max_count": 10, "max_size": "500MB" },
         "type": "files"
       }
     },
     {
       "code": "output_format",
       "data": { "type": "string" },
       "field_name": "OutputFormat",
       "name": "目标格式",
       "validation": "required,oneof=mp4 webm avi mkv",
       "widget": { "config": { "default": "mp4", "options": ["mp4", "webm", "avi", "mkv"] }, "type": "select" }
     }
   ]
   response: [
     { "code": "output_file", "data": { "type": "struct" }, "field_name": "OutputFile", "name": "转换后的视频", "widget": { "config": { "max_count": 5 }, "type": "files" } },
     { "code": "convert_info", "data": { "type": "string" }, "field_name": "ConvertInfo", "name": "转换信息", "widget": { "config": {}, "type": "text_area" } }
   ]
```

**如何使用该返回**：  
- 从【已注册函数】中取 **full_code_path**（如 `/luobei/demos/form/videos/convert.form`）作为 run_form_submit 的第一个参数。  
- 按 **request** 里各字段的 `code` 构造 body：注意 `widget.type === "files"` 的字段（如 input_files）必须传**对象** `{ "files": [...], "widget_type": "files", "data_type": "struct" }`，不能传数组，详见第六节。

---

## 四、禁止与注意

- **禁止**在未调用 search_tools 前就假定「没有」并直接去创建。必须先 search_tools，再根据结果决定是「直接执行」还是「问用户是否同意创建」。
- **禁止**在用户未同意创建时就开始 create_directory、write_go_file 或出 PRD。必须先说明、询问、得到用户明确同意后再执行创建流程。
- **注意**：回复用户时用**大白话**、不带技术词（不说 full_code_path、run_form_submit、search_tools 等），站在不懂技术的用户视角描述「能干啥、结果如何」。

---

## 五、与其它任务类型的关系

- 若用户先提的是杂活（如「帮我把这张图转成 PNG」），你按本文档执行：search_tools → 有则执行 / 无则问再建。  
- 若用户后来又说「那再做一个完整的图片处理系统」，则转为**创建项目**，按 read_doc("/builtin/doc/workspace/create-project") 与 SDK 执行。  
- 杂活只做「单次、零散」能力；要做成「系统/项目」时，走创建项目流程。

---

## 六、示例：先搜索再调用（含 files 参数说明）

以**视频格式转换**为例，演示「search_tools → 找到已注册函数 → run_form_submit 调用」的完整流程，并**重点说明 files 组件的传参方式**（智能体常在此处出错）。

### 6.1 流程示例

1. **用户说**：「帮我把这个视频转成 MP4。」
2. **你先调用**：`search_tools("视频|video|格式|转换")`。
3. **返回中有已注册函数**，例如：
   - **视频格式转换**，full_code_path: `/luobei/demos/form/videos/convert.form`
   - type: form，request 中有 `input_files`（上传视频）、`output_format`（目标格式，如 mp4）。
4. **用户已在工作台上传了文件**：这些文件会出现在当前会话的「用户消息附件」中，你调用 run_form_submit 时要把它们放进 body 里对应字段。
5. **调用**：`run_form_submit("/luobei/demos/form/videos/convert.form", body)`，其中 body 按下方 **files 正确结构** 填写。

### 6.2 files 组件参数与传递（必读，易错）

**request 里凡是 `widget.type === "files"` 的字段（如 input_files），对应的 Go 类型是 `*types.Files`，不是文件数组。** 传参时必须按 **对象 + 内层 files 数组** 的结构来写。

**正确结构**（input_files 是**对象**，内含 `files` 数组，以及建议带上 `widget_type`、`data_type`）：

```json
{
  "input_files": {
    "files": [
      {
        "name": "xxx",
        "source_name": "用户选的原始文件名.mp4",
        "storage": "minio",
        "hash": "...",
        "size": 12345,
        "upload_ts": 1234567890,
        "is_uploaded": true,
        "url": "https://...",
        "server_url": "http://..."
      }
    ],
    "widget_type": "files",
    "data_type": "struct"
  },
  "output_format": "mp4"
}
```

**错误结构**（智能体常错成「直接传数组」）：

```json
{
  "input_files": [
    { "name": "xxx", "url": "..." }
  ],
  "output_format": "mp4"
}
```

**为何会错？** 后端对应的是 `types.Files`：

- SDK 中定义：`Files struct { Files []*File ... }`，JSON 序列化后是 `{ "files": [...], "widget_type": "...", "data_type": "..." }`。
- 若传成 `input_files: [...]`，会得到错误：`json: cannot unmarshal array into Go struct field xxx.input_files of type types.Files`，即期望的是 **对象**，不是数组。

**经验**：不确定参数格式时，先看该函数的 request 里该字段的 `data.type` 与 `widget.type`；凡是 **widget.type 为 files** 的，一律传 **对象，且对象内必有 `files` 数组**；工作台上传的文件列表可直接放入该 `files` 数组，并建议带 `widget_type: "files"`、`data_type: "struct"`。

---

（以上为杂活/通用任务规范。执行前若本对话中尚未读过本文档，须先 read_doc 再按流程执行。）
