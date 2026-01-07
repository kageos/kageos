<!--
  UsersSearchDialog - 多用户搜索弹窗组件
  功能：
  - 弹窗式用户搜索和选择
  - 支持多选模式
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
    <div class="users-search-dialog-search">
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

    <!-- 已选用户列表 -->
    <div v-if="selectedUsers.length > 0" class="users-search-dialog-selected">
      <div class="selected-header">
        <span>已选择 ({{ selectedUsers.length }}{{ maxCount > 0 ? `/${maxCount}` : '' }})</span>
        <el-button type="text" size="small" @click="handleClearAll">清空</el-button>
      </div>
      <div class="selected-users">
        <div
          v-for="user in selectedUsers"
          :key="user.username"
          class="selected-user-item"
        >
          <el-avatar :src="user.avatar" :size="24" class="user-avatar">
            {{ user.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <span class="user-name">{{ formatUserDisplayName(user) }}</span>
          <el-icon class="remove-icon" @click="handleRemoveUser(user)">
            <Close />
          </el-icon>
        </div>
      </div>
    </div>

    <!-- 用户列表 -->
    <div class="users-search-dialog-list">
      <div
        v-if="loading"
        class="users-search-dialog-loading"
      >
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>搜索中...</span>
      </div>
      <div
        v-else-if="userList.length === 0 && searchKeyword"
        class="users-search-dialog-empty"
      >
        <el-empty description="未找到用户" :image-size="80" />
      </div>
      <div
        v-else-if="userList.length === 0 && !searchKeyword"
        class="users-search-dialog-empty"
      >
        <el-empty description="请输入关键词搜索用户" :image-size="80" />
      </div>
      <div
        v-else
        class="users-search-dialog-items"
      >
        <div
          v-for="user in userList"
          :key="user.username"
          class="users-search-dialog-item"
          :class="{ 'is-selected': isUserSelected(user) }"
          @click="handleToggleUser(user)"
        >
          <el-checkbox
            :model-value="isUserSelected(user)"
            @change="handleToggleUser(user)"
            @click.stop
          />
          <el-avatar :src="user.avatar" :size="40" class="user-avatar">
            {{ user.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <div class="user-info">
            <div class="user-name">{{ user.username }}</div>
            <div v-if="user.nickname" class="user-nickname">{{ user.nickname }}</div>
            <div v-if="user.email" class="user-email">{{ user.email }}</div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="users-search-dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" @click="handleConfirm">确认</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { ElDialog, ElInput, ElButton, ElIcon, ElAvatar, ElEmpty, ElCheckbox } from 'element-plus'
import { Search, Loading, Close } from '@element-plus/icons-vue'
import { searchUsersFuzzy } from '@/api/user'
import type { UserInfo } from '@/types'
import { formatUserDisplayName } from '@/utils/userInfo'
import { Logger } from '@/core/utils/logger'

interface Props {
  modelValue: boolean
  title?: string
  placeholder?: string
  initialUsernames?: string | null // 逗号分隔的用户名列表
  maxCount?: number // 最大选择数量，0表示不限制
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'confirm', users: UserInfo[]): void
}

const props = withDefaults(defineProps<Props>(), {
  title: '选择用户',
  placeholder: '请输入用户名或邮箱搜索',
  initialUsernames: null,
  maxCount: 0
})

const emit = defineEmits<Emits>()

const dialogVisible = ref(false)
const searchKeyword = ref('')
const loading = ref(false)
const userList = ref<UserInfo[]>([])
const selectedUsers = ref<UserInfo[]>([])
const inputRef = ref<InstanceType<typeof ElInput> | null>(null)

// 监听 modelValue 变化，控制弹窗显示
watch(() => props.modelValue, async (newValue) => {
  dialogVisible.value = newValue
  if (newValue) {
    // 弹窗打开时，初始化已选用户
    if (props.initialUsernames) {
      const usernames = props.initialUsernames.split(',').map(u => u.trim()).filter(u => u)
      // 🔥 加载已选用户的信息
      if (usernames.length > 0) {
        try {
          const { useUserInfoStore } = await import('@/stores/userInfo')
          const userInfoStore = useUserInfoStore()
          const users: UserInfo[] = []
          
          // 并行加载所有用户信息
          await Promise.all(
            usernames.map(async (username) => {
              try {
                const user = await userInfoStore.getUserInfo(username)
                if (user) {
                  users.push(user)
                }
              } catch (error) {
                Logger.error('UsersSearchDialog', '加载用户信息失败', { username, error })
              }
            })
          )
          
          selectedUsers.value = users
        } catch (error) {
          Logger.error('UsersSearchDialog', '加载已选用户信息失败', { error })
          selectedUsers.value = []
        }
      } else {
        selectedUsers.value = []
      }
    } else {
      selectedUsers.value = []
    }
    // ⭐ 注意：不要在这里清空 searchKeyword，让 handleDialogOpened 来处理自动搜索
    // searchKeyword.value = ''
    userList.value = []
  } else {
    // 弹窗关闭时，清空搜索关键词
    searchKeyword.value = ''
    userList.value = []
  }
})

// 处理弹窗打开完成事件（动画结束后）
const handleDialogOpened = async () => {
  await nextTick()
  await nextTick()
  
  if (inputRef.value) {
    const inputEl = (inputRef.value as any).$el?.querySelector('input') as HTMLInputElement
    if (inputEl) {
      inputEl.focus()
      setTimeout(() => {
        inputEl.focus()
      }, 100)
    }
  }
  
  // 🔥 参考 UserSearchDialog：弹窗打开时，如果有初始用户名，自动搜索这些用户名的第一个字符
  // 这样用户可以看到相关的用户列表，而不是显示"请输入关键词搜索用户"
  // ⭐ 使用 setTimeout 确保在弹窗完全打开后再执行搜索
  setTimeout(() => {
    if (props.initialUsernames) {
      const usernames = props.initialUsernames.split(',').map(u => u.trim()).filter(u => u)
      if (usernames.length > 0 && usernames[0]) {
        // 使用第一个用户名的第一个字符进行搜索
        const firstChar = usernames[0][0]
        if (firstChar) {
          searchKeyword.value = firstChar
          // 直接调用 handleSearch，不需要等待防抖
          handleSearch(firstChar)
        }
      }
    }
  }, 200) // 等待弹窗动画完成
}

// 监听 dialogVisible 变化，同步到 modelValue
watch(dialogVisible, (newValue) => {
  emit('update:modelValue', newValue)
})

// 判断用户是否已选中
const isUserSelected = (user: UserInfo): boolean => {
  return selectedUsers.value.some(u => u.username === user.username)
}

// 切换用户选择状态
const handleToggleUser = (user: UserInfo) => {
  if (isUserSelected(user)) {
    // 取消选择
    selectedUsers.value = selectedUsers.value.filter(u => u.username !== user.username)
  } else {
    // 检查是否超过最大数量
    if (props.maxCount > 0 && selectedUsers.value.length >= props.maxCount) {
      return
    }
    // 添加选择
    selectedUsers.value.push(user)
  }
}

// 移除已选用户
const handleRemoveUser = (user: UserInfo) => {
  selectedUsers.value = selectedUsers.value.filter(u => u.username !== user.username)
}

// 清空所有已选用户
const handleClearAll = () => {
  selectedUsers.value = []
}

// 搜索用户（防抖）
let searchTimer: ReturnType<typeof setTimeout> | null = null
const handleSearch = (keyword: string) => {
  if (searchTimer) {
    clearTimeout(searchTimer)
  }

  searchTimer = setTimeout(async () => {
    if (!keyword || keyword.trim() === '') {
      userList.value = []
      return
    }

    try {
      loading.value = true
      const response = await searchUsersFuzzy(keyword.trim(), 20)
      Logger.debug('UsersSearchDialog', '搜索用户响应', { keyword, response })
      
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
      
      // 🔥 如果有初始用户名，自动选中匹配的用户（参考 UserSearchDialog）
      if (props.initialUsernames && users.length > 0) {
        const initialUsernames = props.initialUsernames.split(',').map(u => u.trim()).filter(u => u)
        initialUsernames.forEach(username => {
          const matchedUser = users.find(u => u.username === username)
          if (matchedUser && !isUserSelected(matchedUser)) {
            // 检查是否超过最大数量
            if (props.maxCount > 0 && selectedUsers.value.length >= props.maxCount) {
              return
            }
            selectedUsers.value.push(matchedUser)
          }
        })
      }
    } catch (error) {
      Logger.error('UsersSearchDialog', '搜索用户失败', { keyword, error })
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
}

// 确认选择
const handleConfirm = () => {
  emit('confirm', [...selectedUsers.value])
  handleClose()
}

// 关闭弹窗
const handleClose = () => {
  dialogVisible.value = false
  searchKeyword.value = ''
  userList.value = []
  // 注意：不清空 selectedUsers，保留选择状态，以便下次打开时继续使用
}
</script>

<style scoped>
.users-search-dialog-search {
  margin-bottom: 20px;
}

.users-search-dialog-selected {
  margin-bottom: 20px;
  padding: 12px;
  background-color: var(--el-fill-color-lighter);
  border-radius: 6px;
}

.selected-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.selected-users {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.selected-user-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background-color: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}

.selected-user-item .user-avatar {
  flex-shrink: 0;
}

.selected-user-item .user-name {
  font-size: 12px;
  color: var(--el-text-color-primary);
}

.selected-user-item .remove-icon {
  cursor: pointer;
  color: var(--el-text-color-secondary);
  font-size: 14px;
  transition: color 0.2s;
}

.selected-user-item .remove-icon:hover {
  color: var(--el-color-danger);
}

.users-search-dialog-list {
  min-height: 300px;
  max-height: 400px;
  overflow-y: auto;
}

.users-search-dialog-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 40px;
  color: var(--el-text-color-secondary);
}

.users-search-dialog-empty {
  padding: 40px 0;
}

.users-search-dialog-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.users-search-dialog-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.users-search-dialog-item:hover {
  border-color: var(--el-color-primary-light-3);
  background-color: var(--el-fill-color-light);
}

.users-search-dialog-item.is-selected {
  border-color: var(--el-color-primary);
  background-color: var(--el-color-primary-light-9);
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

.users-search-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>

