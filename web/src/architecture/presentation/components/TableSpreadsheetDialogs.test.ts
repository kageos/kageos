import { flushPromises, shallowMount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import ImportDialog from './TableSpreadsheetImportDialog.vue'
import ExportDialog from './TableSpreadsheetExportDialog.vue'
import { downloadTableImportErrors, parseTableSpreadsheetFile } from '@/architecture/presentation/views/utils/tableSpreadsheetFile'

vi.mock('@/architecture/presentation/views/utils/tableSpreadsheetFile', () => ({
  parseTableSpreadsheetFile: vi.fn(async () => ({
    rows: [
      { rowNumber: 2, data: { name: '成功' }, errors: [] },
      { rowNumber: 3, data: { name: '=失败' }, errors: [] }
    ],
    recognizedFields: [{ code: 'name', name: '名称', type: 'string' }],
    ignoredHeaders: [], fatalErrors: []
  })),
  downloadTableImportErrors: vi.fn(async () => {}),
  buildTableExportFileName: vi.fn(() => 'data.xlsx')
}))

describe('spreadsheet dialog operations', () => {
  it('protects pending import and includes server failures in filter and download', async () => {
    let finish!: (value: { createdCount: number, failedCount: number, errors: { rowNumber: number, message: string }[] }) => void
    const pending = new Promise<{ createdCount: number, failedCount: number, errors: { rowNumber: number, message: string }[] }>(resolve => { finish = resolve })
    const wrapper = shallowMount(ImportDialog, { props: { modelValue: true, fields: [], importRows: () => pending } })
    const vm = wrapper.vm as unknown as {
      handleFileChange: (event: Event) => Promise<void>
      confirmImport: () => Promise<void>
      downloadErrors: () => Promise<void>
      reset: () => void
      displayRows: { rowNumber: number }[]
    }
    await vm.handleFileChange({ target: { files: [new File([''], 'data.csv')], value: '' } } as unknown as Event)
    const operation = vm.confirmImport()
    await nextTick()
    const dialog = wrapper.findComponent({ name: 'ElDialog' })
    expect(dialog.props('showClose')).toBe(false)
    expect(dialog.props('closeOnClickModal')).toBe(false)
    expect(dialog.props('closeOnPressEscape')).toBe(false)
    dialog.vm.$emit('update:modelValue', false)
    vm.reset()
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    expect(vm.displayRows).toHaveLength(2)
    finish({ createdCount: 1, failedCount: 1, errors: [{ rowNumber: 3, message: '重复记录' }] })
    await operation
    await flushPromises()
    expect(vm.displayRows.map(row => row.rowNumber)).toEqual([3])
    expect(wrapper.emitted('imported')).toEqual([[1]])
    await vm.downloadErrors()
    expect(downloadTableImportErrors).toHaveBeenCalledWith('data.csv', expect.any(Array), [expect.objectContaining({ rowNumber: 3, errors: ['写入失败：重复记录'] })])
    expect(dialog.props('showClose')).toBe(true)
    dialog.vm.$emit('update:modelValue', false)
    expect(wrapper.emitted('update:modelValue')).toEqual([[false]])
    wrapper.unmount()
  })

  it('protects pending export and permits closing after failure', async () => {
    let fail!: (error: Error) => void
    const wrapper = shallowMount(ExportDialog, { props: { modelValue: true, total: 3, tableName: 'test', exportChunks: () => new Promise<void>((_, reject) => { fail = reject }) } })
    const vm = wrapper.vm as unknown as { startExport: () => Promise<void> }
    const operation = vm.startExport()
    await nextTick()
    const dialog = wrapper.findComponent({ name: 'ElDialog' })
    expect(dialog.props('showClose')).toBe(false)
    expect(dialog.props('closeOnClickModal')).toBe(false)
    expect(dialog.props('closeOnPressEscape')).toBe(false)
    dialog.vm.$emit('update:modelValue', false)
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    fail(new Error('network unavailable'))
    await operation
    await nextTick()
    expect(dialog.props('showClose')).toBe(true)
    dialog.vm.$emit('update:modelValue', false)
    expect(wrapper.emitted('update:modelValue')).toEqual([[false]])
    wrapper.unmount()
  })
})

it('keeps the newest file when parses finish out of order and ignores completion after close', async () => {
  type Preview = Awaited<ReturnType<typeof parseTableSpreadsheetFile>>
  let first!: (value: Preview) => void
  let second!: (value: Preview) => void
  vi.mocked(parseTableSpreadsheetFile)
    .mockImplementationOnce(() => new Promise(resolve => { first = resolve }))
    .mockImplementationOnce(() => new Promise(resolve => { second = resolve }))
  const wrapper = shallowMount(ImportDialog, { props: { modelValue: true, fields: [], importRows: vi.fn() } })
  const vm = wrapper.vm as unknown as { handleFileChange: (e: Event) => Promise<void>, preview?: Preview, fileName: string, parsing: boolean }
  const event = (name: string) => ({ target: { files: [new File([''], name)], value: '' } }) as unknown as Event
  const a = vm.handleFileChange(event('a.csv'))
  const b = vm.handleFileChange(event('b.csv'))
  const preview: Preview = { rows: [{ rowNumber: 2, data: { name: 'B' }, errors: [] }], recognizedFields: [], ignoredHeaders: [], fatalErrors: [] }
  second(preview)
  await b
  first({ ...preview, rows: [] })
  await a
  expect(vm.fileName).toBe('b.csv')
  expect(vm.preview?.rows[0]?.data.name).toBe('B')
  expect(vm.parsing).toBe(false)
  vi.mocked(parseTableSpreadsheetFile).mockImplementationOnce(() => new Promise(resolve => { first = resolve }))
  const c = vm.handleFileChange(event('c.csv'))
  await wrapper.setProps({ modelValue: false })
  first(preview)
  await c
  expect(vm.preview).toBeUndefined()
  expect(vm.parsing).toBe(false)
  wrapper.unmount()
})
