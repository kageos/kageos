import type { LocationQuery, LocationQueryRaw } from 'vue-router'

export const PLATFORM_OPEN_QUERY_KEY = '_open'
export const PLATFORM_PANEL_QUERY_KEY = '_panel'
export const PLATFORM_FOCUS_QUERY_KEY = '_focus'
export const PLATFORM_MESSAGE_ID_QUERY_KEY = '_message_id'
export const PLATFORM_LOG_ID_QUERY_KEY = '_log_id'
export const PLATFORM_SESSION_ID_QUERY_KEY = '_session_id'
export const PLATFORM_SOURCE_PATH_QUERY_KEY = '_source_path'
export const PLATFORM_TRACE_ID_QUERY_KEY = '_trace_id'
export const PLATFORM_SCHEDULED_TASK_ID_QUERY_KEY = '_scheduled_task_id'
export const PLATFORM_SCHEDULED_EXECUTION_ID_QUERY_KEY = '_scheduled_execution_id'
export const PLATFORM_SCHEDULED_KIND_QUERY_KEY = '_scheduled_kind'

export const LEGACY_SCHEDULED_OPEN_QUERY_KEY = '_scheduled'
export const LEGACY_MINI_WORKSTATION_OPEN_QUERY_KEY = '_mws'
export const LEGACY_MINI_WORKSTATION_SESSION_ID_QUERY_KEY = '_mws_sid'
export const LEGACY_MINI_WORKSTATION_PATH_QUERY_KEY = '_mws_path'
export const LEGACY_MINI_WORKSTATION_NAME_QUERY_KEY = '_mws_name'
export const LEGACY_MINI_WORKSTATION_EXPANDED_QUERY_KEY = '_mws_expanded'
export const LEGACY_MINI_WORKSTATION_MAXIMIZED_QUERY_KEY = '_mws_maximized'

export type PlatformOpenTarget = 'inbox' | 'operate_log' | 'scheduled' | 'session'
export type PlatformFocusTarget = 'message' | 'operate_log' | 'scheduled_task' | 'scheduled_execution' | 'workspace_session'
export type PlatformScheduledKind = 'function' | 'agent'
export type PlatformPanel = 'operateLog' | 'scheduledTask' | 'scheduledAgentTask'

export interface WorkspaceRouteRequest {
  fullCodePath: string
  query?: LocationQueryRaw
}

export interface MessageSourceRouteRequest {
  fullCodePath: string
  messageId?: number | string
  sourcePath?: string
  traceId?: string
  openInbox?: boolean
}

export interface WorkspaceSessionRouteRequest {
  fullCodePath: string
  sessionId: string
  messageId?: number | string
  sourceName?: string
  sourcePath?: string
  traceId?: string
  expanded?: boolean
  maximized?: boolean
}

export interface ScheduledExecutionRouteRequest {
  fullCodePath: string
  kind: PlatformScheduledKind
  taskId: number | string
  executionId?: number | string
  sourcePath?: string
  traceId?: string
}

export interface OperateLogRouteRequest {
  fullCodePath: string
  logId?: number | string
  traceId?: string
  sourcePath?: string
}

export interface InboxRouteQueryRequest {
  messageId?: number | string
  sourcePath?: string
  traceId?: string
}

export function readStringQuery(query: LocationQuery | LocationQueryRaw, key: string): string {
  const raw = query[key]
  const value = Array.isArray(raw) ? raw[0] : raw
  return value === undefined || value === null ? '' : String(value).trim()
}

export function readNumberQuery(query: LocationQuery | LocationQueryRaw, key: string): number {
  const value = readStringQuery(query, key)
  if (!value) return 0
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0
}

export function normalizeWorkspaceFullCodePath(fullCodePath?: string): string {
  return (fullCodePath || '').trim().replace(/\/+$/g, '')
}

export function workspaceRoutePath(fullCodePath?: string): string {
  const normalized = normalizeWorkspaceFullCodePath(fullCodePath)
  if (!normalized) return ''
  return `/workspace${normalized.startsWith('/') ? normalized : `/${normalized}`}`
}

export function buildWorkspaceRoute(request: WorkspaceRouteRequest) {
  const path = workspaceRoutePath(request.fullCodePath)
  return {
    path,
    query: request.query || {},
  }
}

export function buildMessageSourceRoute(request: MessageSourceRouteRequest) {
  const sourcePath = normalizeWorkspaceFullCodePath(request.sourcePath || request.fullCodePath)
  const query: LocationQueryRaw = {
    [PLATFORM_FOCUS_QUERY_KEY]: 'message',
    ...(request.openInbox ? { [PLATFORM_OPEN_QUERY_KEY]: 'inbox' } : {}),
    ...(request.messageId ? { [PLATFORM_MESSAGE_ID_QUERY_KEY]: String(request.messageId) } : {}),
    ...(sourcePath ? { [PLATFORM_SOURCE_PATH_QUERY_KEY]: sourcePath } : {}),
    ...(request.traceId ? { [PLATFORM_TRACE_ID_QUERY_KEY]: request.traceId } : {}),
  }
  return buildWorkspaceRoute({
    fullCodePath: request.fullCodePath,
    query,
  })
}

export function buildInboxRouteQuery(request: InboxRouteQueryRequest = {}): LocationQueryRaw {
  const messageId = request.messageId ? String(request.messageId) : ''
  const sourcePath = normalizeWorkspaceFullCodePath(request.sourcePath)
  return {
    [PLATFORM_OPEN_QUERY_KEY]: 'inbox',
    ...(messageId ? { [PLATFORM_FOCUS_QUERY_KEY]: 'message' } : {}),
    ...(messageId ? { [PLATFORM_MESSAGE_ID_QUERY_KEY]: messageId } : {}),
    ...(sourcePath ? { [PLATFORM_SOURCE_PATH_QUERY_KEY]: sourcePath } : {}),
    ...(request.traceId ? { [PLATFORM_TRACE_ID_QUERY_KEY]: request.traceId } : {}),
  }
}

export function buildWorkspaceSessionRoute(request: WorkspaceSessionRouteRequest) {
  const fullCodePath = normalizeWorkspaceFullCodePath(request.fullCodePath)
  const sourcePath = normalizeWorkspaceFullCodePath(request.sourcePath || fullCodePath)
  const expanded = request.expanded !== false
  const maximized = request.maximized !== false
  return buildWorkspaceRoute({
    fullCodePath,
    query: {
      [PLATFORM_OPEN_QUERY_KEY]: 'session',
      [PLATFORM_FOCUS_QUERY_KEY]: 'workspace_session',
      [PLATFORM_SESSION_ID_QUERY_KEY]: request.sessionId,
      ...(request.messageId ? { [PLATFORM_MESSAGE_ID_QUERY_KEY]: String(request.messageId) } : {}),
      ...(sourcePath ? { [PLATFORM_SOURCE_PATH_QUERY_KEY]: sourcePath } : {}),
      ...(request.traceId ? { [PLATFORM_TRACE_ID_QUERY_KEY]: request.traceId } : {}),

      // Compatibility for the existing mini workstation router.
      [LEGACY_MINI_WORKSTATION_OPEN_QUERY_KEY]: 'open',
      [LEGACY_MINI_WORKSTATION_SESSION_ID_QUERY_KEY]: request.sessionId,
      [LEGACY_MINI_WORKSTATION_PATH_QUERY_KEY]: fullCodePath,
      ...(request.sourceName ? { [LEGACY_MINI_WORKSTATION_NAME_QUERY_KEY]: request.sourceName } : {}),
      [LEGACY_MINI_WORKSTATION_EXPANDED_QUERY_KEY]: expanded ? '1' : '0',
      [LEGACY_MINI_WORKSTATION_MAXIMIZED_QUERY_KEY]: maximized ? '1' : '0',
    },
  })
}

export function buildScheduledExecutionRoute(request: ScheduledExecutionRouteRequest) {
  const executionId = request.executionId ? String(request.executionId) : ''
  const panel: PlatformPanel = request.kind === 'agent' ? 'scheduledAgentTask' : 'scheduledTask'
  return buildWorkspaceRoute({
    fullCodePath: request.fullCodePath,
    query: {
      [PLATFORM_OPEN_QUERY_KEY]: 'scheduled',
      [PLATFORM_PANEL_QUERY_KEY]: panel,
      [PLATFORM_FOCUS_QUERY_KEY]: executionId ? 'scheduled_execution' : 'scheduled_task',
      [PLATFORM_SCHEDULED_TASK_ID_QUERY_KEY]: String(request.taskId),
      ...(executionId ? { [PLATFORM_SCHEDULED_EXECUTION_ID_QUERY_KEY]: executionId } : {}),
      [PLATFORM_SCHEDULED_KIND_QUERY_KEY]: request.kind,
      ...(request.sourcePath ? { [PLATFORM_SOURCE_PATH_QUERY_KEY]: normalizeWorkspaceFullCodePath(request.sourcePath) } : {}),
      ...(request.traceId ? { [PLATFORM_TRACE_ID_QUERY_KEY]: request.traceId } : {}),

      // Compatibility for the existing scheduled task consumers.
      [LEGACY_SCHEDULED_OPEN_QUERY_KEY]: 'open',
    },
  })
}

export function buildOperateLogRoute(request: OperateLogRouteRequest) {
  const sourcePath = normalizeWorkspaceFullCodePath(request.sourcePath || request.fullCodePath)
  return buildWorkspaceRoute({
    fullCodePath: request.fullCodePath,
    query: {
      [PLATFORM_OPEN_QUERY_KEY]: 'operate_log',
      [PLATFORM_PANEL_QUERY_KEY]: 'operateLog',
      [PLATFORM_FOCUS_QUERY_KEY]: 'operate_log',
      ...(request.logId ? { [PLATFORM_LOG_ID_QUERY_KEY]: String(request.logId) } : {}),
      ...(request.traceId ? { [PLATFORM_TRACE_ID_QUERY_KEY]: request.traceId } : {}),
      ...(sourcePath ? { [PLATFORM_SOURCE_PATH_QUERY_KEY]: sourcePath } : {}),
    },
  })
}

export function isScheduledPanelQuery(query: LocationQuery | LocationQueryRaw, kind: PlatformScheduledKind): boolean {
  const open = readStringQuery(query, PLATFORM_OPEN_QUERY_KEY)
  const panel = readStringQuery(query, PLATFORM_PANEL_QUERY_KEY)
  const legacyOpen = readStringQuery(query, LEGACY_SCHEDULED_OPEN_QUERY_KEY)
  const scheduledKind = readStringQuery(query, PLATFORM_SCHEDULED_KIND_QUERY_KEY)
  const expectedPanel = kind === 'agent' ? 'scheduledAgentTask' : 'scheduledTask'

  if (open === 'scheduled' && panel === expectedPanel) return true
  if (legacyOpen === 'open' && scheduledKind === kind) return true
  return false
}

export function isOperateLogPanelQuery(query: LocationQuery | LocationQueryRaw): boolean {
  return readStringQuery(query, PLATFORM_OPEN_QUERY_KEY) === 'operate_log'
    || readStringQuery(query, PLATFORM_PANEL_QUERY_KEY) === 'operateLog'
}

export function isInboxOpenQuery(query: LocationQuery | LocationQueryRaw): boolean {
  return readStringQuery(query, PLATFORM_OPEN_QUERY_KEY) === 'inbox'
}

export function clearInboxRouteQuery(query: Record<string, unknown>): void {
  const isInboxQuery = query[PLATFORM_OPEN_QUERY_KEY] === 'inbox'
    || query[PLATFORM_MESSAGE_ID_QUERY_KEY] !== undefined
  if (query[PLATFORM_OPEN_QUERY_KEY] === 'inbox') {
    delete query[PLATFORM_OPEN_QUERY_KEY]
  }
  if (query[PLATFORM_FOCUS_QUERY_KEY] === 'message') {
    delete query[PLATFORM_FOCUS_QUERY_KEY]
  }
  delete query[PLATFORM_MESSAGE_ID_QUERY_KEY]
  if (isInboxQuery) {
    delete query[PLATFORM_TRACE_ID_QUERY_KEY]
    delete query[PLATFORM_SOURCE_PATH_QUERY_KEY]
  }
}

export function clearScheduledRouteQuery(query: Record<string, unknown>): void {
  const isScheduledQuery = query[PLATFORM_OPEN_QUERY_KEY] === 'scheduled'
    || query[LEGACY_SCHEDULED_OPEN_QUERY_KEY] === 'open'
    || query[PLATFORM_SCHEDULED_TASK_ID_QUERY_KEY] !== undefined
  if (query[PLATFORM_OPEN_QUERY_KEY] === 'scheduled') {
    delete query[PLATFORM_OPEN_QUERY_KEY]
  }
  const focus = String(query[PLATFORM_FOCUS_QUERY_KEY] || '')
  if (focus === 'scheduled_task' || focus === 'scheduled_execution') {
    delete query[PLATFORM_FOCUS_QUERY_KEY]
  }
  delete query[PLATFORM_SCHEDULED_TASK_ID_QUERY_KEY]
  delete query[PLATFORM_SCHEDULED_EXECUTION_ID_QUERY_KEY]
  delete query[PLATFORM_SCHEDULED_KIND_QUERY_KEY]
  delete query[LEGACY_SCHEDULED_OPEN_QUERY_KEY]
  if (isScheduledQuery) {
    delete query[PLATFORM_TRACE_ID_QUERY_KEY]
    delete query[PLATFORM_SOURCE_PATH_QUERY_KEY]
  }
}

export function clearOperateLogRouteQuery(query: Record<string, unknown>): void {
  const isOperateLogQuery = query[PLATFORM_OPEN_QUERY_KEY] === 'operate_log'
    || query[PLATFORM_FOCUS_QUERY_KEY] === 'operate_log'
    || query[PLATFORM_LOG_ID_QUERY_KEY] !== undefined
  if (query[PLATFORM_OPEN_QUERY_KEY] === 'operate_log') {
    delete query[PLATFORM_OPEN_QUERY_KEY]
  }
  if (query[PLATFORM_FOCUS_QUERY_KEY] === 'operate_log') {
    delete query[PLATFORM_FOCUS_QUERY_KEY]
  }
  delete query[PLATFORM_LOG_ID_QUERY_KEY]
  if (isOperateLogQuery) {
    delete query[PLATFORM_TRACE_ID_QUERY_KEY]
    delete query[PLATFORM_SOURCE_PATH_QUERY_KEY]
  }
}
