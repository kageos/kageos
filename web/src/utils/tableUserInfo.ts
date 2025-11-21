/**
 * 表格用户信息收集工具函数
 */
import type { FieldConfig } from '@/types'

/**
 * 从表格数据中收集用户名
 */
export function collectUsernamesFromTableData(
  tableData: any[],
  userFields: FieldConfig[]
): Set<string> {
  const usernames = new Set<string>()
  
  tableData.forEach((row: any) => {
    userFields.forEach((field: FieldConfig) => {
      const value = row[field.code]
      if (value !== null && value !== undefined && value !== '') {
        usernames.add(String(value))
      }
    })
  })
  
  return usernames
}

/**
 * 从搜索表单中收集用户名
 */
export function collectUsernamesFromSearchForm(
  searchForm: Record<string, any>,
  searchableFields: FieldConfig[]
): Set<string> {
  const usernames = new Set<string>()
  
  searchableFields.forEach((field: FieldConfig) => {
    if (field.widget?.type === 'user' && searchForm[field.code]) {
      const value = searchForm[field.code]
      if (Array.isArray(value)) {
        value.forEach(v => {
          if (v) usernames.add(String(v))
        })
      } else if (value) {
        usernames.add(String(value))
      }
    }
  })
  
  return usernames
}

/**
 * 收集所有用户名（表格数据 + 搜索表单）
 */
export function collectAllUsernames(
  tableData: any[],
  searchForm: Record<string, any>,
  visibleFields: FieldConfig[],
  searchableFields: FieldConfig[]
): string[] {
  // 1. 识别所有 user 类型的字段
  const userFields = visibleFields.filter((field: FieldConfig) => field.widget?.type === 'user')
  
  if (userFields.length === 0) {
    return []
  }
  
  // 2. 收集表格数据中的用户名
  const tableUsernames = collectUsernamesFromTableData(tableData, userFields)
  
  // 3. 收集搜索表单中的用户名
  const searchUsernames = collectUsernamesFromSearchForm(searchForm, searchableFields)
  
  // 4. 合并并去重
  return [...new Set([...tableUsernames, ...searchUsernames])]
}

