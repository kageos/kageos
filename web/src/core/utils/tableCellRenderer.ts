/**
 * 表格单元格渲染工具函数
 * 
 * 用于 TableRenderer 和 TableWidget 等场景，统一渲染表格单元格内容
 * 
 * 设计原则：
 * - 使用 h() 渲染组件为 VNode，支持复杂组件（如 MultiSelectWidget）
 * - 使用全局 userInfoStore 获取用户信息（自动处理缓存）
 * - 统一的错误处理
 * - 支持不同的渲染模式（table-cell, detail 等）
 */

import { h } from 'vue'
import type { FieldConfig, FieldValue } from '../types/field'
import { convertToFieldValue } from '@/utils/field'
import { widgetComponentFactory } from '@/architecture/infrastructure/widgetRegistry'

/**
 * 渲染表格单元格
 * 
 * @param field 字段配置
 * @param rawValue 原始值（来自后端）
 * @param options 可选配置
 * @param options.mode 渲染模式，默认为 'table-cell'
 * @param options.fieldPath 字段路径，默认为 field.code
 * @param options.formRenderer 表单渲染器上下文（可选）
 * @param options.formManager 表单管理器（可选）
 * @returns { content: any, isString: boolean } - 统一返回格式，方便模板处理
 */
export function renderTableCell(
  field: FieldConfig,
  rawValue: any,
  options: {
    mode?: 'table-cell' | 'detail' | 'response'
    fieldPath?: string
    formRenderer?: any
    formManager?: any
  } = {}
): { content: any, isString: boolean } {
  const {
    mode = 'table-cell',
    fieldPath = field.code,
    formRenderer,
    formManager
  } = options

  try {
    // 🔥 将原始值转换为 FieldValue 格式
    const value = convertToFieldValue(rawValue, field)
    
    // 🔥 使用 widgetComponentFactory 获取组件（v2 方式）
    const WidgetComponent = widgetComponentFactory.getRequestComponent(
      field.widget?.type || 'input'
    )
    
    if (!WidgetComponent) {
      // 如果组件未找到，返回 fallback
      const fallbackValue = rawValue !== null && rawValue !== undefined ? String(rawValue) : '-'
      return {
        content: fallbackValue,
        isString: true
      }
    }
    
    // 🔥 构建组件 props
    const componentProps: Record<string, any> = {
      field: field,
      value: value,
      'model-value': value,
      'field-path': fieldPath,
      mode: mode,
    }
    
    // 可选 props
    if (formRenderer) {
      componentProps['form-renderer'] = formRenderer
    }
    if (formManager) {
      componentProps['form-manager'] = formManager
    }
    
    // 🔥 使用 h() 渲染组件为 VNode（v2 方式）
    const vnode = h(WidgetComponent, componentProps)
    
    // 🔥 统一返回 VNode（v2 组件统一返回 VNode）
    return {
      content: vnode,
      isString: false
    }
  } catch (error) {
    // ✅ 错误处理：返回 fallback
    const fallbackValue = rawValue !== null && rawValue !== undefined ? String(rawValue) : '-'
    return {
      content: fallbackValue,
      isString: true
    }
  }
}

