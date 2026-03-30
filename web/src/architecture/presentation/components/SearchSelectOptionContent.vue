<template>
  <div v-if="userInfo" class="user-option">
    <el-avatar :src="userInfo.avatar" :size="24" class="user-avatar">
      {{ userInitial }}
    </el-avatar>
    <span class="user-name">{{ userInfo.username }}</span>
    <span v-if="userInfo.nickname" class="user-nickname">({{ userInfo.nickname }})</span>
  </div>
  <div v-else-if="showColorIndicator" class="option-with-color">
    <span
      v-if="hasColorStyle"
      class="option-color-indicator"
      :style="colorStyle"
    />
    <span>{{ label }}</span>
  </div>
  <span v-else>{{ label }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElAvatar } from 'element-plus'

interface UserInfoLike {
  avatar?: string
  username?: string
  nickname?: string
}

interface Props {
  label: string
  userInfo?: UserInfoLike | null
  showColorIndicator?: boolean
  colorStyle?: Record<string, string>
}

const props = withDefaults(defineProps<Props>(), {
  userInfo: null,
  showColorIndicator: false,
  colorStyle: () => ({})
})

const userInitial = computed(() => {
  return props.userInfo?.username?.[0]?.toUpperCase() || 'U'
})

const hasColorStyle = computed(() => {
  return Object.keys(props.colorStyle || {}).length > 0
})
</script>

<style scoped>
.user-option {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  flex-shrink: 0;
}

.user-name {
  flex: 1;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.user-nickname {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.option-with-color {
  display: flex;
  align-items: center;
}

.option-color-indicator {
  display: inline-block;
  width: 12px;
  height: 12px;
  min-width: 12px;
  min-height: 12px;
  margin-right: 8px;
  border-radius: 2px;
  flex-shrink: 0;
  border: none;
  vertical-align: middle;
  filter: brightness(0.95) saturate(0.9);
  opacity: 0.9;
}
</style>
