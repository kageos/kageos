<template>
  <el-dialog
    v-model="visible"
    title="工作空间设置"
    width="600px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div class="workspace-settings-dialog">
      <el-form label-width="120px">
        <el-form-item label="资源路径">
          <div class="workspace-info">
            <div class="workspace-name">{{ currentApp?.name || currentApp?.code }}</div>
            <div class="workspace-path">{{ workspaceResourcePath }}</div>
          </div>
        </el-form-item>
        
        <el-form-item label="仅展示有权限">
          <el-tooltip
            content="开启后，非管理员进入该工作空间时，左侧目录只展示其有权限的节点（适合 SaaS 多租户场景）"
            placement="top"
          >
            <el-switch v-model="showOnlyPermitted" />
          </el-tooltip>
        </el-form-item>

        <el-form-item label="管理员">
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
        <el-button type="primary" :loading="saving" @click="handleSave">
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
    return
  }

  showOnlyPermitted.value = !!props.currentApp?.show_only_permitted

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
      show_only_permitted: showOnlyPermitted.value
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
  padding: 20px 0;
}

.workspace-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
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
</style>
