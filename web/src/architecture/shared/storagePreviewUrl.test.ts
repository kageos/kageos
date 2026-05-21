import { describe, expect, it } from 'vitest'
import { deriveThumbnailPreviewUrl } from './storagePreviewUrl'

describe('deriveThumbnailPreviewUrl', () => {
  it('derives thumbnail url after the full original file name', () => {
    expect(deriveThumbnailPreviewUrl('/kageos/a/b/file.png')).toBe('/kageos/a/b/file.png.thumb.webp')
    expect(deriveThumbnailPreviewUrl('kageos/a/b/file.mp4')).toBe('kageos/a/b/file.mp4.thumb.webp')
  })

  it('preserves query and hash', () => {
    expect(deriveThumbnailPreviewUrl('/kageos/a/b/file.png?x=1#top')).toBe('/kageos/a/b/file.png.thumb.webp?x=1#top')
  })

  it('keeps existing thumbnail urls stable', () => {
    expect(deriveThumbnailPreviewUrl('/kageos/a/b/file.thumb.webp')).toBe('/kageos/a/b/file.thumb.webp')
  })
})
