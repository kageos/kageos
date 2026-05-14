import { computed, h, ref, watch, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { TagProps } from 'element-plus'
import { formatTimestamp } from '@/utils/date'
import { useLicenseStore } from '@/architecture/infrastructure/stores/license'
import { useUserInfoStore } from '@/architecture/infrastructure/stores/userInfo'
import { getTableOperateLogs, type TableOperateLog } from '@/architecture/infrastructure/api/operateLog'
import { widgetComponentFactory } from '@/architecture/infrastructure/widgetRegistry'
import { convertToFieldValue } from '@/utils/field'
import type { FieldConfig } from '@/architecture/domain/types'
import { getFunctionByPath } from '@/architecture/infrastructure/api/function'
import type { FunctionDetail } from '@/architecture/domain/types'
import { Logger } from '@/architecture/runtime/utils/logger'
import { getTableListFields } from '@/utils/functionSchemaSelectors'

interface UseOperateLogSectionOptions {
  fullCodePath: Ref<string>
  rowId: Ref<number>
  functionDetail: Ref<any>
  autoLoad: Ref<boolean>
}

export function useOperateLogSection({
  fullCodePath,
  rowId,
  functionDetail,
  autoLoad,
}: UseOperateLogSectionOptions) {
  const licenseStore = useLicenseStore()
  const userInfoStore = useUserInfoStore()

  const logs = ref<TableOperateLog[]>([])
  const loading = ref(false)
  const functionDetailCache = ref<FunctionDetail | null>(null)
  const userInfoMap = ref<Map<string, any>>(new Map())
  const hasLoaded = ref(false)
  const lastLoadParams = ref<{ fullCodePath: string; rowId: number } | null>(null)

  const hasOperateLog = computed(() => licenseStore.hasOperateLog)

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
    if (functionDetail.value) {
      const hasResponse = getTableListFields(functionDetail.value as FunctionDetail).length > 0
      if (hasResponse) {
        functionDetailCache.value = functionDetail.value as FunctionDetail
        return
      }
    }

    if (fullCodePath.value && !functionDetailCache.value) {
      try {
        const detail = await getFunctionByPath(fullCodePath.value)
        if (detail && getTableListFields(detail as unknown as FunctionDetail).length > 0) {
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

  const loadOperateLogs = async () => {
    if (!hasOperateLog.value) {
      return
    }
    if (!fullCodePath.value || !rowId.value) {
      return
    }

    loading.value = true
    try {
      await loadFunctionDetail()
      const response = await getTableOperateLogs({
        full_code_path: fullCodePath.value,
        row_id: rowId.value,
        page: 1,
        page_size: 50,
        order_by: 'created_at DESC',
      })
      logs.value = response.logs || []
      await loadUserInfos()
      hasLoaded.value = true
      lastLoadParams.value = {
        fullCodePath: fullCodePath.value,
        rowId: rowId.value,
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

  const getFieldConfig = (fieldCode: string): FieldConfig | null => {
    const detail = functionDetailCache.value || functionDetail.value
    if (!detail) {
      return null
    }

    let fields: FieldConfig[] | null = getTableListFields(detail as FunctionDetail)
    if (fields.length === 0 && Array.isArray(detail)) {
      fields = detail
    }

    if (!Array.isArray(fields) || fields.length === 0) {
      return null
    }

    return fields.find((field: any) => field.code === fieldCode) || null
  }

  const getFieldName = (fieldCode: string | number): string => {
    const normalizedFieldCode = String(fieldCode)
    const field = getFieldConfig(normalizedFieldCode)
    return field?.name || normalizedFieldCode
  }

  const renderFieldValue = (fieldCode: string | number, rawValue: any) => {
    const normalizedFieldCode = String(fieldCode)
    const field = getFieldConfig(normalizedFieldCode)

    if (!field) {
      Logger.warn('[OperateLogSection]', '未找到字段配置', {
        fieldCode,
        functionDetail: functionDetail.value
      })
      return h('span', { class: 'text-fallback' }, rawValue !== null && rawValue !== undefined ? String(rawValue) : '-')
    }

    try {
      let processedValue = rawValue
      if (field.widget?.type === 'files' && rawValue && typeof rawValue === 'object') {
        if (rawValue.files && Array.isArray(rawValue.files)) {
          processedValue = rawValue
        } else {
          processedValue = {
            files: Array.isArray(rawValue) ? rawValue : [rawValue],
            remark: rawValue.remark || '',
            metadata: rawValue.metadata || null,
          }
        }
      }

      const value = convertToFieldValue(processedValue, field)
      const WidgetComponent = widgetComponentFactory.getRequestComponent(field.widget?.type || 'input')

      if (!WidgetComponent) {
        Logger.warn('[OperateLogSection]', '未找到组件', {
          widgetType: field.widget?.type || 'input'
        })
        return h('span', { class: 'text-fallback' }, rawValue !== null && rawValue !== undefined ? String(rawValue) : '-')
      }

      return h(WidgetComponent, {
        field,
        value,
        'model-value': value,
        'field-path': normalizedFieldCode,
        mode: 'detail',
      })
    } catch (error) {
      Logger.error('[OperateLogSection]', '渲染字段值失败', {
        fieldCode,
        rawValue,
        error
      })
      return h('span', { class: 'text-fallback' }, rawValue !== null && rawValue !== undefined ? String(rawValue) : '-')
    }
  }

  const handleUpgrade = () => {
    ElMessage.info('请联系管理员升级到企业版')
  }

  watch(
    [fullCodePath, rowId, functionDetail],
    ([newFullCodePath, newRowId, newFunctionDetail], [oldFullCodePath = '', oldRowId = 0, oldFunctionDetail]) => {
      if (newFullCodePath && newRowId && (newFullCodePath !== oldFullCodePath || newRowId !== oldRowId)) {
        hasLoaded.value = false
        lastLoadParams.value = null
        logs.value = []
      }

      if (newFunctionDetail !== oldFunctionDetail) {
        functionDetailCache.value = null
      }

      if (!autoLoad.value) {
        return
      }

      if (newFullCodePath && newRowId && (newFullCodePath !== oldFullCodePath || newRowId !== oldRowId)) {
        void loadOperateLogs()
      } else if (newFullCodePath && newRowId && !oldFullCodePath && !oldRowId) {
        void loadOperateLogs()
      }
    },
    { immediate: true }
  )

  const load = () => {
    const paramsChanged =
      !lastLoadParams.value ||
      lastLoadParams.value.fullCodePath !== fullCodePath.value ||
      lastLoadParams.value.rowId !== rowId.value

    if (paramsChanged) {
      hasLoaded.value = false
      logs.value = []
      functionDetailCache.value = null
    }

    if (!hasLoaded.value) {
      void loadOperateLogs()
    }
  }

  return {
    hasOperateLog,
    logs,
    loading,
    formatDateTime,
    formatRelativeTime,
    getUserInfo,
    getActionTagType,
    getActionLabel,
    parseJSON,
    getFieldName,
    renderFieldValue,
    handleUpgrade,
    load,
  }
}
