<template>
  <div 
    class="rich-text-editor"
    @dragover.prevent="handleDragOver"
    @dragleave.prevent="handleDragLeave"
    @drop.prevent="handleEditorDrop"
  >
    <div v-if="editor" class="editor-toolbar">
      <!-- 文本格式组 -->
      <div class="toolbar-group">
        <el-button-group>
          <el-tooltip content="粗体" placement="bottom">
            <el-button
              :type="editor.isActive('bold') ? 'primary' : 'default'"
              @click="editor.chain().focus().toggleBold().run()"
            >
              <strong style="font-size: 14px;">B</strong>
            </el-button>
          </el-tooltip>
          <el-tooltip content="斜体" placement="bottom">
            <el-button
              :type="editor.isActive('italic') ? 'primary' : 'default'"
              @click="editor.chain().focus().toggleItalic().run()"
            >
              <em style="font-size: 14px;">I</em>
            </el-button>
          </el-tooltip>
          <el-tooltip content="下划线" placement="bottom">
            <el-button
              :type="editor.isActive('underline') ? 'primary' : 'default'"
              @click="editor.chain().focus().toggleUnderline().run()"
            >
              <u style="font-size: 14px;">U</u>
            </el-button>
          </el-tooltip>
        </el-button-group>
      </div>

      <div class="toolbar-divider"></div>

      <!-- 标题组 -->
      <div class="toolbar-group">
        <el-dropdown @command="handleHeading">
          <el-button>
            <el-icon><Document /></el-icon>
            <el-icon class="el-icon--right"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="paragraph">正文</el-dropdown-item>
              <el-dropdown-item command="heading-1">标题 1</el-dropdown-item>
              <el-dropdown-item command="heading-2">标题 2</el-dropdown-item>
              <el-dropdown-item command="heading-3">标题 3</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>

      <div class="toolbar-divider"></div>

      <!-- 列表组 -->
      <div class="toolbar-group">
        <el-button-group>
          <el-tooltip content="无序列表" placement="bottom">
            <el-button
              :type="editor.isActive('bulletList') ? 'primary' : 'default'"
              :icon="List"
              @click="editor.chain().focus().toggleBulletList().run()"
            />
          </el-tooltip>
          <el-tooltip content="有序列表" placement="bottom">
            <el-button
              :type="editor.isActive('orderedList') ? 'primary' : 'default'"
              :icon="Sort"
              @click="editor.chain().focus().toggleOrderedList().run()"
            />
          </el-tooltip>
        </el-button-group>
      </div>

      <div class="toolbar-divider"></div>

      <!-- 链接、图片和文件 -->
      <div class="toolbar-group">
        <el-tooltip content="插入链接" placement="bottom">
          <el-button
            :icon="LinkIcon"
            @click="handleInsertLink"
          />
        </el-tooltip>
        <el-tooltip content="插入图片" placement="bottom">
          <el-button
            :icon="Picture"
            @click="handleInsertImage"
          />
        </el-tooltip>
        <el-tooltip content="上传文件" placement="bottom">
          <el-button
            :icon="Upload"
            @click="handleUploadFile"
          />
        </el-tooltip>
      </div>

      <div class="toolbar-divider"></div>

      <!-- 撤销/重做 -->
      <div class="toolbar-group">
        <el-button-group>
          <el-tooltip content="撤销" placement="bottom">
            <el-button
              :disabled="!editor.can().undo()"
              :icon="RefreshLeft"
              @click="editor.chain().focus().undo().run()"
            />
          </el-tooltip>
          <el-tooltip content="重做" placement="bottom">
            <el-button
              :disabled="!editor.can().redo()"
              :icon="RefreshRight"
              @click="editor.chain().focus().redo().run()"
            />
          </el-tooltip>
        </el-button-group>
      </div>
    </div>

    <div 
      class="editor-content"
      :class="{ 'is-dragging': isDragging }"
    >
      <editor-content :editor="editor" />
    </div>

    <!-- 链接输入对话框 -->
    <el-dialog
      v-model="linkDialogVisible"
      title="插入链接"
      width="400px"
    >
      <el-input
        v-model="linkUrl"
        placeholder="请输入链接地址"
        clearable
      />
      <template #footer>
        <el-button @click="linkDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmInsertLink">确定</el-button>
      </template>
    </el-dialog>

    <!-- 图片输入对话框 -->
    <el-dialog
      v-model="imageDialogVisible"
      title="插入图片"
      width="400px"
    >
      <el-input
        v-model="imageUrl"
        placeholder="请输入图片地址"
        clearable
      />
      <template #footer>
        <el-button @click="imageDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmInsertImage">确定</el-button>
      </template>
    </el-dialog>

    <!-- 文件上传对话框 -->
    <el-dialog
      v-model="fileUploadDialogVisible"
      title="上传文件"
      width="500px"
      :close-on-click-modal="false"
    >
      <CommonUpload
        ref="fileUploadRef"
        v-model="uploadedFileUrl"
        :router="fileUploadRouter"
        accept="*"
        max-size="100MB"
        @success="handleFileUploadSuccess"
        @error="handleFileUploadError"
        @change="handleFileUploadChange"
      />
      <div v-if="uploadedFileInfo" class="file-info">
        <p><strong>文件名：</strong>{{ uploadedFileInfo.fileName }}</p>
        <p><strong>文件类型：</strong>{{ uploadedFileInfo.fileType }}</p>
        <p v-if="uploadedFileInfo.fileSize > 0"><strong>文件大小：</strong>{{ formatFileSize(uploadedFileInfo.fileSize) }}</p>
      </div>
      <template #footer>
        <el-button @click="fileUploadDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :disabled="!uploadedFileUrl"
          @click="handleConfirmFileInsert"
        >
          插入文件
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onBeforeUnmount, watch } from 'vue'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import type { EditorProps, EditorView } from 'prosemirror-view'
import type { Slice } from 'prosemirror-model'
import StarterKit from '@tiptap/starter-kit'
import { Link } from '@tiptap/extension-link'
import { Image } from '@tiptap/extension-image'
import { Video } from '@/architecture/presentation/shared/tiptap/VideoExtension'
import { Underline } from '@tiptap/extension-underline'
import { Placeholder } from '@tiptap/extension-placeholder'
import {
  Document,
  List,
  Sort,
  Link as LinkIcon,
  Picture,
  RefreshLeft,
  RefreshRight,
  ArrowDown,
  Upload
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import CommonUpload from './CommonUpload.vue'
import { uploadFile, notifyUploadComplete } from '@/architecture/presentation/context/uploadContext'

const props = defineProps<{
  modelValue: string
  placeholder?: string
  router?: string  // 文件上传路由（可选）
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

// 文件上传相关
const fileUploadDialogVisible = ref(false)
const uploadedFileUrl = ref<string>('')
const uploadedFileInfo = ref<{
  fileName: string
  fileType: string
  fileSize: number
} | null>(null)
const fileUploadRef = ref<InstanceType<typeof CommonUpload> | null>(null)

// 拖拽状态
const isDragging = ref(false)

// 文件上传路由（从 localStorage 获取用户名或使用传入的 router）
const fileUploadRouter = computed(() => {
  if (props.router) {
    return props.router
  }
  const savedUserStr = localStorage.getItem('user')
  if (savedUserStr) {
    try {
      const savedUser = JSON.parse(savedUserStr)
      return `${savedUser.username || 'default'}/richtext/files`
    } catch {
      return 'default/richtext/files'
    }
  }
  return 'default/richtext/files'
})

// 判断文件是否为图片
function isImageFile(fileName: string): boolean {
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico'].includes(ext)
}

// 判断文件是否为视频
function isVideoFile(fileName: string): boolean {
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  return ['mp4', 'webm', 'ogg', 'mov', 'avi', 'wmv', 'flv'].includes(ext)
}

// 格式化文件大小
function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

// 处理文件上传变化
function handleFileUploadChange(url: string | null): void {
  uploadedFileUrl.value = url || ''
  if (!url) {
    uploadedFileInfo.value = null
    return
  }
  
  // 从 URL 中提取文件名和类型信息
  try {
    const urlObj = new URL(url)
    const pathParts = urlObj.pathname.split('/')
    const fileName = decodeURIComponent(pathParts[pathParts.length - 1] || '未知文件')
    const fileType = fileName.split('.').pop()?.toUpperCase() || '未知'
    
    uploadedFileInfo.value = {
      fileName,
      fileType,
      fileSize: 0 // 文件大小无法从 URL 中获取，显示为 0
    }
  } catch (error) {
    uploadedFileInfo.value = {
      fileName: '未知文件',
      fileType: '未知',
      fileSize: 0
    }
  }
}

// 处理文件上传成功
function handleFileUploadSuccess(url: string): void {
  uploadedFileUrl.value = url
  handleFileUploadChange(url)
}

// 处理文件上传错误
function handleFileUploadError(error: Error): void {
  ElMessage.error(`文件上传失败: ${error.message || '未知错误'}`)
  uploadedFileUrl.value = ''
  uploadedFileInfo.value = null
}

// 上传文件
const handleUploadFile = () => {
  fileUploadDialogVisible.value = true
  uploadedFileUrl.value = ''
  uploadedFileInfo.value = null
}

// 确认插入文件
const handleConfirmFileInsert = async () => {
  if (!uploadedFileUrl.value || !editor.value) {
    return
  }

  const fileName = uploadedFileInfo.value?.fileName || '文件'
  const isImage = uploadedFileInfo.value ? isImageFile(uploadedFileInfo.value.fileName) : false
  const isVideo = uploadedFileInfo.value ? isVideoFile(uploadedFileInfo.value.fileName) : false

  if (isImage) {
    // 图片：插入为图片
    editor.value.chain().focus().setImage({ src: uploadedFileUrl.value, alt: fileName }).run()
  } else if (isVideo) {
    // 视频：插入为视频
    editor.value.chain().focus().setVideo({ 
      src: uploadedFileUrl.value,
      alt: fileName,
      controls: true
    }).run()
  } else {
    // 其他文件：插入为链接
    editor.value.chain().focus().setLink({ href: uploadedFileUrl.value }).insertContent(fileName).run()
  }

  fileUploadDialogVisible.value = false
  uploadedFileUrl.value = ''
  uploadedFileInfo.value = null
}

const editor = useEditor({
  extensions: [
    StarterKit.configure({
      link: false,
      underline: false, // 排除 StarterKit 中的 underline，使用自定义的 Underline
    }),
    Underline,
    Link.configure({
      openOnClick: false,
      HTMLAttributes: {
        class: 'rich-text-link',
        target: '_blank',
        rel: 'noopener noreferrer'
      }
    }),
    Image.configure({
      HTMLAttributes: {
        class: 'rich-text-image'
      },
      inline: true,
      allowBase64: false // 🔥 禁用 base64，强制使用 URL
    }),
    Video.configure({
      HTMLAttributes: {
        class: 'rich-text-video'
      },
      inline: false,
      allowBase64: false
    }),
    Placeholder.configure({
      placeholder: props.placeholder || '请输入内容...'
    })
  ],
  content: props.modelValue || '',
  onUpdate: ({ editor }) => {
    emit('update:modelValue', editor.getHTML())
  },
  editorProps: {
    // 优化粘贴处理：支持从 Word、网页、Markdown 等粘贴，自动清理格式
    // 特别处理：检测粘贴的文件（任意类型），自动上传而不是使用 base64
    handlePaste: ((view: EditorView, event: ClipboardEvent, slice: Slice) => {
      const clipboardData = event.clipboardData
      
      // 如果没有 clipboardData，让 TipTap 使用默认处理
      if (!clipboardData) {
        return false
      }
      
      try {
        // 检测是否有文件（任意类型）
          const items = Array.from(clipboardData.items || []) as DataTransferItem[]
        const fileItems = items.filter(item => item.kind === 'file')
        
        // 如果有文件，处理文件上传
        if (fileItems.length > 0) {
          // 阻止默认粘贴行为
          event.preventDefault()
          
          const files = fileItems
            .map(item => item.getAsFile())
            .filter((file): file is File => file !== null)
          void uploadEditorFiles(files, 'paste')
          
          return true // 已处理，阻止默认行为
        }
        
        // 如果没有文件，处理文本粘贴
        // 使用 slice 参数直接插入内容（TipTap 已经处理好了格式转换）
        if (slice && slice.content && editor.value) {
          // 使用 slice 插入内容，TipTap 会自动处理 Markdown 转换、HTML 清理等
          const { state, dispatch } = view
          const transaction = view.state.tr.replaceSelection(slice)
          dispatch(transaction)
          return true // 已处理
        }
        
        // 如果 slice 为空，让 TipTap 使用默认处理
        return false
      } catch (error: any) {
        // 如果处理过程中出错，记录错误但让 TipTap 使用默认处理
        console.error('RichTextEditor', '粘贴处理失败', error)
        return false
      }
    }) satisfies NonNullable<EditorProps['handlePaste']>,
    // 支持拖拽粘贴文件（任意类型），自动上传
    handleDrop: ((view: EditorView, event: DragEvent, slice: Slice, moved: boolean) => {
      // 重置拖拽状态
      isDragging.value = false
      
      if (moved) {
        // 如果是编辑器内部的拖拽移动，使用默认处理
        return false
      }
      
      const dataTransfer = event.dataTransfer
      if (!dataTransfer || !dataTransfer.files || dataTransfer.files.length === 0) {
        return false
      }
      
      // 阻止默认拖拽行为
      event.preventDefault()
      
      const files = Array.from(dataTransfer.files || []) as File[]
      void uploadEditorFiles(files, 'drop')
      
      return true // 已处理，阻止默认行为
    }) satisfies NonNullable<EditorProps['handleDrop']>
  }
})

async function uploadEditorFiles(files: File[], source: 'paste' | 'drop'): Promise<void> {
  for (const file of files) {
    try {
      ElMessage.info(`正在上传 ${file.name}...`)
      const uploadResult = await uploadFile(
        fileUploadRouter.value,
        file,
        () => {}
      )

      if (uploadResult.fileInfo) {
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
        if (downloadUrl && editor.value) {
          const isImage = file.type.startsWith('image/')
          const isVideo = file.type.startsWith('video/')

          if (isImage) {
            editor.value.chain().focus().setImage({ src: downloadUrl, alt: file.name }).run()
          } else if (isVideo) {
            editor.value.chain().focus().setVideo({
              src: downloadUrl,
              alt: file.name,
              controls: true
            }).run()
          } else {
            editor.value.chain().focus().setLink({ href: downloadUrl }).insertContent(file.name).run()
          }

          ElMessage.success(`${file.name} 上传成功`)
        } else {
          throw new Error('获取下载地址失败')
        }
      }
    } catch (error: any) {
      console.error('RichTextEditor', `${source === 'paste' ? '粘贴' : '拖拽'}文件上传失败`, error)
      ElMessage.error(`上传 ${file.name} 失败: ${error?.message || '未知错误'}`)
    }
  }
}

// 拖拽悬停（视觉反馈）
function handleDragOver(event: DragEvent) {
  if (event.dataTransfer?.types.includes('Files')) {
    isDragging.value = true
    event.preventDefault()
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'copy'
    }
  }
}

// 拖拽离开
function handleDragLeave(event: DragEvent) {
  // 只有当离开编辑器容器时才取消拖拽状态
  const relatedTarget = event.relatedTarget as HTMLElement
  const currentTarget = event.currentTarget as HTMLElement | null
  if (!relatedTarget || (currentTarget && !currentTarget.contains(relatedTarget))) {
    isDragging.value = false
  }
}

// 编辑器容器上的 drop 事件（作为备用，主要处理在 editorProps.handleDrop 中）
function handleEditorDrop(event: DragEvent) {
  isDragging.value = false
  // 实际处理在 editorProps.handleDrop 中，这里只是重置状态
}

// 监听外部值变化
watch(() => props.modelValue, (newValue) => {
  if (editor.value && editor.value.getHTML() !== newValue) {
    editor.value.commands.setContent(newValue || '')
  }
})

// 链接相关
const linkDialogVisible = ref(false)
const linkUrl = ref('')

const handleInsertLink = () => {
  linkUrl.value = ''
  linkDialogVisible.value = true
}

const confirmInsertLink = () => {
  if (linkUrl.value && editor.value) {
    editor.value.chain().focus().setLink({ href: linkUrl.value }).run()
    linkDialogVisible.value = false
    linkUrl.value = ''
  }
}

// 图片相关
const imageDialogVisible = ref(false)
const imageUrl = ref('')

const handleInsertImage = () => {
  imageUrl.value = ''
  imageDialogVisible.value = true
}

const confirmInsertImage = () => {
  if (imageUrl.value && editor.value) {
    editor.value.chain().focus().setImage({ src: imageUrl.value }).run()
    imageDialogVisible.value = false
    imageUrl.value = ''
  }
}

// 标题处理
const handleHeading = (command: string) => {
  if (!editor.value) return

  if (command === 'paragraph') {
    editor.value.chain().focus().setParagraph().run()
  } else if (command === 'heading-1') {
    editor.value.chain().focus().toggleHeading({ level: 1 }).run()
  } else if (command === 'heading-2') {
    editor.value.chain().focus().toggleHeading({ level: 2 }).run()
  } else if (command === 'heading-3') {
    editor.value.chain().focus().toggleHeading({ level: 3 }).run()
  }
}

onBeforeUnmount(() => {
  if (editor.value) {
    editor.value.destroy()
  }
})
</script>

<style scoped>
.rich-text-editor {
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-base);
  background: var(--el-bg-color);
}

.editor-toolbar {
  display: flex;
  align-items: center;
  padding: 8px;
  border-bottom: 1px solid var(--el-border-color);
  gap: 8px;
  flex-wrap: wrap;
}

.toolbar-group {
  display: flex;
  align-items: center;
}

.toolbar-divider {
  width: 1px;
  height: 24px;
  background: var(--el-border-color);
  margin: 0 4px;
}

.editor-content {
  min-height: 300px;
  padding: 16px;
  transition: all 0.3s ease;
  position: relative;
}

.editor-content.is-dragging {
  background-color: var(--el-color-primary-light-9);
  border: 2px dashed var(--el-color-primary);
  border-radius: var(--el-border-radius-base);
}

.editor-content.is-dragging::before {
  content: '释放文件以上传';
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 16px;
  color: var(--el-color-primary);
  font-weight: 500;
  z-index: 10;
  pointer-events: none;
  background: var(--el-bg-color);
  padding: 8px 16px;
  border-radius: var(--el-border-radius-base);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.editor-content :deep(.ProseMirror) {
  outline: none;
  min-height: 300px;
  transition: opacity 0.3s ease;
}

.editor-content.is-dragging :deep(.ProseMirror) {
  opacity: 0.5;
}

.editor-content :deep(.ProseMirror p.is-editor-empty:first-child::before) {
  color: var(--el-text-color-placeholder);
  content: attr(data-placeholder);
  float: left;
  height: 0;
  pointer-events: none;
}

.editor-content :deep(.rich-text-link) {
  color: var(--el-color-primary);
  text-decoration: underline;
  cursor: pointer;
}

.editor-content :deep(.rich-text-image) {
  max-width: 100%;
  height: auto;
  display: block;
  margin: 16px 0;
}

.editor-content :deep(.rich-text-video),
.editor-content :deep(.ProseMirror video) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
  margin: 8px 0;
  display: block;
  background-color: #000;
}

.file-info {
  margin-top: 16px;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: var(--el-border-radius-base);
}

.file-info p {
  margin: 4px 0;
  font-size: 14px;
  color: var(--el-text-color-regular);
}
</style>
