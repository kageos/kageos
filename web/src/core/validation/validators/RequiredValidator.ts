/**
 * 必填验证器
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'
import { isEmpty, getFieldName, createRequiredErrorMessage } from '../utils/fieldUtils'

export class RequiredValidator implements Validator {
  readonly name = 'required'
  
  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    if (isEmpty(value)) {
      const fieldName = getFieldName(context)
      return {
        valid: false,
        message: createRequiredErrorMessage(fieldName)
      }
    }
    
    return { valid: true }
  }
}

