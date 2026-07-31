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
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  listMyPermissionRequests,
  listPendingPermissionRequests,
} from '@/architecture/presentation/context/api/permission'
import { eventBus } from '@/architecture/presentation/context/eventBusContext'
import {
  getPermissionRequestWorkspaceRoot,
  summarizePermissionRequests,
  type PermissionRequestSummary,
} from '@/architecture/presentation/features/access/utils/permissionRequestSummary'

const props = defineProps<{
  label: string
  resourcePath: string
}>()

const { t } = useI18n()
const emptySummary = (): PermissionRequestSummary => ({
  ownPendingCount: 0,
  reviewPendingCount: 0,
  totalCount: 0,
})
const summary = ref<PermissionRequestSummary>(emptySummary())
let loadSequence = 0

const badgeTitle = computed(() => t('access.permissionRequestBadgeTitle', {
  review: summary.value.reviewPendingCount,
  mine: summary.value.ownPendingCount,
}))

async function loadSummary() {
  const resourcePath = props.resourcePath
  const root = getPermissionRequestWorkspaceRoot(resourcePath)
  const sequence = ++loadSequence
  if (!resourcePath || !root) {
    summary.value = emptySummary()
    return
  }

  const [mineResult, reviewResult] = await Promise.allSettled([
    listMyPermissionRequests(root, 'pending'),
    listPendingPermissionRequests(root),
  ])
  if (sequence !== loadSequence) return
  const mine = mineResult.status === 'fulfilled' ? mineResult.value.requests || [] : []
  const review = reviewResult.status === 'fulfilled' ? reviewResult.value.requests || [] : []
  summary.value = summarizePermissionRequests(resourcePath, mine, review)
}

watch(() => props.resourcePath, () => {
  summary.value = emptySummary()
  void loadSummary()
}, { immediate: true })

const unsubscribe = eventBus.on<{ resource_paths?: string[] }>('permission-request:changed', (payload) => {
  const paths = payload?.resource_paths || []
  if (paths.length === 0 || paths.includes(props.resourcePath)) {
    void loadSummary()
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
