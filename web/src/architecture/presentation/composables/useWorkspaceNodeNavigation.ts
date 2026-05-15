import type { RouteLocationNormalizedLoaded } from 'vue-router'
import { RouteSource, type RouteSourceType } from '@/architecture/shared/routing/routeSource'
import { resolveWorkspaceUrl } from '@/architecture/shared/routing/route'
import { isLinkNavigation as checkLinkNavigation, LINK_TYPE_QUERY_KEY } from '@/architecture/shared/routing/linkNavigation'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import type { ServiceTree } from '../../domain/types'
import type { ServiceTree as ServiceTreeType } from '@/architecture/domain/types'

interface UseWorkspaceNodeNavigationOptions {
  route: RouteLocationNormalizedLoaded
  currentFunction: () => ServiceTreeType | null
  triggerNodeClick: (node: ServiceTree) => void
}

export function useWorkspaceNodeNavigation(options: UseWorkspaceNodeNavigationOptions) {
  const buildWorkspacePath = (fullCodePath: string): string => {
    return resolveWorkspaceUrl(fullCodePath.startsWith('/') ? fullCodePath : `/${fullCodePath}`)
  }

  const isLinkNavigation = (): boolean => {
    return checkLinkNavigation(options.route.query as Record<string, any>)
  }

  const buildLinkNavigationQuery = (): Record<string, string | string[]> => {
    const preservedQuery: Record<string, string | string[]> = {}
    Object.keys(options.route.query).forEach(key => {
      if (key !== LINK_TYPE_QUERY_KEY) {
        const value = options.route.query[key]
        if (value !== null && value !== undefined) {
          preservedQuery[key] = Array.isArray(value)
            ? value.filter(v => v !== null).map(v => String(v))
            : String(value)
        }
      }
    })
    return preservedQuery
  }

  const handleFunctionNodeRoute = (node: ServiceTree, source: RouteSourceType): void => {
    if (!node.full_code_path) {
      return
    }

    const targetPath = buildWorkspacePath(node.full_code_path)
    if (options.route.path === targetPath) {
      options.triggerNodeClick(node)
      return
    }

    const isLink = isLinkNavigation()
    const preservedQuery: Record<string, string | string[]> = isLink
      ? buildLinkNavigationQuery()
      : {}

    eventBus.emit(RouteEvent.updateRequested, {
      path: targetPath,
      query: preservedQuery,
      replace: true,
      preserveParams: {
        table: false,
        search: false,
        state: false,
        linkNavigation: isLink
      },
      source
    })
  }

  const handlePackageNodeRoute = (node: ServiceTree, source: RouteSourceType, customQuery?: Record<string, any>): void => {
    if (!node.full_code_path) return

    const targetPath = buildWorkspacePath(node.full_code_path)
    if (options.route.path === targetPath && !customQuery) {
      return
    }

    eventBus.emit(RouteEvent.updateRequested, {
      path: targetPath,
      query: customQuery || {},
      replace: true,
      preserveParams: {
        table: false,
        search: false,
        state: false,
        linkNavigation: false
      },
      source
    })
  }

  const handleNodeClick = (node: ServiceTreeType) => {
    const serviceTree = node as unknown as ServiceTree

    if (serviceTree.type === 'function') {
      handleFunctionNodeRoute(serviceTree, RouteSource.WORKSPACE_NODE_CLICK)
      return
    }

    if (serviceTree.type === 'package') {
      const targetPath = buildWorkspacePath(serviceTree.full_code_path || '')
      options.triggerNodeClick(serviceTree)
      if (options.route.path !== targetPath) {
        handlePackageNodeRoute(serviceTree, RouteSource.WORKSPACE_NODE_CLICK_PACKAGE)
      }
      return
    }

    if (serviceTree.type === 'docs') {
      const targetPath = buildWorkspacePath(serviceTree.full_code_path || '')
      options.triggerNodeClick(serviceTree)
      if (options.route.path !== targetPath) {
        handlePackageNodeRoute(serviceTree, RouteSource.WORKSPACE_NODE_CLICK_DOCS)
      }
      return
    }

    if (serviceTree.type === 'board') {
      const targetPath = buildWorkspacePath(serviceTree.full_code_path || '')
      options.triggerNodeClick(serviceTree)
      if (options.route.path !== targetPath) {
        handlePackageNodeRoute(serviceTree, RouteSource.WORKSPACE_NODE_CLICK_BOARD)
      }
      return
    }

    options.triggerNodeClick(serviceTree)
  }

  const handleBreadcrumbNodeClick = (node: ServiceTree) => {
    if (node.type === 'function') {
      handleFunctionNodeRoute(node, RouteSource.WORKSPACE_NODE_CLICK)
      return
    }
    if (node.type === 'package') {
      handlePackageNodeRoute(node, RouteSource.WORKSPACE_NODE_CLICK_PACKAGE)
      return
    }
    if (node.type === 'docs') {
      handlePackageNodeRoute(node, RouteSource.BREADCRUMB_NODE_CLICK_DOCS)
      return
    }
    if (node.type === 'board') {
      handlePackageNodeRoute(node, RouteSource.BREADCRUMB_NODE_CLICK_BOARD)
      return
    }
    options.triggerNodeClick(node)
  }

  const backToList = () => {
    const currentFunction = options.currentFunction()
    if (!currentFunction) return

    const query: Record<string, string | string[]> = {}
    Object.keys(options.route.query).forEach(key => {
      if (key !== '_tab' && key !== '_id') {
        const value = options.route.query[key]
        if (value !== null && value !== undefined) {
          query[key] = Array.isArray(value)
            ? value.filter(v => v !== null).map(v => String(v))
            : String(value)
        }
      }
    })

    const path = currentFunction.full_code_path
      ? buildWorkspacePath(currentFunction.full_code_path)
      : ''

    eventBus.emit(RouteEvent.updateRequested, {
      path,
      query,
      replace: false,
      preserveParams: {
        state: true
      },
      source: RouteSource.BACK_TO_LIST
    })
  }

  return {
    buildWorkspacePath,
    handleNodeClick,
    handleBreadcrumbNodeClick,
    backToList,
  }
}
