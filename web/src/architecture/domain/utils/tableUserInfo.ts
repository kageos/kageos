/**
 * 表格用户信息收集工具函数
 */
import type { FieldConfig } from '@/architecture/domain/types/field'
import { getSearchFieldRawValue } from '@/architecture/domain/utils/searchFieldValue'

type TableDataRow = Record<string, unknown>

interface FilesValuePayload {
  files?: Array<{ upload_user?: unknown }>
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function asFilesPayload(value: unknown): FilesValuePayload | null {
  if (!isRecord(value)) {
    return null
  }
  const raw = value.raw
  if (isRecord(raw)) {
    return raw as FilesValuePayload
  }
  return value as FilesValuePayload
}

/**
 * 从表格数据中收集用户名
 * 注意：不收集 files widget 的 upload_user，因为表格列表不需要显示用户信息
 * files widget 的用户信息只在详情模式（detail）下才需要收集和查询
 */
export function collectUsernamesFromTableData(
  tableData: TableDataRow[],
  userFields: FieldConfig[]
): Set<string> {
  const usernames = new Set<string>()
  
  tableData.forEach(row => {
    // 只收集 user 类型字段的用户名
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
  searchForm: Record<string, unknown>,
  searchableFields: FieldConfig[]
): Set<string> {
  const usernames = new Set<string>()
  
  searchableFields.forEach((field: FieldConfig) => {
    if (field.widget?.type === 'user' && searchForm[field.code]) {
      const value = getSearchFieldRawValue(searchForm[field.code])
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
 * 注意：不收集 files widget 的 upload_user，因为表格列表不需要显示用户信息
 * files widget 的用户信息只在详情模式（detail）下才需要收集和查询
 */
export function collectAllUsernames(
  tableData: TableDataRow[],
  searchForm: Record<string, unknown>,
  visibleFields: FieldConfig[],
  searchableFields: FieldConfig[]
): string[] {
  // 1. 识别所有 user 类型的字段
  const userFields = visibleFields.filter((field: FieldConfig) => field.widget?.type === 'user')
  
  if (userFields.length === 0) {
    return []
  }
  
  // 2. 收集表格数据中的用户名（只收集 user 字段）
  const tableUsernames = collectUsernamesFromTableData(tableData, userFields)
  
  // 3. 收集搜索表单中的用户名
  const searchUsernames = collectUsernamesFromSearchForm(searchForm, searchableFields)
  
  // 4. 合并并去重
  return [...new Set([...tableUsernames, ...searchUsernames])]
}

/**
 * 🔥 从单行数据中收集 files widget 的 upload_user（用于详情模式）
 * @param rowData 单行数据
 * @param visibleFields 可见字段列表
 * @returns 用户名数组
 */
export function collectFilesUploadUsersFromRow(
  rowData: TableDataRow,
  visibleFields: FieldConfig[]
): string[] {
  const usernames = new Set<string>()
  
  // 识别所有 files 类型的字段
  const filesFields = visibleFields.filter((field: FieldConfig) => field.widget?.type === 'files')
  
  filesFields.forEach((field: FieldConfig) => {
    const value = rowData[field.code]
    const filesData = asFilesPayload(value)
    if (Array.isArray(filesData?.files)) {
      filesData.files.forEach(file => {
        if (file.upload_user) {
          usernames.add(String(file.upload_user))
        }
      })
    }
  })
  
  return Array.from(usernames)
}
