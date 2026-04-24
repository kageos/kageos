<template>
  <el-dialog
    v-model="dialogVisible"
    class="scheduled-agent-task-dialog"
    title="定时执行会话"
    width="760px"
    destroy-on-close
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="104px">
      <el-form-item label="任务名称" prop="name">
        <el-input v-model="form.name" maxlength="100" show-word-limit placeholder="例如：每日巡检" />
      </el-form-item>

      <el-form-item label="工作空间" prop="full_code_path">
        <el-input v-model="form.full_code_path" disabled />
      </el-form-item>

      <el-form-item label="任务目标" prop="goal">
        <el-input
          v-model="form.goal"
          type="textarea"
          :rows="5"
          maxlength="4000"
          show-word-limit
          placeholder="告诉 Agent 到时间后要完成什么"
        />
      </el-form-item>

      <div class="dialog-grid">
        <el-form-item label="模式" prop="mode_code">
          <el-select
            v-model="form.mode_code"
            filterable
            allow-create
            default-first-option
            style="width: 100%"
          >
            <el-option label="dev" value="dev" />
            <el-option label="execute" value="execute" />
            <el-option label="modify" value="modify" />
          </el-select>
        </el-form-item>

        <el-form-item label="LLM">
          <el-select v-model="form.llm_config_id" filterable style="width: 100%">
            <el-option label="默认 LLM" :value="0" />
            <el-option
              v-for="item in llmOptions"
              :key="item.id"
              :label="`${item.name} / ${item.model}`"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
      </div>

      <el-form-item label="文件引用">
        <el-input
          v-model="form.files"
          type="textarea"
          :rows="2"
          placeholder="bucket/object_key，可逗号分隔"
        />
      </el-form-item>

      <el-divider content-position="left">计划</el-divider>

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

      <el-form-item v-if="form.schedule_type === 'cron'" label="Cron" prop="cron_expr">
        <el-input v-model="form.cron_expr" placeholder="如 0 9 * * *" />
      </el-form-item>

      <div v-if="form.schedule_type === 'every'" class="dialog-grid">
        <el-form-item label="间隔秒数" prop="interval_seconds">
          <el-input-number v-model="form.interval_seconds" :min="1" :max="86400" style="width: 100%" />
        </el-form-item>
        <el-form-item label="最多次数">
          <el-input-number v-model="form.max_runs" :min="0" :max="1000000" style="width: 100%" />
        </el-form-item>
      </div>

      <el-form-item v-else label="最多次数">
        <el-input-number v-model="form.max_runs" :min="0" :max="1000000" style="width: 220px" />
      </el-form-item>

      <el-form-item label="超时">
        <el-input-number
          v-model="form.max_duration_seconds"
          :min="0"
          :max="86400"
          style="width: 220px"
        />
        <span class="form-suffix">秒，0 使用服务端默认值</span>
      </el-form-item>

      <el-divider content-position="left">通知</el-divider>

      <el-form-item label="完成通知">
        <div class="notify-form-block">
          <el-radio-group v-model="form.notify_on" class="notify-radio-group">
            <el-radio-button value="none">不通知</el-radio-button>
            <el-radio-button value="all">每次完成</el-radio-button>
            <el-radio-button value="success">仅成功</el-radio-button>
            <el-radio-button value="failed">仅失败</el-radio-button>
          </el-radio-group>

          <div v-if="form.notify_on !== 'none'" class="notify-recipient-area">
            <div class="recipient-block">
              <div class="recipient-block-header">
                <div>
                  <div class="recipient-title">通知用户</div>
                  <div class="recipient-desc">任务执行完成后通知指定用户</div>
                </div>
                <el-button size="small" plain @click="showUserPicker = true">选择用户</el-button>
              </div>
              <div v-if="form.notify_users.length > 0" class="recipient-list">
                <div
                  v-for="username in form.notify_users"
                  :key="`user-${username}`"
                  class="recipient-item"
                >
                  <UserDisplay :username="username" mode="card" layout="horizontal" size="small" />
                  <button type="button" class="recipient-remove" @click="removeNotifyUser(username)">
                    &times;
                  </button>
                </div>
              </div>
              <div v-else class="recipient-empty">未选择用户</div>
            </div>

            <div class="recipient-block">
              <div class="recipient-block-header">
                <div>
                  <div class="recipient-title">通知组织架构</div>
                  <div class="recipient-desc">通知组织下的相关成员</div>
                </div>
                <el-button size="small" plain @click="showDepartmentPicker = true">选择组织</el-button>
              </div>
              <div v-if="form.notify_departments.length > 0" class="recipient-list">
                <div
                  v-for="departmentPath in form.notify_departments"
                  :key="`dept-${departmentPath}`"
                  class="recipient-item"
                >
                  <DepartmentDisplay
                    :full-code-path="departmentPath"
                    mode="card"
                    layout="horizontal"
                    size="small"
                    show-full-path
                  />
                  <button type="button" class="recipient-remove" @click="removeNotifyDepartment(departmentPath)">
                    &times;
                  </button>
                </div>
              </div>
              <div v-else class="recipient-empty">未选择组织架构</div>
            </div>
          </div>
        </div>
      </el-form-item>
    </el-form>

    <UserPickerDialog
      v-model="showUserPicker"
      title="选择通知用户"
      multiple
      :auto-confirm-single="false"
      :initial-usernames="form.notify_users.join(',')"
      @confirm="handleNotifyUsersConfirm"
    />
    <DepartmentPickerDialog
      v-model="showDepartmentPicker"
      title="选择通知组织架构"
      multiple
      :auto-confirm-single="false"
      :initial-paths="form.notify_departments.join(',')"
      @confirm="handleNotifyDepartmentsConfirm"
    />

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { getLLMList, type LLMInfo } from '@/api/agent'
import {
  createScheduledAgentTask,
  type CreateScheduledAgentTaskReq,
  type ScheduledAgentNotifyOn,
  type ScheduledAgentScheduleType
} from '@/api/scheduledAgentTask'
import UserPickerDialog from '@/shared/components/UserPickerDialog.vue'
import DepartmentPickerDialog from '@/shared/components/DepartmentPickerDialog.vue'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import DepartmentDisplay from '@/shared/components/DepartmentDisplay.vue'
import type { UserInfo } from '@/types'
import type { Department } from '@/api/department'
import { getErrorMessage } from '@/utils/apiError'

interface FormState {
  name: string
  full_code_path: string
  goal: string
  mode_code: string
  files: string
  llm_config_id: number
  schedule_type: ScheduledAgentScheduleType
  run_at: string
  cron_expr: string
  interval_seconds: number
  max_runs: number
  max_duration_seconds: number
  notify_users: string[]
  notify_departments: string[]
  notify_on: ScheduledAgentNotifyOn
}

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    fullCodePath: string
    initialGoal?: string
    initialFiles?: string
    initialLLMConfigId?: number
  }>(),
  {
    initialGoal: '',
    initialFiles: '',
    initialLLMConfigId: 0
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'success'): void
}>()

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const formRef = ref<FormInstance>()
const submitting = ref(false)
const llmLoading = ref(false)
const llmOptions = ref<LLMInfo[]>([])
const showUserPicker = ref(false)
const showDepartmentPicker = ref(false)

const form = ref<FormState>(createDefaultForm())

const rules: FormRules<FormState> = {
  name: [{ required: true, message: '请输入任务名称', trigger: 'blur' }],
  full_code_path: [{ required: true, message: '工作空间路径不能为空', trigger: 'blur' }],
  goal: [{ required: true, message: '请输入任务目标', trigger: 'blur' }],
  mode_code: [{ required: true, message: '请选择模式', trigger: 'change' }],
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
        callback()
      }
    }
  ],
  cron_expr: [
    {
      trigger: 'blur',
      validator: (_rule, value, callback) => {
        if (form.value.schedule_type === 'cron' && !String(value || '').trim()) {
          callback(new Error('请输入 Cron 表达式'))
          return
        }
        callback()
      }
    }
  ],
  interval_seconds: [
    {
      trigger: 'blur',
      validator: (_rule, value, callback) => {
        if (form.value.schedule_type === 'every' && Number(value) < 1) {
          callback(new Error('间隔秒数必须大于 0'))
          return
        }
        callback()
      }
    }
  ]
}

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    form.value = createDefaultForm()
    void loadLLMOptions()
  }
)

function createDefaultForm(): FormState {
  const runAt = defaultRunAt()
  const goal = props.initialGoal.trim()
  return {
    name: goal ? buildDefaultName(goal) : '',
    full_code_path: normalizeFullCodePath(props.fullCodePath),
    goal,
    mode_code: 'dev',
    files: props.initialFiles.trim(),
    llm_config_id: props.initialLLMConfigId || 0,
    schedule_type: 'atime',
    run_at: formatLocalDateTime(runAt),
    cron_expr: '',
    interval_seconds: 60,
    max_runs: 0,
    max_duration_seconds: 300,
    notify_users: [],
    notify_departments: [],
    notify_on: 'none'
  }
}

function buildDefaultName(goal: string): string {
  const compact = goal.replace(/\s+/g, ' ').trim()
  return compact.length > 28 ? `${compact.slice(0, 28)}...` : compact
}

function normalizeFullCodePath(value: string): string {
  const trimmed = value.trim()
  if (!trimmed) return ''
  return trimmed.startsWith('/') ? trimmed : `/${trimmed}`
}

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

function formatLocalDateTime(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

function normalizeStringList(values: string[]): string[] {
  return Array.from(new Set(values.map((item) => String(item || '').trim()).filter(Boolean)))
}

function removeNotifyUser(username: string) {
  form.value.notify_users = form.value.notify_users.filter((item) => item !== username)
}

function removeNotifyDepartment(path: string) {
  form.value.notify_departments = form.value.notify_departments.filter((item) => item !== path)
}

function handleNotifyUsersConfirm(users: UserInfo[]) {
  form.value.notify_users = normalizeStringList(users.map((user) => user.username || ''))
}

function handleNotifyDepartmentsConfirm(departments: Department[]) {
  form.value.notify_departments = normalizeStringList(departments.map((department) => department.full_code_path || ''))
}

async function loadLLMOptions() {
  if (llmOptions.value.length > 0 || llmLoading.value) return
  llmLoading.value = true
  try {
    const [mine, market] = await Promise.allSettled([
      getLLMList({ scope: 'mine', page: 1, page_size: 200 }),
      getLLMList({ scope: 'market', page: 1, page_size: 200 })
    ])
    const map = new Map<number, LLMInfo>()
    for (const result of [mine, market]) {
      if (result.status !== 'fulfilled') continue
      for (const item of result.value.configs || []) {
        map.set(item.id, item)
      }
    }
    llmOptions.value = Array.from(map.values())
  } finally {
    llmLoading.value = false
  }
}

function buildPayload(): CreateScheduledAgentTaskReq {
  const payload: CreateScheduledAgentTaskReq = {
    name: form.value.name.trim(),
    full_code_path: normalizeFullCodePath(form.value.full_code_path),
    goal: form.value.goal.trim(),
    mode_code: form.value.mode_code.trim() || 'dev',
    files: form.value.files.trim(),
    llm_config_id: form.value.llm_config_id || 0,
    schedule_type: form.value.schedule_type,
    max_runs: form.value.max_runs || 0,
    notify_users: normalizeStringList(form.value.notify_users),
    notify_departments: normalizeStringList(form.value.notify_departments),
    notify_on: form.value.notify_on
  }

  if (form.value.schedule_type === 'atime') {
    payload.run_at = form.value.run_at.trim()
  } else if (form.value.schedule_type === 'cron') {
    payload.cron_expr = form.value.cron_expr.trim()
  } else {
    payload.interval_seconds = form.value.interval_seconds
  }

  if (form.value.max_duration_seconds > 0) {
    payload.budget_policy = {
      max_duration_seconds: form.value.max_duration_seconds
    }
  }

  return payload
}

function handleClose() {
  dialogVisible.value = false
  emit('update:modelValue', false)
}

async function handleSubmit() {
  if (!formRef.value) return
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    if (
      form.value.notify_on !== 'none' &&
      form.value.notify_users.length === 0 &&
      form.value.notify_departments.length === 0
    ) {
      ElMessage.warning('请选择通知用户或组织架构')
      return
    }

    submitting.value = true
    try {
      await createScheduledAgentTask(buildPayload())
      ElMessage.success('定时会话已创建')
      handleClose()
      emit('success')
    } catch (error) {
      ElMessage.error(getErrorMessage(error, '创建失败'))
    } finally {
      submitting.value = false
    }
  })
}
</script>

<style scoped>
:deep(.scheduled-agent-task-dialog) {
  border-radius: 12px;
  overflow: hidden;
}

:deep(.scheduled-agent-task-dialog .el-dialog__body) {
  padding: 18px 24px 8px;
}

.dialog-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 12px;
}

.form-suffix {
  margin-left: 10px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.notify-form-block {
  width: 100%;
}

.notify-radio-group {
  flex-wrap: wrap;
}

.notify-recipient-area {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-top: 12px;
}

.recipient-block {
  display: flex;
  min-width: 0;
  flex-direction: column;
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-light);
}

.recipient-block-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.recipient-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.recipient-desc,
.recipient-empty {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.recipient-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.recipient-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  min-width: 0;
  padding: 5px 8px;
  border: 1px solid var(--el-border-color);
  border-radius: 999px;
  background: var(--el-bg-color);
}

.recipient-item :deep(.user-display-wrapper),
.recipient-item :deep(.department-display-wrapper) {
  min-width: 0;
}

.recipient-item :deep(.user-name),
.recipient-item :deep(.department-name) {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.recipient-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  color: var(--el-text-color-secondary);
  background: transparent;
  cursor: pointer;
  line-height: 1;
}

.recipient-remove:hover {
  color: var(--el-color-danger);
  background: var(--el-fill-color);
}

@media (max-width: 760px) {
  .dialog-grid,
  .notify-recipient-area {
    grid-template-columns: 1fr;
  }
}
</style>
