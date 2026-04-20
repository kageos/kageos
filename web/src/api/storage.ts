import { post } from '@/utils/request'

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
  const resp = await post<{ files: ResolvedFile[] }>('/storage/api/v1/files/resolve', {
    refs: cleanRefs,
    audience,
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
