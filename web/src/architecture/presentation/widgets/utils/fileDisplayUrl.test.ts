import { describe, expect, it } from 'vitest'
import { getFileDisplayUrl, normalizeFileDisplayUrl } from './fileDisplayUrl'

describe('fileDisplayUrl', () => {
  it('keeps relative storage proxy paths unchanged', () => {
    expect(normalizeFileDisplayUrl('/ai-agent-os/demo/image.png')).toBe('/ai-agent-os/demo/image.png')
  })

  it('keeps absolute browser urls unchanged', () => {
    expect(normalizeFileDisplayUrl('https://cdn.example.com/demo/image.png')).toBe('https://cdn.example.com/demo/image.png')
  })

  it('converts plain storage keys to download api urls', () => {
    expect(normalizeFileDisplayUrl('demo/path/image 1.png')).toBe('/storage/api/v1/download/demo%2Fpath%2Fimage%201.png')
  })

  it('prefers file url over server url', () => {
    expect(getFileDisplayUrl({
      url: '/ai-agent-os/demo/image.png',
      server_url: 'http://host.containers.internal:9000/ai-agent-os/demo/image.png?sign=1'
    } as any)).toBe('/ai-agent-os/demo/image.png')
  })

  it('falls back to server url when file url is empty', () => {
    expect(getFileDisplayUrl({
      url: '',
      server_url: 'http://host.containers.internal:9000/ai-agent-os/demo/image.png?sign=1'
    } as any)).toBe('http://host.containers.internal:9000/ai-agent-os/demo/image.png?sign=1')
  })
})
