/**
 * 条件必填验证器（required_with）
 * 
 * 示例：required_with=Email
 * 当 Email 字段有值时，当前字段必填
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'

export class RequiredWithValidator implements Validator {
  readonly name = 'required_with'
  
  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    // 检查规则配置
    if (!rule.field) {
      return { valid: true }  // 配置错误，跳过验证
    }
    
    // 🔥 通过 formManager 获取其他字段的值（解耦设计）
    const otherFieldValue = context.formManager.getValue(rule.field)
    
    // 判断其他字段是否有值
    const otherFieldHasValue = otherFieldValue.raw !== null &&
                              otherFieldValue.raw !== undefined &&
                              otherFieldValue.raw !== '' &&
                              !(Array.isArray(otherFieldValue.raw) && otherFieldValue.raw.length === 0)
    
    if (otherFieldHasValue) {
      // 其他字段有值，当前字段必填
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
    }
    
    return { valid: true }
  }
}

