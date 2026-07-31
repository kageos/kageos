<template>
  <div v-if="node" class="permission-panel" :class="{ 'is-embedded': embedded }">
    <div class="access-layout">
      <section class="members-card">
        <div class="members-card-head">
          <div class="members-title-group">
            <div class="section-title">{{ t('access.title') }}</div>
            <div class="members-resource-path">{{ node.full_code_path }}</div>
          </div>
          <div class="members-actions">
            <el-button size="small" :loading="loading || requestLoading" @click="refreshPanel">{{ t('common.refresh') }}</el-button>
            <el-button size="small" type="primary" :icon="Plus" @click="goToGrantPage">
              {{ canAdmin(node) ? t('access.grantCurrent') : t('access.requestTab') }}
            </el-button>
          </div>
        </div>

        <el-tabs v-model="activeSection" class="permission-section-tabs">
          <el-tab-pane :label="t('access.permissionMembers')" name="members" :disabled="!canRead(node)" />
          <el-tab-pane v-if="canAdmin(node)" name="pending">
            <template #label>
              <span class="permission-section-label">
                {{ t('access.pendingTab') }}
                <span
                  v-if="requestCounts.pending > 0"
                  class="permission-section-count is-review"
                >
                  {{ formatBadgeCount(requestCounts.pending) }}
                </span>
              </span>
            </template>
          </el-tab-pane>
          <el-tab-pane name="mine">
            <template #label>
              <span class="permission-section-label">
                {{ t('access.myRequestsTab') }}
                <span
                  v-if="requestCounts.mine > 0"
                  class="permission-section-count"
                >
                  {{ formatBadgeCount(requestCounts.mine) }}
                </span>
              </span>
            </template>
          </el-tab-pane>
          <el-tab-pane v-if="canAdmin(node)" :label="t('access.reviewHistoryTab')" name="history" />
        </el-tabs>

        <el-table
          v-if="activeSection === 'members'"
          v-loading="loading"
          :data="assignments"
          class="access-table"
          :row-key="assignmentRowKey"
          size="small"
          :empty-text="t('access.empty')"
        >
          <el-table-column :label="t('access.principal')" min-width="220">
            <template #default="{ row }">
              <span v-if="row.principal_type === 'user'" class="member-users-cell">
                <UsersWidget
                  :value="principalUserValue(row.principal_key)"
                  :field="memberUsersField"
                  mode="response"
                  :field-path="`permissionPanelPrincipal:${assignmentRowKey(row)}`"
                />
              </span>
              <DepartmentDisplay
                v-else
                :full-code-path="row.principal_key"
                :display-name="departmentPrincipalLabel(row.principal_key)"
                mode="simple"
                size="small"
              />
            </template>
          </el-table-column>
          <el-table-column :label="t('access.role')" width="108">
            <template #default="{ row }">
              <el-tag size="small" :type="roleTagType(row.role_code)">
                {{ roleLabel(row.role_code) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('functionTabs.permission')" min-width="190">
            <template #default="{ row }">
              <div class="permission-tags">
                <el-tag
                  v-for="permission in permissionLabels(row.permissions)"
                  :key="permission"
                  size="small"
                  effect="plain"
                >
                  {{ permission }}
                </el-tag>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('access.source')" min-width="170">
            <template #default="{ row }">
              <span v-if="isDirectMember(row)" class="source-current">{{ t('common.currentResource') }}</span>
              <span v-else class="source-inherited">{{ row.inherited_from || row.resource_path }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="t('access.expiresColumn')" width="150">
            <template #default="{ row }">
              {{ formatExpiresAt(row.expires_at) }}
            </template>
          </el-table-column>
        </el-table>

        <PermissionRequestRecordsPanel
          v-else
          ref="requestRecordsRef"
          :resource-path="node.full_code_path"
          :view="activeRequestView"
          @changed="handleRequestChanged"
          @count-change="handleRequestCountChange"
          @loading-change="requestLoading = $event"
        />
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import type { AccessPermissions, AccessRoleCode, ServiceTree } from '@/architecture/domain/types'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'
import { WidgetType } from '@/architecture/domain/constants/widget'
import {
  listPermissionAssignments,
  type RoleAssignment
} from '@/architecture/presentation/context/api/permission'
import { canAdmin, canRead } from '@/architecture/presentation/composables/useAccessControl'
import DepartmentDisplay from '@/architecture/presentation/shared/components/DepartmentDisplay.vue'
import UsersWidget from '@/architecture/presentation/shared/components/UsersWidget.vue'
import { createStringFieldValue } from '@/architecture/domain/utils/widgetFieldHelpers'
import PermissionRequestRecordsPanel from './PermissionRequestRecordsPanel.vue'

type PermissionRequestView = 'pending' | 'mine' | 'history'
type PermissionSection = 'members' | PermissionRequestView

const props = withDefaults(defineProps<{
  node: ServiceTree | null
  embedded?: boolean
}>(), {
  embedded: false
})

const emit = defineEmits<{
  (e: 'changed'): void
}>()

const { t } = useI18n()
const router = useRouter()

const roleLabelMap = computed<Record<AccessRoleCode, string>>(() => ({
  viewer: t('access.roleViewerTitle'),
  member: t('access.roleMemberTitle'),
  admin: t('access.roleAdminTitle'),
  owner: t('access.roleOwnerTitle'),
}))

const permissionLabelMap = {
  read: 'read',
  write: 'write',
  update: 'update',
  delete: 'delete',
  admin: 'admin',
  owner: 'owner',
} satisfies Record<string, string>

const memberUsersField = computed<FieldConfig>(() => ({
  code: 'permissionPanelMemberUsers',
  name: t('access.member'),
  desc: t('access.member'),
  widget: {
    type: WidgetType.USERS,
    config: {
      max_display_count: 1
    }
  },
  data: {
    type: 'string'
  }
}))

const assignments = ref<RoleAssignment[]>([])
const loading = ref(false)
const requestLoading = ref(false)
const activeSection = ref<PermissionSection>('members')
const requestRecordsRef = ref<{ loadRequests: () => void } | null>(null)
const requestCounts = reactive<Record<'pending' | 'mine', number>>({
  pending: 0,
  mine: 0,
})
let assignmentLoadSequence = 0
const activeRequestView = computed<PermissionRequestView>(() => (
  activeSection.value === 'members' ? 'mine' : activeSection.value
))

watch(
  () => [props.node?.full_code_path || '', canRead(props.node), canAdmin(props.node)] as const,
  ([path, readable]) => {
    assignmentLoadSequence += 1
    loading.value = false
    assignments.value = []
    requestCounts.pending = 0
    requestCounts.mine = 0
    requestLoading.value = false
    if (path) {
      if (readable) {
        const alreadyShowingMembers = activeSection.value === 'members'
        activeSection.value = 'members'
        if (alreadyShowingMembers) {
          void loadAssignments()
        }
      } else {
        activeSection.value = 'mine'
      }
    }
  },
  { immediate: true }
)

watch(activeSection, (section) => {
  if (section === 'members' && canRead(props.node)) {
    void loadAssignments()
  }
})

async function loadAssignments() {
  const path = props.node?.full_code_path
  const sequence = ++assignmentLoadSequence
  if (!path) {
    assignments.value = []
    return
  }

  loading.value = true
  try {
    const resp = await listPermissionAssignments(path)
    if (sequence !== assignmentLoadSequence) return
    assignments.value = resp.assignments || []
  } catch (error: any) {
    if (sequence === assignmentLoadSequence) {
      const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || t('access.loadFailed')
      ElMessage.error(message)
    }
  } finally {
    if (sequence === assignmentLoadSequence) {
      loading.value = false
    }
  }
}

function refreshPanel() {
  if (activeSection.value !== 'members') {
    requestRecordsRef.value?.loadRequests()
    return
  }
  void loadAssignments()
}

function handleRequestChanged() {
  emit('changed')
}

function handleRequestCountChange(payload: { view: PermissionRequestView; count: number }) {
  if (payload.view === 'pending' || payload.view === 'mine') {
    requestCounts[payload.view] = payload.count
  }
}

function formatBadgeCount(count: number): string {
  return count > 99 ? '99+' : String(count)
}

function goToGrantPage() {
  const path = props.node?.full_code_path
  if (!path) {
    return
  }
  void router.push({
    path: '/permissions',
    query: {
      resource: path,
      ...(canAdmin(props.node) ? {} : { mode: 'request' }),
    }
  })
}

function assignmentRowKey(assignment: RoleAssignment): string {
  return `${assignment.principal_type}:${assignment.principal_key}:${assignment.resource_path}:${assignment.role_code}`
}

function roleLabel(role: AccessRoleCode): string {
  return roleLabelMap.value[role] || role
}

function principalUserValue(username: string): FieldValue {
  return createStringFieldValue(memberUsersField.value, username || '', { emptyRaw: '' })
}

function departmentPrincipalLabel(path: string): string | null {
  return path === '/org' ? t('access.allMembers') : null
}

function roleTagType(role: AccessRoleCode): 'danger' | 'warning' | 'success' | 'info' {
  if (role === 'owner') return 'danger'
  if (role === 'admin') return 'warning'
  if (role === 'member') return 'success'
  return 'info'
}

function isDirectMember(assignment: RoleAssignment): boolean {
  return assignment.direct !== false && assignment.source !== 'inherited'
}

function permissionLabels(permissions: AccessPermissions | null | undefined): string[] {
  if (!permissions) return []
  const labels = Object.entries(permissionLabelMap)
    .filter(([key]) => permissions[key as keyof AccessPermissions])
    .map(([, label]) => label)
  return labels.length > 0 ? labels : [t('access.noPermission')]
}

function formatExpiresAt(value?: string): string {
  if (!value) return t('access.permanent')
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

defineExpose({
  loadAssignments,
  refreshPanel,
})
</script>

<style scoped lang="scss">
.permission-panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-height: 0;
}

.permission-panel.is-embedded {
  height: 100%;
}

.access-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 14px;
  min-height: 0;
}

.members-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
  padding: 14px;
  min-width: 0;
}

.section-title {
  margin-bottom: 12px;
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 700;
}

.members-card {
  min-width: 0;
}

.members-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.members-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.members-title-group {
  min-width: 0;
}

.members-title-group .section-title {
  margin-bottom: 2px;
}

.members-resource-path {
  max-width: min(720px, 52vw);
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.access-table {
  width: 100%;
}

.permission-section-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.permission-section-count {
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

.permission-section-count.is-review {
  background: #ef4444;
}

.member-users-cell {
  display: inline-flex;
  align-items: center;
  min-width: 0;

  :deep(.users-response) {
    width: auto;
  }

  :deep(.users-list-horizontal) {
    gap: 6px;
  }

  :deep(.user-display-card-trigger) {
    max-width: 180px;
  }

  :deep(.user-name) {
    max-width: 128px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.permission-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.source-current {
  color: var(--el-text-color-primary);
}

.source-inherited {
  color: var(--el-text-color-secondary);
  word-break: break-all;
}

.disabled-action {
  color: var(--el-text-color-placeholder);
}

@media (max-width: 980px) {
  .access-layout {
    grid-template-columns: 1fr;
  }
}
</style>
