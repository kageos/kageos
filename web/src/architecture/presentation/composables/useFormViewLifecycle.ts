import { onMounted, onUnmounted, watch, type Ref } from 'vue'
import type { FormApplicationService } from '../../application/services/FormApplicationService'
import type { FormDomainService } from '../../domain/services/FormDomainService'
import type { FieldConfig, FunctionDetail } from '../../domain/types'
import type { WorkspaceDomainService } from '../../domain/services/WorkspaceDomainService'
import { FormEvent, WorkspaceEvent, type IEventBus } from '../../infrastructure/eventBus'
import type { FormStateManager } from '../../infrastructure/stateManager/FormStateManager'
import type { WorkspaceStateManager } from '../../infrastructure/stateManager/WorkspaceStateManager'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { Logger } from '@/core/utils/logger'
import type { FormDataStore } from '@/core/stores-v2/formData'
import { getFormRequestFields } from '@/utils/functionSchemaSelectors'
import {
  buildInitialDataFromFormDataStore as buildInitialDataFromFormDataStoreHelper,
  syncFormDataStoreToStateManager as syncFormDataStoreToStateManagerHelper
} from '../views/utils/formViewRuntime'

interface ApplyOperateLogPayload {
  requestBody?: Record<string, any> | null
  responseBody?: Record<string, any> | null
  responseMetadata?: Record<string, any> | null
}

interface UseFormViewLifecycleOptions {
  eventBus: IEventBus
  functionDetail: Ref<FunctionDetail | null>
  propsFunctionDetail: () => FunctionDetail | undefined
  propsInitialData: () => Record<string, any>
  formDataStore: Pick<FormDataStore, 'getValue' | 'clear'>
  responseDataStore: { clear: () => void }
  stateManager: FormStateManager
  domainService: FormDomainService
  applicationService: FormApplicationService
  workspaceStateManager: WorkspaceStateManager
  workspaceDomainService: WorkspaceDomainService
  permissionErrorStore: { clearError: () => void }
  initializeParams: () => Promise<Record<string, any> | undefined>
  hydrateCurrentWidgetDisplays?: (initSource?: 'url' | 'default' | 'initialData') => Promise<void>
  watchFormData: () => void
}

export function useFormViewLifecycle(options: UseFormViewLifecycleOptions) {
  let unsubscribeFunctionLoaded: (() => void) | null = null
  let unsubscribeFormInitialized: (() => void) | null = null
  let latestFormInitializationToken = 0
  let inFlightFormInitializationKey: string | null = null
  let lastAppliedFormInitializationKey: string | null = null

  function syncFormDataStoreToStateManager(fields: FieldConfig[]): void {
    syncFormDataStoreToStateManagerHelper({
      fields,
      formDataStore: options.formDataStore,
      stateManager: options.stateManager
    })
  }

  function buildInitialDataFromFormDataStore(fields: FieldConfig[]): Record<string, any> {
    return buildInitialDataFromFormDataStoreHelper({
      fields,
      formDataStore: options.formDataStore
    })
  }

  function hasNonEmptyInitialData(data?: Record<string, any> | null): boolean {
    return !!data && Object.keys(data).length > 0
  }

  function buildFormInitializationKey(
    detail: FunctionDetail,
    initialData?: Record<string, any> | null
  ): string {
    return JSON.stringify({
      id: detail.id ?? null,
      router: detail.router ?? '',
      mode: hasNonEmptyInitialData(initialData) ? 'update' : 'create',
      initialData: initialData || null
    })
  }

  function restoreResponseParams(metadata?: Record<string, any> | null): void {
    const responseParams = metadata?.responseParams
    if (!responseParams || typeof (options.stateManager as any).setResponse !== 'function') {
      return
    }

    ;(options.stateManager as any).setResponse(responseParams)
    Logger.debug('FormView', '已恢复响应数据', {
      responseParamsKeys: Object.keys(responseParams),
      stateResponse: options.stateManager.getState().response
    })
  }

  function resetFormRuntimeState(): void {
    options.applicationService.clearForm()
    options.responseDataStore.clear()
  }

  async function initializeFormForDetail(
    detail: FunctionDetail,
    config: {
      initialData?: Record<string, any>
      resetRuntime?: boolean
      force?: boolean
    } = {}
  ): Promise<void> {
    const fields = getFormRequestFields(detail) as FieldConfig[]
    if (fields.length === 0) {
      return
    }

    const explicitInitialData = config.initialData ?? options.propsInitialData()
    const initializationKey = buildFormInitializationKey(detail, explicitInitialData)

    if (!config.force) {
      if (initializationKey === inFlightFormInitializationKey || initializationKey === lastAppliedFormInitializationKey) {
        return
      }
    }

    const token = ++latestFormInitializationToken
    inFlightFormInitializationKey = initializationKey

    if (config.resetRuntime !== false) {
      resetFormRuntimeState()
    }

    try {
      if (hasNonEmptyInitialData(explicitInitialData)) {
        options.applicationService.initializeForm(fields, explicitInitialData, true)
        await options.hydrateCurrentWidgetDisplays?.('initialData')
        Logger.debug('FormView', '已完成 initialData 组件展示态 hydrate', {
          fieldCount: fields.length,
          initialDataKeys: Object.keys(explicitInitialData || {})
        })
      } else {
        const metadata = await options.initializeParams()
        if (token !== latestFormInitializationToken) {
          return
        }

        syncFormDataStoreToStateManager(fields)
        const initialData = buildInitialDataFromFormDataStore(fields)
        options.applicationService.initializeForm(fields, initialData, false)

        if (token !== latestFormInitializationToken) {
          return
        }

        restoreResponseParams(metadata)
      }

      if (token !== latestFormInitializationToken) {
        return
      }

      lastAppliedFormInitializationKey = initializationKey
    } finally {
      if (token === latestFormInitializationToken) {
        inFlightFormInitializationKey = null
      }
    }
  }

  async function applyOperateLog(payload: ApplyOperateLogPayload): Promise<void> {
    if (!options.functionDetail.value) {
      throw new Error('函数详情未加载完成')
    }

    const requestBody =
      payload.requestBody && typeof payload.requestBody === 'object' && !Array.isArray(payload.requestBody)
        ? payload.requestBody
        : {}

    await initializeFormForDetail(options.functionDetail.value, {
      initialData: requestBody,
      force: true
    })

    if (typeof (options.stateManager as any).setResponse === 'function') {
      ;(options.stateManager as any).setResponse(payload.responseBody || null)
    }
    if (typeof (options.stateManager as any).setMetadata === 'function') {
      ;(options.stateManager as any).setMetadata(payload.responseMetadata || null)
    }

    Logger.info('[FormView]', '已回填执行记录到表单', {
      router: options.functionDetail.value.router,
      requestKeys: Object.keys(requestBody),
      hasResponseBody: !!payload.responseBody,
      metadataKeys: payload.responseMetadata ? Object.keys(payload.responseMetadata) : []
    })
  }

  const hasInitialDataChanged = (
    newData: Record<string, any>,
    oldData?: Record<string, any>
  ): boolean => {
    const newKeys = Object.keys(newData || {})
    const oldKeys = Object.keys(oldData || {})
    return newKeys.length !== oldKeys.length || newKeys.some((key) => newData[key] !== oldData?.[key])
  }

  onMounted(async () => {
    resetFormRuntimeState()
    options.permissionErrorStore.clearError()

    const propFunctionDetail = options.propsFunctionDetail()
    if (propFunctionDetail && propFunctionDetail.id !== undefined && propFunctionDetail.id !== null) {
      options.functionDetail.value = propFunctionDetail
      Logger.debug('FormView', 'onMounted 时使用 prop 提供的 functionDetail', {
        functionId: propFunctionDetail.id,
        requestFieldsCount: getFormRequestFields(propFunctionDetail).length
      })
    } else {
      const currentFunction = options.workspaceStateManager.getCurrentFunction()
      if (currentFunction && currentFunction.type === 'function') {
        Logger.debug('FormView', 'onMounted 时主动加载 functionDetail', {
          functionNodeId: currentFunction.id,
          refId: currentFunction.ref_id,
          functionPath: currentFunction.full_code_path,
          hasRefId: !!(currentFunction.ref_id && currentFunction.ref_id > 0)
        })

        try {
          const detail = await options.workspaceDomainService.loadFunction(currentFunction)
          options.functionDetail.value = detail
          Logger.info('FormView', 'onMounted 时成功加载 functionDetail', {
            functionId: detail.id,
            refId: currentFunction.ref_id,
            requestFieldsCount: getFormRequestFields(detail).length
          })
        } catch (error) {
          Logger.error('FormView', 'onMounted 时加载 functionDetail 失败', error)
          return
        }
      } else {
        Logger.debug('FormView', 'onMounted 时没有当前函数节点，等待 watch 触发', {
          hasCurrentFunction: !!currentFunction,
          functionType: currentFunction?.type
        })
        return
      }
    }

    if (
      options.functionDetail.value &&
      options.functionDetail.value.id !== undefined &&
      options.functionDetail.value.id !== null &&
      getFormRequestFields(options.functionDetail.value).length > 0
    ) {
      await initializeFormForDetail(options.functionDetail.value, {
        resetRuntime: false
      })
    }

    let lastInitializedFunctionId: number | null = null
    unsubscribeFunctionLoaded = options.eventBus.on(WorkspaceEvent.functionLoaded, async (payload: { detail: FunctionDetail }) => {
      const detailId = payload.detail.id
      if (payload.detail.template_type === TEMPLATE_TYPE.FORM && options.functionDetail.value && detailId != null && detailId === options.functionDetail.value.id) {
        if (lastInitializedFunctionId === detailId) {
          Logger.debug('FormView', '跳过重复的 functionLoaded 事件', { functionId: detailId })
          return
        }
        lastInitializedFunctionId = detailId
        options.functionDetail.value = payload.detail
        await initializeFormForDetail(payload.detail, {
          force: true
        })
      }
    })

    unsubscribeFormInitialized = options.eventBus.on(FormEvent.initialized, () => {})
    options.watchFormData()
  })

  watch(
    () => options.propsInitialData(),
    async (newInitialData: Record<string, any>, oldInitialData?: Record<string, any>) => {
      if (!options.functionDetail.value || getFormRequestFields(options.functionDetail.value).length === 0) {
        return
      }
      if (oldInitialData === undefined) {
        return
      }
      if (!hasInitialDataChanged(newInitialData, oldInitialData)) {
        return
      }

      await initializeFormForDetail(options.functionDetail.value, {
        initialData: newInitialData,
        force: true
      })
    },
    { deep: true }
  )

  watch(
    () => [options.propsFunctionDetail()?.id, options.propsFunctionDetail()?.router, options.propsFunctionDetail()?.schema],
    async ([newId, newRouter, newSchema], [oldId, oldRouter, oldSchema]) => {
      options.permissionErrorStore.clearError()

      const propFunctionDetail = options.propsFunctionDetail()
      if (propFunctionDetail && propFunctionDetail.id !== undefined && propFunctionDetail.id !== null) {
        options.functionDetail.value = propFunctionDetail
      }

      if (
        !options.functionDetail.value ||
        getFormRequestFields(options.functionDetail.value).length === 0 ||
        options.functionDetail.value.id === undefined ||
        options.functionDetail.value.id === null
      ) {
        return
      }
      if (oldId === undefined || oldId === null) {
        return
      }

      const functionDetailChanged =
        newId !== oldId ||
        newRouter !== oldRouter ||
        JSON.stringify(newSchema || null) !== JSON.stringify(oldSchema || null)

      if (!functionDetailChanged) {
        return
      }

      await initializeFormForDetail(options.functionDetail.value, {
        force: true
      })
    },
    { deep: true, immediate: false }
  )

  onUnmounted(() => {
    unsubscribeFunctionLoaded?.()
    unsubscribeFormInitialized?.()
    options.applicationService.dispose()
    options.formDataStore.clear()
    options.responseDataStore.clear()
  })

  return {
    resetFormRuntimeState,
    applyOperateLog,
  }
}
