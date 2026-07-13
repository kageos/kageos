import { del, get, patch, post, put } from '@/architecture/infrastructure/apiClient/request'

export type MessageContentType = 'markdown' | 'html' | 'text' | string
export type MessageInboxStatus = 'all' | 'unread'

export interface MessageSendMeta {
  from?: string
  request_user?: string
  department_full_path?: string
  full_code_path?: string
  trace_id?: string
  client_source?: string
  source_type?: string
  source_ref?: string
  source_path?: string
  source_title?: string
  source_parent_path?: string
  source_parent_title?: string
  source_template_type?: string
  workspace_session_id?: string
  workspace_session_title?: string
  workspace_role?: string
  thread_key?: string
}

export interface MessageSendPayload {
  to_users?: string
  title?: string
  content?: string
  content_type?: MessageContentType
  files?: string
}

export interface MessageSendEnvelope {
  meta?: MessageSendMeta
  message: MessageSendPayload
}

export interface MessageSendToUsersReq {
  to_users?: string
  title?: string
  content?: string
  content_type?: MessageContentType
  files?: string
}

export interface MessageSendResp {
  message: string
  meta: MessageSendMeta
  payload: MessageSendPayload
  from: string
  full_code_path?: string
  to_users?: string
  content_type?: MessageContentType
  files?: string
}

export interface MessageSourceDisplay {
  name: string
  type: string
  template_type?: string
  full_code_path?: string
  parent_name?: string
  parent_full_code_path?: string
  thread_key?: string
}

export interface MessageInboxItem {
  id: number
  recipient_id: number
  from: string
  request_user?: string
  department_full_path?: string
  full_code_path?: string
  trace_id?: string
  client_source?: string
  source_type?: string
  source_ref?: string
  source_path?: string
  source_title?: string
  source_parent_path?: string
  source_parent_title?: string
  source_template_type?: string
  workspace_session_id?: string
  workspace_session_title?: string
  workspace_role?: string
  thread_key?: string
  scheduled_task_id?: number
  scheduled_execution_id?: number
  title?: string
  content: string
  content_type?: MessageContentType
  files?: string
  read_at?: string | null
  created_at: string
  source_display?: MessageSourceDisplay
}

export interface ListMessageInboxParams {
  status?: MessageInboxStatus
  thread_key?: string
  source_path?: string
  include_children?: boolean
  page?: number
  page_size?: number
}

export interface ListMessageInboxResp {
  list: MessageInboxItem[]
  total: number
  page: number
  page_size: number
}

export interface MessageUnreadCountResp {
  unread_count: number
}

export interface MessageActionViewResp {
  token_status: 'open' | 'submitted' | 'expired' | 'revoked' | string
  recipient_user: string
  channel?: string
  authenticated_user?: string
  allowed_actions: string[]
  can_reply: boolean
  expires_at: string
  message: MessageInboxItem
  thread: MessageInboxItem[]
  mobile_ask_url?: string
  workspace_session_id?: string
  submitted_at?: string | null
  reply_message_id?: number
}

export interface MessageActionReplyReq {
  content: string
  files?: string
  action?: string
}

export interface MessageActionReplyResp {
  status: string
  reply_message_id: number
  submitted_at: string
  mobile_ask_url?: string
  channel?: string
  source_path?: string
  full_code_path?: string
  workspace_session_id?: string
  agent_submitted: boolean
  agent_submit_error?: string
  workstation_draft?: string
}

export type MessageNotificationChannel = 'feishu' | 'wecom' | 'dingtalk' | string

export interface MessageNotificationChannelInfo {
  channel: MessageNotificationChannel
  enabled: boolean
  delivery_type: string
  display_name?: string
  has_webhook_url: boolean
  has_secret: boolean
  metadata?: Record<string, string>
  updated_at?: string
  last_success_at?: string
  last_failed_at?: string
  last_test_at?: string
  last_error?: string
  fail_count?: number
}

export interface MessageNotificationChannelListResp {
  list: MessageNotificationChannelInfo[]
}

export interface MessageNotificationRouteInfo {
  id: number
  scope_path: string
  scope_type: 'workspace' | 'directory' | 'function' | string
  channel: MessageNotificationChannel
  enabled: boolean
  delivery_type: string
  display_name?: string
  remark?: string
  has_webhook_url: boolean
  has_secret: boolean
  metadata?: Record<string, string>
  updated_at?: string
  last_success_at?: string
  last_failed_at?: string
  last_test_at?: string
  last_error?: string
  fail_count?: number
}

export interface MessageNotificationRouteListResp {
  list: MessageNotificationRouteInfo[]
}

export interface MessageNotificationRoutePathSummary {
  scope_path: string
  routes: MessageNotificationRouteInfo[]
}

export interface MessageNotificationRouteSummaryResp {
  routes: Record<string, MessageNotificationRoutePathSummary>
}

export interface UpsertMessageNotificationChannelReq {
  channel?: MessageNotificationChannel
  enabled?: boolean
  delivery_type?: string
  display_name?: string
  webhook_url?: string
  secret?: string
  clear_webhook_url?: boolean
  clear_secret?: boolean
  metadata?: Record<string, string>
}

export interface UpsertMessageNotificationRouteReq {
  scope_path: string
  scope_type?: string
  channel?: MessageNotificationChannel
  enabled?: boolean
  delivery_type?: string
  display_name?: string
  remark?: string
  webhook_url?: string
  secret?: string
  clear_webhook_url?: boolean
  clear_secret?: boolean
  metadata?: Record<string, string>
}

export interface TestMessageNotificationChannelResp {
  message: string
  channel: MessageNotificationChannel
}

export interface MessageInboxSourceCount {
  source_path: string
  unread_count: number
  message_count: number
  latest_at?: string
}

export interface ListMessageInboxSourceCountsResp {
  list: MessageInboxSourceCount[]
}

export interface MessageInboxWorkspaceCount {
  workspace_key: string
  workspace_user?: string
  workspace_code?: string
  workspace_path?: string
  title?: string
  unread_count: number
  message_count: number
  latest_at?: string
  latest_source_path?: string
  latest_source_title?: string
}

export interface ListMessageInboxWorkspaceCountsResp {
  list: MessageInboxWorkspaceCount[]
}

export interface MessageInboxThread {
  key: string
  kind: 'directory' | 'function' | 'session' | 'sender' | string
  title: string
  subtitle: string
  path?: string
  unread_count: number
  message_count: number
  latest_at: string
  last_message: MessageInboxItem
  scheduled_task_id?: number
  scheduled_execution_id?: number
}

export interface ListMessageInboxThreadsResp {
  list: MessageInboxThread[]
  total: number
  page: number
  page_size: number
}

export function sendMessage(data: MessageSendEnvelope): Promise<MessageSendResp> {
  return post<MessageSendResp>('/message/api/v1/send', data)
}

export function sendMessageToUsers(data: MessageSendToUsersReq): Promise<MessageSendResp> {
  return post<MessageSendResp>('/message/api/v1/send/users', data)
}

export function listMessageInbox(params: ListMessageInboxParams = {}): Promise<ListMessageInboxResp> {
  return get<ListMessageInboxResp>('/message/api/v1/inbox', params)
}

export function listMessageInboxThreads(params: ListMessageInboxParams = {}): Promise<ListMessageInboxThreadsResp> {
  return get<ListMessageInboxThreadsResp>('/message/api/v1/inbox/threads', params)
}

export function listMessageInboxSourceCounts(params: Pick<ListMessageInboxParams, 'status'> = {}): Promise<ListMessageInboxSourceCountsResp> {
	return get<ListMessageInboxSourceCountsResp>('/message/api/v1/inbox/source-counts', params)
}

export function listMessageInboxWorkspaceCounts(params: Pick<ListMessageInboxParams, 'status'> = {}): Promise<ListMessageInboxWorkspaceCountsResp> {
	return get<ListMessageInboxWorkspaceCountsResp>('/message/api/v1/inbox/workspace-counts', params)
}

export function getMessageInboxUnreadCount(): Promise<MessageUnreadCountResp> {
	return get<MessageUnreadCountResp>('/message/api/v1/inbox/unread-count')
}

export function getMessageInboxItem(id: number): Promise<MessageInboxItem> {
  return get<MessageInboxItem>(`/message/api/v1/inbox/${id}`)
}

export function listMessageNotificationChannels(): Promise<MessageNotificationChannelListResp> {
	return get<MessageNotificationChannelListResp>('/message/api/v1/notification-channels')
}

export function upsertMessageNotificationChannel(
  channel: MessageNotificationChannel,
  data: UpsertMessageNotificationChannelReq
): Promise<MessageNotificationChannelInfo> {
	return put<MessageNotificationChannelInfo>(`/message/api/v1/notification-channels/${channel}`, data)
}

export function deleteMessageNotificationChannel(channel: MessageNotificationChannel): Promise<void> {
	return del<void>(`/message/api/v1/notification-channels/${channel}`)
}

export function testMessageNotificationChannel(channel: MessageNotificationChannel): Promise<TestMessageNotificationChannelResp> {
	return post<TestMessageNotificationChannelResp>(`/message/api/v1/notification-channels/${channel}/test`)
}

export function listMessageNotificationRoutes(scopePath?: string): Promise<MessageNotificationRouteListResp> {
	return get<MessageNotificationRouteListResp>('/message/api/v1/notification-routes', scopePath ? { scope_path: scopePath } : {})
}

export function listMessageNotificationRouteSummary(rootScopePath: string): Promise<MessageNotificationRouteSummaryResp> {
	return get<MessageNotificationRouteSummaryResp>('/message/api/v1/notification-routes/summary', { root_scope_path: rootScopePath })
}

export function upsertMessageNotificationRoute(
  channel: MessageNotificationChannel,
  data: UpsertMessageNotificationRouteReq
): Promise<MessageNotificationRouteInfo> {
	return put<MessageNotificationRouteInfo>(`/message/api/v1/notification-routes/${channel}`, data)
}

export function deleteMessageNotificationRoute(channel: MessageNotificationChannel, scopePath: string): Promise<void> {
  const params = new URLSearchParams({ scope_path: scopePath })
	return del<void>(`/message/api/v1/notification-routes/${channel}?${params.toString()}`)
}

export function testMessageNotificationRoute(channel: MessageNotificationChannel, scopePath: string): Promise<TestMessageNotificationChannelResp> {
	return post<TestMessageNotificationChannelResp>(`/message/api/v1/notification-routes/${channel}/test`, { scope_path: scopePath })
}

export function markMessageInboxItemRead(id: number): Promise<void> {
  return patch<void>(`/message/api/v1/inbox/${id}/read`)
}

export function markMessageInboxSourceRead(sourcePath: string, includeChildren = false): Promise<void> {
  const params = new URLSearchParams({ source_path: sourcePath })
  if (includeChildren) {
    params.set('include_children', 'true')
  }
	return patch<void>(`/message/api/v1/inbox/read-source?${params.toString()}`)
}

export function markAllMessageInboxItemsRead(): Promise<void> {
	return patch<void>('/message/api/v1/inbox/read-all')
}

export function getPublicMessageAction(token: string): Promise<MessageActionViewResp> {
  return get<MessageActionViewResp>(`/message/api/v1/public/actions/${encodeURIComponent(token)}`)
}

export function submitPublicMessageActionReply(token: string, data: MessageActionReplyReq): Promise<MessageActionReplyResp> {
  return post<MessageActionReplyResp>(`/message/api/v1/public/actions/${encodeURIComponent(token)}/reply`, data)
}
