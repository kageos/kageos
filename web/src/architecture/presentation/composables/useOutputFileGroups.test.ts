import { describe, expect, it } from 'vitest'
import { extractFileGroupsFromResult } from './useOutputFileGroups'

describe('extractFileGroupsFromResult', () => {
  it('extracts a root-level files object and normalizes server_url to url', () => {
    const groups = extractFileGroupsFromResult({
      files: [
        {
          source_name: 'report.xlsx',
          server_url: 'https://example.com/report.xlsx'
        }
      ]
    })

    expect(groups).toEqual([
      {
        label: 'Output Files',
        files: [
          {
            source_name: 'report.xlsx',
            server_url: 'https://example.com/report.xlsx',
            url: 'https://example.com/report.xlsx'
          }
        ]
      }
    ])
  })

  it('recursively extracts nested file groups from objects and arrays', () => {
    const groups = extractFileGroupsFromResult({
      data: {
        preview_files: {
          files: [
            {
              name: 'preview.png',
              url: 'https://example.com/preview.png'
            }
          ]
        }
      },
      variants: [
        {
          rendered_files: {
            files: [
              {
                name: 'clip.mp4',
                url: 'https://example.com/clip.mp4'
              }
            ]
          }
        }
      ]
    })

    expect(groups).toEqual([
      {
        label: 'Data / Preview Files',
        files: [
          {
            name: 'preview.png',
            url: 'https://example.com/preview.png'
          }
        ]
      },
      {
        label: 'Variants / #1 / Rendered Files',
        files: [
          {
            name: 'clip.mp4',
            url: 'https://example.com/clip.mp4'
          }
        ]
      }
    ])
  })

  it('supports JSON-string results and skips invalid file entries', () => {
    const groups = extractFileGroupsFromResult(JSON.stringify({
      output_files: {
        files: [
          null,
          {
            name: 'audio.wav',
            url: 'https://example.com/audio.wav'
          },
          {
            name: 'broken-item'
          }
        ]
      }
    }))

    expect(groups).toEqual([
      {
        label: 'Output Files',
        files: [
          {
            name: 'audio.wav',
            url: 'https://example.com/audio.wav'
          }
        ]
      }
    ])
  })
})
