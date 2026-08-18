<template>
  <div
    ref="editorShellRef"
    class="markdown-document-editor"
    :class="{ 'is-dragging': isDragging, 'is-uploading': uploadingCount > 0 }"
    @dragenter="handleDragEnter"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
  >
    <div v-if="editor" class="markdown-document-editor__toolbar">
      <div class="markdown-document-editor__tools">
        <el-tooltip :content="t('richText.undo')" placement="bottom">
          <el-button text :disabled="!editor.can().undo()" :icon="RefreshLeft" @click="editor.chain().focus().undo().run()" />
        </el-tooltip>
        <el-tooltip :content="t('richText.redo')" placement="bottom">
          <el-button text :disabled="!editor.can().redo()" :icon="RefreshRight" @click="editor.chain().focus().redo().run()" />
        </el-tooltip>
        <span class="markdown-document-editor__divider" />
        <el-dropdown @command="setTextStyle">
          <el-button text>
            {{ currentTextStyleLabel }}
            <el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="paragraph">{{ t('richText.paragraph') }}</el-dropdown-item>
              <el-dropdown-item command="heading-1">{{ t('richText.heading1') }}</el-dropdown-item>
              <el-dropdown-item command="heading-2">{{ t('richText.heading2') }}</el-dropdown-item>
              <el-dropdown-item command="heading-3">{{ t('richText.heading3') }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        <el-tooltip :content="t('richText.bold')" placement="bottom">
          <el-button text :type="editor.isActive('bold') ? 'primary' : 'default'" @click="editor.chain().focus().toggleBold().run()">
            <strong>B</strong>
          </el-button>
        </el-tooltip>
        <el-tooltip :content="t('richText.italic')" placement="bottom">
          <el-button text :type="editor.isActive('italic') ? 'primary' : 'default'" @click="editor.chain().focus().toggleItalic().run()">
            <em>I</em>
          </el-button>
        </el-tooltip>
        <el-tooltip :content="t('richText.strike')" placement="bottom">
          <el-button text :type="editor.isActive('strike') ? 'primary' : 'default'" @click="editor.chain().focus().toggleStrike().run()">
            <s>S</s>
          </el-button>
        </el-tooltip>
        <el-tooltip :content="t('richText.bulletList')" placement="bottom">
          <el-button text :type="editor.isActive('bulletList') ? 'primary' : 'default'" :icon="List" @click="editor.chain().focus().toggleBulletList().run()" />
        </el-tooltip>
        <el-tooltip :content="t('richText.orderedList')" placement="bottom">
          <el-button text :type="editor.isActive('orderedList') ? 'primary' : 'default'" :icon="Sort" @click="editor.chain().focus().toggleOrderedList().run()" />
        </el-tooltip>
        <el-tooltip :content="t('richText.taskList')" placement="bottom">
          <el-button text :type="editor.isActive('taskList') ? 'primary' : 'default'" :icon="Finished" @click="editor.chain().focus().toggleTaskList().run()" />
        </el-tooltip>
        <el-tooltip :content="t('richText.blockquote')" placement="bottom">
          <el-button text :type="editor.isActive('blockquote') ? 'primary' : 'default'" :icon="ChatLineSquare" @click="editor.chain().focus().toggleBlockquote().run()" />
        </el-tooltip>
        <el-tooltip :content="t('richText.codeBlock')" placement="bottom">
          <el-button text :type="editor.isActive('codeBlock') ? 'primary' : 'default'" :icon="Tickets" @click="editor.chain().focus().toggleCodeBlock().run()" />
        </el-tooltip>
        <el-tooltip :content="t('richText.insertLink')" placement="bottom">
          <el-button text :type="editor.isActive('link') ? 'primary' : 'default'" :icon="LinkIcon" @click="toggleLinkEditor" />
        </el-tooltip>
        <el-tooltip :content="t('richText.uploadFile')" placement="bottom">
          <el-button text :icon="Upload" :loading="uploadingCount > 0" @click="openFilePicker" />
        </el-tooltip>
        <input ref="fileInputRef" class="markdown-document-editor__file-input" type="file" multiple @change="handleFileSelection" />
      </div>

      <div class="markdown-document-editor__toolbar-end">
        <span v-if="uploadingCount > 0" class="markdown-document-editor__upload-state">
          {{ t('richText.uploadingCount', { count: uploadingCount }) }}
        </span>
        <el-tooltip :content="isFullscreen ? t('richText.exitFullscreen') : t('richText.fullscreen')" placement="bottom">
          <el-button text :icon="isFullscreen ? Close : FullScreen" @click="toggleFullscreen" />
        </el-tooltip>
      </div>
    </div>

    <div v-if="linkEditorVisible" class="markdown-document-editor__link-editor">
      <el-input
        ref="linkInputRef"
        v-model="linkUrl"
        :placeholder="t('richText.linkPlaceholder')"
        clearable
        @keyup.enter="applyLink"
        @keyup.esc="closeLinkEditor"
      />
      <el-button type="primary" @click="applyLink">{{ t('common.confirm') }}</el-button>
      <el-button @click="closeLinkEditor">{{ t('common.cancel') }}</el-button>
      <el-button v-if="editor?.isActive('link')" type="danger" text @click="removeLink">{{ t('richText.removeLink') }}</el-button>
    </div>

    <div class="markdown-document-editor__content">
      <EditorContent :editor="editor" />
    </div>

    <div v-if="isDragging" class="markdown-document-editor__drop-hint">
      <UploadFilled />
      <span>{{ t('richText.dropToUpload') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import type { EditorProps, EditorView } from 'prosemirror-view'
import type { Slice } from 'prosemirror-model'
import StarterKit from '@tiptap/starter-kit'
import { Link } from '@tiptap/extension-link'
import { Image } from '@tiptap/extension-image'
import { Placeholder } from '@tiptap/extension-placeholder'
import { TaskList } from '@tiptap/extension-task-list'
import { TaskItem } from '@tiptap/extension-task-item'
import { Markdown } from '@tiptap/markdown'
import {
  ArrowDown,
  ChatLineSquare,
  Close,
  Finished,
  FullScreen,
  Link as LinkIcon,
  List,
  RefreshLeft,
  RefreshRight,
  Sort,
  Tickets,
  Upload,
  UploadFilled,
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
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

const { t } = useI18n()
const editorShellRef = ref<HTMLElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const linkInputRef = ref<{ focus: () => void } | null>(null)
const isDragging = ref(false)
const dragDepth = ref(0)
const uploadingCount = ref(0)
const isFullscreen = ref(false)
const linkEditorVisible = ref(false)
const linkUrl = ref('')

const fileUploadRouter = computed(() => {
  if (props.router) return props.router
  try {
    const savedUser = JSON.parse(localStorage.getItem('user') || '{}')
    return `${savedUser.username || 'default'}/docs/files`
  } catch {
    return 'default/docs/files'
  }
})

const editor = useEditor({
  extensions: [
    StarterKit.configure({ link: false }),
    Link.configure({ openOnClick: false, autolink: true, defaultProtocol: 'https' }),
    Image.configure({ allowBase64: false }),
    Placeholder.configure({ placeholder: props.placeholder }),
    TaskList,
    TaskItem.configure({ nested: true }),
    Markdown,
  ],
  content: props.modelValue,
  contentType: 'markdown',
  onUpdate: ({ editor: currentEditor }) => {
    emit('update:modelValue', currentEditor.getMarkdown())
  },
  editorProps: {
    handlePaste: ((_view: EditorView, event: ClipboardEvent, _slice: Slice) => {
      const files = getClipboardFiles(event.clipboardData)
      if (files.length === 0) return false
      event.preventDefault()
      void uploadAndInsertFiles(files)
      return true
    }) satisfies NonNullable<EditorProps['handlePaste']>,
    handleDrop: ((_view: EditorView, event: DragEvent, _slice: Slice, moved: boolean) => {
      resetDragState()
      if (moved || !event.dataTransfer?.files.length) return false
      event.preventDefault()
      void uploadAndInsertFiles(Array.from(event.dataTransfer.files))
      return true
    }) satisfies NonNullable<EditorProps['handleDrop']>,
  },
})

const currentTextStyleLabel = computed(() => {
  if (editor.value?.isActive('heading', { level: 1 })) return t('richText.heading1')
  if (editor.value?.isActive('heading', { level: 2 })) return t('richText.heading2')
  if (editor.value?.isActive('heading', { level: 3 })) return t('richText.heading3')
  return t('richText.paragraph')
})

function setTextStyle(command: string) {
  if (!editor.value) return
  if (command === 'paragraph') {
    editor.value.chain().focus().setParagraph().run()
    return
  }
  const level = Number(command.replace('heading-', '')) as 1 | 2 | 3
  editor.value.chain().focus().toggleHeading({ level }).run()
}

function toggleLinkEditor() {
  if (!editor.value) return
  linkUrl.value = editor.value.getAttributes('link').href || ''
  linkEditorVisible.value = true
  void nextTick(() => linkInputRef.value?.focus())
}

function closeLinkEditor() {
  linkEditorVisible.value = false
  linkUrl.value = ''
  editor.value?.commands.focus()
}

function applyLink() {
  const href = linkUrl.value.trim()
  if (!editor.value || !href) return
  const { empty } = editor.value.state.selection
  if (empty) {
    editor.value.commands.insertContent({
      type: 'text',
      text: href,
      marks: [{ type: 'link', attrs: { href } }],
    })
  } else {
    editor.value.chain().focus().extendMarkRange('link').setLink({ href }).run()
  }
  closeLinkEditor()
}

function removeLink() {
  editor.value?.chain().focus().extendMarkRange('link').unsetLink().run()
  closeLinkEditor()
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

function getClipboardFiles(data: DataTransfer | null): File[] {
  if (!data) return []
  const itemFiles = Array.from(data.items || [])
    .filter(item => item.kind === 'file')
    .map(item => item.getAsFile())
    .filter((file): file is File => file !== null)
  return itemFiles.length > 0 ? itemFiles : Array.from(data.files || [])
}

async function uploadAndInsertFiles(files: File[]) {
  const markdownItems: string[] = []
  for (const originalFile of files) {
    const file = ensureClipboardFileName(originalFile)
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
      const downloadUrl = completeResult?.download_url
      if (!downloadUrl || !editor.value) throw new Error(t('richText.downloadUrlFailed'))
      const label = escapeMarkdownLabel(file.name)
      const markdown = isImageFile(file)
        ? `![${label}](${downloadUrl})`
        : `[${label}](${downloadUrl})`
      markdownItems.push(markdown)
      ElMessage.success(t('richText.uploadSuccess', { name: file.name }))
    } catch (error: any) {
      ElMessage.error(t('richText.uploadFailed', {
        name: file.name,
        message: error?.message || t('richText.unknownError'),
      }))
    } finally {
      uploadingCount.value -= 1
    }
  }
  if (markdownItems.length > 0 && editor.value) {
    editor.value.commands.insertContent(markdownItems.join('\n\n'), { contentType: 'markdown' })
    editor.value.commands.focus()
  }
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

watch(() => props.modelValue, (value) => {
  if (!editor.value || editor.value.getMarkdown() === value) return
  editor.value.commands.setContent(value || '', { contentType: 'markdown', emitUpdate: false })
})

onMounted(() => document.addEventListener('fullscreenchange', syncFullscreenState))
onBeforeUnmount(() => {
  document.removeEventListener('fullscreenchange', syncFullscreenState)
  if (document.fullscreenElement === editorShellRef.value) {
    void document.exitFullscreen()
  }
})
</script>

<style scoped>
.markdown-document-editor {
  position: relative;
  display: flex;
  min-height: 520px;
  flex-direction: column;
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

.markdown-document-editor__toolbar {
  display: flex;
  flex: 0 0 auto;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-light);
}

.markdown-document-editor__tools,
.markdown-document-editor__toolbar-end {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 2px;
}

.markdown-document-editor__toolbar-end {
  flex: 0 0 auto;
}

.markdown-document-editor__divider {
  width: 1px;
  height: 22px;
  margin: 0 4px;
  background: var(--el-border-color);
}

.markdown-document-editor__file-input {
  display: none;
}

.markdown-document-editor__upload-state {
  padding: 0 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.markdown-document-editor__link-editor {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) auto auto auto;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}

.markdown-document-editor__content {
  min-height: 0;
  flex: 1 1 auto;
  overflow: auto;
}

.markdown-document-editor__content :deep(.tiptap) {
  min-height: 480px;
  padding: 34px clamp(24px, 6vw, 80px) 80px;
  color: var(--el-text-color-primary);
  font-size: 16px;
  line-height: 1.75;
  outline: none;
}

.markdown-document-editor:fullscreen .markdown-document-editor__content :deep(.tiptap) {
  min-height: calc(100vh - 58px);
  max-width: 1080px;
  margin: 0 auto;
}

.markdown-document-editor__content :deep(.tiptap p.is-editor-empty:first-child::before) {
  height: 0;
  float: left;
  color: var(--el-text-color-placeholder);
  content: attr(data-placeholder);
  pointer-events: none;
}

.markdown-document-editor__content :deep(.tiptap img) {
  display: block;
  max-width: 100%;
  height: auto;
  margin: 18px auto;
  border-radius: 8px;
}

.markdown-document-editor__content :deep(.tiptap pre) {
  overflow-x: auto;
  padding: 16px;
  border-radius: 8px;
  background: var(--el-fill-color-darker);
}

.markdown-document-editor__content :deep(.tiptap blockquote) {
  margin-left: 0;
  padding-left: 16px;
  border-left: 3px solid var(--el-color-primary-light-5);
  color: var(--el-text-color-regular);
}

.markdown-document-editor__content :deep(.tiptap ul[data-type='taskList']) {
  padding-left: 0;
  list-style: none;
}

.markdown-document-editor__content :deep(.tiptap ul[data-type='taskList'] li) {
  display: flex;
  gap: 8px;
}

.markdown-document-editor__drop-hint {
  position: absolute;
  inset: 12px;
  display: grid;
  place-content: center;
  gap: 10px;
  border: 2px dashed var(--el-color-primary);
  border-radius: 10px;
  background: color-mix(in srgb, var(--el-color-primary-light-9) 92%, transparent);
  color: var(--el-color-primary);
  font-weight: 600;
  pointer-events: none;
}

.markdown-document-editor__drop-hint svg {
  width: 32px;
  height: 32px;
  margin: 0 auto;
}

@media (max-width: 720px) {
  .markdown-document-editor__toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .markdown-document-editor__toolbar-end {
    justify-content: flex-end;
  }

  .markdown-document-editor__link-editor {
    grid-template-columns: 1fr auto;
  }

  .markdown-document-editor__content :deep(.tiptap) {
    padding: 24px 18px 60px;
  }
}
</style>
