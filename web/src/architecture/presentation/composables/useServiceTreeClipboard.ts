import { computed, onMounted, ref, type ComputedRef } from 'vue'
import { ElLoading, ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { copyDirectory } from '@/architecture/presentation/context/api/service-tree'
import type { ServiceTree } from '@/architecture/domain/types'
import { Logger } from '@/architecture/shared/logger'

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

function showBlockingLoading(text: string) {
  return ElLoading.service({
    lock: true,
    text,
    background: 'rgba(15, 23, 42, 0.36)'
  })
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
    ElMessage.success(`已复制目录：${toCopy.name}，可粘贴到目标父目录`)
  }

  const handlePaste = async (targetNode?: ServiceTree) => {
    if (isPasting.value) {
      return
    }
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

    const sourceCode = lastPathSegment(copiedDirectory.value.full_code_path)
    if (!sourceCode) {
      ElMessage.warning('复制的目录路径无效，请重新复制目录')
      return
    }
    const finalTargetPath = joinFullCodePath(targetFullCodePath, sourceCode)

    if (copiedDirectory.value.full_code_path === finalTargetPath) {
      ElMessage.warning('不能粘贴到原目录本身，请选择其他父目录')
      return
    }

    if (targetFullCodePath.startsWith(copiedDirectory.value.full_code_path + '/')) {
      ElMessage.warning('不能粘贴到自己的子目录')
      return
    }
    if (copiedDirectory.value.full_code_path.startsWith(finalTargetPath + '/')) {
      ElMessage.warning('不能用子目录覆盖父目录，请先把副本放到兄弟目录或其他目录')
      return
    }

    const sourcePathParts = copiedDirectory.value.full_code_path.split('/').filter(Boolean)
    const targetPathParts = targetFullCodePath.split('/').filter(Boolean)
    const isCrossApp = sourcePathParts.length >= 2
      && targetPathParts.length >= 2
      && (sourcePathParts[0] !== targetPathParts[0] || sourcePathParts[1] !== targetPathParts[1])

    const existingTarget = findNodeByFullCodePath(treeData.value, finalTargetPath)
    const willReplace = existingTarget?.type === 'package'

    let confirmTitle = '确认粘贴'
    let confirmType: 'info' | 'warning' = 'info'
    let confirmMessage = ''
    if (willReplace) {
      confirmTitle = '覆盖同名目录'
      confirmType = 'warning'
      confirmMessage = `目标父目录下已存在同名目录 "${sourceCode}"。\n\n`
      confirmMessage += `确定要用复制内容完全替换它吗？\n\n`
      confirmMessage += `源目录：${copiedDirectory.value.full_code_path}\n`
      confirmMessage += `目标目录：${finalTargetPath}\n\n`
      confirmMessage += `目录名 ${sourceCode} 会保持不变，目标应用数据库和业务数据会保留；旧代码会在编译成功后清理，失败会自动恢复。`
    } else {
      confirmMessage = `确定要将目录 "${copiedDirectory.value.name}" 粘贴到 "${finalTargetNode.name}" 下吗？\n\n`
      confirmMessage += `源目录：${copiedDirectory.value.full_code_path}\n`
      confirmMessage += `目标目录：${finalTargetPath}`
    }
    if (isCrossApp) {
      confirmMessage += '\n\n⚠️ 注意：这是跨应用复制操作'
    }

    try {
      const { value } = await ElMessageBox.prompt(confirmMessage, confirmTitle, {
        confirmButtonText: willReplace ? '覆盖同名目录' : '确定',
        cancelButtonText: '取消',
        type: confirmType,
        inputValue: copiedDirectory.value.name || sourceCode,
        inputPlaceholder: '请输入目录中文名称',
        inputValidator: (inputValue) => {
          if (String(inputValue || '').trim().length > 80) {
            return '目录中文名称不能超过 80 个字符'
          }
          return true
        }
      })
      const targetDirectoryName = String(value || '').trim() || copiedDirectory.value.name || sourceCode

      isPasting.value = true
      const loadingNotify = ElNotification({
        title: willReplace ? '覆盖中' : '复制中',
        message: willReplace ? '正在替换同名目录，请稍候…' : '正在复制目录，请稍候…',
        type: 'info',
        position: 'top-right',
        duration: 0
      })
      const loadingInstance = showBlockingLoading(willReplace ? '正在覆盖同名目录，请稍候...' : '正在复制目录并更新函数列表，请稍候...')

      try {
        if (!finalTargetNode.app_id) {
          throw new Error('无法获取目标应用ID，请确保目标目录有效')
        }

        await copyDirectory({
          source_directory_path: copiedDirectory.value.full_code_path,
          target_directory_path: targetFullCodePath,
          target_app_id: finalTargetNode.app_id,
          target_directory_name: targetDirectoryName,
          replace_existing: willReplace
        })

        loadingNotify.close()
        ElNotification.success({
          title: willReplace ? '覆盖完成' : '复制完成',
          message: willReplace ? `已替换 ${finalTargetPath}` : `已复制到 ${finalTargetPath}`,
          position: 'top-right'
        })
        await onRefreshTree()
      } catch (error: any) {
        if (error !== 'cancel' && error !== 'close') {
          const errorMessage = error?.response?.data?.message || error?.message || '复制失败'
          ElMessage.error(errorMessage)
        }
      } finally {
        loadingNotify.close()
        loadingInstance.close()
        isPasting.value = false
      }
    } catch {
      // 用户取消
    }
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
