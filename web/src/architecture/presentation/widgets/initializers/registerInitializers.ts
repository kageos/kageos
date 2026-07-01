/**
 * 注册所有 Widget 初始化器
 * 
 * 🔥 在应用启动时统一注册，避免在组件中重复注册
 */

import { widgetInitializerRegistry } from './WidgetInitializerRegistry'
import { SelectWidgetInitializer } from './SelectWidgetInitializer'
import { MultiSelectWidgetInitializer } from './MultiSelectWidgetInitializer'
import { FormWidgetInitializer } from './FormWidgetInitializer'
import { TableWidgetInitializer } from './TableWidgetInitializer'

/**
 * 注册所有 Widget 初始化器
 */
export function registerWidgetInitializers(): void {
  // 注册 SelectWidget 初始化器
  widgetInitializerRegistry.register('select', new SelectWidgetInitializer())
  
  // 注册 MultiSelectWidget 初始化器
  widgetInitializerRegistry.register('multiselect', new MultiSelectWidgetInitializer())
  
  // 注册 FormWidget 初始化器（处理嵌套结构）
  widgetInitializerRegistry.register('form', new FormWidgetInitializer())
  
  // 注册 TableWidget 初始化器（处理嵌套结构）
  widgetInitializerRegistry.register('table', new TableWidgetInitializer())
}
