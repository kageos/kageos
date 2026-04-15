import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import WorkspaceFormPage from './WorkspaceFormPage.vue'

const submitFormMock = vi.fn(async () => true)

const FormViewStub = defineComponent({
  name: 'FormView',
  props: {
    showSubmitButton: {
      type: Boolean,
      default: true
    }
  },
  setup(props, { expose }) {
    expose({
      submitForm: submitFormMock
    })

    return () =>
      h('div', {
        class: 'form-view-stub',
        'data-show-submit-button': String(props.showSubmitButton)
      })
  }
})

const ElButtonStub = defineComponent({
  name: 'ElButton',
  emits: ['click'],
  setup(_props, { emit, slots, attrs }) {
    return () =>
      h(
        'button',
        {
          ...attrs,
          onClick: () => emit('click')
        },
        slots.default?.()
      )
  }
})

describe('WorkspaceFormPage', () => {
  it('submits through the exposed FormView method from the footer button', async () => {
    submitFormMock.mockClear()

    const wrapper = mount(WorkspaceFormPage, {
      props: {
        title: '新增数据',
        pageKey: 'form-create-1',
        functionDetail: {
          id: 1,
          template_type: 'form'
        } as any
      },
      global: {
        stubs: {
          FormView: FormViewStub,
          ElButton: ElButtonStub
        }
      }
    })

    expect(wrapper.find('.form-view-stub').attributes('data-show-submit-button')).toBe('false')

    const submitButton = wrapper.findAll('button').find((button) => button.text() === '提交')
    expect(submitButton?.exists()).toBe(true)

    await submitButton?.trigger('click')
    await flushPromises()

    expect(submitFormMock).toHaveBeenCalledTimes(1)
  })
})
