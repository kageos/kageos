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
          :function-method="functionDetail?.method || 'GET'"
          :function-router="functionDetail?.router || ''"
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
        <el-button size="large" @click="handleSaveQuickLink" type="info">
          <el-icon><Link /></el-icon>
          保存快链
        </el-button>
        <el-button size="large" @click="showQuickLinkListDialog = true" type="info">
          <el-icon><List /></el-icon>
          快链列表
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

    <!-- 快链名称输入弹窗 -->
    <el-dialog
      v-model="showQuickLinkNameDialog"
      title="保存快链"
      width="500px"
      :close-on-click-modal="false"
    >
      <div class="quicklink-name-dialog-content">
        <el-form :model="quickLinkForm" label-width="100px">
          <el-form-item label="快链名称" required>
            <el-input
              v-model="quickLinkForm.name"
              placeholder="请输入快链名称"
              maxlength="100"
              show-word-limit
              @keyup.enter="confirmSaveQuickLink"
            />
          </el-form-item>
          <el-form-item label="保存选项">
            <el-checkbox
              v-model="quickLinkForm.saveResponseParams"
              :disabled="!hasResponseData"
            >
              同时保存响应参数
            </el-checkbox>
            <div v-if="!hasResponseData" class="form-item-hint">
              <el-text type="info" size="small">
                当前没有响应数据，请先提交表单后再保存快链
              </el-text>
            </div>
            <div v-else class="form-item-hint">
              <el-text type="info" size="small">
                勾选后将保存当前表单的响应结果，适用于计算结果缓存等场景
              </el-text>
            </div>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showQuickLinkNameDialog = false">取消</el-button>
          <el-button
            type="primary"
            @click="confirmSaveQuickLink"
            :disabled="!quickLinkForm.name || quickLinkForm.name.trim() === ''"
          >
            保存
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 快链地址弹窗 -->
    <el-dialog
      v-model="showQuickLinkDialog"
      title="快链保存成功"
      width="600px"
      :close-on-click-modal="false"
    >
      <div class="quicklink-dialog-content">
        <div class="quicklink-info">
          <p>快链已保存，您可以通过以下链接访问：</p>
        </div>
        <div class="quicklink-url-section">
          <el-input
            v-model="quickLinkUrl"
            readonly
            class="quicklink-url-input"
          >
            <template #append>
              <el-button
                type="primary"
                @click="copyQuickLinkUrl"
                :icon="DocumentCopy"
              >
                复制
              </el-button>
            </template>
          </el-input>
        </div>
        <div class="quicklink-tips">
          <el-alert
            type="info"
            :closable="false"
            show-icon
          >
            <template #default>
              <div>提示：复制链接后，您可以分享给他人或在新标签页中打开</div>
            </template>
          </el-alert>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showQuickLinkDialog = false">关闭</el-button>
          <el-button
            type="primary"
            @click="copyQuickLinkUrl"
          >
            <el-icon><DocumentCopy /></el-icon>
            复制链接
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 快链列表弹窗 -->
    <el-dialog
      v-model="showQuickLinkListDialog"
      title="快链列表"
      width="800px"
      :close-on-click-modal="false"
      @opened="loadQuickLinkList"
    >
      <div class="quicklink-list-content">
        <div v-if="quickLinkListLoading" class="loading-container">
          <el-skeleton :rows="5" animated />
        </div>
        <div v-else-if="quickLinkList.length === 0" class="empty-container">
          <el-empty description="暂无快链" />
        </div>
        <div v-else class="quicklink-list">
          <el-table :data="quickLinkList" stripe>
            <el-table-column prop="name" label="快链名称" min-width="200" />
            <el-table-column prop="created_at" label="创建时间" width="180">
              <template #default="{ row }">
                {{ formatDate(row.created_at) }}
              </template>
            </el-table-column>
            <el-table-column label="操作" width="200" fixed="right">
              <template #default="{ row }">
                <el-button
                  type="primary"
                  size="small"
                  @click="openQuickLink(row.id)"
                >
                  打开
                </el-button>
                <el-button
                  type="danger"
                  size="small"
                  @click="deleteQuickLink(row.id)"
                >
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showQuickLinkListDialog = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>

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
import type { ComputedRef } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Promotion, RefreshLeft, View, DocumentCopy, Link, List } from '@element-plus/icons-vue'
import { ElIcon, ElTag, ElNotification, ElMessage, ElAlert, ElMessageBox, ElText, ElCheckbox } from 'element-plus'
import { eventBus, FormEvent, WorkspaceEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import WidgetComponent from '../widgets/WidgetComponent.vue'
import { Logger } from '@/core/utils/logger'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import { hasAnyRequiredRule } from '@/core/utils/validationUtils'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { useResponseDataStore } from '@/core/stores-v2/responseData'
import { useFunctionParamInitialization } from '../composables/useFunctionParamInitialization'
import { useFormParamURLSync } from '../composables/useFormParamURLSync'

const props = defineProps<{
  functionDetail?: FunctionDetail  // 🔥 改为可选，因为会在 onMounted 中主动获取
}>()

// 路由
const route = useRoute()
const router = useRouter()

// 依赖注入（使用 ServiceFactory 简化）
const stateManager = serviceFactory.getFormStateManager()
const domainService = serviceFactory.getFormDomainService()
const applicationService = serviceFactory.getFormApplicationService()
const workspaceStateManager = serviceFactory.getWorkspaceStateManager()  // 🔥 用于获取当前函数节点
const workspaceDomainService = serviceFactory.getWorkspaceDomainService()  // 🔥 用于获取函数详情

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

// 🔥 移除 formInitialData computed，改为使用统一的数据初始化框架
// URL 参数会在 useFunctionParamInitialization 中统一处理

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

// 快链相关
const showQuickLinkNameDialog = ref(false)
const showQuickLinkDialog = ref(false)
const showQuickLinkListDialog = ref(false)
const quickLinkUrl = ref('')
const quickLinkForm = ref({
  name: '',
  saveResponseParams: false  // 🔥 默认不保存响应参数
})
const quickLinkList = ref<any[]>([])
const quickLinkListLoading = ref(false)

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

// 复制快链 URL
const copyQuickLinkUrl = async (): Promise<void> => {
  try {
    await navigator.clipboard.writeText(quickLinkUrl.value)
    ElMessage.success('快链链接已复制到剪贴板')
  } catch (err) {
    ElMessage.error('复制失败，请手动复制：' + quickLinkUrl.value)
  }
}

// 🔥 Ctrl+S 快捷键监听
const handleKeydown = (event: KeyboardEvent): void => {
  // Ctrl+S 或 Cmd+S（Mac）
  if ((event.ctrlKey || event.metaKey) && event.key === 's') {
    event.preventDefault()
    handleSaveQuickLink()
  }
}

const handleSaveQuickLink = (): void => {
  if (!functionDetail.value) {
    ElNotification.error({
      title: '保存失败',
      message: '函数详情未加载完成，请稍后重试',
      duration: 3000
    })
    return
  }

  // 1. 收集所有字段的 FieldValue
  const requestParams: Record<string, FieldValue> = {}
  requestFields.value.forEach((field: FieldConfig) => {
    const fieldValue = formDataStore.getValue(field.code)
    if (fieldValue && fieldValue.raw !== null && fieldValue.raw !== undefined && fieldValue.raw !== '') {
      requestParams[field.code] = fieldValue
    }
  })

  // 如果没有数据，提示用户
  if (Object.keys(requestParams).length === 0) {
    ElMessage.warning('当前表单没有数据，无法保存快链')
    return
  }

  // 2. 显示名称输入弹窗
  quickLinkForm.value.name = `快链 ${new Date().toLocaleString('zh-CN')}`
  quickLinkForm.value.saveResponseParams = false  // 🔥 重置为默认值（不保存响应参数）
  showQuickLinkNameDialog.value = true
}

const confirmSaveQuickLink = async (): Promise<void> => {
  try {
    if (!functionDetail.value) {
      return
    }

    if (!quickLinkForm.value.name || quickLinkForm.value.name.trim() === '') {
      ElMessage.warning('请输入快链名称')
      return
    }

    // 1. 收集所有字段的 FieldValue（使用提取器递归提取嵌套数据）
    const { FieldExtractorRegistry } = await import('@/core/stores-v2/extractors/FieldExtractorRegistry')
    const extractorRegistry = new FieldExtractorRegistry()
    
    const requestParams: Record<string, FieldValue> = {}
    requestFields.value.forEach((field: FieldConfig) => {
      const fieldValue = formDataStore.getValue(field.code)
      if (!fieldValue) {
        return
      }
      
      // 🔥 对于 form 和 table 类型字段，使用提取器递归提取嵌套数据
      if (field.widget?.type === 'form' || field.widget?.type === 'table') {
        const extractedValue = extractorRegistry.extractField(field, field.code, (path: string) => {
          return formDataStore.getValue(path)
        })
        
        // 🔥 form 类型：如果提取后的对象为空（没有任何子字段），跳过该字段
        if (field.widget?.type === 'form') {
          if (!extractedValue || typeof extractedValue !== 'object' || Object.keys(extractedValue).length === 0) {
            return
          }
        }
        
        // 🔥 table 类型：如果提取后的数组为空，跳过该字段
        if (field.widget?.type === 'table') {
          if (Array.isArray(extractedValue) && extractedValue.length === 0) {
            return
          }
        }
        
        // 使用提取后的值（已递归提取嵌套数据）更新 FieldValue
        requestParams[field.code] = {
          ...fieldValue,
          raw: extractedValue
        }
      } else {
        // 其他类型字段：直接使用
        if (fieldValue.raw !== null && fieldValue.raw !== undefined && fieldValue.raw !== '') {
          requestParams[field.code] = fieldValue
        }
      }
    })

    // 2. 收集响应数据（如果用户勾选了保存响应参数）
    let responseParams: Record<string, any> | undefined = undefined
    if (quickLinkForm.value.saveResponseParams) {
      const state = stateManager.getState()
      if (state.response && Object.keys(state.response).length > 0) {
        responseParams = state.response
      }
    }

    // 3. 调用后端 API 保存快链
    const { createQuickLink } = await import('@/api/quicklink')
    const result = await createQuickLink({
      name: quickLinkForm.value.name.trim(),
      function_router: functionDetail.value.router,
      function_method: functionDetail.value.method,
      template_type: functionDetail.value.template_type || 'form',
      request_params: requestParams,
      response_params: responseParams
    })

    // 4. 生成快链 URL
    const url = `${window.location.origin}${route.path}?_quicklink_id=${result.id}`
    quickLinkUrl.value = url

    // 5. 关闭名称输入弹窗，显示快链地址弹窗
    showQuickLinkNameDialog.value = false
    showQuickLinkDialog.value = true

    // 6. 刷新快链列表（如果列表弹窗已打开）
    if (showQuickLinkListDialog.value) {
      loadQuickLinkList()
    }
  } catch (error: any) {
    let errorMessage = '保存快链失败，请稍后重试'
    if (error?.response?.data) {
      const responseData = error.response.data
      errorMessage = responseData.msg || errorMessage
    } else if (error?.message) {
      errorMessage = error.message
    }
    
    ElNotification.error({
      title: '保存失败',
      message: errorMessage,
      duration: 3000
    })
  }
}

// 加载快链列表
const loadQuickLinkList = async (): Promise<void> => {
  if (!functionDetail.value) {
    return
  }

  try {
    quickLinkListLoading.value = true
    const { listQuickLinks } = await import('@/api/quicklink')
    const result = await listQuickLinks({
      function_router: functionDetail.value.router,
      page: 1,
      page_size: 100
    })
    quickLinkList.value = result.list || []
  } catch (error: any) {
    ElNotification.error({
      title: '加载失败',
      message: error?.response?.data?.msg || error?.message || '加载快链列表失败',
      duration: 3000
    })
  } finally {
    quickLinkListLoading.value = false
  }
}

// 打开快链
const openQuickLink = (id: number): void => {
  showQuickLinkListDialog.value = false
  // 使用路由跳转，添加快链参数
  router.push({
    path: route.path,
    query: {
      ...route.query,
      _quicklink_id: String(id)
    }
  })
}

// 删除快链
const deleteQuickLink = async (id: number): Promise<void> => {
  try {
    await ElMessageBox.confirm('确定要删除这个快链吗？', '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })

    const { deleteQuickLink: deleteQuickLinkApi } = await import('@/api/quicklink')
    await deleteQuickLinkApi(id)
    
    ElMessage.success('删除成功')
    loadQuickLinkList()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElNotification.error({
        title: '删除失败',
        message: error?.response?.data?.msg || error?.message || '删除快链失败',
        duration: 3000
      })
    }
  }
}

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
const formDataStoreForURLSync = {
  getValue: (fieldCode: string) => formDataStore.getValue(fieldCode),
  getAllValues: () => {
    const allValues: Record<string, FieldValue> = {}
    const state = stateManager.getState()
    if (state.data) {
      state.data.forEach((value, key) => {
        allValues[key] = value
      })
    }
    return allValues
  }
}

const { watchFormData } = useFormParamURLSync({
  functionDetail: computed(() => functionDetail.value),
  formDataStore: formDataStoreForURLSync,
  enabled: true,
  debounceMs: 300
})

onMounted(async () => {
  // 🔥 添加 Ctrl+S 快捷键监听
  window.addEventListener('keydown', handleKeydown)
  
  // 🔥 挂载时清理 store，避免之前函数的数据污染
  formDataStore.clear()
  responseDataStore.clear()
  
  // 🔥 在 onMounted 中主动获取 functionDetail
  // 如果 prop 已经提供了 functionDetail，直接使用；否则从 WorkspaceStateManager 获取当前函数节点并加载详情
  if (props.functionDetail && props.functionDetail.id) {
    // 如果 prop 已经提供了 functionDetail，直接使用
    functionDetail.value = props.functionDetail
    console.log('🔍 [FormView] onMounted 时使用 prop 提供的 functionDetail', {
      functionId: props.functionDetail.id,
      requestFieldsCount: props.functionDetail.request?.length || 0
    })
  } else {
    // 否则，从 WorkspaceStateManager 获取当前函数节点并加载详情
    const currentFunction = workspaceStateManager.getCurrentFunction()
    if (currentFunction && currentFunction.type === 'function') {
      console.log('🔍 [FormView] onMounted 时主动加载 functionDetail', {
        functionNodeId: currentFunction.id,
        refId: currentFunction.ref_id,  // 🔥 记录 ref_id（函数 ID）
        functionPath: currentFunction.full_code_path,
        hasRefId: !!(currentFunction.ref_id && currentFunction.ref_id > 0)
      })
      try {
        // 🔥 loadFunction 会优先使用 ref_id 加载函数详情
        const detail = await workspaceDomainService.loadFunction(currentFunction)
        functionDetail.value = detail
        console.log('✅ [FormView] onMounted 时成功加载 functionDetail', {
          functionId: detail.id,
          refId: currentFunction.ref_id,  // 🔥 记录使用的 ref_id
          requestFieldsCount: detail.request?.length || 0,
          requestFields: detail.request?.map((f: any) => ({
            code: f.code,
            name: f.name,
            widgetType: f.widget?.type,
            hasDefault: !!(f.widget?.config as any)?.default,
            defaultValue: (f.widget?.config as any)?.default
          })) || []
        })
      } catch (error) {
        console.error('❌ [FormView] onMounted 时加载 functionDetail 失败', error)
        return
      }
    } else {
      console.log('🔍 [FormView] onMounted 时没有当前函数节点，等待 watch 触发', {
        hasCurrentFunction: !!currentFunction,
        functionType: currentFunction?.type
      })
      return
    }
  }
  
  // 🔥 初始化参数（此时 functionDetail 已经加载完成）
  if (functionDetail.value && functionDetail.value.id && functionDetail.value.request) {
    console.log('🔍 [FormView] onMounted 时初始化参数', {
      functionId: functionDetail.value.id,
      requestFieldsCount: functionDetail.value.request.length
    })
    const metadata = await initializeParams()
    
    // 初始化表单：在参数初始化完成后，初始化表单结构
    const fields = functionDetail.value.request || []
    if (fields.length > 0) {
      // 🔥 同步 formDataStore 的数据到 stateManager，确保 display 值不丢失
      syncFormDataStoreToStateManager(fields)
      
      // 🔥 调用 initializeForm 来触发 FormEvent.initialized 事件和更新字段配置
      // 🔥 注意：FormDomainService.initializeForm 已经优化，会优先保留已有的完整值（包含 display）
      const initialData = buildInitialDataFromFormDataStore(fields)
      console.log('🔍 [FormView] onMounted 时初始化表单', {
        fieldsCount: fields.length,
        initialDataKeys: Object.keys(initialData),
        initialData
      })
      applicationService.initializeForm(fields, initialData)
    }
    
    // 🔥 恢复响应数据（在表单初始化之后，避免被覆盖）
    if (metadata?.responseParams && stateManager && typeof (stateManager as any).setResponse === 'function') {
      (stateManager as any).setResponse(metadata.responseParams)
      console.log('🔍 [FormView] 已恢复响应数据', {
        responseParamsKeys: Object.keys(metadata.responseParams),
        responseParams: metadata.responseParams,
        stateResponse: stateManager.getState().response
      })
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
        const fields = (payload.detail.request || []) as FieldConfig[]
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
          console.log('🔍 [FormView] 已恢复响应数据', {
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
    // 🔥 同步到内部的 functionDetail ref
    if (newDetail && newDetail.id) {
      functionDetail.value = newDetail
    }
    
    // 🔥 检查 functionDetail 是否有效（必须要有 id 和 request 字段）
    if (!newDetail || !newDetail.id || !newDetail.request) {
      console.log('🔍 [FormView] props.functionDetail 无效或未加载完成，跳过初始化', {
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
      console.log('🔍 [FormView] props.functionDetail 变化（函数切换），开始重新初始化', {
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
          const initialData = buildInitialDataFromFormDataStore(fields)
          console.log('🔍 [FormView] 函数切换后初始化表单', {
            fieldsCount: fields.length,
            initialDataKeys: Object.keys(initialData),
            initialData
          })
          applicationService.initializeForm(fields, initialData)
          
          // 🔥 恢复响应数据（在表单初始化之后，避免被覆盖）
          if (metadata?.responseParams && stateManager && typeof (stateManager as any).setResponse === 'function') {
            (stateManager as any).setResponse(metadata.responseParams)
            console.log('🔍 [FormView] 已恢复响应数据', {
              responseParamsKeys: Object.keys(metadata.responseParams),
              responseParams: metadata.responseParams,
              stateResponse: stateManager.getState().response
            })
          }
        })
      }
    }
  }, { deep: false }) // 🔥 移除 immediate: true，避免与 onMounted 重复初始化

  // 🔥 移除 watch route.query，改为使用统一的数据初始化框架处理 URL 参数
  // URL 参数会在 initializeParams 时统一处理，包括类型转换和组件自治初始化

onUnmounted(() => {
  // 🔥 移除 Ctrl+S 快捷键监听
  window.removeEventListener('keydown', handleKeydown)
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

/* 快链弹窗样式 */
.quicklink-dialog-content {
  padding: 10px 0;
}

.quicklink-info {
  margin-bottom: 20px;
}

.quicklink-info p {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 14px;
}

.quicklink-url-section {
  margin-bottom: 20px;
}

.quicklink-url-input {
  width: 100%;
}

.quicklink-tips {
  margin-top: 20px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

/* 快链名称输入弹窗样式 */
.quicklink-name-dialog-content {
  padding: 10px 0;
}

.form-item-hint {
  margin-top: 8px;
  line-height: 1.5;
}

/* 快链列表弹窗样式 */
.quicklink-list-content {
  min-height: 200px;
}

.loading-container {
  padding: 20px;
}

.empty-container {
  padding: 40px 0;
}
</style>

