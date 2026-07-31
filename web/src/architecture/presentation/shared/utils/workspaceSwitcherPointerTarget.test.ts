import { describe, expect, it } from 'vitest'
import { isWorkspaceSwitcherOwnedPointerTarget } from './workspaceSwitcherPointerTarget'

describe('isWorkspaceSwitcherOwnedPointerTarget', () => {
  it('keeps pointer events inside the switcher root and popover', () => {
    const root = document.createElement('div')
    const rootChild = document.createElement('button')
    root.appendChild(rootChild)

    const popover = document.createElement('div')
    popover.className = 'workspace-switcher-popover'
    const popoverChild = document.createElement('button')
    popover.appendChild(popoverChild)

    expect(isWorkspaceSwitcherOwnedPointerTarget(rootChild, root)).toBe(true)
    expect(isWorkspaceSwitcherOwnedPointerTarget(popoverChild, root)).toBe(true)
  })

  it('treats the teleported workspace settings dialog and overlay as owned layers', () => {
    const root = document.createElement('div')
    const overlay = document.createElement('div')
    overlay.className = 'workspace-settings-dialog-overlay'
    const dialog = document.createElement('div')
    dialog.className = 'workspace-settings-dialog'
    const dialogInput = document.createElement('input')
    dialog.appendChild(dialogInput)
    overlay.appendChild(dialog)

    expect(isWorkspaceSwitcherOwnedPointerTarget(overlay, root)).toBe(true)
    expect(isWorkspaceSwitcherOwnedPointerTarget(dialogInput, root)).toBe(true)
  })

  it('allows an outside pointer event to close the workspace popover', () => {
    const root = document.createElement('div')
    const outside = document.createElement('button')

    expect(isWorkspaceSwitcherOwnedPointerTarget(outside, root)).toBe(false)
    expect(isWorkspaceSwitcherOwnedPointerTarget(null, root)).toBe(false)
  })
})
