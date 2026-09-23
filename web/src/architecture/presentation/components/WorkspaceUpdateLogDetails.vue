<template>
  <section class="workspace-update-details">
    <p class="update-summary">{{ workspaceUpdateSummary(log, t) }}</p>
    <template v-if="!details.write_only">
      <p v-if="log.status === 'failed' || workspaceUpdateHasWarnings(log)" class="update-warning">{{ t('workspaceUpdateLog.incompleteHint') }}</p>
      <p v-if="details.force_diff || details.changes?.mode === 'resync'">{{ t('workspaceUpdateLog.resyncHint') }}</p>
      <p v-else-if="details.changes" class="update-hint">{{ t('workspaceUpdateLog.scopeHint') }}</p>
      <div v-for="group in groups" :key="group.key" class="update-group">
        <h4>{{ t(`workspaceUpdateLog.${group.key}`) }} · {{ group.items.length }}</h4>
        <ul>
          <li v-for="item in group.items" :key="item.path">
            <span>{{ item.name || item.path }}</span>
            <span v-if="item.type" class="function-type">{{ typeLabel(item.type) }}</span>
            <code>{{ item.path }}</code>
          </li>
        </ul>
      </div>
    </template>
    <p v-if="details.error" class="update-error">{{ details.error }}</p>
    <ul v-if="details.warnings?.length" class="update-warning">
      <li v-for="(warning, index) in details.warnings" :key="index">{{ warning }}</li>
    </ul>
    <p v-if="details.change_description">{{ t('workspaceUpdateLog.description') }}: {{ details.change_description }}</p>
    <div class="update-references">
      <span v-if="details.git_commit_hash">Commit: {{ details.git_commit_hash }}</span>
      <span v-if="details.build_trace_id">Build trace: {{ details.build_trace_id }}</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { workspaceUpdateHasWarnings, workspaceUpdateSummary, type WorkspaceUpdateLog } from '../composables/workspaceUpdateLog'

const props = defineProps<{ log: WorkspaceUpdateLog }>()
const { t } = useI18n()
const details = computed(() => props.log.details_json || {})
const groups = computed(() => {
  const changes = details.value.changes
  if (!changes || details.value.write_only) return []
  const keys = details.value.force_diff || changes.mode === 'resync'
    ? ['synced'] as const : ['added', 'updated', 'deleted'] as const
  return keys.map(key => ({ key, items: changes[key] || [] })).filter(group => group.items.length)
})
function typeLabel(type: string): string {
  return ['form', 'table', 'chart', 'callback'].includes(type.toLowerCase())
    ? t(`workspaceUpdateLog.type_${type.toLowerCase()}`) : type
}
</script>

<style scoped>
.workspace-update-details { display: grid; gap: 12px; min-width: 0; }
.workspace-update-details p, .workspace-update-details h4 { margin: 0; }
.update-summary, .update-group h4 { font-weight: 600; }
.update-hint, .function-type, .update-references { color: var(--el-text-color-secondary); }
.update-warning { color: var(--el-color-warning-dark-2); white-space: pre-wrap; overflow-wrap: anywhere; }
.update-error { color: var(--el-color-danger); white-space: pre-wrap; overflow-wrap: anywhere; }
.update-group ul { margin: 8px 0 0; padding: 0; list-style: none; display: grid; gap: 8px; }
.update-group li { display: flex; gap: 8px; flex-wrap: wrap; align-items: baseline; }
.update-group code { width: 100%; overflow-wrap: anywhere; color: var(--el-text-color-secondary); }
.function-type { font-size: 12px; }
.update-references { display: flex; flex-wrap: wrap; gap: 12px; overflow-wrap: anywhere; }
</style>
