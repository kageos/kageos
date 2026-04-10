import { formatTimestamp } from '@/utils/date'

export type ExecutionDurationTone = 'fast' | 'medium' | 'slow' | 'unknown'

export function readExecutionNumber(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim() !== '' && !Number.isNaN(Number(value))) {
    return Number(value)
  }
  return null
}

export function parseExecutionObject(raw: unknown): Record<string, any> | null {
  if (!raw) {
    return null
  }
  if (typeof raw === 'string') {
    try {
      const parsed = JSON.parse(raw)
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        return parsed as Record<string, any>
      }
    } catch {
      return null
    }
    return null
  }
  if (typeof raw === 'object' && !Array.isArray(raw)) {
    return raw as Record<string, any>
  }
  return null
}

export function formatExecutionDateTime(value: string | number | null | undefined): string {
  if (!value) {
    return '-'
  }
  if (typeof value === 'string' && !/^\d+$/.test(value)) {
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString('zh-CN')
  }
  return formatTimestamp(value)
}

export function formatExecutionRelativeTime(value: string | number | null | undefined): string {
  if (!value) {
    return '-'
  }

  const timestamp =
    typeof value === 'string'
      ? (/^\d+$/.test(value) ? Number(value) : new Date(value).getTime())
      : value

  if (Number.isNaN(timestamp)) {
    return '-'
  }

  const diff = Date.now() - timestamp
  if (diff < 0) {
    return formatExecutionDateTime(value)
  }

  const minutes = Math.floor(diff / 1000 / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (minutes < 1) {
    return '刚刚'
  }
  if (minutes < 60) {
    return `${minutes}分钟前`
  }
  if (hours < 24) {
    return `${hours}小时前`
  }
  return `${days}天前`
}

export function formatExecutionDuration(duration: number | null | undefined): string {
  if (duration === null || duration === undefined || duration < 0) {
    return '未记录'
  }
  if (duration < 1000) {
    return `${duration}ms`
  }
  if (duration < 60000) {
    return `${(duration / 1000).toFixed(duration < 10000 ? 2 : 1)}s`
  }
  const minutes = Math.floor(duration / 60000)
  const seconds = ((duration % 60000) / 1000).toFixed(1)
  return `${minutes}分${seconds}秒`
}

export function getExecutionDurationTone(duration: number | null | undefined): ExecutionDurationTone {
  if (duration === null || duration === undefined || duration < 0) {
    return 'unknown'
  }
  if (duration < 1000) {
    return 'fast'
  }
  if (duration < 3000) {
    return 'medium'
  }
  return 'slow'
}

export function getExecutionDurationTagType(duration: number | null | undefined): 'success' | 'warning' | 'danger' | 'info' {
  switch (getExecutionDurationTone(duration)) {
    case 'fast':
      return 'success'
    case 'medium':
      return 'warning'
    case 'slow':
      return 'danger'
    default:
      return 'info'
  }
}

export function getExecutionDurationTip(duration: number | null | undefined): string {
  if (duration === null || duration === undefined || duration < 0) {
    return '未记录耗时'
  }
  switch (getExecutionDurationTone(duration)) {
    case 'fast':
      return `执行较快：${formatExecutionDuration(duration)}`
    case 'medium':
      return `执行中等：${formatExecutionDuration(duration)}`
    case 'slow':
      return `执行较慢：${formatExecutionDuration(duration)}`
    default:
      return formatExecutionDuration(duration)
  }
}
