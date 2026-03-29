/**
 * useWidgetValidation - Widget 验证组合式函数
 * 🔥 符合依赖倒置原则：让 Widget 自己负责验证逻辑
 * 
 * 功能：
 * - 提供统一的验证接口
 * - 支持嵌套字段的递归验证
 * - 返回验证错误列表
 */

import type { FieldConfig, FieldValue } from '@/core/types/field'
import type { ValidationEngine, ValidationResult } from '@/core/validation'
import { useFormDataStore, type FormDataStore } from '@/core/stores-v2/formData'
import { Logger } from '@/core/utils/logger'

export interface WidgetValidationContext {
  validationEngine: ValidationEngine | null
  allFields: FieldConfig[]
  fieldErrors: Map<string, ValidationResult[]>
  formDataStore?: Pick<FormDataStore, 'getValue'>
}

/**
 * Widget 验证结果
 */
export interface WidgetValidationResult {
  /** 当前字段的错误列表 */
  errors: ValidationResult[]
  /** 嵌套字段的错误列表（路径 -> 错误列表） */
  nestedErrors: Map<string, ValidationResult[]>
  /** 是否有错误 */
  hasError: boolean
}

/**
 * 验证单个字段（基础验证）
 */
export function validateFieldValue(
  field: FieldConfig,
  fieldPath: string,
  context: WidgetValidationContext
): ValidationResult[] {
  if (!field.validation) {
    return []
  }
  
  if (!context.validationEngine) {
    return []
  }
  
  const formDataStore = context.formDataStore || useFormDataStore()
  const value = formDataStore.getValue(fieldPath)
  
  try {
    return context.validationEngine.validateField(field, value, context.allFields, fieldPath)
  } catch (error) {
    Logger.error('[useWidgetValidation]', `验证字段 ${fieldPath} 失败`, error)
    return []
  }
}

/**
 * 验证 Widget 及其嵌套字段（递归）
 * 
 * @param field 字段配置
 * @param fieldPath 字段路径
 * @param context 验证上下文
 * @returns 验证结果
 */
export function validateWidget(
  field: FieldConfig,
  fieldPath: string,
  context: WidgetValidationContext
): WidgetValidationResult {
  const errors: ValidationResult[] = []
  const nestedErrors = new Map<string, ValidationResult[]>()
  
  // 1. 验证当前字段
  const fieldErrors = validateFieldValue(field, fieldPath, context)
  if (fieldErrors.length > 0) {
    errors.push(...fieldErrors)
  }
  
  // 2. 递归验证嵌套字段（由 Widget 自己决定如何验证）
  if (field.children && field.children.length > 0) {
    const nestedResult = validateNestedFields(field, fieldPath, context)
    nestedResult.forEach((nestedErrorsForPath, path) => {
      nestedErrors.set(path, nestedErrorsForPath)
    })
  }
  
  return {
    errors,
    nestedErrors,
    hasError: errors.length > 0 || nestedErrors.size > 0
  }
}

/**
 * 验证嵌套字段（通用逻辑，由具体 Widget 调用）
 * 
 * 注意：此函数不实现具体逻辑，由 validateFormWidgetNestedFields 和
 * validateTableWidgetNestedFields 实现具体的验证逻辑
 */
function validateNestedFields(
  field: FieldConfig,
  parentPath: string,
  context: WidgetValidationContext
): Map<string, ValidationResult[]> {
  // 此函数已废弃，保留仅为兼容性
  // 实际验证逻辑由 validateFormWidgetNestedFields 和 validateTableWidgetNestedFields 实现
  return new Map<string, ValidationResult[]>()
}

/**
 * 验证 FormWidget 的嵌套字段
 */
export function validateFormWidgetNestedFields(
  field: FieldConfig,
  parentPath: string,
  context: WidgetValidationContext
): Map<string, ValidationResult[]> {
  const nestedErrors = new Map<string, ValidationResult[]>()
  
  if (!field.children || field.children.length === 0) {
    return nestedErrors
  }
  
  // FormWidget: 路径格式为 parentField.subField
  field.children.forEach((subField: FieldConfig) => {
    const subFieldPath = `${parentPath}.${subField.code}`
    
    // 1. 验证子字段本身（如果有验证规则）
    const subErrors = validateFieldValue(subField, subFieldPath, context)
    if (subErrors.length > 0) {
      nestedErrors.set(subFieldPath, subErrors)
    }
    
    // 2. 递归验证更深层的嵌套字段
    if (subField.children && subField.children.length > 0) {
      // 判断子字段的类型
      if (subField.widget?.type === 'form') {
        // 嵌套的 FormWidget：递归验证其嵌套字段
        const deeperErrors = validateFormWidgetNestedFields(subField, subFieldPath, context)
        deeperErrors.forEach((errors, path) => {
          nestedErrors.set(path, errors)
        })
      } else if (subField.widget?.type === 'table') {
        // 嵌套的 TableWidget：递归验证其嵌套字段
        const deeperErrors = validateTableWidgetNestedFields(subField, subFieldPath, context)
        deeperErrors.forEach((errors, path) => {
          nestedErrors.set(path, errors)
        })
      } else {
        // 其他类型：递归验证（可能是其他容器组件）
        const deeperErrors = validateFormWidgetNestedFields(subField, subFieldPath, context)
        deeperErrors.forEach((errors, path) => {
          nestedErrors.set(path, errors)
        })
      }
    }
  })
  
  return nestedErrors
}

/**
 * 验证 TableWidget 的嵌套字段
 */
export function validateTableWidgetNestedFields(
  field: FieldConfig,
  parentPath: string,
  context: WidgetValidationContext
): Map<string, ValidationResult[]> {
  const nestedErrors = new Map<string, ValidationResult[]>()
  
  if (!field.children || field.children.length === 0) {
    return nestedErrors
  }
  
  const formDataStore = context.formDataStore || useFormDataStore()
  const value = formDataStore.getValue(parentPath)
  const tableValue = value.raw
  
  if (!Array.isArray(tableValue)) {
    return nestedErrors
  }
  
  // TableWidget: 路径格式为 parentField[index].subField
  tableValue.forEach((row: any, index: number) => {
    field.children!.forEach((subField: FieldConfig) => {
      const subFieldPath = `${parentPath}[${index}].${subField.code}`
      
      // 验证子字段
      const subErrors = validateFieldValue(subField, subFieldPath, context)
      if (subErrors.length > 0) {
        nestedErrors.set(subFieldPath, subErrors)
      }
      
      // 递归验证更深层的嵌套字段
      if (subField.children && subField.children.length > 0) {
        // 判断子字段的类型
        if (subField.widget?.type === 'form') {
          const deeperErrors = validateFormWidgetNestedFields(subField, subFieldPath, context)
          deeperErrors.forEach((errors, path) => {
            nestedErrors.set(path, errors)
          })
        } else if (subField.widget?.type === 'table') {
          const deeperErrors = validateTableWidgetNestedFields(subField, subFieldPath, context)
          deeperErrors.forEach((errors, path) => {
            nestedErrors.set(path, errors)
          })
        }
      }
    })
  })
  
  return nestedErrors
}
