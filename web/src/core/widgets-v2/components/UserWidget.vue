<!--
  UserWidget - 用户组件
  功能：
  - 输入场景（edit/search）：用户选择器，支持模糊搜索
  - 输出场景（response/table-cell/detail）：显示用户信息（头像、名称等）
-->

<template>
  <div class="user-widget">
    <!-- 编辑模式：用户选择器 -->
    <div v-if="mode === 'edit' || mode === 'search'" class="user-select-wrapper">
      <!-- 选中后的显示（自定义显示） -->
      <div v-if="selectedUserForDisplay" class="user-select-display">
        <el-avatar 
          v-if="selectedUserForDisplay.avatar" 
          :src="selectedUserForDisplay.avatar" 
          :size="20" 
          class="user-avatar-small"
        >
          {{ selectedUserForDisplay.username?.[0]?.toUpperCase() || 'U' }}
        </el-avatar>
        <el-avatar 
          v-else
          :size="20" 
          class="user-avatar-small"
        >
          {{ selectedUserForDisplay.username?.[0]?.toUpperCase() || 'U' }}
        </el-avatar>
        <span class="user-display-text">
          {{ selectedUserForDisplay.nickname ? `${selectedUserForDisplay.username}(${selectedUserForDisplay.nickname})` : selectedUserForDisplay.username }}
        </span>
      </div>
      <el-select
        v-model="internalValue"
        :disabled="field.widget?.config?.disabled"
        :placeholder="field.desc || `请选择${field.name}`"
        :clearable="true"
        :filterable="true"
        :loading="loading"
        :remote="true"
        :remote-method="handleRemoteSearch"
        popper-class="user-select-dropdown-popper"
        class="user-select-hidden-label"
        @change="handleChange"
        @focus="handleFocus"
      >
        <el-option
          v-for="user in userOptions"
          :key="user.username"
          :value="user.username"
          :label="user.nickname ? `${user.username}(${user.nickname})` : user.username"
        >
          <div class="user-option">
            <el-avatar :src="user.avatar" :size="24" class="user-avatar">
              {{ user.username?.[0]?.toUpperCase() || 'U' }}
            </el-avatar>
            <span class="user-name">{{ user.username }}</span>
            <span v-if="user.nickname" class="user-nickname">({{ user.nickname }})</span>
          </div>
        </el-option>
      </el-select>
    </div>
    
    <!-- 响应模式（点击头像显示卡片，点击名称复制） -->
    <span v-else-if="mode === 'response'" class="user-display">
      <el-popover
        placement="top"
        :width="280"
        :trigger="[]"
        popper-class="user-info-popover"
        :teleported="true"
        v-model:visible="showPopover"
        ref="popoverRef"
      >
        <template #reference>
          <el-avatar 
            v-if="userInfo" 
            :src="userInfo.avatar" 
            :size="24" 
            class="user-avatar user-avatar-clickable"
            @click="handleAvatarClick"
          >
            {{ userInfo.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <el-avatar 
            v-else 
            :size="24" 
            class="user-avatar user-avatar-clickable"
            @click="handleAvatarClick"
          >
            {{ displayName?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
        </template>
      <div v-if="userInfo" class="user-info-card">
        <div class="user-card-header">
          <el-avatar :src="userInfo.avatar" :size="48" class="user-avatar-large">
            {{ userInfo.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <div class="user-card-names">
            <div class="user-card-primary">{{ displayName }}</div>
            <div class="user-card-username">@{{ userInfo.username }}</div>
          </div>
        </div>
        <div class="user-card-content">
          <div v-if="userInfo.email" class="user-card-item">
            <span class="user-card-label">邮箱：</span>
            <span class="user-card-value">{{ userInfo.email }}</span>
          </div>
          <div v-if="userInfo.nickname" class="user-card-item">
            <span class="user-card-label">昵称：</span>
            <span class="user-card-value">{{ userInfo.nickname }}</span>
          </div>
          <div v-if="userInfo.signature" class="user-card-item">
            <span class="user-card-label">签名：</span>
            <span class="user-card-value user-card-signature">{{ userInfo.signature }}</span>
          </div>
          <div class="user-card-item">
            <span class="user-card-label">用户名：</span>
            <span class="user-card-value">{{ userInfo.username }}</span>
          </div>
        </div>
        <div class="user-card-footer">
          <el-button size="small" type="primary" @click="handleCopyUserInfo">点击复制</el-button>
        </div>
      </div>
      <div v-else class="user-info-card">
        <div class="user-card-content">
          <div class="user-card-item">
            <span class="user-card-label">用户名：</span>
            <span class="user-card-value">{{ displayName }}</span>
          </div>
        </div>
      </div>
      </el-popover>
      <span 
        class="user-name user-name-clickable" 
        @click="handleCopyName"
      >{{ displayName }}</span>
    </span>
    
    <!-- 表格单元格模式（点击头像显示卡片，点击名称复制） -->
    <span v-else-if="mode === 'table-cell'" class="user-cell">
      <el-popover
        placement="top"
        :width="280"
        :trigger="[]"
        popper-class="user-info-popover"
        :teleported="true"
        v-model:visible="showPopover"
        ref="popoverRef"
      >
        <template #reference>
          <el-avatar 
            v-if="userInfo" 
            :src="userInfo.avatar" 
            :size="24" 
            class="user-avatar user-avatar-clickable"
            @click="handleAvatarClick"
          >
            {{ userInfo.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <el-avatar 
            v-else 
            :size="24" 
            class="user-avatar user-avatar-clickable"
            @click="handleAvatarClick"
          >
            {{ displayName?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
        </template>
        <div v-if="userInfo" class="user-info-card">
          <div class="user-card-header">
            <el-avatar :src="userInfo.avatar" :size="48" class="user-avatar-large">
              {{ userInfo.username?.[0]?.toUpperCase() || 'U' }}
            </el-avatar>
            <div class="user-card-names">
              <div class="user-card-primary">{{ displayName }}</div>
              <div class="user-card-username">@{{ userInfo.username }}</div>
            </div>
          </div>
          <div class="user-card-content">
            <div v-if="userInfo.email" class="user-card-item">
              <span class="user-card-label">邮箱：</span>
              <span class="user-card-value">{{ userInfo.email }}</span>
            </div>
            <div v-if="userInfo.nickname" class="user-card-item">
              <span class="user-card-label">昵称：</span>
              <span class="user-card-value">{{ userInfo.nickname }}</span>
            </div>
            <div class="user-card-item">
              <span class="user-card-label">用户名：</span>
              <span class="user-card-value">{{ userInfo.username }}</span>
            </div>
          </div>
          <div class="user-card-footer">
            <el-button size="small" type="primary" @click="handleCopyUserInfo">点击复制</el-button>
          </div>
        </div>
        <div v-else class="user-info-card">
          <div class="user-card-content">
            <div class="user-card-item">
              <span class="user-card-label">用户名：</span>
              <span class="user-card-value">{{ displayName }}</span>
            </div>
        </div>
      </div>
      </el-popover>
      <span 
        class="user-name user-name-clickable" 
        @click="handleCopyName"
      >{{ displayName }}</span>
    </span>
    
    <!-- 详情模式（点击头像显示卡片，点击名称复制） -->
    <div v-else-if="mode === 'detail'" class="user-detail">
      <el-popover
        placement="top"
        :width="280"
        :trigger="[]"
        popper-class="user-info-popover"
        :teleported="true"
        v-model:visible="showPopover"
        ref="popoverRef"
      >
        <template #reference>
          <el-avatar 
            v-if="userInfo" 
            :src="userInfo.avatar" 
            :size="48" 
            class="user-avatar-large user-avatar-clickable"
            @click.stop="handleAvatarClick"
          >
            {{ userInfo.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <el-avatar 
            v-else 
            :size="48" 
            class="user-avatar-large user-avatar-clickable"
            @click.stop="handleAvatarClick"
          >
            {{ displayName?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
        </template>
      <div v-if="userInfo" class="user-info-card">
        <div class="user-card-header">
          <el-avatar :src="userInfo.avatar" :size="48" class="user-avatar-large">
            {{ userInfo.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <div class="user-card-names">
            <div class="user-card-primary">{{ displayName }}</div>
            <div class="user-card-username">@{{ userInfo.username }}</div>
          </div>
        </div>
        <div class="user-card-content">
          <div v-if="userInfo.email" class="user-card-item">
            <span class="user-card-label">邮箱：</span>
            <span class="user-card-value">{{ userInfo.email }}</span>
          </div>
          <div v-if="userInfo.nickname" class="user-card-item">
            <span class="user-card-label">昵称：</span>
            <span class="user-card-value">{{ userInfo.nickname }}</span>
          </div>
          <div v-if="userInfo.signature" class="user-card-item">
            <span class="user-card-label">签名：</span>
            <span class="user-card-value user-card-signature">{{ userInfo.signature }}</span>
          </div>
          <div class="user-card-item">
            <span class="user-card-label">用户名：</span>
            <span class="user-card-value">{{ userInfo.username }}</span>
          </div>
        </div>
        <div class="user-card-footer">
          <el-button size="small" type="primary" @click="handleCopyUserInfo">点击复制</el-button>
        </div>
      </div>
      <div v-else class="user-info-card">
        <div class="user-card-content">
          <div class="user-card-item">
            <span class="user-card-label">用户名：</span>
            <span class="user-card-value">{{ displayName }}</span>
          </div>
        </div>
      </div>
      </el-popover>
      <div class="user-info">
        <div class="user-name-primary user-name-clickable" @click.stop="handleCopyName">{{ displayName }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { ElSelect, ElOption, ElAvatar, ElPopover, ElButton, ElMessage } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '../types'
import { useFormDataStore } from '../../stores-v2/formData'
import { searchUsersFuzzy, getUsersByUsernames } from '@/api/user'
import type { UserInfo } from '@/types'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 用户选项列表（用于选择器）
const userOptions = ref<UserInfo[]>([])

// 当前用户信息（用于显示）
const userInfo = ref<UserInfo | null>(null)

// 加载状态
const loading = ref(false)

// Popover 显示状态
const showPopover = ref(false)

// 防抖定时器
let searchTimer: ReturnType<typeof setTimeout> | null = null

// 内部值（用于 v-model）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit' || props.mode === 'search') {
      return props.value?.raw ?? null
    }
    return null
  },
  set: (newValue: any) => {
    if (props.mode === 'edit') {
      const selectedUser = userOptions.value.find((u: UserInfo) => u.username === newValue)
      const newFieldValue = {
        raw: newValue, // 提交时只提交 username
        display: selectedUser?.nickname ? `${selectedUser.username}(${selectedUser.nickname})` : (selectedUser?.username || String(newValue)),
        meta: {
          userInfo: selectedUser
        }
      }
      
      formDataStore.setValue(props.fieldPath, newFieldValue)
      emit('update:modelValue', newFieldValue)
    }
  }
})

// 显示名称：username(昵称) 或 username
const displayName = computed(() => {
  if (userInfo.value) {
    return userInfo.value.nickname ? `${userInfo.value.username}(${userInfo.value.nickname})` : userInfo.value.username
  }
  if (props.value?.display) {
    return props.value.display
  }
  if (props.value?.raw) {
    return String(props.value.raw)
  }
  return '-'
})

// 选中用户（用于选择器显示）
const selectedUserForDisplay = computed(() => {
  if (props.mode === 'edit' || props.mode === 'search') {
    const currentValue = internalValue.value
    if (currentValue) {
      // 🔥 优化：优先从 userInfoMap 中获取（避免重复调用接口）
      if (props.userInfoMap && props.userInfoMap.has(currentValue)) {
        const user = props.userInfoMap.get(currentValue) as UserInfo
        // 同时添加到 userOptions 中，以便后续使用
        if (!userOptions.value.find((u: UserInfo) => u.username === currentValue)) {
          userOptions.value.push(user)
        }
        return user
      }
      
      // 优先从 userOptions 中查找
      let user = userOptions.value.find((u: UserInfo) => u.username === currentValue)
      if (user) {
        return user
      }
      // 如果 userOptions 中没有，尝试从 meta 中获取
      if (props.value?.meta?.userInfo && props.value.meta.userInfo.username === currentValue) {
        user = props.value.meta.userInfo
        // 同时添加到 userOptions 中，以便后续使用
        if (!userOptions.value.find((u: UserInfo) => u.username === currentValue)) {
          userOptions.value.push(user)
        }
        return user
      }
      // 如果都没有，尝试从 userInfo 中获取（可能是刚加载的）
      if (userInfo.value && userInfo.value.username === currentValue) {
        user = userInfo.value
        // 同时添加到 userOptions 中
        if (!userOptions.value.find((u: UserInfo) => u.username === currentValue)) {
          userOptions.value.push(user)
        }
        return user
      }
    }
  }
  return null
})

// 处理远程搜索（防抖）
function handleRemoteSearch(query: string): void {
  if (searchTimer) {
    clearTimeout(searchTimer)
  }
  
  searchTimer = setTimeout(async () => {
    if (!query || query.trim() === '') {
      userOptions.value = []
      return
    }
    
    try {
      loading.value = true
      const response = await searchUsersFuzzy(query.trim(), 20)
      userOptions.value = response.users || []
    } catch (error) {
      // 搜索用户失败，静默处理
      userOptions.value = []
    } finally {
      loading.value = false
    }
  }, 300) // 300ms 防抖
}

// 处理选择变化
function handleChange(value: any): void {
  // 已经在 internalValue 的 setter 中处理
  // 如果选中了用户，确保 userOptions 中包含该用户（用于显示）
  if (value) {
    const existingUser = userOptions.value.find((u: UserInfo) => u.username === value)
    if (!existingUser) {
      // 如果 userOptions 中没有，尝试从 meta 中获取或重新加载
      if (props.value?.meta?.userInfo && props.value.meta.userInfo.username === value) {
        userOptions.value.push(props.value.meta.userInfo)
      } else {
        // 如果没有，尝试加载用户信息
        loadUserInfo(value).then((user) => {
          if (user && !userOptions.value.find((u: UserInfo) => u.username === value)) {
            userOptions.value.push(user)
          }
        })
      }
    }
  }
}

// 处理聚焦（如果有初始值，加载用户信息）
function handleFocus(): void {
  if (props.value?.raw && userOptions.value.length === 0) {
    // 如果有值但没有选项，尝试搜索
    handleRemoteSearch(String(props.value.raw))
  }
}

// 加载用户信息（用于显示）
async function loadUserInfo(username: string | null): Promise<UserInfo | null> {
  if (!username) {
    userInfo.value = null
    return null
  }
  
  console.log('[UserWidget] 🔍 loadUserInfo 被调用', {
    username,
    mode: props.mode,
    hasUserInfoMap: !!props.userInfoMap,
    fieldCode: props.field?.code,
    timestamp: new Date().toISOString()
  })
  
  // 🔥 优化：优先从 userInfoMap 中获取（避免重复调用接口）
  if (props.userInfoMap && props.userInfoMap.has(username)) {
    const user = props.userInfoMap.get(username) as UserInfo
    userInfo.value = user
    console.log('[UserWidget] ✅ 从 userInfoMap 获取用户信息', username)
    return user
  }
  
  // 如果 meta 中已有用户信息，直接使用
  if (props.value?.meta?.userInfo && props.value.meta.userInfo.username === username) {
    userInfo.value = props.value.meta.userInfo
    console.log('[UserWidget] ✅ 从 meta 获取用户信息', username)
    return props.value.meta.userInfo
  }
  
  // 🔥 在 table-cell 模式下，如果有 userInfoMap，完全依赖它，不主动调用 API
  // TableRenderer 会在渲染前统一批量查询所有用户信息
  if (props.mode === 'table-cell' && props.userInfoMap) {
    console.log('[UserWidget] ⏭️ table-cell 模式且有 userInfoMap，不主动调用 API', username)
    // 如果 userInfoMap 中没有，说明 TableRenderer 的批量查询还没完成或用户不存在
    // 等待一段时间后再次检查（最多等待 500ms）
    for (let i = 0; i < 5; i++) {
      await new Promise(resolve => setTimeout(resolve, 100))
      if (props.userInfoMap.has(username)) {
        const user = props.userInfoMap.get(username) as UserInfo
        userInfo.value = user
        console.log('[UserWidget] ✅ 批量查询后从 userInfoMap 获取用户信息', username)
        return user
      }
    }
    // 如果等待后还是没有，说明用户不存在或批量查询失败，返回 null
    console.log('[UserWidget] ⚠️ table-cell 模式，等待后仍未找到用户信息', username)
    userInfo.value = null
    return null
  }
  
  // 🔥 使用 userInfoStore 批量查询（自动处理缓存和去重）
  // 注意：在 table-cell 模式下，如果 userInfoMap 存在，应该已经由 TableRenderer 统一查询
  // 这里只处理独立表单页面或其他模式的情况
  try {
    const { useUserInfoStore } = await import('@/stores/userInfo')
    const userInfoStore = useUserInfoStore()
    
    console.log('[UserWidget] 🔍 调用 userInfoStore.batchGetUserInfo', username)
    const users = await userInfoStore.batchGetUserInfo([username])
    
    if (users && users.length > 0) {
      const user = users[0] as UserInfo
      userInfo.value = user
      // 🔥 如果 userInfoMap 存在，也更新到 map 中（缓存）
      if (props.userInfoMap) {
        props.userInfoMap.set(username, user)
      }
      console.log('[UserWidget] ✅ 获取到用户信息', username)
      return user
    } else {
      userInfo.value = null
      console.log('[UserWidget] ⚠️ 未找到用户信息', username)
      return null
    }
  } catch (error) {
    // 查询用户信息失败，静默处理
    console.error('[UserWidget] ❌ 查询用户信息失败', username, error)
    userInfo.value = null
    return null
  }
}

// 监听值变化，加载用户信息
watch(() => props.value?.raw, (newValue: any) => {
  if (props.mode === 'edit' || props.mode === 'search') {
    // 编辑模式：如果有值，确保 userOptions 中包含该用户
    if (newValue) {
      const username = String(newValue)
      const existingUser = userOptions.value.find((u: UserInfo) => u.username === username)
      if (!existingUser) {
        // 如果 meta 中有用户信息，直接添加
        if (props.value?.meta?.userInfo && props.value.meta.userInfo.username === username) {
          userOptions.value.push(props.value.meta.userInfo)
        } else {
          // 否则加载用户信息
          loadUserInfo(username).then((user) => {
            if (user && !userOptions.value.find((u: UserInfo) => u.username === username)) {
              userOptions.value.push(user)
            }
          })
        }
      }
    }
  } else {
    // 显示模式：加载用户信息用于显示
    if (newValue) {
      loadUserInfo(String(newValue)).then((user) => {
      })
    } else {
      userInfo.value = null
    }
  }
}, { immediate: true })

// 监听 mode 变化，如果切换到显示模式，加载用户信息
watch(() => props.mode, (newMode: string) => {
  if (newMode !== 'edit' && newMode !== 'search' && props.value?.raw) {
    loadUserInfo(String(props.value.raw))
  }
})

// 处理用户信息复制
function handleCopyUserInfo(event?: Event): void {
  if (event) {
    event.stopPropagation() // 阻止事件冒泡
  }
  if (userInfo.value) {
    // 复制用户信息：username(昵称) 格式，如果有邮箱也包含
    const copyText = userInfo.value.nickname 
      ? `${userInfo.value.username}(${userInfo.value.nickname})`
      : userInfo.value.username
    
    navigator.clipboard.writeText(copyText).then(() => {
      ElMessage.success('已复制用户信息')
    }).catch(() => {
      ElMessage.error('复制失败')
    })
  } else {
    // 如果没有用户信息，尝试复制原始值
    const rawValue = props.value?.raw
    if (rawValue) {
      navigator.clipboard.writeText(String(rawValue)).then(() => {
        ElMessage.success('已复制')
      }).catch(() => {
        ElMessage.error('复制失败')
      })
    }
  }
}

// 处理名称复制（只复制显示名称）
function handleCopyName(event: Event): void {
  event.stopPropagation()
  event.preventDefault()
  navigator.clipboard.writeText(displayName.value).then(() => {
    ElMessage.success('已复制名称')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}

// 处理头像点击（显示用户信息弹窗）
function handleAvatarClick(event: Event): void {
  event.stopPropagation()
  event.preventDefault()
  showPopover.value = !showPopover.value
}

// 组件挂载时，如果有初始值，加载用户信息
onMounted(() => {
  if (props.value?.raw) {
    if (props.mode === 'edit' || props.mode === 'search') {
      // 编辑模式：如果有初始值，需要加载用户信息到 userOptions 中以便显示
      const username = String(props.value.raw)
      // 如果 meta 中有用户信息，直接添加到 userOptions
      if (props.value?.meta?.userInfo) {
        const existingUser = userOptions.value.find((u: UserInfo) => u.username === username)
        if (!existingUser) {
          userOptions.value.push(props.value.meta.userInfo)
        }
      } else {
        // 如果没有，尝试搜索加载
        loadUserInfo(username).then(() => {
          if (userInfo.value) {
            const existingUser = userOptions.value.find((u: UserInfo) => u.username === username)
            if (!existingUser) {
              userOptions.value.push(userInfo.value)
            }
          }
        })
      }
    } else {
      // 显示模式：加载用户信息用于显示
      loadUserInfo(String(props.value.raw))
    }
  }
})
</script>

<style scoped>
.user-widget {
  width: 100%;
}

/* 用户选项样式 */
.user-option {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  flex-shrink: 0;
}

.user-avatar-small {
  flex-shrink: 0;
}

.user-name {
  flex: 1;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.user-nickname {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* 选择器包装器 */
.user-select-wrapper {
  position: relative;
  width: 100%;
}

/* 选中后的显示（覆盖在 select 上方） */
.user-select-display {
  position: absolute;
  top: 1px;
  left: 11px;
  right: 30px;
  height: calc(100% - 2px);
  display: flex;
  align-items: center;
  gap: 6px;
  pointer-events: none;
  z-index: 10;
  background: var(--el-bg-color);
  border-radius: 4px;
}

.user-select-display .user-avatar-small {
  flex-shrink: 0;
}

.user-select-display .user-display-text {
  font-size: 14px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 隐藏 select 的默认 label 显示 */
.user-select-hidden-label :deep(.el-input__inner) {
  color: transparent !important;
  caret-color: transparent;
}

.user-select-hidden-label :deep(.el-select__caret) {
  z-index: 2;
  position: relative;
}

.user-select-hidden-label :deep(.el-select__tags) {
  display: none !important;
}

/* 当 select 聚焦时，隐藏自定义显示 */
.user-select-wrapper:has(.el-select.is-focus) .user-select-display,
.user-select-wrapper:has(.el-select__wrapper.is-focus) .user-select-display {
  display: none;
}

/* 显示模式样式 */
.user-display,
.user-cell {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}


/* 用户信息卡片样式 */
.user-info-card {
  padding: 0;
}

.user-card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.user-card-names {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.user-card-primary {
  font-size: 16px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.user-card-username {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.user-card-content {
  padding: 12px 16px;
}

.user-card-item {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
  font-size: 14px;
}

.user-card-item:has(.user-card-signature) {
  align-items: flex-start;
}

.user-card-item:last-child {
  margin-bottom: 0;
}

.user-card-label {
  color: var(--el-text-color-secondary);
  margin-right: 8px;
  min-width: 60px;
}

.user-card-value {
  color: var(--el-text-color-primary);
  flex: 1;
  word-break: break-all;
}

.user-card-item:has(.user-card-signature) {
  align-items: flex-start;
}

.user-card-signature {
  word-break: break-word;
  white-space: pre-wrap;
  line-height: 1.5;
}

.user-card-footer {
  padding: 12px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
  text-align: center;
}

.user-detail {
  display: flex;
  align-items: center;
  gap: 16px;
}

.user-avatar-large {
  flex-shrink: 0;
}

.user-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
}

.user-name-primary {
  font-size: 16px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.user-username {
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

.user-email {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>

<style>
/* 可点击样式（全局，供 UserWidget.ts 使用） */
.user-clickable {
  cursor: pointer;
  user-select: none;
  transition: all 0.2s;
}

.user-clickable:hover {
  opacity: 0.8;
  transform: translateY(-1px);
}

/* 头像可点击样式 */
.user-avatar-clickable {
  cursor: pointer;
  transition: all 0.2s;
}

.user-avatar-clickable:hover {
  opacity: 0.8;
  transform: scale(1.05);
}

/* 名称可点击样式 */
.user-name-clickable {
  cursor: pointer;
  user-select: none;
  transition: all 0.2s;
  color: var(--el-color-primary);
}

.user-name-clickable:hover {
  opacity: 0.8;
  text-decoration: underline;
}

/* 下拉选项样式 */
.user-select-dropdown-popper .el-select-dropdown__item {
  height: auto;
  padding: 8px 12px;
}

.user-select-dropdown-popper .el-select-dropdown__item.hover {
  background-color: var(--el-fill-color-light);
}

/* 全局样式：用户信息弹出框 */
.user-info-popover {
  padding: 0 !important;
  z-index: 3000 !important;
}

.user-info-popover .el-popover__reference {
  display: inline-flex;
  align-items: center;
}
</style>

