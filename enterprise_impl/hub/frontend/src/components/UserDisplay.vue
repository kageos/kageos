<!--
  UserDisplay - 用户展示组件（Hub 版本）
  功能：
  - 显示用户头像和名称
  - 支持从 OS API 获取用户信息
  
  显示风格：
  - horizontal：水平布局，头像在左，名称在右
  - vertical：垂直布局，头像在上，名称在下
-->
<template>
  <div class="user-display-wrapper">
    <div class="user-display-simple" :class="[sizeClass, layoutClass]">
      <el-avatar 
        :src="avatarUrl" 
        :size="avatarSize"
        class="user-avatar"
      >
        {{ actualUserInfo?.username?.[0]?.toUpperCase() || props.username?.[0]?.toUpperCase() || 'U' }}
      </el-avatar>
      <span class="user-name">{{ displayName }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, ref } from 'vue'
import { ElAvatar } from 'element-plus'
import type { UserInfo } from '@/api/user'
import { useUserInfoStore } from '@/stores/userInfo'

interface Props {
  /** 用户名 */
  username?: string | null
  /** 用户信息对象（如果已有，直接使用） */
  userInfo?: UserInfo | null
  /** 显示风格：horizontal（水平布局，头像在左名称在右）或 vertical（垂直布局，头像在上名称在下） */
  layout?: 'horizontal' | 'vertical'
  /** 头像大小：small(24px) | medium(32px) | large(48px) | 自定义数字 */
  size?: 'small' | 'medium' | 'large' | number
}

const props = withDefaults(defineProps<Props>(), {
  username: null,
  userInfo: null,
  layout: 'horizontal',
  size: 'medium',
})

const userInfoStore = useUserInfoStore()

// 🔥 使用 ref 存储用户信息，确保响应式更新
const cachedUserInfo = ref<UserInfo | null>(null)

// 🔥 更新缓存的用户信息
const updateCachedUserInfo = async () => {
  // 优先使用 props.userInfo
  if (props.userInfo) {
    cachedUserInfo.value = props.userInfo
    return
  }
  
  // 如果有 username，从 store 中获取（预加载已完成，store 中肯定有缓存）
  if (props.username) {
    try {
      // 🔥 直接从 store 读取（预加载已完成，这里只是从缓存中读取，不会调用接口）
      const user = await userInfoStore.getUserInfo(props.username)
      cachedUserInfo.value = user
    } catch (error) {
      console.error('[UserDisplay] 从 store 加载用户信息失败', error)
      cachedUserInfo.value = null
    }
    return
  }
  
  cachedUserInfo.value = null
}

// 🔥 监听 userInfo 和 username 的变化，更新缓存的用户信息
watch([() => props.userInfo, () => props.username], () => {
  updateCachedUserInfo()
}, { immediate: true, deep: false })

// 🔥 用户信息（从缓存的 ref 中获取）
const actualUserInfo = computed(() => {
  return cachedUserInfo.value
})

// 计算头像大小
const avatarSize = computed(() => {
  if (typeof props.size === 'number') {
    return props.size
  }
  const sizeMap: Record<'small' | 'medium' | 'large', number> = {
    small: 24,
    medium: 32,
    large: 48,
  }
  return sizeMap[props.size as 'small' | 'medium' | 'large']
})

// 计算尺寸类名
const sizeClass = computed(() => {
  if (typeof props.size === 'number') {
    return ''
  }
  return `user-display-${props.size}`
})

// 计算布局类名
const layoutClass = computed(() => {
  return `user-layout-${props.layout}`
})

// 格式化用户显示名称：username（nickname）或 username
const formatUserDisplayName = (user: UserInfo | null): string => {
  if (!user) return ''
  return user.nickname ? `${user.username}（${user.nickname}）` : user.username
}

// 计算显示名称
const displayName = computed(() => {
  const user = cachedUserInfo.value
  if (user) {
    return formatUserDisplayName(user)
  }
  if (props.username) {
    return props.username
  }
  return '-'
})

// 计算头像 URL（处理空字符串和无效 URL）
const avatarUrl = computed(() => {
  const user = actualUserInfo.value
  if (user && user.avatar && user.avatar.trim()) {
    const url = user.avatar.trim()
    // 调试：打印头像 URL
    console.log('[UserDisplay] 头像 URL:', url, '用户:', user.username)
    return url
  }
  // 调试：没有头像
  if (user) {
    console.log('[UserDisplay] 用户没有头像:', user.username)
  }
  return undefined // el-avatar 会显示默认头像或首字母
})
</script>

<style scoped>
.user-display-wrapper {
  display: inline-flex;
  align-items: center;
}

.user-display-simple {
  display: flex;
}

/* 水平布局：头像在左，名称在右 */
.user-layout-horizontal {
  flex-direction: row;
  align-items: center;
  gap: 8px;
}

/* 垂直布局：头像在上，名称在下 */
.user-layout-vertical {
  flex-direction: column;
  align-items: center;
  gap: 6px;
  justify-content: center;
}

.user-display-simple .user-avatar {
  flex-shrink: 0;
}

.user-display-simple .user-name {
  font-size: 14px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

/* 垂直布局下的名称样式 */
.user-layout-vertical .user-name {
  font-size: 12px;
  text-align: center;
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.2;
  display: block;
}

.user-avatar {
  flex-shrink: 0;
}
</style>

