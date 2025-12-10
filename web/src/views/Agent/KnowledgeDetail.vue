<template>
  <div class="knowledge-detail-page">
    <div class="knowledge-detail-content" v-loading="loading">
      <!-- 三栏布局 -->
      <div class="three-column-layout">
        <!-- 左侧：文档树 -->
        <div class="left-panel">
          <div class="panel-header">
            <h3>{{ knowledgeInfo?.name || '文档目录' }}</h3>
            <div class="header-actions">
              <el-button 
                type="primary" 
                size="small" 
                :icon="Plus"
                @click="handleAddDocument"
              >
                添加文档
              </el-button>
            </div>
          </div>
          
          <div v-if="treeData.length > 0" class="tree-container">
            <KnowledgeTree
              v-model="selectedNodeId"
              :data="treeData"
              class="knowledge-tree"
              @node-click="handleNodeClick"
              @node-action="handleNodeAction"
            />
          </div>
          
          <!-- 空状态 -->
          <div v-else class="empty-state">
            <el-icon class="empty-icon"><Folder /></el-icon>
            <p>暂无文档</p>
          </div>
        </div>
        
        <!-- 中间：文档内容显示区域 -->
        <div class="middle-panel">
          <div class="panel-header">
            <div class="header-title">
              <el-icon class="page-icon"><Document /></el-icon>
              <h1 class="page-title">文档预览</h1>
            </div>
          </div>

          <!-- 文档内容显示区域 -->
          <div v-if="selectedDocument" class="document-content">
            <div class="content-preview">
              <pre v-if="selectedDocument.file_type === 'txt' || selectedDocument.file_type === 'md'">{{ selectedDocument.content }}</pre>
              <div v-else class="unsupported-format">
                <el-icon><Document /></el-icon>
                <p>此文件类型暂不支持预览</p>
                <p>文件类型：{{ selectedDocument.file_type?.toUpperCase() }}</p>
              </div>
            </div>
          </div>
          
          <!-- 未选择文档时的提示 -->
          <div v-else class="no-selection">
            <el-icon class="no-selection-icon"><Document /></el-icon>
            <h3>选择文档查看内容</h3>
            <p>点击左侧树形目录中的文档来查看内容</p>
          </div>
        </div>
        
        <!-- 右侧：文档信息 -->
        <div class="right-panel">
          <div class="panel-header">
            <h3>文档信息</h3>
          </div>
          
          <div v-if="selectedDocument" class="document-info">
            <div class="info-section">
              <h4>基本信息</h4>
              <div class="info-item">
                <label>文档标题</label>
                <span>{{ selectedDocument.title }}</span>
              </div>
              <div class="info-item">
                <label>文件类型</label>
                <el-tag size="small">{{ selectedDocument.file_type?.toUpperCase() }}</el-tag>
              </div>
              <div class="info-item">
                <label>文件大小</label>
                <span>{{ formatFileSize(selectedDocument.file_size) }}</span>
              </div>
              <div class="info-item">
                <label>状态</label>
                <el-tag 
                  :type="selectedDocument.status === 'completed' ? 'success' : selectedDocument.status === 'failed' ? 'danger' : 'info'"
                  size="small"
                >
                  {{ getStatusText(selectedDocument.status) }}
                </el-tag>
              </div>
              <div class="info-item">
                <label>创建时间</label>
                <span>{{ formatTime(selectedDocument.created_at) }}</span>
              </div>
              <div class="info-item">
                <label>更新时间</label>
                <span>{{ formatTime(selectedDocument.updated_at) }}</span>
              </div>
              <div class="info-item">
                <label>创建用户</label>
                <span>{{ selectedDocument.user }}</span>
              </div>
            </div>
            
            <div class="action-section">
              <h4>操作</h4>
              <div class="action-buttons">
                <el-button 
                  type="primary" 
                  @click="handleEditDocument(selectedDocument)"
                  :disabled="selectedDocument.status !== 'completed'"
                  block
                >
                  <el-icon><Edit /></el-icon>
                  编辑文档
                </el-button>
                <el-button 
                  type="danger" 
                  @click="handleDeleteDocument(selectedDocument)"
                  block
                >
                  <el-icon><Delete /></el-icon>
                  删除文档
                </el-button>
              </div>
            </div>
          </div>
          
          <!-- 未选择文档时的提示 -->
          <div v-else class="no-selection">
            <el-icon class="no-selection-icon"><InfoFilled /></el-icon>
            <h3>选择文档查看信息</h3>
            <p>点击左侧树形目录中的文档来查看详细信息</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加文档对话框 -->
    <el-dialog
      v-model="addDocDialogVisible"
      title="添加文档"
      width="800px"
      :close-on-click-modal="false"
      @close="handleAddDocDialogClose"
    >
      <el-form
        ref="docFormRef"
        :model="docFormData"
        :rules="docRules"
        label-width="120px"
      >
        <el-form-item label="标题" prop="title">
          <el-input v-model="docFormData.title" placeholder="请输入文档标题" />
        </el-form-item>
        <el-form-item label="内容" prop="content">
          <el-input
            v-model="docFormData.content"
            type="textarea"
            :rows="10"
            placeholder="请输入文档内容（目录可以为空）"
          />
        </el-form-item>
        <el-form-item label="文件类型">
          <el-select v-model="docFormData.file_type" placeholder="请选择文件类型" style="width: 100%">
            <el-option label="PDF" value="pdf" />
            <el-option label="TXT" value="txt" />
            <el-option label="DOC" value="doc" />
            <el-option label="DOCX" value="docx" />
            <el-option label="MD" value="md" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addDocDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="docSubmitting" @click="handleSubmitDocument">
          确定
        </el-button>
      </template>
    </el-dialog>

    <!-- 查看/编辑文档对话框 -->
    <el-dialog
      v-model="viewDocDialogVisible"
      :title="viewDocDialogTitle"
      width="900px"
      :close-on-click-modal="false"
      @close="handleViewDocDialogClose"
    >
      <el-form
        v-if="currentDocument"
        ref="viewDocFormRef"
        :model="viewDocFormData"
        :rules="viewDocRules"
        label-width="120px"
      >
        <el-form-item label="标题" prop="title">
          <el-input
            v-model="viewDocFormData.title"
            :disabled="!isEditingDocument"
            placeholder="请输入文档标题"
          />
        </el-form-item>
        <el-form-item label="内容" prop="content">
          <el-input
            v-model="viewDocFormData.content"
            type="textarea"
            :rows="15"
            :disabled="!isEditingDocument"
            placeholder="请输入文档内容"
          />
        </el-form-item>
        <el-form-item label="文件类型">
          <el-select
            v-model="viewDocFormData.file_type"
            :disabled="!isEditingDocument"
            placeholder="请选择文件类型"
            style="width: 100%"
          >
            <el-option label="PDF" value="pdf" />
            <el-option label="TXT" value="txt" />
            <el-option label="DOC" value="doc" />
            <el-option label="DOCX" value="docx" />
            <el-option label="MD" value="md" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select
            v-model="viewDocFormData.status"
            :disabled="!isEditingDocument"
            placeholder="请选择状态"
            style="width: 100%"
          >
            <el-option label="已完成" value="completed" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-form-item>
        <el-descriptions :column="2" border v-if="!isEditingDocument">
          <el-descriptions-item label="文档ID">{{ currentDocument.doc_id }}</el-descriptions-item>
          <el-descriptions-item label="文件大小">{{ formatFileSize(currentDocument.file_size) }}</el-descriptions-item>
          <el-descriptions-item label="创建用户">{{ currentDocument.user }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTime(currentDocument.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ formatTime(currentDocument.updated_at) }}</el-descriptions-item>
        </el-descriptions>
      </el-form>
      <template #footer>
        <el-button @click="viewDocDialogVisible = false">关闭</el-button>
        <el-button
          v-if="!isEditingDocument"
          type="primary"
          @click="handleStartEditDocument"
        >
          编辑
        </el-button>
        <el-button
          v-if="isEditingDocument"
          @click="handleCancelEditDocument"
        >
          取消
        </el-button>
        <el-button
          v-if="isEditingDocument"
          type="primary"
          :loading="docSubmitting"
          @click="handleSubmitUpdateDocument"
        >
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox, ElForm } from 'element-plus'
import { 
  Plus, 
  Edit,
  Delete,
  Folder, 
  InfoFilled, 
  Document
} from '@element-plus/icons-vue'
import KnowledgeTree from '@/components/KnowledgeTree.vue'
import {
  getKnowledge,
  addKnowledgeDocument,
  getKnowledgeDocumentsTree,
  getKnowledgeDocument,
  updateKnowledgeDocument,
  deleteKnowledgeDocument,
  type KnowledgeInfo,
  type KnowledgeAddDocumentReq,
  type DocumentInfo,
  type KnowledgeUpdateDocumentReq
} from '@/api/agent'
import type { FormRules } from 'element-plus'

const router = useRouter()
const route = useRoute()

// 知识库信息
const loading = ref(false)
const knowledgeInfo = ref<KnowledgeInfo | null>(null)
const knowledgeId = computed(() => {
  const id = route.params.id
  return typeof id === 'string' ? parseInt(id) : (Array.isArray(id) ? parseInt(id[0]) : id)
})

// 文档树
const treeData = ref<any[]>([])
const selectedNodeId = ref<string | number | null>(null)
const selectedDocument = ref<DocumentInfo | null>(null)

// 添加文档对话框
const addDocDialogVisible = ref(false)
const docFormRef = ref<InstanceType<typeof ElForm>>()
const docSubmitting = ref(false)
const docFormData = reactive<KnowledgeAddDocumentReq & { parent_id?: number }>({
  knowledge_base_id: 0,
  title: '',
  content: '',
  file_type: '',
  parent_id: 0
})

const docRules: FormRules = {
  title: [{ required: true, message: '请输入文档标题', trigger: 'blur' }],
  content: [{ required: true, message: '请输入文档内容', trigger: 'blur' }]
}

// 查看/编辑文档对话框
const viewDocDialogVisible = ref(false)
const viewDocDialogTitle = ref('查看文档')
const isEditingDocument = ref(false)
const currentDocument = ref<DocumentInfo | null>(null)
const viewDocFormRef = ref<InstanceType<typeof ElForm>>()
const viewDocFormData = reactive<KnowledgeUpdateDocumentReq & { status?: string }>({
  id: 0,
  title: '',
  content: '',
  file_type: '',
  status: ''
})

const viewDocRules: FormRules = {
  title: [{ required: true, message: '请输入文档标题', trigger: 'blur' }],
  content: [{ required: true, message: '请输入文档内容', trigger: 'blur' }]
}

// 加载知识库信息
async function loadKnowledgeInfo() {
  if (!knowledgeId.value) {
    ElMessage.error('知识库ID无效')
    router.back()
    return
  }

  try {
    loading.value = true
    const res = await getKnowledge({ id: knowledgeId.value })
    knowledgeInfo.value = res
  } catch (error: any) {
    ElMessage.error(error.message || '获取知识库信息失败')
    router.back()
  } finally {
    loading.value = false
  }
}

// 构建树形数据（扁平化，只显示一级）
function buildTreeData(documents: DocumentInfo[]): any[] {
  // 只返回根目录下的文档（parent_id === 0），扁平化显示
  return documents
    .filter(doc => (doc.parent_id || 0) === 0)
    .map(doc => ({
      id: doc.id,
      label: doc.title || '',
      title: doc.title || '',
      type: 'document',
      file_type: doc.file_type,
      status: doc.status,
      content: doc.content,
      file_size: doc.file_size,
      created_at: doc.created_at,
      updated_at: doc.updated_at,
      user: doc.user,
      doc_id: doc.doc_id,
      path: doc.path || '',
      parent_id: 0,
      sort_order: doc.sort_order || 0,
      expanded: false,
      children: []
    }))
    .sort((a, b) => {
      // 按 sort_order 排序，如果相同则按 id 降序
      return (a.sort_order || 0) - (b.sort_order || 0) || b.id - a.id
    })
}

// 加载文档树
async function loadDocuments() {
  if (!knowledgeId.value) return

  try {
    loading.value = true
    const res = await getKnowledgeDocumentsTree({ knowledge_base_id: knowledgeId.value })
    const documents = res.documents || []
    treeData.value = buildTreeData(documents)
  } catch (error: any) {
    ElMessage.error(error.message || '获取文档树失败')
  } finally {
    loading.value = false
  }
}

// 处理节点点击
function handleNodeClick(node: any) {
  if (node.type === 'document') {
    selectedDocument.value = node
    selectedNodeId.value = node.id
  } else {
    // 文件夹节点，切换展开状态
    node.expanded = !node.expanded
  }
}

// 处理节点操作
function handleNodeAction(action: string, node: any) {
  if (action === 'edit') {
    handleEditDocument(node)
  } else if (action === 'view') {
    handleViewDocument(node)
  } else if (action === 'delete') {
    handleDeleteDocument(node)
  }
}

// 添加文档
function handleAddDocument() {
  docFormData.knowledge_base_id = knowledgeId.value
  docFormData.parent_id = 0  // 所有文档都在根目录下
  docFormData.title = ''
  docFormData.content = ''
  docFormData.file_type = ''
  docFormRef.value?.clearValidate()
  addDocDialogVisible.value = true
}

// 提交文档
async function handleSubmitDocument() {
  if (!docFormRef.value) return

  try {
    await docFormRef.value.validate()
    docSubmitting.value = true

    await addKnowledgeDocument(docFormData)
    ElMessage.success('添加成功')
    addDocDialogVisible.value = false
    await loadDocuments()
    await loadKnowledgeInfo()
  } catch (error: any) {
    if (error !== false) {
      ElMessage.error(error.message || '操作失败')
    }
  } finally {
    docSubmitting.value = false
  }
}

// 添加文档对话框关闭
function handleAddDocDialogClose() {
  docFormData.knowledge_base_id = 0
  docFormData.parent_id = 0
  docFormData.title = ''
  docFormData.content = ''
  docFormData.file_type = ''
  docFormRef.value?.clearValidate()
}

// 查看文档
async function handleViewDocument(row: DocumentInfo) {
  try {
    const res = await getKnowledgeDocument({ id: row.id })
    currentDocument.value = res
    viewDocFormData.id = res.id
    viewDocFormData.title = res.title
    viewDocFormData.content = res.content
    viewDocFormData.file_type = res.file_type
    viewDocFormData.status = res.status
    isEditingDocument.value = false
    viewDocDialogTitle.value = '查看文档'
    viewDocDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(error.message || '获取文档详情失败')
  }
}

// 编辑文档
async function handleEditDocument(row: DocumentInfo) {
  await handleViewDocument(row)
  handleStartEditDocument()
}

// 开始编辑文档
function handleStartEditDocument() {
  isEditingDocument.value = true
  viewDocDialogTitle.value = '编辑文档'
  viewDocFormRef.value?.clearValidate()
}

// 取消编辑文档
function handleCancelEditDocument() {
  if (currentDocument.value) {
    viewDocFormData.title = currentDocument.value.title
    viewDocFormData.content = currentDocument.value.content
    viewDocFormData.file_type = currentDocument.value.file_type
    viewDocFormData.status = currentDocument.value.status
  }
  isEditingDocument.value = false
  viewDocDialogTitle.value = '查看文档'
  viewDocFormRef.value?.clearValidate()
}

// 提交更新文档
async function handleSubmitUpdateDocument() {
  if (!viewDocFormRef.value) return

  try {
    await viewDocFormRef.value.validate()
    docSubmitting.value = true

    await updateKnowledgeDocument(viewDocFormData)
    ElMessage.success('更新成功')
    isEditingDocument.value = false
    viewDocDialogTitle.value = '查看文档'
    await loadDocuments()
    // 重新加载当前文档信息
    if (currentDocument.value) {
      await handleViewDocument(currentDocument.value)
    }
    // 更新选中的文档
    if (selectedDocument.value && selectedDocument.value.id === currentDocument.value?.id) {
      selectedDocument.value = currentDocument.value
    }
  } catch (error: any) {
    if (error !== false) {
      ElMessage.error(error.message || '操作失败')
    }
  } finally {
    docSubmitting.value = false
  }
}

// 查看文档对话框关闭
function handleViewDocDialogClose() {
  currentDocument.value = null
  isEditingDocument.value = false
  viewDocFormData.id = 0
  viewDocFormData.title = ''
  viewDocFormData.content = ''
  viewDocFormData.file_type = ''
  viewDocFormData.status = ''
  viewDocFormRef.value?.clearValidate()
}

// 删除文档
async function handleDeleteDocument(row: DocumentInfo) {
  try {
    await ElMessageBox.confirm(`确定要删除文档"${row.title}"吗？`, '提示', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteKnowledgeDocument({ id: row.id })
    ElMessage.success('删除成功')
    selectedDocument.value = null
    selectedNodeId.value = null
    await loadDocuments()
    await loadKnowledgeInfo()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error.message || '删除失败')
    }
  }
}

// 格式化文件大小
function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

// 格式化时间
function formatTime(timeStr: string): string {
  if (!timeStr) return '-'
  try {
    const date = new Date(timeStr)
    return date.toLocaleString('zh-CN')
  } catch {
    return timeStr
  }
}

// 获取状态文本
function getStatusText(status: string): string {
  const statusMap: Record<string, string> = {
    completed: '已完成',
    failed: '失败'
  }
  return statusMap[status] || status
}

// 扁平化树（用于选择父目录）
function flattenTree(nodes: any[]): any[] {
  const result: any[] = []
  const traverse = (items: any[]) => {
    items.forEach(item => {
      result.push(item)
      if (item.children && item.children.length > 0) {
        traverse(item.children)
      }
    })
  }
  traverse(nodes)
  return result
}

// 初始化
onMounted(() => {
  loadKnowledgeInfo()
  loadDocuments()
})
</script>

<style lang="scss" scoped>
.knowledge-detail-page {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);
}

.knowledge-detail-content {
  flex: 1;
  height: calc(100vh - 20px);
  padding: var(--spacing-base, 16px);
  overflow: hidden;
  min-width: 0;

  .three-column-layout {
    display: flex;
    height: 100%;
    gap: var(--spacing-base, 16px);
    min-width: 0;
    width: 100%;
    
    .left-panel {
      width: 280px;
      min-width: 250px;
      flex-shrink: 0;
      background: var(--el-bg-color);
      border: 1px solid var(--el-border-color-light);
      border-radius: 8px;
      display: flex;
      flex-direction: column;
      overflow: hidden;

      .panel-header {
        padding: var(--spacing-base, 16px);
        border-bottom: 1px solid var(--el-border-color-light);
        display: flex;
        justify-content: space-between;
        align-items: center;
        flex-shrink: 0;
        
        h3 {
          margin: 0;
          font-size: 16px;
          color: var(--el-text-color-primary);
        }
        
        .header-actions {
          display: flex;
          gap: var(--spacing-sm, 8px);
        }
      }
      
      .tree-container {
        flex: 1;
        overflow-y: auto;
        padding: var(--spacing-sm, 8px);
        
        .knowledge-tree {
          width: 100%;
        }
      }

      .empty-state {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        color: var(--el-text-color-secondary);

        .empty-icon {
          font-size: 48px;
          margin-bottom: var(--spacing-base, 16px);
        }
      }
    }
    
    .middle-panel {
      flex: 1;
      min-width: 0; 
      background: var(--el-bg-color);
      border: 1px solid var(--el-border-color-light);
      border-radius: 8px;
      display: flex;
      flex-direction: column;
      overflow: hidden;
      
      .panel-header {
        padding: var(--spacing-base, 16px) var(--spacing-lg, 24px);
        border-bottom: 1px solid var(--el-border-color-light);
        background: var(--el-bg-color-page);
        flex-shrink: 0;
        
        .header-title {
          display: flex;
          align-items: center;
          gap: var(--spacing-sm, 8px);
          
          .page-icon {
            font-size: 20px;
            color: var(--el-color-primary);
          }
          
          .page-title {
            margin: 0;
            font-size: 18px;
            font-weight: 600;
            color: var(--el-text-color-primary);
          }
        }
      }
      
      .document-content {
        flex: 1;
        padding: var(--spacing-lg, 24px);
        overflow-y: auto;
        
        .content-preview {
          background: var(--el-bg-color-page);
          border: 1px solid var(--el-border-color-light);
          border-radius: 6px;
          padding: var(--spacing-lg, 24px);
          height: calc(100vh - 200px);
          min-height: 600px;
          
          pre {
            margin: 0;
            white-space: pre-wrap;
            word-wrap: break-word;
            font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
            font-size: 14px;
            line-height: 1.6;
            color: var(--el-text-color-primary);
            height: 100%;
            overflow-y: auto;
          }
          
          .unsupported-format {
            text-align: center;
            padding: var(--spacing-2xl, 48px);
            color: var(--el-text-color-secondary);
            
            .el-icon {
              font-size: 48px;
              margin-bottom: var(--spacing-base, 16px);
            }
          }
        }
        
        .no-selection {
          flex: 1;
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          color: var(--el-text-color-secondary);
          
          .no-selection-icon {
            font-size: 64px;
            margin-bottom: var(--spacing-lg, 24px);
          }
          
          h3 {
            margin: 0 0 var(--spacing-sm, 8px) 0;
            font-size: 18px;
          }
          
          p {
            margin: 0;
            font-size: 14px;
          }
        }
      }
    }
    
    .right-panel {
      width: 300px;
      min-width: 280px;
      flex-shrink: 0;
      background: var(--el-bg-color);
      border: 1px solid var(--el-border-color-light);
      border-radius: 8px;
      display: flex;
      flex-direction: column;
      overflow: hidden;
      
      .panel-header {
        padding: var(--spacing-base, 16px);
        border-bottom: 1px solid var(--el-border-color-light);
        flex-shrink: 0;
        
        h3 {
          margin: 0;
          font-size: 16px;
          color: var(--el-text-color-primary);
        }
      }
      
      .document-info {
        flex: 1;
        overflow-y: auto;
        padding: var(--spacing-base, 16px);
        
        .info-section {
          margin-bottom: var(--spacing-lg, 24px);
          
          h4 {
            margin: 0 0 var(--spacing-base, 16px) 0;
            font-size: 14px;
            color: var(--el-text-color-primary);
            font-weight: 600;
          }
          
          .info-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: var(--spacing-sm, 8px) 0;
            border-bottom: 1px solid var(--el-border-color-lighter);
            
            label {
              font-size: 12px;
              color: var(--el-text-color-secondary);
              font-weight: 500;
            }
            
            span {
              font-size: 12px;
              color: var(--el-text-color-primary);
            }
          }
        }
        
        .action-section {
          h4 {
            margin: 0 0 var(--spacing-base, 16px) 0;
            font-size: 14px;
            color: var(--el-text-color-primary);
            font-weight: 600;
          }
          
          .action-buttons {
            display: flex;
            flex-direction: column;
            gap: var(--spacing-sm, 8px);
          }
        }
      }
      
      .no-selection {
        flex: 1;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        color: var(--el-text-color-secondary);
        padding: var(--spacing-lg, 24px);
        
        .no-selection-icon {
          font-size: 48px;
          margin-bottom: var(--spacing-base, 16px);
        }
        
        h3 {
          margin: 0 0 var(--spacing-sm, 8px) 0;
          font-size: 16px;
        }
        
        p {
          margin: 0;
          font-size: 12px;
          text-align: center;
        }
      }
    }
  }
}
</style>
