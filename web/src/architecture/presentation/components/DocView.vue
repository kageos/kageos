<template>
  <div class="doc-view" v-loading="loading">
    <div v-if="doc" class="doc-content">
      <!-- 文档头部 -->
      <div class="doc-header">
        <div class="doc-title-section">
          <h1 class="doc-title">{{ doc.title }}</h1>
          <div class="doc-meta">
            <el-tag v-if="doc.format" size="small" type="info">{{ doc.format }}</el-tag>
            <span v-if="doc.category" class="doc-category">{{ doc.category }}</span>
          </div>
        </div>
        <div class="doc-actions" v-if="hasEditPermission">
          <el-button 
            type="primary" 
            :icon="Edit" 
            @click="handleEdit"
            v-if="!isEditing"
          >
            编辑文档
          </el-button>
          <el-button 
            v-else
            :icon="Check"
            @click="handleSave"
            :loading="saving"
          >
            保存
          </el-button>
          <el-button 
            v-if="isEditing"
            @click="handleCancel"
          >
            取消
          </el-button>
          <el-button 
            type="danger" 
            :icon="Delete" 
            @click="handleDelete"
            v-if="!isEditing"
          >
            删除文档
          </el-button>
        </div>
      </div>

      <!-- 文档摘要 -->
      <div v-if="doc.summary && !isEditing" class="doc-summary">
        <p>{{ doc.summary }}</p>
      </div>

      <!-- 文档内容 -->
      <div class="doc-body">
        <!-- 编辑模式 -->
        <div v-if="isEditing" class="doc-editor">
          <el-input
            v-model="editTitle"
            placeholder="文档标题"
            class="doc-title-input"
            maxlength="255"
            show-word-limit
          />
          <el-input
            v-model="editSummary"
            type="textarea"
            placeholder="文档摘要（可选）"
            class="doc-summary-input"
            :rows="2"
            maxlength="500"
            show-word-limit
          />
          
          <!-- ✨ 使用 Vditor 所见即所得编辑器 -->
          <VditorEditor
            v-model="editContent"
            height="100%"
            placeholder="请输入文档内容（支持 Markdown）"
            class="doc-vditor-editor"
          />
        </div>

        <!-- 预览模式 -->
        <div v-else class="doc-preview">
          <div 
            v-if="doc.format === 'markdown'"
            v-html="renderedContent"
            class="markdown-content"
          />
          <pre v-else class="plain-text-content">{{ doc.content }}</pre>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else-if="!loading" class="doc-empty">
      <el-empty description="文档不存在或尚未创建">
        <el-button 
          v-if="hasEditPermission"
          type="primary" 
          :icon="Plus"
          @click="handleCreate"
        >
          创建文档
        </el-button>
      </el-empty>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Edit, Check, Plus, Delete } from '@element-plus/icons-vue'
import { marked } from 'marked'
import type { ServiceTree } from '@/types'
import { getDoc, updateDoc, deleteDoc } from '@/api/doc'  // ✅ 使用新的文档 API
import { hasPermission, DirectoryPermissions } from '@/utils/permission'
import VditorEditor from '@/components/VditorEditor.vue'

interface Props {
  node: ServiceTree
}

interface Emits {
  (e: 'deleted'): void  // ⭐ 文档删除后触发，通知父组件刷新
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 文档数据
const doc = ref<any>(null)
const loading = ref(false)
const saving = ref(false)

// 编辑状态
const isEditing = ref(false)
const editTitle = ref('')
const editSummary = ref('')
const editContent = ref('')

// 权限检查
const hasEditPermission = computed(() => {
  return props.node && hasPermission(props.node, DirectoryPermissions.write)
})

// 渲染后的 Markdown 内容
const renderedContent = computed(() => {
  if (!doc.value || !doc.value.content) {
    return ''
  }
  try {
    return marked(doc.value.content)
  } catch (error) {
    console.error('Markdown 渲染失败:', error)
    return doc.value.content
  }
})

// 加载文档
const loadDoc = async () => {
  if (!props.node?.full_code_path) {
    return
  }

  loading.value = true
  try {
    // ✅ 使用 full_code_path 调用新接口
    const data = await getDoc(props.node.full_code_path)
    doc.value = data || null
  } catch (error: any) {
    if (error.response?.status === 404) {
      // 文档不存在，这是正常情况（节点已创建但文档内容未创建）
      doc.value = null
    } else {
      ElMessage.error('加载文档失败: ' + (error.message || '未知错误'))
    }
  } finally {
    loading.value = false
  }
}

// 创建文档
const handleCreate = () => {
  isEditing.value = true
  editTitle.value = props.node.name || ''
  editSummary.value = ''
  editContent.value = ''
}

// 编辑文档
const handleEdit = () => {
  if (doc.value) {
    isEditing.value = true
    editTitle.value = doc.value.title || ''
    editSummary.value = doc.value.summary || ''
    editContent.value = doc.value.content || ''
  }
}

// 保存文档
const handleSave = async () => {
  if (!props.node?.full_code_path) {
    ElMessage.error('文档路径不存在')
    return
  }

  if (!editTitle.value.trim()) {
    ElMessage.warning('请输入文档标题')
    return
  }

  if (!editContent.value.trim()) {
    ElMessage.warning('请输入文档内容')
    return
  }

  saving.value = true
  try {
    if (doc.value) {
      // ✅ 更新文档（使用 full_code_path）
      const data = await updateDoc(props.node.full_code_path, {
        title: editTitle.value.trim(),
        content: editContent.value.trim(),
        summary: editSummary.value.trim() || undefined,
        format: 'markdown'
      })
      doc.value = data
      ElMessage.success('文档保存成功')
      isEditing.value = false
    } else {
      // ❌ 文档不存在时，需要通过 service_tree 创建
      ElMessage.error('文档不存在，请先在服务树中创建文档节点')
    }
  } catch (error: any) {
    ElMessage.error('保存文档失败: ' + (error.message || '未知错误'))
  } finally {
    saving.value = false
  }
}

// 取消编辑
const handleCancel = async () => {
  if (doc.value) {
    // 有文档内容，确认是否放弃修改
    try {
      await ElMessageBox.confirm('确定要放弃修改吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })
      isEditing.value = false
    } catch {
      // 用户取消
    }
  } else {
    // 没有文档内容，直接取消
    isEditing.value = false
  }
}

// 删除文档
const handleDelete = async () => {
  if (!props.node?.full_code_path) {
    ElMessage.error('文档路径不存在')
    return
  }

  if (!doc.value) {
    ElMessage.warning('文档不存在')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除文档"${doc.value.title}"吗？此操作将删除文档内容和文档节点，且无法恢复。`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    loading.value = true
    try {
      // ✅ 使用 full_code_path 调用新接口
      await deleteDoc(props.node.full_code_path)
      ElMessage.success('文档删除成功')
      doc.value = null
      // ⭐ 通知父组件文档已删除
      emit('deleted')
    } catch (error: any) {
      ElMessage.error('删除文档失败: ' + (error.message || '未知错误'))
    } finally {
      loading.value = false
    }
  } catch {
    // 用户取消删除
  }
}

// 监听节点变化
// 监听节点 ID 变化，自动加载文档
// immediate: true 会在组件挂载时立即执行一次，无需在 onMounted 中重复调用
watch(() => props.node?.id, () => {
  if (props.node?.id) {
    loadDoc()
    isEditing.value = false
  }
}, { immediate: true })
</script>

<style scoped lang="scss">
.doc-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  border-radius: var(--border-radius-lg);
  padding: 24px;
  overflow-y: auto;
}

.doc-content {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.doc-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-base);
}

.doc-title-section {
  flex: 1;
}

.doc-title {
  font-size: 28px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 12px 0;
}

.doc-meta {
  display: flex;
  align-items: center;
  gap: 12px;
}

.doc-category {
  font-size: 14px;
  color: var(--text-secondary);
}

.doc-actions {
  display: flex;
  gap: 8px;
}

.doc-summary {
  margin-bottom: 24px;
  padding: 16px;
  background: var(--bg-secondary);
  border-radius: var(--border-radius-base);
  
  p {
    margin: 0;
    color: var(--text-regular);
    line-height: 1.6;
  }
}

.doc-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0; /* 关键：让 flex 子元素可以缩小 */
}

.doc-editor {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 0; /* 关键：让 flex 子元素可以缩小 */
}

.doc-title-input {
  font-size: 18px;
  font-weight: 600;
  flex-shrink: 0; /* 标题输入框不缩小 */
}

.doc-summary-input {
  font-size: 14px;
  flex-shrink: 0; /* 摘要输入框不缩小 */
}

.doc-vditor-editor {
  flex: 1; /* 占据剩余所有空间 */
  min-height: 400px; /* 最小高度 */
  display: flex;
  flex-direction: column;
}

.doc-preview {
  min-height: 400px;
}

.markdown-content {
  font-size: 16px;
  line-height: 1.8;
  color: var(--text-primary);
  
  :deep(h1) {
    font-size: 28px;
    font-weight: 600;
    margin: 24px 0 16px 0;
    padding-bottom: 8px;
    border-bottom: 2px solid var(--border-base);
    color: var(--text-primary);
  }
  
  :deep(h2) {
    font-size: 24px;
    font-weight: 600;
    margin: 20px 0 12px 0;
    color: var(--text-primary);
  }
  
  :deep(h3) {
    font-size: 20px;
    font-weight: 600;
    margin: 16px 0 8px 0;
    color: var(--text-primary);
  }
  
  :deep(p) {
    margin: 12px 0;
    color: var(--text-regular);
  }
  
  :deep(ul), :deep(ol) {
    margin: 12px 0;
    padding-left: 24px;
    color: var(--text-regular);
  }
  
  :deep(li) {
    margin: 6px 0;
  }
  
  :deep(code) {
    background: var(--bg-tertiary);
    color: var(--color-primary);
    padding: 2px 6px;
    border-radius: var(--border-radius-sm);
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 14px;
    font-weight: 500;
  }
  
  :deep(pre) {
    background: var(--bg-secondary);
    border: 1px solid var(--border-base);
    padding: 16px;
    border-radius: var(--border-radius-base);
    overflow-x: auto;
    margin: 16px 0;
    
    code {
      background: transparent;
      color: var(--text-primary);
      padding: 0;
      font-weight: normal;
    }
  }
  
  :deep(blockquote) {
    border-left: 4px solid var(--color-primary);
    padding-left: 16px;
    margin: 16px 0;
    color: var(--text-regular);
    background: var(--bg-secondary);
    padding: 12px 16px;
    border-radius: var(--border-radius-base);
  }
  
  :deep(a) {
    color: var(--color-primary);
    text-decoration: none;
    font-weight: 500;
    transition: all 0.2s ease;
    
    &:hover {
      text-decoration: underline;
      opacity: 0.8;
    }
  }
  
  :deep(table) {
    width: 100%;
    border-collapse: collapse;
    margin: 16px 0;
    
    th, td {
      border: 1px solid var(--border-base);
      padding: 8px 12px;
      text-align: left;
    }
    
    th {
      background: var(--bg-secondary);
      color: var(--text-primary);
      font-weight: 600;
    }
    
    td {
      color: var(--text-regular);
    }
  }
}

.plain-text-content {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-wrap: break-word;
  background: var(--bg-secondary);
  border: 1px solid var(--border-base);
  color: var(--text-primary);
  padding: 16px;
  border-radius: var(--border-radius-base);
}

.doc-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}
</style>
