import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { searchUsersFuzzy } from '@/architecture/presentation/context/api/user'
import { searchResources } from '@/architecture/presentation/context/api/service-tree'
import StructuredPromptComposer from './StructuredPromptComposer.vue'

vi.mock('@/architecture/presentation/context/api/user', () => ({
  getUsersByUsernames: vi.fn(async () => ({ users: [] })),
  searchUsersFuzzy: vi.fn(async () => ({
    users: [{
      username: 'system',
      nickname: 'system(系统)',
      avatar: '',
      email: '',
      signature: '',
    }],
  })),
}))

vi.mock('@/architecture/presentation/context/api/service-tree', () => ({
  getServiceTreeDetail: vi.fn(async () => {
    throw new Error('not found')
  }),
  searchResources: vi.fn(async () => ({
    items: [{
      id: 1,
      name: '订单表',
      code: 'orders',
      type: 'function',
      full_code_path: '/system/app/orders.table',
      description: '订单数据表',
      template_type: 'table',
      run_count: 0,
    }],
  })),
}))

const IconStub = {
  template: '<span><slot /></span>',
}

function mountComposer(modelValue: string) {
  return mount(StructuredPromptComposer, {
    props: {
      modelValue,
      placeholder: '输入任务',
    },
    global: {
      stubs: {
        ElIcon: IconStub,
        EditPen: IconStub,
        View: IconStub,
      },
    },
  })
}

describe('StructuredPromptComposer', () => {
  it('renders resource path tokens in edit mode', () => {
    const wrapper = mountComposer('调用 </system/demos/weixin/wechat_articles/search_articles.form>')

    const token = wrapper.find('.spc-editor-token')
    expect(token.exists()).toBe(true)
    expect(token.attributes('data-token-raw')).toBe('</system/demos/weixin/wechat_articles/search_articles.form>')
    expect(token.text()).toBe('search_articles.form')
  })

  it('renders relative resource tokens against the current workspace path', () => {
    const wrapper = mount(StructuredPromptComposer, {
      props: {
        modelValue: '调用 <./record_screening.form>',
        fullCodePath: '/system/democase/recruit_interview',
      },
      global: {
        stubs: {
          ElIcon: IconStub,
          EditPen: IconStub,
          View: IconStub,
        },
      },
    })

    const token = wrapper.find('.spc-editor-token')
    expect(token.exists()).toBe(true)
    expect(token.attributes('data-token-raw')).toBe('<./record_screening.form>')
    expect(token.attributes('data-path')).toBe('/system/democase/recruit_interview/record_screening.form')
    expect(token.text()).toBe('record_screening.form')
  })

  it('renders invocation cards in preview mode', async () => {
    const wrapper = mountComposer([
      '函数调用：',
      '用途：复制后粘贴到工作台，AI 会按下面信息识别并调用。',
      '工具：run_table_create',
      '函数：</system/app/orders.table>',
      '',
      '参数：',
      'body = [{"title":"测试"}]',
    ].join('\n'))

    await wrapper.findAll('.spc-mode-btn')[1]?.trigger('click')

    expect(wrapper.find('.spc-invocation-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('run_table_create')
    expect(wrapper.text()).toContain('orders.table')
    expect(wrapper.text()).toContain('body')
  })

  it('renders relative invocation resources in preview mode', async () => {
    const wrapper = mount(StructuredPromptComposer, {
      props: {
        modelValue: [
          '函数调用：',
          '工具：run_form_submit',
          '函数：<./record_screening.form>',
        ].join('\n'),
        fullCodePath: '/system/democase/recruit_interview',
      },
      global: {
        stubs: {
          ElIcon: IconStub,
          EditPen: IconStub,
          View: IconStub,
        },
      },
    })

    await wrapper.findAll('.spc-mode-btn')[1]?.trigger('click')

    const resource = wrapper.find('.spc-invocation-resource')
    expect(resource.exists()).toBe(true)
    expect(resource.attributes('title')).toBe('/system/democase/recruit_interview/record_screening.form')
    expect(resource.text()).toContain('record_screening.form')
  })

  it('renders readonly preview without exposing edit mode', async () => {
    const wrapper = mount(StructuredPromptComposer, {
      props: {
        modelValue: '请 @system 检查 </system/app/orders.table>',
        readonlyPreview: true,
        showToolbar: false,
        disabled: true,
      },
      global: {
        stubs: {
          ElIcon: IconStub,
          EditPen: IconStub,
          View: IconStub,
        },
      },
    })

    expect(wrapper.find('[data-testid="structured-prompt-preview"]').isVisible()).toBe(true)
    expect(wrapper.find('[data-testid="structured-prompt-editor"]').isVisible()).toBe(false)
    expect(wrapper.find('.spc-user-chip').text()).toContain('@system')
    expect(wrapper.find('.spc-resource-chip').text()).toContain('orders.table')
  })

  it('emits serialized raw resource tokens when edited', async () => {
    const wrapper = mountComposer('调用 </system/app/search.form>')
    const editor = wrapper.find('[data-testid="structured-prompt-editor"]')

    editor.element.appendChild(document.createTextNode(' 完成后总结'))
    await editor.trigger('input')

    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toBe('调用 </system/app/search.form> 完成后总结')
  })

  it('renders user mentions as chips while keeping raw @username text', async () => {
    const wrapper = mountComposer('请 @beiluo 协助处理')
    const editor = wrapper.find('[data-testid="structured-prompt-editor"]')

    const token = wrapper.find('.spc-editor-token.is-user')
    expect(token.exists()).toBe(true)
    expect(token.attributes('data-token-raw')).toBe('@beiluo')

    editor.element.appendChild(document.createTextNode('，谢谢'))
    await editor.trigger('input')

    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toBe('请 @beiluo 协助处理，谢谢')
  })

  it('shows system mentions with readable labels and a click card', async () => {
    const wrapper = mountComposer('交给 @system 处理')

    const token = wrapper.find('.spc-editor-token.is-user')
    expect(token.text()).toBe('@system(系统)')
    expect(token.attributes('data-token-raw')).toBe('@system')

    await token.trigger('click')

    const card = wrapper.find('[data-testid="structured-prompt-info-card"]')
    expect(card.exists()).toBe(true)
    expect(card.text()).toContain('@system(系统)')
    expect(card.text()).toContain('@system')
  })

  it('normalizes already decorated user mentions instead of nesting labels', async () => {
    const wrapper = mountComposer('交给 @system(system(系统)) 处理')
    const editor = wrapper.find('[data-testid="structured-prompt-editor"]')

    const token = wrapper.find('.spc-editor-token.is-user')
    expect(token.text()).toBe('@system(系统)')
    expect(token.attributes('data-token-raw')).toBe('@system')

    editor.element.appendChild(document.createTextNode('，谢谢'))
    await editor.trigger('input')

    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toBe('交给 @system 处理，谢谢')
  })

  it('does not submit while mention search is open and waiting for options', async () => {
    const wrapper = mount(StructuredPromptComposer, {
      props: {
        modelValue: '',
        submitOnEnter: true,
      },
      global: {
        stubs: {
          ElIcon: IconStub,
          EditPen: IconStub,
          View: IconStub,
        },
      },
    })
    const editor = wrapper.find('[data-testid="structured-prompt-editor"]')

    editor.element.textContent = '@sys'
    await editor.trigger('input')
    await editor.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('enter')).toBeUndefined()
  })

  it('selects the first mention option on enter by default', async () => {
    vi.useFakeTimers()
    try {
      const wrapper = mount(StructuredPromptComposer, {
        props: {
          modelValue: '',
          submitOnEnter: true,
        },
        global: {
          stubs: {
            ElIcon: IconStub,
            EditPen: IconStub,
            View: IconStub,
          },
        },
      })
      const editor = wrapper.find('[data-testid="structured-prompt-editor"]')

      editor.element.textContent = '@sys'
      await editor.trigger('input')
      await vi.advanceTimersByTimeAsync(230)
      await editor.trigger('keydown', { key: 'Enter' })

      expect(searchUsersFuzzy).toHaveBeenCalledWith('sys', 8)
      expect(wrapper.emitted('enter')).toBeUndefined()
      expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toBe('@system ')
      expect(wrapper.find('.spc-editor-token.is-user').text()).toBe('@system(系统)')
    } finally {
      vi.useRealTimers()
    }
  })

  it('selects the first mention option when enter is pressed before search finishes', async () => {
    vi.useFakeTimers()
    try {
      const wrapper = mount(StructuredPromptComposer, {
        props: {
          modelValue: '',
          submitOnEnter: true,
        },
        global: {
          stubs: {
            ElIcon: IconStub,
            EditPen: IconStub,
            View: IconStub,
          },
        },
      })
      const editor = wrapper.find('[data-testid="structured-prompt-editor"]')

      editor.element.textContent = '@sys'
      await editor.trigger('input')
      await editor.trigger('keydown', { key: 'Enter' })
      await vi.advanceTimersByTimeAsync(230)

      expect(searchUsersFuzzy).toHaveBeenCalledWith('sys', 8)
      expect(wrapper.emitted('enter')).toBeUndefined()
      expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toBe('@system ')
      expect(wrapper.find('.spc-editor-token.is-user').text()).toBe('@system(系统)')
    } finally {
      vi.useRealTimers()
    }
  })

  it('selects the first mention option when enter arrives before the mention panel opens', async () => {
    vi.useFakeTimers()
    try {
      const wrapper = mount(StructuredPromptComposer, {
        props: {
          modelValue: '',
          submitOnEnter: true,
        },
        global: {
          stubs: {
            ElIcon: IconStub,
            EditPen: IconStub,
            View: IconStub,
          },
        },
      })
      const editor = wrapper.find('[data-testid="structured-prompt-editor"]')

      editor.element.textContent = '@sys'
      await editor.trigger('keydown', { key: 'Enter' })
      await vi.advanceTimersByTimeAsync(230)

      expect(searchUsersFuzzy).toHaveBeenCalledWith('sys', 8)
      expect(wrapper.emitted('enter')).toBeUndefined()
      expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toBe('@system ')
      expect(wrapper.find('.spc-editor-token.is-user').text()).toBe('@system(系统)')
    } finally {
      vi.useRealTimers()
    }
  })

  it('selects the loaded mention option on the first enter after composition ends', async () => {
    vi.useFakeTimers()
    try {
      const wrapper = mount(StructuredPromptComposer, {
        props: {
          modelValue: '',
          submitOnEnter: true,
        },
        global: {
          stubs: {
            ElIcon: IconStub,
            EditPen: IconStub,
            View: IconStub,
          },
        },
      })
      const editor = wrapper.find('[data-testid="structured-prompt-editor"]')

      await vi.advanceTimersByTimeAsync(10)
      await editor.trigger('compositionstart')
      editor.element.textContent = '@sys'
      await editor.trigger('input')
      await editor.trigger('compositionend')
      await nextTick()
      await vi.advanceTimersByTimeAsync(230)
      await editor.trigger('keydown', { key: 'Enter' })

      expect(searchUsersFuzzy).toHaveBeenCalledWith('sys', 8)
      expect(wrapper.emitted('enter')).toBeUndefined()
      expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toBe('@system ')
      expect(wrapper.find('.spc-editor-token.is-user').text()).toBe('@system(系统)')
    } finally {
      vi.useRealTimers()
    }
  })

  it('keeps resource mention icon components raw when rendering options', async () => {
    vi.useFakeTimers()
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    try {
      const wrapper = mount(StructuredPromptComposer, {
        props: {
          modelValue: '',
        },
        global: {
          stubs: {
            ElIcon: IconStub,
            EditPen: IconStub,
            View: IconStub,
          },
        },
      })
      const editor = wrapper.find('[data-testid="structured-prompt-editor"]')

      editor.element.textContent = '/orders'
      await editor.trigger('input')
      await vi.advanceTimersByTimeAsync(230)
      await nextTick()

      expect(searchResources).toHaveBeenCalledWith(expect.objectContaining({ keyword: 'orders' }))
      expect(document.body.querySelector('.spc-mention-resource-component')).not.toBeNull()
      expect(warnSpy.mock.calls.some((args) => args.join(' ').includes('Component that was made reactive'))).toBe(false)
      wrapper.unmount()
    } finally {
      warnSpy.mockRestore()
      vi.useRealTimers()
    }
  })

  it('does not submit or rerender while Chinese IME composition is committing', async () => {
    const wrapper = mount(StructuredPromptComposer, {
      props: {
        modelValue: '',
        submitOnEnter: true,
      },
      global: {
        stubs: {
          ElIcon: IconStub,
          EditPen: IconStub,
          View: IconStub,
        },
      },
    })
    const editor = wrapper.find('[data-testid="structured-prompt-editor"]')

    await editor.trigger('compositionstart')
    editor.element.textContent = 'ni'
    await editor.trigger('input')

    expect(wrapper.emitted('update:modelValue')).toBeUndefined()

    editor.element.textContent = '你'
    await editor.trigger('compositionend')
    await nextTick()

    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toBe('你')

    await editor.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('enter')).toBeUndefined()
  })

  it('emits enter when submit-on-enter is enabled', async () => {
    const wrapper = mount(StructuredPromptComposer, {
      props: {
        modelValue: '执行任务',
        submitOnEnter: true,
      },
      global: {
        stubs: {
          ElIcon: IconStub,
          EditPen: IconStub,
          View: IconStub,
        },
      },
    })

    await wrapper.find('[data-testid="structured-prompt-editor"]').trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('enter')).toHaveLength(1)
  })
})
