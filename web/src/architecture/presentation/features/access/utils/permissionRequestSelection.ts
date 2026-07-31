function uniqueResourcePaths(paths: Iterable<string>): string[] {
  return [...new Set([...paths].filter(Boolean))]
}

export function getPermissionRequestTargetPaths(
  selectedPaths: Iterable<string>,
  readablePaths: ReadonlySet<string>,
  pendingPaths: ReadonlySet<string>,
): string[] {
  return uniqueResourcePaths(selectedPaths).filter(path => (
    !readablePaths.has(path) && !pendingPaths.has(path)
  ))
}

export function getPermissionRequestCheckedPaths(
  targetPaths: Iterable<string>,
  readablePaths: Iterable<string>,
  pendingPaths: Iterable<string>,
): string[] {
  return uniqueResourcePaths([
    ...readablePaths,
    ...pendingPaths,
    ...targetPaths,
  ])
}
