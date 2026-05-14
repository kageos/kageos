# Runtime 运行时底座

`src/architecture/runtime/` 是当前前端线上链路复用的稳定基础层。

主业务页面和流程已经放到 `src/architecture/`，这些底层能力统一由 `runtime` 提供：

- 状态提取与提交数据组装
- Widget 运行时默认值与校验
- 共享常量、管理器、校验引擎
- 运行时类型门面

## 目录结构

```text
runtime/
├── constants/       # 共享常量
├── managers/        # 运行期管理器
├── stores/       # 表单/响应数据存储与提取器
├── types/           # 运行时类型门面（真实定义已收口到 src/architecture/domain/types）
├── utils/           # 基础工具函数
├── validation/      # 通用校验能力
└── widgetRuntime/   # Widget 默认值 / 校验 / 动态默认值等运行时能力
```

## 当前职责

### 1. 运行时类型门面

- [field.ts](./types/field.ts)：对外保留运行时类型入口
- 真实字段类型定义已经收口到 `src/architecture/domain/types/field.ts`

### 2. 表单数据与提取

- `stores/formData.ts`：表单数据存储
- `stores/extractors/*`：字段提取器
- `stores/extractors/FieldExtractorRegistry.ts`：提取策略注册表

### 3. Widget 运行时

- `widgetRuntime/defaultValue.ts`：默认值处理
- `widgetRuntime/dynamicDefaultValue.ts`：动态默认值（如 `Me()`）
- `widgetRuntime/validation.ts`：Widget 级校验辅助

### 4. 通用底座

- `constants/`：Widget/Event 等常量
- `utils/`：日志、通用工具
- `validation/`：通用校验引擎
- `managers/`：运行期管理器

## 边界说明

1. `runtime` 可以被 `application`、`domain`、`infrastructure`、`presentation` 复用。
2. `runtime` 不负责页面编排和业务流程。
3. Widget Vue 组件和主页面视图不再放在这里，它们位于 `src/architecture/presentation/`。
4. 如果某段能力与 UI 无关、但又不属于单个业务域，优先考虑放到 `runtime`。

## 什么时候往 runtime 里放代码

- 放这里：
  - 通用提取器
  - 通用校验能力
  - Widget 运行时纯逻辑
  - 稳定共享常量和工具

- 不放这里：
  - 页面组件
  - 工作空间/工作台业务流程
  - 具体业务域规则
  - 只能被单一页面使用的临时逻辑

## 和 architecture 的关系

- `architecture/`：主业务页面、流程编排、领域逻辑、基础设施实现
- `architecture/runtime/`：被这些层复用的稳定底座

`runtime` 是架构内的公共底座，不承载页面或流程编排。
