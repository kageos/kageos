/**
 * 条件必填验证器（required_without）
 * 
 * 示例：required_without=Mobile
 * 当 Mobile 字段无值时，当前字段必填
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'

export class RequiredWithoutValidator implements Validator {
  readonly name = 'required_without'
  
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
    
    // 判断其他字段是否为空
    const otherFieldIsEmpty = otherFieldValue.raw === null ||
                             otherFieldValue.raw === undefined ||
                             otherFieldValue.raw === '' ||
                             (Array.isArray(otherFieldValue.raw) && otherFieldValue.raw.length === 0)
    
    if (otherFieldIsEmpty) {
      // 其他字段为空，当前字段必填
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
    }
    
    return { valid: true }
  }
}

