/**
 * 条件渲染评估器
 * 
 * 根据 validation 中的条件验证规则，判断字段是否应该显示/隐藏
 * 
 * 规则：
 * - required_if=Field value: 当字段等于指定值时显示（否则隐藏）
 * - required_unless=Field value: 除非字段等于指定值，否则显示（等于时隐藏）
 * - required_with=Field: 当字段有值时显示（无值时隐藏）
 * - required_without=Field: 当字段无值时显示（有值时隐藏）
 */

import type { FieldConfig, FieldValue } from '../types/field'
import type { ReactiveFormDataManager } from '../managers/ReactiveFormDataManager'

/**
 * 评估字段是否应该显示
 * 
 * @param field 字段配置
 * @param formManager 表单数据管理器
 * @param allFields 所有字段配置（用于查找其他字段）
 * @returns 是否应该显示
 */
export function shouldShowField(
  field: FieldConfig,
  formManager: ReactiveFormDataManager,
  allFields: FieldConfig[]
): boolean {
  if (!field.validation) {
    return true  // 无验证规则，默认显示
  }
  
  // 解析 validation 字符串，查找条件渲染规则
  const rules = parseConditionalRules(field.validation, allFields)
  
  // 如果没有条件规则，默认显示
  if (rules.length === 0) {
    return true
  }
  
  // 评估所有条件规则（OR 关系：任一条件满足即显示）
  // 但通常一个字段只有一个条件规则
  for (const rule of rules) {
    if (evaluateCondition(rule, formManager)) {
      return true
    }
  }
  
  // 所有条件都不满足，隐藏字段
  return false
}

/**
 * 条件规则类型
 */
interface ConditionalRule {
  type: 'required_if' | 'required_unless' | 'required_with' | 'required_without'
  field: string  // 引用的字段 code
  value?: string  // 期望的值（required_if/required_unless 需要）
}

/**
 * 解析条件渲染规则
 */
function parseConditionalRules(validation: string, allFields: FieldConfig[]): ConditionalRule[] {
  const rules: ConditionalRule[] = []
  const parts = validation.split(',').map(s => s.trim())
  
  // 构建字段名映射表（Go字段名 -> code）
  const fieldNameMap = new Map<string, string>()
  for (const f of allFields) {
    if (f.field_name && f.code) {
      fieldNameMap.set(f.field_name, f.code)
    }
  }
  
  for (const part of parts) {
    if (!part || part === 'omitempty' || !part.includes('=')) {
      continue
    }
    
    const [type, valueStr] = part.split('=', 2)
    const typeTrimmed = type.trim()
    const valueTrimmed = valueStr.trim()
    
    // 只处理条件渲染相关的规则
    if (typeTrimmed === 'required_if' || typeTrimmed === 'required_unless') {
      const spaceIndex = valueTrimmed.indexOf(' ')
      if (spaceIndex > 0) {
        const goFieldName = valueTrimmed.substring(0, spaceIndex).trim()
        const value = valueTrimmed.substring(spaceIndex + 1).trim()
        const code = fieldNameMap.get(goFieldName) || goFieldName
        
        rules.push({
          type: typeTrimmed as 'required_if' | 'required_unless',
          field: code,
          value
        })
      }
    } else if (typeTrimmed === 'required_with' || typeTrimmed === 'required_without') {
      const goFieldName = valueTrimmed
      const code = fieldNameMap.get(goFieldName) || goFieldName
      
      rules.push({
        type: typeTrimmed as 'required_with' | 'required_without',
        field: code
      })
    }
  }
  
  return rules
}

/**
 * 评估条件是否满足
 */
function evaluateCondition(
  rule: ConditionalRule,
  formManager: ReactiveFormDataManager
): boolean {
  const otherFieldValue = formManager.getValue(rule.field)
  
  switch (rule.type) {
    case 'required_if':
      // required_if=Field value: 当字段等于指定值时显示
      if (rule.value === undefined) return true
      return isValueEqual(otherFieldValue.raw, rule.value)
      
    case 'required_unless':
      // required_unless=Field value: 除非字段等于指定值，否则显示
      if (rule.value === undefined) return true
      return !isValueEqual(otherFieldValue.raw, rule.value)
      
    case 'required_with':
      // required_with=Field: 当字段有值时显示
      return !isEmpty(otherFieldValue)
      
    case 'required_without':
      // required_without=Field: 当字段无值时显示
      return isEmpty(otherFieldValue)
      
    default:
      return true
  }
}

/**
 * 判断值是否为空
 */
function isEmpty(fieldValue: FieldValue): boolean {
  return fieldValue.raw === null ||
         fieldValue.raw === undefined ||
         fieldValue.raw === '' ||
         (Array.isArray(fieldValue.raw) && fieldValue.raw.length === 0)
}

// 注意：此处的 isEmpty 与 validation/utils/fieldUtils.ts 中的 isEmpty 功能相同
// 但为了保持模块独立性，暂时不合并。如果后续需要统一，可以从 fieldUtils 导入

/**
 * 判断两个值是否相等
 */
function isValueEqual(actual: any, expected: string): boolean {
  if (typeof actual === 'boolean') {
    return String(actual) === expected || actual === (expected === 'true')
  }
  
  if (typeof actual === 'number') {
    const expectedNum = Number(expected)
    return !isNaN(expectedNum) && actual === expectedNum
  }
  
  return String(actual) === expected
}

