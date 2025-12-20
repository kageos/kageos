<template>
  <el-drawer
    v-model="showDetailDrawer"
    title="记录详情"
    direction="rtl"
    size="900px"
    class="grouped-detail-drawer"
    :append-to-body="true"
    :modal="true"
    @close="handleDetailDrawerClose"
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

    <!-- 查看模式：分组布局 -->
    <div class="grouped-detail-content" v-if="currentDetailRow && detailMode === 'view'">
      <el-tabs v-model="activeTab" @tab-change="handleTabChange" class="detail-tabs">
        <!-- 详情 tab -->
        <el-tab-pane label="详情" name="detail">
          <div class="tab-content">
            <!-- 链接操作区域 -->
            <div v-if="linkFields.length > 0" class="detail-links-section">
              <div class="links-section-title">相关链接</div>
              <div class="links-section-content">
                <LinkWidget
                  v-for="linkField in linkFields"
                  :key="linkField.code"
                  :field="linkField"
                  :value="convertToFieldValue(currentDetailRow[linkField.code], linkField)"
                  :field-path="linkField.code"
                  mode="detail"
                  class="detail-link-item"
                />
              </div>
            </div>

            <!-- 🔥 新布局：分组展示 -->
            <div class="grouped-detail-layout">
              <!-- 顶部：状态/分类字段组（横向展示） -->
              <div v-if="groupedFields.statusFields.length > 0" class="status-section">
                <div 
                  v-for="field in groupedFields.statusFields"
                  :key="field.code"
                  class="status-field-card"
                >
                  <span class="status-label">{{ field.name }}</span>
                  <div class="status-value">
                    <component 
                      :is="renderDetailField(field, currentDetailRow[field.code])"
                    />
                  </div>
                </div>
              </div>

              <!-- 主布局：左右分栏 -->
              <div class="main-layout">
                <!-- 左侧：主要业务字段 -->
                <div class="main-content">
                  <div 
                    v-for="field in groupedFields.mainFields"
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

                <!-- 右侧：元数据字段组（侧边栏） -->
                <div class="sidebar-content">
                  <!-- ID 字段 -->
                  <div v-if="groupedFields.idField" class="metadata-section">
                    <div class="metadata-section-title">基本信息</div>
                    <div class="field-row metadata-field">
                      <div class="field-label">ID</div>
                      <div class="field-value">
                        <div class="field-actions">
                          <el-button 
                            type="primary" 
                            size="small" 
                            text 
                            @click="copyFieldValue(groupedFields.idField!, currentDetailRow[groupedFields.idField!.code])"
                            class="copy-btn"
                            title="复制ID"
                          >
                            <el-icon><DocumentCopy /></el-icon>
                          </el-button>
                        </div>
                        <div class="field-content">
                          <component 
                            :is="renderDetailField(groupedFields.idField!, currentDetailRow[groupedFields.idField!.code])"
                          />
                        </div>
                      </div>
                    </div>
                  </div>

                  <!-- 用户字段组 -->
                  <div v-if="groupedFields.userFields.length > 0" class="metadata-section">
                    <div class="metadata-section-title">人员信息</div>
                    <div 
                      v-for="field in groupedFields.userFields"
                      :key="field.code"
                      class="field-row metadata-field"
                    >
                      <div class="field-label">
                        {{ field.name }}
                      </div>
                      <div class="field-value">
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
                        <div class="field-content">
                          <component 
                            :is="renderDetailField(field, currentDetailRow[field.code])"
                          />
                        </div>
                      </div>
                    </div>
                  </div>

                  <!-- 时间字段组 -->
                  <div v-if="groupedFields.timestampFields.length > 0" class="metadata-section">
                    <div class="metadata-section-title">时间信息</div>
                    <div 
                      v-for="field in groupedFields.timestampFields"
                      :key="field.code"
                      class="field-row metadata-field"
                    >
                      <div class="field-label">
                        {{ field.name }}
                      </div>
                      <div class="field-value">
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
                        <div class="field-content">
                          <component 
                            :is="renderDetailField(field, currentDetailRow[field.code])"
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- 底部：复杂字段（可折叠） -->
              <div v-if="groupedFields.complexFields.length > 0" class="complex-section">
                <div 
                  v-for="field in groupedFields.complexFields"
                  :key="field.code"
                  class="complex-field-card"
                >
                  <el-collapse>
                    <el-collapse-item :name="field.code">
                      <template #title>
                        <div class="complex-field-title">
                          <span class="complex-field-name">{{ field.name }}</span>
                        </div>
                      </template>
                      <div class="complex-field-content">
                        <component 
                          :is="renderDetailField(field, currentDetailRow[field.code])"
                        />
                      </div>
                    </el-collapse-item>
                  </el-collapse>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 操作日志 tab -->
        <el-tab-pane label="操作日志" name="operateLog">
          <div class="tab-content">
            <OperateLogSection
              ref="operateLogSectionRef"
              :full-code-path="getFullCodePath"
              :row-id="getCurrentRowId"
              :function-detail="functionData"
              :auto-load="false"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- 🔥 编辑模式：使用 FormRenderer（与原组件保持一致） -->
    <div class="edit-content" v-else-if="currentDetailRow && detailMode === 'edit'">
      <FormRenderer
        v-if="editFunctionDetail"
        ref="detailFormRendererRef"
        :function-detail="editFunctionDetail"
        :initial-data="currentDetailRow"
        :user-info-map="userInfoMap"
        :show-submit-button="false"
        :show-reset-button="false"
      />
      <el-empty v-else description="无法构建编辑表单" />
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, h, ref, watch } from 'vue'
import { Edit, ArrowLeft, ArrowRight, DocumentCopy } from '@element-plus/icons-vue'
import { ElIcon, ElButton, ElMessage, ElEmpty, ElTabs, ElTabPane, ElCollapse, ElCollapseItem } from 'element-plus'
import { useTableDetail, type UseTableDetailOptions } from '@/composables/useTableDetail'
import { widgetComponentFactory } from '@/core/factories-v2'
import { ErrorHandler } from '@/core/utils/ErrorHandler'
import { convertToFieldValue } from '@/utils/field'
import FormRenderer from '@/core/renderers-v2/FormRenderer.vue'
import LinkWidget from '@/core/widgets-v2/components/LinkWidget.vue'
import OperateLogSection from './OperateLogSection.vue'
import { WidgetType } from '@/core/constants/widget'
import type { Function as FunctionType, ServiceTree } from '@/types'
import type { FieldConfig } from '@/core/types/field'

interface Props {
  /** 函数配置数据 */
  functionData: FunctionType
  /** 当前函数节点（来自 ServiceTree） */
  currentFunction?: ServiceTree
  /** 表格数据 */
  tableData: any[]
  /** 可见字段列表 */
  visibleFields: FieldConfig[]
  /** ID 字段 */
  idField?: FieldConfig
  /** 链接字段列表 */
  linkFields: FieldConfig[]
  /** 是否有更新回调 */
  hasUpdateCallback: boolean
  /** 用户信息映射 */
  userInfoMap: Map<string, any>
  /** 更新回调函数 */
  onUpdate: (id: number, data: any, oldData: any) => Promise<boolean>
  /** 刷新回调函数 */
  onRefresh: () => Promise<void>
}

const props = defineProps<Props>()

// ==================== 使用 useTableDetail ====================

const detailOptions: UseTableDetailOptions = {
  functionData: props.functionData,
  currentFunction: props.currentFunction,
  tableData: props.tableData,
  visibleFields: props.visibleFields,
  idField: props.idField,
  linkFields: props.linkFields,
  hasUpdateCallback: props.hasUpdateCallback,
  userInfoMap: props.userInfoMap,
  onUpdate: props.onUpdate,
  onRefresh: props.onRefresh
}

const {
  showDetailDrawer,
  currentDetailRow,
  currentDetailIndex,
  detailMode,
  detailFormRendererRef,
  detailSubmitting,
  getFullCodePath,
  getCurrentRowId,
  editFunctionDetail,
  handleNavigate,
  switchToEditMode,
  switchToViewMode,
  handleDetailSave,
  handleDetailDrawerClose
} = useTableDetail(detailOptions)

// Tab 相关
const activeTab = ref('detail')
const operateLogSectionRef = ref<InstanceType<typeof OperateLogSection> | null>(null)

// 处理 tab 切换
const handleTabChange = (tabName: string) => {
  if (tabName === 'operateLog' && operateLogSectionRef.value) {
    // 切换到操作日志 tab 时，触发加载
    operateLogSectionRef.value.load()
  }
}

// 监听详情行变化，重置 tab
watch(
  () => currentDetailRow.value,
  () => {
    activeTab.value = 'detail'
  }
)

// ==================== 字段分组逻辑 ====================

/**
 * 字段分组：按组件类型分组字段
 */
const groupedFields = computed(() => {
  // 过滤掉 link 字段（已在顶部单独展示）
  const fields = props.visibleFields.filter((f: FieldConfig) => f.widget?.type !== WidgetType.LINK)
  
  // 状态/分类字段组（select, multiselect, radio, checkbox, switch）
  const statusFields = fields.filter((f: FieldConfig) => {
    const type = f.widget?.type
    return [
      WidgetType.SELECT,
      WidgetType.MULTI_SELECT,
      WidgetType.RADIO,
      WidgetType.CHECKBOX,
      WidgetType.SWITCH
    ].includes(type as any)
  })
  
  // 用户字段组
  const userFields = fields.filter((f: FieldConfig) => f.widget?.type === WidgetType.USER)
  
  // 时间字段组
  const timestampFields = fields.filter((f: FieldConfig) => f.widget?.type === WidgetType.TIMESTAMP)
  
  // ID 字段
  const idField = fields.find((f: FieldConfig) => f.widget?.type === WidgetType.ID) || props.idField
  
  // 复杂字段组（form, table, richtext）
  const complexFields = fields.filter((f: FieldConfig) => {
    const type = f.widget?.type
    return [
      WidgetType.FORM,
      WidgetType.TABLE,
      WidgetType.RICH_TEXT
    ].includes(type as any)
  })
  
  // 主要业务字段（其他所有字段）
  const mainFields = fields.filter((f: FieldConfig) => {
    const type = f.widget?.type
    return ![
      WidgetType.SELECT,
      WidgetType.MULTI_SELECT,
      WidgetType.RADIO,
      WidgetType.CHECKBOX,
      WidgetType.SWITCH,
      WidgetType.USER,
      WidgetType.TIMESTAMP,
      WidgetType.ID,
      WidgetType.FORM,
      WidgetType.TABLE,
      WidgetType.RICH_TEXT,
      WidgetType.LINK
    ].includes(type as any)
  })
  
  return {
    statusFields,
    userFields,
    timestampFields,
    idField,
    mainFields,
    complexFields
  }
})

// ==================== 详情字段渲染 ====================

/**
 * 渲染详情字段（使用 widgets-v2，与原组件保持一致）
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
    const idField = props.visibleFields.find((f: FieldConfig) => {
      const code = f.code.toLowerCase()
      return code === 'id' || code === 'ID' || code.endsWith('_id') || code.endsWith('Id')
    })
    const recordId = idField && currentDetailRow.value ? currentDetailRow.value[idField.code] : undefined
    
    // 🔥 从 router 或 currentFunction 获取函数名称、user 和 app 名称
    let functionName: string | undefined = undefined
    let userName: string | undefined = undefined
    let appName: string | undefined = undefined
    
    if (props.currentFunction?.name) {
      functionName = props.currentFunction.name
    } else if (props.functionData?.router) {
      const routerParts = props.functionData.router.split('/').filter(Boolean)
      if (routerParts.length > 0) {
        functionName = routerParts[routerParts.length - 1]
      }
    }
    
    if (props.functionData?.router) {
      const routerParts = props.functionData.router.split('/').filter(Boolean)
      if (routerParts.length >= 1) {
        userName = routerParts[0]
      }
      if (routerParts.length >= 2) {
        appName = routerParts[1]
      }
    }
    
    if (userName && appName && functionName) {
      functionName = `${userName}_${appName}_${functionName}`
    } else if (appName && functionName) {
      functionName = `${appName}_${functionName}`
    }
    
    // 🔥 为详情模式创建 formRendererContext（用于 OnSelectFuzzy 回调）
    const detailFormRendererContext = {
      getFunctionMethod: () => props.functionData.method,
      getFunctionRouter: () => props.functionData.router,
      getSubmitData: () => ({}),
      registerWidget: () => {},
      unregisterWidget: () => {},
      getFieldError: () => undefined
    }
    
    return h(WidgetComponent, {
      field: field,
      value: value,
      'model-value': value,
      'field-path': field.code,
      mode: 'detail',
      'user-info-map': props.userInfoMap,
      'form-renderer': detailFormRendererContext,
      functionName: functionName,
      recordId: recordId
    })
  } catch (error) {
    // ✅ 使用 ErrorHandler 统一处理错误
    return ErrorHandler.handleWidgetError(`TableDetailDrawerGrouped.renderDetailField[${field.code}]`, error, {
      showMessage: false,
      fallbackValue: h('span', rawValue !== null && rawValue !== undefined ? String(rawValue) : '-')
    })
  }
}

/**
 * 复制字段值到剪贴板（与原组件保持一致）
 */
const copyFieldValue = (field: FieldConfig, value: any): void => {
  try {
    const fieldValue = convertToFieldValue(value, field)
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
    ErrorHandler.handleWidgetError(`TableDetailDrawerGrouped.copyFieldValue[${field.code}]`, error, {
      showMessage: true
    })
  }
}

// ==================== 暴露方法给父组件（与原组件保持一致） ====================

// handleShowDetail 和 restoreDetailFromURL 已经在 useTableDetail 的返回值中
defineExpose({
  handleShowDetail,
  restoreDetailFromURL
})
</script>

<style scoped>
.grouped-detail-drawer {
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

  .grouped-detail-content {
    padding: 20px;
  }

  .edit-content {
    padding: 20px;
  }

  /* 链接区域（与原组件保持一致） */
  .detail-links-section {
    margin-bottom: 24px;
    padding: 16px;
    background-color: var(--el-fill-color-lighter);
    border-radius: 8px;
    border: 1px solid var(--el-border-color-light);
  }

  .links-section-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    margin-bottom: 12px;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .links-section-title::before {
    content: '';
    width: 3px;
    height: 16px;
    background-color: var(--el-color-primary);
    border-radius: 2px;
  }

  .links-section-content {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    align-items: center;
  }

  .detail-link-item {
    flex-shrink: 0;
  }

  /* Tab 样式 */
  .detail-tabs {
    :deep(.el-tabs__header) {
      margin-bottom: 20px;
    }

    :deep(.el-tabs__item) {
      font-size: 14px;
      font-weight: 500;
    }

    :deep(.el-tabs__active-bar) {
      background-color: var(--el-color-primary);
    }
  }

  .tab-content {
    padding: 0;
  }

  /* ==================== 新布局样式 ==================== */

  /* 分组布局容器 */
  .grouped-detail-layout {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  /* 顶部：状态/分类字段组（横向展示） */
  .status-section {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    padding: 16px;
    background: var(--el-fill-color-lighter);
    border-radius: 8px;
    border: 1px solid var(--el-border-color-light);
  }

  .status-field-card {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    background: var(--el-bg-color);
    border-radius: 6px;
    border: 1px solid var(--el-border-color);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
    transition: all 0.2s ease;
  }

  .status-field-card:hover {
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
    transform: translateY(-1px);
  }

  .status-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--el-text-color-secondary);
    white-space: nowrap;
  }

  .status-value {
    flex: 1;
    min-width: 0;
  }

  /* 主布局：左右分栏 */
  .main-layout {
    display: grid;
    grid-template-columns: 1fr 320px;
    gap: 24px;
  }

  /* 左侧：主要业务字段 */
  .main-content {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  /* 右侧：元数据字段组（侧边栏） */
  .sidebar-content {
    display: flex;
    flex-direction: column;
    gap: 16px;
    background: var(--el-fill-color-lighter);
    border-radius: 8px;
    padding: 16px;
    border: 1px solid var(--el-border-color-light);
    position: sticky;
    top: 20px;
    max-height: calc(100vh - 200px);
    overflow-y: auto;
  }

  .metadata-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .metadata-section-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    margin-bottom: 4px;
    padding-bottom: 8px;
    border-bottom: 1px solid var(--el-border-color);
  }

  .metadata-field {
    padding: 8px 0;
    border-bottom: 1px solid var(--el-border-color-extra-light);
  }

  .metadata-field:last-child {
    border-bottom: none;
  }

  /* 标准字段行样式（复用） */
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

  /* 底部：复杂字段 */
  .complex-section {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .complex-field-card {
    background: var(--el-fill-color-lighter);
    border-radius: 8px;
    border: 1px solid var(--el-border-color-light);
    overflow: hidden;
  }

  .complex-field-title {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .complex-field-name {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .complex-field-content {
    padding: 16px;
  }

  /* 响应式设计 */
  @media (max-width: 1200px) {
    .main-layout {
      grid-template-columns: 1fr;
    }

    .sidebar-content {
      position: static;
      max-height: none;
    }
  }

  @media (max-width: 768px) {
    .status-section {
      flex-direction: column;
    }

    .status-field-card {
      width: 100%;
    }

    .field-row {
      grid-template-columns: 1fr;
      gap: 8px;
    }

    .field-label {
      margin-bottom: 4px;
    }
  }
}
</style>

