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
          content="合法 Go package 名称：以小写字母开头，只能包含小写字母、数字和下划线，不能使用中划线或 Go 保留关键字，长度 2-50 个字符"
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
      <el-form-item label="仅展示有权限">
        <el-tooltip
          content="启用权限管控后生效；开启后非管理员左侧目录只展示其有权限的节点"
          placement="top"
        >
          <el-switch v-model="form.show_only_permitted" />
        </el-tooltip>
      </el-form-item>
      <el-form-item label="启用权限管控">
        <el-tooltip
          content="开启后按角色授权控制目录和函数访问；默认关闭，避免新旧工作空间因未配置授权而被阻塞"
          placement="top"
        >
          <el-switch v-model="form.permission_enforced" />
        </el-tooltip>
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
          可以设置多个管理员，用逗号分隔。管理员拥有工作空间的管理权限
        </div>
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
import { InfoFilled } from '@element-plus/icons-vue'
import UsersWidget from '@/shared/components/UsersWidget.vue'
import type { CreateAppRequest } from '@/types'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'

const props = defineProps<{
  visible: boolean
  form: CreateAppRequest
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
