<template>
  <el-dialog
    v-model="dialogVisible"
    title="创建新工作空间"
    width="800px"
    :close-on-click-modal="false"
    @close="$emit('close')"
  >
    <el-form :model="form" label-width="120px" data-testid="create-app-dialog">
      <el-form-item label="名称" required>
        <el-input
          v-model="form.name"
          placeholder="请输入名称（如：清北大学、首都市政府、xxx图书馆、xxx医院、xxx银行、xxx科技公司）"
          maxlength="100"
          show-word-limit
          clearable
          data-testid="create-app-name"
        />
      </el-form-item>
      <el-form-item label="英文标识" required>
        <el-tooltip
          content="2-50 个字符，以小写英文字母开头，可包含小写英文字母、数字和下划线，不要使用横线"
          placement="top"
        >
          <el-input
            v-model="form.code"
            placeholder="请输入英文标识（如：tsinghua、pku_gsm）"
            maxlength="50"
            show-word-limit
            clearable
            data-testid="create-app-code"
            @input="form.code = form.code.toLowerCase()"
          />
        </el-tooltip>
      </el-form-item>
      <el-form-item label="公开">
        <el-tooltip
          content="公开的工作空间可以被其他用户搜索到，关闭则仅自己可见"
          placement="top"
        >
          <el-switch v-model="form.is_public" />
        </el-tooltip>
      </el-form-item>
      <el-form-item label="隐藏无权限节点">
        <el-tooltip
          content="开启后，服务目录树不会返回当前用户无 read 权限的节点"
          placement="top"
        >
          <el-switch v-model="form.hide_unauthorized_nodes" />
        </el-tooltip>
      </el-form-item>
    </el-form>

    <template #footer>
      <span class="dialog-footer">
        <el-button data-testid="create-app-cancel" @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" data-testid="create-app-submit" @click="$emit('submit')" :loading="creating">
          创建
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CreateAppRequest } from '@/architecture/domain/types'

const props = defineProps<{
  visible: boolean
  form: CreateAppRequest
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
</script>
