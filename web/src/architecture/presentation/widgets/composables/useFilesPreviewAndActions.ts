import { computed, ref, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Document, Files, Folder, Picture, VideoPlay } from '@element-plus/icons-vue'
import { Logger } from '@/core/utils/logger'
import type { FileItem } from '../filesWidgetTypes'
import { getFileDisplayUrl } from '../utils/fileDisplayUrl'

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

  function isImageFile(file: FileItem): boolean {
    if (file.content_type?.toLowerCase().startsWith('image/')) return true
    const name = file.name || file.source_name || ''
    if (!name) return false
    const imageExtensions = ['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp', '.svg', '.ico', '.avif']
    const fileName = name.toLowerCase()
    return imageExtensions.some(ext => fileName.endsWith(ext))
  }

  function canPreviewInBrowser(file: FileItem): boolean {
    if (!file.is_uploaded || !getFileDisplayUrl(file)) return false

    const fileName = (file.name || '').toLowerCase()
    const previewableExtensions = [
      '.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp', '.svg', '.ico', '.avif',
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
    if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico', 'avif'].includes(ext)) {
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
    if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico', 'avif'].includes(ext)) {
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
    const displayUrl = getFileDisplayUrl(file)
    if (!canPreviewInBrowser(file) || !displayUrl) {
      return
    }

    window.open(displayUrl, '_blank')
  }

  const previewImageList = computed(() => {
    return options.currentFiles.value
      .filter((file: FileItem) => isImageFile(file) && file.is_uploaded && !!getFileDisplayUrl(file))
      .map((file: FileItem) => getFileDisplayUrl(file))
  })

  function getPreviewImageIndex(file: FileItem): number {
    return previewImageList.value.findIndex((url: string) => url === getFileDisplayUrl(file))
  }

  async function handlePreviewImage(file: FileItem): Promise<void> {
    if (!isImageFile(file)) {
      ElMessage.warning('该文件不是图片格式，无法预览')
      return
    }

    try {
      const displayUrl = getFileDisplayUrl(file)
      if (!displayUrl) {
        throw new Error('文件地址缺失')
      }
      previewImageName.value = file.name || '预览图片'
      previewImageUrl.value = displayUrl
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
      const downloadURL = getFileDisplayUrl(file)
      if (!downloadURL) {
        throw new Error('文件地址缺失')
      }

      const link = document.createElement('a')
      link.href = downloadURL
      link.download = file.name || 'download'
      link.rel = 'noopener'
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)

      ElMessage.success('已开始下载')
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
    getFileDisplayUrl,
    getFileIcon,
    getFileIconColor,
    handlePreviewInNewWindow,
    getPreviewImageIndex,
    handlePreviewImage,
    handleClosePreview,
    handleDownloadFile
  }
}
