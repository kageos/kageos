<template>
  <div class="workspace-inbox">
    <el-tooltip v-if="props.showTrigger" content="站内信" placement="bottom" effect="light">
      <el-badge
        :value="unreadCount"
        :hidden="unreadCount <= 0"
        :max="99"
        class="workspace-inbox-badge"
      >
        <el-button
          class="workspace-inbox-button"
          :icon="MessageIcon"
          :loading="countLoading"
          circle
          @click="openDrawer"
        />
      </el-badge>
    </el-tooltip>

    <el-drawer
      v-model="drawerVisible"
      :title="drawerTitle"
      direction="rtl"
      size="min(1120px, 96vw)"
      :destroy-on-close="false"
      append-to-body
      modal-class="workspace-inbox-modal"
      :z-index="Z_INDEX.globalOverlay"
      class="workspace-inbox-drawer"
      @open="handleDrawerOpen"
    >
      <div class="inbox-shell">
        <header class="inbox-toolbar">
          <div class="inbox-filter">
            <el-segmented
              v-model="statusFilter"
              :options="statusOptions"
              size="small"
              @change="loadInbox(true)"
            />
            <div v-if="sourceFilter" class="source-filter-chip">
              <el-tag size="small" effect="plain">节点通知</el-tag>
              <span :title="sourceFilter.sourcePath">{{ sourceFilter.title || sourceFilter.sourcePath }}</span>
              <el-button size="small" text @click="clearSourceFilter">查看全部</el-button>
            </div>
          </div>
          <div class="inbox-actions">
            <el-button :icon="Refresh" :loading="listLoading" @click="loadInbox(true)">
              刷新
            </el-button>
            <el-button :disabled="currentScopeUnreadCount <= 0" @click="markCurrentScopeRead">
              {{ sourceFilter ? '当前节点已读' : '全部已读' }}
            </el-button>
          </div>
        </header>

        <el-alert
          v-if="errorMessage"
          :title="errorMessage"
          type="error"
          show-icon
          :closable="false"
          class="inbox-error"
        />

        <div class="inbox-layout">
          <section class="inbox-list-pane" v-loading="listLoading">
            <el-empty
              v-if="!listLoading && inboxThreads.length === 0"
              description="暂无站内信"
              :image-size="80"
            />

            <button
              v-for="thread in inboxThreads"
              :key="thread.key"
              type="button"
              class="inbox-list-item"
              :class="{ 'is-active': selectedThread?.key === thread.key, 'is-unread': thread.unreadCount > 0 }"
              @click="selectThread(thread)"
            >
              <span class="thread-avatar">
                <el-icon><component :is="threadIcon(thread)" /></el-icon>
              </span>
              <span class="inbox-list-copy">
                <span class="thread-title-row">
                  <span class="inbox-list-title">{{ thread.title }}</span>
                  <span v-if="thread.unreadCount > 0" class="thread-unread-count">{{ thread.unreadCount }}</span>
                </span>
                <span class="inbox-list-preview">{{ previewText(thread.lastMessage.content) }}</span>
                <span class="inbox-list-meta">
                  <span>{{ thread.subtitle }}</span>
                  <span class="thread-time" :title="formatExactTime(thread.lastMessage.created_at)">
                    {{ formatRelativeTime(thread.lastMessage.created_at) }}
                  </span>
                </span>
              </span>
            </button>

            <div v-if="total > pageSize" class="inbox-pagination">
              <el-pagination
                v-model:current-page="page"
                :page-size="pageSize"
                :total="total"
                small
                layout="prev, pager, next"
                @current-change="loadInbox"
              />
            </div>
          </section>

          <section class="inbox-detail-pane" v-loading="detailLoading">
            <el-empty
              v-if="!selectedThread"
              description="选择一个消息源查看通知"
              :image-size="96"
            />

            <article v-else class="inbox-detail">
              <header class="inbox-detail-header">
                <div>
                  <h3>{{ selectedThread.title }}</h3>
                  <div class="inbox-detail-meta">
                    <span>{{ selectedThread.subtitle }}</span>
                    <span>{{ selectedThread.count }} 条消息</span>
                    <el-tag v-if="selectedThread.unreadCount > 0" size="small" type="primary">
                      {{ selectedThread.unreadCount }} 条未读
                    </el-tag>
                    <el-tag v-else size="small" type="info">已读</el-tag>
                  </div>
                </div>
                <el-button
                  v-if="selectedThread.unreadCount > 0"
                  size="small"
                  type="primary"
                  plain
                  @click="markThreadRead(selectedThread)"
                >
                  全部已读
                </el-button>
              </header>

              <section class="inbox-source-card">
                <div class="source-avatar">
                  <el-icon><component :is="threadIcon(selectedThread)" /></el-icon>
                </div>
                <div class="source-copy">
                  <div class="source-title-row">
                    <strong>{{ selectedThread.title }}</strong>
                    <el-tag v-if="sourceTypeText(selectedThread.lastMessage)" size="small" effect="plain">
                      {{ sourceTypeText(selectedThread.lastMessage) }}
                    </el-tag>
                  </div>
                  <div class="source-subtitle">{{ selectedThread.path || selectedThread.subtitle }}</div>
                </div>
                <div class="source-actions">
                  <el-button
                    v-if="sourcePathForMessage(selectedThread.lastMessage)"
                    size="small"
                    plain
                    @click="openSourcePath(selectedThread.lastMessage)"
                  >
                    查看来源
                  </el-button>
                </div>
              </section>

              <div class="inbox-message-stream">
                <article
                  v-for="message in selectedThreadMessages"
                  :key="message.id"
                  class="inbox-message-card"
                  :class="{ 'is-unread': !message.read_at, 'is-active': selectedId === message.id }"
                  @click="selectMessage(message)"
                >
                  <header class="message-card-header">
                    <div class="message-card-title">
                      <strong>{{ message.title || '无标题消息' }}</strong>
                      <el-tag v-if="sourceTypeText(message)" size="small" effect="plain">
                        {{ sourceTypeText(message) }}
                      </el-tag>
                      <el-tag v-if="!message.read_at" size="small" type="primary">未读</el-tag>
                    </div>
                    <span :title="formatExactTime(message.created_at)">
                      {{ formatRelativeTime(message.created_at) }}
                    </span>
                  </header>
                  <div class="message-card-source">{{ sourceSecondaryText(message) }}</div>
                  <div class="inbox-content inbox-rich-content" v-html="renderMessageContent(message)" />
                  <footer class="message-card-actions">
                    <el-button
                      v-if="message.scheduled_task_id || selectedThread?.scheduledTaskID"
                      size="small"
                      type="primary"
                      plain
                      @click.stop="openScheduledExecution(message)"
                    >
                      查看执行
                    </el-button>
                    <el-button
                      v-if="message.workspace_session_id"
                      size="small"
                      type="primary"
                      plain
                      @click.stop="openWorkspaceSession(message)"
                    >
                      查看会话
                    </el-button>
                    <el-button
                      v-if="sourcePathForMessage(message)"
                      size="small"
                      plain
                      @click.stop="openSourcePath(message)"
                    >
                      查看来源
                    </el-button>
                    <el-button
                      v-if="!message.read_at"
                      size="small"
                      plain
                      @click.stop="markMessageRead(message.id)"
                    >
                      标记已读
                    </el-button>
                  </footer>
                </article>
              </div>
            </article>
          </section>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import { ElMessage } from 'element-plus'
import {
  ChatDotRound,
  Document as DocumentIcon,
  FolderOpened,
  Message as MessageIcon,
  Refresh,
  Timer,
} from '@element-plus/icons-vue'
import { Z_INDEX } from '@/architecture/presentation/constants/zIndex'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'
import { escapeHtml, sanitizeHtml } from '@/architecture/shared/sanitizeHtml'
import {
  getMessageInboxItem,
  getMessageInboxUnreadCount,
  listMessageInbox,
  listMessageInboxThreads,
  markAllMessageInboxItemsRead,
  markMessageInboxItemRead,
  type MessageInboxItem,
  type MessageInboxThread,
  type MessageInboxStatus,
} from '@/architecture/presentation/context/api/message'

const props = withDefaults(defineProps<{
  showTrigger?: boolean
}>(), {
  showTrigger: true
})

const emit = defineEmits<{
  (e: 'messages-updated'): void
}>()

interface InboxThread {
  key: string
  title: string
  subtitle: string
  path?: string
  kind: MessageInboxThread['kind']
  lastMessage: MessageInboxItem
  unreadCount: number
  count: number
  scheduledTaskID?: number
  scheduledExecutionID?: number
}

interface SourceFilter {
  sourcePath: string
  title?: string
  includeChildren?: boolean
  kind?: MessageInboxThread['kind']
}

const router = useRouter()
const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()
const drawerVisible = ref(false)
const countLoading = ref(false)
const listLoading = ref(false)
const detailLoading = ref(false)
const errorMessage = ref('')
const unreadCount = ref(0)
const inboxThreads = ref<InboxThread[]>([])
const threadMessages = ref<MessageInboxItem[]>([])
const selectedMessage = ref<MessageInboxItem | null>(null)
const selectedId = computed(() => selectedMessage.value?.id ?? null)
const selectedThreadKey = ref('')
const page = ref(1)
const pageSize = 20
const total = ref(0)
const statusFilter = ref<MessageInboxStatus>('all')
const statusOptions = [
  { label: '全部', value: 'all' },
  { label: '未读', value: 'unread' },
]
const sourceFilter = ref<SourceFilter | null>(null)
const drawerTitle = computed(() => {
  if (!sourceFilter.value) return '站内信'
  return `${sourceFilter.value.title || '节点通知'} · 通知`
})
const selectedThread = computed(() => {
  return inboxThreads.value.find(thread => thread.key === selectedThreadKey.value)
    || inboxThreads.value[0]
    || null
})
const currentScopeUnreadCount = computed(() => {
  if (sourceFilter.value) {
    return selectedThread.value?.unreadCount || 0
  }
  return unreadCount.value
})
const selectedThreadMessages = computed(() => {
  return threadMessages.value
    .slice()
    .sort((a, b) => messageTime(b) - messageTime(a)) || []
})

onMounted(() => {
  void loadUnreadCount()
})

async function loadUnreadCount() {
  countLoading.value = true
  try {
    const resp = await getMessageInboxUnreadCount()
    unreadCount.value = resp.unread_count || 0
  } catch {
    unreadCount.value = 0
  } finally {
    countLoading.value = false
  }
}

function openDrawer() {
  sourceFilter.value = null
  drawerVisible.value = true
}

function openForSource(filter: SourceFilter) {
  const sourcePath = (filter.sourcePath || '').trim()
  if (!sourcePath) return
  sourceFilter.value = {
    ...filter,
    sourcePath,
  }
  const wasVisible = drawerVisible.value
  drawerVisible.value = true
  if (wasVisible) {
    void loadInbox(true)
  }
}

function clearSourceFilter() {
  sourceFilter.value = null
  void loadInbox(true)
}

function handleDrawerOpen() {
  void loadInbox(true)
  void loadUnreadCount()
}

async function loadInbox(resetPage = false) {
  if (resetPage) {
    page.value = 1
  }
  listLoading.value = true
  errorMessage.value = ''
  try {
    if (sourceFilter.value?.sourcePath) {
      await loadSourceInbox()
      return
    }
    const resp = await listMessageInboxThreads({
      status: statusFilter.value === 'unread' ? 'unread' : undefined,
      page: page.value,
      page_size: pageSize,
    })
    inboxThreads.value = (resp.list || []).map(apiThreadToInboxThread)
    total.value = resp.total || 0
    if (!inboxThreads.value.some(thread => thread.key === selectedThreadKey.value)) {
      selectedThreadKey.value = inboxThreads.value[0]?.key || ''
    }
    const current = selectedThread.value
    if (current) {
      selectedMessage.value = current.lastMessage
      await loadThreadMessages(current)
    } else {
      selectedMessage.value = null
      threadMessages.value = []
    }
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '加载站内信失败'
  } finally {
    listLoading.value = false
  }
}

async function loadSourceInbox() {
  const filter = sourceFilter.value
  if (!filter?.sourcePath) return
  const resp = await listMessageInbox({
    status: statusFilter.value === 'unread' ? 'unread' : undefined,
    source_path: filter.sourcePath,
    include_children: Boolean(filter.includeChildren),
    page: page.value,
    page_size: 100,
  })
  const messages = resp.list || []
  threadMessages.value = messages
  total.value = resp.total || 0
  if (messages.length === 0) {
    inboxThreads.value = []
    selectedThreadKey.value = ''
    selectedMessage.value = null
    return
  }
  const firstMessage = messages[0]
  if (!firstMessage) return
  const unreadCount = messages.filter(item => !item.read_at).length
  const thread: InboxThread = {
    key: sourceFilterThreadKey(filter),
    title: filter.title || sourcePrimaryText(firstMessage),
    subtitle: filter.includeChildren ? '当前节点及子节点通知' : '当前节点通知',
    path: filter.sourcePath,
    kind: filter.kind || threadKind(firstMessage),
    lastMessage: firstMessage,
    unreadCount,
    count: Number(resp.total || messages.length),
    scheduledTaskID: firstMessage.scheduled_task_id,
    scheduledExecutionID: firstMessage.scheduled_execution_id,
  }
  inboxThreads.value = [thread]
  selectedThreadKey.value = thread.key
  selectedMessage.value = firstMessage
}

function selectThread(thread: InboxThread) {
  selectedThreadKey.value = thread.key
  selectedMessage.value = thread.lastMessage
  void loadThreadMessages(thread).then(() => markThreadRead(thread))
}

async function loadThreadMessages(thread: InboxThread) {
  detailLoading.value = true
  errorMessage.value = ''
  try {
    if (sourceFilter.value?.sourcePath) {
      await loadSourceInbox()
      return
    }
    const resp = await listMessageInbox({
      thread_key: thread.key,
      page: 1,
      page_size: 100,
    })
    threadMessages.value = resp.list || []
  } catch (error) {
    threadMessages.value = [thread.lastMessage]
    errorMessage.value = error instanceof Error ? error.message : '加载消息会话失败'
  } finally {
    detailLoading.value = false
  }
}

async function selectMessage(item: MessageInboxItem) {
  selectedMessage.value = item
  detailLoading.value = true
  errorMessage.value = ''
  try {
    const detail = await getMessageInboxItem(item.id)
    selectedMessage.value = detail
    if (!detail.read_at) {
      await markMessageInboxItemRead(item.id)
      selectedMessage.value = { ...detail, read_at: new Date().toISOString() }
      updateListReadState(item.id)
      await loadUnreadCount()
      emit('messages-updated')
    }
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '加载消息详情失败'
  } finally {
    detailLoading.value = false
  }
}

async function markThreadRead(thread: InboxThread) {
  const unreadMessages = (selectedThreadKey.value === thread.key ? threadMessages.value : [thread.lastMessage])
    .filter(item => !item.read_at)
  if (unreadMessages.length === 0) return
  try {
    await Promise.all(unreadMessages.map(item => markMessageInboxItemRead(item.id)))
    const now = new Date().toISOString()
    const ids = new Set(unreadMessages.map(item => item.id))
    threadMessages.value = threadMessages.value.map(item => {
      if (!ids.has(item.id)) return item
      return { ...item, read_at: item.read_at || now }
    })
    inboxThreads.value = inboxThreads.value.map(item => {
      if (item.key !== thread.key) return item
      return {
        ...item,
        unreadCount: 0,
        lastMessage: ids.has(item.lastMessage.id)
          ? { ...item.lastMessage, read_at: item.lastMessage.read_at || now }
          : item.lastMessage,
      }
    })
    if (selectedMessage.value && ids.has(selectedMessage.value.id)) {
      selectedMessage.value = { ...selectedMessage.value, read_at: selectedMessage.value.read_at || now }
    }
    await loadUnreadCount()
    emit('messages-updated')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '标记已读失败')
  }
}

async function markMessageRead(id: number) {
  try {
    await markMessageInboxItemRead(id)
    updateListReadState(id)
    if (selectedMessage.value?.id === id) {
      selectedMessage.value = { ...selectedMessage.value, read_at: new Date().toISOString() }
    }
    await loadUnreadCount()
    emit('messages-updated')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '标记已读失败')
  }
}

async function markCurrentScopeRead() {
  if (sourceFilter.value && selectedThread.value) {
    await markThreadRead(selectedThread.value)
    return
  }
  await markAllRead()
}

async function markAllRead() {
  try {
    await markAllMessageInboxItemsRead()
    const now = new Date().toISOString()
    threadMessages.value = threadMessages.value.map(item => ({ ...item, read_at: item.read_at || now }))
    inboxThreads.value = inboxThreads.value.map(thread => ({
      ...thread,
      unreadCount: 0,
      lastMessage: { ...thread.lastMessage, read_at: thread.lastMessage.read_at || now },
    }))
    if (selectedMessage.value) {
      selectedMessage.value = { ...selectedMessage.value, read_at: selectedMessage.value.read_at || now }
    }
    unreadCount.value = 0
    ElMessage.success('已全部标记为已读')
    emit('messages-updated')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '全部已读失败')
  }
}

function updateListReadState(id: number) {
  threadMessages.value = threadMessages.value.map(item => {
    if (item.id !== id) return item
    return { ...item, read_at: item.read_at || new Date().toISOString() }
  })
  inboxThreads.value = inboxThreads.value.map(thread => {
    if (thread.lastMessage.id !== id && thread.key !== selectedThreadKey.value) return thread
    const decrement = thread.key === selectedThreadKey.value && thread.unreadCount > 0 ? 1 : 0
    return {
      ...thread,
      unreadCount: Math.max(0, thread.unreadCount - decrement),
      lastMessage: thread.lastMessage.id === id
        ? { ...thread.lastMessage, read_at: thread.lastMessage.read_at || new Date().toISOString() }
        : thread.lastMessage,
    }
  })
}

function previewText(content?: string) {
  const text = stripHtml(content || '').replace(/\s+/g, ' ').trim()
  return text.length > 90 ? `${text.slice(0, 90)}...` : text || '无内容'
}

function apiThreadToInboxThread(thread: MessageInboxThread): InboxThread {
  return {
    key: thread.key,
    title: thread.title || threadTitle(thread.last_message),
    subtitle: thread.subtitle || threadSubtitle(thread.last_message, thread.message_count || 1),
    path: thread.path || threadPath(thread.last_message),
    kind: thread.kind || threadKind(thread.last_message),
    lastMessage: thread.last_message,
    unreadCount: Number(thread.unread_count || 0),
    count: Number(thread.message_count || 0),
    scheduledTaskID: thread.scheduled_task_id || thread.last_message.scheduled_task_id,
    scheduledExecutionID: thread.scheduled_execution_id || thread.last_message.scheduled_execution_id,
  }
}

function sourceFilterThreadKey(filter: SourceFilter) {
  return `source:${filter.sourcePath}:${filter.includeChildren ? 'children' : 'direct'}`
}

function messageTime(item: MessageInboxItem) {
  const parsed = dayjs(item.created_at)
  return parsed.isValid() ? parsed.valueOf() : 0
}

function threadKeyForMessage(item: MessageInboxItem) {
  const parentPath = sourceParentPathForMessage(item)
  if (parentPath) return `directory:${parentPath}`
  const sourcePath = sourcePathForMessage(item)
  if (sourcePath) return `source:${sourcePath}`
  if (item.workspace_session_id) return `session:${item.workspace_session_id}`
  return `sender:${item.from || 'system'}`
}

function threadTitle(item: MessageInboxItem) {
  return item.source_display?.parent_name
    || item.source_parent_title
    || item.source_display?.name
    || item.source_title
    || item.from
    || 'system'
}

function threadSubtitle(item: MessageInboxItem, count: number) {
  const sourceName = sourceSecondaryText(item)
  const suffix = count > 1 ? ` · ${count} 条消息` : ''
  return `${sourceName}${suffix}`
}

function threadPath(item: MessageInboxItem) {
  return sourceParentPathForMessage(item) || sourcePathForMessage(item)
}

function threadKind(item: MessageInboxItem): InboxThread['kind'] {
  if (sourceParentPathForMessage(item)) return 'directory'
  if (item.workspace_session_id) return 'session'
  if (sourcePathForMessage(item)) return 'function'
  return 'sender'
}

function threadIcon(thread: InboxThread) {
  if (thread.kind === 'directory') return FolderOpened
  if (thread.kind === 'session') return ChatDotRound
  if (thread.lastMessage.source_type === 'scheduled_task') return Timer
  return DocumentIcon
}

function sourcePrimaryText(item?: MessageInboxItem | null) {
  if (!item) return 'system'
  return item.source_display?.parent_name
    || item.source_parent_title
    || item.source_display?.name
    || item.source_title
    || item.from
    || 'system'
}

function sourceSecondaryText(item?: MessageInboxItem | null) {
  if (!item) return '-'
  const functionName = item.source_display?.name || item.source_title || ''
  const parentName = item.source_display?.parent_name || item.source_parent_title || ''
  if (functionName && functionName !== parentName) return functionName
  if (item.workspace_session_title) return item.workspace_session_title
  return item.source_path || item.full_code_path || item.from || '-'
}

function sourceTypeText(item?: MessageInboxItem | null) {
  const type = (item?.source_type || item?.client_source || '').trim()
  const map: Record<string, string> = {
    scheduled_task: '定时任务',
    agent_session: '定时会话',
    agent_tool: '智能体',
    public_share: '公开分享',
    openapi_token: 'OpenAPI',
    sdk_function: '函数',
  }
  return map[type] || type
}

function sourcePathForMessage(item?: MessageInboxItem | null) {
  return item?.source_display?.full_code_path || item?.source_path || item?.full_code_path || ''
}

function sourceParentPathForMessage(item?: MessageInboxItem | null) {
  return item?.source_display?.parent_full_code_path || item?.source_parent_path || ''
}

function workspacePathForMessage(item: MessageInboxItem) {
  return sourceParentPathForMessage(item) || sourcePathForMessage(item)
}

function workspaceRoutePath(fullCodePath?: string) {
  const normalized = (fullCodePath || '').trim()
  if (!normalized) return ''
  return `/workspace${normalized.startsWith('/') ? normalized : `/${normalized}`}`
}

async function openSourcePath(item: MessageInboxItem) {
  const path = workspaceRoutePath(sourcePathForMessage(item))
  if (!path) return
  drawerVisible.value = false
  await router.push({ path })
}

async function openWorkspaceSession(item: MessageInboxItem) {
  const sessionId = (item.workspace_session_id || '').trim()
  const fullCodePath = workspacePathForMessage(item)
  const path = workspaceRoutePath(fullCodePath)
  if (!sessionId || !path) return
  drawerVisible.value = false
  await router.push({
    path,
    query: {
      _mws: 'open',
      _mws_sid: sessionId,
      _mws_path: fullCodePath,
      _mws_name: sourcePrimaryText(item),
      _mws_expanded: '1',
      _mws_maximized: '1',
    },
  })
}

async function openScheduledExecution(item: MessageInboxItem) {
  const taskID = item.scheduled_task_id || selectedThread.value?.scheduledTaskID || 0
  if (!taskID) return
  const executionID = item.scheduled_execution_id || selectedThread.value?.scheduledExecutionID || 0
  const fullCodePath = workspacePathForMessage(item)
  const path = workspaceRoutePath(fullCodePath)
  if (!path) return
  drawerVisible.value = false
  await router.push({
    path,
    query: {
      _scheduled: 'open',
      _scheduled_task_id: String(taskID),
      ...(executionID ? { _scheduled_execution_id: String(executionID) } : {}),
      _scheduled_kind: item.workspace_session_id ? 'agent' : 'function',
    },
  })
}

function renderMessageContent(item: MessageInboxItem) {
  const content = item.content || ''
  const type = (item.content_type || 'markdown').toLowerCase()
  if (type === 'html') return sanitizeHtml(content)
  if (type === 'text' || type === 'plain') return escapeHtml(content).replace(/\n/g, '<br>')
  return renderMarkdown(content)
}

function stripHtml(content: string) {
  if (!content.includes('<')) return content
  if (typeof DOMParser === 'undefined') return content.replace(/<[^>]*>/g, ' ')
  const doc = new DOMParser().parseFromString(sanitizeHtml(content), 'text/html')
  return doc.body.textContent || ''
}

function formatExactTime(value?: string) {
  if (!value) return '-'
  const parsed = dayjs(value)
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm') : value
}

function formatRelativeTime(value?: string) {
  if (!value) return '-'
  const parsed = dayjs(value)
  if (!parsed.isValid()) return value

  const diffMs = Date.now() - parsed.valueOf()
  const absDiffMs = Math.abs(diffMs)
  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour

  if (absDiffMs < minute) return '刚刚'
  if (diffMs < 0) {
    if (absDiffMs < hour) return `${Math.floor(absDiffMs / minute)}分钟后`
    if (absDiffMs < day) return `${Math.floor(absDiffMs / hour)}小时后`
    return parsed.format('MM-DD HH:mm')
  }
  if (diffMs < hour) return `${Math.floor(diffMs / minute)}分钟前`
  if (diffMs < day) return `${Math.floor(diffMs / hour)}小时前`
  if (diffMs < 30 * day) return `${Math.floor(diffMs / day)}天前`
  if (diffMs < 365 * day) return parsed.format('MM-DD HH:mm')
  return parsed.format('YYYY-MM-DD')
}

defineExpose({
  openDrawer,
  openForSource,
})
</script>

<style scoped lang="scss">
.workspace-inbox {
  display: inline-flex;
  align-items: center;
}

.workspace-inbox-button {
  width: 40px;
  height: 40px;
  border-color: var(--app-shell-panel-border);
  background: var(--app-shell-panel-muted-bg);
  color: var(--el-text-color-primary);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);

  &:hover {
    color: var(--el-color-primary);
    border-color: var(--el-color-primary-light-5);
    background: var(--el-color-primary-light-9);
  }
}

.workspace-inbox-badge :deep(.el-badge__content) {
  box-shadow: 0 0 0 2px var(--app-shell-panel-bg);
}

.inbox-shell {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  gap: 12px;
}

.inbox-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.inbox-filter {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
}

.source-filter-chip {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
  color: var(--el-text-color-secondary);
  font-size: 13px;

  span:not(.el-tag__content) {
    max-width: 280px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.inbox-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.inbox-error {
  flex-shrink: 0;
}

.inbox-layout {
  display: grid;
  min-height: 0;
  flex: 1;
  grid-template-columns: minmax(300px, 360px) minmax(0, 1fr);
  gap: 14px;
}

.inbox-list-pane,
.inbox-detail-pane {
  min-height: 0;
  overflow: auto;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 14px;
  background: var(--app-shell-panel-muted-bg);
}

.inbox-list-pane {
  padding: 8px;
}

.inbox-list-item {
  display: grid;
  width: 100%;
  grid-template-columns: 42px minmax(0, 1fr);
  gap: 10px;
  align-items: flex-start;
  padding: 12px 10px;
  border: 1px solid transparent;
  border-radius: 11px;
  background: transparent;
  color: var(--el-text-color-primary);
  cursor: pointer;
  text-align: left;
  transition: background 0.16s ease, border-color 0.16s ease, box-shadow 0.16s ease;

  &:hover,
  &.is-active {
    border-color: rgba(var(--el-color-primary-rgb), 0.2);
    background: rgba(var(--el-color-primary-rgb), 0.08);
  }

  &.is-unread .inbox-list-title {
    font-weight: 800;
  }
}

.thread-avatar {
  display: grid;
  width: 42px;
  height: 42px;
  margin-top: 1px;
  place-items: center;
  border: 1px solid rgba(var(--el-color-primary-rgb), 0.18);
  border-radius: 10px;
  background: var(--app-shell-panel-bg);
  color: var(--el-color-primary);
  font-size: 19px;
}

.inbox-list-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 5px;
}

.thread-title-row {
  display: flex;
  min-width: 0;
  min-height: 22px;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.thread-unread-count {
  display: inline-flex;
  min-width: 20px;
  height: 20px;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--el-color-primary);
  color: #fff;
  font-size: 11px;
  font-weight: 800;
}

.inbox-list-title,
.inbox-list-preview {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.inbox-list-title {
  min-width: 0;
  font-size: 13px;
  font-weight: 650;
  line-height: 22px;
}

.inbox-list-preview {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.inbox-list-meta {
  display: flex;
  min-width: 0;
  justify-content: space-between;
  gap: 8px;
  color: var(--el-text-color-placeholder);
  font-size: 11px;

  span:first-child {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.thread-time {
  flex-shrink: 0;
}

.inbox-pagination {
  display: flex;
  justify-content: center;
  padding: 10px 0 4px;
}

.inbox-detail-pane {
  padding: 18px;
}

.inbox-detail {
  display: flex;
  min-height: 100%;
  flex-direction: column;
  gap: 18px;
}

.inbox-detail-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--app-shell-panel-border);

  h3 {
    margin: 0 0 8px;
    color: var(--el-text-color-primary);
    font-size: 18px;
    line-height: 1.35;
  }
}

.inbox-detail-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.inbox-content {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  color: var(--el-text-color-primary);
  font-size: 14px;
  line-height: 1.72;
}

.inbox-rich-content {
  white-space: normal;

  :deep(p) {
    margin: 0 0 8px;
  }

  :deep(p:last-child) {
    margin-bottom: 0;
  }

  :deep(a) {
    color: var(--el-color-primary);
    text-decoration: none;
  }

  :deep(a:hover) {
    text-decoration: underline;
  }

  :deep(ul),
  :deep(ol) {
    margin: 6px 0 8px;
    padding-left: 20px;
  }

  :deep(blockquote) {
    margin: 8px 0;
    padding: 8px 10px;
    border-left: 3px solid var(--el-color-primary-light-5);
    background: var(--app-shell-panel-muted-bg);
    color: var(--el-text-color-secondary);
  }

  :deep(code) {
    padding: 1px 4px;
    border-radius: 4px;
    background: var(--app-shell-panel-muted-bg);
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 12px;
  }

  :deep(pre) {
    overflow: auto;
    margin: 8px 0;
    padding: 10px;
    border-radius: 8px;
    background: var(--app-shell-panel-muted-bg);
  }

  :deep(pre code) {
    padding: 0;
    background: transparent;
  }

  :deep(table) {
    display: block;
    max-width: 100%;
    overflow: auto;
    border-collapse: collapse;
    margin: 8px 0;
  }

  :deep(th),
  :deep(td) {
    padding: 6px 8px;
    border: 1px solid var(--app-shell-panel-border);
  }

  :deep(img),
  :deep(video) {
    max-width: 100%;
    border-radius: 8px;
  }
}

.inbox-source-card {
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 12px;
  border: 1px solid rgba(var(--el-color-primary-rgb), 0.18);
  border-radius: 12px;
  background: color-mix(in srgb, var(--app-shell-panel-bg) 84%, var(--el-color-primary-light-9));
}

.source-avatar {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  border-radius: 10px;
  background: var(--el-color-primary);
  color: #fff;
  font-size: 20px;
  font-weight: 800;
}

.source-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.source-title-row {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;

  strong {
    min-width: 0;
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-size: 14px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.source-subtitle,
.source-session {
  min-width: 0;
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-session {
  color: var(--el-text-color-placeholder);
}

.source-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.inbox-message-stream {
  display: flex;
  min-height: 0;
  flex-direction: column;
  gap: 12px;
}

.inbox-message-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 12px;
  background: var(--app-shell-panel-bg);
  cursor: pointer;
  transition: border-color 0.16s ease, box-shadow 0.16s ease, transform 0.16s ease;

  &:hover,
  &.is-active {
    border-color: rgba(var(--el-color-primary-rgb), 0.28);
    box-shadow: 0 8px 24px rgba(15, 23, 42, 0.08);
  }

  &.is-unread {
    border-color: rgba(var(--el-color-primary-rgb), 0.32);
  }
}

.message-card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.message-card-title {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;

  strong {
    min-width: 0;
    overflow: hidden;
    color: var(--el-text-color-primary);
    font-size: 14px;
    line-height: 1.4;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.message-card-source {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.message-card-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.inbox-source {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 8px 10px;
  padding: 12px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 12px;
  background: var(--app-shell-panel-bg);
  color: var(--el-text-color-secondary);
  font-size: 12px;

  dt {
    font-weight: 700;
  }

  dd {
    min-width: 0;
    margin: 0;
    overflow-wrap: anywhere;
  }
}

@media (max-width: 760px) {
  .inbox-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .inbox-actions {
    justify-content: flex-end;
  }

  .inbox-layout {
    grid-template-columns: 1fr;
  }

  .inbox-list-pane {
    max-height: 38vh;
  }

  .inbox-source-card {
    grid-template-columns: 42px minmax(0, 1fr);
  }

  .source-actions {
    grid-column: 1 / -1;
    justify-content: flex-start;
  }
}
</style>
