import { describe, expect, it } from 'vitest'
import { formatAcceptLabel, isAcceptAll, isFileAccepted } from './accept'

describe('upload accept matching', () => {
  it('treats wildcard accept values as any file', () => {
    const file = { name: 'archive.zip', type: 'application/zip' }

    expect(isAcceptAll('*')).toBe(true)
    expect(isAcceptAll('*/*')).toBe(true)
    expect(isFileAccepted(file, '*')).toBe(true)
    expect(isFileAccepted(file, '*/*')).toBe(true)
    expect(isFileAccepted(file, 'image/*,*/*')).toBe(true)
    expect(formatAcceptLabel('*/*')).toBe('任意格式')
  })

  it('matches extensions case-insensitively', () => {
    expect(isFileAccepted({ name: 'REPORT.PDF', type: '' }, '.pdf')).toBe(true)
    expect(isFileAccepted({ name: 'report.docx', type: '' }, '.pdf')).toBe(false)
  })

  it('matches exact and wildcard mime types', () => {
    expect(isFileAccepted({ name: 'photo.bin', type: 'image/jpeg' }, 'image/*')).toBe(true)
    expect(isFileAccepted({ name: 'photo.bin', type: 'image/jpeg' }, 'image/jpeg')).toBe(true)
    expect(isFileAccepted({ name: 'photo.bin', type: 'image/jpeg' }, 'application/pdf')).toBe(false)
  })
})
