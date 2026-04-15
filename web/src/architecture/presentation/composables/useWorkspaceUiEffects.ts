import { onMounted, onUnmounted, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import { eventBus } from '../../infrastructure/eventBus'
import type { App as AppType } from '@/types'

interface TableDetailRowPayload {
  row: Record<string, any>
  index?: number
  tableData?: any[]
  initialMode?: 'read' | 'edit'
}

interface UseWorkspaceUiEffectsOptions {
  currentApp: () => AppType | null
  showLeftSidebar: Ref<boolean>
  showRightSidebar: Ref<boolean>
  openPullFromHubDialog: (initialLink?: string) => void
  openDetailDrawer: (row: Record<string, any>, index?: number, tableData?: any[], initialMode?: 'read' | 'edit') => Promise<void> | void
  setupUrlWatch: () => (() => void) | null | void
  handleWorkspaceOpenWorkstation: (payload: any) => void
  initializeMiniWorkstationsFromRoute: () => void
}

export function useWorkspaceUiEffects(options: UseWorkspaceUiEffectsOptions) {
  let unsubscribeTableDetailRow: (() => void) | null = null
  let unsubscribeDetailUrlWatch: (() => void) | null = null
  let unsubscribeWorkspaceOpenWorkstation: (() => void) | null = null

  const handleGlobalPaste = async (event: ClipboardEvent) => {
    const target = event.target instanceof HTMLElement ? event.target : null
    if (target && (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.isContentEditable ||
      target.closest('.ProseMirror') ||
      target.closest('.rich-text-widget') ||
      target.closest('.editor-container')
    )) {
      return
    }

    const pastedText = event.clipboardData?.getData('text')
    if (pastedText && pastedText.trim().startsWith('hub://')) {
      event.preventDefault()

      if (!options.currentApp()) {
        ElMessage.warning('请先选择应用')
        return
      }

      options.openPullFromHubDialog(pastedText.trim())
      ElMessage.info('检测到 Hub 链接，已打开安装对话框')
    }
  }

  onMounted(() => {
    const savedLeft = localStorage.getItem('workspace-left-sidebar')
    if (savedLeft !== null) {
      options.showLeftSidebar.value = savedLeft === 'true'
    }

    const savedRight = localStorage.getItem('workspace-right-sidebar')
    if (savedRight !== null) {
      options.showRightSidebar.value = savedRight === 'true'
    }

    unsubscribeTableDetailRow = eventBus.on('table:detail-row', async (payload: TableDetailRowPayload) => {
      const { row, index, tableData, initialMode = 'read' } = payload
      await options.openDetailDrawer(row, index, tableData, initialMode)
    })

    unsubscribeDetailUrlWatch = options.setupUrlWatch() || null
    document.addEventListener('paste', handleGlobalPaste)
    unsubscribeWorkspaceOpenWorkstation = eventBus.on('workspace:open-workstation', options.handleWorkspaceOpenWorkstation)
    options.initializeMiniWorkstationsFromRoute()
  })

  onUnmounted(() => {
    document.removeEventListener('paste', handleGlobalPaste)
    if (unsubscribeTableDetailRow) {
      unsubscribeTableDetailRow()
    }
    if (unsubscribeDetailUrlWatch) {
      unsubscribeDetailUrlWatch()
    }
    if (unsubscribeWorkspaceOpenWorkstation) {
      unsubscribeWorkspaceOpenWorkstation()
    }
  })
}
