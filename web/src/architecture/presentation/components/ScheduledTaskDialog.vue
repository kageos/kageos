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
      <section class="notify-panel" :class="{ 'is-muted': form.notify_on === 'none' }">
        <div class="notify-panel-header">
          <div class="notify-heading">
            <span class="notify-icon">
              <el-icon><Bell /></el-icon>
            </span>
            <span>
              <span class="notify-title">执行完成通知</span>
              <span class="notify-subtitle">只发布消息事件，投递渠道交给消息系统处理。</span>
            </span>
          </div>
          <el-tag :type="form.notify_on === 'none' ? 'info' : 'success'" effect="light" round>
            {{ notifyConditionLabel(form.notify_on) }}
          </el-tag>
        </div>

        <div class="notify-condition-grid">
          <button
            v-for="option in notifyOptions"
            :key="option.value"
            type="button"
            class="notify-condition-card"
            :class="{ 'is-active': form.notify_on === option.value }"
            @click="form.notify_on = option.value"
          >
            <span class="notify-condition-main">{{ option.label }}</span>
            <span class="notify-condition-desc">{{ option.desc }}</span>
          </button>
        </div>

        <div v-if="form.notify_on !== 'none'" class="notify-recipient-area">
          <div class="recipient-grid">
            <button type="button" class="recipient-card" @click="showUserPicker = true">
              <span class="recipient-card-icon">
                <el-icon><User /></el-icon>
              </span>
              <span class="recipient-card-copy">
                <span class="recipient-card-title">通知用户</span>
                <span class="recipient-card-desc">已选 {{ form.notify_users.length }} 人</span>
              </span>
              <span class="recipient-card-action">选择</span>
            </button>
            <button type="button" class="recipient-card" @click="showDepartmentPicker = true">
              <span class="recipient-card-icon">
                <el-icon><OfficeBuilding /></el-icon>
              </span>
              <span class="recipient-card-copy">
                <span class="recipient-card-title">通知组织架构</span>
                <span class="recipient-card-desc">已选 {{ form.notify_departments.length }} 个组织</span>
              </span>
              <span class="recipient-card-action">选择</span>
            </button>
          </div>

          <div v-if="notifyTargetCount > 0" class="notify-selected">
            <span class="notify-selected-label">接收对象</span>
            <div class="notify-tags">
              <el-tag
                v-for="username in form.notify_users"
                :key="`user-${username}`"
                class="notify-tag"
                closable
                effect="plain"
                @close="removeNotifyUser(username)"
              >
                用户：{{ username }}
              </el-tag>
              <el-tag
                v-for="departmentPath in form.notify_departments"
                :key="`dept-${departmentPath}`"
                class="notify-tag"
                type="success"
                closable
                effect="plain"
                @close="removeNotifyDepartment(departmentPath)"
              >
                组织：{{ departmentLabel(departmentPath) }}
              </el-tag>
            </div>
          </div>
          <div v-else class="notify-empty">
            <el-icon><WarningFilled /></el-icon>
            <span>请选择至少一个用户或组织架构。</span>
          </div>
        </div>
      </section>
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
  type CreateScheduledTaskReq,
  type ScheduledTaskAction,
  type ScheduledTaskNotifyOn
} from '@/api/scheduledTask'
import { getErrorMessage } from '@/utils/apiError'
import UserPickerDialog from '@/shared/components/UserPickerDialog.vue'
import DepartmentPickerDialog from '@/shared/components/DepartmentPickerDialog.vue'
import type { UserInfo } from '@/types'
import type { Department } from '@/api/department'
import { Bell, OfficeBuilding, User, WarningFilled } from '@element-plus/icons-vue'

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

const notifyOptions: Array<{ value: ScheduledTaskNotifyOn; label: string; desc: string }> = [
  { value: 'none', label: '不通知', desc: '任务完成后不触发消息' },
  { value: 'all', label: '每次完成', desc: '成功或失败都通知' },
  { value: 'success', label: '仅成功', desc: '只有执行成功才通知' },
  { value: 'failed', label: '仅失败', desc: '只在失败时提醒处理' }
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

function departmentLabel(path: string): string {
  const parts = path.split('/').filter(Boolean)
  return parts[parts.length - 1] || path
}

function notifyConditionLabel(value: ScheduledTaskNotifyOn): string {
  return notifyOptions.find((option) => option.value === value)?.label || '不通知'
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
        method: 'POST',
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
  border-radius: 18px;
  overflow: hidden;
}

:deep(.scheduled-task-dialog .el-dialog__header) {
  padding: 22px 26px 10px;
  background:
    radial-gradient(circle at 18% 0%, rgba(64, 158, 255, 0.14), transparent 34%),
    linear-gradient(135deg, rgba(15, 23, 42, 0.03), rgba(64, 158, 255, 0.05));
}

:deep(.scheduled-task-dialog .el-dialog__title) {
  font-size: 18px;
  font-weight: 750;
  letter-spacing: 0.01em;
}

:deep(.scheduled-task-dialog .el-dialog__body) {
  padding: 20px 26px 8px;
}

:deep(.scheduled-task-dialog .el-dialog__footer) {
  padding: 14px 26px 22px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-blank);
}

:deep(.scheduled-task-dialog .el-form-item__label) {
  font-weight: 650;
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

.notify-panel {
  margin-top: 20px;
  padding: 18px;
  border: 1px solid rgba(64, 158, 255, 0.18);
  border-radius: 18px;
  background:
    linear-gradient(135deg, rgba(64, 158, 255, 0.08), rgba(103, 194, 58, 0.04)),
    var(--el-fill-color-blank);
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.06);
}

.notify-panel.is-muted {
  border-color: var(--el-border-color-lighter);
  background:
    linear-gradient(135deg, rgba(148, 163, 184, 0.08), rgba(255, 255, 255, 0.02)),
    var(--el-fill-color-blank);
  box-shadow: none;
}

.notify-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 16px;
}

.notify-heading {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.notify-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 14px;
  color: var(--el-color-primary);
  background: rgba(64, 158, 255, 0.13);
  box-shadow: inset 0 0 0 1px rgba(64, 158, 255, 0.16);
}

.notify-title,
.notify-subtitle {
  display: block;
}

.notify-title {
  font-size: 15px;
  font-weight: 750;
  color: var(--el-text-color-primary);
}

.notify-subtitle {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
}

.notify-condition-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.notify-condition-card,
.recipient-card {
  border: 0;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.notify-condition-card {
  min-height: 78px;
  padding: 12px;
  border-radius: 14px;
  color: var(--el-text-color-regular);
  background: rgba(255, 255, 255, 0.72);
  box-shadow: inset 0 0 0 1px var(--el-border-color-lighter);
  transition: transform 0.16s ease, box-shadow 0.16s ease, background 0.16s ease;
}

.notify-condition-card:hover {
  transform: translateY(-1px);
  box-shadow: inset 0 0 0 1px rgba(64, 158, 255, 0.36), 0 10px 22px rgba(15, 23, 42, 0.08);
}

.notify-condition-card.is-active {
  background:
    linear-gradient(135deg, rgba(64, 158, 255, 0.18), rgba(64, 158, 255, 0.06)),
    var(--el-fill-color-blank);
  box-shadow: inset 0 0 0 1px rgba(64, 158, 255, 0.55), 0 12px 26px rgba(64, 158, 255, 0.14);
}

.notify-condition-main,
.notify-condition-desc {
  display: block;
}

.notify-condition-main {
  font-size: 13px;
  font-weight: 750;
  color: var(--el-text-color-primary);
}

.notify-condition-desc {
  margin-top: 7px;
  font-size: 12px;
  line-height: 1.45;
  color: var(--el-text-color-secondary);
}

.notify-recipient-area {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px dashed rgba(64, 158, 255, 0.24);
}

.recipient-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.recipient-card {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 14px;
  border-radius: 16px;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-blank);
  box-shadow: inset 0 0 0 1px var(--el-border-color-lighter);
  transition: transform 0.16s ease, box-shadow 0.16s ease;
}

.recipient-card:hover {
  transform: translateY(-1px);
  box-shadow: inset 0 0 0 1px rgba(64, 158, 255, 0.38), 0 10px 22px rgba(15, 23, 42, 0.08);
}

.recipient-card-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  width: 36px;
  height: 36px;
  border-radius: 12px;
  color: var(--el-color-primary);
  background: rgba(64, 158, 255, 0.1);
}

.recipient-card-copy {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.recipient-card-title {
  font-size: 13px;
  font-weight: 750;
}

.recipient-card-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.recipient-card-action {
  flex: 0 0 auto;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-color-primary);
}

.notify-selected {
  margin-top: 14px;
  padding: 12px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.7);
  box-shadow: inset 0 0 0 1px var(--el-border-color-extra-light);
}

.notify-selected-label {
  display: block;
  margin-bottom: 9px;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}

.notify-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.notify-tag {
  max-width: 100%;
}

.notify-empty {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
  padding: 12px 14px;
  border-radius: 14px;
  color: var(--el-color-warning);
  background: rgba(230, 162, 60, 0.1);
}

@media (max-width: 760px) {
  .notify-panel-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .notify-condition-grid,
  .recipient-grid {
    grid-template-columns: 1fr;
  }
}
</style>
