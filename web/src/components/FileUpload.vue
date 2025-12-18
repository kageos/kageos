<template>
  <div class="file-upload">
    <el-upload
      :auto-upload="false"
      :on-change="handleFileSelect"
      :show-file-list="false"
      drag
      class="upload-dragger"
    >
      <el-icon class="el-icon--upload"><upload-filled /></el-icon>
      <div class="el-upload__text">
        将文件拖到此处，或<em>点击上传</em>
      </div>
      <template #tip>
        <div class="el-upload__tip">
          支持拖拽上传，文件大小不超过 100MB
        </div>
      </template>
    </el-upload>

    <!-- 上传进度 -->
    <div v-if="uploading" class="upload-progress">
      <div class="file-info">
        <el-icon><document /></el-icon>
        <span class="file-name">{{ fileName }}</span>
        <span class="file-size">{{ formatSize(uploadedSize) }} / {{ formatSize(totalSize) }}</span>
      </div>

      <el-progress
        :percentage="uploadPercent"
        :status="uploadStatus"
        :stroke-width="12"
      />

      <div class="upload-details">
        <div class="upload-domain" v-if="uploadDomain">
          <el-icon><link /></el-icon>
          <span>上传到: {{ uploadDomain }}</span>
        </div>

        <div class="upload-speed" v-if="uploadPercent < 100">
          <span>速度: {{ uploadSpeed }}</span>
        </div>

        <div class="upload-status" v-else-if="uploadStatus === 'success'">
          <el-icon><check /></el-icon>
          <span class="success">上传成功</span>
        </div>

        <div class="upload-status" v-else-if="uploadStatus === 'exception'">
          <el-icon><close /></el-icon>
          <span class="error">上传失败</span>
        </div>
      </div>

      <!-- 取消按钮 -->
      <el-button
        v-if="uploadPercent < 100"
        @click="cancelUpload"
        type="danger"
        size="small"
        :icon="Close"
      >
        取消上传
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import {
  UploadFilled,
  Document,
  Check,
  Close,
  Link,
} from '@element-plus/icons-vue'
import { uploadFile } from '@/utils/upload'
import type { UploadProgress } from '@/utils/upload/types'

const props = defineProps<{
  router: string  // 函数路径，例如：luobei/test88888/plugins/cashier_desk
}>()

const emit = defineEmits<{
  success: [key: string, fileName: string]
  error: [error: Error]
}>()

// 上传状态
const uploading = ref(false)
const uploadPercent = ref(0)
const uploadStatus = ref<'success' | 'exception' | undefined>()
const fileName = ref('')
const uploadedSize = ref(0)
const totalSize = ref(0)
const uploadDomain = ref('')  // ✨ 上传域名
const startTime = ref(0)
const currentUploader = ref<{ cancel: () => void } | null>(null)

// 计算上传速度
const uploadSpeed = computed(() => {
  if (!startTime.value || uploadedSize.value === 0) return '0 KB/s'

  const elapsed = (Date.now() - startTime.value) / 1000  // 秒
  const speed = uploadedSize.value / elapsed  // 字节/秒

  if (speed < 1024) return `${speed.toFixed(0)} B/s`
  if (speed < 1024 * 1024) return `${(speed / 1024).toFixed(2)} KB/s`
  return `${(speed / (1024 * 1024)).toFixed(2)} MB/s`
})

/**
 * 处理文件选择
 *
 * 流程：
 * 1. 用户拖文件/选择文件
 * 2. 调用 uploadFile() → 内部会先请求后端获取上传凭证（包含域名）
 * 3. 后端返回上传凭证（包含 upload_host, upload_domain, url 等）
 * 4. 根据凭证中的 method 创建对应的上传器
 * 5. 使用上传器执行上传（此时已知道上传域名）
 */
async function handleFileSelect(file: any) {
  const rawFile = file.raw
  if (!rawFile) return

  fileName.value = rawFile.name
  totalSize.value = rawFile.size
  uploading.value = true
  uploadPercent.value = 0
  uploadedSize.value = 0
  uploadStatus.value = undefined
  uploadDomain.value = ''  // 重置域名
  startTime.value = Date.now()

  try {
    // ✨ 关键：调用 uploadFile，内部会先请求后端获取上传凭证（包含域名）
    // uploadFile() 的流程：
    //   1. getUploadCredentials() → 请求 /storage/api/v1/upload_token
    //   2. 后端返回：{ method, url, upload_host, upload_domain, ... }
    //   3. UploaderFactory.create(method) → 创建对应的上传器
    //   4. uploader.upload(credentials, file, onProgress) → 执行上传
    const key = await uploadFile(
      props.router,
      rawFile,
      (progress: UploadProgress & { uploadDomain?: string }) => {
        // 进度回调
        uploadPercent.value = progress.percent
        uploadedSize.value = progress.loaded
        totalSize.value = progress.total

        // ✨ 从进度回调中获取上传域名（如果上传器返回了）
        if (progress.uploadDomain) {
          uploadDomain.value = progress.uploadDomain
        }
      }
    )

    uploadStatus.value = 'success'
    ElMessage.success('上传成功')
    emit('success', key, rawFile.name)

    // 上传成功后立即隐藏进度条（用户已看到成功提示）
    uploading.value = false
    uploadDomain.value = ''

  } catch (err: any) {
    uploadStatus.value = 'exception'
    ElMessage.error(`上传失败: ${err.message}`)
    emit('error', err)
    uploading.value = false
    uploadDomain.value = ''
  }
}

function cancelUpload() {
  if (currentUploader.value) {
    currentUploader.value.cancel()
    uploading.value = false
    uploadDomain.value = ''
    ElMessage.warning('已取消上传')
  }
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
}
</script>

<style scoped>
.file-upload {
  padding: 20px;
}

.upload-dragger {
  margin-bottom: 20px;
}

.upload-progress {
  margin-top: 20px;
  padding: 20px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: #f5f7fa;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  font-size: 14px;
  color: #606266;
}

.file-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-size {
  color: #909399;
  margin-left: auto;
}

.upload-details {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 13px;
}

.upload-domain {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #606266;
}

.upload-speed {
  color: #606266;
}

.upload-status {
  display: flex;
  align-items: center;
  gap: 4px;
}

.upload-status .success {
  color: #67c23a;
}

.upload-status .error {
  color: #f56c6c;
}

.el-button {
  margin-top: 12px;
}
</style>

