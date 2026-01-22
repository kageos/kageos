<!--
  DocsPathSelector - 文档路径选择器组件
  功能：
  - 支持多选文档路径
  - 显示已选中的路径
  - 点击后弹出对话框，搜索并选择文档
-->
<template>
  <div class="docs-path-selector">
    <!-- 已选中的路径显示 -->
    <div class="selected-paths">
      <div class="input-wrapper">
        <el-input
          v-model="pathsInput"
          type="textarea"
          :rows="2"
          placeholder="请输入文档路径，多个路径用逗号分隔，如：/system/official/sdk,/system/official/plugins"
          @blur="handleInputBlur"
        />
        <el-button
          :icon="Document"
          @click="handleOpenDialog"
          style="margin-left: 8px;"
        >
          搜索文档
        </el-button>
      </div>
      <div v-if="selectedPaths.length > 0" class="path-tags" style="margin-top: 8px;">
        <el-tag
          v-for="(path, index) in selectedPaths"
          :key="index"
          closable
          @close="handleRemovePath(index)"
          style="margin-right: 8px; margin-bottom: 4px;"
        >
          {{ path }}
        </el-tag>
      </div>
    </div>
    
    <!-- 文档选择对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title=""
      :show-close="false"
      :close-on-click-modal="true"
      :close-on-press-escape="true"
      width="600px"
      top="10vh"
      class="docs-selector-dialog"
      append-to-body
      @close="handleClose"
    >
      <div class="docs-selector-modal">
        <!-- 头部 -->
        <div class="docs-selector-header">
          <div class="header-content">
            <el-icon class="header-icon"><Document /></el-icon>
            <h3 class="header-title">选择文档</h3>
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
        <div class="docs-search-section">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索文档名称或路径..."
            size="large"
            class="docs-search-input"
            @input="handleSearchInput"
            clearable
          >
            <template #prefix>
              <el-icon class="search-icon"><Search /></el-icon>
            </template>
          </el-input>
        </div>

        <!-- 文档列表 -->
        <div class="docs-list-section" v-loading="searchLoading">
          <div class="docs-list">
            <div
              v-for="doc in searchResults"
              :key="doc.full_code_path"
              class="doc-item"
              :class="{ 'selected': tempSelectedPaths.includes(doc.full_code_path) }"
              @click="handleToggleDoc(doc.full_code_path)"
            >
              <!-- 文档图标 -->
              <div class="doc-icon-wrapper">
                <el-icon class="doc-icon"><Document /></el-icon>
              </div>

              <!-- 文档信息 -->
              <div class="doc-info">
                <div class="doc-name">{{ doc.name }}</div>
                <div class="doc-meta">
                  <span class="doc-path">{{ doc.full_code_path }}</span>
                  <span v-if="doc.summary" class="doc-summary">{{ doc.summary }}</span>
                </div>
              </div>

              <!-- 选择按钮 -->
              <div class="doc-action">
                <el-checkbox
                  :model-value="tempSelectedPaths.includes(doc.full_code_path)"
                  @change="handleToggleDoc(doc.full_code_path)"
                  @click.stop
                />
              </div>
            </div>
            
            <!-- 空状态 -->
            <div v-if="searchResults.length === 0 && !searchLoading" class="docs-empty">
              <el-icon class="empty-icon"><Document /></el-icon>
              <div class="empty-text">{{ searchKeyword ? '未找到文档' : '暂无文档' }}</div>
              <div class="empty-desc">{{ searchKeyword ? '请尝试其他搜索关键词' : '系统中暂无文档，请先创建文档' }}</div>
            </div>
          </div>
          
          <!-- 分页 -->
          <div v-if="searchTotal > 0" class="docs-pagination">
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
        
        <!-- 已选择提示 -->
        <div v-if="tempSelectedPaths.length > 0" class="docs-selected-info">
          <el-icon><Check /></el-icon>
          <span>已选择 {{ tempSelectedPaths.length }} 个文档</span>
        </div>
      </div>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="handleClose">取消</el-button>
          <el-button
            type="primary"
            @click="handleConfirm"
          >
            确定 ({{ tempSelectedPaths.length }})
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElButton, ElDialog, ElTag, ElInput, ElIcon, ElMessage, ElCheckbox, ElPagination } from 'element-plus'
import { Document, Search, Close, Check } from '@element-plus/icons-vue'
import { searchDocs, type DocSearchResult } from '@/api/doc'

interface Props {
  modelValue: string // 逗号分隔的路径字符串，如："/system/official/sdk,/user/myapp/docs"
  user?: string // 用户（可选）
  app?: string // 应用（可选）
}

const props = withDefaults(defineProps<Props>(), {
  user: '',
  app: ''
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const dialogVisible = ref(false)
const searchKeyword = ref('')
const searchLoading = ref(false)
const searchResults = ref<DocSearchResult[]>([])
const searchTotal = ref(0)
const searchPage = ref(1)
const searchPageSize = ref(20)
const tempSelectedPaths = ref<string[]>([])
const pathsInput = ref('')

// 当前选中的路径数组
const selectedPaths = computed({
  get: () => {
    if (!props.modelValue || typeof props.modelValue !== 'string') return []
    return props.modelValue.split(',').map(p => p.trim()).filter(p => p)
  },
  set: (value: string[]) => {
    emit('update:modelValue', value.length > 0 ? value.join(',') : '')
  }
})

// 同步输入框和选中路径
watch(() => props.modelValue, (newVal) => {
  pathsInput.value = newVal || ''
}, { immediate: true })

// 处理输入框失焦
const handleInputBlur = () => {
  const paths = pathsInput.value.split(',').map(p => p.trim()).filter(p => p)
  selectedPaths.value = paths
}

// 搜索文档（防抖）
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
    const resp = await searchDocs({
      keyword: searchKeyword.value.trim(), // 空字符串时返回最近的数据
      page: searchPage.value,
      page_size: searchPageSize.value,
      include_content: false // 列表展示不需要内容，减少数据传输
    })
    searchResults.value = resp.docs || []
    searchTotal.value = resp.total || 0
  } catch (error: any) {
    console.error('搜索文档失败:', error)
    ElMessage.error(error.message || '搜索文档失败')
    searchResults.value = []
    searchTotal.value = 0
  } finally {
    searchLoading.value = false
  }
}

// 切换文档选择
const handleToggleDoc = (path: string) => {
  const index = tempSelectedPaths.value.indexOf(path)
  if (index > -1) {
    tempSelectedPaths.value.splice(index, 1)
  } else {
    tempSelectedPaths.value.push(path)
  }
}

// 打开对话框
const handleOpenDialog = () => {
  dialogVisible.value = true
  searchKeyword.value = ''
  searchResults.value = []
  searchTotal.value = 0
  searchPage.value = 1
  tempSelectedPaths.value = [...selectedPaths.value]
  
  // 自动执行一次搜索（空关键词，返回最近的数据）
  handleSearch()
  
  // 延迟聚焦搜索框
  setTimeout(() => {
    const input = document.querySelector('.docs-search-input input') as HTMLInputElement
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
  // 合并已选中的路径和从搜索选择的路径（去重）
  const existingPaths = selectedPaths.value
  const newPaths = tempSelectedPaths.value
  const mergedPaths = Array.from(new Set([...existingPaths, ...newPaths]))
  selectedPaths.value = mergedPaths
  pathsInput.value = mergedPaths.join(',')
  handleClose()
}

// 移除路径
const handleRemovePath = (index: number) => {
  const newPaths = [...selectedPaths.value]
  newPaths.splice(index, 1)
  selectedPaths.value = newPaths
  pathsInput.value = newPaths.join(',')
}
</script>

<style lang="scss" scoped>
:deep(.docs-selector-dialog) {
  .el-dialog {
    border-radius: 20px;
    overflow: hidden;
    backdrop-filter: blur(20px);
    background: rgba(255, 255, 255, 0.95);
    border: 1px solid rgba(255, 255, 255, 0.2);
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
    animation: docsSelectorFadeIn 0.4s cubic-bezier(0.4, 0, 0.2, 1);
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

@keyframes docsSelectorFadeIn {
  from {
    opacity: 0;
    transform: scale(0.9) translateY(-20px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.docs-path-selector {
  .selected-paths {
    width: 100%;
    
    .input-wrapper {
      display: flex;
      align-items: flex-start;
      
      .el-textarea {
        flex: 1;
      }
    }
    
    .path-tags {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
    }
  }
}

.docs-selector-modal {
  .docs-selector-header {
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

  .docs-search-section {
    padding: 24px;
    background: var(--el-bg-color);
    
    .docs-search-input {
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

  .docs-list-section {
    max-height: 500px;
    overflow-y: auto;
    padding: 0 24px;
    
    .docs-list {
      .doc-item {
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
      
      .doc-icon-wrapper {
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
        
        .doc-icon {
          font-size: 20px;
          color: var(--el-color-primary);
        }
      }
      
      .doc-info {
        flex: 1;
        overflow: hidden;
        min-width: 0;
        
        .doc-name {
          font-size: 15px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin-bottom: 6px;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }
        
        .doc-meta {
          display: flex;
          flex-wrap: wrap;
          gap: 12px;
          font-size: 12px;
          color: var(--el-text-color-secondary);
          
          .doc-path {
            color: var(--el-text-color-secondary);
            font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
          }
          
          .doc-summary {
            color: var(--el-text-color-regular);
            max-width: 200px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
        }
      }
      
      .doc-action {
        flex-shrink: 0;
        margin-left: 16px;
      }
      
      .docs-empty {
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
    
    .docs-pagination {
      padding: 16px 0 24px;
      display: flex;
      justify-content: center;
    }
  }
  
  .docs-selected-info {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 24px;
    background: var(--el-color-primary-light-9);
    border-top: 1px solid var(--el-border-color-lighter);
    color: var(--el-color-primary);
    font-size: 14px;
    font-weight: 500;
    
    .el-icon {
      font-size: 16px;
    }
  }
}

// 滚动条样式
.docs-list-section::-webkit-scrollbar {
  width: 6px;
}

.docs-list-section::-webkit-scrollbar-track {
  background: var(--el-bg-color-page);
  border-radius: 3px;
}

.docs-list-section::-webkit-scrollbar-thumb {
  background: var(--el-border-color-dark);
  border-radius: 3px;
  transition: background 0.2s;
}

.docs-list-section::-webkit-scrollbar-thumb:hover {
  background: var(--el-text-color-placeholder);
}
</style>
