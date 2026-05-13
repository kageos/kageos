<template>
  <el-dialog
    v-model="visible"
    class="workspace-settings-dialog-shell"
    title="工作空间设置"
    width="600px"
    :append-to-body="true"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div class="workspace-settings-dialog" data-testid="workspace-settings-dialog">
      <el-form label-width="120px">
        <el-form-item label="资源路径">
          <div class="workspace-info">
            <div class="workspace-name">{{ currentApp?.name || currentApp?.code }}</div>
            <div class="workspace-path">{{ workspaceResourcePath }}</div>
          </div>
        </el-form-item>
        
        <el-form-item v-if="featureFlags.permissions" label="仅展示有权限">
          <el-tooltip
            content="启用权限管控后生效；开启后非管理员左侧目录只展示其有权限的节点"
            placement="top"
          >
            <el-switch v-model="showOnlyPermitted" />
          </el-tooltip>
        </el-form-item>

        <el-form-item v-if="featureFlags.permissions" label="启用权限管控">
          <el-tooltip
            content="开启后按角色授权控制目录和函数访问；社区版默认关闭，企业版权限特性开启时默认启用"
            placement="top"
          >
            <el-switch v-model="permissionEnforced" />
          </el-tooltip>
        </el-form-item>

        <el-form-item v-if="featureFlags.permissions" label="管理员">
          <UsersWidget
            :value="adminsFieldValue"
            :field="adminsField"
            mode="edit"
            field-path="admins"
            @update:modelValue="handleAdminsChange"
          />
          <div class="form-tip">
            管理员拥有该资源根路径下的所有权限（app:admin）
          </div>
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" :loading="saving" data-testid="workspace-settings-save" @click="handleSave">
          保存
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'
import type { App } from '@/types'
import { updateWorkspace } from '@/api/app'
import { buildAppResourcePath } from '@/utils/resourcePath'
import UsersWidget from '@/shared/components/UsersWidget.vue'
import type { FieldValue } from '@/core/types/field'
import { WidgetType } from '@/core/constants/widget'
import { createStringFieldValue, createWidgetFieldConfig, extractStringFieldRaw } from '@/utils/widgetFieldHelpers'
import { featureFlags } from '@/config/features'

interface Props {
  modelValue: boolean
  currentApp: App | null
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'saved'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const workspaceResourcePath = computed(() => {
  if (!props.currentApp) return ''
  return buildAppResourcePath(props.currentApp.user, props.currentApp.code)
})

const saving = ref(false)
const adminsRaw = ref('')
const showOnlyPermitted = ref(false)
const permissionEnforced = ref(false)

const adminsField = createWidgetFieldConfig({
  code: 'admins',
  name: '管理员',
  widgetType: WidgetType.USERS
})

const adminsFieldValue = computed(() =>
  createStringFieldValue(adminsField, adminsRaw.value, {
    display: adminsRaw.value.split(',').map(s => s.trim()).filter(Boolean).join(', ')
  })
)

function handleAdminsChange(value: FieldValue) {
  adminsRaw.value = extractStringFieldRaw(value)
}

// 初始化表单数据
function initForm() {
  if (!props.currentApp) {
    adminsRaw.value = ''
    showOnlyPermitted.value = false
    permissionEnforced.value = false
    return
  }

  showOnlyPermitted.value = !!props.currentApp?.show_only_permitted
  permissionEnforced.value = !!props.currentApp?.permission_enforced

  // 直接使用 currentApp 中的 admins 字段（tree 接口已经返回了）
  if (props.currentApp?.admins) {
    adminsRaw.value = props.currentApp.admins
  } else {
    adminsRaw.value = ''
  }
}

// 监听对话框显示状态，初始化表单
watch(visible, (newVal) => {
  if (newVal) {
    initForm()
  }
})

// 保存设置
async function handleSave() {
  if (!props.currentApp) {
    ElMessage.error('请先选择工作空间')
    return
  }

  try {
    saving.value = true

    await updateWorkspace(buildAppResourcePath(props.currentApp.user, props.currentApp.code), {
      admins: adminsRaw.value.trim(),
      show_only_permitted: showOnlyPermitted.value,
      permission_enforced: permissionEnforced.value
    })
    
    ElMessage.success('工作空间设置已保存')
    
    emit('saved')
    handleClose()
  } catch (error: any) {
    const errorMessage = error?.response?.data?.msg || '保存工作空间设置失败'
    ElNotification.error({
      title: '错误',
      message: errorMessage
    })
  } finally {
    saving.value = false
  }
}

// 关闭对话框
function handleClose() {
  visible.value = false
}
</script>

<style scoped>
.workspace-settings-dialog {
  padding: 8px 0;
}

:deep(.workspace-settings-dialog-shell) {
  border-radius: 28px;
  background: var(--app-auth-card-bg);
  border: 1px solid var(--app-auth-card-border);
  box-shadow: var(--app-auth-card-shadow);
  overflow: hidden;
}

:deep(.workspace-settings-dialog-shell .el-dialog__header) {
  padding: 28px 32px 12px;
}

:deep(.workspace-settings-dialog-shell .el-dialog__title) {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
}

:deep(.workspace-settings-dialog-shell .el-dialog__body) {
  padding: 0 32px 24px;
  background: var(--app-auth-surface-bg);
}

:deep(.workspace-settings-dialog-shell .el-dialog__footer) {
  padding: 0 32px 28px;
  background: var(--app-auth-surface-bg);
}

:deep(.workspace-settings-dialog-shell .el-form-item__label) {
  font-weight: 600;
  color: var(--text-primary);
}

:deep(.workspace-settings-dialog-shell .el-input__wrapper),
:deep(.workspace-settings-dialog-shell .el-select__wrapper),
:deep(.workspace-settings-dialog-shell .el-textarea__inner) {
  background: var(--app-auth-input-bg);
  border-color: var(--app-auth-input-border);
  border-radius: 12px;
  box-shadow: none;
  transition: all 0.3s ease;
}

:deep(.workspace-settings-dialog-shell .el-input__wrapper:hover),
:deep(.workspace-settings-dialog-shell .el-select__wrapper:hover),
:deep(.workspace-settings-dialog-shell .el-textarea__inner:hover) {
  border-color: rgba(var(--el-color-primary-rgb), 0.42);
  box-shadow: var(--app-auth-input-shadow-hover);
}

:deep(.workspace-settings-dialog-shell .el-input__wrapper.is-focus),
:deep(.workspace-settings-dialog-shell .el-select__wrapper.is-focused),
:deep(.workspace-settings-dialog-shell .el-textarea__inner:focus) {
  border-color: var(--el-color-primary);
  box-shadow: var(--app-auth-input-shadow-focus);
}

.workspace-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 14px 16px;
  background: var(--app-auth-card-bg-strong);
  border: 1px solid var(--app-auth-card-border);
  border-radius: 18px;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.4);
}

.workspace-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.workspace-path {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.form-tip {
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.dialog-footer :deep(.el-button) {
  height: 44px;
  border-radius: 14px;
  font-weight: 600;
  border: 1px solid var(--app-auth-input-border);
  background: var(--app-auth-input-bg);
  box-shadow: none;
  transition: all 0.3s ease;
}

.dialog-footer :deep(.el-button:hover) {
  transform: translateY(-1px);
  border-color: rgba(var(--el-color-primary-rgb), 0.42);
  color: var(--el-color-primary);
  box-shadow: var(--app-auth-input-shadow-hover);
}

.dialog-footer :deep(.el-button--primary) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: #fff;
  box-shadow: var(--app-auth-primary-shadow);
}

.dialog-footer :deep(.el-button--primary:hover) {
  color: #fff;
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  box-shadow: var(--app-auth-primary-shadow-hover);
}
</style>
