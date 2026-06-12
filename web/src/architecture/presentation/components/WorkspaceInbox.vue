<template>
  <div class="workspace-inbox">
    <el-tooltip content="站内信" placement="bottom" effect="light">
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
      title="站内信"
      direction="rtl"
      size="min(860px, 92vw)"
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
          </div>
          <div class="inbox-actions">
            <el-button :icon="Refresh" :loading="listLoading" @click="loadInbox(true)">
              刷新
            </el-button>
            <el-button :disabled="unreadCount <= 0" @click="markAllRead">
              全部已读
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
              v-if="!listLoading && inboxItems.length === 0"
              description="暂无站内信"
              :image-size="80"
            />

            <button
              v-for="item in inboxItems"
              :key="item.id"
              type="button"
              class="inbox-list-item"
              :class="{ 'is-active': selectedId === item.id, 'is-unread': !item.read_at }"
              @click="selectMessage(item)"
            >
              <span class="inbox-unread-dot"></span>
              <span class="inbox-list-copy">
                <span class="inbox-list-title">{{ item.title || '无标题消息' }}</span>
                <span class="inbox-list-preview">{{ previewText(item.content) }}</span>
                <span class="inbox-list-meta">
                  <span>{{ item.from || 'system' }}</span>
                  <span>{{ formatTime(item.created_at) }}</span>
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
              v-if="!selectedMessage"
              description="选择一条消息查看详情"
              :image-size="96"
            />

            <article v-else class="inbox-detail">
              <header class="inbox-detail-header">
                <div>
                  <h3>{{ selectedMessage.title || '无标题消息' }}</h3>
                  <div class="inbox-detail-meta">
                    <span>来自 {{ selectedMessage.from || 'system' }}</span>
                    <span>{{ formatTime(selectedMessage.created_at) }}</span>
                    <el-tag v-if="selectedMessage.read_at" size="small" type="info">已读</el-tag>
                    <el-tag v-else size="small" type="primary">未读</el-tag>
                  </div>
                </div>
                <el-button
                  v-if="!selectedMessage.read_at"
                  size="small"
                  type="primary"
                  plain
                  @click="markSelectedRead"
                >
                  标记已读
                </el-button>
              </header>

              <div class="inbox-content">{{ selectedMessage.content }}</div>

              <dl class="inbox-source" v-if="selectedMessage.full_code_path || selectedMessage.source_type || selectedMessage.trace_id">
                <dt v-if="selectedMessage.full_code_path">来源路径</dt>
                <dd v-if="selectedMessage.full_code_path">{{ selectedMessage.full_code_path }}</dd>
                <dt v-if="selectedMessage.source_type">来源类型</dt>
                <dd v-if="selectedMessage.source_type">{{ selectedMessage.source_type }}</dd>
                <dt v-if="selectedMessage.trace_id">Trace</dt>
                <dd v-if="selectedMessage.trace_id">{{ selectedMessage.trace_id }}</dd>
              </dl>
            </article>
          </section>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import dayjs from 'dayjs'
import { ElMessage } from 'element-plus'
import { Message as MessageIcon, Refresh } from '@element-plus/icons-vue'
import { Z_INDEX } from '@/architecture/presentation/constants/zIndex'
import {
  getMessageInboxItem,
  getMessageInboxUnreadCount,
  listMessageInbox,
  markAllMessageInboxItemsRead,
  markMessageInboxItemRead,
  type MessageInboxItem,
  type MessageInboxStatus,
} from '@/architecture/presentation/context/api/message'

const drawerVisible = ref(false)
const countLoading = ref(false)
const listLoading = ref(false)
const detailLoading = ref(false)
const errorMessage = ref('')
const unreadCount = ref(0)
const inboxItems = ref<MessageInboxItem[]>([])
const selectedMessage = ref<MessageInboxItem | null>(null)
const selectedId = computed(() => selectedMessage.value?.id ?? null)
const page = ref(1)
const pageSize = 20
const total = ref(0)
const statusFilter = ref<MessageInboxStatus>('all')
const statusOptions = [
  { label: '全部', value: 'all' },
  { label: '未读', value: 'unread' },
]

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
  drawerVisible.value = true
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
    const resp = await listMessageInbox({
      status: statusFilter.value === 'unread' ? 'unread' : undefined,
      page: page.value,
      page_size: pageSize,
    })
    inboxItems.value = resp.list || []
    total.value = resp.total || 0
    if (selectedMessage.value && !inboxItems.value.some(item => item.id === selectedMessage.value?.id)) {
      selectedMessage.value = null
    }
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '加载站内信失败'
  } finally {
    listLoading.value = false
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
    }
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '加载消息详情失败'
  } finally {
    detailLoading.value = false
  }
}

async function markSelectedRead() {
  if (!selectedMessage.value) return
  await markMessageRead(selectedMessage.value.id)
}

async function markMessageRead(id: number) {
  try {
    await markMessageInboxItemRead(id)
    updateListReadState(id)
    if (selectedMessage.value?.id === id) {
      selectedMessage.value = { ...selectedMessage.value, read_at: new Date().toISOString() }
    }
    await loadUnreadCount()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '标记已读失败')
  }
}

async function markAllRead() {
  try {
    await markAllMessageInboxItemsRead()
    inboxItems.value = inboxItems.value.map(item => ({ ...item, read_at: item.read_at || new Date().toISOString() }))
    if (selectedMessage.value) {
      selectedMessage.value = { ...selectedMessage.value, read_at: selectedMessage.value.read_at || new Date().toISOString() }
    }
    unreadCount.value = 0
    ElMessage.success('已全部标记为已读')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '全部已读失败')
  }
}

function updateListReadState(id: number) {
  inboxItems.value = inboxItems.value.map(item => {
    if (item.id !== id) return item
    return { ...item, read_at: item.read_at || new Date().toISOString() }
  })
}

function previewText(content?: string) {
  const text = (content || '').replace(/\s+/g, ' ').trim()
  return text.length > 90 ? `${text.slice(0, 90)}...` : text || '无内容'
}

function formatTime(value?: string) {
  if (!value) return '-'
  const parsed = dayjs(value)
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm') : value
}
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
  grid-template-columns: minmax(260px, 320px) minmax(0, 1fr);
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
  grid-template-columns: 10px minmax(0, 1fr);
  gap: 8px;
  padding: 11px 10px;
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

  &.is-unread .inbox-unread-dot {
    background: var(--el-color-primary);
  }
}

.inbox-unread-dot {
  width: 8px;
  height: 8px;
  margin-top: 6px;
  border-radius: 999px;
  background: transparent;
}

.inbox-list-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 5px;
}

.inbox-list-title,
.inbox-list-preview {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.inbox-list-title {
  font-size: 13px;
  font-weight: 650;
}

.inbox-list-preview {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.inbox-list-meta {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  color: var(--el-text-color-placeholder);
  font-size: 11px;
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
}
</style>
