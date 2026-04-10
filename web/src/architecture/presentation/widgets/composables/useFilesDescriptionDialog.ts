import { ref, type Ref } from 'vue'
import type { FileItem } from '../filesWidgetTypes'

interface UseFilesDescriptionDialogOptions {
  currentFiles: Ref<FileItem[]>
  handleUpdateDescription: (index: number, description: string) => void
}

export function useFilesDescriptionDialog(options: UseFilesDescriptionDialogOptions) {
  const editingDescriptionIndex = ref<number>(-1)
  const editingDescription = ref<string>('')

  function resetEditingState(): void {
    editingDescriptionIndex.value = -1
    editingDescription.value = ''
  }

  function handleEditDescription(index: number): void {
    const currentFilesList = options.currentFiles.value
    if (index < 0 || index >= currentFilesList.length) {
      return
    }
    if (editingDescriptionIndex.value === index) {
      resetEditingState()
      return
    }
    const file = currentFilesList[index]
    if (!file) {
      return
    }
    editingDescriptionIndex.value = index
    editingDescription.value = file.description || ''
  }

  function updateEditingDescription(value: string): void {
    editingDescription.value = value
  }

  function handleSaveDescription(): void {
    if (editingDescriptionIndex.value >= 0) {
      options.handleUpdateDescription(editingDescriptionIndex.value, editingDescription.value)
    }
    resetEditingState()
  }

  function handleCancelDescription(): void {
    resetEditingState()
  }

  return {
    editingDescriptionIndex,
    editingDescription,
    handleEditDescription,
    updateEditingDescription,
    handleSaveDescription,
    handleCancelDescription,
  }
}
