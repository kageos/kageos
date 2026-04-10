import { computed, ref, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import { extractFileGroupsFromResult, type OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'
import { extractAllDisplayFields, type OutputDisplayField } from '@/architecture/presentation/composables/useOutputDisplayFields'
import type { ChatMessage } from '@/architecture/presentation/composables/useWorkspaceChatStream'

export interface FilePanelItem {
  name: string
  url: string
  source: 'upload' | 'output'
}

const IMAGE_EXTS = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.svg'])

export function useMiniWorkstationPanel(messages: Ref<ChatMessage[]>) {
  const getFileGroupsFromCalls = (calls: Array<{ result?: string }>): OutputFileGroup[] => {
    const groups: OutputFileGroup[] = []
    for (const toolCall of calls) {
      groups.push(...extractFileGroupsFromResult(toolCall.result))
    }
    return groups
  }

  const getDisplayFieldsFromCalls = (calls: Array<{ arguments?: string; result?: string }>): OutputDisplayField[] => {
    return extractAllDisplayFields(calls)
  }

  const allPanelFiles = computed<FilePanelItem[]>(() => {
    const list: FilePanelItem[] = []
    for (const msg of messages.value) {
      if (msg.role === 'user' && msg.files?.length) {
        for (const file of msg.files) {
          const url = file.url || ''
          list.push({ name: file.source_name || file.name || '未命名文件', url, source: 'upload' })
        }
      }
      if (msg.role === 'assistant' && msg.tool_calls?.length) {
        const groups = getFileGroupsFromCalls(msg.tool_calls)
        for (const group of groups) {
          for (const file of group.files) {
            list.push({ name: file.source_name || file.name || '输出文件', url: file.url, source: 'output' })
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
    window.open(file.url, '_blank', 'noopener,noreferrer')
  }

  function downloadFile(file: FilePanelItem) {
    const link = document.createElement('a')
    link.href = file.url
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
