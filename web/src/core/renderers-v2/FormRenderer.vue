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
      >
        <component
          v-if="getWidgetComponent(field.widget?.type || 'input')"
          :key="`request_widget_${field.code}_${field.widget?.type || 'input'}`"
          :is="getWidgetComponent(field.widget?.type || 'input')"
          :field="field"
          :value="getFieldValue(field.code)"
          :model-value="getFieldValue(field.code)"
          @update:model-value="(v) => updateFieldValue(field.code, v)"
          :field-path="field.code"
          :form-manager="formManager"
          :form-renderer="formRendererContext"
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
        >
          <component
            v-if="getResponseWidgetComponent(field.widget?.type || 'input')"
            :key="`response_widget_${field.code}_${field.widget?.type || 'input'}_${responseDataStore.renderTrigger}`"
            :is="getResponseWidgetComponent(field.widget?.type || 'input')"
            :field="field"
            :value="responseFieldValues[field.code]"
            :model-value="responseFieldValues[field.code]"
            :field-path="field.code"
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
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { ElForm, ElFormItem, ElButton, ElCard, ElMessage, ElIcon, ElTag } from 'element-plus'
import { Promotion, RefreshLeft } from '@element-plus/icons-vue'
import type { FieldConfig, FunctionDetail, FieldValue } from '../types/field'
import { useFormDataStore } from '../stores-v2/formData'
import { useResponseDataStore } from '../stores-v2/responseData'
import { widgetComponentFactory } from '../factories-v2'
import { executeFunction } from '@/api/function'
import { Logger } from '../utils/logger'
import { shouldShowField } from '../utils/conditionEvaluator'
import { hasAnyRequiredRule } from '../utils/validationUtils'
import type { ReactiveFormDataManager } from '../managers/ReactiveFormDataManager'
import type { FormRendererContext } from '../types/widget'

const props = withDefaults(defineProps<{
  functionDetail: FunctionDetail
  showSubmitButton?: boolean
  showResetButton?: boolean
  initialData?: Record<string, any>
}>(), {
  showSubmitButton: true,
  showResetButton: true,
  initialData: () => ({})
})

// Pinia Stores
const formDataStore = useFormDataStore()
const responseDataStore = useResponseDataStore()

// 表单引用
const formRef = ref()

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
  if (!responseDataStore || !responseDataStore.data) {
    return false
  }
  // 读取 renderTrigger 作为依赖，确保数据更新时重新计算
  const trigger = responseDataStore.renderTrigger
  const data = responseDataStore.data.value
  return data !== null && data !== undefined
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
}

// 获取响应字段值
// 🔥 为每个字段创建 computed，确保响应式更新
const getResponseFieldValue = (fieldCode: string): FieldValue => {
  // 读取 renderTrigger 作为依赖，确保数据更新时重新计算
  const trigger = responseDataStore.renderTrigger
  const responseData = responseDataStore.data.value
  
  // 🔥 添加日志以便调试
  Logger.debug('[FormRenderer-v2]', `getResponseFieldValue: fieldCode=${fieldCode}, trigger=${trigger}, responseData=`, responseData)
  
  if (!responseData) {
    return {
      raw: null,
      display: '',
      meta: {}
    }
  }
  
  const rawValue = responseData[fieldCode]
  
  Logger.debug('[FormRenderer-v2]', `getResponseFieldValue: rawValue=`, rawValue)
  
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
  if (!isMounted.value) {
    return {}
  }
  
  const trigger = responseDataStore.renderTrigger
  const responseData = responseDataStore.data.value
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
    
    values[field.code] = {
      raw: rawValue ?? null,
      display: rawValue !== null && rawValue !== undefined 
        ? (typeof rawValue === 'object' ? JSON.stringify(rawValue) : String(rawValue))
        : '',
      meta: {}
    }
  })
  
  return values
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
    console.warn(`[FormRenderer-v2] 未找到组件: ${type}，使用默认 InputWidget`)
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
    console.warn(`[FormRenderer-v2] 未找到响应组件: ${type}，使用默认 InputWidget`)
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

// 获取字段错误
function getFieldError(fieldCode: string): string {
  // TODO: 集成验证引擎
  return ''
}

// FormRenderer 上下文（兼容旧接口）
const formManager = null as any // 不再使用 ReactiveFormDataManager
const formRendererContext: FormRendererContext = {
  registerWidget: () => {},
  unregisterWidget: () => {},
  getFunctionMethod: () => props.functionDetail.method,
  getFunctionRouter: () => props.functionDetail.router,
  getSubmitData: () => formDataStore.getSubmitData(requestFields.value)
}

// 条件渲染评估（适配 formDataStore）
function shouldShowFieldInForm(
  field: FieldConfig,
  formDataStore: ReturnType<typeof useFormDataStore>,
  allFields: FieldConfig[]
): boolean {
  // 创建一个适配器，将 formDataStore 转换为 ReactiveFormDataManager 接口
  const formManagerAdapter = {
    getValue: (fieldPath: string) => {
      const value = formDataStore.getValue(fieldPath)
      return value
    },
    getAllValues: () => {
      const allValues: Record<string, FieldValue> = {}
      allFields.forEach(f => {
        allValues[f.code] = formDataStore.getValue(f.code)
      })
      return allValues
    }
  } as any
  
  // 使用现有的 shouldShowField 函数
  return shouldShowField(field, formManagerAdapter, allFields)
}

// 初始化表单
function initializeForm(): void {
  // 清空数据
  formDataStore.clear()
  responseDataStore.clear()
  
  // 初始化字段值
  requestFields.value.forEach((field: FieldConfig) => {
    const fieldCode = field.code
    
    // 如果有初始数据，使用初始数据
    if (props.initialData && fieldCode in props.initialData) {
      const initialRawValue = props.initialData[fieldCode]
      const fieldValue: FieldValue = {
        raw: initialRawValue,
        display: typeof initialRawValue === 'object' ? JSON.stringify(initialRawValue) : String(initialRawValue),
        meta: {}
      }
      formDataStore.setValue(fieldCode, fieldValue)
    } else {
      // 使用默认值
      formDataStore.initializeField(fieldCode)
    }
  })
}

// 重置表单
function handleReset(): void {
  initializeForm()
  ElMessage.success('表单已重置')
}

// 提交表单
async function handleSubmit(): Promise<void> {
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
    
    responseDataStore.setData(newResponseData)
    
    // 保存提交结果（用于调试）
    submitResult.value = submitData
    
    ElMessage.success('表单提交成功！')
  } catch (error: any) {
    Logger.error('[FormRenderer-v2]', '提交失败', error)
    ElMessage.error(error?.message || '提交失败')
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
  const submitData = formDataStore.getSubmitData(props.functionDetail.request, '')
  
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
</style>

