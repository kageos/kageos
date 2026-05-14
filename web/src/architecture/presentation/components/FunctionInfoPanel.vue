<template>
  <div class="function-info-panel">
    <section class="function-summary-section">
      <div class="function-summary-main">
        <div class="function-summary-kicker">{{ templateTypeLabel }}函数</div>
        <h2 class="function-summary-title">{{ mergedFunctionData.name }}</h2>
        <div class="function-summary-path">
          <el-icon><Link /></el-icon>
          <span>{{ mergedFunctionData.fullCodePath || mergedFunctionData.router || '-' }}</span>
        </div>
      </div>
      <div class="function-summary-badges">
        <el-tag size="small" effect="plain">{{ mergedFunctionData.method }}</el-tag>
        <el-tag size="small" type="info" effect="plain">{{ templateTypeLabel }}</el-tag>
        <el-tag v-if="mergedFunctionData.runCount !== null" size="small" type="success" effect="plain">
          {{ mergedFunctionData.runCount }} 次运行
        </el-tag>
      </div>
    </section>

    <div class="function-info-layout">
      <section class="function-info-section description-section">
        <div class="section-heading">
          <el-icon><Document /></el-icon>
          <span>详细描述</span>
        </div>
        <div
          v-if="mergedFunctionData.description"
          class="function-markdown-body"
          v-html="renderMarkdown(mergedFunctionData.description)"
        />
        <el-empty v-else description="暂无详细描述" :image-size="72" />
      </section>

      <section class="function-info-section metadata-section">
        <div class="section-heading">
          <el-icon><DataLine /></el-icon>
          <span>基础信息</span>
        </div>
        <div class="metadata-grid">
          <div v-for="item in metadataItems" :key="item.label" class="metadata-item">
            <span class="metadata-label">{{ item.label }}</span>
            <span class="metadata-value">{{ item.value }}</span>
          </div>
        </div>
        <div v-if="tagList.length > 0" class="tag-list">
          <el-tag v-for="tag in tagList" :key="tag" size="small" effect="plain">
            {{ tag }}
          </el-tag>
        </div>
      </section>
    </div>

    <section class="function-info-section schema-section">
      <div class="section-heading schema-heading">
        <div class="section-heading-title">
          <el-icon><Tickets /></el-icon>
          <span>字段结构</span>
        </div>
        <span class="schema-count">{{ requestFields.length + outputFields.length }} 个字段</span>
      </div>

      <div v-if="hasSchemaFields" class="schema-columns">
        <div class="schema-column">
          <div class="schema-column-title">请求字段</div>
          <div v-if="requestFields.length > 0" class="schema-field-list">
            <div v-for="field in requestFields" :key="field.code" class="schema-field-row">
              <div class="schema-field-topline">
                <div class="schema-field-main">
                  <span class="schema-field-name">{{ field.name || field.code }}</span>
                  <code>{{ field.code }}</code>
                </div>
                <div class="schema-field-badges">
                  <el-tag size="small" effect="plain">{{ resolveFieldType(field) }}</el-tag>
                  <el-tag v-if="isRequiredField(field)" size="small" type="danger" effect="plain">必填</el-tag>
                </div>
              </div>
              <div class="schema-field-widget">{{ resolveWidgetType(field) }}</div>
              <p v-if="field.desc" class="schema-field-desc">{{ field.desc }}</p>
              <span v-if="field.children?.length" class="schema-field-children">
                {{ field.children.length }} 个子字段
              </span>
            </div>
          </div>
          <el-empty v-else description="暂无请求字段" :image-size="64" />
        </div>

        <div class="schema-column">
          <div class="schema-column-title">{{ outputFieldsTitle }}</div>
          <div v-if="outputFields.length > 0" class="schema-field-list">
            <div v-for="field in outputFields" :key="field.code" class="schema-field-row">
              <div class="schema-field-topline">
                <div class="schema-field-main">
                  <span class="schema-field-name">{{ field.name || field.code }}</span>
                  <code>{{ field.code }}</code>
                </div>
                <div class="schema-field-badges">
                  <el-tag size="small" effect="plain">{{ resolveFieldType(field) }}</el-tag>
                </div>
              </div>
              <div class="schema-field-widget">{{ resolveWidgetType(field) }}</div>
              <p v-if="field.desc" class="schema-field-desc">{{ field.desc }}</p>
              <span v-if="field.children?.length" class="schema-field-children">
                {{ field.children.length }} 个子字段
              </span>
            </div>
          </div>
          <el-empty v-else :description="`暂无${outputFieldsTitle}`" :image-size="64" />
        </div>
      </div>
      <el-empty v-else description="暂无字段结构" :image-size="76" />
    </section>

    <section v-if="callbackList.length > 0" class="function-info-section callback-section">
      <div class="section-heading">
        <el-icon><Operation /></el-icon>
        <span>回调配置</span>
      </div>
      <div class="callback-list">
        <el-tag v-for="callback in callbackList" :key="callback" effect="plain">
          {{ callback }}
        </el-tag>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { DataLine, Document, Link, Operation, Tickets } from '@element-plus/icons-vue'
import type { FieldConfig, FunctionDetail, ServiceTree } from '@/architecture/domain/types'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'

interface Props {
  functionData?: FunctionDetail | null
  functionNode?: ServiceTree | null
}

const props = defineProps<Props>()
const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()

const mergedFunctionData = computed(() => {
  const detail = props.functionData
  const node = props.functionNode
  const name = detail?.name || node?.name || detail?.code || node?.code || '未命名函数'
  const runCount = typeof node?.run_count === 'number' ? node.run_count : null

  return {
    name,
    code: detail?.code || node?.code || '',
    description: (detail?.description || node?.description || '').trim(),
    method: detail?.method || 'GET',
    router: detail?.router || '',
    fullCodePath: node?.full_code_path || '',
    templateType: detail?.template_type || node?.template_type || detail?.schema?.type || '',
    createdBy: node?.owner || detail?.created_by || '',
    createdAt: detail?.created_at || node?.created_at || '',
    updatedAt: detail?.updated_at || node?.updated_at || '',
    runCount,
    tags: node?.tags || ''
  }
})

const schemaType = computed(() => {
  return props.functionData?.schema?.type || mergedFunctionData.value.templateType
})

const templateTypeLabel = computed(() => {
  switch (schemaType.value) {
    case 'table':
      return '表格'
    case 'form':
      return '表单'
    case 'chart':
      return '图表'
    default:
      return '通用'
  }
})

const requestFields = computed<FieldConfig[]>(() => {
  const schema = props.functionData?.schema
  switch (schemaType.value) {
    case 'table':
      return schema?.table?.request || []
    case 'chart':
      return schema?.chart?.request || []
    case 'form':
      return schema?.form?.request || []
    default:
      return schema?.form?.request || schema?.table?.request || schema?.chart?.request || []
  }
})

const outputFields = computed<FieldConfig[]>(() => {
  const schema = props.functionData?.schema
  switch (schemaType.value) {
    case 'table':
      return schema?.table?.fields || []
    case 'chart':
      return schema?.chart?.response || []
    case 'form':
      return schema?.form?.response || []
    default:
      return schema?.form?.response || schema?.table?.fields || schema?.chart?.response || []
  }
})

const outputFieldsTitle = computed(() => {
  return schemaType.value === 'table' ? '列表字段' : '响应字段'
})

const hasSchemaFields = computed(() => {
  return requestFields.value.length > 0 || outputFields.value.length > 0
})

const callbackList = computed(() => {
  return Array.isArray(props.functionData?.schema?.callbacks) ? props.functionData.schema.callbacks : []
})

const tagList = computed(() => {
  return mergedFunctionData.value.tags
    .split(',')
    .map(tag => tag.trim())
    .filter(Boolean)
})

const metadataItems = computed(() => [
  { label: '函数编码', value: displayValue(mergedFunctionData.value.code) },
  { label: '资源路径', value: displayValue(mergedFunctionData.value.fullCodePath) },
  { label: '接口路径', value: displayValue(mergedFunctionData.value.router) },
  { label: '请求方法', value: displayValue(mergedFunctionData.value.method) },
  { label: '模板类型', value: templateTypeLabel.value },
  { label: '创建者', value: displayValue(mergedFunctionData.value.createdBy) },
  { label: '创建时间', value: formatDate(mergedFunctionData.value.createdAt) },
  { label: '更新时间', value: formatDate(mergedFunctionData.value.updatedAt) }
])

function displayValue(value: unknown): string {
  if (value === undefined || value === null || value === '') {
    return '-'
  }
  return String(value)
}

function formatDate(value: string | Date | undefined): string {
  if (!value) {
    return '-'
  }

  const date = typeof value === 'string' ? new Date(value) : value
  if (Number.isNaN(date.getTime())) {
    return String(value)
  }

  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function resolveFieldType(field: FieldConfig): string {
  return field.type || field.data?.type || field.meta?.dataType || 'unknown'
}

function resolveWidgetType(field: FieldConfig): string {
  return field.widget?.type ? `组件：${field.widget.type}` : '组件：默认'
}

function isRequiredField(field: FieldConfig): boolean {
  return Boolean(field.meta?.isRequired || field.validation?.split(',').some(rule => rule.trim() === 'required'))
}
</script>

<style scoped lang="scss">
.function-info-panel {
  min-height: 100%;
  padding: 18px 4px 28px;
  color: var(--el-text-color-primary);
}

.function-summary-section,
.function-info-section {
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 8px;
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color));
  box-shadow: var(--app-shell-panel-shadow-soft, 0 8px 24px rgba(15, 23, 42, 0.06));
}

.function-summary-section {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  padding: 18px 20px;
  margin-bottom: 16px;
}

.function-summary-main {
  min-width: 0;
}

.function-summary-kicker {
  margin-bottom: 6px;
  color: var(--el-color-primary);
  font-size: 12px;
  font-weight: 700;
}

.function-summary-title {
  margin: 0;
  font-size: 22px;
  line-height: 1.3;
  font-weight: 750;
  color: var(--el-text-color-primary);
}

.function-summary-path {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  margin-top: 10px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.function-summary-path span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.function-summary-badges {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
  flex-shrink: 0;
}

.function-info-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(300px, 0.8fr);
  gap: 16px;
  margin-bottom: 16px;
}

.function-info-section {
  padding: 18px;
}

.section-heading,
.schema-heading,
.section-heading-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.section-heading,
.schema-heading {
  margin-bottom: 14px;
  color: var(--el-text-color-primary);
  font-size: 15px;
  font-weight: 700;
}

.schema-heading {
  justify-content: space-between;
}

.schema-count {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.function-markdown-body {
  color: var(--el-text-color-regular);
  font-size: 14px;
  line-height: 1.72;
}

.function-markdown-body :deep(h1),
.function-markdown-body :deep(h2),
.function-markdown-body :deep(h3),
.function-markdown-body :deep(h4),
.function-markdown-body :deep(h5),
.function-markdown-body :deep(h6) {
  margin: 16px 0 9px;
  color: var(--el-text-color-primary);
  line-height: 1.35;
}

.function-markdown-body :deep(p) {
  margin: 8px 0;
}

.function-markdown-body :deep(ul),
.function-markdown-body :deep(ol) {
  margin: 8px 0;
  padding-left: 22px;
}

.function-markdown-body :deep(blockquote) {
  margin: 12px 0;
  padding: 8px 12px;
  border-left: 3px solid var(--el-color-primary);
  border-radius: 6px;
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-light));
}

.function-markdown-body :deep(code) {
  padding: 2px 5px;
  border-radius: 5px;
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-light));
  color: var(--el-color-primary);
}

.function-markdown-body :deep(pre) {
  padding: 12px;
  border-radius: 8px;
  background: #0f172a;
  color: #e2e8f0;
  overflow-x: auto;
}

.function-markdown-body :deep(pre code) {
  padding: 0;
  background: transparent;
  color: inherit;
}

.function-markdown-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 12px 0;
}

.function-markdown-body :deep(th),
.function-markdown-body :deep(td) {
  padding: 8px 10px;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
}

.function-markdown-body :deep(th) {
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-lighter));
  font-weight: 700;
}

.metadata-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 10px;
}

.metadata-item {
  display: grid;
  grid-template-columns: 86px minmax(0, 1fr);
  gap: 12px;
  align-items: start;
  font-size: 13px;
}

.metadata-label {
  color: var(--el-text-color-secondary);
}

.metadata-value {
  min-width: 0;
  color: var(--el-text-color-primary);
  word-break: break-word;
}

.tag-list,
.callback-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-list {
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
}

.schema-section {
  margin-bottom: 16px;
}

.schema-columns {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.schema-column {
  min-width: 0;
}

.schema-column-title {
  margin-bottom: 10px;
  color: var(--el-text-color-regular);
  font-size: 13px;
  font-weight: 700;
}

.schema-field-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.schema-field-row {
  padding: 12px;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 8px;
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-lighter));
}

.schema-field-topline {
  display: flex;
  justify-content: space-between;
  gap: 10px;
}

.schema-field-main {
  min-width: 0;
}

.schema-field-name {
  display: block;
  margin-bottom: 4px;
  font-size: 14px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.schema-field-main code {
  color: var(--el-text-color-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  word-break: break-all;
}

.schema-field-badges {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
  flex-shrink: 0;
}

.schema-field-widget,
.schema-field-children {
  display: inline-flex;
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.schema-field-desc {
  margin: 8px 0 0;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.6;
}

.callback-section {
  margin-bottom: 16px;
}

@media (max-width: 960px) {
  .function-summary-section,
  .function-info-layout,
  .schema-columns {
    grid-template-columns: 1fr;
  }

  .function-summary-section {
    flex-direction: column;
  }

  .function-summary-badges {
    justify-content: flex-start;
  }
}
</style>
