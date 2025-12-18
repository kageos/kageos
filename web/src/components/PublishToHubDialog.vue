<template>
  <el-dialog
    v-model="dialogVisible"
    title="发布目录到应用中心"
    width="800px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="120px"
      v-loading="loading"
    >
      <!-- 基本信息 -->
      <el-form-item label="目录名称" prop="name">
        <el-input
          v-model="form.name"
          placeholder="请输入目录名称"
          maxlength="100"
          show-word-limit
        />
      </el-form-item>

      <el-form-item label="目录描述" prop="description">
        <RichTextEditor
          v-model="form.description"
          placeholder="请输入目录描述，支持富文本格式..."
        />
      </el-form-item>

      <el-form-item label="分类">
        <el-input
          v-model="form.category"
          placeholder="例如：工具、业务系统、数据管理等"
          maxlength="50"
        />
      </el-form-item>

      <el-form-item label="标签">
        <el-select
          v-model="form.tags"
          multiple
          filterable
          allow-create
          default-first-option
          placeholder="输入标签后按回车添加"
          style="width: 100%"
        >
          <el-option
            v-for="tag in commonTags"
            :key="tag"
            :label="tag"
            :value="tag"
          />
        </el-select>
      </el-form-item>

      <!-- 目录信息 -->
      <el-divider>目录信息</el-divider>

      <el-form-item label="目录路径">
        <el-text type="info">{{ selectedDirectoryPath || '未选择目录' }}</el-text>
      </el-form-item>

      <el-form-item label="提示">
        <el-alert
          type="info"
          :closable="false"
          show-icon
        >
          <template #default>
            <div>
              <p>发布目录时，将包含该目录下的所有子目录和文件。</p>
              <p>请确保目录已创建快照（保存/提交代码时会自动创建）。</p>
            </div>
          </template>
        </el-alert>
      </el-form-item>

      <!-- 服务费设置 -->
      <el-divider>服务费设置</el-divider>

      <el-form-item label="个人用户服务费">
        <el-input-number
          v-model="form.service_fee_personal"
          :min="0"
          :precision="2"
          :step="10"
          placeholder="0"
          style="width: 200px"
        />
        <span style="margin-left: 10px; color: #909399">元</span>
        <el-text type="info" size="small" style="margin-left: 20px">
          个人用户克隆时收取的服务费
        </el-text>
      </el-form-item>

      <el-form-item label="企业用户服务费">
        <el-input-number
          v-model="form.service_fee_enterprise"
          :min="0"
          :precision="2"
          :step="50"
          placeholder="0"
          style="width: 200px"
        />
        <span style="margin-left: 10px; color: #909399">元</span>
        <el-text type="info" size="small" style="margin-left: 20px">
          企业用户克隆时收取的服务费
        </el-text>
      </el-form-item>

      <!-- 私有化部署（可选） -->
      <el-form-item label="API Key">
        <el-input
          v-model="form.api_key"
          type="password"
          placeholder="私有化部署需要填写 API Key（可选）"
          show-password
        />
        <el-text type="info" size="small" style="display: block; margin-top: 5px">
          如果是私有化部署，需要填写 Hub API Key
        </el-text>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" @click="handleSubmit" :loading="submitting">
        发布
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { publishDirectoryToHub, type PublishDirectoryToHubReq } from '@/api/hub'
import type { ServiceTree } from '@/types'
import type { App } from '@/types'
import RichTextEditor from './RichTextEditor.vue'

interface Props {
  modelValue: boolean
  selectedNode?: ServiceTree | null  // 选中的目录节点
  currentApp?: App | null            // 当前应用
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const formRef = ref()
const loading = ref(false)
const submitting = ref(false)

// 表单数据
const form = ref<Partial<PublishDirectoryToHubReq>>({
  name: '',
  description: '',
  category: '',
  tags: [],
  service_fee_personal: 0,
  service_fee_enterprise: 0,
  api_key: '',
})

// 选中的目录路径
const selectedDirectoryPath = ref<string>('')

// 常用标签
const commonTags = ['工具', '业务系统', '数据管理', '工作流', '报表', 'API', '集成']

// 表单验证规则
const rules = {
  name: [
    { required: true, message: '请输入目录名称', trigger: 'blur' }
  ],
  description: [
    {
      required: true,
      message: '请输入目录描述',
      trigger: 'blur',
      validator: (rule: any, value: string, callback: Function) => {
        // 富文本编辑器返回的是 HTML，需要检查是否有实际内容
        if (!value || value.trim() === '' || value === '<p></p>' || value === '<p><br></p>') {
          callback(new Error('请输入目录描述'))
        } else {
          callback()
        }
      }
    }
  ]
}

// 监听对话框打开，初始化表单
watch(dialogVisible, (visible) => {
  if (visible) {
    initForm()
  }
})

// 初始化表单
const initForm = async () => {
  if (!props.selectedNode || !props.currentApp) {
    ElMessage.warning('请先选择要发布的目录')
    return
  }

  const node = props.selectedNode

  // 检查是否是目录节点（package 类型）
  if (node.type !== 'package') {
    ElMessage.warning('请选择目录节点（package 类型）')
    return
  }

  // 设置目录路径
  selectedDirectoryPath.value = node.full_code_path || ''

  // 初始化表单数据
  form.value = {
    name: node.name || '',
    description: node.description || '',
    category: '',
    tags: node.tags ? node.tags.split(',').filter(Boolean) : [],
    service_fee_personal: 0,
    service_fee_enterprise: 0,
    api_key: '',
  }

  // 设置默认目录名称
  if (!form.value.name) {
    form.value.name = node.name || node.code || ''
  }
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid: boolean) => {
    if (!valid) return

    if (!props.currentApp || !props.selectedNode) {
      ElMessage.error('缺少应用信息或目录信息')
      return
    }

    const node = props.selectedNode
    if (node.type !== 'package') {
      ElMessage.error('请选择目录节点')
      return
    }

    submitting.value = true
    try {
      const requestData: PublishDirectoryToHubReq = {
        source_user: props.currentApp.user,
        source_app: props.currentApp.code,
        source_directory_path: node.full_code_path,
        name: form.value.name!,
        description: form.value.description || '',
        category: form.value.category || '',
        tags: form.value.tags || [],
        service_fee_personal: form.value.service_fee_personal || 0,
        service_fee_enterprise: form.value.service_fee_enterprise || 0,
        ...(form.value.api_key ? { api_key: form.value.api_key } : {})
      }

      // 调用发布接口
      const response = await publishDirectoryToHub(requestData)

      ElMessage.success('发布成功！')

      // 提供跳转到 Hub 的选项
      try {
        await ElMessageBox.confirm(
          `目录已成功发布到应用中心！\n\n目录ID: ${response.hub_directory_id}\n包含 ${response.directory_count} 个子目录，${response.file_count} 个文件\n\n是否跳转到应用中心查看？`,
          '发布成功',
          {
            confirmButtonText: '跳转查看',
            cancelButtonText: '稍后查看',
            type: 'success'
          }
        )

        // 用户确认，跳转到 Hub 目录详情页
        const { navigateToHubDirectoryDetail } = await import('@/utils/hub-navigation')
        navigateToHubDirectoryDetail(response.hub_directory_id)
      } catch {
        // 用户取消，不做任何操作
      }

      emit('success')
      handleClose()
    } catch (error: any) {
      ElMessage.error(`发布失败: ${error.message || '未知错误'}`)
      console.error('发布失败:', error)
    } finally {
      submitting.value = false
    }
  })
}

// 关闭对话框
const handleClose = () => {
  form.value = {
    name: '',
    description: '',
    category: '',
    tags: [],
    service_fee_personal: 0,
    service_fee_enterprise: 0,
    api_key: '',
  }
  selectedDirectoryPath.value = ''
  formRef.value?.resetFields()
  emit('update:modelValue', false)
}
</script>

<style scoped>
:deep(.el-form-item__label) {
  font-weight: 500;
}

:deep(.el-divider__text) {
  font-size: 14px;
  font-weight: 500;
  color: #606266;
}
</style>
