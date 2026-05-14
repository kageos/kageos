/**
 * useWorkspaceDetail - 详情管理 Composable
 * 
 * ============================================
 * 📋 需求说明
 * ============================================
 * 
 * 1. **详情抽屉管理**：
 *    - 从表格行点击打开详情抽屉
 *    - 支持查看模式和编辑模式切换
 *    - 支持上一条/下一条导航
 * 
 * 2. **编辑功能**：
 *    - 编辑模式下显示可编辑字段（根据 `hide.scenes` 过滤）
 *    - 提交成功后刷新表格数据
 * 
 * 3. **URL 同步**：
 *    - 查看模式：`_tab=detail&_id=xxx`
 *    - 编辑模式：不设置 `_tab` 参数（只使用 `_id`）
 *    - 关闭抽屉时清除 URL 参数
 *    - 注意：只有新增模式（`_tab=OnTableAddRow`）才同步表单字段参数到 URL
 * 
 * ============================================
 * 🎯 设计思路
 * ============================================
 * 
 * 1. **模式切换**：
 *    - `read` 模式：只读展示，使用 `_tab=detail`
 *    - `edit` 模式：可编辑，不设置 `_tab` 参数（只使用 `_id`）
 *    - 切换模式时更新 URL，清除表单字段参数
 * 
 * 2. **数据流**：
 *    - 从表格行数据构建编辑表单的初始数据
 *    - 编辑模式下只显示 update 场景可编辑字段
 *    - 提交时提取表单数据并调用 TableApplicationService.updateRow
 * 
 * ============================================
 * 📝 关键功能
 * ============================================
 * 
 * 1. **openDetailDrawer**：
 *    - 从表格行数据打开详情抽屉
 *    - 设置 URL 为 `_tab=detail&_id=xxx`
 *    - 清除所有表单字段参数
 * 
 * 2. **toggleDrawerMode**：
 *    - 切换查看/编辑模式
 *    - 查看模式：设置 `_tab=detail`
 *    - 编辑模式：不设置 `_tab` 参数（只使用 `_id`）
 *    - 清除表单字段参数（编辑模式下）
 * 
 * 3. **submitDrawerEdit**：
 *    - 提取表单数据并提交
 *    - 成功后刷新表格数据，清除 URL 参数
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **URL 参数管理**：
 *    - 详情抽屉相关参数：`_tab`、`_id`
 *    - 编辑模式下必须清除表单字段参数
 *    - 关闭抽屉时清除所有相关参数
 * 
 * 3. **字段过滤**：
 *    - 编辑模式下隐藏 `hide.scenes` 包含 update 的字段
 *    - 通过 `editFunctionDetail` computed 过滤字段
 */

import { ref, computed } from 'vue'
import { deepClone } from '@/utils/clone'
import { useRoute, useRouter } from 'vue-router'
import { ElNotification, ElMessage } from 'element-plus'
import { serviceFactory } from '../../infrastructure/factories'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import { eventBus, RouteEvent, TableEvent, WorkspaceEvent } from '../../infrastructure/eventBus'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import FormView from '@/architecture/presentation/views/FormView.vue'
import type { FieldConfig, FieldValue, FunctionDetail } from '../../domain/types'
import type { TableResponse } from '../../domain/services/TableDomainService'
import { useUserInfoStore } from '@/architecture/infrastructure/stores/userInfo'
import { createAutoFieldValue, createEmptyRawFieldValue } from '@/architecture/runtime/utils/createFieldValue'
import {
  buildDetailLookupSearchRequest,
  buildDetailBaseQuery as buildDetailBaseQueryHelper,
  buildEditFunctionDetail,
  findDetailIdField,
  findDetailRowMatch,
  getEditableFieldCodes as getEditableFieldCodesHelper,
  resolveDetailRouteRequest,
  shouldWaitForDetailTableData,
  type DetailRestoreTrigger
} from './utils/workspaceDetailRuntime'
import { getFunctionCallbacks, getTableDetailFields, getTableRequestFields } from '@/utils/functionSchemaSelectors'

export function useWorkspaceDetail(
  options: {
    currentFunctionDetail: () => FunctionDetail | null
    currentFunction: () => any
  },
  serviceProvider: IServiceProvider = serviceFactory  // 🔥 通过参数注入，提高可测试性
) {
  const route = useRoute()
  const router = useRouter()
  const apiClient = serviceProvider.getApiClient()
  const tableApplicationService = serviceProvider.getTableApplicationService()
  const tableStateManager = serviceProvider.getTableStateManager()
  const userInfoStore = useUserInfoStore()

  // 详情抽屉状态
  const detailDrawerVisible = ref(false)
  const detailDrawerTitle = ref('详情')
  const detailRowData = ref<Record<string, any> | null>(null)
  const detailFields = ref<FieldConfig[]>([])
  const detailOriginalRow = ref<Record<string, any> | null>(null)
  const detailDrawerMode = ref<'read' | 'edit'>('read')
  const drawerSubmitting = ref(false)
  const detailFormViewRef = ref<InstanceType<typeof FormView> | null>(null)
  const detailUserInfoMap = ref<Map<string, any>>(new Map())
  const detailTableData = ref<any[]>([])
  const currentDetailIndex = ref<number>(-1)

  // 编辑模式的函数详情（从 response 字段中筛选可编辑的字段）
  const editFunctionDetail = computed<FunctionDetail | null>(() => {
    return buildEditFunctionDetail(options.currentFunctionDetail())
  })

  const supportsDetailEdit = computed(() => {
    return getFunctionCallbacks(options.currentFunctionDetail()).includes('OnTableUpdateRow')
  })

  const getEditableFieldCodes = (): string[] => {
    return getEditableFieldCodesHelper(editFunctionDetail.value)
  }

  const buildDetailBaseQuery = (): Record<string, string | string[]> => {
    const requestFieldCodes = getTableRequestFields(options.currentFunctionDetail())
      .map(field => field.code)

    return buildDetailBaseQueryHelper({
      query: route.query as Record<string, any>,
      editableFieldCodes: getEditableFieldCodes(),
      preserveRawFieldCodes: requestFieldCodes
    })
  }

  let pendingDetailRestoreKey: string | null = null
  let pendingDetailRestoreQuery: Record<string, any> | null = null
  let latestDetailRestoreToken = 0
  let inFlightDetailLookupKey: string | null = null

  const setPendingDetailRestore = (query: Record<string, any>) => {
    const request = resolveDetailRouteRequest(query)
    const nextKey = request?.key ?? null

    if (nextKey !== pendingDetailRestoreKey) {
      latestDetailRestoreToken += 1
    }

    pendingDetailRestoreKey = nextKey
    pendingDetailRestoreQuery = request ? { ...query } : null

    return request
  }

  const clearPendingDetailRestore = (): void => {
    pendingDetailRestoreKey = null
    pendingDetailRestoreQuery = null
  }

  const closeDetailDrawerForRoute = (query: Record<string, any>): void => {
    const hasDetailRouteState = query._tab === 'detail' || query._id !== undefined
    if (!hasDetailRouteState) {
      detailDrawerVisible.value = false
      detailDrawerMode.value = 'read'
    }
  }

  const resolveDetailIndex = (
    row: Record<string, any>,
    fields: FieldConfig[],
    tableData: Record<string, any>[],
    explicitIndex?: number
  ): number => {
    if (typeof explicitIndex === 'number' && explicitIndex >= 0) {
      return explicitIndex
    }

    const matchedById = findDetailRowMatch(tableData, String(row.id ?? row._id ?? ''))
    if (matchedById) {
      return matchedById.index
    }

    const idField = fields.find(field => field.code === 'id' || field.widget?.type === 'number')
    if (idField && row[idField.code] !== undefined && row[idField.code] !== null) {
      return tableData.findIndex((candidate: Record<string, any>) => candidate[idField.code] === row[idField.code])
    }

    return tableData.findIndex((candidate: Record<string, any>) => JSON.stringify(candidate) === JSON.stringify(row))
  }

  const preloadDetailUserInfo = async (fields: FieldConfig[], row: Record<string, any>): Promise<void> => {
    const userFields = fields.filter(field => field.widget?.type === 'user')
    if (userFields.length === 0) {
      detailUserInfoMap.value = new Map()
      return
    }

    const usernames: string[] = []
    userFields.forEach(field => {
      const value = row[field.code]
      if (!value) {
        return
      }

      if (Array.isArray(value)) {
        usernames.push(...value.map(item => String(item)))
        return
      }

      usernames.push(String(value))
    })

    if (usernames.length === 0) {
      detailUserInfoMap.value = new Map()
      return
    }

    try {
      const users = await userInfoStore.batchGetUserInfo([...new Set(usernames)])
      const map = new Map<string, any>()
      users.forEach(user => {
        map.set(user.username, user)
      })
      detailUserInfoMap.value = map
    } catch (error) {
      detailUserInfoMap.value = new Map()
    }
  }

  const applyDetailDrawerState = (options: {
    detail: FunctionDetail
    row: Record<string, any>
    tableData: Record<string, any>[]
    mode?: 'read' | 'edit'
    index?: number
  }): void => {
    const { detail, row, tableData, mode = 'read', index } = options
    const fields = getTableDetailFields(detail) as FieldConfig[]

    clearPendingDetailRestore()
    detailRowData.value = row
    detailOriginalRow.value = deepClone(row)
    detailDrawerTitle.value = detail.name || '详情'
    detailFields.value = fields
    detailTableData.value = tableData
    currentDetailIndex.value = resolveDetailIndex(row, fields, tableData, index)
    detailDrawerMode.value = mode
    detailDrawerVisible.value = true

    void preloadDetailUserInfo(fields, row)
  }

  // 切换抽屉模式
  const toggleDrawerMode = (mode: 'read' | 'edit') => {
    if (mode === 'edit' && !supportsDetailEdit.value) {
      ElNotification.info({
        title: '提示',
        message: '当前表格不支持更新'
      })
      return
    }

    if (mode === 'edit' && (!editFunctionDetail.value || !detailRowData.value)) {
      ElNotification.warning({
        title: '提示',
        message: '无法进入编辑模式'
      })
      return
    }
    detailDrawerMode.value = mode
    
    // 🔥 切换模式时更新 URL：查看模式使用 _tab=detail，编辑模式不设置 _tab（只使用 _id）
    const id = detailRowData.value?.id || detailRowData.value?._id
    if (id) {
      const query = buildDetailBaseQuery()
      
      // 🔥 查看模式：设置 _tab=detail；编辑模式：不设置 _tab（只使用 _id）
      if (mode === 'read') {
        query._tab = 'detail'
      }
      // 编辑模式不设置 _tab，只设置 _id
      query._id = String(id)
      
      // 🔥 发出路由更新请求事件
      eventBus.emit(RouteEvent.updateRequested, {
        query,
        replace: true,
        preserveParams: {
          table: true,   // 保留 table 参数
          search: true,  // 保留搜索参数
          state: true    // 保留其他状态参数
        },
        source: 'detail-drawer-mode-toggle'
      })
    }
  }

  // 导航详情（上一个/下一个）
  const handleNavigateDetail = async (direction: 'prev' | 'next') => {
    if (detailTableData.value.length === 0) return

    let newIndex = currentDetailIndex.value
    if (direction === 'prev' && newIndex > 0) {
      newIndex--
    } else if (direction === 'next' && newIndex < detailTableData.value.length - 1) {
      newIndex++
    } else {
      return
    }

    currentDetailIndex.value = newIndex
    const row = detailTableData.value[newIndex]
    detailRowData.value = row
    detailOriginalRow.value = deepClone(row)
    detailDrawerMode.value = 'read'  // 切换记录时，重置为查看模式

    await preloadDetailUserInfo(detailFields.value, row)
  }

  // 提交编辑
  const submitDrawerEdit = async (formViewRef?: InstanceType<typeof FormView> | null) => {
    const currentDetail = options.currentFunctionDetail()
    // 🔥 优先使用传入的 formViewRef，如果没有则使用 detailFormViewRef
    const viewRef = formViewRef || detailFormViewRef.value
    
    if (!currentDetail || !detailRowData.value || !viewRef) {
      ElMessage.error('编辑表单未准备就绪')
      return
    }

    if (!supportsDetailEdit.value) {
      ElMessage.warning('当前表格不支持更新')
      return
    }
    
    try {
      drawerSubmitting.value = true
      const oldValues = detailOriginalRow.value
        ? deepClone(detailOriginalRow.value)
        : undefined
      
      // 🔥 表格更新场景：使用 prepareUpdateData 只返回变更的字段
      const submitData = await viewRef.prepareUpdateData(oldValues ?? {})
      
      const updatedRow = await tableApplicationService.updateRow(
        currentDetail,
        detailRowData.value.id,
        submitData,
        oldValues
      )
      if (updatedRow) {
        detailRowData.value = { ...updatedRow }
        detailOriginalRow.value = deepClone(updatedRow)
        await refreshDetailRowData()
        
        // 🔥 保存成功后，清除 URL 中的表单草稿参数和详情参数
        const query = buildDetailBaseQuery()
        
        // 🔥 发出路由更新请求事件，清除表单字段参数
        eventBus.emit(RouteEvent.updateRequested, {
          query,
          replace: true,
          preserveParams: {
            table: true,   // 保留 table 参数
            search: true,  // 保留搜索参数
            state: true    // 保留其他状态参数
          },
          source: 'detail-drawer-save-success'
        })
        
        ElNotification.success({
          title: '成功',
          message: '更新成功'
        })
        detailDrawerMode.value = 'read'
        detailDrawerVisible.value = false
      }
    } catch (error: any) {
      ElNotification.error({
        title: '错误',
        // 🔥 统一使用 msg 字段
        message: error?.response?.data?.msg || error?.message || '更新失败'
      })
    } finally {
      drawerSubmitting.value = false
    }
  }

  // 刷新详情行数据
  const refreshDetailRowData = async (): Promise<void> => {
    if (!detailRowData.value) return
    const currentId = detailRowData.value.id
    if (currentId === undefined || currentId === null) return
    const state = tableStateManager?.getState?.()
    const tableData = state?.data
    if (!Array.isArray(tableData)) {
      return
    }
    const updatedRow = tableData.find((row: any) => String(row.id) === String(currentId))
    if (updatedRow) {
      detailRowData.value = { ...updatedRow }
      detailOriginalRow.value = deepClone(updatedRow)
    }
  }

  // 获取详情字段值
  const getDetailFieldValue = (fieldCode: string): FieldValue => {
    if (!detailRowData.value) return createEmptyRawFieldValue()
    const value = detailRowData.value[fieldCode]
    return createAutoFieldValue(value)
  }

  // 处理详情抽屉关闭（移除 URL 参数）
  const handleDetailDrawerClose = () => {
    // 如果当前 URL 有 _tab=detail 或 _id 参数，移除它
    // 🔥 阶段3：改为事件驱动，通过 RouteManager 统一处理路由更新
    if (route.query._tab === 'detail' || route.query._id) {
      const query = buildDetailBaseQuery()
      
      // 🔥 发出路由更新请求事件
      eventBus.emit(RouteEvent.updateRequested, {
        query,
        replace: true,
        preserveParams: {
          table: true,   // 保留 table 参数（分页、排序等）
          search: true,  // 保留搜索参数
          state: true    // 保留其他状态参数
        },
        source: 'detail-drawer-close'
      })
    }
  }

  // 打开详情抽屉（从表格行点击）；initialMode 为 'edit' 时直接进入编辑模式（用于「更新」）
  // 列表的 row + tableData 直接当 props 用，同步设 refs 并打开抽屉；URL 同步更新（仅 _tab/_id 变化不会触发表格重载）
  const openDetailDrawer = (
    row: Record<string, any>,
    index?: number,
    tableData?: any[],
    initialMode: 'read' | 'edit' = 'read'
  ) => {
    const currentDetail = options.currentFunctionDetail()
    if (!currentDetail) return
    const resolvedTableData = Array.isArray(tableData)
      ? tableData
      : (tableStateManager.getState().data || [])

    applyDetailDrawerState({
      detail: currentDetail,
      row,
      tableData: Array.isArray(resolvedTableData) ? resolvedTableData : [],
      mode: initialMode === 'edit' && supportsDetailEdit.value ? 'edit' : 'read',
      index
    })

    // URL 同步更新（分享用）；仅 _tab/_id 变化时 query 监听会跳过表格 load，不会闪
    if (options.currentFunction()) {
      const id = row.id || row._id
      if (id) {
        const query = buildDetailBaseQuery()
        query._tab = 'detail'
        query._id = String(id)
        eventBus.emit(RouteEvent.updateRequested, {
          query,
          replace: true,
          preserveParams: { table: true, search: true, state: true },
          source: 'detail-drawer-open'
        })
      }
    }
  }

  const restoreDetailFromQuery = async (
    rawQuery: Record<string, any>,
    trigger: DetailRestoreTrigger,
    detailOverride?: FunctionDetail | null
  ): Promise<void> => {
    const query = rawQuery || {}
    const request = setPendingDetailRestore(query)

    if (!request) {
      closeDetailDrawerForRoute(query)
      return
    }

    const detail = detailOverride ?? options.currentFunctionDetail()
    if (!detail || detail.template_type !== TEMPLATE_TYPE.TABLE || !options.currentFunction()) {
      return
    }

    const tableState = tableStateManager.getState()
    const currentTableData = Array.isArray(tableState.data) ? tableState.data : []
    const currentMatch = findDetailRowMatch(currentTableData, request.rowId)

    if (currentMatch) {
      applyDetailDrawerState({
        detail,
        row: currentMatch.row,
        tableData: currentTableData,
        mode: 'read',
        index: currentMatch.index
      })
      clearPendingDetailRestore()
      return
    }

    if (shouldWaitForDetailTableData({
      loading: tableState.loading,
      dataLength: currentTableData.length,
      trigger
    })) {
      return
    }

    const idField = findDetailIdField(detail)
    if (!idField) {
      clearPendingDetailRestore()
      return
    }

    if (inFlightDetailLookupKey === request.key) {
      return
    }

    const token = latestDetailRestoreToken
    inFlightDetailLookupKey = request.key

    try {
      const lookupRequest = buildDetailLookupSearchRequest({
        detail,
        idFieldCode: idField.code,
        rowId: request.rowId
      })
      const lookupResponse = await apiClient.get<TableResponse>(lookupRequest.url, lookupRequest.params)

      if (token !== latestDetailRestoreToken || pendingDetailRestoreKey !== request.key) {
        return
      }

      const lookupTableData = Array.isArray(lookupResponse.items) ? lookupResponse.items : []
      const refreshedMatch = findDetailRowMatch(lookupTableData, request.rowId)

      if (!refreshedMatch) {
        clearPendingDetailRestore()
        ElNotification.warning({
          title: '提示',
          message: `未找到 id 为 ${request.rowId} 的记录，可能不在当前页`
        })
        return
      }

      applyDetailDrawerState({
        detail,
        row: refreshedMatch.row,
        tableData: currentTableData,
        mode: 'read',
        index: resolveDetailIndex(refreshedMatch.row, getTableDetailFields(detail) as FieldConfig[], currentTableData)
      })
      clearPendingDetailRestore()
    } catch (error) {
      if (token === latestDetailRestoreToken && pendingDetailRestoreKey === request.key) {
        clearPendingDetailRestore()
      }
    } finally {
      if (inFlightDetailLookupKey === request.key) {
        inFlightDetailLookupKey = null
      }
    }
  }

  // 设置 URL 参数监听（用于分享链接）
  // 🔥 阶段4：改为监听 RouteEvent.queryChanged 事件，而不是直接 watch route.query
  // 这样可以避免程序触发的路由更新导致循环
  const setupUrlWatch = (): (() => void) => {
    const unsubscribeQueryChanged = eventBus.on(RouteEvent.queryChanged, async (payload: { query: any, oldQuery: any, source: string }) => {
      // 🔥 只处理用户操作（浏览器前进/后退）或外部变化，不处理程序触发的更新
      if (payload.source === 'router-change') {
        await restoreDetailFromQuery(payload.query || {}, 'route-change')
      }
    })

    const unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, async (payload: { detail: FunctionDetail }) => {
      if (!pendingDetailRestoreQuery || payload.detail.template_type !== TEMPLATE_TYPE.TABLE) {
        return
      }

      await restoreDetailFromQuery(pendingDetailRestoreQuery, 'function-loaded', payload.detail)
    })

    const unsubscribeTableDataLoaded = eventBus.on(TableEvent.dataLoaded, async () => {
      if (!pendingDetailRestoreQuery) {
        return
      }

      await restoreDetailFromQuery(pendingDetailRestoreQuery, 'table-data-loaded')
    })

    void restoreDetailFromQuery(route.query as Record<string, any>, 'setup')

    return () => {
      unsubscribeQueryChanged()
      unsubscribeFunctionLoaded()
      unsubscribeTableDataLoaded()
    }
  }

  return {
    // 状态
    detailDrawerVisible,
    detailDrawerTitle,
    detailRowData,
    detailFields,
    detailOriginalRow,
    detailDrawerMode,
    drawerSubmitting,
    detailFormViewRef,
    detailUserInfoMap,
    detailTableData,
    currentDetailIndex,
    editFunctionDetail,
    
    // 方法
    toggleDrawerMode,
    handleNavigateDetail,
    submitDrawerEdit,
    refreshDetailRowData,
    getDetailFieldValue,
    handleDetailDrawerClose,
    openDetailDrawer,
    
    // 设置
    setupUrlWatch
  }
}
