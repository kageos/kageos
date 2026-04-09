export interface FileItem {
  name: string
  source_name?: string
  storage?: string
  description: string
  hash: string
  size: number
  upload_ts: number
  local_path: string
  is_uploaded: boolean
  url: string
  server_url?: string
  downloaded?: boolean
  upload_user?: string
}

export interface FilesData {
  files: FileItem[]
  remark: string
  metadata: Record<string, any>
  upload_user?: string
  widget_type?: string
  data_type?: string
}
