<template>
  <el-dialog
    v-model="visible"
    class="workspace-list-dialog-shell"
    :title="forceSelect ? '请选择工作空间' : '工作空间列表'"
    width="900px"
    :append-to-body="true"
    :close-on-click-modal="false"
    :show-close="!forceSelect"
    :before-close="forceSelect ? handleBeforeClose : undefined"
    @close="handleClose"
  >
    <WorkspaceListPanel
      :current-app="currentApp"
      :force-select="forceSelect"
      :visible="visible"
      surface="dialog"
      @switch-app="handleSwitchApp"
      @create-app="$emit('create-app')"
      @update-app="$emit('update-app', $event)"
      @delete-app="$emit('delete-app', $event)"
      @close="handleClose"
    />
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { App } from '@/architecture/domain/types'
import WorkspaceListPanel from './WorkspaceListPanel.vue'

interface Props {
  modelValue: boolean
  currentApp: App | null
  /** 为 true 时不可关闭弹窗，必须选择或创建工作空间（如从 /workspace/:user 进入时） */
  forceSelect?: boolean
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'switch-app', app: App): void
  (e: 'create-app'): void
  (e: 'update-app', app: App): void
  (e: 'delete-app', app: App): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value)
})

const handleSwitchApp = (app: App) => {
  emit('switch-app', app)
  handleClose()
}

const handleClose = () => {
  visible.value = false
}

// 强制选择模式下阻止关闭（ESC 等）：不调用 done() 弹窗不会关闭
const handleBeforeClose = (_done: () => void) => {
  // 保持打开，直到用户选择或创建工作空间。
}
</script>

<style scoped>
:deep(.workspace-list-dialog-shell) {
  border-radius: 28px;
  background: var(--app-auth-card-bg);
  border: 1px solid var(--app-auth-card-border);
  box-shadow: var(--app-auth-card-shadow);
  overflow: hidden;
}

:deep(.workspace-list-dialog-shell .el-dialog__header) {
  padding: 28px 32px 12px;
}

:deep(.workspace-list-dialog-shell .el-dialog__title) {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
}

:deep(.workspace-list-dialog-shell .el-dialog__body) {
  padding: 0 32px 28px;
  background: var(--app-auth-surface-bg);
}
</style>
