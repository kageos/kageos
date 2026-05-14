import { onMounted, onUnmounted, type Ref } from 'vue'
import { eventBus } from '../../infrastructure/eventBus'

interface TableDetailRowPayload {
  row: Record<string, any>
  index?: number
  tableData?: any[]
  initialMode?: 'read' | 'edit'
}

interface UseWorkspaceUiEffectsOptions {
  showLeftSidebar: Ref<boolean>
  openDetailDrawer: (row: Record<string, any>, index?: number, tableData?: any[], initialMode?: 'read' | 'edit') => Promise<void> | void
  setupUrlWatch: () => (() => void) | null | void
  handleWorkspaceOpenWorkstation: (payload: any) => void
  initializeMiniWorkstationsFromRoute: () => void
}

export function useWorkspaceUiEffects(options: UseWorkspaceUiEffectsOptions) {
  let unsubscribeTableDetailRow: (() => void) | null = null
  let unsubscribeDetailUrlWatch: (() => void) | null = null
  let unsubscribeWorkspaceOpenWorkstation: (() => void) | null = null

  onMounted(() => {
    const savedLeft = localStorage.getItem('workspace-left-sidebar')
    if (savedLeft !== null) {
      options.showLeftSidebar.value = savedLeft === 'true'
    }

    unsubscribeTableDetailRow = eventBus.on('table:detail-row', async (payload: TableDetailRowPayload) => {
      const { row, index, tableData, initialMode = 'read' } = payload
      await options.openDetailDrawer(row, index, tableData, initialMode)
    })

    unsubscribeDetailUrlWatch = options.setupUrlWatch() || null
    unsubscribeWorkspaceOpenWorkstation = eventBus.on('workspace:open-workstation', options.handleWorkspaceOpenWorkstation)
    options.initializeMiniWorkstationsFromRoute()
  })

  onUnmounted(() => {
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
