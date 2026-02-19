<!--
  MessageToolCalls - 模仿深度思考模式：固定视口 4-5 行，逐行展示执行过程与输出预览，自动滚到底部
  - 执行中：视口内滚动输出「工具名 + 状态」及结果片段
  - 已完成：同样视口展示完整过程与结果预览
  - 输出文件仍在下方独立展示
-->
<template>
  <div class="message-tool-calls">
    <div class="message-tool-calls-head">
      <span v-if="hasRunning" class="head-status head-status--running">
        <el-icon class="head-icon head-icon--spin"><Loading /></el-icon>
        正在执行工具…
      </span>
      <span v-else class="head-status head-status--done">
        <el-icon class="head-icon"><CircleCheck /></el-icon>
        已完成 {{ toolCalls.length }} 个工具调用
      </span>
    </div>
    <div class="message-tool-calls-viewport" ref="viewportRef">
      <div class="message-tool-calls-output">
        <div
          v-for="(line, idx) in outputLines"
          :key="idx"
          :class="['output-line', line.type]"
        >
          {{ line.text }}
        </div>
        <div v-if="hasRunning && outputLines.length > 0" class="output-line output-line--cursor">▌</div>
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
import { Loading, CircleCheck } from '@element-plus/icons-vue'
import OutputFilesDisplay from './OutputFilesDisplay.vue'
import type { WorkspaceChatToolCallSummary } from '@/api/workspace'
import type { OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'

const props = defineProps<{
  toolCalls: WorkspaceChatToolCallSummary[]
  fileGroups: OutputFileGroup[]
}>()

const viewportRef = ref<HTMLElement | null>(null)

const hasRunning = computed(() =>
  props.toolCalls.some((t) => t.status === 'streaming' || t.status === 'running')
)

/** 状态中文 */
function statusLabel(status: string): string {
  if (status === 'streaming') return '解析中…'
  if (status === 'running') return '执行中…'
  if (status === 'ok') return '成功'
  if (status === 'error') return '失败'
  return status
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

/** 将 toolCalls 转成逐行输出（类似深度思考的 log） */
const outputLines = computed(() => {
  const lines: { text: string; type: string }[] = []
  for (const tc of props.toolCalls) {
    const label = statusLabel(tc.status)
    lines.push({ text: `[${tc.name}] ${label}`, type: 'tool' })
    if (tc.arguments) {
      const argsPreview = formatArgsPreview(tc.arguments)
      if (argsPreview) lines.push({ text: `参数: ${argsPreview}`, type: 'args' })
    }
    if (tc.result) {
      const preview = formatResultOrError(tc.result)
      if (preview) lines.push({ text: preview, type: 'result' })
    }
    if (tc.error) {
      const preview = formatResultOrError(tc.error)
      if (preview) lines.push({ text: `错误: ${preview}`, type: 'error' })
    }
  }
  return lines
})

function scrollViewportToBottom() {
  nextTick(() => {
    const el = viewportRef.value
    if (!el) return
    el.scrollTop = el.scrollHeight - el.clientHeight
  })
}

watch(
  () => outputLines.value,
  () => scrollViewportToBottom(),
  { deep: true }
)
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

.head-status--done {
  color: var(--el-color-success);
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

/* 深度思考式视口：4-5 行高度，内容往上滚动 */
.message-tool-calls-viewport {
  max-height: 7.5em; /* 约 5 行，1.5 line-height */
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px 12px;
  background: var(--el-fill-color-blank);
  border: 1px solid var(--el-border-color-lighter);
  border-top: none;
  border-radius: 0 0 var(--el-border-radius-base) var(--el-border-radius-base);
  scroll-behavior: smooth;
  line-height: 1.5;
  font-size: 12px;
  color: var(--el-text-color-regular);
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
</style>
