/**
 * 条件必填验证器（required_unless）
 * 
 * 示例：required_unless=MemberType vip会员
 * 除非 MemberType 字段的值等于 'vip会员'，否则当前字段必填
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'
import { isEmpty, getFieldName, createRequiredErrorMessage, findFieldInContext } from '../utils/fieldUtils'
import { evaluatePresenceRule } from '../utils/presenceRules'

export class RequiredUnlessValidator implements Validator {
  readonly name = 'required_unless'
  
  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    if (evaluatePresenceRule(rule, context)) {
      // 🔥 从 context 中查找字段配置，用于 table 类型字段的空行过滤
      const field = findFieldInContext(context)
      if (isEmpty(value, field || undefined)) {
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
