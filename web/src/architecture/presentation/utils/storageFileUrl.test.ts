import { describe, expect, it } from 'vitest'
import { normalizeStorageFileDisplayUrl } from './storageFileUrl'

describe('normalizeStorageFileDisplayUrl', () => {
  it('prefixes relative storage refs with the current origin', () => {
    expect(normalizeStorageFileDisplayUrl('kageos/output/preview.png')).toBe('http://localhost:3000/kageos/output/preview.png')
  })

  it('prefixes storage proxy paths with the current origin', () => {
    expect(normalizeStorageFileDisplayUrl('/kageos/output/preview.png')).toBe('http://localhost:3000/kageos/output/preview.png')
  })

  it('rewrites local MinIO URLs to the current origin and keeps query params', () => {
    expect(normalizeStorageFileDisplayUrl('http://localhost:9000/kageos/output/preview.png?sign=1')).toBe('http://localhost:3000/kageos/output/preview.png?sign=1')
  })

  it('keeps non-storage absolute URLs unchanged', () => {
    expect(normalizeStorageFileDisplayUrl('https://cdn.example.com/output/preview.png')).toBe('https://cdn.example.com/output/preview.png')
  })
})
