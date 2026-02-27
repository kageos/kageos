/**
 * OS 导航工具
 * 
 * 用于从 Hub 跳转到 OS 的 workspace
 */

/**
 * 获取 OS 前端地址
 */
export function getOSFrontendURL(): string {
  // 从环境变量获取 OS 前端地址
  const osURL = import.meta.env.VITE_OS_FRONTEND_URL
  
  if (osURL) {
    return osURL
  }
  
  // 判断是否为开发环境
  const isDev = import.meta.env.DEV || import.meta.env.MODE === 'development'
  
  // 自部署生产环境：Hub 8998 → Web 8999
  const currentPort = window.location.port
  if (currentPort === '8998') {
    return `${window.location.protocol}//${window.location.hostname}:8999`
  }

  // 开发环境
  return 'http://localhost:5173'
}

/**
 * 将路由路径转换为 OS workspace URL
 * @param router 路由路径，如：/luobei/demo/crm/crm_ticket
 * @returns OS workspace URL，如：http://localhost:5173/workspace/luobei/demo/crm/crm_ticket
 */
export function routerToOSWorkspaceURL(router: string): string {
  const osURL = getOSFrontendURL()
  
  // 移除开头的斜杠（如果有）
  const cleanRouter = router.startsWith('/') ? router.substring(1) : router
  
  // 构建 workspace URL
  return `${osURL}/workspace/${cleanRouter}`
}

/**
 * 跳转到 OS workspace
 * @param router 路由路径，如：/luobei/demo/crm/crm_ticket
 */
export function navigateToOSWorkspace(router: string) {
  const url = routerToOSWorkspaceURL(router)
  window.open(url, '_blank')
}








