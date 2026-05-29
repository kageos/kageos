<template>
  <el-dialog
    v-model="dialogVisible"
    title="新建文档"
    width="460px"
    class="workspace-create-docs-dialog"
    :close-on-click-modal="false"
    @close="$emit('close')"
  >
    <el-form :model="form" label-position="top" class="create-docs-form" data-testid="create-docs-dialog">
      <el-form-item label="标题" class="create-docs-title">
        <el-input
          v-model="form.name"
          placeholder="未命名文档"
          maxlength="100"
          clearable
          autofocus
          data-testid="create-docs-name"
          @keyup.enter="$emit('submit')"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <span class="dialog-footer">
        <el-button data-testid="create-docs-cancel" @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" data-testid="create-docs-submit" @click="$emit('submit')">
          开始编辑
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ServiceTree as ServiceTreeType } from '@/architecture/domain/types'

interface CreateDocsForm {
  name: string
}

const props = defineProps<{
  visible: boolean
  parentNode: ServiceTreeType | null
  form: CreateDocsForm
  creating: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'submit'): void
  (e: 'close'): void
}>()

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value)
})
</script>

<style scoped lang="scss">
.create-docs-form {
  padding-top: 2px;
}

.create-docs-title {
  margin-bottom: 0;

  :deep(.el-form-item__label) {
    color: var(--text-secondary, #64748b);
    font-weight: 600;
    line-height: 1.2;
    margin-bottom: 10px;
  }

  :deep(.el-input__wrapper) {
    min-height: 44px;
    border-radius: 8px;
    box-shadow: 0 0 0 1px var(--border-primary, #d7dde8) inset;
  }

  :deep(.el-input__inner) {
    font-size: 16px;
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
