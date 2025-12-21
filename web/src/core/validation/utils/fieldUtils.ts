/**
 * 验证器工具函数
 * 用于提取重复逻辑，提高代码复用性
 */

import type { FieldConfig, FieldValue } from '../../types/field'
import type { ValidationContext } from '../types'

/**
 * 判断字段值是否为空
 * 
 * 🔥 对于 table 类型字段，需要检查过滤后的有效行数
 * 因为 TableFieldExtractor 会过滤掉空行（所有字段都为 null/undefined 的行）
 */
export function isEmpty(value: FieldValue, field?: FieldConfig): boolean {
  // 基本空值检查
  if (value.raw === null || value.raw === undefined || value.raw === '') {
    return true
  }
  
  // 数组类型检查
  if (Array.isArray(value.raw)) {
    // 🔥 如果是 table 类型字段，需要检查过滤后的有效行数
    if (field?.widget?.type === 'table') {
      // 🔥 使用与 TableFieldExtractor 相同的逻辑过滤空行
      // 过滤掉空行（所有字段都为 null/undefined 的行）
      const validRows = value.raw.filter((row: any) => {
        if (!row || typeof row !== 'object') {
          return false
        }
        // 🔥 检查行中是否有任何非空字段
        // 注意：这里只检查 null 和 undefined，不检查空字符串
        // 因为空字符串可能是用户有意输入的（例如备注字段可以为空）
        // 只有当所有字段都是 null 或 undefined 时，才认为是空行
        const hasValidValue = Object.values(row).some((val: any) => {
          // 非空值：不是 null、undefined
          return val !== null && val !== undefined
        })
        return hasValidValue
      })
      
      // 🔥 调试日志：帮助排查问题
      if (value.raw.length > 0 && validRows.length === 0) {
        console.warn('[isEmpty] table 字段所有行都被过滤为空', {
          fieldCode: field.code,
          totalRows: value.raw.length,
          rows: value.raw.map((row: any, index: number) => ({
            index,
            row,
            values: Object.entries(row).map(([key, val]) => ({ key, val, isEmpty: val === null || val === undefined }))
          }))
        })
      }
      
      return validRows.length === 0
    }
    
    // 普通数组：检查长度
    return value.raw.length === 0
  }
  
  return false
}

/**
 * 从验证上下文中查找字段配置
 * 
 * @param context 验证上下文
 * @returns 字段配置，如果找不到则返回 null
 */
export function findFieldInContext(context: ValidationContext): FieldConfig | null {
  // 🔥 先尝试匹配 code（因为 fieldPath 通常是 code）
  let foundField = context.allFields.find(f => f.code === context.fieldPath)
  
  // 如果还找不到，尝试匹配 field_path
  if (!foundField) {
    foundField = context.allFields.find(f => {
      if (f.field_path) {
        return f.field_path === context.fieldPath
      }
      return false
    })
  }
  
  // 🔥 调试日志：如果找不到字段配置，记录警告
  if (!foundField) {
    console.warn('[findFieldInContext] 找不到字段配置', {
      fieldPath: context.fieldPath,
      allFieldCodes: context.allFields.map(f => f.code),
      allFieldPaths: context.allFields.map(f => f.field_path || f.code)
    })
  }
  
  return foundField || null
}

/**
 * 生成必填错误消息
 * 
 * @param fieldName 字段名称（用户友好的名称）
 * @returns 错误消息
 */
export function createRequiredErrorMessage(fieldName: string): string {
  return `${fieldName}必填`
}

/**
 * 从验证上下文中获取字段名称
 * 
 * @param context 验证上下文
 * @param fallback 找不到字段时的默认名称
 * @returns 字段名称
 */
export function getFieldName(context: ValidationContext, fallback: string = '此字段'): string {
  const field = findFieldInContext(context)
  return field?.name || fallback
}

/**
 * 判断字段是否为字符串类型
 * 
 * 判断规则：
 * - 仅基于 data.type 判断（数据类型层面）
 * - data.type 包含 'string'、'text' 等认为是字符串类型
 * - 如果 data.type 为空，默认认为是字符串类型
 * 
 * 注意：不判断 widget.type，因为 widget.type 是渲染层面的概念，不是数据类型
 * 
 * @param field 字段配置（可为 null 或 undefined）
 * @returns 是否为字符串类型
 */
export function isStringField(field: FieldConfig | null | undefined): boolean {
  if (!field) return false
  
  const dataType = field.data?.type?.toLowerCase() || ''
  
  // 如果 data.type 为空，默认认为是字符串类型
  if (!dataType || dataType === '') {
    return true
  }
  
  // 检查 data.type 是否包含字符串类型标识
  if (dataType.includes('string') || dataType.includes('text')) {
    return true
  }
  
  return false
}

