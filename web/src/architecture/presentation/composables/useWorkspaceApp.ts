/**
 * useWorkspaceApp - 工作空间管理 Composable
 * 
 * 职责：
 * - 工作空间列表加载
 * - 工作空间切换
 * - 工作空间 CRUD 操作
 */

import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElNotification, ElMessageBox } from 'element-plus'
import { apiClient } from '../../infrastructure/apiClient'
import { serviceFactory } from '../../infrastructure/factories'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import type { App } from '../../domain/services/WorkspaceDomainService'
import type { App as AppType, CreateAppRequest } from '@/types'
import { getAppDetailByUserAndCode, getAppWithServiceTree } from '@/api/app'

export function useWorkspaceApp(
  serviceProvider: IServiceProvider = serviceFactory  // 🔥 通过参数注入，提高可测试性
) {
  const route = useRoute()
  const router = useRouter()
  const applicationService = serviceProvider.getWorkspaceApplicationService()

  // 工作空间列表状态
  const appList = ref<AppType[]>([])
  const loadingApps = ref(false)
  const pendingAppId = ref<number | string | null>(null)

  // 创建工作空间对话框状态
  const createAppDialogVisible = ref(false)
  const creatingApp = ref(false)
  const createAppForm = ref<CreateAppRequest>({
    code: '',
    name: ''
  })

  // 加载工作空间列表
  const loadAppList = async (): Promise<void> => {
    try {
      loadingApps.value = true
      const response = await apiClient.get<any>('/workspace/api/v1/app/list', {
        page_size: 200,
        page: 1
      })
      
      // API 返回的是分页对象 { page, page_size, total_count, items: App[] }
      // 需要提取 items 数组
      if (response && typeof response === 'object') {
        if (Array.isArray(response)) {
          appList.value = response
        } else if ('items' in response && Array.isArray(response.items)) {
          appList.value = response.items
        } else {
          appList.value = []
        }
      } else {
        appList.value = []
      }
    } catch (error) {
      ElNotification.error({
        title: '错误',
        message: '加载工作空间列表失败'
      })
      appList.value = []
    } finally {
      loadingApps.value = false
    }
  }

  // 切换工作空间
  const handleSwitchApp = async (app: AppType, currentApp: () => AppType | null): Promise<void> => {
    // 检查 app 对象是否有效
    if (!app || !app.user || !app.code) {
      console.error('[useWorkspaceApp] handleSwitchApp: app 对象无效', app)
      return
    }
    
    const targetAppId = app.id
    
    // 检查当前应用是否已经是目标应用，避免重复切换
    const currentAppState = currentApp()
    if (currentAppState && String(currentAppState.id) === String(targetAppId)) {
      return
    }

    try {
      const appForService: App = {
        id: app.id,
        user: app.user,
        code: app.code,
        name: app.name
      }
      
      // 切换工作空间（这会触发服务目录树加载）
      await applicationService.triggerAppSwitch(appForService)
      
      // 🔥 阶段4：改为事件驱动，通过 RouteManager 统一处理路由更新
      // 更新路由
      const targetPath = `/workspace/${app.user}/${app.code}`
      if (route.path !== targetPath) {
        eventBus.emit(RouteEvent.updateRequested, {
          path: targetPath,
          query: {},
          replace: false,  // 工作空间切换使用 push，保留历史记录
          preserveParams: {},
          source: 'app-switch'
        })
      }
    } catch (error) {
      console.error('[useWorkspaceApp] handleSwitchApp 失败:', error)
      // 静默失败
    }
  }

  // 显示创建工作空间对话框
  const showCreateAppDialog = (): void => {
    resetCreateAppForm()
    createAppDialogVisible.value = true
  }

  // 重置创建工作空间表单
  const resetCreateAppForm = (): void => {
    createAppForm.value = {
      code: '',
      name: ''
    }
  }

  // 提交创建工作空间
  const submitCreateApp = async (currentApp: () => AppType | null): Promise<void> => {
    if (!createAppForm.value.name || !createAppForm.value.code) {
      ElNotification.warning({
        title: '提示',
        message: '请填写名称和英文标识'
      })
      return
    }

    try {
      creatingApp.value = true
      const createResponse = await apiClient.post<{ user: string; app: string; app_dir: string }>('/workspace/api/v1/app/create', createAppForm.value)
      ElNotification.success({
        title: '成功',
        message: '工作空间创建成功'
      })
      createAppDialogVisible.value = false
      
      // 使用创建响应中的信息获取工作空间详情和服务目录树（合并接口，减少请求次数）
      if (createResponse && createResponse.user && createResponse.app) {
        try {
          // ⭐ 使用合并接口获取工作空间详情和服务目录树
          // 传递 user 和 app，而不是只传 code
          const workspaceData = await getAppWithServiceTree(createResponse.user, createResponse.app)
          
          if (workspaceData && workspaceData.app && workspaceData.app.user && workspaceData.app.code) {
            const newApp = workspaceData.app
            
            // 将新应用添加到列表（如果不在列表中的话）
            const existsInList = appList.value.some(a => a.id === newApp.id)
            if (!existsInList) {
              appList.value.push(newApp)
            }
            
            // 使用获取到的完整 App 对象进行切换
            // 注意：这里我们已经有服务目录树了，但 handleSwitchApp 会再次加载
            // 为了优化，我们可以直接设置服务目录树，但为了保持一致性，还是使用 handleSwitchApp
        await handleSwitchApp(newApp, currentApp)
          } else {
            // 如果获取详情失败，使用创建响应中的信息直接跳转
            const targetPath = `/workspace/${createResponse.user}/${createResponse.app}`
            if (route.path !== targetPath) {
              eventBus.emit(RouteEvent.updateRequested, {
                path: targetPath,
                query: {},
                replace: false,
                preserveParams: {},
                source: 'app-create-fallback'
              })
            }
          }
        } catch (error) {
          // 如果获取详情失败，使用创建响应中的信息直接跳转
          console.error('[useWorkspaceApp] 获取工作空间数据失败:', error)
          const targetPath = `/workspace/${createResponse.user}/${createResponse.app}`
          if (route.path !== targetPath) {
            eventBus.emit(RouteEvent.updateRequested, {
              path: targetPath,
              query: {},
              replace: false,
              preserveParams: {},
              source: 'app-create-fallback'
            })
          }
        }
      }
    } catch (error: any) {
      // 🔥 统一使用 msg 字段
      const errorMessage = error?.response?.data?.msg || '创建工作空间失败'
      ElNotification.error({
        title: '错误',
        message: errorMessage
      })
    } finally {
      creatingApp.value = false
    }
  }

  // 更新工作空间（重新编译）
  const handleUpdateApp = async (app: AppType): Promise<void> => {
    try {
      await apiClient.post(`/workspace/api/v1/app/update/${app.code}`, {})
      ElNotification.success({
        title: '成功',
        message: '工作空间更新成功'
      })
    } catch (error: any) {
      // 🔥 统一使用 msg 字段
      const errorMessage = error?.response?.data?.msg || '更新工作空间失败'
      ElNotification.error({
        title: '错误',
        message: errorMessage
      })
    }
  }

  // 删除工作空间
  const handleDeleteApp = async (app: AppType, currentApp: () => AppType | null): Promise<void> => {
    try {
      await ElMessageBox.confirm(
        `确定要删除工作空间 "${app.name}" 吗？此操作不可恢复。`,
        '确认删除',
        {
          confirmButtonText: '删除',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
      
      await apiClient.delete(`/workspace/api/v1/app/delete/${app.code}`)
      ElNotification.success({
        title: '成功',
        message: '工作空间删除成功'
      })
      
      // 刷新工作空间列表
      await loadAppList()
      
      // 如果删除的是当前工作空间，切换到第一个工作空间或清空
      // 🔥 阶段4：改为事件驱动，通过 RouteManager 统一处理路由更新
      const currentAppState = currentApp()
      if (currentAppState && currentAppState.id === app.id) {
        if (appList.value.length > 0) {
          await handleSwitchApp(appList.value[0], currentApp)
        } else {
          eventBus.emit(RouteEvent.updateRequested, {
            path: '/workspace',
            query: {},
            replace: false,
            preserveParams: {},
            source: 'app-delete-empty'
          })
        }
      }
    } catch (error: any) {
      if (error !== 'cancel') {
        // 🔥 统一使用 msg 字段
        const errorMessage = error?.response?.data?.msg || '删除工作空间失败'
        ElNotification.error({
          title: '错误',
          message: errorMessage
        })
      }
    }
  }

  return {
    // 状态
    appList,
    loadingApps,
    pendingAppId,
    createAppDialogVisible,
    creatingApp,
    createAppForm,
    
    // 方法
    loadAppList,
    handleSwitchApp,
    showCreateAppDialog,
    resetCreateAppForm,
    submitCreateApp,
    handleUpdateApp,
    handleDeleteApp
  }
}

