/**
 * URL 同步工具函数
 * 提取公共逻辑，避免代码重复
 */

import type { FieldConfig, FieldValue } from '../../../domain/types'
import { Logger } from '@/core/utils/logger'
import { WidgetType } from '@/core/constants/widget'

/**
 * 检查字段值是否为空
 */
export function isEmptyValue(fieldValue: FieldValue | undefined | null): boolean {
  if (!fieldValue || fieldValue.raw === null || fieldValue.raw === undefined) {
    return true
  }
  
  // 跳过空字符串
  if (typeof fieldValue.raw === 'string' && fieldValue.raw.trim() === '') {
    return true
  }
  
  // 跳过空数组
  if (Array.isArray(fieldValue.raw) && fieldValue.raw.length === 0) {
    return true
  }
  
  // 跳过空对象
  if (typeof fieldValue.raw === 'object' && !Array.isArray(fieldValue.raw) && Object.keys(fieldValue.raw).length === 0) {
    return true
  }
  
  return false
}

import type { InputWidgetConfig } from '@/core/types/widget-configs'

/**
 * 检查字段是否应该跳过 URL 同步（黑名单检查）
 */
export function shouldSkipURLSync(field: FieldConfig, logPrefix: string = '[URLSync]'): boolean {
  const widgetType = field.widget?.type
  const widgetConfig = field.widget?.config as InputWidgetConfig | undefined
  
  // 排除复杂类型
  const unsupportedTypes = [WidgetType.FORM, WidgetType.TABLE, WidgetType.FILES]
  if (widgetType && unsupportedTypes.includes(widgetType)) {
    Logger.debug(logPrefix, `字段 ${field.code} 是复杂类型（${widgetType}），跳过 URL 同步`)
    return true
  }
  
  // 🔥 排除密码字段（安全性考虑：密码不应出现在 URL 中）
  if (widgetType === WidgetType.INPUT && widgetConfig?.password === true) {
    Logger.debug(logPrefix, `字段 ${field.code} 是密码字段，跳过 URL 同步（安全性考虑）`)
    return true
  }
  
  return false
}

/**
 * 将字段值转换为 URL 查询参数值
 */
export function convertFieldValueToURLParam(fieldValue: FieldValue): string {
  if (Array.isArray(fieldValue.raw)) {
    // 数组类型（如 multiselect）：使用逗号分隔
    return fieldValue.raw.map(v => String(v)).join(',')
  } else {
    // 其他类型：直接转换为字符串
    return String(fieldValue.raw)
  }
}

/**
 * 合并 URL 查询参数
 * 
 * @param currentQuery 当前 URL 查询参数
 * @param newQuery 新的查询参数
 * @param linkType 链接类型（用于判断是否是链接导航）
 * @returns 合并后的查询参数
 */
export function mergeURLQueryParams(
  currentQuery: Record<string, any>,
  newQuery: Record<string, string>,
  linkType?: string
): Record<string, string | string[]> {
  const hasQueryParams = Object.keys(currentQuery).length > 0
  const isLinkNavigation = linkType && currentQuery._link_type === linkType
  
  // 🔥 如果 URL 没有查询参数（刚切换函数），直接使用新的查询参数，不保留任何旧参数
  if (!hasQueryParams && !isLinkNavigation) {
    Logger.debug('[URLSync]', 'URL 没有查询参数，不保留旧参数，直接使用新参数')
    return { ...newQuery }
  }
  
  // URL 有查询参数，保留现有参数（如 _link_type、_tab）并合并新的参数
  const mergedQuery: Record<string, string | string[]> = { ...currentQuery }
  
  // 保留以 _ 开头的参数（前端状态参数，如 _tab=OnTableAddRow），但清除 _link_type（临时参数）
  Object.keys(mergedQuery).forEach(key => {
    if (key.startsWith('_') && key === '_link_type') {
      // 清除临时参数
      delete mergedQuery[key]
    }
    // 其他以 _ 开头的参数保留（如 _tab）
  })
  
  // 合并新的参数（覆盖旧的同名参数）
  Object.assign(mergedQuery, newQuery)
  
  Logger.debug('[URLSync]', 'URL 有查询参数，保留现有参数', {
    preservedQueryKeys: Object.keys(mergedQuery)
  })
  
  return mergedQuery
}

