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
      // 注意：context.fieldPath 可能是 field_path 或 code
      const currentField = context.allFields.find(f => {
        const fieldPath = f.field_path || f.code
        return fieldPath === context.fieldPath
      })
      
      // 如果找不到，尝试只匹配 code
      const foundField = currentField || context.allFields.find(f => f.code === context.fieldPath)
      const fieldName = foundField?.name || '此字段'
      
      return {
        valid: false,
        message: `${fieldName}必填`
      }
    }
    
    return { valid: true }
  }
}

