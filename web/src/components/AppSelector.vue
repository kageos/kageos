<template>
  <el-dialog
    v-model="visible"
    title=""
    :show-close="false"
    :close-on-click-modal="true"
    :close-on-press-escape="true"
    width="480px"
    top="20vh"
    class="app-selector-dialog"
    append-to-body
  >
    <div class="app-selector-modal">
      <!-- 头部 -->
      <div class="app-selector-header">
        <div class="header-content">
          <el-icon class="header-icon"><FolderOpened /></el-icon>
          <h3 class="header-title">选择目标应用</h3>
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
      <div class="app-search-section">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索应用名称或代码..."
          size="large"
          class="app-search-input"
          @input="handleSearchInput"
          clearable
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <!-- 应用列表 -->
      <div class="app-list-section" v-loading="loading">
        <div class="app-list">
          <div
            v-for="app in appOptions"
            :key="app.id"
            class="app-item"
            @click="handleSelectApp(app)"
          >
            <!-- 应用图标 -->
            <div class="app-icon-wrapper">
              <el-icon class="app-icon"><FolderOpened /></el-icon>
            </div>

            <!-- 应用信息 -->
            <div class="app-info">
              <div class="app-name">{{ app.name }}</div>
              <div class="app-code">{{ app.code }}</div>
            </div>

            <!-- 选择按钮 -->
            <div class="app-action">
              <el-button
                type="primary"
                size="small"
                @click.stop="handleSelectApp(app)"
              >
                选择
              </el-button>
            </div>
          </div>
          
          <div v-if="appOptions.length === 0 && !loading" class="app-empty">
            <el-icon class="empty-icon"><FolderOpened /></el-icon>
            <div class="empty-text">暂无应用</div>
            <div class="empty-desc">请尝试其他搜索关键词</div>
          </div>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { Close, Search, FolderOpened } from '@element-plus/icons-vue'
import { getAppList } from '@/api/app'
import type { App } from '@/types'
import { Logger } from '@/core/utils/logger'

// Props
interface Props {
  modelValue: boolean
}

const props = defineProps<Props>()

// Emits
interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'select', app: App): void
}

const emit = defineEmits<Emits>()

// 本地状态
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const searchKeyword = ref('')
const appOptions = ref<App[]>([])
const loading = ref(false)

// 防抖定时器
let searchDebounceTimer: NodeJS.Timeout | null = null

// 应用搜索
const handleAppSearch = async (query: string) => {
  if (!query || !query.trim()) {
    // 如果没有查询词，加载默认应用列表
    await loadDefaultApps()
    return
  }
  
  loading.value = true
  try {
    const apps = await getAppList(200, query.trim())
    appOptions.value = apps || []
    Logger.info('AppSelector', '搜索应用成功', { query, count: appOptions.value.length })
  } catch (error) {
    Logger.error('AppSelector', '搜索应用失败', error)
    appOptions.value = []
  } finally {
    loading.value = false
  }
}

// 加载默认应用列表
const loadDefaultApps = async () => {
  loading.value = true
  try {
    const apps = await getAppList(200)
    appOptions.value = apps || []
    Logger.info('AppSelector', '加载应用列表成功', { count: appOptions.value.length })
  } catch (error) {
    Logger.error('AppSelector', '加载应用列表失败', error)
    appOptions.value = []
  } finally {
    loading.value = false
  }
}

// 处理搜索输入
const handleSearchInput = (value: string) => {
  searchKeyword.value = value
  // 防抖处理
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer)
  }
  
  searchDebounceTimer = setTimeout(() => {
    handleAppSearch(value)
  }, 300)
}

// 处理应用选择
const handleSelectApp = (app: App) => {
  emit('select', app)
  handleClose()
}

// 关闭弹窗
const handleClose = () => {
  visible.value = false
  
  // 清理防抖定时器
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer)
    searchDebounceTimer = null
  }
  
  // 重置状态
  searchKeyword.value = ''
  appOptions.value = []
}

// 监听弹窗打开，自动聚焦搜索框并初始化应用列表
watch(visible, (newVal) => {
  if (newVal) {
    // 清空搜索关键词
    searchKeyword.value = ''
    // 延迟聚焦，确保DOM已渲染
    setTimeout(() => {
      const input = document.querySelector('.app-search-input input') as HTMLInputElement
      if (input) {
        input.focus()
      }
    }, 200)
    
    // 加载应用列表
    loadDefaultApps()
  }
})

// 组件销毁时清理定时器
onUnmounted(() => {
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer)
    searchDebounceTimer = null
  }
})
</script>

<style lang="scss" scoped>
:deep(.app-selector-dialog) {
  .el-dialog {
    border-radius: 20px;
    overflow: hidden;
    backdrop-filter: blur(20px);
    background: rgba(255, 255, 255, 0.95);
    border: 1px solid rgba(255, 255, 255, 0.2);
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
    animation: appSelectorFadeIn 0.4s cubic-bezier(0.4, 0, 0.2, 1);
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

@keyframes appSelectorFadeIn {
  from {
    opacity: 0;
    transform: scale(0.9) translateY(-20px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.app-selector-modal {
  .app-selector-header {
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

  .app-search-section {
    padding: 24px;
    background: var(--el-bg-color);
    
    .app-search-input {
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

  .app-list-section {
    max-height: 400px;
    overflow-y: auto;
    padding: 0 24px 24px;
    
    .app-list {
      .app-item {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 12px 16px;
        margin-bottom: 6px;
        cursor: pointer;
        border-radius: 10px;
        background: var(--el-bg-color);
        border: 1px solid var(--el-border-color-lighter);
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
        
        &:hover {
          background: var(--el-fill-color-light);
          border-color: var(--el-border-color);
          transform: translateY(-1px);
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
        }
        
        &:active {
          transform: translateY(-1px);
        }
      }
      
      .app-icon-wrapper {
        flex-shrink: 0;
        margin-right: 12px;
        width: 32px;
        height: 32px;
        display: flex;
        align-items: center;
        justify-content: center;
        background: var(--el-fill-color-lighter);
        border-radius: 8px;
        border: 1px solid var(--el-border-color-light);
        
        .app-icon {
          font-size: 16px;
          color: var(--el-text-color-regular);
          opacity: 0.7;
        }
      }
      
      .app-info {
        flex: 1;
        overflow: hidden;
        
        .app-name {
          font-size: 14px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin-bottom: 2px;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }
        
        .app-code {
          font-size: 12px;
          color: var(--el-text-color-secondary);
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }
      }
      
      .app-action {
        flex-shrink: 0;
        margin-left: 12px;
        
        .el-button {
          border-radius: 8px;
          font-weight: 500;
          transition: all 0.2s;
          
          &:hover {
            transform: scale(1.02);
            box-shadow: 0 1px 4px rgba(var(--el-color-primary-rgb), 0.2);
          }
        }
      }
      
      .app-empty {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 48px 24px;
        text-align: center;
        
        .empty-icon {
          font-size: 48px;
          color: var(--el-text-color-placeholder);
          margin-bottom: 16px;
          opacity: 0.6;
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
.app-list-section::-webkit-scrollbar {
  width: 6px;
}

.app-list-section::-webkit-scrollbar-track {
  background: var(--el-bg-color-page);
  border-radius: 3px;
}

.app-list-section::-webkit-scrollbar-thumb {
  background: var(--el-border-color-dark);
  border-radius: 3px;
  transition: background 0.2s;
}

.app-list-section::-webkit-scrollbar-thumb:hover {
  background: var(--el-text-color-placeholder);
}
</style>

