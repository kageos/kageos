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
  <div class="organization-management">
    <div class="organization-container">
      <!-- 左侧：组织架构树 -->
      <div class="left-sidebar">
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

      <!-- 右侧：用户列表 -->
      <div class="right-content">
        <el-card shadow="hover" class="users-card">
          <template #header>
            <div class="card-header">
              <h3>{{ selectedDepartment ? `${selectedDepartment.name} - 用户列表` : '请选择部门' }}</h3>
              <div class="header-actions">
                <el-button :icon="Refresh" @click="refreshUsers" size="small">刷新</el-button>
              </div>
            </div>
          </template>

          <!-- 用户列表 -->
          <el-table
            v-loading="usersLoading"
            :data="userList"
            style="width: 100%"
            stripe
          >
            <el-table-column label="用户名" width="200">
              <template #default="{ row }">
                <UserDisplay
                  :username="(row as any).username"
                  mode="card"
                  layout="horizontal"
                  size="small"
                />
              </template>
            </el-table-column>
            <el-table-column prop="nickname" label="昵称" width="150" />
            <el-table-column label="性别" width="80" align="center">
              <template #default="{ row }">
                <span>{{ getGenderText((row as any).gender) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="email" label="邮箱" min-width="200" />
            <el-table-column label="部门" min-width="200">
              <template #default="{ row }">
                <DepartmentDisplay
                  v-if="(row as any).department_full_path"
                  :full-code-path="(row as any).department_full_path"
                  :display-name="(row as any).department_full_name_path || (row as any).department_name"
                  mode="card"
                  layout="horizontal"
                  size="small"
                />
                <el-tag v-else type="info" size="small">未分配</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="Leader" width="200">
              <template #default="{ row }">
                <UserDisplay
                  v-if="(row as any).leader_username"
                  :username="(row as any).leader_username"
                  mode="card"
                  layout="horizontal"
                  size="small"
                />
                <el-tag v-else type="info" size="small">未分配</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="150" fixed="right">
              <template #default="{ row }">
                <el-button
                  link
                  type="primary"
                  size="small"
                  @click="handleEditUser(row)"
                >
                  编辑
                </el-button>
              </template>
            </el-table-column>
          </el-table>

          <!-- 空状态 -->
          <el-empty
            v-if="!usersLoading && userList.length === 0"
            description="暂无用户"
            :image-size="100"
          />
        </el-card>
      </div>
    </div>

    <!-- 部门编辑对话框 -->
    <el-dialog
      v-model="departmentDialogVisible"
      :title="departmentDialogTitle"
      width="600px"
    >
      <el-form
        ref="departmentFormRef"
        :model="departmentForm"
        :rules="departmentFormRules"
        label-width="100px"
      >
        <el-form-item label="部门名称" prop="name">
          <el-input v-model="departmentForm.name" placeholder="请输入部门名称" />
        </el-form-item>
        <el-form-item label="部门编码" prop="code">
          <el-input
            v-model="departmentForm.code"
            placeholder="请输入部门编码"
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
            :rows="3"
            placeholder="请输入部门描述"
          />
        </el-form-item>
        <el-form-item v-if="departmentForm.id" label="负责人">
          <el-input
            v-model="departmentForm.managers"
            placeholder="多个用户名逗号分隔，如：zhangsan,lisi"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="departmentDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="departmentSubmitting" @click="handleSubmitDepartment">
          确定
        </el-button>
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
import { ref, reactive, computed, onMounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus, Refresh } from '@element-plus/icons-vue'
import DepartmentTreePanel from '@/components/DepartmentTreePanel.vue'
import DepartmentDisplay from '@/architecture/presentation/widgets/DepartmentDisplay.vue'
import {
  getDepartmentTree,
  createDepartment,
  updateDepartment,
  deleteDepartment,
  getUsersByDepartment,
  type Department
} from '@/api/department'
import { searchUsersFuzzy, type UserInfo } from '@/api/user'
import UserDisplay from '@/architecture/presentation/widgets/UserDisplay.vue'
import UserEditDialog from '@/views/User/components/UserEditDialog.vue'
import { useAuthStore } from '@/stores/auth'

// ==================== 状态管理 ====================

// 部门树相关
const departmentLoading = ref(false)
const departmentTree = ref<Department[]>([])
const selectedDepartmentId = ref<number | null>(null)
const selectedDepartment = ref<Department | null>(null)

// 用户列表相关
const usersLoading = ref(false)
const userList = ref<UserInfo[]>([])

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
  description: ''
})

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

// ==================== 方法 ====================

// 加载部门树
async function loadDepartmentTree() {
  departmentLoading.value = true
  try {
    const res = await getDepartmentTree()
    departmentTree.value = res.departments || []
    
    // 默认选中用户自己的组织
    await nextTick()
    selectUserDepartment()
  } catch (error: any) {
    ElMessage.error(error.message || '获取部门树失败')
  } finally {
    departmentLoading.value = false
  }
}

// 根据用户的 department_full_path 在树中找到对应的部门并选中
function selectUserDepartment() {
  const authStore = useAuthStore()
  const userDepartmentPath = authStore.user?.department_full_path
  
  if (!userDepartmentPath || !departmentTree.value.length) {
    // 如果没有部门信息或树为空，默认选中第一个部门
    if (departmentTree.value.length > 0) {
      handleDepartmentNodeClick(departmentTree.value[0])
    }
    return
  }
  
  // 在树中查找匹配的部门
  const foundDepartment = findDepartmentByPath(departmentTree.value, userDepartmentPath)
  if (foundDepartment) {
    handleDepartmentNodeClick(foundDepartment)
  } else {
    // 如果找不到，默认选中第一个部门
    if (departmentTree.value.length > 0) {
      handleDepartmentNodeClick(departmentTree.value[0])
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


// 处理部门节点点击
function handleDepartmentNodeClick(node: Department) {
  selectedDepartmentId.value = node.id
  selectedDepartment.value = node
  // 自动加载该部门的用户列表
  loadDepartmentUsers(node)
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
          description: departmentForm.description
        })
        ElMessage.success('创建部门成功')
      }
      departmentDialogVisible.value = false
      loadDepartmentTree()
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

// 获取性别文本
function getGenderText(gender?: string): string {
  const genderMap: Record<string, string> = {
    'male': '男',
    'female': '女',
    'other': '其他',
    '': '未设置'
  }
  return genderMap[gender || ''] || '未设置'
}

// ==================== 生命周期 ====================

onMounted(() => {
  loadDepartmentTree()
})
</script>

<style scoped lang="scss">
.organization-management {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.organization-container {
  display: flex;
  height: 100%;
  gap: 16px;
  overflow: hidden;
}

.left-sidebar {
  width: 400px;
  min-width: 400px;
  height: 100%;
  border-right: 1px solid var(--el-border-color);
  overflow: hidden;
}

.right-content {
  flex: 1;
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-width: 0; /* 确保 flex 子元素可以收缩 */
}

.users-card {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  width: 100%;

  :deep(.el-card__body) {
    flex: 1;
    overflow: auto;
    display: flex;
    flex-direction: column;
    padding: 20px;
  }

  :deep(.el-table) {
    width: 100%;
  }

  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;

    h3 {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }

    .header-actions {
      display: flex;
      gap: 8px;
    }
  }
}
</style>

