/**
 * WidgetComponentFactory 初始化
 * 🔥 完全新增，不依赖旧代码
 * 
 * 功能：
 * - 导入所有组件
 * - 注册到工厂
 */

import { widgetComponentFactory } from './WidgetComponentFactory'
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
import FormWidget from '@/architecture/presentation/widgets/FormWidget.vue'
import TableWidget from '@/architecture/presentation/widgets/TableWidget.vue'
import UserWidget from '@/architecture/presentation/widgets/UserWidget.vue'
import LinkWidget from '@/architecture/presentation/widgets/LinkWidget.vue'
import ProgressWidget from '@/architecture/presentation/widgets/ProgressWidget.vue'

/**
 * 初始化组件工厂
 * 注册所有组件到工厂
 */
export function initializeWidgetComponentFactory(): void {
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

// 自动初始化
initializeWidgetComponentFactory()

// 导出工厂实例
export { widgetComponentFactory }

