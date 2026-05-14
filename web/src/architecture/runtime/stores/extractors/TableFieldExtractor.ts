/**
 * 表格字段提取器
 * 🔥 处理 table（[]struct）类型字段
 */

import type { IFieldExtractor, FieldExtractorRegistry } from './FieldExtractor'
import type { FieldConfig } from '@/architecture/domain/types/field'

export class TableFieldExtractor implements IFieldExtractor {
  extract(
    field: FieldConfig,
    fieldPath: string,
    getValue: (path: string) => any,
    extractorRegistry: FieldExtractorRegistry
  ): any {
    const value = getValue(fieldPath)
    if (!value || !Array.isArray(value.raw)) {
      return []
    }
    
    const itemFields = field.children || []
    const tableData = value.raw as any[]
    
    // 提取所有行数据
    const extractedRows = tableData.map((row, index) => {
      const rowData: Record<string, any> = {}
      
      itemFields.forEach(itemField => {
        const itemFieldPath = `${fieldPath}[${index}].${itemField.code}`
        const itemValue = getValue(itemFieldPath)
        
        if (itemValue) {
          // 从 store 中提取
          rowData[itemField.code] = extractorRegistry.extractField(itemField, itemFieldPath, getValue)
        } else if (row && typeof row === 'object') {
          // 🔥 如果 store 中没有值，从原始 row 数据中读取
          const rawValue = row[itemField.code]
          if (rawValue !== undefined) {
            rowData[itemField.code] = this.extractFromRaw(itemField, rawValue, extractorRegistry)
          } else {
            // 如果原始数据也没有，设置为 null（保持结构完整）
            rowData[itemField.code] = null
          }
        } else {
          // 如果既没有 store 值也没有原始数据，设置为 null
          rowData[itemField.code] = null
        }
      })
      
      return rowData
    })
    
    // 🔥 过滤掉空行（所有字段都为 null/undefined 的行）
    return extractedRows.filter(row => {
      return Object.values(row).some(value => value !== null && value !== undefined)
    })
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
