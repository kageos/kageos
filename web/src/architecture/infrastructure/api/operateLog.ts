import { get } from '@/architecture/infrastructure/apiClient/request'

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

export interface GetTableOperateLogsParams {
  tenant_user?: string
  request_user?: string
  app?: string
  full_code_path?: string
  full_code_path_prefix?: string
  row_id?: number
  action?: string
  page?: number
  page_size?: number
  order_by?: string
}

export interface GetTableOperateLogsResponse {
  logs: TableOperateLog[]
  total: number
  page: number
  page_size: number
}

export function getTableOperateLogs(params: GetTableOperateLogsParams): Promise<GetTableOperateLogsResponse> {
  return get<GetTableOperateLogsResponse>('/workspace/api/v1/operate_log/table', params)
}
