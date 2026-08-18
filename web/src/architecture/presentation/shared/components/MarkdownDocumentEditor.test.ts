import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MdEditor } from 'md-editor-v3'
import { notifyUploadComplete, uploadFile } from '@/architecture/presentation/context/uploadContext'
import MarkdownDocumentEditor from './MarkdownDocumentEditor.vue'

vi.mock('@/architecture/presentation/context/uploadContext', () => ({
  uploadFile: vi.fn(async (_router: string, file: File) => ({
    fileInfo: {
      key: `docs/${file.name}`,
      router: 'tester/docs/files',
      file_name: file.name,
      file_size: file.size,
      content_type: file.type,
      hash: 'hash',
    },
  })),
  notifyUploadComplete: vi.fn(async ({ file_name }: { file_name: string }) => ({
    download_url: `https://files.example/${encodeURIComponent(file_name)}`,
  })),
}))

describe('MarkdownDocumentEditor', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.setItem('user', JSON.stringify({ username: 'tester' }))
    Range.prototype.getClientRects = vi.fn(() => [] as unknown as DOMRectList)
    Range.prototype.getBoundingClientRect = vi.fn(() => new DOMRect())
  })

  function mountEditor(modelValue: string, attachTo?: Element | string) {
    return mount(MarkdownDocumentEditor, {
      props: { modelValue },
      attachTo,
      global: { plugins: [createPinia()] },
    })
  }

  it('loads and emits Markdown instead of editor HTML', async () => {
    const wrapper = mountEditor('# 使用说明\n\n初始内容')

    await flushPromises()
    expect(wrapper.find('.md-editor-preview h1').text()).toBe('使用说明')

    wrapper.findComponent(MdEditor).vm.$emit('update:modelValue', '## 修改后的内容')
    await flushPromises()

    const updates = wrapper.emitted('update:modelValue') || []
    expect(updates.at(-1)?.[0]).toBe('## 修改后的内容')
    wrapper.unmount()
  })

  it('uploads pasted images and regular files with the correct Markdown', async () => {
    const wrapper = mountEditor('')
    await flushPromises()

    const image = new File(['image'], 'clipboard.png', { type: 'image/png' })
    const documentFile = new File(['document'], '操作手册.pdf', { type: 'application/pdf' })
    const pasteEvent = new Event('paste', { bubbles: true, cancelable: true })
    Object.defineProperty(pasteEvent, 'clipboardData', {
      value: {
        getData: () => '',
        types: ['Files'],
        items: [
          { kind: 'file', getAsFile: () => image },
          { kind: 'file', getAsFile: () => documentFile },
        ],
        files: [image, documentFile],
      },
    })
    wrapper.element.dispatchEvent(pasteEvent)
    await flushPromises()

    expect(uploadFile).toHaveBeenCalledWith('tester/docs/files', image, expect.any(Function))
    expect(uploadFile).toHaveBeenCalledWith('tester/docs/files', documentFile, expect.any(Function))
    expect(notifyUploadComplete).toHaveBeenCalledTimes(2)
    const updates = wrapper.emitted('update:modelValue') || []
    expect(updates.at(-1)?.[0]).toContain('![clipboard.png](https://files.example/clipboard.png)')
    expect(updates.at(-1)?.[0]).toContain('[操作手册.pdf](https://files.example/%E6%93%8D%E4%BD%9C%E6%89%8B%E5%86%8C.pdf)')
    wrapper.unmount()
  })

  it('uses the browser fullscreen API without creating another overlay', async () => {
    let fullscreenElement: Element | null = null
    Object.defineProperty(document, 'fullscreenElement', {
      configurable: true,
      get: () => fullscreenElement,
    })
    Object.defineProperty(document, 'exitFullscreen', {
      configurable: true,
      value: vi.fn(async () => {
        fullscreenElement = null
        document.dispatchEvent(new Event('fullscreenchange'))
      }),
    })

    const wrapper = mountEditor('内容', document.body)
    await flushPromises()
    const shell = wrapper.get('.markdown-document-editor').element as HTMLElement
    Object.defineProperty(shell, 'requestFullscreen', {
      configurable: true,
      value: vi.fn(async () => {
        fullscreenElement = shell
        document.dispatchEvent(new Event('fullscreenchange'))
      }),
    })

    await wrapper.get('button[title="全屏编辑"]').trigger('click')
    expect(shell.requestFullscreen).toHaveBeenCalledOnce()
    expect(document.body.querySelector('.markdown-document-editor__drop-hint')).toBeNull()

    await wrapper.get('button[title="退出全屏"]').trigger('click')
    expect(document.exitFullscreen).toHaveBeenCalledOnce()
    wrapper.unmount()
  })
})
