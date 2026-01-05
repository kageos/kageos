/**
 * Widget 初始化器注册表
 * 
 * 🔥 依赖倒置原则：只依赖 IWidgetInitializer 接口，不依赖具体组件
 * 
 * 功能：
 * - 注册组件的初始化器
 * - 调用组件的初始化接口
 * - 不关心具体组件的实现细节
 */

import type { IWidgetInitializer, WidgetInitContext } from '@/architecture/presentation/widgets/interfaces/IWidgetInitializer'
import type { FieldValue } from '@/architecture/domain/types'

/**
 * Widget 初始化器注册表
 * 
 * 🔥 依赖倒置原则：只依赖抽象接口，不依赖具体组件
 */
export class WidgetInitializerRegistry {
  private initializers: Map<string, IWidgetInitializer> = new Map()
  
  /**
   * 注册初始化器
   * 
   * @param widgetType 组件类型（如 'select', 'multiselect'）
   * @param initializer 初始化器实例
   */
  register(widgetType: string, initializer: IWidgetInitializer): void {
    this.initializers.set(widgetType, initializer)
  }
  
  /**
   * 取消注册初始化器
   * 
   * @param widgetType 组件类型
   */
  unregister(widgetType: string): void {
    this.initializers.delete(widgetType)
  }
  
  /**
   * 检查是否已注册初始化器
   * 
   * @param widgetType 组件类型
   * @returns 是否已注册
   */
  has(widgetType: string): boolean {
    return this.initializers.has(widgetType)
  }
  
  /**
   * 初始化组件
   * 
   * 🔥 依赖倒置原则：调用抽象接口，不关心具体实现
   * 
   * @param context 初始化上下文
   * @returns 初始化后的 FieldValue，如果不需要初始化则返回原始值
   */
  async initialize(context: WidgetInitContext): Promise<FieldValue> {
    const widgetType = context.field.widget?.type
    if (!widgetType) {
      console.log(`🔍 [WidgetInitializerRegistry] 字段 ${context.field.code} 没有组件类型，跳过初始化`)
      return context.currentValue  // 没有组件类型，返回原始值
    }
    
    const initializer = this.initializers.get(widgetType)
    if (!initializer) {
      console.log(`🔍 [WidgetInitializerRegistry] 字段 ${context.field.code} 没有注册初始化器（widgetType: ${widgetType}），跳过初始化`)
      return context.currentValue  // 没有注册初始化器，返回原始值
    }
    
    console.log(`🔍 [WidgetInitializerRegistry] 调用组件初始化器`, {
      fieldCode: context.field.code,
      widgetType,
      hasInitializer: !!initializer
    })
    
    try {
      // 🔥 调用抽象接口，不关心具体实现
      const initializedValue = await initializer.initialize(context)
      
      // 如果组件返回 null，表示不需要初始化，返回原始值
      if (initializedValue === null) {
        console.log(`🔍 [WidgetInitializerRegistry] 字段 ${context.field.code} 组件返回 null，不需要初始化`)
        return context.currentValue
      }
      
      if (initializedValue !== context.currentValue) {
        console.log(`✅ [WidgetInitializerRegistry] 字段 ${context.field.code} 组件初始化完成`, {
          widgetType,
          oldValue: {
            raw: context.currentValue.raw,
            display: context.currentValue.display
          },
          newValue: {
            raw: initializedValue.raw,
            display: initializedValue.display
          }
        })
      } else {
        console.log(`🔍 [WidgetInitializerRegistry] 字段 ${context.field.code} 组件返回原始值，未发生变化`)
      }
      
      return initializedValue
    } catch (error) {
      console.error(`❌ [WidgetInitializerRegistry] 初始化组件失败: ${widgetType}`, {
        fieldCode: context.field.code,
        error
      })
      return context.currentValue  // 初始化失败，返回原始值
    }
  }
  
  /**
   * 批量初始化组件
   * 
   * @param contexts 初始化上下文数组
   * @returns 初始化后的 FieldValue 数组
   */
  async initializeBatch(contexts: WidgetInitContext[]): Promise<FieldValue[]> {
    return Promise.all(contexts.map(context => this.initialize(context)))
  }
}

// 全局单例
export const widgetInitializerRegistry = new WidgetInitializerRegistry()

