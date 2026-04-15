/**
 * 定时任务 API
 */

import { get, post, del } from '@/utils/request'

/** 与后端一致：execute=普通函数；table_*=表格增改删（走 /_callback） */
export type ScheduledTaskAction =
  | 'execute'
  | 'table_create'
  | 'table_update'
  | 'table_delete'

export interface CreateScheduledTaskReq {
  name: string
  full_code_path: string
  action?: ScheduledTaskAction
  method?: string
  payload?: Record<string, unknown> | string
  request_user?: string
  request_user_dept?: string
  schedule_type: 'atime' | 'cron' | 'every'
  run_at?: string
  cron_expr?: string
  interval_seconds?: number
  max_runs?: number
  timezone?: string
}

export interface ScheduledTaskItem {
  id: number
  name: string
  user: string
  app: string
  full_code_path: string
  action?: string
  method?: string
  payload?: string
  request_user?: string
  request_user_dept?: string
  created_by: string
  schedule_type: string
  run_at: string
  next_run_at?: string
  cron_expr?: string
  interval_seconds?: number
  max_runs?: number
  timezone?: string
  status: string
  run_count: number
  error_message?: string
  created_at: string
}

export interface ScheduledTaskExecutionItem {
  id: number
  task_id: number
  executed_at: string
  status: string
  duration_millis?: number
  request_payload: string
  response_payload: string
  error_message?: string
  trace_id?: string
  created_at: string
}

export interface ListScheduledTasksResp {
  list: ScheduledTaskItem[]
  total: number
}

export interface ListScheduledTaskExecutionsResp {
  list: ScheduledTaskExecutionItem[]
  total: number
}

export function createScheduledTask(data: CreateScheduledTaskReq): Promise<ScheduledTaskItem> {
  // 直接传对象，后端存成 json.RawMessage 即 {"a":1} 的字节；不要 JSON.stringify(payload) 否则会变成字符串导致执行时 unmarshal 报错
  const payload = data.payload ?? {}
  return post<ScheduledTaskItem>('/workspace/api/v1/scheduled_tasks', {
    name: data.name,
    full_code_path: data.full_code_path,
    action: data.action ?? 'execute',
    method: data.method || 'POST',
    payload,
    request_user: data.request_user,
    request_user_dept: data.request_user_dept,
    schedule_type: data.schedule_type,
    run_at: data.run_at,
    cron_expr: data.cron_expr,
    interval_seconds: data.interval_seconds,
    max_runs: data.max_runs ?? 0,
    timezone: data.timezone
  })
}

/** full_code_path 为前缀：返回该路径及子路径下的任务（目录节点可看到子表单的定时任务） */
export function listScheduledTasks(params: {
  full_code_path?: string
  status?: string
  page?: number
  page_size?: number
}): Promise<ListScheduledTasksResp> {
  return get<ListScheduledTasksResp>('/workspace/api/v1/scheduled_tasks', params)
}

export function cancelScheduledTask(id: number): Promise<void> {
  return del<void>(`/workspace/api/v1/scheduled_tasks/${id}`)
}

export function listScheduledTaskExecutions(
  taskId: number,
  params?: { status?: string; page?: number; page_size?: number }
): Promise<ListScheduledTaskExecutionsResp> {
  return get<ListScheduledTaskExecutionsResp>(
    `/workspace/api/v1/scheduled_tasks/${taskId}/executions`,
    params
  )
}
