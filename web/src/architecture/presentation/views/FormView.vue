<!--
  FormView - 表单视图
  新架构的展示层组件
  
  职责：
  - 纯 UI 展示，不包含业务逻辑
  - 通过事件与 Application Layer 通信
  - 从 StateManager 获取状态并渲染
-->

<template>
  <div class="form-view">
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
          :function-method="props.functionDetail.method || 'GET'"
          :function-router="props.functionDetail.router || ''"
          @update:model-value="(v: FieldValue) => handleFieldUpdate(field.code, v)"
        />
      </el-form-item>
    </el-form>

    <!-- 提交按钮 -->
    <div class="form-actions-section">
      <div class="form-actions-row">
        <el-button
          type="primary"
          size="large"
          @click="handleSubmit"
          :loading="submitting"
          class="submit-button-full-width"
        >
          <el-icon><Promotion /></el-icon>
          提交
        </el-button>
        <el-button size="large" @click="handleReset">
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
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch, ref, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { Promotion, RefreshLeft, View, DocumentCopy } from '@element-plus/icons-vue'
import { ElIcon, ElTag, ElNotification, ElMessage } from 'element-plus'
import { eventBus, FormEvent, WorkspaceEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import WidgetComponent from '../widgets/WidgetComponent.vue'
import { Logger } from '@/core/utils/logger'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import { hasAnyRequiredRule } from '@/core/utils/validationUtils'

const props = defineProps<{
  functionDetail: FunctionDetail
}>()

// 路由
const route = useRoute()

// 依赖注入（使用 ServiceFactory 简化）
const stateManager = serviceFactory.getFormStateManager()
const domainService = serviceFactory.getFormDomainService()
const applicationService = serviceFactory.getFormApplicationService()

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

const requestFields = computed(() => (props.functionDetail.request || []) as FieldConfig[])
const responseFields = computed(() => (props.functionDetail.response || []) as FieldConfig[])

// 从 URL 查询参数中提取表单初始数据
const formInitialData = computed(() => {
  const initialData: Record<string, any> = {}
  const query = route.query
  
  // 遍历所有查询参数，如果字段在 request 中，添加到 initialData
  if (props.functionDetail?.request) {
    props.functionDetail.request.forEach((field: FieldConfig) => {
      const fieldCode = field.code
      const queryValue = query[fieldCode]
      
      // 🔥 处理数组类型的查询参数（取第一个值）
      const value = Array.isArray(queryValue) ? queryValue[0] : queryValue
      
      if (value !== undefined && value !== null && value !== '') {
        // 类型转换：根据字段类型转换值
        if (field.data?.type === 'int' || field.data?.type === 'integer') {
          const intValue = parseInt(String(value), 10)
          if (!isNaN(intValue)) {
            initialData[fieldCode] = intValue
          }
        } else if (field.data?.type === 'float' || field.data?.type === 'number') {
          const floatValue = parseFloat(String(value))
          if (!isNaN(floatValue)) {
            initialData[fieldCode] = floatValue
          }
        } else if (field.data?.type === 'bool' || field.data?.type === 'boolean') {
          const strValue = String(value)
          initialData[fieldCode] = strValue === 'true' || strValue === '1'
        } else {
          initialData[fieldCode] = value
        }
      }
    })
  }
  
  return initialData
})

// 🔥 为所有字段创建响应式的值 Map
const fieldValues = computed(() => {
  const state = stateManager.getState()
  const values: Record<string, FieldValue> = {}
  requestFields.value.forEach((field: FieldConfig) => {
    values[field.code] = state.data.get(field.code) || { raw: null, display: '', meta: {} }
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
    getFunctionMethod: () => props.functionDetail.method || 'GET',
    getFunctionRouter: () => props.functionDetail.router || '',
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
    await applicationService.submitForm(props.functionDetail)
    
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
    
    // 尝试从 error.response.data 中获取错误消息（request.ts 第 99-101 行）
    if (error?.response?.data) {
      const responseData = error.response.data
      // 优先使用 msg，其次使用 message
      errorMessage = responseData.msg || responseData.message || errorMessage
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
  applicationService.clearForm()
  // 重新初始化表单
  const fields = requestFields.value
  if (fields.length > 0) {
    applicationService.initializeForm(fields)
  }
}

// 生命周期
let unsubscribeFunctionLoaded: (() => void) | null = null
let unsubscribeFormInitialized: (() => void) | null = null

onMounted(() => {
  // 初始化表单：在挂载时立即初始化，并传递 URL 参数作为初始数据
  if (requestFields.value.length > 0) {
    const initialData = formInitialData.value
    applicationService.initializeForm(requestFields.value, initialData)
  }

  // 监听函数加载完成事件
  let lastInitializedFunctionId: number | null = null // 🔥 记录上次初始化的函数 ID，防止重复初始化
  unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, (payload: { detail: FunctionDetail }) => {
    if (payload.detail.template_type === TEMPLATE_TYPE.FORM && payload.detail.id === props.functionDetail.id) {
      // 🔥 防重复初始化：如果已经初始化过这个函数，跳过
      if (lastInitializedFunctionId === payload.detail.id) {
        Logger.debug('FormView', '跳过重复的 functionLoaded 事件', { functionId: payload.detail.id })
        return
      }
      lastInitializedFunctionId = payload.detail.id
      
      // 🔥 使用 nextTick 确保 formInitialData 已经更新（因为它依赖于 route.query）
      nextTick(() => {
        // 重新初始化表单（传递 URL 参数作为初始数据）
        const fields = (payload.detail.request || []) as FieldConfig[]
        if (fields.length > 0) {
          const initialData = formInitialData.value
          applicationService.initializeForm(fields, initialData)
        }
      })
    }
  })

  // 监听表单初始化完成事件
  unsubscribeFormInitialized = eventBus.on(FormEvent.initialized, () => {
    // 表单已初始化，可以渲染
  })
})

  // 🔥 监听 functionDetail 变化，重新初始化表单
  // 注意：只在 functionDetail 真正变化时（id 或 router 变化）才重新初始化
  // 如果只是 URL 参数变化，不应该触发这个 watch
  watch(() => props.functionDetail, (newDetail: FunctionDetail, oldDetail?: FunctionDetail) => {
    // 🔥 只在 functionDetail 的 id 或 router 真正变化时重新初始化
    // 如果只是其他属性变化（如字段配置），不应该重新初始化
    if (newDetail.id !== oldDetail?.id || newDetail.router !== oldDetail?.router) {
      const fields = (newDetail.request || []) as FieldConfig[]
      if (fields.length > 0) {
        // 🔥 使用 nextTick 确保 formInitialData 已经更新（因为它依赖于 route.query）
        nextTick(() => {
          // 🔥 重新初始化时，传递 URL 参数作为初始数据，确保 URL 参数不会被清空
          const initialData = formInitialData.value
          applicationService.initializeForm(fields, initialData)
        })
      }
    }
  }, { deep: false }) // 🔥 改为 shallow watch，避免深度监听导致不必要的触发

  // 🔥 移除 watch route.query，改为在表单初始化时统一处理 URL 参数
  // 这样可以避免时序问题，确保表单完全初始化后再处理 URL 参数
  // URL 参数会在 initializeForm 时通过 initialData 传递，FormDomainService 会正确处理

onUnmounted(() => {
  if (unsubscribeFunctionLoaded) {
    unsubscribeFunctionLoaded()
  }
  if (unsubscribeFormInitialized) {
    unsubscribeFormInitialized()
  }
})
</script>

<style scoped>
.form-view {
  padding: 20px;
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
</style>

