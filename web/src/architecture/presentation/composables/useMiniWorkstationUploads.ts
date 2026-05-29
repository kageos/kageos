import { ElMessage } from 'element-plus'
import { nextTick, ref, type Ref } from 'vue'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'
import { uploadFile, notifyUploadComplete, type UploadProgress } from '@/architecture/presentation/context/uploadContext'
import type { WorkspaceChatMessageFile } from '@/architecture/presentation/context/api/workspace'

const UPLOAD_ROUTER = 'workspace/chat'

type ClipboardFileTransfer = Pick<DataTransfer, 'files' | 'items'>

export interface UseMiniWorkstationUploadsOptions {
  fullCodePath: Ref<string>
  inputText: Ref<string>
  inputRef: Ref<HTMLTextAreaElement | undefined>
}

export function extractClipboardFiles(dataTransfer: ClipboardFileTransfer | null | undefined): File[] {
  if (!dataTransfer) {
    return []
  }

  const fileListFiles = Array.from(dataTransfer.files || [])
  if (fileListFiles.length > 0) {
    return fileListFiles
  }

  const itemFiles: File[] = []
  Array.from(dataTransfer.items || []).forEach((item) => {
    if (item.kind === 'file') {
      const file = item.getAsFile()
      if (file) {
        itemFiles.push(file)
      }
    }
  })

  return itemFiles
}

export function useMiniWorkstationUploads(options: UseMiniWorkstationUploadsOptions) {
  const { fullCodePath, inputText, inputRef } = options
  const authStore = useAuthStore()

  const attachedFiles = ref<WorkspaceChatMessageFile[]>([])
  const uploading = ref(false)
  const dragOver = ref(false)
  let dragLeaveTimer: ReturnType<typeof setTimeout> | null = null

  async function onFileChange(uploadFileObj: { raw?: File }) {
    const file = uploadFileObj?.raw
    if (!file || !fullCodePath.value) {
      return
    }

    uploading.value = true
    try {
      const uploadResult = await uploadFile(UPLOAD_ROUTER, file, (_progress: UploadProgress) => {})
      if (!uploadResult.fileInfo) {
        throw new Error('上传失败')
      }

      const completeResult = await notifyUploadComplete({
        key: uploadResult.fileInfo.key,
        bucket: uploadResult.fileInfo.bucket,
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

      attachedFiles.value.push({
        ref: completeResult.ref || uploadResult.fileInfo.ref,
        bucket: completeResult.bucket || uploadResult.fileInfo.bucket,
        key: uploadResult.fileInfo.key,
        name: completeResult.file_name,
        source_name: file.name,
        storage: completeResult.storage || uploadResult.storage,
        hash: completeResult.hash || uploadResult.fileInfo.hash || '',
        size: completeResult.file_size,
        upload_ts: Math.floor(Date.now() / 1000),
        is_uploaded: true,
        download_url: completeResult.download_url,
        server_download_url: completeResult.server_download_url,
        upload_user: authStore.userName || undefined
      })
      ElMessage.success(`已添加：${file.name}`)
    } catch (error: any) {
      ElMessage.error(error?.message || '上传失败')
    } finally {
      uploading.value = false
    }
  }

  async function onPaste(event: ClipboardEvent) {
    if (event.defaultPrevented) {
      return
    }

    const files = extractClipboardFiles(event.clipboardData)
    if (files.length === 0) {
      return
    }

    event.preventDefault()
    event.stopPropagation()

    if (!fullCodePath.value) {
      ElMessage.warning('请先选择目录后再粘贴文件')
      return
    }

    for (const file of files) {
      await onFileChange({ raw: file })
    }
  }

  function removeFile(index: number) {
    attachedFiles.value.splice(index, 1)
  }

  function onDragOver(_event: DragEvent) {
    if (dragLeaveTimer) {
      clearTimeout(dragLeaveTimer)
      dragLeaveTimer = null
    }
    dragOver.value = true
  }

  function onDragLeave(_event: DragEvent) {
    if (dragLeaveTimer) {
      clearTimeout(dragLeaveTimer)
    }
    dragLeaveTimer = setTimeout(() => {
      dragOver.value = false
    }, 80)
  }

  async function onDrop(event: DragEvent) {
    dragOver.value = false
    const dataTransfer = event.dataTransfer
    if (!dataTransfer) {
      return
    }

    if (dataTransfer.types.includes('application/x-workspace-node')) {
      try {
        const raw = dataTransfer.getData('application/x-workspace-node')
        const payload = raw ? JSON.parse(raw) as { type?: string; full_code_path?: string; name?: string } : null
        if (payload?.full_code_path) {
          const label = payload.type === 'package' ? '目录' : '函数'
          const name = payload.name || payload.full_code_path.split('/').pop() || payload.full_code_path
          inputText.value = `请处理以下${label}：${name}（${payload.full_code_path}）`
          await nextTick()
          inputRef.value?.focus()
        }
      } catch {
        // ignore parse error
      }
      return
    }

    const files = dataTransfer.files
    if (!files?.length || !fullCodePath.value) {
      return
    }

    for (const file of Array.from(files)) {
      await onFileChange({ raw: file })
    }
  }

  return {
    attachedFiles,
    uploading,
    dragOver,
    onFileChange,
    removeFile,
    onDragOver,
    onDragLeave,
    onDrop,
    onPaste
  }
}
