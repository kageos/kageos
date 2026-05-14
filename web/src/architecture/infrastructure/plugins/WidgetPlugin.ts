/**
 * Widget 插件接口
 * 
 * 🔥 插件化扩展机制：支持外部注册 Widget 组件、提取器、初始化器
 * 
 * 设计原则：
 * - 统一插件接口，简化扩展流程
 * - 支持一次性注册所有相关组件
 * - 便于第三方开发者扩展
 */

import type { Component } from 'vue'
import type { IFieldExtractor } from '@/architecture/runtime/stores/extractors/FieldExtractor'
import type { IWidgetInitializer } from '@/architecture/presentation/widgets/interfaces/IWidgetInitializer'

/**
 * Widget 插件配置
 */
export interface WidgetPlugin {
  /** 插件名称（用于标识和调试） */
  name: string
  
  /** Widget 类型（对应后端 widget.type） */
  widgetType: string
  
  /** 请求参数组件（用于表单编辑） */
  requestComponent: Component
  
  /** 响应参数组件（用于响应展示，可选，默认使用 requestComponent） */
  responseComponent?: Component
  
  /** 字段提取器（可选，默认使用 BasicFieldExtractor） */
  extractor?: IFieldExtractor
  
  /** 组件初始化器（可选，用于动态初始化组件值） */
  initializer?: IWidgetInitializer
  
  /** 插件元数据（可选） */
  metadata?: {
    description?: string
    version?: string
    author?: string
  }
}

/**
 * Widget 插件注册器
 * 
 * 职责：
 * - 统一管理所有 Widget 插件
 * - 提供插件注册和查询接口
 * - 支持插件热插拔
 */
export class WidgetPluginRegistry {
  private plugins: Map<string, WidgetPlugin> = new Map()
  
  /**
   * 注册 Widget 插件
   * 
   * @param plugin 插件配置
   * @throws 如果插件类型已存在，抛出错误
   */
  register(plugin: WidgetPlugin): void {
    if (this.plugins.has(plugin.widgetType)) {
      throw new Error(
        `[WidgetPluginRegistry] Widget 类型 "${plugin.widgetType}" 已被注册，插件名称: ${this.plugins.get(plugin.widgetType)?.name}`
      )
    }
    
    this.plugins.set(plugin.widgetType, plugin)
  }
  
  /**
   * 取消注册 Widget 插件
   * 
   * @param widgetType Widget 类型
   */
  unregister(widgetType: string): void {
    this.plugins.delete(widgetType)
  }
  
  /**
   * 获取插件
   * 
   * @param widgetType Widget 类型
   * @returns 插件配置，如果不存在返回 undefined
   */
  getPlugin(widgetType: string): WidgetPlugin | undefined {
    return this.plugins.get(widgetType)
  }
  
  /**
   * 检查插件是否已注册
   * 
   * @param widgetType Widget 类型
   * @returns 是否已注册
   */
  hasPlugin(widgetType: string): boolean {
    return this.plugins.has(widgetType)
  }
  
  /**
   * 获取所有已注册的插件类型
   * 
   * @returns 插件类型列表
   */
  getRegisteredTypes(): string[] {
    return Array.from(this.plugins.keys())
  }
  
  /**
   * 获取所有已注册的插件
   * 
   * @returns 插件列表
   */
  getAllPlugins(): WidgetPlugin[] {
    return Array.from(this.plugins.values())
  }
  
  /**
   * 清空所有插件（主要用于测试）
   */
  clear(): void {
    this.plugins.clear()
  }
}

// 导出单例
export const widgetPluginRegistry = new WidgetPluginRegistry()

