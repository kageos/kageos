import { get, patch } from '@/utils/request'

export interface MessageInboxItem {
  id: number
  recipient_id: number
  from: string
  request_user: string
  department_full_path: string
  full_code_path: string
  trace_id: string
  client_source: string
  source_type: string
  source_ref: string
  title: string
  content: string
  content_type: 'markdown' | 'html' | 'text' | string
  read_at?: string | null
  created_at: string
}

export interface MessageInboxListResp {
  list: MessageInboxItem[]
  total: number
  page: number
  page_size: number
}

export interface MessageUnreadCountResp {
  unread_count: number
}

export function listInboxMessages(params?: { page?: number; page_size?: number; status?: 'unread' | 'all' | '' }) {
  return get<MessageInboxListResp>('/message/api/v1/inbox', params)
}

export function getInboxMessage(id: number) {
  return get<MessageInboxItem>(`/message/api/v1/inbox/${id}`)
}

export function getMessageUnreadCount() {
  return get<MessageUnreadCountResp>('/message/api/v1/inbox/unread_count')
}

export function markInboxMessageRead(id: number) {
  return patch(`/message/api/v1/inbox/${id}/read`)
}

export function markAllInboxMessagesRead() {
  return patch('/message/api/v1/inbox/read_all')
}
