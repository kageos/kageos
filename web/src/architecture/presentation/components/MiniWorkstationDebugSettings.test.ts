import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import type { DebugToolStep } from '../composables/useMiniWorkstationDebugCopy'
import MiniWorkstationDebugSettings from './MiniWorkstationDebugSettings.vue'

const debugStep: DebugToolStep = {
  key: 'step-1',
  index: 1,
  name: 'workspace.apply_patch',
  status: 'ok',
  statusLabel: '成功',
  statusClass: 'ok',
  argumentsPreview: '{"file":"MiniWorkstation.vue"}',
  outputPreview: 'updated',
  errorPreview: '',
  copyText: 'copy text',
}

function mountSettings(props: Record<string, unknown> = {}) {
  return mount(MiniWorkstationDebugSettings, {
    props: {
      debugToolSteps: [debugStep],
      debugSuccessCount: 1,
      debugErrorCount: 0,
      ...props,
    },
  })
}

describe('MiniWorkstationDebugSettings', () => {
  it('emits copy conversation mode from copy buttons', async () => {
    const wrapper = mountSettings()
    const buttons = wrapper.findAll('.mini-settings-copy-grid button')

    await buttons[0]!.trigger('click')
    await buttons[3]!.trigger('click')

    expect(wrapper.emitted('copyConversation')?.[0]).toEqual(['all'])
    expect(wrapper.emitted('copyConversation')?.[1]).toEqual(['error-tools'])
  })

  it('renders tool stats and emits summary copy', async () => {
    const wrapper = mountSettings()

    expect(wrapper.text()).toContain('1 步')
    expect(wrapper.text()).toContain('1 成功')
    expect(wrapper.text()).toContain('workspace.apply_patch')

    await wrapper.find('.mini-debug-copy-btn').trigger('click')

    expect(wrapper.emitted('copyToolSummary')).toHaveLength(1)
  })

  it('disables summary copy when there are no tool steps', () => {
    const wrapper = mountSettings({
      debugToolSteps: [],
      debugSuccessCount: 0,
      debugErrorCount: 0,
    })

    expect(wrapper.find('.mini-debug-empty').text()).toBe('暂无工具调用记录')
    expect(wrapper.find('.mini-debug-copy-btn').attributes('disabled')).toBeDefined()
  })
})
