import { describe, expect, it } from 'vitest'
import { detectFilePreviewKind } from './filePreview'

describe('detectFilePreviewKind', () => {
  it('detects images and videos by mime type', () => {
    expect(detectFilePreviewKind({ name: 'ignored.bin', type: 'image/jpeg' })).toBe('image')
    expect(detectFilePreviewKind({ name: 'ignored.bin', type: 'video/mp4' })).toBe('video')
  })

  it('falls back to file extension when mime type is missing or generic', () => {
    expect(detectFilePreviewKind({ name: 'photo.JPG', type: '' })).toBe('image')
    expect(detectFilePreviewKind({ name: 'clip.MP4', type: 'application/octet-stream' })).toBe('video')
  })

  it('ignores unsupported files', () => {
    expect(detectFilePreviewKind({ name: 'contract.pdf', type: 'application/pdf' })).toBeNull()
  })
})
