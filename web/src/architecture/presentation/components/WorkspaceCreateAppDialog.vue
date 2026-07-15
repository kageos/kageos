<template>
  <el-dialog
    v-model="dialogVisible"
    :title="t('workspace.createDialogTitle')"
    width="800px"
    class="workspace-create-app-dialog"
    :close-on-click-modal="false"
    @close="$emit('close')"
  >
    <el-form :model="form" label-width="120px" data-testid="create-app-dialog">
      <el-form-item :label="t('workspace.createName')" required>
        <el-input
          v-model="appName"
          :placeholder="t('workspace.createNamePlaceholder')"
          maxlength="100"
          show-word-limit
          clearable
          data-testid="create-app-name"
        />
      </el-form-item>
      <el-form-item :label="t('workspace.createCode')" required>
        <el-tooltip
          :content="t('workspace.createCodeHelp')"
          placement="top"
        >
          <el-input
            v-model="appCode"
            :placeholder="t('workspace.createCodePlaceholder')"
            maxlength="50"
            show-word-limit
            clearable
            data-testid="create-app-code"
          />
        </el-tooltip>
      </el-form-item>
      <el-form-item :label="t('workspace.createPublicLabel')">
        <el-tooltip
          :content="t('workspace.createPublicTip')"
          placement="top"
        >
          <el-switch v-model="isPublic" />
        </el-tooltip>
      </el-form-item>
      <el-form-item :label="t('workspace.hideUnauthorizedNodes')">
        <el-tooltip
          :content="t('workspace.hideUnauthorizedNodesTip')"
          placement="top"
        >
          <el-switch v-model="hideUnauthorizedNodes" />
        </el-tooltip>
      </el-form-item>
    </el-form>

    <template #footer>
      <span class="dialog-footer">
        <el-button data-testid="create-app-cancel" @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" data-testid="create-app-submit" @click="$emit('submit')" :loading="creating">
          {{ t('common.create') }}
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CreateAppRequest } from '@/architecture/domain/types'

const props = defineProps<{
  visible: boolean
  form: CreateAppRequest
  creating: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'update:form', value: CreateAppRequest): void
  (e: 'submit'): void
  (e: 'close'): void
}>()

const { t } = useI18n()

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value)
})

function updateForm(patch: Partial<CreateAppRequest>): void {
  emit('update:form', {
    ...props.form,
    ...patch,
  })
}

const appName = computed({
  get: () => props.form.name,
  set: (value: string) => updateForm({ name: value })
})

const appCode = computed({
  get: () => props.form.code,
  set: (value: string) => updateForm({ code: value.toLowerCase() })
})

const isPublic = computed({
  get: () => props.form.is_public,
  set: (value: boolean) => updateForm({ is_public: value })
})

const hideUnauthorizedNodes = computed({
  get: () => props.form.hide_unauthorized_nodes,
  set: (value: boolean) => updateForm({ hide_unauthorized_nodes: value })
})
</script>
