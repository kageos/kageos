import { ref, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import { uploadFile, notifyUploadComplete } from '@/utils/upload'
import type { WorkspaceChatMessageFile } from '@/api/workspace'
import type { UploadProgress } from '@/utils/upload/types'
import { useAuthStore } from '@/stores/auth'

const WORKSPACE_CHAT_UPLOAD_ROUTER = 'workspace/chat'

function toPathOnlyUrl(url: string): string {
  if (!url) return url
  try {
    if (url.startsWith('http://') || url.startsWith('https://')) {
      const parsed = new URL(url)
      return parsed.pathname + parsed.search + parsed.hash
    }
  } catch {
    // ignore
  }
  return url
}

export function useWorkstationChatAttachments(fullCodePath: Ref<string>) {
  const authStore = useAuthStore()
  const attachedFiles = ref<WorkspaceChatMessageFile[]>([])
  const uploading = ref(false)
  const isDraggingOver = ref(false)

  async function addFileAsAttachment(file: File): Promise<void> {
    if (!file || !fullCodePath.value) return

    const uploadResult = await uploadFile(
      WORKSPACE_CHAT_UPLOAD_ROUTER,
      file,
      (_progress: UploadProgress) => {}
    )
    if (!uploadResult.fileInfo) {
      throw new Error('上传失败')
    }

    const completeResult = await notifyUploadComplete({
      key: uploadResult.fileInfo.key,
      success: true,
      router: uploadResult.fileInfo.router,
      file_name: uploadResult.fileInfo.file_name,
      file_size: uploadResult.fileInfo.file_size,
      content_type: uploadResult.fileInfo.content_type,
      hash: uploadResult.fileInfo.hash,
      upload_user: authStore.userName || undefined
    })
    if (!completeResult?.download_url) {
      throw new Error('获取下载地址失败')
    }

    const item: WorkspaceChatMessageFile = {
      name: completeResult.file_name,
      source_name: file.name,
      storage: completeResult.storage || uploadResult.storage,
      hash: completeResult.hash || uploadResult.fileInfo.hash || '',
      size: completeResult.file_size,
      upload_ts: Math.floor(Date.now() / 1000),
      is_uploaded: true,
      url: toPathOnlyUrl(completeResult.download_url),
      server_url: completeResult.server_download_url,
      upload_user: authStore.userName || undefined
    }
    attachedFiles.value = [...attachedFiles.value, item]
    ElMessage.success(`已添加：${file.name}`)
  }

  async function onAttachFileChange(uploadFileObj: { raw?: File; name?: string }) {
    const file = uploadFileObj?.raw
    if (!file || !fullCodePath.value) return
    uploading.value = true
    try {
      await addFileAsAttachment(file)
    } catch (error: unknown) {
      console.error('[WorkstationChat] 上传失败:', error)
      ElMessage.error(error instanceof Error ? error.message : '上传失败')
    } finally {
      uploading.value = false
    }
  }

  async function onDropFiles(event: DragEvent) {
    isDraggingOver.value = false
    if (!fullCodePath.value || uploading.value) return
    const files = event.dataTransfer?.files
    if (!files?.length) return

    uploading.value = true
    try {
      for (const file of Array.from(files)) {
        if (!file.name) continue
        try {
          await addFileAsAttachment(file)
        } catch (error: unknown) {
          console.error('[WorkstationChat] 拖拽上传失败:', file.name, error)
          ElMessage.error(`${file.name} 上传失败：${error instanceof Error ? error.message : '未知错误'}`)
        }
      }
    } finally {
      uploading.value = false
    }
  }

  function removeAttachedFile(index: number) {
    attachedFiles.value = attachedFiles.value.filter((_, i) => i !== index)
  }

  function setAttachedFiles(files: WorkspaceChatMessageFile[]) {
    attachedFiles.value = [...files]
  }

  function clearAttachedFiles() {
    attachedFiles.value = []
  }

  return {
    attachedFiles,
    uploading,
    isDraggingOver,
    onAttachFileChange,
    onDropFiles,
    removeAttachedFile,
    setAttachedFiles,
    clearAttachedFiles
  }
}
