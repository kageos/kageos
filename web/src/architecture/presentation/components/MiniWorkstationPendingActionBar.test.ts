import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MiniWorkstationPendingActionBar from './MiniWorkstationPendingActionBar.vue'

const ButtonStub = {
  props: ['loading', 'type', 'size'],
  emits: ['click'],
  template: '<button type="button" :data-loading="loading ? \'true\' : undefined" @click="$emit(\'click\')"><slot /></button>',
}

function mountBar(props: Record<string, unknown> = {}) {
  return mount(MiniWorkstationPendingActionBar, {
    props: {
      variant: 'prd',
      helpText: '请确认需求是否完整',
      sending: false,
      ...props,
    },
    global: {
      stubs: {
        ElButton: ButtonStub,
      },
    },
  })
}

describe('MiniWorkstationPendingActionBar', () => {
  it('renders PRD actions and emits commands', async () => {
    const wrapper = mountBar()
    const buttons = wrapper.findAll('button')

    expect(wrapper.attributes('data-testid')).toBe('mini-prd-confirm-bar')
    expect(wrapper.text()).toContain('PRD 等待确认')
    expect(wrapper.text()).toContain('确认 PRD')

    await buttons[0]!.trigger('click')
    await buttons[1]!.trigger('click')
    await buttons[2]!.trigger('click')
    await buttons[3]!.trigger('click')

    expect(wrapper.emitted('view')).toHaveLength(1)
    expect(wrapper.emitted('revise')).toHaveLength(1)
    expect(wrapper.emitted('cancel')).toHaveLength(1)
    expect(wrapper.emitted('confirm')).toHaveLength(1)
  })

  it('renders test handoff actions', () => {
    const wrapper = mountBar({
      variant: 'test',
      helpText: '进入测试工程师验证',
      sending: true,
    })

    expect(wrapper.attributes('data-testid')).toBe('mini-test-confirm-bar')
    expect(wrapper.text()).toContain('应用等待测试')
    expect(wrapper.text()).toContain('查看构建结果')
    expect(wrapper.text()).toContain('暂不测试')
    expect(wrapper.findAll('button')[3]!.attributes('data-loading')).toBe('true')
  })
})
