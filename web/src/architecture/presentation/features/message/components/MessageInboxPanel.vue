<template>
  <div class="message-inbox-panel">
    <header class="message-inbox-header">
      <div class="message-inbox-title-wrap">
        <span class="message-inbox-signal">
          <el-icon><Bell /></el-icon>
        </span>
        <div>
          <span class="message-inbox-kicker">消息控制台</span>
          <h2>消息中心</h2>
          <p>{{ totalUnreadCount > 0 ? `${totalUnreadCount} 条未读消息待处理` : '当前消息链路已清空' }}</p>
        </div>
      </div>
      <div class="message-inbox-header-side">
        <div class="message-inbox-metric">
          <span>未读</span>
          <strong>{{ totalUnreadCount }}</strong>
        </div>
        <el-button
          v-if="closable"
          class="message-inbox-close"
          text
          circle
          :icon="Close"
          @click="$emit('close')"
        />
      </div>
    </header>

    <div class="message-inbox-toolbar">
      <div class="message-filter-tabs" role="tablist" aria-label="消息筛选">
        <button
          type="button"
          class="message-filter-tab"
          :class="{ 'is-active': !onlyUnread }"
          @click="onlyUnread = false"
        >
          全部会话
        </button>
        <button
          type="button"
          class="message-filter-tab"
          :class="{ 'is-active': onlyUnread }"
          @click="onlyUnread = true"
        >
          未读
        </button>
      </div>
      <div class="message-inbox-toolbar-actions">
        <el-button class="message-action-btn" size="small" :icon="Refresh" :loading="loading" @click="loadMessages">刷新</el-button>
        <el-button
          class="message-action-btn message-action-btn--primary"
          size="small"
          :icon="CircleCheck"
          :disabled="totalUnreadCount === 0"
          @click="handleReadAll"
        >
          全部已读
        </el-button>
      </div>
    </div>

    <section class="message-chat-layout" v-loading="loading">
      <aside class="conversation-pane">
        <div class="conversation-pane-head">
          <span>会话</span>
          <strong>{{ displayConversations.length }}</strong>
        </div>

        <el-empty
          v-if="displayConversations.length === 0 && !loading"
          description="暂无消息会话"
          :image-size="86"
        />

        <button
          v-for="conversation in displayConversations"
          :key="conversation.key"
          type="button"
          class="conversation-item"
          :class="{ 'is-active': activeConversationKey === conversation.key }"
          @click="selectConversation(conversation)"
        >
          <span class="conversation-avatar">
            <img
              :src="getSourceIconSrc(conversation.sourceType, conversation.sourceTemplateType)"
              :alt="getSourceIconAlt(conversation.sourceType, conversation.sourceTemplateType)"
              class="message-source-icon"
              :class="getSourceIconClass(conversation.sourceType, conversation.sourceTemplateType)"
            />
          </span>
          <span class="conversation-main">
            <span class="conversation-title-row">
              <span class="conversation-title" :title="conversation.title">{{ conversation.title }}</span>
              <time>{{ formatConversationTime(conversation.updatedAt) }}</time>
            </span>
            <span class="conversation-preview" :title="getMessageTitlePreview(conversation.lastMessage)">
              {{ getMessageTitlePreview(conversation.lastMessage) }}
            </span>
          </span>
          <span v-if="conversation.unreadCount > 0" class="conversation-unread">{{ conversation.unreadCount }}</span>
        </button>
      </aside>

      <main v-if="selectedConversation" class="chat-pane">
        <header class="chat-pane-header">
          <div class="chat-peer">
            <span class="chat-peer-avatar">
              <img
                :src="getSourceIconSrc(selectedConversation.sourceType, selectedConversation.sourceTemplateType)"
                :alt="getSourceIconAlt(selectedConversation.sourceType, selectedConversation.sourceTemplateType)"
                class="message-source-icon"
                :class="getSourceIconClass(selectedConversation.sourceType, selectedConversation.sourceTemplateType)"
              />
            </span>
            <div class="chat-peer-copy">
              <h3>{{ selectedConversation.title }}</h3>
              <p :title="getMessageTitlePreview(selectedConversation.lastMessage)">
                {{ getMessageTitlePreview(selectedConversation.lastMessage) }}
              </p>
            </div>
          </div>
          <div class="chat-peer-meta">
            <span>{{ selectedConversation.messages.length }} 条消息</span>
            <span v-if="selectedConversation.unreadCount > 0">{{ selectedConversation.unreadCount }} 未读</span>
          </div>
        </header>

        <div class="chat-thread" ref="chatThreadRef">
          <div
            v-for="message in selectedConversation.messages"
            :key="message.id"
            class="chat-message-row"
            :class="{ 'is-unread': !message.read_at }"
          >
            <article class="chat-bubble">
              <header class="chat-bubble-head">
                <UserDisplay
                  :username="getSenderLabel(message)"
                  mode="simple"
                  size="small"
                  class="message-user-display message-user-display--message"
                />
                <div class="chat-bubble-meta">
                  <time>{{ formatTime(message.created_at) }}</time>
                  <span v-if="!message.read_at" class="chat-message-status">未读</span>
                </div>
              </header>
              <div v-if="message.title || hasSourceContext(message)" class="chat-message-context">
                <span v-if="message.title" class="chat-message-context-title">{{ message.title }}</span>
                <span
                  v-if="getSourceName(message)"
                  class="chat-message-context-source"
                  :title="getSourceName(message)"
                >
                  {{ getSourceName(message) }}
                </span>
              </div>
              <div class="chat-message-content" v-html="renderMessageContent(message)" />
              <div v-if="hasMessageMeta(message)" class="chat-message-foot">
                <span v-if="message.client_source">来源端 {{ getClientSourceLabel(message.client_source) }}</span>
                <span v-if="message.source_type">类型 {{ getSourceTypeLabel(message.source_type) }}</span>
                <span v-if="message.source_ref">引用 {{ message.source_ref }}</span>
                <span v-if="message.trace_id">链路 {{ message.trace_id }}</span>
              </div>
            </article>
          </div>
        </div>
      </main>

      <main v-else class="chat-empty-pane">
        <span class="chat-empty-orbit">
          <el-icon><Bell /></el-icon>
        </span>
        <strong>选择一个会话查看消息</strong>
        <p>消息会按来源目录或函数聚合，像聊天一样连续查看。</p>
      </main>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { Bell, CircleCheck, Close, Refresh } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'
import {
  getMessageUnreadCount,
  listInboxMessages,
  markAllInboxMessagesRead,
  markInboxMessageRead,
  type MessageInboxItem
} from '@/architecture/infrastructure/api/message'
import { getServiceTreeDetail, type ServiceTreeDetailResp } from '@/architecture/infrastructure/api/service-tree'
import UserDisplay from '@/architecture/presentation/shared/components/UserDisplay.vue'
import type { ServiceTree } from '@/architecture/domain/types'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'
import { sanitizeHtml } from '@/architecture/shared/sanitizeHtml'
import { findNodeByPath } from '@/architecture/domain/utils/serviceTreeUtils'

const props = withDefaults(defineProps<{
  closable?: boolean
  serviceTree?: ServiceTree[]
}>(), {
  closable: true,
  serviceTree: () => []
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'unread-count-change', value: number): void
}>()

interface MessageConversation {
  key: string
  title: string
  sourceType: string
  sourceTemplateType: string
  unreadCount: number
  updatedAt: string
  lastMessage: MessageInboxItem
  messages: MessageInboxItem[]
}

interface MessageSourceInfo {
  name: string
  type: string
  templateType: string
}

type SourceDetailInfo = Pick<ServiceTreeDetailResp, 'name' | 'type' | 'full_code_path' | 'template_type'>

const loading = ref(false)
const onlyUnread = ref(false)
const messages = ref<MessageInboxItem[]>([])
const activeConversationKey = ref<string | null>(null)
const totalUnreadCount = ref(0)
const markingConversationKey = ref<string | null>(null)
const chatThreadRef = ref<HTMLElement | null>(null)
const sourceDetailMap = ref<Record<string, SourceDetailInfo>>({})
const loadingSourcePaths = new Set<string>()

const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()

const allConversations = computed<MessageConversation[]>(() => {
  const groups = new Map<string, MessageInboxItem[]>()
  for (const message of messages.value) {
    const key = getConversationKey(message)
    const group = groups.get(key)
    if (group) {
      group.push(message)
    } else {
      groups.set(key, [message])
    }
  }

  return Array.from(groups.entries()).map(([key, group]) => {
    const sorted = [...group].sort((a, b) => getMessageTime(a) - getMessageTime(b))
    const lastMessage = sorted[sorted.length - 1]!
    const sourceInfo = getMessageSourceInfo(lastMessage)
    const title = getConversationTitle(lastMessage)
    return {
      key,
      title,
      sourceType: sourceInfo.type,
      sourceTemplateType: sourceInfo.templateType,
      unreadCount: sorted.filter(item => !item.read_at).length,
      updatedAt: lastMessage.created_at,
      lastMessage,
      messages: sorted
    }
  }).sort((a, b) => dayjs(b.updatedAt).valueOf() - dayjs(a.updatedAt).valueOf())
})

const displayConversations = computed(() => {
  if (!onlyUnread.value) return allConversations.value
  return allConversations.value.filter(conversation => {
    return conversation.unreadCount > 0 || conversation.key === activeConversationKey.value
  })
})

const selectedConversation = computed(() => {
  if (!activeConversationKey.value) return displayConversations.value[0] || allConversations.value[0] || null
  return allConversations.value.find(conversation => conversation.key === activeConversationKey.value)
    || displayConversations.value[0]
    || null
})

function getConversationKey(item: MessageInboxItem) {
  const source = getSourcePath(item) || item.source_ref || item.source_type || 'system'
  return source
}

function getSenderLabel(item: MessageInboxItem) {
  return String(item.from || item.request_user || 'system').trim() || 'system'
}

function getSourcePath(item: MessageInboxItem) {
  const displayPath = String(item.source_display?.full_code_path || '').trim()
  const explicitPath = String(item.full_code_path || '').trim()
  const refPath = String(item.source_ref || '').trim()
  return normalizeMessageSourcePath(displayPath || explicitPath || (refPath.startsWith('/') ? refPath : ''))
}

function getSourceLookupPaths(item: MessageInboxItem) {
  const paths: string[] = []
  const pushPath = (path: string) => {
    const normalized = normalizeMessageSourcePath(path)
    if (!normalized || paths.includes(normalized)) return
    paths.push(normalized)
  }

  pushPath(item.source_display?.full_code_path || '')
  pushPath(item.full_code_path || '')
  const refPath = String(item.source_ref || '').trim()
  if (refPath.startsWith('/')) pushPath(refPath)

  const initialCount = paths.length
  for (let index = 0; index < initialCount; index += 1) {
    pushPath(stripFunctionPathSuffix(paths[index] || ''))
  }
  return paths
}

function getSourceDetail(item: MessageInboxItem) {
  for (const path of getSourceLookupPaths(item)) {
    const detail = sourceDetailMap.value[path]
    if (detail) return detail
  }
  return undefined
}

function getMessageSourceInfo(item: MessageInboxItem): MessageSourceInfo {
  const sourceDisplay = item.source_display || undefined
  const sourceNode = getSourceNode(item)
  const path = getSourcePath(item)
  const sourceDetail = getSourceDetail(item)
  const type = getSourceType(item, sourceNode, sourceDetail)
  const templateType = getSourceTemplateType(item, sourceNode, sourceDetail)
  const name = sourceNode?.name
    || sourceDetail?.name
    || sourceDisplay?.name
    || getFallbackSourceName(type, templateType, path)

  return {
    name,
    type,
    templateType
  }
}

function getConversationTitle(item: MessageInboxItem) {
  const sourceInfo = getMessageSourceInfo(item)
  if (sourceInfo.name) return sourceInfo.name
  const sourcePath = getSourcePath(item)
  if (sourcePath) return getFallbackSourceName(sourceInfo.type, sourceInfo.templateType, sourcePath)
  return '未绑定服务目录'
}

function getSourceName(item: MessageInboxItem) {
  return getMessageSourceInfo(item).name
}

function hasSourceContext(item: MessageInboxItem) {
  const sourceInfo = getMessageSourceInfo(item)
  return Boolean(sourceInfo.name)
}

function getSourceType(
  item: MessageInboxItem,
  sourceNode = getSourceNode(item),
  sourceDetail?: SourceDetailInfo
) {
  const templateType = getSourceTemplateType(item, sourceNode, sourceDetail)
  const type = String(sourceNode?.type || sourceDetail?.type || item.source_display?.type || item.source_type || '').trim().toLowerCase()
  if (!type && templateType) return 'function'
  return type
}

function getSourceTemplateType(
  item: MessageInboxItem,
  sourceNode = getSourceNode(item),
  sourceDetail?: SourceDetailInfo
) {
  const explicit = String(sourceNode?.template_type || sourceDetail?.template_type || item.source_display?.template_type || '').trim().toLowerCase()
  if (explicit) return explicit
  return inferTemplateTypeFromPath(getSourcePath(item))
}

function getSourceIconSrc(sourceType: string, sourceTemplateType = '') {
  const type = String(sourceType || '').trim().toLowerCase()
  const templateType = String(sourceTemplateType || '').trim().toLowerCase()
  if (type === 'function') {
    if (templateType === 'table') return '/service-tree/表格.svg'
    if (templateType === 'form') return '/service-tree/编辑.svg'
    if (templateType === 'chart') return '/service-tree/报表.svg'
    return '/service-tree/编辑.svg'
  }
  if (type === 'docs') return '/文档.svg'
  if (type === 'board') return '/讨论区.svg'
  return '/service-tree/custom-folder.svg'
}

function getSourceIconAlt(sourceType: string, sourceTemplateType = '') {
  return `${getFallbackSourceName(sourceType, sourceTemplateType, '') || '服务目录'}图标`
}

function getSourceIconClass(sourceType: string, sourceTemplateType = '') {
  const type = String(sourceType || '').trim().toLowerCase() || 'service'
  const templateType = String(sourceTemplateType || '').trim().toLowerCase()
  return [
    `message-source-icon--${type}`,
    templateType ? `message-source-icon--${templateType}` : ''
  ].filter(Boolean)
}

function getSourceNode(item: MessageInboxItem) {
  if (!props.serviceTree?.length) return null
  for (const sourcePath of getSourceLookupPaths(item)) {
    const node = findNodeByPath(props.serviceTree, sourcePath)
    if (node) return node
  }
  return null
}

function getFallbackSourceName(sourceType: string, templateType: string, path: string) {
  const type = String(sourceType || '').trim().toLowerCase()
  const template = String(templateType || '').trim().toLowerCase()
  if (type === 'function' || template) {
    if (template === 'form') return '表单'
    if (template === 'table') return '表格'
    if (template === 'chart') return '报表'
    return '函数'
  }
  if (type === 'docs') return '文档'
  if (type === 'board') return '讨论区'
  if (type === 'scheduled_task') return '定时任务'
  if (type === 'scheduled_agent_task') return '定时会话'
  if (type === 'agent_tool') return '智能体工具'
  if (type === 'system') return '系统'
  if (!path) return ''
  return '服务目录'
}

function normalizeMessageSourcePath(path: string) {
  return String(path || '')
    .trim()
    .replace(/\/+$/, '')
}

function stripFunctionPathSuffix(path: string) {
  return normalizeMessageSourcePath(path).replace(/\.(form|table|chart)$/i, '')
}

function inferTemplateTypeFromPath(path: string) {
  const matched = String(path || '').match(/\.(table|form|chart)$/i)
  return matched?.[1]?.toLowerCase() || ''
}

function getSourceTypeLabel(sourceType: string) {
  const type = String(sourceType || '').trim()
  const sourceTypeMap: Record<string, string> = {
    function: '函数消息',
    service: '服务目录消息',
    directory: '目录消息',
    catalog: '目录消息',
    scheduled_task: '定时任务',
    scheduled_agent_task: '定时会话',
    agent_tool: '智能体工具',
    system: '系统消息',
    user: '用户消息',
    form: '表单消息',
    table: '表格消息',
    chart: '报表消息'
  }
  return sourceTypeMap[type] || type
}

function getClientSourceLabel(clientSource: string) {
  const source = String(clientSource || '').trim()
  const clientSourceMap: Record<string, string> = {
    browser: '浏览器',
    agent: '智能体',
    scheduled_task: '定时任务',
    scheduled_agent_task: '定时会话',
    server: '服务端',
    api: '接口',
    system: '系统'
  }
  return clientSourceMap[source] || source
}

function getMessageTime(item: MessageInboxItem) {
  return dayjs(item.created_at).valueOf() || 0
}

function getMessageTitlePreview(item: MessageInboxItem) {
  const raw = item.title || '暂无标题'
  return truncateDisplayText(raw.replace(/<[^>]*>/g, '').replace(/[#*_`>\-[\]]/g, ' '), 42)
}

function truncateDisplayText(text: string, limit: number) {
  const normalized = String(text || '').replace(/\s+/g, ' ').trim()
  if (normalized.length <= limit) return normalized
  return `${normalized.slice(0, limit)}...`
}

function hasMessageMeta(item: MessageInboxItem) {
  return Boolean(
    item.client_source
    || item.source_type
    || item.source_ref
    || item.trace_id
  )
}

function renderMessageContent(item: MessageInboxItem) {
  const content = item.content || ''
  const contentType = (item.content_type || 'markdown').toLowerCase()
  if (contentType === 'html') return sanitizeHtml(content)
  return renderMarkdown(content)
}

function formatTime(value?: string | null) {
  if (!value) return ''
  return dayjs(value).format('YYYY-MM-DD HH:mm')
}

function formatConversationTime(value?: string | null) {
  if (!value) return ''
  const time = dayjs(value)
  if (time.isSame(dayjs(), 'day')) return time.format('HH:mm')
  return time.format('MM-DD')
}

function syncActiveConversation() {
  if (activeConversationKey.value && allConversations.value.some(item => item.key === activeConversationKey.value)) {
    return
  }
  activeConversationKey.value = displayConversations.value[0]?.key || allConversations.value[0]?.key || null
}

async function scrollChatToLatest() {
  await nextTick()
  const el = chatThreadRef.value
  if (!el) return
  el.scrollTop = el.scrollHeight
}

async function loadSourceDetailsForMessages(list: MessageInboxItem[]) {
  const paths: string[] = []
  const seen = new Set<string>()
  for (const item of list) {
    for (const path of getSourceLookupPaths(item)) {
      if (!path || seen.has(path) || sourceDetailMap.value[path] || loadingSourcePaths.has(path) || findNodeByPath(props.serviceTree || [], path)) {
        continue
      }
      seen.add(path)
      paths.push(path)
    }
  }
  if (paths.length === 0) return

  paths.forEach(path => loadingSourcePaths.add(path))
  const settled = await Promise.allSettled(paths.map(async (path) => {
    const detail = await getServiceTreeDetail(path)
    return { path, detail }
  }))

  const next = { ...sourceDetailMap.value }
  settled.forEach((result, index) => {
    const path = paths[index]
    if (!path) return
    loadingSourcePaths.delete(path)
    if (result.status === 'fulfilled') {
      const detail = result.value.detail
      if (detail?.name) {
        next[path] = {
          name: detail.name,
          type: detail.type,
          full_code_path: detail.full_code_path || path,
          template_type: detail.template_type
        }
      }
    }
  })
  sourceDetailMap.value = next
}

async function loadUnreadCount() {
  const resp = await getMessageUnreadCount()
  totalUnreadCount.value = resp.unread_count || 0
  emit('unread-count-change', totalUnreadCount.value)
}

async function loadMessages() {
  loading.value = true
  try {
    const [listResp] = await Promise.all([
      listInboxMessages({
        page: 1,
        page_size: 80,
        status: onlyUnread.value ? 'unread' : ''
      }),
      loadUnreadCount()
    ])
    messages.value = listResp.list || []
    await loadSourceDetailsForMessages(messages.value)
    syncActiveConversation()
    await scrollChatToLatest()
  } catch (error: any) {
    ElMessage.error(error?.message || '加载消息失败')
  } finally {
    loading.value = false
  }
}

async function selectConversation(conversation: MessageConversation) {
  activeConversationKey.value = conversation.key
  await markConversationRead(conversation)
}

async function markConversationRead(conversation: MessageConversation) {
  const unreadMessages = conversation.messages.filter(item => !item.read_at)
  if (unreadMessages.length === 0 || markingConversationKey.value === conversation.key) return

  markingConversationKey.value = conversation.key
  const readAt = new Date().toISOString()
  unreadMessages.forEach(item => {
    item.read_at = readAt
  })
  totalUnreadCount.value = Math.max(0, totalUnreadCount.value - unreadMessages.length)
  emit('unread-count-change', totalUnreadCount.value)

  try {
    await Promise.all(unreadMessages.map(item => markInboxMessageRead(item.id)))
    await loadUnreadCount()
  } catch (error: any) {
    ElMessage.error(error?.message || '标记已读失败')
    await loadMessages()
  } finally {
    markingConversationKey.value = null
  }
}

async function handleReadAll() {
  try {
    await markAllInboxMessagesRead()
    const readAt = new Date().toISOString()
    messages.value.forEach(item => {
      item.read_at = item.read_at || readAt
    })
    totalUnreadCount.value = 0
    emit('unread-count-change', 0)
    ElMessage.success('已全部标记为已读')
  } catch (error: any) {
    ElMessage.error(error?.message || '操作失败')
  }
}

watch(onlyUnread, () => {
  void loadMessages()
})

watch(displayConversations, () => {
  syncActiveConversation()
})

watch(() => selectedConversation.value?.key, () => {
  void scrollChatToLatest()
})

watch(() => selectedConversation.value?.messages.map(item => `${item.id}:${item.created_at}:${item.read_at || ''}`).join('|'), () => {
  void scrollChatToLatest()
})

onMounted(() => {
  void preloadMarkdown()
  void loadMessages()
})
</script>

<style scoped lang="scss">
.message-inbox-panel {
  --msg-bg: #06131f;
  --msg-bg-strong: #030a12;
  --msg-panel: rgba(8, 23, 38, 0.86);
  --msg-panel-soft: rgba(13, 42, 62, 0.68);
  --msg-panel-strong: rgba(9, 29, 47, 0.96);
  --msg-line: rgba(54, 244, 255, 0.18);
  --msg-line-strong: rgba(54, 244, 255, 0.58);
  --msg-accent: #36f4ff;
  --msg-accent-rgb: 54, 244, 255;
  --msg-green: #7cffc4;
  --msg-green-rgb: 124, 255, 196;
  --msg-warm: #ffd166;
  --msg-text: #d9fbff;
  --msg-muted: #81a8b8;
  --msg-grid: rgba(54, 244, 255, 0.052);
  position: relative;
  isolation: isolate;
  display: flex;
  min-height: 0;
  height: 100%;
  flex-direction: column;
  overflow: hidden;
  padding: 18px;
  background:
    radial-gradient(circle at 12% 0%, rgba(var(--msg-accent-rgb), 0.22), transparent 34%),
    radial-gradient(circle at 88% 10%, rgba(var(--msg-green-rgb), 0.12), transparent 30%),
    linear-gradient(var(--msg-grid) 1px, transparent 1px),
    linear-gradient(90deg, rgba(var(--msg-accent-rgb), 0.04) 1px, transparent 1px),
    linear-gradient(145deg, rgba(3, 10, 18, 0.98), var(--msg-bg));
  background-size: 100% 100%, 100% 100%, 28px 28px, 28px 28px, 100% 100%;
  color: var(--msg-text);
  box-shadow: inset 1px 0 0 rgba(var(--msg-accent-rgb), 0.1);
}

.message-inbox-panel::before {
  content: '';
  position: absolute;
  inset: 0;
  z-index: -1;
  pointer-events: none;
  background:
    linear-gradient(115deg, transparent 0 44%, rgba(var(--msg-accent-rgb), 0.12) 50%, transparent 56% 100%),
    linear-gradient(180deg, transparent, rgba(var(--msg-green-rgb), 0.04));
  background-size: 260% 100%, 100% 100%;
  animation: messagePanelSweep 11s linear infinite;
  opacity: 0.8;
}

@keyframes messagePanelSweep {
  0% { background-position: 140% 0, 0 0; }
  100% { background-position: -140% 0, 0 0; }
}

.message-inbox-header {
  position: relative;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 16px;
  border: 1px solid var(--msg-line);
  border-radius: 16px;
  background:
    linear-gradient(90deg, rgba(var(--msg-accent-rgb), 0.1), rgba(var(--msg-green-rgb), 0.03)),
    rgba(7, 20, 32, 0.78);
  box-shadow: inset 0 0 22px rgba(var(--msg-accent-rgb), 0.06), 0 16px 38px rgba(0, 0, 0, 0.28);
  backdrop-filter: blur(18px) saturate(1.14);

  &::after {
    content: '';
    position: absolute;
    right: 18px;
    bottom: -1px;
    left: 18px;
    height: 1px;
    background: linear-gradient(90deg, transparent, rgba(var(--msg-accent-rgb), 0.72), transparent);
  }

  h2 {
    margin: 0;
    font-size: 20px;
    line-height: 1.25;
  }

  p {
    margin: 5px 0 0;
    color: var(--msg-muted);
    font-size: 13px;
  }
}

.message-inbox-title-wrap {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 12px;
}

.message-inbox-signal {
  display: inline-flex;
  width: 42px;
  height: 42px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--msg-line-strong);
  border-radius: 14px;
  background:
    radial-gradient(circle at 50% 48%, rgba(var(--msg-accent-rgb), 0.36), transparent 44%),
    conic-gradient(from 130deg, rgba(var(--msg-accent-rgb), 0.18), rgba(var(--msg-green-rgb), 0.32), rgba(var(--msg-accent-rgb), 0.18));
  color: var(--msg-bg-strong);
  box-shadow: 0 0 24px rgba(var(--msg-accent-rgb), 0.48), inset 0 0 12px rgba(var(--msg-accent-rgb), 0.18);
  font-size: 20px;
}

.message-inbox-kicker {
  display: block;
  margin-bottom: 4px;
  color: var(--msg-accent);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0;
  text-transform: uppercase;
}

.message-inbox-header-side {
  display: flex;
  flex: 0 0 auto;
  align-items: flex-start;
  gap: 10px;
}

.message-inbox-metric {
  min-width: 68px;
  padding: 8px 10px;
  border: 1px solid var(--msg-line);
  border-radius: 10px;
  background: rgba(4, 12, 20, 0.7);
  text-align: right;

  span,
  strong {
    display: block;
  }

  span {
    color: var(--msg-muted);
    font-size: 11px;
  }

  strong {
    margin-top: 2px;
    color: var(--msg-accent);
    font-size: 20px;
    line-height: 1;
  }
}

.message-inbox-close {
  border: 1px solid transparent;
  color: var(--msg-muted);

  &:hover {
    border-color: var(--msg-line);
    background: rgba(var(--msg-accent-rgb), 0.08);
    color: var(--msg-accent);
  }
}

.message-inbox-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 0 12px;
}

.message-inbox-toolbar-actions {
  display: flex;
  gap: 8px;
}

.message-filter-tabs {
  display: inline-flex;
  padding: 4px;
  border: 1px solid var(--msg-line);
  border-radius: 12px;
  background: var(--msg-panel);
  box-shadow: inset 0 0 18px rgba(var(--msg-accent-rgb), 0.06);
}

.message-filter-tab {
  min-width: 74px;
  height: 30px;
  padding: 0 12px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--msg-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: background 0.16s ease, color 0.16s ease, box-shadow 0.16s ease;

  &.is-active {
    background: rgba(var(--msg-accent-rgb), 0.13);
    color: var(--msg-accent);
    box-shadow: inset 0 0 0 1px rgba(var(--msg-accent-rgb), 0.18);
  }

  &:hover {
    color: var(--msg-accent);
  }
}

.message-action-btn {
  border-color: var(--msg-line);
  background: var(--msg-panel);
  color: var(--msg-text);
  box-shadow: inset 0 0 14px rgba(var(--msg-accent-rgb), 0.04);

  &:hover {
    border-color: var(--msg-line-strong);
    background: rgba(var(--msg-accent-rgb), 0.09);
    color: var(--msg-accent);
  }
}

.message-action-btn--primary {
  border-color: rgba(var(--msg-accent-rgb), 0.28);
  background: rgba(var(--msg-accent-rgb), 0.12);
  color: var(--msg-accent);

  &:not(.is-disabled):hover {
    border-color: var(--msg-line-strong);
    background: rgba(var(--msg-accent-rgb), 0.18);
  }
}

.message-chat-layout {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  min-height: 0;
  flex: 1;
  overflow: hidden;
  border: 1px solid var(--msg-line);
  border-radius: 18px;
  background: rgba(4, 12, 20, 0.46);
  box-shadow: inset 0 0 24px rgba(var(--msg-accent-rgb), 0.045);
}

.message-chat-layout :deep(.el-loading-mask) {
  background: rgba(3, 10, 18, 0.62);
  backdrop-filter: blur(8px);
}

.message-chat-layout :deep(.el-loading-spinner .path) {
  stroke: var(--msg-accent);
}

.message-chat-layout :deep(.el-loading-text),
.message-chat-layout :deep(.el-empty__description p) {
  color: var(--msg-muted);
}

.conversation-pane {
  min-height: 0;
  overflow: auto;
  border-right: 1px solid rgba(var(--msg-accent-rgb), 0.14);
  background:
    linear-gradient(180deg, rgba(var(--msg-accent-rgb), 0.055), transparent 32%),
    rgba(7, 20, 32, 0.62);
  padding: 12px;
}

.conversation-pane::-webkit-scrollbar,
.chat-thread::-webkit-scrollbar {
  width: 8px;
}

.conversation-pane::-webkit-scrollbar-thumb,
.chat-thread::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background: rgba(var(--msg-accent-rgb), 0.24);
  background-clip: padding-box;
}

.conversation-pane-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 3px 10px;
  color: var(--msg-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;

  strong {
    color: var(--msg-accent);
  }
}

.conversation-item {
  position: relative;
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr) auto;
  gap: 10px;
  width: 100%;
  min-height: 78px;
  padding: 11px 10px;
  border: 1px solid transparent;
  border-radius: 14px;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: background 0.16s ease, border-color 0.16s ease, box-shadow 0.16s ease;

  &:hover,
  &.is-active {
    border-color: rgba(var(--msg-accent-rgb), 0.22);
    background:
      linear-gradient(90deg, rgba(var(--msg-accent-rgb), 0.1), rgba(var(--msg-green-rgb), 0.045)),
      rgba(8, 28, 44, 0.72);
    box-shadow: inset 0 0 18px rgba(var(--msg-accent-rgb), 0.05);
  }

  & + & {
    margin-top: 6px;
  }
}

.conversation-avatar,
.chat-peer-avatar,
.chat-empty-orbit {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(var(--msg-accent-rgb), 0.34);
  background:
    radial-gradient(circle at 50% 48%, rgba(var(--msg-accent-rgb), 0.28), transparent 45%),
    rgba(var(--msg-accent-rgb), 0.08);
  color: var(--msg-accent);
  box-shadow: 0 0 18px rgba(var(--msg-accent-rgb), 0.18), inset 0 0 12px rgba(var(--msg-accent-rgb), 0.08);
  font-weight: 800;
}

.conversation-avatar {
  width: 42px;
  height: 42px;
  border-radius: 14px;
  font-size: 12px;
}

.message-source-icon {
  display: block;
  width: 24px;
  height: 24px;
  object-fit: contain;
}

.message-source-icon--package,
.message-source-icon--service,
.message-source-icon--directory,
.message-source-icon--catalog {
  width: 26px;
  height: 26px;
}

.message-source-icon--table {
  width: 27px;
}

.message-source-icon--board {
  width: 25px;
}

.conversation-main {
  min-width: 0;
}

.message-user-display {
  min-width: 0;
  max-width: 100%;

  :deep(.user-display-wrapper),
  :deep(.user-display-simple) {
    min-width: 0;
    max-width: 100%;
  }

  :deep(.user-display-simple) {
    gap: 6px;
  }

  :deep(.user-name) {
    overflow: hidden;
    color: var(--msg-green);
    font-size: 11px;
    font-weight: 800;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :deep(.user-avatar) {
    border: 1px solid rgba(var(--msg-green-rgb), 0.34);
    background: rgba(var(--msg-green-rgb), 0.11);
    color: var(--msg-green);
    box-shadow: 0 0 12px rgba(var(--msg-green-rgb), 0.18);
  }
}

.conversation-title-row {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 8px;

  time {
    flex: 0 0 auto;
    color: rgba(129, 168, 184, 0.72);
    font-size: 11px;
  }
}

.conversation-title,
.conversation-preview {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conversation-title {
  color: var(--msg-text);
  font-size: 13px;
  font-weight: 800;
}

.conversation-preview {
  margin-top: 6px;
  color: var(--msg-muted);
  font-size: 12px;
}

.conversation-unread {
  min-width: 20px;
  height: 20px;
  align-self: center;
  justify-self: end;
  padding: 0 6px;
  border-radius: 999px;
  background: rgba(255, 209, 102, 0.16);
  color: var(--msg-warm);
  font-size: 11px;
  font-weight: 900;
  line-height: 20px;
  text-align: center;
  box-shadow: 0 0 12px rgba(255, 209, 102, 0.24);
}

.chat-pane,
.chat-empty-pane {
  min-width: 0;
  min-height: 0;
}

.chat-pane {
  display: flex;
  flex-direction: column;
  background:
    linear-gradient(rgba(var(--msg-accent-rgb), 0.035) 1px, transparent 1px),
    linear-gradient(90deg, rgba(var(--msg-accent-rgb), 0.025) 1px, transparent 1px),
    rgba(3, 10, 18, 0.4);
  background-size: 28px 28px, 28px 28px, 100% 100%;
}

.chat-pane-header {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 14px 16px;
  border-bottom: 1px solid rgba(var(--msg-accent-rgb), 0.14);
  background: linear-gradient(90deg, rgba(var(--msg-accent-rgb), 0.08), rgba(7, 20, 32, 0.54));
}

.chat-peer {
  display: flex;
  flex: 1 1 auto;
  min-width: 0;
  align-items: center;
  gap: 12px;
}

.chat-peer-avatar {
  width: 46px;
  height: 46px;
  flex: 0 0 auto;
  border-radius: 16px;
  font-size: 13px;
}

.chat-peer-avatar .message-source-icon {
  width: 28px;
  height: 28px;
}

.chat-peer-avatar .message-source-icon--package,
.chat-peer-avatar .message-source-icon--service,
.chat-peer-avatar .message-source-icon--directory,
.chat-peer-avatar .message-source-icon--catalog {
  width: 30px;
  height: 30px;
}

.chat-peer-avatar .message-source-icon--table {
  width: 31px;
}

.chat-peer-copy {
  flex: 1 1 auto;
  min-width: 0;

  h3,
  p {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  h3 {
    margin: 0;
    color: var(--msg-text);
    font-size: 16px;
    line-height: 1.3;
  }

  p {
    margin: 5px 0 0;
    color: var(--msg-muted);
    font-size: 12px;
  }
}

.chat-peer-meta {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;

  span {
    padding: 4px 8px;
    border: 1px solid rgba(var(--msg-accent-rgb), 0.18);
    border-radius: 999px;
    background: rgba(var(--msg-accent-rgb), 0.065);
    color: var(--msg-muted);
    font-size: 11px;
  }
}

.chat-thread {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding: 18px 18px 24px;
}

.chat-message-row {
  display: flex;
  min-width: 0;
  align-items: flex-start;

  & + & {
    margin-top: 16px;
  }
}

.message-user-display--message {
  flex: 1;

  :deep(.user-name) {
    max-width: 180px;
    color: rgba(var(--msg-green-rgb), 0.88);
  }
}

.chat-bubble {
  position: relative;
  width: min(780px, 100%);
  max-width: 100%;
  padding: 12px 14px 14px;
  border: 1px solid rgba(var(--msg-accent-rgb), 0.18);
  border-radius: 16px;
  background:
    linear-gradient(135deg, rgba(var(--msg-accent-rgb), 0.08), rgba(var(--msg-green-rgb), 0.035)),
    rgba(7, 20, 32, 0.82);
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.22), inset 0 0 18px rgba(var(--msg-accent-rgb), 0.035);
}

.chat-message-row.is-unread .chat-bubble {
  border-color: rgba(255, 209, 102, 0.28);
}

.chat-bubble-head {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid rgba(var(--msg-accent-rgb), 0.11);
}

.chat-bubble-meta {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
  align-items: center;
  color: var(--msg-muted);
  font-size: 11px;

  time {
    color: var(--msg-muted);
  }
}

.chat-message-status {
  padding: 1px 6px;
  border-radius: 999px;
  background: rgba(255, 209, 102, 0.14);
  color: var(--msg-warm) !important;
  font-size: 10px;
}

.chat-message-context {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  gap: 7px;
  margin-top: 10px;
}

.chat-message-context-title,
.chat-message-context-source {
  max-width: 100%;
  padding: 4px 8px;
  overflow: hidden;
  border: 1px solid rgba(var(--msg-accent-rgb), 0.14);
  border-radius: 999px;
  background: rgba(var(--msg-accent-rgb), 0.06);
  color: rgba(var(--msg-accent-rgb), 0.86);
  font-size: 11px;
  font-weight: 800;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-message-context-title {
  color: var(--msg-text);
}

.chat-message-content {
  margin-top: 12px;
  color: var(--msg-text);
  font-size: 14px;
  line-height: 1.75;
  word-break: break-word;

  :deep(h1),
  :deep(h2),
  :deep(h3) {
    margin: 16px 0 9px;
    line-height: 1.35;
  }

  :deep(p) {
    margin: 0 0 10px;
  }

  :deep(p:last-child),
  :deep(ul:last-child),
  :deep(ol:last-child),
  :deep(pre:last-child),
  :deep(blockquote:last-child) {
    margin-bottom: 0;
  }

  :deep(ul),
  :deep(ol) {
    margin: 0 0 10px 20px;
    padding: 0;
  }

  :deep(li + li) {
    margin-top: 4px;
  }

  :deep(a) {
    color: var(--msg-accent);
    text-decoration: none;

    &:hover {
      text-decoration: underline;
    }
  }

  :deep(code) {
    padding: 2px 5px;
    border-radius: 5px;
    background: rgba(var(--msg-accent-rgb), 0.09);
    color: var(--msg-green);
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.92em;
  }

  :deep(pre) {
    max-width: 100%;
    overflow: auto;
    padding: 12px;
    border-radius: 8px;
    background: rgba(var(--msg-accent-rgb), 0.07);

    code {
      padding: 0;
      background: transparent;
      color: inherit;
    }
  }

  :deep(table) {
    display: block;
    max-width: 100%;
    overflow: auto;
    border-collapse: collapse;
  }

  :deep(img) {
    max-width: 100%;
    height: auto;
  }

  :deep(blockquote) {
    margin: 12px 0;
    padding-left: 12px;
    border-left: 3px solid var(--msg-line-strong);
    color: var(--msg-muted);
  }
}

.chat-message-foot {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;

  span {
    max-width: 100%;
    padding: 3px 7px;
    overflow-wrap: anywhere;
    border: 1px solid rgba(var(--msg-accent-rgb), 0.12);
    border-radius: 999px;
    background: rgba(var(--msg-accent-rgb), 0.045);
    color: var(--msg-muted);
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 10px;
  }
}

.chat-empty-pane {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px;
  text-align: center;

  strong {
    margin-top: 12px;
    color: var(--msg-text);
    font-size: 16px;
  }

  p {
    max-width: 320px;
    margin: 8px 0 0;
    color: var(--msg-muted);
    font-size: 13px;
    line-height: 1.6;
  }
}

.chat-empty-orbit {
  width: 56px;
  height: 56px;
  border-radius: 18px;
  font-size: 24px;
}

@media (max-width: 860px) {
  .message-chat-layout {
    grid-template-columns: 1fr;
  }

  .conversation-pane {
    max-height: 260px;
    border-right: 0;
    border-bottom: 1px solid rgba(var(--msg-accent-rgb), 0.14);
  }

  .message-user-display--message {
    :deep(.user-name) {
      max-width: 120px;
    }
  }
}
</style>
