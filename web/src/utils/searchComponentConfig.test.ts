import { describe, expect, it } from 'vitest'
import { createSearchComponentConfig } from './searchComponentConfig'
import { SearchComponent, SearchType } from '@/core/constants/search'
import { WidgetType } from '@/core/constants/widget'

function createRadioField() {
  return {
    code: 'is_urgent',
    name: '是否紧急',
    widget: {
      type: WidgetType.RADIO,
      config: {
        options: ['是', '否', '不确定'],
        options_colors: ['danger', 'success', 'warning']
      }
    }
  } as any
}

describe('createSearchComponentConfig', () => {
  it('builds single-select search config for radio eq search', () => {
    const config = createSearchComponentConfig(createRadioField(), SearchType.EQ)

    expect(config.component).toBe(SearchComponent.EL_SELECT)
    expect(config.props?.multiple).toBeUndefined()
    expect(config.props?.options).toEqual([
      { label: '是', value: '是' },
      { label: '否', value: '否' },
      { label: '不确定', value: '不确定' }
    ])
  })

  it('builds multi-select search config for radio in search', () => {
    const config = createSearchComponentConfig(createRadioField(), SearchType.IN)

    expect(config.component).toBe(SearchComponent.EL_SELECT)
    expect(config.props?.multiple).toBe(true)
    expect(config.props?.options).toEqual([
      { label: '是', value: '是' },
      { label: '否', value: '否' },
      { label: '不确定', value: '不确定' }
    ])
  })
})
