import { get, put, del, post } from '@/utils/request'

export interface Doc {
  id: number
  tree_id: number
  name: string
  content: string
  format: string
  summary?: string
  category?: string
  created_at: string
  updated_at: string
  created_by?: string
  updated_by?: string
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
  // 后端 UpdateDocReq 要求 body 中必传 full_code_path
  const body = { full_code_path: path, ...data }
  return put<Doc>(`/workspace/api/v1/docs/info${path}`, body)
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
 * 搜索文档（模糊搜索）
 */
export interface SearchDocsReq {
  keyword: string
  page?: number
  page_size?: number
  include_content?: boolean // 是否包含文档内容（默认 true，设为 false 时只返回元数据，适合列表展示）
}

export interface DocSearchResult {
  id: number
  name: string
  content?: string // 可选，根据 include_content 决定
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

/**
 * 批量获取文档（精确查询）
 */
/**
 * 批量获取文档（精确查询）
 */
export interface BatchGetDocsReq {
  paths: string[]
  include_content?: boolean // 是否包含文档内容（默认 true）
}

export interface BatchGetDocsResp {
  docs: DocSearchResult[]
}

/**
 * 搜索文档（模糊搜索）
 * @param req 搜索请求
 */
export function searchDocs(req: SearchDocsReq) {
  const params = new URLSearchParams()
  params.append('keyword', req.keyword)
  
  if (req.page) {
    params.append('page', req.page.toString())
  }
  if (req.page_size) {
    params.append('page_size', req.page_size.toString())
  }
  if (req.include_content !== undefined) {
    params.append('include_content', req.include_content.toString())
  }
  
  return get<SearchDocsResp>(`/workspace/api/v1/docs/search?${params.toString()}`)
}

/**
 * 批量获取文档（精确查询）
 * @param req 批量查询请求
 */
export function batchGetDocs(req: BatchGetDocsReq) {
  const params = new URLSearchParams()
  
  req.paths.forEach(path => {
    params.append('paths', path)
  })
  
  if (req.include_content !== undefined) {
    params.append('include_content', req.include_content.toString())
  }
  
  return get<BatchGetDocsResp>(`/workspace/api/v1/docs/batch?${params.toString()}`)
}
