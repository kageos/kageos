<template>
  <div class="table-renderer">
    <!-- 工具栏 -->
    <div class="toolbar" v-if="hasAddCallback">
      <el-button type="primary" @click="handleAdd" :icon="Plus">
        新增
      </el-button>
    </div>

    <!-- 搜索栏 -->
    <div class="search-bar">
      <el-form :inline="true" :model="searchForm" class="search-form">
        <template v-for="field in searchableFields" :key="field.code">
          <!-- 🔥 通过 Widget 渲染搜索输入（组件自治） -->
          <el-form-item :label="field.name">
            <SearchInput
              :field="field"
              :search-type="field.search"
              :model-value="getSearchValue(field)"
              @update:model-value="(value: any) => updateSearchValue(field, value)"
            />
          </el-form-item>
        </template>

        <el-form-item>
          <el-button type="primary" @click="handleSearch">
            <el-icon><Search /></el-icon>
            搜索
          </el-button>
          <el-button @click="handleReset">
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 表格 -->
    <el-table
      v-loading="loading"
      :data="tableData"
      border
      style="width: 100%"
      @sort-change="handleSortChange"
    >
      <!-- 🔥 控制中心列（ID列改造） -->
      <el-table-column
        v-if="idField"
        label="操作"
        fixed="left"
        width="100"
        class-name="control-column"
      >
        <template #default="{ row, $index }">
          <el-button
            link
            type="primary"
            size="small"
            @click="handleShowDetail(row, $index)"
            class="detail-button"
          >
            <el-icon><View /></el-icon>
            <span class="button-text">#{{ row[idField.code] }}</span>
          </el-button>
        </template>
      </el-table-column>

      <!-- 数据列（排除ID列） -->
      <el-table-column
        v-for="field in dataFields"
        :key="field.code"
        :prop="field.code"
        :label="field.name"
        :sortable="field.search ? 'custom' : false"
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
          <component 
            v-else
            :is="getCellContent(field, row[field.code]).content"
          />
        </template>
      </el-table-column>

      <!-- 操作列 -->
      <el-table-column 
        v-if="hasUpdateCallback || hasDeleteCallback" 
        label="操作" 
        fixed="right" 
        :width="getActionColumnWidth()"
      >
        <template #default="{ row }">
          <el-button 
            v-if="hasUpdateCallback"
            link 
            type="primary" 
            size="small"
            @click="handleEdit(row)"
          >
            <el-icon><Edit /></el-icon>
            编辑
          </el-button>
          <el-button 
            v-if="hasDeleteCallback"
            link 
            type="danger" 
            size="small"
            @click="handleDelete(row)"
          >
            <el-icon><Delete /></el-icon>
            删除
          </el-button>
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
      :initial-data="currentRow"
      @submit="handleDialogSubmit"
    />

    <!-- 🔥 详情抽屉 -->
    <el-drawer
      v-model="showDetailDrawer"
      title="记录详情"
      direction="rtl"
      size="600px"
      class="detail-drawer"
    >
      <template #header>
        <div class="drawer-header">
          <span class="drawer-title">记录详情</span>
          <!-- 导航按钮（上一个/下一个） -->
          <div class="drawer-navigation" v-if="tableData.length > 1">
            <el-button
              size="small"
              :disabled="currentDetailIndex <= 0"
              @click="handleNavigate('prev')"
            >
              <el-icon><ArrowLeft /></el-icon>
              上一个
            </el-button>
            <span class="nav-info">{{ currentDetailIndex + 1 }} / {{ tableData.length }}</span>
            <el-button
              size="small"
              :disabled="currentDetailIndex >= tableData.length - 1"
              @click="handleNavigate('next')"
            >
              下一个
              <el-icon><ArrowRight /></el-icon>
            </el-button>
          </div>
        </div>
      </template>

      <!-- 🔥 详情内容：使用 Widget 的 render() 方法（与 Form 一致） -->
      <div class="detail-content" v-if="currentDetailRow">
        <el-descriptions :column="1" border>
          <el-descriptions-item
            v-for="field in visibleFields"
            :key="field.code"
            :label="field.name"
          >
            <!-- 🔥 使用 Widget 的 render() 方法渲染详情（只读模式） -->
            <!-- 这样详情展示与 Form 完全一致，组件可以自定义详情展示效果 -->
            <component 
              :is="renderDetailField(field, currentDetailRow[field.code])"
            />
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
/**
 * TableRenderer - 表格渲染器组件
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

import { computed, ref, watch, h } from 'vue'
import { Search, Refresh, Edit, Delete, Plus, ArrowLeft, ArrowRight, View } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useTableOperations } from '@/composables/useTableOperations'
import { WidgetBuilder } from '@/core/factories/WidgetBuilder'
import { ErrorHandler } from '@/core/utils/ErrorHandler'
import { convertToFieldValue } from '@/utils/field'
import FormDialog from './FormDialog.vue'
import SearchInput from './SearchInput.vue'
import type { Function as FunctionType } from '@/types'
import type { FieldConfig, FieldValue } from '@/core/types/field'

interface Props {
  /** 函数配置数据 */
  functionData: FunctionType
}

const props = defineProps<Props>()

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
  sortField,
  sortOrder,
  
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
  handleSortChange,
  handleSizeChange,
  handleCurrentChange,
  handleAdd: handleAddRow,
  handleUpdate: handleUpdateRow,
  handleDelete: handleDeleteRow
} = useTableOperations({
  functionData: props.functionData
})

// ==================== 详情抽屉状态 ====================

/** 详情抽屉显示状态 */
const showDetailDrawer = ref(false)

/** 当前详情的行数据 */
const currentDetailRow = ref<any>(null)

/** 当前详情的行索引 */
const currentDetailIndex = ref(-1)

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
 * 数据字段（排除ID列，ID列已单独作为控制中心列）
 */
const dataFields = computed(() => {
  return visibleFields.value.filter((field: FieldConfig) => field.widget?.type !== 'ID')
})

// ==================== UI 辅助方法 ====================

/**
 * 获取操作列宽度
 * 根据是否有编辑/删除回调动态计算宽度
 */
const getActionColumnWidth = (): number => {
  let width = 80
  if (hasUpdateCallback.value) width += 60
  if (hasDeleteCallback.value) width += 60
  return width
}

/**
 * 获取列宽度
 * 根据字段类型返回合适的列宽
 */
const getColumnWidth = (field: FieldConfig): number => {
  if (field.widget.type === 'timestamp') return 180
  if (field.widget.type === 'text_area') return 300
  return 150
}

// 注意：isIdColumn 方法已移除，改用 idField computed 和单独的控制中心列

// ==================== 搜索表单相关 ====================

/**
 * 获取搜索值
 * @param field 字段配置
 * @returns 搜索值
 */
const getSearchValue = (field: FieldConfig): any => {
  return searchForm.value[field.code] || null
}

/**
 * 更新搜索值
 * @param field 字段配置
 * @param value 新的搜索值
 */
const updateSearchValue = (field: FieldConfig, value: any): void => {
  searchForm.value[field.code] = value
}

// ==================== 表格单元格渲染（组件自治） ====================

/**
 * 🔥 渲染表格单元格
 * 
 * 使用 Widget 的 renderTableCell() 方法，实现组件自治
 * 
 * 设计优势：
 * - 符合依赖倒置原则：TableRenderer 依赖 Widget 抽象接口
 * - 扩展性强：新增组件只需实现 renderTableCell()，无需修改 TableRenderer
 * - 展示一致：组件自己决定如何展示，如 FileWidget 显示文件图标、MultiSelectWidget 显示标签
 * 
 * @param field 字段配置
 * @param rawValue 原始值（来自后端）
 * @returns { content: string | VNode, isString: boolean } - 统一返回格式，方便模板处理
 * 
 * @example
 * // FileWidget 可以这样实现：
 * renderTableCell(value: FieldValue) {
 *   return h('div', [
 *     h(ElIcon, { File }),
 *     h('span', `共 ${files.length} 个文件`)
 *   ])
 * }
 */
const renderTableCell = (field: FieldConfig, rawValue: any): { content: any, isString: boolean } => {
  try {
    // 🔥 将原始值转换为 FieldValue 格式
    const value = convertToFieldValue(rawValue, field)
    
    // 🔥 将 field 转换为 core 类型的 FieldConfig（类型兼容）
    const coreField: FieldConfig = {
      ...field,
      widget: field.widget || { type: 'input', config: {} },
      data: field.data || {}
    } as FieldConfig
    
    // 🔥 创建临时 Widget（不需要 formManager）
    const tempWidget = WidgetBuilder.createTemporary({
      field: coreField,
      value: value
    })
    
    // 🔥 调用 Widget 的 renderTableCell() 方法（组件自治）
    // 每个 Widget 可以重写此方法来自定义表格展示
    const result = tempWidget.renderTableCell(value)
    
    // 🔥 统一返回格式：区分字符串和 VNode
    const isString = typeof result === 'string'
    return {
      content: result,
      isString
    }
  } catch (error) {
    // ✅ 使用 ErrorHandler 统一处理错误
    const fallbackValue = rawValue !== null && rawValue !== undefined ? String(rawValue) : '-'
    return {
      content: fallbackValue,
      isString: true
    }
  }
}

/**
 * 🔥 获取表格单元格内容（用于模板）
 * 
 * 这是一个包装函数，用于统一处理字符串和 VNode 返回值
 * 返回格式：{ content, isString }
 */
const getCellContent = (field: FieldConfig, rawValue: any): { content: any, isString: boolean } => {
  return renderTableCell(field, rawValue)
}

// ==================== 详情字段渲染（复用 Form 渲染引擎） ====================

/**
 * 🔥 渲染详情字段
 * 
 * 使用 Widget 的 render() 方法，与 Form 渲染完全一致
 * 
 * 设计优势：
 * - 详情展示与 Form 一致：FileWidget 在详情中也显示文件上传组件（只读）
 * - SelectWidget 在详情中显示 label，而不是 raw 值
 * - 无需重复实现详情渲染逻辑
 * 
 * @param field 字段配置
 * @param rawValue 原始值（来自后端）
 * @returns VNode（Vue 虚拟节点）
 * 
 * @example
 * // FileWidget 的 render() 会自动适配只读模式：
 * render() {
 *   if (readonly) {
 *     return h('div', files.map(file => h(FilePreview, { file })))
 *   }
 *   return h(ElUpload, { ... })
 * }
 */
const renderDetailField = (field: FieldConfig, rawValue: any): any => {
  try {
    // 🔥 将原始值转换为 FieldValue 格式
    const value = convertToFieldValue(rawValue, field)
    
    // 🔥 将 field 转换为 core 类型的 FieldConfig（类型兼容）
    const coreField: FieldConfig = {
      ...field,
      widget: field.widget || { type: 'input', config: {} },
      data: field.data || {}
    } as FieldConfig
    
    // 🔥 创建临时 Widget（只读模式）
    // Widget 的 render() 方法会根据只读模式调整展示
    const tempWidget = WidgetBuilder.createTemporary({
      field: coreField,
      value: value
    })
    
    // 🔥 调用 Widget 的 render() 方法（与 Form 一致）
    // 这样详情展示就与 Form 渲染完全一致了
    return tempWidget.render()
  } catch (error) {
    // ✅ 使用 ErrorHandler 统一处理错误
    return ErrorHandler.handleWidgetError(`TableRenderer.renderDetailField[${field.code}]`, error, {
      showMessage: false,
      fallbackValue: h('span', rawValue !== null && rawValue !== undefined ? String(rawValue) : '-')
    })
  }
}

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
 * 编辑记录
 * 打开对话框，模式设为 'update'，加载当前行数据
 * @param row 要编辑的行数据
 */
const handleEdit = (row: any): void => {
  dialogMode.value = 'update'
  currentRow.value = { ...row }
  dialogVisible.value = true
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
    success = await handleUpdateRow(currentRow.value.id, data)
  }
  
  if (success) {
    // 关闭对话框
    dialogVisible.value = false
  }
}

// ==================== 详情抽屉操作 ====================

/**
 * 显示详情
 * 打开详情抽屉，加载指定行的数据
 * @param row 行数据
 * @param index 行索引
 */
const handleShowDetail = (row: any, index: number): void => {
  currentDetailRow.value = row
  currentDetailIndex.value = index
  showDetailDrawer.value = true
}

/**
 * 导航（上一个/下一个）
 * 在详情抽屉中切换记录
 * @param direction 导航方向
 */
const handleNavigate = (direction: 'prev' | 'next'): void => {
  if (!tableData.value || tableData.value.length === 0) return

  if (direction === 'prev' && currentDetailIndex.value > 0) {
    currentDetailIndex.value--
    currentDetailRow.value = tableData.value[currentDetailIndex.value]
  } else if (direction === 'next' && currentDetailIndex.value < tableData.value.length - 1) {
    currentDetailIndex.value++
    currentDetailRow.value = tableData.value[currentDetailIndex.value]
  }
}

// ==================== 监听函数变化 ====================

/**
 * 监听函数配置变化
 * 当函数配置更新时，重新加载数据
 */
watch(() => props.functionData, () => {
  searchForm.value = {}
  currentPage.value = 1
  loadTableData()
}, { immediate: true })
</script>

<style scoped>
.table-renderer {
  padding: 20px;
  background: var(--el-bg-color);
}

.toolbar {
  margin-bottom: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 0;
}

.search-bar {
  margin-bottom: 20px;
  padding: 20px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

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

/* 确保表格单元格背景色正确 */
:deep(.el-table) {
  background-color: var(--el-bg-color) !important;
}

:deep(.el-table__body tr) {
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

/* 🔥 控制中心列样式 */
:deep(.control-column) {
  .detail-button {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 4px 8px;
    font-size: 13px;
    
    .el-icon {
      font-size: 14px;
    }
    
    .button-text {
      font-weight: 500;
      color: var(--el-color-primary);
    }
    
    &:hover {
      .button-text {
        color: var(--el-color-primary-light-3);
      }
    }
  }
}

/* 🔥 详情抽屉样式 */
.detail-drawer {
  :deep(.el-drawer__header) {
    margin-bottom: 0;
    padding: 20px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .drawer-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
  }

  .drawer-title {
    font-size: 18px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .drawer-navigation {
    display: flex;
    align-items: center;
    gap: 12px;

    .nav-info {
      font-size: 14px;
      color: var(--el-text-color-secondary);
      min-width: 60px;
      text-align: center;
    }
  }

  .detail-content {
    padding: 20px;

    :deep(.el-descriptions) {
      .el-descriptions__label {
        width: 150px;
        background-color: var(--el-fill-color-light);
        font-weight: 500;
      }

      .el-descriptions__content {
        color: var(--el-text-color-primary);
      }
    }
  }
}
</style>
