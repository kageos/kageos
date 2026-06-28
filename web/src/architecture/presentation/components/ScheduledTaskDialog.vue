<template>
  <el-dialog
    v-model="dialogVisible"
    class="scheduled-function-dialog"
    :title="t('scheduledTask.dialogFunctionTitle')"
    width="720px"
    destroy-on-close
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="104px">
      <el-form-item :label="t('scheduledTask.taskName')" prop="title">
        <el-input v-model="form.title" maxlength="100" show-word-limit :placeholder="t('scheduledTask.functionTaskNamePlaceholder')" />
      </el-form-item>

      <el-form-item v-if="!usesExternalPayload" :label="t('scheduledTask.payload')" prop="payload_json">
        <el-input
          v-model="form.payload_json"
          type="textarea"
          :rows="7"
          placeholder='{"keyword":"demo"}'
          class="payload-textarea"
        />
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
        {{ t('common.create') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import type { FunctionDetail } from '@/architecture/domain/types'
import { createTimerTask } from '@/architecture/presentation/context/api/timer'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'
import { createRelativeDateTimeShortcuts } from '@/architecture/shared/date'
import {
  buildTimerSchedule,
  createDefaultTimerScheduleForm,
  parseJSONPayload,
  stringifyPayload,
  type TimerScheduleForm,
} from './utils/timerSchedule'

type ScheduledFunctionAction = 'execute' | 'table_create' | 'table_update' | 'table_delete'

interface ScheduledFunctionForm extends TimerScheduleForm {
  title: string
  action: ScheduledFunctionAction
  method: string
  payload_json: string
}

const props = withDefaults(defineProps<{
  modelValue: boolean
  fullCodePath: string
  functionDetail?: FunctionDetail | null
  getPayload?: () => Record<string, unknown> | Promise<Record<string, unknown>>
  tableMode?: boolean
  fixedAction?: ScheduledFunctionAction
}>(), {
  functionDetail: null,
  tableMode: false,
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
const usesExternalPayload = computed(() => typeof props.getPayload === 'function')
const dateTimeShortcuts = computed(() => createRelativeDateTimeShortcuts())

const form = reactive<ScheduledFunctionForm>({
  title: '',
  action: 'execute',
  method: 'POST',
  payload_json: '{}',
  ...createDefaultTimerScheduleForm(),
})

const rules: FormRules = {
  title: [{ required: true, message: () => t('scheduledTask.taskTitleRequired'), trigger: 'blur' }],
  schedule_type: [{ required: true, message: () => t('scheduledTask.scheduleTypeRequired'), trigger: 'change' }],
  run_at: [{ required: true, message: () => t('scheduledTask.runAtRequired'), trigger: 'change' }],
  cron_expr: [{ required: true, message: () => t('scheduledTask.cronRequired'), trigger: 'blur' }],
  interval_seconds: [{ required: true, message: () => t('scheduledTask.intervalRequired'), trigger: 'change' }],
}

function methodForAction(action: ScheduledFunctionAction): string {
  if (action === 'table_update') return 'PUT'
  if (action === 'table_delete') return 'DELETE'
  if (action === 'execute') return props.functionDetail?.method || 'POST'
  return 'POST'
}

function resetForm() {
  const schedule = createDefaultTimerScheduleForm()
  const action = props.fixedAction || 'execute'
  form.title = defaultTaskTitle()
  form.action = action
  form.method = methodForAction(action)
  form.payload_json = stringifyPayload({})
  form.schedule_type = schedule.schedule_type
  form.run_at = schedule.run_at
  form.cron_expr = schedule.cron_expr
  form.interval_seconds = schedule.interval_seconds
  form.timezone = schedule.timezone
  form.max_runs = schedule.max_runs
  formRef.value?.clearValidate()
}

function defaultTaskTitle(): string {
  const name = props.functionDetail?.name || props.fullCodePath.split('/').filter(Boolean).pop() || t('scheduledTask.defaultFunctionName')
  return t('scheduledTask.defaultFunctionTaskTitle', { name })
}

function buildExecutorPayload(payload: unknown): Record<string, unknown> {
  return {
    full_code_path: props.fullCodePath,
    template_type: props.functionDetail?.template_type || '',
    action: form.action,
    method: form.method,
    payload,
  }
}

function currentUsername(): string {
  return authStore.userName || authStore.user?.username || ''
}

async function handleSubmit() {
  if (!props.fullCodePath) {
    ElMessage.warning(t('scheduledTask.selectFunction'))
    return
  }

  await formRef.value?.validate()

  let payload: unknown
  try {
    payload = props.getPayload
      ? await props.getPayload()
      : parseJSONPayload(form.payload_json)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : String(error))
    return
  }

  submitting.value = true
  try {
    await createTimerTask({
      title: form.title.trim(),
      category: 'scheduled_function',
      tags: ['function', form.action],
      executor_key: 'app.function',
      executor_payload: buildExecutorPayload(payload),
      metadata: {
        kind: 'scheduled_function',
        action: form.action,
        method: form.method,
        template_type: props.functionDetail?.template_type || '',
      },
      schedule: buildTimerSchedule(form),
      source_type: 'function',
      source_ref: props.fullCodePath,
      resource_scope: 'function',
      resource_key: props.fullCodePath,
      request_user: currentUsername(),
      created_by: currentUsername(),
    })
    ElMessage.success(t('scheduledTask.createdFunction'))
    emit('success')
    dialogVisible.value = false
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t('scheduledTask.createFailed'))
  } finally {
    submitting.value = false
  }
}

function handleClose() {
  dialogVisible.value = false
}

watch(
  () => dialogVisible.value,
  (visible) => {
    if (visible) {
      resetForm()
    }
  },
  { immediate: true }
)
</script>

<style scoped lang="scss">
.scheduled-function-dialog {
  :deep(.el-dialog__body) {
    padding: 24px 32px;
  }
}

.scheduled-dialog-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 16px;
}

.payload-textarea :deep(.el-textarea__inner) {
  font-family: var(--font-family-mono);
  font-size: 13px;
  line-height: 1.6;
  background-color: var(--bg-tertiary) !important;
}

@media (max-width: 720px) {
  .scheduled-dialog-grid {
    grid-template-columns: 1fr;
  }
}
</style>
