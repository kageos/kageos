/**
 * FormData Store - 表单数据管理
 * 🔥 完全新增，不依赖旧代码
 * 
 * 功能：
 * - 存储所有字段的值（field_path -> FieldValue）
 * - 提供设置和获取值的方法
 * - 提供提交数据提取方法（递归收集）
 */

import { defineStore } from 'pinia'
import { reactive } from 'vue'
import type { FieldConfig, FieldValue } from '../../types/field'

export const useFormDataStore = defineStore('formData-v2', () => {
  // 存储所有字段的值（field_path -> FieldValue）
  const data = reactive<Map<string, FieldValue>>(new Map())
  
  /**
   * 设置字段值
   */
  function setValue(fieldPath: string, value: FieldValue): void {
    data.set(fieldPath, value)
  }
  
  /**
   * 获取字段值
   */
  function getValue(fieldPath: string): FieldValue {
    return data.get(fieldPath) || { raw: null, display: '', meta: {} }
  }
  
  /**
   * 初始化字段值
   */
  function initializeField(fieldPath: string, initialValue?: FieldValue): void {
    if (initialValue) {
      data.set(fieldPath, initialValue)
    } else if (!data.has(fieldPath)) {
      data.set(fieldPath, { raw: null, display: '', meta: {} })
    }
  }
  
  /**
   * 提取提交数据（递归收集）
   * 
   * @param fields 字段配置列表
   * @param basePath 基础路径（用于嵌套场景）
   * @returns 提交数据对象
   */
  function getSubmitData(fields: FieldConfig[], basePath: string = ''): Record<string, any> {
    const result: Record<string, any> = {}
    
    fields.forEach(field => {
      const fieldPath = basePath ? `${basePath}.${field.code}` : field.code
      const value = data.get(fieldPath)
      
      if (!value) {
        // 字段不存在，跳过
        return
      }
      
      // 根据字段类型决定如何提取
      if (field.widget?.type === 'table') {
        // 表格类型：递归收集每行的数据
        result[field.code] = extractTableData(field, fieldPath)
      } else if (field.widget?.type === 'form') {
        // 表单类型：递归收集子字段的数据
        result[field.code] = extractFormData(field, fieldPath)
      } else {
        // 基础类型：直接返回 raw 值
        result[field.code] = value.raw
      }
    })
    
    return result
  }
  
  /**
   * 递归提取表格数据
   */
  function extractTableData(field: FieldConfig, basePath: string): any[] {
    const value = data.get(basePath)
    if (!value || !Array.isArray(value.raw)) {
      return []
    }
    
    const itemFields = field.children || []
    const tableData = value.raw as any[]
    
    return tableData.map((row, index) => {
      const rowData: Record<string, any> = {}
      
      itemFields.forEach(itemField => {
        const itemFieldPath = `${basePath}[${index}].${itemField.code}`
        const itemValue = data.get(itemFieldPath)
        
        // 🔥 如果 store 中有值，使用 store 的值；否则从原始 row 数据中读取
        if (itemValue) {
          // 递归处理嵌套结构
          if (itemField.widget?.type === 'form') {
            rowData[itemField.code] = extractFormData(itemField, itemFieldPath)
          } else if (itemField.widget?.type === 'table') {
            rowData[itemField.code] = extractTableData(itemField, itemFieldPath)
          } else {
            rowData[itemField.code] = itemValue.raw
          }
        } else if (row && typeof row === 'object') {
          // 🔥 如果 store 中没有值，从原始 row 数据中读取
          const rawValue = row[itemField.code]
          if (rawValue !== undefined) {
            // 递归处理嵌套结构
            if (itemField.widget?.type === 'form' && rawValue && typeof rawValue === 'object' && !Array.isArray(rawValue)) {
              // 对于嵌套的 form，需要递归提取
              rowData[itemField.code] = extractFormDataFromRaw(itemField, rawValue)
            } else if (itemField.widget?.type === 'table' && Array.isArray(rawValue)) {
              // 对于嵌套的 table，需要递归提取
              rowData[itemField.code] = rawValue.map((nestedRow: any) => {
                const nestedItemFields = itemField.children || []
                const nestedRowData: Record<string, any> = {}
                nestedItemFields.forEach(nestedItemField => {
                  nestedRowData[nestedItemField.code] = nestedRow[nestedItemField.code]
                })
                return nestedRowData
              })
            } else {
              rowData[itemField.code] = rawValue
            }
          }
        }
      })
      
      return rowData
    })
  }
  
  /**
   * 从原始数据中提取表单数据（用于嵌套结构）
   */
  function extractFormDataFromRaw(field: FieldConfig, rawData: Record<string, any>): Record<string, any> {
    const subFields = field.children || []
    const formData: Record<string, any> = {}
    
    subFields.forEach(subField => {
      const rawValue = rawData[subField.code]
      if (rawValue !== undefined) {
        // 递归处理嵌套结构
        if (subField.widget?.type === 'form' && rawValue && typeof rawValue === 'object' && !Array.isArray(rawValue)) {
          formData[subField.code] = extractFormDataFromRaw(subField, rawValue)
        } else if (subField.widget?.type === 'table' && Array.isArray(rawValue)) {
          formData[subField.code] = rawValue.map((nestedRow: any) => {
            const nestedItemFields = subField.children || []
            const nestedRowData: Record<string, any> = {}
            nestedItemFields.forEach(nestedItemField => {
              nestedRowData[nestedItemField.code] = nestedRow[nestedItemField.code]
            })
            return nestedRowData
          })
        } else {
          formData[subField.code] = rawValue
        }
      }
    })
    
    return formData
  }
  
  /**
   * 递归提取表单数据
   */
  function extractFormData(field: FieldConfig, basePath: string): Record<string, any> {
    const value = data.get(basePath)
    const subFields = field.children || []
    const formData: Record<string, any> = {}
    
    // 🔥 获取原始数据，用于回退
    const rawData = value?.raw && typeof value.raw === 'object' && !Array.isArray(value.raw) 
      ? value.raw as Record<string, any>
      : null
    
    subFields.forEach(subField => {
      const subFieldPath = `${basePath}.${subField.code}`
      const subValue = data.get(subFieldPath)
      
      if (subValue) {
        // 递归处理嵌套结构
        if (subField.widget?.type === 'form') {
          formData[subField.code] = extractFormData(subField, subFieldPath)
        } else if (subField.widget?.type === 'table') {
          formData[subField.code] = extractTableData(subField, subFieldPath)
        } else {
          formData[subField.code] = subValue.raw
        }
      } else if (rawData && rawData[subField.code] !== undefined) {
        // 🔥 如果 store 中没有值，从原始数据中读取
        const rawValue = rawData[subField.code]
        // 递归处理嵌套结构
        if (subField.widget?.type === 'form' && rawValue && typeof rawValue === 'object' && !Array.isArray(rawValue)) {
          formData[subField.code] = extractFormDataFromRaw(subField, rawValue)
        } else if (subField.widget?.type === 'table' && Array.isArray(rawValue)) {
          formData[subField.code] = rawValue.map((nestedRow: any) => {
            const nestedItemFields = subField.children || []
            const nestedRowData: Record<string, any> = {}
            nestedItemFields.forEach(nestedItemField => {
              nestedRowData[nestedItemField.code] = nestedRow[nestedItemField.code]
            })
            return nestedRowData
          })
        } else {
          formData[subField.code] = rawValue
        }
      }
    })
    
    return formData
  }
  
  /**
   * 清空所有数据
   */
  function clear(): void {
    data.clear()
  }
  
  /**
   * 获取所有字段路径
   */
  function getAllFieldPaths(): string[] {
    return Array.from(data.keys())
  }
  
  return {
    data,
    setValue,
    getValue,
    initializeField,
    getSubmitData,
    clear,
    getAllFieldPaths
  }
})

