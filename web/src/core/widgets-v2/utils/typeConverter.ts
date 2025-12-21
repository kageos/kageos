/**
 * 类型转换工具
 * 
 * 🔥 统一处理所有类型转换逻辑，避免硬编码和重复代码
 * 🔥 符合依赖倒置原则：使用常量而非硬编码字符串
 */

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

