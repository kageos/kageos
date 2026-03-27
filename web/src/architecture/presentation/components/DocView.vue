<template>
  <div class="doc-view" v-loading="loading">
    <!-- 阅读区：收窄 + 居中，参考 CSDN 文章区 -->
    <div class="doc-view__reader" v-if="doc">
      <div class="doc-content">
        <!-- 文档头部 -->
        <div class="doc-header">
          <div class="doc-title-section">
            <h1 class="doc-title">{{ doc.name || props.node?.name || '未命名文档' }}</h1>
            <div class="doc-meta">
              <el-tag v-if="doc.format" size="small" type="info">{{ doc.format }}</el-tag>
              <span v-if="doc.category" class="doc-category">{{ doc.category }}</span>
            </div>
            <div v-if="doc.created_by || doc.created_at || doc.updated_at" class="doc-info-row">
              <span v-if="doc.created_by" class="doc-info-item doc-info-user">
                <UserDisplay :username="doc.created_by" mode="simple" size="small" layout="horizontal" />
              </span>
              <span v-if="doc.created_at" class="doc-info-item">
                <el-icon><Clock /></el-icon>
                <span>创建 {{ formatDate(doc.created_at) }}</span>
              </span>
              <span v-if="doc.updated_at && doc.updated_at !== doc.created_at" class="doc-info-item">
                <el-icon><RefreshRight /></el-icon>
                <span>更新 {{ formatDate(doc.updated_at) }}</span>
              </span>
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
              v-model="editSummary"
              type="textarea"
              placeholder="文档摘要（可选）"
              class="doc-summary-input"
              :rows="2"
              maxlength="500"
              show-word-limit
            />
            
            <!-- ✨ 使用 Vditor 所见即所得编辑器（支持拖拽/粘贴上传） -->
            <VditorEditor
              v-model="editContent"
              height="100%"
              placeholder="请输入文档内容，支持拖拽文件到此处或粘贴图片/文件上传"
              class="doc-vditor-editor"
            />
            <div class="doc-editor-upload-hint">
              支持拖拽文件到编辑区上传，或粘贴剪贴板中的图片/文件上传
            </div>
          </div>

          <!-- 预览模式：支持图片点击预览 -->
          <div v-else class="doc-preview">
            <div 
              v-if="doc.format === 'markdown'"
              ref="markdownContentRef"
              v-html="renderedContent"
              class="markdown-content"
              @click="onMarkdownClick"
            />
            <pre v-else class="plain-text-content">{{ doc.content }}</pre>
          </div>
        </div>
      </div>
    </div>

    <!-- 文档 403：与函数无权限一致，展示申请权限组件 -->
    <PermissionDeniedView v-else-if="docPermissionDenied" />

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

    <!-- 图片预览弹层 -->
    <Teleport to="body">
      <Transition name="doc-image-preview">
        <div v-if="imagePreviewVisible" class="doc-image-preview" @click.self="closeImagePreview">
          <button type="button" class="doc-image-preview__close" aria-label="关闭" @click="closeImagePreview">
            <el-icon :size="24"><Close /></el-icon>
          </button>
          <button
            v-if="previewImgList.length > 1 && previewIndex > 0"
            type="button"
            class="doc-image-preview__nav doc-image-preview__nav--prev"
            aria-label="上一张"
            @click="previewIndex = previewIndex - 1"
          >
            <el-icon :size="28"><ArrowLeft /></el-icon>
          </button>
          <button
            v-if="previewImgList.length > 1 && previewIndex < previewImgList.length - 1"
            type="button"
            class="doc-image-preview__nav doc-image-preview__nav--next"
            aria-label="下一张"
            @click="previewIndex = previewIndex + 1"
          >
            <el-icon :size="28"><ArrowRight /></el-icon>
          </button>
          <div class="doc-image-preview__wrap" @click.self="closeImagePreview">
            <img
              :src="previewImgList[previewIndex]"
              :alt="`预览 ${previewIndex + 1}/${previewImgList.length}`"
              class="doc-image-preview__img"
              @click.stop
            />
          </div>
          <div v-if="previewImgList.length > 1" class="doc-image-preview__indicator">
            {{ previewIndex + 1 }} / {{ previewImgList.length }}
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Edit, Check, Plus, Delete, Close, ArrowLeft, ArrowRight, Clock, RefreshRight } from '@element-plus/icons-vue'
import { marked } from 'marked'
import type { ServiceTree } from '@/types'
import { getDoc, updateDoc, deleteDoc } from '@/api/doc'  // ✅ 使用新的文档 API
import { hasPermission, DocsPermission, buildPermissionApplyURL } from '@/utils/permission'
import { usePermissionErrorStore } from '@/stores/permissionError'
import { escapeHtml, sanitizeHtml } from '@/utils/sanitizeHtml'
import VditorEditor from '@/components/VditorEditor.vue'
import UserDisplay from '../widgets/UserDisplay.vue'
import PermissionDeniedView from './PermissionDeniedView.vue'

marked.setOptions({ breaks: true, gfm: true })

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
const editSummary = ref('')
const editContent = ref('')

// 文档加载 403：展示申请权限组件（与函数无权限一致）
const docPermissionDenied = ref(false)
const permissionErrorStore = usePermissionErrorStore()

// 图片预览
const markdownContentRef = ref<HTMLElement | null>(null)
const imagePreviewVisible = ref(false)
const previewImgList = ref<string[]>([])
const previewIndex = ref(0)

function onMarkdownClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (target?.tagName !== 'IMG') return
  const img = target as HTMLImageElement
  const src = img.src || img.getAttribute('src')
  if (!src) return
  const container = markdownContentRef.value
  if (!container) return
  const imgs = container.querySelectorAll<HTMLImageElement>('img')
  const list = Array.from(imgs).map(i => i.src || i.getAttribute('src') || '').filter(Boolean)
  const idx = list.indexOf(src)
  if (idx === -1) return
  previewImgList.value = list
  previewIndex.value = idx
  imagePreviewVisible.value = true
}

function closeImagePreview() {
  imagePreviewVisible.value = false
  previewImgList.value = []
  previewIndex.value = 0
}

function onPreviewKeydown(e: KeyboardEvent) {
  if (!imagePreviewVisible.value) return
  if (e.key === 'Escape') {
    closeImagePreview()
    return
  }
  if (e.key === 'ArrowLeft' && previewIndex.value > 0) {
    previewIndex.value -= 1
    return
  }
  if (e.key === 'ArrowRight' && previewIndex.value < previewImgList.value.length - 1) {
    previewIndex.value += 1
  }
}

onMounted(() => {
  document.addEventListener('keydown', onPreviewKeydown)
})
onUnmounted(() => {
  document.removeEventListener('keydown', onPreviewKeydown)
})

// 权限检查
const hasEditPermission = computed(() => {
  return props.node && hasPermission(props.node, DocsPermission.write)
})

// 渲染后的 Markdown 内容
const renderedContent = computed(() => {
  if (!doc.value || !doc.value.content) {
    return ''
  }
  try {
    return sanitizeHtml(marked.parse(doc.value.content) as string)
  } catch (error) {
    console.error('Markdown 渲染失败:', error)
    return escapeHtml(doc.value.content).replace(/\n/g, '<br>')
  }
})

// 格式化时间（创建/更新展示用）
function formatDate(date: string | undefined): string {
  if (!date) return ''
  try {
    const d = new Date(date)
    return d.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  } catch {
    return date
  }
}

// 加载文档
const loadDoc = async () => {
  if (!props.node?.full_code_path) {
    return
  }

  loading.value = true
  docPermissionDenied.value = false
  try {
    // ✅ 使用 full_code_path 调用新接口
    const data = await getDoc(props.node.full_code_path)
    doc.value = data || null
  } catch (error: any) {
    if (error.response?.status === 404) {
      // 文档不存在，这是正常情况（节点已创建但文档内容未创建）
      doc.value = null
    } else if (error.response?.status === 403) {
      // 与函数无权限一致：写入 store 并展示申请权限组件，不弹窗
      docPermissionDenied.value = true
      const path = props.node.full_code_path
      permissionErrorStore.setError({
        resource_path: path,
        action: DocsPermission.read,
        action_display: '查看文档',
        apply_url: buildPermissionApplyURL(path, DocsPermission.read),
        error_message: '没有查看该文档的权限'
      })
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
  editSummary.value = ''
  editContent.value = ''
}

// 编辑文档
const handleEdit = () => {
  if (doc.value) {
    isEditing.value = true
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

  if (!editContent.value.trim()) {
    ElMessage.warning('请输入文档内容')
    return
  }

  saving.value = true
  try {
    if (doc.value) {
      // ✅ 更新文档（使用 full_code_path）
      const data = await updateDoc(props.node.full_code_path, {
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
      `确定要删除文档"${doc.value.name || props.node?.name || '未命名文档'}"吗？此操作将删除文档内容和文档节点，且无法恢复。`,
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

/* 阅读区收窄 + 居中，参考 CSDN 文章宽度 */
.doc-view__reader {
  width: 100%;
  max-width: 720px;
  margin: 0 auto;
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
  font-size: 26px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 12px 0;
  line-height: 1.35;
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

.doc-info-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 8px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.doc-info-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.doc-info-item .el-icon {
  font-size: 14px;
  color: var(--el-text-color-placeholder);
}

.doc-info-user {
  margin-right: 4px;
}

.doc-info-user .user-display-wrapper {
  display: inline-flex;
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

.doc-editor-upload-hint {
  flex-shrink: 0;
  padding: 8px 0 0;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.doc-preview {
  min-height: 400px;
}

/* Markdown 正文：CSDN 风格排版 */
.markdown-content {
  font-size: 16px;
  line-height: 1.85;
  color: var(--text-primary);
  word-break: break-word;

  :deep(h1) {
    font-size: 26px;
    font-weight: 600;
    margin: 28px 0 16px 0;
    padding-bottom: 10px;
    border-bottom: 2px solid var(--border-base);
    color: var(--text-primary);
    line-height: 1.35;
  }
  :deep(h1:first-child) {
    margin-top: 0;
  }

  :deep(h2) {
    font-size: 22px;
    font-weight: 600;
    margin: 24px 0 12px 0;
    color: var(--text-primary);
    line-height: 1.4;
  }

  :deep(h3) {
    font-size: 18px;
    font-weight: 600;
    margin: 20px 0 10px 0;
    color: var(--text-primary);
  }

  :deep(h4), :deep(h5), :deep(h6) {
    font-size: 16px;
    font-weight: 600;
    margin: 16px 0 8px 0;
    color: var(--text-primary);
  }

  :deep(p) {
    margin: 14px 0;
    color: var(--text-regular);
  }

  :deep(ul), :deep(ol) {
    margin: 14px 0;
    padding-left: 26px;
    color: var(--text-regular);
  }

  :deep(li) {
    margin: 6px 0;
  }

  :deep(code) {
    background: var(--bg-tertiary);
    color: var(--color-primary);
    padding: 2px 6px;
    border-radius: 4px;
    font-family: 'SF Mono', 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 0.92em;
    font-weight: 500;
  }

  :deep(pre) {
    background: var(--el-fill-color-darker, #1e1e1e);
    color: #e0e0e0;
    padding: 16px 18px;
    border-radius: 8px;
    overflow-x: auto;
    margin: 16px 0;
    font-size: 14px;
    line-height: 1.6;

    code {
      background: transparent;
      color: inherit;
      padding: 0;
      font-weight: normal;
      font-size: inherit;
    }
  }

  :deep(blockquote) {
    border-left: 4px solid var(--color-primary);
    padding: 10px 16px;
    margin: 16px 0;
    color: var(--text-regular);
    background: var(--bg-secondary);
    border-radius: 0 8px 8px 0;
  }

  :deep(a) {
    color: var(--color-primary);
    text-decoration: none;
    font-weight: 500;
    transition: color 0.2s ease;

    &:hover {
      text-decoration: underline;
      opacity: 0.9;
    }
  }

  :deep(table) {
    width: 100%;
    border-collapse: collapse;
    margin: 16px 0;
    font-size: 15px;

    th, td {
      border: 1px solid var(--border-base);
      padding: 10px 14px;
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

  :deep(img) {
    max-width: 100%;
    height: auto;
    border-radius: 8px;
    margin: 12px 0;
    cursor: pointer;
    vertical-align: middle;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
    transition: box-shadow 0.2s ease;

    &:hover {
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
    }
  }

  :deep(video) {
    max-width: 100%;
    height: auto;
    border-radius: 8px;
    margin: 12px 0;
  }

  :deep(hr) {
    border: none;
    border-top: 1px solid var(--border-base);
    margin: 24px 0;
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

/* 图片预览弹层 */
.doc-image-preview {
  position: fixed;
  inset: 0;
  z-index: 9999;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px;

  &__close {
    position: absolute;
    top: 20px;
    right: 20px;
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: rgba(255, 255, 255, 0.12);
    color: #fff;
    border-radius: 50%;
    cursor: pointer;
    transition: background 0.2s ease;

    &:hover {
      background: rgba(255, 255, 255, 0.22);
    }
  }

  &__nav {
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: rgba(255, 255, 255, 0.12);
    color: #fff;
    border-radius: 50%;
    cursor: pointer;
    transition: background 0.2s ease;

    &:hover {
      background: rgba(255, 255, 255, 0.22);
    }

    &--prev {
      left: 24px;
    }
    &--next {
      right: 24px;
    }
  }

  &__wrap {
    max-width: 90vw;
    max-height: 85vh;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  &__img {
    max-width: 100%;
    max-height: 85vh;
    width: auto;
    height: auto;
    object-fit: contain;
    border-radius: 8px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }

  &__indicator {
    position: absolute;
    bottom: 24px;
    left: 50%;
    transform: translateX(-50%);
    padding: 6px 14px;
    background: rgba(255, 255, 255, 0.15);
    color: #fff;
    border-radius: 20px;
    font-size: 14px;
  }
}

.doc-image-preview-enter-active,
.doc-image-preview-leave-active {
  transition: opacity 0.2s ease;
}
.doc-image-preview-enter-from,
.doc-image-preview-leave-to {
  opacity: 0;
}
</style>
