import { describe, expect, it } from 'vitest'
import { extractClipboardFiles } from './useMiniWorkstationUploads'

function file(name: string, options: { type?: string; content?: string } = {}) {
  return new File([options.content || 'x'], name, {
    type: options.type || 'text/plain',
    lastModified: 123
  })
}

describe('extractClipboardFiles', () => {
  it('extracts files from clipboard file lists', () => {
    const image = file('clip.png', { type: 'image/png' })
    const doc = file('notes.txt')

    const files = extractClipboardFiles({
      files: [image, doc],
      items: []
    } as unknown as DataTransfer)

    expect(files).toEqual([image, doc])
  })

  it('extracts file items and keeps text-only clipboard data alone', () => {
    const image = file('screenshot.png', { type: 'image/png' })

    const files = extractClipboardFiles({
      files: [],
      items: [
        { kind: 'string', getAsFile: () => null },
        { kind: 'file', getAsFile: () => image }
      ]
    } as unknown as DataTransfer)

    expect(files).toEqual([image])
    expect(extractClipboardFiles({
      files: [],
      items: [{ kind: 'string', getAsFile: () => null }]
    } as unknown as DataTransfer)).toEqual([])
  })

  it('prefers file lists when both clipboard surfaces expose files', () => {
    const image = file('clip.png', { type: 'image/png' })

    const files = extractClipboardFiles({
      files: [image],
      items: [{ kind: 'file', getAsFile: () => image }]
    } as unknown as DataTransfer)

    expect(files).toEqual([image])
  })
})
