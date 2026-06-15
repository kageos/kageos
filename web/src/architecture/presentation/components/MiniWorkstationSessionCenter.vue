<template>
  <section
    :class="['mini-session-center', { 'is-open': open }]"
    :aria-hidden="open ? 'false' : 'true'"
    @click.self="$emit('close')"
  >
    <div class="mini-session-dialog" role="dialog" aria-modal="true" aria-label="工作台会话中心">
      <header class="mini-session-dialog-head">
        <div class="mini-session-dialog-title">
          <strong>工作台会话</strong>
          <span>左侧是当前目录会话，右侧是跨目录最近会话。</span>
        </div>
        <button type="button" class="mini-session-close" @click="$emit('close')">
          <el-icon><Close /></el-icon>
        </button>
      </header>
      <div class="mini-session-dialog-tools">
        <div class="mini-session-dialog-stat">
          <span>当前目录 {{ currentDirectorySessions.length }}/{{ currentDirectoryTotal }}</span>
          <span>最近会话 {{ recentSessions.length }}/{{ recentSourceTotal }}</span>
        </div>
        <label class="mini-session-search">
          <el-icon :size="14"><Search /></el-icon>
          <input v-model="searchKeywordModel" placeholder="搜索目录、函数或需求..." />
        </label>
        <div class="mini-session-filters">
          <button
            v-for="filter in sessionFilters"
            :key="filter.value"
            type="button"
            :class="{ active: sessionFilter === filter.value }"
            @click="$emit('update:sessionFilter', filter.value)"
          >
            {{ filter.label }}
          </button>
        </div>
      </div>
      <div class="mini-session-columns">
        <section class="mini-session-pane mini-session-pane--current" v-loading="loadingCurrent">
          <header class="mini-session-pane-head">
            <div>
              <strong>当前目录</strong>
              <span :title="fullCodePath">{{ directoryLabel }}</span>
            </div>
            <em>{{ currentDirectorySessions.length }}</em>
          </header>
          <div class="mini-session-list">
            <button
              v-for="item in currentDirectorySessions"
              :key="item.session_id"
              type="button"
              :class="['mini-session-row', getSessionStatusClass(item), { active: item.session_id === sessionId }]"
              @click="$emit('select', item)"
            >
              <span class="mini-status-dot" :class="getSessionStatusClass(item)"></span>
              <span class="mini-session-row-copy">
                <span class="mini-session-row-title">{{ getSessionTitle(item) }}</span>
                <span class="mini-session-row-sub">{{ getSessionCenterSubtitle(item) }}</span>
                <span v-if="getSessionContextBadges(item).length" class="mini-session-row-context">
                  <span v-for="badge in getSessionContextBadges(item)" :key="badge">{{ badge }}</span>
                </span>
              </span>
              <span class="mini-session-row-meta">{{ getSessionStatusLabel(item) }} · {{ formatRelativeTime(item.updated_at || item.created_at) }}</span>
              <span class="mini-session-open">打开</span>
            </button>
            <div v-if="currentDirectorySessions.length === 0 && !loadingCurrent" class="mini-session-empty">
              没有匹配的当前目录会话
            </div>
          </div>
        </section>

        <section class="mini-session-pane mini-session-pane--recent" v-loading="loadingRecent">
          <header class="mini-session-pane-head">
            <div>
              <strong>最近会话</strong>
              <span>可打开其他目录的工作台会话</span>
            </div>
            <em>{{ recentSessions.length }}</em>
          </header>
          <div class="mini-session-list">
            <button
              v-for="item in recentSessions"
              :key="item.session_id"
              type="button"
              :class="['mini-session-row', getSessionStatusClass(item), { active: item.session_id === sessionId }]"
              @click="$emit('select', item)"
            >
              <span class="mini-status-dot" :class="getSessionStatusClass(item)"></span>
              <span class="mini-session-row-copy">
                <span class="mini-session-row-title">{{ getSessionTitle(item) }}</span>
                <span class="mini-session-row-sub">{{ getSessionCenterSubtitle(item) }}</span>
                <span v-if="getSessionContextBadges(item).length" class="mini-session-row-context">
                  <span v-for="badge in getSessionContextBadges(item)" :key="badge">{{ badge }}</span>
                </span>
              </span>
              <span class="mini-session-row-meta">{{ getSessionStatusLabel(item) }} · {{ formatRelativeTime(item.updated_at || item.created_at) }}</span>
              <span class="mini-session-open">打开</span>
            </button>
            <div v-if="recentSessions.length === 0 && !loadingRecent" class="mini-session-empty">
              没有匹配的最近会话
            </div>
          </div>
        </section>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Close, Search } from '@element-plus/icons-vue'
import type { WorkspaceSessionItem } from '@/architecture/presentation/context/api/workspace'
import type { SessionFilterValue } from '../composables/useMiniWorkstationSessionView'

const props = defineProps<{
  open: boolean
  currentDirectorySessions: WorkspaceSessionItem[]
  recentSessions: WorkspaceSessionItem[]
  currentDirectoryTotal: number
  recentSourceTotal: number
  loadingCurrent: boolean
  loadingRecent: boolean
  fullCodePath: string
  directoryLabel: string
  sessionId?: string
  sessionFilters: Array<{ label: string; value: SessionFilterValue }>
  sessionSearchKeyword: string
  sessionFilter: SessionFilterValue
  formatRelativeTime: (value: string) => string
  getSessionStatusClass: (session: WorkspaceSessionItem) => string
  getSessionTitle: (session: WorkspaceSessionItem) => string
  getSessionCenterSubtitle: (session: WorkspaceSessionItem) => string
  getSessionStatusLabel: (session: WorkspaceSessionItem) => string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'select', session: WorkspaceSessionItem): void
  (e: 'update:sessionSearchKeyword', value: string): void
  (e: 'update:sessionFilter', value: SessionFilterValue): void
}>()

const searchKeywordModel = computed({
  get: () => props.sessionSearchKeyword,
  set: (value) => emit('update:sessionSearchKeyword', value)
})

function getSessionContextBadges(session: WorkspaceSessionItem): string[] {
  const badges: string[] = []
  if (session.archived_for_model) badges.push('历史兼容标记')
  const policy = formatSessionContextPolicy(session.context_policy)
  if (policy) badges.push(policy)
  if ((session.model_context_anchor_message_id || 0) > 0) badges.push('锚点已忽略')
  return badges
}

function formatSessionContextPolicy(policy?: string): string {
  const normalized = String(policy || '').trim()
  if (!normalized) return ''
  if (normalized === 'artifact_only') return '完整上下文 · 产物重点'
  if (normalized === 'display_only') return '完整上下文 · 展示标签'
  if (normalized === 'full') return '完整上下文'
  return normalized
}
</script>

<style scoped>
.mini-session-center {
  position: fixed;
  inset: 0;
  z-index: 42;
  display: grid;
  place-items: center;
  padding: 56px;
  background: rgba(2, 5, 11, 0.42);
  backdrop-filter: blur(7px);
  opacity: 0;
  visibility: hidden;
  pointer-events: none;
  transition: opacity 0.16s ease, visibility 0.16s ease;
}

.mini-session-center.is-open {
  opacity: 1;
  visibility: visible;
  pointer-events: auto;
}

.mini-session-dialog {
  width: min(1120px, calc(100vw - 96px));
  height: min(720px, calc(100vh - 96px));
  display: grid;
  grid-template-rows: auto auto 1fr;
  overflow: hidden;
  border: 1px solid rgba(130, 153, 190, 0.3);
  border-radius: 18px;
  background:
    linear-gradient(180deg, rgba(15, 23, 42, 0.94), rgba(8, 13, 24, 0.9)),
    rgba(10, 16, 29, 0.9);
  box-shadow: 0 34px 100px rgba(0, 0, 0, 0.52);
  color: var(--mini-cyber-text);
}

.mini-session-dialog-head {
  height: 66px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 18px 0 22px;
  border-bottom: 1px solid rgba(130, 153, 190, 0.18);
}

.mini-session-dialog-title {
  display: grid;
  gap: 4px;
}

.mini-session-dialog-title strong {
  font-size: 17px;
}

.mini-session-dialog-title span {
  color: var(--mini-cyber-muted);
  font-size: 12px;
}

.mini-session-close {
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border: 1px solid rgba(124, 146, 189, 0.24);
  border-radius: 10px;
  background: rgba(30, 42, 68, 0.72);
  color: #d7e5fa;
}

.mini-session-dialog-tools {
  display: grid;
  grid-template-columns: auto minmax(220px, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 14px 18px;
  border-bottom: 1px solid rgba(130, 153, 190, 0.14);
}

.mini-session-dialog-stat,
.mini-session-filters {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mini-session-dialog-stat span,
.mini-session-filters button {
  height: 34px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  border: 1px solid rgba(124, 146, 189, 0.22);
  border-radius: 10px;
  background: rgba(30, 42, 68, 0.56);
  color: #b9c9e4;
  font-size: 12px;
  white-space: nowrap;
}

.mini-session-dialog-stat span {
  border-color: rgba(43, 213, 159, 0.2);
  background: rgba(21, 54, 50, 0.26);
  color: #9ceccd;
}

.mini-session-filters button.active {
  border-color: rgba(83, 174, 255, 0.46);
  background: rgba(34, 113, 205, 0.2);
  color: #8ed0ff;
}

.mini-session-search {
  height: 36px;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  border: 1px solid rgba(124, 146, 189, 0.2);
  border-radius: 10px;
  background: rgba(10, 16, 29, 0.5);
  color: #8e9fbb;
}

.mini-session-search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  color: #e6f0ff;
  font: inherit;
  font-size: 13px;
}

.mini-session-search input::placeholder {
  color: #7586a4;
}

.mini-session-columns {
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(320px, 0.92fr) minmax(420px, 1.18fr);
  gap: 14px;
  padding: 14px 18px 18px;
}

.mini-session-pane {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid rgba(126, 151, 197, 0.18);
  border-radius: 14px;
  background: rgba(10, 16, 29, 0.34);
}

.mini-session-pane--current {
  background: rgba(13, 27, 45, 0.46);
}

.mini-session-pane-head {
  min-width: 0;
  height: 58px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 14px;
  border-bottom: 1px solid rgba(126, 151, 197, 0.14);
}

.mini-session-pane-head div {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.mini-session-pane-head strong,
.mini-session-pane-head span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-session-pane-head strong {
  color: #e6f0ff;
  font-size: 14px;
  font-weight: 850;
}

.mini-session-pane-head span {
  color: #8b9bb7;
  font-size: 12px;
}

.mini-session-pane-head em {
  min-width: 28px;
  height: 28px;
  display: inline-grid;
  place-items: center;
  border-radius: 9px;
  background: rgba(83, 174, 255, 0.14);
  color: #8ed0ff;
  font-size: 12px;
  font-style: normal;
  font-weight: 900;
}

.mini-session-list {
  overflow: auto;
  padding: 12px;
}

.mini-session-row {
  --mini-active-glow: rgba(55, 163, 255, 0.26);
  --mini-active-halo: rgba(55, 163, 255, 0.12);
  position: relative;
  width: 100%;
  min-height: 68px;
  display: grid;
  grid-template-columns: 10px minmax(0, 1fr) auto auto;
  gap: 12px;
  align-items: center;
  margin-bottom: 10px;
  padding: 12px;
  border: 1px solid rgba(126, 151, 197, 0.18);
  border-radius: 12px;
  background: rgba(17, 25, 45, 0.62);
  color: #d7e5fa;
  text-align: left;
  transition: background 0.18s ease, border-color 0.18s ease, box-shadow 0.22s ease;
}

.mini-session-row:last-child {
  margin-bottom: 0;
}

.mini-session-pane--current .mini-session-row {
  grid-template-columns: 10px minmax(0, 1fr) auto;
}

.mini-session-pane--current .mini-session-open {
  display: none;
}

.mini-session-row:hover {
  border-color: rgba(83, 174, 255, 0.42);
  background: rgba(24, 51, 83, 0.48);
}

.mini-status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--mini-cyber-accent);
  box-shadow: 0 0 16px rgba(55, 163, 255, 0.58);
}

.mini-status-dot.is-running {
  background: var(--mini-cyber-green);
  box-shadow: 0 0 16px rgba(43, 213, 159, 0.6);
}

.mini-status-dot.is-waiting {
  background: var(--mini-cyber-warm);
  box-shadow: 0 0 16px rgba(246, 189, 77, 0.6);
}

.mini-status-dot.is-done,
.mini-status-dot.is-cancelled {
  background: var(--mini-cyber-violet);
  box-shadow: 0 0 16px rgba(119, 107, 255, 0.55);
}

.mini-status-dot.is-failed {
  background: #ff6b6b;
  box-shadow: 0 0 16px rgba(255, 107, 107, 0.58);
}

.mini-status-dot.is-active,
.mini-status-dot.is-output {
  background: #37a3ff;
  box-shadow: 0 0 16px rgba(55, 163, 255, 0.58);
}

.mini-session-row.is-running {
  border-color: rgba(43, 213, 159, 0.28);
  background: rgba(21, 54, 50, 0.42);
  box-shadow: inset 3px 0 0 rgba(43, 213, 159, 0.74);
  --mini-active-glow: rgba(43, 213, 159, 0.34);
  --mini-active-halo: rgba(43, 213, 159, 0.16);
}

.mini-session-row.is-waiting {
  border-color: rgba(246, 189, 77, 0.3);
  background: rgba(58, 45, 24, 0.46);
  box-shadow: inset 3px 0 0 rgba(246, 189, 77, 0.72);
  --mini-active-glow: rgba(246, 189, 77, 0.34);
  --mini-active-halo: rgba(246, 189, 77, 0.16);
}

.mini-session-row.is-output {
  border-color: rgba(55, 163, 255, 0.3);
  background: rgba(24, 48, 77, 0.46);
  box-shadow: inset 3px 0 0 rgba(55, 163, 255, 0.72);
  --mini-active-glow: rgba(55, 163, 255, 0.34);
  --mini-active-halo: rgba(55, 163, 255, 0.16);
}

.mini-session-row.is-done {
  border-color: rgba(119, 107, 255, 0.28);
  background: rgba(41, 38, 76, 0.46);
  box-shadow: inset 3px 0 0 rgba(119, 107, 255, 0.7);
  --mini-active-glow: rgba(119, 107, 255, 0.34);
  --mini-active-halo: rgba(119, 107, 255, 0.16);
}

.mini-session-row.is-cancelled {
  border-color: rgba(142, 159, 187, 0.24);
  background: rgba(41, 48, 64, 0.46);
  box-shadow: inset 3px 0 0 rgba(142, 159, 187, 0.5);
  --mini-active-glow: rgba(142, 159, 187, 0.28);
  --mini-active-halo: rgba(142, 159, 187, 0.12);
}

.mini-session-row.is-failed {
  border-color: rgba(255, 108, 108, 0.34);
  background: rgba(74, 30, 38, 0.46);
  box-shadow: inset 3px 0 0 rgba(255, 107, 107, 0.72);
  --mini-active-glow: rgba(255, 107, 107, 0.34);
  --mini-active-halo: rgba(255, 107, 107, 0.16);
}

.mini-session-row.active {
  z-index: 1;
  box-shadow:
    0 0 14px 2px var(--mini-active-glow),
    0 0 38px 8px var(--mini-active-halo),
    0 12px 32px rgba(2, 5, 11, 0.22);
}

.mini-session-row-copy,
.mini-session-row-title,
.mini-session-row-sub {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-session-row-title {
  font-size: 14px;
  font-weight: 820;
}

.mini-session-row-sub {
  margin-top: 5px;
  color: #8798b5;
  font-size: 12px;
}

.mini-session-row-context {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 6px;
}

.mini-session-row-context span {
  height: 18px;
  display: inline-flex;
  align-items: center;
  padding: 0 6px;
  border: 1px solid rgba(83, 174, 255, 0.22);
  border-radius: 6px;
  background: rgba(37, 91, 145, 0.18);
  color: #8ed0ff;
  font-size: 11px;
  line-height: 18px;
}

.mini-session-row-meta {
  color: #9fb0cb;
  font-size: 12px;
  white-space: nowrap;
}

.mini-session-open {
  height: 32px;
  display: inline-flex;
  align-items: center;
  padding: 0 12px;
  border: 1px solid rgba(83, 174, 255, 0.32);
  border-radius: 9px;
  background: rgba(34, 113, 205, 0.18);
  color: #8ed0ff;
  font-size: 12px;
}

.mini-session-empty {
  padding: 46px 0;
  color: var(--mini-cyber-muted);
  text-align: center;
  font-size: 13px;
}

@media (max-width: 1180px) {
  .mini-session-dialog-tools,
  .mini-session-columns,
  .mini-session-row {
    grid-template-columns: 1fr;
  }

  .mini-session-columns {
    overflow: auto;
  }

  .mini-session-dialog-stat,
  .mini-session-filters {
    flex-wrap: wrap;
  }

  .mini-session-dialog {
    width: calc(100vw - 24px);
    height: calc(100vh - 72px);
  }
}
</style>
