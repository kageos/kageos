import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it } from 'vitest'
import SliderWidget from './SliderWidget.vue'
import WidgetComponent from './WidgetComponent.vue'

const sliderField = {
  code: 'progress',
  field_name: 'Progress',
  name: '完成进度',
  data: {
    type: 'int'
  },
  widget: {
    type: 'slider',
    config: {
      default: 50,
      min: 0,
      max: 100,
      step: 5,
      unit: '%'
    }
  }
} as any

describe('SliderWidget', () => {
  it('renders a slider in edit mode with the provided config', () => {
    const wrapper = mount(SliderWidget, {
      props: {
        field: sliderField,
        fieldPath: 'progress',
        mode: 'edit',
        value: {
          raw: null,
          display: '',
          meta: {}
        }
      },
      global: {
        plugins: [createPinia()]
      }
    })

    expect(wrapper.find('.el-slider').exists()).toBe(true)
    const slider = wrapper.findComponent({ name: 'ElSlider' })
    expect(slider.exists()).toBe(true)
    expect(slider.props('step')).toBe(5)
    expect(slider.props('showStops')).toBe(true)
  })

  it('renders a progress bar in response mode', () => {
    const wrapper = mount(SliderWidget, {
      props: {
        field: sliderField,
        fieldPath: 'progress',
        mode: 'response',
        value: {
          raw: 50,
          display: '50%',
          meta: {}
        }
      },
      global: {
        plugins: [createPinia()]
      }
    })

    const progress = wrapper.findComponent({ name: 'ElProgress' })
    expect(progress.exists()).toBe(true)
    expect(wrapper.find('.el-progress').exists()).toBe(true)
    expect(progress.props('percentage')).toBe(50)
    expect(progress.props('textInside')).toBe(true)
  })

  it('renders through WidgetComponent in edit mode', () => {
    const wrapper = mount(WidgetComponent, {
      props: {
        field: sliderField,
        fieldPath: 'progress',
        mode: 'edit',
        value: {
          raw: null,
          display: '',
          meta: {}
        }
      },
      global: {
        plugins: [createPinia()]
      }
    })

    expect(wrapper.find('.slider-widget').exists()).toBe(true)
    expect(wrapper.find('.el-slider').exists()).toBe(true)
  })
})
