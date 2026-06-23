<template>
  <div
    class="structured-prompt-composer"
    :class="{
      'is-disabled': disabled,
      'is-preview': mode === 'preview',
      'is-focused': focused,
    }"
    data-testid="structured-prompt-composer"
  >
    <div class="spc-toolbar">
      <div class="spc-mode-tabs" role="tablist" aria-label="输入模式">
        <button
          type="button"
          :class="['spc-mode-btn', { 'is-active': mode === 'edit' }]"
          :aria-selected="mode === 'edit'"
          role="tab"
          @click="mode = 'edit'"
        >
          <el-icon :size="14"><EditPen /></el-icon>
          编辑
        </button>
        <button
          type="button"
          :class="['spc-mode-btn', { 'is-active': mode === 'preview' }]"
          :aria-selected="mode === 'preview'"
          role="tab"
          @click="mode = 'preview'"
        >
          <el-icon :size="14"><View /></el-icon>
          预览
        </button>
      </div>

      <div v-if="resourceSegments.length > 0" class="spc-token-summary" aria-label="已识别资源">
        <span
          v-for="segment in resourceSegments.slice(0, 3)"
          :key="`${segment.start}-${segment.text}`"
          class="spc-token-pill"
          :title="segment.path"
        >
          {{ resourceDisplayName(segment.path || segment.text) }}
        </span>
        <span v-if="resourceSegments.length > 3" class="spc-token-count">
          +{{ resourceSegments.length - 3 }}
        </span>
      </div>
    </div>

    <div
      v-show="mode === 'edit'"
      ref="editorRef"
      class="spc-editor"
      :class="{ 'is-empty': !modelValue.trim() }"
      :contenteditable="disabled ? 'false' : 'true'"
      :data-placeholder="placeholder"
      :style="editorStyle"
      spellcheck="false"
      data-testid="structured-prompt-editor"
      @input="handleEditorInput"
      @paste="handlePaste"
      @focus="handleFocus"
      @blur="handleBlur"
    />

    <div v-show="mode === 'preview'" class="spc-preview" :style="editorStyle" data-testid="structured-prompt-preview">
      <div v-if="invocationBlocks.length > 0" class="spc-invocation-list">
        <section
          v-for="(block, index) in invocationBlocks"
          :key="`${block.tool}-${block.resourcePath}-${index}`"
          class="spc-invocation-card"
        >
          <div class="spc-invocation-head">
            <span class="spc-invocation-tool">{{ block.tool || '函数调用' }}</span>
            <span class="spc-invocation-resource" :title="block.resourcePath">
              {{ resourceDisplayName(block.resourcePath) }}
            </span>
          </div>
          <div v-if="block.params.length > 0" class="spc-param-row">
            <span
              v-for="param in block.params.slice(0, 4)"
              :key="`${block.tool}-${param.key}`"
              class="spc-param-chip"
              :title="`${param.key} = ${param.value}`"
            >
              {{ param.key }}{{ param.fixed ? ' 固定' : '' }}
            </span>
          </div>
        </section>
      </div>

      <div v-if="modelValue.trim()" class="spc-preview-body">
        <template v-for="segment in promptSegments" :key="`${segment.start}-${segment.end}`">
          <span v-if="segment.type === 'text'" class="spc-preview-text">{{ segment.text }}</span>
          <span
            v-else
            class="spc-resource-chip"
            :title="segment.path"
            :data-path="segment.path"
          >
            <span class="spc-resource-kind">{{ resourceKindLabel(segment.path || '') }}</span>
            <span class="spc-resource-name">{{ resourceDisplayName(segment.path || segment.text) }}</span>
          </span>
        </template>
      </div>
      <div v-else class="spc-preview-placeholder">{{ placeholder }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { EditPen, View } from '@element-plus/icons-vue'
import {
  parseWorkspaceInvocationBlocks,
  parseWorkspacePromptSegments,
  type WorkspacePromptSegment,
} from './utils/workspaceInvocationSnippet'

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  disabled?: boolean
  minRows?: number
  maxRows?: number
}>(), {
  placeholder: '输入任务，可粘贴 </path> 资源引用或函数调用块',
  disabled: false,
  minRows: 4,
  maxRows: 12,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'focus'): void
  (e: 'blur'): void
}>()

const editorRef = ref<HTMLElement | null>(null)
const mode = ref<'edit' | 'preview'>('edit')
const focused = ref(false)
let renderTimer: ReturnType<typeof setTimeout> | null = null
let rendering = false

const promptSegments = computed(() => parseWorkspacePromptSegments(props.modelValue))
const resourceSegments = computed(() => promptSegments.value.filter((segment) => segment.type === 'resource'))
const invocationBlocks = computed(() => parseWorkspaceInvocationBlocks(props.modelValue))
const editorStyle = computed(() => ({
  minHeight: `${Math.max(2, props.minRows) * 24 + 28}px`,
  maxHeight: `${Math.max(props.minRows, props.maxRows) * 24 + 28}px`,
}))

watch(() => props.modelValue, (value) => {
  if (rendering) return
  const editor = editorRef.value
  if (!editor) return
  if (serializeEditorContent(editor) === value) return
  renderEditorContent(value)
})

onMounted(() => {
  renderEditorContent(props.modelValue)
})

onBeforeUnmount(() => {
  if (renderTimer) {
    clearTimeout(renderTimer)
  }
})

function handleEditorInput() {
  if (rendering || props.disabled) return
  const editor = editorRef.value
  if (!editor) return
  emit('update:modelValue', serializeEditorContent(editor))
  scheduleTokenRender()
}

function handlePaste(event: ClipboardEvent) {
  const text = event.clipboardData?.getData('text/plain')
  if (!text) return
  event.preventDefault()
  document.execCommand('insertText', false, text)
  handleEditorInput()
}

function handleFocus() {
  focused.value = true
  emit('focus')
}

function handleBlur() {
  focused.value = false
  renderEditorContent(props.modelValue)
  emit('blur')
}

function scheduleTokenRender() {
  if (renderTimer) {
    clearTimeout(renderTimer)
  }
  renderTimer = setTimeout(() => {
    const editor = editorRef.value
    if (!editor || focused.value === false) return
    const offset = getCaretTextOffset(editor)
    const text = serializeEditorContent(editor)
    renderEditorContent(text)
    void nextTick(() => restoreCaretTextOffset(editor, offset))
  }, 260)
}

function renderEditorContent(text: string) {
  const editor = editorRef.value
  if (!editor) return

  rendering = true
  editor.replaceChildren(...buildEditorNodes(text))
  rendering = false
}

function buildEditorNodes(text: string): Node[] {
  const nodes: Node[] = []
  parseWorkspacePromptSegments(text).forEach((segment) => {
    if (segment.type === 'resource') {
      nodes.push(createResourceTokenNode(segment))
    } else if (segment.text) {
      nodes.push(document.createTextNode(segment.text))
    }
  })
  if (nodes.length === 0) {
    nodes.push(document.createTextNode(''))
  }
  return nodes
}

function createResourceTokenNode(segment: WorkspacePromptSegment) {
  const chip = document.createElement('span')
  chip.className = `spc-editor-token is-${resourceKind(segment.path || '')}`
  chip.contentEditable = 'false'
  chip.dataset.tokenRaw = segment.text
  chip.dataset.path = segment.path || ''
  chip.textContent = resourceDisplayName(segment.path || segment.text)
  return chip
}

function serializeEditorContent(root: HTMLElement): string {
  let text = ''
  root.childNodes.forEach((node) => {
    text += serializeNode(node)
  })
  return text
}

function serializeNode(node: Node): string {
  if (node.nodeType === Node.TEXT_NODE) {
    return node.textContent || ''
  }
  if (node instanceof HTMLElement) {
    if (node.dataset.tokenRaw) {
      return node.dataset.tokenRaw
    }
    let text = ''
    node.childNodes.forEach((child) => {
      text += serializeNode(child)
    })
    return text
  }
  return node.textContent || ''
}

function getCaretTextOffset(root: HTMLElement): number {
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) return serializeEditorContent(root).length
  const range = selection.getRangeAt(0)
  const preRange = range.cloneRange()
  preRange.selectNodeContents(root)
  preRange.setEnd(range.endContainer, range.endOffset)
  const fragment = preRange.cloneContents()
  const container = document.createElement('div')
  container.appendChild(fragment)
  return serializeEditorContent(container).length
}

function restoreCaretTextOffset(root: HTMLElement, targetOffset: number) {
  const range = document.createRange()
  const selection = window.getSelection()
  let consumed = 0
  let placed = false

  const place = (node: Node, offset: number) => {
    range.setStart(node, offset)
    range.collapse(true)
    placed = true
  }

  root.childNodes.forEach((node) => {
    if (placed) return
    const tokenRaw = node instanceof HTMLElement ? node.dataset.tokenRaw : ''
    const length = tokenRaw ? tokenRaw.length : (node.textContent || '').length
    if (targetOffset <= consumed + length) {
      if (node.nodeType === Node.TEXT_NODE) {
        place(node, Math.max(0, targetOffset - consumed))
      } else if (targetOffset - consumed <= Math.floor(length / 2)) {
        range.setStartBefore(node)
        range.collapse(true)
        placed = true
      } else {
        range.setStartAfter(node)
        range.collapse(true)
        placed = true
      }
    }
    consumed += length
  })

  if (!placed) {
    range.selectNodeContents(root)
    range.collapse(false)
  }

  selection?.removeAllRanges()
  selection?.addRange(range)
}

function resourceKind(path: string) {
  if (path.endsWith('.form')) return 'form'
  if (path.endsWith('.table')) return 'table'
  if (path.endsWith('.chart')) return 'chart'
  if (path.endsWith('.docs') || path.includes('/docs/')) return 'docs'
  return 'directory'
}

function resourceKindLabel(path: string) {
  const kind = resourceKind(path)
  if (kind === 'form') return '表单'
  if (kind === 'table') return '表格'
  if (kind === 'chart') return '图表'
  if (kind === 'docs') return '文档'
  return '目录'
}

function resourceDisplayName(path: string) {
  const normalized = String(path || '').replace(/[<>]/g, '')
  const parts = normalized.split('/').filter(Boolean)
  return parts[parts.length - 1] || normalized || '资源'
}
</script>

<style scoped>
.structured-prompt-composer {
  --spc-bg: rgba(9, 15, 28, 0.9);
  --spc-border: rgba(125, 146, 183, 0.28);
  --spc-text: #e6f0ff;
  --spc-muted: #8da0bd;
  --spc-accent: #60e7ff;
  border: 1px solid var(--spc-border);
  border-radius: 10px;
  background: linear-gradient(180deg, rgba(13, 20, 35, 0.92), rgba(8, 13, 24, 0.88));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.05);
}

.structured-prompt-composer.is-focused {
  border-color: rgba(96, 231, 255, 0.42);
  box-shadow: 0 0 0 3px rgba(96, 231, 255, 0.08), inset 0 1px 0 rgba(255, 255, 255, 0.06);
}

.structured-prompt-composer.is-disabled {
  opacity: 0.68;
}

.spc-toolbar {
  min-height: 42px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 8px 6px 10px;
  border-bottom: 1px solid rgba(125, 146, 183, 0.16);
}

.spc-mode-tabs {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px;
  border: 1px solid rgba(125, 146, 183, 0.18);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.04);
}

.spc-mode-btn {
  height: 28px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: 0;
  border-radius: 6px;
  padding: 0 9px;
  background: transparent;
  color: var(--spc-muted);
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
}

.spc-mode-btn.is-active {
  background: rgba(96, 231, 255, 0.14);
  color: var(--spc-accent);
}

.spc-token-summary {
  min-width: 0;
  display: flex;
  justify-content: flex-end;
  gap: 5px;
  overflow: hidden;
}

.spc-token-pill,
.spc-token-count {
  max-width: 150px;
  display: inline-flex;
  align-items: center;
  overflow: hidden;
  border: 1px solid rgba(96, 231, 255, 0.18);
  border-radius: 7px;
  padding: 3px 7px;
  color: rgba(216, 248, 255, 0.86);
  background: rgba(96, 231, 255, 0.08);
  font-size: 11px;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.spc-editor,
.spc-preview {
  box-sizing: border-box;
  overflow-y: auto;
  padding: 12px;
  color: var(--spc-text);
  font-size: 14px;
  line-height: 24px;
}

.spc-editor {
  outline: none;
  white-space: pre-wrap;
  word-break: break-word;
}

.spc-editor.is-empty::before {
  content: attr(data-placeholder);
  color: rgba(141, 160, 189, 0.72);
  pointer-events: none;
}

.spc-preview-body {
  white-space: pre-wrap;
  word-break: break-word;
}

.spc-preview-placeholder {
  color: rgba(141, 160, 189, 0.72);
}

:deep(.spc-editor-token),
.spc-resource-chip {
  max-width: min(420px, 100%);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0 2px;
  border: 1px solid rgba(96, 231, 255, 0.22);
  border-radius: 7px;
  padding: 1px 7px;
  background: rgba(96, 231, 255, 0.1);
  color: #baf7ff;
  font-size: 12px;
  font-weight: 700;
  line-height: 20px;
  vertical-align: baseline;
  white-space: nowrap;
}

:deep(.spc-editor-token.is-table) {
  border-color: rgba(16, 185, 129, 0.2);
}

.spc-resource-kind {
  color: rgba(216, 248, 255, 0.62);
  font-size: 11px;
}

.spc-resource-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.spc-invocation-list {
  display: grid;
  gap: 8px;
  margin-bottom: 10px;
}

.spc-invocation-card {
  border: 1px solid rgba(96, 231, 255, 0.18);
  border-radius: 8px;
  padding: 9px;
  background: rgba(96, 231, 255, 0.06);
}

.spc-invocation-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.spc-invocation-tool {
  flex-shrink: 0;
  border-radius: 6px;
  padding: 2px 6px;
  background: rgba(139, 92, 246, 0.16);
  color: #c8b6ff;
  font-size: 12px;
  font-weight: 800;
}

.spc-invocation-resource {
  min-width: 0;
  overflow: hidden;
  color: #d8f8ff;
  font-size: 13px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.spc-param-row {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 8px;
}

.spc-param-chip {
  border-radius: 6px;
  padding: 2px 6px;
  background: rgba(255, 255, 255, 0.06);
  color: rgba(230, 240, 255, 0.78);
  font-size: 11px;
}
</style>
