import { computed, nextTick, ref, watch, type ComputedRef } from 'vue'
import type { RouteLocationNormalizedLoaded, Router, LocationQueryValue } from 'vue-router'
import { ElMessage } from 'element-plus'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { isServiceTreeNodeAdmin } from '@/utils/permissionActors'
import { useAuthStore } from '@/stores/auth'
import type { FunctionDetail } from '@/architecture/domain/types'
import type { ServiceTree } from '../../domain/services/WorkspaceDomainService'

type FunctionTabName = 'content' | 'permission' | 'operateLog' | 'scheduledTask'
type FunctionPermissionTabName = 'request' | 'manage'

type FunctionFormViewRef = {
  applyOperateLog: (payload: {
    requestBody?: Record<string, any> | null
    responseBody?: Record<string, any> | null
    responseMetadata?: Record<string, any> | null
  }) => Promise<void>
}

type PermissionRequestListRef = {
  loadRequests: () => void
}

type PermissionManageListRef = {
  loadPermissions: () => void
}

type FormOperateLogSectionRef = {
  loadLogs: (options?: { page?: number }) => void
  openWithFilters?: (filters: {
    requestUser?: string
    traceId?: string
    keyword?: string
    status?: string
    source?: string
  }) => void
}

export interface UseWorkspaceFunctionTabsOptions {
  route: RouteLocationNormalizedLoaded
  router: Router
  currentFunction: ComputedRef<ServiceTree | null>
  currentFunctionDetail: ComputedRef<FunctionDetail | null> | { value: FunctionDetail | null }
}

function normalizePanelQuery(tab: LocationQueryValue | LocationQueryValue[] | undefined): string | null {
  if (Array.isArray(tab)) {
    return tab[0] ?? null
  }

  return typeof tab === 'string' ? tab : null
}

export function useWorkspaceFunctionTabs(options: UseWorkspaceFunctionTabsOptions) {
  const { route, router, currentFunction, currentFunctionDetail } = options
  const authStore = useAuthStore()

  const functionActiveTab = ref<FunctionTabName>('content')
  const functionPermissionTab = ref<FunctionPermissionTabName>('request')
  const functionFormViewRef = ref<FunctionFormViewRef | null>(null)
  const functionPermissionRequestListRef = ref<PermissionRequestListRef | null>(null)
  const functionPermissionManageListRef = ref<PermissionManageListRef | null>(null)
  const formOperateLogSectionRef = ref<FormOperateLogSectionRef | null>(null)

  const showFunctionPermissionRequestTab = computed(() => {
    if (!currentFunction.value || currentFunction.value.type !== 'function') {
      return false
    }

    return isServiceTreeNodeAdmin(currentFunction.value, authStore.user?.username)
  })

  const showFormOperateLogTab = computed(() => {
    return currentFunction.value?.type === 'function' && currentFunctionDetail.value?.template_type === TEMPLATE_TYPE.FORM
  })

  const showScheduledTaskTab = computed(() => {
    return currentFunction.value?.type === 'function' && !!currentFunction.value?.full_code_path
  })

  const showFunctionTabsWrapper = computed(() => {
    return showFunctionPermissionRequestTab.value || showFormOperateLogTab.value || showScheduledTaskTab.value
  })

  const loadCurrentFunctionPermissionTab = () => {
    if (functionPermissionTab.value === 'manage') {
      nextTick(() => {
        functionPermissionManageListRef.value?.loadPermissions()
      })
      return
    }

    nextTick(() => {
      functionPermissionRequestListRef.value?.loadRequests()
    })
  }

  const getFunctionTabQueryValue = () => {
    switch (functionActiveTab.value) {
      case 'permission':
        return functionPermissionTab.value === 'manage' ? 'permissionManage' : 'permissionRequest'
      case 'operateLog':
        return 'operateLog'
      case 'scheduledTask':
        return 'scheduledTask'
      default:
        return undefined
    }
  }

  const syncFunctionTabQuery = () => {
    const nextTab = getFunctionTabQueryValue()
    const currentTab = normalizePanelQuery(route.query._panel)

    if ((nextTab ?? null) === currentTab) {
      return
    }

    const nextQuery = { ...route.query }
    if (nextTab) {
      nextQuery._panel = nextTab
    } else {
      delete nextQuery._panel
    }

    router.replace({
      path: route.path,
      query: nextQuery
    })
  }

  const handleFunctionTabChange = (tabName: string) => {
    functionActiveTab.value = (tabName as FunctionTabName) || 'content'
    if (tabName === 'permission') {
      loadCurrentFunctionPermissionTab()
    } else if (tabName === 'operateLog') {
      nextTick(() => {
        formOperateLogSectionRef.value?.loadLogs({ page: 1 })
      })
    }
    syncFunctionTabQuery()
  }

  const handleFunctionPermissionTabChange = (tabName: string) => {
    functionPermissionTab.value = tabName === 'manage' ? 'manage' : 'request'
    if (functionActiveTab.value === 'permission') {
      loadCurrentFunctionPermissionTab()
      syncFunctionTabQuery()
    }
  }

  const handleApplyFormOperateLog = async (payload: {
    requestBody?: Record<string, any> | null
    responseBody?: Record<string, any> | null
    responseMetadata?: Record<string, any> | null
  }) => {
    functionActiveTab.value = 'content'
    syncFunctionTabQuery()
    await nextTick()

    if (!functionFormViewRef.value) {
      ElMessage.warning('当前表单尚未加载完成，请稍后重试')
      return
    }

    try {
      await functionFormViewRef.value.applyOperateLog(payload)
      ElMessage.success('已将执行记录回填到表单')
    } catch (error: any) {
      ElMessage.error(error?.message || '回填执行记录失败')
    }
  }

  const openFunctionOperateLog = async (filters?: {
    requestUser?: string
    traceId?: string
    keyword?: string
    status?: string
    source?: string
  }) => {
    if (!showFormOperateLogTab.value) {
      ElMessage.warning('当前函数暂不支持函数执行记录视图')
      return
    }

    functionActiveTab.value = 'operateLog'
    syncFunctionTabQuery()
    await nextTick()

    if (filters && formOperateLogSectionRef.value?.openWithFilters) {
      formOperateLogSectionRef.value.openWithFilters(filters)
      return
    }

    formOperateLogSectionRef.value?.loadLogs({ page: 1 })
  }

  function onScheduledTaskTotalChange(_total: number) {
  }

  const activateScheduledTaskTab = () => {
    functionActiveTab.value = 'scheduledTask'
    syncFunctionTabQuery()
  }

  const applyFunctionPanelQuery = (tab: LocationQueryValue | LocationQueryValue[] | undefined) => {
    const normalizedTab = normalizePanelQuery(tab)

    if (normalizedTab === 'permissionRequest' && showFunctionPermissionRequestTab.value) {
      functionActiveTab.value = 'permission'
      functionPermissionTab.value = 'request'
      loadCurrentFunctionPermissionTab()
      return
    }

    if (normalizedTab === 'permissionManage' && showFunctionPermissionRequestTab.value) {
      functionActiveTab.value = 'permission'
      functionPermissionTab.value = 'manage'
      loadCurrentFunctionPermissionTab()
      return
    }

    if (normalizedTab === 'permission' && showFunctionPermissionRequestTab.value) {
      functionActiveTab.value = 'permission'
      functionPermissionTab.value = 'request'
      loadCurrentFunctionPermissionTab()
      return
    }

    if (normalizedTab === 'operateLog' && showFormOperateLogTab.value) {
      functionActiveTab.value = 'operateLog'
      nextTick(() => {
        formOperateLogSectionRef.value?.loadLogs({ page: 1 })
      })
      return
    }

    if (normalizedTab === 'scheduledTask' && showScheduledTaskTab.value) {
      functionActiveTab.value = 'scheduledTask'
      return
    }

    if (normalizedTab) {
      if (functionActiveTab.value !== 'scheduledTask') {
        functionActiveTab.value = 'content'
      }
      return
    }

    if (
      (functionActiveTab.value === 'permission' && !showFunctionPermissionRequestTab.value) ||
      (functionActiveTab.value === 'operateLog' && !showFormOperateLogTab.value) ||
      (functionActiveTab.value === 'scheduledTask' && !showScheduledTaskTab.value)
    ) {
      functionActiveTab.value = 'content'
    }
  }

  watch(
    () => [currentFunction.value?.full_code_path, showFunctionPermissionRequestTab.value, showFormOperateLogTab.value, showScheduledTaskTab.value] as const,
    () => {
      if (!showFunctionPermissionRequestTab.value && functionActiveTab.value === 'permission') {
        functionActiveTab.value = 'content'
        syncFunctionTabQuery()
      }
      if (!showFormOperateLogTab.value && functionActiveTab.value === 'operateLog') {
        functionActiveTab.value = 'content'
        syncFunctionTabQuery()
      }
      if (!showScheduledTaskTab.value && functionActiveTab.value === 'scheduledTask') {
        functionActiveTab.value = 'content'
        syncFunctionTabQuery()
      }
    },
    { immediate: true }
  )

  watch(
    () => [route.query._panel, showFunctionPermissionRequestTab.value, showFormOperateLogTab.value, showScheduledTaskTab.value] as const,
    ([tab]) => {
      applyFunctionPanelQuery(tab)
    },
    { immediate: true }
  )

  return {
    functionActiveTab,
    functionPermissionTab,
    functionFormViewRef,
    functionPermissionRequestListRef,
    functionPermissionManageListRef,
    formOperateLogSectionRef,
    showScheduledTaskTab,
    showFunctionPermissionRequestTab,
    showFormOperateLogTab,
    showFunctionTabsWrapper,
    handleFunctionTabChange,
    handleFunctionPermissionTabChange,
    handleApplyFormOperateLog,
    openFunctionOperateLog,
    onScheduledTaskTotalChange,
    syncFunctionTabQuery,
    activateScheduledTaskTab
  }
}
