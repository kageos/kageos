/**
 * 最小值验证器
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'
import { isStringField } from '../utils/fieldUtils'

export class MinValidator implements Validator {
  readonly name = 'min'
  
  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    if (rule.value === undefined) {
      return { valid: true }  // 规则配置错误，跳过
    }
    
    const minValue = typeof rule.value === 'number' ? rule.value : Number(rule.value)
    if (isNaN(minValue)) {
      return { valid: true }  // 值不是数字，跳过
    }
    
    // 判断字段类型
    const field = context.allFields.find(f => (f.field_path || f.code) === context.fieldPath)
    
    if (isStringField(field)) {
      // 字符串类型：比较长度
      const length = String(value.raw || '').length
      if (length < minValue) {
        return {
          valid: false,
          message: `长度不能少于 ${minValue} 个字符`
        }
      }
    } else {
      // 数值类型：比较大小
      const fieldValue = Number(value.raw)
      if (isNaN(fieldValue) || fieldValue < minValue) {
        return {
          valid: false,
          message: `值不能小于 ${minValue}`
        }
      }
    }
    
    return { valid: true }
  }
}

