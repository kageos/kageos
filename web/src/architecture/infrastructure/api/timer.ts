import { authFetch } from '@/architecture/infrastructure/apiClient/request'
import { getApiBaseURL } from '@/architecture/infrastructure/config/runtime'

export type TimerScheduleType = 'atime' | 'cron' | 'every'
export type TimerTaskStatus = 'pending' | 'paused' | 'done' | 'failed' | 'cancelled'
export type TimerExecutionStatus = 'waiting' | 'queued' | 'running' | 'success' | 'failed' | 'timeout' | 'cancelled' | 'skipped'
export type TimerOverlapPolicy = 'forbid' | 'queue_latest' | 'allow'

export interface TimerSchedule {
  type: TimerScheduleType
  run_at?: string
  cron_expr?: string
  interval_seconds?: number
  timezone?: string
  max_runs?: number
}

export interface TimerTask {
  id: number
  title?: string
  description?: string
  category?: string
  tags?: string[]
  idempotency_key?: string
  executor_key: string
  executor_payload?: unknown
  metadata?: Record<string, string>
  status: TimerTaskStatus
  schedule: TimerSchedule
  next_run_at?: string
  run_count: number
  overlap_policy?: TimerOverlapPolicy
  max_parallelism?: number
  inflight_execution_id?: number
  last_execution_id?: number
  last_error_message?: string
  source_type?: string
  source_ref?: string
  resource_scope?: string
  resource_key?: string
  request_user?: string
  request_user_dept?: string
  created_by?: string
  created_at?: string
  updated_at?: string
}

export interface TimerExecution {
  id: number
  task_id: number
  executor_key: string
  status: TimerExecutionStatus
  trigger_type?: string
  executor_run_id?: string
  scheduled_at: string
  started_at?: string
  finished_at?: string
  worker_id?: string
  lease_until?: string
  heartbeat_at?: string
  attempt?: number
  duration_millis?: number
  output_summary?: string
  result_payload?: unknown
  error_message?: string
  trace_id?: string
  source_type?: string
  source_ref?: string
  resource_scope?: string
  resource_key?: string
  request_user?: string
  request_user_dept?: string
  last_dispatched_at?: string
  created_at?: string
  updated_at?: string
}

export interface CreateTimerTaskRequest {
  title?: string
  description?: string
  category?: string
  tags?: string[]
  idempotency_key?: string
  executor_key: string
  executor_payload?: unknown
  metadata?: Record<string, string>
  status?: TimerTaskStatus
  schedule: TimerSchedule
  overlap_policy?: TimerOverlapPolicy
  max_parallelism?: number
  source_type?: string
  source_ref?: string
  resource_scope?: string
  resource_key?: string
  request_user?: string
  request_user_dept?: string
  created_by?: string
}

export interface UpdateTimerTaskRequest {
  title?: string
  description?: string
  category?: string
  tags?: string[]
  executor_payload?: unknown
  metadata?: Record<string, string>
  schedule?: TimerSchedule
  overlap_policy?: TimerOverlapPolicy
  max_parallelism?: number
  source_type?: string
  source_ref?: string
  resource_scope?: string
  resource_key?: string
  request_user?: string
  request_user_dept?: string
}

export interface ListTimerTasksParams {
  executor_key?: string
  status?: string
  category?: string
  source_type?: string
  source_ref?: string
  resource_scope?: string
  resource_key?: string
  created_by?: string
  page?: number
  page_size?: number
}

export interface ListTimerTasksResponse {
  list: TimerTask[]
  total: number
}

export interface ListTimerExecutionsParams {
  status?: string
  page?: number
  page_size?: number
}

export interface ListTimerExecutionsResponse {
  list: TimerExecution[]
  total: number
}

function buildTimerURL(path: string, params?: Record<string, unknown>): string {
  const base = getApiBaseURL()
  const url = new URL(`${base}${path}`, window.location.origin)

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return
    url.searchParams.set(key, String(value))
  })

  return url.toString()
}

function extractTimerError(payload: unknown, fallback: string): string {
  if (payload && typeof payload === 'object') {
    const record = payload as Record<string, unknown>
    const error = record.error || record.msg || record.message
    if (typeof error === 'string' && error.trim()) {
      return error
    }
  }
  return fallback
}

async function parseTimerResponse<T>(response: Response): Promise<T> {
  const text = await response.text()
  const payload = text ? JSON.parse(text) : null

  if (!response.ok) {
    throw new Error(extractTimerError(payload, `HTTP ${response.status}`))
  }

  if (payload && typeof payload === 'object' && 'code' in payload) {
    const wrapped = payload as { code?: number; data?: T }
    if (wrapped.code === 0) {
      return wrapped.data as T
    }
    throw new Error(extractTimerError(payload, '请求失败'))
  }

  return payload as T
}

async function timerRequest<T>(path: string, init: RequestInit = {}, params?: Record<string, unknown>): Promise<T> {
  const response = await authFetch(buildTimerURL(path, params), {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init.headers || {}),
    },
  })
  return parseTimerResponse<T>(response)
}

export function createTimerTask(data: CreateTimerTaskRequest): Promise<TimerTask> {
  return timerRequest<TimerTask>('/timer/api/v1/tasks', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function updateTimerTask(id: number, data: UpdateTimerTaskRequest): Promise<TimerTask> {
  return timerRequest<TimerTask>(`/timer/api/v1/tasks/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function getTimerTask(id: number): Promise<TimerTask> {
  return timerRequest<TimerTask>(`/timer/api/v1/tasks/${id}`)
}

export function listTimerTasks(params: ListTimerTasksParams = {}): Promise<ListTimerTasksResponse> {
  return timerRequest<ListTimerTasksResponse>('/timer/api/v1/tasks', undefined, params as Record<string, unknown>)
}

export function pauseTimerTask(id: number): Promise<void> {
  return timerRequest<void>(`/timer/api/v1/tasks/${id}/pause`, { method: 'POST' })
}

export function resumeTimerTask(id: number): Promise<void> {
  return timerRequest<void>(`/timer/api/v1/tasks/${id}/resume`, { method: 'POST' })
}

export function cancelTimerTask(id: number): Promise<void> {
  return timerRequest<void>(`/timer/api/v1/tasks/${id}/cancel`, { method: 'POST' })
}

export function deleteTimerTask(id: number): Promise<void> {
  return timerRequest<void>(`/timer/api/v1/tasks/${id}`, { method: 'DELETE' })
}

export function runTimerTaskNow(id: number): Promise<TimerExecution> {
  return timerRequest<TimerExecution>(`/timer/api/v1/tasks/${id}/run_now`, { method: 'POST' })
}

export function listTimerExecutions(
  taskId: number,
  params: ListTimerExecutionsParams = {}
): Promise<ListTimerExecutionsResponse> {
  return timerRequest<ListTimerExecutionsResponse>(
    `/timer/api/v1/tasks/${taskId}/executions`,
    undefined,
    params as Record<string, unknown>
  )
}

export function getTimerExecution(taskId: number, executionId: number): Promise<TimerExecution> {
  return timerRequest<TimerExecution>(`/timer/api/v1/tasks/${taskId}/executions/${executionId}`)
}
