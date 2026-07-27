import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AgentEmployeeMascot from './AgentEmployeeMascot.vue'

describe('AgentEmployeeMascot', () => {
  it('renders a compact brand mark with a state users can identify', () => {
    const wrapper = mount(AgentEmployeeMascot, {
      props: {
        variant: 'mark',
        state: 'working',
        label: '巡检员工正在处理',
      },
    })

    expect(wrapper.attributes('data-agent-variant')).toBe('mark')
    expect(wrapper.attributes('data-agent-state')).toBe('working')
    expect(wrapper.attributes('aria-label')).toBe('巡检员工正在处理')
    expect(wrapper.find('.agent-data-sparks').exists()).toBe(true)
  })

  it('changes the employee pose as the task state changes', async () => {
    const wrapper = mount(AgentEmployeeMascot, {
      props: {
        variant: 'employee',
        state: 'paused',
      },
    })

    expect(wrapper.find('.agent-eyes-closed').exists()).toBe(true)
    expect(wrapper.find('.agent-sleep-sign').exists()).toBe(true)

    await wrapper.setProps({ state: 'failed' })

    expect(wrapper.attributes('data-agent-state')).toBe('failed')
    expect(wrapper.find('.agent-alert-sign').exists()).toBe(true)
    expect(wrapper.find('.agent-brows').exists()).toBe(true)
  })
})
