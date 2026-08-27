import { describe, expect, it, vi } from 'vitest'
import {
  getCenteredTreeScrollTop,
  scrollCurrentPermissionTreeNodeIntoView,
} from './permissionTreeScroll'

describe('permission tree scrolling', () => {
  it('centers the current tree node inside its own scroll container', () => {
    const container = {
      scrollTop: 240,
      clientHeight: 320,
      getBoundingClientRect: () => ({ top: 100 }),
    }
    const target = {
      getBoundingClientRect: () => ({ top: 480, height: 34 }),
    }

    expect(getCenteredTreeScrollTop(container as HTMLElement, target as HTMLElement)).toBe(477)
  })

  it('does not scroll past the top of the tree', () => {
    const container = {
      scrollTop: 0,
      clientHeight: 320,
      getBoundingClientRect: () => ({ top: 100 }),
    }
    const target = {
      getBoundingClientRect: () => ({ top: 110, height: 34 }),
    }

    expect(getCenteredTreeScrollTop(container as HTMLElement, target as HTMLElement)).toBe(0)
  })

  it('scrolls only when the current node is rendered', () => {
    const scrollTo = vi.fn()
    const target = document.createElement('div')
    vi.spyOn(target, 'getBoundingClientRect').mockReturnValue({
      top: 480,
      height: 34,
    } as DOMRect)

    const container = document.createElement('div')
    Object.defineProperties(container, {
      scrollTop: { value: 240, writable: true },
      clientHeight: { value: 320 },
      scrollTo: { value: scrollTo },
    })
    vi.spyOn(container, 'getBoundingClientRect').mockReturnValue({ top: 100 } as DOMRect)
    vi.spyOn(container, 'querySelector').mockReturnValue(target)

    expect(scrollCurrentPermissionTreeNodeIntoView(container)).toBe(true)
    expect(scrollTo).toHaveBeenCalledWith({ top: 477, behavior: 'smooth' })

    vi.mocked(container.querySelector).mockReturnValue(null)
    expect(scrollCurrentPermissionTreeNodeIntoView(container)).toBe(false)
    expect(scrollTo).toHaveBeenCalledTimes(1)
  })
})
