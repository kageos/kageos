import { describe, expect, it } from 'vitest'
import { WidgetType } from '@/architecture/runtime/constants/widget'
import type { FieldConfig } from '@/architecture/domain/types/field'
import { getWidgetDefaultValue } from './defaultValue'
import { resolveDynamicDefaultValue } from './dynamicDefaultValue'

describe('widget render_default', () => {
  it('uses render_default', () => {
    const field = {
      code: 'priority',
      name: '优先级',
      data: { type: 'string' },
      widget: {
        type: WidgetType.SELECT,
        config: {
          options: ['低', '中', '高'],
          render_default: '中'
        }
      }
    } as FieldConfig

    expect(getWidgetDefaultValue(field).raw).toBe('中')
  })

  it('resolves whitelisted SQL-style datetime render defaults', () => {
    expect(resolveDynamicDefaultValue('CURRENT_DATE', WidgetType.DATETIME)).toMatch(
      /^\d{4}-\d{2}-\d{2} 00:00:00$/
    )
    expect(
      resolveDynamicDefaultValue('DATE_ADD(CURRENT_TIMESTAMP, INTERVAL 1 HOUR)', WidgetType.DATETIME)
    ).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/)
    expect(resolveDynamicDefaultValue("DATE_FORMAT(created_at, '%Y')", WidgetType.DATETIME)).toBe(
      "DATE_FORMAT(created_at, '%Y')"
    )
  })
})
