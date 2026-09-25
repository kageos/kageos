import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { tableGetDeletedRows } from '@/architecture/presentation/context/api/function'
import type { FunctionDetail } from '@/architecture/domain/types'
import Dialog from './TableDeletedRowsDialog.vue'

vi.mock('@/architecture/presentation/context/api/function', () => ({
  tableGetDeletedRows: vi.fn(), tableGetRecyclePolicy: vi.fn(),
  tablePurgeRows: vi.fn(), tableRestoreRows: vi.fn(), tableUpdateRecyclePolicy: vi.fn()
}))

describe('recycle pagination', () => {
  it('ignores an older response and keeps loading until the newest request finishes', async () => {
    type Result = Awaited<ReturnType<typeof tableGetDeletedRows>>
    let first!: (value: Result) => void
    let second!: (value: Result) => void
    vi.mocked(tableGetDeletedRows)
      .mockImplementationOnce(() => new Promise(resolve => { first = resolve }))
      .mockImplementationOnce(() => new Promise(resolve => { second = resolve }))
    const wrapper = shallowMount(Dialog, { props: { modelValue: false, functionDetail: { router: '/test' } as FunctionDetail }, global: { directives: { loading: () => {} } } })
    const vm = wrapper.vm as unknown as { loadRows: () => Promise<void>, page: number, rows: { id: number }[], loading: boolean }
    const a = vm.loadRows()
    vm.page = 2
    const b = vm.loadRows()
    first({ rows: [{ id: 1 }], total: 30, page: 1, page_size: 20 } as Result)
    await a
    expect(vm.rows).toEqual([])
    expect(vm.loading).toBe(true)
    second({ rows: [{ id: 21 }], total: 30, page: 2, page_size: 20 } as Result)
    await b
    expect(vm.rows).toEqual([{ id: 21 }])
    expect(vm.loading).toBe(false)
    wrapper.unmount()
  })

  it('does not let a late older response overwrite the current page', async () => {
    type Result = Awaited<ReturnType<typeof tableGetDeletedRows>>
    let finish!: (value: Result) => void
    vi.mocked(tableGetDeletedRows).mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
      .mockResolvedValueOnce({ rows: [{ id: 21 }], total: 30, page: 2, page_size: 20 } as Result)
    const wrapper = shallowMount(Dialog, { props: { modelValue: false, functionDetail: { router: '/test' } as FunctionDetail }, global: { directives: { loading: () => {} } } })
    const vm = wrapper.vm as unknown as { loadRows: () => Promise<void>, page: number, rows: { id: number }[] }
    const a = vm.loadRows()
    vm.page = 2
    await vm.loadRows()
    finish({ rows: [{ id: 1 }], total: 30, page: 1, page_size: 20 } as Result)
    await a
    expect(vm.rows).toEqual([{ id: 21 }])
    wrapper.unmount()
  })
})
