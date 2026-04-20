import { describe, expect, it } from 'vitest'
import { getFileDisplayUrl, normalizeFileDisplayUrl } from './fileDisplayUrl'

describe('fileDisplayUrl', () => {
  it('keeps relative storage proxy paths unchanged', () => {
    expect(normalizeFileDisplayUrl('/ai-agent-os/demo/image.png')).toBe('/ai-agent-os/demo/image.png')
  })

  it('keeps absolute browser urls unchanged', () => {
    expect(normalizeFileDisplayUrl('https://cdn.example.com/demo/image.png')).toBe('https://cdn.example.com/demo/image.png')
  })

  it('does not turn storage refs into proxy download urls', () => {
    expect(normalizeFileDisplayUrl('demo/path/image 1.png')).toBe('')
  })

  it('prefers browser download url over server download url', () => {
    expect(getFileDisplayUrl({
      download_url: '/ai-agent-os/demo/image.png',
      server_download_url: 'http://host.containers.internal:9000/ai-agent-os/demo/image.png?sign=1'
    } as any)).toBe('/ai-agent-os/demo/image.png')
  })

  it('falls back to server download url when browser download url is empty', () => {
    expect(getFileDisplayUrl({
      download_url: '',
      server_download_url: 'http://host.containers.internal:9000/ai-agent-os/demo/image.png?sign=1'
    } as any)).toBe('http://host.containers.internal:9000/ai-agent-os/demo/image.png?sign=1')
  })
})
