import { mount, flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ResourceLogArchives from './ResourceLogArchives.vue'
const list = vi.hoisted(() => vi.fn())
vi.mock('@/architecture/presentation/context/api/system-settings', () => ({
  listResourceLogArchives: list,
  downloadResourceLogArchive: vi.fn(),
  retryResourceLogArchive: vi.fn(),
}))
describe('resource log archives', () => {
  it('requests the selected resource and discards an old scope response', async () => {
    let resolveOld!: (value: unknown) => void
    list.mockReset().mockImplementationOnce(() => new Promise(resolve => { resolveOld = resolve }))
      .mockResolvedValueOnce({ list: [], total: 0, min_records: 25, retention_days: 30 })
    const wrapper = mount(ResourceLogArchives, { props: { resourcePath: '/alice/ops/first' } })
    expect(list).toHaveBeenCalledWith('/alice/ops/first', 1)
    await wrapper.setProps({ resourcePath: '/alice/ops/second' })
    await flushPromises()
    expect(list).toHaveBeenLastCalledWith('/alice/ops/second', 1)
    resolveOld({ list: [], total: 0, min_records: 99999, retention_days: 100 })
    await flushPromises()
    expect(wrapper.text()).toContain('/alice/ops/second')
    expect(wrapper.text()).not.toContain('99999')
    wrapper.unmount()
  })
  it('shows permission failures without retaining records from another directory', async () => {
    list.mockReset().mockRejectedValue(new Error('需要管理权限'))
    const wrapper = mount(ResourceLogArchives, { props: { resourcePath: '/alice/ops/restricted' } })
    await flushPromises()
    expect(wrapper.text()).toContain('需要管理权限')
    wrapper.unmount()
  })
})
