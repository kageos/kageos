export interface ZipEntryInput {
  name: string
  data: Blob | ArrayBuffer | Uint8Array
  lastModified?: number
}

type ZipBytes = Uint8Array<ArrayBuffer>

interface PreparedZipEntry {
  nameBytes: ZipBytes
  data: ZipBytes
  crc: number
  time: number
  date: number
  localHeaderOffset: number
}

const UTF8_FLAG = 0x0800
const STORE_METHOD = 0
const LOCAL_FILE_HEADER_SIGNATURE = 0x04034b50
const CENTRAL_DIRECTORY_SIGNATURE = 0x02014b50
const END_OF_CENTRAL_DIRECTORY_SIGNATURE = 0x06054b50

let crcTable: Uint32Array | null = null

export async function createZipBlob(entries: ZipEntryInput[]): Promise<Blob> {
  if (entries.length === 0) {
    return new Blob([], { type: 'application/zip' })
  }

  const encoder = new TextEncoder()
  const prepared: PreparedZipEntry[] = []
  const parts: BlobPart[] = []
  let offset = 0

  for (const entry of entries) {
    const data = await toUint8Array(entry.data)
    const nameBytes = encoder.encode(entry.name)
    const { time, date } = toDosDateTime(entry.lastModified)
    const item: PreparedZipEntry = {
      nameBytes,
      data,
      crc: crc32(data),
      time,
      date,
      localHeaderOffset: offset,
    }
    const localHeader = buildLocalFileHeader(item)
    parts.push(localHeader, data)
    offset += localHeader.byteLength + data.byteLength
    prepared.push(item)
  }

  const centralDirectoryOffset = offset
  for (const item of prepared) {
    const centralHeader = buildCentralDirectoryHeader(item)
    parts.push(centralHeader)
    offset += centralHeader.byteLength
  }

  const centralDirectorySize = offset - centralDirectoryOffset
  parts.push(buildEndOfCentralDirectory(prepared.length, centralDirectorySize, centralDirectoryOffset))

  return new Blob(parts, { type: 'application/zip' })
}

async function toUint8Array(data: Blob | ArrayBuffer | Uint8Array): Promise<ZipBytes> {
  if (ArrayBuffer.isView(data)) {
    const bytes = new Uint8Array(data.byteLength)
    bytes.set(data)
    return bytes
  }
  if (data instanceof ArrayBuffer) {
    return new Uint8Array(data)
  }
  return new Uint8Array(await data.arrayBuffer())
}

function buildLocalFileHeader(entry: PreparedZipEntry): ZipBytes {
  const bytes = new Uint8Array(30 + entry.nameBytes.byteLength)
  const view = new DataView(bytes.buffer)
  view.setUint32(0, LOCAL_FILE_HEADER_SIGNATURE, true)
  view.setUint16(4, 20, true)
  view.setUint16(6, UTF8_FLAG, true)
  view.setUint16(8, STORE_METHOD, true)
  view.setUint16(10, entry.time, true)
  view.setUint16(12, entry.date, true)
  view.setUint32(14, entry.crc, true)
  view.setUint32(18, entry.data.byteLength, true)
  view.setUint32(22, entry.data.byteLength, true)
  view.setUint16(26, entry.nameBytes.byteLength, true)
  view.setUint16(28, 0, true)
  bytes.set(entry.nameBytes, 30)
  return bytes
}

function buildCentralDirectoryHeader(entry: PreparedZipEntry): ZipBytes {
  const bytes = new Uint8Array(46 + entry.nameBytes.byteLength)
  const view = new DataView(bytes.buffer)
  view.setUint32(0, CENTRAL_DIRECTORY_SIGNATURE, true)
  view.setUint16(4, 20, true)
  view.setUint16(6, 20, true)
  view.setUint16(8, UTF8_FLAG, true)
  view.setUint16(10, STORE_METHOD, true)
  view.setUint16(12, entry.time, true)
  view.setUint16(14, entry.date, true)
  view.setUint32(16, entry.crc, true)
  view.setUint32(20, entry.data.byteLength, true)
  view.setUint32(24, entry.data.byteLength, true)
  view.setUint16(28, entry.nameBytes.byteLength, true)
  view.setUint16(30, 0, true)
  view.setUint16(32, 0, true)
  view.setUint16(34, 0, true)
  view.setUint16(36, 0, true)
  view.setUint32(38, 0, true)
  view.setUint32(42, entry.localHeaderOffset, true)
  bytes.set(entry.nameBytes, 46)
  return bytes
}

function buildEndOfCentralDirectory(entryCount: number, centralDirectorySize: number, centralDirectoryOffset: number): ZipBytes {
  const bytes = new Uint8Array(22)
  const view = new DataView(bytes.buffer)
  view.setUint32(0, END_OF_CENTRAL_DIRECTORY_SIGNATURE, true)
  view.setUint16(4, 0, true)
  view.setUint16(6, 0, true)
  view.setUint16(8, entryCount, true)
  view.setUint16(10, entryCount, true)
  view.setUint32(12, centralDirectorySize, true)
  view.setUint32(16, centralDirectoryOffset, true)
  view.setUint16(20, 0, true)
  return bytes
}

function toDosDateTime(value?: number): { time: number; date: number } {
  const source = value == null ? new Date() : new Date(value)
  const year = Math.min(Math.max(source.getFullYear(), 1980), 2107)
  const month = source.getMonth() + 1
  const day = source.getDate()
  const hours = source.getHours()
  const minutes = source.getMinutes()
  const seconds = Math.floor(source.getSeconds() / 2)

  return {
    time: (hours << 11) | (minutes << 5) | seconds,
    date: ((year - 1980) << 9) | (month << 5) | day,
  }
}

function crc32(data: Uint8Array): number {
  const table = getCrcTable()
  let crc = 0xffffffff
  for (const byte of data) {
    crc = (crc >>> 8) ^ table[(crc ^ byte) & 0xff]!
  }
  return (crc ^ 0xffffffff) >>> 0
}

function getCrcTable(): Uint32Array {
  if (crcTable) {
    return crcTable
  }

  const table = new Uint32Array(256)
  for (let i = 0; i < 256; i += 1) {
    let c = i
    for (let bit = 0; bit < 8; bit += 1) {
      c = (c & 1) ? (0xedb88320 ^ (c >>> 1)) : (c >>> 1)
    }
    table[i] = c >>> 0
  }
  crcTable = table
  return table
}
