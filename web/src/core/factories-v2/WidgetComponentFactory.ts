/**
 * WidgetComponentFactory - 组件工厂
 * 🔥 完全新增，不依赖旧代码
 * 
 * 功能：
 * - 注册请求参数组件
 * - 注册响应参数组件
 * - 根据 widget.type 获取组件
 */

import type { Component } from 'vue'
import { WidgetType } from '../constants/widget'

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

// 导出单例
export const widgetComponentFactory = new WidgetComponentFactory()

