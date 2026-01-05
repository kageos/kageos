<!--
  UserSearchDialog - 用户搜索弹窗组件
  功能：
  - 弹窗式用户搜索和选择
  - 支持单选模式
  - 搜索、选择、确认
-->
<template>
  <el-dialog
    v-model="dialogVisible"
    :title="title"
    width="500px"
    :close-on-click-modal="false"
    @close="handleClose"
    @opened="handleDialogOpened"
  >
    <!-- 搜索框 -->
    <div class="user-search-dialog-search">
      <el-input
        ref="inputRef"
        v-model="searchKeyword"
        :placeholder="placeholder"
        :clearable="true"
        :loading="loading"
        @input="handleSearch"
        @clear="handleClearSearch"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
    </div>

    <!-- 用户列表 -->
    <div class="user-search-dialog-list">
      <div
        v-if="loading"
        class="user-search-dialog-loading"
      >
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>搜索中...</span>
      </div>
      <div
        v-else-if="userList.length === 0 && searchKeyword"
        class="user-search-dialog-empty"
      >
        <el-empty description="未找到用户" :image-size="80" />
      </div>
      <div
        v-else-if="userList.length === 0 && !searchKeyword"
        class="user-search-dialog-empty"
      >
        <el-empty description="请输入关键词搜索用户" :image-size="80" />
      </div>
      <div
        v-else
        class="user-search-dialog-items"
      >
        <div
          v-for="user in userList"
          :key="user.username"
          class="user-search-dialog-item"
          :class="{ 'is-selected': selectedUser?.username === user.username }"
          @click="handleSelectUser(user)"
        >
          <el-avatar :src="user.avatar" :size="40" class="user-avatar">
            {{ user.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <div class="user-info">
            <div class="user-name">{{ user.username }}</div>
            <div v-if="user.nickname" class="user-nickname">{{ user.nickname }}</div>
            <div v-if="user.email" class="user-email">{{ user.email }}</div>
            <div v-if="user.signature" class="user-signature">{{ user.signature }}</div>
          </div>
          <el-icon v-if="selectedUser?.username === user.username" class="selected-icon">
            <Check />
          </el-icon>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="user-search-dialog-footer">
        <el-button @click="handleClose">关闭</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { ElDialog, ElInput, ElButton, ElIcon, ElAvatar, ElEmpty } from 'element-plus'
import { Search, Loading, Check } from '@element-plus/icons-vue'
import { searchUsersFuzzy } from '@/api/user'
import type { UserInfo } from '@/types'
import { formatUserDisplayName } from '@/utils/userInfo'
import { Logger } from '@/core/utils/logger'

interface Props {
  modelValue: boolean
  title?: string
  placeholder?: string
  initialUsername?: string | null
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'confirm', user: UserInfo | null): void
}

const props = withDefaults(defineProps<Props>(), {
  title: '选择用户',
  placeholder: '请输入用户名或邮箱搜索',
  initialUsername: null
})

const emit = defineEmits<Emits>()

const dialogVisible = ref(false)
const searchKeyword = ref('')
const loading = ref(false)
const userList = ref<UserInfo[]>([])
const selectedUser = ref<UserInfo | null>(null)
const inputRef = ref<InstanceType<typeof ElInput> | null>(null)

// 监听 modelValue 变化，控制弹窗显示
watch(() => props.modelValue, (newValue) => {
  dialogVisible.value = newValue
  if (newValue) {
    // 🔥 弹窗打开时，清空搜索框，让用户可以直接输入搜索
    searchKeyword.value = ''
    userList.value = []
    selectedUser.value = null
  }
})

// 处理弹窗打开完成事件（动画结束后）
const handleDialogOpened = async () => {
  // 🔥 等待 DOM 完全渲染后聚焦
  await nextTick()
  await nextTick() // 再等待一个 tick，确保 el-dialog 动画完成
  
  if (inputRef.value) {
    // Element Plus 的 ElInput 组件，通过 $el 访问 DOM 元素
    const inputEl = (inputRef.value as any).$el?.querySelector('input') as HTMLInputElement
    if (inputEl) {
      inputEl.focus()
      // 如果还是没聚焦，再试一次
      setTimeout(() => {
        inputEl.focus()
      }, 100)
    }
  }
}

// 监听 dialogVisible 变化，同步到 modelValue
watch(dialogVisible, (newValue) => {
  emit('update:modelValue', newValue)
})

// 搜索用户（防抖）
let searchTimer: ReturnType<typeof setTimeout> | null = null
const handleSearch = (keyword: string) => {
  if (searchTimer) {
    clearTimeout(searchTimer)
  }

  searchTimer = setTimeout(async () => {
    if (!keyword || keyword.trim() === '') {
      userList.value = []
      selectedUser.value = null
      return
    }

    try {
      loading.value = true
      const response = await searchUsersFuzzy(keyword.trim(), 20)
      Logger.debug('UserSearchDialog', '搜索用户响应', { keyword, response })
      
      // 提取 users 数组
      let users: UserInfo[] = []
      if (response && typeof response === 'object') {
        if (Array.isArray(response)) {
          users = response
        } else if (response.users && Array.isArray(response.users)) {
          users = response.users
        } else if (response.data && response.data.users && Array.isArray(response.data.users)) {
          users = response.data.users
        }
      }
      
      userList.value = users
      
      // 如果有初始用户名，自动选中匹配的用户
      if (props.initialUsername && users.length > 0) {
        const matchedUser = users.find(u => u.username === props.initialUsername)
        if (matchedUser) {
          selectedUser.value = matchedUser
        }
      }
    } catch (error) {
      Logger.error('UserSearchDialog', '搜索用户失败', { keyword, error })
      userList.value = []
    } finally {
      loading.value = false
    }
  }, 300) // 300ms 防抖
}

// 清空搜索
const handleClearSearch = () => {
  searchKeyword.value = ''
  userList.value = []
  selectedUser.value = null
}

// 选择用户（选中后自动确认并关闭弹窗）
const handleSelectUser = (user: UserInfo) => {
  selectedUser.value = user
  // 🔥 选中后自动确认并关闭弹窗
  emit('confirm', user)
  handleClose()
}

// 确认选择（保留此方法，以防其他地方调用）
const handleConfirm = () => {
  if (selectedUser.value) {
    emit('confirm', selectedUser.value)
    handleClose()
  }
}

// 关闭弹窗
const handleClose = () => {
  dialogVisible.value = false
  searchKeyword.value = ''
  userList.value = []
  selectedUser.value = null
}
</script>

<style scoped>
.user-search-dialog-search {
  margin-bottom: 20px;
}

.user-search-dialog-list {
  min-height: 300px;
  max-height: 400px;
  overflow-y: auto;
}

.user-search-dialog-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 40px;
  color: var(--el-text-color-secondary);
}

.user-search-dialog-empty {
  padding: 40px 0;
}

.user-search-dialog-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.user-search-dialog-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.user-search-dialog-item:hover {
  border-color: var(--el-color-primary-light-3);
  background-color: var(--el-fill-color-light);
}

.user-search-dialog-item.is-selected {
  border-color: var(--el-color-primary);
  border-width: 2px;
  background-color: transparent;
}

.user-avatar {
  flex-shrink: 0;
  width: 32px !important;
  height: 32px !important;
}

.user-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.user-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.user-nickname {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}

.user-email {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}

.user-signature {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.4;
}

.selected-icon {
  flex-shrink: 0;
  color: var(--el-color-primary);
  font-size: 20px;
}

.user-search-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>

