<!--
  UserSearchInput - 用户搜索选择组件
  用于搜索框中的用户选择，支持单选和多选
  显示效果与表格中的 user-cell 一致
-->
<template>
  <div class="user-search-input">
    <!-- 输入框容器（包含已选中的用户和输入框） -->
    <div class="user-search-input-wrapper" @click="handleWrapperClick">
      <!-- 已选中的用户（显示在输入框内部左侧） -->
      <div v-if="selectedUsers.length > 0" class="selected-users-inline">
        <div
          v-for="user in selectedUsers"
          :key="user.username"
          class="user-cell-inline"
        >
          <el-avatar :src="user.avatar" :size="20" class="user-avatar">
            {{ user.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <span class="user-name">{{ formatUserDisplayName(user) }}</span>
          <el-icon class="remove-icon" @click.stop="handleRemoveUser(user.username)">
            <Close />
          </el-icon>
        </div>
      </div>
      
      <!-- 输入框（flex: 1，占据剩余空间） -->
      <div class="input-wrapper">
        <el-input
          ref="inputRef"
          v-model="searchKeyword"
          :placeholder="selectedUsers.length > 0 ? '' : placeholder"
          :clearable="true"
          :loading="loading"
          class="user-search-input-field"
          @input="handleSearch"
          @clear="handleClearInput"
          @focus="handleFocus"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
    </div>

    <!-- 下拉选项列表 -->
    <div
      v-if="showDropdown && filteredUsers.length > 0"
      class="user-search-dropdown"
    >
      <div
        v-for="user in filteredUsers"
        :key="user.username"
        class="user-search-option"
        :class="{ 'is-selected': isSelected(user.username) }"
        @click="handleSelectUser(user)"
      >
        <el-avatar :src="user.avatar" :size="24" class="user-avatar">
          {{ user.username?.[0]?.toUpperCase() || 'U' }}
        </el-avatar>
        <div class="user-info">
          <span class="user-name">{{ user.username }}</span>
          <span v-if="user.nickname" class="user-nickname">({{ user.nickname }})</span>
        </div>
        <el-icon v-if="isSelected(user.username)" class="selected-icon">
          <Check />
        </el-icon>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { ElInput, ElAvatar, ElIcon } from 'element-plus'
import { Search, Check, Close } from '@element-plus/icons-vue'
import { searchUsersFuzzy } from '@/api/user'
import { useUserInfoStore } from '@/stores/userInfo'
import type { UserInfo } from '@/types'
import {
  createPlaceholderUser,
  formatUserDisplayName,
  parseUsernamesFromModelValue,
  isUsernameListEqual,
  extractUsernames,
  buildUserMap,
  reorderUsersByUsernames
} from '@/utils/userInfo'

interface Props {
  modelValue: string | string[] | null
  placeholder?: string
  multiple?: boolean
}

interface Emits {
  (e: 'update:modelValue', value: string | string[] | null): void
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: '搜索用户',
  multiple: false
})

const emit = defineEmits<Emits>()

// 用户信息 store
const userInfoStore = useUserInfoStore()

// 搜索关键词
const searchKeyword = ref('')
// 加载状态
const loading = ref(false)
// 所有用户选项（搜索结果）
const allUsers = ref<UserInfo[]>([])
// 下拉框显示状态
const showDropdown = ref(false)
// 已选中的用户列表
const selectedUsers = ref<UserInfo[]>([])
// 输入框引用
const inputRef = ref<InstanceType<typeof ElInput> | null>(null)

// 过滤后的用户列表（排除已选中的）
const filteredUsers = computed(() => {
  if (!props.multiple) {
    return allUsers.value
  }
  const selectedUsernames = selectedUsers.value.map(u => u.username)
  const filtered = allUsers.value.filter(user => !selectedUsernames.includes(user.username))
  // 🔥 如果过滤后还有结果，保持下拉框显示
  if (filtered.length > 0 && allUsers.value.length > 0) {
    showDropdown.value = true
  }
  return filtered
})

// 判断用户是否已选中
const isSelected = (username: string): boolean => {
  if (props.multiple) {
    return selectedUsers.value.some(u => u.username === username)
  }
  return selectedUsers.value.length > 0 && selectedUsers.value[0]?.username === username
}

// 搜索用户
const handleSearch = async (keyword: string) => {
  if (!keyword || keyword.trim() === '') {
    // 🔥 如果关键字为空，保持下拉框显示（如果之前有搜索结果且还有未选中的用户）
    if (allUsers.value.length === 0) {
      showDropdown.value = false
    } else {
      // 检查是否还有未选中的用户
      const hasUnselected = props.multiple 
        ? allUsers.value.some(user => !selectedUsers.value.some(su => su.username === user.username))
        : allUsers.value.length > 0
      showDropdown.value = hasUnselected
    }
    return
  }

  loading.value = true
  showDropdown.value = true

  try {
    const response = await searchUsersFuzzy(keyword.trim(), 20)
    allUsers.value = response.users || []
    // 🔥 搜索后，如果有结果，保持下拉框显示
    if (allUsers.value.length > 0) {
      showDropdown.value = true
    } else {
      showDropdown.value = false
    }
  } catch (error) {
    console.error('[UserSearchInput] 搜索用户失败', error)
    allUsers.value = []
    showDropdown.value = false
  } finally {
    loading.value = false
  }
}

// 选择用户
const handleSelectUser = (user: UserInfo) => {
  if (props.multiple) {
    // 多选模式
    const index = selectedUsers.value.findIndex(u => u.username === user.username)
    if (index >= 0) {
      // 已选中，取消选择
      selectedUsers.value.splice(index, 1)
    } else {
      // 未选中，添加到选中列表
      selectedUsers.value.push(user)
    }
    // 🔥 标记为内部更新，避免 watch 触发时覆盖
    isInternalUpdate.value = true
    // 更新 modelValue（确保是数组格式）
    const usernames = extractUsernames(selectedUsers.value)
    // console.log('[UserSearchInput] handleSelectUser 更新 modelValue:', usernames) // 🔥 调试日志已注释
    emit('update:modelValue', props.multiple ? usernames : (usernames[0] ?? null))
    // 🔥 重置内部更新标记（延迟一点，确保 watch 不会触发）
    setTimeout(() => {
      isInternalUpdate.value = false
    }, 100)
    // 🔥 多选模式下清空搜索关键字，但保持下拉框显示（如果还有未选中的用户）
    searchKeyword.value = ''
    // 不清空 allUsers，保持下拉框可以继续选择
    // 🔥 检查是否还有未选中的用户，如果有则保持下拉框显示
    const hasUnselected = allUsers.value.some(u => !selectedUsers.value.some(su => su.username === u.username))
    showDropdown.value = hasUnselected && allUsers.value.length > 0
  } else {
    // 单选模式
    selectedUsers.value = [user]
    isInternalUpdate.value = true
    emit('update:modelValue', user.username)
    nextTick(() => {
      isInternalUpdate.value = false
    })
    showDropdown.value = false
    searchKeyword.value = ''
    allUsers.value = []
  }
}

// 移除用户
const handleRemoveUser = (username: string) => {
  const index = selectedUsers.value.findIndex(u => u.username === username)
  if (index >= 0) {
    selectedUsers.value.splice(index, 1)
    if (props.multiple) {
      emit('update:modelValue', extractUsernames(selectedUsers.value))
    } else {
      emit('update:modelValue', null)
    }
  }
}

// 清空输入
const handleClearInput = () => {
  searchKeyword.value = ''
  allUsers.value = []
  showDropdown.value = false
}

// 聚焦时显示下拉框（如果有搜索结果）
const handleFocus = () => {
  if (allUsers.value.length > 0) {
    showDropdown.value = true
  }
}

// 点击容器时聚焦输入框
const handleWrapperClick = () => {
  // 🔥 使用 ref 引用当前组件实例的输入框，而不是全局查询
  // 这样可以避免在有多个 UserSearchInput 组件时，焦点跳转到第一个组件
  nextTick(() => {
    if (inputRef.value) {
      // Element Plus 的 ElInput 组件，通过 $el 访问 DOM 元素
      const inputEl = (inputRef.value as any).$el?.querySelector('input') as HTMLInputElement
      if (inputEl) {
        inputEl.focus()
  }
    }
  })
}

// 点击外部关闭下拉框
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  const component = document.querySelector('.user-search-input')
  if (component && !component.contains(target)) {
    showDropdown.value = false
  }
}

// 是否正在内部更新（避免 watch 触发时覆盖用户选择）
const isInternalUpdate = ref(false)

// 初始化已选中的用户（用于回显）
const initSelectedUsers = async () => {
  // 🔥 如果是内部更新，不需要重新加载
  if (isInternalUpdate.value) {
    return
  }

  // 🔥 使用工具函数解析用户名
  const usernames = [...new Set(parseUsernamesFromModelValue(props.modelValue, props.multiple))]

  if (usernames.length === 0) {
    selectedUsers.value = []
    return
  }
  
  // console.log('[UserSearchInput] initSelectedUsers usernames:', usernames) // 🔥 调试日志已注释

  // 🔥 检查当前 selectedUsers 是否已经包含了所有需要的用户（按顺序）
  const currentUsernames = extractUsernames(selectedUsers.value)
  
  // 如果用户名列表相同（顺序和内容），直接返回
  if (isUsernameListEqual(usernames, currentUsernames)) {
    // console.log('[UserSearchInput] initSelectedUsers 无需更新') // 🔥 调试日志已注释
    return
  }

  try {
    // 🔥 只加载缺失的用户
    const missingUsernames = usernames.filter(u => !currentUsernames.includes(u))
    // console.log('[UserSearchInput] initSelectedUsers missingUsernames:', missingUsernames) // 🔥 调试日志已注释
    
    // 🔥 使用 store 批量查询（自动处理缓存和过期）
    if (missingUsernames.length > 0) {
      // console.log('[UserSearchInput] 查询缺失的用户信息:', missingUsernames) // 🔥 调试日志已注释
      const loadedUsers = await userInfoStore.batchGetUserInfo(missingUsernames)
      // console.log('[UserSearchInput] 查询完成，获取到', loadedUsers.length, '个用户') // 🔥 调试日志已注释
      
      // 🔥 合并已有用户和新加载的用户
      const userMap = buildUserMap([...selectedUsers.value, ...loadedUsers])
      
      // 🔥 按照 usernames 的顺序重新组织 selectedUsers
      selectedUsers.value = reorderUsersByUsernames(usernames, userMap, true)
      
      // console.log('[UserSearchInput] initSelectedUsers 最终 selectedUsers:', extractUsernames(selectedUsers.value)) // 🔥 调试日志已注释
    } else {
      // 🔥 如果没有缺失的用户，只需要移除和重排序
      const userMap = buildUserMap(selectedUsers.value)
      selectedUsers.value = reorderUsersByUsernames(usernames, userMap, true)
    }
  } catch (error) {
    console.error('[UserSearchInput] 加载用户信息失败', error)
    // 如果加载失败，至少保持已有的用户，但需要移除不在 usernames 中的
    selectedUsers.value = selectedUsers.value.filter(u => usernames.includes(u.username))
  }
}

// 监听 modelValue 变化
watch(() => props.modelValue, async (newValue, oldValue) => {
  // 🔥 如果是内部更新，不需要重新加载
  if (isInternalUpdate.value) {
    // console.log('[UserSearchInput] watch 跳过内部更新') // 🔥 调试日志已注释
    return
  }
  
  // 🔥 只有当值真正变化时才重新加载
  const newValueStr = JSON.stringify(newValue)
  const oldValueStr = JSON.stringify(oldValue)
  if (newValueStr !== oldValueStr) {
    // console.log('[UserSearchInput] watch modelValue 变化:', { // 🔥 调试日志已注释
    //   oldValue,
    //   newValue,
    //   oldValueStr,
    //   newValueStr
    // })
    // 🔥 延迟初始化，等待 TableRenderer 的批量查询完成（如果存在）
    // 这样可以避免重复查询，因为 TableRenderer 会统一收集所有用户并批量查询
    // 使用 nextTick 确保在下一个事件循环中执行，给 TableRenderer 的 batchLoadUserInfo 优先执行的机会
    await nextTick()
    await nextTick() // 再延迟一个 tick，确保 TableRenderer 有机会先执行
    initSelectedUsers()
  }
}, { immediate: true, deep: true })

// 组件挂载时
onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  // 🔥 不需要在这里调用 initSelectedUsers()，因为 watch 已经设置了 immediate: true，会在初始化时自动触发
})

// 组件卸载时
onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.user-search-input {
  position: relative;
  width: 100%;
}

.user-search-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  min-height: 32px;
  padding: 2px 8px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background-color: var(--el-bg-color);
  cursor: text;
  transition: border-color 0.2s;
  flex-wrap: wrap;
  gap: 4px;
}

.user-search-input-wrapper:hover {
  border-color: var(--el-border-color-hover);
}

.user-search-input-wrapper:focus-within {
  border-color: var(--el-color-primary);
}

.selected-users-inline {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
  flex: 0 0 auto;
}

.user-cell-inline {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  background-color: var(--el-fill-color-light);
  border-radius: 4px;
  height: 24px;
  flex-shrink: 0;
}

.user-cell-inline .user-avatar {
  flex-shrink: 0;
  width: 20px !important;
  height: 20px !important;
}

.user-cell-inline .user-name {
  font-size: 12px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  line-height: 20px;
}

.user-cell-inline .remove-icon {
  flex-shrink: 0;
  width: 14px;
  height: 14px;
  cursor: pointer;
  color: var(--el-text-color-secondary);
  transition: color 0.2s;
  margin-left: 2px;
}

.user-cell-inline .remove-icon:hover {
  color: var(--el-text-color-primary);
}

.input-wrapper {
  flex: 1;
  min-width: 120px;
}

.user-search-input-field {
  width: 100%;
}

.user-search-input-field :deep(.el-input__wrapper) {
  box-shadow: none !important;
  border: none !important;
  padding: 0 !important;
  background-color: transparent !important;
}

.user-search-input-field :deep(.el-input__inner) {
  height: 28px;
  line-height: 28px;
  padding: 0 !important;
}

/* 下拉选项列表 */
.user-search-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  margin-top: 4px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  box-shadow: var(--el-box-shadow-light);
  max-height: 300px;
  overflow-y: auto;
}

.user-search-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.user-search-option:hover {
  background-color: var(--el-fill-color-light);
}

.user-search-option.is-selected {
  background-color: var(--el-color-primary-light-9);
}

.user-info {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 4px;
}

.user-name {
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.user-nickname {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.selected-icon {
  color: var(--el-color-primary);
  font-size: 16px;
}
</style>
