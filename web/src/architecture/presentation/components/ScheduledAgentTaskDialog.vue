<template>
  <el-dialog
    v-model="dialogVisible"
    class="scheduled-agent-dialog"
    :title="dialogTitle"
    width="680px"
    destroy-on-close
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="104px">
      <el-form-item :label="t('scheduledTask.taskName')" prop="title">
        <el-input v-model="form.title" maxlength="100" show-word-limit :placeholder="t('scheduledTask.agentTaskNamePlaceholder')" />
      </el-form-item>

      <el-form-item :label="t('scheduledTask.agentDescription')" prop="description">
        <el-input
          v-model="form.description"
          type="textarea"
          :autosize="{ minRows: 2, maxRows: 5 }"
          maxlength="500"
          show-word-limit
          :placeholder="t('scheduledTask.agentDescriptionPlaceholder')"
        />
      </el-form-item>

      <el-form-item :label="t('scheduledTask.agentMessage')" prop="message">
        <div
          class="scheduled-agent-composer"
          :class="{ 'is-dragging': dragOver }"
          @paste="onPaste"
          @dragover.prevent="onDragOver"
          @dragleave.prevent="onDragLeave"
          @drop.prevent="onDrop"
        >
          <MiniWorkstationComposer
            variant="schedule"
            :full-code-path="fullCodePath"
            :attached-files="attachedFiles"
            :uploading="uploading"
            :input-text="form.message"
            :sending="false"
            :session-running="false"
            :stopping="false"
            :selected-l-l-m-config-id="form.llm_config_id"
            :llm-list="[]"
            :llm-loading="false"
            :queued-count="0"
            :register-input-ref="registerMessageInputRef"
            :on-l-l-m-select-visible-change="noop"
            :on-file-change="onFileChange"
            :remove-file="removeFile"
            :on-input-enter="noopInputEnter"
            :placeholder="t('scheduledTask.agentMessagePlaceholder')"
            @update:input-text="form.message = $event"
            @update:selected-l-l-m-config-id="form.llm_config_id = $event"
          />
          <div v-if="dragOver" class="scheduled-agent-drop-hint">
            {{ t('scheduledTask.dropUpload') }}
          </div>
        </div>
      </el-form-item>

      <el-form-item :label="t('scheduledTask.scheduleType')" prop="schedule_type">
        <el-radio-group v-model="form.schedule_type">
          <el-radio-button value="atime">{{ t('scheduledTask.scheduleAtime') }}</el-radio-button>
          <el-radio-button value="cron">{{ t('scheduledTask.scheduleCron') }}</el-radio-button>
          <el-radio-button value="every">{{ t('scheduledTask.scheduleEvery') }}</el-radio-button>
        </el-radio-group>
      </el-form-item>

      <el-form-item v-if="form.schedule_type === 'atime'" :label="t('scheduledTask.runAt')" prop="run_at">
        <el-date-picker
          v-model="form.run_at"
          type="datetime"
          :placeholder="t('scheduledTask.runAtPlaceholder')"
          format="YYYY-MM-DD HH:mm"
          value-format="YYYY-MM-DD HH:mm:ss"
          :shortcuts="dateTimeShortcuts"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item v-if="form.schedule_type === 'cron'" :label="t('scheduledTask.cron')" prop="cron_expr">
        <el-input v-model="form.cron_expr" placeholder="0 9 * * *" />
      </el-form-item>

      <div v-if="form.schedule_type === 'every'" class="scheduled-dialog-grid">
        <el-form-item :label="t('scheduledTask.intervalSeconds')" prop="interval_seconds">
          <el-input-number v-model="form.interval_seconds" :min="1" :max="86400" style="width: 100%" />
        </el-form-item>
        <el-form-item :label="t('scheduledTask.maxRuns')">
          <el-input-number v-model="form.max_runs" :min="0" :max="1000000" style="width: 100%" />
        </el-form-item>
      </div>

      <el-form-item v-if="form.schedule_type !== 'every'" :label="t('scheduledTask.maxRuns')">
        <el-input-number v-model="form.max_runs" :min="0" :max="1000000" style="width: 220px" />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">{{ t('common.cancel') }}</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        {{ submitButtonText }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { createTimerTask, updateTimerTask, type TimerTask } from '@/architecture/presentation/context/api/timer'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'
import type { WorkspaceChatMessageFile } from '@/architecture/presentation/context/api/workspace'
import { useMiniWorkstationUploads } from '@/architecture/presentation/composables/useMiniWorkstationUploads'
import { createRelativeDateTimeShortcuts } from '@/architecture/shared/date'
import MiniWorkstationComposer from './MiniWorkstationComposer.vue'
import {
  buildTimerSchedule,
  createDefaultTimerScheduleForm,
  timerScheduleToForm,
  type TimerScheduleForm,
} from './utils/timerSchedule'

interface ScheduledAgentForm extends TimerScheduleForm {
  title: string
  description: string
  message: string
  mode_code: string
  files: string
  llm_config_id: number
  max_duration_seconds: number
}

const props = withDefaults(defineProps<{
  modelValue: boolean
  fullCodePath: string
  initialMessage?: string
  initialFiles?: string
  initialAttachedFiles?: WorkspaceChatMessageFile[]
  initialLLMConfigId?: number
  editTask?: TimerTask | null
}>(), {
  initialMessage: '',
  initialFiles: '',
  initialAttachedFiles: () => [],
  initialLLMConfigId: 0,
  editTask: null,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'success'): void
}>()

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const formRef = ref<FormInstance>()
const submitting = ref(false)
const authStore = useAuthStore()
const { t } = useI18n()
const messageInputRef = ref<{ focus: () => void }>()
const isEditing = computed(() => !!props.editTask)
const dialogTitle = computed(() => isEditing.value ? t('scheduledTask.editAgentDialogTitle') : t('scheduledTask.dialogAgentTitle'))
const submitButtonText = computed(() => isEditing.value ? t('common.save') : t('common.create'))
const dateTimeShortcuts = computed(() => createRelativeDateTimeShortcuts())
const fullCodePathRef = computed(() => resolvedFullCodePath.value)
const resolvedFullCodePath = computed(() => {
  const payloadPath = stringFromRecord(getTaskPayload(props.editTask), 'full_code_path')
  return props.fullCodePath || props.editTask?.resource_key || props.editTask?.source_ref || payloadPath
})
const form = reactive<ScheduledAgentForm>({
  title: '',
  description: '',
  message: '',
  mode_code: 'dev',
  files: '',
  llm_config_id: 0,
  max_duration_seconds: 0,
  ...createDefaultTimerScheduleForm(),
})
const messageTextRef = computed({
  get: () => form.message,
  set: (value: string) => {
    form.message = value
  }
})

const rules: FormRules = {
  title: [{ required: true, message: () => t('scheduledTask.taskTitleRequired'), trigger: 'blur' }],
  message: [{ required: true, message: () => t('scheduledTask.agentMessageRequired'), trigger: 'blur' }],
  schedule_type: [{ required: true, message: () => t('scheduledTask.scheduleTypeRequired'), trigger: 'change' }],
  run_at: [{ required: true, message: () => t('scheduledTask.runAtRequired'), trigger: 'change' }],
  cron_expr: [{ required: true, message: () => t('scheduledTask.cronRequired'), trigger: 'blur' }],
  interval_seconds: [{ required: true, message: () => t('scheduledTask.intervalRequired'), trigger: 'change' }],
}

const {
  attachedFiles,
  uploading,
  dragOver,
  onFileChange,
  removeFile,
  onDragOver,
  onDragLeave,
  onDrop,
  onPaste
} = useMiniWorkstationUploads({
  fullCodePath: fullCodePathRef,
  inputText: messageTextRef,
  inputRef: messageInputRef
})

const attachedFileRefs = computed(() => {
  return attachedFiles.value
    .map((file) => file.ref)
    .filter((ref): ref is string => !!ref)
    .join(',')
})

function resetForm() {
  const task = props.editTask
  const payload = getTaskPayload(task)
  const schedule = task ? timerScheduleToForm(task.schedule) : createDefaultTimerScheduleForm()
  form.message = task ? getTaskMessage(task) : props.initialMessage || ''
  form.description = task?.description?.trim() || ''
  form.title = task?.title || defaultTaskTitle()
  form.mode_code = task ? stringFromRecord(payload, 'mode_code') || 'dev' : 'dev'
  form.files = task ? stringFromRecord(payload, 'files') : props.initialFiles || ''
  attachedFiles.value = task ? [] : props.initialAttachedFiles.map((file) => ({ ...file }))
  form.llm_config_id = task ? numberFromRecord(payload, 'llm_config_id') : props.initialLLMConfigId || 0
  form.max_duration_seconds = task ? numberFromRecord(payload, 'max_duration_seconds') : 0
  form.schedule_type = schedule.schedule_type
  form.run_at = schedule.run_at
  form.cron_expr = schedule.cron_expr
  form.interval_seconds = schedule.interval_seconds
  form.timezone = schedule.timezone
  form.max_runs = schedule.max_runs
  formRef.value?.clearValidate()
}

function registerMessageInputRef(element: { focus: () => void } | null) {
  messageInputRef.value = element || undefined
}

function noop() {}

function noopInputEnter() {}

function defaultTaskTitle(): string {
  const message = (props.initialMessage || '').trim()
  if (message) {
    return `${message.slice(0, 18)}${message.length > 18 ? '…' : ''}`
  }
  const name = resolvedFullCodePath.value.split('/').filter(Boolean).pop() || t('scheduledTask.defaultWorkspaceName')
  return t('scheduledTask.defaultAgentTaskTitle', { name })
}

function buildExecutorPayload(): Record<string, unknown> {
  const payload: Record<string, unknown> = {
    full_code_path: resolvedFullCodePath.value,
    message: form.message.trim(),
    display_content: form.message.trim(),
  }
  const files = attachedFileRefs.value || form.files.trim()
  if (files) {
    payload.files = files
  }
  if (form.mode_code && form.mode_code !== 'dev') {
    payload.mode_code = form.mode_code
  }
  if (form.llm_config_id > 0) {
    payload.llm_config_id = form.llm_config_id
  }
  if (form.max_duration_seconds > 0) {
    payload.max_duration_seconds = form.max_duration_seconds
  }
  return payload
}

function currentUsername(): string {
  return authStore.userName || authStore.user?.username || ''
}

async function handleSubmit() {
  if (!resolvedFullCodePath.value) {
    ElMessage.warning(t('scheduledTask.selectWorkspace'))
    return
  }

  await formRef.value?.validate()

  submitting.value = true
  try {
    const title = form.title.trim()
    const description = form.description.trim()
    const executorPayload = buildExecutorPayload()
    const metadata = {
      ...(props.editTask?.metadata || {}),
      kind: 'scheduled_agent_session',
      mode_code: form.mode_code || 'dev',
    }
    if (props.editTask) {
      await updateTimerTask(props.editTask.id, {
        title,
        description,
        executor_payload: executorPayload,
        metadata,
        schedule: buildTimerSchedule(form),
        source_type: 'agent_session',
        source_ref: resolvedFullCodePath.value,
        resource_scope: 'workspace_directory',
        resource_key: resolvedFullCodePath.value,
      })
      ElMessage.success(t('scheduledTask.savedAgent'))
    } else {
      await createTimerTask({
        title,
        description,
        category: 'scheduled_agent_session',
        tags: ['agent', 'session'],
        executor_key: 'agent.session',
        executor_payload: executorPayload,
        metadata,
        schedule: buildTimerSchedule(form),
        source_type: 'agent_session',
        source_ref: resolvedFullCodePath.value,
        resource_scope: 'workspace_directory',
        resource_key: resolvedFullCodePath.value,
        request_user: currentUsername(),
        created_by: currentUsername(),
      })
      ElMessage.success(t('scheduledTask.createdAgent'))
    }
    emit('success')
    dialogVisible.value = false
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : isEditing.value ? t('scheduledTask.saveFailed') : t('scheduledTask.createFailed'))
  } finally {
    submitting.value = false
  }
}

function getTaskPayload(task?: TimerTask | null): Record<string, unknown> {
  return task?.executor_payload && typeof task.executor_payload === 'object'
    ? task.executor_payload as Record<string, unknown>
    : {}
}

function getTaskMessage(task?: TimerTask | null): string {
  const payload = getTaskPayload(task)
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

function numberFromRecord(record: Record<string, unknown>, key: string): number {
  const value = record[key]
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string' && value.trim()) {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) return parsed
  }
  return 0
}

function handleClose() {
  dialogVisible.value = false
}

watch(
  () => dialogVisible.value,
  (visible) => {
    if (!visible) return
    resetForm()
  },
  { immediate: true }
)
</script>

<style scoped lang="scss">
.scheduled-agent-composer {
  position: relative;
  width: 100%;
}

.scheduled-agent-composer.is-dragging {
  outline: 2px dashed rgba(var(--el-color-primary-rgb), 0.55);
  outline-offset: 4px;
  border-radius: 16px;
}

.scheduled-agent-drop-hint {
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

.scheduled-dialog-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 12px;
}

@media (max-width: 720px) {
  .scheduled-dialog-grid {
    grid-template-columns: 1fr;
  }
}
</style>
