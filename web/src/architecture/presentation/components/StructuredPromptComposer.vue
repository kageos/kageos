<template>
  <div
    class="structured-prompt-composer"
    :class="{
      'is-disabled': disabled,
      'is-preview': mode === 'preview',
      'is-focused': focused,
      'is-compact': compact,
      'has-toolbar': showToolbar,
      [`mention-${mentionPanelPlacement}`]: true,
    }"
    data-testid="structured-prompt-composer"
  >
    <div v-if="showToolbar" class="spc-toolbar">
      <div class="spc-mode-tabs" role="tablist" aria-label="输入模式">
        <button
          type="button"
          :class="['spc-mode-btn', { 'is-active': mode === 'edit' }]"
          :aria-selected="mode === 'edit'"
          role="tab"
          @click="setMode('edit')"
        >
          <el-icon :size="14"><EditPen /></el-icon>
          编辑
        </button>
        <button
          type="button"
          :class="['spc-mode-btn', { 'is-active': mode === 'preview' }]"
          :aria-selected="mode === 'preview'"
          role="tab"
          @click="setMode('preview')"
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
      :class="{ 'is-empty': !currentText.trim() }"
      :contenteditable="disabled ? 'false' : 'true'"
      :data-placeholder="placeholder"
      :data-testid="editorTestId"
      :style="editorStyle"
      spellcheck="false"
      @input="handleEditorInput"
      @paste="handlePaste"
      @keydown="handleEditorKeydown"
      @keyup="handleEditorCursorChange"
      @click="handleEditorCursorChange"
      @mouseup="handleEditorCursorChange"
      @focus="handleFocus"
      @blur="handleBlur"
      @compositionstart="onCompositionStart"
      @compositionend="onCompositionEnd"
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

      <div v-if="currentText.trim()" class="spc-preview-body">
        <template v-for="segment in structuredSegments" :key="`${segment.start}-${segment.end}-${segment.type}`">
          <span v-if="segment.type === 'text'" class="spc-preview-text">{{ segment.text }}</span>
          <span
            v-else-if="segment.type === 'user'"
            class="spc-user-chip"
            :title="segment.text"
          >
            <span class="spc-user-mark">@</span>
            <span class="spc-user-name">{{ segment.username }}</span>
          </span>
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

    <div
      v-if="mentionPanelOpen"
      class="spc-mention-panel"
      data-testid="structured-prompt-mention-panel"
      @mousedown.prevent="cancelMentionClose"
    >
      <div class="spc-mention-header">
        <span class="spc-mention-trigger">{{ mentionQuery?.trigger }}</span>
        <span>{{ mentionModeLabel }}</span>
      </div>
      <div v-if="mentionLoading" class="spc-mention-state">搜索中...</div>
      <div v-else-if="mentionOptions.length === 0" class="spc-mention-state">
        {{ mentionEmptyText }}
      </div>
      <div v-else class="spc-mention-list">
        <button
          v-for="(option, index) in mentionOptions"
          :key="option.key"
          type="button"
          :class="['spc-mention-option', { 'is-active': index === highlightedMentionIndex }]"
          :data-testid="`structured-prompt-mention-option-${index}`"
          @mouseenter="highlightedMentionIndex = index"
          @mousedown.prevent="applyMentionOption(option)"
        >
          <span :class="['spc-mention-icon', `is-${option.kind}`, option.iconClass]">
            <UserAvatar
              v-if="option.kind === 'user'"
              :src="option.avatar"
              :size="28"
              :alt="option.label"
              class="spc-mention-avatar"
            />
            <img
              v-else-if="option.iconSrc"
              :src="option.iconSrc"
              :alt="option.typeLabel"
              class="spc-mention-resource-img"
            />
            <component
              :is="option.iconComponent"
              v-else-if="option.iconComponent"
              :size="17"
              class="spc-mention-resource-component"
            />
            <el-icon v-else><Document /></el-icon>
          </span>
          <span class="spc-mention-main">
            <span class="spc-mention-title-row">
              <span class="spc-mention-title" :title="option.label">{{ option.label }}</span>
              <span class="spc-mention-type">{{ option.typeLabel }}</span>
            </span>
            <span v-if="option.description" class="spc-mention-desc" :title="option.description">
              {{ option.description }}
            </span>
            <span v-if="option.metaItems.length > 0" class="spc-mention-meta-row">
              <span
                v-for="meta in option.metaItems"
                :key="`${option.key}-${meta}`"
                class="spc-mention-meta"
                :title="meta"
              >
                {{ meta }}
              </span>
            </span>
          </span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch, type Component } from 'vue'
import { Document, EditPen, View } from '@element-plus/icons-vue'
import type { UserInfo } from '@/architecture/domain/types'
import { formatUserDisplayName } from '@/architecture/domain/utils/userInfo'
import { searchUsersFuzzy } from '@/architecture/presentation/context/api/user'
import {
  searchResources,
  type ResourceSearchResult,
} from '@/architecture/presentation/context/api/service-tree'
import UserAvatar from '@/architecture/presentation/shared/components/UserAvatar.vue'
import ChartIcon from '@/architecture/presentation/shared/components/icons/ChartIcon.vue'
import TableIcon from '@/architecture/presentation/shared/components/icons/TableIcon.vue'
import {
  parseWorkspaceInvocationBlocks,
  parseWorkspacePromptSegments,
  wrapWorkspaceResourcePath,
  type WorkspacePromptSegment,
} from './utils/workspaceInvocationSnippet'
import {
  findMiniComposerMentionQuery,
  replaceMiniComposerMention,
  type MiniComposerMentionKind,
  type MiniComposerMentionQuery,
} from './utils/miniComposerMention'

type ComposerMode = 'edit' | 'preview'
type MentionPanelPlacement = 'above' | 'below'

interface StructuredPromptTextSegment {
  type: 'text'
  text: string
  start: number
  end: number
}

interface StructuredPromptUserSegment {
  type: 'user'
  text: string
  username: string
  start: number
  end: number
}

type StructuredPromptResourceSegment = WorkspacePromptSegment & { type: 'resource' }
type StructuredPromptSegment =
  | StructuredPromptTextSegment
  | StructuredPromptUserSegment
  | StructuredPromptResourceSegment

interface StructuredMentionOption {
  key: string
  kind: MiniComposerMentionKind
  label: string
  value: string
  description: string
  typeLabel: string
  resourceType?: ResourceSearchResult['type']
  avatar?: string
  iconSrc?: string
  iconComponent?: Component
  iconClass?: string
  metaItems: string[]
}

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  disabled?: boolean
  minRows?: number
  maxRows?: number
  showToolbar?: boolean
  compact?: boolean
  submitOnEnter?: boolean
  enableMentions?: boolean
  mentionPanelPlacement?: MentionPanelPlacement
  editorTestId?: string
}>(), {
  placeholder: '输入任务，可粘贴 </path> 资源引用或函数调用块',
  disabled: false,
  minRows: 4,
  maxRows: 12,
  showToolbar: true,
  compact: false,
  submitOnEnter: false,
  enableMentions: true,
  mentionPanelPlacement: 'below',
  editorTestId: 'structured-prompt-editor',
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'focus'): void
  (e: 'blur'): void
  (e: 'enter', event: KeyboardEvent): void
}>()

const editorRef = ref<HTMLElement | null>(null)
const mode = ref<ComposerMode>('edit')
const focused = ref(false)
const currentText = ref(props.modelValue)
const mentionQuery = ref<MiniComposerMentionQuery | null>(null)
const mentionOptions = ref<StructuredMentionOption[]>([])
const mentionLoading = ref(false)
const highlightedMentionIndex = ref(0)
const composing = ref(false)
let renderTimer: ReturnType<typeof setTimeout> | null = null
let mentionSearchTimer: ReturnType<typeof setTimeout> | null = null
let mentionCloseTimer: ReturnType<typeof setTimeout> | null = null
let mentionSearchSeq = 0
let activeMentionSearchKey = ''
let rendering = false

const structuredSegments = computed(() => parseStructuredPromptSegments(currentText.value))
const resourceSegments = computed(() => structuredSegments.value.filter((segment): segment is StructuredPromptResourceSegment => segment.type === 'resource'))
const invocationBlocks = computed(() => parseWorkspaceInvocationBlocks(currentText.value))
const mentionPanelOpen = computed(() => props.enableMentions && mode.value === 'edit' && !!mentionQuery.value)
const mentionModeLabel = computed(() => mentionQuery.value?.kind === 'user' ? '选择用户' : '选择目录或工具')
const mentionEmptyText = computed(() => {
  const query = mentionQuery.value
  if (!query) return ''
  if (!query.query.trim()) {
    return query.kind === 'user' ? '继续输入用户名或姓名' : '继续输入目录、函数或工具名称'
  }
  return query.kind === 'user' ? '没有匹配的用户' : '没有匹配的资源'
})
const editorStyle = computed(() => {
  const lineHeight = props.compact ? 20 : 24
  const verticalPadding = props.compact ? 8 : 28
  const minRows = Math.max(1, props.minRows)
  const maxRows = Math.max(minRows, props.maxRows)
  return {
    minHeight: `${minRows * lineHeight + verticalPadding}px`,
    maxHeight: `${maxRows * lineHeight + verticalPadding}px`,
  }
})

watch(() => props.modelValue, (value) => {
  if (value === currentText.value) return
  currentText.value = value
  if (rendering) return
  const editor = editorRef.value
  if (!editor) return
  if (serializeEditorContent(editor) === value) return
  renderEditorContent(value)
})

onMounted(() => {
  renderEditorContent(currentText.value)
})

onBeforeUnmount(() => {
  clearRenderTimer()
  resetMentionSearch()
  cancelMentionClose()
})

function setMode(nextMode: ComposerMode) {
  mode.value = nextMode
  closeMentionPanel()
  if (nextMode === 'edit') {
    void nextTick(() => focus())
  }
}

function handleEditorInput() {
  if (rendering || props.disabled) return
  const editor = editorRef.value
  if (!editor) return
  commitText(serializeEditorContent(editor))
  updateMentionFromEditor()
  scheduleTokenRender()
}

function handlePaste(event: ClipboardEvent) {
  const text = event.clipboardData?.getData('text/plain')
  if (!text) return
  event.preventDefault()
  insertTextAtCaret(text)
}

function handleEditorKeydown(event: KeyboardEvent) {
  if (props.disabled || isIMEComposing(event)) {
    return
  }

  if (mentionPanelOpen.value) {
    if (event.key === 'Escape') {
      event.preventDefault()
      closeMentionPanel()
      return
    }
    if (mentionOptions.value.length > 0) {
      if (event.key === 'ArrowDown') {
        event.preventDefault()
        moveMentionHighlight(1)
        return
      }
      if (event.key === 'ArrowUp') {
        event.preventDefault()
        moveMentionHighlight(-1)
        return
      }
      if (event.key === 'Enter' || event.key === 'Tab') {
        event.preventDefault()
        applyMentionOption(mentionOptions.value[highlightedMentionIndex.value] || mentionOptions.value[0])
        return
      }
    }
  }

  if (event.key === 'Enter') {
    if (props.submitOnEnter && !event.shiftKey) {
      event.preventDefault()
      emit('enter', event)
      return
    }
    event.preventDefault()
    insertTextAtCaret('\n')
  }
}

function handleEditorCursorChange(event?: Event) {
  if (composing.value) {
    return
  }
  if (
    event instanceof KeyboardEvent
    && (
      isIMEComposing(event)
      || (
        mentionPanelOpen.value
        && ['ArrowDown', 'ArrowUp', 'Enter', 'Tab', 'Escape'].includes(event.key)
      )
    )
  ) {
    return
  }
  updateMentionFromEditor()
}

function handleFocus() {
  focused.value = true
  cancelMentionClose()
  emit('focus')
  void nextTick(() => updateMentionFromEditor())
}

function handleBlur() {
  focused.value = false
  scheduleMentionClose()
  renderEditorContent(currentText.value)
  emit('blur')
}

function onCompositionStart() {
  composing.value = true
}

function onCompositionEnd() {
  composing.value = false
  handleEditorInput()
}

function isIMEComposing(event: KeyboardEvent) {
  return composing.value
    || event.isComposing
    || event.key === 'Process'
    || (event as KeyboardEvent & { keyCode?: number }).keyCode === 229
}

function commitText(value: string) {
  currentText.value = value
  emit('update:modelValue', value)
}

function scheduleTokenRender() {
  clearRenderTimer()
  renderTimer = setTimeout(() => {
    const editor = editorRef.value
    if (!editor || focused.value === false) return
    if (mentionQuery.value) return
    const offset = getCaretTextOffset(editor)
    const text = serializeEditorContent(editor)
    currentText.value = text
    renderEditorContent(text)
    void nextTick(() => {
      restoreCaretTextOffset(editor, offset)
      updateMentionFromEditor()
    })
  }, 260)
}

function clearRenderTimer() {
  if (renderTimer) {
    clearTimeout(renderTimer)
    renderTimer = null
  }
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
  parseStructuredPromptSegments(text).forEach((segment) => {
    if (segment.type === 'resource') {
      nodes.push(createResourceTokenNode(segment))
    } else if (segment.type === 'user') {
      nodes.push(createUserTokenNode(segment))
    } else if (segment.text) {
      nodes.push(document.createTextNode(segment.text))
    }
  })
  if (nodes.length === 0) {
    nodes.push(document.createTextNode(''))
  }
  return nodes
}

function createResourceTokenNode(segment: StructuredPromptResourceSegment) {
  const chip = document.createElement('span')
  chip.className = `spc-editor-token is-resource is-${resourceKind(segment.path || '')}`
  chip.contentEditable = 'false'
  chip.dataset.tokenRaw = segment.text
  chip.dataset.path = segment.path || ''
  chip.textContent = resourceDisplayName(segment.path || segment.text)
  return chip
}

function createUserTokenNode(segment: StructuredPromptUserSegment) {
  const chip = document.createElement('span')
  chip.className = 'spc-editor-token is-user'
  chip.contentEditable = 'false'
  chip.dataset.tokenRaw = segment.text
  chip.dataset.username = segment.username
  chip.textContent = `@${segment.username}`
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

function insertTextAtCaret(text: string) {
  if (props.disabled) return
  mode.value = 'edit'
  const editor = editorRef.value
  if (!editor) return
  editor.focus()

  const selection = window.getSelection()
  const range = selection?.rangeCount ? selection.getRangeAt(0) : null
  if (!selection || !range || !isRangeInside(editor, range)) {
    editor.appendChild(document.createTextNode(text))
    commitText(serializeEditorContent(editor))
    restoreCaretTextOffset(editor, currentText.value.length)
    updateMentionFromEditor()
    scheduleTokenRender()
    return
  }

  range.deleteContents()
  const node = document.createTextNode(text)
  range.insertNode(node)
  range.setStart(node, text.length)
  range.collapse(true)
  selection.removeAllRanges()
  selection.addRange(range)
  handleEditorInput()
}

function isRangeInside(root: HTMLElement, range: Range) {
  const container = range.commonAncestorContainer
  return container === root || root.contains(container)
}

function getCaretTextOffset(root: HTMLElement): number {
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) return serializeEditorContent(root).length
  const range = selection.getRangeAt(0)
  if (!isRangeInside(root, range)) return serializeEditorContent(root).length
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

function updateMentionFromEditor() {
  cancelMentionClose()
  if (!props.enableMentions || props.disabled || mode.value !== 'edit') {
    closeMentionPanel()
    return
  }

  const editor = editorRef.value
  if (!editor) {
    closeMentionPanel()
    return
  }

  const text = serializeEditorContent(editor)
  const query = findMiniComposerMentionQuery(text, getCaretTextOffset(editor))
  mentionQuery.value = query
  highlightedMentionIndex.value = 0

  if (!query) {
    resetMentionSearch()
    return
  }

  const keyword = query.query.trim()
  if (!keyword) {
    resetMentionSearch()
    return
  }

  scheduleMentionSearch(query)
}

function resetMentionSearch() {
  if (mentionSearchTimer) {
    clearTimeout(mentionSearchTimer)
    mentionSearchTimer = null
  }
  mentionSearchSeq += 1
  activeMentionSearchKey = ''
  mentionLoading.value = false
  mentionOptions.value = []
}

function getMentionSearchKey(query: MiniComposerMentionQuery) {
  return `${query.kind}:${query.query.trim()}`
}

function scheduleMentionSearch(query: MiniComposerMentionQuery) {
  const searchKey = getMentionSearchKey(query)
  if (searchKey === activeMentionSearchKey) {
    return
  }

  activeMentionSearchKey = searchKey
  if (mentionSearchTimer) {
    clearTimeout(mentionSearchTimer)
  }
  mentionLoading.value = true
  mentionSearchTimer = setTimeout(() => {
    void runMentionSearch(query, searchKey)
  }, 220)
}

async function runMentionSearch(query: MiniComposerMentionQuery, searchKey: string) {
  const currentSeq = ++mentionSearchSeq
  mentionLoading.value = true

  try {
    if (query.kind === 'user') {
      const response = await searchUsersFuzzy(query.query.trim(), 8)
      if (currentSeq !== mentionSearchSeq || activeMentionSearchKey !== searchKey) return
      mentionOptions.value = (response.users || []).map(mapUserMentionOption)
    } else {
      const response = await searchResources({
        keyword: query.query.trim(),
        resource_type: 'all',
        page: 1,
        page_size: 8,
      })
      if (currentSeq !== mentionSearchSeq || activeMentionSearchKey !== searchKey) return
      mentionOptions.value = (response.items || []).map(mapResourceMentionOption)
    }
  } catch {
    if (currentSeq === mentionSearchSeq && activeMentionSearchKey === searchKey) {
      mentionOptions.value = []
    }
  } finally {
    if (currentSeq === mentionSearchSeq && activeMentionSearchKey === searchKey) {
      mentionLoading.value = false
      highlightedMentionIndex.value = 0
    }
  }
}

function mapUserMentionOption(user: UserInfo): StructuredMentionOption {
  const username = user.username || ''
  const department = user.department_full_name_path || user.department_name || ''
  const company = user.company_name || user.company_code || ''
  const signature = cleanMentionText(user.signature)
  const email = cleanMentionText(user.email)

  return {
    key: `user:${username}`,
    kind: 'user',
    label: formatUserDisplayName(user),
    value: username,
    description: email || signature || username,
    typeLabel: '用户',
    avatar: user.avatar,
    metaItems: compactMetaItems([
      username ? `@${username}` : '',
      department,
      company,
    ]),
  }
}

function mapResourceMentionOption(resource: ResourceSearchResult): StructuredMentionOption {
  const path = resource.full_code_path || ''
  const description = cleanMentionText(resource.description)
    || cleanMentionText(resource.snippet)
    || getReadablePath(path)

  return {
    key: `resource:${resource.type}:${resource.id}:${path}`,
    kind: 'resource',
    label: resource.name || resource.code || getPathTail(path) || path,
    value: path,
    description,
    typeLabel: getResourceTypeLabel(resource),
    resourceType: resource.type,
    metaItems: compactMetaItems([
      getReadablePath(path),
      resource.match_source ? `命中 ${getMatchSourceLabel(resource.match_source)}` : '',
      shouldShowResourceHeat(resource) ? `${formatCompactCount(resource.run_count || 0)} 次运行` : '',
    ]),
    ...getResourceIconMeta(resource),
  }
}

function moveMentionHighlight(delta: number) {
  const count = mentionOptions.value.length
  if (count === 0) return
  highlightedMentionIndex.value = (highlightedMentionIndex.value + delta + count) % count
}

function applyMentionOption(option: StructuredMentionOption | undefined) {
  if (!option || !mentionQuery.value) {
    return
  }

  const replacement = option.kind === 'user'
    ? `@${option.value}`
    : wrapWorkspaceResourcePath(option.value)
  const result = replaceMiniComposerMention(currentText.value, mentionQuery.value, replacement)

  commitText(result.value)
  renderEditorContent(result.value)
  closeMentionPanel()

  void nextTick(() => {
    const editor = editorRef.value
    if (!editor) return
    editor.focus()
    restoreCaretTextOffset(editor, result.cursor)
  })
}

function closeMentionPanel() {
  mentionQuery.value = null
  resetMentionSearch()
}

function scheduleMentionClose() {
  cancelMentionClose()
  mentionCloseTimer = setTimeout(() => {
    closeMentionPanel()
  }, 160)
}

function cancelMentionClose() {
  if (mentionCloseTimer) {
    clearTimeout(mentionCloseTimer)
    mentionCloseTimer = null
  }
}

function parseStructuredPromptSegments(text: string): StructuredPromptSegment[] {
  const source = String(text || '')
  const tokens: Array<StructuredPromptResourceSegment | StructuredPromptUserSegment> = []

  parseWorkspacePromptSegments(source).forEach((segment) => {
    if (segment.type === 'resource') {
      tokens.push(segment as StructuredPromptResourceSegment)
    }
  })

  const userMatcher = /(^|\s)(@[^\s<>]+)/g
  let userMatch: RegExpExecArray | null
  while ((userMatch = userMatcher.exec(source)) !== null) {
    const prefix = userMatch[1] || ''
    const raw = userMatch[2] || ''
    const start = userMatch.index + prefix.length
    const end = start + raw.length
    const nextChar = source[end] || ''
    if (raw.length <= 1 || (nextChar && !/\s/.test(nextChar))) {
      continue
    }
    tokens.push({
      type: 'user',
      text: raw,
      username: raw.slice(1),
      start,
      end,
    })
  }

  const orderedTokens = tokens
    .sort((a, b) => a.start - b.start || a.end - b.end)
    .filter((token, index, list) => {
      const previous = list[index - 1]
      return !previous || token.start >= previous.end
    })

  const segments: StructuredPromptSegment[] = []
  let cursor = 0
  orderedTokens.forEach((token) => {
    if (token.start > cursor) {
      segments.push({
        type: 'text',
        text: source.slice(cursor, token.start),
        start: cursor,
        end: token.start,
      })
    }
    segments.push(token)
    cursor = token.end
  })

  if (cursor < source.length) {
    segments.push({
      type: 'text',
      text: source.slice(cursor),
      start: cursor,
      end: source.length,
    })
  }

  if (segments.length === 0) {
    segments.push({
      type: 'text',
      text: source,
      start: 0,
      end: source.length,
    })
  }

  return segments
}

function getPathTail(path: string) {
  const parts = path.split('/').filter(Boolean)
  return parts[parts.length - 1] || ''
}

function getReadablePath(path: string) {
  const parts = path.split('/').filter(Boolean)
  if (parts.length <= 3) return path
  return `/${parts.slice(-3).join('/')}`
}

function cleanMentionText(value?: string) {
  return String(value || '')
    .replace(/<[^>]*>/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}

function compactMetaItems(items: Array<string | undefined | null>) {
  return items
    .map(item => cleanMentionText(item || ''))
    .filter((item, index, list) => item && list.indexOf(item) === index)
}

function getResourceTypeLabel(resource: ResourceSearchResult) {
  if (resource.type === 'package') return '目录'
  if (resource.type === 'docs') return '文档'
  if (resource.template_type === 'table') return '表格工具'
  if (resource.template_type === 'form') return '表单工具'
  if (resource.template_type === 'chart') return '图表工具'
  return '工具'
}

function getMatchSourceLabel(matchSource: string) {
  const normalized = matchSource.toLowerCase()
  if (normalized.includes('description')) return '描述'
  if (normalized.includes('tag')) return '标签'
  if (normalized.includes('code')) return '编码'
  if (normalized.includes('path')) return '路径'
  if (normalized.includes('name')) return '名称'
  return matchSource
}

function shouldShowResourceHeat(resource: ResourceSearchResult) {
  return resource.type === 'function' && !!resource.run_count
}

function formatCompactCount(count: number) {
  if (count >= 10000) return `${(count / 10000).toFixed(count >= 100000 ? 0 : 1)}w`
  if (count >= 1000) return `${(count / 1000).toFixed(count >= 10000 ? 0 : 1)}k`
  return String(count)
}

function getResourceIconMeta(resource: ResourceSearchResult): Pick<StructuredMentionOption, 'iconSrc' | 'iconComponent' | 'iconClass'> {
  if (resource.type === 'package') {
    return { iconSrc: '/service-tree/custom-folder.svg', iconClass: 'package-icon-img' }
  }
  if (resource.type === 'docs') {
    return { iconSrc: '/文档.svg', iconClass: 'docs-icon-img' }
  }
  if (resource.template_type === 'form') {
    return { iconSrc: '/service-tree/编辑.svg', iconClass: 'form-icon-img' }
  }
  if (resource.template_type === 'table') {
    return { iconComponent: TableIcon, iconClass: 'table-icon' }
  }
  if (resource.template_type === 'chart') {
    return { iconComponent: ChartIcon, iconClass: 'chart-icon' }
  }
  return { iconClass: 'function-icon' }
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

function focus() {
  mode.value = 'edit'
  void nextTick(() => {
    const editor = editorRef.value
    if (!editor) return
    editor.focus()
    const selection = window.getSelection()
    const range = selection?.rangeCount ? selection.getRangeAt(0) : null
    if (document.activeElement === editor && (!range || !isRangeInside(editor, range))) {
      restoreCaretTextOffset(editor, currentText.value.length)
    }
  })
}

function blur() {
  editorRef.value?.blur()
}

defineExpose({
  focus,
  blur,
  insertTextAtCaret,
  getElement: () => editorRef.value,
})
</script>

<style scoped>
.structured-prompt-composer {
  --spc-bg: rgba(9, 15, 28, 0.9);
  --spc-border: rgba(125, 146, 183, 0.28);
  --spc-text: #e6f0ff;
  --spc-muted: #8da0bd;
  --spc-accent: #60e7ff;
  position: relative;
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

.is-compact .spc-editor,
.is-compact .spc-preview {
  padding: 4px 0;
  font-size: 14px;
  line-height: 20px;
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
.spc-resource-chip,
.spc-user-chip {
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

:deep(.spc-editor-token.is-user),
.spc-user-chip {
  border-color: rgba(124, 255, 196, 0.2);
  background: rgba(124, 255, 196, 0.08);
  color: #b9ffd9;
}

:deep(.spc-editor-token.is-table) {
  border-color: rgba(16, 185, 129, 0.2);
}

.spc-user-mark,
.spc-resource-kind {
  color: rgba(216, 248, 255, 0.62);
  font-size: 11px;
}

.spc-user-name,
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

.spc-mention-panel {
  position: absolute;
  left: 0;
  right: 0;
  z-index: 24;
  max-height: 338px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  border: 1px solid rgba(96, 231, 255, 0.24);
  border-radius: 10px;
  background:
    linear-gradient(145deg, rgba(5, 17, 31, 0.98), rgba(9, 30, 48, 0.96)),
    rgba(4, 12, 24, 0.98);
  box-shadow: 0 18px 44px rgba(0, 0, 0, 0.34), 0 0 24px rgba(34, 211, 238, 0.12);
}

.mention-below .spc-mention-panel {
  top: calc(100% + 6px);
}

.mention-above .spc-mention-panel {
  bottom: calc(100% + 6px);
}

.spc-mention-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 10px;
  border-bottom: 1px solid rgba(96, 231, 255, 0.14);
  color: rgba(216, 248, 255, 0.78);
  font-size: 12px;
  font-weight: 700;
}

.spc-mention-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 6px;
  background: rgba(34, 211, 238, 0.14);
  color: #22d3ee;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.spc-mention-state {
  min-height: 72px;
  display: flex;
  align-items: center;
  padding: 16px 14px;
  color: rgba(184, 225, 235, 0.68);
  font-size: 12px;
}

.spc-mention-list {
  overflow-y: auto;
  padding: 7px;
}

.spc-mention-list::-webkit-scrollbar {
  width: 7px;
}

.spc-mention-list::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(96, 231, 255, 0.2);
}

.spc-mention-option {
  width: 100%;
  min-width: 0;
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr);
  gap: 11px;
  align-items: start;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  padding: 10px;
  text-align: left;
  transition: border-color 0.18s ease, background 0.18s ease, transform 0.18s ease;
}

.spc-mention-option:hover,
.spc-mention-option.is-active {
  border-color: rgba(34, 211, 238, 0.32);
  background:
    linear-gradient(135deg, rgba(34, 211, 238, 0.13), rgba(124, 255, 196, 0.06)),
    rgba(34, 211, 238, 0.08);
  transform: translateY(-1px);
}

.spc-mention-icon {
  width: 38px;
  height: 38px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(96, 231, 255, 0.18);
  background: rgba(34, 211, 238, 0.08);
  color: #22d3ee;
}

.spc-mention-icon.is-user {
  overflow: hidden;
  color: #7cffc4;
  background: rgba(124, 255, 196, 0.08);
  border-color: rgba(124, 255, 196, 0.2);
}

.spc-mention-icon.table-icon {
  color: #10b981;
}

.spc-mention-icon.form-icon-img {
  color: #3b82f6;
}

.spc-mention-icon.chart-icon {
  color: #9377e0;
}

.spc-mention-icon.function-icon {
  color: #6366f1;
}

.spc-mention-icon.package-icon-img,
.spc-mention-icon.docs-icon-img,
.spc-mention-icon.form-icon-img {
  background: rgba(255, 255, 255, 0.04);
}

.spc-mention-avatar {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  font-size: 12px;
  font-weight: 700;
}

.spc-mention-resource-img {
  width: 22px;
  height: 22px;
  object-fit: contain;
  flex-shrink: 0;
  opacity: 0.94;
}

.spc-mention-resource-component {
  width: 22px;
  height: 22px;
  flex-shrink: 0;
}

.spc-mention-main,
.spc-mention-title-row,
.spc-mention-title,
.spc-mention-desc,
.spc-mention-meta-row,
.spc-mention-meta {
  min-width: 0;
}

.spc-mention-main {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.spc-mention-title-row {
  display: flex;
  align-items: center;
  gap: 6px;
}

.spc-mention-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--spc-text);
  font-size: 13px;
  font-weight: 700;
}

.spc-mention-type {
  flex-shrink: 0;
  border: 1px solid rgba(96, 231, 255, 0.18);
  border-radius: 999px;
  padding: 2px 6px;
  color: rgba(96, 231, 255, 0.86);
  font-size: 11px;
  line-height: 1.2;
}

.spc-mention-desc {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: rgba(184, 225, 235, 0.62);
  font-size: 12px;
}

.spc-mention-meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  overflow: hidden;
}

.spc-mention-meta {
  max-width: 160px;
  display: inline-flex;
  align-items: center;
  overflow: hidden;
  border: 1px solid rgba(96, 231, 255, 0.13);
  border-radius: 7px;
  padding: 2px 6px;
  background: rgba(255, 255, 255, 0.035);
  color: rgba(184, 225, 235, 0.7);
  font-size: 11px;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
