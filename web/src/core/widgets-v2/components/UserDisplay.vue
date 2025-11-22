<!--
  UserDisplay - 通用用户展示组件
  功能：
  - 简单模式：只显示头像和名称（用于列表、详情等）
  - 详细模式：点击头像显示完整用户信息卡片（带 popover）
  
  使用场景：
  - Form 输出用户字段
  - Table 表格中显示用户
  - 详情中显示用户信息
  - 文件上传用户显示
-->
<template>
  <div class="user-display-wrapper">
    <!-- 简单模式：只显示头像和名称 -->
    <div v-if="mode === 'simple'" class="user-display-simple" :class="sizeClass">
      <el-avatar 
        v-if="userInfo" 
        :src="userInfo.avatar" 
        :size="avatarSize"
        class="user-avatar"
      >
        {{ userInfo.username?.[0]?.toUpperCase() || 'U' }}
      </el-avatar>
      <el-avatar 
        v-else 
        :size="avatarSize"
        class="user-avatar"
      >
        {{ displayName?.[0]?.toUpperCase() || 'U' }}
      </el-avatar>
      <span class="user-name">{{ displayName }}</span>
    </div>
    
    <!-- 详细模式：点击头像显示完整信息卡片 -->
    <span v-else-if="mode === 'card'" class="user-display-card">
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
            :size="avatarSize"
            class="user-avatar user-avatar-clickable"
            @click="handleAvatarClick"
          >
            {{ userInfo.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <el-avatar 
            v-else 
            :size="avatarSize"
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElAvatar, ElPopover, ElButton, ElMessage } from 'element-plus'
import type { UserInfo } from '@/types'
import { formatUserDisplayName } from '@/utils/userInfo'

interface Props {
  /** 用户信息对象 */
  userInfo?: UserInfo | null
  /** 用户名（当 userInfo 不存在时使用） */
  username?: string | null
  /** 显示模式：simple（简单模式，只显示头像和名称）或 card（详细模式，点击显示卡片） */
  mode?: 'simple' | 'card'
  /** 头像大小：small(24px) | medium(32px) | large(48px) | 自定义数字 */
  size?: 'small' | 'medium' | 'large' | number
  /** 用户信息 Map（用于从缓存中获取） */
  userInfoMap?: Map<string, UserInfo> | null
}

const props = withDefaults(defineProps<Props>(), {
  userInfo: null,
  username: null,
  mode: 'simple',
  size: 'medium',
  userInfoMap: null,
})

const showPopover = ref(false)

// 计算头像大小
const avatarSize = computed(() => {
  if (typeof props.size === 'number') {
    return props.size
  }
  const sizeMap = {
    small: 24,
    medium: 32,
    large: 48,
  }
  return sizeMap[props.size]
})

// 计算尺寸类名
const sizeClass = computed(() => {
  if (typeof props.size === 'number') {
    return ''
  }
  return `user-display-${props.size}`
})

// 计算显示名称
const displayName = computed(() => {
  if (props.userInfo) {
    return formatUserDisplayName(props.userInfo)
  }
  if (props.username) {
    return props.username
  }
  return '-'
})

// 处理头像点击（显示用户信息弹窗）
const handleAvatarClick = (): void => {
  if (props.mode === 'card') {
    showPopover.value = !showPopover.value
  }
}

// 复制用户信息
const handleCopyUserInfo = (): void => {
  if (props.userInfo) {
    const copyText = props.userInfo.nickname 
      ? `${props.userInfo.username}(${props.userInfo.nickname})`
      : props.userInfo.username
    
    navigator.clipboard.writeText(copyText).then(() => {
      ElMessage.success('已复制用户信息')
    }).catch(() => {
      ElMessage.error('复制失败')
    })
  } else if (props.username) {
    navigator.clipboard.writeText(props.username).then(() => {
      ElMessage.success('已复制')
    }).catch(() => {
      ElMessage.error('复制失败')
    })
  }
}

// 复制名称
const handleCopyName = (): void => {
  navigator.clipboard.writeText(displayName.value).then(() => {
    ElMessage.success('已复制名称')
  }).catch(() => {
    ElMessage.error('复制失败')
  })
}
</script>

<style scoped>
.user-display-wrapper {
  display: inline-flex;
  align-items: center;
}

/* 简单模式 */
.user-display-simple {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-display-simple .user-avatar {
  flex-shrink: 0;
}

.user-display-simple .user-name {
  font-size: 14px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

/* 详细模式 */
.user-display-card {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  flex-shrink: 0;
}

.user-avatar-clickable {
  cursor: pointer;
  transition: all 0.2s;
}

.user-avatar-clickable:hover {
  opacity: 0.8;
  transform: scale(1.05);
}

.user-name {
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.user-name-clickable {
  cursor: pointer;
  user-select: none;
}

.user-name-clickable:hover {
  color: var(--el-color-primary);
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

.user-avatar-large {
  flex-shrink: 0;
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
</style>

<style>
/* 全局样式：用户信息弹窗 */
.user-info-popover {
  padding: 0 !important;
  z-index: 3000 !important;
}

.user-info-popover .el-popover__reference {
  display: inline-flex;
  align-items: center;
}
</style>

