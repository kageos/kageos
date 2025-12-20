<!--
  FormRenderer-v2 - 新的表单渲染器
  🔥 完全新增，使用新的组件系统
  
  功能：
  - 使用 Pinia Store 管理数据
  - 使用新的 Vue 组件系统
  - 支持请求参数和响应参数渲染
  - 支持表单提交和验证
-->

<template>
  <div v-if="isMounted" class="form-renderer-v2" :key="rendererKey">
    <!-- 请求参数表单 -->
    <el-form
      v-if="requestFields.length > 0"
      ref="formRef"
      :model="formData"
      label-width="100px"
      class="function-form"
    >
      <div class="section-title">请求参数</div>
      <el-form-item
        v-for="field in requestFields"
        :key="`request_${field.code}`"
        :label="field.name"
        :required="isFieldRequired(field)"
        :error="getFieldError(field.code)"
        class="request-form-item"
      >
        <component
          v-if="getWidgetComponent(field.widget?.type || 'input')"
          :key="`request_widget_${field.code}_${field.widget?.type || 'input'}`"
          :is="getWidgetComponent(field.widget?.type || 'input')"
          :ref="(el: any) => setWidgetRef(field.code, el)"
          :field="field"
          :value="getFieldValue(field.code)"
          :model-value="getFieldValue(field.code)"
          @update:model-value="(v) => updateFieldValue(field.code, v)"
          :field-path="field.code"
          :form-manager="formManager"
          :form-renderer="formRendererContext"
          :user-info-map="userInfoMap"
          :function-name="functionName"
          :record-id="recordId"
          mode="edit"
        />
        <div v-else class="widget-error">
          组件未找到: {{ field.widget?.type || 'input' }}
        </div>
      </el-form-item>
    </el-form>

    <!-- 提交按钮区域 -->
    <div v-if="showSubmitButton || showResetButton" class="form-actions-section">
      <div class="form-actions-row">
        <el-button
          v-if="showSubmitButton"
          type="primary"
          size="large"
          @click="handleSubmit"
          :loading="submitting"
          class="submit-button-full-width"
        >
          <el-icon><Promotion /></el-icon>
          提交
        </el-button>
        <el-button v-if="showResetButton" size="large" @click="handleReset">
          <el-icon><RefreshLeft /></el-icon>
          重置
        </el-button>
      </div>
    </div>

    <!-- 响应参数展示 -->
    <div v-if="responseFields.length > 0">
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
        class="function-form response-container"
        :class="{ 'is-empty': !hasResponseData }"
      >
        <el-form-item
          v-for="field in responseFields"
          :key="`response_${field.code}`"
          :label="field.name"
          class="response-form-item"
        >
          <component
            v-if="getResponseWidgetComponent(field.widget?.type || 'input')"
            :key="`response_widget_${field.code}_${field.widget?.type || 'input'}_${responseDataStore?.renderTrigger || 0}`"
            :is="getResponseWidgetComponent(field.widget?.type || 'input')"
            :field="field"
            :value="responseFieldValues[field.code] || { raw: null, display: '', meta: {} }"
            :model-value="responseFieldValues[field.code] || { raw: null, display: '', meta: {} }"
            :field-path="field.code"
            :form-renderer="formRendererContext"
            :user-info-map="userInfoMap"
            :function-name="functionName"
            :record-id="recordId"
            mode="response"
          />
          <div v-else class="widget-error">
            响应组件未找到: {{ field.widget?.type || 'input' }}
          </div>
        </el-form-item>
      </el-form>
    </div>

    <!-- 提交结果 -->
    <el-card v-if="submitResult" class="result-card" style="margin-top: 20px;">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>提交结果</span>
          <el-button text @click="submitResult = null">关闭</el-button>
        </div>
      </template>
      <div class="result-content">
        <h4>提交的数据：</h4>
        <pre>{{ submitResult }}</pre>
      </div>
    </el-card>

  </div>
</template>

<script setup lang="ts">
// 设置组件名称
defineOptions({
  name: 'FormRenderer'
})

import { ref, computed, onMounted, onBeforeUnmount, onUnmounted, nextTick, watch, reactive } from 'vue'
import { ElForm, ElFormItem, ElButton, ElCard, ElMessage, ElMessageBox, ElIcon, ElTag } from 'element-plus'
import { Promotion, RefreshLeft } from '@element-plus/icons-vue'
import type { FieldConfig, FunctionDetail, FieldValue } from '../types/field'
import { useFormDataStore } from '../stores-v2/formData'
import { useResponseDataStore } from '../stores-v2/responseData'
import { widgetComponentFactory } from '../factories-v2'
import { executeFunction } from '@/api/function'
import { Logger } from '../utils/logger'
import { shouldShowField } from '../utils/conditionEvaluator'
import { hasAnyRequiredRule } from '../utils/validationUtils'
import { ValidationEngine, createDefaultValidatorRegistry } from '../validation'
import type { ReactiveFormDataManager } from '../managers/ReactiveFormDataManager'
import type { FormRendererContext } from '../types/widget'
import type { ValidationResult } from '../validation/types'
import { getWidgetDefaultValue } from '../widgets-v2/composables/useWidgetDefaultValue'
import { useAuthStore } from '@/stores/auth'
import { convertToFieldValue } from '@/utils/field'

const props = withDefaults(defineProps<{
  functionDetail: FunctionDetail
  showSubmitButton?: boolean
  showResetButton?: boolean
  initialData?: Record<string, any>
  userInfoMap?: Map<string, any>  // 🔥 用户信息映射（用于 UserWidget 批量查询优化）
}>(), {
  showSubmitButton: true,
  showResetButton: true,
  initialData: () => ({}),
  userInfoMap: () => new Map()
})

// Pinia Stores
const formDataStore = useFormDataStore()
const responseDataStore = useResponseDataStore()

// 🔥 用户信息映射（从 props 获取，如果没有则使用空 Map）
const userInfoMap = computed(() => props.userInfoMap || new Map())

// 🔥 从 functionDetail.router 提取函数名称（用于 FilesWidget 打包下载命名）
const functionName = computed(() => {
  if (!props.functionDetail?.router) {
    return undefined
  }
  
  // router 格式通常是：/user/app/function_name 或 /user/app/group/function_name
  const routerParts = props.functionDetail.router.split('/').filter(Boolean)
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

// 🔥 从 initialData 提取 recordId（用于 FilesWidget 打包下载命名）
const recordId = computed(() => {
  if (!props.initialData) {
    return undefined
  }
  
  // 尝试从 initialData 中获取 id 字段（可能是 id、ID、record_id 等）
  const idField = Object.keys(props.initialData).find(key => {
    const lowerKey = key.toLowerCase()
    return lowerKey === 'id' || lowerKey.endsWith('_id') || lowerKey.endsWith('id')
  })
  
  if (idField) {
    const idValue = props.initialData[idField]
    return idValue !== null && idValue !== undefined ? idValue : undefined
  }
  
  return undefined
})

// 表单引用
const formRef = ref()

// 🔥 Widget refs 映射（用于调用 Widget 的 validate 方法）
const widgetRefs = new Map<string, any>()

/**
 * 设置 Widget ref（用于调用 Widget 的 validate 方法）
 */
function setWidgetRef(fieldCode: string, el: any): void {
  if (el) {
    widgetRefs.set(fieldCode, el)
  } else {
    widgetRefs.delete(fieldCode)
  }
}

// 提交状态
const submitting = ref(false)
const submitResult = ref<any>(null)

// 组件挂载状态（用于控制渲染）
const isMounted = ref(false)

// 渲染器 key（用于强制重新渲染）
const rendererKey = computed(() => {
  if (!props.functionDetail) {
    return 'default'
  }
  return String(props.functionDetail.id || props.functionDetail.router || 'default')
})

// 请求字段列表（根据条件渲染规则过滤）
const requestFields = computed(() => {
  // 🔥 关键：追踪 formDataStore.data 的变化，确保条件渲染能响应式更新
  const _ = formDataStore.data  // 触发响应式追踪
  
  const allFields = props.functionDetail?.request || []
  return allFields.filter((field: FieldConfig) => {
    // 条件渲染：根据其他字段的值决定是否显示
    // 注意：这里需要适配 shouldShowField 函数，使其支持 formDataStore
    return shouldShowFieldInForm(field, formDataStore, allFields)
  })
})

// 响应字段列表
const responseFields = computed(() => {
  return props.functionDetail?.response || []
})

// 是否有响应数据
// 🔥 关键：需要追踪 renderTrigger 来确保响应式更新
const hasResponseData = computed(() => {
  if (!responseDataStore || !responseDataStore.data || !isMounted.value) {
    return false
  }
  try {
    // 读取 renderTrigger 作为依赖，确保数据更新时重新计算
    const trigger = responseDataStore.renderTrigger
    // 🔥 注意：Pinia store 返回的 ref 需要直接访问 .value
    const data = responseDataStore.data?.value ?? responseDataStore.data
    return data !== null && data !== undefined
  } catch (error) {
    Logger.warn('[FormRenderer-v2]', 'hasResponseData computed 错误:', error)
    return false
  }
})

// 表单数据（用于 el-form 绑定）
const formData = computed(() => {
  const data: Record<string, any> = {}
  requestFields.value.forEach((field: FieldConfig) => {
    const value = formDataStore.getValue(field.code)
    data[field.code] = value?.raw
  })
  return data
})

// 获取字段值
function getFieldValue(fieldCode: string): FieldValue {
  return formDataStore.getValue(fieldCode)
}

// 更新字段值
function updateFieldValue(fieldCode: string, value: FieldValue): void {
  formDataStore.setValue(fieldCode, value)
  
  // 字段值改变时，重新验证当前字段
  const field = requestFields.value.find(f => f.code === fieldCode)
  if (field) {
    validateField(field)
    
    // 🔥 处理字段依赖：当字段值变化时，清空所有依赖该字段的其他字段
    // 例如：当 topic_id 变化时，自动清空 option_ids（因为选项列表会变化）
    requestFields.value.forEach(otherField => {
      // 🔥 安全检查：确保 otherField 存在且有 code 和 depend_on 属性
      if (!otherField || !otherField.code || !otherField.depend_on) {
        return
      }
      
      if (otherField.depend_on === fieldCode) {
        Logger.debug('FormRenderer', `字段 ${otherField.code} 依赖 ${fieldCode}，清空其值`)
        formDataStore.setValue(otherField.code, {
          raw: null,
          display: '',
          meta: {}
        })
        // 同时清空该字段的验证错误（fieldErrors 是 Map，使用 delete 方法）
        if (fieldErrors.has(otherField.code)) {
          fieldErrors.delete(otherField.code)
        }
      }
    })
    
    // 🔥 同时验证所有其他字段（因为条件验证可能依赖多个字段）
    // 例如：字段A的值改变时，可能影响字段B的 required_if 验证
    requestFields.value.forEach(otherField => {
      if (otherField.code !== fieldCode && otherField.validation) {
        validateField(otherField)
      }
    })
  }
}

// 获取响应字段值
// 🔥 为每个字段创建 computed，确保响应式更新
const getResponseFieldValue = (fieldCode: string): FieldValue => {
  // 读取 renderTrigger 作为依赖，确保数据更新时重新计算
  const trigger = responseDataStore.renderTrigger
  // 🔥 注意：Pinia store 返回的 ref 需要直接访问 .value
  const responseData = responseDataStore.data?.value ?? responseDataStore.data
  
  if (!responseData) {
    return {
      raw: null,
      display: '',
      meta: {}
    }
  }
  
  const rawValue = responseData[fieldCode]
  
  if (rawValue === null || rawValue === undefined) {
    return {
      raw: null,
      display: '',
      meta: {}
    }
  }
  
  return {
    raw: rawValue,
    display: typeof rawValue === 'object' ? JSON.stringify(rawValue) : String(rawValue),
    meta: {}
  }
}

// 🔥 为每个响应字段创建 computed，确保响应式更新
const responseFieldValues = computed(() => {
  // 如果组件未挂载，返回空值，避免在卸载时访问数据
  if (!isMounted.value || !responseDataStore) {
    return {}
  }
  
  try {
    // 🔥 关键：必须读取 renderTrigger 作为依赖，确保数据更新时重新计算
    const trigger = responseDataStore.renderTrigger
    // 🔥 注意：Pinia store 返回的 ref 需要直接访问 .value
    const responseData = responseDataStore.data?.value ?? responseDataStore.data
    
    const values: Record<string, FieldValue> = {}
    
    responseFields.value.forEach(field => {
      if (!responseData) {
        values[field.code] = {
          raw: null,
          display: '',
          meta: {}
        }
        return
      }
      
      const rawValue = responseData[field.code]
      
      // 🔥 使用 convertToFieldValue 来正确转换字段值（特别是时间戳字段）
      // 这样可以确保时间戳字段被正确格式化为日期字符串
      values[field.code] = convertToFieldValue(rawValue, field)
    })
    
    return values
  } catch (error) {
    Logger.warn('FormRenderer', 'responseFieldValues computed 错误', error)
    return {}
  }
})

// 缓存组件查找结果，避免重复查找和确保组件引用稳定
const componentCache = new Map<string, any>()

// 获取请求组件
function getWidgetComponent(type: string) {
  const cacheKey = `request_${type}`
  if (componentCache.has(cacheKey)) {
    return componentCache.get(cacheKey)
  }
  
  const component = widgetComponentFactory.getRequestComponent(type)
  if (!component) {
    Logger.warn('FormRenderer', `未找到组件: ${type}，使用默认 InputWidget`)
    const defaultComponent = widgetComponentFactory.getRequestComponent('input')
    componentCache.set(cacheKey, defaultComponent)
    return defaultComponent
  }
  
  componentCache.set(cacheKey, component)
  return component
}

// 获取响应组件
function getResponseWidgetComponent(type: string) {
  const cacheKey = `response_${type}`
  if (componentCache.has(cacheKey)) {
    return componentCache.get(cacheKey)
  }
  
  // 优先使用响应组件，如果没有则使用请求组件
  const component = widgetComponentFactory.getResponseComponent(type)
  if (!component) {
    Logger.warn('FormRenderer', `未找到响应组件: ${type}，使用默认 InputWidget`)
    const defaultComponent = widgetComponentFactory.getRequestComponent('input')
    componentCache.set(cacheKey, defaultComponent)
    return defaultComponent
  }
  
  componentCache.set(cacheKey, component)
  return component
}

// 检查字段是否必填
function isFieldRequired(field: FieldConfig): boolean {
  return hasAnyRequiredRule(field)
}

// 字段验证错误（field_code -> ValidationResult[]）
const fieldErrors = reactive<Map<string, ValidationResult[]>>(new Map())

// 验证引擎（适配 formDataStore）
const validationEngine = computed(() => {
  const validatorRegistry = createDefaultValidatorRegistry()
  const allFields = props.functionDetail?.request || []
  
  // 创建适配器，将 formDataStore 转换为 ReactiveFormDataManager 接口
  const formManagerAdapter = {
    getValue: (fieldPath: string) => {
      return formDataStore.getValue(fieldPath)
    },
    getAllValues: () => {
      const allValues: Record<string, FieldValue> = {}
      allFields.forEach(f => {
        allValues[f.code] = formDataStore.getValue(f.code)
      })
      return allValues
    }
  } as any
  
  return new ValidationEngine(validatorRegistry, formManagerAdapter, allFields)
})

/**
 * 获取字段错误消息（用于显示在表单项下方）
 */
function getFieldError(fieldCode: string): string {
  const errors = fieldErrors.get(fieldCode)
  if (!errors || errors.length === 0) {
  return ''
  }
  return errors[0].message || ''
}

/**
 * 根据字段路径获取字段名称
 */
function getFieldNameByPath(fieldPath: string): string {
  // 尝试从顶层字段中查找
  const topLevelField = requestFields.value.find((f: FieldConfig) => fieldPath === f.code)
  if (topLevelField) {
    return topLevelField.name
  }
  
  // 处理嵌套字段路径（如 customer.basic_info.name）
  const pathParts = fieldPath.split('.')
  if (pathParts.length > 1 && pathParts[0]) {
    // 查找顶层字段
    const topField = requestFields.value.find((f: FieldConfig) => f.code === pathParts[0])
    if (topField && topField.children) {
      // 递归查找嵌套字段
      let currentField: FieldConfig | undefined = topField
      for (let i = 1; i < pathParts.length; i++) {
        const part = pathParts[i]
        if (!part || !currentField) break
        
        // 处理数组索引（如 products[0].name）
        const fieldCode = part.replace(/\[\d+\]/, '')
        currentField = currentField.children?.find((f: FieldConfig) => f.code === fieldCode)
        if (!currentField) break
      }
      if (currentField) {
        return currentField.name
      }
    }
  }
  
  // 处理数组索引路径（如 products[0].name）
  const arrayMatch = fieldPath.match(/^(.+)\[(\d+)\]\.(.+)$/)
  if (arrayMatch && arrayMatch[1] && arrayMatch[3]) {
    const parentPath = arrayMatch[1]
    const fieldCode = arrayMatch[3]
    const topField = requestFields.value.find((f: FieldConfig) => f.code === parentPath.split('.')[0])
    if (topField && topField.children) {
      const field = topField.children.find((f: FieldConfig) => f.code === fieldCode)
      if (field) {
        return field.name
      }
    }
  }
  
  // 如果找不到，返回字段路径
  return fieldPath
}

/**
 * 收集所有错误消息（包含字段名称）
 */
function collectErrorMessages(): string[] {
  const messages: string[] = []
  fieldErrors.forEach((errors: ValidationResult[], fieldPath: string) => {
    const fieldName = getFieldNameByPath(fieldPath)
    errors.forEach((err: ValidationResult) => {
      if (err.message) {
        // 如果错误消息已经包含字段名，直接使用；否则添加字段名
        const message = err.message.includes(fieldName) 
          ? err.message 
          : `${fieldName}：${err.message}`
        messages.push(message)
      }
    })
  })
  return messages
}

/**
 * 生成友好的错误提示消息
 */
function generateErrorMessage(): string {
  const errorMessages = collectErrorMessages()
  const errorCount = fieldErrors.size
  
  if (errorCount === 0) {
    return '请检查表单中的必填项和错误'
  }
  
  if (errorCount === 1) {
    // 只有一个错误，直接显示
    return errorMessages[0] || '请检查表单中的必填项和错误'
  }
  
  // 多个错误，显示汇总信息
  const uniqueMessages = Array.from(new Set(errorMessages))
  if (uniqueMessages.length <= 3) {
    // 错误数量少，显示所有错误
    return `请检查以下字段：${uniqueMessages.join('；')}`
  } else {
    // 错误数量多，只显示前几个
    return `请检查以下字段：${uniqueMessages.slice(0, 3).join('；')}等共 ${errorCount} 个字段`
  }
}

/**
 * 验证单个字段
 * 
 * 符合依赖倒置原则：让 Widget 自己负责验证逻辑
 * - 容器 Widget（FormWidget、TableWidget）：通过 ref 调用其 validate 方法，自行处理嵌套字段
 * - 基础 Widget：直接使用验证引擎验证
 */
function validateField(field: FieldConfig): void {
  const fieldPath = field.code
  const allFields = props.functionDetail?.request || []
  const widgetRef = widgetRefs.get(fieldPath)
  
  // 容器 Widget：通过 ref 调用其 validate 方法（会递归验证嵌套字段）
  if (widgetRef && typeof widgetRef.validate === 'function') {
    const errors = widgetRef.validate(validationEngine.value, allFields, fieldErrors)
    updateFieldErrors(fieldPath, errors)
    return
  }
  
  // 基础 Widget：直接验证
  const value = formDataStore.getValue(fieldPath)
  if (field.validation) {
    const errors = validationEngine.value.validateField(field, value, allFields)
    updateFieldErrors(fieldPath, errors)
  } else {
    fieldErrors.delete(fieldPath)
  }
}

/**
 * 更新字段错误状态
 */
function updateFieldErrors(fieldPath: string, errors: ValidationResult[]): void {
  if (errors && errors.length > 0) {
    fieldErrors.set(fieldPath, errors)
  } else {
    fieldErrors.delete(fieldPath)
  }
}

/**
 * 验证所有字段
 * 
 * 符合依赖倒置原则：只验证顶层字段，嵌套字段的验证由 Widget 自己负责
 * 
 * @returns 是否有验证错误
 */
function validateAllFields(): boolean {
  fieldErrors.clear()
  
  // 验证所有顶层字段（嵌套字段由 Widget 自行验证）
  requestFields.value.forEach((field: FieldConfig) => {
    validateField(field)
  })
  
  // 检查是否有错误（包括嵌套字段的错误）
  let hasError = false
  fieldErrors.forEach((errors) => {
    if (errors && errors.length > 0) {
      hasError = true
    }
  })
  
  if (hasError) {
    Logger.warn('[FormRenderer-v2]', '表单验证失败', {
      errorCount: fieldErrors.size,
      errors: Array.from(fieldErrors.entries()).map(([path, errs]) => ({
        path,
        messages: errs.map(e => e.message)
      }))
    })
  }
  
  return hasError
}

// FormRenderer 上下文（兼容旧接口）
const formManager = null as any // 不再使用 ReactiveFormDataManager
const formRendererContext: FormRendererContext = {
  registerWidget: () => {},
  unregisterWidget: () => {},
  getFunctionMethod: () => props.functionDetail.method,
  getFunctionRouter: () => props.functionDetail.router,
  getFunctionDetail: () => props.functionDetail, // 🔥 获取函数详情（用于防重复调用）
  getSubmitData: () => formDataStore.getSubmitData(requestFields.value),
  getFieldError: (fieldPath: string) => getFieldError(fieldPath) // 🔥 获取字段错误
}

/**
 * 条件渲染评估（适配 formDataStore）
 * 
 * ⚠️ 重要：条件渲染初始化时的值获取问题
 * 
 * 问题场景：
 * - 字段 A 有验证规则 `required_if=FieldB value`，表示只有当 FieldB 等于 value 时才显示
 * - 在表单初始化时，`requestFields` computed 会计算哪些字段应该显示
 * - 但此时 `formDataStore` 还是空的，导致条件渲染无法获取 FieldB 的值
 * - 结果：字段 A 被错误地过滤掉，即使 initialData 中有 FieldB 的值
 * 
 * 典型案例：
 * - `max_selections` 字段有规则 `required_if=VoteType 多选`
 * - 初始化时，`vote_type` 的值在 `initialData` 中（值为 "多选"）
 * - 但 `formDataStore` 中还没有值，导致条件渲染判断失败
 * - `max_selections` 被过滤，无法显示和初始化
 * 
 * 解决方案：
 * - 在条件渲染时，如果 `formDataStore` 中没有值，尝试从 `initialData` 中获取
 * - 这样可以确保在初始化时，条件渲染能正确判断字段是否应该显示
 * 
 * @param field 字段配置
 * @param formDataStore 表单数据 store
 * @param allFields 所有字段配置
 * @returns 是否应该显示该字段
 */
function shouldShowFieldInForm(
  field: FieldConfig,
  formDataStore: ReturnType<typeof useFormDataStore>,
  allFields: FieldConfig[]
): boolean {
  // 创建一个适配器，将 formDataStore 转换为 ReactiveFormDataManager 接口
  const formManagerAdapter = {
    getValue: (fieldPath: string) => {
      let value = formDataStore.getValue(fieldPath)
      
      // ⚠️ 关键修复：如果 formDataStore 中没有值，且 initialData 中有值，使用 initialData 的值
      // 这样可以确保在初始化时，条件渲染能正确判断字段是否应该显示
      // 例如：max_selections 字段依赖 vote_type 的值，在初始化时需要从 initialData 中获取 vote_type
      if ((!value || value.raw === null || value.raw === undefined) && 
          props.initialData && 
          props.initialData.hasOwnProperty(fieldPath) &&
          props.initialData[fieldPath] !== undefined) {
        const rawValue = props.initialData[fieldPath]
        value = {
          raw: rawValue,
          display: typeof rawValue === 'object' ? JSON.stringify(rawValue) : String(rawValue),
          meta: {}
        }
      }
      
      return value
    },
    getAllValues: () => {
      const allValues: Record<string, FieldValue> = {}
      allFields.forEach(f => {
        let value = formDataStore.getValue(f.code)
        
        // ⚠️ 关键修复：同上，确保 getAllValues 也能从 initialData 中获取值
        if ((!value || value.raw === null || value.raw === undefined) && 
            props.initialData && 
            props.initialData.hasOwnProperty(f.code) &&
            props.initialData[f.code] !== undefined) {
          const rawValue = props.initialData[f.code]
          value = {
            raw: rawValue,
            display: typeof rawValue === 'object' ? JSON.stringify(rawValue) : String(rawValue),
            meta: {}
          }
        }
        
        allValues[f.code] = value
      })
      return allValues
    }
  } as any
  
  // 使用现有的 shouldShowField 函数
  return shouldShowField(field, formManagerAdapter, allFields)
}

// 获取字段默认值
// 🔥 遵循依赖倒置原则：调用组件自己的默认值获取方法
function getFieldDefaultValue(field: FieldConfig): FieldValue {
  // 🔥 提供 getAuthStore 函数，用于解析 $me 动态变量
  return getWidgetDefaultValue(field, undefined, () => useAuthStore())
}

/**
 * 初始化表单
 * 
 * ⚠️ 注意：字段初始化顺序很重要
 * - `requestFields` 是一个 computed，会根据条件渲染规则过滤字段
 * - 条件渲染依赖其他字段的值（如 `required_if=FieldB value`）
 * - 在初始化时，`shouldShowFieldInForm` 会从 `initialData` 中获取值用于条件判断
 * - 这样可以确保依赖字段（如 `vote_type`）的值能被正确读取，从而显示被依赖的字段（如 `max_selections`）
 */
function initializeForm(): void {
  // 初始化字段值
  // ⚠️ 注意：requestFields 已经通过条件渲染过滤，只包含应该显示的字段
  // 条件渲染在 shouldShowFieldInForm 中会从 initialData 获取值，确保正确判断
  requestFields.value.forEach((field: FieldConfig) => {
    const fieldCode = field.code
    
    // 如果有初始数据，使用初始数据
    // 使用 hasOwnProperty 确保字段存在且值不为 undefined
    if (props.initialData && 
        props.initialData.hasOwnProperty(fieldCode) && 
        props.initialData[fieldCode] !== undefined) {
      const initialRawValue = props.initialData[fieldCode]
      const fieldValue: FieldValue = {
        raw: initialRawValue,
        display: typeof initialRawValue === 'object' ? JSON.stringify(initialRawValue) : String(initialRawValue),
        meta: {}
      }
      
      formDataStore.setValue(fieldCode, fieldValue)
    } else {
      // 使用默认值（从字段配置中获取）
      const defaultValue = getFieldDefaultValue(field)
      formDataStore.initializeField(fieldCode, defaultValue)
    }
  })
}

// 重置表单
function handleReset(): void {
  initializeForm()
  ElMessage.success('表单已重置')
}

// 提交表单
/**
 * 提交表单
 */
async function handleSubmit(): Promise<void> {
  // 验证所有字段
  const hasError = validateAllFields()
  
  if (hasError) {
    // 生成并显示友好的错误提示
    const errorMessage = generateErrorMessage()
    ElMessage.error(errorMessage)
    
    // TODO: 实现滚动到第一个错误字段
    return
  }
  
  // 🔥 显示确认框，防止误触
  try {
    await ElMessageBox.confirm(
      '确定要提交表单吗？',
      '确认提交',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
        center: true
      }
    )
  } catch {
    // 用户取消提交
    return
  }
  
  // 验证通过，用户确认提交，开始提交
  
  submitting.value = true
  
  try {
    // 获取提交数据
    const submitData = formDataStore.getSubmitData(requestFields.value)
    
    Logger.info('[FormRenderer-v2]', '提交数据', submitData)
    
    // 调用后端 API
    const response = await executeFunction(
      props.functionDetail.method,
      props.functionDetail.router,
      submitData
    )
    
    // 保存返回值
    // 🔥 注意：request 拦截器已经提取了 data 字段，所以 response 就是 data 的内容
    // 直接使用 response 即可
    const newResponseData = response && typeof response === 'object' 
      ? response 
      : { result: response }
    
    Logger.info('[FormRenderer-v2]', '保存响应数据', newResponseData)
    Logger.info('[FormRenderer-v2]', '响应数据类型:', typeof newResponseData, '是否为对象:', typeof newResponseData === 'object')
    
    // 保存数据
    Logger.info('[FormRenderer-v2]', '调用 setData 前，responseDataStore:', responseDataStore)
    Logger.info('[FormRenderer-v2]', '调用 setData 前，data:', responseDataStore.data)
    // 🔥 保存响应数据
    responseDataStore.setData(newResponseData)
    Logger.info('[FormRenderer-v2]', '调用 setData 后，data:', responseDataStore.data)
    Logger.info('[FormRenderer-v2]', '调用 setData 后，data.value:', responseDataStore.data.value)
    
    // 等待一个 tick，确保 computed 更新
    await nextTick()
    
    // 验证数据是否已保存
    Logger.info('[FormRenderer-v2]', '保存后的 renderTrigger:', responseDataStore.renderTrigger)
    Logger.info('[FormRenderer-v2]', '保存后的 data 对象:', responseDataStore.data)
    Logger.info('[FormRenderer-v2]', '保存后的 data.value:', responseDataStore.data?.value)
    Logger.info('[FormRenderer-v2]', '保存后的 data (直接访问):', responseDataStore.data)
    Logger.info('[FormRenderer-v2]', 'responseFieldValues 值:', responseFieldValues.value)
    
    // 🔥 强制触发一次响应式更新
    await nextTick()
    Logger.info('[FormRenderer-v2]', 'nextTick 后的 responseFieldValues:', responseFieldValues.value)
    
    // 保存提交结果（用于调试）
    submitResult.value = submitData
    
    ElMessage.success('表单提交成功！')
  } catch (error: any) {
    // 🔥 输出详细的错误信息
    Logger.error('[FormRenderer-v2]', '提交失败', error)
    Logger.error('[FormRenderer-v2]', '错误详情:', {
      message: error?.message,
      response: error?.response,
      data: error?.response?.data,
      status: error?.response?.status,
      code: error?.response?.data?.code,
      msg: error?.response?.data?.msg
    })
    
    // 🔥 统一使用 msg 字段
    const errorMessage = error?.response?.data?.msg || error?.message || '提交失败'
    ElMessage.error(errorMessage)
  } finally {
    submitting.value = false
  }
}

/**
 * 准备提交数据（带类型转换）
 * 这个方法会被 FormDialog 等外部组件调用
 */
function prepareSubmitDataWithTypeConversion(): Record<string, any> {
  if (!props.functionDetail?.request) {
    return {}
  }
  
  // 使用 formDataStore 的 getSubmitData 方法递归收集所有字段的数据
  const submitData = formDataStore.getSubmitData(props.functionDetail.request)
  
  Logger.info('[FormRenderer-v2]', '准备提交数据', submitData)
  
  return submitData
}

// 清理函数
function cleanup(): void {
  // 先设置 isMounted 为 false，阻止渲染
  isMounted.value = false
  // 等待一个 tick，确保组件停止渲染
  nextTick(() => {
    // 清理数据
    formDataStore.clear()
    // 🔥 清理响应数据
    responseDataStore.clear()
  })
}

// 监听 functionDetail 变化，在路由切换时清理
watch(
  () => props.functionDetail?.id || props.functionDetail?.router,
  async (newId, oldId) => {
    if (oldId && newId !== oldId) {
      // 路由切换，先清理旧数据
      cleanup()
      // 等待 DOM 更新完成
      await nextTick()
      await nextTick()
      // 重新初始化
      isMounted.value = true
      await nextTick()
      initializeForm()
    }
  },
  { flush: 'post' } // 在 DOM 更新后执行
)

/**
 * 监听 initialData 变化，当初始数据变化时重新初始化表单
 * 
 * ⚠️ 使用场景：
 * - 从查看模式切换到编辑模式时，`initialData` 会变化
 * - 如果 `FormRenderer` 已经挂载，需要重新初始化表单以填充新数据
 * - 例如：在 TableRenderer 的详情抽屉中，点击"编辑"按钮时
 * 
 * ⚠️ 注意：
 * - 只在组件已挂载时重新初始化（避免在初始化时重复初始化）
 * - 使用深度比较避免不必要的重新初始化
 */
watch(
  () => props.initialData,
  async (newData, oldData) => {
    // 只在组件已挂载时重新初始化（避免在初始化时重复初始化）
    if (!isMounted.value) {
      return
    }
    
    // 判断 initialData 是否真的变化了（避免不必要的重新初始化）
    // 使用 JSON.stringify 进行深度比较（对于简单对象足够）
    const newDataStr = JSON.stringify(newData || {})
    const oldDataStr = JSON.stringify(oldData || {})
    if (newDataStr === oldDataStr) {
      return
    }
    
    // initialData 变化，重新初始化表单
    await nextTick()
    initializeForm()
  },
  { deep: true, flush: 'post' } // 深度监听，在 DOM 更新后执行
)

// 生命周期
onMounted(async () => {
  // 延迟挂载，确保 DOM 已准备好
  await nextTick()
  isMounted.value = true
  initializeForm()
})

onBeforeUnmount(() => {
  // 清理工作
  cleanup()
})

// 暴露方法给外部组件（如 FormDialog）使用
defineExpose({
  prepareSubmitDataWithTypeConversion
})
</script>

<style scoped>
.form-renderer-v2 {
  width: 100%;
  padding: 20px;
}

.section-title {
  font-size: 16px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 16px;
  margin-top: 24px;
}

.section-title:first-child {
  margin-top: 0;
}

.form-actions-section {
  margin-top: 24px;
  margin-bottom: 24px;
}

.form-actions-row {
  display: flex;
  gap: 12px;
}

.submit-button-full-width {
  flex: 1;
}

.response-container.is-empty {
  opacity: 0.6;
}

.result-card {
  margin-top: 20px;
}

.result-content {
  font-family: monospace;
  font-size: 12px;
}

/* 表单项间距优化 */
:deep(.function-form .request-form-item),
:deep(.function-form .response-form-item) {
  margin-bottom: 24px;
}

:deep(.function-form .request-form-item:last-child),
:deep(.function-form .response-form-item:last-child) {
  margin-bottom: 0;
}
</style>

