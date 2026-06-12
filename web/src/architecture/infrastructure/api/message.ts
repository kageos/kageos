import { get, patch, post } from '@/architecture/infrastructure/apiClient/request'

export type MessageContentType = 'markdown' | 'text' | string
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
  content: string
  content_type?: MessageContentType
}

export interface MessageSendEnvelope {
  meta?: MessageSendMeta
  message: MessageSendPayload
}

export interface MessageSendToUsersReq {
  to_users: string
  title?: string
  content: string
  content_type?: MessageContentType
}

export interface MessageSendResp {
  message: string
  meta: MessageSendMeta
  payload: MessageSendPayload
  from: string
  full_code_path?: string
  to_users?: string
  content_type?: MessageContentType
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
  title?: string
  content: string
  content_type?: MessageContentType
  read_at?: string | null
  created_at: string
  source_display?: MessageSourceDisplay
}

export interface ListMessageInboxParams {
  status?: MessageInboxStatus
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

export function sendMessage(data: MessageSendEnvelope): Promise<MessageSendResp> {
  return post<MessageSendResp>('/message/api/v1/send', data)
}

export function sendMessageToUsers(data: MessageSendToUsersReq): Promise<MessageSendResp> {
  return post<MessageSendResp>('/message/api/v1/send/users', data)
}

export function listMessageInbox(params: ListMessageInboxParams = {}): Promise<ListMessageInboxResp> {
  return get<ListMessageInboxResp>('/message/api/v1/inbox', params)
}

export function getMessageInboxUnreadCount(): Promise<MessageUnreadCountResp> {
  return get<MessageUnreadCountResp>('/message/api/v1/inbox/unread_count')
}

export function getMessageInboxItem(id: number): Promise<MessageInboxItem> {
  return get<MessageInboxItem>(`/message/api/v1/inbox/${id}`)
}

export function markMessageInboxItemRead(id: number): Promise<void> {
  return patch<void>(`/message/api/v1/inbox/${id}/read`)
}

export function markAllMessageInboxItemsRead(): Promise<void> {
  return patch<void>('/message/api/v1/inbox/read_all')
}
