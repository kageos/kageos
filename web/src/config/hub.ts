/**
 * Hub 配置
 * 
 * Hub 是 AI-Agent-OS 的应用市场，提供应用发布、浏览、克隆等功能。
 * 
 * 配置方式：
 * 1. 通过环境变量配置（推荐）
 *    - VITE_HUB_BASE_URL: Hub API 基础地址
 *    - VITE_HUB_ENABLED: 是否启用 Hub 功能（默认 true）
 * 
 * 2. 开发环境默认值：
 *    - baseURL: http://127.0.0.1:9094/hub（本地开发）
 *    - enabled: true
 * 
 * 3. 生产环境：
 *    - 默认使用官方 Hub: https://www.ai-agent-os.com/hub
 *    - 可以通过环境变量覆盖
 */
export interface HubConfig {
  baseURL: string
  enabled: boolean
}

/**
 * 获取 Hub 配置
 */
export function getHubConfig(): HubConfig {
  // 从环境变量获取配置
  const baseURL = import.meta.env.VITE_HUB_BASE_URL || getDefaultHubURL()
  const enabled = import.meta.env.VITE_HUB_ENABLED !== 'false'

  return {
    baseURL,
    enabled,
  }
}

/**
 * 获取默认 Hub URL
 * 开发环境使用本地地址，生产环境使用官方地址
 */
function getDefaultHubURL(): string {
  // 判断是否为开发环境
  const isDev = import.meta.env.DEV || import.meta.env.MODE === 'development'
  
  if (isDev) {
    // 开发环境：使用本地地址（通过网关代理）
    return '/hub'
  } else {
    // 生产环境：使用官方地址
    return 'https://www.ai-agent-os.com/hub'
  }
}

/**
 * 获取 Hub API 基础地址
 */
export function getHubBaseURL(): string {
  const config = getHubConfig()
  return config.baseURL
}

/**
 * 检查 Hub 是否启用
 */
export function isHubEnabled(): boolean {
  const config = getHubConfig()
  return config.enabled
}

/**
 * Hub 配置实例（单例）
 */
export const HUB_CONFIG = getHubConfig()

