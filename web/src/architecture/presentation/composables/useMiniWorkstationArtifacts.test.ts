import { describe, expect, it } from 'vitest'
import {
  buildDisplayFieldArtifactItem,
  buildFileArtifactItem,
  buildMiniArtifactItems,
  isGeneratedArtifactToolCall
} from './useMiniWorkstationArtifacts'

describe('useMiniWorkstationArtifacts', () => {
  it('classifies output files by extension', () => {
    expect(buildFileArtifactItem({ name: 'chart.PNG', href: '/files/chart.PNG', source: 'output' }, 0)).toMatchObject({
      key: 'file:output:/files/chart.PNG:0',
      meta: 'PNG · 输出文件',
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

    expect(buildFileArtifactItem({ name: 'demo.m4v', href: '/files/demo.m4v', source: 'output' }, 3)).toMatchObject({
      tag: '视频',
      tone: 'media',
      ext: 'M4V'
    })
  })

  it('marks uploaded files as input artifacts', () => {
    expect(buildFileArtifactItem({ name: 'source.csv', href: '/files/source.csv', source: 'upload' }, 0)).toMatchObject({
      key: 'file:upload:/files/source.csv:0',
      meta: 'CSV · 输入文件',
      tag: '数据',
      tone: 'data'
    })
  })

  it('builds one artifact list from output files, display fields, and uploads', () => {
    const items = buildMiniArtifactItems({
      uploadedFiles: [{ name: 'input.pdf', href: '/files/input.pdf', source: 'upload' }],
      outputFiles: [{ name: 'result.xlsx', href: '/files/result.xlsx', source: 'output' }],
      displayFields: [{
        fieldKey: 'summary',
        label: '摘要',
        value: '已完成',
        type: 'text'
      }]
    })

    expect(items.map(item => item.name)).toEqual(['result.xlsx', '摘要', 'input.pdf'])
    expect(items.map(item => item.meta)).toEqual(['XLSX · 输出文件', '已完成', 'PDF · 输入文件'])
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
      name: 'write_file',
      status: 'ok'
    } as any)).toBe(true)

    expect(isGeneratedArtifactToolCall({
      name: 'run_python',
      status: 'ok',
      result_data: { kind: 'workspace_file' }
    } as any)).toBe(true)

    expect(isGeneratedArtifactToolCall({
      name: 'write_file',
      status: 'error'
    } as any)).toBe(false)
  })
})
