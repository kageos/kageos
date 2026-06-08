<template>
  <div v-if="attachedFiles.length > 0" class="mini-ws-files">
    <el-tag
      v-for="(file, idx) in attachedFiles"
      :key="`${file.ref || file.name}-${idx}`"
      size="small"
      closable
      @close="removeFile(idx)"
    >
      {{ file.source_name || file.name }}
    </el-tag>
  </div>

  <div class="mini-ws-input" data-testid="mini-workstation-composer">
    <div class="mini-composer-left-actions">
      <slot name="left-actions" />
	      <el-upload
	        :auto-upload="false"
	        :show-file-list="false"
	        :on-change="onFileChange"
	        :disabled="uploading || blocked"
	        class="mini-upload-btn"
	      >
        <el-button :icon="Paperclip" link :loading="uploading" size="small" title="上传文件" />
      </el-upload>
    </div>
	    <div class="mini-input-wrap">
	      <span class="mini-path-pill" :title="fullCodePath">{{ displayPath }}</span>
	      <span v-if="blocked" class="mini-blocked-pill" :title="blockedLabel || '等待确认'">{{ blockedLabel || '等待确认' }}</span>
	      <textarea
	        :ref="bindInputRef"
	        :value="inputText"
	        class="mini-input"
	        data-testid="mini-workstation-input"
	        :placeholder="blocked ? (blockedPlaceholder || '当前会话需要先处理交互卡片') : '输入命令...（@搜用户，/搜目录/工具/文档，Enter 发送，Shift+Enter 换行）'"
	        :disabled="blocked"
	        rows="1"
        @input="emitInput"
        @keydown="onTextareaKeydown"
        @keyup="onTextareaCursorChange"
        @click="onTextareaCursorChange"
        @select="onTextareaCursorChange"
        @focus="onTextareaCursorChange"
        @blur="scheduleMentionClose"
        @compositionstart="onCompositionStart"
        @compositionend="onCompositionEnd"
      />
    </div>
    <div
      v-if="mentionPanelOpen"
      class="mini-mention-panel"
      data-testid="mini-workstation-mention-panel"
      @mousedown.prevent="cancelMentionClose"
    >
      <div class="mini-mention-header">
        <span class="mini-mention-trigger">{{ mentionQuery?.trigger }}</span>
        <span>{{ mentionModeLabel }}</span>
      </div>
      <div v-if="mentionLoading" class="mini-mention-state">搜索中...</div>
      <div v-else-if="mentionOptions.length === 0" class="mini-mention-state">
        {{ mentionEmptyText }}
      </div>
      <div v-else class="mini-mention-list">
        <button
          v-for="(option, index) in mentionOptions"
          :key="option.key"
          type="button"
          :class="['mini-mention-option', { 'is-active': index === highlightedMentionIndex }]"
          :data-testid="`mini-workstation-mention-option-${index}`"
          @mouseenter="highlightedMentionIndex = index"
          @mousedown.prevent="applyMentionOption(option)"
        >
          <span :class="['mini-mention-icon', `is-${option.kind}`, option.iconClass]">
            <el-avatar
              v-if="option.kind === 'user'"
              :src="option.avatar"
              :size="28"
              class="mini-mention-avatar"
            >
              {{ option.initial }}
            </el-avatar>
            <img
              v-else-if="option.iconSrc"
              :src="option.iconSrc"
              :alt="option.typeLabel"
              class="mini-mention-resource-img"
            />
            <component
              :is="option.iconComponent"
              v-else-if="option.iconComponent"
              :size="17"
              class="mini-mention-resource-component"
            />
            <el-icon v-else><Document /></el-icon>
          </span>
          <span class="mini-mention-main">
            <span class="mini-mention-title-row">
              <span class="mini-mention-title" :title="option.label">{{ option.label }}</span>
              <span class="mini-mention-type">{{ option.typeLabel }}</span>
            </span>
            <span v-if="option.description" class="mini-mention-desc" :title="option.description">
              {{ option.description }}
            </span>
            <span v-if="option.metaItems.length > 0" class="mini-mention-meta-row">
              <span
                v-for="meta in option.metaItems"
                :key="`${option.key}-${meta}`"
                class="mini-mention-meta"
                :title="meta"
              >
                {{ meta }}
              </span>
            </span>
          </span>
        </button>
      </div>
    </div>
    <div class="mini-action-stack">
      <el-select
        :model-value="selectedLLMConfigId"
        placeholder="默认模型"
        filterable
        :loading="llmLoading"
        teleported
        fit-input-width
        :persistent="false"
        placement="top-start"
        :popper-options="modelSelectPopperOptions"
        popper-class="mini-ws-model-select-popper"
        class="mini-ws-model-select"
        @update:model-value="emit('update:selectedLLMConfigId', Number($event))"
        @visible-change="onLLMSelectVisibleChange"
      >
        <el-option label="默认模型" :value="0" />
        <el-option
          v-for="llm in llmList"
          :key="llm.id"
          :label="`${llm.name} (${llm.model})`"
          :value="llm.id"
        />
      </el-select>
      <div class="mini-action-row">
        <el-button
          v-if="sending"
          type="danger"
          size="small"
          :loading="stopping"
          data-testid="mini-workstation-stop"
          @click="$emit('stop')"
          class="mini-stop-btn"
        >
          <el-icon><VideoPause /></el-icon>
          停止
        </el-button>
	        <el-button
	          type="primary"
	          size="small"
	          :disabled="blocked || !fullCodePath || (!inputText.trim() && attachedFiles.length === 0)"
	          data-testid="mini-workstation-send"
          @click="$emit('send')"
          class="mini-send-btn"
        >
          {{ sending ? (queuedCount > 0 ? `排队 ${queuedCount}` : '排队') : '发送' }}
        </el-button>
        <el-tooltip
          :content="miniHideShortcutHint"
          placement="top"
          effect="light"
          popper-class="mini-workstation-tooltip-popper"
        >
          <button
            type="button"
            class="mini-hide-btn"
            :title="miniHideShortcutHint"
            data-testid="mini-workstation-collapse"
            @click="$emit('collapse')"
          >
            <el-icon><Close /></el-icon>
          </button>
        </el-tooltip>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, type Component } from 'vue'
import { Close, Document, Paperclip, VideoPause } from '@element-plus/icons-vue'
import type { LLMInfo } from '@/architecture/presentation/context/api/agent'
import type { WorkspaceChatMessageFile } from '@/architecture/presentation/context/api/workspace'
import { searchUsersFuzzy } from '@/architecture/presentation/context/api/user'
import {
  searchResources,
  type ResourceSearchResult
} from '@/architecture/presentation/context/api/service-tree'
import type { UserInfo } from '@/architecture/domain/types'
import { formatUserDisplayName } from '@/architecture/domain/utils/userInfo'
import {
  findMiniComposerMentionQuery,
  replaceMiniComposerMention,
  type MiniComposerMentionKind,
  type MiniComposerMentionQuery
} from './utils/miniComposerMention'
import ChartIcon from '@/architecture/presentation/shared/components/icons/ChartIcon.vue'
import TableIcon from '@/architecture/presentation/shared/components/icons/TableIcon.vue'

const props = defineProps<{
  fullCodePath: string
  dirName?: string
  attachedFiles: WorkspaceChatMessageFile[]
  uploading: boolean
  inputText: string
  sending: boolean
  stopping: boolean
  selectedLLMConfigId: number
	  llmList: LLMInfo[]
	  llmLoading: boolean
	  queuedCount: number
	  blocked?: boolean
	  blockedLabel?: string
	  blockedPlaceholder?: string
	  registerInputRef: (el: HTMLTextAreaElement | null) => void
  onLLMSelectVisibleChange: (visible: boolean) => void
  onFileChange: (uploadFileObj: { raw?: File }) => void | Promise<void>
  removeFile: (index: number) => void
  onInputEnter: (event: KeyboardEvent) => void
  toggleShortcutLabel?: string
}>()

const emit = defineEmits<{
  (e: 'update:inputText', value: string): void
  (e: 'update:selectedLLMConfigId', value: number): void
  (e: 'send'): void
  (e: 'stop'): void
  (e: 'collapse'): void
}>()

const modelSelectPopperOptions = {
  strategy: 'fixed' as const,
}

const miniHideShortcutHint = computed(() => {
  const toggleShortcut = (props.toggleShortcutLabel || '').trim()
  return toggleShortcut ? `关闭工作台，任务继续后台执行 (Esc / ${toggleShortcut})` : '关闭工作台，任务继续后台执行 (Esc)'
})

const displayPath = computed(() => {
  const label = (props.dirName || '').trim()
  if (label) return label
  if (!props.fullCodePath) return '未选择目录'
  const parts = props.fullCodePath.split('/').filter(Boolean)
  return parts[parts.length - 1] || props.fullCodePath
})

interface MiniMentionOption {
  key: string
  kind: MiniComposerMentionKind
  label: string
  value: string
  description: string
  typeLabel: string
  resourceType?: ResourceSearchResult['type']
  avatar?: string
  initial?: string
  iconSrc?: string
  iconComponent?: Component
  iconClass?: string
  metaItems: string[]
}

const localInputRef = ref<HTMLTextAreaElement | null>(null)
const mentionQuery = ref<MiniComposerMentionQuery | null>(null)
const mentionOptions = ref<MiniMentionOption[]>([])
const mentionLoading = ref(false)
const highlightedMentionIndex = ref(0)
const composing = ref(false)
let mentionSearchTimer: ReturnType<typeof setTimeout> | null = null
let mentionCloseTimer: ReturnType<typeof setTimeout> | null = null
let mentionSearchSeq = 0
let activeMentionSearchKey = ''

const mentionPanelOpen = computed(() => !!mentionQuery.value)
const mentionModeLabel = computed(() => mentionQuery.value?.kind === 'user' ? '搜索用户' : '搜索目录、工具和文档')
const mentionEmptyText = computed(() => {
  const query = mentionQuery.value
  if (!query) return ''
  if (!query.query.trim()) {
    return query.kind === 'user' ? '继续输入用户名或邮箱' : '继续输入目录、工具或文档关键词'
  }
  return query.kind === 'user' ? '没有匹配的用户' : '没有匹配的资源'
})

function emitInput(event: Event) {
  if (props.blocked) return
  const textarea = event.target as HTMLTextAreaElement
  emit('update:inputText', textarea.value)
  updateMentionFromTextarea(textarea)
}

function bindInputRef(element: unknown) {
  const textarea = element instanceof HTMLTextAreaElement ? element : null
  localInputRef.value = textarea
  props.registerInputRef(textarea)
}

function onTextareaCursorChange(event: Event) {
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
  updateMentionFromTextarea(event.target as HTMLTextAreaElement)
}

function onTextareaKeydown(event: KeyboardEvent) {
  if (isIMEComposing(event)) {
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
    props.onInputEnter(event)
  }
}

function isIMEComposing(event: KeyboardEvent) {
  return composing.value
    || event.isComposing
    || event.key === 'Process'
    || (event as KeyboardEvent & { keyCode?: number }).keyCode === 229
}

function onCompositionStart() {
  composing.value = true
}

function onCompositionEnd(event: CompositionEvent) {
  composing.value = false
  updateMentionFromTextarea(event.target as HTMLTextAreaElement)
}

function updateMentionFromTextarea(textarea: HTMLTextAreaElement | null) {
  cancelMentionClose()
  const query = textarea
    ? findMiniComposerMentionQuery(textarea.value, textarea.selectionStart)
    : null

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
        page_size: 8
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

function mapUserMentionOption(user: UserInfo): MiniMentionOption {
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
    initial: username[0]?.toUpperCase() || 'U',
    metaItems: compactMetaItems([
      username ? `@${username}` : '',
      department,
      company
    ])
  }
}

function mapResourceMentionOption(resource: ResourceSearchResult): MiniMentionOption {
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
      shouldShowResourceHeat(resource) ? `${formatCompactCount(resource.run_count || 0)} 次运行` : ''
    ]),
    ...getResourceIconMeta(resource)
  }
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

function getResourceIconMeta(resource: ResourceSearchResult): Pick<MiniMentionOption, 'iconSrc' | 'iconComponent' | 'iconClass'> {
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

function moveMentionHighlight(delta: number) {
  const count = mentionOptions.value.length
  if (count === 0) return
  highlightedMentionIndex.value = (highlightedMentionIndex.value + delta + count) % count
}

function applyMentionOption(option: MiniMentionOption | undefined) {
  if (!option || !mentionQuery.value) {
    return
  }

  const currentText = props.inputText
  const replacement = option.kind === 'user' ? `@${option.value}` : option.value
  const result = replaceMiniComposerMention(currentText, mentionQuery.value, replacement)

  emit('update:inputText', result.value)
  closeMentionPanel()

  void nextTick(() => {
    const input = localInputRef.value
    if (!input) return
    input.focus()
    input.setSelectionRange(result.cursor, result.cursor)
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
</script>

<style scoped>
.mini-ws-files {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 7px 12px;
  border-top: 1px solid rgba(96, 231, 255, 0.12);
  background: rgba(4, 12, 24, 0.46);
}
.mini-ws-files :deep(.el-tag) {
  border-color: rgba(34, 211, 238, 0.32);
  background: rgba(34, 211, 238, 0.1);
  color: var(--mini-cyber-text, #d8f8ff);
}

.mini-ws-model-select {
  width: 112px;
  min-width: 0;
}
.mini-ws-model-select :deep(.el-select__wrapper) {
  min-height: 42px;
  border: 1px solid rgba(128, 151, 198, 0.22);
  border-radius: 8px;
  background: rgba(30, 42, 68, 0.78);
  box-shadow: none;
}
.mini-ws-model-select :deep(.el-select__placeholder),
.mini-ws-model-select :deep(.el-select__selected-item) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--mini-cyber-text, #d8f8ff);
}
.mini-ws-input {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  box-sizing: border-box;
  min-height: 50px;
  padding: 4px 10px;
  position: relative;
  border: 1px solid rgba(130, 153, 190, 0.26);
  border-radius: 14px;
  background:
    linear-gradient(180deg, rgba(12, 18, 32, 0.84), rgba(8, 12, 22, 0.68)),
    rgba(8, 12, 22, 0.72);
  box-shadow:
    0 24px 70px rgba(0, 0, 0, 0.42),
    inset 0 1px 0 rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(24px) saturate(140%);
}
.mini-composer-left-actions {
  flex-shrink: 0;
  min-width: 142px;
  display: flex;
  flex-direction: row;
  align-items: center;
  align-self: center;
  gap: 8px;
}
.mini-upload-btn {
  flex-shrink: 0;
}
.mini-upload-btn :deep(.el-button) {
  width: 42px;
  height: 42px;
  border: 1px solid rgba(128, 151, 198, 0.22);
  border-radius: 8px;
  color: #8ed0ff;
  background: rgba(30, 42, 68, 0.72);
}
.mini-upload-btn :deep(.el-button:hover) {
  color: #ffffff;
  background: rgba(55, 163, 255, 0.16);
  box-shadow: 0 0 18px rgba(55, 163, 255, 0.14);
}
.mini-input-wrap {
  position: relative;
  min-width: 0;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  box-sizing: border-box;
  height: 36px;
  padding: 0 10px;
  border: 1px solid rgba(124, 146, 189, 0.16);
  border-radius: 10px;
  background: rgba(10, 16, 29, 0.32);
}
.mini-path-pill {
  max-width: 148px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  padding: 0 10px;
  overflow: hidden;
  border: 1px solid rgba(55, 163, 255, 0.25);
  border-radius: 8px;
  background: rgba(55, 163, 255, 0.13);
  color: #8ed0ff;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mini-blocked-pill {
  position: absolute;
  right: 10px;
  top: 5px;
  z-index: 1;
  max-width: min(180px, calc(100% - 116px));
  height: 24px;
  display: inline-flex;
  align-items: center;
  padding: 0 8px;
  overflow: hidden;
  border: 1px solid rgba(245, 158, 11, 0.28);
  border-radius: 8px;
  background: rgba(245, 158, 11, 0.1);
  color: #f6c76b;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mini-input {
  flex: 1;
  width: 100%;
  min-width: 0;
  height: 28px;
  min-height: 28px;
  max-height: 80px;
  margin: 3px 0;
  padding: 4px 0;
  border: 0;
  border-radius: 0;
  outline: none;
  box-sizing: border-box;
  font-size: 14px;
  line-height: 20px;
  font-family: inherit;
  background: transparent;
  color: #e6f0ff;
  resize: none;
  overflow-y: hidden;
  box-shadow: none;
  transition: border-color 0.18s ease, box-shadow 0.18s ease, background 0.18s ease;
}
.mini-input:focus {
  box-shadow: none;
}
.mini-input::placeholder {
  color: #8291aa;
}
.mini-input:disabled {
  padding-right: min(190px, 42%);
  cursor: not-allowed;
  color: rgba(230, 240, 255, 0.44);
  -webkit-text-fill-color: rgba(230, 240, 255, 0.44);
}
.mini-mention-panel {
  position: absolute;
  left: 92px;
  right: 76px;
  bottom: calc(100% - 4px);
  z-index: 12;
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
.mini-mention-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 10px;
  border-bottom: 1px solid rgba(96, 231, 255, 0.14);
  color: rgba(216, 248, 255, 0.78);
  font-size: 12px;
  font-weight: 700;
}
.mini-mention-trigger {
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
.mini-mention-state {
  min-height: 72px;
  display: flex;
  align-items: center;
  padding: 16px 14px;
  color: rgba(184, 225, 235, 0.68);
  font-size: 12px;
}
.mini-mention-list {
  overflow-y: auto;
  padding: 7px;
}
.mini-mention-list::-webkit-scrollbar {
  width: 7px;
}
.mini-mention-list::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(96, 231, 255, 0.2);
}
.mini-mention-option {
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
.mini-mention-option:hover,
.mini-mention-option.is-active {
  border-color: rgba(34, 211, 238, 0.32);
  background:
    linear-gradient(135deg, rgba(34, 211, 238, 0.13), rgba(124, 255, 196, 0.06)),
    rgba(34, 211, 238, 0.08);
  transform: translateY(-1px);
}
.mini-mention-icon {
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
.mini-mention-icon.is-user {
  overflow: hidden;
  color: #7cffc4;
  background: rgba(124, 255, 196, 0.08);
  border-color: rgba(124, 255, 196, 0.2);
}
.mini-mention-icon.table-icon {
  color: #10b981;
}
.mini-mention-icon.form-icon-img {
  color: #3b82f6;
}
.mini-mention-icon.chart-icon {
  color: #9377e0;
}
.mini-mention-icon.function-icon {
  color: #6366f1;
}
.mini-mention-icon.package-icon-img,
.mini-mention-icon.docs-icon-img,
.mini-mention-icon.form-icon-img {
  background: rgba(255, 255, 255, 0.04);
}
.mini-mention-avatar {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  font-size: 12px;
  font-weight: 700;
}
.mini-mention-resource-img {
  width: 22px;
  height: 22px;
  object-fit: contain;
  flex-shrink: 0;
  opacity: 0.94;
}
.mini-mention-resource-component {
  width: 22px;
  height: 22px;
  flex-shrink: 0;
}
.mini-mention-main,
.mini-mention-title-row,
.mini-mention-title,
.mini-mention-desc,
.mini-mention-meta-row,
.mini-mention-meta {
  min-width: 0;
}
.mini-mention-main {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.mini-mention-title-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.mini-mention-title {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--mini-cyber-text, #d8f8ff);
  font-size: 13px;
  font-weight: 700;
}
.mini-mention-type {
  flex-shrink: 0;
  border: 1px solid rgba(96, 231, 255, 0.18);
  border-radius: 999px;
  padding: 2px 6px;
  color: rgba(96, 231, 255, 0.86);
  font-size: 11px;
  line-height: 1.2;
}
.mini-mention-desc {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: rgba(184, 225, 235, 0.62);
  font-size: 12px;
}
.mini-mention-meta-row {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  overflow: hidden;
}
.mini-mention-meta {
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
.mini-action-stack {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}
.mini-action-row {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}
.mini-action-row :deep(.el-button + .el-button) {
  margin-left: 0;
}
.mini-send-btn,
.mini-stop-btn {
  flex-shrink: 0;
  min-width: 108px;
  min-height: 42px;
  border-radius: 8px;
  font-weight: 700;
  letter-spacing: 0;
  box-shadow: 0 0 20px rgba(104, 119, 255, 0.16);
}
.mini-send-btn.el-button--primary {
  border: 0;
  background: linear-gradient(135deg, #6d70ff, #8b5cf6);
  color: #ffffff;
}

.mini-hide-btn {
  width: 42px;
  height: 42px;
  min-width: 42px;
  display: grid;
  place-items: center;
  border: 1px solid rgba(124, 146, 189, 0.22);
  border-radius: 8px;
  background: rgba(30, 42, 68, 0.78);
  color: #c8d8ef;
}

.mini-hide-btn:hover {
  border-color: rgba(83, 174, 255, 0.42);
  background: rgba(55, 163, 255, 0.16);
  color: #ffffff;
}

:deep(.mini-ws--maximized) .mini-ws-input {
  padding: 4px 10px;
}
:deep(.mini-ws--maximized) .mini-ws-files {
  padding: 6px 24px;
}

@media (max-width: 520px) {
  .mini-mention-panel {
    left: 12px;
    right: 12px;
    bottom: calc(100% + 6px);
  }
}
</style>

<style lang="scss">
.mini-ws-model-select-popper.el-select__popper {
  max-width: min(360px, calc(100vw - 24px));
  overflow: hidden;
  border: 1px solid rgba(96, 231, 255, 0.18);
  background:
    radial-gradient(circle at 20% 0%, rgba(34, 211, 238, 0.12), transparent 34%),
    linear-gradient(150deg, rgba(6, 18, 33, 0.98), rgba(11, 30, 49, 0.96));
  box-shadow: 0 18px 46px rgba(0, 0, 0, 0.36), 0 0 24px rgba(34, 211, 238, 0.1);
}

.mini-ws-model-select-popper.el-select__popper:not([data-popper-placement]) {
  visibility: hidden;
}

.mini-ws-model-select-popper.el-select__popper .el-select-dropdown {
  width: 100% !important;
  min-width: 0 !important;
  max-width: 100%;
  background: transparent;
}

.mini-ws-model-select-popper.el-select__popper .el-scrollbar,
.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__wrap,
.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__list {
  max-width: 100%;
}

.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__item {
  max-width: 100%;
  padding-right: 28px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: rgba(216, 248, 255, 0.78);
}

.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__item span {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__item.is-hovering,
.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__item:hover {
  background: rgba(34, 211, 238, 0.1);
  color: #d8f8ff;
}

.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__item.is-selected {
  color: #22d3ee;
  background: rgba(34, 211, 238, 0.14);
}

.mini-ws-model-select-popper.el-select__popper .el-popper__arrow::before {
  background: rgba(8, 22, 38, 0.98);
  border-color: rgba(96, 231, 255, 0.18);
}
</style>
