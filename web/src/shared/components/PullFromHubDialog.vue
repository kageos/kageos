<template>
  <el-dialog
    v-model="dialogVisible"
    title="从应用中心安装目录"
    width="640px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="120px"
      v-loading="loading"
      data-testid="pull-from-hub-dialog"
    >
      <el-form-item label="安装方式">
        <el-radio-group v-model="installMode" class="install-mode-group" data-testid="pull-from-hub-mode">
          <el-radio-button value="hub_link">Hub 链接</el-radio-button>
          <el-radio-button value="json_bundle">离线 JSON</el-radio-button>
        </el-radio-group>
      </el-form-item>

      <template v-if="installMode === 'hub_link'">
      <el-form-item label="Hub 链接" prop="hub_link">
        <el-input
          v-model="form.hub_link"
          placeholder="粘贴 Hub 链接，格式：hub://host/full_code_path@version"
          data-testid="pull-from-hub-link"
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
      </template>

      <template v-else>
      <el-form-item label="安装包 JSON">
        <el-input
          v-model="bundleJsonText"
          type="textarea"
          :rows="10"
          placeholder="粘贴从应用中心详情页「导出 JSON 安装包」下载的文件内容，或点击下方选择 .json 文件"
          spellcheck="false"
          data-testid="pull-from-hub-json"
        />
        <el-upload
          class="bundle-upload"
          :auto-upload="false"
          :show-file-list="true"
          :limit="1"
          accept=".json,application/json"
          @change="handleBundleFileChange"
        >
          <el-button type="default" data-testid="pull-from-hub-json-upload">选择 JSON 文件</el-button>
        </el-upload>
        <el-text type="info" size="small" style="display: block; margin-top: 5px">
          与 Hub 链接安装效果相同，适用于内网或未连通 Hub 的场景；导出文件内含目录树与源码
        </el-text>
      </el-form-item>
      </template>

      <el-form-item label="目标目录" prop="target_directory_path">
        <template v-if="initialTargetName">
          <span class="target-directory-name">{{ initialTargetName }}</span>
        </template>
        <template v-else>
          <el-input
            v-model="form.target_directory_path"
            placeholder="留空则安装到应用根目录"
            data-testid="pull-from-hub-target"
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
      <el-button data-testid="pull-from-hub-cancel" @click="handleClose">取消</el-button>
      <el-button type="primary" data-testid="pull-from-hub-submit" @click="handleSubmit">
        安装
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'
import { Link } from '@element-plus/icons-vue'
import {
  pullDirectoryFromHub,
  importHubDirectoryBundle,
  type PullDirectoryFromHubReq
} from '@/api/hub'
import { parseHubDirectoryBundleJson } from '@/utils/hubBundle'
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
const installMode = ref<'hub_link' | 'json_bundle'>('hub_link')
const bundleJsonText = ref('')

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

// 初始化表单
function initForm() {
  if (!props.currentApp) {
    ElMessage.warning('请先选择应用')
    return
  }

  form.value = {
    hub_link: props.initialHubLink ?? '',
    target_directory_path: props.initialTargetPath ?? '',
  }
}

// 监听对话框打开，初始化表单
watch(dialogVisible, (visible) => {
  if (visible) {
    installMode.value = 'hub_link'
    bundleJsonText.value = ''
    initForm()
    if (props.initialHubLink) {
      form.value.hub_link = props.initialHubLink
    }
    if (props.initialTargetPath) {
      form.value.target_directory_path = props.initialTargetPath
    }
  }
}, { immediate: true })

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

// 处理粘贴事件
const handlePaste = (event: ClipboardEvent) => {
  const pastedText = event.clipboardData?.getData('text')
  if (pastedText && pastedText.startsWith('hub://')) {
    form.value.hub_link = pastedText
    event.preventDefault() // 避免默认粘贴再插一次，导致重复
  }
}

function handleBundleFileChange(uploadFile: { raw?: File }) {
  const raw = uploadFile.raw
  if (!raw) return
  const reader = new FileReader()
  reader.onload = () => {
    const t = typeof reader.result === 'string' ? reader.result : ''
    bundleJsonText.value = t
  }
  reader.readAsText(raw, 'UTF-8')
}

function runInstallNotify(
  promise: Promise<{ message?: string }>,
  loadingMessage: string
) {
  handleClose()
  const loadingNotify = ElNotification({
    title: '安装中',
    message: loadingMessage,
    type: 'info',
    position: 'top-right',
    duration: 0
  })
  promise
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
}

// 提交表单：后台安装，弹窗立即关闭，右上角通知进度，成功自动消失
const handleSubmit = async () => {
  if (!props.currentApp) {
    ElMessage.error('缺少应用信息')
    return
  }

  if (installMode.value === 'hub_link') {
    if (!formRef.value) return
    await formRef.value.validate(async (valid: boolean) => {
      if (!valid) return
      const requestData: PullDirectoryFromHubReq = {
        hub_link: form.value.hub_link!,
        target_user: props.currentApp!.user,
        target_app: props.currentApp!.code,
        ...(form.value.target_directory_path ? { target_directory_path: form.value.target_directory_path } : {})
      }
      runInstallNotify(
        pullDirectoryFromHub(requestData),
        '正在从应用中心安装目录，请稍候…'
      )
    })
    return
  }

  const raw = bundleJsonText.value.trim()
  if (!raw) {
    ElMessage.warning('请粘贴或上传 JSON 安装包')
    return
  }
  let bundle: ReturnType<typeof parseHubDirectoryBundleJson>
  try {
    bundle = parseHubDirectoryBundleJson(raw)
  } catch (e: any) {
    ElMessage.error(e?.message || 'JSON 解析失败，请确认是应用中心导出的标准安装包')
    return
  }

  runInstallNotify(
    importHubDirectoryBundle({
      target_user: props.currentApp.user,
      target_app: props.currentApp.code,
      ...(form.value.target_directory_path ? { target_directory_path: form.value.target_directory_path } : {}),
      bundle
    }),
    '正在从离线安装包安装目录，请稍候…'
  )
}

// 关闭对话框
const handleClose = () => {
  installMode.value = 'hub_link'
  bundleJsonText.value = ''
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
.install-mode-group {
  width: 100%;
}
.bundle-upload {
  margin-top: 8px;
}
</style>
