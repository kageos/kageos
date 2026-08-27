export function getCenteredTreeScrollTop(
  container: Pick<HTMLElement, 'scrollTop' | 'clientHeight' | 'getBoundingClientRect'>,
  target: Pick<HTMLElement, 'getBoundingClientRect'>,
): number {
  const containerRect = container.getBoundingClientRect()
  const targetRect = target.getBoundingClientRect()
  const targetTop = container.scrollTop + targetRect.top - containerRect.top

  return Math.max(0, targetTop - (container.clientHeight - targetRect.height) / 2)
}

export function scrollCurrentPermissionTreeNodeIntoView(
  container: HTMLElement | null | undefined,
  behavior: ScrollBehavior = 'smooth',
): boolean {
  if (!container) return false

  const target = container.querySelector<HTMLElement>(
    '.el-tree-node.is-current > .el-tree-node__content',
  )
  if (!target) return false

  container.scrollTo({
    top: getCenteredTreeScrollTop(container, target),
    behavior,
  })
  return true
}
