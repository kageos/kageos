<template>
  <el-dialog
    v-model="dialogVisible"
    title="编辑目录"
    width="600px"
    :close-on-click-modal="false"
  >
    <el-form
      ref="editFormRef"
      :model="form"
      label-width="100px"
      label-position="left"
    >
      <el-form-item label="目录名称" prop="name" :rules="[{ required: true, message: '请输入目录名称', trigger: 'blur' }]">
        <el-input
          v-model="form.name"
          placeholder="请输入目录名称"
          maxlength="100"
          show-word-limit
        />
      </el-form-item>

      <el-form-item label="管理员" prop="admins">
        <UsersWidget
          v-if="visible"
          :key="`admins-${form.admins || 'empty'}`"
          :field="adminsField"
          :value="adminsFieldValue"
          :field-path="adminsField.code"
          mode="edit"
          @update:model-value="$emit('update-admins', $event)"
        />
        <div class="form-item-tip">
          可以添加多个管理员，管理员可以编辑目录信息
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="dialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        保存
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import UsersWidget from '@/shared/components/UsersWidget.vue'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'

const props = defineProps<{
  visible: boolean
  form: {
    name: string
    admins: string
  }
  submitting: boolean
  adminsField: FieldConfig
  adminsFieldValue: FieldValue
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'update-admins', value: FieldValue): void
  (e: 'submit', formRef: any): void
}>()

const editFormRef = ref()

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value)
})

function handleSubmit() {
  emit('submit', editFormRef.value)
}
</script>

<style scoped lang="scss">
.form-item-tip {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
