import { get, post, put } from '@/utils/request'

const WORKFLOW_API = '/workflow/api/v1'

export type WorkflowStatus = 'draft' | 'enabled' | 'disabled'
export type WorkflowRunStatus = 'running' | 'success' | 'failed' | 'canceled'

export interface WorkflowItem {
  id: number
  created_at: string
  updated_at: string
  name: string
  description: string
  app_id: number
  full_code_path: string
  status: WorkflowStatus
  latest_version_id: number
  created_by: string
  updated_by: string
  draft_definition_json?: Record<string, unknown>
}

export interface WorkflowVersionItem {
  id: number
  created_at: string
  workflow_id: number
  version: number
  definition_json: Record<string, unknown>
  input_schema_json?: Record<string, unknown>
  output_schema_json?: Record<string, unknown>
  status: string
  created_by: string
}

export interface WorkflowRunItem {
  id: number
  created_at: string
  updated_at: string
  workflow_id: number
  version_id: number
  status: WorkflowRunStatus
  input_json?: Record<string, unknown>
  output_json?: Record<string, unknown>
  error_message?: string
  request_user: string
  request_user_dept: string
  trace_id: string
  started_at?: string
  finished_at?: string
  duration_millis: number
}

export interface WorkflowStepRunItem {
  id: number
  created_at: string
  updated_at: string
  run_id: number
  step_id: string
  step_name: string
  node_type: string
  node_ref: string
  status: WorkflowRunStatus
  input_json?: Record<string, unknown>
  output_json?: Record<string, unknown>
  error_message?: string
  trace_id: string
  attempt: number
  started_at?: string
  finished_at?: string
  duration_millis: number
}

export interface WorkflowRunDetail {
  run: WorkflowRunItem
  steps: WorkflowStepRunItem[]
}

export function getWorkflowByPath(fullCodePath: string) {
  return get<WorkflowItem | null>(`${WORKFLOW_API}/workflows/by_path`, {
    full_code_path: fullCodePath
  })
}

export function createWorkflow(data: {
  name: string
  description?: string
  app_id?: number
  full_code_path?: string
  definition?: Record<string, unknown>
}) {
  return post<WorkflowItem>(`${WORKFLOW_API}/workflows`, data)
}

export function updateWorkflow(id: number, data: {
  name?: string
  description?: string
  app_id?: number
  full_code_path?: string
  status?: WorkflowStatus
  definition?: Record<string, unknown>
}) {
  return put<WorkflowItem>(`${WORKFLOW_API}/workflows/${id}`, data)
}

export function publishWorkflow(id: number, data: { definition?: Record<string, unknown> } = {}) {
  return post<WorkflowVersionItem>(`${WORKFLOW_API}/workflows/${id}/publish`, data)
}

export function runWorkflow(id: number, data: { version_id?: number; input?: Record<string, unknown> }) {
  return post<WorkflowRunDetail>(`${WORKFLOW_API}/workflows/${id}/run`, data)
}

export function getWorkflowRun(runID: number) {
  return get<WorkflowRunDetail>(`${WORKFLOW_API}/runs/${runID}`)
}
