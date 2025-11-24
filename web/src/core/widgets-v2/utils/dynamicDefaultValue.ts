/**
 * 动态默认值解析工具
 * 
 * 支持以下动态变量：
 * - 用户选择器：$me（当前登录用户）
 * - 时间戳组件：$now（当前时间）、$today（今天开始）、$tomorrow（明天开始）、$yesterday（昨天开始）
 */

/**
 * 解析动态默认值
 * @param defaultValue 默认值（可能是动态变量，如 $me, $now 等）
 * @param widgetType 组件类型
 * @param getAuthStore 获取 authStore 的函数（可选，用于延迟获取）
 * @returns 解析后的值
 */
export function resolveDynamicDefaultValue(
  defaultValue: any,
  widgetType: string,
  getAuthStore?: () => any
): any {
  // 如果不是字符串，直接返回
  if (typeof defaultValue !== 'string') {
    return defaultValue
  }

  // 检查是否是动态变量（以 $ 开头）
  if (!defaultValue.startsWith('$')) {
    return defaultValue
  }

  // 用户选择器：$me
  if (widgetType === 'user') {
    if (defaultValue === '$me') {
      // 🔥 延迟获取 authStore，避免在工具函数中直接调用
      if (getAuthStore) {
        const authStore = getAuthStore()
        return authStore?.user?.username || null
      }
      // 如果没有提供 getAuthStore，返回原值（让组件自己处理）
      return defaultValue
    }
  }

  // 时间戳组件：支持多种动态变量
  if (widgetType === 'timestamp') {
    const now = new Date()
    
    switch (defaultValue) {
      // ========== 基础时间 ==========
      case '$now':
        // 当前时间（毫秒时间戳）
        return now.getTime()
      
      case '$today':
        // 今天开始时间（00:00:00）
        const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
        return today.getTime()
      
      case '$tomorrow':
        // 明天开始时间（00:00:00）
        const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1)
        return tomorrow.getTime()
      
      case '$yesterday':
        // 昨天开始时间（00:00:00）
        const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1)
        return yesterday.getTime()
      
      // ========== 相对时间（此刻） ==========
      case '$yesterday_now':
        // 昨天此刻
        const yesterdayNow = new Date(now.getTime() - 24 * 60 * 60 * 1000)
        return yesterdayNow.getTime()
      
      case '$tomorrow_now':
        // 明天此刻
        const tomorrowNow = new Date(now.getTime() + 24 * 60 * 60 * 1000)
        return tomorrowNow.getTime()
      
      // ========== 相对时间（小时） ==========
      case '$after_1h':
        // 一小时后
        return now.getTime() + 1 * 60 * 60 * 1000
      
      case '$after_2h':
        // 两小时后
        return now.getTime() + 2 * 60 * 60 * 1000
      
      case '$after_3h':
        // 三小时后
        return now.getTime() + 3 * 60 * 60 * 1000
      
      case '$after_6h':
        // 六小时后
        return now.getTime() + 6 * 60 * 60 * 1000
      
      case '$after_12h':
        // 十二小时后
        return now.getTime() + 12 * 60 * 60 * 1000
      
      case '$before_1h':
        // 一小时前
        return now.getTime() - 1 * 60 * 60 * 1000
      
      case '$before_2h':
        // 两小时前
        return now.getTime() - 2 * 60 * 60 * 1000
      
      case '$before_3h':
        // 三小时前
        return now.getTime() - 3 * 60 * 60 * 1000
      
      // ========== 相对时间（天） ==========
      case '$after_1d':
        // 一天后（24小时后）
        return now.getTime() + 24 * 60 * 60 * 1000
      
      case '$after_2d':
        // 两天后
        return now.getTime() + 2 * 24 * 60 * 60 * 1000
      
      case '$after_3d':
        // 三天后
        return now.getTime() + 3 * 24 * 60 * 60 * 1000
      
      case '$after_7d':
        // 一周后
        return now.getTime() + 7 * 24 * 60 * 60 * 1000
      
      case '$after_30d':
        // 一个月后（30天）
        return now.getTime() + 30 * 24 * 60 * 60 * 1000
      
      case '$before_1d':
        // 一天前（24小时前）
        return now.getTime() - 24 * 60 * 60 * 1000
      
      case '$before_2d':
        // 两天前
        return now.getTime() - 2 * 24 * 60 * 60 * 1000
      
      case '$before_7d':
        // 一周前
        return now.getTime() - 7 * 24 * 60 * 60 * 1000
      
      case '$before_30d':
        // 一个月前（30天）
        return now.getTime() - 30 * 24 * 60 * 60 * 1000
      
      // ========== 相对时间（周） ==========
      case '$next_week':
        // 下周开始（下周一 00:00:00）
        const nextWeek = new Date(now)
        const daysUntilNextMonday = (8 - now.getDay()) % 7 || 7
        nextWeek.setDate(now.getDate() + daysUntilNextMonday)
        nextWeek.setHours(0, 0, 0, 0)
        return nextWeek.getTime()
      
      case '$last_week':
        // 上周开始（上周一 00:00:00）
        const lastWeek = new Date(now)
        const daysSinceLastMonday = (now.getDay() + 6) % 7
        lastWeek.setDate(now.getDate() - daysSinceLastMonday - 7)
        lastWeek.setHours(0, 0, 0, 0)
        return lastWeek.getTime()
      
      // ========== 相对时间（月） ==========
      case '$next_month':
        // 下个月开始（下月1号 00:00:00）
        const nextMonth = new Date(now.getFullYear(), now.getMonth() + 1, 1)
        return nextMonth.getTime()
      
      case '$last_month':
        // 上个月开始（上月1号 00:00:00）
        const lastMonth = new Date(now.getFullYear(), now.getMonth() - 1, 1)
        return lastMonth.getTime()
      
      // ========== 相对时间（年） ==========
      case '$next_year':
        // 明年开始（明年1月1日 00:00:00）
        const nextYear = new Date(now.getFullYear() + 1, 0, 1)
        return nextYear.getTime()
      
      case '$last_year':
        // 去年开始（去年1月1日 00:00:00）
        const lastYear = new Date(now.getFullYear() - 1, 0, 1)
        return lastYear.getTime()
      
      default:
        // 未知的动态变量，返回原值
        return defaultValue
    }
  }

  // 其他组件类型，暂时不支持动态变量
  return defaultValue
}

/**
 * 检查是否是动态变量
 * @param value 值
 * @returns 是否是动态变量
 */
export function isDynamicVariable(value: any): boolean {
  return typeof value === 'string' && value.startsWith('$')
}

