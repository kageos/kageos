<!--
  FormView - 表单视图
  🔥 新架构的展示层组件
  
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
          :value="getFieldValue(field.code)"
          @update:model-value="(v) => handleFieldUpdate(field.code, v)"
        />
      </el-form-item>
    </el-form>

    <!-- 提交按钮 -->
    <div class="form-actions">
      <el-button
        type="primary"
        size="large"
        @click="handleSubmit"
        :loading="submitting"
      >
        提交
      </el-button>
      <el-button size="large" @click="handleReset">
        重置
      </el-button>
    </div>

    <!-- 响应参数展示 -->
    <div v-if="hasResponseData" class="response-section">
      <div class="section-title">响应参数</div>
      <el-form label-width="100px">
        <el-form-item
          v-for="field in responseFields"
          :key="field.code"
          :label="field.name"
        >
          <WidgetComponent
            :field="field"
            :value="getResponseFieldValue(field.code)"
            mode="response"
          />
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { eventBus, FormEvent, WorkspaceEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import WidgetComponent from '../widgets/WidgetComponent.vue'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import { hasAnyRequiredRule } from '@/core/utils/validationUtils'

const props = defineProps<{
  functionDetail: FunctionDetail
}>()

// 依赖注入（使用 ServiceFactory 简化）
const stateManager = serviceFactory.getFormStateManager()
const domainService = serviceFactory.getFormDomainService()
const applicationService = serviceFactory.getFormApplicationService()

// 从状态管理器获取状态
const formData = computed(() => {
  const state = stateManager.getState()
  const data: Record<string, any> = {}
  state.data.forEach((value, key) => {
    data[key] = value.raw
  })
  return data
})

const requestFields = computed(() => (props.functionDetail.request || []) as FieldConfig[])
const responseFields = computed(() => (props.functionDetail.response || []) as FieldConfig[])

const submitting = computed(() => {
  const state = stateManager.getState()
  return state.submitting
})

const hasResponseData = computed(() => {
  // TODO: 从状态管理器获取响应数据
  return false
})

// 方法
const getFieldValue = (fieldCode: string): FieldValue => {
  return domainService.getFieldValue(fieldCode)
}

const getFieldError = (fieldCode: string): string => {
  const errors = domainService.getFieldError(fieldCode)
  return errors[0]?.message || ''
}

const getResponseFieldValue = (fieldCode: string): FieldValue => {
  // TODO: 从响应数据中获取
  return { raw: null, display: '', meta: {} }
}

const isFieldRequired = (field: FieldConfig): boolean => {
  return hasAnyRequiredRule(field.validation || '')
}

const handleFieldUpdate = (fieldCode: string, value: FieldValue): void => {
  applicationService.updateFieldValue(fieldCode, value)
}

const handleSubmit = async (): Promise<void> => {
  try {
    await applicationService.submitForm(props.functionDetail)
  } catch (error: any) {
    console.error('表单提交失败:', error)
    // TODO: 显示错误提示
  }
}

const handleReset = (): void => {
  applicationService.clearForm()
  // 重新初始化表单
  applicationService.initializeForm(requestFields.value)
}

// 生命周期
let unsubscribeFunctionLoaded: (() => void) | null = null
let unsubscribeFormInitialized: (() => void) | null = null

onMounted(() => {
  // 监听函数加载完成事件
  unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, (payload: { detail: FunctionDetail }) => {
    if (payload.detail.template_type === 'form') {
      // Application Service 会自动处理
    }
  })

  // 监听表单初始化完成事件
  unsubscribeFormInitialized = eventBus.on(FormEvent.initialized, () => {
    // 表单已初始化，可以渲染
  })
})

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

.response-section {
  margin-top: 40px;
  padding-top: 20px;
  border-top: 1px solid var(--el-border-color);
}
</style>

