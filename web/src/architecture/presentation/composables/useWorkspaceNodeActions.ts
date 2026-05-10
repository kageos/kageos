import { ref, type ComputedRef } from 'vue'
import type { RouteLocationNormalizedLoaded, Router } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  createDocs,
  deleteBoard,
  deleteDocs,
  deletePackage,
  deleteServiceTreeFunction
} from '@/api/service-tree'
import { isRootNode as isRootTreeNode } from '@/utils/tree-utils'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/types'

interface CreateDocsForm {
  name: string
  code: string
  description: string
  tags: string
  content: string
  summary: string
}

const createEmptyDocsForm = (): CreateDocsForm => ({
  name: '',
  code: '',
  description: '',
  tags: '',
  content: '',
  summary: ''
})

export interface UseWorkspaceNodeActionsOptions {
  route: RouteLocationNormalizedLoaded
  router: Router
  currentApp: ComputedRef<AppType | null>
  currentFunction: ComputedRef<ServiceTreeType | null>
  domainService: {
    setCurrentFunction: (node: ServiceTreeType | null) => void
  }
  handleRefreshTree: () => Promise<void>
  afterCreateNode: (response: ServiceTreeType) => Promise<void>
}

export function useWorkspaceNodeActions(options: UseWorkspaceNodeActionsOptions) {
  const { route, router, currentApp, currentFunction, domainService, handleRefreshTree, afterCreateNode } = options

  const createDocsDialogVisible = ref(false)
  const creatingDocs = ref(false)
  const currentDocsParentNode = ref<ServiceTreeType | null>(null)
  const createDocsForm = ref<CreateDocsForm>(createEmptyDocsForm())
  const createBoardDialogVisible = ref(false)
  const currentBoardParentNode = ref<ServiceTreeType | null>(null)

  const handleCreateDocs = (parentNode?: ServiceTreeType) => {
    if (!currentApp.value) {
      ElMessage.warning('请先选择应用')
      return
    }

    currentDocsParentNode.value = parentNode || null
    createDocsForm.value = createEmptyDocsForm()
    createDocsDialogVisible.value = true
  }

  const handleSubmitCreateDocs = async () => {
    if (!currentApp.value) {
      ElMessage.warning('请先选择应用')
      return
    }

    if (!createDocsForm.value.name.trim()) {
      ElMessage.warning('请输入文档名称')
      return
    }

    if (!createDocsForm.value.code.trim()) {
      ElMessage.warning('请输入文档英文标识')
      return
    }

    const codePattern = /^[a-z0-9_]+$/
    if (!codePattern.test(createDocsForm.value.code)) {
      ElMessage.warning('文档英文标识只能包含小写英文字母、数字和下划线')
      return
    }

    let code = createDocsForm.value.code.trim()
    if (!code.endsWith('.docs')) {
      code = `${code}.docs`
    }

    if (!createDocsForm.value.content.trim()) {
      ElMessage.warning('请输入文档内容')
      return
    }

    creatingDocs.value = true
    try {
      const parentFullCodePath = currentDocsParentNode.value?.full_code_path || ''
      const response = await createDocs({
        user: currentApp.value.user,
        app: currentApp.value.code,
        name: createDocsForm.value.name.trim(),
        code,
        parent_full_code_path: parentFullCodePath,
        description: createDocsForm.value.description.trim() || '',
        tags: createDocsForm.value.tags.trim() || '',
        content: createDocsForm.value.content.trim(),
        format: 'markdown',
        summary: createDocsForm.value.summary.trim() || ''
      })

      createDocsDialogVisible.value = false
      if (response && response.id) {
        ElMessage.success('文档节点创建成功')
        await afterCreateNode(response)
      } else {
        ElMessage.warning('创建文档节点成功，但未返回节点信息')
      }
    } catch (error: any) {
      ElMessage.error('创建文档节点失败: ' + (error.message || '未知错误'))
    } finally {
      creatingDocs.value = false
    }
  }

  const handleCloseCreateDocsDialog = () => {
    createDocsForm.value = createEmptyDocsForm()
    currentDocsParentNode.value = null
  }

  const handleCreateBoard = (parentNode?: ServiceTreeType) => {
    if (!currentApp.value) {
      ElMessage.warning('请先选择应用')
      return
    }

    currentBoardParentNode.value = parentNode ?? null
    createBoardDialogVisible.value = true
  }

  const handleDeleteBoard = async (node: ServiceTreeType) => {
    if (node.type !== 'board') {
      ElMessage.warning('只能删除讨论区节点')
      return
    }

    try {
      await ElMessageBox.confirm(
        `确定要删除讨论区 "${node.name}" 吗？将同时删除该版块下全部帖子，且无法恢复。`,
        '确认删除',
        { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }
      )
      await deleteBoard(node.id)
      ElMessage.success('讨论区已删除')
      await handleRefreshTree()
      if (currentFunction.value && currentFunction.value.id === node.id) {
        const parentPath = node.full_code_path?.split('/').slice(0, -1).join('/') || ''
        if (parentPath) {
          router.push(`/workspace${parentPath}`)
        }
      }
    } catch (error: any) {
      if (error !== 'cancel') {
        ElMessage.error('删除讨论区失败: ' + (error.message || '未知错误'))
      }
    }
  }

  const handleDeleteDoc = async (node: ServiceTreeType) => {
    if (node.type !== 'docs') {
      ElMessage.warning('只能删除文档节点')
      return
    }

    try {
      await ElMessageBox.confirm(
        `确定要删除文档 "${node.name}" 吗？此操作将删除文档内容和文档节点，且无法恢复。`,
        '确认删除',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      await deleteDocs(node.id)
      ElMessage.success('文档删除成功')
      await handleRefreshTree()

      if (currentFunction.value && currentFunction.value.id === node.id) {
        const parentPath = node.full_code_path?.split('/').slice(0, -1).join('/') || ''
        if (parentPath) {
          router.push(`/workspace${parentPath}`)
        }
      }
    } catch (error: any) {
      if (error !== 'cancel') {
        ElMessage.error('删除文档失败: ' + (error.message || '未知错误'))
      }
    }
  }

  const handleDocDeleted = async () => {
    await handleRefreshTree()

    if (currentFunction.value && currentFunction.value.type === 'docs') {
      const parentPath = currentFunction.value.full_code_path?.split('/').slice(0, -1).join('/') || ''
      if (parentPath) {
        router.push(`/workspace${parentPath}`)
      }
    }
  }

  const handleDeleteDirectory = async (node: ServiceTreeType) => {
    if (node.type !== 'package') {
      ElMessage.warning('只能删除目录节点')
      return
    }
    if (isRootTreeNode(node as ServiceTreeType)) {
      ElMessage.warning('不能删除工作空间根目录')
      return
    }

    try {
      await ElMessageBox.confirm(
        `确定要删除目录 "${node.name}" 吗？此操作将删除该目录及其下所有子目录、函数和文档，且无法恢复。`,
        '确认删除',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      await deletePackage(node.id)
      ElMessage.success('目录删除成功')

      const deletedPath = node.full_code_path || ''
      if (currentFunction.value) {
        const currentPath = currentFunction.value.full_code_path || ''
        if (currentPath === deletedPath || currentPath.startsWith(deletedPath + '/')) {
          domainService.setCurrentFunction(null)
          const parentPath = deletedPath.split('/').slice(0, -1).join('/') || ''
          if (parentPath) {
            router.replace({ path: `/workspace${parentPath}`, query: { ...route.query } })
          } else {
            router.replace({ path: route.path, query: { ...route.query, _id: undefined, _tab: undefined } })
          }
        }
      }

      await handleRefreshTree()
    } catch (error: any) {
      if (error !== 'cancel' && error !== 'close') {
        const errorMessage = error?.response?.data?.msg || error?.message || '删除目录失败'
        ElMessage.error(errorMessage)
      }
    }
  }

  const handleDeleteFunction = async (node: ServiceTreeType) => {
    if (node.type !== 'function') {
      ElMessage.warning('只能删除函数节点')
      return
    }

    try {
      await ElMessageBox.confirm(
        `确定要删除函数 "${node.name}" 吗？此操作不可恢复。`,
        '确认删除',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      await deleteServiceTreeFunction(node.id)
      ElMessage.success('删除成功')

      if (currentFunction.value && currentFunction.value.id === node.id) {
        domainService.setCurrentFunction(null)
        router.replace({
          path: route.path,
          query: { ...route.query, _id: undefined, _tab: undefined }
        })
      }

      await handleRefreshTree()
    } catch (error: any) {
      if (error !== 'cancel' && error !== 'close') {
        const errorMessage = error?.response?.data?.msg || error?.message || '删除失败'
        ElMessage.error(errorMessage)
      }
    }
  }

  const handleBulkDeleteNodes = async (nodes: ServiceTreeType[]) => {
    const deleteNodes = compactBulkDeleteNodes(nodes).filter((node) => {
      return node.type === 'package' || node.type === 'function' || node.type === 'docs' || node.type === 'board'
    })
    if (deleteNodes.length === 0) {
      ElMessage.warning('请选择可删除的节点')
      return
    }

    const names = deleteNodes.slice(0, 5).map((node) => node.name || node.code || node.full_code_path).join('、')
    const suffix = deleteNodes.length > 5 ? ` 等 ${deleteNodes.length} 个节点` : ''
    try {
      await ElMessageBox.confirm(
        `确定删除 ${deleteNodes.length} 个节点吗？${names}${suffix} 将被删除，且无法恢复。`,
        '批量删除',
        {
          confirmButtonText: '删除',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      const deletedPaths: string[] = []
      const deletedIds = new Set<number | string>()
      for (const node of deleteNodes) {
        if (node.type === 'package') {
          if (isRootTreeNode(node)) {
            continue
          }
          await deletePackage(node.id)
        } else if (node.type === 'function') {
          await deleteServiceTreeFunction(node.id)
        } else if (node.type === 'docs') {
          await deleteDocs(node.id)
        } else if (node.type === 'board') {
          await deleteBoard(node.id)
        }
        if (node.full_code_path) {
          deletedPaths.push(node.full_code_path)
        }
        deletedIds.add(node.id)
      }

      if (currentFunction.value) {
        const currentPath = currentFunction.value.full_code_path || ''
        const currentDeleted = deletedIds.has(currentFunction.value.id) || deletedPaths.some((path) => currentPath === path || currentPath.startsWith(path + '/'))
        if (currentDeleted) {
          domainService.setCurrentFunction(null)
          const parentPath = findNearestRemainingParentPath(currentPath, deletedPaths)
          if (parentPath) {
            router.replace({ path: `/workspace${parentPath}`, query: { ...route.query } })
          } else {
            router.replace({ path: route.path, query: { ...route.query, _id: undefined, _tab: undefined } })
          }
        }
      }

      await handleRefreshTree()
      ElMessage.success(`已删除 ${deletedIds.size} 个节点`)
    } catch (error: any) {
      if (error !== 'cancel' && error !== 'close') {
        const errorMessage = error?.response?.data?.msg || error?.message || '批量删除失败'
        ElMessage.error(errorMessage)
      }
    }
  }

  return {
    createDocsDialogVisible,
    creatingDocs,
    currentDocsParentNode,
    createDocsForm,
    createBoardDialogVisible,
    currentBoardParentNode,
    handleCreateDocs,
    handleSubmitCreateDocs,
    handleCloseCreateDocsDialog,
    handleCreateBoard,
    handleDeleteBoard,
    handleDeleteDoc,
    handleDocDeleted,
    handleDeleteDirectory,
    handleDeleteFunction,
    handleBulkDeleteNodes
  }
}

function compactBulkDeleteNodes(nodes: ServiceTreeType[]): ServiceTreeType[] {
  const seen = new Set<number | string>()
  const uniqueNodes = nodes
    .filter((node) => {
      if (!node.id || seen.has(node.id)) return false
      seen.add(node.id)
      return Boolean(node.full_code_path)
    })
    .sort((left, right) => {
      const leftPath = left.full_code_path || ''
      const rightPath = right.full_code_path || ''
      return leftPath.length - rightPath.length
    })

  const selectedPackagePaths: string[] = []
  const compacted: ServiceTreeType[] = []
  for (const node of uniqueNodes) {
    const nodePath = node.full_code_path || ''
    const coveredByPackage = selectedPackagePaths.some((packagePath) => nodePath.startsWith(packagePath + '/'))
    if (coveredByPackage) {
      continue
    }
    compacted.push(node)
    if (node.type === 'package' && !isRootTreeNode(node)) {
      selectedPackagePaths.push(nodePath)
    }
  }

  return compacted.sort((left, right) => getServiceTreePathDepth(right) - getServiceTreePathDepth(left))
}

function getServiceTreePathDepth(node: ServiceTreeType): number {
  return (node.full_code_path || '').split('/').filter(Boolean).length
}

function findNearestRemainingParentPath(currentPath: string, deletedPaths: string[]): string {
  const parts = currentPath.split('/').filter(Boolean)
  while (parts.length > 0) {
    parts.pop()
    const candidate = `/${parts.join('/')}`
    if (candidate !== '/' && !deletedPaths.some((path) => candidate === path || candidate.startsWith(path + '/'))) {
      return candidate
    }
  }
  return ''
}
