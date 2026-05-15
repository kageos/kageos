import { computed, ref, watch, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { TagProps } from 'element-plus'
import { getFormOperateLogs, type FormOperateLog } from '@/architecture/infrastructure/api/operateLog'
import { useLicenseStore, useUserInfoStore } from '@/architecture/presentation/context/appStoresContext'
import type { FieldConfig, FunctionDetail } from '@/architecture/domain/types'
import type { UserInfo } from '@/architecture/domain/types'
import {
  formatExecutionDateTime,
  formatExecutionDuration,
  formatExecutionRelativeTime,
  readExecutionNumber
} from '@/architecture/presentation/utils/executionLog'
import { Logger } from '@/architecture/shared/logger'
import { getFormRequestFields } from '@/architecture/domain/utils/functionSchemaSelectors'

export interface ApplyOperateLogPayload {
  log: FormOperateLog
  requestBody: Record<string, any>
  responseBody: Record<string, any> | null
  responseMetadata: Record<string, any> | null
}

interface UseFormOperateLogSectionOptions {
  fullCodePath: Ref<string>
  functionDetail: Ref<FunctionDetail | null | undefined>
  autoLoad: Ref<boolean>
  emitApplyLog: (payload: ApplyOperateLogPayload) => void
}

export function useFormOperateLogSection({
  fullCodePath,
  functionDetail,
  autoLoad,
  emitApplyLog
}: UseFormOperateLogSectionOptions) {
  const licenseStore = useLicenseStore()
  const userInfoStore = useUserInfoStore()

  const hasOperateLog = computed(() => licenseStore.hasOperateLog)
  const logs = ref<FormOperateLog[]>([])
  const loading = ref(false)
  const total = ref(0)
  const page = ref(1)
  const pageSize = 10
  const userInfoMap = ref<Map<string, any>>(new Map())
  const previewDialogVisible = ref(false)
  const previewLog = ref<FormOperateLog | null>(null)
  const userFilterDialogVisible = ref(false)
  const selectedFilterUser = ref<UserInfo | null>(null)
  const filters = ref({
    requestUser: '',
    traceId: '',
    keyword: '',
    status: '',
    source: ''
  })

  const sourceOptions = [
    { label: '浏览器', value: 'browser' },
    { label: '定时任务', value: 'scheduled_task' },
    { label: '智能体', value: 'agent' },
    { label: 'API', value: 'api' }
  ]

  const requestFieldMap = computed(() => {
    const map = new Map<string, FieldConfig>()
    getFormRequestFields(functionDetail.value).forEach((field) => {
      map.set(field.code, field)
    })
    return map
  })

  const parseMaybeJSON = (value: unknown): any => {
    if (typeof value === 'string') {
      try {
        return JSON.parse(value)
      } catch {
        return value
      }
    }
    return value
  }

  const stringifyPretty = (value: unknown): string => {
    const parsed = parseMaybeJSON(value)
    if (parsed === null || parsed === undefined || parsed === '') {
      return '{}'
    }
    if (typeof parsed === 'string') {
      return parsed
    }
    try {
      return JSON.stringify(parsed, null, 2)
    } catch {
      return String(parsed)
    }
  }

  const getObjectPayload = (value: unknown): Record<string, any> | null => {
    const parsed = parseMaybeJSON(value)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return null
    }
    return parsed as Record<string, any>
  }

  const getRequestBody = (log: FormOperateLog): Record<string, any> | null => {
    return getObjectPayload(log.request_body)
  }

  const getRequestFieldLabels = (log: FormOperateLog): string[] => {
    const requestBody = getRequestBody(log)
    if (!requestBody) {
      return []
    }
    return Object.keys(requestBody).map((key) => requestFieldMap.value.get(key)?.name || key)
  }

  const getRequestFieldCount = (log: FormOperateLog): number => {
    return getRequestFieldLabels(log).length
  }

  const getResponsePayload = (log: FormOperateLog): Record<string, any> | null => {
    return getObjectPayload(log.response_body)
  }

  const getDuration = (log: FormOperateLog): number | null => {
    const payload = getResponsePayload(log)
    return readExecutionNumber(payload?.total_cost_mill)
  }

  const getResponseResult = (log: FormOperateLog): Record<string, any> | null => {
    const payload = getResponsePayload(log)
    if (!payload || payload.result === undefined || payload.result === null) {
      return null
    }
    if (typeof payload.result === 'object' && !Array.isArray(payload.result)) {
      return payload.result as Record<string, any>
    }
    return {
      result: payload.result
    }
  }

  const getResponseMetadata = (log: FormOperateLog): Record<string, any> | null => {
    const payload = getResponsePayload(log)
    const metadata: Record<string, any> = {}

    const duration = getDuration(log)
    if (duration !== null) {
      metadata.total_cost_mill = duration
    }
    if (payload?.trace_id || log.trace_id) {
      metadata.trace_id = payload?.trace_id || log.trace_id
    }
    if (payload?.version || log.version) {
      metadata.version = payload?.version || log.version
    }

    return Object.keys(metadata).length > 0 ? metadata : null
  }

  const getStatusTagType = (log: FormOperateLog): TagProps['type'] => {
    return log.code === 0 ? 'success' : 'danger'
  }

  const getStatusLabel = (log: FormOperateLog): string => {
    return log.code === 0 ? '成功' : '失败'
  }

  const getResultTitle = (log: FormOperateLog): string => {
    return log.code === 0 ? '执行成功' : '执行失败'
  }

  const getResultMessage = (log: FormOperateLog): string => {
    if (log.msg) {
      return log.msg
    }

    const payload = getResponsePayload(log)
    if (!payload) {
      return log.code === 0 ? '执行成功' : '执行失败'
    }

    return payload.msg || payload.error || (log.code === 0 ? '执行成功' : '执行失败')
  }

  const getResultSummary = (log: FormOperateLog): string => {
    const payload = getResponsePayload(log)
    if (!payload) {
      return log.code === 0 ? '本次执行已完成' : '本次执行返回错误'
    }

    const result = payload.result
    if (Array.isArray(result)) {
      return `返回 ${result.length} 项结果`
    }
    if (result && typeof result === 'object') {
      return `返回 ${Object.keys(result).length} 个结果字段`
    }
    if (result !== undefined && result !== null && result !== '') {
      return `返回结果：${String(result)}`
    }
    return log.code === 0 ? '本次执行已完成' : '请查看预览中的输出结果'
  }

  const getFailureMessage = (log: FormOperateLog): string => {
    if (log.code === 0) {
      return ''
    }
    return getResultMessage(log)
  }

  const inferSourceFromUserAgent = (userAgent?: string | null): string => {
    const normalized = (userAgent || '').toLowerCase()
    if (!normalized) {
      return ''
    }
    if (normalized.includes('mozilla') || normalized.includes('chrome') || normalized.includes('safari')) {
      return 'browser'
    }
    if (
      normalized.includes('postman') ||
      normalized.includes('curl') ||
      normalized.includes('httpie') ||
      normalized.includes('apifox')
    ) {
      return 'api'
    }
    return ''
  }

  const getSourceCode = (log: FormOperateLog): string => {
    const direct = (log.source || '').trim().toLowerCase()
    if (direct === 'browser' || direct === 'scheduled_task' || direct === 'agent' || direct === 'api') {
      return direct
    }
    return inferSourceFromUserAgent(log.user_agent)
  }

  const getSourceLabel = (log: FormOperateLog): string => {
    switch (getSourceCode(log)) {
      case 'scheduled_task':
        return '定时任务'
      case 'agent':
        return '智能体'
      case 'api':
      case 'browser':
        return getSourceCode(log) === 'api' ? 'API' : '浏览器'
      default:
        return '-'
    }
  }

  const formatDateTime = (value: string | number | null | undefined): string => {
    return formatExecutionDateTime(value)
  }

  const formatRelativeTime = (value: string | number | null | undefined): string => {
    return formatExecutionRelativeTime(value)
  }

  const loadUserInfos = async () => {
    const usernames = Array.from(new Set(logs.value.map((log) => log.request_user).filter(Boolean))) as string[]
    if (usernames.length === 0) {
      userInfoMap.value = new Map()
      return
    }

    try {
      const users = await userInfoStore.batchGetUserInfo(usernames)
      const map = new Map<string, any>()
      users.forEach((user: any) => {
        map.set(user.username, user)
      })
      userInfoMap.value = map
    } catch (error) {
      Logger.warn('[FormOperateLogSection]', '加载用户信息失败', { error })
    }
  }

  const getUserInfo = (username: string | null | undefined): any => {
    if (!username) {
      return null
    }
    return userInfoMap.value.get(username) || null
  }

  const loadLogs = async (options?: { page?: number }) => {
    if (!hasOperateLog.value || !fullCodePath.value) {
      return
    }

    if (typeof options?.page === 'number') {
      page.value = options.page
    }

    loading.value = true
    try {
      const response = await getFormOperateLogs({
        full_code_path: fullCodePath.value,
        action: 'form_submit',
        request_user: filters.value.requestUser || undefined,
        source: filters.value.source || undefined,
        status: (filters.value.status as 'success' | 'failed' | '') || undefined,
        trace_id: filters.value.traceId || undefined,
        keyword: filters.value.keyword.trim() || undefined,
        page: page.value,
        page_size: pageSize,
        order_by: 'created_at DESC'
      })
      logs.value = response.logs || []
      total.value = response.total || 0
      await loadUserInfos()
    } catch (error: any) {
      Logger.error('[FormOperateLogSection]', '加载执行记录失败', { error })
      ElMessage.warning(`加载执行记录失败: ${error?.message || '未知错误'}`)
    } finally {
      loading.value = false
    }
  }

  const handlePageChange = (nextPage: number) => {
    void loadLogs({ page: nextPage })
  }

  const handleFilterSubmit = () => {
    void loadLogs({ page: 1 })
  }

  const openUserFilterDialog = () => {
    userFilterDialogVisible.value = true
  }

  const handleUserFilterConfirm = (users: UserInfo[]) => {
    const user = users[0] || null
    selectedFilterUser.value = user
    filters.value.requestUser = user?.username || ''
    handleFilterSubmit()
  }

  const clearUserFilter = () => {
    selectedFilterUser.value = null
    filters.value.requestUser = ''
    handleFilterSubmit()
  }

  const resetFilters = () => {
    filters.value = {
      requestUser: '',
      traceId: '',
      keyword: '',
      status: '',
      source: ''
    }
    selectedFilterUser.value = null
    void loadLogs({ page: 1 })
  }

  const handleRowClick = (row: FormOperateLog) => {
    openPreviewDialog(row)
  }

  const openPreviewDialog = (log: FormOperateLog) => {
    previewLog.value = log
    previewDialogVisible.value = true
  }

  const buildApplyPayload = (log: FormOperateLog): ApplyOperateLogPayload | null => {
    const requestBody = getRequestBody(log)
    if (!requestBody) {
      return null
    }

    return {
      log,
      requestBody,
      responseBody: getResponseResult(log),
      responseMetadata: getResponseMetadata(log)
    }
  }

  const handleApplyLog = (log: FormOperateLog) => {
    const payload = buildApplyPayload(log)
    if (!payload) {
      ElMessage.warning('这条记录没有可回填的输入参数')
      return
    }
    Logger.debug('FormOperateLogSection', '准备回填执行记录到表单', {
      logId: log.id,
      requestKeys: Object.keys(payload.requestBody || {}),
      hasResponseBody: !!payload.responseBody,
      hasResponseMetadata: !!payload.responseMetadata
    })
    emitApplyLog(payload)
  }

  const handlePreviewApply = () => {
    if (!previewLog.value) {
      return
    }
    handleApplyLog(previewLog.value)
    previewDialogVisible.value = false
  }

  const previewRequestContent = computed(() => {
    return previewLog.value ? stringifyPretty(previewLog.value.request_body) : '{}'
  })

  const previewResponseContent = computed(() => {
    return previewLog.value ? stringifyPretty(previewLog.value.response_body) : '{}'
  })

  const handleUpgrade = () => {
    ElMessage.info('请联系管理员升级到企业版')
  }

  const openWithFilters = (nextFilters: Partial<typeof filters.value>) => {
    filters.value = {
      requestUser: '',
      traceId: '',
      keyword: '',
      status: '',
      source: '',
      ...nextFilters
    }
    selectedFilterUser.value = null
    previewDialogVisible.value = false
    void loadLogs({ page: 1 })
  }

  watch(
    () => [fullCodePath.value, autoLoad.value, hasOperateLog.value] as const,
    ([nextFullCodePath, nextAutoLoad, enabled], oldValues) => {
      const oldFullCodePath = oldValues?.[0] || ''
      if (nextFullCodePath !== oldFullCodePath) {
        page.value = 1
        logs.value = []
        total.value = 0
      }

      if (nextFullCodePath && nextAutoLoad && enabled) {
        void loadLogs({ page: 1 })
      }
    },
    { immediate: true }
  )

  return {
    hasOperateLog,
    logs,
    loading,
    total,
    page,
    pageSize,
    previewDialogVisible,
    previewLog,
    userFilterDialogVisible,
    selectedFilterUser,
    filters,
    sourceOptions,
    previewRequestContent,
    previewResponseContent,
    getStatusTagType,
    getStatusLabel,
    getResultTitle,
    getResultSummary,
    getFailureMessage,
    getSourceLabel,
    formatDateTime,
    formatRelativeTime,
    getDuration,
    getUserInfo,
    getRequestFieldCount,
    getResultMessage,
    formatExecutionDuration,
    loadLogs,
    handlePageChange,
    handleFilterSubmit,
    openUserFilterDialog,
    handleUserFilterConfirm,
    clearUserFilter,
    resetFilters,
    handleRowClick,
    openPreviewDialog,
    handleApplyLog,
    handlePreviewApply,
    handleUpgrade,
    openWithFilters
  }
}
