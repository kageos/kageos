/**
 * WidgetComponentFactory - 组件工厂
 * 
 * ============================================
 * 📋 需求说明
 * ============================================
 * 
 * 1. **组件注册**：
 *    - 注册请求参数组件（用于表单编辑）
 *    - 注册响应参数组件（用于响应展示）
 *    - 根据 `widget.type` 获取对应的 Vue 组件
 * 
 * 2. **组件映射**：
 *    - `widget.type` → Vue Component
 *    - 支持一个类型注册多个组件（请求/响应）
 *    - 支持默认组件（未找到时使用 InputWidget）
 * 
 * 3. **扩展性**：
 *    - 支持注册自定义组件
 *    - 支持查询已注册的组件类型
 *    - 不影响现有组件的使用
 * 
 * ============================================
 * 🎯 设计思路
 * ============================================
 * 
 * 1. **工厂模式**：
 *    - 使用 Map 存储组件映射
 *    - 提供注册和获取接口
 *    - 支持请求组件和响应组件分离
 * 
 * 2. **默认组件**：
 *    - 未找到组件时，尝试使用默认组件（InputWidget）
 *    - 如果连默认组件都没有，返回 null
 * 
 * 3. **组件初始化**：
 *    - 在 `initializeWidgetComponentFactory` 中统一注册所有组件
 *    - 应用启动时自动初始化
 * 
 * ============================================
 * 📝 关键功能
 * ============================================
 * 
 * 1. **registerRequestComponent**：
 *    - 注册请求参数组件
 *    - 用于表单编辑模式
 * 
 * 2. **registerResponseComponent**：
 *    - 注册响应参数组件
 *    - 用于响应展示模式
 *    - 如果没有响应组件，使用请求组件
 * 
 * 3. **getRequestComponent / getResponseComponent**：
 *    - 根据 `widget.type` 获取组件
 *    - 未找到时使用默认组件
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **组件注册**：
 *    - 组件必须在应用启动时注册
 *    - 注册在 `initializeWidgetComponentFactory` 函数中
 * 
 * 2. **组件类型**：
 *    - 组件类型必须与后端 `widget.type` 一致
 *    - 支持别名（如 `text` 和 `input` 都指向 InputWidget）
 * 
 * 3. **默认组件**：
 *    - 默认组件是 InputWidget
 *    - 确保默认组件已注册，否则可能返回 null
 * 
 * ============================================
 * 📚 相关文档
 * ============================================
 * 
 * - 组件初始化：`web/src/architecture/infrastructure/widgetRegistry/index.ts`
 * - 插件系统：`web/src/architecture/infrastructure/plugins/pluginManager.ts`
 */

import type { Component } from 'vue'
import { WidgetType } from '@/core/constants/widget'

export class WidgetComponentFactory {
  // 请求参数组件映射（widget.type -> Component）
  private requestComponentMap: Map<string, Component> = new Map()
  
  // 响应参数组件映射（widget.type -> Component）
  private responseComponentMap: Map<string, Component> = new Map()
  
  /**
   * 注册请求参数组件
   */
  registerRequestComponent(type: string, component: Component): void {
    this.requestComponentMap.set(type, component)
  }
  
  /**
   * 注册响应参数组件
   */
  registerResponseComponent(type: string, component: Component): void {
    this.responseComponentMap.set(type, component)
  }
  
  /**
   * 获取请求参数组件
   */
  getRequestComponent(type: string): Component | null {
    const component = this.requestComponentMap.get(type)
    if (!component) {
      console.warn(`[WidgetComponentFactory] 未找到请求参数组件: ${type}，尝试使用默认组件`)
      // 返回默认组件（Input）
      const defaultComponent = this.requestComponentMap.get(WidgetType.INPUT)
      if (!defaultComponent) {
        console.error(`[WidgetComponentFactory] 连默认组件（input）都未找到！`)
        return null
      }
      return defaultComponent
    }
    return component
  }
  
  /**
   * 获取响应参数组件
   * 如果该类型有对应的响应组件，返回它；否则返回请求组件
   */
  getResponseComponent(type: string): Component | null {
    const responseComponent = this.responseComponentMap.get(type)
    if (responseComponent) {
      return responseComponent
    }

    // 没有响应组件，使用请求组件
    return this.getRequestComponent(type)
  }
  
  /**
   * 检查是否已注册请求组件
   */
  hasRequestComponent(type: string): boolean {
    return this.requestComponentMap.has(type)
  }
  
  /**
   * 检查是否已注册响应组件
   */
  hasResponseComponent(type: string): boolean {
    return this.responseComponentMap.has(type)
  }
  
  /**
   * 获取所有已注册的请求组件类型
   */
  getRegisteredRequestTypes(): string[] {
    return Array.from(this.requestComponentMap.keys())
  }
  
  /**
   * 获取所有已注册的响应组件类型
   */
  getRegisteredResponseTypes(): string[] {
    return Array.from(this.responseComponentMap.keys())
  }
}

// 🔥 不在这里导出实例，避免循环依赖
// 实例在 index.ts 中创建和导出，这样 FormWidget 和 TableWidget 可以安全导入

