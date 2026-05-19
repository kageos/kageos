import { computed, ref, watch, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { TagProps } from 'element-plus'
import { formatTimestamp } from '@/architecture/shared/date'
import { useUserInfoStore } from '@/architecture/presentation/context/appStoresContext'
import { getTableOperateLogs, type TableOperateLog } from '@/architecture/presentation/context/api/operateLog'
import type { FieldConfig } from '@/architecture/domain/types'
import { getFunctionByPath } from '@/architecture/presentation/context/api/function'
import type { FunctionDetail } from '@/architecture/domain/types'
import { Logger } from '@/architecture/shared/logger'
import { getTableAllFields } from '@/architecture/domain/utils/functionSchemaSelectors'

type OperateLogScope = 'row' | 'function' | 'directory'

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

const OPERATE_LOG_PAGE_SIZE = 12

interface UseOperateLogSectionOptions {
  fullCodePath: Ref<string>
  rowId: Ref<number>
  functionDetail: Ref<any>
  autoLoad: Ref<boolean>
  scope?: Ref<OperateLogScope>
}

export function useOperateLogSection({
  fullCodePath,
  rowId,
  functionDetail,
  autoLoad,
  scope,
}: UseOperateLogSectionOptions) {
  const userInfoStore = useUserInfoStore()

  const logs = ref<TableOperateLog[]>([])
  const loading = ref(false)
  const keyword = ref('')
  const actionFilter = ref('')
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
    { label: '全部操作', value: '' },
    { label: '新增', value: 'OnTableAddRow' },
    { label: '更新', value: 'OnTableUpdateRow' },
    { label: '删除', value: 'OnTableDeleteRows' },
  ])

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

    if (seconds < 60) return '刚刚'
    if (minutes < 60) return `${minutes}分钟前`
    if (hours < 24) return `${hours}小时前`
    if (days < 30) return `${days}天前`
    if (months < 12) return `${months}个月前`
    return `${years}年前`
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
    } catch (error) {
      Logger.warn('[OperateLogSection]', '加载用户信息失败', { error })
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
      const response = await getTableOperateLogs({
        ...(scopeValue === 'directory'
          ? { full_code_path_prefix: fullCodePath.value }
          : { full_code_path: fullCodePath.value }),
        ...(scopeValue === 'row' ? { row_id: rowId.value } : {}),
        ...(actionFilter.value ? { action: actionFilter.value } : {}),
        ...(keyword.value.trim() ? { keyword: keyword.value.trim() } : {}),
        page: currentPage.value,
        page_size: pageSize.value,
        order_by: 'created_at DESC',
      })
      logs.value = response.logs || []
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
      ElMessage.warning('加载操作日志失败: ' + (error.message || '未知错误'))
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

  const getActionTagType = (action: string): TagProps['type'] => {
    switch (action) {
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
        return '新增'
      case 'OnTableUpdateRow':
        return '更新'
      case 'OnTableDeleteRows':
        return '删除'
      default:
        return action
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

    let fields: FieldConfig[] | null = getTableAllFields(detail as FunctionDetail)
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
      return '-'
    }
    if (typeof rawValue === 'boolean') {
      return rawValue ? '是' : '否'
    }
    if (typeof rawValue === 'number') {
      return String(rawValue)
    }
    if (typeof rawValue === 'string') {
      return rawValue.length > 120 ? `${rawValue.slice(0, 120)}...` : rawValue
    }
    if (Array.isArray(rawValue)) {
      if (rawValue.length === 0) {
        return '-'
      }
      const simpleValues = rawValue.every((item) => ['string', 'number', 'boolean'].includes(typeof item))
      if (simpleValues) {
        const text = rawValue.map((item) => formatLogValue(item)).join('、')
        return text.length > 120 ? `${text.slice(0, 120)}...` : text
      }
      return `${rawValue.length} 项`
    }
    if (typeof rawValue === 'object') {
      if (Array.isArray(rawValue.files)) {
        return `${rawValue.files.length} 个文件`
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

  const getChangeEntries = (log: TableOperateLog): OperateLogChangeEntry[] => {
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

  const getValueEntries = (log: TableOperateLog): OperateLogValueEntry[] => {
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

  const getPrimaryEntries = (log: TableOperateLog): OperateLogValueEntry[] => {
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

  const getLogTitle = (log: TableOperateLog): string => {
    const recordName = log.row_id ? `记录 #${log.row_id}` : '一条记录'
    switch (log.action) {
      case 'OnTableAddRow':
        return `新增了${recordName}`
      case 'OnTableUpdateRow':
        return `更新了${recordName}`
      case 'OnTableDeleteRows':
        return `删除了${recordName}`
      default:
        return `执行了 ${log.action}`
    }
  }

  const getLogEmptyText = (log: TableOperateLog): string => {
    switch (log.action) {
      case 'OnTableAddRow':
        return '记录已新增，暂无字段详情'
      case 'OnTableUpdateRow':
        return '记录已更新，暂无字段变更详情'
      case 'OnTableDeleteRows':
        return '记录已删除'
      default:
        return '暂无更多操作详情'
    }
  }

  const getLogSummary = (log: TableOperateLog): string => {
    const entries = getPrimaryEntries(log)
    if (entries.length === 0) {
      return getLogEmptyText(log)
    }
    return entries
      .map((entry) => `${entry.fieldName}: ${formatLogValue(entry.value)}`)
      .join(' · ')
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
    actionOptions,
    currentPage,
    pageSize,
    total,
    getUserInfo,
    getActionTagType,
    getActionLabel,
    formatLogValue,
    getChangeEntries,
    getValueEntries,
    getPrimaryEntries,
    getLogTitle,
    getLogEmptyText,
    getLogSummary,
    isLogExpanded,
    toggleLogExpanded,
    handleSearch,
    handleActionChange,
    handlePageChange,
    load,
    showRowIdColumn,
    showResourceColumn,
  }
}
