/**
 * 条件必填验证器（required_unless）
 * 
 * 示例：required_unless=MemberType vip会员
 * 除非 MemberType 字段的值等于 'vip会员'，否则当前字段必填
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'

export class RequiredUnlessValidator implements Validator {
  readonly name = 'required_unless'
  
  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    // 检查规则配置
    if (!rule.field || rule.value === undefined) {
      return { valid: true }  // 配置错误，跳过验证
    }
    
    // 🔥 通过 formManager 获取其他字段的值（解耦设计）
    const otherFieldValue = context.formManager.getValue(rule.field)
    
    // 判断条件是否满足（unless 是相反的逻辑）
    const conditionMet = this.isConditionMet(otherFieldValue, rule.value)
    
    // required_unless：除非条件满足，否则必填
    // 即：条件不满足时，当前字段必填
    if (!conditionMet) {
      const isEmpty = value.raw === null ||
                     value.raw === undefined ||
                     value.raw === '' ||
                     (Array.isArray(value.raw) && value.raw.length === 0)
      
      if (isEmpty) {
        // 🔥 获取当前字段的 name，生成更友好的错误消息
        const currentField = context.allFields.find(f => 
          (f.field_path || f.code) === context.fieldPath
        )
        const fieldName = currentField?.name || '此字段'
        
        return {
          valid: false,
          message: `${fieldName}必填`
        }
      }
    }
    
    return { valid: true }
  }
  
  /**
   * 判断条件是否满足
   */
  private isConditionMet(fieldValue: FieldValue, expectedValue: string): boolean {
    const actualValue = fieldValue.raw
    
    if (typeof actualValue === 'boolean') {
      return String(actualValue) === expectedValue || actualValue === (expectedValue === 'true')
    }
    
    if (typeof actualValue === 'number') {
      const expectedNum = Number(expectedValue)
      return !isNaN(expectedNum) && actualValue === expectedNum
    }
    
    return String(actualValue) === expectedValue
  }
}

