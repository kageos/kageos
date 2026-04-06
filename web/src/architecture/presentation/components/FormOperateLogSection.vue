<template>
  <div class="form-operate-log-section">
    <template v-if="hasOperateLog">
      <div v-loading="loading" class="section-body">
        <el-card shadow="never" class="history-card">
          <template #header>
            <div class="section-header">
              <div class="section-title">最近执行记录</div>
              <el-button link :loading="loading" @click="loadLogs({ page: 1 })">刷新</el-button>
            </div>
          </template>
          <el-table
            :data="logs"
            stripe
            class="history-table"
            empty-text="暂无执行记录"
          >
            <el-table-column label="结果" min-width="260">
              <template #default="{ row }">
                <div class="result-cell">
                  <el-tag :type="getStatusTagType(row)" effect="light" round>
                    {{ getStatusLabel(row) }}
                  </el-tag>
                  <div class="result-copy">
                    <div class="result-title">{{ getResultMessage(row) }}</div>
                    <div class="result-subtitle">{{ getResultSummary(row) }}</div>
                  </div>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="执行用户" min-width="170">
              <template #default="{ row }">
                <UserDisplay
                  :user-info="getUserInfo(row.request_user)"
                  :username="row.request_user"
                  mode="card"
                  layout="horizontal"
                  size="small"
                />
              </template>
            </el-table-column>

            <el-table-column label="执行时间" min-width="180">
              <template #default="{ row }">
                <div class="time-cell">
                  <div class="time-primary">{{ formatDateTime(row.created_at) }}</div>
                  <div class="time-secondary">{{ formatRelativeTime(row.created_at) }}</div>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="耗时" min-width="110" align="center">
              <template #default="{ row }">
                <div class="meta-cell">
                  <div class="meta-primary">{{ formatDuration(getDuration(row)) }}</div>
                  <div class="meta-secondary">{{ getDurationHint(row) }}</div>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="版本" min-width="120" align="center">
              <template #default="{ row }">
                <span class="version-text">{{ row.version || '-' }}</span>
              </template>
            </el-table-column>

            <el-table-column label="操作" width="160" align="right" fixed="right">
              <template #default="{ row }">
                <div class="action-cell">
                  <el-button text @click="openPreviewDialog(row)">预览</el-button>
                  <el-button type="primary" @click="handleApplyLog(row)">重放</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <div v-if="total > pageSize" class="pagination-wrapper">
          <el-pagination
            background
            layout="prev, pager, next"
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
      title="执行记录预览"
      width="820px"
      :close-on-click-modal="false"
      class="preview-dialog"
    >
      <template v-if="previewLog">
        <div class="preview-summary">
          <div class="preview-summary-main">
            <el-tag :type="getStatusTagType(previewLog)" effect="light" round>
              {{ getStatusLabel(previewLog) }}
            </el-tag>
            <span class="preview-summary-text">{{ getResultMessage(previewLog) }}</span>
          </div>
          <div class="preview-summary-meta">
            <span>执行时间：{{ formatDateTime(previewLog.created_at) }}</span>
            <span>耗时：{{ formatDuration(getDuration(previewLog)) }}</span>
          </div>
        </div>

        <el-tabs v-model="previewActiveTab" class="preview-tabs">
          <el-tab-pane label="请求参数" name="request">
            <div class="preview-tab-intro">
              本次提交 {{ getRequestFieldCount(previewLog) }} 个字段，可直接重放回当前表单。
            </div>
            <el-input
              :model-value="previewRequestContent"
              type="textarea"
              :rows="16"
              readonly
              class="preview-json-input"
            />
          </el-tab-pane>
          <el-tab-pane label="响应结果" name="response">
            <div class="preview-tab-intro">
              会回填响应参数和执行信息，方便继续调试或比对结果。
            </div>
            <el-input
              :model-value="previewResponseContent"
              type="textarea"
              :rows="16"
              readonly
              class="preview-json-input"
            />
          </el-tab-pane>
        </el-tabs>
      </template>

      <template #footer>
        <div class="preview-footer">
          <el-button @click="previewDialogVisible = false">关闭</el-button>
          <el-button type="primary" @click="handlePreviewApply">重放到表单</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Clock } from '@element-plus/icons-vue'
import {
  ElButton,
  ElCard,
  ElDialog,
  ElEmpty,
  ElIcon,
  ElInput,
  ElMessage,
  ElPagination,
  ElTable,
  ElTableColumn,
  ElTag
} from 'element-plus'
import type { TagProps } from 'element-plus'
import { getFormOperateLogs, type FormOperateLog } from '@/api/operateLog'
import { useLicenseStore } from '@/stores/license'
import { useUserInfoStore } from '@/stores/userInfo'
import type { FieldConfig } from '@/core/types/field'
import { formatTimestamp } from '@/utils/date'
import UserDisplay from '@/shared/components/UserDisplay.vue'

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
const previewActiveTab = ref('request')
const previewLog = ref<FormOperateLog | null>(null)

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

const getRequestFieldHint = (log: FormOperateLog): string => {
  const labels = getRequestFieldLabels(log)
  if (labels.length === 0) {
    return '本次未记录请求字段'
  }
  const visible = labels.slice(0, 3).join('、')
  const remaining = labels.length - Math.min(labels.length, 3)
  return remaining > 0 ? `${visible} 等 ${labels.length} 项` : visible
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

const openPreviewDialog = (log: FormOperateLog) => {
  previewLog.value = log
  previewActiveTab.value = 'request'
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
  display: flex;
  flex-direction: column;
  min-height: 320px;
}

.section-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.history-card {
  border-radius: 12px;
  border-color: var(--el-border-color-lighter);
  background: var(--el-bg-color);
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.history-table :deep(.el-table__cell) {
  vertical-align: top;
}

.history-table :deep(.cell) {
  padding-top: 14px;
  padding-bottom: 14px;
}

.result-cell {
  display: flex;
  align-items: flex-start;
  gap: 12px;
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
}

.result-subtitle {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.time-cell,
.meta-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
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

.version-text {
  display: inline-block;
  font-size: 12px;
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
  gap: 16px;
  padding: 14px 16px;
  border-radius: 16px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  margin-bottom: 16px;
}

.preview-dialog :deep(.el-dialog) {
  background: var(--el-bg-color);
}

.preview-dialog :deep(.el-dialog__body) {
  background: var(--el-bg-color);
}

.preview-summary-main {
  display: flex;
  align-items: center;
  gap: 10px;
}

.preview-summary-text {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.preview-summary-meta {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.preview-tabs :deep(.el-tabs__content) {
  overflow: visible;
}

.preview-tab-intro {
  margin-bottom: 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.preview-json-input :deep(.el-textarea__inner) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  background: var(--el-bg-color);
  border-color: var(--el-border-color);
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

  .preview-summary-meta {
    justify-content: flex-start;
  }
}
</style>
