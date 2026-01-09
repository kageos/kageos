<!--
  DepartmentsWidget - 多组织架构组件
  功能：
  - 输入场景（edit/search）：多组织架构选择器
  - 输出场景（response/table-cell/detail）：显示多个组织架构信息
  - 值使用逗号分隔的字符串存储（如 "/org/nanjing,/org/beijing"），便于存储到数据库
-->
<template>
  <div class="departments-widget">
    <!-- 响应模式：显示多个组织架构 -->
    <div v-if="mode === 'response'" class="departments-response">
      <div v-if="displayDepartments.length > 0" class="departments-list departments-list-horizontal">
        <DepartmentDisplay
          v-for="(dept, index) in displayDepartments"
          :key="dept.full_code_path || index"
          :department-info="dept"
          :full-code-path="dept.full_code_path"
          :display-name="dept.full_name_path || dept.name"
          mode="card"
          layout="horizontal"
          size="small"
          class="department-item"
        />
      </div>
      <span v-else class="empty-text">-</span>
    </div>
    
    <!-- 表格单元格模式：只显示图标和名称，hover 显示详细信息 -->
    <div v-else-if="mode === 'table-cell'" class="departments-table-cell">
      <div v-if="displayDepartments.length > 0" class="departments-icons-list">
        <el-popover
          v-for="(dept, index) in displayDepartments"
          :key="dept.full_code_path || index"
          placement="top"
          :width="520"
          trigger="hover"
          popper-class="departments-popover"
        >
          <template #reference>
            <div class="department-icon-item">
              <img src="/组织架构.svg" alt="组织架构" class="department-icon-small" />
              <span class="department-name-small">{{ dept.full_name_path || dept.name }}</span>
            </div>
          </template>
          <DepartmentDetailCard 
            :department-info="dept"
            :department-tree="departmentTree"
            :current-path="dept.full_code_path"
          />
        </el-popover>
      </div>
      <span v-else class="empty-text">-</span>
    </div>
    
    <!-- 详情模式：显示多个组织架构 -->
    <div v-else-if="mode === 'detail'" class="departments-detail">
      <div v-if="displayDepartments.length > 0" class="departments-list departments-list-horizontal">
        <DepartmentDisplay
          v-for="(dept, index) in displayDepartments"
          :key="dept.full_code_path || index"
          :department-info="dept"
          :full-code-path="dept.full_code_path"
          :display-name="dept.full_name_path || dept.name"
          mode="card"
          layout="horizontal"
          size="small"
          class="department-item"
        />
      </div>
      <span v-else class="empty-text">-</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { ElPopover } from 'element-plus'
import type { Department } from '@/api/department'
import { getDepartmentTree } from '@/api/department'
import DepartmentDisplay from './DepartmentDisplay.vue'
import DepartmentDetailCard from './DepartmentDetailCard.vue'
import type { WidgetComponentProps } from './types'

const COMPONENT_NAME = 'DepartmentsWidget'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  }),
  mode: 'response'
})

// 部门树（用于查找部门信息）
const departmentTree = ref<Department[]>([])

// 显示组织架构列表（用于响应模式）
const displayDepartments = computed(() => {
  // 优先从 meta 中获取
  if (props.value?.meta?.departmentInfoList && Array.isArray(props.value.meta.departmentInfoList)) {
    return props.value.meta.departmentInfoList
  }
  
  // 从 raw 值解析组织架构路径
  if (!props.value?.raw) {
    return []
  }
  
  const paths = String(props.value.raw).split(',').map(p => p.trim()).filter(p => p)
  if (paths.length === 0) {
    return []
  }
  
  // 从部门树中查找对应的部门信息
  const findDepartment = (depts: Department[], path: string): Department | null => {
    for (const dept of depts) {
      if (dept.full_code_path === path) {
        return dept
      }
      if (dept.children && dept.children.length > 0) {
        const found = findDepartment(dept.children, path)
        if (found) return found
      }
    }
    return null
  }
  
  const departments: Department[] = []
  for (const path of paths) {
    const dept = findDepartment(departmentTree.value, path)
    if (dept) {
      departments.push(dept)
    } else {
      // 如果找不到，创建一个临时对象用于显示
      departments.push({
        id: 0,
        name: path.split('/').pop() || path,
        code: '',
        parent_id: null,
        full_code_path: path,
        full_name_path: path,
        managers: '',
        description: '',
        status: 'active',
        sort: 0,
        created_at: '',
        updated_at: ''
      })
    }
  }
  
  return departments
})

// 加载部门树
const loadDepartmentTree = async () => {
  if (departmentTree.value.length === 0) {
    try {
      const treeRes = await getDepartmentTree()
      departmentTree.value = treeRes.departments
    } catch (error) {
      console.error('[DepartmentsWidget] 加载部门树失败', error)
    }
  }
}

// 组件挂载时加载部门树
onMounted(() => {
  if (props.value?.raw) {
    loadDepartmentTree()
  }
})

// 监听 value 变化，如果路径变化则重新加载
watch(() => props.value?.raw, () => {
  if (props.value?.raw && departmentTree.value.length === 0) {
    loadDepartmentTree()
  }
}, { immediate: false })
</script>

<style scoped>
.departments-widget {
  width: 100%;
}

/* 响应模式 */
.departments-response {
  width: 100%;
}

.departments-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.departments-list-horizontal {
  flex-direction: row;
}

.department-item {
  display: inline-flex;
}

.empty-text {
  color: var(--el-text-color-placeholder);
  font-size: 14px;
}

/* 表格单元格模式 */
.departments-table-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.departments-icons-list {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.department-icon-item {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.department-icon-item:hover {
  background-color: var(--el-fill-color-light);
}

.department-icon-small {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  opacity: 0.8;
}

.department-name-small {
  font-size: 12px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 详情模式 */
.departments-detail {
  width: 100%;
}
</style>

<style>
/* Popover 全局样式 */
.departments-popover {
  padding: 0 !important;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
}
</style>
