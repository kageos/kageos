/**
 * useWorkspaceDetail - 详情管理 Composable
 * 
 * ============================================
 * 📋 需求说明
 * ============================================
 * 
 * 1. **详情抽屉管理**：
 *    - 从表格行点击打开详情抽屉
 *    - 支持查看模式和编辑模式切换
 *    - 支持上一条/下一条导航
 * 
 * 2. **编辑功能**：
 *    - 编辑模式下显示可编辑字段（根据 `table_permission` 过滤）
 *    - 提交编辑时检查权限（`function:update`）
 *    - 提交成功后刷新表格数据
 * 
 * 3. **URL 同步**：
 *    - 查看模式：`_tab=detail&_id=xxx`
 *    - 编辑模式：不设置 `_tab` 参数（只使用 `_id`）
 *    - 关闭抽屉时清除 URL 参数
 *    - 注意：只有新增模式（`_tab=OnTableAddRow`）才同步表单字段参数到 URL
 * 
 * ============================================
 * 🎯 设计思路
 * ============================================
 * 
 * 1. **模式切换**：
 *    - `read` 模式：只读展示，使用 `_tab=detail`
 *    - `edit` 模式：可编辑，不设置 `_tab` 参数（只使用 `_id`）
 *    - 切换模式时更新 URL，清除表单字段参数
 * 
 * 2. **权限检查**：
 *    - 提交编辑时检查 `function:update` 权限
 *    - 权限不足时显示提示并跳转到申请页面
 * 
 * 3. **数据流**：
 *    - 从表格行数据构建编辑表单的初始数据
 *    - 编辑模式下只显示可编辑字段（`table_permission=update` 或为空）
 *    - 提交时提取表单数据并调用 TableApplicationService.updateRow
 * 
 * ============================================
 * 📝 关键功能
 * ============================================
 * 
 * 1. **openDetailDrawer**：
 *    - 从表格行数据打开详情抽屉
 *    - 设置 URL 为 `_tab=detail&_id=xxx`
 *    - 清除所有表单字段参数
 * 
 * 2. **toggleDrawerMode**：
 *    - 切换查看/编辑模式
 *    - 查看模式：设置 `_tab=detail`
 *    - 编辑模式：不设置 `_tab` 参数（只使用 `_id`）
 *    - 清除表单字段参数（编辑模式下）
 * 
 * 3. **submitDrawerEdit**：
 *    - 检查权限（`function:update`）
 *    - 提取表单数据并提交
 *    - 成功后刷新表格数据，清除 URL 参数
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **权限检查**：
 *    - 必须在提交前检查权限，防止绕过 UI 检查
 *    - 权限不足时，显示提示并跳转到申请页面
 * 
 * 2. **URL 参数管理**：
 *    - 详情抽屉相关参数：`_tab`、`_id`
 *    - 编辑模式下必须清除表单字段参数
 *    - 关闭抽屉时清除所有相关参数
 * 
 * 3. **字段过滤**：
 *    - 编辑模式下只显示 `table_permission=update` 或为空的字段
 *    - 通过 `editFunctionDetail` computed 过滤字段
 */

import { ref, computed, watch, nextTick } from 'vue'
import { deepClone } from '@/utils/clone'
import { useRoute, useRouter } from 'vue-router'
import { ElNotification, ElMessage } from 'element-plus'
import { serviceFactory } from '../../infrastructure/factories'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import FormView from '@/architecture/presentation/views/FormView.vue'
import type { FieldConfig, FieldValue } from '../../domain/types'
import type { FunctionDetail } from '../../domain/interfaces/IFunctionLoader'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { hasPermission, TablePermissions, buildPermissionApplyURL } from '@/utils/permission'
import type { ServiceTree } from '@/types'

export function useWorkspaceDetail(
  options: {
    currentFunctionDetail: () => FunctionDetail | null
    currentFunction: () => any
  },
  serviceProvider: IServiceProvider = serviceFactory  // 🔥 通过参数注入，提高可测试性
) {
  const route = useRoute()
  const router = useRouter()
  const tableApplicationService = serviceProvider.getTableApplicationService()
  const tableStateManager = serviceProvider.getTableStateManager()
  const stateManager = serviceProvider.getWorkspaceStateManager()

  // 详情抽屉状态
  const detailDrawerVisible = ref(false)
  const detailDrawerTitle = ref('详情')
  const detailRowData = ref<Record<string, any> | null>(null)
  const detailFields = ref<FieldConfig[]>([])
  const detailOriginalRow = ref<Record<string, any> | null>(null)
  const detailDrawerMode = ref<'read' | 'edit'>('read')
  const drawerSubmitting = ref(false)
  const detailFormViewRef = ref<InstanceType<typeof FormView> | null>(null)
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
    
    // 🔥 切换模式时更新 URL：查看模式使用 _tab=detail，编辑模式不设置 _tab（只使用 _id）
    const id = detailRowData.value?.id || detailRowData.value?._id
    if (id) {
      // 🔥 获取编辑模式的字段代码集合，用于清除表单字段参数
      const editableFieldCodes = new Set<string>()
      if (editFunctionDetail.value && editFunctionDetail.value.request) {
        editFunctionDetail.value.request.forEach((field: FieldConfig) => {
          editableFieldCodes.add(field.code)
        })
      }
      
      const query: Record<string, string | string[]> = {}
      // 保留现有参数（除了 _tab、_id 和表单字段参数）
      Object.keys(route.query).forEach(key => {
        // 跳过 _tab 和 _id，后面会根据模式设置
        if (key === '_tab' || key === '_id') {
          return
        }
        
        // 🔥 编辑模式：清除所有表单字段参数（这些参数不应该在编辑模式下存在）
        if (mode === 'edit' && editableFieldCodes.has(key)) {
          return
        }
        
        // 保留其他参数（如 table 参数、搜索参数等）
        const value = route.query[key]
        if (value !== null && value !== undefined) {
          query[key] = Array.isArray(value) 
            ? value.filter(v => v !== null).map(v => String(v))
            : String(value)
        }
      })
      
      // 🔥 查看模式：设置 _tab=detail；编辑模式：不设置 _tab（只使用 _id）
      if (mode === 'read') {
        query._tab = 'detail'
      }
      // 编辑模式不设置 _tab，只设置 _id
      query._id = String(id)
      
      // 🔥 发出路由更新请求事件
      eventBus.emit(RouteEvent.updateRequested, {
        query,
        replace: true,
        preserveParams: {
          table: true,   // 保留 table 参数
          search: true,  // 保留搜索参数
          state: true    // 保留其他状态参数
        },
        source: 'detail-drawer-mode-toggle'
      })
    }
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
  const submitDrawerEdit = async (formViewRef?: InstanceType<typeof FormView> | null) => {
    const currentDetail = options.currentFunctionDetail()
    // 🔥 优先使用传入的 formViewRef，如果没有则使用 detailFormViewRef
    const viewRef = formViewRef || detailFormViewRef.value
    
    if (!currentDetail || !detailRowData.value || !viewRef) {
      ElMessage.error('编辑表单未准备就绪')
      return
    }
    
    // 🔥 安全修复：检查更新权限
    const currentFunction = options.currentFunction() as ServiceTree | null
    if (!currentFunction) {
      ElMessage.error('无法获取函数节点信息，无法验证权限')
      return
    }
    
    if (!hasPermission(currentFunction, TablePermissions.update)) {
      ElNotification.warning({
        title: '权限不足',
        message: '您没有更新该表格记录的权限',
        duration: 3000
      })
      // 跳转到权限申请页面
      const applyUrl = buildPermissionApplyURL(
        currentFunction.full_code_path || '',
        TablePermissions.update,
        currentDetail.template_type
      )
      router.push(applyUrl)
      return
    }
    
    try {
      drawerSubmitting.value = true
      const submitData = viewRef.prepareSubmitDataWithTypeConversion()
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
        
        // 🔥 保存成功后，清除 URL 中的表单字段参数和 _tab 参数
        const editableFieldCodes = new Set<string>()
        if (editFunctionDetail.value && editFunctionDetail.value.request) {
          editFunctionDetail.value.request.forEach((field: FieldConfig) => {
            editableFieldCodes.add(field.code)
          })
        }
        
        const query: Record<string, string | string[]> = {}
        Object.keys(route.query).forEach(key => {
          // 跳过 _tab 和 _id 参数（详情抽屉相关）
          if (key === '_tab' || key === '_id') {
            return
          }
          
          // 🔥 清除所有表单字段参数
          if (editableFieldCodes.has(key)) {
            return
          }
          
          // 保留其他参数（如 table 参数、搜索参数等）
          const value = route.query[key]
          if (value !== null && value !== undefined) {
            query[key] = Array.isArray(value) 
              ? value.filter(v => v !== null).map(v => String(v))
              : String(value)
          }
        })
        
        // 🔥 发出路由更新请求事件，清除表单字段参数
        eventBus.emit(RouteEvent.updateRequested, {
          query,
          replace: true,
          preserveParams: {
            table: true,   // 保留 table 参数
            search: true,  // 保留搜索参数
            state: true    // 保留其他状态参数
          },
          source: 'detail-drawer-save-success'
        })
        
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
        // 🔥 统一使用 msg 字段
        message: error?.response?.data?.msg || error?.message || '更新失败'
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
    // 如果当前 URL 有 _tab=detail 或 _id 参数，移除它
    // 🔥 阶段3：改为事件驱动，通过 RouteManager 统一处理路由更新
    if (route.query._tab === 'detail' || route.query._id) {
      // 🔥 获取编辑模式的字段代码集合，用于清除表单字段参数
      const editableFieldCodes = new Set<string>()
      if (editFunctionDetail.value && editFunctionDetail.value.request) {
        editFunctionDetail.value.request.forEach((field: FieldConfig) => {
          editableFieldCodes.add(field.code)
        })
      }
      
      // 🔥 清空 formDataStore，避免 FormView 重新初始化时从 URL 读取数据
      const formDataStore = useFormDataStore()
      formDataStore.clear()
      
      const query: Record<string, string | string[]> = {}
      Object.keys(route.query).forEach(key => {
        // 跳过 _tab 和 _id 参数（详情抽屉相关）
        if (key === '_tab' || key === '_id') {
          return
        }
        
        // 🔥 跳过表单字段参数（编辑模式下 FormView 不应该同步到 URL，但如果有残留参数需要清除）
        if (editableFieldCodes.has(key)) {
          return
        }
        
        // 保留其他参数（如 table 参数、搜索参数等）
          const value = route.query[key]
          if (value !== null && value !== undefined) {
            query[key] = Array.isArray(value) 
              ? value.filter(v => v !== null).map(v => String(v))
              : String(value)
        }
      })
      
      // 🔥 发出路由更新请求事件
      eventBus.emit(RouteEvent.updateRequested, {
        query,
        replace: true,
        preserveParams: {
          table: true,   // 保留 table 参数（分页、排序等）
          search: true,  // 保留搜索参数
          state: true    // 保留其他状态参数
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
        // 🔥 获取编辑模式的字段代码集合，用于清除表单字段参数
        const editableFieldCodes = new Set<string>()
        if (editFunctionDetail.value && editFunctionDetail.value.request) {
          editFunctionDetail.value.request.forEach((field: FieldConfig) => {
            editableFieldCodes.add(field.code)
          })
        }
        
        const query: Record<string, string | string[]> = {}
        // 保留现有参数（除了表单字段参数）
        Object.keys(route.query).forEach(key => {
          // 🔥 清除所有表单字段参数（详情和编辑模式下都不应该显示这些参数）
          if (editableFieldCodes.has(key)) {
            return
          }
          
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
            table: true,   // 保留 table 参数
            search: true,  // 保留搜索参数
            state: true    // 保留状态参数
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

  // 打开详情抽屉的辅助函数（从 URL 参数）
  const openDetailFromUrl = async (query: any) => {
    const tab = query._tab
    const id = query._id
    const detail = options.currentFunctionDetail()
    
    // 使用 nextTick 确保 detail 已更新
    await nextTick()
    
    // 继续原有的逻辑（从 watch 中复制）
    // 🔥 支持 _tab=detail（查看模式），编辑模式不设置 _tab 参数
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
          
          // 🔥 根据 _tab 参数设置模式：detail 为查看模式，没有 _tab 时默认为查看模式
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
  }

  // 设置 URL 参数监听（用于分享链接）
  // 🔥 阶段4：改为监听 RouteEvent.queryChanged 事件，而不是直接 watch route.query
  // 这样可以避免程序触发的路由更新导致循环
  const setupUrlWatch = () => {
    // 🔥 初始化时检查 URL 参数（页面刷新场景）
    // 如果 URL 中已经有 _tab=detail&_id=xxx，等待函数详情和表格数据加载完成后打开详情
    if (route.query._tab === 'detail' && route.query._id) {
      // 等待函数详情加载完成
      const checkAndOpen = async () => {
        let retries = 0
        while (retries < 20) { // 最多等待 10 秒
          await nextTick()
          await new Promise(resolve => setTimeout(resolve, 500))
          
          const detail = options.currentFunctionDetail()
          const currentFunction = options.currentFunction()
          
          // 如果函数详情已加载，尝试打开详情
          if (detail && currentFunction && detail.template_type === TEMPLATE_TYPE.TABLE) {
            await openDetailFromUrl(route.query)
            break
          }
          
          retries++
        }
      }
      
      checkAndOpen()
    }
    
    // 监听 URL 参数变化（浏览器前进/后退场景）
    eventBus.on(RouteEvent.queryChanged, async (payload: { query: any, oldQuery: any, source: string }) => {
      // 🔥 只处理用户操作（浏览器前进/后退）或外部变化，不处理程序触发的更新
      if (payload.source === 'router-change') {
        await openDetailFromUrl(payload.query)
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
    detailFormViewRef,
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

