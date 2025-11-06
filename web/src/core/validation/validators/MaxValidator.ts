/**
 * 最大值验证器
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'
import { isStringField } from '../utils/fieldUtils'

export class MaxValidator implements Validator {
  readonly name = 'max'
  
  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    if (rule.value === undefined) {
      return { valid: true }  // 规则配置错误，跳过
    }
    
    const maxValue = typeof rule.value === 'number' ? rule.value : Number(rule.value)
    if (isNaN(maxValue)) {
      return { valid: true }  // 值不是数字，跳过
    }
    
    // 判断字段类型
    const field = context.allFields.find(f => (f.field_path || f.code) === context.fieldPath)
    
    // 🔥 数组类型（table 字段）：比较数组长度
    if (Array.isArray(value.raw)) {
      const length = value.raw.length
      if (length > maxValue) {
        return {
          valid: false,
          message: `最多只能有 ${maxValue} 项`
        }
      }
      return { valid: true }
    }
    
    if (isStringField(field)) {
      // 字符串类型：比较长度
      const length = String(value.raw || '').length
      if (length > maxValue) {
        return {
          valid: false,
          message: `长度不能超过 ${maxValue} 个字符`
        }
      }
    } else {
      // 数值类型：比较大小
      const fieldValue = Number(value.raw)
      if (isNaN(fieldValue) || fieldValue > maxValue) {
        return {
          valid: false,
          message: `值不能大于 ${maxValue}`
        }
      }
    }
    
    return { valid: true }
  }
}

