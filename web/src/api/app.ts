import { get, post, put, del } from '@/utils/request'
import type { App, CreateAppRequest, CreateAppResponse } from '@/types'

// 获取工作空间列表
export function getAppList(pageSize: number = 200, search?: string) {
  // 后端返回的是分页数据结构: { page, page_size, total_count, items: App[] }
  // ⚠️ 注意：响应拦截器已经提取了 data 字段，所以 res 就是分页对象本身
  const params: Record<string, any> = {
    page_size: pageSize,
    page: 1
  }
  if (search) {
    params.search = search
  }
  return get<{
    page: number
    page_size: number
    total_count: number
    items: App[]
  }>('/api/v1/app/list', params).then(res => {
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
  // 后端期望的格式: { code: string, name: string }
  // user字段会自动从JWT Token获取，不需要前端传递
  // 后端返回的是 CreateAppResponse，不是完整的 App 对象
  return post<CreateAppResponse>('/api/v1/app/create', {
    code: data.code,
    name: data.name
  })
}

// 更新工作空间（重新编译）
export function updateApp(code: string) {
  return post(`/api/v1/app/update/${code}`, {})
}

// 删除工作空间
export function deleteApp(code: string) {
  return del(`/api/v1/app/delete/${code}`)
}

// 获取工作空间详情
export function getAppDetail(code: string) {
  return get<App>(`/api/v1/app/detail/${code}`)
}

// 运行业务系统函数
export function runFunction(fullCodePath: string, params?: any) {
  return post(`/api/v1/run/${fullCodePath}`, params)
}