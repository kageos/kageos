/**
 * 权限工具函数
 * 用于处理权限相关的逻辑
 */

import type { ServiceTree } from '@/types'

/**
 * 权限信息接口（从 403 响应中获取）
 */
export interface PermissionInfo {
  resource_path: string  // 资源路径
  action: string  // 权限点（如 table:search）
  action_display: string  // 操作显示名称（如 "表格查询"）
  apply_url: string  // 申请权限的 URL
  error_message: string  // 错误消息
}

/**
 * 检查节点是否有指定权限
 * @param node 服务树节点
 * @param action 权限点（如 table:search、function:execute）
 * @returns 是否有权限
 */
export function hasPermission(node: ServiceTree | undefined, action: string): boolean {
  if (!node || !node.permissions) {
    // 如果没有权限信息，默认返回 true（向后兼容）
    return true
  }
  return node.permissions[action] === true
}

/**
 * 检查节点是否有多个权限（只要有一个有权限就返回 true）
 * @param node 服务树节点
 * @param actions 权限点列表
 * @returns 是否有权限
 */
export function hasAnyPermission(node: ServiceTree | undefined, actions: string[]): boolean {
  if (!node || !node.permissions) {
    return true
  }
  return actions.some(action => node.permissions![action] === true)
}

/**
 * 检查节点是否有所有权限（必须全部有权限才返回 true）
 * @param node 服务树节点
 * @param actions 权限点列表
 * @returns 是否有权限
 */
export function hasAllPermissions(node: ServiceTree | undefined, actions: string[]): boolean {
  if (!node || !node.permissions) {
    return true
  }
  return actions.every(action => node.permissions![action] === true)
}

/**
 * 获取权限显示名称
 * @param action 权限点
 * @returns 显示名称
 */
export function getPermissionDisplayName(action: string): string {
  const displayNames: Record<string, string> = {
    // Table 操作
    'table:search': '表格查询',
    'table:create': '表格新增',
    'table:update': '表格更新',
    'table:delete': '表格删除',
    // Form 操作
    'form:submit': '表单提交',
    // Chart 操作
    'chart:query': '图表查询',
    // Callback 操作
    'callback:on_select_fuzzy': '模糊搜索回调',
    // Function 操作
    'function:read': '函数查看',
    'function:execute': '函数执行',
    // Directory 操作
    'directory:read': '目录查看',
    'directory:create': '目录创建',
    'directory:update': '目录更新',
    'directory:delete': '目录删除',
    'directory:manage': '目录管理',
    // App 操作
    'app:read': '应用查看',
    'app:create': '应用创建',
    'app:update': '应用更新',
    'app:delete': '应用删除',
    'app:manage': '应用管理',
    'app:deploy': '应用部署',
  }
  return displayNames[action] || action
}

/**
 * 根据函数类型获取默认权限点
 * @param templateType 模板类型（table、form、chart）
 * @returns 权限点列表
 */
export function getDefaultPermissionsForTemplate(templateType?: string): string[] {
  switch (templateType) {
    case 'table':
      return ['table:search', 'table:create', 'table:update', 'table:delete', 'function:execute']
    case 'form':
      return ['form:submit', 'function:execute']
    case 'chart':
      return ['chart:query', 'function:execute']
    default:
      return ['function:execute']
  }
}

/**
 * 检查 Table 函数的相关权限
 */
export const TablePermissions = {
  search: 'table:search',
  create: 'table:create',
  update: 'table:update',
  delete: 'table:delete',
  execute: 'function:execute',
} as const

/**
 * 检查 Form 函数的相关权限
 */
export const FormPermissions = {
  submit: 'form:submit',
  execute: 'function:execute',
} as const

/**
 * 检查 Chart 函数的相关权限
 */
export const ChartPermissions = {
  query: 'chart:query',
  execute: 'function:execute',
} as const

/**
 * 检查目录的相关权限
 */
export const DirectoryPermissions = {
  read: 'directory:read',
  create: 'directory:create',
  update: 'directory:update',
  delete: 'directory:delete',
  manage: 'directory:manage',
} as const

