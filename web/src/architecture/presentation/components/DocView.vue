<template>
  <div class="doc-view" :class="{ 'doc-view--editing': isEditing }" v-loading="loading">
    <!-- 阅读区 -->
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
          <div class="doc-actions" v-if="canEditDoc">
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
            
            <VditorEditor
              v-model="editContent"
              height="100%"
              placeholder="开始写文档..."
              class="doc-vditor-editor"
            />
          </div>

          <!-- 预览模式：支持图片点击预览 -->
          <div v-else class="doc-preview">
            <div 
              v-if="doc.format === 'markdown'"
              ref="markdownContentRef"
              v-html="renderedContent"
              class="markdown-content"
              @click="onMarkdownClick"
              @mouseover="onMarkdownMouseover"
              @focusin="onMarkdownFocusin"
              @focusout="onMarkdownFocusout"
              @mouseleave="scheduleCloseResourcePreview"
              @copy="onMarkdownCopy"
            />
            <pre v-else class="plain-text-content">{{ doc.content }}</pre>
          </div>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else-if="!loading" class="doc-empty">
      <kageos-empty description="文档不存在或尚未创建">
        <el-button 
          v-if="canEditDoc"
          type="primary" 
          :icon="Plus"
          @click="handleCreate"
        >
          创建文档
        </el-button>
      </kageos-empty>
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

    <!-- 文档内资源引用预览卡片 -->
    <Teleport to="body">
      <Transition name="doc-resource-preview">
        <section
          v-if="resourcePreview.visible"
          ref="resourcePreviewRef"
          :class="['doc-resource-preview', `is-${resourcePreview.kind}`]"
          :style="{ left: `${resourcePreview.left}px`, top: `${resourcePreview.top}px` }"
          role="dialog"
          aria-label="资源预览"
          @mouseenter="cancelCloseResourcePreview"
          @mouseleave="scheduleCloseResourcePreview"
          @click.stop
        >
          <div class="doc-resource-preview__head">
            <span class="doc-resource-preview__icon" aria-hidden="true">
              <img
                v-if="resourcePreview.iconSrc"
                :src="resourcePreview.iconSrc"
                :alt="resourcePreview.typeLabel"
                class="doc-resource-preview__img"
              />
              <span
                v-else
                class="doc-resource-preview__html-icon"
                v-html="resourcePreview.iconHtml"
              />
            </span>
            <span class="doc-resource-preview__main">
              <strong>{{ resourcePreview.label }}</strong>
              <span>{{ resourcePreview.typeLabel }}</span>
            </span>
          </div>
          <p v-if="resourcePreview.description" class="doc-resource-preview__desc">
            {{ resourcePreview.description }}
          </p>
          <div v-if="resourcePreview.metaItems.length > 0" class="doc-resource-preview__meta">
            <span
              v-for="item in resourcePreview.metaItems"
              :key="item"
              class="doc-resource-preview__meta-item"
              :title="item"
            >
              {{ item }}
            </span>
          </div>
          <code class="doc-resource-preview__path">{{ resourcePreview.path }}</code>
          <div class="doc-resource-preview__actions">
            <span v-if="resourcePreview.loading" class="doc-resource-preview__loading">加载详情...</span>
            <button
              v-if="!isWorkspaceToolResourcePath(resourcePreview.path)"
              type="button"
              class="doc-resource-preview__link"
              @click="openResourcePreviewTarget"
            >
              打开资源
            </button>
            <button type="button" class="doc-resource-preview__close" aria-label="关闭资源预览" @click="closeResourcePreview">
              关闭
            </button>
          </div>
        </section>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, defineAsyncComponent, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Edit, Check, Plus, Delete, Close, ArrowLeft, ArrowRight, Clock, RefreshRight } from '@element-plus/icons-vue'
import type { ServiceTree } from '@/architecture/domain/types'
import { getDoc, updateDoc, deleteDoc } from '@/architecture/presentation/context/api/doc'  // ✅ 使用新的文档 API
import { batchGetServiceTreeDetails, type ServiceTreeDetailResp } from '@/architecture/presentation/context/api/service-tree'
import { batchGetWorkspaceToolDetails, type WorkspaceToolDetail } from '@/architecture/presentation/context/api/workspace'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'
import {
  getWorkspaceResourceSelectionText,
  isWorkspaceToolResourcePath,
  renderWorkspaceResourceTokensAsHtml,
  workspaceToolName,
  workspaceResourceIconHtml,
  workspaceResourceIconSrc,
} from '@/architecture/presentation/components/utils/workspaceInvocationSnippet'
import { consumeDocAutoEdit } from '@/architecture/presentation/utils/docAutoEdit'
import UserDisplay from '@/architecture/presentation/shared/components/UserDisplay.vue'

const VditorEditor = defineAsyncComponent(() => import('@/architecture/presentation/shared/components/VditorEditor.vue'))
const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()
const router = useRouter()

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

// 图片预览
const markdownContentRef = ref<HTMLElement | null>(null)
const resourcePreviewRef = ref<HTMLElement | null>(null)
const imagePreviewVisible = ref(false)
const previewImgList = ref<string[]>([])
const previewIndex = ref(0)

interface DocResourcePreviewMeta {
  path: string
  label: string
  typeLabel: string
  kind: string
  description: string
  metaItems: string[]
  iconSrc: string
  iconHtml: string
}

interface DocResourcePreview extends DocResourcePreviewMeta {
  visible: boolean
  href: string
  left: number
  top: number
  loading: boolean
}

const resourcePreview = ref<DocResourcePreview>(createEmptyResourcePreview())
const resourcePreviewMetaByPath = ref<Record<string, DocResourcePreviewMeta>>({})
const resourcePreviewHydratingPaths = new Set<string>()
let resourcePreviewCloseTimer: ReturnType<typeof setTimeout> | null = null
let resourceTokenHydrateTimer: ReturnType<typeof setTimeout> | null = null

function onMarkdownClick(e: MouseEvent) {
  const resourceLink = getResourceLinkFromEventTarget(e.target)
  if (resourceLink) {
    e.preventDefault()
    showResourcePreview(resourceLink)
    return
  }
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

function onMarkdownMouseover(e: MouseEvent) {
  const resourceLink = getResourceLinkFromEventTarget(e.target)
  if (!resourceLink) return
  showResourcePreview(resourceLink)
}

function onMarkdownFocusin(e: FocusEvent) {
  const resourceLink = getResourceLinkFromEventTarget(e.target)
  if (!resourceLink) return
  showResourcePreview(resourceLink)
}

function onMarkdownFocusout(e: FocusEvent) {
  const nextTarget = e.relatedTarget
  if (isResourcePreviewFocusTarget(nextTarget) || isMarkdownFocusTarget(nextTarget)) {
    return
  }
  closeResourcePreview()
}

function onMarkdownCopy(e: ClipboardEvent) {
  const root = e.currentTarget instanceof HTMLElement ? e.currentTarget : markdownContentRef.value
  const text = getWorkspaceResourceSelectionText(root)
  if (!text) return
  e.preventDefault()
  e.clipboardData?.setData('text/plain', text)
}

function getResourceLinkFromEventTarget(target: EventTarget | null): HTMLAnchorElement | null {
  if (!(target instanceof Element)) return null
  const resourceLink = target.closest('a.workspace-resource-token') as HTMLAnchorElement | null
  if (!resourceLink || !markdownContentRef.value?.contains(resourceLink)) return null
  return resourceLink
}

function showResourcePreview(resourceLink: HTMLAnchorElement) {
  cancelCloseResourcePreview()
  const path = resourceLink.dataset.fullCodePath || ''
  if (!path) return

  const fallbackMeta = buildResourcePreviewMetaFromLink(resourceLink)
  const cachedMeta = resourcePreviewMetaByPath.value[path]
  const canHydrate = isHydratableResourcePath(path)
  const position = getResourcePreviewPosition(resourceLink)
  const href = resourceLink.getAttribute('href') || ''
  const nextMeta = cachedMeta || fallbackMeta
  resourcePreview.value = {
    ...nextMeta,
    href,
    visible: true,
    left: position.left,
    top: position.top,
    loading: !cachedMeta && canHydrate,
  }

  if (canHydrate && !cachedMeta && !resourcePreviewHydratingPaths.has(path)) {
    void hydrateResourcePreview(path)
  }
}

function buildResourcePreviewMetaFromLink(resourceLink: HTMLAnchorElement): DocResourcePreviewMeta {
  const path = resourceLink.dataset.fullCodePath || ''
  const kind = resourceLink.dataset.resourceKind || inferResourceKind(path)
  return {
    path,
    label: resourceLink.dataset.resourceLabel || getPathTail(path) || path || '资源',
    typeLabel: resourceLink.dataset.resourceTypeLabel || getResourcePreviewTypeLabel(kind),
    kind,
    description: '',
    metaItems: compactPreviewMetaItems([getReadableResourcePath(path)]),
    iconSrc: getResourcePreviewIconSrc(kind),
    iconHtml: getResourcePreviewIconHtml(kind, resourceLink.dataset.resourceTypeLabel || getResourcePreviewTypeLabel(kind)),
  }
}

async function hydrateResourcePreview(path: string) {
  if (isWorkspaceToolResourcePath(path)) {
    await hydrateToolResourcePreviews([path])
    return
  }
  await hydrateWorkspaceResourcePreviews([path])
}

async function hydrateWorkspaceResourcePreviews(paths: string[]) {
  const uniquePaths = Array.from(new Set(paths))
    .filter(path => !isWorkspaceToolResourcePath(path) && isHydratableResourcePath(path))
    .filter(path => !resourcePreviewMetaByPath.value[path] && !resourcePreviewHydratingPaths.has(path))
  if (uniquePaths.length === 0) return

  uniquePaths.forEach(path => resourcePreviewHydratingPaths.add(path))
  try {
    const resp = await batchGetServiceTreeDetails({ full_code_paths: uniquePaths })
    const nextMetaByPath = { ...resourcePreviewMetaByPath.value }
    ;(resp.items || []).forEach((detail) => {
      const path = detail.full_code_path || ''
      if (!path) return
      const meta = buildResourcePreviewMetaFromDetail(detail, path)
      nextMetaByPath[path] = meta
      applyResourcePreviewMetaToTokens(meta)
      if (resourcePreview.value.visible && resourcePreview.value.path === path) {
        resourcePreview.value = {
          ...resourcePreview.value,
          ...meta,
          loading: false,
        }
      }
    })
    resourcePreviewMetaByPath.value = nextMetaByPath
  } catch {
    // 资源详情只是展示增强，失败时保留 token 上的基础信息。
  } finally {
    uniquePaths.forEach((path) => {
      resourcePreviewHydratingPaths.delete(path)
      if (resourcePreview.value.visible && resourcePreview.value.path === path) {
        resourcePreview.value = {
          ...resourcePreview.value,
          loading: false,
        }
      }
    })
  }
}

async function hydrateToolResourcePreviews(paths: string[]) {
  const uniquePaths = Array.from(new Set(paths))
    .filter(path => isWorkspaceToolResourcePath(path))
    .filter(path => !resourcePreviewMetaByPath.value[path] && !resourcePreviewHydratingPaths.has(path))
  if (uniquePaths.length === 0) return

  uniquePaths.forEach(path => resourcePreviewHydratingPaths.add(path))
  try {
    const resp = await batchGetWorkspaceToolDetails({
      names: uniquePaths.map(path => workspaceToolName(path)).filter(Boolean),
      include_schema: false,
    })
    const nextMetaByPath = { ...resourcePreviewMetaByPath.value }
    ;(resp.tools || []).forEach((tool) => {
      const toolPath = `tool:${tool.name}`
      const meta = buildResourcePreviewMetaFromTool(tool, toolPath)
      nextMetaByPath[toolPath] = meta
      applyResourcePreviewMetaToTokens(meta)
      if (resourcePreview.value.visible && resourcePreview.value.path === toolPath) {
        resourcePreview.value = {
          ...resourcePreview.value,
          ...meta,
          loading: false,
        }
      }
    })
    resourcePreviewMetaByPath.value = nextMetaByPath
  } catch {
    // 工具详情只是展示增强，失败时保留 token 上的基础信息。
  } finally {
    uniquePaths.forEach((path) => {
      resourcePreviewHydratingPaths.delete(path)
      if (resourcePreview.value.visible && resourcePreview.value.path === path) {
        resourcePreview.value = {
          ...resourcePreview.value,
          loading: false,
        }
      }
    })
  }
}

function scheduleResourceTokenHydration() {
  clearResourceTokenHydrateTimer()
  resourceTokenHydrateTimer = setTimeout(() => {
    void hydrateVisibleResourceTokens()
  }, 80)
}

function clearResourceTokenHydrateTimer() {
  if (!resourceTokenHydrateTimer) return
  clearTimeout(resourceTokenHydrateTimer)
  resourceTokenHydrateTimer = null
}

async function hydrateVisibleResourceTokens() {
  await nextTick()
  const container = markdownContentRef.value
  if (!container) return
  const links = Array.from(container.querySelectorAll<HTMLAnchorElement>('a.workspace-resource-token'))
  const paths = Array.from(new Set(
    links
      .map(link => link.dataset.fullCodePath || '')
      .filter(path => path && isHydratableResourcePath(path))
  )).slice(0, 30)

  const toolPaths = paths.filter(path => isWorkspaceToolResourcePath(path))
  if (toolPaths.length > 0) {
    toolPaths.forEach((path) => {
      const cachedMeta = resourcePreviewMetaByPath.value[path]
      if (cachedMeta) {
        applyResourcePreviewMetaToTokens(cachedMeta)
      }
    })
    void hydrateToolResourcePreviews(toolPaths)
  }

  const resourcePathsToHydrate: string[] = []
  paths.filter(path => !isWorkspaceToolResourcePath(path)).forEach((path) => {
    const cachedMeta = resourcePreviewMetaByPath.value[path]
    if (cachedMeta) {
      applyResourcePreviewMetaToTokens(cachedMeta)
      return
    }
    if (!resourcePreviewHydratingPaths.has(path)) {
      resourcePathsToHydrate.push(path)
    }
  })
  if (resourcePathsToHydrate.length > 0) {
    void hydrateWorkspaceResourcePreviews(resourcePathsToHydrate)
  }
}

function applyResourcePreviewMetaToTokens(meta: DocResourcePreviewMeta) {
  const container = markdownContentRef.value
  if (!container) return
  const links = Array.from(container.querySelectorAll<HTMLAnchorElement>('a.workspace-resource-token'))
    .filter(link => link.dataset.fullCodePath === meta.path)
  links.forEach((link) => {
    link.className = `workspace-resource-token is-${meta.kind}`
    link.dataset.resourceKind = meta.kind
    link.dataset.resourceLabel = meta.label
    link.dataset.resourceTypeLabel = meta.typeLabel
    link.dataset.resourceIconSrc = meta.iconSrc
    link.title = `${meta.label} · ${meta.path}`
    const labelEl = link.querySelector<HTMLElement>('.workspace-resource-token__label')
    if (labelEl) {
      labelEl.textContent = meta.label
    }
    const typeEl = link.querySelector<HTMLElement>('.workspace-resource-token__type')
    if (typeEl) {
      typeEl.textContent = meta.typeLabel
    }
    const iconEl = link.querySelector<HTMLElement>('.workspace-resource-token__icon')
    if (iconEl) {
      renderResourceTokenIcon(iconEl, meta)
    }
  })
}

function renderResourceTokenIcon(iconEl: HTMLElement, meta: DocResourcePreviewMeta) {
  iconEl.innerHTML = meta.iconHtml || getResourcePreviewIconHtml(meta.kind, meta.typeLabel)
}

function buildResourcePreviewMetaFromDetail(detail: ServiceTreeDetailResp, fallbackPath: string): DocResourcePreviewMeta {
  const path = detail.full_code_path || fallbackPath
  const kind = inferResourceKind(path, detail.type, detail.template_type)
  return {
    path,
    label: detail.name || detail.code || getPathTail(path) || path || '资源',
    typeLabel: getResourcePreviewTypeLabel(kind, detail.type, detail.template_type),
    kind,
    description: cleanPreviewText(detail.description),
    metaItems: compactPreviewMetaItems([
      getReadableResourcePath(path),
      detail.owner ? `负责人 ${detail.owner}` : '',
      detail.run_count ? `${formatPreviewCount(detail.run_count)} 次运行` : '',
      detail.tags ? `标签 ${detail.tags}` : '',
    ]),
    iconSrc: getResourcePreviewIconSrc(kind),
    iconHtml: getResourcePreviewIconHtml(kind, getResourcePreviewTypeLabel(kind, detail.type, detail.template_type)),
  }
}

function buildResourcePreviewMetaFromTool(tool: WorkspaceToolDetail, fallbackPath: string): DocResourcePreviewMeta {
  const fields = tool.input_fields || []
  const requiredFields = fields.filter(field => field.required).map(field => field.name).filter(Boolean)
  const previewFields = fields.slice(0, 4).map(formatToolFieldMeta).filter(Boolean)
  return {
    path: fallbackPath || `tool:${tool.name}`,
    label: tool.name || getPathTail(fallbackPath) || fallbackPath || '内置工具',
    typeLabel: tool.type_label || '内置工具',
    kind: 'tool',
    description: cleanPreviewText(tool.description),
    metaItems: compactPreviewMetaItems([
      fields.length > 0 ? `${fields.length} 个参数` : '',
      requiredFields.length > 0 ? `必填 ${requiredFields.slice(0, 4).join(', ')}` : '',
      previewFields.length > 0 ? `参数 ${previewFields.join(', ')}` : '',
    ]),
    iconSrc: getResourcePreviewIconSrc('tool'),
    iconHtml: getResourcePreviewIconHtml('tool', tool.type_label || '内置工具'),
  }
}

function getResourcePreviewPosition(resourceLink: HTMLAnchorElement) {
  const rect = resourceLink.getBoundingClientRect()
  const cardWidth = 340
  const estimatedCardHeight = 188
  const gap = 10
  const viewportWidth = typeof window === 'undefined' ? 1024 : window.innerWidth
  const viewportHeight = typeof window === 'undefined' ? 768 : window.innerHeight
  const left = Math.max(12, Math.min(rect.left, viewportWidth - cardWidth - 12))
  const bottomTop = rect.bottom + gap
  const top = bottomTop + estimatedCardHeight > viewportHeight
    ? Math.max(12, rect.top - estimatedCardHeight - gap)
    : bottomTop
  return { left, top }
}

function openResourcePreviewTarget() {
  const href = resourcePreview.value.href
  if (!href) return
  if (isWorkspaceToolResourcePath(resourcePreview.value.path)) {
    closeResourcePreview()
    return
  }
  closeResourcePreview()
  if (href.startsWith('/workspace/')) {
    void router.push(href)
    return
  }
  window.location.href = href
}

function scheduleCloseResourcePreview() {
  clearResourcePreviewCloseTimer()
  resourcePreviewCloseTimer = setTimeout(() => {
    closeResourcePreview()
  }, 180)
}

function cancelCloseResourcePreview() {
  clearResourcePreviewCloseTimer()
}

function handleDocumentMouseDown(e: MouseEvent) {
  if (!resourcePreview.value.visible) return
  const target = e.target
  if (isResourcePreviewFocusTarget(target) || getResourceLinkFromEventTarget(target)) {
    return
  }
  closeResourcePreview()
}

function handleWindowBlur() {
  closeResourcePreview()
}

function isResourcePreviewFocusTarget(target: EventTarget | null): boolean {
  return target instanceof Node && !!resourcePreviewRef.value?.contains(target)
}

function isMarkdownFocusTarget(target: EventTarget | null): boolean {
  return target instanceof Node && !!markdownContentRef.value?.contains(target)
}

function clearResourcePreviewCloseTimer() {
  if (!resourcePreviewCloseTimer) return
  clearTimeout(resourcePreviewCloseTimer)
  resourcePreviewCloseTimer = null
}

function closeResourcePreview() {
  clearResourcePreviewCloseTimer()
  resourcePreview.value = createEmptyResourcePreview()
}

function createEmptyResourcePreview(): DocResourcePreview {
  return {
    visible: false,
    path: '',
    href: '',
    label: '',
    typeLabel: '',
    kind: 'directory',
    description: '',
    metaItems: [],
    iconSrc: '',
    iconHtml: '',
    left: 0,
    top: 0,
    loading: false,
  }
}

function inferResourceKind(path: string, type?: string, templateType?: string) {
  if (isWorkspaceToolResourcePath(path)) return 'tool'
  if (type === 'docs' || path.endsWith('.docs')) return 'docs'
  if (type === 'package') return 'directory'
  if (templateType === 'form' || path.endsWith('.form')) return 'form'
  if (templateType === 'table' || path.endsWith('.table')) return 'table'
  if (templateType === 'chart' || path.endsWith('.chart')) return 'chart'
  return type === 'function' ? 'function' : 'directory'
}

function getResourcePreviewTypeLabel(kind: string, type?: string, templateType?: string) {
  if (type === 'docs' || kind === 'docs') return '文档'
  if (type === 'package' || kind === 'directory') return '目录'
  if (kind === 'tool') return '内置工具'
  if (templateType === 'table' || kind === 'table') return '表格'
  if (templateType === 'form' || kind === 'form') return '表单'
  if (templateType === 'chart' || kind === 'chart') return '图表'
  return '工具'
}

function getResourcePreviewIconSrc(kind: string) {
  return workspaceResourceIconSrc(kind)
}

function getResourcePreviewIconHtml(kind: string, typeLabel: string) {
  return workspaceResourceIconHtml(kind, typeLabel)
}

function getReadableResourcePath(path: string) {
  if (isWorkspaceToolResourcePath(path)) {
    return `内置工具 / ${workspaceToolName(path) || path}`
  }
  return String(path || '').split('/').filter(Boolean).join(' / ')
}

function formatToolFieldMeta(field: { name: string; type?: string }) {
  if (!field.name) return ''
  return field.type ? `${field.name}:${field.type}` : field.name
}

function getPathTail(path: string) {
  if (isWorkspaceToolResourcePath(path)) {
    return workspaceToolName(path)
  }
  return String(path || '').split('/').filter(Boolean).pop() || ''
}

function isHydratableResourcePath(path: string) {
  const value = String(path || '')
  return isWorkspaceToolResourcePath(value) || value.startsWith('/')
}

function cleanPreviewText(value?: string) {
  return String(value || '')
    .replace(/<[^>]*>/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}

function compactPreviewMetaItems(items: Array<string | undefined | null>) {
  return items
    .map(item => cleanPreviewText(item || ''))
    .filter((item, index, list) => item && list.indexOf(item) === index)
}

function formatPreviewCount(count: number) {
  if (count >= 10000) return `${(count / 10000).toFixed(count >= 100000 ? 0 : 1)}w`
  if (count >= 1000) return `${(count / 1000).toFixed(count >= 10000 ? 0 : 1)}k`
  return String(count)
}

function closeImagePreview() {
  imagePreviewVisible.value = false
  previewImgList.value = []
  previewIndex.value = 0
}

function onPreviewKeydown(e: KeyboardEvent) {
  if (resourcePreview.value.visible && e.key === 'Escape') {
    closeResourcePreview()
    return
  }
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
  document.addEventListener('mousedown', handleDocumentMouseDown)
  window.addEventListener('blur', handleWindowBlur)
})
onUnmounted(() => {
  document.removeEventListener('keydown', onPreviewKeydown)
  document.removeEventListener('mousedown', handleDocumentMouseDown)
  window.removeEventListener('blur', handleWindowBlur)
  clearResourcePreviewCloseTimer()
  clearResourceTokenHydrateTimer()
})

const canEditDoc = computed(() => true)

// 渲染后的 Markdown 内容
const renderedContent = computed(() => {
  if (!doc.value || !doc.value.content) {
    return ''
  }
  return renderMarkdown(renderWorkspaceResourceTokensAsHtml(doc.value.content, docResourceBasePath.value))
})

const docResourceBasePath = computed(() => getDocResourceBasePath(props.node?.full_code_path || ''))

function getDocResourceBasePath(fullCodePath: string): string {
  const path = String(fullCodePath || '').trim().replace(/\/+$/g, '')
  if (!path) return ''
  const parts = path.split('/').filter(Boolean)
  if (parts.length === 0) return ''
  parts.pop()
  return `/${parts.join('/')}`
}

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
  try {
    // ✅ 使用 full_code_path 调用新接口
    const data = await getDoc(props.node.full_code_path)
    doc.value = data || null
    if (doc.value && consumeDocAutoEdit(props.node.full_code_path)) {
      enterEditMode(doc.value)
    }
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

function enterEditMode(targetDoc = doc.value) {
  if (!targetDoc) return
  isEditing.value = true
  editSummary.value = targetDoc.summary || ''
  editContent.value = targetDoc.content || ''
}

// 创建文档
const handleCreate = () => {
  isEditing.value = true
  editSummary.value = ''
  editContent.value = ''
}

// 编辑文档
const handleEdit = () => {
  enterEditMode()
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
    isEditing.value = false
    closeResourcePreview()
    loadDoc()
  }
}, { immediate: true })

watch(renderedContent, () => {
  scheduleResourceTokenHydration()
}, { flush: 'post' })
</script>

<style scoped lang="scss">
.doc-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--bg-secondary, #f8fafc) 58%, transparent) 0%, transparent 180px),
    var(--bg-primary);
  padding: 28px 32px 44px;
  overflow-y: auto;
  box-sizing: border-box;
}

.doc-view__reader {
  width: 100%;
  max-width: 960px;
  margin: 0 auto;
  box-sizing: border-box;
}

.doc-view--editing {
  padding-inline: 28px;
}

.doc-view--editing .doc-view__reader {
  max-width: 1120px;
}

.doc-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: transparent;
  padding: 38px 8px 56px;
  box-sizing: border-box;
  width: 100%;
}

.doc-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 28px;
  margin-bottom: 34px;
  padding-bottom: 22px;
  border-bottom: 1px solid color-mix(in srgb, var(--border-base, #d8dee8) 70%, transparent);
}

.doc-title-section {
  flex: 1;
}

.doc-title {
  font-size: clamp(28px, 3vw, 36px);
  font-weight: 700;
  color: var(--el-text-color-primary, #111827);
  margin: 0 0 14px;
  line-height: 1.22;
  letter-spacing: 0;
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
  gap: 20px;
  margin-top: 12px;
  font-size: 14px;
  color: var(--el-text-color-secondary, #6b7280);
}

.doc-info-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.doc-info-item .el-icon {
  font-size: 15px;
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
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 220px;
}

.doc-actions :deep(.el-button) {
  height: 34px;
  padding-inline: 14px;
  border-radius: 7px;
  font-weight: 500;
}

.doc-summary {
  margin: 0 0 34px;
  padding: 2px 0 2px 18px;
  border-left: 3px solid color-mix(in srgb, var(--color-primary, #1677ff) 64%, var(--border-base, #d8dee8));
  
  p {
    margin: 0;
    color: var(--el-text-color-secondary, #4b5563);
    line-height: 1.75;
    font-size: 16px;
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
  gap: 14px;
  min-height: 0;
}

.doc-title-input {
  font-size: 18px;
  font-weight: 600;
  flex-shrink: 0; /* 标题输入框不缩小 */
}

.doc-summary-input {
  font-size: 14px;
  flex-shrink: 0;
}

.doc-summary-input :deep(.el-textarea__inner) {
  border-radius: 8px;
  box-shadow: none;
  border: 1px solid color-mix(in srgb, var(--border-base, #d8dee8) 78%, transparent);
  background: color-mix(in srgb, var(--app-shell-panel-bg, #fff) 86%, transparent);
  line-height: 1.6;
  padding: 12px 14px;
  transition: border-color 0.18s ease, box-shadow 0.18s ease, background 0.18s ease;
}

.doc-summary-input :deep(.el-textarea__inner:focus) {
  border-color: color-mix(in srgb, var(--color-primary, #1677ff) 64%, var(--border-base, #d8dee8));
  box-shadow: 0 0 0 3px rgba(var(--color-primary-rgb, 22, 119, 255), 0.1);
  background: var(--app-shell-panel-bg, #fff);
}

.doc-vditor-editor {
  flex: 1;
  min-height: clamp(560px, calc(100vh - 360px), 820px);
  display: flex;
  flex-direction: column;
}

.doc-preview {
  min-height: 400px;
}

.markdown-content {
  font-size: 16px;
  line-height: 1.75;
  color: var(--el-text-color-regular, #374151);
  word-break: break-word;

  :deep(h1), :deep(h2), :deep(h3), :deep(h4), :deep(h5), :deep(h6) {
    color: var(--el-text-color-primary, #111827);
    font-weight: 600;
    line-height: 1.3;
    margin-top: 2em;
    margin-bottom: 1em;
  }

  :deep(h1) {
    font-size: 2.25em;
    font-weight: 700;
    margin-top: 0;
    padding-bottom: 0.3em;
    border-bottom: 1px solid color-mix(in srgb, var(--border-base, #d8dee8) 70%, transparent);
  }

  :deep(h2) {
    font-size: 1.5em;
    padding-bottom: 0.3em;
    border-bottom: 1px solid color-mix(in srgb, var(--border-base, #d8dee8) 62%, transparent);
  }

  :deep(h3) {
    font-size: 1.25em;
  }

  :deep(h4) {
    font-size: 1em;
  }

  :deep(p) {
    margin-top: 1.25em;
    margin-bottom: 1.25em;
  }

  :deep(ul), :deep(ol) {
    margin-top: 1.25em;
    margin-bottom: 1.25em;
    padding-left: 1.625em;
  }

  :deep(li) {
    margin-top: 0.5em;
    margin-bottom: 0.5em;
  }

  :deep(code) {
    background: var(--el-fill-color-light, #f3f4f6);
    color: var(--el-color-primary, #0369a1);
    padding: 0.2em 0.4em;
    border-radius: 6px;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
    font-size: 0.875em;
    font-weight: 500;
  }

  :deep(pre) {
    background: #1f2937; /* 深灰/黑 背景 */
    color: #e5e7eb;
    padding: 1.25em 1.5em;
    border-radius: 8px;
    overflow-x: auto;
    margin-top: 1.7em;
    margin-bottom: 1.7em;
    font-size: 0.875em;
    line-height: 1.7142857;
    max-width: 100%;
    box-shadow: inset 0 0 0 1px rgba(255,255,255,0.1);

    code {
      background: transparent;
      color: inherit;
      padding: 0;
      font-weight: 400;
      font-size: inherit;
      border-radius: 0;
    }
  }

  :deep(blockquote) {
    font-weight: 400;
    font-style: normal;
    color: var(--el-text-color-secondary, #4b5563);
    border-left: 3px solid color-mix(in srgb, var(--color-primary, #1677ff) 42%, var(--border-base, #d1d5db));
    padding-left: 1rem;
    margin-top: 1.6em;
    margin-bottom: 1.6em;
    background: transparent;
  }

  :deep(.workspace-resource-token) {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    max-width: 100%;
    vertical-align: baseline;
    padding: 1px 7px 1px 5px;
    margin: 0 1px;
    border: 1px solid color-mix(in srgb, var(--el-color-primary, #1677ff) 24%, var(--el-border-color, #dcdfe6));
    border-radius: 7px;
    background: color-mix(in srgb, var(--el-color-primary, #1677ff) 8%, var(--el-bg-color, #fff));
    color: var(--el-text-color-primary, #111827);
    font-size: 0.92em;
    line-height: 1.55;
    font-weight: 500;
    text-decoration: none;
    white-space: nowrap;
    cursor: pointer;
    transition: border-color 0.16s ease, background 0.16s ease, color 0.16s ease, box-shadow 0.16s ease;

    &:hover,
    &:focus-visible {
      border-color: color-mix(in srgb, var(--el-color-primary, #1677ff) 58%, var(--el-border-color, #dcdfe6));
      background: color-mix(in srgb, var(--el-color-primary, #1677ff) 13%, var(--el-bg-color, #fff));
      color: var(--el-color-primary, #1677ff);
      box-shadow: 0 0 0 3px rgba(var(--color-primary-rgb, 22, 119, 255), 0.08);
      outline: none;
      text-decoration: none;
    }
  }

  :deep(.workspace-resource-token__icon) {
    width: 16px;
    height: 16px;
    flex: 0 0 auto;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
    opacity: 0.9;
  }

  :deep(.workspace-resource-token__img),
  :deep(.workspace-resource-token__svg),
  :deep(.workspace-resource-token__glyph) {
    width: 16px;
    height: 16px;
    object-fit: contain;
    display: block;
    margin: 0;
    border-radius: 0;
    box-shadow: none;
    cursor: inherit;
    transition: none;
  }

  :deep(.workspace-resource-token__glyph) {
    border-radius: 5px;
    background: linear-gradient(135deg, var(--el-color-primary, #1677ff), color-mix(in srgb, var(--el-color-primary, #1677ff) 58%, white));
    position: relative;
  }

  :deep(.workspace-resource-token__glyph::after) {
    content: '';
    position: absolute;
    inset: 4px;
    border: 2px solid rgba(255, 255, 255, 0.9);
    border-radius: 999px;
  }

  :deep(.workspace-resource-token__img:hover),
  :deep(.workspace-resource-token__svg:hover) {
    transform: none;
    box-shadow: none;
  }

  :deep(.workspace-resource-token__html-icon) {
    display: inline-flex;
    width: 16px;
    height: 16px;
    align-items: center;
    justify-content: center;
  }

  :deep(.workspace-resource-token__label) {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  :deep(.workspace-resource-token__type) {
    color: var(--el-text-color-secondary, #6b7280);
    font-size: 0.82em;
    font-weight: 600;
  }

  :deep(a) {
    color: var(--el-color-primary);
    text-decoration: underline;
    text-decoration-color: transparent;
    font-weight: 500;
    transition: text-decoration-color 0.2s ease;

    &:hover {
      text-decoration-color: currentColor;
    }
  }

  :deep(table) {
    width: 100%;
    border-collapse: collapse;
    margin-top: 2em;
    margin-bottom: 2em;
    font-size: 0.875em;
    display: block;
    overflow-x: auto;
    white-space: nowrap;

    th, td {
      border: 1px solid var(--el-border-color-lighter, #e5e7eb);
      padding: 0.75em 1em;
      text-align: left;
    }

    th {
      background: var(--el-fill-color-light, #f9fafb);
      color: var(--el-text-color-primary, #111827);
      font-weight: 600;
    }

    tr:nth-child(even) {
      background: var(--el-fill-color-blank, #ffffff);
    }
  }

  :deep(img) {
    max-width: 100%;
    height: auto;
    border-radius: 8px;
    margin-top: 2em;
    margin-bottom: 2em;
    cursor: pointer;
    box-shadow: 0 14px 36px -24px rgba(15, 23, 42, 0.55);
    transition: transform 0.2s ease, box-shadow 0.2s ease;
    display: block;

    &:hover {
      transform: translateY(-2px);
      box-shadow: 0 18px 42px -22px rgba(15, 23, 42, 0.62);
    }
  }

  :deep(video) {
    max-width: 100%;
    height: auto;
    border-radius: 8px;
    margin-top: 2em;
    margin-bottom: 2em;
  }

  :deep(hr) {
    border: none;
    border-top: 1px solid var(--el-border-color-lighter, #e5e7eb);
    margin-top: 3em;
    margin-bottom: 3em;
  }
}

.plain-text-content {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 14px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-wrap: break-word;
  background: var(--el-fill-color-light, #f9fafb);
  border: 1px solid var(--el-border-color-lighter, #e5e7eb);
  color: var(--el-text-color-regular, #374151);
  padding: 24px;
  border-radius: 8px;
}

.doc-resource-preview {
  position: fixed;
  z-index: var(--aos-z-global-overlay, 4200);
  width: min(340px, calc(100vw - 24px));
  box-sizing: border-box;
  border: 1px solid color-mix(in srgb, var(--el-color-primary, #1677ff) 22%, var(--el-border-color, #dcdfe6));
  border-radius: 10px;
  padding: 12px;
  background: color-mix(in srgb, var(--el-bg-color, #fff) 96%, transparent);
  box-shadow: 0 18px 46px -22px rgba(15, 23, 42, 0.48), 0 0 0 1px rgba(255, 255, 255, 0.78) inset;
  color: var(--el-text-color-primary, #111827);
}

.doc-resource-preview__head {
  display: grid;
  grid-template-columns: 40px minmax(0, 1fr);
  gap: 10px;
  align-items: center;
}

.doc-resource-preview__icon {
  width: 40px;
  height: 40px;
  border: 1px solid color-mix(in srgb, var(--el-color-primary, #1677ff) 22%, var(--el-border-color, #dcdfe6));
  border-radius: 9px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--el-color-primary, #1677ff) 8%, var(--el-bg-color, #fff));
  color: var(--el-color-primary, #1677ff);
  font-weight: 700;
}

.doc-resource-preview.is-docs .doc-resource-preview__icon {
  border-color: color-mix(in srgb, var(--el-color-info, #909399) 28%, var(--el-border-color, #dcdfe6));
  background: color-mix(in srgb, var(--el-color-info, #909399) 10%, var(--el-bg-color, #fff));
  color: var(--el-text-color-secondary, #606266);
}

.doc-resource-preview.is-form .doc-resource-preview__icon {
  border-color: color-mix(in srgb, var(--el-color-success, #67c23a) 34%, var(--el-border-color, #dcdfe6));
  background: color-mix(in srgb, var(--el-color-success, #67c23a) 10%, var(--el-bg-color, #fff));
  color: var(--el-color-success, #67c23a);
}

.doc-resource-preview.is-table .doc-resource-preview__icon {
  border-color: color-mix(in srgb, #10b981 34%, var(--el-border-color, #dcdfe6));
  background: color-mix(in srgb, #10b981 10%, var(--el-bg-color, #fff));
  color: #059669;
}

.doc-resource-preview.is-chart .doc-resource-preview__icon {
  border-color: color-mix(in srgb, #8b5cf6 34%, var(--el-border-color, #dcdfe6));
  background: color-mix(in srgb, #8b5cf6 10%, var(--el-bg-color, #fff));
  color: #7c3aed;
}

.doc-resource-preview__img {
  width: 22px;
  height: 22px;
  object-fit: contain;
  display: block;
}

.doc-resource-preview__html-icon,
.doc-resource-preview__html-icon :deep(.workspace-resource-token__icon) {
  width: 22px;
  height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.doc-resource-preview__html-icon :deep(.workspace-resource-token__img),
.doc-resource-preview__html-icon :deep(.workspace-resource-token__svg),
.doc-resource-preview__html-icon :deep(.workspace-resource-token__glyph) {
  width: 22px;
  height: 22px;
  display: block;
}

.doc-resource-preview__main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.doc-resource-preview__main strong,
.doc-resource-preview__main span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.doc-resource-preview__main strong {
  font-size: 14px;
  line-height: 1.35;
}

.doc-resource-preview__main span {
  color: var(--el-text-color-secondary, #6b7280);
  font-size: 12px;
  line-height: 1.35;
  font-weight: 600;
}

.doc-resource-preview__desc {
  margin: 10px 0 0;
  color: var(--el-text-color-regular, #374151);
  font-size: 12px;
  line-height: 1.55;
}

.doc-resource-preview__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}

.doc-resource-preview__meta-item {
  max-width: 100%;
  overflow: hidden;
  border-radius: 6px;
  padding: 2px 6px;
  background: var(--el-fill-color-light, #f5f7fa);
  color: var(--el-text-color-secondary, #6b7280);
  font-size: 11px;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.doc-resource-preview__path {
  display: block;
  margin-top: 10px;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter, #e5e7eb);
  border-radius: 7px;
  padding: 6px 7px;
  background: var(--el-fill-color-light, #f5f7fa);
  color: var(--el-text-color-secondary, #6b7280);
  font-size: 11px;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.doc-resource-preview__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 10px;
}

.doc-resource-preview__loading {
  margin-right: auto;
  color: var(--el-text-color-secondary, #6b7280);
  font-size: 12px;
}

.doc-resource-preview__link,
.doc-resource-preview__close {
  border: 0;
  border-radius: 7px;
  padding: 6px 9px;
  font-size: 12px;
  font-weight: 600;
  line-height: 1;
  cursor: pointer;
}

.doc-resource-preview__link {
  background: var(--el-color-primary, #1677ff);
  color: #fff;
}

.doc-resource-preview__close {
  background: var(--el-fill-color, #f0f2f5);
  color: var(--el-text-color-regular, #374151);
}

.doc-resource-preview-enter-active,
.doc-resource-preview-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
}

.doc-resource-preview-enter-from,
.doc-resource-preview-leave-to {
  opacity: 0;
  transform: translateY(4px);
}

.doc-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}

@media (max-width: 768px) {
  .doc-view {
    padding: 18px 16px 32px;
  }

  .doc-content {
    padding: 22px 0 40px;
  }

  .doc-header {
    flex-direction: column;
    gap: 18px;
    margin-bottom: 26px;
  }

  .doc-actions {
    width: 100%;
    min-width: 0;
    justify-content: flex-start;
  }

  .doc-title {
    font-size: 28px;
  }

  .doc-vditor-editor {
    min-height: 560px;
  }
}

/* 图片预览弹层：层级由全局浮层协议控制 */
.doc-image-preview {
  position: fixed;
  inset: 0;
  z-index: var(--aos-z-critical-preview);
  background: rgba(0, 0, 0, 0.85);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px;

  &__close {
    position: absolute;
    top: 24px;
    right: 24px;
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: rgba(255, 255, 255, 0.1);
    color: #fff;
    border-radius: 50%;
    cursor: pointer;
    transition: background 0.2s ease;

    &:hover {
      background: rgba(255, 255, 255, 0.2);
    }
  }

  &__nav {
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
    width: 56px;
    height: 56px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: rgba(255, 255, 255, 0.1);
    color: #fff;
    border-radius: 50%;
    cursor: pointer;
    transition: background 0.2s ease;

    &:hover {
      background: rgba(255, 255, 255, 0.2);
    }

    &--prev {
      left: 32px;
    }
    &--next {
      right: 32px;
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
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
  }

  &__indicator {
    position: absolute;
    bottom: 32px;
    left: 50%;
    transform: translateX(-50%);
    padding: 8px 16px;
    background: rgba(0, 0, 0, 0.6);
    color: #fff;
    border-radius: 20px;
    font-size: 14px;
    font-variant-numeric: tabular-nums;
  }
}

.doc-image-preview-enter-active,
.doc-image-preview-leave-active {
  transition: opacity 0.25s ease;
}
.doc-image-preview-enter-from,
.doc-image-preview-leave-to {
  opacity: 0;
}
</style>
