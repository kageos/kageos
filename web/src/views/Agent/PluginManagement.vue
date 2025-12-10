<template>
  <div class="plugin-management">
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
          <h2>插件管理</h2>
          <el-button type="primary" :icon="Plus" @click="handleCreate">
            新增插件
          </el-button>
        </div>
      </template>

      <!-- 标签页：我的/市场 -->
      <el-tabs v-model="activeTab" @tab-change="handleTabChange" class="scope-tabs">
        <el-tab-pane label="我的插件" name="mine" />
        <el-tab-pane label="插件市场" name="market" />
      </el-tabs>
      <el-divider />

      <!-- 筛选条件 -->
      <div class="filter-section">
        <el-form :inline="true" :model="filterForm" class="filter-form">
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
          :icon="Connection"
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
          :icon="Operation"
          label="使用中"
          :value="stats.inUse"
          icon-color="var(--el-color-warning)"
        />
      </div>

      <!-- 卡片列表 -->
      <div v-loading="loading" class="cards-container">
        <PluginCard
          v-for="plugin in tableData"
          :key="plugin.id"
          :plugin="plugin"
          :agents="agentsByPlugin.get(plugin.id) || []"
          @detail="handleDetail"
          @edit="handleEdit"
          @toggle="handleToggle"
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

    <!-- 详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="插件详情"
      width="800px"
      :close-on-click-modal="false"
    >
      <el-descriptions v-if="detailData" :column="2" border>
        <el-descriptions-item label="ID">{{ detailData.id }}</el-descriptions-item>
        <el-descriptions-item label="名称">{{ detailData.name }}</el-descriptions-item>
        <el-descriptions-item label="代码">{{ detailData.code }}</el-descriptions-item>
        <el-descriptions-item label="创建用户">{{ detailData.user }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag v-if="detailData.enabled" type="success">已启用</el-tag>
          <el-tag v-else type="danger">已禁用</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="可见性">
          <el-tag :type="detailData.visibility === 0 ? 'success' : 'info'">
            {{ detailData.visibility === 0 ? '公开' : '私有' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="管理员">{{ detailData.admin || '-' }}</el-descriptions-item>
        <el-descriptions-item label="NATS 服务器地址" :span="2">
          <el-input
            :value="detailData.nats_host || '未配置'"
            readonly
            style="width: 100%"
          >
            <template #append>
              <el-button
                :icon="DocumentCopy"
                @click="handleCopySubject(detailData.nats_host || '')"
                :disabled="!detailData.nats_host"
              >
                复制
              </el-button>
            </template>
          </el-input>
        </el-descriptions-item>
        <el-descriptions-item label="NATS 主题" :span="2">
          <el-input
            :value="detailData.subject"
            readonly
            style="width: 100%"
          >
            <template #append>
              <el-button
                :icon="DocumentCopy"
                @click="handleCopySubject(detailData.subject)"
              >
                复制
              </el-button>
            </template>
          </el-input>
        </el-descriptions-item>
        <el-descriptions-item label="描述" :span="2">
          <el-input
            :value="detailData.description || '-'"
            type="textarea"
            :rows="3"
            readonly
          />
        </el-descriptions-item>
        <el-descriptions-item label="配置（JSON）" :span="2">
          <el-input
            :value="detailData.config || '{}'"
            type="textarea"
            :rows="4"
            readonly
          />
        </el-descriptions-item>
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
    >
      <el-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-width="120px"
      >
        <el-form-item label="名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入插件名称" />
        </el-form-item>
        <el-form-item label="代码" prop="code">
          <el-input v-model="formData.code" placeholder="请输入插件代码（唯一标识，如：ExcelToMarkdownTable）" />
          <div style="margin-top: 8px; font-size: 12px; color: #909399;">
            提示：插件代码使用驼峰命名（首字母大写），创建后不可修改
          </div>
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="formData.description"
            type="textarea"
            :rows="3"
            placeholder="请输入插件描述"
          />
        </el-form-item>
        <el-form-item label="配置（JSON）">
          <el-input
            v-model="formData.config"
            type="textarea"
            :rows="4"
            placeholder='请输入JSON格式的插件配置，如：{"timeout": 30, "max_file_size": 10485760}'
          />
          <div style="margin-top: 8px; font-size: 12px; color: #909399;">
            提示：插件配置用于存储插件的运行时参数
          </div>
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="formData.enabled" />
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
import { ArrowLeft, Plus, Search, Refresh, DocumentCopy } from '@element-plus/icons-vue'
import {
  getPluginList,
  getPlugin,
  createPlugin,
  updatePlugin,
  deletePlugin,
  enablePlugin,
  disablePlugin,
  getAgentList,
  type PluginInfo,
  type PluginListReq,
  type PluginCreateReq,
  type PluginUpdateReq,
  type AgentInfo
} from '@/api/agent'
import { useAuthStore } from '@/stores/auth'
import type { FormRules } from 'element-plus'

const router = useRouter()
const authStore = useAuthStore()

// 表格数据
const loading = ref(false)
const tableData = ref<PluginInfo[]>([])

// 每个插件对应的智能体列表（key: plugin_id, value: AgentInfo[]）
const agentsByPlugin = ref<Map<number, AgentInfo[]>>(new Map())

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
  const enabled = tableData.value.filter((p: PluginInfo) => p.enabled).length
  // 计算使用中的插件数量（被至少一个智能体使用）
  const inUsePlugins = new Set<number>()
  agentsByPlugin.value.forEach((agents: AgentInfo[], pluginId: number) => {
    if (agents.length > 0) {
      inUsePlugins.add(pluginId)
    }
  })
  const inUse = inUsePlugins.size
  return { total, enabled, inUse }
})

// 筛选条件
const filterForm = reactive<{
  enabled?: boolean
}>({})

// 对话框
const dialogVisible = ref(false)
const dialogTitle = ref('新增插件')
const formRef = ref<InstanceType<typeof ElForm>>()
const submitting = ref(false)

// 详情对话框
const detailDialogVisible = ref(false)
const detailData = ref<PluginInfo | null>(null)

// 表单数据
const formData = reactive<PluginCreateReq & { id?: number }>({
  name: '',
  code: '',
  description: '',
  enabled: true,
  config: null
})

// 表单验证规则
const rules: FormRules = {
  name: [{ required: true, message: '请输入插件名称', trigger: 'blur' }],
  code: [
    { required: true, message: '请输入插件代码', trigger: 'blur' },
    {
      pattern: /^[A-Z][a-zA-Z0-9]*$/,
      message: '插件代码必须使用驼峰命名（首字母大写，如：ExcelToMarkdownTable）',
      trigger: 'blur'
    }
  ]
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
    const params: PluginListReq = {
      page: pagination.page,
      page_size: pagination.page_size,
      scope: activeTab.value, // 添加 scope 参数
      ...filterForm
    }
    const res = await getPluginList(params)
    // 响应拦截器已经返回了 data，所以 res 就是 PluginListResp
    tableData.value = res.plugins || []
    pagination.total = res.total || 0
    
    // 为每个插件加载使用它的智能体列表
    await loadAgentsForPlugins()
  } catch (error: any) {
    ElMessage.error(error.message || '获取列表失败')
  } finally {
    loading.value = false
  }
}

// 获取使用指定插件的智能体列表
async function getAgentsByPlugin(pluginId: number): Promise<AgentInfo[]> {
  try {
    const res = await getAgentList({
      page: 1,
      page_size: 1000, // 通常不会太多
      plugin_id: pluginId
    })
    return res.agents || []
  } catch (error: any) {
    console.error('加载智能体列表失败:', error)
    return []
  }
}

// 为所有插件加载使用它们的智能体列表
async function loadAgentsForPlugins() {
  agentsByPlugin.value.clear()
  // 并行加载所有插件的智能体列表
  const promises = tableData.value.map(async (plugin: PluginInfo) => {
    const agents = await getAgentsByPlugin(plugin.id)
    agentsByPlugin.value.set(plugin.id, agents)
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

// 搜索
function handleSearch() {
  pagination.page = 1
  loadData()
}

// 重置
function handleReset() {
  filterForm.enabled = undefined
  pagination.page = 1
  loadData()
}

// 详情
async function handleDetail(row: PluginInfo) {
  try {
    const res = await getPlugin({ id: row.id })
    // 响应拦截器已经返回了 data，所以 res 就是 PluginInfo
    detailData.value = res
    detailDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(error.message || '获取详情失败')
  }
}

// 复制主题地址
async function handleCopySubject(subject: string) {
  try {
    await navigator.clipboard.writeText(subject)
    ElMessage.success('已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败')
  }
}

// 新增
function handleCreate() {
  dialogTitle.value = '新增插件'
  resetForm()
  dialogVisible.value = true
}

// 编辑
function handleEdit(row: PluginInfo) {
  // 检查权限：只有管理员可以编辑
  if (!row.is_admin) {
    ElMessage.warning('无权限：只有管理员可以编辑此资源')
    return
  }
  
  dialogTitle.value = '编辑插件'
  formData.id = row.id
  formData.name = row.name
  formData.code = row.code
  formData.description = row.description || ''
  formData.enabled = row.enabled
  formData.config = row.config || null
  formData.visibility = row.visibility ?? 0
  formData.admin = row.admin || ''
  dialogVisible.value = true
}

// 删除
async function handleDelete(row: PluginInfo) {
  // 检查权限：只有管理员可以删除
  if (!row.is_admin) {
    ElMessage.warning('无权限：只有管理员可以删除此资源')
    return
  }
  
  try {
    await ElMessageBox.confirm(`确定要删除插件"${row.name}"吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deletePlugin({ id: row.id })
    ElMessage.success('删除成功')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

// 切换启用/禁用状态
async function handleToggle(row: PluginInfo) {
  // 检查权限：只有管理员可以启用/禁用
  if (!row.is_admin) {
    ElMessage.warning('无权限：只有管理员可以启用/禁用此资源')
    return
  }
  
  try {
    if (row.enabled) {
      // 禁用
      await ElMessageBox.confirm(`确定要禁用插件"${row.name}"吗？`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })
      await disablePlugin({ id: row.id })
      ElMessage.success('禁用成功')
    } else {
      // 启用
      await enablePlugin({ id: row.id })
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
      const updateData: PluginUpdateReq = {
        name: formData.name,
        code: formData.code,
        description: formData.description || '',
        enabled: formData.enabled,
        config: formData.config,
        visibility: formData.visibility ?? 0,
        admin: formData.admin || ''
      }
      await updatePlugin(formData.id, updateData)
      ElMessage.success('更新成功')
      dialogVisible.value = false
      loadData()
    } else {
      // 创建
      const createData: PluginCreateReq = {
        name: formData.name,
        code: formData.code,
        description: formData.description || '',
        enabled: formData.enabled,
        config: formData.config,
        visibility: formData.visibility ?? 0,
        admin: formData.admin || ''
      }
      await createPlugin(createData)
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
  formData.code = ''
  formData.description = ''
  formData.enabled = true
  formData.config = null
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
.plugin-management {
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

