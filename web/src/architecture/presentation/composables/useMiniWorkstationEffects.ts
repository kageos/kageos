import { eventBus } from '@/architecture/infrastructure/eventBus'
import { nextTick, onMounted, onUnmounted, toRaw, watch, type Ref } from 'vue'
import type { ChatMessage } from '@/architecture/presentation/composables/useWorkspaceChatStream'

export interface UseMiniWorkstationEffectsOptions {
  visible: Ref<boolean>
  maximized: Ref<boolean>
  messages: Ref<ChatMessage[]>
  sending: Ref<boolean>
  sessionId: Ref<string | undefined>
  inputRef: Ref<HTMLTextAreaElement | undefined>
  outputRef: Ref<HTMLElement | undefined>
  stopMiniPoll: () => void
  loadMiniSessions: () => Promise<void> | void
}

export function useMiniWorkstationEffects(options: UseMiniWorkstationEffectsOptions) {
  const { visible, maximized, messages, sending, sessionId, inputRef, outputRef, stopMiniPoll, loadMiniSessions } = options
  let outputResizeObserver: ResizeObserver | null = null
  let resizeScrollTimer: number | undefined

  function scrollToBottom() {
    nextTick(() => {
      const element = outputRef.value
      if (element) {
        element.scrollTop = element.scrollHeight + 100
      }
    })
  }

  function scrollToBottomAfterResize() {
    scrollToBottom()
    requestAnimationFrame(scrollToBottom)
    window.clearTimeout(resizeScrollTimer)
    resizeScrollTimer = window.setTimeout(scrollToBottom, 240)
  }

  watch(() => messages.value.length, scrollToBottom)
  watch(() => {
    const last = messages.value[messages.value.length - 1]
    return (last?.content?.length ?? 0) + (last?.blocks?.length ?? 0) + (last?.tool_calls?.length ?? 0)
  }, scrollToBottom)

  watch(
    outputRef,
    (element) => {
      outputResizeObserver?.disconnect()
      outputResizeObserver = null
      if (!element || typeof ResizeObserver === 'undefined') {
        return
      }
      outputResizeObserver = new ResizeObserver(() => {
        if (visible.value && !maximized.value) {
          scrollToBottomAfterResize()
        }
      })
      outputResizeObserver.observe(element)
    },
    { immediate: true, flush: 'post' }
  )

  watch(visible, (currentVisible) => {
    if (currentVisible) {
      nextTick(() => inputRef.value?.focus())
    }
  })

  onMounted(() => {
    if (visible.value) {
      nextTick(() => inputRef.value?.focus())
    }
  })

  watch(messages, (newMessages) => {
    if (!visible.value && sending.value && sessionId.value) {
      eventBus.emit('workspace:stream-update', {
        session_id: sessionId.value,
        messages: JSON.parse(JSON.stringify(toRaw(newMessages)))
      })
    }
  }, { deep: true })

  watch(sending, (current, previous) => {
    if (current) {
      stopMiniPoll()
    }
    if (!visible.value && previous && !current && sessionId.value) {
      eventBus.emit('workspace:stream-done', { session_id: sessionId.value })
    }
    if (previous && !current && maximized.value) {
      void loadMiniSessions()
    }
  })

  onUnmounted(() => {
    outputResizeObserver?.disconnect()
    window.clearTimeout(resizeScrollTimer)
  })
}
