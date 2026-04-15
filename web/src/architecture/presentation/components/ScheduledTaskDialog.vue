<template>
  <el-dialog
    v-model="dialogVisible"
    title="定时执行"
    :width="props.tableMode ? '640px' : '520px'"
    destroy-on-close
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
      <el-form-item label="任务名称" prop="name">
        <el-input v-model="form.name" placeholder="例如：每日 9 点发送报表" maxlength="100" show-word-limit />
      </el-form-item>
      <template v-if="props.tableMode && !props.fixedAction && allowedActions.length > 0">
        <el-form-item label="Table 操作" prop="table_action">
          <el-radio-group v-model="form.table_action" class="table-action-group">
            <el-radio-button
              v-for="a in allowedActions"
              :key="a"
              :value="a"
            >
              {{ tableActionLabel(a) }}
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="请求体" prop="payload_json">
          <el-input
            v-model="form.payload_json"
            type="textarea"
            :rows="8"
            placeholder="JSON"
            class="payload-textarea"
          />
          <div class="form-tip">{{ tablePayloadTip }}</div>
        </el-form-item>
      </template>
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
          style="width: 100%"
          :default-value="defaultRunAt()"
          :disabled-date="disabledDate"
        />
      </el-form-item>
      <el-form-item v-if="form.schedule_type === 'cron'" label="Cron 表达式" prop="cron_expr">
        <el-input v-model="form.cron_expr" placeholder="如 0 9 * * *（每天 9 点）" />
        <div class="form-tip">格式：分 时 日 月 周。创建成功后会从下一次命中开始执行，无需设置开始时间。</div>
      </el-form-item>
      <el-form-item v-if="form.schedule_type === 'every'" label="间隔（秒）" prop="interval_seconds">
        <el-input-number v-model="form.interval_seconds" :min="1" :max="86400" placeholder="如 60" style="width: 100%" />
      </el-form-item>
      <div v-if="form.schedule_type === 'every'" class="form-tip-inline">创建成功后立即执行一次，之后按间隔重复执行。</div>
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
import {
  createScheduledTask,
  type CreateScheduledTaskReq,
  type ScheduledTaskAction
} from '@/api/scheduledTask'
import { getErrorMessage } from '@/utils/apiError'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    fullCodePath: string
    /** Form 等：取当前表单提交体；tableMode 下可不传 */
    getPayload?: () => Record<string, unknown> | Promise<Record<string, unknown>>
    /** Table：展示操作类型 + JSON 请求体 */
    tableMode?: boolean
    /** 允许的 table 动作（与回调、权限在父组件已过滤） */
    allowedTableActions?: ScheduledTaskAction[]
    /** 固定动作（用于“新增弹窗/编辑弹窗”内一键创建定时任务） */
    fixedAction?: ScheduledTaskAction
  }>(),
  {
    getPayload: () => ({}),
    tableMode: false,
    allowedTableActions: () => [],
    fixedAction: undefined
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
  table_action: 'table_create' as ScheduledTaskAction,
  payload_json: '',
  schedule_type: 'atime' as 'atime' | 'cron' | 'every',
  run_at: '' as string,
  cron_expr: '',
  interval_seconds: 60,
  max_runs: 0
})

const allowedActions = computed(() =>
  (props.allowedTableActions || []).filter(
    (a) => a === 'table_create' || a === 'table_update' || a === 'table_delete'
  )
)

function tableActionLabel(a: ScheduledTaskAction): string {
  const m: Record<string, string> = {
    table_create: '新增行',
    table_update: '更新行',
    table_delete: '删除行'
  }
  return m[a] ?? a
}

const tablePayloadTip = computed(() => {
  switch (form.value.table_action) {
    case 'table_create':
      return '与表格「新增」提交的数据结构一致（JSON 对象）。'
    case 'table_update':
      return '需包含 id 与 updates；执行前会自动按 id 查询当前行并补齐 old_values，避免定时创建时的旧值过期。'
    case 'table_delete':
      return '格式：{"ids":[1,2,3]}，与批量删除一致。'
    default:
      return ''
  }
})

function defaultPayloadJsonForAction(a: ScheduledTaskAction): string {
  switch (a) {
    case 'table_create':
      return '{\n  \n}'
    case 'table_update':
      return '{\n  "id": 1,\n  "updates": {\n    \n  }\n}'
    case 'table_delete':
      return '{\n  "ids": []\n}'
    default:
      return '{}'
  }
}

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

function parseDateTimeValue(value: string): Date | null {
  const normalized = value.trim().replace(' ', 'T')
  const date = new Date(normalized)
  return Number.isNaN(date.getTime()) ? null : date
}

const rules: FormRules = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  payload_json: [
    {
      validator: (_rule, value, callback) => {
        if (!props.tableMode) {
          callback()
          return
        }
        if (props.fixedAction) {
          callback()
          return
        }
        const raw = (value as string)?.trim()
        if (!raw) {
          callback(new Error('请输入 JSON 请求体'))
          return
        }
        try {
          const parsed = JSON.parse(raw)
          if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
            callback(new Error('请求体须为 JSON 对象'))
            return
          }
          const act = form.value.table_action
          if (act === 'table_delete') {
            const ids = (parsed as { ids?: unknown }).ids
            if (!Array.isArray(ids) || ids.length === 0) {
              callback(new Error('删除需提供非空 ids 数组'))
              return
            }
          }
          if (act === 'table_update') {
            const id = (parsed as { id?: unknown }).id
            if (id === undefined || id === null || id === '') {
              callback(new Error('更新需提供 id'))
              return
            }
            const updates = (parsed as { updates?: unknown }).updates
            if (typeof updates !== 'object' || updates === null || Array.isArray(updates)) {
              callback(new Error('更新需提供 updates 对象'))
              return
            }
          }
          callback()
        } catch {
          callback(new Error('JSON 格式不正确'))
        }
      },
      trigger: 'blur'
    }
  ],
  run_at: [
    {
      trigger: 'change',
      validator: (_rule, value, callback) => {
        if (form.value.schedule_type !== 'atime') {
          callback()
          return
        }
        if (!value) {
          callback(new Error('请选择执行时间'))
          return
        }
        const date = parseDateTimeValue(String(value))
        if (!date) {
          callback(new Error('时间格式不正确'))
          return
        }
        if (date.getTime() < Date.now()) {
          callback(new Error('执行时间不能早于当前时间'))
          return
        }
        callback()
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
      const firstTable: ScheduledTaskAction = allowedActions.value[0] ?? 'table_create'
      form.value = {
        name: '',
        table_action: firstTable,
        payload_json: props.tableMode ? defaultPayloadJsonForAction(firstTable) : '',
        schedule_type: 'atime',
        run_at: formatLocalDateTime(runAt),
        cron_expr: '',
        interval_seconds: 60,
        max_runs: 0
      }
    }
  }
)

watch(
  () => form.value.table_action,
  (act) => {
    if (!props.modelValue || !props.tableMode || !!props.fixedAction) return
    form.value.payload_json = defaultPayloadJsonForAction(act)
  }
)

function formatLocalDateTime(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

function getRunAtValue(): string {
  return form.value.run_at.trim()
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
      let taskPayload: Record<string, unknown>
      let action: ScheduledTaskAction = 'execute'
      if (props.fixedAction) {
        action = props.fixedAction
        taskPayload = await props.getPayload!()
      } else if (props.tableMode) {
        action = form.value.table_action
        taskPayload = JSON.parse(form.value.payload_json.trim()) as Record<string, unknown>
      } else {
        taskPayload = await props.getPayload!()
      }
      const payload: CreateScheduledTaskReq = {
        name: form.value.name.trim(),
        full_code_path: props.fullCodePath,
        action,
        method: 'POST',
        payload: taskPayload,
        schedule_type: form.value.schedule_type,
        max_runs: form.value.max_runs ?? 0
      }
      if (form.value.schedule_type === 'atime') {
        payload.run_at = getRunAtValue()
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
    } catch (error: unknown) {
      ElMessage.error(getErrorMessage(error, '创建失败'))
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
