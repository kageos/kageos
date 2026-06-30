<template>
  <div class="scheduled-agent-task-list" v-loading="loading">
    <div class="scheduled-list-header">
      <div>
        <div class="scheduled-list-title">{{ t('scheduledTask.agentTitle') }}</div>
        <div class="scheduled-list-subtitle">{{ resourcePath ? t('scheduledTask.currentWorkspace') : t('scheduledTask.noWorkspaceSelected') }}</div>
      </div>
      <div class="scheduled-list-actions">
        <span class="scheduled-total">{{ t('scheduledTask.totalCount', { count: total }) }}</span>
        <el-button :icon="Refresh" @click="handleListRefresh">{{ t('common.refresh') }}</el-button>
        <el-button type="primary" :icon="Plus" :disabled="!resourcePath" @click="handleOpenCreate">
          {{ t('scheduledTask.newTask') }}
        </el-button>
      </div>
    </div>

    <div class="scheduled-agent-workspace">
      <aside class="scheduled-agent-sidebar">
        <div class="scheduled-list-filter">
          <div class="scheduled-sidebar-title">{{ t('scheduledTask.sessionList') }}</div>
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
            <span class="agent-session-item-head">
              <span class="agent-session-item-title">
                {{ task.title || t('scheduledTask.unnamedAgentTask') }}
              </span>
              <el-tag :type="taskStatusTag(task.status)" size="small" effect="light">
                {{ taskStatusLabel(task.status) }}
              </el-tag>
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
              <div class="detail-document-toolbar">
                <div>
                  <div class="detail-kicker">{{ t('scheduledTask.sessionMessage') }}</div>
                  <div class="detail-toolbar-title">
                    {{ inlineEditing ? t('scheduledTask.editAgentDialogTitle') : t('scheduledTask.sessionMessageHint') }}
                  </div>
                </div>
                <div class="detail-edit-actions">
                  <template v-if="inlineEditing">
                    <el-button @click="cancelInlineEdit">{{ t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="inlineSaving" @click="saveInlineEdit">
                      {{ t('common.save') }}
                    </el-button>
                  </template>
                  <el-button
                    v-else
                    type="primary"
                    :icon="EditPen"
                    :disabled="isTerminal(selectedTask.status)"
                    @click="startInlineEdit(selectedTask)"
                  >
                    {{ t('scheduledTask.edit') }}
                  </el-button>
                </div>
              </div>

              <el-form v-if="inlineEditing" class="detail-inline-form" label-position="top">
                <el-form-item :label="t('scheduledTask.taskName')" required>
                  <el-input
                    v-model="inlineForm.title"
                    maxlength="100"
                    show-word-limit
                    :placeholder="t('scheduledTask.agentTaskNamePlaceholder')"
                  />
                </el-form-item>

                <el-form-item :label="t('scheduledTask.agentDescription')">
                  <el-input
                    v-model="inlineForm.description"
                    type="textarea"
                    :autosize="{ minRows: 3, maxRows: 8 }"
                    maxlength="500"
                    show-word-limit
                    :placeholder="t('scheduledTask.agentDescriptionPlaceholder')"
                  />
                </el-form-item>

                <el-form-item :label="t('scheduledTask.agentMessage')" required>
                  <div
                    class="detail-inline-composer"
                    :class="{ 'is-dragging': dragOver }"
                    @paste="onPaste"
                    @dragover.prevent="onDragOver"
                    @dragleave.prevent="onDragLeave"
                    @drop.prevent="onDrop"
                  >
                    <MiniWorkstationComposer
                      variant="schedule"
                      :full-code-path="getTaskWorkspacePath(selectedTask)"
                      :attached-files="inlineComposerFiles"
                      :uploading="uploading"
                      :input-text="inlineForm.message"
                      :sending="false"
                      :session-running="false"
                      :stopping="false"
                      :selected-l-l-m-config-id="0"
                      :llm-list="[]"
                      :llm-loading="false"
                      :queued-count="0"
                      :register-input-ref="registerInlineMessageInputRef"
                      :on-l-l-m-select-visible-change="noop"
                      :on-file-change="onFileChange"
                      :remove-file="removeInlineComposerFile"
                      :on-input-enter="noopInputEnter"
                      :placeholder="t('scheduledTask.agentMessagePlaceholder')"
                      :expanded-title="inlineForm.title || t('scheduledTask.editAgentDialogTitle')"
                      :expanded-subtitle="getTaskWorkspacePath(selectedTask)"
                      :expanded-save-label="t('common.save')"
                      mention-panel-placement="below"
                      @update:input-text="inlineForm.message = $event"
                      @expanded-save="handleInlineComposerExpandedSave"
                    />
                    <div v-if="dragOver" class="detail-inline-drop-hint">
                      {{ t('scheduledTask.dropUpload') }}
                    </div>
                  </div>
                </el-form-item>
              </el-form>

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
                    <StructuredPromptComposer
                      v-if="getAgentMessage(selectedTask)"
                      class="detail-message-preview"
                      :model-value="getAgentMessage(selectedTask)"
                      :placeholder="t('scheduledTask.noMessage')"
                      :disabled="true"
                      :show-toolbar="false"
                      :enable-mentions="false"
                      :readonly-preview="true"
                      :min-rows="8"
                      :max-rows="28"
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

              <section class="detail-document-section is-executions">
                <div class="detail-section-head">
                  <div>
                    <div class="detail-section-title">{{ t('scheduledTask.executionRecords') }}</div>
                    <div class="detail-section-subtitle">{{ t('scheduledTask.executionRecordsHint') }}</div>
                  </div>
                  <div class="drawer-section-controls">
                    <el-select
                      v-model="selectedExecutionState.status"
                      :placeholder="t('scheduledTask.allStatuses')"
                      clearable
                      size="small"
                      style="width: 120px"
                      @change="loadSelectedExecutions(true)"
                    >
                      <el-option :label="t('scheduledTask.allStatuses')" value="" />
                      <el-option :label="t('scheduledTask.executionStatusQueued')" value="queued" />
                      <el-option :label="t('scheduledTask.executionStatusRunning')" value="running" />
                      <el-option :label="t('scheduledTask.executionStatusSuccess')" value="success" />
                      <el-option :label="t('scheduledTask.executionStatusFailed')" value="failed" />
                      <el-option :label="t('scheduledTask.executionStatusTimeout')" value="timeout" />
                    </el-select>
                    <el-button size="small" :icon="Refresh" @click="loadSelectedExecutions(true)">{{ t('common.refresh') }}</el-button>
                  </div>
                </div>

                <div v-if="selectedExecutionState.loading" v-loading="true" class="drawer-executions-loading" />

                <el-alert
                  v-else-if="selectedExecutionState.error"
                  :title="selectedExecutionState.error"
                  type="error"
                  show-icon
                  :closable="false"
                />

                <el-empty
                  v-else-if="selectedExecutionState.loaded && selectedExecutionState.list.length === 0"
                  :description="t('scheduledTask.emptyExecutions')"
                  :image-size="56"
                />

                <div v-else-if="selectedExecutionState.loaded" class="execution-timeline">
                  <article
                    v-for="execution in selectedExecutionState.list"
                    :key="execution.id"
                    :class="['execution-card', executionCardClass(execution), { 'is-focused': execution.id === focusedExecutionId }]"
                  >
                    <div class="execution-card-rail" />
                    <div class="execution-card-main">
                      <div class="execution-card-head">
                        <div class="execution-card-title-line">
                          <el-tag :type="executionStatusTag(execution.status)" size="small" effect="light">
                            {{ executionStatusLabel(execution.status) }}
                          </el-tag>
                          <span class="execution-trigger">{{ executionTriggerLabel(execution) }}</span>
                        </div>
                        <el-button
                          v-if="getExecutionOpenSessionID(execution)"
                          link
                          type="primary"
                          size="small"
                          class="execution-open-session"
                          @click="openExecutionSession(selectedTask!, execution)"
                        >
                          {{ t('scheduledTask.openSession') }}
                        </el-button>
                      </div>

                      <div class="execution-time">{{ formatDateTime(execution.scheduled_at) }}</div>

                      <div class="execution-facts">
                        <span v-if="getExecutionOpenSessionID(execution)">
                          {{ t('scheduledTask.session', { id: shortSessionID(getExecutionOpenSessionID(execution)) }) }}
                        </span>
                        <span>{{ executionToolStats(execution) }}</span>
                        <span v-if="execution.duration_millis">{{ formatDuration(execution.duration_millis) }}</span>
                      </div>

                      <div v-if="executionHumanSummary(execution)" class="execution-summary">
                        {{ executionHumanSummary(execution) }}
                      </div>

                      <div v-if="executionErrorMessage(execution)" class="execution-error-card">
                        <div class="execution-error-title">{{ executionErrorTitle(execution) }}</div>
                        <div class="execution-error-hint">{{ executionErrorHint(selectedTask!, execution) }}</div>
                        <div class="execution-error-detail">{{ executionErrorMessage(execution) }}</div>
                      </div>
                    </div>
                  </article>
                </div>

                <div
                  v-if="selectedExecutionState.loaded && selectedExecutionState.total > selectedExecutionState.pageSize"
                  class="execution-pagination"
                >
                  <el-pagination
                    small
                    :current-page="selectedExecutionState.page"
                    :page-size="selectedExecutionState.pageSize"
                    :total="selectedExecutionState.total"
                    layout="prev, pager, next"
                    @current-change="(nextPage: number) => handleSelectedExecutionPageChange(nextPage)"
                  />
                </div>
              </section>
            </section>

            <aside class="detail-aside">
              <section class="detail-aside-card">
                <div class="detail-aside-card-head">
                  <div class="detail-aside-title">{{ t('scheduledTask.agentDetailTitle') }}</div>
                  <el-tag :type="taskStatusTag(selectedTask.status)" effect="light">
                    {{ taskStatusLabel(selectedTask.status) }}
                  </el-tag>
                </div>
                <div class="detail-aside-name">{{ selectedTask.title || t('scheduledTask.unnamedAgentTask') }}</div>
                <div class="detail-aside-path">{{ getTaskWorkspacePath(selectedTask) || '-' }}</div>
                <div class="detail-aside-actions">
                  <el-tooltip :content="t('scheduledTask.runNow')" placement="top" effect="light">
                    <el-button
                      type="primary"
                      :icon="VideoPlay"
                      :disabled="inlineEditing || isTerminal(selectedTask.status)"
                      @click="handleRunNow(selectedTask)"
                    />
                  </el-tooltip>
                  <el-tooltip
                    :content="selectedTask.status === 'paused' ? t('scheduledTask.resume') : t('scheduledTask.pause')"
                    placement="top"
                    effect="light"
                  >
                    <el-button
                      :type="selectedTask.status === 'paused' ? 'primary' : 'warning'"
                      :icon="selectedTask.status === 'paused' ? CaretRight : VideoPause"
                      :disabled="inlineEditing || isTerminal(selectedTask.status)"
                      @click="selectedTask.status === 'paused' ? handleResume(selectedTask) : handlePause(selectedTask)"
                    />
                  </el-tooltip>
                  <el-tooltip :content="t('scheduledTask.cancel')" placement="top" effect="light">
                    <el-button
                      type="danger"
                      :icon="Close"
                      :disabled="inlineEditing || isTerminal(selectedTask.status)"
                      @click="handleCancel(selectedTask)"
                    />
                  </el-tooltip>
                  <el-tooltip :content="t('scheduledTask.delete')" placement="top" effect="light">
                    <el-button
                      type="danger"
                      plain
                      :icon="Delete"
                      :disabled="inlineEditing || !!selectedTask.inflight_execution_id"
                      @click="handleDelete(selectedTask)"
                    />
                  </el-tooltip>
                </div>
                <el-alert
                  v-if="isTaskPaused(selectedTask)"
                  class="detail-enable-hint"
                  type="info"
                  show-icon
                  :closable="false"
                  :title="t('scheduledTask.enableForUnattendedHint')"
                />
              </section>

              <section class="detail-aside-card">
                <div class="detail-aside-title">{{ t('scheduledTask.schedule') }}</div>
                <div v-if="!inlineEditing" class="detail-property-list">
                  <div class="detail-property">
                    <span>{{ t('scheduledTask.schedule') }}</span>
                    <strong>{{ scheduleLabel(selectedTask.schedule) }}</strong>
                  </div>
                  <div class="detail-property">
                    <span>{{ t('scheduledTask.nextRun') }}</span>
                    <strong>{{ formatDateTime(selectedTask.next_run_at) }}</strong>
                  </div>
                  <div class="detail-property">
                    <span>{{ t('scheduledTask.runCount') }}</span>
                    <strong>{{ selectedTask.run_count || 0 }}</strong>
                  </div>
                  <div class="detail-property">
                    <span>{{ t('scheduledTask.createdBy') }}</span>
                    <strong>{{ selectedTask.created_by || selectedTask.request_user || '-' }}</strong>
                  </div>
                </div>

                <el-form v-else class="detail-schedule-form" label-position="top">
                  <el-form-item :label="t('scheduledTask.scheduleType')" required>
                    <el-radio-group v-model="inlineForm.schedule_type" class="detail-schedule-type">
                      <el-radio-button value="atime">{{ t('scheduledTask.scheduleAtime') }}</el-radio-button>
                      <el-radio-button value="cron">{{ t('scheduledTask.scheduleCron') }}</el-radio-button>
                      <el-radio-button value="every">{{ t('scheduledTask.scheduleEvery') }}</el-radio-button>
                    </el-radio-group>
                  </el-form-item>

                  <el-form-item v-if="inlineForm.schedule_type === 'atime'" :label="t('scheduledTask.runAt')" required>
                    <el-date-picker
                      v-model="inlineForm.run_at"
                      type="datetime"
                      :placeholder="t('scheduledTask.runAtPlaceholder')"
                      format="YYYY-MM-DD HH:mm"
                      value-format="YYYY-MM-DD HH:mm:ss"
                      :shortcuts="dateTimeShortcuts"
                      style="width: 100%"
                    />
                  </el-form-item>

                  <el-form-item v-if="inlineForm.schedule_type === 'cron'" :label="t('scheduledTask.cron')" required>
                    <el-input v-model="inlineForm.cron_expr" placeholder="0 9 * * *" />
                  </el-form-item>

                  <el-form-item v-if="inlineForm.schedule_type === 'every'" :label="t('scheduledTask.intervalSeconds')" required>
                    <el-input-number v-model="inlineForm.interval_seconds" :min="1" :max="86400" style="width: 100%" />
                  </el-form-item>

                  <el-form-item v-if="inlineForm.schedule_type === 'every'" :label="t('scheduledTask.maxRuns')">
                    <el-input-number v-model="inlineForm.max_runs" :min="0" :max="1000000" style="width: 100%" />
                  </el-form-item>
                </el-form>
              </section>

              <el-alert
                v-if="selectedTask.last_error_message"
                class="detail-alert"
                type="error"
                show-icon
                :closable="false"
                :title="selectedTask.last_error_message"
              />
            </aside>
          </div>
        </template>
      </main>
    </div>

    <ScheduledAgentTaskDialog
      v-model="showCreateDialog"
      :full-code-path="resourcePath || ''"
      @success="handleCreated"
    />

  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { CaretRight, Close, Delete, EditPen, Paperclip, Plus, Refresh, VideoPause, VideoPlay } from '@element-plus/icons-vue'
import {
  cancelTimerTask,
  deleteTimerTask,
  getTimerExecution,
  listTimerExecutions,
  listTimerTasks,
  pauseTimerTask,
  resumeTimerTask,
  runTimerTaskNow,
  updateTimerTask,
  type TimerExecution,
  type TimerTask,
} from '@/architecture/presentation/context/api/timer'
import {
  buildTimerSchedule,
  createDefaultTimerScheduleForm,
  executionStatusLabel,
  executionStatusTag,
  formatDateTime,
  formatDuration,
  scheduleLabel,
  taskStatusLabel,
  taskStatusTag,
  timerScheduleToForm,
  type TimerScheduleForm,
} from './utils/timerSchedule'
import { buildScheduledExecutionRoute } from '@/architecture/shared/routing/platformRouteParams'
import ScheduledAgentTaskDialog from './ScheduledAgentTaskDialog.vue'
import MiniWorkstationComposer from './MiniWorkstationComposer.vue'
import StructuredPromptComposer from './StructuredPromptComposer.vue'
import { eventBus } from '@/architecture/presentation/context/eventBusContext'
import { createRelativeDateTimeShortcuts } from '@/architecture/shared/date'
import { useMiniWorkstationUploads } from '@/architecture/presentation/composables/useMiniWorkstationUploads'
import type { WorkspaceChatMessageFile } from '@/architecture/presentation/context/api/workspace'
import { fileNameFromRef, parseFileRefs, stringifyFileRefs } from '@/architecture/presentation/widgets/filesWidgetTypes'

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
const inlineEditing = ref(false)
const inlineSaving = ref(false)
const inlineInitialSnapshot = ref('')
const existingFileRefs = ref<string[]>([])
const dateTimeShortcuts = computed(() => createRelativeDateTimeShortcuts())
const inlineForm = reactive<InlineScheduledAgentForm>({
  title: '',
  description: '',
  message: '',
  files: '',
  ...createDefaultTimerScheduleForm(),
})
const inlineMessageInputRef = ref<{ focus: () => void }>()
const inlineFullCodePath = computed(() => selectedTask.value ? getTaskWorkspacePath(selectedTask.value) : props.resourcePath || '')
const inlineMessageTextRef = computed({
  get: () => inlineForm.message,
  set: (value: string) => {
    inlineForm.message = value
  },
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

const {
  attachedFiles,
  uploading,
  dragOver,
  onFileChange,
  removeFile,
  onDragOver,
  onDragLeave,
  onDrop,
  onPaste,
} = useMiniWorkstationUploads({
  fullCodePath: inlineFullCodePath,
  inputText: inlineMessageTextRef,
  inputRef: inlineMessageInputRef,
})

const currentInlineFileRefs = computed(() => {
  return stringifyFileRefs([
    ...existingFileRefs.value,
    ...attachedFiles.value.map((file) => file.ref).filter((ref): ref is string => !!ref),
  ])
})
const inlineComposerFiles = computed<WorkspaceChatMessageFile[]>(() => {
  return [
    ...existingFileRefs.value.map(fileRefToMessageFile),
    ...attachedFiles.value,
  ]
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
      resource_key: props.resourcePath,
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
  if (syncRoute && props.resourcePath) {
    await router.replace({
      path: route.path,
      query: buildScheduledExecutionRoute({
        fullCodePath: props.resourcePath,
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

function stringFromRecord(record: Record<string, unknown>, key: string): string {
  const value = record[key]
  return typeof value === 'string' ? value.trim() : ''
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
  return compactText(getTaskDescription(task) || getAgentMessage(task), t('scheduledTask.noDescription'))
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
  const payload = getAgentPayload(task)
  const scheduleForm = timerScheduleToForm(task.schedule)
  const fileRefs = getTaskFileRefs(task)
  inlineForm.title = task.title || ''
  inlineForm.description = getTaskDescription(task)
  inlineForm.message = getAgentMessage(task)
  inlineForm.files = stringifyFileRefs(fileRefs)
  existingFileRefs.value = fileRefs
  attachedFiles.value = []
  applyInlineScheduleForm(scheduleForm)
  inlineInitialSnapshot.value = buildInlineSnapshot()
}

function startInlineEdit(task: TimerTask) {
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
  attachedFiles.value = []
  existingFileRefs.value = []
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

function removeInlineComposerFile(index: number) {
  if (index < existingFileRefs.value.length) {
    existingFileRefs.value.splice(index, 1)
    inlineForm.files = currentInlineFileRefs.value
    return
  }
  removeFile(index - existingFileRefs.value.length)
  inlineForm.files = currentInlineFileRefs.value
}

function buildInlineSnapshot(): string {
  return JSON.stringify({
    title: inlineForm.title,
    description: inlineForm.description,
    message: inlineForm.message,
    files: currentInlineFileRefs.value,
    schedule_type: inlineForm.schedule_type,
    run_at: inlineForm.run_at,
    cron_expr: inlineForm.cron_expr,
    interval_seconds: Number(inlineForm.interval_seconds || 0),
    timezone: inlineForm.timezone,
    max_runs: Number(inlineForm.max_runs || 0),
  })
}

function registerInlineMessageInputRef(element: { focus: () => void } | null) {
  inlineMessageInputRef.value = element || undefined
}

function noop() {}

function noopInputEnter() {}

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
  const files = currentInlineFileRefs.value
  payload.full_code_path = fullCodePath
  payload.message = message
  payload.display_content = message
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

function handleInlineComposerExpandedSave(value: string) {
  inlineForm.message = value
  void saveInlineEdit()
}

function getExecutionSessionID(execution: TimerExecution): string {
  const payload = execution.result_payload
  if (payload && typeof payload === 'object') {
    const record = payload as Record<string, unknown>
    const sessionID = record.session_id || record.sessionId
    return typeof sessionID === 'string' ? sessionID : ''
  }
  return ''
}

function executionTriggerLabel(execution: TimerExecution): string {
  return execution.trigger_type === 'manual' ? t('scheduledTask.manualTrigger') : t('scheduledTask.scheduledTrigger')
}

function shortSessionID(sessionID: string): string {
  const value = sessionID.trim()
  if (value.length <= 12) return value
  return `${value.slice(0, 8)}...${value.slice(-4)}`
}

function executionCardClass(execution: TimerExecution): string {
  return `is-${execution.status || 'unknown'}`
}

function executionToolStats(execution: TimerExecution): string {
  const summary = execution.output_summary || ''
  const match = summary.match(/工具调用\s*(\d+)\s*次，失败\s*(\d+)\s*次/)
  if (match) {
    return t('scheduledTask.toolsSummary', { toolCalls: match[1], failures: match[2] })
  }

  const payload = execution.result_payload
  if (payload && typeof payload === 'object') {
    const record = payload as Record<string, unknown>
    const toolCalls = record.tool_calls || record.toolCalls
    if (typeof toolCalls === 'number') return t('scheduledTask.toolsCount', { toolCalls })
  }
  return t('scheduledTask.toolsZero')
}

function executionHumanSummary(execution: TimerExecution): string {
  return (execution.output_summary || '')
    .split('；')
    .map((item) => item.trim())
    .filter((item) => item && !item.startsWith('session_id=') && !/^工具调用\s*\d+\s*次/.test(item))
    .join('；')
}

function executionErrorMessage(execution: TimerExecution): string {
  return (execution.error_message || '')
    .replace(/^业务错误\s*\[\d+\]:\s*/, '')
    .trim()
}

function executionErrorTitle(execution: TimerExecution): string {
  const message = executionErrorMessage(execution)
  if (message.includes('服务目录不存在')) return t('scheduledTask.directoryMissingTitle')
  if (message.includes('timeout') || execution.status === 'timeout') return t('scheduledTask.timeoutTitle')
  if (message.includes('权限')) return t('scheduledTask.permissionFailedTitle')
  return t('scheduledTask.executionError')
}

function executionErrorHint(task: TimerTask, execution: TimerExecution): string {
  const message = executionErrorMessage(execution)
  if (message.includes('服务目录不存在')) {
    const path = getTaskWorkspacePath(task)
    return path
      ? t('scheduledTask.directoryMissingHint', { path })
      : t('scheduledTask.workspaceMissingHint')
  }
  if (message.includes('权限')) return t('scheduledTask.permissionHint')
  return t('scheduledTask.genericErrorHint')
}

function getExecutionOpenSessionID(execution: TimerExecution): string {
  return getExecutionSessionID(execution) || (execution.executor_run_id || '').trim()
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

function getWorkspaceName(fullCodePath: string): string {
  const parts = fullCodePath.split('/').filter(Boolean)
  return parts[parts.length - 1] || fullCodePath || t('scheduledTask.workspaceFallback')
}

function openExecutionSession(task: TimerTask, execution: TimerExecution) {
  const sessionID = getExecutionOpenSessionID(execution)
  if (!sessionID) {
    ElMessage.warning(t('scheduledTask.noOpenSession'))
    return
  }
  const fullCodePath = getTaskWorkspacePath(task)
  if (!fullCodePath) {
    ElMessage.warning(t('scheduledTask.missingSessionPath'))
    return
  }

  eventBus.emit('workspace:open-workstation', {
    full_code_path: fullCodePath,
    session_id: sessionID,
    directory_name: getWorkspaceName(fullCodePath),
    initial_maximized: true,
    open_as_mini: true,
  })
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
  grid-template-columns: minmax(270px, 340px) minmax(0, 1fr);
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
  display: flex;
  flex-direction: column;
  gap: 7px;
  padding: 12px 13px;
  border: 1px solid var(--scheduled-session-line);
  border-left: 3px solid transparent;
  border-radius: 10px;
  background: var(--scheduled-session-paper);
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.16s ease, background 0.16s ease, box-shadow 0.16s ease, transform 0.16s ease;
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

.drawer-section-controls {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
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
  grid-template-columns: minmax(0, 1fr) minmax(260px, 310px);
  overflow: hidden;
}

.detail-document {
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 24px 28px 30px;
  background: var(--app-shell-panel-bg, var(--el-bg-color));
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

.detail-inline-form {
  max-width: 980px;
  padding-top: 18px;
}

.detail-inline-form :deep(.el-form-item__label),
.detail-schedule-form :deep(.el-form-item__label) {
  color: var(--scheduled-session-ink);
  font-weight: 650;
}

.detail-section-subtitle {
  color: var(--scheduled-session-muted);
  font-size: 12px;
  line-height: 1.45;
}

.detail-inline-composer {
  position: relative;
  width: 100%;
  z-index: 2;
}

.detail-inline-composer.is-dragging {
  outline: 2px dashed rgba(var(--el-color-primary-rgb), 0.55);
  outline-offset: 4px;
  border-radius: 14px;
}

.detail-inline-composer :deep(.mini-ws-input) {
  min-height: 240px;
}

.detail-inline-composer :deep(.mini-input-wrap) {
  min-height: 220px;
}

.detail-inline-composer :deep(.mini-structured-input .spc-editor),
.detail-inline-composer :deep(.mini-structured-input .spc-preview) {
  min-height: 190px !important;
  max-height: 520px !important;
}

.detail-inline-composer :deep(.structured-prompt-composer.is-focused) {
  z-index: 30;
}

.detail-inline-composer :deep(.spc-mention-panel) {
  z-index: 2600;
}

.detail-inline-drop-hint {
  position: absolute;
  inset: 0;
  z-index: 3;
  display: grid;
  place-items: center;
  border-radius: 14px;
  background: rgba(15, 23, 42, 0.54);
  color: #fff;
  font-weight: 700;
  pointer-events: none;
}

.detail-alert {
  flex-shrink: 0;
}

.detail-document-section {
  margin-top: 20px;
  padding-top: 18px;
  border-top: 1px solid var(--scheduled-session-line);
}

.detail-document-toolbar + .detail-document-section {
  border-top: 0;
}

.detail-document-section.is-executions {
  padding-bottom: 4px;
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

.detail-message-preview {
  border: 0;
  background: transparent;
  box-shadow: none;
}

.detail-message-preview.is-disabled {
  opacity: 1;
}

.detail-message-preview :deep(.spc-preview) {
  min-height: 0 !important;
  padding: 0;
  color: var(--el-text-color-regular);
  line-height: 1.76;
}

.detail-message-preview :deep(.spc-preview-body) {
  white-space: pre-wrap;
}

.detail-message-preview :deep(.spc-resource-chip),
.detail-message-preview :deep(.spc-user-chip) {
  border-color: color-mix(in srgb, var(--scheduled-session-accent) 26%, var(--scheduled-session-line));
  background: color-mix(in srgb, var(--scheduled-session-accent) 9%, var(--scheduled-session-paper));
  color: var(--scheduled-session-accent);
  box-shadow: none;
}

.detail-message-preview :deep(.spc-user-chip) {
  border-color: color-mix(in srgb, var(--el-color-success) 28%, var(--scheduled-session-line));
  background: color-mix(in srgb, var(--el-color-success) 9%, var(--scheduled-session-paper));
  color: var(--el-color-success);
}

.detail-message-preview :deep(.spc-invocation-card) {
  border-color: color-mix(in srgb, var(--scheduled-session-accent) 18%, var(--scheduled-session-line));
  background: color-mix(in srgb, var(--scheduled-session-accent) 6%, var(--scheduled-session-paper));
}

.detail-message-preview :deep(.spc-invocation-resource) {
  color: var(--scheduled-session-ink);
}

.detail-message-preview :deep(.spc-param-chip) {
  background: var(--scheduled-session-tint);
  color: var(--scheduled-session-muted);
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

.detail-aside {
  min-width: 0;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border-left: 1px solid var(--scheduled-session-line);
  background: color-mix(in srgb, var(--scheduled-session-tint) 52%, var(--app-shell-panel-bg, var(--el-bg-color)));
}

.detail-aside-card {
  min-width: 0;
  padding: 14px;
  border: 1px solid color-mix(in srgb, var(--scheduled-session-line) 74%, transparent);
  border-radius: 8px;
  background: var(--scheduled-session-paper);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight, rgba(255, 255, 255, 0.7));
}

.detail-aside-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.detail-aside-title {
  color: var(--scheduled-session-muted);
  font-size: 12px;
  font-weight: 700;
  line-height: 1.45;
}

.detail-aside-name {
  margin-top: 10px;
  color: var(--scheduled-session-ink);
  font-size: 16px;
  font-weight: 760;
  line-height: 1.35;
  word-break: break-word;
}

.detail-aside-path {
  margin-top: 8px;
  color: var(--scheduled-session-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.55;
  word-break: break-all;
}

.detail-aside-actions {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  margin-top: 14px;
}

.detail-aside-actions :deep(.el-button) {
  width: 100%;
  height: 34px;
  padding: 0;
  margin: 0;
}

.detail-enable-hint {
  margin-top: 12px;
}

.detail-property-list {
  display: grid;
  gap: 8px;
  margin-top: 12px;
}

.detail-property {
  min-width: 0;
  padding: 10px 0;
  border-top: 1px solid color-mix(in srgb, var(--scheduled-session-line) 62%, transparent);
}

.detail-property:first-child {
  border-top: 0;
  padding-top: 0;
}

.detail-property span {
  display: block;
  color: var(--scheduled-session-muted);
  font-size: 12px;
  line-height: 1.45;
}

.detail-property strong {
  display: block;
  margin-top: 5px;
  color: var(--scheduled-session-ink);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.45;
  word-break: break-word;
}

.detail-schedule-form {
  margin-top: 12px;
}

.detail-schedule-form :deep(.el-form-item) {
  margin-bottom: 14px;
}

.detail-schedule-form :deep(.el-form-item:last-child) {
  margin-bottom: 0;
}

.detail-schedule-type {
  display: grid;
  width: 100%;
  gap: 6px;
}

.detail-schedule-type :deep(.el-radio-button__inner) {
  width: 100%;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
}

.detail-schedule-type :deep(.el-radio-button:first-child .el-radio-button__inner),
.detail-schedule-type :deep(.el-radio-button:last-child .el-radio-button__inner) {
  border-radius: 8px;
}

.drawer-executions-loading {
  min-height: 120px;
}

/* ─── 执行时间线 ─── */

.execution-timeline {
  display: grid;
  gap: 10px;
}

.execution-card {
  position: relative;
  display: grid;
  grid-template-columns: 3px minmax(0, 1fr);
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--scheduled-session-line) 60%, transparent);
  border-radius: 10px;
  background: var(--scheduled-session-paper);
  transition: border-color 0.16s ease, box-shadow 0.16s ease;
}

.execution-card.is-focused {
  border-color: var(--scheduled-session-accent);
  box-shadow: 0 0 0 2px rgba(var(--el-color-primary-rgb), 0.16);
}

.execution-card:hover {
  border-color: rgba(var(--el-color-primary-rgb), 0.28);
  box-shadow: 0 8px 20px rgba(15, 23, 42, 0.06);
}

.execution-card-rail {
  background: var(--el-color-info);
  border-radius: 3px 0 0 3px;
}

.execution-card.is-success .execution-card-rail {
  background: var(--el-color-success);
}

.execution-card.is-failed .execution-card-rail,
.execution-card.is-timeout .execution-card-rail {
  background: var(--el-color-danger);
}

.execution-card.is-running .execution-card-rail,
.execution-card.is-queued .execution-card-rail {
  background: var(--el-color-primary);
}

.execution-card-main {
  min-width: 0;
  padding: 12px 14px 14px;
}

.execution-card-head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
}

.execution-card-title-line {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.execution-trigger {
  font-size: 12px;
  color: var(--scheduled-session-muted);
}

.execution-time {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.execution-open-session {
  flex-shrink: 0;
  padding: 0;
}

.execution-facts {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 10px;
}

.execution-facts span {
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  color: var(--scheduled-session-muted);
  background: var(--scheduled-session-tint);
  border: 1px solid color-mix(in srgb, var(--scheduled-session-line) 54%, transparent);
}

.execution-summary {
  margin-top: 10px;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.6;
  word-break: break-word;
}

.execution-error-card {
  margin-top: 10px;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--el-color-danger) 24%, transparent);
  border-radius: 8px;
  background: color-mix(in srgb, var(--el-color-danger) 6%, #fff);
}

.execution-error-title {
  font-size: 13px;
  font-weight: 650;
  color: var(--el-color-danger);
}

.execution-error-hint {
  margin-top: 4px;
  color: var(--el-text-color-regular);
  font-size: 12px;
  line-height: 1.5;
}

.execution-error-detail {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed rgba(245, 108, 108, 0.28);
  color: var(--el-text-color-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.45;
  word-break: break-word;
}

.execution-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
  padding: 0;
}

@media (max-width: 1100px) {
  .scheduled-agent-workspace {
    grid-template-columns: minmax(240px, 300px) minmax(0, 1fr);
  }

  .scheduled-detail-shell {
    grid-template-columns: minmax(0, 1fr) minmax(240px, 280px);
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

  .detail-document,
  .detail-aside {
    overflow: visible;
  }

  .detail-aside {
    border-top: 1px solid var(--scheduled-session-line);
    border-left: 0;
  }

  .detail-document-toolbar,
  .detail-section-head {
    flex-direction: column;
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

  .detail-aside {
    padding: 12px;
  }
}
</style>
