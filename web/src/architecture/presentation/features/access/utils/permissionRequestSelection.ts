function uniqueResourcePaths(paths: Iterable<string>): string[] {
  return [...new Set([...paths].filter(Boolean))]
}

export function isDescendantResourcePath(path: string, ancestorPath: string): boolean {
  return Boolean(path && ancestorPath && path !== ancestorPath && path.startsWith(`${ancestorPath}/`))
}

export function findNearestPermissionRequestAncestor(
  path: string,
  candidatePaths: Iterable<string>,
): string | undefined {
  return uniqueResourcePaths(candidatePaths)
    .filter(candidate => isDescendantResourcePath(path, candidate))
    .sort((left, right) => right.length - left.length)[0]
}

export function getPermissionRequestTargetPaths(
  selectedPaths: Iterable<string>,
  coveredPaths: ReadonlySet<string>,
  pendingPaths: ReadonlySet<string>,
  inheritingPendingPaths: ReadonlySet<string> = pendingPaths,
): string[] {
  const inheritingPending = uniqueResourcePaths(inheritingPendingPaths)
  const requestablePaths = uniqueResourcePaths(selectedPaths).filter(path => (
    !coveredPaths.has(path)
    && !pendingPaths.has(path)
    && !findNearestPermissionRequestAncestor(path, inheritingPending)
  ))

  return requestablePaths.filter(path => (
    !findNearestPermissionRequestAncestor(path, requestablePaths)
  ))
}

export function getPermissionRequestCheckedPaths(
  targetPaths: Iterable<string>,
  coveredPaths: Iterable<string>,
  pendingPaths: Iterable<string>,
): string[] {
  return uniqueResourcePaths([
    ...coveredPaths,
    ...pendingPaths,
    ...targetPaths,
  ])
}
