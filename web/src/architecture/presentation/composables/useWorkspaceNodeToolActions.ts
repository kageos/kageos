import { ref, type ComputedRef } from 'vue'
import { ElMessage } from 'element-plus'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/architecture/domain/types'

type UpdateHistoryMode = 'app' | 'directory'

export interface UseWorkspaceNodeToolActionsOptions {
  currentApp: ComputedRef<AppType | null>
}

export function useWorkspaceNodeToolActions(options: UseWorkspaceNodeToolActionsOptions) {
  const { currentApp } = options

  const updateHistoryDialogVisible = ref(false)
  const updateHistoryMode = ref<UpdateHistoryMode>('app')
  const updateHistoryAppId = ref(0)
  const updateHistoryAppVersion = ref('')
  const updateHistoryFullCodePath = ref('')

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

  return {
    updateHistoryDialogVisible,
    updateHistoryMode,
    updateHistoryAppId,
    updateHistoryAppVersion,
    updateHistoryFullCodePath,
    handleUpdateHistory,
  }
}
