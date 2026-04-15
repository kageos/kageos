<template>
  <div class="form-page">
    <div class="form-page-header">
      <el-button :icon="ArrowLeft" @click="$emit('back')">返回列表</el-button>
      <h2 class="form-page-title">{{ title }}</h2>
    </div>

    <div class="form-page-content">
      <FormView
        v-if="isFormSupported"
        ref="formViewRef"
        :key="pageKey"
        :function-detail="props.functionDetail ?? undefined"
        :initial-data="initialData"
        :show-submit-button="false"
      />
      <div v-else class="empty-state">
        <p>{{ unsupportedMessage }}</p>
      </div>
    </div>

    <div class="form-page-footer">
      <el-button @click="$emit('back')" :disabled="submitting">取消</el-button>
      <el-button
        type="primary"
        :loading="submitting"
        :disabled="!isFormSupported"
        @click="handleSubmit"
      >
        {{ submitText }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ArrowLeft } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import FormView from '@/architecture/presentation/views/FormView.vue'
import type { FunctionDetail } from '@/architecture/domain/types'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'

const props = withDefaults(defineProps<{
  title: string
  functionDetail: FunctionDetail | null
  initialData?: Record<string, any>
  pageKey: string
  submitText?: string
  unsupportedMessage?: string
}>(), {
  initialData: undefined,
  submitText: '提交',
  unsupportedMessage: '该函数不支持此操作'
})

const formViewRef = ref<InstanceType<typeof FormView> | null>(null)
const submitting = ref(false)
const isFormSupported = computed(() => props.functionDetail?.template_type === TEMPLATE_TYPE.FORM)

defineEmits<{
  (e: 'back'): void
}>()

const handleSubmit = async (): Promise<void> => {
  if (!isFormSupported.value) {
    ElMessage.warning(props.unsupportedMessage)
    return
  }

  if (!formViewRef.value) {
    ElMessage.warning('表单尚未加载完成，请稍后重试')
    return
  }

  try {
    submitting.value = true
    await formViewRef.value.submitForm()
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped lang="scss">
.form-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
  overflow-y: auto;
}

.form-page-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.form-page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.form-page-content {
  flex: 1;
  min-height: 0;
}

.form-page-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}
</style>
