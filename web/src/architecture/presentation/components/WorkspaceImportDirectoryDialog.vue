<template>
  <el-dialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="560px"
    class="workspace-import-directory-dialog"
    :close-on-click-modal="false"
    @closed="resetForm"
  >
    <div class="import-target" data-testid="import-directory-target">
      <span class="import-target-label">目标目录</span>
      <strong>{{ targetLabel }}</strong>
      <code>{{ targetPath || '-' }}</code>
    </div>

    <el-tabs v-model="activeSource" class="import-source-tabs" data-testid="import-directory-tabs">
      <el-tab-pane name="hub">
        <template #label>
          <span class="import-source-label">
            <el-icon><Link /></el-icon>
            Hub 导入
          </span>
        </template>
        <el-form label-position="top" class="import-source-form" @submit.prevent>
          <el-form-item label="Hub 安装命令">
            <el-input
              v-model="hubInstallLink"
              type="textarea"
              :autosize="{ minRows: 3, maxRows: 5 }"
              placeholder="kageos install user_1210227080/meeting_room_booking:0.1.0 --key ..."
              maxlength="1200"
              show-word-limit
              clearable
              data-testid="import-directory-hub-link"
            />
          </el-form-item>
          <el-form-item label="安装密钥">
            <el-input
              v-model="hubInstallKey"
              placeholder="可选"
              maxlength="240"
              clearable
              show-password
              data-testid="import-directory-hub-key"
            />
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane name="json">
        <template #label>
          <span class="import-source-label">
            <el-icon><Upload /></el-icon>
            JSON 导入
          </span>
        </template>
        <div class="json-import-box" :class="{ 'has-file': selectedJsonFile }">
          <input
            ref="jsonInputRef"
            type="file"
            accept=".json,application/json"
            class="json-file-input"
            data-testid="import-directory-json-input"
            @change="handleJsonFileSelect"
          />
          <el-icon class="json-import-icon"><UploadFilled /></el-icon>
          <div class="json-file-name">
            {{ selectedJsonFile?.name || '选择 capability bundle JSON 文件' }}
          </div>
          <el-button
            :icon="FolderOpened"
            data-testid="import-directory-json-select"
            @click="openJsonFilePicker"
          >
            选择文件
          </el-button>
        </div>
      </el-tab-pane>
    </el-tabs>

    <template #footer>
      <span class="dialog-footer">
        <el-button data-testid="import-directory-cancel" @click="dialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :icon="Upload"
          :loading="importing"
          :disabled="!canSubmit"
          data-testid="import-directory-submit"
          @click="handleSubmit"
        >
          导入
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElLoading, ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { FolderOpened, Link, Upload, UploadFilled } from '@element-plus/icons-vue'
import type { ServiceTree } from '@/architecture/domain/types'
import {
  installCapabilityBundle,
  installCapabilityBundleFromURL
} from '@/architecture/presentation/context/api/service-tree'
import { parseCapabilityBundleJson } from '@/architecture/presentation/utils/directoryBundleFile'
import { parseHubInstallInput } from '@/architecture/presentation/utils/hubInstallCommand'

type ImportSource = 'hub' | 'json'

const props = defineProps<{
  visible: boolean
  targetNode: ServiceTree | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'imported', path?: string): void
}>()

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value)
})

const activeSource = ref<ImportSource>('hub')
const hubInstallLink = ref('')
const hubInstallKey = ref('')
const selectedJsonFile = ref<File | null>(null)
const jsonInputRef = ref<HTMLInputElement | null>(null)
const importing = ref(false)

const targetPath = computed(() => props.targetNode?.full_code_path || '')

const targetLabel = computed(() => {
  const node = props.targetNode
  if (!node) return '未选择目录'
  return node.name || node.code || node.full_code_path || '未命名目录'
})

const dialogTitle = computed(() => `导入目录到「${targetLabel.value}」`)

const canSubmit = computed(() => {
  if (importing.value || !targetPath.value || props.targetNode?.type !== 'package') {
    return false
  }
  if (activeSource.value === 'hub') {
    return hubInstallLink.value.trim().length > 0
  }
  return Boolean(selectedJsonFile.value)
})

function resetForm() {
  activeSource.value = 'hub'
  hubInstallLink.value = ''
  hubInstallKey.value = ''
  selectedJsonFile.value = null
  if (jsonInputRef.value) {
    jsonInputRef.value.value = ''
  }
}

function showBlockingLoading(text: string) {
  return ElLoading.service({
    lock: true,
    text,
    background: 'rgba(15, 23, 42, 0.36)'
  })
}

function openJsonFilePicker() {
  jsonInputRef.value?.click()
}

function handleJsonFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  selectedJsonFile.value = input.files?.[0] || null
}

async function handleSubmit() {
  if (activeSource.value === 'hub') {
    await submitHubImport()
    return
  }
  await submitJsonImport()
}

async function submitHubImport() {
  const targetNode = props.targetNode
  if (!targetNode?.full_code_path || targetNode.type !== 'package') {
    ElMessage.warning('请选择一个目录作为导入目标')
    return
  }

  const command = parseHubInstallInput(hubInstallLink.value, hubInstallKey.value)
  if (!command) {
    ElMessage.warning('请输入有效的 Hub 安装命令')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要把 Hub 目录导入到「${targetLabel.value}」下吗？\n\n目标目录：${targetNode.full_code_path}\n来源：${command.displaySource}\n\n同名目录或文件会按导入规则覆盖。`,
      'Hub 导入目录',
      {
        confirmButtonText: '导入',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    importing.value = true
    const loadingNotify = ElNotification({
      title: '导入中',
      message: '正在从 Hub 下载并导入目录，请稍候...',
      type: 'info',
      position: 'top-right',
      duration: 0
    })
    const loadingInstance = showBlockingLoading('正在从 Hub 导入目录并更新函数列表，请稍候...')

    try {
      const resp = await installCapabilityBundleFromURL({
        target_directory_path: targetNode.full_code_path,
        overwrite: true,
        force_diff: true,
        bundle_url: command.bundleUrl,
        install_key: command.installKey
      })
      loadingNotify.close()
      ElNotification.success({
        title: '导入完成',
        message: resp.message || `已导入到 ${resp.target_directory_path || targetNode.full_code_path}`,
        position: 'top-right'
      })
      emit('imported', resp.target_directory_path || targetNode.full_code_path)
      dialogVisible.value = false
    } catch (error: any) {
      const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || '导入失败'
      ElMessage.error(message)
    } finally {
      loadingNotify.close()
      loadingInstance.close()
      importing.value = false
    }
  } catch {
    // 用户取消
  }
}

async function submitJsonImport() {
  const targetNode = props.targetNode
  const file = selectedJsonFile.value
  if (!targetNode?.full_code_path || targetNode.type !== 'package') {
    ElMessage.warning('请选择一个目录作为导入目标')
    return
  }
  if (!file) {
    ElMessage.warning('请选择 JSON 文件')
    return
  }

  let loadingInstance: ReturnType<typeof showBlockingLoading> | null = null
  try {
    const bundle = parseCapabilityBundleJson(await file.text())
    await ElMessageBox.confirm(
      `将能力包「${bundle.name || file.name}」导入到 ${targetNode.full_code_path}，同名文件会被覆盖。`,
      'JSON 导入目录',
      {
        confirmButtonText: '覆盖导入',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    importing.value = true
    loadingInstance = showBlockingLoading('正在导入目录并更新函数列表，请稍候...')
    const resp = await installCapabilityBundle({
      target_directory_path: targetNode.full_code_path,
      overwrite: true,
      force_diff: true,
      bundle
    })
    ElMessage.success(resp.message || '导入成功')
    emit('imported', resp.target_directory_path || targetNode.full_code_path)
    dialogVisible.value = false
  } catch (error: any) {
    if (error === 'cancel' || error === 'close') {
      return
    }
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || '导入失败'
    ElMessage.error(message)
  } finally {
    importing.value = false
    loadingInstance?.close()
  }
}
</script>

<style scoped lang="scss">
.import-target {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 6px 10px;
  padding: 12px;
  margin-bottom: 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);

  strong,
  code {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  code {
    grid-column: 2;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
}

.import-target-label {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.import-source-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: 16px;
  }
}

.import-source-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.import-source-form {
  margin-bottom: -8px;
}

.json-import-box {
  display: grid;
  place-items: center;
  gap: 10px;
  min-height: 188px;
  padding: 24px;
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
  text-align: center;
  transition: border-color 0.18s ease, background 0.18s ease;

  &.has-file {
    border-color: var(--el-color-primary);
    background: rgba(var(--el-color-primary-rgb), 0.06);
  }
}

.json-file-input {
  display: none;
}

.json-import-icon {
  color: var(--el-color-primary);
  font-size: 28px;
}

.json-file-name {
  max-width: 100%;
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
