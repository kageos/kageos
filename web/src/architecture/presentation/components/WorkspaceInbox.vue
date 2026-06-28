<template>
  <div class="workspace-inbox">
    <el-tooltip v-if="props.showTrigger" :content="t('workspaceInbox.title')" placement="bottom" effect="light">
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
      @closed="handleDrawerClosed"
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
              <el-tag size="small" effect="plain">{{ t('workspaceInbox.nodeNotifications') }}</el-tag>
              <span :title="sourceFilter.sourcePath">{{ sourceFilter.title || sourceFilter.sourcePath }}</span>
              <el-button size="small" text @click="clearSourceFilter">{{ t('workspaceInbox.viewAll') }}</el-button>
            </div>
          </div>
          <div class="inbox-actions">
            <el-button :icon="Refresh" :loading="listLoading" @click="loadInbox(true)">
              {{ t('common.refresh') }}
            </el-button>
            <el-button :disabled="currentScopeUnreadCount <= 0" @click="markCurrentScopeRead">
              {{ sourceFilter ? t('workspaceInbox.markCurrentNodeRead') : t('workspaceInbox.markAllRead') }}
            </el-button>
          </div>
        </header>

        <div v-if="shouldShowWorkspaceTabs" class="inbox-workspace-tabs">
          <button
            v-for="workspace in workspaceTabs"
            :key="workspace.workspace_key"
            type="button"
            class="workspace-tab"
            :class="{ 'is-active': isWorkspaceTabActive(workspace), 'has-unread': Number(workspace.unread_count || 0) > 0 }"
            @click="handleWorkspaceTabClick(workspace)"
          >
            <span class="workspace-tab-copy">
              <span class="workspace-tab-title">{{ workspaceTabTitle(workspace) }}</span>
              <span class="workspace-tab-path">{{ workspaceTabPath(workspace) }}</span>
            </span>
            <span class="workspace-tab-counts">
              <span v-if="Number(workspace.unread_count || 0) > 0" class="workspace-tab-unread">
                {{ workspace.unread_count }}
              </span>
              <span class="workspace-tab-total">{{ t('workspaceInbox.messageCount', { count: workspace.message_count }) }}</span>
            </span>
          </button>
        </div>

        <el-alert
          v-if="errorMessage"
          :title="errorMessage"
          type="error"
          show-icon
          :closable="false"
          class="inbox-error"
        />

        <div class="inbox-layout">
          <section class="inbox-list-pane" v-loading="!showServiceTreeInbox && listLoading">
            <template v-if="showServiceTreeInbox">
              <el-tree
                :key="sourceTreeRenderKey"
                class="inbox-source-tree"
                :data="props.serviceTree"
                :props="sourceTreeProps"
                node-key="full_code_path"
                :default-expanded-keys="sourceTreeExpandedKeys"
                :expand-on-click-node="false"
                :highlight-current="true"
                :current-node-key="activeSourceTreeKey"
                @node-click="handleSourceTreeNodeClick"
              >
                <template #default="{ data }">
                  <ServiceTreeNodeContent
                    :node="data"
                    :active="isSourceTreeNodeActive(data)"
                    :show-notification-badge="hasSourceTreeMessages(data)"
                    :notification-badge-value="sourceTreeNotificationCount(data)"
                    :notification-badge-class="sourceTreeNotificationClass(data)"
                    :notification-badge-title="getSourceTreeNotificationTitle(data)"
                    @notification-click="handleSourceTreeNodeClick(data)"
                  />
                </template>
              </el-tree>
            </template>
            <template v-else>
              <el-empty
                v-if="!listLoading && inboxThreads.length === 0"
                :description="t('workspaceInbox.empty')"
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
            </template>
          </section>

          <section class="inbox-detail-pane" v-loading="detailLoading || (showServiceTreeInbox && listLoading)">
            <el-empty
              v-if="!selectedThread"
              :description="sourceFilter ? t('workspaceInbox.currentNodeEmpty') : t('workspaceInbox.selectSource')"
              :image-size="96"
            />

            <article v-else class="inbox-detail">
              <header class="inbox-detail-header">
                <div>
                  <h3>{{ selectedThread.title }}</h3>
                  <div class="inbox-detail-meta">
                    <span>{{ selectedThread.subtitle }}</span>
                    <span>{{ t('workspaceInbox.messageCount', { count: selectedThread.count }) }}</span>
                    <el-tag v-if="selectedThread.unreadCount > 0" size="small" type="primary">
                      {{ t('workspaceInbox.unreadMessageCount', { count: selectedThread.unreadCount }) }}
                    </el-tag>
                    <el-tag v-else size="small" type="info">{{ t('workspaceInbox.read') }}</el-tag>
                  </div>
                </div>
                <el-button
                  v-if="selectedThread.unreadCount > 0"
                  size="small"
                  type="primary"
                  plain
                  @click="markThreadRead(selectedThread)"
                >
                  {{ t('workspaceInbox.markAllRead') }}
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
                    {{ t('workspaceInbox.viewSource') }}
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
                      <strong>{{ message.title || t('workspaceInbox.untitledMessage') }}</strong>
                      <el-tag v-if="sourceTypeText(message)" size="small" effect="plain">
                        {{ sourceTypeText(message) }}
                      </el-tag>
                      <el-tag v-if="!message.read_at" size="small" type="primary">{{ t('workspaceInbox.unread') }}</el-tag>
                    </div>
                    <span class="message-card-time" :title="formatExactTime(message.created_at)">
                      <span>{{ formatRelativeTime(message.created_at) }}</span>
                      <small>{{ formatExactTime(message.created_at) }}</small>
                    </span>
                  </header>
                  <div class="message-card-meta">
                    <span>{{ t('workspaceInbox.sender') }}: {{ messageSenderText(message) }}</span>
                    <span>{{ t('workspaceInbox.source') }}: {{ sourceSecondaryText(message) }}</span>
                  </div>
                  <div class="inbox-content inbox-rich-content" v-html="renderMessageContent(message)" />
                  <footer class="message-card-actions">
                    <el-button
                      v-if="message.scheduled_task_id || selectedThread?.scheduledTaskID"
                      size="small"
                      type="primary"
                      plain
                      @click.stop="openScheduledExecution(message)"
                    >
                      {{ t('workspaceInbox.viewExecution') }}
                    </el-button>
                    <el-button
                      v-if="message.workspace_session_id"
                      size="small"
                      type="primary"
                      plain
                      @click.stop="openWorkspaceSession(message)"
                    >
                      {{ t('workspaceInbox.viewSession') }}
                    </el-button>
                    <el-button
                      v-if="sourcePathForMessage(message)"
                      size="small"
                      plain
                      @click.stop="openSourcePath(message)"
                    >
                      {{ t('workspaceInbox.viewSource') }}
                    </el-button>
                    <el-button
                      v-if="!message.read_at"
                      size="small"
                      plain
                      @click.stop="markMessageRead(message.id)"
                    >
                      {{ t('workspaceInbox.markRead') }}
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
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
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
import ServiceTreeNodeContent from './ServiceTreeNodeContent.vue'
import {
  buildInboxRouteQuery,
  buildScheduledExecutionRoute,
  buildWorkspaceSessionRoute,
  clearInboxRouteQuery,
  clearOperateLogRouteQuery,
  clearScheduledRouteQuery,
  isInboxOpenQuery,
  normalizeWorkspaceFullCodePath,
  readNumberQuery,
  readStringQuery,
  workspaceRoutePath,
  PLATFORM_MESSAGE_ID_QUERY_KEY,
  PLATFORM_SOURCE_PATH_QUERY_KEY,
  PLATFORM_TRACE_ID_QUERY_KEY,
} from '@/architecture/shared/routing/platformRouteParams'
import {
  getMessageInboxItem,
  getMessageInboxUnreadCount,
  listMessageInbox,
  listMessageInboxSourceCounts,
  listMessageInboxThreads,
  listMessageInboxWorkspaceCounts,
  markAllMessageInboxItemsRead,
  markMessageInboxItemRead,
  type MessageInboxItem,
  type MessageInboxSourceCount,
  type MessageInboxThread,
  type MessageInboxWorkspaceCount,
  type MessageInboxStatus,
} from '@/architecture/presentation/context/api/message'
import { getAppList } from '@/architecture/presentation/context/api/app'
import type { App, ServiceTree } from '@/architecture/domain/types'

const props = withDefaults(defineProps<{
  showTrigger?: boolean
  syncRoute?: boolean
  serviceTree?: ServiceTree[]
  currentApp?: App | null
  appList?: App[]
}>(), {
  showTrigger: true,
  syncRoute: true,
  serviceTree: () => [],
  currentApp: null,
  appList: () => []
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

interface SourceTreeSummary {
  unread_count?: number
  message_count?: number
  latest_at?: string
}

interface LoadInboxOptions {
  markSourceRead?: boolean
}

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
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
const statusOptions = computed(() => [
  { label: t('workspaceInbox.all'), value: 'all' },
  { label: t('workspaceInbox.unread'), value: 'unread' },
])
const sourceFilter = ref<SourceFilter | null>(null)
const markSourceReadOnOpen = ref(false)
const sourceCountMap = ref<Record<string, MessageInboxSourceCount>>({})
const workspaceCounts = ref<MessageInboxWorkspaceCount[]>([])
const resolvedWorkspaceApps = ref<Record<string, App>>({})
const appliedRouteInboxKey = ref('')
let inboxLoadSeq = 0
let detailLoadSeq = 0
let routeIntentOpening = false
let workspaceListHydratePromise: Promise<void> | null = null
let workspaceListHydrated = false
const sourceTreeProps = {
  children: 'children',
  label: 'name',
}
const showServiceTreeInbox = computed(() => props.showTrigger && props.serviceTree.length > 0)
const activeSourceTreeKey = computed(() => normalizeSourceTreePath(sourceFilter.value?.sourcePath))
const currentWorkspaceKey = computed(() => {
  return workspaceKeyFromRoutePath(route.path) || workspaceKeyFromApp(props.currentApp)
})
const workspaceAppLookup = computed<Record<string, App>>(() => {
  const lookup: Record<string, App> = { ...resolvedWorkspaceApps.value }
  const addApp = (app?: App | null) => {
    const key = workspaceKeyFromApp(app)
    if (!key || !app) return
    lookup[key] = {
      ...lookup[key],
      ...app,
      name: app.name?.trim() || lookup[key]?.name || app.code,
    }
  }

  for (const app of props.appList || []) {
    addApp(app)
  }
  addApp(props.currentApp)
  return lookup
})
const workspaceTabs = computed(() => {
  return workspaceCounts.value
    .filter(item => workspaceKeyForCount(item))
    .sort((a, b) => messageTimeFromString(b.latest_at) - messageTimeFromString(a.latest_at))
})
const shouldShowWorkspaceTabs = computed(() => {
  const tabs = workspaceTabs.value
  if (tabs.length > 1) return true
  const onlyTab = tabs[0]
  return Boolean(onlyTab && workspaceKeyForCount(onlyTab) !== currentWorkspaceKey.value)
})
const sourceTreeSummaries = computed<Record<string, SourceTreeSummary>>(() => {
  const summaries: Record<string, SourceTreeSummary> = {}
  const walk = (node: ServiceTree) => {
    const path = normalizeSourceTreePath(node.full_code_path)
    if (path) {
      summaries[path] = sourceCountMap.value[path] || {}
    }
    for (const child of node.children || []) {
      walk(child)
    }
  }
  for (const node of props.serviceTree || []) {
    walk(node)
  }
  return summaries
})
const sourceTreeExpandedKeys = computed(() => {
  const expanded = new Set<string>()

  const addChain = (chain: string[]) => {
    for (const path of chain) {
      if (path) expanded.add(path)
    }
  }

  const selectedPath = normalizeSourceTreePath(sourceFilter.value?.sourcePath)
  const walk = (node: ServiceTree, ancestors: string[]) => {
    const path = normalizeSourceTreePath(node.full_code_path)
    const chain = path ? [...ancestors, path] : ancestors
    const summary = sourceTreeSummaryByPath(path)
    const hasMessages = Number(summary?.message_count || 0) > 0 || Number(summary?.unread_count || 0) > 0
    const isRoot = ancestors.length === 0 && path
    const isSelectedChain = Boolean(selectedPath && path && (selectedPath === path || selectedPath.startsWith(`${path}/`)))

    if (isRoot || hasMessages || isSelectedChain) {
      addChain(chain)
    }

    for (const child of node.children || []) {
      walk(child, chain)
    }
  }

  for (const node of props.serviceTree || []) {
    walk(node, [])
  }

  return [...expanded]
})
const sourceTreeRenderKey = computed(() => {
  return sourceTreeExpandedKeys.value.join('|') || 'empty'
})
const drawerTitle = computed(() => {
  if (!sourceFilter.value) return t('workspaceInbox.title')
  return t('workspaceInbox.sourceNotificationTitle', {
    title: sourceFilter.value.title || t('workspaceInbox.nodeNotifications')
  })
})
const selectedThread = computed(() => {
  return inboxThreads.value.find(thread => thread.key === selectedThreadKey.value)
    || inboxThreads.value[0]
    || null
})
const currentScopeUnreadCount = computed(() => {
  if (sourceFilter.value) {
    return selectedThread.value ? selectedThread.value.unreadCount : sourceFilterUnreadCount()
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

watch(
  () => [
    route.query._open,
    route.query[PLATFORM_MESSAGE_ID_QUERY_KEY],
    route.query[PLATFORM_SOURCE_PATH_QUERY_KEY],
    route.query[PLATFORM_TRACE_ID_QUERY_KEY],
    props.showTrigger,
  ],
  () => {
    void openInboxFromRouteIntent()
  },
  { immediate: true }
)

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
  void preloadMarkdown()
  sourceFilter.value = null
  markSourceReadOnOpen.value = false
  void syncInboxRoute()
  drawerVisible.value = true
}

async function openInboxFromRouteIntent() {
  if (!props.syncRoute || !isInboxOpenQuery(route.query)) return
  void preloadMarkdown()
  const messageID = readNumberQuery(route.query, PLATFORM_MESSAGE_ID_QUERY_KEY)
  const sourcePath = normalizeSourceTreePath(readStringQuery(route.query, PLATFORM_SOURCE_PATH_QUERY_KEY))
  const key = `${currentWorkspaceKey.value}:${sourcePath}:${messageID}`
  if (appliedRouteInboxKey.value === key && drawerVisible.value) return
  appliedRouteInboxKey.value = key

  if (sourcePath) {
    const sourceNode = findServiceTreeNodeByPath(sourcePath)
    sourceFilter.value = {
      sourcePath,
      title: sourceNode?.name || sourceNode?.code || sourcePath,
      includeChildren: false,
      kind: sourceNode?.type === 'package' ? 'directory' : 'function',
    }
  } else {
    sourceFilter.value = null
  }

  markSourceReadOnOpen.value = false
  routeIntentOpening = true
  drawerVisible.value = true
  try {
    await loadInbox(true)
    if (messageID) {
      await focusMessageByID(messageID)
    }
  } finally {
    routeIntentOpening = false
  }
}

function openForSource(filter: SourceFilter) {
  const sourcePath = (filter.sourcePath || '').trim()
  if (!sourcePath) return
  void preloadMarkdown()
  sourceFilter.value = {
    ...filter,
    sourcePath,
  }
  const wasVisible = drawerVisible.value
  markSourceReadOnOpen.value = !wasVisible
  void syncInboxRoute({ sourcePath })
  drawerVisible.value = true
  if (wasVisible) {
    void loadInbox(true, { markSourceRead: true })
  }
}

function clearSourceFilter() {
  sourceFilter.value = null
  markSourceReadOnOpen.value = false
  void syncInboxRoute()
  void loadInbox(true)
}

function handleDrawerOpen() {
  void preloadMarkdown()
  if (routeIntentOpening) {
    void loadUnreadCount()
    return
  }
  const markSourceRead = markSourceReadOnOpen.value
  markSourceReadOnOpen.value = false
  void loadInbox(true, { markSourceRead })
  void loadUnreadCount()
}

function handleDrawerClosed() {
  if (!props.syncRoute || !isInboxOpenQuery(route.query)) return
  appliedRouteInboxKey.value = ''
  const query = { ...route.query }
  clearInboxRouteQuery(query)
  void router.replace({ path: route.path, query })
}

async function loadInbox(resetPage = false, options: LoadInboxOptions = {}) {
  const loadSeq = ++inboxLoadSeq
  detailLoadSeq += 1
  detailLoading.value = false
  if (resetPage) {
    page.value = 1
  }
  listLoading.value = true
  errorMessage.value = ''
  try {
    await loadWorkspaceCounts()
    if (loadSeq !== inboxLoadSeq) return
    if (showServiceTreeInbox.value) {
      await loadSourceCounts()
      if (loadSeq !== inboxLoadSeq) return
    }
    if (sourceFilter.value?.sourcePath) {
      await loadSourceInbox({ markRead: options.markSourceRead, loadSeq })
      return
    }
    if (showServiceTreeInbox.value) {
      if (loadSeq !== inboxLoadSeq) return
      inboxThreads.value = []
      threadMessages.value = []
      selectedThreadKey.value = ''
      selectedMessage.value = null
      total.value = 0
      return
    }
    const resp = await listMessageInboxThreads({
      status: statusFilter.value === 'unread' ? 'unread' : undefined,
      page: page.value,
      page_size: pageSize,
    })
    if (loadSeq !== inboxLoadSeq) return
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
    if (loadSeq === inboxLoadSeq) {
      errorMessage.value = error instanceof Error ? error.message : t('workspaceInbox.loadFailed')
    }
  } finally {
    if (loadSeq === inboxLoadSeq) {
      listLoading.value = false
    }
  }
}

async function loadWorkspaceCounts() {
  try {
    const resp = await listMessageInboxWorkspaceCounts()
    const counts = (resp.list || [])
      .map(item => {
        const workspaceKey = workspaceKeyForCount(item)
        return {
          ...item,
          workspace_key: workspaceKey,
          workspace_path: workspaceKey,
        }
      })
      .filter(item => item.workspace_key)
    workspaceCounts.value = counts
    await hydrateWorkspaceAppsForCounts(counts)
  } catch {
    workspaceCounts.value = []
  }
}

async function loadSourceCounts() {
  try {
    const resp = await listMessageInboxSourceCounts()
    const next: Record<string, MessageInboxSourceCount> = {}
    for (const item of resp.list || []) {
      const path = normalizeSourceTreePath(item.source_path)
      if (!path) continue
      next[path] = {
        ...item,
        source_path: path,
      }
    }
    sourceCountMap.value = next
  } catch {
    sourceCountMap.value = {}
  }
}

async function loadSourceInbox(options: { markRead?: boolean; loadSeq?: number } = {}) {
  const filter = sourceFilter.value
  if (!filter?.sourcePath) return
  const resp = await listMessageInbox({
    status: statusFilter.value === 'unread' ? 'unread' : undefined,
    source_path: filter.sourcePath,
    include_children: false,
    page: page.value,
    page_size: 100,
  })
  if (options.loadSeq && options.loadSeq !== inboxLoadSeq) return
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
  const unreadCount = sourceFilterUnreadCount() || messages.filter(item => !item.read_at).length
  const thread: InboxThread = {
    key: sourceFilterThreadKey(filter),
    title: filter.title || sourcePrimaryText(firstMessage),
    subtitle: t('workspaceInbox.currentNodeNotifications'),
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
  if (options.markRead && unreadCount > 0) {
    await markThreadRead(thread)
  }
}

function selectThread(thread: InboxThread) {
  selectedThreadKey.value = thread.key
  selectedMessage.value = thread.lastMessage
  void loadThreadMessages(thread).then((loaded) => {
    if (loaded) {
      void markThreadRead(thread)
    }
  })
}

async function loadThreadMessages(thread: InboxThread): Promise<boolean> {
  const loadSeq = ++detailLoadSeq
  detailLoading.value = true
  errorMessage.value = ''
  try {
    if (sourceFilter.value?.sourcePath) {
      await loadSourceInbox()
      return loadSeq === detailLoadSeq
    }
    const resp = await listMessageInbox({
      thread_key: thread.key,
      page: 1,
      page_size: 100,
    })
    if (loadSeq !== detailLoadSeq) return false
    threadMessages.value = resp.list || []
    return true
  } catch (error) {
    if (loadSeq === detailLoadSeq) {
      threadMessages.value = [thread.lastMessage]
      errorMessage.value = error instanceof Error ? error.message : t('workspaceInbox.loadThreadFailed')
    }
    return false
  } finally {
    if (loadSeq === detailLoadSeq) {
      detailLoading.value = false
    }
  }
}

async function selectMessage(item: MessageInboxItem) {
  const loadSeq = ++detailLoadSeq
  selectedMessage.value = item
  detailLoading.value = true
  errorMessage.value = ''
  try {
    const detail = await getMessageInboxItem(item.id)
    if (loadSeq !== detailLoadSeq) return
    selectedMessage.value = detail
    void syncInboxRoute({
      messageId: detail.id,
      sourcePath: sourcePathForMessage(detail) || sourceFilter.value?.sourcePath,
      traceId: detail.trace_id,
    })
    if (!detail.read_at) {
      await markMessageInboxItemRead(item.id)
      selectedMessage.value = { ...detail, read_at: new Date().toISOString() }
      updateListReadState(item.id)
      await loadUnreadCount()
      await loadWorkspaceCounts()
      if (showServiceTreeInbox.value) {
        await loadSourceCounts()
      }
      emit('messages-updated')
    }
  } catch (error) {
    if (loadSeq === detailLoadSeq) {
      errorMessage.value = error instanceof Error ? error.message : t('workspaceInbox.loadDetailFailed')
    }
  } finally {
    if (loadSeq === detailLoadSeq) {
      detailLoading.value = false
    }
  }
}

async function focusMessageByID(id: number) {
  const existing = threadMessages.value.find(item => item.id === id)
  if (existing) {
    await selectMessage(existing)
    return
  }

  const loadSeq = ++detailLoadSeq
  detailLoading.value = true
  errorMessage.value = ''
  try {
    const detail = await getMessageInboxItem(id)
    if (loadSeq !== detailLoadSeq) return
    selectedMessage.value = detail
    upsertFocusedThread(detail)
    if (!threadMessages.value.some(item => item.id === detail.id)) {
      threadMessages.value = [detail, ...threadMessages.value]
    }
    if (!detail.read_at) {
      await markMessageInboxItemRead(detail.id)
      const readDetail = { ...detail, read_at: new Date().toISOString() }
      selectedMessage.value = readDetail
      threadMessages.value = threadMessages.value.map(item => item.id === detail.id ? readDetail : item)
      updateListReadState(detail.id)
      await loadUnreadCount()
      await loadWorkspaceCounts()
      if (showServiceTreeInbox.value) {
        await loadSourceCounts()
      }
      emit('messages-updated')
    }
  } catch (error) {
    if (loadSeq === detailLoadSeq) {
      errorMessage.value = error instanceof Error ? error.message : t('workspaceInbox.loadDetailFailed')
    }
  } finally {
    if (loadSeq === detailLoadSeq) {
      detailLoading.value = false
    }
  }
}

function upsertFocusedThread(detail: MessageInboxItem) {
  const filter = sourceFilter.value
  const key = filter?.sourcePath ? sourceFilterThreadKey(filter) : threadKeyForMessage(detail)
  if (!key) return
  const thread: InboxThread = {
    key,
    title: filter?.title || sourcePrimaryText(detail),
    subtitle: filter?.sourcePath ? t('workspaceInbox.currentNodeNotifications') : threadSubtitle(detail, 1),
    path: filter?.sourcePath || threadPath(detail),
    kind: filter?.kind || threadKind(detail),
    lastMessage: detail,
    unreadCount: detail.read_at ? 0 : 1,
    count: Math.max(1, Number(total.value || 0)),
    scheduledTaskID: detail.scheduled_task_id,
    scheduledExecutionID: detail.scheduled_execution_id,
  }
  const existingIndex = inboxThreads.value.findIndex(item => item.key === key)
  if (existingIndex >= 0) {
    const existing = inboxThreads.value[existingIndex]
    if (!existing) return
    inboxThreads.value.splice(existingIndex, 1, {
      ...existing,
      lastMessage: existing.lastMessage?.id === detail.id ? detail : existing.lastMessage,
      scheduledTaskID: existing.scheduledTaskID || detail.scheduled_task_id,
      scheduledExecutionID: existing.scheduledExecutionID || detail.scheduled_execution_id,
    })
  } else {
    inboxThreads.value = [thread, ...inboxThreads.value]
  }
  selectedThreadKey.value = key
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
    await loadWorkspaceCounts()
    if (showServiceTreeInbox.value) {
      await loadSourceCounts()
    }
    emit('messages-updated')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('workspaceInbox.markReadFailed'))
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
    await loadWorkspaceCounts()
    if (showServiceTreeInbox.value) {
      await loadSourceCounts()
    }
    emit('messages-updated')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('workspaceInbox.markReadFailed'))
  }
}

async function markCurrentScopeRead() {
  if (sourceFilter.value) {
    if (selectedThread.value) {
      await markThreadRead(selectedThread.value)
    }
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
    await loadWorkspaceCounts()
    if (showServiceTreeInbox.value) {
      await loadSourceCounts()
    }
    ElMessage.success(t('workspaceInbox.allReadSuccess'))
    emit('messages-updated')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('workspaceInbox.allReadFailed'))
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
  return text.length > 90 ? `${text.slice(0, 90)}...` : text || t('workspaceInbox.noContent')
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
  return `source:${filter.sourcePath}:direct`
}

function normalizeSourceTreePath(path?: string) {
  return normalizeWorkspaceFullCodePath(path)
}

function findServiceTreeNodeByPath(fullCodePath: string): ServiceTree | null {
  const target = normalizeSourceTreePath(fullCodePath)
  if (!target) return null
  const walk = (nodes: ServiceTree[]): ServiceTree | null => {
    for (const node of nodes) {
      if (normalizeSourceTreePath(node.full_code_path) === target) return node
      const child = walk(node.children || [])
      if (child) return child
    }
    return null
  }
  return walk(props.serviceTree || [])
}

function sourceTreeSummaryByPath(path?: string) {
  const normalized = normalizeSourceTreePath(path)
  if (!normalized) return undefined
  return sourceTreeSummaries.value[normalized] || sourceCountMap.value[normalized]
}

function getSourceTreeSummary(node: ServiceTree) {
  return sourceTreeSummaryByPath(node.full_code_path)
}

function sourceFilterUnreadCount() {
  const filter = sourceFilter.value
  if (!filter?.sourcePath) return 0
  return Number(sourceTreeSummaryByPath(filter.sourcePath)?.unread_count || 0)
}

function hasSourceTreeMessages(node: ServiceTree) {
  const summary = getSourceTreeSummary(node)
  return Number(summary?.message_count || 0) > 0 || Number(summary?.unread_count || 0) > 0
}

function sourceTreeNotificationCount(node: ServiceTree) {
  const summary = getSourceTreeSummary(node)
  const unread = Number(summary?.unread_count || 0)
  if (unread > 0) return unread
  return Number(summary?.message_count || 0) || ''
}

function sourceTreeNotificationClass(node: ServiceTree) {
  const unread = Number(getSourceTreeSummary(node)?.unread_count || 0)
  return unread > 0 ? 'is-unread' : 'is-history'
}

function getSourceTreeNotificationTitle(node: ServiceTree) {
  const summary = getSourceTreeSummary(node)
  const unread = Number(summary?.unread_count || 0)
  const total = Number(summary?.message_count || 0)
  if (unread > 0) return t('workspaceInbox.notificationTitleUnread', { unread, total })
  return t('workspaceInbox.notificationTitleTotal', { total })
}

function isSourceTreeNodeActive(node: ServiceTree) {
  return normalizeSourceTreePath(node.full_code_path) === normalizeSourceTreePath(sourceFilter.value?.sourcePath)
}

function handleSourceTreeNodeClick(node: ServiceTree) {
  const sourcePath = normalizeSourceTreePath(node.full_code_path)
  if (!sourcePath) return
  const previousSourcePath = normalizeSourceTreePath(sourceFilter.value?.sourcePath)
  sourceFilter.value = {
    sourcePath,
    title: node.name || node.code || sourcePath,
    includeChildren: false,
    kind: node.type === 'package' ? 'directory' : 'function',
  }
  if (previousSourcePath !== sourcePath) {
    inboxThreads.value = []
    threadMessages.value = []
    selectedThreadKey.value = ''
    selectedMessage.value = null
    total.value = 0
  }
  void syncInboxRoute({ sourcePath })
  void loadInbox(true, { markSourceRead: true })
}

function messageTime(item: MessageInboxItem) {
  const parsed = dayjs(item.created_at)
  return parsed.isValid() ? parsed.valueOf() : 0
}

function messageTimeFromString(value?: string) {
  const parsed = dayjs(value)
  return parsed.isValid() ? parsed.valueOf() : 0
}

function cacheWorkspaceApps(apps: App[]) {
  if (!apps.length) return
  const next: Record<string, App> = { ...resolvedWorkspaceApps.value }
  for (const app of apps) {
    const key = workspaceKeyFromApp(app)
    if (!key) continue
    next[key] = {
      ...next[key],
      ...app,
      name: app.name?.trim() || next[key]?.name || app.code,
    }
  }
  resolvedWorkspaceApps.value = next
}

async function hydrateWorkspaceAppsForCounts(items: MessageInboxWorkspaceCount[]) {
  const hasMissingWorkspaceName = items.some(item => {
    const key = workspaceKeyForCount(item)
    return key && !workspaceAppLookup.value[key]
  })
  if (!hasMissingWorkspaceName || workspaceListHydrated) return

  if (!workspaceListHydratePromise) {
    workspaceListHydratePromise = (async () => {
      const [allApps, systemApps] = await Promise.all([
        getAppList(500, undefined, true),
        getAppList(500, undefined, false, 1),
      ])
      cacheWorkspaceApps([...allApps, ...systemApps])
      workspaceListHydrated = true
    })()
  }

  try {
    await workspaceListHydratePromise
  } catch {
    // 非关键路径：接口失败时仍按 message-server 返回的 title/path 展示。
  } finally {
    workspaceListHydratePromise = null
  }
}

function workspaceKeyFromApp(app?: App | null) {
  if (!app?.user || !app?.code) return ''
  return `/${app.user}/${app.code}`
}

function workspaceKeyFromRoutePath(path: string) {
  const normalized = path.replace(/^\/workspace\/?/, '').split('?')[0] || ''
  const parts = normalized.split('/').filter(Boolean)
  if (parts.length < 2) return ''
  return `/${parts[0]}/${parts[1]}`
}

function workspaceKeyForCount(item: MessageInboxWorkspaceCount) {
  const raw = normalizeSourceTreePath(item.workspace_path || item.workspace_key)
  if (!raw || raw === 'global') return ''
  const key = raw.startsWith('/') ? raw : `/${raw}`
  const parts = key.split('/').filter(Boolean)
  return parts.length >= 2 ? `/${parts[0]}/${parts[1]}` : ''
}

function workspaceAppForCount(item: MessageInboxWorkspaceCount) {
  return workspaceAppLookup.value[workspaceKeyForCount(item)] || null
}

function workspaceTabTitle(item: MessageInboxWorkspaceCount) {
  const app = workspaceAppForCount(item)
  return app?.name?.trim() || item.title || workspaceTabPath(item) || t('workspaceInbox.globalMessages')
}

function workspaceTabPath(item: MessageInboxWorkspaceCount) {
  const key = workspaceKeyForCount(item)
  if (!key) return item.title || ''
  return key.replace(/^\//, '')
}

function isWorkspaceTabActive(item: MessageInboxWorkspaceCount) {
  return workspaceKeyForCount(item) === currentWorkspaceKey.value
}

async function handleWorkspaceTabClick(item: MessageInboxWorkspaceCount) {
  const workspacePath = workspaceKeyForCount(item)
  if (!workspacePath) return
  sourceFilter.value = null
  markSourceReadOnOpen.value = false
  if (workspacePath === currentWorkspaceKey.value) {
    void syncInboxRoute()
    void loadInbox(true)
    return
  }
  inboxThreads.value = []
  threadMessages.value = []
  selectedThreadKey.value = ''
  selectedMessage.value = null
  total.value = 0
  await router.push({
    path: workspaceRoutePath(workspacePath),
    query: buildInboxRouteQuery(),
  })
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
  const suffix = count > 1 ? ` · ${t('workspaceInbox.messageCount', { count })}` : ''
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

function messageSenderText(item?: MessageInboxItem | null) {
  const sender = (item?.from || item?.request_user || '').trim()
  if (!sender) return 'system'
  if (sender === 'system') return t('workspaceInbox.systemSender')
  return sender
}

function sourceTypeText(item?: MessageInboxItem | null) {
  const type = (item?.source_type || item?.client_source || '').trim()
  const map: Record<string, string> = {
    scheduled_task: t('workspaceInbox.sourceTypeScheduledTask'),
    agent_session: t('workspaceInbox.sourceTypeAgentSession'),
    agent_tool: t('workspaceInbox.sourceTypeAgentTool'),
    public_share: t('workspaceInbox.sourceTypePublicShare'),
    openapi_token: 'OpenAPI',
    sdk_function: t('workspaceInbox.sourceTypeSdkFunction'),
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

async function syncInboxRoute(options: {
  messageId?: number | string
  sourcePath?: string
  traceId?: string
} = {}) {
  if (!props.syncRoute) return
  const sourcePath = normalizeSourceTreePath(options.sourcePath)
  const messageID = options.messageId ? String(options.messageId) : ''
  appliedRouteInboxKey.value = `${currentWorkspaceKey.value}:${sourcePath}:${messageID ? Number(messageID) || messageID : 0}`
  const query = { ...route.query }
  clearScheduledRouteQuery(query)
  clearOperateLogRouteQuery(query)
  clearInboxRouteQuery(query)
  Object.assign(query, buildInboxRouteQuery({
    messageId: options.messageId,
    sourcePath,
    traceId: options.traceId,
  }))
  await router.replace({ path: route.path, query })
}

async function openSourcePath(item: MessageInboxItem) {
  const sourcePath = sourcePathForMessage(item)
  const targetPath = workspaceRoutePath(sourcePath)
  if (!targetPath) return
  const query = { ...route.query }
  clearScheduledRouteQuery(query)
  clearOperateLogRouteQuery(query)
  clearInboxRouteQuery(query)
  drawerVisible.value = false
  await router.push({ path: targetPath, query })
}

async function openWorkspaceSession(item: MessageInboxItem) {
  const sessionId = (item.workspace_session_id || '').trim()
  const fullCodePath = workspacePathForMessage(item)
  if (!sessionId || !workspaceRoutePath(fullCodePath)) return
  const target = buildWorkspaceSessionRoute({
    fullCodePath,
    sessionId,
    sourceName: sourcePrimaryText(item),
    sourcePath: sourcePathForMessage(item),
    traceId: item.trace_id,
  })
  const opened = window.open(router.resolve(target).href, '_blank')
  if (opened) {
    opened.opener = null
    return
  }
  await router.push(target)
}

async function openScheduledExecution(item: MessageInboxItem) {
  const taskID = item.scheduled_task_id || selectedThread.value?.scheduledTaskID || 0
  if (!taskID) return
  const executionID = item.scheduled_execution_id || selectedThread.value?.scheduledExecutionID || 0
  const fullCodePath = workspacePathForMessage(item)
  if (!workspaceRoutePath(fullCodePath)) return
  drawerVisible.value = false
  await router.push(buildScheduledExecutionRoute({
    fullCodePath,
    kind: item.workspace_session_id || item.source_type === 'agent_session' ? 'agent' : 'function',
    taskId: taskID,
    executionId: executionID || undefined,
    sourcePath: sourcePathForMessage(item),
    traceId: item.trace_id,
  }))
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

  if (absDiffMs < minute) return t('workspaceInbox.justNow')
  if (diffMs < 0) {
    if (absDiffMs < hour) return t('workspaceInbox.inMinutes', { count: Math.floor(absDiffMs / minute) })
    if (absDiffMs < day) return t('workspaceInbox.inHours', { count: Math.floor(absDiffMs / hour) })
    return parsed.format('MM-DD HH:mm')
  }
  if (diffMs < hour) return t('workspaceInbox.minutesAgo', { count: Math.floor(diffMs / minute) })
  if (diffMs < day) return t('workspaceInbox.hoursAgo', { count: Math.floor(diffMs / hour) })
  if (diffMs < 30 * day) return t('workspaceInbox.daysAgo', { count: Math.floor(diffMs / day) })
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

.inbox-workspace-tabs {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding: 2px 0 4px;
}

.workspace-tab {
  display: inline-flex;
  min-width: 180px;
  max-width: 260px;
  flex-shrink: 0;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 10px;
  border: 1px solid transparent;
  border-radius: var(--border-radius-base);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  cursor: pointer;
  text-align: left;
  transition: all 0.25s cubic-bezier(0.25, 0.8, 0.25, 1);

  &:hover {
    background: var(--el-fill-color);
  }

  &.is-active {
    background: var(--el-fill-color-blank);
    border-color: var(--color-primary);
    box-shadow: 0 0 0 2px rgba(var(--color-primary-rgb), 0.1);
  }
}

.workspace-tab-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.workspace-tab-title,
.workspace-tab-path {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-tab-title {
  font-size: 13px;
  font-weight: 700;
}

.workspace-tab-path {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.workspace-tab-counts {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  gap: 6px;
}

.workspace-tab-unread {
  display: inline-flex;
  min-width: 20px;
  height: 20px;
  align-items: center;
  justify-content: center;
  padding: 0 6px;
  border-radius: 999px;
  background: #ef4444;
  color: #fff;
  font-size: 11px;
  font-weight: 800;
}

.workspace-tab-total {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  white-space: nowrap;
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
  border: 1px solid transparent;
  border-radius: var(--border-radius-lg);
  background: var(--bg-secondary);
}

.inbox-list-pane {
  padding: 8px;
}

.inbox-source-tree {
  background: transparent;

  :deep(.el-tree-node__content) {
    height: 40px;
    margin: 2px 0;
    border-radius: 10px;
    transition: background 0.16s ease, color 0.16s ease;
  }

  :deep(.el-tree-node__content:hover) {
    background: rgba(var(--el-color-primary-rgb), 0.07);
  }

  :deep(.el-tree-node__expand-icon) {
    color: var(--el-text-color-placeholder);
  }

  :deep(.el-tree-node.is-current > .el-tree-node__content) {
    border-color: transparent;
    background: var(--el-fill-color);
    box-shadow: none;
  }

  :deep(.tree-node.is-active .node-label) {
    color: var(--el-color-primary);
    font-weight: 800;
  }

  :deep(.tree-node.is-active .node-icon) {
    opacity: 1;
  }
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
    border-color: transparent;
    background: var(--el-fill-color);
  }

  &.is-unread .inbox-list-title {
    font-weight: 700;
  }
}

.thread-avatar {
  display: grid;
  width: 42px;
  height: 42px;
  margin-top: 1px;
  place-items: center;
  border: 1px solid transparent;
  border-radius: var(--border-radius-base);
  background: var(--bg-tertiary);
  color: var(--color-primary);
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
  padding: 18px 24px;
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
  border-bottom: 1px solid var(--border-light);

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
  padding: 14px 16px;
  border: 1px solid var(--color-primary-light-8);
  border-radius: var(--border-radius-lg);
  background: var(--color-primary-light-9);
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
  padding: 16px;
  border: 1px solid transparent;
  border-radius: var(--border-radius-lg);
  background: var(--bg-primary);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  cursor: pointer;
  transition: all 0.25s cubic-bezier(0.25, 0.8, 0.25, 1);

  &:hover,
  &.is-active {
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
    transform: translateY(-1px);
  }

  &.is-unread {
    border-color: var(--color-primary-light-6);
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
  flex: 1;
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

.message-card-time {
  display: flex;
  flex-shrink: 0;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  color: var(--el-text-color-secondary);
  line-height: 1.2;

  small {
    color: var(--el-text-color-placeholder);
    font-size: 11px;
    font-weight: 500;
  }
}

.message-card-meta {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  gap: 6px 12px;
  color: var(--el-text-color-secondary);
  font-size: 12px;

  span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
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
