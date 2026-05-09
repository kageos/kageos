import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import RateWidget from './RateWidget.vue'

const rateField = {
  code: 'score',
  field_name: 'Score',
  name: '评分',
  data: {
    type: 'float'
  },
  widget: {
    type: 'rate',
    config: {
      max: 5,
      allow_half: true,
      texts: ['很差', '差', '一般', '好', '很好']
    }
  }
} as any

describe('RateWidget', () => {
  it('uses a larger yellow rate control in form edit mode', () => {
    const wrapper = mount(RateWidget, {
      props: {
        field: rateField,
        fieldPath: 'score',
        mode: 'edit',
        value: {
          raw: 3.5,
          display: '',
          meta: {}
        }
      },
      global: {
        plugins: [createPinia()]
      }
    })

    const rate = wrapper.findComponent({ name: 'ElRate' })

    expect(rate.exists()).toBe(true)
    expect(rate.classes()).toContain('rate-widget-control--edit')
    expect(rate.props('size')).toBe('large')
    expect(rate.props('colors')).toEqual(['#f7ba2a', '#f7ba2a', '#f7ba2a'])
    expect(rate.props('textColor')).toBe('#995c00')
  })

  it('keeps table-cell rate compact while using the same yellow palette', () => {
    const wrapper = mount(RateWidget, {
      props: {
        field: rateField,
        fieldPath: 'score',
        mode: 'table-cell',
        value: {
          raw: 4,
          display: '',
          meta: {}
        }
      },
      global: {
        plugins: [createPinia()]
      }
    })

    const rate = wrapper.findComponent({ name: 'ElRate' })

    expect(rate.exists()).toBe(true)
    expect(rate.classes()).toContain('rate-widget-control--table')
    expect(rate.props('colors')).toEqual(['#f7ba2a', '#f7ba2a', '#f7ba2a'])
    expect(rate.props('showScore')).toBe(true)
  })

  it('uses the larger yellow display treatment in response mode', () => {
    const wrapper = mount(RateWidget, {
      props: {
        field: rateField,
        fieldPath: 'score',
        mode: 'response',
        value: {
          raw: 4.5,
          display: '',
          meta: {}
        }
      },
      global: {
        plugins: [createPinia()]
      }
    })

    const rate = wrapper.findComponent({ name: 'ElRate' })

    expect(rate.exists()).toBe(true)
    expect(rate.classes()).toContain('rate-widget-control--response')
    expect(rate.props('size')).toBe('large')
    expect(rate.props('colors')).toEqual(['#f7ba2a', '#f7ba2a', '#f7ba2a'])
    expect(rate.props('showScore')).toBe(true)
    expect(rate.props('textColor')).toBe('#995c00')
  })

  it('uses the larger yellow display treatment in detail mode', () => {
    const wrapper = mount(RateWidget, {
      props: {
        field: rateField,
        fieldPath: 'score',
        mode: 'detail',
        value: {
          raw: 5,
          display: '',
          meta: {}
        }
      },
      global: {
        plugins: [createPinia()]
      }
    })

    const rate = wrapper.findComponent({ name: 'ElRate' })

    expect(rate.exists()).toBe(true)
    expect(rate.classes()).toContain('rate-widget-control--detail')
    expect(rate.props('size')).toBe('large')
    expect(rate.props('colors')).toEqual(['#f7ba2a', '#f7ba2a', '#f7ba2a'])
    expect(rate.props('showScore')).toBe(true)
    expect(rate.props('textColor')).toBe('#995c00')
  })
})
