import type { Component } from 'vue'
import {
  Picture,
  DataAnalysis,
  DocumentCopy,
  Files,
  Film,
  Headset,
  Tickets,
  Document as DocumentIcon
} from '@element-plus/icons-vue'
import type { ChatMessageToolCall } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import type { OutputDisplayField } from '@/architecture/presentation/composables/useOutputDisplayFields'
import type { FilePanelItem } from './useMiniWorkstationPanel'

export type MiniArtifactTone = 'image' | 'data' | 'document' | 'media' | 'archive' | 'field' | 'file'

export interface MiniArtifactItem {
  key: string
  name: string
  meta: string
  tag: string
  ext: string
  tone: MiniArtifactTone
  iconComponent: Component
  previewUrl?: string
  file?: FilePanelItem
  field?: OutputDisplayField
}

const GENERATED_ARTIFACT_TOOL_NAMES = new Set([
  'write_prd',
  'build_workspace',
  'write_go_file',
  'write_doc',
  'create_directory',
  'copy_directory'
])

const ARTIFACT_IMAGE_EXTENSIONS = new Set(['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'svg', 'avif'])
const ARTIFACT_DATA_EXTENSIONS = new Set(['csv', 'xlsx', 'xls', 'json', 'xml', 'yaml', 'yml'])
const ARTIFACT_DOCUMENT_EXTENSIONS = new Set(['md', 'txt', 'doc', 'docx', 'pdf', 'ppt', 'pptx'])
const ARTIFACT_VIDEO_EXTENSIONS = new Set(['mp4', 'mov', 'webm', 'avi', 'mkv'])
const ARTIFACT_AUDIO_EXTENSIONS = new Set(['mp3', 'wav', 'aac', 'flac', 'm4a'])
const ARTIFACT_ARCHIVE_EXTENSIONS = new Set(['zip', 'rar', '7z', 'tar', 'gz'])

export function isGeneratedArtifactToolCall(call: ChatMessageToolCall): boolean {
  if (call.status !== 'ok' && call.status !== 'success') return false
  if (GENERATED_ARTIFACT_TOOL_NAMES.has(call.name)) return true
  if (call.metadata?.display_file_fields?.length) return true
  return resultDataLooksLikeArtifact(call.result_data)
}

function resultDataLooksLikeArtifact(resultData: unknown): boolean {
  if (!resultData || typeof resultData !== 'object') return false
  const kind = String((resultData as { kind?: unknown }).kind || '').trim()
  return kind.startsWith('agent_app_') || kind.startsWith('workspace_')
}

export function buildFileArtifactItem(file: FilePanelItem, index: number): MiniArtifactItem {
  const ext = getArtifactExtension(file.name)
  const profile = getFileArtifactProfile(file)
  const extLabel = ext ? ext.toUpperCase() : ''

  return {
    key: `file:${file.href}:${index}`,
    name: file.name,
    meta: [extLabel, '输出文件'].filter(Boolean).join(' · '),
    tag: profile.tag,
    ext: extLabel,
    tone: profile.tone,
    iconComponent: profile.iconComponent,
    previewUrl: profile.previewUrl,
    file
  }
}

export function buildDisplayFieldArtifactItem(field: OutputDisplayField, index: number): MiniArtifactItem {
  return {
    key: `field:${field.label}:${index}`,
    name: field.label,
    meta: truncateOneLine(field.value || '展示字段'),
    tag: '字段',
    ext: '',
    tone: 'field',
    iconComponent: Tickets,
    field
  }
}

export function truncateOneLine(value: string, max = 36): string {
  const text = String(value || '').replace(/\s+/g, ' ').trim()
  return text.length > max ? `${text.slice(0, max)}…` : text
}

function getArtifactExtension(name: string): string {
  return (name || '').split('?')[0]?.split('#')[0]?.split('.').pop()?.toLowerCase() || ''
}

function getFileArtifactProfile(file: FilePanelItem): {
  tag: string
  tone: MiniArtifactTone
  iconComponent: Component
  previewUrl?: string
} {
  const ext = getArtifactExtension(file.name)
  if (ARTIFACT_IMAGE_EXTENSIONS.has(ext)) {
    return { tag: '图片', tone: 'image', iconComponent: Picture, previewUrl: file.href }
  }
  if (ARTIFACT_DATA_EXTENSIONS.has(ext)) {
    return { tag: '数据', tone: 'data', iconComponent: DataAnalysis }
  }
  if (ARTIFACT_DOCUMENT_EXTENSIONS.has(ext)) {
    return { tag: '文档', tone: 'document', iconComponent: DocumentCopy }
  }
  if (ARTIFACT_VIDEO_EXTENSIONS.has(ext)) {
    return { tag: '视频', tone: 'media', iconComponent: Film }
  }
  if (ARTIFACT_AUDIO_EXTENSIONS.has(ext)) {
    return { tag: '音频', tone: 'media', iconComponent: Headset }
  }
  if (ARTIFACT_ARCHIVE_EXTENSIONS.has(ext)) {
    return { tag: '压缩包', tone: 'archive', iconComponent: Files }
  }
  return { tag: '文件', tone: 'file', iconComponent: DocumentIcon }
}
