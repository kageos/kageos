import { ref, type ComputedRef } from 'vue'
import { ElMessage } from 'element-plus'
import { addFunctionsToDirectory } from '@/api/service-tree'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/types'

type UpdateHistoryMode = 'app' | 'directory'

async function readFileAsText(file: File): Promise<string> {
  return await new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error)
    reader.readAsText(file, 'utf-8')
  })
}

export interface UseWorkspaceNodeToolActionsOptions {
  currentApp: ComputedRef<AppType | null>
  handleRefreshTree: () => Promise<void>
}

export function useWorkspaceNodeToolActions(options: UseWorkspaceNodeToolActionsOptions) {
  const { currentApp, handleRefreshTree } = options

  const publishToHubDialogVisible = ref(false)
  const publishSelectedNode = ref<ServiceTreeType | null>(null)
  const pushToHubDialogVisible = ref(false)
  const pushSelectedNode = ref<ServiceTreeType | null>(null)
  const pullFromHubDialogVisible = ref(false)
  const pastedHubLink = ref('')
  const pullFromHubTargetPath = ref('')
  const pullFromHubTargetName = ref('')

  const updateHistoryDialogVisible = ref(false)
  const updateHistoryMode = ref<UpdateHistoryMode>('app')
  const updateHistoryAppId = ref(0)
  const updateHistoryAppVersion = ref('')
  const updateHistoryFullCodePath = ref('')

  const importGoFileInputRef = ref<HTMLInputElement | null>(null)
  const importGoTargetNode = ref<ServiceTreeType | null>(null)

  const handleImportGoFiles = (node: ServiceTreeType) => {
    const fullCodePath = node.full_code_path
    if (!fullCodePath) {
      ElMessage.warning('该目录无完整路径')
      return
    }
    importGoTargetNode.value = node
    importGoFileInputRef.value?.click()
  }

  async function doImportGoFiles(files: FileList | File[], fullCodePath: string) {
    let ok = 0
    let fail = 0
    try {
      const fileArray = Array.from(files)
      for (let i = 0; i < fileArray.length; i++) {
        const file = fileArray[i]
        if (!file || !file.name.toLowerCase().endsWith('.go')) continue
        const content = await readFileAsText(file)
        const fileName = file.name.endsWith('.go') ? file.name : file.name + '.go'
        try {
          const res = await addFunctionsToDirectory({
            full_code_path: fullCodePath,
            file_name: fileName,
            source_code: content,
            skip_build: true
          })
          if (res?.success !== false) ok++
          else fail++
        } catch (err: any) {
          fail++
          ElMessage.warning(`${file.name}: ${err?.message || err?.response?.data?.msg || '写入失败'}`)
        }
      }
      if (ok > 0) {
        ElMessage.success(`已导入 ${ok} 个代码文件，可在工作台更新后生效。`)
        await handleRefreshTree()
      }
      if (fail > 0 && ok === 0) {
        ElMessage.error('导入失败')
      }
    } finally {
      importGoTargetNode.value = null
    }
  }

  const onImportGoFilesSelected = async (e: Event) => {
    const input = e.target as HTMLInputElement
    const files = input.files
    if (!files?.length || !importGoTargetNode.value) {
      input.value = ''
      return
    }
    const fullCodePath = importGoTargetNode.value.full_code_path!
    input.value = ''
    await doImportGoFiles(files, fullCodePath)
  }

  const handlePublishToHub = (node: ServiceTreeType) => {
    publishSelectedNode.value = node
    publishToHubDialogVisible.value = true
  }

  const handlePushToHub = (node: ServiceTreeType) => {
    pushSelectedNode.value = node
    pushToHubDialogVisible.value = true
  }

  const openPullFromHubDialog = (initialLink?: string, targetFullCodePath?: string, targetName?: string) => {
    pastedHubLink.value = initialLink ?? ''
    pullFromHubTargetPath.value = targetFullCodePath ?? ''
    pullFromHubTargetName.value = targetName ?? ''
    pullFromHubDialogVisible.value = true
  }

  const handleUpdateHistory = (node?: ServiceTreeType) => {
    if (!currentApp.value) {
      ElMessage.warning('请先选择应用')
      return
    }

    const appId = currentApp.value.id
    if (!appId || appId === 0) {
      ElMessage.error('应用ID无效，无法加载变更记录。请刷新页面后重试。')
      return
    }

    if (node) {
      updateHistoryMode.value = 'directory'
      updateHistoryAppId.value = appId
      updateHistoryFullCodePath.value = node.full_code_path || ''
      updateHistoryAppVersion.value = ''
    } else {
      updateHistoryMode.value = 'app'
      updateHistoryAppId.value = appId
      updateHistoryAppVersion.value = ''
      updateHistoryFullCodePath.value = ''
    }

    updateHistoryDialogVisible.value = true
  }

  const handlePublishSuccess = async () => {
    await handleRefreshTree()
  }

  const handlePushSuccess = async () => {
    await handleRefreshTree()
  }

  const handlePullSuccess = async () => {
    pastedHubLink.value = ''
    pullFromHubTargetPath.value = ''
    pullFromHubTargetName.value = ''
    await handleRefreshTree()
  }

  return {
    publishToHubDialogVisible,
    publishSelectedNode,
    pushToHubDialogVisible,
    pushSelectedNode,
    pullFromHubDialogVisible,
    pastedHubLink,
    pullFromHubTargetPath,
    pullFromHubTargetName,
    updateHistoryDialogVisible,
    updateHistoryMode,
    updateHistoryAppId,
    updateHistoryAppVersion,
    updateHistoryFullCodePath,
    importGoFileInputRef,
    handleImportGoFiles,
    onImportGoFilesSelected,
    handlePublishToHub,
    handlePushToHub,
    openPullFromHubDialog,
    handleUpdateHistory,
    handlePublishSuccess,
    handlePushSuccess,
    handlePullSuccess,
  }
}
