<!--
  FormView - 表单视图
  统一架构的展示层组件
  
  ============================================
  📋 需求说明
  ============================================
  
  1. **表单渲染**：
     - 根据字段配置（`functionDetail.schema.form.request`）渲染表单
     - 支持多种字段类型（input、select、form、table 等）
     - 支持嵌套结构（form 嵌套 table，table 嵌套 form）
  
  2. **数据初始化**：
     - 编辑模式：使用 `initialData` 回显数据
     - 新增模式：从 URL 参数或字段默认值初始化
     - 支持从 URL 参数恢复表单状态（仅新增模式，`_tab=OnTableAddRow`）
  
  3. **URL 参数同步**：
     - 仅在新增模式下同步表单数据到 URL（`_tab=OnTableAddRow`）
     - 编辑模式和详情模式不同步 URL（`_tab=detail` 或不设置 `_tab`）
     - 同步时保留表格参数、搜索参数等其他参数
  
  4. **表单验证**：
     - 提交时验证所有字段
     - 验证错误使用字段的中文名称（`field.name`）
     - 验证错误显示在字段下方
  
  ============================================
  🎯 设计思路
  ============================================
  
  1. **分层架构**：
     - Presentation Layer：纯 UI 展示，不包含业务逻辑
     - 通过 FormApplicationService 调用业务逻辑
     - 从 FormStateManager 获取状态并渲染
  
  2. **数据流**：
     - 初始化：FormApplicationService.initializeForm → FormDomainService.initializeForm
     - 更新：用户输入 → updateFieldValue → StateManager → 事件总线
     - 提交：FormApplicationService.submitForm → FormDomainService.validateForm → API
  
  3. **URL 同步控制**：
     - 使用 `shouldSyncURL` computed 判断是否需要同步
     - 只在 `_tab=OnTableAddRow` 时同步 URL
     - 其他模式（`_tab=detail` 或不设置 `_tab`）不同步
  
  ============================================
  📝 关键功能
  ============================================
  
  1. **表单初始化**：
     - `initializeFormWithData`：编辑模式，使用 `initialData` 初始化
     - `initializeFormNormal`：新增模式，从 URL 或默认值初始化
  
  2. **字段渲染**：
     - 使用 `WidgetComponent` 渲染字段
     - 根据字段的 `widget.type` 选择对应的组件
     - 支持多种渲染模式（edit、response、table-cell、detail、search）
  
  3. **表单验证**：
     - `validateForm`：验证所有字段，返回验证结果
     - `getFieldError`：获取字段的验证错误信息
     - 验证错误使用字段的中文名称
  
  ============================================
  ⚠️ 注意事项
  ============================================
  
  1. **URL 同步时机**：
     - 只在新增模式下同步（`_tab=OnTableAddRow`）
     - 编辑模式和详情模式不同步，避免 URL 污染
  
  2. **初始数据优先级**：
     - 编辑模式：`initialData` > 字段默认值
     - 新增模式：URL 参数 > 字段默认值
  
  3. **字段 ID 生成**：
     - 使用 `:prop="field.code"` 确保 Element Plus 生成正确的 `id` 属性
     - 嵌套字段使用 `:prop="`${fieldPath}.${subField.code}`"` 格式
  
  4. **验证错误显示**：
     - 使用字段的中文名称（`field.name`）显示错误
     - 验证错误显示在字段下方，使用 `:error` 属性
-->

<template>
  <div
    class="form-view"
    :class="{ 'form-view-flat': flatSurface }"
    data-testid="form-view"
  >
    <div
      v-if="submitFeedback"
      :class="['submit-feedback', `is-${submitFeedback.type}`]"
      role="alert"
    >
      <span class="submit-feedback-message">{{ submitFeedback.message }}</span>
      <button type="button" class="submit-feedback-close" aria-label="关闭提示" @click="submitFeedback = null">
        ×
      </button>
    </div>
    <!-- 主内容区域：使用 flex 布局，左侧表单，右侧详情 -->
    <div class="form-view-container">
      <!-- 左侧：表单内容 -->
      <div class="form-view-main">
        <!-- 输入参数表单 -->
        <el-form
          v-if="visibleRequestFields.length > 0"
          :model="formData"
          label-position="left"
          :label-width="FORM_LABEL_WIDTH"
          class="function-form"
          data-testid="form-request"
        >
      <div class="section-title">输入参数</div>
      <template v-for="field in visibleRequestFields" :key="field.code">
        <div v-if="requestLabelsOnTop" class="form-field-label-top">
          <label class="field-label">
            {{ field.name }}
            <span v-if="isFieldRequired(field)" class="required">*</span>
          </label>
          <el-form-item :error="getFieldError(field.code)" class="form-item-no-label">
            <WidgetComponent
              :field="field"
              :value="getFieldValue(field.code)"
              :field-path="field.code"
              :form-renderer="formRendererContext"
              :function-method="functionDetail?.method || 'GET'"
              :function-router="functionDetail?.router || ''"
              @update:model-value="(v: FieldValue) => handleFieldUpdate(field.code, v)"
            />
          </el-form-item>
        </div>
        <el-form-item
          v-else
          :label="field.name"
          :required="isFieldRequired(field)"
          :error="getFieldError(field.code)"
        >
          <WidgetComponent
            :field="field"
            :value="getFieldValue(field.code)"
            :field-path="field.code"
            :form-renderer="formRendererContext"
            :function-method="functionDetail?.method || 'GET'"
            :function-router="functionDetail?.router || ''"
            @update:model-value="(v: FieldValue) => handleFieldUpdate(field.code, v)"
          />
        </el-form-item>
      </template>
    </el-form>

    <!-- 提交按钮 -->
    <div v-if="showSubmitButton || showResetButton" class="form-actions-section">
      <div class="form-actions-row">
        <el-button
          v-if="showSubmitButton"
          type="primary"
          size="large"
          @click="handleSubmit"
          :loading="submitting"
          class="submit-button-full-width"
          data-testid="form-submit"
        >
          <el-icon><Promotion /></el-icon>
          提交
        </el-button>
        <el-button
          v-if="canCreateScheduledSubmit"
          size="large"
          :disabled="submitting"
          @click="showScheduledTaskDialog = true"
          data-testid="form-schedule-submit"
        >
          <el-icon><Clock /></el-icon>
          定时提交
        </el-button>
        <el-button
          v-if="canCopyWorkspaceInvocation"
          size="large"
          :disabled="submitting"
          @click="handleCopyWorkspaceInvocation"
          data-testid="form-copy-workspace-invocation"
          title="复制后可粘贴到工作台让 AI 调用"
        >
          <el-icon><DocumentCopy /></el-icon>
          复制给工作台
        </el-button>
        <el-button v-if="showResetButton" size="large" @click="handleReset">
          <el-icon><RefreshLeft /></el-icon>
          重置
        </el-button>
        <el-button v-if="showDebugButton" size="large" @click="showDebugDialog = true" type="info">
          <el-icon><View /></el-icon>
          Debug
        </el-button>
      </div>
    </div>

    <!-- 输出参数展示：提交前就显示，显示"等待提交"标签 -->
    <div v-if="showInlineResponseSection" class="response-section">
      <div class="section-title">
        输出参数
        <el-tag v-if="!hasResponseData" type="info" size="small" style="margin-left: 12px">
          等待提交
        </el-tag>
        <el-tag v-else type="success" size="small" style="margin-left: 12px">
          已返回
        </el-tag>
      </div>
      <el-form 
        label-position="left"
        :label-width="FORM_LABEL_WIDTH"
        :class="{ 'is-empty': !hasResponseData }"
      >
        <template v-for="field in responseFields" :key="field.code">
          <div v-if="responseLabelsOnTop" class="form-field-label-top">
            <label class="field-label">{{ field.name }}</label>
            <el-form-item class="form-item-no-label">
              <WidgetComponent
                :field="field"
                :value="getResponseFieldValue(field.code)"
                :field-path="field.code"
                mode="response"
                :form-renderer="formRendererContext"
                :function-method="functionDetail?.method || 'GET'"
                :function-router="functionDetail?.router || ''"
              />
            </el-form-item>
          </div>
          <el-form-item v-else :label="field.name">
            <WidgetComponent
              :field="field"
              :value="getResponseFieldValue(field.code)"
              :field-path="field.code"
              mode="response"
              :form-renderer="formRendererContext"
              :function-method="functionDetail?.method || 'GET'"
              :function-router="functionDetail?.router || ''"
            />
          </el-form-item>
        </template>
      </el-form>
    </div>

    <div v-if="showDialogResponseButton" class="response-dialog-entry">
      <el-button type="primary" plain @click="responseDialogVisible = true">
        查看提交结果
      </el-button>
    </div>

    <!-- 执行信息（元数据）：显示函数执行耗时等信息，明确区分不是输出参数 -->
    <div v-if="showInlineMetadataSection" class="metadata-section">
      <div class="metadata-title">
        <el-icon class="metadata-icon"><InfoFilled /></el-icon>
        <span>执行信息</span>
      </div>
      <div class="metadata-content">
        <span class="metadata-label">执行耗时：</span>
        <span class="metadata-value">
          {{ formatCostTime(metadataTotalCost) }}
        </span>
      </div>
    </div>

    <el-dialog
      v-model="responseDialogVisible"
      title="提交结果"
      :width="responseDialogWidth"
      class="response-result-dialog"
      modal-class="response-result-dialog-overlay"
      align-center
      append-to-body
    >
      <div class="response-dialog-content">
        <el-form
          v-if="responseFields.length > 0"
          label-position="left"
          :label-width="FORM_LABEL_WIDTH"
          :class="{ 'is-empty': !hasResponseData }"
        >
          <template v-for="field in responseFields" :key="field.code">
            <div v-if="responseLabelsOnTop" class="form-field-label-top">
              <label class="field-label">{{ field.name }}</label>
              <el-form-item class="form-item-no-label">
                <WidgetComponent
                  :field="field"
                  :value="getResponseFieldValue(field.code)"
                  :field-path="field.code"
                  mode="response"
                  :form-renderer="formRendererContext"
                  :function-method="functionDetail?.method || 'GET'"
                  :function-router="functionDetail?.router || ''"
                />
              </el-form-item>
            </div>
            <el-form-item v-else :label="field.name">
              <WidgetComponent
                :field="field"
                :value="getResponseFieldValue(field.code)"
                :field-path="field.code"
                mode="response"
                :form-renderer="formRendererContext"
                :function-method="functionDetail?.method || 'GET'"
                :function-router="functionDetail?.router || ''"
              />
            </el-form-item>
          </template>
        </el-form>
        <el-empty v-else description="暂无输出数据" />

        <div v-if="showDialogMetadataSection" class="metadata-section response-dialog-metadata">
          <div class="metadata-title">
            <el-icon class="metadata-icon"><InfoFilled /></el-icon>
            <span>执行信息</span>
          </div>
          <div class="metadata-content">
            <span class="metadata-label">执行耗时：</span>
            <span class="metadata-value">
              {{ formatCostTime(metadataTotalCost) }}
            </span>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button type="primary" @click="responseDialogVisible = false">知道了</el-button>
      </template>
    </el-dialog>

    <!-- Debug 弹窗 -->
    <el-dialog
      v-model="showDebugDialog"
      title="Debug - 输入和输出数据"
      width="80%"
      :close-on-click-modal="false"
    >
      <el-tabs v-model="debugActiveTab">
        <!-- 输入参数 -->
        <el-tab-pane label="输入参数" name="request">
          <div class="debug-section">
            <div class="debug-header">
              <span class="debug-label">提交数据（实时）</span>
              <el-button
                size="small"
                type="primary"
                @click="copyToClipboard(debugRequestData)"
              >
                <el-icon><DocumentCopy /></el-icon>
                复制
              </el-button>
            </div>
            <el-input
              v-model="debugRequestData"
              type="textarea"
              :rows="20"
              readonly
              class="debug-json-input"
            />
          </div>
        </el-tab-pane>

        <!-- 输出参数 -->
        <el-tab-pane label="输出参数" name="response">
          <div class="debug-section">
            <div class="debug-header">
              <span class="debug-label">输出数据</span>
              <el-button
                v-if="debugResponseData"
                size="small"
                type="primary"
                @click="copyToClipboard(debugResponseData)"
              >
                <el-icon><DocumentCopy /></el-icon>
                复制
              </el-button>
            </div>
            <el-input
              v-if="debugResponseData"
              v-model="debugResponseData"
              type="textarea"
              :rows="20"
              readonly
              class="debug-json-input"
            />
            <el-empty v-else description="暂无输出数据，请先提交表单" />
          </div>
        </el-tab-pane>

        <!-- 原始状态 -->
        <el-tab-pane label="原始状态" name="raw">
          <div class="debug-section">
            <div class="debug-header">
              <span class="debug-label">FormDataStore 原始数据</span>
              <el-button
                size="small"
                type="primary"
                @click="copyToClipboard(debugRawData)"
              >
                <el-icon><DocumentCopy /></el-icon>
                复制
              </el-button>
            </div>
            <el-input
              v-model="debugRawData"
              type="textarea"
              :rows="20"
              readonly
              class="debug-json-input"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-dialog>

    <ScheduledTaskDialog
      v-if="showScheduledTaskDialog && canCreateScheduledSubmit"
      v-model="showScheduledTaskDialog"
      :full-code-path="scheduledFunctionPath"
      :function-detail="functionDetail"
      fixed-action="execute"
      :get-payload="buildScheduledSubmitPayload"
    />

      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, provide } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Promotion, RefreshLeft, View, DocumentCopy, InfoFilled, Clock } from '@element-plus/icons-vue'
import { ElIcon, ElTag, ElNotification, ElMessage, ElEmpty } from 'element-plus'
import { eventBus } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import WidgetComponent from '../widgets/WidgetComponent.vue'
import ScheduledTaskDialog from '@/architecture/presentation/components/ScheduledTaskDialog.vue'
import { getErrorMessage } from '@/architecture/shared/apiError'
import { TEMPLATE_TYPE } from '@/architecture/domain/constants/functionTypes'
import { getChangedFields } from '@/architecture/domain/utils/objectDiff'
import { featureFlags } from '@/architecture/shared/config/features'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import { formDataStoreKey } from '@/architecture/presentation/context/formRuntimeContext'
import { useFunctionParamInitialization } from '../composables/useFunctionParamInitialization'
import { useFormDebug } from '../composables/useFormDebug'
import { useFormParamURLSync } from '../composables/useFormParamURLSync'
import { useFormViewState } from '../composables/useFormViewState'
import { useFormViewLifecycle } from '../composables/useFormViewLifecycle'
import { createFormViewRuntime } from './utils/formViewRuntime'
import { FORM_LABEL_WIDTH } from '../utils/formLayout'
import { getFormRequestFields, getFormResponseFields } from '@/architecture/domain/utils/functionSchemaSelectors'
import { createDisplayAwareFieldValue } from '@/architecture/domain/utils/createFieldValue'
import { widgetInitializerRegistry } from '@/architecture/presentation/widgets/initializers/WidgetInitializerRegistry'
import type { IFormGateway } from '@/architecture/domain/interfaces/IFormGateway'
import {
  buildWorkspaceInvocationSnippet,
  copyTextToClipboard,
  filterEmptyInvocationParams,
} from '@/architecture/presentation/components/utils/workspaceInvocationSnippet'

const props = withDefaults(defineProps<{
  functionDetail?: FunctionDetail  // 🔥 改为可选，因为会在 onMounted 中主动获取
  showSubmitButton?: boolean  // 🔥 是否显示提交按钮（用于 FormDialog 等场景）
  showResetButton?: boolean  // 🔥 是否显示重置按钮
  showDebugButton?: boolean
  flatSurface?: boolean
  initialData?: Record<string, any>  // 🔥 初始数据（用于编辑模式）
  formGateway?: IFormGateway
  responseDisplayMode?: 'inline' | 'dialog'
}>(), {
  showSubmitButton: true,
  showResetButton: true,
  showDebugButton: true,
  flatSurface: false,
  initialData: () => ({}),
  responseDisplayMode: 'inline',
})

// 路由
const route = useRoute()
const router = useRouter()

const {
  formDataStore,
  responseDataStore,
  stateManager,
  domainService,
  applicationService
} = createFormViewRuntime({
  eventBus,
  formGateway: props.formGateway || serviceFactory.getFormGateway()
})
provide(formDataStoreKey, formDataStore)
const workspaceStateManager = serviceFactory.getWorkspaceStateManager()
const workspaceDomainService = serviceFactory.getWorkspaceDomainService()

// 🔥 内部维护 functionDetail（在 onMounted 中主动获取）
const functionDetail = ref<FunctionDetail | null>(props.functionDetail || null)
const {
  formData,
  requestFields,
  visibleRequestFields,
  responseFields,
  requestLabelsOnTop,
  responseLabelsOnTop,
  hasResponseData,
  responseMetadata,
  submitting,
  formRendererContext,
  getFieldValue,
  getFieldError,
  getResponseFieldValue,
  isFieldRequired,
  handleFieldUpdate,
} = useFormViewState({
  functionDetail,
  stateManager,
  domainService,
  applicationService
})

const currentFunctionNode = computed(() => {
  return workspaceStateManager.getCurrentFunction()
})
const showScheduledTaskDialog = ref(false)
const scheduledFunctionPath = computed(() => {
  return functionDetail.value?.full_code_path || currentFunctionNode.value?.full_code_path || ''
})
const canCreateScheduledSubmit = computed(() => {
  return featureFlags.scheduledTasks && props.showSubmitButton && !!scheduledFunctionPath.value
})
const canCopyWorkspaceInvocation = computed(() => {
  return props.showSubmitButton && !!scheduledFunctionPath.value
})

// 🔥 移除 formInitialData computed，改为使用统一的数据初始化框架
// URL 参数会在 useFunctionParamInitialization 中统一处理

// 🔥 格式化耗时显示
const formatCostTime = (milliseconds: number): string => {
  if (milliseconds < 1000) {
    return `${milliseconds}ms`
  } else if (milliseconds < 60000) {
    return `${(milliseconds / 1000).toFixed(2)}s`
  } else {
    const minutes = Math.floor(milliseconds / 60000)
    const seconds = ((milliseconds % 60000) / 1000).toFixed(2)
    return `${minutes}分${seconds}秒`
  }
}

const {
  showDebugDialog,
  debugActiveTab,
  debugRequestData,
  debugResponseData,
  debugRawData,
  copyToClipboard,
} = useFormDebug({
  stateManager,
  domainService,
  requestFields,
})

const submitFeedback = ref<{ type: 'success' | 'error'; message: string } | null>(null)
const responseDialogVisible = ref(false)
const responseInDialog = computed(() => props.responseDisplayMode === 'dialog')
const responseDialogWidth = computed(() => 'min(560px, calc(100vw - 24px))')
const showInlineResponseSection = computed(() => responseFields.value.length > 0 && !responseInDialog.value)
const showDialogResponseButton = computed(() => responseInDialog.value && responseFields.value.length > 0 && hasResponseData.value)
const showInlineMetadataSection = computed(() => {
  return !responseInDialog.value && responseMetadata.value && responseMetadata.value.total_cost_mill !== undefined
})
const showDialogMetadataSection = computed(() => {
  return responseInDialog.value && responseMetadata.value && responseMetadata.value.total_cost_mill !== undefined
})
const metadataTotalCost = computed(() => Number(responseMetadata.value?.total_cost_mill || 0))

const submitForm = async (): Promise<boolean> => {
  try {
    submitFeedback.value = null

    if (!functionDetail.value) {
      ElMessage.error('函数详情未加载完成，请稍后重试')
      return false
    }
    await applicationService.submitForm(functionDetail.value)
    
    // 🔥 如果执行到这里，说明 API 调用成功（request.ts 的响应拦截器在 code !== 0 时会 reject）
    // request.ts 在 code === 0 时返回 data，所以这里 response 是 data 部分
    // 显示成功通知
    ElNotification.success({
      title: '提交成功',
      message: '操作成功',
      duration: 3000
    })

    return true
  } catch (error: any) {
    const errorMessage = getErrorMessage(error, '提交失败，请稍后重试')
    submitFeedback.value = {
      type: 'error',
      message: errorMessage
    }

    return false
  }
}

const handleSubmit = async (): Promise<void> => {
  const success = await submitForm()
  if (success) {
    submitFeedback.value = null
    if (responseInDialog.value && responseFields.value.length > 0) {
      responseDialogVisible.value = true
    }
  }
}

const handleReset = (): void => {
  submitFeedback.value = null
  responseDialogVisible.value = false
  lifecycle.resetFormRuntimeState()
  // 重新初始化表单
  const fields = requestFields.value
  if (fields.length > 0) {
    applicationService.initializeForm(fields)
  }
}

async function handleCopyWorkspaceInvocation(): Promise<void> {
  if (!functionDetail.value || !scheduledFunctionPath.value) {
    ElMessage.warning('函数详情未加载完成，请稍后重试')
    return
  }

  try {
    const params = filterEmptyInvocationParams(prepareSubmitDataWithTypeConversion())
    const snippet = buildWorkspaceInvocationSnippet({
      tool: 'run_form_submit',
      resourcePath: scheduledFunctionPath.value,
      params,
    })
    await copyTextToClipboard(snippet)
    ElMessage.success('已复制给工作台使用的函数调用')
  } catch {
    ElMessage.error('复制失败，请手动复制')
  }
}

/**
 * 准备提交数据（带类型转换）
 * 🔥 用于表单提交场景（新增/创建），返回所有字段的数据
 * 这个方法会被 FormDialog 等外部组件调用
 * 
 * ⚠️ 注意：这个方法只用于表单提交（新增场景），不用于表格更新
 * 表格更新应该使用 `prepareUpdateData` 方法，只返回变更的字段
 * 
 * @returns 提交数据对象（包含所有字段）
 */
function prepareSubmitDataWithTypeConversion(): Record<string, any> {
  const request = getFormRequestFields(functionDetail.value)
  if (request.length === 0) {
    return {}
  }
  
  // 使用 domainService 的 getSubmitData 方法递归收集所有字段的数据
  const submitData = domainService.getSubmitData(request)

  return submitData
}

/**
 * 准备更新数据（只返回变更的字段）
 * 🔥 用于表格更新场景，只返回用户实际修改的字段
 * 
 * @param oldValues 旧值对象（完整的记录数据）
 * @returns 只包含变更字段的数据对象
 */
async function prepareUpdateData(oldValues: Record<string, any>): Promise<Record<string, any>> {
  const request = getFormRequestFields(functionDetail.value)
  if (request.length === 0) {
    return {}
  }
  
  // 先获取所有字段的数据
  const allSubmitData = domainService.getSubmitData(request)
  
  // 使用 getChangedFields 过滤出只变更的字段
  const { updates } = getChangedFields(oldValues, allSubmitData)

  return updates
}

/**
 * 验证表单
 * 这个方法会被 FormDialog 等外部组件调用
 */
function validateForm(): boolean {
  if (!functionDetail.value) {
    return false
  }
  
  const fields = getFormRequestFields(functionDetail.value) as FieldConfig[]
  return domainService.validateForm(fields)
}

async function buildScheduledSubmitPayload(): Promise<Record<string, unknown>> {
  if (!functionDetail.value) {
    throw new Error('函数详情未加载完成，请稍后重试')
  }

  const isValid = validateForm()
  if (!isValid) {
    throw new Error('请先修正表单校验错误')
  }

  return prepareSubmitDataWithTypeConversion()
}

async function applyOperateLog(requestBody: Record<string, any>, responseBody?: Record<string, any> | null): Promise<void> {
  const nextData = new Map<string, FieldValue>()
  requestFields.value.forEach((field: FieldConfig) => {
    if (Object.prototype.hasOwnProperty.call(requestBody, field.code)) {
      nextData.set(field.code, createDisplayAwareFieldValue(requestBody[field.code], field))
    }
  })
  if (nextData.size > 0) {
    stateManager.setState({ data: nextData })
    await nextTick()
    await hydrateCurrentWidgetDisplays('initialData')
  }

  if (responseBody) {
    const responseResult = await hydrateOperateLogResponse(unwrapOperateLogResponseBody(responseBody))
    stateManager.setResponse(responseResult)
    stateManager.setMetadata({
      trace_id: responseBody.trace_id,
      version: responseBody.version,
      total_cost_mill: responseBody.total_cost_mill,
    })
  }

  ElMessage.success('已回填本次执行记录')
}

async function hydrateOperateLogResponse(responseResult: Record<string, unknown>): Promise<Record<string, unknown>> {
  const detail = functionDetail.value
  if (!detail) {
    return responseResult
  }

  const hydrated: Record<string, unknown> = { ...responseResult }
  const requestValues = buildAllFormDataForInitializer()

  for (const field of getFormResponseFields(detail) as FieldConfig[]) {
    if (!Object.prototype.hasOwnProperty.call(responseResult, field.code)) {
      continue
    }

    const hydrationField = resolveOperateLogResponseHydrationField(detail, field)
    const currentValue = createDisplayAwareFieldValue(responseResult[field.code], hydrationField)
    try {
      const initializedValue = await widgetInitializerRegistry.initialize({
        field: hydrationField,
        currentValue,
        allFormData: requestValues,
        functionDetail: detail,
        initSource: 'initialData',
        fieldPath: field.code,
      })
      hydrated[field.code] = initializedValue
    } catch {
      hydrated[field.code] = currentValue
    }
  }

  return hydrated
}

function resolveOperateLogResponseHydrationField(detail: FunctionDetail, responseField: FieldConfig): FieldConfig {
  if (Array.isArray(responseField.callbacks) && responseField.callbacks.length > 0) {
    return responseField
  }

  const requestField = (getFormRequestFields(detail) as FieldConfig[])
    .find((field) => field.code === responseField.code)
  if (!requestField || !Array.isArray(requestField.callbacks) || requestField.callbacks.length === 0) {
    return responseField
  }

  return {
    ...responseField,
    callbacks: requestField.callbacks,
    depend_on: responseField.depend_on || requestField.depend_on,
  }
}

function buildAllFormDataForInitializer(): Record<string, FieldValue> {
  const allValues: Record<string, FieldValue> = {}
  const state = stateManager.getState()
  state.data?.forEach((value: FieldValue, key: string) => {
    allValues[key] = value
  })
  return allValues
}

function unwrapOperateLogResponseBody(responseBody: Record<string, any>): Record<string, unknown> {
  for (const key of ['result', 'data', 'response']) {
    const value = responseBody[key]
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      return value as Record<string, unknown>
    }
    if (value !== undefined && value !== null && value !== '') {
      return { result: value }
    }
  }
  return responseBody
}

// 🔥 暴露方法给外部组件调用（兼容 FormRenderer 的接口）
defineExpose({
  submitForm,
  prepareSubmitDataWithTypeConversion,  // 表单提交（新增场景）
  prepareUpdateData,                     // 表格更新（更新场景，只返回变更的字段）
  validateForm,
  applyOperateLog
})


// 🔥 使用统一的数据初始化框架
const { initialize: initializeParams, hydrateCurrentWidgetDisplays } = useFunctionParamInitialization({
  functionDetail: computed(() => functionDetail.value),
  formDataStore: {
    getValue: (fieldCode: string) => formDataStore.getValue(fieldCode),
    setValue: (fieldCode: string, value: any) => formDataStore.setValue(fieldCode, value),
    getAllValues: () => {
      const allValues: Record<string, any> = {}
      const state = stateManager.getState()
      if (state.data) {
        state.data.forEach((value: FieldValue, key: string) => {
          allValues[key] = value
        })
      }
      return allValues
    },
    clear: () => formDataStore.clear()
  }
})

// 🔥 使用 Form 参数 URL 同步
// ⚠️ 关键：必须直接从 formDataStore.data 获取数据，确保响应式追踪
const formDataStoreForURLSync = {
  getValue: (fieldCode: string) => formDataStore.getValue(fieldCode),
  getAllValues: () => {
    // 🔥 直接从 formDataStore.data 获取，确保响应式追踪
    const allValues: Record<string, FieldValue> = {}
    const data = formDataStore.data
    if (data) {
      data.forEach((value: FieldValue, key: string) => {
        allValues[key] = value
      })
    }
    return allValues
  }
}

/**
 * 🔥 判断是否应该启用 URL 同步
 * 独立 Form 函数页面和 Table 新增模式需要同步 URL 参数。
 * 编辑模式、详情模式、合成表单不需要同步。
 */
const shouldSyncURL = computed(() => {
  const currentTab = Array.isArray(route.query._tab) ? route.query._tab[0] : route.query._tab
  if (currentTab === 'OnTableAddRow') {
    return true
  }
  if (currentTab) {
    return false
  }

  const detail = functionDetail.value
  const hasInitialData = Object.keys(props.initialData || {}).length > 0
  return !hasInitialData &&
    detail?.template_type === TEMPLATE_TYPE.FORM &&
    detail.id !== undefined &&
    detail.id !== null &&
    detail.id !== 0
})

const { watchFormData } = useFormParamURLSync({
  functionDetail: computed(() => functionDetail.value),
  formDataStore: formDataStoreForURLSync,
  enabled: shouldSyncURL,
  debounceMs: 300
})
const lifecycle = useFormViewLifecycle({
  eventBus,
  functionDetail,
  propsFunctionDetail: () => props.functionDetail,
  propsInitialData: () => props.initialData,
  formDataStore,
  responseDataStore,
  stateManager,
  domainService,
  applicationService,
  workspaceStateManager,
  workspaceDomainService,
  initializeParams,
  hydrateCurrentWidgetDisplays,
  watchFormData
})
</script>

<style scoped lang="scss">
.form-view {
  padding: 0;
}

.form-view-flat {
  border: none !important;
  border-radius: 0 !important;
  background: transparent !important;
  box-shadow: none !important;
}

.submit-feedback {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 12px;
  padding: 8px 10px;
  border: 1px solid var(--el-color-danger-light-7);
  border-radius: 6px;
  background: var(--el-color-danger-light-9);
  color: var(--el-color-danger-dark-2);
  font-size: 13px;
  line-height: 1.45;
}

.submit-feedback-message {
  min-width: 0;
  word-break: break-word;
}

.submit-feedback-close {
  flex: 0 0 auto;
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: currentColor;
  font-size: 16px;
  line-height: 18px;
  cursor: pointer;
  opacity: 0.7;
}

.submit-feedback-close:hover {
  background: color-mix(in srgb, var(--el-color-danger) 10%, transparent);
  opacity: 1;
}

/* 长 label：label 在上方 */
.form-field-label-top {
  margin-bottom: 18px;

  .field-label {
    display: block;
    font-size: 14px;
    color: var(--text-primary);
    margin-bottom: 10px;
    line-height: 1.4;
    text-align: left;
    font-weight: 600;

    .required {
      color: var(--el-color-danger);
      margin-left: 2px;
    }
  }

  .form-item-no-label {
    margin-bottom: 0;

    :deep(.el-form-item__label) {
      display: none;
    }
    :deep(.el-form-item__content) {
      margin-left: 0 !important;
    }
  }
}

/* 短 label：右对齐，靠近右侧输入框，减少 label 与输入框间距 */
.form-view-main :deep(.el-form .el-form-item:not(.form-item-no-label) .el-form-item__label) {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  text-align: right;
  padding-right: 8px;
}

.form-view-container {
  display: flex;
  gap: 20px;
  align-items: flex-start;
}

.form-view-main {
  flex: 1;
  min-width: 0; // 防止 flex 子元素溢出
  padding: 32px 34px 34px;
  background: var(--bg-primary);
  border: 1px solid var(--border-light);
  border-radius: 12px;
  box-shadow: 0 4px 16px rgba(var(--color-primary-rgb), 0.04);
}

.form-view-flat .form-view-main {
  padding: 12px 0 6px;
  border: none;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.form-view-flat :deep(.editor-container) {
  border: 1px solid var(--app-auth-input-border) !important;
  border-radius: 12px !important;
  background: var(--app-auth-input-bg) !important;
  box-shadow: none !important;
  overflow: hidden;
}

.form-view-flat :deep(.editor-content) {
  background: transparent !important;
  border: none !important;
  border-radius: 0 !important;
  overflow: hidden;
}

.form-view-flat :deep(.editor-container:hover) {
  border-color: rgba(var(--el-color-primary-rgb), 0.42) !important;
  box-shadow: var(--app-auth-input-shadow-hover) !important;
}

.form-view-flat :deep(.editor-container:focus-within) {
  border-color: var(--el-color-primary) !important;
  box-shadow: var(--app-auth-input-shadow-focus) !important;
}

.form-view-flat :deep(.editor-content .ProseMirror) {
  padding: 16px 18px !important;
}

.form-view-flat :deep(.preview-content) {
  padding: 16px 18px !important;
  background: transparent !important;
  border: none !important;
  border-radius: 0 !important;
}

.form-view-flat .form-view-main :deep(.vditor) {
  border: none;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.form-view-flat .form-view-main :deep(.vditor:hover),
.form-view-flat .form-view-main :deep(.vditor:focus-within) {
  border: none;
  box-shadow: none;
}

.form-view-flat .form-view-main :deep(.vditor-toolbar) {
  background: transparent;
  border-bottom: none;
  padding: 10px 0 12px;
}

.form-view-flat .form-view-main :deep(.vditor-content),
.form-view-flat .form-view-main :deep(.vditor-ir),
.form-view-flat .form-view-main :deep(.vditor-wysiwyg),
.form-view-flat .form-view-main :deep(.vditor-sv),
.form-view-flat .form-view-main :deep(.vditor-reset),
.form-view-flat .form-view-main :deep(.vditor-ir pre.vditor-reset),
.form-view-flat .form-view-main :deep(.vditor-ir .vditor-reset),
.form-view-flat .form-view-main :deep(.vditor-resize) {
  border: none;
  outline: none;
  box-shadow: none;
  background: transparent;
}

.form-view-flat .form-view-main :deep(.vditor-counter) {
  background: transparent;
  border-top: none;
}

.form-view-flat .form-view-main :deep(.vditor-ir pre.vditor-reset) {
  padding-left: 14px;
  padding-right: 14px;
}


/* Debug 弹窗样式 */
.debug-section {
  margin-bottom: 20px;
}

.debug-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.debug-label {
  font-weight: 600;
  color: var(--el-text-color-regular);
}

.debug-json-input {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  font-size: 13px;
  line-height: 1.5;
}

.debug-json-input :deep(.el-textarea__inner) {
  background-color: var(--app-code-bg);
  border: 1px solid var(--app-code-border);
  color: var(--app-code-text);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  resize: none;
}

.debug-json-input :deep(.el-textarea__inner):focus {
  border-color: var(--el-color-primary);
  background-color: var(--el-fill-color-blank);
}

.section-title {
  font-size: 22px;
  font-weight: 700;
  margin-bottom: 26px;
  color: var(--text-primary);
  letter-spacing: -0.2px;
}

.form-actions {
  margin-top: 20px;
  display: flex;
  gap: 10px;
}

.form-actions-section {
  margin-top: 28px;
  padding-top: 20px;
  border-top: 1px solid var(--app-auth-card-border);
}

.form-actions-row {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.form-actions-row :deep(.el-button) {
  height: 44px;
  border-radius: 12px;
  padding: 0 18px;
  font-weight: 600;
  border: 1px solid var(--app-auth-input-border);
  background: var(--app-auth-input-bg);
  box-shadow: none;
  transition: all 0.3s ease;
}

.form-actions-row :deep(.el-button:hover) {
  transform: translateY(-1px);
  border-color: rgba(var(--el-color-primary-rgb), 0.42);
  color: var(--el-color-primary);
  box-shadow: var(--app-auth-input-shadow-hover);
}

.form-actions-row :deep(.el-button--primary) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  color: #fff;
  box-shadow: var(--app-auth-primary-shadow);
}

.form-actions-row :deep(.el-button--primary:hover) {
  color: #fff;
  border-color: var(--el-color-primary);
  background: var(--el-color-primary);
  box-shadow: var(--app-auth-primary-shadow-hover);
}

.submit-button-full-width {
  flex: 1;
}

.response-section {
  margin-top: 36px;
  padding-top: 24px;
  border-top: 1px solid var(--app-auth-card-border);
}

.response-section .is-empty {
  opacity: 0.6;
}

.response-dialog-entry {
  margin-top: 18px;
  display: flex;
  justify-content: flex-end;
}

.response-dialog-content {
  max-height: min(68vh, 640px);
  overflow-y: auto;
  overflow-x: hidden;
  padding-right: 4px;
  -webkit-overflow-scrolling: touch;
}

.response-dialog-content :deep(.el-form-item:last-child) {
  margin-bottom: 0;
}

:global(.response-result-dialog-overlay .el-overlay-dialog) {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}

:global(.response-result-dialog) {
  margin: 0 auto;
}

:global(.response-result-dialog .el-dialog__body) {
  padding-top: 8px;
}

.response-dialog-metadata {
  margin-top: 18px;
}

.metadata-section {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--app-auth-card-border);
}

.metadata-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-regular);
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.metadata-icon {
  color: var(--el-color-info);
  font-size: 16px;
}

.metadata-content {
  padding: 12px 16px;
  background: var(--app-auth-card-bg-strong);
  border: 1px solid var(--app-auth-card-border);
  border-radius: 16px;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.metadata-label {
  color: var(--el-text-color-regular);
  font-weight: 500;
}

.metadata-value {
  color: var(--el-color-primary);
  font-weight: 600;
  font-size: 14px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.loading-container {
  padding: 20px;
}

.empty-container {
  padding: 40px 0;
}

.form-view-main :deep(.function-form > .el-form-item) {
  margin-bottom: 22px;
}

.form-view-main :deep(.el-form .el-form-item__label) {
  color: var(--text-primary);
  font-weight: 500;
}

.form-view-main :deep(.vditor) {
  border-radius: var(--border-radius-input);
  border: 1px solid var(--border-light);
  background: transparent;
  box-shadow: none;
  overflow: hidden;
  transition: all 0.3s ease;
}

.form-view-main :deep(.vditor:hover) {
  background: var(--el-fill-color-light);
  border-color: var(--border-base);
}

.form-view-main :deep(.vditor:focus-within) {
  background: transparent;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(var(--color-primary-rgb), 0.2);
}

.form-view-main :deep(.vditor-toolbar) {
  background: var(--bg-primary);
  border-bottom: 1px solid var(--border-light);
  padding: 10px 12px;
}

.form-view-main :deep(.vditor-toolbar button) {
  border-radius: 8px;
  color: var(--text-secondary);
  transition: all 0.2s ease;
}

.form-view-main :deep(.vditor-toolbar button:hover) {
  color: var(--text-primary);
  background: var(--el-fill-color-light);
}

.form-view-main :deep(.vditor-toolbar button.vditor-toolbar__item--current) {
  color: var(--color-primary);
  background: var(--el-fill-color);
}

.form-view-main :deep(.vditor-toolbar .vditor-toolbar__divider) {
  background: var(--border-light);
}

.form-view-main :deep(.vditor-ir),
.form-view-main :deep(.vditor-ir pre.vditor-reset),
.form-view-main :deep(.vditor-content),
.form-view-main :deep(.vditor-reset) {
  background: transparent;
  color: var(--text-primary);
}

.form-view-main :deep(.vditor-ir pre.vditor-reset) {
  min-height: 300px;
  padding: 20px 18px;
}

.form-view-main :deep(.vditor-counter) {
  background: var(--bg-primary);
  border-top: 1px solid var(--border-light);
  color: var(--text-secondary);
}

.form-view-main :deep(.files-editor-shell),
.form-view-main :deep(.upload-area),
.form-view-main :deep(.file-upload-trigger) {
  border-radius: var(--border-radius-input);
  border-color: var(--border-light);
  background: transparent;
}

.form-view-main :deep(.upload-area:hover),
.form-view-main :deep(.upload-area.is-dragging) {
  border-color: var(--border-base);
  background: var(--el-fill-color-light);
}

@media (max-width: 700px) {
  .form-view-main {
    padding: 22px 18px 24px;
    border-radius: 16px;
  }

  .form-view-container {
    display: block;
  }

  .form-view-main :deep(.el-form .el-form-item:not(.form-item-no-label)) {
    display: block;
  }

  .form-view-main :deep(.el-form .el-form-item:not(.form-item-no-label) .el-form-item__label) {
    justify-content: flex-start;
    width: auto !important;
    height: auto;
    margin-bottom: 8px;
    padding-right: 0;
    text-align: left;
  }

  .form-view-main :deep(.el-form .el-form-item:not(.form-item-no-label) .el-form-item__content) {
    margin-left: 0 !important;
  }

  .form-actions-section {
    margin-top: 22px;
    padding-top: 18px;
  }

  .form-actions-row {
    display: grid;
    grid-template-columns: 1fr;
  }

  .form-actions-row :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }

  .response-dialog-entry {
    justify-content: stretch;
  }

  .response-dialog-entry :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }

  :global(.response-result-dialog) {
    width: min(560px, calc(100vw - 24px)) !important;
    margin: 0 auto !important;
    transform: none;
    border-radius: 16px;
  }

  :global(.response-result-dialog .el-dialog__header) {
    padding: 16px 18px 8px;
  }

  :global(.response-result-dialog .el-dialog__body) {
    max-height: 62vh;
    overflow: hidden;
    padding: 8px 18px 16px;
  }

  :global(.response-result-dialog .el-dialog__footer) {
    padding: 0 18px 16px;
  }

  :global(.response-result-dialog .el-dialog__footer .el-button) {
    width: 100%;
    margin-left: 0;
  }

  :global(.response-result-dialog .el-form-item:not(.form-item-no-label)) {
    display: block;
    margin-bottom: 18px;
  }

  :global(.response-result-dialog .el-form-item:not(.form-item-no-label) .el-form-item__label) {
    justify-content: flex-start;
    width: auto !important;
    height: auto;
    margin-bottom: 8px;
    padding-right: 0;
    text-align: left;
  }

  :global(.response-result-dialog .el-form-item:not(.form-item-no-label) .el-form-item__content) {
    width: 100%;
    margin-left: 0 !important;
  }

  :global(.response-result-dialog .form-field-label-top .field-label) {
    text-align: left;
  }

  :global(.response-result-dialog .form-item-no-label) {
    display: block;
    margin-bottom: 18px;
  }

  :global(.response-result-dialog .form-item-no-label .el-form-item__content) {
    width: 100%;
    margin-left: 0 !important;
  }
}
</style>
