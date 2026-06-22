import { computed, onMounted, onUnmounted, ref, type ComputedRef } from 'vue'
import { ElLoading, ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { copyDirectory, installCapabilityBundleFromURL } from '@/architecture/presentation/context/api/service-tree'
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

interface HubInstallCommand {
  bundleUrl: string
  installKey?: string
}

function parseHubInstallCommand(text: string): HubInstallCommand | null {
  const tokens = tokenizeInstallCommand(text.trim())
  if (tokens.length < 3 || tokens[0] !== 'kageos' || tokens[1] !== 'install') {
    return null
  }
  const bundleUrl = tokens[2] || ''
  if (!/^https?:\/\//i.test(bundleUrl)) {
    return null
  }
  let installKey = ''
  for (let index = 3; index < tokens.length; index += 1) {
    const token = tokens[index] || ''
    if (token === '--key' || token === '--install-key') {
      installKey = tokens[index + 1] || ''
      index += 1
    } else if (token.startsWith('--key=')) {
      installKey = token.slice('--key='.length)
    } else if (token.startsWith('--install-key=')) {
      installKey = token.slice('--install-key='.length)
    }
  }
  return {
    bundleUrl,
    installKey: installKey || undefined
  }
}

function tokenizeInstallCommand(command: string): string[] {
  const tokens: string[] = []
  let current = ''
  let quote: '"' | "'" | '' = ''
  let escaping = false

  for (const char of command) {
    if (escaping) {
      current += char
      escaping = false
      continue
    }
    if (char === '\\') {
      escaping = true
      continue
    }
    if (quote) {
      if (char === quote) {
        quote = ''
      } else {
        current += char
      }
      continue
    }
    if (char === '"' || char === "'") {
      quote = char
      continue
    }
    if (/\s/.test(char)) {
      if (current) {
        tokens.push(current)
        current = ''
      }
      continue
    }
    current += char
  }
  if (current) {
    tokens.push(current)
  }
  return tokens
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
  const hubInstallPasteHandled = ref(false)

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

  const handleHubInstallPaste = async (command: HubInstallCommand, targetNode?: ServiceTree) => {
    if (isPasting.value) {
      return
    }
    const finalTargetNode = resolveTargetNode(targetNode)
    if (!finalTargetNode || finalTargetNode.type !== 'package') {
      ElMessage.warning('请先选择一个目录作为安装目标')
      return
    }
    if (!finalTargetNode.full_code_path) {
      ElMessage.warning('无法获取目标目录路径，请重新选择目标目录')
      return
    }

    try {
      await ElMessageBox.confirm(
        `确定要把 Hub 应用安装到 "${finalTargetNode.name}" 下吗？\n\n目标目录：${finalTargetNode.full_code_path}\n来源：${command.bundleUrl}\n\n同名目录或文件会按能力包导入规则覆盖。`,
        '安装 Hub 应用',
        {
          confirmButtonText: '安装',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      isPasting.value = true
      const loadingNotify = ElNotification({
        title: '安装中',
        message: '正在从 Hub 下载并安装应用，请稍候…',
        type: 'info',
        position: 'top-right',
        duration: 0
      })
      const loadingInstance = showBlockingLoading('正在安装 Hub 应用并更新函数列表，请稍候...')
      try {
        const resp = await installCapabilityBundleFromURL({
          target_directory_path: finalTargetNode.full_code_path,
          overwrite: true,
          force_diff: true,
          bundle_url: command.bundleUrl,
          install_key: command.installKey
        })
        loadingNotify.close()
        ElNotification.success({
          title: '安装完成',
          message: resp.message || `已安装到 ${resp.target_directory_path || finalTargetNode.full_code_path}`,
          position: 'top-right'
        })
        await onRefreshTree()
      } catch (error: any) {
        const errorMessage = error?.response?.data?.msg || error?.response?.data?.message || error?.message || '安装失败'
        ElMessage.error(errorMessage)
      } finally {
        loadingNotify.close()
        loadingInstance.close()
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

      if (copiedDirectory.value) {
        window.setTimeout(() => {
          if (!hubInstallPasteHandled.value) {
            void handlePaste()
          }
        }, 0)
      }
    }
  }

  const handleClipboardPaste = (event: ClipboardEvent) => {
    const target = event.target as HTMLElement
    if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
      return
    }
    const command = parseHubInstallCommand(event.clipboardData?.getData('text/plain') || '')
    if (!command) {
      return
    }
    hubInstallPasteHandled.value = true
    window.setTimeout(() => {
      hubInstallPasteHandled.value = false
    }, 100)
    event.preventDefault()
    void handleHubInstallPaste(command)
  }

  onMounted(() => {
    restoreCopiedDirectory()
    window.addEventListener('keydown', handleKeyDown)
    window.addEventListener('paste', handleClipboardPaste)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', handleKeyDown)
    window.removeEventListener('paste', handleClipboardPaste)
  })

  return {
    copiedDirectory: computed(() => copiedDirectory.value),
    isPasting: computed(() => isPasting.value),
    handleCopy,
    handlePaste
  }
}
