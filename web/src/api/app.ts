import { get, post, put, del } from '@/utils/request'
import type { App, CreateAppRequest } from '@/types'

// 获取应用列表
export function getAppList() {
  // 后端返回的是分页数据结构: { page, page_size, total_count, items: App[] }
  return get<{
    page: number
    page_size: number
    total_count: number
    items: App[]
  }>('/api/v1/app/list').then(res => res.items || [])
}

// 创建应用
export function createApp(data: CreateAppRequest) {
  // 后端期望的格式: { code: string, name: string }
  // user字段会自动从JWT Token获取，不需要前端传递
  return post<App>('/api/v1/app/create', {
    code: data.code,
    name: data.name
  })
}

// 更新应用（重新编译）
export function updateApp(code: string) {
  return post(`/api/v1/app/update/${code}`, {})
}

// 删除应用
export function deleteApp(code: string) {
  return del(`/api/v1/app/delete/${code}`)
}

// 获取应用详情
export function getAppDetail(code: string) {
  return get<App>(`/api/v1/app/detail/${code}`)
}

// 运行应用函数
export function runFunction(fullCodePath: string, params?: any) {
  return post(`/api/v1/run/${fullCodePath}`, params)
}