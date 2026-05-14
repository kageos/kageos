/**
 * 必填验证器
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '@/architecture/domain/types/field'
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
      // 🔥 再次检查值是否真的为空（防止时序问题）
      // 如果 value.raw 有值，说明字段已经填写，不应该报错
      if (value.raw !== null && value.raw !== undefined && value.raw !== '') {
        return { valid: true }
      }
      
      // 🔥 使用统一的 getFieldName 函数获取字段名称（中文名称）
      const fieldName = getFieldName(context, '此字段')
      const errorMessage = createRequiredErrorMessage(fieldName)
      
      return {
        valid: false,
        message: errorMessage,
        field: field || undefined
      }
    }
    
    return { valid: true }
  }
}

