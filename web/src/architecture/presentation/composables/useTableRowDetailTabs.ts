import { computed, ref, watch, type Ref } from 'vue'
import { useRoute } from 'vue-router'
import { isServiceTreeNodeAdmin } from '@/utils/permissionActors'

interface UseTableRowDetailTabsOptions {
  currentFunction: Ref<any>
  currentUsername: Ref<string | undefined>
  rowData: Ref<Record<string, any> | null>
}

export function useTableRowDetailTabs({
  currentFunction,
  currentUsername,
  rowData
}: UseTableRowDetailTabsOptions) {
  const route = useRoute()
  const activeTab = ref('detail')

  const showPermissionRequestTab = computed(() => {
    const functionNode = currentFunction.value
    if (!functionNode) {
      return false
    }

    if (functionNode.type !== 'package' && functionNode.type !== 'function') {
      return false
    }

    return isServiceTreeNodeAdmin(functionNode, currentUsername.value)
  })

  watch(
    () => rowData.value,
    () => {
      activeTab.value = 'detail'
    }
  )

  watch(
    () => route.query.tab,
    (tab) => {
      const normalizedTab = Array.isArray(tab) ? tab[0] : tab
      if (normalizedTab === 'permissionRequest' && showPermissionRequestTab.value) {
        activeTab.value = 'permissionRequest'
      }
    },
    { immediate: true }
  )

  return {
    activeTab,
    showPermissionRequestTab
  }
}
