# architecture 目录

这是当前前端主业务页面和流程的承载目录，采用分层架构组织代码。

## 目录结构

```
architecture/
├── presentation/      # Presentation Layer (UI 组件层)
├── application/      # Application Layer (应用层)
├── domain/           # Domain Layer (领域层)
└── infrastructure/   # Infrastructure Layer (基础设施层)
```

跨业务域复用、但不属于 workspace 专属链路的展示组件，统一放到 `src/architecture/presentation/shared`。

## 架构说明

详细说明请参考：
- `web/README.md`：前端整体架构、目录职责与开发约定
- 本目录下各层 README：application / domain / infrastructure / presentation 的边界说明

## 设计原则

1. **分层组织**：页面渲染、流程编排、领域逻辑、基础设施分开
2. **依赖倒置**：Application/Domain 优先依赖接口，不直接依赖实现
3. **事件驱动**：使用事件总线实现页面与运行时解耦
4. **统一架构根**：源码入口统一在 `src/architecture`，各层只暴露自己的职责边界

## 当前状态

- ✅ 工作空间、工作台、表单/表格/图表等主页面已在这里运行
- ✅ `application/domain/infrastructure/presentation` 四层目录已经落地
- ✅ 架构内类型入口已统一收口到 `architecture/domain/types`
- ✅ 历史 `runtime` 层已移除，公共能力按职责收口到 `domain/shared/infrastructure/presentation`
- ✅ 顶层源码目录不再保留新旧混合入口
