import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import MiniWorkstationComposer from './MiniWorkstationComposer.vue'

const ButtonStub = {
  props: ['disabled', 'loading'],
  emits: ['click'],
  template: '<button type="button" :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
}

const UploadStub = {
  props: ['disabled'],
  template: '<div data-testid="upload" :data-disabled="disabled ? \'true\' : \'false\'"><slot /></div>',
}

function mountComposer(props: Record<string, unknown> = {}) {
  return mount(MiniWorkstationComposer, {
    props: {
      fullCodePath: '/system/ticket_sys/v1',
      dirName: 'v1',
      attachedFiles: [],
      uploading: false,
      inputText: '',
      sending: false,
      stopping: false,
      selectedLLMConfigId: 0,
      llmList: [],
      llmLoading: false,
      queuedCount: 0,
      registerInputRef: vi.fn(),
      onLLMSelectVisibleChange: vi.fn(),
      onFileChange: vi.fn(),
      removeFile: vi.fn(),
      onInputEnter: vi.fn(),
      ...props,
    },
    global: {
      stubs: {
        ElButton: ButtonStub,
        ElUpload: UploadStub,
        ElSelect: { template: '<div><slot /></div>' },
        ElOption: { template: '<span />' },
        ElTooltip: { template: '<span><slot /></span>' },
        ElIcon: { template: '<span><slot /></span>' },
      },
    },
  })
}

describe('MiniWorkstationComposer', () => {
  it('keeps the input visible but blocked during pending interactions', async () => {
    const wrapper = mountComposer({
      blocked: true,
      blockedLabel: 'PRD 待确认',
      blockedPlaceholder: '请先确认 PRD',
    })

    const editor = wrapper.find('[data-testid="mini-workstation-input"]')
    expect(editor.exists()).toBe(true)
    expect(editor.attributes('contenteditable')).toBe('false')
    expect(editor.attributes('data-placeholder')).toBe('请先确认 PRD')
    expect(wrapper.text()).toContain('PRD 待确认')

    editor.element.textContent = '继续生成'
    await editor.trigger('input')
    expect(wrapper.emitted('update:inputText')).toBeUndefined()
    expect(wrapper.find('[data-testid="upload"]').attributes('data-disabled')).toBe('true')
  })

  it('emits input updates when unblocked', async () => {
    const wrapper = mountComposer()
    const editor = wrapper.find('[data-testid="mini-workstation-input"]')
    editor.element.textContent = '继续生成'
    await editor.trigger('input')

    expect(wrapper.emitted('update:inputText')?.[0]?.[0]).toBe('继续生成')
  })

  it('keeps Enter as submit for chat input', async () => {
    const onInputEnter = vi.fn()
    const wrapper = mountComposer({
      inputText: '继续生成',
      onInputEnter,
    })
    const editor = wrapper.find('[data-testid="mini-workstation-input"]')

    await editor.trigger('keydown', { key: 'Enter' })

    expect(onInputEnter).toHaveBeenCalledTimes(1)
    expect(wrapper.emitted('update:inputText')).toBeUndefined()
  })

  it('uses Enter as a newline in schedule instructions', async () => {
    const onInputEnter = vi.fn()
    const wrapper = mountComposer({
      variant: 'schedule',
      inputText: '第一行',
      onInputEnter,
    })
    const editor = wrapper.find('[data-testid="mini-workstation-input"]')

    await editor.trigger('keydown', { key: 'Enter' })

    expect(onInputEnter).not.toHaveBeenCalled()
    expect(wrapper.emitted('update:inputText')?.at(-1)?.[0]).toBe('第一行\n')
  })
})
