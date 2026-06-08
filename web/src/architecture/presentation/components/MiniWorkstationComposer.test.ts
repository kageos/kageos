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

    const textarea = wrapper.find('[data-testid="mini-workstation-input"]')
    expect(textarea.exists()).toBe(true)
    expect(textarea.attributes('disabled')).toBeDefined()
    expect(textarea.attributes('placeholder')).toBe('请先确认 PRD')
    expect(wrapper.text()).toContain('PRD 待确认')

    await textarea.setValue('继续生成')
    expect(wrapper.emitted('update:inputText')).toBeUndefined()
    expect(wrapper.find('[data-testid="upload"]').attributes('data-disabled')).toBe('true')
  })

  it('emits input updates when unblocked', async () => {
    const wrapper = mountComposer()
    await wrapper.find('[data-testid="mini-workstation-input"]').setValue('继续生成')

    expect(wrapper.emitted('update:inputText')?.[0]?.[0]).toBe('继续生成')
  })
})
