/**
 * 权限常量定义
 * 
 * 说明：
 * 1. 所有权限点格式：resource_type:action_type
 * 2. 资源类型：directory、function、table、form、chart、docs
 * 3. 操作类型：read、write、update、delete、admin
 * 
 * ⭐ 前后端统一使用此常量定义，避免硬编码
 */

// ========================================
// 操作类型常量
// ========================================

/**
 * 操作类型枚举
 */
export const ActionType = {
  /** 查看权限 */
  read: 'read',
  /** 写入权限 */
  write: 'write',
  /** 更新权限 */
  update: 'update',
  /** 删除权限 */
  delete: 'delete',
  /** 管理员权限（包含所有权限） */
  admin: 'admin',
  /** 查询权限（用于 chart） */
  query: 'query',
} as const

// ========================================
// 资源类型常量
// ========================================

/**
 * 资源类型枚举
 */
export const ResourceType = {
  /** 目录（包括根目录/工作空间） */
  directory: 'directory',
  /** 函数 */
  function: 'function',
  /** 表格 */
  table: 'table',
  /** 表单 */
  form: 'form',
  /** 图表 */
  chart: 'chart',
  /** 文档 */
  docs: 'docs',
  /** 讨论区/板块 */
  board: 'board',
  /** 工作流 */
  workflow: 'workflow',
} as const

// ========================================
// 权限点常量（按资源类型分组）
// ========================================

/**
 * 目录权限（包括根目录/工作空间）
 * ⭐ 工作空间就是根目录，使用相同的权限体系
 */
export const DirectoryPermission = {
  /** 查看目录 */
  read: 'directory:read',
  /** 创建子目录/子资源 */
  write: 'directory:write',
  /** 更新目录信息 */
  update: 'directory:update',
  /** 删除目录 */
  delete: 'directory:delete',
  /** 目录管理员（拥有所有权限） */
  admin: 'directory:admin',
} as const

/**
 * 函数权限
 */
export const FunctionPermission = {
  /** 查看函数 */
  read: 'function:read',
  /** 创建/写入函数 */
  write: 'function:write',
  /** 更新函数 */
  update: 'function:update',
  /** 删除函数 */
  delete: 'function:delete',
  /** 函数管理员（拥有所有权限） */
  admin: 'function:admin',
} as const

/**
 * 表格权限
 */
export const TablePermission = {
  /** 查看表格 */
  read: 'table:read',
  /** 插入数据 */
  write: 'table:write',
  /** 更新数据 */
  update: 'table:update',
  /** 删除数据 */
  delete: 'table:delete',
  /** 表格管理员（拥有所有权限） */
  admin: 'table:admin',
} as const

/**
 * 表单权限
 */
export const FormPermission = {
  /** 查看表单 */
  read: 'form:read',
  /** 提交表单 */
  write: 'form:write',
  /** 更新表单数据 */
  update: 'form:update',
  /** 删除表单数据 */
  delete: 'form:delete',
  /** 表单管理员（拥有所有权限） */
  admin: 'form:admin',
} as const

/**
 * 图表权限
 */
export const ChartPermission = {
  /** 查看图表 */
  read: 'chart:read',
  /** 查询图表数据 */
  query: 'chart:query',
  /** 更新图表配置 */
  update: 'chart:update',
  /** 删除图表 */
  delete: 'chart:delete',
  /** 图表管理员（拥有所有权限） */
  admin: 'chart:admin',
} as const

/**
 * 文档权限
 */
export const DocsPermission = {
  /** 查看文档 */
  read: 'docs:read',
  /** 创建/写入文档 */
  write: 'docs:write',
  /** 更新文档 */
  update: 'docs:update',
  /** 删除文档 */
  delete: 'docs:delete',
  /** 文档管理员（拥有所有权限） */
  admin: 'docs:admin',
} as const

/**
 * 讨论区/板块权限
 */
export const BoardPermission = {
  /** 查看帖子 */
  read: 'board:read',
  /** 发帖 */
  write: 'board:write',
  /** 更新帖子 */
  update: 'board:update',
  /** 删除帖子 */
  delete: 'board:delete',
  /** 板块管理员（拥有所有权限） */
  admin: 'board:admin',
} as const

/**
 * 工作流权限
 */
export const WorkflowPermission = {
  /** 查看工作流 */
  read: 'workflow:read',
  /** 编辑工作流草稿 */
  write: 'workflow:write',
  /** 发布/更新工作流 */
  update: 'workflow:update',
  /** 删除工作流 */
  delete: 'workflow:delete',
  /** 工作流管理员（拥有所有权限） */
  admin: 'workflow:admin',
} as const

// ========================================
// 统一导出（保持向后兼容）
// ========================================

/**
 * 所有权限点的集合
 */
export const Permission = {
  Directory: DirectoryPermission,
  Function: FunctionPermission,
  Table: TablePermission,
  Form: FormPermission,
  Chart: ChartPermission,
  Docs: DocsPermission,
  Board: BoardPermission,
  Workflow: WorkflowPermission,
} as const

// ========================================
// 工具函数
// ========================================

/**
 * 构建权限点
 * @param resourceType 资源类型
 * @param actionType 操作类型
 * @returns 权限点字符串（如 'directory:read'）
 */
export function buildPermission(
  resourceType: keyof typeof ResourceType,
  actionType: keyof typeof ActionType
): string {
  return `${ResourceType[resourceType]}:${ActionType[actionType]}`
}

/**
 * 解析权限点
 * @param permission 权限点字符串（如 'directory:read'）
 * @returns { resourceType, actionType } 或 null
 */
export function parsePermission(permission: string): {
  resourceType: string
  actionType: string
} | null {
  const parts = permission.split(':')
  if (parts.length !== 2) {
    return null
  }
  const resourceType = parts[0]
  const actionType = parts[1]
  if (!resourceType || !actionType) {
    return null
  }
  return {
    resourceType,
    actionType,
  }
}

/**
 * 根据资源类型获取权限对象
 * @param resourceType 资源类型
 * @returns 权限对象
 */
export function getPermissionsByResourceType(resourceType: string) {
  switch (resourceType) {
    case ResourceType.directory:
      return DirectoryPermission
    case ResourceType.function:
      return FunctionPermission
    case ResourceType.table:
      return TablePermission
    case ResourceType.form:
      return FormPermission
    case ResourceType.chart:
      return ChartPermission
    case ResourceType.docs:
      return DocsPermission
    case ResourceType.board:
      return BoardPermission
    case ResourceType.workflow:
      return WorkflowPermission
    default:
      return null
  }
}

// ========================================
// TypeScript 类型定义
// ========================================

/** 所有权限点的联合类型 */
export type PermissionString =
  | (typeof DirectoryPermission)[keyof typeof DirectoryPermission]
  | (typeof FunctionPermission)[keyof typeof FunctionPermission]
  | (typeof TablePermission)[keyof typeof TablePermission]
  | (typeof FormPermission)[keyof typeof FormPermission]
  | (typeof ChartPermission)[keyof typeof ChartPermission]
  | (typeof DocsPermission)[keyof typeof DocsPermission]
  | (typeof BoardPermission)[keyof typeof BoardPermission]
  | (typeof WorkflowPermission)[keyof typeof WorkflowPermission]

/** 资源类型的联合类型 */
export type ResourceTypeString = (typeof ResourceType)[keyof typeof ResourceType]

/** 操作类型的联合类型 */
export type ActionTypeString = (typeof ActionType)[keyof typeof ActionType]
