<template>
  <div class="permission-manage-list" v-loading="loading">
    <!-- 筛选条件 -->
    <div class="filter-section">
      <el-form :inline="true" :model="filterForm" class="filter-form">
        <el-form-item label="范围筛选">
          <el-select v-model="filterForm.scope" placeholder="选择范围" style="width: 150px" @change="handleFilterChange">
            <el-option label="全部" value="all" />
            <el-option label="仅当前节点" value="current" />
            <el-option label="仅父目录" value="parent" />
          </el-select>
        </el-form-item>
        <el-form-item label="用户搜索">
          <el-input
            v-model="filterForm.userSearch"
            placeholder="搜索用户"
            clearable
            style="width: 200px"
            @input="handleFilterChange"
          />
        </el-form-item>
        <el-form-item label="组织架构搜索">
          <el-input
            v-model="filterForm.departmentSearch"
            placeholder="搜索组织架构"
            clearable
            style="width: 200px"
            @input="handleFilterChange"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadPermissions">刷新</el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 权限列表 -->
    <div class="permission-list">
      <el-empty v-if="!loading && assignments.length === 0" description="暂无权限分配" />
      <el-table
        v-else
        :data="assignments"
        stripe
        style="width: 100%"
      >
        <el-table-column label="权限主体" min-width="200">
          <template #default="{ row }">
            <template v-if="row.subject_type === 'user'">
              <UsersWidget
                :value="getSubjectUsersValue(row.subject)"
                :field="subjectUsersField"
                mode="response"
                field-path="subject"
              />
            </template>
            <template v-else>
              <DepartmentsWidget
                :value="getSubjectDepartmentsValue(row.subject)"
                :field="subjectDepartmentsField"
                mode="response"
                field-path="subject"
              />
            </template>
          </template>
        </el-table-column>
        
        <el-table-column label="角色" min-width="150">
          <template #default="{ row }">
            <el-tag>{{ row.role_name || row.role_code }}</el-tag>
          </template>
        </el-table-column>
        
        <el-table-column label="来源" width="120">
          <template #default="{ row }">
            <el-tag :type="getSourceTagType(row.resource_path)" size="small">
              {{ getSourceLabel(row.resource_path) }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column label="资源" min-width="250" show-overflow-tooltip>
          <template #default="{ row }">
            <div>
              <div v-if="row.resource_name" class="resource-name">{{ row.resource_name }}</div>
              <div class="resource-path">{{ row.resource_path }}</div>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column label="有效期" min-width="200">
          <template #default="{ row }">
            <span>{{ formatTimeRange(row.start_time, row.end_time) }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="创建者" width="150">
          <template #default="{ row }">
            <UserDisplay
              v-if="row.created_by"
              :username="row.created_by"
              mode="card"
              size="small"
              layout="horizontal"
            />
            <span v-else>-</span>
          </template>
        </el-table-column>
        
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            <span>{{ formatDateTime(row.created_at) }}</span>
          </template>
        </el-table-column>
        
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button
              type="danger"
              link
              size="small"
              @click="handleDelete(row)"
              :loading="deletingId === row.id"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getResourcePermissions, type ResourcePermissionAssignment } from '@/api/permission'
import { removeRoleFromUser, removeRoleFromDepartment } from '@/api/role'
import { useRoute } from 'vue-router'
import { extractWorkspacePath } from '@/utils/route'
import UserDisplay from '@/architecture/presentation/widgets/UserDisplay.vue'
import UsersWidget from '@/architecture/presentation/widgets/UsersWidget.vue'
import DepartmentsWidget from '@/architecture/presentation/widgets/DepartmentsWidget.vue'
import { WidgetType } from '@/core/constants/widget'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'

interface Props {
  resourcePath?: string  // 资源路径（可选，如果提供则使用该路径，否则从路由获取）
  user?: string  // 租户用户（可选，如果提供则使用，否则从 resourcePath 解析）
  app?: string  // 应用代码（可选，如果提供则使用，否则从 resourcePath 解析）
  autoLoad?: boolean  // 是否自动加载
}

const props = withDefaults(defineProps<Props>(), {
  autoLoad: true
})

const route = useRoute()

// 状态
const loading = ref(false)
const allAssignments = ref<ResourcePermissionAssignment[]>([]) // 所有权限（未筛选）
const deletingId = ref<number | null>(null)

// 筛选条件
const filterForm = ref({
  scope: 'all', // 范围筛选：all-全部, current-仅当前节点, parent-仅父目录
  userSearch: '', // 用户搜索
  departmentSearch: '' // 组织架构搜索
})

// 当前资源路径
const currentResourcePath = computed(() => getResourcePath.value)

// 获取资源路径
const getResourcePath = computed(() => {
  if (props.resourcePath) {
    return props.resourcePath
  }
  // 从路由获取
  const fullPath = extractWorkspacePath(route.path)
  if (!fullPath) {
    return ''
  }
  return '/' + fullPath.split('/').filter(Boolean).join('/')
})

// 获取 user 和 app
const getUserAndApp = computed(() => {
  // 优先使用 props 传入的 user 和 app
  if (props.user && props.app) {
    return {
      user: props.user,
      app: props.app
    }
  }
  
  // 否则从 resourcePath 解析
  const pathParts = getResourcePath.value.split('/').filter(Boolean)
  if (pathParts.length < 2) {
    return { user: '', app: '' }
  }
  return {
    user: pathParts[0],
    app: pathParts[1]
  }
})

// 加载权限列表
const loadPermissions = async () => {
  const resourcePath = getResourcePath.value
  if (!resourcePath) {
    ElMessage.warning('无法获取资源路径')
    return
  }

  const { user, app } = getUserAndApp.value
  if (!user || !app) {
    ElMessage.warning('无法获取用户和应用信息')
    return
  }

  loading.value = true
  try {
    const response = await getResourcePermissions({
      user,
      app,
      resource_path: resourcePath
    })
    allAssignments.value = response.assignments || []
  } catch (error: any) {
    ElMessage.error('加载权限列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 获取所有父路径
const getAllParentPaths = (path: string): string[] => {
  if (!path || path === '/') return []
  const parts = path.split('/').filter(Boolean)
  if (parts.length <= 1) return []
  
  const parentPaths: string[] = []
  for (let i = parts.length - 1; i > 0; i--) {
    parentPaths.push('/' + parts.slice(0, i).join('/'))
  }
  return parentPaths
}

// 判断权限来源（当前节点还是父目录）
const getSourceLabel = (resourcePath: string): string => {
  if (!resourcePath || !currentResourcePath.value) {
    return '未知'
  }
  if (resourcePath === currentResourcePath.value) {
    return '当前节点'
  }
  const parentPaths = getAllParentPaths(currentResourcePath.value)
  if (parentPaths.includes(resourcePath)) {
    return '父目录'
  }
  return '未知'
}

// 获取来源标签类型
const getSourceTagType = (resourcePath: string): string => {
  if (!resourcePath || !currentResourcePath.value) {
    return 'info'
  }
  if (resourcePath === currentResourcePath.value) {
    return 'primary'
  }
  return 'info'
}

// 应用筛选
const assignments = computed(() => {
  let result = [...allAssignments.value]
  
  // 如果没有当前资源路径，返回空数组
  if (!currentResourcePath.value) {
    return []
  }
  
  // 范围筛选
  if (filterForm.value.scope === 'current') {
    // 仅当前节点
    result = result.filter(item => item.resource_path === currentResourcePath.value)
  } else if (filterForm.value.scope === 'parent') {
    // 仅父目录
    const parentPaths = getAllParentPaths(currentResourcePath.value)
    result = result.filter(item => parentPaths.includes(item.resource_path))
  }
  // scope === 'all' 时不过滤
  
  // 用户搜索（只筛选用户类型的权限）
  if (filterForm.value.userSearch) {
    const searchText = filterForm.value.userSearch.toLowerCase()
    result = result.filter(item => {
      if (item.subject_type === 'user') {
        return item.subject.toLowerCase().includes(searchText) ||
               (item.subject_name && item.subject_name.toLowerCase().includes(searchText))
      }
      // 如果不是用户类型，不显示（因为用户搜索框有值，只显示用户）
      return false
    })
  }
  
  // 组织架构搜索（只筛选组织架构类型的权限）
  if (filterForm.value.departmentSearch) {
    const searchText = filterForm.value.departmentSearch.toLowerCase()
    result = result.filter(item => {
      if (item.subject_type === 'department') {
        return item.subject.toLowerCase().includes(searchText) ||
               (item.subject_name && item.subject_name.toLowerCase().includes(searchText))
      }
      // 如果不是组织架构类型，不显示（因为组织架构搜索框有值，只显示组织架构）
      return false
    })
  }
  
  return result
})

// 筛选条件变化处理
const handleFilterChange = () => {
  // 筛选通过 computed 自动应用，这里可以添加其他逻辑
}

// 删除权限
const handleDelete = async (assignment: ResourcePermissionAssignment) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除该权限分配吗？\n权限主体：${assignment.subject_name || assignment.subject}\n角色：${assignment.role_name}`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    const { user, app } = getUserAndApp.value
    if (!user || !app) {
      ElMessage.error('无法获取用户和应用信息')
      return
    }

    deletingId.value = assignment.id

    // 需要从角色信息中获取 resource_type，这里先使用一个临时方案
    // 从 resource_path 推断 resource_type（不准确，但可以工作）
    // TODO: 后端应该返回 resource_type 信息
    let resourceType = 'directory' // 默认值
    const pathParts = assignment.resource_path.split('/').filter(Boolean)
    if (pathParts.length >= 3) {
      // 假设路径格式为 /user/app/...，第3部分开始是资源路径
      // 这里简单判断：如果路径很深，可能是 function，否则是 directory
      resourceType = pathParts.length > 3 ? 'table' : 'directory'
    }

    if (assignment.subject_type === 'user') {
      await removeRoleFromUser({
        user,
        app,
        username: assignment.subject,
        role_code: assignment.role_code,
        resource_type: resourceType,
        resource_path: assignment.resource_path
      })
    } else {
      await removeRoleFromDepartment({
        user,
        app,
        department_path: assignment.subject,
        role_code: assignment.role_code,
        resource_type: resourceType,
        resource_path: assignment.resource_path
      })
    }

    ElMessage.success('删除成功')
    await loadPermissions()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败: ' + (error.message || '未知错误'))
    }
  } finally {
    deletingId.value = null
  }
}

// 格式化时间范围
const formatTimeRange = (startTime: string, endTime?: string) => {
  if (!endTime) {
    return '永久'
  }
  const start = new Date(startTime).toLocaleString('zh-CN')
  const end = new Date(endTime).toLocaleString('zh-CN')
  return `${start} - ${end}`
}

// 格式化日期时间
const formatDateTime = (dateTime: string) => {
  return new Date(dateTime).toLocaleString('zh-CN')
}

// 用户字段配置（用于 UsersWidget）
const subjectUsersField: FieldConfig = {
  type: WidgetType.USERS,
  name: 'subject',
  label: '权限主体',
  data: {
    type: 'string'
  }
}

// 部门字段配置（用于 DepartmentsWidget）
const subjectDepartmentsField: FieldConfig = {
  type: WidgetType.DEPARTMENTS,
  name: 'subject',
  label: '权限主体',
  data: {
    type: 'string'
  }
}

// 获取用户字段值
const getSubjectUsersValue = (subject: string): FieldValue => {
  return {
    raw: subject,
    display: subject,
    meta: {}
  }
}

// 获取部门字段值
const getSubjectDepartmentsValue = (subject: string): FieldValue => {
  return {
    raw: subject,
    display: subject,
    meta: {}
  }
}

// 监听 autoLoad 和 resourcePath 变化，自动加载
watch(
  () => [props.autoLoad, props.resourcePath],
  ([autoLoad, resourcePath]) => {
    if (autoLoad && resourcePath) {
      loadPermissions()
    }
  },
  { immediate: true }
)

// 暴露方法供父组件调用
defineExpose({
  loadPermissions
})
</script>

<style scoped lang="scss">
.permission-manage-list {
  padding: 20px;

  .filter-section {
    margin-bottom: 20px;

    .filter-form {
      margin: 0;
    }
  }

  .permission-list {
    min-height: 200px;
  }

  .resource-name {
    font-weight: 500;
    color: var(--el-text-color-primary);
    margin-bottom: 4px;
  }

  .resource-path {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}
</style>
