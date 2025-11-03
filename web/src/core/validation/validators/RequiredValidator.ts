/**
 * 必填验证器
 */

import type { ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'

export class RequiredValidator implements Validator {
  readonly name = 'required'
  
  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    // 判断是否为空
    const isEmpty = value.raw === null ||
                   value.raw === undefined ||
                   value.raw === '' ||
                   (Array.isArray(value.raw) && value.raw.length === 0)
    
    if (isEmpty) {
      return {
        valid: false,
        message: '此字段为必填项'
      }
    }
    
    return { valid: true }
  }
}

