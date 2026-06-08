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
        :class="['mini-session-card', getSessionStatusClass(session.status), { active: session.session_id === activeSessionId }]"
        @click="$emit('select', session.session_id)"
      >
        <div class="mini-session-card-head">
          <el-icon
            v-if="isGeneratingSession(session.status)"
            class="is-loading"
            :size="12"
            color="var(--el-color-primary)"
          >
            <Loading />
          </el-icon>
          <span class="mini-session-card-title">{{ session.title || '未命名会话' }}</span>
        </div>
        <div v-if="session.role_display_name" class="mini-session-role">
          {{ session.role_display_name }}
        </div>
        <div v-if="session.user" class="mini-session-card-user">
          <UserDisplay :username="session.user" mode="simple" size="small" />
        </div>
        <div class="mini-session-card-time">
          <span
            v-if="getSessionStatusLabel(session.status)"
            :class="['mini-session-status', getSessionStatusClass(session.status)]"
          >
            {{ getSessionStatusLabel(session.status) }}
          </span>
          <span>{{ formatRelativeTime(session.updated_at) }}</span>
        </div>
        <div class="mini-session-card-timestamp">
          {{ formatSessionTimestamp(session.updated_at || session.created_at) }}
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
import type { WorkspaceSessionItem } from '@/architecture/presentation/context/api/workspace'
import UserDisplay from '@/architecture/presentation/shared/components/UserDisplay.vue'

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

function formatSessionTimestamp(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  const y = date.getFullYear()
  const M = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  const h = String(date.getHours()).padStart(2, '0')
  const m = String(date.getMinutes()).padStart(2, '0')
  const s = String(date.getSeconds()).padStart(2, '0')
  return `${y}-${M}-${d} ${h}:${m}:${s}`
}

function normalizeSessionStatus(status?: string): string {
  return String(status || '').trim().toLowerCase()
}

function isGeneratingSession(status?: string): boolean {
  return normalizeSessionStatus(status) === 'generating'
}

function getSessionStatusClass(status?: string): string {
  const normalized = normalizeSessionStatus(status)
  if (normalized === 'generating') return 'generating'
  if (['output', 'new_file', 'new_output', 'has_output', 'pending_test'].includes(normalized)) return 'output'
  if (['pending_confirmation', 'pending_build_repair', 'waiting', 'pending'].includes(normalized)) return 'waiting'
  if (normalized === 'done') return 'done'
  if (normalized === 'cancelled') return 'cancelled'
  if (normalized === 'failed') return 'failed'
  return 'active'
}

function getSessionStatusLabel(status?: string): string {
  const normalized = normalizeSessionStatus(status)
  const map: Record<string, string> = {
    generating: '执行中',
    output: '新文件',
    new_file: '新文件',
    new_output: '新文件',
    has_output: '新文件',
    pending_confirmation: 'PRD 待确认',
    pending_test: '待自动测试',
    pending_build_repair: '修复待确认',
    done: '已完成',
    cancelled: '已取消',
    failed: '失败'
  }
  return map[normalized] || ''
}
</script>

<style scoped>
.mini-session-sidebar {
  width: 220px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid rgba(96, 231, 255, 0.14);
  background:
    radial-gradient(circle at 0% 0%, rgba(34, 211, 238, 0.12), transparent 34%),
    linear-gradient(180deg, rgba(8, 21, 37, 0.86), rgba(4, 12, 24, 0.76));
}

.mini-session-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-bottom: 1px solid rgba(96, 231, 255, 0.14);
  background: rgba(34, 211, 238, 0.055);
  flex-shrink: 0;
}
.mini-session-header :deep(.el-button) {
  color: var(--mini-cyber-accent, #22d3ee);
  border-radius: 8px;
}
.mini-session-header :deep(.el-button:hover) {
  background: rgba(34, 211, 238, 0.12);
}

.mini-session-title {
  font-size: 13px;
  font-weight: 800;
  letter-spacing: 0.08em;
  color: var(--mini-cyber-text, #d8f8ff);
}

.mini-session-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  scrollbar-color: rgba(34, 211, 238, 0.3) transparent;
}

.mini-session-card {
  padding: 10px;
  margin-bottom: 6px;
  border: 1px solid rgba(96, 231, 255, 0.14);
  border-radius: 12px;
  background:
    linear-gradient(145deg, rgba(9, 28, 48, 0.62), rgba(4, 12, 24, 0.46)),
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), transparent);
  cursor: pointer;
  transition: transform 0.16s ease, border-color 0.16s ease, background 0.16s ease, box-shadow 0.16s ease;
}

.mini-session-card:hover {
  transform: translateY(-1px);
  border-color: rgba(34, 211, 238, 0.46);
  background:
    linear-gradient(145deg, rgba(16, 46, 72, 0.72), rgba(5, 16, 30, 0.52)),
    linear-gradient(180deg, rgba(255, 255, 255, 0.05), transparent);
  box-shadow: 0 10px 28px rgba(0, 0, 0, 0.18), 0 0 18px rgba(34, 211, 238, 0.08);
}

.mini-session-card.active {
  border-color: rgba(34, 211, 238, 0.68);
  box-shadow: inset 3px 0 0 rgba(34, 211, 238, 0.88), 0 0 22px rgba(34, 211, 238, 0.14);
}

.mini-session-card.generating {
  border-left: 2px solid var(--mini-cyber-accent, #22d3ee);
  animation: miniSessionRunning 1.8s ease-in-out infinite;
}

.mini-session-card.waiting {
  border-left: 2px solid #f6bd4d;
  box-shadow: inset 3px 0 0 rgba(246, 189, 77, 0.52), 0 0 18px rgba(246, 189, 77, 0.12);
}

.mini-session-card.output {
  border-left: 2px solid #37a3ff;
  box-shadow: inset 3px 0 0 rgba(55, 163, 255, 0.52), 0 0 18px rgba(55, 163, 255, 0.12);
}

.mini-session-card.done {
  border-left: 2px solid #7df5c4;
}

.mini-session-card.cancelled {
  border-left: 2px solid rgba(184, 225, 235, 0.48);
}

.mini-session-card.failed {
  border-left: 2px solid #ff6b6b;
}

.mini-session-new {
  border-style: dashed;
  background: rgba(34, 211, 238, 0.08);
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--mini-cyber-muted, rgba(184, 225, 235, 0.68));
}

.mini-session-new:hover {
  color: var(--mini-cyber-accent, #22d3ee);
  border-color: rgba(34, 211, 238, 0.54);
}

.mini-session-new-icon {
  color: var(--mini-cyber-accent, #22d3ee);
}

.mini-session-card-head {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 4px;
}

.mini-session-card-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--mini-cyber-text, #d8f8ff);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.mini-session-card-user {
  margin-top: 4px;
  font-size: 11px;
  color: var(--mini-cyber-muted, rgba(184, 225, 235, 0.68));
}

.mini-session-card-user :deep(.user-display-wrapper) {
  display: inline-flex;
}

.mini-session-role {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  max-width: 100%;
  padding: 2px 6px;
  margin-bottom: 4px;
  border: 1px solid rgba(96, 231, 255, 0.2);
  border-radius: 6px;
  color: var(--mini-cyber-accent, #22d3ee);
  background: rgba(34, 211, 238, 0.1);
  font-size: 11px;
  font-weight: 700;
  line-height: 1.4;
}

.mini-session-card-time {
  font-size: 11px;
  color: var(--mini-cyber-dim, rgba(143, 187, 204, 0.48));
  display: flex;
  align-items: center;
  gap: 6px;
}

.mini-session-card-timestamp {
  margin-top: 3px;
  font-size: 10px;
  color: var(--mini-cyber-dim, rgba(143, 187, 204, 0.44));
  font-family: 'SF Mono', 'Fira Code', monospace;
  letter-spacing: 0.02em;
}

.mini-session-status {
  color: var(--mini-cyber-accent, #22d3ee);
  font-size: 11px;
  font-weight: 700;
}

.mini-session-status.waiting {
  color: #f6bd4d;
}

.mini-session-status.output {
  color: #37a3ff;
}

.mini-session-status.done {
  color: #7df5c4;
}

.mini-session-status.cancelled {
  color: var(--mini-cyber-muted, rgba(184, 225, 235, 0.68));
}

.mini-session-status.failed {
  color: #ff6b6b;
}

.mini-session-empty {
  padding: 24px;
  text-align: center;
  color: var(--mini-cyber-dim, rgba(143, 187, 204, 0.48));
  font-size: 13px;
}

@keyframes miniSessionRunning {
  0%, 100% { box-shadow: inset 3px 0 0 rgba(34, 211, 238, 0.68), 0 0 10px rgba(34, 211, 238, 0.08); }
  50% { box-shadow: inset 3px 0 0 rgba(34, 211, 238, 1), 0 0 22px rgba(34, 211, 238, 0.2); }
}
</style>
