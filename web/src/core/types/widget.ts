/**
 * Widget 相关类型定义
 */

import type { FieldConfig, FieldValue } from './field'
import type { ReactiveFormDataManager } from '../managers/ReactiveFormDataManager'
import type { BaseWidget } from '../widgets/BaseWidget'

/**
 * FormRenderer 上下文接口
 * 提供给 Widget 的 FormRenderer 能力
 */
export interface FormRendererContext {
  /** 注册 Widget 实例 */
  registerWidget: (fieldPath: string, widget: BaseWidget) => void
  
  /** 注销 Widget 实例 */
  unregisterWidget: (fieldPath: string) => void
  
  /** 获取函数的 HTTP 方法 */
  getFunctionMethod: () => string
  
  /** 获取函数的路由 */
  getFunctionRouter: () => string
  
  /** 获取完整的提交数据（递归收集） */
  getSubmitData: () => Record<string, any>
}

/**
 * Widget 渲染属性
 */
export interface WidgetRenderProps {
  field: FieldConfig
  currentFieldPath: string
  value: FieldValue
  onChange: (newValue: FieldValue) => void
  formManager: ReactiveFormDataManager
  formRenderer: FormRendererContext | null
  depth?: number
}

/**
 * Widget 快照数据
 */
export interface WidgetSnapshot {
  widget_type: string
  field_path: string
  field_code: string
  field_value: {
    raw: any
    display: string
    meta?: any
  }
  component_data?: any  // 各组件特定数据
}
