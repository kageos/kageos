/**
 * 日期时间快捷选择工具函数
 * 参考旧版本实现，提供丰富的快捷选择选项
 */

/**
 * 获取日期时间快捷选择选项
 * @param type 日期时间类型：'date' | 'datetime' | 'daterange' | 'datetimerange'
 * @returns 快捷选项数组
 */
export function getDateTimeShortcuts(type: string = 'datetime'): Array<{ text: string; value: Date | (() => Date) | (() => [Date, Date]) }> {
  // 范围选择（daterange, datetimerange）
  if (type === 'daterange' || type === 'datetimerange') {
    return [
      {
        text: '今天',
        value: () => {
          const today = new Date()
          const start = new Date(today.getFullYear(), today.getMonth(), today.getDate(), 0, 0, 0)
          const end = new Date(today.getFullYear(), today.getMonth(), today.getDate(), 23, 59, 59)
          return [start, end]
        }
      },
      {
        text: '昨天',
        value: () => {
          const yesterday = new Date()
          yesterday.setDate(yesterday.getDate() - 1)
          const start = new Date(yesterday.getFullYear(), yesterday.getMonth(), yesterday.getDate(), 0, 0, 0)
          const end = new Date(yesterday.getFullYear(), yesterday.getMonth(), yesterday.getDate(), 23, 59, 59)
          return [start, end]
        }
      },
      {
        text: '最近3天',
        value: () => {
          const end = new Date()
          const start = new Date()
          start.setDate(start.getDate() - 2)
          start.setHours(0, 0, 0, 0)
          return [start, end]
        }
      },
      {
        text: '最近7天',
        value: () => {
          const end = new Date()
          const start = new Date()
          start.setDate(start.getDate() - 6)
          start.setHours(0, 0, 0, 0)
          return [start, end]
        }
      },
      {
        text: '最近15天',
        value: () => {
          const end = new Date()
          const start = new Date()
          start.setDate(start.getDate() - 14)
          start.setHours(0, 0, 0, 0)
          return [start, end]
        }
      },
      {
        text: '最近30天',
        value: () => {
          const end = new Date()
          const start = new Date()
          start.setDate(start.getDate() - 29)
          start.setHours(0, 0, 0, 0)
          return [start, end]
        }
      },
      {
        text: '本周',
        value: () => {
          const now = new Date()
          const day = now.getDay()
          const start = new Date(now)
          start.setDate(now.getDate() - (day === 0 ? 6 : day - 1))
          start.setHours(0, 0, 0, 0)
          const end = new Date(start)
          end.setDate(start.getDate() + 6)
          end.setHours(23, 59, 59, 999)
          return [start, end]
        }
      },
      {
        text: '上周',
        value: () => {
          const now = new Date()
          const day = now.getDay()
          const thisWeekStart = new Date(now)
          thisWeekStart.setDate(now.getDate() - (day === 0 ? 6 : day - 1))
          const start = new Date(thisWeekStart)
          start.setDate(thisWeekStart.getDate() - 7)
          start.setHours(0, 0, 0, 0)
          const end = new Date(start)
          end.setDate(start.getDate() + 6)
          end.setHours(23, 59, 59, 999)
          return [start, end]
        }
      },
      {
        text: '本月',
        value: () => {
          const now = new Date()
          const start = new Date(now.getFullYear(), now.getMonth(), 1, 0, 0, 0)
          const end = new Date(now.getFullYear(), now.getMonth() + 1, 0, 23, 59, 59)
          return [start, end]
        }
      },
      {
        text: '上月',
        value: () => {
          const now = new Date()
          const start = new Date(now.getFullYear(), now.getMonth() - 1, 1, 0, 0, 0)
          const end = new Date(now.getFullYear(), now.getMonth(), 0, 23, 59, 59)
          return [start, end]
        }
      }
    ]
  }
  
  // 单个日期时间选择
  return [
    {
      text: '现在',
      value: new Date()
    },
    {
      text: '昨天此刻',
      value: () => {
        const date = new Date()
        date.setDate(date.getDate() - 1)
        return date
      }
    },
    {
      text: '明天此刻',
      value: () => {
        const date = new Date()
        date.setDate(date.getDate() + 1)
        return date
      }
    },
    {
      text: '今天 00:00',
      value: () => {
        const today = new Date()
        today.setHours(0, 0, 0, 0)
        return today
      }
    },
    {
      text: '今天 23:59',
      value: () => {
        const today = new Date()
        today.setHours(23, 59, 59, 999)
        return today
      }
    },
    {
      text: '昨天 00:00',
      value: () => {
        const yesterday = new Date()
        yesterday.setDate(yesterday.getDate() - 1)
        yesterday.setHours(0, 0, 0, 0)
        return yesterday
      }
    },
    {
      text: '昨天 23:59',
      value: () => {
        const yesterday = new Date()
        yesterday.setDate(yesterday.getDate() - 1)
        yesterday.setHours(23, 59, 59, 999)
        return yesterday
      }
    },
    {
      text: '明天 00:00',
      value: () => {
        const tomorrow = new Date()
        tomorrow.setDate(tomorrow.getDate() + 1)
        tomorrow.setHours(0, 0, 0, 0)
        return tomorrow
      }
    },
    {
      text: '下周此刻',
      value: () => {
        const date = new Date()
        date.setDate(date.getDate() + 7)
        return date
      }
    },
    {
      text: '下月此刻',
      value: () => {
        const date = new Date()
        date.setMonth(date.getMonth() + 1)
        return date
      }
    },
    {
      text: '下季度此刻',
      value: () => {
        const date = new Date()
        const quarter = Math.floor(date.getMonth() / 3)
        const nextQuarter = quarter === 3 ? 0 : quarter + 1
        if (nextQuarter === 0) {
          date.setFullYear(date.getFullYear() + 1)
        }
        date.setMonth(nextQuarter * 3)
        return date
      }
    },
    {
      text: '明年此刻',
      value: () => {
        const date = new Date()
        date.setFullYear(date.getFullYear() + 1)
        return date
      }
    }
  ]
}

