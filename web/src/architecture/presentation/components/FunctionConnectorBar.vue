<template>
  <div v-if="connectorItems.length > 0" class="function-connector-bar">
    <div class="connector-bar-title">
      <el-icon><Connection /></el-icon>
      <span>连接器</span>
      <em>{{ readyConnectorCount }}/{{ connectorItems.length }} 可用</em>
    </div>
    <div class="connector-bar-list">
      <div
        v-for="item in connectorItems"
        :key="item.provider"
        class="connector-chip"
        :class="{ 'is-ready': isConnectorReady(item), 'is-scope-missing': hasMissingScopes(item) }"
      >
        <span class="connector-lamp" />
        <div class="connector-chip-copy">
          <span class="connector-provider">{{ item.provider }}</span>
          <span class="connector-status" :title="connectorStatusText(item)">
            {{ connectorStatusText(item) }}
          </span>
        </div>
        <div v-if="item.required_scopes?.length" class="connector-scopes">
          <el-tag
            v-for="scope in item.required_scopes"
            :key="scope"
            size="small"
            effect="plain"
            :type="item.missing_scopes?.includes(scope) ? 'danger' : 'info'"
          >
            {{ scope }}
          </el-tag>
        </div>
        <el-button
          v-if="!isConnectorReady(item)"
          size="small"
          :type="hasMissingScopes(item) ? 'danger' : 'primary'"
          plain
          @click.stop="handleConnectConnector(item.provider, connectorAuthorizeScopes(item))"
        >
          {{ hasMissingScopes(item) ? '补授权' : '连接' }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection } from '@element-plus/icons-vue'
import type { FunctionConnectorEndpoint, FunctionConnectorStatus, FunctionDetail, ServiceTree } from '@/architecture/domain/types'
import { CONNECTOR_GLOBAL_RESOURCE_PATH, startConnectorOAuth } from '@/architecture/presentation/context/api/connector'

const props = defineProps<{
  currentFunction: ServiceTree | null
  functionDetail: FunctionDetail | null
}>()

const connectorEndpointItems = computed<FunctionConnectorEndpoint[]>(() => {
  const endpoints = Array.isArray(props.functionDetail?.connector_endpoints)
    ? props.functionDetail.connector_endpoints
    : Array.isArray(props.currentFunction?.connector_endpoints)
      ? props.currentFunction.connector_endpoints
      : []
  return endpoints.filter(endpoint => endpoint && (endpoint.provider || endpoint.url))
})

const connectorRequiredScopesByProvider = computed(() => {
  const scopeMap = new Map<string, string[]>()
  for (const endpoint of connectorEndpointItems.value) {
    const provider = (endpoint.provider || '').trim().toLowerCase()
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

const connectorItems = computed<FunctionConnectorStatus[]>(() => {
  const required = Array.isArray(props.functionDetail?.connectors)
    ? props.functionDetail.connectors
    : Array.isArray(props.currentFunction?.connectors)
      ? props.currentFunction.connectors
      : []
  const statusMap = new Map((props.functionDetail?.connector_status || []).map(item => [item.provider, item]))
  const requiredScopeMap = connectorRequiredScopesByProvider.value

  return required
    .map(provider => provider.trim().toLowerCase())
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

const readyConnectorCount = computed(() => connectorItems.value.filter(item => isConnectorReady(item)).length)

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

function connectorStatusText(item: FunctionConnectorStatus): string {
  if (hasMissingScopes(item)) {
    return `权限不足：${item.missing_scopes?.join('、')}`
  }
  if (item.connected) {
    return item.display_name || item.resolved_from || '已连接'
  }
  return item.message || '未连接'
}

function connectorAuthorizeScopes(item: FunctionConnectorStatus): string[] {
  return normalizeScopes([
    ...(item.granted_scopes || []),
    ...(item.missing_scopes || []),
    ...(item.required_scopes || [])
  ])
}

function connectorResourcePath(): string {
  return props.functionDetail?.full_code_path || props.currentFunction?.full_code_path || CONNECTOR_GLOBAL_RESOURCE_PATH
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
.function-connector-bar {
  flex: 0 0 auto;
  margin: 4px 0 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
  padding: 10px 12px;
}

.connector-bar-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 700;
}

.connector-bar-title em {
  margin-left: auto;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-style: normal;
  font-weight: 600;
}

.connector-bar-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.connector-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 220px;
  max-width: 100%;
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.connector-lamp {
  width: 9px;
  height: 9px;
  flex: 0 0 9px;
  border-radius: 50%;
  background: var(--el-color-warning);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--el-color-warning) 16%, transparent);
}

.connector-chip.is-ready .connector-lamp {
  background: var(--el-color-success);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--el-color-success) 16%, transparent);
}

.connector-chip.is-scope-missing .connector-lamp {
  background: var(--el-color-danger);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--el-color-danger) 16%, transparent);
}

.connector-chip-copy {
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

.connector-status {
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.connector-scopes {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
  margin-left: auto;
}
</style>
