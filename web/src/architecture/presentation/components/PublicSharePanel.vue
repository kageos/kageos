<template>
  <section class="public-share-panel">
    <div class="history-card">
      <div class="section-header">
        <div class="section-heading">
          <div class="section-title">{{ t('publicSharePanel.title') }}</div>
          <div class="section-subtitle">{{ t('publicSharePanel.subtitle') }}</div>
        </div>
        <el-button type="primary" :icon="Plus" @click="openCreateDialog">{{ t('publicSharePanel.createShare') }}</el-button>
      </div>

      <div class="form-history-toolbar">
        <el-input
          v-model="filters.keyword"
          class="history-search"
          clearable
          :prefix-icon="Search"
          :placeholder="t('publicSharePanel.searchPlaceholder')"
          @keyup.enter="load"
          @clear="load"
        />
        <el-input
          v-model="filters.createdBy"
          class="history-user-select"
          clearable
          :placeholder="t('publicSharePanel.createdBy')"
          @keyup.enter="load"
          @clear="load"
        />
        <el-select
          v-model="filters.status"
          class="history-action-select"
          clearable
          :placeholder="t('publicSharePanel.status')"
          @change="load"
        >
          <el-option :label="t('publicSharePanel.statusEnabled')" value="enabled" />
          <el-option :label="t('publicSharePanel.statusDisabled')" value="disabled" />
          <el-option :label="t('publicSharePanel.statusExpired')" value="expired" />
          <el-option :label="t('shareGovernance.exhausted')" value="exhausted" />
        </el-select>
        <el-button type="primary" plain :icon="Search" @click="load">{{ t('publicSharePanel.filter') }}</el-button>
        <el-button plain :icon="Refresh" :loading="loading" @click="resetFilters">{{ t('common.reset') }}</el-button>
      </div>

      <div v-if="!loading && !loadError" class="share-list-caption">
        <span>{{ t('publicSharePanel.resultCount', { count: shares.length }) }}</span>
      </div>
      <div v-if="selectedShares.length" class="share-bulk-toolbar">
        <span class="metric-hint">{{ t('shareGovernance.scope') }}</span>
        <el-button type="danger" plain :loading="bulkClosing" @click="closeSelected">{{ t('shareGovernance.closeSelected', { count: selectedShares.length }) }}</el-button>
        <el-button text :disabled="bulkClosing" @click="selectedShares = []">{{ t('common.cancel') }}</el-button>
      </div>
      <div v-loading="loading" class="share-list" :aria-busy="loading">
        <el-result v-if="loadError" icon="warning" :title="t('publicSharePanel.loadFailed')" :sub-title="loadError">
          <template #extra><el-button @click="load">{{ t('common.refresh') }}</el-button></template>
        </el-result>
        <el-empty v-else-if="!loading && shares.length === 0" :description="hasFilters ? t('publicSharePanel.noMatches') : t('publicSharePanel.empty')" :image-size="88">
          <el-button v-if="hasFilters" @click="resetFilters">{{ t('common.reset') }}</el-button>
          <el-button v-else type="primary" :icon="Plus" @click="openCreateDialog">{{ t('publicSharePanel.createShare') }}</el-button>
        </el-empty>
        <article v-for="row in shares" v-else :key="row.share_id" class="share-row">
          <div class="share-identity">
            <el-checkbox :model-value="selectedShares.some(item => item.share_id === row.share_id)" :disabled="!row.enabled || bulkClosing" :aria-label="`${t('shareGovernance.select')} ${shareDisplayTitle(row)}`" @change="toggleSelected(row)" />
            <div class="share-information">
              <div class="share-title-line">
                <h3>{{ shareDisplayTitle(row) }}</h3>
                <el-tag size="small" :type="statusTagType(row)" effect="light">{{ statusLabel(row) }}</el-tag>
              </div>
              <p v-if="row.description" class="link-description">{{ row.description }}</p>
              <button class="url-cell" type="button" :title="publicLink(row)" @click="copyLink(publicLink(row))"><el-icon><Link /></el-icon><span>{{ publicLink(row) }}</span></button>
              <div class="share-origin">{{ t('publicSharePanel.createdBy') }} {{ row.created_by || '—' }} <span>·</span> {{ formatDate(row.created_at) }}</div>
            </div>
          </div>
          <div class="share-metric">
            <span class="metric-label">{{ t('publicSharePanel.submissionCount') }}</span>
            <strong>{{ row.use_count.toLocaleString() }}<small v-if="row.max_uses > 0"> / {{ row.max_uses.toLocaleString() }}</small></strong>
            <span class="metric-hint">{{ usageLimitText(row.max_uses) }}</span>
          </div>
          <div class="share-expiry">
            <span class="metric-label">{{ t('publicSharePanel.expirationTime') }}</span>
            <strong>{{ row.expires_at ? formatDate(row.expires_at) : t('publicSharePanel.permanent') }}</strong>
            <span class="metric-hint">{{ row.expires_at ? expiryHint(row.expires_at) : t('publicSharePanel.neverExpires') }}</span>
          </div>
          <div class="share-actions">
            <el-button type="primary" plain :icon="CopyDocument" @click="copyLink(publicLink(row))">{{ t('publicSharePanel.copyLink') }}</el-button>
            <div class="share-secondary-actions">
              <el-button text @click="openQrDialog(row)">{{ t('publicSharePanel.qrCode') }}</el-button>
              <el-button v-if="row.enabled" text type="danger" :disabled="bulkClosing" :loading="disablingId === row.share_id" @click="disableShare(row.share_id)">{{ t('publicSharePanel.close') }}</el-button>
            </div>
          </div>
        </article>
      </div>
    </div>

    <PublicShareCreateDialog
      v-model="dialogVisible"
      :full-code-path="fullCodePath"
      :default-title="functionDetail?.name || functionNode?.name || ''"
      @created="handleShareCreated"
    />

    <el-dialog
      v-model="qrDialogVisible"
      :title="t('publicSharePanel.qrDialogTitle')"
      :width="qrDialogWidth"
      class="public-share-qr-dialog"
    >
        <div class="qr-dialog-body">
          <div class="qr-title">{{ qrShare ? shareDisplayTitle(qrShare) : t('publicSharePanel.untitled') }}</div>
          <div v-if="qrShare?.description" class="qr-description">{{ qrShare.description }}</div>
          <div v-if="qrShare" class="qr-meta">
            <span>{{ qrShare.expires_at ? `${t('publicSharePanel.expirationTime')}：${formatDate(qrShare.expires_at)}` : t('publicSharePanel.permanent') }}</span>
            <span>{{ qrShare.max_uses > 0 ? `${t('publicSharePanel.submissionCount')}：${qrShare.use_count}/${qrShare.max_uses}` : t('publicSharePanel.unlimited') }}</span>
          </div>
        <div class="qr-box">
          <el-skeleton v-if="qrGenerating" :rows="5" animated />
          <img v-else-if="qrDataUrl" class="qr-image" :src="qrDataUrl" :alt="t('publicSharePanel.qrAlt')" />
          <el-empty v-else :description="t('publicSharePanel.qrGenerateFailed')" :image-size="80" />
        </div>
        <div class="qr-link-group">
          <div class="qr-link-label">{{ t('publicSharePanel.scanLink') }}</div>
          <button class="qr-link" type="button" @click="copyLink(currentQrLink)">
            {{ currentQrLink }}
          </button>
        </div>
      </div>

      <template #footer>
        <div class="qr-footer-actions">
          <el-button @click="copyLink(currentQrLink)">{{ t('publicSharePanel.copyLink') }}</el-button>
          <el-button :disabled="!qrDataUrl" @click="downloadQrCode">{{ t('publicSharePanel.downloadQr') }}</el-button>
        </div>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import QRCode from 'qrcode'
import { CopyDocument, Link, Plus, Refresh, Search } from '@element-plus/icons-vue'
import PublicShareCreateDialog from '@/architecture/presentation/components/PublicShareCreateDialog.vue'
import {
  disablePublicShare,
  listPublicShares,
  type PublicShareItem,
} from '@/architecture/presentation/context/api/publicShare'
import { getErrorMessage } from '@/architecture/shared/apiError'
import type { FunctionDetail, ServiceTree } from '@/architecture/domain/types'

const props = defineProps<{
  functionDetail: FunctionDetail | null
  functionNode: ServiceTree | null
}>()

const { t } = useI18n()

const loading = ref(false)
const loadError = ref('')
let loadGeneration = 0
const disablingId = ref('')
const selectedShares = ref<PublicShareItem[]>([])
const bulkClosing = ref(false)
const shares = ref<PublicShareItem[]>([])
const dialogVisible = ref(false)
const qrDialogVisible = ref(false)
const qrShare = ref<PublicShareItem | null>(null)
const qrDataUrl = ref('')
const qrGenerating = ref(false)
const filters = reactive({
  keyword: '',
  createdBy: '',
  status: '',
})

const hasFilters = computed(() => !!(filters.keyword || filters.createdBy || filters.status))

const qrDialogWidth = computed(() => 'min(420px, calc(100vw - 32px))')

const fullCodePath = computed(() => {
  return props.functionNode?.full_code_path || props.functionDetail?.full_code_path || props.functionDetail?.router || ''
})

function publicLink(row: PublicShareItem) {
  return row.public_url
}

function shareDisplayTitle(row: Pick<PublicShareItem, 'title' | 'share_id'>) {
  return row.title || t('publicSharePanel.untitled')
}

function usageLimitText(maxUses: number) {
  return maxUses > 0
    ? t('publicSharePanel.maxUses', { count: maxUses })
    : t('publicSharePanel.unlimited')
}

async function load() {
  const generation = ++loadGeneration
  shares.value = []
  selectedShares.value = []
  loadError.value = ''
  loading.value = false
  if (!fullCodePath.value) return
  loading.value = true
  try {
    const resp = await listPublicShares({
      full_code_path: fullCodePath.value,
      keyword: filters.keyword,
      created_by: filters.createdBy,
      status: filters.status,
    })
    if (generation === loadGeneration) shares.value = resp.items || []
  } catch (error) {
    if (generation === loadGeneration) loadError.value = getErrorMessage(error, t('publicSharePanel.loadFailed'))
  } finally {
    if (generation === loadGeneration) loading.value = false
  }
}

function resetFilters() {
  filters.keyword = ''
  filters.createdBy = ''
  filters.status = ''
  load()
}

function openCreateDialog() {
  dialogVisible.value = true
}

function handleShareCreated() {
  void load()
}

async function disableShare(shareId: string) {
  disablingId.value = shareId
  try {
    await disablePublicShare(shareId)
    await load()
    ElMessage.success(t('publicSharePanel.closeSuccess'))
  } catch (error) {
    ElMessage.error(getErrorMessage(error, t('publicSharePanel.closeFailed')))
  } finally {
    disablingId.value = ''
  }
}

function toggleSelected(row: PublicShareItem) {
  selectedShares.value = selectedShares.value.some(item => item.share_id === row.share_id)
    ? selectedShares.value.filter(item => item.share_id !== row.share_id)
    : [...selectedShares.value, row]
}
async function closeSelected() {
  const selected = [...selectedShares.value]
  if (!selected.length || bulkClosing.value) return
  try { await ElMessageBox.confirm(t('shareGovernance.confirm', { count: selected.length }), t('publicSharePanel.close'), { type: 'warning' }) }
  catch { return }
  bulkClosing.value = true
  let failed = 0
  try {
    for (const row of selected) {
      try { await disablePublicShare(row.share_id) } catch { failed++ }
    }
    await load()
    if (failed) ElMessage.warning(t('shareGovernance.partial', { count: failed }))
    else ElMessage.success(t('publicSharePanel.closeSuccess'))
  } catch { ElMessage.error(t('shareGovernance.refreshFailed')) }
  finally { bulkClosing.value = false }
}

async function copyLink(link: string) {
  if (!link) {
    return
  }
  try {
    await navigator.clipboard.writeText(link)
    ElMessage.success(t('publicSharePanel.linkCopied'))
  } catch {
    ElMessage.error(t('publicSharePanel.copyFailed'))
  }
}

const currentQrLink = computed(() => {
  return qrShare.value ? publicLink(qrShare.value) : ''
})

async function openQrDialog(share: PublicShareItem) {
  qrShare.value = share
  qrDialogVisible.value = true
  qrDataUrl.value = ''
  qrGenerating.value = true
  try {
    qrDataUrl.value = await QRCode.toDataURL(publicLink(share), {
      errorCorrectionLevel: 'M',
      margin: 2,
      width: 256,
      color: {
        dark: '#111827',
        light: '#ffffff',
      },
    })
  } catch (_error) {
    ElMessage.error(t('publicSharePanel.qrGenerateFailed'))
  } finally {
    qrGenerating.value = false
  }
}

function qrFileName(share: PublicShareItem) {
  const safeName = safeQrBaseName(share)
  return `${safeName}-qrcode.png`
}

function safeQrBaseName(share: PublicShareItem) {
  return (share.title || share.share_id || 'public-share')
    .trim()
    .replace(/[\\/:*?"<>|]+/g, '-')
}

function downloadQrCode() {
  if (!qrDataUrl.value || !qrShare.value) {
    return
  }
  const link = document.createElement('a')
  link.href = qrDataUrl.value
  link.download = qrFileName(qrShare.value)
  link.click()
}

function isExpired(value?: string) {
  return !!value && new Date(value).getTime() <= Date.now()
}

function statusLabel(row: PublicShareItem) {
  if (!row.enabled) return t('publicSharePanel.statusDisabled')
  if (isExpired(row.expires_at)) return t('publicSharePanel.statusExpired')
  if (row.max_uses > 0 && row.use_count >= row.max_uses) return t('shareGovernance.exhausted')
  return t('publicSharePanel.statusEnabled')
}

function statusTagType(row: PublicShareItem) {
  if (!row.enabled) return 'info'
  if (isExpired(row.expires_at) || (row.max_uses > 0 && row.use_count >= row.max_uses)) return 'warning'
  return 'success'
}

function expiryHint(value: string) {
  const diff = new Date(value).getTime() - Date.now()
  if (diff <= 0) {
    return t('publicSharePanel.statusExpired')
  }
  const days = Math.ceil(diff / (24 * 60 * 60 * 1000))
  return t('publicSharePanel.expiresInDays', { count: days })
}

function formatDate(value: string) {
  return new Date(value).toLocaleString()
}

watch(fullCodePath, load, { immediate: true })
onBeforeUnmount(() => { loadGeneration++ })
</script>

<style scoped lang="scss">
.public-share-panel :deep(.el-button) { box-shadow: none; }

.public-share-panel { min-width: 0; container-type: inline-size; }
.history-card { min-width: 0; }
.section-header { display: flex; justify-content: space-between; align-items: center; gap: 20px; padding: 4px 0 24px; }
.section-heading { min-width: 0; }
.section-title { font-size: 20px; font-weight: 600; color: var(--el-text-color-primary); }
.section-subtitle { margin-top: 8px; font-size: 13px; line-height: 1.6; color: var(--el-text-color-secondary); }
.form-history-toolbar { display: grid; grid-template-columns: minmax(180px, 1fr) minmax(100px, 150px) minmax(110px, 140px) auto auto; align-items: center; gap: 10px; }
.form-history-toolbar > * { min-width: 0; width: 100%; }
.form-history-toolbar .el-button { margin: 0; }
.share-list-caption { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 8px; padding: 16px 0; color: var(--el-text-color-secondary); font-size: 12px; }
.share-bulk-toolbar { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; padding: 12px; margin-bottom: 12px; border-radius: 8px; background: var(--el-fill-color-light); }
.share-list { min-height: 180px; border: 1px solid var(--el-border-color-lighter); border-radius: 12px; overflow: hidden; background: var(--el-bg-color); }
.share-row { display: grid; grid-template-columns: minmax(220px, 1fr) 110px 180px 126px; align-items: center; gap: 24px; padding: 24px; }
.share-row + .share-row { border-top: 1px solid var(--el-border-color-lighter); }
.share-row:hover { background: var(--el-fill-color-lighter); }
.share-identity { display: flex; align-items: flex-start; gap: 14px; min-width: 0; }
.share-identity > .el-checkbox { flex: none; height: 26px; }
.share-information { min-width: 0; }
.share-title-line { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.share-title-line h3 { margin: 0; font-size: 15px; font-weight: 600; line-height: 1.6; overflow-wrap: anywhere; color: var(--el-text-color-primary); }
.link-description { margin: 6px 0; font-size: 13px; line-height: 1.6; overflow-wrap: anywhere; color: var(--el-text-color-regular); }
.url-cell { display: flex; align-items: center; gap: 6px; max-width: 100%; margin: 8px 0; padding: 0; border: 0; background: none; font: inherit; font-size: 12px; color: var(--el-color-primary); cursor: pointer; }
.url-cell .el-icon { flex: none; }
.url-cell span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.share-origin { display: flex; flex-wrap: wrap; gap: 6px; color: var(--el-text-color-placeholder); font-size: 11px; line-height: 1.6; }
.share-metric, .share-expiry { display: flex; flex-direction: column; align-self: center; gap: 7px; min-width: 0; }
.metric-label, .metric-hint { font-size: 12px; color: var(--el-text-color-secondary); line-height: 1.5; }
.share-metric strong { font-size: 22px; font-weight: 600; font-variant-numeric: tabular-nums; color: var(--el-text-color-primary); }
.share-metric small { font-size: 12px; font-weight: 400; color: var(--el-text-color-secondary); }
.share-expiry strong { font-size: 13px; font-weight: 500; line-height: 1.6; color: var(--el-text-color-primary); }
.share-actions { display: flex; flex-direction: column; gap: 8px; }
.share-secondary-actions { display: flex; justify-content: center; }
.share-secondary-actions .el-button { margin: 0; }
button:focus-visible { outline: 2px solid var(--el-color-primary); outline-offset: 3px; }
@container (max-width: 980px) {
  .share-row { grid-template-columns: minmax(0, 1fr) 150px; gap: 20px; }
  .share-identity { grid-column: 1 / -1; }
  .share-metric { padding-left: 28px; }
  .share-actions { grid-column: 1 / -1; flex-direction: row; justify-content: flex-end; border-top: 1px solid var(--el-border-color-extra-light); padding-top: 12px; }
}
@container (max-width: 640px) {
  .section-header { align-items: flex-start; flex-direction: column; gap: 14px; }
  .form-history-toolbar { grid-template-columns: 1fr 1fr; }
  .history-search { grid-column: 1 / -1; }
  .share-row { padding: 18px 14px; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 16px; }
  .share-metric { padding-left: 0; }
  .share-list-caption { line-height: 1.6; }
}
.qr-dialog-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
}

.qr-title {
  max-width: 100%;
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 700;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.qr-description {
  max-width: 100%;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.55;
  text-align: center;
  overflow-wrap: anywhere;
}

.qr-meta {
  display: flex;
  max-width: 100%;
  flex-wrap: wrap;
  justify-content: center;
  gap: 6px 14px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.qr-box {
  display: grid;
  place-items: center;
  width: 288px;
  min-height: 288px;
  padding: 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: #fff;
}

.qr-image {
  display: block;
  width: 256px;
  height: 256px;
}

.qr-link {
  display: block;
  width: 100%;
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--el-color-primary);
  font: inherit;
  font-size: 13px;
  line-height: 1.5;
  text-align: center;
  word-break: break-all;
  cursor: pointer;
}

.qr-link-group {
  width: 100%;
}

.qr-link-label {
  margin-bottom: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
  text-align: center;
}

.qr-footer-actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
}

.qr-footer-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}


@media (max-width: 520px) {
  .qr-box { width: min(288px, calc(100vw - 80px)); min-height: auto; padding: 12px; box-sizing: border-box; }
  .qr-image { width: 100%; height: auto; }
  .qr-footer-actions { justify-content: center; }
}
</style>
