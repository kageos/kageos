# Shared

`src/shared` 用来放跨业务域复用、但不属于 workspace 专属架构层的代码。

当前约定：

- `src/architecture`：workspace / workstation / form / table / chart 等新架构主链路
- `src/shared`：新旧页面都会复用的通用组件
- `src/components`：仍待迁移或仅旧页面使用的历史组件

迁移原则：

- 如果组件被 `architecture/*` 和 `views/*` 同时依赖，优先迁到 `src/shared`
- 如果组件只服务 workspace 域，优先继续内聚到 `src/architecture`
- 不再新增“仅目录老、实际又被新链路复用”的组件到 `src/components`
