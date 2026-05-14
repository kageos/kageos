/**
 * Permission naming and display helpers.
 *
 * Permission enforcement is currently disabled in the frontend, so the
 * hasPermission helpers below intentionally allow all actions. Keep the action
 * constants and labels because other UI surfaces still use them for copy and
 * compatibility with service-tree payloads.
 */

import type { ServiceTree } from '@/types'
import {
  // 权限常量对象
  DirectoryPermission,
  FunctionPermission,
  TablePermission,
  FormPermission,
  ChartPermission,
  DocsPermission,
  BoardPermission,
  // 资源类型和操作类型
  ResourceType,
  ActionType,
  // 工具函数
  buildPermission,
  parsePermission,
  getPermissionsByResourceType,
  // TypeScript 类型
  type PermissionString,
  type ResourceTypeString,
  type ActionTypeString,
} from '@/constants/permissions'

export type PermissionResourceType = 'function' | 'directory' | 'app' | 'docs' | 'board'

/**
 * 获取权限的详细说明
 * @param action 权限点
 * @param resourceType 资源类型（function、directory、app）
 * @param templateType 模板类型（table、form、chart，仅对 function 有效）
 * @returns 权限说明对象，包含 description（说明）和 inheritance（继承规则）
 */
export function getPermissionDescription(
  action: string,
  resourceType?: PermissionResourceType,
  templateType?: string
): { description: string; inheritance?: string } {
  const descriptions: Record<string, { description: string; inheritance?: string }> = {
    // 目录权限
    'directory:read': {
      description: '查看目录信息',
      inheritance: '子资源继承查看权限'
    },
    'directory:write': {
      description: '创建子目录和函数',
      inheritance: '子资源继承相应权限'
    },
    'directory:update': {
      description: '修改目录信息',
      inheritance: '子资源继承更新权限'
    },
    'directory:delete': {
      description: '删除目录及子资源',
      inheritance: '子资源继承删除权限'
    },
    'directory:admin': {
      description: '拥有目录所有权限',
      inheritance: '子资源继承所有权限'
    },
    
    // 工作空间权限
    'app:read': {
      description: '查看工作空间信息',
      inheritance: '子资源继承查看权限'
    },
    'app:create': {
      description: '创建工作空间资源'
    },
    'app:update': {
      description: '修改工作空间信息'
    },
    'app:delete': {
      description: '删除工作空间及资源'
    },
    'app:admin': {
      description: '拥有工作空间所有权限',
      inheritance: '子资源继承所有权限'
    },
    
    // 函数权限
    'function:read': {
      description: '查看函数信息'
    },
    'function:write': {
      description: '新增记录或提交表单'
    },
    'function:update': {
      description: '更新记录'
    },
    'function:delete': {
      description: '删除记录'
    },
    'function:admin': {
      description: '拥有函数所有权限'
    },
  }
  
  return descriptions[action] || { description: '未知权限' }
}

/**
 * 权限信息接口（从 403 响应中获取）
 */
export interface PermissionInfo {
  resource_path: string  // 资源路径
  action: string  // 权限点（如 table:search）
  action_display: string  // 操作显示名称（如 "表格查询"）
  apply_url?: string  // 申请权限的 URL（历史字段，权限申请入口已下线）
  error_message: string  // 错误消息
}

/**
 * Permission enforcement is globally open for the current MVP.
 */
export function hasPermission(_node: ServiceTree | undefined, _action: string): boolean {
  return true
}

/**
 * 判断节点是否有任何权限（不指定具体权限点）
 * @param node 服务树节点
 * @returns 是否有任何权限
 */
export function hasAnyPermissionForNode(_node: ServiceTree | undefined): boolean {
  return true
}

/**
 * 检查节点是否有多个权限（只要有一个有权限就返回 true）
 * @param node 服务树节点
 * @param actions 权限点列表
 * @returns 是否有权限
 */
export function hasAnyPermission(_node: ServiceTree | undefined, _actions: string[]): boolean {
  return true
}

/**
 * 检查节点是否有所有权限（必须全部有权限才返回 true）
 * @param node 服务树节点
 * @param actions 权限点列表
 * @returns 是否有权限
 */
export function hasAllPermissions(_node: ServiceTree | undefined, _actions: string[]): boolean {
  return true
}

/**
 * 获取权限显示名称
 * @param action 权限点
 * @returns 显示名称
 */
export function getPermissionDisplayName(action: string): string {
  const displayNames: Record<string, string> = {
    // Table 操作（新的权限点格式）
    'table:read': '查看表格',
    'table:write': '新增记录',
    'table:update': '更新记录',
    'table:delete': '删除记录',
    'table:admin': '所有权',
    // Form 操作（新的权限点格式）
    'form:read': '查看表单',
    'form:write': '提交表单',
    'form:admin': '所有权',
    // Chart 操作（新的权限点格式）
    'chart:read': '查看图表',
    'chart:admin': '所有权',
    // Docs 操作
    'docs:read': '查看文档',
    'docs:write': '写入文档',
    'docs:update': '更新文档',
    'docs:delete': '删除文档',
    'docs:admin': '所有权',
    // Board 操作
    'board:read': '查看帖子',
    'board:write': '发帖',
    'board:update': '更新帖子',
    'board:delete': '删除帖子',
    'board:admin': '所有权',
    // ⭐ 兼容旧格式（function:read、function:write 等）
    'function:read': '查看函数',
    'function:write': '写入函数',
    'function:update': '更新函数',
    'function:delete': '删除函数',
    'function:admin': '所有权',
    // Directory 操作
    'directory:read': '查看目录',
    'directory:write': '写入目录',
    'directory:update': '更新目录',
    'directory:delete': '删除目录',
    'directory:admin': '所有权',
    // App 操作（工作空间）
    'app:read': '查看工作空间',
    'app:create': '创建工作空间',
    'app:update': '更新工作空间',
    'app:delete': '删除工作空间',
    'app:admin': '所有权',
  }
  return displayNames[action] || action
}

/**
 * 获取权限点的简短显示名称（用于按钮提示）
 * @param action 权限点
 * @returns 简短显示名称（如 "read权限"、"write权限"）
 */
export function getPermissionShortName(action: string): string {
  const shortNames: Record<string, string> = {
    // Table 操作（新的权限点格式）
    'table:read': 'read权限',
    'table:write': 'write权限',
    'table:update': 'update权限',
    'table:delete': 'delete权限',
    'table:admin': 'admin权限',
    // Form 操作（新的权限点格式）
    'form:read': 'read权限',
    'form:write': 'write权限',
    'form:admin': 'admin权限',
    // Chart 操作（新的权限点格式）
    'chart:read': 'read权限',
    'chart:admin': 'admin权限',
    'docs:read': 'read权限',
    'docs:write': 'write权限',
    'docs:update': 'update权限',
    'docs:delete': 'delete权限',
    'docs:admin': 'admin权限',
    'board:read': 'read权限',
    'board:write': 'write权限',
    'board:update': 'update权限',
    'board:delete': 'delete权限',
    'board:admin': 'admin权限',
    // ⭐ 兼容旧格式（function:read、function:write 等）
    'function:read': 'read权限',
    'function:write': 'write权限',
    'function:update': 'update权限',
    'function:delete': 'delete权限',
    'function:admin': 'admin权限',
    'directory:read': 'read权限',
    'directory:write': 'write权限',
    'directory:update': 'update权限',
    'directory:delete': 'delete权限',
    'directory:admin': 'admin权限',
    'app:read': 'read权限',
    'app:create': 'create权限',
    'app:update': 'update权限',
    'app:delete': 'delete权限',
    'app:admin': 'admin权限',
  }
  return shortNames[action] || `${action.split(':')[1] || action}权限`
}

/**
 * 根据函数类型获取默认权限点
 * @param templateType 模板类型（table、form、chart）
 * @returns 权限点列表
 */
export function getDefaultPermissionsForTemplate(templateType?: string): string[] {
  switch (templateType) {
    case 'table':
      return ['table:read', 'table:write', 'table:update', 'table:delete', 'table:admin']
    case 'form':
      return ['form:write', 'form:admin']
    case 'chart':
      return ['chart:read', 'chart:admin']
    default:
      // 默认使用 table 类型的权限点
      return ['table:read', 'table:write', 'table:update', 'table:delete', 'table:admin']
  }
}

export interface PermissionOption {
  action: string
  displayName: string
  isMinimal?: boolean
  isManage?: boolean
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
  resourceType?: PermissionResourceType,
  templateType?: string
): PermissionOption[] {
  const permissions: PermissionOption[] = []

  // 根据资源类型返回相关权限点
  // ⭐ 权限顺序：小权限（具体操作）在前，大权限（所有权/管理）在后
  if (resourceType === 'function') {
    // ⭐ 根据模板类型使用不同的权限点格式：table:read、form:write、chart:read 等
    if (templateType === 'table') {
      permissions.push(
        { action: 'table:read', displayName: '查看表格', isMinimal: true },
        { action: 'table:write', displayName: '新增记录', isMinimal: false },
        { action: 'table:update', displayName: '更新记录', isMinimal: false },
        { action: 'table:delete', displayName: '删除记录', isMinimal: false }
      )
      // 大权限（所有权）放在最后
      permissions.push(
        { action: 'table:admin', displayName: '所有权', isMinimal: false, isManage: true }
      )
    } else if (templateType === 'form') {
      permissions.push(
        { action: 'form:write', displayName: '提交表单', isMinimal: true }
      )
      // 大权限（所有权）放在最后
      permissions.push(
        { action: 'form:admin', displayName: '所有权', isMinimal: false, isManage: true }
      )
      // form 类型虽然定义了 read/update/delete，但业务逻辑中不使用，所以不显示
    } else if (templateType === 'chart') {
      permissions.push(
        { action: 'chart:read', displayName: '查看图表', isMinimal: true }
      )
      // 大权限（所有权）放在最后
      permissions.push(
        { action: 'chart:admin', displayName: '所有权', isMinimal: false, isManage: true }
      )
      // chart 类型虽然定义了 write/update/delete，但业务逻辑中不使用，所以不显示
    } else {
      // 默认使用 table 类型的权限点
      permissions.push(
        { action: 'table:read', displayName: '查看函数', isMinimal: true },
        { action: 'table:write', displayName: '写入函数', isMinimal: false },
        { action: 'table:update', displayName: '更新函数', isMinimal: false },
        { action: 'table:delete', displayName: '删除函数', isMinimal: false }
      )
    // 大权限（所有权）放在最后
    permissions.push(
        { action: 'table:admin', displayName: '所有权', isMinimal: false, isManage: true }
    )
    }
  } else if (resourceType === 'directory') {
    // 目录相关权限：小权限在前
    permissions.push(
      { action: 'directory:read', displayName: '查看目录', isMinimal: true },
      { action: 'directory:write', displayName: '写入目录', isMinimal: false },
      { action: 'directory:update', displayName: '更新目录', isMinimal: false },
      { action: 'directory:delete', displayName: '删除目录', isMinimal: false }
    )
    // 大权限（所有权）放在最后
    permissions.push(
      { action: 'directory:admin', displayName: '所有权', isMinimal: false, isManage: true }
    )
  } else if (resourceType === 'app') {
    // 工作空间相关权限：小权限在前
    permissions.push(
      { action: 'app:read', displayName: '查看工作空间', isMinimal: true },
      { action: 'app:create', displayName: '创建工作空间', isMinimal: false },
      { action: 'app:update', displayName: '更新工作空间', isMinimal: false },
      { action: 'app:delete', displayName: '删除工作空间', isMinimal: false }
    )
    // 大权限（所有权）放在最后
    permissions.push(
      { action: 'app:admin', displayName: '所有权', isMinimal: false, isManage: true }
    )
  } else if (resourceType === 'docs') {
    permissions.push(
      { action: 'docs:read', displayName: '查看文档', isMinimal: true },
      { action: 'docs:write', displayName: '写入文档', isMinimal: false },
      { action: 'docs:update', displayName: '更新文档', isMinimal: false },
      { action: 'docs:delete', displayName: '删除文档', isMinimal: false }
    )
    permissions.push(
      { action: 'docs:admin', displayName: '所有权', isMinimal: false, isManage: true }
    )
  } else if (resourceType === 'board') {
    permissions.push(
      { action: 'board:read', displayName: '查看帖子', isMinimal: true },
      { action: 'board:write', displayName: '发帖', isMinimal: false },
      { action: 'board:update', displayName: '更新帖子', isMinimal: false },
      { action: 'board:delete', displayName: '删除帖子', isMinimal: false }
    )
    permissions.push(
      { action: 'board:admin', displayName: '所有权', isMinimal: false, isManage: true }
    )
  } else {
    // 未知类型，返回通用权限
    permissions.push(
      { action: 'function:read', displayName: '查看函数', isMinimal: true },
      { action: 'function:admin', displayName: '所有权', isMinimal: false, isManage: true }
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
  availablePermissions: PermissionOption[]
): string[] {
  return availablePermissions
    .filter(p => p.isMinimal === true)
    .map(p => p.action)
}

/**
 * ⭐ 权限常量统一管理
 * 从 @/constants/permissions 导入，避免重复定义和硬编码
 * 
 * 使用示例：
 * - DirectoryPermission.read  // 'directory:read'
 * - TablePermission.write     // 'table:write'
 * - DocsPermission.admin      // 'docs:admin'
 */
export {
  // 权限常量对象
  DirectoryPermission,
  FunctionPermission,
  TablePermission,
  FormPermission,
  ChartPermission,
  DocsPermission,
  BoardPermission,
  // 资源类型和操作类型
  ResourceType,
  ActionType,
  // 工具函数
  buildPermission,
  parsePermission,
  getPermissionsByResourceType,
  // TypeScript 类型
  type PermissionString,
  type ResourceTypeString,
  type ActionTypeString,
}

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
  
  const user = pathParts[0]!
  const app = pathParts[1]!
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
    const functionName = pathParts[pathParts.length - 1] ?? null
    
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
  resourceType: PermissionResourceType
  resourceName: string
  displayName: string
  permissions: PermissionOption[]
  quickSelect?: {
    label: string
    actions: string[]
  }
}

export function getPermissionScopes(
  resourcePath: string,
  resourceType?: PermissionResourceType,
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
      : `工作空间：${parsed.app}`,
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
        actions: ['directory:admin']
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
      displayName: `工作空间：${parsed.app}`,
      permissions: appPermissions,
      quickSelect: {
        label: '申请此工作空间的管理权限',
        actions: ['app:admin']
      },
    })
  }
  
  return scopes
}
