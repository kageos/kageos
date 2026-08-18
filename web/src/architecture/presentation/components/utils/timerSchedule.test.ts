import { describe, expect, it } from 'vitest'
import { buildTimerSchedule, createDefaultManualTimerScheduleForm, createDefaultTimerScheduleForm, timerScheduleToForm } from './timerSchedule'

describe('timerSchedule manual tasks', () => {
  it('defaults new tasks to manual operation without a synthetic run time', () => {
    const form = createDefaultManualTimerScheduleForm()

    expect(form.schedule_type).toBe('manual')
    expect(buildTimerSchedule(form)).toEqual({ type: 'manual', max_runs: 0 })
  })

  it('keeps function schedules on their existing one-time default', () => {
    expect(createDefaultTimerScheduleForm().schedule_type).toBe('atime')
  })

  it('loads an existing manual task into the editor', () => {
    expect(timerScheduleToForm({ type: 'manual' }).schedule_type).toBe('manual')
  })
})
