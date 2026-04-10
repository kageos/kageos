import { computed, nextTick, ref, watch, type Ref } from 'vue'
import type { TabPaneName } from 'element-plus'
import { useRoute } from 'vue-router'
import { isServiceTreeNodeAdmin } from '@/utils/permissionActors'

interface LoadableOperateLogSection {
  load: () => void
}

interface LoadablePermissionRequestList {
  loadRequests: () => void
}

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
  const operateLogSectionRef = ref<LoadableOperateLogSection | null>(null)
  const permissionRequestListRef = ref<LoadablePermissionRequestList | null>(null)

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

  const handleTabChange = (tabName: TabPaneName) => {
    if (tabName === 'operateLog' && operateLogSectionRef.value) {
      operateLogSectionRef.value.load()
    } else if (tabName === 'permissionRequest' && permissionRequestListRef.value) {
      permissionRequestListRef.value.loadRequests()
    }
  }

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
        nextTick(() => {
          permissionRequestListRef.value?.loadRequests()
        })
      }
    },
    { immediate: true }
  )

  return {
    activeTab,
    operateLogSectionRef,
    permissionRequestListRef,
    showPermissionRequestTab,
    handleTabChange
  }
}
