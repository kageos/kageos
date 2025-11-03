/**
 * 邮箱格式验证器
 */

import type { ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'

export class EmailValidator implements Validator {
  readonly name = 'email'
  
  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    // 如果值为空，跳过验证（由 required 验证器处理）
    if (value.raw === null || value.raw === undefined || value.raw === '') {
      return { valid: true }
    }
    
    const email = String(value.raw)
    // 简单的邮箱格式验证（可以使用更复杂的正则）
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    
    if (!emailRegex.test(email)) {
      return {
        valid: false,
        message: '请输入有效的邮箱地址'
      }
    }
    
    return { valid: true }
  }
}

