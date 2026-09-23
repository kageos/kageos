<template>
  <section class="archive-progress" aria-live="polite">
    <template v-if="progress">
      <div class="progress-heading"><strong>{{ t(`maintenance.archivePhases.${progress.phase}`) }}</strong><span>{{ t('maintenance.progressUpdated') }} {{ time(progress.updated_at) }}</span></div>
      <div class="progress-facts"><div><b>{{ progress.records.toLocaleString() }}</b><span>{{ t('maintenance.archivedRecords') }}</span></div><div><b>{{ progress.batches.toLocaleString() }}</b><span>{{ t('maintenance.completedBatches') }}</span></div><div v-if="progress.failed_batches" class="failed"><b>{{ progress.failed_batches }}</b><span>{{ t('maintenance.failedBatches') }}</span></div></div>
      <p v-if="!progress.finished_at && progress.batch_id">{{ t('maintenance.currentBatch', { id: progress.batch_id, count: progress.batch_records.toLocaleString() }) }}</p>
      <p v-if="progress.stop_reason">{{ t(`maintenance.archiveStops.${progress.stop_reason}`) }}</p>
      <p v-if="!progress.finished_at && stale" class="stale">{{ t('maintenance.progressStale') }}</p>
    </template>
    <p v-else class="stale">{{ t('maintenance.progressUnavailable') }}</p>
    <p v-if="error" class="stale">{{ t('maintenance.progressRefreshFailed') }}</p>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ArchiveRunProgress } from '@/architecture/presentation/context/api/system-settings'
const props = defineProps<{ progress?: ArchiveRunProgress | null; error?: boolean; now: number }>()
const { t, locale } = useI18n()
const stale = computed(() => !!props.progress && props.now - Date.parse(props.progress.updated_at) > 120000)
function time(value: string) { return new Date(value).toLocaleString(locale.value, { timeZone: 'Asia/Shanghai', hour12: false }) }
</script>
<style scoped>
.archive-progress { grid-column: 1 / -1; border: 1px solid var(--el-border-color-lighter); background: var(--el-fill-color-light); border-radius: 10px; padding: 16px 18px; margin-bottom: 16px; }.progress-heading { display: flex; justify-content: space-between; flex-wrap: wrap; gap: 8px; }.progress-heading strong { font-size: 13px; }.progress-heading span,p { color: var(--el-text-color-secondary); font-size: 12px; line-height: 1.7; }.progress-facts { display: flex; flex-wrap: wrap; gap: 32px; padding: 16px 0 6px; }.progress-facts > div { display: grid; gap: 4px; }.progress-facts b { font-size: 24px; font-variant-numeric: tabular-nums; font-weight: 600; }.progress-facts span { font-size: 12px; color: var(--el-text-color-secondary); }.failed,.stale { color: var(--el-color-warning); }p { margin: 8px 0 0; }
</style>
