import type { TimerExecutionStatus, TimerSchedule, TimerScheduleType, TimerTaskStatus } from '@/architecture/presentation/context/api/timer'
import { getCurrentLocale, translate } from '@/architecture/shared/i18n'

export interface TimerScheduleForm {
  schedule_type: TimerScheduleType
  run_at: string
  cron_expr: string
  interval_seconds: number
  timezone: string
  max_runs: number
}

export function createDefaultTimerScheduleForm(): TimerScheduleForm {
  return {
    schedule_type: 'atime',
    run_at: defaultRunAtValue(),
    cron_expr: '0 9 * * *',
    interval_seconds: 3600,
    timezone: guessTimezone(),
    max_runs: 0,
  }
}

export function guessTimezone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai'
}

export function defaultRunAtValue(): string {
  const date = new Date(Date.now() + 60 * 60 * 1000)
  const pad = (value: number) => String(value).padStart(2, '0')
  return [
    date.getFullYear(),
    pad(date.getMonth() + 1),
    pad(date.getDate()),
  ].join('-') + ' ' + [
    pad(date.getHours()),
    pad(date.getMinutes()),
    '00',
  ].join(':')
}

export function buildTimerSchedule(form: TimerScheduleForm): TimerSchedule {
  const schedule: TimerSchedule = {
    type: form.schedule_type,
    max_runs: form.max_runs || 0,
  }

  if (form.schedule_type === 'atime') {
    schedule.run_at = toRFC3339(form.run_at)
  } else if (form.schedule_type === 'cron') {
    schedule.cron_expr = form.cron_expr.trim()
    schedule.timezone = form.timezone.trim() || guessTimezone()
  } else {
    schedule.interval_seconds = Number(form.interval_seconds || 0)
  }

  return schedule
}

export function timerScheduleToForm(schedule?: TimerSchedule): TimerScheduleForm {
  const form = createDefaultTimerScheduleForm()
  if (!schedule) return form

  form.schedule_type = schedule.type || form.schedule_type
  form.max_runs = Number(schedule.max_runs || 0)
  if (schedule.type === 'atime') {
    form.run_at = toLocalDateTimeInput(schedule.run_at) || form.run_at
  } else if (schedule.type === 'cron') {
    form.cron_expr = schedule.cron_expr || form.cron_expr
    form.timezone = schedule.timezone || form.timezone
  } else if (schedule.type === 'every') {
    form.interval_seconds = Number(schedule.interval_seconds || 0)
  }
  return form
}

export function toLocalDateTimeInput(value?: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const pad = (raw: number) => String(raw).padStart(2, '0')
  return [
    date.getFullYear(),
    pad(date.getMonth() + 1),
    pad(date.getDate()),
  ].join('-') + ' ' + [
    pad(date.getHours()),
    pad(date.getMinutes()),
    pad(date.getSeconds()),
  ].join(':')
}

export function toRFC3339(value: string): string {
  const raw = (value || '').trim()
  if (!raw) return raw
  const normalized = raw.includes('T') ? raw : raw.replace(' ', 'T')
  const parsed = new Date(normalized)
  if (Number.isNaN(parsed.getTime())) {
    return raw
  }
  return parsed.toISOString()
}

export function formatDateTime(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString(getCurrentLocale(), { hour12: false })
}

export function formatDuration(milliseconds?: number): string {
  const value = Number(milliseconds || 0)
  if (value <= 0) return '-'
  if (value < 1000) return `${value} ms`
  return `${(value / 1000).toFixed(1)} s`
}

export function scheduleLabel(schedule?: TimerSchedule): string {
  if (!schedule) return '-'
  if (schedule.type === 'atime') {
    return translate('scheduledTask.planOnce', { time: formatDateTime(schedule.run_at) })
  }
  if (schedule.type === 'cron') {
    return translate('scheduledTask.planCron', { expr: schedule.cron_expr || '-' })
  }
  return translate('scheduledTask.planEvery', { seconds: schedule.interval_seconds || 0 })
}

export function taskStatusLabel(status?: string): string {
  const labels: Record<TimerTaskStatus, string> = {
    pending: translate('scheduledTask.taskStatusPending'),
    paused: translate('scheduledTask.taskStatusPaused'),
    done: translate('scheduledTask.taskStatusDone'),
    failed: translate('scheduledTask.taskStatusFailed'),
    cancelled: translate('scheduledTask.taskStatusCancelled'),
  }
  return labels[status as TimerTaskStatus] || status || '-'
}

export function taskStatusTag(status?: string): 'primary' | 'success' | 'warning' | 'info' | 'danger' {
  if (status === 'pending') return 'primary'
  if (status === 'done') return 'success'
  if (status === 'paused') return 'warning'
  if (status === 'failed') return 'danger'
  return 'info'
}

export function executionStatusLabel(status?: string): string {
  const labels: Record<TimerExecutionStatus, string> = {
    queued: translate('scheduledTask.executionStatusQueued'),
    running: translate('scheduledTask.executionStatusRunning'),
    success: translate('scheduledTask.executionStatusSuccess'),
    failed: translate('scheduledTask.executionStatusFailed'),
    timeout: translate('scheduledTask.executionStatusTimeout'),
    cancelled: translate('scheduledTask.executionStatusCancelled'),
  }
  return labels[status as TimerExecutionStatus] || status || '-'
}

export function executionStatusTag(status?: string): 'primary' | 'success' | 'warning' | 'info' | 'danger' {
  if (status === 'queued') return 'info'
  if (status === 'running') return 'primary'
  if (status === 'success') return 'success'
  if (status === 'failed' || status === 'timeout') return 'danger'
  return 'info'
}

export function parseJSONPayload(value: string): unknown {
  const raw = value.trim()
  if (!raw) return {}
  try {
    return JSON.parse(raw)
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    throw new Error(translate('scheduledTask.invalidJSON', { message }))
  }
}

export function stringifyPayload(value: unknown): string {
  if (value === undefined || value === null) return '{}'
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return '{}'
  }
}
