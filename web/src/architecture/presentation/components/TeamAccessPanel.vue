<template>
  <div v-if="node" class="team-access-panel" :class="{ 'is-embedded': embedded }">
    <div class="access-layout">
      <section class="members-card">
        <div class="members-card-head">
          <div class="members-title-group">
            <div class="section-title">{{ t('access.title') }}</div>
            <div class="members-resource-path">{{ node.full_code_path }}</div>
          </div>
          <el-button size="small" :loading="loading" @click="loadMembers">{{ t('common.refresh') }}</el-button>
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
          <el-table-column :label="t('common.operation')" width="86" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="isDirectMember(row)"
                type="danger"
                link
                size="small"
                :icon="Delete"
                :loading="removingKey === memberRowKey(row)"
                @click="removeMember(row)"
              >
                {{ t('common.remove') }}
              </el-button>
              <span v-else class="disabled-action">{{ t('access.inherited') }}</span>
            </template>
          </el-table-column>
        </el-table>
      </section>

      <section class="grant-card">
        <div class="section-title">{{ t('access.grantCurrent') }}</div>
        <div class="resource-summary">
          <div class="resource-title">{{ node.name || t('common.currentResource') }}</div>
          <div class="resource-path">{{ node.full_code_path }}</div>
        </div>

        <el-form label-position="top" class="grant-form" @submit.prevent>
          <el-form-item :label="t('access.member')">
            <UsersWidget
              :value="grantUsersValue"
              :field="grantUsersField"
              mode="edit"
              field-path="teamAccessPanelUsers"
              @update:modelValue="handleGrantUsersChange"
            />
          </el-form-item>

          <el-form-item :label="t('access.role')">
            <div class="role-cards">
              <button
                v-for="role in roleOptions"
                :key="role.value"
                type="button"
                class="role-card"
                :class="[`tone-${role.tone}`, { 'is-selected': grantRole === role.value }]"
                :aria-pressed="grantRole === role.value"
                @click="grantRole = role.value"
              >
                <span class="role-card-head">
                  <span class="role-icon-badge">
                    <el-icon><component :is="role.icon" /></el-icon>
                  </span>
                  <span class="role-card-copy">
                    <strong>{{ role.title }}</strong>
                    <em>{{ role.codeLabel }} · {{ role.subtitle }}</em>
                  </span>
                  <span class="role-state" :class="{ 'is-selected': grantRole === role.value }">
                    <el-icon v-if="grantRole === role.value"><CircleCheck /></el-icon>
                    <span v-else class="role-empty-dot" />
                  </span>
                </span>
                <span class="role-description">{{ role.description }}</span>
                <span class="role-action-grid">
                  <span
                    v-for="permission in role.permissions"
                    :key="permission"
                    class="role-action-item"
                  >
                    {{ permission }}
                  </span>
                </span>
              </button>
            </div>
          </el-form-item>

          <el-form-item :label="t('access.expires')">
            <el-radio-group v-model="grantPermanent">
              <el-radio :label="true">{{ t('access.permanent') }}</el-radio>
              <el-radio :label="false">{{ t('access.customTime') }}</el-radio>
            </el-radio-group>
            <el-date-picker
              v-if="!grantPermanent"
              v-model="grantExpiresAt"
              class="expires-picker"
              type="datetime"
              :placeholder="t('access.expiresPlaceholder')"
              clearable
            />
          </el-form-item>

          <el-button
            type="primary"
            class="submit-button"
            :icon="Plus"
            :loading="submitting"
            :disabled="!canSubmitGrant"
            @click="submitGrant"
          >
            {{ t('access.submitGrant') }}
          </el-button>
        </el-form>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { CircleCheck, Delete, EditPen, Key, Plus, User, View } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { Component } from 'vue'
import type { AccessPermissions, AccessRoleCode, ServiceTree } from '@/architecture/domain/types'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'
import { WidgetType } from '@/architecture/domain/constants/widget'
import {
  batchAssignTeamRoles,
  listTeamMembers,
  removeTeamRole,
  type TeamMemberAccess
} from '@/architecture/presentation/context/api/team-access'
import UsersWidget from '@/architecture/presentation/shared/components/UsersWidget.vue'
import { createStringFieldValue, extractStringFieldRaw } from '@/architecture/domain/utils/widgetFieldHelpers'

interface RoleOption {
  value: AccessRoleCode
  title: string
  codeLabel: string
  subtitle: string
  description: string
  permissions: string[]
  tone: string
  icon: Component
}

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

const roleOptions = computed<RoleOption[]>(() => [
  {
    value: 'viewer',
    title: t('access.roleViewerTitle'),
    codeLabel: 'Viewer',
    subtitle: t('access.roleViewerSubtitle'),
    description: t('access.roleViewerDescription'),
    permissions: ['read'],
    tone: 'view',
    icon: View,
  },
  {
    value: 'member',
    title: t('access.roleMemberTitle'),
    codeLabel: 'Member',
    subtitle: t('access.roleMemberSubtitle'),
    description: t('access.roleMemberDescription'),
    permissions: ['read', 'write', 'update'],
    tone: 'edit',
    icon: EditPen,
  },
  {
    value: 'admin',
    title: t('access.roleAdminTitle'),
    codeLabel: 'Admin',
    subtitle: t('access.roleAdminSubtitle'),
    description: t('access.roleAdminDescription'),
    permissions: ['read', 'write', 'update', 'delete', 'admin'],
    tone: 'admin',
    icon: Key,
  },
  {
    value: 'owner',
    title: t('access.roleOwnerTitle'),
    codeLabel: 'Owner',
    subtitle: t('access.roleOwnerSubtitle'),
    description: t('access.roleOwnerDescription'),
    permissions: ['read', 'write', 'update', 'delete', 'admin', 'owner'],
    tone: 'owner',
    icon: User,
  },
])

const permissionLabelMap = {
  read: 'read',
  write: 'write',
  update: 'update',
  delete: 'delete',
  admin: 'admin',
  owner: 'owner',
} satisfies Record<string, string>

const grantUsersField = computed<FieldConfig>(() => ({
  code: 'teamAccessPanelUsers',
  name: t('access.member'),
  desc: t('access.memberPickerDesc'),
  widget: {
    type: WidgetType.USERS,
    config: {
      max_count: 50
    }
  },
  data: {
    type: 'string'
  }
}))

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
const submitting = ref(false)
const removingKey = ref('')
const grantUsersValue = ref<FieldValue>(createStringFieldValue(grantUsersField.value, '', { emptyRaw: '' }))
const grantRole = ref<AccessRoleCode>('viewer')
const grantPermanent = ref(true)
const grantExpiresAt = ref<Date | null>(null)

const selectedUsernames = computed(() => {
  return extractStringFieldRaw(grantUsersValue.value)
    .split(',')
    .map(item => item.trim())
    .filter(Boolean)
})

const canSubmitGrant = computed(() => {
  return Boolean(props.node?.full_code_path && selectedUsernames.value.length > 0 && grantRole.value)
})

watch(
  () => props.node?.full_code_path || '',
  (path) => {
    grantUsersValue.value = createStringFieldValue(grantUsersField.value, '', { emptyRaw: '' })
    grantRole.value = 'viewer'
    grantPermanent.value = true
    grantExpiresAt.value = null
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

function handleGrantUsersChange(value: FieldValue) {
  grantUsersValue.value = value
}

async function submitGrant() {
  const path = props.node?.full_code_path
  if (!path || !canSubmitGrant.value) {
    ElMessage.warning(t('access.selectMemberAndRole'))
    return
  }

  submitting.value = true
  try {
    await batchAssignTeamRoles({
      resource_paths: [path],
      usernames: selectedUsernames.value,
      role_codes: [grantRole.value],
      expires_at: grantPermanent.value ? null : (grantExpiresAt.value ? grantExpiresAt.value.toISOString() : null)
    })
    ElMessage.success(t('access.grantSuccess', { count: selectedUsernames.value.length, role: roleLabel(grantRole.value) }))
    grantUsersValue.value = createStringFieldValue(grantUsersField.value, '', { emptyRaw: '' })
    await loadMembers()
    emit('changed')
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || t('access.grantFailed')
    ElMessage.error(message)
  } finally {
    submitting.value = false
  }
}

async function removeMember(member: TeamMemberAccess) {
  const key = memberRowKey(member)
  try {
    await ElMessageBox.confirm(
      t('access.removeConfirm', { username: member.username, role: roleLabel(member.role_code) }),
      t('access.removeConfirmTitle'),
      {
        confirmButtonText: t('common.remove'),
        cancelButtonText: t('common.cancel'),
        type: 'warning',
      }
    )
  } catch {
    return
  }

  removingKey.value = key
  try {
    await removeTeamRole({
      resource_path: member.resource_path,
      username: member.username,
      role_code: member.role_code
    })
    ElMessage.success(t('access.removeSuccess'))
    await loadMembers()
    emit('changed')
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || t('access.removeFailed')
    ElMessage.error(message)
  } finally {
    removingKey.value = ''
  }
}

function memberRowKey(member: TeamMemberAccess): string {
  return `${member.username}:${member.resource_path}:${member.role_code}`
}

function roleLabel(role: AccessRoleCode): string {
  return roleOptions.value.find(option => option.value === role)?.title || role
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
  grid-template-columns: minmax(0, 1fr) minmax(280px, 360px);
  gap: 14px;
  min-height: 0;
}

.grant-card,
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

.grant-form {
  :deep(.el-form-item) {
    margin-bottom: 14px;
  }
}

.resource-summary {
  padding: 10px 12px;
  margin-bottom: 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.resource-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.resource-path {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}

.role-cards {
  display: grid;
  gap: 10px;
}

.role-card {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
  padding: 12px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.14s ease, background-color 0.14s ease, box-shadow 0.14s ease;
  overflow: hidden;

  &::before {
    content: '';
    position: absolute;
    inset: 0 auto 0 0;
    width: 4px;
    background: transparent;
    transition: background-color 0.16s ease;
  }

  &:hover {
    border-color: rgba(var(--el-color-primary-rgb), 0.5);
    background: color-mix(in srgb, var(--el-bg-color) 94%, var(--el-color-primary) 6%);
  }

  &.is-selected {
    border-color: var(--el-color-primary);
    background: color-mix(in srgb, var(--el-bg-color) 90%, var(--el-color-primary) 10%);
    box-shadow: inset 0 0 0 1px rgba(var(--el-color-primary-rgb), 0.12);

    &::before {
      background: var(--el-color-primary);
    }

    .role-icon-badge {
      background: rgba(var(--el-color-primary-rgb), 0.16);
      color: var(--el-color-primary);
    }

    .role-card-copy strong {
      color: var(--el-color-primary);
      font-weight: 700;
    }

    .role-action-item {
      border-color: rgba(var(--el-color-primary-rgb), 0.18);
      background: rgba(var(--el-color-primary-rgb), 0.08);
      color: var(--el-color-primary);
    }
  }
}

.role-card-head {
  display: flex;
  align-items: center;
  gap: 10px;
}

.role-icon-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: rgba(var(--el-color-primary-rgb), 0.1);
  color: var(--el-color-primary);
  transition: background-color 0.16s ease, color 0.16s ease, box-shadow 0.16s ease;
}

.role-card-copy {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;

  strong {
    font-size: 14px;
  }

  em {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    font-style: normal;
  }
}

.role-state {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border: 1px solid var(--el-border-color);
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-placeholder);
  flex: 0 0 22px;
  transition: all 0.16s ease;
}

.role-state.is-selected {
  border-color: var(--el-color-primary);
  background: rgba(var(--el-color-primary-rgb), 0.1);
  color: var(--el-color-primary);
}

.role-empty-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: var(--el-border-color);
}

.role-description {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.55;
}

.role-action-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.role-action-item {
  padding: 3px 7px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 12px;
  transition: border-color 0.16s ease, background-color 0.16s ease, color 0.16s ease;
}

.expires-picker {
  width: 100%;
  margin-top: 10px;
}

.submit-button {
  width: 100%;
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

.members-title-group {
  min-width: 0;
}

.members-title-group .section-title {
  margin-bottom: 2px;
}

.members-resource-path {
  max-width: 260px;
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
