import { get, post, put, del } from '@/utils/request'
import type { Function, SearchParams, TableResponse } from '@/types'
import type { FieldConfig } from '@/core/types/field'

// 获取函数详情（根据路径和函数类型）
// ⭐ 使用新的路由：/function/info/:func-type/*full-code-path
// @param fullCodePath 函数完整路径
// @param funcType 函数类型：table、form、chart（从 node.template_type 获取）
export function getFunctionByPath(fullCodePath: string, funcType: string = 'table') {
  // 确保路径以 / 开头
  const path = fullCodePath.startsWith('/') ? fullCodePath : `/${fullCodePath}`
  // ⭐ 函数类型作为路径参数，这样后端无需查询数据库即可构造权限点
  return get<Function>(`/workspace/api/v1/function/info/${funcType}${path}`)
}

// 获取函数详情（根据ID，已废弃，建议使用 getFunctionByPath）
// ⭐ 注意：新路由只支持 full-code-path，如果只有 function_id，需要先查询 full-code-path
export function getFunctionDetail(functionId: number) {
  // ⭐ 临时兼容：使用旧的 API（如果后端还支持）
  // TODO: 建议改为先查询 function_id 对应的 full-code-path，然后调用 getFunctionByPath
  return get<Function>(`/workspace/api/v1/function/get`, { function_id: functionId })
}

// 获取应用下所有函数
export function getFunctionList(appId: number) {
  return get<Function[]>('/workspace/api/v1/function/list', { app_id: appId })
}

// 获取服务目录下函数列表
export function getFunctionByTree(treeId: number) {
  return get<Function[]>(`/workspace/api/v1/function/tree/${treeId}`)
}

// 执行函数（通用）
/**
 * 执行函数（通用接口，根据 template_type 自动选择标准 API）
 * 
 * @param method 原函数的 HTTP 方法（GET/POST 等）
 * @param router 函数路由（如 /luobei/test999/plugins/cashier_desk），将转换为 full-code-path
 * @param params 请求参数
 * @param templateType 模板类型（table/form/chart），用于选择标准 API
 */
export function executeFunction(method: string, router: string, params?: SearchParams | any, templateType?: string) {
  const fullCodePath = router.startsWith('/') ? router : `/${router}`
  
  // ⭐ 根据 template_type 选择标准 API
  if (templateType === 'table') {
    // Table 查询：使用 /table/search/{full-code-path}
    return get<any>(`/workspace/api/v1/table/search${fullCodePath}`, params || {})
  } else if (templateType === 'form') {
    // Form 提交：使用 /form/submit/{full-code-path}
    const submitMethod = method.toUpperCase() || 'POST'
    if (submitMethod === 'GET') {
      return get<any>(`/workspace/api/v1/form/submit${fullCodePath}`, params || {})
    } else {
      return post<any>(`/workspace/api/v1/form/submit${fullCodePath}`, params || {})
    }
  } else if (templateType === 'chart') {
    // Chart 查询：使用 /chart/query/{full-code-path}
    return get<any>(`/workspace/api/v1/chart/query${fullCodePath}`, params || {})
  }
  
  // ⭐ 如果没有指定 template_type，使用旧的 /run 接口（向后兼容）
  const url = `/workspace/api/v1/run${router}`
  switch (method.toUpperCase()) {
    case 'GET':
      return get<any>(url, params || {})
    case 'POST':
      return post<any>(url, params || {})
    case 'PUT':
      return put<any>(url, params || {})
    case 'DELETE':
      return del<any>(url)
    default:
      return get<any>(url, params || {})
  }
}

// ⭐ 旧版本（已注释，保留用于参考）
// export function executeFunction_OLD(method: string, router: string, params?: SearchParams | any) {
//   const url = `/workspace/api/v1/run${router}`
//   switch (method.toUpperCase()) {
//     case 'GET':
//       return get<any>(url, params || {})
//     case 'POST':
//       return post<any>(url, params || {})
//     case 'PUT':
//       return put<any>(url, params || {})
//     case 'DELETE':
//       return del<any>(url)
//     default:
//       return get<any>(url, params || {})
//   }
// }

// 创建函数
export function createFunction(data: Partial<Function>) {
  return post<Function>('/workspace/api/v1/function/create', data)
}

// 更新函数
export function updateFunction(id: number, data: Partial<Function>) {
  return put(`/workspace/api/v1/function/${id}`, data)
}

// 删除函数
export function deleteFunction(id: number) {
  return del(`/workspace/api/v1/function/${id}`)
}

// ⭐ Table 回调操作 - 新增记录（使用标准 API）
export function tableAddRow(method: string, router: string, data: any) {
  // ⭐ 使用标准 API：/table/create/{full-code-path}
  const fullCodePath = router.startsWith('/') ? router : `/${router}`
  const url = `/workspace/api/v1/table/create${fullCodePath}`
  return post(url, data)
}

// ⭐ Table 回调操作 - 更新记录（使用标准 API）
export function tableUpdateRow(method: string, router: string, data: any) {
  // ⭐ 使用标准 API：/table/update/{full-code-path}
  const fullCodePath = router.startsWith('/') ? router : `/${router}`
  const url = `/workspace/api/v1/table/update${fullCodePath}`
  return put(url, data)
}

// ⭐ Table 回调操作 - 删除记录（使用标准 API）
export function tableDeleteRows(method: string, router: string, ids: number[]) {
  // ⭐ 使用标准 API：/table/delete/{full-code-path}
  const fullCodePath = router.startsWith('/') ? router : `/${router}`
  const url = `/workspace/api/v1/table/delete${fullCodePath}`
  const data = { ids }
  return del(url, data)  // DELETE 请求带 body
}

/**
 * Select 回调操作 - 模糊查询选项
 *
 * @param method 原函数的 HTTP 方法（GET/POST 等），用于标识回调所属的函数
 * @param router 函数路由（如 /luobei/test999/plugins/cashier_desk）
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

/**
 * Select 回调操作 - 模糊查询选项（使用标准 API）
 * 
 * @param method 原函数的 HTTP 方法（GET/POST 等），用于标识回调所属的函数（已废弃，保留用于兼容）
 * @param router 函数路由（如 /luobei/test999/plugins/cashier_desk），将转换为 full-code-path
 * @param data 回调数据
 */
export function selectFuzzy(method: string, router: string, data: {
  code: string
  type: 'by_keyword' | 'by_value' | 'by_values'
  value: any
  request: Record<string, any>
  value_type: string
}) {
  // ⭐ 使用标准 API：/callback/on_select_fuzzy/{full-code-path}
  // router 格式：/luobei/app/dir/func，需要确保以 / 开头
  const fullCodePath = router.startsWith('/') ? router : `/${router}`
  const url = `/workspace/api/v1/callback/on_select_fuzzy${fullCodePath}`

  // 统一使用 POST 方法
  return post(url, data)
}

// ⭐ 旧版本（已注释，保留用于参考）
// export function selectFuzzy_OLD(method: string, router: string, data: {
//   code: string
//   type: 'by_keyword' | 'by_value' | 'by_values'
//   value: any
//   request: Record<string, any>
//   value_type: string
// }) {
//   const url = `/workspace/api/v1/callback${router}?_type=OnSelectFuzzy&_function_method=${method.toUpperCase()}`
//   return post(url, data)
// }

// 导出数据
export function exportData(router: string, params: SearchParams) {
  return post(`/workspace/api/v1/export`, { router, ...params })
}

// 导入数据
export function importData(router: string, formData: FormData) {
  return post(`/workspace/api/v1/import`, formData, {
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
  return post<{ message: string }>('/workspace/api/v1/function/fork', data)
}

/**
 * 获取函数组信息（用于 Hub 发布）
 * @param fullGroupCode 完整函数组代码，例如：/luobei/testgroup/plugins/tools_cashier
 */
export interface FunctionInfo {
  id: number
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
  name: string        // 函数名称
  description: string // 函数描述
}

export interface GetFunctionGroupInfoResp {
  source_code: string        // 源代码
  description: string        // 描述信息（通常为空，由用户在 Hub 填写）
  full_group_code: string    // 完整函数组代码
  group_code: string         // 函数组代码
  group_name: string         // 函数组名称
  full_path: string          // 完整路径
  version: string            // 版本号
  app_id: number            // 应用ID
  app_name: string           // 应用名称
  function_count: number    // 函数数量
  functions: FunctionInfo[]  // 函数列表（用于 Hub 展示功能列表）
}

export function getFunctionGroupInfo(fullGroupCode: string) {
  return get<GetFunctionGroupInfoResp>('/workspace/api/v1/function/group-info', { full_group_code: fullGroupCode })
}
