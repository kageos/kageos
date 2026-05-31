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

    <section v-if="connectorItems.length > 0" class="function-info-section connector-section">
      <div class="section-heading connector-heading">
        <div class="section-heading-title">
          <el-icon><Connection /></el-icon>
          <span>连接器依赖</span>
        </div>
        <span class="connector-ready-count">{{ connectedConnectorCount }}/{{ connectorItems.length }} 可用</span>
      </div>
      <div class="connector-list">
        <div
          v-for="item in connectorItems"
          :key="item.provider"
          class="connector-item"
          :class="{ 'is-connected': isConnectorReady(item), 'is-scope-missing': hasMissingScopes(item) }"
        >
          <span class="connector-lamp" />
          <div class="connector-copy">
            <span class="connector-provider">{{ item.provider }}</span>
            <span class="connector-message" :title="connectorStatusText(item)">
              {{ connectorStatusText(item) }}
            </span>
          </div>
          <el-button
            v-if="!isConnectorReady(item)"
            class="connector-action"
            size="small"
            :type="hasMissingScopes(item) ? 'danger' : 'primary'"
            plain
            @click.stop="handleConnectConnector(item.provider, connectorAuthorizeScopes(item))"
          >
            {{ hasMissingScopes(item) ? '补授权' : '连接' }}
          </el-button>
        </div>
      </div>
      <div v-if="connectorEndpointItems.length > 0" class="connector-endpoint-list">
        <div v-for="(endpoint, index) in connectorEndpointItems" :key="`${endpoint.provider || 'connector'}-${endpoint.method || 'GET'}-${endpoint.url || index}`" class="connector-endpoint-item">
          <el-tag size="small" effect="plain">{{ endpoint.provider || '-' }}</el-tag>
          <el-tag size="small" type="info" effect="plain">{{ endpoint.method || 'GET' }}</el-tag>
          <code>{{ endpoint.url || '-' }}</code>
          <span v-if="endpoint.name" class="connector-endpoint-name">{{ endpoint.name }}</span>
          <div v-if="endpoint.required_scopes?.length" class="connector-endpoint-scopes">
            <el-tag
              v-for="scope in endpoint.required_scopes"
              :key="scope"
              size="small"
              type="warning"
              effect="plain"
            >
              {{ scope }}
            </el-tag>
          </div>
        </div>
      </div>
    </section>

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
import { ElMessage } from 'element-plus'
import { Connection, DataLine, Document, Link, Operation, Tickets } from '@element-plus/icons-vue'
import type { FieldConfig, FunctionConnectorEndpoint, FunctionConnectorStatus, FunctionDetail, ServiceTree } from '@/architecture/domain/types'
import { CONNECTOR_GLOBAL_RESOURCE_PATH, startConnectorOAuth } from '@/architecture/presentation/context/api/connector'
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
    fullCodePath: detail?.full_code_path || node?.full_code_path || '',
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

const connectorItems = computed<FunctionConnectorStatus[]>(() => {
  const detail = props.functionData
  const required = Array.isArray(detail?.connectors)
    ? detail.connectors
    : Array.isArray(props.functionNode?.connectors)
      ? props.functionNode.connectors
      : []
  const statusMap = new Map((detail?.connector_status || []).map(item => [item.provider, item]))
  const requiredScopeMap = connectorRequiredScopesByProvider.value

  return required
    .map(provider => provider.trim())
    .filter(Boolean)
    .map(provider => {
      const status = statusMap.get(provider)
      if (status) {
        return {
          ...status,
          required_scopes: normalizeScopes([
            ...(status.required_scopes || []),
            ...(requiredScopeMap.get(provider) || [])
          ])
        }
      }
      return {
        provider,
        required: true,
        connected: false,
        required_scopes: requiredScopeMap.get(provider) || []
      }
    })
})

const connectedConnectorCount = computed(() => connectorItems.value.filter(item => isConnectorReady(item)).length)

const connectorEndpointItems = computed<FunctionConnectorEndpoint[]>(() => {
  const detail = props.functionData
  const endpoints = Array.isArray(detail?.connector_endpoints)
    ? detail.connector_endpoints
    : Array.isArray(props.functionNode?.connector_endpoints)
      ? props.functionNode.connector_endpoints
      : []
  return endpoints.filter(endpoint => endpoint && (endpoint.provider || endpoint.url))
})

const connectorRequiredScopesByProvider = computed(() => {
  const scopeMap = new Map<string, string[]>()
  for (const endpoint of connectorEndpointItems.value) {
    const provider = (endpoint.provider || '').trim()
    if (!provider) {
      continue
    }
    scopeMap.set(provider, normalizeScopes([
      ...(scopeMap.get(provider) || []),
      ...(endpoint.required_scopes || [])
    ]))
  }
  return scopeMap
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

function connectorStatusText(item: FunctionConnectorStatus): string {
  if (hasMissingScopes(item)) {
    return `权限不足：${item.missing_scopes?.join('、')}`
  }
  if (item.connected) {
    return item.display_name || item.resolved_from || '已连接'
  }
  return item.message || '未连接'
}

function normalizeScopes(scopes: string[]): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const scope of scopes) {
    for (const part of String(scope || '').replace(/,/g, ' ').split(/\s+/)) {
      const trimmed = part.trim()
      if (!trimmed || seen.has(trimmed)) {
        continue
      }
      seen.add(trimmed)
      out.push(trimmed)
    }
  }
  return out
}

function hasMissingScopes(item: FunctionConnectorStatus): boolean {
  return Boolean(item.connected && item.missing_scopes?.length)
}

function isConnectorReady(item: FunctionConnectorStatus): boolean {
  return Boolean(item.connected && !item.missing_scopes?.length)
}

function connectorAuthorizeScopes(item: FunctionConnectorStatus): string[] {
  return normalizeScopes([
    ...(item.granted_scopes || []),
    ...(item.missing_scopes || []),
    ...(item.required_scopes || [])
  ])
}

function connectorResourcePath(): string {
  return mergedFunctionData.value.fullCodePath || CONNECTOR_GLOBAL_RESOURCE_PATH
}

async function handleConnectConnector(provider: string, scopes: string[] = []) {
  try {
    const redirectAfter = `${window.location.pathname}${window.location.search}${window.location.hash}`
    const resp = await startConnectorOAuth({
      provider,
      resource_path: connectorResourcePath(),
      scopes,
      redirect_after: redirectAfter
    })
    window.location.href = resp.authorize_url
  } catch (error) {
    const message = error instanceof Error ? error.message : '发起连接器授权失败'
    ElMessage.error(message)
  }
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

.connector-heading {
  justify-content: space-between;
}

.schema-count {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.connector-ready-count {
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

.connector-section {
  margin-bottom: 16px;
}

.connector-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.connector-endpoint-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
}

.connector-endpoint-item {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-width: 0;
  font-size: 12px;
}

.connector-endpoint-item code {
  min-width: 0;
  color: var(--el-text-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  word-break: break-all;
}

.connector-endpoint-name {
  color: var(--el-text-color-secondary);
}

.connector-endpoint-scopes {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 6px;
}

.connector-item {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-width: 240px;
  max-width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--app-shell-panel-muted-bg, var(--el-fill-color-lighter));
}

.connector-lamp {
  width: 10px;
  height: 10px;
  flex: 0 0 10px;
  border-radius: 50%;
  background: var(--el-color-warning);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--el-color-warning) 16%, transparent);
}

.connector-item.is-connected .connector-lamp {
  background: var(--el-color-success);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--el-color-success) 16%, transparent);
}

.connector-item.is-scope-missing .connector-lamp {
  background: var(--el-color-danger);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--el-color-danger) 16%, transparent);
}

.connector-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.connector-provider {
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 700;
}

.connector-message {
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.connector-action {
  flex: 0 0 auto;
  margin-left: auto;
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
