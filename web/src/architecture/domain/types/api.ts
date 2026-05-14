// API响应基础类型
export interface ApiResponse<T = any> {
  code: number
  data: T
  message?: string
  msg?: string  // 统一使用 msg 字段（兼容 message）
  metadata?: Record<string, any>  // 元数据（如 total_cost_mill、trace_id 等）
}

// 用户相关类型
export interface UserInfo {
  id: number
  username: string
  email: string
  register_type: string
  avatar: string
  nickname?: string           // 昵称
  signature?: string          // 个人签名/简介
  gender?: string            // 性别: 'male' | 'female' | 'other' | ''
  email_verified: boolean
  status: string
  created_at: string
  department_full_path?: string      // 部门完整路径（可选）
  department_name?: string           // 部门名称（可选，用于显示）
  department_full_name_path?: string // 部门完整名称路径（可选，用于展示组织架构全称，如：技术部/后端组）
  leader_username?: string           // Leader 用户名（可选）
  leader_display_name?: string  // Leader 显示名称（可选，用于显示，格式：username(nickname) 或 username）
}

export interface LoginRequest {
  username: string
  password: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
  code?: string
}

// 应用相关类型
export interface App {
  id: number
  user: string
  code: string
  name: string
  nats_id: number
  host_id: number
  status: 'enabled' | 'disabled'
  type?: number  // 应用类型：0=用户空间，1=系统空间
  version: string
  is_public: boolean
  admins?: string
  show_only_permitted?: boolean  // 仅展示有权限的空间
  permission_enforced?: boolean  // 是否启用权限管控
  created_at: string
  updated_at: string
}

export interface CreateAppRequest {
  code: string
  name: string
  is_public?: boolean
  admins?: string
  /** 仅展示有权限的空间：开启后非管理员只看到有权限的目录（SaaS 多租户） */
  show_only_permitted?: boolean
  /** 是否启用权限管控：默认关闭，避免老工作空间升级后被阻塞 */
  permission_enforced?: boolean
}

// 创建应用响应（后端实际返回的结构）
export interface CreateAppResponse {
  user: string
  app: string  // 对应 App 的 code 字段
  app_dir: string
}

// 服务目录相关类型
export interface ServiceTree {
  id: number
  name: string
  code: string
  type: 'package' | 'function' | 'docs' | 'board'
  description: string
  tags: string
  admins?: string  // 节点管理员列表，逗号分隔的用户名
  pending_count?: number  // ⭐ 待审批的权限申请数量
  owner?: string   // 节点创建者（owner）
  app_id: number
  ref_id: number
  full_code_path: string
  full_group_code?: string  // 完整函数组代码：{full_path}/{group_code}，与 source_code.full_group_code 对齐
  group_name?: string  // 组名称（用于展示，不参与路由）
  template_type?: string  // 模板类型（函数的类型，如 form、table）
  has_function?: boolean  // ⭐ 是否有函数（仅对package类型有效）：如果该package下直接或间接包含function类型的子节点，则为true
  run_count?: number  // ⭐ 运行次数（仅 function 类型有意义），用于展示「已使用 N 次」
  is_admin?: boolean  // ⭐ 是否是管理员（企业版功能）：如果用户是工作空间管理员，则为 true，前端优先判断此字段，无需构造每个节点的权限
  permissions?: Record<string, boolean>  // ⭐ 权限标识（企业版功能）：权限点 -> 是否有权限
  created_at: string
  updated_at: string
  children?: ServiceTree[]
}

export interface CreateServiceTreeRequest {
  user: string
  app: string
  name: string
  code: string
  parent_full_code_path?: string
  type?: string  // 节点类型: 'package' | 'docs' | 'function' | 'board'
  description?: string
  tags?: string
  admins?: string  // 管理员列表，逗号分隔的用户名
  // ⭐ 文档相关字段（仅当 type=docs 时使用）
  doc_content?: string  // 文档内容
  doc_format?: string   // 文档格式（默认为 markdown）
  doc_summary?: string  // 文档摘要（可选）
}

// 统一类型系统：从本目录 field.ts 重新导出 Widget 相关类型
export type { 
  FieldConfig, 
  WidgetConfig, 
  FieldValue,
  FieldMeta,
  FunctionDetail,
  FunctionSchema,
  WidgetMode,
  ValidationRule,
  PermissionConfig
} from './field'

// 导出 WidgetTypes 命名空间（推荐新代码使用）
export type { WidgetTypes } from './field'

// 函数相关类型
export interface Function {
  id: number
  app_id: number
  tree_id: number
  method: string
  router: string
  has_config: boolean
  create_tables: string
  schema: import('./field').FunctionSchema
  template_type: string
  created_at: string
  updated_at: string
  created_by?: string  // 创建者用户名
}

// 组件类型枚举
export enum WidgetType {
  INPUT = 'input',
  TEXT = 'text',
  TEXT_AREA = 'text_area',
  SELECT = 'select',
  SWITCH = 'switch',
  DATETIME = 'datetime',
  USER = 'user',
  USERS = 'users',
  DEPARTMENT = 'department',
  DEPARTMENTS = 'departments',
  ID = 'ID',
  NUMBER = 'number',
  FLOAT = 'float',
  FILES = 'files',
  CHECKBOX = 'checkbox',
  RADIO = 'radio',
  MULTI_SELECT = 'multiselect',
  SLIDER = 'slider',
  RATE = 'rate',
  COLOR = 'color',
  RICH_TEXT = 'richtext',
  TABLE = 'table',
  FORM = 'form',
  LINK = 'link',
  PROGRESS = 'progress',
  LIST = 'list'
}

// Table 查询参数
export interface SearchParams {
  sorts?: string    // 结构化排序 JSON
  page?: number     // 页码
  page_size?: number // 页大小
  [key: string]: any
}

// 表格响应类型
export interface TableResponse<T = any> {
  list: T[]
  total: number
  page: number
  page_size: number
}

// 路由配置
export interface RouteConfig {
  path: string
  name: string
  component: any
  meta?: {
    title?: string
    icon?: string
    requireAuth?: boolean
  }
}

// WebSocket消息类型
export interface WSMessage {
  type: string
  data: any
  timestamp: number
}

// Fork 函数组相关类型
export interface ForkFunctionGroupRequest {
  source_to_target_map: Record<string, string>  // key=函数组的full_group_code，value=服务目录的full_code_path
  target_app_id: number  // 目标应用 ID
}

export interface ForkFunctionGroupResponse {
  message: string  // 响应消息
}
