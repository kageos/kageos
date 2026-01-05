/**
 * 插件系统统一导出
 * 
 * 🔥 插件化扩展机制：支持外部注册 Widget 组件、提取器、初始化器
 */

export { widgetPluginRegistry, type WidgetPlugin } from './WidgetPlugin'
export {
  registerWidgetPlugin,
  registerWidgetPlugins,
  unregisterWidgetPlugin,
  getWidgetPlugin,
  getRegisteredPluginTypes
} from './pluginManager'

