<template>
  <section v-loading="loading" class="maintenance">
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <header class="maintenance-intro"><div class="intro-symbol"><el-icon :size="26"><Timer /></el-icon></div><div><h3>{{ t('maintenance.taskList') }}</h3><p>{{ t('maintenance.scopeHint') }}</p></div></header>
    <div class="task-toolbar">
      <div class="list-caption"><strong>{{ t('maintenance.registeredCount', {count:total}) }}</strong><span class="muted">{{ t('maintenance.timeHint') }}</span></div>
      <div class="toolbar-controls">
        <el-input v-model="query" :placeholder="t('maintenance.search')" :prefix-icon="Search" clearable class="task-search" />
        <el-radio-group v-model="filter"><el-radio-button value="all">{{ t('maintenance.all') }}</el-radio-button><el-radio-button value="failed">{{ t('maintenance.needsAttention') }}</el-radio-button><el-radio-button value="paused">{{ t('maintenance.paused') }}</el-radio-button></el-radio-group>
      </div>
    </div>
    <el-empty v-if="!loading && !filteredTasks.length" :description="t('maintenance.emptyTasks')" :image-size="72" />
    <div class="task-list">
      <article v-for="task in filteredTasks" :key="task.id" class="task-card" :class="{ 'has-error': ['failed', 'timeout'].includes(latest?.[task.id]?.status || '') }">
        <div class="task-heading">
          <span class="task-symbol"><el-icon :size="21"><component :is="taskIcon(task)" /></el-icon></span><div class="task-identity"><h4>{{ task.title || task.executor_key }}</h4><el-tag size="small" effect="light" :type="tone(task.inflight_execution_id ? 'running' : task.status)">{{ label(task.inflight_execution_id ? 'running' : task.status) }}</el-tag></div>

        </div>
        <p class="task-description">{{ purpose(task) }}</p>
        <div class="task-facts">
          <div><span class="fact-label">{{ t('maintenance.cadence') }}</span><strong>{{ schedule(task) }}</strong><span class="muted">{{ task.schedule.timezone || 'Asia/Shanghai' }}</span></div>
          <div><span class="fact-label">{{ t('maintenance.last') }}</span><strong class="timestamp">{{ time(latest?.[task.id]?.started_at || latest?.[task.id]?.scheduled_at) }}</strong><span v-if="latest?.[task.id]" class="last-result"><el-tag size="small" :type="tone(latest?.[task.id]?.status)">{{ label(latest?.[task.id]?.status || '') }}</el-tag><span class="muted">{{ elapsed(latest?.[task.id]) }}</span></span><span v-else class="muted">{{ t('maintenance.neverRun') }}</span></div>
          <div><span class="fact-label">{{ t('maintenance.next') }}</span><strong class="timestamp">{{ task.status === 'paused' ? '—' : time(task.next_run_at) }}</strong><span class="muted">{{ task.status === 'paused' ? t('maintenance.pauseHint') : t('maintenance.automatic') }}</span></div>
        </div>
        <div v-if="latest?.[task.id]?.status === 'running'" class="heartbeat-note"><span>{{ t('maintenance.heartbeat') }} {{ time(latest?.[task.id]?.heartbeat_at) }}</span><strong v-if="heartbeatStale(latest?.[task.id]?.heartbeat_at)">{{ t('maintenance.heartbeatStale') }}</strong></div>
        <ArchiveProgress v-if="task.executor_key === 'platform.log_archive' && latest?.[task.id]?.status === 'running'" :progress="progress[latest[task.id]!.id]" :error="progressErrors[latest[task.id]!.id]" :now="now" />
        <ExecutionError v-if="task.last_error_message" :message="task.last_error_message" />
        <footer class="task-footer"><el-button :icon="Timer" type="primary" link @click="showHistory(task)">{{ t('maintenance.history') }}<el-icon><ArrowRight /></el-icon></el-button>          <div class="task-actions"><el-button :icon="task.status === 'paused' ? VideoPlay : VideoPause" size="small" :disabled="busy === task.id" @click="control(task, task.status === 'paused' ? 'resume' : 'pause')">{{ task.status === 'paused' ? t('maintenance.resume') : t('maintenance.pause') }}</el-button><el-button :icon="VideoPlay" size="small" type="primary" plain :loading="busy === task.id" :disabled="!!task.inflight_execution_id || !!busy" @click="control(task, 'run')">{{ t('maintenance.run') }}</el-button></div></footer>
      </article>
    </div>
    <el-pagination v-if="total > 50" v-model:current-page="page" :total="total" :page-size="50" layout="prev, pager, next" @current-change="refresh()" />
    <article class="backup-card">
      <div class="task-heading"><div class="backup-title"><span class="task-symbol"><el-icon :size="21"><Coin /></el-icon></span><div><h3>{{ t('maintenance.backup') }}</h3><p class="muted">{{ t('maintenance.backupHint') }}</p></div></div><el-button @click="emit('backup-settings')">{{ t('maintenance.backupSettings') }}</el-button></div>
      <el-alert v-if="backupError" :title="backupError" type="error" :closable="false" />
      <template v-if="backup">
        <div class="backup-overview"><div class="last-result"><el-tag :type="backup.agent_available ? 'success' : 'danger'">{{ backup.agent_available ? t('maintenance.agentOnline') : t('maintenance.agentOffline') }}</el-tag><span>{{ t('maintenance.dailyAt', { time: backup.config.schedule_time }) }}</span><span class="muted">{{ backup.config.enabled ? t('maintenance.enabled') : t('maintenance.paused') }}</span></div><el-button type="primary" plain :loading="backupBusy" :disabled="!backup.agent_available || backup.running" @click="runBackup">{{ backup.running ? t('maintenance.running') : t('maintenance.run') }}</el-button></div>
        <div v-for="record in backup.records.slice(0, 3)" :key="record.started_at" class="backup-record"><el-tag size="small" :type="tone(record.status)">{{ label(record.status) }}</el-tag><span class="timestamp">{{ time(record.started_at) }}</span><span class="muted">{{ t('maintenance.finished') }} {{ time(record.finished_at) }}</span><span v-if="record.error_message" class="error-text">{{ record.error_message }}</span></div>
        <p v-if="!backup.records.length" class="muted">{{ t('maintenance.emptyHistory') }}</p>
      </template>
    </article>
    <el-drawer v-model="visible" size="min(1080px, 96vw)" class="maintenance-drawer">
      <template #header><div class="drawer-title"><span class="eyebrow">{{ t('maintenance.history') }}</span><h3>{{ selected?.title }}</h3><p class="muted">{{ t('maintenance.timeHint') }}</p></div></template>
      <div class="history-toolbar"><el-radio-group v-model="historyStatus" @change="changeHistoryFilter"><el-radio-button value="">{{ t('maintenance.all') }}</el-radio-button><el-radio-button value="failed">{{ t('maintenance.statuses.failed') }}</el-radio-button><el-radio-button value="timeout">{{ t('maintenance.statuses.timeout') }}</el-radio-button><el-radio-button value="success">{{ t('maintenance.statuses.success') }}</el-radio-button></el-radio-group><el-button :icon="Refresh" :loading="historyLoading" @click="loadHistory()">{{ t('common.refresh') }}</el-button></div>
      <el-alert v-if="historyError" :title="historyError" type="error" :closable="false" />
      <div v-loading="historyLoading" class="history-list">
        <el-empty v-if="!historyLoading && !history.length" :description="t('maintenance.emptyHistory')" :image-size="80" />
        <article v-for="row in history" :key="row.id" class="execution-card" :class="{ 'has-error': row.status === 'failed' || row.status === 'timeout' }">
          <div class="execution-heading"><div class="last-result"><el-tag :type="tone(row.status)">{{ label(row.status) }}</el-tag><strong class="timestamp">{{ time(row.started_at || row.scheduled_at) }}</strong></div><span class="execution-id">#{{ row.id }}</span></div>
          <div v-if="row.status === 'running'" class="heartbeat-note"><span>{{ t('maintenance.heartbeat') }} {{ time(row.heartbeat_at) }}</span><strong v-if="heartbeatStale(row.heartbeat_at)">{{ t('maintenance.heartbeatStale') }}</strong></div><div class="execution-meta"><span>{{ row.trigger_type === 'manual' ? t('maintenance.manual') : row.trigger_type === 'scheduled' ? t('maintenance.automatic') : row.trigger_type || '—' }}</span><span>{{ t('maintenance.duration') }} {{ elapsed(row) }}</span></div>
          <ArchiveProgress v-if="selected?.executor_key === 'platform.log_archive' && row.status === 'running'" :progress="progress[row.id]" :error="progressErrors[row.id]" :now="now" />
          <ExecutionError v-if="row.error_message" :message="row.error_message" />
          <p v-if="row.output_summary || row.result_payload != null" class="execution-summary">{{ summary(row) }}</p>
          <details class="execution-details"><summary>{{ t('maintenance.executionDetails') }}</summary><dl class="detail-grid"><div><dt>{{ t('maintenance.scheduled') }}</dt><dd>{{ time(row.scheduled_at) }}</dd></div><div><dt>{{ t('maintenance.started') }}</dt><dd>{{ time(row.started_at) }}</dd></div><div><dt>{{ t('maintenance.finished') }}</dt><dd>{{ time(row.finished_at) }}</dd></div><div v-if="row.trace_id"><dt>Trace ID</dt><dd>{{ row.trace_id }}</dd></div></dl><h5>{{ t('maintenance.output') }}</h5><pre>{{ payload(row.result_payload ?? row.output_summary) }}</pre></details>
        </article>
      </div>
      <template #footer><div class="history-footer"><span class="muted">{{ t('maintenance.recordCount', { count: historyTotal }) }}</span><el-pagination v-model:current-page="historyPage" :total="historyTotal" :page-size="20" layout="prev, pager, next" @current-change="loadHistory()" /></div></template>
    </el-drawer>
  </section>
</template>
<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Timer, Search, VideoPlay, VideoPause, Refresh, ArrowRight, Coin, DataAnalysis, Files, Delete, FolderDelete } from '@element-plus/icons-vue'
import { listTimerTasks, listTimerExecutions, pauseTimerTask, resumeTimerTask, runTimerTaskNow, type TimerTask, type TimerExecution, type ListTimerTasksResponse } from '@/architecture/presentation/context/api/timer'
import { listLogArchiveBatches, getArchiveProgress, type ArchiveRunProgress, getSystemBackupOverview, runSystemBackupNow, type SystemBackupOverview } from '@/architecture/presentation/context/api/system-settings'
import ExecutionError from '@/architecture/presentation/components/ExecutionError.vue'
import ArchiveProgress from './ArchiveProgress.vue'
const progressSupported = ref(false)
const progress = ref<Record<number, ArchiveRunProgress | null>>({}), progressErrors = ref<Record<number, boolean>>({}), now = ref(Date.now())
const emit = defineEmits<{ 'backup-settings': [] }>()
const { t, locale } = useI18n()
const latest = ref<ListTimerTasksResponse['last_executions']>({})
const tasks = ref<TimerTask[]>([]), total = ref(0), page = ref(1), loading = ref(false), error = ref(''), busy = ref(0)
const backup = ref<SystemBackupOverview | null>(null), backupError = ref(''), backupBusy = ref(false)
const selected = ref<TimerTask | null>(null), visible = ref(false), history = ref<TimerExecution[]>([]), historyTotal = ref(0), historyPage = ref(1), historyLoading = ref(false), historyError = ref('')
let generation = 0, historyGeneration = 0
const query = ref(''), filter = ref('all'), historyStatus = ref('')
function taskIcon(task: TimerTask) {
 switch(task.executor_key) {
  case 'platform.platform_snapshot': return DataAnalysis
  case 'platform.capacity_snapshot': return Coin
  case 'platform.log_archive': return Files
  case 'platform.runtime_cleanup': return FolderDelete
  case 'platform.orphan_tasks': return Delete
  case 'platform.soft_delete_cleanup': return Delete
  default: return Timer
 }
}
function purpose(task: TimerTask) {
 const key = task.executor_key.replace('platform.', '')
 return ['platform_snapshot', 'capacity_snapshot', 'log_archive', 'runtime_cleanup', 'orphan_tasks', 'soft_delete_cleanup'].includes(key) && task.executor_key.startsWith('platform.') ? t(`maintenance.purposes.${key}`) : task.description || ''
}
const filteredTasks = computed(() => tasks.value.filter(task => {
 if (['platform.log_groups', 'platform.log_groups_reconcile'].includes(task.executor_key)) return false
 const matches = `${task.title} ${purpose(task)} ${task.executor_key}`.toLowerCase().includes(query.value.toLowerCase())
 return matches && (filter.value === 'all' || (filter.value === 'failed' ? ['failed', 'timeout'].includes(latest.value?.[task.id]?.status || '') : task.status === 'paused'))
}))
function time(value?: string) { return value ? new Date(value).toLocaleString(locale.value, {year:'numeric', month:'2-digit', day:'2-digit', hour:'2-digit', minute:'2-digit', second:'2-digit', hour12:false, timeZone:'Asia/Shanghai'}) : '—' }
function heartbeatStale(value?: string) { return !!value && Date.now() - Date.parse(value) > 120000 }
function elapsed(row?: Pick<TimerExecution, 'status' | 'started_at' | 'duration_millis'>) { return row?.status === 'running' && row.started_at ? duration(Math.max(0, Date.now() - Date.parse(row.started_at))) : duration(row?.duration_millis) }
function duration(ms?: number) { return ms == null ? '—' : ms < 1000 ? `${ms} ms` : ms < 60000 ? `${(ms / 1000).toFixed(1)} s` : `${Math.floor(ms / 60000)} min ${Math.round(ms % 60000 / 1000)} s` }
function tone(status?: string) { return status === 'success' || status === 'pending' ? 'success' : status === 'failed' || status === 'timeout' ? 'danger' : status === 'running' || status === 'queued' ? 'primary' : 'info' }
function label(status: string) { const key = `maintenance.statuses.${status}`; return ['success','failed','timeout','cancelled','skipped','waiting','queued','running','pending','paused','done'].includes(status) ? t(key) : status }
function schedule(task: TimerTask) {
 const value = task.schedule
 if(value.type === 'every') return t('maintenance.every', {seconds:value.interval_seconds})
 if(value.type === 'cron') {
  const parts = value.cron_expr?.trim().split(/\s+/) || []
  if(parts.length === 5 && /^\d+$/.test(parts[0] || '') && /^\d+$/.test(parts[1] || '') && parts.slice(2).every(p => p === '*')) return t('maintenance.dailyAt', {time:`${parts[1]?.padStart(2,'0')}:${parts[0]?.padStart(2,'0')}`})
  return value.cron_expr
 }
 return value.type === 'atime' ? time(value.run_at) : t('maintenance.manual')
}
function payload(value: unknown) { return value == null ? t('maintenance.noResult') : typeof value === 'string' ? value : JSON.stringify(value, null, 2) }
function summary(row: TimerExecution) {
 const data = row.result_payload as {records?: number; batches?: number; stop_reason?: string} | undefined
 if (selected.value?.executor_key === 'platform.log_archive' && data && typeof data.records === 'number' && typeof data.batches === 'number') return t('maintenance.archiveSummary', {count:data.records.toLocaleString(), batches:data.batches}) + (data.stop_reason ? ' · ' + t(`maintenance.archiveStops.${data.stop_reason}`) : '')
 const value = row.output_summary?.trim(); return value && !value.startsWith('{') && !value.startsWith('[') ? value : t('maintenance.resultAvailable') }
async function refreshProgress() {
 if (!progressSupported.value) return
 const ids = tasks.value.filter(task => task.executor_key === 'platform.log_archive').map(task => latest.value?.[task.id]).filter(row => row?.status === 'running').map(row => row!.id)
 await Promise.allSettled(ids.map(async id => { try { const result = await getArchiveProgress(id); progress.value[id] = result.progress; progressErrors.value[id] = false } catch { progressErrors.value[id] = true } }))
}
function changeHistoryFilter() { historyPage.value = 1; void loadHistory() }

async function refresh(silent = false) {
 const current = ++generation; if (!silent) loading.value = true; error.value = ''
 const [a,b,c] = await Promise.allSettled([listTimerTasks({resource_scope:'system',page:page.value,page_size:50}),getSystemBackupOverview(),listLogArchiveBatches(1,1)])
 if (current !== generation) return
 if (a.status === 'fulfilled') {tasks.value=a.value.list;total.value=a.value.total;latest.value=a.value.last_executions || {}} else error.value=String(a.reason)
 if (b.status === 'fulfilled') {backup.value=b.value;backupError.value=''} else backupError.value=String(b.reason)
 progressSupported.value = c.status === 'fulfilled' && c.value.progress_supported === true
 loading.value=false
 void refreshProgress()
}
async function control(task: TimerTask, action: 'run'|'pause'|'resume') {
 if (action==='run') {try {await ElMessageBox.confirm(t('maintenance.confirmRun',{name:task.title,description:purpose(task)}),t('maintenance.run'))} catch {return}}
 busy.value=task.id
 try {if(action==='run'){await runTimerTaskNow(task.id);ElMessage.success(t('maintenance.queued'));await showHistory(task)} else if(action==='pause') await pauseTimerTask(task.id);else await resumeTimerTask(task.id);await refresh()}
 catch(e){ElMessage.error(String(e))}finally{busy.value=0}
}
async function showHistory(task: TimerTask){selected.value=task;historyPage.value=1;historyStatus.value='';history.value=[];visible.value=true;await loadHistory()}
async function loadHistory(silent = false){
 if(!selected.value)return
 const current=++historyGeneration;if(!silent)historyLoading.value=true;historyError.value=''
 try{const result=await listTimerExecutions(selected.value.id,{page:historyPage.value,page_size:20,status:historyStatus.value || undefined});if(current!==historyGeneration)return;history.value=result.list;historyTotal.value=result.total}
 catch(e){if(current===historyGeneration)historyError.value=String(e)}finally{if(current===historyGeneration)historyLoading.value=false}
}
async function runBackup(){backupBusy.value=true;try{backup.value=await runSystemBackupNow();ElMessage.success(t('maintenance.queued'))}catch(e){ElMessage.error(String(e))}finally{backupBusy.value=false}}
let poll: ReturnType<typeof setInterval> | undefined
onMounted(()=>{void refresh();poll=setInterval(()=>{now.value=Date.now();if(document.hidden)return;if(!loading.value)void refresh(true);if(visible.value&&!historyLoading.value)void loadHistory(true)},15000)})
onUnmounted(()=>{clearInterval(poll);generation++;historyGeneration++})
defineExpose({refresh})
</script>
<style scoped>
.maintenance { display: grid; gap: 24px; color: var(--el-text-color-primary); }
h3, h4, h5, p { margin: 0; }
h3 { font-size: 16px; font-weight: 600; } h4 { font-size: 15px; font-weight: 600; line-height: 1.5; }
.muted, .fact-label { color: var(--el-text-color-secondary); font-size: 12px; line-height: 1.6; }
.task-toolbar, .task-heading, .task-footer, .backup-overview, .execution-heading, .history-toolbar, .history-footer { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.task-toolbar p, .backup-card p, .drawer-title p { margin-top: 6px; }
.count { margin-left: 6px; padding: 2px 7px; border-radius: 6px; background: var(--el-fill-color); color: var(--el-text-color-secondary); font-size: 12px; }
.toolbar-controls, .task-identity, .last-result, .task-actions { display: flex; align-items: center; gap: 10px; }
.scope-note { padding: 12px 16px; border-radius: 8px; background: var(--el-fill-color-light); color: var(--el-text-color-secondary); font-size: 13px; line-height: 1.7; }
.task-search { width: 200px; }.task-list { display: grid; gap: 14px; }
.task-card, .backup-card, .execution-card { min-width: 0; border: 1px solid var(--el-border-color-lighter); border-radius: 12px; background: var(--el-bg-color); overflow: hidden; }
.task-card { display: grid; grid-template-columns: minmax(260px, 1fr) minmax(460px, 1.4fr); column-gap: 32px; padding: 22px 24px 0; }.task-heading { align-items: flex-start; }.task-identity { flex-wrap: wrap; gap: 8px; align-content: center; }.task-actions { flex-shrink: 0; gap: 0; }
.task-description { grid-column: 1; font-size: 13px; line-height: 1.7; color: var(--el-text-color-secondary); margin: 10px 0 20px 48px; overflow-wrap: anywhere; }
.task-facts { grid-column: 2; grid-row: 1 / 3; align-self: center; display: grid; grid-template-columns: .8fr 1.2fr 1.2fr; gap: 16px; margin-bottom: 18px; }.task-facts > div { display: flex; flex-direction: column; gap: 7px; }.task-facts strong { font-size: 12px; font-weight: 500; line-height: 1.6; }.timestamp { font-variant-numeric: tabular-nums; }
.task-footer { grid-column: 1 / -1; border-top: 1px solid var(--el-border-color-extra-light); padding: 10px 0; }.task-footer code { color: var(--el-text-color-placeholder); font-size: 11px; overflow-wrap: anywhere; }
.has-error { border-color: color-mix(in srgb, var(--el-color-danger) 35%, var(--el-border-color-lighter)); }.error-notice { grid-column: 1 / -1; display: grid; gap: 5px; padding: 12px; margin: 0 0 14px; border-radius: 8px; background: var(--el-color-danger-light-9); color: var(--el-color-danger); font-size: 12px; line-height: 1.7; overflow-wrap: anywhere; white-space: pre-wrap; }.error-notice strong { font-weight: 600; }
.backup-card { padding: 22px; }.backup-overview { padding: 20px 0; }.backup-record { display: flex; align-items: center; flex-wrap: wrap; gap: 14px; padding: 12px 0; border-top: 1px solid var(--el-border-color-extra-light); font-size: 12px; }.error-text { color: var(--el-color-danger); overflow-wrap: anywhere; }
.drawer-title h3 { margin-top: 6px; font-size: 20px; }.eyebrow { font-size: 12px; color: var(--el-text-color-secondary); }.history-toolbar { margin-bottom: 20px; }.history-list { min-height: 160px; display: grid; gap: 14px; }.execution-card { padding: 18px; }.execution-heading strong { font-size: 14px; font-weight: 500; }.execution-id { color: var(--el-text-color-placeholder); font-size: 12px; }.execution-meta { display: flex; gap: 18px; font-size: 12px; color: var(--el-text-color-secondary); margin: 12px 0 16px; }.execution-summary { font-size: 13px; line-height: 1.7; margin-bottom: 14px; overflow-wrap: anywhere; white-space: pre-wrap; }
.execution-details { border-top: 1px solid var(--el-border-color-extra-light); padding-top: 12px; }.execution-details summary { color: var(--el-color-primary); font-size: 12px; cursor: pointer; width: fit-content; }.execution-details summary:focus-visible { outline: 2px solid var(--el-color-primary); outline-offset: 4px; }.detail-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin: 18px 0; }.detail-grid dt { font-size: 12px; color: var(--el-text-color-secondary); }.detail-grid dd { margin: 5px 0 0; font-size: 12px; overflow-wrap: anywhere; font-variant-numeric: tabular-nums; }.execution-details h5 { font-size: 12px; font-weight: 500; margin-bottom: 8px; }.execution-details pre { margin: 0; white-space: pre-wrap; overflow-wrap: anywhere; max-height: 360px; overflow: auto; padding: 14px; border-radius: 8px; background: var(--el-fill-color-light); color: var(--el-text-color-regular); font-size: 12px; line-height: 1.7; }
@media (max-width: 1150px) { .task-card { grid-template-columns: 1fr; }.task-facts { grid-column: 1; grid-row: auto; margin-left: 48px; } }
@media (max-width: 700px) { .task-toolbar, .toolbar-controls, .backup-card > .task-heading, .backup-overview { align-items: stretch; flex-direction: column; }.task-search { width: 100%; }.task-card { padding: 16px 16px 0; }.task-heading { flex-wrap: wrap; }.task-facts { grid-template-columns: 1fr; gap: 14px; margin-left: 0; }.task-description { margin-left: 0; }.task-footer { flex-wrap: wrap; }.list-caption { align-items: flex-start; flex-direction: column; }.task-facts > div { gap: 4px; }.backup-overview .last-result { flex-wrap: wrap; }.execution-heading { align-items: flex-start; }.execution-heading .last-result { flex-wrap: wrap; }.history-toolbar { gap: 8px; flex-wrap: wrap; }.detail-grid { grid-template-columns: 1fr; }.task-footer code { max-width: 65%; } }
.maintenance-intro { display: flex; align-items: flex-start; gap: 16px; padding: 22px 24px; background: var(--el-fill-color-light); border-radius: 12px; }
.maintenance-intro h3 { font-size: 18px; margin: 2px 0 8px; }.maintenance-intro p { color: var(--el-text-color-secondary); font-size: 13px; line-height: 1.8; max-width: 850px; }
.intro-symbol { display: flex; align-items: center; justify-content: center; flex-shrink: 0; width: 48px; height: 48px; border-radius: 12px; color: var(--el-color-primary); background: var(--el-color-primary-light-9); }
.task-symbol { display: flex; align-items: center; justify-content: center; flex-shrink: 0; width: 36px; height: 36px; border: 1px solid var(--el-border-color-lighter); border-radius: 9px; color: var(--el-text-color-regular); background: var(--el-fill-color-light); }
.task-card > .task-heading { justify-content: flex-start; align-items: center; gap: 12px; }.list-caption { display: flex; gap: 10px; flex-wrap: wrap; font-size: 13px; }.backup-title { display: flex; align-items: flex-start; gap: 12px; }.task-facts > div + div { border-left: 1px solid var(--el-border-color-extra-light); padding-left: 16px; }.task-footer { min-height: 40px; }.task-actions .el-button { min-width: 72px; }
@media (max-width: 700px) { .maintenance-intro { padding: 16px; }.task-facts > div + div { border-left: 0; padding-left: 0; }.task-card { column-gap: 0; }.task-toolbar .toolbar-controls { width: 100%; } }
.heartbeat-note { grid-column: 1 / -1; display: flex; flex-wrap: wrap; gap: 12px; margin: 0 0 14px; font-size: 12px; color: var(--el-text-color-secondary); }.heartbeat-note strong { color: var(--el-color-warning); font-weight: 500; }
</style>
