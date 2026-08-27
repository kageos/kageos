# 案例目录

本目录收录可供参考的真实案例，按函数组合类型分目录组织，目录 code 使用英文，标题用于树上的中文展示名。

开干前根据需求选择 1 到多个案例读取，除非需求非常简单且 SDK 主文档已经足够。案例是写法参考，不是业务字段模板。

案例目录的文件分工：

- `prd.json`：结构化 PRD 标准样例，优先参考；调用 `write_prd` 时按其中 `project/tables/forms/charts/rules` 组织参数。
- `prd.md`：实现参考和 SDK 写法说明，只用于理解业务链路、代码结构和坑点。
- `prd.md` 的实现章节：从真实 democase 提炼出的关键写法、边界和验收路径；最终仍以 SDK 主文档为准。

| 需求场景 | 读取路径 |
| --- | --- |
| 单表 CRUD、列表筛选、新增编辑删除 | `/system/prompt/case_catalog/table/ticket` |
| 多表关联、预约、资源占用、空闲查询、定时提醒 | `/system/prompt/case_catalog/tables/meeting` |
| 多表管理、人员/职位/候选人管理后台 | `/system/prompt/case_catalog/tables/hr` |
| 问卷、投票、表单提交后进入列表统计 | `/system/prompt/case_catalog/formandtable/vote` |
| Form + Table + Chart、库存、经营统计、图表组合 | `/system/prompt/case_catalog/form_table_chart/cashier` |
| Excel/CSV 上传、解析、转换、批量处理 | `/system/prompt/case_catalog/form/excelorcsv` |
| PDF 解析、生成、提取内容 | `/system/prompt/case_catalog/form/pdf` |
| 图片处理、OCR、图片生成结果 | `/system/prompt/case_catalog/form/images` |
| 视频处理、转码、截图、提取信息 | `/system/prompt/case_catalog/form/videos` |
| 文本/NLP 处理、摘要、分类、抽取 | `/system/prompt/case_catalog/form/nlp` |
| Python 子进程处理并返回文件 | `/system/prompt/case_catalog/form/python_output` |
| docs-first、AgentTask、低风险无人值守、知识缺口转人工 | `/system/prompt/case_catalog/agent/docs_service_desk` |
| 确定性定时任务、站点巡检、故障与恢复通知 | `/system/prompt/case_catalog/automation/site_monitor` |
| 事务、行锁、库存余额、不可变流水、防止并发超领 | `/system/prompt/case_catalog/transaction/consumable_inventory` |
| 公开表单、外部客户预约、内部处理、容量与取消 | `/system/prompt/case_catalog/public/service_booking` |
| 程序维护事实、Agent 生成内部建议、销售跟进准备 | `/system/prompt/case_catalog/agent/hybrid_crm_followup` |

## 容易混淆的案例怎么选

| 需求中的真正难点 | 首选案例 | 与相邻案例的区别 |
| --- | --- | --- |
| 固定时间扫描、确定条件、确定动作 | `automation/site_monitor` | 用 Form schedule；不需要模型参与每轮判断。 |
| 要读自然语言方案后判断能否处理 | `agent/docs_service_desk` | 用 AgentTask；必须有 docs、风险边界、读回验证和人工接管。 |
| 一部分是确定性提醒，一部分需要语义建议 | `agent/hybrid_crm_followup` | schedule 和 AgentTask 分开；事实与建议分开。 |
| 当前余额与每次增减必须一致 | `transaction/consumable_inventory` | 重点是事务、行锁和不可变流水，不是普通 CRUD。 |
| 外部客户提交后转内部履约 | `public/service_booking` | 公开面与内部面分开，并处理容量、幂等、取消释放。 |
| 内部员工预约共享资源 | `tables/meeting` | 重点是登录用户、时间冲突和会前提醒，不是公开入口。 |
| 单次提交后做统计，不占用共享资源 | `formandtable/vote` | 不需要容量锁和复杂状态机。 |
| 多商品结算并形成支付记录 | `form_table_chart/cashier` | 重点是结算和经营统计；库存流水案例更强调单项增减审计。 |

每个新增案例固定包含“适用场景”“不适用场景”“与相邻案例怎么选”“五分钟价值路径”和“不要照搬”。选择案例时先看业务风险和一致性要求，再看页面长得像不像；不要因为字段名称相近就直接复制。
