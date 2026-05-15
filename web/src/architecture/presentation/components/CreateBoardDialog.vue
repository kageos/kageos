<template>
  <el-dialog
    :model-value="modelValue"
    :title="parentNode ? `在「${parentNode.name || parentNode.code}」下新增讨论区` : '新增讨论区'"
    width="560px"
    :close-on-click-modal="false"
    @update:model-value="emit('update:modelValue', $event)"
    @close="handleClose"
  >
    <el-form :model="form" label-width="90px" data-testid="create-board-dialog">
      <el-form-item label="讨论区名称" required>
        <el-input
          v-model="form.name"
          placeholder="如：变更记录、问答区"
          maxlength="100"
          show-word-limit
          clearable
          data-testid="create-board-name"
        />
      </el-form-item>
      <el-form-item label="讨论区英文标识" required>
        <el-input
          v-model="form.code"
          placeholder="英文，如 issue"
          maxlength="50"
          show-word-limit
          clearable
          data-testid="create-board-code"
          @input="form.code = form.code.toLowerCase().replace(/[^a-z0-9_]/g, '')"
        >
          <template #suffix>
            <span class="code-suffix">.board</span>
          </template>
        </el-input>
        <div class="form-tip">
          <el-icon><InfoFilled /></el-icon>
          可使用小写英文字母、数字和下划线，保存后自动带后缀 .board
        </div>
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
        <el-button data-testid="create-board-cancel" @click="emit('update:modelValue', false)">取消</el-button>
        <el-button type="primary" :loading="submitting" data-testid="create-board-submit" @click="handleSubmit">
          创建
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { InfoFilled } from '@element-plus/icons-vue'
import { createBoard } from '@/architecture/presentation/context/api/service-tree'
import type { ServiceTree } from '@/architecture/domain/types'

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
    ElMessage.warning('请输入讨论区名称')
    return
  }
  if (!form.value.code.trim()) {
    ElMessage.warning('请输入讨论区英文标识')
    return
  }
  if (!CODE_PATTERN.test(form.value.code)) {
    ElMessage.warning('讨论区英文标识只能包含小写英文字母、数字和下划线')
    return
  }

  // 自动补全 type 后缀（与 form/table/chart 一致）
  let code = form.value.code.trim()
  if (!code.endsWith('.board')) code = code + '.board'

  submitting.value = true
  try {
    const response = await createBoard({
      user: props.currentApp.user,
      app: props.currentApp.code,
      name: form.value.name.trim(),
      code,
      parent_full_code_path: props.parentNode?.full_code_path ?? '',
      description: form.value.description.trim() || '',
      tags: form.value.tags.trim() || ''
    })
    if (response?.id) {
      ElMessage.success('讨论区创建成功')
      emit('update:modelValue', false)
      emit('success', response)
    } else {
      emit('update:modelValue', false)
    }
  } catch (err: any) {
    ElMessage.error('创建讨论区失败: ' + (err?.message || '未知错误'))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.form-tip {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.code-suffix {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  padding-right: 4px;
}
</style>
