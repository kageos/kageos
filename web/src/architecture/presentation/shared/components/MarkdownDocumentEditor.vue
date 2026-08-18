<template>
  <div
    ref="editorShellRef"
    class="markdown-document-editor"
    :class="{ 'is-dragging': isDragging }"
    @paste.capture="handlePaste"
    @dragenter="handleDragEnter"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
    @drop.capture="handleDrop"
  >
    <MdEditor
      ref="editorRef"
      v-model="content"
      class="markdown-document-editor__instance"
      :theme="editorTheme"
      :language="editorLanguage"
      preview-theme="github"
      code-theme="github"
      :placeholder="placeholder"
      :toolbars="toolbars"
      :footers="footers"
      :no-prettier="true"
      :auto-detect-code="true"
      :show-code-row-number="true"
      input-box-width="52%"
      @onUploadImg="handleUploadImages"
      @onError="handleEditorError"
    >
      <template #defToolbars>
        <NormalToolbar :title="t('richText.uploadFile')" @onClick="openFilePicker">
          <el-icon class="markdown-document-editor__toolbar-icon"><Paperclip /></el-icon>
        </NormalToolbar>
        <NormalToolbar
          :title="isFullscreen ? t('richText.exitFullscreen') : t('richText.fullscreen')"
          @onClick="toggleFullscreen"
        >
          <el-icon class="markdown-document-editor__toolbar-icon">
            <Close v-if="isFullscreen" />
            <FullScreen v-else />
          </el-icon>
        </NormalToolbar>
      </template>
    </MdEditor>

    <input
      ref="fileInputRef"
      class="markdown-document-editor__file-input"
      type="file"
      multiple
      @change="handleFileSelection"
    />

    <div v-if="uploadingCount > 0" class="markdown-document-editor__upload-state">
      {{ t('richText.uploadingCount', { count: uploadingCount }) }}
    </div>

    <div v-if="isDragging" class="markdown-document-editor__drop-hint">
      <el-icon><UploadFilled /></el-icon>
      <span>{{ t('richText.dropToUpload') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { MdEditor, NormalToolbar } from 'md-editor-v3'
import type { ExposeParam, Footers, InnerError, ToolbarNames, UploadImgCallBack } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'
import { Close, FullScreen, Paperclip, UploadFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useThemeStore } from '@/architecture/presentation/context/appStoresContext'
import { uploadFile, notifyUploadComplete } from '@/architecture/presentation/context/uploadContext'

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  router?: string
}>(), {
  placeholder: '开始写文档...',
  router: '',
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { locale, t } = useI18n()
const themeStore = useThemeStore()
const editorShellRef = ref<HTMLElement | null>(null)
const editorRef = ref<ExposeParam | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const isDragging = ref(false)
const dragDepth = ref(0)
const uploadingCount = ref(0)
const isFullscreen = ref(false)

const content = computed({
  get: () => props.modelValue,
  set: (value: string) => emit('update:modelValue', value),
})

const editorTheme = computed(() => themeStore.currentTheme.mode === 'dark' ? 'dark' : 'light')
const editorLanguage = computed(() => locale.value.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en-US')
const toolbars: ToolbarNames[] = [
  'bold',
  'italic',
  'strikeThrough',
  '-',
  'title',
  'quote',
  'unorderedList',
  'orderedList',
  'task',
  '-',
  'codeRow',
  'code',
  'link',
  'image',
  'table',
  0,
  '-',
  'revoke',
  'next',
  '=',
  'preview',
  'catalog',
  1,
]
const footers: Footers[] = ['markdownTotal', '=', 'scrollSwitch']

const fileUploadRouter = computed(() => {
  if (props.router) return props.router
  try {
    const savedUser = JSON.parse(localStorage.getItem('user') || '{}')
    return `${savedUser.username || 'default'}/docs/files`
  } catch {
    return 'default/docs/files'
  }
})

async function handleUploadImages(files: File[], callback: UploadImgCallBack) {
  const uploaded: Array<{ url: string; alt: string; title: string }> = []
  for (const originalFile of files) {
    const file = ensureClipboardFileName(originalFile)
    try {
      const url = await uploadOne(file)
      uploaded.push({ url, alt: file.name, title: file.name })
      ElMessage.success(t('richText.uploadSuccess', { name: file.name }))
    } catch (error: any) {
      showUploadError(file, error)
    }
  }
  callback(uploaded)
}

function handlePaste(event: ClipboardEvent) {
  const files = getClipboardFiles(event.clipboardData)
  if (files.length === 0 || files.every(isImageFile)) return
  event.preventDefault()
  event.stopPropagation()
  void uploadAndInsertFiles(files)
}

function handleDrop(event: DragEvent) {
  resetDragState()
  const files = Array.from(event.dataTransfer?.files || [])
  if (files.length === 0 || files.every(isImageFile)) return
  event.preventDefault()
  event.stopPropagation()
  void uploadAndInsertFiles(files)
}

function openFilePicker() {
  fileInputRef.value?.click()
}

function handleFileSelection(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (files.length > 0) void uploadAndInsertFiles(files)
}

async function uploadAndInsertFiles(files: File[]) {
  const markdownItems: string[] = []
  for (const originalFile of files) {
    const file = ensureClipboardFileName(originalFile)
    try {
      const url = await uploadOne(file)
      const label = escapeMarkdownLabel(file.name)
      markdownItems.push(isImageFile(file) ? `![${label}](${url})` : `[${label}](${url})`)
      ElMessage.success(t('richText.uploadSuccess', { name: file.name }))
    } catch (error: any) {
      showUploadError(file, error)
    }
  }
  insertMarkdown(markdownItems.join('\n\n'))
}

async function uploadOne(file: File): Promise<string> {
  uploadingCount.value += 1
  try {
    const uploadResult = await uploadFile(fileUploadRouter.value, file, () => {})
    if (!uploadResult.fileInfo) throw new Error(t('richText.downloadUrlFailed'))
    const completeResult = await notifyUploadComplete({
      key: uploadResult.fileInfo.key,
      success: true,
      router: uploadResult.fileInfo.router,
      file_name: uploadResult.fileInfo.file_name,
      file_size: uploadResult.fileInfo.file_size,
      content_type: uploadResult.fileInfo.content_type,
      hash: uploadResult.fileInfo.hash,
    })
    const url = completeResult?.download_url
    if (!url) throw new Error(t('richText.downloadUrlFailed'))
    return url
  } finally {
    uploadingCount.value -= 1
  }
}

function insertMarkdown(markdown: string) {
  if (!markdown || !editorRef.value) return
  editorRef.value.insert(() => ({
    targetValue: `\n${markdown}\n`,
    select: false,
  }))
  editorRef.value.focus()
}

function getClipboardFiles(data: DataTransfer | null): File[] {
  if (!data) return []
  const itemFiles = Array.from(data.items || [])
    .filter(item => item.kind === 'file')
    .map(item => item.getAsFile())
    .filter((file): file is File => file !== null)
  return itemFiles.length > 0 ? itemFiles : Array.from(data.files || [])
}

function ensureClipboardFileName(file: File): File {
  if (file.name) return file
  const extension = file.type.split('/')[1]?.replace('jpeg', 'jpg') || 'bin'
  return new File([file], `clipboard-${Date.now()}.${extension}`, { type: file.type })
}

function isImageFile(file: File): boolean {
  if (file.type.startsWith('image/')) return true
  return /\.(avif|bmp|gif|ico|jpe?g|png|svg|webp)$/i.test(file.name)
}

function escapeMarkdownLabel(value: string): string {
  return value.replace(/([\\\[\]])/g, '\\$1')
}

function showUploadError(file: File, error: any) {
  ElMessage.error(t('richText.uploadFailed', {
    name: file.name,
    message: error?.message || t('richText.unknownError'),
  }))
}

function handleDragEnter(event: DragEvent) {
  if (!event.dataTransfer?.types.includes('Files')) return
  dragDepth.value += 1
  isDragging.value = true
}

function handleDragOver(event: DragEvent) {
  if (!event.dataTransfer?.types.includes('Files')) return
  event.preventDefault()
  event.dataTransfer.dropEffect = 'copy'
}

function handleDragLeave() {
  dragDepth.value = Math.max(0, dragDepth.value - 1)
  if (dragDepth.value === 0) isDragging.value = false
}

function resetDragState() {
  dragDepth.value = 0
  isDragging.value = false
}

async function toggleFullscreen() {
  const shell = editorShellRef.value
  if (!shell) return
  try {
    if (document.fullscreenElement === shell) {
      await document.exitFullscreen()
    } else {
      await shell.requestFullscreen()
    }
  } catch (error: any) {
    ElMessage.error(error?.message || t('richText.fullscreenFailed'))
  }
}

function syncFullscreenState() {
  isFullscreen.value = document.fullscreenElement === editorShellRef.value
}

function handleEditorError(error: InnerError) {
  ElMessage.error(error.message || t('richText.editorError'))
}

onMounted(() => document.addEventListener('fullscreenchange', syncFullscreenState))
onBeforeUnmount(() => {
  document.removeEventListener('fullscreenchange', syncFullscreenState)
  if (document.fullscreenElement === editorShellRef.value) void document.exitFullscreen()
})
</script>

<style scoped>
.markdown-document-editor {
  position: relative;
  min-height: 600px;
  overflow: hidden;
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
  background: var(--el-bg-color);
}

.markdown-document-editor:fullscreen {
  width: 100vw;
  height: 100vh;
  min-height: 0;
  border: 0;
  border-radius: 0;
}

.markdown-document-editor__instance {
  height: clamp(600px, calc(100vh - 310px), 840px);
  border: 0;
}

.markdown-document-editor:fullscreen .markdown-document-editor__instance {
  height: 100vh;
}

.markdown-document-editor__file-input {
  display: none;
}

.markdown-document-editor__toolbar-icon {
  font-size: 18px;
}

.markdown-document-editor__upload-state {
  position: absolute;
  right: 16px;
  bottom: 10px;
  padding: 5px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  pointer-events: none;
}

.markdown-document-editor__drop-hint {
  position: absolute;
  inset: 12px;
  display: grid;
  place-content: center;
  gap: 10px;
  border: 2px dashed var(--el-color-primary);
  border-radius: 10px;
  background: color-mix(in srgb, var(--el-color-primary-light-9) 94%, transparent);
  color: var(--el-color-primary);
  font-weight: 600;
  pointer-events: none;
}

.markdown-document-editor__drop-hint :deep(svg) {
  width: 32px;
  height: 32px;
  margin: 0 auto;
}

.markdown-document-editor :deep(.md-editor-toolbar-wrapper) {
  padding-inline: 10px;
}

.markdown-document-editor :deep(.md-editor-content) {
  min-height: 0;
}

.markdown-document-editor :deep(.md-editor-input-wrapper) {
  background: var(--el-bg-color);
}

.markdown-document-editor :deep(.md-editor-preview-wrapper) {
  background: var(--el-fill-color-blank);
}

@media (max-width: 760px) {
  .markdown-document-editor,
  .markdown-document-editor__instance {
    min-height: 560px;
  }
}
</style>
