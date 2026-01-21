<template>
  <el-dialog
    v-model="visible"
    title="工作空间设置"
    width="600px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div class="workspace-settings-dialog">
      <el-form label-width="100px">
        <el-form-item label="工作空间">
          <div class="workspace-info">
            <div class="workspace-name">{{ currentApp?.name || currentApp?.code }}</div>
            <div class="workspace-path">{{ currentApp?.user }}/{{ currentApp?.code }}</div>
          </div>
        </el-form-item>
        
        <el-form-item label="管理员">
          <UsersWidget
            :value="adminsFieldValue"
            :field="adminsField"
            mode="edit"
            @update:modelValue="handleAdminsChange"
          />
          <div class="form-tip">
            管理员拥有该工作空间的所有权限（app:admin）
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
import UsersWidget from '@/architecture/presentation/widgets/UsersWidget.vue'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import { WidgetType } from '@/core/constants/widget'

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

const saving = ref(false)
const adminsArray = ref<string[]>([])

// 管理员字段配置（用于 UsersWidget）
const adminsField = computed<FieldConfig>(() => ({
  code: 'admins',
  name: '管理员',
  widget: {
    type: WidgetType.USERS,
    config: {}
  }
}))

// 管理员字段值（用于 UsersWidget）
const adminsFieldValue = computed<FieldValue>(() => {
  if (adminsArray.value.length === 0) {
    return {
      raw: null,
      display: '',
      meta: {}
    }
  }
  
  return {
    raw: adminsArray.value.join(','),
    display: adminsArray.value.join(', '),
    meta: {}
  }
})

// 处理管理员字段变化
function handleAdminsChange(value: FieldValue) {
  if (value.raw === null || value.raw === '') {
    adminsArray.value = []
  } else {
    const admins = String(value.raw).split(',').map(s => s.trim()).filter(s => s)
    adminsArray.value = admins
  }
}

// 初始化表单数据
function initForm() {
  if (!props.currentApp) {
    adminsArray.value = []
    return
  }

  // 直接使用 currentApp 中的 admins 字段（tree 接口已经返回了）
  if (props.currentApp?.admins) {
    adminsArray.value = props.currentApp.admins
      .split(',')
      .map(s => s.trim())
      .filter(s => s)
  } else {
    adminsArray.value = []
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
    
    const admins = adminsArray.value.length > 0 ? adminsArray.value.join(',') : ''
    
    await updateWorkspace(props.currentApp.user, props.currentApp.code, {
      admins
    })
    
    ElNotification.success({
      title: '成功',
      message: '工作空间设置已保存'
    })
    
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
