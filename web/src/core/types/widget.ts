/**
 * Widget 相关类型定义
 */

import type { FieldConfig, FieldValue } from './field'
import type { ReactiveFormDataManager } from '../managers/ReactiveFormDataManager'

/**
 * Widget 渲染属性
 */
export interface WidgetRenderProps {
  field: FieldConfig
  currentFieldPath: string
  value: FieldValue
  onChange: (newValue: FieldValue) => void
  formManager: ReactiveFormDataManager
  formRenderer?: {
    registerWidget: (fieldPath: string, widget: any) => void
    unregisterWidget: (fieldPath: string) => void
    getFunctionMethod?: () => string  // 🔥 获取函数的 HTTP 方法
    getFunctionRouter?: () => string  // 🔥 获取函数的路由
  }
  depth?: number  // 嵌套深度
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

