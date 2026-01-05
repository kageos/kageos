<!--
  UserDetailCard - 用户详情卡片（用于 Popover）
  功能：
  - 显示用户完整信息（头像、用户名、昵称、邮箱、性别、签名等）
  - 显示组织架构信息（部门完整名称路径）
  - 显示 Leader 信息
  - 显示状态和注册信息
-->
<template>
  <div v-if="userInfo" class="user-detail-card" :class="{ compact: props.compact }">
    <!-- 用户头像和基本信息 -->
    <div class="user-header">
      <el-avatar :size="72" :src="userInfo.avatar" class="user-avatar-large">
        {{ userInfo.username?.[0]?.toUpperCase() || 'U' }}
      </el-avatar>
      <div class="user-basic-info">
        <div class="user-name-primary">
          {{ formatUserDisplayName(userInfo) }}
        </div>
        <div class="user-username">@{{ userInfo.username }}</div>
        <div v-if="userInfo.email" class="user-email">
          <el-icon class="info-icon"><Message /></el-icon>
          <span>{{ userInfo.email }}</span>
        </div>
        <div v-if="userInfo.gender" class="user-gender">
          <el-icon class="info-icon"><User /></el-icon>
          <span>{{ genderText }}</span>
        </div>
      </div>
    </div>

    <!-- 个人签名 -->
    <div v-if="userInfo.signature" class="signature-section">
      <el-icon class="section-icon"><EditPen /></el-icon>
      <span class="signature-text">{{ userInfo.signature }}</span>
    </div>

    <!-- 组织架构信息 -->
    <div v-if="userInfo.department_full_name_path || userInfo.department_name || userInfo.department_full_path" class="info-item">
      <div class="info-label">
        <el-icon class="info-icon"><OfficeBuilding /></el-icon>
        <span>组织架构</span>
      </div>
      <div class="info-value">
        {{ userInfo.department_full_name_path || userInfo.department_name || userInfo.department_full_path }}
      </div>
    </div>

    <!-- Leader 信息（仅在紧凑模式下显示） -->
    <div v-if="props.compact && (userInfo.leader_display_name || userInfo.leader_username)" class="info-item">
      <div class="info-label">
        <el-icon class="info-icon"><UserFilled /></el-icon>
        <span>直接上级</span>
      </div>
      <div class="info-value">
        {{ userInfo.leader_display_name || userInfo.leader_username }}
      </div>
    </div>

    <!-- 状态和注册信息（仅在紧凑模式下显示） -->
    <div v-if="props.compact" class="info-footer">
      <el-tag :type="statusTagType" size="small" class="status-tag">
        {{ statusText }}
      </el-tag>
      <span v-if="userInfo.register_type" class="register-type">
        {{ registerTypeText }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElAvatar, ElIcon, ElTag } from 'element-plus'
import { OfficeBuilding, UserFilled, Message, User, EditPen } from '@element-plus/icons-vue'
import type { UserInfo } from '@/types'
import { formatUserDisplayName } from '@/utils/userInfo'

interface Props {
  userInfo: UserInfo | null
  /** 是否紧凑模式（用于弹窗，默认 true） */
  compact?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  compact: true
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

// 状态文本和标签类型
const statusText = computed(() => {
  const statusMap: Record<string, string> = {
    'active': '已激活',
    'pending': '待验证',
    'inactive': '已停用',
    'banned': '已封禁'
  }
  return statusMap[props.userInfo?.status || ''] || props.userInfo?.status || '未知'
})

const statusTagType = computed(() => {
  if (props.userInfo?.status === 'active') {
    return 'success'
  } else if (props.userInfo?.status === 'pending') {
    return 'warning'
  } else if (props.userInfo?.status === 'inactive' || props.userInfo?.status === 'banned') {
    return 'danger'
  }
  return 'info'
})

// 注册类型文本
const registerTypeText = computed(() => {
  const typeMap: Record<string, string> = {
    'email': '邮箱注册',
    'wechat': '微信注册',
    'github': 'GitHub注册',
    'google': 'Google注册',
    'qq': 'QQ注册',
    'phone': '手机号注册'
  }
  return typeMap[props.userInfo?.register_type || ''] || props.userInfo?.register_type || ''
})
</script>

<style scoped>
.user-detail-card {
  padding: 16px;
  min-width: 320px;
  max-width: 400px;
}

/* 非紧凑模式（用于直接展示，rich 模式） */
.user-detail-card:not(.compact) {
  padding: 20px;
  min-width: auto;
  max-width: none;
  width: 100%;
  background: var(--el-bg-color);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}

.user-header {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.user-avatar-large {
  flex-shrink: 0;
  border: 2px solid var(--el-border-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.user-basic-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.user-name-primary {
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.4;
  word-break: break-word;
}

.user-username {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}

.user-email,
.user-gender {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--el-text-color-regular);
  line-height: 1.4;
}

.info-icon {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}

.signature-section {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 16px;
  padding: 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
  border-left: 3px solid var(--el-color-primary);
}

.section-icon {
  font-size: 16px;
  color: var(--el-color-primary);
  margin-top: 2px;
  flex-shrink: 0;
}

.signature-text {
  font-size: 13px;
  color: var(--el-text-color-regular);
  line-height: 1.6;
  flex: 1;
  word-break: break-word;
}

.info-item {
  margin-bottom: 12px;
  
  &:last-of-type {
    margin-bottom: 0;
  }
}

.info-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
}

.info-value {
  font-size: 14px;
  color: var(--el-text-color-primary);
  line-height: 1.5;
  padding-left: 20px;
  word-break: break-word;
}

.info-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.status-tag {
  flex-shrink: 0;
}

.register-type {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  flex: 1;
  text-align: right;
}
</style>

<style>
/* Popover 全局样式 */
.user-info-popover {
  padding: 0 !important;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
}
</style>

