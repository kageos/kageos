/**
 * Hub 导航工具
 * 
 * 用于从 OS 跳转到 Hub 时传递 token，实现跨站点登录
 */

/**
 * 获取 Hub 前端地址
 */
export function getHubFrontendURL(): string {
  // 优先使用环境变量
  const hubURL = import.meta.env.VITE_HUB_FRONTEND_URL
  if (hubURL) {
    return hubURL
  }

  // 自部署生产环境：Web 8999 → Hub 8998
  const currentPort = window.location.port
  if (currentPort === '8999') {
    return `${window.location.protocol}//${window.location.hostname}:8998`
  }

  // 开发环境或其他情况
  return 'http://localhost:5174'
}

/**
 * 跳转到 Hub 应用中心
 * @param path Hub 路径（可选，如 '/app/123'）
 */
export function navigateToHub(path: string = '/') {
  const hubURL = getHubFrontendURL()
  const token = localStorage.getItem('token')
  
  // 构建完整 URL
  const url = new URL(path, hubURL)
  
  // 如果有 token，通过 URL 参数传递
  if (token) {
    url.searchParams.set('token', token)
  }
  
  // 在新窗口打开 Hub
  window.open(url.toString(), '_blank')
}

/**
 * 跳转到 Hub 目录详情页
 * @param hubFullCodePath Hub 目录完整路径（如 luobei/demos/xxx 或 /luobei/demos/xxx），内部拼成 /directory/xxx
 */
export function navigateToHubDirectoryDetail(hubFullCodePath: string) {
  const path = hubFullCodePath.replace(/^\//, '')
  navigateToHub(path ? `/directory/${path}` : '/directory')
}

/**
 * 跳转到 Hub 管理页面
 */
export function navigateToHubManage() {
  navigateToHub('/manage')
}





