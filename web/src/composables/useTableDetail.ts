/**
 * useTableDetail - 表格详情抽屉 Composable
 * 
 * 负责详情抽屉的状态管理和业务逻辑：
 * - 详情抽屉的显示/隐藏
 * - 详情记录的导航（上一个/下一个）
 * - 编辑模式的切换
 * - URL 状态恢复
 * 
 * 设计原则：
 * - 单一职责：只负责详情相关的逻辑
 * - 可复用：可在多个表格组件中复用
 * - 可测试：独立的函数，易于单元测试
 */

import { ref, computed, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElNotification } from 'element-plus'
import { Logger } from '@/core/utils/logger'
import { ErrorHandler } from '@/core/utils/ErrorHandler'
import { useUserInfoStore } from '@/stores/userInfo'
import { collectFilesUploadUsersFromRow } from '@/utils/tableUserInfo'
import { eventBus, RouteEvent } from '@/architecture/infrastructure/eventBus'
import { RouteSource } from '@/utils/routeSource'
import type { Function as FunctionType, ServiceTree } from '@/types'
import type { FieldConfig, FunctionDetail } from '@/core/types/field'
import FormView from '@/architecture/presentation/views/FormView.vue'

export interface UseTableDetailOptions {
  functionData: FunctionType
  currentFunction?: ServiceTree
  tableData: any[]
  visibleFields: FieldConfig[]
  idField?: FieldConfig
  linkFields: FieldConfig[]
  hasUpdateCallback: boolean
  onUpdate: (id: number, data: any, oldData: any) => Promise<boolean>
  onRefresh: () => Promise<void>
}

export function useTableDetail(options: UseTableDetailOptions) {
  const router = useRouter()
  const userInfoStore = useUserInfoStore()

  // ==================== 状态 ====================
  
  /** 详情抽屉显示状态 */
  const showDetailDrawer = ref(false)
  
  /** 当前详情的行数据 */
  const currentDetailRow = ref<any>(null)
  
  /** 当前详情的行索引 */
  const currentDetailIndex = ref(-1)
  
  /** 详情模式：查看/编辑 */
  const detailMode = ref<'view' | 'edit'>('view')
  
  /** 详情编辑模式的 FormView 引用 */
  const detailFormViewRef = ref<InstanceType<typeof FormView>>()
  
  /** 详情编辑提交状态 */
  const detailSubmitting = ref(false)

  // ==================== URL 状态管理 ====================
  
  /** 防止循环调用的标志 */
  let isClosingDetail = false
  let isRestoringDetail = false
  /** 当前表格的 functionData ID，用于判断 _detail_id 是否属于当前表格 */
  let currentFunctionDataId: number | null = null

  // ==================== 计算属性 ====================
  
  /**
   * 获取 full_code_path
   * 优先使用 currentFunction.full_code_path，否则从 functionData.router 构建
   */
  const getFullCodePath = computed(() => {
    if (options.currentFunction?.full_code_path) {
      return options.currentFunction.full_code_path
    }
    if (options.functionData?.full_code_path) {
      return options.functionData.full_code_path
    }
    // 从 router 构建：/user/app/router -> /user/app/router
    if (options.functionData?.router) {
      return options.functionData.router
    }
    return ''
  })

  /**
   * 获取当前行的 row_id
   */
  const getCurrentRowId = computed(() => {
    if (!currentDetailRow.value || !options.idField) {
      return 0
    }
    const rowId = currentDetailRow.value[options.idField.code]
    return rowId ? Number(rowId) : 0
  })

  /**
   * 构建编辑用的 FunctionDetail
   * 只包含可编辑的字段（根据 table_permission 过滤）
   */
  const editFunctionDetail = computed<FunctionDetail>(() => {
    // 过滤字段（只显示可编辑的字段）
    const editableFields = options.functionData.response.filter((field: FieldConfig) => {
      const permission = field.table_permission
      // 编辑模式：显示空、update 权限的字段
      return !permission || permission === '' || permission === 'update'
    })
    
    // 🔥 method 是必需的，如果不存在应该抛出错误，而不是使用默认值
    if (!options.functionData.method) {
      throw new Error(`[useTableDetail] functionData.method 不存在，无法构建 editFunctionDetail。router: ${options.functionData.router}`)
    }
    
    return {
      id: 0,
      app_id: 0,
      tree_id: 0,
      // 🔥 使用原函数的 method（GET），而不是编辑操作的 method（PUT）
      // 这样 OnSelectFuzzy 回调才能正确获取到原函数的 method
      method: options.functionData.method,
      router: options.functionData.router,
      has_config: false,
      create_tables: '',
      callbacks: options.functionData.callbacks,
      template_type: 'form',
      request: editableFields,  // 使用过滤后的字段
      response: [],
      created_at: '',
      updated_at: '',
      full_code_path: ''
    }
  })

  // ==================== 方法 ====================
  
  /**
   * 显示详情
   * 打开详情抽屉，加载指定行的数据
   * @param row 行数据
   * @param index 行索引
   */
  const handleShowDetail = async (row: any, index: number): Promise<void> => {
    currentDetailRow.value = row
    currentDetailIndex.value = index
    detailMode.value = 'view'  // 重置为查看模式
    showDetailDrawer.value = true
    
    // 🔥 收集当前行的 files widget 的 upload_user 并查询用户信息
    const filesUploadUsers = collectFilesUploadUsersFromRow(row, options.visibleFields)
    
    if (filesUploadUsers.length > 0) {
      // 批量查询用户信息（自动处理缓存）
      const users = await userInfoStore.batchGetUserInfo(filesUploadUsers)
      
      // 🔥 userInfoStore 已经缓存了用户信息，FilesWidget 会直接从 store 读取
    }

    // 🔥 更新 URL，添加 _detail_id 和 _detail_function_id 参数（用于分享和刷新后恢复状态）
    // 只有在 URL 中没有相同的 _detail_id 时才更新，避免循环触发
    if (options.idField && row[options.idField.code]) {
      const detailId = String(row[options.idField.code])
      const currentDetailId = String(router.currentRoute.value.query._detail_id || '')
      const currentFunctionId = options.functionData.id
      
      // 🔥 关键：只有在不是恢复过程中，且 URL 中没有相同的 _detail_id 时才更新
      if (currentDetailId !== detailId && !isRestoringDetail) {
        // 🔥 更新当前表格的 ID，确保 _detail_id 属于当前表格
        if (currentFunctionDataId !== currentFunctionId) {
          currentFunctionDataId = currentFunctionId
        }
        
        const query = { ...router.currentRoute.value.query }
        query._detail_id = detailId
        query._detail_function_id = String(currentFunctionId)  // 🔥 同时存储 functionDataId
        // 🔥 通过事件总线更新路由，统一管理
        eventBus.emit(RouteEvent.updateRequested, {
          query,
          replace: true,
          preserveParams: {
            table: true,  // 保留 table 参数
            search: true, // 保留搜索参数
            state: true   // 保留状态参数
          },
          source: RouteSource.TABLE_DETAIL_OPEN
        })
      }
    }
  }

  /**
   * 导航（上一个/下一个）
   * 在详情抽屉中切换记录
   * @param direction 导航方向
   */
  const handleNavigate = async (direction: 'prev' | 'next'): Promise<void> => {
    if (!options.tableData || options.tableData.length === 0) return

    let newIndex = currentDetailIndex.value
    if (direction === 'prev' && newIndex > 0) {
      newIndex--
    } else if (direction === 'next' && newIndex < options.tableData.length - 1) {
      newIndex++
    } else {
      return
    }

    currentDetailIndex.value = newIndex
    const row = options.tableData[newIndex]
    currentDetailRow.value = row
    detailMode.value = 'view'  // 切换记录时，重置为查看模式
    
    // 🔥 收集新行的 files widget 的 upload_user 并查询用户信息
    const filesUploadUsers = collectFilesUploadUsersFromRow(row, options.visibleFields)
    if (filesUploadUsers.length > 0) {
      // 批量查询用户信息（自动处理缓存）
      const users = await userInfoStore.batchGetUserInfo(filesUploadUsers)
      // 🔥 userInfoStore 已经缓存了用户信息，FilesWidget 会直接从 store 读取
    }
    
    // 🔥 更新 URL，更新 _detail_id 和 _detail_function_id 参数
    if (options.idField && row[options.idField.code]) {
      const detailId = String(row[options.idField.code])
      const currentFunctionId = options.functionData.id
      const query = { ...router.currentRoute.value.query }
      query._detail_id = detailId
      query._detail_function_id = String(currentFunctionId)  // 🔥 同时更新 functionDataId
      // 🔥 通过事件总线更新路由，统一管理
      eventBus.emit(RouteEvent.updateRequested, {
        query,
        replace: true,
        preserveParams: {
          table: true,  // 保留 table 参数
          search: true, // 保留搜索参数
          state: true   // 保留状态参数
        },
        source: RouteSource.TABLE_DETAIL_NAVIGATE
      })
    }
  }

  /**
   * 切换到编辑模式
   */
  const switchToEditMode = async (): Promise<void> => {
    if (!currentDetailRow.value) {
      ElMessage.error('记录数据不存在')
      return
    }
    
    detailMode.value = 'edit'
    
    // 等待 FormRenderer 初始化完成
    await nextTick()
    
    // 再次等待，确保 FormRenderer 完全准备好
    let retries = 0
    while (retries < 10 && !detailFormViewRef.value) {
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 50))
      retries++
    }
    
    if (!detailFormViewRef.value) {
      ElMessage.error('编辑表单未准备就绪，请稍后重试')
      detailMode.value = 'view'
    }
  }

  /**
   * 切换回查看模式
   */
  const switchToViewMode = (): void => {
    detailMode.value = 'view'
  }

  /**
   * 保存（详情编辑模式）
   */
  const handleDetailSave = async (): Promise<void> => {
    if (!detailFormViewRef.value) {
      ElMessage.error('表单引用不存在')
      return
    }
    
    if (!currentDetailRow.value || !currentDetailRow.value.id) {
      ElMessage.error('记录 ID 不存在')
      return
    }
    
    try {
      detailSubmitting.value = true
      
      const oldValues = currentDetailRow.value
      
      // 1. 准备更新数据（表格更新场景，只返回变更的字段）
      const submitData = await detailFormViewRef.value.prepareUpdateData(oldValues)
      
      // 2. 调用更新接口（复用现有的更新逻辑）
      const success = await options.onUpdate(currentDetailRow.value.id, submitData, oldValues)
      
      if (success) {
        // 3. 刷新当前记录数据
        await refreshCurrentDetailRow()
        
        // 4. 关闭抽屉（保存成功后关闭）
        showDetailDrawer.value = false
        detailMode.value = 'view'
      }
    } catch (error: any) {
      Logger.error('useTableDetail', '保存失败', error)
      // 🔥 统一使用 msg 字段
      const errorMessage = error?.response?.data?.msg || error?.message || '保存失败'
      // 🔥 使用 ElNotification 替代 ElMessage，确保显示在抽屉上方（z-index 更高）
      ElNotification({
        title: '保存失败',
        message: errorMessage,
        type: 'error',
        duration: 5000,
        position: 'top-right'
      })
    } finally {
      detailSubmitting.value = false
    }
  }

  /**
   * 刷新当前详情记录数据
   */
  const refreshCurrentDetailRow = async (): Promise<void> => {
    if (!currentDetailRow.value || !currentDetailRow.value.id) {
      return
    }
    
    try {
      // 🔥 不需要重新加载表格数据，因为 onUpdate 已经加载过了
      // 直接从最新的表格数据中找到当前记录
      const rowId = currentDetailRow.value.id
      let updatedRow: any = null
      let index = -1
      
      for (let i = 0; i < options.tableData.length; i++) {
        if (options.tableData[i].id === rowId) {
          updatedRow = options.tableData[i]
          index = i
          break
        }
      }
      
      if (updatedRow) {
        currentDetailRow.value = updatedRow
        if (index >= 0) {
          currentDetailIndex.value = index
        }
        
        // 🔥 收集更新后的 files widget 的 upload_user 并查询用户信息
        const filesUploadUsers = collectFilesUploadUsersFromRow(updatedRow, options.visibleFields)
        
        if (filesUploadUsers.length > 0) {
          // 批量查询用户信息（自动处理缓存）
          const users = await userInfoStore.batchGetUserInfo(filesUploadUsers)
          
          // 🔥 userInfoStore 已经缓存了用户信息，FilesWidget 会直接从 store 读取
        }
      }
    } catch (error) {
      Logger.error('useTableDetail', '刷新记录数据失败', error)
    }
  }

  /**
   * 处理详情抽屉关闭
   * 清理详情状态和 URL 参数
   */
  const handleDetailDrawerClose = (): void => {
    // 防止重复调用
    if (isClosingDetail) {
      return
    }
    isClosingDetail = true
    
    // 清空详情数据
    currentDetailRow.value = null
    currentDetailIndex.value = -1
    detailMode.value = 'view'
    
    // 清理 URL 中的 _detail_id 和 _detail_function_id 参数
    const query = { ...router.currentRoute.value.query }
    let hasChanges = false
    if (query._detail_id) {
      delete query._detail_id
      hasChanges = true
    }
    if (query._detail_function_id) {
      delete query._detail_function_id
      hasChanges = true
    }
    
    if (hasChanges) {
      // 🔥 通过事件总线更新路由，统一管理
      eventBus.emit(RouteEvent.updateRequested, {
        query,
        replace: true,
        preserveParams: {
          table: true,  // 保留 table 参数
          search: true, // 保留搜索参数
          state: true   // 保留状态参数（除了 _detail_id 和 _detail_function_id）
        },
        source: RouteSource.TABLE_DETAIL_CLOSE
      })
      // 使用 nextTick 确保路由更新完成
      nextTick().finally(() => {
        isClosingDetail = false
      })
    } else {
      isClosingDetail = false
    }
  }

  /**
   * 从 URL 恢复详情状态
   * 如果 URL 中有 _detail_id 参数，自动打开对应的详情
   */
  const restoreDetailFromURL = async (): Promise<void> => {
    // 防止循环调用
    if (isRestoringDetail || isClosingDetail) {
      return
    }
    
    const query = router.currentRoute.value.query
    const detailId = query._detail_id
    const detailFunctionId = query._detail_function_id  // 🔥 获取 _detail_id 对应的 functionDataId
    
    if (!detailId || !options.idField) {
      return
    }
    
    // 🔥 关键：检查 _detail_id 是否属于当前表格
    const currentFunctionId = options.functionData.id
    
    // 🔥 如果 URL 中有 _detail_function_id，且与当前 functionData.id 不匹配，说明这个 _detail_id 不属于当前表格
    if (detailFunctionId && String(detailFunctionId) !== String(currentFunctionId)) {
      // 清理不属于当前表格的 _detail_id
      const queryToClean = { ...router.currentRoute.value.query }
      if (queryToClean._detail_id) {
        delete queryToClean._detail_id
      }
      if (queryToClean._detail_function_id) {
        delete queryToClean._detail_function_id
      }
      // 🔥 通过事件总线更新路由，统一管理
      eventBus.emit(RouteEvent.updateRequested, {
        query: queryToClean,
        replace: true,
        preserveParams: {
          table: true,  // 保留 table 参数
          search: true, // 保留搜索参数
          state: true   // 保留状态参数（除了 _detail_id 和 _detail_function_id）
        },
        source: RouteSource.TABLE_DETAIL_CLEANUP
      })
      return
    }
    
    // 🔥 如果 currentFunctionDataId 与当前 functionData.id 不匹配，说明切换了表格
    // 此时旧的 _detail_id 不应该恢复，应该清理
    if (currentFunctionDataId !== null && currentFunctionDataId !== currentFunctionId) {
      // 更新 currentFunctionDataId 为新的表格 ID
      currentFunctionDataId = currentFunctionId
      // 清理不属于当前表格的 _detail_id
      const queryToClean = { ...router.currentRoute.value.query }
      if (queryToClean._detail_id) {
        delete queryToClean._detail_id
      }
      if (queryToClean._detail_function_id) {
        delete queryToClean._detail_function_id
      }
      // 🔥 通过事件总线更新路由，统一管理
      eventBus.emit(RouteEvent.updateRequested, {
        query: queryToClean,
        replace: true,
        preserveParams: {
          table: true,  // 保留 table 参数
          search: true, // 保留搜索参数
          state: true   // 保留状态参数（除了 _detail_id 和 _detail_function_id）
        },
        source: RouteSource.TABLE_DETAIL_CLEANUP
      })
      return
    }
    
    // 🔥 更新 currentFunctionDataId（如果还是 null，说明是首次加载）
    if (currentFunctionDataId === null) {
      currentFunctionDataId = currentFunctionId
    }
    
    // 如果详情已经打开，且是同一个记录，不需要重复打开
    if (showDetailDrawer.value && currentDetailRow.value) {
      const currentId = currentDetailRow.value[options.idField.code]
      if (String(currentId) === String(detailId)) {
        return
      }
    }
    
    // 等待表格数据加载完成
    if (!options.tableData || options.tableData.length === 0) {
      return
    }
    
    isRestoringDetail = true
    
    try {
      // 查找对应的记录
      const detailIdStr = String(detailId)
      const rowIndex = options.tableData.findIndex((row: any) => {
        const rowId = row[options.idField!.code]
        return String(rowId) === detailIdStr
      })
      
      if (rowIndex >= 0) {
        const row = options.tableData[rowIndex]
        const rowId = row[options.idField!.code]
        
        // 🔥 关键：验证找到的记录 ID 是否真的匹配
        // 如果 rowId 与 detailId 不匹配，说明这个 _detail_id 不属于当前表格，应该清理
        if (String(rowId) !== detailIdStr) {
          Logger.warn('useTableDetail', `找到的记录 ID 不匹配（期望: ${detailIdStr}, 实际: ${rowId}），清理 _detail_id`)
          // 清理不属于当前表格的 _detail_id
          const queryToClean = { ...router.currentRoute.value.query }
          if (queryToClean._detail_id) {
            delete queryToClean._detail_id
            // 🔥 通过事件总线更新路由，统一管理
            eventBus.emit(RouteEvent.updateRequested, {
              query: queryToClean,
              replace: true,
              preserveParams: {
                table: true,  // 保留 table 参数
                search: true, // 保留搜索参数
                state: true   // 保留状态参数
              },
              source: RouteSource.TABLE_DETAIL_CLEANUP_INVALID_ID
            })
          }
          return
        }
        
        // 🔥 直接设置状态，不更新 URL（避免循环）
        currentDetailRow.value = row
        currentDetailIndex.value = rowIndex
        detailMode.value = 'view'
        showDetailDrawer.value = true
        
        // 收集用户信息
        const filesUploadUsers = collectFilesUploadUsersFromRow(row, options.visibleFields)
        if (filesUploadUsers.length > 0) {
          const users = await userInfoStore.batchGetUserInfo(filesUploadUsers)
          for (const user of users) {
            if (user.username) {
              // 🔥 userInfoStore 已经缓存了用户信息，FilesWidget 会直接从 store 读取
            }
          }
        }
      } else {
        // 如果当前页没有找到，可能是分页问题，或者这个 _detail_id 不属于当前表格
        Logger.warn('useTableDetail', `未找到 ID 为 ${detailId} 的记录（可能在其他页、已被删除或不属于当前表格）`)
        // 🔥 清理 URL 中的 _detail_id 和 _detail_function_id，因为找不到对应的记录
        const queryToClean = { ...router.currentRoute.value.query }
        let hasChanges = false
        if (queryToClean._detail_id) {
          delete queryToClean._detail_id
          hasChanges = true
        }
        if (queryToClean._detail_function_id) {
          delete queryToClean._detail_function_id
          hasChanges = true
        }
        if (hasChanges) {
          // 🔥 通过事件总线更新路由，统一管理
          eventBus.emit(RouteEvent.updateRequested, {
            query: queryToClean,
            replace: true,
            preserveParams: {
              table: true,  // 保留 table 参数
              search: true, // 保留搜索参数
              state: true   // 保留状态参数
            },
            source: RouteSource.TABLE_DETAIL_CLEANUP_NOT_FOUND
          })
        }
      }
    } finally {
      isRestoringDetail = false
    }
  }

  // ==================== 监听器 ====================
  
  // 🔥 监听 showDetailDrawer 变化，确保关闭时清理状态
  watch(showDetailDrawer, (newValue: boolean, oldValue: boolean) => {
    // 当抽屉从打开变为关闭时，清理状态
    if (oldValue === true && newValue === false && !isClosingDetail) {
      handleDetailDrawerClose()
    }
  })

  // 🔥 监听 functionData 变化，切换表格时清空详情状态
  watch(() => options.functionData, (newFunctionData: FunctionType, oldFunctionData?: FunctionType) => {
    const oldId = oldFunctionData?.id
    const newId = newFunctionData?.id
    
    // 🔥 关键：如果表格 ID 真的变化了，才清理状态
    // 如果 oldId 和 newId 相同，说明是同一个表格重新渲染，不需要清理
    if (oldId !== undefined && newId !== undefined && oldId !== newId) {
      // 更新当前表格的 ID（立即更新，确保后续检查正确）
      currentFunctionDataId = newId || null
      
      // 切换表格时，清空详情状态
      currentDetailRow.value = null
      currentDetailIndex.value = -1
      detailMode.value = 'view'
      showDetailDrawer.value = false
      
      // 清理 URL 中的 _detail_id 和 _detail_function_id 参数（因为这是上一个表格的详情 ID）
      const query = { ...router.currentRoute.value.query }
      let hasChanges = false
      if (query._detail_id) {
        delete query._detail_id
        hasChanges = true
      }
      if (query._detail_function_id) {
        delete query._detail_function_id
        hasChanges = true
      }
      if (hasChanges) {
        // 🔥 通过事件总线更新路由，统一管理
        eventBus.emit(RouteEvent.updateRequested, {
          query,
          replace: true,
          preserveParams: {
            table: true,  // 保留 table 参数
            search: true, // 保留搜索参数
            state: true   // 保留状态参数
          },
          source: RouteSource.TABLE_DETAIL_CLEANUP_FUNCTION_CHANGE
        })
      }
    } else {
      // 如果是首次加载或同一个表格，只更新 currentFunctionDataId（如果还是 null 或需要更新）
      if (newId !== undefined && (currentFunctionDataId === null || currentFunctionDataId !== newId)) {
        currentFunctionDataId = newId
      }
    }
  }, { deep: true, immediate: true })

  // 🔥 监听表格数据变化，当数据加载完成且 URL 中有 _detail_id 时，自动打开详情
  watch(() => [options.tableData, router.currentRoute.value.query._detail_id], () => {
    if (options.tableData && options.tableData.length > 0 && router.currentRoute.value.query._detail_id) {
      // 延迟执行，确保数据已完全渲染
      nextTick(() => {
        restoreDetailFromURL()
      })
    }
  }, { deep: true })

  // ==================== 初始化 ====================
  
  // 🔥 初始化当前表格的 ID
  currentFunctionDataId = options.functionData.id || null

  return {
    // 状态
    showDetailDrawer,
    currentDetailRow,
    currentDetailIndex,
    detailMode,
    detailFormViewRef,
    detailSubmitting,
    
    // 计算属性
    getFullCodePath,
    getCurrentRowId,
    editFunctionDetail,
    
    // 方法
    handleShowDetail,
    handleNavigate,
    switchToEditMode,
    switchToViewMode,
    handleDetailSave,
    handleDetailDrawerClose,
    restoreDetailFromURL
  }
}

