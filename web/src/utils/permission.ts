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
    'table:create': '表格新增',
    'table:update': '表格更新',
    'table:delete': '表格删除',
    // Form 操作
    'form:submit': '表单提交',
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
      return ['function:read', 'table:create', 'table:update', 'table:delete', 'function:execute']
    case 'form':
      return ['form:submit', 'function:execute']
    case 'chart':
      return ['function:read', 'function:execute']
    default:
      return ['function:execute']
  }
}

/**
 * 根据资源路径和类型获取可申请的权限点列表
 * @param resourcePath 资源路径（full-code-path）
 * @param resourceType 资源类型（function、directory、app）
 * @param templateType 模板类型（table、form、chart，仅对 function 有效）
 * @returns 权限点选项列表（包含 action 和 displayName）
 */
export function getAvailablePermissions(
  resourcePath: string,
  resourceType?: 'function' | 'directory' | 'app',
  templateType?: string
): Array<{ action: string; displayName: string; isMinimal?: boolean }> {
  const permissions: Array<{ action: string; displayName: string; isMinimal?: boolean }> = []

  // 根据资源类型返回相关权限点
  if (resourceType === 'function') {
    // 函数相关权限
    permissions.push(
      { action: 'function:read', displayName: '函数查看', isMinimal: true },
      { action: 'function:execute', displayName: '函数执行', isMinimal: false }
    )

    // 根据模板类型添加特定权限
    if (templateType === 'table') {
      permissions.push(
        { action: 'table:create', displayName: '表格新增', isMinimal: false },
        { action: 'table:update', displayName: '表格更新', isMinimal: false },
        { action: 'table:delete', displayName: '表格删除', isMinimal: false }
      )
    } else if (templateType === 'form') {
      permissions.push(
        { action: 'form:submit', displayName: '表单提交', isMinimal: true }
      )
    }
  } else if (resourceType === 'directory') {
    // 目录相关权限
    permissions.push(
      { action: 'directory:read', displayName: '目录查看', isMinimal: true },
      { action: 'directory:create', displayName: '目录创建', isMinimal: false },
      { action: 'directory:update', displayName: '目录更新', isMinimal: false },
      { action: 'directory:delete', displayName: '目录删除', isMinimal: false },
      { action: 'directory:manage', displayName: '目录管理', isMinimal: false }
    )
  } else if (resourceType === 'app') {
    // 应用相关权限
    permissions.push(
      { action: 'app:read', displayName: '应用查看', isMinimal: true },
      { action: 'app:create', displayName: '应用创建', isMinimal: false },
      { action: 'app:update', displayName: '应用更新', isMinimal: false },
      { action: 'app:delete', displayName: '应用删除', isMinimal: false },
      { action: 'app:deploy', displayName: '应用部署', isMinimal: false },
      { action: 'app:manage', displayName: '应用管理', isMinimal: false }
    )
  } else {
    // 未知类型，返回通用权限
    permissions.push(
      { action: 'function:read', displayName: '函数查看', isMinimal: true },
      { action: 'function:execute', displayName: '函数执行', isMinimal: false }
    )
  }

  return permissions
}

/**
 * 获取默认选中的权限点（最小粒度）
 * @param availablePermissions 可用的权限点列表
 * @returns 默认选中的权限点列表
 */
export function getDefaultSelectedPermissions(
  availablePermissions: Array<{ action: string; displayName: string; isMinimal?: boolean }>
): string[] {
  return availablePermissions
    .filter(p => p.isMinimal === true)
    .map(p => p.action)
}

/**
 * 检查 Table 函数的相关权限
 */
export const TablePermissions = {
  read: 'function:read',
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
  read: 'function:read',
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

/**
 * 解析资源路径，提取父级路径
 * @param resourcePath 资源路径（full-code-path）
 * @returns 父级路径信息
 */
export function parseResourcePath(resourcePath: string): {
  user: string
  app: string
  appPath: string  // /user/app
  directoryPath: string | null  // 父目录路径（如果存在）
  functionName: string | null  // 函数名（如果存在）
  isFunction: boolean
  isDirectory: boolean
  isApp: boolean
} {
  const pathParts = resourcePath.split('/').filter(Boolean)
  
  if (pathParts.length < 2) {
    throw new Error('资源路径格式错误，至少需要 user/app')
  }
  
  const user = pathParts[0]
  const app = pathParts[1]
  const appPath = `/${user}/${app}`
  
  if (pathParts.length === 2) {
    // 应用级别
    return {
      user,
      app,
      appPath,
      directoryPath: null,
      functionName: null,
      isFunction: false,
      isDirectory: false,
      isApp: true,
    }
  } else if (pathParts.length === 3) {
    // 可能是目录或函数（需要根据实际节点类型判断，这里默认按目录处理）
    return {
      user,
      app,
      appPath,
      directoryPath: resourcePath,
      functionName: null,
      isFunction: false,
      isDirectory: true,
      isApp: false,
    }
  } else {
    // 可能是函数（最后一段是函数名）
    const directoryPath = '/' + pathParts.slice(0, -1).join('/')
    const functionName = pathParts[pathParts.length - 1]
    
    return {
      user,
      app,
      appPath,
      directoryPath,
      functionName,
      isFunction: true,
      isDirectory: false,
      isApp: false,
    }
  }
}

/**
 * 获取权限范围选项（包括当前资源和父级资源）
 * @param resourcePath 资源路径（full-code-path）
 * @param resourceType 资源类型（function、directory、app）
 * @param templateType 模板类型（table、form、chart，仅对 function 有效）
 * @returns 权限范围选项列表
 */
export interface PermissionScope {
  resourcePath: string
  resourceType: 'function' | 'directory' | 'app'
  resourceName: string
  displayName: string
  permissions: Array<{ action: string; displayName: string; isMinimal?: boolean }>
  quickSelect?: {
    label: string
    actions: string[]
  }
}

export function getPermissionScopes(
  resourcePath: string,
  resourceType?: 'function' | 'directory' | 'app',
  templateType?: string
): PermissionScope[] {
  const scopes: PermissionScope[] = []
  const parsed = parseResourcePath(resourcePath)
  
  // 1. 当前资源的权限
  const currentPermissions = getAvailablePermissions(resourcePath, resourceType, templateType)
  scopes.push({
    resourcePath,
    resourceType: resourceType || (parsed.isFunction ? 'function' : parsed.isDirectory ? 'directory' : 'app'),
    resourceName: parsed.functionName || parsed.directoryPath?.split('/').pop() || parsed.app || '当前资源',
    displayName: parsed.isFunction 
      ? `函数：${parsed.functionName}` 
      : parsed.isDirectory 
      ? `目录：${parsed.directoryPath}` 
      : `应用：${parsed.app}`,
    permissions: currentPermissions,
    quickSelect: parsed.isFunction ? {
      label: '申请此函数的全部权限',
      actions: currentPermissions.map(p => p.action)
    } : undefined,
  })
  
  // 2. 父级目录的权限（如果存在）
  if (parsed.directoryPath && parsed.directoryPath !== parsed.appPath) {
    const directoryPermissions = getAvailablePermissions(parsed.directoryPath, 'directory')
    scopes.push({
      resourcePath: parsed.directoryPath,
      resourceType: 'directory',
      resourceName: parsed.directoryPath.split('/').pop() || '目录',
      displayName: `父级目录：${parsed.directoryPath}`,
      permissions: directoryPermissions,
      quickSelect: {
        label: '申请此目录的管理权限',
        actions: ['directory:manage']
      },
    })
  }
  
  // 3. 应用的权限（如果当前不是应用）
  if (!parsed.isApp) {
    const appPermissions = getAvailablePermissions(parsed.appPath, 'app')
    scopes.push({
      resourcePath: parsed.appPath,
      resourceType: 'app',
      resourceName: parsed.app,
      displayName: `应用：${parsed.app}`,
      permissions: appPermissions,
      quickSelect: {
        label: '申请此应用的管理权限',
        actions: ['app:manage']
      },
    })
  }
  
  return scopes
}

/**
 * 构建权限申请 URL
 * @param resourcePath 资源路径（full-code-path）
 * @param action 权限点（如 table:update）
 * @returns 权限申请页面的 URL
 */
export function buildPermissionApplyURL(resourcePath: string, action: string): string {
  return `/permissions/apply?resource=${encodeURIComponent(resourcePath)}&action=${encodeURIComponent(action)}`
}

