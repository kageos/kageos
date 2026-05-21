import { describe, expect, it } from 'vitest'
import { extractFileGroupsFromResult } from './useOutputFileGroups'

describe('extractFileGroupsFromResult', () => {
  it('extracts a declared root-level file field', () => {
    const groups = extractFileGroupsFromResult({
      output_files: 'kageos/output/report.xlsx'
    }, {
      display_file_fields: ['output_files']
    })

    expect(groups).toEqual([
      {
        label: 'Output Files',
        files: [
          {
            ref: 'kageos/output/report.xlsx',
            name: 'report.xlsx',
          }
        ]
      }
    ])
  })

  it('extracts a declared nested file field', () => {
    const groups = extractFileGroupsFromResult({
      data: {
        preview_files: 'kageos/output/preview.png'
      },
      variants: [
        {
          rendered_files: 'kageos/output/clip.mp4'
        }
      ]
    }, {
      display_file_fields: ['data.preview_files']
    })

    expect(groups).toEqual([
      {
        label: 'Data / Preview Files',
        files: [
          {
            ref: 'kageos/output/preview.png',
            name: 'preview.png',
          }
        ]
      }
    ])
  })

  it('supports JSON-string results with declared comma-separated refs', () => {
    const groups = extractFileGroupsFromResult(JSON.stringify({
      output_files: 'kageos/output/audio.wav,kageos/output/subtitle.srt'
    }), {
      display_file_fields: ['output_files']
    })

    expect(groups).toEqual([
      {
        label: 'Output Files',
        files: [
          {
            ref: 'kageos/output/audio.wav',
            name: 'audio.wav',
          },
          {
            ref: 'kageos/output/subtitle.srt',
            name: 'subtitle.srt',
          }
        ]
      }
    ])
  })

  it('does not extract refs without metadata', () => {
    const groups = extractFileGroupsFromResult({
      output_files: 'kageos/output/report.xlsx'
    })

    expect(groups).toEqual([])
  })

  it('does not extract refs from non-json text results', () => {
    const groups = extractFileGroupsFromResult(`
      搜索结果：未传 keyword
      full_code_path: /system/tools/runtime/python.form
      - input_files: widget=files, accept=image/*,*/*
      - font /usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc
    `, {
      display_file_fields: ['output_files']
    })

    expect(groups).toEqual([])
  })

  it('rejects file-like fields when the whole string is not refs only', () => {
    const groups = extractFileGroupsFromResult({
      output_files: '执行完成，文件: kageos/output/report.xlsx',
      preview_files: 'kageos/output/preview.png, image/*',
      file_name: 'report.xlsx'
    }, {
      display_file_fields: ['output_files', 'preview_files', 'file_name']
    })

    expect(groups).toEqual([])
  })

  it('extracts only fields declared by metadata', () => {
    const groups = extractFileGroupsFromResult({
      output_files: 'kageos/output/report.xlsx',
      preview_files: 'kageos/output/preview.png'
    }, {
      display_file_fields: ['output_files']
    })

    expect(groups).toEqual([
      {
        label: 'Output Files',
        files: [
          {
            ref: 'kageos/output/report.xlsx',
            name: 'report.xlsx',
          }
        ]
      }
    ])
  })

  it('does not fall back to guessed fields when metadata is present', () => {
    const groups = extractFileGroupsFromResult({
      output_files: 'kageos/output/report.xlsx'
    }, {
      display_file_fields: []
    })

    expect(groups).toEqual([])
  })
})
