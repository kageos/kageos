<template>
  <el-dialog
    v-model="dialogVisible"
    width="min(640px, calc(100vw - 32px))"
    class="workspace-import-directory-dialog"
    :close-on-click-modal="false"
    @closed="resetForm"
  >
    <template #header>
      <div class="import-dialog-header">
        <span class="import-dialog-icon">
          <el-icon><Upload /></el-icon>
        </span>
        <div class="import-dialog-title-block">
          <div class="import-dialog-title">{{ dialogTitle }}</div>
          <div class="import-dialog-subtitle">任务会在后台执行，完成后自动刷新服务目录。</div>
        </div>
      </div>
    </template>

    <section class="import-target" data-testid="import-directory-target">
      <span class="import-target-label">目标服务目录</span>
      <div class="import-target-main">
        <strong>{{ targetLabel }}</strong>
        <code>{{ targetPath || '-' }}</code>
      </div>
    </section>

    <div class="import-notice">
      <el-icon><InfoFilled /></el-icon>
      <span>同名服务目录或文件会按导入规则覆盖。</span>
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
            <div class="hub-source-actions">
              <el-button link type="primary" :icon="Compass" @click="openHubDirectory">
                去 Hub 查找服务目录
              </el-button>
            </div>
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
            {{ selectedJsonFile?.name || '选择服务目录 JSON 文件' }}
          </div>
          <el-button
            :icon="FolderOpened"
            data-testid="import-directory-json-select"
            @click="openJsonFilePicker"
          >
            选择 JSON
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
          开始导入
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, h, ref } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'
import { Compass, FolderOpened, InfoFilled, Link, Upload, UploadFilled } from '@element-plus/icons-vue'
import type { CapabilityBundle, ServiceTree } from '@/architecture/domain/types'
import {
  installCapabilityBundle,
  installCapabilityBundleFromURL
} from '@/architecture/presentation/context/api/service-tree'
import { parseCapabilityBundleJson } from '@/architecture/presentation/utils/directoryBundleFile'
import { parseHubInstallInput } from '@/architecture/presentation/utils/hubInstallCommand'
import { Z_INDEX } from '@/architecture/presentation/constants/zIndex'
import { getKageosHubURL, openExternalURL } from '@/architecture/shared/config/externalLinks'

type ImportSource = 'hub' | 'json'
type NotificationHandle = { close: () => void }

interface InstallDirectoryResp {
  message: string
  directory_count: number
  file_count: number
  target_directory_path: string
  created_paths?: string[]
  written_paths?: string[]
  old_version?: string
  new_version?: string
  warnings?: string[]
}

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
  if (!node) return '未选择服务目录'
  return node.name || node.code || node.full_code_path || '未命名服务目录'
})

const dialogTitle = computed(() => `导入服务目录到「${targetLabel.value}」`)

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

function openJsonFilePicker() {
  jsonInputRef.value?.click()
}

function openHubDirectory() {
  openExternalURL(getKageosHubURL())
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

function directoryTaskNotificationOptions() {
  return {
    appendTo: 'body',
    customClass: 'workspace-task-notification',
    offset: 72,
    position: 'top-right' as const,
    zIndex: Z_INDEX.notification
  }
}

function renderImportResultMessage(resp: InstallDirectoryResp, fallbackPath: string) {
  const lines = [
    `目标：${resp.target_directory_path || fallbackPath}`,
    `写入：${resp.directory_count || 0} 个服务目录，${resp.file_count || 0} 个文件`
  ]
  if (resp.old_version || resp.new_version) {
    lines.push(`版本：${resp.old_version || '-'} → ${resp.new_version || '-'}`)
  }
  if (resp.warnings?.length) {
    const warnings = resp.warnings.slice(0, 2).join('；')
    const suffix = resp.warnings.length > 2 ? ` 等 ${resp.warnings.length} 条` : ''
    lines.push(`提醒：${warnings}${suffix}`)
  }

  return h(
    'div',
    { class: 'workspace-update-notification' },
    lines.map((line) => h('div', { class: 'workspace-update-notification-line' }, line))
  )
}

function getErrorMessage(error: any, fallback: string) {
  return error?.response?.data?.msg || error?.response?.data?.message || error?.message || fallback
}

async function runDirectoryImportTask(options: {
  targetPath: string
  targetLabel: string
  sourceLabel: string
  request: () => Promise<InstallDirectoryResp>
}) {
  importing.value = true
  dialogVisible.value = false

  const progressNotification = ElNotification({
    ...directoryTaskNotificationOptions(),
    type: 'info',
    title: '服务目录导入中',
    message: `正在后台导入「${options.sourceLabel}」到「${options.targetLabel}」，页面可以继续使用。`,
    duration: 0,
    showClose: true
  }) as NotificationHandle

  try {
    const resp = await options.request()
    progressNotification.close()
    ElNotification({
      ...directoryTaskNotificationOptions(),
      type: resp.warnings?.length ? 'warning' : 'success',
      title: resp.warnings?.length ? '服务目录导入完成（有提醒）' : '服务目录导入完成',
      message: renderImportResultMessage(resp, options.targetPath),
      duration: resp.warnings?.length ? 0 : 9000
    })
    emit('imported', resp.target_directory_path || options.targetPath)
  } catch (error: any) {
    progressNotification.close()
    ElNotification({
      ...directoryTaskNotificationOptions(),
      type: 'error',
      title: '服务目录导入失败',
      message: getErrorMessage(error, '导入失败'),
      duration: 0
    })
  } finally {
    importing.value = false
  }
}

async function submitHubImport() {
  const targetNode = props.targetNode
  if (!targetNode?.full_code_path || targetNode.type !== 'package') {
    ElMessage.warning('请选择一个服务目录作为导入目标')
    return
  }

  const command = parseHubInstallInput(hubInstallLink.value, hubInstallKey.value)
  if (!command) {
    ElMessage.warning('请输入有效的 Hub 安装命令')
    return
  }

  const targetPathSnapshot = targetNode.full_code_path
  const targetLabelSnapshot = targetLabel.value
  void runDirectoryImportTask({
    targetPath: targetPathSnapshot,
    targetLabel: targetLabelSnapshot,
    sourceLabel: command.displaySource,
    request: () => installCapabilityBundleFromURL({
      target_directory_path: targetPathSnapshot,
      overwrite: true,
      force_diff: true,
      bundle_subpath: command.bundleSubpath,
      bundle_url: command.bundleUrl,
      install_key: command.installKey
    })
  })
}

async function submitJsonImport() {
  const targetNode = props.targetNode
  const file = selectedJsonFile.value
  if (!targetNode?.full_code_path || targetNode.type !== 'package') {
    ElMessage.warning('请选择一个服务目录作为导入目标')
    return
  }
  if (!file) {
    ElMessage.warning('请选择 JSON 文件')
    return
  }

  let bundle: CapabilityBundle
  try {
    bundle = parseCapabilityBundleJson(await file.text())
  } catch (error: any) {
    ElMessage.error(getErrorMessage(error, 'JSON 文件格式不正确'))
    return
  }

  const targetPathSnapshot = targetNode.full_code_path
  const targetLabelSnapshot = targetLabel.value
  const sourceLabel = bundle.name || file.name
  void runDirectoryImportTask({
    targetPath: targetPathSnapshot,
    targetLabel: targetLabelSnapshot,
    sourceLabel,
    request: () => installCapabilityBundle({
      target_directory_path: targetPathSnapshot,
      overwrite: true,
      force_diff: true,
      bundle
    })
  })
}
</script>

<style scoped lang="scss">
:global(.workspace-import-directory-dialog.el-dialog) {
  border-radius: 12px;
  box-shadow: 0 24px 72px rgba(15, 23, 42, 0.22);
}

:global(.workspace-import-directory-dialog .el-dialog__header) {
  padding: 20px 24px 12px;
  margin-right: 0;
}

:global(.workspace-import-directory-dialog .el-dialog__body) {
  padding: 0 24px 8px;
}

:global(.workspace-import-directory-dialog .el-dialog__footer) {
  padding: 14px 24px 20px;
}

.import-dialog-header {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.import-dialog-icon {
  display: inline-grid;
  place-items: center;
  flex: 0 0 38px;
  width: 38px;
  height: 38px;
  border: 1px solid rgba(var(--el-color-primary-rgb), 0.18);
  border-radius: 10px;
  background: rgba(var(--el-color-primary-rgb), 0.08);
  color: var(--el-color-primary);
  font-size: 18px;
}

.import-dialog-title-block {
  min-width: 0;
}

.import-dialog-title {
  color: var(--el-text-color-primary);
  font-size: 18px;
  font-weight: 700;
  line-height: 1.35;
}

.import-dialog-subtitle {
  margin-top: 2px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.45;
}

.import-target {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px;
  margin-bottom: 14px;
  border: 1px solid rgba(var(--el-color-primary-rgb), 0.14);
  border-radius: 10px;
  background: linear-gradient(180deg, rgba(var(--el-color-primary-rgb), 0.06), rgba(var(--el-color-primary-rgb), 0.025));

  .import-target-main {
    min-width: 0;
    flex: 1 1 auto;
  }

  strong,
  code {
    display: block;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  code {
    margin-top: 3px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
}

.import-target-label {
  flex: 0 0 auto;
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--el-fill-color-blank);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.import-notice {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 10px;
  margin-bottom: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.45;

  .el-icon {
    flex: 0 0 auto;
    color: var(--el-color-warning);
  }
}

.import-source-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: 18px;
  }

  :deep(.el-tabs__nav-wrap::after) {
    height: 1px;
    background: var(--el-border-color-lighter);
  }

  :deep(.el-tabs__item) {
    height: 36px;
    color: var(--el-text-color-secondary);
    font-weight: 650;
  }

  :deep(.el-tabs__item.is-active) {
    color: var(--el-color-primary);
  }
}

.import-source-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.import-source-form {
  margin-bottom: -8px;

  :deep(.el-form-item__label) {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    font-weight: 650;
  }

  :deep(.el-textarea__inner),
  :deep(.el-input__wrapper) {
    border-radius: 8px;
    box-shadow: 0 0 0 1px var(--el-border-color-light) inset;
  }

  :deep(.el-textarea__inner:focus),
  :deep(.el-input__wrapper.is-focus) {
    box-shadow: 0 0 0 1px rgba(var(--el-color-primary-rgb), 0.48) inset, 0 0 0 3px rgba(var(--el-color-primary-rgb), 0.08);
  }
}

.hub-source-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 6px;

  :deep(.el-button) {
    height: 24px;
    padding: 0;
    font-size: 12px;
    font-weight: 650;
  }
}

.json-import-box {
  display: grid;
  place-items: center;
  gap: 12px;
  min-height: 180px;
  padding: 24px;
  border: 1px dashed var(--el-border-color-light);
  border-radius: 10px;
  background: var(--el-fill-color-lighter);
  text-align: center;
  transition: border-color 0.18s ease, background 0.18s ease, box-shadow 0.18s ease;

  &.has-file {
    border-color: var(--el-color-primary);
    background: rgba(var(--el-color-primary-rgb), 0.06);
    box-shadow: 0 0 0 3px rgba(var(--el-color-primary-rgb), 0.08);
  }
}

.json-file-input {
  display: none;
}

.json-import-icon {
  color: var(--el-color-primary);
  font-size: 30px;
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

  :deep(.el-button) {
    border-radius: 8px;
  }
}
</style>
