export type MiniComposerMentionKind = 'user' | 'resource'
export type MiniComposerMentionTrigger = '@' | '/'

export interface MiniComposerMentionQuery {
  kind: MiniComposerMentionKind
  trigger: MiniComposerMentionTrigger
  query: string
  start: number
  end: number
}

export interface MiniComposerMentionReplacement {
  value: string
  cursor: number
}

export interface MiniComposerMentionToken {
  kind: MiniComposerMentionKind
  trigger: MiniComposerMentionTrigger
  raw: string
  value: string
  start: number
  end: number
}

function normalizeCursor(text: string, cursor: number | null | undefined): number {
  if (typeof cursor !== 'number' || Number.isNaN(cursor)) {
    return text.length
  }
  return Math.max(0, Math.min(cursor, text.length))
}

function getTokenStartBeforeCursor(text: string, cursor: number): number {
  const head = text.slice(0, cursor)
  const lastSpace = Math.max(
    head.lastIndexOf(' '),
    head.lastIndexOf('\n'),
    head.lastIndexOf('\t')
  )
  return lastSpace + 1
}

export function findMiniComposerMentionQuery(
  text: string,
  cursor: number | null | undefined
): MiniComposerMentionQuery | null {
  const safeCursor = normalizeCursor(text, cursor)
  const start = getTokenStartBeforeCursor(text, safeCursor)
  const token = text.slice(start, safeCursor)
  const trigger = token[0]

  if (trigger !== '@' && trigger !== '/') {
    return null
  }

  return {
    kind: trigger === '@' ? 'user' : 'resource',
    trigger,
    query: token.slice(1),
    start,
    end: safeCursor
  }
}

export function replaceMiniComposerMention(
  text: string,
  query: MiniComposerMentionQuery,
  replacement: string
): MiniComposerMentionReplacement {
  const before = text.slice(0, query.start)
  const after = text.slice(query.end)
  const spacer = after.length > 0 && /^\s/.test(after) ? '' : ' '
  const insertText = `${replacement}${spacer}`
  const value = `${before}${insertText}${after}`

  return {
    value,
    cursor: before.length + insertText.length
  }
}

export function findMiniComposerMentionTokens(text: string): MiniComposerMentionToken[] {
  const tokens: MiniComposerMentionToken[] = []
  const matcher = /(^|\s)([@/][^\s]+)/g
  let match: RegExpExecArray | null

  while ((match = matcher.exec(text)) !== null) {
    const prefix = match[1] || ''
    const raw = match[2] || ''
    const trigger = raw[0]

    if ((trigger !== '@' && trigger !== '/') || raw.length <= 1 || raw.startsWith('//')) {
      continue
    }

    const start = match.index + prefix.length
    tokens.push({
      kind: trigger === '@' ? 'user' : 'resource',
      trigger,
      raw,
      value: raw.slice(1),
      start,
      end: start + raw.length
    })
  }

  return tokens
}
