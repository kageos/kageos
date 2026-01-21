/**
 * WidgetComponentFactory 初始化
 * 
 * ============================================
 * 📋 功能说明
 * ============================================
 * 
 * - 导入所有组件
 * - 注册到工厂
 * 
 * ============================================
 * ⚠️ 重要：循环依赖处理
 * ============================================
 * 
 * 🔥 问题：为什么 FormWidget 和 TableWidget 使用动态 import？
 * 
 * 原因：
 *   - FormWidget.vue 和 TableWidget.vue 都导入了 widgetComponentFactory
 *   - 如果在这里顶层导入它们，会导致循环依赖：
 *     index.ts → FormWidget → widgetRegistry → index.ts
 * 
 * 解决方案：
 *   1. widgetComponentFactory 在 factory.ts 中创建（不依赖组件）
 *   2. 这里从 factory.ts 导入 widgetComponentFactory
 *   3. FormWidget 和 TableWidget 使用动态 import() 延迟加载
 *   4. 在 initializeWidgetComponentFactory() 函数内部动态导入并注册
 * 
 * 依赖关系：
 *   factory.ts (创建实例)
 *     ↓
 *   index.ts (导入实例，注册组件)
 *     ↓ (动态 import)
 *   FormWidget.vue / TableWidget.vue (使用实例)
 * 
 * ============================================
 * 🤔 是否可以在顶层导入？
 * ============================================
 * 
 * **理论上可以，但需要满足以下条件之一：**
 * 
 * 1. ✅ **改变导入路径**（不推荐）：
 *    - 让 FormWidget 和 TableWidget 直接从 factory.ts 导入
 *    - 问题：破坏了封装性，暴露了内部文件结构
 * 
 * 2. ✅ **改变架构**（需要重构）：
 *    - 将组件注册逻辑分离到单独的 register.ts
 *    - 在应用启动时手动调用注册函数
 *    - 优点：可以在顶层导入，初始化是同步的
 *    - 缺点：需要重构现有代码
 * 
 * 3. ✅ **使用依赖注入**（需要大重构）：
 *    - 通过 props 或 provide/inject 传入 factory
 *    - 优点：完全消除循环依赖，更好的可测试性
 *    - 缺点：需要大量重构，改变组件使用方式
 * 
 * **当前方案（动态 import）的优势：**
 * - ✅ 最小改动：不需要修改现有组件代码
 * - ✅ 保持封装：导入路径不变
 * - ✅ 简单清晰：代码逻辑直观
 * - ✅ 性能影响可忽略：组件注册在应用启动时完成
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **不要在这里顶层导入 FormWidget 和 TableWidget**：
 *    - 必须在函数内部使用动态 import()
 *    - 否则会导致循环依赖错误："Cannot access 'FormWidget' before initialization"
 * 
 * 2. **初始化是异步的**：
 *    - initializeWidgetComponentFactory() 是异步函数
 *    - 使用 ensureInitialized() 确保初始化完成
 * 
 * 3. **其他组件可以正常导入**：
 *    - InputWidget、SelectWidget 等不依赖 widgetComponentFactory
 *    - 可以在顶层正常导入
 */

import { widgetComponentFactory } from './factory'
import { WidgetType } from '@/core/constants/widget'

// 导入组件（按需导入，避免循环依赖）
import InputWidget from '@/architecture/presentation/widgets/InputWidget.vue'
import NumberWidget from '@/architecture/presentation/widgets/NumberWidget.vue'
import FloatWidget from '@/architecture/presentation/widgets/FloatWidget.vue'
import TextAreaWidget from '@/architecture/presentation/widgets/TextAreaWidget.vue'
import SwitchWidget from '@/architecture/presentation/widgets/SwitchWidget.vue'
import SelectWidget from '@/architecture/presentation/widgets/SelectWidget.vue'
import MultiSelectWidget from '@/architecture/presentation/widgets/MultiSelectWidget.vue'
import CheckboxWidget from '@/architecture/presentation/widgets/CheckboxWidget.vue'
import RadioWidget from '@/architecture/presentation/widgets/RadioWidget.vue'
import TextWidget from '@/architecture/presentation/widgets/TextWidget.vue'
import FilesWidget from '@/architecture/presentation/widgets/FilesWidget.vue'
import TimestampWidget from '@/architecture/presentation/widgets/TimestampWidget.vue'
import SliderWidget from '@/architecture/presentation/widgets/SliderWidget.vue'
import RateWidget from '@/architecture/presentation/widgets/RateWidget.vue'
import ColorWidget from '@/architecture/presentation/widgets/ColorWidget.vue'
import RichTextWidget from '@/architecture/presentation/widgets/RichTextWidget.vue'
// 🔥 延迟导入容器组件，避免循环依赖
// FormWidget 和 TableWidget 都导入了 widgetComponentFactory，会导致循环依赖
// 解决方案：在函数内部动态导入，而不是在模块顶层导入
import UserWidget from '@/architecture/presentation/widgets/UserWidget.vue'
import UsersWidget from '@/architecture/presentation/widgets/UsersWidget.vue'
import LinkWidget from '@/architecture/presentation/widgets/LinkWidget.vue'
import ProgressWidget from '@/architecture/presentation/widgets/ProgressWidget.vue'

/**
 * 同步注册基础组件（不依赖 widgetComponentFactory 的组件）
 * 这些组件可以在模块加载时立即注册，无需等待
 */
function registerBasicComponents(): void {
  // 注册请求参数组件
  widgetComponentFactory.registerRequestComponent(WidgetType.INPUT, InputWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.TEXT, InputWidget)  // text 别名
  widgetComponentFactory.registerRequestComponent(WidgetType.ID, InputWidget)  // ID 字段
  
  widgetComponentFactory.registerRequestComponent(WidgetType.NUMBER, NumberWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.FLOAT, FloatWidget)
  
  widgetComponentFactory.registerRequestComponent(WidgetType.TEXT_AREA, TextAreaWidget)
  widgetComponentFactory.registerRequestComponent('textarea', TextAreaWidget)  // 兼容旧命名
  
  widgetComponentFactory.registerRequestComponent(WidgetType.SWITCH, SwitchWidget)
  
  widgetComponentFactory.registerRequestComponent(WidgetType.SELECT, SelectWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.MULTI_SELECT, MultiSelectWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.CHECKBOX, CheckboxWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.RADIO, RadioWidget)
  
  widgetComponentFactory.registerRequestComponent(WidgetType.TIMESTAMP, TimestampWidget)
  
  widgetComponentFactory.registerRequestComponent(WidgetType.SLIDER, SliderWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.RATE, RateWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.COLOR, ColorWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.RICH_TEXT, RichTextWidget)
  
  widgetComponentFactory.registerRequestComponent(WidgetType.FILES, FilesWidget)
  
  widgetComponentFactory.registerRequestComponent(WidgetType.USER, UserWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.USERS, UsersWidget)
  
  // 链接组件
  widgetComponentFactory.registerRequestComponent(WidgetType.LINK, LinkWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.LINK, LinkWidget)
  
  // 进度条组件（主要用于响应模式展示）
  widgetComponentFactory.registerRequestComponent(WidgetType.PROGRESS, ProgressWidget)
  widgetComponentFactory.registerResponseComponent(WidgetType.PROGRESS, ProgressWidget)
  
  // 响应参数组件（Text 主要用于响应模式）
  widgetComponentFactory.registerResponseComponent(WidgetType.TEXT, TextWidget)
  // Text 也可以用于请求参数（详情模式等场景）
  widgetComponentFactory.registerRequestComponent(WidgetType.TEXT, TextWidget)
}

/**
 * 异步注册容器组件（依赖 widgetComponentFactory 的组件）
 * 这些组件需要动态导入，避免循环依赖
 */
async function registerContainerComponents(): Promise<void> {
  // 🔥 动态导入容器组件，避免循环依赖
  // FormWidget 和 TableWidget 都导入了 widgetComponentFactory，如果在顶层导入会导致循环依赖
  const { default: FormWidget } = await import('@/architecture/presentation/widgets/FormWidget.vue')
  const { default: TableWidget } = await import('@/architecture/presentation/widgets/TableWidget.vue')
  
  // 容器组件
  widgetComponentFactory.registerRequestComponent(WidgetType.FORM, FormWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.TABLE, TableWidget)
}

/**
 * 初始化组件工厂
 * 注册所有组件到工厂
 * 
 * 🔥 优化：分两步注册
 * 1. 同步注册基础组件（立即完成，无需等待）
 * 2. 异步注册容器组件（需要动态导入，避免循环依赖）
 * 
 * 这样大部分组件在模块加载时就已注册，只有容器组件需要异步注册
 */
export async function initializeWidgetComponentFactory(): Promise<void> {
  // 第一步：同步注册基础组件（立即完成）
  registerBasicComponents()
  
  // 第二步：异步注册容器组件（需要动态导入）
  await registerContainerComponents()
}

// 🔥 立即同步注册基础组件（在模块加载时完成）
// 这样大部分组件在应用启动前就已经注册好了，避免"组件未找到"的闪现
registerBasicComponents()

// 🔥 异步初始化容器组件（需要动态导入，避免循环依赖）
// 注意：由于 FormWidget 和 TableWidget 使用了动态 import，这部分是异步的
// 但基础组件已经同步注册完成，所以不会影响大部分组件的使用
let initializationPromise: Promise<void> | null = null

export function ensureInitialized(): Promise<void> {
  if (!initializationPromise) {
    initializationPromise = registerContainerComponents()
  }
  return initializationPromise
}

// 立即开始异步初始化容器组件
ensureInitialized().catch(err => {
  console.error('[WidgetComponentFactory] 容器组件初始化失败', err)
})

// 重新导出工厂实例（从 factory.ts 导入）
export { widgetComponentFactory } from './factory'

