/**
 * 类型转换工具
 * 
 * 🔥 统一处理所有类型转换逻辑，避免硬编码和重复代码
 * 🔥 符合依赖倒置原则：使用常量而非硬编码字符串
 * 
 * ⚠️ 重要：类型转换是硬性要求
 * - 函数详情中的 `data.type` 字段明确说明了提交时应该使用的类型
 * - 不符合类型会导致后端解析失败
 * - 所有组件都必须使用这些工具函数进行类型转换
 * 
 * 使用场景：
 * 1. URL 参数初始化：URL 参数都是字符串，需要根据 `field.data.type` 转换
 * 2. 提交数据：提交时需要根据 `field.data.type` 转换
 * 3. 回调接口的 request 参数：需要根据字段的 `field.data.type` 转换
 * 4. 组件显示：需要正确匹配类型（数字 vs 字符串）
 */

import type { FieldConfig, FieldValue } from '../../types/field'
import { DataType } from '../../constants/widget'
import { convertValueToType } from './valueConverter'

/**
 * 转换基础类型值（用于 URL 参数等场景）
 * 
 * @param value 原始值
 * @param fieldType 字段类型（如 'int', 'float', 'bool' 等）
 * @returns 转换后的值
 */
export function convertBasicType(value: any, fieldType: string | undefined | null): any {
  const type = fieldType || DataType.STRING
  
  // 使用统一的 convertValueToType 工具
  return convertValueToType(String(value), type, 'TypeConverter')
}

/**
 * 转换数组类型值（用于 multiselect 等场景）
 * 
 * @param value 原始值（可能是逗号分隔的字符串或数组）
 * @param fieldType 字段类型（如 '[]int', '[]string' 等）
 * @returns 转换后的数组
 */
export function convertArrayType(
  value: any,
  fieldType: string | undefined | null
): any[] {
  const type = fieldType || DataType.STRINGS
  
  // 检查是否是数组类型
  if (!type.startsWith('[]')) {
    // 不是数组类型，转换为单元素数组
    return [convertBasicType(value, type)]
  }
  
  const elementType = type.slice(2)  // 获取元素类型，如 []int -> int
  
  // 如果 value 是字符串，尝试按逗号分隔转换为数组
  if (typeof value === 'string') {
    const strValue = String(value)
    if (strValue.includes(',')) {
      const stringArray = strValue.split(',').map(s => s.trim()).filter(Boolean)
      return stringArray.map(s => convertBasicType(s, elementType))
    } else {
      // 单个值，转换为单元素数组
      return [convertBasicType(strValue, elementType)]
    }
  } else if (Array.isArray(value)) {
    // 如果已经是数组，根据元素类型转换
    return value.map(v => convertBasicType(v, elementType))
  } else {
    // 单个值，转换为单元素数组
    return [convertBasicType(value, elementType)]
  }
}

/**
 * 判断字段类型是否需要基础类型转换
 * 
 * @param fieldType 字段类型
 * @returns 是否需要转换
 */
export function needsBasicTypeConversion(fieldType: string | undefined | null): boolean {
  if (!fieldType) return false
  
  return fieldType === DataType.INT ||
         fieldType === 'integer' ||  // 兼容别名
         fieldType === DataType.FLOAT ||
         fieldType === 'number' ||  // 兼容别名
         fieldType === DataType.BOOL ||
         fieldType === 'boolean'  // 兼容别名
}

/**
 * 🔥 根据字段配置转换值（统一入口）
 * 
 * ⚠️ 重要：这是类型转换的统一入口，所有组件都应该使用这个函数
 * 
 * 根据 `field.data.type` 将值转换为正确的类型：
 * - `int` / `integer`: 转换为数字
 * - `float` / `number`: 转换为浮点数
 * - `bool` / `boolean`: 转换为布尔值
 * - `[]int`: 转换为数字数组
 * - `[]string`: 转换为字符串数组
 * - `string`: 保持字符串
 * 
 * @param value 原始值（可能是字符串、数字、数组等）
 * @param field 字段配置（必须包含 `data.type`）
 * @returns 转换后的值
 * 
 * @example
 * // 基础类型
 * convertValueByFieldType('1', { data: { type: 'int' } })  // 1
 * convertValueByFieldType('1.5', { data: { type: 'float' } })  // 1.5
 * convertValueByFieldType('true', { data: { type: 'bool' } })  // true
 * 
 * // 数组类型
 * convertValueByFieldType('1,2,3', { data: { type: '[]int' } })  // [1, 2, 3]
 * convertValueByFieldType(['1', '2'], { data: { type: '[]int' } })  // [1, 2]
 */
export function convertValueByFieldType(value: any, field: FieldConfig): any {
  const fieldType = field.data?.type
  
  if (!fieldType) {
    // 没有类型配置，保持原样
    return value
  }
  
  // 数组类型
  if (fieldType.startsWith('[]')) {
    return convertArrayType(value, fieldType)
  }
  
  // 基础类型
  return convertBasicType(value, fieldType)
}

/**
 * 🔥 将表单数据转换为请求格式，并根据字段类型进行转换（统一函数）
 * 
 * ⚠️ 重要：这是提交数据和回调接口 request 参数转换的统一函数
 * - 所有组件都应该使用这个函数，而不是自己实现
 * - 确保所有字段都根据 `field.data.type` 正确转换
 * 
 * @param formData 表单数据（FieldValue 格式或 raw 值格式）
 * @param functionDetail 函数详情（必须包含 `request` 字段数组）
 * @returns 转换后的请求数据（所有值都根据字段类型转换）
 * 
 * @example
 * const formData = {
 *   topic_id: { raw: '1', display: '主题1', meta: {} },
 *   option_ids: { raw: '1,2', display: '选项1,选项2', meta: {} }
 * }
 * const functionDetail = {
 *   request: [
 *     { code: 'topic_id', data: { type: 'int' } },
 *     { code: 'option_ids', data: { type: '[]int' } }
 *   ]
 * }
 * convertFormDataToRequestByType(formData, functionDetail)
 * // { topic_id: 1, option_ids: [1, 2] }
 */
export function convertFormDataToRequestByType(
  formData: Record<string, FieldValue | any>,
  functionDetail: { request?: FieldConfig[] }
): Record<string, any> {
  if (!functionDetail || !functionDetail.request || !Array.isArray(functionDetail.request)) {
    // 没有字段配置，尝试直接提取 raw 值
    const result: Record<string, any> = {}
    Object.keys(formData).forEach(key => {
      const value = formData[key]
      // 如果是 FieldValue 格式，提取 raw；否则直接使用
      result[key] = value && typeof value === 'object' && 'raw' in value ? value.raw : value
    })
    return result
  }
  
  const request: Record<string, any> = {}
  const requestFields = functionDetail.request
  
  // 构建字段配置映射（code -> field）
  const fieldMap = new Map<string, FieldConfig>()
  requestFields.forEach((field: FieldConfig) => {
    fieldMap.set(field.code, field)
  })
  
  // 遍历表单数据，根据字段类型进行转换
  Object.keys(formData).forEach(key => {
    const fieldValue = formData[key]
    const field = fieldMap.get(key)
    
    // 提取 raw 值（如果是 FieldValue 格式）
    const rawValue = fieldValue && typeof fieldValue === 'object' && 'raw' in fieldValue
      ? fieldValue.raw
      : fieldValue
    
    if (rawValue === null || rawValue === undefined) {
      // 值为空，直接使用
      request[key] = rawValue
      return
    }
    
    if (!field) {
      // 没有字段配置，直接使用 raw 值
      request[key] = rawValue
      return
    }
    
    // 🔥 根据字段类型进行转换（这是关键！）
    request[key] = convertValueByFieldType(rawValue, field)
  })
  
  return request
}

/**
 * 🔥 构建支持多种类型匹配的选项映射（用于初始化器）
 * 
 * 问题：回调接口返回的 item.value 可能是数字，但组件中的值可能是字符串
 * 解决方案：同时使用数字和字符串作为 key，确保类型匹配
 * 
 * @param items 回调接口返回的选项列表
 * @returns 包含 optionMap 和 displayInfoMap 的对象
 * 
 * @example
 * const { optionMap, displayInfoMap } = buildOptionMaps(response.items)
 * // optionMap 支持数字和字符串作为 key
 * optionMap.get(1) === optionMap.get('1')  // true
 */
export function buildOptionMaps(items: Array<{ value: any; label?: string; display_info?: any; displayInfo?: any }>): {
  optionMap: Map<any, string>
  displayInfoMap: Map<any, any>
} {
  const optionMap = new Map<any, string>()
  const displayInfoMap = new Map<any, any>()
  
  if (items && Array.isArray(items)) {
    items.forEach((item: any) => {
      const itemValue = item.value
      const label = item.label || String(itemValue)
      const displayInfo = item.display_info || item.displayInfo
      
      // 🔥 同时支持数字和字符串作为 key，确保类型匹配
      optionMap.set(itemValue, label)
      
      // 如果 value 是数字，同时设置字符串版本作为 key
      if (typeof itemValue === 'number') {
        optionMap.set(String(itemValue), label)
      } else if (typeof itemValue === 'string') {
        // 如果 value 是字符串，尝试转换为数字作为 key
        const numValue = parseInt(itemValue, 10)
        if (!isNaN(numValue)) {
          optionMap.set(numValue, label)
        }
      }
      
      // 同样处理 displayInfo
      if (displayInfo) {
        displayInfoMap.set(itemValue, displayInfo)
        if (typeof itemValue === 'number') {
          displayInfoMap.set(String(itemValue), displayInfo)
        } else if (typeof itemValue === 'string') {
          const numValue = parseInt(itemValue, 10)
          if (!isNaN(numValue)) {
            displayInfoMap.set(numValue, displayInfo)
          }
        }
      }
    })
  }
  
  return { optionMap, displayInfoMap }
}

/**
 * 🔥 从选项映射中获取标签（支持多种类型匹配）
 * 
 * @param optionMap 选项映射
 * @param value 要查找的值
 * @returns 标签，如果找不到则返回值的字符串形式
 */
export function getOptionLabelFromMap(optionMap: Map<any, string>, value: any): string {
  // 🔥 尝试多种方式匹配：直接匹配、字符串匹配、数字匹配
  let label = optionMap.get(value)
  if (label) {
    return label
  }
  
  // 尝试字符串匹配
  label = optionMap.get(String(value))
  if (label) {
    return label
  }
  
  // 如果 value 是字符串，尝试转换为数字后匹配
  if (typeof value === 'string') {
    const numVal = parseInt(value, 10)
    if (!isNaN(numVal)) {
      label = optionMap.get(numVal)
      if (label) {
        return label
      }
    }
  }
  
  // 如果 value 是数字，尝试字符串匹配
  if (typeof value === 'number') {
    label = optionMap.get(String(value))
    if (label) {
      return label
    }
  }
  
  // 如果都找不到，返回值的字符串形式
  return String(value)
}

