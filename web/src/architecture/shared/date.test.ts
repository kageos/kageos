import { afterEach, describe, expect, it, vi } from 'vitest'
import { createRelativeDateTimeShortcuts, formatDateTimeValue, type DateTimeShortcut } from './date'

function findShortcut(shortcuts: DateTimeShortcut[], text: string): DateTimeShortcut {
  const shortcut = shortcuts.find((item) => item.text === text)
  if (!shortcut) {
    throw new Error(`找不到快捷项：${text}`)
  }
  return shortcut
}

describe('date shortcuts', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('creates rich relative datetime shortcuts', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 5, 12, 10, 20, 30))

    const shortcuts = createRelativeDateTimeShortcuts()

    expect(shortcuts.map((item) => item.text)).toEqual([
      '现在',
      '10分钟后',
      '15分钟后',
      '30分钟后',
      '1小时后',
      '2小时后',
      '3小时后',
      '6小时后',
      '12小时后',
      '今天18:00',
      '明早09:00',
      '明晚18:00',
      '明天现在',
      '后天09:00',
      '一天后',
      '一周后',
      '下周一09:00',
      '昨天现在',
    ])
    expect(formatDateTimeValue(findShortcut(shortcuts, '30分钟后').value())).toBe('2026-06-12 10:50:30')
    expect(formatDateTimeValue(findShortcut(shortcuts, '2小时后').value())).toBe('2026-06-12 12:20:30')
    expect(formatDateTimeValue(findShortcut(shortcuts, '明早09:00').value())).toBe('2026-06-13 09:00:00')
    expect(formatDateTimeValue(findShortcut(shortcuts, '昨天现在').value())).toBe('2026-06-11 10:20:30')
  })

  it('calculates relative values when a shortcut is clicked', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 5, 12, 10, 0, 0))
    const shortcut = findShortcut(createRelativeDateTimeShortcuts(), '10分钟后')

    vi.setSystemTime(new Date(2026, 5, 12, 11, 0, 0))

    expect(formatDateTimeValue(shortcut.value())).toBe('2026-06-12 11:10:00')
  })
})
