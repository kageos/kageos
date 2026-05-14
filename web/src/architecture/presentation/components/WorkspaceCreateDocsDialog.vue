<template>
  <el-dialog
    v-model="dialogVisible"
    :title="parentNode ? `在「${parentNode.name || parentNode.code}」下创建文档` : '创建文档'"
    width="600px"
    :close-on-click-modal="false"
    @close="$emit('close')"
  >
    <el-form :model="form" label-width="120px" data-testid="create-docs-dialog">
      <el-form-item label="文档名称" required>
        <el-input
          v-model="form.name"
          placeholder="请输入文档名称"
          maxlength="100"
          show-word-limit
          clearable
          data-testid="create-docs-name"
        />
      </el-form-item>
      <el-form-item label="文档英文标识" required>
        <el-input
          v-model="form.code"
          placeholder="英文，如 readme"
          maxlength="50"
          show-word-limit
          clearable
          data-testid="create-docs-code"
          @input="normalizeCode"
        >
          <template #suffix>
            <span class="create-docs-code-suffix">.docs</span>
          </template>
        </el-input>
        <div class="form-tip">
          <el-icon><InfoFilled /></el-icon>
          可使用小写英文字母、数字和下划线，保存后自动带后缀 .docs
        </div>
      </el-form-item>
      <el-form-item label="描述">
        <el-input
          v-model="form.description"
          type="textarea"
          placeholder="请输入文档描述（可选）"
          :rows="3"
          maxlength="500"
          show-word-limit
        />
      </el-form-item>
      <el-form-item label="标签">
        <el-input
          v-model="form.tags"
          placeholder="请输入标签，多个标签用逗号分隔（可选）"
          maxlength="200"
          clearable
        />
      </el-form-item>
      <el-form-item label="文档内容" required>
        <el-input
          v-model="form.content"
          type="textarea"
          placeholder="请输入文档内容（支持 Markdown 格式）"
          :rows="15"
          maxlength="50000"
          show-word-limit
          data-testid="create-docs-content"
        />
        <div class="form-tip">
          <el-icon><InfoFilled /></el-icon>
          支持 Markdown 格式，可以使用标题、列表、代码块、链接等语法
        </div>
      </el-form-item>
      <el-form-item label="文档摘要">
        <el-input
          v-model="form.summary"
          type="textarea"
          placeholder="请输入文档摘要（可选）"
          :rows="2"
          maxlength="500"
          show-word-limit
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <span class="dialog-footer">
        <el-button data-testid="create-docs-cancel" @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" data-testid="create-docs-submit" @click="$emit('submit')">
          创建
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { InfoFilled } from '@element-plus/icons-vue'
import type { ServiceTree as ServiceTreeType } from '@/architecture/domain/types'

interface CreateDocsForm {
  name: string
  code: string
  description: string
  tags: string
  content: string
  summary: string
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

function normalizeCode() {
  props.form.code = props.form.code.toLowerCase().replace(/[^a-z0-9_]/g, '')
}
</script>

<style scoped lang="scss">
.create-docs-code-suffix {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  padding-right: 4px;
}

.form-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}
</style>
