/**
 * 验证引擎
 * 
 * 负责解析 validation 字符串，调用相应的验证器
 * 关键：将 validation 中的 Go 字段名转换为 code（JSON标签）
 */

import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'
import type { ReactiveFormDataManager } from '../managers/ReactiveFormDataManager'
import type { ValidationRule, ValidationResult, ValidationContext } from './types'
import { ValidatorRegistry } from './ValidatorRegistry'
import { Logger } from '../utils/logger'
import { parsePresenceRule } from './utils/presenceRules'

export class ValidationEngine {
  private fieldNameMap: Map<string, string>
  
  constructor(
    private registry: ValidatorRegistry,
    private formManager: ReactiveFormDataManager,
    fields: FieldConfig[]
  ) {
    // 初始化时构建字段名映射表
    this.fieldNameMap = this.buildFieldNameMap(fields)
  }
  
  /**
   * 构建字段名映射表（Go字段名 -> code）
   * 
   * 递归遍历所有字段，构建 field_name -> code 的映射
   */
  private buildFieldNameMap(fields: FieldConfig[]): Map<string, string> {
    const map = new Map<string, string>()
    
    const traverse = (fieldList: FieldConfig[]) => {
      for (const field of fieldList) {
        if (field.field_name && field.code) {
          map.set(field.field_name, field.code)
        }
        
        // 递归处理嵌套字段
        if (field.children && field.children.length > 0) {
          traverse(field.children)
        }
      }
    }
    
    traverse(fields)
    return map
  }
  
  /**
   * 验证单个字段
   * 
   * @param field 字段配置
   * @param value 字段值
   * @param allFields 所有字段配置（用于查找其他字段）
   * @returns 验证错误列表（空数组表示验证通过）
   */
  validateField(
    field: FieldConfig,
    value: FieldValue,
    allFields: FieldConfig[],
    fieldPath: string = field.code
  ): ValidationResult[] {
    if (!field.validation) {
      return []  // 无验证规则，直接通过
    }
    
    // 解析 validation 字符串（包含字段名转换）
    const rules = this.parseValidationString(field.validation)
    
    // 构建验证上下文
    // 🔥 使用 field.code 作为 fieldPath，确保 findFieldInContext 能正确找到字段配置
    const context: ValidationContext = {
      formManager: this.formManager,
      fieldPath,
      allFields
    }
    
    const errors: ValidationResult[] = []
    
    // 遍历所有规则，执行验证
    for (const rule of rules) {
      const validator = this.registry.get(rule.type)
      if (!validator) {
        // 未知验证器，跳过（可能是 omitempty 等前端不验证的规则）
        continue
      }
      
      try {
        const result = validator.validate(value, rule, context)
        if (!result.valid) {
          // 🔥 将字段信息附加到验证结果中，用于错误消息格式化
          errors.push({ ...result, field })
        }
      } catch (error) {
        Logger.error('[ValidationEngine]', `验证器 ${rule.type} 执行失败`, error)
        // 验证器执行失败，不阻止表单提交（后端会兜底）
      }
    }
    
    return errors
  }
  
  /**
   * 解析 validation 字符串
   * 
   * 将 Go 字段名替换为 code（JSON标签）
   * 
   * 示例：
   * - "required" -> [{ type: 'required' }]
   * - "required,min=2,max=20" -> [{ type: 'required' }, { type: 'min', value: 2 }, { type: 'max', value: 20 }]
   * - "required_if=MemberType vip会员" -> [{ type: 'required_if', field: 'member_type', value: 'vip会员' }]
   * 
   * 注意：忽略 omitempty（前端不需要验证，由后端处理）
   */
  private parseValidationString(validation: string): ValidationRule[] {
    const rules: ValidationRule[] = []
    const parts = validation.split(',').map(s => s.trim())
    
    for (const part of parts) {
      if (!part || part === 'omitempty') {
        continue  // 跳过空值和 omitempty
      }
      
      // 处理带参数的规则：min=2, max=20
      if (part.includes('=')) {
        const [type, valueStr] = part.split('=', 2)
        if (!type || valueStr === undefined) {
          continue
        }
        const typeTrimmed = type.trim()
        const valueTrimmed = valueStr.trim()
        
        // 判断是否是条件验证规则
        if (this.isConditionalRule(typeTrimmed)) {
          const parsedPresenceRule = parsePresenceRule(typeTrimmed, valueTrimmed, this.fieldNameMap)
          if (parsedPresenceRule) {
            rules.push({
              type: typeTrimmed,
              ...parsedPresenceRule
            })
            continue
          }

          const goFieldName = valueTrimmed
          const code = this.fieldNameMap.get(goFieldName) || goFieldName
          rules.push({ type: typeTrimmed, field: code })
      } else {
        // 普通带参数规则：min=2, max=20, oneof=选项1 选项2
        // 注意：oneof 使用空格分隔选项，如果选项值包含空格，需要用单引号括起来
        // 例如：oneof=cat dog bird 或 oneof='small size' 'medium size'
        if (typeTrimmed === 'oneof') {
          // oneof 的值：空格分隔的选项列表（支持单引号括起来的选项）
          rules.push({ 
            type: typeTrimmed, 
            value: valueTrimmed  // 保持原样，由 OneOfValidator 解析
          })
        } else {
          // 其他规则：尝试解析为数字
          const numValue = this.parseNumber(valueTrimmed)
          rules.push({ 
            type: typeTrimmed, 
            value: numValue !== null ? numValue : valueTrimmed 
          })
        }
      }
    } else {
      // 无参数规则：required, email
      rules.push({ type: part })
    }
    }
    
    return rules
  }
  
  /**
   * 判断是否是条件验证规则
   */
  private isConditionalRule(type: string): boolean {
    return [
      'required_if',
      'required_unless',
      'required_with',
      'required_without',
      'required_with_all',
      'required_without_all',
      'excluded_if',
      'excluded_unless',
      'excluded_with',
      'excluded_without',
      'eqfield',
      'nefield',
      'gtfield',
      'gtefield',
      'ltfield',
      'ltefield'
    ].includes(type)
  }
  
  /**
   * 解析数字值
   */
  private parseNumber(str: string): number | null {
    const num = Number(str)
    return isNaN(num) ? null : num
  }
}
