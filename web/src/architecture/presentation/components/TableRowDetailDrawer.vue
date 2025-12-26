<!--
  TableRowDetailDrawer - 表格行详情抽屉组件
  
  职责：
  - 详情展示
  - 详情编辑
  - 详情导航（上一条/下一条）
-->

<template>
  <el-drawer
    v-model="visible"
    :title="title"
    size="60%"
    destroy-on-close
    :modal="true"
    :close-on-click-modal="true"
    class="detail-drawer"
    :show-close="true"
    @close="handleClose"
  >
    <template #header>
      <div class="drawer-header">
        <span class="drawer-title">{{ title }}</span>
        <div class="drawer-header-actions">
          <!-- 模式切换按钮 -->
          <div class="drawer-mode-actions">
            <el-button
              v-if="mode === 'read'"
              :type="canEdit ? 'primary' : 'default'"
              :plain="!canEdit"
              size="small"
              class="edit-btn"
              :class="{ 'action-btn-no-permission': !canEdit }"
              @click="handleToggleMode('edit')"
            >
              <el-icon><component :is="canEdit ? Edit : Lock" /></el-icon>
              {{ canEdit ? '编辑' : '编辑（需权限）' }}
            </el-button>
            <el-button
              v-if="mode === 'edit'"
              size="small"
              @click="handleToggleMode('read')"
            >
              取消
            </el-button>
            <el-button
              v-if="mode === 'edit'"
              type="primary"
              size="small"
              :loading="submitting"
              :disabled="!isFormRendererReady"
              @click="handleSubmit"
            >
              保存
            </el-button>
          </div>
          <!-- 布局切换按钮 -->
          <el-button
            v-if="mode === 'read'"
            size="small"
            text
            @click="toggleDetailLayout"
            :title="useGroupedDetailLayout ? '切换到原布局' : '切换到分组布局'"
          >
            <el-icon><component :is="useGroupedDetailLayout ? List : Grid" /></el-icon>
            {{ useGroupedDetailLayout ? '原布局' : '分组布局' }}
          </el-button>
          <!-- 导航按钮（上一个/下一个） -->
          <div class="drawer-navigation" v-if="tableData && tableData.length > 1 && mode === 'read'">
            <el-button
              size="small"
              :disabled="currentIndex <= 0"
              @click="handleNavigate('prev')"
            >
              <el-icon><ArrowLeft /></el-icon>
              上一个
            </el-button>
            <span class="nav-info">{{ (currentIndex >= 0 ? currentIndex + 1 : 0) }} / {{ tableData.length }}</span>
            <el-button
              size="small"
              :disabled="currentIndex >= tableData.length - 1"
              @click="handleNavigate('next')"
            >
              下一个
              <el-icon><ArrowRight /></el-icon>
            </el-button>
          </div>
        </div>
      </div>
    </template>

    <div class="detail-content">
      <!-- 详情模式 - 使用更美观的布局 -->
      <div v-if="mode === 'read'">
        <el-tabs v-model="activeTab" @tab-change="handleTabChange" class="detail-tabs">
          <!-- 详情 tab -->
          <el-tab-pane label="详情" name="detail">
            <div class="tab-content">
              <!-- 链接操作区域：收集所有 link 字段显示在顶部 -->
              <div v-if="linkFields.length > 0" class="detail-links-section">
                <div class="links-section-title">相关链接</div>
                <div class="links-section-content">
                  <LinkWidget
                    v-for="linkField in linkFields"
                    :key="linkField.code"
                    :field="linkField"
                    :value="getFieldValue(linkField.code)"
                    :field-path="linkField.code"
                    mode="detail"
                    class="detail-link-item"
                  />
                </div>
              </div>
              
              <!-- 🔥 根据布局模式渲染不同的布局 -->
              <!-- 分组布局 -->
              <div v-if="useGroupedDetailLayout" class="grouped-detail-layout">
                <!-- 顶部：状态/分类字段组（横向展示） -->
                <div v-if="groupedFields.statusFields.length > 0" class="status-section">
                  <div 
                    v-for="field in groupedFields.statusFields"
                    :key="field.code"
                    class="status-field-card"
                  >
                    <span class="status-label">{{ field.name }}</span>
                    <div class="status-value">
                      <WidgetComponent
                        :field="field"
                        :value="getFieldValue(field.code)"
                        mode="detail"
                        :user-info-map="userInfoMap"
                        :function-name="functionName"
                        :record-id="recordId"
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
                        <WidgetComponent
                          :field="field"
                          :value="getFieldValue(field.code)"
                          mode="detail"
                          :user-info-map="userInfoMap"
                          :function-name="functionName"
                          :record-id="recordId"
                        />
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
                          <WidgetComponent
                            :field="groupedFields.idField"
                            :value="getFieldValue(groupedFields.idField.code)"
                            mode="detail"
                            :user-info-map="userInfoMap"
                            :function-name="functionName"
                            :record-id="recordId"
                          />
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
                          <WidgetComponent
                            :field="field"
                            :value="getFieldValue(field.code)"
                            mode="detail"
                            :user-info-map="userInfoMap"
                            :function-name="functionName"
                            :record-id="recordId"
                          />
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
                          <WidgetComponent
                            :field="field"
                            :value="getFieldValue(field.code)"
                            mode="detail"
                            :user-info-map="userInfoMap"
                            :function-name="functionName"
                            :record-id="recordId"
                          />
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
                          <WidgetComponent
                            :field="field"
                            :value="getFieldValue(field.code)"
                            mode="detail"
                            :user-info-map="userInfoMap"
                            :function-name="functionName"
                            :record-id="recordId"
                          />
                        </div>
                      </el-collapse-item>
                    </el-collapse>
                  </div>
                </div>
              </div>

              <!-- 原布局（网格布局） -->
              <div v-else class="detail-fields-grid">
                <div
                  v-for="field in fields.filter((f: FieldConfig) => f.widget?.type !== WidgetType.LINK)"
                  :key="field.code"
                  class="detail-field-row"
                >
                  <div class="detail-field-label">
                    {{ field.name }}
                  </div>
                  <div class="detail-field-value">
                    <WidgetComponent
                      :field="field"
                      :value="getFieldValue(field.code)"
                      mode="detail"
                      :user-info-map="userInfoMap"
                      :function-name="functionName"
                      :record-id="recordId"
                    />
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
                :full-code-path="fullCodePath"
                :row-id="rowId"
                :function-detail="currentFunctionDetail || editFunctionDetail"
                :auto-load="false"
              />
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>

      <!-- 编辑模式（复用 FormRenderer） -->
      <div v-else class="edit-form-wrapper" v-loading="submitting">
        <FormRenderer
          v-if="editFunctionDetail && mode === 'edit'"
          ref="formRendererRef"
          :key="`detail-edit-${rowData?.id || ''}-${mode}`"
          :function-detail="editFunctionDetail"
          :initial-data="rowData || {}"
          :show-submit-button="false"
          :show-reset-button="false"
          :show-share-button="false"
          :show-debug-button="false"
        />
        <el-empty v-else-if="!editFunctionDetail" description="无法构建编辑表单" />
        <div v-else class="form-loading">
          <el-skeleton :rows="5" animated />
        </div>
      </div>
    </div>

    <template #footer>
      <div class="drawer-footer">
        <el-button @click="handleClose">关闭</el-button>
      </div>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import { Edit, ArrowLeft, ArrowRight, Grid, List, Lock } from '@element-plus/icons-vue'
import { ElMessage, ElTabs, ElTabPane } from 'element-plus'
import { useRouter } from 'vue-router'
import { buildPermissionApplyURL } from '@/utils/permission'
import FormRenderer from '@/core/renderers-v2/FormRenderer.vue'
import WidgetComponent from '../widgets/WidgetComponent.vue'
import LinkWidget from '@/core/widgets-v2/components/LinkWidget.vue'
import OperateLogSection from '@/components/OperateLogSection.vue'
import { WidgetType } from '@/core/constants/widget'
import type { FieldConfig, FieldValue } from '../../domain/types'
import type { FunctionDetail } from '../../domain/interfaces/IFunctionLoader'

interface Props {
  visible: boolean
  title: string
  mode: 'read' | 'edit'
  fields: FieldConfig[]
  rowData: Record<string, any> | null
  tableData?: any[]
  currentIndex?: number
  canEdit?: boolean
  editFunctionDetail?: FunctionDetail | null
  currentFunctionDetail?: FunctionDetail | null  // 原始的 functionDetail（未修改的，用于操作日志）
  userInfoMap?: Map<string, any>
  submitting?: boolean
  currentFunction?: any  // ServiceTree 节点，包含 full_code_path
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'update:mode', value: 'read' | 'edit'): void
  (e: 'navigate', direction: 'prev' | 'next'): void
  (e: 'submit', formRendererRef: InstanceType<typeof FormRenderer>): void
  (e: 'close'): void
}

const props = withDefaults(defineProps<Props>(), {
  tableData: () => [],
  currentIndex: -1,
  canEdit: false,
  editFunctionDetail: null,
  currentFunctionDetail: null,
  userInfoMap: () => new Map(),
  submitting: false,
  currentFunction: null
})

const emit = defineEmits<Emits>()

const router = useRouter()

const formRendererRef = ref<InstanceType<typeof FormRenderer> | null>(null)
const isFormRendererReady = ref(false)

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
    console.error('[TableRowDetailDrawer] 读取布局设置失败:', error)
    // 出错时默认使用新布局
    return true
  }
}
const useGroupedDetailLayout = ref<boolean>(getInitialLayout())

/**
 * 切换详情布局
 */
const toggleDetailLayout = (): void => {
  useGroupedDetailLayout.value = !useGroupedDetailLayout.value
  localStorage.setItem('useGroupedDetailLayout', String(useGroupedDetailLayout.value))
  // 设置版本标记，表示这是用户明确的选择
  localStorage.setItem('useGroupedDetailLayoutVersion', '1.0')
}

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

// 监听 rowData 变化，重置 tab
watch(
  () => props.rowData,
  () => {
    activeTab.value = 'detail'
  }
)

// 监听 formRendererRef 的变化
watch(formRendererRef, (newVal) => {
  isFormRendererReady.value = !!newVal
}, { immediate: true })

// 监听 mode 变化，重置 ready 状态
watch(() => props.mode, (newMode) => {
  if (newMode === 'edit') {
    // 重置 ready 状态，等待 watch(formRendererRef) 自动更新
    isFormRendererReady.value = false
  } else {
    isFormRendererReady.value = false
  }
})

const visible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})

// 详情页的 Link 字段（用于顶部链接区域显示）
const linkFields = computed(() => {
  return props.fields.filter((f: FieldConfig) => f.widget?.type === WidgetType.LINK)
})

// ==================== 分组布局字段分组 ====================

/**
 * 分组布局的字段分组
 */
const groupedFields = computed(() => {
  // 排除 link 字段（link 字段单独显示在顶部）
  const fieldsToGroup = props.fields.filter((f: FieldConfig) => f.widget?.type !== WidgetType.LINK)
  
  // ID 字段
  const idField = fieldsToGroup.find((f: FieldConfig) => f.widget?.type === WidgetType.ID)
  
  // 状态/分类字段（select, multiselect, radio, checkbox, switch）
  const statusFields = fieldsToGroup.filter((f: FieldConfig) => {
    const widgetType = f.widget?.type
    return widgetType === WidgetType.SELECT || 
           widgetType === WidgetType.MULTISELECT || 
           widgetType === WidgetType.RADIO || 
           widgetType === WidgetType.CHECKBOX || 
           widgetType === WidgetType.SWITCH
  })
  
  // 用户字段
  const userFields = fieldsToGroup.filter((f: FieldConfig) => f.widget?.type === WidgetType.USER)
  
  // 时间字段
  const timestampFields = fieldsToGroup.filter((f: FieldConfig) => f.widget?.type === WidgetType.TIMESTAMP)
  
  // 复杂字段（form, table, richtext）
  const complexFields = fieldsToGroup.filter((f: FieldConfig) => {
    const widgetType = f.widget?.type
    return widgetType === WidgetType.FORM || 
           widgetType === WidgetType.TABLE || 
           widgetType === WidgetType.RICHTEXT
  })
  
  // 主要业务字段（排除上述所有字段）
  const mainFields = fieldsToGroup.filter((f: FieldConfig) => {
    const widgetType = f.widget?.type
    return widgetType !== WidgetType.ID &&
           widgetType !== WidgetType.SELECT &&
           widgetType !== WidgetType.MULTISELECT &&
           widgetType !== WidgetType.RADIO &&
           widgetType !== WidgetType.CHECKBOX &&
           widgetType !== WidgetType.SWITCH &&
           widgetType !== WidgetType.USER &&
           widgetType !== WidgetType.TIMESTAMP &&
           widgetType !== WidgetType.FORM &&
           widgetType !== WidgetType.TABLE &&
           widgetType !== WidgetType.RICHTEXT
  })
  
  return {
    idField,
    statusFields,
    userFields,
    timestampFields,
    complexFields,
    mainFields
  }
})

const getFieldValue = (fieldCode: string): FieldValue => {
  if (!props.rowData) return { raw: null, display: '', meta: {} }
  const value = props.rowData[fieldCode]
  return { 
    raw: value, 
    display: typeof value === 'object' ? JSON.stringify(value) : String(value ?? ''), 
    meta: {} 
  }
}

// 🔥 从 editFunctionDetail.router 提取函数名称（用于 FilesWidget 打包下载命名）
const functionName = computed(() => {
  if (!props.editFunctionDetail?.router) {
    return undefined
  }
  
  // router 格式通常是：/user/app/function_name 或 /user/app/group/function_name
  const routerParts = props.editFunctionDetail.router.split('/').filter(Boolean)
  if (routerParts.length === 0) {
    return undefined
  }
  
  // 提取函数名称（最后一段）
  let funcName = routerParts[routerParts.length - 1]
  
  // 提取 user 和 app 名称（格式：/user/app/...）
  if (routerParts.length >= 2) {
    const userName = routerParts[0]  // 第一段是 user 名称
    const appName = routerParts[1]    // 第二段是 app 名称
    
    // 如果有 user 和 app 名称，在函数名称前面加上
    if (userName && appName && funcName) {
      funcName = `${userName}_${appName}_${funcName}`
    } else if (appName && funcName) {
      // 如果只有 app 名称，也加上
      funcName = `${appName}_${funcName}`
    }
  }
  
  return funcName
})

// 🔥 从 rowData 提取 recordId（用于 FilesWidget 打包下载命名）
const recordId = computed(() => {
  if (!props.rowData) {
    return undefined
  }
  
  // 尝试从 rowData 中获取 id 字段（可能是 id、ID、record_id 等）
  const idField = Object.keys(props.rowData).find(key => {
    const lowerKey = key.toLowerCase()
    return lowerKey === 'id' || lowerKey.endsWith('_id') || lowerKey.endsWith('id')
  })
  
  if (idField) {
    const idValue = props.rowData[idField]
    return idValue !== null && idValue !== undefined ? idValue : undefined
  }
  
  return undefined
})

/**
 * 获取 full_code_path（用于操作日志查询）
 * 优先使用 currentFunction.full_code_path，否则从 editFunctionDetail.router 构建
 */
const fullCodePath = computed(() => {
  if (props.currentFunction?.full_code_path) {
    return props.currentFunction.full_code_path
  }
  if (props.editFunctionDetail?.full_code_path) {
    return props.editFunctionDetail.full_code_path
  }
  // 从 router 构建：/user/app/router -> /user/app/router
  if (props.editFunctionDetail?.router) {
    return props.editFunctionDetail.router
  }
  return ''
})

/**
 * 获取 row_id（用于操作日志查询）
 */
const rowId = computed(() => {
  if (!props.rowData) {
    return 0
  }
  // 尝试从 rowData 中获取 id 字段
  const idField = Object.keys(props.rowData).find(key => {
    const lowerKey = key.toLowerCase()
    return lowerKey === 'id' || lowerKey.endsWith('_id') || lowerKey.endsWith('id')
  })
  
  if (idField) {
    const idValue = props.rowData[idField]
    return idValue !== null && idValue !== undefined ? Number(idValue) : 0
  }
  
  return 0
})

const handleToggleMode = (newMode: 'read' | 'edit') => {
  // 如果尝试进入编辑模式但没有权限，跳转到权限申请页面
  if (newMode === 'edit' && !props.canEdit) {
    const path = fullCodePath.value
    if (path) {
      // 获取 template_type（从 currentFunctionDetail 或 functionDetail）
      const templateType = props.currentFunctionDetail?.template_type || props.functionDetail?.template_type
      const applyURL = buildPermissionApplyURL(path, 'table:update', templateType)
      router.push(applyURL)
    } else {
      ElMessage.warning('无法获取资源路径，无法申请权限')
    }
    return
  }
  emit('update:mode', newMode)
}

const handleNavigate = (direction: 'prev' | 'next') => {
  emit('navigate', direction)
}

const handleSubmit = () => {
  // 直接检查 isFormRendererReady，这个状态由 watch(formRendererRef) 自动维护
  if (!isFormRendererReady.value || !formRendererRef.value) {
    ElMessage.warning('编辑表单正在初始化，请稍后再试')
    return
  }
  
  // 直接传递 formRendererRef 给父组件
  emit('submit', formRendererRef.value)
}

const handleClose = () => {
  emit('close')
}

// 暴露方法供父组件调用
defineExpose({
  formRendererRef
})
</script>

<style scoped lang="scss">
.detail-drawer :deep(.el-drawer__header) {
  margin-bottom: 0;
  padding: 16px 20px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.detail-drawer :deep(.el-drawer__body) {
  padding: 20px;
  overflow: auto;
}

// ⭐ 无权限按钮样式优化
.action-btn-no-permission {
  color: var(--el-text-color-secondary) !important;
  border-color: var(--el-border-color-light) !important;
  
  &:hover {
    color: var(--el-color-primary) !important;
    border-color: var(--el-color-primary-light-7) !important;
    background-color: var(--el-color-primary-light-9) !important;
  }
  
  .el-icon {
    margin-right: 4px;
  }
}

.detail-content {
  height: 100%;
}

.drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.drawer-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.drawer-header-actions {
  display: flex;
  align-items: center;
  gap: 16px;
}

.drawer-mode-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.drawer-navigation {
  display: flex;
  align-items: center;
  gap: 8px;
}

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

/* 详情字段网格布局 */
.detail-fields-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 4px;
}

.detail-field-row {
  display: grid;
  grid-template-columns: 140px 1fr;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  align-items: start;
  min-height: auto;
  transition: all 0.2s ease;
  border-radius: 4px;
  background: transparent;
}

.detail-field-row:hover {
  background: var(--el-fill-color-light);
  border-color: var(--el-border-color);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

.detail-field-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  display: flex;
  align-items: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.detail-field-value {
  font-size: 14px;
  color: var(--el-text-color-primary);
  word-break: break-word;
  line-height: 1.6;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-height: 24px;
  pointer-events: auto;
  position: relative;
}

/* 详情页链接区域 */
.detail-links-section {
  margin-bottom: 24px;
  padding: 16px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}

.links-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
}

.links-section-content {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.detail-link-item {
  flex-shrink: 0;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 10px;
}

.edit-form-wrapper {
  min-height: 400px;
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

/* ==================== 分组布局样式 ==================== */

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

/* 响应式：小屏幕时改为单列 */
@media (max-width: 1200px) {
  .main-layout {
    grid-template-columns: 1fr;
  }
  
  .sidebar-content {
    position: static !important;
    max-height: none !important;
  }
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
  padding: 0;
  position: sticky;
  top: 20px;
  max-height: calc(100vh - 200px);
  overflow-y: auto;
}

.metadata-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.metadata-section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.metadata-field {
  padding: 8px 0;
  border-bottom: none;
}

/* 标准字段行样式（用于分组布局） */
/* 左侧：左右布局（label 在左，value 在右） */
.grouped-detail-layout .main-content .field-row {
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

.grouped-detail-layout .main-content .field-row:hover {
  background: var(--el-fill-color-light);
  border-color: var(--el-border-color);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

/* 右侧：上下布局（label 在上，value 在下） */
.grouped-detail-layout .sidebar-content .field-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 0;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  align-items: stretch;
  min-height: auto;
  transition: all 0.2s ease;
  border-radius: 4px;
  background: transparent;
}

.grouped-detail-layout .sidebar-content .field-row:hover {
  background: var(--el-fill-color-light);
  border-color: var(--el-border-color);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.05);
}

.grouped-detail-layout .field-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  display: flex;
  align-items: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 右侧 label 样式（更小，更紧凑） */
.grouped-detail-layout .sidebar-content .field-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}

.grouped-detail-layout .field-value {
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

/* 右侧 value 样式 */
.grouped-detail-layout .sidebar-content .field-value {
  font-size: 13px;
  width: 100%;
}

/* 底部：复杂字段 */
.complex-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-light);
}

.complex-field-card {
  background: var(--el-bg-color);
  border-radius: 6px;
  border: 1px solid var(--el-border-color);
  overflow: hidden;
}

.complex-field-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.complex-field-name {
  flex: 1;
}

.complex-field-content {
  padding: 16px;
}
</style>



