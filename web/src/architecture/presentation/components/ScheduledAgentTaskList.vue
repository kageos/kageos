<template>
  <div class="scheduled-agent-task-list" v-loading="loading">
    <div class="scheduled-list-header">
      <div>
        <div class="scheduled-list-title">{{ t('scheduledTask.agentTitle') }}</div>
        <div class="scheduled-list-subtitle">{{ resourcePath ? `${t('scheduledTask.currentWorkspace')} · ${resourcePath}` : t('scheduledTask.noWorkspaceSelected') }}</div>
      </div>
      <div class="scheduled-list-actions">
        <span class="scheduled-total">{{ t('scheduledTask.totalCount', { count: total }) }}</span>
        <el-button :icon="Refresh" @click="handleListRefresh">{{ t('common.refresh') }}</el-button>
        <el-button type="primary" :icon="Plus" :disabled="!resourcePath" @click="handleOpenCreate">
          新建数字员工
        </el-button>
      </div>
    </div>

    <div class="scheduled-agent-workspace">
      <aside class="scheduled-agent-sidebar">
        <div class="scheduled-list-filter">
          <div class="scheduled-sidebar-title">数字员工队伍</div>
          <el-select v-model="statusFilter" :placeholder="t('scheduledTask.allStatuses')" clearable size="small" @change="handleStatusFilterChange">
            <el-option :label="t('scheduledTask.allStatuses')" value="" />
            <el-option :label="t('scheduledTask.taskStatusPending')" value="pending" />
            <el-option :label="t('scheduledTask.taskStatusPaused')" value="paused" />
            <el-option :label="t('scheduledTask.taskStatusDone')" value="done" />
            <el-option :label="t('scheduledTask.taskStatusFailed')" value="failed" />
            <el-option :label="t('scheduledTask.taskStatusCancelled')" value="cancelled" />
          </el-select>
        </div>

        <el-empty
          v-if="!loading && list.length === 0"
          :description="t('scheduledTask.emptyAgents')"
          :image-size="72"
          class="scheduled-sidebar-empty"
        />

        <div v-else class="agent-session-list" role="list">
          <button
            v-for="task in list"
            :key="task.id"
            type="button"
            :class="[
              'agent-session-item',
              `is-${task.status || 'unknown'}`,
              { 'is-active': selectedTask?.id === task.id, 'is-running': !!task.inflight_execution_id }
            ]"
            @click="selectTask(task)"
          >
            <span class="agent-session-item-avatar">
              <AgentEmployeeMascot
                variant="employee"
                :state="agentEmployeeState(task)"
                :label="`${task.title || t('scheduledTask.unnamedAgentTask')}，${agentEmployeeStatus(task)}`"
              />
            </span>
            <span class="agent-session-item-content">
              <span class="agent-session-item-head">
                <span class="agent-session-item-title">
                  {{ task.title || t('scheduledTask.unnamedAgentTask') }}
                </span>
                <el-tag :type="agentEmployeeTagType(task)" size="small" effect="light">
                  {{ agentEmployeeStatus(task) }}
                </el-tag>
              </span>
              <span class="agent-session-item-badges">
                <el-tag v-if="isBuiltinTask(task)" size="small" type="info" effect="plain">目录内置</el-tag>
                <el-tag v-else size="small" type="success" effect="plain">自定义</el-tag>
              </span>
              <span class="agent-session-item-summary">
                {{ getTaskSummary(task) }}
              </span>
              <span v-if="task.last_error_message" class="agent-session-item-error">
                {{ task.last_error_message }}
              </span>
              <span class="agent-session-item-meta">
                <span>{{ scheduleLabel(task.schedule) }}</span>
                <span>{{ t('scheduledTask.nextRun') }} {{ formatDateTime(task.next_run_at) }}</span>
                <span>{{ t('scheduledTask.runCount') }} {{ task.run_count || 0 }}</span>
              </span>
              <span v-if="isTaskPaused(task)" class="agent-session-item-hint">
                {{ t('scheduledTask.enableForUnattendedHint') }}
              </span>
            </span>
          </button>
        </div>

        <div v-if="total > pageSize" class="scheduled-pagination">
          <el-pagination
            v-model:current-page="page"
            small
            :page-size="pageSize"
            :total="total"
            layout="prev, pager, next"
            @current-change="handlePageChange"
          />
        </div>
      </aside>

      <main class="scheduled-agent-detail">
        <el-empty
          v-if="!selectedTask"
          :description="t('scheduledTask.selectSessionHint')"
          :image-size="96"
          class="scheduled-detail-empty"
        />

        <template v-else>
          <div class="scheduled-detail-shell" :class="{ 'is-editing': inlineEditing }">
            <section class="detail-document">
              <div v-if="!inlineEditing" class="detail-employee-hero">
                <div class="detail-employee-avatar">
                  <AgentEmployeeMascot
                    variant="employee"
                    :state="agentEmployeeState(selectedTask)"
                    :label="`${selectedTask.title || t('scheduledTask.unnamedAgentTask')}，${agentEmployeeStatus(selectedTask)}`"
                  />
                </div>
                <div class="detail-employee-copy">
                  <div class="detail-employee-title-row">
                    <h2>{{ selectedTask.title || t('scheduledTask.unnamedAgentTask') }}</h2>
                    <el-tag v-if="isBuiltinTask(selectedTask)" type="info" effect="plain">目录内置</el-tag>
                    <el-tag v-else type="success" effect="plain">自定义</el-tag>
                    <el-tag :type="agentEmployeeTagType(selectedTask)" effect="light">
                      {{ agentEmployeeStatus(selectedTask) }}
                    </el-tag>
                  </div>
                  <p>{{ getTaskDescription(selectedTask) || t('scheduledTask.noDescription') }}</p>
                  <div class="detail-employee-promise">看见谁在值守，也看见每一次真实执行。</div>
                </div>
                <div class="detail-edit-actions">
                  <el-button
                    v-if="isBuiltinTask(selectedTask)"
                    :icon="CopyDocument"
                    @click="handleCopyAsCustom(selectedTask)"
                  >
                    复制为自定义
                  </el-button>
                  <el-button
                    :icon="EditPen"
                    :disabled="isTerminal(selectedTask.status) || isBuiltinTask(selectedTask)"
                    @click="startInlineEdit(selectedTask)"
                  >
                    {{ t('scheduledTask.edit') }}
                  </el-button>
                  <el-button
                    type="primary"
                    :icon="VideoPlay"
                    :disabled="isTerminal(selectedTask.status)"
                    @click="handleRunNow(selectedTask)"
                  >
                    {{ t('scheduledTask.runNow') }}
                  </el-button>
                  <el-button
                    :type="selectedTask.status === 'paused' ? 'primary' : 'warning'"
                    :icon="selectedTask.status === 'paused' ? CaretRight : VideoPause"
                    :disabled="isTerminal(selectedTask.status)"
                    @click="selectedTask.status === 'paused' ? handleResume(selectedTask) : handlePause(selectedTask)"
                  >
                    {{ selectedTask.status === 'paused' ? t('scheduledTask.resume') : t('scheduledTask.pause') }}
                  </el-button>
                </div>
              </div>

              <div v-if="!inlineEditing" class="detail-employee-facts">
                <div class="detail-employee-fact">
                  <span>{{ t('scheduledTask.schedule') }}</span>
                  <strong>{{ scheduleLabel(selectedTask.schedule) }}</strong>
                </div>
                <div class="detail-employee-fact">
                  <span>{{ t('scheduledTask.nextRun') }}</span>
                  <strong>{{ formatDateTime(selectedTask.next_run_at) }}</strong>
                </div>
                <div class="detail-employee-fact">
                  <span>{{ t('scheduledTask.runCount') }}</span>
                  <strong>{{ selectedTask.run_count || 0 }} 次</strong>
                </div>
                <div class="detail-employee-fact is-path">
                  <span>工作目录</span>
                  <strong>{{ selectedTaskWorkspacePath || '-' }}</strong>
                </div>
                <div class="detail-employee-fact">
                  <span>{{ t('scheduledTask.overlapPolicy') }}</span>
                  <strong>{{ overlapPolicyLabel(selectedTask.overlap_policy) }}</strong>
                </div>
                <div class="detail-employee-fact">
                  <span>{{ t('scheduledTask.agentModel') }}</span>
                  <strong>{{ getTaskLLMConfigLabel(selectedTask) }}</strong>
                </div>
                <div class="detail-employee-fact">
                  <span>{{ t('scheduledTask.createdBy') }}</span>
                  <strong>{{ selectedTask.created_by || selectedTask.request_user || '-' }}</strong>
                </div>
              </div>

              <div v-else class="detail-document-toolbar">
                <div>
                  <div class="detail-kicker">编辑数字员工</div>
                  <div class="detail-toolbar-title">修改职责、工作说明和执行设置</div>
                </div>
                <div class="detail-edit-actions">
                  <el-button @click="cancelInlineEdit">{{ t('common.cancel') }}</el-button>
                  <el-button type="primary" :loading="inlineSaving" @click="saveInlineEdit">
                    {{ t('common.save') }}
                  </el-button>
                </div>
              </div>

              <ScheduledAgentInlineEditor
                v-if="inlineEditing"
                v-model:title="inlineForm.title"
                v-model:description="inlineForm.description"
                v-model:message="inlineForm.message"
                v-model:files="inlineForm.files"
                v-model:llm-config-id="inlineForm.llm_config_id"
                :llm-list="llmList"
                :llm-loading="llmLoading"
                :llm-option-label="llmOptionLabel"
                :full-code-path="selectedTaskWorkspacePath"
                @llm-visible-change="handleLLMSelectVisibleChange"
                @save="saveInlineEdit"
              />

              <ScheduledAgentTaskAside
                v-if="inlineEditing"
                class="is-inline-schedule"
                :task="selectedTask"
                :builtin="isBuiltinTask(selectedTask)"
                :inline-editing="true"
                :workspace-path="selectedTaskWorkspacePath"
                :llm-config-label="getTaskLLMConfigLabel(selectedTask)"
                v-model:schedule-type="inlineForm.schedule_type"
                v-model:run-at="inlineForm.run_at"
                v-model:cron-expr="inlineForm.cron_expr"
                v-model:interval-seconds="inlineForm.interval_seconds"
                v-model:max-runs="inlineForm.max_runs"
                v-model:overlap-policy="inlineForm.overlap_policy"
                v-model:max-parallelism="inlineForm.max_parallelism"
              />

              <template v-else>
                <section class="detail-document-section">
                  <div class="detail-section-head">
                    <div>
                      <div class="detail-section-title">{{ t('scheduledTask.agentDescription') }}</div>
                      <div class="detail-section-subtitle">{{ t('scheduledTask.agentDescriptionHint') }}</div>
                    </div>
                  </div>
                  <div class="detail-description-body">
                    {{ getTaskDescription(selectedTask) || t('scheduledTask.noDescription') }}
                  </div>
                </section>

                <section class="detail-document-section is-message">
                  <div class="detail-section-head">
                    <div>
                      <div class="detail-section-title">{{ t('scheduledTask.sessionMessage') }}</div>
                      <div class="detail-section-subtitle">{{ t('scheduledTask.sessionMessageHint') }}</div>
                    </div>
                  </div>
                  <div class="detail-message-body">
                    <div
                      v-if="getAgentDisplayMessage(selectedTask)"
                      class="detail-message-markdown"
                      v-html="renderAgentMessageMarkdown(selectedTask)"
                      @mouseover="showResourcePreviewFromEvent"
                      @focusin="showResourcePreviewFromEvent"
                      @focusout="scheduleCloseResourcePreview"
                      @mouseleave="scheduleCloseResourcePreview"
                      @copy="onAgentMessageMarkdownCopy"
                    />
                    <div v-else class="detail-message-empty">
                      {{ t('scheduledTask.noMessage') }}
                    </div>
                  </div>
                </section>

                <section v-if="selectedTaskFiles.length > 0" class="detail-document-section is-files">
                  <div class="detail-section-head">
                    <div>
                      <div class="detail-section-title">{{ t('scheduledTask.attachments') }}</div>
                      <div class="detail-section-subtitle">{{ t('scheduledTask.attachmentsHint') }}</div>
                    </div>
                  </div>
                  <div class="detail-file-list">
                    <el-tag
                      v-for="file in selectedTaskFiles"
                      :key="file.ref"
                      size="large"
                      effect="plain"
                      class="detail-file-tag"
                      :title="file.ref"
                    >
                      <el-icon><Paperclip /></el-icon>
                      <span>{{ fileDisplayName(file) }}</span>
                    </el-tag>
                  </div>
                </section>
              </template>

              <section v-if="!inlineEditing && !isBuiltinTask(selectedTask)" class="detail-document-section detail-management-section">
                <div>
                  <div class="detail-section-title">更多管理</div>
                  <div class="detail-section-subtitle">取消会保留记录；删除后会从数字员工队伍中移除。</div>
                </div>
                <div class="detail-edit-actions">
                  <el-button type="danger" plain :disabled="isTerminal(selectedTask.status)" @click="handleCancel(selectedTask)">
                    {{ t('scheduledTask.cancel') }}
                  </el-button>
                  <el-button type="danger" plain :disabled="!!selectedTask.inflight_execution_id" @click="handleDelete(selectedTask)">
                    {{ t('scheduledTask.delete') }}
                  </el-button>
                </div>
              </section>

              <ScheduledAgentExecutionRecords
                :state="selectedExecutionState"
                :focused-execution-id="focusedExecutionId"
                :workspace-path="selectedTaskWorkspacePath"
                @status-change="handleSelectedExecutionStatusChange"
                @refresh="loadSelectedExecutions(true)"
                @page-change="handleSelectedExecutionPageChange"
              />
            </section>

          </div>
        </template>
      </main>
    </div>

    <ScheduledAgentTaskDialog
      v-model="showCreateDialog"
      :full-code-path="resourcePath || ''"
      @success="handleCreated"
    />

    <WorkspaceResourceHoverCard
      :preview="resourcePreview"
      @mouseenter="cancelCloseResourcePreview"
      @mouseleave="scheduleCloseResourcePreview"
      @focusin="cancelCloseResourcePreview"
      @focusout="scheduleCloseResourcePreview"
      @open="openScheduledResourcePreviewTarget"
      @close="closeResourcePreview"
    />

  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CaretRight, CopyDocument, EditPen, Paperclip, Plus, Refresh, VideoPause, VideoPlay } from '@element-plus/icons-vue'
import {
  cancelTimerTask,
  createTimerTask,
  deleteTimerTask,
  getTimerExecution,
  listTimerExecutions,
  listTimerTasks,
  pauseTimerTask,
  resumeTimerTask,
  runTimerTaskNow,
  updateTimerTask,
  type TimerExecution,
  type TimerOverlapPolicy,
  type TimerTask,
} from '@/architecture/presentation/context/api/timer'
import {
  buildTimerSchedule,
  createDefaultTimerScheduleForm,
  formatDateTime,
  scheduleLabel,
  taskStatusLabel,
  taskStatusTag,
  timerScheduleToForm,
  type TimerScheduleForm,
} from './utils/timerSchedule'
import { buildScheduledExecutionRoute } from '@/architecture/shared/routing/platformRouteParams'
import ScheduledAgentTaskDialog from './ScheduledAgentTaskDialog.vue'
import ScheduledAgentExecutionRecords from './ScheduledAgentExecutionRecords.vue'
import ScheduledAgentInlineEditor from './ScheduledAgentInlineEditor.vue'
import ScheduledAgentTaskAside from './ScheduledAgentTaskAside.vue'
import AgentEmployeeMascot from './AgentEmployeeMascot.vue'
import WorkspaceResourceHoverCard from './WorkspaceResourceHoverCard.vue'
import type { WorkspaceChatMessageFile } from '@/architecture/presentation/context/api/workspace'
import { fileNameFromRef, parseFileRefs, stringifyFileRefs } from '@/architecture/presentation/widgets/filesWidgetTypes'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'
import {
  getWorkspaceResourceSelectionText,
  renderWorkspaceResourceTokensAsHtml
} from '@/architecture/presentation/components/utils/workspaceInvocationSnippet'
import { useLLMConfigOptions } from '@/architecture/presentation/composables/useLLMConfigOptions'
import { useWorkspaceResourceHoverPreview } from '@/architecture/presentation/composables/useWorkspaceResourceHoverPreview'

interface ExecutionState {
  loading: boolean
  loaded: boolean
  error: string
  status: string
  page: number
  pageSize: number
  total: number
  list: TimerExecution[]
}

interface InlineScheduledAgentForm extends TimerScheduleForm {
  title: string
  description: string
  message: string
  files: string
  llm_config_id: number
  overlap_policy: TimerOverlapPolicy
  max_parallelism: number
}

const props = withDefaults(defineProps<{
  resourcePath?: string
  autoLoad?: boolean
  focusTaskId?: number | string
  focusExecutionId?: number | string
}>(), {
  resourcePath: '',
  autoLoad: false,
  focusTaskId: '',
  focusExecutionId: '',
})

const emit = defineEmits<{
  (e: 'total-change', value: number): void
}>()

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const {
  resourcePreview,
  showResourcePreviewFromEvent,
  scheduleCloseResourcePreview,
  cancelCloseResourcePreview,
  closeResourcePreview,
  openResourcePreviewTarget,
} = useWorkspaceResourceHoverPreview()
const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()
const {
  llmList,
  llmLoading,
  loadLLMOptions,
  handleLLMSelectVisibleChange,
  llmConfigLabel,
  llmOptionLabel,
} = useLLMConfigOptions()
const loading = ref(false)
const list = ref<TimerTask[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const statusFilter = ref('')
const appliedStatusFilter = ref('')
const appliedPage = ref(1)
const showCreateDialog = ref(false)
const selectedTask = ref<TimerTask | null>(null)
const focusedExecutionId = ref(0)
const appliedFocusKey = ref('')
const selectedTaskId = computed(() => selectedTask.value?.id || 0)
const selectedTaskWorkspacePath = computed(() => getTaskWorkspacePath(selectedTask.value))
const inlineEditing = ref(false)
const inlineSaving = ref(false)
const inlineInitialSnapshot = ref('')
const inlineForm = reactive<InlineScheduledAgentForm>({
  title: '',
  description: '',
  message: '',
  files: '',
  llm_config_id: 0,
  overlap_policy: 'forbid',
  max_parallelism: 2,
  ...createDefaultTimerScheduleForm(),
})

const selectedExecutionState = reactive<ExecutionState>({
  loading: false,
  loaded: false,
  error: '',
  status: '',
  page: 1,
  pageSize: 10,
  total: 0,
  list: [],
})

const selectedTaskFiles = computed<WorkspaceChatMessageFile[]>(() => {
  return getTaskFileRefs(selectedTask.value).map(fileRefToMessageFile)
})
const hasUnsavedInlineChanges = computed(() => {
  return inlineEditing.value && buildInlineSnapshot() !== inlineInitialSnapshot.value
})

async function loadList() {
  if (!props.resourcePath) {
    list.value = []
    total.value = 0
    selectedTask.value = null
    discardInlineEdit()
    resetSelectedExecutionState()
    emit('total-change', 0)
    return
  }

  loading.value = true
  try {
    const resp = await listTimerTasks({
      executor_key: 'agent.session',
      resource_scope: 'workspace_directory',
      resource_key_prefix: props.resourcePath,
      status: statusFilter.value,
      page: page.value,
      page_size: pageSize,
    })
    list.value = resp.list || []
    total.value = Number(resp.total || 0)
    appliedStatusFilter.value = statusFilter.value
    appliedPage.value = page.value
    emit('total-change', total.value)
    await syncSelectedTaskAfterListLoad()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.loadAgentsFailed'))
  } finally {
    loading.value = false
  }
}

async function handleListRefresh() {
  if (!await confirmDiscardInlineChanges()) {
    return
  }
  discardInlineEdit()
  await loadList()
}

async function handleOpenCreate() {
  if (!await confirmDiscardInlineChanges()) {
    return
  }
  discardInlineEdit()
  showCreateDialog.value = true
}

async function handleStatusFilterChange() {
  if (!await confirmDiscardInlineChanges()) {
    statusFilter.value = appliedStatusFilter.value
    return
  }
  discardInlineEdit()
  page.value = 1
  await loadList()
}

async function handlePageChange(nextPage: number) {
  if (!await confirmDiscardInlineChanges()) {
    page.value = appliedPage.value
    return
  }
  discardInlineEdit()
  page.value = nextPage
  await loadList()
}

async function loadSelectedExecutions(reset = false) {
  if (!selectedTask.value) return
  if (reset) selectedExecutionState.page = 1
  selectedExecutionState.loading = true
  selectedExecutionState.error = ''
  try {
    const resp = await listTimerExecutions(selectedTask.value.id, {
      status: selectedExecutionState.status,
      page: selectedExecutionState.page,
      page_size: selectedExecutionState.pageSize,
    })
    selectedExecutionState.list = resp.list || []
    await ensureFocusedExecutionLoaded()
    selectedExecutionState.total = Number(resp.total || 0)
    selectedExecutionState.loaded = true
  } catch (error) {
    selectedExecutionState.error = error instanceof Error ? error.message : t('scheduledTask.loadExecutionsFailed')
  } finally {
    selectedExecutionState.loading = false
  }
}

function handleSelectedExecutionPageChange(nextPage: number) {
  selectedExecutionState.page = nextPage
  void loadSelectedExecutions()
}

function handleSelectedExecutionStatusChange(status: string) {
  selectedExecutionState.status = status
  void loadSelectedExecutions(true)
}

function isTaskPaused(task?: TimerTask | null): boolean {
  return task?.status === 'paused'
}

function resetSelectedExecutionState() {
  selectedExecutionState.page = 1
  selectedExecutionState.status = ''
  selectedExecutionState.loaded = false
  selectedExecutionState.list = []
  selectedExecutionState.total = 0
  selectedExecutionState.error = ''
  selectedExecutionState.loading = false
}

async function syncSelectedTaskAfterListLoad() {
  const focused = await openFocusedTaskIfNeeded()
  if (focused) return

  if (selectedTask.value) {
    const refreshed = list.value.find(item => item.id === selectedTask.value?.id)
    if (refreshed) {
      selectedTask.value = refreshed
      return
    }
  }

  const firstTask = list.value[0]
  if (firstTask) {
    await selectTask(firstTask, false)
    return
  }

  selectedTask.value = null
  discardInlineEdit()
  resetSelectedExecutionState()
}

async function selectTask(task: TimerTask, syncRoute = true) {
  if (selectedTaskId.value === task.id && selectedExecutionState.loaded) {
    selectedTask.value = task
    return
  }
  if (!await confirmDiscardInlineChanges()) {
    return
  }
  discardInlineEdit()
  selectedTask.value = task
  resetSelectedExecutionState()
  const taskWorkspacePath = getTaskWorkspacePath(task)
  if (syncRoute && taskWorkspacePath) {
    await router.replace({
      path: route.path,
      query: buildScheduledExecutionRoute({
        fullCodePath: taskWorkspacePath,
        kind: 'agent',
        taskId: task.id,
      }).query,
    })
  }
  await loadSelectedExecutions()
}

function normalizeFocusID(value?: number | string): number {
  const num = Number(value || 0)
  return Number.isFinite(num) && num > 0 ? num : 0
}

function currentFocusKey(): string {
  const taskID = normalizeFocusID(props.focusTaskId)
  if (!taskID) return ''
  return `${taskID}:${normalizeFocusID(props.focusExecutionId)}`
}

async function openFocusedTaskIfNeeded(): Promise<boolean> {
  const key = currentFocusKey()
  if (!key || appliedFocusKey.value === key) return false
  const taskID = normalizeFocusID(props.focusTaskId)
  const task = list.value.find(item => item.id === taskID)
  if (!task) return false
  appliedFocusKey.value = key
  focusedExecutionId.value = normalizeFocusID(props.focusExecutionId)
  await selectTask(task, false)
  return true
}

async function ensureFocusedExecutionLoaded() {
  if (!selectedTask.value || !focusedExecutionId.value) return
  if (selectedExecutionState.list.some(item => item.id === focusedExecutionId.value)) return
  try {
    const execution = await getTimerExecution(selectedTask.value.id, focusedExecutionId.value)
    selectedExecutionState.list = [execution, ...selectedExecutionState.list]
  } catch {
    // 聚焦执行可能已被删除或不属于当前任务，忽略即可。
  }
}

function getAgentPayload(task?: TimerTask | null): Record<string, unknown> {
  return task?.executor_payload && typeof task.executor_payload === 'object'
    ? task.executor_payload as Record<string, unknown>
    : {}
}

function numberFromUnknown(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim()) {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : 0
  }
  return 0
}

function getTaskLLMConfigID(task?: TimerTask | null): number {
  const id = numberFromUnknown(getAgentPayload(task).llm_config_id)
  return id > 0 ? id : 0
}

function getTaskLLMConfigLabel(task?: TimerTask | null): string {
  return llmConfigLabel(getTaskLLMConfigID(task), t('scheduledTask.defaultModel'))
}

function getTaskFileRefs(task?: TimerTask | null): string[] {
  const payload = getAgentPayload(task)
  return parsePayloadFileRefs(payload.files)
}

function parsePayloadFileRefs(value: unknown): string[] {
  if (typeof value === 'string') {
    return parseFileRefs(value)
  }
  if (!Array.isArray(value)) {
    return []
  }
  return stringifyFileRefs(value.map((item) => {
    if (typeof item === 'string') return item
    if (item && typeof item === 'object') {
      const ref = (item as Record<string, unknown>).ref
      return typeof ref === 'string' ? ref : ''
    }
    return ''
  })).split(',').filter(Boolean)
}

function fileRefToMessageFile(ref: string): WorkspaceChatMessageFile {
  const name = fileNameFromRef(ref)
  return {
    ref,
    name,
    source_name: name,
    is_uploaded: true,
  }
}

function fileDisplayName(file: WorkspaceChatMessageFile): string {
  return file.source_name || file.name || fileNameFromRef(file.ref)
}

function getAgentMessage(task?: TimerTask | null): string {
  const payload = getAgentPayload(task)
  const value = payload.message
  if (typeof value === 'string' && value.trim()) {
    return value.trim()
  }
  return ''
}

function getAgentDisplayMessage(task?: TimerTask | null): string {
  const payload = getAgentPayload(task)
  const value = payload.display_content
  if (typeof value === 'string' && value.trim()) {
    return value.trim()
  }
  return getAgentMessage(task)
}

function renderAgentMessageMarkdown(task?: TimerTask | null): string {
  const message = getAgentDisplayMessage(task)
  if (!message) return ''
  return renderMarkdown(renderWorkspaceResourceTokensAsHtml(message, getTaskWorkspacePath(task)))
}

function onAgentMessageMarkdownCopy(event: ClipboardEvent) {
  const root = event.currentTarget instanceof HTMLElement ? event.currentTarget : null
  const text = getWorkspaceResourceSelectionText(root)
  if (!text) return
  event.preventDefault()
  event.clipboardData?.setData('text/plain', text)
}

function openScheduledResourcePreviewTarget() {
  openResourcePreviewTarget(router)
}

function getTaskDescription(task?: TimerTask | null): string {
  return (task?.description || '').trim()
}

function compactText(value: string, fallback: string): string {
  const text = value.replace(/\s+/g, ' ').trim()
  if (!text) return fallback
  return text.length > 96 ? `${text.slice(0, 96)}...` : text
}

function getTaskSummary(task?: TimerTask | null): string {
  return compactText(getTaskDescription(task) || getAgentDisplayMessage(task), t('scheduledTask.noDescription'))
}

function agentEmployeeState(task: TimerTask): 'working' | 'ready' | 'paused' | 'failed' {
  if (task.inflight_execution_id) return 'working'
  if (task.status === 'failed' || Boolean(task.last_error_message?.trim())) return 'failed'
  if (task.status === 'pending') return 'ready'
  return 'paused'
}

function agentEmployeeStatus(task: TimerTask): string {
  const state = agentEmployeeState(task)
  if (state === 'working') return '工作中'
  if (state === 'ready') return '待命'
  if (state === 'failed') return '需要关注'
  return taskStatusLabel(task.status)
}

function agentEmployeeTagType(task: TimerTask): 'primary' | 'success' | 'warning' | 'danger' | 'info' {
  const state = agentEmployeeState(task)
  if (state === 'working') return 'success'
  if (state === 'ready') return 'primary'
  if (state === 'failed') return 'danger'
  return 'warning'
}

function overlapPolicyLabel(policy?: TimerOverlapPolicy): string {
  if (policy === 'queue_latest') return t('scheduledTask.overlapQueueLatest')
  if (policy === 'allow') return t('scheduledTask.overlapAllow')
  return t('scheduledTask.overlapForbid')
}

function applyInlineScheduleForm(scheduleForm: TimerScheduleForm) {
  inlineForm.schedule_type = scheduleForm.schedule_type
  inlineForm.run_at = scheduleForm.run_at
  inlineForm.cron_expr = scheduleForm.cron_expr
  inlineForm.interval_seconds = scheduleForm.interval_seconds
  inlineForm.timezone = scheduleForm.timezone
  inlineForm.max_runs = scheduleForm.max_runs
}

function resetInlineForm(task: TimerTask) {
  const scheduleForm = timerScheduleToForm(task.schedule)
  const fileRefs = getTaskFileRefs(task)
  inlineForm.title = task.title || ''
  inlineForm.description = getTaskDescription(task)
  inlineForm.message = getAgentMessage(task)
  inlineForm.files = stringifyFileRefs(fileRefs)
  inlineForm.llm_config_id = getTaskLLMConfigID(task)
  inlineForm.overlap_policy = task.overlap_policy || 'forbid'
  inlineForm.max_parallelism = task.max_parallelism || 2
  if (inlineForm.llm_config_id > 0) {
    void loadLLMOptions()
  }
  applyInlineScheduleForm(scheduleForm)
  inlineInitialSnapshot.value = buildInlineSnapshot()
}

function startInlineEdit(task: TimerTask) {
  if (isBuiltinTask(task)) {
    ElMessage.info('目录内置任务不能直接修改，请先复制为自定义任务。')
    return
  }
  resetInlineForm(task)
  inlineEditing.value = true
}

async function cancelInlineEdit() {
  if (!await confirmDiscardInlineChanges()) {
    return
  }
  discardInlineEdit()
  if (selectedTask.value) {
    resetInlineForm(selectedTask.value)
  }
}

function discardInlineEdit() {
  inlineEditing.value = false
  inlineInitialSnapshot.value = ''
}

async function confirmDiscardInlineChanges(): Promise<boolean> {
  if (!hasUnsavedInlineChanges.value) {
    return true
  }
  try {
    await ElMessageBox.confirm(t('scheduledTask.discardEditConfirm'), t('scheduledTask.discardEditTitle'), {
      type: 'warning',
      confirmButtonText: t('scheduledTask.discardEditButton'),
      cancelButtonText: t('common.back'),
    })
    return true
  } catch (error) {
    if (error === 'cancel' || error === 'close') {
      return false
    }
    throw error
  }
}

function buildInlineSnapshot(): string {
  return JSON.stringify({
    title: inlineForm.title,
    description: inlineForm.description,
    message: inlineForm.message,
    files: inlineForm.files,
    llm_config_id: Number(inlineForm.llm_config_id || 0),
    schedule_type: inlineForm.schedule_type,
    run_at: inlineForm.run_at,
    cron_expr: inlineForm.cron_expr,
    interval_seconds: Number(inlineForm.interval_seconds || 0),
    timezone: inlineForm.timezone,
    max_runs: Number(inlineForm.max_runs || 0),
    overlap_policy: inlineForm.overlap_policy,
    max_parallelism: Number(inlineForm.max_parallelism || 1),
  })
}

function validateInlineEditForm(): boolean {
  if (!inlineForm.title.trim()) {
    ElMessage.warning(t('scheduledTask.taskTitleRequired'))
    return false
  }
  if (!inlineForm.message.trim()) {
    ElMessage.warning(t('scheduledTask.agentMessageRequired'))
    return false
  }
  if (inlineForm.schedule_type === 'atime' && !inlineForm.run_at.trim()) {
    ElMessage.warning(t('scheduledTask.runAtRequired'))
    return false
  }
  if (inlineForm.schedule_type === 'cron' && !inlineForm.cron_expr.trim()) {
    ElMessage.warning(t('scheduledTask.cronRequired'))
    return false
  }
  if (inlineForm.schedule_type === 'every' && Number(inlineForm.interval_seconds || 0) < 1) {
    ElMessage.warning(t('scheduledTask.intervalRequired'))
    return false
  }
  return true
}

function buildInlineExecutorPayload(task: TimerTask, fullCodePath: string): Record<string, unknown> {
  const payload = { ...getAgentPayload(task) }
  const message = inlineForm.message.trim()
  const files = inlineForm.files.trim()
  payload.full_code_path = fullCodePath
  payload.message = message
  payload.display_content = message
  if (inlineForm.llm_config_id > 0) {
    payload.llm_config_id = inlineForm.llm_config_id
  } else {
    delete payload.llm_config_id
  }
  if (files) {
    payload.files = files
  } else {
    delete payload.files
  }
  return payload
}

function replaceTaskInList(task: TimerTask) {
  list.value = list.value.map((item) => item.id === task.id ? task : item)
}

async function saveInlineEdit() {
  const task = selectedTask.value
  if (!task || !validateInlineEditForm()) return

  const fullCodePath = getTaskWorkspacePath(task)
  if (!fullCodePath) {
    ElMessage.warning(t('scheduledTask.selectWorkspace'))
    return
  }

  inlineSaving.value = true
  try {
    const updatedTask = await updateTimerTask(task.id, {
      title: inlineForm.title.trim(),
      description: inlineForm.description.trim(),
      executor_payload: buildInlineExecutorPayload(task, fullCodePath),
      metadata: {
        ...(task.metadata || {}),
        kind: 'scheduled_agent_session',
      },
      schedule: buildTimerSchedule(inlineForm),
      overlap_policy: inlineForm.overlap_policy,
      max_parallelism: inlineForm.overlap_policy === 'allow' ? inlineForm.max_parallelism : 1,
      source_type: task.source_type || 'agent_session',
      source_ref: fullCodePath,
      resource_scope: task.resource_scope || 'workspace_directory',
      resource_key: fullCodePath,
    })
    selectedTask.value = updatedTask
    replaceTaskInList(updatedTask)
    discardInlineEdit()
    ElMessage.success(t('scheduledTask.savedAgent'))
    await loadList()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.saveFailed'))
  } finally {
    inlineSaving.value = false
  }
}

function getTaskWorkspacePath(task?: TimerTask | null): string {
  const payload = getAgentPayload(task)
  const payloadPath = payload.full_code_path
  for (const path of [
    task?.resource_key,
    task?.source_ref,
    typeof payloadPath === 'string' ? payloadPath : '',
    props.resourcePath,
  ]) {
    if (typeof path === 'string' && path.trim()) {
      return path.trim()
    }
  }
  return ''
}

function isTerminal(status: string): boolean {
  return ['done', 'failed', 'cancelled'].includes(status)
}

async function handleRunNow(task: TimerTask) {
  try {
    await runTimerTaskNow(task.id)
    ElMessage.success(t('scheduledTask.submittedRunNow'))
    await loadList()
    if (selectedTask.value?.id === task.id) await loadSelectedExecutions(true)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.runNowFailed'))
  }
}

async function handlePause(task: TimerTask) {
  try {
    await pauseTimerTask(task.id)
    ElMessage.success(t('scheduledTask.pausedSuccess'))
    await loadList()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.pauseFailed'))
  }
}

async function handleResume(task: TimerTask) {
  try {
    await resumeTimerTask(task.id)
    ElMessage.success(t('scheduledTask.resumedSuccess'))
    await loadList()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.resumeFailed'))
  }
}

async function handleCancel(task: TimerTask) {
  if (isBuiltinTask(task)) {
    ElMessage.info('目录内置任务不能取消；可以暂停，或复制为自定义任务。')
    return
  }
  try {
    await ElMessageBox.confirm(t('scheduledTask.cancelAgentConfirm'), t('scheduledTask.cancelAgentTitle'), {
      type: 'warning',
      confirmButtonText: t('scheduledTask.cancelTaskButton'),
      cancelButtonText: t('common.back'),
    })
    await cancelTimerTask(task.id)
    ElMessage.success(t('scheduledTask.cancelledSuccess'))
    await loadList()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.cancelFailed'))
  }
}

async function handleDelete(task: TimerTask) {
  if (isBuiltinTask(task)) {
    ElMessage.info('目录内置任务不能删除；可以暂停，或复制为自定义任务。')
    return
  }
  try {
    await ElMessageBox.confirm(
      t('scheduledTask.deleteAgentConfirm'),
      t('scheduledTask.deleteAgentTitle'),
      {
        type: 'warning',
        confirmButtonText: t('scheduledTask.delete'),
        cancelButtonText: t('common.back'),
        confirmButtonClass: 'el-button--danger',
      }
    )
    await deleteTimerTask(task.id)
    ElMessage.success(t('scheduledTask.deletedSuccess'))
    if (selectedTask.value?.id === task.id) {
      selectedTask.value = null
      discardInlineEdit()
      resetSelectedExecutionState()
    }
    if (list.value.length <= 1 && page.value > 1) page.value -= 1
    await loadList()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.deleteFailed'))
  }
}

function isBuiltinTask(task?: TimerTask | null): boolean {
  const metadata = task?.metadata || {}
  return metadata.origin === 'manifest'
    || metadata.managed_by === 'app_manifest'
    || metadata.managed_by === 'capability_bundle'
}

async function handleCopyAsCustom(task: TimerTask) {
  try {
    const copy = await createTimerTask({
      title: `${task.title || t('scheduledTask.unnamedAgentTask')}（自定义）`,
      description: task.description,
      category: task.category || 'scheduled_agent_session',
      tags: [...(task.tags || []).filter(tag => tag !== 'app_manifest'), 'custom'],
      idempotency_key: `custom-agent-${task.id}-${crypto.randomUUID()}`,
      executor_key: task.executor_key,
      executor_payload: task.executor_payload,
      metadata: {
        ...(task.metadata || {}),
        origin: 'user',
        managed_by: 'user',
        derived_from: String(task.id),
      },
      status: 'paused',
      schedule: task.schedule,
      overlap_policy: task.overlap_policy,
      max_parallelism: task.max_parallelism,
      source_type: task.source_type,
      source_ref: task.source_ref,
      resource_scope: task.resource_scope,
      resource_key: task.resource_key,
    })
    try {
      await ElMessageBox.confirm(
        '自定义副本已创建并保持暂停。为避免两份任务重复执行，是否同时暂停原来的目录内置任务？',
        '复制完成',
        {
          type: 'warning',
          confirmButtonText: '暂停原任务',
          cancelButtonText: '保持原任务',
        }
      )
      if (task.status === 'pending') {
        await pauseTimerTask(task.id)
      }
    } catch (error) {
      if (error !== 'cancel' && error !== 'close') throw error
    }
    ElMessage.success('已复制为自定义任务；副本默认暂停，请确认后再启用。')
    await loadList()
    const created = list.value.find(item => item.id === copy.id)
    if (created) await selectTask(created)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '复制任务失败')
  }
}

function handleCreated() {
  page.value = 1
  void loadList()
}

watch(
  () => [props.autoLoad, props.resourcePath],
  ([autoLoad]) => {
    if (autoLoad) {
      page.value = 1
      void loadList()
    }
  },
  { immediate: true }
)

watch(
  () => [props.focusTaskId, props.focusExecutionId],
  () => {
    appliedFocusKey.value = ''
    void openFocusedTaskIfNeeded()
  }
)

watch(
  () => getTaskLLMConfigID(selectedTask.value),
  (llmConfigID) => {
    if (llmConfigID > 0) {
      void loadLLMOptions()
    }
  },
  { immediate: true }
)

onBeforeUnmount(closeResourcePreview)

defineExpose({ load: loadList })
</script>

<style scoped lang="scss">
.scheduled-agent-task-list {
  --scheduled-session-ink: var(--el-text-color-primary);
  --scheduled-session-muted: var(--el-text-color-secondary);
  --scheduled-session-soft: var(--app-shell-bg, var(--el-bg-color-page));
  --scheduled-session-paper: var(--app-shell-panel-bg-strong, var(--el-bg-color));
  --scheduled-session-tint: var(--app-shell-panel-muted-bg, var(--el-fill-color-light));
  --scheduled-session-line: var(--app-shell-panel-border, var(--el-border-color-lighter));
  --scheduled-session-accent: var(--el-color-primary);
  box-sizing: border-box;
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
  overflow: hidden;
  padding: 16px;
  border: 1px solid var(--scheduled-session-line);
  border-radius: 14px;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--app-shell-panel-muted-bg, var(--el-fill-color-light)) 78%, transparent), var(--scheduled-session-soft)),
    var(--scheduled-session-soft);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight, rgba(255, 255, 255, 0.8));
}

.scheduled-list-header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
  flex-shrink: 0;
  padding: 14px 16px;
  border: 1px solid var(--scheduled-session-line);
  border-radius: 12px;
  background: var(--scheduled-session-paper);
  box-shadow: var(--app-shell-panel-shadow-soft, 0 10px 24px rgba(15, 23, 42, 0.06));
}

.scheduled-list-title {
  font-size: 15px;
  font-weight: 700;
  line-height: 1.35;
  color: var(--scheduled-session-ink);
}

.scheduled-list-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: var(--scheduled-session-muted);
  word-break: break-all;
}

.scheduled-list-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.scheduled-total {
  display: inline-flex;
  align-items: center;
  height: 26px;
  padding: 0 9px;
  border: 1px solid var(--scheduled-session-line);
  border-radius: 999px;
  background: var(--scheduled-session-tint);
  font-size: 12px;
  color: var(--scheduled-session-accent);
}

.scheduled-list-filter {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  padding: 12px;
  border-bottom: 1px solid var(--scheduled-session-line);
  background: var(--scheduled-session-tint);
}

.scheduled-list-filter :deep(.el-select) {
  width: 132px;
  flex: 0 0 auto;
}

.scheduled-sidebar-title {
  min-width: 0;
  font-size: 13px;
  font-weight: 700;
  color: var(--scheduled-session-ink);
}

.scheduled-agent-workspace {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(310px, 360px) minmax(0, 1fr);
  gap: 14px;
}

.scheduled-agent-sidebar,
.scheduled-agent-detail {
  min-width: 0;
  border: 1px solid var(--scheduled-session-line);
  border-radius: 12px;
  background: var(--app-shell-panel-bg, var(--el-bg-color));
  box-shadow: var(--app-shell-panel-shadow-soft, 0 10px 24px rgba(15, 23, 42, 0.06));
}

.scheduled-agent-sidebar {
  display: flex;
  min-height: 0;
  overflow: hidden;
  flex-direction: column;
  background: var(--app-shell-panel-bg, var(--el-bg-color));
}

.agent-session-list {
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: grid;
  align-content: start;
  gap: 9px;
  padding: 12px;
}

.agent-session-item {
  appearance: none;
  width: 100%;
  min-width: 0;
  display: grid;
  grid-template-columns: 76px minmax(0, 1fr);
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--scheduled-session-line);
  border-left: 3px solid transparent;
  border-radius: 10px;
  background: var(--scheduled-session-paper);
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.16s ease, background 0.16s ease, box-shadow 0.16s ease, transform 0.16s ease;
}

.agent-session-item-avatar {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: center;
  padding-top: 2px;
}

.agent-session-item-avatar :deep(.agent-employee-mascot) {
  width: 72px;
  height: 64px;
}

.agent-session-item-content {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 7px;
}

.agent-session-item:hover {
  border-color: rgba(var(--el-color-primary-rgb), 0.34);
  background: color-mix(in srgb, var(--el-color-primary) 5%, var(--scheduled-session-paper));
  box-shadow: 0 6px 16px rgba(15, 23, 42, 0.06);
}

.agent-session-item.is-active {
  border-color: rgba(var(--el-color-primary-rgb), 0.5);
  border-left-color: var(--scheduled-session-accent);
  background: color-mix(in srgb, var(--el-color-primary) 8%, var(--scheduled-session-paper));
  box-shadow: 0 8px 20px rgba(var(--el-color-primary-rgb), 0.12);
}

.agent-session-item.is-failed {
  border-left-color: var(--el-color-danger);
}

.agent-session-item.is-paused {
  border-left-color: var(--el-color-warning);
}

.agent-session-item.is-running {
  border-left-color: var(--el-color-primary);
}

.agent-session-item-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.agent-session-item-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.agent-session-item-title {
  min-width: 0;
  color: var(--scheduled-session-ink);
  font-size: 13px;
  font-weight: 700;
  line-height: 1.4;
  word-break: break-word;
}

.agent-session-item-summary {
  color: var(--el-text-color-regular);
  font-size: 12px;
  line-height: 1.55;
  word-break: break-word;
}

.agent-session-item-error {
  padding: 7px 8px;
  border-radius: 7px;
  color: var(--el-color-danger);
  background: color-mix(in srgb, var(--el-color-danger) 8%, var(--scheduled-session-paper));
  font-size: 12px;
  line-height: 1.45;
  word-break: break-word;
}

.agent-session-item-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 2px 8px;
  margin-top: 1px;
}

.agent-session-item-meta span {
  position: relative;
  max-width: 100%;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: none;
  color: var(--scheduled-session-muted);
  font-size: 11.5px;
  line-height: 1.5;
}

.agent-session-item-meta span:not(:first-child)::before {
  content: '·';
  margin-right: 8px;
  color: color-mix(in srgb, var(--scheduled-session-muted) 55%, transparent);
}

.agent-session-item-hint {
  align-self: flex-start;
  max-width: 100%;
  padding: 3px 8px;
  border: 1px solid color-mix(in srgb, var(--scheduled-session-accent) 20%, var(--scheduled-session-line));
  border-radius: 999px;
  background: color-mix(in srgb, var(--scheduled-session-accent) 7%, var(--scheduled-session-paper));
  color: var(--scheduled-session-accent);
  font-size: 11px;
  font-weight: 650;
  line-height: 1.45;
}

.scheduled-pagination {
  display: flex;
  justify-content: flex-end;
  flex-shrink: 0;
  padding: 8px 10px 10px;
  border-top: 1px solid var(--scheduled-session-line);
}

.scheduled-agent-task-list :deep(.el-button) {
  border-radius: 8px;
}

.scheduled-sidebar-empty,
.scheduled-detail-empty {
  flex: 1;
  border: 1px dashed var(--scheduled-session-line);
  background: color-mix(in srgb, var(--scheduled-session-paper) 72%, var(--scheduled-session-tint) 28%);
}

.scheduled-agent-detail {
  min-height: 0;
  overflow: hidden;
  padding: 0;
  display: flex;
  background: var(--app-shell-panel-bg, var(--el-bg-color));
}

.scheduled-detail-empty {
  flex: 1;
  min-height: 360px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.scheduled-detail-shell {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  overflow: hidden;
}

.detail-document {
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 24px 28px 30px;
  background: var(--app-shell-panel-bg, var(--el-bg-color));
}

.detail-employee-hero {
  display: grid;
  grid-template-columns: 124px minmax(0, 1fr);
  gap: 20px;
  align-items: center;
  padding: 6px 0 22px;
}

.detail-employee-avatar {
  display: flex;
  min-height: 108px;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in srgb, var(--scheduled-session-accent) 22%, var(--scheduled-session-line));
  border-radius: 16px;
  background: color-mix(in srgb, var(--scheduled-session-accent) 8%, var(--scheduled-session-paper));
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight, rgba(255, 255, 255, 0.7));
}

.detail-employee-avatar :deep(.agent-employee-mascot) {
  width: 104px;
  height: 90px;
}

.detail-employee-copy {
  min-width: 0;
}

.detail-employee-title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.detail-employee-title-row h2 {
  min-width: 0;
  margin: 0 4px 0 0;
  color: var(--scheduled-session-ink);
  font-size: clamp(22px, 2vw, 30px);
  font-weight: 760;
  line-height: 1.25;
  word-break: break-word;
}

.detail-employee-copy p {
  max-width: 760px;
  margin: 12px 0 0;
  color: var(--el-text-color-regular);
  font-size: 14px;
  line-height: 1.7;
}

.detail-employee-promise {
  margin-top: 10px;
  color: var(--scheduled-session-accent);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.detail-employee-hero .detail-edit-actions {
  grid-column: 1 / -1;
  max-width: none;
  justify-content: flex-start;
}

.detail-employee-facts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 10px;
}

.detail-employee-fact {
  min-width: 0;
  padding: 14px 16px;
  border: 1px solid var(--scheduled-session-line);
  border-radius: 10px;
  background: color-mix(in srgb, var(--scheduled-session-paper) 90%, var(--scheduled-session-tint) 10%);
}

.detail-employee-fact span,
.detail-employee-fact strong {
  display: block;
}

.detail-employee-fact span {
  color: var(--scheduled-session-muted);
  font-size: 11px;
  line-height: 1.4;
}

.detail-employee-fact strong {
  margin-top: 6px;
  overflow: hidden;
  color: var(--scheduled-session-ink);
  font-size: 14px;
  font-weight: 720;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-employee-fact.is-path strong {
  font-family: var(--font-family-mono, ui-monospace, monospace);
  font-size: 12px;
}

.detail-document-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--scheduled-session-line);
}

.detail-kicker {
  color: var(--scheduled-session-accent);
  font-size: 12px;
  font-weight: 700;
  line-height: 1.4;
}

.detail-toolbar-title {
  margin-top: 4px;
  color: var(--el-text-color-regular);
  font-size: 14px;
  line-height: 1.45;
}

.detail-edit-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
  flex: 0 0 auto;
}

.detail-section-subtitle {
  color: var(--scheduled-session-muted);
  font-size: 12px;
  line-height: 1.45;
}

.detail-document-section {
  margin-top: 20px;
  padding-top: 18px;
  border-top: 1px solid var(--scheduled-session-line);
}

.detail-document-toolbar + .detail-document-section {
  border-top: 0;
}

.detail-employee-facts + .detail-document-section {
  border-top: 0;
}

.detail-section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.detail-section-title {
  color: var(--scheduled-session-ink);
  font-size: 15px;
  font-weight: 740;
  line-height: 1.35;
}

.detail-section-subtitle {
  margin-top: 4px;
}

.detail-description-body,
.detail-message-body {
  color: var(--el-text-color-regular);
  font-size: 14px;
  line-height: 1.76;
  white-space: pre-wrap;
  word-break: break-word;
}

.detail-description-body {
  padding: 13px 14px;
  border-left: 3px solid var(--scheduled-session-accent);
  border-radius: 0 8px 8px 0;
  background: color-mix(in srgb, var(--el-color-primary) 6%, var(--scheduled-session-paper));
}

.detail-message-body {
  padding: 18px 20px;
  border: 1px solid color-mix(in srgb, var(--scheduled-session-line) 72%, transparent);
  border-radius: 8px;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--scheduled-session-paper) 92%, var(--scheduled-session-tint) 8%), var(--scheduled-session-paper)),
    var(--scheduled-session-paper);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight, rgba(255, 255, 255, 0.72));
}

.detail-message-markdown {
  color: var(--el-text-color-regular);
  font-size: 14px;
  line-height: 1.76;
  white-space: normal;
  word-break: break-word;
}

.detail-message-markdown :deep(h1),
.detail-message-markdown :deep(h2),
.detail-message-markdown :deep(h3),
.detail-message-markdown :deep(h4),
.detail-message-markdown :deep(h5),
.detail-message-markdown :deep(h6) {
  margin: 1.2em 0 0.55em;
  color: var(--scheduled-session-ink);
  font-weight: 720;
  line-height: 1.35;
}

.detail-message-markdown :deep(h1) {
  margin-top: 0;
  font-size: 1.45em;
}

.detail-message-markdown :deep(h2) {
  font-size: 1.24em;
}

.detail-message-markdown :deep(h3) {
  font-size: 1.08em;
}

.detail-message-markdown :deep(p),
.detail-message-markdown :deep(ul),
.detail-message-markdown :deep(ol),
.detail-message-markdown :deep(blockquote),
.detail-message-markdown :deep(pre),
.detail-message-markdown :deep(table) {
  margin: 0.72em 0;
}

.detail-message-markdown :deep(p:first-child),
.detail-message-markdown :deep(ul:first-child),
.detail-message-markdown :deep(ol:first-child),
.detail-message-markdown :deep(blockquote:first-child),
.detail-message-markdown :deep(pre:first-child),
.detail-message-markdown :deep(table:first-child) {
  margin-top: 0;
}

.detail-message-markdown :deep(p:last-child),
.detail-message-markdown :deep(ul:last-child),
.detail-message-markdown :deep(ol:last-child),
.detail-message-markdown :deep(blockquote:last-child),
.detail-message-markdown :deep(pre:last-child),
.detail-message-markdown :deep(table:last-child) {
  margin-bottom: 0;
}

.detail-message-markdown :deep(ul),
.detail-message-markdown :deep(ol) {
  padding-left: 1.45em;
}

.detail-message-markdown :deep(li) {
  margin: 0.28em 0;
}

.detail-message-markdown :deep(code) {
  padding: 0.14em 0.34em;
  border-radius: 5px;
  background: var(--scheduled-session-tint);
  color: var(--scheduled-session-accent);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.9em;
}

.detail-message-markdown :deep(pre) {
  overflow-x: auto;
  padding: 12px 14px;
  border-radius: 8px;
  background: #1f2937;
  color: #e5e7eb;
}

.detail-message-markdown :deep(pre code) {
  padding: 0;
  background: transparent;
  color: inherit;
}

.detail-message-markdown :deep(blockquote) {
  padding-left: 12px;
  border-left: 3px solid color-mix(in srgb, var(--scheduled-session-accent) 44%, var(--scheduled-session-line));
  color: var(--scheduled-session-muted);
}

.detail-message-markdown :deep(table) {
  display: block;
  width: 100%;
  overflow-x: auto;
  border-collapse: collapse;
}

.detail-message-markdown :deep(th),
.detail-message-markdown :deep(td) {
  padding: 6px 8px;
  border: 1px solid var(--scheduled-session-line);
}

.detail-message-markdown :deep(th) {
  background: var(--scheduled-session-tint);
  color: var(--scheduled-session-ink);
  font-weight: 700;
}

.detail-message-markdown :deep(a) {
  color: var(--scheduled-session-accent);
  text-decoration: none;
}

.detail-message-markdown :deep(a:hover) {
  text-decoration: underline;
}

.detail-message-markdown :deep(.workspace-resource-token) {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  max-width: 100%;
  vertical-align: baseline;
  padding: 1px 7px 1px 5px;
  margin: 0 1px;
  border: 1px solid color-mix(in srgb, var(--scheduled-session-accent) 24%, var(--scheduled-session-line));
  border-radius: 7px;
  background: color-mix(in srgb, var(--scheduled-session-accent) 8%, var(--scheduled-session-paper));
  color: var(--scheduled-session-ink);
  font-size: 0.92em;
  font-weight: 600;
  line-height: 1.55;
  white-space: nowrap;
}

.detail-message-markdown :deep(.workspace-resource-token:hover),
.detail-message-markdown :deep(.workspace-resource-token:focus-visible) {
  border-color: color-mix(in srgb, var(--scheduled-session-accent) 54%, var(--scheduled-session-line));
  background: color-mix(in srgb, var(--scheduled-session-accent) 13%, var(--scheduled-session-paper));
  color: var(--scheduled-session-accent);
  text-decoration: none;
  outline: none;
}

.detail-message-markdown :deep(.workspace-resource-token__icon) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  width: 16px;
  height: 16px;
  line-height: 1;
}

.detail-message-markdown :deep(.workspace-resource-token__img),
.detail-message-markdown :deep(.workspace-resource-token__svg),
.detail-message-markdown :deep(.workspace-resource-token__glyph) {
  display: block;
  width: 16px;
  height: 16px;
  max-width: 16px;
  max-height: 16px;
  margin: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  object-fit: contain;
}

.detail-message-markdown :deep(.workspace-resource-token__glyph) {
  border-radius: 5px;
  background: linear-gradient(135deg, var(--scheduled-session-accent), color-mix(in srgb, var(--scheduled-session-accent) 58%, white));
  position: relative;
}

.detail-message-markdown :deep(.workspace-resource-token__glyph::after) {
  content: '';
  position: absolute;
  inset: 4px;
  border: 2px solid rgba(255, 255, 255, 0.9);
  border-radius: 999px;
}

.detail-message-markdown :deep(.workspace-resource-token__label) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.detail-message-markdown :deep(.workspace-resource-token__type) {
  color: var(--scheduled-session-muted);
  font-size: 0.82em;
  font-weight: 700;
}

.detail-message-empty {
  color: var(--scheduled-session-muted);
}

.detail-document-section.is-message .detail-message-body {
  min-height: 260px;
}

.detail-file-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.detail-file-tag {
  max-width: 100%;
  border-radius: 8px;
}

.detail-file-tag :deep(.el-tag__content) {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.detail-file-tag span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-management-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

:deep(.is-inline-schedule.detail-aside) {
  margin-top: 20px;
  padding: 0;
  overflow: visible;
  border-left: 0;
}

@media (max-width: 1100px) {
  .scheduled-agent-workspace {
    grid-template-columns: minmax(280px, 320px) minmax(0, 1fr);
  }

  .detail-employee-hero {
    grid-template-columns: 104px minmax(0, 1fr);
  }

  .detail-employee-hero .detail-edit-actions {
    grid-column: 1 / -1;
    max-width: none;
    justify-content: flex-start;
  }

  .detail-employee-facts {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

}

@media (max-width: 860px) {
  .scheduled-agent-workspace {
    grid-template-columns: 1fr;
  }

  .scheduled-agent-sidebar {
    max-height: 360px;
  }

  .scheduled-detail-shell {
    grid-template-columns: 1fr;
    overflow: auto;
  }

  .detail-document {
    overflow: visible;
  }

  .detail-document-toolbar,
  .detail-section-head,
  .detail-management-section {
    flex-direction: column;
    align-items: flex-start;
  }

  .detail-edit-actions {
    justify-content: flex-start;
  }
}

@media (max-width: 768px) {
  .scheduled-agent-task-list {
    padding: 12px;
  }

  .scheduled-list-header {
    align-items: stretch;
    flex-direction: column;
  }

  .detail-document {
    padding: 18px 16px 22px;
  }

  .detail-employee-hero {
    grid-template-columns: 82px minmax(0, 1fr);
    gap: 14px;
  }

  .detail-employee-avatar {
    min-height: 82px;
  }

  .detail-employee-avatar :deep(.agent-employee-mascot) {
    width: 76px;
    height: 66px;
  }

  .detail-employee-title-row h2 {
    font-size: 20px;
  }

  .detail-employee-facts {
    grid-template-columns: 1fr;
  }

}
</style>
