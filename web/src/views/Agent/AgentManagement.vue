<template>
  <div class="agent-management">
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
          <h2>智能体管理</h2>
          <el-button type="primary" :icon="Plus" @click="handleCreate">
            新增智能体
          </el-button>
        </div>
      </template>

      <!-- 标签页：我的/市场 -->
      <el-tabs v-model="activeTab" @tab-change="handleTabChange" class="scope-tabs">
        <el-tab-pane label="我的智能体" name="mine" />
        <el-tab-pane label="智能体市场" name="market" />
      </el-tabs>
      <el-divider />

      <!-- 筛选条件 -->
      <div class="filter-section">
        <el-form :inline="true" :model="filterForm" class="filter-form">
          <el-form-item label="智能体类型">
            <el-select
              v-model="filterForm.agent_type"
              placeholder="全部类型"
              clearable
              style="width: 150px"
            >
              <el-option label="纯知识库类型" value="knowledge_only" />
              <el-option label="插件类型" value="plugin" />
            </el-select>
          </el-form-item>
          <el-form-item label="启用状态">
            <el-select
              v-model="filterForm.enabled"
              placeholder="全部状态"
              clearable
              style="width: 120px"
            >
              <el-option label="已启用" :value="true" />
              <el-option label="已禁用" :value="false" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :icon="Search" @click="handleSearch">
              查询
            </el-button>
            <el-button :icon="Refresh" @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 统计卡片区 -->
      <div class="stats-section">
        <StatCard
          :icon="Operation"
          label="总数"
          :value="stats.total"
          icon-color="var(--el-color-primary)"
        />
        <StatCard
          :icon="CircleCheck"
          label="已启用"
          :value="stats.enabled"
          icon-color="var(--el-color-success)"
        />
        <StatCard
          :icon="Document"
          label="知识库类型"
          :value="stats.knowledgeOnly"
          icon-color="var(--el-color-info)"
        />
        <StatCard
          :icon="Connection"
          label="插件类型"
          :value="stats.plugin"
          icon-color="var(--el-color-warning)"
        />
      </div>

      <!-- 卡片列表 -->
      <div v-loading="loading" class="cards-container">
        <AgentCard
          v-for="agent in tableData"
          :key="agent.id"
          :agent="agent"
          @detail="handleDetail"
          @edit="handleEdit"
          @toggle="handleToggle"
          @delete="handleDelete"
          @copy="handleCopy"
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

    <!-- 详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="智能体详情"
      width="800px"
      :close-on-click-modal="false"
    >
      <el-descriptions :column="2" border v-if="detailData">
        <el-descriptions-item label="ID">{{ detailData.id }}</el-descriptions-item>
        <el-descriptions-item label="名称">{{ detailData.name }}</el-descriptions-item>
        <el-descriptions-item label="智能体类型">
          <el-tag v-if="detailData.agent_type === 'knowledge_only'" type="success">纯知识库</el-tag>
          <el-tag v-else-if="detailData.agent_type === 'plugin'" type="warning">插件类型</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="聊天类型">{{ detailData.chat_type }}</el-descriptions-item>
        <el-descriptions-item label="启用状态">
          <el-tag :type="detailData.enabled ? 'success' : 'danger'">
            {{ detailData.enabled ? '已启用' : '已禁用' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="超时时间">{{ detailData.timeout }} 秒</el-descriptions-item>
        <el-descriptions-item label="描述" :span="2">{{ detailData.description || '-' }}</el-descriptions-item>
        <el-descriptions-item label="文档路径" :span="2">
          <div v-if="detailData.docs_paths && detailData.docs_paths.trim()">
            <el-tag
              v-for="(path, index) in detailData.docs_paths.split(',').filter(p => p.trim())"
              :key="index"
              size="small"
              style="margin-right: 8px; margin-bottom: 4px;"
            >
              {{ path.trim() }}
            </el-tag>
          </div>
          <span v-else style="color: #909399;">未配置</span>
        </el-descriptions-item>
        <el-descriptions-item label="LLM 配置" :span="2">
          <span v-if="detailData.llm_config">
            {{ detailData.llm_config.name }} ({{ detailData.llm_config.provider }}/{{ detailData.llm_config.model }})
            <el-tag v-if="detailData.llm_config.is_default" size="small" type="success" style="margin-left: 8px;">默认</el-tag>
          </span>
          <span v-else-if="detailData.llm_config_id === 0">使用默认 LLM</span>
          <span v-else>ID: {{ detailData.llm_config_id }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="系统提示词模板" :span="2">
          <el-input
            :value="detailData.system_prompt_template || '未设置，使用默认模板'"
            type="textarea"
            :rows="4"
            readonly
          />
        </el-descriptions-item>
        <el-descriptions-item label="可见性">
          <el-tag :type="detailData.visibility === 0 ? 'success' : 'info'">
            {{ detailData.visibility === 0 ? '公开' : '私有' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="管理员">{{ detailData.admin || '-' }}</el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ detailData.created_at }}</el-descriptions-item>
        <el-descriptions-item label="更新时间">{{ detailData.updated_at }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="800px"
      :close-on-click-modal="false"
      @close="handleDialogClose"
      @opened="handleDialogOpened"
    >
      <el-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-width="120px"
      >
        <el-form-item label="名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入智能体名称" />
        </el-form-item>
        <el-form-item label="智能体类型" prop="agent_type">
          <el-select
            v-model="formData.agent_type"
            placeholder="请选择智能体类型"
            style="width: 100%"
            @change="handleAgentTypeChange"
          >
            <el-option label="纯知识库类型" value="knowledge_only" />
            <el-option label="插件调用类型" value="plugin" />
          </el-select>
        </el-form-item>
        <el-form-item
          v-if="formData.agent_type === 'plugin'"
          label="插件函数路径"
          prop="plugin_function_path"
        >
          <FunctionPathSelector
            v-model="formData.plugin_function_path"
            user="system"
            app="official"
            template-type="form"
          />
          <div style="margin-top: 8px; font-size: 12px; color: #909399;">
            提示：插件类型智能体必须指定一个插件函数路径（full-code-path），支持搜索
          </div>
        </el-form-item>
        <el-form-item label="LLM 配置">
          <el-select
            v-model="formData.llm_config_id"
            filterable
            :loading="llmLoading"
            placeholder="选择 LLM 配置（留空则使用默认 LLM）"
            style="width: 100%"
            clearable
            @focus="handleLLMSelectFocus"
          >
            <el-option
              v-for="llm in llmOptions"
              :key="llm.id"
              :label="`${llm.name} (${llm.provider}/${llm.model})`"
              :value="llm.id"
            >
              <div style="display: flex; justify-content: space-between; align-items: center;">
                <span>{{ llm.name }}</span>
                <el-tag size="small" :type="llm.is_default ? 'success' : 'info'" style="margin-left: 8px;">
                  {{ llm.provider }}/{{ llm.model }}{{ llm.is_default ? ' (默认)' : '' }}
                </el-tag>
              </div>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="文档路径">
          <DocsPathSelector
            v-model="formData.docs_paths"
          />
          <div style="margin-top: 8px; font-size: 12px; color: #909399;">
            提示：可以手动输入文档路径（逗号分隔），如：/system/official/sdk,/system/official/plugins。也可以点击按钮选择服务树中的路径（需要指定应用）。
          </div>
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="formData.description"
            type="textarea"
            :rows="3"
            placeholder="请输入描述"
          />
        </el-form-item>
        <el-form-item label="系统提示词模板">
          <el-input
            v-model="formData.system_prompt_template"
            type="textarea"
            :rows="5"
            placeholder="请输入系统提示词模板，支持 {knowledge} 变量，例如：你是一个专业的代码生成助手。以下是相关的知识库内容，请参考这些内容来生成代码：\n{knowledge}"
          />
        </el-form-item>
        <el-form-item label="开场白">
          <RichTextEditor
            v-model="formData.greeting"
            placeholder="请输入开场白内容，用于显示智能体的使用教程等（可选）"
          />
          <div style="margin-top: 8px; font-size: 12px; color: #909399;">
            提示：开场白会在用户选择智能体时显示，使用富文本编辑器编辑，自动保存为 HTML 格式
          </div>
        </el-form-item>
        <el-form-item label="超时时间（秒）">
          <el-input-number
            v-model="formData.timeout"
            :min="1"
            :max="300"
            placeholder="默认30秒"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="元数据（JSON）">
          <el-input
            v-model="formData.metadata"
            type="textarea"
            :rows="3"
            placeholder='请输入JSON格式的元数据，如：{"category": "office"}'
          />
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
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElForm } from 'element-plus'
import { ArrowLeft, Plus, Search, Refresh, DocumentCopy, Operation, CircleCheck, Document, Connection } from '@element-plus/icons-vue'
import StatCard from '@/components/Agent/StatCard.vue'
import AgentCard from '@/components/Agent/AgentCard.vue'
import RichTextEditor from '@/components/RichTextEditor.vue'
import DocsPathSelector from '@/components/DocsPathSelector.vue'
import FunctionPathSelector from '@/components/FunctionPathSelector.vue'
import {
  getAgentList,
  getAgent,
  createAgent,
  updateAgent,
  deleteAgent,
  enableAgent,
  disableAgent,
  getLLMList,
  type AgentInfo,
  type AgentListReq,
  type AgentCreateReq,
  type AgentUpdateReq,
  type LLMInfo,
} from '@/api/agent'
import type { FormRules } from 'element-plus'

const router = useRouter()

// 表格数据
const loading = ref(false)
const tableData = ref<AgentInfo[]>([])

// 分页
const pagination = reactive({
  page: 1,
  page_size: 10,
  total: 0
})

// 标签页
const activeTab = ref<'mine' | 'market'>('mine')

// 筛选条件
const filterForm = reactive<{
  agent_type?: 'knowledge_only' | 'plugin'
  enabled?: boolean
}>({})

// 统计信息
const stats = computed(() => {
  const total = tableData.value.length
  const enabled = tableData.value.filter(a => a.enabled).length
  const knowledgeOnly = tableData.value.filter(a => a.agent_type === 'knowledge_only').length
  const plugin = tableData.value.filter(a => a.agent_type === 'plugin').length
  return { total, enabled, knowledgeOnly, plugin }
})

// 对话框
const dialogVisible = ref(false)
const dialogTitle = ref('新增智能体')
const formRef = ref<InstanceType<typeof ElForm>>()
const submitting = ref(false)

// 详情对话框
const detailDialogVisible = ref(false)
const detailData = ref<AgentInfo | null>(null)

// 表单数据
const formData = reactive<AgentCreateReq & { id?: number }>({
  name: '',
  agent_type: 'knowledge_only',
  chat_type: 'function_gen', // 默认值
  description: '',
  timeout: 30,
  plugin_function_path: '', // 插件函数路径（仅 plugin 类型需要）
  docs_paths: '', // 文档路径（逗号分隔）
  llm_config_id: 0, // 0 表示使用默认 LLM
  metadata: '',
  visibility: 0, // 默认公开
  admin: '' // 默认空，后端会自动设置为创建用户
})

// LLM 配置
const llmOptions = ref<LLMInfo[]>([])
const llmLoading = ref(false)


// 表单验证规则
const rules: FormRules = {
  name: [{ required: true, message: '请输入智能体名称', trigger: 'blur' }],
  agent_type: [{ required: true, message: '请选择智能体类型', trigger: 'change' }],
  plugin_function_path: [
    {
      validator: (rule: any, value: any, callback: any) => {
        if (formData.agent_type === 'plugin' && (!value || value.trim() === '')) {
          callback(new Error('插件类型智能体必须指定插件函数路径'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ],
}


// 智能体类型变化时的处理
function handleAgentTypeChange() {
  // 如果切换到非 plugin 类型，清空 plugin_function_path
  if (formData.agent_type !== 'plugin') {
    formData.plugin_function_path = ''
  }
}

// 标签页切换处理
function handleTabChange(tabName: string) {
  activeTab.value = tabName as 'mine' | 'market'
  pagination.page = 1 // 切换标签页时重置页码
  loadData()
}

// 加载数据（同时提取知识库和 LLM 选项）
async function loadData() {
  loading.value = true
  try {
    const params: AgentListReq = {
      page: pagination.page,
      page_size: pagination.page_size,
      scope: activeTab.value, // 添加 scope 参数
      ...filterForm
    }
    const res = await getAgentList(params)
    // 响应拦截器已经返回了 data，所以 res 就是 { agents: [], total: 0 }
    tableData.value = res.agents || []
    pagination.total = res.total || 0
    
    // 🔥 从智能体列表中提取 LLM 选项（去重）
    const llmMap = new Map<number, LLMInfo>()
    
    res.agents?.forEach(agent => {
      // 提取 LLM 配置信息
      if (agent.llm_config && !llmMap.has(agent.llm_config.id)) {
        llmMap.set(agent.llm_config.id, {
          id: agent.llm_config.id,
          name: agent.llm_config.name,
          provider: agent.llm_config.provider,
          model: agent.llm_config.model,
          api_base: '',
          timeout: 0,
          max_tokens: 0,
          extra_config: '',
          is_default: agent.llm_config.is_default,
          created_at: '',
          updated_at: ''
        })
      }
    })
    
    // 更新选项列表（合并，不覆盖已有数据）
    llmOptions.value = Array.from(llmMap.values())
  } catch (error: any) {
    ElMessage.error(error.message || '获取列表失败')
  } finally {
    loading.value = false
  }
}

// 搜索
function handleSearch() {
  pagination.page = 1
  loadData()
}

// 重置
function handleReset() {
  filterForm.agent_type = undefined
  filterForm.enabled = undefined
  pagination.page = 1
  loadData()
}

// 分页变化
function handleSizeChange() {
  loadData()
}

function handlePageChange() {
  loadData()
}

// 对话框打开时（确保 LLM 选项已加载）
async function handleDialogOpened() {
  // 🔥 强制重新加载，确保数据是最新的
  await loadAllLLMs()
}

// 加载所有 LLM 配置（合并到现有列表，不去重覆盖）
async function loadAllLLMs() {
  llmLoading.value = true
  try {
    const res = await getLLMList({
      page: 1,
      page_size: 1000 // 加载所有
    })
    const newLLMs = res.configs || []
    // 合并到现有列表，避免重复
    const llmMap = new Map<number, LLMInfo>()
    llmOptions.value.forEach(llm => llmMap.set(llm.id, llm))
    newLLMs.forEach(llm => {
      if (!llmMap.has(llm.id)) {
        llmMap.set(llm.id, llm)
      }
    })
    llmOptions.value = Array.from(llmMap.values())
  } catch (error: any) {
    console.error('加载 LLM 配置失败:', error)
    ElMessage.error(error.message || '加载 LLM 配置失败，请稍后重试')
  } finally {
    llmLoading.value = false
  }
}

// LLM 选择框获得焦点时（确保数据已加载）
async function handleLLMSelectFocus() {
  // 如果 LLM 选项为空，加载所有 LLM 配置
  if (llmOptions.value.length === 0) {
    await loadAllLLMs()
  }
}


// 详情
async function handleDetail(row: AgentInfo) {
  try {
    // 调用详情 API 获取完整数据（包括 msg_subject）
    const res = await getAgent({ id: row.id })
    // 响应拦截器已经返回了 data，所以 res 就是 AgentInfo
    console.log('[AgentManagement] 获取详情响应:', res)
    console.log('[AgentManagement] msg_subject:', res.msg_subject)
    detailData.value = res
    detailDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(error.message || '获取详情失败')
  }
}

// 复制主题地址
async function handleCopyMsgSubject(msgSubject: string) {
  try {
    await navigator.clipboard.writeText(msgSubject)
    ElMessage.success('已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败')
  }
}

// 新增
async function handleCreate() {
  dialogTitle.value = '新增智能体'
  resetForm()
  dialogVisible.value = true
}

// 编辑
async function handleEdit(row: AgentInfo) {
  // 检查权限：只有管理员可以编辑
  if (!row.is_admin) {
    ElMessage.warning('无权限：只有管理员可以编辑此资源')
    return
  }
  
  dialogTitle.value = '编辑智能体'
  formData.id = row.id
  formData.name = row.name
  formData.agent_type = row.agent_type
  formData.chat_type = row.chat_type || 'function_gen'
  formData.description = row.description
  formData.system_prompt_template = row.system_prompt_template || ''
  formData.timeout = row.timeout
  formData.plugin_function_path = row.plugin_function_path || ''
  formData.docs_paths = row.docs_paths
  formData.llm_config_id = row.llm_config_id || 0
  formData.metadata = row.metadata || ''
  formData.greeting = row.greeting || ''
  formData.greeting_type = (row.greeting_type as 'text' | 'md' | 'html') || 'text'
  formData.visibility = row.visibility ?? 0
  formData.admin = row.admin || ''
  
  dialogVisible.value = true
}

// 复制
async function handleCopy(row: AgentInfo) {
  dialogTitle.value = '复制智能体'
  // 清空ID，表示新增
  formData.id = undefined
  // 复制数据，名称添加"副本"后缀
  formData.name = `${row.name} 副本`
  formData.agent_type = row.agent_type
  formData.chat_type = row.chat_type || 'function_gen'
  formData.description = row.description
  formData.system_prompt_template = row.system_prompt_template || ''
  formData.timeout = row.timeout
  formData.plugin_function_path = row.plugin_function_path || ''
  formData.docs_paths = row.docs_paths
  formData.llm_config_id = row.llm_config_id || 0
  formData.metadata = row.metadata || ''
  formData.greeting = row.greeting || ''
  formData.greeting_type = (row.greeting_type as 'text' | 'md' | 'html') || 'text'
  formData.visibility = row.visibility ?? 0
  formData.admin = row.admin || ''
  
  dialogVisible.value = true
}

// 删除
async function handleDelete(row: AgentInfo) {
  // 检查权限：只有管理员可以删除
  if (!row.is_admin) {
    ElMessage.warning('无权限：只有管理员可以删除此资源')
    return
  }
  
  try {
    await ElMessageBox.confirm(`确定要删除智能体"${row.name}"吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteAgent({ id: row.id })
    ElMessage.success('删除成功')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

// 切换启用/禁用状态
async function handleToggle(row: AgentInfo) {
  // 检查权限：只有管理员可以启用/禁用
  if (!row.is_admin) {
    ElMessage.warning('无权限：只有管理员可以启用/禁用此资源')
    return
  }
  
  try {
    if (row.enabled) {
      // 禁用
      await ElMessageBox.confirm(`确定要禁用智能体"${row.name}"吗？`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })
      await disableAgent({ id: row.id })
      ElMessage.success('禁用成功')
    } else {
      // 启用
      await enableAgent({ id: row.id })
      ElMessage.success('启用成功')
    }
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '操作失败')
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
      const updateData: AgentUpdateReq = {
        id: formData.id,
        name: formData.name,
        agent_type: formData.agent_type,
        chat_type: formData.chat_type || 'function_gen',
        description: formData.description,
        system_prompt_template: formData.system_prompt_template || '',
        timeout: formData.timeout,
        plugin_function_path: formData.agent_type === 'plugin' ? formData.plugin_function_path : undefined,
        docs_paths: formData.docs_paths,
        llm_config_id: formData.llm_config_id || 0,
        metadata: formData.metadata,
        greeting: formData.greeting || '',
        greeting_type: formData.greeting ? 'html' : 'text', // 使用富文本编辑器时，自动设置为 html
        visibility: formData.visibility ?? 0,
        admin: formData.admin || ''
      }
      await updateAgent(updateData)
      ElMessage.success('更新成功')
      dialogVisible.value = false
      loadData()
    } else {
      // 创建
      const createData: AgentCreateReq = {
        name: formData.name,
        agent_type: formData.agent_type,
        chat_type: formData.chat_type || 'function_gen',
        description: formData.description,
        system_prompt_template: formData.system_prompt_template || '',
        timeout: formData.timeout,
        plugin_function_path: formData.agent_type === 'plugin' ? formData.plugin_function_path : undefined,
        docs_paths: formData.docs_paths,
        llm_config_id: formData.llm_config_id || 0,
        metadata: formData.metadata,
        greeting: formData.greeting || '',
        greeting_type: formData.greeting ? 'html' : 'text', // 使用富文本编辑器时，自动设置为 html
        visibility: formData.visibility ?? 0,
        admin: formData.admin || ''
      }
      await createAgent(createData)
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
  formData.agent_type = 'knowledge_only'
  formData.chat_type = 'function_gen'
  formData.description = ''
  formData.system_prompt_template = ''
  formData.timeout = 30
  formData.plugin_id = null
  formData.docs_paths = ''
  formData.llm_config_id = 0
  formData.metadata = ''
  formData.greeting = ''
  formData.greeting_type = 'text'
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
onMounted(async () => {
  // 🔥 并行加载：智能体列表、知识库列表和 LLM 配置列表
  // 这样可以确保即使智能体列表为空，也能有选项可以选择
  await Promise.all([
    loadData(),
    loadAllLLMs()
  ])
})
</script>

<style scoped>
.agent-management {
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

.back-button {
  padding: 0;
}

.scope-tabs {
  margin-bottom: 20px;
}

.filter-section {
  margin-bottom: 20px;
}

.filter-form {
  margin: 0;
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

