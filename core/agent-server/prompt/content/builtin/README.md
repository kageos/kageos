# 内置文档（builtin）

本目录下的文档可通过 **read_doc(directory)** 读取，供工作台开发模式下的模型按需拉取。

## 路径约定

- **模型调用**：`read_doc(directory: "/builtin/doc/xxx")`，如 `/builtin/doc/sdk/agent-app-sdk-readme`、`/builtin/doc/case_catalog/table/ticket`。
- **磁盘路径**：均在 `content/builtin/doc/` 下，与暴露路径一致。例如 `read_doc("/builtin/doc/case_catalog/table/ticket")` 会依次尝试：`doc/case_catalog/table/ticket/prd.md` → `doc/case_catalog/table/ticket.md` → `doc/case_catalog/table/ticket/README.md`。

## SDK 文档

| directory | 说明 |
|-----------|------|
| /builtin/doc/sdk/agent-app-sdk-readme | agent-app SDK 使用手册（Table/Form/Chart、widget、search、permission、OnSelectFuzzy 等） |
| /builtin/doc/sdk/agent-app-sdk-crud-readme | agent-app CRUD 相关说明 |

## 案例目录（case_catalog）

| 类型 | directory | 说明 |
|------|-----------|------|
| 单 Table | /builtin/doc/case_catalog/table/ticket | 工单管理 |
| 单 Form | /builtin/doc/case_catalog/form/excelorcsv | Excel/CSV 工具 |
| 单 Form | /builtin/doc/case_catalog/form/images | 图片工具 |
| 单 Form | /builtin/doc/case_catalog/form/nlp | 分词/词频 |
| 单 Form | /builtin/doc/case_catalog/form/pdf | PDF 工具 |
| 单 Form | /builtin/doc/case_catalog/form/videos | 视频工具 |
| 多 Table | /builtin/doc/case_catalog/tables/hr | 招聘投递 |
| 多 Table | /builtin/doc/case_catalog/tables/meeting | 会议室预约 |
| Table+Form | /builtin/doc/case_catalog/formandtable/vote | 投票系统 |
| Table+Form+Chart | /builtin/doc/case_catalog/form_table_chart/cashier | 收银台 |

新增案例：在 `doc/case_catalog/` 下按类型建目录并放 `prd.md`；需同步到可读目录时运行 `go run ./scripts/sync-case-catalog/main.go`。
