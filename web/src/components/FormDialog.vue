<template>
  <el-dialog
    v-model="dialogVisible"
    :title="title"
    :width="width"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="120px"
      label-position="right"
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
    </el-form>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          确定
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { InfoFilled } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import type { FieldConfig } from '@/types'

interface Props {
  modelValue: boolean  // 对话框显示状态
  title: string  // 对话框标题
  fields: FieldConfig[]  // 表单字段
  mode: 'create' | 'update'  // 模式：新增或编辑
  initialData?: Record<string, any>  // 初始数据（编辑模式）
  width?: string | number  // 对话框宽度
}

const props = withDefaults(defineProps<Props>(), {
  width: '600px',
  initialData: () => ({})
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  submit: [data: Record<string, any>]
  close: []
}>()

// 对话框显示状态
const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// 表单引用
const formRef = ref<FormInstance>()

// 表单数据
const formData = ref<Record<string, any>>({})

// 提交状态
const submitting = ref(false)

// 根据权限过滤字段
const formFields = computed(() => {
  return props.fields.filter(field => {
    const permission = field.table_permission
    
    // 新增模式
    if (props.mode === 'create') {
      // read: 不显示（后端自动生成）
      // update: 不显示（只能编辑时修改）
      // create: 显示（只能新增时填写）
      // 空: 显示（全部权限）
      return !permission || permission === '' || permission === 'create'
    }
    
    // 编辑模式
    if (props.mode === 'update') {
      // read: 不显示（只读）
      // update: 显示（只能编辑时修改）
      // create: 不显示（只能新增时填写）
      // 空: 显示（全部权限）
      return !permission || permission === '' || permission === 'update'
    }
    
    return true
  })
})

// 初始化表单数据
const initFormData = () => {
  const data: Record<string, any> = {}
  
  formFields.value.forEach(field => {
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
          data[field.code] = undefined
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
  
  formFields.value.forEach(field => {
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

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return
  
  try {
    // 验证表单
    await formRef.value.validate()
    
    submitting.value = true
    
    // 触发提交事件
    emit('submit', formData.value)
    
  } catch (error) {
    console.error('[FormDialog] 表单验证失败:', error)
  } finally {
    submitting.value = false
  }
}

// 关闭对话框
const handleClose = () => {
  formRef.value?.resetFields()
  emit('close')
  emit('update:modelValue', false)
}

// 监听对话框显示状态
watch(() => props.modelValue, (visible) => {
  if (visible) {
    initFormData()
  }
}, { immediate: true })

// 暴露方法给父组件
defineExpose({
  formRef,
  formData,
  validate: () => formRef.value?.validate(),
  resetFields: () => formRef.value?.resetFields()
})
</script>

<style scoped>
.field-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>

