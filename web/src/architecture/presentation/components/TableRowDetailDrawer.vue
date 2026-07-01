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
    :size="detailDrawerSize"
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
              v-if="mode === 'read' && detailEditAccess !== 'unsupported'"
              type="primary"
              size="small"
              class="edit-btn"
              @click="handleToggleMode('edit')"
            >
              <el-icon><Edit /></el-icon>
              {{ t('common.edit') }}
            </el-button>
            <span
              v-else-if="mode === 'read'"
              class="edit-unsupported-hint"
            >
              {{ t('tableDetail.updateUnsupported') }}
            </span>
            <el-button
              v-if="mode === 'edit'"
              size="small"
              @click="handleToggleMode('read')"
            >
              {{ t('common.cancel') }}
            </el-button>
            <el-button
              v-if="mode === 'edit'"
              type="primary"
              size="small"
              :loading="submitting"
              :disabled="!isFormViewReady"
              @click="handleSubmit"
            >
              {{ t('common.save') }}
            </el-button>
            <el-button
              v-if="mode === 'edit'"
              size="small"
              :disabled="submitting || !canCopyWorkspaceUpdateInvocation"
              :title="t('formView.copyInvocationTitle')"
              @click="handleCopyWorkspaceUpdateInvocation"
            >
              <el-icon><CopyDocument /></el-icon>
              {{ t('formView.copyToWorkbench') }}
            </el-button>
            <el-button
              v-if="mode === 'edit' && featureFlags.scheduledTasks"
              size="small"
              :disabled="submitting || !canCreateScheduledUpdate"
              @click="openScheduledTaskDialog"
            >
              {{ t('formView.scheduledExecute') }}
            </el-button>
          </div>
          <!-- 布局切换按钮 -->
          <el-button
            v-if="mode === 'read'"
            size="small"
            text
            @click="toggleDetailLayout"
            :title="useGroupedDetailLayout ? t('tableDetail.switchToOriginalLayout') : t('tableDetail.switchToGroupedLayout')"
          >
            <el-icon><component :is="useGroupedDetailLayout ? List : Grid" /></el-icon>
            {{ useGroupedDetailLayout ? t('tableDetail.originalLayout') : t('tableDetail.groupedLayout') }}
          </el-button>
          <!-- 导航按钮（上一个/下一个） -->
          <div class="drawer-navigation" v-if="tableData && tableData.length > 1 && mode === 'read'">
            <el-button
              size="small"
              :disabled="currentIndex <= 0"
              @click="handleNavigate('prev')"
            >
              <el-icon><ArrowLeft /></el-icon>
              {{ t('tableDetail.previous') }}
            </el-button>
            <span class="nav-info">{{ (currentIndex >= 0 ? currentIndex + 1 : 0) }} / {{ tableData.length }}</span>
            <el-button
              size="small"
              :disabled="currentIndex >= tableData.length - 1"
              @click="handleNavigate('next')"
            >
              {{ t('tableDetail.next') }}
              <el-icon><ArrowRight /></el-icon>
            </el-button>
          </div>
        </div>
      </div>
    </template>

    <div class="detail-content">
      <!-- 详情模式 - 使用更美观的布局 -->
      <TableRowDetailReadTabs
        v-if="mode === 'read'"
        v-model="activeTab"
        :fields="fields"
        :link-fields="linkFields"
        :grouped-fields="groupedFields"
        :use-grouped-detail-layout="useGroupedDetailLayout"
        :rich-text-preview-height="RICH_TEXT_PREVIEW_HEIGHT"
        :full-code-path="fullCodePath"
        :row-id="rowId"
        :function-detail="currentFunctionDetail || editFunctionDetail"
        :get-field-value="getFieldValue"
        :set-rich-text-content-ref="setRichTextContentRef"
        :is-rich-text-expanded="isRichTextExpanded"
        :is-rich-text-overflow="isRichTextOverflow"
        :toggle-rich-text-expanded="toggleRichTextExpanded"
      />

      <!-- 编辑模式（复用 FormRenderer） -->
      <div v-else class="edit-form-wrapper" v-loading="submitting">
        <FormView
          v-if="editFunctionDetail && mode === 'edit' && editFormState.readiness === 'ready'"
          ref="formViewRef"
          :key="editFormKey"
          :function-detail="editFunctionDetail"
          :initial-data="filteredInitialData"
          :show-submit-button="false"
          :show-reset-button="false"
        />
        <el-empty v-else-if="!editFunctionDetail" :description="t('tableDetail.buildEditFormFailed')" />
        <el-empty
          v-else-if="editFormState.readiness === 'no-editable-fields'"
          :description="t('tableDetail.noEditableFields')"
        />
        <el-empty
          v-else-if="editFormState.readiness === 'missing-edit-values'"
          :description="t('tableDetail.missingEditValues')"
        />
        <el-empty
          v-else-if="editFormState.readiness === 'missing-row-data'"
          :description="t('tableDetail.missingRowData')"
        />
        <div v-else class="form-loading">
          <el-skeleton :rows="5" animated />
          <div style="text-align: center; margin-top: 16px; color: var(--el-text-color-secondary);">
            {{ t('tableDetail.loadingEditForm') }}
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="drawer-footer">
        <el-button @click="handleClose">{{ t('common.close') }}</el-button>
      </div>
    </template>
    <ScheduledTaskDialog
      v-if="showScheduledTaskDialog && canCreateScheduledUpdate"
      v-model="showScheduledTaskDialog"
      :full-code-path="fullCodePath"
      :function-detail="editFunctionDetail"
      table-mode
      fixed-action="table_update"
      :get-payload="buildScheduledUpdatePayload"
    />
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, computed, toRef, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { Edit, ArrowLeft, ArrowRight, Grid, List, CopyDocument } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import FormView from '@/architecture/presentation/views/FormView.vue'
import ScheduledTaskDialog from '@/architecture/presentation/components/ScheduledTaskDialog.vue'
import TableRowDetailReadTabs from './TableRowDetailReadTabs.vue'
import type { FieldConfig, FunctionDetail } from '../../domain/types'
import { buildDetailEditFormState } from '../composables/utils/workspaceDetailRuntime'
import { useTableRowDetailTabs } from '@/architecture/presentation/composables/useTableRowDetailTabs'
import { useTableRowDetailLayout } from '@/architecture/presentation/composables/useTableRowDetailLayout'
import { resolveTableDetailEditAccess } from '../views/utils/tableViewActionRuntime'
import { featureFlags } from '@/architecture/shared/config/features'
import {
  buildWorkspaceInvocationSnippet,
  copyTextToClipboard,
  filterEmptyInvocationParams,
} from '@/architecture/presentation/components/utils/workspaceInvocationSnippet'

interface Props {
  visible: boolean
  title: string
  mode: 'read' | 'edit'
  fields: FieldConfig[]
  rowData: Record<string, any> | null
  tableData?: any[]
  currentIndex?: number
  supportsEdit?: boolean
  canEdit?: boolean
  editFunctionDetail?: FunctionDetail | null
  currentFunctionDetail?: FunctionDetail | null  // 原始的 functionDetail（未修改的，用于操作日志）
  submitting?: boolean
  currentFunction?: any  // ServiceTree 节点，包含 full_code_path
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'update:mode', value: 'read' | 'edit'): void
  (e: 'navigate', direction: 'prev' | 'next'): void
  (e: 'submit', formViewRef: InstanceType<typeof FormView>): void
  (e: 'close'): void
}

const props = withDefaults(defineProps<Props>(), {
  tableData: () => [],
  currentIndex: -1,
  supportsEdit: false,
  canEdit: false,
  editFunctionDetail: null,
  currentFunctionDetail: null,
  submitting: false,
  currentFunction: null
})

const emit = defineEmits<Emits>()
const { t } = useI18n()

const DETAIL_DRAWER_MAX_WIDTH = 1360
const DETAIL_DRAWER_DESKTOP_RATIO = 0.78
const DETAIL_DRAWER_TABLET_RATIO = 0.88
const DETAIL_DRAWER_MOBILE_RATIO = 0.96

const formViewRef = ref<InstanceType<typeof FormView> | null>(null)
const showScheduledTaskDialog = ref(false)
const viewportWidth = ref(typeof window === 'undefined' ? 1440 : window.innerWidth)
const detailEditAccess = computed(() => {
  return resolveTableDetailEditAccess({
    supportsUpdate: props.supportsEdit
  })
})

const detailDrawerSize = computed(() => {
  const width = viewportWidth.value

  if (width <= 768) {
    return `${Math.floor(width * DETAIL_DRAWER_MOBILE_RATIO)}px`
  }

  if (width <= 1200) {
    return `${Math.floor(width * DETAIL_DRAWER_TABLET_RATIO)}px`
  }

  return `${Math.min(DETAIL_DRAWER_MAX_WIDTH, Math.floor(width * DETAIL_DRAWER_DESKTOP_RATIO))}px`
})

const updateViewportWidth = () => {
  viewportWidth.value = window.innerWidth
}

onMounted(() => {
  updateViewportWidth()
  window.addEventListener('resize', updateViewportWidth)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateViewportWidth)
})
const {
  activeTab
} = useTableRowDetailTabs({
  rowData: toRef(props, 'rowData')
})

const visible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})

const {
  RICH_TEXT_PREVIEW_HEIGHT,
  useGroupedDetailLayout,
  toggleDetailLayout,
  linkFields,
  groupedFields,
  setRichTextContentRef,
  isRichTextExpanded,
  isRichTextOverflow,
  toggleRichTextExpanded,
  getFieldValue
} = useTableRowDetailLayout({
  fields: toRef(props, 'fields'),
  rowData: toRef(props, 'rowData')
})

/**
 * 🔥 过滤 initialData，只包含 editFunctionDetail.request 中的字段
 * 这样可以确保传递给 FormView 的 initialData 只包含可编辑的字段
 */
const editFormState = computed(() => buildDetailEditFormState({
  rowData: props.rowData,
  editFunctionDetail: props.editFunctionDetail
}))

const filteredInitialData = computed(() => editFormState.value.initialData)

const editFormKey = computed(() => {
  const rowIdentity = props.rowData?.id ?? props.rowData?._id ?? 'unknown'
  const editableFieldSignature = editFormState.value.editableFieldCodes.join(',')
  return `detail-edit-${rowIdentity}-${props.mode}-${props.editFunctionDetail?.router || ''}-${props.editFunctionDetail?.id || ''}-${editableFieldSignature}`
})

const isFormViewReady = computed(() => {
  return props.mode === 'edit' && editFormState.value.readiness === 'ready' && !!formViewRef.value
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

const canCreateScheduledUpdate = computed(() => {
  return featureFlags.scheduledTasks &&
    props.mode === 'edit' &&
    isFormViewReady.value &&
    !!fullCodePath.value &&
    rowId.value > 0
})

const canCopyWorkspaceUpdateInvocation = computed(() => {
  return props.mode === 'edit' &&
    isFormViewReady.value &&
    !!fullCodePath.value &&
    rowId.value > 0
})

const handleToggleMode = (newMode: 'read' | 'edit') => {
  if (newMode === 'edit' && detailEditAccess.value === 'unsupported') {
    ElMessage.info(t('tableDetail.updateUnsupported'))
    return
  }

  if (newMode === 'edit') {
    switch (editFormState.value.readiness) {
      case 'missing-edit-detail':
        ElMessage.warning(t('tableDetail.editFormInitializing'))
        return
      case 'missing-row-data':
        ElMessage.warning(t('tableDetail.missingRowDataCannotEdit'))
        return
      case 'no-editable-fields':
        ElMessage.warning(t('tableDetail.noEditableFields'))
        return
      case 'missing-edit-values':
        ElMessage.warning(t('tableDetail.missingEditValuesCannotEdit'))
        return
    }
  }
  
  emit('update:mode', newMode)
}

const handleNavigate = (direction: 'prev' | 'next') => {
  emit('navigate', direction)
}

const handleSubmit = () => {
  // 只有 FormView 实例已挂载且编辑表单就绪时才允许提交
  if (!isFormViewReady.value || !formViewRef.value) {
    ElMessage.warning(t('tableDetail.editFormInitializing'))
    return
  }
  
  // 直接传递 formViewRef 给父组件
  emit('submit', formViewRef.value)
}

const openScheduledTaskDialog = () => {
  if (!canCreateScheduledUpdate.value) {
    ElMessage.warning(t('tableDetail.editFormNotReadySchedule'))
    return
  }
  showScheduledTaskDialog.value = true
}

const buildScheduledUpdatePayload = async (): Promise<Record<string, unknown>> => {
  if (!isFormViewReady.value || !formViewRef.value) {
    throw new Error(t('tableDetail.editFormNotReadySchedule'))
  }
  if (!rowId.value || rowId.value <= 0) {
    throw new Error(t('tableDetail.missingRowIdSchedule'))
  }

  const isValid = formViewRef.value.validateForm()
  if (!isValid) {
    throw new Error(t('formView.fixValidationFirst'))
  }

  const updates = await formViewRef.value.prepareUpdateData(filteredInitialData.value || {})
  if (Object.keys(updates).length === 0) {
    throw new Error(t('tableDetail.noChangesToSave'))
  }

  return {
    id: rowId.value,
    updates
  }
}

async function handleCopyWorkspaceUpdateInvocation(): Promise<void> {
  if (!isFormViewReady.value || !formViewRef.value || !fullCodePath.value) {
    ElMessage.warning(t('tableDetail.editFormNotReadyCopy'))
    return
  }
  if (!rowId.value || rowId.value <= 0) {
    ElMessage.warning(t('tableDetail.missingRowIdCopy'))
    return
  }

  const isValid = formViewRef.value.validateForm()
  if (!isValid) {
    ElMessage.warning(t('formView.fixValidationFirst'))
    return
  }

  try {
    const updates = filterEmptyInvocationParams(await formViewRef.value.prepareUpdateData(filteredInitialData.value || {}))
    if (Object.keys(updates).length === 0) {
      ElMessage.warning(t('tableDetail.noChangesToSave'))
      return
    }

    const snippet = buildWorkspaceInvocationSnippet({
      tool: 'run_table_update',
      resourcePath: fullCodePath.value,
      params: {
        body: [{ id: rowId.value, updates }],
      },
    })
    await copyTextToClipboard(snippet)
    ElMessage.success(t('tableDetail.updateInvocationCopied'))
  } catch {
    ElMessage.error(t('formView.copyFailedManual'))
  }
}

const handleClose = () => {
  emit('close')
}

// 暴露方法供父组件调用
defineExpose({
  formViewRef
})
</script>

<style scoped lang="scss">
.detail-drawer {
  background: var(--bg-page) !important;
}

.detail-drawer :deep(.el-drawer__header) {
  margin-bottom: 0;
  padding: 20px 32px;
  border-bottom: 1px solid var(--border-light);
  background: var(--bg-page);
}

.detail-drawer :deep(.el-drawer__body) {
  padding: 24px 32px;
  overflow: auto;
  background: var(--bg-page);
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

.edit-unsupported-hint {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
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

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: 10px;
}

.edit-form-wrapper {
  min-height: 400px;
}

</style>
