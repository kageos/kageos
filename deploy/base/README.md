# 部署共享资源（Canonical Base）

本目录存放 **dev / prod 共享** 的部署与构建资源。

原则：

- **共享的** Dockerfile、基础镜像启动脚本、Nginx 模板、初始化 SQL、通用脚本放这里。
- **环境入口** 不放这里；本地开发看 `deploy/dev/`，线上部署看 `deploy/prod/`。
- **业务源码** 不迁入本目录；这里只收敛交付与运行资源。

当前主线内容：

- `images/app-base/`：用户应用基础镜像的 canonical Dockerfile 与启动脚本（默认 tag：`kagebase:latest`）
- `images/web/`：Web 前端镜像 Dockerfile 与 Nginx 配置
- `images/dev-server/`：本地开发用镜像
- `infra/mysql/`：MySQL 初始化 SQL
- `infra/nats/`：NATS 配置
- `scripts/`：共享构建脚本与构建辅助脚本（例如 app-base 构建、APT mirror 配置）

新增和维护时，请只修改 `deploy/base/` 下的 canonical 资源。


| 字段名 | 含义 | 值示例 | 备注 |
  |---|---|---|---|
| id | 主键 ID | 12345 | 自增主键 |
| username | 用户名 | zhangsan | 当前是调用方传入的用户名 |
| source_platform | 来源平台 | kate-mcp | 表里有这个字段；走 MCP tool 时当前代码基本写死为 kate-mcp，走 OpenAPI 时可由调用方自定义传入 |
| call_method | 调用方式 | kate-mcp | 当前实现里通常和 source_platform 一样，区分度不高 |
| call_type | 调用类型 | 1 | 枚举值，见下方说明 |
| user_question | 用户原始问题 | 怎么接入 KSearch SDK？ | 原始提问 |
| user_intent | 澄清后的用户意图 | 在 Java/Spring Boot 项目中如何接入 KSearch SDK？ | 更适合召回的改写问题 |
| call_collection_id | 命中的知识集合 ID 列表 | ["101","205"] | JSON 数组字符串，不是单值 |
| call_knowledge_id | 命中的知识切片 ID | {"SKILLS":["11"],"RPC":["22","23"]} | JSON 对象字符串，按知识类型分组 |
| is_badcase | 是否 badcase | false | 用户后续点踩后会变成 true |
| is_goodcase | 是否 goodcase | true | 用户后续点赞后会变成 true |
| deleted | 逻辑删除标记 | false | 正常数据一般是 false |
| query_time | 查询时间 | 1713255600000 | 毫秒时间戳 |
