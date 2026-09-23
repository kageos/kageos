<template>
  <section class="resource-logs-panel">
    <el-radio-group
      v-if="canManageArchives"
      :model-value="activeView"
      :aria-label="t('functionTabs.operateLog')"
      class="log-view-switch"
      @change="$emit('change', $event === 'logArchives' ? 'logArchives' : 'operateLog')"
    >
      <el-radio-button value="operateLog">{{ t('logStorage.current') }}</el-radio-button>
      <el-radio-button value="logArchives">{{ t('logStorage.archives') }}</el-radio-button>
    </el-radio-group>
    <ResourceLogArchives
      v-if="activeView === 'logArchives' && canManageArchives"
      :resource-path="resourcePath"
    />
    <slot v-else />
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ResourceLogArchives from './ResourceLogArchives.vue'

defineProps<{
  resourcePath: string
  canManageArchives: boolean
  activeView: 'operateLog' | 'logArchives'
}>()
defineEmits<{ (e: 'change', value: 'operateLog' | 'logArchives'): void }>()
const { t } = useI18n()
</script>

<style scoped>
.resource-logs-panel { min-width: 0; }
.log-view-switch { margin-bottom: 18px; }
</style>
