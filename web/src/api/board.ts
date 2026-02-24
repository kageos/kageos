import { get, post, put, del } from '@/utils/request'

export interface PostItem {
  id: number
  tree_id: number
  title: string
  cover?: string[]
  author: string
  status: string
  created_at: string
  updated_at: string
}

export interface ListPostsResp {
  list: PostItem[]
  total: number
}

export interface GetPostResp extends PostItem {
  full_code_path: string
  cover: string[]
  content: string
  content_format: string
}

/** 帖子列表（按版块路径分页） */
export function listPosts(params: { full_code_path: string; page?: number; page_size?: number }) {
  return get<ListPostsResp>('/workspace/api/v1/posts', {
    full_code_path: params.full_code_path,
    page: params.page ?? 1,
    page_size: params.page_size ?? 20
  })
}

/** 帖子详情 */
export function getPost(id: number) {
  return get<GetPostResp>(`/workspace/api/v1/posts/${id}`)
}

/** 发帖 */
export function createPost(data: {
  full_code_path: string
  title: string
  cover?: string[]
  content?: string
  content_format?: string
  status?: string
}) {
  return post<GetPostResp>('/workspace/api/v1/posts', data)
}

/** 更新帖子 */
export function updatePost(
  id: number,
  data: { title?: string; cover?: string[]; content?: string; content_format?: string; status?: string }
) {
  return put<GetPostResp>(`/workspace/api/v1/posts/${id}`, data)
}

/** 删除帖子 */
export function deletePost(id: number) {
  return del(`/workspace/api/v1/posts/${id}`)
}
