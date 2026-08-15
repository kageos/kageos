<template>
  <el-avatar
    :size="size"
    :shape="shape"
    class="kage-user-avatar"
    :class="{ 'is-logo-fallback': isFallback }"
  >
    <img
      class="kage-user-avatar__img"
      :src="displaySrc"
      :alt="alt"
      :style="{ objectFit: displayFit }"
      @error="handleImageError"
    />
  </el-avatar>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElAvatar } from 'element-plus'
import { DEFAULT_BUILTIN_USER_AVATAR } from '@/architecture/domain/utils/builtinUserAvatar'

type AvatarSize = number | 'large' | 'default' | 'small'
type AvatarShape = 'circle' | 'square'
type ObjectFit = 'fill' | 'contain' | 'cover' | 'none' | 'scale-down'

const props = withDefaults(defineProps<{
  src?: string | null
  size?: AvatarSize
  shape?: AvatarShape
  fit?: ObjectFit
  alt?: string
  fallback?: string
}>(), {
  src: null,
  size: 'default',
  shape: 'circle',
  fit: 'cover',
  alt: 'User avatar',
  fallback: DEFAULT_BUILTIN_USER_AVATAR,
})

const loadFailed = ref(false)
const normalizedSrc = computed(() => props.src?.trim() || '')

watch(normalizedSrc, () => {
  loadFailed.value = false
})

const isFallback = computed(() => loadFailed.value || !normalizedSrc.value)
const displaySrc = computed(() => isFallback.value ? props.fallback : normalizedSrc.value)
const displayFit = computed(() => isFallback.value ? 'contain' : props.fit)

function handleImageError() {
  if (displaySrc.value !== props.fallback) {
    loadFailed.value = true
  }
}
</script>

<style scoped>
.kage-user-avatar {
  background: color-mix(in srgb, var(--el-fill-color-light) 74%, transparent);
}

.kage-user-avatar__img {
  width: 100%;
  height: 100%;
  display: block;
}

.kage-user-avatar.is-logo-fallback .kage-user-avatar__img {
  padding: 0;
}
</style>
