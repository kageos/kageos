/**
 * 枚举值验证器（oneof）
 */

import type { Validator, ValidationRule, ValidationResult, ValidationContext } from '../types'
import type { FieldValue } from '../../types/field'

export class OneOfValidator implements Validator {
  readonly name = 'oneof'
  
  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult {
    if (rule.value === undefined) {
      return { valid: true }  // 规则配置错误，跳过
    }
    
    // oneof 的值格式：oneof=选项1 选项2 选项3
    // 注意：后端返回的是空格分隔的字符串，如 "普通会员 vip会员"
    // 但选项值可能包含空格（如 "vip会员"），所以不能简单 split(' ')
    // 实际上，oneof 的值应该是：oneof=选项1,选项2,选项3（逗号分隔）
    // 但根据后端示例，似乎是空格分隔，这里先按空格处理
    
    // 处理空格分隔的选项（如果值中包含空格，需要特殊处理）
    // 简单实现：先按空格分割，然后过滤空字符串
    const optionsStr = String(rule.value)
    const validOptions = optionsStr.split(/\s+/).filter(opt => opt.length > 0)
    
    const fieldValue = String(value.raw || '')
    
    if (!validOptions.includes(fieldValue)) {
      return {
        valid: false,
        message: `值必须是以下选项之一：${validOptions.join('、')}`
      }
    }
    
    return { valid: true }
  }
}

