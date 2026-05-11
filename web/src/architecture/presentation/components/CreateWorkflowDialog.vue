<template>
  <el-dialog
    :model-value="modelValue"
    :title="parentNode ? `在「${parentNode.name || parentNode.code}」下新增工作流` : '新增工作流'"
    width="560px"
    :close-on-click-modal="false"
    @update:model-value="emit('update:modelValue', $event)"
    @close="handleClose"
  >
    <el-form :model="form" label-width="104px" data-testid="create-workflow-dialog">
      <el-form-item label="工作流名称" required>
        <el-input
          v-model="form.name"
          placeholder="如：线索入库审批"
          maxlength="100"
          show-word-limit
          clearable
          data-testid="create-workflow-name"
        />
      </el-form-item>
      <el-form-item label="英文标识" required>
        <el-input
          v-model="form.code"
          placeholder="英文，如 lead_approval"
          maxlength="50"
          show-word-limit
          clearable
          data-testid="create-workflow-code"
          @input="form.code = form.code.toLowerCase().replace(/[^a-z0-9_]/g, '')"
        >
          <template #suffix>
            <span class="code-suffix">.workflow</span>
          </template>
        </el-input>
      </el-form-item>
      <el-form-item label="描述">
        <el-input
          v-model="form.description"
          type="textarea"
          placeholder="可选"
          :rows="2"
          maxlength="500"
          show-word-limit
        />
      </el-form-item>
      <el-form-item label="标签">
        <el-input
          v-model="form.tags"
          placeholder="多个标签用逗号分隔（可选）"
          maxlength="200"
          clearable
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <span class="dialog-footer">
        <el-button data-testid="create-workflow-cancel" @click="emit('update:modelValue', false)">取消</el-button>
        <el-button type="primary" :loading="submitting" data-testid="create-workflow-submit" @click="handleSubmit">
          创建
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { createWorkflowNode } from '@/api/service-tree'
import type { ServiceTree } from '@/types'

const CODE_PATTERN = /^[a-z0-9_]+$/

interface Props {
  modelValue: boolean
  currentApp: { user: string; code: string } | null
  parentNode: ServiceTree | null
}

interface Emits {
  (e: 'update:modelValue', v: boolean): void
  (e: 'success', node: ServiceTree): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const form = ref({
  name: '',
  code: '',
  description: '',
  tags: ''
})
const submitting = ref(false)

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) {
      form.value = { name: '', code: '', description: '', tags: '' }
    }
  }
)

function handleClose() {
  form.value = { name: '', code: '', description: '', tags: '' }
  emit('update:modelValue', false)
}

async function handleSubmit() {
  if (!props.currentApp) {
    ElMessage.warning('请先选择应用')
    return
  }
  if (!form.value.name.trim()) {
    ElMessage.warning('请输入工作流名称')
    return
  }
  if (!form.value.code.trim()) {
    ElMessage.warning('请输入工作流英文标识')
    return
  }
  if (!CODE_PATTERN.test(form.value.code)) {
    ElMessage.warning('工作流英文标识只能包含小写英文字母、数字和下划线')
    return
  }

  let code = form.value.code.trim()
  if (!code.endsWith('.workflow')) code = `${code}.workflow`

  submitting.value = true
  try {
    const response = await createWorkflowNode({
      user: props.currentApp.user,
      app: props.currentApp.code,
      name: form.value.name.trim(),
      code,
      parent_full_code_path: props.parentNode?.full_code_path ?? '',
      description: form.value.description.trim() || '',
      tags: form.value.tags.trim() || ''
    })
    if (response?.id) {
      ElMessage.success('工作流创建成功')
      emit('update:modelValue', false)
      emit('success', response)
    } else {
      emit('update:modelValue', false)
    }
  } catch (err: any) {
    ElMessage.error('创建工作流失败: ' + (err?.message || '未知错误'))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.code-suffix {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  padding-right: 4px;
}
</style>
