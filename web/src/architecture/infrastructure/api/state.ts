import { get } from '@/utils/request'

export type RuntimeStateKind = 'workspace_session' | 'scheduled_agent_session'
export type RuntimeStateStatus = 'thinking' | 'running' | 'tool_running' | 'waiting_approval' | 'failed' | 'cancelled'

export interface RuntimeStateItem {
  key: string
  kind: RuntimeStateKind | string
  status: RuntimeStateStatus | string
  stage?: string
  full_code_path: string
  title?: string
  user?: string
  mode_code?: string
  session_id?: string
  source_type?: string
  source_ref?: string
  started_at: string
  updated_at: string
  expires_at?: string
  metadata?: Record<string, unknown>
}

export interface RuntimeStateSummary {
  running_count: number
  manual_running_count: number
  scheduled_running_count: number
  thinking_count: number
  tool_running_count: number
  waiting_approval_count: number
  failed_recent_count: number
  last_activity_at: string
  dominant_status?: RuntimeStateStatus | string
  badge_text?: string
  badge_tone?: string
  tooltip?: string
}

export interface RuntimeStateSummaryResp {
  summaries: Record<string, RuntimeStateSummary>
}

export interface RuntimeStateItemsResp {
  items: RuntimeStateItem[]
}

export interface RuntimeStateQuery {
  root_full_code_path?: string
  kind?: RuntimeStateKind | string
  status?: RuntimeStateStatus | string
}

const BASE_URL = '/agent/api/v1/state'

export function getRuntimeStateSummary(params?: RuntimeStateQuery): Promise<RuntimeStateSummaryResp> {
  return get<RuntimeStateSummaryResp>(`${BASE_URL}/runtime-summary`, params || {})
}

export function getRuntimeStateItems(params?: RuntimeStateQuery): Promise<RuntimeStateItemsResp> {
  return get<RuntimeStateItemsResp>(`${BASE_URL}/runtime-items`, params || {})
}
