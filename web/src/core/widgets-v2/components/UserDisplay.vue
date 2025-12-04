<!--
  UserDisplay - 通用用户展示组件
  功能：
  - 简单模式：只显示头像和名称（用于列表、详情等）
  - 详细模式：点击头像显示完整用户信息卡片（使用 el-tooltip，简单直接）
  
  显示风格：
  - horizontal：水平布局，头像在左，名称在右（适用于 table、详情字段等）
  - vertical：垂直布局，头像在上，名称在下（适用于文件上传用户等）
  
  使用场景：
  - Form 输出用户字段（horizontal）
  - Table 表格中显示用户（horizontal）
  - 详情中显示用户信息（horizontal）
  - 文件上传用户显示（vertical）
-->
<template>
  <div class="user-display-wrapper">
    <!-- 简单模式：只显示头像和名称 -->
    <div v-if="mode === 'simple'" class="user-display-simple" :class="[sizeClass, layoutClass]">
      <el-avatar 
        v-if="actualUserInfo" 
        :src="actualUserInfo.avatar" 
        :size="avatarSize"
        class="user-avatar"
      >
        {{ actualUserInfo.username?.[0]?.toUpperCase() || 'U' }}
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
    
    <!-- 详细模式：暂时只显示头像和名称（弹窗功能已移除，后续再加） -->
    <div v-else-if="mode === 'card'" class="user-display-simple" :class="[sizeClass, layoutClass]">
      <el-avatar 
        v-if="actualUserInfo" 
        :src="actualUserInfo.avatar" 
        :size="avatarSize"
        class="user-avatar"
      >
        {{ actualUserInfo.username?.[0]?.toUpperCase() || 'U' }}
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
  </div>
</template>

<script setup lang="ts">
import { computed, watch, ref, onUnmounted } from 'vue'
import { ElAvatar, ElMessage } from 'element-plus'
import type { UserInfo } from '@/types'
import { formatUserDisplayName } from '@/utils/userInfo'
import { useUserInfoStore } from '@/stores/userInfo'

interface Props {
  /** 用户信息对象 */
  userInfo?: UserInfo | null
  /** 用户名（当 userInfo 不存在时使用） */
  username?: string | null
  /** 显示模式：simple（简单模式，只显示头像和名称）或 card（详细模式，hover 显示卡片） */
  mode?: 'simple' | 'card'
  /** 显示风格：horizontal（水平布局，头像在左名称在右）或 vertical（垂直布局，头像在上名称在下） */
  layout?: 'horizontal' | 'vertical'
  /** 头像大小：small(24px) | medium(32px) | large(48px) | 自定义数字 */
  size?: 'small' | 'medium' | 'large' | number
  /** 用户信息 Map（用于从缓存中获取） */
  userInfoMap?: Map<string, UserInfo> | null
}

const props = withDefaults(defineProps<Props>(), {
  userInfo: null,
  username: null,
  mode: 'simple',
  layout: 'horizontal',
  size: 'medium',
  userInfoMap: null,
})

const userInfoStore = useUserInfoStore()

// 🔥 使用 ref 存储用户信息，确保响应式更新
// 问题：Vue 无法追踪 Map 内部的变化，所以使用 ref 来存储用户信息
const cachedUserInfo = ref<UserInfo | null>(null)

// 🔥 更新缓存的用户信息
const updateCachedUserInfo = () => {
  if (props.userInfo) {
    cachedUserInfo.value = props.userInfo
    return
  }
  if (props.username && props.userInfoMap && props.userInfoMap.has(props.username)) {
    cachedUserInfo.value = props.userInfoMap.get(props.username) || null
    return
  }
  cachedUserInfo.value = null
}

// 🔥 用户信息（从缓存的 ref 中获取）
const actualUserInfo = computed(() => {
  return cachedUserInfo.value
})

// 🔥 监听 userInfo 和 username 变化，更新缓存的用户信息
watch([() => props.userInfo, () => props.username], () => {
  updateCachedUserInfo()
}, { immediate: true })

// 🔥 监听 userInfoMap 的变化（通过轮询检查 Map 内容）
// 注意：Vue 无法直接追踪 Map 内部的变化，所以使用轮询来检查
// 这是一个 workaround，但性能影响很小（只在有 username 且没有用户信息时检查）
let mapCheckInterval: number | null = null
watch(() => [props.username, props.userInfoMap], ([newUsername, newUserInfoMap]) => {
  // 清除旧的定时器
  if (mapCheckInterval !== null) {
    clearInterval(mapCheckInterval)
    mapCheckInterval = null
  }
  
  // 立即检查一次
  updateCachedUserInfo()
  
  // 如果还没有用户信息，且 userInfoMap 存在，设置定时器定期检查
  // 最多检查 10 次，每次 200ms（总共 2 秒）
  if (newUsername && newUserInfoMap && !cachedUserInfo.value) {
    let checkCount = 0
    mapCheckInterval = window.setInterval(() => {
      checkCount++
      updateCachedUserInfo()
      // 如果已经获取到用户信息，或者检查次数达到上限，停止检查
      if (cachedUserInfo.value || checkCount >= 10) {
        if (mapCheckInterval !== null) {
          clearInterval(mapCheckInterval)
          mapCheckInterval = null
        }
      }
    }, 200)
  }
}, { immediate: true })

// 🔥 监听 username 变化，自动加载用户信息
watch(() => props.username, async (newUsername) => {
  if (newUsername && !cachedUserInfo.value) {
    // 如果 userInfoMap 中没有，尝试从 store 加载
    if (!props.userInfoMap || !props.userInfoMap.has(newUsername)) {
      try {
        const users = await userInfoStore.batchGetUserInfo([newUsername])
        if (users && users.length > 0) {
          // 更新到 userInfoMap（如果存在）
          if (props.userInfoMap) {
            props.userInfoMap.set(newUsername, users[0])
          }
          // 🔥 手动更新缓存
          cachedUserInfo.value = users[0]
        }
      } catch (error) {
        console.error('[UserDisplay] 加载用户信息失败', error)
      }
    }
  }
}, { immediate: true })

// 🔥 组件卸载时清除定时器
onUnmounted(() => {
  if (mapCheckInterval !== null) {
    clearInterval(mapCheckInterval)
    mapCheckInterval = null
  }
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

// 计算显示名称
const displayName = computed(() => {
  const user = actualUserInfo.value
  if (user) {
    return formatUserDisplayName(user)
  }
  if (props.username) {
    return props.username
  }
  return '-'
})

// 复制用户信息（手动复制，由用户点击按钮触发）
// 注意：弹窗功能已移除，此函数暂时保留供后续使用
const handleCopyUserInfo = (): void => {
  const user = actualUserInfo.value
  if (user) {
    const copyText = user.nickname 
      ? `${user.username}(${user.nickname})`
      : user.username
    
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
</script>

<style scoped>
.user-display-wrapper {
  display: inline-flex;
  align-items: center;
}

/* 简单模式 */
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
