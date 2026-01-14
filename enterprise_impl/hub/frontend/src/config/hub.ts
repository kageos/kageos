/**
 * Hub 配置
 * 
 * Hub 是 AI-Agent-OS 的应用市场，提供应用发布、浏览、克隆等功能。
 * 
 * 配置方式：
 * 1. 通过环境变量配置（推荐）
 *    - VITE_HUB_API_BASE_URL: Hub API 基础地址
 * 
 * 2. 开发环境默认值：
 *    - baseURL: /hub/api/v1（通过 Vite 代理到网关）
 * 
 * 3. 生产环境：
 *    - 默认使用官方 Hub: https://www.ai-agent-os.com/hub/api/v1
 *    - 可以通过环境变量覆盖
 */

/**
 * 获取 Hub API 基础地址
 */
export function getHubBaseURL(): string {
  // 从环境变量获取配置
  const baseURL = import.meta.env.VITE_HUB_API_BASE_URL || getDefaultHubURL()
  return baseURL
}

/**
 * 获取默认 Hub URL
 * 开发环境使用本地地址，生产环境使用官方地址
 */
function getDefaultHubURL(): string {
  // 判断是否为开发环境
  const isDev = import.meta.env.DEV || import.meta.env.MODE === 'development'
  
  if (isDev) {
    // 开发环境：使用本地地址（通过 Vite 代理到网关）
    return '/hub/api/v1'
  } else {
    // 生产环境：使用官方地址
    return 'https://www.ai-agent-os.com/hub/api/v1'
  }
}

