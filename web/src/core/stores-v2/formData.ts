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
import { fieldExtractorRegistry } from './extractors/FieldExtractorRegistry'
import { Logger } from '@/core/utils/logger'

export const useFormDataStore = defineStore('formData-v2', () => {
  // 存储所有字段的值（field_path -> FieldValue）
  const data = reactive<Map<string, FieldValue>>(new Map())
  
  // 🔥 使用全局字段提取器注册表（支持插件扩展）
  
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
    const result: Record<string, any> = {}
    
    fields.forEach(field => {
      const fieldPath = basePath ? `${basePath}.${field.code}` : field.code
      
      // 🔥 使用全局提取器注册表提取字段值（即使字段不存在也会尝试从原始数据中提取）
      const fieldValue = data.get(fieldPath)
      const extractedValue = fieldExtractorRegistry.extractField(field, fieldPath, (path: string) => {
        return data.get(path)
      })
      
      // 🔥 调试日志：检查字段值提取（仅对必填字段，使用 Logger.debug）
      if (field.validation && field.validation.includes('required')) {
        Logger.debug('[getSubmitData]', '必填字段提取', {
          fieldCode: field.code,
          fieldPath,
          fieldValue,
          extractedValue,
          extractedValueType: typeof extractedValue,
          isUndefined: extractedValue === undefined,
          isNull: extractedValue === null
        })
      }
      
      // 🔥 对于 form 和 table 类型，即使提取的值是空对象或空数组，也要添加到结果中
      // 对于其他类型，只有当提取的值不为 undefined 时才添加
      // ⚠️ 注意：null 值也要添加到结果中，让后端可以正确验证必填字段
      if (extractedValue !== undefined) {
        result[field.code] = extractedValue
      } else if (field.widget?.type === 'form') {
        // 🔥 form 类型字段，即使没有值也要返回空对象，保持结构完整
        result[field.code] = {}
      } else if (field.widget?.type === 'table') {
        // 🔥 table 类型字段，即使没有值也要返回空数组，保持结构完整
        result[field.code] = []
      }
      // 🔥 其他类型字段如果没有值（extractedValue === undefined），不添加到结果中
      // 这样后端可以正确验证必填字段（如果字段不在提交数据中，后端会报错）
    })
    
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
  
  /**
   * 获取所有字段值（用于 URL 同步等场景）
   */
  function getAllValues(): Record<string, FieldValue> {
    const allValues: Record<string, FieldValue> = {}
    data.forEach((value, key) => {
      allValues[key] = value
    })
    return allValues
  }
  
  return {
    data,
    setValue,
    getValue,
    initializeField,
    getSubmitData,
    clear,
    getAllFieldPaths,
    getAllValues
  }
})
