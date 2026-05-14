import { describe, expect, it } from 'vitest'
import { createZipBlob } from './downloadZip'

describe('createZipBlob', () => {
  it('creates a zip containing file names and stored bytes', async () => {
    const zip = await createZipBlob([
      { name: 'report.txt', data: new TextEncoder().encode('hello') },
      { name: 'images/page-01.png', data: new TextEncoder().encode('png-data') },
    ])

    const bytes = new Uint8Array(await readBlob(zip))
    const text = new TextDecoder().decode(bytes)
    const view = new DataView(bytes.buffer)

    expect(zip.type).toBe('application/zip')
    expect(view.getUint32(0, true)).toBe(0x04034b50)
    expect(text).toContain('report.txt')
    expect(text).toContain('hello')
    expect(text).toContain('images/page-01.png')
    expect(text).toContain('png-data')
    expect(view.getUint32(bytes.byteLength - 22, true)).toBe(0x06054b50)
  })
})

function readBlob(blob: Blob): Promise<ArrayBuffer> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as ArrayBuffer)
    reader.onerror = () => reject(reader.error)
    reader.readAsArrayBuffer(blob)
  })
}
