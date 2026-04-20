# 创建项目

当用户要**生成应用、生成系统、创建 XX 管理**等需要写代码并落盘时，按本目录执行。

## 入口规则

- 先读 SDK：`read_doc("/system/prompt/sdk/agent-app-sdk-readme")`
- 再读本目录：`read_doc("/system/prompt/workspace/create-project")`
- 出 PRD 前必须先读 1～2 个匹配案例

## 本文分工

- 前半段讲 PRD 格式
- 中间讲易错点和生成 SOP
- 最后附参考案例

## 最小执行顺序

1. 先判断需求能否用平台现有能力实现
2. 先看匹配案例
3. 按本文的 PRD 格式输出方案
4. 用户确认后，按本文的生成 SOP 写代码

## 强制要求

- 禁止未读案例就直接出 PRD
- 禁止未确认 PRD 就直接写代码
- 路由名必须带类型后缀：`.table` / `.form` / `.chart`

---

## 一、PRD 格式

### 必须包含两个 Markdown 表格

- **表单字段（新增/编辑）**：五列「字段 | 类型 | 必填 | 默认值 | 说明」
- **列表模式**：系统字段 + 业务字段 + 仅列表展示的计算字段

约束：

- 表格必须用纯 Markdown，不能放代码块
- 默认值没有就写 `—`
- 只列出用户需要填写的字段
- 计算字段、后端自动生成字段不要放进表单字段表
- `select` / `multiselect` 须配 `options_colors`
- 最后必须明确写「是否新建目录」

### 表单字段表怎么写

五列固定为：

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|

类型要用用户能看懂的话描述：

- 文本输入
- 多行文本
- 下拉选择
- 用户选择
- 时间选择
- 数字输入
- 滑块
- 多选下拉
- 文件上传

### 列表模式表怎么写

必须包含：

- 系统字段：`ID`、`创建时间`、`更新时间`、`创建人`
- 业务字段：与表单顺序尽量一致
- 仅列表展示的计算字段：放在列表表头里，但不要放进表单字段表

### 示例

#### 表单字段（新增/编辑）

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| 任务标题 | 文本输入 | ✓ | — | 任务名称 |
| 任务描述 | 多行文本 | ✓ | — | 详细描述 |
| 优先级 | 下拉选择 | ✓ | 中 | 高/中/低 |
| 状态 | 下拉选择 | ✓ | 待处理 | 待处理/进行中/已完成/已关闭 |
| 负责人 | 用户选择 | ✓ | — | 任务负责人 |
| 截止时间 | 时间选择 | ✓ | — | 任务截止时间 |
| 预计工时 | 数字输入 | ✓ | — | 小时数 |
| 完成进度 | 滑块 | ✗ | 0% | 0-100% |
| 标签 | 多选下拉 | ✗ | — | 紧急/重要/普通/低优先级 |
| 附件 | 文件上传 | ✗ | — | 相关文件 |

#### 列表模式

| ID | 创建时间 | 更新时间 | 创建人 | 任务标题 | 任务描述 | 优先级 | 状态 | 负责人 | 截止时间 | 预计工时 | 完成进度 | 标签 | 附件 | 剩余天数 |
|----|----------|----------|--------|----------|----------|--------|------|--------|----------|----------|----------|------|------|----------|
| 2 | 2025-01-20 10:00 | 2025-01-20 15:30 | beiluo(北洛) | 完成需求评审 | 需同步产品 | 高 | 进行中 | zhangsan(张三) | 2025-01-25 | 8 | 50% | 紧急,重要 | 1 个 | 5 天 |
| 1 | 2025-01-19 14:30 | 2025-01-19 14:30 | lisi(李四) | 修复登录 Bug | — | 中 | 待处理 | lisi(李四) | 2025-01-22 | 2 | 0% | 普通 | — | 3 天 |

#### 是否新建目录

- 新建，会创建：任务管理系统（`task_manage`）

---

## 二、PRD 易错点

### 1. Table 与 Form 的存储边界

AutoCrudTable 的落库字段只能是基础类型、`string`（`gorm:"type:text"`）、`gorm.DeletedAt`。不要把嵌套 `table` / `form` 直接作为表列。

正确做法：

- 主从两表分别建表
- 如果需要“一次填主表 + 多行明细”，用 `FormTemplate` 提交，Handler 里写多张表

### 2. 自动计算 / 自动生成 / 仅列表展示

- 自动生成字段：表单里不填，说明由后端生成
- 仅列表展示字段：只在列表模式表里写，说明“仅列表展示、后端计算”

### 3. 状态联动

如果状态变化会连带改别的数据，PRD 必须写清楚：

- 什么状态变化
- 改哪张表
- 改哪个字段
- 做什么运算

### 4. Form + Table 的边界

复杂一次性提交不要硬塞进 Table 的增删改回调。复杂提交优先用 Form。

### 5. 规模匹配

默认按最小可用版本设计：

- 小门店、小团队、轻量场景，不要擅自扩成复杂平台
- 只有用户明确要求，才加多门店、多仓库、复杂流程等

### 6. 流水/记录类表默认只读

像支付记录、消费流水、操作日志、审计记录，默认只做查询表。

不要默认提供：

- 新增
- 编辑
- 删除

### 7. 跨表联动必须放事务

涉及余额、库存、数量、状态、流水等强一致数据时，必须事务化处理。

### 8. 数据库兼容默认 SQLite 优先

优先顺序：

1. 先用 GORM
2. 再用 Go 代码补逻辑
3. 最后才按数据库方言分支

典型差异：

- 日期格式化
- 字符串拼接
- upsert
- JSON 函数

### 9. 常见补充说明

这些情况在 PRD 里按需说明：

| 情况 | PRD 怎么写 |
|------|------------|
| 仅列表展示的计算字段 | 列表里写，表单不写 |
| 自动生成字段 | 标明不展示或只读，由后端生成 |
| 新增可填、编辑只读 | 用业务说明写清楚 |
| 关联展示 | 表单写关联选择，列表写展示名称 |
| 默认值 | 在默认值列写清楚 |
| 列表可筛选/可排序 | 明确写支持哪些字段 |
| 软删除 | 写明默认不展示已删除数据 |
| 状态流转 | 写明状态和操作关系 |
| 条件展示 | 直接写成 `validate` 语义，条件值必须与真实提交值一致，规则用逗号分隔 |

---

## 三、确认后生成 SOP

PRD 末尾必须问用户一句：

`请确认以上是否 OK，确认后我再生成代码。`

得到确认后再执行。

### 执行顺序

0. 若本轮还没读过 SDK 或关键案例，先补读
1. 判断放在哪个目录
2. 如需新目录，先 `create_directory`
3. 用 `write_go_file` 落盘
4. 多文件全部写完后，再统一 `build_workspace`
5. 给用户简短总结生成结果

### write_go_file 约束

- `directory` 填目标目录完整路径
- `.go` 文件里的 `package` 必须与目标目录 code 一致
- `write_go_file` 只落盘，不编译
- 不要每写一个文件就编译一次

### 依赖

- 可以直接在代码里引用开源 Go 依赖
- `build_workspace` 时会自动拉依赖

### 禁止项

- 不要创建或修改 `init.go` / `init_.go`
- 不要擅自帮用户写项目文档，除非用户明确要求
- 不要在 PRD 只有一张表时，擅自加 dashboard 或额外模块
- 不要在 PRD 或代码里重造平台已有横切能力
- 消息提醒优先复用 `ctx.SendMessage(...)`

---

## 四、参考案例

出 PRD 前必须先读至少 1 个匹配案例；写代码前如有需要可再回看。

按关键特性匹配：你要用到的技术点在哪个案例里出现过，就去读那个案例。

| 案例 | read_doc 路径 | 关键特性 |
|------|---------------|----------|
| 工单管理（单 Table） | `/system/prompt/case_catalog/table/ticket` | `单表CRUD` `AutoCrudTable` `多种组件(input/select/switch/slider/rate/radio/number)` `search筛选` |
| Excel/CSV 工具（单 Form） | `/system/prompt/case_catalog/form/excelorcsv` | `文件上传(files)` `excelize库` `多POST同目录` `文件转换` |
| 图片工具（单 Form） | `/system/prompt/case_catalog/form/images` | `文件上传(files)` `ImageMagick(convert/identify)` `exec.Command调用可执行程序` `图片处理` `GetTraceOutputDir` `ResponseFiles` |
| NLP 工具（单 Form） | `/system/prompt/case_catalog/form/nlp` | `pythonRuntime` `defer Close` `jieba` `响应中含table组件` `同机Python子进程` |
| PDF 工具（单 Form） | `/system/prompt/case_catalog/form/pdf` | `文件上传(files)` `Poppler(pdftotext/pdftoppm)` `Ghostscript(gs)` `exec.Command调用可执行程序` |
| Python 容器内产物输出（单 Form） | `/system/prompt/case_catalog/form/python_output` | `pythonRuntime` `defer Close` `绝对路径落盘` `output_json` `GetTraceOutputDir` `ResponseFiles` `响应 string` `用户可下载附件` `同机子进程` `非宿主机` |
| 视频工具（单 Form） | `/system/prompt/case_catalog/form/videos` | `文件上传(files)` `FFmpeg(exec.Command)` `音视频处理` `GetTraceOutputDir` `ResponseFiles` `输入文件复制到输出目录` |
| 招聘投递系统（多 Table） | `/system/prompt/case_catalog/tables/hr` | `主从两表` `link跳转` `select关联另一表` `文件上传(files)` |
| 会议室预约（多 Table） | `/system/prompt/case_catalog/tables/meeting` | `主从两表` `OnSelectFuzzy模糊搜索选择` `link跳转` `时间状态自动计算` `列表筛外表字段` |
| 投票系统（Table + Form） | `/system/prompt/case_catalog/formandtable/vote` | `Table+Form混合` `multiselect+depend_on联动` `OnSelectFuzzy` `link按状态切换` `POST提交写多表` |
| 收银台（Table + Form + Chart） | `/system/prompt/case_catalog/form_table_chart/cashier` | `Table+Form+Chart全类型` `Form请求中嵌套table子组件` `OnSelectFuzzy` `统计图表(Chart)` `主从表+库存联动` |

补充提醒：

- 路由名必须带类型后缀：`.table` / `.form` / `.chart`
