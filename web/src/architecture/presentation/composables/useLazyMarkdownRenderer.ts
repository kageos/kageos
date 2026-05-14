import { onScopeDispose, ref } from 'vue'
import { escapeHtml, sanitizeHtml } from '@/architecture/shared/sanitizeHtml'

type MarkdownRenderer = (content: string) => string

const MARKDOWN_OPTIONS = {
  breaks: true,
  gfm: true,
} as const

let markdownRenderer: MarkdownRenderer | null = null
let markdownLoader: Promise<void> | null = null
const subscribers = new Set<() => void>()

function notifySubscribers(): void {
  subscribers.forEach((subscriber) => subscriber())
}

function fallbackMarkdown(content: string): string {
  return escapeHtml(content).replace(/\n/g, '<br>')
}

async function ensureMarkdownRenderer(): Promise<void> {
  if (markdownRenderer) {
    return
  }

  if (!markdownLoader) {
    markdownLoader = import('marked')
      .then(({ marked }) => {
        markdownRenderer = (content: string) => marked.parse(content, MARKDOWN_OPTIONS) as string
        notifySubscribers()
      })
      .finally(() => {
        markdownLoader = null
      })
  }

  await markdownLoader
}

export function useLazyMarkdownRenderer() {
  const renderVersion = ref(0)
  const bumpRenderVersion = (): void => {
    renderVersion.value += 1
  }

  subscribers.add(bumpRenderVersion)
  onScopeDispose(() => {
    subscribers.delete(bumpRenderVersion)
  })

  function renderMarkdown(content: string): string {
    renderVersion.value

    if (!content) {
      return ''
    }

    if (markdownRenderer) {
      try {
        return sanitizeHtml(markdownRenderer(content))
      } catch {
        return fallbackMarkdown(content)
      }
    }

    void ensureMarkdownRenderer()
    return fallbackMarkdown(content)
  }

  return {
    renderMarkdown,
    preloadMarkdown: ensureMarkdownRenderer,
  }
}
