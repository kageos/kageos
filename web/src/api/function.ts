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
  
  // 统一使用 POST 方法
  return post(url, data)
}

// Table 回调操作 - 更新记录
// 统一使用 POST 方法，原函数的 method 通过 _method 查询参数传递，参数放在 body 里
export function tableUpdateRow(method: string, router: string, data: any) {
  const url = `/api/v1/callback${router}?_type=OnTableUpdateRow&_method=${method.toUpperCase()}`
  
  // 统一使用 POST 方法
  return post(url, data)
}

// Table 回调操作 - 删除记录
// 统一使用 POST 方法，原函数的 method 通过 _method 查询参数传递，参数放在 body 里
export function tableDeleteRows(method: string, router: string, ids: number[]) {
  const url = `/api/v1/callback${router}?_type=OnTableDeleteRows&_method=${method.toUpperCase()}`
  const data = { ids }
  
  // 统一使用 POST 方法
  return post(url, data)
}

/**
 * Select 回调操作 - 模糊查询选项
 * 
 * @param method 原函数的 HTTP 方法（GET/POST 等），用于标识回调所属的函数
 * @param router 函数路由（如 /luobei/test999/tools/cashier_desk）
 * @param data 回调数据
 * @param data.code 字段代码（如 product_id）
 * @param data.type 查询类型：'by_keyword' | 'by_value' | 'by_values'
 *   - by_keyword: 根据用户输入的关键字模糊搜索（默认）
 *   - by_value: 根据字段的实际值查询（用于回显、URL 恢复等场景，单个值）
 *   - by_values: 根据字段的实际值查询（用于多选回显，数组值）
 * @param data.value 查询值（关键字或实际值）
 * @param data.request 当前表单的所有字段值
 * @param data.value_type 字段类型（int/string/float 等）
 * 
 * @returns Promise<{
 *   data: {
 *     error_msg: string,              // 错误信息（空表示成功）
 *     items: Array<{                  // 选项列表
 *       value: any,                   // 选项值
 *       label: string,                // 显示标签
 *       icon: string,                 // 图标（可选）
 *       display_info: Record<string, any>  // 额外展示信息
 *     }>,
 *     statistics: Record<string, string>  // 聚合统计表达式
 *   }
 * }>
 * 
 * @example
 * // 用户输入搜索（by_keyword）
 * // 注意：method 参数是原函数的 HTTP 方法，不是回调请求的 HTTP 方法
 * selectFuzzy('GET', '/luobei/demo/crm/meeting_room_booking_list', {
 *   code: 'room_id',
 *   type: 'by_keyword',
 *   value: '会议室',
 *   request: {},
 *   value_type: 'int'
 * })
 * 
 * @example
 * // 根据值查询（by_value）- 用于编辑回显
 * selectFuzzy('GET', '/luobei/demo/crm/meeting_room_booking_list', {
 *   code: 'room_id',
 *   type: 'by_value',
 *   value: 1,
 *   request: {},
 *   value_type: 'int'
 * })
 */
import { SelectFuzzyQueryType } from '@/core/constants/select'
import { Logger } from '@/core/utils/logger'

export function selectFuzzy(method: string, router: string, data: {
  code: string
  type: 'by_keyword' | 'by_value' | 'by_values'
  value: any
  request: Record<string, any>
  value_type: string
}) {
  const url = `/api/v1/callback${router}?_type=OnSelectFuzzy&_function_method=${method.toUpperCase()}`
  
  Logger.debug('selectFuzzy', 'Select 回调查询', {
    originalMethod: method,
    url,
    queryType: data.type,
    fieldCode: data.code,
    searchValue: data.value,
    requestBody: data
  })
  
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

// Fork 函数组（支持批量）
export function forkFunctionGroup(data: {
  source_to_target_map: Record<string, string>
  target_app_id: number
}) {
  return post<{ message: string }>('/api/v1/function/fork', data)
}