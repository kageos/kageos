<template>
  <el-dialog
    v-model="dialogVisible"
    class="form-dialog-shell"
    :title="title"
    :width="width"
    :close-on-click-modal="false"
    :append-to-body="true"
    @close="handleClose"
  >
    <!-- 🔥 使用新的 FormView 替代所有渲染逻辑 -->
    <template v-if="dialogVisible">
    <FormView
        v-if="formFunctionDetail"
      class="form-dialog-view"
      ref="formViewRef"
      :function-detail="formFunctionDetail"
      :show-submit-button="false"
      :show-reset-button="false"
      :initial-data="props.initialData"
      flat-surface
    />
      <div v-else class="error-message">
        <el-alert
          type="error"
          :title="`无法构建表单：method 参数不存在。router: ${props.router}`"
          :closable="false"
          show-icon
        />
      </div>
    </template>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button
          v-if="featureFlags.scheduledTasks && props.mode === 'create' && !!props.router"
          @click="openScheduledTaskDialog"
          :disabled="submitting"
        >
          定时执行
        </el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          确定
        </el-button>
      </span>
    </template>
    <ScheduledTaskDialog
      v-if="featureFlags.scheduledTasks"
      v-model="showScheduledTaskDialog"
      :full-code-path="props.router"
      table-mode
      fixed-action="table_create"
      :get-payload="buildScheduledPayload"
    />
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import FormView from '@/architecture/presentation/views/FormView.vue'
import { Logger } from '@/architecture/shared/logger'
import type { FieldConfig, FunctionDetail } from '@/architecture/domain/types'
import ScheduledTaskDialog from '@/architecture/presentation/components/ScheduledTaskDialog.vue'
import { featureFlags } from '@/architecture/runtime/config/features'

interface Props {
  modelValue: boolean  // 对话框显示状态
  title: string  // 对话框标题
  fields: FieldConfig[]  // 表单字段
  mode: 'create' | 'update'  // 模式：新增或编辑
  router: string  // ✨ 函数路由（用于文件上传等）
  method?: string  // 🔥 原函数的 HTTP 方法（用于 OnSelectFuzzy 回调）
  initialData?: Record<string, any>  // 初始数据（编辑模式）
  width?: string | number  // 对话框宽度
}

const props = withDefaults(defineProps<Props>(), {
  width: '1200px',
  initialData: () => ({}),
  router: ''
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [data: Record<string, any>]
  close: []
}>()

// 对话框显示状态
const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// FormView 引用
const formViewRef = ref<InstanceType<typeof FormView>>()

// 提交状态
const submitting = ref(false)
const showScheduledTaskDialog = ref(false)

/**
 * 🔥 将 fields 包装成 FunctionDetail 格式，供 FormRenderer 使用
 * 
 * ⚠️ 关键说明：
 * - 对于 table 函数的新增表单：fields 来自 table schema 的 create 场景字段
 * - request 字段用于 FormRenderer 渲染可编辑的表单字段
 * - response 字段为空数组（新增表单不需要显示响应参数）
 * - id 设置为 0（FormRenderer 需要正确处理 id === 0 的情况）
 */
const formFunctionDetail = computed<FunctionDetail | null>(() => {
  // 🔥 method 是必需的，如果不存在应该返回 null，让模板不渲染 FormRenderer
  if (!props.method) {
    Logger.error('FormDialog', `method 参数不存在，无法构建 formFunctionDetail。router: ${props.router}`)
    return null
  }
  
  return {
    id: 0,  // ⚠️ 注意：id 为 0，FormRenderer 需要正确处理这种情况
    app_id: 0,
    tree_id: 0,
    code: props.router || '',
    name: props.title || '表单',
    // 🔥 使用原函数的 method，这样 OnSelectFuzzy 回调才能正确获取到原函数的 method
    method: props.method,
    router: props.router,  // ✨ 使用传入的 router
    has_config: false,
    create_tables: '',
    callbacks: [],
    template_type: 'form',
    schema: {
      version: 1,
      type: 'form',
      callbacks: [],
      form: {
        request: props.fields,
        response: []
      }
    },
    created_at: '',
    updated_at: '',
    full_code_path: ''
  }
})

/**
 * 提交表单
 */
const handleSubmit = async () => {
  if (!formViewRef.value) {
    Logger.error('FormDialog', 'FormView 引用不存在')
    return
  }
  
  try {
    submitting.value = true
    
    // 🔥 提交时验证表单
    const isValid = formViewRef.value.validateForm()
    if (!isValid) {
      Logger.warn('FormDialog', '表单验证失败')
      return
    }
    
    // 🔥 验证通过后，准备提交数据
    const submitData = await formViewRef.value.prepareSubmitDataWithTypeConversion()
    
    // 触发提交事件
    emit('submit', submitData)
    
  } catch (error) {
    Logger.error('FormDialog', '提交失败', error)
    throw error
  } finally {
    submitting.value = false
  }
}

/**
 * 关闭对话框
 */
const handleClose = () => {
  emit('close')
  emit('update:modelValue', false)
}

const openScheduledTaskDialog = () => {
  if (!featureFlags.scheduledTasks) {
    return
  }
  showScheduledTaskDialog.value = true
}

const buildScheduledPayload = async (): Promise<Record<string, unknown>> => {
  if (!formViewRef.value) {
    throw new Error('表单尚未就绪，无法创建定时任务')
  }
  const isValid = formViewRef.value.validateForm()
  if (!isValid) {
    throw new Error('请先修正表单校验错误')
  }
  return await formViewRef.value.prepareSubmitDataWithTypeConversion()
}

/**
 * 暴露方法给父组件
 */
defineExpose({
  formViewRef,
  submit: handleSubmit
})
</script>

<style scoped>
:deep(.form-dialog-shell) {
  border-radius: 22px;
  background: var(--app-auth-card-bg);
  border: none !important;
  box-shadow: var(--app-auth-card-shadow);
  overflow: hidden;
}

:deep(.form-dialog-shell .el-dialog__header) {
  padding: 32px 36px 14px;
}

:deep(.form-dialog-shell .el-dialog__title) {
  font-size: 30px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.5px;
}

:deep(.form-dialog-shell .el-dialog__body) {
  padding: 0 36px 24px;
  background: var(--app-auth-surface-bg);
}

:deep(.form-dialog-shell .el-dialog__footer) {
  padding: 0 36px 32px;
  background: var(--app-auth-surface-bg);
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.dialog-footer :deep(.el-button) {
  height: 44px;
  border-radius: 12px;
  padding: 0 18px;
  font-weight: 600;
  border: 1px solid var(--app-auth-input-border);
  background: var(--app-auth-input-bg);
  box-shadow: none;
  transition: all 0.3s ease;
}

.dialog-footer :deep(.el-button:hover) {
  transform: translateY(-1px);
  border-color: rgba(var(--el-color-primary-rgb), 0.42);
  color: var(--el-color-primary);
  box-shadow: var(--app-auth-input-shadow-hover);
}

.dialog-footer :deep(.el-button--primary) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: #fff;
  box-shadow: var(--app-auth-primary-shadow);
}

.dialog-footer :deep(.el-button--primary:hover) {
  color: #fff;
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  box-shadow: var(--app-auth-primary-shadow-hover);
}

:deep(.form-dialog-shell .form-actions-section),
:deep(.form-dialog-shell .response-section),
:deep(.form-dialog-shell .metadata-section) {
  border-top-color: rgba(148, 163, 184, 0.16);
}

.error-message {
  padding: 20px;
}
</style>
