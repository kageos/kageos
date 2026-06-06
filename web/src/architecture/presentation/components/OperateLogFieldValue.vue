<template>
  <div class="operate-log-field-value" :class="{ 'is-compact': compact }">
    <WidgetComponent
      v-if="field"
      :field="field"
      :value="fieldValue"
      :model-value="fieldValue"
      :field-path="resolvedFieldPath"
      :mode="mode"
    />
    <span v-else class="operate-log-field-fallback">{{ fallbackText }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import type { WidgetMode } from '@/architecture/presentation/widgets/types'
import { WidgetType } from '@/architecture/domain/constants/widget'
import { convertToFieldValue } from '@/architecture/domain/utils/field'
import WidgetComponent from '@/architecture/presentation/widgets/WidgetComponent.vue'

const props = withDefaults(defineProps<{
  field?: FieldConfig | null
  rawValue: unknown
  mode?: WidgetMode
  fieldPath?: string
  emptyText?: string
  compact?: boolean
}>(), {
  field: null,
  mode: 'detail',
  fieldPath: '',
  emptyText: '-',
  compact: false,
})

const resolvedFieldPath = computed(() => props.fieldPath || props.field?.field_path || props.field?.code || '')

const fieldValue = computed<FieldValue>(() => {
  if (!props.field) {
    return { raw: props.rawValue ?? null, display: fallbackText.value, meta: {} }
  }
  return convertToFieldValue(normalizeLogRawValue(props.rawValue, props.field), props.field)
})

const fallbackText = computed(() => formatFallbackValue(props.rawValue, props.emptyText))

function isRecord(value: unknown): value is Record<string, any> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function isFieldValue(value: unknown): value is FieldValue {
  return isRecord(value) && 'raw' in value && 'display' in value
}

function normalizeLogRawValue(rawValue: unknown, field: FieldConfig): unknown {
  if (isFieldValue(rawValue)) {
    if (field.widget?.type === WidgetType.FILES) {
      return {
        ...rawValue,
        raw: normalizeFilesRawValue(rawValue.raw),
      }
    }
    return rawValue
  }
  if (field.widget?.type !== WidgetType.FILES) {
    return rawValue
  }
  return normalizeFilesRawValue(rawValue)
}

function normalizeFilesRawValue(rawValue: unknown): unknown {
  if (rawValue === null || rawValue === undefined || rawValue === '') {
    return rawValue
  }
  if (typeof rawValue === 'string') {
    return rawValue
  }

  const refs = collectFileRefs(rawValue)
  return refs.length > 0 ? refs.join(',') : rawValue
}

function collectFileRefs(value: unknown): string[] {
  if (typeof value === 'string') {
    const trimmed = value.trim().replace(/^\/+/, '')
    return trimmed ? [trimmed] : []
  }
  if (Array.isArray(value)) {
    return value.flatMap(collectFileRefs)
  }
  if (!isRecord(value)) {
    return []
  }
  if (Array.isArray(value.files)) {
    return collectFileRefs(value.files)
  }

  const ref = value.ref || value.file_ref || value.fileRef || value.path
  if (typeof ref === 'string' && ref.trim()) {
    return [ref.trim().replace(/^\/+/, '')]
  }
  if (typeof value.bucket === 'string' && typeof value.key === 'string' && value.key.trim()) {
    return [`${value.bucket.replace(/\/+$/, '')}/${value.key.replace(/^\/+/, '')}`]
  }
  return []
}

function formatFallbackValue(value: unknown, emptyText: string): string {
  if (value === null || value === undefined || value === '') {
    return emptyText
  }
  if (typeof value === 'boolean') {
    return value ? '是' : '否'
  }
  if (typeof value === 'number') {
    return String(value)
  }
  if (typeof value === 'string') {
    return value
  }
  if (Array.isArray(value)) {
    if (value.length === 0) {
      return emptyText
    }
    if (value.every((item) => ['string', 'number', 'boolean'].includes(typeof item))) {
      return value.map((item) => formatFallbackValue(item, emptyText)).join('、')
    }
    return `${value.length} 项`
  }
  if (isRecord(value)) {
    if (Array.isArray(value.files)) {
      return `${value.files.length} 个文件`
    }
    for (const key of ['name', 'title', 'label', 'text', 'value']) {
      if (typeof value[key] === 'string' && value[key]) {
        return value[key]
      }
    }
    try {
      return JSON.stringify(value)
    } catch {
      return String(value)
    }
  }
  return String(value)
}
</script>

<style scoped>
.operate-log-field-value {
  min-width: 0;
  max-width: 100%;
  overflow-wrap: anywhere;
  word-break: break-word;
}

.operate-log-field-value.is-compact {
  font-size: 13px;
  line-height: 1.5;
}

.operate-log-field-value :deep(.rich-text-widget),
.operate-log-field-value :deep(.files-widget),
.operate-log-field-value :deep(.form-widget),
.operate-log-field-value :deep(.table-widget) {
  max-width: 100%;
}

.operate-log-field-value :deep(.detail-value),
.operate-log-field-value :deep(.response-value),
.operate-log-field-value :deep(.detail-files) {
  max-width: 100%;
}

.operate-log-field-value :deep(.html-content) {
  max-width: 100%;
  overflow-x: auto;
}

.operate-log-field-value :deep(.uploaded-files),
.operate-log-field-value :deep(.files-list) {
  max-width: 100%;
}

.operate-log-field-value.is-compact :deep(.files-list) {
  gap: 8px;
}

.operate-log-field-value.is-compact :deep(.file-card-shell) {
  border-radius: 8px;
}

.operate-log-field-value.is-compact :deep(.file-list-item) {
  gap: 8px;
  padding: 9px 10px;
}

.operate-log-field-value.is-compact :deep(.file-card-main) {
  grid-template-columns: 44px minmax(0, 1fr);
  gap: 9px;
}

.operate-log-field-value.is-compact :deep(.file-thumbnail) {
  width: 44px;
  height: 44px;
  padding: 3px;
}

.operate-log-field-value.is-compact :deep(.thumbnail-icon) {
  font-size: 24px !important;
}

.operate-log-field-value.is-compact :deep(.file-name) {
  font-size: 13px;
  line-height: 1.4;
}

.operate-log-field-value.is-compact :deep(.file-card-footer) {
  align-items: stretch;
  flex-direction: column;
  gap: 7px;
}

.operate-log-field-value.is-compact :deep(.file-actions) {
  width: 100%;
  max-width: none;
  justify-content: flex-end;
}

.operate-log-field-fallback {
  color: var(--el-text-color-primary);
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
}
</style>
