<template>
  <el-dialog
    v-model="dialogVisible"
    title="定时执行"
    width="520px"
    destroy-on-close
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
      <el-form-item label="任务名称" prop="name">
        <el-input v-model="form.name" placeholder="例如：每日 9 点发送报表" maxlength="100" show-word-limit />
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
          value-format="YYYY-MM-DDTHH:mm:ss.000Z"
          style="width: 100%"
          :default-value="defaultRunAt()"
          :disabled-date="disabledDate"
        />
      </el-form-item>
      <el-form-item v-if="form.schedule_type === 'cron'" label="Cron 表达式" prop="cron_expr">
        <el-input v-model="form.cron_expr" placeholder="如 0 9 * * *（每天 9 点）" />
        <div class="form-tip">格式：分 时 日 月 周；以当前时间为起点计算下次执行</div>
      </el-form-item>
      <el-form-item v-if="form.schedule_type === 'every'" label="间隔（秒）" prop="interval_seconds">
        <el-input-number v-model="form.interval_seconds" :min="1" :max="86400" placeholder="如 60" style="width: 100%" />
      </el-form-item>
      <div v-if="form.schedule_type === 'every'" class="form-tip-inline">以当前时间为起点，按间隔重复执行</div>
      <el-form-item v-if="form.schedule_type === 'every'" label="最多执行次数">
        <el-input-number v-model="form.max_runs" :min="0" placeholder="0 表示不限制" style="width: 100%" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { createScheduledTask, type CreateScheduledTaskReq } from '@/api/scheduledTask'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    fullCodePath: string
    getPayload: () => Record<string, unknown>
  }>(),
  {
    getPayload: () => ({})
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'success'): void
}>()

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const formRef = ref<FormInstance>()
const submitting = ref(false)

const form = ref({
  name: '',
  schedule_type: 'atime' as 'atime' | 'cron' | 'every',
  run_at: '' as string,
  cron_expr: '',
  interval_seconds: 60,
  max_runs: 0
})

/** 默认执行时间：当前时间往后 1 分钟（用于日期选择器默认展示） */
function defaultRunAt(): Date {
  const d = new Date()
  d.setMinutes(d.getMinutes() + 1)
  return d
}

function disabledDate(time: Date) {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return time.getTime() < today.getTime()
}

const rules: FormRules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  run_at: [
    {
      required: true,
      message: '请选择执行时间',
      trigger: 'change',
      validator: (_rule, value, callback) => {
        if (form.value.schedule_type === 'atime' && !value) {
          callback(new Error('请选择执行时间'))
        } else if (form.value.schedule_type === 'atime' && value && new Date(value).getTime() < Date.now()) {
          callback(new Error('执行时间不能早于当前时间'))
        } else {
          callback()
        }
      }
    }
  ],
  cron_expr: [
    {
      required: true,
      message: '请输入 Cron 表达式',
      trigger: 'blur',
      validator: (_rule, value, callback) => {
        if (form.value.schedule_type === 'cron' && !value) {
          callback(new Error('请输入 Cron 表达式'))
        } else {
          callback()
        }
      }
    }
  ],
  interval_seconds: [
    {
      required: true,
      message: '请输入间隔秒数',
      trigger: 'blur',
      validator: (_rule, value, callback) => {
        if (form.value.schedule_type === 'every' && (value == null || value < 1)) {
          callback(new Error('间隔至少 1 秒'))
        } else {
          callback()
        }
      }
    }
  ]
}

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      const runAt = new Date()
      runAt.setMinutes(runAt.getMinutes() + 1)
      form.value = {
        name: '',
        schedule_type: 'atime',
        run_at: runAt.toISOString().slice(0, 19) + 'Z',
        cron_expr: '',
        interval_seconds: 60,
        max_runs: 0
      }
    }
  }
)

/** 提交时 atime 用表单选择的 run_at（RFC3339），其他类型用当前时间 */
function getRunAtRFC3339(): string {
  if (form.value.schedule_type === 'atime' && form.value.run_at) {
    const d = new Date(form.value.run_at)
    return d.toISOString().slice(0, 19) + 'Z'
  }
  return new Date().toISOString().slice(0, 19) + 'Z'
}

function handleClose() {
  dialogVisible.value = false
  emit('update:modelValue', false)
}

async function handleSubmit() {
  if (!formRef.value || !props.fullCodePath) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    if (form.value.schedule_type === 'every' && (form.value.interval_seconds == null || form.value.interval_seconds < 1)) {
      ElMessage.warning('请设置间隔秒数')
      return
    }
    submitting.value = true
    try {
      const payload: CreateScheduledTaskReq = {
        name: form.value.name.trim(),
        full_code_path: props.fullCodePath,
        method: 'POST',
        payload: props.getPayload(),
        schedule_type: form.value.schedule_type,
        run_at: getRunAtRFC3339(),
        max_runs: form.value.max_runs ?? 0
      }
      if (form.value.schedule_type === 'cron') {
        payload.cron_expr = form.value.cron_expr?.trim() || ''
      }
      if (form.value.schedule_type === 'every') {
        payload.interval_seconds = form.value.interval_seconds
      }
      await createScheduledTask(payload)
      ElMessage.success('定时任务已创建')
      handleClose()
      emit('success')
    } catch (e: any) {
      ElMessage.error(e?.message || '创建失败')
    } finally {
      submitting.value = false
    }
  })
}
</script>

<style scoped>
.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}
.form-tip-inline {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin: -8px 0 12px 0;
}
</style>
