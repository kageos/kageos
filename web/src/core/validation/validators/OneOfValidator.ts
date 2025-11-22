/**
 * 枚举值验证器（oneof）
 * 
 * 支持两种格式：
 * 1. 选项值不包含空格：oneof=cat dog bird
 * 2. 选项值包含空格（用单引号括起来）：oneof='small size' 'medium size' 'large size'
 * 
 * 解析逻辑：
 * - 先提取单引号括起来的内容（包含空格的选项）
 * - 然后按空格分割剩余部分（不包含空格的选项）
 * - 合并两部分结果
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
    
    const optionsStr = String(rule.value)
    const validOptions = this.parseOneOfOptions(optionsStr)
    
    const fieldValue = String(value.raw || '')
    
    if (!validOptions.includes(fieldValue)) {
      return {
        valid: false,
        message: `值必须是以下选项之一：${validOptions.join('、')}`
      }
    }
    
    return { valid: true }
  }
  
  /**
   * 解析 oneof 选项字符串
   * 
   * 处理规则：
   * 1. 单引号括起来的内容作为一个选项（如 'small size'）
   * 2. 剩余部分按空格分割（如 cat dog bird）
   * 
   * @param optionsStr oneof 的值，如 "cat dog bird" 或 "'small size' 'medium size'"
   * @returns 选项数组
   */
  private parseOneOfOptions(optionsStr: string): string[] {
    const options: string[] = []
    let currentIndex = 0
    const str = optionsStr.trim()
    
    while (currentIndex < str.length) {
      // 跳过空格
      if (str[currentIndex] === ' ') {
        currentIndex++
        continue
      }
      
      // 如果遇到单引号，提取单引号内的内容
      if (str[currentIndex] === "'") {
        const endQuoteIndex = str.indexOf("'", currentIndex + 1)
        if (endQuoteIndex === -1) {
          // 没有找到结束引号，将剩余部分作为一个选项
          const option = str.substring(currentIndex + 1).trim()
          if (option) {
            options.push(option)
          }
          break
        }
        
        // 提取单引号内的内容（不包含引号本身）
        const option = str.substring(currentIndex + 1, endQuoteIndex).trim()
        if (option) {
          options.push(option)
        }
        
        currentIndex = endQuoteIndex + 1
      } else {
        // 没有引号，找到下一个空格或字符串结尾
        let nextSpaceIndex = str.indexOf(' ', currentIndex)
        if (nextSpaceIndex === -1) {
          nextSpaceIndex = str.length
        }
        
        const option = str.substring(currentIndex, nextSpaceIndex).trim()
        if (option) {
          options.push(option)
        }
        
        currentIndex = nextSpaceIndex
      }
    }
    
    return options.filter(opt => opt.length > 0)
  }
}

