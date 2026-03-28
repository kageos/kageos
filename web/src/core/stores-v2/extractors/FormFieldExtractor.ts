/**
 * 表单字段提取器
 * 🔥 处理 form（struct）类型字段
 */

import type { IFieldExtractor, FieldExtractorRegistry } from './FieldExtractor'
import type { FieldConfig } from '../../types/field'

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
        const extracted = extractorRegistry.extractField(subField, subFieldPath, getValue)
        
        // 🔥 即使提取的值是 undefined，也要添加到结果中（对于嵌套结构，需要保持结构完整）
        if (extracted !== undefined) {
          formData[subField.code] = extracted
        } else if (subField.widget?.type === 'form' || subField.widget?.type === 'table') {
          // 🔥 对于嵌套的 form 或 table，即使没有值也要返回空结构
          formData[subField.code] = subField.widget?.type === 'table' ? [] : {}
        }
      } else if (rawData && rawData[subField.code] !== undefined) {
        // 🔥 如果 store 中没有值，从原始数据中读取
        const rawValue = rawData[subField.code]
        formData[subField.code] = this.extractFromRaw(subField, rawValue, extractorRegistry)
      } else {
        // 🔥 如果 store 和原始数据都没有值，根据字段类型返回默认值
        // 对于嵌套的 form，需要递归提取所有子字段，即使值为空也要保持结构完整
        if (subField.widget?.type === 'form') {
          const nestedFormData: Record<string, any> = {}
          const nestedSubFields = subField.children || []
          nestedSubFields.forEach(nestedSubField => {
            // 🔥 递归提取嵌套字段，确保结构完整
            const nestedExtracted = extractorRegistry.extractField(nestedSubField, `${subFieldPath}.${nestedSubField.code}`, getValue)
            if (nestedExtracted !== undefined) {
              nestedFormData[nestedSubField.code] = nestedExtracted
            } else if (nestedSubField.widget?.type === 'form') {
              nestedFormData[nestedSubField.code] = {}
            } else if (nestedSubField.widget?.type === 'table') {
              nestedFormData[nestedSubField.code] = []
            }
          })
          formData[subField.code] = nestedFormData
        } else if (subField.widget?.type === 'table') {
          formData[subField.code] = []
        }
        // 对于基础字段，不添加到 formData 中（undefined 会被忽略）
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
