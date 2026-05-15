/**
 * 动态默认值解析工具
 *
 * 支持函数调用形式：
 * - 时间函数：CURRENT_TIMESTAMP、CURRENT_DATE、DATE_ADD/DATE_SUB(..., INTERVAL n UNIT)
 * - 用户函数：Me()、MyLeader()
 * - 组织架构函数：MyDepartment()
 */

import { WidgetType } from '@/architecture/domain/types/api'
import { formatDateTimeValue } from '@/architecture/shared/date'

const DynamicFunctionName = {
  ME: 'me',
  MY_LEADER: 'myleader',
  MY_DEPARTMENT: 'mydepartment'
} as const

interface AuthUserDefaults {
  username?: string
  leader_username?: string
  department_full_path?: string
}

interface AuthStoreDefaults {
  user?: AuthUserDefaults | null
}

type DynamicDefaultValue = unknown

function readAuthUser(store: unknown): AuthUserDefaults | null {
  if (!store || typeof store !== 'object') {
    return null
  }
  const user = (store as AuthStoreDefaults).user
  return user && typeof user === 'object' ? user : null
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

function isTemporalWidget(widgetType: string): boolean {
  return widgetType === WidgetType.DATETIME
}

function toSqlKeyword(value: string): string {
  return value.trim().replace(/\s+/g, ' ').toUpperCase()
}

function resolveSqlBaseDate(value: string): Date | null {
  const keyword = toSqlKeyword(value)
  const now = new Date()

  if (keyword === 'CURRENT_TIMESTAMP' || keyword === 'CURRENT_TIMESTAMP()') {
    return now
  }

  if (keyword === 'CURRENT_DATE' || keyword === 'CURRENT_DATE()') {
    return new Date(now.getFullYear(), now.getMonth(), now.getDate())
  }

  return null
}

function parseSqlInterval(value: string): { amount: number; unit: string } | null {
  const match = value.trim().match(/^INTERVAL\s+([+-]?\d+)\s+(SECOND|MINUTE|HOUR|DAY|WEEK|MONTH|YEAR)S?$/i)
  if (!match) return null

  return {
    amount: Number(match[1]),
    unit: (match[2] ?? '').toUpperCase()
  }
}

function addSqlInterval(base: Date, interval: { amount: number; unit: string }, direction: 1 | -1): Date {
  const result = new Date(base.getTime())
  const amount = interval.amount * direction

  switch (interval.unit) {
    case 'SECOND':
      result.setSeconds(result.getSeconds() + amount)
      break
    case 'MINUTE':
      result.setMinutes(result.getMinutes() + amount)
      break
    case 'HOUR':
      result.setHours(result.getHours() + amount)
      break
    case 'DAY':
      result.setDate(result.getDate() + amount)
      break
    case 'WEEK':
      result.setDate(result.getDate() + amount * 7)
      break
    case 'MONTH':
      result.setMonth(result.getMonth() + amount)
      break
    case 'YEAR':
      result.setFullYear(result.getFullYear() + amount)
      break
  }

  return result
}

function formatTemporalDefault(date: Date): string {
  return formatDateTimeValue(date)
}

function resolveSqlStyleTemporalDefault(defaultValue: string, widgetType: string): { resolved: boolean; value: DynamicDefaultValue } {
  if (!isTemporalWidget(widgetType)) {
    return { resolved: false, value: defaultValue }
  }

  const base = resolveSqlBaseDate(defaultValue)
  if (base) {
    return { resolved: true, value: formatTemporalDefault(base) }
  }

  const funcCall = parseFunctionCall(defaultValue)
  if (!funcCall) {
    return { resolved: false, value: defaultValue }
  }

  const funcName = funcCall.name.toLowerCase()
  if (funcName !== 'date_add' && funcName !== 'date_sub') {
    return { resolved: false, value: defaultValue }
  }

  if (funcCall.args.length !== 2) {
    return { resolved: false, value: defaultValue }
  }

  const baseDate = resolveSqlBaseDate(funcCall.args[0] ?? '')
  const interval = parseSqlInterval(funcCall.args[1] ?? '')
  if (!baseDate || !interval) {
    return { resolved: false, value: defaultValue }
  }

  const direction = funcName === 'date_add' ? 1 : -1
  return {
    resolved: true,
    value: formatTemporalDefault(addSqlInterval(baseDate, interval, direction))
  }
}

export function resolveDynamicDefaultValue(
  defaultValue: DynamicDefaultValue,
  widgetType: string,
  getAuthStore?: () => unknown
): DynamicDefaultValue {
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
              const user = readAuthUser(getAuthStore())
              const username = user?.username
              if (username) {
                resolvedParts.push(username)
              }
            }
          } else if (funcName === DynamicFunctionName.MY_LEADER) {
            if (getAuthStore) {
              const user = readAuthUser(getAuthStore())
              const leaderUsername = user?.leader_username
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

  const sqlStyleDefault = resolveSqlStyleTemporalDefault(defaultValue, widgetType)
  if (sqlStyleDefault.resolved) {
    return sqlStyleDefault.value
  }

  if (!defaultValue.includes('(') || !defaultValue.includes(')')) {
    return defaultValue
  }

  const funcCall = parseFunctionCall(defaultValue)
  if (!funcCall) {
    return defaultValue
  }

  const { name } = funcCall
  const funcName = name.toLowerCase()

  if (widgetType === WidgetType.USER || widgetType === WidgetType.USERS) {
    if (funcName === DynamicFunctionName.ME) {
      if (getAuthStore) {
        return readAuthUser(getAuthStore())?.username || null
      }
      return defaultValue
    }

    if (funcName === DynamicFunctionName.MY_LEADER) {
      if (getAuthStore) {
        return readAuthUser(getAuthStore())?.leader_username || null
      }
      return defaultValue
    }
  }

  if (widgetType === WidgetType.DEPARTMENT || widgetType === WidgetType.DEPARTMENTS) {
    if (funcName === DynamicFunctionName.MY_DEPARTMENT) {
      if (getAuthStore) {
        return readAuthUser(getAuthStore())?.department_full_path || null
      }
      return defaultValue
    }
  }

  return defaultValue
}

export function isFunctionCall(value: unknown): boolean {
  if (typeof value !== 'string') return false
  return /^\w+\(.*\)$/.test(value.trim())
}
