import type { FilesData, FileItem } from '../filesWidgetTypes'

interface UseFilesValueSyncOptions {
  value: () => any
  fieldPath: () => string
  currentFiles: () => FileItem[]
  formDataStore: {
    setValue: (path: string, value: any) => void
  }
  emitUpdateModelValue: (value: any) => void
  resolveUploadUser: () => string
}

export function useFilesValueSync(options: UseFilesValueSyncOptions) {
  const emitFieldValue = (newFieldValue: any): void => {
    options.formDataStore.setValue(options.fieldPath(), newFieldValue)
    options.emitUpdateModelValue(newFieldValue)
  }

  const updateFiles = async (files: FileItem[]): Promise<void> => {
    const currentValue = options.value()
    const data = (currentValue?.raw as FilesData) || {
      files: [],
      remark: '',
      metadata: {},
      upload_user: '',
      widget_type: 'files',
      data_type: 'struct',
    }

    const uploadUser = data.upload_user || options.resolveUploadUser()

    const newData: FilesData = {
      ...data,
      files,
      upload_user: uploadUser,
      widget_type: 'files',
      data_type: 'struct',
    }

    emitFieldValue({
      raw: newData,
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
      void updateFiles(newFiles)
    }
  }

  const updateRemark = (remarkValue: string): void => {
    const currentValue = options.value()
    const data = (currentValue?.raw as FilesData) || {
      files: [],
      remark: '',
      metadata: {},
    }

    const newData: FilesData = {
      ...data,
      remark: remarkValue,
    }

    emitFieldValue({
      raw: newData,
      display: `${data.files.length} 个文件`,
      meta: {},
    })
  }

  return {
    updateFiles,
    handleDeleteFile,
    handleUpdateDescription,
    updateRemark,
  }
}
