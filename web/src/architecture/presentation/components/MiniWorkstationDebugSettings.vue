<template>
  <div class="mini-settings-panel" @mousedown.stop @click.stop>
    <section class="mini-settings-section">
      <div class="mini-settings-section-title">{{ t('miniWorkstation.copy') }}</div>
      <div class="mini-settings-copy-grid">
        <button type="button" @click="$emit('copyConversation', 'all')">{{ t('miniWorkstation.copyAllConversation') }}</button>
        <button type="button" @click="$emit('copyConversation', 'last-turn')">{{ t('miniWorkstation.copyLastTurn') }}</button>
        <button type="button" @click="$emit('copyConversation', 'all-tools')">{{ t('miniWorkstation.copyAllTools') }}</button>
        <button type="button" @click="$emit('copyConversation', 'error-tools')">{{ t('miniWorkstation.copyErrorTools') }}</button>
        <button type="button" @click="$emit('copyConversation', 'success-tools')">{{ t('miniWorkstation.copySuccessTools') }}</button>
      </div>
    </section>
    <section class="mini-settings-section">
      <header class="mini-debug-head">
        <div>
          <span class="mini-debug-kicker">{{ t('miniWorkstation.toolTrace') }}</span>
          <strong>{{ t('miniWorkstation.callSummary') }}</strong>
        </div>
        <button
          type="button"
          class="mini-debug-copy-btn"
          :disabled="debugToolSteps.length === 0"
          @click="$emit('copyToolSummary')"
        >
          {{ t('miniWorkstation.copySummary') }}
        </button>
      </header>
      <div class="mini-debug-stats">
        <span>{{ t('miniWorkstation.stepCount', { count: debugToolSteps.length }) }}</span>
        <span>{{ t('miniWorkstation.successCount', { count: debugSuccessCount }) }}</span>
        <span>{{ t('miniWorkstation.errorCount', { count: debugErrorCount }) }}</span>
      </div>
      <div v-if="debugToolSteps.length" class="mini-debug-list">
        <article
          v-for="step in debugToolSteps"
          :key="step.key"
          class="mini-debug-step"
          :class="`is-${step.statusClass}`"
        >
          <div class="mini-debug-step-title">
            <span>{{ t('miniWorkstation.step', { count: step.index }) }}</span>
            <strong>{{ step.name }}</strong>
            <em>{{ step.statusLabel }}</em>
          </div>
          <pre v-if="step.argumentsPreview" class="mini-debug-snippet">{{ t('miniWorkstation.arguments') }}: {{ step.argumentsPreview }}</pre>
          <pre v-if="step.outputPreview" class="mini-debug-snippet">{{ t('miniWorkstation.output') }}: {{ step.outputPreview }}</pre>
          <pre v-if="step.errorPreview" class="mini-debug-snippet mini-debug-snippet--error">{{ t('miniWorkstation.error') }}: {{ step.errorPreview }}</pre>
        </article>
      </div>
      <div v-else class="mini-debug-empty">{{ t('miniWorkstation.noToolCalls') }}</div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type {
  CopyDebugMode,
  DebugToolStep
} from '../composables/useMiniWorkstationDebugCopy'

defineProps<{
  debugToolSteps: DebugToolStep[]
  debugSuccessCount: number
  debugErrorCount: number
}>()

defineEmits<{
  (e: 'copyConversation', mode: CopyDebugMode): void
  (e: 'copyToolSummary'): void
}>()

const { t } = useI18n()
</script>

<style scoped>
:global(.mini-settings-popover.el-popper) {
  padding: 0;
  border: 1px solid rgba(96, 231, 255, 0.22);
  border-radius: 14px;
  background:
    radial-gradient(circle at 14% 0%, rgba(34, 211, 238, 0.16), transparent 36%),
    linear-gradient(150deg, rgba(5, 16, 30, 0.98), rgba(8, 27, 45, 0.97));
  box-shadow: 0 20px 54px rgba(0, 0, 0, 0.42), 0 0 28px rgba(34, 211, 238, 0.14);
  backdrop-filter: blur(18px) saturate(1.16);
}

:global(.mini-settings-popover .el-popper__arrow::before) {
  border-color: rgba(96, 231, 255, 0.22);
  background: rgba(5, 16, 30, 0.98);
}

.mini-settings-panel {
  max-height: min(520px, calc(100vh - 110px));
  overflow: auto;
  color: var(--mini-cyber-text);
}

.mini-settings-section + .mini-settings-section {
  border-top: 1px solid rgba(96, 231, 255, 0.14);
}

.mini-settings-section-title {
  padding: 12px 12px 0;
  color: var(--mini-cyber-accent);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.mini-settings-copy-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  padding: 10px 12px 12px;
}

.mini-settings-copy-grid button {
  min-width: 0;
  height: 32px;
  border: 1px solid rgba(96, 231, 255, 0.18);
  border-radius: 9px;
  background: rgba(34, 211, 238, 0.08);
  color: var(--mini-cyber-muted);
  cursor: pointer;
  font-size: 12px;
}

.mini-settings-copy-grid button:hover {
  border-color: rgba(96, 231, 255, 0.34);
  background: rgba(34, 211, 238, 0.14);
  color: #ffffff;
  box-shadow: inset 0 0 0 1px rgba(34, 211, 238, 0.16);
}

.mini-debug-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  border-bottom: 1px solid rgba(96, 231, 255, 0.16);
  background: rgba(34, 211, 238, 0.055);
}

.mini-debug-head strong {
  display: block;
  margin-top: 3px;
  font-size: 14px;
}

.mini-debug-kicker {
  display: block;
  color: var(--mini-cyber-accent);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 10px;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.mini-debug-copy-btn {
  height: 28px;
  padding: 0 10px;
  border: 1px solid rgba(96, 231, 255, 0.26);
  border-radius: 9px;
  background: rgba(34, 211, 238, 0.11);
  color: var(--mini-cyber-text);
  font-size: 12px;
  cursor: pointer;
}

.mini-debug-copy-btn:hover:not(:disabled) {
  border-color: rgba(96, 231, 255, 0.46);
  background: rgba(34, 211, 238, 0.17);
}

.mini-debug-copy-btn:disabled {
  cursor: not-allowed;
  opacity: 0.48;
}

.mini-debug-stats {
  display: flex;
  gap: 7px;
  padding: 9px 12px;
  border-bottom: 1px solid rgba(96, 231, 255, 0.12);
}

.mini-debug-stats span {
  padding: 3px 7px;
  border: 1px solid rgba(96, 231, 255, 0.14);
  border-radius: 999px;
  background: rgba(34, 211, 238, 0.06);
  color: var(--mini-cyber-muted);
  font-size: 11px;
}

.mini-debug-list {
  max-height: 390px;
  overflow: auto;
  padding: 10px 12px 12px;
}

.mini-debug-list::-webkit-scrollbar {
  width: 7px;
}

.mini-debug-list::-webkit-scrollbar-thumb {
  border: 2px solid transparent;
  border-radius: 999px;
  background: rgba(96, 231, 255, 0.26);
  background-clip: padding-box;
}

.mini-debug-step {
  padding: 9px;
  border: 1px solid rgba(96, 231, 255, 0.14);
  border-radius: 12px;
  background:
    linear-gradient(145deg, rgba(8, 22, 38, 0.7), rgba(3, 10, 20, 0.62)),
    rgba(34, 211, 238, 0.04);
}

.mini-debug-step + .mini-debug-step {
  margin-top: 8px;
}

.mini-debug-step.is-ok {
  border-color: rgba(103, 194, 58, 0.28);
}

.mini-debug-step.is-error {
  border-color: rgba(245, 108, 108, 0.34);
}

.mini-debug-step.is-running {
  border-color: rgba(230, 162, 60, 0.3);
}

.mini-debug-step-title {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 7px;
  margin-bottom: 7px;
}

.mini-debug-step-title span {
  flex: 0 0 auto;
  color: var(--mini-cyber-dim);
  font-size: 11px;
}

.mini-debug-step-title strong {
  min-width: 0;
  overflow: hidden;
  color: var(--mini-cyber-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-debug-step-title em {
  flex: 0 0 auto;
  margin-left: auto;
  color: var(--mini-cyber-accent);
  font-size: 11px;
  font-style: normal;
}

.mini-debug-snippet {
  max-height: 190px;
  margin: 6px 0 0;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  padding: 7px 8px;
  border: 1px solid rgba(96, 231, 255, 0.12);
  border-radius: 9px;
  background: rgba(2, 8, 18, 0.46);
  color: var(--mini-cyber-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  line-height: 1.45;
}

.mini-debug-snippet--error {
  border-color: rgba(245, 108, 108, 0.24);
  color: #ffc7c7;
}

.mini-debug-empty {
  padding: 42px 16px;
  color: var(--mini-cyber-dim);
  font-size: 13px;
  text-align: center;
}
</style>
