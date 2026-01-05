# 循环依赖分析：是否可以在顶层导入组件？

## 当前架构

### 文件结构
```
widgetRegistry/
├── WidgetComponentFactory.ts  (类定义)
├── factory.ts                 (创建实例，不导入组件)
└── index.ts                   (注册组件，动态导入 FormWidget/TableWidget)
```

### 依赖关系

```
factory.ts
  ↓ (创建实例)
widgetComponentFactory (实例)
  ↑ (导入)
index.ts
  ↓ (重新导出)
FormWidget.vue / TableWidget.vue
  ↑ (导入 widgetComponentFactory)
index.ts (循环！)
```

## 问题分析

### 如果我们在 index.ts 顶层导入 FormWidget 和 TableWidget

```typescript
// index.ts
import { widgetComponentFactory } from './factory'
import FormWidget from '@/architecture/presentation/widgets/FormWidget.vue'  // ❌ 顶层导入
import TableWidget from '@/architecture/presentation/widgets/TableWidget.vue'  // ❌ 顶层导入

// 注册组件
widgetComponentFactory.registerRequestComponent(WidgetType.FORM, FormWidget)
widgetComponentFactory.registerRequestComponent(WidgetType.TABLE, TableWidget)
```

### 模块加载顺序（ES 模块）

当 `index.ts` 被导入时：

1. **开始执行 index.ts**
   - 第1行：`import { widgetComponentFactory } from './factory'`
   - 此时需要加载 `factory.ts`
   
2. **加载 factory.ts**
   - 创建 `widgetComponentFactory` 实例
   - 返回实例给 `index.ts`
   
3. **继续执行 index.ts**
   - 第2行：`import FormWidget from '...FormWidget.vue'`
   - 此时需要加载 `FormWidget.vue`
   
4. **加载 FormWidget.vue**
   - 执行到：`import { widgetComponentFactory } from '@/architecture/infrastructure/widgetRegistry'`
   - 这个路径指向 `index.ts`（通过 package.json 的 exports 或默认导出）
   - **问题：此时 index.ts 还在执行中，还没有执行到导出语句！**
   - 结果：`ReferenceError: Cannot access 'FormWidget' before initialization`

## 为什么动态 import() 可以工作？

### 使用动态 import()

```typescript
// index.ts
import { widgetComponentFactory } from './factory'

export async function initializeWidgetComponentFactory(): Promise<void> {
  // 在函数内部动态导入
  const { default: FormWidget } = await import('...FormWidget.vue')
  const { default: TableWidget } = await import('...TableWidget.vue')
  
  // 注册组件
  widgetComponentFactory.registerRequestComponent(WidgetType.FORM, FormWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.TABLE, TableWidget)
}
```

### 为什么可以工作？

1. **函数调用时机**：
   - `initializeWidgetComponentFactory()` 在模块加载完成后才被调用
   - 此时 `index.ts` 已经执行完毕，`widgetComponentFactory` 已经导出

2. **异步加载**：
   - `import()` 是异步的，不会阻塞模块的初始化
   - 在函数执行时，所有模块都已经加载完成

## 可能的解决方案

### 方案 1：改变导入路径（❌ 不推荐）

让 `FormWidget` 和 `TableWidget` 直接从 `factory.ts` 导入：

```typescript
// FormWidget.vue
import { widgetComponentFactory } from '@/architecture/infrastructure/widgetRegistry/factory'
```

**问题**：
- 破坏了封装性（暴露了内部文件结构）
- 如果以后重构，需要修改所有导入路径

### 方案 2：延迟注册（✅ 当前方案）

使用动态 `import()` 延迟加载组件：

**优点**：
- 保持导入路径不变
- 打破循环依赖
- 代码清晰

**缺点**：
- 初始化是异步的
- 需要确保初始化完成才能使用组件

### 方案 3：改变架构（✅ 可行但需要重构）

将 `widgetComponentFactory` 的创建和组件注册分离：

```typescript
// factory.ts - 创建实例
export const widgetComponentFactory = new WidgetComponentFactory()

// register.ts - 注册组件（不导出 factory）
import FormWidget from '...'
import TableWidget from '...'
import { widgetComponentFactory } from './factory'

export function registerComponents() {
  widgetComponentFactory.registerRequestComponent(WidgetType.FORM, FormWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.TABLE, TableWidget)
}

// index.ts - 统一导出
export { widgetComponentFactory } from './factory'
export { registerComponents } from './register'

// 在应用启动时调用
registerComponents()
```

**优点**：
- 可以在顶层导入组件
- 初始化是同步的
- 代码更清晰

**缺点**：
- 需要重构现有代码
- 需要手动调用注册函数

### 方案 4：使用依赖注入（✅ 最优雅但需要大重构）

```typescript
// factory.ts
export const widgetComponentFactory = new WidgetComponentFactory()

// FormWidget.vue - 不导入 factory，通过 props 传入
const props = defineProps<{
  factory?: WidgetComponentFactory
}>()

// 或者使用 provide/inject
```

**优点**：
- 完全消除循环依赖
- 更好的可测试性
- 符合依赖倒置原则

**缺点**：
- 需要大量重构
- 改变组件使用方式

## 结论

### 是否可以在顶层导入？

**理论上可以，但需要满足以下条件之一：**

1. ✅ **改变导入路径**：让组件直接从 `factory.ts` 导入（不推荐）
2. ✅ **改变架构**：分离创建和注册逻辑（需要重构）
3. ✅ **使用依赖注入**：通过 props 或 provide/inject 传入 factory（需要大重构）

### 当前方案（动态 import）的优势

1. **最小改动**：不需要修改现有组件代码
2. **保持封装**：导入路径不变
3. **简单清晰**：代码逻辑直观

### 建议

**保持当前方案（动态 import）**，因为：
- 已经解决了循环依赖问题
- 代码清晰易懂
- 不需要大量重构
- 性能影响可忽略（组件注册在应用启动时完成）

如果未来需要优化，可以考虑方案 3（分离注册逻辑），但需要评估重构成本。

