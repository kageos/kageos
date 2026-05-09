<!--
  MiniWorkstation - 迷你浮动工作台
  右下角弹出的小窗口，支持输入命令、上传文件、SSE 实时输出、最小化。
-->
<template>
  <transition name="mini-ws-pop">
    <div
      v-if="visible"
      ref="rootRef"
      :class="['mini-ws', { 'mini-ws--maximized': maximized, 'mini-ws--sending': sending, 'mini-ws--interaction-open': interactionOpen }]"
      data-testid="mini-workstation"
      :data-full-code-path="fullCodePath"
      :style="windowStyle"
      @dragover.prevent="onDragOver"
      @dragleave.prevent="onDragLeave"
      @drop.prevent="onDrop"
    >
      <!-- 四边 + 四角 resize 手柄 -->
      <template v-if="!maximized">
        <div class="mini-resize-handle mini-resize-n" @mousedown.stop="startResize($event, 'n')"></div>
        <div class="mini-resize-handle mini-resize-s" @mousedown.stop="startResize($event, 's')"></div>
        <div class="mini-resize-handle mini-resize-e" @mousedown.stop="startResize($event, 'e')"></div>
        <div class="mini-resize-handle mini-resize-w" @mousedown.stop="startResize($event, 'w')"></div>
        <div class="mini-resize-handle mini-resize-ne" @mousedown.stop="startResize($event, 'ne')"></div>
        <div class="mini-resize-handle mini-resize-nw" @mousedown.stop="startResize($event, 'nw')"></div>
        <div class="mini-resize-handle mini-resize-se" @mousedown.stop="startResize($event, 'se')"></div>
        <div class="mini-resize-handle mini-resize-sw" @mousedown.stop="startResize($event, 'sw')"></div>
      </template>

      <!-- 标题栏：左标题 + 居中目录名 + 右按钮，可拖拽 -->
      <div class="mini-ws-header" data-testid="mini-workstation-header" @mousedown="startDrag" @dblclick.prevent="onHeaderDblClick">
        <span class="mini-ws-title" data-testid="mini-workstation-title">
          <span class="mini-ws-title-orb">
            <el-icon v-if="sending" class="is-loading" :size="14"><Loading /></el-icon>
            <el-icon v-else :size="14"><FolderOpened /></el-icon>
          </span>
          <span class="mini-ws-title-label">MINI</span>
        </span>
        <span class="mini-ws-dir-name" :title="fullCodePath + (firstUserMessageFull ? '\n\n' + firstUserMessageFull : '')">
          {{ dirName || displayPath }}{{ firstUserMessagePreview ? ' · ' + firstUserMessagePreview : '' }}
        </span>
        <div class="mini-ws-header-actions" @mousedown.stop>
          <el-dropdown
            ref="keyInfoDropdownRef"
            v-if="panelHasContent"
            trigger="click"
            placement="left-start"
            popper-class="mini-files-dropdown-popper"
            :hide-on-click="false"
            @visible-change="onKeyInfoDropdownVisibleChange"
          >
            <el-button link size="small" class="mini-header-files-btn" title="查看关键信息">
              <el-icon :size="14"><DocumentIcon /></el-icon>
              <span class="mini-header-files-count">{{ panelItemCount }}</span>
            </el-button>
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
          <el-button link size="small" @click="$emit('minimize')" title="最小化">
            <el-icon :size="14"><Minus /></el-icon>
          </el-button>
          <el-button link size="small" @click="toggleMaximize" :title="maximized ? '还原' : '最大化'">
            <el-icon :size="14"><component :is="maximized ? CopyDocument : FullScreen" /></el-icon>
          </el-button>
          <el-button link size="small" data-testid="mini-workstation-close" @click="$emit('close')" title="关闭">
            <el-icon :size="14"><Close /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- 最大化时的主体区域：左侧会话列表 + 右侧消息 -->
      <div class="mini-ws-body">

      <!-- 最大化时：会话列表侧边栏 -->
      <MiniWorkstationSessionList
        v-if="maximized"
        :sessions="miniSessionList"
        :loading="loadingSessions"
        :active-session-id="sessionId"
        :format-relative-time="formatRelativeTime"
        @new="handleNewSession"
        @select="handleSelectSession"
      />

      <!-- SSE 输出区 -->
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

      <!-- 最大化时：右侧关键信息面板 -->
      <div v-if="maximized && panelHasContent" class="mini-file-sidebar">
        <div class="mini-file-sidebar-header">关键信息</div>
        <MiniWorkstationKeyInfoSection
          :uploaded-files="uploadedFiles"
          :output-files="outputFiles"
          :display-fields="allPanelDisplayFields"
          @preview-file="previewFile"
          @download-file="downloadFile"
          @preview-field="openDfPreview"
          @copy-field="copyDisplayFieldValue"
        />
      </div>

      </div><!-- /.mini-ws-body -->

      <MiniWorkstationComposer
        :full-code-path="fullCodePath"
        :attached-files="attachedFiles"
        :uploading="uploading"
        :input-text="inputText"
        :sending="sending"
        :stopping="stopping"
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
import { ref, onUnmounted, computed, watch } from 'vue'
import { Loading, Close, Minus, FullScreen, CopyDocument, FolderOpened, UploadFilled, Document as DocumentIcon, Setting } from '@element-plus/icons-vue'
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
import MiniWorkstationSessionList from './MiniWorkstationSessionList.vue'
import ScheduledAgentTaskDialog from './ScheduledAgentTaskDialog.vue'
import { useLazyMarkdownRenderer } from '@/composables/useLazyMarkdownRenderer'
import { useMiniWorkstationPanel } from '../composables/useMiniWorkstationPanel'
import { useMiniWorkstationWindow } from '../composables/useMiniWorkstationWindow'
import { useMiniWorkstationSessions } from '../composables/useMiniWorkstationSessions'
import { useMiniWorkstationUploads } from '../composables/useMiniWorkstationUploads'
import { useMiniWorkstationComposer } from '../composables/useMiniWorkstationComposer'
import { useMiniWorkstationEffects } from '../composables/useMiniWorkstationEffects'
import { eventBus, WorkspaceEvent } from '@/architecture/infrastructure/eventBus'
import { createWorkspaceHandoff } from '@/api/workspace'

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
const firstUserMessageFull = computed(() => {
  const first = messages.value?.find(m => m.role === 'user')
  return typeof first?.content === 'string' ? first.content.trim() : ''
})

// ─── 最大化 / 还原 ───
const maximized = ref(!!props.initialMaximized)
const preMaxRect = ref<{ x: number; y: number; w: number; h: number } | null>(null)

const {
  miniSessionList,
  loadingSessions,
  stopping,
  loadMiniSessions,
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

const {
  rootRef,
  windowStyle,
  startDrag,
  startResize,
  captureWindowRect,
  restoreWindowRect,
  dispose: disposeWindowState
} = useMiniWorkstationWindow({
  maximized,
  initialOffset: props.initialOffset,
  initialPosition: props.initialPosition
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
    restoreWindowRect(preMaxRect.value)
    emit('maximize-change', { maximized: false })
  } else {
    preMaxRect.value = captureWindowRect()
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
      target_role: 'app_developer',
      artifact_kind: 'agent_app_prd',
      artifact: payload.prd,
      remark,
      context_policy: 'artifact_only'
    })
  } catch (error: any) {
    ElMessage.error(error?.message || '创建交接会话失败')
    return
  }
  setMessages([])
  sessionId.value = handoff.session_id
  void sendTextToSession(
    handoff.session_id,
    handoff.content,
    handoff.display_content,
    { contextUsage: 'artifact', artifactKind: handoff.artifact_kind, resume: true }
  )
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

/** 双击标题栏切换最大化 */
function onHeaderDblClick() {
  toggleMaximize()
}

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

onUnmounted(() => {
  disposeWindowState()
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
