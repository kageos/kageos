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

跨业务域复用、但不属于 workspace 专属链路的组件，统一放到 `src/shared`。

## 架构说明

详细说明请参考：
- `blueprint/新架构设计方案.md` - 完整架构设计文档
- `blueprint/重构方案.md` - 重构方案和迁移计划

## 设计原则

1. **分层组织**：页面渲染、流程编排、领域逻辑、基础设施分开
2. **依赖倒置**：Application/Domain 优先依赖接口，不直接依赖实现
3. **事件驱动**：使用事件总线实现页面与运行时解耦
4. **渐进收边界**：主页面在这里演进，但允许复用 `src/core`、`src/shared`、`src/utils`

## 当前状态

- ✅ 工作空间、工作台、表单/表格/图表等主页面已在这里运行
- ✅ `application/domain/infrastructure/presentation` 四层目录已经落地
- ✅ 架构内类型入口已统一收口到 `architecture/domain/types`
- ⏳ 当前仍会复用 `src/core`、`src/shared`、`src/utils` 的稳定公共能力
- ⏳ 当前目标是持续收边界，而不是再做一轮“大迁移”
