<template>
  <section class="resource-archives" v-loading="loading">
    <header><div><h3>{{ t('logStorage.archives') }}</h3><p>{{ resourcePath }}</p></div><el-button :loading="loading" :disabled="busy !== null" @click="load">{{ t('common.refresh') }}</el-button></header>
    <p>{{ t('logStorage.retention', { days: policy.scheduled_retention_days || 7 }) }} {{ t('logStorage.threshold', { days: policy.retention_days || 90, count: policy.min_records || 1000 }) }} {{ t('logStorage.localTime') }}</p>
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <el-table :data="rows" :empty-text="t('systemSettings.noArchives')">
      <el-table-column :label="t('systemSettings.archiveTime')" min-width="160"><template #default="{row}">{{ formatDateTimeValue(row.archived_at || row.created_at) }}</template></el-table-column>
      <el-table-column :label="t('logStorage.resourceScope')" min-width="220"><template #default="{row}"><strong>{{ row.resource_path || `/${row.tenant_user}/${row.app}` }}</strong><p>{{ formatDateTimeValue(row.range_started_at) }} — {{ formatDateTimeValue(row.range_ended_at) }}</p></template></el-table-column>
      <el-table-column :label="t('logStorage.archiveKind')" width="145"><template #default="{row}">{{ t(row.archive_type === 'scheduled_execution' ? 'logStorage.scheduledKind' : 'logStorage.operateKind') }}</template></el-table-column>
      <el-table-column :label="t('systemSettings.archiveData')" min-width="190"><template #default="{row}"><strong>{{ t('systemSettings.archiveRecordValue', { count: row.record_count }) }}</strong><p>{{ size(row.file_size) }}</p><p v-if="row.summary_json?.status_counts">{{ t('logStorage.counts', {success: row.summary_json.status_counts.success || 0, failed: row.summary_json.status_counts.failed || 0}) }}</p></template></el-table-column>
      <el-table-column :label="t('systemSettings.archiveStatus')" width="110"><template #default="{row}"><el-tag :type="row.status === 'completed' ? 'success' : row.status === 'failed' ? 'danger' : 'warning'">{{ t(`systemSettings.archiveStatuses.${row.status}`) }}</el-tag></template></el-table-column>
      <el-table-column width="190"><template #default="{row}"><el-button v-if="row.object_ref && row.sha256" link type="primary" :loading="busy === row.id" @click="download(row)">{{ t('logStorage.download') }}</el-button><el-button v-if="row.status !== 'completed'" link :disabled="busy !== null" @click="retry(row)">{{ t('systemSettings.archiveRetry') }}</el-button></template></el-table-column>
      <el-table-column type="expand"><template #default="{row}"><LogArchiveDetails :batch="row" /></template></el-table-column>
    </el-table>
    <el-pagination v-if="total > 20" v-model:current-page="page" :total="total" :page-size="20" layout="prev, pager, next" @current-change="load" />
  </section>
</template>
<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue'
import { formatDateTimeValue } from '@/architecture/shared/date'
import LogArchiveDetails from './LogArchiveDetails.vue'
import { useI18n } from 'vue-i18n'
import { listResourceLogArchives, downloadResourceLogArchive, retryResourceLogArchive, type LogArchiveBatch, type ListLogArchiveBatchesResp } from '@/architecture/presentation/context/api/system-settings'
const props = defineProps<{resourcePath: string}>()
const { t } = useI18n()
const rows = ref<LogArchiveBatch[]>([]), policy = ref<Partial<ListLogArchiveBatchesResp>>({}), total = ref(0), page = ref(1), loading = ref(false), busy = ref<number | null>(null), error = ref('')
let generation = 0
function size(bytes: number) { return bytes < 1024 ? `${bytes} B` : bytes < 1048576 ? `${(bytes / 1024).toFixed(1)} KB` : `${(bytes / 1048576).toFixed(1)} MB` }
async function load() {
  const current = ++generation
  rows.value = []; total.value = 0; error.value = ''; busy.value = null
  if (!props.resourcePath) { loading.value = false; return }
  loading.value = true
  try {
    const result = await listResourceLogArchives(props.resourcePath, page.value)
    if (current !== generation) return
    rows.value = result.list; total.value = result.total; policy.value = result
  } catch (e) { if (current === generation) error.value = String(e) }
  finally { if (current === generation) loading.value = false }
}
async function download(row: LogArchiveBatch) {
  const current = generation
  busy.value = row.id; error.value = ''
  try {
    const result = await downloadResourceLogArchive(props.resourcePath, row.id)
    if (current !== generation) return
    const link = document.createElement('a')
    link.href = result.download_url; link.download = row.file_name; link.target = '_blank'; link.rel = 'noopener'; link.click()
  } catch (e) { if (current === generation) error.value = String(e) }
  finally { if (current === generation) busy.value = null }
}
async function retry(row: LogArchiveBatch) {
  const current = generation
  busy.value = row.id; error.value = ''
  try { await retryResourceLogArchive(props.resourcePath, row.id); if (current === generation) { busy.value = null; await load() } }
  catch (e) { if (current === generation) error.value = String(e) }
  finally { if (current === generation) busy.value = null }
}
watch(() => props.resourcePath, () => { page.value = 1; busy.value = null; void load() }, { immediate: true })
onUnmounted(() => { generation++ })
</script>
<style scoped>
.resource-archives { container-type: inline-size; display: grid; gap: 18px; min-width: 0; } header { display: flex; justify-content: space-between; gap: 16px; align-items: center; } h3 { margin: 0 0 6px; font-size: 16px; } p { margin: 5px 0; font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.7; overflow-wrap: anywhere; } strong { font-size: 13px; overflow-wrap: anywhere; } .archive-details { padding: 12px 20px; }
</style>
