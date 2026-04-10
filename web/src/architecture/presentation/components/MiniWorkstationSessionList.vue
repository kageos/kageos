<template>
  <div class="mini-session-sidebar">
    <div class="mini-session-header">
      <span class="mini-session-title">会话列表</span>
      <el-button text :icon="Plus" size="small" @click="$emit('new')" title="新建会话" />
    </div>
    <div class="mini-session-list" v-loading="loading">
      <div
        :class="['mini-session-card', 'mini-session-new', { active: !activeSessionId }]"
        @click="$emit('new')"
      >
        <el-icon class="mini-session-new-icon"><Plus /></el-icon>
        <span>新建会话</span>
      </div>
      <div
        v-for="session in sessions"
        :key="session.session_id"
        :class="['mini-session-card', { active: session.session_id === activeSessionId }, { generating: session.status === 'generating' }]"
        @click="$emit('select', session.session_id)"
      >
        <div class="mini-session-card-head">
          <el-icon
            v-if="session.status === 'generating'"
            class="is-loading"
            :size="12"
            color="var(--el-color-primary)"
          >
            <Loading />
          </el-icon>
          <span class="mini-session-card-title">{{ session.title || '未命名会话' }}</span>
        </div>
        <div v-if="session.user" class="mini-session-card-user">
          <UserDisplay :username="session.user" mode="simple" size="small" />
        </div>
        <div class="mini-session-card-time">
          <span v-if="session.status === 'generating'" class="mini-session-status">执行中</span>
          <span>{{ formatRelativeTime(session.updated_at) }}</span>
        </div>
      </div>
      <div v-if="sessions.length === 0 && !loading" class="mini-session-empty">
        <span>暂无会话</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Loading, Plus } from '@element-plus/icons-vue'
import type { WorkspaceSessionItem } from '@/api/workspace'
import UserDisplay from '@/shared/components/UserDisplay.vue'

defineProps<{
  sessions: WorkspaceSessionItem[]
  loading: boolean
  activeSessionId?: string
  formatRelativeTime: (timeStr: string) => string
}>()

defineEmits<{
  (e: 'new'): void
  (e: 'select', sessionId: string): void
}>()
</script>

<style scoped>
.mini-session-sidebar {
  width: 220px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-blank);
}

.mini-session-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  flex-shrink: 0;
}

.mini-session-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.mini-session-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.mini-session-card {
  padding: 10px;
  margin-bottom: 6px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  background: var(--el-bg-color);
  cursor: pointer;
  transition: all 0.15s;
}

.mini-session-card:hover {
  border-color: var(--el-color-primary);
  background: var(--el-fill-color-lighter);
}

.mini-session-card.active {
  border-color: var(--el-color-primary);
  border-width: 2px;
}

.mini-session-card.generating {
  border-left: 2px solid var(--el-color-primary);
}

.mini-session-new {
  border-style: dashed;
  background: var(--el-fill-color-lighter);
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.mini-session-new:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary);
}

.mini-session-new-icon {
  color: var(--el-color-primary);
}

.mini-session-card-head {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 4px;
}

.mini-session-card-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.mini-session-card-user {
  margin-top: 4px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.mini-session-card-user :deep(.user-display-wrapper) {
  display: inline-flex;
}

.mini-session-card-time {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  display: flex;
  align-items: center;
  gap: 6px;
}

.mini-session-status {
  color: var(--el-color-primary);
  font-size: 11px;
  font-weight: 500;
}

.mini-session-empty {
  padding: 24px;
  text-align: center;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}
</style>
