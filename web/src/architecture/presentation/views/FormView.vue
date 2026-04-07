<!--
  FormView - 表单视图
  新架构的展示层组件
  
  ============================================
  📋 需求说明
  ============================================
  
  1. **表单渲染**：
     - 根据字段配置（`functionDetail.request`）渲染表单
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
  <div class="form-view">
    <!-- ⭐ 权限不足提示：使用 PermissionDeniedView 组件 -->
    <PermissionDeniedView v-if="permissionError" />

    <!-- 主内容区域：使用 flex 布局，左侧表单，右侧详情 -->
    <div class="form-view-container">
      <!-- 左侧：表单内容 -->
      <div class="form-view-main">
        <!-- 请求参数表单 -->
        <el-form
          v-if="requestFields.length > 0"
          :model="formData"
          label-position="left"
          label-width="90px"
          class="function-form"
        >
      <div class="section-title">请求参数</div>
      <template v-for="field in requestFields" :key="field.code">
        <!-- 有任一 label 超过 8 字：全部上方 -->
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
        <!-- 无超过 8 字的 label：全部左侧一行 -->
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
        <!-- ⭐ 提交按钮：需要 form:write 权限 -->
        <!-- 如果没有权限，显示禁用状态的按钮，点击后跳转到权限申请页面 -->
        <el-button
          v-if="showSubmitButton && canSubmit"
          type="primary"
          size="large"
          @click="handleSubmit"
          :loading="submitting"
          class="submit-button-full-width"
        >
          <el-icon><Promotion /></el-icon>
          提交
        </el-button>
        <el-button
          v-if="showSubmitButton && canSubmit"
          type="primary"
          plain
          size="large"
          @click="showScheduledTaskDialog = true"
          :disabled="!currentFunctionNode?.full_code_path"
        >
          <el-icon><Clock /></el-icon>
          定时执行
        </el-button>
        <el-button
          v-else-if="showSubmitButton"
          type="default"
          size="large"
          :disabled="false"
          class="submit-button-full-width action-btn-no-permission"
          @click="handleApplyPermissionForSubmit"
        >
          <el-icon><Lock /></el-icon>
          提交（需{{ getPermissionShortName('function:write') }}）
        </el-button>
        <el-button v-if="showResetButton" size="large" @click="handleReset">
          <el-icon><RefreshLeft /></el-icon>
          重置
        </el-button>
        <el-button size="large" @click="showDebugDialog = true" type="info">
          <el-icon><View /></el-icon>
          Debug
        </el-button>
      </div>
    </div>

    <!-- 响应参数展示：提交前就显示，显示"等待提交"标签 -->
    <div v-if="responseFields.length > 0" class="response-section">
      <div class="section-title">
        响应参数
        <el-tag v-if="!hasResponseData" type="info" size="small" style="margin-left: 12px">
          等待提交
        </el-tag>
        <el-tag v-else type="success" size="small" style="margin-left: 12px">
          已返回
        </el-tag>
      </div>
      <el-form 
        label-position="left"
        label-width="90px"
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
              />
            </el-form-item>
          </div>
          <el-form-item v-else :label="field.name">
            <WidgetComponent
              :field="field"
              :value="getResponseFieldValue(field.code)"
              :field-path="field.code"
              mode="response"
            />
          </el-form-item>
        </template>
      </el-form>
    </div>

    <!-- 执行信息（元数据）：显示函数执行耗时等信息，明确区分不是响应参数 -->
    <div v-if="responseMetadata && responseMetadata.total_cost_mill !== undefined" class="metadata-section">
      <div class="metadata-title">
        <el-icon class="metadata-icon"><InfoFilled /></el-icon>
        <span>执行信息</span>
      </div>
      <div class="metadata-content">
        <span class="metadata-label">执行耗时：</span>
        <span class="metadata-value">
          {{ formatCostTime(responseMetadata.total_cost_mill) }}
        </span>
      </div>
    </div>

    <!-- Debug 弹窗 -->
    <el-dialog
      v-model="showDebugDialog"
      title="Debug - 请求和响应数据"
      width="80%"
      :close-on-click-modal="false"
    >
      <el-tabs v-model="debugActiveTab">
        <!-- 请求参数 -->
        <el-tab-pane label="请求参数" name="request">
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

        <!-- 响应参数 -->
        <el-tab-pane label="响应参数" name="response">
          <div class="debug-section">
            <div class="debug-header">
              <span class="debug-label">响应数据</span>
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
            <el-empty v-else description="暂无响应数据，请先提交表单" />
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

    <!-- 定时执行弹窗：以当前表单参数为 payload -->
    <ScheduledTaskDialog
      v-model="showScheduledTaskDialog"
      :full-code-path="currentFunctionNode?.full_code_path ?? ''"
      :get-payload="prepareSubmitDataWithTypeConversion"
      @success="onScheduledTaskCreated"
    />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch, ref, withDefaults, provide } from 'vue'
import type { ComputedRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Promotion, RefreshLeft, View, DocumentCopy, InfoFilled, Lock, Document, List, User, Clock } from '@element-plus/icons-vue'
import { ElIcon, ElTag, ElNotification, ElMessage, ElAlert, ElMessageBox, ElText, ElCheckbox, ElCard, ElEmpty } from 'element-plus'
import { eventBus, FormEvent, WorkspaceEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import { apiClient } from '../../infrastructure/apiClient'
import WidgetComponent from '../widgets/WidgetComponent.vue'
import { Logger } from '@/core/utils/logger'
import { getErrorMessage } from '@/utils/apiError'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { getChangedFields } from '@/utils/objectDiff'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import { hasAnyRequiredRule } from '@/core/utils/validationUtils'
import { formDataStoreKey } from '@/core/stores-v2/formData'
import { useFunctionParamInitialization } from '../composables/useFunctionParamInitialization'
import { useFormParamURLSync } from '../composables/useFormParamURLSync'
import { hasPermission, FormPermission, FunctionPermission, buildPermissionApplyURL, getPermissionShortName } from '@/utils/permission'
import { usePermissionErrorStore } from '@/stores/permissionError'
import type { PermissionInfo } from '@/utils/permission'
import PermissionDeniedView from '../components/PermissionDeniedView.vue'
import ScheduledTaskDialog from '../components/ScheduledTaskDialog.vue'
import { WorkspaceStateManager } from '../../infrastructure/stateManager/WorkspaceStateManager'
import { createAutoFieldValue, createEmptyFieldValue, createEmptyRawFieldValue } from '@/core/utils/createFieldValue'
import {
  buildInitialDataFromFormDataStore as buildInitialDataFromFormDataStoreHelper,
  createFormViewRuntime,
  syncFormDataStoreToStateManager as syncFormDataStoreToStateManagerHelper
} from './utils/formViewRuntime'
import { FORM_INLINE_LABEL_MAX_CHARS } from '../utils/formLayout'

const props = withDefaults(defineProps<{
  functionDetail?: FunctionDetail  // 🔥 改为可选，因为会在 onMounted 中主动获取
  showSubmitButton?: boolean  // 🔥 是否显示提交按钮（用于 FormDialog 等场景）
  showResetButton?: boolean  // 🔥 是否显示重置按钮
  initialData?: Record<string, any>  // 🔥 初始数据（用于编辑模式）
}>(), {
  showSubmitButton: true,
  showResetButton: true,
  initialData: () => ({}),
})

interface ApplyOperateLogPayload {
  requestBody?: Record<string, any> | null
  responseBody?: Record<string, any> | null
  responseMetadata?: Record<string, any> | null
}

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
  apiClient
})
provide(formDataStoreKey, formDataStore)
const workspaceStateManager = serviceFactory.getWorkspaceStateManager() as WorkspaceStateManager
const workspaceDomainService = serviceFactory.getWorkspaceDomainService()

// 🔥 内部维护 functionDetail（在 onMounted 中主动获取）
const functionDetail = ref<FunctionDetail | null>(props.functionDetail || null)

// 从状态管理器获取状态
const formData = computed(() => {
  const state = stateManager.getState()
  const data: Record<string, any> = {}
  if (state.data) {
    state.data.forEach((value: FieldValue, key: string) => {
      if (value) {
        data[key] = value.raw
      }
    })
  }
  return data
})

const requestFields = computed(() => (functionDetail.value?.request || []) as FieldConfig[])
const responseFields = computed(() => (functionDetail.value?.response || []) as FieldConfig[])

/** 表单级：有任一 label 超过 8 字则全部放上方，否则全部左侧一行 */
const requestLabelsOnTop = computed(() =>
  requestFields.value.some((f) => (f.name?.length ?? 0) > FORM_INLINE_LABEL_MAX_CHARS)
)
const responseLabelsOnTop = computed(() =>
  responseFields.value.some((f) => (f.name?.length ?? 0) > FORM_INLINE_LABEL_MAX_CHARS)
)

// ⭐ 权限检查：获取当前函数节点的权限信息
const currentFunctionNode = computed(() => {
  return workspaceStateManager.getCurrentFunction()
})

// ⭐ 是否有提交权限
const canSubmit = computed(() => {
  const node = currentFunctionNode.value
  if (!node) return true  // 如果没有节点信息，默认允许（向后兼容）
  return hasPermission(node, FormPermission.write)
})

// ⭐ 权限错误状态
const permissionErrorStore = usePermissionErrorStore()
const permissionError = computed<PermissionInfo | null>(() => permissionErrorStore.currentError)

// ⭐ 处理提交按钮的权限申请（PermissionDeniedView 组件已处理权限错误显示）
const handleApplyPermissionForSubmit = () => {
  const node = currentFunctionNode.value
  if (!node || !node.full_code_path) return
  
  // 构建权限申请 URL（传递 template_type 以便正确显示权限选项）
  const applyUrl = buildPermissionApplyURL(node.full_code_path, FunctionPermission.write, node.template_type)
  router.push(applyUrl)
}

// 🔥 移除 formInitialData computed，改为使用统一的数据初始化框架
// URL 参数会在 useFunctionParamInitialization 中统一处理

// 🔥 为所有字段创建响应式的值 Map
// ⭐ 直接访问 formDataStore.data，确保响应式更新
// ⚠️ 注意：Vue 3 的 reactive Map 的 .get() 可能不会建立响应式依赖，需要使用 forEach 遍历
const fieldValues = computed(() => {
  const values: Record<string, FieldValue> = {}
  // ⭐ 先遍历 formDataStore.data 建立响应式依赖
  formDataStore.data.forEach((value, key) => {
    // 只包含 requestFields 中的字段
    if (requestFields.value.some((f: FieldConfig) => f.code === key)) {
      values[key] = value
    }
  })
  // ⭐ 确保所有 requestFields 中的字段都有值（即使 formDataStore 中没有）
  requestFields.value.forEach((field: FieldConfig) => {
    if (!values[field.code]) {
      values[field.code] = createEmptyFieldValue(field)
    }
  })
  return values
})

const submitting = computed(() => {
  const state = stateManager.getState()
  return state.submitting
})

// 🔥 为所有响应字段创建响应式的值 Map
const responseFieldValues = computed(() => {
  const state = stateManager.getState()
  const values: Record<string, FieldValue> = {}
  responseFields.value.forEach((field: FieldConfig) => {
    const rawValue = state.response?.[field.code]
    values[field.code] = createAutoFieldValue(rawValue, field)
  })
  return values
})

const hasResponseData = computed(() => {
  const state = stateManager.getState()
  return state.response !== null && state.response !== undefined
})

// 🔥 获取响应元数据（如 total_cost_mill、trace_id 等）
const responseMetadata = computed(() => {
  const state = stateManager.getState()
  return state.metadata || null
})

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

// Debug 相关
const showDebugDialog = ref(false)
const debugActiveTab = ref('request')
const showScheduledTaskDialog = ref(false)

function onScheduledTaskCreated() {
  eventBus.emit(WorkspaceEvent.scheduledTaskCreated)
}


// 实时获取提交数据（用于 Debug）
const debugRequestData = computed(() => {
  try {
    const submitData = domainService.getSubmitData(requestFields.value)
    return JSON.stringify(submitData, null, 2)
  } catch (error) {
    return JSON.stringify({ error: '获取提交数据失败' }, null, 2)
  }
})

// 获取响应数据（用于 Debug）
const debugResponseData = computed(() => {
  const state = stateManager.getState()
  if (state.response) {
    try {
      return JSON.stringify(state.response, null, 2)
    } catch (error) {
      return JSON.stringify({ error: '格式化响应数据失败' }, null, 2)
    }
  }
  return ''
})

// 获取原始状态数据（用于 Debug）
const debugRawData = computed(() => {
  const state = stateManager.getState()
  try {
    const rawData: Record<string, any> = {}
    state.data.forEach((value: FieldValue, key: string) => {
      // 🔥 dataType 和 widgetType 已经是通用字段，直接显示
      rawData[key] = {
        raw: value.raw,
        display: value.display,
        dataType: value.dataType || 'unknown',  // 🔥 通用字段，和 display 同级别
        widgetType: value.widgetType || 'unknown',  // 🔥 通用字段，和 display 同级别
        meta: value.meta
      }
    })
    return JSON.stringify(rawData, null, 2)
  } catch (error) {
    return JSON.stringify({ error: '格式化原始数据失败' }, null, 2)
  }
})

// 复制到剪贴板
const copyToClipboard = async (text: string): Promise<void> => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败，请手动复制')
  }
}


// FormRenderer 上下文（用于 OnSelectFuzzy 回调）
// 注意：使用 computed 确保响应式更新，并且每次访问都返回新的对象（但方法引用稳定）
const formRendererContext = computed(() => {
  return {
    getFunctionMethod: () => functionDetail.value?.method || 'GET',
    getFunctionRouter: () => functionDetail.value?.router || '',
    getSubmitData: () => {
      const state = stateManager.getState()
      const data: Record<string, any> = {}
      if (state.data) {
        state.data.forEach((value: FieldValue, key: string) => {
          if (value) {
            data[key] = value.raw
          }
        })
      }
      return data
    },
    registerWidget: () => {},
    unregisterWidget: () => {},
    getFieldError: (fieldPath: string) => {
      const errors = domainService.getFieldError(fieldPath)
      return errors[0]?.message || null
    }
  }
})

// 方法
const getFieldValue = (fieldCode: string): FieldValue => {
  return fieldValues.value[fieldCode] || createEmptyRawFieldValue()
}

const getFieldError = (fieldCode: string): string => {
  // 🔥 只在提交时显示验证错误
  const errors = domainService.getFieldError(fieldCode)
  return errors[0]?.message || ''
}

const getResponseFieldValue = (fieldCode: string): FieldValue => {
  return responseFieldValues.value[fieldCode] || createEmptyRawFieldValue()
}

const isFieldRequired = (field: FieldConfig): boolean => {
  return hasAnyRequiredRule(field)
}

const handleFieldUpdate = (fieldCode: string, value: FieldValue): void => {
  // 🔥 调试日志：检查值是否正确传递
  if (!value || value.raw === null || value.raw === undefined) {
    // 空值处理
  }
  applicationService.updateFieldValue(fieldCode, value)
}

const handleSubmit = async (): Promise<void> => {
  try {
    if (!functionDetail.value) {
      ElMessage.error({
        message: '函数详情未加载完成，请稍后重试',
        duration: 3000,
        showClose: true
      })
      return
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
  } catch (error: any) {
    const errorMessage = getErrorMessage(error, '提交失败，请稍后重试')
    Logger.error('FormView', '表单提交失败', {
      router: functionDetail.value?.router,
      message: errorMessage,
      error
    })

    ElMessage.error({
      message: errorMessage,
      duration: 5000,
      showClose: true
    })
  }
}

const handleReset = (): void => {
  resetFormRuntimeState()
  // 重新初始化表单
  const fields = requestFields.value
  if (fields.length > 0) {
    applicationService.initializeForm(fields)
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
  const request = functionDetail.value?.request
  if (!Array.isArray(request) || request.length === 0) {
    return {}
  }
  
  // 使用 domainService 的 getSubmitData 方法递归收集所有字段的数据
  const submitData = domainService.getSubmitData(request)
  
  // 🔥 调试日志：检查提交数据中是否包含所有必填字段
  const requiredFields = request.filter(f => f.validation && f.validation.includes('required'))
  const missingFields = requiredFields.filter(f => submitData[f.code] === undefined || submitData[f.code] === null || submitData[f.code] === '')
  
  if (missingFields.length > 0) {
    Logger.warn('[FormView]', '提交数据中缺少必填字段', {
      missingFields: missingFields.map(f => f.code),
      submitDataKeys: Object.keys(submitData),
      allFieldCodes: request.map(f => f.code)
    })
  }
  
  Logger.info('[FormView]', '准备提交数据（表单提交）', {
    submitData,
    fieldCount: request.length,
    submitDataKeys: Object.keys(submitData),
    allFieldCodes: request.map(f => f.code)
  })
  
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
  const request = functionDetail.value?.request
  if (!Array.isArray(request) || request.length === 0) {
    return {}
  }
  
  // 先获取所有字段的数据
  const allSubmitData = domainService.getSubmitData(request)
  
  // 使用 getChangedFields 过滤出只变更的字段
  const { updates } = getChangedFields(oldValues, allSubmitData)
  
  Logger.info('[FormView]', '准备更新数据（表格更新）', {
    allFieldsCount: Object.keys(allSubmitData).length,
    changedFieldsCount: Object.keys(updates).length,
    changedFields: Object.keys(updates),
    allSubmitData,
    updates
  })
  
  return updates
}

/**
 * 验证表单
 * 这个方法会被 FormDialog 等外部组件调用
 */
function validateForm(): boolean {
  if (!functionDetail.value) {
    Logger.warn('[FormView]', 'functionDetail 不存在，无法验证')
    return false
  }
  
  const fields = (Array.isArray(functionDetail.value.request) ? functionDetail.value.request : []) as FieldConfig[]
  const isValid = domainService.validateForm(fields)
  
  Logger.debug('[FormView]', '表单验证结果', {
    isValid,
    fieldsCount: fields.length,
    fieldCodes: fields.map(f => f.code)
  })
  
  return isValid
}

async function applyOperateLog(payload: ApplyOperateLogPayload): Promise<void> {
  if (!functionDetail.value) {
    throw new Error('函数详情未加载完成')
  }

  const requestBody =
    payload.requestBody && typeof payload.requestBody === 'object' && !Array.isArray(payload.requestBody)
      ? payload.requestBody
      : {}

  await initializeFormForDetail(functionDetail.value, {
    initialData: requestBody,
    force: true
  })

  if (typeof (stateManager as any).setResponse === 'function') {
    ;(stateManager as any).setResponse(payload.responseBody || null)
  }
  if (typeof (stateManager as any).setMetadata === 'function') {
    ;(stateManager as any).setMetadata(payload.responseMetadata || null)
  }

  Logger.info('[FormView]', '已回填执行记录到表单', {
    router: functionDetail.value.router,
    requestKeys: Object.keys(requestBody),
    hasResponseBody: !!payload.responseBody,
    metadataKeys: payload.responseMetadata ? Object.keys(payload.responseMetadata) : []
  })
}

// 🔥 暴露方法给外部组件调用（兼容 FormRenderer 的接口）
defineExpose({
  prepareSubmitDataWithTypeConversion,  // 表单提交（新增场景）
  prepareUpdateData,                     // 表格更新（更新场景，只返回变更的字段）
  validateForm,
  applyOperateLog
})


// 格式化日期
const formatDate = (dateStr: string): string => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 生命周期
let unsubscribeFunctionLoaded: (() => void) | null = null
let unsubscribeFormInitialized: (() => void) | null = null

/**
 * 同步 formDataStore 的数据到 stateManager
 * 🔥 确保 SelectWidgetInitializer 更新后的 display 值不丢失
 * 
 * @param fields 字段配置列表
 */
function syncFormDataStoreToStateManager(fields: FieldConfig[]): void {
  syncFormDataStoreToStateManagerHelper({
    fields,
    formDataStore,
    stateManager
  })
}

/**
 * 从 formDataStore 构建 initialData（只包含 raw 值）
 * 用于传递给 applicationService.initializeForm
 * 
 * @param fields 字段配置列表
 * @returns initialData 对象
 */
function buildInitialDataFromFormDataStore(fields: FieldConfig[]): Record<string, any> {
  return buildInitialDataFromFormDataStoreHelper({
    fields,
    formDataStore
  })
}

function hasNonEmptyInitialData(data?: Record<string, any> | null): boolean {
  return !!data && Object.keys(data).length > 0
}

function buildFormInitializationKey(
  detail: FunctionDetail,
  initialData?: Record<string, any> | null
): string {
  return JSON.stringify({
    id: detail.id ?? null,
    router: detail.router ?? '',
    mode: hasNonEmptyInitialData(initialData) ? 'update' : 'create',
    initialData: initialData || null
  })
}

function restoreResponseParams(metadata?: Record<string, any> | null): void {
  const responseParams = metadata?.responseParams
  if (!responseParams || !stateManager || typeof (stateManager as any).setResponse !== 'function') {
    return
  }

  ;(stateManager as any).setResponse(responseParams)
  Logger.debug('FormView', '已恢复响应数据', {
    responseParamsKeys: Object.keys(responseParams),
    responseParams,
    stateResponse: stateManager.getState().response
  })
}

function resetFormRuntimeState(): void {
  applicationService.clearForm()
  responseDataStore.clear()
}

let latestFormInitializationToken = 0
let inFlightFormInitializationKey: string | null = null
let lastAppliedFormInitializationKey: string | null = null

async function initializeFormForDetail(
  detail: FunctionDetail,
  options: {
    initialData?: Record<string, any>
    resetRuntime?: boolean
    force?: boolean
  } = {}
): Promise<void> {
  const fields = (Array.isArray(detail.request) ? detail.request : []) as FieldConfig[]
  if (fields.length === 0) {
    return
  }

  const explicitInitialData = options.initialData ?? props.initialData
  const initializationKey = buildFormInitializationKey(detail, explicitInitialData)

  if (!options.force) {
    if (initializationKey === inFlightFormInitializationKey || initializationKey === lastAppliedFormInitializationKey) {
      return
    }
  }

  const token = ++latestFormInitializationToken
  inFlightFormInitializationKey = initializationKey

  if (options.resetRuntime !== false) {
    resetFormRuntimeState()
  }

  try {
    if (hasNonEmptyInitialData(explicitInitialData)) {
      applicationService.initializeForm(fields, explicitInitialData, true)
    } else {
      const metadata = await initializeParams()
      if (token !== latestFormInitializationToken) {
        return
      }

      syncFormDataStoreToStateManager(fields)
      const initialData = buildInitialDataFromFormDataStore(fields)
      applicationService.initializeForm(fields, initialData, false)

      if (token !== latestFormInitializationToken) {
        return
      }

      restoreResponseParams(metadata)
    }

    if (token !== latestFormInitializationToken) {
      return
    }

    lastAppliedFormInitializationKey = initializationKey
  } finally {
    if (token === latestFormInitializationToken) {
      inFlightFormInitializationKey = null
    }
  }
}

// 🔥 使用统一的数据初始化框架
const { initialize: initializeParams } = useFunctionParamInitialization({
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
 * 只有新增模式（_tab=OnTableAddRow）才需要同步 URL 参数
 * 其他所有情况（编辑模式、详情模式等）都不需要同步
 */
const shouldSyncURL = computed(() => {
  // 🔥 只有 _tab=OnTableAddRow 时才启用 URL 同步
  return route.query._tab === 'OnTableAddRow'
})

const { watchFormData } = useFormParamURLSync({
  functionDetail: computed(() => functionDetail.value),
  formDataStore: formDataStoreForURLSync,
  enabled: shouldSyncURL,
  debounceMs: 300
})

onMounted(async () => {
  resetFormRuntimeState()
  // ⭐ 清除之前的权限错误（切换函数时清除）
  permissionErrorStore.clearError()
  
  // 🔥 在 onMounted 中主动获取 functionDetail
  // 如果 prop 已经提供了 functionDetail，直接使用；否则从 WorkspaceStateManager 获取当前函数节点并加载详情
  // ⚠️ 注意：id 可能为 0（FormDialog 中设置），所以不能直接用 truthy 判断
  if (props.functionDetail && (props.functionDetail.id !== undefined && props.functionDetail.id !== null)) {
    // 如果 prop 已经提供了 functionDetail，直接使用
    functionDetail.value = props.functionDetail
    Logger.debug('FormView', 'onMounted 时使用 prop 提供的 functionDetail', {
      functionId: props.functionDetail.id,
      requestFieldsCount: Array.isArray(props.functionDetail.request) ? props.functionDetail.request.length : 0
    })
  } else {
    // 否则，从 WorkspaceStateManager 获取当前函数节点并加载详情
    const currentFunction = workspaceStateManager.getCurrentFunction()
    if (currentFunction && currentFunction.type === 'function') {
      Logger.debug('FormView', 'onMounted 时主动加载 functionDetail', {
        functionNodeId: currentFunction.id,
        refId: currentFunction.ref_id,  // 🔥 记录 ref_id（函数 ID）
        functionPath: currentFunction.full_code_path,
        hasRefId: !!(currentFunction.ref_id && currentFunction.ref_id > 0)
      })
      try {
        // 🔥 loadFunction 会优先使用 ref_id 加载函数详情
        const detail = await workspaceDomainService.loadFunction(currentFunction)
        functionDetail.value = detail
        Logger.info('FormView', 'onMounted 时成功加载 functionDetail', {
          functionId: detail.id,
          refId: currentFunction.ref_id,  // 🔥 记录使用的 ref_id
          requestFieldsCount: detail.request?.length || 0,
          requestFields: Array.isArray(detail.request) ? detail.request.map((f: any) => ({
            code: f.code,
            name: f.name,
            widgetType: f.widget?.type,
            hasDefault: !!(f.widget?.config as any)?.default,
            defaultValue: (f.widget?.config as any)?.default
          })) : []
        })
      } catch (error) {
        Logger.error('FormView', 'onMounted 时加载 functionDetail 失败', error)
        return
      }
    } else {
      Logger.debug('FormView', 'onMounted 时没有当前函数节点，等待 watch 触发', {
        hasCurrentFunction: !!currentFunction,
        functionType: currentFunction?.type
      })
      return
    }
  }
  
  if (functionDetail.value && (functionDetail.value.id !== undefined && functionDetail.value.id !== null) && functionDetail.value.request) {
    await initializeFormForDetail(functionDetail.value, {
      resetRuntime: false
    })
  }

  // 监听函数加载完成事件
  let lastInitializedFunctionId: number | null = null // 🔥 记录上次初始化的函数 ID，防止重复初始化
  unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, async (payload: { detail: FunctionDetail }) => {
    const detailId = payload.detail.id
    if (payload.detail.template_type === TEMPLATE_TYPE.FORM && functionDetail.value && detailId != null && detailId === functionDetail.value.id) {
      // 🔥 防重复初始化：如果已经初始化过这个函数，跳过
      if (lastInitializedFunctionId === detailId) {
        Logger.debug('FormView', '跳过重复的 functionLoaded 事件', { functionId: detailId })
        return
      }
      lastInitializedFunctionId = detailId
      functionDetail.value = payload.detail
      await initializeFormForDetail(payload.detail, {
        force: true
      })
    }
  })

  // 监听表单初始化完成事件
  unsubscribeFormInitialized = eventBus.on(FormEvent.initialized, () => {
    // 表单已初始化，可以渲染
  })
  
  // 🔥 开始监听表单数据变化，自动同步到 URL
  watchFormData()
})

  /**
   * 🔥 检查 initialData 是否真的变化了
   */
  const hasInitialDataChanged = (
    newData: Record<string, any>,
    oldData?: Record<string, any>
  ): boolean => {
    const newKeys = Object.keys(newData || {})
    const oldKeys = Object.keys(oldData || {})
    
    // 如果 key 数量不同，或者有新的 key，或者值有变化，才重新初始化
    return newKeys.length !== oldKeys.length || 
      newKeys.some(key => newData[key] !== oldData?.[key])
  }

  // 🔥 监听 initialData 变化，当切换到编辑模式时重新初始化表单
watch(() => props.initialData, async (newInitialData: Record<string, any>, oldInitialData?: Record<string, any>) => {
    // 只在 initialData 真正变化时（且不是首次设置）才重新初始化
    if (!functionDetail.value || !functionDetail.value.request) {
      return
    }
    
    if (oldInitialData === undefined) {
      return
    }
    
    if (!hasInitialDataChanged(newInitialData, oldInitialData)) {
      return
    }

    await initializeFormForDetail(functionDetail.value, {
      initialData: newInitialData,
      force: true
    })
  }, { deep: true })

  // 🔥 统一监听 functionDetail 的关键属性变化，避免多个 watch 走不同初始化分支
  watch(
    () => [props.functionDetail?.id, props.functionDetail?.router, props.functionDetail?.request],
    async ([newId, newRouter, newRequest], [oldId, oldRouter, oldRequest]) => {
      permissionErrorStore.clearError()

      if (props.functionDetail && props.functionDetail.id !== undefined && props.functionDetail.id !== null) {
        functionDetail.value = props.functionDetail
      }

      if (!functionDetail.value || !functionDetail.value.request || functionDetail.value.id === undefined || functionDetail.value.id === null) {
        return
      }
      
      if (oldId === undefined || oldId === null) {
        return
      }
      
      const functionDetailChanged =
        newId !== oldId ||
        newRouter !== oldRouter ||
        JSON.stringify(newRequest || []) !== JSON.stringify(oldRequest || [])

      if (!functionDetailChanged) {
        return
      }

      await initializeFormForDetail(functionDetail.value, {
        force: true
      })
    },
    { deep: true, immediate: false }
  )

// 🔥 移除 watch route.query，改为使用统一的数据初始化框架处理 URL 参数
// URL 参数会在 initializeParams 时统一处理，包括类型转换和组件自治初始化

onUnmounted(() => {
  if (unsubscribeFunctionLoaded) {
    unsubscribeFunctionLoaded()
  }
  if (unsubscribeFormInitialized) {
    unsubscribeFormInitialized()
  }
  applicationService.dispose()
  formDataStore.clear()
  responseDataStore.clear()
})
</script>

<style scoped lang="scss">
.form-view {
  padding: 20px;
}

/* 长 label：label 在上方 */
.form-field-label-top {
  margin-bottom: 18px;

  .field-label {
    display: block;
    font-size: 14px;
    color: var(--el-text-color-regular);
    margin-bottom: 8px;
    line-height: 1.4;
    text-align: right;

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
}


/* 🔥 权限错误显示样式已移至 PermissionDeniedView 组件 */

/* 无权限按钮样式优化 */
.action-btn-no-permission {
  color: var(--el-text-color-secondary) !important;
  border-color: var(--el-border-color-light) !important;
  background-color: var(--el-fill-color-lighter) !important;
  
  &:hover {
    color: var(--el-text-color-secondary) !important;
    border-color: var(--el-border-color-light) !important;
    background-color: var(--el-fill-color-light) !important;
  }
  
  .el-icon {
    margin-right: 4px;
  }
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
  color: #606266;
}

.debug-json-input {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  font-size: 13px;
  line-height: 1.5;
}

.debug-json-input :deep(.el-textarea__inner) {
  background-color: #f5f7fa;
  border: 1px solid #dcdfe6;
  color: #303133;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  resize: none;
}

.debug-json-input :deep(.el-textarea__inner):focus {
  border-color: #409eff;
  background-color: #fff;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 20px;
  color: var(--el-text-color-primary);
}

.form-actions {
  margin-top: 20px;
  display: flex;
  gap: 10px;
}

.form-actions-section {
  margin-top: 20px;
}

.form-actions-row {
  display: flex;
  gap: 12px;
}

.submit-button-full-width {
  flex: 1;
}

.response-section {
  margin-top: 40px;
  padding-top: 20px;
  border-top: 1px solid var(--el-border-color);
}

.response-section .is-empty {
  opacity: 0.6;
}

.metadata-section {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--el-border-color);
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
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
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
</style>
