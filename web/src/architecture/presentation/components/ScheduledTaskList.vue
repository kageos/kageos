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
        row-key="id"
        :preserve-expanded-content="true"
        stripe
        style="width: 100%"
        class="task-table"
        @expand-change="handleTaskExpandChange"
      >
        <el-table-column type="expand" width="48">
          <template #default="{ row }">
            <div
              class="task-execution-expand"
              v-loading="getInlineExecutionState(row.id).loading"
              @click.stop
            >
              <div class="inline-execution-panel">
                <div class="inline-execution-filter">
                  <div class="inline-execution-title">执行记录</div>
                  <el-form
                    :inline="true"
                    :model="getInlineExecutionState(row.id)"
                    class="inline-filter-form"
                  >
                    <el-form-item label="状态">
                      <el-select
                        v-model="getInlineExecutionState(row.id).status"
                        placeholder="全部状态"
                        clearable
                        size="small"
                        class="inline-status-select"
                        @change="handleInlineExecutionStatusChange(row)"
                      >
                        <el-option label="全部状态" value="" />
                        <el-option label="成功" value="success" />
                        <el-option label="失败" value="failed" />
                      </el-select>
                    </el-form-item>
                    <el-form-item>
                      <el-button
                        size="small"
                        :icon="Refresh"
                        @click.stop="refreshInlineExecutions(row)"
                      >
                        刷新
                      </el-button>
                    </el-form-item>
                  </el-form>
                </div>

                <el-alert
                  v-if="getInlineExecutionState(row.id).error"
                  :title="getInlineExecutionState(row.id).error"
                  type="error"
                  show-icon
                  :closable="false"
                  class="inline-execution-alert"
                />
                <el-empty
                  v-else-if="
                    getInlineExecutionState(row.id).loaded &&
                    getInlineExecutionState(row.id).list.length === 0
                  "
                  description="暂无执行记录"
                  :image-size="56"
                  class="inline-execution-empty"
                />
                <el-table
                  v-else-if="getInlineExecutionState(row.id).loaded"
                  :data="getInlineExecutionState(row.id).list"
                  stripe
                  size="small"
                  class="inline-execution-table"
                  @row-click="openInlineExecutionDetail(row, $event)"
                >
                  <el-table-column prop="executed_at" label="执行时间" width="180">
                    <template #default="{ row: execution }">
                      {{ formatDateTime(execution.executed_at) }}
                    </template>
                  </el-table-column>
                  <el-table-column prop="status" label="状态" width="100">
                    <template #default="{ row: execution }">
                      <el-tag :type="execution.status === 'success' ? 'success' : 'danger'" size="small">
                        {{ statusLabel(execution.status) }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column prop="error_message" label="错误信息" min-width="240" show-overflow-tooltip>
                    <template #default="{ row: execution }">{{ execution.error_message || '-' }}</template>
                  </el-table-column>
                  <el-table-column label="耗时" width="120" align="center">
                    <template #default="{ row: execution }">
                      <ExecutionDurationTag :duration="getExecutionDuration(execution)" />
                    </template>
                  </el-table-column>
                  <el-table-column prop="trace_id" label="Trace ID" min-width="180" show-overflow-tooltip>
                    <template #default="{ row: execution }">{{ execution.trace_id || '-' }}</template>
                  </el-table-column>
                  <el-table-column
                    label="操作"
                    :width="canReplayExecution ? 132 : 104"
                    align="center"
                    fixed="right"
                  >
                    <template #default="{ row: execution }">
                      <div class="table-row-actions">
                        <el-tooltip content="执行详情" placement="top" effect="light">
                          <el-button
                            class="icon-action-button"
                            type="primary"
                            text
                            :icon="View"
                            aria-label="执行详情"
                            @click.stop="openExecutionDetail(execution, row)"
                          />
                        </el-tooltip>
                        <el-tooltip
                          v-if="canReplayExecution"
                          content="回填到表单"
                          placement="top"
                          effect="light"
                        >
                          <el-button
                            class="icon-action-button"
                            type="primary"
                            text
                            :icon="RefreshLeft"
                            aria-label="回填到表单"
                            @click.stop="applyExecutionToForm(execution, row)"
                          />
                        </el-tooltip>
                        <el-tooltip
                          v-if="canOpenFunctionOperateLog(row)"
                          content="函数执行记录"
                          placement="top"
                          effect="light"
                        >
                          <el-button
                            class="icon-action-button"
                            type="primary"
                            text
                            :icon="Tickets"
                            aria-label="函数执行记录"
                            @click.stop="openFunctionOperateLog(execution)"
                          />
                        </el-tooltip>
                      </div>
                    </template>
                  </el-table-column>
                </el-table>
                <div
                  v-if="
                    getInlineExecutionState(row.id).loaded &&
                    !getInlineExecutionState(row.id).error &&
                    getInlineExecutionState(row.id).total > 0
                  "
                  class="inline-execution-footer"
                >
                  <span class="inline-execution-summary">
                    第 {{ getInlineExecutionState(row.id).page }} 页 / 共
                    {{ getInlineExecutionState(row.id).total }} 条
                  </span>
                  <el-pagination
                    v-if="getInlineExecutionState(row.id).total > getInlineExecutionState(row.id).pageSize"
                    small
                    :current-page="getInlineExecutionState(row.id).page"
                    :page-size="getInlineExecutionState(row.id).pageSize"
                    :total="getInlineExecutionState(row.id).total"
                    :pager-count="5"
                    layout="prev, pager, next"
                    class="inline-execution-pagination"
                    @current-change="handleInlineExecutionPageChange(row, $event)"
                  />
                </div>
                <div
                  v-if="!getInlineExecutionState(row.id).loaded && !getInlineExecutionState(row.id).error"
                  class="inline-execution-placeholder"
                />
              </div>
            </div>
          </template>
        </el-table-column>

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

        <el-table-column label="操作" width="168" align="center" fixed="right">
          <template #default="{ row }">
            <div class="table-row-actions">
              <el-tooltip content="任务详情" placement="top" effect="light">
                <el-button
                  class="icon-action-button"
                  type="primary"
                  text
                  :icon="View"
                  aria-label="任务详情"
                  @click.stop="openTaskDetail(row)"
                />
              </el-tooltip>
              <el-tooltip content="全部执行记录" placement="top" effect="light">
                <el-button
                  class="icon-action-button"
                  type="primary"
                  text
                  :icon="Tickets"
                  aria-label="全部执行记录"
                  @click.stop="openExecutions(row)"
                />
              </el-tooltip>
              <el-tooltip v-if="canCancelTask(row)" content="取消任务" placement="top" effect="light">
                <el-button
                  class="icon-action-button"
                  type="danger"
                  text
                  :icon="Close"
                  aria-label="取消任务"
                  @click.stop="handleCancel(row)"
                />
              </el-tooltip>
              <el-tooltip v-if="canDeleteTask(row)" content="删除任务" placement="top" effect="light">
                <el-button
                  class="icon-action-button"
                  type="danger"
                  text
                  :icon="Delete"
                  aria-label="删除任务"
                  @click.stop="handleDelete(row)"
                />
              </el-tooltip>
            </div>
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
          <div class="overview-item">
            <span class="overview-label">通知条件</span>
            <span class="overview-value">{{ notifyOnLabel(currentTask.notify_on) }}</span>
          </div>
          <div class="overview-item">
            <span class="overview-label">通知用户</span>
            <span class="overview-value">
              <span v-if="!currentTask.notify_users?.length">-</span>
              <span v-else class="notify-inline-list">
                <UserDisplay
                  v-for="username in currentTask.notify_users"
                  :key="`detail-user-${username}`"
                  :username="username"
                  mode="card"
                  layout="horizontal"
                  size="small"
                  class="notify-inline-item"
                />
              </span>
            </span>
          </div>
          <div class="overview-item">
            <span class="overview-label">通知组织</span>
            <span class="overview-value">
              <span v-if="!currentTask.notify_departments?.length">-</span>
              <span v-else class="notify-inline-list">
                <DepartmentDisplay
                  v-for="departmentPath in currentTask.notify_departments"
                  :key="`detail-dept-${departmentPath}`"
                  :full-code-path="departmentPath"
                  mode="card"
                  layout="horizontal"
                  size="small"
                  show-full-path
                  class="notify-inline-item"
                />
              </span>
            </span>
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
          <el-button type="primary" @click="openExecutionsFromDetail">查看全部记录</el-button>
          <el-button
            v-if="canCancelTask(currentTask)"
            type="danger"
            plain
            @click="handleCancelFromDetail"
          >
            取消任务
          </el-button>
          <el-button
            v-if="canDeleteTask(currentTask)"
            type="danger"
            @click="handleDeleteFromDetail"
          >
            删除任务
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
        <el-table-column
          label="操作"
          :width="canReplayExecution ? 132 : 104"
          align="center"
          fixed="right"
        >
          <template #default="{ row }">
            <div class="table-row-actions">
              <el-tooltip content="执行详情" placement="top" effect="light">
                <el-button
                  class="icon-action-button"
                  type="primary"
                  text
                  :icon="View"
                  aria-label="执行详情"
                  @click.stop="openExecutionDetail(row)"
                />
              </el-tooltip>
              <el-tooltip
                v-if="canReplayExecution"
                content="回填到表单"
                placement="top"
                effect="light"
              >
                <el-button
                  class="icon-action-button"
                  type="primary"
                  text
                  :icon="RefreshLeft"
                  aria-label="回填到表单"
                  @click.stop="applyExecutionToForm(row, currentExecutionTask)"
                />
              </el-tooltip>
              <el-tooltip
                v-if="canOpenFunctionOperateLog(currentExecutionTask)"
                content="函数执行记录"
                placement="top"
                effect="light"
              >
                <el-button
                  class="icon-action-button"
                  type="primary"
                  text
                  :icon="Tickets"
                  aria-label="函数执行记录"
                  @click.stop="openFunctionOperateLog(row)"
                />
              </el-tooltip>
            </div>
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
      width="1040px"
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
        <FunctionExecutionResultReadonly
          :function-detail="props.functionDetail"
          :request-payload="currentExecutionRequestPayload"
          :response-payload="currentExecutionResponsePayload"
          :response-metadata="currentExecutionResponseMetadata"
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
          <el-button
            v-if="currentExecution && canReplayExecution"
            type="primary"
            @click="applyExecutionToForm(currentExecution, currentExecutionTask)"
          >
            回填到表单
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Close, Delete, Refresh, RefreshLeft, Tickets, View } from '@element-plus/icons-vue'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import DepartmentDisplay from '@/shared/components/DepartmentDisplay.vue'
import ExecutionDurationTag from '@/architecture/presentation/components/ExecutionDurationTag.vue'
import FunctionExecutionResultReadonly from '@/architecture/presentation/components/FunctionExecutionResultReadonly.vue'
import { useScheduledTaskList } from '@/architecture/presentation/composables/useScheduledTaskList'
import type { FunctionDetail } from '@/architecture/domain/types'
import { TEMPLATE_TYPE } from '@/architecture/runtime/utils/functionTypes'

const props = withDefaults(
  defineProps<{
    resourcePath?: string
    autoLoad?: boolean
    functionDetail?: FunctionDetail | null
  }>(),
  { autoLoad: false, functionDetail: null }
)
const emit = defineEmits<{
  (e: 'total-change', total: number): void
  (e: 'open-function-operate-log', payload: { source: 'scheduled_task'; traceId?: string }): void
  (e: 'apply-execution', payload: {
    requestBody?: Record<string, any> | null
    responseBody?: Record<string, any> | null
    responseMetadata?: Record<string, any> | null
    replayContext?: {
      source: 'scheduled_task'
      title?: string
      taskId?: number
      executionId?: number
      traceId?: string
      executedAt?: string
    } | null
  }): void
}>()
const canReplayExecution = computed(() => props.functionDetail?.template_type === TEMPLATE_TYPE.FORM)
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
  currentExecutionRequestPayload,
  currentExecutionResponsePayload,
  currentExecutionResponseMetadata,
  scheduleTypeLabel,
  actionLabel,
  statusTagType,
  statusLabel,
  notifyOnLabel,
  formatDateTime,
  formatPayload,
  getScheduleSummary,
  runAtLabel,
  getExecutionDuration,
  loadList,
  handleFilterChange,
  handlePageSizeChange,
  resetFilters,
  canCancelTask,
  canDeleteTask,
  handleCancel,
  handleDelete,
  openTaskDetail,
  handleCancelFromDetail,
  handleDeleteFromDetail,
  openExecutionsFromDetail,
  handleTaskExpandChange,
  refreshInlineExecutions,
  handleInlineExecutionStatusChange,
  handleInlineExecutionPageChange,
  getInlineExecutionState,
  openExecutions,
  loadExecutions,
  handleExecutionFilterChange,
  handleExecutionPageSizeChange,
  handleExecutionRowClick,
  openExecutionDetail,
  openInlineExecutionDetail,
  canOpenFunctionOperateLog,
  openFunctionOperateLog,
  applyExecutionToForm
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

.task-table :deep(.el-table__expanded-cell) {
  padding: 0;
  background: var(--el-fill-color);
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

.task-execution-expand {
  position: relative;
  min-height: 72px;
  padding: 10px 14px 10px 58px;
  background: var(--el-fill-color-light);
  box-shadow: inset 0 1px 0 var(--el-border-color-lighter);
}

.task-execution-expand::before {
  content: '';
  position: absolute;
  left: 30px;
  top: 14px;
  bottom: 14px;
  width: 2px;
  border-radius: 999px;
  background: var(--el-border-color);
}

.inline-execution-panel {
  overflow: hidden;
  border: 1px solid var(--app-auth-card-border, var(--el-border-color-lighter));
  border-radius: 12px;
  background: var(--app-auth-card-bg, var(--el-bg-color));
  box-shadow: var(--app-auth-card-shadow-soft, 0 8px 24px rgba(15, 23, 42, 0.06));
}

.inline-execution-filter {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 46px;
  padding: 8px 12px;
  margin: 0;
  border-bottom: 1px solid var(--app-auth-card-border, var(--el-border-color-lighter));
  background: var(--app-auth-card-bg, var(--el-bg-color));
}

.inline-execution-title {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.inline-filter-form {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  margin-left: auto;
  flex-wrap: wrap;
}

.inline-filter-form :deep(.el-form-item) {
  margin: 0;
}

.inline-filter-form :deep(.el-form-item__label) {
  padding-right: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.inline-filter-form :deep(.el-form-item__content) {
  line-height: 1;
}

.inline-status-select {
  width: 136px;
}

.inline-filter-form :deep(.el-select__wrapper) {
  min-height: 30px;
  border-radius: 10px;
  background: var(--app-auth-input-bg, var(--el-bg-color));
  border-color: var(--app-auth-input-border, var(--el-border-color));
  box-shadow: none;
}

.inline-filter-form :deep(.el-button) {
  height: 30px;
  padding: 0 12px;
  border: 1px solid var(--app-auth-input-border, var(--el-border-color));
  border-radius: 10px;
  background: var(--app-auth-input-bg, var(--el-bg-color));
  font-weight: 600;
  box-shadow: none;
}

.inline-filter-form :deep(.el-button:hover),
.inline-filter-form :deep(.el-select__wrapper:hover) {
  border-color: rgba(var(--el-color-primary-rgb), 0.42);
  box-shadow: var(--app-auth-input-shadow-hover, none);
}

.inline-execution-summary {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.inline-execution-placeholder {
  min-height: 40px;
}

.inline-execution-alert,
.inline-execution-empty {
  background: transparent;
}

.inline-execution-alert {
  margin: 10px 12px;
}

.inline-execution-empty {
  padding: 10px 0 14px;
}

.inline-execution-table {
  --el-table-bg-color: var(--app-auth-card-bg, var(--el-bg-color));
  --el-table-tr-bg-color: var(--app-auth-card-bg, var(--el-bg-color));
  --el-table-header-bg-color: var(--app-shell-panel-muted-bg, var(--el-fill-color-lighter));
  --el-table-row-hover-bg-color: var(--el-fill-color-light);
  border-top: 0;
  border-bottom: 0;
  border-radius: 0;
  overflow: hidden;
  background: transparent;
}

.inline-execution-table :deep(.el-table__header-wrapper th.el-table__cell) {
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-lighter));
  color: var(--el-text-color-secondary);
  font-weight: 500;
}

.inline-execution-table :deep(.el-table__cell) {
  padding: 7px 0;
}

.inline-execution-table :deep(td.el-table__cell) {
  background: var(--app-auth-card-bg, var(--el-bg-color));
}

.inline-execution-table :deep(td.el-table-fixed-column--right) {
  background: var(--app-auth-card-bg, var(--el-bg-color));
}

.inline-execution-table :deep(th.el-table-fixed-column--right) {
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-lighter));
}

.inline-execution-table :deep(.el-table__row) {
  cursor: pointer;
}

.inline-execution-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 32px;
  padding: 4px 12px;
  border-top: 1px solid var(--app-auth-card-border, var(--el-border-color-lighter));
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-lighter));
}

.inline-execution-pagination {
  --el-pagination-bg-color: transparent;
  --el-pagination-button-bg-color: transparent;
}

.table-row-actions {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  width: 100%;
}

.table-row-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}

.icon-action-button {
  width: 28px;
  height: 28px;
  padding: 0;
  border-radius: 6px;
}

.icon-action-button:hover {
  background: var(--el-fill-color);
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

.notify-inline-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
}

.notify-inline-item {
  max-width: 100%;
}

.notify-inline-item :deep(.user-name),
.notify-inline-item :deep(.department-name) {
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
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

.execution-overview .overview-item {
  border: 1px solid rgba(148, 163, 184, 0.16);
  background: var(--el-fill-color-light);
}

@media (max-width: 960px) {
  .scheduled-task-list {
    padding: 16px;
  }

  .section-header,
  .filter-section,
  .inline-execution-filter,
  .execution-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .section-actions {
    justify-content: space-between;
  }

  .task-execution-expand {
    padding: 8px 8px 8px 28px;
  }

  .task-execution-expand::before {
    left: 14px;
    top: 10px;
    bottom: 10px;
  }

  .inline-filter-form {
    justify-content: flex-start;
    width: 100%;
    margin-left: 0;
  }

  .inline-status-select {
    width: 100%;
  }

  .inline-execution-title {
    flex-wrap: wrap;
  }

  .inline-execution-summary {
    width: 100%;
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
