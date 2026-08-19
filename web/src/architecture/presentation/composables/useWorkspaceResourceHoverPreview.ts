import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { Router } from 'vue-router'
import type { ServiceTree } from '@/architecture/domain/types'
import {
  batchGetServiceTreeDetails,
  type ServiceTreeDetailResp
} from '@/architecture/presentation/context/api/service-tree'
import {
  batchGetWorkspaceToolDetails,
  type WorkspaceToolDetail
} from '@/architecture/presentation/context/api/workspace'
import {
  isWorkspaceToolResourcePath,
  workspaceResourceIconHtml,
  workspaceResourceIconSrc,
  workspaceToolName,
} from '@/architecture/presentation/components/utils/workspaceInvocationSnippet'
import { resolveWorkspaceUrl } from '@/architecture/shared/routing/route'

export interface WorkspaceResourcePreviewMeta {
  path: string
  label: string
  typeLabel: string
  kind: string
  description: string
  metaItems: string[]
  iconSrc: string
  iconHtml: string
}

export interface WorkspaceResourcePreview extends WorkspaceResourcePreviewMeta {
  visible: boolean
  href: string
  left: number
  top: number
  loading: boolean
}

const previewMetaCache: Record<string, WorkspaceResourcePreviewMeta> = {}

export function useWorkspaceResourceHoverPreview() {
  const resourcePreview = ref<WorkspaceResourcePreview>(createEmptyResourcePreview())
  const hydratingPaths = new Set<string>()
  let closeTimer: ReturnType<typeof setTimeout> | null = null

  onMounted(() => {
    if (typeof window !== 'undefined') {
      window.addEventListener('blur', closeResourcePreview)
    }
    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', closeResourcePreviewOnHidden)
    }
  })

  onBeforeUnmount(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('blur', closeResourcePreview)
    }
    if (typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', closeResourcePreviewOnHidden)
    }
  })

  function showResourcePreviewFromEvent(e: MouseEvent | FocusEvent) {
    const root = e.currentTarget instanceof HTMLElement ? e.currentTarget : null
    const link = getResourceLinkFromTarget(e.target, root)
    if (!link) return
    showResourcePreviewFromLink(link)
  }

  function showResourcePreviewFromLink(link: HTMLAnchorElement) {
    const path = link.dataset.fullCodePath || ''
    if (!path) return
    showResourcePreview({
      meta: previewMetaCache[path] || buildPreviewMetaFromLink(link),
      target: link,
      href: link.getAttribute('href') || '',
      loading: !previewMetaCache[path] && isHydratableResourcePath(path),
    })
    if (!previewMetaCache[path] && isHydratableResourcePath(path)) {
      void hydrateResourcePreviews([path])
    }
  }

  function showResourcePreviewFromNode(node: ServiceTree, target: HTMLElement) {
    const path = node.full_code_path || ''
    if (!path) return
    showResourcePreview({
      meta: previewMetaCache[path] || buildPreviewMetaFromServiceTree(node),
      target,
      href: resolveWorkspaceUrl(path),
      loading: !previewMetaCache[path] && isHydratableResourcePath(path),
    })
    if (!previewMetaCache[path] && isHydratableResourcePath(path)) {
      void hydrateResourcePreviews([path])
    }
  }

  async function hydrateResourcePreviews(paths: string[]) {
    const uniquePaths = Array.from(new Set(paths))
      .filter(path => path && isHydratableResourcePath(path))
      .filter(path => !previewMetaCache[path] && !hydratingPaths.has(path))
    if (uniquePaths.length === 0) return

    uniquePaths.forEach(path => hydratingPaths.add(path))
    const toolPaths = uniquePaths.filter(path => isWorkspaceToolResourcePath(path))
    const resourcePaths = uniquePaths.filter(path => !isWorkspaceToolResourcePath(path))
    try {
      await Promise.all([
        hydrateWorkspaceResourcePreviews(resourcePaths),
        hydrateToolResourcePreviews(toolPaths),
      ])
    } catch {
      // Hover 预览是辅助信息，加载失败时保留已有的浅层信息即可。
    } finally {
      uniquePaths.forEach((path) => {
        hydratingPaths.delete(path)
        if (resourcePreview.value.visible && resourcePreview.value.path === path) {
          resourcePreview.value = {
            ...resourcePreview.value,
            ...(previewMetaCache[path] || {}),
            loading: false,
          }
        }
      })
    }
  }

  function showResourcePreview(input: {
    meta: WorkspaceResourcePreviewMeta
    target: HTMLElement
    href: string
    loading: boolean
  }) {
    cancelCloseResourcePreview()
    const position = getResourcePreviewPosition(input.target)
    resourcePreview.value = {
      ...input.meta,
      visible: true,
      href: input.href,
      left: position.left,
      top: position.top,
      loading: input.loading,
    }
  }

  function scheduleCloseResourcePreview() {
    clearCloseTimer()
    closeTimer = setTimeout(closeResourcePreview, 180)
  }

  function cancelCloseResourcePreview() {
    clearCloseTimer()
  }

  function closeResourcePreview() {
    clearCloseTimer()
    resourcePreview.value = createEmptyResourcePreview()
  }

  function closeResourcePreviewOnHidden() {
    if (typeof document !== 'undefined' && document.hidden) {
      closeResourcePreview()
    }
  }

  function openResourcePreviewTarget(router?: Router) {
    const href = resourcePreview.value.href
    if (!href || isWorkspaceToolResourcePath(resourcePreview.value.path)) {
      closeResourcePreview()
      return
    }
    closeResourcePreview()
    if (href.startsWith('/workspace/') && router) {
      void router.push(href)
      return
    }
    window.location.href = href
  }

  function clearCloseTimer() {
    if (!closeTimer) return
    clearTimeout(closeTimer)
    closeTimer = null
  }

  return {
    resourcePreview,
    showResourcePreviewFromEvent,
    showResourcePreviewFromLink,
    showResourcePreviewFromNode,
    scheduleCloseResourcePreview,
    cancelCloseResourcePreview,
    closeResourcePreview,
    openResourcePreviewTarget,
  }
}

async function hydrateWorkspaceResourcePreviews(paths: string[]) {
  if (paths.length === 0) return
  const resp = await batchGetServiceTreeDetails({ full_code_paths: paths })
  ;(resp.items || []).forEach((detail) => {
    const path = detail.full_code_path || ''
    if (!path) return
    previewMetaCache[path] = buildPreviewMetaFromDetail(detail, path)
  })
}

async function hydrateToolResourcePreviews(paths: string[]) {
  if (paths.length === 0) return
  const resp = await batchGetWorkspaceToolDetails({
    names: paths.map(path => workspaceToolName(path)).filter(Boolean),
    include_schema: false,
  })
  ;(resp.tools || []).forEach((tool) => {
    const path = `tool:${tool.name}`
    previewMetaCache[path] = buildPreviewMetaFromTool(tool, path)
  })
}

function createEmptyResourcePreview(): WorkspaceResourcePreview {
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

function getResourceLinkFromTarget(target: EventTarget | null, root: HTMLElement | null): HTMLAnchorElement | null {
  if (!(target instanceof Element)) return null
  const link = target.closest('a.workspace-resource-token') as HTMLAnchorElement | null
  if (!link || (root && !root.contains(link))) return null
  return link
}

function buildPreviewMetaFromLink(link: HTMLAnchorElement): WorkspaceResourcePreviewMeta {
  const path = link.dataset.fullCodePath || ''
  const kind = link.dataset.resourceKind || inferResourceKind(path)
  return {
    path,
    label: link.dataset.resourceLabel || getPathTail(path) || path || '资源',
    typeLabel: link.dataset.resourceTypeLabel || getResourcePreviewTypeLabel(kind),
    kind,
    description: '',
    metaItems: compactPreviewMetaItems([getReadableResourcePath(path)]),
    iconSrc: getResourcePreviewIconSrc(kind),
    iconHtml: getResourcePreviewIconHtml(kind, link.dataset.resourceTypeLabel || getResourcePreviewTypeLabel(kind)),
  }
}

function buildPreviewMetaFromServiceTree(node: ServiceTree): WorkspaceResourcePreviewMeta {
  const path = node.full_code_path || ''
  const kind = inferResourceKind(path, node.type, node.template_type)
  return {
    path,
    label: node.name || node.code || getPathTail(path) || path || '资源',
    typeLabel: getResourcePreviewTypeLabel(kind, node.type, node.template_type),
    kind,
    description: cleanPreviewText(node.description),
    metaItems: compactPreviewMetaItems([
      getReadableResourcePath(path),
      node.run_count ? `${formatPreviewCount(node.run_count)} 次运行` : '',
      node.tags ? `标签 ${node.tags}` : '',
    ]),
    iconSrc: getResourcePreviewIconSrc(kind),
    iconHtml: getResourcePreviewIconHtml(kind, getResourcePreviewTypeLabel(kind, node.type, node.template_type)),
  }
}

function buildPreviewMetaFromDetail(detail: ServiceTreeDetailResp, fallbackPath: string): WorkspaceResourcePreviewMeta {
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
      detail.run_count ? `${formatPreviewCount(detail.run_count)} 次运行` : '',
      detail.tags ? `标签 ${detail.tags}` : '',
    ]),
    iconSrc: getResourcePreviewIconSrc(kind),
    iconHtml: getResourcePreviewIconHtml(kind, getResourcePreviewTypeLabel(kind, detail.type, detail.template_type)),
  }
}

function buildPreviewMetaFromTool(tool: WorkspaceToolDetail, fallbackPath: string): WorkspaceResourcePreviewMeta {
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

function getResourcePreviewPosition(target: HTMLElement) {
  const rect = target.getBoundingClientRect()
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
  if (type === 'package' || kind === 'directory') return '服务目录'
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

function formatToolFieldMeta(field: { name: string; type?: string }) {
  if (!field.name) return ''
  return field.type ? `${field.name}:${field.type}` : field.name
}
