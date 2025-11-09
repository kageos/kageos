/**
 * WidgetComponentFactory 初始化
 * 🔥 完全新增，不依赖旧代码
 * 
 * 功能：
 * - 导入所有组件
 * - 注册到工厂
 */

import { widgetComponentFactory } from './WidgetComponentFactory'
import { WidgetType } from '../constants/widget'

// 导入组件（按需导入，避免循环依赖）
import InputWidget from '../widgets-v2/components/InputWidget.vue'
import NumberWidget from '../widgets-v2/components/NumberWidget.vue'
import FloatWidget from '../widgets-v2/components/FloatWidget.vue'
import TextAreaWidget from '../widgets-v2/components/TextAreaWidget.vue'
import SwitchWidget from '../widgets-v2/components/SwitchWidget.vue'

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
  
  // 后续添加其他组件时，在这里注册
  // widgetComponentFactory.registerRequestComponent(WidgetType.SELECT, SelectWidget)
  // widgetComponentFactory.registerRequestComponent(WidgetType.TABLE, TableWidget)
  // ...
}

// 自动初始化
initializeWidgetComponentFactory()

// 导出工厂实例
export { widgetComponentFactory }

