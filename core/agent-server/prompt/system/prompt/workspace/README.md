# 工作台文档

本目录用于**工作台开发模式**的「角色 + 路由 + 按需文档」方案：system_prompt 只做角色与文档索引，详细规范放在本目录下各**子目录**中，由模型通过 read_doc 按需读取。

**read_doc 可读目录**：`read_doc(directory: "/system/prompt/workspace/xxx")` 可读取该子目录下所有文档；各子目录内使用英文文件名（如 `01-create-project.md`），展示名从文档标题读取。

---

## 子目录与文档（按任务类型）

| 子目录 | read_doc 路径 | 说明 |
|--------|----------------|------|
| **misc-tasks/** | /system/prompt/workspace/misc-tasks | 杂活/通用任务：图片转格式、视频处理、解析 Excel 等零散需求；先 search_tools → 有则直接执行 → 无则问用户是否同意创建再走创建项目流程。 |
| **create-project/** | /system/prompt/workspace/create-project | 创建项目：PRD 格式（含完整示例：表单字段表、列表模式表、是否新建目录）、确认流程、生成 SOP、禁止项；须先 read_doc SDK。 |
| **modify-project/** | /system/prompt/workspace/modify-project | 修改项目：修改 PRD、改代码（search_replace_file 优先、search_string 完全一致、read_go_file 复制原文、禁止整文件重写、build_workspace、编译失败应对）、写项目文档。 |
| **execute/** | /system/prompt/workspace/execute | 操作项目：查列表、提交表单、查图表、新增/批量导入/更新/删除记录；操作 SOP、易错点、工具用法与传参（full_code_path 须到具体函数、url_query 约定等）均在本目录文档内。 |
| **explain-project/** | /system/prompt/workspace/explain-project | 了解项目：用户问「有什么能力」「怎么用」「有哪些接口」时，根据「当前目录下的可执行函数」与可读目录 summary/prd 作答；仅作答，不写代码、不调执行类工具。 |

SDK 能力与案例：`read_doc("/system/prompt/sdk/agent-app-sdk-readme")`、`/system/prompt/case_catalog/xxx`。

---

## 使用说明

- **按任务类型读文档**：杂活/通用任务 → read_doc("/system/prompt/workspace/misc-tasks")；创建项目 → read_doc("/system/prompt/workspace/create-project")；修改项目 → read_doc("/system/prompt/workspace/modify-project")；操作项目 → read_doc("/system/prompt/workspace/execute")；了解项目 → read_doc("/system/prompt/workspace/explain-project")。read_doc 读目录时会返回该目录下所有 .md（如 01-xxx.md），按文件名顺序展示。
- **新增文档**：在对应子目录下增加 01-xxx.md、02-xxx.md 等；并在本文 README 与 `mode/dev/system_prompt.md` 的路由说明中补充入口。
