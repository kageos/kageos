/**
 * Widget 相关类型定义
 * 
 * 🔥 重构说明：
 * - 移除了对旧版本 BaseWidget 的依赖
 * - FormRendererContext 接口保持兼容，但 registerWidget/unregisterWidget 已不再实际使用（v2 系统）
 */

import type { FieldConfig, FieldValue } from './field'
import type { ReactiveFormDataManager } from '../managers/ReactiveFormDataManager'

/**
 * FormRenderer 上下文接口
 * 提供给 Widget 的 FormRenderer 能力
 * 
 * 注意：v2 系统中 registerWidget/unregisterWidget 已不再实际使用
 * 保留这些方法是为了类型兼容性
 */
export interface FormRendererContext {
  /** 注册 Widget 实例（v2 系统中已不再使用，保留仅为兼容性） */
  registerWidget: (fieldPath: string, widget: any) => void
  
  /** 注销 Widget 实例（v2 系统中已不再使用，保留仅为兼容性） */
  unregisterWidget: (fieldPath: string) => void
  
  /** 获取函数的 HTTP 方法 */
  getFunctionMethod: () => string
  
  /** 获取函数的路由 */
  getFunctionRouter: () => string
  
  /** 获取完整的提交数据（递归收集） */
  getSubmitData: () => Record<string, any>
  
  /** 获取字段错误（v2 系统新增） */
  getFieldError?: (fieldPath: string) => string | null
}

/**
 * Widget 渲染属性
 * 
 * 设计说明：
 * - formManager 和 formRenderer 可以为 null（临时 Widget 场景）
 * - 临时 Widget：用于表格渲染、搜索输入配置等只读场景
 * - 标准 Widget：用于表单编辑，formManager 和 formRenderer 必需
 */
export interface WidgetRenderProps {
  field: FieldConfig
  currentFieldPath: string
  value: FieldValue
  onChange: (newValue: FieldValue) => void
  formManager: ReactiveFormDataManager | null  // ✅ 明确可以为 null
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

/**
 * Widget 静态方法接口
 * 用于类型安全地检查 Widget 类是否实现了静态方法
 */
export interface WidgetStaticMethods {
  /**
   * 从原始数据加载为 FieldValue 格式
   * 用于处理后端返回的原始数据，转换为前端使用的 FieldValue 格式
   */
  loadFromRawData?(rawValue: any, field: FieldConfig): FieldValue
}

/**
 * MarkRaw 后的 Widget 类型
 * Vue 的 markRaw 会移除响应式，但类型系统无法正确推断
 * 使用此类型可以安全地访问 Widget 的方法
 * 
 * 注意：此类型主要用于旧版本 Widget 系统，v2 系统不再使用
 */
export type MarkRawWidget = {
  render: () => any
  getValue: () => FieldValue
  getRawValueForSubmit: () => any
  renderTableCell?: (value?: FieldValue) => any
  [key: string]: any  // 允许其他方法
}
