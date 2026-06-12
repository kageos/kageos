<template>
  <el-dialog
    v-model="dialogVisible"
    class="scheduled-function-dialog"
    title="定时函数"
    width="720px"
    destroy-on-close
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="104px">
      <el-form-item label="任务名称" prop="title">
        <el-input v-model="form.title" maxlength="100" show-word-limit placeholder="例如：每日 9 点生成报表" />
      </el-form-item>

      <el-form-item v-if="!usesExternalPayload" label="执行参数" prop="payload_json">
        <el-input
          v-model="form.payload_json"
          type="textarea"
          :rows="7"
          placeholder='{"keyword":"demo"}'
          class="payload-textarea"
        />
      </el-form-item>

      <el-form-item label="执行方式" prop="schedule_type">
        <el-radio-group v-model="form.schedule_type">
          <el-radio-button value="atime">指定时间一次</el-radio-button>
          <el-radio-button value="cron">Cron 表达式</el-radio-button>
          <el-radio-button value="every">每 N 秒</el-radio-button>
        </el-radio-group>
      </el-form-item>

      <el-form-item v-if="form.schedule_type === 'atime'" label="执行时间" prop="run_at">
        <el-date-picker
          v-model="form.run_at"
          type="datetime"
          placeholder="选择执行时间"
          format="YYYY-MM-DD HH:mm"
          value-format="YYYY-MM-DD HH:mm:ss"
          :shortcuts="dateTimeShortcuts"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item v-if="form.schedule_type === 'cron'" label="Cron" prop="cron_expr">
        <el-input v-model="form.cron_expr" placeholder="0 9 * * *" />
      </el-form-item>

      <div v-if="form.schedule_type === 'every'" class="scheduled-dialog-grid">
        <el-form-item label="间隔秒数" prop="interval_seconds">
          <el-input-number v-model="form.interval_seconds" :min="1" :max="86400" style="width: 100%" />
        </el-form-item>
        <el-form-item label="最多次数">
          <el-input-number v-model="form.max_runs" :min="0" :max="1000000" style="width: 100%" />
        </el-form-item>
      </div>

      <el-form-item v-if="form.schedule_type !== 'every'" label="最多次数">
        <el-input-number v-model="form.max_runs" :min="0" :max="1000000" style="width: 220px" />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        创建
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
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
const usesExternalPayload = computed(() => typeof props.getPayload === 'function')
const dateTimeShortcuts = createRelativeDateTimeShortcuts()

const form = reactive<ScheduledFunctionForm>({
  title: '',
  action: 'execute',
  method: 'POST',
  payload_json: '{}',
  ...createDefaultTimerScheduleForm(),
})

const rules: FormRules = {
  title: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  schedule_type: [{ required: true, message: '请选择执行方式', trigger: 'change' }],
  run_at: [{ required: true, message: '请选择执行时间', trigger: 'change' }],
  cron_expr: [{ required: true, message: '请输入 Cron 表达式', trigger: 'blur' }],
  interval_seconds: [{ required: true, message: '请输入间隔秒数', trigger: 'change' }],
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
  const name = props.functionDetail?.name || props.fullCodePath.split('/').filter(Boolean).pop() || '函数'
  return `${name} 定时任务`
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
    ElMessage.warning('请选择函数')
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
    ElMessage.success('定时函数已创建')
    emit('success')
    dialogVisible.value = false
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '创建失败')
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
.scheduled-dialog-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 12px;
}

.payload-textarea :deep(.el-textarea__inner) {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 12px;
  line-height: 1.6;
}

@media (max-width: 720px) {
  .scheduled-dialog-grid {
    grid-template-columns: 1fr;
  }
}
</style>
