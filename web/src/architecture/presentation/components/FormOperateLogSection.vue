<template>
  <div class="form-operate-log-section">
    <template v-if="hasOperateLog">
      <div v-loading="loading" class="section-body">
        <div class="section-header">
          <div class="section-title-block">
            <div class="section-title">执行记录</div>
            <div class="section-subtitle">支持筛选、预览详情和直接重放到当前表单。</div>
          </div>
          <div class="section-count">共 {{ total }} 条记录</div>
        </div>

        <div class="filter-section">
          <el-form :inline="true" :model="filters" class="filter-form">
            <el-form-item label="执行用户">
              <div class="user-filter-group">
                <button type="button" class="user-filter-trigger" @click="openUserFilterDialog">
                  <UserDisplay
                    v-if="selectedFilterUser || filters.requestUser"
                    :user-info="selectedFilterUser"
                    :username="filters.requestUser"
                    mode="card"
                    layout="horizontal"
                    size="small"
                  />
                  <span v-else class="user-filter-placeholder">选择用户</span>
                </button>
                <el-button v-if="filters.requestUser" link @click="clearUserFilter">清空</el-button>
              </div>
            </el-form-item>
            <el-form-item label="结果">
              <el-select
                v-model="filters.status"
                clearable
                class="filter-select"
                placeholder="全部"
                @change="handleFilterSubmit"
              >
                <el-option label="成功" value="success" />
                <el-option label="失败" value="failed" />
              </el-select>
            </el-form-item>
            <el-form-item label="来源">
              <el-select
                v-model="filters.source"
                clearable
                class="filter-select"
                placeholder="全部"
                @change="handleFilterSubmit"
              >
                <el-option
                  v-for="item in sourceOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="关键字">
              <el-input
                v-model="filters.keyword"
                clearable
                class="filter-search"
                placeholder="搜索版本、错误、请求或响应内容"
                @keyup.enter="handleFilterSubmit"
                @clear="handleFilterSubmit"
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleFilterSubmit">查询</el-button>
              <el-button @click="resetFilters">重置</el-button>
              <el-button @click="loadLogs({ page: 1 })">刷新</el-button>
            </el-form-item>
          </el-form>
        </div>

        <div class="history-list">
          <el-table
            :data="logs"
            stripe
            size="small"
            class="history-table"
            empty-text="暂无执行记录"
            row-key="id"
            @row-click="handleRowClick"
          >
            <el-table-column label="结果" min-width="280">
              <template #default="{ row }">
                <div class="clickable-cell result-cell">
                  <el-tag :type="getStatusTagType(row)" effect="light" round>
                    {{ getStatusLabel(row) }}
                  </el-tag>
                  <div class="result-copy">
                    <div class="result-title">
                      <span>{{ getResultTitle(row) }}</span>
                      <el-tooltip
                        v-if="getFailureMessage(row)"
                        :content="getFailureMessage(row)"
                        placement="top"
                      >
                        <el-icon class="result-warning-icon"><WarningFilled /></el-icon>
                      </el-tooltip>
                    </div>
                    <div class="result-subtitle">{{ getResultSummary(row) }}</div>
                  </div>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="执行用户" min-width="180">
              <template #default="{ row }">
                <div class="clickable-cell user-cell">
                  <UserDisplay
                    :user-info="getUserInfo(row.request_user)"
                    :username="row.request_user"
                    mode="card"
                    layout="horizontal"
                    size="small"
                  />
                </div>
              </template>
            </el-table-column>

            <el-table-column label="来源" width="110" align="center">
              <template #default="{ row }">
                <div class="clickable-cell source-cell">
                  <el-tag size="small" effect="plain" round class="source-tag">
                    {{ getSourceLabel(row) }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="执行时间" min-width="180">
              <template #default="{ row }">
                <div class="clickable-cell time-cell">
                  <div class="time-primary">{{ formatDateTime(row.created_at) }}</div>
                  <div class="time-secondary">{{ formatRelativeTime(row.created_at) }}</div>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="耗时" width="120" align="center">
              <template #default="{ row }">
                <div class="clickable-cell meta-cell">
                  <div class="meta-primary">{{ formatDuration(getDuration(row)) }}</div>
                  <div class="meta-secondary">{{ getDurationHint(row) }}</div>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="版本" width="140" align="center">
              <template #default="{ row }">
                <div class="clickable-cell version-cell">
                  <span class="version-text">{{ row.version || '-' }}</span>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="操作" width="160" align="right" fixed="right">
              <template #default="{ row }">
                <div class="action-cell">
                  <el-button @click.stop="openPreviewDialog(row)">详情</el-button>
                  <el-button type="primary" @click.stop="handleApplyLog(row)">重放</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div v-if="total > pageSize" class="pagination-wrapper">
          <el-pagination
            background
            layout="total, prev, pager, next"
            :current-page="page"
            :page-size="pageSize"
            :total="total"
            @current-change="handlePageChange"
          />
        </div>
      </div>
    </template>

    <template v-else>
      <el-card shadow="never" class="upgrade-card">
        <div class="upgrade-content">
          <div class="hero-icon-shell">
            <el-icon class="hero-icon"><Clock /></el-icon>
          </div>
          <div class="upgrade-text">
            <div class="upgrade-title">执行记录功能</div>
            <div class="upgrade-desc">升级到企业版即可查看表单的执行历史</div>
          </div>
          <el-button type="primary" size="small" @click="handleUpgrade">
            升级企业版
          </el-button>
        </div>
      </el-card>
    </template>

    <el-dialog
      v-model="previewDialogVisible"
      title="执行详情"
      width="1120px"
      :close-on-click-modal="false"
      class="preview-dialog"
    >
      <template v-if="previewLog">
        <div class="preview-summary">
          <div class="preview-summary-main">
            <el-tag :type="getStatusTagType(previewLog)" effect="light" round>
              {{ getStatusLabel(previewLog) }}
            </el-tag>
            <div class="preview-summary-copy">
              <div class="preview-summary-text">{{ getResultTitle(previewLog) }}</div>
              <div class="preview-summary-desc">{{ getResultMessage(previewLog) }}</div>
            </div>
          </div>
          <div class="preview-summary-meta">
            <span>{{ formatDateTime(previewLog.created_at) }}</span>
            <span>{{ formatDuration(getDuration(previewLog)) }}</span>
          </div>
        </div>

        <div class="preview-overview-grid">
          <div class="overview-item">
            <div class="overview-label">执行用户</div>
            <div class="overview-value">{{ previewLog.request_user || '-' }}</div>
          </div>
          <div class="overview-item">
            <div class="overview-label">来源</div>
            <div class="overview-value">{{ getSourceLabel(previewLog) }}</div>
          </div>
          <div class="overview-item">
            <div class="overview-label">执行时间</div>
            <div class="overview-value">{{ formatDateTime(previewLog.created_at) }}</div>
          </div>
          <div class="overview-item">
            <div class="overview-label">耗时</div>
            <div class="overview-value">{{ formatDuration(getDuration(previewLog)) }}</div>
          </div>
          <div class="overview-item">
            <div class="overview-label">版本</div>
            <div class="overview-value">{{ previewLog.version || '-' }}</div>
          </div>
          <div class="overview-item">
            <div class="overview-label">本次提交</div>
            <div class="overview-value">{{ getRequestFieldCount(previewLog) }} 个字段</div>
          </div>
        </div>

        <div class="preview-panels">
          <div class="preview-panel">
            <div class="preview-panel-header">
              <div class="preview-panel-title">请求参数</div>
              <div class="preview-panel-desc">
                本次提交 {{ getRequestFieldCount(previewLog) }} 个字段，可直接重放回当前表单。
              </div>
            </div>
            <el-input
              :model-value="previewRequestContent"
              type="textarea"
              :rows="18"
              readonly
              class="preview-json-input"
            />
          </div>
          <div class="preview-panel">
            <div class="preview-panel-header">
              <div class="preview-panel-title">响应结果</div>
              <div class="preview-panel-desc">
                会一起回填响应参数和执行信息，方便继续调试或比对结果。
              </div>
            </div>
            <el-input
              :model-value="previewResponseContent"
              type="textarea"
              :rows="18"
              readonly
              class="preview-json-input"
            />
          </div>
        </div>
      </template>

      <template #footer>
        <div class="preview-footer">
          <el-button @click="previewDialogVisible = false">关闭</el-button>
          <el-button type="primary" @click="handlePreviewApply">重放到表单</el-button>
        </div>
      </template>
    </el-dialog>

    <UserPickerDialog
      v-model="userFilterDialogVisible"
      title="选择执行用户"
      placeholder="请输入用户名或邮箱搜索"
      :initial-usernames="filters.requestUser || null"
      @confirm="handleUserFilterConfirm"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Clock, WarningFilled } from '@element-plus/icons-vue'
import {
  ElButton,
  ElCard,
  ElDialog,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElMessage,
  ElOption,
  ElPagination,
  ElSelect,
  ElTable,
  ElTableColumn,
  ElTag,
  ElTooltip
} from 'element-plus'
import type { TagProps } from 'element-plus'
import { getFormOperateLogs, type FormOperateLog } from '@/api/operateLog'
import { useLicenseStore } from '@/stores/license'
import { useUserInfoStore } from '@/stores/userInfo'
import type { FieldConfig } from '@/architecture/domain/types'
import { formatTimestamp } from '@/utils/date'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import UserPickerDialog from '@/shared/components/UserPickerDialog.vue'
import type { UserInfo } from '@/types'

interface Props {
  fullCodePath: string
  functionDetail?: { request?: FieldConfig[] } | null
  autoLoad?: boolean
}

interface ApplyOperateLogPayload {
  log: FormOperateLog
  requestBody: Record<string, any>
  responseBody: Record<string, any> | null
  responseMetadata: Record<string, any> | null
}

const props = withDefaults(defineProps<Props>(), {
  fullCodePath: '',
  functionDetail: null,
  autoLoad: true
})

const emit = defineEmits<{
  (e: 'apply-log', payload: ApplyOperateLogPayload): void
}>()

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
  ;(props.functionDetail?.request || []).forEach((field) => {
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

const readNumber = (value: unknown): number | null => {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim() !== '' && !Number.isNaN(Number(value))) {
    return Number(value)
  }
  return null
}

const getDuration = (log: FormOperateLog): number | null => {
  const payload = getResponsePayload(log)
  return readNumber(payload?.total_cost_mill)
}

const formatDuration = (value: number | null): string => {
  if (value === null || value < 0) {
    return '未记录'
  }
  if (value < 1000) {
    return `${value}ms`
  }
  if (value < 60000) {
    return `${(value / 1000).toFixed(value < 10000 ? 2 : 1)}s`
  }
  const minutes = Math.floor(value / 60000)
  const seconds = ((value % 60000) / 1000).toFixed(1)
  return `${minutes}分${seconds}秒`
}

const getDurationHint = (log: FormOperateLog): string => {
  return getDuration(log) === null ? '旧记录暂未保存耗时' : '来自本次接口执行耗时'
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
  return log.code === 0 ? '本次执行已完成' : '请查看预览中的响应结果'
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
  if (!value) {
    return '-'
  }
  if (typeof value === 'string' && !/^\d+$/.test(value)) {
    return value
  }
  return formatTimestamp(value)
}

const formatRelativeTime = (value: string | number | null | undefined): string => {
  if (!value) {
    return '-'
  }

  const timestamp =
    typeof value === 'string'
      ? (/^\d+$/.test(value) ? Number(value) : new Date(value).getTime())
      : value

  if (Number.isNaN(timestamp)) {
    return '-'
  }

  const diff = Date.now() - timestamp
  if (diff < 0) {
    return formatDateTime(value)
  }

  const minutes = Math.floor(diff / 1000 / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (minutes < 1) {
    return '刚刚'
  }
  if (minutes < 60) {
    return `${minutes}分钟前`
  }
  if (hours < 24) {
    return `${hours}小时前`
  }
  return `${days}天前`
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
    console.warn('[FormOperateLogSection] 加载用户信息失败:', error)
  }
}

const getUserInfo = (username: string | null | undefined): any => {
  if (!username) {
    return null
  }
  return userInfoMap.value.get(username) || null
}

const loadLogs = async (options?: { page?: number }) => {
  if (!hasOperateLog.value || !props.fullCodePath) {
    return
  }

  if (typeof options?.page === 'number') {
    page.value = options.page
  }

  loading.value = true
  try {
    const response = await getFormOperateLogs({
      full_code_path: props.fullCodePath,
      action: 'form_submit',
      request_user: filters.value.requestUser || undefined,
      source: filters.value.source || undefined,
      status: (filters.value.status as 'success' | 'failed' | '') || undefined,
      keyword: filters.value.keyword.trim() || undefined,
      page: page.value,
      page_size: pageSize,
      order_by: 'created_at DESC'
    })
    logs.value = response.logs || []
    total.value = response.total || 0
    await loadUserInfos()
  } catch (error: any) {
    console.error('[FormOperateLogSection] 加载执行记录失败:', error)
    ElMessage.warning(`加载执行记录失败: ${error?.message || '未知错误'}`)
  } finally {
    loading.value = false
  }
}

const handlePageChange = (nextPage: number) => {
  loadLogs({ page: nextPage })
}

const handleFilterSubmit = () => {
  loadLogs({ page: 1 })
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
    keyword: '',
    status: '',
    source: ''
  }
  selectedFilterUser.value = null
  loadLogs({ page: 1 })
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
    ElMessage.warning('这条记录没有可回填的请求参数')
    return
  }
  emit('apply-log', payload)
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

watch(
  () => [props.fullCodePath, props.autoLoad, hasOperateLog.value] as const,
  ([fullCodePath, autoLoad, enabled], oldValues) => {
    const oldFullCodePath = oldValues?.[0] || ''
    if (fullCodePath !== oldFullCodePath) {
      page.value = 1
      logs.value = []
      total.value = 0
    }

    if (fullCodePath && autoLoad && enabled) {
      loadLogs({ page: 1 })
    }
  },
  { immediate: true }
)

defineExpose({
  loadLogs
})
</script>

<style scoped lang="scss">
.form-operate-log-section {
  min-height: 320px;
  padding: 20px;
}

.section-body {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.section-title-block {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.section-subtitle {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.section-count {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.filter-section {
  margin-bottom: 4px;
}

.filter-form {
  margin: 0;
}

.filter-search {
  width: 320px;
}

.filter-select {
  width: 130px;
}

.user-filter-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-filter-trigger {
  display: flex;
  align-items: center;
  min-width: 180px;
  min-height: 32px;
  padding: 0 10px;
  border-radius: 8px;
  border: 1px solid var(--el-border-color);
  background: var(--el-bg-color);
  cursor: pointer;
}

.user-filter-trigger:hover {
  border-color: var(--el-color-primary-light-5);
}

.user-filter-placeholder {
  font-size: 13px;
  color: var(--el-text-color-placeholder);
}

.history-list {
  min-height: 220px;
}

.history-table {
  --el-table-row-hover-bg-color: var(--el-fill-color-light);
}

.history-table :deep(.el-table__row) {
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.history-table :deep(.el-table__cell) {
  vertical-align: top;
}

.history-table :deep(.cell) {
  padding-top: 9px;
  padding-bottom: 9px;
}

.clickable-cell {
  min-height: 46px;
}

.result-cell {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  min-width: 0;
}

.result-copy {
  min-width: 0;
}

.result-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.5;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.result-subtitle {
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.result-warning-icon {
  font-size: 14px;
  color: var(--el-color-warning);
  cursor: help;
}

.user-cell,
.time-cell,
.meta-cell {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.source-tag {
  border-color: var(--el-border-color);
}

.source-cell {
  display: flex;
  align-items: center;
  justify-content: center;
}

.time-primary,
.meta-primary {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.time-secondary,
.meta-secondary {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.version-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.version-text {
  display: inline-block;
  font-size: 13px;
  font-weight: 600;
  line-height: 1.5;
  color: var(--el-text-color-primary);
  word-break: break-all;
}

.action-cell {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
}

.upgrade-card {
  border-radius: 12px;
}

.upgrade-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.upgrade-text {
  flex: 1;
}

.hero-icon-shell {
  width: 42px;
  height: 42px;
  border-radius: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));
  border: 1px solid var(--el-color-primary-light-5);
}

.hero-icon {
  color: var(--el-color-primary);
  font-size: 18px;
}

.upgrade-title {
  font-size: 16px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.upgrade-desc {
  margin-top: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.preview-summary {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  padding: 18px 20px;
  border-radius: 16px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  margin-bottom: 18px;
}

.preview-dialog :deep(.el-dialog) {
  background: var(--el-bg-color);
}

.preview-dialog :deep(.el-dialog__body) {
  background: var(--el-bg-color);
}

.preview-summary-main {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.preview-summary-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.preview-summary-text {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.preview-summary-desc {
  font-size: 13px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}

.preview-summary-meta {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.preview-overview-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 18px;
}

.overview-item {
  padding: 14px 16px;
  border-radius: 14px;
  background: var(--el-fill-color-blank);
  border: 1px solid var(--el-border-color-lighter);
}

.overview-label {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.overview-value {
  margin-top: 6px;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.6;
  color: var(--el-text-color-primary);
  word-break: break-word;
}

.preview-panels {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.preview-panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.preview-panel-header {
  margin-bottom: 12px;
}

.preview-panel-title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.5;
  color: var(--el-text-color-primary);
}

.preview-panel-desc {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.preview-json-input :deep(.el-textarea__inner) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  background: var(--el-bg-color);
  border-color: var(--el-border-color);
  min-height: 420px;
}

.preview-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

@media (max-width: 768px) {
  .section-header,
  .preview-summary {
    flex-direction: column;
    align-items: stretch;
  }

  .form-operate-log-section {
    padding: 16px;
  }

  .preview-overview-grid {
    grid-template-columns: 1fr;
  }

  .preview-panels {
    grid-template-columns: 1fr;
  }

  .filter-search,
  .filter-select,
  .user-filter-trigger {
    width: 100%;
  }

  .preview-summary-meta {
    justify-content: flex-start;
  }
}
</style>
