/**
 * 表单字段提取器
 * 🔥 处理 form（struct）类型字段
 */

import type { IFieldExtractor, FieldExtractorRegistry } from './FieldExtractor'
import type { FieldConfig } from '../../../types/field'

export class FormFieldExtractor implements IFieldExtractor {
  extract(
    field: FieldConfig,
    fieldPath: string,
    getValue: (path: string) => any,
    extractorRegistry: FieldExtractorRegistry
  ): any {
    const value = getValue(fieldPath)
    const subFields = field.children || []
    
    if (!subFields.length) {
      return null
    }
    
    const formData: Record<string, any> = {}
    
    // 🔥 获取原始数据，用于回退
    const rawData = value?.raw && typeof value.raw === 'object' && !Array.isArray(value.raw)
      ? value.raw as Record<string, any>
      : null
    
    subFields.forEach(subField => {
      const subFieldPath = `${fieldPath}.${subField.code}`
      const subValue = getValue(subFieldPath)
      
      if (subValue) {
        // 从 store 中提取
        formData[subField.code] = extractorRegistry.extractField(subField, subFieldPath, getValue)
      } else if (rawData && rawData[subField.code] !== undefined) {
        // 🔥 如果 store 中没有值，从原始数据中读取
        const rawValue = rawData[subField.code]
        formData[subField.code] = this.extractFromRaw(subField, rawValue, extractorRegistry)
      }
    })
    
    return formData
  }
  
  /**
   * 从原始数据中提取（用于回退）
   */
  private extractFromRaw(
    field: FieldConfig,
    rawValue: any,
    extractorRegistry: FieldExtractorRegistry
  ): any {
    // 递归处理嵌套结构
    if (field.widget?.type === 'form' && rawValue && typeof rawValue === 'object' && !Array.isArray(rawValue)) {
      const subFields = field.children || []
      const formData: Record<string, any> = {}
      subFields.forEach(subField => {
        if (rawValue[subField.code] !== undefined) {
          formData[subField.code] = this.extractFromRaw(subField, rawValue[subField.code], extractorRegistry)
        }
      })
      return formData
    } else if (field.widget?.type === 'table' && Array.isArray(rawValue)) {
      return rawValue.map((nestedRow: any) => {
        const nestedItemFields = field.children || []
        const nestedRowData: Record<string, any> = {}
        nestedItemFields.forEach(nestedItemField => {
          nestedRowData[nestedItemField.code] = nestedRow[nestedItemField.code]
        })
        return nestedRowData
      })
    } else {
      return rawValue
    }
  }
}

