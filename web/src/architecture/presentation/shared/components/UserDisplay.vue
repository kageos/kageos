<!--
  UserDisplay - 通用用户展示组件
  功能：
  - simple：简单模式，只显示头像和名称（不可点击）
  - card：卡片模式，显示头像和名称，点击弹窗显示详情（适用于表格、操作日志等空间有限的地方）
  - rich：详细模式，直接展示完整用户信息（组织架构、个性签名、邮箱等，适用于有足够空间的地方）
  
  显示风格：
  - horizontal：水平布局，头像在左，名称在右（适用于 table、详情字段等）
  - vertical：垂直布局，头像在上，名称在下（适用于文件上传用户等）
  
  使用场景：
  - simple：只读展示，不需要交互
  - card：表格、操作日志等空间有限的地方，点击查看详情
  - rich：函数详情右侧等有足够空间的地方，直接展示完整信息
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
    
    <!-- 卡片模式：点击头像弹出用户信息卡片 -->
    <div v-else-if="mode === 'card'" class="user-display-card" :class="[sizeClass, layoutClass]">
      <el-popover
        v-if="actualUserInfo"
        placement="bottom-start"
        :width="380"
        trigger="click"
        popper-class="user-info-popover"
      >
        <template #reference>
          <div class="user-display-simple user-display-card-trigger">
            <el-avatar 
              :src="actualUserInfo.avatar" 
              :size="avatarSize"
              class="user-avatar"
            >
              {{ actualUserInfo.username?.[0]?.toUpperCase() || 'U' }}
            </el-avatar>
            <span class="user-name">{{ displayName }}</span>
          </div>
        </template>
        <UserDetailCard :user-info="actualUserInfo" />
      </el-popover>
      <!-- 如果没有用户信息，只显示头像和名称（不可点击） -->
      <div v-else class="user-display-simple" :class="[sizeClass, layoutClass]">
        <el-avatar 
          :size="avatarSize"
          class="user-avatar"
        >
          {{ displayName?.[0]?.toUpperCase() || 'U' }}
        </el-avatar>
        <span class="user-name">{{ displayName }}</span>
      </div>
    </div>
    
    <!-- 详细模式：直接展示完整用户信息 -->
    <div v-else-if="mode === 'rich'" class="user-display-rich">
      <UserDetailCard :user-info="actualUserInfo" :compact="false" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, ref } from 'vue'
import { ElAvatar, ElPopover } from 'element-plus'
import type { UserInfo } from '@/architecture/domain/types'
import { formatUserDisplayName } from '@/architecture/domain/utils/userInfo'
import { useUserInfoStore } from '@/architecture/presentation/context/appStoresContext'
import UserDetailCard from './UserDetailCard.vue'

interface Props {
  /** 用户信息对象 */
  userInfo?: UserInfo | null
  /** 用户名（当 userInfo 不存在时使用） */
  username?: string | null
  /** 显示模式：simple（简单模式，只显示头像和名称）| card（卡片模式，点击弹窗显示详情）| rich（详细模式，直接展示完整信息） */
  mode?: 'simple' | 'card' | 'rich'
  /** 显示风格：horizontal（水平布局，头像在左名称在右）或 vertical（垂直布局，头像在上名称在下） */
  layout?: 'horizontal' | 'vertical'
  /** 头像大小：small(24px) | medium(32px) | large(48px) | 自定义数字 */
  size?: 'small' | 'medium' | 'large' | number
}

const props = withDefaults(defineProps<Props>(), {
  userInfo: null,
  username: null,
  mode: 'simple',
  layout: 'horizontal',
  size: 'medium',
})

const userInfoStore = useUserInfoStore()

// 🔥 使用 ref 存储用户信息，确保响应式更新
// 问题：Vue 无法追踪 Map 内部的变化，所以使用 ref 来存储用户信息
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

// 🔥 用户信息（从缓存的 ref 中获取）
const actualUserInfo = computed(() => {
  return cachedUserInfo.value
})

// 🔥 监听 userInfo 和 username 的变化，更新缓存的用户信息
watch([() => props.userInfo, () => props.username], () => {
  updateCachedUserInfo()
}, { immediate: true, deep: false })

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

.user-display-card-trigger {
  min-width: 0;
  padding: 3px 7px 3px 3px;
  border-radius: 999px;
  cursor: pointer;
  transition: background-color 0.18s ease, box-shadow 0.18s ease, color 0.18s ease, transform 0.18s ease;
}

.user-display-card-trigger:hover {
  background: rgba(var(--el-color-primary-rgb), 0.1);
  box-shadow: 0 0 0 1px rgba(var(--el-color-primary-rgb), 0.16);
  transform: translateY(-1px);
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
  font-weight: 600;
  white-space: nowrap;
}

.user-display-card-trigger:hover .user-name {
  color: var(--el-color-primary);
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

/* 详细模式 */
.user-display-rich {
  width: 100%;
}
</style>
