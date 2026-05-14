export type FilePreviewKind = 'image' | 'video'

export interface GeneratedFilePreview {
  file: File
  kind: FilePreviewKind
}

const PREVIEW_MAX_EDGE = 480
const PREVIEW_TIMEOUT_MS = 8000
const IMAGE_EXTENSIONS = new Set(['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg', 'avif', 'heic', 'heif'])
const VIDEO_EXTENSIONS = new Set(['mp4', 'webm', 'ogv', 'ogg', 'mov', 'm4v'])

export async function generateFilePreview(file: File): Promise<GeneratedFilePreview | null> {
  const kind = detectFilePreviewKind(file)
  if (kind === 'image') {
    return generateImagePreview(file)
  }
  if (kind === 'video') {
    return generateVideoPreview(file)
  }
  return null
}

export function detectFilePreviewKind(file: Pick<File, 'name' | 'type'>): FilePreviewKind | null {
  const contentType = (file.type || '').trim().toLowerCase()
  if (contentType.startsWith('image/')) {
    return 'image'
  }
  if (contentType.startsWith('video/')) {
    return 'video'
  }

  const extension = fileExtension(file.name)
  if (IMAGE_EXTENSIONS.has(extension)) {
    return 'image'
  }
  if (VIDEO_EXTENSIONS.has(extension)) {
    return 'video'
  }
  return null
}

async function generateImagePreview(file: File): Promise<GeneratedFilePreview | null> {
  const image = await loadImage(file)
  try {
    const canvas = drawToPreviewCanvas(image.naturalWidth, image.naturalHeight, (ctx, width, height) => {
      ctx.drawImage(image, 0, 0, width, height)
    })
    const { blob, extension } = await canvasToPreviewBlob(canvas)
    return {
      file: new File([blob], previewFileName(file.name, extension), { type: blob.type }),
      kind: 'image',
    }
  } finally {
    image.src = ''
  }
}

async function generateVideoPreview(file: File): Promise<GeneratedFilePreview | null> {
  const url = URL.createObjectURL(file)
  const video = document.createElement('video')
  video.preload = 'metadata'
  video.muted = true
  video.playsInline = true

  try {
    video.src = url
    await waitForMediaEvent(video, 'loadedmetadata')

    const duration = Number.isFinite(video.duration) ? video.duration : 0
    const safeMaxSeek = Math.max(0, duration - 0.05)
    const seekTime = duration > 0 ? Math.min(1, Math.max(0.1, duration * 0.1), safeMaxSeek) : 0

    if (seekTime > 0) {
      video.currentTime = seekTime
      await waitForMediaEvent(video, 'seeked')
    } else if (video.readyState < video.HAVE_CURRENT_DATA) {
      await waitForMediaEvent(video, 'loadeddata')
    }

    if (!video.videoWidth || !video.videoHeight) {
      return null
    }

    const canvas = drawToPreviewCanvas(video.videoWidth, video.videoHeight, (ctx, width, height) => {
      ctx.drawImage(video, 0, 0, width, height)
    })
    const { blob, extension } = await canvasToPreviewBlob(canvas)
    return {
      file: new File([blob], previewFileName(file.name, extension), { type: blob.type }),
      kind: 'video',
    }
  } finally {
    video.pause()
    video.removeAttribute('src')
    video.load()
    URL.revokeObjectURL(url)
  }
}

function drawToPreviewCanvas(
  sourceWidth: number,
  sourceHeight: number,
  draw: (ctx: CanvasRenderingContext2D, width: number, height: number) => void
): HTMLCanvasElement {
  const scale = Math.min(1, PREVIEW_MAX_EDGE / Math.max(sourceWidth, sourceHeight))
  const width = Math.max(1, Math.round(sourceWidth * scale))
  const height = Math.max(1, Math.round(sourceHeight * scale))
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height

  const ctx = canvas.getContext('2d')
  if (!ctx) {
    throw new Error('浏览器不支持生成文件预览')
  }

  draw(ctx, width, height)
  return canvas
}

async function canvasToPreviewBlob(canvas: HTMLCanvasElement): Promise<{ blob: Blob; extension: string }> {
  const webp = await canvasToBlob(canvas, 'image/webp', 0.82)
  if (webp) {
    return { blob: webp, extension: 'webp' }
  }
  const jpeg = await canvasToBlob(canvas, 'image/jpeg', 0.82)
  if (jpeg) {
    return { blob: jpeg, extension: 'jpg' }
  }
  throw new Error('浏览器无法生成文件预览')
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob | null> {
  return new Promise(resolve => {
    canvas.toBlob(resolve, type, quality)
  })
}

function loadImage(file: File): Promise<HTMLImageElement> {
  const url = URL.createObjectURL(file)
  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(() => {
      URL.revokeObjectURL(url)
      reject(new Error('图片预览生成超时'))
    }, PREVIEW_TIMEOUT_MS)

    const image = new Image()
    image.onload = () => {
      window.clearTimeout(timer)
      URL.revokeObjectURL(url)
      resolve(image)
    }
    image.onerror = () => {
      window.clearTimeout(timer)
      URL.revokeObjectURL(url)
      reject(new Error('图片无法解码'))
    }
    image.src = url
  })
}

function waitForMediaEvent(media: HTMLMediaElement, eventName: keyof HTMLMediaElementEventMap): Promise<void> {
  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(() => {
      cleanup()
      reject(new Error('视频封面生成超时'))
    }, PREVIEW_TIMEOUT_MS)

    const cleanup = () => {
      window.clearTimeout(timer)
      media.removeEventListener(eventName, onReady)
      media.removeEventListener('error', onError)
    }
    const onReady = () => {
      cleanup()
      resolve()
    }
    const onError = () => {
      cleanup()
      reject(new Error('视频无法解码'))
    }

    media.addEventListener(eventName, onReady, { once: true })
    media.addEventListener('error', onError, { once: true })
  })
}

function previewFileName(fileName: string, extension: string): string {
  const clean = fileName.trim() || 'file'
  const dotIndex = clean.lastIndexOf('.')
  const baseName = dotIndex > 0 ? clean.slice(0, dotIndex) : clean
  return `${baseName}.preview.${extension}`
}

function fileExtension(fileName: string): string {
  const clean = fileName.trim().toLowerCase()
  const dotIndex = clean.lastIndexOf('.')
  if (dotIndex < 0 || dotIndex === clean.length - 1) {
    return ''
  }
  return clean.slice(dotIndex + 1)
}
