/**
 * useWorkspaceDetail - 详情管理 Composable
 * 
 * 职责：
 * - 详情抽屉打开/关闭
 * - 详情导航（上一条/下一条）
 * - 详情编辑提交
 */

import { ref, computed, watch, nextTick } from 'vue'
import { deepClone } from '@/utils/clone'
import { useRoute, useRouter } from 'vue-router'
import { ElNotification, ElMessage } from 'element-plus'
import { serviceFactory } from '../../infrastructure/factories'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import FormRenderer from '@/core/renderers-v2/FormRenderer.vue'
import type { FieldConfig, FieldValue } from '../../domain/types'
import type { FunctionDetail } from '../../domain/interfaces/IFunctionLoader'

export function useWorkspaceDetail(options: {
  currentFunctionDetail: () => FunctionDetail | null
  currentFunction: () => any
}) {
  const route = useRoute()
  const router = useRouter()
  const tableApplicationService = serviceFactory.getTableApplicationService()
  const tableStateManager = serviceFactory.getTableStateManager()
  const stateManager = serviceFactory.getWorkspaceStateManager()

  // 详情抽屉状态
  const detailDrawerVisible = ref(false)
  const detailDrawerTitle = ref('详情')
  const detailRowData = ref<Record<string, any> | null>(null)
  const detailFields = ref<FieldConfig[]>([])
  const detailOriginalRow = ref<Record<string, any> | null>(null)
  const detailDrawerMode = ref<'read' | 'edit'>('read')
  const drawerSubmitting = ref(false)
  const detailFormRendererRef = ref<InstanceType<typeof FormRenderer> | null>(null)
  const detailUserInfoMap = ref<Map<string, any>>(new Map())
  const detailTableData = ref<any[]>([])
  const currentDetailIndex = ref<number>(-1)

  // 编辑模式的函数详情（从 response 字段中筛选可编辑的字段）
  const editFunctionDetail = computed<FunctionDetail | null>(() => {
    const current = options.currentFunctionDetail()
    if (!current) return null
    
    // 如果是 table 类型，从 response 字段中筛选可编辑的字段
    if (current.template_type === TEMPLATE_TYPE.TABLE) {
      const fields = (current.response || []) as FieldConfig[]
      const editableFields = fields.filter(field => {
        const permission = field.table_permission
        return !permission || permission === '' || permission === 'update'
      })
      return {
        ...current,
        template_type: 'form',
        request: editableFields,
        response: []
      }
    }
    
    // 如果是 form 类型，直接使用 request 字段
    if (current.template_type === TEMPLATE_TYPE.FORM) {
      return current
    }
    
    return null
  })

  // 切换抽屉模式
  const toggleDrawerMode = (mode: 'read' | 'edit') => {
    if (mode === 'edit' && (!editFunctionDetail.value || !detailRowData.value)) {
      ElNotification.warning({
        title: '提示',
        message: '无法进入编辑模式'
      })
      return
    }
    detailDrawerMode.value = mode
  }

  // 导航详情（上一个/下一个）
  const handleNavigateDetail = async (direction: 'prev' | 'next') => {
    if (detailTableData.value.length === 0) return

    let newIndex = currentDetailIndex.value
    if (direction === 'prev' && newIndex > 0) {
      newIndex--
    } else if (direction === 'next' && newIndex < detailTableData.value.length - 1) {
      newIndex++
    } else {
      return
    }

    currentDetailIndex.value = newIndex
    const row = detailTableData.value[newIndex]
    detailRowData.value = row
    detailOriginalRow.value = deepClone(row)
    detailDrawerMode.value = 'read'  // 切换记录时，重置为查看模式
    
    // 收集新行的用户字段并查询用户信息
    const userFields = detailFields.value.filter(f => f.widget?.type === 'user')
    if (userFields.length > 0) {
      const usernames: string[] = []
      userFields.forEach(field => {
        const value = row[field.code]
        if (value) {
          if (Array.isArray(value)) {
            usernames.push(...value.map(v => String(v)))
          } else {
            usernames.push(String(value))
          }
        }
      })
      
      if (usernames.length > 0) {
        try {
          const { useUserInfoStore } = await import('@/stores/userInfo')
          const userInfoStore = useUserInfoStore()
          const users = await userInfoStore.batchGetUserInfo([...new Set(usernames)])
          // 更新到 detailUserInfoMap
          detailUserInfoMap.value = new Map()
          users.forEach(user => {
            detailUserInfoMap.value.set(user.username, user)
          })
        } catch (error) {
          // 静默失败
        }
      }
    }
  }

  // 提交编辑
  const submitDrawerEdit = async () => {
    const currentDetail = options.currentFunctionDetail()
    if (!currentDetail || !detailRowData.value || !detailFormRendererRef.value) {
      ElMessage.error('编辑表单未准备就绪')
      return
    }
    
    try {
      drawerSubmitting.value = true
      const submitData = detailFormRendererRef.value.prepareSubmitDataWithTypeConversion()
      const oldValues = detailOriginalRow.value
        ? deepClone(detailOriginalRow.value)
        : undefined
      const updatedRow = await tableApplicationService.updateRow(
        currentDetail,
        detailRowData.value.id,
        submitData,
        oldValues
      )
      if (updatedRow) {
        detailRowData.value = { ...updatedRow }
        detailOriginalRow.value = deepClone(updatedRow)
        await refreshDetailRowData()
        ElNotification.success({
          title: '成功',
          message: '更新成功'
        })
        detailDrawerMode.value = 'read'
        detailDrawerVisible.value = false
      }
    } catch (error: any) {
      ElNotification.error({
        title: '错误',
        message: error?.response?.data?.message || error?.message || '更新失败'
      })
    } finally {
      drawerSubmitting.value = false
    }
  }

  // 刷新详情行数据
  const refreshDetailRowData = async (): Promise<void> => {
    if (!detailRowData.value) return
    const currentId = detailRowData.value.id
    if (currentId === undefined || currentId === null) return
    const state = tableStateManager?.getState?.()
    const tableData = state?.tableData
    if (!Array.isArray(tableData)) {
      return
    }
    const updatedRow = tableData.find((row: any) => String(row.id) === String(currentId))
    if (updatedRow) {
      detailRowData.value = { ...updatedRow }
      detailOriginalRow.value = deepClone(updatedRow)
    }
  }

  // 获取详情字段值
  const getDetailFieldValue = (fieldCode: string): FieldValue => {
    if (!detailRowData.value) return { raw: null, display: '', meta: {} }
    const value = detailRowData.value[fieldCode]
    return { 
      raw: value, 
      display: typeof value === 'object' ? JSON.stringify(value) : String(value ?? ''), 
      meta: {} 
    }
  }

  // 处理详情抽屉关闭（移除 URL 参数）
  const handleDetailDrawerClose = () => {
    // 如果当前 URL 有 _tab=detail 参数，移除它
    // 🔥 阶段3：改为事件驱动，通过 RouteManager 统一处理路由更新
    if (route.query._tab === 'detail') {
      const query: Record<string, string | string[]> = {}
      Object.keys(route.query).forEach(key => {
        if (key !== '_tab' && key !== '_id') {
          const value = route.query[key]
          if (value !== null && value !== undefined) {
            query[key] = Array.isArray(value) 
              ? value.filter(v => v !== null).map(v => String(v))
              : String(value)
          }
        }
      })
      
      // 🔥 发出路由更新请求事件
      eventBus.emit(RouteEvent.updateRequested, {
        query,
        replace: true,
        preserveParams: {
          state: true  // 保留其他状态参数
        },
        source: 'detail-drawer-close'
      })
    }
  }

  // 打开详情抽屉（从表格行点击）
  const openDetailDrawer = async (row: Record<string, any>, index?: number, tableData?: any[]) => {
    const currentDetail = options.currentFunctionDetail()
    if (!currentDetail) return
    
    detailRowData.value = row
    detailOriginalRow.value = deepClone(row)
    detailDrawerTitle.value = currentDetail.name || '详情'
    detailFields.value = (currentDetail.response || []) as FieldConfig[]
    
    // 更新 URL 为 ?_tab=detail&_id=xxx（用于分享）
    // 🔥 阶段3：改为事件驱动，通过 RouteManager 统一处理路由更新
    if (options.currentFunction()) {
      const id = row.id || row._id
      if (id) {
        const query: Record<string, string | string[]> = {}
        // 保留现有参数
        Object.keys(route.query).forEach(key => {
          const value = route.query[key]
          if (value !== null && value !== undefined) {
            query[key] = Array.isArray(value) 
              ? value.filter(v => v !== null).map(v => String(v))
              : String(value)
          }
        })
        // 添加详情参数
        query._tab = 'detail'
        query._id = String(id)
        
        // 🔥 发出路由更新请求事件
        eventBus.emit(RouteEvent.updateRequested, {
          query,
          replace: true,
          preserveParams: {
            state: true  // 保留状态参数
          },
          source: 'detail-drawer-open'
        })
      }
    }
    
    // 保存表格数据和索引（用于上一条下一条导航）
    if (tableData && Array.isArray(tableData) && tableData.length > 0) {
      detailTableData.value = tableData
      if (typeof index === 'number' && index >= 0) {
        currentDetailIndex.value = index
      } else {
        // 如果没有传递 index，尝试从 tableData 中查找
        const idField = detailFields.value.find(f => f.code === 'id' || f.widget?.type === 'number')
        if (idField && row[idField.code]) {
          const foundIndex = tableData.findIndex((r: any) => r[idField.code] === row[idField.code])
          currentDetailIndex.value = foundIndex >= 0 ? foundIndex : -1
        } else {
          // 如果没有 id 字段，尝试通过对象匹配
          const foundIndex = tableData.findIndex((r: any) => JSON.stringify(r) === JSON.stringify(row))
          currentDetailIndex.value = foundIndex >= 0 ? foundIndex : -1
        }
      }
    } else {
      // 如果没有传递 tableData，尝试从 StateManager 获取
      try {
        const tableStateManager = serviceFactory.getTableStateManager()
        const tableData = tableStateManager.getData() || []
        if (tableData && Array.isArray(tableData) && tableData.length > 0) {
          detailTableData.value = tableData
          const idField = detailFields.value.find(f => f.code === 'id' || f.widget?.type === 'number')
          if (idField && row[idField.code]) {
            const foundIndex = tableData.findIndex((r: any) => r[idField.code] === row[idField.code])
            currentDetailIndex.value = foundIndex >= 0 ? foundIndex : -1
          } else {
            // 如果没有 id 字段，尝试通过对象匹配
            const foundIndex = tableData.findIndex((r: any) => JSON.stringify(r) === JSON.stringify(row))
            currentDetailIndex.value = foundIndex >= 0 ? foundIndex : -1
          }
        } else {
          detailTableData.value = []
          currentDetailIndex.value = -1
        }
      } catch (error) {
        detailTableData.value = []
        currentDetailIndex.value = -1
      }
    }
    
    // 收集详情中的用户字段，批量查询用户信息
    const userFields = detailFields.value.filter(f => f.widget?.type === 'user')
    if (userFields.length > 0) {
      const usernames: string[] = []
      userFields.forEach(field => {
        const value = row[field.code]
        if (value) {
          if (Array.isArray(value)) {
            usernames.push(...value.map(v => String(v)))
          } else {
            usernames.push(String(value))
          }
        }
      })
      
      if (usernames.length > 0) {
        try {
          const { useUserInfoStore } = await import('@/stores/userInfo')
          const userInfoStore = useUserInfoStore()
          const users = await userInfoStore.batchGetUserInfo([...new Set(usernames)])
          // 更新到 detailUserInfoMap
          detailUserInfoMap.value = new Map()
          users.forEach(user => {
            detailUserInfoMap.value.set(user.username, user)
          })
        } catch (error) {
          // 静默失败
        }
      }
    }
    
    // 重置为只读模式
    detailDrawerMode.value = 'read'
    detailDrawerVisible.value = true
  }

  // 设置 URL 参数监听（用于分享链接）
  // 🔥 阶段4：改为监听 RouteEvent.queryChanged 事件，而不是直接 watch route.query
  // 这样可以避免程序触发的路由更新导致循环
  const setupUrlWatch = () => {
    eventBus.on(RouteEvent.queryChanged, async (payload: { query: any, oldQuery: any, source: string }) => {
      // 🔥 只处理用户操作（浏览器前进/后退）或外部变化，不处理程序触发的更新
      if (payload.source === 'router-change') {
        const tab = payload.query._tab
        const id = payload.query._id
        const detail = options.currentFunctionDetail()
        
        // 使用 nextTick 确保 detail 已更新
        await nextTick()
        
        // 继续原有的逻辑（从 watch 中复制）
        if (tab === 'detail' && id && detail && detail.template_type === TEMPLATE_TYPE.TABLE) {
        // 确保函数详情已加载
        if (!options.currentFunction()) {
          return
        }
        
        const rowId = Number(id)
        if (isNaN(rowId)) {
          return
        }
        
        // 从表格数据中查找对应 id 的记录
        try {
          const tableStateManager = serviceFactory.getTableStateManager()
          let tableData = tableStateManager.getData() || []
          
          // 尝试通过 id 字段查找
          let targetRow = tableData.find((r: any) => r.id === rowId || r._id === rowId)
          
          // 如果当前页没有找到，尝试通过搜索 id 来加载数据
          if (!targetRow) {
            // 先等待表格数据加载完成（如果表格正在加载）
            let retries = 0
            while (tableData.length === 0 && retries < 10) {
              await nextTick()
              await new Promise(resolve => setTimeout(resolve, 300))
              tableData = tableStateManager.getData() || []
              targetRow = tableData.find((r: any) => r.id === rowId || r._id === rowId)
              if (targetRow) break
              retries++
            }
            
            // 如果还是没有找到，尝试通过搜索 id 来加载
            if (!targetRow && options.currentFunctionDetail()) {
              try {
                const tableApplicationService = serviceFactory.getTableApplicationService()
                // 通过搜索 id 字段来加载数据
                const idField = options.currentFunctionDetail()?.response?.find((f: FieldConfig) => 
                  f.code === 'id' || f.code.toLowerCase() === 'id'
                )
                
                if (idField) {
                  // 设置搜索条件为 id = rowId
                  const searchParams: Record<string, any> = {}
                  searchParams[idField.code] = rowId
                  
                  // 加载数据（使用搜索参数）
                  await tableApplicationService.loadData(
                    options.currentFunctionDetail()!,
                    searchParams, // 搜索参数
                    undefined, // 排序参数
                    { page: 1, pageSize: 20 } // 分页参数
                  )
                  
                  // 重新获取数据
                  tableData = tableStateManager.getData() || []
                  targetRow = tableData.find((r: any) => r.id === rowId || r._id === rowId)
                }
              } catch (error) {
                // 静默失败
              }
            }
          }
          
          if (targetRow) {
            // 找到记录，打开详情抽屉
            const index = tableData.findIndex((r: any) => r.id === rowId || r._id === rowId)
            detailRowData.value = targetRow
            detailOriginalRow.value = deepClone(targetRow)
            detailDrawerTitle.value = detail.name || '详情'
            detailFields.value = (detail.response || []) as FieldConfig[]
            detailTableData.value = tableData
            currentDetailIndex.value = index >= 0 ? index : -1
            
            // 收集用户字段信息
            const userFields = detailFields.value.filter(f => f.widget?.type === 'user')
            if (userFields.length > 0) {
              const usernames: string[] = []
              userFields.forEach(field => {
                const value = targetRow[field.code]
                if (value) {
                  if (Array.isArray(value)) {
                    usernames.push(...value.map(v => String(v)))
                  } else {
                    usernames.push(String(value))
                  }
                }
              })
              
              if (usernames.length > 0) {
                try {
                  const { useUserInfoStore } = await import('@/stores/userInfo')
                  const userInfoStore = useUserInfoStore()
                  const users = await userInfoStore.batchGetUserInfo([...new Set(usernames)])
                  detailUserInfoMap.value = new Map()
                  users.forEach(user => {
                    detailUserInfoMap.value.set(user.username, user)
                  })
                  } catch (error) {
                  // 静默失败
                }
              }
            }
            
            detailDrawerMode.value = 'read'
            detailDrawerVisible.value = true
          } else {
            ElNotification.warning({
              title: '提示',
              message: `未找到 id 为 ${rowId} 的记录，可能不在当前页`
            })
          }
        } catch (error) {
          // 静默失败
        }
      }
    })
  }

  return {
    // 状态
    detailDrawerVisible,
    detailDrawerTitle,
    detailRowData,
    detailFields,
    detailOriginalRow,
    detailDrawerMode,
    drawerSubmitting,
    detailFormRendererRef,
    detailUserInfoMap,
    detailTableData,
    currentDetailIndex,
    editFunctionDetail,
    
    // 方法
    toggleDrawerMode,
    handleNavigateDetail,
    submitDrawerEdit,
    refreshDetailRowData,
    getDetailFieldValue,
    handleDetailDrawerClose,
    openDetailDrawer,
    
    // 设置
    setupUrlWatch
  }
}

