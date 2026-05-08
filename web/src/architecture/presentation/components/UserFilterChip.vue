<template>
  <div class="user-filter-chip">
    <el-avatar :src="avatar || undefined" :size="24" class="user-avatar">
      {{ initial }}
    </el-avatar>
    <span class="user-name">{{ label }}</span>
    <el-icon class="user-chip-close" @click.stop="emit('remove')">
      <Close />
    </el-icon>
  </div>
</template>

<script setup lang="ts">
import { ElAvatar, ElIcon } from 'element-plus'
import { Close } from '@element-plus/icons-vue'

interface Props {
  label: string
  avatar?: string | null
  initial?: string
}

withDefaults(defineProps<Props>(), {
  avatar: null,
  initial: 'U'
})

const emit = defineEmits<{
  (e: 'remove'): void
}>()
</script>

<style scoped>
.user-filter-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  position: relative;
  padding-right: 20px;
  min-width: 0;
  max-width: 160px;
}

.user-avatar {
  flex-shrink: 0;
  width: 24px !important;
  height: 24px !important;
}

.user-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 14px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

.user-chip-close {
  position: absolute;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 16px;
  height: 16px;
  cursor: pointer;
  color: var(--el-text-color-secondary);
  transition: color 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.user-chip-close:hover {
  color: var(--el-text-color-primary);
}
</style>
