<template>
  <el-drawer
    v-model="showDetailDrawer"
    title="记录详情"
    direction="rtl"
    size="900px"
    class="detail-drawer"
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
          <!-- 布局切换按钮 -->
          <el-button
            v-if="detailMode === 'view' && props.onToggleLayout"
            size="small"
            text
            @click="props.onToggleLayout"
            title="切换到分组布局"
          >
            <el-icon><Grid /></el-icon>
            分组布局
          </el-button>
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

    <!-- 查看模式：纯展示模式 -->
    <div class="detail-content" v-if="currentDetailRow && detailMode === 'view'">
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
            
            <div class="fields-grid">
              <div 
                v-for="field in visibleFields.filter((f: FieldConfig) => f.widget?.type !== 'link')"
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

    <!-- 🔥 编辑模式：使用 FormRenderer -->
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
import { Edit, ArrowLeft, ArrowRight, DocumentCopy, Grid } from '@element-plus/icons-vue'
import { ElIcon, ElButton, ElMessage, ElEmpty, ElTabs, ElTabPane } from 'element-plus'
import { useTableDetail, type UseTableDetailOptions } from '@/composables/useTableDetail'
import { widgetComponentFactory } from '@/core/factories-v2'
import { ErrorHandler } from '@/core/utils/ErrorHandler'
import { convertToFieldValue } from '@/utils/field'
import FormRenderer from '@/core/renderers-v2/FormRenderer.vue'
import LinkWidget from '@/core/widgets-v2/components/LinkWidget.vue'
import OperateLogSection from './OperateLogSection.vue'
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
  /** 切换布局回调函数 */
  onToggleLayout?: () => void
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

// ==================== 详情字段渲染 ====================

/**
 * 渲染详情字段（使用 widgets-v2）
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
    
    // 🔥 为详情模式创建 formRendererContext（用于 OnSelectFuzzy 回调）
    const detailFormRendererContext = {
      getFunctionMethod: () => props.functionData.method,
      getFunctionRouter: () => props.functionData.router,
      getSubmitData: () => ({}), // 详情模式下不需要提交数据
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
    return ErrorHandler.handleWidgetError(`TableDetailDrawer.renderDetailField[${field.code}]`, error, {
      showMessage: false,
      fallbackValue: h('span', rawValue !== null && rawValue !== undefined ? String(rawValue) : '-')
    })
  }
}

/**
 * 复制字段值到剪贴板
 */
const copyFieldValue = (field: FieldConfig, value: any): void => {
  try {
    // 🔥 将原始值转换为 FieldValue 格式
    const fieldValue = convertToFieldValue(value, field)
    
    // 🔥 简化实现：优先使用 display，否则使用 raw
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
    ErrorHandler.handleWidgetError(`TableDetailDrawer.copyFieldValue[${field.code}]`, error, {
      showMessage: true
    })
  }
}

// ==================== 暴露方法给父组件 ====================

defineExpose({
  handleShowDetail,
  restoreDetailFromURL
})
</script>

<style scoped>
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

  /* 字段网格布局 */
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

  /* 详情页面链接区域 */
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
}
</style>

