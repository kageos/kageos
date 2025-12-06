<template>
  <el-dialog
    v-model="dialogVisible"
    title="发布到应用中心"
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
      <el-form-item label="应用名称" prop="name">
        <el-input
          v-model="form.name"
          placeholder="请输入应用名称"
          maxlength="100"
          show-word-limit
        />
      </el-form-item>

      <el-form-item label="应用描述" prop="description">
        <RichTextEditor
          v-model="form.description"
          placeholder="请输入应用描述，支持富文本格式..."
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

      <!-- 函数组信息 -->
      <el-divider>函数组信息</el-divider>

      <el-form-item label="已选函数组">
        <el-tag
          v-for="(pkg, index) in selectedPackages"
          :key="index"
          type="info"
          style="margin-right: 8px; margin-bottom: 8px"
        >
          {{ pkg.group_name || pkg.full_group_code }}
        </el-tag>
        <el-text v-if="selectedPackages.length === 0" type="info" size="small">
          请选择要发布的函数组
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
import { publishHubApp, type PublishHubAppReq, type PackageSourceCode } from '@/api/hub'
import { getFunctionGroupInfo } from '@/api/function'
import type { ServiceTree } from '@/types'
import type { App } from '@/types'
import RichTextEditor from './RichTextEditor.vue'

interface Props {
  modelValue: boolean
  selectedNode?: ServiceTree | null  // 选中的函数组节点
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
const form = ref<Partial<PublishHubAppReq>>({
  name: '',
  description: '',
  category: '',
  tags: [],
  service_fee_personal: 0,
  service_fee_enterprise: 0,
  version: '',
  api_key: '',
  packages: []
})

// 已选择的函数组信息（用于显示）
const selectedPackages = ref<Array<{
  full_group_code: string
  group_name: string
  source_code: string
  functions?: FunctionInfo[]  // 函数列表
}>>([])

// 常用标签
const commonTags = ['工具', '业务系统', '数据管理', '工作流', '报表', 'API', '集成']

// 表单验证规则
const rules = {
  name: [
    { required: true, message: '请输入应用名称', trigger: 'blur' }
  ],
  description: [
    { 
      required: true, 
      message: '请输入应用描述', 
      trigger: 'blur',
      validator: (rule: any, value: string, callback: Function) => {
        // 富文本编辑器返回的是 HTML，需要检查是否有实际内容
        if (!value || value.trim() === '' || value === '<p></p>' || value === '<p><br></p>') {
          callback(new Error('请输入应用描述'))
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
    ElMessage.warning('请先选择要发布的函数组')
    return
  }

  const node = props.selectedNode as any
  
  // 检查是否是函数组节点
  if (!node.isGroup || !node.full_group_code) {
    ElMessage.warning('请选择业务系统（函数组）节点')
    return
  }

  loading.value = true
  try {
    // 获取函数组信息
    const groupInfo = await getFunctionGroupInfo(node.full_group_code)
    
    // 初始化表单数据
    form.value = {
      name: groupInfo.group_name || node.name || '',
      description: groupInfo.description || '',
      category: '',
      tags: [],
      service_fee_personal: 0,
      service_fee_enterprise: 0,
      version: groupInfo.version || 'v1',
      api_key: '',
      packages: []
    }

    // 保存函数组信息（包括函数列表）
    selectedPackages.value = [{
      full_group_code: groupInfo.full_group_code,
      group_name: groupInfo.group_name,
      source_code: groupInfo.source_code,
      functions: groupInfo.functions || [] // 保存函数列表
    }]

    // 设置默认应用名称和描述
    if (!form.value.name) {
      form.value.name = `${props.currentApp.name || props.currentApp.code} - ${groupInfo.group_name}`
    }
  } catch (error: any) {
    ElMessage.error(`获取函数组信息失败: ${error.message || '未知错误'}`)
    console.error('获取函数组信息失败:', error)
  } finally {
    loading.value = false
  }
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid: boolean) => {
    if (!valid) return

    if (!props.currentApp) {
      ElMessage.error('缺少应用信息')
      return
    }

    if (selectedPackages.value.length === 0) {
      ElMessage.error('请至少选择一个函数组')
      return
    }

    submitting.value = true
    try {
      // 组装请求数据
      const packages: PackageSourceCode[] = selectedPackages.value.map(pkg => {
        // 从 full_group_code 提取 package 路径
        // 例如：/luobei/testgroup/tools/tools_cashier -> tools/tools_cashier
        const parts = pkg.full_group_code.split('/').filter(Boolean)
        const packagePath = parts.slice(2).join('/') // 跳过 user 和 app

        return {
          package: packagePath,
          full_group_code: pkg.full_group_code,
          source_code: pkg.source_code,
          functions: pkg.functions || [] // 传递函数列表
        }
      })

      const requestData: PublishHubAppReq = {
        source_user: props.currentApp.user,
        source_app: props.currentApp.code,
        name: form.value.name!,
        description: form.value.description!,
        category: form.value.category || '',
        tags: form.value.tags || [],
        service_fee_personal: form.value.service_fee_personal || 0,
        service_fee_enterprise: form.value.service_fee_enterprise || 0,
        version: form.value.version || 'v1',
        packages,
        ...(form.value.api_key ? { api_key: form.value.api_key } : {})
      }

      // 调用发布接口
      const response = await publishHubApp(requestData)

      ElMessage.success('发布成功！')
      
      // 提供跳转到 Hub 的选项
      try {
        await ElMessageBox.confirm(
          `应用已成功发布到应用中心！\n\n应用ID: ${response.hub_app_id}\n\n是否跳转到应用中心查看？`,
          '发布成功',
          {
            confirmButtonText: '跳转查看',
            cancelButtonText: '稍后查看',
            type: 'success'
          }
        )
        
        // 用户确认，跳转到 Hub 应用详情页
        const { navigateToHubAppDetail } = await import('@/utils/hub-navigation')
        navigateToHubAppDetail(response.hub_app_id)
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
    version: '',
    api_key: '',
    packages: []
  }
  selectedPackages.value = []
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

