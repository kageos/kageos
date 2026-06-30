import { ref, watch, type ComputedRef } from 'vue'
import type { LocationQueryRaw, LocationQueryValue, RouteLocationNormalizedLoaded, Router } from 'vue-router'
import type { ServiceTree } from '@/architecture/domain/types'
import { featureFlags } from '@/architecture/shared/config/features'
import {
  clearOperateLogRouteQuery,
  clearScheduledRouteQuery,
  isOperateLogPanelQuery,
  isScheduledPanelQuery,
  PLATFORM_PANEL_QUERY_KEY,
  PLATFORM_SCHEDULED_TASK_ID_QUERY_KEY,
  readStringQuery,
} from '@/architecture/shared/routing/platformRouteParams'

export type PackageTabName = 'detail' | 'permission' | 'notification' | 'operateLog' | 'scheduledAgentTask'

export interface UsePackageDetailTabsOptions {
  route: RouteLocationNormalizedLoaded
  router: Router
  currentPackageNode: ComputedRef<ServiceTree | null>
}

function normalizePanelQuery(tab: LocationQueryValue | LocationQueryValue[] | undefined): string | null {
  if (Array.isArray(tab)) {
    return tab[0] ?? null
  }

  return typeof tab === 'string' ? tab : null
}

function getDefaultPackageTab(): PackageTabName {
  return 'detail'
}

function normalizePackageTab(tabName: string | number): PackageTabName {
  if (tabName === 'permission') return 'permission'
  if (tabName === 'notification') return 'notification'
  if (tabName === 'detail') return 'detail'
  if (tabName === 'operateLog' && featureFlags.operateLogs) return 'operateLog'
  if (tabName === 'scheduledAgentTask' && featureFlags.scheduledTasks) return 'scheduledAgentTask'
  return 'detail'
}

function queryValueEquals(left: unknown, right: unknown): boolean {
  if (Array.isArray(left) || Array.isArray(right)) {
    return JSON.stringify(left ?? []) === JSON.stringify(right ?? [])
  }

  return left === right
}

function queryEquals(left: Record<string, unknown>, right: Record<string, unknown>): boolean {
  const leftKeys = Object.keys(left)
  const rightKeys = Object.keys(right)
  if (leftKeys.length !== rightKeys.length) return false

  return leftKeys.every((key) => queryValueEquals(left[key], right[key]))
}

export function usePackageDetailTabs(options: UsePackageDetailTabsOptions) {
  const { route, router, currentPackageNode } = options
  const activeTab = ref<PackageTabName>(getDefaultPackageTab())

  const resolveTabFromRoute = (): PackageTabName => {
    const panel = normalizePanelQuery(route.query[PLATFORM_PANEL_QUERY_KEY])

    if (panel === 'permission') return 'permission'
    if (panel === 'notification') return 'notification'
    if (panel === 'detail') return 'detail'
    if (panel === 'operateLog' && featureFlags.operateLogs) return 'operateLog'
    if (panel === 'scheduledAgentTask' && featureFlags.scheduledTasks) return 'scheduledAgentTask'

    if (
      isScheduledPanelQuery(route.query, 'agent')
      && readStringQuery(route.query, PLATFORM_SCHEDULED_TASK_ID_QUERY_KEY)
      && featureFlags.scheduledTasks
    ) {
      return 'scheduledAgentTask'
    }

    if (isOperateLogPanelQuery(route.query) && featureFlags.operateLogs) {
      return 'operateLog'
    }

    return getDefaultPackageTab()
  }

  const applyRouteQuery = () => {
    if (currentPackageNode.value?.type !== 'package') return

    activeTab.value = resolveTabFromRoute()
  }

  const syncPackageTabQuery = () => {
    if (currentPackageNode.value?.type !== 'package') return

    const nextQuery: Record<string, unknown> = {
      ...route.query,
      [PLATFORM_PANEL_QUERY_KEY]: activeTab.value,
    }

    if (activeTab.value !== 'scheduledAgentTask') {
      clearScheduledRouteQuery(nextQuery)
    }
    if (activeTab.value !== 'operateLog') {
      clearOperateLogRouteQuery(nextQuery)
    }

    if (queryEquals(route.query as Record<string, unknown>, nextQuery)) {
      return
    }

    router.replace({
      path: route.path,
      query: nextQuery as LocationQueryRaw,
    })
  }

  const handlePackageTabChange = (tabName: string | number) => {
    activeTab.value = normalizePackageTab(tabName)
    syncPackageTabQuery()
  }

  watch(
    () => [
      route.query[PLATFORM_PANEL_QUERY_KEY],
      route.query._open,
      route.query._scheduled,
      route.query._scheduled_kind,
      route.query[PLATFORM_SCHEDULED_TASK_ID_QUERY_KEY],
      currentPackageNode.value?.full_code_path,
    ],
    applyRouteQuery,
    { immediate: true }
  )

  return {
    activeTab,
    handlePackageTabChange,
    syncPackageTabQuery,
  }
}
