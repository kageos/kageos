# 工作环境信息

**约定：所有占位符均由代码注入完整内容，不截断、不省略。**

### 用户信息
- 当前用户：{{USER}}
- 部门路径（存储/逻辑用）：{{DEPARTMENT_FULL_PATH}}
- 部门（展示用）：{{DEPARTMENT_FULL_NAME_PATH}}

### 时间信息
- 当前时间：{{CURRENT_TIME}}
- 当前日期：{{CURRENT_DATE}}
- 时间戳：{{TIMESTAMP}}

### 当前工作目录
- 目录名称：{{DIR_NAME}}
- 目录代码：{{DIR_CODE}}
- 完整路径：{{FULL_CODE_PATH}}
- 目录类型：{{DIR_TYPE}}
- Go package：`{{DIR_CODE}}`
- 应用中心（Hub）：{{HUB_SECTION}}
{{DIR_DESCRIPTION}}

### 目录结构
{{CHILDREN_SECTION}}
{{FUNCTIONS_SECTION}}

{{FILES_SECTION}}

{{INIT_GO_SECTION}}

---

## 可读的目录

以下可用 `read_doc(directory)` 读取文档，或用 `read_go_file(directory, file_name)` 读取工作区代码文件。

{{DIRECTORY_LIST}}
