<template>
  <div v-if="connectorItems.length > 0" class="function-connector-bar">
    <div class="connector-bar-title">
      <span class="connector-title-icon">
        <el-icon><Connection /></el-icon>
      </span>
      <span class="connector-title-text">连接器</span>
      <em>{{ readyConnectorCount }}/{{ connectorItems.length }} 可用</em>
    </div>
    <div class="connector-bar-list">
      <div
        v-for="item in connectorItems"
        :key="item.provider"
        class="connector-card"
        :class="connectorCardClass(item)"
        :style="connectorIconStyle(item)"
      >
        <span
          class="connector-logo-stack"
          :class="{ 'has-account': hasConnectorAccount(item) }"
          :title="connectorLogoTitle(item)"
        >
          <span class="connector-logo-box connector-platform-logo">
            <img
              v-if="connectorProviderLogo(item)"
              :src="connectorProviderLogo(item)"
              :alt="connectorProviderName(item)"
              @error="handleConnectorIconError"
            />
            <span>{{ connectorProviderInitial(item) }}</span>
          </span>
          <span v-if="hasConnectorAccount(item)" class="connector-logo-box connector-account-logo">
            <img
              v-if="connectorAccountIcon(item)"
              :src="connectorAccountIcon(item)"
              :alt="connectorProfileName(item) || item.display_name || connectorProviderName(item)"
              @error="handleConnectorIconError"
            />
            <span>{{ connectorAccountInitial(item) }}</span>
          </span>
        </span>

        <div class="connector-card-body">
          <div class="connector-card-heading">
            <span class="connector-provider">{{ connectorProviderName(item) }}</span>
            <span class="connector-state-pill">
              <span class="connector-state-dot" />
              {{ connectorStateLabel(item) }}
            </span>
          </div>
          <div class="connector-copy-lines">
            <span class="connector-status" :title="connectorStatusText(item)">
              {{ connectorStatusText(item) }}
            </span>
            <span v-if="connectorProfileDetail(item)" class="connector-profile-detail">
              {{ connectorProfileDetail(item) }}
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
        </div>

        <el-button
          v-if="!isConnectorReady(item)"
          class="connector-action"
          size="small"
          :type="hasMissingScopes(item) ? 'danger' : 'primary'"
          plain
          @click.stop="handleConnectConnector(item.provider, connectorAuthorizeScopes(item), item.connection_id)"
        >
          {{ hasMissingScopes(item) ? '补授权' : '连接' }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection } from '@element-plus/icons-vue'
import type { FunctionConnectorEndpoint, FunctionConnectorStatus, FunctionDetail, ServiceTree } from '@/architecture/domain/types'
import {
  CONNECTOR_GLOBAL_RESOURCE_PATH,
  listConnectorOAuthProviders,
  startConnectorOAuth,
  type ConnectorOAuthProviderInfo
} from '@/architecture/presentation/context/api/connector'

const props = defineProps<{
  currentFunction: ServiceTree | null
  functionDetail: FunctionDetail | null
}>()

type ConnectorProviderDisplay = Pick<
  ConnectorOAuthProviderInfo,
  'code' | 'name' | 'logo_url' | 'brand_color' | 'provider_account_url'
>

const connectorProviderDisplays = ref<Map<string, ConnectorProviderDisplay>>(new Map())

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
  const statusMap = new Map((props.functionDetail?.connector_status || []).map(item => [item.provider.trim().toLowerCase(), item]))
  const requiredScopeMap = connectorRequiredScopesByProvider.value

  return required
    .map(provider => provider.trim().toLowerCase())
    .filter(Boolean)
    .map(provider => {
      const status = statusMap.get(provider)
      if (status) {
        return withConnectorProviderDisplay({
          ...status,
          provider,
          required_scopes: normalizeScopes([
            ...(status.required_scopes || []),
            ...(requiredScopeMap.get(provider) || [])
          ])
        })
      }
      return withConnectorProviderDisplay({
        provider,
        required: true,
        connected: false,
        required_scopes: requiredScopeMap.get(provider) || []
      })
    })
})

const readyConnectorCount = computed(() => connectorItems.value.filter(item => isConnectorReady(item)).length)

onMounted(loadConnectorProviderDisplays)

async function loadConnectorProviderDisplays() {
  try {
    const resp = await listConnectorOAuthProviders()
    const items = Array.isArray(resp.providers) ? resp.providers : []
    connectorProviderDisplays.value = new Map(
      items
        .filter(item => item?.code)
        .map(item => [item.code.trim().toLowerCase(), item])
    )
  } catch {
    connectorProviderDisplays.value = new Map()
  }
}

function withConnectorProviderDisplay(item: FunctionConnectorStatus): FunctionConnectorStatus {
  const display = connectorProviderDisplays.value.get(item.provider)
  if (!display) {
    return item
  }
  return {
    ...item,
    provider_name: item.provider_name || display.name || item.provider,
    provider_logo_url: item.provider_logo_url || display.logo_url || '',
    provider_brand_color: item.provider_brand_color || display.brand_color || '',
    provider_account_url: item.provider_account_url || display.provider_account_url || ''
  }
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

function connectorCardClass(item: FunctionConnectorStatus) {
  return {
    'is-ready': isConnectorReady(item),
    'is-scope-missing': hasMissingScopes(item),
    'is-disconnected': !item.connected
  }
}

function connectorStateLabel(item: FunctionConnectorStatus): string {
  if (hasMissingScopes(item)) {
    return '需补授权'
  }
  if (isConnectorReady(item)) {
    return '已连接'
  }
  return '未连接'
}

function connectorStatusText(item: FunctionConnectorStatus): string {
  if (hasMissingScopes(item)) {
    return `权限不足：${item.missing_scopes?.join('、')}`
  }
  if (item.connected) {
    return connectorProfileName(item) || item.display_name || item.resolved_from || '已连接'
  }
  return item.message || '未连接'
}

function connectorProfileName(item: FunctionConnectorStatus): string {
  return item.profile?.workspace_name || item.profile?.display_name || item.profile?.account_name || ''
}

function connectorProviderName(item: FunctionConnectorStatus): string {
  return item.provider_name || item.provider
}

function connectorProfileIcon(item: FunctionConnectorStatus): string {
  return item.profile?.workspace_icon || item.profile?.avatar_url || ''
}

function connectorProviderLogo(item: FunctionConnectorStatus): string {
  return item.provider_logo_url || ''
}

function connectorAccountIcon(item: FunctionConnectorStatus): string {
  return item.connected ? connectorProfileIcon(item) : ''
}

function hasConnectorAccount(item: FunctionConnectorStatus): boolean {
  return Boolean(
    item.connected && (
      connectorAccountIcon(item) ||
      connectorProfileName(item) ||
      item.display_name ||
      item.resolved_from
    )
  )
}

function connectorIconStyle(item: FunctionConnectorStatus) {
  const color = (item.provider_brand_color || '').trim()
  return color ? { '--connector-provider-color': color } : {}
}

function connectorProviderInitial(item: FunctionConnectorStatus): string {
  return connectorProviderName(item).trim().slice(0, 1).toUpperCase() || '?'
}

function connectorAccountInitial(item: FunctionConnectorStatus): string {
  return (connectorProfileName(item) || item.display_name || connectorProviderName(item)).trim().slice(0, 1).toUpperCase() || '?'
}

function connectorLogoTitle(item: FunctionConnectorStatus): string {
  const accountName = connectorProfileName(item) || item.display_name || ''
  return accountName ? `${connectorProviderName(item)} · ${accountName}` : connectorProviderName(item)
}

function handleConnectorIconError(event: Event) {
  const target = event.target as HTMLImageElement | null
  if (target) {
    target.style.display = 'none'
  }
}

function connectorProfileDetail(item: FunctionConnectorStatus): string {
  const summary = item.profile?.resource_summary
  const parts: string[] = []
  if (item.profile?.account_name && item.profile.account_name !== connectorProfileName(item)) {
    parts.push(item.profile.account_name)
  }
  if (summary?.page_count || summary?.database_count) {
    const resourceParts: string[] = []
    if (summary.page_count) resourceParts.push(`${summary.page_count} 页面`)
    if (summary.database_count) resourceParts.push(`${summary.database_count} 数据库`)
    parts.push(resourceParts.join(' / '))
  }
  if (!parts.length && item.resolved_from) {
    parts.push(`绑定 ${item.resolved_from}`)
  }
  return parts.join(' · ')
}

function connectorAuthorizeScopes(item: FunctionConnectorStatus): string[] {
  return normalizeScopes([
    ...(item.granted_scopes || []),
    ...(item.missing_scopes || []),
    ...(item.required_scopes || [])
  ])
}

function connectorResourcePath(): string {
  return CONNECTOR_GLOBAL_RESOURCE_PATH
}

async function handleConnectConnector(provider: string, scopes: string[] = [], connectionId = '') {
  try {
    const redirectAfter = `${window.location.pathname}${window.location.search}${window.location.hash}`
    const resp = await startConnectorOAuth({
      provider,
      resource_path: connectorResourcePath(),
      connection_id: connectionId || undefined,
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
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--el-color-primary) 4%, var(--el-bg-color) 96%), var(--el-bg-color));
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
  padding: 12px;
}

.connector-bar-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 700;
}

.connector-title-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 22%, var(--el-border-color-lighter) 78%);
  border-radius: 8px;
  background: color-mix(in srgb, var(--el-color-primary) 10%, var(--el-fill-color-blank) 90%);
  color: var(--el-color-primary);
}

.connector-title-text {
  letter-spacing: 0;
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
  gap: 10px;
}

.connector-card {
  --connector-provider-color: var(--el-color-primary);
  position: relative;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  flex: 1 1 320px;
  min-width: 0;
  min-height: 82px;
  overflow: hidden;
  padding: 12px;
  border: 1px solid color-mix(in srgb, var(--connector-provider-color) 18%, var(--el-border-color-lighter) 82%);
  border-radius: 8px;
  background:
    linear-gradient(90deg, color-mix(in srgb, var(--connector-provider-color) 10%, transparent), transparent 42%),
    var(--el-fill-color-lighter);
}

.connector-card::before {
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  content: "";
  background: var(--connector-provider-color);
  opacity: 0.78;
}

.connector-card.is-ready {
  border-color: color-mix(in srgb, var(--el-color-success) 22%, var(--el-border-color-lighter) 78%);
}

.connector-card.is-scope-missing {
  border-color: color-mix(in srgb, var(--el-color-danger) 24%, var(--el-border-color-lighter) 76%);
}

.connector-card.is-disconnected {
  border-color: color-mix(in srgb, var(--el-color-warning) 24%, var(--el-border-color-lighter) 76%);
}

.connector-logo-stack {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 56px;
  width: 56px;
  min-width: 56px;
  height: 56px;
  color: var(--connector-provider-color);
}

.connector-logo-stack::before {
  position: absolute;
  inset: 4px;
  border-radius: 8px;
  content: "";
  background: color-mix(in srgb, var(--connector-provider-color) 12%, var(--el-fill-color-blank) 88%);
}

.connector-logo-stack.has-account::after {
  position: absolute;
  right: 18px;
  bottom: 15px;
  z-index: 1;
  width: 18px;
  height: 2px;
  border-radius: 999px;
  content: "";
  background: color-mix(in srgb, var(--connector-provider-color) 56%, var(--el-border-color) 44%);
  transform: rotate(-32deg);
}

.connector-logo-stack.has-account {
  justify-content: flex-start;
}

.connector-logo-box {
  position: absolute;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: 8px;
  background: var(--el-fill-color-blank);
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.08);
}

.connector-platform-logo {
  left: 6px;
  top: 6px;
  z-index: 2;
  width: 38px;
  height: 38px;
  border: 1px solid color-mix(in srgb, var(--connector-provider-color) 30%, var(--el-border-color-lighter) 70%);
  font-size: 15px;
}

.connector-account-logo {
  right: 1px;
  bottom: 1px;
  z-index: 3;
  width: 30px;
  height: 30px;
  border: 2px solid var(--el-bg-color);
  font-size: 12px;
}

.connector-logo-box img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: var(--el-fill-color-blank);
}

.connector-account-logo img {
  padding: 0;
  object-fit: cover;
}

.connector-platform-logo img,
.connector-account-logo img[src*="favicon"],
.connector-account-logo img[src$=".svg"],
.connector-account-logo img[src$=".ico"] {
  padding: 5px;
  object-fit: contain;
}

.connector-card-body {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 6px;
}

.connector-card-heading {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.connector-copy-lines {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.connector-provider {
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.connector-state-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  flex: 0 0 auto;
  height: 22px;
  padding: 0 8px;
  border: 1px solid color-mix(in srgb, var(--el-color-warning) 24%, var(--el-border-color-lighter) 76%);
  border-radius: 999px;
  background: color-mix(in srgb, var(--el-color-warning) 9%, var(--el-fill-color-blank) 91%);
  color: color-mix(in srgb, var(--el-color-warning) 78%, var(--el-text-color-primary) 22%);
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
}

.connector-state-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

.connector-card.is-ready .connector-state-pill {
  border-color: color-mix(in srgb, var(--el-color-success) 26%, var(--el-border-color-lighter) 74%);
  background: color-mix(in srgb, var(--el-color-success) 10%, var(--el-fill-color-blank) 90%);
  color: var(--el-color-success);
}

.connector-card.is-scope-missing .connector-state-pill {
  border-color: color-mix(in srgb, var(--el-color-danger) 26%, var(--el-border-color-lighter) 74%);
  background: color-mix(in srgb, var(--el-color-danger) 9%, var(--el-fill-color-blank) 91%);
  color: var(--el-color-danger);
}

.connector-status {
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.connector-profile-detail {
  overflow: hidden;
  color: var(--el-text-color-placeholder);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.connector-scopes {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}

.connector-scopes :deep(.el-tag) {
  max-width: 100%;
}

.connector-action {
  margin-left: auto;
  align-self: center;
  white-space: nowrap;
}

@media (max-width: 720px) {
  .connector-card {
    flex-basis: 100%;
    grid-template-columns: auto minmax(0, 1fr);
  }

  .connector-action {
    grid-column: 2;
    justify-self: start;
    margin-left: 0;
  }
}
</style>
