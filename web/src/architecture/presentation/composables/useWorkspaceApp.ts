/**
 * useWorkspaceApp - 应用管理 Composable
 * 
 * 职责：
 * - 应用列表加载
 * - 应用切换
 * - 应用 CRUD 操作
 */

import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElNotification, ElMessageBox } from 'element-plus'
import { apiClient } from '../../infrastructure/apiClient'
import { serviceFactory } from '../../infrastructure/factories'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import type { App } from '../../domain/services/WorkspaceDomainService'
import type { App as AppType, CreateAppRequest } from '@/types'

export function useWorkspaceApp() {
  const route = useRoute()
  const router = useRouter()
  const applicationService = serviceFactory.getWorkspaceApplicationService()

  // 应用列表状态
  const appList = ref<AppType[]>([])
  const loadingApps = ref(false)
  const pendingAppId = ref<number | string | null>(null)

  // 创建应用对话框状态
  const createAppDialogVisible = ref(false)
  const creatingApp = ref(false)
  const createAppForm = ref<CreateAppRequest>({
    code: '',
    name: ''
  })

  // 加载应用列表
  const loadAppList = async (): Promise<void> => {
    try {
      loadingApps.value = true
      const response = await apiClient.get<any>('/api/v1/app/list', {
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
        message: '加载应用列表失败'
      })
      appList.value = []
    } finally {
      loadingApps.value = false
    }
  }

  // 切换应用
  const handleSwitchApp = async (app: AppType, currentApp: () => AppType | null): Promise<void> => {
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
      
      // 切换应用（这会触发服务树加载）
      await applicationService.triggerAppSwitch(appForService)
      
      // 🔥 阶段4：改为事件驱动，通过 RouteManager 统一处理路由更新
      // 更新路由
      const targetPath = `/workspace/${app.user}/${app.code}`
      if (route.path !== targetPath) {
        eventBus.emit(RouteEvent.updateRequested, {
          path: targetPath,
          query: {},
          replace: false,  // 应用切换使用 push，保留历史记录
          preserveParams: {},
          source: 'app-switch'
        })
      }
    } catch (error) {
      // 静默失败
    }
  }

  // 显示创建应用对话框
  const showCreateAppDialog = (): void => {
    resetCreateAppForm()
    createAppDialogVisible.value = true
  }

  // 重置创建应用表单
  const resetCreateAppForm = (): void => {
    createAppForm.value = {
      code: '',
      name: ''
    }
  }

  // 提交创建应用
  const submitCreateApp = async (currentApp: () => AppType | null): Promise<void> => {
    if (!createAppForm.value.name || !createAppForm.value.code) {
      ElNotification.warning({
        title: '提示',
        message: '请填写应用名称和应用代码'
      })
      return
    }

    try {
      creatingApp.value = true
      await apiClient.post('/api/v1/app/create', createAppForm.value)
      ElNotification.success({
        title: '成功',
        message: '应用创建成功'
      })
      createAppDialogVisible.value = false
      
      // 刷新应用列表
      await loadAppList()
      
      // 如果应用列表中有新创建的应用，自动切换
      const newApp = appList.value.find(
        (a: AppType) => a.code === createAppForm.value.code
      )
      if (newApp) {
        await handleSwitchApp(newApp, currentApp)
      }
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || '创建应用失败'
      ElNotification.error({
        title: '错误',
        message: errorMessage
      })
    } finally {
      creatingApp.value = false
    }
  }

  // 更新应用（重新编译）
  const handleUpdateApp = async (app: AppType): Promise<void> => {
    try {
      await apiClient.post(`/api/v1/app/update/${app.code}`, {})
      ElNotification.success({
        title: '成功',
        message: '应用更新成功'
      })
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || '更新应用失败'
      ElNotification.error({
        title: '错误',
        message: errorMessage
      })
    }
  }

  // 删除应用
  const handleDeleteApp = async (app: AppType, currentApp: () => AppType | null): Promise<void> => {
    try {
      await ElMessageBox.confirm(
        `确定要删除应用 "${app.name}" 吗？此操作不可恢复。`,
        '确认删除',
        {
          confirmButtonText: '删除',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )
      
      await apiClient.delete(`/api/v1/app/delete/${app.code}`)
      ElNotification.success({
        title: '成功',
        message: '应用删除成功'
      })
      
      // 刷新应用列表
      await loadAppList()
      
      // 如果删除的是当前应用，切换到第一个应用或清空
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
        const errorMessage = error?.response?.data?.message || '删除应用失败'
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

