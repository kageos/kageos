import { computed, onMounted, onUnmounted, ref, type ComputedRef } from 'vue'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { copyDirectory } from '@/architecture/infrastructure/api/service-tree'
import type { ServiceTree } from '@/architecture/domain/types'
import { Logger } from '@/architecture/runtime/utils/logger'

const COPIED_DIRECTORY_KEY = 'copied_directory'

export interface UseServiceTreeClipboardOptions {
  treeData: ComputedRef<ServiceTree[]>
  currentFunction: ComputedRef<ServiceTree | null | undefined>
  currentNodeId: ComputedRef<number | string | null | undefined>
  onRefreshTree: () => void
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

export function useServiceTreeClipboard(options: UseServiceTreeClipboardOptions) {
  const { treeData, currentFunction, currentNodeId, onRefreshTree } = options

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
      Logger.error('[ServiceTreeClipboard]', '恢复复制内容失败', { error })
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
      Logger.error('[ServiceTreeClipboard]', '保存复制目录失败', { error })
    }
  }

  const handleCopy = (node: ServiceTree) => {
    if (node.type !== 'package') {
      ElMessage.warning('只能复制目录')
      return
    }

    const resolved = treeData.value.length && node.id != null
      ? findNodeByIdInTree(treeData.value, node.id)
      : null
    const toCopy = resolved?.type === 'package' && resolved.full_code_path ? resolved : node

    if (!toCopy.full_code_path) {
      ElMessage.warning('无法获取目录路径，请刷新树后重试')
      return
    }

    copiedDirectory.value = toCopy
    saveCopiedDirectory(toCopy)
    ElMessage.success(`已复制目录：${toCopy.name}`)
  }

  const handlePaste = async (targetNode?: ServiceTree) => {
    if (!copiedDirectory.value) {
      ElMessage.warning('没有可粘贴的目录')
      return
    }

    if (!copiedDirectory.value.full_code_path) {
      ElMessage.warning('复制的目录路径无效，请重新复制目录')
      copiedDirectory.value = null
      localStorage.removeItem(COPIED_DIRECTORY_KEY)
      return
    }

    const finalTargetNode = resolveTargetNode(targetNode)
    if (!finalTargetNode) {
      ElMessage.warning('请先选择一个目录作为粘贴目标')
      return
    }

    if (finalTargetNode.type !== 'package') {
      ElMessage.warning('只能粘贴到目录')
      return
    }

    const targetFullCodePath = finalTargetNode.full_code_path
    if (!targetFullCodePath) {
      ElMessage.warning('无法获取目标目录路径，请重新选择目标目录')
      return
    }

    if (copiedDirectory.value.full_code_path === targetFullCodePath) {
      ElMessage.warning('不能粘贴到自己')
      return
    }

    if (targetFullCodePath.startsWith(copiedDirectory.value.full_code_path + '/')) {
      ElMessage.warning('不能粘贴到自己的子目录')
      return
    }

    const sourcePathParts = copiedDirectory.value.full_code_path.split('/').filter(Boolean)
    const targetPathParts = targetFullCodePath.split('/').filter(Boolean)
    const isCrossApp = sourcePathParts.length >= 2
      && targetPathParts.length >= 2
      && (sourcePathParts[0] !== targetPathParts[0] || sourcePathParts[1] !== targetPathParts[1])

    let confirmMessage = `确定要将目录 "${copiedDirectory.value.name}" 复制到 "${finalTargetNode.name}" 吗？\n\n`
    confirmMessage += `源目录：${copiedDirectory.value.full_code_path}\n`
    confirmMessage += `目标目录：${targetFullCodePath}`
    if (isCrossApp) {
      confirmMessage += '\n\n⚠️ 注意：这是跨应用复制操作'
    }

    try {
      await ElMessageBox.confirm(confirmMessage, '确认粘贴', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'info'
      })

      isPasting.value = true
      const loadingNotify = ElNotification({
        title: '复制中',
        message: '正在复制目录，请稍候…',
        type: 'info',
        position: 'top-right',
        duration: 0
      })

      try {
        if (!finalTargetNode.app_id) {
          throw new Error('无法获取目标应用ID，请确保目标目录有效')
        }

        await copyDirectory({
          source_directory_path: copiedDirectory.value.full_code_path,
          target_directory_path: targetFullCodePath,
          target_app_id: finalTargetNode.app_id
        })

        loadingNotify.close()
        ElNotification.success({
          title: '复制完成',
          message: '目录已复制成功',
          position: 'top-right'
        })
        onRefreshTree()
      } catch (error: any) {
        loadingNotify.close()
        if (error !== 'cancel' && error !== 'close') {
          const errorMessage = error?.response?.data?.message || error?.message || '复制失败'
          ElMessage.error(errorMessage)
        }
      } finally {
        isPasting.value = false
      }
    } catch {
      // 用户取消
    }
  }

  const handleKeyDown = (event: KeyboardEvent) => {
    if ((event.ctrlKey || event.metaKey) && event.key === 'v') {
      const target = event.target as HTMLElement
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
        return
      }

      event.preventDefault()
      void handlePaste()
    }
  }

  onMounted(() => {
    restoreCopiedDirectory()
    window.addEventListener('keydown', handleKeyDown)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', handleKeyDown)
  })

  return {
    copiedDirectory: computed(() => copiedDirectory.value),
    handleCopy,
    handlePaste
  }
}
