<!--
  组织架构和用户管理 - 入口页面
  
  需求：
  - 获取部门树
  - 查看部门下的用户
  - 管理组织架构（创建、编辑、删除部门）
  - 编辑用户的组织架构和 Leader
  - 查看待分配用户
  
  设计思路：
  - 使用树形结构展示部门层级
  - 支持点击部门查看该部门下的用户
  - 支持编辑用户的组织架构和 Leader
  - 支持管理组织架构
-->

<template>
  <div class="organization-management-page">
    <section class="page-header">
      <div class="page-header-copy">
        <h1>组织架构管理</h1>
        <p>查看部门结构、负责人和成员归属，常用操作尽量放在同一屏完成。</p>
      </div>
    </section>

    <section class="organization-workspace">
      <aside class="navigation-panel">
        <div class="tree-panel-shell">
          <DepartmentTreePanel
            :tree-data="departmentTree"
            :loading="departmentLoading"
            :current-node-id="selectedDepartmentId"
            @node-click="handleDepartmentNodeClick"
            @create-department="handleCreateDepartment"
            @view-users="handleViewDepartmentUsers"
            @edit="handleEditDepartment"
            @delete="handleDeleteDepartment"
            @refresh="loadDepartmentTree"
          />
        </div>
      </aside>

      <section class="detail-panel">
        <template v-if="selectedDepartment">
          <section class="detail-card">
            <div class="detail-header">
              <div class="detail-main">
                <div class="detail-title-row">
                  <h2>{{ selectedDepartment.name }}</h2>
                  <div class="detail-tags">
                    <el-tag effect="dark" round size="small">{{ selectedDepartment.code }}</el-tag>
                    <el-tag
                      v-if="selectedDepartment.is_system_default"
                      class="neutral-tag"
                      round
                      size="small"
                    >
                      系统默认组织
                    </el-tag>
                  </div>
                </div>

                <p class="detail-description">
                  {{ selectedDepartment.description || '这个部门还没有补充介绍，可以在右上角编辑后补全职责和边界。' }}
                </p>
              </div>

              <div class="detail-actions">
                <el-button type="primary" :icon="Plus" @click="handleCreateDepartment(selectedDepartment)">
                  新增子部门
                </el-button>
                <el-button :icon="Edit" @click="handleEditDepartment(selectedDepartment)">编辑部门</el-button>
                <el-button :icon="Refresh" @click="refreshUsers">刷新成员</el-button>
              </div>
            </div>

            <div class="detail-summary">
              <div class="summary-item">
                <span class="summary-label">组织路径</span>
                <span class="summary-value mono">
                  {{ selectedDepartment.full_name_path || selectedDepartment.name }}
                </span>
              </div>
              <div class="summary-item">
                <span class="summary-label">直属子部门</span>
                <span class="summary-value">{{ selectedDepartment.children?.length || 0 }}</span>
              </div>
              <div class="summary-item">
                <span class="summary-label">当前成员</span>
                <span class="summary-value">{{ userList.length }}</span>
              </div>
              <div class="summary-item">
                <span class="summary-label">负责人</span>
                <span v-if="managersLoading" class="summary-value subtle-text">负责人信息加载中...</span>
                <span v-else-if="selectedDepartmentManagerText" class="summary-value">
                  {{ selectedDepartmentManagerText }}
                </span>
                <span v-else class="summary-value subtle-text">当前部门尚未配置负责人</span>
              </div>
            </div>
          </section>

          <section class="members-panel">
            <div class="members-header">
              <div>
                <h3>成员列表</h3>
                <p class="members-summary">{{ filteredUserList.length }} / {{ userList.length }} 名成员</p>
              </div>

              <div class="members-toolbar">
                <el-input
                  v-model="userSearch"
                  clearable
                  :prefix-icon="Search"
                  placeholder="搜索用户名、昵称、邮箱"
                />
                <el-button :icon="Refresh" @click="refreshUsers">刷新</el-button>
              </div>
            </div>

            <div class="table-shell">
              <el-table
                v-loading="usersLoading"
                :data="filteredUserList"
                class="members-table"
                stripe
                row-key="username"
              >
                <el-table-column label="成员" width="250">
                  <template #default="{ row }">
                    <div class="member-stack">
                      <UserDisplay
                        :username="(row as any).username"
                        mode="card"
                        layout="horizontal"
                        size="small"
                      />
                      <span class="member-meta">
                        {{ (row as any).nickname || `@${(row as any).username}` }}
                      </span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="邮箱" min-width="220">
                  <template #default="{ row }">
                    <div class="member-stack">
                      <span class="member-value">{{ (row as any).email || '未设置邮箱' }}</span>
                      <span class="member-meta">
                        {{ (row as any).email_verified ? '邮箱已验证' : '邮箱未验证' }}
                      </span>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="部门" min-width="220">
                  <template #default="{ row }">
                    <div class="member-stack">
                      <DepartmentDisplay
                        v-if="(row as any).department_full_path"
                        :full-code-path="(row as any).department_full_path"
                        :display-name="(row as any).department_full_name_path || (row as any).department_name"
                        :department-tree="departmentTree"
                        mode="card"
                        layout="horizontal"
                        size="small"
                      />
                      <el-tag v-else type="info" size="small" class="soft-tag">未分配</el-tag>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="Leader" width="220">
                  <template #default="{ row }">
                    <div class="member-stack">
                      <UserDisplay
                        v-if="(row as any).leader_username"
                        :username="(row as any).leader_username"
                        mode="card"
                        layout="horizontal"
                        size="small"
                      />
                      <el-tag v-else type="info" size="small" class="soft-tag">未分配</el-tag>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="120" fixed="right">
                  <template #default="{ row }">
                    <el-button
                      text
                      type="primary"
                      size="small"
                      class="member-action"
                      @click="handleEditUser(row)"
                    >
                      编辑
                    </el-button>
                  </template>
                </el-table-column>

                <template #empty>
                  <div class="table-empty">
                    <el-empty
                      :description="userSearch ? '没有匹配到成员' : '当前部门还没有成员'"
                      :image-size="84"
                    />
                  </div>
                </template>
              </el-table>
            </div>
          </section>
        </template>

        <div v-else class="detail-empty">
          <el-icon class="detail-empty-icon"><OfficeBuilding /></el-icon>
          <h3>先选一个部门</h3>
          <p>左侧选择组织节点后，这里会显示部门信息、负责人和成员列表。</p>
        </div>
      </section>
    </section>

    <!-- 部门编辑对话框 -->
    <el-dialog
      v-model="departmentDialogVisible"
      class="department-dialog"
      width="600px"
      :close-on-click-modal="false"
    >
      <template #header>
        <div class="dialog-header">
          <span class="dialog-kicker">Department Editor</span>
          <h3>{{ departmentDialogTitle }}</h3>
          <p>{{ departmentDialogDescription }}</p>
        </div>
      </template>

      <div class="department-dialog-body">
        <div class="dialog-intro-card">
          <article class="dialog-intro-item">
            <span class="intro-label">父部门</span>
            <strong>{{ departmentFormParentName }}</strong>
          </article>
          <article class="dialog-intro-item">
            <span class="intro-label">负责人</span>
            <strong>{{ departmentFormManagerCount }} 人</strong>
          </article>
          <article class="dialog-intro-item">
            <span class="intro-label">编码状态</span>
            <strong>{{ departmentForm.id ? '已锁定' : '创建后锁定' }}</strong>
          </article>
        </div>

      <el-form
        ref="departmentFormRef"
        :model="departmentForm"
        :rules="departmentFormRules"
        label-width="100px"
        class="department-form"
      >
        <el-form-item label="部门名称" prop="name">
          <el-input v-model="departmentForm.name" placeholder="例如：客户成功部、华东运营组" />
        </el-form-item>
        <el-form-item label="部门编码" prop="code">
          <el-input
            v-model="departmentForm.code"
            placeholder="请输入稳定的部门编码"
            :disabled="!!departmentForm.id"
          />
        </el-form-item>
        <el-form-item label="父部门">
          <el-select
            v-model="departmentForm.parent_id"
            placeholder="请选择父部门（不选则为根部门）"
            clearable
            filterable
            style="width: 100%"
          >
            <el-option
              v-for="dept in flatDepartmentList"
              :key="dept.id"
              :label="dept.name"
              :value="dept.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="部门描述">
          <el-input
            v-model="departmentForm.description"
            type="textarea"
            :rows="4"
            placeholder="补充职责边界、协作对象或这个部门的主要工作内容"
          />
        </el-form-item>
        <el-form-item label="负责人">
          <UsersWidget
            :value="managersFieldValue"
            :field="managersField"
            mode="edit"
            field-path="managers"
            @update:modelValue="handleManagersChange"
          />
        </el-form-item>
      </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer-actions">
          <el-button @click="departmentDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="departmentSubmitting" @click="handleSubmitDepartment">
            {{ departmentForm.id ? '保存更新' : '创建部门' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 用户编辑对话框 -->
    <UserEditDialog
      v-model="userEditDialogVisible"
      :user-info="currentEditUser"
      :department-tree="departmentTree"
      @success="handleEditUserSuccess"
    />

  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import {
  Edit,
  OfficeBuilding,
  Plus,
  Refresh,
  Search
} from '@element-plus/icons-vue'
import DepartmentTreePanel from '@/shared/components/DepartmentTreePanel.vue'
import DepartmentDisplay from '@/shared/components/DepartmentDisplay.vue'
import {
  getDepartmentTree,
  createDepartment,
  updateDepartment,
  deleteDepartment,
  getUsersByDepartment,
  type Department
} from '@/architecture/infrastructure/api/department'
import type { UserInfo } from '@/architecture/domain/types'
import { useUserInfoStore } from '@/architecture/infrastructure/stores/userInfo'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import UserEditDialog from '@/features/user/components/UserEditDialog.vue'
import { useAuthStore } from '@/architecture/infrastructure/stores/auth'
import UsersWidget from '@/shared/components/UsersWidget.vue'
import { WidgetType } from '@/architecture/runtime/constants/widget'
import type { FieldValue } from '@/architecture/runtime/types/field'
import { createStringFieldValue, createWidgetFieldConfig, extractStringFieldRaw } from '@/utils/widgetFieldHelpers'

// ==================== 状态管理 ====================

// 部门树相关
const departmentLoading = ref(false)
const departmentTree = ref<Department[]>([])
const selectedDepartmentId = ref<number | null>(null)
const selectedDepartment = ref<Department | null>(null)
const authStore = useAuthStore()

// 用户列表相关
const usersLoading = ref(false)
const userList = ref<UserInfo[]>([])
const userSearch = ref('')

// 负责人用户列表
const managerUsers = ref<UserInfo[]>([])
const managersLoading = ref(false)

// 对话框相关
const departmentDialogVisible = ref(false)
const departmentDialogTitle = ref('新增部门')
const departmentSubmitting = ref(false)
const departmentFormRef = ref<FormInstance>()
const departmentForm = reactive<{
  id?: number
  name: string
  code: string
  parent_id: number | null
  description: string
  managers?: string
}>({
  name: '',
  code: '',
  parent_id: null,
  description: '',
  managers: ''
})

const managersField = createWidgetFieldConfig({
  code: 'managers',
  name: '负责人',
  widgetType: WidgetType.USERS
})

const managersFieldValue = computed(() =>
  createStringFieldValue(managersField, departmentForm.managers, { emptyRaw: '' })
)

const handleManagersChange = (value: FieldValue) => {
  departmentForm.managers = extractStringFieldRaw(value)
}

const departmentFormRules: FormRules = {
  name: [{ required: true, message: '请输入部门名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入部门编码', trigger: 'blur' }]
}

// 用户编辑对话框
const userEditDialogVisible = ref(false)
const currentEditUser = ref<UserInfo | null>(null)

// ==================== 计算属性 ====================

// 扁平化部门列表（用于下拉选择）
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

const filteredUserList = computed(() => {
  const keyword = userSearch.value.trim().toLowerCase()
  if (!keyword) return userList.value

  return userList.value.filter((user) => {
    return [
      user.username,
      user.nickname,
      user.email,
      user.department_name,
      user.department_full_name_path,
      user.leader_username,
      user.leader_display_name
    ]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(keyword))
  })
})

const departmentFormParentName = computed(() => {
  if (!departmentForm.parent_id) return '根部门'
  return findDepartmentById(departmentTree.value, departmentForm.parent_id)?.name || '未匹配到父部门'
})

const departmentFormManagerCount = computed(() => {
  return departmentForm.managers
    ?.split(',')
    .map((item) => item.trim())
    .filter(Boolean)
    .length || 0
})

const departmentDialogDescription = computed(() => {
  if (departmentForm.id) {
    return '调整部门说明、负责人和层级归属，保存后组织树和成员关系会立即同步。'
  }
  return '创建新的组织节点，并在生成时明确它的编码、归属和负责人配置。'
})

const selectedDepartmentManagerText = computed(() => {
  return managerUsers.value.map((user) => user.username).join('、')
})

// ==================== 方法 ====================

// 加载部门树
async function loadDepartmentTree() {
  departmentLoading.value = true
  try {
    const res = await getDepartmentTree()
    departmentTree.value = res.departments || []

    const previousSelectedId = selectedDepartmentId.value

    // 优先保持当前选中，再退回到用户自己的组织
    await nextTick()
    if (previousSelectedId) {
      const previousDepartment = findDepartmentById(departmentTree.value, previousSelectedId)
      if (previousDepartment) {
        handleDepartmentNodeClick(previousDepartment)
        return
      }
    }
    selectUserDepartment()
  } catch (error: any) {
    ElMessage.error(error.message || '获取部门树失败')
  } finally {
    departmentLoading.value = false
  }
}

// 根据用户的 department_full_path 在树中找到对应的部门并选中
function selectUserDepartment() {
  const userDepartmentPath = authStore.user?.department_full_path
  
  if (!userDepartmentPath || !departmentTree.value.length) {
    // 如果没有部门信息或树为空，默认选中第一个部门
    const firstDepartment = departmentTree.value[0]
    if (firstDepartment) {
      handleDepartmentNodeClick(firstDepartment)
    }
    return
  }
  
  // 在树中查找匹配的部门
  const foundDepartment = findDepartmentByPath(departmentTree.value, userDepartmentPath)
  if (foundDepartment) {
    handleDepartmentNodeClick(foundDepartment)
  } else {
    // 如果找不到，默认选中第一个部门
    const firstDepartment = departmentTree.value[0]
    if (firstDepartment) {
      handleDepartmentNodeClick(firstDepartment)
    }
  }
}

// 在树中根据 full_code_path 查找部门
function findDepartmentByPath(tree: Department[], targetPath: string): Department | null {
  for (const dept of tree) {
    if (dept.full_code_path === targetPath) {
      return dept
    }
    if (dept.children && dept.children.length > 0) {
      const found = findDepartmentByPath(dept.children, targetPath)
      if (found) {
        return found
      }
    }
  }
  return null
}

// 在树中根据 id 查找部门
function findDepartmentById(tree: Department[], id: number): Department | null {
  for (const dept of tree) {
    if (dept.id === id) {
      return dept
    }
    if (dept.children && dept.children.length > 0) {
      const found = findDepartmentById(dept.children, id)
      if (found) {
        return found
      }
    }
  }
  return null
}


// 处理部门节点点击
function handleDepartmentNodeClick(node: Department) {
  selectedDepartmentId.value = node.id
  selectedDepartment.value = node
  // 自动加载该部门的用户列表
  loadDepartmentUsers(node)
  // 加载负责人信息
  loadManagerUsers(node)
}

// 加载负责人用户信息（使用用户 SDK，带缓存）
async function loadManagerUsers(department: Department) {
  if (!department.managers || department.managers.trim() === '') {
    managerUsers.value = []
    return
  }
  
  const usernames = department.managers.split(',').map(u => u.trim()).filter(Boolean)
  if (usernames.length === 0) {
    managerUsers.value = []
    return
  }
  
  managersLoading.value = true
  try {
    const userInfoStore = useUserInfoStore()
    const users: UserInfo[] = []
    
    // 并行加载所有负责人信息（使用 SDK，自动处理缓存）
    await Promise.all(
      usernames.map(async (username) => {
        try {
          const user = await userInfoStore.getUserInfo(username)
          if (user) {
            users.push(user)
          }
        } catch (error) {
          console.error(`[Organization] 加载负责人 ${username} 信息失败:`, error)
        }
      })
    )
    
    managerUsers.value = users
  } catch (error: any) {
    console.error('加载负责人信息失败:', error)
    managerUsers.value = []
  } finally {
    managersLoading.value = false
  }
}

// 新增部门
function handleCreateDepartment(parentNode?: Department) {
  departmentDialogTitle.value = '新增部门'
  Object.assign(departmentForm, {
    id: undefined,
    name: '',
    code: '',
    parent_id: parentNode ? parentNode.id : null,
    description: '',
    managers: ''
  })
  departmentDialogVisible.value = true
}

// 编辑部门
function handleEditDepartment(dept: Department) {
  departmentDialogTitle.value = '编辑部门'
  Object.assign(departmentForm, {
    id: dept.id,
    name: dept.name,
    code: dept.code,
    parent_id: dept.parent_id,
    description: dept.description,
    managers: dept.managers
  })
  departmentDialogVisible.value = true
}

// 提交部门表单
async function handleSubmitDepartment() {
  if (!departmentFormRef.value) return

  await departmentFormRef.value.validate(async (valid) => {
    if (!valid) return

    departmentSubmitting.value = true
    try {
      if (departmentForm.id) {
        // 更新
        await updateDepartment(departmentForm.id, {
          name: departmentForm.name,
          description: departmentForm.description,
          managers: departmentForm.managers
        })
        ElMessage.success('更新部门成功')
      } else {
        // 创建
        await createDepartment({
          name: departmentForm.name,
          code: departmentForm.code,
          parent_id: departmentForm.parent_id ?? 0, // null 转换为 0 传给后端（后端会将 0 转换为 NULL）
          description: departmentForm.description,
          managers: departmentForm.managers
        })
        ElMessage.success('创建部门成功')
      }
      departmentDialogVisible.value = false
      await loadDepartmentTree()
      // 如果更新的是当前选中的部门，重新加载负责人信息
      if (departmentForm.id && selectedDepartment.value && selectedDepartment.value.id === departmentForm.id) {
        // 从树中重新获取更新后的部门信息
        const updatedDept = findDepartmentById(departmentTree.value, departmentForm.id)
        if (updatedDept) {
          selectedDepartment.value = updatedDept
          loadManagerUsers(updatedDept)
        }
      }
    } catch (error: any) {
      ElMessage.error(error.message || '操作失败')
    } finally {
      departmentSubmitting.value = false
    }
  })
}

// 删除部门
async function handleDeleteDepartment(dept: Department) {
  // ⭐ 检查是否为系统默认组织
  if (dept.is_system_default) {
    ElMessage.warning('系统默认组织不可删除')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除部门 "${dept.name}" 吗？`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await deleteDepartment(dept.id)
    ElMessage.success('删除部门成功')
    loadDepartmentTree()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除部门失败')
    }
  }
}

// 加载部门用户列表
async function loadDepartmentUsers(dept: Department) {
  usersLoading.value = true
  try {
    const res = await getUsersByDepartment(dept.full_code_path)
    userList.value = res.users || []
  } catch (error: any) {
    ElMessage.error(error.message || '获取部门用户失败')
    userList.value = []
  } finally {
    usersLoading.value = false
  }
}

// 查看部门用户（右键菜单）
async function handleViewDepartmentUsers(dept: Department) {
  selectedDepartmentId.value = dept.id
  selectedDepartment.value = dept
  await loadDepartmentUsers(dept)
  // 加载负责人信息
  loadManagerUsers(dept)
}

// 刷新用户列表
function refreshUsers() {
  if (selectedDepartment.value) {
    loadDepartmentUsers(selectedDepartment.value)
  }
}

// 编辑用户
function handleEditUser(user: UserInfo) {
  currentEditUser.value = user
  userEditDialogVisible.value = true
}

// 用户编辑成功回调
function handleEditUserSuccess() {
  // 如果当前选中了部门，刷新该部门的用户列表
  if (selectedDepartment.value) {
    loadDepartmentUsers(selectedDepartment.value)
  }
}

// ==================== 生命周期 ====================

onMounted(() => {
  loadDepartmentTree()
})

watch(selectedDepartmentId, () => {
  userSearch.value = ''
})
</script>

<style scoped lang="scss">
.organization-management-page {
  --org-ink: var(--text-primary);
  --org-muted: var(--text-secondary);
  --org-line: color-mix(in srgb, var(--border-base) 82%, var(--color-primary) 18%);
  --org-border-soft: color-mix(in srgb, var(--border-base) 70%, transparent);
  --org-card: var(--bg-primary);
  --org-shadow: var(--box-shadow-sm);
  --org-accent: var(--color-primary);
  --org-accent-soft: color-mix(in srgb, var(--color-primary) 12%, transparent);
  height: 100%;
  overflow: auto;
  padding: 24px;
  background: var(--bg-page);
  color: var(--org-ink);
}

.page-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
}

.page-header-copy h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 700;
}

.page-header-copy p {
  margin: 6px 0 0;
  color: var(--org-muted);
  line-height: 1.6;
}

.organization-workspace {
  min-height: calc(100% - 80px);
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  gap: 16px;
}

.navigation-panel,
.detail-panel {
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.tree-panel-shell {
  flex: 1;
  min-height: 0;
  padding: 14px;
  border-radius: 14px;
  background: var(--org-card);
  border: 1px solid var(--border-base);
  box-shadow: var(--org-shadow);
}

.tree-panel-shell :deep(.department-tree-panel) {
  height: 100%;
  border-radius: 12px;
  overflow: hidden;
  background: transparent;
}

.detail-card,
.members-panel,
.detail-empty {
  padding: 18px;
  border-radius: 14px;
  background: var(--org-card);
  border: 1px solid var(--border-base);
  box-shadow: var(--org-shadow);
}

.detail-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.detail-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.detail-main {
  min-width: 0;
}

.detail-title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.detail-title-row h2 {
  margin: 0;
  font-size: 24px;
  line-height: 1.2;
}

.detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.neutral-tag {
  color: var(--org-muted);
  border-color: var(--org-line);
  background: var(--bg-secondary);
}

.detail-description {
  margin: 8px 0 0;
  color: var(--org-muted);
  line-height: 1.7;
}

.detail-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
}

.detail-summary {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 16px;
  padding-top: 12px;
  border-top: 1px solid var(--org-border-soft);
}

.summary-item {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.summary-label {
  font-size: 12px;
  color: var(--org-muted);
}

.summary-value {
  font-size: 14px;
  font-weight: 500;
  color: var(--org-ink);
  word-break: break-word;
}

.summary-value.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  font-weight: 500;
}

.subtle-text {
  font-size: 13px;
  color: var(--org-muted);
}

.members-panel {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.members-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.members-header h3 {
  margin: 0 0 6px;
  font-size: 18px;
  line-height: 1.2;
}

.members-summary {
  margin: 0;
  color: var(--org-muted);
  font-size: 13px;
}

.members-toolbar {
  width: min(360px, 100%);
  display: flex;
  align-items: center;
  gap: 12px;
}

.members-toolbar :deep(.el-input) {
  flex: 1;
}

.table-shell {
  flex: 1;
  min-height: 0;
}

.table-shell :deep(.el-table) {
  width: 100%;
  border-radius: 12px;
  --el-table-header-bg-color: color-mix(in srgb, var(--color-primary) 7%, var(--bg-primary) 93%);
  --el-table-row-hover-bg-color: color-mix(in srgb, var(--color-primary) 5%, var(--bg-primary) 95%);
  --el-table-border-color: var(--org-line);
  --el-table-tr-bg-color: transparent;
}

.table-shell :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.table-shell :deep(.el-table th.el-table__cell) {
  padding: 12px 0;
  font-size: 12px;
  color: var(--org-muted);
}

.table-shell :deep(.el-table td.el-table__cell) {
  padding: 14px 0;
}

.member-stack {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.member-meta {
  font-size: 12px;
  color: var(--org-muted);
}

.member-value {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  color: var(--org-ink);
}

.member-value {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
}

.member-action {
  border-radius: 10px;
  padding-inline: 10px;
}

.soft-tag {
  width: fit-content;
}

.table-empty {
  padding: 32px 0;
}

.dialog-header {
  display: flex;
  flex-direction: column;
  gap: 6px;

  h3 {
    margin: 0;
    font-size: 20px;
    color: var(--org-ink);
  }

  p {
    margin: 0;
    line-height: 1.7;
    color: var(--org-muted);
    font-size: 13px;
  }
}

.dialog-kicker {
  display: inline-flex;
  align-items: center;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  font-size: 11px;
  font-weight: 700;
  color: color-mix(in srgb, var(--color-primary) 68%, var(--text-secondary) 32%);
}

.department-dialog-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.dialog-intro-card {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  padding: 14px;
  border-radius: 12px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-base);
}

.dialog-intro-item {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;

  strong {
    font-size: 14px;
    color: var(--org-ink);
    word-break: break-word;
  }
}

.intro-label {
  font-size: 12px;
  color: var(--org-muted);
}

.department-form {
  :deep(.el-form-item) {
    margin-bottom: 20px;
  }

  :deep(.el-form-item__label) {
    font-weight: 600;
    color: var(--text-regular);
  }

  :deep(.el-input__wrapper),
  :deep(.el-select__wrapper),
  :deep(.el-textarea__inner) {
    border-radius: 14px;
    box-shadow: none;
    background: var(--bg-primary);
    border: 1px solid var(--border-base);
  }

  :deep(.el-textarea__inner) {
    min-height: 112px;
  }
}

.dialog-footer-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.department-dialog) {
  border-radius: 28px;
  overflow: hidden;
  background: var(--bg-primary);
}

:deep(.department-dialog .el-dialog__header) {
  padding: 24px 24px 12px;
}

:deep(.department-dialog .el-dialog__body) {
  padding: 0 24px 8px;
}

:deep(.department-dialog .el-dialog__footer) {
  padding: 12px 24px 24px;
}

.detail-empty {
  flex: 1;
  min-height: 360px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.detail-empty-icon {
  margin-bottom: 16px;
  font-size: 34px;
  color: color-mix(in srgb, var(--color-primary) 52%, var(--text-secondary) 48%);
}

.detail-empty h3 {
  margin: 0 0 10px;
  font-size: 24px;
}

.detail-empty p {
  max-width: 380px;
  margin: 0;
  line-height: 1.7;
  color: var(--org-muted);
}

@media (max-width: 1380px) {
  .organization-workspace,
  .members-header,
  .detail-header {
    grid-template-columns: 1fr;
    flex-direction: column;
  }

  .organization-workspace {
    display: flex;
  }

  .navigation-panel {
    min-height: 420px;
  }

  .members-toolbar {
    width: 100%;
  }

  .dialog-intro-card {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 920px) {
  .organization-management-page {
    padding: 16px;
  }

  .page-header,
  .tree-panel-shell,
  .detail-card,
  .members-panel,
  .detail-empty {
    border-radius: 12px;
  }

  .page-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .members-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .detail-summary {
    grid-template-columns: 1fr;
  }

  .detail-title-row h2 {
    font-size: 22px;
  }
}
</style>
