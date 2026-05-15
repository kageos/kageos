import { computed, ref, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { ToolResultMetadata } from '@/architecture/presentation/context/api/workspace'
import { extractFileGroupsFromResult, type OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'
import { extractAllDisplayFields, type OutputDisplayField } from '@/architecture/presentation/composables/useOutputDisplayFields'
import type { ChatMessage } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import { normalizeStorageFileDisplayUrl } from '@/architecture/presentation/utils/storageFileUrl'

export interface FilePanelItem {
  name: string
  href: string
  source: 'upload' | 'output'
}

const IMAGE_EXTS = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.svg'])

type FileGroupToolCall = { result?: string; result_data?: unknown; metadata?: ToolResultMetadata }
type DisplayFieldToolCall = { arguments?: string; result?: string; result_data?: unknown }

export { normalizeStorageFileDisplayUrl as normalizeMiniFileDisplayUrl }

export function useMiniWorkstationPanel(messages: Ref<ChatMessage[]>) {
  const fileGroupsCache = new WeakMap<FileGroupToolCall[], OutputFileGroup[]>()
  const displayFieldsCache = new WeakMap<DisplayFieldToolCall[], OutputDisplayField[]>()

  const getFileGroupsFromCalls = (calls: FileGroupToolCall[]): OutputFileGroup[] => {
    const cached = fileGroupsCache.get(calls)
    if (cached) return cached

    const groups: OutputFileGroup[] = []
    for (const toolCall of calls) {
      groups.push(...extractFileGroupsFromResult(toolCall.result_data ?? toolCall.result, toolCall.metadata))
    }
    fileGroupsCache.set(calls, groups)
    return groups
  }

  const getDisplayFieldsFromCalls = (calls: DisplayFieldToolCall[]): OutputDisplayField[] => {
    const cached = displayFieldsCache.get(calls)
    if (cached) return cached

    const fields = extractAllDisplayFields(calls)
    displayFieldsCache.set(calls, fields)
    return fields
  }

  const allPanelFiles = computed<FilePanelItem[]>(() => {
    const list: FilePanelItem[] = []
    for (const msg of messages.value) {
      if (msg.role === 'user' && msg.files?.length) {
        for (const file of msg.files) {
          const href = normalizeStorageFileDisplayUrl(file.download_url || file.ref || '')
          list.push({ name: file.source_name || file.name || '未命名文件', href, source: 'upload' })
        }
      }
      if (msg.role === 'assistant' && msg.tool_calls?.length) {
        const groups = getFileGroupsFromCalls(msg.tool_calls)
        for (const group of groups) {
          for (const file of group.files) {
            const href = normalizeStorageFileDisplayUrl(file.download_url || file.ref || '')
            list.push({ name: file.source_name || file.name || '输出文件', href, source: 'output' })
          }
        }
      }
    }
    return list
  })

  const uploadedFiles = computed(() => allPanelFiles.value.filter(file => file.source === 'upload'))
  const outputFiles = computed(() => allPanelFiles.value.filter(file => file.source === 'output'))

  const allPanelDisplayFields = computed<OutputDisplayField[]>(() => {
    const list: OutputDisplayField[] = []
    for (const msg of messages.value) {
      if (msg.role === 'assistant' && msg.tool_calls?.length) {
        list.push(...extractAllDisplayFields(msg.tool_calls))
      }
    }
    return list
  })

  const panelHasContent = computed(() => allPanelFiles.value.length > 0 || allPanelDisplayFields.value.length > 0)
  const panelItemCount = computed(() => allPanelFiles.value.length + allPanelDisplayFields.value.length)

  async function copyDisplayFieldValue(field: OutputDisplayField) {
    try {
      await navigator.clipboard.writeText(field.value)
      ElMessage.success(`已复制「${field.label}」`)
    } catch {
      ElMessage.error('复制失败')
    }
  }

  const keyInfoDropdownRef = ref<any>(null)
  const dfPreviewVisible = ref(false)
  const dfPreviewLabel = ref('')
  const dfPreviewContent = ref('')

  function onKeyInfoDropdownVisibleChange(visible: boolean) {
    if (!visible && dfPreviewVisible.value) {
      setTimeout(() => { keyInfoDropdownRef.value?.handleOpen?.() }, 50)
    }
  }

  function openDfPreview(field: OutputDisplayField) {
    dfPreviewLabel.value = field.label
    dfPreviewContent.value = field.value
    dfPreviewVisible.value = true
  }

  function closeDfPreview() {
    dfPreviewVisible.value = false
  }

  async function copyDfPreviewContent() {
    try {
      await navigator.clipboard.writeText(dfPreviewContent.value)
      ElMessage.success(`已复制「${dfPreviewLabel.value}」`)
    } catch {
      ElMessage.error('复制失败')
    }
  }

  function isImageFile(file: FilePanelItem): boolean {
    const ext = (file.name || '').toLowerCase().match(/\.\w+$/)?.[0] || ''
    return IMAGE_EXTS.has(ext)
  }

  function fileExt(file: FilePanelItem): string {
    return ((file.name || '').match(/\.(\w+)$/)?.[1] || '').toUpperCase()
  }

  function previewFile(file: FilePanelItem) {
    window.open(file.href, '_blank', 'noopener,noreferrer')
  }

  function downloadFile(file: FilePanelItem) {
    const link = document.createElement('a')
    link.href = file.href
    link.download = file.name
    link.target = '_blank'
    link.rel = 'noopener noreferrer'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  return {
    getFileGroupsFromCalls,
    getDisplayFieldsFromCalls,
    keyInfoDropdownRef,
    allPanelDisplayFields,
    uploadedFiles,
    outputFiles,
    panelHasContent,
    panelItemCount,
    copyDisplayFieldValue,
    onKeyInfoDropdownVisibleChange,
    dfPreviewVisible,
    dfPreviewLabel,
    dfPreviewContent,
    openDfPreview,
    closeDfPreview,
    copyDfPreviewContent,
    isImageFile,
    fileExt,
    previewFile,
    downloadFile
  }
}
