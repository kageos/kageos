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

0. **写代码前**：① **SDK**：PRD 阶段若已拉取过 read_doc("/builtin/doc/sdk/agent-app-sdk-readme")，直接按 SDK 规范写即可；若未拉取过则先 read_doc 再写。② **最佳实践案例**：**动手写代码前建议** read_doc 与当前项目类型匹配的最佳实践案例（路径即本文档末尾「参考案例（最佳实践）」表格中的 read_doc 路径），对照目录结构、Handler、link/search/options_colors 再落盘，避免写错或漏写。出 PRD 前案例为**可选**（需求与某案例很像时可先读以借鉴 PRD 结构）。禁止用 HTML/CSS/JS、localStorage、纯前端等方案。
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
- **禁止**在 create_directory 之后再用 write_go_file 创建 init.go 或 init_.go；该目录下 **init_.go 已由系统自动生成**（packageContext 由脚手架生成），只需写业务 .go 并用 packageContext.GET(...) 注册路由。
- **禁止**自作主张帮用户生成文档（write_doc）；仅当用户明确要求时再 write_doc。

**参考示例（案例）何时读**：**动手写代码前建议必读**与项目类型匹配的案例（见上条）；出 PRD 前为可选（需求与某案例类似时可先读以借鉴 PRD 结构）。需要时 read_doc 对应路径：单表 `/builtin/doc/case_catalog/table/ticket`、多表 `/builtin/doc/case_catalog/tables/hr` 或 meeting、单 Form（excelorcsv/images/pdf 等）、Table+Form `/builtin/doc/case_catalog/formandtable/vote`、Table+Form+Chart `/builtin/doc/case_catalog/form_table_chart/cashier`。

---

## 参考案例（最佳实践）

**生成代码前可参考下方已有最佳实践案例**，按类型（单 Table / 单 Form / 多 Table / Table+Form / Table+Form+Chart）选读；需要时 read_doc 对应路径即可。

<!-- BEGIN CASE CATALOG -->
| 类型 | 案例名 | read_doc 路径 | 说明 |
|------|--------|----------------|------|
| 1. 单 Table | 工单管理（单 Table） | `/builtin/doc/case_catalog/table/ticket` | 本案例有一个模块：**工单管理**（GET `ticket_list`），单表 CRUD。 单表 CRUD、input/select/switch/slider/rate/radio/number、search 筛选、AutoCrudTable。 |
| 2. 单 Form | Excel/CSV 工具（单 Form） | `/builtin/doc/case_catalog/form/excelorcsv` | 本案例有四个模块，分别是**Excel 转 CSV**（POST）、**Excel 转 JSON**（POST）、**填列**（POST）、**CSV 转 Excel**（POST），均为 files 上传 + 可选参数 → 响应 text_area 或 files。 files 上传、多 POST 同目录、excelize。 |
| 2. 单 Form | 图片工具（单 Form） | `/builtin/doc/case_catalog/form/images` | 本案例有三个模块，分别是**格式转换**（POST `images_convert`）、**尺寸调整**（POST `images_resize`）、**颜色提取**（POST `images_colors`），均为 files 上传 + 参数 → 响应 files 或 text_area。 files 上传、图片处理、多 POST 同目录。 |
| 2. 单 Form | NLP 工具（单 Form） | `/builtin/doc/case_catalog/form/nlp` | 本案例有一个模块：**分词/词频**（POST `jieba_segment`）此功能内部调用了python的jieba库，请求为待分词文本 + 分词模式/关键词数量/移除停用词等，响应为分词结果、关键词列表、词频统计（table）。 text_area/select/number/switch、响应里 table。如果有调用python需求的也可参考这个。 |
| 2. 单 Form | PDF 工具（单 Form） | `/builtin/doc/case_catalog/form/pdf` | 本案例有三个模块：**提取文本**（POST `extract_text`，用 **pdftotext**）、**合并 PDF**（POST `merge`，用 **Ghostscript**）、**转图片**（POST `to_images`，用 **pdftoppm**）。均为 files 上传 + 可选参数 → 响应 text_area 或 files。 |
| 2. 单 Form | 视频工具（单 Form） | `/builtin/doc/case_catalog/form/videos` | 本案例有一个模块：**视频转换**（POST `video_convert`），请求为上传视频 + 目标格式，FFmpeg 转换后返回文件。 files 上传、GetFS、exec、响应 files。 |
| 3. 多 Table | 招聘投递系统（多 Table） | `/builtin/doc/case_catalog/tables/hr` | 本案例有两个模块，分别是**职位管理**（GET `hr_job_list`）、**简历/投递管理**（GET `hr_resume_list`），主从两表。 主从表、两 .go 两 GET、link、select 关联另一表、files。 |
| 3. 多 Table | 会议室预约（多 Table） | `/builtin/doc/case_catalog/tables/meeting` | 本案例有两个模块，分别是**会议室管理**（GET `meeting_room_list`）、**预约管理**（GET `meeting_room_booking_list`），主从两表。 主从两表、两 .go 两 GET、OnSelectFuzzy、link、时间状态计算、列表筛外表字段。 |
| 4. Table + Form | 投票系统（Table + Form） | `/builtin/doc/case_catalog/formandtable/vote` | 本案例有五个模块，分别是**投票主题管理**（GET `vote_topic_list`）、**投票选项管理**（GET `vote_option_list`）、**投票记录查询**（GET `vote_record_list`）、**提交投票**（POST `vote_submit`）、**查看结果**（POST `vote_result`）。 主从多表、multiselect+depend_on、OnSelectFuzzy、link、时间状态、POST 提交与得票率。 |
| 5. Table + Form + Chart | 收银台（Table + Form + Chart） | `/builtin/doc/case_catalog/form_table_chart/cashier` | 本案例有五个模块，分别是**商品管理**（GET 商品列表）、**会员管理**（GET 会员列表）、**支付记录列表**（GET）、**收银台**（POST `cashier_desk`，请求里 table 子项 + 会员 select）、**统计/图表**（GET 多个 statistics：销售趋势、分类销售、客单价等折线图）。 FormTemplate 请求中 table 子组件、OnSelectFuzzy、主从表、统计/图表。 |
<!-- END CASE CATALOG -->

**说明**：各案例目录下 `init_.go` 由系统/脚手架自动生成（含 packageContext），禁止用 write_go_file 再创建 init_.go；业务代码写在对应 .go 中，通过 packageContext.GET/POST(...) 注册路由。具体目录与文件名见 read_doc 各案例路径返回的 PRD+代码。
