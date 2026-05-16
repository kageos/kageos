import { computed, nextTick, ref, watch, type ComputedRef } from 'vue'
import type { RouteLocationNormalizedLoaded, Router, LocationQueryValue } from 'vue-router'
import { ElMessage } from 'element-plus'
import { TEMPLATE_TYPE } from '@/architecture/domain/constants/functionTypes'
import type { FunctionDetail } from '@/architecture/domain/types'
import { Logger } from '@/architecture/shared/logger'
import type { ServiceTree } from '../../domain/types'
import { featureFlags } from '@/architecture/shared/config/features'

type FunctionTabName = 'content' | 'detail' | 'operateLog'
type ReplayContext = {
  source: 'operate_log'
  title?: string
  taskId?: number
  executionId?: number
  traceId?: string
  executedAt?: string
}
type FormOperateLogApplyPayload = {
  requestBody?: Record<string, any> | null
  responseBody?: Record<string, any> | null
  responseMetadata?: Record<string, any> | null
  replayContext?: ReplayContext | null
}

type FunctionFormViewRef = {
  applyOperateLog: (payload: FormOperateLogApplyPayload) => Promise<void>
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

function cloneOperateLogObject<T extends Record<string, any> | null | undefined>(value: T): T {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return value
  }

  try {
    return JSON.parse(JSON.stringify(value)) as T
  } catch {
    return { ...value } as T
  }
}

function cloneOperateLogPayload(payload: FormOperateLogApplyPayload): FormOperateLogApplyPayload {
  return {
    requestBody: cloneOperateLogObject(payload.requestBody),
    responseBody: cloneOperateLogObject(payload.responseBody),
    responseMetadata: cloneOperateLogObject(payload.responseMetadata),
    replayContext: payload.replayContext ? { ...payload.replayContext } : payload.replayContext
  }
}

function normalizePanelQuery(tab: LocationQueryValue | LocationQueryValue[] | undefined): string | null {
  if (Array.isArray(tab)) {
    return tab[0] ?? null
  }

  return typeof tab === 'string' ? tab : null
}

function normalizeQueryString(value: LocationQueryValue | LocationQueryValue[] | undefined): string {
  const normalized = normalizePanelQuery(value)
  return normalized ? normalized.trim() : ''
}

export function useWorkspaceFunctionTabs(options: UseWorkspaceFunctionTabsOptions) {
  const { route, router, currentFunction, currentFunctionDetail } = options

  const functionActiveTab = ref<FunctionTabName>('content')
  const functionFormViewRef = ref<FunctionFormViewRef | null>(null)
  const formOperateLogSectionRef = ref<FormOperateLogSectionRef | null>(null)

  const setFunctionFormViewRef = (instance: FunctionFormViewRef | null) => {
    functionFormViewRef.value = instance
    Logger.debug('WorkspaceFunctionTabs', '更新 FormView 引用', {
      ready: !!instance
    })
  }

  const setFormOperateLogSectionRef = (instance: FormOperateLogSectionRef | null) => {
    formOperateLogSectionRef.value = instance
  }

  const waitForFormViewRef = async (maxAttempts = 8): Promise<FunctionFormViewRef | null> => {
    for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
      if (functionFormViewRef.value) {
        return functionFormViewRef.value
      }
      await nextTick()
    }

    return functionFormViewRef.value
  }

  const showFormOperateLogTab = computed(() => {
    return featureFlags.operateLogs && currentFunction.value?.type === 'function' && currentFunctionDetail.value?.template_type === TEMPLATE_TYPE.FORM
  })

  const showFunctionTabsWrapper = computed(() => {
    return currentFunction.value?.type === 'function'
  })

  const getFunctionTabQueryValue = () => {
    switch (functionActiveTab.value) {
      case 'detail':
        return 'detail'
      case 'operateLog':
        return 'operateLog'
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
    if (tabName === 'permissionRequest' || tabName === 'permissionManage' || tabName === 'permission') {
      functionActiveTab.value = 'content'
      syncFunctionTabQuery()
      return
    }

    functionActiveTab.value = (tabName as FunctionTabName) || 'content'
    if (tabName === 'operateLog') {
      nextTick(() => {
        formOperateLogSectionRef.value?.loadLogs({ page: 1 })
      })
    }
    syncFunctionTabQuery()
  }

  const handleApplyFormOperateLog = async (payload: FormOperateLogApplyPayload) => {
    functionActiveTab.value = 'content'
    syncFunctionTabQuery()

    Logger.debug('WorkspaceFunctionTabs', '收到执行记录回填请求', {
      requestKeys: Object.keys(payload.requestBody || {}),
      hasResponseBody: !!payload.responseBody,
      hasResponseMetadata: !!payload.responseMetadata,
      refReady: !!functionFormViewRef.value
    })

    try {
      let applied = false
      let lastAppliedRef: FunctionFormViewRef | null = null

      for (let attempt = 0; attempt < 3; attempt += 1) {
        const targetRef = attempt === 0
          ? functionFormViewRef.value || await waitForFormViewRef()
          : functionFormViewRef.value

        if (!targetRef || targetRef === lastAppliedRef) {
          break
        }

        await targetRef.applyOperateLog(cloneOperateLogPayload(payload))
        applied = true
        lastAppliedRef = targetRef
        await nextTick()
      }

      if (!applied) {
        Logger.warn('WorkspaceFunctionTabs', '执行记录回填失败：FormView 引用不可用', {
          requestKeys: Object.keys(payload.requestBody || {})
        })
        ElMessage.warning('当前表单尚未加载完成，请稍后重试')
        return
      }

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

  const applyFunctionPanelQuery = (tab: LocationQueryValue | LocationQueryValue[] | undefined) => {
    const normalizedTab = normalizePanelQuery(tab)

    if (normalizedTab === 'detail' && currentFunction.value?.type === 'function') {
      functionActiveTab.value = 'detail'
      return
    }

    if (normalizedTab === 'operateLog' && showFormOperateLogTab.value) {
      functionActiveTab.value = 'operateLog'
      nextTick(() => {
        const traceId = normalizeQueryString(route.query._trace_id)
        if (traceId && formOperateLogSectionRef.value?.openWithFilters) {
          formOperateLogSectionRef.value.openWithFilters({ traceId })
          return
        }
        formOperateLogSectionRef.value?.loadLogs({ page: 1 })
      })
      return
    }

    if (normalizedTab) {
      functionActiveTab.value = 'content'
      return
    }

    if (
      functionActiveTab.value === 'operateLog' && !showFormOperateLogTab.value
    ) {
      functionActiveTab.value = 'content'
    }
  }

  watch(
    () => [currentFunction.value?.full_code_path, showFormOperateLogTab.value] as const,
    () => {
      if (!showFormOperateLogTab.value && functionActiveTab.value === 'operateLog') {
        functionActiveTab.value = 'content'
        syncFunctionTabQuery()
      }
    },
    { immediate: true }
  )

  watch(
    () => [route.query._panel, route.query._trace_id, showFormOperateLogTab.value] as const,
    ([tab]) => {
      applyFunctionPanelQuery(tab)
    },
    { immediate: true }
  )

  return {
    functionActiveTab,
    functionFormViewRef,
    formOperateLogSectionRef,
    setFunctionFormViewRef,
    setFormOperateLogSectionRef,
    showFormOperateLogTab,
    showFunctionTabsWrapper,
    handleFunctionTabChange,
    handleApplyFormOperateLog,
    openFunctionOperateLog,
    syncFunctionTabQuery
  }
}
