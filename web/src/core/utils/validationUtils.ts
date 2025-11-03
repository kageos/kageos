/**
 * 验证工具函数
 */

import type { FieldConfig } from '../types/field'

/**
 * 检查字段是否为必填（仅无条件必填）
 * 
 * @param field 字段配置
 * @returns 是否为必填
 */
export function isFieldRequired(field: FieldConfig): boolean {
  if (!field.validation) {
    return false
  }
  
  // 解析 validation 字符串
  const rules = field.validation.split(',').map(r => r.trim())
  
  // 检查是否有 omitempty，如果有则不是必填
  if (rules.includes('omitempty')) {
    return false
  }
  
  // 只检查 unconditional required（没有 = 号的 required）
  return rules.some(r => r === 'required')
}

/**
 * 检查字段是否有任何必填规则（包括条件必填）
 * 
 * @param field 字段配置
 * @returns 是否有任何必填规则
 */
export function hasAnyRequiredRule(field: FieldConfig): boolean {
  if (!field.validation) {
    return false
  }
  
  const rules = field.validation.split(',').map(r => r.trim())
  
  // 检查是否有 omitempty，如果有则不是必填
  if (rules.includes('omitempty')) {
    return false
  }
  
  // 检查是否有任何 required 相关的规则
  return rules.some(r => 
    r === 'required' || 
    r.startsWith('required_if') || 
    r.startsWith('required_unless') ||
    r.startsWith('required_with') ||
    r.startsWith('required_without')
  )
}

