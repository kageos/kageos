<template>
  <el-dialog
    v-model="visible"
    title=""
    :show-close="false"
    :close-on-click-modal="true"
    :close-on-press-escape="true"
    width="600px"
    top="10vh"
    class="user-selector-dialog"
    append-to-body
    @close="handleClose"
  >
    <div class="user-selector-modal">
      <!-- 头部 -->
      <div class="user-selector-header">
        <div class="header-content">
          <el-icon class="header-icon"><User /></el-icon>
          <h3 class="header-title">选择用户</h3>
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
      <div class="user-search-section">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索用户名或邮箱..."
          size="large"
          class="user-search-input"
          @input="handleSearchInput"
          clearable
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <!-- 用户列表 -->
      <div class="user-list-section" v-loading="loading">
        <div class="user-list">
          <div
            v-for="user in userOptions"
            :key="user.username"
            class="user-item"
            :class="{ 'selected': selectedUser?.username === user.username }"
            @click="handleSelectUser(user)"
          >
            <!-- 用户头像 -->
            <div class="user-avatar-wrapper">
              <el-avatar
                :src="user.avatar"
                :size="40"
                class="user-avatar"
              >
                {{ user.username?.[0]?.toUpperCase() || 'U' }}
              </el-avatar>
            </div>

            <!-- 用户信息 -->
            <div class="user-info">
              <div class="user-name">{{ user.username }}</div>
              <div class="user-meta">
                <span v-if="user.nickname" class="user-nickname">{{ user.nickname }}</span>
                <span v-if="user.email" class="user-email">{{ user.email }}</span>
                <span v-if="user.department_name || user.department_full_path" class="user-department">
                  <el-icon><OfficeBuilding /></el-icon>
                  {{ user.department_name || user.department_full_path }}
                </span>
              </div>
            </div>

            <!-- 选择按钮 -->
            <div class="user-action">
              <el-button
                type="primary"
                size="small"
                :class="{ 'is-selected': selectedUser?.username === user.username }"
                @click.stop="handleSelectUser(user)"
              >
                {{ selectedUser?.username === user.username ? '已选择' : '选择' }}
              </el-button>
            </div>
          </div>
          
          <div v-if="userOptions.length === 0 && !loading" class="user-empty">
            <el-icon class="empty-icon"><User /></el-icon>
            <div class="empty-text">暂无用户</div>
            <div class="empty-desc">请尝试其他搜索关键词</div>
          </div>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { Close, Search, User, OfficeBuilding } from '@element-plus/icons-vue'
import { searchUsersFuzzy } from '@/api/user'
import type { UserInfo } from '@/types'

// Props
interface Props {
  modelValue: boolean
  selectedUser?: UserInfo | null
}

const props = withDefaults(defineProps<Props>(), {
  selectedUser: null
})

// Emits
interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'select', user: UserInfo): void
}

const emit = defineEmits<Emits>()

// 本地状态
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const searchKeyword = ref('')
const userOptions = ref<UserInfo[]>([])
const loading = ref(false)

// 防抖定时器
let searchDebounceTimer: NodeJS.Timeout | null = null

// 用户搜索
const handleUserSearch = async (query: string) => {
  if (!query || query.trim().length < 1) {
    userOptions.value = []
    return
  }
  
  loading.value = true
  try {
    const response = await searchUsersFuzzy(query.trim(), 30)
    userOptions.value = response.users || []
  } catch (error) {
    console.error('搜索用户失败:', error)
    userOptions.value = []
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
    handleUserSearch(value)
  }, 300)
}

// 处理用户选择
const handleSelectUser = (user: UserInfo) => {
  emit('select', user)
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
  userOptions.value = []
}

// 监听弹窗打开，自动聚焦搜索框
watch(visible, (newVal) => {
  if (newVal) {
    // 清空搜索关键词
    searchKeyword.value = ''
    userOptions.value = []
    // 延迟聚焦，确保DOM已渲染
    setTimeout(() => {
      const input = document.querySelector('.user-search-input input') as HTMLInputElement
      if (input) {
        input.focus()
      }
    }, 200)
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
:deep(.user-selector-dialog) {
  .el-dialog {
    border-radius: 20px;
    overflow: hidden;
    backdrop-filter: blur(20px);
    background: rgba(255, 255, 255, 0.95);
    border: 1px solid rgba(255, 255, 255, 0.2);
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
    animation: userSelectorFadeIn 0.4s cubic-bezier(0.4, 0, 0.2, 1);
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

@keyframes userSelectorFadeIn {
  from {
    opacity: 0;
    transform: scale(0.9) translateY(-20px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.user-selector-modal {
  .user-selector-header {
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

  .user-search-section {
    padding: 24px;
    background: var(--el-bg-color);
    
    .user-search-input {
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

  .user-list-section {
    max-height: 500px;
    overflow-y: auto;
    padding: 0 24px 24px;
    
    .user-list {
      .user-item {
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
      
      .user-avatar-wrapper {
        flex-shrink: 0;
        margin-right: 16px;
        
        .user-avatar {
          border: 2px solid var(--el-border-color-light);
        }
      }
      
      .user-info {
        flex: 1;
        overflow: hidden;
        min-width: 0;
        
        .user-name {
          font-size: 15px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin-bottom: 6px;
          white-space: nowrap;
          overflow: hidden;
          text-overflow: ellipsis;
        }
        
        .user-meta {
          display: flex;
          flex-wrap: wrap;
          gap: 12px;
          font-size: 12px;
          color: var(--el-text-color-secondary);
          
          .user-nickname {
            color: var(--el-text-color-regular);
          }
          
          .user-email {
            color: var(--el-text-color-secondary);
          }
          
          .user-department {
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
      
      .user-action {
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
      
      .user-empty {
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
.user-list-section::-webkit-scrollbar {
  width: 6px;
}

.user-list-section::-webkit-scrollbar-track {
  background: var(--el-bg-color-page);
  border-radius: 3px;
}

.user-list-section::-webkit-scrollbar-thumb {
  background: var(--el-border-color-dark);
  border-radius: 3px;
  transition: background 0.2s;
}

.user-list-section::-webkit-scrollbar-thumb:hover {
  background: var(--el-text-color-placeholder);
}
</style>
