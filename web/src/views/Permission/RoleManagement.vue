<!--
  角色管理页面
  
  功能：
  - 查看所有角色列表
  - 创建/编辑角色
  - 配置角色权限（按资源类型分组）
  - 删除角色（系统角色不可删除）
  - 给用户/组织架构分配角色
-->

<template>
  <div class="role-management">
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <h2>角色管理</h2>
          <div class="header-actions">
            <el-select
              v-model="selectedResourceType"
              placeholder="筛选资源类型"
              clearable
              style="width: 150px; margin-right: 10px;"
              @change="handleResourceTypeChange"
            >
              <el-option label="全部" value="" />
              <el-option
                v-for="type in resourceTypes"
                :key="type"
                :label="getResourceTypeLabel(type)"
                :value="type"
              />
            </el-select>
            <el-button type="primary" :icon="Plus" @click="handleCreateRole">新建角色</el-button>
            <el-button :icon="Refresh" @click="loadRoles">刷新</el-button>
          </div>
        </div>
      </template>

      <!-- 按资源类型分组的角色列表 -->
      <div v-loading="loading" class="role-groups-container">
        <!-- 遍历每个资源类型 -->
        <div
          v-for="resourceType in resourceTypes"
          :key="resourceType"
          class="resource-group"
        >
          <el-card shadow="hover" class="resource-group-card">
            <template #header>
              <div class="group-header">
                <div class="group-title">
                  <el-icon class="group-icon"><FolderOpened /></el-icon>
                  <span class="group-name">{{ getResourceTypeLabel(resourceType) }}</span>
                  <el-tag type="info" size="small" class="group-count">
                    {{ getRolesByResourceType(resourceType).length }} 个角色
                  </el-tag>
                </div>
                <el-button
                  type="primary"
                  size="small"
                  :icon="Plus"
                  @click="handleCreateRoleForResourceType(resourceType)"
                >
                  新建{{ getResourceTypeLabel(resourceType) }}角色
                </el-button>
              </div>
            </template>

            <!-- 该资源类型的角色列表 -->
            <el-table
              v-if="getRolesByResourceType(resourceType).length > 0"
              :data="getRolesByResourceType(resourceType)"
              style="width: 100%"
              stripe
              :show-header="true"
            >
              <el-table-column prop="name" label="角色名称" width="200">
                <template #default="{ row }">
                  <div class="role-name-cell">
                    <span class="role-name">{{ row.name }}</span>
                    <div class="role-tags">
                      <el-tag
                        v-if="row.is_default === true"
                        type="warning"
                        size="small"
                        style="margin-left: 8px;"
                      >
                        默认
                      </el-tag>
                      <el-tag
                        v-if="row.is_system === true"
                        type="success"
                        size="small"
                        style="margin-left: 8px;"
                      >
                        系统
                      </el-tag>
                    </div>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="code" label="角色代码" width="150" />
              <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
              <el-table-column label="权限配置" min-width="300">
                <template #default="{ row }">
                  <div class="permissions-display">
                    <el-tag
                      v-for="(actions, rt) in getRolePermissions(row)"
                      :key="rt"
                      size="small"
                      :type="rt === resourceType ? 'primary' : 'info'"
                      style="margin-right: 8px; margin-bottom: 4px;"
                    >
                      {{ getResourceTypeLabel(rt) }}: {{ actions.length }} 个权限
                    </el-tag>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="created_at" label="创建时间" width="180">
                <template #default="{ row }">
                  {{ formatDateTime(row.created_at) }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="200" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="handleEditRole(row)">
                    编辑
                  </el-button>
                  <el-button link type="primary" size="small" @click="handleAssignRole(row)">
                    分配
                  </el-button>
                  <el-button
                    v-if="!row.is_system"
                    link
                    type="danger"
                    size="small"
                    @click="handleDeleteRole(row)"
                  >
                    删除
                  </el-button>
                </template>
              </el-table-column>
            </el-table>

            <!-- 该资源类型无角色时的空状态 -->
            <el-empty
              v-else
              :description="`暂无${getResourceTypeLabel(resourceType)}角色`"
              :image-size="80"
            />
          </el-card>
        </div>

        <!-- 全局空状态（所有资源类型都没有角色） -->
        <el-empty
          v-if="!loading && roleList.length === 0"
          description="暂无角色（请检查后端是否已初始化预设角色）"
          :image-size="100"
        />
      </div>
    </el-card>

    <!-- 创建/编辑角色对话框 -->
    <el-dialog
      v-model="roleDialogVisible"
      :title="roleDialogTitle"
      width="800px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="roleFormRef"
        :model="roleForm"
        :rules="roleFormRules"
        label-width="100px"
      >
        <el-form-item label="角色名称" prop="name">
          <el-input v-model="roleForm.name" placeholder="请输入角色名称" />
        </el-form-item>
        <el-form-item label="角色代码" prop="code">
          <el-input
            v-model="roleForm.code"
            placeholder="请输入角色代码（英文，如：developer）"
            :disabled="!!roleForm.id"
          />
        </el-form-item>
        <el-form-item label="角色描述">
          <el-input
            v-model="roleForm.description"
            type="textarea"
            :rows="3"
            placeholder="请输入角色描述"
          />
        </el-form-item>
        <el-form-item label="设为默认角色" v-if="roleForm.id">
          <el-switch
            v-model="roleForm.is_default"
            active-text="是"
            inactive-text="否"
          />
          <div style="margin-top: 8px; color: #909399; font-size: 12px;">
            💡 设置为默认角色后，在权限申请时会自动推荐此角色
          </div>
        </el-form-item>
        <el-form-item label="权限配置" prop="permissions">
          <div class="permissions-config">
            <div
              v-for="resourceType in getAvailableResourceTypesForRole"
              :key="resourceType"
              class="resource-type-section"
            >
              <div class="resource-type-header">
                <el-checkbox
                  :model-value="isAllSelected(resourceType)"
                  :indeterminate="isIndeterminate(resourceType)"
                  @change="handleSelectAll(resourceType, $event)"
                >
                  <strong>{{ getResourceTypeLabel(resourceType) }}</strong>
                </el-checkbox>
              </div>
              <div class="permissions-list">
                <el-checkbox-group
                  v-model="roleForm.permissions[resourceType]"
                  @change="handlePermissionChange(resourceType)"
                  class="permissions-checkbox-group"
                >
                  <el-checkbox
                    v-for="action in getAvailableActions(resourceType)"
                    :key="action.value"
                    :label="action.value"
                    :value="action.value"
                  >
                    {{ action.label }}
                  </el-checkbox>
                </el-checkbox-group>
              </div>
            </div>
          </div>
          <div class="permissions-tip">
            <el-alert
              v-if="isDirectoryRole"
              type="info"
              :closable="false"
              show-icon
            >
              <template #title>
                <span>提示：目录开发者角色可以配置所有资源类型的权限（目录 + 函数），这样目录下的函数也能继承相应的权限。</span>
              </template>
            </el-alert>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="roleDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="roleSubmitting" @click="handleSubmitRole">
          确定
        </el-button>
      </template>
    </el-dialog>

    <!-- 角色分配对话框 -->
    <el-dialog
      v-model="assignDialogVisible"
      title="分配角色"
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="assignFormRef"
        :model="assignForm"
        :rules="assignFormRules"
        label-width="120px"
      >
        <el-form-item label="分配类型" prop="subject_type">
          <el-radio-group v-model="assignForm.subject_type">
            <el-radio label="user">用户</el-radio>
            <el-radio label="department">组织架构</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item
          v-if="assignForm.subject_type === 'user'"
          label="用户名"
          prop="username"
        >
          <UserSearchInput
            v-model="assignForm.username"
            placeholder="请输入用户名"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item
          v-if="assignForm.subject_type === 'department'"
          label="组织架构"
          prop="department_path"
        >
          <DepartmentSelector
            v-model="assignForm.department_path"
            placeholder="请选择组织架构"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="工作空间" prop="user">
          <el-input v-model="assignForm.user" placeholder="工作空间所属用户" />
        </el-form-item>
        <el-form-item label="应用代码" prop="app">
          <el-input v-model="assignForm.app" placeholder="工作空间应用代码" />
        </el-form-item>
        <el-form-item label="资源路径" prop="resource_path">
          <el-input
            v-model="assignForm.resource_path"
            placeholder="资源路径（支持通配符，如：/user/app/*）"
          />
        </el-form-item>
        <el-form-item label="有效期">
          <el-checkbox v-model="assignForm.isPermanent">永久有效</el-checkbox>
        </el-form-item>
        <el-form-item
          v-if="!assignForm.isPermanent"
          label="开始时间"
          prop="start_time"
        >
          <el-date-picker
            v-model="assignForm.start_time"
            type="datetime"
            placeholder="选择开始时间"
            style="width: 100%"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
          />
        </el-form-item>
        <el-form-item
          v-if="!assignForm.isPermanent"
          label="结束时间"
          prop="end_time"
        >
          <el-date-picker
            v-model="assignForm.end_time"
            type="datetime"
            placeholder="选择结束时间"
            style="width: 100%"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="assignDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="assignSubmitting" @click="handleSubmitAssign">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus, Refresh, FolderOpened } from '@element-plus/icons-vue'
import {
  getRoles,
  getRole,
  createRole,
  updateRole,
  deleteRole,
  assignRoleToUser,
  assignRoleToDepartment,
  type Role,
  type CreateRoleReq,
  type UpdateRoleReq,
  type AssignRoleToUserReq,
  type AssignRoleToDepartmentReq,
} from '@/api/role'
import UserSearchInput from '@/components/UserSearchInput.vue'
import DepartmentSelector from '@/components/DepartmentSelector.vue'

// ==================== 数据定义 ====================

// 加载状态
const loading = ref(false)
const roleSubmitting = ref(false)
const assignSubmitting = ref(false)

// 资源类型过滤
const selectedResourceType = ref<string>('')

// 角色列表
const roleList = ref<Role[]>([])

// 角色对话框
const roleDialogVisible = ref(false)
const roleDialogTitle = computed(() => (roleForm.id ? '编辑角色' : '新建角色'))
const roleFormRef = ref<FormInstance>()
const roleForm = reactive<{
  id?: number
  name: string
  code: string
  description: string
  is_default: boolean
  permissions: Record<string, string[]>
}>({
  name: '',
  code: '',
  description: '',
  is_default: false,
  permissions: {},
})

// 角色表单验证规则
const roleFormRules: FormRules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  code: [
    { required: true, message: '请输入角色代码', trigger: 'blur' },
    { pattern: /^[a-z][a-z0-9_]*$/, message: '角色代码只能包含小写字母、数字和下划线，且必须以字母开头', trigger: 'blur' },
  ],
  permissions: [
    {
      validator: (rule, value, callback) => {
        const hasPermissions = Object.values(value).some(actions => actions.length > 0)
        if (!hasPermissions) {
          callback(new Error('请至少配置一个权限'))
        } else {
          callback()
        }
      },
      trigger: 'change',
    },
  ],
}

// 分配对话框
const assignDialogVisible = ref(false)
const assignFormRef = ref<FormInstance>()
const currentAssignRole = ref<Role | null>(null)
const assignForm = reactive<{
  subject_type: 'user' | 'department'
  username: string
  department_path: string
  user: string
  app: string
  resource_path: string
  isPermanent: boolean
  start_time?: string
  end_time?: string
}>({
  subject_type: 'user',
  username: '',
  department_path: '',
  user: '',
  app: '',
  resource_path: '',
  isPermanent: true,
})

// 分配表单验证规则
const assignFormRules: FormRules = {
  username: [
    {
      required: true,
      message: '请输入用户名',
      trigger: 'blur',
      validator: (rule, value, callback) => {
        if (assignForm.subject_type === 'user' && !value) {
          callback(new Error('请输入用户名'))
        } else {
          callback()
        }
      },
    },
  ],
  department_path: [
    {
      required: true,
      message: '请选择组织架构',
      trigger: 'change',
      validator: (rule, value, callback) => {
        if (assignForm.subject_type === 'department' && !value) {
          callback(new Error('请选择组织架构'))
        } else {
          callback()
        }
      },
    },
  ],
  user: [{ required: true, message: '请输入工作空间所属用户', trigger: 'blur' }],
  app: [{ required: true, message: '请输入工作空间应用代码', trigger: 'blur' }],
  resource_path: [{ required: true, message: '请输入资源路径', trigger: 'blur' }],
}

// ==================== 资源类型和权限配置 ====================

// 资源类型列表
const resourceTypes = ['directory', 'table', 'form', 'chart', 'app']

// 资源类型标签映射
const resourceTypeLabels: Record<string, string> = {
  directory: '目录',
  table: '表格函数',
  form: '表单函数',
  chart: '图表函数',
  app: '工作空间',
}

// 权限点配置（按资源类型）
// ⭐ 使用新的权限点格式：resource_type:action_type（如 table:read, form:write）
const permissionConfig: Record<string, Array<{ value: string; label: string }>> = {
  directory: [
    { value: 'directory:read', label: '查看目录' },
    { value: 'directory:write', label: '写入目录' },
    { value: 'directory:update', label: '更新目录' },
    { value: 'directory:delete', label: '删除目录' },
    { value: 'directory:admin', label: '所有权' },
  ],
  table: [
    { value: 'table:read', label: '查看表格' },
    { value: 'table:write', label: '新增记录' },
    { value: 'table:update', label: '更新记录' },
    { value: 'table:delete', label: '删除记录' },
    { value: 'table:admin', label: '所有权' },
  ],
  form: [
    { value: 'form:read', label: '查看表单' },
    { value: 'form:write', label: '提交表单' },
    { value: 'form:admin', label: '所有权' },
  ],
  chart: [
    { value: 'chart:read', label: '查看图表' },
    { value: 'chart:admin', label: '所有权' },
  ],
  app: [
    { value: 'app:read', label: '查看工作空间' },
    { value: 'app:create', label: '创建工作空间' },
    { value: 'app:update', label: '更新工作空间' },
    { value: 'app:delete', label: '删除工作空间' },
    { value: 'app:admin', label: '所有权' },
  ],
}

// ==================== 计算属性和方法 ====================

/**
 * 获取资源类型标签
 */
function getResourceTypeLabel(resourceType: string): string {
  return resourceTypeLabels[resourceType] || resourceType
}

/**
 * 获取资源类型可用的权限点列表
 */
function getAvailableActions(resourceType: string) {
  return permissionConfig[resourceType] || []
}

/**
 * 判断当前编辑的角色是否是目录类型
 */
const isDirectoryRole = computed(() => {
  if (roleForm.id) {
    const currentRole = roleList.value.find(r => r.id === roleForm.id)
    return currentRole && (currentRole as any).resource_type === 'directory'
  }
  return false
})

/**
 * 获取角色可配置的资源类型列表
 * ⭐ 目录类型的角色可以配置所有资源类型的权限（目录 + 函数）
 * ⭐ 其他类型的角色只能配置对应资源类型的权限
 */
const getAvailableResourceTypesForRole = computed(() => {
  // 如果正在编辑角色，检查角色的 resource_type
  if (roleForm.id) {
    // 从角色列表中查找当前编辑的角色
    const currentRole = roleList.value.find(r => r.id === roleForm.id)
    if (currentRole && (currentRole as any).resource_type === 'directory') {
      // 目录角色可以配置所有资源类型的权限
      return resourceTypes
    } else if (currentRole && (currentRole as any).resource_type) {
      // 其他类型角色只配置对应资源类型的权限
      return [(currentRole as any).resource_type]
    }
  }
  
  // 如果是新建角色，显示所有资源类型（用户可以选择）
  // 实际创建时会根据选择的权限推断主要资源类型
  return resourceTypes
})

/**
 * 获取角色的权限配置（按资源类型分组）
 */
function getRolePermissions(role: Role): Record<string, string[]> {
  if (!role.permissions || role.permissions.length === 0) {
    return {}
  }

  const result: Record<string, string[]> = {}
  for (const perm of role.permissions) {
    if (!result[perm.resource_type]) {
      result[perm.resource_type] = []
    }
    result[perm.resource_type].push(perm.action)
  }
  return result
}

/**
 * 根据资源类型获取角色列表
 */
/**
 * 根据资源类型获取角色列表
 * ⭐ 只返回 resource_type 字段匹配的角色，不检查权限（函数只显示对应资源类型的角色）
 */
function getRolesByResourceType(resourceType: string): Role[] {
  // ⭐ 只根据角色的 resource_type 字段过滤，不检查权限
  // 这样表格函数只显示 table 资源类型的角色，表单函数只显示 form 资源类型的角色
  // 目录角色（resource_type: "directory"）不会出现在函数分组中
  return roleList.value.filter(role => {
    return (role as any).resource_type === resourceType
  })
}

/**
 * 为指定资源类型创建角色
 */
function handleCreateRoleForResourceType(resourceType: string) {
  // 重置表单
  Object.assign(roleForm, {
    id: undefined,
    name: '',
    code: '',
    description: '',
    permissions: {},
  })

  // 初始化权限配置
  for (const rt of resourceTypes) {
    roleForm.permissions[rt] = []
  }

  // 默认选中当前资源类型的第一个权限（如果有）
  const availableActions = getAvailableActions(resourceType)
  if (availableActions.length > 0) {
    roleForm.permissions[resourceType] = [availableActions[0].value]
  }

  roleDialogVisible.value = true
}

/**
 * 检查某个资源类型是否全选
 */
function isAllSelected(resourceType: string): boolean {
  const selected = roleForm.permissions[resourceType] || []
  const available = getAvailableActions(resourceType)
  return available.length > 0 && selected.length === available.length
}

/**
 * 检查某个资源类型是否部分选中（不确定状态）
 */
function isIndeterminate(resourceType: string): boolean {
  const selected = roleForm.permissions[resourceType] || []
  const available = getAvailableActions(resourceType)
  return selected.length > 0 && selected.length < available.length
}

/**
 * 全选/取消全选某个资源类型的权限
 */
function handleSelectAll(resourceType: string, checked: boolean) {
  if (!roleForm.permissions[resourceType]) {
    roleForm.permissions[resourceType] = []
  }

  if (checked) {
    // 全选
    const available = getAvailableActions(resourceType)
    roleForm.permissions[resourceType] = available.map(a => a.value)
  } else {
    // 取消全选
    roleForm.permissions[resourceType] = []
  }
}

/**
 * 权限选择变化时的处理
 */
function handlePermissionChange(resourceType: string) {
  // 触发表单验证
  roleFormRef.value?.validateField('permissions')
}

/**
 * 格式化日期时间
 */
function formatDateTime(dateStr: string): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

// ==================== 数据加载 ====================

/**
 * 加载角色列表
 */
async function loadRoles() {
  try {
    loading.value = true
    // ⭐ 传递资源类型过滤参数
    const resp = await getRoles(selectedResourceType.value || undefined)
    
    // 处理响应数据
    // 注意：响应拦截器已经提取了 data 字段，所以 resp 就是 GetRolesResp
    if (resp && resp.roles && Array.isArray(resp.roles)) {
      roleList.value = resp.roles
      if (roleList.value.length === 0) {
        ElMessage.warning('暂无角色数据，请检查后端是否已初始化预设角色')
      }
    } else {
      console.warn('[RoleManagement] 响应数据格式异常:', resp)
      ElMessage.warning('响应数据格式异常，请检查后端 API')
      roleList.value = []
    }
  } catch (error: any) {
    console.error('[RoleManagement] 加载角色列表失败:', error)
    ElMessage.error(`加载角色列表失败: ${error.message || '未知错误'}`)
    roleList.value = []
  } finally {
    loading.value = false
  }
}

/**
 * 处理资源类型变化
 */
function handleResourceTypeChange() {
  loadRoles()
}

// ==================== 角色 CRUD ====================

/**
 * 创建角色
 */
function handleCreateRole() {
  // 重置表单
  Object.assign(roleForm, {
    id: undefined,
    name: '',
    code: '',
    description: '',
    is_default: false,
    permissions: {},
  })

  // 初始化权限配置
  for (const resourceType of resourceTypes) {
    roleForm.permissions[resourceType] = []
  }

  roleDialogVisible.value = true
}

/**
 * 编辑角色
 */
async function handleEditRole(role: Role) {
  try {
    loading.value = true
    const resp = await getRole(role.id)
    const roleData = resp.role

    // 填充表单
    Object.assign(roleForm, {
      id: roleData.id,
      name: roleData.name,
      code: roleData.code,
      description: roleData.description || '',
      is_default: roleData.is_default || false,
      permissions: {},
    })

    // 初始化权限配置
    for (const resourceType of resourceTypes) {
      roleForm.permissions[resourceType] = []
    }

    // 填充权限配置
    if (roleData.permissions && roleData.permissions.length > 0) {
      for (const perm of roleData.permissions) {
        if (!roleForm.permissions[perm.resource_type]) {
          roleForm.permissions[perm.resource_type] = []
        }
        roleForm.permissions[perm.resource_type].push(perm.action)
      }
    }

    roleDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(`加载角色详情失败: ${error.message || '未知错误'}`)
  } finally {
    loading.value = false
  }
}

/**
 * 提交角色（创建或更新）
 */
async function handleSubmitRole() {
  if (!roleFormRef.value) return

  try {
    await roleFormRef.value.validate()
    roleSubmitting.value = true

    // 构建权限配置（只包含有权限的资源类型）
    const permissions: Record<string, string[]> = {}
    for (const [resourceType, actions] of Object.entries(roleForm.permissions)) {
      if (actions && actions.length > 0) {
        permissions[resourceType] = actions
      }
    }

    if (roleForm.id) {
      // 更新角色
      const req: UpdateRoleReq = {
        name: roleForm.name,
        description: roleForm.description,
        is_default: roleForm.is_default,
        permissions,
      }
      await updateRole(roleForm.id, req)
      ElMessage.success('更新角色成功')
    } else {
      // 创建角色
      const req: CreateRoleReq = {
        name: roleForm.name,
        code: roleForm.code,
        description: roleForm.description,
        permissions,
      }
      await createRole(req)
      ElMessage.success('创建角色成功')
    }

    roleDialogVisible.value = false
    await loadRoles()
  } catch (error: any) {
    if (error.message && !error.message.includes('验证')) {
      ElMessage.error(`操作失败: ${error.message || '未知错误'}`)
    }
  } finally {
    roleSubmitting.value = false
  }
}

/**
 * 删除角色
 */
async function handleDeleteRole(role: Role) {
  try {
    await ElMessageBox.confirm(
      `确定要删除角色 "${role.name}" 吗？此操作不可恢复。`,
      '确认删除',
      {
        type: 'warning',
      }
    )

    await deleteRole(role.id)
    ElMessage.success('删除角色成功')
    await loadRoles()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`删除角色失败: ${error.message || '未知错误'}`)
    }
  }
}

// ==================== 角色分配 ====================

/**
 * 分配角色
 */
function handleAssignRole(role: Role) {
  currentAssignRole.value = role

  // 重置表单
  Object.assign(assignForm, {
    subject_type: 'user',
    username: '',
    department_path: '',
    user: '',
    app: '',
    resource_path: '',
    isPermanent: true,
    start_time: undefined,
    end_time: undefined,
  })

  assignDialogVisible.value = true
}

/**
 * 提交角色分配
 */
async function handleSubmitAssign() {
  if (!assignFormRef.value || !currentAssignRole.value) return

  try {
    await assignFormRef.value.validate()
    assignSubmitting.value = true

    if (assignForm.subject_type === 'user') {
      const req: AssignRoleToUserReq = {
        user: assignForm.user,
        app: assignForm.app,
        username: assignForm.username,
        role_code: currentAssignRole.value.code,
        resource_path: assignForm.resource_path,
        start_time: assignForm.isPermanent ? undefined : assignForm.start_time,
        end_time: assignForm.isPermanent ? undefined : assignForm.end_time,
      }
      await assignRoleToUser(req)
      ElMessage.success('分配角色成功')
    } else {
      const req: AssignRoleToDepartmentReq = {
        user: assignForm.user,
        app: assignForm.app,
        department_path: assignForm.department_path,
        role_code: currentAssignRole.value.code,
        resource_path: assignForm.resource_path,
        start_time: assignForm.isPermanent ? undefined : assignForm.start_time,
        end_time: assignForm.isPermanent ? undefined : assignForm.end_time,
      }
      await assignRoleToDepartment(req)
      ElMessage.success('分配角色成功')
    }

    assignDialogVisible.value = false
  } catch (error: any) {
    if (error.message && !error.message.includes('验证')) {
      ElMessage.error(`分配角色失败: ${error.message || '未知错误'}`)
    }
  } finally {
    assignSubmitting.value = false
  }
}

// ==================== 生命周期 ====================

onMounted(() => {
  loadRoles()
})
</script>

<style scoped lang="scss">
.role-management {
  padding: 20px;
  min-height: calc(100vh - 40px);
  width: 100%;
  box-sizing: border-box;

  :deep(.el-card) {
    height: auto;
    overflow: visible;
  }

  :deep(.el-card__body) {
    overflow: visible;
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;

    h2 {
      margin: 0;
      font-size: 18px;
      font-weight: 600;
    }

    .header-actions {
      display: flex;
      gap: 8px;
    }
  }

  .role-groups-container {
    display: flex;
    flex-direction: column;
    gap: 20px;
    min-height: 0; // 允许 flex 子元素收缩
  }

  .resource-group {
    .resource-group-card {
      border: 1px solid var(--el-border-color-lighter);
      transition: all 0.3s;
      margin-bottom: 0;

      &:hover {
        border-color: var(--el-color-primary);
        box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
      }

      :deep(.el-card__body) {
        overflow: visible;
        padding: 20px;
      }

      .group-header {
        display: flex;
        justify-content: space-between;
        align-items: center;

        .group-title {
          display: flex;
          align-items: center;
          gap: 8px;

          .group-icon {
            font-size: 18px;
            color: var(--el-color-primary);
          }

          .group-name {
            font-size: 16px;
            font-weight: 600;
            color: var(--el-text-color-primary);
          }

          .group-count {
            margin-left: 4px;
          }
        }
      }
    }
  }

  .role-name-cell {
    display: flex;
    align-items: center;
  }

  .permissions-display {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .permissions-config {
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
    padding: 16px;
    max-height: 500px;
    overflow-y: auto;

    .resource-type-section {
      margin-bottom: 24px;

      &:last-child {
        margin-bottom: 0;
      }

      .resource-type-header {
        margin-bottom: 12px;
        padding-bottom: 8px;
        border-bottom: 1px solid var(--el-border-color-lighter);
      }

      .permissions-list {
        padding-left: 24px;

        .permissions-checkbox-group {
          display: flex;
          flex-direction: row;
          flex-wrap: wrap;
          gap: 16px;
          align-items: center;
        }
      }
    }
  }

  .permissions-tip {
    margin-top: 12px;
  }
}
</style>
