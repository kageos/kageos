/**
 * 插件管理器
 * 
 * ============================================
 * 📋 需求说明
 * ============================================
 * 
 * 1. **插件化扩展**：
 *    - 支持外部注册自定义 Widget 组件
 *    - 支持注册自定义字段提取器
 *    - 支持注册自定义组件初始化器
 *    - 一次性注册所有相关组件（组件、提取器、初始化器）
 * 
 * 2. **统一注册入口**：
 *    - 提供 `registerWidgetPlugin` 统一注册插件
 *    - 自动注册到 WidgetComponentFactory、FieldExtractorRegistry、WidgetInitializerRegistry
 *    - 支持批量注册多个插件
 * 
 * 3. **插件管理**：
 *    - 支持查询已注册的插件
 *    - 支持取消注册插件
 *    - 插件类型唯一性检查（防止重复注册）
 * 
 * ============================================
 * 🎯 设计思路
 * ============================================
 * 
 * 1. **统一接口**：
 *    - 使用 `WidgetPlugin` 接口定义插件配置
 *    - 包含组件、提取器、初始化器等所有相关信息
 *    - 一次性注册，避免分散注册导致的不一致
 * 
 * 2. **协调注册**：
 *    - 自动注册到各个注册表（WidgetComponentFactory、FieldExtractorRegistry 等）
 *    - 确保所有相关组件都正确注册
 *    - 提供统一的错误处理和日志记录
 * 
 * 3. **扩展性**：
 *    - 支持第三方开发者扩展系统
 *    - 插件可以独立开发和维护
 *    - 不影响现有组件的使用
 * 
 * ============================================
 * 📝 使用场景
 * ============================================
 * 
 * 1. **注册自定义组件**：
 *    ```typescript
 *    registerWidgetPlugin({
 *      name: 'Custom Widget',
 *      widgetType: 'custom',
 *      requestComponent: CustomWidget,
 *      extractor: new CustomExtractor(),
 *      initializer: new CustomInitializer()
 *    })
 *    ```
 * 
 * 2. **批量注册插件**：
 *    ```typescript
 *    registerWidgetPlugins([
 *      { name: 'Widget 1', widgetType: 'widget1', requestComponent: Widget1 },
 *      { name: 'Widget 2', widgetType: 'widget2', requestComponent: Widget2 }
 *    ])
 *    ```
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **插件类型唯一性**：
 *    - 每个 `widgetType` 只能注册一次
 *    - 重复注册会抛出错误
 * 
 * 2. **注册时机**：
 *    - 插件应该在应用启动时（`main.ts`）注册
 *    - 确保在使用前已经注册
 * 
 * 3. **组件规范**：
 *    - 自定义组件必须遵循 `WidgetComponentProps` 接口规范
 *    - 必须实现 `update:modelValue` 事件
 *    - 必须使用 `FieldValue` 格式
 * 
 * ============================================
 * 📚 相关文档
 * ============================================
 * 
 * - 插件系统使用指南：`web/docs/插件系统使用指南.md`
 * - Widget 插件接口：`web/src/architecture/infrastructure/plugins/WidgetPlugin.ts`
 */

import { widgetPluginRegistry, type WidgetPlugin } from './WidgetPlugin'
import { widgetComponentFactory } from '../widgetRegistry/WidgetComponentFactory'
import { fieldExtractorRegistry } from '@/core/stores-v2/extractors/FieldExtractorRegistry'
import { widgetInitializerRegistry } from '@/architecture/presentation/widgets/initializers/WidgetInitializerRegistry'
import { Logger } from '@/core/utils/logger'

/**
 * 注册 Widget 插件
 * 
 * 此函数会：
 * 1. 将插件注册到 WidgetPluginRegistry
 * 2. 将组件注册到 WidgetComponentFactory
 * 3. 将提取器注册到 FieldExtractorRegistry（如果提供）
 * 4. 将初始化器注册到 WidgetInitializerRegistry（如果提供）
 * 
 * @param plugin 插件配置
 * @throws 如果插件类型已存在，抛出错误
 */
export function registerWidgetPlugin(plugin: WidgetPlugin): void {
  try {
    // 1. 注册到插件注册表
    widgetPluginRegistry.register(plugin)
    
    // 2. 注册组件到工厂
    widgetComponentFactory.registerRequestComponent(plugin.widgetType, plugin.requestComponent)
    if (plugin.responseComponent) {
      widgetComponentFactory.registerResponseComponent(plugin.widgetType, plugin.responseComponent)
    } else {
      // 如果没有提供响应组件，使用请求组件作为响应组件
      widgetComponentFactory.registerResponseComponent(plugin.widgetType, plugin.requestComponent)
    }
    
    // 3. 注册提取器（如果提供）
    if (plugin.extractor) {
      fieldExtractorRegistry.registerExtractor(plugin.widgetType, plugin.extractor)
    }
    
    // 4. 注册初始化器（如果提供）
    if (plugin.initializer) {
      widgetInitializerRegistry.register(plugin.widgetType, plugin.initializer)
    }
    
    Logger.debug('[pluginManager]', 'Widget 插件注册成功', {
      name: plugin.name,
      widgetType: plugin.widgetType,
      hasExtractor: !!plugin.extractor,
      hasInitializer: !!plugin.initializer,
      hasResponseComponent: !!plugin.responseComponent
    })
  } catch (error) {
    Logger.error('[pluginManager]', 'Widget 插件注册失败', error, {
      name: plugin.name,
      widgetType: plugin.widgetType
    })
    throw error
  }
}

/**
 * 批量注册 Widget 插件
 * 
 * @param plugins 插件列表
 */
export function registerWidgetPlugins(plugins: WidgetPlugin[]): void {
  plugins.forEach(plugin => {
    try {
      registerWidgetPlugin(plugin)
    } catch (error) {
      Logger.error('[pluginManager]', `插件 "${plugin.name}" 注册失败，跳过`, error)
    }
  })
}

/**
 * 取消注册 Widget 插件
 * 
 * @param widgetType Widget 类型
 */
export function unregisterWidgetPlugin(widgetType: string): void {
  const plugin = widgetPluginRegistry.getPlugin(widgetType)
  if (!plugin) {
    Logger.warn('[pluginManager]', `插件 "${widgetType}" 不存在，无法取消注册`)
    return
  }
  
  // 从各个注册表中移除
  widgetPluginRegistry.unregister(widgetType)
  // 注意：组件工厂、提取器注册表、初始化器注册表目前不支持取消注册
  // 如果需要支持，需要在这些类中添加 unregister 方法
  
  Logger.debug('[pluginManager]', 'Widget 插件已取消注册', {
    widgetType,
    name: plugin.name
  })
}

/**
 * 获取已注册的插件信息
 * 
 * @param widgetType Widget 类型
 * @returns 插件配置，如果不存在返回 undefined
 */
export function getWidgetPlugin(widgetType: string): WidgetPlugin | undefined {
  return widgetPluginRegistry.getPlugin(widgetType)
}

/**
 * 获取所有已注册的插件类型
 * 
 * @returns 插件类型列表
 */
export function getRegisteredPluginTypes(): string[] {
  return widgetPluginRegistry.getRegisteredTypes()
}

