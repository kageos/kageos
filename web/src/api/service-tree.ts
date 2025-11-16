import { get, post, put, del } from '@/utils/request'
import type { ServiceTree, CreateServiceTreeRequest } from '@/types'

// 获取服务目录树（使用user和app参数）
// @param typeFilter 可选，节点类型过滤：'package'（只显示服务目录/包）、'function'（只显示函数/文件）
export function getServiceTree(user: string, app: string, typeFilter?: 'package' | 'function') {
  const params: Record<string, string> = { user, app }
  if (typeFilter) {
    params.type = typeFilter
  }
  return get<ServiceTree[]>('/api/v1/service_tree', params)
}

// 创建服务目录（使用user和app参数）
export function createServiceTree(data: CreateServiceTreeRequest) {
  return post<ServiceTree>('/api/v1/service_tree', {
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
  return put(`/api/v1/service_tree/${id}`, data)
}

// 删除服务目录
export function deleteServiceTree(id: number) {
  return del(`/api/v1/service_tree/${id}`)
}

// 获取服务目录详情
export function getServiceTreeDetail(id: number) {
  return get<ServiceTree>(`/api/v1/service_tree/${id}`)
}

// 移动服务目录
export function moveServiceTree(id: number, newParentId: number) {
  return put(`/api/v1/service_tree/${id}/move`, { parent_id: newParentId })
}

// 复制服务目录
export function copyServiceTree(id: number, targetAppId: number, targetParentId?: number) {
  return post(`/api/v1/service_tree/${id}/copy`, {
    app_id: targetAppId,
    parent_id: targetParentId
  })
}

// Fork服务目录
export function forkServiceTree(id: number, targetAppId: number) {
  return post(`/api/v1/service_tree/${id}/fork`, {
    app_id: targetAppId
  })
}