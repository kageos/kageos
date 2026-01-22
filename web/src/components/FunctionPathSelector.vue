<!--
  FunctionPathSelector - 函数路径选择器组件
  功能：
  - 单选函数路径
  - 显示已选中的路径
  - 点击后弹出对话框，搜索并选择函数
-->
<template>
  <div class="function-path-selector">
    <!-- 已选中的路径显示 -->
    <div class="selected-path">
      <el-input
        v-model="pathInput"
        placeholder="请输入函数路径，如：/system/official/agent/plugin/excel_or_csv"
        clearable
        @blur="handleInputBlur"
      >
        <template #append>
          <el-button
            :icon="Operation"
            @click="handleOpenDialog"
          >
            搜索函数
          </el-button>
        </template>
      </el-input>
      <div v-if="selectedPath" class="path-tag" style="margin-top: 8px;">
        <el-tag closable @close="handleClear">
          {{ selectedPath }}
        </el-tag>
      </div>
    </div>
    
    <!-- 函数选择对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title=""
      :show-close="false"
      :close-on-click-modal="true"
      :close-on-press-escape="true"
      width="600px"
      top="10vh"
      class="function-selector-dialog"
      append-to-body
      @close="handleClose"
    >
      <div class="function-selector-modal">
        <!-- 头部 -->
        <div class="function-selector-header">
          <div class="header-content">
            <el-icon class="header-icon"><Operation /></el-icon>
            <h3 class="header-title">选择函数</h3>
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
        <div class="function-search-section">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索函数名称或路径..."
            size="large"
            class="function-search-input"
            @input="handleSearchInput"
            clearable
          >
            <template #prefix>
              <el-icon class="search-icon"><Search /></el-icon>
            </template>
          </el-input>
        </div>

        <!-- 函数列表 -->
        <div class="function-list-section" v-loading="searchLoading">
          <div class="function-list">
            <div
              v-for="func in searchResults"
              :key="func.full_code_path"
              class="function-item"
              :class="{ 'selected': tempSelectedPath === func.full_code_path }"
              @click="handleSelectFunction(func.full_code_path)"
            >
              <!-- 函数图标 -->
              <div class="function-icon-wrapper">
                <el-icon class="function-icon"><Operation /></el-icon>
              </div>

              <!-- 函数信息 -->
              <div class="function-info">
                <div class="function-name">{{ func.name }}</div>
                <div class="function-meta">
                  <span class="function-path">{{ func.full_code_path }}</span>
                  <el-tag v-if="func.template_type" size="small" type="info" style="margin-left: 8px;">
                    {{ func.template_type }}
                  </el-tag>
                </div>
                <div v-if="func.description" class="function-description">
                  {{ func.description }}
                </div>
              </div>

              <!-- 选择指示 -->
              <div class="function-action">
                <el-icon v-if="tempSelectedPath === func.full_code_path" class="check-icon">
                  <Check />
                </el-icon>
              </div>
            </div>
            
            <!-- 空状态 -->
            <div v-if="searchResults.length === 0 && !searchLoading" class="function-empty">
              <el-icon class="empty-icon"><Operation /></el-icon>
              <div class="empty-text">{{ searchKeyword ? '未找到函数' : '请输入关键词搜索函数' }}</div>
              <div class="empty-desc">{{ searchKeyword ? '请尝试其他搜索关键词' : '搜索函数名称或路径' }}</div>
            </div>
          </div>
          
          <!-- 分页 -->
          <div v-if="searchTotal > 0" class="function-pagination">
            <el-pagination
              v-model:current-page="searchPage"
              v-model:page-size="searchPageSize"
              :total="searchTotal"
              :page-sizes="[10, 20, 50, 100]"
              layout="total, sizes, prev, pager, next"
              @size-change="handleSearch"
              @current-change="handleSearch"
            />
          </div>
        </div>
      </div>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="handleClose">取消</el-button>
          <el-button
            type="primary"
            :disabled="!tempSelectedPath"
            @click="handleConfirm"
          >
            确定
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElButton, ElDialog, ElTag, ElInput, ElIcon, ElMessage, ElPagination } from 'element-plus'
import { Operation, Search, Close, Check } from '@element-plus/icons-vue'
import { searchFunctions, type FunctionSearchResult } from '@/api/service-tree'

interface Props {
  modelValue: string // 函数路径（单选）
  user?: string // 用户（可选，默认 system）
  app?: string // 应用（可选，默认 official）
  templateType?: string // 函数类型（可选，默认 form）
}

const props = withDefaults(defineProps<Props>(), {
  user: 'system',
  app: 'official',
  templateType: 'form'
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const dialogVisible = ref(false)
const searchKeyword = ref('')
const searchLoading = ref(false)
const searchResults = ref<FunctionSearchResult[]>([])
const searchTotal = ref(0)
const searchPage = ref(1)
const searchPageSize = ref(20)
const tempSelectedPath = ref<string>('')
const pathInput = ref('')

// 当前选中的路径
const selectedPath = computed({
  get: () => props.modelValue || '',
  set: (value: string) => {
    emit('update:modelValue', value)
  }
})

// 同步输入框和选中路径
watch(() => props.modelValue, (newVal) => {
  pathInput.value = newVal || ''
}, { immediate: true })

// 处理输入框失焦
const handleInputBlur = () => {
  selectedPath.value = pathInput.value.trim()
}

// 搜索函数（防抖）
let searchTimer: ReturnType<typeof setTimeout> | null = null
const handleSearchInput = () => {
  if (searchTimer) {
    clearTimeout(searchTimer)
  }
  searchTimer = setTimeout(() => {
    searchPage.value = 1
    handleSearch()
  }, 300) // 300ms 防抖
}

// 执行搜索（支持空关键词，返回最近的数据）
const handleSearch = async () => {
  searchLoading.value = true
  try {
    const resp = await searchFunctions({
      user: props.user,
      app: props.app,
      keyword: searchKeyword.value.trim(), // 空字符串时返回最近的数据
      template_type: props.templateType,
      page: searchPage.value,
      page_size: searchPageSize.value
    })
    searchResults.value = resp.functions || []
    searchTotal.value = resp.total || 0
  } catch (error: any) {
    console.error('搜索函数失败:', error)
    ElMessage.error(error.message || '搜索函数失败')
    searchResults.value = []
    searchTotal.value = 0
  } finally {
    searchLoading.value = false
  }
}

// 选择函数
const handleSelectFunction = (path: string) => {
  tempSelectedPath.value = path
}

// 打开对话框
const handleOpenDialog = () => {
  dialogVisible.value = true
  searchKeyword.value = ''
  searchResults.value = []
  searchTotal.value = 0
  searchPage.value = 1
  tempSelectedPath.value = selectedPath.value
  
  // 自动执行一次搜索（空关键词，返回最近的数据）
  handleSearch()
  
  // 延迟聚焦搜索框
  setTimeout(() => {
    const input = document.querySelector('.function-search-input input') as HTMLInputElement
    if (input) {
      input.focus()
    }
  }, 200)
}

// 关闭对话框
const handleClose = () => {
  dialogVisible.value = false
  searchKeyword.value = ''
}

// 确认选择
const handleConfirm = () => {
  selectedPath.value = tempSelectedPath.value
  pathInput.value = tempSelectedPath.value
  handleClose()
}

// 清空选择
const handleClear = () => {
  selectedPath.value = ''
  pathInput.value = ''
}
</script>

<style lang="scss" scoped>
:deep(.function-selector-dialog) {
  .el-dialog {
    border-radius: 20px;
    overflow: hidden;
    backdrop-filter: blur(20px);
    background: rgba(255, 255, 255, 0.95);
    border: 1px solid rgba(255, 255, 255, 0.2);
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
    animation: functionSelectorFadeIn 0.4s cubic-bezier(0.4, 0, 0.2, 1);
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

@keyframes functionSelectorFadeIn {
  from {
    opacity: 0;
    transform: scale(0.9) translateY(-20px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.function-path-selector {
  .selected-path {
    width: 100%;
    
    .path-tag {
      display: flex;
      align-items: center;
    }
  }
}

.function-selector-modal {
  .function-selector-header {
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
      padding: 4px;
      transition: color 0.2s;
      color: var(--el-text-color-secondary);
      
      &:hover {
        color: var(--el-text-color-primary);
      }
    }
  }

  .function-search-section {
    padding: 24px;
    background: var(--el-bg-color);
    
    .function-search-input {
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

  .function-list-section {
    max-height: 500px;
    overflow-y: auto;
    padding: 0 24px;
    
    .function-list {
      .function-item {
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
          border-color: var(--el-color-primary);
          border-width: 2px;
          box-shadow: 0 0 0 2px rgba(var(--el-color-primary-rgb), 0.1);
        }
      }
      
      .function-icon-wrapper {
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
        
        .function-icon {
          font-size: 20px;
          color: var(--el-color-primary);
        }
      }
      
      .function-info {
        flex: 1;
        overflow: hidden;
        min-width: 0;
        
        .function-name {
          font-size: 15px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin-bottom: 6px;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }
        
        .function-meta {
          display: flex;
          flex-wrap: wrap;
          align-items: center;
          gap: 8px;
          font-size: 12px;
          color: var(--el-text-color-secondary);
          margin-bottom: 4px;
          
          .function-path {
            color: var(--el-text-color-secondary);
            font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
          }
        }
        
        .function-description {
          font-size: 12px;
          color: var(--el-text-color-regular);
          line-height: 1.5;
          margin-top: 4px;
          display: -webkit-box;
          -webkit-line-clamp: 2;
          -webkit-box-orient: vertical;
          overflow: hidden;
        }
      }
      
      .function-action {
        flex-shrink: 0;
        margin-left: 16px;
        
        .check-icon {
          font-size: 20px;
          color: var(--el-color-primary);
        }
      }
      
      .function-empty {
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
    
    .function-pagination {
      padding: 16px 0 24px;
      display: flex;
      justify-content: center;
    }
  }
}

// 滚动条样式
.function-list-section::-webkit-scrollbar {
  width: 6px;
}

.function-list-section::-webkit-scrollbar-track {
  background: var(--el-bg-color-page);
  border-radius: 3px;
}

.function-list-section::-webkit-scrollbar-thumb {
  background: var(--el-border-color-dark);
  border-radius: 3px;
  transition: background 0.2s;
}

.function-list-section::-webkit-scrollbar-thumb:hover {
  background: var(--el-text-color-placeholder);
}
</style>
