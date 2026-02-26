import { get, post, put, del } from '@/utils/request'
import type { ServiceTree, CreateServiceTreeRequest } from '@/types'

// 获取服务目录树（使用user和app参数）
// @param typeFilter 可选，节点类型过滤：'package'（只显示服务目录/包）、'function'（只显示函数/文件）
export function getServiceTree(user: string, app: string, typeFilter?: 'package' | 'function') {
  const params: Record<string, string> = { user, app }
  if (typeFilter) {
    params.type = typeFilter
  }
  return get<ServiceTree[]>('/workspace/api/v1/service_tree', params)
}

// ⭐ 创建 package 类型节点（推荐使用）
export function createPackage(data: CreateServiceTreeRequest) {
  const payload = {
    user: data.user,
    app: data.app,
    name: data.name,
    code: data.code,
    parent_id: data.parent_id || 0,
    description: data.description || '',
    tags: data.tags || '',
    admins: data.admins || ''
  }
  return post<ServiceTree>('/workspace/api/v1/packages', payload)
}

// ⭐ 创建 docs 类型节点（推荐使用）
export function createDocs(data: CreateServiceTreeRequest & {
  content?: string
  format?: string
  summary?: string
}) {
  const payload: any = {
    user: data.user,
    app: data.app,
    name: data.name,
    code: data.code,
    parent_id: data.parent_id || 0,
    description: data.description || '',
    tags: data.tags || '',
    admins: data.admins || ''
  }
  
  if (data.content) {
    payload.content = data.content
  }
  if (data.format) {
    payload.format = data.format
  }
  if (data.summary) {
    payload.summary = data.summary
  }
  
  return post<ServiceTree>('/workspace/api/v1/docs/crud', payload)
}

// ⭐ 创建 function 类型节点（推荐使用）
// 注意：为了避免与 function.ts 中的 createFunction 冲突，这里命名为 createServiceTreeFunction
export function createServiceTreeFunction(data: {
  user: string
  app: string
  name: string
  code: string
  directory_path: string
  template_type?: string
  source_code: string
  description?: string
  tags?: string
}) {
  const payload = {
    user: data.user,
    app: data.app,
    name: data.name,
    code: data.code,
    directory_path: data.directory_path,
    template_type: data.template_type || '',
    source_code: data.source_code,
    description: data.description || '',
    tags: data.tags || ''
  }
  return post<ServiceTree>('/workspace/api/v1/functions', payload)
}

// 创建服务目录（使用user和app参数）
// ⚠️ 保留向后兼容，推荐使用 createPackage、createDocs、createFunction
export function createServiceTree(data: CreateServiceTreeRequest & { type?: string }) {
  const payload: any = {
    user: data.user,
    app: data.app,
    name: data.name,
    code: data.code,
    parent_id: data.parent_id || 0,
    type: data.type || 'package',
    description: data.description || '',
    tags: data.tags || '',
    admins: data.admins || '' // ⭐ 添加管理员字段
  }
  
  // ⭐ 如果是 docs 类型，添加文档相关字段
  if (data.type === 'docs') {
    if (data.doc_content) {
      payload.doc_content = data.doc_content
    }
    if (data.doc_format) {
      payload.doc_format = data.doc_format
    }
    if (data.doc_summary) {
      payload.doc_summary = data.doc_summary
    }
  }
  
  return post<ServiceTree>('/workspace/api/v1/service_tree', payload)
}

// ⭐ 更新 package 类型节点（推荐使用）
export function updatePackage(id: number, data: { name?: string; code?: string; description?: string; tags?: string; admins?: string }) {
  return put(`/workspace/api/v1/packages/${id}`, data)
}

// ⭐ 删除 package 类型节点（推荐使用）
export function deletePackage(id: number) {
  return del(`/workspace/api/v1/packages/${id}`)
}

// ⭐ 更新 function 类型节点（推荐使用）
// 注意：为了避免与 function.ts 中的 updateFunction 冲突，这里命名为 updateServiceTreeFunction
export function updateServiceTreeFunction(id: number, data: { name?: string; code?: string; description?: string; tags?: string }) {
  return put(`/workspace/api/v1/functions/${id}`, data)
}

// ⭐ 删除 function 类型节点（推荐使用）
// 注意：为了避免与 function.ts 中的 deleteFunction 冲突，这里命名为 deleteServiceTreeFunction
export function deleteServiceTreeFunction(id: number) {
  return del(`/workspace/api/v1/functions/${id}`)
}

// ⭐ 更新 docs 类型节点（推荐使用）
export function updateDocs(id: number, data: { name?: string; code?: string; description?: string; tags?: string; admins?: string; content?: string; format?: string; summary?: string }) {
  return put(`/workspace/api/v1/docs/crud/${id}`, data)
}

// ⭐ 删除 docs 类型节点（推荐使用）
export function deleteDocs(id: number) {
  return del(`/workspace/api/v1/docs/crud/${id}`)
}

// ⭐ 创建 board 类型节点（版块/讨论区）
export function createBoard(data: CreateServiceTreeRequest) {
  const payload = {
    user: data.user,
    app: data.app,
    name: data.name,
    code: data.code,
    parent_id: data.parent_id || 0,
    description: data.description || '',
    tags: data.tags || '',
    admins: data.admins || ''
  }
  return post<ServiceTree>('/workspace/api/v1/boards/crud', payload)
}

// ⭐ 更新 board 类型节点
export function updateBoard(id: number, data: { name?: string; description?: string; tags?: string; admins?: string }) {
  return put(`/workspace/api/v1/boards/crud/${id}`, data)
}

// ⭐ 删除 board 类型节点（会先删除该版块下全部帖子）
export function deleteBoard(id: number) {
  return del(`/workspace/api/v1/boards/crud/${id}`)
}

// 更新服务目录
// ⚠️ 保留向后兼容，推荐使用 updatePackage、updateFunction、updateDocs
export function updateServiceTree(id: number, data: { name?: string; admins?: string }) {
  return put('/workspace/api/v1/service_tree', {
    id,
    name: data.name,
    admins: data.admins
  })
}

// 删除服务目录
// ⚠️ 保留向后兼容，推荐使用 deletePackage、deleteFunction、deleteDocs
export function deleteServiceTree(id: number) {
  return del(`/workspace/api/v1/service_tree/${id}`)
}

// 文档相关 API
export interface Doc {
  id: number
  title: string
  content: string
  format: string
  app_id: number
  tree_id: number
  summary?: string
  category?: string
  created_at: string
  updated_at: string
}

// 文档相关 API 已迁移到 @/api/doc.ts，使用基于 full_code_path 的新接口

// 获取服务目录详情（包含权限信息）
export interface ServiceTreeDetail {
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
  template_type?: string
  version: string
  version_num: number
  hub_full_code_path?: string
  hub_version_num?: number
  run_count?: number  // ⭐ 运行次数（仅 function 类型有意义），用于展示「已使用 N 次」
  permissions?: Record<string, boolean>  // ⭐ 权限信息
}

// 获取服务目录详情（支持 ID 或 full_code_path）
// ⚠️ 注意：函数权限请使用函数详情接口，此接口主要用于兼容旧代码
export function getServiceTreeDetail(params: { id?: number; full_code_path?: string }) {
  const queryParams: Record<string, string> = {}
  if (params.id) {
    queryParams.id = params.id.toString()
  }
  if (params.full_code_path) {
    queryParams.full_code_path = params.full_code_path
  }
  return get<ServiceTreeDetail>('/workspace/api/v1/service_tree/detail', queryParams)
}

// 获取目录信息（仅用于获取目录权限）
export interface PackageInfo {
  id: number
  name: string
  code: string
  full_code_path: string
  permissions?: Record<string, boolean>  // ⭐ 权限信息：directory:read, directory:write, directory:update, directory:delete, directory:admin
}

// 获取目录信息（支持 ID 或 full_code_path）
// ⭐ 优化：专门用于获取目录权限，函数权限从函数详情接口获取
export function getPackageInfo(params: { id?: number; full_code_path?: string }) {
  const queryParams: Record<string, string> = {}
  if (params.id) {
    queryParams.id = params.id.toString()
  }
  if (params.full_code_path) {
    queryParams.full_code_path = params.full_code_path
  }
  return get<PackageInfo>('/workspace/api/v1/service_tree/package_info', queryParams)
}

// 移动服务目录
export function moveServiceTree(id: number, newParentId: number) {
  return put(`/workspace/api/v1/service_tree/${id}/move`, { parent_id: newParentId })
}

// 复制服务目录（新接口，支持递归复制）
export function copyDirectory(data: {
  source_directory_path: string
  target_directory_path: string
  target_app_id: number
}) {
  return post<{
    message: string
    directory_count: number
    file_count: number
  }>('/workspace/api/v1/service_tree/copy', data)
}

// 复制服务目录（旧接口，保留向后兼容）
export function copyServiceTree(id: number, targetAppId: number, targetParentId?: number) {
  return post(`/workspace/api/v1/service_tree/${id}/copy`, {
    app_id: targetAppId,
    parent_id: targetParentId
  })
}

// 搜索函数
export interface SearchFunctionsReq {
  user: string
  app: string
  keyword?: string
  template_type?: string
  page: number
  page_size: number
}

export interface FunctionSearchResult {
  id: number
  name: string
  code: string
  full_code_path: string
  description: string
  template_type: string
  app_id: number
  app_user: string
  app_code: string
  /** 请求参数（表单/接口入参结构），便于构造 run_form_submit 的 body */
  request?: unknown[]
  /** 响应参数（返回结构说明） */
  response?: unknown[]
}

export interface SearchFunctionsResp {
  functions: FunctionSearchResult[]
  total: number
  page: number
  page_size: number
}

export function searchFunctions(req: SearchFunctionsReq) {
  return get<SearchFunctionsResp>('/workspace/api/v1/service_tree/search_functions', {
    user: req.user,
    app: req.app,
    keyword: req.keyword || '',
    template_type: req.template_type || '',
    page: req.page.toString(),
    page_size: req.page_size.toString()
  })
}