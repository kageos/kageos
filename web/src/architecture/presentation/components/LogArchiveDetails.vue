<template>
  <div class="archive-details">
    <header class="dashboard-header"><h4>{{ t('logStorage.batchOverview') }} <span>#{{ batch.id }}</span></h4><span>{{ t(batch.archive_type === 'scheduled_execution' ? 'logStorage.scheduledKind' : 'logStorage.operateKind') }}</span></header>
    <div class="metrics">
      <section class="metric"><span>{{ t('logStorage.recordCount') }}</span><strong>{{ batch.record_count.toLocaleString() }}</strong><small>{{ t('logStorage.archives') }}</small></section>
      <section class="metric"><span>{{ t('logStorage.successRate') }}</span><strong class="success-value">{{ batch.summary_json?.status_counts ? share(batch.summary_json.status_counts.success || 0) : '—' }}</strong><small>{{ t('logStorage.resultStats') }}</small></section>
      <section class="metric"><span>{{ t('logStorage.fileSize') }}</span><strong>{{ compactSize(batch.file_size) }}</strong><small>JSONL · Gzip</small></section>
      <section class="metric"><span>{{ t('logStorage.timeRange') }}</span><div class="time-range"><time>{{ formatDateTimeValue(batch.range_started_at) }}</time><span>↓</span><time>{{ formatDateTimeValue(batch.range_ended_at) }}</time></div></section>
    </div>
    <div class="statistics">
      <section class="chart-card">
        <h4>{{ t('logStorage.resultStats') }}</h4>
        <ul v-if="results.length">
          <li v-for="item in results" :key="item.name"><span class="status-label"><i :class="item.name" />{{ statusLabel(item.name) }}</span><strong>{{ item.count.toLocaleString() }}</strong><span>{{ share(item.count) }}</span><div class="bar-track" aria-hidden="true"><div :style="{ width: barWidth(item.count) }" /></div></li>
        </ul>
        <p v-else>{{ t('logStorage.statsUnavailable') }}</p>
      </section>
      <section class="chart-card">
        <h4>{{ t('logStorage.resourceStats') }}</h4>
        <ul v-if="batch.summary_json?.top_resource_paths?.length">
          <li v-for="item in batch.summary_json.top_resource_paths" :key="item.resource_path"><span class="path">{{ item.resource_path || '—' }}</span><strong>{{ item.count.toLocaleString() }}</strong><span>{{ share(item.count) }}</span><div class="bar-track" aria-hidden="true"><div :style="{ width: barWidth(item.count) }" /></div></li>
        </ul>
        <p v-else>{{ t('logStorage.statsUnavailable') }}</p>
      </section>
    </div>
    <details class="file-card">
      <summary>{{ t('logStorage.fileMetadata') }}</summary>
      <dl class="metadata">
        <div v-for="item in metadata" :key="item.label"><dt>{{ item.label }}</dt><dd>{{ item.value }}</dd></div>
      </dl>
    </details>
    <el-alert v-if="batch.error_message" :title="t('logStorage.errorDetail')" :description="batch.error_message" type="error" :closable="false" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatDateTimeValue } from '@/architecture/shared/date'
import type { LogArchiveBatch } from '@/architecture/presentation/context/api/system-settings'

const props = defineProps<{ batch: LogArchiveBatch }>()
const { t } = useI18n()
const results = computed(() => Object.entries(props.batch.summary_json?.status_counts || {}).map(([name, count]) => ({ name, count })).sort((a, b) => b.count - a.count))
function statusLabel(status: string) {
  return ['success', 'failed', 'pending'].includes(status) ? t(`logStorage.${status}`) : status || t('logStorage.unknownStatus')
}
function compactSize(bytes: number) {
  return bytes < 1024 ? `${bytes} B` : bytes < 1048576 ? `${(bytes / 1024).toFixed(1)} KB` : `${(bytes / 1048576).toFixed(1)} MB`
}
function barWidth(count: number) {
  return `${props.batch.record_count > 0 ? Math.min(100, Math.max(0, count / props.batch.record_count * 100)) : 0}%`
}
function share(count: number) {
  return props.batch.record_count > 0 ? `${(count / props.batch.record_count * 100).toFixed(1)}%` : '—'
}
const metadata = computed(() => {
  const b = props.batch
  const absent = t('logStorage.sourcePending')
  const fields = [
    ['batchId', String(b.id)], ['archiveKey', b.archive_key || '—'],
    ['fileName', b.file_name || absent], ['fileFormat', t('logStorage.jsonlHint')],
    ['fileSize', `${b.file_size.toLocaleString()} B`],
    ['createdAt', formatDateTimeValue(b.created_at)],
    ['archivedAt', b.archived_at ? formatDateTimeValue(b.archived_at) : absent],
    ['deletedAt', b.deleted_at_source ? formatDateTimeValue(b.deleted_at_source) : absent],
    ['attempts', b.attempts === undefined ? absent : String(b.attempts)],
  ].map(([key, value]) => ({ label: t(`logStorage.${key}`), value }))
  if (b.next_retry_at && b.status !== 'completed') fields.push({ label: t('logStorage.nextRetry'), value: formatDateTimeValue(b.next_retry_at) })
  fields.push({ label: 'SHA256', value: b.sha256 || absent })
  return fields
})
</script>

<style scoped>
.archive-details { position: sticky; left: 0; width: 100cqw; box-sizing: border-box; padding: 22px 24px; display: grid; gap: 16px; background: var(--el-fill-color-light); color: var(--el-text-color-primary); }
.dashboard-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
h4 { margin: 0; font-size: 14px; font-weight: 600; }
.dashboard-header span { color: var(--el-text-color-secondary); font-weight: 400; font-size: 12px; }
h4 span { margin-left: 8px; }
.metrics { display: grid; grid-template-columns: 1fr 1fr 1fr 1.6fr; gap: 12px; }
.metric, .chart-card, .file-card { background: var(--el-bg-color); border: 1px solid var(--el-border-color-lighter); border-radius: 10px; }
.metric { padding: 18px 20px; display: flex; flex-direction: column; gap: 10px; }
.metric > span, small { font-size: 12px; color: var(--el-text-color-secondary); }
.metric > strong { font-size: 28px; font-weight: 600; letter-spacing: -.5px; line-height: 1.2; font-variant-numeric: tabular-nums; }
.success-value { color: var(--el-color-success); }
.time-range { display: grid; gap: 2px; font-size: 13px; font-variant-numeric: tabular-nums; }
.time-range span { color: var(--el-text-color-placeholder); }
.statistics { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1.6fr); gap: 12px; }
.chart-card { padding: 20px; min-width: 0; }
.chart-card h4 { margin-bottom: 18px; }
p, dt { color: var(--el-text-color-secondary); font-size: 12px; line-height: 1.7; }
p { margin: 0; padding: 12px 0; }
ul { list-style: none; margin: 0; padding: 0; max-height: 260px; overflow-y: auto; }
li { display: grid; grid-template-columns: minmax(0, 1fr) auto 60px; gap: 10px; padding: 6px 0 14px; font-size: 13px; align-items: center; }
li > span:last-of-type { color: var(--el-text-color-secondary); text-align: right; }
.status-label { display: flex; gap: 8px; align-items: center; }
.status-label i { width: 7px; height: 7px; border-radius: 50%; background: var(--el-text-color-placeholder); }
.status-label i.success { background: var(--el-color-success); }
.status-label i.failed { background: var(--el-color-danger); }
.bar-track { grid-column: 1 / -1; height: 6px; border-radius: 4px; background: var(--el-fill-color); overflow: hidden; }
.bar-track > div { height: 100%; background: var(--el-color-primary); border-radius: inherit; }
li:has(.success) .bar-track > div { background: var(--el-color-success); }
li:has(.failed) .bar-track > div { background: var(--el-color-danger); }
.path { overflow-wrap: anywhere; font-size: 12px; }
.file-card summary { cursor: pointer; padding: 16px 20px; font-size: 13px; font-weight: 500; }
.metadata { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 18px 24px; padding: 4px 20px 20px; margin: 0; }
dd { margin: 6px 0 0; font-size: 12px; overflow-wrap: anywhere; font-variant-numeric: tabular-nums; }
@container (max-width: 1000px) { .metrics, .metadata { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@container (max-width: 600px) { .statistics, .metadata, .metrics { grid-template-columns: 1fr; } .archive-details { position: sticky; left: 0; width: 100cqw; box-sizing: border-box; padding: 14px; } }
</style>
