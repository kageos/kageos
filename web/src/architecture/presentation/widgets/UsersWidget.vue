<!--
  UsersWidget - 多用户组件
  功能：
  - 输入场景（edit/search）：多用户选择器，支持模糊搜索
  - 输出场景（response/table-cell/detail）：显示多个用户信息（头像、名称等）
  - 值使用逗号分隔的字符串存储（如 "user1,user2"），便于存储到数据库
-->
<template>
  <div class="users-widget">
    <!-- 编辑模式：多用户选择器（使用弹窗搜索） -->
    <div v-if="mode === 'edit' || mode === 'search'" class="users-select-wrapper">
      <!-- 选中后的显示 -->
      <div
        v-if="selectedUsersForDisplay.length > 0"
        class="users-select-display"
        @click="handleOpenDialog()"
      >
        <div class="selected-users-list">
          <div
            v-for="(user, index) in selectedUsersForDisplay"
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
      <!-- 未选中时显示按钮 -->
      <el-button
        v-else
        :placeholder="field.desc || `请选择${field.name}`"
        @click="handleOpenDialog()"
      >
        <el-icon><User /></el-icon>
        {{ field.desc || `请选择${field.name}` }}
      </el-button>
      
      <!-- 多用户搜索弹窗 -->
      <UsersSearchDialog
        v-model="dialogVisible"
        :title="`选择${field.name || '用户'}`"
        :placeholder="field.desc || '请输入用户名或邮箱搜索'"
        :initial-usernames="value?.raw"
        :max-count="maxCount"
        @confirm="handleUsersSelected"
      />
    </div>
    
    <!-- 响应模式：显示多个用户 -->
    <div v-else-if="mode === 'response'" class="users-response">
      <div v-if="displayUsers.length > 0" class="users-list users-list-horizontal">
        <UserDisplay
          v-for="(user, index) in displayUsers"
          :key="user.username || index"
          :user-info="user"
          :username="user.username"
          mode="card"
          layout="horizontal"
          size="small"
          class="user-item"
        />
      </div>
      <span v-else class="empty-text">-</span>
    </div>
    
    <!-- 表格单元格模式：只显示头像，hover 显示详细信息 -->
    <div v-else-if="mode === 'table-cell'" class="users-table-cell">
      <div v-if="displayUsers.length > 0" class="users-avatars-list">
        <el-popover
          v-for="(user, index) in displayUsers"
          :key="user.username || index"
          placement="top"
          :width="380"
          trigger="hover"
          popper-class="users-popover"
        >
          <template #reference>
            <el-avatar 
              v-if="user.avatar" 
              :src="user.avatar" 
              :size="24"
              class="user-avatar-item"
            >
              {{ user.username?.[0]?.toUpperCase() || 'U' }}
            </el-avatar>
            <el-avatar 
              v-else
              :size="24"
              class="user-avatar-item"
            >
              {{ user.username?.[0]?.toUpperCase() || 'U' }}
            </el-avatar>
          </template>
          <UserDetailCard :user-info="user" />
        </el-popover>
      </div>
      <span v-else class="empty-text">-</span>
    </div>
    
    <!-- 详情模式：只展示头像，支持最多显示数量限制 -->
    <div v-else-if="mode === 'detail'" class="users-detail">
      <div v-if="displayUsers.length > 0" class="users-avatars-list">
        <!-- 显示的头像（最多 maxDisplayCount 个） -->
        <el-popover
          v-for="(user, index) in displayedUsers"
          :key="user.username || index"
          placement="top"
          :width="380"
          trigger="hover"
          popper-class="users-popover"
        >
          <template #reference>
            <el-avatar 
              v-if="user.avatar" 
              :src="user.avatar" 
              :size="32"
              class="user-avatar-item"
            >
              {{ user.username?.[0]?.toUpperCase() || 'U' }}
            </el-avatar>
            <el-avatar 
              v-else
              :size="32"
              class="user-avatar-item"
            >
              {{ user.username?.[0]?.toUpperCase() || 'U' }}
            </el-avatar>
          </template>
          <UserDetailCard :user-info="user" />
        </el-popover>
        
        <!-- 省略号：点击显示全部 -->
        <el-popover
          v-if="hasMoreUsers"
          placement="top"
          :width="400"
          trigger="click"
          popper-class="users-popover"
        >
          <template #reference>
            <div class="users-more-indicator" @click.stop>
              <span class="more-text">+{{ remainingUsersCount }}</span>
            </div>
          </template>
          <div class="users-full-list">
            <div class="users-full-list-header">
              <span>全部管理员 ({{ displayUsers.length }})</span>
            </div>
            <div class="users-full-list-content">
              <div
                v-for="(user, index) in displayUsers"
                :key="user.username || index"
                class="users-full-list-item"
              >
                <el-avatar 
                  v-if="user.avatar" 
                  :src="user.avatar" 
                  :size="40"
                  class="user-avatar"
                >
                  {{ user.username?.[0]?.toUpperCase() || 'U' }}
                </el-avatar>
                <el-avatar 
                  v-else
                  :size="40"
                  class="user-avatar"
                >
                  {{ user.username?.[0]?.toUpperCase() || 'U' }}
                </el-avatar>
                <div class="user-info">
                  <div class="user-name">{{ user.username }}</div>
                  <div v-if="user.nickname" class="user-nickname">{{ user.nickname }}</div>
                </div>
              </div>
            </div>
          </div>
        </el-popover>
      </div>
      <span v-else class="empty-text">-</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import UserDisplay from './UserDisplay.vue'
import UserDetailCard from './UserDetailCard.vue'
import UsersSearchDialog from './UsersSearchDialog.vue'
import { ElAvatar, ElButton, ElIcon, ElPopover } from 'element-plus'
import { User, Edit, Close } from '@element-plus/icons-vue'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { formatUserDisplayName } from '@/utils/userInfo'
import type { UserInfo } from '@/types'
import { Logger } from '@/core/utils/logger'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import { useUserInfoStore } from '@/stores/userInfo'

const COMPONENT_NAME = 'UsersWidget'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 弹窗显示状态
const dialogVisible = ref(false)

// 当前用户信息列表（用于显示）
const userInfoList = ref<UserInfo[]>([])

// 获取配置
const config = computed(() => {
  return (props.field.widget?.config || {}) as UsersWidgetConfig
})

// 最大选择数量
const maxCount = computed(() => {
  return config.value?.max_count || 0
})

// 详情模式最多显示的头像数量（默认 5 个）
const maxDisplayCount = computed(() => {
  return config.value?.max_display_count || 5
})

interface UsersWidgetConfig {
  default?: string
  max_count?: number
  max_display_count?: number // 详情模式最多显示的头像数量
}

// 处理打开弹窗
function handleOpenDialog(): void {
  dialogVisible.value = true
}

// 处理用户选择（多个）
function handleUsersSelected(users: UserInfo[]): void {
  // 将用户列表转换为逗号分隔的字符串
  const usernames = users.map(u => u.username).join(',')
  const displayNames = users.map(u => formatUserDisplayName(u)).join(', ')
  
  // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
  const newFieldValue = createFieldValue(
    props.field,
    usernames, // 提交时使用逗号分隔的字符串
    displayNames,
    {
      userInfoList: users // 保存用户信息列表到 meta，用于显示
    }
  )
  
  formDataStore.setValue(props.fieldPath, newFieldValue)
  emit('update:modelValue', newFieldValue)
  
  // 更新 userInfoList 用于显示
  userInfoList.value = users
}

// 移除单个用户
function handleRemoveUser(user: UserInfo): void {
  const currentUsernames = props.value?.raw ? String(props.value.raw).split(',').map(u => u.trim()).filter(u => u) : []
  const newUsernames = currentUsernames.filter(u => u !== user.username)
  
  // 重新加载用户信息
  if (newUsernames.length > 0) {
    loadUsersInfo(newUsernames.join(','))
  } else {
    // 清空
    const newFieldValue = createFieldValue(
      props.field,
      '',
      '',
      {}
    )
    formDataStore.setValue(props.fieldPath, newFieldValue)
    emit('update:modelValue', newFieldValue)
    userInfoList.value = []
  }
}

// 选中用户列表（用于编辑模式显示）
const selectedUsersForDisplay = computed(() => {
  if (props.mode === 'edit' || props.mode === 'search') {
    // 优先从 meta 中获取
    if (props.value?.meta?.userInfoList && Array.isArray(props.value.meta.userInfoList)) {
      return props.value.meta.userInfoList
    }
    // 从 userInfoList 中获取
    if (userInfoList.value.length > 0) {
      return userInfoList.value
    }
  }
  return []
})

// 显示用户列表（用于响应模式）
const displayUsers = computed(() => {
  // 优先从 meta 中获取
  if (props.value?.meta?.userInfoList && Array.isArray(props.value.meta.userInfoList)) {
    return props.value.meta.userInfoList
  }
  // 从 userInfoList 中获取
  if (userInfoList.value.length > 0) {
    return userInfoList.value
  }
  return []
})

// 详情模式：显示的头像列表（最多 maxDisplayCount 个）
const displayedUsers = computed(() => {
  if (props.mode !== 'detail') {
    return displayUsers.value
  }
  return displayUsers.value.slice(0, maxDisplayCount.value)
})

// 详情模式：是否有更多用户
const hasMoreUsers = computed(() => {
  if (props.mode !== 'detail') {
    return false
  }
  return displayUsers.value.length > maxDisplayCount.value
})

// 详情模式：剩余用户数量
const remainingUsersCount = computed(() => {
  if (props.mode !== 'detail') {
    return 0
  }
  return displayUsers.value.length - maxDisplayCount.value
})

// 加载用户信息列表（用于显示）
async function loadUsersInfo(usernames: string): Promise<void> {
  if (!usernames || usernames.trim() === '') {
    userInfoList.value = []
    return
  }
  
  const usernameList = usernames.split(',').map(u => u.trim()).filter(u => u)
  if (usernameList.length === 0) {
    userInfoList.value = []
    return
  }
  
  // 🔥 使用全局 userInfoStore 获取（自动处理缓存和去重）
  try {
    const userInfoStore = useUserInfoStore()
    const users: UserInfo[] = []
    
    // 并行加载所有用户信息
    await Promise.all(
      usernameList.map(async (username) => {
        try {
          const user = await userInfoStore.getUserInfo(username)
          if (user) {
            users.push(user)
          }
        } catch (error) {
          Logger.error(COMPONENT_NAME, '加载用户信息失败', { username, error })
        }
      })
    )
    
    userInfoList.value = users
  } catch (error) {
    Logger.error(COMPONENT_NAME, '加载用户信息列表失败', { usernames, error })
    userInfoList.value = []
  }
}

// 监听值变化，加载用户信息
watch(() => props.value?.raw, (newValue: any) => {
  if (newValue) {
    loadUsersInfo(String(newValue))
  } else {
    userInfoList.value = []
  }
}, { immediate: true })

// 监听 mode 变化，如果切换到显示模式，加载用户信息
watch(() => props.mode, (newMode: string) => {
  if (newMode !== 'edit' && newMode !== 'search' && props.value?.raw) {
    loadUsersInfo(String(props.value.raw))
  }
})

// 组件挂载时，如果有初始值，加载用户信息
// 🔥 同时检查是否有动态默认值（如 $me）
onMounted(async () => {
  // 🔥 检查是否有动态默认值需要设置（$me）
  // ⚠️ 重要：只有在新增模式下才使用默认值，编辑模式下不应该使用默认值
  if (props.mode === 'edit') {
    // ⚠️ 使用 nextTick 等待一下，确保 initializeForm 已经完成
    // 这样可以避免在编辑模式下错误地使用默认值
    await nextTick()
    
    const currentRaw = props.value?.raw
    const existingValue = formDataStore.getValue(props.fieldPath)
    const config = props.field.widget?.config
    const defaultValue = config && typeof config === 'object' && 'default' in config 
      ? (config as Record<string, any>).default 
      : undefined
    
    // 🔥 检查是否需要解析 $me 动态变量
    // 情况1：value.raw 是 "$me" 字符串（FormDomainService 还没有解析）
    // 情况2：value.raw 包含 "$me"（如 "$me,user2"）
    // 情况3：value.raw 是 null/undefined/空字符串，且配置中有 "$me" 默认值
    const needsResolveMe = (typeof currentRaw === 'string' && currentRaw.includes('$me')) ||
      ((!currentRaw || currentRaw === '') && 
       typeof defaultValue === 'string' && defaultValue.includes('$me'))
    
    if (needsResolveMe) {
      // ⚠️ 检查是否是编辑模式：如果 existingValue 存在且 raw 不是 "$me" 且不包含 "$me"，说明是编辑模式
      // 编辑模式下，existingValue.raw 应该是实际的用户名，不应该是 "$me"
      const isEditMode = existingValue && 
                        existingValue.raw !== null && 
                        existingValue.raw !== undefined && 
                        existingValue.raw !== '' && 
                        (typeof existingValue.raw !== 'string' || !existingValue.raw.includes('$me'))
      
      // 只有在新增模式下才解析 $me
      if (!isEditMode) {
        const { useAuthStore } = await import('@/stores/auth')
        const authStore = useAuthStore()
        const currentUsername = authStore.user?.username
        
        if (currentUsername) {
          let processedValue: string
          
          if (typeof defaultValue === 'string' && defaultValue === '$me') {
            // 单个 $me
            processedValue = currentUsername
          } else if (typeof defaultValue === 'string' && defaultValue.includes(',')) {
            // 多个默认值，用逗号分隔（如 "$me,user2"）
            processedValue = defaultValue.replace(/\$me/g, currentUsername)
          } else if (typeof currentRaw === 'string' && currentRaw === '$me') {
            // value.raw 是 "$me"，直接替换
            processedValue = currentUsername
          } else if (typeof currentRaw === 'string' && currentRaw.includes('$me')) {
            // value.raw 包含 "$me"（如 "$me,user2"）
            processedValue = currentRaw.replace(/\$me/g, currentUsername)
          } else {
            processedValue = currentUsername
          }
          
          if (processedValue && processedValue.trim()) {
            // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
            const newFieldValue = createFieldValue(
              props.field,
              processedValue,
              processedValue
            )
            formDataStore.setValue(props.fieldPath, newFieldValue)
            emit('update:modelValue', newFieldValue)
            // 加载用户信息
            loadUsersInfo(processedValue)
            return
          }
        }
      }
    }
  }

  if (props.value?.raw) {
    // 加载用户信息用于显示
    loadUsersInfo(String(props.value.raw))
  }
})
</script>

<style scoped>
.users-widget {
  width: 100%;
}

/* 选择器包装器 */
.users-select-wrapper {
  position: relative;
  width: 100%;
}

/* 选中后的显示（可点击） */
.users-select-display {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background-color: var(--el-bg-color);
  cursor: pointer;
  transition: all 0.2s;
  min-height: 40px;
}

.users-select-display:hover {
  border-color: var(--el-color-primary);
  background-color: var(--el-fill-color-light);
}

.selected-users-list {
  flex: 1;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.selected-user-tag {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background-color: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}

.selected-user-tag .user-avatar-small {
  flex-shrink: 0;
}

.selected-user-tag .user-display-text {
  font-size: 12px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

.selected-user-tag .remove-icon {
  cursor: pointer;
  color: var(--el-text-color-secondary);
  font-size: 14px;
  transition: color 0.2s;
  flex-shrink: 0;
}

.selected-user-tag .remove-icon:hover {
  color: var(--el-color-danger);
}

.users-select-display .edit-icon {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
  font-size: 16px;
  transition: color 0.2s;
  margin-top: 2px;
}

.users-select-display:hover .edit-icon {
  color: var(--el-color-primary);
}

/* 显示模式样式 */
.users-response,
.users-table-cell,
.users-detail {
  width: 100%;
}

.users-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* 横向展示的用户列表 */
.users-list-horizontal {
  flex-direction: row;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
}

.user-item {
  width: 100%;
}

/* 横向展示时，每个用户项不需要占满宽度 */
.users-list-horizontal .user-item {
  width: auto;
  flex-shrink: 0;
}

/* 表格单元格模式：头像列表 */
.users-avatars-list {
  display: flex;
  flex-direction: row;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.user-avatar-item {
  cursor: pointer;
  transition: transform 0.2s;
  flex-shrink: 0;
}

.user-avatar-item:hover {
  transform: scale(1.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.empty-text {
  color: var(--el-text-color-placeholder);
  font-size: 14px;
}

/* 详情模式：头像列表 */
.users-detail .users-avatars-list {
  display: flex;
  flex-direction: row;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.users-detail .user-avatar-item {
  cursor: pointer;
  transition: transform 0.2s;
  flex-shrink: 0;
}

.users-detail .user-avatar-item:hover {
  transform: scale(1.1);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

/* 省略号指示器 */
.users-more-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background-color: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  cursor: pointer;
  transition: all 0.2s;
  flex-shrink: 0;
}

.users-more-indicator:hover {
  background-color: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}

.users-more-indicator .more-text {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-regular);
}

.users-more-indicator:hover .more-text {
  color: var(--el-color-primary);
}

/* 全部用户列表弹窗 */
.users-full-list {
  max-height: 400px;
  overflow-y: auto;
}

.users-full-list-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.users-full-list-content {
  padding: 8px 0;
}

.users-full-list-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  transition: background-color 0.2s;
}

.users-full-list-item:hover {
  background-color: var(--el-fill-color-light);
}

.users-full-list-item .user-avatar {
  flex-shrink: 0;
}

.users-full-list-item .user-info {
  flex: 1;
  min-width: 0;
}

.users-full-list-item .user-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.users-full-list-item .user-nickname {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
  margin-top: 2px;
}
</style>

<style>
/* 全局样式：多个用户 popover */
.users-popover {
  padding: 0 !important;
}
</style>

