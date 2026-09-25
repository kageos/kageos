<template>
  <div class="workspace-inbox">
    <el-tooltip v-if="props.showTrigger" :content="t('workspaceInbox.title')" placement="bottom" effect="light">
      <el-badge :value="unreadCount" :hidden="unreadCount <= 0" :max="99" class="workspace-inbox-badge">
        <el-button class="workspace-inbox-button" :icon="MessageIcon" :loading="countLoading" :aria-label="t('workspaceInbox.title')" circle @click="openDrawer" />
      </el-badge>
    </el-tooltip>
    <el-drawer v-model="drawerVisible" :title="drawerTitle" direction="rtl"
      :size="maximized ? '100vw' : 'min(1480px, 96vw)'" :destroy-on-close="false" append-to-body
      modal-class="workspace-inbox-modal" :z-index="Z_INDEX.globalOverlay" class="workspace-inbox-drawer"
      @open="handleDrawerOpen" @closed="handleDrawerClosed">
      <template #header>
        <div class="inbox-heading">
          <h2>{{ t('workspaceInbox.title') }}</h2>
          <el-button text class="inbox-maximize" @click="maximized = !maximized">
            {{ t(maximized ? 'workspaceInbox.restore' : 'workspaceInbox.maximize') }}
          </el-button>
        </div>
      </template>
      <div class="inbox-shell">
        <form class="inbox-search" @submit.prevent="applySearch">
          <el-input v-model="searchInput" :prefix-icon="Search" :placeholder="t('workspaceInbox.searchPlaceholder')"
            :aria-label="t('workspaceInbox.searchPlaceholder')" maxlength="200" clearable @input="scheduleSearch" />
        </form>
        <header class="inbox-toolbar">
          <div class="inbox-filter">
            <el-button v-if="showServiceTreeInbox" text :aria-expanded="sourcesVisible" @click="sourcesVisible = !sourcesVisible">
              {{ t(sourcesVisible ? 'workspaceInbox.hideSources' : 'workspaceInbox.showSources') }}
            </el-button>
            <el-segmented v-model="statusFilter" :options="statusOptions" size="small" @change="loadInbox(true)" />
            <el-select v-model="timeRange" class="inbox-time-filter" :aria-label="t('workspaceInbox.timeRange')" @change="loadInbox(true)">
              <el-option v-for="days in [0, 7, 30]" :key="days" :value="days"
                :label="days ? t('workspaceInbox.lastDays', { count: days }) : t('workspaceInbox.anyTime')" />
            </el-select>
            <div v-if="sourceFilter" class="source-filter-chip">
              <span :title="sourceFilter.sourcePath">{{ sourceFilter.title || sourceFilter.sourcePath }}</span>
              <el-button size="small" text @click="clearSourceFilter">{{ t('workspaceInbox.viewAll') }}</el-button>
            </div>
          </div>
          <div class="inbox-actions">
            <el-button :icon="Refresh" :loading="listLoading" @click="loadInbox(true)">{{ t('common.refresh') }}</el-button>
            <el-button :disabled="currentScopeUnreadCount <= 0 || markingScope" :loading="markingScope" @click="markCurrentScopeRead">
              {{ sourceFilter ? t('workspaceInbox.markCurrentNodeRead') : t('workspaceInbox.markAllRead') }}
            </el-button>
          </div>
        </header>
        <div v-if="shouldShowWorkspaceTabs" class="inbox-workspace-tabs">
          <button v-for="workspace in workspaceTabs" :key="workspace.workspace_key" type="button" class="workspace-tab"
            :class="{ 'is-active': isWorkspaceTabActive(workspace) }" @click="handleWorkspaceTabClick(workspace)">
            <span class="workspace-tab-logo"><el-icon><Monitor /></el-icon></span>
            <span class="workspace-tab-copy"><span class="workspace-tab-title">{{ workspaceTabTitle(workspace) }}</span></span>
            <span v-if="Number(workspace.unread_count || 0) > 0" class="workspace-tab-unread">{{ workspace.unread_count }}</span>
          </button>
        </div>
        <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon :closable="false" class="inbox-error" />
        <div class="inbox-layout" :class="{ 'sources-hidden': !showServiceTreeInbox || !sourcesVisible }">
          <aside v-if="showServiceTreeInbox && sourcesVisible" class="inbox-list-pane" v-loading="sourceTreeLoading">
            <button class="inbox-all-sources" :class="{ 'is-active': !sourceFilter }" @click="clearSourceFilter">
              {{ t('workspaceInbox.allNotifications') }} <span>{{ unreadCount || '' }}</span>
            </button>
            <el-checkbox v-model="showAllSources" class="inbox-source-toggle">{{ t('workspaceInbox.showAllSources') }}</el-checkbox>
            <el-alert v-if="sourceTreeError" :title="sourceTreeError" type="error" :closable="false" />
            <el-button v-if="sourceTreeError" text @click="loadDirectoryTree">{{ t('common.refresh') }}</el-button>
            <el-tree v-else :key="sourceTreeRenderKey" class="inbox-source-tree" :data="visibleSourceTree" :props="sourceTreeProps"
              node-key="full_code_path" :default-expanded-keys="sourceTreeExpandedKeys" :expand-on-click-node="false"
              :highlight-current="true" :current-node-key="activeSourceTreeKey" @node-click="handleSourceTreeNodeClick">
              <template #default="{ data }">
                <ServiceTreeNodeContent :node="data" :active="isSourceTreeNodeActive(data)"
                  :show-notification-badge="hasSourceTreeMessages(data)" :notification-badge-value="sourceTreeNotificationCount(data)"
                  :notification-badge-class="sourceTreeNotificationClass(data)" :notification-badge-title="getSourceTreeNotificationTitle(data)"
                  @notification-click="handleSourceTreeNodeClick(data)" />
              </template>
            </el-tree>
          </aside>
          <section ref="messagePane" class="inbox-detail-pane" v-loading="listLoading" :aria-busy="listLoading">
            <header class="inbox-detail-header">
              <div>
                <h3>{{ sourceFilter?.title || sourceFilter?.sourcePath || t('workspaceInbox.allNotifications') }}</h3>
                <div class="inbox-detail-meta" aria-live="polite">
                  {{ t(searchQuery ? 'workspaceInbox.resultCount' : 'workspaceInbox.messageCount', { count: total }) }}
                </div>
              </div>
            </header>
            <el-empty v-if="!listLoading && !errorMessage && !threadMessages.length" :description="t(searchQuery || timeRange ? 'workspaceInbox.noResults' : 'workspaceInbox.empty')" :image-size="80" />
            <div class="inbox-message-stream">
              <template v-for="(message, index) in selectedThreadMessages" :key="message.id">
                <h4 v-if="index === 0 || dateGroup(message.created_at) !== dateGroup(selectedThreadMessages[index - 1]?.created_at)" class="inbox-date-group">
                  {{ dateGroup(message.created_at) }}
                </h4>
                <article :id="`inbox-message-${message.id}`" class="inbox-message-card"
                  :class="{ 'is-unread': !message.read_at, 'is-active': selectedId === message.id }">
                  <header class="message-summary">
                    <span class="message-card-header">
                      <span class="message-card-title">
                        <span v-if="!message.read_at" class="message-unread-dot" :aria-label="t('workspaceInbox.unread')" />
                        <strong v-html="highlightText(message.title || t('workspaceInbox.untitledMessage'))" />
                      </span>
                      <time class="message-card-time" :datetime="message.created_at" :title="formatExactTime(message.created_at)">{{ formatRelativeTime(message.created_at) }}</time>
                    </span>
                    <span class="message-card-meta">
                      <span v-html="highlightText(sourceSecondaryText(message))" />
                      <span>{{ messageSenderText(message) }}</span>
                      <span v-if="parseMessageFileRefs(message.files).length">{{ t('workspaceInbox.attachmentCount', { count: parseMessageFileRefs(message.files).length }) }}</span>
                    </span>
                  </header>
                  <div :id="`inbox-body-${message.id}`" class="message-expanded" v-loading="detailLoading && selectedId === message.id">
                    <div class="inbox-content inbox-rich-content" v-html="renderMessageContent(selectedId === message.id && selectedMessage ? selectedMessage : message)" />
                    <OutputFilesDisplay v-if="messageFileGroups(selectedId === message.id && selectedMessage ? selectedMessage : message).length" class="inbox-message-files"
                      :file-groups="messageFileGroups(selectedId === message.id && selectedMessage ? selectedMessage : message)" :section-title="t('workspaceInbox.attachments')" />
                    <footer class="message-card-actions">
                      <el-button v-if="message.scheduled_task_id" size="small" type="primary" plain @click="openScheduledExecution(message)">{{ t('workspaceInbox.viewExecution') }}</el-button>
                      <el-button v-if="message.workspace_session_id" size="small" type="primary" plain @click="openWorkspaceSession(message)">{{ t('workspaceInbox.viewSession') }}</el-button>
                      <el-button v-if="sourcePathForMessage(message)" size="small" plain @click="openSourcePath(message)">{{ t('workspaceInbox.viewSource') }}</el-button>
                      <el-button v-if="sourcePathForMessage(message)" size="small" text @click="openMessageSource(message)">{{ t('workspaceInbox.sourceHistory') }}</el-button>
                      <el-button v-if="!message.read_at" size="small" text @click="markMessageRead(message.id)">{{ t('workspaceInbox.markRead') }}</el-button>
                    </footer>
                  </div>
                </article>
              </template>
            </div>
            <div v-if="total > pageSize" class="inbox-pagination">
              <el-pagination v-model:current-page="page" :page-size="pageSize" :total="total" :disabled="listLoading"
                :pager-count="5" size="small" layout="prev, pager, next" @current-change="loadInbox(false)" />
            </div>
          </section>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { ElMessage } from 'element-plus'
import {
  Message as MessageIcon,
  Refresh,
  Monitor,
  Search
} from '@element-plus/icons-vue'
import { Z_INDEX } from '@/architecture/presentation/constants/zIndex'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'
import { escapeHtml, sanitizeHtml } from '@/architecture/shared/sanitizeHtml'
import ServiceTreeNodeContent from './ServiceTreeNodeContent.vue'
import OutputFilesDisplay from './OutputFilesDisplay.vue'
import type { OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'
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
  listMessageInboxWorkspaceCounts,
  markAllMessageInboxItemsRead,
  markMessageInboxItemRead,
  markMessageInboxSourceRead,
  type MessageInboxItem,
  type MessageInboxSourceCount,
  type MessageInboxThread,
  type MessageInboxWorkspaceCount,
  type MessageInboxStatus,
} from '@/architecture/presentation/context/api/message'
import { getAppList, getAppWithServiceTree } from '@/architecture/presentation/context/api/app'
import {
  notifyMessageInboxChanged,
  subscribeToMessageInboxChanges,
} from '@/architecture/presentation/components/utils/messageInboxSync'
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
const inboxInstanceID = Symbol('workspace-inbox')

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
const threadMessages = ref<MessageInboxItem[]>([])
const selectedMessage = ref<MessageInboxItem | null>(null)
const selectedId = computed(() => selectedMessage.value?.id ?? null)
const page = ref(1)
const pageSize = 20
const total = ref(0)
const statusFilter = ref<MessageInboxStatus>('all')
const statusOptions = computed(() => [
  { label: t('workspaceInbox.all'), value: 'all' },
  { label: t('workspaceInbox.unread'), value: 'unread' },
])
const sourceFilter = ref<SourceFilter | null>(null)
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
const remoteSourceTree = ref<ServiceTree[]>([])
const remoteTreeWorkspaceKey = ref('')
const sourceTreeLoading = ref(false)
const sourceTreeError = ref('')
let sourceTreeLoadSeq = 0
const directoryWorkspaceKey = computed(() => workspaceKeyFromRoutePath(sourceFilter.value?.sourcePath || '') || currentWorkspaceKey.value)
const directoryTree = computed(() => directoryWorkspaceKey.value === currentWorkspaceKey.value
  ? props.serviceTree
  : remoteTreeWorkspaceKey.value === directoryWorkspaceKey.value ? remoteSourceTree.value : [])
const showServiceTreeInbox = computed(() => Boolean(directoryWorkspaceKey.value) || directoryTree.value.length > 0)
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
      summaries[path] = node.type === 'package'
        ? Object.values(sourceCountMap.value).reduce<SourceTreeSummary>((summary, item) => {
          const childPath = normalizeSourceTreePath(item.source_path)
          if (childPath === path || childPath.startsWith(`${path}/`)) {
            summary.message_count = Number(summary.message_count || 0) + Number(item.message_count || 0)
            summary.unread_count = Number(summary.unread_count || 0) + Number(item.unread_count || 0)
          }
          return summary
        }, {})
        : sourceCountMap.value[path] || {}
    }
    for (const child of node.children || []) {
      walk(child)
    }
  }
  for (const node of directoryTree.value) {
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

  for (const node of directoryTree.value) {
    walk(node, [])
  }

  return [...expanded]
})
const sourceTreeRenderKey = computed(() => {
  return `${directoryWorkspaceKey.value}:${sourceTreeExpandedKeys.value.join('|')}`
})
const drawerTitle = computed(() => {
  if (!sourceFilter.value) return t('workspaceInbox.title')
  return t('workspaceInbox.sourceNotificationTitle', {
    title: sourceFilter.value.title || t('workspaceInbox.nodeNotifications')
  })
})
const currentScopeUnreadCount = computed(() => sourceFilter.value ? sourceFilterUnreadCount() : unreadCount.value)
const maximized = ref(false)
const sourcesVisible = ref(typeof window === 'undefined' || window.innerWidth > 1024)
const showAllSources = ref(false)
const searchInput = ref('')
const searchQuery = ref('')
const timeRange = ref(0)
const markingScope = ref(false)
const messagePane = ref<HTMLElement | null>(null)
let searchTimer: ReturnType<typeof setTimeout> | undefined
let markSourceReadOnOpen = false
const visibleSourceTree = computed(() => {
  if (showAllSources.value) return directoryTree.value
  const prune = (nodes: ServiceTree[]): ServiceTree[] => nodes.flatMap(node => {
    const children = prune(node.children || [])
    return children.length || hasSourceTreeMessages(node) || isSourceTreeNodeActive(node)
      ? [{ ...node, children }] : []
  })
  return prune(directoryTree.value)
})
onBeforeUnmount(() => clearTimeout(searchTimer))

function scheduleSearch() {
  clearTimeout(searchTimer)
  // Invalidate pending results immediately, before the debounce finishes.
  inboxLoadSeq += 1
  detailLoadSeq += 1
  searchTimer = setTimeout(applySearch, 300)
}
function applySearch() {
  clearTimeout(searchTimer)
  searchQuery.value = searchInput.value.trim()
  void loadInbox(true)
}
function highlightText(text: string) {
  const query = searchQuery.value
  if (!query) return escapeHtml(text)
  const index = text.toLocaleLowerCase().indexOf(query.toLocaleLowerCase())
  if (index < 0) return escapeHtml(text)
  return escapeHtml(text.slice(0, index)) + '<mark>' + escapeHtml(text.slice(index, index + query.length)) + '</mark>' + escapeHtml(text.slice(index + query.length))
}
function dateGroup(value?: string) {
  const date = dayjs(value)
  if (date.isSame(dayjs(), 'day')) return t('workspaceInbox.today')
  if (date.isSame(dayjs().subtract(1, 'day'), 'day')) return t('workspaceInbox.yesterday')
  return t('workspaceInbox.earlier')
}
function openMessageSource(message: MessageInboxItem) {
  clearTimeout(searchTimer)
  searchInput.value = ''
  searchQuery.value = ''
  statusFilter.value = 'all'
  timeRange.value = 0
  openForSource({ sourcePath: sourcePathForMessage(message), title: sourceSecondaryText(message) })
}

const selectedThreadMessages = computed(() => {
  return threadMessages.value
    .slice()
    .sort((a, b) => messageTime(b) - messageTime(a) || b.id - a.id) || []
})

async function loadDirectoryTree() {
  const key = directoryWorkspaceKey.value
  const seq = ++sourceTreeLoadSeq
  sourceTreeError.value = ''
  remoteSourceTree.value = []
  remoteTreeWorkspaceKey.value = ''
  if (!key || key === currentWorkspaceKey.value) {
    sourceTreeLoading.value = false
    return
  }
  sourceTreeLoading.value = true
  try {
    const response = await getAppWithServiceTree(key)
    if (seq !== sourceTreeLoadSeq) return
    remoteSourceTree.value = response.service_tree || []
    remoteTreeWorkspaceKey.value = key
    cacheWorkspaceApps([response.app])
  } catch (error) {
    if (seq === sourceTreeLoadSeq) sourceTreeError.value = error instanceof Error ? error.message : t('workspaceInbox.loadFailed')
  } finally {
    if (seq === sourceTreeLoadSeq) sourceTreeLoading.value = false
  }
}
watch([directoryWorkspaceKey, currentWorkspaceKey], () => { void loadDirectoryTree() }, { immediate: true })
onBeforeUnmount(() => { sourceTreeLoadSeq += 1 })

onMounted(() => {
  void loadUnreadCount()
})

const unsubscribeMessagesUpdated = subscribeToMessageInboxChanges({
  source: inboxInstanceID,
  shouldRefresh: () => props.showTrigger || drawerVisible.value,
  refresh: refreshMessageCountsAfterMutation,
})

onBeforeUnmount(unsubscribeMessagesUpdated)

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
    // Keep the last known count when a transient refresh fails.
  } finally {
    countLoading.value = false
  }
}

function openDrawer() {
  void preloadMarkdown()
  sourceFilter.value = null
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
      includeChildren: sourceNode?.type === 'package' || sourcePath.split('/').filter(Boolean).length === 2,
      kind: sourceNode?.type === 'package' ? 'directory' : 'function',
    }
  } else {
    sourceFilter.value = null
  }

  routeIntentOpening = true
  drawerVisible.value = true
  try {
    await loadInbox(true, Boolean(sourcePath) && !messageID)
    if (messageID) {
      await focusMessageByID(messageID)
      await nextTick()
      messagePane.value?.querySelector(`#inbox-message-${messageID}`)?.scrollIntoView?.({ block: 'nearest' })
    }
  } finally {
    routeIntentOpening = false
  }
}

function openForSource(filter: SourceFilter, autoRead = true) {
  const sourcePath = (filter.sourcePath || '').trim()
  if (!sourcePath) return
  void preloadMarkdown()
  sourceFilter.value = {
    ...filter,
    sourcePath,
  }
  const wasVisible = drawerVisible.value
  markSourceReadOnOpen = !wasVisible && autoRead
  void syncInboxRoute({ sourcePath })
  drawerVisible.value = true
  if (wasVisible) {
    void loadInbox(true, autoRead)
  }
}

function clearSourceFilter() {
  sourceFilter.value = null
  if (window.innerWidth <= 1024) sourcesVisible.value = false
  void syncInboxRoute()
  void loadInbox(true)
}

function handleDrawerOpen() {
  void preloadMarkdown()
  if (routeIntentOpening) {
    void loadUnreadCount()
    return
  }
  const autoRead = markSourceReadOnOpen
  markSourceReadOnOpen = false
  void loadInbox(true, autoRead)
  void loadUnreadCount()
}

function handleDrawerClosed() {
  clearTimeout(searchTimer)
  inboxLoadSeq += 1
  detailLoadSeq += 1
  if (!props.syncRoute || !isInboxOpenQuery(route.query)) return
  appliedRouteInboxKey.value = ''
  const query = { ...route.query }
  clearInboxRouteQuery(query)
  void router.replace({ path: route.path, query })
}

async function loadInbox(resetPage = false, autoRead = false) {
  clearTimeout(searchTimer)
  searchQuery.value = searchInput.value.trim()
  const loadSeq = ++inboxLoadSeq
  detailLoadSeq += 1
  detailLoading.value = false
  selectedMessage.value = null
  if (resetPage) page.value = 1
  listLoading.value = true
  errorMessage.value = ''
  const filter = sourceFilter.value
  try {
    const [resp] = await Promise.all([
      listMessageInbox({
        status: statusFilter.value === 'unread' ? 'unread' : undefined,
        q: searchQuery.value || undefined,
        since: timeRange.value ? dayjs().subtract(timeRange.value, 'day').toISOString() : undefined,
        source_path: filter?.sourcePath,
        include_children: Boolean(filter?.includeChildren),
        page: page.value,
        page_size: pageSize,
      }),
      loadWorkspaceCounts(),
      loadSourceCounts(),
    ])
    if (loadSeq !== inboxLoadSeq) return
    threadMessages.value = resp.list || []
    total.value = resp.total || 0
    if (page.value > 1 && !threadMessages.value.length && total.value > 0) {
      page.value = Math.ceil(total.value / pageSize)
      await loadInbox()
      return
    }
    messagePane.value?.scrollTo?.({ top: 0 })
    if (autoRead && filter?.sourcePath) {
      try {
        await markMessageInboxSourceRead(filter.sourcePath, Boolean(filter.includeChildren))
        if (loadSeq === inboxLoadSeq) {
          const now = new Date().toISOString()
          threadMessages.value = threadMessages.value.map(message => ({ ...message, read_at: message.read_at || now }))
        }
        notifyMessagesUpdated()
        await refreshMessageCountsAfterMutation()
      } catch (error) {
        if (loadSeq === inboxLoadSeq) ElMessage.error(error instanceof Error ? error.message : t('workspaceInbox.markReadFailed'))
      }
    }
  } catch (error) {
    if (loadSeq === inboxLoadSeq) {
      threadMessages.value = []
      total.value = 0
      errorMessage.value = error instanceof Error ? error.message : t('workspaceInbox.loadFailed')
    }
  } finally {
    if (loadSeq === inboxLoadSeq) listLoading.value = false
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
    // Keep the last known counts when a transient refresh fails.
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
    // Keep the last known counts when a transient refresh fails.
  }
}

function refreshMessageCountsAfterMutation() {
  return Promise.all([
    loadUnreadCount(),
    loadWorkspaceCounts(),
    ...(showServiceTreeInbox.value ? [loadSourceCounts()] : []),
  ])
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
    threadMessages.value = threadMessages.value.map(message => message.id === detail.id ? detail : message)
    void syncInboxRoute({
      messageId: detail.id,
      sourcePath: sourceFilter.value?.sourcePath,
      traceId: detail.trace_id,
    })
    if (!detail.read_at) {
      await markMessageInboxItemRead(item.id)
      if (loadSeq === detailLoadSeq) selectedMessage.value = { ...detail, read_at: new Date().toISOString() }
      updateListReadState(item.id)
      notifyMessagesUpdated()
      await refreshMessageCountsAfterMutation()
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
    if (!threadMessages.value.some(item => item.id === detail.id)) {
      threadMessages.value = [detail, ...threadMessages.value]
    }
    if (!detail.read_at) {
      await markMessageInboxItemRead(detail.id)
      const readDetail = { ...detail, read_at: new Date().toISOString() }
      if (loadSeq === detailLoadSeq) selectedMessage.value = readDetail
      threadMessages.value = threadMessages.value.map(item => item.id === detail.id ? readDetail : item)
      updateListReadState(detail.id)
      notifyMessagesUpdated()
      await refreshMessageCountsAfterMutation()
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

async function markMessageRead(id: number) {
  try {
    await markMessageInboxItemRead(id)
    updateListReadState(id)
    if (selectedMessage.value?.id === id) {
      selectedMessage.value = { ...selectedMessage.value, read_at: new Date().toISOString() }
    }
    notifyMessagesUpdated()
    await refreshMessageCountsAfterMutation()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('workspaceInbox.markReadFailed'))
  }
}

async function markCurrentScopeRead() {
  if (markingScope.value) return
  markingScope.value = true
  const filter = sourceFilter.value
  try {
    if (filter) await markMessageInboxSourceRead(filter.sourcePath, Boolean(filter.includeChildren))
    else await markAllMessageInboxItemsRead()
    notifyMessagesUpdated()
    await Promise.all([loadInbox(), refreshMessageCountsAfterMutation()])
    ElMessage.success(t('workspaceInbox.allReadSuccess'))
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('workspaceInbox.markReadFailed'))
  } finally {
    markingScope.value = false
  }
}

function updateListReadState(id: number) {
  threadMessages.value = threadMessages.value.map(item => item.id === id
    ? { ...item, read_at: item.read_at || new Date().toISOString() } : item)
}

function notifyMessagesUpdated() {
  emit('messages-updated')
  notifyMessageInboxChanged(inboxInstanceID)
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
  return walk(directoryTree.value)
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
  return Object.values(sourceCountMap.value).reduce((count, item) => {
    const path = normalizeSourceTreePath(item.source_path)
    return count + (path === filter.sourcePath || (filter.includeChildren && path.startsWith(`${filter.sourcePath}/`)) ? Number(item.unread_count || 0) : 0)
  }, 0)
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
    includeChildren: node.type === 'package',
    kind: node.type === 'package' ? 'directory' : 'function',
  }
  if (window.innerWidth <= 1024) sourcesVisible.value = false
  if (previousSourcePath !== sourcePath) {
    threadMessages.value = []
    selectedMessage.value = null
    total.value = 0
  }
  void syncInboxRoute({ sourcePath })
  void loadInbox(true, true)
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
  return directoryWorkspaceKey.value === workspaceKeyForCount(item)
}

function handleWorkspaceTabClick(item: MessageInboxWorkspaceCount) {
  const path = workspaceKeyForCount(item)
  if (!path) return
  if (window.innerWidth > 1024) sourcesVisible.value = true
  openForSource({ sourcePath: path, title: workspaceTabTitle(item), includeChildren: true }, false)
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
  const taskID = item.scheduled_task_id || 0
  if (!taskID) return
  const executionID = item.scheduled_execution_id || 0
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

function parseMessageFileRefs(files?: string): string[] {
  return Array.from(new Set((files || '')
    .split(',')
    .map(ref => ref.trim().replace(/^\/+/, ''))
    .filter(Boolean)))
}

function messageFileGroups(item?: MessageInboxItem | null): OutputFileGroup[] {
  const refs = parseMessageFileRefs(item?.files)
  if (refs.length === 0) return []
  return [{
    label: t('workspaceInbox.attachments'),
    files: refs.map(ref => ({
      ref,
      name: ref.split('/').pop() || t('workspaceInbox.attachment')
    }))
  }]
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
.workspace-inbox { display: inline-flex; align-items: center; }
.workspace-inbox-button { width: 34px; height: 34px; min-height: 34px; }
.inbox-heading { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.inbox-heading h2 { margin: 0; font-size: 20px; color: var(--el-text-color-primary); }
:global(.workspace-inbox-drawer .el-drawer__header) { margin-bottom: 0; padding: 20px 24px 12px; }
:global(.workspace-inbox-drawer .el-drawer__body) { padding: 12px 24px 24px; overflow: hidden; }
.inbox-shell {
  --inbox-line: var(--app-shell-panel-border, var(--el-border-color-lighter));
  --inbox-paper: var(--app-shell-panel-bg-strong, var(--el-bg-color));
  display: flex; height: 100%; min-height: 0; flex-direction: column; gap: 14px;
  color: var(--el-text-color-primary);
}
.inbox-search { flex-shrink: 0; }
.inbox-search :deep(.el-input__wrapper) { min-height: 42px; border-radius: 10px; }
.inbox-toolbar, .inbox-filter, .inbox-actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.inbox-toolbar { justify-content: space-between; }
.inbox-time-filter { width: 132px; }
.source-filter-chip { display: inline-flex; align-items: center; min-width: 0; gap: 8px; font-size: 13px; }
.source-filter-chip > span { max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.inbox-workspace-tabs { display: flex; gap: 8px; overflow-x: auto; flex-shrink: 0; padding: 1px; }
.workspace-tab { display: flex; align-items: center; gap: 8px; border: 1px solid var(--inbox-line); border-radius: 8px; background: transparent; color: inherit; padding: 8px 12px; cursor: pointer; white-space: nowrap; }
.workspace-tab.is-active { border-color: rgba(var(--el-color-primary-rgb), .25); background: rgba(var(--el-color-primary-rgb), .05); }
.workspace-tab-unread { background: var(--el-color-primary); color: white; border-radius: 12px; padding: 1px 6px; font-size: 11px; }
.inbox-error { flex-shrink: 0; }
.inbox-layout { display: grid; grid-template-columns: 260px minmax(0, 1fr); gap: 20px; flex: 1; min-height: 0; }
.inbox-layout.sources-hidden { grid-template-columns: minmax(0, 1fr); }
.inbox-list-pane, .inbox-detail-pane { min-width: 0; min-height: 0; overflow: auto; overscroll-behavior: contain; }
.inbox-list-pane { border-right: 1px solid var(--inbox-line); padding-right: 12px; }
.inbox-all-sources { display: flex; justify-content: space-between; width: 100%; border: 0; border-radius: 8px; padding: 12px; background: transparent; color: inherit; text-align: left; font: inherit; cursor: pointer; }
.inbox-all-sources.is-active { background: rgba(var(--el-color-primary-rgb), .06); color: var(--el-text-color-regular); font-weight: 500; }
.inbox-source-toggle { margin: 8px 12px; }
.inbox-source-tree { background: transparent; }
.inbox-source-tree :deep(.el-tree-node__content) { height: 38px; border-radius: 6px; }
.inbox-shell .inbox-source-tree :deep(.el-tree-node.is-current > .el-tree-node__content),
.inbox-shell .inbox-source-tree :deep(.el-tree-node:focus > .el-tree-node__content) {
  background: rgba(var(--el-color-primary-rgb), .06) !important;
  color: var(--el-text-color-regular) !important;
  box-shadow: inset 2px 0 0 rgba(var(--el-color-primary-rgb), .35);
}
.inbox-source-tree :deep(.tree-node.is-active .node-label) { color: var(--el-text-color-regular); font-weight: 500; }
.inbox-source-tree :deep(.el-tree-node__content:hover) { background: rgba(var(--el-color-primary-rgb), .04) !important; }
.inbox-detail-pane { padding: 0 8px 12px; }
.inbox-detail-header { display: flex; justify-content: space-between; padding: 4px 0 12px; border-bottom: 1px solid var(--inbox-line); }
.inbox-detail-header h3 { margin: 0 0 6px; font-size: 17px; overflow-wrap: anywhere; }
.inbox-detail-meta { font-size: 12px; color: var(--el-text-color-secondary); }
.inbox-message-stream { display: flex; flex-direction: column; gap: 18px; }
.inbox-date-group { margin: 18px 0 4px; font-size: 12px; font-weight: 500; color: var(--el-text-color-secondary); }
.inbox-message-card { border: 1px solid var(--inbox-line); border-radius: 12px; background: var(--inbox-paper); box-shadow: 0 2px 8px rgba(0, 0, 0, .06); overflow: hidden; flex-shrink: 0; }
.inbox-message-card.is-active { border-color: rgba(var(--el-color-primary-rgb), .45); }
.message-summary { display: flex; flex-direction: column; gap: 8px; width: 100%; padding: 16px 18px; border: 0; background: transparent; color: inherit; font: inherit; text-align: left; border-bottom: 1px solid var(--inbox-line); }
.message-summary:focus-visible, .inbox-all-sources:focus-visible { outline: 2px solid var(--el-color-primary); outline-offset: -2px; }
.message-card-header { display: flex; justify-content: space-between; gap: 16px; align-items: baseline; }
.message-card-title { display: flex; align-items: baseline; gap: 8px; min-width: 0; }
.message-card-title strong { font-size: 15px; font-weight: 500; line-height: 1.5; overflow-wrap: anywhere; }
.is-unread .message-card-title strong { font-weight: 650; }
.message-unread-dot { width: 7px; height: 7px; flex-shrink: 0; border-radius: 50%; background: var(--el-color-primary); }
.message-card-time { flex-shrink: 0; font-size: 12px; color: var(--el-text-color-secondary); }
.message-card-meta { display: flex; flex-wrap: wrap; gap: 4px 12px; color: var(--el-text-color-secondary); font-size: 12px; }
.message-card-meta > span { max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.message-summary :deep(mark) { color: inherit; background: color-mix(in srgb, var(--el-color-warning) 30%, transparent); border-radius: 2px; }
.message-expanded { padding: 20px; }
.message-expanded :deep(p), .message-expanded :deep(ul), .message-expanded :deep(ol) { max-width: 85ch; }
.message-card-actions { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 20px; padding-top: 14px; border-top: 1px solid var(--inbox-line); }
.message-card-actions :deep(.el-button + .el-button) { margin-left: 0; }
.inbox-message-files { margin-top: 16px; }
.inbox-pagination { display: flex; justify-content: center; padding: 20px 0 4px; }
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


@media (max-width: 1024px) {
  .inbox-layout { position: relative; grid-template-columns: minmax(0, 1fr); }
  .inbox-list-pane { position: absolute; inset: 0; z-index: 2; padding: 8px; border: 1px solid var(--inbox-line); border-radius: 10px; background: var(--inbox-paper); }
}
@media (max-width: 760px) {
  :global(.workspace-inbox-drawer) { width: 100vw !important; }
  :global(.workspace-inbox-drawer .el-drawer__header) { padding: 16px 12px 8px; }
  :global(.workspace-inbox-drawer .el-drawer__body) { padding: 8px 12px 12px; }
  .inbox-maximize { display: none; }
  .inbox-layout { position: relative; grid-template-columns: minmax(0, 1fr); }
  .inbox-list-pane { position: absolute; inset: 0; z-index: 2; padding: 8px; border: 1px solid var(--inbox-line); border-radius: 10px; background: var(--inbox-paper); }
  .inbox-detail-pane { padding: 0 0 12px; }
  .inbox-actions { width: 100%; justify-content: flex-end; }
  .message-summary { padding: 14px 12px; }
  .message-expanded { padding: 16px 12px; }
}
</style>
