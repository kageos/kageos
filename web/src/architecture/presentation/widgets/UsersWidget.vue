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
    
    <!-- 详情模式：横着一行展示多个用户 -->
    <div v-else-if="mode === 'detail'" class="users-detail">
      <div v-if="displayUsers.length > 0" class="users-list users-list-horizontal">
        <UserDisplay
          v-for="(user, index) in displayUsers"
          :key="user.username || index"
          :user-info="user"
          :username="user.username"
          mode="card"
          layout="horizontal"
          size="medium"
          class="user-item"
        />
      </div>
      <span v-else class="empty-text">-</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
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

interface UsersWidgetConfig {
  default?: string
  max_count?: number
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
  if (props.mode === 'edit') {
    const currentRaw = props.value?.raw
    const shouldSetDefault = !currentRaw || currentRaw === '' || currentRaw === '$me'
    
    if (shouldSetDefault) {
      const config = props.field.widget?.config
      if (config && typeof config === 'object' && 'default' in config) {
        const defaultValue = (config as Record<string, any>).default
        if (typeof defaultValue === 'string' && defaultValue === '$me') {
          // 动态默认值：$me（当前登录用户）
          const { useAuthStore } = await import('@/stores/auth')
          const authStore = useAuthStore()
          const currentUsername = authStore.user?.username
          if (currentUsername) {
            // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
            const newFieldValue = createFieldValue(
              props.field,
              currentUsername,
              currentUsername
            )
            formDataStore.setValue(props.fieldPath, newFieldValue)
            emit('update:modelValue', newFieldValue)
            // 加载用户信息
            loadUsersInfo(currentUsername)
            return
          }
        } else if (typeof defaultValue === 'string' && defaultValue.includes(',')) {
          // 多个默认值，用逗号分隔（如 "$me,user2"）
          // 处理 $me 变量
          let processedDefault = defaultValue
          if (processedDefault.includes('$me')) {
            const { useAuthStore } = await import('@/stores/auth')
            const authStore = useAuthStore()
            const currentUsername = authStore.user?.username
            if (currentUsername) {
              processedDefault = processedDefault.replace(/\$me/g, currentUsername)
            } else {
              processedDefault = processedDefault.replace(/,\s*\$me|\$me\s*,/g, '').replace(/\$me/g, '')
            }
          }
          
          if (processedDefault && processedDefault.trim()) {
            const newFieldValue = createFieldValue(
              props.field,
              processedDefault,
              processedDefault
            )
            formDataStore.setValue(props.fieldPath, newFieldValue)
            emit('update:modelValue', newFieldValue)
            loadUsersInfo(processedDefault)
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
</style>

<style>
/* 全局样式：多个用户 popover */
.users-popover {
  padding: 0 !important;
}
</style>

