<template>
  <div class="knowledge-management">
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
          <h2>知识库管理</h2>
          <el-button type="primary" :icon="Plus" @click="handleCreate">
            新增知识库
          </el-button>
        </div>
      </template>

      <!-- 标签页：我的/市场 -->
      <el-tabs v-model="activeTab" @tab-change="handleTabChange" class="scope-tabs">
        <el-tab-pane label="我的知识库" name="mine" />
        <el-tab-pane label="知识库市场" name="market" />
      </el-tabs>
      <el-divider />

      <!-- 统计卡片区 -->
      <div class="stats-section">
        <StatCard
          :icon="Document"
          label="总数"
          :value="stats.total"
          icon-color="var(--el-color-primary)"
        />
        <StatCard
          :icon="CircleCheck"
          label="激活"
          :value="stats.active"
          icon-color="var(--el-color-success)"
        />
        <StatCard
          :icon="Document"
          label="文档总数"
          :value="stats.totalDocuments"
          icon-color="var(--el-color-info)"
        />
      </div>

      <!-- 卡片列表 -->
      <div v-loading="loading" class="cards-container">
        <KnowledgeCard
          v-for="knowledge in tableData"
          :key="knowledge.id"
          :knowledge="knowledge"
          :agents="agentsByKnowledge.get(knowledge.id) || []"
          @enter="handleViewDetail"
          @edit="handleEdit"
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
          <el-input v-model="formData.name" placeholder="请输入知识库名称" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="formData.status" placeholder="请选择状态" style="width: 100%">
            <el-option label="激活" value="active" />
            <el-option label="停用" value="inactive" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="formData.description"
            type="textarea"
            :rows="3"
            placeholder="请输入描述"
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
import { ArrowLeft, Plus, Document, CircleCheck } from '@element-plus/icons-vue'
import StatCard from '@/components/Agent/StatCard.vue'
import KnowledgeCard from '@/components/Agent/KnowledgeCard.vue'
import {
  getKnowledgeList,
  createKnowledge,
  updateKnowledge,
  deleteKnowledge,
  getAgentList,
  type KnowledgeInfo,
  type KnowledgeListReq,
  type KnowledgeCreateReq,
  type KnowledgeUpdateReq,
  type AgentInfo
} from '@/api/agent'
import type { FormRules } from 'element-plus'

const router = useRouter()

// 表格数据
const loading = ref(false)
const tableData = ref<KnowledgeInfo[]>([])

// 每个知识库对应的智能体列表（key: knowledge_id, value: AgentInfo[]）
const agentsByKnowledge = ref<Map<number, AgentInfo[]>>(new Map())

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
  const active = tableData.value.filter(kb => kb.status === 'active').length
  const totalDocuments = tableData.value.reduce((sum, kb) => sum + (kb.document_count || 0), 0)
  return { total, active, totalDocuments }
})

// 对话框
const dialogVisible = ref(false)
const dialogTitle = ref('新增知识库')
const formRef = ref<InstanceType<typeof ElForm>>()
const submitting = ref(false)

// 表单数据
const formData = reactive<KnowledgeCreateReq & { id?: number }>({
  name: '',
  description: '',
  status: 'active',
  visibility: 0, // 默认公开
  admin: '' // 默认空，后端会自动设置为创建用户
})


// 表单验证规则
const rules: FormRules = {
  name: [{ required: true, message: '请输入知识库名称', trigger: 'blur' }]
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
    const params: KnowledgeListReq = {
      page: pagination.page,
      page_size: pagination.page_size,
      scope: activeTab.value // 添加 scope 参数
    }
    const res = await getKnowledgeList(params)
    // 响应拦截器已经返回了 data，所以 res 就是 { knowledge_bases: [], total: 0 }
    tableData.value = res.knowledge_bases || []
    pagination.total = res.total || 0
    
    // 为每个知识库加载使用它的智能体列表
    await loadAgentsForKnowledgeBases()
  } catch (error: any) {
    ElMessage.error(error.message || '获取列表失败')
  } finally {
    loading.value = false
  }
}

// 获取使用指定知识库的智能体列表
async function getAgentsByKnowledgeBase(knowledgeBaseId: number): Promise<AgentInfo[]> {
  try {
    const res = await getAgentList({
      page: 1,
      page_size: 1000, // 通常不会太多
      knowledge_base_id: knowledgeBaseId
    })
    return res.agents || []
  } catch (error: any) {
    console.error('加载智能体列表失败:', error)
    return []
  }
}

// 为所有知识库加载使用它们的智能体列表
async function loadAgentsForKnowledgeBases() {
  agentsByKnowledge.value.clear()
  // 并行加载所有知识库的智能体列表
  const promises = tableData.value.map(async (knowledge) => {
    const agents = await getAgentsByKnowledgeBase(knowledge.id)
    agentsByKnowledge.value.set(knowledge.id, agents)
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
  dialogTitle.value = '新增知识库'
  resetForm()
  dialogVisible.value = true
}

// 编辑
function handleEdit(row: KnowledgeInfo) {
  // 检查权限：只有管理员可以编辑
  if (!row.is_admin) {
    ElMessage.warning('无权限：只有管理员可以编辑此资源')
    return
  }
  
  dialogTitle.value = '编辑知识库'
  formData.id = row.id
  formData.name = row.name
  formData.description = row.description || ''
  formData.status = row.status || 'active'
  formData.visibility = row.visibility ?? 0
  formData.admin = row.admin || ''
  dialogVisible.value = true
}

// 删除
async function handleDelete(row: KnowledgeInfo) {
  // 检查权限：只有管理员可以删除
  if (!row.is_admin) {
    ElMessage.warning('无权限：只有管理员可以删除此资源')
    return
  }
  
  try {
    await ElMessageBox.confirm(`确定要删除知识库"${row.name}"吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteKnowledge({ id: row.id })
    ElMessage.success('删除成功')
    loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

// 查看详情（跳转到详情页面）
function handleViewDetail(row: KnowledgeInfo) {
  router.push(`/agent/knowledge/${row.id}`)
}


// 提交表单
async function handleSubmit() {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    submitting.value = true

    if (formData.id) {
      // 更新
      const updateData: KnowledgeUpdateReq = {
        id: formData.id,
        name: formData.name,
        description: formData.description,
        status: formData.status,
        visibility: formData.visibility ?? 0,
        admin: formData.admin || ''
      }
      await updateKnowledge(updateData)
      ElMessage.success('更新成功')
      dialogVisible.value = false
      loadData()
    } else {
      // 创建
      const createData: KnowledgeCreateReq = {
        name: formData.name,
        description: formData.description,
        status: formData.status,
        visibility: formData.visibility ?? 0,
        admin: formData.admin || ''
      }
      await createKnowledge(createData)
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
  formData.description = ''
  formData.status = 'active'
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
.knowledge-management {
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

