<!--
  UserWidget - 用户组件
  功能：
  - 输入场景（edit/search）：用户选择器，支持模糊搜索
  - 输出场景（response/table-cell/detail）：显示用户信息（头像、名称等）
-->

<template>
  <div class="user-widget">
    <!-- 编辑模式：用户选择器（使用弹窗搜索） -->
    <div v-if="mode === 'edit' || mode === 'search'" class="user-select-wrapper">
      <!-- 选中后的显示 -->
      <div
        v-if="selectedUserForDisplay"
        class="user-select-display"
        :class="{ 'is-disabled': widgetConfig.disabled }"
        @click="!widgetConfig.disabled && handleOpenDialog()"
      >
        <el-avatar 
          v-if="selectedUserForDisplay.avatar" 
          :src="selectedUserForDisplay.avatar" 
          :size="24" 
          class="user-avatar-small"
        >
          {{ selectedUserForDisplay.username?.[0]?.toUpperCase() || 'U' }}
        </el-avatar>
        <el-avatar 
          v-else
          :size="24" 
          class="user-avatar-small"
        >
          {{ selectedUserForDisplay.username?.[0]?.toUpperCase() || 'U' }}
        </el-avatar>
        <span class="user-display-text">
          {{ formatUserDisplayName(selectedUserForDisplay) }}
        </span>
        <el-icon v-if="!widgetConfig.disabled" class="edit-icon">
          <Edit />
        </el-icon>
      </div>
      <!-- 未选中时显示按钮 -->
      <el-button
        v-else
        :disabled="widgetConfig.disabled"
        :placeholder="field.desc || `请选择${field.name}`"
        @click="!widgetConfig.disabled && handleOpenDialog()"
      >
        <el-icon><User /></el-icon>
        {{ field.desc || `请选择${field.name}` }}
      </el-button>
      
      <!-- 用户搜索弹窗 -->
      <UserSearchDialog
        v-model="dialogVisible"
        :title="`选择${field.name || '用户'}`"
        :placeholder="field.desc || '请输入用户名或邮箱搜索'"
        :initial-username="value?.raw"
        @confirm="handleUserSelected"
      />
    </div>
    
    <!-- 响应模式（使用 UserDisplay 组件） -->
    <UserDisplay
      v-else-if="mode === 'response'"
      :user-info="userInfo"
      :username="value?.raw"
      mode="card"
      layout="horizontal"
      size="small"
    />
    
    <!-- 表格单元格模式（使用 UserDisplay 组件） -->
    <UserDisplay
      v-else-if="mode === 'table-cell'"
      :user-info="userInfo"
      :username="value?.raw"
      mode="card"
      layout="horizontal"
      size="small"
    />
    
    <!-- 详情模式（使用 UserDisplay 组件） -->
    <div v-else-if="mode === 'detail'" class="user-detail">
      <UserDisplay
        :user-info="userInfo"
        :username="value?.raw"
        mode="card"
        layout="horizontal"
        size="large"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import UserDisplay from './UserDisplay.vue'
import UserSearchDialog from './UserSearchDialog.vue'
import { ElAvatar, ElButton, ElIcon } from 'element-plus'
import { User, Edit } from '@element-plus/icons-vue'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { formatUserDisplayName } from '@/utils/userInfo'
import type { UserInfo } from '@/types'
import { Logger } from '@/core/utils/logger'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import { useUserInfoStore } from '@/stores/userInfo'

const COMPONENT_NAME = 'UserWidget'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 获取配置（带类型）
const widgetConfig = computed(() => {
  return (props.field.widget?.config || {}) as import('@/core/types/widget-configs').UserWidgetConfig
})

// 弹窗显示状态
const dialogVisible = ref(false)

// 当前用户信息（用于显示）
const userInfo = ref<UserInfo | null>(null)

// 处理打开弹窗
function handleOpenDialog(): void {
  dialogVisible.value = true
}

// 处理用户选择
function handleUserSelected(user: UserInfo): void {
  // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
  const newFieldValue = createFieldValue(
    props.field,
    user.username, // 提交时只提交 username
    formatUserDisplayName(user),
    {
      userInfo: user
    }
  )
  
  formDataStore.setValue(props.fieldPath, newFieldValue)
  emit('update:modelValue', newFieldValue)
  
  // 更新 userInfo 用于显示
  userInfo.value = user
}

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

// 选中用户（用于显示）
const selectedUserForDisplay = computed(() => {
  if (props.mode === 'edit' || props.mode === 'search') {
    const currentValue = props.value?.raw
    if (currentValue) {
      // 从 meta 中获取（优先）
      if (props.value?.meta?.userInfo && props.value.meta.userInfo.username === currentValue) {
        userInfo.value = props.value.meta.userInfo
        return props.value.meta.userInfo
      }
      
      // 从 userInfo 中获取（可能是刚加载的）
      if (userInfo.value && userInfo.value.username === currentValue) {
        return userInfo.value
      }
      
      // 🔥 如果都没有，loadUserInfo 会从全局 userInfoStore 加载（有缓存，不会重复调用接口）
    }
  }
  return null
})

// ⭐ 注意：UserWidget 现在使用 UserSearchDialog 弹窗，不再使用 el-select 下拉框
// 以下代码已移除，因为不再需要：
// - handleRemoteSearch（搜索逻辑在 UserSearchDialog 中）
// - handleChange（选择逻辑在 UserSearchDialog 中）
// - handleFocus（聚焦逻辑在 UserSearchDialog 中）
// - handleVisibleChange（下拉框显示逻辑已移除）
// - handleClear（清空逻辑在 UserSearchDialog 中）

// 加载用户信息（用于显示）
async function loadUserInfo(username: string | null): Promise<UserInfo | null> {
  if (!username) {
    userInfo.value = null
    return null
  }
  
  // 如果 meta 中已有用户信息，直接使用
  if (props.value?.meta?.userInfo && props.value.meta.userInfo.username === username) {
    userInfo.value = props.value.meta.userInfo
    return props.value.meta.userInfo
  }
  
  // 🔥 使用全局 userInfoStore 获取（自动处理缓存和去重）
  // userInfoStore 是全局的，有缓存机制，不会重复调用接口
  try {
    const userInfoStore = useUserInfoStore()
    
    // 使用 getUserInfo 方法（会自动从缓存读取，如果过期会后台刷新）
    const user = await userInfoStore.getUserInfo(username)
    
    if (user) {
      userInfo.value = user
      return user
    } else {
      userInfo.value = null
      return null
    }
  } catch (error) {
    // 查询用户信息失败，静默处理
    Logger.error(COMPONENT_NAME, '查询用户信息失败', { username, error })
    userInfo.value = null
    return null
  }
}

// 监听值变化，加载用户信息
watch(() => props.value?.raw, (newValue: any) => {
  if (props.mode === 'edit' || props.mode === 'search') {
    // 编辑模式：如果有值，加载用户信息用于显示
    if (newValue) {
      loadUserInfo(String(newValue))
    } else {
      userInfo.value = null
    }
  } else {
    // 显示模式：加载用户信息用于显示
    if (newValue) {
      loadUserInfo(String(newValue))
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
// 以下函数已移除，现在使用 UserDisplay 组件处理用户信息展示和复制
// handleCopyUserInfo, handleCopyName, handleAvatarClick 已由 UserDisplay 组件处理

// 组件挂载时，如果有初始值，加载用户信息
// 🔥 同时检查是否有动态默认值（如 Me()）
onMounted(async () => {
  // 🔥 检查是否有动态默认值需要设置（Me()）
  // ⚠️ 重要：只有在新增模式下才使用默认值，编辑模式下不应该使用默认值
  if (props.mode === 'edit') {
    // ⚠️ 使用 nextTick 等待一下，确保 initializeForm 已经完成
    // 这样可以避免在编辑模式下错误地使用默认值
    await nextTick()
    
    const currentRaw = props.value?.raw
    const existingValue = formDataStore.getValue(props.fieldPath)
    
    // 🔥 检查是否需要解析 Me() 或 MyLeader() 函数调用
    // 情况1：value.raw 是 "Me()" 或 "MyLeader()" 字符串（FormDomainService 还没有解析）
    // 情况2：value.raw 是 null/undefined/空字符串，且配置中有 "Me()" 或 "MyLeader()" 默认值
    const defaultValue = props.field.widget?.config?.default
    const needsResolveMe = currentRaw === 'Me()' || 
      ((!currentRaw || currentRaw === '') && defaultValue === 'Me()')
    const needsResolveMyLeader = currentRaw === 'MyLeader()' || 
      ((!currentRaw || currentRaw === '') && defaultValue === 'MyLeader()')
    
    if (needsResolveMe || needsResolveMyLeader) {
      // ⚠️ 检查是否是编辑模式：
      // 1. 如果 meta.fromInitialData 为 true，说明字段来自 initialData（编辑模式）
      // 2. 如果 existingValue 存在且 raw 不是 "Me()"，说明是编辑模式
      // 编辑模式下，existingValue.raw 应该是实际的用户名，不应该是 "Me()"
      const isEditMode = props.value?.meta?.fromInitialData === true ||
                        (existingValue && 
                         existingValue.raw !== null && 
                         existingValue.raw !== undefined && 
                         existingValue.raw !== '' && 
                         existingValue.raw !== 'Me()')
      
      // 只有在新增模式下才解析 Me() 或 MyLeader()
      if (!isEditMode) {
        const { useAuthStore } = await import('@/stores/auth')
        const authStore = useAuthStore()
        
        let targetUsername: string | null = null
        
        if (needsResolveMe) {
          // Me() - 当前登录用户
          targetUsername = authStore.user?.username || null
        } else if (needsResolveMyLeader) {
          // MyLeader() - 当前用户的上级领导
          targetUsername = authStore.user?.leader_username || null
        }
        
        if (targetUsername) {
          // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
          const newFieldValue = createFieldValue(
            props.field,
            targetUsername,
            targetUsername
          )
          formDataStore.setValue(props.fieldPath, newFieldValue)
          emit('update:modelValue', newFieldValue)
          // 加载用户信息
          loadUserInfo(targetUsername)
          return
        }
      }
    }
  }

  if (props.value?.raw) {
    // 加载用户信息用于显示
    loadUserInfo(String(props.value.raw))
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
  width: 100%;
}

.user-avatar {
  flex-shrink: 0;
}

.user-avatar-small {
  flex-shrink: 0;
}

.user-name {
  flex: 0 0 auto;
  font-size: 14px;
  color: var(--el-text-color-primary);
  font-weight: 500;
  white-space: nowrap;
}

.user-signature {
  flex: 1;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-left: auto;
}

/* 选择器包装器 */
.user-select-wrapper {
  position: relative;
  width: 100%;
}

/* 选中后的显示（可点击） */
.user-select-display {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background-color: var(--el-bg-color);
  cursor: pointer;
  transition: all 0.2s;
}

.user-select-display:hover:not(.is-disabled) {
  border-color: var(--el-color-primary);
  background-color: var(--el-fill-color-light);
}

.user-select-display.is-disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.user-select-display .user-avatar-small {
  flex-shrink: 0;
}

.user-select-display .user-display-text {
  flex: 1;
  font-size: 14px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-select-display .edit-icon {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
  font-size: 16px;
  transition: color 0.2s;
}

.user-select-display:hover:not(.is-disabled) .edit-icon {
  color: var(--el-color-primary);
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
}

.user-info-popover .el-popover__reference {
  display: inline-flex;
  align-items: center;
}
</style>

