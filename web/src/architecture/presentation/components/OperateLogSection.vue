<template>
  <div class="operate-log-section" :class="{ 'is-embedded': embedded }">
    <el-divider v-if="!embedded" />
    <div v-if="!isFormOperateLog" class="operate-log-header">
      <div class="operate-log-title-group">
        <el-icon class="operate-log-icon"><Clock /></el-icon>
        <span class="operate-log-title">{{ title || t('operateLog.title') }}</span>
      </div>
      <div>
        <el-button
          v-if="showRefresh"
          size="small"
          :icon="Refresh"
          :loading="loading"
          @click="load"
        >
          {{ t('common.refresh') }}
        </el-button>
      </div>
    </div>
    <div v-loading="loading" class="operate-log-content">
      <template v-if="isFormOperateLog">
        <div class="form-operate-log-section">
          <div class="history-card">
            <div class="section-header">
              <div class="section-heading">
                <div class="section-title">{{ t('operateLog.recentExecutions') }}</div>
                <div class="section-subtitle">{{ t('operateLog.executionSubtitle') }}</div>
              </div>
              <el-button
                link
                size="small"
                :icon="Refresh"
                :loading="loading"
                @click="load"
              >
                {{ t('common.refresh') }}
              </el-button>
            </div>

            <div class="form-history-toolbar">
              <el-input
                v-model="keyword"
                class="history-search"
                clearable
                :placeholder="t('operateLog.keywordPlaceholder')"
                :prefix-icon="Search"
                @keyup.enter="handleSearch"
                @clear="handleSearch"
              />
              <el-select
                v-model="userFilter"
                class="history-user-select"
                filterable
                remote
                reserve-keyword
                clearable
                :remote-method="searchUserOptions"
                :loading="userFilterLoading"
                :placeholder="t('operateLog.userPlaceholder')"
                @change="handleUserChange"
                @clear="handleUserChange"
              >
                <el-option
                  v-for="option in userOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
              <el-select
                v-model="actionFilter"
                class="history-action-select"
                :placeholder="t('operateLog.actionPlaceholder')"
                clearable
                @change="handleActionChange"
              >
                <el-option
                  v-for="option in actionOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
              <el-select
                v-model="sourceFilter"
                class="history-source-select"
                :placeholder="t('operateLog.sourcePlaceholder')"
                clearable
                @change="handleSourceChange"
              >
                <el-option
                  v-for="option in sourceOptions"
                  :key="option.value"
                  :label="option.label"
                  :value="option.value"
                />
              </el-select>
              <el-button
                type="primary"
                plain
                :icon="Search"
                @click="handleSearch"
              >
                {{ t('common.search') }}
              </el-button>
            </div>

            <el-table
              :data="logs"
              stripe
              class="history-table form-history-table"
              :empty-text="t('operateLog.empty')"
              :row-key="getLogRowKey"
              :row-class-name="getLogRowClassName"
              :expand-row-keys="expandedLogRowKeys"
              @expand-change="handleLogExpandChange"
            >
              <el-table-column type="expand" width="40">
                <template #default="{ row }">
                  <div class="table-log-details">
                    <div
                      v-if="getFormRequestEntries(asOperateLogEntry(row)).length > 0"
                      class="value-list"
                    >
                      <div
                        v-for="item in getFormRequestEntries(asOperateLogEntry(row))"
                        :key="item.fieldCode"
                        class="value-row"
                        :class="{ 'is-file-value': isFilesField(item.field) }"
                      >
                        <span class="value-field">{{ item.fieldName }}</span>
                        <OperateLogFieldValue
                          class="value-text"
                          :field="item.field"
                          :raw-value="item.value"
                          :field-path="item.fieldCode"
                          :empty-text="t('operateLog.emptyValue')"
                          compact
                        />
                      </div>
                    </div>

                    <div v-else class="text-muted">{{ getLogEmptyText(asOperateLogEntry(row)) }}</div>

                    <div class="log-meta-grid">
                      <span
                        v-for="item in getLogMetaEntries(asOperateLogEntry(row))"
                        :key="`${item.label}:${item.value}`"
                      >
                        {{ item.label }}: {{ item.value }}
                      </span>
                      <span v-if="row.user_agent">UA: {{ row.user_agent }}</span>
                    </div>
                  </div>
                </template>
              </el-table-column>

              <el-table-column :label="t('operateLog.result')" min-width="260">
                <template #default="{ row }">
                  <div class="result-cell">
                    <el-tag :type="getFormStatusTagType(row)" effect="light" round>
                      {{ getFormStatusLabel(row) }}
                    </el-tag>
                    <div class="result-copy">
                      <div class="result-title">{{ getFormResultMessage(row) }}</div>
                      <div class="result-subtitle">{{ getFormResultSummary(row) }}</div>
                    </div>
                  </div>
                </template>
              </el-table-column>

              <el-table-column :label="t('operateLog.actor')" min-width="170">
                <template #default="{ row }">
                  <UserDisplay
                    :user-info="getUserInfo(row.request_user)"
                    :username="row.request_user"
                    mode="simple"
                    layout="horizontal"
                    size="small"
                  />
                </template>
              </el-table-column>

              <el-table-column :label="t('operateLog.source')" min-width="110" align="center">
                <template #default="{ row }">
                  <el-tag :type="getSourceTagType(row.source)" size="small" effect="light">
                    {{ getLogSourceLabel(asOperateLogEntry(row)) }}
                  </el-tag>
                </template>
              </el-table-column>

              <el-table-column :label="t('operateLog.executor')" min-width="120" align="center">
                <template #default="{ row }">
                  <el-tag :type="getExecutorTagType(row.executor_type)" size="small" effect="light">
                    {{ getExecutorLabel(row.executor_type) }}
                  </el-tag>
                </template>
              </el-table-column>

              <el-table-column :label="t('operateLog.executedAt')" min-width="180">
                <template #default="{ row }">
                  <div class="time-cell">
                    <div class="time-primary">{{ formatDateTime(row.created_at) }}</div>
                    <div class="time-secondary">{{ formatRelativeTime(row.created_at) }}</div>
                  </div>
                </template>
              </el-table-column>

              <el-table-column :label="t('operateLog.duration')" min-width="110" align="center">
                <template #default="{ row }">
                  <div class="meta-cell">
                    <div class="meta-primary">{{ formatDuration(getFormDuration(row)) }}</div>
                    <div class="meta-secondary">{{ getDurationHint(row) }}</div>
                  </div>
                </template>
              </el-table-column>

              <el-table-column :label="t('operateLog.version')" min-width="120" align="center">
                <template #default="{ row }">
                  <span class="version-text">{{ row.version || '-' }}</span>
                </template>
              </el-table-column>

              <el-table-column :label="t('operateLog.actions')" width="220" align="right" fixed="right">
                <template #default="{ row }">
                  <div class="action-cell">
                    <el-button
                      v-if="row.workspace_session_id"
                      text
                      type="primary"
                      @click="openWorkspaceSession(row)"
                    >
                      {{ t('operateLog.viewSession') }}
                    </el-button>
                    <el-button text @click="openPreviewDialog(row)">
                      {{ t('operateLog.preview') }}
                    </el-button>
                    <el-button type="primary" @click="applyFormLog(asOperateLogEntry(row))">
                      {{ t('operateLog.replayForm') }}
                    </el-button>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </div>

          <div class="pagination-wrapper">
            <el-pagination
              background
              layout="total, prev, pager, next"
              :current-page="currentPage"
              :page-size="pageSize"
              :total="total"
              @current-change="handlePageChange"
            />
          </div>
        </div>
      </template>

      <template v-else>
        <div class="operate-log-toolbar">
          <el-input
            v-model="keyword"
            class="operate-log-search"
            clearable
            :placeholder="t('operateLog.keywordPlaceholder')"
            :prefix-icon="Search"
            @keyup.enter="handleSearch"
            @clear="handleSearch"
          />
          <el-select
            v-model="userFilter"
            class="operate-log-user-select"
            filterable
            remote
            reserve-keyword
            clearable
            :remote-method="searchUserOptions"
            :loading="userFilterLoading"
            :placeholder="t('operateLog.userPlaceholder')"
            @change="handleUserChange"
            @clear="handleUserChange"
          >
            <el-option
              v-for="option in userOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
          <el-select
            v-model="actionFilter"
            class="operate-log-action-select"
            :placeholder="t('operateLog.actionPlaceholder')"
            clearable
            @change="handleActionChange"
          >
            <el-option
              v-for="option in actionOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
          <el-select
            v-model="sourceFilter"
            class="operate-log-source-select"
            :placeholder="t('operateLog.sourcePlaceholder')"
            clearable
            @change="handleSourceChange"
          >
            <el-option
              v-for="option in sourceOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
          <el-button
            type="primary"
            plain
            :icon="Search"
            @click="handleSearch"
          >
            {{ t('common.search') }}
          </el-button>
        </div>

        <el-table
          v-if="logs.length > 0"
          :data="logs"
          stripe
          class="history-table table-history-table"
          :empty-text="t('operateLog.empty')"
          :row-key="getLogRowKey"
          :row-class-name="getLogRowClassName"
          :expand-row-keys="expandedLogRowKeys"
          @expand-change="handleLogExpandChange"
        >
          <el-table-column type="expand" width="40">
            <template #default="{ row }">
              <div class="table-log-details">
                <div
                  v-if="row.action === 'OnTableUpdateRow' && getChangeEntries(asOperateLogEntry(row)).length > 0"
                  class="change-list"
                >
                  <OperateLogFieldChange
                    v-for="item in getChangeEntries(asOperateLogEntry(row))"
                    :key="item.fieldCode"
                    :field-code="item.fieldCode"
                    :field-name="item.fieldName"
                    :field="item.field"
                    :old-value="item.oldValue"
                    :new-value="item.newValue"
                    :has-old-value="item.hasOldValue"
                    :empty-text="t('operateLog.emptyValue')"
                    :no-old-value-text="t('operateLog.noOldValue')"
                  />
                </div>

                <div
                  v-else-if="getValueEntries(asOperateLogEntry(row)).length > 0"
                  class="value-list"
                >
                  <div
                    v-for="item in getValueEntries(asOperateLogEntry(row))"
                    :key="item.fieldCode"
                    class="value-row"
                    :class="{ 'is-file-value': isFilesField(item.field) }"
                  >
                    <span class="value-field">{{ item.fieldName }}</span>
                    <OperateLogFieldValue
                      class="value-text"
                      :field="item.field"
                      :raw-value="item.value"
                      :field-path="item.fieldCode"
                      :empty-text="t('operateLog.emptyValue')"
                      compact
                    />
                  </div>
                </div>

                <div v-else class="text-muted">{{ getLogEmptyText(asOperateLogEntry(row)) }}</div>

                <div class="log-meta-grid">
                  <span
                    v-for="item in getLogMetaEntries(asOperateLogEntry(row))"
                    :key="`${item.label}:${item.value}`"
                  >
                    {{ item.label }}: {{ item.value }}
                  </span>
                  <span v-if="row.user_agent">UA: {{ row.user_agent }}</span>
                </div>
              </div>
            </template>
          </el-table-column>

          <el-table-column :label="t('operateLog.executedAt')" min-width="170">
            <template #default="{ row }">
              <div class="time-cell">
                <div class="time-primary">{{ formatDateTime(row.created_at) }}</div>
                <div class="time-secondary">{{ formatRelativeTime(row.created_at) }}</div>
              </div>
            </template>
          </el-table-column>

          <el-table-column :label="t('operateLog.actor')" min-width="150">
            <template #default="{ row }">
              <UserDisplay
                :user-info="getUserInfo(row.request_user)"
                :username="row.request_user"
                mode="simple"
                layout="horizontal"
                size="small"
              />
            </template>
          </el-table-column>

          <el-table-column :label="t('operateLog.action')" width="96">
            <template #default="{ row }">
              <el-tag :type="getActionTagType(row.action)" size="small" effect="light">
                {{ getActionLabel(row.action) }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column :label="t('operateLog.recordColumn')" min-width="220">
            <template #default="{ row }">
              <div class="table-record-cell">
                <div class="table-record-title">{{ getLogTitle(asOperateLogEntry(row)) }}</div>
                <div v-if="showRowIdColumn && row.row_id" class="table-record-meta">
                  {{ t('common.rowRecord', { id: row.row_id }) }}
                </div>
                <div v-if="showResourceColumn" class="table-record-path">
                  {{ row.full_code_path || '-' }}
                </div>
              </div>
            </template>
          </el-table-column>

          <el-table-column :label="t('operateLog.summary')" min-width="240" show-overflow-tooltip>
            <template #default="{ row }">
              <div class="table-summary-cell">
                <div class="table-summary-text">{{ getLogSummary(asOperateLogEntry(row)) }}</div>
              </div>
            </template>
          </el-table-column>

          <el-table-column :label="t('operateLog.result')" width="96" align="center">
            <template #default="{ row }">
              <el-tag :type="getLogStatusTagType(asOperateLogEntry(row))" size="small" effect="light">
                {{ getLogStatusLabel(asOperateLogEntry(row)) }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column :label="t('operateLog.source')" width="110" align="center">
            <template #default="{ row }">
              <el-tag :type="getSourceTagType(row.source)" size="small" effect="light">
                {{ getLogSourceLabel(asOperateLogEntry(row)) }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column :label="t('operateLog.executor')" width="118" align="center">
            <template #default="{ row }">
              <el-tag :type="getExecutorTagType(row.executor_type)" size="small" effect="light">
                {{ getExecutorLabel(row.executor_type) }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column :label="t('operateLog.workspaceSession')" width="118" align="center">
            <template #default="{ row }">
              <el-button
                v-if="row.workspace_session_id"
                text
                type="primary"
                @click="openWorkspaceSession(row)"
              >
                {{ t('operateLog.viewSession') }}
              </el-button>
              <span v-else class="text-muted">-</span>
            </template>
          </el-table-column>

          <el-table-column :label="t('operateLog.version')" width="96" align="center">
            <template #default="{ row }">
              <span class="version-text">{{ row.version || '-' }}</span>
            </template>
          </el-table-column>

          <el-table-column :label="t('operateLog.duration')" width="110" align="center">
            <template #default="{ row }">
              <span class="duration-text">{{ formatDuration(getLogDuration(asOperateLogEntry(row))) }}</span>
            </template>
          </el-table-column>
        </el-table>
      <el-empty v-else :description="t('operateLog.empty')" :image-size="80" />

      <div class="operate-log-pagination">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :current-page="currentPage"
          :page-size="pageSize"
          :total="total"
          @current-change="handlePageChange"
        />
      </div>
      </template>
        </div>
    <el-dialog
      v-model="previewDialogVisible"
      :title="t('operateLog.previewTitle')"
      width="820px"
      :close-on-click-modal="false"
      class="preview-dialog"
    >
      <template v-if="previewLog">
        <div class="preview-summary">
          <div class="preview-summary-main">
            <el-tag :type="getFormStatusTagType(previewLog)" effect="light" round>
              {{ getFormStatusLabel(previewLog) }}
            </el-tag>
            <span class="preview-summary-text">{{ getFormResultMessage(previewLog) }}</span>
          </div>
          <div class="preview-summary-meta">
            <span>{{ t('operateLog.executedAt') }}: {{ formatDateTime(previewLog.created_at) }}</span>
            <span>{{ t('operateLog.duration') }}: {{ formatDuration(getFormDuration(previewLog)) }}</span>
          </div>
        </div>

        <el-tabs v-model="previewActiveTab" class="preview-tabs">
          <el-tab-pane :label="t('operateLog.requestPayload')" name="request">
            <div class="preview-tab-intro">
              {{ t('operateLog.requestFieldCount', { count: getRequestFieldCount(previewLog) }) }}
            </div>
            <el-input
              :model-value="previewRequestContent"
              type="textarea"
              :rows="16"
              readonly
              class="preview-json-input"
            />
          </el-tab-pane>
          <el-tab-pane :label="t('operateLog.responsePayload')" name="response">
            <div class="preview-tab-intro">
              {{ t('operateLog.responseReplayHint') }}
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
          <el-button @click="previewDialogVisible = false">{{ t('common.cancel') }}</el-button>
          <el-button type="primary" @click="handlePreviewApply">{{ t('operateLog.replayForm') }}</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, toRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Clock, Refresh, Search } from '@element-plus/icons-vue'

import {
  ElButton,
  ElDialog,
  ElDivider,
  ElEmpty,
  ElIcon,
  ElInput,
  ElOption,
  ElPagination,
  ElSelect,
  ElTable,
  ElTableColumn,
  ElTabPane,
  ElTabs,
  ElTag
} from 'element-plus'
import type { TagProps } from 'element-plus'
import { WidgetType } from '@/architecture/domain/constants/widget'
import UserDisplay from '@/architecture/presentation/shared/components/UserDisplay.vue'
import {
  useOperateLogSection,
  type OperateLogEntry,
} from '@/architecture/presentation/composables/useOperateLogSection'
import OperateLogFieldValue from './OperateLogFieldValue.vue'
import OperateLogFieldChange from './OperateLogFieldChange.vue'
import {
  buildOperateLogRoute,
  buildWorkspaceSessionRoute,
  PLATFORM_LOG_ID_QUERY_KEY,
  PLATFORM_TRACE_ID_QUERY_KEY,
  readStringQuery,
} from '@/architecture/shared/routing/platformRouteParams'

interface Props {
  fullCodePath: string
  rowId: number
  functionDetail?: any
  autoLoad?: boolean
  scope?: 'row' | 'function' | 'directory'
  embedded?: boolean
  showRefresh?: boolean
  title?: string
  onApplyFormLog?: (requestBody: Record<string, any>, responseBody: Record<string, any> | null) => void
}

const props = withDefaults(defineProps<Props>(), {
  fullCodePath: '',
  rowId: 0,
  functionDetail: undefined,
  autoLoad: false,
  scope: 'row',
  embedded: false,
  showRefresh: false,
  title: ''
})

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const previewDialogVisible = ref(false)
const previewActiveTab = ref('request')
const previewLog = ref<any | null>(null)

function asOperateLogEntry(row: Record<string, unknown>): OperateLogEntry {
  return row as OperateLogEntry
}

const focusLogId = computed(() => readStringQuery(route.query, PLATFORM_LOG_ID_QUERY_KEY))
const focusTraceId = computed(() => readStringQuery(route.query, PLATFORM_TRACE_ID_QUERY_KEY))

const {
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
  expandedLogRowKeys,
  getUserInfo,
  getActionTagType,
  getActionLabel,
  getLogSourceLabel,
  getSourceTagType,
  getExecutorLabel,
  getExecutorTagType,
  getChangeEntries,
  getValueEntries,
  getFormRequestEntries,
  getLogTitle,
  getLogEmptyText,
  getLogSummary,
  getLogDuration,
  getLogStatusLabel,
  getLogStatusTagType,
  getLogMetaEntries,
  applyFormLog,
  getLogRowKey,
  getLogRowClassName,
  handleLogExpandChange,
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
} = useOperateLogSection({
  fullCodePath: toRef(props, 'fullCodePath'),
  rowId: toRef(props, 'rowId'),
  functionDetail: toRef(props, 'functionDetail'),
  autoLoad: toRef(props, 'autoLoad'),
  scope: toRef(props, 'scope'),
  focusLogId,
  focusTraceId,
  onApplyFormLog: props.onApplyFormLog,
})

function parseMaybeJSON(value: unknown): any {
  if (typeof value === 'string') {
    try {
      return JSON.parse(value)
    } catch {
      return value
    }
  }
  return value
}

function stringifyPretty(value: unknown): string {
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

function getObjectPayload(value: unknown): Record<string, any> | null {
  const parsed = parseMaybeJSON(value)
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    return null
  }
  return parsed as Record<string, any>
}

function getFormRequestBody(log: any): Record<string, any> | null {
  return getObjectPayload(log.old_values_json ?? log.updates)
}

function getFormResponseBody(log: any): Record<string, any> | null {
  return getObjectPayload(log.new_values_json)
}

function getRequestFieldCount(log: any): number {
  return Object.keys(getFormRequestBody(log) || {}).length
}

function readNumber(value: unknown): number | null {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string' && value.trim() !== '' && !Number.isNaN(Number(value))) return Number(value)
  return null
}

function getFormDuration(log: any): number | null {
  return readNumber(log.details_json?.duration_millis ?? getFormResponseBody(log)?.total_cost_mill)
}

function formatDuration(value: number | null): string {
  if (value === null || value < 0) return t('operateLog.durationMissing')
  if (value < 1000) return `${value}ms`
  if (value < 60000) return `${(value / 1000).toFixed(value < 10000 ? 2 : 1)}s`
  const minutes = Math.floor(value / 60000)
  const seconds = ((value % 60000) / 1000).toFixed(1)
  return `${minutes}m ${seconds}s`
}

function getDurationHint(log: any): string {
  return getFormDuration(log) === null ? t('operateLog.durationMissingHint') : t('operateLog.durationHint')
}

function getFormStatusTagType(log: any): TagProps['type'] {
  return log.status === 'failed' ? 'danger' : 'success'
}

function getFormStatusLabel(log: any): string {
  return log.status === 'failed' ? t('operateLog.failed') : t('operateLog.success')
}

function getFormResultMessage(log: any): string {
  const payload = getFormResponseBody(log)
  return payload?.msg || payload?.error || log.summary || (log.status === 'failed' ? t('operateLog.formSubmitFailed') : t('operateLog.formSubmitted'))
}

function getFormResultSummary(log: any): string {
  const payload = getFormResponseBody(log)
  const result = payload?.result
  if (Array.isArray(result)) return t('operateLog.resultArray', { count: result.length })
  if (result && typeof result === 'object') return t('operateLog.resultFields', { count: Object.keys(result).length })
  if (result !== undefined && result !== null && result !== '') return t('operateLog.resultValue', { value: String(result) })
  return log.status === 'failed' ? t('operateLog.responseErrorHint') : t('operateLog.responseCompleteHint')
}

function isFilesField(field: any): boolean {
  return field?.widget?.type === WidgetType.FILES
}

function openPreviewDialog(log: any) {
  syncOperateLogRoute(log)
  previewLog.value = log
  previewActiveTab.value = 'request'
  previewDialogVisible.value = true
}

function syncOperateLogRoute(log: any) {
  if (!props.fullCodePath || !log?.id) return
  const target = buildOperateLogRoute({
    fullCodePath: props.fullCodePath,
    logId: log.id,
    traceId: log.trace_id,
    sourcePath: log.full_code_path || props.fullCodePath,
  })
  router.replace({
    path: route.path,
    query: {
      ...route.query,
      ...target.query,
    },
  })
}

async function openWorkspaceSession(log: any) {
  const sessionId = String(log?.workspace_session_id || '').trim()
  if (!sessionId) return
  const fullCodePath = log.full_code_path || props.fullCodePath
  const target = buildWorkspaceSessionRoute({
    fullCodePath,
    sessionId,
    messageId: log.workspace_message_id,
    sourceName: log.workspace_session_title || t('operateLog.workspaceSession'),
    sourcePath: log.full_code_path || props.fullCodePath,
    traceId: log.trace_id,
  })
  const opened = window.open(router.resolve(target).href, '_blank')
  if (opened) {
    opened.opener = null
    return
  }
  await router.push(target)
}

function handlePreviewApply() {
  if (!previewLog.value) return
  applyFormLog(previewLog.value)
  previewDialogVisible.value = false
}

const previewRequestContent = computed(() =>
  previewLog.value ? stringifyPretty(getFormRequestBody(previewLog.value)) : '{}'
)

const previewResponseContent = computed(() =>
  previewLog.value ? stringifyPretty(getFormResponseBody(previewLog.value)) : '{}'
)

defineExpose({
  load
})
</script>

<style scoped>
.operate-log-section {
  margin-top: 24px;
}

.operate-log-section.is-embedded {
  height: 100%;
  margin-top: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.operate-log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.operate-log-title-group {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.operate-log-icon {
  font-size: 18px;
  color: var(--el-color-primary);
}

.operate-log-title {
  flex: 1;
}

.operate-log-content {
  margin-top: 12px;
  min-height: 0;
}

.operate-log-section.is-embedded .operate-log-content {
  flex: 1;
  overflow: auto;
  margin-top: 0;
}

.operate-log-toolbar {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) minmax(150px, 200px) minmax(140px, 170px) minmax(130px, 160px) auto;
  align-items: center;
  gap: 10px;
  margin-bottom: 16px;
  padding: 12px;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 12px;
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color));
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight, rgba(255, 255, 255, 0.65));
}

.operate-log-search {
  min-width: 0;
}

.operate-log-action-select,
.operate-log-source-select,
.operate-log-user-select {
  width: 100%;
  min-width: 0;
}

.operate-log-toolbar > .el-button {
  min-width: 86px;
}

.form-operate-log-section {
  min-height: 0;
}

.history-card {
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-bg-color);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.04);
}

.section-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px 14px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  background: var(--el-fill-color-blank);
}

.section-heading {
  min-width: 0;
}

.section-title {
  font-size: 15px;
  font-weight: 700;
  line-height: 1.4;
  color: var(--el-text-color-primary);
}

.section-subtitle {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.form-history-toolbar {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) minmax(150px, 200px) minmax(140px, 170px) minmax(130px, 160px) auto;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color));
}

.history-search {
  min-width: 0;
}

.history-action-select,
.history-source-select,
.history-user-select {
  width: 100%;
  min-width: 0;
}

.form-history-toolbar > .el-button {
  min-width: 86px;
}

.operate-log-toolbar :deep(.el-input__wrapper),
.operate-log-toolbar :deep(.el-select__wrapper),
.form-history-toolbar :deep(.el-input__wrapper),
.form-history-toolbar :deep(.el-select__wrapper) {
  min-height: 36px;
  border-radius: 10px;
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color));
  box-shadow: 0 0 0 1px var(--app-shell-panel-border, var(--el-border-color-light)) inset;
  transition: box-shadow 0.18s ease, background-color 0.18s ease;
}

.operate-log-toolbar :deep(.el-input__wrapper:hover),
.operate-log-toolbar :deep(.el-select__wrapper:hover),
.form-history-toolbar :deep(.el-input__wrapper:hover),
.form-history-toolbar :deep(.el-select__wrapper:hover) {
  box-shadow: 0 0 0 1px rgba(var(--el-color-primary-rgb), 0.28) inset;
}

.operate-log-toolbar :deep(.el-input__wrapper.is-focus),
.operate-log-toolbar :deep(.el-select__wrapper.is-focused),
.form-history-toolbar :deep(.el-input__wrapper.is-focus),
.form-history-toolbar :deep(.el-select__wrapper.is-focused) {
  box-shadow: 0 0 0 1px rgba(var(--el-color-primary-rgb), 0.45) inset, 0 0 0 3px rgba(var(--el-color-primary-rgb), 0.1);
}

.operate-log-toolbar :deep(.el-input__inner),
.operate-log-toolbar :deep(.el-select__placeholder),
.operate-log-toolbar :deep(.el-select__selected-item),
.form-history-toolbar :deep(.el-input__inner),
.form-history-toolbar :deep(.el-select__placeholder),
.form-history-toolbar :deep(.el-select__selected-item) {
  font-size: 13px;
}

.operate-log-toolbar > .el-button,
.form-history-toolbar > .el-button {
  height: 36px;
  border-radius: 10px;
  font-weight: 600;
  box-shadow: none;
}

.history-table {
  width: 100%;
}

.history-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.history-table :deep(.el-table__header th.el-table__cell) {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-weight: 700;
}

.history-table :deep(.el-table__cell) {
  vertical-align: top;
}

.history-table :deep(.cell) {
  padding-top: 8px;
  padding-bottom: 8px;
}

.history-table :deep(.el-table__row.is-focused-log > td.el-table__cell) {
  background: color-mix(in srgb, var(--el-color-primary) 9%, var(--el-bg-color));
}

.table-history-table {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
}

.table-history-table :deep(.el-table__expanded-cell) {
  padding: 0;
  background: var(--el-fill-color-lighter);
}

.form-history-table {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
}

.form-history-table :deep(.el-table__expanded-cell) {
  padding: 0;
  background: var(--el-fill-color-lighter);
}

.table-log-details {
  padding: 10px 16px 12px 56px;
}

.table-record-cell,
.table-summary-cell {
  min-width: 0;
}

.table-record-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--el-text-color-primary);
}

.table-record-meta {
  margin-top: 2px;
  font-size: 12px;
  line-height: 1.3;
  color: var(--el-text-color-secondary);
}

.table-record-path {
  margin-top: 2px;
  max-width: 100%;
  overflow: hidden;
  color: var(--el-text-color-placeholder);
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.table-summary-text {
  display: block;
  overflow: hidden;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.duration-text {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.result-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.result-copy {
  min-width: 0;
}

.result-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.35;
  word-break: break-word;
}

.result-subtitle {
  margin-top: 2px;
  font-size: 12px;
  line-height: 1.35;
  color: var(--el-text-color-secondary);
  word-break: break-word;
}

.time-cell,
.meta-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.time-primary,
.meta-primary {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.3;
}

.time-secondary,
.meta-secondary {
  font-size: 12px;
  line-height: 1.3;
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

.preview-summary {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  margin-bottom: 16px;
}

.preview-summary-main {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.preview-summary-text {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  word-break: break-word;
}

.preview-summary-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
  white-space: nowrap;
}

.preview-tab-intro {
  margin-bottom: 10px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.preview-json-input :deep(.el-textarea__inner) {
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.55;
}

.preview-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.operate-log-timeline {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.log-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
  box-shadow: 0 8px 18px rgba(15, 23, 42, 0.04);
}

.log-card-main {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  gap: 10px;
  padding: 14px 16px 16px 12px;
}

.log-card-marker {
  position: relative;
  display: flex;
  justify-content: center;
  padding-top: 5px;
}

.marker-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--el-color-info);
  box-shadow: 0 0 0 4px var(--el-fill-color-light);
}

.marker-dot.is-success {
  background: var(--el-color-success);
  box-shadow: 0 0 0 4px rgba(103, 194, 58, 0.12);
}

.marker-dot.is-warning {
  background: var(--el-color-warning);
  box-shadow: 0 0 0 4px rgba(230, 162, 60, 0.14);
}

.marker-dot.is-danger {
  background: var(--el-color-danger);
  box-shadow: 0 0 0 4px rgba(245, 108, 108, 0.13);
}

.log-card-body {
  min-width: 0;
}

.log-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
}

.log-title-wrap {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
}

.log-title {
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.45;
}

.log-time {
  flex: 0 0 auto;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  line-height: 1.35;
  white-space: nowrap;
}

.log-time span:first-child {
  color: var(--el-text-color-secondary);
  font-weight: 600;
}

.log-actor-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.log-chip {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.log-resource {
  margin-top: 10px;
  padding: 8px 10px;
  border-radius: 6px;
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-secondary);
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.45;
  word-break: break-all;
}

.log-summary {
  margin-top: 10px;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.6;
  word-break: break-word;
}

.log-detail-block {
  margin-top: 12px;
  padding-top: 2px;
}

.change-list,
.value-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 12px;
}

.change-row,
.value-row {
  display: grid;
  grid-template-columns: minmax(96px, 160px) minmax(0, 1fr);
  gap: 10px;
  align-items: start;
  min-width: 0;
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
  background: var(--el-fill-color-blank);
}

.value-row.is-file-value {
  grid-template-columns: 1fr;
  gap: 7px;
  padding: 10px;
}

.change-field,
.value-field {
  color: var(--el-text-color-regular);
  font-size: 13px;
  font-weight: 700;
  line-height: 24px;
  word-break: break-word;
}

.value-row.is-file-value .value-field {
  line-height: 1.35;
}

.change-values {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 13px;
  line-height: 24px;
}

.change-old,
.change-new,
.value-text {
  min-width: 0;
  max-width: 100%;
  word-break: break-word;
}

.change-old {
  color: var(--el-text-color-secondary);
  text-decoration: line-through;
  text-decoration-thickness: 1px;
  text-decoration-color: rgba(245, 108, 108, 0.5);
}

.change-old.is-empty {
  text-decoration: none;
  color: var(--el-text-color-placeholder);
}

.change-arrow {
  color: var(--el-text-color-placeholder);
  font-weight: 700;
}

.change-new {
  color: var(--el-text-color-primary);
  font-weight: 600;
}

.value-text {
  color: var(--el-text-color-primary);
  font-size: 13px;
  line-height: 24px;
}

.value-row.is-file-value .value-text {
  line-height: 1.5;
}

.value-row.is-file-value .value-text :deep(.files-widget),
.value-row.is-file-value .value-text :deep(.detail-files),
.value-row.is-file-value .value-text :deep(.uploaded-files),
.value-row.is-file-value .value-text :deep(.files-list) {
  width: 100%;
  max-width: 100%;
}

.log-meta-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 10px;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
  word-break: break-all;
}

.log-card-foot {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}

.operate-log-pagination,
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 14px;
}

.text-muted {
  display: block;
  margin-top: 12px;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

@media (max-width: 720px) {
  .operate-log-toolbar,
  .form-history-toolbar {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  }

  .operate-log-search,
  .history-search {
    grid-column: 1 / -1;
  }

  .operate-log-toolbar > .el-button,
  .form-history-toolbar > .el-button {
    width: 100%;
  }

  .log-card-head {
    flex-direction: column;
  }

  .log-time {
    align-items: flex-start;
  }

  .change-row,
  .value-row {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 520px) {
  .operate-log-toolbar,
  .form-history-toolbar {
    grid-template-columns: 1fr;
  }
}
</style>
