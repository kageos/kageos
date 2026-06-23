import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StructuredPromptComposer from './StructuredPromptComposer.vue'

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

  it('emits serialized raw resource tokens when edited', async () => {
    const wrapper = mountComposer('调用 </system/app/search.form>')
    const editor = wrapper.find('[data-testid="structured-prompt-editor"]')

    editor.element.appendChild(document.createTextNode(' 完成后总结'))
    await editor.trigger('input')

    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toBe('调用 </system/app/search.form> 完成后总结')
  })
})
