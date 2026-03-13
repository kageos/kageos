# AI Agent OS - Frontend AI Developer Guide (aiReadme.md)

> **这是一份专门为 AI Developer 准备的项目导读大纲**。
> 本项目由一位具有较强后端思维的开发者构建，因此在代码组织上大量借鉴了后端工程的架构模式（如四层架构、DDD、SOLID、各种设计模式），但在前端 UI 表现层面相对朴素。后续的主要目标是从**专业前端角度优化交互与视觉**，同时在现有严谨结构下开发新功能。

## 一、 项目背景与定调

- **定位**：AI 原生的“一站式”应用构建与运行平台（支持自然语言生成应用、闪电克隆、私有化部署、统一登录等）。
- **技术栈**：Vue 3 (Composition API) + TypeScript + Vite + Pinia + Vue Router + Element Plus。
- **页面设计感**：目前由后端主导开发，样式多采用现成的 Element Plus 默认风格结合简单 CSS，**界面较为朴素，动态交互较少**，是未来 UI/UX 优化的重点。

## 二、 核心架构设计 (后端思维主导)

项目在经历了重构后，目前主要采用了**严谨的领域驱动（DDD）与四层架构**（位于 `src/architecture` 目录下）：

1. **Presentation Layer (表示层)**
   - 路径：`src/architecture/presentation/`
   - 职责：只负责 Vue 组件渲染（如 `WorkspaceView.vue`、`FormView.vue`），不处理核心业务逻辑。数据和状态来自 `StateManager`，操作通过触发 `EventBus` 或直接调用 `ApplicationService` 完成。
2. **Application Layer (应用层)**
   - 路径：`src/architecture/application/`
   - 职责：负责用例的流程编排和调度（如表单提交、工作空间初始化调度）。
3. **Domain Layer (领域层)**
   - 路径：`src/architecture/domain/`
   - 职责：包含业务核心逻辑（如 `FormDomainService`、`WorkspaceDomainService`）以及**抽象接口定义**（`IStateManager`, `IEventBus`, `IApiClient`），贯彻了**依赖倒置（DIP）**原则。
4. **Infrastructure Layer (基础设施层)**
   - 路径：`src/architecture/infrastructure/`
   - 职责：技术细节的实现，包括网络请求封装（Api Client）、事件总线实现（Event Bus）、具体的状态管理器（基于 Pinia 初始化响应式），和依赖注入的工厂（Service Factory）。

### 主要的设计模式

- **策略模式 (Strategy)**：负责不同字段值提取逻辑（如 `FieldExtractorRegistry`）。
- **工厂模式 (Factory)**：根据类型渲染不同的输入表单或表格组件（`WidgetComponentFactory`），并且用于 Service 的依赖注入。
- **观察者模式 (Observer)**：各组件、Service 直接利用自定义的 EventBus 松耦合通讯。

## 三、 核心功能运行机制与数据流向

### 1. 动态表单与表格渲染系统 (`src/core/widgets-v2` 与 `src/core/renderers-v2`)

这是系统的核心亮点，基于后端动态配置的 JSON 结构来渲染表单或表格：

- **基于数据驱动 UI**：后端返回元数据（FieldConfig等），前端根据 `widget.type` 交由 `WidgetComponentFactory` 实例化对应的 Widget 组件。
- **多态渲染机制**：每个 Widget （如 `InputWidget`, `SelectWidget`, `FilesWidget`）内部都必须自定处理不同的渲染模式（`mode`）：
  - `edit`: 表单填写时的可输入状态。
  - `response`: 表单提交后的结构化只读展示。
  - `table-cell`: 列表栏里的简要展示。
  - `search`: 作为表格过滤条件的输入状态。
  - `detail`: 抽屉或详情页内的完整内容展示（部分抽象不够完善，需要优化）。

### 2. 工作空间树形导航 (`WorkspaceView.vue`)

- 应用以类似操作系统的形式呈现：左侧为服务树（ServiceTree），涵盖应用内的文件夹（Directory）、表单/表格（Function）、文档（Docs）和讨论区（Board）。
- 右侧主区域根据选中的左侧节点，动态挂载对应的视图模块（`FormView`, `TableView`, `DocView`, `BoardView` 等）。工作台（Workstation AI会话）作为抽屉形式全局悬浮或右侧并排展示。

## 四、 核心目录与文件地图

```text
src/
├── architecture/           # 🌟 核心业务四层架构（新重构）
│   ├── application/        # 流程编排 (Service)
│   ├── domain/             # 核心逻辑与接口规范
│   ├── infrastructure/     # API实现、Store实现、EventBus、服务工厂
│   └── presentation/       # 视图组件 (Workspace、各类 View 和 对应的 Composables)
├── core/                   # 🌟 核心引擎层：与业务解耦的通用能力
│   ├── renderers-v2/       # 动态渲染引擎入口 (FormRenderer, TableRenderer)
│   ├── widgets-v2/         # 底层基础组件库 (Input, Select, Struct nested obj 等)
│   ├── factories-v2/       # Widget 组件注册工厂
│   ├── stores-v2/          # 渲染体系的状态和数据提取器 (Extractors：处理复杂的嵌套提取)
│   └── validation/         # 验证引擎
├── views/                  # 老架构入口与其他独立页面 (Home.vue, Login.vue 等)
├── router/                 # Vue Router 路由配置 (注意工作空间的泛路由 `/workspace/:user/:app/:path*`)
├── stores/                 # 全局基础状态 (Auth, Theme 等)
├── styles/                 # 全局/主题样式 (需要前端角度深度介入优化)
├── types/                  # TS 全局类型定义
└── utils/                  # 工具类函数 (权限、网络、路由防抖等)
```

## 五、 后续优化方向 (AI 介入重点)

既然当前是由后端开发者搭建的体系，你的主要优化维度应该是**提升用户体验 (UX)、增强视觉表现 (UI)** 以及 **优化不合理的前端心智负担**：

1. **界面美化与动效 (UI/UX)**
   - 现有的 `Home.vue` 与 `WorkspaceView.vue` 非常“干净”，缺少微交互、骨架屏的平滑过渡、以及深度定制的现代化组件（如 Glassmorphism, 细致的 hover 动效等）。
   - 需要引入前沿的 CSS 布局或动画库手段重塑页面美感，特别是官方 Landing Page (`Home.vue`) 和工作空间的操作反馈机制。
2. **完善“组件自治”的架构缝隙**
   - 当前在 `ARCHITECTURE_ANALYSIS.md` 中已知的设计痛点：
     - `TableRenderer` 在处理详情抽屉 (`renderForDetail`) 时，存在 `if/else` 硬编码，违反了依赖倒置原则。需要将其下沉到各个 Widget 基类中实现多态。
     - 组件数据的复制策略（`onCopy`）以及未来可能的导出逻辑，也需要由具体 `WidgetComponent` 自治提供。

3. **前端状态和事件的总线瘦身**
   - 当前四层架构在前端落地虽然严谨，但增加了非常高昂的心智负担（写一个状态需要改 Domain Interface -> Infrastructure 实现封装 -> Application Service -> Presentation 调用）。
   - 在开发新功能时，要评估是否可以利用现代 Vue3 Composables 模式（`useXxx`）进行轻量化替代，避免层层包裹；或者在保留当前架构的情况下，提炼通用代码模板加速开发。

## 六、 给 AI 的阅读建议

当收到具体修改需求时：

1. **如果是 UI 交互、重构单组件表现**：直接前往 `src/core/widgets-v2/` 和对应的 `src/architecture/presentation/views/`，仅涉及 `presentation` 层面，无需深入四层架构服务。
2. **如果是新增数据提交、网络拦截、表单交互逻辑**：从 `presentation/composables` 入手，找到它调用的 `ApplicationService`。
3. **如果是新增字段类型**：
   - 先在 `widgets-v2/components` 写 Vue 组件模板；
   - 在 `factories-v2` 注册进去；
   - 视层级复杂度在 `stores-v2/extractors/` 加入值的提取策略。

**牢记你的身份**：你是专业的**前端专家**，任务是在维护这个宏大的“领域驱动架构”正常运行的基础上，把它“打扮”成一个具备顶级互联网大厂质感、交互极度顺滑的专业化平台。

---

# UI 改版文档 v1

## 改版目标

- 筛选区对齐、统一节奏、紧凑但可读。
- 弹窗统一规格与内嵌表单布局。
- 表格与列表的密度、按钮间距、空态一致。
- 全站支持暗黑模式：新增样式均用 CSS 变量。

## 一、设计规范（最终定稿）

### 1. 筛选区（Search/Filter Bar）

- Label 宽度：96px（左对齐）。
- 控件宽度：基础 220px，最小 200px，允许自适应 1fr。
- 布局：grid，2~4 列响应式。
- 间距：行距 12px，列距 12px（紧凑）。
- 容器：
  - padding: 12px 16px
  - border-radius: 8px
  - border: 1px solid var(--el-border-color-lighter)
  - background: var(--el-bg-color)

### 2. 弹窗（Dialog）

- 宽度体系：sm=520、md=720、lg=920
- header/body/footer 统一：
  - header 16px 20px
  - body 16px 20px
  - footer 12px 20px
- 弹窗内搜索区统一复用“筛选区规范”。

### 3. 表格

- 行高：48px
- 表头背景：var(--el-fill-color-light)
- 操作列按钮间距：8px
- 表格外容器统一留白：padding 16px

### 4. 字体与间距

- 统一使用 src/styles/theme.scss 里的字体，不在 App.vue 再覆盖。
- 统一 spacing：4 / 8 / 12 / 16 / 24 / 32

### 暗黑模式约束

- 禁止写死浅色背景、边框、阴影。
- 所有颜色用 --el-* 或 --bg-* / --border-*。
- 新增阴影用已有 --box-shadow-*。

## 二、执行清单（工程落地）

### 1. 新增统一样式模块

建议新增 src/styles/ui-system.scss（或合并到已有主题），包含：

- .ui-filter-bar：筛选区容器
- .ui-filter-form：grid 布局
- .ui-filter-item：label/控件对齐
- .ui-dialog-sm/md/lg：弹窗宽度
- .ui-table：表格行高、表头背景

### 2. 统一筛选区组件

重点改动文件：

- src/components/TableSearchBar.vue
- src/components/Permission/PermissionManageList.vue
- src/components/Permission/PermissionRequestList.vue
- src/components/ChartRenderer.vue

改动方向：

- 由 el-form inline 改为 grid
- label 宽度统一为 96px
- 控件宽度统一为 220px（最小 200px）
- 统一容器样式（背景/边框/圆角/内边距）

### 3. 弹窗统一规范

重点文件：

- src/components/UserSelectorDialog.vue
- src/components/FunctionPathSelector.vue
- src/components/DocsPathSelector.vue
- src/components/LLMSelector.vue
- src/components/DepartmentSelectorDialog.vue
- src/views/Permission/RoleManagement.vue
- src/components/FormDialog.vue（当前已打开，建议对齐新规范）

改动方向：

- 标准化 width（sm/md/lg）
- header/body/footer padding 统一
- 弹窗内搜索区/表单复用筛选区规范

### 4. 表格规范

涉及页面中 el-table：

- src/components/Permission/PermissionManageList.vue
- src/views/Agent/LLMManagement.vue 等

改动方向：

- row-height: 48px
- header-bg: var(--el-fill-color-light)
- 操作按钮间距 8px

### 5. 全局字体一致性

- src/App.vue：移除系统字体覆盖，使用主题字体
- 统一字体引用到 theme.scss

## 三、交付验收标准

- 筛选区 label 对齐（左侧 96px），控件宽度一致。
- 多个筛选区页面整体密度一致，看起来“规整”。
- 所有弹窗视觉统一（宽度、padding、标题样式）。
- 暗黑模式下无“亮底/灰底突兀”区域。
- 表格行高一致，操作按钮排版整齐。
