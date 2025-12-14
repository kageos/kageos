/**
 * Hub 导航工具
 * 
 * 用于从 OS 跳转到 Hub 时传递 token，实现跨站点登录
 */

/**
 * 获取 Hub 前端地址
 */
export function getHubFrontendURL(): string {
  // 从环境变量获取 Hub 前端地址
  const hubURL = import.meta.env.VITE_HUB_FRONTEND_URL
  
  if (hubURL) {
    return hubURL
  }
  
  // 判断是否为开发环境
  const isDev = import.meta.env.DEV || import.meta.env.MODE === 'development'
  
  if (isDev) {
    // 开发环境：默认使用本地 Hub 前端地址
    return 'http://localhost:5174' // Hub 前端默认端口
  } else {
    // 生产环境：默认使用官方 Hub 地址
    return 'https://www.ai-agent-os.com/hub'
  }
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
 * 跳转到 Hub 应用详情页
 * @param appId Hub 应用ID
 */
export function navigateToHubAppDetail(appId: number) {
  navigateToHub(`/app/${appId}`)
}

/**
 * 跳转到 Hub 管理页面
 */
export function navigateToHubManage() {
  navigateToHub('/manage')
}





