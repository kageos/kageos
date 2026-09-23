import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { FunctionDetail } from '@/architecture/domain/types'

const listShares = vi.hoisted(() => vi.fn())
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/architecture/presentation/context/api/publicShare', () => ({
  listPublicShares: listShares,
  disablePublicShare: vi.fn(),
}))
vi.mock('qrcode', () => ({ default: { toDataURL: vi.fn() } }))
import PublicSharePanel from './PublicSharePanel.vue'

const row = (title: string) => ({ share_id: title, title, public_url: `https://example.test/s/${title}`, enabled: true, use_count: 12, max_uses: 0, created_at: '2026-09-14T08:00:00Z', created_by: 'owner' })
const detail = (path: string) => ({ full_code_path: path }) as FunctionDetail
function render() {
  return mount(PublicSharePanel, {
    props: { functionDetail: detail('demo/a.form'), functionNode: null },
    global: {
      directives: { loading: () => {} },
      stubs: {
        PublicShareCreateDialog: true,
        ElButton: { template: '<button><slot /></button>' },
        ElInput: true, ElSelect: true, ElOption: true, ElCheckbox: true,
        ElIcon: { template: '<i><slot /></i>' },
        ElTag: { template: '<span><slot /></span>' },
        ElDialog: true,
        ElResult: { props: ['title', 'subTitle'], template: '<div>{{ title }} {{ subTitle }}<slot name="extra" /></div>' },
        ElEmpty: { props: ['description'], template: '<div>{{ description }}<slot /></div>' },
      },
    },
  })
}

describe('PublicSharePanel', () => {
  beforeEach(() => listShares.mockReset())

  it('ignores a previous function response after navigation', async () => {
    let resolveOld!: (value: unknown) => void
    listShares.mockReturnValueOnce(new Promise(resolve => { resolveOld = resolve }))
      .mockResolvedValueOnce({ items: [row('new-link')] })
    const wrapper = render()
    await wrapper.setProps({ functionDetail: detail('demo/b.form') })
    await flushPromises()
    resolveOld({ items: [row('old-link')] })
    await flushPromises()
    expect(wrapper.findAll('.share-row')).toHaveLength(1)
    expect(wrapper.text()).toContain('new-link')
    expect(wrapper.text()).not.toContain('old-link')
    await wrapper.setProps({ functionDetail: null })
    expect(wrapper.findAll('.share-row')).toHaveLength(0)
    wrapper.unmount()
  })

  it('shows failed loads separately from an empty list and supports retry', async () => {
    listShares.mockRejectedValueOnce(new Error('Request failed')).mockResolvedValueOnce({ items: [row('recovered')] })
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain('publicSharePanel.loadFailed')
    expect(wrapper.text()).not.toContain('publicSharePanel.empty')
    await wrapper.findAll('button').find(button => button.text() === 'common.refresh')!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('recovered')
    expect(wrapper.text()).not.toContain('publicSharePanel.loadFailed')
    wrapper.unmount()
  })
})
