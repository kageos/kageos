import { computed, ref, watch, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { TagProps } from 'element-plus'
import { formatTimestamp } from '@/architecture/shared/date'
import { useUserInfoStore } from '@/architecture/presentation/context/appStoresContext'
import { getOperateLogs, type OperateLog } from '@/architecture/presentation/context/api/operateLog'
import { searchUsersFuzzy } from '@/architecture/presentation/context/api/user'
import type { FieldConfig, UserInfo } from '@/architecture/domain/types'
import { getFunctionByPath } from '@/architecture/presentation/context/api/function'
import type { FunctionDetail } from '@/architecture/domain/types'
import { Logger } from '@/architecture/shared/logger'
import { getFormRequestFields, getTableAllFields } from '@/architecture/domain/utils/functionSchemaSelectors'
import { translate } from '@/architecture/shared/i18n'

type OperateLogScope = 'row' | 'function' | 'directory'
type OperateLogEntry = {
  id: number
  tenant_user: string
  request_user: string
  action: string
  app: string
  full_code_path: string
  row_id: number
  updates?: any
  old_values?: any
  ip_address?: string
  user_agent?: string
  trace_id?: string
  version?: string
  created_at: string
  resource_type?: string
  status?: string
  summary?: string
  details_json?: any
  old_values_json?: any
  new_values_json?: any
  source?: string
}

interface OperateLogChangeEntry {
  fieldCode: string
  fieldName: string
  oldValue: any
  newValue: any
  hasOldValue: boolean
}

interface OperateLogValueEntry {
  fieldCode: string
  fieldName: string
  value: any
}

interface OperateLogMetaEntry {
  label: string
  value: string
}

const OPERATE_LOG_PAGE_SIZE = 12

interface UseOperateLogSectionOptions {
  fullCodePath: Ref<string>
  rowId: Ref<number>
  functionDetail: Ref<any>
  autoLoad: Ref<boolean>
  scope?: Ref<OperateLogScope>
  onApplyFormLog?: (requestBody: Record<string, any>, responseBody: Record<string, any> | null) => void
}

export function useOperateLogSection({
  fullCodePath,
  rowId,
  functionDetail,
  autoLoad,
  scope,
  onApplyFormLog,
}: UseOperateLogSectionOptions) {
  const t = translate
  const userInfoStore = useUserInfoStore()

  const logs = ref<OperateLogEntry[]>([])
  const loading = ref(false)
  const keyword = ref('')
  const actionFilter = ref('')
  const sourceFilter = ref('')
  const userFilter = ref('')
  const userOptions = ref<Array<{ label: string; value: string; userInfo?: UserInfo }>>([])
  const userFilterLoading = ref(false)
  const currentPage = ref(1)
  const pageSize = ref(OPERATE_LOG_PAGE_SIZE)
  const total = ref(0)
  const expandedLogIds = ref<number[]>([])
  const functionDetailCache = ref<FunctionDetail | null>(null)
  const functionDetailMap = ref<Map<string, FunctionDetail>>(new Map())
  const userInfoMap = ref<Map<string, any>>(new Map())
  const hasLoaded = ref(false)
  const currentScope = () => scope?.value || 'row'
  const lastLoadParams = ref<{ fullCodePath: string; rowId: number; scope: OperateLogScope } | null>(null)
  const showRowIdColumn = computed(() => currentScope() !== 'row')
  const showResourceColumn = computed(() => currentScope() === 'directory')
  const actionOptions = computed(() => [
    { label: t('operateLog.allActions'), value: '' },
    ...(isFormOperateLog.value ? [{ label: t('operateLog.submit'), value: 'form_submit' }] : []),
    { label: t('operateLog.add'), value: 'OnTableAddRow' },
    { label: t('operateLog.update'), value: 'OnTableUpdateRow' },
    { label: t('operateLog.delete'), value: 'OnTableDeleteRows' },
  ])
  const sourceOptions = computed(() => [
    { label: t('operateLog.allSources'), value: '' },
    { label: t('operateLog.sourceBrowser'), value: 'browser' },
    { label: t('operateLog.sourceOpenAPI'), value: 'openapi' },
    { label: t('operateLog.sourceAgent'), value: 'agent' },
    { label: t('operateLog.sourcePublicShare'), value: 'public_share' },
    { label: t('operateLog.sourceUnknown'), value: 'unknown' },
  ])

  const isFormOperateLog = computed(() => {
    const detail = functionDetail.value as FunctionDetail | null
    return detail?.template_type === 'form' || detail?.schema?.type === 'form'
  })

  const formatDateTime = (dateTime: string | number | null | undefined): string => {
    if (!dateTime) return '-'
    if (typeof dateTime === 'string') {
      if (/^\d+$/.test(dateTime)) {
        return formatTimestamp(Number(dateTime))
      }
      return dateTime
    }
    return formatTimestamp(dateTime)
  }

  const formatRelativeTime = (dateTime: string | number | null | undefined): string => {
    if (!dateTime) return '-'

    let timestamp: number
    if (typeof dateTime === 'string') {
      timestamp = /^\d+$/.test(dateTime) ? Number(dateTime) : new Date(dateTime).getTime()
    } else {
      timestamp = dateTime
    }

    if (isNaN(timestamp)) {
      return '-'
    }

    const now = Date.now()
    const diff = now - timestamp
    if (diff < 0) {
      return formatDateTime(dateTime)
    }

    const seconds = Math.floor(diff / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)
    const months = Math.floor(days / 30)
    const years = Math.floor(days / 365)

    if (seconds < 60) return t('operateLog.justNow')
    if (minutes < 60) return t('operateLog.minutesAgo', { count: minutes })
    if (hours < 24) return t('operateLog.hoursAgo', { count: hours })
    if (days < 30) return t('operateLog.daysAgo', { count: days })
    if (months < 12) return t('operateLog.monthsAgo', { count: months })
    return t('operateLog.yearsAgo', { count: years })
  }

  const loadFunctionDetail = async () => {
    if (currentScope() === 'directory') {
      return
    }

    if (functionDetail.value) {
      const hasResponse = getTableAllFields(functionDetail.value as FunctionDetail).length > 0
      if (hasResponse) {
        functionDetailCache.value = functionDetail.value as FunctionDetail
        return
      }
    }

    if (fullCodePath.value && !functionDetailCache.value) {
      try {
        const detail = await getFunctionByPath(fullCodePath.value)
        if (detail && getTableAllFields(detail as unknown as FunctionDetail).length > 0) {
          functionDetailCache.value = detail as unknown as FunctionDetail
        }
      } catch (error) {
        Logger.warn('[OperateLogSection]', '加载函数详情失败', { error })
      }
    }
  }

  const loadUserInfos = async () => {
    if (logs.value.length === 0) {
      return
    }

    const usernames = new Set<string>()
    logs.value.forEach((log) => {
      if (log.request_user) {
        usernames.add(log.request_user)
      }
    })

    if (usernames.size === 0) {
      return
    }

    try {
      const users = await userInfoStore.batchGetUserInfo(Array.from(usernames))
      userInfoMap.value = new Map()
      users.forEach((user: any) => {
        userInfoMap.value.set(user.username, user)
      })
      mergeUserOptions(Array.from(usernames).map((username) => {
        const userInfo = userInfoMap.value.get(username)
        return {
          label: formatUserOptionLabel(username, userInfo),
          value: username,
          userInfo,
        }
      }))
    } catch (error) {
      Logger.warn('[OperateLogSection]', '加载用户信息失败', { error })
    }
  }

  const formatUserOptionLabel = (username: string, userInfo?: UserInfo | null): string => {
    if (!userInfo?.nickname) {
      return username
    }
    return `${username}(${userInfo.nickname})`
  }

  const mergeUserOptions = (options: Array<{ label: string; value: string; userInfo?: UserInfo }>) => {
    const nextMap = new Map<string, { label: string; value: string; userInfo?: UserInfo }>()
    userOptions.value.forEach((option) => nextMap.set(option.value, option))
    options.forEach((option) => nextMap.set(option.value, option))
    userOptions.value = Array.from(nextMap.values())
  }

  const searchUserOptions = async (query: string) => {
    const keywordText = query.trim()
    if (!keywordText) {
      return
    }

    userFilterLoading.value = true
    try {
      const response = await searchUsersFuzzy(keywordText, 20)
      userOptions.value = (response.users || []).map((user) => ({
        label: formatUserOptionLabel(user.username, user),
        value: user.username,
        userInfo: user,
      }))
    } catch (error) {
      Logger.warn('[OperateLogSection]', '搜索操作用户失败', { keyword: keywordText, error })
      userOptions.value = []
    } finally {
      userFilterLoading.value = false
    }
  }

  const loadDirectoryFunctionDetails = async () => {
    if (currentScope() !== 'directory' || logs.value.length === 0) {
      return
    }

    const paths = Array.from(new Set(
      logs.value
        .map((log) => log.full_code_path)
        .filter((path): path is string => Boolean(path))
    ))
    const missingPaths = paths.filter((path) => !functionDetailMap.value.has(path))
    if (missingPaths.length === 0) {
      return
    }

    const nextMap = new Map(functionDetailMap.value)
    await Promise.all(missingPaths.map(async (path) => {
      try {
        const detail = await getFunctionByPath(path)
        if (detail && getTableAllFields(detail as unknown as FunctionDetail).length > 0) {
          nextMap.set(path, detail as unknown as FunctionDetail)
        }
      } catch (error) {
        Logger.warn('[OperateLogSection]', '加载目录日志函数详情失败', { path, error })
      }
    }))
    functionDetailMap.value = nextMap
  }

  const loadOperateLogs = async () => {
    const scopeValue = currentScope()
    if (!fullCodePath.value || (scopeValue === 'row' && !rowId.value)) {
      return
    }

    loading.value = true
    try {
      await loadFunctionDetail()
      const resourceType = scopeValue === 'directory'
        ? ''
        : (isFormOperateLog.value ? 'form' : 'table')
      const response = await getOperateLogs({
        ...(resourceType ? { resource_type: resourceType } : {}),
        ...(scopeValue === 'directory'
          ? { resource_path_prefix: fullCodePath.value }
          : { resource_path: fullCodePath.value }),
        ...(scopeValue === 'row' ? { row_id: rowId.value } : {}),
        ...(actionFilter.value ? { action: actionFilter.value } : {}),
        ...(sourceFilter.value ? { source: sourceFilter.value } : {}),
        ...(userFilter.value ? { actor_user: userFilter.value } : {}),
        ...(keyword.value.trim() ? { keyword: keyword.value.trim() } : {}),
        page: currentPage.value,
        page_size: pageSize.value,
        order_by: 'created_at DESC',
      })
      logs.value = (response.logs || []).map(normalizeOperateLog)
      total.value = response.total || 0
      expandedLogIds.value = []
      await loadDirectoryFunctionDetails()
      await loadUserInfos()
      hasLoaded.value = true
      lastLoadParams.value = {
        fullCodePath: fullCodePath.value,
        rowId: rowId.value,
        scope: scopeValue,
      }
    } catch (error: any) {
      Logger.error('[OperateLogSection]', '加载操作日志失败', { error })
      ElMessage.warning(t('operateLog.loadFailed', { message: error.message || t('common.none') }))
    } finally {
      loading.value = false
    }
  }

  const getUserInfo = (username: string | null | undefined): any => {
    if (!username) {
      return null
    }
    return userInfoMap.value.get(username) || null
  }

  const normalizeOperateLog = (log: OperateLog): OperateLogEntry => ({
    id: log.id,
    tenant_user: log.tenant_user,
    request_user: log.actor_user,
    action: log.action,
    app: log.app,
    full_code_path: log.resource_path,
    row_id: readOperateLogRowId(log),
    updates: log.new_values_json,
    old_values: log.old_values_json,
    ip_address: log.ip_address,
    user_agent: log.user_agent,
    trace_id: log.trace_id,
    version: log.details_json?.version || log.new_values_json?.version,
    created_at: log.created_at,
    resource_type: log.resource_type,
    status: log.status,
    summary: log.summary,
    details_json: log.details_json,
    old_values_json: log.old_values_json,
    new_values_json: log.new_values_json,
    source: log.source,
  })

  const readOperateLogRowId = (log: OperateLog): number => {
    const raw = log.details_json?.row_id ?? log.target_id
    if (typeof raw === 'number' && Number.isFinite(raw)) return raw
    if (typeof raw === 'string' && raw.trim() !== '' && !Number.isNaN(Number(raw))) return Number(raw)
    return 0
  }

  const getActionTagType = (action: string): TagProps['type'] => {
    switch (action) {
      case 'form_submit':
        return 'success'
      case 'OnTableAddRow':
        return 'success'
      case 'OnTableUpdateRow':
        return 'warning'
      case 'OnTableDeleteRows':
        return 'danger'
      default:
        return 'info'
    }
  }

  const getActionLabel = (action: string): string => {
    switch (action) {
      case 'OnTableAddRow':
        return t('operateLog.add')
      case 'form_submit':
        return t('operateLog.submit')
      case 'OnTableUpdateRow':
        return t('operateLog.update')
      case 'OnTableDeleteRows':
        return t('operateLog.delete')
      default:
        return action
    }
  }

  const getSourceLabel = (source?: string): string => {
    switch (source) {
      case 'browser':
        return t('operateLog.sourceBrowser')
      case 'openapi':
      case 'api':
        return t('operateLog.sourceOpenAPI')
      case 'agent':
        return t('operateLog.sourceAgent')
      case 'public_share':
        return t('operateLog.sourcePublicShare')
      case 'unknown':
      case '':
      case undefined:
        return t('operateLog.sourceUnknown')
      default:
        return source
    }
  }

  const getSourceTagType = (source?: string): TagProps['type'] => {
    switch (source) {
      case 'agent':
        return 'warning'
      case 'openapi':
      case 'api':
        return 'success'
      case 'public_share':
        return 'info'
      case 'unknown':
      case '':
      case undefined:
        return 'danger'
      default:
        return 'info'
    }
  }

  const parseJSON = (jsonStr: string | any): any => {
    if (typeof jsonStr === 'string') {
      try {
        return JSON.parse(jsonStr)
      } catch {
        return {}
      }
    }
    return jsonStr || {}
  }

  const getFieldConfig = (fieldCode: string, fullCodePathForLog?: string): FieldConfig | null => {
    const detail = fullCodePathForLog
      ? functionDetailMap.value.get(fullCodePathForLog) || functionDetailCache.value || functionDetail.value
      : functionDetailCache.value || functionDetail.value
    if (!detail) {
      return null
    }

    let fields: FieldConfig[] | null = isFormOperateLog.value
      ? getFormRequestFields(detail as FunctionDetail)
      : getTableAllFields(detail as FunctionDetail)
    if (fields.length === 0 && Array.isArray(detail)) {
      fields = detail
    }

    if (!Array.isArray(fields) || fields.length === 0) {
      return null
    }

    return fields.find((field: any) => field.code === fieldCode) || null
  }

  const getFieldName = (fieldCode: string | number, fullCodePathForLog?: string): string => {
    const normalizedFieldCode = String(fieldCode)
    const field = getFieldConfig(normalizedFieldCode, fullCodePathForLog)
    return field?.name || normalizedFieldCode
  }

  const formatLogValue = (rawValue: any): string => {
    if (rawValue === null || rawValue === undefined || rawValue === '') {
      return t('operateLog.emptyValue')
    }
    if (typeof rawValue === 'boolean') {
      return rawValue ? t('operateLog.boolYes') : t('operateLog.boolNo')
    }
    if (typeof rawValue === 'number') {
      return String(rawValue)
    }
    if (typeof rawValue === 'string') {
      return rawValue.length > 120 ? `${rawValue.slice(0, 120)}...` : rawValue
    }
    if (Array.isArray(rawValue)) {
      if (rawValue.length === 0) {
        return t('operateLog.emptyValue')
      }
      const simpleValues = rawValue.every((item) => ['string', 'number', 'boolean'].includes(typeof item))
      if (simpleValues) {
        const text = rawValue.map((item) => formatLogValue(item)).join('、')
        return text.length > 120 ? `${text.slice(0, 120)}...` : text
      }
      return t('operateLog.items', { count: rawValue.length })
    }
    if (typeof rawValue === 'object') {
      if (Array.isArray(rawValue.files)) {
        return t('operateLog.files', { count: rawValue.files.length })
      }
      for (const key of ['name', 'title', 'label', 'text', 'value']) {
        if (typeof rawValue[key] === 'string' && rawValue[key]) {
          return rawValue[key]
        }
      }
      try {
        const text = JSON.stringify(rawValue)
        return text.length > 120 ? `${text.slice(0, 120)}...` : text
      } catch {
        return String(rawValue)
      }
    }
    return String(rawValue)
  }

  const getChangeEntries = (log: OperateLogEntry): OperateLogChangeEntry[] => {
    const updates = parseJSON(log.updates)
    if (!updates || typeof updates !== 'object' || Array.isArray(updates)) {
      return []
    }

    const oldValues = parseJSON(log.old_values)
    return Object.entries(updates).map(([fieldCode, newValue]) => ({
      fieldCode,
      fieldName: getFieldName(fieldCode, log.full_code_path),
      oldValue: oldValues?.[fieldCode],
      newValue,
      hasOldValue: oldValues && Object.prototype.hasOwnProperty.call(oldValues, fieldCode),
    }))
  }

  const getValueEntries = (log: OperateLogEntry): OperateLogValueEntry[] => {
    const values = parseJSON(log.action === 'OnTableDeleteRows' ? log.old_values : log.updates)
    if (!values || typeof values !== 'object' || Array.isArray(values)) {
      return []
    }

    return Object.entries(values).map(([fieldCode, value]) => ({
      fieldCode,
      fieldName: getFieldName(fieldCode, log.full_code_path),
      value,
    }))
  }

  const getFormRequestEntries = (log: OperateLogEntry): OperateLogValueEntry[] => {
    const values = parseJSON(log.old_values_json ?? log.old_values ?? log.details_json?.request_body)
    if (!values || typeof values !== 'object' || Array.isArray(values)) {
      return []
    }

    return Object.entries(values).map(([fieldCode, value]) => ({
      fieldCode,
      fieldName: getFieldName(fieldCode, log.full_code_path),
      value,
    }))
  }

  const getPrimaryEntries = (log: OperateLogEntry): OperateLogValueEntry[] => {
    if (log.action === 'OnTableUpdateRow') {
      return getChangeEntries(log)
        .slice(0, 3)
        .map((item) => ({
          fieldCode: item.fieldCode,
          fieldName: item.fieldName,
          value: item.newValue,
        }))
    }
    return getValueEntries(log).slice(0, 3)
  }

  const getLogTitle = (log: OperateLogEntry): string => {
    if (log.action === 'form_submit') {
      return log.status === 'failed' ? t('operateLog.formSubmitFailed') : t('operateLog.formSubmitted')
    }
    const recordName = log.row_id ? t('common.rowRecord', { id: log.row_id }) : t('operateLog.record')
    switch (log.action) {
      case 'OnTableAddRow':
        return t('operateLog.added', { record: recordName })
      case 'OnTableUpdateRow':
        return t('operateLog.updated', { record: recordName })
      case 'OnTableDeleteRows':
        return t('operateLog.deleted', { record: recordName })
      default:
        return t('operateLog.executed', { action: log.action })
    }
  }

  const getLogEmptyText = (log: OperateLogEntry): string => {
    switch (log.action) {
      case 'OnTableAddRow':
        return t('operateLog.addEmpty')
      case 'OnTableUpdateRow':
        return t('operateLog.updateEmpty')
      case 'OnTableDeleteRows':
        return t('operateLog.deleteEmpty')
      case 'form_submit':
        return t('operateLog.formSubmitEmpty')
      default:
        return t('operateLog.detailEmpty')
    }
  }

  const getLogSummary = (log: OperateLogEntry): string => {
    if (log.action === 'form_submit' && log.summary) {
      return log.summary
    }
    const response = getLogResponseBody(log)
    if (log.status === 'failed' && response?.error) {
      return String(response.error)
    }
    if (log.summary && (log.resource_type === 'team_access' || !['OnTableAddRow', 'OnTableUpdateRow', 'OnTableDeleteRows'].includes(log.action))) {
      return log.summary
    }
    const entries = getPrimaryEntries(log)
    if (entries.length === 0) {
      return getLogEmptyText(log)
    }
    return entries
      .map((entry) => `${entry.fieldName}: ${formatLogValue(entry.value)}`)
      .join(' · ')
  }

  const readNumber = (value: unknown): number | null => {
    if (typeof value === 'number' && Number.isFinite(value)) return value
    if (typeof value === 'string' && value.trim() !== '' && !Number.isNaN(Number(value))) return Number(value)
    return null
  }

  const getLogResponseBody = (log: OperateLogEntry): Record<string, any> | null => {
    const response = parseJSON(log.details_json?.response_body)
    if (!response || typeof response !== 'object' || Array.isArray(response)) {
      return null
    }
    return response as Record<string, any>
  }

  const getLogDuration = (log: OperateLogEntry): number | null => {
    return readNumber(log.details_json?.duration_millis ?? getLogResponseBody(log)?.total_cost_mill)
  }

  const getLogStatusLabel = (log: OperateLogEntry): string => {
    return log.status === 'failed' ? t('operateLog.failed') : t('operateLog.success')
  }

  const getLogStatusTagType = (log: OperateLogEntry): TagProps['type'] => {
    return log.status === 'failed' ? 'danger' : 'success'
  }

  const formatDuration = (value: number | null): string => {
    if (value === null || value < 0) return t('operateLog.durationMissing')
    if (value < 1000) return `${value}ms`
    if (value < 60000) return `${(value / 1000).toFixed(value < 10000 ? 2 : 1)}s`
    const minutes = Math.floor(value / 60000)
    const seconds = ((value % 60000) / 1000).toFixed(1)
    return `${minutes}m ${seconds}s`
  }

  const getLogMetaEntries = (log: OperateLogEntry): OperateLogMetaEntry[] => {
    const response = getLogResponseBody(log)
    const entries: OperateLogMetaEntry[] = []
    const duration = getLogDuration(log)
    if (duration !== null) entries.push({ label: t('operateLog.duration'), value: formatDuration(duration) })
    if (log.version) entries.push({ label: t('operateLog.version'), value: log.version })
    if (response?.error) entries.push({ label: t('operateLog.error'), value: String(response.error) })
    if (log.trace_id) entries.push({ label: 'Trace', value: log.trace_id })
    if (log.source) entries.push({ label: t('operateLog.source'), value: getSourceLabel(log.source) })
    if (log.ip_address) entries.push({ label: 'IP', value: log.ip_address })
    return entries
  }

  const isLogExpanded = (logId: number): boolean => {
    return expandedLogIds.value.includes(logId)
  }

  const toggleLogExpanded = (logId: number) => {
    if (isLogExpanded(logId)) {
      expandedLogIds.value = expandedLogIds.value.filter((id) => id !== logId)
      return
    }
    expandedLogIds.value = [...expandedLogIds.value, logId]
  }

  const canApplyFormLog = (log: OperateLogEntry): boolean => {
    return log.action === 'form_submit' && typeof onApplyFormLog === 'function'
  }

  const applyFormLog = (log: OperateLogEntry) => {
    const requestBody = parseJSON(log.old_values_json ?? log.updates)
    const responseBody = parseJSON(log.new_values_json)
    if (!requestBody || typeof requestBody !== 'object' || Array.isArray(requestBody)) {
      ElMessage.warning(t('operateLog.formReplayEmpty'))
      return
    }
    onApplyFormLog?.(
      requestBody as Record<string, any>,
      responseBody && typeof responseBody === 'object' && !Array.isArray(responseBody)
        ? responseBody as Record<string, any>
        : null
    )
  }

  const resetAndLoad = () => {
    currentPage.value = 1
    hasLoaded.value = false
    logs.value = []
    void loadOperateLogs()
  }

  const handleSearch = () => {
    resetAndLoad()
  }

  const handleActionChange = () => {
    resetAndLoad()
  }

  const handleSourceChange = () => {
    resetAndLoad()
  }

  const handleUserChange = () => {
    resetAndLoad()
  }

  const handlePageChange = (page: number) => {
    currentPage.value = page
    hasLoaded.value = false
    logs.value = []
    void loadOperateLogs()
  }

  watch(
    [fullCodePath, rowId, functionDetail, scope || ref<OperateLogScope>('row')],
    ([newFullCodePath, newRowId, newFunctionDetail, newScope], [oldFullCodePath = '', oldRowId = 0, oldFunctionDetail, oldScope = 'row']) => {
      const canLoad = Boolean(newFullCodePath) && (newScope !== 'row' || Boolean(newRowId))
      const paramsChanged = newFullCodePath !== oldFullCodePath || newRowId !== oldRowId || newScope !== oldScope

      if (canLoad && paramsChanged) {
        hasLoaded.value = false
        lastLoadParams.value = null
        logs.value = []
        total.value = 0
        currentPage.value = 1
        userFilter.value = ''
        sourceFilter.value = ''
        functionDetailMap.value = new Map()
      }

      if (newFunctionDetail !== oldFunctionDetail) {
        functionDetailCache.value = null
      }

      if (!autoLoad.value) {
        return
      }

      if (canLoad && paramsChanged) {
        void loadOperateLogs()
      } else if (canLoad && !oldFullCodePath && !oldRowId) {
        void loadOperateLogs()
      }
    },
    { immediate: true }
  )

  const load = () => {
    const paramsChanged =
      !lastLoadParams.value ||
      lastLoadParams.value.fullCodePath !== fullCodePath.value ||
      lastLoadParams.value.rowId !== rowId.value ||
      lastLoadParams.value.scope !== currentScope()

    if (paramsChanged) {
      hasLoaded.value = false
      logs.value = []
      total.value = 0
      currentPage.value = 1
      userFilter.value = ''
      sourceFilter.value = ''
      functionDetailCache.value = null
      functionDetailMap.value = new Map()
    }

    hasLoaded.value = false
    void loadOperateLogs()
  }

  return {
    logs,
    loading,
    formatDateTime,
    formatRelativeTime,
    keyword,
    actionFilter,
    sourceFilter,
    userFilter,
    userOptions,
    userFilterLoading,
    actionOptions,
    sourceOptions,
    currentPage,
    pageSize,
    total,
    getUserInfo,
    getActionTagType,
    getActionLabel,
    getSourceLabel,
    getSourceTagType,
    formatLogValue,
    getChangeEntries,
    getValueEntries,
    getFormRequestEntries,
    getPrimaryEntries,
    getLogTitle,
    getLogEmptyText,
    getLogSummary,
    getLogDuration,
    getLogStatusLabel,
    getLogStatusTagType,
    getLogMetaEntries,
    formatDuration,
    canApplyFormLog,
    applyFormLog,
    isLogExpanded,
    toggleLogExpanded,
    handleSearch,
    handleActionChange,
    handleSourceChange,
    handleUserChange,
    searchUserOptions,
    handlePageChange,
    load,
    showRowIdColumn,
    showResourceColumn,
    isFormOperateLog,
  }
}
