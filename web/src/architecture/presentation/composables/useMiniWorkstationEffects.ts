import { eventBus } from '@/architecture/presentation/context/eventBusContext'
import { nextTick, onMounted, onUnmounted, toRaw, watch, type Ref } from 'vue'
import type { ChatMessage } from '@/architecture/presentation/composables/useWorkspaceChatStream'

export interface UseMiniWorkstationEffectsOptions {
  visible: Ref<boolean>
  maximized: Ref<boolean>
  messages: Ref<ChatMessage[]>
  sending: Ref<boolean>
  streamingDisplayLength: Ref<number>
  sessionId: Ref<string | undefined>
  inputRef: Ref<{ focus: () => void } | undefined>
  outputRef: Ref<HTMLElement | undefined>
  stopMiniPoll: () => void
  loadMiniSessions: () => Promise<void> | void
}

export function useMiniWorkstationEffects(options: UseMiniWorkstationEffectsOptions) {
  const { visible, maximized, messages, sending, streamingDisplayLength, sessionId, inputRef, outputRef, stopMiniPoll, loadMiniSessions } = options
  let outputResizeObserver: ResizeObserver | null = null
  let resizeScrollTimer: number | undefined
  const AUTO_SCROLL_BOTTOM_THRESHOLD = 96

  function shouldAutoScroll() {
    const element = outputRef.value
    if (!element) {
      return true
    }
    if (!maximized.value) {
      return true
    }
    return element.scrollHeight - element.scrollTop - element.clientHeight <= AUTO_SCROLL_BOTTOM_THRESHOLD
  }

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

  function scrollToBottomIfNeeded() {
    if (shouldAutoScroll()) {
      scrollToBottom()
    }
  }

  watch(messages, scrollToBottomIfNeeded, { deep: true, flush: 'pre' })
  watch(streamingDisplayLength, scrollToBottomIfNeeded, { flush: 'pre' })

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
