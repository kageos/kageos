import { computed, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import type {
  AssistantBlock,
  ChatMessage,
  ChatMessageToolCall
} from '@/architecture/presentation/composables/useWorkspaceChatStream'

export type CopyDebugMode = 'all' | 'last-turn' | 'all-tools' | 'error-tools' | 'success-tools'

export interface DebugToolStep {
  key: string
  index: number
  name: string
  status: string
  statusLabel: string
  statusClass: 'running' | 'ok' | 'error' | 'default'
  argumentsPreview: string
  outputPreview: string
  errorPreview: string
  copyText: string
}

interface UseMiniWorkstationDebugCopyOptions {
  messages: Ref<ChatMessage[]>
  fullCodePath: () => string
  dirName: () => string | undefined
  displayPath: Ref<string>
  sessionId: Ref<string | undefined>
}

const DEBUG_HEAD_LINES = 10
const DEBUG_TAIL_LINES = 10
const DEBUG_SINGLE_LINE_LIMIT = 220

export function useMiniWorkstationDebugCopy(options: UseMiniWorkstationDebugCopyOptions) {
  const debugToolSteps = computed<DebugToolStep[]>(() => {
    const steps: DebugToolStep[] = []
    for (const [messageIndex, message] of options.messages.value.entries()) {
      const calls = collectMessageToolCalls(message)
      calls.forEach((call, callIndex) => {
        const index = steps.length + 1
        const status = call.status || '-'
        const argumentsPreview = buildDebugPreview(call.arguments, true)
        const outputPreview = call.result
          ? buildDebugPreview(call.result)
          : call.result_data != null
            ? buildDebugPreview(call.result_data, true)
            : ''
        const errorPreview = buildDebugPreview(call.error)
        steps.push({
          key: `${messageIndex}-${callIndex}-${call.name || 'tool'}-${index}`,
          index,
          name: call.name || '(unknown)',
          status,
          statusLabel: getToolStatusLabel(status),
          statusClass: getToolStatusClass(status),
          argumentsPreview,
          outputPreview,
          errorPreview,
          copyText: formatDebugToolStepForCopy(index, call, argumentsPreview, outputPreview, errorPreview)
        })
      })
    }
    return steps
  })

  const debugSuccessCount = computed(() => debugToolSteps.value.filter(step => step.statusClass === 'ok').length)
  const debugErrorCount = computed(() => debugToolSteps.value.filter(step => step.statusClass === 'error').length)

  async function copyDebugConversation(mode: CopyDebugMode) {
    const text = buildDebugCopyText(mode)
    if (!text.trim()) {
      ElMessage.warning('当前没有可复制的调试内容')
      return
    }

    try {
      await copyTextToClipboard(text)
      ElMessage.success(getCopySuccessLabel(mode))
    } catch {
      ElMessage.error('复制失败')
    }
  }

  async function copyDebugToolSummary() {
    const text = buildDebugToolSummaryText()
    if (!text.trim()) {
      ElMessage.warning('当前没有工具调用记录')
      return
    }

    try {
      await copyTextToClipboard(text)
      ElMessage.success('已复制调用摘要')
    } catch {
      ElMessage.error('复制失败')
    }
  }

  function buildDebugToolSummaryText() {
    if (debugToolSteps.value.length === 0) return ''
    return [
      '# Mini 工具调用摘要',
      `目录: ${options.fullCodePath() || '-'}`,
      `目录名: ${options.dirName() || options.displayPath.value || '-'}`,
      `会话ID: ${options.sessionId.value || '-'}`,
      `工具调用: ${debugToolSteps.value.length} 步，成功 ${debugSuccessCount.value}，失败 ${debugErrorCount.value}`,
      `复制时间: ${new Date().toISOString()}`,
      '',
      debugToolSteps.value.map(step => step.copyText).join('\n\n')
    ].join('\n')
  }

  function buildDebugCopyText(mode: CopyDebugMode) {
    const list = options.messages.value || []
    if (list.length === 0) return ''

    const header = [
      '# Mini 工作台调试对话',
      `目录: ${options.fullCodePath() || '-'}`,
      `目录名: ${options.dirName() || options.displayPath.value || '-'}`,
      `会话ID: ${options.sessionId.value || '-'}`,
      `复制范围: ${getCopyModeLabel(mode)}`,
      `复制时间: ${new Date().toISOString()}`,
      ''
    ].join('\n')

    if (mode === 'all') {
      return header + formatMessagesForCopy(list, { includeContent: true, includeToolCalls: true })
    }

    if (mode === 'last-turn') {
      return header + formatMessagesForCopy(getLastTurnMessages(list), { includeContent: true, includeToolCalls: true })
    }

    const statusFilter = getToolStatusFilter(mode)
    return header + formatMessagesWithToolFilter(list, statusFilter)
  }

  return {
    debugToolSteps,
    debugSuccessCount,
    debugErrorCount,
    copyDebugConversation,
    copyDebugToolSummary
  }
}

export function collectMessageToolCalls(message: ChatMessage): ChatMessageToolCall[] {
  if (message.blocks?.length) {
    return message.blocks.flatMap((block) => block.type === 'tool_calls' ? block.calls : [])
  }
  return message.tool_calls || []
}

function formatDebugToolStepForCopy(
  index: number,
  call: ChatMessageToolCall,
  argumentsPreview: string,
  outputPreview: string,
  errorPreview: string
) {
  const parts = [`## 第 ${index} 步 ${call.name || '(unknown)'} [${getToolStatusLabel(call.status || '-')}]`]
  if (argumentsPreview) parts.push('', '参数:', fenceContent(argumentsPreview, 'json'))
  if (outputPreview) parts.push('', '输出摘要:', fenceContent(outputPreview))
  if (errorPreview) parts.push('', '错误摘要:', fenceContent(errorPreview))
  return parts.join('\n')
}

function buildDebugPreview(value: unknown, preferJson = false) {
  if (value == null) return ''
  const raw = typeof value === 'string'
    ? (preferJson ? formatMaybeJson(value) : formatLooseText(value))
    : formatJsonValue(value)
  return truncateDebugPreview(raw)
}

function truncateDebugPreview(value: string) {
  const text = String(value || '').trim()
  if (!text) return ''

  const lines = text.split(/\r?\n/)
  if (lines.length > DEBUG_HEAD_LINES + DEBUG_TAIL_LINES) {
    const omitted = lines.length - DEBUG_HEAD_LINES - DEBUG_TAIL_LINES
    return [
      ...lines.slice(0, DEBUG_HEAD_LINES),
      `... 省略 ${omitted} 行 ...`,
      ...lines.slice(-DEBUG_TAIL_LINES)
    ].join('\n')
  }

  if (lines.length === 1 && text.length > DEBUG_SINGLE_LINE_LIMIT) {
    const head = text.slice(0, 80)
    const tail = text.slice(-80)
    return `${head}\n... 省略 ${text.length - 160} 字符 ...\n${tail}`
  }

  return text
}

function getToolStatusLabel(status: string) {
  if (status === 'streaming') return '解析中'
  if (status === 'running') return '执行中'
  if (status === 'ok' || status === 'success') return '成功'
  if (status === 'error' || status === 'failed') return '失败'
  return status || '-'
}

function getToolStatusClass(status: string): DebugToolStep['statusClass'] {
  if (status === 'streaming' || status === 'running') return 'running'
  if (status === 'ok' || status === 'success') return 'ok'
  if (status === 'error' || status === 'failed') return 'error'
  return 'default'
}

function getLastTurnMessages(list: ChatMessage[]) {
  const lastUserIndex = [...list].reverse().findIndex(item => item.role === 'user')
  if (lastUserIndex < 0) return list.slice(-1)
  const start = list.length - 1 - lastUserIndex
  return list.slice(start)
}

function getToolStatusFilter(mode: CopyDebugMode): ((call: ChatMessageToolCall) => boolean) | null {
  if (mode === 'error-tools') {
    return (call) => call.status === 'error' || call.status === 'failed'
  }
  if (mode === 'success-tools') {
    return (call) => call.status === 'ok' || call.status === 'success'
  }
  if (mode === 'all-tools') {
    return () => true
  }
  return null
}

function formatMessagesWithToolFilter(
  list: ChatMessage[],
  filter: ((call: ChatMessageToolCall) => boolean) | null
) {
  if (!filter) return ''

  const chunks: string[] = []
  let lastUser: ChatMessage | null = null
  for (const msg of list) {
    if (msg.role === 'user') {
      lastUser = msg
      continue
    }
    const calls = collectMessageToolCalls(msg).filter(filter)
    if (calls.length === 0) continue

    if (lastUser) {
      chunks.push(formatMessageForCopy(lastUser, { includeContent: true, includeToolCalls: false }))
    }
    chunks.push(formatMessageForCopy(msg, {
      includeContent: true,
      includeToolCalls: true,
      toolCallFilter: filter
    }))
    lastUser = null
  }

  return chunks.join('\n').trim()
}

function formatMessagesForCopy(
  list: ChatMessage[],
  options: {
    includeContent: boolean
    includeToolCalls: boolean
    toolCallFilter?: (call: ChatMessageToolCall) => boolean
  }
) {
  return list
    .map((message) => formatMessageForCopy(message, options))
    .filter(Boolean)
    .join('\n')
    .trim()
}

function formatMessageForCopy(
  message: ChatMessage,
  options: {
    includeContent: boolean
    includeToolCalls: boolean
    toolCallFilter?: (call: ChatMessageToolCall) => boolean
  }
) {
  const title = message.role === 'user' ? '## User' : '## Assistant'
  const meta = [
    message.user ? `用户: ${message.user}` : '',
    message.created_at ? `时间: ${message.created_at}` : ''
  ].filter(Boolean)

  const parts: string[] = [meta.length ? `${title} (${meta.join('，')})` : title]
  if (message.role === 'user') {
    if (options.includeContent && message.content) {
      parts.push('', message.content.trim())
    }
    if (message.files?.length) {
      parts.push('', '### 上传文件')
      for (const file of message.files) {
        parts.push(`- ${file.name}${file.ref ? ` (${file.ref})` : ''}`)
      }
    }
    return parts.join('\n').trim()
  }

  if (message.blocks?.length) {
    const blockText = formatAssistantBlocksForCopy(message.blocks, options)
    if (blockText) parts.push('', blockText)
  } else {
    if (options.includeContent && message.content) {
      parts.push('', message.content.trim())
    }
    if (options.includeToolCalls && message.tool_calls?.length) {
      const toolText = formatToolCallsForCopy(message.tool_calls, options.toolCallFilter)
      if (toolText) parts.push('', toolText)
    }
  }

  return parts.join('\n').trim()
}

function formatAssistantBlocksForCopy(
  blocks: AssistantBlock[],
  options: {
    includeContent: boolean
    includeToolCalls: boolean
    toolCallFilter?: (call: ChatMessageToolCall) => boolean
  }
) {
  const parts: string[] = []
  for (const block of blocks) {
    if (block.type === 'content' && options.includeContent && block.text.trim()) {
      parts.push(block.text.trim())
    }
    if (block.type === 'tool_calls' && options.includeToolCalls) {
      const toolText = formatToolCallsForCopy(block.calls, options.toolCallFilter)
      if (toolText) parts.push(toolText)
    }
  }
  return parts.join('\n\n').trim()
}

function formatToolCallsForCopy(
  calls: ChatMessageToolCall[],
  filter?: (call: ChatMessageToolCall) => boolean
) {
  const targetCalls = filter ? calls.filter(filter) : calls
  if (targetCalls.length === 0) return ''

  const parts: string[] = ['### 工具调用']
  targetCalls.forEach((call, index) => {
    parts.push('', `#### ${index + 1}. ${call.name || '(unknown)'} [${call.status || '-'}]`)
    if (call.arguments) {
      parts.push('', '参数:', fenceContent(formatMaybeJson(call.arguments), 'json'))
    }
    if (call.result) {
      parts.push('', '结果:', fenceContent(formatLooseText(call.result)))
    }
    if (call.result_data != null) {
      parts.push('', '结果数据:', fenceContent(formatJsonValue(call.result_data), 'json'))
    }
    if (call.error) {
      parts.push('', '错误:', fenceContent(formatLooseText(call.error)))
    }
  })
  return parts.join('\n')
}

function formatMaybeJson(value: string) {
  const text = formatLooseText(value)
  try {
    return JSON.stringify(JSON.parse(text), null, 2)
  } catch {
    return text
  }
}

function formatJsonValue(value: unknown) {
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

function formatLooseText(value: string) {
  return String(value || '').replace(/\\n/g, '\n').replace(/\\r/g, '\r').trim()
}

function fenceContent(value: string, lang = '') {
  const body = value || ''
  const fence = body.includes('```') ? '````' : '```'
  return `${fence}${lang}\n${body}\n${fence}`
}

async function copyTextToClipboard(text: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', 'true')
  textarea.style.position = 'fixed'
  textarea.style.left = '-9999px'
  document.body.appendChild(textarea)
  textarea.select()
  const ok = document.execCommand('copy')
  document.body.removeChild(textarea)
  if (!ok) throw new Error('copy failed')
}

function getCopyModeLabel(mode: CopyDebugMode) {
  const map: Record<CopyDebugMode, string> = {
    all: '全部对话',
    'last-turn': '最后一轮',
    'all-tools': '全部工具调用',
    'error-tools': '失败工具调用',
    'success-tools': '成功工具调用'
  }
  return map[mode]
}

function getCopySuccessLabel(mode: CopyDebugMode) {
  return `已复制${getCopyModeLabel(mode)}`
}
