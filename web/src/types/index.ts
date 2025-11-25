// API响应基础类型
export interface ApiResponse<T = any> {
  code: number
  data: T
  message?: string
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
  version: string
  created_at: string
  updated_at: string
}

export interface CreateAppRequest {
  code: string
  name: string
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
  parent_id: number
  type: 'package' | 'function'
  description: string
  tags: string
  app_id: number
  ref_id: number
  full_code_path: string
  full_group_code?: string  // 完整函数组代码：{full_path}/{group_code}，与 source_code.full_group_code 对齐
  group_name?: string  // 组名称（用于展示，不参与路由）
  created_at: string
  updated_at: string
  children?: ServiceTree[]
}

export interface CreateServiceTreeRequest {
  user: string
  app: string
  name: string
  code: string
  parent_id?: number
  description?: string
  tags?: string
}

// 函数相关类型
export interface Function {
  id: number
  request: any
  response: FieldConfig[]
  app_id: number
  tree_id: number
  method: string
  router: string
  has_config: boolean
  create_tables: string
  callbacks: string
  template_type: string
  created_at: string
  updated_at: string
}

// 字段配置类型
export interface FieldConfig {
  code: string
  name: string
  data: {
    type: string
    format?: string
    source?: string
    example?: string
    default_value?: string
  }
  desc?: string
  search?: string | null
  table_permission?: string | null
  widget: WidgetConfig
  callbacks?: any
  permission?: string | null
  validation?: string
}

// 组件配置类型
export interface WidgetConfig {
  type: string
  config: Record<string, any>
}

// 组件类型枚举
export enum WidgetType {
  INPUT = 'input',
  SELECT = 'select',
  TEXT_AREA = 'text_area',
  FILE_UPLOAD = 'file_upload',
  USER = 'user',
  DATETIME = 'datetime',
  NUMBER = 'number',
  SWITCH = 'switch',
  CHECKBOX = 'checkbox',
  RADIO = 'radio'
}

// 搜索类型
export interface SearchParams {
  eq?: string       // 精确匹配 eq=id:1
  like?: string     // 模糊匹配 like=title:xxx
  in?: string       // 包含查询 in=status:待处理,处理中
  contains?: string // 包含查询（用于多选场景，使用 FIND_IN_SET）contains=tags:高,中
  gte?: string      // 大于等于 gte=created_at:timestamp
  lte?: string      // 小于等于 lte=created_at:timestamp
  sorts?: string    // 排序 sorts=category:asc,price:desc（支持多列排序，格式：field:order,field:order）
  page?: number     // 页码
  page_size?: number // 页大小
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