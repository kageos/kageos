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
                placeholder="搜索 Trace ID、版本、错误、请求或响应内容"
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
            <el-table-column label="执行时间" min-width="180">
              <template #default="{ row }">
                <div class="clickable-cell time-cell">
                  <div class="time-primary">{{ formatDateTime(row.created_at) }}</div>
                  <div class="time-secondary">{{ formatRelativeTime(row.created_at) }}</div>
                </div>
              </template>
            </el-table-column>

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

            <el-table-column label="耗时" width="120" align="center">
              <template #default="{ row }">
                <div class="clickable-cell meta-cell">
                  <ExecutionDurationTag :duration="getDuration(row)" placeholder="未记录" />
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
            <span>{{ formatExecutionDuration(getDuration(previewLog)) }}</span>
          </div>
        </div>

        <div class="preview-overview-grid">
          <div class="overview-item">
            <div class="overview-label">执行用户</div>
            <div class="overview-value">
              <UserDisplay
                v-if="previewLog.request_user"
                :user-info="getUserInfo(previewLog.request_user)"
                :username="previewLog.request_user"
                mode="card"
                layout="horizontal"
                size="small"
              />
              <span v-else>-</span>
            </div>
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
            <div class="overview-value">
              <ExecutionDurationTag :duration="getDuration(previewLog)" placeholder="未记录" />
            </div>
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
              <div class="preview-panel-title">输入参数</div>
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
              <div class="preview-panel-title">输出结果</div>
              <div class="preview-panel-desc">
                会一起回填输出参数和执行信息，方便继续调试或比对结果。
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
import { toRef } from 'vue'
import { Clock, WarningFilled } from '@element-plus/icons-vue'
import {
  ElButton,
  ElCard,
  ElDialog,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElOption,
  ElPagination,
  ElSelect,
  ElTable,
  ElTableColumn,
  ElTag,
  ElTooltip
} from 'element-plus'
import type { FunctionDetail } from '@/architecture/domain/types'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import UserPickerDialog from '@/shared/components/UserPickerDialog.vue'
import ExecutionDurationTag from '@/architecture/presentation/components/ExecutionDurationTag.vue'
import {
  useFormOperateLogSection,
  type ApplyOperateLogPayload
} from '@/architecture/presentation/composables/useFormOperateLogSection'

interface Props {
  fullCodePath: string
  functionDetail?: FunctionDetail | null
  autoLoad?: boolean
}
const props = withDefaults(defineProps<Props>(), {
  fullCodePath: '',
  functionDetail: null,
  autoLoad: true
})

const emit = defineEmits<{
  (e: 'apply-log', payload: ApplyOperateLogPayload): void
}>()

const {
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
} = useFormOperateLogSection({
  fullCodePath: toRef(props, 'fullCodePath'),
  functionDetail: toRef(props, 'functionDetail'),
  autoLoad: toRef(props, 'autoLoad'),
  emitApplyLog: (payload) => emit('apply-log', payload)
})

defineExpose({
  loadLogs,
  openWithFilters
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
