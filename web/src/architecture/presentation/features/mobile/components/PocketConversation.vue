<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Clock, MagicStick, Paperclip, Plus, Promotion, RefreshRight, User } from '@element-plus/icons-vue'
import {
  getPublicMessageAction,
  submitPublicMessageActionReply,
  type MessageActionViewResp,
  type MessageInboxItem,
} from '@/architecture/presentation/context/api/message'
import {
  getWorkspaceSessions,
  getWorkspaceMessages,
  workspaceChatStream,
  type WorkspaceAutomationAgentItem,
  type WorkspaceChatMessageFile,
  type WorkspaceSessionItem,
} from '@/architecture/presentation/context/api/workspace'
import MiniWorkstationMessages from '@/architecture/presentation/components/MiniWorkstationMessages.vue'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'
import { useMiniWorkstationPanel } from '@/architecture/presentation/composables/useMiniWorkstationPanel'
import { normalizeWorkspaceSessionMessages } from '@/architecture/presentation/composables/useMiniWorkstationSessions'
import { useMiniWorkstationUploads } from '@/architecture/presentation/composables/useMiniWorkstationUploads'
import {
  useWorkspaceChatStream,
  type ChatMessage,
  type ChatMessageFile,
} from '@/architecture/presentation/composables/useWorkspaceChatStream'
import { fileNameFromRef, parseFileRefs } from '@/architecture/presentation/widgets/filesWidgetTypes'
import { resolveMobileWorkspacePath } from '@/architecture/presentation/features/mobile/utils/workspacePath'
import { getErrorMessage } from '@/architecture/shared/apiError'

const props = defineProps<{
  mode: 'action' | 'ask'
}>()

const POLL_INTERVAL_MS = 5000
const MOBILE_ASK_DRAFT_STORAGE_KEY = 'kageos_mobile_ask_draft'

const route = useRoute()
const router = useRouter()
const actionView = ref<MessageActionViewResp | null>(null)
const fullCodePath = ref(resolveMobileWorkspacePath(queryString('source_path')))
const draft = ref('')
const loading = ref(false)
const refreshing = ref(false)
const error = ref('')
const lastSyncedAt = ref('')
const historyVisible = ref(false)
const historyLoading = ref(false)
const historySessions = ref<WorkspaceSessionItem[]>([])
const automationAgents = ref<WorkspaceAutomationAgentItem[]>([])
const historyFilterValue = ref('human')
const currentSessionItem = ref<WorkspaceSessionItem | null>(null)
const conversationEndRef = ref<HTMLElement | null>(null)
const composerInputRef = ref<{ focus: () => void }>()

const {
  messages,
  sending,
  sessionId,
  streamingDisplayLength,
  send,
  setMessages,
} = useWorkspaceChatStream()
sessionId.value = queryString('session_id') || undefined

const { getFileGroupsFromCalls, getDisplayFieldsFromCalls } = useMiniWorkstationPanel(messages)
const {
  attachedFiles,
  uploading,
  onFileChange,
  removeFile,
  onPaste,
} = useMiniWorkstationUploads({
  fullCodePath,
  inputText: draft,
  inputRef: composerInputRef,
})
const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()

let pollTimer: ReturnType<typeof setInterval> | null = null

const actionMode = computed(() => props.mode === 'action')
const token = computed(() => queryString('t'))
const canSend = computed(() => Boolean(
  fullCodePath.value.trim()
  && (draft.value.trim() || attachedFiles.value.length > 0)
  && !sending.value
  && !uploading.value
))
const sourceName = computed(() => firstNonEmpty(
  actionView.value?.message.source_display?.name,
  actionView.value?.message.source_title,
  actionView.value?.message.workspace_session_title,
  actionView.value?.message.source_parent_title,
  leafName(fullCodePath.value),
  'kageos'
))
const conversationTitle = computed(() => firstNonEmpty(
  currentSessionItem.value?.title,
  actionView.value?.message.workspace_session_title,
  actionView.value?.message.title,
  sourceName.value,
  '新会话'
))
const currentAutomationTitle = computed(() => firstNonEmpty(
  currentSessionItem.value?.automation_task_title,
))
const statusText = computed(() => {
  if (sending.value) return '正在处理'
  if (sessionId.value) return '每 5 秒同步'
  if (actionView.value?.can_reply) return '等待回复'
  return '新会话'
})
const statusClass = computed(() => sending.value ? 'is-running' : sessionId.value ? 'is-live' : '')
const sessionLabel = computed(() => {
  const id = sessionId.value || ''
  if (!id) return ''
  return id.length > 18 ? `${id.slice(0, 8)}…${id.slice(-6)}` : id
})
const composerPlaceholder = computed(() => {
  if (!fullCodePath.value.trim()) return '先填写工作台服务目录'
  if (actionView.value?.can_reply) return '回复这条通知…'
  return '给 kageos 发消息…'
})

function queryString(key: string) {
  const raw = route.query[key]
  return Array.isArray(raw) ? String(raw[0] || '') : String(raw || '')
}

function firstNonEmpty(...values: Array<string | undefined | null>) {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) return value.trim()
  }
  return ''
}

function requestErrorMessage(err: unknown, fallback: string) {
  return getErrorMessage(err, fallback)
}

function leafName(value: string) {
  return value.split('/').filter(Boolean).pop() || ''
}

function resolveActionPath(view: MessageActionViewResp) {
  return resolveMobileWorkspacePath(
    view.message.source_path,
    view.message.full_code_path,
    view.message.source_parent_path,
    view.message.source_template_type,
  )
}

function notificationContent(item: MessageInboxItem) {
  const title = (item.title || '').trim()
  const content = (item.content || '').trim()
  if (!title) return content
  if (!content) return `**${title}**`
  if (content.includes(title)) return content
  return `**${title}**\n\n${content}`
}

function notificationMessages(): ChatMessage[] {
  const thread = actionView.value?.thread || []
  return [...thread]
    .sort((left, right) => new Date(left.created_at).getTime() - new Date(right.created_at).getTime())
    .map(item => ({
      role: 'assistant' as const,
      user: item.from || 'kageos',
      content: notificationContent(item),
      files: item.files
        ? parseFileRefs(item.files).map(ref => ({ ref, name: fileNameFromRef(ref), source_name: fileNameFromRef(ref) }))
        : [],
      created_at: item.created_at,
    }))
    .filter(item => Boolean(item.content || item.files?.length))
}

function normalizedContent(value: string) {
  return value
    .replace(/\*\*/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}

function mergeNotificationContext(workspaceMessages: ChatMessage[]) {
  const combined = [...notificationMessages(), ...workspaceMessages]
    .sort((left, right) => new Date(left.created_at || 0).getTime() - new Date(right.created_at || 0).getTime())
  const seen = new Set<string>()
  return combined.filter(item => {
    const key = `${item.role}:${normalizedContent(item.content || '')}`
    if (key.endsWith(':') || seen.has(key)) return false
    seen.add(key)
    return true
  })
}

async function loadAction() {
  if (!token.value.trim()) {
    error.value = '处理链接缺少 token'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const view = await getPublicMessageAction(token.value)
    actionView.value = view
    fullCodePath.value = resolveActionPath(view)
    const linkedSession = firstNonEmpty(view.workspace_session_id, view.message.workspace_session_id)
    if (linkedSession) sessionId.value = linkedSession
    if (sessionId.value) {
      await refreshMessages(true)
    } else {
      setMessages(notificationMessages())
    }
  } catch (err) {
    error.value = requestErrorMessage(err, '加载会话失败')
  } finally {
    loading.value = false
  }
}

async function refreshMessages(force = false) {
  const targetSession = sessionId.value
  if (!targetSession || (sending.value && !force) || refreshing.value) return
  refreshing.value = true
  try {
    const response = await getWorkspaceMessages({ session_id: targetSession })
    if (sessionId.value !== targetSession) return
    const normalized = normalizeWorkspaceSessionMessages(response.messages || [])
    setMessages(actionMode.value ? mergeNotificationContext(normalized) : normalized)
    lastSyncedAt.value = new Date().toISOString()
  } catch (err) {
    if (force) {
      error.value = requestErrorMessage(err, '同步会话失败')
    }
  } finally {
    refreshing.value = false
  }
}

function selectedAutomationTaskID() {
  if (!historyFilterValue.value.startsWith('agent:')) return 0
  return Number(historyFilterValue.value.slice('agent:'.length)) || 0
}

async function loadHistory() {
  const path = fullCodePath.value.trim()
  if (!path) return
  const taskID = selectedAutomationTaskID()
  historyLoading.value = true
  try {
    const response = await getWorkspaceSessions({
      full_code_path: path,
      page: 1,
      page_size: 50,
      session_scope: taskID > 0 ? 'automation' : 'human',
      automation_task_id: taskID || undefined,
    })
    historySessions.value = response.sessions || []
    automationAgents.value = response.automation_agents || []
  } catch (err) {
    ElMessage.error(requestErrorMessage(err, '加载历史会话失败'))
  } finally {
    historyLoading.value = false
  }
}

function openHistory() {
  if (!fullCodePath.value.trim()) {
    ElMessage.warning('请先填写工作台服务目录')
    return
  }
  historyVisible.value = true
  void loadHistory()
}

async function selectHistorySession(item: WorkspaceSessionItem) {
  const path = firstNonEmpty(item.resource_full_code_path, item.full_code_path, fullCodePath.value)
  historyVisible.value = false
  if (actionMode.value) {
    await router.push({
      name: 'mobile-ask',
      query: { source_path: path, session_id: item.session_id },
    })
    return
  }
  currentSessionItem.value = item
  fullCodePath.value = path
  sessionId.value = item.session_id
  setMessages([])
  error.value = ''
  await router.replace({
    name: 'mobile-ask',
    query: { source_path: path, session_id: item.session_id },
  })
  await refreshMessages(true)
}

function buildPocketMessage(rawMessage: string) {
  if (sessionId.value) return rawMessage.trim()
  return [
    '【kageos Pocket 会话】',
    '用户正在移动端实时查看本会话，工作台回复会在页面中每 5 秒同步显示。',
    '请像 PC 工作台会话一样直接回答；最终回复使用适合手机阅读的 Markdown。',
    '只有确实需要离开会话后异步提醒时，才使用 send_notification；不要每轮重复发送通知。',
    '',
    '用户消息：',
    rawMessage.trim(),
  ].join('\n')
}

function fileRefs(files: WorkspaceChatMessageFile[]) {
  return files.map(file => file.ref).filter(Boolean).join(',')
}

function chatMessageFiles(files: WorkspaceChatMessageFile[]): ChatMessageFile[] {
  return files.map(file => ({ ...file }))
}

async function submitFirstActionReply(content: string, files: WorkspaceChatMessageFile[]) {
  const before = [...messages.value]
  setMessages([
    ...before,
    { role: 'user', content, files: chatMessageFiles(files), created_at: new Date().toISOString() },
  ])
  sending.value = true
  try {
    const result = await submitPublicMessageActionReply(token.value, {
      content,
      files: fileRefs(files),
      action: 'reply',
    })
    if (actionView.value) {
      actionView.value = {
        ...actionView.value,
        can_reply: false,
        token_status: 'submitted',
        submitted_at: result.submitted_at,
        workspace_session_id: result.workspace_session_id || actionView.value.workspace_session_id,
      }
    }
    if (result.workspace_session_id) sessionId.value = result.workspace_session_id
    if (result.agent_submit_error) {
      error.value = `消息已记录，但工作台接收失败：${result.agent_submit_error}`
      ElMessage.warning('消息已记录，但工作台接收失败')
    } else {
      ElMessage.success('已发送到会话')
    }
  } catch (err) {
    setMessages(before)
    throw err
  } finally {
    sending.value = false
  }
  if (sessionId.value) await refreshMessages(true)
}

async function submitWorkspaceMessage(content: string, files: WorkspaceChatMessageFile[]) {
  const payloadContent = buildPocketMessage(content)
  await send(content, onEvent => workspaceChatStream({
    full_code_path: fullCodePath.value.trim(),
    resource_full_code_path: fullCodePath.value.trim(),
    session_id: sessionId.value,
    message: {
      content: payloadContent,
      display_content: content,
      files: fileRefs(files) || undefined,
    },
  }, onEvent), chatMessageFiles(files))
  if (sessionId.value) await refreshMessages(true)
}

async function sendDraft() {
  const rawContent = draft.value.trim()
  const files = [...attachedFiles.value]
  if ((!rawContent && files.length === 0) || sending.value || uploading.value) return
  if (!fullCodePath.value.trim()) {
    ElMessage.warning('请先填写工作台服务目录')
    return
  }
  const content = rawContent || '请处理我上传的文件。'
  draft.value = ''
  attachedFiles.value = []
  error.value = ''
  try {
    if (actionMode.value && actionView.value?.can_reply) {
      await submitFirstActionReply(content, files)
    } else {
      await submitWorkspaceMessage(content, files)
    }
  } catch (err) {
    if (!draft.value) draft.value = rawContent
    attachedFiles.value = files
    error.value = requestErrorMessage(err, '发送失败')
    ElMessage.error(error.value)
  }
}

function startNewConversation() {
  stopPolling()
  sending.value = false
  sessionId.value = undefined
  currentSessionItem.value = null
  setMessages([])
  draft.value = ''
  attachedFiles.value = []
  error.value = ''
  void router.push({
    name: 'mobile-ask',
    query: fullCodePath.value.trim() ? { source_path: fullCodePath.value.trim() } : {},
  })
}

function loadStoredDraft() {
  const raw = sessionStorage.getItem(MOBILE_ASK_DRAFT_STORAGE_KEY)
  if (!raw) return
  sessionStorage.removeItem(MOBILE_ASK_DRAFT_STORAGE_KEY)
  try {
    const stored = JSON.parse(raw) as { full_code_path?: string; session_id?: string; message?: string }
    if (stored.full_code_path?.trim()) fullCodePath.value = stored.full_code_path.trim()
    if (stored.session_id?.trim()) sessionId.value = stored.session_id.trim()
    if (stored.message?.trim()) draft.value = stored.message.trim()
  } catch {
    // Ignore stale drafts created by an older Pocket version.
  }
}

function startPolling() {
  stopPolling()
  if (!sessionId.value) return
  pollTimer = setInterval(() => void refreshMessages(), POLL_INTERVAL_MS)
}

function stopPolling() {
  if (!pollTimer) return
  clearInterval(pollTimer)
  pollTimer = null
}

function formatMessageTime(value: string) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(date)
}

async function scrollToBottom(behavior: ScrollBehavior = 'smooth') {
  await nextTick()
  conversationEndRef.value?.scrollIntoView({ behavior, block: 'end' })
}

function handleVisibilityChange() {
  if (document.visibilityState === 'visible') void refreshMessages()
}

watch(sessionId, id => {
  startPolling()
  if (!actionMode.value && id && route.query.session_id !== id) {
    void router.replace({ query: { ...route.query, session_id: id } })
  }
})

watch(
  () => messages.value.map(item => `${item.role}:${item.content.length}:${item.tool_calls?.length || 0}`).join('|'),
  () => void scrollToBottom(),
  { flush: 'post' }
)

onMounted(async () => {
  document.addEventListener('visibilitychange', handleVisibilityChange)
  if (actionMode.value) {
    await loadAction()
    return
  }
  loadStoredDraft()
  if (sessionId.value) await refreshMessages(true)
  startPolling()
})

onBeforeUnmount(() => {
  stopPolling()
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
</script>

<template>
  <main class="pocket-page">
    <section class="pocket-shell">
      <header class="pocket-header">
        <div class="pocket-title-block">
          <div class="pocket-brand">kageos Pocket</div>
          <h1>{{ conversationTitle }}</h1>
          <div class="pocket-context-line">
            <span v-if="fullCodePath" :title="fullCodePath">{{ fullCodePath }}</span>
            <span v-if="sessionLabel" :title="sessionId">会话 {{ sessionLabel }}</span>
            <span v-if="currentAutomationTitle" :title="currentAutomationTitle">Agent · {{ currentAutomationTitle }}</span>
          </div>
        </div>
        <div class="pocket-header-actions">
          <span class="pocket-status" :class="statusClass">
            <i></i>{{ statusText }}
          </span>
          <el-button circle plain :icon="Clock" aria-label="历史会话" @click="openHistory" />
          <el-button circle plain :icon="Plus" aria-label="新会话" @click="startNewConversation" />
        </div>
      </header>

      <section v-if="!fullCodePath && !loading" class="pocket-setup">
        <strong>选择工作台上下文</strong>
        <p>输入一个服务目录路径，后续消息会像 PC 工作台一样在同一会话中连续处理。</p>
        <el-input
          v-model="fullCodePath"
          size="large"
          placeholder="/user/app 或 /user/app/package"
          clearable
        />
      </section>

      <section class="pocket-conversation" aria-live="polite">
        <el-skeleton v-if="loading" :rows="7" animated />
        <template v-else>
          <el-alert
            v-if="error"
            class="pocket-error"
            type="error"
            :title="error"
            show-icon
            :closable="true"
            @close="error = ''"
          />

          <MiniWorkstationMessages
            :messages="messages"
            :maximized="false"
            :sending="sending"
            :counterpart-name="sourceName"
            :full-code-path="fullCodePath"
            :streaming-display-length="streamingDisplayLength"
            :render-markdown="renderMarkdown"
            :format-message-time="formatMessageTime"
            :get-file-groups-from-calls="getFileGroupsFromCalls"
            :get-display-fields-from-calls="getDisplayFieldsFromCalls"
          />

          <div v-if="refreshing && !sending" class="pocket-syncing">
            <el-icon class="is-loading"><RefreshRight /></el-icon>
            正在同步会话
          </div>
          <div ref="conversationEndRef" class="pocket-conversation-end" aria-hidden="true" />
        </template>
      </section>

      <footer class="pocket-composer" @paste="onPaste">
        <div v-if="attachedFiles.length > 0" class="pocket-attachments">
          <el-tag
            v-for="(file, index) in attachedFiles"
            :key="`${file.ref || file.name}-${index}`"
            size="small"
            :closable="!sending"
            disable-transitions
            @close="removeFile(index)"
          >
            {{ file.source_name || file.name }}
          </el-tag>
        </div>

        <el-upload
          :auto-upload="false"
          :show-file-list="false"
          :on-change="onFileChange"
          :disabled="!fullCodePath || sending || uploading"
          multiple
          class="pocket-upload"
        >
          <el-button
            class="pocket-attach"
            circle
            plain
            :icon="Paperclip"
            :loading="uploading"
            :disabled="!fullCodePath || sending || uploading"
            aria-label="上传文件"
          />
        </el-upload>

        <el-input
          ref="composerInputRef"
          v-model="draft"
          type="textarea"
          :autosize="{ minRows: 1, maxRows: 6 }"
          resize="none"
          :placeholder="composerPlaceholder"
          :disabled="!fullCodePath || sending"
          @keydown.enter.exact.prevent="sendDraft"
        />
        <el-button
          class="pocket-send"
          type="primary"
          circle
          :icon="Promotion"
          :loading="sending"
          :disabled="!canSend"
          aria-label="发送消息"
          @click="sendDraft"
        />
        <div class="pocket-composer-meta">
          <span>Enter 发送 · Shift + Enter 换行</span>
          <span v-if="lastSyncedAt">已同步 {{ formatMessageTime(lastSyncedAt) }}</span>
        </div>
      </footer>
    </section>

    <el-drawer
      v-model="historyVisible"
      direction="btt"
      size="72%"
      class="pocket-history-drawer"
      :with-header="false"
      append-to-body
    >
      <section class="pocket-history">
        <header class="pocket-history-header">
          <div>
            <strong>历史会话</strong>
            <span>默认只看人工发起的会话</span>
          </div>
          <el-select
            v-model="historyFilterValue"
            class="pocket-history-filter"
            aria-label="会话来源筛选"
            @change="loadHistory"
          >
            <el-option label="人工会话" value="human">
              <span class="pocket-filter-option"><el-icon><User /></el-icon>人工会话</span>
            </el-option>
            <el-option
              v-for="agent in automationAgents"
              :key="agent.task_id"
              :label="agent.task_title"
              :value="`agent:${agent.task_id}`"
            >
              <span class="pocket-filter-option"><el-icon><MagicStick /></el-icon>{{ agent.task_title }}</span>
            </el-option>
          </el-select>
        </header>

        <div v-loading="historyLoading" class="pocket-history-list">
          <button
            v-for="item in historySessions"
            :key="item.session_id"
            type="button"
            class="pocket-history-item"
            :class="{ 'is-current': item.session_id === sessionId }"
            @click="selectHistorySession(item)"
          >
            <span class="pocket-history-item-main">
              <span class="pocket-history-title">{{ item.title || '未命名会话' }}</span>
              <span class="pocket-history-meta">
                {{ formatMessageTime(item.updated_at || item.created_at) }} · {{ item.status === 'generating' ? '执行中' : '可继续' }}
              </span>
            </span>
            <span v-if="item.source === 'automation_agent'" class="pocket-agent-badge">
              <el-icon><MagicStick /></el-icon>{{ item.automation_task_title || '自动化 Agent' }}
            </span>
          </button>
          <el-empty v-if="!historyLoading && historySessions.length === 0" description="暂无会话" :image-size="72" />
        </div>
      </section>
    </el-drawer>
  </main>
</template>

<style scoped>
.pocket-page {
  min-height: 100dvh;
  background:
    radial-gradient(circle at 50% -20%, rgba(var(--color-primary-rgb), 0.1), transparent 38%),
    var(--bg-primary);
  color: var(--text-primary);
}

.pocket-shell {
  width: min(100%, 840px);
  height: 100dvh;
  min-height: 100dvh;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  border-inline: 1px solid var(--border-lighter);
}

.pocket-header {
  position: sticky;
  top: 0;
  z-index: 10;
  min-height: 72px;
  padding: calc(10px + env(safe-area-inset-top)) 16px 10px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid var(--border-light);
  background: color-mix(in srgb, var(--bg-primary) 92%, transparent);
  backdrop-filter: blur(18px);
}

.pocket-title-block {
  flex: 1;
  min-width: 0;
}

.pocket-brand {
  margin-bottom: 3px;
  color: var(--color-primary);
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.pocket-header h1 {
  overflow: hidden;
  margin: 0;
  color: var(--text-primary);
  font-size: 17px;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pocket-context-line {
  min-width: 0;
  margin-top: 4px;
  display: flex;
  gap: 8px;
  color: var(--text-secondary);
  font-size: 11px;
}

.pocket-context-line span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pocket-context-line span:first-child {
  max-width: min(54vw, 430px);
}

.pocket-header-actions {
  flex-shrink: 0;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.pocket-header-actions :deep(.el-button) {
  width: 34px;
  height: 34px;
  border-color: var(--border-light);
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.pocket-history {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  color: var(--text-primary);
}

.pocket-history-header {
  padding: 4px 2px 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border-bottom: 1px solid var(--border-light);
}

.pocket-history-header > div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.pocket-history-header strong {
  font-size: 17px;
}

.pocket-history-header span {
  color: var(--text-secondary);
  font-size: 11px;
}

.pocket-history-filter {
  width: min(46vw, 220px);
}

.pocket-filter-option {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.pocket-history-list {
  flex: 1;
  min-height: 120px;
  padding-block: 10px calc(10px + env(safe-area-inset-bottom));
  overflow-y: auto;
}

.pocket-history-item {
  width: 100%;
  padding: 13px 10px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border: 0;
  border-bottom: 1px solid var(--border-lighter);
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
}

.pocket-history-item.is-current {
  border-radius: 10px;
  background: rgba(var(--color-primary-rgb), 0.08);
}

.pocket-history-item-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.pocket-history-title {
  overflow: hidden;
  font-size: 14px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pocket-history-meta {
  color: var(--text-secondary);
  font-size: 11px;
}

.pocket-agent-badge {
  max-width: 42%;
  flex-shrink: 0;
  padding: 4px 7px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(var(--color-primary-rgb), 0.1);
  color: var(--color-primary);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pocket-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 8px;
  border: 1px solid var(--border-light);
  border-radius: 999px;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  font-size: 11px;
  white-space: nowrap;
}

.pocket-status i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-placeholder);
}

.pocket-status.is-live i {
  background: var(--color-success);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-success) 14%, transparent);
}

.pocket-status.is-running i {
  background: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(var(--color-primary-rgb), 0.14);
  animation: pocketPulse 1.2s ease-in-out infinite;
}

.pocket-setup {
  margin: 16px 16px 0;
  padding: 16px;
  border: 1px solid var(--border-light);
  border-radius: 12px;
  background: var(--bg-secondary);
}

.pocket-setup strong {
  font-size: 14px;
}

.pocket-setup p {
  margin: 6px 0 12px;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.55;
}

.pocket-setup :deep(.el-input__wrapper) {
  background: var(--bg-primary);
}

.pocket-conversation {
  flex: 1;
  width: 100%;
  min-height: 0;
  min-width: 0;
  padding: 18px 18px 34px;
  overflow-y: auto;
  font-size: 14px;
  line-height: 1.65;
  scrollbar-color: rgba(var(--color-primary-rgb), 0.28) transparent;
}

.pocket-error {
  margin-bottom: 14px;
}

.pocket-syncing {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px;
  color: var(--text-placeholder);
  font-size: 11px;
}

.pocket-conversation-end {
  height: 1px;
}

.pocket-composer {
  position: sticky;
  bottom: 0;
  z-index: 9;
  padding: 10px 12px calc(9px + env(safe-area-inset-bottom));
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 8px;
  border-top: 1px solid var(--border-light);
  background: color-mix(in srgb, var(--bg-primary) 94%, transparent);
  backdrop-filter: blur(18px);
}

.pocket-header,
.pocket-composer {
  width: 100%;
  min-width: 0;
}

.pocket-setup {
  min-width: 0;
}

.pocket-composer > :deep(.el-textarea) {
  width: 100%;
  min-width: 0;
}

.pocket-attachments {
  grid-column: 1 / -1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.pocket-attachments :deep(.el-tag) {
  max-width: 100%;
  border-color: rgba(var(--color-primary-rgb), 0.22);
  background: rgba(var(--color-primary-rgb), 0.08);
  color: var(--text-primary);
}

.pocket-attachments :deep(.el-tag__content) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pocket-upload {
  align-self: end;
}

.pocket-upload :deep(.el-upload) {
  display: block;
}

.pocket-attach {
  width: 42px;
  height: 42px;
  border-color: var(--border-light);
  background: var(--bg-secondary);
  color: var(--text-secondary);
}

.pocket-composer :deep(.el-textarea__inner) {
  min-height: 42px !important;
  padding: 10px 13px;
  border: 1px solid var(--border-light);
  border-radius: 12px;
  background: var(--bg-secondary);
  color: var(--text-primary);
  box-shadow: none;
  font-size: 15px;
  line-height: 1.45;
}

.pocket-composer :deep(.el-textarea__inner:focus) {
  border-color: rgba(var(--color-primary-rgb), 0.5);
  box-shadow: 0 0 0 3px rgba(var(--color-primary-rgb), 0.1);
}

.pocket-send {
  width: 42px;
  height: 42px;
  align-self: end;
}

.pocket-composer-meta {
  grid-column: 1 / -1;
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding-inline: 2px;
  color: var(--text-placeholder);
  font-size: 10px;
}

@keyframes pocketPulse {
  50% { opacity: 0.45; }
}

@media (max-width: 560px) {
  .pocket-shell {
    border-inline: 0;
  }

  .pocket-header {
    min-height: 66px;
    padding-inline: 12px;
  }

  .pocket-header h1 {
    font-size: 16px;
  }

  .pocket-context-line span:first-child {
    max-width: 48vw;
  }

  .pocket-status {
    max-width: 88px;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .pocket-conversation {
    padding: 14px 12px 28px;
  }

  .pocket-composer-meta span:first-child {
    display: none;
  }

  .pocket-composer-meta {
    justify-content: flex-end;
  }
}
</style>
