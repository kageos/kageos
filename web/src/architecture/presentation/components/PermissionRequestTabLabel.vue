<template>
  <span class="permission-tab-label">
    <span>{{ label }}</span>
    <span
      v-if="summary.totalCount > 0"
      :class="['permission-tab-badge', { 'needs-review': summary.reviewPendingCount > 0 }]"
      :title="badgeTitle"
    >
      {{ summary.totalCount > 99 ? '99+' : summary.totalCount }}
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { eventBus } from '@/architecture/presentation/context/eventBusContext'
import {
  getPermissionRequestWorkspaceRoot,
  type PermissionRequestSummary,
} from '@/architecture/presentation/features/access/utils/permissionRequestSummary'
import {
  getPermissionRequestSummaryState,
  loadPermissionRequestSummary,
  permissionRequestPathSummary,
} from '@/architecture/presentation/features/access/utils/permissionRequestSummaryStore'

const props = defineProps<{
  label: string
  resourcePath: string
}>()

const { t } = useI18n()
const workspaceRoot = computed(() => getPermissionRequestWorkspaceRoot(props.resourcePath))
const workspaceSummary = computed(() => getPermissionRequestSummaryState(workspaceRoot.value))
const summary = computed<PermissionRequestSummary>(() => {
  const pathSummary = permissionRequestPathSummary(workspaceSummary.value, props.resourcePath)
  const ownPendingCount = Number(pathSummary.own_pending_count || 0)
  const reviewPendingCount = Number(pathSummary.review_pending_count || 0)
  return {
    ownPendingCount,
    reviewPendingCount,
    totalCount: ownPendingCount + reviewPendingCount,
  }
})

const badgeTitle = computed(() => t('access.permissionRequestBadgeTitle', {
  review: summary.value.reviewPendingCount,
  mine: summary.value.ownPendingCount,
}))

watch(workspaceRoot, (root) => {
  if (root) void loadPermissionRequestSummary(root)
}, { immediate: true })

const unsubscribe = eventBus.on<{ resource_paths?: string[] }>('permission-request:changed', (payload) => {
  const paths = payload?.resource_paths || []
  if (paths.length === 0 || paths.includes(props.resourcePath)) {
    void loadPermissionRequestSummary(workspaceRoot.value, { force: true })
  }
})

onBeforeUnmount(unsubscribe)
</script>

<style scoped>
.permission-tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.permission-tab-badge {
  min-width: 16px;
  height: 16px;
  padding: 0 5px;
  border-radius: 999px;
  background: #f59e0b;
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  line-height: 16px;
  text-align: center;
}

.permission-tab-badge.needs-review {
  background: #ef4444;
}
</style>
