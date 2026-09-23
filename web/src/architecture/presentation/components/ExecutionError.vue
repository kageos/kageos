<template>
  <div class="execution-error">
    <strong>{{ t('maintenance.failureReason') }}</strong>
    <div v-for="item in groups.slice(0, 3)" :key="item.message" class="error-line"><span>{{ item.message }}</span><b v-if="item.count > 1">× {{ item.count }}</b></div>
    <p v-if="groups.length > 3">{{ t('maintenance.moreErrors', {count: groups.length - 3}) }}</p>
    <details><summary>{{ t('maintenance.originalError') }}</summary><pre>{{ message }}</pre></details>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
const props = defineProps<{ message: string }>()
const { t } = useI18n()
const groups = computed(() => {
 const counts = new Map<string, number>()
 for (const line of props.message.split(/\r?\n/).map(line => line.trim()).filter(Boolean)) counts.set(line, (counts.get(line) || 0) + 1)
 return [...counts].map(([message, count]) => ({ message, count }))
})
</script>
<style scoped>
.execution-error { grid-column: 1 / -1; padding: 14px 16px; margin-bottom: 14px; border-radius: 8px; background: color-mix(in srgb, var(--el-color-danger) 9%, var(--el-bg-color)); border-left: 3px solid var(--el-color-danger); font-size: 12px; color: var(--el-text-color-primary); line-height: 1.7; min-width: 0; }
.execution-error > strong { display: block; margin-bottom: 6px; color: var(--el-color-danger); }.error-line { display: flex; align-items: start; gap: 12px; }.error-line span { display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden; overflow-wrap: anywhere; }.error-line b { white-space: nowrap; padding: 0 6px; border-radius: 4px; background: var(--el-bg-color); }details { margin-top: 8px; }summary { cursor: pointer; width: fit-content; }pre { max-height: 260px; overflow: auto; white-space: pre-wrap; overflow-wrap: anywhere; margin: 10px 0 0; }
</style>
