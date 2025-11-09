/**
 * 多选字段提取器
 * 🔥 处理 multiselect 和 []string 类型字段
 */

import type { IFieldExtractor, FieldExtractorRegistry } from './FieldExtractor'
import type { FieldConfig } from '../../../types/field'

export class MultiSelectFieldExtractor implements IFieldExtractor {
  extract(
    field: FieldConfig,
    fieldPath: string,
    getValue: (path: string) => any,
    extractorRegistry: FieldExtractorRegistry
  ): any {
    const value = getValue(fieldPath)
    const raw = value?.raw
    
    // 确保返回数组
    if (Array.isArray(raw)) {
      return raw
    } else if (raw !== null && raw !== undefined) {
      // 兼容旧数据：如果是字符串，转换为数组
      return [raw]
    } else {
      return []
    }
  }
}

