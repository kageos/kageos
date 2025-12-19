<template>
  <el-dialog
    v-model="dialogVisible"
    title="从应用中心安装目录"
    width="600px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="120px"
      v-loading="loading"
    >
      <el-form-item label="Hub 链接" prop="hub_link">
        <el-input
          v-model="form.hub_link"
          placeholder="粘贴 Hub 链接，格式：hub://host/full_code_path@version"
          @paste="handlePaste"
        >
          <template #prepend>
            <el-icon><Link /></el-icon>
          </template>
        </el-input>
        <el-text type="info" size="small" style="display: block; margin-top: 5px">
          从应用中心复制 Hub 链接，粘贴到这里即可自动安装
        </el-text>
      </el-form-item>

      <el-form-item label="目标目录" prop="target_directory_path">
        <el-input
          v-model="form.target_directory_path"
          placeholder="留空则安装到应用根目录"
        />
        <el-text type="info" size="small" style="display: block; margin-top: 5px">
          指定目标目录路径，留空则安装到应用根目录
        </el-text>
      </el-form-item>

      <el-form-item label="提示">
        <el-alert
          type="info"
          :closable="false"
          show-icon
        >
          <template #default>
            <div>
              <p>从应用中心安装目录时，将自动创建目录结构和文件。</p>
              <p>安装完成后，可以在服务目录树中查看安装的目录。</p>
            </div>
          </template>
        </el-alert>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" @click="handleSubmit" :loading="submitting">
        安装
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Link } from '@element-plus/icons-vue'
import { pullDirectoryFromHub, type PullDirectoryFromHubReq } from '@/api/hub'
import type { App } from '@/types'

interface Props {
  modelValue: boolean
  currentApp?: App | null  // 当前应用
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const formRef = ref()
const loading = ref(false)
const submitting = ref(false)

// 表单数据
const form = ref<Partial<PullDirectoryFromHubReq>>({
  hub_link: '',
  target_directory_path: '',
})

// 表单验证规则
const rules = {
  hub_link: [
    { required: true, message: '请输入 Hub 链接', trigger: 'blur' },
    {
      validator: (rule: any, value: string, callback: Function) => {
        if (!value || value.trim() === '') {
          callback(new Error('请输入 Hub 链接'))
        } else if (!value.startsWith('hub://')) {
          callback(new Error('Hub 链接格式不正确，应以 hub:// 开头'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

// 监听对话框打开，初始化表单
watch(dialogVisible, (visible) => {
  if (visible) {
    initForm()
  }
})

// 初始化表单
const initForm = () => {
  if (!props.currentApp) {
    ElMessage.warning('请先选择应用')
    return
  }

  form.value = {
    hub_link: '',
    target_directory_path: '',
  }
}

// 处理粘贴事件
const handlePaste = (event: ClipboardEvent) => {
  const pastedText = event.clipboardData?.getData('text')
  if (pastedText && pastedText.startsWith('hub://')) {
    form.value.hub_link = pastedText
  }
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid: boolean) => {
    if (!valid) return

    if (!props.currentApp) {
      ElMessage.error('缺少应用信息')
      return
    }

    submitting.value = true
    try {
      const requestData: PullDirectoryFromHubReq = {
        hub_link: form.value.hub_link!,
        target_user: props.currentApp.user,
        target_app: props.currentApp.code,
        ...(form.value.target_directory_path ? { target_directory_path: form.value.target_directory_path } : {})
      }

      // 调用拉取接口
      const response = await pullDirectoryFromHub(requestData)

      ElMessage.success(response.message || '安装成功！')

      emit('success')
      handleClose()
    } catch (error: any) {
      ElMessage.error(`安装失败: ${error.message || '未知错误'}`)
      console.error('安装失败:', error)
    } finally {
      submitting.value = false
    }
  })
}

// 关闭对话框
const handleClose = () => {
  form.value = {
    hub_link: '',
    target_directory_path: '',
  }
  formRef.value?.resetFields()
  emit('update:modelValue', false)
}
</script>

<style scoped>
:deep(.el-form-item__label) {
  font-weight: 500;
}
</style>

