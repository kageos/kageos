<!--
  MessageToolCalls - 工作台工具调用展示
  - 只展示「调用了哪个具体工具」+ 状态（解析中/执行中/成功/失败），执行过程可见
  - 不展示「已完成 x 个工具调用」文案
  - 工具输出的参数/结果/错误默认折叠，点击某条工具可展开查看
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
    <div
      class="message-tool-calls-viewport"
      :class="{ 'message-tool-calls-viewport--no-head': !hasRunning }"
      ref="viewportRef"
    >
      <div class="message-tool-calls-list">
        <div
          v-for="(tc, idx) in toolCalls"
          :key="tc.id ?? idx"
          :class="['tool-call-row', { 'tool-call-row--expanded': expandedSet.has(idx) }]"
          @click="toggleExpand(idx)"
        >
          <div class="tool-call-row-header">
            <el-icon :class="['tool-call-icon', iconClass(tc.status)]">
              <component :is="statusIcon(tc.status)" />
            </el-icon>
            <span class="tool-call-name">{{ tc.name }}</span>
            <el-tag v-if="tc.status === 'streaming'" type="info" size="small" class="tool-call-status">解析中</el-tag>
            <el-tag v-else-if="tc.status === 'running'" type="info" size="small" class="tool-call-status">执行中</el-tag>
            <el-tag v-else :type="tc.status === 'ok' ? 'success' : 'danger'" size="small" class="tool-call-status">
              {{ statusLabel(tc.status) }}
            </el-tag>
            <el-icon class="tool-call-chevron"><ArrowDown v-if="!expandedSet.has(idx)" /><ArrowUp v-else /></el-icon>
          </div>
          <div v-show="expandedSet.has(idx)" class="tool-call-detail">
            <div v-if="tc.arguments" class="detail-section">
              <div class="detail-label">参数</div>
              <pre class="detail-content">{{ formatArgsPreview(tc.arguments) }}</pre>
            </div>
            <div v-if="tc.result" class="detail-section">
              <div class="detail-label">结果</div>
              <pre class="detail-content">{{ formatResultOrError(tc.result) }}</pre>
            </div>
            <div v-if="tc.error" class="detail-section detail-section--error">
              <div class="detail-label">错误</div>
              <pre class="detail-content">{{ formatResultOrError(tc.error) }}</pre>
            </div>
          </div>
        </div>
        <div v-if="hasRunning && toolCalls.length > 0" class="output-line output-line--cursor">▌</div>
      </div>
    </div>
    <OutputFilesDisplay
      v-if="fileGroups.length > 0"
      :file-groups="fileGroups"
      class="message-output-files"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import { Loading, CircleCheck, CircleClose, ArrowDown, ArrowUp } from '@element-plus/icons-vue'
import OutputFilesDisplay from './OutputFilesDisplay.vue'
import type { WorkspaceChatToolCallSummary } from '@/api/workspace'
import type { OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'

const props = defineProps<{
  toolCalls: WorkspaceChatToolCallSummary[]
  fileGroups: OutputFileGroup[]
}>()

const viewportRef = ref<HTMLElement | null>(null)
/** 展开的工具行（按索引），默认折叠 */
const expandedSet = ref<Set<number>>(new Set())

const hasRunning = computed(() =>
  props.toolCalls.some((t) => t.status === 'streaming' || t.status === 'running')
)

function toggleExpand(idx: number) {
  const next = new Set(expandedSet.value)
  if (next.has(idx)) next.delete(idx)
  else next.add(idx)
  expandedSet.value = next
}

/** 状态中文 */
function statusLabel(status: string): string {
  if (status === 'streaming') return '解析中'
  if (status === 'running') return '执行中'
  if (status === 'ok') return '成功'
  if (status === 'error') return '失败'
  return status
}

function statusIcon(status: string) {
  if (status === 'streaming' || status === 'running') return Loading
  if (status === 'ok') return CircleCheck
  if (status === 'error') return CircleClose
  return CircleCheck
}

function iconClass(status: string): string {
  if (status === 'streaming' || status === 'running') return 'tool-call-icon--running'
  if (status === 'ok') return 'tool-call-icon--ok'
  if (status === 'error') return 'tool-call-icon--error'
  return ''
}

/** 把字符串里字面的 \n、\r 转成真实换行 */
function unescapeNewlines(s: string): string {
  return s.replace(/\\n/g, '\n').replace(/\\r/g, '\r')
}

function formatResultOrError(s: string): string {
  if (!s || !s.trim()) return ''
  return unescapeNewlines(s.trim())
}

function formatArgsPreview(argsStr: string | undefined): string {
  if (!argsStr || !argsStr.trim()) return ''
  const trimmed = argsStr.trim()
  try {
    const obj = JSON.parse(trimmed)
    return unescapeNewlines(JSON.stringify(obj, null, 2))
  } catch {
    return unescapeNewlines(trimmed)
  }
}

function scrollViewportToBottom() {
  nextTick(() => {
    nextTick(() => {
      const el = viewportRef.value
      if (!el) return
      el.scrollTop = el.scrollHeight - el.clientHeight
    })
  })
}

watch(
  () => props.toolCalls.length,
  () => scrollViewportToBottom()
)
watch(hasRunning, (running) => {
  if (running) scrollViewportToBottom()
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

.message-tool-calls-viewport {
  max-height: 12em;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px 12px;
  background: var(--el-fill-color-blank);
  border: 1px solid var(--el-border-color-lighter);
  border-top: none;
  border-radius: 0 0 var(--el-border-radius-base) var(--el-border-radius-base);
  scroll-behavior: smooth;
  font-size: 12px;
  color: var(--el-text-color-regular);
}
.message-tool-calls-viewport--no-head {
  border-top: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
}

.message-tool-calls-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tool-call-row {
  border-radius: var(--el-border-radius-base);
  cursor: pointer;
  transition: background 0.15s;
}
.tool-call-row:hover {
  background: var(--el-fill-color-light);
}

.tool-call-row-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  line-height: 1.4;
}

.tool-call-icon {
  font-size: 14px;
  flex-shrink: 0;
}
.tool-call-icon--ok {
  color: var(--el-color-success);
}
.tool-call-icon--error {
  color: var(--el-color-danger);
}
.tool-call-icon--running {
  color: var(--el-color-primary);
  animation: think-spin 0.8s linear infinite;
}

.tool-call-name {
  font-weight: 500;
  color: var(--el-text-color-primary);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tool-call-status {
  flex-shrink: 0;
}

.tool-call-chevron {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}

.tool-call-detail {
  padding: 8px 8px 8px 28px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-lighter);
  font-size: 11px;
  line-height: 1.45;
}

.detail-section {
  margin-bottom: 8px;
}
.detail-section:last-child {
  margin-bottom: 0;
}
.detail-section--error .detail-content {
  color: var(--el-color-danger);
}

.detail-label {
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
  font-weight: 500;
}

.detail-content {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: var(--el-font-family);
  font-size: 11px;
  color: var(--el-text-color-regular);
}

.output-line--cursor {
  display: inline-block;
  animation: think-blink 0.8s step-end infinite;
  color: var(--el-color-primary);
  padding: 4px 0;
}
@keyframes think-blink {
  50% { opacity: 0; }
}

.message-output-files {
  margin-top: 10px;
}
</style>
