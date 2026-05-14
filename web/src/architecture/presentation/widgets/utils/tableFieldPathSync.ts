import { useFormDataStore } from '@/architecture/runtime/stores/formData'

function getRowFieldMatch(tablePath: string, fieldPath: string): RegExpMatchArray | null {
  if (!fieldPath.startsWith(`${tablePath}[`)) {
    return null
  }

  const suffix = fieldPath.slice(tablePath.length)
  return suffix.match(/^\[(\d+)\](.*)$/)
}

export function reindexTableRowFieldPaths(
  formDataStore: ReturnType<typeof useFormDataStore>,
  tablePath: string,
  removedIndex: number
): void {
  const allValues = formDataStore.getAllValues()
  const pathsToDelete: string[] = []
  const pathsToMove: Array<{ from: string; to: string }> = []

  formDataStore.getAllFieldPaths().forEach((fieldPath) => {
    const match = getRowFieldMatch(tablePath, fieldPath)
    if (!match) {
      return
    }

    const rowIndex = Number(match[1])
    const restPath = match[2]
    if (Number.isNaN(rowIndex)) {
      return
    }

    if (rowIndex === removedIndex) {
      pathsToDelete.push(fieldPath)
      return
    }

    if (rowIndex > removedIndex) {
      pathsToDelete.push(fieldPath)
      pathsToMove.push({
        from: fieldPath,
        to: `${tablePath}[${rowIndex - 1}]${restPath}`
      })
    }
  })

  pathsToDelete.forEach((fieldPath) => {
    formDataStore.deleteValue(fieldPath)
  })

  pathsToMove.forEach(({ from, to }) => {
    const value = allValues[from]
    if (value) {
      formDataStore.setValue(to, value)
    }
  })
}
