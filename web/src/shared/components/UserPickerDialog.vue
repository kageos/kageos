<template>
  <el-dialog
    v-model="dialogVisible"
    class="entity-search-dialog-shell"
    :title="title"
    width="620px"
    :close-on-click-modal="false"
    @close="handleClose"
    @opened="handleDialogOpened"
  >
    <div class="user-picker-dialog-search">
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
      <div class="dialog-status">
        <span class="status-chip">
          {{ loading ? '搜索中...' : (searchKeyword ? `${userList.length} 个结果` : '输入关键词开始搜索') }}
        </span>
        <span v-if="selectedUsers.length > 0" class="status-chip status-chip-active">
          已选 {{ selectedUsers.length }}{{ multiple && maxCount > 0 ? `/${maxCount}` : '' }} 项
        </span>
      </div>
    </div>

    <div v-if="multiple && selectedUsers.length > 0" class="user-picker-dialog-selected">
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

    <div class="user-picker-dialog-list">
      <div
        v-if="loading"
        class="user-picker-dialog-loading"
      >
        <el-icon class="is-loading"><Loading /></el-icon>
        <span>搜索中...</span>
      </div>
      <div
        v-else-if="userList.length === 0 && searchKeyword"
        class="user-picker-dialog-empty"
      >
        <el-empty description="未找到用户" :image-size="80" />
      </div>
      <div
        v-else-if="userList.length === 0 && !searchKeyword"
        class="user-picker-dialog-empty"
      >
        <el-empty description="请输入关键词搜索用户" :image-size="80" />
      </div>
      <div
        v-else
        class="user-picker-dialog-items"
      >
        <div
          v-for="user in userList"
          :key="user.username"
          class="user-picker-dialog-item"
          :class="{ 'is-selected': isUserSelected(user) }"
          @click="handlePickUser(user)"
        >
          <el-checkbox
            v-if="multiple"
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
            <div v-if="user.signature" class="user-signature">{{ user.signature }}</div>
          </div>
          <el-icon
            v-if="!multiple && isUserSelected(user)"
            class="selected-icon"
          >
            <Check />
          </el-icon>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="user-picker-dialog-footer">
        <el-button @click="handleClose">{{ multiple || !autoConfirmSingle ? '取消' : '关闭' }}</el-button>
        <el-button
          v-if="multiple || !autoConfirmSingle"
          type="primary"
          :disabled="selectedUsers.length === 0"
          @click="handleConfirm"
        >
          确认
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { ElAvatar, ElButton, ElCheckbox, ElDialog, ElEmpty, ElIcon, ElInput } from 'element-plus'
import { Check, Close, Loading, Search } from '@element-plus/icons-vue'
import { searchUsersFuzzy } from '@/architecture/infrastructure/api/user'
import { Logger } from '@/architecture/runtime/utils/logger'
import { useUserInfoStore } from '@/architecture/infrastructure/stores/userInfo'
import type { UserInfo } from '@/types'
import { formatUserDisplayName } from '@/utils/userInfo'

interface Props {
  modelValue: boolean
  title?: string
  placeholder?: string
  initialUsernames?: string | null
  multiple?: boolean
  maxCount?: number
  autoConfirmSingle?: boolean
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'confirm', users: UserInfo[]): void
}

const props = withDefaults(defineProps<Props>(), {
  title: '选择用户',
  placeholder: '请输入用户名或邮箱搜索',
  initialUsernames: null,
  multiple: false,
  maxCount: 0,
  autoConfirmSingle: true
})

const emit = defineEmits<Emits>()
const userInfoStore = useUserInfoStore()

const dialogVisible = ref(false)
const searchKeyword = ref('')
const loading = ref(false)
const userList = ref<UserInfo[]>([])
const selectedUsers = ref<UserInfo[]>([])
const inputRef = ref<InstanceType<typeof ElInput> | null>(null)

const multiple = computed(() => props.multiple)
const maxCount = computed(() => props.maxCount)
const autoConfirmSingle = computed(() => props.autoConfirmSingle)

watch(
  () => props.modelValue,
  async (newValue) => {
    dialogVisible.value = newValue
    if (!newValue) {
      resetSearchState()
      return
    }
    await initializeSelectedUsers()
    resetSearchState()
  }
)

watch(dialogVisible, (newValue) => {
  emit('update:modelValue', newValue)
})

const handleDialogOpened = async () => {
  await nextTick()
  await nextTick()

  const inputEl = (inputRef.value as any)?.$el?.querySelector('input') as HTMLInputElement | undefined
  if (inputEl) {
    inputEl.focus()
    setTimeout(() => inputEl.focus(), 100)
  }
}

const normalizeUsernames = (value: string | null): string[] => {
  if (!value) {
    return []
  }
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

const initializeSelectedUsers = async () => {
  const usernames = normalizeUsernames(props.initialUsernames)
  if (usernames.length === 0) {
    selectedUsers.value = []
    return
  }

  try {
    const loadedUsers = await Promise.all(
      usernames.map(async (username) => {
        try {
          return await userInfoStore.getUserInfo(username)
        } catch (error) {
          Logger.error('UserPickerDialog', '加载用户信息失败', { username, error })
          return null
        }
      })
    )

    const validUsers = loadedUsers.filter((user): user is UserInfo => !!user)
    selectedUsers.value = multiple.value ? validUsers : validUsers.slice(0, 1)
  } catch (error) {
    Logger.error('UserPickerDialog', '初始化已选用户失败', { error })
    selectedUsers.value = []
  }
}

const isUserSelected = (user: UserInfo): boolean => {
  return selectedUsers.value.some((item) => item.username === user.username)
}

const handleToggleUser = (user: UserInfo) => {
  if (!multiple.value) {
    selectedUsers.value = [user]
    return
  }

  if (isUserSelected(user)) {
    selectedUsers.value = selectedUsers.value.filter((item) => item.username !== user.username)
    return
  }

  if (maxCount.value > 0 && selectedUsers.value.length >= maxCount.value) {
    return
  }

  selectedUsers.value = [...selectedUsers.value, user]
}

const handlePickUser = (user: UserInfo) => {
  if (multiple.value) {
    handleToggleUser(user)
    return
  }

  selectedUsers.value = [user]
  if (autoConfirmSingle.value) {
    emit('confirm', [user])
    handleClose()
  }
}

const handleRemoveUser = (user: UserInfo) => {
  selectedUsers.value = selectedUsers.value.filter((item) => item.username !== user.username)
}

const handleClearAll = () => {
  selectedUsers.value = []
}

const handleConfirm = () => {
  emit('confirm', [...selectedUsers.value])
  handleClose()
}

const resetSearchState = () => {
  searchKeyword.value = ''
  userList.value = []
}

const handleClose = () => {
  dialogVisible.value = false
  resetSearchState()
}

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
      Logger.debug('UserPickerDialog', '搜索用户响应', { keyword, response })
      userList.value = response.users || []
    } catch (error) {
      Logger.error('UserPickerDialog', '搜索用户失败', { keyword, error })
      userList.value = []
    } finally {
      loading.value = false
    }
  }, 300)
}

const handleClearSearch = () => {
  resetSearchState()
}
</script>

<style scoped>
:deep(.entity-search-dialog-shell) {
  border-radius: 18px;
  overflow: hidden;
}

:deep(.entity-search-dialog-shell .el-dialog__header) {
  padding: 18px 22px 0;
}

:deep(.entity-search-dialog-shell .el-dialog__body) {
  padding: 18px 22px 12px;
}

:deep(.entity-search-dialog-shell .el-dialog__footer) {
  padding: 0 22px 20px;
}

.user-picker-dialog-search {
  margin-bottom: 18px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.user-picker-dialog-search :deep(.el-input__wrapper) {
  min-height: 42px;
  border-radius: 12px;
  box-shadow: none;
}

.dialog-status {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.status-chip {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.status-chip-active {
  background: rgba(24, 144, 255, 0.12);
  color: var(--el-color-primary);
}

.user-picker-dialog-selected {
  margin-bottom: 18px;
  padding: 14px;
  background:
    linear-gradient(180deg, rgba(24, 144, 255, 0.06), rgba(24, 144, 255, 0)),
    var(--el-fill-color-lighter);
  border-radius: 14px;
  border: 1px solid rgba(24, 144, 255, 0.12);
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
  padding: 5px 9px;
  background-color: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 999px;
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

.user-picker-dialog-list {
  min-height: 300px;
  max-height: 400px;
  overflow-y: auto;
  padding-right: 2px;
}

.user-picker-dialog-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 40px;
  color: var(--el-text-color-secondary);
}

.user-picker-dialog-empty {
  padding: 40px 0;
}

.user-picker-dialog-items {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.user-picker-dialog-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  background:
    linear-gradient(180deg, rgba(24, 144, 255, 0.02), rgba(24, 144, 255, 0)),
    var(--el-bg-color);
  cursor: pointer;
  transition: all 0.2s;
}

.user-picker-dialog-item:hover {
  border-color: rgba(24, 144, 255, 0.24);
  background-color: var(--el-fill-color-light);
  transform: translateY(-1px);
}

.user-picker-dialog-item.is-selected {
  border-color: rgba(24, 144, 255, 0.34);
  background: linear-gradient(135deg, rgba(24, 144, 255, 0.12), rgba(24, 144, 255, 0.04));
  box-shadow: 0 8px 20px rgba(24, 144, 255, 0.08);
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

.user-picker-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

@media (max-width: 768px) {
  .user-picker-dialog-item {
    align-items: flex-start;
  }
}
</style>
