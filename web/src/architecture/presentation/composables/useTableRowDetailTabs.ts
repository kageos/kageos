import { ref, watch, type Ref } from 'vue'

interface UseTableRowDetailTabsOptions {
  rowData: Ref<Record<string, any> | null>
}

export function useTableRowDetailTabs({
  rowData
}: UseTableRowDetailTabsOptions) {
  const activeTab = ref('detail')

  watch(
    () => rowData.value,
    () => {
      activeTab.value = 'detail'
    }
  )

  return {
    activeTab
  }
}
