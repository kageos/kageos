import { computed, ref, watch, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import { resolveFileRefs, type ResolvedFile } from '@/architecture/presentation/context/api/storage'
import type { ToolResultMetadata } from '@/architecture/presentation/context/api/workspace'
import { extractFileGroupsFromResult, type OutputFileGroup, type OutputFileItem } from '@/architecture/presentation/composables/useOutputFileGroups'
import { extractAllDisplayFields, type OutputDisplayField } from '@/architecture/presentation/composables/useOutputDisplayFields'
import type { ChatMessage } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import { normalizeStorageFileDisplayUrl } from '@/architecture/presentation/utils/storageFileUrl'

export interface FilePanelItem {
  name: string
  href: string
  source: 'upload' | 'output'
  ref?: string
}

const IMAGE_EXTS = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.svg'])
const OUTPUT_FILE_RESOLVE_BATCH_SIZE = 100
const OUTPUT_FILE_RESOLVE_MAX_ATTEMPTS = 2
const OUTPUT_FILE_RESOLVE_RETRY_DELAY_MS = 200

type FileGroupToolCall = { result?: string; result_data?: unknown; metadata?: ToolResultMetadata }
type DisplayFieldToolCall = { arguments?: string; result?: string; result_data?: unknown }
type ResolveFileRefsFn = (refs: string[], audience: 'browser' | 'server' | 'all') => Promise<ResolvedFile[]>

export { normalizeStorageFileDisplayUrl as normalizeMiniFileDisplayUrl }

export function buildOutputPanelFile(file: OutputFileItem, resolved?: ResolvedFile): FilePanelItem {
  return {
    name: resolved?.source_name || file.source_name || resolved?.name || file.name || '输出文件',
    href: normalizeStorageFileDisplayUrl(resolved?.download_url || file.download_url || file.ref || ''),
    source: 'output',
    ref: file.ref,
  }
}

export async function resolveOutputFileBatch(
  refs: string[],
  resolver: ResolveFileRefsFn = resolveFileRefs,
  retryDelayMs = OUTPUT_FILE_RESOLVE_RETRY_DELAY_MS
): Promise<ResolvedFile[]> {
  let lastError: unknown
  for (let attempt = 1; attempt <= OUTPUT_FILE_RESOLVE_MAX_ATTEMPTS; attempt += 1) {
    try {
      return await resolver(refs, 'browser')
    } catch (error) {
      lastError = error
      if (attempt < OUTPUT_FILE_RESOLVE_MAX_ATTEMPTS && retryDelayMs > 0) {
        await new Promise(resolve => setTimeout(resolve, retryDelayMs))
      }
    }
  }
  throw lastError
}

function chunkOutputFileRefs(refs: string[]): string[][] {
  const batches: string[][] = []
  for (let index = 0; index < refs.length; index += OUTPUT_FILE_RESOLVE_BATCH_SIZE) {
    batches.push(refs.slice(index, index + OUTPUT_FILE_RESOLVE_BATCH_SIZE))
  }
  return batches
}

export function useMiniWorkstationPanel(messages: Ref<ChatMessage[]>) {
  const fileGroupsCache = new WeakMap<FileGroupToolCall[], OutputFileGroup[]>()
  const displayFieldsCache = new WeakMap<DisplayFieldToolCall[], OutputDisplayField[]>()
  const resolvedOutputFiles = ref<Record<string, ResolvedFile>>({})
  const resolvingOutputRefs = new Set<string>()

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

  const outputFileRefs = computed(() => {
    const refs = new Set<string>()
    for (const msg of messages.value) {
      if (msg.role !== 'assistant' || !msg.tool_calls?.length) continue
      for (const group of getFileGroupsFromCalls(msg.tool_calls)) {
        for (const file of group.files) {
          if (file.ref) refs.add(file.ref)
        }
      }
    }
    return Array.from(refs)
  })

  const outputFileRefsSignature = computed(() => outputFileRefs.value.join('\n'))

  watch(outputFileRefsSignature, async () => {
    const refs = outputFileRefs.value.filter(ref => !resolvedOutputFiles.value[ref] && !resolvingOutputRefs.has(ref))
    if (refs.length === 0) return

    refs.forEach(ref => resolvingOutputRefs.add(ref))
    try {
      const results = await Promise.allSettled(
        chunkOutputFileRefs(refs).map(batch => resolveOutputFileBatch(batch))
      )
      const resolved = results.flatMap(result => result.status === 'fulfilled' ? result.value : [])
      if (resolved.length > 0) {
        const next = { ...resolvedOutputFiles.value }
        for (const file of resolved) {
          next[file.ref] = file
        }
        resolvedOutputFiles.value = next
      }
    } finally {
      // 失败批次保留对象 key basename；同一批次内部已做一次短延迟重试。
      refs.forEach(ref => resolvingOutputRefs.delete(ref))
    }
  }, { immediate: true })

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
            list.push(buildOutputPanelFile(file, file.ref ? resolvedOutputFiles.value[file.ref] : undefined))
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
