import type { FileItem } from '../filesWidgetTypes'
import { stringifyFileRefs } from '../filesWidgetTypes'

interface UseFilesValueSyncOptions {
  value: () => any
  fieldPath: () => string
  currentFiles: () => FileItem[]
  setCurrentFiles?: (files: FileItem[]) => void
  persistDescription?: (file: FileItem, description: string) => Promise<void>
  formDataStore: {
    setValue: (path: string, value: any) => void
  }
  emitUpdateModelValue: (value: any) => void
}

export function useFilesValueSync(options: UseFilesValueSyncOptions) {
  const emitFieldValue = (newFieldValue: any): void => {
    options.formDataStore.setValue(options.fieldPath(), newFieldValue)
    options.emitUpdateModelValue(newFieldValue)
  }

  const updateFiles = async (files: FileItem[]): Promise<void> => {
    const refs = files.map(file => file.ref).filter(Boolean)
    const raw = stringifyFileRefs(refs)

    emitFieldValue({
      raw,
      display: `${files.length} 个文件`,
      meta: {},
    })
  }

  const handleDeleteFile = (index: number): void => {
    const currentFilesList = options.currentFiles()
    const newFiles = [...currentFilesList]
    newFiles.splice(index, 1)
    void updateFiles(newFiles)
  }

  const handleUpdateDescription = (index: number, description: string): void => {
    const currentFilesList = options.currentFiles()
    if (index < 0 || index >= currentFilesList.length) {
      return
    }

    const newFiles = [...currentFilesList]
    const fileToUpdate = newFiles[index]
    if (fileToUpdate) {
      newFiles[index] = { ...fileToUpdate, description }
      options.setCurrentFiles?.(newFiles)
      if (options.persistDescription) {
        void options.persistDescription(fileToUpdate, description).catch(() => {
          options.setCurrentFiles?.(currentFilesList)
        })
      }
    }
  }

  return {
    updateFiles,
    handleDeleteFile,
    handleUpdateDescription,
  }
}
