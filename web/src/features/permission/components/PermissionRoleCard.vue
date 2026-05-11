<template>
  <div
    class="permission-role-card"
    :class="[`tone-${roleVisual.tone}`, { 'is-selected': selected }]"
    role="button"
    tabindex="0"
    @click="emit('select')"
    @keydown.enter.prevent="emit('select')"
    @keydown.space.prevent="emit('select')"
  >
    <div class="role-card-head">
      <div class="role-card-identity">
        <span class="role-icon-badge" :class="`role-icon-badge-${roleVisual.tone}`">
          <el-icon>
            <component :is="roleVisual.icon" />
          </el-icon>
        </span>

        <div class="role-card-copy">
          <div class="role-card-title-row">
            <strong>{{ role.name }}</strong>
            <div class="role-tags">
              <el-tag v-if="role.is_default" size="small" type="warning" effect="plain">默认</el-tag>
              <el-tag v-if="role.is_system" size="small" type="success" effect="plain">系统</el-tag>
            </div>
          </div>
          <span class="role-card-subtitle">{{ roleVisual.label }}</span>
          <span class="role-card-meta-text">{{ permissionCount }} 个权限点</span>
        </div>
      </div>

      <span class="role-card-state" :class="{ 'is-selected': selected }">
        <el-icon><component :is="selected ? CircleCheck : CircleClose" /></el-icon>
        {{ selected ? '当前选中' : '点击选择' }}
      </span>
    </div>

    <p class="role-card-description">
      {{ role.description || roleFallbackDescription }}
    </p>

    <div class="role-action-grid">
      <div
        v-for="actionState in roleActionStates"
        :key="actionState.value"
        class="role-action-item"
        :class="{
          'is-enabled': actionState.enabled,
          'is-disabled': !actionState.enabled,
        }"
      >
        <el-icon class="role-action-icon">
          <component :is="actionState.enabled ? CircleCheck : CircleClose" />
        </el-icon>
        <div class="role-action-copy">
          <span>{{ actionState.label }}</span>
          <small v-if="actionState.description">{{ actionState.description }}</small>
        </div>
      </div>
    </div>

    <div v-if="secondaryPermissionGroups.length > 0" class="role-extended-groups">
      <span class="role-extended-title">附带影响</span>
      <div class="role-extended-list">
        <div
          v-for="group in secondaryPermissionGroups"
          :key="group.resourceType"
          class="role-extended-item"
        >
          <img
            v-if="isImageResourceIcon(group.resourceType)"
            :src="getResourceTypeIconSrc(group.resourceType)"
            :alt="getResourceTypeLabel(group.resourceType)"
            class="resource-icon"
            :class="getResourceTypeIconClass(group.resourceType)"
          />
          <component
            :is="getResourceTypeIconComponent(group.resourceType)"
            v-else
            class="resource-icon"
            :class="getResourceTypeIconClass(group.resourceType)"
          />
          <span>{{ getResourceTypeLabel(group.resourceType) }}</span>
          <em>{{ group.count }} 项</em>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { CircleCheck, CircleClose, Connection, EditPen, Key, User, View } from '@element-plus/icons-vue'
import TableIcon from '@/shared/components/icons/TableIcon.vue'
import ChartIcon from '@/shared/components/icons/ChartIcon.vue'
import type { Role } from '@/api/role'

type RoleVisualTone = 'view' | 'edit' | 'admin' | 'other'
type RoleResourceType = 'directory' | 'table' | 'form' | 'chart' | 'docs' | 'board' | 'workflow' | 'app'

type RoleActionConfig = {
  value: string
  label: string
  description?: string
}

const props = withDefaults(defineProps<{
  role: Role
  selected?: boolean
  fallbackResourceType?: string
}>(), {
  selected: false,
  fallbackResourceType: 'directory',
})

const emit = defineEmits<{
  (e: 'select'): void
}>()

const resourceTypeOrder: RoleResourceType[] = ['app', 'directory', 'table', 'form', 'chart', 'docs', 'board', 'workflow']

const resourceTypeLabels: Record<RoleResourceType, string> = {
  app: '工作空间',
  directory: '目录',
  table: '表格函数',
  form: '表单函数',
  chart: '图表函数',
  docs: '文档',
  board: '讨论区',
  workflow: '工作流',
}

const permissionConfig: Record<RoleResourceType, RoleActionConfig[]> = {
  directory: [
    { value: 'directory:read', label: '查看目录' },
    { value: 'directory:write', label: '写入目录' },
    { value: 'directory:update', label: '更新目录' },
    { value: 'directory:delete', label: '删除目录' },
    { value: 'directory:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
  table: [
    { value: 'table:read', label: '查看表格' },
    { value: 'table:write', label: '新增记录' },
    { value: 'table:update', label: '更新记录' },
    { value: 'table:delete', label: '删除记录' },
    { value: 'table:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
  form: [
    { value: 'form:read', label: '查看表单' },
    { value: 'form:write', label: '提交表单' },
    { value: 'form:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
  chart: [
    { value: 'chart:read', label: '查看图表' },
    { value: 'chart:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
  docs: [
    { value: 'docs:read', label: '查看文档' },
    { value: 'docs:write', label: '编辑文档' },
    { value: 'docs:delete', label: '删除文档' },
    { value: 'docs:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
  board: [
    { value: 'board:read', label: '查看帖子' },
    { value: 'board:write', label: '发帖' },
    { value: 'board:update', label: '更新帖子' },
    { value: 'board:delete', label: '删除帖子' },
    { value: 'board:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
  workflow: [
    { value: 'workflow:read', label: '查看工作流' },
    { value: 'workflow:write', label: '编辑工作流' },
    { value: 'workflow:update', label: '发布工作流' },
    { value: 'workflow:delete', label: '删除工作流' },
    { value: 'workflow:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
  app: [
    { value: 'app:read', label: '查看工作空间' },
    { value: 'app:create', label: '创建工作空间' },
    { value: 'app:update', label: '更新工作空间' },
    { value: 'app:delete', label: '删除工作空间' },
    { value: 'app:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
}

const groupedPermissions = computed<Record<string, string[]>>(() => {
  const grouped: Record<string, string[]> = {}
  for (const permission of props.role.permissions || []) {
    const actions = grouped[permission.resource_type] ?? (grouped[permission.resource_type] = [])
    actions.push(permission.action)
  }
  return grouped
})

const primaryResourceType = computed<RoleResourceType>(() => {
  const inferred = props.role.resource_type
    || props.role.permissions?.[0]?.resource_type
    || props.fallbackResourceType

  if (resourceTypeOrder.includes(inferred as RoleResourceType)) {
    return inferred as RoleResourceType
  }

  return 'directory'
})

const permissionCount = computed(() => props.role.permissions?.length || 0)

const roleVisual = computed(() => {
  const actions = (props.role.permissions || []).map((permission) => permission.action)

  if (actions.some((action) => getActionVerb(action) === 'admin')) {
    return {
      icon: Key,
      tone: 'admin' as RoleVisualTone,
      label: '管理员角色',
    }
  }

  if (actions.some((action) => ['write', 'update', 'delete', 'create'].includes(getActionVerb(action)))) {
    return {
      icon: EditPen,
      tone: 'edit' as RoleVisualTone,
      label: '可编辑角色',
    }
  }

  if (actions.some((action) => getActionVerb(action) === 'read')) {
    return {
      icon: View,
      tone: 'view' as RoleVisualTone,
      label: '只读角色',
    }
  }

  return {
    icon: User,
    tone: 'other' as RoleVisualTone,
    label: '自定义角色',
  }
})

const roleFallbackDescription = computed(() => {
  return `适用于${getResourceTypeLabel(primaryResourceType.value)}场景的权限模板。`
})

const roleActionStates = computed(() => {
  const resourceType = primaryResourceType.value
  const availableActions = permissionConfig[resourceType] || []
  const enabledActions = new Set(groupedPermissions.value[resourceType] || [])
  const hasAdminCoverage = enabledActions.has(`${resourceType}:admin`)

  return availableActions.map((action) => ({
    ...action,
    enabled: hasAdminCoverage || enabledActions.has(action.value),
  }))
})

const secondaryPermissionGroups = computed(() => {
  return Object.entries(groupedPermissions.value)
    .filter(([resourceType]) => resourceType !== primaryResourceType.value)
    .filter(([resourceType, actions]) => actions.length > 0 && resourceTypeOrder.includes(resourceType as RoleResourceType))
    .sort(([left], [right]) => {
      return resourceTypeOrder.indexOf(left as RoleResourceType) - resourceTypeOrder.indexOf(right as RoleResourceType)
    })
    .map(([resourceType, actions]) => ({
      resourceType: resourceType as RoleResourceType,
      count: actions.length,
    }))
})

function getActionVerb(actionValue: string): string {
  return actionValue.split(':').at(-1) || actionValue
}

function getResourceTypeLabel(resourceType: RoleResourceType): string {
  return resourceTypeLabels[resourceType]
}

function isImageResourceIcon(resourceType: RoleResourceType): boolean {
  return resourceType === 'directory'
    || resourceType === 'form'
    || resourceType === 'docs'
    || resourceType === 'board'
    || resourceType === 'app'
}

function getResourceTypeIconSrc(resourceType: RoleResourceType): string {
  switch (resourceType) {
    case 'directory':
      return '/service-tree/custom-folder.svg'
    case 'form':
      return '/service-tree/编辑.svg'
    case 'docs':
      return '/文档.svg'
    case 'board':
      return '/讨论区.svg'
    case 'app':
      return '/service-tree/custom-folder.svg'
    default:
      return '/service-tree/custom-folder.svg'
  }
}

function getResourceTypeIconComponent(resourceType: RoleResourceType) {
  switch (resourceType) {
    case 'chart':
      return ChartIcon
    case 'workflow':
      return Connection
    case 'table':
    default:
      return TableIcon
  }
}

function getResourceTypeIconClass(resourceType: RoleResourceType): string {
  switch (resourceType) {
    case 'directory':
      return 'package-icon-img'
    case 'form':
      return 'form-icon-img'
    case 'docs':
      return 'docs-icon-img'
    case 'board':
      return 'board-icon-img'
    case 'workflow':
      return 'workflow-icon'
    case 'app':
      return 'app-icon-img'
    case 'table':
      return 'table-icon'
    case 'chart':
      return 'chart-icon'
    default:
      return ''
  }
}
</script>

<style scoped lang="scss">
.permission-role-card {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 18px;
  border-radius: 16px;
  border: 1px solid var(--el-border-color-light);
  background: color-mix(in srgb, var(--el-bg-color) 95%, var(--el-color-primary-light-9) 5%);
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease, background 0.2s ease;
  outline: none;
}

.permission-role-card:hover,
.permission-role-card:focus-visible {
  border-color: color-mix(in srgb, var(--el-color-primary) 35%, var(--el-border-color) 65%);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08);
  transform: translateY(-1px);
}

.permission-role-card.is-selected {
  border-color: color-mix(in srgb, var(--el-color-primary) 58%, var(--el-border-color) 42%);
  background: color-mix(in srgb, var(--el-bg-color) 88%, var(--el-color-primary-light-8) 12%);
  box-shadow: 0 12px 28px rgba(64, 158, 255, 0.14);
}

.role-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.role-card-identity {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
}

.role-icon-badge {
  width: 40px;
  height: 40px;
  border-radius: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  flex-shrink: 0;
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

.role-icon-badge-other {
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color);
}

.role-card-copy {
  min-width: 0;
}

.role-card-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.role-card-title-row strong {
  font-size: 17px;
  line-height: 1.25;
  color: var(--el-text-color-primary);
}

.role-card-subtitle {
  display: inline-block;
  margin-top: 6px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.role-card-meta-text {
  display: inline-block;
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.role-tags {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.role-card-state {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
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

.role-card-description {
  margin: 0;
  color: var(--el-text-color-regular);
  line-height: 1.7;
  font-size: 13px;
}

.resource-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.resource-icon.workflow-icon {
  color: #0f766e;
}

.role-action-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 8px;
}

.role-action-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 9px 10px;
  border-radius: 12px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
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
}

.role-action-copy span {
  font-size: 12px;
  font-weight: 600;
}

.role-action-copy small {
  font-size: 11px;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
}

.role-extended-groups {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.role-extended-title {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.role-extended-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.role-extended-item {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 12px;
  background: var(--el-fill-color-extra-light);
  border: 1px solid var(--el-border-color-lighter);
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.role-extended-item em {
  font-style: normal;
  color: var(--el-text-color-secondary);
}

@media (max-width: 768px) {
  .permission-role-card {
    padding: 16px;
  }

  .role-card-head {
    flex-direction: column;
  }

  .role-card-topline {
    flex-direction: column;
    align-items: flex-start;
  }

  .role-card-state {
    align-self: flex-start;
  }

  .role-action-grid {
    grid-template-columns: 1fr;
  }
}
</style>
