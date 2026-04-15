import type { FieldValue } from '../../types/field'
import type { ValidationContext, ValidationResult, ValidationRule, Validator } from '../types'
import {
  createExcludedErrorMessage,
  findFieldInContext,
  getFieldName,
  isEmpty,
} from '../utils/fieldUtils'
import { evaluatePresenceRule } from '../utils/presenceRules'

export class ExcludedConditionalValidator implements Validator {
  constructor(readonly name:
    | 'excluded_if'
    | 'excluded_unless'
    | 'excluded_with'
    | 'excluded_with_all'
    | 'excluded_without'
    | 'excluded_without_all'
  ) {}

  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    if (!evaluatePresenceRule(rule, context)) {
      return { valid: true }
    }

    const field = findFieldInContext(context)
    if (isEmpty(value, field || undefined)) {
      return { valid: true }
    }

    return {
      valid: false,
      message: createExcludedErrorMessage(getFieldName(context))
    }
  }
}
