<!--
  MiniWorkstation - 迷你浮动工作台
  右下角弹出的小窗口，支持输入命令、上传文件、SSE 实时输出、最小化。
-->
<template>
  <transition name="mini-ws-pop">
    <div
      v-if="visible && !collapsed"
      ref="rootRef"
      :class="['mini-ws', { 'mini-ws--maximized': maximized, 'mini-ws--sending': sending, 'mini-ws--interaction-open': interactionOpen }]"
      data-testid="mini-workstation"
      :data-full-code-path="fullCodePath"
      :style="windowStyle"
      @dragover.prevent="onDragOver"
      @dragleave.prevent="onDragLeave"
      @drop.prevent="onDrop"
    >
      <div class="mini-workspace-backdrop" aria-hidden="true"></div>
      <section class="mini-shell">
        <section v-if="showCurrentOutput" class="mini-current-output">
          <div class="mini-current-layout">
            <aside class="mini-current-meta">
              <header class="mini-current-session-head">
                <div>
                  <strong>当前目录会话</strong>
                  <span :title="fullCodePath">{{ dirName || displayPath }}</span>
                </div>
                <em>{{ currentOutputSessionList.length }}</em>
              </header>
              <div class="mini-current-session-list">
                <button
                  v-for="item in currentOutputSessionList"
                  :key="item.session_id"
                  type="button"
                  :class="['mini-current-session-row', getSessionStatusClass(item), { active: item.session_id === sessionId }]"
                  :title="getSessionTitle(item)"
                  @click="handleCurrentOutputSessionSelect(item)"
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
                  v-if="currentOutputSessionList.length === 0"
                  type="button"
                  class="mini-current-session-row active is-draft"
                  @click="startNewSession"
                >
                  <span class="mini-status-dot"></span>
                  <span class="mini-current-session-copy">
                    <span class="mini-current-session-title">新建会话</span>
                    <span class="mini-current-session-sub">当前目录 · 待输入</span>
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
            <aside class="mini-artifact-panel" :class="{ 'is-empty': artifactItems.length === 0 }" aria-label="当前产物">
              <div class="mini-artifact-head">
                <span>{{ maximized ? '当前产物' : '产物' }}</span>
                <strong>{{ artifactItems.length }} 项</strong>
                <span class="mini-artifact-actions">
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
                </span>
              </div>
            <button
              v-for="item in artifactItems"
              :key="item.key"
              type="button"
              class="mini-artifact-item"
              :class="`is-${item.tone}`"
              @click="handleArtifactClick(item)"
            >
              <span class="mini-artifact-preview" :class="`is-${item.tone}`">
                <img v-if="item.previewUrl" :src="item.previewUrl" :alt="item.name" loading="lazy" />
                <template v-else>
                  <el-icon :size="maximized ? 22 : 16">
                    <component :is="item.iconComponent" />
                  </el-icon>
                  <span v-if="item.ext" class="mini-artifact-ext">{{ item.ext }}</span>
                </template>
              </span>
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
            <span class="mini-count-badge">{{ recentSessionSourceList.length || miniSessionList.length }}</span>
            <span>会话中心</span>
          </button>
          <button type="button" class="mini-session-new-btn" title="新建会话" @click="startNewSession">
            <el-icon :size="17"><Plus /></el-icon>
          </button>
          <div class="mini-session-summary-list">
            <button
              v-if="summarySessions.length === 0"
              type="button"
              class="mini-session-summary-card active is-draft"
              @click="startNewSession"
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
              @click="handleSummarySessionSelect(item)"
            >
              <span class="mini-status-dot" :class="getSessionStatusClass(item)"></span>
              <span class="mini-session-summary-copy">
                <span class="mini-session-summary-title">{{ getSessionTitle(item) }}</span>
                <span class="mini-session-summary-sub">{{ getSessionSubtitle(item) }}</span>
              </span>
              <span v-if="getSessionStatusKind(item) === 'running'" class="mini-count-badge">•</span>
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
          :show-schedule-action="featureFlags.scheduledTasks"
          :register-input-ref="registerInputRef"
          :on-l-l-m-select-visible-change="onLLMSelectVisibleChange"
          :on-file-change="onFileChange"
          :remove-file="removeFile"
          :on-input-enter="onInputEnter"
          :toggle-shortcut-label="toggleShortcutLabel"
          @update:input-text="inputText = $event"
          @update:selected-l-l-m-config-id="selectedLLMConfigId = $event"
          @schedule="openNewScheduledAgentTaskDialog"
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
              <span>左侧是当前目录会话，右侧是跨目录最近会话。</span>
            </div>
            <button type="button" class="mini-session-close" @click="closeSessionCenter">
              <el-icon><Close /></el-icon>
            </button>
          </header>
          <div class="mini-session-dialog-tools">
            <div class="mini-session-dialog-stat">
              <span>当前目录 {{ currentDirectorySessionList.length }}/{{ miniSessionList.length }}</span>
              <span>最近会话 {{ recentSessionCenterList.length }}/{{ recentSessionCenterSourceList.length }}</span>
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
          <div class="mini-session-columns">
            <section class="mini-session-pane mini-session-pane--current" v-loading="loadingSessions">
              <header class="mini-session-pane-head">
                <div>
                  <strong>当前目录</strong>
                  <span :title="fullCodePath">{{ dirName || displayPath }}</span>
                </div>
                <em>{{ currentDirectorySessionList.length }}</em>
              </header>
              <div class="mini-session-list">
                <button
                  v-for="item in currentDirectorySessionList"
                  :key="item.session_id"
                  type="button"
                  :class="['mini-session-row', getSessionStatusClass(item), { active: item.session_id === sessionId }]"
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
                <div v-if="currentDirectorySessionList.length === 0 && !loadingSessions" class="mini-session-empty">
                  没有匹配的当前目录会话
                </div>
              </div>
            </section>

            <section class="mini-session-pane mini-session-pane--recent" v-loading="loadingGlobalSessions">
              <header class="mini-session-pane-head">
                <div>
                  <strong>最近会话</strong>
                  <span>可打开其他目录的工作台会话</span>
                </div>
                <em>{{ recentSessionCenterList.length }}</em>
              </header>
              <div class="mini-session-list">
                <button
                  v-for="item in recentSessionCenterList"
                  :key="item.session_id"
                  type="button"
                  :class="['mini-session-row', getSessionStatusClass(item), { active: item.session_id === sessionId }]"
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
                <div v-if="recentSessionCenterList.length === 0 && !loadingGlobalSessions" class="mini-session-empty">
                  没有匹配的最近会话
                </div>
              </div>
            </section>
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
    v-if="featureFlags.scheduledTasks"
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
import { nextTick, ref, computed, watch } from 'vue'
import {
  Close,
  Plus,
  UploadFilled,
  Document as DocumentIcon,
  Setting,
  Search
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import {
  useWorkspaceChatStream,
  type ChatMessage
} from '@/architecture/presentation/composables/useWorkspaceChatStream'
import MiniWorkstationDisplayFieldPreviewDialog from './MiniWorkstationDisplayFieldPreviewDialog.vue'
import MiniWorkstationComposer from './MiniWorkstationComposer.vue'
import MiniWorkstationKeyInfoSection from './MiniWorkstationKeyInfoSection.vue'
import MiniWorkstationMessages from './MiniWorkstationMessages.vue'
import ScheduledAgentTaskDialog from './ScheduledAgentTaskDialog.vue'
import { useLazyMarkdownRenderer } from '@/composables/useLazyMarkdownRenderer'
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
import { eventBus, WorkspaceEvent } from '@/architecture/infrastructure/eventBus'
import { createWorkspaceHandoff, resolveWorkspaceSessionInteraction, type WorkspaceSessionItem } from '@/api/workspace'
import { featureFlags } from '@/config/features'

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
  toggleShortcutLabel?: string
}>()

const fullCodePathRef = computed(() => props.fullCodePath)
const initialSessionIdRef = computed(() => props.initialSessionId)

const emit = defineEmits<{
  (e: 'minimize'): void
  (e: 'close'): void
  (e: 'task-started', sessionId: string): void
  (e: 'tool-call-ok', payload: { name: string }): void
  (e: 'expanded-change', payload: { expanded: boolean; sessionId?: string }): void
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
const collapsed = ref(props.initialExpanded === false)
const suppressAutoSelectLatestSession = ref(false)
const sessionCenterOpen = ref(false)
const sessionSearchKeyword = ref('')
const sessionFilter = ref<SessionFilterValue>('all')

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

const displayPath = computed(() => {
  if (!props.fullCodePath) return '未选择目录'
  return resolvePathDisplayName(props.fullCodePath)
})

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
type SessionStatusKind = 'running' | 'waiting' | 'done' | 'cancelled' | 'failed' | 'active' | 'output'

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

const currentFallbackSession = computed<WorkspaceSessionItem | null>(() => {
  if (!sessionId.value || currentSessionItem.value) return null
  const now = new Date().toISOString()
  return {
    session_id: sessionId.value,
    title: firstUserMessagePreview.value || props.dirName || displayPath.value || '新建会话',
    status: sending.value ? 'generating' : 'active',
    full_code_path: props.fullCodePath,
    directory_name: props.dirName || displayPath.value,
    role_display_name: props.dirName || displayPath.value,
    created_at: now,
    updated_at: now
  }
})

const activeSummarySession = computed(() => currentSessionItem.value || currentFallbackSession.value)

const currentOutputSessionList = computed(() => {
  const currentPath = normalizeFullCodePath(props.fullCodePath)
  const active = activeSummarySession.value
  const seenIds = new Set<string>()
  const list: WorkspaceSessionItem[] = []

  const addIfCurrentPath = (session: WorkspaceSessionItem | null | undefined) => {
    if (!session?.session_id || seenIds.has(session.session_id)) return
    const sessionPath = normalizeFullCodePath(session.full_code_path || props.fullCodePath || '')
    if (currentPath && sessionPath && sessionPath !== currentPath) return
    seenIds.add(session.session_id)
    list.push(session)
  }

  for (const session of miniSessionList.value) {
    addIfCurrentPath(session)
  }

  if (active?.session_id && !seenIds.has(active.session_id)) {
    addIfCurrentPath(active)
  }

  return list
})

const recentSessionSourceList = computed(() => {
  const byId = new Map<string, WorkspaceSessionItem>()
  for (const session of [...globalSessionList.value, ...miniSessionList.value]) {
    if (!session.session_id) continue
    byId.set(session.session_id, session)
  }
  if (currentFallbackSession.value) {
    byId.set(currentFallbackSession.value.session_id, currentFallbackSession.value)
  }
  return Array.from(byId.values())
    .sort((left, right) => getSessionTimestamp(right) - getSessionTimestamp(left))
})

const summarySessions = computed(() => {
  const active = activeSummarySession.value
  const list = [...recentSessionSourceList.value]
  if (active && !list.some(item => item.session_id === active.session_id)) {
    list.unshift(active)
  }
  const visible = list.slice(0, 4)
  if (!active || visible.some(item => item.session_id === active.session_id)) {
    return visible
  }
  return [...visible.slice(0, 3), active]
})

const artifactItems = computed<MiniArtifactItem[]>(() => {
  const files = outputFiles.value.map((file, index) => buildFileArtifactItem(file, index))
  const fields = allPanelDisplayFields.value.map((field, index) => buildDisplayFieldArtifactItem(field, index))
  return [...files, ...fields]
})

const recentSessionCenterSourceList = computed(() => {
  return globalSessionList.value.length > 0 ? globalSessionList.value : recentSessionSourceList.value
})

const currentDirectorySessionList = computed(() => filterSessionCenterList(miniSessionList.value))
const recentSessionCenterList = computed(() => filterSessionCenterList(recentSessionCenterSourceList.value))

function filterSessionCenterList(list: WorkspaceSessionItem[]) {
  const keyword = sessionSearchKeyword.value.trim().toLowerCase()
  return list.filter((session) => {
    if (!matchesSessionFilter(session, sessionFilter.value)) return false
    return matchesSessionKeyword(session, keyword)
  })
}

function matchesSessionKeyword(session: WorkspaceSessionItem, keyword: string) {
  if (!keyword) return true
  return [
    session.title,
    session.user,
    session.agent_name,
    session.role_display_name,
    session.directory_name,
    session.full_code_path,
    getSessionDirectoryPath(session)
  ].some(value => (value || '').toLowerCase().includes(keyword))
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

function startNewSession() {
  suppressAutoSelectLatestSession.value = true
  handleNewSession()
  resetOutputScrollState()
  if (maximized.value) {
    maximized.value = false
    emit('maximize-change', { maximized: false, sessionId: '' })
  }
  setCollapsed(false, '')
}

async function openSessionCenter() {
  sessionCenterOpen.value = true
  await loadMiniSessions()
  await loadGlobalSessions()
}

function closeSessionCenter() {
  sessionCenterOpen.value = false
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

function handleCurrentOutputSessionSelect(session: WorkspaceSessionItem) {
  if (session.session_id && session.session_id === sessionId.value) {
    return
  }
  requestSessionSwitch(session)
}

function handleSummarySessionSelect(session: WorkspaceSessionItem) {
  if (session.session_id && session.session_id === sessionId.value) {
    if (!maximized.value) {
      maximized.value = true
      stopMiniPoll()
      restoreOutputScroll()
      emit('maximize-change', { maximized: true, sessionId: sessionId.value })
    }
    return
  }

  requestSessionSwitch(session)
}

function handleSessionCenterSelect(session: WorkspaceSessionItem) {
  closeSessionCenter()
  requestSessionSwitch(session)
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

function decodePathSegment(segment: string) {
  try {
    return decodeURIComponent(segment)
  } catch {
    return segment
  }
}

function normalizeFullCodePath(fullCodePath: string) {
  return (fullCodePath || '').trim().replace(/\/+$/g, '')
}

function getMappedPathName(fullCodePath: string) {
  const normalizedPath = normalizeFullCodePath(fullCodePath)
  if (!normalizedPath) return ''
  return props.pathNameMap?.[normalizedPath]
    || props.pathNameMap?.[normalizedPath.replace(/^\/+/, '')]
    || ''
}

function resolvePathDisplayName(fullCodePath: string) {
  const normalizedPath = normalizeFullCodePath(fullCodePath)
  if (!normalizedPath) return ''
  const mappedName = getMappedPathName(normalizedPath)
  if (mappedName) return mappedName
  if (normalizedPath === normalizeFullCodePath(props.fullCodePath) && props.dirName) {
    return props.dirName
  }
  const parts = normalizedPath.split('/').filter(Boolean).map(decodePathSegment)
  if (parts.length >= 2) {
    return parts.slice(-2).join(' / ')
  }
  return parts[0] || normalizedPath
}

function getSessionDirectoryPath(session: WorkspaceSessionItem) {
  const explicitDirectoryName = (session.directory_name || '').trim()
  if (explicitDirectoryName) {
    return explicitDirectoryName
  }

  const path = normalizeFullCodePath(session.full_code_path || props.fullCodePath || '')
  if (!path) {
    return props.dirName || displayPath.value || '当前目录'
  }

  const mappedName = getMappedPathName(path)
  if (mappedName) {
    return mappedName
  }

  if (path === normalizeFullCodePath(props.fullCodePath) && props.dirName) {
    return props.dirName
  }

  return resolvePathDisplayName(path) || props.dirName || displayPath.value || '当前目录'
}

function getSessionSubtitle(session: WorkspaceSessionItem) {
  return [getSessionDirectoryPath(session), getSessionStatusLabel(session)].filter(Boolean).join(' · ')
}

function getSessionCenterSubtitle(session: WorkspaceSessionItem) {
  return [getSessionDirectoryPath(session), session.role_display_name || session.user || getSessionStatusLabel(session)]
    .filter(Boolean)
    .join(' · ') || '当前目录'
}

function getSessionTimestamp(session: WorkspaceSessionItem) {
  const time = new Date(session.updated_at || session.created_at).getTime()
  return Number.isFinite(time) ? time : 0
}

function getSessionStatusLabel(session: WorkspaceSessionItem) {
  const status = getSessionRawStatus(session)
  if (status === 'pending_confirmation') return 'PRD 待确认'
  if (status === 'pending_test') return '测试待确认'
  const labels: Record<SessionStatusKind, string> = {
    running: '执行中',
    waiting: '待确认',
    done: '已完成',
    cancelled: '已取消',
    failed: '失败',
    active: '会话',
    output: '新文件'
  }
  return labels[getSessionStatusKind(session)]
}

function getSessionRawStatus(session: WorkspaceSessionItem) {
  return String(session.status || '').trim().toLowerCase()
}

function getSessionStatusKind(session: WorkspaceSessionItem): SessionStatusKind {
  const status = getSessionRawStatus(session)
  if ([
    'generating',
    'running',
    'tool_running',
    'thinking',
    'streaming',
    'processing',
    'executing'
  ].includes(status)) return 'running'

  if ([
    'waiting',
    'pending',
    'pending_confirmation',
    'pending_test',
    'waiting_approval',
    'paused',
    'queued'
  ].includes(status)) return 'waiting'

  if (['cancelled', 'canceled', 'abort', 'aborted'].includes(status)) return 'cancelled'
  if (['failed', 'failure', 'error', 'timeout'].includes(status)) return 'failed'
  if (['output', 'new_file', 'new_output', 'has_output', 'artifact', 'artifact_ready'].includes(status)) return 'output'
  if (sessionHasGeneratedArtifacts(session)) return 'output'
  if (session.handoff_kind) return 'output'
  if (['done', 'completed', 'complete', 'success', 'succeeded', 'finished'].includes(status)) return 'done'
  if (status === 'active' || !status) return 'active'
  return 'active'
}

function sessionHasGeneratedArtifacts(session: WorkspaceSessionItem) {
  return !!session.session_id
    && session.session_id === sessionId.value
    && currentMessagesHaveGeneratedArtifacts()
}

function currentMessagesHaveGeneratedArtifacts() {
  if (artifactItems.value.length > 0) return true
  return messages.value.some(messageHasGeneratedArtifacts)
}

function messageHasGeneratedArtifacts(message: ChatMessage) {
  if (message.role !== 'assistant') return false
  return collectMessageToolCalls(message).some(isGeneratedArtifactToolCall)
}

function getSessionStatusClass(session: WorkspaceSessionItem) {
  return `is-${getSessionStatusKind(session)}`
}

function matchesSessionFilter(session: WorkspaceSessionItem, filter: SessionFilterValue) {
  const kind = getSessionStatusKind(session)
  if (filter === 'all') return true
  if (filter === 'running') return kind === 'running'
  if (filter === 'waiting') return kind === 'waiting'
  if (filter === 'output') return kind === 'output'
  if (filter === 'done') return kind === 'done' || kind === 'cancelled' || kind === 'failed'
  return true
}

const currentAttachedFileRefs = computed(() => {
  return attachedFiles.value
    .map((file) => file.ref)
    .filter(Boolean)
    .join(',')
})

function openNewScheduledAgentTaskDialog() {
  if (!featureFlags.scheduledTasks) {
    return
  }
  scheduledDraftGoal.value = inputText.value.trim()
  scheduledDraftFiles.value = currentAttachedFileRefs.value
  scheduledDialogKey.value += 1
  showScheduledAgentTaskDialog.value = true
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
      restoreOutputScroll()
    }
  },
  { flush: 'pre' }
)

watch(
  () => props.fullCodePath,
  () => {
    suppressAutoSelectLatestSession.value = false
  }
)

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
  grid-template-rows: auto minmax(0, 1fr) auto;
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
  min-height: 24px;
  display: flex;
  align-items: center;
  gap: 8px;
  justify-content: flex-start;
  margin-bottom: 8px;
  color: var(--mini-cyber-muted);
  font-size: 12px;
}

.mini-artifact-head strong {
  color: #8ed0ff;
  white-space: nowrap;
}

.mini-artifact-actions {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.mini-artifact-item {
  width: 100%;
  height: 42px;
  display: grid;
  grid-template-columns: 30px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 5px 8px;
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
  min-height: 66px;
  padding: 10px;
  grid-template-columns: 48px minmax(0, 1fr) auto;
  margin-bottom: 8px;
}

.mini-ws--maximized .mini-artifact-item + .mini-artifact-item {
  display: grid;
}

.mini-artifact-preview {
  width: 30px;
  height: 30px;
  min-width: 0;
  position: relative;
  display: inline-grid;
  place-items: center;
  overflow: hidden;
  border: 1px solid rgba(130, 153, 190, 0.18);
  border-radius: 8px;
  background: rgba(10, 16, 29, 0.62);
  color: #8ed0ff;
}

.mini-ws--maximized .mini-artifact-preview {
  width: 48px;
  height: 48px;
  border-radius: 10px;
}

.mini-artifact-preview img {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: cover;
}

.mini-artifact-preview.is-image {
  border-color: rgba(83, 174, 255, 0.28);
  background: rgba(24, 48, 77, 0.46);
}

.mini-artifact-preview.is-data {
  color: #7df5c4;
  background: rgba(21, 54, 50, 0.42);
}

.mini-artifact-preview.is-document {
  color: #bcb7ff;
  background: rgba(41, 38, 76, 0.46);
}

.mini-artifact-preview.is-media {
  color: #ffd78d;
  background: rgba(58, 45, 24, 0.46);
}

.mini-artifact-preview.is-archive,
.mini-artifact-preview.is-file {
  color: #b9c9e4;
  background: rgba(41, 48, 64, 0.46);
}

.mini-artifact-preview.is-field {
  color: #8ed0ff;
  background: rgba(24, 51, 83, 0.46);
}

.mini-artifact-ext {
  position: absolute;
  right: 2px;
  bottom: 2px;
  max-width: calc(100% - 4px);
  overflow: hidden;
  padding: 1px 3px;
  border-radius: 4px;
  background: rgba(2, 7, 15, 0.72);
  color: #ffffff;
  font-size: 8px;
  font-weight: 900;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-ws--maximized .mini-artifact-ext {
  font-size: 9px;
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
  background: rgba(83, 174, 255, 0.14);
  color: #8ed0ff;
  font-size: 11px;
  font-weight: 800;
  white-space: nowrap;
}

.mini-artifact-item.is-data .mini-artifact-tag {
  background: rgba(43, 213, 159, 0.14);
  color: #7df5c4;
}

.mini-artifact-item.is-document .mini-artifact-tag {
  background: rgba(119, 107, 255, 0.16);
  color: #bcb7ff;
}

.mini-artifact-item.is-media .mini-artifact-tag {
  background: rgba(246, 189, 77, 0.15);
  color: #ffd78d;
}

.mini-artifact-item.is-archive .mini-artifact-tag,
.mini-artifact-item.is-file .mini-artifact-tag {
  background: rgba(142, 159, 187, 0.14);
  color: #b9c9e4;
}

.mini-artifact-item.is-field .mini-artifact-tag {
  background: rgba(83, 174, 255, 0.14);
  color: #8ed0ff;
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
  position: relative;
  display: block;
  margin: 0 14px 8px 204px;
  padding: 6px;
  border-radius: 14px;
  background: rgba(9, 14, 25, 0.68);
}

.mini-session-center-btn {
  position: absolute;
  left: -146px;
  top: 6px;
  width: 132px;
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

.mini-session-new-btn {
  position: absolute;
  left: -194px;
  top: 6px;
  width: 40px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(43, 213, 159, 0.42);
  border-radius: 12px;
  background:
    linear-gradient(135deg, rgba(43, 213, 159, 0.22), rgba(55, 163, 255, 0.12)),
    rgba(12, 22, 38, 0.78);
  color: #8dffd8;
  box-shadow: 0 12px 30px rgba(43, 213, 159, 0.14);
  cursor: pointer;
}

.mini-session-new-btn:hover {
  border-color: rgba(43, 213, 159, 0.62);
  background:
    linear-gradient(135deg, rgba(43, 213, 159, 0.32), rgba(55, 163, 255, 0.18)),
    rgba(12, 22, 38, 0.9);
  color: #ffffff;
}

.mini-session-summary-list {
  min-width: 0;
  display: grid;
  grid-template-columns: repeat(4, minmax(130px, 1fr));
  gap: 8px;
}

.mini-session-summary-card {
  --mini-active-glow: rgba(55, 163, 255, 0.26);
  --mini-active-halo: rgba(55, 163, 255, 0.12);
  --mini-active-arrow-color: #8ed0ff;
  --mini-active-arrow-shadow: rgba(55, 163, 255, 0.72);
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
  transition: background 0.18s ease, border-color 0.18s ease, box-shadow 0.22s ease;
}

.mini-session-summary-card:hover {
  border-color: rgba(87, 182, 255, 0.5);
  background: rgba(24, 51, 83, 0.62);
}

.mini-session-summary-card::before {
  content: "▼";
  position: absolute;
  left: 50%;
  top: -20px;
  transform: translateX(-50%) translateY(3px);
  opacity: 0;
  color: var(--mini-active-arrow-color);
  font-size: 16px;
  line-height: 1;
  text-shadow: 0 0 16px var(--mini-active-arrow-shadow);
  pointer-events: none;
  transition: opacity 0.18s ease, transform 0.18s ease, color 0.18s ease, text-shadow 0.18s ease;
}

.mini-session-summary-card.active::before {
  opacity: 1;
  transform: translateX(-50%) translateY(0);
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

.mini-session-summary-card.is-active {
  border-color: rgba(126, 151, 197, 0.24);
  background: rgba(30, 42, 68, 0.5);
}

.mini-session-summary-card.is-cancelled {
  border-color: rgba(142, 159, 187, 0.24);
  background: rgba(41, 48, 64, 0.46);
}

.mini-session-summary-card.is-failed {
  border-color: rgba(255, 108, 108, 0.34);
  background: rgba(74, 30, 38, 0.46);
}

.mini-session-summary-card.active.is-running::before {
  --mini-active-arrow-color: #7df5c4;
  --mini-active-arrow-shadow: rgba(43, 213, 159, 0.72);
}

.mini-session-summary-card.active.is-waiting::before {
  --mini-active-arrow-color: #ffd78d;
  --mini-active-arrow-shadow: rgba(246, 189, 77, 0.72);
}

.mini-session-summary-card.active.is-output::before {
  --mini-active-arrow-color: #8ed0ff;
  --mini-active-arrow-shadow: rgba(55, 163, 255, 0.72);
}

.mini-session-summary-card.active.is-done::before {
  --mini-active-arrow-color: #bcb7ff;
  --mini-active-arrow-shadow: rgba(119, 107, 255, 0.72);
}

.mini-session-summary-card.active.is-failed::before {
  --mini-active-arrow-color: #ff9ba4;
  --mini-active-arrow-shadow: rgba(255, 107, 107, 0.72);
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

.mini-status-dot.is-failed {
  background: #ff6b6b;
  box-shadow: 0 0 16px rgba(255, 107, 107, 0.58);
}

.mini-status-dot.is-active,
.mini-status-dot.is-output {
  background: #37a3ff;
  box-shadow: 0 0 16px rgba(55, 163, 255, 0.58);
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
  width: min(1120px, calc(100vw - 96px));
  height: min(720px, calc(100vh - 96px));
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
  grid-template-columns: auto minmax(220px, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 14px 18px;
  border-bottom: 1px solid rgba(130, 153, 190, 0.14);
}

.mini-session-dialog-stat,
.mini-session-filters {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mini-session-dialog-stat span,
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

.mini-session-dialog-stat span {
  border-color: rgba(43, 213, 159, 0.2);
  background: rgba(21, 54, 50, 0.26);
  color: #9ceccd;
}

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

.mini-session-columns {
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(320px, 0.92fr) minmax(420px, 1.18fr);
  gap: 14px;
  padding: 14px 18px 18px;
}

.mini-session-pane {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid rgba(126, 151, 197, 0.18);
  border-radius: 14px;
  background: rgba(10, 16, 29, 0.34);
}

.mini-session-pane--current {
  background: rgba(13, 27, 45, 0.46);
}

.mini-session-pane-head {
  min-width: 0;
  height: 58px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 14px;
  border-bottom: 1px solid rgba(126, 151, 197, 0.14);
}

.mini-session-pane-head div {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.mini-session-pane-head strong,
.mini-session-pane-head span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-session-pane-head strong {
  color: #e6f0ff;
  font-size: 14px;
  font-weight: 850;
}

.mini-session-pane-head span {
  color: #8b9bb7;
  font-size: 12px;
}

.mini-session-pane-head em {
  min-width: 28px;
  height: 28px;
  display: inline-grid;
  place-items: center;
  border-radius: 9px;
  background: rgba(83, 174, 255, 0.14);
  color: #8ed0ff;
  font-size: 12px;
  font-style: normal;
  font-weight: 900;
}

.mini-session-list {
  overflow: auto;
  padding: 12px;
}

.mini-session-row {
  --mini-active-glow: rgba(55, 163, 255, 0.26);
  --mini-active-halo: rgba(55, 163, 255, 0.12);
  position: relative;
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
  transition: background 0.18s ease, border-color 0.18s ease, box-shadow 0.22s ease;
}

.mini-session-row:last-child {
  margin-bottom: 0;
}

.mini-session-pane--current .mini-session-row {
  grid-template-columns: 10px minmax(0, 1fr) auto;
}

.mini-session-pane--current .mini-session-open {
  display: none;
}

.mini-session-row:hover {
  border-color: rgba(83, 174, 255, 0.42);
  background: rgba(24, 51, 83, 0.48);
}

.mini-session-row.is-running {
  border-color: rgba(43, 213, 159, 0.28);
  background: rgba(21, 54, 50, 0.42);
  box-shadow: inset 3px 0 0 rgba(43, 213, 159, 0.74);
}

.mini-session-row.is-waiting {
  border-color: rgba(246, 189, 77, 0.3);
  background: rgba(58, 45, 24, 0.46);
  box-shadow: inset 3px 0 0 rgba(246, 189, 77, 0.72);
}

.mini-session-row.is-output {
  border-color: rgba(55, 163, 255, 0.3);
  background: rgba(24, 48, 77, 0.46);
  box-shadow: inset 3px 0 0 rgba(55, 163, 255, 0.72);
}

.mini-session-row.is-done {
  border-color: rgba(119, 107, 255, 0.28);
  background: rgba(41, 38, 76, 0.46);
  box-shadow: inset 3px 0 0 rgba(119, 107, 255, 0.7);
}

.mini-session-row.is-cancelled {
  border-color: rgba(142, 159, 187, 0.24);
  background: rgba(41, 48, 64, 0.46);
  box-shadow: inset 3px 0 0 rgba(142, 159, 187, 0.5);
}

.mini-session-row.is-failed {
  border-color: rgba(255, 108, 108, 0.34);
  background: rgba(74, 30, 38, 0.46);
  box-shadow: inset 3px 0 0 rgba(255, 107, 107, 0.72);
}

.mini-current-session-row.is-running,
.mini-session-summary-card.is-running,
.mini-session-row.is-running {
  --mini-active-glow: rgba(43, 213, 159, 0.34);
  --mini-active-halo: rgba(43, 213, 159, 0.16);
}

.mini-current-session-row.is-waiting,
.mini-session-summary-card.is-waiting,
.mini-session-row.is-waiting {
  --mini-active-glow: rgba(246, 189, 77, 0.34);
  --mini-active-halo: rgba(246, 189, 77, 0.16);
}

.mini-current-session-row.is-output,
.mini-session-summary-card.is-output,
.mini-session-row.is-output {
  --mini-active-glow: rgba(55, 163, 255, 0.34);
  --mini-active-halo: rgba(55, 163, 255, 0.16);
}

.mini-current-session-row.is-done,
.mini-session-summary-card.is-done,
.mini-session-row.is-done {
  --mini-active-glow: rgba(119, 107, 255, 0.34);
  --mini-active-halo: rgba(119, 107, 255, 0.16);
}

.mini-current-session-row.is-cancelled,
.mini-session-summary-card.is-cancelled,
.mini-session-row.is-cancelled {
  --mini-active-glow: rgba(142, 159, 187, 0.28);
  --mini-active-halo: rgba(142, 159, 187, 0.12);
}

.mini-current-session-row.is-failed,
.mini-session-summary-card.is-failed,
.mini-session-row.is-failed {
  --mini-active-glow: rgba(255, 107, 107, 0.34);
  --mini-active-halo: rgba(255, 107, 107, 0.16);
}

.mini-current-session-row.active,
.mini-session-summary-card.active,
.mini-session-row.active {
  z-index: 1;
  box-shadow:
    0 0 14px 2px var(--mini-active-glow),
    0 0 38px 8px var(--mini-active-halo),
    0 12px 32px rgba(2, 5, 11, 0.22);
}

.mini-session-summary-card.active {
  z-index: 1;
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

.mini-files-dropdown-panel {
  background:
    radial-gradient(circle at 12% 0%, rgba(55, 163, 255, 0.16), transparent 34%),
    linear-gradient(150deg, rgba(8, 13, 24, 0.98), rgba(17, 25, 45, 0.96));
  border-color: rgba(130, 153, 190, 0.24);
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

  .mini-session-dock {
    margin-left: 184px;
  }

  .mini-session-center-btn {
    left: -130px;
    width: 118px;
  }

  .mini-session-new-btn {
    left: -176px;
    width: 38px;
  }

  .mini-session-summary-list {
    grid-template-columns: repeat(4, minmax(112px, 1fr));
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

  .mini-artifact-panel {
    display: none;
  }

  .mini-session-dock {
    display: grid;
    grid-template-columns: 1fr;
    margin: 0 14px 8px;
  }

  .mini-session-center-btn {
    position: static;
    width: 100%;
    justify-content: flex-start;
  }

  .mini-session-new-btn {
    position: static;
    width: 100%;
    justify-content: center;
  }

  .mini-session-summary-list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .mini-session-dialog-tools,
  .mini-session-columns,
  .mini-session-row {
    grid-template-columns: 1fr;
  }

  .mini-session-columns {
    overflow: auto;
  }

  .mini-session-dialog-stat,
  .mini-session-filters {
    flex-wrap: wrap;
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
