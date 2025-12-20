<template>
  <div class="table-renderer">
    <!-- 工具栏 -->
    <div class="toolbar" v-if="hasAddCallback || (hasDeleteCallback && selectedRows.length > 0)">
      <div class="toolbar-left">
        <el-button v-if="hasAddCallback" type="primary" @click="handleAdd" :icon="Plus">
          新增
        </el-button>
        <el-button 
          v-if="hasDeleteCallback && selectedRows.length > 0" 
          type="danger" 
          @click="handleBatchDelete"
          :icon="Delete"
        >
          批量删除 ({{ selectedRows.length }})
        </el-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <TableSearchBar
      :searchable-fields="searchableFields"
      :search-form="searchForm"
      :function-data="props.functionData"
      @search="handleSearch"
      @reset="handleReset"
      @update:search-form="(value: Record<string, any>) => {
        // 更新搜索表单并同步到 URL
        Object.keys(searchForm.value).forEach(key => {
          if (!(key in value)) {
            delete searchForm.value[key]
          }
        })
        Object.assign(searchForm.value, value)
        syncToURL()
      }"
    />

    <!-- 🔥 排序信息条：显示当前排序状态 -->
    <TableSortBar
      :sorts="sorts"
      :display-sorts="displaySorts"
      :visible-fields="visibleFields"
      @remove-sort="handleRemoveSort"
      @clear-all-sorts="handleClearAllSorts"
    />

    <!-- 表格 -->
    <!-- 
      ⚠️ 关键：在 custom 模式下，需要为每个列设置 sort-order 来显示排序状态
      不要使用 default-sort，因为它会干扰多列排序的显示
    -->
    <el-table
      ref="tableRef"
      v-loading="loading"
      :data="tableData"
      :stripe="false"
      style="width: 100%"
      class="table-with-fixed-column"
      :key="`table-${Object.keys(sortOrderMap).length}`"
      @sort-change="handleSortChange"
      @selection-change="handleSelectionChange"
    >
      <!-- 复选框列（用于批量操作） -->
      <el-table-column
        v-if="hasDeleteCallback"
        type="selection"
        width="55"
        fixed="left"
        :selectable="checkSelectable"
      />

      <!-- 🔥 控制中心列（ID列改造） -->
      <!-- 
        注意：ID 列默认启用排序，显示默认的 id 降序排序状态
        当用户手动排序其他字段时，id 排序会被移除
        ⚠️ ID 字段通常非常适合排序，使用智能识别
      -->
      <el-table-column
        v-if="idField"
        :prop="idField.code"
        label=""
        fixed="left"
        width="80"
        class-name="control-column"
        :sortable="getSortableConfig(idField)"
        :sort-order="sortOrderMap[idField.code] || null"
      >
        <template #default="{ row, $index }">
          <el-button
            link
            type="danger"
            size="small"
            @click="handleShowDetail(row, $index)"
          >
            #{{ row[idField.code] }}
          </el-button>
        </template>
      </el-table-column>

      <!-- 数据列（排除ID列） -->
      <!-- 
        ⚠️ 使用智能识别判断字段是否适合排序
        - 文件字段、结构体字段：不支持排序
        - 大文本字段、多选字段：不推荐排序（默认禁用）
        - 其他字段：适合排序
      -->
      <el-table-column
        v-for="field in dataFields"
        :key="field.code"
        :prop="field.code"
        :label="field.name"
        :sortable="getSortableConfig(field)"
        :sort-order="sortOrderMap[field.code] || null"
        :min-width="getColumnWidth(field)"
      >
        <template #default="{ row, $index }">
          <!-- 🔥 使用 Widget 的 renderTableCell() 方法（组件自治） -->
          <!-- 
            注意：renderTableCell 可能返回字符串或 VNode
            - 字符串：直接显示（用于简单字段）
            - VNode：作为组件渲染（用于复杂字段如 MultiSelect）
          -->
          <template v-if="getCellContent(field, row[field.code]).isString">
            {{ getCellContent(field, row[field.code]).content }}
          </template>
          <!-- 🔥 VNode 直接渲染：使用 render 函数 -->
          <CellRenderer v-else :vnode="getCellContent(field, row[field.code]).content" />
        </template>
      </el-table-column>

      <!-- 操作列 -->
      <el-table-column 
        v-if="hasDeleteCallback || linkFields.length > 0" 
        label="操作" 
        fixed="right" 
        :width="getActionColumnWidth()"
        class-name="action-column"
      >
        <template #default="{ row }">
          <TableActionColumn
            :link-fields="linkFields"
            :has-delete-callback="hasDeleteCallback"
            :row="row"
            :user-info-map="userInfoMap"
            @link-click="handleLinkClick"
            @delete="handleDelete"
          />
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页 -->
    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>

    <!-- 新增/编辑对话框 -->
    <FormDialog
      v-model="dialogVisible"
      :title="dialogTitle"
      :fields="props.functionData.response"
      :mode="dialogMode"
      :router="props.functionData.router"
      :method="props.functionData.method"
      :initial-data="currentRow"
      :user-info-map="userInfoMap"
      @submit="handleDialogSubmit"
    />

    <!-- 🔥 详情抽屉 -->
    <!-- 通过配置切换使用分组布局或原布局 -->
    <TableDetailDrawerGrouped
      v-if="useGroupedDetailLayout"
      :function-data="props.functionData"
      :current-function="props.currentFunction"
      :table-data="tableData"
      :visible-fields="visibleFields"
      :id-field="idField"
      :link-fields="linkFields"
      :has-update-callback="hasUpdateCallback"
      :user-info-map="userInfoMap"
      :on-update="handleUpdateRow"
      :on-refresh="loadTableData"
      :on-toggle-layout="toggleDetailLayout"
      ref="tableDetailDrawerRef"
    />
    <TableDetailDrawer
      v-else
      :function-data="props.functionData"
      :current-function="props.currentFunction"
      :table-data="tableData"
      :visible-fields="visibleFields"
      :id-field="idField"
      :link-fields="linkFields"
      :has-update-callback="hasUpdateCallback"
      :user-info-map="userInfoMap"
      :on-update="handleUpdateRow"
      :on-refresh="loadTableData"
      :on-toggle-layout="toggleDetailLayout"
      ref="tableDetailDrawerRef"
    />

  </div>
</template>

<script setup lang="ts">
// 设置组件名称，用于 keep-alive 缓存
defineOptions({
  name: 'TableRenderer'
})

/**
 * TableRenderer - 表格渲染器组件（旧架构）
 * 
 * ⚠️ 注意：这是旧架构的组件，已被新架构替代
 * - 新架构使用：TableView.vue + WorkspaceDetailDrawer.vue
 * - 此组件保留作为备用，但新功能应在新架构中实现
 * 
 * 设计原则：
 * 1. **依赖倒置**：依赖 Widget 抽象接口，不依赖具体实现
 * 2. **组件自治**：每个 Widget 负责自己的表格展示逻辑（renderTableCell）
 * 3. **一致性**：详情展示使用 Widget.render()，与 Form 渲染一致
 * 4. **扩展性**：新增组件时，只需实现 Widget 方法，无需修改 TableRenderer
 * 
 * 功能特性：
 * - 搜索、排序、分页
 * - CRUD 操作（新增、编辑、删除）
 * - 详情查看（点击 ID 列）
 * - 记录导航（上一个/下一个）
 */

import { computed, ref, watch, h, nextTick, onMounted, onUpdated, onUnmounted, isVNode, defineComponent } from 'vue'
import { Plus, Delete } from '@element-plus/icons-vue'
import { ElIcon, ElButton, ElMessage, ElNotification, ElMessageBox, ElTable } from 'element-plus'
import { formatTimestamp } from '@/utils/date'
import { useTableOperations } from '@/composables/useTableOperations'
import { widgetComponentFactory } from '@/core/factories-v2'
import { ErrorHandler } from '@/core/utils/ErrorHandler'
import { Logger } from '@/core/utils/logger'
import { convertToFieldValue } from '@/utils/field'
import { resolveWorkspaceUrl } from '@/utils/route'
import { WidgetType } from '@/core/constants/widget'
import { useUserInfoStore } from '@/stores/userInfo'
import { collectAllUsernames, collectFilesUploadUsersFromRow } from '@/utils/tableUserInfo'
import { getSortableConfig } from '@/utils/fieldSort'
import { useRouter, useRoute } from 'vue-router'
import { TABLE_PARAM_KEYS, SEARCH_PARAM_KEYS } from '@/utils/urlParams'
import FormDialog from './FormDialog.vue'
import { renderTableCell } from '@/core/utils/tableCellRenderer'
import TableDetailDrawer from './TableDetailDrawer.vue'
import TableDetailDrawerGrouped from './TableDetailDrawerGrouped.vue'
import TableActionColumn from './TableActionColumn.vue'
import TableSearchBar from './TableSearchBar.vue'
import TableSortBar from './TableSortBar.vue'
import type { Function as FunctionType, ServiceTree } from '@/types'
import type { FieldConfig, FieldValue, FunctionDetail } from '@/core/types/field'

const router = useRouter()
const route = useRoute()

interface Props {
  /** 函数配置数据 */
  functionData: FunctionType
  /** 当前函数节点（来自 ServiceTree） */
  currentFunction?: ServiceTree
}

const props = defineProps<Props>()

// ==================== 详情布局配置 ====================

/**
 * 是否使用分组布局的详情页面
 * 默认使用新布局，可以通过切换按钮或 localStorage 控制
 */
const getInitialLayout = (): boolean => {
  try {
    // 优先从 localStorage 读取用户设置
    const stored = localStorage.getItem('useGroupedDetailLayout')
    const layoutVersion = localStorage.getItem('useGroupedDetailLayoutVersion')
    
    // 如果用户明确设置了布局且有版本标记，使用用户设置
    if (stored === 'true' || stored === 'false') {
      if (layoutVersion) {
        // 有版本标记，说明是用户明确的选择，使用用户设置
        return stored === 'true'
      } else {
        // 没有版本标记，说明是旧的设置，清除它
        localStorage.removeItem('useGroupedDetailLayout')
      }
    }
    
    // 默认使用新布局
    return true
  } catch (error) {
    console.error('[TableRenderer] 读取布局设置失败:', error)
    // 出错时默认使用新布局
    return true
  }
}
const useGroupedDetailLayout = ref<boolean>(getInitialLayout())

// 监听布局变化
watch(useGroupedDetailLayout, (newVal: boolean) => {
  // 布局状态变化时更新 localStorage
  localStorage.setItem('useGroupedDetailLayout', String(newVal))
  localStorage.setItem('useGroupedDetailLayoutVersion', '1.0')
}, { immediate: false })

/**
 * 切换详情布局
 */
const toggleDetailLayout = (): void => {
  // 保存当前详情状态（如果详情已打开）
  const savedState = currentDetailState.value
  
  // 切换布局
  useGroupedDetailLayout.value = !useGroupedDetailLayout.value
  localStorage.setItem('useGroupedDetailLayout', String(useGroupedDetailLayout.value))
  // 设置版本标记，表示这是用户明确的选择
  localStorage.setItem('useGroupedDetailLayoutVersion', '1.0')
  
  // 如果当前有打开的详情，需要重新打开以应用新布局
  if (savedState) {
    // 先关闭当前详情（如果 ref 还存在）
    if (tableDetailDrawerRef.value) {
      try {
        ;(tableDetailDrawerRef.value as any).handleDetailDrawerClose()
      } catch (e) {
        // 忽略错误，组件可能已经销毁
      }
    }
    
    // 等待组件切换后重新打开详情
    nextTick(async () => {
      if (tableDetailDrawerRef.value && savedState) {
        await (tableDetailDrawerRef.value as any).handleShowDetail(savedState.row, savedState.index)
      }
    })
  }
}

// ==================== 使用 Composable（业务逻辑层） ====================

/**
 * 🔥 使用 useTableOperations 管理所有业务逻辑
 * 
 * 优势：
 * - 业务逻辑可复用
 * - 易于单元测试
 * - TableRenderer 只负责 UI 渲染
 */
const {
  // 状态
  loading,
  tableData,
  searchForm,
  currentPage,
  pageSize,
  total,
  
  // 计算属性
  searchableFields,
  visibleFields,
  hasAddCallback,
  hasUpdateCallback,
  hasDeleteCallback,
  
  // 方法
  loadTableData,
  handleSearch,
  handleReset,
  handleSortChange: originalHandleSortChange,
  getFieldSortOrder,
  sorts,
  hasManualSort,
  handleSizeChange,
  handleCurrentChange,
  handleAdd: handleAddRow,
  handleUpdate: handleUpdateRow,
  handleDelete: handleDeleteRow,
  restoreFromURL,
  syncToURL
} = useTableOperations({
  functionData: props.functionData
})

// ==================== 链接处理（已移至 TableActionColumn 组件） ====================

/**
 * 处理链接点击（用于事件传递）
 */
const handleLinkClick = (fieldCode: string, row: any): void => {
  // TableActionColumn 组件内部已经处理了链接点击逻辑
  // 这里只是事件传递，如果需要额外处理可以在这里添加
}

// ==================== 排序相关 ====================

/**
 * 获取第一个排序配置（用于 el-table 的 default-sort）
 * 
 * ⚠️ 关键：Element Plus 的 el-table 的 default-sort 只能在表格级别设置一个
 * 所以只能显示第一个排序字段的排序标识
 * 
 * @returns default-sort 配置对象，如果没有排序则返回 undefined
 */
const getFirstSortConfig = () => {
  if (sorts.value.length === 0) {
    // 如果没有手动排序，使用默认的 id 降序
    if (idField.value && !hasManualSort.value) {
      return {
        prop: idField.value.code,
        order: 'descending' as const
      }
    }
    return undefined
  }
  
  // 返回第一个排序字段的配置
  const firstSort = sorts.value[0]
  return {
    prop: firstSort.field,
    order: (firstSort.order === 'asc' ? 'ascending' : 'descending') as 'ascending' | 'descending'
  }
}

// 导出 handleSortChange 供模板使用
// 🔥 包装 handleSortChange，确保在排序变化后 DOM 能正确更新
const handleSortChange = async (sortInfo: { prop?: string; order?: string }) => {
  originalHandleSortChange(sortInfo)
  // 使用 nextTick 确保 DOM 更新
  await nextTick()
}

/**
 * 🔥 排序状态映射（计算属性，确保响应式）
 * 
 * 在 custom 模式下，需要为每个列设置 sort-order 来显示排序状态
 * 使用计算属性确保当 sorts 变化时，所有列的排序状态都会更新
 * 
 * ⚠️ 关键：使用对象而不是 Map，确保 Vue 能正确追踪响应式依赖
 */
const sortOrderMap = computed<Record<string, 'ascending' | 'descending' | null>>(() => {
  const map: Record<string, 'ascending' | 'descending' | null> = {}
  sorts.value.forEach((sort: { field: string; order: 'asc' | 'desc' }) => {
    map[sort.field] = sort.order === 'asc' ? 'ascending' : 'descending'
  })
  return map
})

/**
 * 获取字段的排序状态（用于模板）
 * 
 * ⚠️ 关键：直接访问计算属性，确保响应式更新
 * 
 * @param fieldCode 字段 code
 * @returns 排序方向：'ascending' | 'descending' | null
 */
const getSortOrder = (fieldCode: string): 'ascending' | 'descending' | null => {
  return sortOrderMap.value[fieldCode] || null
}

// ==================== 排序信息条相关 ====================

/**
 * 🔥 显示排序列表（用于排序信息条）
 * 
 * 包含所有手动排序的字段，如果没有手动排序且存在 ID 字段，则显示默认的 ID 排序
 */
const displaySorts = computed(() => {
  if (sorts.value.length > 0) {
    return sorts.value
  }
  // 如果没有手动排序且存在 ID 字段，显示默认的 ID 排序
  if (idField.value && !hasManualSort.value) {
    return [{ field: idField.value.code, order: 'desc' as const }]
  }
  return []
})

/**
 * 获取字段名称
 * @param fieldCode 字段 code
 * @returns 字段名称
 */
const getFieldName = (fieldCode: string): string => {
  const field = visibleFields.value.find((f: FieldConfig) => f.code === fieldCode)
  return field?.name || fieldCode
}

/**
 * 移除单个排序
 * @param fieldCode 字段 code
 */
const handleRemoveSort = (fieldCode: string): void => {
  // 调用 composable 的 handleSortChange，传入空 order 来移除排序
  originalHandleSortChange({ prop: fieldCode, order: '' })
}

/**
 * 清除所有排序
 */
const handleClearAllSorts = (): void => {
  // 逐个移除所有排序
  const fieldsToRemove = [...sorts.value]
  fieldsToRemove.forEach(sort => {
    originalHandleSortChange({ prop: sort.field, order: '' })
  })
}

// ==================== 详情抽屉 ====================

/** TableDetailDrawer 组件引用（兼容两种组件） */
const tableDetailDrawerRef = ref<InstanceType<typeof TableDetailDrawer> | InstanceType<typeof TableDetailDrawerGrouped>>()

/** 当前详情状态（用于布局切换时保存状态） */
const currentDetailState = ref<{ row: any; index: number } | null>(null)

/**
 * 显示详情
 * 通过 ref 调用 TableDetailDrawer 的方法
 */
const handleShowDetail = async (row: any, index: number): Promise<void> => {
  // 保存当前详情状态
  currentDetailState.value = { row, index }
  // TableDetailDrawer 内部使用 useTableDetail 管理状态
  // 通过 ref 调用内部方法
  if (tableDetailDrawerRef.value) {
    await tableDetailDrawerRef.value.handleShowDetail(row, index)
  }
}

// ==================== 用户信息批量查询优化 ====================

const userInfoStore = useUserInfoStore()

/** 用户信息映射（username -> UserInfo） */
const userInfoMap = ref<Map<string, any>>(new Map())

/**
 * 🔥 批量查询用户信息
 * 统一收集表格数据和搜索表单中的用户，使用 store 批量查询（自动处理缓存）
 */
async function batchLoadUserInfo(): Promise<void> {
  try {
    // 🔥 使用工具函数收集所有用户名
    const allUsernames = collectAllUsernames(
      tableData.value || [],
      searchForm.value,
      visibleFields.value,
      searchableFields.value
    )
    
    if (allUsernames.length === 0) {
      userInfoMap.value = new Map()
      return
    }
    
    // 🔥 使用 store 统一批量查询（自动处理缓存和过期）
    const users = await userInfoStore.batchGetUserInfo(allUsernames)
    
    // 🔥 构建映射（供表格渲染使用）
    const map = new Map<string, any>()
    for (const user of users) {
      if (user.username) {
        map.set(user.username, user)
      }
    }
    
    userInfoMap.value = map
  } catch (error) {
    Logger.error('TableRenderer', '批量查询用户信息失败', error)
    userInfoMap.value = new Map()
  }
}

// 监听 tableData 变化，自动批量查询用户信息
watch(() => tableData.value, () => {
  if (tableData.value && tableData.value.length > 0) {
    batchLoadUserInfo()
  } else {
    userInfoMap.value = new Map()
  }
}, { immediate: true, deep: false })

// 🔥 监听搜索表单变化，提前查询搜索表单中的用户信息
// 这样可以确保搜索表单中的用户信息在 UserSearchInput 初始化前就已经查询完成
// 避免 UserSearchInput 重复查询
watch(() => searchForm.value, () => {
  // 延迟执行，避免在 searchForm 初始化时立即触发
  nextTick(() => {
    const hasUserFields = searchableFields.value.some((field: FieldConfig) => 
      field.widget?.type === 'user' && searchForm.value[field.code]
    )
    if (hasUserFields) {
      batchLoadUserInfo()
    }
  })
}, { deep: true, immediate: false })

// ==================== 批量选择相关 ====================

/** 选中的行数据 */
const selectedRows = ref<any[]>([])

/** 表格引用（用于控制复选框状态） */
const tableRef = ref<InstanceType<typeof ElTable> | null>(null)

/**
 * 处理选择变化
 * @param selection 选中的行数组
 */
const handleSelectionChange = (selection: any[]): void => {
  selectedRows.value = selection
}

/**
 * 判断行是否可选
 * @param row 行数据
 * @param index 行索引
 * @returns 是否可选
 */
const checkSelectable = (row: Record<string, any>, index: number): boolean => {
  // 所有行都可以选择
  return true
}

/**
 * 批量删除
 */
const handleBatchDelete = async (): Promise<void> => {
  if (selectedRows.value.length === 0) {
    ElMessage.warning('请先选择要删除的记录')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedRows.value.length} 条记录吗？`,
      '批量删除确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    // 获取所有选中行的 ID
    const ids = selectedRows.value
      .map((row: Record<string, any>) => {
        // 尝试从 id 字段获取，如果没有则尝试从 idField 获取
        if (row.id) return row.id
        if (idField.value && row[idField.value.code]) {
          return row[idField.value.code]
        }
        return null
      })
      .filter((id: any): id is number => id !== null && typeof id === 'number')

    if (ids.length === 0) {
      ElMessage.error('无法获取记录 ID，删除失败')
      return
    }

    // 调用批量删除 API
    const { tableDeleteRows } = await import('@/api/function')
    await tableDeleteRows(props.functionData.method, props.functionData.router, ids)

    // 显示成功提示
    ElNotification({
      title: '删除成功',
      message: `已成功删除 ${ids.length} 条记录`,
      type: 'success',
      duration: 3000,
      position: 'top-right'
    })

    // 清空选择
    selectedRows.value = []
    if (tableRef.value) {
      tableRef.value.clearSelection()
    }

    // 重新加载数据
    await loadTableData()
  } catch (error: any) {
    if (error !== 'cancel') {
      const errorMessage = error?.response?.data?.msg || error?.message || '批量删除失败'
      ElNotification({
        title: '删除失败',
        message: errorMessage,
        type: 'error',
        duration: 5000,
        position: 'top-right'
      })
    }
  }
}

// ==================== 对话框相关 ====================

/** 对话框显示状态 */
const dialogVisible = ref(false)

/** 对话框模式（新增/编辑） */
const dialogMode = ref<'create' | 'update'>('create')

/** 对话框标题 */
const dialogTitle = computed(() => dialogMode.value === 'create' ? '新增' : '编辑')

/** 当前编辑的行数据 */
const currentRow = ref<Record<string, any>>({})

// ==================== 字段计算属性 ====================

/**
 * ID 字段（用于控制中心列）
 */
const idField = computed(() => {
  return props.functionData.response.find((field: FieldConfig) => field.widget?.type === 'ID')
})

/**
 * Link 字段（用于操作区域）
 */
const linkFields = computed(() => {
  return visibleFields.value.filter((field: FieldConfig) => field.widget?.type === 'link')
})

/**
 * 数据字段（排除ID列和Link列，ID列已单独作为控制中心列，Link列在操作区域显示）
 */
const dataFields = computed(() => {
  return visibleFields.value.filter((field: FieldConfig) => 
    field.widget?.type !== 'ID' && field.widget?.type !== 'link'
  )
})

// ==================== UI 辅助方法 ====================

/**
 * 获取操作列宽度
 * 根据是否有删除回调动态计算宽度
 */
/**
 * 获取操作列宽度
 * 根据是否有删除回调和链接字段动态计算宽度
 * 🔥 超过 1 个链接时使用下拉菜单，减少操作列宽度
 */
const getActionColumnWidth = (): number => {
  let width = 60  // 基础宽度
  if (hasDeleteCallback.value) width += 60  // 删除按钮宽度（确保"删除"文字完整显示）
  
  // 🔥 只有 1 个链接时直接显示，超过 1 个时使用下拉菜单
  if (linkFields.value.length === 1) {
    // 单个链接约 80px（文本 + 图标 + 间距）
    width += 80
  } else if (linkFields.value.length > 1) {
    // 多个链接使用下拉菜单，只需要一个按钮宽度
    width += 50  // 下拉菜单按钮宽度（"链接"按钮）
  }
  
  // 限制最大宽度，防止变形，但确保删除按钮能完整显示
  return Math.min(Math.max(width, 140), 200)  // 最小 140px，最大 200px（减少最大宽度）
}

/**
 * 获取列宽度
 * 根据字段类型返回合适的列宽
 */
const getColumnWidth = (field: FieldConfig): number => {
  if (field.widget.type === WidgetType.TIMESTAMP) return 180
  if (field.widget.type === WidgetType.TEXT_AREA) return 300
  if (field.widget.type === WidgetType.RATE) {
    // Rate 组件：根据 max 值计算宽度
    const max = field.widget?.config?.max || 5
    if (max > 10) {
      // 圆点样式：更紧凑，但需要显示数字
      // 每个圆点 4px + 间距 1px = 5px，加上数字约 40px
      return Math.max(150, max * 5 + 40)
    } else {
      // 星星样式：每个星星约 14px + 间距 1px = 15px，加上文字约 60px
      return Math.max(150, max * 15 + 60)
    }
  }
  return 150
}

// 注意：isIdColumn 方法已移除，改用 idField computed 和单独的控制中心列

// ==================== 搜索表单相关（已移至 TableSearchBar 组件） ====================

// ==================== 表格单元格渲染（组件自治） ====================

/**
 * 🔥 获取表格单元格内容（用于模板）
 * 
 * 使用共享的 renderTableCell 函数，确保与 TableWidget 渲染逻辑一致
 * 
 * 设计优势：
 * - 符合依赖倒置原则：TableRenderer 依赖 Widget 抽象接口
 * - 扩展性强：新增组件只需实现 table-cell 模式，无需修改 TableRenderer
 * - 展示一致：组件自己决定如何展示，如 FileWidget 显示文件图标、MultiSelectWidget 显示标签
 * - 代码复用：与 TableWidget 使用相同的渲染逻辑，减少重复代码
 * 
 * @param field 字段配置
 * @param rawValue 原始值（来自后端）
 * @returns { content: string | VNode, isString: boolean } - 统一返回格式，方便模板处理
 */
const getCellContent = (field: FieldConfig, rawValue: any): { content: any, isString: boolean } => {
  return renderTableCell(field, rawValue, {
    mode: 'table-cell',
    userInfoMap: userInfoMap.value,
    fieldPath: field.code
  })
}

// 🔥 VNode 渲染组件（用于在模板中渲染 VNode，避免循环引用）
const CellRenderer = defineComponent({
  props: {
    vnode: {
      type: Object,
      required: true
    }
  },
  setup(props: { vnode: any }) {
    return () => props.vnode
  }
})

// ==================== 详情字段渲染（已移至 TableDetailDrawer 组件） ====================

// ==================== CRUD 操作 ====================

/**
 * 新增记录
 * 打开对话框，模式设为 'create'
 */
const handleAdd = (): void => {
  dialogMode.value = 'create'
  currentRow.value = {}
  dialogVisible.value = true
}

/**
 * 编辑记录（已废弃，现在在详情抽屉中直接编辑）
 * 保留此函数以防其他地方调用，但不再使用
 * @deprecated 使用详情抽屉中的编辑功能
 */
const handleEdit = (row: any): void => {
  // 现在编辑功能在详情抽屉中，这里不再使用
  // 如果点击了编辑，直接打开详情抽屉
  const index = tableData.value.findIndex((r: any) => r.id === row.id)
  if (index >= 0) {
    handleShowDetail(row, index)
  }
}

/**
 * 删除记录
 * 调用 composable 的删除方法
 * @param row 要删除的行数据
 */
const handleDelete = async (row: any): Promise<void> => {
  await handleDeleteRow(row.id)
}

/**
 * 对话框提交
 * 根据模式调用新增或更新方法
 * @param data 表单数据
 */
const handleDialogSubmit = async (data: Record<string, any>): Promise<void> => {
  let success = false
  
  if (dialogMode.value === 'create') {
    success = await handleAddRow(data)
  } else {
    // ⚠️ 关键：传递旧值（currentRow.value），用于对比找出变更的字段
    success = await handleUpdateRow(currentRow.value.id, data, currentRow.value)
  }
  
  if (success) {
    // 关闭对话框
    dialogVisible.value = false
  }
}

// ==================== 详情抽屉操作（已移至 TableDetailDrawer 组件） ====================

// ==================== 监听函数变化 ====================

/**
 * 监听函数配置变化
 * 当函数配置更新时，重新加载数据
 * 
 * 🔥 注意：不设置 immediate: true，因为 useTableOperations 的 initialize() 已经会在初始化时调用 loadTableData()
 * 如果设置 immediate: true，会导致初始化时调用两次 loadTableData()
 */
watch(() => props.functionData, () => {
  // 🔥 清空搜索表单，确保没有残留值
  // 先清空所有属性，避免对象引用残留
  Object.keys(searchForm.value).forEach(key => {
    delete searchForm.value[key]
  })
  currentPage.value = 1
  
  // 🔥 清理 URL 中不属于当前函数的搜索参数
  // 获取当前函数的所有字段 code
  const currentFieldCodes = new Set<string>()
  if (Array.isArray(props.functionData.request)) {
    props.functionData.request.forEach((field: FieldConfig) => {
      currentFieldCodes.add(field.code)
    })
  }
  if (Array.isArray(props.functionData.response)) {
    props.functionData.response.forEach((field: FieldConfig) => {
      currentFieldCodes.add(field.code)
    })
  }
  
  // 清理 URL 中不属于当前函数的参数
  const query = router.currentRoute.value.query
  const newQuery: Record<string, string> = {}
  
  // 只保留属于当前函数的参数和通用参数（page, page_size, sorts）
  Object.keys(query).forEach(key => {
    if (TABLE_PARAM_KEYS.includes(key as any)) {
      // 保留分页和排序参数
      newQuery[key] = String(query[key])
    } else if (SEARCH_PARAM_KEYS.includes(key as any)) {
      // 对于搜索参数（eq, like, in 等），需要解析并过滤字段
      const value = String(query[key])
      const parts = value.split(',')
      const filteredParts: string[] = []
      
      for (const part of parts) {
        const colonIndex = part.indexOf(':')
        if (colonIndex > 0) {
          const fieldCode = part.substring(0, colonIndex).trim()
          if (currentFieldCodes.has(fieldCode)) {
            filteredParts.push(part.trim())
          }
        }
      }
      
      if (filteredParts.length > 0) {
        newQuery[key] = filteredParts.join(',')
      }
    } else if (currentFieldCodes.has(key)) {
      // 保留属于当前函数的 request 字段参数
      newQuery[key] = String(query[key])
    }
    // 其他参数（不属于当前函数的）都会被忽略
  })
  
  // 更新 URL（清理不属于当前函数的参数）
  if (Object.keys(newQuery).length !== Object.keys(query).length || 
      Object.keys(newQuery).some(key => query[key] !== newQuery[key])) {
    router.replace({ query: newQuery }).then(() => {
      // 🔥 URL 更新后，从 URL 恢复状态（只恢复属于当前函数的参数）
      restoreFromURL()
      loadTableData()
    })
  } else {
    // 🔥 如果 URL 没有变化，直接恢复状态
    restoreFromURL()
    loadTableData()
  }
})

// ==================== 修复 fixed 列按钮点击问题 ====================

/**
 * 修复 fixed 列按钮在窗口缩小时无法点击的问题
 * 通过强制设置 fixed 列的 pointer-events 和 z-index
 */
const fixFixedColumnClick = () => {
  nextTick(() => {
    // 查找所有 fixed 列的操作按钮
    const fixedRight = document.querySelector('.el-table__fixed-right')
    if (fixedRight) {
      // 强制设置样式
      const fixedElement = fixedRight as HTMLElement
      fixedElement.style.zIndex = '2000'
      fixedElement.style.pointerEvents = 'auto'
      
      // 确保所有按钮可点击
      const buttons = fixedElement.querySelectorAll('.el-button')
      buttons.forEach(btn => {
        const button = btn as HTMLElement
        button.style.pointerEvents = 'auto'
        button.style.zIndex = '2005'
        button.style.position = 'relative'
        button.style.cursor = 'pointer'
      })
    }
  })
}

// ==================== 详情抽屉相关逻辑（已移至 TableDetailDrawer 组件和 useTableDetail composable） ====================

onMounted(() => {
  fixFixedColumnClick()
  // 监听窗口大小变化
  window.addEventListener('resize', fixFixedColumnClick)
})

onUpdated(() => {
  fixFixedColumnClick()
})

onUnmounted(() => {
  // 移除事件监听
  window.removeEventListener('resize', fixFixedColumnClick)
})
</script>

<style scoped>
.table-renderer {
  padding: 20px;
  background: var(--el-bg-color);
  position: relative;
  display: flex;
  flex-direction: column;
  /* 🔥 不设置固定高度，让内容自然流动，支持整体滚动 */
  width: 100%;
  /* 🔥 移除高度限制，让内容可以超出容器 */
}

/* 🔥 表格容器：在小屏幕下，让整个页面滚动而不是表格内部滚动 */
.table-renderer :deep(.el-table__body-wrapper) {
  /* 🔥 移除内部滚动，让整个页面滚动 */
  overflow: visible !important;
  max-height: none !important;
}

/* 文件表格单元格样式 */
:deep(.files-table-cell-wrapper) {
  position: relative;
}

:deep(.files-table-cell) {
  min-width: 0;
}

:deep(.file-item-clickable) {
  user-select: none;
}

:deep(.file-item-clickable:hover) {
  background-color: var(--el-fill-color) !important;
}

.toolbar {
  margin-bottom: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 0;
}

.toolbar-left {
  display: flex;
  gap: 12px;
  align-items: center;
}

.search-bar {
  margin-bottom: 20px;
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

/* 排序信息条样式（已移至 TableSortBar 组件） */

.search-form {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}

.pagination-wrapper {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

/* 🔥 表格基础样式：背景色和边框 */
:deep(.el-table) {
  background-color: var(--el-bg-color) !important;
  border: none !important;
}

:deep(.el-table__inner-wrapper) {
  border: none !important;
}

:deep(.el-table__header-wrapper) {
  border: none !important;
}

/* 🔥 表格 body-wrapper 的边框样式（滚动由外层容器处理） */
:deep(.el-table__body-wrapper) {
  border: none !important;
  /* 注意：滚动由外层 .tab-content 容器处理，这里不设置滚动 */
}

:deep(.el-table th),
:deep(.el-table td) {
  border-right: none !important;
  border-left: none !important;
}

:deep(.el-table th:first-child),
:deep(.el-table td:first-child) {
  border-left: none !important;
}

:deep(.el-table th:last-child),
:deep(.el-table td:last-child) {
  border-right: none !important;
}

:deep(.el-table__body tr) {
  background-color: var(--el-bg-color) !important;
}

/* 🔥 移除斑马纹：确保所有行背景色一致 */
:deep(.el-table__body tr.el-table__row--striped) {
  background-color: var(--el-bg-color) !important;
}

:deep(.el-table__body tr.el-table__row--striped td) {
  background-color: var(--el-bg-color) !important;
}

:deep(.el-table__body tr:hover > td) {
  background-color: var(--el-fill-color-light) !important;
}

/* 确保table内的link按钮清晰可见 */
:deep(.el-button.is-link) {
  font-weight: 500 !important;
}

:deep(.el-button.is-link.el-button--primary) {
  color: var(--el-text-color-primary) !important;
}

:deep(.el-button.is-link.el-button--primary:hover) {
  color: var(--el-color-primary) !important;
}

:deep(.el-button.is-link.el-button--danger) {
  color: var(--el-text-color-primary) !important;
}

:deep(.el-button.is-link.el-button--danger:hover) {
  color: var(--el-color-danger) !important;
}

/* 🔥 操作列样式 - 修复 fixed 列按钮点击问题 */
:deep(.action-column) {
  position: relative;
}

:deep(.action-column .cell) {
  position: relative;
  pointer-events: auto;
}

/* 操作列样式（已移至 TableActionColumn 组件） */

/* 详情页面链接区域（已移至 TableDetailDrawer 组件） */

/* 确保 fixed 列的操作按钮可以点击 */
/* 修复非全屏模式下按钮无法点击的问题 */
.table-with-fixed-column {
  position: relative;
}

/* 关键修复：确保 fixed 列及其所有子元素都在最上层且可点击 */
:deep(.el-table__fixed-right) {
  z-index: 2000 !important;
  pointer-events: auto !important;
}

:deep(.el-table__fixed-right *) {
  pointer-events: auto !important;
}

/* fixed 列的所有 wrapper 和容器 */
:deep(.el-table__fixed-right-patch) {
  z-index: 1999 !important;
  pointer-events: none !important; /* 补丁层不拦截事件 */
}

:deep(.el-table__fixed-right .el-table__fixed-body-wrapper) {
  z-index: 2001 !important;
  pointer-events: auto !important;
}

:deep(.el-table__fixed-right .el-table__fixed-header-wrapper) {
  z-index: 2001 !important;
  pointer-events: auto !important;
}

/* 操作列及其内容 */
:deep(.el-table__fixed-right .action-column) {
  z-index: 2002 !important;
  pointer-events: auto !important;
}

:deep(.el-table__fixed-right .action-column .cell) {
  position: relative !important;
  z-index: 2003 !important;
  pointer-events: auto !important;
}

/* action-buttons 样式已移至 TableActionColumn 组件 */

/* 🔥 表格主体样式：确保不会遮挡 fixed 列，并支持整体滚动 */
:deep(.el-table__body-wrapper) {
  z-index: 1 !important;
  position: relative;
  pointer-events: auto !important;
  overflow: visible !important; /* 滚动由外层容器处理 */
  clip-path: none !important; /* 在 fixed 列区域，让点击事件穿透 */
}

:deep(.el-table__body) {
  z-index: 1 !important;
}

/* 表格主体单元格 - 确保它们不会覆盖 fixed 列 */
:deep(.el-table__body-wrapper .el-table__body tr) {
  position: relative;
  z-index: 1 !important;
}

:deep(.el-table__body-wrapper .el-table__body tr td) {
  position: relative;
  z-index: 1 !important;
}

/* 🔥 表格容器样式：确保不会遮挡 fixed 列 */
:deep(.el-table) {
  position: relative;
  z-index: 1;
  overflow: visible !important;
}

:deep(.el-table__inner-wrapper) {
  position: relative;
  z-index: 1;
  overflow: visible !important;
  border: none !important;
}

/* 确保滚动条不会遮挡 */
:deep(.el-scrollbar) {
  z-index: 1 !important;
}

:deep(.el-scrollbar__wrap) {
  z-index: 1 !important;
}

/* 移除 fixed 列的遮罩层（如果有） */
:deep(.el-table__fixed-right::before),
:deep(.el-table__fixed-right::after) {
  display: none !important;
  pointer-events: none !important;
}

/* 详情抽屉样式（已移至 TableDetailDrawer 组件） */
</style>


