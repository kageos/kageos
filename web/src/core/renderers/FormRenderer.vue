<template>
  <div class="form-renderer">
    <!-- 请求参数表单 -->
    <el-form
      v-if="fields.length > 0"
      ref="formRef"
      :model="formData"
      label-width="100px"
      class="function-form"
    >
      <div class="section-title">请求参数</div>
      <el-form-item
        v-for="field in fields"
        :key="field.code"
        :label="field.name"
        :prop="field.code"
      >
        <component :is="renderField(field)" />
      </el-form-item>
    </el-form>

    <!-- 提交按钮区域 - 将请求参数和响应参数分开 -->
    <div v-if="showSubmitButton || showResetButton" class="form-actions-section">
      <div class="form-actions-row">
        <el-button v-if="showSubmitButton" type="primary" size="large" @click="handleRealSubmit" :loading="submitting" class="submit-button-full-width">
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
        <el-tag v-if="!responseData" type="info" size="small" style="margin-left: 12px">等待提交</el-tag>
        <el-tag v-else type="success" size="small" style="margin-left: 12px">已返回</el-tag>
      </div>
      <el-form
        label-width="100px"
        class="function-form response-container"
        :class="{ 'is-empty': !responseData }"
      >
        <el-form-item
          v-for="field in responseFields"
          :key="field.code"
          :label="field.name"
        >
          <component :is="renderResponseField(field)" />
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

    <!-- 分享信息 -->
    <el-card v-if="shareInfo" class="share-card" style="margin-top: 20px;">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>分享信息</span>
          <el-button text @click="shareInfo = null">关闭</el-button>
        </div>
      </template>
      <div class="share-content">
        <h4>快照ID：</h4>
        <el-input v-model="shareInfo.viewId" readonly>
          <template #append>
            <el-button @click="handleCopyViewId">复制</el-button>
          </template>
        </el-input>
        
        <h4 style="margin-top: 20px;">分享链接：</h4>
        <el-input v-model="shareInfo.shareUrl" readonly>
          <template #append>
            <el-button @click="handleCopyShareUrl">复制</el-button>
          </template>
        </el-input>
        
        <h4 style="margin-top: 20px;">快照数据：</h4>
        <pre>{{ shareInfo.snapshot }}</pre>
      </div>
    </el-card>

    <!-- 调试信息 -->
    <el-card v-if="showDebug" class="debug-card" style="margin-top: 20px;">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>调试信息</span>
          <el-button text @click="showDebug = false">关闭</el-button>
        </div>
      </template>
      <pre>{{ debugInfo }}</pre>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, h } from 'vue'
import { ElForm, ElFormItem, ElButton, ElCard, ElMessage, ElInput, ElIcon, ElDivider, ElTag } from 'element-plus'
import { Promotion, RefreshLeft } from '@element-plus/icons-vue'
import type { FieldConfig, FunctionDetail, FieldValue } from '../types/field'
import type { WidgetRenderProps, WidgetSnapshot } from '../types/widget'
import { ReactiveFormDataManager } from '../managers/ReactiveFormDataManager'
import { widgetFactory } from '../factories/WidgetFactory'
import { BaseWidget } from '../widgets/BaseWidget'
import { ResponseTableWidget } from '../widgets/ResponseTableWidget'
import { ResponseFormWidget } from '../widgets/ResponseFormWidget'
import { executeFunction } from '@/api/function'

const props = withDefaults(defineProps<{
  functionDetail: FunctionDetail
  showSubmitButton?: boolean
  showShareButton?: boolean
  showResetButton?: boolean
  showDebugButton?: boolean
}>(), {
  showSubmitButton: true,
  showShareButton: true,
  showResetButton: true,
  showDebugButton: true
})

// 表单引用
const formRef = ref()

// 请求字段列表（根据 table_permission 过滤）
const fields = computed(() => {
  const allFields = props.functionDetail?.request || []
  
  // 🔥 根据 table_permission 过滤字段（默认为"新增"模式）
  return allFields.filter(field => {
    const permission = field.table_permission
    
    // ✅ 显示：空、create
    // ❌ 不显示：read（后端自动生成）、update（仅编辑时可修改）
    return !permission || permission === '' || permission === 'create'
  })
})

// 返回值字段列表
const responseFields = computed(() => props.functionDetail?.response || [])

// 返回值数据
const responseData = ref<any>(null)

// FormDataManager
const formManager = new ReactiveFormDataManager()

// Widget 缓存（field_path -> Widget 实例）
const allWidgets = new Map<string, BaseWidget>()

// 表单数据（用于 el-form 绑定）
const formData = reactive<Record<string, any>>({})

// 调试信息
const showDebug = ref(false)
const debugInfo = ref('')

// 提交结果
const submitResult = ref<any>(null)

// 分享信息
const shareInfo = ref<any>(null)

// 提交状态
const submitting = ref(false)

/**
 * 初始化表单
 */
function initializeForm(): void {
  console.log('[FormRenderer] 初始化表单', props.functionDetail)
  
  // 初始化所有字段
  fields.value.forEach(field => {
    const fieldPath = field.code
    
    // 🔥 使用 Widget 的静态方法获取默认值（面向对象）
    const WidgetClass = widgetFactory.getWidgetClass(field.widget.type)
    const defaultValue = WidgetClass.getDefaultValue(field)
    
    // 初始化 FormDataManager
    formManager.initializeField(fieldPath, defaultValue)
    
    // 初始化 formData（用于 el-form）
    formData[field.code] = defaultValue.raw
  })
}

/**
 * 注册 Widget
 */
function registerWidget(fieldPath: string, widget: BaseWidget): void {
  allWidgets.set(fieldPath, widget)
  console.log(`[FormRenderer] 注册 Widget: ${fieldPath}`)
}

/**
 * 注销 Widget
 */
function unregisterWidget(fieldPath: string): void {
  allWidgets.delete(fieldPath)
  console.log(`[FormRenderer] 注销 Widget: ${fieldPath}`)
}

/**
 * 渲染单个字段
 */
function renderField(field: FieldConfig): any {
  const fieldPath = field.code
  
  // 检查是否已缓存
  let widget = allWidgets.get(fieldPath)
  
  if (!widget) {
    // 创建 Widget
    const WidgetClass = widgetFactory.getWidgetClass(field.widget.type)
    
    // 🔥 捕获 functionDetail，避免闭包访问 props 的问题
    const functionDetail = props.functionDetail
    
    const widgetProps: WidgetRenderProps = {
      field: field,
      currentFieldPath: fieldPath,
      value: formManager.getValue(fieldPath),
      onChange: (newValue: FieldValue) => {
        formManager.setValue(fieldPath, newValue)
        // 同步到 formData
        formData[field.code] = newValue.raw
      },
      formManager: formManager,
      formRenderer: {
        registerWidget,
        unregisterWidget,
        getFunctionMethod: () => functionDetail.method,
        getFunctionRouter: () => functionDetail.router,
        getSubmitData: () => prepareSubmitDataWithTypeConversion()  // 🔥 提供统一的数据收集方法
      },
      depth: 0
    }
    
    widget = new WidgetClass(widgetProps)  // 🔥 使用 widgetProps 而不是 props
    registerWidget(fieldPath, widget)
  }
  
  return widget.render()
}

/**
 * 渲染单个返回值字段（只读展示）
 * 即使没有数据也渲染框架结构，提供更好的用户体验
 */
function renderResponseField(field: FieldConfig): any {
  // 获取返回值（可能为 undefined）
  const value = responseData.value?.[field.code]
  
  // 根据字段类型渲染不同的组件
  const widgetType = field.widget?.type || 'input'
  
  // 对于表格类型，使用 ResponseTableWidget（始终渲染，即使没有数据也显示空表格）
  if (widgetType === 'table' || field.data?.type?.includes('[]')) {
    const widget = new ResponseTableWidget({
      field: field,
      currentFieldPath: field.code,
      value: {
        raw: value || [],  // 没有数据时使用空数组
        display: Array.isArray(value) ? `共${value.length}条` : '等待数据...',
        meta: {}
      },
      onChange: () => {}, // 返回值是只读的，不需要 onChange
      formManager: formManager,
      formRenderer: {
        registerWidget: () => {},
        unregisterWidget: () => {},
        getFunctionMethod: () => props.functionDetail.method,
        getFunctionRouter: () => props.functionDetail.router,
        getSubmitData: () => ({})
      },
      depth: 0
    })
    return widget.render()
  }
  
  // 对于对象类型，使用 ResponseFormWidget（始终渲染，即使没有数据也显示空表单框架）
  if (widgetType === 'form' || field.data?.type === 'struct') {
    const widget = new ResponseFormWidget({
      field: field,
      currentFieldPath: field.code,
      value: {
        raw: value || {},  // 没有数据时使用空对象
        display: value ? JSON.stringify(value) : '等待数据...',
        meta: {}
      },
      onChange: () => {}, // 返回值是只读的，不需要 onChange
      formManager: formManager,
      formRenderer: {
        registerWidget: () => {},
        unregisterWidget: () => {},
        getFunctionMethod: () => props.functionDetail.method,
        getFunctionRouter: () => props.functionDetail.router,
        getSubmitData: () => ({})
      },
      depth: 0
    })
    return widget.render()
  }
  
  // 对于文本域
  if (widgetType === 'text_area' || widgetType === 'textarea') {
    return h(ElInput, {
      modelValue: value || '',
      type: 'textarea',
      rows: 4,
      disabled: true,
      placeholder: responseData.value ? '' : `等待提交后显示${field.name}`,
      style: { width: '100%' }
    })
  }
  
  // 默认使用只读输入框
  return h(ElInput, {
    modelValue: value !== undefined && value !== null ? String(value) : '',
    disabled: true,
    placeholder: responseData.value ? '' : `等待提交后显示${field.name}`,
    style: { width: '100%' }
  })
}

/**
 * 预览提交数据（调试用）
 */
function handlePreviewSubmit(): void {
  console.log('[FormRenderer] 预览提交数据')
  
  // 🔥 使用统一的数据收集方法（递归收集所有字段）
  const submitData = prepareSubmitDataWithTypeConversion()
  
  // 显示提交结果
  submitResult.value = JSON.stringify(submitData, null, 2)
  
  ElMessage.info({
    message: '预览提交数据成功！查看下方调试信息',
    duration: 3000
  })
  
  console.log('[FormRenderer] 提交数据:', submitData)
}

/**
 * 准备提交数据（使用 Widget 的转换逻辑）
 */
/**
 * 🔥 准备提交数据（方案 4：统一使用 widget.getRawValueForSubmit()）
 * 
 * 核心思想：
 * 1. 基础组件（Input/Select/...）：直接返回 raw 值
 * 2. 容器组件（List/Struct）：递归调用子组件的 getRawValueForSubmit()
 * 3. FormRenderer 只需遍历顶层字段，递归由各组件自己处理
 */
function prepareSubmitDataWithTypeConversion(): Record<string, any> {
  const result: Record<string, any> = {}
  
  console.log('[FormRenderer] 🚀 开始收集提交数据（方案4-递归）')
  
  // 🔥 统一处理：无论基础类型还是嵌套类型，都调用 getRawValueForSubmit()
  fields.value.forEach(field => {
    const fieldPath = field.code
    const widget = allWidgets.get(fieldPath)
    
    if (widget) {
      result[fieldPath] = widget.getRawValueForSubmit()
      console.log(`[FormRenderer]   ✅ ${fieldPath}:`, result[fieldPath])
    } else {
      console.warn(`[FormRenderer]   ⚠️ ${fieldPath}: Widget 未注册`)
    }
  })
  
  console.log('[FormRenderer] ✅ 收集完成，最终数据:', result)
  return result
}

/**
 * 真正提交表单到后端
 */
async function handleRealSubmit(): Promise<void> {
  console.log('[FormRenderer] 提交表单到后端')
  
  submitting.value = true
  
  try {
    // 使用带类型转换的数据准备方法
    const submitData = prepareSubmitDataWithTypeConversion()
    console.log('[FormRenderer] 提交数据:', submitData)
    
    // 调用后端 API
    const response = await executeFunction(
      props.functionDetail.method,
      props.functionDetail.router,
      submitData
    )
    
    console.log('[FormRenderer] 后端响应:', response)
    
    // 保存返回值
    // 后端返回格式：{ code: 0, data: {...}, msg: "成功" }
    // response 已经由 request 拦截器处理，直接就是 data 字段的内容
    if (response && typeof response === 'object') {
      // 如果返回的数据有 data 字段，使用 data 字段；否则直接使用整个响应
      responseData.value = response.data !== undefined ? response.data : response
    } else {
      // 如果返回的不是对象，包装一下
      responseData.value = { result: response }
    }
    
    
    ElMessage.success({
      message: '表单提交成功！',
      duration: 3000
    })
    
  } catch (error: any) {
    console.error('[FormRenderer] 提交失败:', error)
    
    // 提取错误信息
    const errorMessage = error?.response?.data?.msg || 
                       error?.response?.data?.message || 
                       error?.message || 
                       '提交失败'
    
    ElMessage.error({
      message: errorMessage,
      duration: 5000
    })
    
    // 清空返回值（如果有之前的错误数据）
    responseData.value = null
  } finally {
    submitting.value = false
  }
}

/**
 * 重置表单
 */
function handleReset(): void {
  formRef.value?.resetFields()
  formManager.clear()
  initializeForm()
  
  // 清空结果和分享信息
  submitResult.value = null
  shareInfo.value = null
  
  ElMessage.info('表单已重置')
}

/**
 * 分享表单（生成快照）
 */
function handleShare(): void {
  console.log('[FormRenderer] 生成分享快照')
  
  const snapshots: WidgetSnapshot[] = []
  
  // 捕获所有 Widget 的快照
  for (const [fieldPath, widget] of allWidgets) {
    const snapshot = widget.captureSnapshot()
    snapshots.push(snapshot)
  }
  
  // 生成快照ID（实际项目中应该调用后端API）
  const viewId = `test_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  
  // 生成分享链接
  const shareUrl = `${window.location.origin}/test/form-renderer?view_id=${viewId}`
  
  // 显示分享信息
  shareInfo.value = {
    viewId,
    shareUrl,
    snapshot: JSON.stringify({
      view_id: viewId,
      function_code: props.functionDetail.code,
      widget_snapshots: snapshots,
      metadata: {
        created_at: new Date().toISOString(),
        title: props.functionDetail.name
      }
    }, null, 2)
  }
  
  ElMessage.success({
    message: '快照生成成功！查看下方分享信息',
    duration: 3000
  })
  
  console.log('[FormRenderer] 快照数据:', snapshots)
}

/**
 * 复制 ViewID
 */
function handleCopyViewId(): void {
  navigator.clipboard.writeText(shareInfo.value.viewId)
  ElMessage.success('ViewID 已复制到剪贴板')
}

/**
 * 复制分享链接
 */
function handleCopyShareUrl(): void {
  navigator.clipboard.writeText(shareInfo.value.shareUrl)
  ElMessage.success('分享链接已复制到剪贴板')
}

/**
 * 调试输出
 */
function handleDebug(): void {
  showDebug.value = !showDebug.value
  
  debugInfo.value = JSON.stringify({
    functionDetail: props.functionDetail,
    fields: fields.value,
    allFieldPaths: formManager.getAllFieldPaths(),
    submitData: prepareSubmitDataWithTypeConversion(),  // 🔥 使用统一的数据收集方法
    registeredWidgets: Array.from(allWidgets.keys()),
    registeredWidgetTypes: widgetFactory.getRegisteredTypes()
  }, null, 2)
}

// 初始化
initializeForm()

/**
 * 暴露方法给父组件（如 FormDialog）
 */
defineExpose({
  prepareSubmitDataWithTypeConversion,
  formManager,
  allWidgets,
  handleRealSubmit
})
</script>

<style scoped>
.form-renderer {
  padding: 20px;
  width: 100%;
  max-width: 100%;
}

.request-card {
  margin-bottom: 20px;
  width: 100%;
}

/* 🔥 确保卡片内容占满宽度 */
.request-card :deep(.el-card__body) {
  width: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.form-container {
  max-width: 100%;
}

/* 🔥 强制内容区域占满剩余空间 */
.form-container :deep(.el-form-item__content) {
  flex: 1 !important;
  max-width: 100% !important;
  width: 100% !important;
}

/* 🔥 确保表单项使用 flex 布局 */
.form-container :deep(.el-form-item) {
  display: flex !important;
}

/* 🔥 确保所有输入控件占满宽度 */
.form-container :deep(.el-input),
.form-container :deep(.el-select),
.form-container :deep(.el-textarea),
.form-container :deep(.el-date-picker) {
  width: 100% !important;
}

/* 🔥 确保 FormWidget 占满宽度 */
.form-container :deep(.form-widget) {
  width: 100% !important;
}

.form-container :deep(.form-widget .el-card) {
  width: 100% !important;
}

.form-container :deep(.form-widget .el-card__body) {
  width: 100% !important;
}

.form-container :deep(.form-widget .el-form) {
  width: 100% !important;
}

.form-container :deep(.form-widget .el-form-item) {
  display: flex !important;
  width: 100% !important;
  margin-bottom: 18px !important;  /* 🔥 确保表单项之间有合适的间距 */
}

.form-container :deep(.form-widget .el-form-item__content) {
  flex: 1 !important;
  width: 100% !important;
  max-width: 100% !important;
}

.form-container :deep(.form-widget .el-input),
.form-container :deep(.form-widget .el-select),
.form-container :deep(.form-widget .el-textarea),
.form-container :deep(.form-widget .el-date-picker) {
  width: 100% !important;
}

/* 章节标题样式 - 照抄旧版本 */
.section-title {
  font-size: 16px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin: 24px 0 16px;
  padding-left: 12px;
  border-left: 3px solid var(--el-color-primary);
  display: flex;
  align-items: center;
}

/* 表单样式 - 照抄旧版本 */
.function-form {
  :deep(.el-form-item) {
    margin-bottom: 20px;

    .el-form-item__label {
      font-weight: 500;
      color: var(--el-text-color-primary);
      padding-bottom: 8px;
    }
  }
}

/* 提交按钮区域 - 照抄旧版本 */
.form-actions-section {
  margin: 32px 0;
  padding: 0;
}

.form-actions-row {
  display: flex;
  gap: 12px;
  width: 100%;
  margin-bottom: 0;
  
  .el-button {
    &.el-button--large {
      height: 40px;
      font-size: 16px;
      font-weight: 500;
    }
  }
  
  .submit-button-full-width {
    flex: 1;
    width: 100%;
  }
}

.response-container {
  max-width: 100%;
}

/* 🔥 强制返回值内容区域占满剩余空间 */
.response-container :deep(.el-form-item__content) {
  flex: 1 !important;
  max-width: 100% !important;
  width: 100% !important;
}

/* 🔥 确保返回值表单项使用 flex 布局 */
.response-container :deep(.el-form-item) {
  display: flex !important;
}

/* 🔥 确保返回值所有输入控件占满宽度 */
.response-container :deep(.el-input),
.response-container :deep(.el-select),
.response-container :deep(.el-textarea),
.response-container :deep(.el-date-picker) {
  width: 100% !important;
}

/* 🔥 确保返回值的卡片和表格组件占满宽度 */
.response-container :deep(.el-card),
.response-container :deep(.el-table) {
  width: 100% !important;
}

/* 🔥 确保返回值的表单组件占满宽度 */
.response-container :deep(.el-form) {
  width: 100% !important;
}

.response-container.is-empty {
  opacity: 0.6;
}

/* 调试卡片 */
.result-card,
.share-card,
.debug-card {
  margin-top: 20px;
  max-width: 100%;
}

.result-card pre,
.share-card pre,
.debug-card pre {
  max-height: 400px;
  overflow: auto;
  font-size: 12px;
  background: #f5f7fa;
  padding: 12px;
  border-radius: 4px;
  margin: 0;
}

.result-content h4,
.share-content h4 {
  margin: 0 0 10px 0;
  color: #606266;
  font-size: 14px;
  font-weight: 600;
}

.share-content {
  padding: 10px 0;
}
</style>

