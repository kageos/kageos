import { computed, ref, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Document, Files, Folder, Picture, VideoPlay } from '@element-plus/icons-vue'
import { Logger } from '@/core/utils/logger'
import type { FileItem } from '../filesWidgetTypes'

interface UseFilesPreviewAndActionsOptions {
  currentFiles: Ref<FileItem[]>
}

export function useFilesPreviewAndActions(options: UseFilesPreviewAndActionsOptions) {
  const previewVisible = ref(false)
  const previewImageUrl = ref('')
  const previewImageName = ref('')

  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
  }

  function isDirectAccessUrl(url: string | undefined): boolean {
    if (!url) return false
    return url.startsWith('http://') || url.startsWith('https://') || url.startsWith('/')
  }

  function isImageFile(file: FileItem): boolean {
    if (!file.name) return false
    const imageExtensions = ['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp', '.svg', '.ico']
    const fileName = file.name.toLowerCase()
    return imageExtensions.some(ext => fileName.endsWith(ext))
  }

  function canPreviewInBrowser(file: FileItem): boolean {
    if (!file.is_uploaded || !file.url) return false

    const fileName = (file.name || '').toLowerCase()
    const previewableExtensions = [
      '.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp', '.svg',
      '.mp4', '.avi', '.mov', '.wmv', '.flv', '.mkv', '.webm',
      '.pdf',
      '.txt', '.md', '.html', '.htm', '.css', '.js', '.json', '.xml', '.yaml', '.yml',
      '.log', '.ini', '.conf', '.sh', '.bat', '.py', '.go', '.java', '.cpp', '.c', '.h',
      '.vue', '.ts', '.tsx', '.jsx', '.sql'
    ]
    return previewableExtensions.some(ext => fileName.endsWith(ext))
  }

  function getFileIcon(fileName: string): any {
    const ext = fileName.split('.').pop()?.toLowerCase() || ''
    if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp'].includes(ext)) {
      return Picture
    }
    if (['mp4', 'avi', 'mov', 'wmv', 'flv', 'mkv', 'webm'].includes(ext)) {
      return VideoPlay
    }
    if (['zip', 'rar', '7z', 'tar', 'gz'].includes(ext)) {
      return Folder
    }
    if (['doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'pdf'].includes(ext)) {
      return Files
    }
    return Document
  }

  function getFileIconColor(fileName: string): string {
    const ext = fileName.split('.').pop()?.toLowerCase() || ''
    if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp'].includes(ext)) {
      return '#409EFF'
    }
    if (['mp4', 'avi', 'mov', 'wmv', 'flv', 'mkv', 'webm'].includes(ext)) {
      return '#F56C6C'
    }
    if (['pdf'].includes(ext)) {
      return '#E6A23C'
    }
    return '#909399'
  }

  function handlePreviewInNewWindow(file: FileItem): void {
    if (!canPreviewInBrowser(file) || !file.url) {
      return
    }

    const previewURL = isDirectAccessUrl(file.url)
      ? file.url
      : `/storage/api/v1/download/${encodeURIComponent(file.url)}`

    window.open(previewURL, '_blank')
  }

  const previewImageList = computed(() => {
    return options.currentFiles.value
      .filter((file: FileItem) => isImageFile(file) && file.is_uploaded && file.url)
      .map((file: FileItem) => file.url || '')
  })

  function getPreviewImageIndex(file: FileItem): number {
    return previewImageList.value.findIndex((url: string) => url === file.url)
  }

  async function getPreviewUrl(file: FileItem): Promise<string> {
    if (isDirectAccessUrl(file.url)) {
      return file.url!
    }

    return `/api/v1/storage/download/${encodeURIComponent(file.url)}`
  }

  async function handlePreviewImage(file: FileItem): Promise<void> {
    if (!isImageFile(file)) {
      ElMessage.warning('该文件不是图片格式，无法预览')
      return
    }

    try {
      previewImageName.value = file.name || '预览图片'
      previewImageUrl.value = await getPreviewUrl(file)
      previewVisible.value = true
    } catch (error: any) {
      Logger.error('[FilesWidget]', 'Preview failed', error)
      ElMessage.error(`预览失败: ${error.message}`)
    }
  }

  function handleClosePreview(): void {
    previewVisible.value = false
    if (previewImageUrl.value.startsWith('blob:')) {
      window.URL.revokeObjectURL(previewImageUrl.value)
    }
    previewImageUrl.value = ''
    previewImageName.value = ''
  }

  async function handleDownloadFile(file: FileItem): Promise<void> {
    try {
      const downloadURL = isDirectAccessUrl(file.url)
        ? file.url
        : `/storage/api/v1/download/${encodeURIComponent(file.url)}`

      const token = localStorage.getItem('token') || ''
      const res = await fetch(downloadURL, {
        headers: {
          'X-Token': token,
        },
      })

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({ msg: res.statusText }))
        throw new Error(errorData.msg || `下载失败: ${res.statusText}`)
      }

      const blob = await res.blob()
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = file.name || 'download'
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      ElMessage.success('下载成功')
    } catch (error: any) {
      Logger.error('[FilesWidget]', 'Download failed', error)
      ElMessage.error(`下载失败: ${error.message}`)
    }
  }

  return {
    previewVisible,
    previewImageUrl,
    previewImageName,
    previewImageList,
    formatSize,
    isImageFile,
    canPreviewInBrowser,
    getFileIcon,
    getFileIconColor,
    handlePreviewInNewWindow,
    getPreviewImageIndex,
    handlePreviewImage,
    handleClosePreview,
    handleDownloadFile
  }
}
