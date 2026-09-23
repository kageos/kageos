<template>
  <section class="management-log">
    <p v-if="log.status === 'pending'" class="audit-warning">{{ t('managementLog.pendingHint') }}</p>
    <p v-if="log.status === 'failed'" class="audit-warning">{{ t('managementLog.failedHint') }}</p>
    <p v-if="details.error" class="audit-error">{{ details.error }}</p>
    <ul v-if="details.warnings?.length" class="audit-warning"><li v-for="(warning, index) in details.warnings" :key="index">{{ warning }}</li></ul>
    <div v-if="rows.length" class="audit-table-scroll">
      <table>
        <thead><tr><th>{{ t('managementLog.field') }}</th><th>{{ t('managementLog.before') }}</th><th>{{ t('managementLog.after') }}</th></tr></thead>
        <tbody><tr v-for="row in rows" :key="row.key"><th>{{ label(row.key) }}</th><td>{{ display(row.before) }}</td><td>{{ display(row.after) }}</td></tr></tbody>
      </table>
    </div>
    <dl>
      <template v-for="([key, value]) in detailEntries" :key="key"><dt>{{ label(key) }}</dt><dd>{{ key === 'stage' && te(`managementLog.stages.${value}`) ? t(`managementLog.stages.${value}`) : display(value) }}</dd></template>
    </dl>
    <p v-if="log.action.startsWith('timer.task.')" class="audit-hint">{{ t('managementLog.payloadHint') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { managementChanges } from '../composables/managementLog'

interface ManagementLog {
  action: string
  status?: string
  old_values_json?: Record<string, unknown> | null
  new_values_json?: Record<string, unknown> | null
  details_json?: Record<string, any>
}
const props = defineProps<{ log: ManagementLog }>()
const { t, te } = useI18n()
const details = computed(() => props.log.details_json || {})
const rows = computed(() => managementChanges(props.log.old_values_json, props.log.new_values_json))
const detailEntries = computed(() => Object.entries(details.value).filter(([key, value]) => !['warnings', 'error', 'duration_millis'].includes(key) && value !== null && value !== undefined && value !== ''))
function label(key: string): string { return te(`managementLog.fields.${key}`) ? t(`managementLog.fields.${key}`) : key }
function display(value: unknown): string {
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'boolean') return t(value ? 'managementLog.yes' : 'managementLog.no')
  if (Array.isArray(value)) return value.map(display).join('\n') || '—'
  if (typeof value === 'object') return JSON.stringify(value, null, 2)
  return String(value)
}
</script>

<style scoped>
.management-log { display: grid; gap: 12px; min-width: 0; }
.management-log p, .management-log dl { margin: 0; }
.audit-table-scroll { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; table-layout: fixed; }
th, td { padding: 8px 12px; text-align: left; vertical-align: top; border-bottom: 1px solid var(--el-border-color-lighter); overflow-wrap: anywhere; white-space: pre-wrap; }
th { font-weight: 500; }
thead { color: var(--el-text-color-secondary); }
dl { display: grid; grid-template-columns: minmax(100px, 160px) minmax(0, 1fr); gap: 8px 12px; }
dt { color: var(--el-text-color-secondary); }
dd { margin: 0; white-space: pre-wrap; overflow-wrap: anywhere; }
.audit-warning { color: var(--el-color-warning-dark-2); }
.audit-error { color: var(--el-color-danger); white-space: pre-wrap; overflow-wrap: anywhere; }
.audit-hint { color: var(--el-text-color-secondary); }
</style>
