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
    expect(wrapper.get('img').attributes('src')).toContain('service-icon')
  })

  it('changes the employee artwork as the task state changes', async () => {
    const wrapper = mount(AgentEmployeeMascot, {
      props: {
        variant: 'employee',
        state: 'paused',
      },
    })

    const pausedSource = wrapper.get('img').attributes('src')
    expect(pausedSource).toContain('employee-paused')

    await wrapper.setProps({ state: 'failed' })

    expect(wrapper.attributes('data-agent-state')).toBe('failed')
    expect(wrapper.get('img').attributes('src')).toContain('employee-failed')
    expect(wrapper.get('img').attributes('src')).not.toBe(pausedSource)
  })
})
