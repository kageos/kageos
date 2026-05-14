/**
 * 搜索值规范化工具
 * 🔥 统一处理不同组件类型的值转换逻辑，遵循单一职责原则
 */

import { WidgetType } from '@/architecture/domain/constants/widget'
import { SearchType } from '@/architecture/domain/constants/search'
import { parseCommaSeparatedString } from '@/architecture/domain/utils/stringUtils'
import type { FieldConfig } from '@/architecture/domain/types/field'

/**
 * 值规范化选项
 */
export interface NormalizeOptions {
  widgetType?: string
  searchType?: string
  field?: FieldConfig
}

/**
 * 规范化搜索值
 * 根据组件类型和搜索类型，将值转换为后端期望的格式
 * 
 * @param value 原始值
 * @param options 规范化选项
 * @returns 规范化后的值
 */
export function normalizeSearchValue(value: any, options: NormalizeOptions): any {
  // 清空值统一转换为 null
  if (value === '' || value === null || value === undefined) {
    return null
  }

  const { widgetType, searchType } = options

  // 开关组件：将布尔值转换为字符串
  if (widgetType === WidgetType.SWITCH && value !== null) {
    return normalizeSwitchValue(value)
  }

  // 多选组件且搜索类型是 contains：将数组转换为逗号分隔的字符串
  if (widgetType === WidgetType.MULTI_SELECT && hasSearchType(searchType, SearchType.CONTAINS)) {
    return normalizeMultiselectContainsValue(value)
  }

  return value
}

/**
 * 规范化开关组件的值
 * @param value 原始值
 * @returns 规范化后的字符串值（"true" 或 "false"）
 */
function normalizeSwitchValue(value: any): string | null {
  if (value === null || value === undefined) {
    return null
  }

  // 已经是布尔值，直接转换
  if (typeof value === 'boolean') {
    return String(value)
  }

  // 字符串或数字，转换为布尔值再转字符串
  if (value === 'true' || value === true || value === 1 || value === '1') {
    return 'true'
  }

  if (value === 'false' || value === false || value === 0 || value === '0') {
    return 'false'
  }

  // 其他情况，返回 null
  return null
}

/**
 * 规范化多选组件的 contains 搜索值
 * @param value 原始值
 * @returns 规范化后的字符串值（逗号分隔）
 */
function normalizeMultiselectContainsValue(value: any): string | null {
  if (Array.isArray(value)) {
    return value.length > 0 ? value.join(',') : null
  }

  if (value && typeof value === 'string') {
    // 已经是字符串，保持不变
    return value
  }

  // 其他情况，转换为 null
  return null
}

/**
 * 从 URL 恢复值（反向规范化）
 * 将 URL 中的字符串值转换为前端组件需要的格式
 * 
 * @param value URL 中的值
 * @param options 规范化选项
 * @returns 恢复后的值
 */
export function denormalizeSearchValue(value: any, options: NormalizeOptions): any {
  if (value === null || value === undefined || value === '') {
    return null
  }

  const { widgetType, searchType } = options

  // 开关组件：将字符串转换为布尔值
  if (widgetType === WidgetType.SWITCH) {
    return denormalizeSwitchValue(value)
  }

  // 多选组件且搜索类型是 contains：将逗号分隔的字符串转换为数组
  if (widgetType === WidgetType.MULTI_SELECT && hasSearchType(searchType, SearchType.CONTAINS)) {
    return denormalizeMultiselectContainsValue(value)
  }

  return value
}

/**
 * 反向规范化开关组件的值
 * @param value URL 中的字符串值
 * @returns 布尔值
 */
function denormalizeSwitchValue(value: any): boolean | null {
  if (value === null || value === undefined || value === '') {
    return null
  }

  return value === 'true' || value === true
}

/**
 * 反向规范化多选组件的 contains 搜索值
 * @param value URL 中的字符串值
 * @returns 数组值
 */
function denormalizeMultiselectContainsValue(value: any): string[] {
  if (Array.isArray(value)) {
    return value
  }

  if (typeof value === 'string' && value) {
    return parseCommaSeparatedString(value)
  }

  return []
}

/**
 * 检查搜索类型是否包含指定类型（工具函数）
 */
function hasSearchType(searchType: string | undefined | null, type: string): boolean {
  if (!searchType) return false
  return searchType.includes(type)
}

