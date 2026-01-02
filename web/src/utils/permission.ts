/**
 * 权限工具函数
 * 用于处理权限相关的逻辑
 */

import type { ServiceTree } from '@/types'
import { useNodePermissionsStore } from '@/stores/nodePermissions'

/**
 * 获取权限的详细说明
 * @param action 权限点
 * @param resourceType 资源类型（function、directory、app）
 * @param templateType 模板类型（table、form、chart，仅对 function 有效）
 * @returns 权限说明对象，包含 description（说明）和 inheritance（继承规则）
 */
export function getPermissionDescription(
  action: string,
  resourceType?: 'function' | 'directory' | 'app',
  templateType?: string
): { description: string; inheritance?: string } {
  const descriptions: Record<string, { description: string; inheritance?: string }> = {
    // 目录权限
    'directory:read': {
      description: '可以查看目录的基本信息和子资源列表',
      inheritance: '子目录和子函数会继承查看权限'
    },
    'directory:write': {
      description: '可以在目录下创建新的子目录和函数',
      inheritance: '子目录会继承写入权限；子函数中，表格函数继承"新增记录"权限，表单函数继承"表单提交"权限'
    },
    'directory:update': {
      description: '可以修改目录的基本信息（名称、描述等）',
      inheritance: '子目录会继承更新权限；子函数中，表格函数继承"更新记录"权限'
    },
    'directory:delete': {
      description: '可以删除目录及其所有子资源',
      inheritance: '子目录会继承删除权限；子函数中，表格函数继承"删除记录"权限'
    },
    'directory:manage': {
      description: '拥有目录的所有权限（查看、创建、更新、删除），以及所有子资源的完整权限',
      inheritance: '子目录会继承管理权限（所有权）；子函数会继承所有相关权限（所有权）'
    },
    
    // 工作空间权限
    'app:read': {
      description: '可以查看工作空间的基本信息和资源列表',
      inheritance: '子目录和子函数会继承查看权限'
    },
    'app:create': {
      description: '可以在工作空间下创建新的目录和函数'
    },
    'app:update': {
      description: '可以修改工作空间的基本信息（名称、描述等）'
    },
    'app:delete': {
      description: '可以删除工作空间及其所有资源'
    },
    'app:deploy': {
      description: '可以部署工作空间到运行环境'
    },
    'app:manage': {
      description: '拥有工作空间的所有权限（查看、创建、更新、删除、部署），以及所有子资源的完整权限',
      inheritance: '子目录和子函数会继承所有相关权限（所有权）'
    },
    
    // 函数权限
    // 函数权限（统一权限点）
    'function:read': {
      description: '可以查看函数的基本信息和配置（适用于所有函数类型：table、form、chart 等）'
    },
    'function:write': {
      description: '可以执行写入操作（table 类型：新增记录；form 类型：提交表单）'
    },
    'function:update': {
      description: '可以执行更新操作（table 类型：更新记录）'
    },
    'function:delete': {
      description: '可以执行删除操作（table 类型：删除记录）'
    },
    'function:manage': {
      description: '拥有函数的所有权限，包括所有操作权限（查看、新增、更新、删除等）'
    },
    
    // ⭐ 统一权限点：form:write 已统一为 function:write，此条目保留用于向后兼容（可删除）
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
  apply_url: string  // 申请权限的 URL
  error_message: string  // 错误消息
}

/**
 * 检查节点是否有指定权限
 * ⭐ 优化：优先从权限缓存中获取，如果没有则从节点本身的 permissions 字段获取
 * ⭐ 支持权限层级关系：manage 权限包含所有其他权限
 * @param node 服务树节点
 * @param action 权限点（如 table:search、function:manage）
 * @returns 是否有权限
 */
export function hasPermission(node: ServiceTree | undefined, action: string): boolean {
  if (!node) {
    // 如果没有节点，默认返回 true（向后兼容）
    return true
  }

  // 获取权限对象（优先从缓存获取，否则从节点获取）
  const permissionStore = useNodePermissionsStore()
  const cachedPermissions = permissionStore.getPermissions(node)
  const permissions = cachedPermissions || node.permissions

  if (!permissions) {
    // 如果都没有，默认返回 true（向后兼容）
    return true
  }

  // 直接检查该权限
  if (action in permissions) {
    if (permissions[action] === true) {
      return true
    }
  }

  // ⭐ 权限层级关系：如果有 manage 权限，自动拥有所有相关权限
  // directory:manage 包含 directory:read、directory:write、directory:update、directory:delete、directory:create
  if (action.startsWith('directory:')) {
    if (permissions['directory:manage'] === true) {
      return true
    }
  }

  // function:manage 包含 function:read、function:write、function:update、function:delete
  if (action.startsWith('function:')) {
    if (permissions['function:manage'] === true) {
      return true
    }
  }

  // app:manage 包含 app:read、app:create、app:update、app:delete、app:deploy
  if (action.startsWith('app:')) {
    if (permissions['app:manage'] === true) {
      return true
    }
  }

  // 如果权限信息中没有该权限点，默认返回 true（向后兼容，避免权限信息不完整时按钮消失）
  return true
}

/**
 * 判断节点是否有任何权限（不指定具体权限点）
 * @param node 服务树节点
 * @returns 是否有任何权限
 */
export function hasAnyPermissionForNode(node: ServiceTree | undefined): boolean {
  if (!node) {
    return false
  }

  // ⭐ 优先从权限缓存中获取
  const permissionStore = useNodePermissionsStore()
  const cachedPermissions = permissionStore.getPermissions(node)
  if (cachedPermissions) {
    // 检查缓存中是否有任何权限为 true
    return Object.values(cachedPermissions).some(hasPerm => hasPerm === true)
  }

  // 如果没有缓存，从节点本身的 permissions 字段获取
  if (node.permissions) {
    // 检查节点权限信息中是否有任何权限为 true
    return Object.values(node.permissions).some(hasPerm => hasPerm === true)
  }

  // 如果都没有，返回 false（没有权限）
  return false
}

/**
 * 检查节点是否有多个权限（只要有一个有权限就返回 true）
 * ⭐ 优化：优先从权限缓存中获取
 * @param node 服务树节点
 * @param actions 权限点列表
 * @returns 是否有权限
 */
export function hasAnyPermission(node: ServiceTree | undefined, actions: string[]): boolean {
  if (!node) {
    return true
  }

  // ⭐ 优先从权限缓存中获取
  const permissionStore = useNodePermissionsStore()
  const cachedPermissions = permissionStore.getPermissions(node)
  if (cachedPermissions) {
    return actions.some(action => cachedPermissions[action] === true)
  }

  // 如果没有缓存，从节点本身的 permissions 字段获取（向后兼容）
  if (node.permissions) {
    return actions.some(action => node.permissions![action] === true)
  }

  return true
}

/**
 * 检查节点是否有所有权限（必须全部有权限才返回 true）
 * ⭐ 优化：优先从权限缓存中获取
 * @param node 服务树节点
 * @param actions 权限点列表
 * @returns 是否有权限
 */
export function hasAllPermissions(node: ServiceTree | undefined, actions: string[]): boolean {
  if (!node) {
    return true
  }

  // ⭐ 优先从权限缓存中获取
  const permissionStore = useNodePermissionsStore()
  const cachedPermissions = permissionStore.getPermissions(node)
  if (cachedPermissions) {
    return actions.every(action => cachedPermissions[action] === true)
  }

  // 如果没有缓存，从节点本身的 permissions 字段获取（向后兼容）
  if (node.permissions) {
    return actions.every(action => node.permissions![action] === true)
  }

  return true
}

/**
 * 获取权限显示名称
 * @param action 权限点
 * @returns 显示名称
 */
export function getPermissionDisplayName(action: string): string {
  const displayNames: Record<string, string> = {
    // Table 操作
    // 统一权限点：所有函数类型统一使用 function:read/write/update/delete
    // 显示名称根据模板类型在 getAvailablePermissions 中动态设置
    // Function 操作
    'function:read': '函数查看',
    'function:write': '函数写入',
    'function:update': '函数更新',
    'function:delete': '函数删除',
    'function:manage': '所有权',
    // Directory 操作
    'directory:read': '目录查看',
    'directory:write': '目录写入',
    'directory:update': '目录更新',
    'directory:delete': '目录删除',
    'directory:manage': '所有权',
    // App 操作（工作空间）
    'app:read': '工作空间查看',
    'app:create': '工作空间创建',
    'app:update': '工作空间更新',
    'app:delete': '工作空间删除',
    'app:deploy': '工作空间部署',
    'app:manage': '所有权',
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
    // ⭐ 统一权限点：所有函数类型统一使用 function:read/write/update/delete
    case 'table':
    case 'form':
    case 'chart':
    default:
      return ['function:read', 'function:write', 'function:update', 'function:delete', 'function:manage']
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
  // ⭐ 权限顺序：小权限（具体操作）在前，大权限（所有权/管理）在后
  if (resourceType === 'function') {
    // ⭐ 统一权限点：所有函数类型统一使用 function:read/write/update/delete
    // 根据模板类型显示不同的权限名称（但底层权限点统一）
    if (templateType === 'table') {
      permissions.push(
        { action: 'function:read', displayName: '表格查看', isMinimal: true },
        { action: 'function:write', displayName: '新增表格记录', isMinimal: false },
        { action: 'function:update', displayName: '更新表格记录', isMinimal: false },
        { action: 'function:delete', displayName: '删除表格记录', isMinimal: false }
      )
    } else if (templateType === 'form') {
      permissions.push(
        { action: 'function:write', displayName: '表单提交', isMinimal: true }
      )
      // form 类型虽然定义了 read/update/delete，但业务逻辑中不使用，所以不显示
    } else if (templateType === 'chart') {
      permissions.push(
        { action: 'function:read', displayName: '图表查看', isMinimal: true }
      )
      // chart 类型虽然定义了 write/update/delete，但业务逻辑中不使用，所以不显示
    } else {
      permissions.push(
        { action: 'function:read', displayName: '函数查看', isMinimal: true },
        { action: 'function:write', displayName: '函数写入', isMinimal: false },
        { action: 'function:update', displayName: '函数更新', isMinimal: false },
        { action: 'function:delete', displayName: '函数删除', isMinimal: false }
      )
    }
    
    // 大权限（所有权）放在最后
    permissions.push(
      { action: 'function:manage', displayName: '所有权', isMinimal: false, isManage: true }
    )
  } else if (resourceType === 'directory') {
    // 目录相关权限：小权限在前
    permissions.push(
      { action: 'directory:read', displayName: '目录查看', isMinimal: true },
      { action: 'directory:write', displayName: '目录写入', isMinimal: false },
      { action: 'directory:update', displayName: '目录更新', isMinimal: false },
      { action: 'directory:delete', displayName: '目录删除', isMinimal: false }
    )
    // 大权限（所有权）放在最后
    permissions.push(
      { action: 'directory:manage', displayName: '所有权', isMinimal: false, isManage: true }
    )
  } else if (resourceType === 'app') {
    // 工作空间相关权限：小权限在前
    permissions.push(
      { action: 'app:read', displayName: '工作空间查看', isMinimal: true },
      { action: 'app:create', displayName: '工作空间创建', isMinimal: false },
      { action: 'app:update', displayName: '工作空间更新', isMinimal: false },
      { action: 'app:delete', displayName: '工作空间删除', isMinimal: false },
      { action: 'app:deploy', displayName: '工作空间部署', isMinimal: false }
    )
    // 大权限（所有权）放在最后
    permissions.push(
      { action: 'app:manage', displayName: '所有权', isMinimal: false, isManage: true }
    )
  } else {
    // 未知类型，返回通用权限
    permissions.push(
      { action: 'function:read', displayName: '函数查看', isMinimal: true },
      { action: 'function:manage', displayName: '所有权', isMinimal: false, isManage: true }
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
  write: 'function:write',
  update: 'function:update',
  delete: 'function:delete',
  manage: 'function:manage',
} as const

/**
 * 检查 Form 函数的相关权限
 */
export const FormPermissions = {
  write: 'function:write',
  manage: 'function:manage',
} as const

/**
 * 检查 Chart 函数的相关权限
 */
export const ChartPermissions = {
  read: 'function:read',
  manage: 'function:manage',
} as const

/**
 * 检查目录的相关权限
 */
export const DirectoryPermissions = {
  read: 'directory:read',
  write: 'directory:write',
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
      displayName: `工作空间：${parsed.app}`,
      permissions: appPermissions,
      quickSelect: {
        label: '申请此工作空间的管理权限',
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
 * @param templateType 模板类型（table、form、chart，可选）
 * @returns 权限申请页面的 URL
 */
export function buildPermissionApplyURL(resourcePath: string, action: string, templateType?: string): string {
  let url = `/permissions/apply?resource=${encodeURIComponent(resourcePath)}&action=${encodeURIComponent(action)}`
  if (templateType) {
    url += `&templateType=${encodeURIComponent(templateType)}`
  }
  return url
}

