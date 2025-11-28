# 新架构目录

这是新架构的根目录，采用分层架构设计，遵循依赖倒置原则。

## 目录结构

```
architecture/
├── presentation/      # Presentation Layer (UI 组件层)
├── application/      # Application Layer (应用层)
├── domain/           # Domain Layer (领域层)
└── infrastructure/   # Infrastructure Layer (基础设施层)
```

## 架构说明

详细说明请参考：
- `blueprint/新架构设计方案.md` - 完整架构设计文档
- `blueprint/重构方案.md` - 重构方案和迁移计划

## 设计原则

1. **依赖倒置**：所有层都依赖接口，不依赖具体实现
2. **事件驱动**：使用事件总线实现组件间通信
3. **单一数据源**：所有状态都通过 StateManager 管理
4. **渐进式重构**：创建新目录和文件，不修改旧代码

## 当前状态

- ✅ 目录结构已创建
- ⬜ Phase 1：基础设施层（进行中）
- ⬜ Phase 2：领域层
- ⬜ Phase 3：应用层
- ⬜ Phase 4：展示层

