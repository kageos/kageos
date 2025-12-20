/**
 * FormData Store - 表单数据管理（函数粒度缓存）
 * 🔥 完全新增，不依赖旧代码
 * 
 * 功能：
 * - 存储所有字段的值（field_path -> FieldValue）
 * - 提供设置和获取值的方法
 * - 提供提交数据提取方法（递归收集，使用策略模式）
 * - 🔥 支持函数粒度缓存，每个函数独立存储数据
 */

import { defineStore } from 'pinia'
import { reactive, computed } from 'vue'
import type { FieldConfig, FieldValue } from '../types/field'
import { FieldExtractorRegistry } from './extractors/FieldExtractorRegistry'

/**
 * 获取函数唯一标识
 * 优先使用 id，如果没有则使用 router
 */
function getFunctionKey(functionId?: number | string, functionRouter?: string): string {
  if (functionId && functionId !== 0) {
    return `function_${functionId}`
  }
  if (functionRouter) {
    return `router_${functionRouter}`
  }
  return 'default'
}

export const useFormDataStore = defineStore('formData-v2', () => {
  // 🔥 函数粒度的数据缓存：functionKey -> Map<fieldPath, FieldValue>
  const functionDataCache = reactive<Map<string, Map<string, FieldValue>>>(new Map())
  
  // 🔥 当前激活的函数标识（用于向后兼容，如果没有传入 functionKey 则使用这个）
  const currentFunctionKey = reactive<{ value: string }>({ value: 'default' })
  
  // 🔥 字段提取器注册表（遵循依赖倒置原则）
  const extractorRegistry = new FieldExtractorRegistry()
  
  /**
   * 获取指定函数的数据 Map
   */
  function getFunctionData(functionKey?: string): Map<string, FieldValue> {
    const key = functionKey || currentFunctionKey.value
    if (!functionDataCache.has(key)) {
      functionDataCache.set(key, reactive(new Map()))
    }
    return functionDataCache.get(key)!
  }
  
  /**
   * 设置当前函数标识
   */
  function setCurrentFunction(functionId?: number | string, functionRouter?: string): void {
    currentFunctionKey.value = getFunctionKey(functionId, functionRouter)
  }
  
  /**
   * 设置字段值
   */
  function setValue(fieldPath: string, value: FieldValue, functionId?: number | string, functionRouter?: string): void {
    const functionKey = getFunctionKey(functionId, functionRouter)
    const data = getFunctionData(functionKey)
    data.set(fieldPath, value)
  }
  
  /**
   * 获取字段值
   */
  function getValue(fieldPath: string, functionId?: number | string, functionRouter?: string): FieldValue {
    const functionKey = getFunctionKey(functionId, functionRouter)
    const data = getFunctionData(functionKey)
    return data.get(fieldPath) || { raw: null, display: '', meta: {} }
  }
  
  /**
   * 初始化字段值
   */
  function initializeField(fieldPath: string, initialValue?: FieldValue, functionId?: number | string, functionRouter?: string): void {
    const functionKey = getFunctionKey(functionId, functionRouter)
    const data = getFunctionData(functionKey)
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
   * @param functionId 函数 ID（可选）
   * @param functionRouter 函数路由（可选）
   * @returns 提交数据对象
   */
  function getSubmitData(fields: FieldConfig[], basePath: string = '', functionId?: number | string, functionRouter?: string): Record<string, any> {
    const functionKey = getFunctionKey(functionId, functionRouter)
    const data = getFunctionData(functionKey)
    const result: Record<string, any> = {}
    
    fields.forEach(field => {
      const fieldPath = basePath ? `${basePath}.${field.code}` : field.code
      
      // 🔥 使用提取器注册表提取字段值（即使字段不存在也会尝试从原始数据中提取）
      const extractedValue = extractorRegistry.extractField(field, fieldPath, (path: string) => {
        return data.get(path)
      })
      
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
    
    return result
  }
  
  /**
   * 清空指定函数的数据
   */
  function clear(functionId?: number | string, functionRouter?: string): void {
    const functionKey = getFunctionKey(functionId, functionRouter)
    const data = getFunctionData(functionKey)
    data.clear()
  }
  
  /**
   * 清空所有函数的数据
   */
  function clearAll(): void {
    functionDataCache.clear()
    currentFunctionKey.value = 'default'
  }
  
  /**
   * 获取指定函数的所有字段路径
   */
  function getAllFieldPaths(functionId?: number | string, functionRouter?: string): string[] {
    const functionKey = getFunctionKey(functionId, functionRouter)
    const data = getFunctionData(functionKey)
    return Array.from(data.keys())
  }
  
  /**
   * 获取当前函数的数据（用于向后兼容）
   */
  const data = computed(() => {
    return getFunctionData(currentFunctionKey.value)
  })
  
  return {
    data,
    setCurrentFunction,
    setValue,
    getValue,
    initializeField,
    getSubmitData,
    clear,
    clearAll,
    getAllFieldPaths
  }
})
