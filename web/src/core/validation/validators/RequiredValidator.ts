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
      // 🔥 优先使用字段配置中的 name（中文名称）
      // 如果找不到字段配置，尝试从 allFields 中查找
      let fieldName = field?.name
      if (!fieldName) {
        // 如果 findFieldInContext 没找到，尝试直接从 allFields 中查找
        const foundField = context.allFields.find(f => f.code === context.fieldPath)
        fieldName = foundField?.name
      }
      // 如果还是找不到，使用 getFieldName 的 fallback
      if (!fieldName) {
        fieldName = getFieldName(context, '此字段')
      }
      
      // 🔥 再次检查值是否真的为空（防止时序问题）
      // 如果 value.raw 有值，说明字段已经填写，不应该报错
      if (value.raw !== null && value.raw !== undefined && value.raw !== '') {
        return { valid: true }
      }
      
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

