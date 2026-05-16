<!--
  LLMSelector - LLM 配置选择器组件
  功能：
  - 单选 LLM 配置
  - 显示已选中的配置
  - 点击后弹出对话框，搜索并选择 LLM
-->
<template>
  <div class="llm-selector">
    <!-- 已选中的配置显示 -->
    <div class="selected-llm">
      <el-input
        v-model="llmInput"
        placeholder="请选择 LLM 配置（留空则使用默认 LLM）"
        clearable
        readonly
        @click="handleOpenDialog"
      >
        <template #append>
          <el-button
            :icon="CircleCheck"
            @click="handleOpenDialog"
          >
            选择 LLM
          </el-button>
        </template>
      </el-input>
      <div v-if="selectedLLM" class="llm-tag" style="margin-top: 8px;">
        <el-tag closable @close="handleClear" :type="selectedLLM.is_default ? 'success' : 'info'">
          {{ selectedLLM.name }} ({{ selectedLLM.model }}){{ selectedLLM.is_default ? ' (默认)' : '' }}
        </el-tag>
      </div>
    </div>
    
    <!-- LLM 选择对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title=""
      :show-close="false"
      :close-on-click-modal="true"
      :close-on-press-escape="true"
      width="800px"
      top="10vh"
      class="llm-selector-dialog"
      append-to-body
      @close="handleClose"
    >
      <div class="llm-selector-modal">
        <!-- 头部 -->
        <div class="llm-selector-header">
          <div class="header-content">
            <el-icon class="header-icon"><CircleCheck /></el-icon>
            <h3 class="header-title">选择 LLM 配置</h3>
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
        <div class="llm-search-section">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索 LLM 配置名称、模型或 API Base..."
            size="large"
            class="llm-search-input"
            @input="handleSearchInput"
            clearable
          >
            <template #prefix>
              <el-icon class="search-icon"><Search /></el-icon>
            </template>
          </el-input>
        </div>

        <!-- LLM 列表 -->
        <div class="llm-list-section" v-loading="searchLoading">
          <div class="llm-list">
            <div
              v-for="llm in searchResults"
              :key="llm.id"
              class="llm-item"
              :class="{ 'selected': tempSelectedLLMId === llm.id }"
              @click="handleSelectLLM(llm)"
            >
              <!-- LLM 图标 -->
              <div class="llm-icon-wrapper">
                <el-icon class="llm-icon"><CircleCheck /></el-icon>
              </div>

              <!-- LLM 信息 -->
              <div class="llm-info">
                <div class="llm-name">
                  {{ llm.name }}
                  <el-tag v-if="llm.is_default" size="small" type="success" style="margin-left: 8px;">
                    默认
                  </el-tag>
                </div>
                <div class="llm-meta">
                  <span class="llm-model">{{ llm.model }}</span>
                </div>
                <div v-if="llm.api_base" class="llm-description">
                  API Base: {{ llm.api_base }}
                </div>
              </div>

              <!-- 选择指示 -->
              <div class="llm-action">
                <el-icon v-if="tempSelectedLLMId === llm.id" class="check-icon">
                  <Check />
                </el-icon>
              </div>
            </div>
            
            <!-- 空状态 -->
            <div v-if="searchResults.length === 0 && !searchLoading" class="llm-empty">
              <el-icon class="empty-icon"><CircleCheck /></el-icon>
              <div class="empty-text">{{ searchKeyword ? '未找到 LLM 配置' : '暂无 LLM 配置' }}</div>
              <div class="empty-desc">{{ searchKeyword ? '请尝试其他搜索关键词' : '请先创建 LLM 配置' }}</div>
            </div>
          </div>
          
          <!-- 分页 -->
          <div v-if="searchTotal > 0" class="llm-pagination">
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
            :disabled="tempSelectedLLMId === null || tempSelectedLLMId === 0"
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
import { CircleCheck, Search, Close, Check } from '@element-plus/icons-vue'
import { getLLMList, type LLMInfo } from '@/architecture/presentation/context/api/agent'

interface Props {
  modelValue: number // LLM 配置 ID（0 表示使用默认 LLM）
  scope?: 'mine' | 'market' // 范围：mine 或 market
}

const props = withDefaults(defineProps<Props>(), {
  scope: 'market'
})

const emit = defineEmits<{
  'update:modelValue': [value: number]
}>()

const dialogVisible = ref(false)
const searchKeyword = ref('')
const searchLoading = ref(false)
const searchResults = ref<LLMInfo[]>([])
const searchTotal = ref(0)
const searchPage = ref(1)
const searchPageSize = ref(20)
const tempSelectedLLMId = ref<number | null>(null)
const llmInput = ref('')
const allLLMs = ref<LLMInfo[]>([]) // 所有已加载的 LLM 列表，用于查找

// 当前选中的 LLM
const selectedLLM = computed(() => {
  if (!props.modelValue || props.modelValue === 0) return null
  return allLLMs.value.find(llm => llm.id === props.modelValue) || null
})

// 同步输入框和选中 LLM
watch(() => props.modelValue, async (newVal) => {
  if (newVal === 0 || !newVal) {
    llmInput.value = '使用默认 LLM'
  } else {
    // 先从已加载的列表中查找
    let llm = allLLMs.value.find(l => l.id === newVal)
    
    // 如果找不到，尝试加载
    if (!llm) {
      try {
        const resp = await getLLMList({
          scope: props.scope,
          page: 1,
          page_size: 1000
        })
        allLLMs.value = resp.configs || []
        llm = allLLMs.value.find(l => l.id === newVal)
      } catch (error) {
        console.error('加载 LLM 配置失败:', error)
      }
    }
    
    if (llm) {
      llmInput.value = `${llm.name} (${llm.model})`
    } else {
      llmInput.value = `ID: ${newVal}`
    }
  }
}, { immediate: true })

// 搜索 LLM（防抖）
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

// 执行搜索（支持空关键词，返回所有数据）
const handleSearch = async () => {
  searchLoading.value = true
  try {
    const resp = await getLLMList({
      scope: props.scope,
      page: searchPage.value,
      page_size: searchPageSize.value
    })
    
    let llms = resp.configs || []
    
    // 更新所有 LLM 列表（合并，避免重复）
    const llmMap = new Map<number, LLMInfo>()
    allLLMs.value.forEach((llm: LLMInfo) => llmMap.set(llm.id, llm))
    llms.forEach((llm: LLMInfo) => llmMap.set(llm.id, llm))
    allLLMs.value = Array.from(llmMap.values())
    
    // 如果有搜索关键词，进行过滤
    if (searchKeyword.value.trim()) {
      const keyword = searchKeyword.value.trim().toLowerCase()
      llms = llms.filter(llm => 
        llm.name.toLowerCase().includes(keyword) ||
        llm.model.toLowerCase().includes(keyword) ||
        (llm.api_base || '').toLowerCase().includes(keyword)
      )
    }
    
    searchResults.value = llms
    searchTotal.value = llms.length
  } catch (error: any) {
    console.error('搜索 LLM 配置失败:', error)
    ElMessage.error(error.message || '搜索 LLM 配置失败')
    searchResults.value = []
    searchTotal.value = 0
  } finally {
    searchLoading.value = false
  }
}

// 选择 LLM
const handleSelectLLM = (llm: LLMInfo) => {
  tempSelectedLLMId.value = llm.id
}

// 打开对话框
const handleOpenDialog = () => {
  dialogVisible.value = true
  searchKeyword.value = ''
  searchResults.value = []
  searchTotal.value = 0
  searchPage.value = 1
  tempSelectedLLMId.value = props.modelValue === 0 ? null : props.modelValue
  
  // 自动执行一次搜索（空关键词，返回所有数据）
  handleSearch()
  
  // 延迟聚焦搜索框
  setTimeout(() => {
    const input = document.querySelector('.llm-search-input input') as HTMLInputElement
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
  emit('update:modelValue', tempSelectedLLMId.value || 0)
  handleClose()
}

// 清空选择
const handleClear = () => {
  emit('update:modelValue', 0)
  llmInput.value = '使用默认 LLM'
}
</script>

<style lang="scss" scoped>
:deep(.llm-selector-dialog) {
  .el-dialog {
    border-radius: 20px;
    overflow: hidden;
    backdrop-filter: blur(20px);
    background: rgba(255, 255, 255, 0.95);
    border: 1px solid rgba(255, 255, 255, 0.2);
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
    animation: llmSelectorFadeIn 0.4s cubic-bezier(0.4, 0, 0.2, 1);
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

@keyframes llmSelectorFadeIn {
  from {
    opacity: 0;
    transform: scale(0.9) translateY(-20px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.llm-selector {
  .selected-llm {
    width: 100%;
    
    .llm-tag {
      display: flex;
      align-items: center;
    }
  }
}

.llm-selector-modal {
  .llm-selector-header {
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

  .llm-search-section {
    padding: 24px;
    background: var(--el-bg-color);
    
    .llm-search-input {
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

  .llm-list-section {
    max-height: 500px;
    overflow-y: auto;
    padding: 0 24px;
    
    .llm-list {
      .llm-item {
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
      
      .llm-icon-wrapper {
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
        
        .llm-icon {
          font-size: 20px;
          color: var(--el-color-primary);
        }
      }
      
      .llm-info {
        flex: 1;
        overflow: hidden;
        min-width: 0;
        
        .llm-name {
          font-size: 15px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin-bottom: 6px;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
          display: flex;
          align-items: center;
        }
        
        .llm-meta {
          display: flex;
          align-items: center;
          gap: 4px;
          font-size: 12px;
          color: var(--el-text-color-secondary);
          margin-bottom: 4px;
          
          .llm-model {
            color: var(--el-text-color-secondary);
            font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
          }
        }
        
        .llm-description {
          font-size: 12px;
          color: var(--el-text-color-regular);
          line-height: 1.5;
          margin-top: 4px;
          display: -webkit-box;
          -webkit-line-clamp: 1;
          -webkit-box-orient: vertical;
          overflow: hidden;
        }
      }
      
      .llm-action {
        flex-shrink: 0;
        margin-left: 16px;
        
        .check-icon {
          font-size: 20px;
          color: var(--el-color-primary);
        }
      }
      
      .llm-empty {
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
    
    .llm-pagination {
      padding: 16px 0 24px;
      display: flex;
      justify-content: center;
    }
  }
}

// 滚动条样式
.llm-list-section::-webkit-scrollbar {
  width: 6px;
}

.llm-list-section::-webkit-scrollbar-track {
  background: var(--el-bg-color-page);
  border-radius: 3px;
}

.llm-list-section::-webkit-scrollbar-thumb {
  background: var(--el-border-color-dark);
  border-radius: 3px;
  transition: background 0.2s;
}

.llm-list-section::-webkit-scrollbar-thumb:hover {
  background: var(--el-text-color-placeholder);
}
</style>
