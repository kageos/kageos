# 杂活 / 通用任务

当用户提的是**零散、单次、跨能力**的需求（不限定在当前项目）时，按本文档执行。典型如：图片转格式、视频处理、解析 Excel、文件压缩等。

核心原则：**先搜后用，没有则问再建**。

---

## 执行流程

### 1. 先 search_tools

根据用户意图提取关键词，调用 `search_tools(keyword)`。keyword 支持多关键词用 `|` 分隔（OR 语义），如 `图片|png|转换`。

### 2. 有则直接执行

若返回中有**已注册函数**，根据类型选用工具直接执行（form 用 `run_form_submit`、table 用 `run_table_search` 等），无需询问用户。

### 3. 无则先问再建

若未匹配到可用函数：
- 禁止直接创建。
- 向用户说明「当前没有现成能力，需要走创建项目流程」。
- 用户同意后，转入创建项目流程：read_doc 创建项目文档 + SDK → 出 PRD → 确认 → write_go_file → build_workspace → 执行。

---

## files 组件传参（易错）

request 里凡是 `widget.type === "files"` 的字段，传参时须为**对象**（不是数组），内含 `files` 数组：

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

返回包含：
- **【内置工具】**：匹配到的内置工具，每条为「名称：描述」。
- **【已注册函数】**：包含 full_code_path、description、type、request（JSON）、response（JSON）。

从已注册函数中取 full_code_path 作为 `run_form_submit` 的第一个参数，按 request 各字段的 code 构造 body。
