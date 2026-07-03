const THINK_BLOCK_RE = /<think>[\s\S]*?<\/think>/gi
const OPEN_THINK_TAIL_RE = /<think>[\s\S]*$/i
const STRAY_CLOSE_THINK_RE = /<\/think>/gi
const THINK_TAG_RE = /<\/?think>/gi

export interface WorkspaceThinkSegment {
  type: 'content' | 'thinking'
  text: string
}

export function stripWorkspaceThinkBlocks(content: string): string {
  if (!content) return content
  const stripped = content
    .replace(THINK_BLOCK_RE, '')
    .replace(OPEN_THINK_TAIL_RE, '')
    .replace(STRAY_CLOSE_THINK_RE, '')
  return stripped === content ? content : stripped.trimStart()
}

export function splitWorkspaceThinkBlocks(content: string): WorkspaceThinkSegment[] {
  if (!content) return []
  const segments: WorkspaceThinkSegment[] = []
  let inThink = false
  let lastIndex = 0
  let sawThink = false
  for (const match of content.matchAll(THINK_TAG_RE)) {
    const index = match.index ?? 0
    appendThinkSegment(segments, inThink ? 'thinking' : 'content', content.slice(lastIndex, index))
    const tag = match[0].toLowerCase()
    if (tag === '<think>') {
      inThink = true
      sawThink = true
    } else if (inThink) {
      inThink = false
      sawThink = true
    } else {
      sawThink = true
    }
    lastIndex = index + match[0].length
  }
  appendThinkSegment(segments, inThink ? 'thinking' : 'content', content.slice(lastIndex))
  if (sawThink) trimFirstContentSegment(segments)
  return segments
}

function appendThinkSegment(segments: WorkspaceThinkSegment[], type: WorkspaceThinkSegment['type'], text: string) {
  if (!text) return
  const last = segments[segments.length - 1]
  if (last?.type === type) {
    last.text += text
    return
  }
  segments.push({ type, text })
}

function trimFirstContentSegment(segments: WorkspaceThinkSegment[]) {
  const firstContent = segments.find(segment => segment.type === 'content')
  if (firstContent) firstContent.text = firstContent.text.trimStart()
}
