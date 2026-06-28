<!--
  MiniWorkstationSessionPanel - 工作台「会话中心」抽屉
  从 MiniWorkstation.vue 抽离：会话列表、目录范围切换、搜索、状态筛选、当前目录上下文切换。
  纯展示组件：数据与展示用 helper 通过 props 传入，交互通过 emit 上抛（与同目录 Messages/Composer 子组件保持一致）。
-->
<template>
  <aside class="mini-current-meta">
    <header class="mini-current-session-head">
      <div>
        <strong>{{ t('miniWorkstation.sessionList') }}</strong>
        <span :title="fullCodePath">{{ dirLabel }}</span>
      </div>
      <em>{{ sessions.length }}</em>
    </header>
    <div v-if="hasDifferentContext" class="mini-current-context-switch">
      <span>{{ t('miniWorkstation.currentPage') }}</span>
      <strong :title="currentContextPath">{{ currentContextName }}</strong>
      <button type="button" @click="emit('context-new-session')">
        {{ t('miniWorkstation.currentDirectoryNewSession') }}
      </button>
    </div>
    <div class="mini-drawer-scope-tabs" role="tablist" :aria-label="t('miniWorkstation.sessionList')">
      <button
        type="button"
        :class="{ active: scope === 'current' }"
        @click="emit('scope-change', 'current')"
      >
        {{ t('miniWorkstation.currentDirectory') }}
      </button>
      <button
        type="button"
        :class="{ active: scope === 'all' }"
        @click="emit('scope-change', 'all')"
      >
        {{ t('miniWorkstation.allSessions') }}
      </button>
    </div>
    <label class="mini-drawer-session-search">
      <el-icon :size="14"><Search /></el-icon>
      <input
        :value="searchKeyword"
        :placeholder="t('miniWorkstation.searchSessionsPlaceholder')"
        @input="emit('update:searchKeyword', ($event.target as HTMLInputElement).value)"
      />
    </label>
    <div class="mini-drawer-session-filters">
      <button
        v-for="item in filters"
        :key="item.value"
        type="button"
        :class="{ active: filter === item.value }"
        @click="emit('update:filter', item.value)"
      >
        {{ item.label }}
      </button>
    </div>
    <div class="mini-current-session-list">
      <button
        v-for="item in sessions"
        :key="item.session_id"
        type="button"
        :class="['mini-current-session-row', getSessionStatusClass(item), { active: item.session_id === activeSessionId }]"
        :title="getSessionTitle(item)"
        @click="emit('select', item)"
      >
        <span class="mini-status-dot" :class="getSessionStatusClass(item)"></span>
        <span class="mini-current-session-copy">
          <span class="mini-current-session-title">{{ getSessionTitle(item) }}</span>
          <span class="mini-current-session-sub">
            {{ getSessionStatusLabel(item) }} · {{ formatRelativeTime(item.updated_at || item.created_at) }}
          </span>
        </span>
      </button>
      <button
        v-if="sessions.length === 0"
        type="button"
        class="mini-current-session-row active is-draft"
        @click="emit('new-session')"
      >
        <span class="mini-status-dot"></span>
        <span class="mini-current-session-copy">
          <span class="mini-current-session-title">
            {{ scope === 'current' ? t('miniWorkstation.noCurrentDirectorySessions') : t('miniWorkstation.noMatchingSessions') }}
          </span>
          <span class="mini-current-session-sub">{{ t('miniWorkstation.clickNewSession') }}</span>
        </span>
      </button>
    </div>
    <div v-if="queuedCount > 0" class="mini-queue-chip">{{ t('miniWorkstation.queuedCount', { count: queuedCount }) }}</div>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Search } from '@element-plus/icons-vue'
import type { WorkspaceSessionItem } from '@/architecture/presentation/context/api/workspace'
import type { SessionFilterValue } from '../composables/useMiniWorkstationSessionView'

defineProps<{
  fullCodePath: string
  dirLabel: string
  sessions: WorkspaceSessionItem[]
  activeSessionId: string | undefined
  scope: 'current' | 'all'
  searchKeyword: string
  filter: SessionFilterValue
  filters: Array<{ label: string; value: SessionFilterValue }>
  queuedCount: number
  hasDifferentContext: boolean
  currentContextName: string
  currentContextPath: string
  getSessionStatusClass: (item: WorkspaceSessionItem) => string
  getSessionTitle: (item: WorkspaceSessionItem) => string
  getSessionStatusLabel: (item: WorkspaceSessionItem) => string
  formatRelativeTime: (value: string) => string
}>()

const emit = defineEmits<{
  (e: 'update:searchKeyword', value: string): void
  (e: 'update:filter', value: SessionFilterValue): void
  (e: 'select', item: WorkspaceSessionItem): void
  (e: 'new-session'): void
  (e: 'scope-change', scope: 'current' | 'all'): void
  (e: 'context-new-session'): void
}>()

const { t } = useI18n()
</script>

<style scoped>
/* 基础布局（compact 区段） */
.mini-current-meta {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: auto auto auto auto auto minmax(0, 1fr) auto;
  gap: 12px;
  padding-right: 12px;
  border-right: 1px solid var(--border-light);
  color: var(--text-primary);
  font-size: 12px;
  overflow: hidden;
}

.mini-current-session-head {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 4px 4px;
}

.mini-current-session-head div {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.mini-current-session-head strong,
.mini-current-session-head span {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-current-session-head strong {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 600;
}

.mini-current-session-head span {
  color: var(--text-disabled);
  font-size: 12px;
}

.mini-current-session-head em {
  min-width: 24px;
  height: 24px;
  display: inline-grid;
  place-items: center;
  border-radius: 8px;
  background: var(--bg-secondary);
  color: var(--color-primary);
  font-size: 11px;
  font-style: normal;
  font-weight: 900;
}

.mini-current-session-list {
  min-height: 0;
  overflow: auto;
  display: grid;
  align-content: start;
  gap: 8px;
  padding: 1px 2px 3px 0;
  scrollbar-color: rgba(83, 174, 255, 0.24) transparent;
}

.mini-current-session-row {
  --mini-active-glow: rgba(55, 163, 255, 0.26);
  --mini-active-halo: rgba(55, 163, 255, 0.12);
  position: relative;
  width: 100%;
  min-width: 0;
  min-height: 52px;
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  padding: 10px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: var(--text-primary);
  text-align: left;
  cursor: pointer;
  transition: background 0.18s ease, border-color 0.18s ease, box-shadow 0.22s ease;
}

.mini-current-session-row:hover {
  background: var(--bg-tertiary);
}

.mini-current-session-row.is-running {
  background: transparent;
}

.mini-current-session-row.is-waiting {
  background: transparent;
}

.mini-current-session-row.is-output {
  background: transparent;
}

.mini-current-session-row.is-done {
  background: transparent;
}

.mini-current-session-row.is-cancelled {
  background: transparent;
  opacity: 0.5;
}

.mini-current-session-row.is-failed {
  background: transparent;
  border-left: 3px solid var(--color-danger);
  border-radius: 6px;
}

.mini-current-session-copy,
.mini-current-session-title,
.mini-current-session-sub {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-current-session-title {
  color: var(--text-primary);
  font-size: 13px;
  font-weight: 500;
  line-height: 1.2;
}

.mini-current-session-sub {
  margin-top: 4px;
  color: var(--text-secondary);
  font-size: 11px;
  line-height: 1.15;
}

.mini-queue-chip {
  width: fit-content;
  max-width: 100%;
  height: 22px;
  display: inline-flex;
  align-items: center;
  padding: 0 7px;
  border: 1px solid var(--border-light);
  border-radius: 999px;
  background: var(--bg-tertiary);
  color: var(--color-warning);
  font-size: 11px;
  font-weight: 600;
}

/* 会话状态高亮（active glow / halo） */
.mini-current-session-row.is-running {
  --mini-active-glow: rgba(43, 213, 159, 0.34);
  --mini-active-halo: rgba(43, 213, 159, 0.16);
}

.mini-current-session-row.is-waiting {
  --mini-active-glow: rgba(246, 189, 77, 0.34);
  --mini-active-halo: rgba(246, 189, 77, 0.16);
}

.mini-current-session-row.is-output {
  --mini-active-glow: rgba(55, 163, 255, 0.34);
  --mini-active-halo: rgba(55, 163, 255, 0.16);
}

.mini-current-session-row.is-done {
  --mini-active-glow: rgba(119, 107, 255, 0.34);
  --mini-active-halo: rgba(119, 107, 255, 0.16);
}

.mini-current-session-row.is-cancelled {
  --mini-active-glow: rgba(142, 159, 187, 0.28);
  --mini-active-halo: rgba(142, 159, 187, 0.12);
}

.mini-current-session-row.is-failed {
  --mini-active-glow: rgba(255, 107, 107, 0.34);
  --mini-active-halo: rgba(255, 107, 107, 0.16);
}

.mini-current-session-row.active {
  z-index: 1;
  background: var(--bg-tertiary);
  border-left: 3px solid var(--color-primary);
  border-radius: 6px;
}

.mini-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(var(--color-primary-rgb), 0.15);
}

.mini-status-dot.is-running {
  background: var(--color-success);
  box-shadow: 0 0 0 2px rgba(var(--color-success-rgb), 0.15);
}

.mini-status-dot.is-waiting {
  background: var(--color-warning);
  box-shadow: 0 0 0 2px rgba(var(--color-warning-rgb), 0.15);
}

.mini-status-dot.is-done,
.mini-status-dot.is-cancelled {
  background: var(--text-disabled);
  box-shadow: none;
}

.mini-status-dot.is-failed {
  background: var(--color-danger);
  box-shadow: 0 0 0 2px rgba(var(--color-danger-rgb), 0.15);
}

.mini-status-dot.is-active,
.mini-status-dot.is-output {
  background: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(var(--color-primary-rgb), 0.15);
}

/* 全屏 shell 区段覆盖（与 MiniWorkstation 源序一致，later wins） */
.mini-current-meta {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: auto auto auto auto auto minmax(0, 1fr) auto;
  gap: 10px;
  padding: 0 12px 0 0;
  border-right: 1px solid var(--border-light);
  overflow: hidden;
}

.mini-current-session-head {
  padding: 0;
}

.mini-current-session-head strong {
  color: #eef5ff;
}

.mini-current-context-switch {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 6px;
  padding: 9px 10px;
  border: 1px solid var(--border-light);
  border-radius: 8px;
  background: var(--bg-tertiary);
}

.mini-current-context-switch span {
  color: rgba(246, 217, 150, 0.72);
  font-size: 10px;
  font-weight: 760;
}

.mini-current-context-switch strong {
  min-width: 0;
  color: #ffe4a3;
  font-size: 12px;
  font-weight: 850;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-current-context-switch button {
  width: 100%;
  height: 28px;
  border: 1px solid var(--border-light);
  border-radius: 7px;
  background: var(--bg-tertiary);
  color: var(--color-primary);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.mini-current-context-switch button:hover {
  background: var(--el-fill-color-light);
  color: var(--color-primary-light-1);
}

.mini-drawer-scope-tabs,
.mini-drawer-session-filters {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.mini-drawer-scope-tabs button,
.mini-drawer-session-filters button {
  min-width: 0;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.mini-drawer-scope-tabs button:hover,
.mini-drawer-session-filters button:hover {
  background: var(--el-fill-color-light);
  color: var(--text-primary);
}

.mini-drawer-scope-tabs button {
  flex: 1 1 0;
}

.mini-drawer-session-filters {
  flex-wrap: wrap;
}

.mini-drawer-session-filters button {
  flex: 1 1 calc(50% - 4px);
}

.mini-drawer-scope-tabs button.active,
.mini-drawer-session-filters button.active {
  background: var(--bg-tertiary);
  color: var(--color-primary);
  font-weight: 600;
}

.mini-drawer-session-search {
  height: 32px;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  border: 1px solid var(--border-light);
  border-radius: 8px;
  background: transparent;
  color: var(--text-secondary);
}

.mini-drawer-session-search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  color: var(--text-primary);
  font-size: 12px;
}

.mini-drawer-session-search input::placeholder {
  color: var(--text-disabled);
}

.mini-current-session-list {
  padding: 0 2px 3px 0;
}

.mini-current-session-row {
  min-height: 48px;
  padding: 8px;
  border-radius: 8px;
}

/* 窄屏隐藏会话中心（与父级媒体查询一致） */
@media (max-width: 820px) {
  .mini-current-meta {
    display: none;
  }
}
</style>
