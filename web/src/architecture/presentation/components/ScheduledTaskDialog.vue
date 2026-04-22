<template>
  <el-dialog
    v-model="dialogVisible"
    class="scheduled-task-dialog"
    title="定时执行"
    :width="props.tableMode ? '760px' : '720px'"
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
      <el-form-item label="完成通知">
        <div class="notify-form-block">
          <el-radio-group v-model="form.notify_on" class="notify-radio-group">
            <el-radio-button
              v-for="option in notifyOptions"
              :key="option.value"
              :value="option.value"
            >
              {{ option.label }}
            </el-radio-button>
          </el-radio-group>
          <div class="form-tip">只发布消息事件，通知渠道由消息系统处理。</div>

          <div v-if="form.notify_on !== 'none'" class="notify-recipient-area">
            <div class="recipient-block">
              <div class="recipient-block-header">
                <div>
                  <div class="recipient-title">通知用户</div>
                  <div class="recipient-desc">任务执行完成后通知指定用户</div>
                </div>
                <el-button size="small" plain @click="showUserPicker = true">
                  选择用户
                </el-button>
              </div>
              <div v-if="form.notify_users.length > 0" class="recipient-list">
                <div
                  v-for="username in form.notify_users"
                  :key="`user-${username}`"
                  class="recipient-item"
                >
                  <UserDisplay
                    :username="username"
                    mode="card"
                    layout="horizontal"
                    size="small"
                  />
                  <button
                    type="button"
                    class="recipient-remove"
                    aria-label="移除用户"
                    @click="removeNotifyUser(username)"
                  >
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
                <el-button size="small" plain @click="showDepartmentPicker = true">
                  选择组织
                </el-button>
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
                  <button
                    type="button"
                    class="recipient-remove"
                    aria-label="移除组织"
                    @click="removeNotifyDepartment(departmentPath)"
                  >
                    &times;
                  </button>
                </div>
              </div>
              <div v-else class="recipient-empty">未选择组织架构</div>
            </div>

            <el-alert
              v-if="notifyTargetCount === 0"
              title="开启通知后至少选择一个用户或组织架构"
              type="warning"
              show-icon
              :closable="false"
              class="notify-warning"
            />
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
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import {
  createScheduledTask,
  methodForScheduledTaskAction,
  type CreateScheduledTaskReq,
  type ScheduledTaskAction,
  type ScheduledTaskNotifyOn
} from '@/api/scheduledTask'
import { getErrorMessage } from '@/utils/apiError'
import UserPickerDialog from '@/shared/components/UserPickerDialog.vue'
import DepartmentPickerDialog from '@/shared/components/DepartmentPickerDialog.vue'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import DepartmentDisplay from '@/shared/components/DepartmentDisplay.vue'
import type { UserInfo } from '@/types'
import type { Department } from '@/api/department'

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
const showUserPicker = ref(false)
const showDepartmentPicker = ref(false)

const notifyOptions: Array<{ value: ScheduledTaskNotifyOn; label: string }> = [
  { value: 'none', label: '不通知' },
  { value: 'all', label: '每次完成' },
  { value: 'success', label: '仅成功' },
  { value: 'failed', label: '仅失败' }
]

const form = ref({
  name: '',
  table_action: 'table_create' as ScheduledTaskAction,
  payload_json: '',
  schedule_type: 'atime' as 'atime' | 'cron' | 'every',
  run_at: '' as string,
  cron_expr: '',
  interval_seconds: 60,
  max_runs: 0,
  notify_users: [] as string[],
  notify_departments: [] as string[],
  notify_on: 'none' as ScheduledTaskNotifyOn
})

const notifyTargetCount = computed(() => form.value.notify_users.length + form.value.notify_departments.length)

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
        max_runs: 0,
        notify_users: [],
        notify_departments: [],
        notify_on: 'none'
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

function removeNotifyUser(username: string) {
  form.value.notify_users = form.value.notify_users.filter((item) => item !== username)
}

function removeNotifyDepartment(path: string) {
  form.value.notify_departments = form.value.notify_departments.filter((item) => item !== path)
}

function normalizeStringList(values: string[]): string[] {
  return Array.from(new Set(values.map((item) => item.trim()).filter(Boolean)))
}

function handleNotifyUsersConfirm(users: UserInfo[]) {
  form.value.notify_users = normalizeStringList(users.map((user) => user.username || ''))
}

function handleNotifyDepartmentsConfirm(departments: Department[]) {
  form.value.notify_departments = normalizeStringList(departments.map((department) => department.full_code_path || ''))
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
        method: methodForScheduledTaskAction(action),
        payload: taskPayload,
        schedule_type: form.value.schedule_type,
        max_runs: form.value.max_runs ?? 0,
        notify_users: form.value.notify_users,
        notify_departments: form.value.notify_departments,
        notify_on: form.value.notify_on
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
:deep(.scheduled-task-dialog) {
  border-radius: 12px;
  overflow: hidden;
}

:deep(.scheduled-task-dialog .el-dialog__header) {
  padding: 20px 24px 10px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

:deep(.scheduled-task-dialog .el-dialog__title) {
  font-size: 18px;
  font-weight: 600;
}

:deep(.scheduled-task-dialog .el-dialog__body) {
  padding: 20px 24px 8px;
}

:deep(.scheduled-task-dialog .el-dialog__footer) {
  padding: 14px 24px 20px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}

:deep(.scheduled-task-dialog .el-form-item__label) {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

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
  border-radius: 10px;
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
  font-weight: 600;
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

.notify-warning {
  grid-column: 1 / -1;
}

@media (max-width: 760px) {
  .notify-recipient-area {
    grid-template-columns: 1fr;
  }
}
</style>
