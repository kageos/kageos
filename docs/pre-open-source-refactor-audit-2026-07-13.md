# Kageos 开放源码前重构审计报告

> 审计日期：2026-07-13
>
> 审计目标：在尚无历史用户、允许破坏性变更的窗口内，找出应在源码公开前一次性解决的安全、契约、架构、数据一致性、性能和工程治理问题。
>
> 重要：本报告包含可直接定位安全缺陷的细节。P0 修复并完成密钥轮换前，不建议把本报告原样发布到公开仓库。

## 1. 结论先行

当前代码基线可以通过，工程也已经具备 CI、贡献指南、安全策略、许可证说明等基础设施；但还不适合直接进入“以后尽量保持兼容”的阶段。

最需要利用当前窗口处理的不是格式、命名或单个大文件，而是五类会形成长期包袱的问题：

1. **存储系统存在 P0 级权限与凭据边界问题。** HTTP 请求可以选择 `upload_source=server`，服务会把 MinIO 长期 `access_key/secret_key` 放进响应；同时多个文件接口只按客户端传入的 `router/bucket/key/ref` 操作，没有把对象绑定到当前用户、工作区或公开分享能力。
2. **公开分享令牌不是分享级能力令牌。** 令牌有效期一年，只包含匿名 session，不包含 `share_id`、资源范围和允许操作；存储服务又没有独立验证分享是否存在、是否启用、是否匹配路由。
3. **应用创建、更新、删除横跨 MySQL、runtime 文件、Git、容器、SQLite，但没有持久化操作状态、补偿和对账。** 任何中途失败都可能制造控制面与运行态漂移；同一工作区并发修改也没有可靠串行化。
4. **HTTP 契约、数据库约束和依赖装配尚未定型。** 业务失败普遍返回 HTTP 200/code 7，Swagger 与真实响应不一致；关键自然键缺少数据库唯一约束；生产启动依赖 `AutoMigrate`；Service 使用 setter 补依赖只是“组合根失控”和职责过大的一个表象。
5. **用户/AI 生成代码的运行隔离还没有形成可信默认值。** 生产用户 App 默认使用 host network，挂载工作区可写目录，容器启动参数没有统一的 CPU/内存/PID、capability、只读根文件系统和 egress 限制；主容器本身又要求 privileged。只要工作区代码不是完全可信，这就是平台级安全边界。

建议把开放源码前的改造范围锁定为：**P0 全部、P1 全部、P2 中会改变公开 API/数据模型/扩展接口的部分**。纯文件拆分、日志美化、一般性能优化可在契约冻结后继续迭代。

### 1.1 2026-07-13 第一批改造进度

本报告是改造前基线，下面项目已经在当前工作树完成，原问题章节保留作为设计依据和验收清单：

| 项目 | 当前状态 |
|---|---|
| MinIO 长期凭据响应、客户端 `upload_source=server` | 已移除；浏览器统一使用短期预签名 URL，仍需在部署侧轮换历史凭据 |
| 固定 LLM 密钥 fallback、宽松 CORS、前端全局 store 暴露 | 已移除或改为显式配置 |
| Service setter 注入 | App/runtime 关键服务已改为构造注入，组合根一次性提供必需依赖 |
| HTTP 错误协议 | 已引入稳定错误码、真实 HTTP 状态和 transport-independent `apperror`；518 个 `FailWithMessage` 调用及文本猜测器已清零，未知错误统一隐藏为 500 |
| API 路由 | app、access、agent、storage、HR、message、timer、connector 及 workspace Table/Form/Chart/selection/directory 路由已资源化；未保留旧别名 |
| Repository context | 235 个旧导出方法和 587 个调用点已迁移；数据库访问统一使用 `WithContext(ctx)`，仅 3 个不执行 I/O 的 `WithDB` 构造方法不接收 context |
| 分页与 N+1 | App 可见性分页改为单查询后分页并返回真实 total；Connector 目录绑定改为批量读取连接 |

本批没有处理 SEC-02～SEC-05、跨系统 Saga/补偿、数据库唯一约束/迁移治理等高风险项目，不能因为本表完成就认为已达到开放发布条件。

## 2. 审计范围与方法

### 2.1 范围

- Go：`cmd/`、`core/`、`dto/`、`pkg/`
- Web：`web/src/` 及构建、lint、类型、单测配置
- 数据模型、服务装配、跨服务调用、HTTP 路由、鉴权、存储、公开分享
- CI、许可证、贡献与发布治理

没有进行真实生产环境渗透测试，也没有连接外部 MySQL、MinIO、NATS、Podman/Docker 集群做故障注入。因此本文是**代码级架构与安全审计**，不是正式渗透测试或容量测试结论。

### 2.2 规模信号

- 约 1,627 个文本文件，手写代码约 30.6 万行。
- Go 约 15.5 万行，Vue 约 8.3 万行，TypeScript 约 6.7 万行。
- 140 个手写文件超过 500 行，58 个超过 800 行，28 个超过 1,200 行。
- 123 个 Go 包中 62 个没有 `_test.go`；存储 handler/server/repository、connector API/repository、runtime API/repository/model 等关键层存在明显空白。
- 生产前端代码约有 480 处 `any`；兼容/legacy/deprecated/临时类标记较多，说明旧契约尚未完成收敛。

这些数字不是问题本身，但说明“继续在现有形态上加兼容层”会快速放大维护成本。

### 2.3 基线验证

以下检查本次均通过：

- 后端 `go vet` 与现有 Go 测试脚本。
- app-server、app-runtime、app-storage、hr-server、api-gateway、connector-server、message-server、timer-scheduler 的 `go test -race`。
- 前端架构边界检查、ESLint、Vue TypeScript 类型检查。
- 前端 106 个测试文件、450 个单元测试。
- 前端生产构建。

生产构建存在依赖被同时静态/动态导入的 chunk 警告，以及 ECharts、编辑器、主工作区等较大的产物，但没有构建失败。需要强调：现有测试通过只代表覆盖到的行为稳定，**不代表缺少负向权限测试的路径是安全的**。

## 3. 优先级定义

| 等级 | 含义 | 开放源码前策略 |
|---|---|---|
| P0 | 可导致凭据泄漏、跨租户访问、公开入口越权或不可接受的数据安全风险 | 立即冻结相关公开能力，修复、补负向测试、轮换密钥后再发布 |
| P1 | 会造成数据/运行态漂移、核心契约锁死、认证失效或大规模返工 | 必须在公开 API 和数据模型冻结前完成 |
| P2 | 显著影响可维护性、性能、可测试性和扩展者体验 | 其中涉及公开接口者应前置，其余可分阶段完成 |
| P3 | 代码整洁度、日志风格、局部重复和体验优化 | 可在开源后按模块持续处理 |

## 4. 问题清单

## 4.1 P0：开放源码前必须先修

### SEC-01：浏览器可请求服务端上传模式并获得 MinIO 长期根凭据

**证据**

- [`getDefaultUploadSource`](../core/app-storage/api/v1/helpers.go#L15) 对客户端传入的枚举直接放行。
- 普通上传接口在 [`storage.go`](../core/app-storage/api/v1/storage.go#L541)、批量上传和公开分享上传中都使用请求里的 `upload_source`。
- [`MinIOStorage`](../core/app-storage/storage/minio.go#L206) 在 `server` 模式把 `access_key`、`secret_key`、endpoint、bucket 放入 `SDKConfig`。
- [`buildUploadTokenResponse`](../core/app-storage/api/v1/helpers.go#L59) 把该配置原样写入 HTTP DTO。

**影响**

任一可调用上传令牌接口的用户都可能拿到存储长期凭据。公开分享上传入口还会与 SEC-03 叠加，使风险不再局限于已登录用户。拿到凭据后，攻击面不受应用层 `router` 校验约束。

**改造要求**

- 从所有 HTTP DTO 中永久删除长期存储凭据和 `SDKConfig`。
- 浏览器只获得单对象、短时、限定 method/content-type/content-length 的预签名能力。
- 真正的内部服务上传使用独立机器身份；如需临时凭据，使用 STS/短期 scope，并走仅内部可达且独立鉴权的接口。
- 上传来源由服务端根据调用身份决定，不能由客户端枚举决定。
- 修复完成后轮换当前 MinIO access/secret key，并审查日志、历史响应和示例配置是否留有旧密钥。

**验收**

- 对所有公开/登录上传接口断言响应中永远不存在长期凭据。
- `upload_source=server` 来自浏览器时返回 400/403，而不是降级或返回凭据。
- 泄漏后的旧凭据已失效。

### SEC-02：文件接口缺少对象级、工作区级授权，存在 IDOR/跨租户操作面

**证据**

`app-storage` 路由主要只挂 JWT，handler 接受客户端传入的 `router`、`bucket`、`key` 或 `ref`；以下路径没有建立“对象属于当前用户/当前工作区”的统一授权条件：

- 生成/批量生成上传令牌与上传完成通知。
- [`ResolveFileRefs`](../core/app-storage/api/v1/storage.go#L907) 与描述更新。
- [`DownloadFile`](../core/app-storage/api/v1/storage.go#L961)、[`DeleteFile`](../core/app-storage/api/v1/storage.go#L1032)、文件信息。
- 按 router 统计、列举和批量删除。

服务层同样主要按传入的 bucket/key/router 执行，handler 的 JWT 只证明“是谁”，没有证明“能否操作这个对象”。

**影响**

一个登录用户可能通过猜测或获取其他工作区的 ref/key，生成下载地址、查看信息、改描述、删除文件，甚至按 router 批量删除。

**改造要求**

- 建立规范化 `StorageObject` 所有权模型：`tenant/workspace/resource_path/object_id/owner/visibility/status`，外部只使用不可猜测的 object ID，不接受任意 bucket/key 作为资源身份。
- 统一 `StorageAuthorizer`，每个读写操作都以 `(subject, action, object/workspace)` 判定；bucket 和物理 key 由服务端解析。
- 将物理存储 key 视为内部实现细节；禁止普通 API 任意指定 bucket。
- 下载同样使用短时、单对象 URL；删除、描述更新、批量操作必须走同一权限判定。
- 增加用户 A/用户 B、owner/admin/member/viewer、公开/私有、父子资源路径的权限矩阵测试。

### SEC-03：公开分享匿名令牌没有绑定 share、资源和操作

**证据**

- [`AnonymousClaims`](../pkg/publicshare/anonymous_token.go#L23) 只有 `typ/sid/iat/exp`，有效期为一年；没有 `share_id`、tenant、resource、audience、operations。
- [`publicShareRequestUser`](../core/app-storage/api/v1/storage.go#L189) 只验证通用匿名 token 和 URL 中存在 `share_id`，没有加载并验证分享记录，也没有证明请求 router 属于该分享。
- [`PublicShareGetUploadToken`](../core/app-storage/api/v1/storage.go#L223) 继续使用客户端 router/bucket/upload_source。
- 公开文件 ref 解析没有把每个 ref 绑定到该分享产生的对象集合。

**影响**

一个匿名 token 可在不同分享之间复用，任意拼接 share ID/router；已撤销、禁用或过期的分享也不能由 storage 独立识别。与 SEC-01、SEC-02 叠加时，公开入口可能变成存储系统通用能力入口。

**改造要求**

- 改成短时分享能力令牌，至少包含 `share_id`、tenant/workspace、精确资源前缀、允许操作、audience、issued/expiry、nonce/version。
- storage 必须通过可靠的分享 introspection 或可验证签名，确认分享存在、启用、未过期、路径匹配。
- 撤销通过短 TTL + share version/denylist 生效。
- 每个匿名上传对象记录 share/session，解析和下载只能访问同一 capability 创建或明确授权的对象。

### SEC-04：上传完整性依赖客户端声明，公开 SVG/任意大文件形成滥用与存储型 XSS 面

**证据**

- 企业 Logo 公开上传允许 `image/svg+xml`，MIME 来自请求声明；没有内容嗅探、SVG 清洗或栅格化。
- 普通上传签发 PUT URL 时只检查声明的 `file_size`；PUT 本身没有可靠绑定实际大小，完成通知也信任客户端回报的 key、size、content type。
- 没有看到统一的租户配额、上传后 `StatObject` 校验、病毒/恶意内容扫描和隔离状态。

**影响**

攻击者可谎报类型/大小，上传超大文件或主动内容。若 SVG/HTML 等由同源 CDN inline 返回，可形成存储型 XSS；即使不能执行，也可造成存储和带宽滥用。

**改造要求**

- 使用带 `content-length-range`、MIME 白名单和对象 key 约束的 POST policy，或上传完成后服务端 stat 校验并自动删除不合规对象。
- 公开 Logo 只接受经过解码后重新编码的 PNG/JPEG/WebP；默认禁用 SVG，确需支持时用成熟 sanitizer 且从独立静态域名返回。
- 用户内容使用独立 origin，设置 `Content-Disposition`、`X-Content-Type-Options: nosniff` 和严格 CSP。
- 增加用户/工作区配额、并发限制、过期未完成上传清理与审计。

### SEC-05：用户 App 默认 host network 且缺少完整沙箱资源/网络策略

**证据**

- 生产配置说明用户 App 容器由内层 Podman 以 host network 启动，并可直接访问 `127.0.0.1` 上的基础设施。
- [`podmanRunBaseArgs`](../core/app-runtime/service/container_service.go#L1014) 主要设置名称、可写 volume、时区和可选 network；没有统一加入 `--read-only`、`--cap-drop=all`、`no-new-privileges`、非 root 用户、CPU/内存/PID 限额和 egress policy。
- AppArmor/SELinux 保护在 profile 未加载时会记录日志后继续启动，不是 fail closed。
- 生产 main 容器本身要求 `privileged: true` 并在内部暴露 Podman API，扩大了运行时被攻破后的爆炸半径。

**影响**

Kageos 会构建并运行用户或 AI 生成的 Go/Python 代码。若把这些代码视为不完全可信，host network 允许它探测控制面、数据库、NATS、MinIO 和云 metadata 等内部地址；缺少资源上限会造成宿主机 DoS；可写工作区会破坏源码、版本和运行元数据。这里不能只依赖应用层鉴权，因为容器隔离本身就是产品的安全边界。

**改造要求**

- 先写正式 threat model，明确“谁可以提交代码、代码是否可信、租户之间承诺什么隔离”。如果只支持可信单管理员，UI、文档和部署模式必须明确，不能暗示多租户安全。
- 默认使用独立 bridge/network namespace；只通过受鉴权的 gateway/代理访问必要服务，禁止直连控制面和基础设施端口。
- 以非 root、drop all capabilities、no-new-privileges、只读根文件系统、临时 tmpfs、seccomp/LSM、CPU/内存/PID/磁盘/超时限额运行。
- 源码和元数据只读挂载，输出目录单独可写；容器不应看到 Podman/Docker socket、宿主机敏感路径或平台管理员凭据。
- 默认 deny egress，再按能力授予域名/端口；必须防 DNS rebinding、重定向到内网、loopback/link-local/metadata 地址。
- 隔离策略不可用时，多租户/非可信模式必须启动失败，而不是记录 warning 后继续。

**验收**

恶意示例 App 无法访问平台 `127.0.0.1` 服务、云 metadata、其他工作区文件或容器 socket；fork bomb、内存/磁盘占满和超时进程均被配额终止；重启后工作区源码和元数据未被应用进程篡改。

## 4.2 P1：会锁死架构或造成状态不一致

### ARC-01：应用生命周期跨系统操作没有 Saga、幂等和对账

**现状**

- 创建流程先 provision runtime，再写 app-server MySQL；数据库失败会留下运行态和文件孤儿。
- [`AppService.DeleteApp`](../core/app-server/service/app_service.go#L982) 先调用 runtime 删除，再删除控制面记录；后一步失败会留下“元数据仍在、运行态已没”的半删除状态。
- [`AppManageService.DeleteApp`](../core/app-runtime/service/app_manage_service.go#L273) 对部分容器、文件、SQLite 删除错误只记录日志并继续，最终仍可能返回成功。
- 更新流程涉及源码写入、构建、Git commit、版本元数据、容器和控制面版本更新，中途失败没有持久化步骤状态。

**建议目标模型**

- 控制面存 `AppOperation`：`operation_id/idempotency_key/type/desired_version/state/current_step/error/retry_count`。
- create/update/delete 每一步可重复执行，明确成功判定和补偿；删除采用 tombstone，资源全部清理后再 finalize。
- runtime 上报 observed state；后台 reconciler 对比 desired/observed 并自动修复或标记人工处理。
- 所有跨服务命令带 operation ID 和期望版本；重复请求返回同一结果。

**验收**

对每一步注入超时、5xx、进程退出，重启后均可继续或补偿，不出现无记录容器、无文件版本、控制面与 runtime 永久漂移。

### ARC-02：同一工作区变更没有可靠串行化和乐观并发控制

`UpdateApp` 会写源码、运行构建、提交 Git、分配版本、写 `version.json/current_version`、更新容器。现有 mutex 主要保护 waiter/cleanup，并不是工作区 mutation lock。并发更新、批量写、删除可能发生版本号碰撞、Git worktree 冲突和 lost update。

**改造要求**

- 单进程不是充分假设：使用按 workspace 的数据库 lease/advisory lock 或有 fencing token 的分布式锁。
- 请求携带 `expected_version`/ETag，控制面用 CAS 更新；冲突返回 409。
- 读操作与 mutation 明确分离；创建、更新、删除、恢复、数据库迁移都进入同一 operation coordinator。

### ARC-03：公开分享使用次数在业务副作用之后扣减

当前提交逻辑先执行被分享的应用函数，再调用 [`IncrementUseCount`](../core/app-server/service/public_share_service.go#L181)。虽然 repository 的条件更新可以防止计数超过上限，但并发请求可能都先完成业务副作用，后扣减失败的请求只是在“事情已经发生后”收到超限。

**改造要求**

- 调用前原子 reserve 一次使用额度，生成 invocation ID/idempotency key。
- 定义哪些失败释放 reservation，哪些已产生副作用仍计费。
- 应用调用与结果记录可重试但不重复副作用；公开写操作默认要求显式幂等键。

### API-01：HTTP 状态码与业务错误模型不标准，Swagger 和客户端被迫猜错误

[`pkg/ginx/response`](../pkg/ginx/response/response.go#L21) 除未认证外，成功和失败都返回 HTTP 200；参数错误、权限不足、未找到、冲突和内部错误通常都压成 `code=7`。Swagger 注解却声明 400/403/404/500，真实协议与文档不一致。前端 `authSession` 已出现根据 code 7 和中文关键词猜会话错误的补偿逻辑。

**破坏性改造建议**

- HTTP status 表达协议语义：400 validation、401 unauthenticated、403 forbidden、404 not found、409 conflict、422 domain validation、429 limit、500/502/503 系统错误。
- 统一错误体：`{ code: stable_machine_code, message, details?, trace_id }`；`code` 使用稳定字符串，不依赖中文消息。
- 定义 typed domain error 与集中 mapper，handler 不再随手 `err.Error()` 返回客户端。
- 成功响应可以统一 envelope，也可以直接资源；二选一后全仓一致。
- OpenAPI 描述真实 envelope/error schema，并生成前端 client/types 和契约测试。

### API-02：路由和资源身份尚未形成可长期承诺的规范

当前混合 action 风格、单复数、下划线和不同身份字段，例如 `/app/list|detail|delete`、`/llm/get_default|set_default`、`/batch_detail`、`/run_now`、`/notification_channels`，同一对象又可能用 `resource_path`、`full_code_path`、router、user/app、wildcard 表示。

**建议**

- 在无用户窗口直接发布 `/api/v1` 新规范，不保留旧路由兼容层。
- 资源化：`GET /apps`、`POST /apps`、`GET/PATCH/DELETE /apps/{app_id}`、`POST /tasks/{id}:run`（确属 command 时统一冒号动作）。
- 建立唯一 `ResourcePath` 值对象：标准化、解析、父子关系、编码和权限判断只实现一次；外部响应优先稳定 ID，path 作为可读定位符。
- 禁止 GET body；分页、过滤、排序规范统一。

### API-03：列表分页在合并授权应用后失真

[`GetApps`](../core/app-server/service/app_service.go#L1015) 先分页查询自己的应用并丢弃 repository 返回的 total，再合并未分页的授权应用，最后用当前返回数组长度作为 `TotalCount`。这会导致页大小不稳定、跨页重复/遗漏、总数错误。

**改造要求**

把 owner/public/role access、search、type filter 合并为一个数据库查询或稳定 union，固定排序后只分页一次，并从同一过滤条件计算 total。增加多页授权混合数据测试。

### DATA-01：关键自然键和角色分配主要靠“先查后写”，数据库没有最终裁决

需要重点补齐的约束包括：

- app 的 `(tenant/user, code)`；如业务要求，也包括 `(tenant/user, name)`。
- service tree 的 canonical full resource path。
- runtime app 的 `(user, app)`、app version 的 `(user, app, version)`。
- workspace role assignment 的主体、资源、角色组合唯一性。

当前部分 upsert 是 select 后 create/save，两个并发请求可同时通过检查并插入重复记录。

**改造要求**

- 先清理重复数据，再加数据库 composite unique index。
- 使用数据库原子 upsert，并正确区分 conflict/duplicate。
- admins 等 CSV 字符串改为关系表；关键状态/类型建立受控值与状态机。
- 明确外键、软删除后唯一性、级联与孤儿清理策略。

### DATA-02：生产启动时 AutoMigrate 和 ALTER 数据库，不是可审计的 schema 生命周期

多个服务的 `model/init.go` 在启动时 `AutoMigrate`；agent-server 还会执行 `ALTER DATABASE`/字符集转换，并要求生产运行账号拥有 ALTER 权限。迁移 SQL 同时散落在启动逻辑和脚本中。

**改造要求**

- 使用有版本号、顺序、checksum 和 `schema_migrations` 的显式 migration。
- migration 使用单独高权限身份，应用运行账号最小权限。
- 发布物包含 upgrade/downgrade 或 forward-fix 说明、备份要求和大表风险说明。
- `AutoMigrate` 仅允许测试/本地开发；CI 必须测试“空库到当前”和“上一个发布版本到当前”。

### CFG-01：全局配置 singleton 和服务内部重新取配置制造隐藏依赖

`pkg/config` 使用 `sync.Once` 全局配置；部分读取失败打印后返回空/默认配置。虽然 server composition root 已拿到 cfg，多个 service/package 又重新调用 `GetGlobalSharedConfig()`。这会造成测试相互污染、不同组件看到不一致 snapshot、无法支持多实例配置和严格启动校验。

**改造要求**

- 配置只在 composition root 加载一次、校验一次，失败立即退出。
- 以不可变的最小配置 struct 注入组件，不把整个全局大配置传入所有服务。
- secret、URL、timeout、feature flag 分层；禁止生产 secret fallback。
- 去掉可变全局指针；测试显式构造配置。

### AUTH-01：令牌签发在 session 落库失败时仍返回成功

[`IssueTokensForUser`](../core/hr-server/service/auth_service.go#L215) 对 session 保存失败只记录日志，仍返回 access/refresh token；[`RefreshToken`](../core/hr-server/service/auth_service.go#L286) 对旋转后的 session 更新失败也可能继续返回新 token。这样令牌可能无法按预期撤销或刷新，持久化状态与客户端状态不一致。

**改造要求**

- session 创建/旋转失败必须 fail closed，不返回新 token。
- 数据库存 refresh token 的不可逆 hash，而不是可直接使用的 token；实现 rotation family、reuse detection 和并发刷新冲突处理。
- 登录与验证码加入 IP、账号、设备维度的速率限制和尝试次数；验证码限流采用原子操作。
- 错误响应避免暴露账号是否存在。

### AUTH-02：LLM API key 加密存在全项目共享的硬编码开发 fallback

[`llm_service.go`](../core/agent-server/service/llm_service.go#L120) 在专用 secret 和 JWT secret 都缺失时使用固定字符串 `kageos-llm-api-key-dev-secret`。任何漏配部署都会使用公开源码中相同的密钥。

**改造要求**

- 生产模式缺少独立 encryption key 时启动失败。
- 密文带 key ID/version，提供密钥轮换和重加密流程。
- JWT 签名 key、数据加密 key、匿名分享 key、OpenAPI token key 分离。

### WEB-01：refresh/access token 放 localStorage，XSS 后可被长期窃取

前端 auth store 与请求层从 localStorage 读取 access/refresh token；同时应用存在多处 `v-html`/富文本渲染，当前主要依靠自研 sanitizer，未发现统一 CSP/Trusted Types/security headers 落地。另有 [`Cors`](../pkg/middleware/cors.go#L7) 同时设置 `Access-Control-Allow-Origin: *` 和 `Access-Control-Allow-Credentials: true`：这组策略本身不符合 credentialed CORS 规则，也无法支撑后续安全 cookie 方案。

**改造要求**

- refresh token 放 `HttpOnly; Secure; SameSite` cookie，access token 优先仅存内存并缩短 TTL。
- 使用成熟且严格 allowlist 的 sanitizer（如 DOMPurify），集中审计所有 `v-html`、URL scheme、SVG/data URL。
- 网关统一返回 CSP、frame-ancestors、nosniff、referrer policy 等安全头；逐步启用 Trusted Types。
- CORS 使用显式 origin allowlist、正确 `Vary: Origin` 和最小 methods/headers；只有确需跨站 cookie 的受信 origin 才允许 credentials，并增加 CSRF 防护。

## 4.3 P2：接口冻结前优先处理

### DI-01：Service 的 `SetXxx` 多数确实是设计问题，但 setter 不是原罪

用户指出的现象成立。需要区分三类：

1. **必需的构造依赖**：`AppService.SetTeamAccessService`、`SetFunctionSensitiveFieldService`、`SetDocService`，以及 runtime 的 `SetAppDatabaseService`。这些对象在 setter 前处于“可构造但不能可靠使用”的半初始化状态；并发启动或漏装配会变成 nil 分支/运行时错误。应改为 constructor/config 参数，并在构造时校验。
2. **真正的运行期 callback/hook**：如 discovery callbacks、watchdog recovered callback。如果语义就是启动前注册并可替换，setter 可以保留，但要有线程安全和生命周期约束；更稳妥的是 constructor option。
3. **Builder/DTO 设置字段**：例如 builder 的 `SetFields`，不属于依赖注入问题。

[`server.go`](../core/app-server/server/server.go#L291) 的现有装配没有显示出无法解开的真实循环依赖；相关服务可以先构造再一次性传给 `NewAppService`。runtime 的 AppDatabaseService 也在 AppManageService 前创建，可以直接注入。

更根本的问题是 `AppService` 同时承担应用生命周期、权限、API 元数据、文档同步、审计等职责，甚至跨越 service 边界访问文档内部 repository。只把 setter 塞进一个越来越长的 constructor 不够，应拆为 `CreateApp/UpdateApp/DeleteApp/ListApps/ManageAppDocs` 等用例服务，共享小接口。

**验收原则**

- 构造完成的 service 立即可用，不要求调用者记住第二阶段初始化。
- 必需依赖不可为 nil；可选行为使用明确 `Noop` 实现或 option，而不是到处 `if dep != nil`。
- composition root 是唯一装配位置，service 不自行从全局取其他 service/config。

### ARC-04：God Service/God Component 与层边界泄漏

后端 `app_manage_service.go`、`app_service.go`、storage API、standard API 等文件超过千行；前端 TeamAccessPage、MiniWorkstation、WorkspaceInbox、StructuredPromptComposer 等超过两千行。大文件不是按行数机械拆分，而应按“变化原因”和事务边界拆分。

**建议边界**

- application use case 只编排 domain 与 ports；HTTP/DTO mapping 留在 adapter。
- runtime build、source workspace、versioning、container deployment、database migration 分为明确组件，由 operation coordinator 编排。
- Vue 页面拆为 view model/composable + 场景组件；网络和持久化不留在巨大 presentation component。
- 移除 service 直接访问另一个 service 私有 repository 的做法。

### PERF-01：TeamAccess 批量授权是三重循环逐项远程/数据库调用且无事务

[`BatchAssign`](../core/app-server/service/team_access_service.go#L180) 对 resource × username × role 逐个调用 `Assign`，每次重复权限检查、HR 查询、upsert 和异步审计；中途失败产生部分成功。

**改造要求**：一次性标准化与鉴权、批量校验用户、事务内 bulk upsert、单次/批量审计；明确 all-or-nothing 或返回逐项结果，不能隐式部分成功。

### PERF-02：Connector 目录绑定列表存在逐行查询连接信息的 N+1

[`ListDirectoryBindings`](../core/connector-server/service/connector_service.go#L221) 列出 binding 后逐个调用 `GetOwnedConnection`。应使用 join/preload 或收集 connection IDs 后单次批量查询，并补 query-count 测试。

### PERF-03：App 列表缓存返回共享 model 指针，失效规则复杂且可能数据竞争

AppRepository 的 map 缓存保存并返回 `*model.App`。mutex 只保护 map，不保护调用者拿到指针后的字段修改；多个 key 还可能指向同一对象。已有调用点需要手动 copy，说明抽象不安全。

在当前规模下优先移除该缓存，让数据库索引和查询先正确；确需缓存时返回不可变 value/deep copy，设置容量/TTL/指标，并设计事务提交后的统一失效或 outbox。

### IO-01：大量 repository I/O 方法不接收 context

改造前粗略扫描 326 个导出 repository 方法，只有约 86 个显式接收 `context.Context`，约 240 个没有。即使上层 service 有 ctx，GORM 未 `WithContext` 时，请求取消、deadline 和 trace 无法传到数据库。

**改造要求**

- 所有外部 I/O port 第一个参数为 ctx；repository 查询统一 `db.WithContext(ctx)`。
- transaction 从同一 ctx 创建；禁止请求路径换成 `context.Background()`。
- linter/架构测试阻止新增无 ctx repository 方法。

### REL-01：审计日志用无界 goroutine fire-and-forget

App、TeamAccess、表单等路径在每次操作中 `go func` 写日志；批量删除可为每行创建 goroutine。storage 下载日志还在 handler 返回后使用被 Gin 池复用的 context。进程退出会丢日志，高负载下没有背压、重试和容量上限。

**改造要求**

- 对安全/合规审计使用事务 outbox；普通操作日志使用有界队列和固定 worker。
- 批量插入、失败重试、死信、队列深度指标、优雅停机 flush。
- goroutine 只持有复制后的不可变字段和标准 ctx，绝不在请求结束后捕获 `*gin.Context`。

### API-04：OpenAPI、Go DTO 和 TS 类型多套手工事实源

Swagger 成功注解常写 raw DTO，而真实响应有 envelope；部分服务文档覆盖不足。前端又维护大量手写接口和 `any`，容易在改名/空值/错误结构上静默漂移。

**改造要求**

- 选择唯一规范源：code-first 也可以，但必须生成稳定 OpenAPI artifact。
- CI 校验 spec diff，生成 TS client/types，禁止手工复制核心 DTO。
- 对每个 endpoint 加契约测试，至少覆盖成功、validation、401、403、404、409、500。

### GW-01：配置支持多个 target，但网关并未负载均衡且静默使用第一个

[`createLoadBalanceProxy`](../core/api-gateway/server/router.go#L371) 明确记录“未实现真正负载均衡”，却接受多 target 配置并回退第一个。这是伪功能：运维以为有冗余，实际上没有。

开放源码前二选一：实现带健康检查、熔断和明确重试语义的负载均衡；或 schema 只允许一个 target，配置多个时启动失败。流式请求也不应靠 path 包含 `/stream` 推断，应由 route metadata 明确声明 timeout/streaming/retry policy。

### OBS-01：错误暴露、日志风格和可观测性不足以支持自托管排障

- 很多 handler 把 `err.Error()` 直接返回，可能泄露 SQL、内部地址或第三方响应。
- 日志包含大量装饰符号和自由文本，字段规范不统一。
- 需要明确 liveness/readiness；readiness 应验证必要的 MySQL/NATS/MinIO/runtime 依赖。
- 缺少统一的 request count、latency、error、queue、operation/reconcile、DB pool 和外部调用指标。

建议统一结构化日志字段与 redaction；客户端消息稳定安全，完整 cause 只进带 trace ID 的服务端日志。

### WEB-02：前端边界检查偏文本规则，类型与生产调试口仍需收紧

- 当前架构脚本能挡住一部分 import 越界，但 regex 不能替代模块公开 API 和类型约束。
- [`main.ts`](../web/src/main.ts#L63) 把 stores 暴露到 `window.__stores__`，生产构建也会存在。
- 大量 `any` 和巨型页面让 transport/domain/presentation 边界容易回流。

改造为：生产移除全局 store 暴露；ESLint/no-restricted-imports + package public entry enforcement；生成 API 类型；按场景拆 composable/view model，并为关键 composable 增加状态转换测试。

## 4.4 P3：可以持续治理，但要设防回归

- 删除经引用证明无用的 legacy 字段、兼容 re-export、旧状态和旧路由；因为当前无用户，不建议为它们新增兼容层。不能只凭 `legacy` 关键字删除，必须先做引用、数据与配置扫描。
- 统一日志语言、标点和字段；删除 emoji/star 类装饰日志。
- 合并重复 path/router 处理函数，避免不同模块对前导斜杠、大小写和编码产生不同解释。
- 为大文件设置新增代码预算：先禁止继续增长，再随业务改动按边界拆分，避免纯机械搬文件。
- 前端做 route-level/component-level chunk 分析，重点处理编辑器、ECharts 和 WorkspaceView；性能优化以真实首屏与交互指标为准，不按 bundle 数字盲拆。

## 5. 建议的目标架构原则

### 5.1 依赖与用例

```text
HTTP/NATS/CLI adapter
        ↓ DTO mapping + auth context
Application use case（事务/operation 编排）
        ↓ ports/interfaces
Domain policy/value objects
        ↓
Repository / Runtime / Storage / HR adapters
```

- Service 不读 Gin context，不返回 HTTP 语义，不依赖全局配置。
- Domain error 在 adapter 边界映射为 HTTP/NATS 错误。
- 跨系统修改必须是持久化 operation；单库修改必须是明确事务。
- 权限判定输入始终是规范化 subject/action/resource，而不是散落字符串比较。

### 5.2 三个必须唯一的事实源

1. **API schema**：OpenAPI + 生成客户端。
2. **资源身份**：ResourcePath/ObjectID 值对象及其解析规则。
3. **数据库 schema**：版本化 migration，不是各服务启动时猜测。

## 6. 推荐执行顺序

### Phase 0：立刻止血与建立护栏（1–3 天）

- 暂停公开 storage/share 上传入口或禁用 `server` source。
- 轮换前先完成代码修复；修复后轮换 MinIO 和相关签名 secret。
- 写会失败的安全测试：凭据不出现在响应、跨租户读写删拒绝、匿名 token 不能跨 share。
- 冻结新增旧式 route、`FailWithMessage`、service setter、无 ctx repository 方法。

### Phase 1：存储与认证边界重建（约 1–2 周）

- 引入 StorageObject + authorizer + share capability token。
- 统一上传 policy、完成校验、配额、独立内容域。
- 修复 token/session fail-open、密钥分离、登录/验证码限流。
- 这是任何公开发布前的硬门槛。

### Phase 2：一次性重做公开契约（约 1–2 周）

- 定义 API style guide、typed errors、HTTP status、ResourcePath。
- 发布干净 `/api/v1`，移除旧 action route；生成 OpenAPI/TS client。
- 修复分页与 batch partial-success 语义。

### Phase 3：数据模型与生命周期（约 2–4 周）

- 上版本化 migration，清理数据并建立唯一/外键约束。
- App operation、幂等、workspace lock/CAS、desired/observed reconcile。
- 通过故障注入验证创建/更新/删除的可恢复性。

### Phase 4：服务拆分与性能治理（约 2–4 周，可并行模块化推进）

- 用 use case 拆 AppService/AppManageService，构造函数完整注入。
- repository 全量 ctx 化，去危险指针缓存，修 N+1/batch。
- 审计 outbox/worker 与结构化可观测性。
- 拆前端巨型场景，清 `any`、生产 debug globals 和高风险 HTML sink。

### Phase 5：开放源码发布门禁

- P0/P1 验收清单全绿，安全回归与 migration upgrade 测试进入 CI。
- 生成 SBOM，发布制品签名/provenance；GitHub Actions 第三方 action 尽量 pin 到 commit SHA。
- 发布 threat model、架构决策记录、API versioning policy、数据库升级/备份手册。
- 对外许可证表述必须准确：当前仓库是 **BSL 1.1 source-available**，不是当前即 OSI open source。若目标是真正 OSI 开源，需要单独决定并切换到 OSI 批准许可证；代码重构不会改变许可证性质。

## 7. 发布阻断清单（Definition of Done）

以下任一未完成，建议阻断第一次公开发布：

- [ ] 所有 HTTP 响应、日志、示例中不存在 MinIO/对象存储长期凭据。
- [ ] 存储所有读写接口具有对象级授权，并通过跨租户负向测试。
- [ ] 公开分享 token 绑定 share、资源、操作和短 TTL，撤销可生效。
- [ ] 上传实际大小/MIME/内容经过服务端验证；主动内容使用独立域与安全响应头。
- [ ] 用户 App 运行时采用非 host 网络、最小权限和资源/egress 限制，恶意 App 隔离测试通过。
- [ ] token/session 持久化失败不再发放可用令牌；生产无硬编码 secret fallback。
- [ ] API v1 的 status/error/schema/route 规范统一，前端 client 从规范生成。
- [ ] 关键自然键由数据库唯一约束保证，生产不再在启动时 AutoMigrate/ALTER。
- [ ] create/update/delete 具备 operation ID、幂等、并发控制和失败恢复/对账。
- [ ] 关键安全、生命周期、migration、契约测试进入 CI。
- [ ] README 与发布文案准确使用 source-available/BSL，或已经完成 OSI 许可证切换。

## 8. 不建议采用的重构方式

- 不要先按“每个文件不超过 N 行”机械拆分；那只会把耦合移动到更多文件。
- 不要为了未来用户保留双路由、双错误模型、旧字段兼容；当前窗口的价值就是一次性清掉错误契约。
- 不要先把所有 setter 换成超长 constructor 就算完成；需要同时拆职责和明确 required/optional dependency。
- 不要用更多重试掩盖跨系统不一致；没有幂等和 operation state 的重试会重复副作用。
- 不要仅补 happy-path 单测；本项目当前最缺的是权限矩阵、并发、失败注入和升级路径测试。

## 9. 建议的首批任务拆分

为了后续逐个处理，可先建立以下 epic：

1. `SEC-STORAGE-CREDENTIALS`：移除 SDKConfig/长期凭据，轮换密钥。
2. `SEC-STORAGE-AUTHZ`：StorageObject 与对象级授权。
3. `SEC-PUBLIC-SHARE`：share capability token 与撤销。
4. `SEC-UPLOAD-CONTENT`：policy、quota、MIME/size/stat、独立内容域。
5. `AUTH-SESSION-HARDENING`：session fail-closed、refresh rotation、rate limit。
6. `API-V1-CONTRACT`：状态码、错误码、路由、OpenAPI 和 TS client。
7. `RESOURCE-IDENTITY`：ResourcePath 统一值对象。
8. `DB-MIGRATIONS-CONSTRAINTS`：显式 migration 与关键唯一约束。
9. `APP-OPERATION-SAGA`：生命周期、幂等、reconcile。
10. `WORKSPACE-CONCURRENCY`：lease/CAS/版本冲突。
11. `SERVICE-COMPOSITION`：去二阶段 setter，按用例拆 God Service。
12. `IO-AND-PERFORMANCE`：repository ctx、N+1、batch、缓存。
13. `AUDIT-OBSERVABILITY`：outbox/worker、指标、readiness、redaction。
14. `WEB-SECURITY-BOUNDARIES`：token cookie、sanitizer/CSP、类型与组件拆分。
15. `OPEN-RELEASE-GATES`：SBOM、签名、文档、许可证与发布检查。

这些 epic 的优先级不是按“最容易改”排序，而是按**先消除可被利用的风险，再冻结正确契约，再治理内部结构**排序。
