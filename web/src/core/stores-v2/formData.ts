/**
 * FormData Store - 表单数据管理
 * 🔥 完全新增，不依赖旧代码
 * 
 * 功能：
 * - 存储所有字段的值（field_path -> FieldValue）
 * - 提供设置和获取值的方法
 * - 提供提交数据提取方法（递归收集，使用策略模式）
 */

import { defineStore } from 'pinia'
import { reactive } from 'vue'
import type { FieldConfig, FieldValue } from '../types/field'
import { FieldExtractorRegistry } from './extractors/FieldExtractorRegistry'

export const useFormDataStore = defineStore('formData-v2', () => {
  // 存储所有字段的值（field_path -> FieldValue）
  const data = reactive<Map<string, FieldValue>>(new Map())
  
  // 🔥 字段提取器注册表（遵循依赖倒置原则）
  const extractorRegistry = new FieldExtractorRegistry()
  
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
   * 🔥 使用策略模式，遵循依赖倒置原则
   * 
   * @param fields 字段配置列表
   * @param basePath 基础路径（用于嵌套场景）
   * @returns 提交数据对象
   */
  function getSubmitData(fields: FieldConfig[], basePath: string = ''): Record<string, any> {
    console.log('[FormDataStore] getSubmitData 开始', {
      fieldsCount: fields.length,
      basePath,
      fieldCodes: fields.map(f => f.code)
    })
    
    const result: Record<string, any> = {}
    
    fields.forEach(field => {
      const fieldPath = basePath ? `${basePath}.${field.code}` : field.code
      
      console.log(`[FormDataStore] 提取字段 ${field.code}`, {
        fieldPath,
        widgetType: field.widget?.type,
        hasChildren: !!field.children
      })
      
      // 🔥 使用提取器注册表提取字段值（即使字段不存在也会尝试从原始数据中提取）
      const extractedValue = extractorRegistry.extractField(field, fieldPath, (path: string) => {
        return data.get(path)
      })
      
      console.log(`[FormDataStore] 字段 ${field.code} 提取结果:`, extractedValue)
      
      // 🔥 对于 form 和 table 类型，即使提取的值是空对象或空数组，也要添加到结果中
      // 对于其他类型，只有当提取的值不为 undefined 时才添加
      if (extractedValue !== undefined) {
        result[field.code] = extractedValue
      } else if (field.widget?.type === 'form') {
        // 🔥 form 类型字段，即使没有值也要返回空对象，保持结构完整
        result[field.code] = {}
      } else if (field.widget?.type === 'table') {
        // 🔥 table 类型字段，即使没有值也要返回空数组，保持结构完整
        result[field.code] = []
      }
    })
    
    console.log('[FormDataStore] getSubmitData 完成', result)
    
    return result
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

