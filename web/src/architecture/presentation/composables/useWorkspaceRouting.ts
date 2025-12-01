/**
 * useWorkspaceRouting - 路由管理 Composable
 * 
 * 职责：
 * - 路由同步到 Tab
 * - 从路由恢复 Tab
 * - 路由变化处理
 */

import { watch, ref, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { extractWorkspacePath } from '@/utils/route'
import { preserveQueryParamsForTable, preserveQueryParamsForForm } from '@/utils/queryParams'
import { serviceFactory } from '../../infrastructure/factories'
import type { ServiceTree, App } from '../../domain/services/WorkspaceDomainService'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/types'

export function useWorkspaceRouting(options: {
  tabs: () => any[]
  activeTabId: () => string
  serviceTree: () => ServiceTreeType[]
  currentApp: () => AppType | null
  appList: () => AppType[]
  loadAppList: () => Promise<void>
  findNodeByPath: (tree: ServiceTreeType[], path: string) => ServiceTreeType | null
  checkAndExpandForkedPaths: () => void
  expandCurrentRoutePath: () => void
}) {
  const route = useRoute()
  const router = useRouter()
  const stateManager = serviceFactory.getWorkspaceStateManager()
  const applicationService = serviceFactory.getWorkspaceApplicationService()

  // 防重复调用保护
  let isLoadingAppFromRoute = false
  let isSyncingRouteToTab = false

  // 从路由同步到 Tab 状态（路由变化时调用）
  const syncRouteToTab = async () => {
    const fullPath = extractWorkspacePath(route.path)
    
    if (!fullPath) {
      // 空路径，不处理
      return
    }
    
    // 解析路径，找到对应的 Tab
    const targetTab = options.tabs().find(t => {
      const tabPath = t.path?.replace(/^\//, '') || ''
      const routePath = fullPath?.replace(/^\//, '') || ''
      return tabPath === routePath
    })
    
    if (targetTab) {
      // Tab 已存在，激活它（不触发路由更新）
      if (options.activeTabId() !== targetTab.id) {
        isSyncingRouteToTab = true
        applicationService.activateTab(targetTab.id)
        isSyncingRouteToTab = false
      }
      
      // 检查函数详情是否已加载（刷新后切换 Tab 时可能需要加载）
      if (targetTab.node && targetTab.node.type === 'function') {
        const detail = stateManager.getFunctionDetail(targetTab.node)
        if (!detail) {
          // 使用 handleNodeClick 加载函数详情
          applicationService.handleNodeClick(targetTab.node)
        }
      }
    } else {
      // Tab 不存在，从路由打开新 Tab
      // 注意：这里需要确保服务树已加载，否则无法找到节点
      if (options.serviceTree().length > 0) {
        await loadAppFromRoute()
      }
    }
  }

  // 从路由解析应用并加载（主要用于刷新时）
  const loadAppFromRoute = async () => {
    // 防止重复调用
    if (isLoadingAppFromRoute) {
      return
    }
    
    // 提取路径
    const fullPath = extractWorkspacePath(route.path)
    
    if (!fullPath) {
      return
    }

    const pathSegments = fullPath.split('/').filter(Boolean)
    if (pathSegments.length < 2) {
      return
    }

    const [user, appCode] = pathSegments
    
    try {
      isLoadingAppFromRoute = true
      
      // 确保应用列表已加载
      if (options.appList().length === 0) {
        await options.loadAppList()
      }
      
      // 从已加载的应用列表中查找
      const app = options.appList().find((a: AppType) => a.user === user && a.code === appCode)
      
      if (!app) {
        return
      }
      
      const targetAppId = app.id
      let appSwitched = false
      const pendingAppId = ref<number | string | null>(null)

      // 检查当前应用是否已经是目标应用
      const currentAppState = options.currentApp()
      if (!currentAppState || String(currentAppState.id) !== String(targetAppId)) {
        // 需要切换应用
        if (String(pendingAppId.value) !== String(targetAppId)) {
          pendingAppId.value = targetAppId
          try {
            const appForService: App = {
              id: app.id,
              user: app.user,
              code: app.code,
              name: app.name
            }
            await applicationService.triggerAppSwitch(appForService)
            appSwitched = true
          } catch (error) {
            // 静默失败
            pendingAppId.value = null
            return
          }
        }
      }

      // 处理子路径（打开 Tab）
      if (pathSegments.length > 2) {
        const functionPath = '/' + pathSegments.join('/') // 构造完整路径，如 /luobei/demo/crm/list
        
        // 检查是否有 _tab 参数（create/edit/detail 模式）
        const tabParam = route.query._tab as string
        if (tabParam === 'create' || tabParam === 'edit' || tabParam === 'detail') {
          // create/edit 模式不需要打开 Tab，直接加载函数详情
          const tryLoadFunction = () => {
            const tree = options.serviceTree()
            if (tree && tree.length > 0) {
              const node = options.findNodeByPath(tree, functionPath)
              if (node) {
                const serviceNode: ServiceTree = node as any
                // 设置当前函数，但不打开 Tab
                applicationService.handleNodeClick(serviceNode)
              }
            }
          }
          
          if (appSwitched) {
            let retries = 0
            const interval = setInterval(() => {
              if (options.serviceTree().length > 0 || retries > 10) {
                clearInterval(interval)
                tryLoadFunction()
              }
              retries++
            }, 200)
          } else {
            tryLoadFunction()
          }
          
          // 检查 _forked 参数，自动展开路径
          if (route.query._forked) {
            nextTick(() => {
              options.checkAndExpandForkedPaths()
            })
          }
          
          return // create/edit 模式不打开 Tab
        }
        
        // 检查 _forked 参数，自动展开路径
        if (route.query._forked) {
          nextTick(() => {
            options.checkAndExpandForkedPaths()
          })
        }
        
        // 尝试查找节点并打开/激活 Tab
        // 使用早期返回优化条件判断
        const tryOpenTab = () => {
          const tree = options.serviceTree()
          
          // 早期返回：服务树为空
          if (!tree || tree.length === 0) {
            return
          }
          
          const node = options.findNodeByPath(tree, functionPath)
          
          // 早期返回：节点不存在
          if (!node) {
            return
          }
          
          const serviceNode: ServiceTree = node as any
          
          // 检查 Tab 是否存在
          const tabsArray = Array.isArray(options.tabs()) ? options.tabs() : []
          const existingTab = tabsArray.find(t => 
            t.path === serviceNode.full_code_path || t.path === String(serviceNode.id)
          )
          
          if (existingTab) {
            // Tab 已存在，激活它（不触发路由更新）
            if (options.activeTabId() !== existingTab.id) {
              isSyncingRouteToTab = true
              applicationService.activateTab(existingTab.id)
              isSyncingRouteToTab = false
            }
            
            // 无论是否激活，都检查函数详情是否已加载
            if (existingTab.node && existingTab.node.type === 'function') {
              const detail = stateManager.getFunctionDetail(existingTab.node)
              if (!detail) {
                applicationService.handleNodeClick(existingTab.node)
              }
            }
            return
          }
          
          // Tab 不存在，打开新 Tab
          // 🔥 注意：triggerNodeClick 会触发 handleNodeClick，handleNodeClick 会根据函数类型
          // 决定是否保留 table 参数。如果目标函数是 form 类型，会使用 preserveQueryParamsForForm
          // 清除 page、page_size、sorts 等参数
          applicationService.triggerNodeClick(serviceNode)
        }

        // 等待服务树加载
        if (appSwitched) {
          let retries = 0
          const interval = setInterval(() => {
            if (options.serviceTree().length > 0 || retries > 10) {
              clearInterval(interval)
              tryOpenTab()
            }
            retries++
          }, 200)
        } else {
          tryOpenTab()
        }
        
        // 展开目录树
        if (route.query._forked) {
          nextTick(() => {
            options.checkAndExpandForkedPaths()
          })
        } else {
          options.expandCurrentRoutePath()
        }
      }
    } catch (error) {
      // 静默失败
    } finally {
      isLoadingAppFromRoute = false
    }
  }

  // 设置路由监听
  const setupRouteWatch = () => {
    let routeWatchTimer: ReturnType<typeof setTimeout> | null = null
    // 🔥 同时监听 path 和 query，确保 Tab 切换时即使路径相同也能触发更新
    watch(() => [route.path, route.query], async () => {
      // 防抖：避免频繁调用
      if (routeWatchTimer) {
        clearTimeout(routeWatchTimer)
      }
      routeWatchTimer = setTimeout(() => {
        syncRouteToTab()
      }, 50) // 50ms 防抖，足够快但避免频繁调用
    }, { immediate: false })
  }

  return {
    syncRouteToTab,
    loadAppFromRoute,
    setupRouteWatch,
    isSyncingRouteToTab: () => isSyncingRouteToTab
  }
}

