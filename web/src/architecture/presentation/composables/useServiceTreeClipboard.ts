import { computed, h, onMounted, ref, type ComputedRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { copyDirectory } from '@/architecture/presentation/context/api/service-tree'
import type { ServiceTree } from '@/architecture/domain/types'
import { Logger } from '@/architecture/shared/logger'
import { Z_INDEX } from '@/architecture/presentation/constants/zIndex'

const COPIED_DIRECTORY_KEY = 'copied_directory'

export interface UseServiceTreeClipboardOptions {
  treeData: ComputedRef<ServiceTree[]>
  currentFunction: ComputedRef<ServiceTree | null | undefined>
  currentNodeId: ComputedRef<number | string | null | undefined>
  onRefreshTree: () => void | Promise<void>
}

function findNodeByIdInTree(nodes: ServiceTree[], id: number | string): ServiceTree | null {
  for (const node of nodes) {
    if (Number(node.id) === Number(id)) {
      return node
    }
    if (node.children?.length) {
      const found = findNodeByIdInTree(node.children, id)
      if (found) {
        return found
      }
    }
  }
  return null
}

function findNodeByFullCodePath(nodes: ServiceTree[], fullCodePath: string): ServiceTree | null {
  for (const node of nodes) {
    if (node.full_code_path === fullCodePath) {
      return node
    }
    if (node.children?.length) {
      const found = findNodeByFullCodePath(node.children, fullCodePath)
      if (found) {
        return found
      }
    }
  }
  return null
}

function lastPathSegment(fullCodePath: string): string {
  const parts = fullCodePath.split('/').filter(Boolean)
  return parts[parts.length - 1] || ''
}

function joinFullCodePath(parentPath: string, childCode: string): string {
  return `${parentPath.replace(/\/+$/, '')}/${childCode}`
}

export function useServiceTreeClipboard(options: UseServiceTreeClipboardOptions) {
  const { treeData, currentFunction, currentNodeId, onRefreshTree } = options
  const { t } = useI18n()

  const copiedDirectory = ref<ServiceTree | null>(null)
  const isPasting = ref(false)

  const resolveTargetNode = (targetNode?: ServiceTree) => {
    if (targetNode?.id != null) {
      const fromTree = findNodeByIdInTree(treeData.value, targetNode.id)
      if (fromTree?.type === 'package') {
        return fromTree
      }
    }

    if (targetNode?.type === 'package') {
      return targetNode
    }

    if (currentFunction.value?.type === 'package') {
      return currentFunction.value
    }

    if (currentNodeId.value != null) {
      return findNodeByIdInTree(treeData.value, currentNodeId.value) ?? undefined
    }

    return undefined
  }

  const restoreCopiedDirectory = () => {
    try {
      const savedDirectory = localStorage.getItem(COPIED_DIRECTORY_KEY)
      if (savedDirectory) {
        const parsed = JSON.parse(savedDirectory)
        if (parsed && parsed.full_code_path && parsed.name) {
          copiedDirectory.value = parsed as ServiceTree
        } else {
          localStorage.removeItem(COPIED_DIRECTORY_KEY)
        }
      }
    } catch (error) {
      Logger.error('[ServiceTreeClipboard]', 'Failed to restore copied directory', { error })
      localStorage.removeItem(COPIED_DIRECTORY_KEY)
    }
  }

  const saveCopiedDirectory = (node: ServiceTree) => {
    try {
      const dataToSave = {
        id: node.id,
        name: node.name,
        full_code_path: node.full_code_path,
        app_id: node.app_id,
        type: node.type
      }
      localStorage.setItem(COPIED_DIRECTORY_KEY, JSON.stringify(dataToSave))
    } catch (error) {
      Logger.error('[ServiceTreeClipboard]', 'Failed to save copied directory', { error })
    }
  }

  const handleCopy = (node: ServiceTree) => {
    if (node.type !== 'package') {
      ElMessage.warning(t('serviceTreeClipboard.copyDirectoryOnly'))
      return
    }

    const resolved = treeData.value.length && node.id != null
      ? findNodeByIdInTree(treeData.value, node.id)
      : null
    const toCopy = resolved?.type === 'package' && resolved.full_code_path ? resolved : node

    if (!toCopy.full_code_path) {
      ElMessage.warning(t('serviceTreeClipboard.pathMissingRefresh'))
      return
    }

    copiedDirectory.value = toCopy
    saveCopiedDirectory(toCopy)
    ElMessage.success(t('serviceTreeClipboard.directoryCopied', { name: toCopy.name }))
  }

  const handlePaste = async (targetNode?: ServiceTree) => {
    if (isPasting.value) {
      return
    }
    if (!copiedDirectory.value) {
      ElMessage.warning(t('serviceTreeClipboard.noDirectoryToPaste'))
      return
    }

    if (!copiedDirectory.value.full_code_path) {
      ElMessage.warning(t('serviceTreeClipboard.invalidCopiedPath'))
      copiedDirectory.value = null
      localStorage.removeItem(COPIED_DIRECTORY_KEY)
      return
    }

    const finalTargetNode = resolveTargetNode(targetNode)
    if (!finalTargetNode) {
      ElMessage.warning(t('serviceTreeClipboard.selectPasteTarget'))
      return
    }

    if (finalTargetNode.type !== 'package') {
      ElMessage.warning(t('serviceTreeClipboard.pasteDirectoryOnly'))
      return
    }

    const targetFullCodePath = finalTargetNode.full_code_path
    if (!targetFullCodePath) {
      ElMessage.warning(t('serviceTreeClipboard.targetPathMissing'))
      return
    }

    const sourceCode = lastPathSegment(copiedDirectory.value.full_code_path)
    if (!sourceCode) {
      ElMessage.warning(t('serviceTreeClipboard.invalidCopiedPath'))
      return
    }
    const finalTargetPath = joinFullCodePath(targetFullCodePath, sourceCode)

    if (copiedDirectory.value.full_code_path === finalTargetPath) {
      ElMessage.warning(t('serviceTreeClipboard.cannotPasteToSelf'))
      return
    }

    if (targetFullCodePath.startsWith(copiedDirectory.value.full_code_path + '/')) {
      ElMessage.warning(t('serviceTreeClipboard.cannotPasteToChild'))
      return
    }
    if (copiedDirectory.value.full_code_path.startsWith(finalTargetPath + '/')) {
      ElMessage.warning(t('serviceTreeClipboard.cannotOverwriteParentWithChild'))
      return
    }

    const sourcePathParts = copiedDirectory.value.full_code_path.split('/').filter(Boolean)
    const targetPathParts = targetFullCodePath.split('/').filter(Boolean)
    const isCrossApp = sourcePathParts.length >= 2
      && targetPathParts.length >= 2
      && (sourcePathParts[0] !== targetPathParts[0] || sourcePathParts[1] !== targetPathParts[1])

    const existingTarget = findNodeByFullCodePath(treeData.value, finalTargetPath)
    const willReplace = existingTarget?.type === 'package'

    let confirmMessage = ''
    if (willReplace) {
      confirmMessage = t('serviceTreeClipboard.replaceConfirmMessage', {
        sourceCode,
        sourcePath: copiedDirectory.value.full_code_path,
        targetPath: finalTargetPath,
      })
      if (isCrossApp) {
        confirmMessage += `\n\n${t('serviceTreeClipboard.crossAppWarning')}`
      }
    }

    if (willReplace) {
      try {
        await ElMessageBox.confirm(confirmMessage, t('serviceTreeClipboard.replaceSameNameTitle'), {
          confirmButtonText: t('serviceTreeClipboard.replaceSameNameTitle'),
          cancelButtonText: t('common.cancel'),
          type: 'warning',
          customClass: 'service-tree-paste-confirm',
        })
      } catch {
        return
      }
    }

    const sourceDirectoryPath = copiedDirectory.value.full_code_path
    const sourceDirectoryName = copiedDirectory.value.name || sourceCode
    const targetDirectoryName = copiedDirectory.value.name || sourceCode

    const runPasteTask = async () => {
      isPasting.value = true
      const progressNotification = ElNotification({
        ...directoryTaskNotificationOptions(),
        type: willReplace ? 'warning' : 'info',
        title: willReplace ? t('serviceTreeClipboard.replacing') : t('serviceTreeClipboard.copying'),
        message: renderPasteProgressMessage({
          sourceName: sourceDirectoryName,
          targetName: finalTargetNode.name,
          targetPath: finalTargetPath,
          isCrossApp,
        }),
        duration: 0,
        showClose: true,
      }) as NotificationHandle

      try {
        if (!finalTargetNode.app_id) {
          throw new Error(t('serviceTreeClipboard.targetAppMissing'))
        }

        const resp = await copyDirectory({
          source_directory_path: sourceDirectoryPath,
          target_directory_path: targetFullCodePath,
          target_app_id: finalTargetNode.app_id,
          target_directory_name: targetDirectoryName,
          replace_existing: willReplace
        })

        progressNotification.close()
        ElNotification({
          ...directoryTaskNotificationOptions(),
          type: 'success',
          title: willReplace ? t('serviceTreeClipboard.replaceComplete') : t('serviceTreeClipboard.copyComplete'),
          message: renderPasteResultMessage(resp, finalTargetPath),
          duration: 9000
        })
        await onRefreshTree()
      } catch (error: any) {
        progressNotification.close()
        const errorMessage = error?.response?.data?.msg
          || error?.response?.data?.message
          || error?.message
          || t('serviceTreeClipboard.copyFailed')
        ElNotification({
          ...directoryTaskNotificationOptions(),
          type: 'error',
          title: t('serviceTreeClipboard.copyFailed'),
          message: errorMessage,
          duration: 0
        })
      } finally {
        isPasting.value = false
      }
    }

    void runPasteTask()
  }

  type NotificationHandle = {
    close: () => void
  }

  function directoryTaskNotificationOptions() {
    return {
      appendTo: 'body',
      customClass: 'workspace-task-notification',
      offset: 72,
      position: 'top-right' as const,
      zIndex: Z_INDEX.notification
    }
  }

  function renderPasteProgressMessage(options: {
    sourceName: string
    targetName: string
    targetPath: string
    isCrossApp: boolean
  }) {
    const lines = [
      t('serviceTreeClipboard.pasteRunningMessage', {
        sourceName: options.sourceName,
        targetName: options.targetName,
      }),
      t('serviceTreeClipboard.targetPathLine', { path: options.targetPath }),
    ]
    if (options.isCrossApp) {
      lines.push(t('serviceTreeClipboard.crossAppWarning'))
    }

    return h(
      'div',
      { class: 'workspace-update-notification' },
      lines.map((line) => h('div', { class: 'workspace-update-notification-line' }, line))
    )
  }

  function renderPasteResultMessage(resp: Awaited<ReturnType<typeof copyDirectory>>, fallbackPath: string) {
    const lines = [
      t('serviceTreeClipboard.targetPathLine', { path: resp.target_directory_path || fallbackPath }),
      t('serviceTreeClipboard.copyResultCount', {
        directories: resp.directory_count || 0,
        files: resp.file_count || 0,
        docs: resp.doc_count || 0,
        agentTasks: resp.agent_task_count || 0,
      })
    ]
    if (resp.old_version || resp.new_version) {
      lines.push(t('serviceTreeClipboard.versionChanged', {
        oldVersion: resp.old_version || '-',
        newVersion: resp.new_version || '-',
      }))
    }

    return h(
      'div',
      { class: 'workspace-update-notification' },
      lines.map((line) => h('div', { class: 'workspace-update-notification-line' }, line))
    )
  }

  onMounted(() => {
    restoreCopiedDirectory()
  })

  return {
    copiedDirectory: computed(() => copiedDirectory.value),
    isPasting: computed(() => isPasting.value),
    handleCopy,
    handlePaste
  }
}
