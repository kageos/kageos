# 内置文档（builtin）

本目录下的文档可通过 **read_doc(directory)** 读取，供工作台开发模式下的模型按需拉取。

## 路径对应关系

- **directory**（模型调用 read_doc 时传入）：`/builtin/doc/xxx` 形式，与磁盘路径对齐，如 `/builtin/doc/case_catalog/table/ticket`、`/builtin/doc/sdk/agent-app-sdk-readme`。
- **本目录下的文件路径**：与暴露路径一致，均在 `content/builtin/doc/` 下：
  - SDK：`doc/sdk/agent-app-sdk-readme.md`
  - 案例：`doc/case_catalog/table/ticket/prd.md` 或 `doc/case_catalog/xxx/README.md`

例如：`read_doc(directory: "/builtin/doc/case_catalog/table/ticket")` 会依次尝试读取：
1. `content/builtin/doc/case_catalog/table/ticket/prd.md`
2. `content/builtin/doc/case_catalog/table/ticket.md`
3. `content/builtin/doc/case_catalog/table/ticket/README.md`

## 目录结构

| 暴露路径（directory） | 磁盘路径（content/builtin/ 下） | 说明 |
|------------------------|--------------------------------|------|
| /builtin/doc/sdk/agent-app-sdk-readme | doc/sdk/agent-app-sdk-readme.md | SDK 使用手册 |
| /builtin/doc/case_catalog/table/ticket | doc/case_catalog/table/ticket/prd.md | 单 Table：工单 |
| /builtin/doc/case_catalog/form/excelorcsv | doc/case_catalog/form/excelorcsv/prd.md | 单 Form：Excel/CSV |
| /builtin/doc/case_catalog/form_table_chart/cashier | doc/case_catalog/form_table_chart/cashier/prd.md | Table+Form+Chart：收银台 |

新增案例时：在 `文档目录.json` 中增加一条（name、full_code_path、when_to_use），再在 `builtin/doc/case_catalog/` 下按示例项目结构建目录并放 `prd.md`；运行 `go run ./scripts/sync-case-catalog` 可自动从示例项目同步并更新目录。
