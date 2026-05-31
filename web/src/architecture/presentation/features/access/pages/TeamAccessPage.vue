<template>
  <main class="team-access-page">
    <div class="access-shell">
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

      <section v-else v-loading="pageLoading" class="apply-layout">
        <aside class="apply-sidebar">
          <section class="tree-card">
            <div class="panel-toolbar">
              <div class="panel-title-copy">
                <h3>{{ t('access.selectResource') }}</h3>
                <p>{{ t('access.selectResourceDesc') }}</p>
              </div>
              <div class="resource-tools">
                <el-tag size="small" effect="plain">
                  {{ t('common.selectedCount', { count: selectedResourcePaths.length }) }}
                </el-tag>
                <el-tooltip :content="t('access.viewExisting')" placement="bottom">
                  <el-button
                    size="small"
                    :icon="UserFilled"
                    :loading="membersLoading && membersDialogVisible"
                    @click="openMembersDialog"
                  >
                    {{ t('access.existing') }}
                  </el-button>
                </el-tooltip>
              </div>
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
        </aside>

        <section class="apply-main">
          <div class="scope-header-main">
            <div class="scope-title-main">
              <span class="scope-icon">
                <img
                  v-if="currentResourceIconSrc"
                  :src="currentResourceIconSrc"
                  :alt="currentResourceTypeLabel"
                  class="scope-icon-img"
                  :class="currentResourceIconClass"
                />
                <component
                  v-else
                  :is="currentResourceIconComponent"
                  class="scope-icon-component"
                  :class="currentResourceIconClass"
                />
              </span>
              <span class="scope-name-main">{{ activeResource?.name || appName || t('common.currentResource') }}</span>
              <el-tag size="small" :type="currentResourceTagType">{{ currentResourceTypeLabel }}</el-tag>
            </div>
            <div class="scope-path-main">
              <code>{{ activeResourcePath }}</code>
            </div>
          </div>

          <div class="role-selection-section">
            <div class="role-selection-header">
              <div class="role-selection-copy">
                <span class="role-selection-kicker">{{ t('access.roleSwitch') }}</span>
                <h4 class="role-selection-title">
                  <el-icon><UserFilled /></el-icon>
                  {{ t('access.selectRole') }}
                </h4>
                <div class="role-selection-meta">
                  <div class="role-selection-resource-pill">
                    <img
                      v-if="currentResourceIconSrc"
                      :src="currentResourceIconSrc"
                      :alt="currentResourceTypeLabel"
                      class="role-selection-resource-icon"
                      :class="currentResourceIconClass"
                    />
                    <component
                      v-else
                      :is="currentResourceIconComponent"
                      class="role-selection-resource-icon"
                      :class="currentResourceIconClass"
                    />
                    <span>{{ currentResourceTypeLabel }}</span>
                  </div>
                </div>
                <p class="role-selection-desc">{{ t('access.roleCardDesc') }}</p>
              </div>
              <span class="role-selection-hint">
                {{ t('access.selectedRole', { role: roleLabel(grantRole) }) }}
              </span>
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
                <span class="role-card-aside">
                  <span class="role-card-identity">
                    <span class="role-icon-badge" :class="`role-icon-badge-${role.tone}`">
                      <el-icon><component :is="role.icon" /></el-icon>
                    </span>
                    <span class="role-card-copy">
                      <span class="role-card-title-row">
                        <strong>{{ role.title }}</strong>
                      </span>
                      <span class="role-card-subtitle">{{ role.subtitle }}</span>
                      <span class="role-card-meta-text">{{ role.codeLabel }} · {{ rolePermissionPointCount(role) }} {{ t('access.permissionPoints') }}</span>
                    </span>
                  </span>
                  <span class="role-card-state" :class="{ 'is-selected': grantRole === role.value }">
                    <el-icon><component :is="grantRole === role.value ? CircleCheck : CircleClose" /></el-icon>
                    {{ grantRole === role.value ? t('access.roleSelected') : t('access.roleSelect') }}
                  </span>
                </span>
                <span class="role-action-groups">
                  <span
                    v-for="group in roleActionGroups(role)"
                    :key="`${role.value}-${group.kind}`"
                    class="role-action-group"
                  >
                    <span class="role-action-group-title">
                      <img
                        v-if="group.iconSrc"
                        :src="group.iconSrc"
                        :alt="group.label"
                        class="role-action-group-icon"
                        :class="group.iconClass"
                      />
                      <component
                        v-else
                        :is="group.iconComponent"
                        class="role-action-group-icon"
                        :class="group.iconClass"
                      />
                      <span>{{ group.label }}</span>
                    </span>
                    <span class="role-action-grid">
                      <span
                        v-for="actionState in group.actions"
                        :key="`${role.value}-${group.kind}-${actionState.value}`"
                        class="role-action-item"
                        :class="{
                          'is-enabled': actionState.enabled,
                          'is-disabled': !actionState.enabled,
                        }"
                      >
                        <el-icon class="role-action-icon">
                          <component :is="actionState.enabled ? CircleCheck : CircleClose" />
                        </el-icon>
                        <span class="role-action-copy">
                          <span>{{ actionState.label }}</span>
                          <small v-if="actionState.description">{{ actionState.description }}</small>
                        </span>
                      </span>
                    </span>
                  </span>
                </span>
              </button>
            </div>
          </div>
        </section>

        <aside class="apply-sidebar-right">
          <section class="form-card">
            <div class="panel-toolbar">
              <div class="panel-title-copy">
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
              <el-button class="cancel-button" @click="goBack">
                {{ t('common.cancel') }}
              </el-button>
            </el-form>
          </section>
        </aside>
      </section>
    </div>

    <el-dialog
      v-model="membersDialogVisible"
      class="access-members-dialog"
      width="960px"
      :title="t('access.existing')"
      destroy-on-close
    >
      <div class="members-dialog-header">
        <div>
          <h3>{{ activeResource?.name || t('common.currentResource') }}</h3>
          <p>{{ activeResourcePath }}</p>
        </div>
        <el-button size="small" :icon="Refresh" :loading="membersLoading" @click="loadMembers">
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
    </el-dialog>
  </main>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  CircleCheck,
  CircleClose,
  Delete,
  Document,
  EditPen,
  Key,
  Plus,
  Refresh,
  User,
  UserFilled,
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
type ResourceKind = 'app' | 'directory' | 'table' | 'form' | 'chart' | 'docs'

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

interface RoleActionConfig {
  value: AccessPermissionsKey
  label: string
  description?: string
}

interface RoleActionState extends RoleActionConfig {
  enabled: boolean
}

interface RoleActionGroup {
  kind: ResourceKind
  label: string
  iconSrc: string
  iconComponent: Component
  iconClass: string
  actions: RoleActionState[]
}

type AccessPermissionsKey = keyof AccessPermissions

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
const membersDialogVisible = ref(false)
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

const directoryRoleActionKinds: ResourceKind[] = ['directory', 'table', 'form', 'chart', 'docs']
const appRoleActionKinds: ResourceKind[] = ['app', ...directoryRoleActionKinds]

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

const currentResourceKind = computed<ResourceKind>(() => {
  const node = activeResource.value
  if (!node) return 'directory'
  if (node.type === 'docs') return 'docs'
  if (node.type === 'function') {
    if (node.template_type === TEMPLATE_TYPE.FORM) return 'form'
    if (node.template_type === TEMPLATE_TYPE.CHART) return 'chart'
    return 'table'
  }
  const depth = node.full_code_path?.split('/').filter(Boolean).length || 0
  return depth <= 2 ? 'app' : 'directory'
})

const currentResourceTypeLabel = computed(() => getResourceKindLabel(currentResourceKind.value))

const currentResourceTagType = computed((): 'primary' | 'success' | 'warning' | 'info' => {
  if (currentResourceKind.value === 'table') return 'primary'
  if (currentResourceKind.value === 'form' || currentResourceKind.value === 'directory') return 'success'
  if (currentResourceKind.value === 'chart') return 'warning'
  return 'info'
})

const currentResourceIconSrc = computed(() => getResourceIconSrc(currentResourceKind.value))

const currentResourceIconComponent = computed<Component>(() => getResourceIconComponent(currentResourceKind.value))

const currentResourceIconClass = computed(() => getResourceIconClass(currentResourceKind.value))

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

const roleActionConfigs = computed<Record<ResourceKind, RoleActionConfig[]>>(() => ({
  app: [
    { value: 'read', label: t('access.actionAppRead') },
    { value: 'write', label: t('access.actionAppCreate') },
    { value: 'update', label: t('access.actionAppUpdate') },
    { value: 'delete', label: t('access.actionAppDelete') },
    { value: 'admin', label: t('access.actionAdmin'), description: t('access.actionAdminDesc') },
    { value: 'owner', label: t('access.actionOwnership'), description: t('access.actionOwnershipDesc') },
  ],
  directory: [
    { value: 'read', label: t('access.actionDirectoryRead') },
    { value: 'write', label: t('access.actionDirectoryWrite') },
    { value: 'update', label: t('access.actionDirectoryUpdate') },
    { value: 'delete', label: t('access.actionDirectoryDelete') },
    { value: 'admin', label: t('access.actionAdmin'), description: t('access.actionAdminDesc') },
    { value: 'owner', label: t('access.actionOwnership'), description: t('access.actionOwnershipDesc') },
  ],
  table: [
    { value: 'read', label: t('access.actionTableRead') },
    { value: 'write', label: t('access.actionTableWrite') },
    { value: 'update', label: t('access.actionTableUpdate') },
    { value: 'delete', label: t('access.actionTableDelete') },
    { value: 'admin', label: t('access.actionAdmin'), description: t('access.actionAdminDesc') },
    { value: 'owner', label: t('access.actionOwnership'), description: t('access.actionOwnershipDesc') },
  ],
  form: [
    { value: 'read', label: t('access.actionFormRead') },
    { value: 'write', label: t('access.actionFormSubmit') },
    { value: 'admin', label: t('access.actionAdmin'), description: t('access.actionAdminDesc') },
    { value: 'owner', label: t('access.actionOwnership'), description: t('access.actionOwnershipDesc') },
  ],
  chart: [
    { value: 'read', label: t('access.actionChartRead') },
    { value: 'admin', label: t('access.actionAdmin'), description: t('access.actionAdminDesc') },
    { value: 'owner', label: t('access.actionOwnership'), description: t('access.actionOwnershipDesc') },
  ],
  docs: [
    { value: 'read', label: t('access.actionDocsRead') },
    { value: 'write', label: t('access.actionDocsWrite') },
    { value: 'update', label: t('access.actionDocsUpdate') },
    { value: 'delete', label: t('access.actionDocsDelete') },
    { value: 'admin', label: t('access.actionAdmin'), description: t('access.actionAdminDesc') },
    { value: 'owner', label: t('access.actionOwnership'), description: t('access.actionOwnershipDesc') },
  ],
}))

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
    const initialResourcePaths = collectSelectionWithDescendants(initialPath)
    activeTab.value = 'current'
    activeResourcePath.value = initialPath
    selectedResourcePaths.value = initialResourcePaths
    resetGrantForm()

    await nextTick()
    treeRef.value?.setCheckedKeys?.(initialResourcePaths)
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

function handleResourceCheck(data: ServiceTree) {
  if (data?.full_code_path) {
    applyResourceSelectionCascade(data, isResourceChecked(data))
  } else {
    syncCheckedResourcePaths()
  }
}

function syncCheckedResourcePaths() {
  const checkedKeys = treeRef.value?.getCheckedKeys?.() as string[] | undefined
  selectedResourcePaths.value = normalizeResourcePathList(checkedKeys || [])
  if (selectedResourcePaths.value.length > 0 && !selectedResourcePaths.value.includes(activeResourcePath.value)) {
    activeResourcePath.value = selectedResourcePaths.value[0] || ''
    treeRef.value?.setCurrentKey?.(activeResourcePath.value)
    void loadMembers()
  }
}

function isResourceChecked(node: ServiceTree): boolean {
  if (!node.full_code_path) return false
  const checkedKeys = treeRef.value?.getCheckedKeys?.() as string[] | undefined
  return new Set(checkedKeys || []).has(node.full_code_path)
}

function applyResourceSelectionCascade(node: ServiceTree, checked: boolean) {
  const checkedKeys = new Set((treeRef.value?.getCheckedKeys?.() as string[] | undefined) || [])
  for (const path of collectNodeAndDescendantPaths(node)) {
    if (checked) {
      checkedKeys.add(path)
    } else {
      checkedKeys.delete(path)
    }
  }
  treeRef.value?.setCheckedKeys?.([...checkedKeys])
  syncCheckedResourcePaths()
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

async function openMembersDialog() {
  membersDialogVisible.value = true
  await loadMembers()
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

function collectNodeAndDescendantPaths(node: ServiceTree): string[] {
  const paths: string[] = []
  const walk = (current: ServiceTree) => {
    if (current.full_code_path) {
      paths.push(current.full_code_path)
    }
    for (const child of current.children || []) {
      walk(child)
    }
  }
  walk(node)
  return paths
}

function collectSelectionWithDescendants(path: string): string[] {
  const node = findNodeByPath(treeData.value, path)
  return node ? normalizeResourcePathList(collectNodeAndDescendantPaths(node)) : [path]
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

function roleActionGroups(role: RoleOption): RoleActionGroup[] {
  return getRoleActionKinds()
    .map(kind => ({
      kind,
      label: getResourceKindLabel(kind),
      iconSrc: getResourceIconSrc(kind),
      iconComponent: getResourceIconComponent(kind),
      iconClass: getResourceIconClass(kind),
      actions: roleActionStatesForKind(role, kind),
    }))
    .filter(group => group.actions.length > 0)
}

function getRoleActionKinds(): ResourceKind[] {
  if (currentResourceKind.value === 'app') return appRoleActionKinds
  if (currentResourceKind.value === 'directory') return directoryRoleActionKinds
  return [currentResourceKind.value]
}

function roleActionStatesForKind(role: RoleOption, kind: ResourceKind): RoleActionState[] {
  const actionConfigs = (roleActionConfigs.value[kind] || []).filter(action => shouldShowRoleAction(role, action.value))
  const enabledActions = new Set(role.permissions)
  const hasAdminCoverage = enabledActions.has('admin')
  const hasOwnerCoverage = enabledActions.has('owner')
  return actionConfigs.map(action => ({
    ...action,
    enabled: hasOwnerCoverage
      || (action.value !== 'owner' && hasAdminCoverage)
      || enabledActions.has(action.value)
  }))
}

function shouldShowRoleAction(role: RoleOption, action: AccessPermissionsKey): boolean {
  if (action === 'admin') return role.value === 'admin' || role.value === 'owner'
  if (action === 'owner') return role.value === 'owner'
  return true
}

function rolePermissionPointCount(role: RoleOption): number {
  return roleActionGroups(role).reduce((total, group) => {
    return total + group.actions.filter(action => action.enabled).length
  }, 0)
}

function getResourceKindLabel(kind: ResourceKind): string {
  const labels: Record<ResourceKind, string> = {
    app: t('access.resourceApp'),
    directory: t('access.resourceDirectory'),
    table: t('packageDetail.table'),
    form: t('packageDetail.form'),
    chart: t('packageDetail.chart'),
    docs: t('packageDetail.docs'),
  }
  return labels[kind]
}

function getResourceIconSrc(kind: ResourceKind): string {
  if (kind === 'app' || kind === 'directory') return '/service-tree/custom-folder.svg'
  if (kind === 'form') return '/service-tree/编辑.svg'
  if (kind === 'docs') return '/文档.svg'
  return ''
}

function getResourceIconComponent(kind: ResourceKind): Component {
  if (kind === 'chart') return ChartIcon
  return TableIcon
}

function getResourceIconClass(kind: ResourceKind): string {
  if (kind === 'app') return 'app-icon-img'
  if (kind === 'directory') return 'package-icon-img'
  if (kind === 'form') return 'form-icon-img'
  if (kind === 'docs') return 'docs-icon-img'
  if (kind === 'chart') return 'chart-icon'
  return 'table-icon'
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

.team-access-page {
  box-sizing: border-box;
  height: 100vh;
  padding: 12px 10px;
  overflow: hidden;
}

.access-shell {
  width: 100%;
  max-width: none;
  height: 100%;
  margin: 0;
}

.apply-layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr) 300px;
  gap: 12px;
  height: 100%;
  min-height: 0;
  align-items: stretch;
  overflow: hidden;
}

.apply-sidebar,
.apply-sidebar-right,
.apply-main {
  min-height: 0;
  height: 100%;
}

.tree-card,
.form-card {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
  overflow: hidden;
}

.apply-main {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  max-height: 100%;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding-right: 4px;
  scrollbar-gutter: stable;
}

.panel-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  padding: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-lighter);
}

.panel-title-copy {
  min-width: 0;

  h3 {
    margin: 0;
    font-size: 15px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  p {
    margin: 5px 0 0;
    font-size: 12px;
    line-height: 1.5;
    color: var(--el-text-color-secondary);
  }
}

.resource-tools {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.tree-container {
  height: auto;
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  padding: 12px 10px 18px;
}

.resource-tree {
  :deep(.el-tree-node__content) {
    height: 34px;
    padding: 0 8px;
    border-radius: 6px;
  }

  :deep(.el-tree-node.is-current > .el-tree-node__content) {
    border-left: none;
    box-shadow: inset 2px 0 0 var(--el-color-primary);
  }
}

.scope-header-main {
  padding: 0 0 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.scope-title-main {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.scope-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
}

.scope-icon-img,
.scope-icon-component,
.role-selection-resource-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.scope-name-main {
  min-width: 0;
  max-width: 100%;
  color: var(--el-text-color-primary);
  font-size: 18px;
  font-weight: 700;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scope-path-main {
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  word-break: break-all;

  code {
    padding: 2px 6px;
    border-radius: 4px;
    background: var(--el-fill-color-lighter);
    color: var(--el-text-color-secondary);
  }
}

.role-selection-section {
  padding: 14px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  overflow: visible;
}

.role-selection-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.role-selection-copy {
  min-width: 0;
}

.role-selection-kicker {
  display: inline-block;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.role-selection-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.role-selection-meta {
  margin-top: 10px;
}

.role-selection-resource-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 7px 12px;
  border-radius: 999px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  font-size: 12px;
  color: var(--el-text-color-primary);
}

.role-selection-desc {
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 1.55;
  color: var(--el-text-color-secondary);
}

.role-selection-hint {
  flex-shrink: 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
  padding: 8px 12px;
  border-radius: 999px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
}

.role-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding-bottom: 10px;
}

.role-card {
  display: grid;
  grid-template-columns: 150px minmax(0, 1fr);
  align-items: stretch;
  gap: 12px;
  min-height: 0;
  padding: 10px;
  border-radius: 8px;
  background: var(--el-bg-color);
  transition: border-color 0.18s ease, background-color 0.18s ease, box-shadow 0.18s ease, transform 0.18s ease;
}

.role-card:hover,
.role-card:focus-visible {
  border-color: color-mix(in srgb, var(--el-color-primary) 35%, var(--el-border-color) 65%);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08);
  transform: translateY(-1px);
  outline: none;
}

.role-card.is-selected {
  border-color: color-mix(in srgb, #15803d 48%, var(--el-border-color) 52%);
  background: var(--el-bg-color);
  box-shadow: 0 10px 22px rgba(15, 23, 42, 0.08), inset 0 0 0 1px rgba(21, 128, 61, 0.16);
}

.role-card-aside {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
  padding: 2px 10px 2px 0;
  border-right: 1px solid var(--el-border-color-lighter);
}

.role-card-identity {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
}

.role-icon-badge {
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  font-size: 16px;
}

.role-icon-badge-view {
  color: #2563eb;
  background: rgba(37, 99, 235, 0.12);
}

.role-icon-badge-edit {
  color: #0f766e;
  background: rgba(15, 118, 110, 0.12);
}

.role-icon-badge-admin {
  color: #b45309;
  background: rgba(245, 158, 11, 0.14);
}

.role-icon-badge-owner {
  color: #be123c;
  background: rgba(244, 63, 94, 0.12);
}

.role-card-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.role-card-title-row strong {
  font-size: 15px;
  line-height: 1.25;
  color: var(--el-text-color-primary);
}

.role-card-subtitle,
.role-card-meta-text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.role-card-state {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  width: fit-content;
  padding: 4px 8px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.role-card-state.is-selected {
  background: rgba(34, 197, 94, 0.12);
  color: #15803d;
}

.role-description {
  display: none;
}

.role-action-groups {
  align-self: center;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(128px, 1fr));
  gap: 6px;
  min-width: 0;
}

.role-action-group {
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
  padding: 6px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: color-mix(in srgb, var(--el-bg-color) 94%, var(--el-fill-color) 6%);
}

.role-action-group-title {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 11px;
  font-weight: 700;

  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.role-action-group-icon {
  width: 13px;
  height: 13px;
  flex-shrink: 0;
  object-fit: contain;
}

.role-action-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(112px, 1fr));
  gap: 6px;
}

.role-action-group .role-action-grid {
  grid-template-columns: repeat(auto-fit, minmax(48px, max-content));
  gap: 4px;
}

.role-action-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 7px 8px;
  border-radius: 8px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  font-size: 12px;
  font-weight: 600;
  min-width: 0;
}

.role-action-group .role-action-item {
  min-height: 26px;
  gap: 5px;
  padding: 4px 6px;
  border-radius: 7px;
  font-size: 11px;
}

.role-action-group .role-action-icon {
  font-size: 12px;
}

.role-action-item.is-enabled {
  border-color: rgba(34, 197, 94, 0.22);
  background: rgba(34, 197, 94, 0.08);
  color: #15803d;
}

.role-action-item.is-disabled {
  color: var(--el-text-color-secondary);
  background: color-mix(in srgb, var(--el-bg-color) 96%, var(--el-text-color-secondary) 4%);
}

.role-action-icon {
  flex-shrink: 0;
}

.role-action-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;

  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  small {
    color: var(--el-text-color-secondary);
    font-size: 11px;
    font-weight: 400;
    line-height: 1.35;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.role-action-group .role-action-copy small {
  display: none;
}

.role-card.is-selected .role-card-title-row strong {
  color: var(--el-text-color-primary);
}

.role-card.is-selected .role-icon-badge-view {
  color: #2563eb;
  background: rgba(37, 99, 235, 0.12);
}

.role-card.is-selected .role-icon-badge-edit {
  color: #0f766e;
  background: rgba(15, 118, 110, 0.12);
}

.role-card.is-selected .role-icon-badge-admin {
  color: #b45309;
  background: rgba(245, 158, 11, 0.14);
}

.role-card.is-selected .role-icon-badge-owner {
  color: #be123c;
  background: rgba(244, 63, 94, 0.12);
}

.role-card.is-selected .role-action-item.is-enabled {
  border-color: rgba(34, 197, 94, 0.24);
  background: rgba(34, 197, 94, 0.08);
  color: #15803d;
}

.role-card.is-selected .role-action-item.is-disabled {
  border-color: var(--el-border-color-lighter);
  background: color-mix(in srgb, var(--el-bg-color) 96%, var(--el-text-color-secondary) 4%);
  color: var(--el-text-color-secondary);
}

.grant-form {
  flex: 1 1 auto;
  min-height: 0;
  padding: 12px;
  overflow-y: auto;

  :deep(.el-form-item) {
    margin-bottom: 18px;
  }
}

.grant-preview {
  padding: 14px;
  margin: 2px 0 16px;
}

.submit-button,
.cancel-button {
  width: 100%;
}

.cancel-button {
  margin: 12px 0 0;
}

.members-dialog-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 10px;

  h3 {
    margin: 0;
    font-size: 15px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  p {
    margin: 5px 0 0;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    word-break: break-all;
  }
}

@media (max-width: 1500px) {
  .apply-layout {
    grid-template-columns: 260px minmax(0, 1fr) 280px;
    gap: 10px;
  }
}

@media (max-width: 1180px) {
  .team-access-page {
    height: auto;
    min-height: 100vh;
    overflow: auto;
  }

  .access-shell {
    height: auto;
  }

  .apply-layout {
    grid-template-columns: 1fr;
    height: auto;
  }

  .tree-card,
  .form-card {
    height: auto;
    max-height: none;
  }

  .role-selection-header {
    flex-direction: column;
  }

  .role-card {
    grid-template-columns: 1fr;
  }

  .role-card-aside {
    flex-direction: row;
    align-items: center;
    border-right: 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
    padding: 0 0 8px;
  }

  .role-card-identity {
    flex-direction: row;
    align-items: center;
  }
}

@media (max-width: 640px) {
  .resource-tools {
    align-items: flex-end;
    flex-direction: column;
  }

  .role-action-groups,
  .role-action-grid {
    grid-template-columns: 1fr;
  }
}
</style>
