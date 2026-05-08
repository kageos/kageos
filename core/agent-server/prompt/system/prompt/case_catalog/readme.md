# 案例目录

本目录收录可供参考的真实案例，按函数组合类型分目录组织，目录 code 使用英文，标题用于树上的中文展示名。

开干前根据需求选择 1 到多个案例读取，除非需求非常简单且 SDK 主文档已经足够。案例是写法参考，不是业务字段模板。

案例目录的文件分工：

- `prd.json`：结构化 PRD 标准样例，优先参考；调用 `write_prd` 时按其中 `project/models/functions/acceptance_cases/confirmation` 组织参数。
- `prd.md`：旧版 PRD 和 Go 代码实现参考，只用于理解业务链路、SDK 写法和坑点，不要模仿其中的 Markdown 表格 PRD 格式。
- Go 文件：最终实现参考，确认 PRD 之后的 `app.create` 阶段再重点参考。

| 需求场景 | 读取路径 |
| --- | --- |
| 单表 CRUD、列表筛选、新增编辑删除 | `/system/prompt/case_catalog/table/ticket` |
| 多表关联、预约、资源占用、明细展示 | `/system/prompt/case_catalog/tables/meeting` |
| 多表管理、人员/职位/候选人管理后台 | `/system/prompt/case_catalog/tables/hr` |
| 问卷、投票、表单提交后进入列表统计 | `/system/prompt/case_catalog/formandtable/vote` |
| Form + Table + Chart、库存、经营统计、图表组合 | `/system/prompt/case_catalog/form_table_chart/cashier` |
| Excel/CSV 上传、解析、转换、批量处理 | `/system/prompt/case_catalog/form/excelorcsv` |
| PDF 解析、生成、提取内容 | `/system/prompt/case_catalog/form/pdf` |
| 图片处理、OCR、图片生成结果 | `/system/prompt/case_catalog/form/images` |
| 视频处理、转码、截图、提取信息 | `/system/prompt/case_catalog/form/videos` |
| 文本/NLP 处理、摘要、分类、抽取 | `/system/prompt/case_catalog/form/nlp` |
| Python 子进程处理并返回文件 | `/system/prompt/case_catalog/form/python_output` |
