/**
 * 条件必填验证器（required_without_all）
 *
 * 示例：required_without_all=Phone Email
 * 当 Phone 和 Email 都为空时，当前字段必填
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

export class RequiredWithoutAllValidator implements Validator {
  readonly name = 'required_without_all'

  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    if (evaluatePresenceRule(rule, context)) {
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
