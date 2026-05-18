# 开源前治理 TODO

> 状态：执行看板
> 更新时间：2026-05-17
> 负责人窗口：总控文档窗口
> 用法：每个并行会话领取一个事项，完成后更新状态、风险和验证结果。

## 总目标

把项目从“内部能跑、核心链路正在收敛”治理到：

- 外部开发者能理解项目定位。
- 外部开发者能按文档跑起来。
- 维护者能用 CI 判断改动是否可合并。
- 仓库里没有明显敏感信息和内部工作区污染。
- 许可证、商业边界、贡献方式说清楚。

## 状态说明

| 状态 | 含义 |
| --- | --- |
| TODO | 未开始 |
| Doing | 有窗口正在处理 |
| Review | 已提交，等待人工确认 |
| Done | 已合并或确认完成 |
| Blocked | 需要产品/商业/法务/架构决策 |

## P0：开源前必须完成

### 事项 1：文档入口和开源门面

- 状态：Review
- 建议窗口：`codex/open-source-governance`
- 目标：让第一次打开仓库的人知道这是什么、怎么跑、怎么贡献。
- 交付物：
  - 重写根 `README.md` 为开源首页。
  - 新增 `CONTRIBUTING.md`。
  - 新增 `SECURITY.md`。
  - 新增 `CODE_OF_CONDUCT.md`。
  - 根 README 链接到 `docs/README.md`。
- 验收：
  - README 不再只是总蓝图。
  - 5 分钟内能找到本地启动、测试、部署、贡献入口。
  - BSL/source-available 表述准确，不误称 OSI 开源。
- 本轮结果：
  - 已重写根 `README.md`，新增 `CONTRIBUTING.md`、`SECURITY.md`、`CODE_OF_CONDUCT.md`。
  - README 已链接到 `docs/README.md`、本地开发、后端质量、示例、发布流程和部署文档。
  - 保持 BSL 1.1/source-available 表述，不误称 OSI 开源。

### 事项 2：许可证和商业边界

- 状态：Blocked
- 建议窗口：`codex/license-boundary-cleanup`
- 阻塞决策：项目到底用 BSL、Apache-2.0、AGPL，还是双许可证。
- 目标：避免“说开源但 LICENSE 是 BSL”带来的信任风险。
- 交付物：
  - 明确 `LICENSE` 口径。
  - 明确 README 中使用“开源 / 源码可见 / source-available”的措辞。
  - 明确 `enterprise_impl/`、`pkg/license/` 是否公开。
  - 检查 `THIRD_PARTY_LICENSES.md` 是否覆盖当前依赖。
- 验收：
  - 许可证和宣传口径一致。
  - 商业版/企业版边界清楚。
- 本轮结果：
  - 未擅自变更许可证；README 已按当前 `LICENSE` 明确为 BSL 1.1/source-available，并说明当前不是 OSI 开源。
  - `THIRD_PARTY_LICENSES.md` 未在本轮重算，仍需最终许可证策略确认后复核。

### 事项 3：敏感信息和发布包边界

- 状态：Review
- 建议窗口：`codex/public-secret-scan`
- 目标：开源仓库不带真实配置、客户信息、测试工作区、token、私钥。
- 交付物：
  - 清理 git 跟踪的真实 `.env.*`，只保留 `.env.example`。
  - 审查 `license.json`、`pkg/license/example_license.json`。
  - 审查 `namespace/**` 是否应该进入公开仓库。
  - 审查 `local/hermes-agent/**` 是否独立项目或第三方代码。
  - 补充 `.gitignore` 和敏感信息扫描命令。
- 验收：
  - `git ls-files | rg '(^|/)\\.env|secret|private|credential|token'` 没有误跟踪配置。
  - 公开包边界清楚。
- 本轮结果：
  - 已从 git 索引移出 `web/.env.development`、`web/.env.production`、`deploy/prod/aos.yaml`，本地文件保留。
  - 新增 `web/.env.example`，并强化 `.gitignore` 对 `.env*`、生产配置、生成物和本地工作区的忽略。
  - 新增 `scripts/check-sensitive-files.sh`；`license.json` 当前为本地未跟踪文件，`.gitignore` 已覆盖。
  - `namespace/`、`local/` 保持忽略，并在 README/后端文档中明确为非公开包边界。

### 事项 4：CI 和质量门禁

- 状态：Review
- 建议窗口：`codex/ci-quality-gates`
- 目标：PR 合并前有自动质量检查。
- 交付物：
  - 前端 CI：`npm run check:architecture`、`npm run type-check`、`npm run test:unit -- --run`、`npm run build`。
  - 后端 CI：明确官方回归脚本，不直接污染 `namespace/**`。
  - Markdown/链接检查可选，但先不阻塞。
  - README 写清本地验证命令。
- 验收：
  - fork 后能跑 CI。
  - 失败信息可读。
- 本轮结果：
  - 新增 `.github/workflows/ci.yml`，包含治理检查、后端测试、前端四件套。
  - README 写清本地验证命令。
  - 官方后端回归脚本继续使用 `scripts/test-core-go.sh`，避免直接污染 `namespace/**`。

### 事项 5：本地启动和部署体验

- 状态：Review
- 建议窗口：`codex/local-dev-onboarding`
- 目标：新人按文档能跑起最小系统。
- 交付物：
  - 依赖清单：Go、Node、数据库、NATS、MinIO、容器运行时。
  - 开发启动顺序。
  - `.env.example`。
  - smoke test。
  - 常见错误排查。
- 验收：
  - 新人不需要读 10 个历史文档也能启动。
- 本轮结果：
  - 新增 `docs/local-development.md`，覆盖依赖清单、启动顺序、前端 env 示例、smoke test 和常见错误。
  - 根 README、`docs/README.md` 已链接该入口。

## P1：开源质量增强

### 事项 6：前端架构债继续清

- 状态：Review
- 建议窗口：`codex/frontend-type-debt-next`
- 目标：继续降低表现层 `any` 和旧 runtime 命名。
- 已完成基础：
  - 移除历史 frontend runtime 层。
  - presentation API 统一走 context。
  - search/select/multiselect 主链路已类型化。
- 下一批优先：
  - `web/src/architecture/presentation/composables/useWorkspaceDetail.ts`
  - `web/src/architecture/presentation/composables/useFormOperateLogSection.ts`
  - `web/src/architecture/presentation/components/utils/chartRendererOption.ts`
  - `web/src/architecture/presentation/views/TableView.vue`
  - `web/src/architecture/presentation/composables/utils/tableInitializationRuntime.ts`
- 验收：
  - 每批提交后必须跑前端四件套。
- 本轮结果：
  - 已清理本批仍存在文件中的显式 `any`：`useWorkspaceDetail.ts`、`chartRendererOption.ts`、`TableView.vue`、`tableInitializationRuntime.ts`。
  - `web/src/architecture/presentation/composables/useFormOperateLogSection.ts` 当前已不存在；相关能力收敛为 `useOperateLogSection.ts`，本批按路径过期处理。
  - 图表渲染工具补充 tooltip、series config、gauge option 局部类型；表格初始化 query 类型收窄；详情抽屉行数据和用户缓存改为显式类型。
  - 已补充 `useWorkspaceDetail.ts` 的详情导航空值保护和更新 ID 类型收窄。
  - 已跑前端四件套：架构检查、类型检查、单测、生产构建均通过。

### 事项 7：后端开源质量盘点

- 状态：Review
- 建议窗口：`codex/backend-open-source-readiness`
- 目标：后端服务、配置、测试入口对外可理解。
- 交付物：
  - `core/*` 服务清单。
  - 官方后端测试命令。
  - 数据库 migration/seed 初始化说明。
  - NATS、MinIO、MySQL 等依赖说明。
  - 哪些目录不能直接 `go test ./...` 的原因说明。
- 验收：
  - 后端 README 不依赖口口相传。
- 本轮结果：
  - 新增 `docs/backend-open-source-readiness.md`，列出 core 服务、依赖、官方测试命令、migration/seed 说明和公开边界。
  - 根 README、`docs/README.md` 已链接该入口。

### 事项 8：文档归档和删除

- 状态：Review
- 建议窗口：`codex/document-archive-cleanup`
- 目标：把历史文档从根目录和散乱位置收干净。
- 交付物：
  - 已建立 `docs/archive/`。
  - 已归档根目录历史文档。
  - 已清理 `web/` 下阶段性报告。
  - 已归档确认无有效引用的重复 TODO。
- 候选：
  - `Agent工作流流程图.md` -> `docs/archive/Agent工作流流程图.md`
  - `项目介绍.md` -> `docs/archive/项目介绍.md`
  - `项目架构.md` -> `docs/archive/项目架构.md`
  - `web/ARCHITECTURE_ANALYSIS.md` -> `docs/archive/web-ARCHITECTURE_ANALYSIS.md`
  - `web/COLOR_ANALYSIS.md` -> `docs/archive/web-COLOR_ANALYSIS.md`
  - `web/THEME_COMPARISON_REPORT.md` -> `docs/archive/web-THEME_COMPARISON_REPORT.md`
  - `web/THEME_FIX_SUMMARY.md` -> `docs/archive/web-THEME_FIX_SUMMARY.md`
  - `web/todos.md` -> `docs/archive/web-todos.md`：内容为粗粒度旧 TODO，后续治理统一以本看板为准。
- 验收：
  - 已执行引用检查，仅治理清单引用候选文件。
  - 仍有价值的内容已迁移并标注“历史归档”。
  - 需人工复核归档内容是否仍有敏感信息或未迁移决策。
- 本轮结果：
  - 新增 `docs/archive/README.md`，`docs/README.md` 已标出历史归档层。
  - 未归档当前仍作为入口引用的 `docs/工作台最新设计.md` 及其配套文档。
  - 已按 `DOCUMENT_GOVERNANCE.md` 为当前主项目文档、模块入口文档和归档文档补齐状态、更新时间和负责人窗口。

### 事项 9：产品示例和 SDK 入门

- 状态：Review
- 建议窗口：`codex/examples-and-sdk-docs`
- 目标：让外部用户看到项目能做什么。
- 交付物：
  - Form 最小示例。
  - Table 最小示例。
  - Chart 最小示例。
  - 一个完整轻应用 walkthrough。
  - SDK 入口文档整理。
- 验收：
  - 用户能从 0 创建一个可运行轻应用。
- 本轮结果：
  - 新增 `docs/examples/README.md`，包含 Form、Table、Chart 最小示例和轻应用 walkthrough。
  - 示例入口已加入根 README 和 `docs/README.md`。

## P2：持续治理

### 事项 10：文档链接和过期检查

- 状态：Review
- 建议窗口：`codex/docs-link-check`
- 目标：减少失效链接、冲突文档和过期说明。
- 交付物：
  - 文档索引生成脚本或检查脚本。
  - 过期文档顶部标记规范。
  - CI 可选检查。
- 本轮结果：
  - 新增 `scripts/check-doc-links.sh`，检查 Markdown 本地链接。
  - CI 已将文档链接检查作为治理检查的一部分。
  - `docs/archive/README.md` 明确历史归档文档不作为默认入口。
  - 已执行 `bash scripts/check-doc-links.sh`，本地 Markdown 链接检查通过。
  - 已执行 `bash scripts/check-sensitive-files.sh`，敏感文件边界检查通过。

### 事项 11：发布说明和版本策略

- 状态：Review
- 建议窗口：`codex/release-process`
- 目标：外部用户知道版本稳定性和升级路径。
- 交付物：
  - `CHANGELOG.md`
  - release checklist
  - 版本号策略
  - breaking change 说明模板
- 本轮结果：
  - 新增 `CHANGELOG.md`。
  - 新增 `docs/governance/RELEASE_PROCESS.md`，包含版本策略、release checklist、breaking change 模板。

## 并行领取规则

1. 一个窗口只领取一个事项。
2. 每个事项独立分支，分支名使用本文件建议名。
3. 每个窗口交付时必须更新本 TODO 状态。
4. 如果事项需要改根 `README.md`，先在本文件登记，避免多个窗口互相覆盖。
5. 涉及删除、许可证、公开边界的事项必须人工确认后再合并。

## 当前推荐开工顺序

第一批并行：

1. 事项 1：文档入口和开源门面。
2. 事项 3：敏感信息和发布包边界。
3. 事项 4：CI 和质量门禁。
4. 事项 6：前端架构债继续清。

第二批并行：

1. 事项 5：本地启动和部署体验。
2. 事项 7：后端开源质量盘点。
3. 事项 8：文档归档和删除。
4. 事项 9：产品示例和 SDK 入门。

许可证事项需要先拍板，不建议让窗口自行决定。
