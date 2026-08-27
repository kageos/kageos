<!--
  MiniWorkstationSessionPanel - 工作台「会话中心」抽屉
  从 MiniWorkstation.vue 抽离：会话列表、服务目录范围切换、搜索、状态筛选、当前服务目录上下文切换。
  纯展示组件：数据与展示用 helper 通过 props 传入，交互通过 emit 上抛（与同服务目录 Messages/Composer 子组件保持一致）。
-->
<template>
  <aside class="mini-current-meta">
    <header class="mini-current-session-head">
      <div>
        <strong>{{ t('miniWorkstation.sessionList') }}</strong>
        <span :title="fullCodePath">{{ dirLabel }}</span>
      </div>
      <div class="mini-session-head-actions">
        <em>{{ sessions.length }}</em>
        <button
          type="button"
          :title="t('miniWorkstation.newSession')"
          :aria-label="t('miniWorkstation.newSession')"
          @click="emit('new-session')"
        >
          <el-icon :size="14"><Plus /></el-icon>
        </button>
      </div>
    </header>
    <div v-if="hasDifferentContext" class="mini-current-context-switch">
      <span>{{ t('miniWorkstation.currentPage') }}</span>
      <strong :title="currentContextPath">{{ currentContextName }}</strong>
      <button type="button" @click="emit('context-new-session')">
        {{ t('miniWorkstation.currentDirectoryNewSession') }}
      </button>
    </div>
    <div class="mini-session-toolbar">
      <div class="mini-drawer-scope-tabs" role="tablist" :aria-label="t('miniWorkstation.sessionList')">
        <button
          type="button"
          role="tab"
          :aria-selected="scope === 'current'"
          :class="{ active: scope === 'current' }"
          @click="emit('scope-change', 'current')"
        >
          {{ t('miniWorkstation.currentDirectoryShort') }}
        </button>
        <button
          v-if="!automationMode"
          type="button"
          role="tab"
          :aria-selected="scope === 'all'"
          :class="{ active: scope === 'all' }"
          @click="emit('scope-change', 'all')"
        >
          {{ t('miniWorkstation.allSessionsShort') }}
        </button>
      </div>
      <label v-if="automationAgents.length > 0 || automationMode" class="mini-session-source-filter">
        <el-icon :size="14"><MagicStick v-if="automationMode" /><User v-else /></el-icon>
        <select
          :value="sessionSourceFilter"
          :aria-label="t('miniWorkstation.sessionSource')"
          @change="emit('update:sessionSourceFilter', ($event.target as HTMLSelectElement).value)"
        >
          <option value="human">{{ t('miniWorkstation.humanSessions') }}</option>
          <option v-for="agent in automationAgents" :key="agent.task_id" :value="`agent:${agent.task_id}`">
            {{ agent.task_title }}
          </option>
        </select>
      </label>
    </div>
    <div class="mini-session-query-row">
      <label class="mini-drawer-session-search">
        <el-icon :size="14"><Search /></el-icon>
        <input
          :value="searchKeyword"
          :placeholder="t('miniWorkstation.searchSessionsPlaceholder')"
          @input="emit('update:searchKeyword', ($event.target as HTMLInputElement).value)"
        />
      </label>
      <label class="mini-session-status-filter">
        <span class="sr-only">{{ t('miniWorkstation.sessionStatusFilter') }}</span>
        <select
          :value="filter"
          :aria-label="t('miniWorkstation.sessionStatusFilter')"
          @change="emit('update:filter', ($event.target as HTMLSelectElement).value as SessionFilterValue)"
        >
          <option v-for="item in filters" :key="item.value" :value="item.value">{{ item.label }}</option>
        </select>
      </label>
    </div>
    <div v-if="activeSessionHidden && !loading && !loadFailed" class="mini-current-session-hidden">
      <span>{{ t('miniWorkstation.currentSessionHidden') }}</span>
      <button type="button" @click="emit('show-active')">{{ t('miniWorkstation.showCurrentSession') }}</button>
    </div>
    <div class="mini-current-session-list">
      <button
        v-for="item in loading || loadFailed ? [] : sessions"
        :key="item.session_id"
        type="button"
        :class="['mini-current-session-row', getSessionStatusClass(item), { active: item.session_id === activeSessionId }]"
        :title="getSessionTitle(item)"
        :aria-current="item.session_id === activeSessionId ? 'true' : undefined"
        @click="emit('select', item)"
      >
        <span class="mini-status-dot" :class="getSessionStatusClass(item)"></span>
        <span class="mini-current-session-copy">
          <span class="mini-current-session-title">{{ getSessionTitle(item) }}</span>
          <span v-if="scope === 'all'" class="mini-current-session-directory" :title="item.full_code_path">
            {{ getSessionDirectoryPath(item) }}
          </span>
          <span v-if="item.source === 'automation_agent'" class="mini-current-session-agent">
            <el-icon :size="11"><MagicStick /></el-icon>
            {{ item.automation_task_title || t('miniWorkstation.automationAgent') }}
          </span>
          <span class="mini-current-session-sub">
            <span :class="['mini-session-status-label', getSessionStatusClass(item)]">{{ getSessionStatusLabel(item) }}</span>
            <time>{{ formatRelativeTime(item.updated_at || item.created_at) }}</time>
          </span>
        </span>
      </button>
      <div v-if="loading" class="mini-session-state" role="status">
        <span class="mini-session-loading-dot" aria-hidden="true"></span>
        <strong>{{ t('miniWorkstation.sessionLoading') }}</strong>
      </div>
      <div v-else-if="loadFailed" class="mini-session-state is-error" role="alert">
        <strong>{{ t('miniWorkstation.sessionLoadFailed') }}</strong>
        <button type="button" @click="emit('retry')">{{ t('miniWorkstation.retry') }}</button>
      </div>
      <button
        v-else-if="sessions.length === 0"
        type="button"
        class="mini-current-session-row active is-draft"
        @click="hasActiveFilters ? emit('reset-filters') : emit('new-session')"
      >
        <span class="mini-status-dot"></span>
        <span class="mini-current-session-copy">
          <span class="mini-current-session-title">
            {{ hasActiveFilters
              ? t('miniWorkstation.noMatchingSessions')
              : scope === 'current'
                ? t('miniWorkstation.noCurrentDirectorySessions')
                : t('miniWorkstation.noSessions') }}
          </span>
          <span class="mini-current-session-sub">
            {{ hasActiveFilters ? t('miniWorkstation.clearFilters') : t('miniWorkstation.clickNewSession') }}
          </span>
        </span>
      </button>
    </div>
    <div v-if="queuedCount > 0" class="mini-queue-chip">{{ t('miniWorkstation.queuedCount', { count: queuedCount }) }}</div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { MagicStick, Plus, Search, User } from '@element-plus/icons-vue'
import type { WorkspaceAutomationAgentItem, WorkspaceSessionItem } from '@/architecture/presentation/context/api/workspace'
import type { SessionFilterValue } from '../composables/useMiniWorkstationSessionView'

const props = defineProps<{
  fullCodePath: string
  dirLabel: string
  sessions: WorkspaceSessionItem[]
  activeSessionId: string | undefined
  scope: 'current' | 'all'
  sessionSourceFilter: string
  automationAgents: WorkspaceAutomationAgentItem[]
  automationMode: boolean
  searchKeyword: string
  filter: SessionFilterValue
  filters: Array<{ label: string; value: SessionFilterValue }>
  queuedCount: number
  loading: boolean
  loadFailed: boolean
  hasDifferentContext: boolean
  currentContextName: string
  currentContextPath: string
  getSessionStatusClass: (item: WorkspaceSessionItem) => string
  getSessionTitle: (item: WorkspaceSessionItem) => string
  getSessionStatusLabel: (item: WorkspaceSessionItem) => string
  getSessionDirectoryPath: (item: WorkspaceSessionItem) => string
  formatRelativeTime: (value: string) => string
}>()

const emit = defineEmits<{
  (e: 'update:searchKeyword', value: string): void
  (e: 'update:filter', value: SessionFilterValue): void
  (e: 'select', item: WorkspaceSessionItem): void
  (e: 'new-session'): void
  (e: 'scope-change', scope: 'current' | 'all'): void
  (e: 'update:sessionSourceFilter', value: string): void
  (e: 'context-new-session'): void
  (e: 'retry'): void
  (e: 'reset-filters'): void
  (e: 'show-active'): void
}>()

const { t } = useI18n()
const hasActiveFilters = computed(() => Boolean(props.searchKeyword.trim()) || props.filter !== 'all')
const activeSessionHidden = computed(() => Boolean(props.activeSessionId)
  && !props.sessions.some(session => session.session_id === props.activeSessionId))
</script>

<style scoped>
/* 基础布局（compact 区段） */
.mini-current-meta {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: auto auto auto auto auto minmax(0, 1fr) auto;
  gap: 8px;
  padding-right: 10px;
  border-right: 1px solid var(--border-light);
  color: var(--text-primary);
  font-size: 12px;
  overflow: hidden;
}

.mini-session-source-filter {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 2px;
  color: var(--text-secondary);
}

.mini-session-toolbar {
  display: grid;
  gap: 6px;
  padding: 4px;
  border: 1px solid rgba(var(--color-primary-rgb), 0.1);
  border-radius: 9px;
  background: rgba(var(--color-primary-rgb), 0.035);
}

.mini-session-source-filter select {
  min-width: 0;
  width: 100%;
  height: 30px;
  padding: 0 8px;
  border: 1px solid var(--border-light);
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--text-primary);
}

.mini-current-session-agent {
  margin-top: 3px;
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--color-primary);
  font-size: 10px;
}

.mini-current-session-head {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 4px 2px 8px;
  border-bottom: 1px solid rgba(var(--color-primary-rgb), 0.1);
}

.mini-current-session-head div {
  min-width: 0;
  display: grid;
  gap: 3px;
}

.mini-current-session-head .mini-session-head-actions {
  flex: 0 0 auto;
  display: flex;
  grid-auto-flow: unset;
  align-items: center;
  gap: 6px;
}

.mini-session-head-actions button {
  width: 26px;
  height: 26px;
  display: inline-grid;
  place-items: center;
  padding: 0;
  border: 1px solid rgba(var(--color-primary-rgb), 0.18);
  border-radius: 8px;
  background: rgba(var(--color-primary-rgb), 0.08);
  color: var(--color-primary);
  cursor: pointer;
}

.mini-session-head-actions button:hover {
  background: rgba(var(--color-primary-rgb), 0.16);
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
  border: 1px solid rgba(var(--color-primary-rgb), 0.16);
  border-radius: 8px;
  background: rgba(var(--color-primary-rgb), 0.08);
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
  gap: 6px;
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
  gap: 10px;
  padding: 9px;
  border: 1px solid rgba(var(--color-primary-rgb), 0.08);
  border-radius: 10px;
  background: rgba(var(--color-primary-rgb), 0.025);
  color: var(--text-primary);
  text-align: left;
  cursor: pointer;
  transition: background 0.18s ease, border-color 0.18s ease, box-shadow 0.22s ease;
}

.mini-current-session-row:hover {
  border-color: rgba(var(--color-primary-rgb), 0.2);
  background: rgba(var(--color-primary-rgb), 0.07);
}

.mini-current-session-row.is-cancelled {
  opacity: 0.5;
}

.mini-current-session-row.is-failed {
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
  font-weight: 650;
  line-height: 1.2;
}

.mini-current-session-directory {
  min-width: 0;
  margin-top: 3px;
  display: block;
  overflow: hidden;
  color: var(--text-secondary);
  font-size: 11px;
  line-height: 1.15;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-current-session-sub {
  margin-top: 4px;
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-secondary);
  font-size: 11px;
  line-height: 1.15;
}

.mini-current-session-sub time {
  min-width: 0;
  overflow: hidden;
  color: var(--text-disabled);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-session-status-label {
  flex: 0 0 auto;
  padding: 2px 6px;
  border-radius: 999px;
  background: var(--el-fill-color);
  color: var(--text-secondary);
  font-size: 10px;
  font-weight: 700;
}

.mini-session-status-label.is-running { background: rgba(var(--color-success-rgb), 0.12); color: var(--color-success); }
.mini-session-status-label.is-waiting { background: rgba(var(--color-warning-rgb), 0.12); color: var(--color-warning); }
.mini-session-status-label.is-failed { background: rgba(var(--color-danger-rgb), 0.12); color: var(--color-danger); }
.mini-session-status-label.is-output,
.mini-session-status-label.is-active { background: rgba(var(--color-primary-rgb), 0.12); color: var(--color-primary); }

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
  border-color: rgba(var(--color-primary-rgb), 0.32);
  background: linear-gradient(90deg, rgba(var(--color-primary-rgb), 0.14), rgba(var(--color-primary-rgb), 0.06));
  box-shadow: inset 0 0 0 1px rgba(var(--color-primary-rgb), 0.04);
}

.mini-current-session-row.active::before {
  content: '';
  position: absolute;
  top: 9px;
  bottom: 9px;
  left: 0;
  width: 3px;
  border-radius: 0 999px 999px 0;
  background: var(--color-primary);
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
  gap: 8px;
  padding: 0 10px 0 0;
  border-right: 1px solid var(--border-light);
  overflow: hidden;
}

.mini-current-session-head {
  padding: 4px 2px 8px;
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
  border: 1px solid rgba(var(--color-primary-rgb), 0.14);
  border-radius: 8px;
  background: rgba(var(--color-primary-rgb), 0.06);
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
  border: 1px solid transparent;
  border-radius: 7px;
  background: var(--el-fill-color-light);
  color: var(--color-primary);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.mini-current-context-switch button:hover {
  border-color: rgba(var(--color-primary-rgb), 0.18);
  background: var(--el-fill-color);
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
  border-color: rgba(var(--color-primary-rgb), 0.14);
  background: rgba(var(--color-primary-rgb), 0.06);
  color: var(--color-primary);
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
  border-color: rgba(var(--color-primary-rgb), 0.2);
  background: rgba(var(--color-primary-rgb), 0.1);
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
  border: 1px solid transparent;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--text-secondary);
  transition: background-color 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
}

.mini-drawer-session-search:hover {
  border-color: var(--border-light);
  background: var(--el-fill-color);
}

.mini-drawer-session-search:focus-within {
  border-color: var(--color-primary);
  background: var(--el-bg-color);
  box-shadow: none;
  color: var(--color-primary);
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

.mini-session-query-row {
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 82px;
  gap: 6px;
}

.mini-session-status-filter,
.mini-session-status-filter select {
  min-width: 0;
  width: 100%;
}

.mini-session-status-filter select {
  height: 32px;
  padding: 0 7px;
  border: 1px solid transparent;
  border-radius: 8px;
  outline: none;
  background: var(--el-fill-color-light);
  color: var(--text-primary);
  font-size: 11px;
  cursor: pointer;
}

.mini-session-status-filter select:hover,
.mini-session-status-filter select:focus-visible {
  border-color: var(--color-primary);
  background: var(--el-bg-color);
}

.mini-current-session-hidden {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 7px 8px;
  border: 1px solid rgba(var(--color-warning-rgb), 0.2);
  border-radius: 8px;
  background: rgba(var(--color-warning-rgb), 0.08);
  color: var(--text-secondary);
  font-size: 11px;
}

.mini-current-session-hidden button,
.mini-session-state button {
  flex: 0 0 auto;
  border: 0;
  background: transparent;
  color: var(--color-primary);
  font-size: 11px;
  font-weight: 700;
  cursor: pointer;
}

.mini-session-state {
  min-height: 96px;
  display: grid;
  place-content: center;
  justify-items: center;
  gap: 8px;
  padding: 16px 10px;
  border: 1px dashed var(--border-light);
  border-radius: 9px;
  color: var(--text-secondary);
  text-align: center;
}

.mini-session-state strong {
  font-size: 12px;
  font-weight: 600;
}

.mini-session-state.is-error {
  border-color: rgba(var(--color-danger-rgb), 0.24);
  color: var(--color-danger);
}

.mini-session-loading-dot {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(var(--color-primary-rgb), 0.18);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: mini-session-spin 0.8s linear infinite;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

@keyframes mini-session-spin {
  to { transform: rotate(360deg); }
}

.mini-current-session-list {
  padding: 0 2px 3px 0;
}

.mini-current-session-row {
  min-height: 52px;
  padding: 9px;
  border-radius: 8px;
}

</style>
