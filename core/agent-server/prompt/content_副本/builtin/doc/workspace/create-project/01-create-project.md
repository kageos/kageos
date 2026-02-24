# 创建项目

当用户要**生成应用、生成系统、创建 XXX 管理**等需要写代码并落盘时，按本文档执行。**read_doc 支持一次读取多个**：可 `read_doc("/builtin/doc/sdk/agent-app-sdk-readme", "/builtin/doc/workspace/create-project")` 一次拉取 SDK 与本文档，先建立能力边界（组件类型、options_colors、search、列表写法等）再按本文档流程（PRD → 确认 → SOP）执行。出 PRD 前须已读过 SDK；禁止在不知道 SDK 用法的前提下拍脑袋出 PRD。

---

## 一、不准直接开写代码

必须先**基于 SDK 能力**输出一份**精简的 PRD**（产品需求文档），等用户确认后再写代码。

---

## 二、PRD 格式（必须包含两个 Markdown 表格）

- **表单字段（新增/编辑）**：四列「字段 | 类型 | 必填 | 说明」。**必填列用 ✓（必填）和 ✗（非必填）**，一眼可辨。类型必须对应 SDK 支持的组件，如：文本输入、多行文本、下拉选择、用户选择、多用户选择、时间选择、数字输入、滑块、多选下拉、文件上传。说明里写取值，如优先级写「高/中/低」，状态写「待处理/进行中/已完成/已关闭」。select、multiselect **须配 options_colors**，与 options 一一对应，前端用颜色区分选项。
- **列表模式**：列表/表格会展示的列。**须包含默认字段**：ID、创建时间、更新时间、创建人（创建人展示格式为 `code(显示名)`，如 `beiluo(北洛)`）；**须与表单字段对应**（表单里有的关键业务字段在列表里都要能体现）；**须写完整表格**：表头行、分隔行、至少一行示例数据，这样前端才能渲染成表格。
- **禁止把表格放在代码块内**：表单字段表、列表模式表必须用**纯 Markdown 表格**直接写出，**不要**用 \`\`\` 代码块包裹；否则前端会按源代码显示、无法渲染成表格。
- **是否新建目录**：明确写「不新建目录，放在当前目录」或「会新建目录」，若新建则**必须列出会创建哪些目录**（目录名称 + code，如：任务管理 task）。
- 其他：要做什么（一两句）、放在当前目录下的大致位置或新建目录下的文件名；若有多种实现方式选一种并写清。PRD 要**精简**：几段或两张表格即可，不要长文档、不要废话。

**技术方案**：必须基于 **agent-app SDK**（Go + TableTemplate 等），**禁止**用 HTML/CSS/JS、localStorage、纯前端、单页面应用等方案。

---

## 二点五、PRD 易错点（避免自相矛盾与缺业务规则）

以下约定能减少 PRD 自相矛盾、实现时歧义，**出 PRD 时请自觉遵守**：

0. **Table 与 Form 的存储边界（禁止在 Table 里嵌套 table/form）**
   - **AutoCrudTable 的 model 结构体**里，凡是有 **gorm 列**（会被 GORM 写入数据库）的字段，**只能是**以下可落库类型：**基础类型**（int、string、bool、int64、float64 等）、**files.Files**（实现 JSON 序列化，可 `gorm:"type:json"` 存一列）、**gorm.DeletedAt**（软删除，GORM 特例，占一列）。除此以外，**其他 struct、slice（如 type:table / type:form）不能作为一列写入数据库**——GORM 无法把这类类型写进一张表的某一列。若要在 model 里出现这类 struct/slice，只能是：**外键关联**（如 `Room *MeetingRoom` 配 `gorm:"foreignKey:RoomID"`，实际存的是 RoomID，不占一列）或 **gorm:"-"**（不落库，仅展示/表单用，如计算字段、link、投票选项等）。
   - 因此 **禁止**在 Table 的「表单字段（新增/编辑）」里出现**嵌套的表格（type:table）**或**嵌套的表单（type:form）**作为要落库的列。例如：采购订单**表**的新增/编辑里**不能**写「采购明细 表格 ✓ type:table，至少1行」——否则这一行记录无法落库，实现时会失败。
   - **主从表/明细表的正确做法**：
     - **主表**、**明细表**各是**独立的 Table**（各一张表，每行一条可落库记录）。例如：采购订单表（订单号、供应商、状态、总金额等）、采购明细表（订单ID、商品ID、数量、单价等），每张表的每一行字段都是可落库类型。
     - 若需要「一个界面同时填主表 + 多行明细」：必须用 **FormTemplate** 单独做一个 **POST**（如「创建采购订单」），**请求体**里用 type:table 放明细行（商品、数量、单价）；Handler 内先写主表一条，再写明细表多条。**Table 只做存储和展示**（列表、单行 CRUD），**Form 做复杂录入**（主表+多行明细的一次性提交）。
   - **最佳实践**：参考收银台 `read_doc("/builtin/doc/case_catalog/form_table_chart/cashier")`——商品表/会员表/支付记录表是 **Table**（存储与展示）；「收银台」是 **Form**（请求体里 type:table 商品清单 + 会员选择），提交后 Handler 写支付记录、扣库存等。**不要在 Table 的新增/编辑里嵌套 table**。

1. **自动计算 / 自动生成的字段**
   - 若某字段是**自动计算**（如当前库存、总金额、库存状态），则表单里**不应**标为「必填」且让用户输入——应标为**只读**或**不展示**，说明「由后端根据 xxx 计算」。
   - 若某字段是**自动生成**（如采购单号、销售单号），则表单里**不应**标为「文本输入、必填」——应标为**不展示**或**只读**，说明「新增时由后端按规则生成（如 PO+日期+序号）」。
   - **禁止**同一字段既写「自动计算/自动生成」又写「必填、用户输入」，否则实现时无法同时满足。

2. **业务规则 / 状态与数据联动**
   - 有**状态流转**且会**改其它数据**时（如「已入库」要加库存、「已出库」要减库存），PRD 里**必须**单独写一小节「业务规则」或「状态与数据」：写清**在什么状态下、对哪张表/哪个字段、做什么运算**（如：采购订单状态变为「已入库」时，按采购明细将各商品当前库存增加对应数量；销售订单状态变为「已出库/已完成」时，按销售明细扣减各商品当前库存）。
   - 有**单号、编码**由后端生成时，在业务规则或字段说明里写清「新增时由后端生成，用户不填」。

3. **主从表 / 明细表**
   - 有**明细表**时，**结构上**必须先遵守上条「Table 与 Form 的存储边界」：主表 Table、明细表 Table 各为独立可落库表；若需「一次填主表+多行明细」的界面，用 **Form**（POST）请求体 type:table 做明细录入，Handler 写主表+明细表，**禁止**在 Table 的新增/编辑里嵌套 type:table。
   - PRD 里还须写清：选择商品后**是否带出单价**（如带出采购价/销售价，可改）、**总金额**如何计算（如 Σ 数量×单价，只读）、**主表状态**与库存/从表的联动（见上条业务规则）。

4. **只读视图 / 汇总列表**
   - 若某列表是**只读、自动计算**（如库存汇总、统计报表），须写清**数据来源**：来自哪张表、哪些字段汇总、计算公式或规则；若有「期初」「入库合计」「出库合计」等，须说明各自从哪来、如何算。

遵守以上约定，PRD 与实现会更一致，用户确认后也少返工。

---

## 二点六、最佳实践：Form + Table 结合（以投票系统为例）

**原则**：复杂表单提交**不要**全部塞进 Table 的 template 里做——Table 的新增/编辑适合「单行、字段可落库」的简单 CRUD；**非常复杂的提交**（主表 + 多行明细、多步骤、多联动）Table 的 template 实现不了或很难维护。正确做法是：**用 Form 的 template 做复杂提交，Form 的 Handler 里把数据写到 Table 对应的表**；Table 只做存储与展示（列表、简单单行增删改查）。

**以投票系统贯穿**：

| 能力 | 用谁做 | 说明 |
|------|--------|------|
| 主题列表 / 选项列表 / 记录列表 | **TableTemplate**（GET） | 列表展示、简单单行 CRUD；数据由 Form 或 Table 回调写入 |
| 提交投票（选主题+选选项+校验） | **FormTemplate**（POST `vote_submit.form`） | 复杂表单：主题 select、选项 multiselect + depend_on，Handler 里写 vote_record、更新选项得票与主题总票数 |
| 查看结果（选主题→返回得票统计） | **FormTemplate**（POST `vote_result.form`） | 请求选主题，响应里 table 展示各选项得票；不落库，只读 |
| 创建投票主题（主题字段 + 多行选项） | **推荐 FormTemplate**（POST） | 请求体 = 主题字段 + 投票选项 type:table；Handler 里先 insert 主题，再 insert 选项多条。若放在 Table 的 OnTableAddRow 里也能跑，但更复杂的主从表提交建议一律用 Form，Table 只做列表与简单编辑 |

**一句话**：Form 做复杂录入、提交数据到 Table 对应的表（Handler 里 db.Create/Update）；Table 做存储与展示，不做「一整块复杂表单」的新增/编辑。参考案例：`read_doc("/builtin/doc/case_catalog/formandtable/vote")`（提交投票、查看结果）、`read_doc("/builtin/doc/case_catalog/form_table_chart/cashier")`（收银台 Form 写支付记录）。

---

## 三、示例（任务管理）

- **表单字段（新增/编辑）**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| 任务标题 | 文本输入 | ✓ | 任务名称 |
| 任务描述 | 多行文本 | ✓ | 详细描述 |
| 优先级 | 下拉选择 | ✓ | 高/中/低 |
| 状态 | 下拉选择 | ✓ | 待处理/进行中/已完成/已关闭 |
| 负责人 | 用户选择 | ✓ | 任务负责人 |
| 抄送人 | 多用户选择 | ✗ | 需要知悉的人员 |
| 截止时间 | 时间选择 | ✓ | 任务截止时间 |
| 预计工时 | 数字输入 | ✓ | 预计需要的小时数 |
| 完成进度 | 滑块 | ✗ | 0-100% |
| 标签 | 多选下拉 | ✗ | 紧急/重要/普通/低优先级 |
| 附件 | 文件上传 | ✗ | 相关文件 |
| 备注 | 多行文本 | ✗ | 额外说明 |

- **列表模式**（须含系统字段 ID、创建时间、更新时间、创建人；再列业务字段如标签、备注、附件等）：

| ID | 创建时间 | 更新时间 | 创建人 | 任务标题 | 负责人 | 优先级 | 预计工时 | 完成进度 | 截止时间 | 状态 | 标签 | 附件 | 备注 |
|----|----------|----------|--------|----------|--------|--------|----------|----------|----------|------|------|------|------|
| 2 | 2025-01-20 10:00 | 2025-01-20 15:30 | beiluo(北洛) | 完成需求评审 | zhangsan(张三) | 高 | 8 | 50% | 2025-01-25 | 进行中 | 紧急,重要 | 1 个 | 需同步产品 |
| 1 | 2025-01-19 14:30 | 2025-01-19 14:30 | lisi(李四) | 修复登录 Bug | lisi(李四) | 中 | 2 | 0% | 2025-01-22 | 待处理 | 普通 | — | — |

- **是否新建目录**：不新建，放在当前目录下。（若会新建则写「会新建目录：任务管理 task」等，列出名称与 code。）

---

## 四、等用户明确确认

PRD 末尾问一句：「请确认以上是否 OK，确认后我再按此生成代码。」**收到用户明确确认**（如「可以」「按这个来」「确认」）后，再进入生成 SOP 动手生成。

PRD 经用户确认后，即表示**符合用户预期**。实现时必须**严格按照 PRD** 来写，**不要画蛇添足**——不要自作主张加 PRD 里没有的字段、选项或功能。**只做 PRD 里写出的范围**：PRD 若只包含「表单字段表 + 列表模式」，则**只实现一张表、一个业务 .go、一个 GET 路由**（如 task）。**禁止**添加 PRD 里**没写**的模块，例如：仪表盘、统计页、任务统计(task_stats)、我的任务(my_tasks)、dashboard.go、stats.go 等——除非 PRD 里**明确写了**「需要统计页」「需要仪表盘」「需要我的任务」等。否则一律只写一个 .go、只注册一个 GET，不额外加路由、不加文件。

若用户一开始就给了非常细的需求（已相当于 PRD）且无需补充，你可先 read_doc 拉取 SDK（若未拉取过），再归纳成两三句「将按以下方案生成：……」（方案须符合 SDK 能力），再问「确认后我直接生成？」用户确认后再执行 SOP。

---

## 五、生成代码 SOP（确认后执行）

0. **写代码前**：① **SDK**：PRD 阶段若已拉取过 read_doc("/builtin/doc/sdk/agent-app-sdk-readme")，直接按 SDK 规范写即可（**路由名必须带类型后缀**：.table / .form / .chart，看到后缀即知函数类型）；若未拉取过则先 read_doc 再写。② **最佳实践案例**：**动手写代码前建议** read_doc 与当前项目类型匹配的最佳实践案例（路径即本文档末尾「参考案例（最佳实践）」表格中的 read_doc 路径），对照目录结构、Handler、link/search/options_colors 再落盘，避免写错或漏写。出 PRD 前案例为**可选**（需求与某案例很像时可先读以借鉴 PRD 结构）。禁止用 HTML/CSS/JS、localStorage、纯前端等方案。
0.5. **先思考放哪里**：判断这个功能/项目适合放在当前目录还是需要新建目录。**若是当前项目的新增或扩展**（如对现有模块的补全、小功能增强），放在**当前目录**下新增文件即可。**若是新增的、独立且完整的功能**（如当前在 /odv 运营中心，用户要「任务管理系统」），应**先 create_directory 新建子目录**再在子目录下写代码。例如：当前在 /odv，用户说「需要一个任务管理系统」→ 应新建目录（如 task）；若用户说「给运营中心加一个数据导出按钮」→ 是当前目录的扩展，可在当前目录下新增文件。
1. **（可选）** 若需确认目标目录是否存在，可调 read_dir；系统消息已给当前目录结构时可跳过。
2. **write_go_file**：传 file_name（如 xxx.go）、content、可选 directory。**directory 填目标目录的完整路径**（full_code_path），不传则当前工作目录；写子目录时填该子目录的完整路径（如 `/odv/task`），不能只填子目录 code。系统消息里会给出当前目录的 **Go package（目录代码）**，.go 文件内必须写 `package <目标目录的 code>`，否则编译失败。单文件直接写（默认会编译）；多文件时每次 write_go_file 传 build_workspace=false，全部写完后调用一次 **build_workspace**。**write_doc** 仅当用户明确要求写文档时再调用；不要自作主张帮用户生成文档。目标目录需已存在则先 create_directory。
3. 生成完成后简短总结：生成了哪些文件、放在哪、实现了什么。

**重要**：生成代码后**必须**调用 write_go_file 落盘，不要只输出代码不调用工具。**用户说「开干」「可以」「按这个来」等确认后**：只做与落盘相关的事（read_doc 若需要 → write_go_file / write_doc），不要调用与写代码、落盘无关的工具；不要重复调用任何对落盘无帮助的工具。

---

## 六、禁止与注意

- **禁止**未获用户确认就开写应用/系统代码。
- **禁止**在实现时偏离或超出已确认的 PRD；严格按 PRD 的表单字段和列表模式实现。
- **禁止**添加 PRD 外的模块或文件；只实现 PRD 中的「一张表、一个 .go、一个 GET 路由」。
- **禁止**用 HTML/CSS/JS、纯前端等方案；必须按 agent-app SDK，生成前先 read_doc SDK。
- **禁止**只生成代码不调用 write_go_file。
- **禁止**在 create_directory 之后再用 write_go_file 创建 init.go 或 init_.go；该目录下 **init_.go 已由系统自动生成**（packageContext 由脚手架生成），只需写业务 .go 并用 packageContext.GET/POST(...) 注册路由。**注册时路由名必须带类型后缀**：Table 用 `.table`、Form 用 `.form`、Chart 用 `.chart`；这样从路径即可知函数类型（看到后缀即知类型）。
- **禁止**自作主张帮用户生成文档（write_doc）；仅当用户明确要求时再 write_doc。

**参考示例（案例）何时读**：**动手写代码前建议必读**与项目类型匹配的案例（见上条）；出 PRD 前为可选（需求与某案例类似时可先读以借鉴 PRD 结构）。需要时 read_doc 对应路径：单表 `/builtin/doc/case_catalog/table/ticket`、多表 `/builtin/doc/case_catalog/tables/hr` 或 meeting、单 Form（excelorcsv/images/pdf 等）、Table+Form `/builtin/doc/case_catalog/formandtable/vote`、Table+Form+Chart `/builtin/doc/case_catalog/form_table_chart/cashier`。

---

## 参考案例（最佳实践）

**生成代码前可参考下方已有最佳实践案例**，按类型（单 Table / 单 Form / 多 Table / Table+Form / Table+Form+Chart）选读；需要时 read_doc 对应路径即可。

<!-- BEGIN CASE CATALOG -->
| 类型 | 案例名 | read_doc 路径 | 说明 |
|------|--------|----------------|------|
| 1. 单 Table | 工单管理（单 Table） | `/builtin/doc/case_catalog/table/ticket` | 本案例有一个模块：**工单管理**（GET `ticket_list.table`），单表 CRUD。 单表 CRUD、input/select/switch/slider/rate/radio/number、search 筛选、AutoCrudTable。 |
| 2. 单 Form | Excel/CSV 工具（单 Form） | `/builtin/doc/case_catalog/form/excelorcsv` | 本案例有多个模块，分别是**Excel 转 CSV**（POST `office_excel_to_csv.form` 等）、**Excel 转 JSON**（POST `office_excel_to_json.form`）、**填列**（POST `office_excel_fill_column.form`）、**CSV 转 Excel**（POST `office_csv_to_excel.form` 等），均为 files 上传 + 可选参数 → 响应 text_area 或 files。 files 上传、多 POST 同目录、excelize。 |
| 2. 单 Form | 图片工具（单 Form） | `/builtin/doc/case_catalog/form/images` | 本案例有三个模块，分别是**格式转换**（POST `convert.form`）、**尺寸调整**（POST `resize.form`）、**颜色提取**（POST `colors.form`），均为 files 上传 + 参数 → 响应 files 或 text_area。 files 上传、图片处理、多 POST 同目录。 |
| 2. 单 Form | NLP 工具（单 Form） | `/builtin/doc/case_catalog/form/nlp` | 本案例有一个模块：**分词/词频**（POST `jieba_segment.form`）此功能内部调用了python的jieba库，请求为待分词文本 + 分词模式/关键词数量/移除停用词等，响应为分词结果、关键词列表、词频统计（table）。 text_area/select/number/switch、响应里 table。如果有调用python需求的也可参考这个。 |
| 2. 单 Form | PDF 工具（单 Form） | `/builtin/doc/case_catalog/form/pdf` | 本案例有三个模块：**提取文本**（POST `extract_text.form`，用 **pdftotext**）、**合并 PDF**（POST `merge.form`，用 **Ghostscript**）、**转图片**（POST `to_images.form`，用 **pdftoppm**）。均为 files 上传 + 可选参数 → 响应 text_area 或 files。 |
| 2. 单 Form | 视频工具（单 Form） | `/builtin/doc/case_catalog/form/videos` | 本案例有一个模块：**视频转换**（POST `convert.form`），请求为上传视频 + 目标格式，FFmpeg 转换后返回文件。 files 上传、GetFS、exec、响应 files。 |
| 3. 多 Table | 招聘投递系统（多 Table） | `/builtin/doc/case_catalog/tables/hr` | 本案例有两个模块，分别是**职位管理**（GET `hr_job_list.table`）、**简历/投递管理**（GET `hr_resume_list.table`），主从两表。 主从表、两 .go 两 GET、link、select 关联另一表、files。 |
| 3. 多 Table | 会议室预约（多 Table） | `/builtin/doc/case_catalog/tables/meeting` | 本案例有两个模块，分别是**会议室管理**（GET `meeting_room_list.table`）、**预约管理**（GET `meeting_room_booking_list.table`），主从两表。 主从两表、两 .go 两 GET、OnSelectFuzzy、link、时间状态计算、列表筛外表字段。 |
| 4. Table + Form | 投票系统（Table + Form） | `/builtin/doc/case_catalog/formandtable/vote` | 本案例有五个模块，分别是**投票主题管理**（GET `vote_topic_list.table`）、**投票选项管理**（GET `vote_option_list.table`）、**投票记录查询**（GET `vote_record_list.table`）、**提交投票**（POST `vote_submit.form`）、**查看结果**（POST `vote_result.form`）。 主从多表、multiselect+depend_on、OnSelectFuzzy、link、时间状态、POST 提交与得票率。 |
| 5. Table + Form + Chart | 收银台（Table + Form + Chart） | `/builtin/doc/case_catalog/form_table_chart/cashier` | 本案例有五个模块，分别是**商品管理**（GET `cashier_product_list.table`）、**会员管理**（GET `cashier_member_list.table`）、**支付记录列表**（GET `cashier_payment_record_list.table`）、**收银台**（POST `cashier_desk.form`，请求里 table 子项 + 会员 select）、**统计/图表**（GET 多个 xxx_statistics.chart：销售趋势、分类销售、客单价等折线图）。 FormTemplate 请求中 table 子组件、OnSelectFuzzy、主从表、统计/图表。 |
<!-- END CASE CATALOG -->

**说明**：各案例目录下 `init_.go` 由系统/脚手架自动生成（含 packageContext），禁止用 write_go_file 再创建 init_.go；业务代码写在对应 .go 中，通过 packageContext.GET/POST(...) 注册路由。**生成代码时路由名必须带类型后缀**：`.table` = 表格列表、`.form` = 表单、`.chart` = 图表；看到后缀即可知该函数类型，禁止使用无后缀的路由名。具体目录与文件名见 read_doc 各案例路径返回的 PRD+代码。
