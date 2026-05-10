<!--
  MiniWorkstation - 迷你浮动工作台
  右下角弹出的小窗口，支持输入命令、上传文件、SSE 实时输出、最小化。
-->
<template>
  <transition name="mini-ws-pop">
    <div
      v-if="visible"
      ref="rootRef"
      :class="['mini-ws', { 'mini-ws--maximized': maximized, 'mini-ws--sending': sending, 'mini-ws--collapsed': collapsed, 'mini-ws--interaction-open': interactionOpen }]"
      data-testid="mini-workstation"
      :data-full-code-path="fullCodePath"
      :style="windowStyle"
      @dragover.prevent="onDragOver"
      @dragleave.prevent="onDragLeave"
      @drop.prevent="onDrop"
    >
      <button
        v-if="collapsed"
        type="button"
        :class="['mini-collapsed-launcher', { 'has-update': sending || panelHasContent }]"
        @click="setCollapsed(false)"
      >
        <span class="mini-collapsed-pulse"></span>
        <strong>工作台</strong>
        <span>{{ collapsedSummaryText }}</span>
        <span v-if="summaryBadgeCount > 0" class="mini-count-badge">{{ summaryBadgeCount }}</span>
      </button>

      <section v-else class="mini-shell">
        <section class="mini-current-output">
          <header class="mini-current-head">
            <div class="mini-current-title">
              <span class="mini-status-dot" :class="activeStatusClass"></span>
              <span class="mini-current-name" :title="activeSessionTitle">{{ activeSessionTitle }}</span>
              <span class="mini-current-state">{{ activeSessionStateLabel }}</span>
            </div>
            <div class="mini-current-actions">
              <span v-if="queuedCount > 0" class="mini-queue-chip">{{ queuedCount }} 条排队</span>
              <el-dropdown
                ref="keyInfoDropdownRef"
                v-if="panelHasContent"
                trigger="click"
                placement="top-end"
                popper-class="mini-files-dropdown-popper"
                :hide-on-click="false"
                @visible-change="onKeyInfoDropdownVisibleChange"
              >
                <button type="button" class="mini-icon-action" title="查看关键信息">
                  <el-icon :size="14"><DocumentIcon /></el-icon>
                  <span class="mini-header-files-count">{{ panelItemCount }}</span>
                </button>
                <template #dropdown>
                  <div class="mini-files-dropdown-panel">
                    <div class="mini-files-dropdown-title">关键信息</div>
                    <MiniWorkstationKeyInfoSection
                      compact
                      :uploaded-files="uploadedFiles"
                      :output-files="outputFiles"
                      :display-fields="allPanelDisplayFields"
                      @preview-file="previewFile"
                      @download-file="downloadFile"
                      @preview-field="openDfPreview"
                      @copy-field="copyDisplayFieldValue"
                    />
                  </div>
                </template>
              </el-dropdown>
              <button type="button" class="mini-icon-action" :title="maximized ? '还原当前输出' : '放大当前输出'" @click="toggleMaximize">
                <el-icon :size="14"><component :is="maximized ? CopyDocument : FullScreen" /></el-icon>
              </button>
              <button type="button" class="mini-icon-action" title="折叠到底部" @click="setCollapsed(true)">
                <el-icon :size="14"><Minus /></el-icon>
              </button>
              <button type="button" class="mini-icon-action" data-testid="mini-workstation-close" title="关闭" @click="$emit('close')">
                <el-icon :size="14"><Close /></el-icon>
              </button>
            </div>
          </header>

          <div class="mini-current-body">
            <div class="mini-ws-output" ref="outputRef">
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
            <aside class="mini-artifact-panel" :class="{ 'is-empty': artifactItems.length === 0 }" aria-label="当前产物">
              <div class="mini-artifact-head">
                <span>{{ maximized ? '当前产物' : '产物' }}</span>
                <strong>{{ artifactItems.length }} 项</strong>
              </div>
              <button
                v-for="item in artifactItems"
                :key="item.key"
                type="button"
                class="mini-artifact-item"
                @click="handleArtifactClick(item)"
              >
                <span class="mini-artifact-copy">
                  <span class="mini-artifact-name">{{ item.name }}</span>
                  <span class="mini-artifact-meta">{{ item.meta }}</span>
                </span>
                <span class="mini-artifact-tag">
                  {{ item.tag }}
                  <span v-if="!maximized && artifactItems.length > 1" class="mini-artifact-mini-count">{{ artifactItems.length }}项</span>
                </span>
              </button>
              <div v-if="artifactItems.length === 0" class="mini-artifact-empty">
                <span>暂无产物</span>
                <em>等待</em>
              </div>
            </aside>
          </div>
        </section>

        <section class="mini-session-dock" aria-label="会话摘要">
          <button type="button" class="mini-session-center-btn" @click="openSessionCenter">
            <span class="mini-count-badge">{{ miniSessionList.length }}</span>
            <span>会话中心</span>
          </button>
          <div class="mini-session-summary-list">
            <button
              v-if="summarySessions.length === 0"
              type="button"
              class="mini-session-summary-card active is-draft"
              @click="handleNewSession"
            >
              <span class="mini-status-dot"></span>
              <span class="mini-session-summary-copy">
                <span class="mini-session-summary-title">新建会话</span>
                <span class="mini-session-summary-sub">{{ dirName || displayPath }}</span>
              </span>
            </button>
            <button
              v-for="item in summarySessions"
              :key="item.session_id"
              type="button"
              :class="['mini-session-summary-card', getSessionStatusClass(item), { active: item.session_id === sessionId }]"
              :title="getSessionTitle(item)"
              @click="handleSelectSession(item.session_id)"
            >
              <span class="mini-status-dot" :class="getSessionStatusClass(item)"></span>
              <span class="mini-session-summary-copy">
                <span class="mini-session-summary-title">{{ getSessionTitle(item) }}</span>
                <span class="mini-session-summary-sub">{{ getSessionSubtitle(item) }}</span>
              </span>
              <span v-if="item.status === 'generating'" class="mini-count-badge">•</span>
            </button>
          </div>
        </section>

        <div v-if="pendingPrd" class="mini-prd-confirm-bar" data-testid="mini-prd-confirm-bar">
          <div class="mini-prd-confirm-copy">
            <strong>PRD 等待确认</strong>
            <span>{{ pendingPrdHelpText }}</span>
          </div>
          <div class="mini-prd-confirm-actions">
            <el-button size="small" @click="focusPrdPreview">查看 PRD</el-button>
            <el-button size="small" @click="prepareRevisePrd">修改 PRD</el-button>
            <el-button size="small" @click="cancelPendingPrd">取消</el-button>
            <el-button type="primary" size="small" :loading="sending" @click="confirmPendingPrd">
              确认 PRD
            </el-button>
          </div>
        </div>

        <div v-else-if="pendingTestHandoff" class="mini-prd-confirm-bar" data-testid="mini-test-confirm-bar">
          <div class="mini-prd-confirm-copy">
            <strong>应用等待测试</strong>
            <span>{{ pendingTestHandoffHelpText }}</span>
          </div>
          <div class="mini-prd-confirm-actions">
            <el-button size="small" @click="focusPrdPreview">查看构建结果</el-button>
            <el-button size="small" @click="prepareContinueDevelopment">继续修改</el-button>
            <el-button size="small" @click="cancelPendingTestHandoff">暂不测试</el-button>
            <el-button type="primary" size="small" :loading="sending" @click="confirmPendingTestHandoff">
              开始测试
            </el-button>
          </div>
        </div>

        <MiniWorkstationComposer
          :full-code-path="fullCodePath"
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
          @update:input-text="inputText = $event"
          @update:selected-l-l-m-config-id="selectedLLMConfigId = $event"
          @schedule="openNewScheduledAgentTaskDialog"
          @stop="handleStopSession"
          @send="handleSend"
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
              <div class="mini-settings-panel" @mousedown.stop @click.stop>
                <section class="mini-settings-section">
                  <div class="mini-settings-section-title">复制</div>
                  <div class="mini-settings-copy-grid">
                    <button type="button" @click="copyDebugConversation('all')">全部对话</button>
                    <button type="button" @click="copyDebugConversation('last-turn')">最后一轮</button>
                    <button type="button" @click="copyDebugConversation('all-tools')">全部工具</button>
                    <button type="button" @click="copyDebugConversation('error-tools')">失败工具</button>
                    <button type="button" @click="copyDebugConversation('success-tools')">成功工具</button>
                  </div>
                </section>
                <section class="mini-settings-section">
                  <header class="mini-debug-head">
                    <div>
                      <span class="mini-debug-kicker">Tool Trace</span>
                      <strong>调用摘要</strong>
                    </div>
                    <button
                      type="button"
                      class="mini-debug-copy-btn"
                      :disabled="debugToolSteps.length === 0"
                      @click="copyDebugToolSummary"
                    >
                      复制摘要
                    </button>
                  </header>
                  <div class="mini-debug-stats">
                    <span>{{ debugToolSteps.length }} 步</span>
                    <span>{{ debugSuccessCount }} 成功</span>
                    <span>{{ debugErrorCount }} 失败</span>
                  </div>
                  <div v-if="debugToolSteps.length" class="mini-debug-list">
                    <article
                      v-for="step in debugToolSteps"
                      :key="step.key"
                      class="mini-debug-step"
                      :class="`is-${step.statusClass}`"
                    >
                      <div class="mini-debug-step-title">
                        <span>第 {{ step.index }} 步</span>
                        <strong>{{ step.name }}</strong>
                        <em>{{ step.statusLabel }}</em>
                      </div>
                      <pre v-if="step.argumentsPreview" class="mini-debug-snippet">参数: {{ step.argumentsPreview }}</pre>
                      <pre v-if="step.outputPreview" class="mini-debug-snippet">输出: {{ step.outputPreview }}</pre>
                      <pre v-if="step.errorPreview" class="mini-debug-snippet mini-debug-snippet--error">错误: {{ step.errorPreview }}</pre>
                    </article>
                  </div>
                  <div v-else class="mini-debug-empty">暂无工具调用记录</div>
                </section>
              </div>
            </el-popover>
          </template>
        </MiniWorkstationComposer>
      </section>

      <section
        :class="['mini-session-center', { 'is-open': sessionCenterOpen }]"
        :aria-hidden="sessionCenterOpen ? 'false' : 'true'"
        @click.self="closeSessionCenter"
      >
        <div class="mini-session-dialog" role="dialog" aria-modal="true" aria-label="工作台会话中心">
          <header class="mini-session-dialog-head">
            <div class="mini-session-dialog-title">
              <strong>工作台会话</strong>
              <span>底部只保留活跃摘要，更多历史会话在这里集中管理。</span>
            </div>
            <button type="button" class="mini-session-close" @click="closeSessionCenter">
              <el-icon><Close /></el-icon>
            </button>
          </header>
          <div class="mini-session-dialog-tools">
            <div class="mini-session-tabs">
              <button type="button" :class="{ active: sessionScope === 'current' }" @click="setSessionScope('current')">
                当前目录 <span>{{ miniSessionList.length }}</span>
              </button>
              <button type="button" :class="{ active: sessionScope === 'all' }" @click="setSessionScope('all')">
                全部会话 <span>{{ globalSessionList.length || miniSessionList.length }}</span>
              </button>
            </div>
            <label class="mini-session-search">
              <el-icon :size="14"><Search /></el-icon>
              <input v-model="sessionSearchKeyword" placeholder="搜索目录、函数或需求..." />
            </label>
            <div class="mini-session-filters">
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
          </div>
          <div class="mini-session-list" v-loading="sessionCenterLoading">
            <button
              v-for="item in filteredSessionCenterList"
              :key="item.session_id"
              type="button"
              class="mini-session-row"
              @click="handleSessionCenterSelect(item)"
            >
              <span class="mini-status-dot" :class="getSessionStatusClass(item)"></span>
              <span class="mini-session-row-copy">
                <span class="mini-session-row-title">{{ getSessionTitle(item) }}</span>
                <span class="mini-session-row-sub">{{ getSessionCenterSubtitle(item) }}</span>
              </span>
              <span class="mini-session-row-meta">{{ getSessionStatusLabel(item) }} · {{ formatRelativeTime(item.updated_at || item.created_at) }}</span>
              <span class="mini-session-open">打开</span>
            </button>
            <div v-if="filteredSessionCenterList.length === 0 && !sessionCenterLoading" class="mini-session-empty">
              没有匹配的会话
            </div>
          </div>
        </div>
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

  <ScheduledAgentTaskDialog
    :key="scheduledDialogKey"
    v-model="showScheduledAgentTaskDialog"
    :full-code-path="fullCodePath"
    :initial-goal="scheduledDraftGoal"
    :initial-files="scheduledDraftFiles"
    :initial-l-l-m-config-id="selectedLLMConfigId"
    :task="null"
    @success="handleScheduledAgentTaskCreated"
  />
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Close, Minus, FullScreen, CopyDocument, UploadFilled, Document as DocumentIcon, Setting, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import {
  useWorkspaceChatStream,
  type AssistantBlock,
  type ChatMessage,
  type ChatMessageToolCall
} from '@/architecture/presentation/composables/useWorkspaceChatStream'
import MiniWorkstationDisplayFieldPreviewDialog from './MiniWorkstationDisplayFieldPreviewDialog.vue'
import MiniWorkstationComposer from './MiniWorkstationComposer.vue'
import MiniWorkstationKeyInfoSection from './MiniWorkstationKeyInfoSection.vue'
import MiniWorkstationMessages from './MiniWorkstationMessages.vue'
import ScheduledAgentTaskDialog from './ScheduledAgentTaskDialog.vue'
import { useLazyMarkdownRenderer } from '@/composables/useLazyMarkdownRenderer'
import { useMiniWorkstationPanel, type FilePanelItem } from '../composables/useMiniWorkstationPanel'
import { useMiniWorkstationSessions } from '../composables/useMiniWorkstationSessions'
import { useMiniWorkstationUploads } from '../composables/useMiniWorkstationUploads'
import { useMiniWorkstationComposer } from '../composables/useMiniWorkstationComposer'
import { useMiniWorkstationEffects } from '../composables/useMiniWorkstationEffects'
import { eventBus, WorkspaceEvent } from '@/architecture/infrastructure/eventBus'
import { createWorkspaceHandoff, type WorkspaceSessionItem } from '@/api/workspace'
import type { OutputDisplayField } from '@/architecture/presentation/composables/useOutputDisplayFields'

const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()

const props = defineProps<{
  visible: boolean
  fullCodePath: string
  dirName?: string
  initialSessionId?: string
  initialOffset?: number
  initialPosition?: 'center'
  initialMaximized?: boolean
}>()

const fullCodePathRef = computed(() => props.fullCodePath)
const initialSessionIdRef = computed(() => props.initialSessionId)

const emit = defineEmits<{
  (e: 'minimize'): void
  (e: 'close'): void
  (e: 'task-started', sessionId: string): void
  (e: 'tool-call-ok', payload: { name: string }): void
  (e: 'maximize-change', payload: { maximized: boolean; sessionId?: string }): void
}>()

const { messages, sending, sessionId, streamingDisplayLength, send: sendMessage, setMessages } = useWorkspaceChatStream()
const rootRef = ref<HTMLElement>()
const outputRef = ref<HTMLElement>()
const inputText = ref('')
const inputRef = ref<HTMLTextAreaElement>()
const showScheduledAgentTaskDialog = ref(false)
const scheduledDialogKey = ref(0)
const scheduledDraftGoal = ref('')
const scheduledDraftFiles = ref('')
const llmSelectOpen = ref(false)
const settingsPopoverOpen = ref(false)
const interactionOpen = computed(() => llmSelectOpen.value || settingsPopoverOpen.value)
const confirmedPrdKeys = ref<Set<string>>(new Set())
const confirmedTestHandoffKeys = ref<Set<string>>(new Set())
const collapsed = ref(false)
const sessionCenterOpen = ref(false)
const sessionSearchKeyword = ref('')
const sessionScope = ref<'current' | 'all'>('current')
const sessionFilter = ref<SessionFilterValue>('all')

const windowStyle = computed(() => ({
  '--mini-stack-offset': `${props.initialOffset || 0}px`
}))

function registerInputRef(element: HTMLTextAreaElement | null) {
  inputRef.value = element || undefined
}

const displayPath = computed(() => {
  if (!props.fullCodePath) return '未选择目录'
  const parts = props.fullCodePath.split('/').filter(Boolean)
  return parts[parts.length - 1] || props.fullCodePath
})

// 首条用户消息摘要，用于同目录多 Mini 时区分（如「分析数据」「帮我把xxx改成」）
const firstUserMessagePreview = computed(() => {
  const first = messages.value?.find(m => m.role === 'user')
  const content = typeof first?.content === 'string' ? first.content.trim() : ''
  if (!content) return ''
  const maxLen = 12
  return content.length > maxLen ? content.slice(0, maxLen) + '…' : content
})

// ─── 最大化 / 还原 ───
const maximized = ref(!!props.initialMaximized)

const {
  miniSessionList,
  globalSessionList,
  loadingSessions,
  loadingGlobalSessions,
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

function toggleMaximize() {
  if (maximized.value) {
    maximized.value = false
    // 从最大化恢复：若当前会话仍在执行中，重新开轮询兜底（可能连接已断）
    const cur = miniSessionList.value.find(s => s.session_id === sessionId.value)
    if (sessionId.value && cur?.status === 'generating') {
      startMiniStreamListening(sessionId.value)
      startMiniPoll(sessionId.value)
    }
    emit('maximize-change', { maximized: false })
  } else {
    maximized.value = true
    stopMiniPoll()
    emit('maximize-change', { maximized: true, sessionId: sessionId.value })
  }
}

// ─── 文件预览辅助 ───
const {
  getFileGroupsFromCalls,
  getDisplayFieldsFromCalls,
  keyInfoDropdownRef,
  allPanelDisplayFields,
  uploadedFiles,
  outputFiles,
  panelHasContent,
  panelItemCount,
  copyDisplayFieldValue,
  onKeyInfoDropdownVisibleChange,
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
  onDrop
} = useMiniWorkstationUploads({
  fullCodePath: fullCodePathRef,
  inputText,
  inputRef
})

type SessionFilterValue = 'all' | 'running' | 'waiting' | 'output' | 'done'

interface MiniArtifactItem {
  key: string
  name: string
  meta: string
  tag: string
  file?: FilePanelItem
  field?: OutputDisplayField
}

const sessionFilters: Array<{ label: string; value: SessionFilterValue }> = [
  { label: '全部', value: 'all' },
  { label: '执行中', value: 'running' },
  { label: '待确认', value: 'waiting' },
  { label: '有产出', value: 'output' },
  { label: '已完成', value: 'done' }
]

const currentSessionItem = computed(() => {
  if (!sessionId.value) return null
  return miniSessionList.value.find(item => item.session_id === sessionId.value)
    || globalSessionList.value.find(item => item.session_id === sessionId.value)
    || null
})

const activeSessionTitle = computed(() => {
  return currentSessionItem.value?.title
    || firstUserMessagePreview.value
    || props.dirName
    || displayPath.value
    || '新建会话'
})

const activeSessionStateLabel = computed(() => {
  if (pendingPrd.value || pendingTestHandoff.value) return '等待确认'
  if (sending.value) return '正在执行'
  if (currentSessionItem.value) return getSessionStatusLabel(currentSessionItem.value)
  return messages.value.length > 0 ? '当前输出' : '待输入'
})

const activeStatusClass = computed(() => {
  if (pendingPrd.value || pendingTestHandoff.value) return 'is-waiting'
  if (sending.value) return 'is-running'
  return currentSessionItem.value ? getSessionStatusClass(currentSessionItem.value) : 'is-draft'
})

const summarySessions = computed(() => {
  const active = currentSessionItem.value
  const list = active
    ? [active, ...miniSessionList.value.filter(item => item.session_id !== active.session_id)]
    : miniSessionList.value
  return list.slice(0, 4)
})

const summaryBadgeCount = computed(() => {
  const running = miniSessionList.value.filter(item => item.status === 'generating').length
  return Math.max(running, sending.value ? 1 : 0, queuedCount.value)
})

const collapsedSummaryText = computed(() => {
  if (sending.value) return '任务执行中'
  if (summaryBadgeCount.value > 0) return `${summaryBadgeCount.value} 个活跃会话`
  return `${miniSessionList.value.length || 1} 个会话`
})

const artifactItems = computed<MiniArtifactItem[]>(() => {
  const files = outputFiles.value.map((file, index) => ({
    key: `file:${file.href}:${index}`,
    name: file.name,
    meta: '输出文件',
    tag: getFileArtifactTag(file.name),
    file
  }))
  const fields = allPanelDisplayFields.value.map((field, index) => ({
    key: `field:${field.label}:${index}`,
    name: field.label,
    meta: truncateOneLine(field.value || '展示字段'),
    tag: '字段',
    field
  }))
  return [...files, ...fields]
})

const sessionCenterSourceList = computed(() => {
  if (sessionScope.value === 'all') {
    return globalSessionList.value.length > 0 ? globalSessionList.value : miniSessionList.value
  }
  return miniSessionList.value
})

const filteredSessionCenterList = computed(() => {
  const keyword = sessionSearchKeyword.value.trim().toLowerCase()
  return sessionCenterSourceList.value.filter((session) => {
    if (!matchesSessionFilter(session, sessionFilter.value)) return false
    if (!keyword) return true
    return [
      session.title,
      session.user,
      session.agent_name,
      session.role_display_name,
      session.full_code_path
    ].some(value => (value || '').toLowerCase().includes(keyword))
  })
})

const sessionCenterLoading = computed(() => {
  return sessionScope.value === 'all' ? loadingGlobalSessions.value : loadingSessions.value
})

function setCollapsed(value: boolean) {
  collapsed.value = value
  if (!value) {
    setTimeout(() => inputRef.value?.focus(), 80)
  }
}

async function openSessionCenter() {
  sessionCenterOpen.value = true
  await loadMiniSessions()
  if (sessionScope.value === 'all') {
    await loadGlobalSessions()
  }
}

function closeSessionCenter() {
  sessionCenterOpen.value = false
}

function setSessionScope(scope: 'current' | 'all') {
  sessionScope.value = scope
  if (scope === 'all') {
    void loadGlobalSessions()
  }
}

function handleSessionCenterSelect(session: WorkspaceSessionItem) {
  closeSessionCenter()
  if (session.full_code_path && session.full_code_path !== props.fullCodePath) {
    eventBus.emit('workspace:open-workstation', {
      full_code_path: session.full_code_path,
      session_id: session.session_id,
      open_as_mini: true
    })
    return
  }
  void handleSelectSession(session.session_id)
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

function getSessionTitle(session: WorkspaceSessionItem) {
  return session.title || session.role_display_name || '未命名会话'
}

function getSessionSubtitle(session: WorkspaceSessionItem) {
  return session.role_display_name || getSessionStatusLabel(session)
}

function getSessionCenterSubtitle(session: WorkspaceSessionItem) {
  const path = session.full_code_path || props.fullCodePath || ''
  return [path, session.role_display_name || session.user].filter(Boolean).join(' · ') || '当前目录'
}

function getSessionStatusLabel(session: WorkspaceSessionItem) {
  if (session.status === 'generating') return '执行中'
  if (session.status === 'done') return '已完成'
  if (session.status === 'cancelled') return '已取消'
  if (session.status === 'waiting' || session.status === 'pending') return '待确认'
  return session.status || '会话'
}

function getSessionStatusClass(session: WorkspaceSessionItem) {
  if (session.status === 'generating') return 'is-running'
  if (session.status === 'waiting' || session.status === 'pending') return 'is-waiting'
  if (session.status === 'done') return 'is-done'
  if (session.status === 'cancelled') return 'is-cancelled'
  return 'is-output'
}

function matchesSessionFilter(session: WorkspaceSessionItem, filter: SessionFilterValue) {
  if (filter === 'all') return true
  if (filter === 'running') return session.status === 'generating'
  if (filter === 'waiting') return session.status === 'waiting' || session.status === 'pending'
  if (filter === 'output') return !!session.handoff_kind || session.status === 'done'
  if (filter === 'done') return session.status === 'done' || session.status === 'cancelled'
  return true
}

function getFileArtifactTag(name: string) {
  const ext = (name || '').split('.').pop()?.toLowerCase() || ''
  if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg'].includes(ext)) return '图片'
  if (['csv', 'xlsx', 'xls', 'json'].includes(ext)) return '数据'
  if (['md', 'txt', 'doc', 'docx', 'pdf'].includes(ext)) return '文档'
  return '文件'
}

function truncateOneLine(value: string, max = 36) {
  const text = String(value || '').replace(/\s+/g, ' ').trim()
  return text.length > max ? `${text.slice(0, max)}…` : text
}

const currentAttachedFileRefs = computed(() => {
  return attachedFiles.value
    .map((file) => file.ref)
    .filter(Boolean)
    .join(',')
})

function openNewScheduledAgentTaskDialog() {
  scheduledDraftGoal.value = inputText.value.trim()
  scheduledDraftFiles.value = currentAttachedFileRefs.value
  scheduledDialogKey.value += 1
  showScheduledAgentTaskDialog.value = true
}

type CopyDebugMode = 'all' | 'last-turn' | 'all-tools' | 'error-tools' | 'success-tools'
interface DebugToolStep {
  key: string
  index: number
  name: string
  status: string
  statusLabel: string
  statusClass: 'running' | 'ok' | 'error' | 'default'
  argumentsPreview: string
  outputPreview: string
  errorPreview: string
  copyText: string
}

const DEBUG_HEAD_LINES = 10
const DEBUG_TAIL_LINES = 10
const DEBUG_SINGLE_LINE_LIMIT = 220

const debugToolSteps = computed<DebugToolStep[]>(() => {
  const steps: DebugToolStep[] = []
  for (const [messageIndex, message] of messages.value.entries()) {
    const calls = collectMessageToolCalls(message)
    calls.forEach((call, callIndex) => {
      const index = steps.length + 1
      const status = call.status || '-'
      const argumentsPreview = buildDebugPreview(call.arguments, true)
      const outputPreview = call.result
        ? buildDebugPreview(call.result)
        : call.result_data != null
          ? buildDebugPreview(call.result_data, true)
          : ''
      const errorPreview = buildDebugPreview(call.error)
      steps.push({
        key: `${messageIndex}-${callIndex}-${call.name || 'tool'}-${index}`,
        index,
        name: call.name || '(unknown)',
        status,
        statusLabel: getToolStatusLabel(status),
        statusClass: getToolStatusClass(status),
        argumentsPreview,
        outputPreview,
        errorPreview,
        copyText: formatDebugToolStepForCopy(index, call, argumentsPreview, outputPreview, errorPreview)
      })
    })
  }
  return steps
})

const debugSuccessCount = computed(() => debugToolSteps.value.filter(step => step.statusClass === 'ok').length)
const debugErrorCount = computed(() => debugToolSteps.value.filter(step => step.statusClass === 'error').length)

async function copyDebugConversation(mode: CopyDebugMode) {
  const text = buildDebugCopyText(mode)
  if (!text.trim()) {
    ElMessage.warning('当前没有可复制的调试内容')
    return
  }

  try {
    await copyTextToClipboard(text)
    ElMessage.success(getCopySuccessLabel(mode))
  } catch {
    ElMessage.error('复制失败')
  }
}

async function copyDebugToolSummary() {
  const text = buildDebugToolSummaryText()
  if (!text.trim()) {
    ElMessage.warning('当前没有工具调用记录')
    return
  }

  try {
    await copyTextToClipboard(text)
    ElMessage.success('已复制调用摘要')
  } catch {
    ElMessage.error('复制失败')
  }
}

function buildDebugToolSummaryText() {
  if (debugToolSteps.value.length === 0) return ''
  return [
    '# Mini 工具调用摘要',
    `目录: ${props.fullCodePath || '-'}`,
    `目录名: ${props.dirName || displayPath.value || '-'}`,
    `会话ID: ${sessionId.value || '-'}`,
    `工具调用: ${debugToolSteps.value.length} 步，成功 ${debugSuccessCount.value}，失败 ${debugErrorCount.value}`,
    `复制时间: ${new Date().toISOString()}`,
    '',
    debugToolSteps.value.map(step => step.copyText).join('\n\n')
  ].join('\n')
}

function formatDebugToolStepForCopy(
  index: number,
  call: ChatMessageToolCall,
  argumentsPreview: string,
  outputPreview: string,
  errorPreview: string
) {
  const parts = [`## 第 ${index} 步 ${call.name || '(unknown)'} [${getToolStatusLabel(call.status || '-')}]`]
  if (argumentsPreview) parts.push('', '参数:', fenceContent(argumentsPreview, 'json'))
  if (outputPreview) parts.push('', '输出摘要:', fenceContent(outputPreview))
  if (errorPreview) parts.push('', '错误摘要:', fenceContent(errorPreview))
  return parts.join('\n')
}

function buildDebugPreview(value: unknown, preferJson = false) {
  if (value == null) return ''
  const raw = typeof value === 'string'
    ? (preferJson ? formatMaybeJson(value) : formatLooseText(value))
    : formatJsonValue(value)
  return truncateDebugPreview(raw)
}

function truncateDebugPreview(value: string) {
  const text = String(value || '').trim()
  if (!text) return ''

  const lines = text.split(/\r?\n/)
  if (lines.length > DEBUG_HEAD_LINES + DEBUG_TAIL_LINES) {
    const omitted = lines.length - DEBUG_HEAD_LINES - DEBUG_TAIL_LINES
    return [
      ...lines.slice(0, DEBUG_HEAD_LINES),
      `... 省略 ${omitted} 行 ...`,
      ...lines.slice(-DEBUG_TAIL_LINES)
    ].join('\n')
  }

  if (lines.length === 1 && text.length > DEBUG_SINGLE_LINE_LIMIT) {
    const head = text.slice(0, 80)
    const tail = text.slice(-80)
    return `${head}\n... 省略 ${text.length - 160} 字符 ...\n${tail}`
  }

  return text
}

function getToolStatusLabel(status: string) {
  if (status === 'streaming') return '解析中'
  if (status === 'running') return '执行中'
  if (status === 'ok' || status === 'success') return '成功'
  if (status === 'error' || status === 'failed') return '失败'
  return status || '-'
}

function getToolStatusClass(status: string): DebugToolStep['statusClass'] {
  if (status === 'streaming' || status === 'running') return 'running'
  if (status === 'ok' || status === 'success') return 'ok'
  if (status === 'error' || status === 'failed') return 'error'
  return 'default'
}

function buildDebugCopyText(mode: CopyDebugMode) {
  const list = messages.value || []
  if (list.length === 0) return ''

  const header = [
    '# Mini 工作台调试对话',
    `目录: ${props.fullCodePath || '-'}`,
    `目录名: ${props.dirName || displayPath.value || '-'}`,
    `会话ID: ${sessionId.value || '-'}`,
    `复制范围: ${getCopyModeLabel(mode)}`,
    `复制时间: ${new Date().toISOString()}`,
    ''
  ].join('\n')

  if (mode === 'all') {
    return header + formatMessagesForCopy(list, { includeContent: true, includeToolCalls: true })
  }

  if (mode === 'last-turn') {
    return header + formatMessagesForCopy(getLastTurnMessages(list), { includeContent: true, includeToolCalls: true })
  }

  const statusFilter = getToolStatusFilter(mode)
  return header + formatMessagesWithToolFilter(list, statusFilter)
}

function getLastTurnMessages(list: ChatMessage[]) {
  const lastUserIndex = [...list].reverse().findIndex(item => item.role === 'user')
  if (lastUserIndex < 0) return list.slice(-1)
  const start = list.length - 1 - lastUserIndex
  return list.slice(start)
}

function getToolStatusFilter(mode: CopyDebugMode): ((call: ChatMessageToolCall) => boolean) | null {
  if (mode === 'error-tools') {
    return (call) => call.status === 'error' || call.status === 'failed'
  }
  if (mode === 'success-tools') {
    return (call) => call.status === 'ok' || call.status === 'success'
  }
  if (mode === 'all-tools') {
    return () => true
  }
  return null
}

function formatMessagesWithToolFilter(
  list: ChatMessage[],
  filter: ((call: ChatMessageToolCall) => boolean) | null
) {
  if (!filter) return ''

  const chunks: string[] = []
  let lastUser: ChatMessage | null = null
  for (const msg of list) {
    if (msg.role === 'user') {
      lastUser = msg
      continue
    }
    const calls = collectMessageToolCalls(msg).filter(filter)
    if (calls.length === 0) continue

    if (lastUser) {
      chunks.push(formatMessageForCopy(lastUser, { includeContent: true, includeToolCalls: false }))
    }
    chunks.push(formatMessageForCopy(msg, {
      includeContent: true,
      includeToolCalls: true,
      toolCallFilter: filter
    }))
    lastUser = null
  }

  return chunks.join('\n').trim()
}

function formatMessagesForCopy(
  list: ChatMessage[],
  options: {
    includeContent: boolean
    includeToolCalls: boolean
    toolCallFilter?: (call: ChatMessageToolCall) => boolean
  }
) {
  return list
    .map((message) => formatMessageForCopy(message, options))
    .filter(Boolean)
    .join('\n')
    .trim()
}

function formatMessageForCopy(
  message: ChatMessage,
  options: {
    includeContent: boolean
    includeToolCalls: boolean
    toolCallFilter?: (call: ChatMessageToolCall) => boolean
  }
) {
  const title = message.role === 'user' ? '## User' : '## Assistant'
  const meta = [
    message.user ? `用户: ${message.user}` : '',
    message.created_at ? `时间: ${message.created_at}` : ''
  ].filter(Boolean)

  const parts: string[] = [meta.length ? `${title} (${meta.join('，')})` : title]
  if (message.role === 'user') {
    if (options.includeContent && message.content) {
      parts.push('', message.content.trim())
    }
    if (message.files?.length) {
      parts.push('', '### 上传文件')
      for (const file of message.files) {
        parts.push(`- ${file.name}${file.ref ? ` (${file.ref})` : ''}`)
      }
    }
    return parts.join('\n').trim()
  }

  if (message.blocks?.length) {
    const blockText = formatAssistantBlocksForCopy(message.blocks, options)
    if (blockText) parts.push('', blockText)
  } else {
    if (options.includeContent && message.content) {
      parts.push('', message.content.trim())
    }
    if (options.includeToolCalls && message.tool_calls?.length) {
      const toolText = formatToolCallsForCopy(message.tool_calls, options.toolCallFilter)
      if (toolText) parts.push('', toolText)
    }
  }

  return parts.join('\n').trim()
}

function formatAssistantBlocksForCopy(
  blocks: AssistantBlock[],
  options: {
    includeContent: boolean
    includeToolCalls: boolean
    toolCallFilter?: (call: ChatMessageToolCall) => boolean
  }
) {
  const parts: string[] = []
  for (const block of blocks) {
    if (block.type === 'content' && options.includeContent && block.text.trim()) {
      parts.push(block.text.trim())
    }
    if (block.type === 'tool_calls' && options.includeToolCalls) {
      const toolText = formatToolCallsForCopy(block.calls, options.toolCallFilter)
      if (toolText) parts.push(toolText)
    }
  }
  return parts.join('\n\n').trim()
}

function collectMessageToolCalls(message: ChatMessage) {
  if (message.blocks?.length) {
    return message.blocks.flatMap((block) => block.type === 'tool_calls' ? block.calls : [])
  }
  return message.tool_calls || []
}

function formatToolCallsForCopy(
  calls: ChatMessageToolCall[],
  filter?: (call: ChatMessageToolCall) => boolean
) {
  const targetCalls = filter ? calls.filter(filter) : calls
  if (targetCalls.length === 0) return ''

  const parts: string[] = ['### 工具调用']
  targetCalls.forEach((call, index) => {
    parts.push('', `#### ${index + 1}. ${call.name || '(unknown)'} [${call.status || '-'}]`)
    if (call.arguments) {
      parts.push('', '参数:', fenceContent(formatMaybeJson(call.arguments), 'json'))
    }
    if (call.result) {
      parts.push('', '结果:', fenceContent(formatLooseText(call.result)))
    }
    if (call.result_data != null) {
      parts.push('', '结果数据:', fenceContent(formatJsonValue(call.result_data), 'json'))
    }
    if (call.error) {
      parts.push('', '错误:', fenceContent(formatLooseText(call.error)))
    }
  })
  return parts.join('\n')
}

function formatMaybeJson(value: string) {
  const text = formatLooseText(value)
  try {
    return JSON.stringify(JSON.parse(text), null, 2)
  } catch {
    return text
  }
}

function formatJsonValue(value: unknown) {
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

function formatLooseText(value: string) {
  return String(value || '').replace(/\\n/g, '\n').replace(/\\r/g, '\r').trim()
}

function fenceContent(value: string, lang = '') {
  const body = value || ''
  const fence = body.includes('```') ? '````' : '```'
  return `${fence}${lang}\n${body}\n${fence}`
}

async function copyTextToClipboard(text: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', 'true')
  textarea.style.position = 'fixed'
  textarea.style.left = '-9999px'
  document.body.appendChild(textarea)
  textarea.select()
  const ok = document.execCommand('copy')
  document.body.removeChild(textarea)
  if (!ok) throw new Error('copy failed')
}

function getCopyModeLabel(mode: CopyDebugMode) {
  const map: Record<CopyDebugMode, string> = {
    all: '全部对话',
    'last-turn': '最后一轮',
    'all-tools': '全部工具调用',
    'error-tools': '失败工具调用',
    'success-tools': '成功工具调用'
  }
  return map[mode]
}

function getCopySuccessLabel(mode: CopyDebugMode) {
  return `已复制${getCopyModeLabel(mode)}`
}

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
    emit('task-started', startedSessionId)
  },
  onToolCallOk: (payload) => {
    emit('tool-call-ok', payload)
  },
  onMaximizedSessionStarted: (startedSessionId) => {
    void loadMiniSessions()
    emit('maximize-change', { maximized: true, sessionId: startedSessionId })
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

function cancelPendingPrd() {
  if (!pendingPrd.value) return
  markPrdConfirmed(pendingPrd.value)
  ElMessage.info('已取消本次 PRD 确认')
}

function cancelPendingTestHandoff() {
  if (!pendingTestHandoff.value) return
  markTestHandoffHandled(pendingTestHandoff.value)
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
    }
  },
  { immediate: true }
)

function handleScheduledAgentTaskCreated() {
  eventBus.emit(WorkspaceEvent.scheduledAgentTaskCreated, { full_code_path: props.fullCodePath })
}

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

:global(.mini-settings-popover.el-popper) {
  padding: 0;
  border: 1px solid rgba(96, 231, 255, 0.22);
  border-radius: 14px;
  background:
    radial-gradient(circle at 14% 0%, rgba(34, 211, 238, 0.16), transparent 36%),
    linear-gradient(150deg, rgba(5, 16, 30, 0.98), rgba(8, 27, 45, 0.97));
  box-shadow: 0 20px 54px rgba(0, 0, 0, 0.42), 0 0 28px rgba(34, 211, 238, 0.14);
  backdrop-filter: blur(18px) saturate(1.16);
}

:global(.mini-settings-popover .el-popper__arrow::before) {
  border-color: rgba(96, 231, 255, 0.22);
  background: rgba(5, 16, 30, 0.98);
}

.mini-settings-panel {
  max-height: min(520px, calc(100vh - 110px));
  overflow: auto;
  color: var(--mini-cyber-text);
}

.mini-settings-section + .mini-settings-section {
  border-top: 1px solid rgba(96, 231, 255, 0.14);
}

.mini-settings-section-title {
  padding: 12px 12px 0;
  color: var(--mini-cyber-accent);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.mini-settings-copy-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  padding: 10px 12px 12px;
}

.mini-settings-copy-grid button {
  min-width: 0;
  height: 32px;
  border: 1px solid rgba(96, 231, 255, 0.18);
  border-radius: 9px;
  background: rgba(34, 211, 238, 0.08);
  color: var(--mini-cyber-muted);
  cursor: pointer;
  font-size: 12px;
}

.mini-settings-copy-grid button:hover {
  border-color: rgba(96, 231, 255, 0.34);
  background: rgba(34, 211, 238, 0.14);
  color: #ffffff;
  box-shadow: inset 0 0 0 1px rgba(34, 211, 238, 0.16);
}

.mini-debug-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  border-bottom: 1px solid rgba(96, 231, 255, 0.16);
  background: rgba(34, 211, 238, 0.055);
}

.mini-debug-head strong {
  display: block;
  margin-top: 3px;
  font-size: 14px;
}

.mini-debug-kicker {
  display: block;
  color: var(--mini-cyber-accent);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.mini-debug-copy-btn {
  height: 28px;
  padding: 0 10px;
  border: 1px solid rgba(96, 231, 255, 0.26);
  border-radius: 9px;
  background: rgba(34, 211, 238, 0.11);
  color: var(--mini-cyber-text);
  font-size: 12px;
  cursor: pointer;
}

.mini-debug-copy-btn:hover:not(:disabled) {
  border-color: rgba(96, 231, 255, 0.46);
  background: rgba(34, 211, 238, 0.17);
}

.mini-debug-copy-btn:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

.mini-debug-stats {
  display: flex;
  gap: 7px;
  padding: 9px 12px;
  border-bottom: 1px solid rgba(96, 231, 255, 0.12);
}

.mini-debug-stats span {
  padding: 3px 7px;
  border: 1px solid rgba(96, 231, 255, 0.14);
  border-radius: 999px;
  background: rgba(34, 211, 238, 0.06);
  color: var(--mini-cyber-muted);
  font-size: 11px;
}

.mini-debug-list {
  max-height: 390px;
  overflow: auto;
  padding: 10px 12px 12px;
}

.mini-debug-list::-webkit-scrollbar {
  width: 7px;
}

.mini-debug-list::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background: rgba(96, 231, 255, 0.26);
  background-clip: padding-box;
}

.mini-debug-step {
  padding: 9px;
  border: 1px solid rgba(96, 231, 255, 0.14);
  border-radius: 12px;
  background:
    linear-gradient(145deg, rgba(8, 22, 38, 0.7), rgba(3, 10, 20, 0.62)),
    rgba(34, 211, 238, 0.04);
}

.mini-debug-step + .mini-debug-step {
  margin-top: 8px;
}

.mini-debug-step.is-ok {
  border-color: rgba(103, 194, 58, 0.28);
}

.mini-debug-step.is-error {
  border-color: rgba(245, 108, 108, 0.34);
}

.mini-debug-step.is-running {
  border-color: rgba(230, 162, 60, 0.3);
}

.mini-debug-step-title {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 7px;
  margin-bottom: 7px;
}

.mini-debug-step-title span {
  flex: 0 0 auto;
  color: var(--mini-cyber-dim);
  font-size: 11px;
}

.mini-debug-step-title strong {
  min-width: 0;
  overflow: hidden;
  color: var(--mini-cyber-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-debug-step-title em {
  flex: 0 0 auto;
  margin-left: auto;
  color: var(--mini-cyber-accent);
  font-size: 11px;
  font-style: normal;
}

.mini-debug-snippet {
  max-height: 190px;
  margin: 6px 0 0;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  padding: 7px 8px;
  border: 1px solid rgba(96, 231, 255, 0.12);
  border-radius: 9px;
  background: rgba(2, 8, 18, 0.46);
  color: var(--mini-cyber-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  line-height: 1.45;
}

.mini-debug-snippet--error {
  border-color: rgba(245, 108, 108, 0.24);
  color: #ffc7c7;
}

.mini-debug-empty {
  padding: 42px 16px;
  color: var(--mini-cyber-dim);
  font-size: 13px;
  text-align: center;
}

/* ── 标题栏文件下拉（不遮挡内容区） ── */
.mini-files-dropdown-panel {
  min-width: 260px;
  max-width: 320px;
  color: var(--mini-cyber-text);
  background:
    radial-gradient(circle at 12% 0%, rgba(34, 211, 238, 0.18), transparent 34%),
    linear-gradient(150deg, rgba(6, 18, 33, 0.98), rgba(11, 30, 49, 0.96));
  border: 1px solid rgba(96, 231, 255, 0.22);
  border-radius: 14px;
  box-shadow: 0 18px 46px rgba(0, 0, 0, 0.36), 0 0 24px rgba(34, 211, 238, 0.1);
  overflow: hidden;
}
.mini-files-dropdown-title {
  padding: 11px 12px;
  font-size: 13px;
  font-weight: 700;
  color: var(--mini-cyber-text);
  border-bottom: 1px solid rgba(96, 231, 255, 0.16);
  background: rgba(34, 211, 238, 0.06);
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

.mini-prd-confirm-bar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-top: 1px solid rgba(246, 199, 107, 0.28);
  border-bottom: 1px solid rgba(96, 231, 255, 0.12);
  background:
    linear-gradient(90deg, rgba(246, 199, 107, 0.16), rgba(34, 211, 238, 0.08)),
    rgba(5, 16, 30, 0.88);
  box-shadow: 0 -10px 28px rgba(0, 0, 0, 0.18);
}

.mini-prd-confirm-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.mini-prd-confirm-copy strong {
  color: #ffe4a3;
  font-size: 13px;
  line-height: 1.2;
}

.mini-prd-confirm-copy span {
  max-width: 560px;
  overflow: hidden;
  color: var(--mini-cyber-muted);
  font-size: 12px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-prd-confirm-actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.mini-prd-confirm-actions :deep(.el-button) {
  margin-left: 0;
}

.mini-ws:not(.mini-ws--maximized) .mini-prd-confirm-bar {
  align-items: stretch;
  flex-direction: column;
  gap: 8px;
}

.mini-ws:not(.mini-ws--maximized) .mini-prd-confirm-copy span {
  max-width: 100%;
}

.mini-ws:not(.mini-ws--maximized) .mini-prd-confirm-actions {
  justify-content: flex-end;
  flex-wrap: wrap;
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

.mini-shell {
  position: absolute;
  left: max(90px, 12vw);
  right: 86px;
  bottom: calc(24px + var(--mini-stack-offset, 0px));
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: var(--mini-cyber-text);
  pointer-events: auto;
  transition: left 0.18s ease, right 0.18s ease, bottom 0.18s ease;
}

.mini-ws--maximized .mini-shell {
  left: 42px;
  right: 42px;
}

.mini-current-output,
.mini-session-dock,
.mini-shell :deep(.mini-ws-input),
.mini-prd-confirm-bar {
  border: 1px solid var(--mini-cyber-line);
  background:
    linear-gradient(180deg, rgba(12, 18, 32, 0.84), rgba(8, 12, 22, 0.68)),
    rgba(8, 12, 22, 0.72);
  box-shadow: 0 24px 70px rgba(0, 0, 0, 0.42), inset 0 1px 0 rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(24px) saturate(140%);
}

.mini-current-output {
  min-height: 46px;
  max-height: 118px;
  display: flex;
  flex-direction: column;
  margin: 0 14px;
  border-color: rgba(104, 119, 255, 0.28);
  border-radius: 12px;
  overflow: hidden;
  transition: min-height 0.18s ease, max-height 0.18s ease;
}

.mini-ws--maximized .mini-current-output {
  height: min(540px, calc(100vh - 268px));
  min-height: 360px;
  max-height: none;
}

.mini-current-head {
  min-height: 39px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 7px 12px 5px;
  color: #b9c9e4;
  font-size: 12px;
}

.mini-current-title,
.mini-current-actions {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.mini-current-name {
  min-width: 0;
  overflow: hidden;
  color: #88d6ff;
  font-weight: 850;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-current-state {
  flex-shrink: 0;
  color: var(--mini-cyber-muted);
}

.mini-current-actions {
  flex-shrink: 0;
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

.mini-current-body {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 230px;
  gap: 12px;
  padding: 0 12px 10px;
}

.mini-ws--maximized .mini-current-body {
  grid-template-columns: minmax(0, 1fr) 288px;
}

.mini-ws-output {
  min-height: 0;
  height: 36px;
  overflow: hidden;
  padding: 0;
  background: transparent;
  color: #d7e5fa;
  font-size: 13px;
  line-height: 18px;
  scrollbar-color: rgba(83, 174, 255, 0.3) transparent;
}

.mini-ws:not(.mini-ws--maximized):not(.mini-ws--interaction-open):not(:hover):not(:focus-within) .mini-ws-output {
  padding: 0;
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
  margin-bottom: 6px;
}

.mini-ws:not(.mini-ws--maximized) .mini-ws-output :deep(.mini-msg:not(:last-child)) {
  display: none;
}

.mini-artifact-panel {
  min-height: 36px;
  overflow: hidden;
  padding-left: 12px;
  border-left: 1px solid rgba(130, 153, 190, 0.18);
}

.mini-ws--maximized .mini-artifact-panel {
  overflow: auto;
  padding: 12px;
  border: 1px solid rgba(130, 153, 190, 0.18);
  border-radius: 12px;
  background: rgba(12, 20, 35, 0.48);
}

.mini-artifact-head {
  display: none;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  color: var(--mini-cyber-muted);
  font-size: 12px;
}

.mini-ws--maximized .mini-artifact-head {
  display: flex;
}

.mini-artifact-head strong {
  color: #8ed0ff;
}

.mini-artifact-item {
  width: 100%;
  height: 32px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border: 1px solid rgba(83, 174, 255, 0.18);
  border-radius: 8px;
  background: rgba(14, 27, 45, 0.62);
  color: inherit;
  text-align: left;
}

.mini-artifact-item:hover {
  border-color: rgba(83, 174, 255, 0.42);
  background: rgba(24, 51, 83, 0.52);
}

.mini-artifact-item + .mini-artifact-item {
  display: none;
}

.mini-ws--maximized .mini-artifact-item {
  height: auto;
  min-height: 58px;
  padding: 10px;
  margin-bottom: 8px;
}

.mini-ws--maximized .mini-artifact-item + .mini-artifact-item {
  display: grid;
}

.mini-artifact-copy,
.mini-artifact-name,
.mini-artifact-meta {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-artifact-name {
  color: #dce9fb;
  font-size: 12px;
  font-weight: 760;
}

.mini-artifact-meta {
  margin-top: 2px;
  color: #8a9ab6;
  font-size: 11px;
}

.mini-artifact-tag {
  height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 7px;
  border-radius: 7px;
  background: rgba(43, 213, 159, 0.14);
  color: #7df5c4;
  font-size: 11px;
  font-weight: 800;
  white-space: nowrap;
}

.mini-artifact-mini-count {
  margin-left: 6px;
  padding-left: 6px;
  border-left: 1px solid rgba(125, 245, 196, 0.24);
  color: #dff7ef;
}

.mini-artifact-empty {
  height: 32px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 0 8px;
  border: 1px solid rgba(83, 174, 255, 0.14);
  border-radius: 8px;
  background: rgba(14, 27, 45, 0.42);
  color: #8a9ab6;
  font-size: 12px;
}

.mini-artifact-empty em {
  border-radius: 7px;
  padding: 3px 7px;
  background: rgba(142, 159, 187, 0.12);
  color: #9fb0cb;
  font-style: normal;
  font-weight: 800;
}

.mini-session-dock {
  min-height: 54px;
  display: grid;
  grid-template-columns: 132px minmax(0, 1fr);
  gap: 12px;
  margin: 0 14px;
  padding: 6px;
  border-radius: 14px;
  background: rgba(9, 14, 25, 0.68);
}

.mini-session-center-btn {
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 0 12px;
  border: 1px solid rgba(83, 174, 255, 0.44);
  border-radius: 12px;
  background:
    linear-gradient(135deg, rgba(34, 113, 205, 0.34), rgba(119, 107, 255, 0.18)),
    rgba(12, 22, 38, 0.78);
  color: #dff1ff;
  box-shadow: 0 12px 30px rgba(37, 110, 194, 0.2);
  font-size: 12px;
  font-weight: 800;
}

.mini-session-summary-list {
  min-width: 0;
  display: grid;
  grid-template-columns: repeat(4, minmax(112px, 1fr));
  gap: 8px;
}

.mini-session-summary-card {
  position: relative;
  width: 100%;
  height: 42px;
  min-width: 0;
  display: grid;
  grid-template-columns: 10px minmax(0, 1fr) auto;
  align-items: center;
  gap: 7px;
  padding: 0 10px;
  border: 1px solid rgba(126, 151, 197, 0.2);
  border-radius: 10px;
  background: rgba(30, 42, 68, 0.5);
  color: #d7e5fa;
  text-align: left;
}

.mini-session-summary-card:hover,
.mini-session-summary-card.active {
  border-color: rgba(87, 182, 255, 0.5);
  background: rgba(24, 51, 83, 0.62);
}

.mini-session-summary-card.active::before {
  content: "▼";
  position: absolute;
  left: 50%;
  top: -20px;
  transform: translateX(-50%);
  color: #8ed0ff;
  font-size: 16px;
  line-height: 1;
  text-shadow: 0 0 16px rgba(83, 174, 255, 0.72);
  pointer-events: none;
}

.mini-session-summary-card.is-running {
  border-color: rgba(43, 213, 159, 0.28);
  background: rgba(21, 54, 50, 0.42);
}

.mini-session-summary-card.is-waiting {
  border-color: rgba(246, 189, 77, 0.3);
  background: rgba(58, 45, 24, 0.46);
}

.mini-session-summary-card.is-done {
  border-color: rgba(119, 107, 255, 0.28);
  background: rgba(41, 38, 76, 0.46);
}

.mini-session-summary-card.is-output {
  border-color: rgba(55, 163, 255, 0.3);
  background: rgba(24, 48, 77, 0.46);
}

.mini-session-summary-copy,
.mini-session-summary-title,
.mini-session-summary-sub {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-session-summary-title {
  color: #d7e5fa;
  font-size: 12px;
  font-weight: 780;
  line-height: 1.15;
}

.mini-session-summary-sub {
  margin-top: 2px;
  color: #8596b2;
  font-size: 10px;
  line-height: 1.1;
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

.mini-count-badge {
  min-width: 18px;
  height: 18px;
  display: inline-grid;
  place-items: center;
  padding: 0 5px;
  border-radius: 999px;
  background: rgba(255, 109, 126, 0.9);
  color: #fff;
  font-size: 11px;
  font-weight: 900;
}

.mini-collapsed-launcher {
  position: absolute;
  left: 50%;
  bottom: calc(18px + var(--mini-stack-offset, 0px));
  z-index: 2;
  transform: translateX(-50%);
  height: 46px;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 0 16px;
  border: 1px solid rgba(130, 153, 190, 0.3);
  border-radius: 999px;
  background: rgba(10, 16, 29, 0.78);
  box-shadow: 0 24px 70px rgba(0, 0, 0, 0.42);
  backdrop-filter: blur(24px);
  color: #dce9fb;
  pointer-events: auto;
}

.mini-collapsed-launcher.has-update {
  border-color: rgba(83, 174, 255, 0.5);
  box-shadow:
    0 24px 70px rgba(0, 0, 0, 0.42),
    0 0 0 1px rgba(83, 174, 255, 0.16),
    0 0 26px rgba(83, 174, 255, 0.2);
}

.mini-collapsed-pulse {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--mini-cyber-green);
  box-shadow: 0 0 18px rgba(43, 213, 159, 0.8);
}

.mini-session-center {
  position: fixed;
  inset: 0;
  z-index: 42;
  display: grid;
  place-items: center;
  padding: 56px;
  background: rgba(2, 5, 11, 0.42);
  backdrop-filter: blur(7px);
  opacity: 0;
  visibility: hidden;
  pointer-events: none;
  transition: opacity 0.16s ease, visibility 0.16s ease;
}

.mini-session-center.is-open {
  opacity: 1;
  visibility: visible;
  pointer-events: auto;
}

.mini-session-dialog {
  width: min(920px, calc(100vw - 96px));
  height: min(620px, calc(100vh - 126px));
  display: grid;
  grid-template-rows: auto auto 1fr;
  overflow: hidden;
  border: 1px solid rgba(130, 153, 190, 0.3);
  border-radius: 18px;
  background:
    linear-gradient(180deg, rgba(15, 23, 42, 0.94), rgba(8, 13, 24, 0.9)),
    rgba(10, 16, 29, 0.9);
  box-shadow: 0 34px 100px rgba(0, 0, 0, 0.52);
  color: var(--mini-cyber-text);
}

.mini-session-dialog-head {
  height: 66px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 18px 0 22px;
  border-bottom: 1px solid rgba(130, 153, 190, 0.18);
}

.mini-session-dialog-title {
  display: grid;
  gap: 4px;
}

.mini-session-dialog-title strong {
  font-size: 17px;
}

.mini-session-dialog-title span {
  color: var(--mini-cyber-muted);
  font-size: 12px;
}

.mini-session-close {
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border: 1px solid rgba(124, 146, 189, 0.24);
  border-radius: 10px;
  background: rgba(30, 42, 68, 0.72);
  color: #d7e5fa;
}

.mini-session-dialog-tools {
  display: grid;
  grid-template-columns: auto minmax(180px, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 14px 18px;
  border-bottom: 1px solid rgba(130, 153, 190, 0.14);
}

.mini-session-tabs,
.mini-session-filters {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mini-session-tabs button,
.mini-session-filters button {
  height: 34px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  border: 1px solid rgba(124, 146, 189, 0.22);
  border-radius: 10px;
  background: rgba(30, 42, 68, 0.56);
  color: #b9c9e4;
  font-size: 12px;
  white-space: nowrap;
}

.mini-session-tabs button.active,
.mini-session-filters button.active {
  border-color: rgba(83, 174, 255, 0.46);
  background: rgba(34, 113, 205, 0.2);
  color: #8ed0ff;
}

.mini-session-search {
  height: 36px;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  border: 1px solid rgba(124, 146, 189, 0.2);
  border-radius: 10px;
  background: rgba(10, 16, 29, 0.5);
  color: #8e9fbb;
}

.mini-session-search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  color: #e6f0ff;
  font: inherit;
  font-size: 13px;
}

.mini-session-search input::placeholder {
  color: #7586a4;
}

.mini-session-list {
  overflow: auto;
  padding: 12px 18px 18px;
}

.mini-session-row {
  width: 100%;
  min-height: 68px;
  display: grid;
  grid-template-columns: 10px minmax(0, 1fr) auto auto;
  gap: 12px;
  align-items: center;
  margin-bottom: 10px;
  padding: 12px;
  border: 1px solid rgba(126, 151, 197, 0.18);
  border-radius: 12px;
  background: rgba(17, 25, 45, 0.62);
  color: #d7e5fa;
  text-align: left;
}

.mini-session-row:hover {
  border-color: rgba(83, 174, 255, 0.42);
  background: rgba(24, 51, 83, 0.48);
}

.mini-session-row-copy,
.mini-session-row-title,
.mini-session-row-sub {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-session-row-title {
  font-size: 14px;
  font-weight: 820;
}

.mini-session-row-sub {
  margin-top: 5px;
  color: #8798b5;
  font-size: 12px;
}

.mini-session-row-meta {
  color: #9fb0cb;
  font-size: 12px;
  white-space: nowrap;
}

.mini-session-open {
  height: 32px;
  display: inline-flex;
  align-items: center;
  padding: 0 12px;
  border: 1px solid rgba(83, 174, 255, 0.32);
  border-radius: 9px;
  background: rgba(34, 113, 205, 0.18);
  color: #8ed0ff;
  font-size: 12px;
}

.mini-session-empty {
  padding: 46px 0;
  color: var(--mini-cyber-muted);
  text-align: center;
  font-size: 13px;
}

.mini-prd-confirm-bar {
  margin: 0 14px;
  border-radius: 12px;
  border-color: rgba(246, 189, 77, 0.28);
  background:
    linear-gradient(90deg, rgba(246, 189, 77, 0.16), rgba(55, 163, 255, 0.08)),
    rgba(8, 13, 24, 0.88);
}

.mini-settings-btn {
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

.mini-files-dropdown-panel {
  background:
    radial-gradient(circle at 12% 0%, rgba(55, 163, 255, 0.16), transparent 34%),
    linear-gradient(150deg, rgba(8, 13, 24, 0.98), rgba(17, 25, 45, 0.96));
  border-color: rgba(130, 153, 190, 0.24);
}

.mini-ws-drop-overlay {
  inset: auto 86px calc(24px + var(--mini-stack-offset, 0px)) max(90px, 12vw);
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

  .mini-session-summary-list {
    grid-template-columns: repeat(4, minmax(96px, 1fr));
  }

  .mini-current-body {
    grid-template-columns: minmax(0, 1fr) 210px;
  }
}

@media (max-width: 820px) {
  .mini-shell,
  .mini-ws--maximized .mini-shell {
    left: 12px;
    right: 12px;
    bottom: 12px;
  }

  .mini-current-body,
  .mini-ws--maximized .mini-current-body {
    grid-template-columns: 1fr;
  }

  .mini-artifact-panel {
    display: none;
  }

  .mini-session-dock {
    grid-template-columns: 1fr;
  }

  .mini-session-center-btn {
    justify-content: flex-start;
  }

  .mini-session-summary-list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .mini-session-dialog-tools,
  .mini-session-row {
    grid-template-columns: 1fr;
  }

  .mini-session-dialog {
    width: calc(100vw - 24px);
    height: calc(100vh - 72px);
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
