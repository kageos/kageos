/**
 * 必填验证器
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'

export class RequiredValidator implements Validator {
  readonly name = 'required'
  
  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    // 判断是否为空
    const isEmpty = value.raw === null ||
                   value.raw === undefined ||
                   value.raw === '' ||
                   (Array.isArray(value.raw) && value.raw.length === 0)
    
    if (isEmpty) {
      // 🔥 获取当前字段的 name，生成更友好的错误消息
      // 注意：context.fieldPath 可能是 field_path 或 code
      // 先尝试匹配 field_path，再尝试匹配 code
      let foundField = context.allFields.find(f => {
        if (f.field_path) {
          return f.field_path === context.fieldPath
        }
        return f.code === context.fieldPath
      })
      
      // 如果还找不到，尝试只匹配 code（可能 field_path 为空）
      if (!foundField) {
        foundField = context.allFields.find(f => f.code === context.fieldPath)
      }
      
      const fieldName = foundField?.name || '此字段'
      
      // 🔥 调试日志（开发时使用）
      if (!foundField) {
        console.warn(`[RequiredValidator] 未找到字段: fieldPath=${context.fieldPath}, allFields=`, 
          context.allFields.map(f => ({ code: f.code, field_path: f.field_path, name: f.name }))
        )
      }
      
      return {
        valid: false,
        message: `${fieldName}必填`
      }
    }
    
    return { valid: true }
  }
}

