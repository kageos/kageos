/**
 * 术语映射配置
 * 
 * 代码层面使用技术术语（App、Function、ServiceTree）
 * 用户界面使用业务术语（工作空间、业务系统、服务目录）
 * 
 * 使用方式：
 * ```typescript
 * import { TERMINOLOGY } from '@/config/terminology'
 * 
 * // 在 UI 中使用
 * <el-button>创建{{ TERMINOLOGY.workspace }}</el-button>
 * ```
 */

export const TERMINOLOGY = {
  // 工作空间相关
  workspace: '工作空间',
  workspacePlural: '工作空间',
  createWorkspace: '创建工作空间',
  workspaceList: '工作空间列表',
  workspaceName: '工作空间名称',
  workspaceCode: '工作空间代码',
  selectWorkspace: '选择工作空间',
  switchWorkspace: '切换工作空间',
  workspaceDetail: '工作空间详情',
  workspaceManagement: '工作空间管理',
  
  // 业务系统相关
  businessSystem: '业务系统',
  businessSystemPlural: '业务系统',
  createBusinessSystem: '创建业务系统',
  businessSystemList: '业务系统列表',
  businessSystemName: '业务系统名称',
  businessSystemCode: '业务系统代码',
  selectBusinessSystem: '选择业务系统',
  businessSystemDetail: '业务系统详情',
  functionGroup: '业务系统', // 函数组 = 业务系统
  
  // 服务目录相关
  serviceDirectory: '服务目录',
  serviceDirectoryPlural: '服务目录',
  createServiceDirectory: '创建服务目录',
  serviceDirectoryList: '服务目录列表',
  serviceDirectoryName: '服务目录名称',
  serviceDirectoryCode: '服务目录代码',
  selectServiceDirectory: '选择服务目录',
  package: '服务目录', // Package = 服务目录
  
  // API 相关
  api: 'API 接口',
  apiPlural: 'API 接口',
  apiList: 'API 接口列表',
  apiDetail: 'API 接口详情',
  
  // 其他
  publishToHub: '发布到应用市场',
  cloneFromHub: '从应用市场克隆',
} as const

/**
 * 获取用户术语
 * 
 * @param key 术语键
 * @returns 用户术语
 * 
 * @example
 * getUserTerm('workspace') // '工作空间'
 * getUserTerm('businessSystem') // '业务系统'
 */
export function getUserTerm(key: keyof typeof TERMINOLOGY): string {
  return TERMINOLOGY[key] || key
}

/**
 * 术语映射（代码层面 → 用户层面）
 */
export const TERM_MAP = {
  // 工作空间相关
  app: TERMINOLOGY.workspace,
  App: TERMINOLOGY.workspace,
  workspace: TERMINOLOGY.workspace,
  Workspace: TERMINOLOGY.workspace,
  
  // 业务系统相关
  function: TERMINOLOGY.businessSystem,
  Function: TERMINOLOGY.businessSystem,
  functionGroup: TERMINOLOGY.businessSystem,
  FunctionGroup: TERMINOLOGY.businessSystem,
  businessSystem: TERMINOLOGY.businessSystem,
  BusinessSystem: TERMINOLOGY.businessSystem,
  
  // 服务目录相关
  package: TERMINOLOGY.serviceDirectory,
  Package: TERMINOLOGY.serviceDirectory,
  serviceTree: TERMINOLOGY.serviceDirectory,
  ServiceTree: TERMINOLOGY.serviceDirectory,
  serviceDirectory: TERMINOLOGY.serviceDirectory,
  ServiceDirectory: TERMINOLOGY.serviceDirectory,
  
  // API 相关
  api: TERMINOLOGY.api,
  API: TERMINOLOGY.api,
} as const

/**
 * 将代码术语转换为用户术语
 * 
 * @param codeTerm 代码术语
 * @returns 用户术语
 * 
 * @example
 * translateTerm('app') // '工作空间'
 * translateTerm('function') // '业务系统'
 */
export function translateTerm(codeTerm: string): string {
  return TERM_MAP[codeTerm as keyof typeof TERM_MAP] || codeTerm
}

