<!--
  RichTextEditorWidget - 富文本编辑器组件
  🔥 完全新增，不依赖旧代码
  
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
      <div v-if="editor" class="editor-toolbar">
        <!-- 预览切换按钮 -->
        <div class="toolbar-group" style="margin-right: auto;">
          <el-tooltip :content="isPreviewMode ? '编辑模式' : '预览模式'" placement="bottom">
            <button
              type="button"
              @click="togglePreview"
              class="toolbar-button preview-toggle"
              :class="{ 'is-active': isPreviewMode }"
            >
              <el-icon v-if="!isPreviewMode"><View /></el-icon>
              <el-icon v-else><Edit /></el-icon>
            </button>
          </el-tooltip>
        </div>
        
        <div class="toolbar-divider"></div>
        
        <!-- 工具栏按钮（预览模式下隐藏） -->
        <template v-if="!isPreviewMode">
        <!-- 文本格式组 -->
        <div class="toolbar-group">
          <el-tooltip content="粗体" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleBold().run()"
              :class="{ 'is-active': editor.isActive('bold') }"
              class="toolbar-button"
            >
              <strong style="font-size: 14px;">B</strong>
            </button>
          </el-tooltip>
          <el-tooltip content="斜体" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleItalic().run()"
              :class="{ 'is-active': editor.isActive('italic') }"
              class="toolbar-button"
            >
              <em style="font-size: 14px;">I</em>
            </button>
          </el-tooltip>
          <el-tooltip content="删除线" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleStrike().run()"
              :class="{ 'is-active': editor.isActive('strike') }"
              class="toolbar-button"
            >
              <s style="font-size: 14px;">S</s>
            </button>
          </el-tooltip>
          <el-tooltip content="下划线" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleUnderline().run()"
              :class="{ 'is-active': editor.isActive('underline') }"
              class="toolbar-button"
            >
              <u style="font-size: 14px;">U</u>
            </button>
          </el-tooltip>
        </div>
        
        <div class="toolbar-divider"></div>
        
        <!-- 标题组 -->
        <div class="toolbar-group">
          <el-tooltip content="正文" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().setParagraph().run()"
              :class="{ 'is-active': editor.isActive('paragraph') }"
              class="toolbar-button"
            >
              <el-icon><Document /></el-icon>
            </button>
          </el-tooltip>
          <el-tooltip content="标题 1" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleHeading({ level: 1 }).run()"
              :class="{ 'is-active': editor.isActive('heading', { level: 1 }) }"
              class="toolbar-button"
            >
              <span class="heading-text">H1</span>
            </button>
          </el-tooltip>
          <el-tooltip content="标题 2" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleHeading({ level: 2 }).run()"
              :class="{ 'is-active': editor.isActive('heading', { level: 2 }) }"
              class="toolbar-button"
            >
              <span class="heading-text">H2</span>
            </button>
          </el-tooltip>
          <el-tooltip content="标题 3" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleHeading({ level: 3 }).run()"
              :class="{ 'is-active': editor.isActive('heading', { level: 3 }) }"
              class="toolbar-button"
            >
              <span class="heading-text">H3</span>
            </button>
          </el-tooltip>
        </div>
        
        <div class="toolbar-divider"></div>
        
        <!-- 列表组 -->
        <div class="toolbar-group">
          <el-tooltip content="无序列表" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleBulletList().run()"
              :class="{ 'is-active': editor.isActive('bulletList') }"
              class="toolbar-button"
            >
              <el-icon><List /></el-icon>
            </button>
          </el-tooltip>
          <el-tooltip content="有序列表" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleOrderedList().run()"
              :class="{ 'is-active': editor.isActive('orderedList') }"
              class="toolbar-button"
            >
              <el-icon><Sort /></el-icon>
            </button>
          </el-tooltip>
          <el-tooltip content="任务列表" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleTaskList().run()"
              :class="{ 'is-active': editor.isActive('taskList') }"
              class="toolbar-button"
            >
              <el-icon><CircleCheck /></el-icon>
            </button>
          </el-tooltip>
          <el-tooltip content="引用" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleBlockquote().run()"
              :class="{ 'is-active': editor.isActive('blockquote') }"
              class="toolbar-button"
            >
              <el-icon><ChatLineRound /></el-icon>
            </button>
          </el-tooltip>
        </div>
        
        <div class="toolbar-divider"></div>
        
        <!-- 对齐组 -->
        <div class="toolbar-group">
          <el-tooltip content="左对齐" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().setTextAlign('left').run()"
              :class="{ 'is-active': editor.isActive({ textAlign: 'left' }) }"
              class="toolbar-button"
            >
              <span style="font-size: 14px; font-weight: bold;">◀</span>
            </button>
          </el-tooltip>
          <el-tooltip content="居中" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().setTextAlign('center').run()"
              :class="{ 'is-active': editor.isActive({ textAlign: 'center' }) }"
              class="toolbar-button"
            >
              <span style="font-size: 14px; font-weight: bold;">⬌</span>
            </button>
          </el-tooltip>
          <el-tooltip content="右对齐" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().setTextAlign('right').run()"
              :class="{ 'is-active': editor.isActive({ textAlign: 'right' }) }"
              class="toolbar-button"
            >
              <span style="font-size: 14px; font-weight: bold;">▶</span>
            </button>
          </el-tooltip>
        </div>
        
        <div class="toolbar-divider"></div>
        
        <!-- 代码组 -->
        <div class="toolbar-group">
          <el-tooltip content="行内代码" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleCode().run()"
              :class="{ 'is-active': editor.isActive('code') }"
              class="toolbar-button"
            >
              <span style="font-size: 12px; font-family: monospace;">&lt;/&gt;</span>
            </button>
          </el-tooltip>
          <el-tooltip content="代码块" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleCodeBlock().run()"
              :class="{ 'is-active': editor.isActive('codeBlock') }"
              class="toolbar-button"
            >
              <el-icon><Operation /></el-icon>
            </button>
          </el-tooltip>
        </div>
        
        <div class="toolbar-divider"></div>
        
        <!-- 颜色组 -->
        <div class="toolbar-group">
          <el-tooltip content="文字颜色" placement="bottom">
            <div class="color-picker-wrapper">
              <input
                type="color"
                :value="getTextColor()"
                @input="handleTextColorChange"
                class="color-picker-input"
              />
              <button
                type="button"
                class="toolbar-button color-picker-button"
                :style="{ color: getTextColor() }"
              >
                A
              </button>
            </div>
          </el-tooltip>
          <el-tooltip content="背景高亮" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().toggleHighlight().run()"
              :class="{ 'is-active': editor.isActive('highlight') }"
              class="toolbar-button"
            >
              <span style="background-color: yellow; padding: 2px 4px; border-radius: 2px;">高</span>
            </button>
          </el-tooltip>
        </div>
        
        <div class="toolbar-divider"></div>
        
        <!-- 插入组 -->
        <div class="toolbar-group">
          <el-tooltip content="链接" placement="bottom">
            <button
              type="button"
              @click="handleSetLink"
              :class="{ 'is-active': editor.isActive('link') }"
              class="toolbar-button"
            >
              <el-icon><LinkIcon /></el-icon>
            </button>
          </el-tooltip>
          <el-tooltip content="表格" placement="bottom">
            <el-dropdown trigger="click" @command="handleTableCommand" placement="bottom-start">
              <button
                type="button"
                :class="{ 'is-active': editor.isActive('table') }"
                class="toolbar-button"
              >
                <el-icon><Grid /></el-icon>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="insert">
                    <el-icon style="margin-right: 8px;"><Plus /></el-icon>
                    插入表格 (3x3)
                  </el-dropdown-item>
                  <el-dropdown-item 
                    command="addColumnBefore" 
                    :disabled="!editor.isActive('table')"
                    divided
                  >
                    <el-icon style="margin-right: 8px;"><Plus /></el-icon>
                    左侧插入列
                  </el-dropdown-item>
                  <el-dropdown-item command="addColumnAfter" :disabled="!editor.isActive('table')">
                    <el-icon style="margin-right: 8px;"><Plus /></el-icon>
                    右侧插入列
                  </el-dropdown-item>
                  <el-dropdown-item command="deleteColumn" :disabled="!editor.isActive('table')">
                    <el-icon style="margin-right: 8px;"><Remove /></el-icon>
                    删除当前列
                  </el-dropdown-item>
                  <el-dropdown-item 
                    command="addRowBefore" 
                    :disabled="!editor.isActive('table')"
                    divided
                  >
                    <el-icon style="margin-right: 8px;"><Plus /></el-icon>
                    上方插入行
                  </el-dropdown-item>
                  <el-dropdown-item command="addRowAfter" :disabled="!editor.isActive('table')">
                    <el-icon style="margin-right: 8px;"><Plus /></el-icon>
                    下方插入行
                  </el-dropdown-item>
                  <el-dropdown-item command="deleteRow" :disabled="!editor.isActive('table')">
                    <el-icon style="margin-right: 8px;"><Remove /></el-icon>
                    删除当前行
                  </el-dropdown-item>
                  <el-dropdown-item 
                    command="deleteTable" 
                    :disabled="!editor.isActive('table')"
                    divided
                  >
                    <el-icon style="margin-right: 8px;"><Delete /></el-icon>
                    删除表格
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </el-tooltip>
          <el-tooltip content="分隔线" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().setHorizontalRule().run()"
              class="toolbar-button"
            >
              <el-icon><Minus /></el-icon>
            </button>
          </el-tooltip>
        </div>
        
        <div class="toolbar-divider"></div>
        
        <!-- 操作组 -->
        <div class="toolbar-group">
          <el-tooltip content="清除格式" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().clearNodes().unsetAllMarks().run()"
              class="toolbar-button"
            >
              <el-icon><Delete /></el-icon>
            </button>
          </el-tooltip>
          <el-tooltip content="撤销" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().undo().run()"
              :disabled="!editor.can().undo()"
              class="toolbar-button"
            >
              <el-icon><RefreshLeft /></el-icon>
            </button>
          </el-tooltip>
          <el-tooltip content="重做" placement="bottom">
            <button
              type="button"
              @click="editor.chain().focus().redo().run()"
              :disabled="!editor.can().redo()"
              class="toolbar-button"
            >
              <el-icon><RefreshRight /></el-icon>
            </button>
          </el-tooltip>
        </div>
        </template>
      </div>
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
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { Logger } from '@/core/utils/logger'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import type { EditorView } from 'prosemirror-view'
import type { Slice } from 'prosemirror-model'
import StarterKit from '@tiptap/starter-kit'
import { Link } from '@tiptap/extension-link'
import { Image } from '@tiptap/extension-image'
import { Video } from '@/shared/tiptap/VideoExtension'
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
import { ElInput, ElIcon, ElTooltip, ElMessageBox, ElDropdown, ElDropdownMenu, ElDropdownItem, ElDialog, ElMessage } from 'element-plus'
import CommonUpload from '@/shared/components/CommonUpload.vue'
import { uploadFile, notifyUploadComplete } from '@/utils/upload'
import {
  Document,
  List,
  Sort,
  ChatLineRound,
  Link as LinkIcon,
  Grid,
  Minus,
  RefreshLeft,
  RefreshRight,
  Operation,
  Delete,
  CircleCheck,
  Plus,
  Remove,
  View,
  Edit
} from '@element-plus/icons-vue'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import type { RichTextWidgetConfig } from '@/core/types/widget-configs'
import { sanitizeHtml } from '@/utils/sanitizeHtml'

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

// 文件上传对话框
const fileUploadDialogVisible = ref(false)
const uploadedFileUrl = ref<string>('')
const uploadedFileInfo = ref<{
  fileName: string
  fileType: string
  fileSize: number
} | null>(null)
const fileUploadRef = ref<InstanceType<typeof CommonUpload> | null>(null)

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

// 判断文件是否为图片
function isImageFile(fileName: string): boolean {
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico'].includes(ext)
}

// 判断文件是否为视频
function isVideoFile(fileName: string): boolean {
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  return ['mp4', 'avi', 'mov', 'wmv', 'flv', 'mkv', 'webm', 'm4v', '3gp'].includes(ext)
}

// 格式化文件大小
function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

// 获取视频 MIME 类型
function getVideoMimeType(ext: string): string {
  const mimeTypes: Record<string, string> = {
    'mp4': 'video/mp4',
    'webm': 'video/webm',
    'ogg': 'video/ogg',
    'avi': 'video/x-msvideo',
    'mov': 'video/quicktime',
    'wmv': 'video/x-ms-wmv',
    'flv': 'video/x-flv',
    'mkv': 'video/x-matroska',
    'm4v': 'video/x-m4v',
    '3gp': 'video/3gpp'
  }
  return mimeTypes[ext.toLowerCase()] || 'video/mp4'
}

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

// 处理插入文件
function handleInsertFile(): void {
  if (!editor.value) return
  
  // 打开上传对话框
  fileUploadDialogVisible.value = true
  uploadedFileUrl.value = '' // 重置上传的文件 URL
  uploadedFileInfo.value = null // 重置文件信息
}

// 文件上传变化（包括成功和失败）
function handleFileUploadChange(url: string | null): void {
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
    Logger.warn('RichTextWidget', '解析文件信息失败', error)
    uploadedFileInfo.value = {
      fileName: '未知文件',
      fileType: '未知',
      fileSize: 0
    }
  }
}

// 文件上传成功
function handleFileUploadSuccess(url: string): void {
  uploadedFileUrl.value = url
  handleFileUploadChange(url)
}

// 文件上传失败
function handleFileUploadError(error: Error): void {
  Logger.error('RichTextWidget', '文件上传失败', error)
  uploadedFileInfo.value = null
}

// 确认插入文件
function handleConfirmFileInsert(): void {
  if (!editor.value || !uploadedFileUrl.value) return
  
  const fileName = uploadedFileInfo.value?.fileName || '文件'
  const isImage = uploadedFileInfo.value ? isImageFile(uploadedFileInfo.value.fileName) : false
  const isVideo = uploadedFileInfo.value ? isVideoFile(uploadedFileInfo.value.fileName) : false
  
  if (isImage) {
    // 图片：插入为图片标签
    editor.value.chain().focus().setImage({ src: uploadedFileUrl.value, alt: fileName }).run()
  } else if (isVideo) {
    // 视频：使用 Video 扩展插入视频
    editor.value.chain().focus().setVideo({ 
      src: uploadedFileUrl.value,
      alt: fileName,
      controls: true
    }).run()
  } else {
    // 其他文件：插入为链接
    editor.value.chain().focus().setLink({ href: uploadedFileUrl.value }).insertContent(fileName).run()
  }
  
  // 关闭对话框并重置
  fileUploadDialogVisible.value = false
  uploadedFileUrl.value = ''
  uploadedFileInfo.value = null
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

// 获取文字颜色
function getTextColor(): string {
  if (!editor.value) return '#000000'
  return editor.value.getAttributes('textStyle').color || '#000000'
}

// 处理文字颜色变化
function handleTextColorChange(event: Event): void {
  if (!editor.value) return
  const target = event.target as HTMLInputElement
  editor.value.chain().focus().setColor(target.value).run()
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
  let cleanedHtml = html
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

.editor-toolbar {
  display: flex;
  align-items: center;
  gap: 0;
  padding: 8px 12px;
  background: linear-gradient(to bottom, var(--el-fill-color-lighter), var(--el-fill-color));
  border-bottom: 1px solid var(--el-border-color);
  flex-wrap: wrap;
}

.toolbar-group {
  display: flex;
  align-items: center;
  gap: 2px;
}

.toolbar-divider {
  width: 1px;
  height: 24px;
  background-color: var(--el-border-color);
  margin: 0 8px;
}

.toolbar-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background-color: transparent;
  color: var(--el-text-color-regular);
  cursor: pointer;
  font-size: 16px;
  transition: all 0.2s;
}

.toolbar-button:hover:not(:disabled) {
  background-color: var(--el-fill-color);
  color: var(--el-color-primary);
}

.toolbar-button.is-active {
  background-color: var(--el-color-primary);
  color: var(--el-color-white);
}

.toolbar-button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.heading-text {
  font-size: 12px;
  font-weight: bold;
}

.color-picker-wrapper {
  position: relative;
  display: inline-block;
}

.color-picker-input {
  position: absolute;
  width: 32px;
  height: 32px;
  opacity: 0;
  cursor: pointer;
}

.color-picker-button {
  position: relative;
  font-weight: bold;
  font-size: 16px;
}

.editor-content {
  min-height: v-bind('editorHeight + "px"');
}

.editor-content :deep(.ProseMirror) {
  outline: none;
  min-height: v-bind('editorHeight + "px"');
  padding: 16px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
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

.file-info {
  margin-top: 16px;
  padding: 12px;
  background-color: var(--el-fill-color-lighter);
  border-radius: 4px;
  font-size: 14px;
}

.file-info p {
  margin: 4px 0;
  color: var(--el-text-color-primary);
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

.word-count {
  user-select: none;
}

.upload-hint {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  max-width: 320px;
  text-align: right;
}

.preview-toggle {
  margin-right: 8px;
}

.preview-content {
  min-height: v-bind('editorHeight + "px"');
  padding: 16px;
  background-color: var(--el-bg-color);
  overflow-y: auto;
  max-height: 600px;
}

.preview-content .html-content {
  width: 100%;
  word-wrap: break-word;
}
</style>
