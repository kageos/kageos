<!--
  RichTextEditorWidget - 富文本编辑器组件
  🔥 统一架构组件
  
  功能：
  - 编辑模式：显示为 TipTap 富文本编辑器
  - 响应模式：显示 HTML 内容
  - 表格单元格模式：显示 HTML 内容（简化）
  - 详情模式：显示 HTML 内容
  - 搜索模式：文本输入框（搜索 HTML 内容）
-->

<template>
  <div class="rich-text-widget">
    <!-- 编辑模式：TipTap 编辑器 -->
    <div v-if="mode === 'edit'" class="editor-container">
      <RichTextEditorToolbar
        v-if="editor"
        :editor="editor"
        :is-preview-mode="isPreviewMode"
        :text-color="textColor"
        @toggle-preview="togglePreview"
        @set-link="handleSetLink"
        @table-command="handleTableCommand"
        @text-color-change="handleTextColorChange"
      />
      <!-- 编辑模式 -->
      <editor-content v-if="!isPreviewMode" :editor="editor" class="editor-content" />
      <!-- 预览模式 -->
      <div v-else class="preview-content">
        <div v-if="htmlContent" class="html-content" v-html="htmlContent"></div>
        <div v-else class="empty-preview">
          <p style="color: var(--el-text-color-placeholder);">暂无内容</p>
        </div>
      </div>
      <!-- 字数统计与上传说明 -->
      <div v-if="editor && !isPreviewMode" class="editor-footer">
        <span class="word-count">
          字数：{{ getWordCount() }} | 字符：{{ getCharCount() }}
        </span>
        <span class="upload-hint">
          支持拖拽文件到编辑区上传，或粘贴剪贴板中的图片/文件上传
        </span>
      </div>
    </div>
    
    <!-- 响应模式（只读） -->
    <div v-else-if="mode === 'response'" class="response-value">
      <div v-if="htmlContent" class="html-content" v-html="htmlContent"></div>
      <span v-else class="empty-text">-</span>
    </div>
    
    <!-- 表格单元格模式：显示 HTML 内容（简化） -->
    <div v-else-if="mode === 'table-cell'" class="table-cell-value">
      <div v-if="htmlContent" class="html-content-preview" v-html="stripHtml(htmlContent)"></div>
      <span v-else class="empty-text">-</span>
    </div>
    
    <!-- 详情模式：显示 HTML 内容 -->
    <div v-else-if="mode === 'detail'" class="detail-value">
      <div v-if="htmlContent" class="html-content" v-html="htmlContent"></div>
      <span v-else class="empty-text">-</span>
    </div>
    
    <!-- 搜索模式：文本输入 -->
    <el-input
      v-else-if="mode === 'search'"
      v-model="searchValue"
      :placeholder="`搜索${field.name}`"
      :clearable="true"
      @input="handleSearchChange"
      @clear="handleSearchClear"
    />
    
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { Logger } from '@/architecture/runtime/utils/logger'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import type { EditorView } from 'prosemirror-view'
import type { Slice } from 'prosemirror-model'
import StarterKit from '@tiptap/starter-kit'
import { Link } from '@tiptap/extension-link'
import { Image } from '@tiptap/extension-image'
import { Video } from '@/architecture/presentation/shared/tiptap/VideoExtension'
import { Table } from '@tiptap/extension-table'
import { TableRow } from '@tiptap/extension-table-row'
import { TableCell } from '@tiptap/extension-table-cell'
import { TableHeader } from '@tiptap/extension-table-header'
import { Underline } from '@tiptap/extension-underline'
import { CodeBlock } from '@tiptap/extension-code-block'
import { Code } from '@tiptap/extension-code'
import { TextStyle } from '@tiptap/extension-text-style'
import { Color } from '@tiptap/extension-color'
import { Highlight } from '@tiptap/extension-highlight'
import { TextAlign } from '@tiptap/extension-text-align'
import { TaskList } from '@tiptap/extension-task-list'
import { TaskItem } from '@tiptap/extension-task-item'
import { Placeholder } from '@tiptap/extension-placeholder'
import { Dropcursor } from '@tiptap/extension-dropcursor'
import { Gapcursor } from '@tiptap/extension-gapcursor'
import { ElMessage } from 'element-plus'
import { uploadFile, notifyUploadComplete } from '@/architecture/infrastructure/upload'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/architecture/runtime/stores/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import type { RichTextWidgetConfig } from '@/architecture/runtime/types/widget-configs'
import { sanitizeHtml } from '@/architecture/runtime/utils/sanitizeHtml'
import RichTextEditorToolbar from './RichTextEditorToolbar.vue'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 预览模式
const isPreviewMode = ref(false)

// 切换预览模式
function togglePreview(): void {
  isPreviewMode.value = !isPreviewMode.value
}

// 文件上传路由（从 localStorage 获取用户名）
const fileUploadRouter = computed(() => {
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

// 获取配置（带类型）
const config = computed(() => {
  return (props.field.widget?.config || {}) as RichTextWidgetConfig
})

// 编辑器高度
const editorHeight = computed(() => {
  const height = config.value.height
  if (height && typeof height === 'number' && height > 0) {
    return height
  }
  return 300 // 默认300px
})

// HTML 内容（用于显示）
const htmlContent = computed(() => {
  const fieldValue = props.value || (props as any).modelValue
  if (!fieldValue) {
    return ''
  }
  
  const raw = fieldValue.raw
  if (raw === null || raw === undefined || raw === '') {
    return ''
  }
  
  const html = String(raw)
  
  // 🔥 对于非编辑模式，清理 HTML 以避免触发资源加载
  // 编辑模式下保留原始 HTML（因为用户可能需要编辑）
  if (props.mode === 'edit') {
    return html
  }
  
  // 其他模式（response、detail、table-cell 等）清理 HTML
  return sanitizeHtml(html)
})

// TipTap 编辑器（使用完整工具栏，最高级模式）
// 🔥 修复：StarterKit 已经包含了 link, code, codeBlock, dropCursor, gapCursor
// 需要排除它们，使用自定义配置的版本
const editor = useEditor({
  extensions: [
    StarterKit.configure({
      // 排除 StarterKit 中已包含的扩展，使用自定义配置的版本
      link: false,
      code: false,
      codeBlock: false,
      dropcursor: false,
      gapcursor: false,
      underline: false, // 🔥 排除 underline，使用自定义的 Underline
    }),
    Underline,
    Code, // 单独添加，使用默认配置
    CodeBlock.configure({
      HTMLAttributes: {
        class: 'rich-text-code-block'
      }
    }),
    TextStyle,
    Color,
    Highlight.configure({
      multicolor: true
    }),
    TextAlign.configure({
      types: ['heading', 'paragraph']
    }),
    TaskList,
    TaskItem.configure({
      nested: true
    }),
    Placeholder.configure({
      placeholder: '请输入内容，支持拖拽文件到此处或粘贴图片/文件上传'
    }),
    Dropcursor, // 单独添加
    Gapcursor, // 单独添加
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
      allowBase64: false // 🔥 禁用 base64，强制使用 URL（避免文件过大）
    }),
    Video.configure({
      HTMLAttributes: {
        class: 'rich-text-video'
      },
      inline: false,
      allowBase64: false
    }),
    Table.configure({
      resizable: true,
      HTMLAttributes: {
        class: 'rich-text-table'
      }
    }),
    TableRow,
    TableHeader,
    TableCell,
  ],
  content: htmlContent.value || '',
  editorProps: {
    attributes: {
      class: 'prose prose-sm sm:prose lg:prose-lg xl:prose-2xl mx-auto focus:outline-none',
      style: `min-height: ${editorHeight.value}px; padding: 16px;`,
      placeholder: '请输入内容，支持拖拽文件到此处或粘贴图片/文件上传'
    },
    // 优化粘贴处理：支持从 Word、网页、Markdown 等粘贴，自动清理格式
    // 特别处理：检测粘贴的文件（任意类型），自动上传而不是使用 base64
    handlePaste: (async (view: EditorView, event: ClipboardEvent, slice: Slice) => {
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
          
          // 处理每个文件
          for (const item of fileItems) {
            const file = item.getAsFile()
            if (!file) continue
            
            try {
              // 显示上传提示
              ElMessage.info(`正在上传 ${file.name}...`)
              
              // 上传文件
              const uploadResult = await uploadFile(
                fileUploadRouter.value,
                file,
                () => {} // 粘贴上传不显示进度
              )
              
              // 通知后端上传完成
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
                  // 判断文件类型并插入
                  const isImage = file.type.startsWith('image/')
                  const isVideo = file.type.startsWith('video/')
                  
                  if (isImage) {
                    // 图片：插入为图片
                    editor.value.chain().focus().setImage({ src: downloadUrl, alt: file.name }).run()
                  } else if (isVideo) {
                    // 视频：插入为视频
                    editor.value.chain().focus().setVideo({ 
                      src: downloadUrl,
                      alt: file.name,
                      controls: true
                    }).run()
                  } else {
                    // 其他文件：插入为链接
                    editor.value.chain().focus().setLink({ href: downloadUrl }).insertContent(file.name).run()
                  }
                  
                  ElMessage.success(`${file.name} 上传成功`)
                } else {
                  throw new Error('获取下载地址失败')
                }
              }
            } catch (error: any) {
              Logger.error('RichTextWidget', '粘贴文件上传失败', error)
              ElMessage.error(`上传 ${file.name} 失败: ${error?.message || '未知错误'}`)
            }
          }
          
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
        Logger.error('RichTextWidget', '粘贴处理失败', error)
        return false
      }
    }) as any,
    // 支持拖拽粘贴文件（任意类型），自动上传
    handleDrop: (async (view: EditorView, event: DragEvent, slice: Slice, moved: boolean) => {
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
      
      // 处理每个文件（任意类型）
      const files = Array.from(dataTransfer.files || []) as File[]
      for (const file of files) {
        try {
          // 显示上传提示
          ElMessage.info(`正在上传 ${file.name}...`)
          
          // 上传文件
          const uploadResult = await uploadFile(
            fileUploadRouter.value,
            file,
            () => {} // 拖拽上传不显示进度
          )
          
          // 通知后端上传完成
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
              // 判断文件类型并插入
              const isImage = file.type.startsWith('image/')
              const isVideo = file.type.startsWith('video/')
              
              if (isImage) {
                // 图片：插入为图片
                editor.value.chain().focus().setImage({ src: downloadUrl, alt: file.name }).run()
              } else if (isVideo) {
                // 视频：插入为视频
                editor.value.chain().focus().setVideo({ 
                  src: downloadUrl,
                  alt: file.name,
                  controls: true
                }).run()
              } else {
                // 其他文件：插入为链接
                editor.value.chain().focus().setLink({ href: downloadUrl }).insertContent(file.name).run()
              }
              
              ElMessage.success(`${file.name} 上传成功`)
            } else {
              throw new Error('获取下载地址失败')
            }
          }
        } catch (error: any) {
          Logger.error('RichTextWidget', '拖拽文件上传失败', error)
          ElMessage.error(`上传 ${file.name} 失败: ${error?.message || '未知错误'}`)
        }
      }
      
      return true // 已处理，阻止默认行为
    }) as any
  },
  onUpdate: ({ editor }) => {
    const html = editor.getHTML()
    // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
    const newFieldValue = createFieldValue(
      props.field,
      html,
      stripHtml(html) // 显示时去除 HTML 标签
    )
    
    formDataStore.setValue(props.fieldPath, newFieldValue)
    emit('update:modelValue', newFieldValue)
  }
})

// 处理链接
function handleSetLink(): void {
  if (!editor.value) return
  
  const previousUrl = editor.value.getAttributes('link').href
  const url = window.prompt('请输入链接地址', previousUrl)
  
  if (url === null) {
    return
  }
  
  if (url === '') {
    editor.value.chain().focus().extendMarkRange('link').unsetLink().run()
    return
  }
  
  editor.value.chain().focus().extendMarkRange('link').setLink({ href: url }).run()
}

// 处理表格命令
function handleTableCommand(command: string): void {
  if (!editor.value) return
  
  switch (command) {
    case 'insert':
      editor.value.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
      break
    case 'addColumnBefore':
      editor.value.chain().focus().addColumnBefore().run()
      break
    case 'addColumnAfter':
      editor.value.chain().focus().addColumnAfter().run()
      break
    case 'deleteColumn':
      editor.value.chain().focus().deleteColumn().run()
      break
    case 'addRowBefore':
      editor.value.chain().focus().addRowBefore().run()
      break
    case 'addRowAfter':
      editor.value.chain().focus().addRowAfter().run()
      break
    case 'deleteRow':
      editor.value.chain().focus().deleteRow().run()
      break
    case 'deleteTable':
      editor.value.chain().focus().deleteTable().run()
      break
  }
}

const textColor = computed(() => editor.value?.getAttributes('textStyle').color || '#000000')

// 处理文字颜色变化
function handleTextColorChange(color: string): void {
  if (!editor.value) return
  editor.value.chain().focus().setColor(color).run()
}

// 获取字数（不含HTML标签）
function getWordCount(): number {
  if (!editor.value) return 0
  const text = editor.value.getText()
  // 去除空白字符后计算
  return text.trim().split(/\s+/).filter(word => word.length > 0).length
}

// 获取字符数（不含HTML标签）
function getCharCount(): number {
  if (!editor.value) return 0
  return editor.value.getText().length
}

// 监听外部值变化（用于初始化或恢复值）
watch(
  () => htmlContent.value,
  (newValue) => {
    if (editor.value && editor.value.getHTML() !== newValue) {
      editor.value.commands.setContent(newValue || '')
    }
  },
  { immediate: true }
)

// 组件卸载时销毁编辑器
onBeforeUnmount(() => {
  if (editor.value) {
    editor.value.destroy()
  }
})
// 去除 HTML 标签（用于表格单元格显示）
function stripHtml(html: string): string {
  if (!html) return ''
  
  // 🔥 先过滤掉可能导致资源加载的标签（如 img、video、audio 等）
  // 这样可以避免浏览器尝试加载不存在的资源（如 localhost:63342 的 markdown 预览资源）
  const cleanedHtml = html
    // 移除 img 标签
    .replace(/<img[^>]*>/gi, '')
    // 移除 video 标签
    .replace(/<video[^>]*>.*?<\/video>/gi, '')
    // 移除 audio 标签
    .replace(/<audio[^>]*>.*?<\/audio>/gi, '')
    // 移除 iframe 标签
    .replace(/<iframe[^>]*>.*?<\/iframe>/gi, '')
    // 移除 script 标签
    .replace(/<script[^>]*>.*?<\/script>/gi, '')
    // 移除 style 标签
    .replace(/<style[^>]*>.*?<\/style>/gi, '')
  
  // 使用 DOMParser 来安全地解析 HTML（不会触发资源加载）
  try {
    const parser = new DOMParser()
    const doc = parser.parseFromString(cleanedHtml, 'text/html')
    return doc.body.textContent || doc.body.innerText || ''
  } catch (error) {
    // 如果 DOMParser 失败，使用传统方法（但先清理了资源标签）
    const tmp = document.createElement('DIV')
    tmp.innerHTML = cleanedHtml
    return tmp.textContent || tmp.innerText || ''
  }
}

// 搜索模式
const searchValue = ref<string>('')

function handleSearchChange(): void {
  const newFieldValue = searchValue.value
    ? {
        raw: searchValue.value,
        display: searchValue.value,
        meta: {}
      }
    : {
        raw: null,
        display: '',
        meta: {}
      }
  
  formDataStore.setValue(props.fieldPath, newFieldValue)
  emit('update:modelValue', newFieldValue)
}

function handleSearchClear(): void {
  searchValue.value = ''
  handleSearchChange()
}

// 监听搜索值恢复（用于 URL 恢复）
watch(
  () => props.value,
  (newValue: any) => {
    if (props.mode === 'search') {
      const raw = newValue?.raw
      if (raw !== null && raw !== undefined) {
        searchValue.value = String(raw)
      } else {
        searchValue.value = ''
      }
    }
  },
  { immediate: true, deep: true }
)
</script>

<style scoped>
.rich-text-widget {
  width: 100%;
}

.editor-container {
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  overflow: hidden;
  background-color: var(--el-bg-color);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

:global(.form-view-flat) .editor-container {
  border: 1px solid var(--app-auth-input-border);
  border-radius: 12px;
  background: var(--app-auth-input-bg);
  box-shadow: none;
  transition: all 0.3s ease;
}

.editor-content {
  min-height: v-bind('editorHeight + "px"');
}

:global(.form-view-flat) .editor-content {
  background: transparent;
  border: none;
  border-radius: 0;
}

.editor-content :deep(.ProseMirror) {
  outline: none;
  min-height: v-bind('editorHeight + "px"');
  padding: 16px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
}

:global(.form-view-flat) .editor-content :deep(.ProseMirror) {
  padding: 16px 18px;
}

.editor-content :deep(.ProseMirror p) {
  margin: 8px 0;
}

.editor-content :deep(.ProseMirror h1),
.editor-content :deep(.ProseMirror h2),
.editor-content :deep(.ProseMirror h3) {
  margin: 16px 0 8px 0;
  font-weight: bold;
  line-height: 1.4;
}

.editor-content :deep(.ProseMirror ul),
.editor-content :deep(.ProseMirror ol) {
  margin: 8px 0;
  padding-left: 24px;
}

.editor-content :deep(.ProseMirror blockquote) {
  border-left: 4px solid var(--el-color-primary);
  padding-left: 16px;
  margin: 16px 0;
  color: var(--el-text-color-secondary);
  font-style: italic;
}

.editor-content :deep(.ProseMirror .rich-text-link) {
  color: var(--el-color-primary);
  text-decoration: underline;
  cursor: pointer;
}

.editor-content :deep(.ProseMirror video) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
  margin: 8px 0;
}

.editor-content :deep(.ProseMirror .rich-text-video),
.editor-content :deep(.ProseMirror video) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
  margin: 8px 0;
  display: block;
  background-color: #000;
}

.editor-content :deep(.ProseMirror .rich-text-image) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
  margin: 8px 0;
}

.editor-content :deep(.ProseMirror .rich-text-table) {
  border-collapse: collapse;
  width: 100%;
  margin: 16px 0;
  border: 1px solid var(--el-border-color);
}

.editor-content :deep(.ProseMirror .rich-text-table td),
.editor-content :deep(.ProseMirror .rich-text-table th) {
  border: 1px solid var(--el-border-color);
  padding: 8px 12px;
  text-align: left;
}

.editor-content :deep(.ProseMirror .rich-text-table th) {
  background-color: var(--el-fill-color-lighter);
  font-weight: bold;
}

.editor-content :deep(.ProseMirror code) {
  background-color: var(--el-fill-color);
  color: var(--el-color-danger);
  padding: 2px 4px;
  border-radius: 3px;
  font-size: 0.9em;
  font-family: 'Courier New', monospace;
}

.editor-content :deep(.ProseMirror .rich-text-code-block) {
  background-color: var(--el-fill-color-darker);
  color: var(--el-text-color-primary);
  padding: 16px;
  border-radius: 4px;
  margin: 16px 0;
  overflow-x: auto;
  font-family: 'Courier New', monospace;
  font-size: 14px;
  line-height: 1.5;
  border: 1px solid var(--el-border-color);
}

.editor-content :deep(.ProseMirror mark) {
  background-color: #fef08a;
  padding: 2px 0;
  border-radius: 2px;
}

.editor-content :deep(.ProseMirror ul[data-type="taskList"]) {
  list-style: none;
  padding: 0;
}

.editor-content :deep(.ProseMirror li[data-type="taskItem"]) {
  display: flex;
  align-items: flex-start;
  margin: 8px 0;
}

.editor-content :deep(.ProseMirror li[data-type="taskItem"] > label) {
  flex: 0 0 auto;
  margin-right: 8px;
  user-select: none;
}

.editor-content :deep(.ProseMirror li[data-type="taskItem"] > div) {
  flex: 1 1 auto;
}

.editor-content :deep(.ProseMirror li[data-type="taskItem"][data-checked="true"] > div) {
  text-decoration: line-through;
  opacity: 0.6;
}

.editor-content :deep(.ProseMirror p.is-editor-empty:first-child::before) {
  color: var(--el-text-color-placeholder);
  content: attr(data-placeholder);
  float: left;
  height: 0;
  pointer-events: none;
}

.editor-content :deep(.ProseMirror p.is-editor-empty:first-child::before) {
  color: var(--el-text-color-placeholder);
  content: attr(data-placeholder);
  float: left;
  height: 0;
  pointer-events: none;
}

.response-value,
.detail-value {
  width: 100%;
}

.html-content {
  width: 100%;
  word-wrap: break-word;
}

.html-content :deep(p) {
  margin: 8px 0;
}

.html-content :deep(h1),
.html-content :deep(h2),
.html-content :deep(h3) {
  margin: 16px 0 8px 0;
  font-weight: bold;
}

.html-content :deep(ul),
.html-content :deep(ol) {
  margin: 8px 0;
  padding-left: 24px;
}

.html-content :deep(blockquote) {
  border-left: 4px solid var(--el-border-color);
  padding-left: 16px;
  margin: 8px 0;
  color: var(--el-text-color-secondary);
}

.html-content :deep(video) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
  margin: 8px 0;
  display: block;
  background-color: #000;
}

.html-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
  margin: 8px 0;
  display: block;
  /* 图片加载失败时的占位符 */
  background-color: var(--el-fill-color-lighter);
  /* 图片加载错误处理 */
  object-fit: contain;
}

.html-content :deep(img[src=""]) {
  display: none;
}

/* 图片加载失败时的样式 */
.html-content :deep(img:not([src])) {
  display: none;
}

.table-cell-value {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.html-content-preview {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-text {
  color: var(--el-text-color-placeholder);
}

.editor-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 8px 16px;
  background-color: var(--el-fill-color-lighter);
  border-top: 1px solid var(--el-border-color);
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

:global(.form-view-flat) .editor-footer {
  padding: 8px 14px 10px;
  background: transparent;
  border-top: 1px solid var(--app-auth-input-border);
}

.word-count {
  user-select: none;
}

.upload-hint {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  max-width: 320px;
  text-align: right;
}

.preview-content {
  min-height: v-bind('editorHeight + "px"');
  padding: 16px;
  background-color: var(--el-bg-color);
  overflow-y: auto;
  max-height: 600px;
}

:global(.form-view-flat) .preview-content {
  padding: 16px 18px;
  background: transparent;
  border: none;
  border-radius: 0;
}

.preview-content .html-content {
  width: 100%;
  word-wrap: break-word;
}
</style>
