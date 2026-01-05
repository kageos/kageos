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
          label-width="100px"
          class="function-form"
        >
      <div class="section-title">请求参数</div>
      <el-form-item
        v-for="field in requestFields"
        :key="field.code"
        :label="field.name"
        :required="isFieldRequired(field)"
        :error="getFieldError(field.code)"
      >
        <WidgetComponent
          :field="field"
          :value="fieldValues[field.code]"
          :field-path="field.code"
          :form-renderer="formRendererContext"
          :function-method="functionDetail?.method || 'GET'"
          :function-router="functionDetail?.router || ''"
          @update:model-value="(v: FieldValue) => handleFieldUpdate(field.code, v)"
        />
      </el-form-item>
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
          v-else-if="showSubmitButton"
          type="primary"
          size="large"
          :disabled="false"
          plain
          class="submit-button-full-width action-btn-no-permission"
          @click="handleApplyPermissionForSubmit"
        >
          <el-icon><Lock /></el-icon>
          提交（需权限）
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
        label-width="100px"
        :class="{ 'is-empty': !hasResponseData }"
      >
        <el-form-item
          v-for="field in responseFields"
          :key="field.code"
          :label="field.name"
        >
          <WidgetComponent
            :field="field"
            :value="responseFieldValues[field.code]"
            :field-path="field.code"
            mode="response"
          />
        </el-form-item>
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
      </div>
      <!-- 右侧：函数详情面板 -->
      <div class="form-view-sidebar" v-if="functionDetail && currentFunctionNode">
        <el-card class="function-detail-card" shadow="hover">
          <template #header>
            <div class="function-detail-header">
              <el-icon><InfoFilled /></el-icon>
              <span>函数介绍</span>
            </div>
          </template>
          
          <!-- 创建用户信息 -->
          <div class="detail-section" v-if="functionDetail.created_by">
            <div class="detail-section-title">
              <el-icon><User /></el-icon>
              <span>创建者</span>
            </div>
            <div class="creator-info">
              <UserDisplay 
                :username="functionDetail.created_by" 
                mode="rich"
                layout="horizontal"
              />
            </div>
          </div>
          
          <!-- 函数名称 -->
          <div class="detail-section">
            <div class="function-name">{{ functionDetail.name || currentFunctionNode.name || '-' }}</div>
          </div>

          <!-- 函数描述 -->
          <div class="detail-section" v-if="currentFunctionNode.description">
            <div class="description-wrapper">
              <div class="description-content">{{ currentFunctionNode.description }}</div>
            </div>
          </div>

          <!-- 标签 -->
          <div class="detail-section" v-if="currentFunctionNode.tags">
            <div class="detail-section-title">
              <el-icon><PriceTag /></el-icon>
              <span>标签</span>
            </div>
            <div class="tags-list">
              <el-tag
                v-for="tag in (currentFunctionNode.tags?.split(',') || []).filter((t: string) => t.trim())"
                :key="tag"
                size="small"
                class="tag-item"
              >
                {{ tag.trim() }}
              </el-tag>
            </div>
          </div>

          <!-- 如果没有描述和标签，显示提示 -->
          <div v-if="!currentFunctionNode.description && !currentFunctionNode.tags" class="empty-tip">
            <el-empty description="暂无介绍信息" :image-size="80" />
          </div>
        </el-card>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch, ref, nextTick, withDefaults } from 'vue'
import type { ComputedRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Promotion, RefreshLeft, View, DocumentCopy, InfoFilled, Lock, Document, List, PriceTag, User } from '@element-plus/icons-vue'
import { ElIcon, ElTag, ElNotification, ElMessage, ElAlert, ElMessageBox, ElText, ElCheckbox, ElCard, ElEmpty } from 'element-plus'
import { eventBus, FormEvent, WorkspaceEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import WidgetComponent from '../widgets/WidgetComponent.vue'
import UserDisplay from '../widgets/UserDisplay.vue'
import { Logger } from '@/core/utils/logger'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import { hasAnyRequiredRule } from '@/core/utils/validationUtils'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { useResponseDataStore } from '@/core/stores-v2/responseData'
import { useFunctionParamInitialization } from '../composables/useFunctionParamInitialization'
import { useFormParamURLSync } from '../composables/useFormParamURLSync'
import { hasPermission, FormPermissions, buildPermissionApplyURL } from '@/utils/permission'
import { usePermissionErrorStore } from '@/stores/permissionError'
import type { PermissionInfo } from '@/utils/permission'
import PermissionDeniedView from '../components/PermissionDeniedView.vue'

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

// 路由
const route = useRoute()
const router = useRouter()

// 依赖注入（使用 IServiceProvider 接口，遵循依赖倒置原则）
const serviceProvider: IServiceProvider = serviceFactory
const stateManager = serviceProvider.getFormStateManager() as FormStateManager  // 🔥 类型断言：FormStateManager 有 setResponse 方法
const domainService = serviceProvider.getFormDomainService()
const applicationService = serviceProvider.getFormApplicationService()
const workspaceStateManager = serviceProvider.getWorkspaceStateManager()  // 🔥 用于获取当前函数节点
const workspaceDomainService = serviceProvider.getWorkspaceDomainService()  // 🔥 用于获取函数详情

// 🔥 内部维护 functionDetail（在 onMounted 中主动获取）
const functionDetail = ref<FunctionDetail | null>(props.functionDetail || null)

// 🔥 获取全局 formDataStore 和 responseDataStore（用于清理，因为 WidgetComponent 内部使用的组件会直接使用这些 store）
const formDataStore = useFormDataStore()
const responseDataStore = useResponseDataStore()

// 从状态管理器获取状态
const formData = computed(() => {
  const state = stateManager.getState()
  const data: Record<string, any> = {}
  if (state.data) {
    state.data.forEach((value, key) => {
      if (value) {
        data[key] = value.raw
      }
    })
  }
  return data
})

const requestFields = computed(() => (functionDetail.value?.request || []) as FieldConfig[])
const responseFields = computed(() => (functionDetail.value?.response || []) as FieldConfig[])

// ⭐ 权限检查：获取当前函数节点的权限信息
const currentFunctionNode = computed(() => {
  return workspaceStateManager.getCurrentFunction()
})

// ⭐ 是否有提交权限
const canSubmit = computed(() => {
  const node = currentFunctionNode.value
  if (!node) return true  // 如果没有节点信息，默认允许（向后兼容）
  return hasPermission(node, FormPermissions.write)
})

// ⭐ 权限错误状态
const permissionErrorStore = usePermissionErrorStore()
const permissionError = computed<PermissionInfo | null>(() => permissionErrorStore.currentError)

// ⭐ 处理提交按钮的权限申请（PermissionDeniedView 组件已处理权限错误显示）
const handleApplyPermissionForSubmit = () => {
  const node = currentFunctionNode.value
  if (!node || !node.full_code_path) return
  
  // 构建权限申请 URL（传递 template_type 以便正确显示权限选项）
        const applyUrl = buildPermissionApplyURL(node.full_code_path, 'function:write', node.template_type)
  router.push(applyUrl)
}

// 🔥 移除 formInitialData computed，改为使用统一的数据初始化框架
// URL 参数会在 useFunctionParamInitialization 中统一处理

// 🔥 为所有字段创建响应式的值 Map
const fieldValues = computed(() => {
  const state = stateManager.getState()
  const values: Record<string, FieldValue> = {}
  requestFields.value.forEach((field: FieldConfig) => {
    const fieldValue = state.data?.get(field.code) || { raw: null, display: '', meta: {} }
    values[field.code] = fieldValue
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
    values[field.code] = {
      raw: rawValue !== undefined ? rawValue : null,
      display: rawValue !== null && rawValue !== undefined 
        ? (typeof rawValue === 'object' ? JSON.stringify(rawValue) : String(rawValue))
        : '',
      meta: {}
    }
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
    state.data.forEach((value, key) => {
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
        state.data.forEach((value, key) => {
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
  return fieldValues.value[fieldCode] || { raw: null, display: '', meta: {} }
}

const getFieldError = (fieldCode: string): string => {
  // 🔥 只在提交时显示验证错误
  const errors = domainService.getFieldError(fieldCode)
  return errors[0]?.message || ''
}

const getResponseFieldValue = (fieldCode: string): FieldValue => {
  return responseFieldValues.value[fieldCode] || { raw: null, display: '', meta: {} }
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
      ElNotification.error({
        title: '提交失败',
        message: '函数详情未加载完成，请稍后重试',
        duration: 3000
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
    // 🔥 从错误对象中提取错误消息
    // request.ts 的响应拦截器在 code !== 0 时会 reject，并创建错误对象
    // 错误对象包含 response 属性，其中包含完整的响应数据
    let errorMessage = '提交失败，请稍后重试'
    
    // 🔥 统一使用 msg 字段
    // 尝试从 error.response.data 中获取错误消息（request.ts 第 99-101 行）
    if (error?.response?.data) {
      const responseData = error.response.data
      errorMessage = responseData.msg || errorMessage
    } else if (error?.message) {
      // 如果错误对象本身有 message（request.ts 第 99 行创建的）
      errorMessage = error.message
    }
    
    ElNotification.error({
      title: '提交失败',
      message: errorMessage,
      duration: 3000
    })
  }
}

const handleReset = (): void => {
  // 🔥 重置时清理 store 数据
  formDataStore.clear()
  responseDataStore.clear()
  
  applicationService.clearForm()
  // 重新初始化表单
  const fields = requestFields.value
  if (fields.length > 0) {
    applicationService.initializeForm(fields)
  }
}

/**
 * 准备提交数据（带类型转换）
 * 这个方法会被 FormDialog 等外部组件调用
 * 🔥 兼容 FormRenderer 的接口
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
  
  Logger.info('[FormView]', '准备提交数据', {
    submitData,
    fieldCount: request.length,
    submitDataKeys: Object.keys(submitData),
    allFieldCodes: request.map(f => f.code)
  })
  
  return submitData
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

// 🔥 暴露方法给外部组件调用（兼容 FormRenderer 的接口）
defineExpose({
  prepareSubmitDataWithTypeConversion,
  validateForm
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
  const state = stateManager.getState()
  const newData = new Map<string, FieldValue>()
  
  fields.forEach((field: FieldConfig) => {
    const fieldValue = formDataStore.getValue(field.code)
    if (fieldValue) {
      // 🔥 直接使用 formDataStore 中的完整 FieldValue（包含 display）
      newData.set(field.code, fieldValue)
    } else {
      // 如果没有值，使用默认值
      newData.set(field.code, { raw: null, display: '', meta: {} })
    }
  })
  
  // 🔥 同步更新 stateManager，确保 fieldValues computed 能获取到最新的 display 值
  stateManager.setState({
    ...state,
    data: newData
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
  const initialData: Record<string, any> = {}
  fields.forEach((field: FieldConfig) => {
    const fieldValue = formDataStore.getValue(field.code)
    if (fieldValue) {
      initialData[field.code] = fieldValue.raw
    }
  })
  return initialData
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
        state.data.forEach((value, key) => {
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
      data.forEach((value, key) => {
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
  // 🔥 挂载时清理 store，避免之前函数的数据污染
  formDataStore.clear()
  responseDataStore.clear()
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
  
  /**
   * 🔥 使用 initialData 初始化表单（用于编辑模式）
   * 先清空 stateManager，然后直接使用 initialData 初始化，避免默认值干扰
   */
  const initializeFormWithData = (fields: FieldConfig[], initialData: Record<string, any>) => {
    // 🔥 重要：先清空 stateManager，避免已有值影响 initialData 的初始化
    const currentState = stateManager.getState()
    stateManager.setState({
      ...currentState,
      data: new Map(),
      errors: new Map(),
      submitting: false
    })
    
    // 🔥 直接调用 initializeForm，不使用 syncFormDataStoreToStateManager
    // 因为 formDataStore 可能是空的，会设置默认值，影响 initialData 的初始化
    applicationService.initializeForm(fields, initialData)
  }

  /**
   * 🔥 使用正常流程初始化表单（用于新建模式）
   * 先初始化参数，然后从 formDataStore 或 props.initialData 构建初始数据
   */
  const initializeFormNormal = async (fields: FieldConfig[]) => {
    Logger.debug('FormView', 'onMounted 时初始化参数', {
      functionId: functionDetail.value?.id,
      requestFieldsCount: fields.length
    })
    const metadata = await initializeParams()
    
    // 初始化表单：在参数初始化完成后，初始化表单结构
    if (fields.length > 0) {
      // 🔥 同步 formDataStore 的数据到 stateManager，确保 display 值不丢失
      syncFormDataStoreToStateManager(fields)
      
      // 🔥 优先使用 props.initialData，如果没有则使用 formDataStore 中的数据
      const initialData = Object.keys(props.initialData).length > 0 
        ? props.initialData 
        : buildInitialDataFromFormDataStore(fields)
      applicationService.initializeForm(fields, initialData)
    }
    
    // 🔥 恢复响应数据（在表单初始化之后，避免被覆盖）
    if (metadata?.responseParams && stateManager) {
      stateManager.setResponse(metadata.responseParams)
      Logger.debug('FormView', '已恢复响应数据', {
        responseParamsKeys: Object.keys(metadata.responseParams)
      })
    }
  }

  // 🔥 初始化参数（此时 functionDetail 已经加载完成）
  // ⚠️ 注意：id 可能为 0（FormDialog 中设置），所以不能直接用 truthy 判断
  if (functionDetail.value && (functionDetail.value.id !== undefined && functionDetail.value.id !== null) && functionDetail.value.request) {
    const fields = Array.isArray(functionDetail.value.request) ? functionDetail.value.request : []
    
    // 🔥 如果 props.initialData 有值，直接使用它初始化，跳过 initializeParams
    // 这样可以避免 initializeParams 使用默认值覆盖 initialData
    if (Object.keys(props.initialData).length > 0 && fields.length > 0) {
      initializeFormWithData(fields, props.initialData)
    } else {
      // 🔥 如果没有 initialData，使用正常的初始化流程
      await initializeFormNormal(fields)
    }
  }

  // 监听函数加载完成事件
  let lastInitializedFunctionId: number | null = null // 🔥 记录上次初始化的函数 ID，防止重复初始化
  unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, async (payload: { detail: FunctionDetail }) => {
    if (payload.detail.template_type === TEMPLATE_TYPE.FORM && functionDetail.value && payload.detail.id === functionDetail.value.id) {
      // 🔥 防重复初始化：如果已经初始化过这个函数，跳过
      if (lastInitializedFunctionId === payload.detail.id) {
        Logger.debug('FormView', '跳过重复的 functionLoaded 事件', { functionId: payload.detail.id })
        return
      }
      lastInitializedFunctionId = payload.detail.id
      
      // 🔥 切换函数时，先清理全局 store（因为 WidgetComponent 内部使用的组件会直接使用这些 store）
      formDataStore.clear()
      responseDataStore.clear()
      
      // 🔥 使用统一的数据初始化框架初始化参数
      const metadata = await initializeParams()
      
      // 🔥 使用 nextTick 确保参数初始化完成
      nextTick(() => {
        // 重新初始化表单（从 formDataStore 获取已初始化的数据）
        // 🔥 确保 fields 是数组，防止类型错误
        const fields = (Array.isArray(payload.detail.request) ? payload.detail.request : []) as FieldConfig[]
        if (fields.length > 0) {
          // 🔥 同步 formDataStore 的数据到 stateManager，确保 display 值不丢失
          syncFormDataStoreToStateManager(fields)
          
          // 🔥 构建 initialData 并调用 initializeForm
          const initialData = buildInitialDataFromFormDataStore(fields)
          applicationService.initializeForm(fields, initialData)
        }
        
        // 🔥 恢复响应数据（在表单初始化之后，避免被覆盖）
        if (metadata?.responseParams && stateManager && typeof (stateManager as any).setResponse === 'function') {
          (stateManager as any).setResponse(metadata.responseParams)
          Logger.debug('FormView', '已恢复响应数据', {
            responseParamsKeys: Object.keys(metadata.responseParams),
            responseParams: metadata.responseParams,
            stateResponse: stateManager.getState().response
          })
        }
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

  // 🔥 监听 props.functionDetail 变化，同步到内部的 functionDetail ref
  // 注意：只在 props.functionDetail 真正变化时（id 或 router 变化）才重新初始化
  // 初始化逻辑在 onMounted 中处理，这里只处理函数切换的场景
  watch(() => props.functionDetail, async (newDetail: FunctionDetail | undefined, oldDetail?: FunctionDetail) => {
    // ⭐ 切换函数时清除权限错误
    permissionErrorStore.clearError()
    
    // 🔥 同步到内部的 functionDetail ref
    // ⚠️ 注意：id 可能为 0（FormDialog 中设置），所以不能直接用 truthy 判断
    if (newDetail && (newDetail.id !== undefined && newDetail.id !== null)) {
      functionDetail.value = newDetail
    }
    
    // 🔥 检查 functionDetail 是否有效（必须要有 id 和 request 字段）
    // ⚠️ 注意：id 可能为 0（FormDialog 中设置），所以不能直接用 truthy 判断
    if (!newDetail || (newDetail.id === undefined || newDetail.id === null) || !newDetail.request) {
      Logger.debug('FormView', 'props.functionDetail 无效或未加载完成，跳过初始化', {
        hasDetail: !!newDetail,
        hasId: !!newDetail?.id,
        hasRequest: !!newDetail?.request,
        requestCount: newDetail?.request?.length || 0
      })
      return
    }
    
    // 🔥 只在 functionDetail 的 id 或 router 真正变化时重新初始化
    // 如果只是其他属性变化（如字段配置），不应该重新初始化
    // 注意：oldDetail 为 undefined 时，说明是首次设置，此时 onMounted 已经处理过了，不需要重复初始化
    if (oldDetail && (newDetail.id !== oldDetail.id || newDetail.router !== oldDetail.router)) {
      Logger.debug('FormView', 'props.functionDetail 变化（函数切换），开始重新初始化', {
        oldId: oldDetail.id,
        newId: newDetail.id,
        oldRouter: oldDetail.router,
        newRouter: newDetail.router,
        requestFieldsCount: newDetail.request?.length || 0
      })
      
      // 🔥 切换函数时，先清理全局 store（因为 WidgetComponent 内部使用的组件会直接使用这些 store）
      formDataStore.clear()
      responseDataStore.clear()
      
      // 🔥 使用统一的数据初始化框架初始化参数（此时 functionDetail 已经加载完成）
      const metadata = await initializeParams()
      
      const fields = (newDetail.request || []) as FieldConfig[]
      if (fields.length > 0) {
        // 🔥 使用 nextTick 确保参数初始化完成
        nextTick(() => {
          // 🔥 同步 formDataStore 的数据到 stateManager，确保 display 值不丢失
          syncFormDataStoreToStateManager(fields)
          
          // 🔥 构建 initialData 并调用 initializeForm
          // 🔥 优先使用 props.initialData，如果没有则使用 formDataStore 中的数据
          const initialData = Object.keys(props.initialData).length > 0 
            ? props.initialData 
            : buildInitialDataFromFormDataStore(fields)
          console.log('[FormView] 函数切换后初始化表单', {
            fieldsCount: fields.length,
            fieldCodes: fields.map((f: FieldConfig) => f.code),
            initialDataKeys: Object.keys(initialData),
            initialData,
            propsInitialDataKeys: Object.keys(props.initialData),
            propsInitialData: props.initialData,
            fromProps: Object.keys(props.initialData).length > 0
          })
          applicationService.initializeForm(fields, initialData)
          
          // 🔥 恢复响应数据（在表单初始化之后，避免被覆盖）
          if (metadata?.responseParams && stateManager && typeof (stateManager as any).setResponse === 'function') {
            (stateManager as any).setResponse(metadata.responseParams)
            Logger.debug('FormView', '已恢复响应数据', {
              responseParamsKeys: Object.keys(metadata.responseParams),
              responseParams: metadata.responseParams,
              stateResponse: stateManager.getState().response
            })
          }
        })
      }
    }
  }, { deep: false }) // 🔥 移除 immediate: true，避免与 onMounted 重复初始化

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
    
    if (!hasInitialDataChanged(newInitialData, oldInitialData)) {
      return
    }
    
    const fields = (functionDetail.value.request || []) as FieldConfig[]
    if (fields.length > 0 && newInitialData && Object.keys(newInitialData).length > 0) {
      // 🔥 使用新的 initialData 重新初始化表单
      nextTick(() => {
        syncFormDataStoreToStateManager(fields)
        applicationService.initializeForm(fields, newInitialData)
      })
    }
  }, { deep: true })

  // 🔥 监听 functionDetail 和 initialData 的组合变化，确保在编辑模式下能够正确初始化
  // 当切换到编辑模式时，functionDetail 可能会变化，此时需要重新初始化表单
  watch(
    () => [props.functionDetail?.id, props.functionDetail?.request, Object.keys(props.initialData || {})],
    async ([newId, newRequest, newInitialDataKeys], [oldId, oldRequest, oldInitialDataKeys]) => {
      // 只在 functionDetail 准备好且有 initialData 时才初始化
      // ⚠️ 注意：id 可能为 0（FormDialog 中设置），所以不能直接用 truthy 判断
      if (!functionDetail.value || !functionDetail.value.request || (functionDetail.value.id === undefined || functionDetail.value.id === null)) {
        return
      }
      
      // 检查是否有新的 initialData（从空变为有值）
      const hasNewInitialData = newInitialDataKeys && newInitialDataKeys.length > 0 && 
        (!oldInitialDataKeys || oldInitialDataKeys.length === 0)
      
      // 检查 functionDetail 是否变化了（id 或 request 变化）
      const functionDetailChanged = newId !== oldId || 
        (newRequest && oldRequest && JSON.stringify(newRequest) !== JSON.stringify(oldRequest))
      
      // 如果 functionDetail 变化了，或者有新的 initialData，重新初始化
      if (functionDetailChanged || hasNewInitialData) {
        const fields = (functionDetail.value.request || []) as FieldConfig[]
        if (fields.length > 0) {
          const initialData = Object.keys(props.initialData).length > 0 
            ? props.initialData 
            : buildInitialDataFromFormDataStore(fields)
          
          if (Object.keys(initialData).length > 0) {
            nextTick(() => {
              syncFormDataStoreToStateManager(fields)
              applicationService.initializeForm(fields, initialData)
            })
          }
        }
      }
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
})
</script>

<style scoped lang="scss">
.form-view {
  padding: 20px;
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

.form-view-sidebar {
  width: 360px;
  flex-shrink: 0;
  position: sticky;
  top: 20px;
  max-height: calc(100vh - 40px);
  overflow-y: auto;
}

.function-detail-card {
  .function-detail-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    font-size: 16px;
    color: var(--el-text-color-primary);
  }
}

.detail-section {
  margin-bottom: 24px;
  
  .detail-section-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    font-weight: 500;
    color: var(--el-text-color-regular);
    margin-bottom: 12px;
  }
  
  .creator-info {
    display: flex;
    align-items: flex-start;
    width: 100%;
  }
  
  &:last-child {
    margin-bottom: 0;
  }
}

.function-name {
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.detail-section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 14px;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
}

.description-wrapper {
  position: relative;
}

.description-content {
  font-size: 14px;
  color: var(--el-text-color-primary);
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-word;
  padding: 16px 20px;
  background: linear-gradient(135deg, rgba(64, 158, 255, 0.05) 0%, rgba(64, 158, 255, 0.02) 100%);
  border-radius: 8px;
  border-left: 3px solid var(--el-color-primary);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  position: relative;
  
  &::before {
    content: '';
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    width: 3px;
    background: linear-gradient(180deg, var(--el-color-primary) 0%, rgba(64, 158, 255, 0.6) 100%);
    border-radius: 8px 0 0 8px;
  }
}

.tags-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-item {
  margin: 0;
}

.empty-tip {
  padding: 20px 0;
}

/* 🔥 权限错误显示样式已移至 PermissionDeniedView 组件 */

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

