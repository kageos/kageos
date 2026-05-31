<template>
  <main class="team-access-page">
    <header class="page-header">
      <div class="page-title-area">
        <el-button class="back-button" text :icon="ArrowLeft" @click="goBack">
          {{ t('access.backToWorkspace') }}
        </el-button>
        <div>
          <div class="page-kicker">{{ t('access.title') }}</div>
          <h1>{{ activeResource?.name || appName || t('access.title') }}</h1>
          <p>{{ activeResourcePath || requestedResourcePath || t('access.invalidResource') }}</p>
        </div>
      </div>
      <div class="page-actions">
        <el-button :loading="pageLoading" @click="reloadPage">{{ t('common.refresh') }}</el-button>
        <el-button type="primary" plain @click="openActiveResource">
          {{ t('access.openResource') }}
        </el-button>
      </div>
    </header>

    <el-alert
      v-if="invalidResource"
      class="page-alert"
      type="warning"
      :title="t('access.invalidResource')"
      :closable="false"
      show-icon
    />

    <el-alert
      v-else-if="loadError"
      class="page-alert"
      type="error"
      :title="loadError"
      :closable="false"
      show-icon
    />

    <section v-else v-loading="pageLoading" class="access-page-body">
      <div class="grant-flow">
        <section class="resource-panel">
          <div class="panel-header">
            <div>
              <h3>{{ t('access.selectResource') }}</h3>
              <p>{{ t('access.selectResourceDesc') }}</p>
            </div>
            <el-tag size="small" effect="plain">
              {{ t('common.selectedCount', { count: selectedResourcePaths.length }) }}
            </el-tag>
          </div>

          <div class="tree-container">
            <el-tree
              ref="treeRef"
              class="resource-tree"
              :data="treeData"
              :props="treeProps"
              node-key="full_code_path"
              show-checkbox
              :check-strictly="true"
              :default-expand-all="true"
              :expand-on-click-node="false"
              :check-on-click-node="true"
              :highlight-current="true"
              :current-node-key="activeResourcePath"
              @node-click="handleResourceClick"
              @check="handleResourceCheck"
            >
              <template #default="{ node: treeNode, data }">
                <span class="tree-node" :class="{ 'is-selected': activeResourcePath === data.full_code_path }">
                  <img
                    v-if="data.type === 'package'"
                    src="/service-tree/custom-folder.svg"
                    :alt="t('packageDetail.detail')"
                    class="node-icon package-icon-img"
                    :class="getNodeIconClass(data)"
                  />
                  <template v-else-if="data.type === 'function'">
                    <img
                      v-if="data.template_type === TEMPLATE_TYPE.FORM"
                      src="/service-tree/编辑.svg"
                      :alt="t('packageDetail.form')"
                      class="node-icon form-icon-img"
                      :class="getNodeIconClass(data)"
                    />
                    <component
                      v-else
                      :is="getFunctionIcon(data)"
                      class="node-icon"
                      :class="getNodeIconClass(data)"
                    />
                  </template>
                  <img
                    v-else-if="data.type === 'docs'"
                    src="/文档.svg"
                    :alt="t('packageDetail.docs')"
                    class="node-icon docs-icon-img"
                    :class="getNodeIconClass(data)"
                  />
                  <span v-else class="node-icon fx-icon" :class="getNodeIconClass(data)">fx</span>
                  <span class="node-label">{{ treeNode.label }}</span>
                </span>
              </template>
            </el-tree>
          </div>
        </section>

        <section class="role-panel">
          <div class="panel-header">
            <div>
              <h3>{{ t('access.selectRole') }}</h3>
              <p>{{ t('access.selectRoleDesc') }}</p>
            </div>
          </div>

          <div class="resource-summary">
            <div class="resource-title">{{ activeResource?.name || t('common.currentResource') }}</div>
            <div class="resource-path">{{ activeResourcePath }}</div>
          </div>

          <div class="role-cards">
            <button
              v-for="role in roleOptions"
              :key="role.value"
              class="role-card"
              :class="[`tone-${role.tone}`, { 'is-selected': grantRole === role.value }]"
              :aria-pressed="grantRole === role.value"
              type="button"
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
        </section>

        <section class="grant-panel">
          <div class="panel-header">
            <div>
              <h3>{{ t('access.grantInfo') }}</h3>
              <p>{{ t('access.grantInfoDesc') }}</p>
            </div>
          </div>

          <el-form label-position="top" class="grant-form" @submit.prevent>
            <el-form-item :label="t('access.grantUser')">
              <UsersWidget
                :value="grantUsersValue"
                :field="grantUsersField"
                mode="edit"
                field-path="teamAccessPageUsers"
                @update:modelValue="handleGrantUsersChange"
              />
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

            <div class="grant-preview">
              <div class="preview-row">
                <span>{{ t('access.resource') }}</span>
                <strong>{{ t('common.resourceCount', { count: selectedResourcePaths.length }) }}</strong>
              </div>
              <div class="preview-row">
                <span>{{ t('access.users') }}</span>
                <strong>{{ t('common.userCount', { count: selectedUsernames.length }) }}</strong>
              </div>
              <div class="preview-row">
                <span>{{ t('access.role') }}</span>
                <strong>{{ roleLabel(grantRole) }}</strong>
              </div>
            </div>

            <el-button
              class="submit-button"
              type="primary"
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

      <section class="members-panel">
        <div class="members-header">
          <div>
            <h3>{{ t('access.existing') }}</h3>
            <p>{{ activeResourcePath }}</p>
          </div>
          <el-button size="small" :loading="membersLoading" @click="loadMembers">
            {{ t('common.refresh') }}
          </el-button>
        </div>

        <el-tabs v-model="activeTab" class="access-tabs">
          <el-tab-pane :label="t('access.currentDirectoryCount', { count: currentMembers.length })" name="current" />
          <el-tab-pane :label="t('access.inheritedParentCount', { count: inheritedMembers.length })" name="inherited" />
        </el-tabs>

        <el-table
          v-loading="membersLoading"
          :data="visibleMembers"
          class="access-table"
          :row-key="memberRowKey"
          size="small"
          :empty-text="t('access.empty')"
        >
          <el-table-column :label="t('access.member')" min-width="220">
            <template #default="{ row }">
              <div class="member-users-cell">
                <UsersWidget
                  :value="memberUsersValue(row.username)"
                  :field="memberUsersField"
                  mode="response"
                  :field-path="`teamAccessPageMember:${memberRowKey(row)}`"
                />
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('access.role')" width="116">
            <template #default="{ row }">
              <el-tag size="small" :type="roleTagType(row.role_code)">
                {{ roleLabel(row.role_code) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('functionTabs.permission')" min-width="220">
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
          <el-table-column :label="t('access.source')" min-width="190">
            <template #default="{ row }">
              <span v-if="row.direct" class="source-current">{{ t('access.currentDirectory') }}</span>
              <span v-else class="source-inherited">{{ row.inherited_from || row.resource_path }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="t('access.expiresColumn')" width="150">
            <template #default="{ row }">
              {{ formatExpiresAt(row.expires_at) }}
            </template>
          </el-table-column>
          <el-table-column :label="t('common.operation')" width="92" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="row.direct"
                type="danger"
                link
                size="small"
                :icon="Delete"
                :loading="removingKey === memberRowKey(row)"
                @click="removeMember(row)"
              >
                {{ t('common.remove') }}
              </el-button>
              <el-tooltip v-else :content="t('access.inheritedRemoveTip')" placement="top">
                <span class="disabled-action">{{ t('access.inherited') }}</span>
              </el-tooltip>
            </template>
          </el-table-column>
        </el-table>
      </section>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  CircleCheck,
  Delete,
  Document,
  EditPen,
  Key,
  Plus,
  User,
  View
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { Component } from 'vue'
import type { AccessPermissions, AccessRoleCode, App, ServiceTree } from '@/architecture/domain/types'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'
import { TEMPLATE_TYPE } from '@/architecture/domain/constants/functionTypes'
import { WidgetType } from '@/architecture/domain/constants/widget'
import { getAppWithServiceTree } from '@/architecture/presentation/context/api/app'
import {
  batchAssignTeamRoles,
  listTeamMembers,
  removeTeamRole,
  type TeamMemberAccess
} from '@/architecture/presentation/context/api/team-access'
import ChartIcon from '@/architecture/presentation/shared/components/icons/ChartIcon.vue'
import FormIcon from '@/architecture/presentation/shared/components/icons/FormIcon.vue'
import TableIcon from '@/architecture/presentation/shared/components/icons/TableIcon.vue'
import UsersWidget from '@/architecture/presentation/shared/components/UsersWidget.vue'
import { createStringFieldValue, extractStringFieldRaw } from '@/architecture/domain/utils/widgetFieldHelpers'
import { buildAppResourcePath, normalizeResourcePath, parseResourcePath } from '@/architecture/shared/resourcePath'
import { resolveWorkspaceUrl } from '@/architecture/shared/routing/route'

type AccessTab = 'current' | 'inherited'
type RoleTone = 'view' | 'edit' | 'admin' | 'owner'

interface RoleOption {
  value: AccessRoleCode
  title: string
  codeLabel: string
  subtitle: string
  description: string
  permissions: string[]
  tone: RoleTone
  icon: Component
}

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const permissionLabelMap = {
  read: 'read',
  write: 'write',
  update: 'update',
  delete: 'delete',
  admin: 'admin',
  owner: 'owner',
} satisfies Record<string, string>

const treeRef = ref()
const treeData = ref<ServiceTree[]>([])
const appName = ref('')
const pageLoading = ref(false)
const membersLoading = ref(false)
const submitting = ref(false)
const removingKey = ref('')
const loadError = ref('')
const activeTab = ref<AccessTab>('current')
const activeResourcePath = ref('')
const selectedResourcePaths = ref<string[]>([])
const members = ref<TeamMemberAccess[]>([])
const grantRole = ref<AccessRoleCode>('viewer')
const grantPermanent = ref(true)
const grantExpiresAt = ref<Date | null>(null)

const grantUsersField = computed<FieldConfig>(() => ({
  code: 'teamAccessPageUsers',
  name: t('access.users'),
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
  code: 'teamAccessPageMemberUsers',
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

const grantUsersValue = ref<FieldValue>(createStringFieldValue(grantUsersField.value, '', { emptyRaw: '' }))

const treeProps = {
  children: 'children',
  label: 'name'
}

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

const requestedResourcePath = computed(() => {
  const raw = Array.isArray(route.query.resource) ? route.query.resource[0] : route.query.resource
  return normalizeResourcePath(String(raw || ''))
})

const parsedResource = computed(() => parseResourcePath(requestedResourcePath.value))
const invalidResource = computed(() => !parsedResource.value)

const activeResource = computed(() => {
  if (!activeResourcePath.value) return null
  return findNodeByPath(treeData.value, activeResourcePath.value)
})

const selectedUsernames = computed(() => {
  return extractStringFieldRaw(grantUsersValue.value)
    .split(',')
    .map(item => item.trim())
    .filter(Boolean)
})

const canSubmitGrant = computed(() => {
  return selectedResourcePaths.value.length > 0 && selectedUsernames.value.length > 0 && Boolean(grantRole.value)
})

const currentMembers = computed(() => {
  return members.value.filter(member => member.direct !== false && member.source !== 'inherited')
})

const inheritedMembers = computed(() => {
  return members.value.filter(member => member.direct === false || member.source === 'inherited')
})

const visibleMembers = computed(() => {
  return activeTab.value === 'current' ? currentMembers.value : inheritedMembers.value
})

watch(requestedResourcePath, () => {
  void reloadPage()
}, { immediate: true })

async function reloadPage() {
  const parsed = parsedResource.value
  loadError.value = ''

  if (!parsed) {
    appName.value = ''
    treeData.value = []
    activeResourcePath.value = ''
    selectedResourcePaths.value = []
    members.value = []
    return
  }

  pageLoading.value = true
  try {
    const appResourcePath = buildAppResourcePath(parsed.user, parsed.app)
    const resp = await getAppWithServiceTree(appResourcePath)
    appName.value = resp.app?.name || parsed.app
    treeData.value = buildTreeData(resp.app, resp.service_tree || [])

    const initialPath = findNodeByPath(treeData.value, parsed.resourcePath)?.full_code_path || parsed.resourcePath
    activeTab.value = 'current'
    activeResourcePath.value = initialPath
    selectedResourcePaths.value = [initialPath]
    resetGrantForm()

    await nextTick()
    treeRef.value?.setCheckedKeys?.([initialPath])
    treeRef.value?.setCurrentKey?.(initialPath)
    await loadMembers()
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || t('access.loadTreeFailed')
    loadError.value = message
    ElMessage.error(message)
  } finally {
    pageLoading.value = false
  }
}

function buildTreeData(app: App, serviceTree: ServiceTree[]): ServiceTree[] {
  const appResourcePath = buildAppResourcePath(app.user, app.code)
  const root = serviceTree.find(node => node.full_code_path === appResourcePath)
  if (root) {
    return serviceTree
  }

  return [{
    id: app.id,
    name: app.name || app.code,
    code: app.code,
    type: 'package',
    description: '',
    tags: '',
    app_id: app.id,
    ref_id: app.id,
    full_code_path: appResourcePath,
    created_at: app.created_at,
    updated_at: app.updated_at,
    children: serviceTree
  }]
}

async function loadMembers() {
  const path = activeResourcePath.value
  if (!path) {
    members.value = []
    return
  }

  membersLoading.value = true
  try {
    const resp = await listTeamMembers(path)
    members.value = resp.members || []
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || t('access.loadFailed')
    ElMessage.error(message)
  } finally {
    membersLoading.value = false
  }
}

function handleResourceClick(data: ServiceTree) {
  if (!data.full_code_path) return
  activeTab.value = 'current'
  activeResourcePath.value = data.full_code_path
  void nextTick(() => treeRef.value?.setCurrentKey?.(data.full_code_path))
  void loadMembers()
}

function handleResourceCheck() {
  const checkedKeys = treeRef.value?.getCheckedKeys?.() as string[] | undefined
  selectedResourcePaths.value = normalizeResourcePathList(checkedKeys || [])
  if (selectedResourcePaths.value.length > 0 && !selectedResourcePaths.value.includes(activeResourcePath.value)) {
    activeResourcePath.value = selectedResourcePaths.value[0] || ''
    treeRef.value?.setCurrentKey?.(activeResourcePath.value)
    void loadMembers()
  }
}

function handleGrantUsersChange(value: FieldValue) {
  grantUsersValue.value = value
}

function resetGrantForm() {
  grantUsersValue.value = createStringFieldValue(grantUsersField.value, '', { emptyRaw: '' })
  grantRole.value = 'viewer'
  grantPermanent.value = true
  grantExpiresAt.value = null
}

function goBack() {
  const target = requestedResourcePath.value ? resolveWorkspaceUrl(requestedResourcePath.value) : '/workspace'
  void router.push(target)
}

function openActiveResource() {
  if (!activeResourcePath.value) {
    goBack()
    return
  }
  void router.push(resolveWorkspaceUrl(activeResourcePath.value))
}

function getFunctionIcon(data: ServiceTree): Component {
  if (data.template_type === TEMPLATE_TYPE.TABLE) return TableIcon
  if (data.template_type === TEMPLATE_TYPE.FORM) return FormIcon
  if (data.template_type === TEMPLATE_TYPE.CHART) return ChartIcon
  return Document
}

function getNodeIconClass(data: ServiceTree): string {
  if (data.type === 'package') return 'package-icon'
  if (data.type === 'function') {
    if (data.template_type === TEMPLATE_TYPE.TABLE) return 'table-icon'
    if (data.template_type === TEMPLATE_TYPE.FORM) return 'form-icon'
    if (data.template_type === TEMPLATE_TYPE.CHART) return 'chart-icon'
    return 'function-icon'
  }
  if (data.type === 'docs') return 'docs-icon'
  return 'function-icon'
}

async function submitGrant() {
  if (!canSubmitGrant.value) {
    ElMessage.warning(t('access.selectResourceUserRole'))
    return
  }

  submitting.value = true
  try {
    await batchAssignTeamRoles({
      resource_paths: selectedResourcePaths.value,
      usernames: selectedUsernames.value,
      role_codes: [grantRole.value],
      expires_at: grantPermanent.value ? null : (grantExpiresAt.value ? grantExpiresAt.value.toISOString() : null)
    })
    ElMessage.success(t('access.grantResourcesSuccess', {
      users: selectedUsernames.value.length,
      resources: selectedResourcePaths.value.length
    }))
    grantUsersValue.value = createStringFieldValue(grantUsersField.value, '', { emptyRaw: '' })
    await loadMembers()
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
      t('access.removeDirectoryConfirm', { username: member.username, role: roleLabel(member.role_code) }),
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
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || t('access.removeFailed')
    ElMessage.error(message)
  } finally {
    removingKey.value = ''
  }
}

function findNodeByPath(nodes: ServiceTree[], path: string): ServiceTree | null {
  for (const item of nodes) {
    if (item.full_code_path === path) return item
    const child = item.children?.length ? findNodeByPath(item.children, path) : null
    if (child) return child
  }
  return null
}

function normalizeResourcePathList(paths: string[]): string[] {
  return [...new Set(paths.map(item => String(item).trim()).filter(Boolean))]
}

function memberRowKey(member: TeamMemberAccess): string {
  return `${member.username}:${member.resource_path}:${member.role_code}`
}

function roleLabel(role: AccessRoleCode): string {
  const option = roleOptions.value.find(item => item.value === role)
  return option?.title || role
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
</script>

<style scoped lang="scss">
.team-access-page {
  min-height: 100vh;
  padding: 18px 22px 24px;
  background: var(--el-bg-color-page);
  color: var(--el-text-color-primary);
}

.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin: 0 auto 16px;
  max-width: 1440px;
}

.page-title-area {
  min-width: 0;

  h1 {
    margin: 4px 0 0;
    font-size: 24px;
    line-height: 1.25;
    letter-spacing: 0;
  }

  p {
    max-width: min(860px, calc(100vw - 380px));
    margin: 6px 0 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    word-break: break-all;
  }
}

.back-button {
  padding-left: 0;
  margin-bottom: 4px;
}

.page-kicker {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.page-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
}

.page-alert,
.access-page-body {
  max-width: 1440px;
  margin: 0 auto;
}

.access-page-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 0;
}

.grant-flow {
  display: grid;
  grid-template-columns: minmax(280px, 0.9fr) minmax(360px, 1.1fr) minmax(300px, 0.95fr);
  gap: 14px;
  min-height: 0;
  align-items: start;
}

.resource-panel,
.role-panel,
.grant-panel,
.members-panel {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.resource-panel,
.role-panel,
.grant-panel {
  max-height: min(690px, calc(100vh - 132px));
  min-height: 0;
  padding: 14px;
}

.resource-panel {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.role-panel,
.grant-panel {
  overflow-y: auto;
}

.panel-header,
.members-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;

  h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  p {
    margin: 4px 0 0;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}

.tree-container {
  flex: 1 1 auto;
  min-height: 0;
  height: min(590px, calc(100vh - 216px));
  overflow-y: auto;
  padding-bottom: 8px;
}

.resource-tree {
  :deep(.el-tree-node__content) {
    height: 32px;
    padding: 0 8px;
    display: flex;
    align-items: center;

    &:hover {
      background-color: var(--el-fill-color-light);
    }
  }

  :deep(.el-tree-node__content:hover) {
    background-color: var(--el-fill-color-light);
  }

  :deep(.el-tree-node__expand-icon) {
    padding: 6px;
    color: var(--el-text-color-secondary);
    border-radius: 2px;
    cursor: pointer;
  }

  :deep(.el-tree-node__expand-icon:hover) {
    background-color: var(--el-fill-color);
  }

  :deep(.el-tree-node.is-expanded > .el-tree-node__content .el-tree-node__expand-icon) {
    transform: rotate(90deg);
  }

  :deep(.el-tree-node__expand-icon.is-leaf) {
    color: transparent;
  }

  :deep(.el-tree-node.is-current > .el-tree-node__content) {
    background-color: rgba(var(--el-color-primary-rgb), 0.12) !important;
    border-left: 2px solid var(--el-color-primary);
  }

  :deep(.el-tree-node.is-current .el-tree-node__children .el-tree-node__content) {
    background-color: transparent;
    border-left: none;
  }
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  width: 100%;
  min-width: 0;

  .node-icon {
    width: 16px;
    height: 16px;
    margin-right: 8px;
    color: #6366f1;
    opacity: 0.86;
    flex-shrink: 0;
  }

  .package-icon-img,
  .form-icon-img,
  .docs-icon-img {
    object-fit: contain;
  }

  .table-icon {
    color: #10b981;
  }

  .form-icon {
    color: #3b82f6;
  }

  .chart-icon {
    color: #f59e0b;
  }

  .docs-icon {
    color: #9b42f8;
  }

  .fx-icon {
    font-size: 12px;
    font-weight: 600;
    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
    font-style: italic;
  }

  .node-label {
    font-size: 14px;
    color: var(--el-text-color-primary);
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.resource-summary {
  padding: 10px 12px;
  margin-bottom: 12px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.resource-title {
  font-size: 14px;
  font-weight: 700;
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
    font-style: normal;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
}

.role-state {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 999px;
  border: 1px solid var(--el-border-color);
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
  font-size: 12px;
  line-height: 1.55;
  color: var(--el-text-color-secondary);
}

.role-action-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.role-action-item {
  padding: 3px 7px;
  border-radius: 999px;
  border: 1px solid transparent;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 12px;
}

.grant-form {
  :deep(.el-form-item) {
    margin-bottom: 14px;
  }
}

.expires-picker {
  width: 100%;
  margin-top: 10px;
}

.grant-preview {
  display: grid;
  gap: 8px;
  padding: 12px;
  margin: 2px 0 14px;
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.preview-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 13px;

  span {
    color: var(--el-text-color-secondary);
  }
}

.submit-button {
  width: 100%;
}

.members-panel {
  padding: 14px;
}

.members-header p {
  max-width: min(900px, calc(100vw - 260px));
  word-break: break-all;
}

.access-tabs {
  :deep(.el-tabs__header) {
    margin: 0 0 4px;
  }
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
  cursor: not-allowed;
}

@media (max-width: 1180px) {
  .page-header {
    align-items: stretch;
    flex-direction: column;
  }

  .page-title-area p {
    max-width: 100%;
  }

  .grant-flow {
    grid-template-columns: 1fr;
  }

  .resource-panel,
  .role-panel,
  .grant-panel {
    max-height: none;
  }

  .tree-container {
    height: min(460px, 52vh);
  }
}

@media (max-width: 640px) {
  .team-access-page {
    padding: 14px 12px 18px;
  }

  .page-actions {
    width: 100%;

    .el-button {
      flex: 1;
    }
  }
}
</style>
