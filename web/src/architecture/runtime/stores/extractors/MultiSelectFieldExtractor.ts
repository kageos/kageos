/**
 * 多选字段提取器
 * 🔥 处理 multiselect 和 []string 类型字段
 * 
 * 支持两种数据类型：
 * 1. string 类型：返回逗号分隔的字符串格式（如 "紧急,低优先级"）
 * 2. []string 或其他数组类型：返回数组格式（如 ["紧急", "低优先级"]）
 * 
 * 根据 field.data.type 自动决定返回格式，确保与后端字段类型严格对齐
 */

import type { IFieldExtractor, FieldExtractorRegistry } from './FieldExtractor'
import type { FieldConfig } from '@/architecture/domain/types/field'
import { isStringDataType, getMultiSelectDefaultDataType } from '@/architecture/domain/constants/widget'

export class MultiSelectFieldExtractor implements IFieldExtractor {
  extract(
    field: FieldConfig,
    fieldPath: string,
    getValue: (path: string) => any,
    extractorRegistry: FieldExtractorRegistry
  ): any {
    const value = getValue(fieldPath)
    const raw = value?.raw
    // 🔥 优先使用 value.dataType（通用字段），如果没有则从 field.data.type 获取
    const dataType = value?.dataType || field.data?.type || getMultiSelectDefaultDataType()
    
    /**
     * 🔥 根据 field.data.type 决定返回格式
     * - 如果 type 是 string：返回逗号分隔的字符串
     * - 如果 type 是 []string 或其他数组类型：返回数组
     */
    if (isStringDataType(dataType)) {
      // 如果类型是 string，返回逗号分隔的字符串
      if (Array.isArray(raw)) {
        // 如果 raw 是数组，转换为逗号分隔的字符串
        return raw.length > 0 ? raw.join(',') : ''
      } else if (typeof raw === 'string') {
        // 如果 raw 已经是字符串，直接返回
        return raw
      } else {
        // 空值返回空字符串
        return ''
      }
    } else {
      // 如果类型是 []string 或 array，返回数组
      if (Array.isArray(raw)) {
        return raw
      } else if (typeof raw === 'string' && raw) {
        // 兼容旧数据：如果是逗号分隔的字符串，转换为数组
        if (raw.includes(',')) {
          return raw.split(',').map(v => v.trim()).filter(v => v)
        }
        // 单个值
        return [raw]
      } else {
        return []
      }
    }
  }
}
