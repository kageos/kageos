<template>
  <el-dialog
    v-model="dialogVisible"
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
      ref="formViewRef"
      :function-detail="formFunctionDetail"
      :show-submit-button="false"
      :show-reset-button="false"
      :initial-data="props.initialData"
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
          v-if="props.mode === 'create' && !!props.router"
          type="primary"
          plain
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
import { Logger } from '@/core/utils/logger'
import type { FieldConfig, FunctionDetail } from '@/core/types/field'
import ScheduledTaskDialog from '@/architecture/presentation/components/ScheduledTaskDialog.vue'

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
 * 根据 table_permission 过滤字段
 */
const filteredFields = computed(() => {
  return props.fields.filter(field => {
    const permission = field.table_permission
    
    // 新增模式
    if (props.mode === 'create') {
      // read: 不显示（后端自动生成）
      // update: 不显示（只能编辑时修改）
      // create: 显示（只能新增时填写）
      // 空: 显示（全部权限）
      return !permission || permission === '' || permission === 'create'
    }
    
    // 编辑模式
    if (props.mode === 'update') {
      // read: 不显示（只读）
      // update: 显示（只能编辑时修改）
      // create: 不显示（只能新增时填写）
      // 空: 显示（全部权限）
      return !permission || permission === '' || permission === 'update'
    }
    
    return true
  })
})

/**
 * 🔥 将 fields 包装成 FunctionDetail 格式，供 FormRenderer 使用
 * 
 * ⚠️ 关键说明：
 * - 对于 table 函数的新增表单：fields 来自 functionDetail.response（新增时需要填写的字段）
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
    request: filteredFields.value,  // 🔥 使用过滤后的字段（对于 table 函数，这是 response 字段）
    response: [],  // 🔥 新增表单不需要显示响应参数
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
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.error-message {
  padding: 20px;
}
</style>
