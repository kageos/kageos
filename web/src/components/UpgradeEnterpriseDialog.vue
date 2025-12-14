<template>
  <el-dialog
    v-model="dialogVisible"
    title="升级企业版"
    width="600px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form :model="formData" label-width="120px">
      <el-form-item label="License 文件">
        <el-upload
          :auto-upload="false"
          :on-change="handleFileChange"
          :on-remove="handleFileRemove"
          :file-list="fileList"
          accept=".json"
          :limit="1"
        >
          <el-button type="primary">选择 License 文件</el-button>
          <template #tip>
            <div class="el-upload__tip">
              请选择有效的 License JSON 文件
            </div>
          </template>
        </el-upload>
      </el-form-item>

      <!-- 预览选中的文件信息 -->
      <el-form-item v-if="selectedFile" label="文件信息">
        <div class="file-info">
          <div><strong>文件名：</strong>{{ selectedFile.name }}</div>
          <div v-if="fileSize"><strong>文件大小：</strong>{{ fileSize }}</div>
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button 
          type="primary" 
          :loading="activating" 
          :disabled="!selectedFile"
          @click="handleActivate"
        >
          激活
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import type { UploadFile } from 'element-plus'
import { useLicenseStore } from '@/stores/license'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'activated': [status: any]
}>()

const licenseStore = useLicenseStore()

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const formData = ref({}) // 表单数据（虽然这里不需要，但 el-form 需要 model）
const fileList = ref<UploadFile[]>([])
const selectedFile = ref<File | null>(null)
const activating = ref(false)

const fileSize = computed(() => {
  if (!selectedFile.value) return ''
  const bytes = selectedFile.value.size
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
})

const handleFileChange = (file: UploadFile) => {
  if (file.raw) {
    selectedFile.value = file.raw
    fileList.value = [file]
  }
}

const handleFileRemove = () => {
  selectedFile.value = null
  fileList.value = []
}

const handleActivate = async () => {
  if (!selectedFile.value) {
    ElMessage.warning('请选择 License 文件')
    return
  }

  activating.value = true
  try {
    const status = await licenseStore.activate(selectedFile.value)
    emit('activated', status)
    handleClose()
  } catch (error: any) {
    // 错误已在 store 中处理
    console.error('激活失败:', error)
  } finally {
    activating.value = false
  }
}

const handleClose = () => {
  fileList.value = []
  selectedFile.value = null
  dialogVisible.value = false
}
</script>

<style scoped lang="scss">
.file-info {
  padding: 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
  
  div {
    margin-bottom: 8px;
    
    &:last-child {
      margin-bottom: 0;
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>

