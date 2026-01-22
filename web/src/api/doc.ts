import { get, put, del } from '@/utils/request'

export interface Doc {
  id: number
  tree_id: number
  name: string
  content: string
  format: string
  summary?: string
  created_at: string
  updated_at: string
}

/**
 * 获取文档（基于路径）
 * @param fullCodePath 完整路径（如：/user/app/docs/guide 或 user/app/docs/guide）
 */
export function getDoc(fullCodePath: string) {
  const path = fullCodePath.startsWith('/') ? fullCodePath : `/${fullCodePath}`
  return get<Doc>(`/workspace/api/v1/docs/info${path}`)
}

/**
 * 更新文档（基于路径）
 * @param fullCodePath 完整路径（如：/user/app/docs/guide 或 user/app/docs/guide）
 * @param data 更新内容
 */
export function updateDoc(
  fullCodePath: string,
  data: { content?: string; format?: string; summary?: string }
) {
  const path = fullCodePath.startsWith('/') ? fullCodePath : `/${fullCodePath}`
  return put<Doc>(`/workspace/api/v1/docs/info${path}`, data)
}

/**
 * 删除文档（基于路径）
 * @param fullCodePath 完整路径（如：/user/app/docs/guide 或 user/app/docs/guide）
 */
export function deleteDoc(fullCodePath: string) {
  const path = fullCodePath.startsWith('/') ? fullCodePath : `/${fullCodePath}`
  return del(`/workspace/api/v1/docs/info${path}`)
}

/**
 * 搜索文档
 */
export interface SearchDocsReq {
  keyword?: string
  page: number
  page_size: number
}

export interface DocSearchResult {
  id: number
  name: string
  content: string
  format: string
  full_code_path: string
  summary?: string
  category?: string
}

export interface SearchDocsResp {
  docs: DocSearchResult[]
  total: number
  page: number
  page_size: number
}

export function searchDocs(req: SearchDocsReq) {
  return get<SearchDocsResp>('/workspace/api/v1/docs/search', {
    keyword: req.keyword || '',
    page: req.page.toString(),
    page_size: req.page_size.toString()
  })
}
