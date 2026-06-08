<template>
  <div v-if="node" class="team-access-panel" :class="{ 'is-embedded': embedded }">
    <div class="access-layout">
      <section class="members-card">
        <div class="members-card-head">
          <div class="members-title-group">
            <div class="section-title">{{ t('access.title') }}</div>
            <div class="members-resource-path">{{ node.full_code_path }}</div>
          </div>
          <div class="members-actions">
            <el-button size="small" :loading="loading" @click="loadMembers">{{ t('common.refresh') }}</el-button>
            <el-button size="small" type="primary" :icon="Plus" @click="goToGrantPage">
              {{ t('access.grantCurrent') }}
            </el-button>
          </div>
        </div>

        <el-table
          v-loading="loading"
          :data="members"
          class="access-table"
          :row-key="memberRowKey"
          size="small"
          :empty-text="t('access.empty')"
        >
          <el-table-column :label="t('access.member')" min-width="200">
            <template #default="{ row }">
              <span class="member-users-cell">
                <UsersWidget
                  :value="memberUsersValue(row.username)"
                  :field="memberUsersField"
                  mode="response"
                  :field-path="`teamAccessPanelMember:${memberRowKey(row)}`"
                />
              </span>
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
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import type { AccessPermissions, AccessRoleCode, ServiceTree } from '@/architecture/domain/types'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'
import { WidgetType } from '@/architecture/domain/constants/widget'
import {
  listTeamMembers,
  type TeamMemberAccess
} from '@/architecture/presentation/context/api/team-access'
import UsersWidget from '@/architecture/presentation/shared/components/UsersWidget.vue'
import { createStringFieldValue } from '@/architecture/domain/utils/widgetFieldHelpers'

const props = withDefaults(defineProps<{
  node: ServiceTree | null
  embedded?: boolean
}>(), {
  embedded: false
})

defineEmits<{
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
  code: 'teamAccessPanelMemberUsers',
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

const members = ref<TeamMemberAccess[]>([])
const loading = ref(false)

watch(
  () => props.node?.full_code_path || '',
  (path) => {
    if (path) {
      void loadMembers()
    } else {
      members.value = []
    }
  },
  { immediate: true }
)

async function loadMembers() {
  const path = props.node?.full_code_path
  if (!path) {
    members.value = []
    return
  }

  loading.value = true
  try {
    const resp = await listTeamMembers(path)
    members.value = resp.members || []
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || t('access.loadFailed')
    ElMessage.error(message)
  } finally {
    loading.value = false
  }
}

function goToGrantPage() {
  const path = props.node?.full_code_path
  if (!path) {
    return
  }
  void router.push({
    path: '/permissions/access',
    query: { resource: path }
  })
}

function memberRowKey(member: TeamMemberAccess): string {
  return `${member.username}:${member.resource_path}:${member.role_code}`
}

function roleLabel(role: AccessRoleCode): string {
  return roleLabelMap.value[role] || role
}

function memberUsersValue(username: string): FieldValue {
  return createStringFieldValue(memberUsersField.value, username || '', { emptyRaw: '' })
}

function roleTagType(role: AccessRoleCode): 'danger' | 'warning' | 'success' | 'info' {
  if (role === 'owner') return 'danger'
  if (role === 'admin') return 'warning'
  if (role === 'member') return 'success'
  return 'info'
}

function isDirectMember(member: TeamMemberAccess): boolean {
  return member.direct !== false && member.source !== 'inherited'
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
  loadMembers
})
</script>

<style scoped lang="scss">
.team-access-panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-height: 0;
}

.team-access-panel.is-embedded {
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
