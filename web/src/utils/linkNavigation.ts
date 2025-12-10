/**
 * Link 导航工具函数
 */

/**
 * 解析 Link 组件的 JSON 值
 */
export interface ParsedLinkValue {
  type?: 'table' | 'form'  // 函数类型（外链时为空）
  name: string             // 链接文本
  url: string              // 链接 URL
}

/**
 * 解析 Link 组件的值（JSON 格式）
 */
export function parseLinkValue(raw: string): ParsedLinkValue {
  if (!raw) {
    return { name: '', url: '', type: undefined }
  }
  
  try {
    const jsonValue = JSON.parse(raw)
    if (jsonValue && typeof jsonValue === 'object' && jsonValue.url) {
      return {
        name: jsonValue.name || '',
        url: jsonValue.url,
        type: jsonValue.type  // 'table' 或 'form' 或 undefined（外链）
      }
    }
  } catch (error) {
    // JSON 解析失败，返回空值
    console.error('[parseLinkValue] 解析失败:', error, raw)
    return { name: '', url: raw, type: undefined }
  }
  
  return { name: '', url: '', type: undefined }
}

/**
 * 为内部链接添加 _link_type 参数（用于传递函数类型信息）
 */
export function addLinkTypeToUrl(url: string, linkType?: 'table' | 'form'): string {
  if (!linkType) {
    return url
  }
  
  try {
    const urlObj = new URL(url, window.location.origin)
    urlObj.searchParams.set('_link_type', linkType)
    return urlObj.pathname + urlObj.search
  } catch {
    // URL 解析失败，返回原始 URL
    return url
  }
}

