<!--
  WorkspaceDetailDrawer - 工作空间详情抽屉组件
  
  职责：
  - 详情展示
  - 详情编辑
  - 详情导航（上一条/下一条）
-->

<template>
  <el-drawer
    v-model="visible"
    :title="title"
    size="50%"
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
              v-if="mode === 'read' && canEdit"
              type="primary"
              size="small"
              @click="handleToggleMode('edit')"
            >
              <el-icon><Edit /></el-icon>
              编辑
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
        
        <!-- 字段网格（排除 link 字段） -->
        <div class="detail-fields-grid">
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

        <!-- 操作日志区域 -->
        <OperateLogSection
          :full-code-path="fullCodePath"
          :row-id="rowId"
          :function-detail="currentFunctionDetail || editFunctionDetail"
        />
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
import { Edit, ArrowLeft, ArrowRight } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
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

const formRendererRef = ref<InstanceType<typeof FormRenderer> | null>(null)
const isFormRendererReady = ref(false)

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
</style>



