import { get, post, put, del } from '@/utils/request'
import type { Function, FieldConfig, SearchParams, TableResponse } from '@/types'

// 获取函数详情
export function getFunctionDetail(functionId: number) {
  return get<Function>(`/api/v1/function/get`, { function_id: functionId })
}

// 根据路径获取函数详情
export function getFunctionByPath(fullCodePath: string) {
  return get<Function>(`/api/v1/function/by-path`, { path: fullCodePath })
}

// 获取应用下所有函数
export function getFunctionList(appId: number) {
  return get<Function[]>('/api/v1/function/list', { app_id: appId })
}

// 获取服务目录下函数列表
export function getFunctionByTree(treeId: number) {
  return get<Function[]>(`/api/v1/function/tree/${treeId}`)
}

// 执行函数（通用）
export function executeFunction(method: string, router: string, params?: SearchParams | any) {
  const url = `/api/v1/run${router}`
  
  // 根据方法类型调用不同的请求方法
  switch (method.toUpperCase()) {
    case 'GET':
      // GET 请求，params 作为查询参数
      return get<any>(url, params || {})
    case 'POST':
      // POST 请求，params 作为 body
      return post<any>(url, params || {})
    case 'PUT':
      // PUT 请求，params 作为 body
      return put<any>(url, params || {})
    case 'DELETE':
      // DELETE 请求，params 作为查询参数
      return del<any>(url)
    default:
      // 默认使用 GET
      return get<any>(url, params || {})
  }
}

// 创建函数
export function createFunction(data: Partial<Function>) {
  return post<Function>('/api/v1/function/create', data)
}

// 更新函数
export function updateFunction(id: number, data: Partial<Function>) {
  return put(`/api/v1/function/${id}`, data)
}

// 删除函数
export function deleteFunction(id: number) {
  return del(`/api/v1/function/${id}`)
}

// Table 回调操作 - 新增记录
// 统一使用 POST 方法，原函数的 method 通过 _method 查询参数传递，参数放在 body 里
export function tableAddRow(method: string, router: string, data: any) {
  const url = `/api/v1/callback${router}?_type=OnTableAddRow&_method=${method.toUpperCase()}`
  
  console.log('[tableAddRow] 准备新增')
  console.log('[tableAddRow]   Original Method:', method)
  console.log('[tableAddRow]   URL:', url)
  console.log('[tableAddRow]   Body:', data)
  
  // 统一使用 POST 方法
  return post(url, data)
}

// Table 回调操作 - 更新记录
// 统一使用 POST 方法，原函数的 method 通过 _method 查询参数传递，参数放在 body 里
export function tableUpdateRow(method: string, router: string, data: any) {
  const url = `/api/v1/callback${router}?_type=OnTableUpdateRow&_method=${method.toUpperCase()}`
  
  console.log('[tableUpdateRow] 准备更新')
  console.log('[tableUpdateRow]   Original Method:', method)
  console.log('[tableUpdateRow]   URL:', url)
  console.log('[tableUpdateRow]   Body:', data)
  
  // 统一使用 POST 方法
  return post(url, data)
}

// Table 回调操作 - 删除记录
// 统一使用 POST 方法，原函数的 method 通过 _method 查询参数传递，参数放在 body 里
export function tableDeleteRows(method: string, router: string, ids: number[]) {
  const url = `/api/v1/callback${router}?_type=OnTableDeleteRows&_method=${method.toUpperCase()}`
  const data = { ids }
  
  console.log('[tableDeleteRows] 准备删除')
  console.log('[tableDeleteRows]   Original Method:', method)
  console.log('[tableDeleteRows]   URL:', url)
  console.log('[tableDeleteRows]   Body:', data)
  console.log('[tableDeleteRows]   IDs:', ids)
  
  // 统一使用 POST 方法
  return post(url, data)
}

// 导出数据
export function exportData(router: string, params: SearchParams) {
  return post(`/api/v1/export`, { router, ...params })
}

// 导入数据
export function importData(router: string, formData: FormData) {
  return post(`/api/v1/import`, formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })
}