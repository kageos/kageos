export type WorkspaceInvocationTool =
  | 'run_form_submit'
  | 'run_table_create'
  | 'run_table_update'
  | 'run_table_delete'
  | string

export interface WorkspaceInvocationParam {
  key: string
  value: unknown
  fixed?: boolean
}

export interface WorkspaceInvocationSnippetInput {
  tool: WorkspaceInvocationTool
  resourcePath: string
  params?: Record<string, unknown> | WorkspaceInvocationParam[]
  note?: string
}

export interface WorkspacePromptSegment {
  type: 'text' | 'resource'
  text: string
  path?: string
  start: number
  end: number
}

export interface WorkspaceInvocationBlock {
  tool: string
  resourcePath: string
  params: WorkspaceInvocationParam[]
  startLine: number
  endLine: number
}

const RESOURCE_TOKEN_PATTERN = /<(?:\/|\.\/|\.\.\/)[^>\s]+>/g
const RESOURCE_TOKEN_BODY_PATTERN = /^(?:\/|\.\/|\.\.\/)[^>\s]+$/
const DEFAULT_INVOCATION_NOTE = '复制后粘贴到工作台，AI 会按下面信息识别并调用。'

export function wrapWorkspaceResourcePath(path: string): string {
  const normalized = normalizeWorkspaceResourcePath(path)
  return normalized ? `<${normalized}>` : ''
}

export function normalizeWorkspaceResourcePath(path: string): string {
  const body = stripWorkspaceResourceWrapper(path)
  if (!body) return ''
  if (isRelativeWorkspaceResourcePath(body)) {
    return body
  }
  return body.startsWith('/') ? body : `/${body}`
}

export function resolveWorkspaceResourcePath(path: string, basePath = ''): string {
  const normalized = normalizeWorkspaceResourcePath(path)
  if (!normalized || !isRelativeWorkspaceResourcePath(normalized)) {
    return normalized
  }

  return joinWorkspacePath(basePath, normalized) || normalized
}

export function unwrapWorkspaceResourceToken(token: string, basePath = ''): string {
  const trimmed = String(token || '').trim()
  if (!trimmed.startsWith('<') || !trimmed.endsWith('>')) return ''
  const body = trimmed.slice(1, -1).trim()
  if (!RESOURCE_TOKEN_BODY_PATTERN.test(body)) return ''
  return resolveWorkspaceResourcePath(body, basePath)
}

export function parseWorkspacePromptSegments(text: string, basePath = ''): WorkspacePromptSegment[] {
  const source = String(text || '')
  const segments: WorkspacePromptSegment[] = []
  let cursor = 0
  let match: RegExpExecArray | null

  RESOURCE_TOKEN_PATTERN.lastIndex = 0
  while ((match = RESOURCE_TOKEN_PATTERN.exec(source)) !== null) {
    const raw = match[0]
    const start = match.index
    const end = start + raw.length
    if (start > cursor) {
      segments.push({
        type: 'text',
        text: source.slice(cursor, start),
        start: cursor,
        end: start,
      })
    }
    segments.push({
      type: 'resource',
      text: raw,
      path: unwrapWorkspaceResourceToken(raw, basePath),
      start,
      end,
    })
    cursor = end
  }

  if (cursor < source.length) {
    segments.push({
      type: 'text',
      text: source.slice(cursor),
      start: cursor,
      end: source.length,
    })
  }

  if (segments.length === 0) {
    segments.push({
      type: 'text',
      text: source,
      start: 0,
      end: source.length,
    })
  }

  return segments
}

export function buildWorkspaceInvocationSnippet(input: WorkspaceInvocationSnippetInput): string {
  const resourceToken = wrapWorkspaceResourcePath(input.resourcePath)
  const lines = [
    '函数调用：',
    `用途：${input.note || DEFAULT_INVOCATION_NOTE}`,
    `工具：${input.tool}`,
    `函数：${resourceToken}`,
  ]
  const params = normalizeInvocationParams(input.params)

  if (params.length > 0) {
    lines.push('', '参数：')
    params.forEach((param) => {
      const fixedSuffix = param.fixed ? '（固定）' : ''
      lines.push(`${param.key} = ${formatInvocationValue(param.value)}${fixedSuffix}`)
    })
  }

  return lines.join('\n')
}

export function normalizeInvocationParams(
  params?: Record<string, unknown> | WorkspaceInvocationParam[]
): WorkspaceInvocationParam[] {
  if (!params) return []
  if (Array.isArray(params)) {
    return params
      .map((param) => ({
        key: String(param.key || '').trim(),
        value: param.value,
        fixed: !!param.fixed,
      }))
      .filter((param) => param.key)
  }

  return Object.entries(params)
    .filter(([key]) => key.trim())
    .map(([key, value]) => ({ key, value }))
}

export function filterEmptyInvocationParams(params: Record<string, unknown>): Record<string, unknown> {
  return Object.fromEntries(
    Object.entries(params || {}).filter(([, value]) => !isEmptyInvocationValue(value))
  )
}

export function parseWorkspaceInvocationBlocks(text: string, basePath = ''): WorkspaceInvocationBlock[] {
  const lines = String(text || '').split(/\r?\n/)
  const blocks: WorkspaceInvocationBlock[] = []

  for (let index = 0; index < lines.length; index += 1) {
    if (lines[index]?.trim() !== '函数调用：') {
      continue
    }

    let endLine = lines.length - 1
    for (let next = index + 1; next < lines.length; next += 1) {
      if (lines[next]?.trim() === '函数调用：') {
        endLine = next - 1
        break
      }
    }

    const blockLines = lines.slice(index, endLine + 1)
    const tool = readBlockValue(blockLines, '工具')
    const functionLine = readBlockValue(blockLines, '函数')
    const resourcePath = unwrapWorkspaceResourceToken(functionLine, basePath)
    const params = readBlockParams(blockLines)

    if (tool || resourcePath || params.length > 0) {
      blocks.push({
        tool,
        resourcePath,
        params,
        startLine: index,
        endLine,
      })
    }
  }

  return blocks
}

export async function copyTextToClipboard(text: string): Promise<void> {
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
  if (!ok) {
    throw new Error('copy failed')
  }
}

function readBlockValue(lines: string[], key: string): string {
  const prefix = `${key}：`
  const line = lines.find((item) => item.trim().startsWith(prefix))
  return line ? line.trim().slice(prefix.length).trim() : ''
}

function readBlockParams(lines: string[]): WorkspaceInvocationParam[] {
  const paramTitleIndex = lines.findIndex((line) => line.trim() === '参数：')
  if (paramTitleIndex < 0) return []

  return lines
    .slice(paramTitleIndex + 1)
    .map((line) => parseParamLine(line))
    .filter((param): param is WorkspaceInvocationParam => !!param)
}

function parseParamLine(line: string): WorkspaceInvocationParam | null {
  const trimmed = line.trim()
  if (!trimmed || !trimmed.includes('=')) return null
  const [rawKey, ...rawValueParts] = trimmed.split('=')
  const key = String(rawKey || '').trim()
  const rawValue = rawValueParts.join('=').trim()
  if (!key) return null

  return {
    key,
    value: rawValue.replace(/（固定）$/, '').trim(),
    fixed: rawValue.endsWith('（固定）'),
  }
}

function formatInvocationValue(value: unknown): string {
  if (value === undefined || value === null || value === '') return '（留空）'
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

function isEmptyInvocationValue(value: unknown): boolean {
  if (value === undefined || value === null || value === '') return true
  if (Array.isArray(value)) return value.length === 0
  if (typeof value === 'object') return Object.keys(value as Record<string, unknown>).length === 0
  return false
}

function stripWorkspaceResourceWrapper(value: string): string {
  const trimmed = String(value || '').trim()
  if (!trimmed) return ''
  if (trimmed.startsWith('<') && trimmed.endsWith('>')) {
    return trimmed.slice(1, -1).trim()
  }
  return trimmed
}

function isRelativeWorkspaceResourcePath(path: string): boolean {
  return path.startsWith('./') || path.startsWith('../')
}

function joinWorkspacePath(basePath: string, relativePath: string): string {
  const baseDir = workspacePathDirectory(basePath)
  if (!baseDir) return ''

  const { pathPart, queryPart } = splitWorkspacePathQuery(relativePath)
  const stack = baseDir.split('/').filter(Boolean)
  pathPart.split('/').forEach((part) => {
    if (!part || part === '.') return
    if (part === '..') {
      stack.pop()
      return
    }
    stack.push(part)
  })
  const resolved = `/${stack.join('/')}`
  return queryPart ? `${resolved}?${queryPart}` : resolved
}

function workspacePathDirectory(basePath: string): string {
  const normalized = normalizeWorkspaceResourcePath(basePath)
  if (!normalized.startsWith('/')) return ''

  const { pathPart } = splitWorkspacePathQuery(normalized)
  const parts = pathPart.split('/').filter(Boolean)
  if (parts.length === 0) return ''

  const tail = parts[parts.length - 1] || ''
  if (tail.includes('.')) {
    parts.pop()
  }
  return `/${parts.join('/')}`
}

function splitWorkspacePathQuery(value: string): { pathPart: string; queryPart: string } {
  const index = value.indexOf('?')
  if (index < 0) {
    return { pathPart: value, queryPart: '' }
  }
  return {
    pathPart: value.slice(0, index),
    queryPart: value.slice(index + 1),
  }
}
