<template>
  <el-dialog
    v-model="dialogVisible"
    class="team-access-dialog"
    width="860px"
    destroy-on-close
  >
    <template #header>
      <div class="dialog-title">
        <el-icon><Key /></el-icon>
        <span>权限管理</span>
      </div>
    </template>

    <div v-if="node" class="access-dialog-body">
      <div class="node-summary">
        <div class="node-name">{{ node.name }}</div>
        <div class="node-path">{{ node.full_code_path }}</div>
      </div>

      <el-form class="grant-form" label-position="top" @submit.prevent>
        <el-form-item label="成员账号">
          <el-input
            v-model="grantUsername"
            placeholder="输入用户名"
            clearable
            @keyup.enter="submitGrant"
          />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="grantRole" class="role-select">
            <el-option
              v-for="role in roleOptions"
              :key="role.value"
              :label="role.label"
              :value="role.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="到期时间">
          <el-date-picker
            v-model="grantExpiresAt"
            class="expires-picker"
            type="datetime"
            placeholder="长期有效"
            clearable
          />
        </el-form-item>
        <el-button
          class="grant-button"
          type="primary"
          :icon="Plus"
          :loading="submitting"
          @click="submitGrant"
        >
          赋权
        </el-button>
      </el-form>

      <el-tabs v-model="activeTab" class="access-tabs">
        <el-tab-pane :label="`当前目录 ${currentMembers.length}`" name="current" />
        <el-tab-pane :label="`继承上级 ${inheritedMembers.length}`" name="inherited" />
      </el-tabs>

      <el-table
        v-loading="loading"
        :data="visibleMembers"
        class="access-table"
        :row-key="memberRowKey"
        size="small"
        empty-text="暂无权限记录"
      >
        <el-table-column prop="username" label="成员" min-width="140" />
        <el-table-column label="角色" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="roleTagType(row.role_code)">
              {{ roleLabel(row.role_code) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="权限" min-width="220">
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
        <el-table-column label="来源" min-width="190">
          <template #default="{ row }">
            <span v-if="row.direct" class="source-current">当前目录</span>
            <span v-else class="source-inherited">{{ row.inherited_from || row.resource_path }}</span>
          </template>
        </el-table-column>
        <el-table-column label="到期" width="150">
          <template #default="{ row }">
            {{ formatExpiresAt(row.expires_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="92" fixed="right">
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
              移除
            </el-button>
            <el-tooltip v-else content="继承权限需要到来源目录移除" placement="top">
              <span class="disabled-action">继承</span>
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Delete, Key, Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { AccessPermissions, AccessRoleCode, ServiceTree } from '@/architecture/domain/types'
import {
  assignTeamRole,
  listTeamMembers,
  removeTeamRole,
  type TeamMemberAccess
} from '@/architecture/presentation/context/api/team-access'

type AccessTab = 'current' | 'inherited'

const props = defineProps<{
  modelValue: boolean
  node: ServiceTree | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'changed'): void
}>()

const roleOptions: Array<{ value: AccessRoleCode; label: string }> = [
  { value: 'viewer', label: 'Viewer 只读' },
  { value: 'member', label: 'Member 编辑' },
  { value: 'admin', label: 'Admin 管理' },
  { value: 'owner', label: 'Owner 拥有者' },
]

const permissionLabelMap = {
  read: 'read',
  write: 'write',
  update: 'update',
  delete: 'delete',
  admin: 'admin',
  owner: 'owner',
} satisfies Record<string, string>

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value)
})

const members = ref<TeamMemberAccess[]>([])
const loading = ref(false)
const submitting = ref(false)
const removingKey = ref('')
const activeTab = ref<AccessTab>('current')
const grantUsername = ref('')
const grantRole = ref<AccessRoleCode>('viewer')
const grantExpiresAt = ref<Date | null>(null)

const currentMembers = computed(() => {
  return members.value.filter(member => member.direct !== false && member.source !== 'inherited')
})

const inheritedMembers = computed(() => {
  return members.value.filter(member => member.direct === false || member.source === 'inherited')
})

const visibleMembers = computed(() => {
  return activeTab.value === 'current' ? currentMembers.value : inheritedMembers.value
})

watch(() => ({
  visible: props.modelValue,
  path: props.node?.full_code_path || ''
}), ({ visible, path }) => {
  if (!visible || !path) return
  activeTab.value = 'current'
  void loadMembers()
}, { immediate: true })

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
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || '获取权限列表失败'
    ElMessage.error(message)
  } finally {
    loading.value = false
  }
}

async function submitGrant() {
  const path = props.node?.full_code_path
  const username = grantUsername.value.trim()
  if (!path) {
    ElMessage.warning('无法获取当前目录路径')
    return
  }
  if (!username) {
    ElMessage.warning('请输入成员账号')
    return
  }

  submitting.value = true
  try {
    await assignTeamRole({
      resource_path: path,
      username,
      role_code: grantRole.value,
      expires_at: grantExpiresAt.value ? grantExpiresAt.value.toISOString() : null
    })
    ElMessage.success('赋权成功')
    grantUsername.value = ''
    grantExpiresAt.value = null
    await loadMembers()
    emit('changed')
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || '赋权失败'
    ElMessage.error(message)
  } finally {
    submitting.value = false
  }
}

async function removeMember(member: TeamMemberAccess) {
  const key = memberRowKey(member)
  try {
    await ElMessageBox.confirm(
      `确认移除 ${member.username} 在当前目录的 ${roleLabel(member.role_code)} 权限？`,
      '移除授权',
      {
        confirmButtonText: '移除',
        cancelButtonText: '取消',
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
    ElMessage.success('已移除授权')
    await loadMembers()
    emit('changed')
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || '移除授权失败'
    ElMessage.error(message)
  } finally {
    removingKey.value = ''
  }
}

function memberRowKey(member: TeamMemberAccess): string {
  return `${member.username}:${member.resource_path}:${member.role_code}`
}

function roleLabel(role: AccessRoleCode): string {
  const option = roleOptions.find(item => item.value === role)
  return option?.label.split(' ')[0] || role
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
  return labels.length > 0 ? labels : ['无']
}

function formatExpiresAt(value?: string): string {
  if (!value) return '长期有效'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}
</script>

<style scoped lang="scss">
.dialog-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}

.access-dialog-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.node-summary {
  padding: 12px 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.node-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.node-path {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}

.grant-form {
  display: grid;
  grid-template-columns: minmax(180px, 1.2fr) minmax(150px, 0.8fr) minmax(190px, 1fr) auto;
  align-items: end;
  gap: 10px;

  :deep(.el-form-item) {
    margin-bottom: 0;
  }
}

.role-select,
.expires-picker {
  width: 100%;
}

.grant-button {
  min-width: 92px;
}

.access-tabs {
  :deep(.el-tabs__header) {
    margin: 0 0 4px;
  }
}

.access-table {
  width: 100%;
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
  display: inline-flex;
  align-items: center;
  height: 24px;
  color: var(--el-text-color-placeholder);
  cursor: not-allowed;
}

@media (max-width: 760px) {
  .grant-form {
    grid-template-columns: 1fr;
  }

  .grant-button {
    width: 100%;
  }
}
</style>
