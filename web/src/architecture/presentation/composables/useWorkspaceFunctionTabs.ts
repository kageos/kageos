import { computed, ref, watch, type ComputedRef } from 'vue'
import type { LocationQueryValue, RouteLocationNormalizedLoaded, Router } from 'vue-router'
import type { FunctionDetail } from '@/architecture/domain/types'
import { Logger } from '@/architecture/shared/logger'
import type { ServiceTree } from '../../domain/types'
import { featureFlags } from '@/architecture/shared/config/features'
import {
  clearOperateLogRouteQuery,
  clearScheduledRouteQuery,
} from '@/architecture/shared/routing/platformRouteParams'

type FunctionTabName = 'content' | 'permission' | 'publicShare' | 'operateLog' | 'scheduledTask'

type FunctionFormViewRef = Record<string, unknown>

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
  const { route, router, currentFunction } = options

  const functionActiveTab = ref<FunctionTabName>('content')
  const functionFormViewRef = ref<FunctionFormViewRef | null>(null)

  const setFunctionFormViewRef = (instance: FunctionFormViewRef | null) => {
    functionFormViewRef.value = instance
    Logger.debug('WorkspaceFunctionTabs', '更新 FormView 引用', {
      ready: !!instance
    })
  }

  const showFunctionTabsWrapper = computed(() => {
    return currentFunction.value?.type === 'function'
  })

  const getFunctionTabQueryValue = () => {
    if (functionActiveTab.value === 'permission') return 'permission'
    if (functionActiveTab.value === 'publicShare') return 'publicShare'
    if (functionActiveTab.value === 'operateLog') return 'operateLog'
    if (functionActiveTab.value === 'scheduledTask') return 'scheduledTask'
    return undefined
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
    if (nextTab !== 'scheduledTask') {
      clearScheduledRouteQuery(nextQuery)
    }
    if (nextTab !== 'operateLog') {
      clearOperateLogRouteQuery(nextQuery)
    }

    router.replace({
      path: route.path,
      query: nextQuery
    })
  }

  const handleFunctionTabChange = (tabName: string) => {
    if (tabName === 'permission') functionActiveTab.value = 'permission'
    else if (tabName === 'publicShare') functionActiveTab.value = 'publicShare'
    else if (tabName === 'operateLog' && featureFlags.operateLogs) functionActiveTab.value = 'operateLog'
    else if (tabName === 'scheduledTask' && featureFlags.scheduledTasks) functionActiveTab.value = 'scheduledTask'
    else functionActiveTab.value = 'content'
    syncFunctionTabQuery()
  }

  const applyFunctionPanelQuery = (tab: LocationQueryValue | LocationQueryValue[] | undefined) => {
    const normalizedTab = normalizePanelQuery(tab)

    if (normalizedTab === 'permission' && currentFunction.value?.type === 'function') {
      functionActiveTab.value = 'permission'
      return
    }

    if (normalizedTab === 'publicShare' && currentFunction.value?.type === 'function') {
      functionActiveTab.value = 'publicShare'
      return
    }

    if (normalizedTab === 'operateLog' && featureFlags.operateLogs && currentFunction.value?.type === 'function') {
      functionActiveTab.value = 'operateLog'
      return
    }

    if (normalizedTab === 'scheduledTask' && featureFlags.scheduledTasks && currentFunction.value?.type === 'function') {
      functionActiveTab.value = 'scheduledTask'
      return
    }

    if (normalizedTab) {
      functionActiveTab.value = 'content'
      return
    }

    if (functionActiveTab.value !== 'content' && currentFunction.value?.type !== 'function') {
      functionActiveTab.value = 'content'
    }
  }

  watch(
    () => currentFunction.value?.full_code_path,
    () => {
      if (currentFunction.value?.type !== 'function') {
        functionActiveTab.value = 'content'
        syncFunctionTabQuery()
      }
    },
    { immediate: true }
  )

  watch(
    () => route.query._panel,
    (tab) => {
      applyFunctionPanelQuery(tab)
    },
    { immediate: true }
  )

  return {
    functionActiveTab,
    functionFormViewRef,
    setFunctionFormViewRef,
    showFunctionTabsWrapper,
    handleFunctionTabChange,
    syncFunctionTabQuery
  }
}
