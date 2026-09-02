import { get, post } from '@/architecture/infrastructure/apiClient/request'
import {
  ensurePublicAnonymousToken,
  getCurrentPublicShareId,
  publicShareAnonymousHeaders,
} from '@/architecture/infrastructure/api/publicShare'

export interface ResolvedFile {
  ref: string
  bucket: string
  key: string
  name?: string
  source_name?: string
  storage?: string
  description?: string
  size?: number
  content_type?: string
  hash?: string
  upload_user?: string
  upload_ts?: number
  download_url?: string
  server_download_url?: string
  thumbnail_ref?: string
  thumbnail_url?: string
  server_thumbnail_url?: string
  preview_kind?: string
  status?: string
  can_delete?: boolean
  deleted_at?: number
  deleted_by?: string
  error?: string
}

export async function resolveFileRefs(
  refs: string[],
  audience: 'browser' | 'server' | 'all' = 'browser'
): Promise<ResolvedFile[]> {
  const cleanRefs = refs.map(ref => ref.trim()).filter(Boolean)
  if (cleanRefs.length === 0) {
    return []
  }
  const publicShareId = getCurrentPublicShareId()
  if (publicShareId) {
    await ensurePublicAnonymousToken()
  }
  const endpoint = publicShareId
    ? `/storage/api/v1/public/share/${encodeURIComponent(publicShareId)}/files/resolve`
    : '/storage/api/v1/files/resolve'
  const resp = await post<{ files: ResolvedFile[] }>(endpoint, {
    refs: cleanRefs,
    audience,
  }, {
    headers: publicShareId ? publicShareAnonymousHeaders() : undefined,
  })
  return resp?.files ?? []
}

export interface UpdateFileDescriptionResult {
  ref: string
  bucket: string
  key: string
  description: string
}

export async function updateFileDescription(
  ref: string,
  description: string
): Promise<UpdateFileDescriptionResult> {
  return post<UpdateFileDescriptionResult>('/storage/api/v1/files/description', {
    ref,
    description,
  })
}

export interface DeleteFileRefResult {
  ref: string
  status: 'deleted' | 'already_deleted' | 'failed'
  released_bytes?: number
  deleted_at?: number
  deleted_by?: string
  error?: string
}

export interface DeleteFileRefsResult {
  results: DeleteFileRefResult[]
  deleted_count: number
  released_bytes: number
}

export async function deleteFileRefs(refs: string[]): Promise<DeleteFileRefsResult> {
  const cleanRefs = Array.from(new Set(refs.map(ref => ref.trim()).filter(Boolean)))
  if (cleanRefs.length === 0) {
    return { results: [], deleted_count: 0, released_bytes: 0 }
  }
  return post<DeleteFileRefsResult>('/storage/api/v1/files/delete', { refs: cleanRefs })
}

export interface SystemStorageAsset {
  id: number
  bucket: string
  ref: string
  file_key: string
  router: string
  file_name: string
  description?: string
  file_size: number
  content_type: string
  thumbnail_ref?: string
  thumbnail_url?: string
  preview_kind?: 'image' | 'video' | 'pdf' | string
  previewable: boolean
  username: string
  tenant: string
  status: string
  uploaded_at: string
  deleted_at?: string
  deleted_by?: string
  delete_error?: string
  download_count: number
  preview_count: number
  last_accessed_at?: string
}

export interface SystemStorageAssetDirectory {
  router: string
  file_count: number
  size_bytes: number
}

export interface SystemStorageAssetWorkspace {
  path: string
  file_count: number
  size_bytes: number
}

export interface SystemStorageAssetsResp {
  list: SystemStorageAsset[]
  total: number
  page: number
  page_size: number
  summary: {
    active_files: number
    active_bytes: number
    deleted_files: number
    failed_files: number
  }
  directories: SystemStorageAssetDirectory[]
  workspaces: SystemStorageAssetWorkspace[]
  metadata_available: boolean
  console_url?: string
  coverage: 'tracked_uploads' | string
}

export function listSystemStorageAssets(params: {
  page?: number
  page_size?: number
  router_prefix?: string
  status?: string
  keyword?: string
} = {}) {
  return get<SystemStorageAssetsResp>('/storage/api/v1/system/files', params)
}

export function getSystemStorageAssetAccessURL(ref: string, action: 'download' | 'preview' = 'download') {
  return post<{ url: string }>('/storage/api/v1/system/files/download', { ref, action })
}

export interface SystemStorageAssetAudit {
  id: number
  action: 'download' | 'preview' | string
  username?: string
  ip_address?: string
  user_agent?: string
  accessed_at: string
}

export function listSystemStorageAssetAudits(ref: string, pageSize = 30) {
  return get<{ list: SystemStorageAssetAudit[] }>('/storage/api/v1/system/files/audits', { ref, page_size: pageSize })
}
