/**
 * 基础字段提取器
 * 🔥 处理普通字段（input、select、number 等）
 */

import type { IFieldExtractor, FieldExtractorRegistry } from './FieldExtractor'
import type { FieldConfig } from '../../../types/field'

export class BasicFieldExtractor implements IFieldExtractor {
  extract(
    field: FieldConfig,
    fieldPath: string,
    getValue: (path: string) => any,
    extractorRegistry: FieldExtractorRegistry
  ): any {
    const value = getValue(fieldPath)
    return value?.raw ?? null
  }
}

