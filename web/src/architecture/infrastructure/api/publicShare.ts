import { get, post } from '@/architecture/infrastructure/apiClient/request'
import type { FunctionDetail, FunctionSchema } from '@/architecture/domain/types'
import type { FormSubmitRequest, IFormGateway } from '@/architecture/domain/interfaces/IFormGateway'

const PUBLIC_ANONYMOUS_TOKEN_HEADER = 'X-Public-Anonymous-Token'
const PUBLIC_ANONYMOUS_TOKEN_STORAGE_KEY = 'public_anonymous_token'
let anonymousTokenPromise: Promise<string> | null = null

export interface PublicShareView {
  share_id: string
  title: string
  description?: string
  full_code_path: string
  schema: FunctionSchema
  expires_at?: string
  remaining_uses?: number
}

export interface PublicAnonymousToken {
  anonymous_token: string
  expires_at: string
}

export type PublicShareSubmitResult = unknown

export interface CreatePublicShareRequest {
  full_code_path: string
  title?: string
  description?: string
  expires_at?: string | null
  max_uses?: number
}

export interface PublicShareItem {
  share_id: string
  tenant_user: string
  app: string
  full_code_path: string
  resource_type: string
  action: string
  title: string
  description?: string
  enabled: boolean
  expires_at?: string
  max_uses: number
  use_count: number
  last_used_at?: string
  created_at: string
  created_by: string
  public_url?: string
}

export interface PublicShareList {
  items: PublicShareItem[]
}

export interface PublicShareListParams {
  full_code_path: string
  keyword?: string
  created_by?: string
  status?: string
}

function getAnonymousToken(): string {
  return localStorage.getItem(PUBLIC_ANONYMOUS_TOKEN_STORAGE_KEY) || ''
}

function saveAnonymousToken(token?: string) {
  if (token) {
    localStorage.setItem(PUBLIC_ANONYMOUS_TOKEN_STORAGE_KEY, token)
  }
}

export function publicShareAnonymousHeaders(): Record<string, string> {
  const token = getAnonymousToken()
  return token ? { [PUBLIC_ANONYMOUS_TOKEN_HEADER]: token } : {}
}

async function requestPublicAnonymousToken(): Promise<string> {
  const resp = await post<PublicAnonymousToken>('/public/api/anonymous-token', undefined, {
    headers: publicShareAnonymousHeaders(),
  })
  saveAnonymousToken(resp.anonymous_token)
  return resp.anonymous_token
}

export async function ensurePublicAnonymousToken(options: { refresh?: boolean } = {}): Promise<string> {
  const currentToken = getAnonymousToken()
  if (currentToken && !options.refresh) {
    return currentToken
  }

  if (!anonymousTokenPromise) {
    anonymousTokenPromise = requestPublicAnonymousToken().finally(() => {
      anonymousTokenPromise = null
    })
  }
  return anonymousTokenPromise
}

export function getCurrentPublicShareId(): string {
  const match = window.location.pathname.match(/^\/public\/s\/([^/]+)/)
  return match?.[1] ? decodeURIComponent(match[1]) : ''
}

export async function getPublicShareView(shareId: string): Promise<PublicShareView> {
  await ensurePublicAnonymousToken({ refresh: true })
  return get<PublicShareView>(`/public/api/s/${shareId}`, undefined, false, {
    headers: publicShareAnonymousHeaders(),
  })
}

export async function submitPublicShare(shareId: string, data: Record<string, unknown>): Promise<PublicShareSubmitResult> {
  await ensurePublicAnonymousToken()
  return post<PublicShareSubmitResult>(`/public/api/s/${shareId}/submit`, data, {
    headers: publicShareAnonymousHeaders(),
  })
}

export function createPublicShare(req: CreatePublicShareRequest): Promise<PublicShareItem> {
  return post<PublicShareItem>('/workspace/api/v1/public_shares', req)
}

export function listPublicShares(params: PublicShareListParams): Promise<PublicShareList> {
  return get<PublicShareList>('/workspace/api/v1/public_shares', params)
}

export function disablePublicShare(shareId: string): Promise<unknown> {
  return post(`/workspace/api/v1/public_shares/${shareId}/disable`)
}

export function createPublicShareFunctionDetail(view: PublicShareView): FunctionDetail {
  return {
    id: 0,
    method: 'POST',
    router: view.full_code_path,
    template_type: 'form',
    name: view.title,
    description: view.description,
    schema: view.schema,
  }
}

export class PublicShareFormGateway implements IFormGateway {
  constructor(private shareId: string) {}

  async submitForm(request: FormSubmitRequest): Promise<unknown> {
    const response = await submitPublicShare(this.shareId, request.data)
    return response && typeof response === 'object'
      ? response
      : { result: response }
  }
}
