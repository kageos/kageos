<template>
  <div class="scheduled-task-list" v-loading="loading">
    <div class="section-header">
      <div class="section-copy">
        <div class="section-title">定时任务</div>
        <div class="section-desc">查看当前函数及子路径下的调度任务，并追踪每次执行结果。</div>
      </div>
      <div class="section-actions">
        <span class="section-total">共 {{ resourceTotal }} 个任务</span>
        <el-button type="primary" @click="loadList">刷新</el-button>
      </div>
    </div>

    <div class="filter-section">
      <el-form :inline="true" :model="filterForm" class="filter-form">
        <el-form-item label="状态">
          <el-select
            v-model="filterForm.status"
            placeholder="全部状态"
            clearable
            style="width: 160px"
            @change="handleFilterChange"
          >
            <el-option label="全部状态" value="" />
            <el-option label="待执行" value="pending" />
            <el-option label="已完成" value="done" />
            <el-option label="失败" value="failed" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button @click="resetFilters">重置</el-button>
        </el-form-item>
      </el-form>
      <div class="filter-summary">
        <span>当前仅展示当前节点及其子路径下的任务</span>
        <span v-if="filterForm.status">筛选后 {{ total }} 条</span>
      </div>
    </div>

    <div class="table-section">
      <el-empty
        v-if="!loading && list.length === 0"
        :description="filterForm.status ? '当前筛选条件下暂无定时任务' : '暂无定时任务'"
      />
      <el-table
        v-else
        :data="list"
        stripe
        style="width: 100%"
        class="task-table"
        @row-click="handleTaskRowClick"
      >
        <el-table-column prop="name" label="任务名称" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="task-name">{{ row.name || '未命名任务' }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="full_code_path" label="函数路径" min-width="260" show-overflow-tooltip />

        <el-table-column label="动作" width="110">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ actionLabel(row.action) }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="调度方式" width="140">
          <template #default="{ row }">
            <div class="schedule-cell">
              <el-tag size="small">{{ scheduleTypeLabel(row.schedule_type) }}</el-tag>
              <el-tooltip :content="getScheduleSummary(row)" placement="top" effect="light">
                <span class="schedule-summary-trigger">说明</span>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="request_user" label="执行身份" width="140" show-overflow-tooltip>
          <template #default="{ row }">
            <UserDisplay
              :username="row.request_user || row.created_by || null"
              mode="card"
              layout="horizontal"
              size="small"
            />
          </template>
        </el-table-column>

        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <div class="status-cell">
              <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
              <el-tooltip
                v-if="row.error_message"
                :content="row.error_message"
                placement="top"
                effect="light"
              >
                <span class="error-dot" />
              </el-tooltip>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="下次执行" width="180">
          <template #default="{ row }">
            {{ row.next_run_at ? formatDateTime(row.next_run_at) : '-' }}
          </template>
        </el-table-column>

        <el-table-column prop="run_count" label="已执行" width="90" />

        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>

        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click.stop="openTaskDetail(row)">
              详情
            </el-button>
            <el-button type="primary" link size="small" @click.stop="openExecutions(row)">
              执行记录
            </el-button>
            <el-button
              v-if="canCancelTask(row)"
              type="danger"
              link
              size="small"
              @click.stop="handleCancel(row)"
            >
              取消
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-pagination
      v-if="total > 0"
      v-model:current-page="page"
      v-model:page-size="pageSize"
      :total="total"
      :page-sizes="[10, 20, 50]"
      layout="total, sizes, prev, pager, next"
      class="list-pagination"
      @current-change="loadList"
      @size-change="handlePageSizeChange"
    />

    <el-dialog
      v-model="taskDetailVisible"
      :title="taskDetailTitle"
      width="960px"
      destroy-on-close
    >
      <template v-if="currentTask">
        <div class="detail-overview">
          <div class="overview-item">
            <span class="overview-label">任务名称</span>
            <span class="overview-value">{{ currentTask.name || '-' }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">函数路径</span>
            <span class="overview-value">{{ currentTask.full_code_path }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">执行动作</span>
            <span class="overview-value">{{ actionLabel(currentTask.action) }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">请求方法</span>
            <span class="overview-value">{{ currentTask.method || 'POST' }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">调度方式</span>
            <span class="overview-value">
              {{ scheduleTypeLabel(currentTask.schedule_type) }} / {{ getScheduleSummary(currentTask) }}
            </span>
          </div>
          <div class="overview-item">
            <span class="overview-label">执行身份</span>
            <span class="overview-value">
              <UserDisplay
                :username="currentTask.request_user || currentTask.created_by || null"
                mode="card"
                layout="horizontal"
                size="small"
              />
            </span>
          </div>
          <div class="overview-item">
            <span class="overview-label">创建者</span>
            <span class="overview-value">{{ currentTask.created_by || '-' }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">当前状态</span>
            <span class="overview-value">
              <el-tag :type="statusTagType(currentTask.status)" size="small">
                {{ statusLabel(currentTask.status) }}
              </el-tag>
            </span>
          </div>
          <div class="overview-item">
            <span class="overview-label">{{ runAtLabel(currentTask) }}</span>
            <span class="overview-value">{{ formatDateTime(currentTask.run_at) }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">下次执行</span>
            <span class="overview-value">
              {{ currentTask.next_run_at ? formatDateTime(currentTask.next_run_at) : '-' }}
            </span>
          </div>
          <div class="overview-item">
            <span class="overview-label">时区</span>
            <span class="overview-value">{{ currentTask.timezone || '-' }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">已执行次数</span>
            <span class="overview-value">{{ currentTask.run_count || 0 }}</span>
          </div>
        </div>

        <el-alert
          v-if="currentTask.error_message"
          title="最近一次失败原因"
          :description="currentTask.error_message"
          type="warning"
          show-icon
          :closable="false"
          class="detail-alert"
        />

        <div class="payload-section">
          <div class="payload-header">
            <span class="payload-title">任务输入参数</span>
            <span class="payload-tip">创建任务时保存的输入载荷</span>
          </div>
          <pre class="payload-pre">{{ formatPayload(currentTask.payload) }}</pre>
        </div>
      </template>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="taskDetailVisible = false">关闭</el-button>
          <el-button type="primary" @click="openExecutionsFromDetail">查看执行记录</el-button>
          <el-button
            v-if="canCancelTask(currentTask)"
            type="danger"
            @click="handleCancelFromDetail"
          >
            取消任务
          </el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog
      v-model="executionsVisible"
      :title="`执行记录：${currentTaskName}`"
      width="88%"
      destroy-on-close
    >
      <div class="execution-toolbar">
        <el-form :inline="true" :model="executionFilterForm" class="filter-form">
          <el-form-item label="状态">
            <el-select
              v-model="executionFilterForm.status"
              placeholder="全部状态"
              clearable
              style="width: 160px"
              @change="handleExecutionFilterChange"
            >
              <el-option label="全部状态" value="" />
              <el-option label="成功" value="success" />
              <el-option label="失败" value="failed" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button @click="loadExecutions">刷新</el-button>
          </el-form-item>
        </el-form>
        <span class="section-total">共 {{ executionsTotal }} 条记录</span>
      </div>

      <el-empty
        v-if="!executionsLoading && executions.length === 0"
        :description="executionFilterForm.status ? '当前筛选条件下暂无执行记录' : '暂无执行记录'"
      />
      <el-table
        v-else
        :data="executions"
        stripe
        class="execution-table"
        v-loading="executionsLoading"
        @row-click="handleExecutionRowClick"
      >
        <el-table-column prop="executed_at" label="执行时间" width="180">
          <template #default="{ row }">{{ formatDateTime(row.executed_at) }}</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
              {{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="error_message" label="错误信息" min-width="260" show-overflow-tooltip>
          <template #default="{ row }">{{ row.error_message || '-' }}</template>
        </el-table-column>
        <el-table-column label="耗时" width="120" align="center">
          <template #default="{ row }">
            <ExecutionDurationTag :duration="getExecutionDuration(row)" />
          </template>
        </el-table-column>
        <el-table-column prop="trace_id" label="Trace ID" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">{{ row.trace_id || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click.stop="openExecutionDetail(row)">
              详情
            </el-button>
            <el-button
              v-if="canOpenFunctionOperateLog(currentExecutionTask)"
              type="primary"
              link
              size="small"
              @click.stop="openFunctionOperateLog(row)"
            >
              函数记录
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        v-if="executionsTotal > 0"
        v-model:current-page="executionsPage"
        v-model:page-size="executionsPageSize"
        :total="executionsTotal"
        layout="total, prev, pager, next"
        class="executions-pagination"
        @current-change="loadExecutions"
        @size-change="handleExecutionPageSizeChange"
      />
    </el-dialog>

    <el-dialog
      v-model="executionDetailVisible"
      title="执行详情"
      width="720px"
      destroy-on-close
    >
      <template v-if="currentExecution">
        <div class="detail-overview execution-overview">
          <div class="overview-item">
            <span class="overview-label">执行时间</span>
            <span class="overview-value">{{ formatDateTime(currentExecution.executed_at) }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">执行状态</span>
            <span class="overview-value">
              <el-tag :type="currentExecution.status === 'success' ? 'success' : 'danger'" size="small">
                {{ statusLabel(currentExecution.status) }}
              </el-tag>
            </span>
          </div>
          <div class="overview-item">
            <span class="overview-label">Trace ID</span>
            <span class="overview-value">{{ currentExecution.trace_id || '-' }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">耗时</span>
            <span class="overview-value">
              <ExecutionDurationTag :duration="getExecutionDuration(currentExecution)" />
            </span>
          </div>
        </div>

        <el-alert
          v-if="currentExecution.error_message"
          title="执行失败"
          :description="currentExecution.error_message"
          type="error"
          show-icon
          :closable="false"
          class="detail-alert"
        />
      </template>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="executionDetailVisible = false">关闭</el-button>
          <el-button
            v-if="currentExecution && canOpenFunctionOperateLog(currentExecutionTask)"
            type="primary"
            @click="openFunctionOperateLog(currentExecution)"
          >
            查看函数执行记录
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { withDefaults } from 'vue'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import ExecutionDurationTag from '@/architecture/presentation/components/ExecutionDurationTag.vue'
import { useScheduledTaskList } from '@/architecture/presentation/composables/useScheduledTaskList'

const props = withDefaults(
  defineProps<{
    resourcePath?: string
    autoLoad?: boolean
  }>(),
  { autoLoad: false }
)
const emit = defineEmits<{
  (e: 'total-change', total: number): void
  (e: 'open-function-operate-log', payload: { source: 'scheduled_task'; traceId?: string }): void
}>()
const {
  loading,
  list,
  total,
  resourceTotal,
  page,
  pageSize,
  filterForm,
  taskDetailVisible,
  currentTask,
  taskDetailTitle,
  executionsVisible,
  executions,
  executionsTotal,
  executionsPage,
  executionsPageSize,
  executionsLoading,
  currentTaskName,
  currentExecutionTask,
  executionFilterForm,
  executionDetailVisible,
  currentExecution,
  scheduleTypeLabel,
  actionLabel,
  statusTagType,
  statusLabel,
  formatDateTime,
  formatPayload,
  getScheduleSummary,
  runAtLabel,
  getExecutionDuration,
  loadList,
  handleFilterChange,
  handlePageSizeChange,
  resetFilters,
  handleTaskRowClick,
  canCancelTask,
  handleCancel,
  openTaskDetail,
  handleCancelFromDetail,
  openExecutionsFromDetail,
  openExecutions,
  loadExecutions,
  handleExecutionFilterChange,
  handleExecutionPageSizeChange,
  handleExecutionRowClick,
  openExecutionDetail,
  canOpenFunctionOperateLog,
  openFunctionOperateLog
} = useScheduledTaskList(props, emit)
</script>

<style scoped>
.scheduled-task-list {
  padding: 20px;
}

.section-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.section-copy {
  min-width: 0;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.section-desc {
  margin-top: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.section-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.section-total {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.filter-section {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px 8px;
  margin-bottom: 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-bg-color);
}

.filter-form {
  flex: 1;
}

.filter-summary {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
  white-space: nowrap;
}

.table-section {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  overflow: hidden;
  background: var(--el-bg-color);
}

.task-table :deep(.el-table__row) {
  cursor: pointer;
}

.task-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.5;
}

.schedule-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.schedule-summary-trigger {
  font-size: 12px;
  color: var(--el-color-primary);
  cursor: help;
  white-space: nowrap;
}

.status-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.error-dot {
  display: inline-flex;
  align-items: center;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--el-color-danger);
}

.list-pagination,
.executions-pagination {
  margin-top: 16px;
  justify-content: flex-end;
}

.detail-overview {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 20px;
}

.overview-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px 14px;
  border-radius: 10px;
  background: var(--el-fill-color-light);
}

.overview-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}

.overview-value {
  font-size: 13px;
  color: var(--el-text-color-primary);
  line-height: 1.6;
  word-break: break-all;
}

.detail-alert {
  margin-top: 16px;
}

.payload-section,
.payload-panel {
  margin-top: 18px;
}

.payload-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-top: 18px;
}

.payload-panel {
  min-width: 0;
}

.payload-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.payload-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.payload-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.payload-pre {
  margin: 0;
  padding: 14px 16px;
  border-radius: 10px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 52vh;
  overflow: auto;
}

.dialog-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
}

.execution-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 8px;
}

.execution-table :deep(.el-table__row) {
  cursor: pointer;
}

.execution-overview {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

@media (max-width: 960px) {
  .scheduled-task-list {
    padding: 16px;
  }

  .section-header,
  .filter-section,
  .execution-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .section-actions {
    justify-content: space-between;
  }

  .filter-summary {
    white-space: normal;
  }

  .detail-overview,
  .execution-overview,
  .payload-grid {
    grid-template-columns: 1fr;
  }
}
</style>
