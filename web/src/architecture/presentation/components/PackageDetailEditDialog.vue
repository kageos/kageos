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

const props = defineProps<{
  visible: boolean
  form: {
    name: string
  }
  submitting: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
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
