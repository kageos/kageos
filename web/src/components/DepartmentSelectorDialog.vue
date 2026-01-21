<template>
  <el-dialog
    v-model="visible"
    title=""
    :show-close="false"
    :close-on-click-modal="true"
    :close-on-press-escape="true"
    width="600px"
    top="10vh"
    class="department-selector-dialog"
    append-to-body
    @close="handleClose"
  >
    <div class="department-selector-modal">
      <!-- 头部 -->
      <div class="department-selector-header">
        <div class="header-content">
          <el-icon class="header-icon"><OfficeBuilding /></el-icon>
          <h3 class="header-title">选择组织架构</h3>
        </div>
        <el-button
          text
          type="primary"
          @click="handleClose"
          class="close-btn"
        >
          <el-icon size="18"><Close /></el-icon>
        </el-button>
      </div>

      <!-- 搜索框 -->
      <div class="department-search-section">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索部门名称或路径..."
          size="large"
          class="department-search-input"
          @input="handleSearchInput"
          clearable
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <!-- 部门列表 -->
      <div class="department-list-section" v-loading="loading">
        <div class="department-list">
          <div
            v-for="dept in filteredDepartments"
            :key="dept.full_code_path"
            class="department-item"
            :class="{ 'selected': selectedDepartment?.full_code_path === dept.full_code_path }"
            @click="handleSelectDepartment(dept)"
          >
            <!-- 部门图标 -->
            <div class="department-icon-wrapper">
              <img src="/组织架构.svg" alt="部门" class="department-icon" />
            </div>

            <!-- 部门信息 -->
            <div class="department-info">
              <div class="department-name">{{ dept.name }}</div>
              <div class="department-meta">
                <span class="department-path">{{ dept.full_code_path }}</span>
                <span v-if="dept.full_name_path && dept.full_name_path !== dept.name" class="department-full-name">
                  {{ dept.full_name_path }}
                </span>
                <span v-if="dept.managers" class="department-managers">
                  <el-icon><UserFilled /></el-icon>
                  负责人: {{ dept.managers }}
                </span>
              </div>
            </div>

            <!-- 选择按钮 -->
            <div class="department-action">
              <el-button
                type="primary"
                size="small"
                :class="{ 'is-selected': selectedDepartment?.full_code_path === dept.full_code_path }"
                @click.stop="handleSelectDepartment(dept)"
              >
                {{ selectedDepartment?.full_code_path === dept.full_code_path ? '已选择' : '选择' }}
              </el-button>
            </div>
          </div>
          
          <div v-if="filteredDepartments.length === 0 && !loading" class="department-empty">
            <el-icon class="empty-icon"><OfficeBuilding /></el-icon>
            <div class="empty-text">暂无部门</div>
            <div class="empty-desc">{{ searchKeyword ? '请尝试其他搜索关键词' : '暂无组织架构数据' }}</div>
          </div>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { Close, Search, OfficeBuilding, UserFilled } from '@element-plus/icons-vue'
import { getDepartmentTree } from '@/api/department'
import type { Department } from '@/api/department'

// Props
interface Props {
  modelValue: boolean
  selectedDepartment?: Department | null
  departmentTree?: Department[]
}

const props = withDefaults(defineProps<Props>(), {
  selectedDepartment: null,
  departmentTree: () => []
})

// Emits
interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'select', department: Department): void
}

const emit = defineEmits<Emits>()

// 本地状态
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const searchKeyword = ref('')
const departmentTree = ref<Department[]>([])
const loading = ref(false)

// 扁平化部门列表（用于搜索和显示）
const flattenDepartments = (depts: Department[]): Department[] => {
  const result: Department[] = []
  const traverse = (list: Department[]) => {
    for (const dept of list) {
      result.push(dept)
      if (dept.children && dept.children.length > 0) {
        traverse(dept.children)
      }
    }
  }
  traverse(depts)
  return result
}

// 过滤后的部门列表
const filteredDepartments = computed(() => {
  const allDepartments = flattenDepartments(departmentTree.value)
  
  if (!searchKeyword.value || searchKeyword.value.trim().length === 0) {
    return allDepartments
  }
  
  const keyword = searchKeyword.value.trim().toLowerCase()
  return allDepartments.filter(dept => {
    return (
      dept.name.toLowerCase().includes(keyword) ||
      dept.full_code_path.toLowerCase().includes(keyword) ||
      (dept.full_name_path && dept.full_name_path.toLowerCase().includes(keyword)) ||
      (dept.code && dept.code.toLowerCase().includes(keyword))
    )
  })
})

// 加载部门树
const loadDepartmentTree = async () => {
  if (props.departmentTree && props.departmentTree.length > 0) {
    departmentTree.value = props.departmentTree
    return
  }
  
  loading.value = true
  try {
    const response = await getDepartmentTree()
    departmentTree.value = response.departments || []
  } catch (error) {
    console.error('加载部门树失败:', error)
    departmentTree.value = []
  } finally {
    loading.value = false
  }
}

// 处理搜索输入
const handleSearchInput = (value: string) => {
  searchKeyword.value = value
  // 不需要防抖，因为过滤是实时的
}

// 处理部门选择
const handleSelectDepartment = (department: Department) => {
  emit('select', department)
  handleClose()
}

// 关闭弹窗
const handleClose = () => {
  visible.value = false
  
  // 重置状态
  searchKeyword.value = ''
}

// 监听弹窗打开，加载部门树
watch(visible, (newVal) => {
  if (newVal) {
    // 清空搜索关键词
    searchKeyword.value = ''
    // 加载部门树
    loadDepartmentTree()
    // 延迟聚焦，确保DOM已渲染
    setTimeout(() => {
      const input = document.querySelector('.department-search-input input') as HTMLInputElement
      if (input) {
        input.focus()
      }
    }, 200)
  }
})
</script>

<style lang="scss" scoped>
:deep(.department-selector-dialog) {
  .el-dialog {
    border-radius: 20px;
    overflow: hidden;
    backdrop-filter: blur(20px);
    background: rgba(255, 255, 255, 0.95);
    border: 1px solid rgba(255, 255, 255, 0.2);
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
    animation: departmentSelectorFadeIn 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  }
  
  .el-dialog__header {
    padding: 0;
    margin: 0;
  }
  
  .el-dialog__body {
    padding: 0;
  }
  
  @media (prefers-color-scheme: dark) {
    .el-dialog {
      background: rgba(30, 30, 30, 0.95);
      border: 1px solid rgba(255, 255, 255, 0.1);
    }
  }
}

@keyframes departmentSelectorFadeIn {
  from {
    opacity: 0;
    transform: scale(0.9) translateY(-20px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.department-selector-modal {
  .department-selector-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 24px 24px 16px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    background: var(--el-bg-color);
    
    .header-content {
      display: flex;
      align-items: center;
      gap: 12px;
      
      .header-icon {
        font-size: 24px;
        color: var(--el-color-primary);
        background: var(--el-color-primary-light-9);
        padding: 8px;
        border-radius: 12px;
        opacity: 0.8;
      }
      
      .header-title {
        margin: 0;
        font-size: 20px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }
    }
    
    .close-btn {
      padding: 8px;
      border-radius: 12px;
      transition: all 0.2s;
      
      &:hover {
        background: var(--el-color-danger-light-9);
        transform: scale(1.1);
      }
    }
  }

  .department-search-section {
    padding: 24px;
    background: var(--el-bg-color);
    
    .department-search-input {
      :deep(.el-input__wrapper) {
        border-radius: 16px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
        border: 2px solid transparent;
        transition: all 0.3s;
        
        &:hover {
          box-shadow: 0 6px 16px rgba(0, 0, 0, 0.15);
        }
        
        &.is-focus {
          border-color: var(--el-color-primary);
          box-shadow: 0 6px 20px rgba(var(--el-color-primary-rgb), 0.3);
        }
      }
      
      .search-icon {
        color: var(--el-color-primary);
        font-size: 18px;
      }
    }
  }

  .department-list-section {
    max-height: 500px;
    overflow-y: auto;
    padding: 0 24px 24px;
    
    .department-list {
      .department-item {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 16px;
        margin-bottom: 8px;
        cursor: pointer;
        border-radius: 12px;
        background: var(--el-bg-color);
        border: 2px solid var(--el-border-color-lighter);
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
        
        &:hover {
          background: var(--el-fill-color-light);
          border-color: var(--el-color-primary-light-5);
          transform: translateY(-2px);
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
        }
        
        &.selected {
          background: var(--el-color-primary-light-9);
          border-color: var(--el-color-primary);
          box-shadow: 0 4px 16px rgba(var(--el-color-primary-rgb), 0.2);
        }
      }
      
      .department-icon-wrapper {
        flex-shrink: 0;
        margin-right: 16px;
        width: 40px;
        height: 40px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--el-fill-color-lighter);
        border-radius: 10px;
        border: 1px solid var(--el-border-color-light);
        
        .department-icon {
          width: 24px;
          height: 24px;
          object-fit: contain;
        }
      }
      
      .department-info {
        flex: 1;
        overflow: hidden;
        min-width: 0;
        
        .department-name {
          font-size: 15px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin-bottom: 6px;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }
        
        .department-meta {
          display: flex;
          flex-wrap: wrap;
          gap: 12px;
          font-size: 12px;
          color: var(--el-text-color-secondary);
          
          .department-path {
            color: var(--el-text-color-secondary);
            font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
          }
          
          .department-full-name {
            color: var(--el-text-color-regular);
          }
          
          .department-managers {
            display: inline-flex;
            align-items: center;
            gap: 4px;
            color: var(--el-text-color-secondary);
            
            .el-icon {
              font-size: 12px;
            }
          }
        }
      }
      
      .department-action {
        flex-shrink: 0;
        margin-left: 16px;
        
        .el-button {
          border-radius: 8px;
          font-weight: 500;
          transition: all 0.2s;
          
          &:hover {
            transform: scale(1.02);
            box-shadow: 0 2px 8px rgba(var(--el-color-primary-rgb), 0.3);
          }
          
          &.is-selected {
            background: var(--el-color-success);
            border-color: var(--el-color-success);
          }
        }
      }
      
      .department-empty {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 60px 24px;
        text-align: center;
        
        .empty-icon {
          font-size: 64px;
          color: var(--el-text-color-placeholder);
          margin-bottom: 16px;
          opacity: 0.4;
        }
        
        .empty-text {
          font-size: 16px;
          font-weight: 500;
          color: var(--el-text-color-secondary);
          margin-bottom: 8px;
        }
        
        .empty-desc {
          font-size: 14px;
          color: var(--el-text-color-placeholder);
        }
      }
    }
  }
}

// 滚动条样式
.department-list-section::-webkit-scrollbar {
  width: 6px;
}

.department-list-section::-webkit-scrollbar-track {
  background: var(--el-bg-color-page);
  border-radius: 3px;
}

.department-list-section::-webkit-scrollbar-thumb {
  background: var(--el-border-color-dark);
  border-radius: 3px;
  transition: background 0.2s;
}

.department-list-section::-webkit-scrollbar-thumb:hover {
  background: var(--el-text-color-placeholder);
}
</style>
