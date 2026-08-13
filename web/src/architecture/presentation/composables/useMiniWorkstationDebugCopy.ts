import { computed, type Ref } from 'vue'
import { ElMessage } from 'element-plus'
import type {
  AssistantBlock,
  ChatMessage,
  ChatMessageToolCall
} from '@/architecture/presentation/composables/useWorkspaceChatStream'
import { translate } from '@/architecture/shared/i18n'
import { escapeHtml, sanitizeHtml } from '@/architecture/shared/sanitizeHtml'

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
      ElMessage.warning(translate('miniWorkstation.debugNoCopyContent'))
      return
    }

    try {
      await copyTextToClipboard(text)
      ElMessage.success(getCopySuccessLabel(mode))
    } catch {
      ElMessage.error(translate('miniWorkstation.copyFailed'))
    }
  }

  async function copyDebugToolSummary() {
    const text = buildDebugToolSummaryText()
    if (!text.trim()) {
      ElMessage.warning(translate('miniWorkstation.debugNoToolCalls'))
      return
    }

    try {
      await copyTextToClipboard(text)
      ElMessage.success(translate('miniWorkstation.copiedCallSummary'))
    } catch {
      ElMessage.error(translate('miniWorkstation.copyFailed'))
    }
  }

  async function exportDebugConversationPdf() {
    const text = buildDebugCopyText('all')
    if (!text.trim()) {
      ElMessage.warning(translate('miniWorkstation.debugNoCopyContent'))
      return
    }

    try {
      const html = await buildPrintableDebugHtml(
        text,
        `${translate('miniWorkstation.debugConversationTitle')} - ${options.sessionId.value || options.fullCodePath() || 'kageos'}`
      )
      printHtmlDocument(html)
      ElMessage.success(translate('miniWorkstation.exportPdfReady'))
    } catch {
      ElMessage.error(translate('miniWorkstation.exportPdfFailed'))
    }
  }

  function buildDebugToolSummaryText() {
    if (debugToolSteps.value.length === 0) return ''
    return [
      `# ${translate('miniWorkstation.debugToolSummaryTitle')}`,
      `${translate('miniWorkstation.directory')}: ${options.fullCodePath() || '-'}`,
      `${translate('miniWorkstation.directoryName')}: ${options.dirName() || options.displayPath.value || '-'}`,
      `${translate('miniWorkstation.sessionId')}: ${options.sessionId.value || '-'}`,
      translate('miniWorkstation.debugToolSummaryStats', {
        total: debugToolSteps.value.length,
        success: debugSuccessCount.value,
        error: debugErrorCount.value
      }),
      `${translate('miniWorkstation.copiedAt')}: ${new Date().toISOString()}`,
      '',
      debugToolSteps.value.map(step => step.copyText).join('\n\n')
    ].join('\n')
  }

  function buildDebugCopyText(mode: CopyDebugMode) {
    const list = options.messages.value || []
    if (list.length === 0) return ''

    const header = [
      `# ${translate('miniWorkstation.debugConversationTitle')}`,
      `${translate('miniWorkstation.directory')}: ${options.fullCodePath() || '-'}`,
      `${translate('miniWorkstation.directoryName')}: ${options.dirName() || options.displayPath.value || '-'}`,
      `${translate('miniWorkstation.sessionId')}: ${options.sessionId.value || '-'}`,
      `${translate('miniWorkstation.copyScope')}: ${getCopyModeLabel(mode)}`,
      `${translate('miniWorkstation.copiedAt')}: ${new Date().toISOString()}`,
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
    copyDebugToolSummary,
    exportDebugConversationPdf
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
  const parts = [`## ${translate('miniWorkstation.step', { count: index })} ${call.name || '(unknown)'} [${getToolStatusLabel(call.status || '-')}]`]
  if (argumentsPreview) parts.push('', `${translate('miniWorkstation.arguments')}:`, fenceContent(argumentsPreview, 'json'))
  if (outputPreview) parts.push('', `${translate('miniWorkstation.debugOutputSummary')}:`, fenceContent(outputPreview))
  if (errorPreview) parts.push('', `${translate('miniWorkstation.debugErrorSummary')}:`, fenceContent(errorPreview))
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
      translate('miniWorkstation.omittedLines', { count: omitted }),
      ...lines.slice(-DEBUG_TAIL_LINES)
    ].join('\n')
  }

  if (lines.length === 1 && text.length > DEBUG_SINGLE_LINE_LIMIT) {
    const head = text.slice(0, 80)
    const tail = text.slice(-80)
    return `${head}\n${translate('miniWorkstation.omittedChars', { count: text.length - 160 })}\n${tail}`
  }

  return text
}

function getToolStatusLabel(status: string) {
  if (status === 'streaming') return translate('miniWorkstation.toolStatusStreaming')
  if (status === 'running') return translate('miniWorkstation.toolStatusRunning')
  if (status === 'ok' || status === 'success') return translate('miniWorkstation.toolStatusSuccess')
  if (status === 'error' || status === 'failed') return translate('miniWorkstation.toolStatusFailed')
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
    message.user ? `${translate('miniWorkstation.user')}: ${message.user}` : '',
    message.created_at ? `${translate('miniWorkstation.time')}: ${message.created_at}` : ''
  ].filter(Boolean)

  const parts: string[] = [meta.length ? `${title} (${meta.join(', ')})` : title]
  if (message.role === 'user') {
    if (options.includeContent && message.content) {
      parts.push('', message.content.trim())
    }
    if (message.files?.length) {
      parts.push('', `### ${translate('miniWorkstation.uploadedFiles')}`)
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

  const parts: string[] = [`### ${translate('miniWorkstation.toolCalls')}`]
  targetCalls.forEach((call, index) => {
    parts.push('', `#### ${index + 1}. ${call.name || '(unknown)'} [${call.status || '-'}]`)
    if (call.arguments) {
      parts.push('', `${translate('miniWorkstation.arguments')}:`, fenceContent(formatMaybeJson(call.arguments), 'json'))
    }
    if (call.result) {
      parts.push('', `${translate('miniWorkstation.result')}:`, fenceContent(formatLooseText(call.result)))
    }
    if (call.result_data != null) {
      parts.push('', `${translate('miniWorkstation.resultData')}:`, fenceContent(formatJsonValue(call.result_data), 'json'))
    }
    if (call.error) {
      parts.push('', `${translate('miniWorkstation.error')}:`, fenceContent(formatLooseText(call.error)))
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

async function buildPrintableDebugHtml(markdown: string, title: string) {
  const html = await renderDebugMarkdown(markdown)
  const safeTitle = escapeHtml(title)
  return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>${safeTitle}</title>
  <style>
    @page {
      size: A4;
      margin: 14mm 13mm;
    }

    * {
      box-sizing: border-box;
    }

    body {
      margin: 0;
      color: #111827;
      background: #ffffff;
      font-family: Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
      font-size: 12.5px;
      line-height: 1.58;
    }

    .print-root {
      width: 100%;
    }

    h1,
    h2,
    h3,
    h4 {
      color: #0f172a;
      page-break-after: avoid;
    }

    h1 {
      margin: 0 0 12px;
      padding-bottom: 10px;
      border-bottom: 2px solid #0f172a;
      font-size: 24px;
      line-height: 1.25;
    }

    h2 {
      margin: 22px 0 8px;
      padding-top: 12px;
      border-top: 1px solid #dbe3ef;
      font-size: 17px;
    }

    h3 {
      margin: 16px 0 6px;
      font-size: 14px;
    }

    h4 {
      margin: 12px 0 6px;
      font-size: 12.5px;
    }

    p {
      margin: 7px 0;
    }

    ul,
    ol {
      margin: 7px 0 9px 22px;
      padding: 0;
    }

    li + li {
      margin-top: 3px;
    }

    pre {
      margin: 8px 0 12px;
      padding: 9px 10px;
      border: 1px solid #d6dee9;
      border-radius: 6px;
      background: #f8fafc;
      color: #0f172a;
      font-family: "JetBrains Mono", SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
      font-size: 10.5px;
      line-height: 1.48;
      white-space: pre-wrap;
      word-break: break-word;
      page-break-inside: auto;
    }

    code {
      font-family: "JetBrains Mono", SFMono-Regular, Consolas, "Liberation Mono", Menlo, monospace;
      word-break: break-word;
    }

    p code,
    li code {
      padding: 1px 4px;
      border-radius: 4px;
      background: #eef3f8;
      font-size: 0.92em;
    }

    blockquote {
      margin: 10px 0;
      padding: 7px 10px;
      border-left: 3px solid #94a3b8;
      background: #f8fafc;
      color: #334155;
    }

    table {
      width: 100%;
      margin: 10px 0;
      border-collapse: collapse;
      font-size: 11.5px;
    }

    th,
    td {
      padding: 6px 7px;
      border: 1px solid #d6dee9;
      vertical-align: top;
      word-break: break-word;
    }

    th {
      background: #f1f5f9;
      text-align: left;
    }

    a {
      color: #0f766e;
      text-decoration: none;
      word-break: break-all;
    }

    hr {
      margin: 16px 0;
      border: 0;
      border-top: 1px solid #dbe3ef;
    }

    @media print {
      body {
        print-color-adjust: exact;
        -webkit-print-color-adjust: exact;
      }
    }
  </style>
</head>
<body>
  <main class="print-root">${html}</main>
</body>
</html>`
}

async function renderDebugMarkdown(markdown: string) {
  try {
    const { marked } = await import('marked')
    return sanitizeHtml(marked.parse(markdown, { breaks: true, gfm: true }) as string)
  } catch {
    return `<pre>${escapeHtml(markdown)}</pre>`
  }
}

function printHtmlDocument(html: string) {
  if (typeof document === 'undefined' || !document.body) {
    throw new Error('document unavailable')
  }

  const iframe = document.createElement('iframe')
  let printed = false
  let cleanupTimer: ReturnType<typeof setTimeout> | undefined

  const cleanup = () => {
    if (cleanupTimer) {
      clearTimeout(cleanupTimer)
      cleanupTimer = undefined
    }
    iframe.remove()
  }

  iframe.title = translate('miniWorkstation.exportPdf')
  iframe.style.position = 'fixed'
  iframe.style.right = '0'
  iframe.style.bottom = '0'
  iframe.style.width = '1px'
  iframe.style.height = '1px'
  iframe.style.border = '0'
  iframe.style.opacity = '0.01'
  iframe.style.pointerEvents = 'none'
  iframe.setAttribute('aria-hidden', 'true')

  document.body.appendChild(iframe)

  const printWindow = iframe.contentWindow
  const printDocument = iframe.contentDocument || printWindow?.document
  if (!printWindow || !printDocument) {
    cleanup()
    throw new Error('print frame unavailable')
  }

  const handleAfterPrint = () => {
    printWindow.removeEventListener('afterprint', handleAfterPrint)
    setTimeout(cleanup, 300)
  }

  const printNow = () => {
    if (printed) return
    printed = true
    printWindow.addEventListener('afterprint', handleAfterPrint)
    printWindow.focus()
    printWindow.print()
    cleanupTimer = setTimeout(cleanup, 60000)
  }

  iframe.onload = () => {
    setTimeout(printNow, 120)
  }

  printDocument.open()
  printDocument.write(html)
  printDocument.close()
  setTimeout(printNow, 500)
}

function getCopyModeLabel(mode: CopyDebugMode) {
  const map: Record<CopyDebugMode, string> = {
    all: translate('miniWorkstation.copyModeAll'),
    'last-turn': translate('miniWorkstation.copyModeLastTurn'),
    'all-tools': translate('miniWorkstation.copyModeAllTools'),
    'error-tools': translate('miniWorkstation.copyModeErrorTools'),
    'success-tools': translate('miniWorkstation.copyModeSuccessTools')
  }
  return map[mode]
}

function getCopySuccessLabel(mode: CopyDebugMode) {
  return translate('miniWorkstation.copiedMode', { mode: getCopyModeLabel(mode) })
}
