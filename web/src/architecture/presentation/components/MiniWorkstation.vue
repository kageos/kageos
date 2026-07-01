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
            <MiniWorkstationResourceIdentity
              class="mini-drawer-resource"
              :name="resourceDisplayName"
              :full-code-path="fullCodePath"
              :resource-type="resourceType"
              :resource-template-type="resourceTemplateType"
            />
            <span :title="fullCodePath">{{ fullCodePath || displayPath }}</span>
          </div>
          <div class="mini-drawer-actions">
            <button
              v-if="panelHasContent"
              type="button"
              class="mini-drawer-secondary-action"
              :class="{ 'is-active': artifactPanelExpanded }"
              :title="artifactPanelExpanded ? t('miniWorkstation.collapse') : t('miniWorkstation.expandArtifactsTitle')"
              @click="toggleArtifactPanel"
            >
              <el-icon><DataBoard /></el-icon>
              <span>{{ t('miniWorkstation.artifact') }} ({{ artifactToggleCount }})</span>
            </button>
            <button
              type="button"
              class="mini-drawer-primary-action"
              :title="t('miniWorkstation.newSessionTitle')"
              @click="startNewSession"
            >
              <el-icon><Plus /></el-icon>
              <span>{{ t('miniWorkstation.newSession') }}</span>
            </button>
            <button
              type="button"
              class="mini-drawer-secondary-action"
              :disabled="!lastDrawerSession"
              :title="lastDrawerSession ? t('miniWorkstation.openSessionTitle', { title: getSessionTitle(lastDrawerSession) }) : t('miniWorkstation.noPreviousSession')"
              @click="openLastDrawerSession"
            >
              <el-icon><Clock /></el-icon>
              <span>{{ t('miniWorkstation.openPrevious') }}</span>
            </button>
            <button
              type="button"
              class="mini-drawer-icon-action"
              :title="maximized ? t('miniWorkstation.collapsePanel') : t('miniWorkstation.maximizePanel')"
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
              :title="t('miniWorkstation.closePanelTitle', { shortcut: toggleShortcutLabel || t('miniWorkstation.shortcut') })"
              @click="hideWorkstation"
            >
              <el-icon><Close /></el-icon>
            </button>
          </div>
        </header>

        <section class="mini-current-output">
          <div :class="['mini-current-layout', { 'is-artifact-open': artifactPanelExpanded }]">
            <MiniWorkstationSessionPanel
              :full-code-path="fullCodePath"
              :dir-label="resourceDisplayName"
              :sessions="drawerSessionList"
              :active-session-id="sessionId"
              :scope="drawerSessionScope"
              :search-keyword="sessionSearchKeyword"
              :filter="sessionFilter"
              :filters="sessionFilters"
              :queued-count="queuedCount"
              :has-different-context="hasDifferentCurrentContext"
              :current-context-name="currentContextName"
              :current-context-path="normalizedCurrentContextPath"
              :get-session-status-class="getSessionStatusClass"
              :get-session-title="getSessionTitle"
              :get-session-status-label="getSessionStatusLabel"
              :format-relative-time="formatRelativeTime"
              @update:search-keyword="sessionSearchKeyword = $event"
              @update:filter="sessionFilter = $event"
              @select="handleDrawerSessionSelect"
              @new-session="startNewSession"
              @scope-change="setDrawerSessionScope"
              @context-new-session="openCurrentContextNewSession"
            />
            <div class="mini-current-stream">
              <div class="mini-ws-output" ref="outputRef" @scroll.passive="captureOutputScroll">
                <MiniWorkstationMessages
                  :messages="messages"
                  :maximized="maximized"
                  :sending="sending"
                  :counterpart-name="resourceDisplayName"
                  :full-code-path="fullCodePath"
                  :resource-type="resourceType"
                  :resource-template-type="resourceTemplateType"
                  :streaming-display-length="streamingDisplayLength"
                  :render-markdown="renderMarkdown"
                  :format-message-time="formatMessageTime"
                  :get-file-groups-from-calls="getFileGroupsFromCalls"
                  :get-display-fields-from-calls="getDisplayFieldsFromCalls"
                  :pending-interaction="pendingInteraction"
                  @confirm-prd="handleConfirmPrd"
                  @view="viewPendingInteraction"
                  @revise="revisePendingInteraction"
                  @cancel="cancelPendingInteraction"
                  @confirm="confirmPendingInteraction"
                />
              </div>
            </div>
            <section v-if="artifactPanelExpanded" :class="['mini-artifact-drawer', { 'is-open': artifactPanelExpanded }]">
              <MiniWorkstationArtifactPanel
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

        <MiniWorkstationComposer
          :full-code-path="fullCodePath"
          :dir-name="resourceDisplayName"
          :resource-type="resourceType"
          :resource-template-type="resourceTemplateType"
          :attached-files="attachedFiles"
          :uploading="uploading"
          :input-text="inputText"
          :sending="sending"
          :session-running="currentSessionRunning"
          :stopping="stopping"
          :queued-count="queuedCount"
          :selected-l-l-m-config-id="selectedLLMConfigId"
          :llm-list="llmList"
          :llm-loading="llmLoading"
          :blocked="composerBlocked"
          :blocked-label="composerBlockedLabel"
          :blocked-placeholder="composerBlockedPlaceholder"
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
            <el-button
              v-if="featureFlags.scheduledTasks"
              link
              size="small"
              class="mini-settings-btn"
              :title="t('miniWorkstation.scheduledSession')"
              data-testid="mini-workstation-schedule"
              :disabled="!fullCodePath"
              @mousedown.stop
              @click.stop="openScheduledAgentTaskDialog"
            >
              <el-icon :size="15"><Calendar /></el-icon>
            </el-button>
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
                  :title="t('miniWorkstation.settings')"
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
          <span>{{ t('miniWorkstation.dropUpload') }}</span>
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

  <ScheduledAgentTaskDialog
    v-if="featureFlags.scheduledTasks"
    v-model="showScheduledAgentTaskDialog"
    :full-code-path="fullCodePath"
    :initial-message="scheduledDraftMessage"
    :initial-files="scheduledDraftFiles"
    :initial-attached-files="attachedFiles"
    :initial-l-l-m-config-id="scheduledDraftLLMConfigId"
  />

</template>

<script setup lang="ts">
import { nextTick, onMounted, ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowLeft,
  ArrowRight,
  Calendar,
  Clock,
  Close,
  DataBoard,
  Plus,
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
import MiniWorkstationResourceIdentity from './MiniWorkstationResourceIdentity.vue'
import MiniWorkstationSessionPanel from './MiniWorkstationSessionPanel.vue'
import ScheduledAgentTaskDialog from './ScheduledAgentTaskDialog.vue'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'
import { useMiniWorkstationPanel } from '../composables/useMiniWorkstationPanel'
import { useMiniWorkstationSessions } from '../composables/useMiniWorkstationSessions'
import { useMiniWorkstationUploads } from '../composables/useMiniWorkstationUploads'
import { useMiniWorkstationComposer } from '../composables/useMiniWorkstationComposer'
import { useMiniWorkstationEffects } from '../composables/useMiniWorkstationEffects'
import {
  buildMiniArtifactItems,
  isGeneratedArtifactToolCall,
  type MiniArtifactItem
} from '../composables/useMiniWorkstationArtifacts'
import {
  collectMessageToolCalls,
  useMiniWorkstationDebugCopy
} from '../composables/useMiniWorkstationDebugCopy'
import {
  getMiniWorkstationSessionFilters,
  useMiniWorkstationSessionView,
  type SessionFilterValue
} from '../composables/useMiniWorkstationSessionView'
import { eventBus } from '@/architecture/presentation/context/eventBusContext'
import { createWorkspaceHandoff, recordWorkspaceInteractionEvent, resolveWorkspaceSessionInteraction, type WorkspaceInteraction, type WorkspaceSessionItem } from '@/architecture/presentation/context/api/workspace'
import { featureFlags } from '@/architecture/shared/config/features'

const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()
const { t } = useI18n()

const props = defineProps<{
  visible: boolean
  fullCodePath: string
  dirName?: string
  resourceType?: string
  resourceTemplateType?: string
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
const inputRef = ref<{ focus: () => void }>()
const llmSelectOpen = ref(false)
const settingsPopoverOpen = ref(false)
const showScheduledAgentTaskDialog = ref(false)
const scheduledDraftMessage = ref('')
const scheduledDraftFiles = ref('')
const scheduledDraftLLMConfigId = ref(0)
const interactionOpen = computed(() => llmSelectOpen.value || settingsPopoverOpen.value)
const handledInteractionKeys = ref<Set<string>>(new Set())
const artifactPanelExpanded = ref(false)
const collapsed = ref(props.initialExpanded === false)
const suppressAutoSelectLatestSession = ref(false)
const sessionSearchKeyword = ref('')
const sessionFilter = ref<SessionFilterValue>('all')
const abortActiveWorkspaceStream = ref<(() => void) | null>(null)
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

function registerInputRef(element: { focus: () => void } | null) {
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
  loadMiniSessionMessages,
  handleNewSession,
  handleStopSession,
  handleSelectSession,
  formatRelativeTime,
  formatMessageTime,
  startMiniPoll,
  stopMiniPoll
} = useMiniWorkstationSessions({
  fullCodePath: fullCodePathRef,
  initialSessionId: initialSessionIdRef,
  maximized,
  sending,
  sessionId,
  setMessages,
  abortActiveStream: () => abortActiveWorkspaceStream.value?.(),
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

const attachedFileRefs = computed(() => {
  return attachedFiles.value
    .map((file) => file.ref)
    .filter((ref): ref is string => !!ref)
    .join(',')
})

const artifactItems = computed<MiniArtifactItem[]>(() => {
  return buildMiniArtifactItems({
    uploadedFiles: uploadedFiles.value,
    outputFiles: outputFiles.value,
    displayFields: allPanelDisplayFields.value
  })
})

const artifactToggleCount = computed(() => Math.max(artifactItems.value.length, panelItemCount.value))

const hasCurrentGeneratedArtifacts = computed(() => {
  if (artifactItems.value.length > 0) return true
  return messages.value.some(messageHasGeneratedArtifacts)
})

const sessionFilters = computed(() => getMiniWorkstationSessionFilters())

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
function getMappedWorkspaceName(fullCodePath: string) {
  const normalizedPath = normalizeFullCodePath(fullCodePath || '')
  if (!normalizedPath) return ''
  const map = props.pathNameMap || {}
  return map[normalizedPath] || map[normalizedPath.replace(/^\/+/, '')] || ''
}

const resourceDisplayName = computed(() => {
  return getMappedWorkspaceName(props.fullCodePath)
    || (props.dirName || '').trim()
    || displayPath.value
    || t('miniWorkstation.currentDirectory')
})

const currentContextName = computed(() => {
  const mappedName = getMappedWorkspaceName(props.currentFullCodePath || '')
  if (mappedName) return mappedName
  const label = (props.currentDirName || '').trim()
  if (label) return label
  const path = normalizedCurrentContextPath.value
  return path.split('/').filter(Boolean).pop() || t('miniWorkstation.currentDirectory')
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

const currentSessionItem = computed<WorkspaceSessionItem | null>(() => {
  if (!sessionId.value) return null
  return miniSessionList.value.find(item => item.session_id === sessionId.value)
    || globalSessionList.value.find(item => item.session_id === sessionId.value)
    || null
})

const currentSessionRunning = computed(() => {
  if (sending.value) return false
  const status = currentSessionItem.value?.status || ''
  return [
    'generating',
    'running',
    'tool_running',
    'thinking',
    'streaming',
    'processing',
    'executing'
  ].includes(status)
})

const currentSessionDisablesPendingInteraction = computed(() => {
  const session = currentSessionItem.value
  const status = session?.status
  return !!session?.archived_for_model || status === 'done' || status === 'cancelled'
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

const composer = useMiniWorkstationComposer({
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

abortActiveWorkspaceStream.value = composer.abortActiveStream

const {
  llmList,
  llmLoading,
  selectedLLMConfigId,
  queuedCount,
  onLLMSelectVisibleChange: loadLLMOptionsOnVisibleChange,
  onInputEnter,
  handleSend,
  sendTextToSession
} = composer

function openScheduledAgentTaskDialog() {
  scheduledDraftMessage.value = inputText.value.trim()
  scheduledDraftFiles.value = attachedFileRefs.value
  scheduledDraftLLMConfigId.value = selectedLLMConfigId.value || 0
  showScheduledAgentTaskDialog.value = true
}

type StageInteractionArtifact = Record<string, unknown> & {
  kind?: string
  interaction?: Partial<WorkspaceInteraction>
}

const pendingInteraction = computed<WorkspaceInteraction | null>(() => {
  if (currentSessionDisablesPendingInteraction.value) return null
  const auditedInteractionKeys = new Set<string>()
  let hasUnscopedAuditAfter = false
  for (let i = messages.value.length - 1; i >= 0; i--) {
    const message = messages.value[i]
    if (!message) continue
    const auditedKey = getWorkspaceInteractionAuditResolutionKey(message)
    if (auditedKey !== undefined) {
      if (auditedKey) {
        auditedInteractionKeys.add(auditedKey)
      } else {
        hasUnscopedAuditAfter = true
      }
      continue
    }
    const calls = collectMessageToolCalls(message)
    for (let j = calls.length - 1; j >= 0; j--) {
      const call = calls[j]
      if (!call) continue
      const interaction = buildWorkspaceInteractionFromArtifact(call.result_data)
      if (!interaction) continue
      const key = getInteractionKey(interaction)
      if (handledInteractionKeys.value.has(key) || auditedInteractionKeys.has(key) || hasUnscopedAuditAfter) {
        return null
      }
      return interaction
    }
  }
  return null
})

const composerBlocked = computed(() => {
  const interaction = pendingInteraction.value
  return !!interaction && isComposerBlockingInteraction(interaction)
})
const composerBlockedLabel = computed(() => {
  const interaction = pendingInteraction.value
  if (!interaction) return ''
  if (interaction.card_type === 'prd_confirmation') return t('miniWorkstation.blockingPrd')
  if (interaction.card_type === 'build_repair') return t('miniWorkstation.blockingRepair')
  return t('miniWorkstation.blockingGeneric')
})
const composerBlockedPlaceholder = computed(() => {
  const interaction = pendingInteraction.value
  if (!interaction) return t('miniWorkstation.composerDefaultPlaceholder')
  return interaction.help_text || interaction.description || t('miniWorkstation.interactionNeedAction')
})

const hasCurrentOutputContent = computed(() => {
  return sending.value
    || messages.value.length > 0
    || artifactItems.value.length > 0
    || !!pendingInteraction.value
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

function buildWorkspaceInteractionFromArtifact(value: unknown): WorkspaceInteraction | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  const artifact = value as StageInteractionArtifact
  const rawInteraction = artifact.interaction
  if (!rawInteraction || typeof rawInteraction !== 'object') return null
  const status = typeof rawInteraction.status === 'string' ? rawInteraction.status.trim() : ''
  if (!status.startsWith('pending_')) return null
  const cardType = typeof rawInteraction.card_type === 'string' ? rawInteraction.card_type : fallbackCardType(artifact.kind, status)
  return {
    id: typeof rawInteraction.id === 'string' ? rawInteraction.id : getStageArtifactKey(artifact),
    card_type: cardType,
    artifact_kind: typeof rawInteraction.artifact_kind === 'string' ? rawInteraction.artifact_kind : artifact.kind,
    status,
    blocking: typeof rawInteraction.blocking === 'boolean' ? rawInteraction.blocking : true,
    title: typeof rawInteraction.title === 'string' ? rawInteraction.title : fallbackInteractionTitle(cardType),
    description: typeof rawInteraction.description === 'string' ? rawInteraction.description : undefined,
    help_text: typeof rawInteraction.help_text === 'string' ? rawInteraction.help_text : undefined,
    view_text: typeof rawInteraction.view_text === 'string' ? rawInteraction.view_text : undefined,
    confirm_text: typeof rawInteraction.confirm_text === 'string' ? rawInteraction.confirm_text : undefined,
    revise_text: typeof rawInteraction.revise_text === 'string' ? rawInteraction.revise_text : undefined,
    cancel_text: typeof rawInteraction.cancel_text === 'string' ? rawInteraction.cancel_text : undefined,
    target_role_on_confirm: typeof rawInteraction.target_role_on_confirm === 'string' ? rawInteraction.target_role_on_confirm : undefined,
    allowed_actions: Array.isArray(rawInteraction.allowed_actions) ? rawInteraction.allowed_actions.map(String) : undefined,
    artifact
  }
}

function getStageArtifactKey(artifact: unknown) {
  try {
    return JSON.stringify(artifact)
  } catch {
    return String(artifact)
  }
}

function getInteractionKey(interaction: WorkspaceInteraction) {
  return interaction.id || getStageArtifactKey(interaction.artifact) || `${interaction.status}:${interaction.card_type}`
}

function getWorkspaceInteractionAuditResolutionKey(message: ChatMessage): string | undefined {
  if (message.artifact_kind !== 'workspace_interaction_event') return undefined
  const raw = (message.raw_content || '').trim()
  if (!raw) return workspaceInteractionAuditDisplayResolves(message.content) ? '' : undefined
  try {
    const event = JSON.parse(raw) as { kind?: unknown; interaction_id?: unknown; action?: unknown }
    if (event.kind === 'workspace_interaction_event') {
      if (!workspaceInteractionAuditActionResolves(typeof event.action === 'string' ? event.action : '')) {
        return undefined
      }
      return typeof event.interaction_id === 'string' ? event.interaction_id : ''
    }
  } catch {
    return workspaceInteractionAuditDisplayResolves(message.content) ? '' : undefined
  }
  return workspaceInteractionAuditDisplayResolves(message.content) ? '' : undefined
}

function workspaceInteractionAuditActionResolves(action: string) {
  return [
    'confirm_prd',
    'revise_prd',
    'cancel_prd',
    'start_build_repair',
    'continue_development',
    'skip_build_repair',
  ].includes(action)
}

function workspaceInteractionAuditDisplayResolves(content: string) {
  const text = content || ''
  if (text.includes('查看 PRD') || text.includes('查看构建诊断')) return false
  return text.includes('确认 PRD') ||
    text.includes('修改 PRD') ||
    text.includes('取消 PRD') ||
    text.includes('交接构建修复') ||
    text.includes('继续修改') ||
    text.includes('暂不修复')
}

function markInteractionHandled(interaction: WorkspaceInteraction) {
  const next = new Set(handledInteractionKeys.value)
  next.add(getInteractionKey(interaction))
  handledInteractionKeys.value = next
}

async function handleBeforeSend(_payload: { text: string; files: unknown[] | null }) {
  const interaction = pendingInteraction.value
  if (!interaction) {
    return false
  }
  if (isComposerBlockingInteraction(interaction)) {
    ElMessage.warning(t('miniWorkstation.interactionHandleFirst'))
    return { cancel: true, preserveDraft: true }
  }
  if (interaction.card_type === 'build_repair') {
    await recordPendingInteractionAction(
      interaction,
      'continue_development',
      _payload.text ? `${t('miniWorkstation.continueDevelopment')}：${_payload.text}` : undefined
    )
    markInteractionHandled(interaction)
    return { interactionAction: 'continue_development' }
  }
  return false
}

function isComposerBlockingInteraction(interaction: WorkspaceInteraction) {
  if (interaction.card_type === 'build_repair') return false
  return interaction.blocking
}

async function clearCurrentPendingInteractionStatus() {
  if (!sessionId.value) return
  try {
    await resolveWorkspaceSessionInteraction(sessionId.value)
    void loadMiniSessions()
    void loadGlobalSessions()
  } catch (error: any) {
    ElMessage.warning(error?.message || t('miniWorkstation.pendingSyncFailed'))
  }
}

async function recordPendingInteractionAction(interaction: WorkspaceInteraction, action: string, displayContent?: string) {
  if (!sessionId.value) return
  const interactionKey = getInteractionKey(interaction)
  try {
    await recordWorkspaceInteractionEvent({
      session_id: sessionId.value,
      action,
      interaction_id: interactionKey,
      card_type: interaction.card_type,
      status: interaction.status,
      artifact_kind: interaction.artifact_kind,
      content: JSON.stringify({
        kind: 'workspace_interaction_event',
        interaction_id: interactionKey,
        action,
        card_type: interaction.card_type,
        status: interaction.status,
        artifact_kind: interaction.artifact_kind,
      }),
      display_content: displayContent || interactionAuditText(interaction, action),
    })
    await loadMiniSessionMessages(sessionId.value)
  } catch (error: any) {
    ElMessage.warning(error?.message || t('miniWorkstation.interactionRecordFailed'))
  }
}

async function viewPendingInteraction(target?: WorkspaceInteraction) {
  const interaction = target || pendingInteraction.value
  if (!interaction) return
  await recordPendingInteractionAction(interaction, viewInteractionAction(interaction))
  await nextTick()
  focusInteractionArtifact(interaction)
}

function focusInteractionArtifact(interaction: WorkspaceInteraction) {
  const root = outputRef.value
  if (!root) return
  const selector = interaction.card_type === 'build_repair'
    ? '.mini-msg-build-diagnostics'
    : '.mini-msg-prd-preview'
  const targets = root.querySelectorAll(selector)
  const target = targets[targets.length - 1]
  if (target instanceof HTMLElement) {
    target.scrollIntoView({ behavior: 'smooth', block: 'center' })
    return
  }
  root.scrollTo({ top: root.scrollHeight, behavior: 'smooth' })
}

function onLLMSelectVisibleChange(visible: boolean) {
  llmSelectOpen.value = visible
  loadLLMOptionsOnVisibleChange(visible)
}

async function handleConfirmPrd(payload: { remark: string; prd: unknown }, options: { auditRecorded?: boolean } = {}) {
  const remark = payload.remark.trim()
  if (!sessionId.value || !props.fullCodePath || sending.value) {
    ElMessage.warning(t('miniWorkstation.confirmPrdNotReady'))
    return
  }
  const interaction = buildWorkspaceInteractionFromArtifact(payload.prd)
  let handoff
  try {
    handoff = await createWorkspaceHandoff({
      source_session_id: sessionId.value,
      full_code_path: props.fullCodePath,
      target_role: getPrdTargetRole(payload.prd),
      artifact_kind: 'agent_app_prd',
      artifact: payload.prd,
      remark,
      context_policy: 'full'
    })
  } catch (error: any) {
    ElMessage.error(error?.message || t('miniWorkstation.confirmPrdFailed'))
    return
  }
  if (interaction && !options.auditRecorded) {
    await recordPendingInteractionAction(interaction, 'confirm_prd')
  }
  if (interaction) markInteractionHandled(interaction)
  sessionId.value = handoff.session_id
  void sendTextToSession(
    handoff.session_id,
    handoff.content,
    handoff.display_content,
    { contextUsage: 'artifact', artifactKind: handoff.artifact_kind, resume: true }
  )
}

async function handleConfirmBuildHandoff(payload: { artifact: unknown }) {
  if (!sessionId.value || !props.fullCodePath || sending.value) {
    ElMessage.warning(t('miniWorkstation.repairNotReady'))
    return
  }
  let handoff
  try {
    handoff = await createWorkspaceHandoff({
      source_session_id: sessionId.value,
      full_code_path: props.fullCodePath,
      target_role: getBuildHandoffTargetRole(payload.artifact),
      artifact_kind: getStageArtifactKind(payload.artifact, 'agent_app_build_failure'),
      artifact: payload.artifact,
      remark: '',
      context_policy: 'full',
      display_content: t('miniWorkstation.buildRepairDisplayContent')
    })
  } catch (error: any) {
    ElMessage.error(error?.message || t('miniWorkstation.buildRepairCreateFailed'))
    return
  }
  const interaction = buildWorkspaceInteractionFromArtifact(payload.artifact)
  if (interaction) markInteractionHandled(interaction)
  sessionId.value = handoff.session_id
  void sendTextToSession(
    handoff.session_id,
    handoff.content,
    handoff.display_content,
    { contextUsage: 'artifact', artifactKind: handoff.artifact_kind, resume: true }
  )
}

function confirmPendingInteraction(target?: WorkspaceInteraction) {
  const interaction = target || pendingInteraction.value
  if (!interaction) return
  if (interaction.card_type === 'build_repair') {
    void (async () => {
      await recordPendingInteractionAction(interaction, 'start_build_repair')
      await handleConfirmBuildHandoff({ artifact: interaction.artifact })
    })()
    return
  }
  if (interaction.card_type === 'prd_confirmation') {
    void (async () => {
      await recordPendingInteractionAction(interaction, 'confirm_prd')
      await handleConfirmPrd({ remark: '', prd: interaction.artifact }, { auditRecorded: true })
    })()
    return
  }
  ElMessage.warning(t('miniWorkstation.confirmActionMissing'))
}

async function revisePendingInteraction(payload: { text: string; interaction?: WorkspaceInteraction }) {
  const interaction = payload.interaction || pendingInteraction.value
  const text = payload.text.trim()
  if (!interaction || !sessionId.value || !text || sending.value) return
  const isBuildRepair = interaction.card_type === 'build_repair'
  if (!isBuildRepair && interaction.card_type !== 'prd_confirmation') {
    ElMessage.warning(t('miniWorkstation.reviseActionMissing'))
    return
  }
  const prefix = isBuildRepair ? t('miniWorkstation.continueDevelopment') : t('miniWorkstation.revisePrd')
  const action = isBuildRepair ? 'continue_development' : 'revise_prd'
  await recordPendingInteractionAction(interaction, action, `${prefix}：${text}`)
  markInteractionHandled(interaction)
  await sendTextToSession(
    sessionId.value,
    `${prefix}：${text}`,
    `${prefix}：${text}`,
    { interactionAction: action }
  )
}

async function cancelPendingInteraction(target?: WorkspaceInteraction) {
  const interaction = target || pendingInteraction.value
  if (!interaction) return
  await recordPendingInteractionAction(interaction, cancelInteractionAction(interaction))
  markInteractionHandled(interaction)
  await clearCurrentPendingInteractionStatus()
  ElMessage.info(interaction.card_type === 'build_repair' ? t('miniWorkstation.cancelBuildRepairInfo') : t('miniWorkstation.cancelConfirmationInfo'))
}

function getPrdTargetRole(prd: unknown) {
  const interaction = buildWorkspaceInteractionFromArtifact(prd)
  if (interaction?.target_role_on_confirm) {
    return interaction.target_role_on_confirm
  }
  return 'app_developer'
}

function getBuildHandoffTargetRole(artifact: unknown) {
  const interaction = buildWorkspaceInteractionFromArtifact(artifact)
  if (interaction?.target_role_on_confirm) {
    return interaction.target_role_on_confirm
  }
  return 'build_engineer'
}

function getStageArtifactKind(artifact: unknown, fallback: string) {
  if (artifact && typeof artifact === 'object') {
    const data = artifact as { interaction?: { artifact_kind?: string }, kind?: string }
    return data.interaction?.artifact_kind || data.kind || fallback
  }
  return fallback
}

function fallbackCardType(kind: unknown, status: string) {
  if (kind === 'agent_app_build_failure' || status === 'pending_build_repair') return 'build_repair'
  if (kind === 'agent_app_prd' || status === 'pending_confirmation') return 'prd_confirmation'
  return 'stage_confirmation'
}

function fallbackInteractionTitle(cardType: string) {
  if (cardType === 'build_repair') return t('miniWorkstation.buildRepairTitle')
  if (cardType === 'prd_confirmation') return t('miniWorkstation.interactionPrdTitle')
  return t('miniWorkstation.interactionWaitingTitle')
}

function viewInteractionAction(interaction: WorkspaceInteraction) {
  return interaction.card_type === 'build_repair' ? 'view_build_diagnostics' : 'view_prd'
}

function cancelInteractionAction(interaction: WorkspaceInteraction) {
  return interaction.card_type === 'build_repair' ? 'skip_build_repair' : 'cancel_prd'
}

function interactionAuditText(interaction: WorkspaceInteraction, action: string) {
  const label = interactionActionLabel(action)
  const title = interaction.title || fallbackInteractionTitle(interaction.card_type || '')
  return t('miniWorkstation.interactionAudit', { label, title })
}

function interactionActionLabel(action: string) {
  const labels: Record<string, string> = {
    view_prd: t('miniWorkstation.viewPrd'),
    confirm_prd: t('miniWorkstation.confirmPrd'),
    revise_prd: t('miniWorkstation.revisePrd'),
    cancel_prd: t('miniWorkstation.cancelPrd'),
    view_build_diagnostics: t('miniWorkstation.actionViewBuildDiagnostics'),
    start_build_repair: t('miniWorkstation.actionStartBuildRepair'),
    continue_development: t('miniWorkstation.continueDevelopment'),
    skip_build_repair: t('miniWorkstation.skipRepair'),
  }
  return labels[action] || action
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
  --mini-cyber-bg: var(--bg-primary);
  --mini-cyber-bg-strong: var(--bg-secondary);
  --mini-cyber-panel: var(--bg-primary);
  --mini-cyber-panel-soft: var(--bg-secondary);
  --mini-cyber-line: var(--border-light);
  --mini-cyber-line-strong: var(--border-base);
  --mini-cyber-text: var(--text-primary);
  --mini-cyber-muted: var(--text-secondary);
  --mini-cyber-dim: var(--text-placeholder);
  --mini-cyber-accent: var(--color-primary);
  --mini-cyber-warm: var(--color-warning);
  position: fixed;
  right: 24px;
  bottom: 80px;
  isolation: isolate;
  background: var(--bg-primary);
  border: 1px solid var(--border-light);
  border-radius: var(--border-radius-xl);
  box-shadow: var(--app-shell-panel-shadow-hover);
  color: var(--text-primary);
  backdrop-filter: blur(24px) saturate(1.2);
  z-index: var(--aos-z-mini-workstation);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: left 0.3s ease, top 0.3s ease, width 0.3s ease, height 0.3s ease, max-height 0.3s ease, border-radius 0.3s ease, box-shadow 0.2s ease;
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
  border-bottom: 1px solid var(--border-light);
  cursor: move;
  user-select: none;
  background: transparent;
  flex-shrink: 0;
  transition: padding 0.2s ease, border-color 0.2s ease;
}
.mini-ws:not(.mini-ws--maximized):not(.mini-ws--interaction-open):not(:hover):not(:focus-within) .mini-ws-header {
  padding: 8px 10px;
  border-bottom-color: transparent;
}
.mini-ws--maximized .mini-ws-header {
  cursor: default;
}
.mini-ws-title {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--color-primary);
}
.mini-ws-title-orb {
  position: relative;
  width: 25px;
  height: 25px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-light);
  border-radius: 50%;
  background: var(--bg-tertiary);
  box-shadow: none;
}
.mini-ws-title-orb::after {
  content: '';
  position: absolute;
  inset: -4px;
  border-radius: inherit;
  border: 1px solid var(--border-light);
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
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 0 4px;
  text-shadow: none;
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
  color: var(--text-secondary);
  border-radius: 8px;
}
.mini-ws-header-actions :deep(.el-button:hover) {
  color: var(--text-primary);
  background: var(--bg-tertiary);
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
  border: 1px solid var(--border-light);
  border-radius: 999px;
  background: var(--bg-tertiary);
  color: var(--color-primary);
  font-size: 10px;
  font-weight: 800;
}

.mini-settings-btn {
  width: 32px;
  height: 32px;
  border: 1px solid transparent;
  border-radius: 10px;
  color: var(--color-primary);
  background: var(--el-fill-color-light);
  box-shadow: none;
  transition: background-color 0.2s ease, border-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}
.mini-settings-btn:hover,
.mini-settings-btn:focus {
  border-color: var(--border-light);
  color: var(--color-primary-light-1);
  background: var(--el-fill-color);
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
  border-left: 1px solid var(--border-light);
  background: var(--bg-secondary);
}
.mini-file-sidebar-header {
  padding: 10px 12px;
  font-size: 13px;
  font-weight: 700;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-light);
  background: var(--bg-tertiary);
  flex-shrink: 0;
}

/* ── SSE 输出区 ── */
.mini-ws-output {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px 24px;
  min-height: 0;
  font-size: 13px;
  line-height: 1.6;
  background: transparent;
  scrollbar-color: rgba(var(--color-primary-rgb), 0.36) transparent;
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
  background: var(--bg-tertiary);
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
  background: transparent;
  border: 1px dashed var(--border-base);
  border-radius: 12px;
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 500;
  pointer-events: none;
  box-shadow: none;
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
  0%, 100% { opacity: 0.72; transform: scale(1); box-shadow: none; }
  50% { opacity: 1; transform: scale(1.4); box-shadow: none; }
}



.mini-ws--sending {
  box-shadow: none;
}

.mini-ws:not(.mini-ws--maximized):not(.mini-ws--interaction-open):not(:hover):not(:focus-within) {
  border-radius: 0;
  box-shadow: none;
}





.mini-shell {
  position: absolute;
  left: 42px;
  right: 42px;
  bottom: calc(24px + var(--mini-stack-offset, 0px));
  display: flex;
  flex-direction: column;
  gap: 0;
  color: var(--text-primary);
  pointer-events: auto;
  transition: left 0.18s ease, right 0.18s ease, bottom 0.18s ease;
}

.mini-current-output,
.mini-shell :deep(.mini-ws-input) {
  border: 1px solid transparent;
  background: transparent;
  box-shadow: none;
  backdrop-filter: none;
}

.mini-current-output {
  height: min(720px, calc(100vh - 180px));
  min-height: min(520px, calc(100vh - 180px));
  max-height: none;
  display: flex;
  flex-direction: column;
  margin: 0 14px 8px;
  padding: 12px 14px;
  border-color: transparent;
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
  gap: 12px;
  padding-right: 12px;
  border-right: 1px solid var(--border-light);
  color: var(--text-primary);
  font-size: 12px;
  overflow: hidden;
}

.mini-current-session-head {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 4px 4px;
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
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 600;
}

.mini-current-session-head span {
  color: var(--text-disabled);
  font-size: 12px;
}

.mini-current-session-head em {
  min-width: 24px;
  height: 24px;
  display: inline-grid;
  place-items: center;
  border-radius: 8px;
  background: var(--bg-secondary);
  color: var(--color-primary);
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
  grid-template-columns: 8px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  padding: 10px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: var(--text-primary);
  text-align: left;
  cursor: pointer;
  transition: background 0.18s ease, border-color 0.18s ease, box-shadow 0.22s ease;
}

.mini-current-session-row:hover {
  background: var(--bg-tertiary);
}

.mini-current-session-row.is-running {
  background: transparent;
}

.mini-current-session-row.is-waiting {
  background: transparent;
}

.mini-current-session-row.is-output {
  background: transparent;
}

.mini-current-session-row.is-done {
  background: transparent;
}

.mini-current-session-row.is-cancelled {
  background: transparent;
  opacity: 0.5;
}

.mini-current-session-row.is-failed {
  background: transparent;
  border-left: 3px solid var(--color-danger);
  border-radius: 6px;
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
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 500;
  line-height: 1.2;
}

.mini-current-session-sub {
  margin-top: 4px;
  color: var(--text-secondary);
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
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: var(--text-secondary);
  transition: all 0.2s ease;
}

.mini-icon-action:hover {
  background: var(--el-fill-color-light);
  color: var(--text-primary);
}

.mini-queue-chip {
  width: fit-content;
  max-width: 100%;
  height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 7px;
  border: 1px solid var(--border-light);
  border-radius: 999px;
  background: var(--bg-tertiary);
  color: var(--color-warning);
  font-size: 11px;
  font-weight: 600;
}

.mini-ws-output {
  min-height: 0;
  height: 100%;
  overflow: auto;
  padding: 12px 14px;
  border: 1px solid transparent;
  border-radius: 12px;
  background: transparent;
  color: var(--text-primary);
  font-size: 13px;
  line-height: 1.6;
  scrollbar-color: rgba(var(--color-primary-rgb), 0.3) transparent;
}

.mini-ws:not(.mini-ws--maximized):not(.mini-ws--interaction-open):not(:hover):not(:focus-within) .mini-ws-output {
  padding: 12px 14px;
  font-size: 13px;
}

.mini-ws--maximized .mini-ws-output {
  height: 100%;
  overflow: auto;
  padding: 14px 16px;
  border: 1px solid transparent;
  border-radius: 12px;
  background: transparent;
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
  background: var(--bg-tertiary);
  border-left: 3px solid var(--color-primary);
  border-radius: 6px;
}

.mini-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(var(--color-primary-rgb), 0.15);
}

.mini-status-dot.is-running {
  background: var(--color-success);
  box-shadow: 0 0 0 2px rgba(var(--color-success-rgb), 0.15);
}

.mini-status-dot.is-waiting {
  background: var(--color-warning);
  box-shadow: 0 0 0 2px rgba(var(--color-warning-rgb), 0.15);
}

.mini-status-dot.is-done,
.mini-status-dot.is-cancelled {
  background: var(--text-disabled);
  box-shadow: none;
}

.mini-status-dot.is-failed {
  background: var(--color-danger);
  box-shadow: 0 0 0 2px rgba(var(--color-danger-rgb), 0.15);
}

.mini-status-dot.is-active,
.mini-status-dot.is-output {
  background: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(var(--color-primary-rgb), 0.15);
}

.mini-settings-btn {
  width: 40px;
  height: 40px;
  border-color: transparent;
  border-radius: 8px;
  color: var(--color-primary);
  background: var(--el-fill-color-light);
  box-shadow: none;
  transition: background-color 0.2s ease, border-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.mini-settings-btn:hover,
.mini-settings-btn:focus {
  border-color: var(--border-light);
  color: var(--color-primary-light-1);
  background: var(--el-fill-color);
}

.mini-ws-drop-overlay {
  inset: auto 86px calc(24px + var(--mini-stack-offset, 0px)) max(162px, 12vw);
  height: 210px;
  pointer-events: none;
  border-color: var(--border-light);
  background: var(--bg-secondary);
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

.mini-workspace-backdrop {
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
  border: 1px solid var(--border-light);
  border-radius: 12px;
  background: var(--bg-primary);
  box-shadow:
    0 24px 72px rgba(8, 14, 24, 0.28),
    0 0 0 1px rgba(255, 255, 255, 0.05);
  color: var(--text-primary);
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
  border-bottom: 1px solid var(--border-light);
  background: transparent;
}

.mini-drawer-title {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.mini-drawer-resource {
  max-width: 100%;
  color: #f5f8ff;
  font-size: 15px;
  font-weight: 850;
  line-height: 1.2;
}

.mini-drawer-resource :deep(.mini-resource-identity__icon) {
  --mini-resource-icon-size: 22px;
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
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: all 0.2s ease;
}

.mini-drawer-primary-action:hover,
.mini-drawer-secondary-action:hover,
.mini-drawer-icon-action:hover {
  border-color: var(--border-light);
  background: var(--el-fill-color);
  color: var(--color-primary);
}

.mini-drawer-secondary-action.is-active {
  border-color: rgba(var(--color-primary-rgb), 0.2);
  background: rgba(var(--color-primary-rgb), 0.1);
  color: var(--color-primary);
  font-weight: 600;
}

.mini-drawer-primary-action {
  padding: 0 11px;
  border-color: rgba(var(--color-primary-rgb), 0.18);
  background: rgba(var(--color-primary-rgb), 0.09);
  color: var(--color-primary);
  font-weight: 600;
}

.mini-drawer-secondary-action {
  padding: 0 11px;
}

.mini-drawer-icon-action {
  width: 32px;
  padding: 0;
}

.mini-drawer-secondary-action:disabled {
  cursor: not-allowed;
  opacity: 0.42;
  background: transparent;
}

.mini-current-output,
.mini-ws--maximized .mini-current-output {
  height: 100%;
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
  border-right: 1px solid var(--border-light);
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
  border: 1px solid var(--border-light);
  border-radius: 8px;
  background: var(--bg-tertiary);
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
  border: 1px solid var(--border-light);
  border-radius: 7px;
  background: var(--bg-tertiary);
  color: var(--color-primary);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.mini-current-context-switch button:hover {
  background: var(--el-fill-color-light);
  color: var(--color-primary-light-1);
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
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.mini-drawer-scope-tabs button:hover,
.mini-drawer-session-filters button:hover {
  background: var(--el-fill-color-light);
  color: var(--text-primary);
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
  background: var(--bg-tertiary);
  color: var(--color-primary);
  font-weight: 600;
}

.mini-drawer-session-search {
  height: 32px;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  border: 1px solid var(--border-light);
  border-radius: 8px;
  background: transparent;
  color: var(--text-secondary);
}

.mini-drawer-session-search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  color: var(--text-primary);
  font-size: 12px;
}

.mini-drawer-session-search input::placeholder {
  color: var(--text-disabled);
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
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
}

.mini-artifact-drawer {
  grid-column: 1 / -1;
  min-width: 0;
  display: grid;
  gap: 8px;
  overflow: hidden;
  padding: 8px 0 0;
  border-top: 1px solid var(--border-light);
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
  border: 1px solid var(--border-light);
  border-radius: 8px;
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 760;
  cursor: pointer;
  transition: background 0.16s ease, border-color 0.16s ease, color 0.16s ease;
}

.mini-artifact-toggle:hover {
  border-color: var(--border-light);
  background: var(--bg-tertiary);
  color: var(--text-primary);
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
  border-left: 1px solid var(--border-light);
}

.mini-ws--maximized .mini-artifact-drawer.is-open .mini-artifact-panel {
  height: 100%;
  max-height: none;
}

.mini-shell :deep(.mini-ws-input) {
  margin: 0;
  border-width: 1px 0 0;
  border-radius: 0;
  border-color: var(--border-light);
  background: var(--el-bg-color);
  box-shadow: none;
}

.mini-shell :deep(.mini-ws-input:hover) {
  border-color: var(--border-light);
  background: var(--el-bg-color);
}

.mini-shell :deep(.mini-ws-input:focus-within) {
  border-color: var(--border-light);
  background: var(--el-bg-color);
  box-shadow: none;
}

.mini-ws-files {
  margin: 0;
  padding: 8px 12px;
  border-top: 1px solid var(--border-light);
  background: transparent;
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
    border-top: 1px solid var(--border-light);
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
    border-bottom: 1px solid var(--border-light);
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
  background: var(--bg-tertiary);
  border-color: var(--border-light);
}
</style>
