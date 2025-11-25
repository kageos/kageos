<!--
  UserDetailDialog - 用户详情对话框
  显示用户的完整信息
-->

<template>
  <el-dialog
    v-model="dialogVisible"
    title="用户详情"
    width="500px"
    :close-on-click-modal="true"
  >
    <div v-if="userInfo" class="user-detail-content">
      <!-- 用户头像和基本信息 -->
      <div class="user-header">
        <el-avatar :size="80" :src="userInfo.avatar" class="user-avatar-large">
          {{ userInfo.username?.[0]?.toUpperCase() || 'U' }}
        </el-avatar>
        <div class="user-basic-info">
          <div class="user-name-primary">
            {{ userInfo.nickname ? `${userInfo.username}(${userInfo.nickname})` : userInfo.username }}
          </div>
          <div class="user-username">@{{ userInfo.username }}</div>
        </div>
      </div>

      <!-- 详细信息 -->
      <el-descriptions :column="1" border class="user-descriptions">
        <el-descriptions-item label="用户名">
          {{ userInfo.username }}
        </el-descriptions-item>
        <el-descriptions-item v-if="userInfo.nickname" label="昵称">
          {{ userInfo.nickname }}
        </el-descriptions-item>
        <el-descriptions-item v-if="userInfo.email" label="邮箱">
          {{ userInfo.email }}
        </el-descriptions-item>
        <el-descriptions-item v-if="userInfo.gender" label="性别">
          {{ genderText }}
        </el-descriptions-item>
        <el-descriptions-item v-if="userInfo.signature" label="个人签名">
          {{ userInfo.signature }}
        </el-descriptions-item>
        <el-descriptions-item label="注册类型">
          {{ userInfo.register_type }}
        </el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="statusTagType" size="small">{{ userInfo.status }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item v-if="userInfo.created_at" label="注册时间">
          {{ formatDate(userInfo.created_at) }}
        </el-descriptions-item>
      </el-descriptions>
    </div>
    <div v-else class="user-detail-loading">
      <el-empty description="用户信息加载中..." />
    </div>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="dialogVisible = false">关闭</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElDialog, ElAvatar, ElDescriptions, ElDescriptionsItem, ElTag, ElButton, ElEmpty } from 'element-plus'
import type { UserInfo } from '@/types'

interface Props {
  modelValue: boolean
  userInfo: UserInfo | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// 性别文本
const genderText = computed(() => {
  const genderMap: Record<string, string> = {
    'male': '男',
    'female': '女',
    'other': '其他',
    '': '未设置'
  }
  return genderMap[props.userInfo?.gender || ''] || '未设置'
})

// 状态标签类型
const statusTagType = computed(() => {
  if (props.userInfo?.status === 'active') {
    return 'success'
  } else if (props.userInfo?.status === 'inactive') {
    return 'info'
  } else if (props.userInfo?.status === 'banned') {
    return 'danger'
  }
  return ''
})

// 格式化日期
function formatDate(dateString: string): string {
  if (!dateString) return '-'
  try {
    const date = new Date(dateString)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  } catch (error) {
    return dateString
  }
}
</script>

<style scoped>
.user-detail-content {
  padding: 10px 0;
}

.user-header {
  display: flex;
  align-items: center;
  gap: 20px;
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.user-avatar-large {
  flex-shrink: 0;
}

.user-basic-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.user-name-primary {
  font-size: 20px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.user-username {
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

.user-descriptions {
  margin-top: 20px;
}

.user-detail-loading {
  padding: 40px 0;
  text-align: center;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style>

