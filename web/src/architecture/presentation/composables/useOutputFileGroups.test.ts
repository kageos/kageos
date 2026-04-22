import { describe, expect, it } from 'vitest'
import { extractFileGroupsFromResult } from './useOutputFileGroups'

describe('extractFileGroupsFromResult', () => {
  it('extracts a root-level files ref string', () => {
    const groups = extractFileGroupsFromResult({
      output_files: 'ai-agent-os/output/report.xlsx'
    })

    expect(groups).toEqual([
      {
        label: 'Output Files',
        files: [
          {
            ref: 'ai-agent-os/output/report.xlsx',
            name: 'report.xlsx',
          }
        ]
      }
    ])
  })

  it('recursively extracts nested file ref strings from objects and arrays', () => {
    const groups = extractFileGroupsFromResult({
      data: {
        preview_files: 'ai-agent-os/output/preview.png'
      },
      variants: [
        {
          rendered_files: 'ai-agent-os/output/clip.mp4'
        }
      ]
    })

    expect(groups).toEqual([
      {
        label: 'Data / Preview Files',
        files: [
          {
            ref: 'ai-agent-os/output/preview.png',
            name: 'preview.png',
          }
        ]
      },
      {
        label: 'Variants / #1 / Rendered Files',
        files: [
          {
            ref: 'ai-agent-os/output/clip.mp4',
            name: 'clip.mp4',
          }
        ]
      }
    ])
  })

  it('supports JSON-string results with comma-separated refs', () => {
    const groups = extractFileGroupsFromResult(JSON.stringify({
      output_files: 'ai-agent-os/output/audio.wav,ai-agent-os/output/subtitle.srt'
    }))

    expect(groups).toEqual([
      {
        label: 'Output Files',
        files: [
          {
            ref: 'ai-agent-os/output/audio.wav',
            name: 'audio.wav',
          },
          {
            ref: 'ai-agent-os/output/subtitle.srt',
            name: 'subtitle.srt',
          }
        ]
      }
    ])
  })

  it('does not extract refs from non-json text results', () => {
    const groups = extractFileGroupsFromResult(`
      搜索结果：未传 keyword
      full_code_path: /system/official/python/execute.form
      - input_files: widget=files, accept=image/*,*/*
      - font /usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc
    `)

    expect(groups).toEqual([])
  })

  it('rejects file-like fields when the whole string is not refs only', () => {
    const groups = extractFileGroupsFromResult({
      output_files: '执行完成，文件: ai-agent-os/output/report.xlsx',
      preview_files: 'ai-agent-os/output/preview.png, image/*',
      file_name: 'report.xlsx'
    })

    expect(groups).toEqual([])
  })
})
