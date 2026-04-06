<template>
  <div class="permission-apply-wrapper">
    <div class="permission-apply">
      <el-card shadow="hover" class="apply-card">
      <template #header>
        <div class="card-header">
          <h2>权限申请</h2>
        </div>
      </template>

      <div v-if="loading" class="loading-container">
        <el-skeleton :rows="5" animated />
      </div>

      <div v-else-if="error" class="error-container">
        <el-alert
          :title="error"
          type="error"
          :closable="false"
          show-icon
        />
      </div>

      <div v-else class="apply-content">
        <div class="apply-layout">
          <!-- 左侧：资源树选择 -->
          <div class="apply-sidebar">
            <el-card shadow="never" class="tree-card">
              <template #header>
                <h3>选择资源</h3>
              </template>
              <div class="tree-container">
                <el-tree
                  ref="treeRef"
                  :data="serviceTree"
                  :props="treeProps"
                  :default-expand-all="true"
                  :expand-on-click-node="true"
                  :highlight-current="true"
                  node-key="full_code_path"
                  :default-expanded-keys="defaultExpandedKeys"
                  :current-node-key="selectedResourcePath"
                  show-checkbox
                  :check-strictly="true"
                  :checked-keys="checkedNodeKeys"
                  @node-click="handleTreeNodeClick"
                  @check="handleTreeNodeCheck"
                  class="resource-tree"
                >
                  <template #default="{ node, data }">
                    <span class="tree-node" :class="{ 'is-selected': selectedResourcePath === data.full_code_path }">
                      <!-- package 类型（包括根节点）：统一使用目录图标 -->
                      <img 
                        v-if="data.type === 'package'" 
                        src="/service-tree/custom-folder.svg" 
                        alt="目录" 
                        class="node-icon package-icon-img"
                        :class="getNodeIconClass(data)"
                      />
                      <!-- function 类型：根据 template_type 显示不同图标 -->
                      <template v-else-if="data.type === 'function'">
                        <!-- 表单类型：使用编辑图标 -->
                        <img 
                          v-if="data.template_type === TEMPLATE_TYPE.FORM"
                          src="/service-tree/编辑.svg" 
                          alt="表单" 
                          class="node-icon form-icon-img"
                          :class="getNodeIconClass(data)"
                        />
                        <!-- 其他类型：使用组件图标 -->
                        <component 
                          v-else 
                          :is="getFunctionIcon(data)"
                          class="node-icon"
                          :class="getNodeIconClass(data)"
                        />
                      </template>
                      <!-- docs 类型：使用文档图标 -->
                      <img 
                        v-else-if="data.type === 'docs'" 
                        src="/文档.svg" 
                        alt="文档" 
                        class="node-icon docs-icon-img"
                        :class="getNodeIconClass(data)"
                      />
                      <!-- board 类型：使用讨论区图标 -->
                      <img 
                        v-else-if="data.type === 'board'" 
                        src="/讨论区.svg" 
                        alt="讨论区" 
                        class="node-icon board-icon-img"
                        :class="getNodeIconClass(data)"
                      />
                      <!-- 其他类型：显示 fx 文本 -->
                      <span v-else class="node-icon fx-icon" :class="getNodeIconClass(data)">fx</span>
                      <span class="node-label" :class="{ 'no-permission': !hasAnyPermissionForNode(data) }">{{ node.label }}</span>
                      
                      <!-- 无权限标识 - 没有权限的节点显示 -->
                      <img 
                        v-if="!hasAnyPermissionForNode(data)" 
                        src="/锁定.svg" 
                        alt="无权限" 
                        class="no-permission-icon" 
                        :title="'该节点没有权限'"
                      />
                      
                      <!-- 节点元信息：只显示已选择的权限提示 -->
                      <div class="node-meta">
                        <el-tag
                          v-if="getNodePermissionDisplayText(data.full_code_path)"
                          size="small"
                          :type="getNodePermissionTagType(data.full_code_path)"
                          class="permission-hint-tag"
                        >
                          {{ getNodePermissionDisplayText(data.full_code_path) }}
                        </el-tag>
                      </div>
                    </span>
                  </template>
                </el-tree>
              </div>
            </el-card>
          </div>

          <!-- 中间：权限选择区域 -->
          <div class="apply-main">
            <div v-if="currentScope" class="permission-scopes">
              <div class="scope-header-main">
                <div class="scope-title-main">
                    <el-icon><Document /></el-icon>
                  <span class="scope-name-main">{{ currentScope.displayName }}</span>
                  <el-tag
                    size="small"
                    :type="getScopeTagType(currentScope.resourceType)"
                  >
                    {{ getScopeTypeLabel(currentScope.resourceType) }}
                  </el-tag>
                  </div>
                </div>
              <div class="scope-path-main">
                <code>{{ currentScope.resourcePath }}</code>
                </div>
                
              <div class="permission-list">
                <!-- 角色选择区域 -->
                <div v-if="availableRoles.length > 0" class="role-selection-section">
                  <div class="role-selection-header">
                    <h4 class="role-selection-title">
                      <el-icon><UserFilled /></el-icon>
                      快速选择角色
                    </h4>
                    <el-alert
                      type="info"
                      :closable="false"
                      show-icon
                      class="role-tip"
                    >
                      <template #default>
                        <p class="tip-text">💡 选择角色后，系统会自动填充该角色的所有权限点</p>
                      </template>
                    </el-alert>
                  </div>
                  <div class="role-list">
                    <div class="role-cards">
                      <div
                        v-for="role in availableRoles"
                        :key="role.id"
                        class="role-card"
                        :class="{ 'is-selected': selectedRoleId === role.id }"
                        @click="handleRoleCardClick(role.id)"
                      >
                        <div class="role-card-header">
                          <span class="role-name">{{ role.name }}</span>
                          <div class="role-tags">
                            <el-tag v-if="role.is_default" type="warning" size="small">默认</el-tag>
                            <el-tag v-if="role.is_system" type="success" size="small">系统角色</el-tag>
                          </div>
                        </div>
                        <p class="role-description">{{ role.description || '无描述' }}</p>
                        <div class="role-permissions-preview">
                          <el-tag
                            v-for="(actions, resourceType) in getRolePermissions(role)"
                            :key="resourceType"
                            size="small"
                            class="permission-tag"
                          >
                            {{ getResourceTypeLabel(resourceType) }}: {{ actions.length }} 个权限
                          </el-tag>
                        </div>
                      </div>
                    </div>
                    <div v-if="selectedRoleId" class="role-selected-actions">
                      <el-button size="small" @click="clearRoleSelection">清除角色选择</el-button>
                    </div>
                  </div>
                </div>
                
                <!-- 如果没有可用角色，显示提示信息 -->
                <template v-if="availableRoles.length === 0 && !rolesLoading">
                  <el-divider />
                  <el-empty description="暂无可用角色" />
                </template>
              </div>
            </div>
            <div v-else class="empty-state">
              <el-empty description="请从左侧树中选择一个资源" />
            </div>
          </div>

          <!-- 右侧：申请表单 -->
          <div class="apply-sidebar-right">
            <el-card shadow="never" class="form-card">
              <template #header>
                <h3>提交申请</h3>
              </template>
              <el-form
                ref="formRef"
                :model="formData"
                :rules="rules"
                label-width="80px"
                class="apply-form"
              >
                <!-- 赋权对象选择 -->
                <el-form-item label="赋权对象">
                  <el-radio-group 
                    v-model="grantTargetType" 
                    class="grant-target-type-radio"
                  >
                    <el-radio label="self">给自己申请</el-radio>
                    <el-radio label="user">给其他用户申请</el-radio>
                    <el-radio label="department">给部门申请</el-radio>
                  </el-radio-group>
                  
                  <!-- 当前用户显示 -->
                  <div v-if="grantTargetType === 'self'" class="grant-target-display">
                    <div class="current-user-info">
                      <el-avatar :src="currentUser?.avatar" :size="32">
                        {{ currentUser?.username?.[0]?.toUpperCase() || 'U' }}
                      </el-avatar>
                      <div class="user-details">
                        <div class="user-name">{{ formatUserDisplayName(currentUser) }}</div>
                        <div class="user-email" v-if="currentUser?.email">{{ currentUser.email }}</div>
                        <!-- 组织架构信息 -->
                        <div v-if="currentUser?.department_name || currentUser?.department_full_path" class="user-org-info">
                          <el-icon><OfficeBuilding /></el-icon>
                          <span>{{ currentUser.department_name || currentUser.department_full_path }}</span>
                        </div>
                        <!-- Leader 信息 -->
                        <div v-if="currentUser?.leader_display_name || currentUser?.leader_username" class="user-leader-info">
                          <el-icon><UserFilled /></el-icon>
                          <span>{{ currentUser.leader_display_name || currentUser.leader_username }}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                  
                  <!-- 用户选择（多选） -->
                  <div v-if="grantTargetType === 'user'" class="grant-target-input">
                    <UsersWidget
                      :value="grantTargetUsersValue"
                      :field="grantTargetUsersField"
                      mode="edit"
                      field-path="grantTargetUsers"
                      @update:modelValue="handleGrantTargetUsersChange"
                    />
                  </div>
                  
                  <!-- 部门选择（支持多选） -->
                  <div v-if="grantTargetType === 'department'" class="grant-target-input">
                    <div>
                      <el-button
                        type="primary"
                        @click="showDepartmentSelector = true"
                        style="width: 100%"
                        :icon="selectedDepartments.length ? null : OfficeBuilding"
                      >
                        {{ selectedDepartments.length ? `已选择 ${selectedDepartments.length} 个部门` : '选择组织架构（可多选）' }}
                      </el-button>
                      <div v-if="selectedDepartments.length" class="selected-department-details">
                        <div
                          v-for="dept in selectedDepartments"
                          :key="dept.full_code_path"
                          class="selected-department-card"
                        >
                          <div class="department-content">
                            <img src="/组织架构.svg" alt="部门" class="department-icon" />
                            <div class="department-info">
                              <div class="department-name">{{ dept.name }}</div>
                              <div class="department-meta">
                                <span class="department-path">{{ dept.full_code_path }}</span>
                              </div>
                            </div>
                          </div>
                          <el-button
                            text
                            type="danger"
                            @click="removeSelectedDepartment(dept)"
                            :icon="Close"
                            circle
                            class="remove-btn"
                          />
                        </div>
                      </div>
                      <el-alert
                        type="info"
                        :closable="false"
                        show-icon
                        style="margin-top: 12px"
                      >
                        <template #default>
                          <div class="tip-content">
                            <p class="tip-text">可多选部门，每个部门将单独提交一条赋权申请</p>
                          </div>
                        </template>
                      </el-alert>
                    </div>
                  </div>
                </el-form-item>

                <!-- 审批人显示 -->
                <el-form-item label="审批人" v-if="approvers.length > 0">
                  <div class="approvers-display">
                    <UsersWidget
                      :value="approversFieldValue"
                      :field="approversField"
                      mode="detail"
                      field-path="approvers"
                    />
                  </div>
                </el-form-item>

                <!-- 有效期选择 -->
                <el-form-item label="有效期">
                  <el-radio-group v-model="formData.isPermanent">
                    <el-radio :label="true">永久</el-radio>
                    <el-radio :label="false">指定有效期</el-radio>
                  </el-radio-group>
                  <el-date-picker
                    v-if="!formData.isPermanent"
                    v-model="formData.endTime"
                    type="datetime"
                    placeholder="选择权限到期时间"
                    format="YYYY-MM-DD HH:mm:ss"
                    value-format="YYYY-MM-DDTHH:mm:ssZ"
                    style="width: 100%; margin-top: 12px"
                    :disabled-date="(date: Date) => date.getTime() < Date.now()"
                    :shortcuts="datePickerShortcuts"
                  />
                </el-form-item>

                <el-form-item label="申请理由" prop="reason">
                  <el-input
                    v-model="formData.reason"
                    type="textarea"
                    :rows="6"
                    placeholder="请填写申请权限的理由，以便管理员审核"
                    maxlength="500"
                    show-word-limit
                  />
                </el-form-item>

                <el-form-item>
                  <el-button
                    type="primary"
                    :loading="submitting"
                    @click="handleSubmit"
                    style="width: 100%"
                    :disabled="!canSubmit"
                  >
                    {{ submitButtonText }}
                  </el-button>
                  <el-button 
                    @click="handleCancel"
                    style="width: 100%; margin-top: 12px"
                  >
                    取消
                  </el-button>
                </el-form-item>
              </el-form>
            </el-card>
          </div>
        </div>
      </div>
    </el-card>
    </div>
  </div>

  <!-- 组织架构选择器对话框（多选） -->
  <DepartmentPickerDialog
    v-model="showDepartmentSelector"
    :initial-paths="grantTargetDepartmentPaths.join(',')"
    :multiple="true"
    @confirm="handleDepartmentsSelect"
  />
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElText, ElIcon, ElTree, ElDivider } from 'element-plus'
import { Document, Folder, Lock, OfficeBuilding, UserFilled, User, Close } from '@element-plus/icons-vue'
import ChartIcon from '@/shared/components/icons/ChartIcon.vue'
import TableIcon from '@/shared/components/icons/TableIcon.vue'
import FormIcon from '@/shared/components/icons/FormIcon.vue'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { 
  getPermissionDisplayName, 
  getPermissionScopes,
  getAvailablePermissions,
  getPermissionDescription,
  hasAnyPermissionForNode,
  hasPermission,
  DirectoryPermission,
  FunctionPermission,
  type PermissionScope,
  type PermissionResourceType
} from '@/utils/permission'
import { applyPermission, getWorkspacePermissions, addPermission, type AddPermissionReq } from '@/api/permission'
import { getDepartmentTree, getUsersByDepartment, type Department } from '@/api/department'
import type { FormInstance, FormRules } from 'element-plus'
import { getAppWithServiceTree } from '@/api/app'
import { getRolesForPermissionRequest, type Role, type RolePermission } from '@/api/role'
import { useAuthStore } from '@/stores/auth'
import type { ServiceTree, App } from '@/types'
import DepartmentPickerDialog from '@/shared/components/DepartmentPickerDialog.vue'
import type { UserInfo } from '@/types'
import UsersWidget from '@/shared/components/UsersWidget.vue'
import type { FieldConfig, FieldValue } from '@/core/types/field'
import { WidgetType } from '@/core/constants/widget'
import { isServiceTreeNodeAdmin, parseUsernameList } from '@/utils/permissionActors'
import { buildAppResourcePath, parseResourcePath } from '@/utils/resourcePath'
import { createStringFieldValue, createWidgetFieldConfig } from '@/utils/widgetFieldHelpers'

const route = useRoute()
const router = useRouter()

// 权限信息
const permissionInfo = ref({
  resource_path: '',
  action: '',
  action_display: '',
  error_message: '',
})

// 加载状态
const loading = ref(true)
const error = ref('')
const submitting = ref(false)

// 服务树数据（包含工作空间节点）
const serviceTree = ref<ServiceTree[]>([])
const treeRef = ref<InstanceType<typeof ElTree>>()
const treeProps = {
  children: 'children',
  label: 'name'
}

// 当前工作空间信息
const currentApp = ref<App | null>(null)

// 当前选中的资源路径
const selectedResourcePath = ref<string>('')
const defaultExpandedKeys = ref<string[]>([])

// 当前选中资源的权限范围
const currentScope = ref<PermissionScope | null>(null)

const getScopeTagType = (resourceType: PermissionResourceType): 'primary' | 'success' | 'warning' | 'info' => {
  switch (resourceType) {
    case 'function':
      return 'primary'
    case 'docs':
      return 'info'
    case 'board':
      return 'warning'
    default:
      return 'success'
  }
}

const getScopeTypeLabel = (resourceType: PermissionResourceType): string => {
  switch (resourceType) {
    case 'function':
      return '函数'
    case 'directory':
      return '目录'
    case 'app':
      return '资源根路径'
    case 'docs':
      return '文档'
    case 'board':
      return '讨论区'
    default:
      return '资源'
  }
}

// 角色选择相关
const availableRoles = ref<Role[]>([]) // 当前资源可用的角色列表
const selectedRoleId = ref<number | null>(null) // 选中的角色ID
const rolesLoading = ref(false) // 加载角色列表的状态

// 用户选择的资源路径（用于申请权限的资源）
// ⭐ 使用数组而不是 Set，确保 Vue 响应式系统能正确追踪变化
const selectedResourcePaths = ref<string[]>([])

// 所有资源的已有权限（从后端获取，仅用于显示）
// key: resourcePath, value: 该资源已有的权限（action -> hasPermission）
const existingPermissions = ref<Map<string, Record<string, boolean>>>(new Map())

// 表单数据
const formRef = ref<FormInstance>()
const formData = ref({
  reason: '',
  isPermanent: true,  // 是否永久权限
  endTime: null as string | null,  // 有效期结束时间（ISO 8601 格式）
})

// 表单验证规则
const rules: FormRules = {
  reason: [
    { min: 10, message: '申请理由至少需要10个字符（如果填写）', trigger: 'blur' },
  ],
}

// 计算应该选中的节点（基于已有权限和用户选择的资源）
const checkedNodeKeys = computed(() => {
  const keys: string[] = []
  // 遍历所有资源的已有权限
  for (const [resourcePath, existingPerms] of existingPermissions.value.entries()) {
    // 如果该资源有任何已有权限，则选中该节点
    const hasAnyExistingPerm = Object.values(existingPerms).some(hasPerm => hasPerm === true)
    if (hasAnyExistingPerm) {
      keys.push(resourcePath)
    }
  }
  // 添加用户选择的资源（用于申请权限）
  for (const resourcePath of selectedResourcePaths.value) {
    if (!keys.includes(resourcePath)) {
      keys.push(resourcePath)
    }
  }
  // 调试信息
  if (import.meta.env.DEV && selectedResourcePaths.value.length > 0) {
    console.log({
      selectedPaths: [...selectedResourcePaths.value],
      computedKeys: keys
    })
  }
  return keys
})

// 计算应该禁用的节点（已有权限的节点）
const disabledNodeKeys = computed(() => {
  const keys: string[] = []
  // 遍历所有资源的已有权限
  for (const [resourcePath, existingPerms] of existingPermissions.value.entries()) {
    // 如果该资源有任何已有权限，则禁用该节点
    const hasAnyExistingPerm = Object.values(existingPerms).some(hasPerm => hasPerm === true)
    if (hasAnyExistingPerm) {
      keys.push(resourcePath)
    }
  }
  return keys
})

// ==================== 赋权相关状态 ====================

// 获取当前用户
const authStore = useAuthStore()
const currentUser = computed(() => authStore.user)

// 检查是否是管理员
const isAdmin = (node: ServiceTree): boolean => {
  return isServiceTreeNodeAdmin(node, currentUser.value?.username)
}

// 检查当前节点是否有 manage 权限
// ⭐ 管理员也应该能够赋权，即使没有显式的 manage 权限
const hasManagePermission = computed(() => {
  if (!selectedResourcePath.value || !serviceTree.value.length) {
    return false
  }
  const node = findNodeInTree(serviceTree.value, selectedResourcePath.value)
  if (!node) return false
  
  // ⭐ 首先检查是否是管理员（管理员可以赋权）
  if (isAdmin(node)) {
    return true
  }
  
  // 检查是否有 admin 权限（根据资源类型）
  if (node.type === 'function') {
    return hasPermission(node, FunctionPermission.admin)
  } else if (node.type === 'package') {
    // ⭐ 所有 package 类型统一使用 directory:admin 权限（包括根目录/工作空间）
    return hasPermission(node, DirectoryPermission.admin)
  }
  return false
})

// 赋权对象类型：self（自己）、user（其他用户）、department（部门）
const grantTargetType = ref<'self' | 'user' | 'department'>('self')

// 赋权目标：个人（用户对象）或组织架构（部门路径，支持多选）
const grantTargetDepartment = ref<string>('') // 保留兼容，实际用 selectedDepartments
const selectedDepartments = ref<Department[]>([])

// 部门路径数组（用于提交和 initial-paths）
const grantTargetDepartmentPaths = computed(() => selectedDepartments.value.map(d => d.full_code_path))

// 对话框状态
const showDepartmentSelector = ref(false)

// 监听部门路径变化（兼容旧逻辑，selectedDepartments 为主）
watch(grantTargetDepartmentPaths, (paths) => {
  if (paths.length === 0) {
    grantTargetDepartment.value = ''
    return
  }
  grantTargetDepartment.value = paths.join(',')
})

// 处理部门选择（多选）
const handleDepartmentsSelect = (departments: Department[]) => {
  selectedDepartments.value = departments
}

const removeSelectedDepartment = (dept: Department) => {
  selectedDepartments.value = selectedDepartments.value.filter(d => d.full_code_path !== dept.full_code_path)
}

// 日期选择器快捷选项
const datePickerShortcuts = computed(() => {
  const now = new Date()
  
  return [
    {
      text: '1天后',
      value: () => {
        const date = new Date(now.getTime() + 1 * 24 * 60 * 60 * 1000)
        return date
      }
    },
    {
      text: '3天后',
      value: () => {
        const date = new Date(now.getTime() + 3 * 24 * 60 * 60 * 1000)
        return date
      }
    },
    {
      text: '7天后',
      value: () => {
        const date = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000)
        return date
      }
    },
    {
      text: '15天后',
      value: () => {
        const date = new Date(now.getTime() + 15 * 24 * 60 * 60 * 1000)
        return date
      }
    },
    {
      text: '1个月后',
      value: () => {
        const date = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000)
        return date
      }
    },
    {
      text: '3个月后',
      value: () => {
        const date = new Date(now.getTime() + 90 * 24 * 60 * 60 * 1000)
        return date
      }
    },
    {
      text: '6个月后',
      value: () => {
        const date = new Date(now.getTime() + 180 * 24 * 60 * 60 * 1000)
        return date
      }
    },
    {
      text: '1年后',
      value: () => {
        const date = new Date(now.getTime() + 365 * 24 * 60 * 60 * 1000)
        return date
      }
    },
    {
      text: '2年后',
      value: () => {
        const date = new Date(now.getTime() + 730 * 24 * 60 * 60 * 1000)
        return date
      }
    },
    {
      text: '3年后',
      value: () => {
        const date = new Date(now.getTime() + 1095 * 24 * 60 * 60 * 1000)
        return date
      }
    },
  ]
})

// 审批人（从当前选中资源的 admins 字段获取）
const approvers = computed(() => {
  if (!selectedResourcePath.value) return []
  const node = findNodeInTree(serviceTree.value, selectedResourcePath.value)
  if (!node || !node.admins) return []
  return parseUsernameList(node.admins)
})

const approversField = createWidgetFieldConfig({
  code: 'approvers',
  name: '审批人',
  widgetType: WidgetType.USERS
})

const approversFieldValue = computed(() => createStringFieldValue(
  approversField,
  approvers.value.join(','),
  {
    emptyRaw: '',
    display: approvers.value.join(',')
  }
))

// 部门列表（用于组织架构赋权）
const departmentTree = ref<Department[]>([])
const flatDepartmentList = computed(() => {
  const flatten = (depts: Department[]): Department[] => {
    const result: Department[] = []
    depts.forEach(dept => {
      result.push(dept)
      if (dept.children && dept.children.length > 0) {
        result.push(...flatten(dept.children))
      }
    })
    return result
  }
  return flatten(departmentTree.value)
})

// 格式化用户显示名称
function formatUserDisplayName(user: UserInfo | null): string {
  if (!user) return ''
  if (user.nickname) {
    return `${user.username}(${user.nickname})`
  }
  return user.username
}

const grantTargetUsersField = createWidgetFieldConfig({
  code: 'grantTargetUsers',
  name: '申请权限的用户',
  widgetType: WidgetType.USERS
})

const grantTargetUsersValue = ref<FieldValue>(createStringFieldValue(grantTargetUsersField, '', { emptyRaw: '' }))

// 处理赋权目标用户变化
const handleGrantTargetUsersChange = (value: FieldValue) => {
  grantTargetUsersValue.value = value
}

// 是否可以提交
const canSubmit = computed(() => {
  // ⭐ 必须选择了角色才能提交
  if (!selectedRoleId.value) {
    return false
  }
  if (grantTargetType.value === 'user') {
    return grantTargetUsersValue.value?.raw && String(grantTargetUsersValue.value.raw).trim() !== ''
  } else if (grantTargetType.value === 'department') {
    return selectedDepartments.value.length > 0
  }
  // self 类型总是可以提交（如果已选择角色）
  return true
})

// 提交按钮文本
const submitButtonText = computed(() => {
  if (grantTargetType.value === 'self') {
    return '提交申请'
  } else if (grantTargetType.value === 'user') {
    return '提交赋权'
  } else if (grantTargetType.value === 'department') {
    return '提交赋权'
  }
  return '提交申请'
})

// 获取函数图标组件（根据 template_type）
const getFunctionIcon = (data: ServiceTree) => {
  if (data.template_type === TEMPLATE_TYPE.TABLE) {
    return TableIcon
  } else if (data.template_type === TEMPLATE_TYPE.FORM) {
    return FormIcon
  } else if (data.template_type === TEMPLATE_TYPE.CHART) {
    return ChartIcon
  }
  // 默认使用 Document 图标（如果没有 template_type 或不是已知类型）
  return Document
}

// 获取节点图标样式类
const getNodeIconClass = (data: ServiceTree) => {
  if (data.type === 'package') {
    return 'package-icon'
  } else if (data.type === 'function') {
    // 根据 template_type 返回不同的样式类
    if (data.template_type === TEMPLATE_TYPE.TABLE) {
      return 'table-icon'
    } else if (data.template_type === TEMPLATE_TYPE.FORM) {
      return 'form-icon'
    } else if (data.template_type === TEMPLATE_TYPE.CHART) {
      return 'chart-icon'
    }
    return 'function-icon'
  } else if (data.type === 'docs') {
    return 'docs-icon'
  } else if (data.type === 'board') {
    return 'board-icon'
  }
  return 'function-icon'
}

// 获取模板类型标签
const getTemplateTypeLabel = (templateType: string) => {
  const labels: Record<string, string> = {
    table: '表格',
    form: '表单',
    chart: '图表'
  }
  return labels[templateType] || templateType
}

// 获取模板类型标签样式
const getTemplateTypeTagType = (templateType: string) => {
  const types: Record<string, string> = {
    table: 'primary',
    form: 'success',
    chart: 'warning'
  }
  return types[templateType] || 'info'
}

// 初始化权限信息
onMounted(async () => {
  // 从 URL 参数中获取权限信息
  const resource = route.query.resource as string
  const action = route.query.action as string  // 可选，用于默认选中
  const templateType = route.query.templateType as string  // 可选，函数类型（table、form、chart）
  const mode = route.query.mode as string  // 可选，模式：grant（授权模式）或 apply（申请模式，默认）

  // 如果 mode=grant，默认设置为授权模式（给其他用户赋权）
  if (mode === 'grant') {
    grantTargetType.value = 'user'
  }

  if (!resource) {
    error.value = '缺少必要的参数：resource'
    loading.value = false
    return
  }

  const resourcePath = decodeURIComponent(resource)
  permissionInfo.value = {
    resource_path: resourcePath,
    action: action || '',
    action_display: action ? getPermissionDisplayName(action) : '',
    error_message: '',
  }

  const parsedResourcePath = parseResourcePath(resourcePath)
  if (!parsedResourcePath) {
    error.value = '资源路径格式错误'
    loading.value = false
    return
  }

  const { user, app } = parsedResourcePath
  const workspaceResourcePath = buildAppResourcePath(user, app)

  // 加载服务树和工作空间信息
  try {
    // ⭐ 加载服务树
    const treeResponse = await getAppWithServiceTree(workspaceResourcePath)
    
    // ⭐ 直接使用 user 和 app 查询权限（无需查询 app_id，性能更好）
    const permissionsResponse = await getWorkspacePermissions({ resource_path: workspaceResourcePath }).catch(err => {
          console.warn('获取工作空间权限失败:', err)
          return null
        })
    
    if (treeResponse) {
      // 保存工作空间信息
      currentApp.value = treeResponse.app || null
      
      // ⭐ 保存已有权限
      if (permissionsResponse && permissionsResponse.records) {
        const permissionsMap = new Map<string, Record<string, boolean>>()
        
        // ⭐ 前端自己处理原始权限记录
        for (const record of permissionsResponse.records) {
          const resourcePath = record.resource
          const action = record.action
          
          if (!permissionsMap.has(resourcePath)) {
            permissionsMap.set(resourcePath, {})
          }
          const perms = permissionsMap.get(resourcePath)!
          perms[action] = true
        }
        
        existingPermissions.value = permissionsMap
        
        // ⭐ 更新树数据中的 disabled 字段（已有权限的节点应该禁用）
        updateTreeDisabledState()
      }
      
      // 🔥 直接使用后端返回的 service_tree（后端已经包含了 app 根节点）
      serviceTree.value = treeResponse.service_tree || []
      
      // 设置默认选中的资源
      selectedResourcePath.value = resourcePath
      
      // ⭐ 将资源路径添加到选中数组中（用于复选框显示）
      if (!selectedResourcePaths.value.includes(resourcePath)) {
        selectedResourcePaths.value.push(resourcePath)
      }
      
      // ⭐ 查找所有子资源（子目录和函数），并添加到选中数组
      // 注意：需要在服务树加载完成后才能查找子资源
      const childResources = findAllChildResources(resourcePath)
      childResources.forEach(childPath => {
        if (!selectedResourcePaths.value.includes(childPath)) {
          selectedResourcePaths.value.push(childPath)
        }
      })
      
      // ⭐ 在树渲染完成后设置复选框为选中状态（包括所有子节点）
      nextTick(() => {
        setTimeout(() => {
          const tree = treeRef.value
          if (tree) {
            // 设置当前节点复选框为选中状态
            tree.setChecked(resourcePath, true, false)
            
            // ⭐ 设置所有子节点的复选框为选中状态
            childResources.forEach(childPath => {
              // 检查子节点是否已有权限（已有权限的节点不应该被操作）
              const childExistingPerms = existingPermissions.value.get(childPath)
              const childHasAnyExistingPerm = childExistingPerms && Object.values(childExistingPerms).some(hasPerm => hasPerm === true)
              
              if (!childHasAnyExistingPerm) {
                // 设置子节点为选中状态
                tree.setChecked(childPath, true, false)
              }
            })
          }
        }, 300) // 延迟一点，确保树完全渲染（增加到 300ms 以确保子节点也能被正确选中）
      })
      
      // 展开到选中节点的路径（包括工作空间节点）
      const expandedPaths: string[] = []
      const appPath = `/${user}/${app}`
      
      // 如果选中的资源是工作空间本身，展开工作空间节点
      if (resourcePath === appPath) {
        expandedPaths.push(appPath)
      } else {
        // 否则展开工作空间节点，然后查找子节点路径
        expandedPaths.push(appPath)
        
        const findPath = (nodes: ServiceTree[], targetPath: string): boolean => {
          for (const node of nodes) {
            const fullPath = node.full_code_path
            
            if (fullPath === targetPath) {
              // 找到目标节点
              return true
            }
            
            if (node.children && node.children.length > 0) {
              if (findPath(node.children, targetPath)) {
                // 在子节点中找到目标，展开当前节点
                if (!expandedPaths.includes(fullPath)) {
                  expandedPaths.push(fullPath)
                }
                return true
              }
            }
          }
          return false
        }
        
        // 查找并展开路径
        const rootNode = serviceTree.value[0]
        findPath(rootNode?.children || [], resourcePath)
      }
      
      defaultExpandedKeys.value = expandedPaths
      
      // 注意：不在 onMounted 中直接定位，而是通过 watch 监听 treeRef 和 serviceTree 的变化
      // 在树完全渲染后再执行定位逻辑
      
      // 加载选中资源的权限范围
      await loadResourcePermissions(resourcePath, action, templateType)
      
      // 加载部门树（用于组织架构赋权）
      await loadDepartmentTree()
    } else {
      error.value = '无法加载服务树数据'
    }
  } catch (err: any) {
    console.error('加载服务树失败:', err)
    error.value = '加载服务树失败: ' + (err?.message || '未知错误')
  }

  loading.value = false
})

// 加载部门树
async function loadDepartmentTree() {
  try {
    const res = await getDepartmentTree()
    departmentTree.value = res.departments || []
  } catch (error: any) {
    console.warn('加载部门树失败:', error)
    // 不显示错误，因为赋权功能是可选的
  }
}

// 监听赋权对象类型变化，重置相关状态
watch(() => grantTargetType.value, (newType) => {
  if (newType === 'self') {
    selectedDepartments.value = []
  } else if (newType === 'user') {
    selectedDepartments.value = []
  }
})

// 在服务树中查找节点
const findNodeInTree = (nodes: ServiceTree[], path: string): ServiceTree | null => {
        for (const node of nodes) {
          if (node.full_code_path === path) {
            return node
          }
          if (node.children && node.children.length > 0) {
      const found = findNodeInTree(node.children, path)
            if (found) return found
          }
        }
        return null
      }

// 加载资源的权限范围
const loadResourcePermissions = async (resourcePath: string, defaultAction?: string, urlTemplateType?: string) => {
  // 解析资源路径
  const pathParts = resourcePath.split('/').filter(Boolean)
  
  if (pathParts.length < 2) {
    error.value = '资源路径格式错误'
    return
  }

  let resourceType: 'function' | 'directory' | 'app' | 'board' | 'docs' | undefined
  let templateType: string | undefined = urlTemplateType

  // 从服务树中查找节点
  const node = findNodeInTree(serviceTree.value, resourcePath)
  if (node) {
    const nodeType = (node as any).type || node.type
    if (nodeType === 'app') {
      resourceType = 'app'
    } else if (node.type === 'function') {
      resourceType = 'function'
      templateType = node.template_type || urlTemplateType
    } else if (node.type === 'package') {
      resourceType = 'directory'
    } else if (node.type === 'board') {
      resourceType = 'board'
    } else if (node.type === 'docs') {
      resourceType = 'docs'
    }
  } else {
    // 如果找不到节点，根据路径长度判断
    if (pathParts.length === 2) {
      resourceType = 'app'
    } else {
      resourceType = 'function'  // 默认按函数处理
    }
  }
  
  // 获取权限范围
  const parsed = resourcePath.split('/').filter(Boolean)
  const resourceName = parsed[parsed.length - 1] || '资源'
  
  // ⭐ 构建中文路径（使用节点的 name 字段）
  const buildChinesePath = (path: string): string => {
    const pathParts = path.split('/').filter(Boolean)
    const chineseParts: string[] = []
    
    // 从根路径开始，逐步构建路径
    let currentPath = ''
    for (let i = 0; i < pathParts.length; i++) {
      const pathPart = pathParts[i]
      if (!pathPart) {
        continue
      }
      currentPath += '/' + pathPart
      const node = findNodeInTree(serviceTree.value, currentPath)
      if (node && node.name) {
        chineseParts.push(node.name)
      } else {
        // 如果找不到节点，使用原始代码
        chineseParts.push(pathPart)
      }
    }
    
    return chineseParts.join(' / ')
  }
  
  const displayName = resourceType === 'function' 
    ? `函数：${node?.name || resourceName}` 
    : resourceType === 'directory' 
    ? `目录：${buildChinesePath(resourcePath)}` 
    : resourceType === 'docs'
    ? `文档：${node?.name || resourceName}`
    : resourceType === 'board'
    ? `讨论区：${node?.name || resourceName}`
    : `资源根路径：${node?.name || parsed[1] || '资源根路径'}`
  
  const permissions = getAvailablePermissions(resourcePath, resourceType, templateType)
  
  currentScope.value = {
    resourcePath,
    resourceType: resourceType || 'function',
    resourceName,
    displayName,
    permissions,
    quickSelect: resourceType === 'function' ? {
      label: '申请此函数的全部权限',
      actions: permissions.map(p => p.action)
    } : resourceType === 'directory' ? {
      label: '申请此目录的管理权限',
      actions: ['directory:admin']
    } : resourceType === 'docs' ? {
      label: '申请此文档的全部权限',
      actions: permissions.map(p => p.action)
    } : resourceType === 'board' ? {
      label: '申请此讨论区的全部权限',
      actions: permissions.map(p => p.action)
    } : {
      label: '申请此资源根路径的管理权限',
      actions: ['app:admin']
    }
  }
  
  // ⭐ 映射 URL 中的 action 到实际的权限点（向后兼容旧格式）
  // 注意：现在统一使用 function:* 格式，此映射仅用于向后兼容
  // table:update -> function:update
  // table:delete -> function:delete
  // table:read -> function:read
  // form:write -> function:write
  // chart:query -> function:read
  const mapActionToPermission = (action: string, templateType?: string): string => {
    // 向后兼容：映射旧格式到新格式
    if (action === 'table:update') {
      return 'function:update'
    } else if (action === 'table:delete') {
      return 'function:delete'
    } else if (action === 'table:read') {
      return 'function:read'
    } else if (action === 'form:write') {
      return 'function:write'
    } else if (action === 'chart:query') {
      return 'function:read'
    }
    // 如果已经是 function:* 格式，直接返回
    return action
  }
  
  // ⭐ 加载可用角色列表（根据资源类型过滤）
  if (node) {
    loadAvailableRoles(node.type, node.template_type || '')
  } else {
    // 如果找不到节点，根据 resourceType 推断
    if (resourceType === 'app') {
      loadAvailableRoles('app', '')
    } else if (resourceType === 'directory') {
      loadAvailableRoles('package', '')
    } else {
      loadAvailableRoles('function', templateType || '')
    }
  }
}

// 更新树数据中的 disabled 字段（只有已有权限的节点应该禁用，子节点不禁用以便可以点击）
const updateTreeDisabledState = () => {
  const updateNodeDisabled = (nodes: ServiceTree[]) => {
    for (const node of nodes) {
      const existingPerms = existingPermissions.value.get(node.full_code_path)
      const hasAnyExistingPerm = existingPerms && Object.values(existingPerms).some(hasPerm => hasPerm === true)
      
      // 只禁用已有权限的节点（不能取消选中），子节点不禁用以便可以点击查看权限
      ;(node as any).disabled = hasAnyExistingPerm
      
      // 递归处理子节点
      if (node.children && node.children.length > 0) {
        updateNodeDisabled(node.children)
      }
    }
  }
  
  updateNodeDisabled(serviceTree.value)
}


// 定位节点的函数（提取为独立函数，可复用）
const scrollToResourceNode = (resourcePath: string) => {
  if (!resourcePath || !treeRef.value) {
    return false
  }
  
  
  try {
    // 使用 el-tree 的内部 store 确保节点可见并展开所有父节点
    try {
      const node = (treeRef.value as any).store?.nodesMap?.[resourcePath]
      if (node) {
        node.visible = true
        // 确保所有父节点都展开
        let parent = node.parent
        while (parent) {
          if (!parent.expanded) {
            parent.expand()
          }
          parent = parent.parent
        }
      }
    } catch (e) {
      console.warn('无法访问 el-tree store:', e)
    }
    
    // 设置当前节点（高亮显示）- 确保在展开和滚动之前设置
    if (treeRef.value) {
      treeRef.value.setCurrentKey(resourcePath)
    }
    
    // 等待节点渲染和展开完成（减少延迟）
    nextTick(() => {
      setTimeout(() => {
        // 使用更可靠的方法：通过 el-tree 的内部 store 和 DOM 查找
        const scrollToNode = (attempt = 0) => {
          
          if (attempt > 10) {
            console.warn('❌ [定位节点] 无法定位到节点:', resourcePath, '已尝试', attempt, '次')
            return
          }
          
          if (!treeRef.value) {
            setTimeout(() => scrollToNode(attempt + 1), 100) // 从 150ms 减少到 100ms
            return
          }
          
          try {
            // 确保当前节点被正确选中（多次调用确保生效）
            if (treeRef.value) {
              treeRef.value.setCurrentKey(resourcePath)
              // 再次调用，确保样式生效
              nextTick(() => {
                if (treeRef.value) {
                  treeRef.value.setCurrentKey(resourcePath)
                }
              })
            }
            
            // 等待 DOM 更新
            nextTick(() => {
              const treeEl = treeRef.value?.$el
              if (!treeEl) {
                setTimeout(() => scrollToNode(attempt + 1), 100) // 从 150ms 减少到 100ms
                return
              }
              
              
              // 方法1: 通过 el-tree 的内部 store 获取节点，然后找到对应的 DOM 元素
              let targetElement: HTMLElement | null = null
              try {
                const store = (treeRef.value as any).store
                if (store && store.nodesMap) {
                  const node = store.nodesMap[resourcePath]
                  if (node) {
                    // 确保节点可见并展开父节点
                    node.visible = true
                    let parent = node.parent
                    while (parent) {
                      if (!parent.expanded) {
                        parent.expand()
                      }
                      parent = parent.parent
                    }
                    
                    // 通过节点的 key 查找 DOM 元素
                    // el-tree 会在节点元素上添加 data-key 属性（对应 node-key 的值）
                    targetElement = treeEl.querySelector(`[data-key="${resourcePath}"]`) as HTMLElement
                    
                    // 如果找不到 data-key，尝试通过节点的 index 或其他方式查找
                    if (!targetElement) {
                      // 遍历所有节点，通过 Vue 实例匹配
                      const allNodes = treeEl.querySelectorAll('.el-tree-node')
                      for (const nodeEl of Array.from(allNodes)) {
                        const vueInstance = (nodeEl as any).__vueParentComponent
                        if (vueInstance) {
                          const nodeData = vueInstance.props?.data || vueInstance.ctx?.data
                          if (nodeData && nodeData.full_code_path === resourcePath) {
                            targetElement = nodeEl as HTMLElement
                            break
                          }
                        }
                      }
                    }
                  }
                }
              } catch (e) {
                console.warn('无法访问 el-tree store:', e)
              }
              
              // 方法2: 如果还是找不到，通过 is-current 类查找
              if (!targetElement) {
                targetElement = treeEl.querySelector('.el-tree-node.is-current') as HTMLElement
              }
              
              // 方法3: 如果还是找不到，通过节点标签文本匹配
              if (!targetElement) {
                const pathParts = resourcePath.split('/').filter(Boolean)
                const targetName = pathParts[pathParts.length - 1]
                const allNodes = treeEl.querySelectorAll('.el-tree-node')
                for (const node of Array.from(allNodes)) {
                  const nodeEl = node as HTMLElement
                  const nodeLabel = nodeEl.querySelector('.node-label')?.textContent?.trim()
                  if (nodeLabel && nodeLabel === targetName) {
                    targetElement = nodeEl
                    break
                  }
                }
              }
              
              if (targetElement) {
                // 找到节点后，使用 scrollIntoView 滚动
                targetElement.scrollIntoView({
                  behavior: 'smooth',
                  block: 'center',
                  inline: 'nearest'
                })
                
                // 验证滚动是否成功（延迟检查）
                setTimeout(() => {
                  const rect = targetElement!.getBoundingClientRect()
                  const container = document.querySelector('.tree-container') as HTMLElement
                  if (container) {
                    const containerRect = container.getBoundingClientRect()
                    const isVisible = rect.top >= containerRect.top && rect.bottom <= containerRect.bottom
                    if (!isVisible && attempt < 5) {
                      // 如果不可见，手动计算滚动位置
                      const nodeTop = rect.top - containerRect.top + container.scrollTop
                      const nodeHeight = rect.height
                      const containerHeight = containerRect.height
                      const targetScrollTop = nodeTop - (containerHeight / 2) + (nodeHeight / 2)
                      container.scrollTop = Math.max(0, targetScrollTop)
                    } else {
                    }
                  }
                }, 200) // 从 300ms 减少到 200ms
              } else {
                // 如果找不到，继续尝试
                setTimeout(() => scrollToNode(attempt + 1), 150) // 从 250ms 减少到 150ms
              }
            })
          } catch (error) {
            console.error('❌ [定位节点] 定位失败:', error)
            setTimeout(() => scrollToNode(attempt + 1), 150) // 从 200ms 减少到 150ms
          }
        }
        
        scrollToNode()
      }, 200) // 从 500ms 减少到 200ms
    })
    
    return true
  } catch (error) {
    console.error('❌ [定位节点] 定位失败:', error)
    return false
  }
}

// 监听 treeRef 和 selectedResourcePath 的变化，在树渲染完成后自动定位
let scrollTimeout: ReturnType<typeof setTimeout> | null = null
watch([() => treeRef.value, () => selectedResourcePath.value, () => serviceTree.value.length], 
  ([newTreeRef, newPath, treeLength]) => {
    if (newTreeRef && newPath && treeLength > 0) {
      // 清除之前的延迟
      if (scrollTimeout) {
        clearTimeout(scrollTimeout)
      }
      // 延迟执行，确保 DOM 完全渲染（减少延迟时间）
      nextTick(() => {
        scrollTimeout = setTimeout(() => {
          scrollToResourceNode(newPath)
        }, 100) // 从 300ms 减少到 100ms
      })
    }
  },
  { immediate: true }
)

// 监听 selectedResourcePath 变化，自动滚动到对应节点
watch(() => selectedResourcePath.value, (newPath) => {
  if (!newPath || !treeRef.value) return
  
  // 等待 DOM 更新
  nextTick(() => {
    setTimeout(() => {
      try {
        // 确保当前节点被正确选中
        treeRef.value?.setCurrentKey(newPath)
        
        nextTick(() => {
          const treeEl = treeRef.value?.$el
          if (!treeEl) return
          
          // 通过 is-current 类查找
          let targetElement = treeEl.querySelector('.el-tree-node.is-current') as HTMLElement
          
          // 如果找不到，遍历所有节点查找
          if (!targetElement) {
            const allNodes = treeEl.querySelectorAll('.el-tree-node')
            for (const node of Array.from(allNodes)) {
              const nodeEl = node as HTMLElement
              const vueInstance = (nodeEl as any).__vueParentComponent
              if (vueInstance) {
                const nodeData = vueInstance.props?.data || vueInstance.ctx?.data
                if (nodeData && nodeData.full_code_path === newPath) {
                  targetElement = nodeEl
                  nodeEl.classList.add('is-current')
                  break
                }
              }
            }
          }
          
          if (targetElement) {
            targetElement.scrollIntoView({
              behavior: 'smooth',
              block: 'center',
              inline: 'nearest'
            })
          }
        })
      } catch (error) {
        console.error('监听 selectedResourcePath 变化时定位失败:', error)
      }
    }, 300)
  })
}, { immediate: false })

// 监听已有权限变化，更新树节点的选中和禁用状态
watch([existingPermissions], () => {
  // 更新树数据中的 disabled 字段
  updateTreeDisabledState()
  
  // 更新树节点的选中状态（仅基于已有权限）
  nextTick(() => {
    if (!treeRef.value) return
    
    // 遍历所有资源，设置选中状态
    for (const [resourcePath, existingPerms] of existingPermissions.value.entries()) {
      const hasAnyExistingPerm = Object.values(existingPerms).some(hasPerm => hasPerm === true)
      // 设置选中状态（仅基于已有权限）
      treeRef.value.setChecked(resourcePath, hasAnyExistingPerm, false)
    }
  })
}, { deep: true })



// 检查权限是否有继承（目录和工作空间权限会继承到子资源）
const hasInheritance = (action: string, resourceType?: string): boolean => {
  // 目录权限会继承到子资源
  if (action.startsWith('directory:')) {
    return true
  }
  // 工作空间权限会继承到子资源
  if (action.startsWith('app:')) {
    return true
  }
  // 函数权限不会继承到子资源（但会被父资源继承）
  return false
}

// 获取继承权限对应的子资源权限描述
const getInheritanceText = (action: string): string => {
  // 根据权限类型获取对应的子资源权限名称
  const permissionNameMap: Record<string, string> = {
    'directory:read': '查看权限',
    'directory:write': '写入权限',
    'directory:update': '更新权限',
    'directory:delete': '删除权限',
    'directory:admin': '所有权',
    'app:read': '查看权限',
    'app:create': '创建权限',
    'app:update': '更新权限',
    'app:delete': '删除权限',
    'app:admin': '所有权',
  }
  
  const permissionName = permissionNameMap[action] || '对应权限'
  return `包含子资源${permissionName}`
}

// 获取权限的简化显示名称（用于树节点显示，去掉前缀）
const getSimplifiedPermissionName = (action: string): string => {
  const fullName = getPermissionDisplayName(action)
  
  // ⭐ 如果权限显示名称就是 action 本身（说明没有映射），返回空字符串，表示不显示
  if (fullName === action) {
    return ''
  }
  
  // 简化规则映射表
  const simplifiedMap: Record<string, string> = {
    '新增表格记录': '新增',
    '更新表格记录': '更新',
    '删除表格记录': '删除',
    '表单提交': '提交',
    '目录查看': '查看',
    '目录写入': '写入',
    '目录更新': '更新',
    '目录删除': '删除',
    '工作空间查看': '查看',
    '工作空间创建': '创建',
    '工作空间更新': '更新',
    '工作空间删除': '删除',
    '工作空间部署': '部署',
    '函数查看': '查看',
    '所有权': '所有权',  // 保持不变
  }
  
  // 如果映射表中有，使用简化名称
  if (fullName in simplifiedMap) {
    const simplifiedName = simplifiedMap[fullName]
    if (simplifiedName) {
      return simplifiedName
    }
  }
  
  // 如果没有映射，尝试通用简化：去掉"表格"、"表单"、"目录"、"工作空间"等前缀
  let simplified = fullName
    .replace(/^新增表格记录$/, '新增')
    .replace(/^更新表格记录$/, '更新')
    .replace(/^删除表格记录$/, '删除')
    .replace(/^表单提交$/, '提交')
    .replace(/^目录(.+)$/, '$1')
    .replace(/^工作空间(.+)$/, '$1')
    .replace(/^函数(.+)$/, '$1')
  
  // 如果简化后还是原名称，说明无法简化，返回空字符串
  if (simplified === fullName) {
    return ''
  }
  
  return simplified
}

// 获取权限的简短标识（用于树节点显示）
const getPermissionShortLabel = (action: string): string | null => {
  const labelMap: Record<string, string> = {
    // 目录权限
    'directory:read': '读',
    'directory:write': '写',
    'directory:update': '改',
    'directory:delete': '删',
    'directory:admin': '所有权',
    // 工作空间权限
    'app:read': '读',
    'app:create': '创建',
    'app:update': '改',
    'app:delete': '删',
    'app:admin': '所有权',
    // 函数权限
    'function:read': '读',
    'function:write': '写',
    'function:update': '改',
    'function:delete': '删',
    'function:admin': '所有权',
  }
  return labelMap[action] || null
}

// 获取节点权限显示文本（用于树节点显示，显示简短标识：读、写、改、删、所有权）
const getNodePermissionDisplayText = (resourcePath: string): string | null => {
  // ⭐ 收集已有权限
  const existingPerms = existingPermissions.value.get(resourcePath)
  const existingPermissionsList: string[] = []
  if (existingPerms) {
    for (const [action, hasPerm] of Object.entries(existingPerms)) {
      if (hasPerm) {
        existingPermissionsList.push(action)
      }
    }
  }
  
  // 如果既没有已有权限，返回 null
  if (existingPermissionsList.length === 0) {
    return null
  }
  
  // ⭐ 只显示已有权限（使用简短标识）
    // 检查是否有管理权限（优先级最高）
    if (existingPermissionsList.some(p => p === 'directory:admin' || p === 'app:admin' || p === 'function:admin')) {
    return '所有权'
  }
  
      // 显示所有已有权限的简短标识
      const labels = existingPermissionsList
        .map(action => getPermissionShortLabel(action))
        .filter(label => label !== null) as string[]
  
  if (labels.length === 0) {
    return null
  }
  
  // 最多显示3个权限标识
  return labels.slice(0, 3).join('、')
}

// 获取节点权限标签的类型（已有权限用 info）
const getNodePermissionTagType = (resourcePath: string): 'info' | 'success' => {
  const existingPerms = existingPermissions.value.get(resourcePath)
  if (existingPerms) {
    const hasAnyPermission = Object.values(existingPerms).some(v => v === true)
    if (hasAnyPermission) {
      return 'info'  // 已有权限用 info 类型（蓝色）
    }
  }
  return 'info'  // 默认使用 info 类型
}


// 查找所有子资源（递归）
const findAllChildResources = (parentPath: string): string[] => {
  const childPaths: string[] = []
  
  // 递归遍历函数，找到所有子节点
  const traverse = (node: ServiceTree) => {
    if (!node.full_code_path) return
    
    // 如果节点是父路径的子节点（不是父路径本身）
    if (node.full_code_path !== parentPath && node.full_code_path.startsWith(parentPath + '/')) {
      childPaths.push(node.full_code_path)
    }
    
    // 继续遍历子节点
    if (node.children && node.children.length > 0) {
      for (const child of node.children) {
        traverse(child)
      }
    }
  }
  
  // 从服务树的根节点开始遍历
  for (const rootNode of serviceTree.value) {
    traverse(rootNode)
  }
  
  return childPaths
}

// 将父资源的权限映射到子资源
const mapPermissionsForChild = (childPath: string, childNode: ServiceTree, parentPermissions: string[]): string[] => {
  const childPermissions: string[] = []
  
  // 检查父资源选择的权限
  for (const parentAction of parentPermissions) {
    if (parentAction === 'directory:read') {
      // 查看权限：子节点继承查看权限
      if (childNode.type === 'package') {
        // 子目录：继承 directory:read
        if (!childPermissions.includes('directory:read')) {
          childPermissions.push('directory:read')
        }
      } else if (childNode.type === 'function') {
        // ⭐ 统一权限点：所有函数类型统一使用 function:read
        // 子函数：映射为 function:read
        if (!childPermissions.includes('function:read')) {
          childPermissions.push('function:read')
        }
      }
    } else if (parentAction === 'directory:admin' || parentAction === 'app:admin') {
      // 管理权限：子节点显示"所有权"
      if (childNode.type === 'package') {
        // 子目录：保存 directory:admin（显示时会显示为"所有权"）
        if (!childPermissions.includes('directory:admin')) {
          childPermissions.push('directory:admin')
        }
      } else if (childNode.type === 'function') {
        // 子函数：保存所有相关权限，但显示时会显示为"所有权"
        // ⭐ 统一权限点：所有函数类型统一使用 function:read/write/update/delete
        const childType = childNode.template_type
        if (childType === TEMPLATE_TYPE.TABLE) {
          // table 类型：使用 function:read/write/update/delete
          if (!childPermissions.includes('function:read')) childPermissions.push('function:read')
          if (!childPermissions.includes('function:write')) childPermissions.push('function:write')
          if (!childPermissions.includes('function:update')) childPermissions.push('function:update')
          if (!childPermissions.includes('function:delete')) childPermissions.push('function:delete')
        } else if (childType === TEMPLATE_TYPE.FORM) {
          // form 类型：使用 function:write（虽然定义了 read/update/delete，但业务逻辑中不使用）
          if (!childPermissions.includes('function:write')) childPermissions.push('function:write')
        } else if (childType === TEMPLATE_TYPE.CHART) {
          // chart 类型：使用 function:read（虽然定义了 write/update/delete，但业务逻辑中不使用）
          if (!childPermissions.includes('function:read')) childPermissions.push('function:read')
    } else {
          // 其他类型：使用 function:read/write/update/delete
          if (!childPermissions.includes('function:read')) childPermissions.push('function:read')
          if (!childPermissions.includes('function:write')) childPermissions.push('function:write')
          if (!childPermissions.includes('function:update')) childPermissions.push('function:update')
          if (!childPermissions.includes('function:delete')) childPermissions.push('function:delete')
        }
        // 所有权权限
        if (!childPermissions.includes('function:admin')) childPermissions.push('function:admin')
        // 添加一个特殊标记，表示这是管理权限下的子节点
        if (!childPermissions.includes('_has_admin_permission')) {
          childPermissions.push('_has_admin_permission')
        }
      }
    } else if (parentAction === 'directory:write') {
      // 写入权限：子节点继承写入权限（只继承给 table 和 form）
      if (childNode.type === 'package') {
        // 子目录：继承 directory:write
        if (!childPermissions.includes('directory:write')) {
          childPermissions.push('directory:write')
        }
      } else if (childNode.type === 'function') {
        // ⭐ 统一权限点：所有函数类型统一使用 function:write
        // 子函数：根据类型映射写入权限（只继承给 table 和 form）
        const childType = childNode.template_type
        if (childType === TEMPLATE_TYPE.TABLE || childType === TEMPLATE_TYPE.FORM) {
          // table 和 form 类型：映射为 function:write
          if (!childPermissions.includes('function:write')) {
            childPermissions.push('function:write')
          }
        }
        // chart 和其他类型：不继承 write 权限（用户要求不要乱映射）
      }
    } else if (parentAction === 'directory:update') {
      // 更新权限：子目录继承更新权限
      if (childNode.type === 'package') {
        // 子目录：继承 directory:update
        if (!childPermissions.includes('directory:update')) {
          childPermissions.push('directory:update')
        }
      } else if (childNode.type === 'function') {
        // ⭐ 统一权限点：table 类型使用 function:update
        const childType = childNode.template_type
        if (childType === TEMPLATE_TYPE.TABLE) {
          // table 类型：映射为 function:update
          if (!childPermissions.includes('function:update')) {
            childPermissions.push('function:update')
          }
        }
        // form、chart 和其他类型：不继承 update 权限（只有 table 有 update）
      }
    } else if (parentAction === 'directory:delete') {
      // 删除权限：子目录继承删除权限
      if (childNode.type === 'package') {
        // 子目录：继承 directory:delete
        if (!childPermissions.includes('directory:delete')) {
          childPermissions.push('directory:delete')
        }
      } else if (childNode.type === 'function') {
        // ⭐ 统一权限点：table 类型使用 function:delete
        const childType = childNode.template_type
        if (childType === TEMPLATE_TYPE.TABLE) {
          // table 类型：映射为 function:delete
          if (!childPermissions.includes('function:delete')) {
            childPermissions.push('function:delete')
          }
        }
        // form、chart 和其他类型：不继承 delete 权限（只有 table 有 delete）
      }
    } else if (parentAction === 'app:read') {
      // 工作空间查看权限：子节点继承查看权限
      if (childNode.type === 'package') {
        // 子目录：继承 directory:read
        if (!childPermissions.includes('directory:read')) {
          childPermissions.push('directory:read')
        }
      } else if (childNode.type === 'function') {
        // ⭐ 统一权限点：所有函数类型统一使用 function:read
        const childType = childNode.template_type
        if (childType === TEMPLATE_TYPE.TABLE || childType === TEMPLATE_TYPE.CHART || !childType) {
          // table、chart 和其他类型：使用 function:read
          if (!childPermissions.includes('function:read')) childPermissions.push('function:read')
        }
        // form 类型：虽然定义了 function:read，但业务逻辑中不使用（form 只有 write 权限）
      }
    }
  }
  
  return childPermissions
}

// 处理树节点点击
const handleTreeNodeClick = (data: ServiceTree) => {
  selectedResourcePath.value = data.full_code_path
  // ⭐ 将选中的资源添加到选中数组中（用于复选框显示）
  if (!selectedResourcePaths.value.includes(data.full_code_path)) {
    selectedResourcePaths.value.push(data.full_code_path)
  }
    loadResourcePermissions(data.full_code_path)
  // ⭐ 加载可用角色列表（根据资源类型过滤）
  loadAvailableRoles(data.type, data.template_type || '')
  
  // ⭐ 设置复选框为选中状态（在 nextTick 中执行，确保响应式更新完成）
  nextTick(() => {
    if (treeRef.value) {
      // 使用 setChecked 方法确保复选框被选中
      treeRef.value.setChecked(data.full_code_path, true, false)
    }
  })
}

// 处理树节点复选框变化（强制继承：父节点选中/取消时，子节点必须跟随）
const handleTreeNodeCheck = (data: ServiceTree, checked: { checkedKeys: string[], halfCheckedKeys: string[] }) => {
  const resourcePath = data.full_code_path
  const isChecked = checked.checkedKeys.includes(resourcePath)
  
  // 检查节点是否已有权限（如果已有权限，不应该取消选中）
  const existingPerms = existingPermissions.value.get(resourcePath)
  const hasAnyExistingPerm = existingPerms && Object.values(existingPerms).some(hasPerm => hasPerm === true)
  
  // ⭐ 允许所有节点（包括目录节点）直接操作复选框
  // 不再阻止有父节点的节点操作复选框，因为用户需要能够选中目录节点来申请权限
  
  // 如果节点已有权限，不允许取消选中（应该通过禁用来防止）
  if (hasAnyExistingPerm && !isChecked) {
    // 恢复选中状态
    nextTick(() => {
      if (treeRef.value) {
        treeRef.value.setChecked(resourcePath, true, false)
      }
    })
    return
  }
  
  if (isChecked) {
    // 节点被选中：加载该节点的权限范围
    // 如果节点已有权限，不需要做任何操作（因为已有权限的节点应该是禁用且选中的）
    if (!hasAnyExistingPerm) {
      // ⭐ 添加到选中数组
      if (!selectedResourcePaths.value.includes(resourcePath)) {
        selectedResourcePaths.value.push(resourcePath)
      }
      // 如果节点没有已有权限，加载该节点的权限范围
      loadResourcePermissions(resourcePath)
      
      // ⭐ 强制继承：自动选中所有子节点（包括子目录和子函数）
      const childResources = findAllChildResources(resourcePath)
      if (childResources.length > 0) {
        
        nextTick(() => {
          const tree = treeRef.value
          if (tree) {
            childResources.forEach(childPath => {
              // 检查子节点是否已有权限（已有权限的节点不应该被操作）
              const childExistingPerms = existingPermissions.value.get(childPath)
              const childHasAnyExistingPerm = childExistingPerms && Object.values(childExistingPerms).some(hasPerm => hasPerm === true)
              
              if (!childHasAnyExistingPerm) {
                // 设置子节点为选中状态
                tree.setChecked(childPath, true, false)
                // 添加到选中数组
                if (!selectedResourcePaths.value.includes(childPath)) {
                  selectedResourcePaths.value.push(childPath)
                }
              }
            })
          }
        })
      }
    }
  } else {
    // 父节点被取消选中：强制取消所有子节点
    // 如果节点已有权限，不允许取消选中（应该通过禁用来防止）
    if (!hasAnyExistingPerm) {
      // ⭐ 从选中数组中移除
      const index = selectedResourcePaths.value.indexOf(resourcePath)
      if (index > -1) {
        selectedResourcePaths.value.splice(index, 1)
      }
      // 如果当前选中的资源就是这个节点，清空当前范围
      if (selectedResourcePath.value === resourcePath) {
        currentScope.value = null
        selectedRoleId.value = null
      }
      
      // ⭐ 强制继承：取消所有子节点（包括子目录和子函数）
      const childResources = findAllChildResources(resourcePath)
      
      childResources.forEach(childPath => {
        // ⭐ 从选中数组中移除子节点
        const childIndex = selectedResourcePaths.value.indexOf(childPath)
        if (childIndex > -1) {
          selectedResourcePaths.value.splice(childIndex, 1)
        }
        // 取消选中子节点的复选框
        nextTick(() => {
          if (treeRef.value) {
            const childExistingPerms = existingPermissions.value.get(childPath)
            const childHasAnyExistingPerm = childExistingPerms && Object.values(childExistingPerms).some(hasPerm => hasPerm === true)
            // 只有非禁用的子节点才能取消选中
            if (!childHasAnyExistingPerm) {
              treeRef.value.setChecked(childPath, false, false)
            }
          }
        })
      })
    } else {
      // 如果节点已有权限但用户尝试取消选中，重新选中它
      nextTick(() => {
        if (treeRef.value) {
          treeRef.value.setChecked(resourcePath, true, false)
        }
      })
    }
  }
}

// 获取父节点路径
const getParentPath = (resourcePath: string): string | null => {
  const pathParts = resourcePath.split('/').filter(Boolean)
  if (pathParts.length <= 2) {
    // 根节点或工作空间节点，没有父节点
    return null
  }
  // 返回父节点路径（去掉最后一个部分）
  return '/' + pathParts.slice(0, -1).join('/')
}


// 提交申请/赋权
const handleSubmit = async () => {
  if (!formRef.value) return

  // ⭐ 检查是否选择了角色（权限申请必须通过角色）
  if (!selectedRoleId.value) {
    ElMessage.warning('请先选择一个角色')
    return
  }

  // 检查赋权对象是否有效
  if (!canSubmit.value) {
    if (grantTargetType.value === 'user') {
      ElMessage.warning('请选择要赋权的用户')
    } else if (grantTargetType.value === 'department') {
      ElMessage.warning('请选择要赋权的部门')
    }
    return
  }

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  submitting.value = true

  try {
    // ⭐ 检查是否选择了角色（权限申请必须通过角色）
    if (!selectedRoleId.value) {
      ElMessage.warning('请先选择一个角色')
      return
    }

    if (!currentScope.value) {
      ElMessage.warning('请选择一个资源')
      return
    }

    const resourcePath = currentScope.value.resourcePath

    // 准备有效期参数
    const endTime = formData.value.isPermanent ? undefined : (formData.value.endTime || undefined)

    // ⭐ 统一使用申请流程（不再区分申请和赋权）
    // 所有权限申请都需要经过审批流程
    let subjectType: 'user' | 'department' = 'user'
    let subject: string = ''
    
    if (grantTargetType.value === 'self') {
      // 给自己申请权限
      subjectType = 'user'
      subject = '' // 后端会使用当前用户
    } else if (grantTargetType.value === 'user') {
      // 给其他用户申请权限（支持多选）
      const selectedUsernames = grantTargetUsersValue.value?.raw
      if (!selectedUsernames || !String(selectedUsernames).trim()) {
        ElMessage.warning('请至少选择一个要申请权限的用户')
        return
      }
      subjectType = 'user'
      subject = String(selectedUsernames).trim() // 多个用户名用逗号分隔
    } else if (grantTargetType.value === 'department') {
      // 给部门申请权限（支持多选，每个部门单独提交一条）
      if (selectedDepartments.value.length === 0) {
        ElMessage.warning('请至少选择一个要申请权限的部门')
        return
      }
      subjectType = 'department'
      // 多部门：循环提交，每个部门一条申请
      for (const dept of selectedDepartments.value) {
        await applyPermission({
          resource_path: resourcePath,
          role_id: selectedRoleId.value,
          subject_type: 'department',
          subject: dept.full_code_path,
          reason: formData.value.reason,
          end_time: endTime,
        })
      }
      const targetText = `共 ${selectedDepartments.value.length} 个部门`
      ElMessage.success(`赋权成功：${targetText}`)
      router.push('/workspace/' + (currentApp.value?.user || '') + '/' + (currentApp.value?.code || ''))
      return
    }

    // ⭐ 提交权限申请（必须通过角色申请）
    await applyPermission({
      resource_path: resourcePath,
      role_id: selectedRoleId.value,
      subject_type: subjectType,
      subject: subject,
      reason: formData.value.reason,
      end_time: endTime,
    })
    
    const targetText = grantTargetType.value === 'self' 
      ? '自己' 
      : grantTargetType.value === 'user' 
      ? `用户 "${grantTargetUsersValue.value?.display || grantTargetUsersValue.value?.raw || ''}"` 
      : `部门 "${grantTargetDepartment.value}"`
    
    ElMessage.success(`已为${targetText}提交权限申请，等待审批`)
    
    // 延迟后返回上一页
    setTimeout(() => {
      router.back()
    }, 1500)
  } catch (err: any) {
    // 显示详细的错误信息
    const errorMessage = err?.response?.data?.msg || err?.message || '提交失败'
    ElMessage.error(errorMessage)
  } finally {
    submitting.value = false
  }
}

// 取消申请
const handleCancel = () => {
  router.back()
}

// ⭐ 加载可用角色列表（根据资源类型过滤）
const loadAvailableRoles = async (nodeType: string, templateType: string) => {
  try {
    rolesLoading.value = true
    selectedRoleId.value = null // 清空之前的选择
    
    // 检查 nodeType 是否为空
    if (!nodeType || nodeType.trim() === '') {
      console.warn('[PermissionApply] nodeType 为空，跳过加载角色列表')
      availableRoles.value = []
      return
    }
    
    
    // 调用 API 获取可用角色
    const resp = await getRolesForPermissionRequest({
      node_type: nodeType,
      template_type: templateType && templateType.trim() !== '' ? templateType : undefined,
    })
    
    
    if (resp && resp.roles) {
      availableRoles.value = resp.roles
      
      // ⭐ 自动选择默认角色（如果有的话）
      const defaultRole = resp.roles.find((role: Role) => role.is_default)
      if (defaultRole && !selectedRoleId.value) {
        selectedRoleId.value = defaultRole.id
      }
    } else {
      availableRoles.value = []
      console.warn('[PermissionApply] 角色列表为空')
    }
  } catch (error: any) {
    console.error('[PermissionApply] 加载角色列表失败:', error)
    // 不显示错误提示，因为角色功能是可选的
    availableRoles.value = []
  } finally {
    rolesLoading.value = false
  }
}

// ⭐ 获取角色的权限点（按资源类型分组）
const getRolePermissions = (role: Role): Record<string, string[]> => {
  if (!role.permissions || role.permissions.length === 0) {
    return {}
  }
  
  // 按资源类型分组
  const grouped: Record<string, string[]> = {}
  for (const perm of role.permissions) {
    const resourceType = perm.resource_type
    if (!grouped[resourceType]) {
      grouped[resourceType] = []
    }
    grouped[resourceType]!.push(perm.action)
  }
  
  return grouped
}

// ⭐ 获取资源类型标签
const getResourceTypeLabel = (resourceType: string): string => {
  const labels: Record<string, string> = {
    'app': '工作空间',
    'directory': '目录',
    'function': '函数',
    'function:table': '表格',
    'function:form': '表单',
    'function:chart': '图表',
    'table': '表格',
    'form': '表单',
    'chart': '图表',
    'docs': '文档',
    'board': '讨论区',
  }
  return labels[resourceType] || resourceType
}

// ⭐ 处理角色卡片点击
const handleRoleCardClick = (roleId: number) => {
  if (selectedRoleId.value === roleId) {
    // 如果点击的是已选中的角色，取消选择
    clearRoleSelection()
  } else {
    // 选择新角色
    selectedRoleId.value = roleId
    handleRoleSelect(roleId)
  }
}

// ⭐ 处理角色选择
const handleRoleSelect = (roleId: number) => {
  const role = availableRoles.value.find(r => r.id === roleId)
  if (!role) {
    return
  }
  
  ElMessage.success(`已选择角色"${role.name}"`)
}

// ⭐ 清除角色选择
const clearRoleSelection = () => {
  selectedRoleId.value = null
  ElMessage.info('已清除角色选择')
}

</script>

<style scoped lang="scss">
.permission-apply-wrapper {
  width: 100%;
  height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--el-bg-color-page);
  padding: 24px;
  box-sizing: border-box;
}

.permission-apply {
  max-width: 1600px;
  margin: 0 auto;
  padding-bottom: 40px;

  .apply-card {
    border-radius: 12px;
    border: none;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
    background: var(--el-bg-color);

    :deep(.el-card__header) {
      padding: 20px 24px;
      border-bottom: 1px solid var(--el-border-color-lighter);
      background: var(--el-fill-color-lighter);
      border-radius: 12px 12px 0 0;
    }

    :deep(.el-card__body) {
      padding: 24px;
    }

    .card-header {
      display: flex;
      align-items: center;
      gap: 12px;

      h2 {
        margin: 0;
        font-size: 22px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }
    }

    .loading-container {
      padding: 20px;
    }

    .error-container {
      padding: 20px;
    }

    .apply-content {
      .apply-layout {
        display: grid;
        grid-template-columns: 400px 1fr 480px;
        gap: 24px;
        align-items: start;
        
        // 响应式调整
        @media (max-width: 1600px) {
          grid-template-columns: 350px 1fr 450px;
        }
        
        @media (max-width: 1400px) {
          grid-template-columns: 320px 1fr 400px;
        }
      }

      .apply-sidebar {
        position: sticky;
        top: 24px;

        .tree-card {
          border-radius: 12px;
          border: 1px solid var(--el-border-color-lighter);
          background: var(--el-bg-color);

          :deep(.el-card__header) {
            padding: 16px 20px;
            border-bottom: 1px solid var(--el-border-color-lighter);
            background: var(--el-fill-color-lighter);
            border-radius: 12px 12px 0 0;

            h3 {
              margin: 0;
              font-size: 16px;
              font-weight: 600;
              color: var(--el-text-color-primary);
            }
          }

          :deep(.el-card__body) {
            padding: 20px;
          }

          .tree-container {
            max-height: calc(100vh - 200px);
            overflow-y: auto;
            
            .resource-tree {
              :deep(.el-tree-node__content) {
                height: auto;
                padding: 0;
                margin-bottom: 2px;
              }
              
              :deep(.el-tree-node__content:hover) {
                background-color: transparent;
              }
              
              :deep(.el-tree-node__expand-icon) {
                padding: 6px;
                transition: all 0.2s ease;
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
                background-color: transparent;
                font-weight: normal;
              }
              
              .tree-node {
                display: flex;
                align-items: center;
                gap: 8px;
                flex: 1;
                width: 100%;
                
                .node-icon {
                  width: 16px;
                  height: 16px;
                  margin-right: 8px;
                  color: #6366f1;  /* 紫色主题色（indigo-500） */
                  opacity: 0.8;
                  flex-shrink: 0;
                  transition: color 0.2s ease;
                  
                  &.app-icon {
                    color: #f59e0b; /* amber-500 - 工作空间用橙色 */
                    opacity: 0.9;
                  }
                  
                  &.app-icon-img {
                    width: 16px;
                    height: 16px;
                    object-fit: contain;
                    opacity: 0.9;
                  }
                  
                  &.package-icon {
                    color: #6366f1;
                    opacity: 0.8;
                  }
                  
                  &.package-icon-img {
                    width: 16px;
                    height: 16px;
                    object-fit: contain;
                    opacity: 0.9;
                  }
                  
                  &.table-icon {
                    color: #10b981; /* green-500 - 表格用绿色 */
                    opacity: 0.9;
                  }
                  
                  &.form-icon {
                    color: #3b82f6; /* blue-500 - 表单用蓝色 */
                    opacity: 0.9;
                  }
                  
                  &.form-icon-img {
                    width: 16px;
                    height: 16px;
                    object-fit: contain;
                    opacity: 0.9;
                  }
                  
                  &.chart-icon {
                    color: #f59e0b; /* amber-500 - 图表用橙色 */
                    opacity: 0.9;
                  }
                  
                  &.docs-icon {
                    color: #9b42f8; /* purple-500 - 文档用紫色 */
                    opacity: 0.9;
                  }
                  
                  &.docs-icon-img {
                    width: 16px;
                    height: 16px;
                    object-fit: contain;
                    opacity: 0.9;
                  }

                  &.board-icon {
                    color: #10b981; /* emerald - 讨论区 */
                    opacity: 0.9;
                  }

                  &.board-icon-img {
                    width: 16px;
                    height: 16px;
                    object-fit: contain;
                    opacity: 0.9;
                  }
                  
                  &.function-icon {
                    color: #6366f1; /* indigo-500 - 默认函数图标 */
                    opacity: 0.8;
                  }
                  
                  &.fx-icon {
                    font-size: 12px;
            font-weight: 600;
                    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
                    font-style: italic;
                    color: #6366f1;
                    opacity: 0.8;
                  }
                }
                
                .node-label {
                  font-size: 14px;
            color: var(--el-text-color-primary);
                  flex: 1;
                  overflow: hidden;
                  text-overflow: ellipsis;
                  white-space: nowrap;
                  
                  &.no-permission {
                    color: var(--el-text-color-disabled);
                    opacity: 0.6;
                  }
                }
                
                .no-permission-icon {
                  width: 16px;
                  height: 16px;
                  margin-left: 4px;
                  opacity: 0.7;
                  flex-shrink: 0;
                  transition: opacity 0.2s ease;
                  
                  &:hover {
                    opacity: 1;
                  }
                }
                
                .node-meta {
                  display: flex;
                  align-items: center;
                  gap: 8px;
                  flex-shrink: 0;
                  
                  .node-type-tag {
                    font-size: 10px;
                  }
                  
                  .template-tag {
                    font-size: 10px;
                  }
                  
                  .selected-permissions-hint {
                    display: flex;
                    align-items: center;
                    gap: 4px;
                    flex-wrap: wrap;
                    
                    .permission-hint-tag {
                      font-size: 10px;
                      padding: 2px 6px;
                      margin: 0;
                    }
                  }
                }
              }
              
              :deep(.el-tree-node__content) {
                height: 32px;
                padding: 0 8px;
                display: flex;
                align-items: center;

            &:hover {
                  background-color: var(--el-fill-color-light);
                }
              }
              
              :deep(.el-tree-node.is-current > .el-tree-node__content) {
                background-color: rgba(99, 102, 241, 0.15) !important;
                border-left: 2px solid #6366f1;
                
                .tree-node {
                  .node-label {
                    color: var(--el-text-color-primary);
                    font-weight: 500;
                  }
                  
                  .node-icon {
                    color: #6366f1;
                    opacity: 0.8;
                  }
                }
              }
              
              /* 确保子节点不受父节点选中状态影响 */
              :deep(.el-tree-node.is-current .el-tree-node__children .el-tree-node__content) {
                background-color: transparent;
                border-left: none;
              }
            }
          }
        }
      }

      .apply-main {
        min-width: 0; // 防止 grid 溢出

        .permission-scopes {
          .scope-header-main {
              display: flex;
              justify-content: space-between;
              align-items: center;
            margin-bottom: 16px;
            padding-bottom: 16px;
            border-bottom: 1px solid var(--el-border-color-lighter);

            .scope-title-main {
                display: flex;
                align-items: center;
                gap: 8px;
              flex-wrap: wrap;

              .scope-name-main {
                font-size: 18px;
                font-weight: 600;
                  color: var(--el-text-color-primary);
                }
              
              .selected-permissions-display {
                display: flex;
                align-items: center;
                gap: 6px;
                flex-wrap: wrap;
                
                .selected-permission-tag {
                  font-size: 12px;
                  padding: 4px 8px;
                }
              }
            }
          }

          .scope-path-main {
            margin-bottom: 24px;
            padding: 12px 16px;
              background: var(--el-fill-color-lighter);
            border-radius: 8px;
              border: 1px solid var(--el-border-color-lighter);

              code {
                font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
                font-size: 13px;
                color: var(--el-text-color-primary);
              }
          }

          // ⭐ 角色选择区域样式
          .role-selection-section {
            margin-bottom: 24px;
            padding: 24px;
            background: var(--el-fill-color-lighter);
            border-radius: 12px;
            border: 1px solid var(--el-border-color-lighter);
            width: 100%;
            max-width: 100%;
            box-sizing: border-box;
            display: block;
            overflow: hidden;
            position: relative;

            .role-selection-header {
              margin-bottom: 20px;

              .role-selection-title {
                display: flex;
                align-items: center;
                gap: 8px;
                margin: 0 0 12px 0;
                font-size: 16px;
                font-weight: 600;
                color: var(--el-text-color-primary);
              }

              .role-tip {
                margin-top: 12px;
              }
            }

            .role-list {
              width: 100%;
              display: block;
              box-sizing: border-box;

              .role-cards {
                display: flex;
                flex-direction: column;
                gap: 16px;
                width: 100%;
                margin: 0;
                padding: 0;
                box-sizing: border-box;

                .role-card {
                  padding: 20px;
                  border: 2px solid var(--el-border-color);
                  border-radius: 10px;
                  background: var(--el-bg-color);
                  cursor: pointer;
                  transition: all 0.3s ease;
                  position: relative;
                  display: block;
                  width: 100%;
                  box-sizing: border-box;
                  margin: 0;
                  overflow: hidden;

                  // 添加选中指示器
                  &::before {
                    content: '';
                    position: absolute;
                    left: 0;
                    top: 0;
                    bottom: 0;
                    width: 4px;
                    background: transparent;
                    transition: background 0.3s ease;
                  }

                  &:hover {
                    border-color: var(--el-color-primary-light-5);
                  }

                  &.is-selected {
                    border-color: var(--el-color-primary);
                    border-width: 2px;

                    &::before {
                      background: var(--el-color-primary);
                      width: 4px;
                    }
                  }

                  .role-card-header {
                    display: flex;
                    align-items: center;
                    justify-content: space-between;
                    margin-bottom: 12px;
                    flex-wrap: nowrap;
                    gap: 12px;

                    .role-name {
                      font-size: 16px;
                      font-weight: 600;
                      color: var(--el-text-color-primary);
                      flex: 1;
                      min-width: 0;
                      line-height: 1.4;
                    }
                  }

                  .role-description {
                    margin: 0 0 12px 0;
                    font-size: 14px;
                    color: var(--el-text-color-regular);
                    line-height: 1.6;
                    word-break: break-word;
                    min-height: 20px;
                  }

                  .role-permissions-preview {
                    margin-top: 12px;
                    display: flex;
                    flex-wrap: wrap;
                    gap: 6px;
                    align-items: center;
                    min-height: 24px;

                    .permission-tag {
                      margin: 0;
                    }
                  }
                }
              }

              .role-selected-actions {
                margin-top: 20px;
                padding-top: 16px;
                border-top: 1px solid var(--el-border-color-lighter);
                display: flex;
                gap: 12px;
                justify-content: flex-end;
              }
            }
          }

          .permission-list {
            width: 100%;
            max-width: 100%;
            box-sizing: border-box;
            display: block;
            overflow: visible;
            
            .permission-list-header {
              margin-bottom: 20px;
              
              .permission-list-title {
                margin: 0 0 12px 0;
                font-size: 16px;
                font-weight: 600;
                color: var(--el-text-color-primary);
              }
              
              .permission-tip {
                margin-top: 12px;
                
                :deep(.el-alert__content) {
                  .tip-content {
                    .tip-text {
                      margin: 4px 0;
                      font-size: 13px;
                      line-height: 1.6;
                      color: var(--el-text-color-regular);
                      
                      &:first-child {
                        margin-top: 0;
                      }
                      
                      strong {
                        color: var(--el-color-primary);
                      }
                    }
                  }
                }
              }
            }
            
            .permission-section {
              margin-bottom: 24px;
              
              &:last-child {
                margin-bottom: 0;
              }
              
              &.manage-permissions {
                padding: 16px;
                background: var(--el-fill-color-lighter);
                border-radius: 8px;
                border: 1px solid var(--el-border-color-lighter);
                
                .manage-permissions-header {
                  display: flex;
                  align-items: center;
                  gap: 8px;
                  margin-bottom: 12px;
                  
                  .el-icon {
                    color: var(--el-color-warning);
                    font-size: 16px;
                  }
                  
                  .manage-permissions-title {
                    font-size: 16px;
                    font-weight: 600;
                    color: var(--el-text-color-primary);
                  }
                  
                  .manage-tag {
                    margin-left: auto;
                  }
                }
                
                .manage-alert {
                  margin-bottom: 16px;
                  
                  :deep(.el-alert__content) {
                    .alert-content {
                      .alert-text {
                        margin: 4px 0;
                        font-size: 13px;
                        line-height: 1.6;
                        color: var(--el-text-color-regular);
                        
                        &:first-child {
                          margin-top: 0;
                        }
                        
                        strong {
                          color: var(--el-color-warning-dark-2);
                        }
                      }
                    }
                  }
                }
              }
            }

            .permission-checkbox-group {
              display: flex;
              flex-direction: column;
              gap: 8px;
              width: 100%;

              :deep(.el-checkbox) {
                margin: 0;
                height: auto;
                align-items: flex-start;
                width: 100%;
                max-width: 100%;
                
                .el-checkbox__input {
                  margin-top: 2px;
                  flex-shrink: 0;
                }
                
                .el-checkbox__label {
                  width: 100%;
                  max-width: 100%;
                  padding-left: 8px;
                  line-height: 1.5;
                  word-break: break-word;
                  overflow-wrap: break-word;
                }
              }

              :deep(.el-checkbox.is-checked) {
                .permission-checkbox {
                  border-color: var(--el-color-primary);
                  background-color: var(--el-color-primary-light-9);
                }
              }
              
              :deep(.el-checkbox.manage-checkbox.is-checked) {
                .permission-checkbox {
                  border-color: var(--el-color-warning);
                  background-color: var(--el-color-warning-light-9);
                }
              }

              .permission-checkbox {
                width: 100%;
                max-width: 100%;
                margin: 0;
                padding: 10px 12px;
                border: 1px solid var(--el-border-color-lighter);
                border-radius: 6px;
                transition: all 0.2s ease;
                background: var(--el-fill-color-lighter);
                min-height: auto;
                display: flex;
                flex-direction: column;
                justify-content: flex-start;
                box-sizing: border-box;

                &:hover {
                  border-color: var(--el-color-primary-light-7);
                  background-color: var(--el-fill-color);
                  transform: translateY(-1px);
                  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
                }
                
                // ⭐ 已有权限且选中：禁用状态，显示为已选中
                &.has-existing-selected {
                  // 禁用状态的样式
                  :deep(.el-checkbox__input.is-disabled) {
                    .el-checkbox__inner {
                      background-color: var(--el-color-success);
                      border-color: var(--el-color-success);
                      cursor: not-allowed;
                    }
                    
                    &.is-checked .el-checkbox__inner {
                      background-color: var(--el-color-success);
                      border-color: var(--el-color-success);
                    }
                    
                    .el-checkbox__label {
                      color: var(--el-text-color-primary);
                      cursor: not-allowed;
                      opacity: 0.9;
                    }
                  }
                  
                  // 禁用状态下不显示hover效果
                  &:hover {
                    border-color: var(--el-border-color-lighter);
                    background-color: var(--el-fill-color-lighter);
                    transform: none;
                    box-shadow: none;
                  }
                }
                
                // ⭐ 已有权限但未选中（理论上不应该出现，因为已有权限会自动选中）
                &.has-existing-unselected {
                  :deep(.el-checkbox__input.is-disabled) {
                    .el-checkbox__inner {
                      background-color: var(--el-fill-color);
                      border-color: var(--el-border-color);
                      cursor: not-allowed;
                    }
                    
                    .el-checkbox__label {
                      color: var(--el-text-color-regular);
                      cursor: not-allowed;
                      opacity: 0.6;
                    }
                  }
                  
                  &:hover {
                    border-color: var(--el-border-color-lighter);
                    background-color: var(--el-fill-color-lighter);
                    transform: none;
                    box-shadow: none;
                  }
                }

                .permission-option {
                  display: flex;
                  flex-direction: column;
                  align-items: flex-start;
                  gap: 4px;
                  width: 100%;
                  max-width: 100%;
                  min-width: 0;

                  .permission-header {
                    display: flex;
                    align-items: center;
                    gap: 8px;
                    width: 100%;
                    max-width: 100%;
                    min-width: 0;
                    flex-wrap: wrap;

                  .permission-name {
                      font-weight: 500;
                    color: var(--el-text-color-primary);
                      font-size: 14px;
                    line-height: 1.3;
                    word-break: break-word;
                      overflow-wrap: break-word;
                      flex: 1;
                      min-width: 0;
                    }

                    .permission-tags {
                      display: flex;
                      align-items: center;
                      gap: 6px;
                      flex-wrap: wrap;
                      flex-shrink: 0;
                      
                      .existing-tag,
                      .new-selected-tag,
                      .minimal-tag {
                        flex-shrink: 0;
                      }
                    }
                  }
                  
                  .permission-description {
                    margin: 0;
                    font-size: 12px;
                    color: var(--el-text-color-secondary);
                    line-height: 1.4;
                    word-break: break-word;
                    overflow-wrap: break-word;
                    width: 100%;
                  }
                  
                  .inheritance-badge {
                    display: inline-flex;
                    align-items: center;
                    gap: 4px;
                    margin-left: 8px;
                    padding: 2px 6px;
                    background: var(--el-fill-color-darker);
                    border-radius: 3px;
                    font-size: 11px;
                    color: var(--el-text-color-secondary);
                    
                    .inheritance-icon-small {
                      font-size: 12px;
                      flex-shrink: 0;
                    }
                  }

                  }
                }
              }
            }
          }

        .empty-state {
          display: flex;
          justify-content: center;
          align-items: center;
          min-height: 400px;
        }
      }

      .apply-sidebar-right {
        position: sticky;
        top: 24px;

        .form-card {
          border-radius: 12px;
          border: 1px solid var(--el-border-color-lighter);
          background: var(--el-bg-color);

          :deep(.el-card__header) {
            padding: 16px 20px;
            border-bottom: 1px solid var(--el-border-color-lighter);
            background: var(--el-fill-color-lighter);
            border-radius: 12px 12px 0 0;

            h3 {
              margin: 0;
              font-size: 16px;
              font-weight: 600;
              color: var(--el-text-color-primary);
            }
          }

          :deep(.el-card__body) {
            padding: 20px;
          }
        }

        .apply-form {
          width: 100%;
          max-width: 100%;
          overflow: hidden;
          box-sizing: border-box;

          .form-item-tip {
            margin-top: 8px;
          }

          :deep(.el-form-item) {
            width: 100%;
            max-width: 100%;
            overflow: hidden;
            box-sizing: border-box;
          }

          :deep(.el-form-item__content) {
            width: 100%;
            max-width: 100%;
            overflow: hidden;
            box-sizing: border-box;
          }

          :deep(.el-form-item__label) {
            font-weight: 500;
            color: var(--el-text-color-primary);
          }

          :deep(.el-textarea__inner) {
            border-radius: 8px;
            border-color: var(--el-border-color);
            background: var(--el-fill-color-lighter);
            transition: all 0.2s ease;

            &:focus {
              border-color: var(--el-color-primary);
              background: var(--el-bg-color);
            }
          }


          :deep(.el-button) {
            border-radius: 8px;
            padding: 10px 20px;
          }

          .grant-target-type-radio {
            width: 100%;
            margin-bottom: 16px;

            :deep(.el-radio) {
              margin-right: 24px;
            }
          }

          .grant-target-display {
            margin-top: 12px;
            padding: 12px;
            background: var(--el-fill-color-lighter);
            border-radius: 8px;
            border: 1px solid var(--el-border-color-lighter);
            width: 100%;
            max-width: 100%;
            overflow: hidden;
            box-sizing: border-box;

            .current-user-info {
              display: flex;
              align-items: flex-start;
              gap: 12px;
              width: 100%;
              max-width: 100%;
              overflow: hidden;

              .el-avatar {
                flex-shrink: 0;
                width: 36px !important;
                height: 36px !important;
                border: 1px solid var(--el-border-color);
              }

              .user-details {
                flex: 1;
                min-width: 0;
                max-width: 100%;
                overflow: hidden;

                .user-name {
                  font-size: 14px;
                  font-weight: 600;
                  color: var(--el-text-color-primary);
                  line-height: 1.4;
                  margin-bottom: 4px;
                  white-space: nowrap;
                  overflow: hidden;
                  text-overflow: ellipsis;
                  width: 100%;
                }

                .user-email {
                  font-size: 12px;
                  color: var(--el-text-color-secondary);
                  line-height: 1.4;
                  overflow: hidden;
                  text-overflow: ellipsis;
                  white-space: nowrap;
                  margin-bottom: 4px;
                  width: 100%;
                }

                .user-org-info,
                .user-leader-info {
                  display: flex;
                  align-items: center;
                  gap: 4px;
                  font-size: 12px;
                  color: var(--el-text-color-regular);
                  margin-top: 4px;
                  line-height: 1.4;
                  width: 100%;
                  max-width: 100%;
                  overflow: hidden;

                  .el-icon {
                    font-size: 12px;
                    color: var(--el-text-color-secondary);
                    flex-shrink: 0;
                  }

                  span {
                    flex: 1;
                    min-width: 0;
                    overflow: hidden;
                    text-overflow: ellipsis;
                    white-space: nowrap;
                  }
                }
              }
            }
          }

          .selected-user-details {
            margin-top: 12px;
            width: 100%;
            max-width: 100%;
            overflow: hidden;

            .selected-user-card {
              position: relative;
              padding: 12px;
              padding-right: 36px;
              background: var(--el-bg-color);
              border: 1px solid var(--el-color-primary-light-7);
              border-radius: 8px;
              box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
              width: 100%;
              max-width: 100%;
              box-sizing: border-box;
              overflow: hidden;

              .user-content {
                display: flex;
                gap: 12px;
                width: 100%;
                max-width: 100%;
                min-width: 0;
                overflow: hidden;

                .user-avatar {
                  flex-shrink: 0;
                  width: 36px !important;
                  height: 36px !important;
                  border: 1px solid var(--el-color-primary-light-5);
                }

                .user-info {
                  flex: 1;
                  min-width: 0;
                  max-width: 100%;
                  overflow: hidden;

                  .user-name {
                    font-size: 14px;
                    font-weight: 600;
                    color: var(--el-text-color-primary);
                    margin-bottom: 4px;
                    white-space: nowrap;
                    overflow: hidden;
                    text-overflow: ellipsis;
                    width: 100%;
                    max-width: 100%;
                  }

                  .user-meta {
                    display: flex;
                    flex-wrap: wrap;
                    gap: 8px;
                    font-size: 12px;
                    color: var(--el-text-color-secondary);
                    margin-bottom: 4px;
                    line-height: 1.4;
                    width: 100%;
                    max-width: 100%;
                    overflow: hidden;

                    .user-nickname {
                      color: var(--el-text-color-regular);
                      white-space: nowrap;
                      overflow: hidden;
                      text-overflow: ellipsis;
                      max-width: 120px;
                    }

                    .user-email {
                      color: var(--el-text-color-secondary);
                      white-space: nowrap;
                      overflow: hidden;
                      text-overflow: ellipsis;
                      max-width: 150px;
                    }
                  }

                  .user-org-info,
                  .user-leader-info {
                    display: flex;
                    align-items: center;
                    gap: 4px;
                    font-size: 12px;
                    color: var(--el-text-color-regular);
                    margin-top: 4px;
                    line-height: 1.4;
                    width: 100%;
                    max-width: 100%;
                    overflow: hidden;

                    .el-icon {
                      font-size: 13px;
                      color: var(--el-text-color-secondary);
                      flex-shrink: 0;
                    }

                    span {
                      flex: 1;
                      min-width: 0;
                      overflow: hidden;
                      text-overflow: ellipsis;
                      white-space: nowrap;
                    }
                  }
                }
              }

              .remove-btn {
                position: absolute;
                top: 8px;
                right: 8px;
                width: 24px;
                height: 24px;
                padding: 0;
                flex-shrink: 0;
                z-index: 1;
              }
            }
          }

          .selected-department-details {
            margin-top: 12px;
            width: 100%;
            max-width: 100%;
            overflow: hidden;

            .selected-department-card {
              position: relative;
              padding: 12px;
              padding-right: 36px;
              background: var(--el-bg-color);
              border: 1px solid var(--el-color-primary-light-7);
              border-radius: 8px;
              box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
              width: 100%;
              max-width: 100%;
              box-sizing: border-box;
              overflow: hidden;

              .department-content {
                display: flex;
                gap: 12px;
                width: 100%;
                max-width: 100%;
                min-width: 0;
                overflow: hidden;

                .department-icon {
                  flex-shrink: 0;
                  width: 36px;
                  height: 36px;
                  object-fit: contain;
                }

                .department-info {
                  flex: 1;
                  min-width: 0;
                  max-width: 100%;
                  overflow: hidden;

                  .department-name {
                    font-size: 14px;
                    font-weight: 600;
                    color: var(--el-text-color-primary);
                    margin-bottom: 4px;
                    white-space: nowrap;
                    overflow: hidden;
                    text-overflow: ellipsis;
                    width: 100%;
                    max-width: 100%;
                  }

                  .department-meta {
                    display: flex;
                    flex-wrap: wrap;
                    gap: 8px;
                    font-size: 12px;
                    color: var(--el-text-color-secondary);
                    line-height: 1.4;
                    width: 100%;
                    max-width: 100%;
                    overflow: hidden;

                    .department-path {
                      font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
                      color: var(--el-text-color-secondary);
                      white-space: nowrap;
                      overflow: hidden;
                      text-overflow: ellipsis;
                      max-width: 200px;
                    }

                    .department-full-name {
                      color: var(--el-text-color-regular);
                      white-space: nowrap;
                      overflow: hidden;
                      text-overflow: ellipsis;
                      max-width: 150px;
                    }

                    .department-managers {
                      display: inline-flex;
                      align-items: center;
                      gap: 4px;
                      color: var(--el-text-color-secondary);
                      white-space: nowrap;
                      overflow: hidden;
                      text-overflow: ellipsis;
                      max-width: 180px;

                      .el-icon {
                        font-size: 12px;
                        flex-shrink: 0;
                      }
                    }
                  }
                }
              }

              .remove-btn {
                position: absolute;
                top: 8px;
                right: 8px;
                width: 24px;
                height: 24px;
                padding: 0;
                flex-shrink: 0;
                z-index: 1;
              }
            }
          }

          .grant-target-input {
            margin-top: 12px;
            width: 100%;
            max-width: 100%;
            overflow: hidden;
            box-sizing: border-box;

            .disabled-overlay {
              opacity: 0.6;
            }

            // 优化用户选择器的显示效果
            :deep(.user-search-input) {
              .user-search-input-wrapper {
                background-color: var(--el-fill-color-lighter);
                border: 1px solid var(--el-border-color);
                border-radius: 6px;
                padding: 6px 10px;
                min-height: 38px;
                transition: all 0.2s ease;

                &:hover {
                  border-color: var(--el-border-color-hover);
                  background-color: var(--el-bg-color);
                }

                &:focus-within {
                  border-color: var(--el-color-primary);
                  background-color: var(--el-bg-color);
                  box-shadow: 0 0 0 2px var(--el-color-primary-light-9);
                }
              }

              .user-cell-inline {
                background-color: var(--el-color-primary-light-9);
                border: 1px solid var(--el-color-primary-light-7);
                padding: 5px 10px;
                border-radius: 5px;
                height: 28px;
                margin-right: 4px;

                .user-avatar {
                  width: 20px !important;
                  height: 20px !important;
                  flex-shrink: 0;
                }

                .user-name {
                  color: var(--el-color-primary);
                  font-weight: 500;
                  font-size: 13px;
                  line-height: 18px;
                  white-space: nowrap;
                }

                .remove-icon {
                  color: var(--el-text-color-secondary);
                  width: 16px;
                  height: 16px;
                  margin-left: 6px;
                  flex-shrink: 0;

                  &:hover {
                    color: var(--el-color-primary);
                  }
                }
              }

              .input-wrapper {
                flex: 1;
                min-width: 100px;

                .user-search-input-field {
                  :deep(.el-input__inner) {
                    font-size: 14px;
                    height: 26px;
                    line-height: 26px;
                  }
                }
              }
            }

            // 优化部门选择器的显示效果
            :deep(.el-select) {
              .el-input__wrapper {
                background-color: var(--el-fill-color-lighter);
                border: 1px solid var(--el-border-color);
                border-radius: 6px;
                transition: all 0.2s ease;

                &:hover {
                  border-color: var(--el-border-color-hover);
                  background-color: var(--el-bg-color);
                }
              }

              &.is-focus .el-input__wrapper {
                border-color: var(--el-color-primary);
                box-shadow: 0 0 0 2px var(--el-color-primary-light-9);
              }
            }
          }
        }
      }
    }
  }
}
</style>
