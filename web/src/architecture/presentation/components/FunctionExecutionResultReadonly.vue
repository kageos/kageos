<template>
  <div class="function-execution-result">
    <div v-if="metadataEntries.length > 0" class="result-summary">
      <span class="result-summary-title">执行结果摘要</span>
      <div v-if="metadataEntries.length > 0" class="metadata-tags">
        <el-tag v-for="entry in metadataEntries" :key="entry.key" size="small" effect="plain">
          {{ entry.label }}：{{ entry.value }}
        </el-tag>
      </div>
    </div>

    <section class="result-section request-section">
      <div class="result-section-header">
        <div class="result-section-heading">
          <span class="result-section-icon request-icon">
            <el-icon><Upload /></el-icon>
          </span>
          <div>
            <div class="result-section-title">{{ requestTitle }}</div>
            <div class="result-section-desc">本次执行实际提交给函数的输入。</div>
          </div>
        </div>
        <el-tag size="small" type="info" effect="plain">
          {{ requestFields.length || 'JSON' }} {{ requestFields.length ? '个字段' : '载荷' }}
        </el-tag>
      </div>

      <div v-if="requestFields.length > 0" class="field-grid">
        <article v-for="field in requestFields" :key="fieldKey(field)" class="field-card">
          <header class="field-label">
            <span>{{ field.name || field.code }}</span>
            <code>{{ field.code }}</code>
          </header>
          <div class="field-value-shell">
            <WidgetComponent
              :field="field"
              :value="fieldValue(requestPayload, field)"
              mode="detail"
              :field-path="field.field_path || field.code"
            />
          </div>
        </article>
      </div>
      <pre v-else class="json-fallback">{{ formatJSON(requestPayload) }}</pre>
    </section>

    <section class="result-section response-section">
      <div class="result-section-header">
        <div class="result-section-heading">
          <span class="result-section-icon response-icon">
            <el-icon><DataAnalysis /></el-icon>
          </span>
          <div>
            <div class="result-section-title">{{ responseTitle }}</div>
            <div class="result-section-desc">按函数响应配置只读展示输出结果。</div>
          </div>
        </div>
        <el-tag size="small" type="success" effect="plain">
          {{ responseFields.length || 'JSON' }} {{ responseFields.length ? '个字段' : '结果' }}
        </el-tag>
      </div>

      <div v-if="responseFields.length > 0" class="field-grid">
        <article v-for="field in responseFields" :key="fieldKey(field)" class="field-card">
          <header class="field-label">
            <span>{{ field.name || field.code }}</span>
            <code>{{ field.code }}</code>
          </header>
          <div class="field-value-shell">
            <WidgetComponent
              :field="field"
              :value="fieldValue(responsePayload, field)"
              mode="response"
              :field-path="field.field_path || field.code"
            />
          </div>
        </article>
      </div>
      <pre v-else class="json-fallback">{{ formatJSON(responsePayload) }}</pre>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FieldConfig, FunctionDetail } from '@/architecture/domain/types'
import WidgetComponent from '@/architecture/presentation/widgets/WidgetComponent.vue'
import { convertToFieldValue } from '@/architecture/runtime/utils/field'
import { DataAnalysis, Upload } from '@element-plus/icons-vue'
import { getFormRequestFields, getFormResponseFields } from '@/architecture/domain/utils/functionSchemaSelectors'

const props = withDefaults(defineProps<{
  functionDetail?: FunctionDetail | null
  requestPayload?: Record<string, any> | null
  responsePayload?: unknown | null
  responseMetadata?: Record<string, any> | null
  requestTitle?: string
  responseTitle?: string
}>(), {
  requestTitle: '输入参数',
  responseTitle: '输出结果'
})

const requestFields = computed<FieldConfig[]>(() => getFormRequestFields(props.functionDetail))
const responseFields = computed<FieldConfig[]>(() => getFormResponseFields(props.functionDetail))

const metadataEntries = computed(() => {
  const metadata = props.responseMetadata || {}
  return [
    { key: 'version', label: '版本', value: metadata.version },
    { key: 'total_cost_mill', label: '耗时', value: metadata.total_cost_mill ? `${metadata.total_cost_mill} ms` : '' },
    { key: 'err_code', label: '错误码', value: metadata.err_code },
    { key: 'trace_id', label: 'Trace', value: metadata.trace_id }
  ].filter((entry) => entry.value !== undefined && entry.value !== null && entry.value !== '')
})

function fieldKey(field: FieldConfig): string {
  return field.field_path || field.code
}

function isRecord(value: unknown): value is Record<string, any> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function readByPath(data: Record<string, any>, path: string): unknown {
  const normalized = path.replace(/\[(\d+)\]/g, '.$1')
  const parts = normalized.split('.').filter(Boolean)
  let current: unknown = data
  for (const part of parts) {
    if (!isRecord(current) && !Array.isArray(current)) {
      return undefined
    }
    current = (current as any)[part]
  }
  return current
}

function rawFieldValue(payload: unknown, field: FieldConfig): unknown {
  if (!isRecord(payload)) {
    return null
  }
  if (Object.prototype.hasOwnProperty.call(payload, field.code)) {
    return payload[field.code]
  }
  if (field.field_path) {
    const pathValue = readByPath(payload, field.field_path)
    if (pathValue !== undefined) {
      return pathValue
    }
  }
  return null
}

function fieldValue(payload: unknown, field: FieldConfig) {
  return convertToFieldValue(rawFieldValue(payload, field), field)
}

function formatJSON(value: unknown): string {
  if (value === null || value === undefined) {
    return '{}'
  }
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}
</script>

<style scoped>
.function-execution-result {
  display: flex;
  flex-direction: column;
  gap: 16px;
  margin-top: 16px;
}

.result-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-fill-color-light);
}

.result-summary-title {
  flex: 0 0 auto;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.metadata-tags {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.metadata-tags :deep(.el-tag) {
  background: var(--el-bg-color);
}

.result-section {
  padding: 18px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-bg-color);
}

.result-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.result-section-heading {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.result-section-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  width: 34px;
  height: 34px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
}

.request-icon {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.response-icon {
  color: var(--el-color-success);
  background: var(--el-color-success-light-9);
}

.result-section-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.result-section-desc {
  margin-top: 3px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.field-card {
  min-width: 0;
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-fill-color-light);
}

.field-label {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.field-label code {
  max-width: 48%;
  padding: 2px 7px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  font-size: 11px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}

.field-value-shell {
  min-height: 34px;
}

.json-fallback {
  position: relative;
  z-index: 1;
  margin: 0;
  padding: 16px 18px;
  border-radius: 10px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font-size: 12px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 42vh;
  overflow: auto;
}

@media (max-width: 960px) {
  .result-summary,
  .result-section-header {
    flex-direction: column;
    align-items: stretch;
  }

  .metadata-tags {
    justify-content: flex-start;
  }

  .field-grid {
    grid-template-columns: 1fr;
  }
}
</style>
