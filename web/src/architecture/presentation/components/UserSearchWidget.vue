<!--
  UserSearchWidget - 用户搜索专用组件（用于搜索表单）
  功能：
  - 专门为搜索场景设计，直接处理原始值格式（string | string[] | null）
  - 使用弹窗搜索，体验更好，可以展示更多信息
  - 支持单选和多选（根据 search 类型自动判断）
-->
<template>
  <div class="user-search-widget">
    <!-- 选中后的显示（多选模式） -->
    <div
      v-if="supportsMultiple && selectedUsers.length > 0"
      class="user-search-display"
      @click="handleOpenDialog()"
    >
      <div class="selected-users-list">
        <div
          v-for="user in selectedUsers"
          :key="user.username"
          class="selected-user-tag"
        >
          <el-avatar 
            v-if="user.avatar" 
            :src="user.avatar" 
            :size="20" 
            class="user-avatar-small"
          >
            {{ user.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <el-avatar 
            v-else
            :size="20" 
            class="user-avatar-small"
          >
            {{ user.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <span class="user-display-text">
            {{ formatUserDisplayName(user) }}
          </span>
          <el-icon class="remove-icon" @click.stop="handleRemoveUser(user)">
            <Close />
          </el-icon>
        </div>
      </div>
      <el-icon class="edit-icon">
        <Edit />
      </el-icon>
    </div>
    
    <!-- 选中后的显示（单选模式） -->
    <div
      v-else-if="!supportsMultiple && selectedUser"
      class="user-search-display"
      @click="handleOpenDialog()"
    >
      <el-avatar 
        v-if="selectedUser.avatar" 
        :src="selectedUser.avatar" 
        :size="24" 
        class="user-avatar-small"
      >
        {{ selectedUser.username?.[0]?.toUpperCase() || 'U' }}
      </el-avatar>
      <el-avatar 
        v-else
        :size="24" 
        class="user-avatar-small"
      >
        {{ selectedUser.username?.[0]?.toUpperCase() || 'U' }}
      </el-avatar>
      <span class="user-display-text">
        {{ formatUserDisplayName(selectedUser) }}
      </span>
      <el-icon class="edit-icon">
        <Edit />
      </el-icon>
    </div>
    <!-- 未选中时显示按钮 -->
    <el-button
      v-else
      class="search-trigger-button"
      @click="handleOpenDialog()"
    >
      <el-icon><User /></el-icon>
      <span class="search-trigger-text">{{ field.desc || `请选择${field.name}` }}</span>
    </el-button>
    
    <!-- 多用户搜索弹窗（支持 IN 查询时使用） -->
    <UsersSearchDialog
      v-if="supportsMultiple"
      v-model="dialogVisible"
      :title="`选择${field.name || '用户'}`"
      :placeholder="field.desc || '请输入用户名或邮箱搜索'"
      :initial-usernames="normalizedModelValue"
      @confirm="handleUsersSelected"
    />
    
    <!-- 单用户搜索弹窗（EQ 查询时使用） -->
    <UserSearchDialog
      v-else
      v-model="dialogVisible"
      :title="`选择${field.name || '用户'}`"
      :placeholder="field.desc || '请输入用户名或邮箱搜索'"
      :initial-username="typeof modelValue === 'string' ? modelValue : null"
      @confirm="handleUserSelected"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElButton, ElIcon, ElAvatar } from 'element-plus'
import { User, Edit, Close } from '@element-plus/icons-vue'
import UserSearchDialog from '@/shared/components/UserSearchDialog.vue'
import UsersSearchDialog from '@/shared/components/UsersSearchDialog.vue'
import { useUserInfoStore } from '@/stores/userInfo'
import type { UserInfo } from '@/types'
import { formatUserDisplayName } from '@/utils/userInfo'
import type { FieldConfig } from '@/core/types/field'
import { Logger } from '@/core/utils/logger'
import { SearchType, hasSearchType } from '@/core/constants/search'

interface Props {
  field: FieldConfig
  modelValue: string | string[] | null  // 搜索表单的原始值格式（逗号分隔的字符串或数组）
  searchType?: string  // 搜索类型，用于判断单选还是多选
}

interface Emits {
  (e: 'update:modelValue', value: string | string[] | null): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const userInfoStore = useUserInfoStore()
const dialogVisible = ref(false)
const selectedUsers = ref<UserInfo[]>([])
const selectedUser = ref<UserInfo | null>(null)

// 是否支持多选（根据 search 类型判断）
const supportsMultiple = computed(() => {
  const searchType = props.searchType || ''
  return hasSearchType(searchType, SearchType.IN)
})

// 规范化 modelValue 为字符串格式（用于传递给对话框）
const normalizedModelValue = computed(() => {
  if (!props.modelValue) return null
  if (Array.isArray(props.modelValue)) {
    return props.modelValue.map(v => String(v).trim()).filter(v => v).join(',') || null
  }
  return String(props.modelValue).trim() || null
})

// 处理打开弹窗
function handleOpenDialog(): void {
  dialogVisible.value = true
}

// 处理用户选择（多选）
function handleUsersSelected(users: UserInfo[]): void {
  // 直接返回原始值格式（逗号分隔的字符串）
  const usernames = users.map(u => u.username).join(',')
  selectedUsers.value = users
  emit('update:modelValue', usernames || null)
}

// 处理用户选择（单选）
function handleUserSelected(user: UserInfo | null): void {
  selectedUser.value = user
  emit('update:modelValue', user ? user.username : null)
}

// 移除单个用户
function handleRemoveUser(user: UserInfo): void {
  const currentUsernames = getUsernamesFromValue(props.modelValue)
  const newUsernames = currentUsernames.filter(u => u !== user.username)
  selectedUsers.value = selectedUsers.value.filter(u => u.username !== user.username)
  emit('update:modelValue', newUsernames.length > 0 ? newUsernames.join(',') : null)
}

// 从值中提取用户名列表
function getUsernamesFromValue(value: string | string[] | null): string[] {
  if (!value) return []
  if (Array.isArray(value)) {
    return value.map(v => String(v).trim()).filter(v => v)
  }
  return String(value).split(',').map(u => u.trim()).filter(u => u)
}

// 加载已选用户信息
async function loadSelectedUsers(): Promise<void> {
  if (supportsMultiple.value) {
    // 多选模式
    const usernames = getUsernamesFromValue(props.modelValue)
    if (usernames.length === 0) {
      selectedUsers.value = []
      return
    }
    
    try {
      // 从 store 批量获取用户信息
      const users: UserInfo[] = []
      for (const username of usernames) {
        try {
          const user = await userInfoStore.getUserInfo(username)
          if (user) {
            users.push(user)
          }
        } catch (error) {
          Logger.error('UserSearchWidget', '加载用户信息失败', { username, error })
        }
      }
      selectedUsers.value = users
    } catch (error) {
      Logger.error('UserSearchWidget', '加载已选用户信息失败', { error })
      selectedUsers.value = []
    }
  } else {
    // 单选模式
    const username = props.modelValue ? String(props.modelValue).trim() : null
    if (!username) {
      selectedUser.value = null
      return
    }
    
    try {
      const user = await userInfoStore.getUserInfo(username)
      selectedUser.value = user || null
    } catch (error) {
      Logger.error('UserSearchWidget', '加载用户信息失败', { username, error })
      selectedUser.value = null
    }
  }
}

// 监听值变化，加载用户信息
watch(() => [props.modelValue, supportsMultiple], () => {
  loadSelectedUsers()
}, { immediate: true })
</script>

<style scoped>
.user-search-widget {
  width: 100%;
}

.search-trigger-button {
  width: 100%;
  min-height: 32px;
  justify-content: flex-start;
  padding: 0 12px;
}

.search-trigger-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-search-display {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 10px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  background-color: var(--el-bg-color);
  cursor: pointer;
  transition: all 0.2s;
  min-height: 32px;
}

.user-search-display .user-avatar-small {
  flex-shrink: 0;
}

.user-search-display .user-display-text {
  flex: 1;
  font-size: 13px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-search-display:hover {
  border-color: var(--el-color-primary);
  background-color: var(--el-fill-color-light);
}

.selected-users-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  flex: 1;
  align-items: center;
}

.selected-user-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  background-color: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  font-size: 12px;
}

.user-avatar-small {
  flex-shrink: 0;
}

.user-display-text {
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

.remove-icon {
  cursor: pointer;
  color: var(--el-text-color-secondary);
  font-size: 14px;
  margin-left: 4px;
  transition: color 0.2s;
}

.remove-icon:hover {
  color: var(--el-color-primary);
}

.edit-icon {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
  font-size: 16px;
  transition: color 0.2s;
}

.user-search-display:hover .edit-icon {
  color: var(--el-color-primary);
}
</style>
