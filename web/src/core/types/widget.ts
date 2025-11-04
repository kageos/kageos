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
 */
export type MarkRawWidget = BaseWidget & {
  render: () => any
  getValue: () => FieldValue
  getRawValueForSubmit: () => any
  renderTableCell?: (value?: FieldValue) => any
}
