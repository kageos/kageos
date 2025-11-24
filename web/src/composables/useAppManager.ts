/**
 * useAppManager - 应用管理 Composable
 * 负责应用列表加载、切换、CRUD 操作
 */

import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAppList, createApp, updateApp, deleteApp } from '@/api'
import type { App, CreateAppRequest } from '@/types'

export function useAppManager() {
  const router = useRouter()

  // 状态
  const currentApp = ref<App | null>(null)
  const appList = ref<App[]>([])
  const loading = ref(false)

  /**
   * 加载应用列表
   */
  const loadAppList = async (): Promise<App[]> => {
    try {
      loading.value = true
      const items = await getAppList()
      appList.value = items
      return items
    } catch (error) {
      Logger.error('useAppManager', '获取应用列表失败', error)
      ElMessage.error('获取应用列表失败')
      return []
    } finally {
      loading.value = false
    }
  }

  /**
   * 从路由解析应用
   */
  const parseAppFromRoute = (): App | null => {
    const fullPath = window.location.pathname.replace('/workspace/', '').replace(/^\/+|\/+$/g, '')
    if (!fullPath) {
      return null
    }

    const pathSegments = fullPath.split('/').filter(Boolean)
    if (pathSegments.length < 2) {
      return null
    }

    const [user, appCode] = pathSegments
    const app = appList.value.find((a: App) => a.user === user && a.code === appCode)
    
    return app || null
  }

  /**
   * 切换应用
   */
  const switchApp = async (app: App, updateRoute = true): Promise<void> => {
    currentApp.value = app

    if (updateRoute) {
      await router.push(`/workspace/${app.user}/${app.code}`)
    }
  }

  /**
   * 创建应用
   */
  const handleCreateApp = async (form: CreateAppRequest): Promise<App | null> => {
    try {
      const resp = await createApp(form)
      ElMessage.success('应用创建成功')
      
      // 刷新应用列表
      await loadAppList()
      
      // 从刷新后的应用列表中查找新创建的应用
      // 后端返回的 resp 结构是 { user, app, app_dir }，其中 app 对应前端的 code
      const createdApp = appList.value.find(
        (a: App) => a.user === resp.user && a.code === resp.app
      )
      
      if (!createdApp) {
        Logger.error('useAppManager', '创建应用后未在列表中找到新应用', resp)
        return null
      }
      
      return createdApp
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || '创建应用失败'
      ElMessage.error(errorMessage)
      return null
    }
  }

  /**
   * 更新应用
   */
  const handleUpdateApp = async (app: App): Promise<boolean> => {
    try {
      await updateApp(app.code)
      ElMessage.success('应用更新成功')
      
      // 刷新应用列表
      await loadAppList()
      
      return true
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || '更新应用失败'
      ElMessage.error(errorMessage)
      return false
    }
  }

  /**
   * 删除应用
   */
  const handleDeleteApp = async (app: App): Promise<boolean> => {
    try {
      await ElMessageBox.confirm(
        `确定要删除应用「${app.name || app.code}」吗？此操作不可恢复。`,
        '删除应用',
        {
          confirmButtonText: '确定删除',
          cancelButtonText: '取消',
          type: 'warning',
          confirmButtonClass: 'el-button--danger'
        }
      )

      await deleteApp(app.code)
      ElMessage.success('应用删除成功')
      
      // 刷新应用列表
      await loadAppList()
      
      return true
    } catch (error: any) {
      // 用户取消删除
      if (error === 'cancel') {
        return false
      }
      
      const errorMessage = error?.response?.data?.message || '删除应用失败'
      ElMessage.error(errorMessage)
      return false
    }
  }

  return {
    // 状态
    currentApp,
    appList,
    loading,
    
    // 方法
    loadAppList,
    parseAppFromRoute,
    switchApp,
    handleCreateApp,
    handleUpdateApp,
    handleDeleteApp
  }
}

