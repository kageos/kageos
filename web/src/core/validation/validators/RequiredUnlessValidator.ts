/**
 * 条件必填验证器（required_unless）
 * 
 * 示例：required_unless=MemberType vip会员
 * 除非 MemberType 字段的值等于 'vip会员'，否则当前字段必填
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'
import { isEmpty, getFieldName, createRequiredErrorMessage, findFieldInContext } from '../utils/fieldUtils'

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
  
  /**
   * 判断条件是否满足
   */
  private isConditionMet(fieldValue: FieldValue, expectedValue: string | number): boolean {
    const actualValue = fieldValue.raw
    const expectedValueStr = String(expectedValue)
    
    if (typeof actualValue === 'boolean') {
      return String(actualValue) === expectedValueStr || actualValue === (expectedValueStr === 'true')
    }
    
    if (typeof actualValue === 'number') {
      const expectedNum = Number(expectedValueStr)
      return !isNaN(expectedNum) && actualValue === expectedNum
    }
    
    return String(actualValue) === expectedValueStr
  }
}
