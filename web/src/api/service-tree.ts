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

// 创建服务目录（使用user和app参数）
export function createServiceTree(data: CreateServiceTreeRequest) {
  return post<ServiceTree>('/workspace/api/v1/service_tree', {
    user: data.user,
    app: data.app,
    name: data.name,
    code: data.code,
    parent_id: data.parent_id || 0,
    description: data.description || '',
    tags: data.tags || ''
  })
}

// 更新服务目录
export function updateServiceTree(id: number, data: Partial<ServiceTree>) {
  return put(`/workspace/api/v1/service_tree/${id}`, data)
}

// 删除服务目录
export function deleteServiceTree(id: number) {
  return del(`/workspace/api/v1/service_tree/${id}`)
}

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
  hub_directory_id?: number
  hub_version?: string
  hub_version_num?: number
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
  permissions?: Record<string, boolean>  // ⭐ 权限信息：directory:read, directory:create, directory:update, directory:delete, directory:manage
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

// Fork服务目录
export function forkServiceTree(id: number, targetAppId: number) {
  return post(`/workspace/api/v1/service_tree/${id}/fork`, {
    app_id: targetAppId
  })
}