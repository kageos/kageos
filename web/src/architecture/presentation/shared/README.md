# Shared

`src/architecture/presentation/shared` 用来放跨业务域复用、但不属于单一 feature 的展示层代码。

当前约定：

- `src/architecture`：workspace / workstation / form / table / chart 等统一架构主链路
- `src/architecture/presentation/features`：认证、用户、组织、消息等业务入口
- `src/architecture/presentation/shared`：架构主链路和业务入口都会复用的展示组件

归属原则：

- 如果组件被多个 feature 或架构主链路复用，优先放到 `src/architecture/presentation/shared`
- 如果组件只服务 workspace 域，优先内聚到 `src/architecture/presentation`
- 不再新增“顶层目录老、实际又被架构主链路复用”的组件
