import { ref, type ComputedRef } from 'vue'
import { ElMessage } from 'element-plus'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/types'
import { featureFlags } from '@/config/features'

type UpdateHistoryMode = 'app' | 'directory'

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

  const handlePublishToHub = (node: ServiceTreeType) => {
    if (!featureFlags.hub) {
      return
    }
    publishSelectedNode.value = node
    publishToHubDialogVisible.value = true
  }

  const handlePushToHub = (node: ServiceTreeType) => {
    if (!featureFlags.hub) {
      return
    }
    pushSelectedNode.value = node
    pushToHubDialogVisible.value = true
  }

  const openPullFromHubDialog = (initialLink?: string, targetFullCodePath?: string, targetName?: string) => {
    if (!featureFlags.hub) {
      return
    }
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
    handlePublishToHub,
    handlePushToHub,
    openPullFromHubDialog,
    handleUpdateHistory,
    handlePublishSuccess,
    handlePushSuccess,
    handlePullSuccess,
  }
}
