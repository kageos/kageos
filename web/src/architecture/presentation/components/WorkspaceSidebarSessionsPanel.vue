<template>
  <div class="right-sidebar-session-panel" data-testid="workspace-sidebar-sessions">
    <div class="right-session-header" data-testid="workspace-sidebar-sessions-header">
      <el-icon :size="16" color="var(--el-color-primary)"><FolderOpened /></el-icon>
      <span class="right-session-dir">{{ dirName }}</span>
    </div>

    <div class="right-session-tabs" data-testid="workspace-sidebar-sessions-tabs">
      <div :class="['right-tab', { active: activeTab === 'all' }]" data-testid="workspace-sidebar-tab-all" @click="updateActiveTab('all')">
        全部
      </div>
      <div :class="['right-tab', { active: activeTab === 'running' }]" data-testid="workspace-sidebar-tab-running" @click="updateActiveTab('running')">
        执行中
        <span v-if="runningCount > 0" class="right-tab-badge">{{ runningCount }}</span>
      </div>
      <div :class="['right-tab', { active: activeTab === 'finished' }]" data-testid="workspace-sidebar-tab-finished" @click="updateActiveTab('finished')">
        已结束
      </div>
    </div>

    <el-input
      :model-value="searchKeyword"
      class="right-session-search"
      placeholder="搜索会话…"
      clearable
      :prefix-icon="Search"
      data-testid="workspace-sidebar-search"
      @update:model-value="updateSearchKeyword"
    />

    <div class="right-session-list" data-testid="workspace-sidebar-list" v-loading="loading">
      <div
        v-for="session in sessions"
        :key="session.session_id"
        :class="['right-session-card', { generating: session.status === 'generating' }]"
        :data-testid="`workspace-session-card-${session.session_id}`"
        @click="$emit('open-session', session)"
      >
        <div class="right-session-card-head">
          <el-icon
            v-if="session.status === 'generating'"
            class="is-loading"
            :size="12"
            color="var(--el-color-primary)"
          >
            <Loading />
          </el-icon>
          <span class="right-session-card-title">{{ session.title || '未命名会话' }}</span>
        </div>

        <div v-if="session.user" class="right-session-card-user">
          <UserDisplay :username="session.user" mode="simple" size="small" />
        </div>

        <div class="right-session-card-meta">
          <el-tag v-if="session.status === 'generating'" type="primary" size="small" effect="light">执行中</el-tag>
          <el-tag v-else-if="session.status === 'done'" type="success" size="small" effect="plain">已完成</el-tag>
          <el-tag v-else-if="session.status === 'cancelled'" type="info" size="small" effect="plain">已取消</el-tag>
          <span class="right-session-time">{{ formatRelativeTime(session.updated_at) }}</span>
        </div>

        <div v-if="session.status === 'generating'" class="right-session-card-actions">
          <el-button
            size="small"
            link
            type="danger"
            :loading="cancellingTaskId === session.session_id"
            :data-testid="`workspace-session-stop-${session.session_id}`"
            @click.stop="$emit('cancel-task', session)"
          >
            停止
          </el-button>
        </div>
      </div>

      <div v-if="sessions.length === 0 && !loading" class="right-session-empty">
        <el-empty :description="emptyDescription" :image-size="48" />
      </div>
    </div>

    <div class="right-session-footer" data-testid="workspace-sidebar-footer">
      <el-button type="primary" :icon="ChatDotRound" class="right-new-session-btn" data-testid="workspace-sidebar-create-session" @click="$emit('create-session')">
        新增会话
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ChatDotRound, FolderOpened, Loading, Search } from '@element-plus/icons-vue'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import type { WorkspaceSessionItem } from '@/api/workspace'

type SidebarTab = 'all' | 'running' | 'finished'

const props = defineProps<{
  dirName: string
  loading: boolean
  activeTab: SidebarTab
  searchKeyword: string
  runningCount: number
  sessions: WorkspaceSessionItem[]
  cancellingTaskId: string | null
  formatRelativeTime: (timeStr: string) => string
}>()

const emit = defineEmits<{
  (e: 'update:activeTab', value: SidebarTab): void
  (e: 'update:searchKeyword', value: string): void
  (e: 'open-session', session: WorkspaceSessionItem): void
  (e: 'cancel-task', session: WorkspaceSessionItem): void
  (e: 'create-session'): void
}>()

const emptyDescription = computed(() => {
  if (props.searchKeyword) return '无匹配会话'
  if (props.activeTab === 'running') return '暂无执行中的会话'
  if (props.activeTab === 'finished') return '暂无已结束的会话'
  return '暂无会话记录'
})

function updateActiveTab(value: SidebarTab) {
  emit('update:activeTab', value)
}

function updateSearchKeyword(value: string | number) {
  emit('update:searchKeyword', String(value ?? ''))
}
</script>

<style scoped lang="scss">
.right-sidebar-session-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  padding: 10px 0 0;
}

.right-session-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 14px 10px;
  padding: 14px 14px 12px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 18px;
  background: var(--app-shell-panel-bg-strong);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
  flex-shrink: 0;
}

.right-session-dir {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.right-session-tabs {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 14px 2px;
  flex-shrink: 0;
}

.right-tab {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 13px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  border: 1px solid transparent;
  border-radius: 999px;
  transition: all 0.2s ease;
  white-space: nowrap;

  &:hover {
    color: var(--el-color-primary);
    background: rgba(var(--el-color-primary-rgb), 0.06);
  }

  &.active {
    color: var(--el-color-primary);
    font-weight: 600;
    border-color: rgba(var(--el-color-primary-rgb), 0.16);
    background: rgba(var(--el-color-primary-rgb), 0.1);
    box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
  }
}

.right-tab-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  margin-left: 4px;
  font-size: 10px;
  line-height: 1;
  color: #fff;
  background: var(--el-color-danger);
  border-radius: 8px;
}

.right-session-search {
  flex-shrink: 0;
  padding: 6px 14px 4px;
}

.right-session-search :deep(.el-input__wrapper) {
  border-radius: 14px;
  padding: 4px 12px;
  background: var(--app-shell-panel-bg-strong);
  border: 1px solid var(--app-shell-panel-border);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
}

.right-session-list {
  flex: 1;
  overflow-y: auto;
  padding: 10px 14px 18px;
}

.right-session-card {
  padding: 14px 14px 12px;
  margin-bottom: 10px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 20px;
  background: var(--app-shell-panel-bg-strong);
  box-shadow: 0 16px 32px rgba(15, 23, 42, 0.07);
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--el-color-primary);
    background: var(--app-shell-panel-bg-strong);
    box-shadow: 0 20px 36px rgba(15, 23, 42, 0.11);
    transform: translateY(-2px);
  }

  &.generating {
    border-left: 3px solid var(--el-color-primary);
  }
}

.right-session-card-head {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}

.right-session-card-user {
  margin-bottom: 8px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.right-session-card-user :deep(.user-display-wrapper) {
  display: inline-flex;
}

.right-session-card-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.right-session-card-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  font-size: 11px;
  color: var(--el-text-color-placeholder);
}

.right-session-time {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
}

.right-session-card-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.right-session-empty {
  padding: 32px 8px;
  text-align: center;
}

.right-session-footer {
  flex-shrink: 0;
  padding: 12px 14px 16px;
  border-top: 1px solid var(--app-shell-panel-border);
  background: var(--app-shell-panel-bg);
}

.right-new-session-btn {
  width: 100%;
  height: 44px;
  border-radius: 16px;
  box-shadow: 0 14px 30px rgba(var(--el-color-primary-rgb), 0.2);
}
</style>
