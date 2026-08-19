import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MiniWorkstationPendingActionBar from './MiniWorkstationPendingActionBar.vue'

const ButtonStub = {
  props: ['loading', 'type', 'size', 'disabled'],
  emits: ['click'],
  template: '<button type="button" :disabled="disabled" :data-loading="loading ? \'true\' : undefined" @click="$emit(\'click\')"><slot /></button>',
}

function mountBar(props: Record<string, unknown> = {}) {
  return mount(MiniWorkstationPendingActionBar, {
    props: {
      interaction: {
        id: 'prd-1',
        card_type: 'prd_confirmation',
        artifact_kind: 'agent_app_prd',
        status: 'pending_confirmation',
        blocking: true,
        title: 'PRD 等待确认',
        description: '请确认需求是否完整',
        view_text: '查看 PRD',
        revise_text: '修改 PRD',
        cancel_text: '取消 PRD',
        confirm_text: '确认 PRD',
      },
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
  it('renders interaction actions and emits commands', async () => {
    const wrapper = mountBar()
    const buttons = wrapper.findAll('button')

    expect(wrapper.attributes('data-testid')).toBe('mini-interaction-gate-prd_confirmation')
    expect(wrapper.text()).toContain('PRD 等待确认')
    expect(wrapper.text()).toContain('需要处理')

    await buttons[0]!.trigger('click')
    await buttons[2]!.trigger('click')
    await buttons[3]!.trigger('click')

    expect(wrapper.emitted('view')).toHaveLength(1)
    expect(wrapper.emitted('cancel')).toHaveLength(1)
    expect(wrapper.emitted('confirm')).toHaveLength(1)
  })

  it('submits revision text from the gate', async () => {
    const wrapper = mountBar()

    await wrapper.findAll('button')[1]!.trigger('click')
    await wrapper.find('[data-testid="mini-interaction-revision-input"]').setValue('增加审批状态')
    await wrapper.findAll('button').find(button => button.text().includes('提交修改'))!.trigger('click')

    expect(wrapper.emitted('revise')?.[0]?.[0]).toEqual({ text: '增加审批状态' })
  })

  it('does not render retired build repair cards', () => {
    const wrapper = mountBar({
      interaction: {
        id: 'build-1',
        card_type: 'build_repair',
        status: 'pending_build_repair',
        blocking: true,
        title: '构建等待修复',
        description: '交接构建修复工程师',
        view_text: '查看诊断',
        revise_text: '继续修改',
        cancel_text: '暂不修复',
        confirm_text: '交接修复',
      },
      sending: true,
    })

    expect(wrapper.find('[data-testid="mini-interaction-gate-build_repair"]').exists()).toBe(false)
    expect(wrapper.text()).toBe('')
  })

  it('keeps historical cards read-only for audit', async () => {
    const wrapper = mountBar({ readonly: true })

    expect(wrapper.text()).toContain('已记录')
    expect(wrapper.findAll('button')).toHaveLength(0)
    expect(wrapper.find('[data-testid="mini-interaction-revision-input"]').exists()).toBe(false)
  })
})
