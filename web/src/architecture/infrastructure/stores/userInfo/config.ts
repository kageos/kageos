/**
 * 用户信息缓存配置
 */
export const USER_INFO_CACHE_CONFIG = {
  /** 缓存过期时间（毫秒），默认30分钟 */
  CACHE_EXPIRY_TIME: 30 * 60 * 1000,
  
  /** 降级策略：接口超时时间（毫秒），超过此时间使用过期缓存 */
  API_TIMEOUT: 300,
  
  /** 等待加载的轮询间隔（毫秒） */
  LOADING_CHECK_INTERVAL: 50,
  
  /** 等待加载的最大超时时间（毫秒） */
  LOADING_MAX_TIMEOUT: 5000,
  
  /** localStorage 存储键名 */
  STORAGE_KEY: 'user-info-cache',
} as const

