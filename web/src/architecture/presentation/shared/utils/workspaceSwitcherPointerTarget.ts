const WORKSPACE_SWITCHER_OWNED_PORTAL_SELECTOR = [
  '.workspace-switcher-popover',
  '.workspace-settings-dialog',
  '.workspace-settings-dialog-overlay',
  '.el-message-box',
  '.el-overlay-message-box',
].join(', ')

export function isWorkspaceSwitcherOwnedPointerTarget(
  target: EventTarget | null,
  switcherRoot: HTMLElement | null,
): boolean {
  if (!(target instanceof Node)) return false
  if (switcherRoot?.contains(target)) return true

  const targetElement = target instanceof Element ? target : target.parentElement
  return Boolean(targetElement?.closest(WORKSPACE_SWITCHER_OWNED_PORTAL_SELECTOR))
}
