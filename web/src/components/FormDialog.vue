<template>
  <el-dialog
    v-model="dialogVisible"
    :title="title"
    :width="width"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <!-- 🔥 使用新的 FormRenderer 替代所有渲染逻辑 -->
    <FormRenderer
      v-if="dialogVisible"
      ref="formRendererRef"
      :function-detail="formFunctionDetail"
      :show-submit-button="false"
      :show-share-button="false"
      :show-reset-button="false"
      :show-debug-button="false"
    />

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          确定
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import FormRenderer from '@/core/renderers/FormRenderer.vue'
import type { FieldConfig, FunctionDetail } from '@/core/types/field'

interface Props {
  modelValue: boolean  // 对话框显示状态
  title: string  // 对话框标题
  fields: FieldConfig[]  // 表单字段
  mode: 'create' | 'update'  // 模式：新增或编辑
  initialData?: Record<string, any>  // 初始数据（编辑模式）
  width?: string | number  // 对话框宽度
}

const props = withDefaults(defineProps<Props>(), {
  width: '600px',
  initialData: () => ({})
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

// FormRenderer 引用
const formRendererRef = ref<InstanceType<typeof FormRenderer>>()

// 提交状态
const submitting = ref(false)

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
 */
const formFunctionDetail = computed<FunctionDetail>(() => ({
  id: 0,
  app_id: 0,
  tree_id: 0,
  method: 'POST',
  router: '',
  has_config: false,
  create_tables: '',
  callbacks: '',
  template_type: 'form',
  request: filteredFields.value,  // 🔥 使用过滤后的字段
  response: [],
  created_at: '',
  updated_at: '',
  full_code_path: ''
}))

/**
 * 提交表单
 */
const handleSubmit = async () => {
  if (!formRendererRef.value) {
    console.error('[FormDialog] FormRenderer 引用不存在')
    return
  }
  
  try {
    submitting.value = true
    
    // 🔥 调用 FormRenderer 的内部方法准备提交数据
    const submitData = formRendererRef.value.prepareSubmitDataWithTypeConversion()
    
    console.log('[FormDialog] 提交数据:', submitData)
    
    // 触发提交事件
    emit('submit', submitData)
    
  } catch (error) {
    console.error('[FormDialog] 提交失败:', error)
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

/**
 * 监听对话框显示状态
 */
watch(() => props.modelValue, (visible) => {
  if (visible) {
    console.log('[FormDialog] 对话框打开', {
      mode: props.mode,
      fields: props.fields.length,
      initialData: props.initialData
    })
  }
})

/**
 * 暴露方法给父组件
 */
defineExpose({
  formRendererRef,
  submit: handleSubmit
})
</script>

<style scoped>
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
