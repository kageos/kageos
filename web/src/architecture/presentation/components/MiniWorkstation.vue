<!--
  MiniWorkstation - 工作台面板
  支持输入命令、上传文件、SSE 实时输出、后台保持任务状态。
-->
<template>
  <transition name="mini-ws-pop">
    <div
      v-if="visible && !collapsed"
      ref="rootRef"
      :class="['mini-ws', { 'mini-ws--maximized': maximized, 'mini-ws--compact': !maximized, 'mini-ws--sending': sending, 'mini-ws--interaction-open': interactionOpen }]"
      data-testid="mini-workstation"
      :data-full-code-path="fullCodePath"
      :style="windowStyle"
      @dragover.prevent="onDragOver"
      @dragleave.prevent="onDragLeave"
      @drop.prevent="onDrop"
      @paste="onPaste"
    >
      <div class="mini-workspace-backdrop" aria-hidden="true"></div>
      <section class="mini-shell">
        <header class="mini-drawer-head">
          <div class="mini-drawer-title">
            <strong>工作台</strong>
            <span :title="fullCodePath">{{ dirName || displayPath }}</span>
          </div>
          <div class="mini-drawer-actions">
            <button
              type="button"
              class="mini-drawer-primary-action"
              title="在当前工作台目录新建会话"
              @click="startNewSession"
            >
              <el-icon><Plus /></el-icon>
              <span>新建会话</span>
            </button>
            <button
              type="button"
              class="mini-drawer-secondary-action"
              :disabled="!lastDrawerSession"
              :title="lastDrawerSession ? `打开：${getSessionTitle(lastDrawerSession)}` : '暂无可打开的上次会话'"
              @click="openLastDrawerSession"
            >
              <el-icon><Clock /></el-icon>
              <span>打开上次</span>
            </button>
            <button
              type="button"
              class="mini-drawer-icon-action"
              :title="maximized ? '收回右侧面板' : '全屏展示工作台'"
              @click="toggleDrawerWidth"
            >
              <el-icon>
                <ArrowRight v-if="maximized" />
                <ArrowLeft v-else />
              </el-icon>
            </button>
            <button
              type="button"
              class="mini-drawer-icon-action"
              :title="`关闭工作台，任务会继续在后台执行（${toggleShortcutLabel || '快捷键'}）`"
              @click="hideWorkstation"
            >
              <el-icon><Close /></el-icon>
            </button>
          </div>
        </header>

        <section class="mini-current-output">
          <div :class="['mini-current-layout', { 'is-artifact-open': artifactPanelExpanded }]">
            <aside class="mini-current-meta">
              <header class="mini-current-session-head">
                <div>
                  <strong>会话列表</strong>
                  <span :title="fullCodePath">{{ dirName || displayPath }}</span>
                </div>
                <em>{{ drawerSessionList.length }}</em>
              </header>
              <div v-if="hasDifferentCurrentContext" class="mini-current-context-switch">
                <span>当前页面</span>
                <strong :title="normalizedCurrentContextPath">{{ currentContextName }}</strong>
                <button type="button" @click="openCurrentContextNewSession">
                  当前目录新会话
                </button>
              </div>
              <div class="mini-drawer-scope-tabs" role="tablist" aria-label="会话范围">
                <button
                  type="button"
                  :class="{ active: drawerSessionScope === 'current' }"
                  @click="setDrawerSessionScope('current')"
                >
                  当前目录
                </button>
                <button
                  type="button"
                  :class="{ active: drawerSessionScope === 'all' }"
                  @click="setDrawerSessionScope('all')"
                >
                  全部会话
                </button>
              </div>
              <label class="mini-drawer-session-search">
                <el-icon :size="14"><Search /></el-icon>
                <input v-model="sessionSearchKeyword" placeholder="搜索会话、目录或角色" />
              </label>
              <div class="mini-drawer-session-filters">
                <button
                  v-for="filter in sessionFilters"
                  :key="filter.value"
                  type="button"
                  :class="{ active: sessionFilter === filter.value }"
                  @click="sessionFilter = filter.value"
                >
                  {{ filter.label }}
                </button>
              </div>
              <div class="mini-current-session-list">
                <button
                  v-for="item in drawerSessionList"
                  :key="item.session_id"
                  type="button"
                  :class="['mini-current-session-row', getSessionStatusClass(item), { active: item.session_id === sessionId }]"
                  :title="getSessionTitle(item)"
                  @click="handleDrawerSessionSelect(item)"
                >
                  <span class="mini-status-dot" :class="getSessionStatusClass(item)"></span>
                  <span class="mini-current-session-copy">
                    <span class="mini-current-session-title">{{ getSessionTitle(item) }}</span>
                    <span class="mini-current-session-sub">
                      {{ getSessionStatusLabel(item) }} · {{ formatRelativeTime(item.updated_at || item.created_at) }}
                    </span>
                  </span>
                </button>
                <button
                  v-if="drawerSessionList.length === 0"
                  type="button"
                  class="mini-current-session-row active is-draft"
                  @click="startNewSession"
                >
                  <span class="mini-status-dot"></span>
                  <span class="mini-current-session-copy">
                    <span class="mini-current-session-title">
                      {{ drawerSessionScope === 'current' ? '当前目录暂无会话' : '暂无匹配会话' }}
                    </span>
                    <span class="mini-current-session-sub">点击新建会话开始</span>
                  </span>
                </button>
              </div>
              <div v-if="queuedCount > 0" class="mini-queue-chip">{{ queuedCount }} 条排队</div>
            </aside>
            <div class="mini-current-stream">
              <div class="mini-ws-output" ref="outputRef" @scroll.passive="captureOutputScroll">
                <MiniWorkstationMessages
                  :messages="messages"
                  :maximized="maximized"
                  :sending="sending"
                  :streaming-display-length="streamingDisplayLength"
                  :render-markdown="renderMarkdown"
                  :format-message-time="formatMessageTime"
                  :get-file-groups-from-calls="getFileGroupsFromCalls"
                  :get-display-fields-from-calls="getDisplayFieldsFromCalls"
                  @confirm-prd="handleConfirmPrd"
                />
              </div>
            </div>
            <section :class="['mini-artifact-drawer', { 'is-open': artifactPanelExpanded }]">
              <button
                type="button"
                class="mini-artifact-toggle"
                :aria-expanded="artifactPanelExpanded"
                title="展开或收起产物"
                @click="toggleArtifactPanel"
              >
                <span>产物</span>
                <strong>{{ artifactToggleCount }} 项</strong>
                <em>{{ artifactPanelExpanded ? '收起' : '展开' }}</em>
                <el-icon>
                  <ArrowUp v-if="artifactPanelExpanded" />
                  <ArrowDown v-else />
                </el-icon>
              </button>
              <MiniWorkstationArtifactPanel
                v-if="artifactPanelExpanded"
                :artifact-items="artifactItems"
                :maximized="maximized"
                :panel-has-content="panelHasContent"
                :panel-item-count="panelItemCount"
                :uploaded-files="uploadedFiles"
                :output-files="outputFiles"
                :display-fields="allPanelDisplayFields"
                :display-field-preview-visible="dfPreviewVisible"
                @artifact-click="handleArtifactClick"
                @preview-file="previewFile"
                @download-file="downloadFile"
                @preview-field="openDfPreview"
                @copy-field="copyDisplayFieldValue"
              />
            </section>
          </div>
        </section>

        <MiniWorkstationPendingActionBar
          v-if="pendingPrd"
          variant="prd"
          :help-text="pendingPrdHelpText"
          :sending="sending"
          @view="focusPrdPreview"
          @revise="prepareRevisePrd"
          @cancel="cancelPendingPrd"
          @confirm="confirmPendingPrd"
        />

        <MiniWorkstationPendingActionBar
          v-else-if="pendingTestHandoff"
          variant="test"
          :help-text="pendingTestHandoffHelpText"
          :sending="sending"
          @view="focusPrdPreview"
          @revise="prepareContinueDevelopment"
          @cancel="cancelPendingTestHandoff"
          @confirm="confirmPendingTestHandoff"
        />

        <MiniWorkstationComposer
          :full-code-path="fullCodePath"
          :dir-name="dirName || displayPath"
          :attached-files="attachedFiles"
          :uploading="uploading"
          :input-text="inputText"
          :sending="sending"
          :stopping="stopping"
          :queued-count="queuedCount"
          :selected-l-l-m-config-id="selectedLLMConfigId"
          :llm-list="llmList"
          :llm-loading="llmLoading"
          :register-input-ref="registerInputRef"
          :on-l-l-m-select-visible-change="onLLMSelectVisibleChange"
          :on-file-change="onFileChange"
          :remove-file="removeFile"
          :on-input-enter="onInputEnter"
          :toggle-shortcut-label="toggleShortcutLabel"
          @update:input-text="inputText = $event"
          @update:selected-l-l-m-config-id="selectedLLMConfigId = $event"
          @stop="handleStopSession"
          @send="handleSend"
          @collapse="hideWorkstation"
        >
          <template #left-actions>
            <el-popover
              trigger="click"
              placement="top-start"
              width="430"
              popper-class="mini-settings-popover"
              @visible-change="settingsPopoverOpen = $event"
            >
              <template #reference>
                <el-button
                  link
                  size="small"
                  class="mini-settings-btn"
                  title="设置"
                  data-testid="mini-workstation-settings"
                  @mousedown.stop
                  @click.stop
                >
                  <el-icon :size="15"><Setting /></el-icon>
                </el-button>
              </template>
              <MiniWorkstationDebugSettings
                :debug-tool-steps="debugToolSteps"
                :debug-success-count="debugSuccessCount"
                :debug-error-count="debugErrorCount"
                @copy-conversation="copyDebugConversation"
                @copy-tool-summary="copyDebugToolSummary"
              />
            </el-popover>
          </template>
        </MiniWorkstationComposer>
      </section>

      <!-- 拖拽上传遮罩 -->
      <transition name="el-fade-in-linear">
        <div v-if="dragOver" class="mini-ws-drop-overlay">
          <el-icon :size="28"><UploadFilled /></el-icon>
          <span>松开上传文件</span>
        </div>
      </transition>

    </div>
  </transition>

  <MiniWorkstationDisplayFieldPreviewDialog
    :visible="dfPreviewVisible"
    :label="dfPreviewLabel"
    :content="dfPreviewContent"
    @close="closeDfPreview"
    @copy="copyDfPreviewContent"
    @update:content="dfPreviewContent = $event"
  />

</template>

<script setup lang="ts">
import { nextTick, onMounted, ref, computed, watch } from 'vue'
import {
  ArrowDown,
  ArrowLeft,
  ArrowRight,
  ArrowUp,
  Clock,
  Close,
  Plus,
  Search,
  UploadFilled,
  Setting
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import {
  useWorkspaceChatStream,
  type ChatMessage
} from '@/architecture/presentation/composables/useWorkspaceChatStream'
import MiniWorkstationArtifactPanel from './MiniWorkstationArtifactPanel.vue'
import MiniWorkstationDisplayFieldPreviewDialog from './MiniWorkstationDisplayFieldPreviewDialog.vue'
import MiniWorkstationComposer from './MiniWorkstationComposer.vue'
import MiniWorkstationDebugSettings from './MiniWorkstationDebugSettings.vue'
import MiniWorkstationMessages from './MiniWorkstationMessages.vue'
import MiniWorkstationPendingActionBar from './MiniWorkstationPendingActionBar.vue'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'
import { useMiniWorkstationPanel } from '../composables/useMiniWorkstationPanel'
import { useMiniWorkstationSessions } from '../composables/useMiniWorkstationSessions'
import { useMiniWorkstationUploads } from '../composables/useMiniWorkstationUploads'
import { useMiniWorkstationComposer } from '../composables/useMiniWorkstationComposer'
import { useMiniWorkstationEffects } from '../composables/useMiniWorkstationEffects'
import {
  buildDisplayFieldArtifactItem,
  buildFileArtifactItem,
  isGeneratedArtifactToolCall,
  type MiniArtifactItem
} from '../composables/useMiniWorkstationArtifacts'
import {
  collectMessageToolCalls,
  useMiniWorkstationDebugCopy
} from '../composables/useMiniWorkstationDebugCopy'
import {
  miniWorkstationSessionFilters,
  useMiniWorkstationSessionView,
  type SessionFilterValue
} from '../composables/useMiniWorkstationSessionView'
import { eventBus } from '@/architecture/presentation/context/eventBusContext'
import { createWorkspaceHandoff, resolveWorkspaceSessionInteraction, type WorkspaceSessionItem } from '@/architecture/presentation/context/api/workspace'

const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()

const props = defineProps<{
  visible: boolean
  fullCodePath: string
  dirName?: string
  initialSessionId?: string
  initialOffset?: number
  initialPosition?: 'center'
  initialExpanded?: boolean
  initialMaximized?: boolean
  pathNameMap?: Record<string, string>
  currentFullCodePath?: string
  currentDirName?: string
  toggleShortcutLabel?: string
}>()

const fullCodePathRef = computed(() => props.fullCodePath)
const initialSessionIdRef = computed(() => props.initialSessionId)

const emit = defineEmits<{
  (e: 'minimize'): void
  (e: 'close'): void
  (e: 'task-started', sessionId: string): void
  (e: 'tool-call-ok', payload: { name: string }): void
  (e: 'open-current-new-session', payload: { fullCodePath: string; dirName: string }): void
  (e: 'expanded-change', payload: { expanded: boolean; sessionId?: string }): void
  (e: 'maximize-change', payload: { maximized: boolean; sessionId?: string }): void
}>()

const { messages, sending, sessionId, streamingDisplayLength, send: sendMessage, setMessages } = useWorkspaceChatStream()
const rootRef = ref<HTMLElement>()
const outputRef = ref<HTMLElement>()
const inputText = ref('')
const inputRef = ref<HTMLTextAreaElement>()
const llmSelectOpen = ref(false)
const settingsPopoverOpen = ref(false)
const interactionOpen = computed(() => llmSelectOpen.value || settingsPopoverOpen.value)
const confirmedPrdKeys = ref<Set<string>>(new Set())
const confirmedTestHandoffKeys = ref<Set<string>>(new Set())
const artifactPanelExpanded = ref(false)
const collapsed = ref(props.initialExpanded === false)
const suppressAutoSelectLatestSession = ref(false)
const sessionSearchKeyword = ref('')
const sessionFilter = ref<SessionFilterValue>('all')
type DrawerSessionScope = 'current' | 'all'
const DRAWER_SESSION_SCOPE_STORAGE_KEY = 'workspace-mini-session-scope'
const drawerSessionScope = ref<DrawerSessionScope>(readStoredDrawerSessionScope())

const windowStyle = computed(() => ({
  '--mini-stack-offset': `${props.initialOffset || 0}px`
}))

const OUTPUT_SCROLL_BOTTOM_THRESHOLD = 96
const savedOutputScrollTop = ref(0)
const savedOutputWasNearBottom = ref(true)

function isOutputNearBottom(element: HTMLElement) {
  return element.scrollHeight - element.scrollTop - element.clientHeight <= OUTPUT_SCROLL_BOTTOM_THRESHOLD
}

function captureOutputScroll() {
  const element = outputRef.value
  if (!element) return
  savedOutputScrollTop.value = element.scrollTop
  savedOutputWasNearBottom.value = isOutputNearBottom(element)
}

function restoreOutputScroll() {
  nextTick(() => {
    const restore = () => {
      const element = outputRef.value
      if (!element) return
      if (savedOutputWasNearBottom.value) {
        element.scrollTop = element.scrollHeight
        return
      }
      const maxScrollTop = Math.max(0, element.scrollHeight - element.clientHeight)
      element.scrollTop = Math.min(savedOutputScrollTop.value, maxScrollTop)
    }

    requestAnimationFrame(restore)
  })
}

function resetOutputScrollState() {
  savedOutputScrollTop.value = 0
  savedOutputWasNearBottom.value = true
}

function registerInputRef(element: HTMLTextAreaElement | null) {
  inputRef.value = element || undefined
}

// 首条用户消息摘要，用于同目录多 Mini 时区分（如「分析数据」「帮我把xxx改成」）
const firstUserMessagePreview = computed(() => {
  const first = messages.value?.find(m => m.role === 'user')
  const content = typeof first?.content === 'string' ? first.content.trim() : ''
  if (!content) return ''
  const maxLen = 12
  return content.length > maxLen ? content.slice(0, maxLen) + '…' : content
})

// ─── 当前输出显示 ───
const maximized = ref(!!props.initialMaximized)

watch(() => props.initialMaximized, (value) => {
  maximized.value = !!value
})

watch(() => props.initialExpanded, (value) => {
  collapsed.value = value === false
})

const {
  miniSessionList,
  globalSessionList,
  stopping,
  loadMiniSessions,
  loadGlobalSessions,
  handleNewSession,
  handleStopSession,
  handleSelectSession,
  formatRelativeTime,
  formatMessageTime,
  startMiniStreamListening,
  startMiniPoll,
  stopMiniPoll
} = useMiniWorkstationSessions({
  fullCodePath: fullCodePathRef,
  initialSessionId: initialSessionIdRef,
  maximized,
  sending,
  sessionId,
  setMessages,
  onSelectMaximizedSession: (targetSessionId) => {
    emit('maximize-change', { maximized: true, sessionId: targetSessionId })
  }
})

// ─── 文件预览辅助 ───
const {
  getFileGroupsFromCalls,
  getDisplayFieldsFromCalls,
  allPanelDisplayFields,
  uploadedFiles,
  outputFiles,
  panelHasContent,
  panelItemCount,
  copyDisplayFieldValue,
  dfPreviewVisible,
  dfPreviewLabel,
  dfPreviewContent,
  openDfPreview,
  closeDfPreview,
  copyDfPreviewContent,
  previewFile,
  downloadFile
} = useMiniWorkstationPanel(messages)

const {
  attachedFiles,
  uploading,
  dragOver,
  onFileChange,
  removeFile,
  onDragOver,
  onDragLeave,
  onDrop,
  onPaste
} = useMiniWorkstationUploads({
  fullCodePath: fullCodePathRef,
  inputText,
  inputRef
})

const artifactItems = computed<MiniArtifactItem[]>(() => {
  const files = outputFiles.value.map((file, index) => buildFileArtifactItem(file, index))
  const fields = allPanelDisplayFields.value.map((field, index) => buildDisplayFieldArtifactItem(field, index))
  return [...files, ...fields]
})

const artifactToggleCount = computed(() => Math.max(artifactItems.value.length, panelItemCount.value))

const hasCurrentGeneratedArtifacts = computed(() => {
  if (artifactItems.value.length > 0) return true
  return messages.value.some(messageHasGeneratedArtifacts)
})

const sessionFilters = miniWorkstationSessionFilters

const {
  displayPath,
  currentDirectorySessionList,
  recentSessionCenterList,
  getSessionTitle,
  getSessionDirectoryPath,
  getSessionTimestamp,
  getSessionStatusLabel,
  getSessionStatusClass,
  normalizeFullCodePath
} = useMiniWorkstationSessionView({
  miniSessionList,
  globalSessionList,
  sessionId,
  sending,
  fullCodePath: fullCodePathRef,
  dirName: () => props.dirName,
  pathNameMap: () => props.pathNameMap,
  firstUserMessagePreview,
  hasCurrentGeneratedArtifacts,
  sessionSearchKeyword,
  sessionFilter
})

const normalizedWorkbenchPath = computed(() => normalizeFullCodePath(props.fullCodePath || ''))
const normalizedCurrentContextPath = computed(() => normalizeFullCodePath(props.currentFullCodePath || ''))
const currentContextName = computed(() => {
  const label = (props.currentDirName || '').trim()
  if (label) return label
  const path = normalizedCurrentContextPath.value
  return path.split('/').filter(Boolean).pop() || '当前目录'
})
const hasDifferentCurrentContext = computed(() => {
  return !!normalizedCurrentContextPath.value && normalizedCurrentContextPath.value !== normalizedWorkbenchPath.value
})

const drawerSessionList = computed(() => {
  return drawerSessionScope.value === 'current'
    ? currentDirectorySessionList.value
    : recentSessionCenterList.value
})

const lastDrawerSession = computed<WorkspaceSessionItem | null>(() => {
  return [...drawerSessionList.value]
    .sort((left, right) => getSessionTimestamp(right) - getSessionTimestamp(left))[0] || null
})

watch(drawerSessionScope, (scope) => {
  try {
    localStorage.setItem(DRAWER_SESSION_SCOPE_STORAGE_KEY, scope)
  } catch {
    // localStorage 不可用时仅使用本次会话状态。
  }
  if (props.visible) {
    loadDrawerSessions()
  }
})

function loadDrawerSessions() {
  void loadMiniSessions()
  if (drawerSessionScope.value === 'all') {
    void loadGlobalSessions()
  }
}

function readStoredDrawerSessionScope(): DrawerSessionScope {
  try {
    const stored = localStorage.getItem(DRAWER_SESSION_SCOPE_STORAGE_KEY)
    return stored === 'all' ? 'all' : 'current'
  } catch {
    return 'current'
  }
}

function setDrawerSessionScope(scope: DrawerSessionScope) {
  drawerSessionScope.value = scope
}

function handleDrawerSessionSelect(session: WorkspaceSessionItem) {
  if (session.session_id && session.session_id === sessionId.value) {
    return
  }
  requestSessionSwitch(session)
}

function openLastDrawerSession() {
  if (!lastDrawerSession.value) return
  requestSessionSwitch(lastDrawerSession.value)
}

function toggleArtifactPanel() {
  artifactPanelExpanded.value = !artifactPanelExpanded.value
  restoreOutputScroll()
}

function setCollapsed(value: boolean, sessionIdOverride?: string) {
  if (value) {
    captureOutputScroll()
  }
  if (value && maximized.value) {
    maximized.value = false
    emit('maximize-change', { maximized: false, sessionId: sessionId.value })
  }
  collapsed.value = value
  emit('expanded-change', {
    expanded: !value,
    sessionId: sessionIdOverride !== undefined ? sessionIdOverride : sessionId.value
  })
  if (!value) {
    restoreOutputScroll()
    setTimeout(() => inputRef.value?.focus(), 80)
  }
}

function hideWorkstation() {
  captureOutputScroll()
  emit('minimize')
}

function toggleDrawerWidth() {
  maximized.value = !maximized.value
  if (maximized.value) {
    stopMiniPoll()
  } else if (sessionId.value && !sending.value) {
    startMiniPoll(sessionId.value)
  }
  restoreOutputScroll()
  emit('maximize-change', { maximized: maximized.value, sessionId: sessionId.value })
}

function startNewSession() {
  suppressAutoSelectLatestSession.value = true
  handleNewSession()
  resetOutputScrollState()
  setCollapsed(false, '')
}

function openCurrentContextNewSession() {
  const fullCodePath = normalizedCurrentContextPath.value
  if (!fullCodePath) return
  emit('open-current-new-session', {
    fullCodePath,
    dirName: currentContextName.value
  })
}

function requestSessionSwitch(session: WorkspaceSessionItem) {
  const targetSessionId = (session.session_id || '').trim()
  const targetFullCodePath = (session.full_code_path || props.fullCodePath || '').trim()
  if (!targetSessionId || !targetFullCodePath) {
    return
  }

  eventBus.emit('workspace:open-workstation', {
    full_code_path: targetFullCodePath,
    session_id: targetSessionId,
    directory_name: session.directory_name || getSessionDirectoryPath(session),
    initial_maximized: maximized.value,
    open_as_mini: true
  })
}

function handleArtifactClick(item: MiniArtifactItem) {
  if (item.file) {
    previewFile(item.file)
    return
  }
  if (item.field) {
    openDfPreview(item.field)
  }
}

function messageHasGeneratedArtifacts(message: ChatMessage) {
  if (message.role !== 'assistant') return false
  return collectMessageToolCalls(message).some(isGeneratedArtifactToolCall)
}

const {
  debugToolSteps,
  debugSuccessCount,
  debugErrorCount,
  copyDebugConversation,
  copyDebugToolSummary
} = useMiniWorkstationDebugCopy({
  messages,
  fullCodePath: () => props.fullCodePath,
  dirName: () => props.dirName,
  displayPath,
  sessionId
})

const {
  llmList,
  llmLoading,
  selectedLLMConfigId,
  queuedCount,
  onLLMSelectVisibleChange: loadLLMOptionsOnVisibleChange,
  onInputEnter,
  handleSend,
  sendTextToSession
} = useMiniWorkstationComposer({
  fullCodePath: fullCodePathRef,
  sessionId,
  maximized,
  inputText,
  inputRef,
  attachedFiles,
  sending,
  sendMessage,
  beforeSend: handleBeforeSend,
  onTaskStarted: (startedSessionId) => {
    void loadMiniSessions()
    void loadGlobalSessions()
    setCollapsed(false, startedSessionId)
    maximized.value = true
    stopMiniPoll()
    emit('task-started', startedSessionId)
    emit('maximize-change', { maximized: true, sessionId: startedSessionId })
  },
  onToolCallOk: (payload) => {
    emit('tool-call-ok', payload)
  }
})

interface PrdInteractionData {
  kind?: string
  interaction?: {
    artifact_kind?: string
    status?: string
    help_text?: string
    target_role_on_confirm?: string
    allowed_actions?: string[]
  }
  project?: {
    name?: string
    code?: string
  }
}

interface BuildTestInteractionData {
  kind?: string
  workspace_path?: string
  app?: string
  old_version?: string
  new_version?: string
  interaction?: {
    artifact_kind?: string
    status?: string
    help_text?: string
    target_role_on_confirm?: string
    allowed_actions?: string[]
  }
}

const pendingPrd = computed<PrdInteractionData | null>(() => {
  for (let i = messages.value.length - 1; i >= 0; i--) {
    const message = messages.value[i]
    if (!message) continue
    const calls = collectMessageToolCalls(message)
    for (let j = calls.length - 1; j >= 0; j--) {
      const call = calls[j]
      if (!call) continue
      if (call.name !== 'write_prd' || call.status !== 'ok' || !isPrdInteractionData(call.result_data)) continue
      const key = getStageArtifactKey(call.result_data)
      if (confirmedPrdKeys.value.has(key)) continue
      return call.result_data
    }
  }
  return null
})

const pendingPrdHelpText = computed(() => {
  return pendingPrd.value?.interaction?.help_text || 'PRD 已生成，请确认后进入开发；看不到按钮也可以直接回复：确认 PRD。'
})

const pendingTestHandoff = computed<BuildTestInteractionData | null>(() => {
  for (let i = messages.value.length - 1; i >= 0; i--) {
    const message = messages.value[i]
    if (!message) continue
    const calls = collectMessageToolCalls(message)
    for (let j = calls.length - 1; j >= 0; j--) {
      const call = calls[j]
      if (!call) continue
      if (call.name !== 'build_workspace' || call.status !== 'ok' || !isBuildTestInteractionData(call.result_data)) continue
      const key = getStageArtifactKey(call.result_data)
      if (confirmedTestHandoffKeys.value.has(key)) continue
      return call.result_data
    }
  }
  return null
})

const pendingTestHandoffHelpText = computed(() => {
  return pendingTestHandoff.value?.interaction?.help_text || '应用已编译部署，请确认是否进入测试工程师验证。'
})

const hasCurrentOutputContent = computed(() => {
  return sending.value
    || messages.value.length > 0
    || artifactItems.value.length > 0
    || !!pendingPrd.value
    || !!pendingTestHandoff.value
})

const showCurrentOutput = computed(() => {
  return hasCurrentOutputContent.value
})

watch(
  () => props.visible,
  (visible, previousVisible) => {
    if (!visible && previousVisible) {
      captureOutputScroll()
      return
    }
    if (visible) {
      if (!previousVisible) {
        suppressAutoSelectLatestSession.value = false
      }
      loadDrawerSessions()
      restoreOutputScroll()
    }
  },
  { flush: 'pre' }
)

watch(
  () => props.fullCodePath,
  () => {
    suppressAutoSelectLatestSession.value = false
    if (!props.visible) return
    loadDrawerSessions()
  }
)

onMounted(() => {
  if (props.visible) {
    loadDrawerSessions()
  }
})

watch(
  [() => props.visible, collapsed, initialSessionIdRef, sessionId, miniSessionList],
  () => {
    if (!props.visible || collapsed.value) return
    if (initialSessionIdRef.value || sessionId.value || suppressAutoSelectLatestSession.value) return
    const latestSession = [...miniSessionList.value]
      .sort((left, right) => getSessionTimestamp(right) - getSessionTimestamp(left))[0]
    if (!latestSession?.session_id) return
    void handleSelectSession(latestSession.session_id)
  },
  { flush: 'post' }
)

watch(showCurrentOutput, (visible, previousVisible) => {
  if (!visible && previousVisible) {
    captureOutputScroll()
    return
  }
  if (visible) {
    restoreOutputScroll()
  }
}, { flush: 'pre' })

watch(outputRef, (element) => {
  if (element) {
    restoreOutputScroll()
  }
}, { flush: 'post' })

watch(sessionId, (current, previous) => {
  if (current !== previous) {
    resetOutputScrollState()
  }
})

function isPrdInteractionData(value: unknown): value is PrdInteractionData {
  if (!value || typeof value !== 'object') return false
  const data = value as PrdInteractionData
  return data.kind === 'agent_app_prd' && data.interaction?.status === 'pending_confirmation'
}

function isBuildTestInteractionData(value: unknown): value is BuildTestInteractionData {
  if (!value || typeof value !== 'object') return false
  const data = value as BuildTestInteractionData
  return data.kind === 'agent_app_build' && data.interaction?.status === 'pending_test'
}

function getStageArtifactKey(artifact: unknown) {
  try {
    return JSON.stringify(artifact)
  } catch {
    return String(artifact)
  }
}

function markPrdConfirmed(prd: unknown) {
  const next = new Set(confirmedPrdKeys.value)
  next.add(getStageArtifactKey(prd))
  confirmedPrdKeys.value = next
}

function markTestHandoffHandled(artifact: unknown) {
  const next = new Set(confirmedTestHandoffKeys.value)
  next.add(getStageArtifactKey(artifact))
  confirmedTestHandoffKeys.value = next
}

async function clearCurrentPendingInteractionStatus() {
  if (!sessionId.value) return
  try {
    await resolveWorkspaceSessionInteraction(sessionId.value)
    void loadMiniSessions()
    void loadGlobalSessions()
  } catch (error: any) {
    ElMessage.warning(error?.message || '待确认状态同步失败')
  }
}

async function handleBeforeSend(payload: { text: string; files: unknown[] | null }) {
  const text = payload.text.trim()
  if (payload.files?.length) return false
  if (pendingPrd.value) {
    if (isConfirmPrdText(text)) {
      await handleConfirmPrd({ remark: '', prd: pendingPrd.value })
      return true
    }
    if (isCancelPrdText(text)) {
      markPrdConfirmed(pendingPrd.value)
      await clearCurrentPendingInteractionStatus()
      ElMessage.info('已取消本次 PRD 确认')
      return true
    }
  }
  if (pendingTestHandoff.value) {
    if (isStartTestText(text)) {
      await handleConfirmTestHandoff({ artifact: pendingTestHandoff.value })
      return true
    }
    if (isSkipTestText(text)) {
      markTestHandoffHandled(pendingTestHandoff.value)
      await clearCurrentPendingInteractionStatus()
      ElMessage.info('已暂不进入测试')
      return true
    }
  }
  return false
}

function isConfirmPrdText(text: string) {
  const normalized = text.replace(/\s+/g, '').toLowerCase()
  return ['确认', '确认prd', '可以', '没问题', '按这个做', '开始开发'].includes(normalized)
}

function isCancelPrdText(text: string) {
  const normalized = text.replace(/\s+/g, '').toLowerCase()
  return ['取消', '取消prd', '先不做', '不用了'].includes(normalized)
}

function isStartTestText(text: string) {
  const normalized = text.replace(/\s+/g, '').toLowerCase()
  return ['开始测试', '测试', '进入测试', '切到测试', '切换到测试', '验证', '开始验证'].includes(normalized)
}

function isSkipTestText(text: string) {
  const normalized = text.replace(/\s+/g, '').toLowerCase()
  return ['暂不测试', '先不测试', '不用测试', '跳过测试'].includes(normalized)
}

function confirmPendingPrd() {
  if (!pendingPrd.value) return
  void handleConfirmPrd({ remark: '', prd: pendingPrd.value })
}

function confirmPendingTestHandoff() {
  if (!pendingTestHandoff.value) return
  void handleConfirmTestHandoff({ artifact: pendingTestHandoff.value })
}

function prepareRevisePrd() {
  inputText.value = inputText.value.trim() || '修改 PRD：'
  setTimeout(() => inputRef.value?.focus(), 0)
}

function prepareContinueDevelopment() {
  inputText.value = inputText.value.trim() || '继续修改：'
  setTimeout(() => inputRef.value?.focus(), 0)
}

async function cancelPendingPrd() {
  if (!pendingPrd.value) return
  markPrdConfirmed(pendingPrd.value)
  await clearCurrentPendingInteractionStatus()
  ElMessage.info('已取消本次 PRD 确认')
}

async function cancelPendingTestHandoff() {
  if (!pendingTestHandoff.value) return
  markTestHandoffHandled(pendingTestHandoff.value)
  await clearCurrentPendingInteractionStatus()
  ElMessage.info('已暂不进入测试')
}

function focusPrdPreview() {
  outputRef.value?.scrollTo({ top: outputRef.value.scrollHeight, behavior: 'smooth' })
}

function onLLMSelectVisibleChange(visible: boolean) {
  llmSelectOpen.value = visible
  loadLLMOptionsOnVisibleChange(visible)
}

async function handleConfirmPrd(payload: { remark: string; prd: unknown }) {
  const remark = payload.remark.trim()
  if (!sessionId.value || !props.fullCodePath || sending.value) {
    ElMessage.warning('当前会话还未准备好，暂时不能确认 PRD')
    return
  }
  let handoff
  try {
    handoff = await createWorkspaceHandoff({
      source_session_id: sessionId.value,
      full_code_path: props.fullCodePath,
      target_role: getPrdTargetRole(payload.prd),
      artifact_kind: 'agent_app_prd',
      artifact: payload.prd,
      remark,
      context_policy: 'artifact_only'
    })
  } catch (error: any) {
    ElMessage.error(error?.message || '创建交接会话失败')
    return
  }
  markPrdConfirmed(payload.prd)
  setMessages([])
  sessionId.value = handoff.session_id
  void sendTextToSession(
    handoff.session_id,
    handoff.content,
    handoff.display_content,
    { contextUsage: 'artifact', artifactKind: handoff.artifact_kind, resume: true }
  )
}

async function handleConfirmTestHandoff(payload: { artifact: unknown }) {
  if (!sessionId.value || !props.fullCodePath || sending.value) {
    ElMessage.warning('当前会话还未准备好，暂时不能进入测试')
    return
  }
  let handoff
  try {
    handoff = await createWorkspaceHandoff({
      source_session_id: sessionId.value,
      full_code_path: props.fullCodePath,
      target_role: getBuildTestTargetRole(payload.artifact),
      artifact_kind: getStageArtifactKind(payload.artifact, 'agent_app_build'),
      artifact: payload.artifact,
      remark: '',
      context_policy: 'artifact_only',
      display_content: '已构建成功，开始测试验证。'
    })
  } catch (error: any) {
    ElMessage.error(error?.message || '创建测试会话失败')
    return
  }
  markTestHandoffHandled(payload.artifact)
  setMessages([])
  sessionId.value = handoff.session_id
  void sendTextToSession(
    handoff.session_id,
    handoff.content,
    handoff.display_content,
    { contextUsage: 'artifact', artifactKind: handoff.artifact_kind, resume: true }
  )
}

function getPrdTargetRole(prd: unknown) {
  if (isPrdInteractionData(prd) && prd.interaction?.target_role_on_confirm) {
    return prd.interaction.target_role_on_confirm
  }
  return 'app_developer'
}

function getBuildTestTargetRole(artifact: unknown) {
  if (isBuildTestInteractionData(artifact) && artifact.interaction?.target_role_on_confirm) {
    return artifact.interaction.target_role_on_confirm
  }
  return 'qa_engineer'
}

function getStageArtifactKind(artifact: unknown, fallback: string) {
  if (artifact && typeof artifact === 'object') {
    const data = artifact as { interaction?: { artifact_kind?: string }, kind?: string }
    return data.interaction?.artifact_kind || data.kind || fallback
  }
  return fallback
}

watch(
  () => [props.visible, props.fullCodePath] as const,
  ([visible, fullCodePath]) => {
    if (visible && fullCodePath) {
      void loadMiniSessions()
      void loadGlobalSessions()
    }
  },
  { immediate: true }
)

useMiniWorkstationEffects({
  visible: computed(() => props.visible),
  maximized,
  messages,
  sending,
  sessionId,
  inputRef,
  outputRef,
  stopMiniPoll,
  loadMiniSessions
})
</script>

<style scoped>
.mini-ws {
  --mini-cyber-bg: #07111f;
  --mini-cyber-bg-strong: #0b1829;
  --mini-cyber-panel: rgba(8, 22, 38, 0.78);
  --mini-cyber-panel-soft: rgba(13, 34, 55, 0.58);
  --mini-cyber-line: rgba(40, 214, 255, 0.24);
  --mini-cyber-line-strong: rgba(78, 229, 255, 0.48);
  --mini-cyber-text: #d8f8ff;
  --mini-cyber-muted: rgba(184, 225, 235, 0.68);
  --mini-cyber-dim: rgba(143, 187, 204, 0.48);
  --mini-cyber-accent: #22d3ee;
  --mini-cyber-warm: #f6c76b;
  position: fixed;
  right: 24px;
  bottom: 80px;
  isolation: isolate;
  background:
    radial-gradient(circle at 12% -8%, rgba(44, 214, 255, 0.25), transparent 36%),
    radial-gradient(circle at 92% 8%, rgba(246, 199, 107, 0.15), transparent 30%),
    linear-gradient(145deg, rgba(4, 11, 24, 0.96), rgba(8, 22, 38, 0.94) 48%, rgba(5, 13, 24, 0.98));
  border: 1px solid var(--mini-cyber-line);
  border-radius: 18px;
  box-shadow:
    0 24px 70px rgba(0, 0, 0, 0.42),
    0 0 0 1px rgba(138, 232, 255, 0.08),
    0 0 44px rgba(34, 211, 238, 0.16);
  color: var(--mini-cyber-text);
  backdrop-filter: blur(22px) saturate(1.2);
  z-index: var(--aos-z-mini-workstation);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: left 0.3s ease, top 0.3s ease, width 0.3s ease, height 0.3s ease, max-height 0.3s ease, border-radius 0.3s ease, box-shadow 0.2s ease;
}
.mini-ws::before {
  content: '';
  position: absolute;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background:
    linear-gradient(rgba(80, 214, 255, 0.055) 1px, transparent 1px),
    linear-gradient(90deg, rgba(80, 214, 255, 0.045) 1px, transparent 1px),
    linear-gradient(115deg, transparent 0 44%, rgba(36, 221, 255, 0.12) 50%, transparent 56% 100%);
  background-size: 28px 28px, 28px 28px, 260% 100%;
  opacity: 0.72;
  animation: miniCyberSweep 9s linear infinite;
}
.mini-ws::after {
  content: '';
  position: absolute;
  inset: 1px;
  z-index: 2;
  pointer-events: none;
  border-radius: inherit;
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.12),
    inset 0 -24px 60px rgba(2, 7, 15, 0.38);
}
.mini-ws > * {
  position: relative;
  z-index: 1;
}
.mini-ws--maximized {
  border-radius: 0;
  box-shadow: none;
  border: none;
}
.mini-ws--sending {
  box-shadow:
    0 24px 76px rgba(0, 0, 0, 0.46),
    0 0 0 1px rgba(138, 232, 255, 0.1),
    0 0 58px rgba(34, 211, 238, 0.26);
}

.mini-ws:not(.mini-ws--maximized):not(.mini-ws--interaction-open):not(:hover):not(:focus-within) {
  border-radius: 14px;
  box-shadow:
    0 14px 38px rgba(0, 0, 0, 0.34),
    0 0 0 1px rgba(138, 232, 255, 0.06),
    0 0 24px rgba(34, 211, 238, 0.1);
}
/* ── Resize 手柄 ── */
.mini-resize-handle { position: absolute; z-index: 6; }
.mini-resize-n  { top: -3px; left: 6px; right: 6px; height: 6px; cursor: n-resize; }
.mini-resize-s  { bottom: -3px; left: 6px; right: 6px; height: 6px; cursor: s-resize; }
.mini-resize-e  { right: -3px; top: 6px; bottom: 6px; width: 6px; cursor: e-resize; }
.mini-resize-w  { left: -3px; top: 6px; bottom: 6px; width: 6px; cursor: w-resize; }
.mini-resize-ne { top: -3px; right: -3px; width: 10px; height: 10px; cursor: ne-resize; }
.mini-resize-nw { top: -3px; left: -3px; width: 10px; height: 10px; cursor: nw-resize; }
.mini-resize-se { bottom: -3px; right: -3px; width: 10px; height: 10px; cursor: se-resize; }
.mini-resize-sw { bottom: -3px; left: -3px; width: 10px; height: 10px; cursor: sw-resize; }

/* ── 标题栏 ── */
.mini-ws-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 12px;
  border-bottom: 1px solid rgba(96, 231, 255, 0.16);
  cursor: move;
  user-select: none;
  background:
    linear-gradient(90deg, rgba(10, 29, 49, 0.92), rgba(7, 18, 33, 0.62)),
    linear-gradient(180deg, rgba(255, 255, 255, 0.08), transparent);
  flex-shrink: 0;
  box-shadow: inset 0 -1px 0 rgba(255, 255, 255, 0.04);
  transition: padding 0.2s ease, border-color 0.2s ease;
}
.mini-ws:not(.mini-ws--maximized):not(.mini-ws--interaction-open):not(:hover):not(:focus-within) .mini-ws-header {
  padding: 8px 10px;
  border-bottom-color: rgba(96, 231, 255, 0.08);
}
.mini-ws--maximized .mini-ws-header {
  cursor: default;
}
.mini-ws-title {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--mini-cyber-accent);
}
.mini-ws-title-orb {
  position: relative;
  width: 25px;
  height: 25px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(65, 230, 255, 0.38);
  border-radius: 50%;
  background:
    radial-gradient(circle at 50% 48%, rgba(34, 211, 238, 0.36), transparent 42%),
    conic-gradient(from 130deg, rgba(34, 211, 238, 0.18), rgba(246, 199, 107, 0.32), rgba(34, 211, 238, 0.18));
  box-shadow: 0 0 18px rgba(34, 211, 238, 0.28), inset 0 0 12px rgba(34, 211, 238, 0.18);
}
.mini-ws-title-orb::after {
  content: '';
  position: absolute;
  inset: -4px;
  border-radius: inherit;
  border: 1px solid rgba(34, 211, 238, 0.14);
}
.mini-ws-title-label {
  font-size: 10px;
  line-height: 1;
  letter-spacing: 0.16em;
  color: rgba(216, 248, 255, 0.74);
  transition: opacity 0.16s ease, width 0.16s ease;
}
.mini-ws:not(.mini-ws--maximized):not(.mini-ws--interaction-open):not(:hover):not(:focus-within) .mini-ws-title-label {
  width: 0;
  opacity: 0;
  overflow: hidden;
}
.mini-ws-dir-name {
  flex: 1;
  text-align: center;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: var(--mini-cyber-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 0 4px;
  text-shadow: 0 0 12px rgba(34, 211, 238, 0.22);
}
.mini-ws-header-actions {
  flex-shrink: 0;
  display: flex;
  gap: 4px;
  align-items: center;
  transition: opacity 0.16s ease, width 0.16s ease, margin 0.16s ease;
}
.mini-ws:not(.mini-ws--maximized):not(.mini-ws--interaction-open):not(:hover):not(:focus-within) .mini-ws-header-actions {
  width: 0;
  margin-left: -4px;
  opacity: 0;
  overflow: hidden;
  pointer-events: none;
}
.mini-ws-header-actions :deep(.el-button) {
  min-height: 24px;
  color: var(--mini-cyber-muted);
  border-radius: 8px;
}
.mini-ws-header-actions :deep(.el-button:hover) {
  color: #ffffff;
  background: rgba(34, 211, 238, 0.12);
  box-shadow: inset 0 0 0 1px rgba(34, 211, 238, 0.18);
}
.mini-header-files-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.mini-header-files-count {
  min-width: 17px;
  height: 17px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 5px;
  border: 1px solid rgba(34, 211, 238, 0.36);
  border-radius: 999px;
  background: rgba(34, 211, 238, 0.12);
  color: var(--mini-cyber-accent);
  font-size: 10px;
  font-weight: 800;
}

.mini-settings-btn {
  width: 32px;
  height: 32px;
  border: 1px solid rgba(96, 231, 255, 0.2);
  border-radius: 10px;
  color: var(--mini-cyber-accent, #22d3ee);
  background: rgba(34, 211, 238, 0.08);
}
.mini-settings-btn:hover,
.mini-settings-btn:focus {
  color: #ffffff;
  background: rgba(34, 211, 238, 0.16);
  box-shadow: 0 0 18px rgba(34, 211, 238, 0.16);
}

/* ── 主体区域（sidebar + output） ── */
.mini-ws-body {
  position: relative;
  flex: 1;
  display: flex;
  overflow: hidden;
  min-height: 0;
}

/* ── 最大化右侧文件面板 ── */
.mini-file-sidebar {
  width: 220px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-left: 1px solid rgba(96, 231, 255, 0.14);
  background: linear-gradient(180deg, rgba(8, 21, 37, 0.82), rgba(4, 12, 24, 0.72));
}
.mini-file-sidebar-header {
  padding: 10px 12px;
  font-size: 13px;
  font-weight: 700;
  color: var(--mini-cyber-text);
  border-bottom: 1px solid rgba(96, 231, 255, 0.14);
  background: rgba(34, 211, 238, 0.06);
  flex-shrink: 0;
}

/* ── SSE 输出区 ── */
.mini-ws-output {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px 24px;
  min-height: 0;
  font-size: 12px;
  line-height: 1.6;
  background:
    radial-gradient(circle at 90% 8%, rgba(34, 211, 238, 0.1), transparent 26%),
    linear-gradient(180deg, rgba(2, 8, 18, 0.18), rgba(2, 8, 18, 0.42));
  scrollbar-color: rgba(34, 211, 238, 0.36) transparent;
  transition: padding 0.2s ease, font-size 0.2s ease;
}
.mini-ws:not(.mini-ws--maximized):not(.mini-ws--interaction-open):not(:hover):not(:focus-within) .mini-ws-output {
  padding: 6px 10px 10px;
  font-size: 11px;
}
.mini-ws--maximized .mini-ws-output {
  padding: 16px 24px;
  font-size: 13px;
}
.mini-ws-output::-webkit-scrollbar {
  width: 8px;
}
.mini-ws-output::-webkit-scrollbar-thumb {
  background: rgba(34, 211, 238, 0.26);
  border-radius: 999px;
}

/* ── 拖拽上传遮罩 ── */
.mini-ws-drop-overlay {
  position: absolute;
  inset: 0;
  z-index: 10;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background:
    radial-gradient(circle at center, rgba(34, 211, 238, 0.2), rgba(2, 8, 18, 0.82)),
    repeating-linear-gradient(90deg, rgba(96, 231, 255, 0.08) 0 1px, transparent 1px 16px);
  border: 1px dashed rgba(96, 231, 255, 0.78);
  border-radius: 12px;
  color: #d8f8ff;
  font-size: 14px;
  font-weight: 700;
  letter-spacing: 0.08em;
  backdrop-filter: blur(8px);
  pointer-events: none;
  box-shadow: inset 0 0 48px rgba(34, 211, 238, 0.2);
}

/* ── 弹出动画 ── */
.mini-ws-pop-enter-active {
  transition: all 0.28s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.mini-ws-pop-leave-active {
  transition: all 0.2s ease-in;
}
.mini-ws-pop-enter-from {
  transform: translateY(24px) scale(0.94) rotateX(4deg);
  opacity: 0;
}
.mini-ws-pop-leave-to {
  transform: translateY(10px) scale(0.97);
  opacity: 0;
}

@keyframes miniCyberSweep {
  0% { background-position: 0 0, 0 0, 140% 0; }
  100% { background-position: 0 28px, 28px 0, -140% 0; }
}

@keyframes miniStatePulse {
  0%, 100% { opacity: 0.72; transform: scale(1); box-shadow: 0 0 10px rgba(34, 211, 238, 0.68); }
  50% { opacity: 1; transform: scale(1.4); box-shadow: 0 0 18px rgba(34, 211, 238, 1); }
}

/* Bottom floating workbench layout */
.mini-ws {
  --mini-cyber-bg: #070b12;
  --mini-cyber-bg-strong: #11192d;
  --mini-cyber-panel: rgba(12, 18, 32, 0.82);
  --mini-cyber-panel-soft: rgba(17, 25, 45, 0.72);
  --mini-cyber-line: rgba(128, 151, 198, 0.22);
  --mini-cyber-line-strong: rgba(104, 119, 255, 0.48);
  --mini-cyber-text: #edf4ff;
  --mini-cyber-muted: #8e9fbb;
  --mini-cyber-dim: #61718c;
  --mini-cyber-accent: #37a3ff;
  --mini-cyber-violet: #776bff;
  --mini-cyber-green: #2bd59f;
  --mini-cyber-warm: #f6bd4d;
  --mini-cyber-red: #ff6d7e;
  inset: 0;
  width: auto;
  height: auto;
  border: 0;
  border-radius: 0;
  overflow: visible;
  background: transparent;
  box-shadow: none;
  backdrop-filter: none;
  pointer-events: none;
}

.mini-ws::before,
.mini-ws::after {
  content: none;
}

.mini-ws--sending {
  box-shadow: none;
}

.mini-ws:not(.mini-ws--maximized):not(.mini-ws--interaction-open):not(:hover):not(:focus-within) {
  border-radius: 0;
  box-shadow: none;
}

.mini-workspace-backdrop {
  position: fixed;
  inset: 0;
  z-index: 0;
  overflow: hidden;
  pointer-events: none;
  background:
    radial-gradient(circle at 50% 86%, rgba(55, 163, 255, 0.18), transparent 34%),
    radial-gradient(circle at 16% 14%, rgba(119, 107, 255, 0.12), transparent 28%),
    linear-gradient(180deg, rgba(2, 5, 11, 0.34), rgba(2, 5, 11, 0.58));
  backdrop-filter: blur(9px) saturate(72%);
  -webkit-backdrop-filter: blur(9px) saturate(72%);
}

.mini-workspace-backdrop::after {
  content: '';
  position: absolute;
  left: 32px;
  right: 32px;
  bottom: calc(18px + var(--mini-stack-offset, 0px));
  height: min(220px, 32vh);
  border-radius: 28px 28px 0 0;
  background:
    linear-gradient(180deg, rgba(7, 11, 18, 0), rgba(7, 11, 18, 0.78)),
    radial-gradient(circle at 50% 100%, rgba(55, 163, 255, 0.2), transparent 58%);
  box-shadow: 0 -28px 72px rgba(2, 5, 11, 0.34);
}

.mini-ws--maximized .mini-workspace-backdrop {
  background:
    radial-gradient(circle at 50% 88%, rgba(55, 163, 255, 0.14), transparent 38%),
    linear-gradient(180deg, rgba(2, 5, 11, 0.44), rgba(2, 5, 11, 0.68));
  backdrop-filter: blur(12px) saturate(68%);
  -webkit-backdrop-filter: blur(12px) saturate(68%);
}

.mini-shell {
  position: absolute;
  left: 42px;
  right: 42px;
  bottom: calc(24px + var(--mini-stack-offset, 0px));
  display: flex;
  flex-direction: column;
  gap: 0;
  color: var(--mini-cyber-text);
  pointer-events: auto;
  transition: left 0.18s ease, right 0.18s ease, bottom 0.18s ease;
}

.mini-current-output,
.mini-shell :deep(.mini-ws-input) {
  border: 1px solid var(--mini-cyber-line);
  background:
    linear-gradient(180deg, rgba(12, 18, 32, 0.84), rgba(8, 12, 22, 0.68)),
    rgba(8, 12, 22, 0.72);
  box-shadow: 0 24px 70px rgba(0, 0, 0, 0.42), inset 0 1px 0 rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(24px) saturate(140%);
}

.mini-current-output {
  height: min(720px, calc(100vh - 180px));
  min-height: min(520px, calc(100vh - 180px));
  max-height: none;
  display: flex;
  flex-direction: column;
  margin: 0 14px 8px;
  padding: 12px 14px;
  border-color: rgba(104, 119, 255, 0.28);
  border-radius: 12px;
  overflow: hidden;
  transition: height 0.18s ease, min-height 0.18s ease;
}

.mini-ws--maximized .mini-current-output {
  height: min(980px, calc(100vh - 184px));
  min-height: min(760px, calc(100vh - 184px));
  max-height: none;
}

.mini-current-layout {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 230px minmax(0, 1fr) 260px;
  gap: 12px;
  align-items: stretch;
}

.mini-ws--maximized .mini-current-layout {
  grid-template-columns: 280px minmax(0, 1fr) 300px;
}

.mini-current-meta {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: auto auto auto auto auto minmax(0, 1fr) auto;
  gap: 10px;
  padding-right: 12px;
  border-right: 1px solid rgba(130, 153, 190, 0.18);
  color: #b9c9e4;
  font-size: 12px;
  overflow: hidden;
}

.mini-current-session-head {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 3px 2px 0;
}

.mini-current-session-head div {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.mini-current-session-head strong,
.mini-current-session-head span {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-current-session-head strong {
  color: #88d6ff;
  font-size: 12px;
  font-weight: 850;
}

.mini-current-session-head span {
  color: var(--mini-cyber-muted);
  font-size: 11px;
}

.mini-current-session-head em {
  min-width: 24px;
  height: 24px;
  display: inline-grid;
  place-items: center;
  border-radius: 8px;
  background: rgba(83, 174, 255, 0.14);
  color: #8ed0ff;
  font-size: 11px;
  font-style: normal;
  font-weight: 900;
}

.mini-current-session-list {
  min-height: 0;
  overflow: auto;
  display: grid;
  align-content: start;
  gap: 8px;
  padding: 1px 2px 3px 0;
  scrollbar-color: rgba(83, 174, 255, 0.24) transparent;
}

.mini-current-session-row {
  --mini-active-glow: rgba(55, 163, 255, 0.26);
  --mini-active-halo: rgba(55, 163, 255, 0.12);
  position: relative;
  width: 100%;
  min-width: 0;
  min-height: 52px;
  display: grid;
  grid-template-columns: 10px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  padding: 8px 9px;
  border: 1px solid rgba(126, 151, 197, 0.18);
  border-radius: 10px;
  background: rgba(17, 25, 45, 0.54);
  color: #d7e5fa;
  text-align: left;
  cursor: pointer;
  transition: background 0.18s ease, border-color 0.18s ease, box-shadow 0.22s ease;
}

.mini-current-session-row:hover {
  border-color: rgba(83, 174, 255, 0.38);
  background: rgba(24, 51, 83, 0.48);
}

.mini-current-session-row.is-running {
  border-color: rgba(43, 213, 159, 0.28);
  background: rgba(21, 54, 50, 0.42);
}

.mini-current-session-row.is-waiting {
  border-color: rgba(246, 189, 77, 0.3);
  background: rgba(58, 45, 24, 0.46);
}

.mini-current-session-row.is-output {
  border-color: rgba(55, 163, 255, 0.3);
  background: rgba(24, 48, 77, 0.46);
}

.mini-current-session-row.is-done {
  border-color: rgba(119, 107, 255, 0.28);
  background: rgba(41, 38, 76, 0.46);
}

.mini-current-session-row.is-cancelled {
  border-color: rgba(142, 159, 187, 0.24);
  background: rgba(41, 48, 64, 0.46);
}

.mini-current-session-row.is-failed {
  border-color: rgba(255, 108, 108, 0.34);
  background: rgba(74, 30, 38, 0.46);
}

.mini-current-session-copy,
.mini-current-session-title,
.mini-current-session-sub {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-current-session-title {
  color: #d7e5fa;
  font-size: 12px;
  font-weight: 820;
  line-height: 1.2;
}

.mini-current-session-sub {
  margin-top: 4px;
  color: #8798b5;
  font-size: 11px;
  line-height: 1.15;
}

.mini-current-stream {
  min-width: 0;
  min-height: 0;
}

.mini-icon-action {
  min-width: 28px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: 1px solid rgba(124, 146, 189, 0.24);
  border-radius: 7px;
  background: rgba(30, 42, 68, 0.68);
  color: #d7e5fa;
}

.mini-icon-action:hover {
  border-color: rgba(83, 174, 255, 0.42);
  background: rgba(34, 113, 205, 0.18);
  color: #ffffff;
}

.mini-queue-chip {
  width: fit-content;
  max-width: 100%;
  height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 7px;
  border: 1px solid rgba(246, 189, 77, 0.28);
  border-radius: 999px;
  background: rgba(246, 189, 77, 0.12);
  color: #ffd78d;
  font-size: 11px;
  font-weight: 800;
}

.mini-ws-output {
  min-height: 0;
  height: 100%;
  overflow: auto;
  padding: 12px 14px;
  border: 1px solid rgba(130, 153, 190, 0.16);
  border-radius: 12px;
  background: rgba(10, 16, 29, 0.32);
  color: #d7e5fa;
  font-size: 13px;
  line-height: 18px;
  scrollbar-color: rgba(83, 174, 255, 0.3) transparent;
}

.mini-ws:not(.mini-ws--maximized):not(.mini-ws--interaction-open):not(:hover):not(:focus-within) .mini-ws-output {
  padding: 12px 14px;
  font-size: 13px;
}

.mini-ws--maximized .mini-ws-output {
  height: 100%;
  overflow: auto;
  padding: 14px 16px;
  border: 1px solid rgba(130, 153, 190, 0.16);
  border-radius: 12px;
  background: rgba(10, 16, 29, 0.36);
  font-size: 13px;
}

.mini-ws:not(.mini-ws--maximized) .mini-ws-output :deep(.mini-ws-empty) {
  min-height: 36px;
  justify-content: flex-start;
  font-size: 12px;
  letter-spacing: 0;
  text-transform: none;
}

.mini-ws:not(.mini-ws--maximized) .mini-ws-output :deep(.mini-msg) {
  margin-bottom: 12px;
}

.mini-current-session-row.is-running {
  --mini-active-glow: rgba(43, 213, 159, 0.34);
  --mini-active-halo: rgba(43, 213, 159, 0.16);
}

.mini-current-session-row.is-waiting {
  --mini-active-glow: rgba(246, 189, 77, 0.34);
  --mini-active-halo: rgba(246, 189, 77, 0.16);
}

.mini-current-session-row.is-output {
  --mini-active-glow: rgba(55, 163, 255, 0.34);
  --mini-active-halo: rgba(55, 163, 255, 0.16);
}

.mini-current-session-row.is-done {
  --mini-active-glow: rgba(119, 107, 255, 0.34);
  --mini-active-halo: rgba(119, 107, 255, 0.16);
}

.mini-current-session-row.is-cancelled {
  --mini-active-glow: rgba(142, 159, 187, 0.28);
  --mini-active-halo: rgba(142, 159, 187, 0.12);
}

.mini-current-session-row.is-failed {
  --mini-active-glow: rgba(255, 107, 107, 0.34);
  --mini-active-halo: rgba(255, 107, 107, 0.16);
}

.mini-current-session-row.active {
  z-index: 1;
  box-shadow:
    0 0 14px 2px var(--mini-active-glow),
    0 0 38px 8px var(--mini-active-halo),
    0 12px 32px rgba(2, 5, 11, 0.22);
}

.mini-status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--mini-cyber-accent);
  box-shadow: 0 0 16px rgba(55, 163, 255, 0.58);
}

.mini-status-dot.is-running {
  background: var(--mini-cyber-green);
  box-shadow: 0 0 16px rgba(43, 213, 159, 0.6);
}

.mini-status-dot.is-waiting {
  background: var(--mini-cyber-warm);
  box-shadow: 0 0 16px rgba(246, 189, 77, 0.6);
}

.mini-status-dot.is-done,
.mini-status-dot.is-cancelled {
  background: var(--mini-cyber-violet);
  box-shadow: 0 0 16px rgba(119, 107, 255, 0.55);
}

.mini-status-dot.is-failed {
  background: #ff6b6b;
  box-shadow: 0 0 16px rgba(255, 107, 107, 0.58);
}

.mini-status-dot.is-active,
.mini-status-dot.is-output {
  background: #37a3ff;
  box-shadow: 0 0 16px rgba(55, 163, 255, 0.58);
}

.mini-settings-btn {
  width: 42px;
  height: 42px;
  border-color: rgba(128, 151, 198, 0.22);
  border-radius: 8px;
  color: #8ed0ff;
  background: rgba(30, 42, 68, 0.72);
}

.mini-settings-btn:hover,
.mini-settings-btn:focus {
  background: rgba(55, 163, 255, 0.16);
  box-shadow: 0 0 18px rgba(55, 163, 255, 0.14);
}

.mini-ws-drop-overlay {
  inset: auto 86px calc(24px + var(--mini-stack-offset, 0px)) max(162px, 12vw);
  height: 210px;
  pointer-events: none;
  border-color: rgba(83, 174, 255, 0.74);
  background:
    radial-gradient(circle at center, rgba(55, 163, 255, 0.2), rgba(2, 5, 11, 0.82)),
    repeating-linear-gradient(90deg, rgba(128, 151, 198, 0.08) 0 1px, transparent 1px 16px);
}

.mini-ws--maximized .mini-ws-drop-overlay {
  left: 42px;
  right: 42px;
}

@media (max-width: 1180px) {
  .mini-shell {
    left: 34px;
    right: 34px;
  }

  .mini-current-layout {
    grid-template-columns: 210px minmax(0, 1fr) 236px;
  }
}

@media (max-width: 820px) {
  .mini-shell,
  .mini-ws--maximized .mini-shell {
    left: 12px;
    right: 12px;
    bottom: 12px;
  }

  .mini-current-layout,
  .mini-ws--maximized .mini-current-layout {
    grid-template-columns: 1fr;
  }

  .mini-current-meta {
    display: none;
  }

}

/* Fullscreen workbench shell */
.mini-ws,
.mini-ws--maximized {
  top: 18px;
  right: 18px;
  bottom: 18px;
  left: 18px;
  width: auto;
  height: auto;
  min-height: 0;
  overflow: visible;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  pointer-events: none;
}

.mini-ws--compact {
  top: 88px;
  left: auto;
  width: clamp(560px, 44vw, 680px);
}

.mini-workspace-backdrop,
.mini-workspace-backdrop::after,
.mini-ws--maximized .mini-workspace-backdrop {
  display: none;
}

.mini-shell,
.mini-ws--maximized .mini-shell {
  position: relative;
  inset: auto;
  width: 100%;
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto auto;
  overflow: hidden;
  border: 1px solid rgba(124, 146, 189, 0.22);
  border-radius: 12px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), transparent 120px),
    rgba(16, 22, 34, 0.96);
  box-shadow:
    0 24px 72px rgba(8, 14, 24, 0.28),
    0 0 0 1px rgba(255, 255, 255, 0.05);
  color: var(--mini-cyber-text);
  pointer-events: auto;
  backdrop-filter: blur(20px) saturate(120%);
  -webkit-backdrop-filter: blur(20px) saturate(120%);
}

.mini-drawer-head {
  min-height: 68px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 12px 14px 12px 16px;
  border-bottom: 1px solid rgba(124, 146, 189, 0.18);
  background: rgba(22, 30, 46, 0.78);
}

.mini-drawer-title {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.mini-drawer-title strong {
  color: #f5f8ff;
  font-size: 15px;
  font-weight: 850;
  line-height: 1.2;
}

.mini-drawer-title span {
  min-width: 0;
  overflow: hidden;
  color: #8e9fbb;
  font-size: 12px;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-drawer-actions {
  display: flex;
  align-items: center;
  gap: 7px;
}

.mini-drawer-primary-action,
.mini-drawer-secondary-action,
.mini-drawer-icon-action {
  height: 34px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid rgba(124, 146, 189, 0.24);
  border-radius: 8px;
  background: rgba(30, 42, 68, 0.68);
  color: #d7e5fa;
  font-size: 12px;
  font-weight: 760;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.16s ease, border-color 0.16s ease, color 0.16s ease;
}

.mini-drawer-primary-action {
  padding: 0 11px;
  border-color: rgba(43, 213, 159, 0.35);
  background: rgba(23, 78, 61, 0.42);
  color: #a5f7d5;
}

.mini-drawer-secondary-action {
  padding: 0 11px;
  border-color: rgba(83, 174, 255, 0.32);
  background: rgba(29, 73, 118, 0.36);
  color: #a8d7ff;
}

.mini-drawer-icon-action {
  width: 34px;
  padding: 0;
}

.mini-drawer-primary-action:hover,
.mini-drawer-secondary-action:hover,
.mini-drawer-icon-action:hover {
  border-color: rgba(83, 174, 255, 0.55);
  background: rgba(37, 56, 88, 0.84);
  color: #ffffff;
}

.mini-drawer-secondary-action:disabled {
  cursor: not-allowed;
  opacity: 0.42;
}

.mini-current-output,
.mini-ws--maximized .mini-current-output {
  height: auto;
  min-height: 0;
  max-height: none;
  margin: 0;
  padding: 12px;
  overflow: hidden;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  backdrop-filter: none;
}

.mini-current-layout {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 184px minmax(0, 1fr);
  grid-template-rows: minmax(0, 1fr) auto;
  gap: 12px;
}

.mini-ws--maximized .mini-current-layout {
  grid-template-columns: 280px minmax(0, 1fr);
  grid-template-rows: minmax(0, 1fr) auto;
}

.mini-ws--maximized .mini-current-layout.is-artifact-open {
  grid-template-columns: 280px minmax(0, 1fr) 320px;
  grid-template-rows: minmax(0, 1fr);
}

.mini-current-meta {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: auto auto auto auto auto minmax(0, 1fr) auto;
  gap: 10px;
  padding: 0 12px 0 0;
  border-right: 1px solid rgba(124, 146, 189, 0.16);
  overflow: hidden;
}

.mini-current-session-head {
  padding: 0;
}

.mini-current-session-head strong {
  color: #eef5ff;
}

.mini-current-context-switch {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 6px;
  padding: 9px 10px;
  border: 1px solid rgba(246, 199, 107, 0.2);
  border-radius: 8px;
  background: rgba(62, 47, 18, 0.22);
}

.mini-current-context-switch span {
  color: rgba(246, 217, 150, 0.72);
  font-size: 10px;
  font-weight: 760;
}

.mini-current-context-switch strong {
  min-width: 0;
  color: #ffe4a3;
  font-size: 12px;
  font-weight: 850;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-current-context-switch button {
  width: 100%;
  height: 28px;
  border: 1px solid rgba(246, 199, 107, 0.34);
  border-radius: 7px;
  background: rgba(246, 199, 107, 0.12);
  color: #ffdda0;
  font-size: 11px;
  font-weight: 800;
  cursor: pointer;
}

.mini-current-context-switch button:hover {
  border-color: rgba(246, 199, 107, 0.56);
  background: rgba(246, 199, 107, 0.18);
  color: #fff2d6;
}

.mini-drawer-scope-tabs,
.mini-drawer-session-filters {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.mini-drawer-scope-tabs button,
.mini-drawer-session-filters button {
  min-width: 0;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(124, 146, 189, 0.2);
  border-radius: 7px;
  background: rgba(30, 42, 68, 0.46);
  color: #9eadc8;
  font-size: 11px;
  font-weight: 720;
  cursor: pointer;
}

.mini-drawer-scope-tabs button {
  flex: 1 1 0;
}

.mini-drawer-session-filters {
  flex-wrap: wrap;
}

.mini-drawer-session-filters button {
  flex: 1 1 calc(50% - 4px);
}

.mini-drawer-scope-tabs button.active,
.mini-drawer-session-filters button.active {
  border-color: rgba(83, 174, 255, 0.45);
  background: rgba(34, 113, 205, 0.2);
  color: #9bd4ff;
}

.mini-drawer-session-search {
  height: 32px;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  border: 1px solid rgba(124, 146, 189, 0.18);
  border-radius: 8px;
  background: rgba(10, 16, 29, 0.36);
  color: #8e9fbb;
}

.mini-drawer-session-search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  color: #d7e5fa;
  font-size: 12px;
}

.mini-drawer-session-search input::placeholder {
  color: #687996;
}

.mini-current-session-list {
  padding: 0 2px 3px 0;
}

.mini-current-session-row {
  min-height: 48px;
  padding: 8px;
  border-radius: 8px;
}

.mini-current-stream {
  min-width: 0;
  min-height: 0;
  display: flex;
}

.mini-ws-output,
.mini-ws--maximized .mini-ws-output {
  width: 100%;
  height: 100%;
  min-height: 0;
  padding: 13px;
  border: 1px solid rgba(124, 146, 189, 0.16);
  border-radius: 10px;
  background: rgba(10, 16, 29, 0.34);
}

.mini-artifact-drawer {
  grid-column: 1 / -1;
  min-width: 0;
  display: grid;
  gap: 8px;
  overflow: hidden;
  padding: 8px 0 0;
  border-top: 1px solid rgba(124, 146, 189, 0.16);
}

.mini-artifact-toggle {
  width: 100%;
  min-width: 0;
  height: 34px;
  display: grid;
  grid-template-columns: auto auto minmax(0, 1fr) 18px;
  align-items: center;
  gap: 7px;
  padding: 0 10px;
  border: 1px solid rgba(83, 174, 255, 0.22);
  border-radius: 8px;
  background: rgba(30, 42, 68, 0.5);
  color: #c8d8ef;
  font-size: 12px;
  font-weight: 760;
  cursor: pointer;
  transition: background 0.16s ease, border-color 0.16s ease, color 0.16s ease;
}

.mini-artifact-toggle:hover {
  border-color: rgba(83, 174, 255, 0.44);
  background: rgba(34, 113, 205, 0.18);
  color: #ffffff;
}

.mini-artifact-toggle strong {
  color: #8ed0ff;
  font-size: 12px;
  white-space: nowrap;
}

.mini-artifact-toggle em {
  min-width: 0;
  overflow: hidden;
  color: #8e9fbb;
  font-size: 11px;
  font-style: normal;
  text-align: right;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-artifact-drawer .mini-artifact-panel {
  max-height: 116px;
  overflow: auto;
  padding: 0;
  border: 0;
}

.mini-ws--maximized .mini-artifact-drawer.is-open {
  grid-column: 3;
  grid-row: 1;
  max-height: none;
  grid-template-rows: auto minmax(0, 1fr);
  padding: 0 0 0 12px;
  border-top: 0;
  border-left: 1px solid rgba(124, 146, 189, 0.16);
}

.mini-ws--maximized .mini-artifact-drawer.is-open .mini-artifact-panel {
  height: 100%;
  max-height: none;
}

.mini-shell :deep(.mini-ws-input) {
  margin: 0;
  border-width: 1px 0 0;
  border-radius: 0;
  background: rgba(22, 30, 46, 0.78);
  box-shadow: none;
}

.mini-ws-files {
  margin: 0;
  padding: 8px 12px;
  border-top: 1px solid rgba(124, 146, 189, 0.14);
  background: rgba(22, 30, 46, 0.78);
}

.mini-ws-drop-overlay,
.mini-ws--maximized .mini-ws-drop-overlay {
  inset: 12px;
  height: auto;
  border-radius: 12px;
}

@media (max-width: 1180px) {
  .mini-ws,
  .mini-ws--maximized {
    left: 18px;
    width: auto;
  }

  .mini-ws--compact {
    left: auto;
    width: min(720px, calc(100vw - 330px));
  }

  .mini-current-layout,
  .mini-ws--maximized .mini-current-layout,
  .mini-ws--maximized .mini-current-layout.is-artifact-open {
    grid-template-columns: 180px minmax(0, 1fr);
    grid-template-rows: minmax(0, 1fr) auto;
  }

  .mini-ws--maximized .mini-artifact-drawer.is-open {
    grid-column: 1 / -1;
    grid-row: auto;
    max-height: 150px;
    padding: 8px 0 0;
    border-top: 1px solid rgba(124, 146, 189, 0.16);
    border-left: 0;
  }

  .mini-ws--maximized .mini-artifact-drawer.is-open .mini-artifact-panel {
    height: auto;
    max-height: 108px;
  }
}

@media (max-width: 820px) {
  .mini-ws,
  .mini-ws--maximized {
    inset: 0;
    width: auto;
  }

  .mini-shell,
  .mini-ws--maximized .mini-shell {
    height: 100%;
    border-radius: 0;
  }

  .mini-drawer-head {
    grid-template-columns: 1fr;
  }

  .mini-drawer-actions {
    width: 100%;
    min-width: 0;
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 34px 34px;
    gap: 7px;
  }

  .mini-drawer-primary-action,
  .mini-drawer-secondary-action {
    width: 100%;
    min-width: 0;
    padding: 0 8px;
  }

  .mini-drawer-primary-action span,
  .mini-drawer-secondary-action span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .mini-current-layout,
  .mini-ws--maximized .mini-current-layout,
  .mini-ws--maximized .mini-current-layout.is-artifact-open {
    grid-template-columns: 1fr;
    grid-template-rows: auto minmax(0, 1fr) auto;
  }

  .mini-current-meta {
    display: grid;
    width: 100%;
    max-width: 100%;
    box-sizing: border-box;
    max-height: 260px;
    padding: 0 0 10px;
    border-right: 0;
    border-bottom: 1px solid rgba(124, 146, 189, 0.16);
  }

  .mini-shell :deep(.mini-ws-input) {
    grid-template-columns: 42px minmax(0, 1fr);
    grid-template-rows: auto auto;
    gap: 8px;
    padding: 6px 8px;
  }

  .mini-shell :deep(.mini-composer-left-actions) {
    width: 42px;
    min-width: 0;
  }

  .mini-shell :deep(.mini-path-pill) {
    display: none;
  }

  .mini-shell :deep(.mini-input-wrap) {
    min-width: 0;
    grid-template-columns: minmax(0, 1fr);
  }

  .mini-shell :deep(.mini-action-stack) {
    grid-column: 1 / -1;
    width: 100%;
    min-width: 0;
    display: grid;
    grid-template-columns: minmax(0, 116px) auto;
    align-items: center;
    justify-content: space-between;
  }

  .mini-shell :deep(.mini-ws-model-select) {
    width: 116px;
  }

  .mini-shell :deep(.mini-action-row) {
    min-width: 0;
    justify-content: flex-end;
  }

  .mini-shell :deep(.mini-send-btn),
  .mini-shell :deep(.mini-stop-btn) {
    min-width: 88px;
  }

  .mini-shell :deep(.mini-hide-btn) {
    width: 38px;
    min-width: 38px;
  }
}
</style>

<style lang="scss">
/* 文件下拉 popper：去掉默认内边距（z-index 已放在 main.css 全局，确保高于迷你窗） */
.mini-files-dropdown-popper.el-dropdown__popper {
  padding: 0;
  border: none;
  background: transparent;
  box-shadow: none;
}
.mini-files-dropdown-popper.el-dropdown__popper .el-popper__arrow::before {
  background: rgba(8, 22, 38, 0.98);
  border-color: rgba(96, 231, 255, 0.22);
}
</style>
