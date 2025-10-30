<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElForm, ElMessage } from 'element-plus'
import type { ServiceTree } from '@/types'

interface Props {
  visible: boolean
  mode: 'create' | 'edit'
  data: Partial<ServiceTree>
  parentNode?: ServiceTree | null
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'submit', data: Partial<ServiceTree>): void
}

const props = withDefaults(defineProps<Props>(), {
  parentNode: null
})

const emit = defineEmits<Emits>()

// 表单ref
const formRef = ref<InstanceType<typeof ElForm>>()

// 表单数据
const formData = ref<Partial<ServiceTree>>({
  name: '',
  code: '',
  description: '',
  tags: '',
  type: 'package',
  parent_id: 0,
  app_id: 0
})

// 计算属性
const dialogTitle = computed(() => {
  return props.mode === 'create' ? '创建服务节点' : '编辑服务节点'
})

const isRootNode = computed(() => {
  return props.mode === 'create' && props.data.parent_id === 0
})

const parentPath = computed(() => {
  if (!props.parentNode) return '/'
  return props.parentNode.full_code_path || '/'
})

// 表单验证规则
const rules = {
  name: [
    { required: true, message: '请输入节点名称', trigger: 'blur' },
    { min: 2, max: 50, message: '节点名称长度在 2 到 50 个字符', trigger: 'blur' },
    { pattern: /^[\u4e00-\u9fa5a-zA-Z0-9_\s]+$/, message: '节点名称只能包含中文、英文、数字、下划线和空格', trigger: 'blur' }
  ],
  code: [
    { required: true, message: '请输入节点代码', trigger: 'blur' },
    { min: 2, max: 30, message: '节点代码长度在 2 到 30 个字符', trigger: 'blur' },
    { pattern: /^[a-zA-Z][a-zA-Z0-9_]*$/, message: '节点代码必须以字母开头，只能包含字母、数字和下划线', trigger: 'blur' }
  ],
  description: [
    { max: 200, message: '描述长度不能超过 200 个字符', trigger: 'blur' }
  ],
  tags: [
    { max: 100, message: '标签长度不能超过 100 个字符', trigger: 'blur' }
  ],
  type: [
    { required: true, message: '请选择节点类型', trigger: 'change' }
  ]
}

// 生成代码建议
const generateCode = () => {
  const name = formData.value.name?.trim()
  if (!name) return

  // 将中文或英文转换为适合的代码
  let code = name.toLowerCase()

  // 移除特殊字符，保留字母、数字、中文和空格
  code = code.replace(/[^\u4e00-\u9fa5a-zA-Z0-9\s]/g, '')

  // 将空格替换为下划线
  code = code.replace(/\s+/g, '_')

  // 移除连续的下划线
  code = code.replace(/_+/g, '_')

  // 移除首尾的下划线
  code = code.replace(/^_+|_+$/g, '')

  // 如果包含中文，尝试使用拼音或直接使用
  if (/[\u4e00-\u9fa5]/.test(code)) {
    // 简单的中文到拼音映射（这里只是示例，实际应该使用完整的拼音库）
    const pinyinMap: Record<string, string> = {
      '管': 'guan',
      '理': 'li',
      '系': 'xi',
      '统': 'tong',
      '招': 'zhao',
      '聘': 'pin',
      '绩': 'ji',
      '效': 'xiao',
      '薪': 'xin',
      '酬': 'chou',
      '项': 'xiang',
      '目': 'mu',
      '任': 'ren',
      '务': 'wu',
      '财': 'cai',
      '务': 'wu'
    }

    let pinyinCode = ''
    for (const char of code) {
      pinyinCode += pinyinMap[char] || char
    }
    code = pinyinCode
  }

  formData.value.code = code
}

// 处理类型变化
const handleTypeChange = (type: string) => {
  // 根据类型调整默认值
  if (type === 'function') {
    if (!formData.value.description) {
      formData.value.description = `${formData.value.name}功能模块`
    }
  } else {
    if (!formData.value.description) {
      formData.value.description = `${formData.value.name}管理目录`
    }
  }
}

// 重置表单
const resetForm = () => {
  formData.value = {
    name: '',
    code: '',
    description: '',
    tags: '',
    type: 'package',
    parent_id: props.data.parent_id || 0,
    app_id: props.data.app_id || 0
  }
  formRef.value?.clearValidate()
}

// 处理取消
const handleCancel = () => {
  emit('update:visible', false)
  resetForm()
}

// 处理提交
const handleSubmit = async () => {
  try {
    await formRef.value?.validate()

    const submitData = { ...formData.value }

    // 处理标签
    if (submitData.tags) {
      // 移除空格，确保用逗号分隔
      submitData.tags = submitData.tags.split(',').map(tag => tag.trim()).filter(Boolean).join(',')
    }

    emit('submit', submitData)

  } catch (error) {
    console.error('表单验证失败:', error)
  }
}

// 监听显示状态
watch(() => props.visible, (visible) => {
  if (visible) {
    // 重置表单并填充数据
    resetForm()
    if (props.mode === 'edit') {
      Object.assign(formData.value, props.data)
    } else {
      formData.value.parent_id = props.data.parent_id || 0
      formData.value.app_id = props.data.app_id || 0
    }
  }
})

// 监听名称变化，自动生成代码
watch(() => formData.value.name, (newName) => {
  if (props.mode === 'create' && !formData.value.code) {
    generateCode()
  }
})
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="dialogTitle"
    width="600px"
    :before-close="handleCancel"
    destroy-on-close
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="rules"
      label-width="100px"
      label-position="left"
    >
      <!-- 基本信息 -->
      <div class="form-section">
        <h4 class="section-title">基本信息</h4>

        <el-form-item label="节点名称" prop="name">
          <el-input
            v-model="formData.name"
            placeholder="请输入节点名称"
            maxlength="50"
            show-word-limit
            clearable
          />
        </el-form-item>

        <el-form-item label="节点代码" prop="code">
          <el-input
            v-model="formData.code"
            placeholder="请输入节点代码（英文）"
            maxlength="30"
            clearable
          >
            <template #append>
              <el-button @click="generateCode" size="small">
                自动生成
              </el-button>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="节点类型" prop="type">
          <el-radio-group v-model="formData.type" @change="handleTypeChange">
            <el-radio label="package">
              <el-icon><Folder /></el-icon>
              目录（Package）
            </el-radio>
            <el-radio label="function">
              <el-icon><Document /></el-icon>
              功能（Function）
            </el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item label="父级目录" v-if="parentNode">
          <el-input
            :value="`${parentNode.name} (${parentNode.full_code_path})`"
            readonly
            placeholder="根目录"
          />
        </el-form-item>
      </div>

      <!-- 详细信息 -->
      <div class="form-section">
        <h4 class="section-title">详细信息</h4>

        <el-form-item label="描述" prop="description">
          <el-input
            v-model="formData.description"
            type="textarea"
            :rows="3"
            placeholder="请输入节点描述"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="标签" prop="tags">
          <el-input
            v-model="formData.tags"
            placeholder="请输入标签，用逗号分隔"
            maxlength="100"
            clearable
          >
            <template #prepend>
              <el-icon><CollectionTag /></el-icon>
            </template>
          </el-input>
          <div class="form-tip">
            例如：管理,系统,HR 或 recruitment,interview,hr
          </div>
        </el-form-item>
      </div>

      <!-- 预览信息 -->
      <div class="form-section" v-if="formData.name && formData.code">
        <h4 class="section-title">预览信息</h4>

        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="完整路径">
            <code>{{ parentPath === '/' ? '' : parentPath + '.' }}{{ formData.code }}</code>
          </el-descriptions-item>
          <el-descriptions-item label="节点名称">
            {{ formData.name }}
          </el-descriptions-item>
          <el-descriptions-item label="节点类型">
            <el-tag :type="formData.type === 'package' ? 'primary' : 'success'" size="small">
              {{ formData.type === 'package' ? '目录' : '功能' }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleCancel">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="false">
          {{ mode === 'create' ? '创建' : '更新' }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<style scoped>
.form-section {
  margin-bottom: 24px;
}

.section-title {
  margin: 0 0 16px 0;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  padding-bottom: 8px;
  border-bottom: 1px solid #e4e7ed;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* Element Plus 样式覆盖 */
:deep(.el-radio) {
  margin-right: 20px;
  margin-bottom: 8px;
}

:deep(.el-radio__label) {
  display: flex;
  align-items: center;
  gap: 6px;
}

:deep(.el-input-group__append) {
  padding: 0 8px;
}

:deep(.el-descriptions) {
  margin: 0;
}

:deep(.el-descriptions__label) {
  font-weight: 500;
  color: #606266;
}

:deep(.el-descriptions__content) {
  color: #303133;
}

code {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 4px;
  color: #e6a23c;
}

/* 响应式设计 */
@media (max-width: 768px) {
  :deep(.el-dialog) {
    width: 90%;
    margin: 5vh auto;
  }

  :deep(.el-form-item__label) {
    font-size: 14px;
  }

  .section-title {
    font-size: 13px;
  }
}
</style>