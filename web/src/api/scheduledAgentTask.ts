import { del, get, post, put } from '@/utils/request'

export type ScheduledAgentScheduleType = 'atime' | 'cron' | 'every'
export type ScheduledAgentTaskStatus = 'pending' | 'paused' | 'done' | 'failed' | 'cancelled'
export type ScheduledAgentExecutionStatus = 'pending' | 'running' | 'success' | 'failed' | 'timeout' | 'cancelled'
export type ScheduledAgentNotifyOn = 'none' | 'all' | 'success' | 'failed'

export interface CreateScheduledAgentTaskReq {
  name: string
  full_code_path: string
  goal: string
  mode_code?: string
  files?: string
  llm_config_id?: number
  context_policy?: Record<string, unknown>
  tool_policy?: Record<string, unknown>
  approval_policy?: Record<string, unknown>
  budget_policy?: Record<string, unknown>
  schedule_type: ScheduledAgentScheduleType
  run_at?: string
  cron_expr?: string
  interval_seconds?: number
  max_runs?: number
  timezone?: string
  request_user?: string
  request_user_dept?: string
  notify_users?: string[]
  notify_departments?: string[]
  notify_on?: ScheduledAgentNotifyOn
}

export interface UpdateScheduledAgentTaskReq extends Partial<CreateScheduledAgentTaskReq> {}

export interface ScheduledAgentTaskItem {
  id: number
  name: string
  full_code_path: string
  goal: string
  mode_code: string
  files?: string
  llm_config_id: number
  context_policy?: Record<string, unknown>
  tool_policy?: Record<string, unknown>
  approval_policy?: Record<string, unknown>
  budget_policy?: Record<string, unknown>
  schedule_type: ScheduledAgentScheduleType
  run_at: string
  next_run_at?: string
  cron_expr?: string
  interval_seconds?: number
  max_runs?: number
  timezone?: string
  status: ScheduledAgentTaskStatus
  timer_task_id?: number
  run_count: number
  last_session_id?: string
  last_execution_id?: number
  last_error_message?: string
  request_user?: string
  request_user_dept?: string
  notify_users?: string[]
  notify_departments?: string[]
  notify_on?: ScheduledAgentNotifyOn
  source_type?: string
  source_ref?: string
  created_by: string
  created_at: string
  updated_at: string
}

export interface ScheduledAgentExecutionItem {
  id: number
  task_id: number
  session_id?: string
  scheduled_at: string
  started_at?: string
  finished_at?: string
  status: ScheduledAgentExecutionStatus
  worker_id?: string
  duration_millis: number
  input_goal?: string
  output_summary?: string
  tool_call_count: number
  token_usage?: Record<string, unknown>
  error_message?: string
  trace_id?: string
  source_type?: string
  source_ref?: string
  created_at: string
  updated_at: string
}

export interface ScheduledAgentRunNowResp {
  task_id: number
  timer_task_id: number
  timer_execution_id: number
  status: string
  scheduled_at: string
}

export interface ListScheduledAgentTasksResp {
  list: ScheduledAgentTaskItem[]
  total: number
}

export interface ListScheduledAgentExecutionsResp {
  list: ScheduledAgentExecutionItem[]
  total: number
}

const BASE_URL = '/agent/api/v1/scheduled_agent_tasks'

export function createScheduledAgentTask(data: CreateScheduledAgentTaskReq): Promise<ScheduledAgentTaskItem> {
  return post<ScheduledAgentTaskItem>(BASE_URL, data)
}

export function updateScheduledAgentTask(id: number, data: UpdateScheduledAgentTaskReq): Promise<ScheduledAgentTaskItem> {
  return put<ScheduledAgentTaskItem>(`${BASE_URL}/${id}`, data)
}

export function getScheduledAgentTask(id: number): Promise<ScheduledAgentTaskItem> {
  return get<ScheduledAgentTaskItem>(`${BASE_URL}/${id}`)
}

export function listScheduledAgentTasks(params?: {
  status?: string
  full_code_path?: string
  page?: number
  page_size?: number
}): Promise<ListScheduledAgentTasksResp> {
  return get<ListScheduledAgentTasksResp>(BASE_URL, params || {})
}

export function deleteScheduledAgentTask(id: number): Promise<void> {
  return del<void>(`${BASE_URL}/${id}`)
}

export const cancelScheduledSessionTask = deleteScheduledAgentTask
export const cancelScheduledAgentTask = deleteScheduledAgentTask

export function pauseScheduledAgentTask(id: number): Promise<void> {
  return post<void>(`${BASE_URL}/${id}/pause`)
}

export function resumeScheduledAgentTask(id: number): Promise<void> {
  return post<void>(`${BASE_URL}/${id}/resume`)
}

export function runScheduledAgentTaskNow(id: number): Promise<ScheduledAgentRunNowResp> {
  return post<ScheduledAgentRunNowResp>(`${BASE_URL}/${id}/run`)
}

export function listScheduledAgentExecutions(
  taskId: number,
  params?: { status?: string; page?: number; page_size?: number }
): Promise<ListScheduledAgentExecutionsResp> {
  return get<ListScheduledAgentExecutionsResp>(`${BASE_URL}/${taskId}/executions`, params || {})
}

export function getScheduledAgentExecution(taskId: number, executionId: number): Promise<ScheduledAgentExecutionItem> {
  return get<ScheduledAgentExecutionItem>(`${BASE_URL}/${taskId}/executions/${executionId}`)
}
