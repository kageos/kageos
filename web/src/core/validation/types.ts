/**
 * 验证系统类型定义
 */

import type { FieldConfig, FieldValue } from '../types/field'
import type { ReactiveFormDataManager } from '../managers/ReactiveFormDataManager'

/**
 * 验证规则（解析后的结构）
 */
export interface ValidationRule {
  /** 验证器类型，如 'required', 'min', 'required_if' */
  type: string
  /** 规则的值（如 min=2 中的 2） */
  value?: string | number
  /** 条件验证中引用的字段 code（如 required_if=MemberType vip 中的 member_type） */
  field?: string
}

/**
 * 验证结果
 */
export interface ValidationResult {
  /** 是否通过验证 */
  valid: boolean
  /** 错误信息（验证失败时） */
  message?: string
  /** 字段配置（用于错误消息格式化） */
  field?: FieldConfig
}

/**
 * 验证上下文（用于依赖注入）
 */
export interface ValidationContext {
  /** 表单数据管理器（用于访问其他字段值） */
  formManager: ReactiveFormDataManager
  /** 当前字段的路径 */
  fieldPath: string
  /** 所有字段配置（用于查找字段信息） */
  allFields: FieldConfig[]
}

/**
 * 验证器接口
 * 
 * 所有验证器必须实现此接口
 * 符合开闭原则：新增验证器只需实现此接口并注册
 */
export interface Validator {
  /** 验证器名称（对应 validation 字符串中的规则名） */
  readonly name: string
  
  /**
   * 执行验证
   * 
   * @param value 当前字段的值
   * @param rule 验证规则（已解析）
   * @param context 验证上下文（包含 formManager 等依赖）
   * @returns 验证结果
   */
  validate(
    value: FieldValue,
    rule: ValidationRule,
    context: ValidationContext
  ): ValidationResult
}

