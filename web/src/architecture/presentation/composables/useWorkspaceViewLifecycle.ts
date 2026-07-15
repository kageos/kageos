import { nextTick, onMounted, onUnmounted, watch, type Ref } from 'vue'
import type { RouteLocationNormalizedLoaded, Router } from 'vue-router'
import { RouteManager } from '../router/routeManager'
import { eventBus, WorkspaceEvent } from '../../infrastructure/eventBus'
import { extractWorkspacePath } from '@/architecture/shared/routing/route'
import type { FunctionDetail } from '@/architecture/domain/types'
import type { ServiceTree as ServiceTreeType, App as AppType } from '@/architecture/domain/types'

interface UseWorkspaceViewLifecycleOptions {
  route: RouteLocationNormalizedLoaded
  router: Router
  currentFunction: () => ServiceTreeType | null
  currentFunctionDetail: () => FunctionDetail | null
  setCurrentFunctionDetail: (detail: FunctionDetail | null) => void
  expandedKeys: Ref<number[]>
  currentApp: () => AppType | null
  serviceTree: () => ServiceTreeType[]
  loadAppFromRoute: () => Promise<void>
  setupRouteWatch: () => void
  expandCurrentRoutePath: () => void
  queryTab: () => string
  loadNodeDetail: (node: ServiceTreeType) => Promise<void> | void
  updateAppInfo: (app: AppType) => void
  findNodeByPath: (tree: ServiceTreeType[], path: string) => ServiceTreeType | null
  openWorkspaceListDialog: () => void
}

export function resolveWorkspaceRootNodeForRoute(
  routePath: string,
  tree: ServiceTreeType[],
  findNodeByPath: (tree: ServiceTreeType[], path: string) => ServiceTreeType | null
): ServiceTreeType | null {
  const fullPath = extractWorkspacePath(routePath)
  const pathSegments = fullPath.split('/').filter(Boolean)
  if (pathSegments.length !== 2 || !tree || tree.length === 0) {
    return null
  }

  const rootPath = '/' + pathSegments.join('/')
  const exactRootNode = findNodeByPath(tree, rootPath)
  if (exactRootNode?.type === 'package') {
    return exactRootNode
  }

  return tree.find((node) => node.type === 'package') || null
}

export function useWorkspaceViewLifecycle(options: UseWorkspaceViewLifecycleOptions) {
  let unsubscribeFunctionLoaded: (() => void) | null = null
  let unsubscribeServiceTreeLoaded: (() => void) | null = null
  let unsubscribeAppInfoUpdated: (() => void) | null = null
  let routeManager: RouteManager | null = null

  const ensureFunctionDetailLoaded = async () => {
    const currentFunction = options.currentFunction()
    if (!currentFunction || currentFunction.type !== 'function') {
      return
    }
    await options.loadNodeDetail(currentFunction)
  }

  const selectWorkspaceRootIfEmpty = async (tree: ServiceTreeType[] = options.serviceTree()) => {
    if (options.currentFunction()) {
      return
    }

    const rootNode = resolveWorkspaceRootNodeForRoute(
      options.route.path,
      tree,
      options.findNodeByPath
    )
    if (!rootNode) {
      return
    }

    await options.loadNodeDetail(rootNode)
  }

  onMounted(async () => {
    if (routeManager) {
      routeManager.destroy()
      routeManager = null
    }

    routeManager = new RouteManager(options.router, options.route, eventBus)

    unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, (payload: { node: any, detail: FunctionDetail }) => {
      const currentFunction = options.currentFunction()
      if (currentFunction && (
        currentFunction.id === payload.node.id ||
        currentFunction.full_code_path === payload.node.full_code_path
      )) {
        options.setCurrentFunctionDetail(payload.detail)
      }
    })

    unsubscribeServiceTreeLoaded = eventBus.on(WorkspaceEvent.serviceTreeLoaded, async (payload: { tree: ServiceTreeType[], expandedKeys?: number[] }) => {
      options.expandedKeys.value = payload.expandedKeys && payload.expandedKeys.length > 0
        ? [...payload.expandedKeys]
        : []

      await nextTick()
      await selectWorkspaceRootIfEmpty(payload.tree)
    })

    unsubscribeAppInfoUpdated = eventBus.on(WorkspaceEvent.appInfoUpdated, (payload: { app: AppType }) => {
      options.updateAppInfo(payload.app)
    })

    await options.loadAppFromRoute()
    options.setupRouteWatch()

    if (options.route.name === 'workspace-user') {
      nextTick(() => options.openWorkspaceListDialog())
    }
  })

  watch(() => options.serviceTree().length, (newLength: number) => {
    if (newLength > 0 && options.currentApp()) {
      options.expandCurrentRoutePath()
    }
  }, { immediate: true })

  watch(
    () => ({
      routePath: options.route.path,
      tree: options.serviceTree(),
      currentPath: options.currentFunction()?.full_code_path || '',
      appUser: options.currentApp()?.user || '',
      appCode: options.currentApp()?.code || ''
    }),
    () => {
      void selectWorkspaceRootIfEmpty()
    },
    { immediate: true }
  )

  watch(() => options.currentFunction()?.id, (newId: number | undefined, oldId: number | undefined) => {
    if (newId !== oldId && oldId !== undefined) {
      options.setCurrentFunctionDetail(null)
    }
  })

  watch(() => options.queryTab(), async (newTab: string) => {
    if (newTab === 'create' || newTab === 'edit' || newTab === 'detail') {
      const currentFunction = options.currentFunction()
      if (!currentFunction) {
        return
      }

      if (!options.currentFunctionDetail() || newTab === 'detail') {
        await ensureFunctionDetailLoaded()
      }
    }
  }, { immediate: false })

  watch(() => options.route.query._tab, async (newTab: any) => {
    if (newTab === 'create' || newTab === 'edit' || newTab === 'detail') {
      await ensureFunctionDetailLoaded()
    }
  }, { immediate: false })

  onUnmounted(() => {
    options.setCurrentFunctionDetail(null)

    if (unsubscribeFunctionLoaded) {
      unsubscribeFunctionLoaded()
    }
    if (unsubscribeServiceTreeLoaded) {
      unsubscribeServiceTreeLoaded()
    }
    if (unsubscribeAppInfoUpdated) {
      unsubscribeAppInfoUpdated()
    }
    if (routeManager) {
      routeManager.destroy()
      routeManager = null
    }
  })
}
