# NATS 组织约定

这份文档定义仓库内 NATS 调用的组织方式，目标是统一连接生命周期、主题注册、消息解码和业务处理的分层。

## 分层规则

### 1. Server 持有连接和订阅

每个服务自己的 `Server` 负责：

- 初始化 `nats.Conn`
- 注册订阅
- 保存 `subscriptions`
- 在 `Stop()` 中统一 `Unsubscribe()` 和 `Close()`

不要把 NATS 连接藏在局部函数里，也不要让业务 service 自己负责订阅生命周期。

### 2. Router 只做 subject 装配

`nats_router.go` 只负责：

- `subject -> handler` 绑定
- queue group 注册
- 订阅对象回收登记

不要在 router 里写 JSON 反序列化和业务逻辑。

### 3. Handler 只做 transport 适配

handler 负责：

- 解析 `nats.Msg`
- 读取 header
- 反序列化 body
- 调用下层 service
- request-reply 场景下回包

handler 不负责真正的业务决策。

### 4. Service 只做业务

service 负责：

- 业务规则
- 仓库访问
- 外部调用编排

如果 service 需要向别的服务发 NATS 命令，优先拆成明确的 publisher / query client，而不是直接在业务 service 里散写 `Publish`。

## 推荐结构

### 监听侧

```text
server
  -> initNATS / subscribeNATS / unsubscribeNATS
  -> nats_router.go
  -> handler
  -> service
```

### 调用侧

```text
service
  -> publisher / query client
  -> msgx / natsx
```

## 主题和协议

- 主题真值统一放在 `pkg/subjects`
- 基础 request-reply 原语统一放在 `pkg/msgx`
- 某条业务链自己的 subject / header / request 语义放在领域 transport 包里，例如 `pkg/appinvoke`
- 固定主题用 `const`
- 动态主题用 `BuildXxxSubject()`
- 新服务连接 NATS 时优先使用 `natsx.ConnectNamed(url, serviceName)`

## 当前参考实现

- `app-runtime`：路由注册基准实现
- `api-gateway`：Server 持有连接/订阅 + token command handler
- `hr-server`：publisher / router / handler / consumer service 分层
