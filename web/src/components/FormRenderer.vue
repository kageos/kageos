<template>
  <div class="form-renderer">
    <el-form
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="120px"
      label-position="right"
      @submit.prevent="handleSubmit"
    >
      <el-form-item
        v-for="field in formFields"
        :key="field.code"
        :label="field.name"
        :prop="field.code"
        :required="isRequired(field)"
      >
        <!-- Input 输入框 -->
        <el-input
          v-if="field.widget.type === 'input'"
          v-model="formData[field.code]"
          :placeholder="field.widget.config.placeholder || `请输入${field.name}`"
          :disabled="field.widget.config.disabled"
          :type="field.widget.config.password ? 'password' : 'text'"
          :maxlength="getMaxLength(field)"
          show-word-limit
          clearable
        >
          <template v-if="field.widget.config.prepend" #prepend>
            {{ field.widget.config.prepend }}
          </template>
          <template v-if="field.widget.config.append" #append>
            {{ field.widget.config.append }}
          </template>
        </el-input>

        <!-- Number 数字输入框 -->
        <el-input-number
          v-else-if="field.widget.type === 'number'"
          v-model="formData[field.code]"
          :placeholder="field.widget.config.placeholder || `请输入${field.name}`"
          :disabled="field.widget.config.disabled"
          :min="getMinValue(field)"
          :max="getMaxValue(field)"
          :step="field.widget.config.step || 1"
          :precision="field.widget.config.precision"
          style="width: 100%"
        />

        <!-- TextArea 文本域 -->
        <el-input
          v-else-if="field.widget.type === 'text_area'"
          v-model="formData[field.code]"
          type="textarea"
          :placeholder="field.widget.config.placeholder || `请输入${field.name}`"
          :disabled="field.widget.config.disabled"
          :rows="field.widget.config.rows || 4"
          :maxlength="getMaxLength(field)"
          show-word-limit
        />

        <!-- Select 下拉选择 -->
        <el-select
          v-else-if="field.widget.type === 'select'"
          v-model="formData[field.code]"
          :placeholder="field.widget.config.placeholder || `请选择${field.name}`"
          :disabled="field.widget.config.disabled"
          :multiple="field.widget.config.multiple"
          :clearable="!isRequired(field)"
          style="width: 100%"
        >
          <el-option
            v-for="option in field.widget.config.options"
            :key="option"
            :label="option"
            :value="option"
          />
        </el-select>

        <!-- Timestamp 时间选择器 -->
        <el-date-picker
          v-else-if="field.widget.type === 'timestamp'"
          v-model="formData[field.code]"
          type="datetime"
          :placeholder="field.widget.config.placeholder || `请选择${field.name}`"
          :disabled="field.widget.config.disabled"
          :format="field.widget.config.format || 'YYYY-MM-DD HH:mm:ss'"
          value-format="x"
          style="width: 100%"
        />

        <!-- Switch 开关 -->
        <el-switch
          v-else-if="field.widget.type === 'switch'"
          v-model="formData[field.code]"
          :disabled="field.widget.config.disabled"
        />

        <!-- Checkbox 多选框 -->
        <el-checkbox-group
          v-else-if="field.widget.type === 'checkbox'"
          v-model="formData[field.code]"
          :disabled="field.widget.config.disabled"
        >
          <el-checkbox
            v-for="option in field.widget.config.options"
            :key="option"
            :label="option"
          />
        </el-checkbox-group>

        <!-- Radio 单选框 -->
        <el-radio-group
          v-else-if="field.widget.type === 'radio'"
          v-model="formData[field.code]"
          :disabled="field.widget.config.disabled"
        >
          <el-radio
            v-for="option in field.widget.config.options"
            :key="option"
            :label="option"
          />
        </el-radio-group>

        <!-- 其他未支持的类型 -->
        <el-input
          v-else
          v-model="formData[field.code]"
          :placeholder="`请输入${field.name}`"
        />

        <!-- 字段描述 -->
        <div v-if="field.desc" class="field-desc">
          <el-icon><InfoFilled /></el-icon>
          {{ field.desc }}
        </div>
      </el-form-item>

      <!-- 提交按钮 -->
      <el-form-item>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          <el-icon><Select /></el-icon>
          提交
        </el-button>
        <el-button @click="handleReset">
          <el-icon><RefreshLeft /></el-icon>
          重置
        </el-button>
      </el-form-item>
    </el-form>

    <!-- 结果展示区域 -->
    <div v-if="showResult && resultData" class="result-section">
      <el-divider content-position="left">
        <el-icon><Document /></el-icon>
        执行结果
      </el-divider>
      
      <el-descriptions :column="1" border>
        <el-descriptions-item
          v-for="field in resultFields"
          :key="field.code"
          :label="field.name"
        >
          <template v-if="field.widget.type === 'text' || field.widget.type === 'text_area'">
            <div class="result-text" style="white-space: pre-wrap;">{{ resultData[field.code] }}</div>
          </template>
          <template v-else-if="field.widget.type === 'timestamp'">
            {{ formatTimestamp(resultData[field.code], field.widget.config.format) }}
          </template>
          <template v-else>
            {{ resultData[field.code] }}
          </template>
        </el-descriptions-item>
      </el-descriptions>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { InfoFilled, Select, RefreshLeft, Document } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { executeFunction } from '@/api/function'
import type { FieldConfig } from '@/types'

interface Props {
  fields: FieldConfig[]  // 表单字段（来自 request）
  responseFields?: FieldConfig[]  // 结果字段（来自 response）
  method: string  // HTTP 方法
  router: string  // 路由
  mode?: 'form' | 'dialog'  // 模式：独立表单 或 弹窗表单
  initialData?: Record<string, any>  // 初始数据（编辑模式）
}

const props = withDefaults(defineProps<Props>(), {
  mode: 'form',
  initialData: () => ({})
})

const emit = defineEmits<{
  submit: [data: any]
  success: [data: any]
  error: [error: any]
}>()

// 表单引用
const formRef = ref<FormInstance>()

// 表单数据
const formData = ref<Record<string, any>>({})

// 提交状态
const submitting = ref(false)

// 结果数据
const resultData = ref<Record<string, any> | null>(null)
const showResult = ref(false)

// 表单字段
const formFields = computed(() => props.fields)

// 结果字段
const resultFields = computed(() => props.responseFields || [])

// 初始化表单数据
const initFormData = () => {
  const data: Record<string, any> = {}
  
  props.fields.forEach(field => {
    // 如果有初始数据，使用初始数据
    if (props.initialData && field.code in props.initialData) {
      data[field.code] = props.initialData[field.code]
    }
    // 否则使用默认值
    else if (field.widget.config.default !== undefined && field.widget.config.default !== '') {
      data[field.code] = field.widget.config.default
    }
    // 根据类型设置默认值
    else {
      switch (field.data.type) {
        case 'int':
        case 'float':
        case 'number':
          data[field.code] = field.widget.config.default || undefined
          break
        case 'bool':
          data[field.code] = false
          break
        case 'array':
          data[field.code] = []
          break
        default:
          data[field.code] = ''
      }
    }
  })
  
  formData.value = data
}

// 解析验证规则
const parseValidationRules = (field: FieldConfig) => {
  const rules: any[] = []
  
  if (!field.validation) return rules
  
  // 判断字段是否为数字类型
  const isNumberType = ['int', 'float', 'number'].includes(field.data.type)
  
  // 特殊处理 oneof，因为它的值中可能包含逗号
  let validationStr = field.validation
  let oneofOptions: string[] = []
  
  // 先提取 oneof 部分
  const oneofMatch = validationStr.match(/oneof=([^,]+(?:,[^,]+)*)/)
  if (oneofMatch) {
    // 提取 oneof 的所有选项
    const oneofPart = oneofMatch[0] // 例如 "oneof=低,中,高"
    oneofOptions = oneofMatch[1].split(',').map(v => v.trim()) // ["低", "中", "高"]
    // 从原字符串中移除 oneof 部分，避免被后续 split 分割
    validationStr = validationStr.replace(oneofPart, '')
  }
  
  // 提取 min 和 max 值
  let minValue: number | undefined
  let maxValue: number | undefined
  
  const minMatch = validationStr.match(/min=(\d+)/)
  const maxMatch = validationStr.match(/max=(\d+)/)
  
  if (minMatch) {
    minValue = parseInt(minMatch[1])
    validationStr = validationStr.replace(minMatch[0], '')
  }
  
  if (maxMatch) {
    maxValue = parseInt(maxMatch[1])
    validationStr = validationStr.replace(maxMatch[0], '')
  }
  
  // 处理其他验证规则
  const validations = validationStr.split(',').map(v => v.trim()).filter(v => v)
  
  validations.forEach(validation => {
    if (validation === 'required') {
      rules.push({
        required: true,
        message: `请输入${field.name}`,
        trigger: ['blur', 'change']
      })
    }
  })
  
  // 处理 min/max 验证（根据字段类型区分）
  if (minValue !== undefined || maxValue !== undefined) {
    if (isNumberType) {
      // 数字类型：验证数值大小
      rules.push({
        validator: (rule: any, value: any, callback: any) => {
          if (value === undefined || value === null || value === '') {
            callback()
            return
          }
          
          const numValue = Number(value)
          
          if (minValue !== undefined && numValue < minValue) {
            callback(new Error(`${field.name}不能小于${minValue}`))
            return
          }
          
          if (maxValue !== undefined && numValue > maxValue) {
            callback(new Error(`${field.name}不能大于${maxValue}`))
            return
          }
          
          callback()
        },
        trigger: 'blur'
      })
    } else {
      // 字符串类型：验证字符串长度
      if (minValue !== undefined) {
        rules.push({
          min: minValue,
          message: `${field.name}最少${minValue}个字符`,
          trigger: 'blur'
        })
      }
      
      if (maxValue !== undefined) {
        rules.push({
          max: maxValue,
          message: `${field.name}最多${maxValue}个字符`,
          trigger: 'blur'
        })
      }
    }
  }
  
  // 处理 oneof 验证
  if (oneofOptions.length > 0) {
    rules.push({
      validator: (rule: any, value: any, callback: any) => {
        if (value && !oneofOptions.includes(value)) {
          callback(new Error(`${field.name}必须是以下值之一: ${oneofOptions.join(', ')}`))
        } else {
          callback()
        }
      },
      trigger: 'change'
    })
  }
  
  return rules
}

// 表单验证规则
const formRules = computed<FormRules>(() => {
  const rules: FormRules = {}
  
  props.fields.forEach(field => {
    const fieldRules = parseValidationRules(field)
    if (fieldRules.length > 0) {
      rules[field.code] = fieldRules
    }
  })
  
  return rules
})

// 判断字段是否必填
const isRequired = (field: FieldConfig) => {
  return field.validation?.includes('required') || false
}

// 获取最大长度
const getMaxLength = (field: FieldConfig) => {
  const match = field.validation?.match(/max=(\d+)/)
  return match ? parseInt(match[1]) : undefined
}

// 获取最小值
const getMinValue = (field: FieldConfig) => {
  const match = field.validation?.match(/min=(\d+)/)
  return match ? parseInt(match[1]) : undefined
}

// 获取最大值
const getMaxValue = (field: FieldConfig) => {
  const match = field.validation?.match(/max=(\d+)/)
  return match ? parseInt(match[1]) : undefined
}

// 格式化时间戳
const formatTimestamp = (timestamp: number, format = 'YYYY-MM-DD HH:mm:ss') => {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  
  if (format.includes('HH:mm:ss')) {
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
  }
  return `${year}-${month}-${day}`
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return
  
  try {
    // 验证表单
    await formRef.value.validate()
    
    submitting.value = true
    
    console.log('[FormRenderer] 提交表单')
    console.log('[FormRenderer]   Method:', props.method)
    console.log('[FormRenderer]   Router:', props.router)
    console.log('[FormRenderer]   Data:', formData.value)
    
    // 调用执行函数接口
    const response = await executeFunction(props.method, props.router, formData.value)
    
    console.log('[FormRenderer] 提交成功:', response)
    
    // 如果有 response 字段，显示结果
    if (props.responseFields && props.responseFields.length > 0) {
      resultData.value = response
      showResult.value = true
    }
    
    ElMessage.success('提交成功')
    emit('success', response)
    
  } catch (error: any) {
    console.error('[FormRenderer] 提交失败:', error)
    ElMessage.error(error.message || '提交失败')
    emit('error', error)
  } finally {
    submitting.value = false
  }
}

// 重置表单
const handleReset = () => {
  formRef.value?.resetFields()
  resultData.value = null
  showResult.value = false
}

// 监听字段变化，重新初始化表单
watch(() => props.fields, () => {
  initFormData()
}, { immediate: true, deep: true })

// 监听初始数据变化
watch(() => props.initialData, () => {
  initFormData()
}, { deep: true })

// 暴露方法给父组件
defineExpose({
  formRef,
  formData,
  validate: () => formRef.value?.validate(),
  resetFields: () => formRef.value?.resetFields(),
  handleSubmit,
  handleReset
})
</script>

<style scoped>
.form-renderer {
  padding: 20px;
  background: var(--el-bg-color);
}

.field-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.result-section {
  margin-top: 32px;
  padding-top: 24px;
}

.result-text {
  line-height: 1.6;
}

/* 弹窗模式下的样式调整 */
:deep(.el-form--label-top) {
  .el-form-item__label {
    padding-bottom: 8px;
  }
}
</style>

