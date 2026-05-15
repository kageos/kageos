<template>
  <el-dialog
    v-model="dialogVisible"
    class="scheduled-agent-task-dialog"
    modal-class="scheduled-agent-task-dialog-modal"
    width="820px"
    append-to-body
    destroy-on-close
    :z-index="Z_INDEX.scheduledAgentDialog"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <template #header>
      <div class="scheduled-dialog-header">
        <div class="header-orbit">
          <span class="orbit-core"></span>
        </div>
        <div class="header-copy">
          <div class="header-kicker">AGENT AUTOMATION</div>
          <div class="header-title">{{ isEditMode ? '编辑定时会话' : '定时执行会话' }}</div>
          <div class="header-desc">{{ isEditMode ? '修改目标、提示词、附件和调度策略' : '把当前工作台目标封装为可追踪的自动运行任务' }}</div>
        </div>
        <div class="header-status">
          <span class="status-dot"></span>
          {{ form.schedule_type === 'atime' ? '一次执行' : form.schedule_type === 'cron' ? 'Cron 调度' : '循环执行' }}
        </div>
      </div>
    </template>

    <div class="scheduled-agent-console">
    <div class="console-overview">
      <div class="overview-card">
        <span class="overview-label">目录</span>
        <strong>{{ form.full_code_path || '未选择' }}</strong>
      </div>
      <div class="overview-card">
        <span class="overview-label">模式</span>
        <strong>{{ form.mode_code || 'dev' }}</strong>
      </div>
      <div class="overview-card">
        <span class="overview-label">附件</span>
        <strong>{{ scheduledFiles.length }}</strong>
      </div>
    </div>

    <el-form ref="formRef" class="scheduled-agent-form" :model="form" :rules="rules" label-width="104px">
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
            teleported
            popper-class="scheduled-agent-task-popper"
            style="width: 100%"
          >
            <el-option label="dev" value="dev" />
            <el-option label="execute" value="execute" />
            <el-option label="modify" value="modify" />
          </el-select>
        </el-form-item>

        <el-form-item label="LLM">
          <el-select
            v-model="form.llm_config_id"
            filterable
            teleported
            popper-class="scheduled-agent-task-popper"
            style="width: 100%"
          >
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

      <el-form-item label="文件">
        <div class="file-upload-block">
          <el-upload
            class="scheduled-file-upload"
            drag
            multiple
            :auto-upload="false"
            :show-file-list="false"
            :disabled="fileUploading"
            :on-change="handleScheduleFileChange"
          >
            <div class="upload-inner">
              <el-icon class="upload-icon" :class="{ 'is-loading': fileUploading }">
                <UploadFilled />
              </el-icon>
              <div>
                <div class="upload-title">{{ fileUploading ? '正在上传文件...' : '拖拽或点击上传文件' }}</div>
                <div class="upload-desc">上传后会自动生成文件引用，执行时传给 Agent</div>
              </div>
            </div>
          </el-upload>

          <div v-if="scheduledFiles.length > 0" class="scheduled-file-list">
            <div
              v-for="file in scheduledFiles"
              :key="file.ref"
              class="scheduled-file-card"
            >
              <el-icon><DocumentIcon /></el-icon>
              <div class="scheduled-file-meta">
                <strong :title="file.name">{{ file.name }}</strong>
                <span :title="file.ref">{{ file.ref }}{{ file.size ? ` · ${formatFileSize(file.size)}` : '' }}</span>
              </div>
              <button type="button" class="scheduled-file-remove" @click="removeScheduledFile(file.ref)">
                <el-icon><Close /></el-icon>
              </button>
            </div>
          </div>

          <details class="file-ref-details">
            <summary>查看或粘贴文件引用</summary>
            <el-input
              v-model="form.files"
              type="textarea"
              :rows="2"
              placeholder="bucket/object_key，可逗号分隔"
              @blur="syncScheduledFilesFromRefs"
              @change="syncScheduledFilesFromRefs"
            />
          </details>
        </div>
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
          teleported
          popper-class="scheduled-agent-task-popper"
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
    </div>

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
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        {{ isEditMode ? '保存修改' : '确定' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { Close, Document as DocumentIcon, UploadFilled } from '@element-plus/icons-vue'
import { getLLMList, type LLMInfo } from '@/architecture/presentation/context/api/agent'
import {
  createScheduledAgentTask,
  updateScheduledAgentTask,
  type CreateScheduledAgentTaskReq,
  type ScheduledAgentNotifyOn,
  type ScheduledAgentScheduleType,
  type ScheduledAgentTaskItem
} from '@/architecture/presentation/context/api/scheduledAgentTask'
import UserPickerDialog from '@/architecture/presentation/shared/components/UserPickerDialog.vue'
import DepartmentPickerDialog from '@/architecture/presentation/shared/components/DepartmentPickerDialog.vue'
import UserDisplay from '@/architecture/presentation/shared/components/UserDisplay.vue'
import DepartmentDisplay from '@/architecture/presentation/shared/components/DepartmentDisplay.vue'
import type { UserInfo } from '@/architecture/domain/types'
import type { Department } from '@/architecture/presentation/context/api/department'
import { getErrorMessage } from '@/architecture/shared/apiError'
import { uploadFile, notifyUploadComplete, type UploadProgress } from '@/architecture/presentation/context/uploadContext'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'
import { eventBus, WorkspaceEvent } from '@/architecture/presentation/context/eventBusContext'
import { Z_INDEX } from '@/architecture/presentation/constants/zIndex'

const UPLOAD_ROUTER = 'workspace/chat'

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

interface ScheduledFileItem {
  ref: string
  name: string
  size?: number
}

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    fullCodePath: string
    initialGoal?: string
    initialFiles?: string
    initialLLMConfigId?: number
    task?: ScheduledAgentTaskItem | null
  }>(),
  {
    initialGoal: '',
    initialFiles: '',
    initialLLMConfigId: 0,
    task: null
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
const fileUploading = ref(false)
const llmLoading = ref(false)
const llmOptions = ref<LLMInfo[]>([])
const showUserPicker = ref(false)
const showDepartmentPicker = ref(false)
const scheduledFiles = ref<ScheduledFileItem[]>([])
const authStore = useAuthStore()

const form = ref<FormState>(createDefaultForm())
const isEditMode = computed(() => !!props.task?.id)

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
    scheduledFiles.value = buildScheduledFilesFromRefs(form.value.files)
    void loadLLMOptions()
  }
)

function createDefaultForm(): FormState {
  if (props.task) {
    return createFormFromTask(props.task)
  }

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
    max_duration_seconds: 0,
    notify_users: [],
    notify_departments: [],
    notify_on: 'none'
  }
}

function createFormFromTask(task: ScheduledAgentTaskItem): FormState {
  const budgetPolicy = (task.budget_policy || {}) as Record<string, unknown>
  const maxDurationSeconds = typeof budgetPolicy.max_duration_seconds === 'number'
    ? budgetPolicy.max_duration_seconds
    : 0

  return {
    name: task.name || '',
    full_code_path: normalizeFullCodePath(task.full_code_path || props.fullCodePath),
    goal: task.goal || '',
    mode_code: task.mode_code || 'dev',
    files: String(task.files || ''),
    llm_config_id: task.llm_config_id || 0,
    schedule_type: task.schedule_type || 'atime',
    run_at: formatDateTimeForInput(task.run_at) || formatLocalDateTime(defaultRunAt()),
    cron_expr: task.cron_expr || '',
    interval_seconds: task.interval_seconds || 60,
    max_runs: task.max_runs || 0,
    max_duration_seconds: maxDurationSeconds,
    notify_users: normalizeStringList(task.notify_users || []),
    notify_departments: normalizeStringList(task.notify_departments || []),
    notify_on: task.notify_on || 'none'
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

function formatDateTimeForInput(value?: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return String(value).replace('T', ' ').slice(0, 19)
  }
  return formatLocalDateTime(date)
}

function normalizeStringList(values: string[]): string[] {
  return Array.from(new Set(values.map((item) => String(item || '').trim()).filter(Boolean)))
}

function buildScheduledFilesFromRefs(raw: string): ScheduledFileItem[] {
  return normalizeStringList(String(raw || '').split(',')).map((ref) => ({
    ref,
    name: ref.split('/').filter(Boolean).pop() || ref
  }))
}

function applyScheduledFilesToForm() {
  form.value.files = normalizeStringList(scheduledFiles.value.map((file) => file.ref)).join(',')
}

function syncScheduledFilesFromRefs() {
  const existing = new Map(scheduledFiles.value.map((file) => [file.ref, file]))
  scheduledFiles.value = normalizeStringList(form.value.files.split(',')).map((ref) => {
    return existing.get(ref) || {
      ref,
      name: ref.split('/').filter(Boolean).pop() || ref
    }
  })
}

function removeScheduledFile(ref: string) {
  scheduledFiles.value = scheduledFiles.value.filter((file) => file.ref !== ref)
  applyScheduledFilesToForm()
}

function formatFileSize(size: number): string {
  if (!Number.isFinite(size) || size <= 0) return ''
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

async function handleScheduleFileChange(uploadFileObj: { raw?: File }) {
  const file = uploadFileObj?.raw
  if (!file) return

  fileUploading.value = true
  try {
    const uploadResult = await uploadFile(UPLOAD_ROUTER, file, (_progress: UploadProgress) => {})
    if (!uploadResult.fileInfo) {
      throw new Error('上传失败')
    }

    const completeResult = await notifyUploadComplete({
      key: uploadResult.fileInfo.key,
      bucket: uploadResult.fileInfo.bucket,
      success: true,
      router: uploadResult.fileInfo.router,
      file_name: uploadResult.fileInfo.file_name,
      file_size: uploadResult.fileInfo.file_size,
      content_type: uploadResult.fileInfo.content_type,
      hash: uploadResult.fileInfo.hash,
      storage: uploadResult.storage,
      upload_user: authStore.userName || undefined
    })

    if (!completeResult?.download_url) {
      throw new Error('获取文件引用失败')
    }

    const ref = completeResult.ref || uploadResult.fileInfo.ref
    if (!ref) {
      throw new Error('文件引用为空')
    }

    const next = [
      ...scheduledFiles.value,
      {
        ref,
        name: completeResult.file_name || file.name,
        size: completeResult.file_size || file.size
      }
    ]
    const byRef = new Map<string, ScheduledFileItem>()
    for (const item of next) {
      byRef.set(item.ref, item)
    }
    scheduledFiles.value = Array.from(byRef.values())
    applyScheduledFilesToForm()
    ElMessage.success(`已添加：${file.name}`)
  } catch (error: any) {
    ElMessage.error(error?.message || '上传失败')
  } finally {
    fileUploading.value = false
  }
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

  payload.budget_policy = {
    max_duration_seconds: form.value.max_duration_seconds || 0
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
      if (isEditMode.value && props.task) {
        await updateScheduledAgentTask(props.task.id, buildPayload())
        ElMessage.success('定时会话已更新')
      } else {
        await createScheduledAgentTask(buildPayload())
        ElMessage.success('定时会话已创建')
      }
      eventBus.emit(WorkspaceEvent.scheduledAgentTaskCreated, { full_code_path: form.value.full_code_path })
      handleClose()
      emit('success')
    } catch (error) {
      ElMessage.error(getErrorMessage(error, isEditMode.value ? '更新失败' : '创建失败'))
    } finally {
      submitting.value = false
    }
  })
}
</script>

<style scoped>
:global(.scheduled-agent-task-dialog-modal) {
  z-index: var(--aos-z-scheduled-agent-dialog-mask) !important;
  background: rgba(2, 8, 23, 0.42);
  backdrop-filter: blur(8px);
}

:global(.scheduled-agent-task-dialog) {
  overflow: hidden;
  border: 1px solid rgba(14, 165, 233, 0.32);
  border-radius: 22px;
  background:
    radial-gradient(circle at 16% 0%, rgba(14, 165, 233, 0.22), transparent 32%),
    linear-gradient(135deg, rgba(8, 20, 38, 0.96), rgba(12, 20, 36, 0.94) 46%, rgba(4, 12, 24, 0.97));
  box-shadow:
    0 28px 80px rgba(2, 8, 23, 0.48),
    0 0 0 1px rgba(125, 211, 252, 0.08),
    inset 0 1px 0 rgba(255, 255, 255, 0.08);
}

:global(.scheduled-agent-task-dialog .el-dialog__header) {
  padding: 0;
  margin: 0;
}

:global(.scheduled-agent-task-dialog .el-dialog__body) {
  padding: 18px 24px 8px;
}

:global(.scheduled-agent-task-dialog .el-dialog__footer) {
  padding: 12px 24px 22px;
  border-top: 1px solid rgba(125, 211, 252, 0.12);
}

:global(.scheduled-agent-task-dialog .el-dialog__headerbtn) {
  top: 18px;
  right: 18px;
  z-index: 2;
}

:global(.scheduled-agent-task-dialog .el-dialog__headerbtn .el-dialog__close) {
  color: rgba(226, 232, 240, 0.82);
}

:global(.scheduled-agent-task-popper.el-popper) {
  z-index: var(--aos-z-scheduled-agent-dialog-popper) !important;
}

:global(.scheduled-agent-task-popper.el-select__popper),
:global(.scheduled-agent-task-popper.el-picker__popper) {
  border: 1px solid rgba(125, 211, 252, 0.2);
  background:
    radial-gradient(circle at 18% 0%, rgba(14, 165, 233, 0.12), transparent 34%),
    linear-gradient(150deg, rgba(8, 20, 38, 0.98), rgba(12, 20, 36, 0.96));
  box-shadow: 0 20px 52px rgba(2, 8, 23, 0.42), 0 0 24px rgba(14, 165, 233, 0.1);
}

:global(.scheduled-agent-task-popper.el-select__popper .el-select-dropdown),
:global(.scheduled-agent-task-popper.el-picker__popper .el-picker-panel) {
  background: transparent;
}

:global(.scheduled-agent-task-popper.el-select__popper .el-select-dropdown__item) {
  color: rgba(226, 232, 240, 0.82);
}

:global(.scheduled-agent-task-popper.el-select__popper .el-select-dropdown__item.is-hovering),
:global(.scheduled-agent-task-popper.el-select__popper .el-select-dropdown__item:hover) {
  color: #f8fafc;
  background: rgba(14, 165, 233, 0.12);
}

:global(.scheduled-agent-task-popper.el-select__popper .el-select-dropdown__item.is-selected) {
  color: #7dd3fc;
  background: rgba(14, 165, 233, 0.16);
}

:global(.scheduled-agent-task-popper.el-popper .el-popper__arrow::before) {
  border-color: rgba(125, 211, 252, 0.2);
  background: rgba(8, 20, 38, 0.98);
}

.scheduled-dialog-header {
  position: relative;
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 22px 24px 18px;
  color: #e0f2fe;
  border-bottom: 1px solid rgba(125, 211, 252, 0.14);
  background:
    linear-gradient(90deg, rgba(14, 165, 233, 0.16), transparent 62%),
    repeating-linear-gradient(90deg, rgba(125, 211, 252, 0.08) 0 1px, transparent 1px 34px);
}

.scheduled-dialog-header::after {
  content: '';
  position: absolute;
  right: 24px;
  bottom: -1px;
  left: 24px;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(56, 189, 248, 0.72), transparent);
}

.header-orbit {
  position: relative;
  display: grid;
  width: 44px;
  height: 44px;
  flex-shrink: 0;
  place-items: center;
  border: 1px solid rgba(125, 211, 252, 0.42);
  border-radius: 50%;
  box-shadow: 0 0 24px rgba(14, 165, 233, 0.2);
}

.header-orbit::before {
  content: '';
  position: absolute;
  inset: 7px;
  border: 1px dashed rgba(125, 211, 252, 0.36);
  border-radius: 50%;
}

.orbit-core {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #38bdf8;
  box-shadow: 0 0 18px rgba(56, 189, 248, 0.9);
}

.header-copy {
  min-width: 0;
  flex: 1;
}

.header-kicker {
  color: rgba(125, 211, 252, 0.82);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.16em;
}

.header-title {
  margin-top: 4px;
  color: #f8fafc;
  font-size: 22px;
  font-weight: 800;
}

.header-desc {
  margin-top: 4px;
  color: rgba(226, 232, 240, 0.66);
  font-size: 13px;
}

.header-status {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 7px 12px;
  border: 1px solid rgba(14, 165, 233, 0.28);
  border-radius: 999px;
  color: #bae6fd;
  background: rgba(14, 165, 233, 0.1);
  font-size: 12px;
  font-weight: 700;
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #22c55e;
  box-shadow: 0 0 12px rgba(34, 197, 94, 0.85);
}

.scheduled-agent-console {
  max-height: min(72vh, 720px);
  overflow-y: auto;
  padding-right: 4px;
}

.console-overview {
  display: grid;
  grid-template-columns: 1.7fr 0.8fr 0.6fr;
  gap: 12px;
  margin-bottom: 18px;
}

.overview-card {
  min-width: 0;
  padding: 12px 14px;
  border: 1px solid rgba(125, 211, 252, 0.14);
  border-radius: 14px;
  background: rgba(15, 23, 42, 0.62);
}

.overview-card strong {
  display: block;
  margin-top: 5px;
  overflow: hidden;
  color: #e0f2fe;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overview-label {
  color: rgba(148, 163, 184, 0.96);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.scheduled-agent-form :deep(.el-form-item__label) {
  color: rgba(226, 232, 240, 0.78);
  font-weight: 700;
}

.scheduled-agent-form :deep(.el-input__wrapper),
.scheduled-agent-form :deep(.el-textarea__inner),
.scheduled-agent-form :deep(.el-select__wrapper) {
  border: 1px solid rgba(125, 211, 252, 0.16);
  border-radius: 12px;
  background: rgba(15, 23, 42, 0.72);
  box-shadow: none;
}

.scheduled-agent-form :deep(.el-input__inner),
.scheduled-agent-form :deep(.el-textarea__inner) {
  color: #e2e8f0;
}

.scheduled-agent-form :deep(.el-divider__text) {
  color: #7dd3fc;
  background: transparent;
  font-weight: 800;
}

.dialog-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 12px;
}

.form-suffix {
  margin-left: 10px;
  color: rgba(148, 163, 184, 0.9);
  font-size: 12px;
}

.file-upload-block {
  width: 100%;
}

.scheduled-file-upload,
.scheduled-file-upload :deep(.el-upload) {
  width: 100%;
}

.scheduled-file-upload :deep(.el-upload-dragger) {
  width: 100%;
  padding: 16px;
  border: 1px dashed rgba(56, 189, 248, 0.42);
  border-radius: 16px;
  background:
    linear-gradient(135deg, rgba(14, 165, 233, 0.1), rgba(34, 197, 94, 0.05)),
    rgba(15, 23, 42, 0.68);
}

.upload-inner {
  display: flex;
  align-items: center;
  gap: 14px;
  text-align: left;
}

.upload-icon {
  color: #38bdf8;
  font-size: 28px;
}

.upload-title {
  color: #e0f2fe;
  font-weight: 800;
}

.upload-desc {
  margin-top: 4px;
  color: rgba(148, 163, 184, 0.94);
  font-size: 12px;
}

.scheduled-file-list {
  display: grid;
  gap: 8px;
  margin-top: 10px;
}

.scheduled-file-card {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 9px 10px;
  border: 1px solid rgba(125, 211, 252, 0.14);
  border-radius: 12px;
  color: #dbeafe;
  background: rgba(15, 23, 42, 0.58);
}

.scheduled-file-meta {
  min-width: 0;
  flex: 1;
}

.scheduled-file-meta strong,
.scheduled-file-meta span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scheduled-file-meta strong {
  color: #e2e8f0;
  font-size: 13px;
}

.scheduled-file-meta span {
  margin-top: 2px;
  color: rgba(148, 163, 184, 0.9);
  font-size: 12px;
}

.scheduled-file-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: 0;
  border-radius: 9px;
  color: rgba(226, 232, 240, 0.72);
  background: rgba(148, 163, 184, 0.12);
  cursor: pointer;
}

.scheduled-file-remove:hover {
  color: #fecaca;
  background: rgba(239, 68, 68, 0.18);
}

.file-ref-details {
  margin-top: 10px;
  color: rgba(148, 163, 184, 0.92);
  font-size: 12px;
}

.file-ref-details summary {
  margin-bottom: 8px;
  cursor: pointer;
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
  border: 1px solid rgba(125, 211, 252, 0.14);
  border-radius: 14px;
  background: rgba(15, 23, 42, 0.58);
}

.recipient-block-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.recipient-title {
  color: #e0f2fe;
  font-size: 13px;
  font-weight: 700;
}

.recipient-desc,
.recipient-empty {
  color: rgba(148, 163, 184, 0.94);
  font-size: 12px;
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
  border: 1px solid rgba(125, 211, 252, 0.16);
  border-radius: 999px;
  background: rgba(2, 8, 23, 0.32);
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
  .notify-recipient-area,
  .console-overview {
    grid-template-columns: 1fr;
  }
}
</style>
