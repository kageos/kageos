<template>
  <div class="llm-management">
    <el-card shadow="hover" class="management-card">
      <template #header>
        <div class="card-header">
          <el-button
            link
            :icon="ArrowLeft"
            @click="handleBack"
            class="back-button"
          >
            返回
          </el-button>
          <h2>LLM 管理</h2>
          <el-button type="primary" :icon="Plus" @click="handleCreate">
            新增 LLM 配置
          </el-button>
        </div>
      </template>

      <!-- 标签页：我的/市场 -->
      <el-tabs v-model="activeTab" @tab-change="handleTabChange" class="scope-tabs">
        <el-tab-pane label="我的 LLM 配置" name="mine" />
        <el-tab-pane label="LLM 配置市场" name="market" />
      </el-tabs>
      <el-divider />

      <!-- 统计卡片区 -->
      <div class="stats-section">
        <StatCard
          :icon="Cpu"
          label="总数"
          :value="stats.total"
          icon-color="var(--el-color-primary)"
        />
        <StatCard
          :icon="CircleCheck"
          label="默认配置"
          :value="stats.default"
          icon-color="var(--el-color-success)"
        />
        <StatCard
          :icon="Shop"
          label="提供商数"
          :value="stats.providers"
          icon-color="var(--el-color-info)"
        />
      </div>

      <!-- 卡片列表 -->
      <div v-loading="loading" class="cards-container">
        <LLMCard
          v-for="llm in tableData"
          :key="llm.id"
          :llm="llm"
          :agents="agentsByLLM.get(llm.id) || []"
          @detail="handleDetail"
          @edit="handleEdit"
          @set-default="handleSetDefault"
          @delete="handleDelete"
        />
        <el-empty v-if="!loading && tableData.length === 0" description="暂无数据" />
      </div>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.page_size"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </el-card>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="800px"
      :close-on-click-modal="false"
      @close="handleDialogClose"
    >
      <el-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-width="120px"
      >
        <el-form-item label="名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入配置名称" />
        </el-form-item>
        <el-form-item label="提供商" prop="provider">
          <el-select
            v-model="formData.provider"
            placeholder="请选择或输入提供商"
            filterable
            allow-create
            default-first-option
            style="width: 100%"
          >
            <el-option label="OpenAI" value="openai" />
            <el-option label="Claude" value="claude" />
            <el-option label="GLM" value="glm" />
            <el-option label="DeepSeek" value="deepseek" />
            <el-option label="千问" value="qwen" />
            <el-option label="Kimi" value="kimi" />
            <el-option label="豆包" value="doubao" />
            <el-option label="Gemini" value="gemini" />
            <el-option label="本地模型" value="local" />
            <el-option label="其他" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item label="模型" prop="model">
          <el-input v-model="formData.model" placeholder="请输入模型名称，如：gpt-4, claude-3" />
        </el-form-item>
        <el-form-item label="API Key">
          <el-input
            v-model="formData.api_key"
            type="password"
            placeholder="请输入 API Key"
            show-password
          />
        </el-form-item>
        <el-form-item label="API 地址">
          <el-input
            v-model="formData.api_base"
            placeholder="请输入 API 地址，如：https://api.openai.com/v1"
          />
        </el-form-item>
        <el-form-item label="超时时间（秒）">
          <el-input-number
            v-model="formData.timeout"
            :min="1"
            :max="600"
            placeholder="默认120秒"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="最大 Token">
          <el-input-number
            v-model="formData.max_tokens"
            :min="1"
            :max="100000"
            placeholder="默认4000"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="额外配置（JSON）">
          <el-input
            v-model="formData.extra_config"
            type="textarea"
            :rows="3"
            placeholder='请输入JSON格式的额外配置，如：{"temperature": 0.7}'
          />
        </el-form-item>
        <el-form-item label="设为默认">
          <el-switch v-model="formData.is_default" />
        </el-form-item>
        <el-form-item label="可见性">
          <el-radio-group v-model="formData.visibility">
            <el-radio :label="0">公开（所有人可见）</el-radio>
            <el-radio :label="1">私有（仅管理员可见）</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="管理员">
          <el-input
            v-model="formData.admin"
            placeholder="管理员列表（逗号分隔，如：user1,user2）"
          />
          <div style="margin-top: 8px; font-size: 12px; color: #909399;">
            提示：多个管理员用逗号分隔，留空则默认为创建用户
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          确定
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElForm } from 'element-plus'
import { ArrowLeft, Plus } from '@element-plus/icons-vue'
import {
  getLLMList,
  createLLM,
  updateLLM,
  deleteLLM,
  setDefaultLLM,
  type LLMInfo,
  type LLMListReq,
  type LLMCreateReq,
  type LLMUpdateReq
} from '@/api/agent'
import type { FormRules } from 'element-plus'

const router = useRouter()

// 表格数据
const loading = ref(false)
const tableData = ref<LLMInfo[]>([])

// 每个LLM对应的智能体列表（key: llm_id, value: AgentInfo[]）
const agentsByLLM = ref<Map<number, AgentInfo[]>>(new Map())

// 标签页
const activeTab = ref<'mine' | 'market'>('mine')

// 分页
const pagination = reactive({
  page: 1,
  page_size: 10,
  total: 0
})

// 统计信息
const stats = computed(() => {
  const total = tableData.value.length
  const defaultCount = tableData.value.filter(llm => llm.is_default).length
  const providers = new Set(tableData.value.map(llm => llm.provider)).size
  return { total, default: defaultCount, providers }
})

// 对话框
const dialogVisible = ref(false)
const dialogTitle = ref('新增 LLM 配置')
const formRef = ref<InstanceType<typeof ElForm>>()
const submitting = ref(false)

// 表单数据
const formData = reactive<LLMCreateReq & { id?: number }>({
  name: '',
  provider: '',
  model: '',
  api_key: '',
  api_base: '',
  timeout: 120,
  max_tokens: 4000,
  extra_config: '',
  is_default: false,
  visibility: 0, // 默认公开
  admin: '' // 默认空，后端会自动设置为创建用户
})

// 表单验证规则
const rules: FormRules = {
  name: [{ required: true, message: '请输入配置名称', trigger: 'blur' }],
  provider: [{ required: true, message: '请选择提供商', trigger: 'change' }],
  model: [{ required: true, message: '请输入模型名称', trigger: 'blur' }]
}

// 标签页切换处理
function handleTabChange(tabName: string) {
  activeTab.value = tabName as 'mine' | 'market'
  pagination.page = 1 // 切换标签页时重置页码
  loadData()
}

// 加载数据
async function loadData() {
  loading.value = true
  try {
    const params: LLMListReq = {
      page: pagination.page,
      page_size: pagination.page_size,
      scope: activeTab.value // 添加 scope 参数
    }
    const res = await getLLMList(params)
    // 响应拦截器已经返回了 data，所以 res 就是 { configs: [], total: 0 }
    tableData.value = res.configs || []
    pagination.total = res.total || 0
    
    // 为每个LLM加载使用它的智能体列表
    await loadAgentsForLLMs()
  } catch (error: any) {
    ElMessage.error(error.message || '获取列表失败')
  } finally {
    loading.value = false
  }
}

// 获取使用指定LLM配置的智能体列表
async function getAgentsByLLMConfig(llmConfigId: number): Promise<AgentInfo[]> {
  try {
    const res = await getAgentList({
      page: 1,
      page_size: 1000, // 通常不会太多
      llm_config_id: llmConfigId
    })
    return res.agents || []
  } catch (error: any) {
    console.error('加载智能体列表失败:', error)
    return []
  }
}

// 为所有LLM加载使用它们的智能体列表
async function loadAgentsForLLMs() {
  agentsByLLM.value.clear()
  // 并行加载所有LLM的智能体列表
  const promises = tableData.value.map(async (llm) => {
    const agents = await getAgentsByLLMConfig(llm.id)
    agentsByLLM.value.set(llm.id, agents)
  })
  await Promise.all(promises)
}

// 分页变化
function handleSizeChange() {
  loadData()
}

function handlePageChange() {
  loadData()
}

// 新增
function handleCreate() {
  dialogTitle.value = '新增 LLM 配置'
  resetForm()
  dialogVisible.value = true
}

// 编辑
function handleEdit(row: LLMInfo) {
  // 检查权限：只有管理员可以编辑
  if (!row.is_admin) {
    ElMessage.warning('无权限：只有管理员可以编辑此资源')
    return
  }
  
  dialogTitle.value = '编辑 LLM 配置'
  formData.id = row.id
  formData.name = row.name
  formData.provider = row.provider
  formData.model = row.model
  formData.api_key = '' // 不显示 API Key
  formData.api_base = row.api_base
  formData.timeout = row.timeout
  formData.max_tokens = row.max_tokens
  formData.extra_config = row.extra_config || ''
  formData.is_default = row.is_default
  formData.visibility = row.visibility ?? 0
  formData.admin = row.admin || ''
  dialogVisible.value = true
}

// 删除
async function handleDelete(row: LLMInfo) {
  // 检查权限：只有管理员可以删除
  if (!row.is_admin) {
    ElMessage.warning('无权限：只有管理员可以删除此资源')
    return
  }
  
  try {
    await ElMessageBox.confirm(`确定要删除 LLM 配置"${row.name}"吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteLLM({ id: row.id })
    ElMessage.success('删除成功')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

// 详情
function handleDetail(row: LLMInfo) {
  // TODO: 实现详情查看
  ElMessage.info('详情功能待实现')
}

// 设为默认
async function handleSetDefault(row: LLMInfo) {
  // 检查权限：只有管理员可以设置默认
  if (!row.is_admin) {
    ElMessage.warning('无权限：只有管理员可以设置默认配置')
    return
  }
  
  try {
    await ElMessageBox.confirm(`确定要将"${row.name}"设为默认 LLM 配置吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await setDefaultLLM({ id: row.id })
    ElMessage.success('设置成功')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '设置失败')
    }
  }
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    submitting.value = true

    if (formData.id) {
      // 更新
      const updateData: LLMUpdateReq = {
        id: formData.id,
        name: formData.name,
        provider: formData.provider,
        model: formData.model,
        api_key: formData.api_key,
        api_base: formData.api_base,
        timeout: formData.timeout,
        max_tokens: formData.max_tokens,
        extra_config: formData.extra_config,
        is_default: formData.is_default,
        visibility: formData.visibility ?? 0,
        admin: formData.admin || ''
      }
      await updateLLM(updateData)
      ElMessage.success('更新成功')
      dialogVisible.value = false
      loadData()
    } else {
      // 创建
      const createData: LLMCreateReq = {
        name: formData.name,
        provider: formData.provider,
        model: formData.model,
        api_key: formData.api_key,
        api_base: formData.api_base,
        timeout: formData.timeout,
        max_tokens: formData.max_tokens,
        extra_config: formData.extra_config,
        is_default: formData.is_default,
        visibility: formData.visibility ?? 0,
        admin: formData.admin || ''
      }
      await createLLM(createData)
      ElMessage.success('创建成功')
      dialogVisible.value = false
      loadData()
    }
  } catch (error: any) {
    if (error !== false) {
      ElMessage.error(error.message || '操作失败')
    }
  } finally {
    submitting.value = false
  }
}

// 重置表单
function resetForm() {
  formData.id = undefined
  formData.name = ''
  formData.provider = ''
  formData.model = ''
  formData.api_key = ''
  formData.api_base = ''
  formData.timeout = 120
  formData.max_tokens = 4000
  formData.extra_config = ''
  formData.is_default = false
  formData.visibility = 0
  formData.admin = ''
  formRef.value?.clearValidate()
}

// 对话框关闭
function handleDialogClose() {
  resetForm()
}

// 返回
function handleBack() {
  router.back()
}

// 初始化
onMounted(() => {
  loadData()
})
</script>

<style scoped>
.llm-management {
  padding: 20px;
}

.management-card {
  min-height: calc(100vh - 100px);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.card-header h2 {
  flex: 1;
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.scope-tabs {
  margin-bottom: 20px;
}

.back-button {
  padding: 0;
}

.stats-section {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.cards-container {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  gap: 20px;
  margin-bottom: 20px;
}

.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>

