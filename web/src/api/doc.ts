import { get, put, del, post } from '@/utils/request'

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
 * 查询文档（统一接口，支持路径批量查询和关键词搜索）
 */
export interface QueryDocsReq {
  // 路径批量查询模式：提供 paths
  paths?: string[]
  
  // 关键词搜索模式：提供 keyword
  keyword?: string
  page?: number
  page_size?: number
  
  // 通用参数
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

export interface QueryDocsResp {
  docs: DocSearchResult[]
  total: number
  page: number
  page_size: number
}

/**
 * 查询文档（统一接口）
 * @param req 查询请求
 */
export function queryDocs(req: QueryDocsReq) {
  return post<QueryDocsResp>('/workspace/api/v1/docs/query', {
    paths: req.paths || [],
    keyword: req.keyword || '',
    page: req.page || 1,
    page_size: req.page_size || 20,
    include_content: req.include_content !== false // 默认 true
  })
}

/**
 * 搜索文档（兼容旧接口，内部调用 queryDocs）
 * @deprecated 使用 queryDocs 代替
 */
export function searchDocs(req: { keyword?: string; page: number; page_size: number; include_content?: boolean }) {
  return queryDocs({
    keyword: req.keyword,
    page: req.page,
    page_size: req.page_size,
    include_content: req.include_content !== false
  })
}
