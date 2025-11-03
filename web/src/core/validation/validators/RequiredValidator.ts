/**
 * 必填验证器
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
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
      // 🔥 获取当前字段的 name，生成更友好的错误消息
      const currentField = context.allFields.find(f => 
        (f.field_path || f.code) === context.fieldPath
      )
      const fieldName = currentField?.name || '此字段'
      
      return {
        valid: false,
        message: `${fieldName}必填`
      }
    }
    
    return { valid: true }
  }
}

