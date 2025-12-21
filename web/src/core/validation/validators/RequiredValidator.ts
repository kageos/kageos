/**
 * 必填验证器
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'
import { isEmpty, getFieldName, createRequiredErrorMessage, findFieldInContext } from '../utils/fieldUtils'

export class RequiredValidator implements Validator {
  readonly name = 'required'
  
  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    // 🔥 从 context 中查找字段配置，用于 table 类型字段的空行过滤
    const field = findFieldInContext(context)
    if (isEmpty(value, field || undefined)) {
      const fieldName = getFieldName(context)
      return {
        valid: false,
        message: createRequiredErrorMessage(fieldName)
      }
    }
    
    return { valid: true }
  }
}

