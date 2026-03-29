/**
 * Widget 初始化接口
 * 
 * 🔥 依赖倒置原则：框架依赖抽象接口，不依赖具体组件
 * 
 * 设计原则：
 * - 每个组件实现自己的初始化逻辑
 * - 框架只调用接口，不关心具体实现
 * - 组件可以决定是否需要初始化（返回 null 表示不需要）
 */

import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import type { FunctionDetail } from '@/architecture/domain/types'

export interface WidgetInitFormDataStore {
  getValue: (fieldCode: string) => FieldValue | undefined
  setValue: (fieldCode: string, value: FieldValue) => void
  getAllValues: () => Record<string, FieldValue>
}

/**
 * Widget 初始化上下文
 */
export interface WidgetInitContext {
  /** 字段配置 */
  field: FieldConfig
  
  /** 当前字段值（可能来自 URL、默认值或初始数据） */
  currentValue: FieldValue
  
  /** 表单所有字段的值（用于依赖字段的初始化） */
  allFormData: Record<string, FieldValue>

  /** 当前表单的数据存取接口（用于嵌套字段初始化） */
  formDataStore?: WidgetInitFormDataStore
  
  /** 函数详情（用于调用回调接口） */
  functionDetail: FunctionDetail
  
  /** 初始化源信息（用于判断是否需要初始化） */
  initSource: 'url' | 'default'
  
  /** 字段完整路径（用于嵌套字段，如 payment_info.discount_info） */
  fieldPath?: string
}

/**
 * Widget 初始化接口
 * 
 * 每个组件实现此接口，负责自己的初始化逻辑
 */
export interface IWidgetInitializer {
  /**
   * 初始化组件
   * 
   * @param context 初始化上下文
   * @returns 初始化后的 FieldValue，如果不需要初始化则返回 null
   */
  initialize(context: WidgetInitContext): Promise<FieldValue | null>
}
