/**
 * 动态默认值解析工具
 *
 * 支持函数调用形式：
 * - 时间函数：Now()、Today()、Tomorrow()、Yesterday()
 * - 用户函数：Me()、MyLeader()
 * - 组织架构函数：MyDepartment()
 *
 * 函数参数格式（参数不需要引号）：
 * - Now(+1h): 一小时后
 * - Now(-2d): 两天前
 * - Now(+3600s): 3600秒后
 * - Now(+2): 2小时后（默认单位是小时）
 * - Now(24h): 24小时后（不带+号，默认为正）
 * - Now(-2): 2小时前
 */

import { WidgetType, DynamicFunctionName } from '@/core/constants/widget'

function parseTimeOffset(offset: string): number {
  if (!offset) return 0

  offset = offset.trim().replace(/^["']|["']$/g, '')

  const sign = offset.startsWith('-') ? -1 : 1
  const valueStr = offset.replace(/^[+-]/, '')

  if (/^\d+$/.test(valueStr)) {
    const hours = parseInt(valueStr, 10)
    return sign * hours * 60 * 60 * 1000
  }

  const match = valueStr.match(/^(\d+)([smhdwy])?$/i)
  if (!match) return 0

  const numStr = match[1] ?? '0'
  const unit = match[2] ?? 'h'
  const num = parseInt(numStr, 10)

  switch (unit.toLowerCase()) {
    case 's':
      return sign * num * 1000
    case 'm':
      return sign * num * 60 * 1000
    case 'h':
      return sign * num * 60 * 60 * 1000
    case 'd':
      return sign * num * 24 * 60 * 60 * 1000
    case 'w':
      return sign * num * 7 * 24 * 60 * 60 * 1000
    case 'y':
      return sign * num * 365 * 24 * 60 * 60 * 1000
    default:
      return 0
  }
}

function parseFunctionCall(funcCall: string): { name: string; args: string[] } | null {
  const match = funcCall.match(/^(\w+)\((.*)\)$/)
  if (!match) return null

  const name = match[1] ?? ''
  const argsStr = match[2] ?? ''
  const args: string[] = []

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

export function resolveDynamicDefaultValue(
  defaultValue: any,
  widgetType: string,
  getAuthStore?: () => any
): any {
  if (typeof defaultValue !== 'string') {
    return defaultValue
  }

  if (widgetType === WidgetType.USERS && defaultValue.includes(',')) {
    const parts = defaultValue.split(',').map(s => s.trim())
    const resolvedParts: string[] = []

    for (const part of parts) {
      if (part.includes('(') && part.includes(')')) {
        const funcCall = parseFunctionCall(part)
        if (funcCall) {
          const { name } = funcCall
          const funcName = name.toLowerCase()

          if (funcName === DynamicFunctionName.ME) {
            if (getAuthStore) {
              const authStore = getAuthStore()
              const username = authStore?.user?.username
              if (username) {
                resolvedParts.push(username)
              }
            }
          } else if (funcName === DynamicFunctionName.MY_LEADER) {
            if (getAuthStore) {
              const authStore = getAuthStore()
              const leaderUsername = authStore?.user?.leader_username
              if (leaderUsername) {
                resolvedParts.push(leaderUsername)
              }
            }
          } else {
            resolvedParts.push(part)
          }
        } else {
          resolvedParts.push(part)
        }
      } else {
        resolvedParts.push(part)
      }
    }

    return resolvedParts.filter(Boolean).join(',')
  }

  if (!defaultValue.includes('(') || !defaultValue.includes(')')) {
    return defaultValue
  }

  const funcCall = parseFunctionCall(defaultValue)
  if (!funcCall) {
    return defaultValue
  }

  const { name, args } = funcCall
  const funcName = name.toLowerCase()

  if (widgetType === WidgetType.TIMESTAMP) {
    const now = new Date()

    switch (funcName) {
      case DynamicFunctionName.NOW: {
        if (args.length === 0) {
          return now.getTime()
        }
        const offset = parseTimeOffset(args[0] ?? '')
        return now.getTime() + offset
      }
      case DynamicFunctionName.TODAY: {
        const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
        if (args.length === 0) {
          return today.getTime()
        }
        const offset = parseTimeOffset(args[0] ?? '')
        return today.getTime() + offset
      }
      case DynamicFunctionName.TOMORROW: {
        const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1)
        return tomorrow.getTime()
      }
      case DynamicFunctionName.YESTERDAY: {
        const yesterday = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1)
        return yesterday.getTime()
      }
      default:
        return defaultValue
    }
  }

  if (widgetType === WidgetType.USER || widgetType === WidgetType.USERS) {
    if (funcName === DynamicFunctionName.ME) {
      if (getAuthStore) {
        const authStore = getAuthStore()
        return authStore?.user?.username || null
      }
      return defaultValue
    }

    if (funcName === DynamicFunctionName.MY_LEADER) {
      if (getAuthStore) {
        const authStore = getAuthStore()
        return authStore?.user?.leader_username || null
      }
      return defaultValue
    }
  }

  if (widgetType === WidgetType.DEPARTMENT || widgetType === WidgetType.DEPARTMENTS) {
    if (funcName === DynamicFunctionName.MY_DEPARTMENT) {
      if (getAuthStore) {
        const authStore = getAuthStore()
        return authStore?.user?.department_full_path || null
      }
      return defaultValue
    }
  }

  return defaultValue
}

export function isFunctionCall(value: any): boolean {
  if (typeof value !== 'string') return false
  return /^\w+\(.*\)$/.test(value.trim())
}
