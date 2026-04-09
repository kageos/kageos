import { ref, type Ref } from 'vue'
import type { FileItem } from '../filesWidgetTypes'

interface UseFilesDescriptionDialogOptions {
  currentFiles: Ref<FileItem[]>
  handleUpdateDescription: (index: number, description: string) => void
}

export function useFilesDescriptionDialog(options: UseFilesDescriptionDialogOptions) {
  const descriptionDialogVisible = ref(false)
  const editingDescriptionIndex = ref<number>(-1)
  const editingDescription = ref<string>('')

  function resetDialog(): void {
    descriptionDialogVisible.value = false
    editingDescriptionIndex.value = -1
    editingDescription.value = ''
  }

  function handleEditDescription(index: number): void {
    const currentFilesList = options.currentFiles.value
    if (index < 0 || index >= currentFilesList.length) {
      return
    }
    const file = currentFilesList[index]
    if (!file) {
      return
    }
    editingDescriptionIndex.value = index
    editingDescription.value = file.description || ''
    descriptionDialogVisible.value = true
  }

  function handleSaveDescription(): void {
    if (editingDescriptionIndex.value >= 0) {
      options.handleUpdateDescription(editingDescriptionIndex.value, editingDescription.value)
    }
    resetDialog()
  }

  function handleCancelDescription(): void {
    resetDialog()
  }

  return {
    descriptionDialogVisible,
    editingDescriptionIndex,
    editingDescription,
    handleEditDescription,
    handleSaveDescription,
    handleCancelDescription,
  }
}
