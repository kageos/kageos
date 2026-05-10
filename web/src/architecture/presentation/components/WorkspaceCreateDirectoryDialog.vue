<template>
  <el-dialog
    v-model="dialogVisible"
    :title="parentNode ? `在「${parentNode.name || parentNode.code}」下创建服务目录` : '创建服务目录'"
    width="520px"
    :close-on-click-modal="false"
    @close="$emit('close')"
  >
    <el-form :model="form" label-width="120px" data-testid="create-directory-dialog">
      <el-form-item label="目录名称" required>
        <el-input
          v-model="form.name"
          placeholder="请输入目录名称（如：用户管理）"
          maxlength="50"
          show-word-limit
          clearable
          data-testid="create-directory-name"
        />
      </el-form-item>
      <el-form-item label="目录英文标识" required>
        <el-input
          v-model="form.code"
          placeholder="请输入目录英文标识，如：user"
          maxlength="50"
          show-word-limit
          clearable
          data-testid="create-directory-code"
          @input="form.code = form.code.toLowerCase()"
        />
        <div class="form-tip">
          <el-icon><InfoFilled /></el-icon>
          以小写英文字母开头，可包含小写英文字母、数字和下划线，不要使用横线
        </div>
      </el-form-item>
      <el-form-item label="描述">
        <el-input
          v-model="form.description"
          type="textarea"
          :rows="3"
          placeholder="请输入目录描述（可选）"
          maxlength="200"
          show-word-limit
        />
      </el-form-item>
      <el-form-item label="标签">
        <el-input
          v-model="form.tags"
          placeholder="请输入标签，多个标签用逗号分隔（可选）"
          maxlength="100"
          clearable
        />
      </el-form-item>
      <el-form-item label="管理员">
        <UsersWidget
          :field="adminsField"
          :value="adminsFieldValue"
          :field-path="adminsField.code"
          mode="edit"
          @update:modelValue="value => $emit('update-admins', value)"
        />
        <div class="form-tip">
          <el-icon><InfoFilled /></el-icon>
          默认当前用户为管理员，可以添加其他用户
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <span class="dialog-footer">
        <el-button data-testid="create-directory-cancel" @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" data-testid="create-directory-submit" @click="$emit('submit')" :loading="creating">
          创建
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { InfoFilled } from '@element-plus/icons-vue'
import UsersWidget from '@/shared/components/UsersWidget.vue'
import type { CreateServiceTreeRequest, ServiceTree as ServiceTreeType } from '@/types'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'

const props = defineProps<{
  visible: boolean
  parentNode: ServiceTreeType | null
  form: CreateServiceTreeRequest
  creating: boolean
  adminsField: FieldConfig
  adminsFieldValue: FieldValue
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'update-admins', value: FieldValue): void
  (e: 'submit'): void
  (e: 'close'): void
}>()

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value)
})
</script>
