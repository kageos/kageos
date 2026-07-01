<template>
  <el-dialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="520px"
    :close-on-click-modal="false"
    @close="$emit('close')"
  >
    <el-form :model="form" label-width="120px" data-testid="create-directory-dialog">
      <el-form-item :label="t('serviceTree.directoryName')" required>
        <el-input
          v-model="directoryName"
          :placeholder="t('serviceTree.directoryNamePlaceholder')"
          maxlength="50"
          show-word-limit
          clearable
          data-testid="create-directory-name"
        />
      </el-form-item>
      <el-form-item :label="t('serviceTree.directoryCode')" required>
        <el-input
          v-model="directoryCode"
          :placeholder="t('serviceTree.directoryCodePlaceholder')"
          maxlength="50"
          show-word-limit
          clearable
          data-testid="create-directory-code"
        />
        <div class="form-tip">
          <el-icon><InfoFilled /></el-icon>
          {{ t('serviceTree.directoryCodeHelp') }}
        </div>
      </el-form-item>
      <el-form-item :label="t('serviceTree.directoryDescription')">
        <el-input
          v-model="directoryDescription"
          type="textarea"
          :rows="3"
          :placeholder="t('serviceTree.directoryDescriptionPlaceholder')"
          maxlength="200"
          show-word-limit
        />
      </el-form-item>
      <el-form-item :label="t('serviceTree.directoryTags')">
        <el-input
          v-model="directoryTags"
          :placeholder="t('serviceTree.directoryTagsPlaceholder')"
          maxlength="100"
          clearable
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <span class="dialog-footer">
        <el-button data-testid="create-directory-cancel" @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" data-testid="create-directory-submit" @click="$emit('submit')" :loading="creating">
          {{ t('common.create') }}
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { InfoFilled } from '@element-plus/icons-vue'
import type { CreateServiceTreeRequest, ServiceTree as ServiceTreeType } from '@/architecture/domain/types'
import { buildUniqueGoPackageCode, createGoPackageCodeFromLabel } from '@/architecture/domain/utils/goPackageCode'

const props = defineProps<{
  visible: boolean
  parentNode: ServiceTreeType | null
  form: CreateServiceTreeRequest
  creating: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'update:form', value: CreateServiceTreeRequest): void
  (e: 'submit'): void
  (e: 'close'): void
}>()

const { t } = useI18n()

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value)
})

const dialogTitle = computed(() => {
  const parentName = props.parentNode?.name || props.parentNode?.code
  return parentName
    ? t('serviceTree.createDirectoryUnderTitle', { name: parentName })
    : t('serviceTree.createDirectoryTitle')
})

const codeManuallyEdited = ref(false)

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      codeManuallyEdited.value = Boolean(props.form.code)
    }
  }
)

function updateForm(patch: Partial<CreateServiceTreeRequest>): void {
  emit('update:form', {
    ...props.form,
    ...patch,
  })
}

const directoryName = computed({
  get: () => props.form.name,
  set: (value: string) => {
    const patch: Partial<CreateServiceTreeRequest> = { name: value }
    if (!codeManuallyEdited.value) {
      const baseCode = createGoPackageCodeFromLabel(value)
      patch.code = buildUniqueGoPackageCode(baseCode, siblingCodes())
    }
    updateForm(patch)
  }
})

const directoryCode = computed({
  get: () => props.form.code,
  set: (value: string) => {
    codeManuallyEdited.value = true
    updateForm({ code: value.toLowerCase() })
  }
})

const directoryDescription = computed({
  get: () => props.form.description,
  set: (value: string) => updateForm({ description: value })
})

const directoryTags = computed({
  get: () => props.form.tags,
  set: (value: string) => updateForm({ tags: value })
})

function siblingCodes(): string[] {
  return (props.parentNode?.children || [])
    .map(child => child.code || '')
    .filter(Boolean)
}
</script>
