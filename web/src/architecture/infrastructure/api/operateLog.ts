import { get } from '@/architecture/infrastructure/apiClient/request'

export interface OperateLog {
  id: number
  tenant_user: string
  company_code?: string
  app: string
  actor_user: string
  action: string
  resource_type: string
  resource_path: string
  resource_name?: string
  target_user?: string
  target_id?: string
  summary?: string
  details_json?: any
  old_values_json?: any
  new_values_json?: any
  status?: string
  source?: string
  source_type?: string
  source_ref?: string
  executor_type?: string
  workspace_session_id?: string
  workspace_session_title?: string
  workspace_role?: string
  initiator_user?: string
  workspace_message_id?: number
  tool_call_id?: string
  tool_name?: string
  ip_address?: string
  user_agent?: string
  trace_id?: string
  created_at: string
}

export interface GetOperateLogsParams {
  id?: number
  tenant_user?: string
  company_code?: string
  actor_user?: string
  target_user?: string
  app?: string
  resource_type?: string
  resource_path?: string
  resource_path_prefix?: string
  action?: string
  status?: string
  source?: string
  source_type?: string
  source_ref?: string
  executor_type?: string
  workspace_session_id?: string
  initiator_user?: string
  workspace_message_id?: number
  tool_call_id?: string
  tool_name?: string
  trace_id?: string
  row_id?: number
  keyword?: string
  page?: number
  page_size?: number
  order_by?: string
}

export interface GetOperateLogsResponse {
  logs: OperateLog[]
  total: number
  page: number
  page_size: number
}

export function getOperateLogs(params: GetOperateLogsParams): Promise<GetOperateLogsResponse> {
  return get<GetOperateLogsResponse>('/workspace/api/v1/operate_log/general', params)
}
