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
              @update:model-value="(value: any) => {
                // 🔥 判断是否清空：值为 null 或空字符串，且之前有值
                const isClearing = (value === null || value === '') && 
                                   searchForm.value && 
                                   searchForm.value[field.code] !== undefined
                updateSearchValue(field, value, isClearing)
              }"
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
    <!-- 
      ⚠️ 关键：Element Plus 的 el-table 在 custom 模式下，需要手动控制每个列的排序状态
      使用 :key 强制重新渲染，确保排序状态正确显示
      使用 ref 来获取表格实例，以便在排序变化后更新排序状态
    -->
    <el-table
      v-loading="loading"
      :data="tableData"
      border
      style="width: 100%"
      class="table-with-fixed-column"
      :key="`table-${sorts.map((s: any) => `${s.field}:${s.order}`).join(',')}`"
      @sort-change="handleSortChange"
    >
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
        :sort-orders="['descending', 'ascending']"
        :default-sort="getFieldSortOrder(idField.code) || (sorts.length === 0 && !hasManualSort ? 'descending' : null) ? { prop: idField.code, order: getFieldSortOrder(idField.code) || (sorts.length === 0 && !hasManualSort ? 'descending' : null) } : undefined"
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
        :sort-orders="['ascending', 'descending']"
        :default-sort="getFieldSortOrder(field.code) ? { prop: field.code, order: getFieldSortOrder(field.code) } : undefined"
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
        v-if="hasDeleteCallback" 
        label="操作" 
        fixed="right" 
        :width="getActionColumnWidth()"
        class-name="action-column"
      >
        <template #default="{ row }">
          <div class="action-buttons">
            <el-button 
              v-if="hasDeleteCallback"
              link 
              type="danger" 
              size="small"
              @click.stop="handleDelete(row)"
            >
              <el-icon><Delete /></el-icon>
              删除
            </el-button>
          </div>
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
      :initial-data="currentRow"
      :user-info-map="userInfoMap"
      @submit="handleDialogSubmit"
    />

    <!-- 🔥 详情抽屉 -->
    <el-drawer
      v-model="showDetailDrawer"
      title="记录详情"
      direction="rtl"
      size="900px"
      class="detail-drawer"
    >
      <template #header>
        <div class="drawer-header">
          <span class="drawer-title">记录详情</span>
          <div class="drawer-header-actions">
            <!-- 模式切换按钮 -->
            <div class="drawer-mode-actions">
              <el-button
                v-if="detailMode === 'view' && hasUpdateCallback"
                type="primary"
                size="small"
                @click="switchToEditMode"
              >
                <el-icon><Edit /></el-icon>
                编辑
              </el-button>
              <el-button
                v-if="detailMode === 'edit'"
                size="small"
                @click="switchToViewMode"
              >
                取消
              </el-button>
              <el-button
                v-if="detailMode === 'edit'"
                type="primary"
                size="small"
                :loading="detailSubmitting"
                @click="handleDetailSave"
              >
                保存
              </el-button>
            </div>
            <!-- 导航按钮（上一个/下一个） -->
            <div class="drawer-navigation" v-if="tableData.length > 1 && detailMode === 'view'">
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
        </div>
      </template>

      <!-- 🔥 查看模式：纯展示模式，参考旧版本设计 -->
      <div class="detail-content" v-if="currentDetailRow && detailMode === 'view'">
        <div class="fields-grid">
          <div 
            v-for="field in visibleFields"
            :key="field.code"
            class="field-row"
          >
            <div class="field-label">
              {{ field.name }}
            </div>
            <div class="field-value">
              <!-- 复制按钮（hover 时显示） -->
              <div class="field-actions">
                <el-button 
                  type="primary" 
                  size="small" 
                  text 
                  @click="copyFieldValue(field, currentDetailRow[field.code])"
                  class="copy-btn"
                  :title="`复制${field.name}`"
                >
                  <el-icon><DocumentCopy /></el-icon>
                </el-button>
              </div>
              
              <!-- 字段内容 -->
              <div class="field-content">
                <component 
                  :is="renderDetailField(field, currentDetailRow[field.code])"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 🔥 编辑模式：使用 FormRenderer -->
      <div class="edit-content" v-else-if="currentDetailRow && detailMode === 'edit'">
        <FormRenderer
          ref="detailFormRendererRef"
          :function-detail="editFunctionDetail"
          :initial-data="currentDetailRow"
          :user-info-map="userInfoMap"
          :show-submit-button="false"
          :show-reset-button="false"
        />
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

import { computed, ref, watch, h, nextTick, onMounted, onUpdated, onUnmounted, isVNode, defineComponent } from 'vue'
import { Search, Refresh, Edit, Delete, Plus, ArrowLeft, ArrowRight, DocumentCopy, Document, Download } from '@element-plus/icons-vue'
import { ElIcon, ElButton, ElMessage } from 'element-plus'
import { formatTimestamp } from '@/utils/date'
import { useTableOperations } from '@/composables/useTableOperations'
import { widgetComponentFactory } from '@/core/factories-v2'
import { ErrorHandler } from '@/core/utils/ErrorHandler'
import { convertToFieldValue } from '@/utils/field'
import { WidgetType } from '@/core/constants/widget'
import { useUserInfoStore } from '@/stores/userInfo'
import { collectAllUsernames, collectFilesUploadUsersFromRow } from '@/utils/tableUserInfo'
import { getSortableConfig } from '@/utils/fieldSort'
import FormDialog from './FormDialog.vue'
import FormRenderer from '@/core/renderers-v2/FormRenderer.vue'
import SearchInput from './SearchInput.vue'
import type { Function as FunctionType, ServiceTree } from '@/types'
import type { FieldConfig, FieldValue, FunctionDetail } from '@/core/types/field'

interface Props {
  /** 函数配置数据 */
  functionData: FunctionType
  /** 当前函数节点（来自 ServiceTree） */
  currentFunction?: ServiceTree
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

// 导出 handleSortChange 供模板使用
// ⚠️ 关键：Element Plus 的 el-table 在 custom 模式下，排序状态显示需要特殊处理
// 使用 :key 强制重新渲染整个表格，确保所有列的排序状态正确显示
const handleSortChange = (sortInfo: { prop?: string; order?: string }) => {
  originalHandleSortChange(sortInfo)
  // ⚠️ 关键：在排序变化后，使用 nextTick 确保 DOM 更新完成
  // 然后强制更新表格的排序状态显示
  nextTick(() => {
    // Element Plus 的 el-table 在 custom 模式下，排序状态是通过 sort-change 事件控制的
    // 但显示状态需要通过 default-sort 来设置，而 default-sort 只能设置一个
    // 所以我们使用 :key 强制重新渲染整个表格，确保所有列的排序状态正确显示
    // 这里不需要额外操作，因为 :key 已经会触发重新渲染
  })
}

// ==================== 详情抽屉状态 ====================

/** 详情抽屉显示状态 */
const showDetailDrawer = ref(false)

/** 当前详情的行数据 */
const currentDetailRow = ref<any>(null)

/** 当前详情的行索引 */
const currentDetailIndex = ref(-1)

/** 详情模式：查看/编辑 */
const detailMode = ref<'view' | 'edit'>('view')

/** 详情编辑模式的 FormRenderer 引用 */
const detailFormRendererRef = ref<InstanceType<typeof FormRenderer>>()

/** 详情编辑提交状态 */
const detailSubmitting = ref(false)

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
    users.forEach(user => {
        if (user.username) {
          map.set(user.username, user)
        }
      })
    
    userInfoMap.value = map
  } catch (error) {
    console.error('[TableRenderer] ❌ 批量查询用户信息失败:', error)
    userInfoMap.value = new Map()
  }
}

// 监听 tableData 变化，自动批量查询用户信息
watch(() => tableData.value, (newData, oldData) => {
  console.log('[TableRenderer] 🔍 watch tableData 触发', {
    newLength: newData?.length || 0,
    oldLength: oldData?.length || 0,
    timestamp: new Date().toISOString()
  })
  
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
 * 根据是否有删除回调动态计算宽度
 */
const getActionColumnWidth = (): number => {
  let width = 60  // 基础宽度（减小）
  if (hasDeleteCallback.value) width += 50  // 删除按钮宽度（减小）
  return width
}

/**
 * 获取列宽度
 * 根据字段类型返回合适的列宽
 */
const getColumnWidth = (field: FieldConfig): number => {
  if (field.widget.type === WidgetType.TIMESTAMP) return 180
  if (field.widget.type === WidgetType.TEXT_AREA) return 300
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
  const value = searchForm.value[field.code]
  // 🔥 如果值是 undefined，返回 null；否则返回原值（包括空对象、空数组等）
  return value === undefined ? null : value
}

/**
 * 更新搜索值
 * @param field 字段配置
 * @param value 新的搜索值
 * @param shouldSearch 是否自动搜索（默认 false，清空时设为 true）
 */
const updateSearchValue = (field: FieldConfig, value: any, shouldSearch: boolean = false): void => {
  // 🔥 如果值为空（空数组、空字符串、null、undefined），删除该字段
  if (value === null || value === undefined || 
      (Array.isArray(value) && value.length === 0) || 
      (typeof value === 'string' && value.trim() === '')) {
    delete searchForm.value[field.code]
  } else {
    searchForm.value[field.code] = value
  }
  // 🔥 更新搜索值后，同步到 URL
  syncToURL()
  // 🔥 如果需要自动搜索（清空时），触发搜索
  if (shouldSearch) {
    loadTableData()
  }
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
/**
 * 🔥 渲染表格单元格（使用 widgets-v2）
 * 
 * 重构说明：
 * - 按照 v2 的设计思路重新实现
 * - 使用 widgetComponentFactory 获取组件
 * - 使用 h() 渲染组件为 VNode
 * - 统一返回 VNode（不再需要区分字符串和 VNode）
 */
const renderTableCell = (field: FieldConfig, rawValue: any): { content: any, isString: boolean } => {
  try {
    // 🔥 将原始值转换为 FieldValue 格式
    const value = convertToFieldValue(rawValue, field)
    
    // 🔥 使用 widgetComponentFactory 获取组件（v2 方式）
    const WidgetComponent = widgetComponentFactory.getRequestComponent(
      field.widget?.type || 'input'
    )
    
    if (!WidgetComponent) {
      // 如果组件未找到，返回 fallback
      const fallbackValue = rawValue !== null && rawValue !== undefined ? String(rawValue) : '-'
      return {
        content: fallbackValue,
        isString: true
      }
    }
    
    // 🔥 使用 h() 渲染组件为 VNode（v2 方式）
    // 传递 mode="table-cell" 让组件自己决定如何渲染
    // 传递 userInfoMap 用于批量查询优化
    const vnode = h(WidgetComponent, {
      field: field,
      value: value,
      'model-value': value,
      'field-path': field.code,
      mode: 'table-cell',
      'user-info-map': userInfoMap.value
    })
    
    // 🔥 统一返回 VNode（v2 组件统一返回 VNode）
    return {
      content: vnode,
      isString: false
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

// ==================== 详情字段渲染（纯展示模式） ====================

/**
 * 🔥 渲染详情字段（遵循依赖倒置原则）
 * 
 * 设计原则：
 * - 遵循依赖倒置原则：TableRenderer 不需要知道具体 Widget 类型
 * - 组件自治：每个 Widget 自己决定如何在详情中展示
 * - 统一使用 widget.renderForDetail() 方法
 * 
 * @param field 字段配置
 * @param rawValue 原始值（来自后端）
 * @returns 渲染结果（VNode 或字符串）
 */
/**
 * 🔥 渲染详情字段（使用 widgets-v2）
 * 
 * 重构说明：
 * - 按照 v2 的设计思路重新实现
 * - 使用 widgetComponentFactory 获取组件
 * - 使用 h() 渲染组件为 VNode
 * - 统一返回 VNode（v2 组件统一返回 VNode）
 */
const renderDetailField = (field: FieldConfig, rawValue: any): any => {
  try {
    // 🔥 将原始值转换为 FieldValue 格式
    const value = convertToFieldValue(rawValue, field)
    
    // 🔥 使用 widgetComponentFactory 获取组件（v2 方式）
    const WidgetComponent = widgetComponentFactory.getRequestComponent(
      field.widget?.type || 'input'
    )
    
    if (!WidgetComponent) {
      // 如果组件未找到，返回 fallback
      return h('span', rawValue !== null && rawValue !== undefined ? String(rawValue) : '-')
    }
    
    // 🔥 使用 h() 渲染组件为 VNode（v2 方式）
    // 传递 mode="detail" 让组件自己决定如何渲染详情
    // 传递 userInfoMap 用于批量查询优化
    // 传递 functionName 和 recordId 用于 FilesWidget 打包下载命名
    const idField = visibleFields.value.find(f => {
      const code = f.code.toLowerCase()
      return code === 'id' || code === 'ID' || code.endsWith('_id') || code.endsWith('Id')
    })
    const recordId = idField && currentDetailRow.value ? currentDetailRow.value[idField.code] : undefined
    
    // 🔥 从 router 或 currentFunction 获取函数名称、user 和 app 名称
    // router 格式通常是：/user/app/function_name 或 /user/app/group/function_name
    let functionName: string | undefined = undefined
    let userName: string | undefined = undefined
    let appName: string | undefined = undefined
    
    if (props.currentFunction?.name) {
      // 优先使用 currentFunction.name
      functionName = props.currentFunction.name
    } else if (props.functionData?.router) {
      // 从 router 中提取函数名称（取最后一段）
      const routerParts = props.functionData.router.split('/').filter(Boolean)
      if (routerParts.length > 0) {
        functionName = routerParts[routerParts.length - 1]
      }
    }
    
    // 🔥 从 router 中提取 user 和 app 名称（格式：/user/app/...）
    if (props.functionData?.router) {
      const routerParts = props.functionData.router.split('/').filter(Boolean)
      if (routerParts.length >= 1) {
        userName = routerParts[0]  // 第一段是 user 名称
      }
      if (routerParts.length >= 2) {
        appName = routerParts[1]  // 第二段是 app 名称
      }
    }
    
    // 🔥 如果有 user 和 app 名称，在函数名称前面加上
    if (userName && appName && functionName) {
      functionName = `${userName}_${appName}_${functionName}`
    } else if (appName && functionName) {
      // 如果只有 app 名称，也加上
      functionName = `${appName}_${functionName}`
    }
    
    
    return h(WidgetComponent, {
      field: field,
      value: value,
      'model-value': value,
      'field-path': field.code,
      mode: 'detail',
      'user-info-map': userInfoMap.value,
      functionName: functionName,  // 🔥 使用 camelCase，Vue 会自动处理
      recordId: recordId
    })
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
const handleShowDetail = async (row: any, index: number): Promise<void> => {
  currentDetailRow.value = row
  currentDetailIndex.value = index
  detailMode.value = 'view'  // 重置为查看模式
  showDetailDrawer.value = true
  
  // 🔥 收集当前行的 files widget 的 upload_user 并查询用户信息
  const filesUploadUsers = collectFilesUploadUsersFromRow(row, visibleFields.value)
  
  if (filesUploadUsers.length > 0) {
    // 批量查询用户信息（自动处理缓存）
    const users = await userInfoStore.batchGetUserInfo(filesUploadUsers)
    
    // 更新 userInfoMap，供详情中的 FilesWidget 使用
    users.forEach((user: any) => {
      if (user.username) {
        userInfoMap.value.set(user.username, user)
      }
    })
  }
}

/**
 * 导航（上一个/下一个）
 * 在详情抽屉中切换记录
 * @param direction 导航方向
 */
const handleNavigate = async (direction: 'prev' | 'next'): Promise<void> => {
  if (!tableData.value || tableData.value.length === 0) return

  let newIndex = currentDetailIndex.value
  if (direction === 'prev' && newIndex > 0) {
    newIndex--
  } else if (direction === 'next' && newIndex < tableData.value.length - 1) {
    newIndex++
  } else {
    return
  }

  currentDetailIndex.value = newIndex
  const row = tableData.value[newIndex]
  currentDetailRow.value = row
  detailMode.value = 'view'  // 切换记录时，重置为查看模式
  
  // 🔥 收集新行的 files widget 的 upload_user 并查询用户信息
  const filesUploadUsers = collectFilesUploadUsersFromRow(row, visibleFields.value)
  if (filesUploadUsers.length > 0) {
    // 批量查询用户信息（自动处理缓存）
    const users = await userInfoStore.batchGetUserInfo(filesUploadUsers)
    // 更新 userInfoMap，供详情中的 FilesWidget 使用
    users.forEach((user: any) => {
      if (user.username) {
        userInfoMap.value.set(user.username, user)
      }
    })
  }
}

/**
 * 🔥 复制字段值到剪贴板（简化实现）
 * 
 * 重构说明：
 * - v2 组件没有统一的 getCopyText 方法
 * - 简化实现：直接使用 value.display 或 value.raw
 * - 如果后续需要更复杂的复制逻辑，可以在组件内部处理
 * 
 * @param field 字段配置
 * @param value 字段值（原始值）
 */
const copyFieldValue = (field: FieldConfig, value: any): void => {
  try {
    // 🔥 将原始值转换为 FieldValue 格式
    const fieldValue = convertToFieldValue(value, field)
    
    // 🔥 简化实现：优先使用 display，否则使用 raw
    // v2 组件没有统一的 getCopyText 方法，每个组件自己处理复制逻辑
    const textToCopy = fieldValue?.display || (fieldValue?.raw !== null && fieldValue?.raw !== undefined ? String(fieldValue.raw) : '')
    
    if (!textToCopy) {
      ElMessage.warning('没有可复制的内容')
      return
    }
    
    navigator.clipboard.writeText(textToCopy).then(() => {
      ElMessage.success(`已复制 ${field.name}`)
    }).catch(() => {
      ElMessage.error('复制失败')
    })
  } catch (error) {
    // ✅ 使用 ErrorHandler 统一处理错误
    ErrorHandler.handleWidgetError(`TableRenderer.copyFieldValue[${field.code}]`, error, {
      showMessage: true
    })
  }
}

// ==================== 详情抽屉编辑模式 ====================

/**
 * 构建编辑用的 FunctionDetail
 * 只包含可编辑的字段（根据 table_permission 过滤）
 */
const editFunctionDetail = computed<FunctionDetail>(() => {
  // 过滤字段（只显示可编辑的字段）
  const editableFields = props.functionData.response.filter((field: FieldConfig) => {
    const permission = field.table_permission
    // 编辑模式：显示空、update 权限的字段
    return !permission || permission === '' || permission === 'update'
  })
  
  return {
    id: 0,
    app_id: 0,
    tree_id: 0,
    method: 'PUT',  // 编辑使用 PUT 方法
    router: props.functionData.router,
    has_config: false,
    create_tables: '',
    callbacks: props.functionData.callbacks,
    template_type: 'form',
    request: editableFields,  // 使用过滤后的字段
    response: [],
    created_at: '',
    updated_at: '',
    full_code_path: ''
  }
})

/**
 * 切换到编辑模式
 */
const switchToEditMode = (): void => {
  if (!currentDetailRow.value) {
    ElMessage.error('记录数据不存在')
    return
  }
  detailMode.value = 'edit'
  // FormRenderer 会自动使用 initialData 填充数据
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
  if (!detailFormRendererRef.value) {
    ElMessage.error('表单引用不存在')
    return
  }
  
  if (!currentDetailRow.value || !currentDetailRow.value.id) {
    ElMessage.error('记录 ID 不存在')
    return
  }
  
  try {
    detailSubmitting.value = true
    
    // 1. 准备提交数据
    const submitData = detailFormRendererRef.value.prepareSubmitDataWithTypeConversion()
    
    // 2. 调用更新接口（复用现有的更新逻辑）
    const success = await handleUpdateRow(currentDetailRow.value.id, submitData)
    
    if (success) {
      // 3. 刷新当前记录数据
      await refreshCurrentDetailRow()
      
      // 4. 切换回查看模式
      detailMode.value = 'view'
      
      ElMessage.success('保存成功')
    }
  } catch (error: any) {
    console.error('保存失败:', error)
    const errorMessage = error?.response?.data?.msg 
      || error?.response?.data?.message 
      || error?.message 
      || '保存失败'
    ElMessage.error(errorMessage)
  } finally {
    detailSubmitting.value = false
  }
}

/**
 * 刷新当前详情记录数据
 * 
 * 🔥 注意：handleUpdateRow 已经会调用 loadTableData() 刷新表格数据
 * 所以这里只需要从已刷新的 tableData 中更新当前记录即可，不需要再次调用 loadTableData()
 */
const refreshCurrentDetailRow = async (): Promise<void> => {
  if (!currentDetailRow.value || !currentDetailRow.value.id) {
    return
  }
  
  try {
    // 🔥 不需要重新加载表格数据，因为 handleUpdateRow 已经加载过了
    // 直接从最新的表格数据中找到当前记录
    const updatedRow = tableData.value.find((row: any) => row.id === currentDetailRow.value.id)
    if (updatedRow) {
      currentDetailRow.value = updatedRow
      // 更新索引
      const index = tableData.value.findIndex((row: any) => row.id === currentDetailRow.value.id)
      if (index >= 0) {
        currentDetailIndex.value = index
      }
      
      // 🔥 收集更新后的 files widget 的 upload_user 并查询用户信息
      const filesUploadUsers = collectFilesUploadUsersFromRow(updatedRow, visibleFields.value)
      
      if (filesUploadUsers.length > 0) {
        // 批量查询用户信息（自动处理缓存）
        const users = await userInfoStore.batchGetUserInfo(filesUploadUsers)
        
        // 更新 userInfoMap，供详情中的 FilesWidget 使用
        users.forEach((user: any) => {
          if (user.username) {
            userInfoMap.value.set(user.username, user)
          }
        })
      }
    }
  } catch (error) {
    console.error('刷新记录数据失败:', error)
  }
}

// ==================== 监听函数变化 ====================

/**
 * 监听函数配置变化
 * 当函数配置更新时，重新加载数据
 * 
 * 🔥 注意：不设置 immediate: true，因为 useTableOperations 的 initialize() 已经会在初始化时调用 loadTableData()
 * 如果设置 immediate: true，会导致初始化时调用两次 loadTableData()
 */
watch(() => props.functionData, () => {
  // 🔥 清空搜索表单，但保留 URL 中的搜索参数（restoreFromURL 会恢复）
  searchForm.value = {}
  currentPage.value = 1
  // 🔥 从 URL 恢复状态（包括搜索参数）
  restoreFromURL()
  loadTableData()
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
  z-index: 1;
  overflow: visible;
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

/* 🔥 操作列样式 - 修复 fixed 列按钮点击问题 */
:deep(.action-column) {
  position: relative;
  z-index: 10;
}

:deep(.action-column .cell) {
  position: relative;
  z-index: 10;
  pointer-events: auto;
}

.action-buttons {
  position: relative;
  z-index: 11;
  display: flex;
  gap: 8px;
  align-items: center;
  pointer-events: auto;
}

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

.action-buttons {
  position: relative !important;
  z-index: 2004 !important;
  pointer-events: auto !important;
}

:deep(.el-table__fixed-right .action-buttons) {
  z-index: 2004 !important;
  pointer-events: auto !important;
}

:deep(.el-table__fixed-right .action-buttons .el-button) {
  position: relative !important;
  z-index: 2005 !important;
  pointer-events: auto !important;
  cursor: pointer !important;
}

/* 关键：确保表格主体内容不会遮挡 fixed 列 */
:deep(.el-table__body-wrapper) {
  z-index: 1 !important;
  position: relative;
  pointer-events: auto !important;
  /* 确保主体内容不会覆盖 fixed 列区域 */
  overflow: visible !important;
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

/* 关键修复：当窗口缩小时，确保 fixed 列区域的表格主体单元格不拦截点击 */
:deep(.el-table__body-wrapper) {
  /* 在 fixed 列区域，让点击事件穿透 */
  clip-path: none !important;
}

/* 确保表格整体容器不会遮挡 */
:deep(.el-table) {
  position: relative;
  z-index: 1;
  overflow: visible !important;
}

:deep(.el-table__inner-wrapper) {
  position: relative;
  z-index: 1;
  overflow: visible !important;
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

/* 🔥 详情抽屉样式 - 参考旧版本设计 */
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

  .drawer-header-actions {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .drawer-mode-actions {
    display: flex;
    align-items: center;
    gap: 8px;
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
      background: var(--el-fill-color-light);
      padding: 6px 12px;
      border-radius: 4px;
      border: 1px solid var(--el-border-color-lighter);
      font-weight: 500;
    }
  }

  .detail-content {
    padding: 20px;
  }

  .edit-content {
    padding: 20px;
  }

  /* 🔥 字段网格布局 - 参考旧版本 */
  .fields-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 4px;
  }

  .field-row {
    display: grid;
    grid-template-columns: 140px 1fr;
    gap: 12px;
    padding: 8px 12px;
    border-bottom: 1px solid var(--el-border-color-extra-light);
    align-items: start;
    min-height: auto;
    transition: all 0.2s ease;
    border-radius: 4px;
    background: transparent;
  }

  .field-row:hover {
    background: var(--el-fill-color-light);
    border-color: var(--el-border-color);
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
  }

  .field-label {
    font-size: 14px;
    font-weight: 500;
    color: var(--el-text-color-secondary);
    display: flex;
    align-items: center;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .field-value {
    font-size: 14px;
    color: var(--el-text-color-primary);
    word-break: break-word;
    line-height: 1.6;
    display: flex;
    align-items: flex-start;
    gap: 8px;
    min-height: 24px;
    position: relative;
  }

  .field-actions {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    margin-top: 2px;
    opacity: 0;
    transition: opacity 0.2s ease;
  }

  .field-row:hover .field-actions {
    opacity: 1;
  }

  .copy-btn {
    padding: 4px 6px;
    font-size: 12px;
    height: 24px;
    min-height: 24px;
    border-radius: 4px;
    font-weight: 500;
    transition: all 0.2s ease;
    background: var(--el-color-primary-light-8);
    color: var(--el-color-primary);
    border: 1px solid var(--el-color-primary-light-5);
  }

  .copy-btn:hover {
    background: var(--el-color-primary-light-7);
    border-color: var(--el-color-primary-light-3);
    transform: scale(1.05);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  }

  .field-content {
    flex: 1;
    min-width: 0;
  }
}
</style>
