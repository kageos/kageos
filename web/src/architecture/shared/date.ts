/**
 * 日期/时间工具函数
 */

export interface DateTimeShortcut {
  text: string
  value: () => Date
}

const MINUTE_MS = 60 * 1000
const HOUR_MS = 60 * MINUTE_MS
const DAY_MS = 24 * HOUR_MS

/**
 * 创建单点日期时间快捷选项。
 * value 使用函数，确保用户点击时才基于当前时间计算。
 */
export function createRelativeDateTimeShortcuts(): DateTimeShortcut[] {
  return [
    { text: '现在', value: () => new Date() },
    { text: '10分钟后', value: () => fromNow(10 * MINUTE_MS) },
    { text: '15分钟后', value: () => fromNow(15 * MINUTE_MS) },
    { text: '30分钟后', value: () => fromNow(30 * MINUTE_MS) },
    { text: '1小时后', value: () => fromNow(HOUR_MS) },
    { text: '2小时后', value: () => fromNow(2 * HOUR_MS) },
    { text: '3小时后', value: () => fromNow(3 * HOUR_MS) },
    { text: '6小时后', value: () => fromNow(6 * HOUR_MS) },
    { text: '12小时后', value: () => fromNow(12 * HOUR_MS) },
    { text: '今天18:00', value: () => todayAt(18, 0) },
    { text: '明早09:00', value: () => daysFromNowAt(1, 9, 0) },
    { text: '明晚18:00', value: () => daysFromNowAt(1, 18, 0) },
    { text: '明天现在', value: () => shiftCalendarDays(1) },
    { text: '后天09:00', value: () => daysFromNowAt(2, 9, 0) },
    { text: '一天后', value: () => fromNow(DAY_MS) },
    { text: '一周后', value: () => shiftCalendarDays(7) },
    { text: '下周一09:00', value: () => nextWeekdayAt(1, 9, 0) },
    { text: '昨天现在', value: () => shiftCalendarDays(-1) },
  ]
}

function fromNow(offsetMilliseconds: number): Date {
  return new Date(Date.now() + offsetMilliseconds)
}

function todayAt(hours: number, minutes: number): Date {
  const date = new Date()
  date.setHours(hours, minutes, 0, 0)
  return date
}

function daysFromNowAt(days: number, hours: number, minutes: number): Date {
  const date = new Date()
  date.setDate(date.getDate() + days)
  date.setHours(hours, minutes, 0, 0)
  return date
}

function shiftCalendarDays(days: number): Date {
  const date = new Date()
  date.setDate(date.getDate() + days)
  return date
}

function nextWeekdayAt(weekday: number, hours: number, minutes: number): Date {
  const date = new Date()
  const daysUntilWeekday = (weekday + 7 - date.getDay()) % 7 || 7
  date.setDate(date.getDate() + daysUntilWeekday)
  date.setHours(hours, minutes, 0, 0)
  return date
}

/**
 * 格式化时间值
 * 
 * @param value 日期时间值
 * @param format 格式字符串，支持：
 *   - 'YYYY-MM-DD HH:mm:ss' - 完整日期时间
 *   - 'YYYY-MM-DD' - 仅日期
 *   - 其他格式按需扩展
 * @returns 格式化后的字符串
 * 
 * @example
 * formatTimestamp('2022-01-01 00:00:00') // '2022-01-01 00:00:00'
 * formatTimestamp('2022-01-01 00:00:00', 'YYYY-MM-DD') // '2022-01-01'
 */
export function formatTimestamp(value: number | string | null | undefined, format = 'YYYY-MM-DD HH:mm:ss'): string {
  if (!value) return '-'

  const date = parseDateTimeValue(value)
  if (!date) {
    console.warn('[formatTimestamp] 无效的时间:', value)
    return '-'
  }

  return formatDateObject(date, format)
}

export function formatDateTimeValue(value: number | string | Date | null | undefined, format = 'YYYY-MM-DD HH:mm:ss'): string {
  if (value === null || value === undefined || value === '') return '-'

  const date = parseDateTimeValue(value)
  if (!date) {
    console.warn('[formatDateTimeValue] 无效的时间:', value)
    return '-'
  }

  return formatDateObject(date, format)
}

export function parseDateTimeValue(value: number | string | Date): Date | null {
  if (value instanceof Date) {
    return isNaN(value.getTime()) ? null : value
  }

  if (typeof value === 'number') {
    const normalized = value > 0 && value < 9999999999 ? value * 1000 : value
    const date = new Date(normalized)
    return isNaN(date.getTime()) ? null : date
  }

  const trimmed = value.trim()
  if (!trimmed) {
    return null
  }

  if (/^-?\d+$/.test(trimmed)) {
    return parseDateTimeValue(Number(trimmed))
  }

  const timeOnlyMatch = trimmed.match(/^(\d{2}):(\d{2})(?::(\d{2}))?$/)
  if (timeOnlyMatch) {
    const now = new Date()
    const hours = Number(timeOnlyMatch[1])
    const minutes = Number(timeOnlyMatch[2])
    const seconds = Number(timeOnlyMatch[3] ?? 0)
    const date = new Date(now.getFullYear(), now.getMonth(), now.getDate(), hours, minutes, seconds, 0)
    return isNaN(date.getTime()) ? null : date
  }

  const localMatch = trimmed.match(/^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::(\d{2}))?)?$/)
  if (localMatch) {
    const year = Number(localMatch[1])
    const month = Number(localMatch[2])
    const day = Number(localMatch[3])
    const hours = Number(localMatch[4] ?? 0)
    const minutes = Number(localMatch[5] ?? 0)
    const seconds = Number(localMatch[6] ?? 0)
    const date = new Date(year, month - 1, day, hours, minutes, seconds, 0)
    return isNaN(date.getTime()) ? null : date
  }

  const date = new Date(trimmed)
  return isNaN(date.getTime()) ? null : date
}

function formatDateObject(date: Date, format: string): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  
  // 根据格式返回
  if (format === 'HH:mm' || format === 'hh:mm') {
    return `${hours}:${minutes}`
  }
  if (format === 'HH:mm:ss') {
    return `${hours}:${minutes}:${seconds}`
  }
  if (format.includes('HH:mm:ss')) {
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
  }
  if (format.includes('HH:mm')) {
    return `${year}-${month}-${day} ${hours}:${minutes}`
  }
  return `${year}-${month}-${day}`
}

/**
 * 格式化耗时（秒转换为友好的格式）
 * @param seconds 秒数
 * @returns 格式化后的字符串，如 "30秒"、"1分30秒"、"1小时5分钟"
 * 
 * @example
 * formatDuration(30) // '30秒'
 * formatDuration(90) // '1分30秒'
 * formatDuration(3665) // '1小时1分5秒'
 */
export function formatDuration(seconds: number): string {
  if (seconds < 60) {
    return `${seconds}秒`
  } else if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60)
    const secs = seconds % 60
    return secs > 0 ? `${minutes}分${secs}秒` : `${minutes}分钟`
  } else {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60
    if (minutes > 0 && secs > 0) {
      return `${hours}小时${minutes}分${secs}秒`
    } else if (minutes > 0) {
      return `${hours}小时${minutes}分钟`
    } else {
      return `${hours}小时`
    }
  }
}
