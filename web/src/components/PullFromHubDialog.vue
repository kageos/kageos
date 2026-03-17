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
        <template v-if="initialTargetName">
          <span class="target-directory-name">{{ initialTargetName }}</span>
        </template>
        <template v-else>
          <el-input
            v-model="form.target_directory_path"
            placeholder="留空则安装到应用根目录"
          />
          <el-text type="info" size="small" style="display: block; margin-top: 5px">
            指定目标目录路径，留空则安装到应用根目录
          </el-text>
        </template>
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
      <el-button type="primary" @click="handleSubmit">
        安装
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'
import { Link } from '@element-plus/icons-vue'
import { pullDirectoryFromHub, type PullDirectoryFromHubReq } from '@/api/hub'
import type { App } from '@/types'

interface Props {
  modelValue: boolean
  currentApp?: App | null  // 当前应用
  initialHubLink?: string  // 初始 Hub 链接（用于外部传入）
  initialTargetPath?: string  // 初始目标目录路径（提交用）
  initialTargetName?: string  // 初始目标目录名称（展示用，如「长宁」）
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
    if (props.initialHubLink) {
      form.value.hub_link = props.initialHubLink
    }
    if (props.initialTargetPath) {
      form.value.target_directory_path = props.initialTargetPath
    }
  }
})

// 监听 initialHubLink 变化，更新表单
watch(() => props.initialHubLink, (newLink) => {
  if (newLink && dialogVisible.value) {
    form.value.hub_link = newLink
  }
})

// 监听 initialTargetPath / initialTargetName 变化（目标目录默认当前选中目录）
watch(() => props.initialTargetPath, (newPath) => {
  if (newPath && dialogVisible.value) {
    form.value.target_directory_path = newPath
  }
})

// 初始化表单
const initForm = () => {
  if (!props.currentApp) {
    ElMessage.warning('请先选择应用')
    return
  }

  form.value = {
    hub_link: props.initialHubLink ?? '',
    target_directory_path: props.initialTargetPath ?? '',
  }
}

// 处理粘贴事件
const handlePaste = (event: ClipboardEvent) => {
  const pastedText = event.clipboardData?.getData('text')
  if (pastedText && pastedText.startsWith('hub://')) {
    form.value.hub_link = pastedText
    event.preventDefault() // 避免默认粘贴再插一次，导致重复
  }
}

// 提交表单：后台安装，弹窗立即关闭，右上角通知进度，成功自动消失
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid: boolean) => {
    if (!valid) return

    if (!props.currentApp) {
      ElMessage.error('缺少应用信息')
      return
    }

    const requestData: PullDirectoryFromHubReq = {
      hub_link: form.value.hub_link!,
      target_user: props.currentApp.user,
      target_app: props.currentApp.code,
      ...(form.value.target_directory_path ? { target_directory_path: form.value.target_directory_path } : {})
    }

    // 先关弹窗，不阻塞用户
    handleClose()

    // 右上角常驻「安装中」通知
    const loadingNotify = ElNotification({
      title: '安装中',
      message: '正在从应用中心安装目录，请稍候…',
      type: 'info',
      position: 'top-right',
      duration: 0
    })

    pullDirectoryFromHub(requestData)
      .then((response) => {
        loadingNotify.close()
        ElNotification.success({
          title: '安装成功',
          message: response.message || '目录已安装',
          position: 'top-right',
          duration: 3000
        })
        emit('success')
      })
      .catch((error: any) => {
        loadingNotify.close()
        const msg = error?.response?.data?.msg || error?.message || '未知错误'
        ElNotification.error({
          title: '安装失败',
          message: msg,
          position: 'top-right',
          duration: 5000
        })
        console.error('安装失败:', error)
      })
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
.target-directory-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}
</style>

