<template>
  <section class="mini-session-dock" :aria-label="t('miniWorkstation.sessionSummary')">
    <button type="button" class="mini-session-center-btn" @click="$emit('openCenter')">
      <span class="mini-count-badge">{{ centerCount }}</span>
      <span>{{ t('miniWorkstation.sessionCenter') }}</span>
    </button>
    <button type="button" class="mini-session-new-btn" :title="t('miniWorkstation.newSession')" @click="$emit('newSession')">
      <el-icon :size="17"><Plus /></el-icon>
    </button>
    <div class="mini-session-summary-list">
      <button
        v-if="summarySessions.length === 0"
        type="button"
        class="mini-session-summary-card active is-draft"
        @click="$emit('newSession')"
      >
        <span class="mini-status-dot"></span>
        <span class="mini-session-summary-copy">
          <span class="mini-session-summary-title">{{ t('miniWorkstation.newSession') }}</span>
          <span class="mini-session-summary-sub">{{ directoryLabel }}</span>
        </span>
      </button>
      <button
        v-for="item in summarySessions"
        :key="item.session_id"
        type="button"
        :class="['mini-session-summary-card', getSessionStatusClass(item), { active: item.session_id === sessionId }]"
        :title="getSessionTitle(item)"
        @click="$emit('select', item)"
      >
        <span class="mini-status-dot" :class="getSessionStatusClass(item)"></span>
        <span class="mini-session-summary-copy">
          <span class="mini-session-summary-title">{{ getSessionTitle(item) }}</span>
          <span class="mini-session-summary-sub">{{ getSessionSubtitle(item) }}</span>
        </span>
        <span v-if="getSessionStatusKind(item) === 'running'" class="mini-count-badge">•</span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Plus } from '@element-plus/icons-vue'
import type { WorkspaceSessionItem } from '@/architecture/presentation/context/api/workspace'

defineProps<{
  summarySessions: WorkspaceSessionItem[]
  centerCount: number
  directoryLabel: string
  sessionId?: string
  getSessionStatusClass: (session: WorkspaceSessionItem) => string
  getSessionStatusKind: (session: WorkspaceSessionItem) => string
  getSessionTitle: (session: WorkspaceSessionItem) => string
  getSessionSubtitle: (session: WorkspaceSessionItem) => string
}>()

defineEmits<{
  (e: 'openCenter'): void
  (e: 'newSession'): void
  (e: 'select', session: WorkspaceSessionItem): void
}>()

const { t } = useI18n()
</script>

<style scoped>
.mini-session-dock {
  min-height: 54px;
  position: relative;
  display: block;
  margin: 0 14px 8px 204px;
  padding: 6px;
  border: 1px solid var(--mini-cyber-line);
  border-radius: 14px;
  background:
    linear-gradient(180deg, rgba(12, 18, 32, 0.84), rgba(8, 12, 22, 0.68)),
    rgba(8, 12, 22, 0.72);
  box-shadow: 0 24px 70px rgba(0, 0, 0, 0.42), inset 0 1px 0 rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(24px) saturate(140%);
}

.mini-session-center-btn {
  position: absolute;
  left: -146px;
  top: 6px;
  width: 132px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 0 12px;
  border: 1px solid rgba(83, 174, 255, 0.44);
  border-radius: 12px;
  background:
    linear-gradient(135deg, rgba(34, 113, 205, 0.34), rgba(119, 107, 255, 0.18)),
    rgba(12, 22, 38, 0.78);
  color: #dff1ff;
  box-shadow: 0 12px 30px rgba(37, 110, 194, 0.2);
  font-size: 12px;
  font-weight: 800;
}

.mini-session-new-btn {
  position: absolute;
  left: -194px;
  top: 6px;
  width: 40px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(43, 213, 159, 0.42);
  border-radius: 12px;
  background:
    linear-gradient(135deg, rgba(43, 213, 159, 0.22), rgba(55, 163, 255, 0.12)),
    rgba(12, 22, 38, 0.78);
  color: #8dffd8;
  box-shadow: 0 12px 30px rgba(43, 213, 159, 0.14);
  cursor: pointer;
}

.mini-session-new-btn:hover {
  border-color: rgba(43, 213, 159, 0.62);
  background:
    linear-gradient(135deg, rgba(43, 213, 159, 0.32), rgba(55, 163, 255, 0.18)),
    rgba(12, 22, 38, 0.9);
  color: #ffffff;
}

.mini-session-summary-list {
  min-width: 0;
  display: grid;
  grid-template-columns: repeat(4, minmax(130px, 1fr));
  gap: 8px;
}

.mini-session-summary-card {
  --mini-active-glow: rgba(55, 163, 255, 0.26);
  --mini-active-halo: rgba(55, 163, 255, 0.12);
  --mini-active-arrow-color: #8ed0ff;
  --mini-active-arrow-shadow: rgba(55, 163, 255, 0.72);
  position: relative;
  width: 100%;
  height: 42px;
  min-width: 0;
  display: grid;
  grid-template-columns: 10px minmax(0, 1fr) auto;
  align-items: center;
  gap: 7px;
  padding: 0 10px;
  border: 1px solid rgba(126, 151, 197, 0.2);
  border-radius: 10px;
  background: rgba(30, 42, 68, 0.5);
  color: #d7e5fa;
  text-align: left;
  transition: background 0.18s ease, border-color 0.18s ease, box-shadow 0.22s ease;
}

.mini-session-summary-card:hover {
  border-color: rgba(87, 182, 255, 0.5);
  background: rgba(24, 51, 83, 0.62);
}

.mini-session-summary-card::before {
  content: "▼";
  position: absolute;
  left: 50%;
  top: -20px;
  transform: translateX(-50%) translateY(3px);
  opacity: 0;
  color: var(--mini-active-arrow-color);
  font-size: 16px;
  line-height: 1;
  text-shadow: 0 0 16px var(--mini-active-arrow-shadow);
  pointer-events: none;
  transition: opacity 0.18s ease, transform 0.18s ease, color 0.18s ease, text-shadow 0.18s ease;
}

.mini-session-summary-card.active::before {
  opacity: 1;
  transform: translateX(-50%) translateY(0);
}

.mini-session-summary-card.is-running {
  border-color: rgba(43, 213, 159, 0.28);
  background: rgba(21, 54, 50, 0.42);
  --mini-active-glow: rgba(43, 213, 159, 0.34);
  --mini-active-halo: rgba(43, 213, 159, 0.16);
}

.mini-session-summary-card.is-waiting {
  border-color: rgba(246, 189, 77, 0.3);
  background: rgba(58, 45, 24, 0.46);
  --mini-active-glow: rgba(246, 189, 77, 0.34);
  --mini-active-halo: rgba(246, 189, 77, 0.16);
}

.mini-session-summary-card.is-done {
  border-color: rgba(119, 107, 255, 0.28);
  background: rgba(41, 38, 76, 0.46);
  --mini-active-glow: rgba(119, 107, 255, 0.34);
  --mini-active-halo: rgba(119, 107, 255, 0.16);
}

.mini-session-summary-card.is-output {
  border-color: rgba(55, 163, 255, 0.3);
  background: rgba(24, 48, 77, 0.46);
  --mini-active-glow: rgba(55, 163, 255, 0.34);
  --mini-active-halo: rgba(55, 163, 255, 0.16);
}

.mini-session-summary-card.is-active {
  border-color: rgba(126, 151, 197, 0.24);
  background: rgba(30, 42, 68, 0.5);
}

.mini-session-summary-card.is-cancelled {
  border-color: rgba(142, 159, 187, 0.24);
  background: rgba(41, 48, 64, 0.46);
  --mini-active-glow: rgba(142, 159, 187, 0.28);
  --mini-active-halo: rgba(142, 159, 187, 0.12);
}

.mini-session-summary-card.is-failed {
  border-color: rgba(255, 108, 108, 0.34);
  background: rgba(74, 30, 38, 0.46);
  --mini-active-glow: rgba(255, 107, 107, 0.34);
  --mini-active-halo: rgba(255, 107, 107, 0.16);
}

.mini-session-summary-card.active {
  z-index: 1;
  box-shadow:
    0 0 14px 2px var(--mini-active-glow),
    0 0 38px 8px var(--mini-active-halo),
    0 12px 32px rgba(2, 5, 11, 0.22);
}

.mini-session-summary-card.active.is-running::before {
  --mini-active-arrow-color: #7df5c4;
  --mini-active-arrow-shadow: rgba(43, 213, 159, 0.72);
}

.mini-session-summary-card.active.is-waiting::before {
  --mini-active-arrow-color: #ffd78d;
  --mini-active-arrow-shadow: rgba(246, 189, 77, 0.72);
}

.mini-session-summary-card.active.is-output::before {
  --mini-active-arrow-color: #8ed0ff;
  --mini-active-arrow-shadow: rgba(55, 163, 255, 0.72);
}

.mini-session-summary-card.active.is-done::before {
  --mini-active-arrow-color: #bcb7ff;
  --mini-active-arrow-shadow: rgba(119, 107, 255, 0.72);
}

.mini-session-summary-card.active.is-failed::before {
  --mini-active-arrow-color: #ff9ba4;
  --mini-active-arrow-shadow: rgba(255, 107, 107, 0.72);
}

.mini-session-summary-copy,
.mini-session-summary-title,
.mini-session-summary-sub {
  min-width: 0;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-session-summary-title {
  color: #d7e5fa;
  font-size: 12px;
  font-weight: 780;
  line-height: 1.15;
}

.mini-session-summary-sub {
  margin-top: 2px;
  color: #8596b2;
  font-size: 10px;
  line-height: 1.1;
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

.mini-count-badge {
  min-width: 18px;
  height: 18px;
  display: inline-grid;
  place-items: center;
  padding: 0 5px;
  border-radius: 999px;
  background: rgba(255, 109, 126, 0.9);
  color: #fff;
  font-size: 11px;
  font-weight: 900;
}

@media (max-width: 1180px) {
  .mini-session-dock {
    margin-left: 184px;
  }

  .mini-session-center-btn {
    left: -130px;
    width: 118px;
  }

  .mini-session-new-btn {
    left: -176px;
    width: 38px;
  }

  .mini-session-summary-list {
    grid-template-columns: repeat(4, minmax(112px, 1fr));
  }
}

@media (max-width: 820px) {
  .mini-session-dock {
    display: grid;
    grid-template-columns: 1fr;
    margin: 0 14px 8px;
  }

  .mini-session-center-btn {
    position: static;
    width: 100%;
    justify-content: flex-start;
  }

  .mini-session-new-btn {
    position: static;
    width: 100%;
    justify-content: center;
  }

  .mini-session-summary-list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
