/**
 * 动态默认值解析工具
 * 
 * 支持函数调用形式：
 * - 时间函数：Now()、Today()、Tomorrow()、Yesterday()
 * - 用户函数：me()
 * 
 * 函数参数格式：
 * - Now("+1h"): 一小时后
 * - Now("-2d"): 两天前
 * - Now("+3600s"): 3600秒后
 * - Now("+2"): 2小时后（默认单位是小时）
 */

/**
 * 解析时间偏移参数
 * @param offset 偏移字符串，如 "+1h", "-2d", "+3600s", "+2", "24h", "2d", "-3600s"
 * @returns 偏移的毫秒数
 * 
 * 支持格式：
 * - 带符号："+1h", "-2d", "+3600s"
 * - 不带符号（默认为正）："24h", "2d", "3600s"
 * - 只写数字（默认单位是小时）："+2", "-2", "2"
 */
function parseTimeOffset(offset: string): number {
  if (!offset) return 0
  
  // 移除空格和引号
  offset = offset.trim().replace(/^["']|["']$/g, '')
  
  // 提取符号和数值
  const sign = offset.startsWith('-') ? -1 : 1
  const valueStr = offset.replace(/^[+-]/, '')
  
  // 如果只写数字，默认单位是小时
  if (/^\d+$/.test(valueStr)) {
    const hours = parseInt(valueStr, 10)
    return sign * hours * 60 * 60 * 1000
  }
  
  // 解析带单位的字符串
  const match = valueStr.match(/^(\d+)([smhdwy])?$/i)
  if (!match) return 0
  
  const [, numStr, unit = 'h'] = match
  const num = parseInt(numStr, 10)
  
  switch (unit.toLowerCase()) {
    case 's': // 秒
      return sign * num * 1000
    case 'm': // 分钟
      return sign * num * 60 * 1000
    case 'h': // 小时
      return sign * num * 60 * 60 * 1000
    case 'd': // 天
      return sign * num * 24 * 60 * 60 * 1000
    case 'w': // 周
      return sign * num * 7 * 24 * 60 * 60 * 1000
    case 'y': // 年（按365天计算）
      return sign * num * 365 * 24 * 60 * 60 * 1000
    default:
      return 0
  }
}

/**
 * 解析函数调用
 * @param funcCall 函数调用字符串，如 "Now()", "Now(\"+1h\")", "me()"
 * @returns 解析后的函数名和参数
 */
function parseFunctionCall(funcCall: string): { name: string; args: string[] } | null {
  // 匹配函数调用：函数名(参数1,参数2,...)
  const match = funcCall.match(/^(\w+)\((.*)\)$/)
  if (!match) return null
  
  const [, name, argsStr] = match
  const args: string[] = []
  
  // 解析参数（支持引号字符串，也支持不带引号的参数）
  if (argsStr.trim()) {
    let current = ''
    let inQuotes = false
    let quoteChar = ''
    
    for (let i = 0; i < argsStr.length; i++) {
      const char = argsStr[i]
      
      if ((char === '"' || char === "'") && (i === 0 || argsStr[i - 1] !== '\\')) {
        if (!inQuotes) {
          inQuotes = true
          quoteChar = char
        } else if (char === quoteChar) {
          inQuotes = false
          quoteChar = ''
        } else {
          current += char
        }
      } else if (char === ',' && !inQuotes) {
        if (current.trim()) {
          args.push(current.trim())
        }
        current = ''
      } else {
        current += char
      }
    }
    
    if (current.trim()) {
      args.push(current.trim())
    }
  }
  
  return { name, args }
}

/**
 * 解析动态默认值
 * @param defaultValue 默认值（可能是函数调用，如 Now()、me() 等）
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

  // 检查是否是函数调用（包含括号）
  if (!defaultValue.includes('(') || !defaultValue.includes(')')) {
    return defaultValue
  }

  // 解析函数调用
  const funcCall = parseFunctionCall(defaultValue)
  if (!funcCall) {
    return defaultValue
  }

  const { name, args } = funcCall
  const funcName = name.toLowerCase()

  // 时间戳组件：支持时间函数
  if (widgetType === 'timestamp') {
    const now = new Date()
    
    switch (funcName) {
      case 'now': {
        // Now() - 当前时间
        // Now("+1h") - 一小时后
        // Now("-2d") - 两天前
        if (args.length === 0) {
          return now.getTime()
        }
        const offset = parseTimeOffset(args[0])
        return now.getTime() + offset
      }
      
      case 'today': {
        // Today() - 今天 00:00:00
        const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
        if (args.length === 0) {
          return today.getTime()
        }
        // Today("+1d") - 明天 00:00:00
        const offset = parseTimeOffset(args[0])
        return today.getTime() + offset
      }
      
      case 'tomorrow': {
        // Tomorrow() - 明天 00:00:00
        const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1)
        return tomorrow.getTime()
      }
      
      case 'yesterday': {
        // Yesterday() - 昨天 00:00:00
        const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1)
        return yesterday.getTime()
      }
      
      default:
        // 未知函数，返回原值
        return defaultValue
    }
  }

  // 用户选择器：支持用户函数
  if (widgetType === 'user' || widgetType === 'users') {
    if (funcName === 'me') {
      // me() - 当前登录用户
      if (getAuthStore) {
        const authStore = getAuthStore()
        return authStore?.user?.username || null
      }
      // 如果没有提供 getAuthStore，返回原值（让组件自己处理）
      return defaultValue
    }
  }

  // 其他组件类型，暂时不支持函数调用
  return defaultValue
}

/**
 * 检查是否是函数调用
 * @param value 值
 * @returns 是否是函数调用
 */
export function isFunctionCall(value: any): boolean {
  if (typeof value !== 'string') return false
  // 检查是否包含函数调用模式：函数名(...)
  return /^\w+\(.*\)$/.test(value.trim())
}
