import { flushPromises, shallowMount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import WorkspaceInbox from './WorkspaceInbox.vue'
import { getAppWithServiceTree } from '@/architecture/presentation/context/api/app'
import type { ServiceTree, App } from '@/architecture/domain/types'
import * as api from '@/architecture/presentation/context/api/message'

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/workspace/owner/app', query: {} }),
  useRouter: () => ({ replace: vi.fn(), push: vi.fn() }),
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/architecture/presentation/context/api/message', () => ({
  listMessageInbox: vi.fn(),
  getMessageInboxItem: vi.fn(),
  getMessageInboxUnreadCount: vi.fn().mockResolvedValue({ unread_count: 2 }),
  listMessageInboxSourceCounts: vi.fn().mockResolvedValue({ list: [] }),
  listMessageInboxWorkspaceCounts: vi.fn().mockResolvedValue({ list: [] }),
  markAllMessageInboxItemsRead: vi.fn().mockResolvedValue(undefined),
  markMessageInboxItemRead: vi.fn().mockResolvedValue(undefined),
  markMessageInboxSourceRead: vi.fn().mockResolvedValue(undefined),
}))
vi.mock('@/architecture/presentation/context/api/app', () => ({ getAppList: vi.fn().mockResolvedValue([]), getAppWithServiceTree: vi.fn() }))
vi.mock('@/architecture/presentation/composables/useLazyMarkdownRenderer', () => ({
  useLazyMarkdownRenderer: () => ({ renderMarkdown: (value: string) => value, preloadMarkdown: vi.fn() }),
}))

const messages = [1, 2].map(id => ({
  id, recipient_id: id, from: 'system', title: `Message ${id}`, content: `Body ${id}`,
  created_at: '2026-09-23T10:00:00Z', read_at: null, source_path: '/owner/app/notify.form',
}))
const drawer = defineComponent({ name: 'ElDrawer', props: ['modelValue'], emits: ['open', 'closed'], template: '<div><slot name="header"/><slot/></div>' })
const wrappers: ReturnType<typeof shallowMount>[] = []
async function openInbox() {
  const wrapper = shallowMount(WorkspaceInbox, {
    props: { syncRoute: false },
    global: { stubs: { ElDrawer: drawer }, directives: { loading: {} } },
  })
  wrappers.push(wrapper)
  wrapper.vm.openDrawer()
  wrapper.findComponent(drawer).vm.$emit('open')
  await flushPromises()
  return wrapper
}
beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listMessageInboxWorkspaceCounts).mockResolvedValue({ list: [] })
  vi.mocked(api.listMessageInbox).mockResolvedValue({ list: messages, total: 125, page: 1, page_size: 20 })
  vi.mocked(api.getMessageInboxItem).mockImplementation(async id => messages.find(item => item.id === id)!)
})
afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()); vi.useRealTimers(); vi.unstubAllGlobals() })

describe('WorkspaceInbox', () => {
  it('opens every message as a complete card without requiring a click', async () => {
    const wrapper = await openInbox()
    expect(wrapper.findAll('.message-summary')).toHaveLength(2)
    expect(wrapper.findAll('.message-expanded')).toHaveLength(2)
    expect(api.markMessageInboxItemRead).not.toHaveBeenCalled()
    expect(api.markMessageInboxSourceRead).not.toHaveBeenCalled()
    expect(api.listMessageInbox).toHaveBeenCalledWith(expect.objectContaining({ page: 1, page_size: 20 }))
  })
  it('does not apply a delayed directory read to another directory', async () => {
    const wrapper = await openInbox()
    let finish!: () => void
    vi.mocked(api.markMessageInboxSourceRead).mockReturnValueOnce(new Promise<void>(resolve => { finish = resolve }))
    wrapper.vm.openForSource({ sourcePath: '/owner/app/first', includeChildren: true })
    await flushPromises()
    vi.mocked(api.listMessageInbox).mockResolvedValueOnce({ list: [messages[1]!], total: 1, page: 1, page_size: 20 })
    wrapper.vm.openForSource({ sourcePath: '/owner/app/second' }, false)
    await flushPromises()
    finish()
    await flushPromises()
    expect(wrapper.get('#inbox-message-2').classes()).toContain('is-unread')
    expect(wrapper.get('.inbox-content').text()).toBe('Body 2')
    expect(api.markMessageInboxSourceRead).toHaveBeenCalledExactlyOnceWith('/owner/app/first', true)
  })
  it('does not mark a directory read if its messages fail to load', async () => {
    const wrapper = await openInbox()
    vi.mocked(api.listMessageInbox).mockRejectedValueOnce(new Error('unavailable'))
    wrapper.vm.openForSource({ sourcePath: '/owner/app/first' })
    await flushPromises()
    expect(api.markMessageInboxSourceRead).not.toHaveBeenCalled()
  })
  it('requests the selected historical page without resetting to page one', async () => {
    const wrapper = await openInbox()
    const pagination = wrapper.findComponent({ name: 'ElPagination' })
    pagination.vm.$emit('update:current-page', 6)
    pagination.vm.$emit('current-change', 6)
    await flushPromises()
    expect(api.listMessageInbox).toHaveBeenLastCalledWith(expect.objectContaining({ page: 6, page_size: 20 }))
  })
  it('searches on the server, resets pagination, and safely highlights a matching title', async () => {
    const wrapper = await openInbox()
    vi.mocked(api.listMessageInbox).mockResolvedValueOnce({ list: [{ ...messages[0]!, title: '<img src=x>needle', content: 'prefix '.repeat(80) + 'needle details' }], total: 1, page: 1, page_size: 20 })
    wrapper.findComponent({ name: 'ElInput' }).vm.$emit('update:modelValue', 'needle')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(api.listMessageInbox).toHaveBeenLastCalledWith(expect.objectContaining({ q: 'needle', page: 1 }))
    expect(wrapper.get('.message-card-title mark').text()).toBe('needle')
    expect(wrapper.find('.message-summary img').exists()).toBe(false)
  })
  it('ignores a slow earlier search after a later search finishes', async () => {
    const wrapper = await openInbox()
    let finish!: (value: api.ListMessageInboxResp) => void
    vi.mocked(api.listMessageInbox).mockReturnValueOnce(new Promise(resolve => { finish = resolve }))
    const input = wrapper.findComponent({ name: 'ElInput' })
    input.vm.$emit('update:modelValue', 'old')
    await wrapper.get('form').trigger('submit')
    vi.mocked(api.listMessageInbox).mockResolvedValueOnce({ list: [{ ...messages[0]!, title: 'New result' }], total: 1, page: 1, page_size: 20 })
    input.vm.$emit('update:modelValue', 'new')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    finish({ list: [{ ...messages[0]!, title: 'Old result' }], total: 1, page: 1, page_size: 20 })
    await flushPromises()
    expect(wrapper.text()).toContain('New result')
    expect(wrapper.text()).not.toContain('Old result')
  })
  it('source navigation marks the directory and descendants read while keeping cards visible', async () => {
    const wrapper = await openInbox()
    wrapper.vm.openForSource({ sourcePath: '/owner/app/orders', includeChildren: true })
    await flushPromises()
    expect(api.listMessageInbox).toHaveBeenLastCalledWith(expect.objectContaining({ source_path: '/owner/app/orders', include_children: true }))
    expect(api.markMessageInboxSourceRead).toHaveBeenCalledExactlyOnceWith('/owner/app/orders', true)
    expect(wrapper.findAll('.inbox-message-card.is-unread')).toHaveLength(0)
    expect(wrapper.findAll('.inbox-content').map(body => body.text()).sort()).toEqual(['Body 1', 'Body 2'])
    expect(api.markMessageInboxItemRead).not.toHaveBeenCalled()
  })
  it('switches the directory tree with the workspace and ignores a late previous response', async () => {
    vi.stubGlobal('innerWidth', 1440)
    const workspaces = ['other', 'third'].map(code => ({ workspace_key: `/owner/${code}`, workspace_path: `/owner/${code}`, title: code, unread_count: 0, message_count: 1 }))
    vi.mocked(api.listMessageInboxWorkspaceCounts).mockResolvedValue({ list: workspaces })
    const wrapper = await openInbox()
    let finish!: (value: Awaited<ReturnType<typeof getAppWithServiceTree>>) => void
    vi.mocked(getAppWithServiceTree).mockReturnValueOnce(new Promise(resolve => { finish = resolve }))
    const tabs = wrapper.findAll('.workspace-tab')
    await tabs[0]!.trigger('click')
    expect(getAppWithServiceTree).toHaveBeenLastCalledWith('/owner/other')
    const node = { id: 1, name: 'Third workspace directory', full_code_path: '/owner/third/reports', type: 'package', children: [] } as unknown as ServiceTree
    vi.mocked(getAppWithServiceTree).mockResolvedValueOnce({ app: { user: 'owner', code: 'third' } as App, service_tree: [node] })
    await tabs[1]!.trigger('click')
    await flushPromises()
    wrapper.findComponent({ name: 'ElCheckbox' }).vm.$emit('update:modelValue', true)
    await flushPromises()
    expect(wrapper.findComponent({ name: 'ElTree' }).props('data')).toEqual([node])
    expect(api.listMessageInbox).toHaveBeenLastCalledWith(expect.objectContaining({ source_path: '/owner/third', include_children: true }))
    finish({ app: { user: 'owner', code: 'other' } as App, service_tree: [] })
    await flushPromises()
    expect(wrapper.findComponent({ name: 'ElTree' }).props('data')).toEqual([node])
    expect(api.markMessageInboxSourceRead).not.toHaveBeenCalled()
    wrapper.findComponent({ name: 'ElTree' }).vm.$emit('node-click', node)
    await flushPromises()
    expect(wrapper.findAll('.workspace-tab')[1]!.classes()).toContain('is-active')
    expect(api.listMessageInbox).toHaveBeenLastCalledWith(expect.objectContaining({ source_path: node.full_code_path }))
    expect(api.markMessageInboxSourceRead).toHaveBeenCalledExactlyOnceWith(node.full_code_path, true)
  })

})
