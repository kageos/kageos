/**
 * WidgetComponentFactory 初始化
 * 🔥 完全新增，不依赖旧代码
 * 
 * 功能：
 * - 导入所有组件
 * - 注册到工厂
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
import LinkWidget from '@/architecture/presentation/widgets/LinkWidget.vue'
import ProgressWidget from '@/architecture/presentation/widgets/ProgressWidget.vue'

/**
 * 初始化组件工厂
 * 注册所有组件到工厂
 * 
 * 🔥 注意：由于 FormWidget 和 TableWidget 都导入了 widgetComponentFactory，
 * 如果在顶层导入会导致循环依赖，所以使用动态 import 延迟加载
 */
export async function initializeWidgetComponentFactory(): Promise<void> {
  // 🔥 动态导入容器组件，避免循环依赖
  // FormWidget 和 TableWidget 都导入了 widgetComponentFactory，如果在顶层导入会导致循环依赖
  const { default: FormWidget } = await import('@/architecture/presentation/widgets/FormWidget.vue')
  const { default: TableWidget } = await import('@/architecture/presentation/widgets/TableWidget.vue')
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
  
  // 容器组件
  widgetComponentFactory.registerRequestComponent(WidgetType.FORM, FormWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.TABLE, TableWidget)
  widgetComponentFactory.registerRequestComponent(WidgetType.USER, UserWidget)
  
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
  
  // 后续添加其他组件时，在这里注册
  // ...
}

// 🔥 自动初始化（异步，避免循环依赖）
// 注意：由于使用了动态 import，初始化是异步的
// 但组件注册应该在应用启动时完成，所以这里使用立即执行的异步函数
let initializationPromise: Promise<void> | null = null

export function ensureInitialized(): Promise<void> {
  if (!initializationPromise) {
    initializationPromise = initializeWidgetComponentFactory()
  }
  return initializationPromise
}

// 立即开始初始化
ensureInitialized().catch(err => {
  console.error('[WidgetComponentFactory] 初始化失败', err)
})

// 重新导出工厂实例（从 factory.ts 导入）
export { widgetComponentFactory } from './factory'

