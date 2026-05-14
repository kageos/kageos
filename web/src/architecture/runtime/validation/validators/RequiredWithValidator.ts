/**
 * 条件必填验证器（required_with）
 * 
 * 示例：required_with=Email
 * 当 Email 字段有值时，当前字段必填
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '@/architecture/domain/types/field'
import {
  isEmpty as isEmptyValue,
  getFieldName,
  createRequiredErrorMessage,
  findFieldInContext,
} from '../utils/fieldUtils'
import { evaluatePresenceRule } from '../utils/presenceRules'

export class RequiredWithValidator implements Validator {
  readonly name = 'required_with'
  
  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    if (evaluatePresenceRule(rule, context)) {
      // 其他字段有值，当前字段必填
      // 🔥 从 context 中查找字段配置，用于 table 类型字段的空行过滤
      const field = findFieldInContext(context)
      if (isEmptyValue(value, field || undefined)) {
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
