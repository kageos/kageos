import { ref, type ComputedRef } from 'vue'
import type { RouteLocationNormalizedLoaded, Router } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElLoading, ElMessage, ElMessageBox } from 'element-plus'
import {
  createDocs,
  deleteDocs,
  deletePackage,
  deleteServiceTreeFunction
} from '@/architecture/presentation/context/api/service-tree'
import { isRootNode as isRootTreeNode } from '@/architecture/domain/utils/tree-utils'
import { markDocForAutoEdit } from '@/architecture/presentation/utils/docAutoEdit'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/architecture/domain/types'

interface CreateDocsForm {
  name: string
}

const createEmptyDocsForm = (): CreateDocsForm => ({
  name: ''
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
  const { t } = useI18n()

  const createDocsDialogVisible = ref(false)
  const creatingDocs = ref(false)
  const currentDocsParentNode = ref<ServiceTreeType | null>(null)
  const createDocsForm = ref<CreateDocsForm>(createEmptyDocsForm())

  const showBlockingLoading = (text: string) => {
    return ElLoading.service({
      lock: true,
      text,
      background: 'rgba(15, 23, 42, 0.36)'
    })
  }

  const handleCreateDocs = (parentNode?: ServiceTreeType) => {
    if (!currentApp.value) {
      ElMessage.warning(t('serviceTree.selectAppFirst'))
      return
    }

    currentDocsParentNode.value = parentNode || null
    createDocsForm.value = createEmptyDocsForm()
    createDocsDialogVisible.value = true
  }

  const handleSubmitCreateDocs = async () => {
    if (!currentApp.value) {
      ElMessage.warning(t('serviceTree.selectAppFirst'))
      return
    }

    const title = createDocsForm.value.name.trim() || t('workspaceNodeActions.untitledDoc')
    const code = buildUniqueDocsCode(createDocsCodeBase(title), currentDocsParentNode.value)

    creatingDocs.value = true
    try {
      const parentFullCodePath = currentDocsParentNode.value?.full_code_path || ''
      const response = await createDocs({
        user: currentApp.value.user,
        app: currentApp.value.code,
        name: title,
        code,
        parent_full_code_path: parentFullCodePath,
        content: createInitialDocContent(title),
        format: 'markdown'
      })

      createDocsDialogVisible.value = false
      if (response && response.id) {
        markDocForAutoEdit(response.full_code_path)
        ElMessage.success(t('workspaceNodeActions.docCreated'))
        await afterCreateNode(response)
      } else {
        ElMessage.warning(t('workspaceNodeActions.docCreatedMissingNode'))
      }
    } catch (error: any) {
      ElMessage.error(t('workspaceNodeActions.createDocFailed', { error: error.message || t('workspaceNodeActions.unknownError') }))
    } finally {
      creatingDocs.value = false
    }
  }

  const handleCloseCreateDocsDialog = () => {
    createDocsForm.value = createEmptyDocsForm()
    currentDocsParentNode.value = null
  }

  const handleDeleteDoc = async (node: ServiceTreeType) => {
    if (node.type !== 'docs') {
      ElMessage.warning(t('workspaceNodeActions.deleteDocOnly'))
      return
    }

    let loadingInstance: ReturnType<typeof showBlockingLoading> | null = null
    try {
      await ElMessageBox.confirm(
        t('workspaceNodeActions.deleteDocConfirm', { name: node.name }),
        t('workspaceNodeActions.deleteConfirmTitle'),
        {
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel'),
          type: 'warning'
        }
      )

      loadingInstance = showBlockingLoading(t('workspaceNodeActions.deletingDoc'))
      await deleteDocs(node.id)
      ElMessage.success(t('workspaceNodeActions.docDeleted'))
      await handleRefreshTree()

      if (currentFunction.value && currentFunction.value.id === node.id) {
        const parentPath = node.full_code_path?.split('/').slice(0, -1).join('/') || ''
        if (parentPath) {
          router.push(`/workspace${parentPath}`)
        }
      }
    } catch (error: any) {
      if (error !== 'cancel' && error !== 'close') {
        ElMessage.error(t('workspaceNodeActions.deleteDocFailed', { error: error.message || t('workspaceNodeActions.unknownError') }))
      }
    } finally {
      loadingInstance?.close()
    }
  }

  const handleDocDeleted = async () => {
    const loadingInstance = showBlockingLoading(t('workspaceNodeActions.refreshingTree'))
    try {
      await handleRefreshTree()
    } finally {
      loadingInstance.close()
    }

    if (currentFunction.value && currentFunction.value.type === 'docs') {
      const parentPath = currentFunction.value.full_code_path?.split('/').slice(0, -1).join('/') || ''
      if (parentPath) {
        router.push(`/workspace${parentPath}`)
      }
    }
  }

  const handleDeleteDirectory = async (node: ServiceTreeType) => {
    if (node.type !== 'package') {
      ElMessage.warning(t('workspaceNodeActions.deleteDirectoryOnly'))
      return
    }
    if (isRootTreeNode(node as ServiceTreeType)) {
      ElMessage.warning(t('workspaceNodeActions.deleteRootDirectoryDenied'))
      return
    }

    let loadingInstance: ReturnType<typeof showBlockingLoading> | null = null
    try {
      await ElMessageBox.confirm(
        t('workspaceNodeActions.deleteDirectoryConfirm', { name: node.name }),
        t('workspaceNodeActions.deleteConfirmTitle'),
        {
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel'),
          type: 'warning'
        }
      )

      loadingInstance = showBlockingLoading(t('workspaceNodeActions.deletingDirectory'))
      await deletePackage(node.id)
      ElMessage.success(t('workspaceNodeActions.directoryDeleted'))

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
        const errorMessage = error?.response?.data?.msg || error?.message || t('workspaceNodeActions.deleteDirectoryFailedDefault')
        ElMessage.error(errorMessage)
      }
    } finally {
      loadingInstance?.close()
    }
  }

  const handleDeleteFunction = async (node: ServiceTreeType) => {
    if (node.type !== 'function') {
      ElMessage.warning(t('workspaceNodeActions.deleteFunctionOnly'))
      return
    }

    let loadingInstance: ReturnType<typeof showBlockingLoading> | null = null
    try {
      await ElMessageBox.confirm(
        t('workspaceNodeActions.deleteFunctionConfirm', { name: node.name }),
        t('workspaceNodeActions.deleteConfirmTitle'),
        {
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel'),
          type: 'warning'
        }
      )

      loadingInstance = showBlockingLoading(t('workspaceNodeActions.deletingFunction'))
      await deleteServiceTreeFunction(node.id)
      ElMessage.success(t('workspaceNodeActions.deleteSuccess'))

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
        const errorMessage = error?.response?.data?.msg || error?.message || t('workspaceNodeActions.deleteFailedDefault')
        ElMessage.error(errorMessage)
      }
    } finally {
      loadingInstance?.close()
    }
  }

  const handleBulkDeleteNodes = async (nodes: ServiceTreeType[]) => {
    const deleteNodes = compactBulkDeleteNodes(nodes).filter((node) => {
      return node.type === 'package' || node.type === 'function' || node.type === 'docs'
    })
    if (deleteNodes.length === 0) {
      ElMessage.warning(t('serviceTree.deleteSelectableWarning'))
      return
    }

    const names = deleteNodes.slice(0, 5).map((node) => node.name || node.code || node.full_code_path).join('、')
    const suffix = deleteNodes.length > 5 ? t('workspaceNodeActions.bulkDeleteOverflow', { count: deleteNodes.length }) : ''
    let loadingInstance: ReturnType<typeof showBlockingLoading> | null = null
    try {
      await ElMessageBox.confirm(
        t('workspaceNodeActions.bulkDeleteConfirm', { count: deleteNodes.length, names, suffix }),
        t('workspaceNodeActions.bulkDeleteTitle'),
        {
          confirmButtonText: t('common.delete'),
          cancelButtonText: t('common.cancel'),
          type: 'warning'
        }
      )

      loadingInstance = showBlockingLoading(t('workspaceNodeActions.deletingNodes', { count: deleteNodes.length }))
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
      ElMessage.success(t('workspaceNodeActions.nodesDeleted', { count: deletedIds.size }))
    } catch (error: any) {
      if (error !== 'cancel' && error !== 'close') {
        const errorMessage = error?.response?.data?.msg || error?.message || t('workspaceNodeActions.bulkDeleteFailed')
        ElMessage.error(errorMessage)
      }
    } finally {
      loadingInstance?.close()
    }
  }

  return {
    createDocsDialogVisible,
    creatingDocs,
    currentDocsParentNode,
    createDocsForm,
    handleCreateDocs,
    handleSubmitCreateDocs,
    handleCloseCreateDocsDialog,
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

function createDocsCodeBase(title: string): string {
  const normalized = title
    .normalize('NFKD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/_+/g, '_')
    .replace(/^_+|_+$/g, '')

  let base = normalized || `doc_${Date.now().toString(36)}`
  if (!/^[a-z]/.test(base)) {
    base = `doc_${base}`
  }
  if (RESERVED_DOC_CODES.has(base)) {
    base = `doc_${base}`
  }

  return trimDocsCodeBase(base)
}

function buildUniqueDocsCode(base: string, parentNode: ServiceTreeType | null): string {
  const existingCodes = new Set(
    (parentNode?.children || [])
      .map((child) => (child.code || '').toLowerCase())
      .filter(Boolean)
  )

  let candidate = `${base}.docs`
  let index = 2
  while (existingCodes.has(candidate.toLowerCase())) {
    const suffix = `_${index}`
    candidate = `${trimDocsCodeBase(base, suffix.length)}${suffix}.docs`
    index += 1
  }

  return candidate
}

function trimDocsCodeBase(base: string, reservedSuffixLength = 0): string {
  const maxLength = Math.max(1, 45 - reservedSuffixLength)
  const trimmed = base.slice(0, maxLength).replace(/_+$/g, '')
  return trimmed || `doc_${Date.now().toString(36)}`
}

function createInitialDocContent(title: string): string {
  return `# ${title}\n\n`
}

const RESERVED_DOC_CODES = new Set([
  'break',
  'case',
  'chan',
  'const',
  'continue',
  'default',
  'defer',
  'else',
  'fallthrough',
  'for',
  'func',
  'go',
  'goto',
  'if',
  'import',
  'interface',
  'map',
  'package',
  'range',
  'return',
  'select',
  'struct',
  'switch',
  'type',
  'var'
])
