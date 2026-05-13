import { describe, expect, it } from 'vitest'
import {
  buildDisplayFieldArtifactItem,
  buildFileArtifactItem,
  isGeneratedArtifactToolCall
} from './useMiniWorkstationArtifacts'

describe('useMiniWorkstationArtifacts', () => {
  it('classifies output files by extension', () => {
    expect(buildFileArtifactItem({ name: 'chart.PNG', href: '/files/chart.PNG', source: 'output' }, 0)).toMatchObject({
      tag: '图片',
      tone: 'image',
      ext: 'PNG',
      previewUrl: '/files/chart.PNG'
    })

    expect(buildFileArtifactItem({ name: 'report.xlsx?download=1', href: '/files/report.xlsx', source: 'output' }, 1)).toMatchObject({
      tag: '数据',
      tone: 'data',
      ext: 'XLSX'
    })

    expect(buildFileArtifactItem({ name: 'archive.zip', href: '/files/archive.zip', source: 'output' }, 2)).toMatchObject({
      tag: '压缩包',
      tone: 'archive',
      ext: 'ZIP'
    })
  })

  it('builds display field artifacts with compact metadata', () => {
    const item = buildDisplayFieldArtifactItem({
      fieldKey: 'result',
      label: '处理结果',
      value: '第一行\n第二行   第三行',
      type: 'text'
    }, 0)

    expect(item).toMatchObject({
      key: 'field:处理结果:0',
      name: '处理结果',
      meta: '第一行 第二行 第三行',
      tag: '字段',
      tone: 'field'
    })
  })

  it('recognizes tool calls that generated workspace artifacts', () => {
    expect(isGeneratedArtifactToolCall({
      name: 'write_go_file',
      status: 'ok'
    } as any)).toBe(true)

    expect(isGeneratedArtifactToolCall({
      name: 'run_python',
      status: 'ok',
      result_data: { kind: 'workspace_file' }
    } as any)).toBe(true)

    expect(isGeneratedArtifactToolCall({
      name: 'write_go_file',
      status: 'error'
    } as any)).toBe(false)
  })
})
