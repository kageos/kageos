import { get } from '@/utils/request'

// Table 操作日志
export interface TableOperateLog {
  id: number
  tenant_user: string
  request_user: string
  action: string
  app: string
  full_code_path: string
  row_id: number
  updates?: any
  old_values?: any
  ip_address?: string
  user_agent?: string
  trace_id?: string
  version?: string
  created_at: string
}

// Form 操作日志
export interface FormOperateLog {
  id: number
  tenant_user: string
  request_user: string
  action: string
  app: string
  full_code_path: string
  function_method?: string
  request_body?: any
  response_body?: any
  code: number
  msg?: string
  ip_address?: string
  user_agent?: string
  trace_id?: string
  version?: string
  created_at: string
}

// 查询 Table 操作日志请求参数
export interface GetTableOperateLogsParams {
  tenant_user?: string
  request_user?: string
  app?: string
  full_code_path?: string
  row_id?: number
  action?: string
  page?: number
  page_size?: number
  order_by?: string
}

// 查询 Form 操作日志请求参数
export interface GetFormOperateLogsParams {
  tenant_user?: string
  request_user?: string
  app?: string
  full_code_path?: string
  action?: string
  page?: number
  page_size?: number
  order_by?: string
}

// 查询 Table 操作日志响应
export interface GetTableOperateLogsResponse {
  logs: TableOperateLog[]
  total: number
  page: number
  page_size: number
}

// 查询 Form 操作日志响应
export interface GetFormOperateLogsResponse {
  logs: FormOperateLog[]
  total: number
  page: number
  page_size: number
}

/**
 * 查询 Table 操作日志
 */
export function getTableOperateLogs(params: GetTableOperateLogsParams): Promise<GetTableOperateLogsResponse> {
  return get<GetTableOperateLogsResponse>('/api/v1/operate_log/table', params)
}

/**
 * 查询 Form 操作日志
 */
export function getFormOperateLogs(params: GetFormOperateLogsParams): Promise<GetFormOperateLogsResponse> {
  return get<GetFormOperateLogsResponse>('/api/v1/operate_log/form', params)
}

