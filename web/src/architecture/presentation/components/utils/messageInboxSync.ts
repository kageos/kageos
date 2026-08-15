import { eventBus } from '@/architecture/presentation/context/eventBusContext'

export interface MessageInboxChangedPayload {
  source?: symbol
}

interface MessageInboxSubscriptionOptions {
  source?: symbol
  shouldRefresh?: () => boolean
  refresh: () => void | Promise<unknown>
}

const MESSAGE_INBOX_CHANGED_EVENT = 'message-inbox:changed'

export function notifyMessageInboxChanged(source?: symbol): void {
  eventBus.emit(MESSAGE_INBOX_CHANGED_EVENT, { source })
}

export function subscribeToMessageInboxChanges(
  options: MessageInboxSubscriptionOptions,
): () => void {
  return eventBus.on<MessageInboxChangedPayload>(MESSAGE_INBOX_CHANGED_EVENT, (payload) => {
    if (options.source && payload?.source === options.source) return
    if (options.shouldRefresh && !options.shouldRefresh()) return
    void options.refresh()
  })
}
