/**
 * 值类型转换工具
 * 
 * 用于将字符串值转换为对应的数据类型
 * 遵循单一职责原则，统一处理类型转换逻辑
 */

import { FieldDataType } from '../../constants/select'
import { Logger } from '../../utils/logger'

/**
 * 将字符串值转换为指定类型
 * 
 * @param value 原始值（可能是字符串）
 * @param targetType 目标类型
 * @param componentName 组件名称（用于日志）
 * @returns 转换后的值，如果转换失败返回原值
 */
export function convertValueToType(
  value: string | number | boolean,
  targetType: string,
  componentName: string = 'ValueConverter'
): any {
  // 如果已经是目标类型，直接返回
  if (typeof value !== 'string') {
    return value
  }

  // 如果目标类型是字符串，直接返回
  if (targetType === FieldDataType.STRING) {
    return value
  }

  try {
    switch (targetType) {
      case FieldDataType.INT:
      case FieldDataType.INTEGER:
        const intValue = parseInt(value, 10)
        if (isNaN(intValue)) {
          Logger.warn(componentName, `无法将 "${value}" 转换为整数`)
          return value
        }
        return intValue

      case FieldDataType.FLOAT:
      case FieldDataType.NUMBER:
        const floatValue = parseFloat(value)
        if (isNaN(floatValue)) {
          Logger.warn(componentName, `无法将 "${value}" 转换为浮点数`)
          return value
        }
        return floatValue

      case FieldDataType.BOOL:
      case FieldDataType.BOOLEAN:
        return value === 'true' || value === '1' || value === 1 || value === true

      default:
        // 未知类型，保持原样
        return value
    }
  } catch (error) {
    Logger.error(componentName, `类型转换失败: ${value} -> ${targetType}`, error)
    return value
  }
}

