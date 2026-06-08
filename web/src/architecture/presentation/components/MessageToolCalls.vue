<!--
  MessageToolCalls - 连续多个工具时，每个工具一块：工具名+状态在「该块」上方，下面是对应 viewport
  - 顶部仅执行中时显示：正在执行工具…
  - 每个工具：块头写工具名+状态（如 read_go_file 成功），下面才是该工具的 viewport，不把所有工具名挤在顶部
  - 输出文件仍在下方独立展示
-->
<template>
  <div class="message-tool-calls">
    <div v-if="hasRunning" class="message-tool-calls-head">
      <span class="head-status head-status--running">
        <el-icon class="head-icon head-icon--spin"><Loading /></el-icon>
        正在执行工具…
      </span>
    </div>
    <template v-for="(tc, idx) in toolCalls" :key="tc.id ?? `${idx}-${tc.name}`">
      <div :class="['message-tool-calls-block', { 'message-tool-calls-block--first': idx === 0 && !hasRunning }]">
        <div class="message-tool-calls-block-head">
          <el-icon v-if="tc.status === 'ok'" class="block-head-icon block-head-icon--ok"><CircleCheck /></el-icon>
          <el-icon v-else-if="tc.status === 'error'" class="block-head-icon block-head-icon--error"><CircleClose /></el-icon>
          <el-icon v-else class="block-head-icon block-head-icon--running"><Loading /></el-icon>
          <span class="block-head-name">{{ tc.name }}</span>
          <el-tag v-if="tc.status === 'streaming'" type="info" size="small" class="block-head-status">解析中</el-tag>
          <el-tag v-else-if="tc.status === 'running'" type="info" size="small" class="block-head-status">执行中</el-tag>
          <el-tag v-else :type="tc.status === 'ok' ? 'success' : 'danger'" size="small" class="block-head-status">
            {{ statusLabel(tc.status) }}
          </el-tag>
          <span
            v-if="getToolDurationLabel(tc, idx)"
            :class="['block-head-duration', { 'block-head-duration--running': isToolTimerRunning(tc, idx) }]"
          >
            <span v-if="isToolTimerRunning(tc, idx)" class="block-head-duration-dot"></span>
            耗时 {{ getToolDurationLabel(tc, idx) }}
          </span>
        </div>
        <div v-if="isRenderablePrdToolCall(tc)" class="message-tool-calls-prd mini-msg-prd-preview">
          <PrdPreview
            :data="tc.result_data"
            :confirm-disabled="confirmDisabled"
            @confirm="emit('confirm-prd', $event)"
          />
        </div>
        <RoleHandoffCard
          v-else-if="isRoleHandoffToolCall(tc)"
          :tool-call="tc"
        />
        <BuildWorkspaceDiagnosticsCard
          v-else-if="isBuildWorkspaceFailureToolCall(tc)"
          :tool-call="tc"
          class="mini-msg-build-diagnostics"
        />
        <div
          v-else
          :class="['message-tool-calls-viewport', { 'message-tool-calls-viewport--first': idx === 0 }]"
          :ref="(el) => setViewportRef(el as HTMLElement | null, idx)"
        >
          <div class="message-tool-calls-output">
            <div
              v-for="(line, lineIdx) in getLinesForTool(tc)"
              :key="lineIdx"
              :class="['output-line', line.type]"
            >
              {{ line.text }}
            </div>
            <div
              v-if="hasRunning && idx === toolCalls.length - 1"
              class="output-line output-line--cursor"
            >
              ▌
            </div>
          </div>
        </div>
      </div>
    </template>
    <OutputFilesDisplay
      v-if="fileGroups.length > 0"
      :file-groups="fileGroups"
      class="message-output-files"
    />
    <OutputDisplayFields
      v-if="displayFields.length > 0"
      :fields="displayFields"
      class="message-output-display-fields"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { Loading, CircleCheck, CircleClose } from '@element-plus/icons-vue'
import OutputFilesDisplay from './OutputFilesDisplay.vue'
import OutputDisplayFields from './OutputDisplayFields.vue'
import PrdPreview from './PrdPreview.vue'
import RoleHandoffCard from './RoleHandoffCard.vue'
import BuildWorkspaceDiagnosticsCard from './BuildWorkspaceDiagnosticsCard.vue'
import type { WorkspaceChatToolCallSummary } from '@/architecture/presentation/context/api/workspace'
import type { OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'
import { extractAllDisplayFields } from '@/architecture/presentation/composables/useOutputDisplayFields'

const props = withDefaults(defineProps<{
  toolCalls: WorkspaceChatToolCallSummary[]
  fileGroups: OutputFileGroup[]
  confirmDisabled?: boolean
}>(), {
  confirmDisabled: false,
})

const emit = defineEmits<{
  (e: 'confirm-prd', payload: { remark: string; prd: unknown }): void
}>()

const displayFields = computed(() => extractAllDisplayFields(props.toolCalls))

/** 每个工具一个 viewport，用数组存 DOM，滚动时滚最后一个 */
const viewportRefs = ref<(HTMLElement | null)[]>([])

function setViewportRef(el: HTMLElement | null, idx: number) {
  const arr = viewportRefs.value
  if (el) {
    while (arr.length <= idx) arr.push(null)
    arr[idx] = el
    // 不在这里 trim，避免 Vue ref 回调乱序时把后面已设置的 ref 清掉导致错乱
  } else {
    viewportRefs.value = arr.slice(0, idx)
  }
}

const hasRunning = computed(() =>
  props.toolCalls.some((t) => t.status === 'streaming' || t.status === 'running')
)

interface RuntimeTimer {
  startedAt: number
  completedAt?: number
}

const toolTimers = new Map<string, RuntimeTimer>()
const timerNow = ref(Date.now())
let timerInterval: ReturnType<typeof setInterval> | null = null

function isToolPending(status: string): boolean {
  return status === 'streaming' || status === 'running'
}

function isToolDone(status: string): boolean {
  return status === 'ok' || status === 'error'
}

function getToolTimerKey(tc: WorkspaceChatToolCallSummary, idx: number): string {
  return tc.id ? `id:${tc.id}` : `idx:${idx}:${tc.name}`
}

function hasActiveToolTimer(): boolean {
  for (const timer of toolTimers.values()) {
    if (timer.completedAt == null) return true
  }
  return false
}

function syncTimerInterval() {
  if (hasActiveToolTimer()) {
    if (timerInterval == null) {
      timerInterval = setInterval(() => {
        timerNow.value = Date.now()
      }, 250)
    }
    return
  }
  if (timerInterval != null) {
    clearInterval(timerInterval)
    timerInterval = null
  }
}

function syncToolTimers() {
  const now = Date.now()
  const visibleKeys = new Set<string>()
  let changed = false

  props.toolCalls.forEach((tc, idx) => {
    const key = getToolTimerKey(tc, idx)
    visibleKeys.add(key)
    const existing = toolTimers.get(key)

    if (isToolPending(tc.status)) {
      if (!existing) {
        toolTimers.set(key, { startedAt: now })
        changed = true
      } else if (existing.completedAt != null) {
        delete existing.completedAt
        changed = true
      }
      return
    }

    if (isToolDone(tc.status) && existing && existing.completedAt == null) {
      existing.completedAt = now
      changed = true
    }
  })

  for (const key of toolTimers.keys()) {
    if (!visibleKeys.has(key)) {
      toolTimers.delete(key)
      changed = true
    }
  }

  if (changed) timerNow.value = now
  syncTimerInterval()
}

function formatDuration(ms: number): string {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000))
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  if (hours > 0) return `${hours}时${minutes}分${seconds}秒`
  if (minutes > 0) return `${minutes}分${seconds}秒`
  return `${seconds}秒`
}

function getToolDurationLabel(tc: WorkspaceChatToolCallSummary, idx: number): string {
  const timer = toolTimers.get(getToolTimerKey(tc, idx))
  if (!timer) return ''
  const end = timer.completedAt ?? timerNow.value
  return formatDuration(end - timer.startedAt)
}

function isToolTimerRunning(tc: WorkspaceChatToolCallSummary, idx: number): boolean {
  const timer = toolTimers.get(getToolTimerKey(tc, idx))
  return !!timer && timer.completedAt == null && isToolPending(tc.status)
}

/** 状态中文 */
function statusLabel(status: string): string {
  if (status === 'streaming') return '解析中…'
  if (status === 'running') return '执行中…'
  if (status === 'ok') return '成功'
  if (status === 'error') return '失败'
  return status
}

function isRenderablePrdToolCall(tc: WorkspaceChatToolCallSummary): boolean {
  return tc.name === 'write_prd' && tc.status === 'ok' && tc.result_data != null
}

function isRoleHandoffToolCall(tc: WorkspaceChatToolCallSummary): boolean {
  return tc.name === 'change_role'
}

function isBuildWorkspaceFailureToolCall(tc: WorkspaceChatToolCallSummary): boolean {
  return tc.name === 'build_workspace' &&
    tc.result_data != null &&
    typeof tc.result_data === 'object' &&
    (tc.result_data as { kind?: string }).kind === 'agent_app_build_failure'
}

/** 把字符串里字面的 \n、\r 转成真实换行，这样嵌套 JSON 里的换行能正确展示 */
function unescapeNewlines(s: string): string {
  return s.replace(/\\n/g, '\n').replace(/\\r/g, '\r')
}

/** 结果/错误：仅还原转义的换行，不截断，完整内容在视口内滚动查看 */
function formatResultOrError(s: string): string {
  if (!s || !s.trim()) return ''
  return unescapeNewlines(s.trim())
}

function formatResultData(val: unknown): string {
  if (val == null) return ''
  try {
    return JSON.stringify(val, null, 2)
  } catch {
    return String(val)
  }
}

/** 参数：多行展示（pretty JSON），仅还原换行，不截断，完整内容在视口内滚动查看 */
function formatArgsPreview(argsStr: string | undefined): string {
  if (!argsStr || !argsStr.trim()) return ''
  const trimmed = argsStr.trim()
  let multiLine: string
  try {
    const obj = JSON.parse(trimmed)
    multiLine = JSON.stringify(obj, null, 2)
    multiLine = unescapeNewlines(multiLine)
  } catch {
    multiLine = unescapeNewlines(trimmed)
  }
  return multiLine
}

/** 单个工具的输出行（工具名+状态已在块头展示，这里只含参数/结果/错误） */
function getLinesForTool(tc: WorkspaceChatToolCallSummary): { text: string; type: string }[] {
  const lines: { text: string; type: string }[] = []
  if (tc.arguments) {
    const argsPreview = formatArgsPreview(tc.arguments)
    if (argsPreview) lines.push({ text: `参数: ${argsPreview}`, type: 'args' })
  }
  if (tc.result) {
    const preview = formatResultOrError(tc.result)
    if (preview) lines.push({ text: preview, type: 'result' })
  } else if (tc.result_data != null) {
    const preview = formatResultData(tc.result_data)
    if (preview) lines.push({ text: preview, type: 'result' })
  }
  if (tc.error) {
    const preview = formatResultOrError(tc.error)
    if (preview) lines.push({ text: `错误: ${preview}`, type: 'error' })
  }
  return lines
}

/** 只用 scrollTop 滚 viewport 自身，不用 scrollIntoView 避免带动外层容器跳动 */
let _vpRafId = 0
function scrollAllViewportsToBottom() {
  if (_vpRafId) return
  _vpRafId = requestAnimationFrame(() => {
    _vpRafId = 0
    const arr = viewportRefs.value
    for (const vp of arr) {
      if (vp) vp.scrollTop = vp.scrollHeight - vp.clientHeight
    }
  })
}

watch(
  () => props.toolCalls.map((t) => {
    const resultDataLen = t.result_data == null ? 0 : formatResultData(t.result_data).length
    return (t.arguments?.length ?? 0) + (t.result?.length ?? 0) + resultDataLen + (t.error?.length ?? 0)
  }),
  () => scrollAllViewportsToBottom(),
  { deep: true }
)
watch(
  () => props.toolCalls.length,
  (len) => {
    const arr = viewportRefs.value
    if (arr.length > len) viewportRefs.value = arr.slice(0, len)
    scrollAllViewportsToBottom()
  }
)
watch(hasRunning, (running) => {
  if (running) scrollAllViewportsToBottom()
})
watch(
  () => props.toolCalls.map((t, idx) => `${getToolTimerKey(t, idx)}:${t.status}`).join('|'),
  syncToolTimers,
  { immediate: true }
)
onBeforeUnmount(() => {
  if (timerInterval != null) clearInterval(timerInterval)
})
</script>

<style scoped lang="scss">
.message-tool-calls {
  width: 100%;
  margin-top: 8px;
}

.message-tool-calls-head {
  padding: 6px 10px 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-lighter);
  border-radius: var(--el-border-radius-base) var(--el-border-radius-base) 0 0;
}

.head-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
}

.head-status--running {
  color: var(--el-color-primary);
}

.head-icon {
  font-size: 14px;
}

.head-icon--spin {
  animation: think-spin 0.8s linear infinite;
}
@keyframes think-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* 每个工具一块：块头（工具名+状态）+ viewport */
.message-tool-calls-block {
  margin-top: 8px;
}
.message-tool-calls-block--first {
  margin-top: 0;
}

.message-tool-calls-block-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px 8px;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base) var(--el-border-radius-base) 0 0;
  font-size: 13px;
}

.block-head-icon {
  font-size: 14px;
  flex-shrink: 0;
}
.block-head-icon--ok {
  color: var(--el-color-success);
}
.block-head-icon--error {
  color: var(--el-color-danger);
}
.block-head-icon--running {
  color: var(--el-color-primary);
  animation: think-spin 0.8s linear infinite;
}

.block-head-name {
  min-width: 0;
  overflow: hidden;
  font-weight: 500;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.block-head-status {
  flex-shrink: 0;
}

.block-head-duration {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  margin-left: auto;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1;
}

.block-head-duration--running {
  color: var(--el-color-primary);
}

.block-head-duration-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  box-shadow: 0 0 8px rgba(64, 158, 255, 0.42);
}

/* 每个工具一块 viewport，约 4-5 行高度，内容可滚动 */
.message-tool-calls-viewport {
  max-height: 7.5em;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px 12px;
  background: var(--el-fill-color-blank);
  border: 1px solid var(--el-border-color-lighter);
  border-top: none;
  border-radius: 0 0 var(--el-border-radius-base) var(--el-border-radius-base);
  line-height: 1.5;
  font-size: 12px;
  color: var(--el-text-color-regular);
}
.message-tool-calls-viewport--first {
  /* 紧接在块头下方，样式已由 border-top: none 体现 */
}

.message-tool-calls-prd {
  :deep(.prd-preview) {
    margin-top: 0;
    border-top: none;
    border-radius: 0 0 var(--el-border-radius-base) var(--el-border-radius-base);
  }
}

.message-tool-calls-block :deep(.role-handoff-card) {
  border-top: none;
  border-radius: 0 0 var(--el-border-radius-base) var(--el-border-radius-base);
}

.message-tool-calls-output {
  white-space: pre-wrap;
  word-break: break-word;
}

.output-line {
  margin-bottom: 2px;
  white-space: pre-wrap; /* 保留参数/结果中的换行，避免整段挤在一行 */
  word-break: break-word;

  &:last-child {
    margin-bottom: 0;
  }
}

.output-line.tool {
  color: var(--el-text-color-primary);
  font-weight: 500;
}

.output-line.args {
  color: var(--el-text-color-secondary);
  padding-left: 10px;
  border-left: 2px solid var(--el-color-primary-light-5);
  font-size: 11px;
}

.output-line.result {
  color: var(--el-text-color-regular);
  padding-left: 10px;
  border-left: 2px solid var(--el-border-color-lighter);
}

.output-line.error {
  color: var(--el-color-danger);
  padding-left: 10px;
  border-left: 2px solid var(--el-color-danger-light-5);
}

.output-line--cursor {
  display: inline-block;
  animation: think-blink 0.8s step-end infinite;
  color: var(--el-color-primary);
}
@keyframes think-blink {
  50% { opacity: 0; }
}

.message-output-files {
  margin-top: 10px;
}

.message-output-display-fields {
  margin-top: 10px;
}
</style>
