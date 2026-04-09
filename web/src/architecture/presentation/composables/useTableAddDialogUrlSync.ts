import { nextTick, onMounted, onUnmounted, type Ref } from 'vue'
import { useRoute } from 'vue-router'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { resolveTableAddDialogVisibility } from '../views/utils/tableViewRouteRuntime'

interface UseTableAddDialogUrlSyncOptions {
  createDialogVisible: Ref<boolean>
  hasAddCallback: () => boolean
  isMounted: () => boolean
}

export function useTableAddDialogUrlSync(options: UseTableAddDialogUrlSyncOptions) {
  const route = useRoute()
  let unsubscribeAddDialogQueryChanged: (() => void) | null = null

  const restoreAddDialogFromURL = (query: any): void => {
    options.createDialogVisible.value = resolveTableAddDialogVisibility({
      query,
      hasAddCallback: options.hasAddCallback(),
      isMounted: options.isMounted(),
      currentVisible: options.createDialogVisible.value
    })
  }

  onMounted(() => {
    if (route.query._tab === 'OnTableAddRow') {
      nextTick(() => {
        restoreAddDialogFromURL(route.query)
      })
    }

    unsubscribeAddDialogQueryChanged = eventBus.on(RouteEvent.queryChanged, async (payload: { query: any, source: string }) => {
      if (payload.source === 'router-change') {
        restoreAddDialogFromURL(payload.query)
      }
    })
  })

  onUnmounted(() => {
    if (unsubscribeAddDialogQueryChanged) {
      unsubscribeAddDialogQueryChanged()
    }
  })

  return {
    restoreAddDialogFromURL,
  }
}
