/**
 * 条件必填验证器（required_with）
 * 
 * 示例：required_with=Email
 * 当 Email 字段有值时，当前字段必填
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'
import { isEmpty as isEmptyValue, getFieldName, createRequiredErrorMessage } from '../utils/fieldUtils'

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
    const otherFieldHasValue = !isEmptyValue(otherFieldValue)
    
    if (otherFieldHasValue) {
      // 其他字段有值，当前字段必填
      if (isEmptyValue(value)) {
        const fieldName = getFieldName(context)
        return {
          valid: false,
          message: createRequiredErrorMessage(fieldName)
        }
      }
    }
    
    return { valid: true }
  }
}

