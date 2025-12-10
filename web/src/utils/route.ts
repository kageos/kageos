/**
 * 路由工具函数
 * 
 * 用于处理工作空间相关的路由路径转换
 */

import type { RouteLocationNormalized } from 'vue-router'

/**
 * 从工作空间路径中提取相对路径
 * 
 * @param path 完整路径（如 '/workspace/user/app/function'）
 * @returns 相对路径（如 'user/app/function'）
 * 
 * @example
 * extractWorkspacePath('/workspace/user/app/function') // 'user/app/function'
 * extractWorkspacePath('/workspace/user/app') // 'user/app'
 */
export function extractWorkspacePath(path: string): string {
  return path.replace('/workspace/', '').replace(/^\/+|\/+$/g, '')
}

/**
 * 将 URL 转换为工作空间路由路径
 * 
 * @param url 原始 URL（可能是相对路径、绝对路径或完整 URL）
 * @param currentRoute 当前路由对象（可选，用于解析相对路径）
 * @returns 处理后的路由路径
 * 
 * @example
 * // 外链直接返回
 * resolveWorkspaceUrl('https://example.com') // 'https://example.com'
 * 
 * // 已经是完整路径，直接返回
 * resolveWorkspaceUrl('/workspace/user/app/function') // '/workspace/user/app/function'
 * 
 * // 绝对路径，添加 /workspace 前缀
 * resolveWorkspaceUrl('/user/app/function') // '/workspace/user/app/function'
 * 
 * // 相对路径，需要从当前路由获取 user 和 app
 * resolveWorkspaceUrl('function_name', { path: '/workspace/user/app' }) // '/workspace/user/app/function_name'
 */
export function resolveWorkspaceUrl(
  url: string,
  currentRoute?: RouteLocationNormalized | { path: string }
): string {
  if (!url) {
    return ''
  }
  
  // 如果是外链（包含协议），直接返回
  if (url.startsWith('http://') || url.startsWith('https://')) {
    return url
  }
  
  // 如果已经是完整路径（包含 /workspace），直接返回
  if (url.startsWith('/workspace/')) {
    return url
  }
  
  // 如果是绝对路径（以 / 开头），添加 /workspace 前缀
  if (url.startsWith('/')) {
    const pathWithoutSlash = url.substring(1)
    return `/workspace/${pathWithoutSlash}`
  }
  
  // 相对路径，需要转换为完整路径
  if (currentRoute) {
    const pathParts = currentRoute.path.split('/').filter(Boolean)
    if (pathParts.length >= 3) {
      // 从当前路由获取 user 和 app
      const user = pathParts[1]
      const app = pathParts[2]
      const [functionPath, query] = url.split('?')
      const fullPath = `/workspace/${user}/${app}/${functionPath}`
      return query ? `${fullPath}?${query}` : fullPath
    }
  }
  
  // 如果路径格式不正确，尝试添加 /workspace 前缀
  return `/workspace/${url}`
}

