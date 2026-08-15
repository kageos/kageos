import { describe, expect, it, vi } from 'vitest'
import {
  notifyMessageInboxChanged,
  subscribeToMessageInboxChanges,
} from './messageInboxSync'

describe('message inbox synchronization', () => {
  it('refreshes other visible consumers but skips the publisher and hidden inboxes', () => {
    const publisher = Symbol('publisher')
    const publisherRefresh = vi.fn()
    const visibleInboxRefresh = vi.fn()
    const hiddenInboxRefresh = vi.fn()
    const workspaceTreeRefresh = vi.fn()

    const unsubscribers = [
      subscribeToMessageInboxChanges({ source: publisher, refresh: publisherRefresh }),
      subscribeToMessageInboxChanges({ source: Symbol('visible'), refresh: visibleInboxRefresh }),
      subscribeToMessageInboxChanges({
        source: Symbol('hidden'),
        shouldRefresh: () => false,
        refresh: hiddenInboxRefresh,
      }),
      subscribeToMessageInboxChanges({ refresh: workspaceTreeRefresh }),
    ]

    notifyMessageInboxChanged(publisher)

    expect(publisherRefresh).not.toHaveBeenCalled()
    expect(hiddenInboxRefresh).not.toHaveBeenCalled()
    expect(visibleInboxRefresh).toHaveBeenCalledTimes(1)
    expect(workspaceTreeRefresh).toHaveBeenCalledTimes(1)
    unsubscribers.forEach(unsubscribe => unsubscribe())
  })

  it('stops refreshing after the consumer unsubscribes', () => {
    const refresh = vi.fn()
    const unsubscribe = subscribeToMessageInboxChanges({ refresh })

    unsubscribe()
    notifyMessageInboxChanged()

    expect(refresh).not.toHaveBeenCalled()
  })
})
