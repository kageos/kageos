import { get, post, put, del } from '@/architecture/infrastructure/apiClient/request'
import type { FunctionDetail, SearchParams } from '@/architecture/domain/types'
import { ensurePublicAnonymousToken, getCurrentPublicShareId, publicShareAnonymousHeaders } from './publicShare'

export interface SelectFuzzyItem {
  value: unknown
  label?: string
  icon?: string
  rich_text?: string
  files?: string
  display_info?: Record<string, unknown>
  displayInfo?: Record<string, unknown>
}

export interface SelectFuzzyResponse {
  error_msg?: string
  items?: SelectFuzzyItem[]
  statistics?: Record<string, unknown>
  max_selections?: number
}

// 获取函数详情（根据路径和函数类型）
// ⭐ 使用新的路由：/function/info/:func-type/*full-code-path
// @param fullCodePath 函数完整路径
// @param funcType 函数类型：table、form、chart（从 node.template_type 获取）
export function getFunctionByPath(fullCodePath: string, funcType: string = 'table') {
  // 确保路径以 / 开头
  const path = fullCodePath.startsWith('/') ? fullCodePath : `/${fullCodePath}`
  // ⭐ 函数类型作为路径参数，这样后端无需查询数据库即可构造权限点
  return get<FunctionDetail>(`/workspace/api/v1/function/info/${funcType}${path}`)
}

// 执行函数（标准接口）
/**
 * 执行函数（根据 template_type 选择标准 API）
 * 
 * @param method 原函数的 HTTP 方法（GET/POST 等）
 * @param router 函数路由（如 /luobei/test999/plugins/cashier_desk），将转换为 full-code-path
 * @param params 请求参数
 * @param templateType 模板类型（table/form/chart），必须传入
 */
export function executeFunction(method: string, router: string, params?: SearchParams | Record<string, unknown>, templateType?: string): Promise<unknown> {
  const fullCodePath = router.startsWith('/') ? router : `/${router}`
  
  // ⭐ 根据 template_type 选择标准 API
  if (templateType === 'table') {
    // Table 查询：使用 /table/search/{full-code-path}
    return get<unknown>(`/workspace/api/v1/table/search${fullCodePath}`, params || {})
  } else if (templateType === 'form') {
    // Form 提交：使用 /form/submit/{full-code-path}
    const submitMethod = method.toUpperCase() || 'POST'
    if (submitMethod === 'GET') {
      return get<unknown>(`/workspace/api/v1/form/submit${fullCodePath}`, params || {})
    } else {
      return post<unknown>(`/workspace/api/v1/form/submit${fullCodePath}`, params || {})
    }
  } else if (templateType === 'chart') {
    // Chart 查询：使用 /chart/query/{full-code-path}
    return get<unknown>(`/workspace/api/v1/chart/query${fullCodePath}`, params || {})
  }

  throw new Error('executeFunction 缺少合法的 templateType，/run 兼容接口已下线')
}

// ⭐ Table 回调操作 - 新增记录（使用标准 API）
export function tableAddRow(method: string, router: string, data: Record<string, unknown>): Promise<unknown> {
  // ⭐ 使用标准 API：/table/create/{full-code-path}
  const fullCodePath = router.startsWith('/') ? router : `/${router}`
  const url = `/workspace/api/v1/table/create${fullCodePath}`
  return post(url, data)
}

// ⭐ Table 回调操作 - 更新记录（使用标准 API）
export function tableUpdateRow(method: string, router: string, data: Record<string, unknown>): Promise<unknown> {
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
 *       value: unknown,               // 选项值
 *       label: string,                // 显示标签
 *       icon: string,                 // 图标（可选）
 *       display_info: Record<string, unknown>  // 额外展示信息
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

/**
 * Select 回调操作 - 模糊查询选项（使用标准 API）
 * 
 * @param method 原函数的 HTTP 方法（GET/POST 等），用于标识回调所属的函数（已废弃，保留用于兼容）
 * @param router 函数路由（如 /luobei/test999/plugins/cashier_desk），将转换为 full-code-path
 * @param data 回调数据
 */
export async function selectFuzzy(method: string, router: string, data: {
  code: string
  type: 'by_keyword' | 'by_value' | 'by_values'
  value: unknown
  request: Record<string, unknown>
  value_type: string
}): Promise<SelectFuzzyResponse> {
  const publicShareId = getCurrentPublicShareId()
  if (publicShareId) {
    await ensurePublicAnonymousToken()
    return post<SelectFuzzyResponse>(
      `/public/api/s/${publicShareId}/callback/on_select_fuzzy`,
      data,
      { headers: publicShareAnonymousHeaders() }
    )
  }

  // ⭐ 使用标准 API：/callback/on_select_fuzzy/{full-code-path}
  // router 格式：/luobei/app/dir/func，需要确保以 / 开头
  const fullCodePath = router.startsWith('/') ? router : `/${router}`
  const url = `/workspace/api/v1/callback/on_select_fuzzy${fullCodePath}`

  // 统一使用 POST 方法
  return post<SelectFuzzyResponse>(url, data)
}
