<template>
  <div :class="['mini-interaction-gate', { 'is-readonly': readonly }]" :data-testid="dataTestId">
    <div class="mini-interaction-icon">
      <el-icon :size="readonly ? 18 : 24">
        <component :is="statusIcon" />
      </el-icon>
    </div>

    <div class="mini-interaction-main">
      <div class="mini-interaction-copy">
        <div class="mini-interaction-title-row">
          <div class="mini-interaction-title-copy">
            <span class="mini-interaction-eyebrow">{{ eyebrowText }}</span>
            <strong>{{ title }}</strong>
          </div>
          <span v-if="badgeText" class="mini-interaction-badge">{{ badgeText }}</span>
        </div>
        <p>{{ description }}</p>
      </div>

      <div v-if="!readonly && revisionOpen" class="mini-interaction-revision">
        <textarea
          v-model="revisionText"
          rows="3"
          :placeholder="revisionPlaceholder"
          :disabled="sending"
          data-testid="mini-interaction-revision-input"
        />
        <div class="mini-interaction-revision-actions">
          <el-button size="default" :disabled="sending" @click="revisionOpen = false">
            <el-icon><Close /></el-icon>
            {{ t('miniWorkstation.hideRevision') }}
          </el-button>
          <el-button
            type="primary"
            size="default"
            :loading="sending"
            :disabled="!revisionText.trim()"
            @click="submitRevision"
          >
            <el-icon><CircleCheck /></el-icon>
            {{ t('miniWorkstation.submitRevision') }}
          </el-button>
        </div>
      </div>

      <div v-else-if="!readonly" class="mini-interaction-actions">
        <el-button size="default" @click="$emit('view')">
          <el-icon><View /></el-icon>
          {{ viewLabel }}
        </el-button>
        <el-button size="default" @click="openRevision">
          <el-icon><EditPen /></el-icon>
          {{ reviseLabel }}
        </el-button>
        <el-button size="default" @click="$emit('cancel')">
          <el-icon><CircleClose /></el-icon>
          {{ cancelLabel }}
        </el-button>
        <el-button type="primary" size="default" :loading="sending" @click="$emit('confirm')">
          <el-icon><CircleCheck /></el-icon>
          {{ confirmLabel }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { CircleCheck, CircleClose, Close, EditPen, View, WarningFilled } from '@element-plus/icons-vue'
import type { WorkspaceInteraction } from '@/architecture/presentation/context/api/workspace'

const props = defineProps<{
  interaction: WorkspaceInteraction
  sending: boolean
  readonly?: boolean
}>()

const emit = defineEmits<{
  (e: 'view'): void
  (e: 'revise', payload: { text: string }): void
  (e: 'cancel'): void
  (e: 'confirm'): void
}>()

const revisionOpen = ref(false)
const revisionText = ref('')
const { t } = useI18n()

const cardType = computed(() => props.interaction.card_type || 'stage_confirmation')
const title = computed(() => props.interaction.title || defaultTitle(cardType.value))
const description = computed(() => {
  return props.interaction.description ||
    props.interaction.help_text ||
    t('miniWorkstation.interactionDefaultDesc')
})
const viewLabel = computed(() => props.interaction.view_text || defaultViewLabel(cardType.value))
const reviseLabel = computed(() => props.interaction.revise_text || defaultReviseLabel(cardType.value))
const cancelLabel = computed(() => props.interaction.cancel_text || defaultCancelLabel(cardType.value))
const confirmLabel = computed(() => props.interaction.confirm_text || defaultConfirmLabel(cardType.value))
const dataTestId = computed(() => `mini-interaction-gate-${cardType.value}`)
const statusIcon = computed(() => props.readonly ? CircleCheck : WarningFilled)
const eyebrowText = computed(() => props.readonly ? t('miniWorkstation.historicalInteraction') : t('miniWorkstation.currentSessionPaused'))
const badgeText = computed(() => {
  if (props.readonly) return t('miniWorkstation.recorded')
  if (props.interaction.blocking) return t('miniWorkstation.needsHandling')
  return ''
})
const revisionPlaceholder = computed(() => {
  if (cardType.value === 'prd_confirmation') return t('miniWorkstation.revisePrdPlaceholder')
  if (cardType.value === 'build_repair') return t('miniWorkstation.reviseBuildPlaceholder')
  return t('miniWorkstation.reviseGenericPlaceholder')
})

watch(() => props.interaction.id, () => {
  revisionOpen.value = false
  revisionText.value = ''
})

watch(() => props.readonly, (readonly) => {
  if (!readonly) return
  revisionOpen.value = false
  revisionText.value = ''
})

function openRevision() {
  if (props.readonly) return
  revisionOpen.value = true
}

function submitRevision() {
  const text = revisionText.value.trim()
  if (!text) return
  emit('revise', { text })
}

function defaultTitle(type: string) {
  if (type === 'prd_confirmation') return t('miniWorkstation.interactionPrdTitle')
  if (type === 'build_repair') return t('miniWorkstation.buildRepairTitle')
  return t('miniWorkstation.interactionWaitingTitle')
}

function defaultViewLabel(type: string) {
  if (type === 'prd_confirmation') return t('miniWorkstation.viewPrd')
  if (type === 'build_repair') return t('miniWorkstation.viewDiagnostics')
  return t('miniWorkstation.viewDetails')
}

function defaultReviseLabel(type: string) {
  if (type === 'prd_confirmation') return t('miniWorkstation.revisePrd')
  if (type === 'build_repair') return t('miniWorkstation.continueDevelopment')
  return t('miniWorkstation.supplement')
}

function defaultCancelLabel(type: string) {
  if (type === 'build_repair') return t('miniWorkstation.skipRepair')
  return t('scheduledTask.cancel')
}

function defaultConfirmLabel(type: string) {
  if (type === 'prd_confirmation') return t('miniWorkstation.confirmPrd')
  if (type === 'build_repair') return t('miniWorkstation.repairHandoff')
  return t('miniWorkstation.confirm')
}
</script>

<style scoped>
.mini-interaction-gate {
  flex-shrink: 0;
  display: grid;
  grid-template-columns: 48px minmax(0, 1fr);
  gap: 14px;
  width: 100%;
  min-height: 132px;
  margin: 18px 0 20px;
  padding: 18px;
  border: 1px solid rgba(246, 199, 107, 0.55);
  border-radius: 8px;
  background:
    linear-gradient(145deg, rgba(22, 37, 54, 0.96), rgba(6, 16, 30, 0.92)),
    linear-gradient(90deg, rgba(246, 199, 107, 0.22), rgba(34, 211, 238, 0.08) 56%, transparent);
  box-shadow:
    0 16px 40px rgba(0, 0, 0, 0.3),
    0 0 0 1px rgba(255, 255, 255, 0.03) inset,
    0 0 28px rgba(246, 199, 107, 0.12);
}

.mini-interaction-gate.is-readonly {
  grid-template-columns: 34px minmax(0, 1fr);
  min-height: 0;
  margin: 10px 0;
  padding: 11px 12px;
  border-color: rgba(148, 163, 184, 0.2);
  background: linear-gradient(145deg, rgba(8, 21, 35, 0.62), rgba(4, 12, 24, 0.44));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.mini-interaction-icon {
  display: grid;
  place-items: center;
  width: 48px;
  height: 48px;
  color: #25180a;
  border: 1px solid rgba(255, 226, 145, 0.66);
  border-radius: 8px;
  background: linear-gradient(180deg, #ffe39b, #f6bd4d);
  box-shadow: 0 8px 18px rgba(246, 189, 77, 0.22);
}

.mini-interaction-gate.is-readonly .mini-interaction-icon {
  width: 34px;
  height: 34px;
  color: rgba(205, 232, 240, 0.76);
  border-color: rgba(148, 163, 184, 0.22);
  background: rgba(148, 163, 184, 0.1);
  box-shadow: none;
}

.mini-interaction-main {
  min-width: 0;
  display: grid;
  gap: 16px;
  align-content: start;
}

.mini-interaction-gate.is-readonly .mini-interaction-main {
  gap: 6px;
}

.mini-interaction-copy {
  min-width: 0;
  display: grid;
  gap: 8px;
}

.mini-interaction-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.mini-interaction-title-copy {
  min-width: 0;
  display: grid;
  gap: 4px;
}

.mini-interaction-eyebrow {
  color: #ffe7ad;
  font-size: 12px;
  font-weight: 700;
  line-height: 1.2;
}

.mini-interaction-gate.is-readonly .mini-interaction-eyebrow {
  color: var(--mini-cyber-dim, rgba(143, 187, 204, 0.48));
  font-size: 11px;
  font-weight: 600;
}

.mini-interaction-title-row strong {
  color: var(--mini-cyber-text, #d8f8ff);
  font-size: 18px;
  line-height: 1.25;
  overflow-wrap: anywhere;
}

.mini-interaction-gate.is-readonly .mini-interaction-title-row strong {
  font-size: 13px;
  font-weight: 650;
}

.mini-interaction-badge {
  flex-shrink: 0;
  padding: 4px 9px;
  color: #2b1906;
  font-size: 12px;
  font-weight: 800;
  line-height: 18px;
  border: 1px solid rgba(255, 230, 166, 0.7);
  border-radius: 999px;
  background: linear-gradient(180deg, #ffe19a, #f5bd4c);
  box-shadow: 0 6px 14px rgba(246, 189, 77, 0.18);
}

.mini-interaction-gate.is-readonly .mini-interaction-badge {
  padding: 2px 7px;
  color: rgba(210, 231, 237, 0.72);
  font-size: 11px;
  font-weight: 650;
  line-height: 16px;
  border-color: rgba(148, 163, 184, 0.22);
  background: rgba(148, 163, 184, 0.1);
  box-shadow: none;
}

.mini-interaction-copy p {
  margin: 0;
  color: rgba(226, 247, 252, 0.82);
  font-size: 13px;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.mini-interaction-gate.is-readonly .mini-interaction-copy p {
  color: var(--mini-cyber-muted, rgba(184, 225, 235, 0.68));
  font-size: 12px;
  line-height: 1.45;
}

.mini-interaction-actions,
.mini-interaction-revision-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex-wrap: wrap;
}

.mini-interaction-actions :deep(.el-button),
.mini-interaction-revision-actions :deep(.el-button) {
  margin-left: 0;
  min-height: 36px;
  padding: 8px 13px;
  font-size: 13px;
  font-weight: 650;
  border-radius: 8px;
}

.mini-interaction-actions :deep(.el-button .el-icon),
.mini-interaction-revision-actions :deep(.el-button .el-icon) {
  margin-right: 5px;
}

.mini-interaction-actions :deep(.el-button--primary),
.mini-interaction-revision-actions :deep(.el-button--primary) {
  color: #06101c;
  border-color: rgba(122, 234, 255, 0.74);
  background: linear-gradient(180deg, #8befff, #22d3ee);
  box-shadow: 0 10px 24px rgba(34, 211, 238, 0.22);
}

.mini-interaction-revision {
  display: grid;
  gap: 10px;
}

.mini-interaction-revision textarea {
  width: 100%;
  min-height: 104px;
  resize: vertical;
  padding: 11px 12px;
  color: var(--mini-cyber-text);
  font: inherit;
  font-size: 13px;
  line-height: 1.5;
  background: rgba(3, 8, 16, 0.72);
  border: 1px solid rgba(102, 229, 255, 0.3);
  border-radius: 8px;
  outline: none;
}

.mini-interaction-revision textarea:focus {
  border-color: rgba(102, 229, 255, 0.5);
  box-shadow: 0 0 0 2px rgba(34, 211, 238, 0.12);
}

:global(.mini-ws:not(.mini-ws--maximized)) .mini-interaction-actions,
:global(.mini-ws:not(.mini-ws--maximized)) .mini-interaction-revision-actions {
  justify-content: stretch;
  flex-wrap: wrap;
}

:global(.mini-ws:not(.mini-ws--maximized)) .mini-interaction-actions :deep(.el-button),
:global(.mini-ws:not(.mini-ws--maximized)) .mini-interaction-revision-actions :deep(.el-button) {
  flex: 1 1 132px;
}

:global(.mini-ws:not(.mini-ws--maximized)) .mini-interaction-gate {
  grid-template-columns: 38px minmax(0, 1fr);
  min-height: 120px;
  margin: 14px 0 16px;
  padding: 14px;
}

:global(.mini-ws:not(.mini-ws--maximized)) .mini-interaction-icon {
  width: 38px;
  height: 38px;
}

:global(.mini-ws:not(.mini-ws--maximized)) .mini-interaction-title-row {
  align-items: stretch;
  flex-direction: column;
  gap: 8px;
}

:global(.mini-ws:not(.mini-ws--maximized)) .mini-interaction-badge {
  align-self: flex-start;
}

:global(.mini-ws:not(.mini-ws--maximized)) .mini-interaction-title-row strong {
  font-size: 16px;
}

:global(.mini-ws:not(.mini-ws--maximized)) .mini-interaction-gate.is-readonly {
  grid-template-columns: 30px minmax(0, 1fr);
  min-height: 0;
  padding: 10px;
}

:global(.mini-ws:not(.mini-ws--maximized)) .mini-interaction-gate.is-readonly .mini-interaction-icon {
  width: 30px;
  height: 30px;
}
</style>
