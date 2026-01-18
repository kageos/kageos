import { get, post, put, del } from '@/utils/request'
import type { App, CreateAppRequest, CreateAppResponse } from '@/types'

// 获取工作空间列表
export function getAppList(pageSize: number = 200, search?: string, includeAll: boolean = false, type?: number) {
  // 后端返回的是分页数据结构: { page, page_size, total_count, items: App[] }
  // ⚠️ 注意：响应拦截器已经提取了 data 字段，所以 res 就是分页对象本身
  const params: Record<string, any> = {
    page_size: pageSize,
    page: 1
  }
  if (search) {
    params.search = search
  }
  if (includeAll) {
    params.include_all = true
  }
  if (type !== undefined) {
    params.type = type
  }
  return get<{
    page: number
    page_size: number
    total_count: number
    items: App[]
  }>('/workspace/api/v1/app/list', params).then(res => {
    // ⚠️ 响应拦截器返回的是 data 对象，所以 res 就是分页对象
    // 需要检查 res 是否有 items 字段
    if (res && typeof res === 'object' && 'items' in res) {
      return (res as any).items || []
    }
    // 如果 res 本身就是数组，直接返回
    if (Array.isArray(res)) {
      return res
    }
    // 否则返回空数组
    return []
  })
}

// 创建工作空间
export function createApp(data: CreateAppRequest) {
  // 后端期望的格式: { code: string, name: string, is_public?: boolean, admins?: string }
  // user字段会自动从JWT Token获取，不需要前端传递
  // 后端返回的是 CreateAppResponse，不是完整的 App 对象
  const payload: Record<string, any> = {
    code: data.code,
    name: data.name
  }
  if (data.is_public !== undefined) {
    payload.is_public = data.is_public
  }
  if (data.admins !== undefined && data.admins !== '') {
    payload.admins = data.admins
  }
  return post<CreateAppResponse>('/workspace/api/v1/app/create', payload)
}

// 更新工作空间（重新编译）
export function updateApp(code: string) {
  return post(`/workspace/api/v1/app/update/${code}`, {})
}

// 删除工作空间
export function deleteApp(code: string) {
  return del(`/workspace/api/v1/app/delete/${code}`)
}

// 获取工作空间详情
export function getAppDetail(code: string) {
  return get<App>(`/workspace/api/v1/app/detail/${code}`)
}

// 根据 user 和 code 获取工作空间详情（创建后使用）
export function getAppDetailByUserAndCode(user: string, code: string) {
  // 注意：后端接口只需要 code，user 从 JWT Token 获取
  return get<App>(`/workspace/api/v1/app/detail/${code}`)
}

// ⭐ 获取工作空间详情和服务目录树（合并接口，减少请求次数）
// 使用 full-code-path（至少包含 user/app）
export function getAppWithServiceTree(user: string, app: string, nodeType?: string) {
  const params: Record<string, any> = {}
  if (nodeType) {
    params.type = nodeType
  }
  // ⭐ 使用 full-code-path：/{user}/{app}/tree
  return get<{
    app: App
    service_tree: import('@/types').ServiceTree[]
    expanded_keys?: number[] // ⭐ 需要自动展开的节点ID列表（包含所有 pending_count > 0 的节点及其父节点）
  }>(`/workspace/api/v1/app/${user}/${app}/tree`, params)
}

// 更新工作空间配置（只更新 MySQL 记录，不涉及容器更新）
export function updateWorkspace(user: string, app: string, data: { admins?: string }) {
  return put<{
    user: string
    app: string
    admins: string
  }>(`/workspace/api/v1/app/workspace/${user}/${app}`, data)
}

// 运行业务系统函数
export function runFunction(fullCodePath: string, params?: any) {
  return post(`/workspace/api/v1/run/${fullCodePath}`, params)
}