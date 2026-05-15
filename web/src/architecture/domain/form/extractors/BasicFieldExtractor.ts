/**
 * 基础字段提取器
 * 🔥 处理普通字段（input、select、number 等）
 */

import type { IFieldExtractor, FieldExtractorRegistry } from './FieldExtractor'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'

export class BasicFieldExtractor implements IFieldExtractor {
  extract(
    field: FieldConfig,
    fieldPath: string,
    getValue: (path: string) => FieldValue | undefined,
    extractorRegistry: FieldExtractorRegistry
  ): unknown {
    const value = getValue(fieldPath)
    return value?.raw ?? null
  }
}
