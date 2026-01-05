<!--
  TextWidget - 文本显示组件
  用于输出参数，支持多种格式化显示（json、yaml、xml、markdown、html、csv等）
-->

<template>
  <div class="text-widget">
    <!-- 响应模式（主要使用场景） -->
    <div v-if="mode === 'response'" class="response-text">
      <div v-if="formattedContent || markdownContent || csvTableData" class="formatted-content" :class="formatClass">
        <!-- 超大文本处理：截断显示 + 弹窗 -->
        <template v-if="shouldTruncate && !isSpecialFormat">
          <div class="text-content-truncated">
            <pre v-if="isCodeFormat" class="code-content">{{ truncatedContent }}</pre>
            <div v-else class="text-content">{{ truncatedContent }}</div>
            <div class="text-actions">
              <el-button 
                type="primary" 
                link 
                size="small" 
                @click="showFullTextDialog = true"
              >
                查看全部 ({{ contentLength }} 字符)
              </el-button>
            </div>
          </div>
        </template>
        <!-- 正常显示 -->
        <template v-else>
          <pre v-if="isCodeFormat" class="code-content">{{ formattedContent }}</pre>
          <div v-else-if="format === 'html'" v-html="formattedContent" class="html-content"></div>
          <div v-else-if="format === 'markdown'" v-html="markdownContent" class="markdown-content"></div>
          <div v-else-if="format === 'csv' && csvTableData" class="csv-content">
            <table>
              <thead>
                <tr>
                  <th v-for="header in csvTableData.headers" :key="header">{{ header }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, rowIndex) in csvTableData.rows" :key="rowIndex">
                  <td v-for="(cell, cellIndex) in row" :key="cellIndex">{{ cell }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="text-content">{{ formattedContent }}</div>
        </template>
      </div>
      <span v-else class="empty-text">-</span>
      
      <!-- 全文显示弹窗 -->
      <el-dialog
        v-model="showFullTextDialog"
        :title="field.name || '查看全文'"
        width="80%"
        :close-on-click-modal="false"
        class="text-full-dialog"
      >
        <div class="dialog-content">
          <el-input
            v-model="editableContent"
            type="textarea"
            :rows="20"
            :placeholder="field.desc || '文本内容'"
            class="full-text-editor"
          />
        </div>
        <template #footer>
          <div class="dialog-footer">
            <el-button @click="handleCancelEdit">取消</el-button>
            <el-button type="primary" @click="handleSaveEdit">保存</el-button>
            <el-button type="info" @click="handleCopyToClipboard">复制</el-button>
          </div>
        </template>
      </el-dialog>
    </div>
    
    <!-- 表格单元格模式 -->
    <div v-else-if="mode === 'table-cell'" class="table-cell-text">
      <div v-if="formattedContent || markdownContent || csvTableData" class="formatted-content" :class="formatClass">
        <pre v-if="isCodeFormat" class="code-content">{{ formattedContent }}</pre>
        <div v-else-if="format === 'html'" v-html="formattedContent" class="html-content html-table-cell"></div>
        <div v-else-if="format === 'markdown'" v-html="markdownContent" class="markdown-content markdown-table-cell"></div>
        <div v-else-if="format === 'csv' && csvTableData" class="csv-content csv-table-cell">
          <!-- 列表模式下显示简化预览 -->
          <div class="csv-preview">
            <span class="csv-preview-text">
              {{ csvTableData.headers.join(' | ') }} ({{ csvTableData.rows.length }} 行)
            </span>
          </div>
        </div>
        <div v-else class="text-content">{{ formattedContent }}</div>
      </div>
      <span v-else class="empty-text">-</span>
    </div>
    
    <!-- 详情模式 -->
    <div v-else-if="mode === 'detail'" class="detail-text">
      <div class="detail-content">
        <!-- CSV 格式：优先渲染表格 -->
        <template v-if="format === 'csv'">
          <div v-if="csvTableData" class="csv-content csv-detail">
            <table>
              <thead>
                <tr>
                  <th v-for="header in csvTableData.headers" :key="header">{{ header }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(row, rowIndex) in csvTableData.rows" :key="rowIndex">
                  <td v-for="(cell, cellIndex) in row" :key="cellIndex">{{ cell }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="text-content">{{ formattedContent }}</div>
        </template>
        <!-- 其他格式 -->
        <div v-else-if="formattedContent || markdownContent" class="formatted-content" :class="formatClass">
          <pre v-if="isCodeFormat" class="code-content">{{ formattedContent }}</pre>
          <div v-else-if="format === 'html'" v-html="formattedContent" class="html-content html-detail"></div>
          <div v-else-if="format === 'markdown'" v-html="markdownContent" class="markdown-content markdown-detail"></div>
          <div v-else class="text-content">{{ formattedContent }}</div>
        </div>
        <span v-else class="empty-text">-</span>
      </div>
    </div>
    
    <!-- 编辑模式（通常不使用，但保留兼容性） -->
    <div v-else-if="mode === 'edit'" class="edit-text">
      <el-input
        v-model="internalValue"
        type="textarea"
        :rows="6"
        :placeholder="field.desc || `请输入${field.name}`"
        :disabled="false"
        @blur="handleBlur"
      />
    </div>
    
    <!-- 其他模式（默认显示） -->
    <div v-else class="default-text">
      <div v-if="formattedContent || markdownContent || csvTableData" class="formatted-content" :class="formatClass">
        <pre v-if="isCodeFormat" class="code-content">{{ formattedContent }}</pre>
        <div v-else-if="format === 'html'" v-html="formattedContent" class="html-content"></div>
        <div v-else-if="format === 'markdown'" v-html="markdownContent" class="markdown-content"></div>
        <div v-else-if="format === 'csv' && csvTableData" class="csv-content">
          <table>
            <thead>
              <tr>
                <th v-for="header in csvTableData.headers" :key="header">{{ header }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(row, rowIndex) in csvTableData.rows" :key="rowIndex">
                <td v-for="(cell, cellIndex) in row" :key="cellIndex">{{ cell }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="text-content">{{ formattedContent }}</div>
      </div>
      <span v-else class="empty-text">-</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElInput, ElButton, ElDialog } from 'element-plus'
import { ElMessage } from 'element-plus'
import { marked } from 'marked'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import type { TextWidgetConfig } from '@/core/types/widget-configs'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 获取配置（带类型）
const config = computed(() => {
  return (props.field.widget?.config || {}) as TextWidgetConfig
})

// 格式化类型
const format = computed(() => {
  const fmt = (config.value.format || '').toLowerCase()
  // 调试：在详情模式下输出 format 值
  if (props.mode === 'detail' && props.field.code === 'details_display') {
    console.log('[TextWidget] format computed:', {
      format: fmt,
      config: config.value,
      field: props.field,
      widget: props.field.widget,
      value: props.value
    })
  }
  return fmt
})

// 是否为代码格式（需要代码高亮）
// CSV 不应该作为代码格式，应该渲染成表格
const isCodeFormat = computed(() => {
  return ['json', 'yaml', 'xml', 'javascript', 'typescript', 'python', 'java', 'go', 'sql'].includes(format.value)
})

// 格式类名（用于样式）
const formatClass = computed(() => {
  return `format-${format.value || 'text'}`
})

// 原始内容
const rawContent = computed(() => {
  const value = props.value
  if (!value) {
    return ''
  }
  
  // 对于 CSV 格式，优先使用 raw 值（因为 display 可能是格式化后的文本）
  if (format.value === 'csv') {
    const raw = value.raw
    if (raw === null || raw === undefined || raw === '') {
      // 如果 raw 为空，尝试使用 display
      return value.display ? String(value.display) : ''
    }
    return String(raw)
  }
  
  // 其他格式：优先使用 display，如果没有则使用 raw
  if (value.display) {
    return String(value.display)
  }
  
  const raw = value.raw
  if (raw === null || raw === undefined || raw === '') {
    return ''
  }
  
  return String(raw)
})

// 格式化后的内容
const formattedContent = computed(() => {
  const content = rawContent.value
  if (!content) {
    return ''
  }
  
  const fmt = format.value
  
  // JSON 格式化
  if (fmt === 'json') {
    try {
      const parsed = typeof content === 'string' ? JSON.parse(content) : content
      return JSON.stringify(parsed, null, 2)
    } catch {
      return content
    }
  }
  
  // YAML 格式化（简单处理，实际可以使用 yaml 库）
  if (fmt === 'yaml') {
    return content
  }
  
  // XML 格式化（简单处理，实际可以使用 xml 格式化库）
  if (fmt === 'xml') {
    return content
  }
  
  // HTML 直接返回（使用 v-html 渲染）
  if (fmt === 'html') {
    return content
  }
  
  // Markdown 需要转换为 HTML（在 markdownContent computed 中处理）
  if (fmt === 'markdown') {
    return content
  }
  
  // CSV 格式化（简单处理）
  if (fmt === 'csv') {
    return content
  }
  
  // 其他格式直接返回
  return content
})

// Markdown 渲染后的 HTML 内容
const markdownContent = computed(() => {
  if (format.value !== 'markdown') {
    return ''
  }
  
  const content = formattedContent.value
  if (!content) {
    return ''
  }
  
  try {
    // 使用 marked 渲染 Markdown
    // 配置 marked 选项
    const markedOptions = {
      breaks: true, // 支持换行
      gfm: true // 支持 GitHub Flavored Markdown
    }
    
    return marked.parse(content, markedOptions) as string
  } catch (error) {
    console.error('[TextWidget] Markdown 渲染失败:', error)
    // 如果渲染失败，返回转义后的原始内容
    return content
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/\n/g, '<br>')
  }
})

// CSV 解析后的表格数据
interface CSVTableData {
  headers: string[]
  rows: string[][]
}

const csvTableData = computed<CSVTableData | null>(() => {
  // 调试：在详情模式下输出
  if (props.mode === 'detail' && props.field?.code === 'details_display') {
    console.log('[TextWidget] csvTableData computed:', {
      format: format.value,
      isCsv: format.value === 'csv',
      rawContent: rawContent.value,
      value: props.value,
      config: config.value
    })
  }
  
  if (format.value !== 'csv') {
    return null
  }
  
  const content = rawContent.value
  if (!content) {
    console.warn('[TextWidget] CSV format but no content', { 
      format: format.value, 
      content,
      rawContent: rawContent.value,
      value: props.value,
      field: props.field?.code
    })
    return null
  }
  
  try {
    const lines = content.trim().split('\n')
    if (lines.length === 0) {
      console.warn('[TextWidget] CSV format but empty lines', { content })
      return null
    }
    
    console.log('[TextWidget] Parsing CSV:', { 
      lines, 
      lineCount: lines.length,
      firstLine: lines[0],
      field: props.field?.code
    })
    
    // 解析 CSV 行（支持引号包裹的字段）
    const parseCSVLine = (line: string): string[] => {
      const result: string[] = []
      let current = ''
      let inQuotes = false
      
      for (let i = 0; i < line.length; i++) {
        const char = line[i]
        const nextChar = line[i + 1]
        
        if (char === '"') {
          if (inQuotes && nextChar === '"') {
            // 转义的引号
            current += '"'
            i++ // 跳过下一个引号
          } else {
            // 开始或结束引号
            inQuotes = !inQuotes
          }
        } else if (char === ',' && !inQuotes) {
          // 字段分隔符
          result.push(current)
          current = ''
        } else {
          current += char
        }
      }
      
      // 添加最后一个字段
      result.push(current)
      return result
    }
    
    // 解析表头
    const headers = parseCSVLine(lines[0])
    
    // 解析数据行
    const rows: string[][] = []
    for (let i = 1; i < lines.length; i++) {
      if (lines[i].trim()) {
        rows.push(parseCSVLine(lines[i]))
      }
    }
    
    const result = {
      headers,
      rows
    }
    console.log('[TextWidget] CSV parsed successfully:', result)
    return result
  } catch (error) {
    console.error('[TextWidget] CSV 解析失败:', error, { content, format: format.value })
    return null
  }
})

// 内部值（用于编辑模式）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit') {
      return rawContent.value
    }
    return ''
  },
  set: (newValue: string) => {
    if (props.mode === 'edit') {
      const fieldValue = {
        raw: newValue,
        display: newValue,
        meta: {}
      }
      
      formDataStore.setValue(props.fieldPath, fieldValue)
      emit('update:modelValue', fieldValue)
    }
  }
})

// 失焦处理
function handleBlur(): void {
  // 可以在这里添加验证逻辑
}

// 超大文本处理相关
const MAX_DISPLAY_LENGTH = 500 // 最大显示长度（字符数）
const showFullTextDialog = ref(false)
const editableContent = ref('')

// 是否为特殊格式（不需要截断的格式）
const isSpecialFormat = computed(() => {
  return format.value === 'html' || format.value === 'markdown' || format.value === 'csv'
})

// 内容长度
const contentLength = computed(() => {
  return rawContent.value.length
})

// 是否需要截断
const shouldTruncate = computed(() => {
  if (isSpecialFormat.value) {
    return false
  }
  return contentLength.value > MAX_DISPLAY_LENGTH
})

// 截断后的内容
const truncatedContent = computed(() => {
  if (!shouldTruncate.value) {
    return formattedContent.value
  }
  const content = formattedContent.value
  return content.substring(0, MAX_DISPLAY_LENGTH) + '...'
})

// 监听弹窗打开，初始化可编辑内容
watch(showFullTextDialog, (newVal) => {
  if (newVal) {
    editableContent.value = rawContent.value
  }
})

// 取消编辑
function handleCancelEdit(): void {
  showFullTextDialog.value = false
  editableContent.value = ''
}

// 保存编辑
function handleSaveEdit(): void {
  const fieldValue = {
    raw: editableContent.value,
    display: editableContent.value,
    meta: {}
  }
  
  formDataStore.setValue(props.fieldPath, fieldValue)
  emit('update:modelValue', fieldValue)
  
  ElMessage.success('保存成功')
  showFullTextDialog.value = false
}

// 复制到剪贴板
async function handleCopyToClipboard(): Promise<void> {
  try {
    await navigator.clipboard.writeText(editableContent.value)
    ElMessage.success('已复制到剪贴板')
  } catch (error) {
    console.error('复制失败:', error)
    ElMessage.error('复制失败，请手动复制')
  }
}
</script>

<style scoped>
.text-widget {
  width: 100%;
}

.response-text,
.table-cell-text,
.default-text {
  width: 100%;
}

.formatted-content {
  width: 100%;
  overflow: auto;
}

.code-content {
  margin: 0;
  padding: 12px;
  background-color: var(--el-fill-color-light);
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  font-size: 13px;
  line-height: 1.5;
  overflow-x: auto;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.html-content {
  padding: 8px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 4px;
  background-color: var(--el-bg-color);
}

/* 详情模式下的 HTML 表格样式 */
.html-detail {
  overflow-x: auto;
}

.html-detail :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 8px 0;
  font-size: 14px;
}

.html-detail :deep(table th),
.html-detail :deep(table td) {
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  text-align: left;
}

.html-detail :deep(table th) {
  background-color: var(--el-fill-color-light);
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.html-detail :deep(table tr:nth-child(even)) {
  background-color: var(--el-fill-color-extra-light);
}

.html-detail :deep(table tr:hover) {
  background-color: var(--el-fill-color);
}

/* 表格单元格模式下的 HTML 表格样式 */
.html-table-cell {
  padding: 4px;
  max-width: 100%;
  overflow-x: auto;
}

.html-table-cell :deep(table) {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  margin: 0;
}

.html-table-cell :deep(table th),
.html-table-cell :deep(table td) {
  padding: 4px 8px;
  border: 1px solid var(--el-border-color-lighter);
  text-align: left;
}

.html-table-cell :deep(table th) {
  background-color: var(--el-fill-color-light);
  font-weight: 500;
}

.html-table-cell :deep(table tr:nth-child(even)) {
  background-color: var(--el-fill-color-extra-light);
}

/* Markdown 内容样式 */
.markdown-content {
  padding: 8px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 4px;
  background-color: var(--el-bg-color);
  line-height: 1.6;
}

/* 详情模式下的 Markdown 表格样式 */
.markdown-detail {
  overflow-x: auto;
}

.markdown-detail :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 12px 0;
  font-size: 14px;
}

.markdown-detail :deep(table th),
.markdown-detail :deep(table td) {
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  text-align: left;
}

.markdown-detail :deep(table th) {
  background-color: var(--el-fill-color-light);
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.markdown-detail :deep(table tr:nth-child(even)) {
  background-color: var(--el-fill-color-extra-light);
}

.markdown-detail :deep(table tr:hover) {
  background-color: var(--el-fill-color);
}

/* CSV 内容样式 */
.csv-content {
  overflow-x: auto;
}

.csv-content table {
  width: 100%;
  border-collapse: collapse;
  margin: 0;
  font-size: 14px;
  background-color: var(--el-bg-color);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.csv-content table th,
.csv-content table td {
  padding: 12px 16px;
  border: 1px solid var(--el-border-color);
  text-align: left;
  line-height: 1.5;
}

.csv-content table th {
  background: linear-gradient(to bottom, var(--el-fill-color-light), var(--el-fill-color));
  font-weight: 600;
  color: var(--el-text-color-primary);
  position: sticky;
  top: 0;
  z-index: 1;
  border-bottom: 2px solid var(--el-border-color);
}

.csv-content table td {
  color: var(--el-text-color-regular);
  background-color: var(--el-bg-color);
}

.csv-content table tbody tr {
  transition: background-color 0.2s;
}

.csv-content table tbody tr:nth-child(even) {
  background-color: var(--el-fill-color-extra-light);
}

.csv-content table tbody tr:hover {
  background-color: var(--el-fill-color);
  cursor: default;
}

.csv-content table tbody tr:last-child td {
  border-bottom: 1px solid var(--el-border-color);
}

/* 详情模式下的 CSV 表格样式 */
.csv-detail {
  padding: 0;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  background-color: var(--el-bg-color);
  overflow: hidden;
}

.csv-detail table {
  border-radius: 6px;
  overflow: hidden;
}

.csv-detail table th:first-child,
.csv-detail table td:first-child {
  padding-left: 20px;
}

.csv-detail table th:last-child,
.csv-detail table td:last-child {
  padding-right: 20px;
}

/* 表格单元格模式下的 CSV 表格样式 */
.csv-table-cell {
  padding: 0;
  max-width: 100%;
}

.csv-preview {
  display: inline-block;
  padding: 4px 8px;
  background-color: var(--el-fill-color-light);
  border-radius: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.csv-preview-text {
  display: inline-block;
}

.markdown-content :deep(h1),
.markdown-content :deep(h2),
.markdown-content :deep(h3),
.markdown-content :deep(h4),
.markdown-content :deep(h5),
.markdown-content :deep(h6) {
  margin-top: 16px;
  margin-bottom: 8px;
  font-weight: 600;
}

.markdown-content :deep(p) {
  margin: 8px 0;
}

.markdown-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 12px 0;
}

.markdown-content :deep(table th),
.markdown-content :deep(table td) {
  padding: 8px 12px;
  border: 1px solid var(--el-border-color-lighter);
  text-align: left;
}

.markdown-content :deep(table th) {
  background-color: var(--el-fill-color-light);
  font-weight: 500;
}

.markdown-content :deep(table tr:nth-child(even)) {
  background-color: var(--el-fill-color-extra-light);
}

/* 表格单元格模式下的 Markdown 样式 */
.markdown-table-cell {
  padding: 4px;
  max-width: 100%;
  overflow-x: auto;
}

.markdown-table-cell :deep(table) {
  font-size: 12px;
  margin: 0;
}

.markdown-table-cell :deep(table th),
.markdown-table-cell :deep(table td) {
  padding: 4px 8px;
}

.text-content {
  padding: 8px;
  white-space: pre-wrap;
  word-wrap: break-word;
}

/* 超大文本截断样式 */
.text-content-truncated {
  position: relative;
}

.text-content-truncated .text-content,
.text-content-truncated .code-content {
  max-height: 300px;
  overflow: hidden;
  position: relative;
}

.text-actions {
  margin-top: 8px;
  padding: 8px 0;
  text-align: right;
}

/* 全文弹窗样式 */
.text-full-dialog :deep(.el-dialog__body) {
  padding: 20px;
}

.dialog-content {
  width: 100%;
}

.full-text-editor {
  width: 100%;
}

.full-text-editor :deep(.el-textarea__inner) {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  font-size: 13px;
  line-height: 1.5;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.empty-text {
  color: var(--el-text-color-placeholder);
}

.detail-text {
  margin-bottom: 16px;
}

.detail-label {
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
}

.detail-content {
  color: var(--el-text-color-regular);
}

.edit-text {
  width: 100%;
}

/* 不同格式的样式 */
.format-json .code-content {
  color: #2c3e50;
}

.format-yaml .code-content {
  color: #2c3e50;
}

.format-xml .code-content {
  color: #2c3e50;
}

.format-csv .code-content {
  color: #2c3e50;
}
</style>

