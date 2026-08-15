# kageos 最佳实践案例索引

写 kageos 应用时，以案例为主。先看最接近需求的完整案例，再设计，再写代码。

完整案例的最低阅读标准：

1. 先读案例 `readme.md` 或开头说明，确认它适合当前需求。
2. 再读案例 `prd.md`，里面通常包含 PRD、结构体、模板、Handler 和注册代码。
3. 如果案例有 Go 文件或 `prd.json`，按需继续读。不要只看几行片段就开始写。

## 完整平台案例

这些案例已经完整复制到本 skill 的 `references/case_catalog/`。优先读这些真实工作台案例：

| 需求场景 | 推荐案例 |
|---|---|
| 单表 CRUD、列表筛选、新增编辑删除 | `../case_catalog/table/ticket` |
| 简单多表管理、职位/候选人管理后台 | `../case_catalog/tables/hr` |
| 资源预约、主从表、冲突校验、定时提醒 | `../case_catalog/tables/meeting` |
| 表单提交后写入表格、投票/问卷 | `../case_catalog/formandtable/vote` |
| Form + Table + Chart、经营统计 | `../case_catalog/form_table_chart/cashier` |
| Excel/CSV 上传、解析、转换、批量处理 | `../case_catalog/form/excelorcsv` |
| PDF 解析、生成、提取内容 | `../case_catalog/form/pdf` |
| 图片处理、OCR、图片结果 | `../case_catalog/form/images` |
| 视频处理、转码、截图、提取信息 | `../case_catalog/form/videos` |
| 文本/NLP 处理、摘要、分类、抽取 | `../case_catalog/form/nlp` |
| Python 子进程处理并返回文件 | `../case_catalog/form/python_output` |

每个目录通常包含 `readme.md`、`prd.md`、`prd.json`。写代码前至少读一个目录里的完整 `prd.md`。

## 设计补充材料

这些不是默认入口，只有需求已经超过简单 Table/Form/Chart 时再读：

| 需求场景 | 推荐材料 |
|---|---|
| 设计可运营的场景目录，区分 V1 和后续增强 | `../solution-design-principles.md` |
| 已证明一张主表不够，需要多对象、多状态或 AI 后台流程 | `../ai-native-workflow-modeling.md` |
| 内容工厂、审核流、发布包、自动复盘等复杂工作流 | `../workflow-product-quality.md` |

## 随 skill 分发的轻量案例

这些案例可在没有平台源码时使用，也适合快速找模式：

| 需求场景 | 推荐案例 |
|---|---|
| 业务台账、列表查询、自动 CRUD、分页排序 | `table-crud-case.md` |
| 给 Form 声明默认定时任务，做巡检/提醒 | `scheduled-function-case.md` |
| 无人值守 AgentTask 和 runbook 写法 | `agent-session-runbook-case.md` |
| Go 调 Python runtime，做 NLP/文件/图片产物 | `go-python-runtime-case.md` |
| 资源预约、冲突校验、空闲查询、提醒 | `meeting-room-solution-case.md` |
| 证书台账、自动续期、巡检提醒 | `certificate-ops-solution-case.md` |
| 合同管理简单版和多节点高级版边界 | `contract-milestone-workflow-skeleton.md` |
| 基于业务表做标准图表 | `chart-case.md` |

## 选案例规则

| 当前需求 | 先选什么 |
|---|---|
| “做一个系统/管理后台/台账” | 单表案例，先设计 1 个主 `TableTemplate` |
| “上传文件处理一下” | Form 文件处理案例 |
| “提交一次动作并生成结果” | 单 Form 案例 |
| “表单提交后形成记录” | Form + Table 案例 |
| “有一批记录要长期维护” | Table 案例 |
| “看趋势/排行/分布” | Chart 案例；一个 `.chart` 只返回一张图 |
| “每天/每周自动提醒或巡检” | 先看是否能给已有 Form 加 schedule；只有需要模型判断、跨工具查询、生成报告时才看 AgentTask 案例 |
| “想要漂亮页面/自定义页面/仪表盘布局” | 先回到 `../boundaries.md`，说明 kageos 只能 Form/Table/Chart 组合 |

## 反向约束

- 不要从“完整业务系统”开始设计；先找最小案例。
- 不要把一个普通台账拆成申请表、节点表、日志表、统计图。
- 不要为了用上案例而照搬案例里的全部入口；只取当前需求需要的模板组合。
- 不要把案例里的业务字段当成固定字段；字段要按用户场景重新命名。
