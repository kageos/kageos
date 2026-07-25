import { flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { resolveFileRefs } from '@/architecture/presentation/context/api/storage'
import type { ChatMessage } from './useWorkspaceChatStream'
import { buildOutputPanelFile, resolveOutputFileBatch, useMiniWorkstationPanel } from './useMiniWorkstationPanel'

vi.mock('@/architecture/presentation/context/api/storage', () => ({
  resolveFileRefs: vi.fn(),
}))

const resolveFileRefsMock = vi.mocked(resolveFileRefs)

describe('useMiniWorkstationPanel', () => {
  beforeEach(() => {
    resolveFileRefsMock.mockReset()
  })

  it('prefers resolved source names for artifact display and download', () => {
    expect(buildOutputPanelFile({
      ref: 'kageos/output/generated-12345.xlsx',
      name: 'generated-12345.xlsx',
    }, {
      ref: 'kageos/output/generated-12345.xlsx',
      bucket: 'kageos',
      key: 'output/generated-12345.xlsx',
      name: '商品导入模板_已填充.xlsx',
      source_name: '商品导入模板_已填充.xlsx',
      download_url: '/kageos/output/generated-12345.xlsx',
    })).toEqual({
      name: '商品导入模板_已填充.xlsx',
      href: 'http://localhost:3000/kageos/output/generated-12345.xlsx',
      source: 'output',
      ref: 'kageos/output/generated-12345.xlsx',
    })
  })

  it('resolves output refs before building artifact panel files', async () => {
    resolveFileRefsMock.mockResolvedValue([{
      ref: 'kageos/output/generated-12345.xlsx',
      bucket: 'kageos',
      key: 'output/generated-12345.xlsx',
      name: '商品导入模板_已填充.xlsx',
      source_name: '商品导入模板_已填充.xlsx',
      download_url: '/kageos/output/generated-12345.xlsx',
    }])
    const messages = ref<ChatMessage[]>([{
      role: 'assistant',
      content: '',
      tool_calls: [{
        name: 'run_python',
        status: 'ok',
        result_data: { output_files: 'kageos/output/generated-12345.xlsx' },
        metadata: { display_file_fields: ['output_files'] },
      }],
    }])

    const panel = useMiniWorkstationPanel(messages)
    await flushPromises()

    expect(resolveFileRefsMock).toHaveBeenCalledWith(['kageos/output/generated-12345.xlsx'], 'browser')
    expect(panel.outputFiles.value).toEqual([{
      name: '商品导入模板_已填充.xlsx',
      href: 'http://localhost:3000/kageos/output/generated-12345.xlsx',
      source: 'output',
      ref: 'kageos/output/generated-12345.xlsx',
    }])
  })

  it('retries a failed output file metadata request once', async () => {
    const resolver = vi.fn()
      .mockRejectedValueOnce(new Error('temporary network error'))
      .mockResolvedValueOnce([{
        ref: 'kageos/output/report.xlsx',
        bucket: 'kageos',
        key: 'output/report.xlsx',
        source_name: '报告.xlsx',
      }])

    const resolved = await resolveOutputFileBatch(['kageos/output/report.xlsx'], resolver, 0)

    expect(resolver).toHaveBeenCalledTimes(2)
    expect(resolved[0]?.source_name).toBe('报告.xlsx')
  })

  it('resolves long artifact histories in batches of at most 100 refs', async () => {
    const refs = Array.from({ length: 205 }, (_, index) => `kageos/output/file-${index}.xlsx`)
    resolveFileRefsMock.mockImplementation(async batch => batch.map(refValue => ({
      ref: refValue,
      bucket: 'kageos',
      key: refValue.slice('kageos/'.length),
      source_name: `已命名-${refValue.split('/').pop()}`,
      download_url: `/${refValue}`,
    })))
    const messages = ref<ChatMessage[]>([{
      role: 'assistant',
      content: '',
      tool_calls: [{
        name: 'run_python',
        status: 'ok',
        result_data: { output_files: refs.join(',') },
        metadata: { display_file_fields: ['output_files'] },
      }],
    }])

    const panel = useMiniWorkstationPanel(messages)
    await flushPromises()

    expect(resolveFileRefsMock.mock.calls.map(call => call[0].length)).toEqual([100, 100, 5])
    expect(panel.outputFiles.value).toHaveLength(205)
    expect(panel.outputFiles.value[204]?.name).toBe('已命名-file-204.xlsx')
  })
})
