import { post } from '@/architecture/infrastructure/apiClient/request'
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
