<template>
  <el-dialog
    v-model="dialogVisible"
    title="推送目录到应用中心"
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
      data-testid="push-to-hub-dialog"
    >
      <!-- 基本信息 -->
      <el-form-item label="目录名称" prop="name">
        <el-input
          v-model="form.name"
          placeholder="留空则保持原值"
          maxlength="100"
          show-word-limit
          data-testid="push-to-hub-name"
        />
      </el-form-item>

      <el-form-item label="目录描述" prop="description">
        <RichTextEditor
          :model-value="form.description || ''"
          @update:modelValue="form.description = $event"
          placeholder="留空则保持原值，支持富文本格式..."
        />
      </el-form-item>

      <el-form-item label="分类">
        <el-input
          v-model="form.category"
          placeholder="留空则保持原值，例如：工具、业务系统、数据管理等"
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
          placeholder="留空则保持原值，可输入自定义标签按回车添加，或从常用标签中选择"
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

      <el-form-item label="当前版本">
        <el-text type="info">{{ currentVersion || '未知' }}</el-text>
      </el-form-item>

      <el-form-item label="下一版本">
        <el-text type="primary" style="font-weight: 600">{{ nextVersion || '—' }}</el-text>
        <el-text type="info" size="small" style="display: block; margin-top: 4px">
          版本将自动递增，无需填写
        </el-text>
      </el-form-item>

      <el-form-item label="本版本更新说明" prop="update_description">
        <el-input
          v-model="form.update_description"
          type="textarea"
          :rows="3"
          placeholder="选填，例如：新增 xxx 功能、修复 xxx 问题。便于在 Hub 历史版本中查看本版本做了哪些改动"
          maxlength="500"
          show-word-limit
        />
      </el-form-item>

      <el-form-item label="提示">
        <el-alert
          type="info"
          :closable="false"
          show-icon
        >
          <template #default>
            <div>
              <p>推送目录时，将更新该目录在 Hub 中的内容。</p>
              <p>请确保目录已创建快照（保存/提交代码时会自动创建）。</p>
            </div>
          </template>
        </el-alert>
      </el-form-item>

      <!-- 推送目标 -->
      <el-divider>推送目标</el-divider>

      <el-form-item label="推送到">
        <el-radio-group v-model="pushTarget" @change="handleTargetChange">
          <el-radio label="local">本站 Hub</el-radio>
          <el-radio label="remote">远程站点</el-radio>
        </el-radio-group>
      </el-form-item>

      <template v-if="pushTarget === 'remote'">
        <el-form-item label="远程地址" prop="remote_hub_url" :rules="[{ required: true, message: '请输入远程 Hub 地址', trigger: 'blur' }]">
          <el-input
            v-model="form.remote_hub_url"
            placeholder="例如：http://hub.example.com 或 125.122.96.207:8999"
          />
        </el-form-item>

        <el-form-item label="Pub Key" prop="pub_key" :rules="[{ required: true, message: '请输入远程站点的 Pub Key', trigger: 'blur' }]">
          <el-input
            v-model="form.pub_key"
            placeholder="在远程站点的个人设置中生成的发布密钥"
            show-password
          />
          <el-text type="info" size="small">在远程站点「个人设置 > 发布密钥」中生成</el-text>
        </el-form-item>
      </template>

      <!-- 服务费设置 -->
      <el-divider>服务费设置（可选）</el-divider>

      <el-form-item label="个人用户服务费">
        <el-input-number
          v-model="form.service_fee_personal"
          :min="0"
          :precision="2"
          :step="10"
          placeholder="留空则保持原值"
          style="width: 200px"
        />
        <span class="price-unit">元</span>
      </el-form-item>

      <el-form-item label="企业用户服务费">
        <el-input-number
          v-model="form.service_fee_enterprise"
          :min="0"
          :precision="2"
          :step="50"
          placeholder="留空则保持原值"
          style="width: 200px"
        />
        <span class="price-unit">元</span>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button data-testid="push-to-hub-cancel" @click="handleClose">取消</el-button>
      <el-button type="primary" data-testid="push-to-hub-submit" @click="handleSubmit" :loading="submitting">
        推送
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, defineAsyncComponent } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { pushDirectoryToHub, getHubPushFormInfo, type PushDirectoryToHubReq } from '@/api/hub'
import type { ServiceTree } from '@/types'
import type { App } from '@/types'
import { navigateToHubDirectoryDetail } from '@/utils/hub-navigation'
const RichTextEditor = defineAsyncComponent(() => import('@/shared/components/RichTextEditor.vue'))

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
const pushTarget = ref<'local' | 'remote'>('local')

// 表单数据
const form = ref<Partial<PushDirectoryToHubReq>>({
  name: '',
  description: '',
  category: '',
  tags: [],
  service_fee_personal: 0,
  service_fee_enterprise: 0,
  update_description: '',
  remote_hub_url: '',
  pub_key: '',
})

// 选中的目录路径、当前版本、下一版本（自动递增，只读展示）
const selectedDirectoryPath = ref<string>('')
const currentVersion = ref<string>('')
const nextVersion = ref<string>('')

// 常用标签
const commonTags = ['工具', '业务系统', '数据管理', '工作流', '报表', 'API', '集成']

// 表单验证规则（版本由后端自动递增，无需校验）
const rules = {}

// 监听对话框打开，初始化表单
watch(dialogVisible, (visible) => {
  if (visible) {
    initForm()
  }
})

// 初始化表单：调用接口获取当前已发布信息并预填，同时拿到下一版本号
const initForm = async () => {
  if (!props.selectedNode || !props.currentApp) {
    ElMessage.warning('请先选择要推送的目录')
    return
  }

  const node = props.selectedNode

  if (node.type !== 'package') {
    ElMessage.warning('请选择目录节点（package 类型）')
    return
  }

  if (!node.hub_full_code_path) {
    ElMessage.warning('该目录尚未发布到 Hub，请先使用"发布到应用中心"功能')
    dialogVisible.value = false
    return
  }

  selectedDirectoryPath.value = node.full_code_path || ''
  currentVersion.value = node.hub_version_num != null ? `v${node.hub_version_num}` : '未知'
  nextVersion.value = ''

  // 先清空表单，再请求预填数据
  form.value = {
    name: '',
    description: '',
    category: '',
    tags: [],
    service_fee_personal: 0,
    service_fee_enterprise: 0,
    update_description: '',
    api_key: '',
  }

  loading.value = true
  try {
    const info = await getHubPushFormInfo({
      source_user: props.currentApp.user,
      source_app: props.currentApp.code,
      source_directory_path: node.full_code_path || '',
    })
    currentVersion.value = info.current_version
    nextVersion.value = info.next_version
    form.value = {
      name: info.name ?? '',
      description: info.description ?? '',
      category: info.category ?? '',
      tags: Array.isArray(info.tags) ? [...info.tags] : [],
      service_fee_personal: info.service_fee_personal ?? 0,
      service_fee_enterprise: info.service_fee_enterprise ?? 0,
      update_description: '',
    }
  } catch (e: any) {
    ElMessage.warning(e?.message || '获取已发布信息失败，可手动填写后推送')
  } finally {
    loading.value = false
  }
}

const handleTargetChange = (val: 'local' | 'remote') => {
  if (val === 'local') {
    form.value.remote_hub_url = ''
    form.value.pub_key = ''
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
      const requestData: PushDirectoryToHubReq = {
        source_user: props.currentApp.user,
        source_app: props.currentApp.code,
        source_directory_path: node.full_code_path,
        ...(form.value.name ? { name: form.value.name } : {}),
        ...(form.value.description ? { description: form.value.description } : {}),
        ...(form.value.category ? { category: form.value.category } : {}),
        ...(form.value.tags && form.value.tags.length > 0 ? { tags: form.value.tags } : {}),
        ...(form.value.service_fee_personal ? { service_fee_personal: form.value.service_fee_personal } : {}),
        ...(form.value.service_fee_enterprise ? { service_fee_enterprise: form.value.service_fee_enterprise } : {}),
        ...(form.value.update_description ? { update_description: form.value.update_description } : {}),
      }
      if (pushTarget.value === 'remote') {
        requestData.remote_hub_url = form.value.remote_hub_url
        requestData.pub_key = form.value.pub_key
      }

      // 调用推送接口
      const response = await pushDirectoryToHub(requestData)

      ElMessage.success('推送成功！')

      // 提供跳转到 Hub 的选项
      try {
        await ElMessageBox.confirm(
          `目录已成功推送到应用中心！\n\n版本: ${response.old_version} → ${response.new_version}\n包含 ${response.directory_count} 个子目录，${response.file_count} 个文件\n\n是否跳转到应用中心查看？`,
          '推送成功',
          {
            confirmButtonText: '跳转查看',
            cancelButtonText: '稍后查看',
            type: 'success'
          }
        )

        // 用户确认，跳转到 Hub 目录详情页
        navigateToHubDirectoryDetail(response.hub_full_code_path)
      } catch {
        // 用户取消，不做任何操作
      }

      emit('success')
      handleClose()
    } catch (error: any) {
      ElMessage.error(`推送失败: ${error.message || '未知错误'}`)
      console.error('推送失败:', error)
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
    update_description: '',
    remote_hub_url: '',
    pub_key: '',
  }
  pushTarget.value = 'local'
  selectedDirectoryPath.value = ''
  currentVersion.value = ''
  nextVersion.value = ''
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
  color: var(--el-text-color-regular);
}

.price-unit {
  margin-left: 10px;
  color: var(--el-text-color-placeholder);
}
</style>
