<template>
  <section class="public-share-panel">
    <div class="history-card">
      <div class="section-header">
        <div class="section-heading">
          <div class="section-title">公开链接</div>
          <div class="section-subtitle">管理这个 Form 的外部提交入口。</div>
        </div>
        <el-button type="primary" size="small" @click="openCreateDialog">创建公开链接</el-button>
      </div>

      <div class="form-history-toolbar">
        <el-input
          v-model="filters.keyword"
          class="history-search"
          size="small"
          clearable
          :prefix-icon="Search"
          placeholder="搜索标题、描述、链接 ID"
          @keyup.enter="load"
          @clear="load"
        />
        <el-input
          v-model="filters.createdBy"
          class="history-user-select"
          size="small"
          clearable
          placeholder="创建人"
          @keyup.enter="load"
          @clear="load"
        />
        <el-select
          v-model="filters.status"
          class="history-action-select"
          size="small"
          clearable
          placeholder="状态"
          @change="load"
        >
          <el-option label="启用中" value="enabled" />
          <el-option label="已关闭" value="disabled" />
          <el-option label="已过期" value="expired" />
        </el-select>
        <el-button size="small" type="primary" plain :icon="Search" @click="load">筛选</el-button>
        <el-button size="small" link :icon="Refresh" :loading="loading" @click="resetFilters">重置</el-button>
      </div>

      <div v-loading="loading" class="mobile-share-list">
        <el-empty v-if="shares.length === 0" description="还没有公开链接" :image-size="80" />
        <article v-for="row in shares" :key="row.share_id" class="mobile-share-card">
          <div class="mobile-share-head">
            <div class="mobile-share-title">
              <div class="title-name">{{ row.title || '未命名公开链接' }}</div>
              <div v-if="row.description" class="link-description">{{ row.description }}</div>
            </div>
            <el-tag size="small" :type="statusTagType(row)" effect="light" round>
              {{ statusLabel(row) }}
            </el-tag>
          </div>

          <button class="mobile-share-url" type="button" @click="copyLink(publicLink(row))">
            {{ publicLink(row) }}
          </button>

          <div class="mobile-share-meta">
            <div>
              <span>提交</span>
              <strong>{{ row.use_count }}</strong>
              <em>{{ row.max_uses > 0 ? `最多 ${row.max_uses}` : '不限次数' }}</em>
            </div>
            <div>
              <span>过期</span>
              <strong>{{ row.expires_at ? expiryHint(row.expires_at) : '永久有效' }}</strong>
              <em>{{ row.expires_at ? formatDate(row.expires_at) : '不过期' }}</em>
            </div>
          </div>

          <div class="mobile-share-foot">
            <span>{{ row.created_by || '-' }} · {{ formatDate(row.created_at) }}</span>
            <div class="mobile-share-actions">
              <el-button size="small" text @click="copyLink(publicLink(row))">复制</el-button>
              <el-button size="small" text @click="openQrDialog(row)">二维码</el-button>
              <el-button
                v-if="row.enabled"
                size="small"
                text
                type="danger"
                :loading="disablingId === row.share_id"
                @click="disableShare(row.share_id)"
              >
                关闭
              </el-button>
            </div>
          </div>
        </article>
      </div>

      <el-table
        v-loading="loading"
        :data="shares"
        stripe
        class="history-table"
        empty-text="还没有公开链接"
      >
        <el-table-column label="标题" min-width="220">
          <template #default="{ row }">
            <div class="title-cell">
              <div class="title-name">{{ row.title || '未命名公开链接' }}</div>
              <div v-if="row.description" class="link-description">{{ row.description }}</div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="公开链接" min-width="260">
          <template #default="{ row }">
            <button class="url-cell" type="button" @click="copyLink(publicLink(row))">
              {{ publicLink(row) }}
            </button>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="statusTagType(row)" effect="light" round>
              {{ statusLabel(row) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="创建人" width="150">
          <template #default="{ row }">
            <span class="muted-text">{{ row.created_by || '-' }}</span>
          </template>
        </el-table-column>

        <el-table-column label="提交次数" width="130" align="center">
          <template #default="{ row }">
            <div class="count-cell">
              <div>{{ row.use_count }}</div>
              <span>{{ row.max_uses > 0 ? `最多 ${row.max_uses}` : '不限次数' }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="过期时间" width="190">
          <template #default="{ row }">
            <div class="time-cell">
              <div>{{ row.expires_at ? formatDate(row.expires_at) : '永久有效' }}</div>
              <span>{{ row.expires_at ? expiryHint(row.expires_at) : '不过期' }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            <div class="time-cell">
              <div>{{ formatDate(row.created_at) }}</div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="190" align="right" fixed="right">
          <template #default="{ row }">
            <div class="action-cell">
              <el-button text @click="copyLink(publicLink(row))">复制</el-button>
              <el-button text @click="openQrDialog(row)">二维码</el-button>
              <el-button
                v-if="row.enabled"
                text
                type="danger"
                :loading="disablingId === row.share_id"
                @click="disableShare(row.share_id)"
              >
                关闭
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog
      v-model="dialogVisible"
      title="创建公开链接"
      :width="createDialogWidth"
      :close-on-click-modal="false"
      class="public-share-dialog"
    >
      <el-form label-position="top">
        <el-form-item label="标题">
          <el-input v-model="createForm.title" maxlength="80" show-word-limit placeholder="可选，用于区分不同公开链接" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="createForm.description"
            type="textarea"
            :rows="3"
            maxlength="300"
            show-word-limit
            placeholder="可选，会展示在公开表单页面"
          />
        </el-form-item>
        <el-form-item label="过期时间">
          <el-radio-group v-model="expireMode">
            <el-radio-button label="never">不过期</el-radio-button>
            <el-radio-button label="custom">自定义</el-radio-button>
          </el-radio-group>
          <el-date-picker
            v-if="expireMode === 'custom'"
            v-model="customExpiresAt"
            type="datetime"
            placeholder="选择过期时间"
            class="custom-expire-picker"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
          />
        </el-form-item>

        <el-form-item label="提交次数">
          <el-radio-group v-model="limitMode">
            <el-radio-button label="unlimited">不限次数</el-radio-button>
            <el-radio-button label="limited">限制次数</el-radio-button>
          </el-radio-group>
          <el-input-number
            v-if="limitMode === 'limited'"
            v-model="maxUses"
            :min="1"
            :step="10"
            controls-position="right"
            class="max-uses-input"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="createShare">创建并生成二维码</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="qrDialogVisible"
      title="公开链接二维码"
      :width="qrDialogWidth"
      class="public-share-qr-dialog"
    >
      <div class="qr-dialog-body">
        <div class="qr-title">{{ qrShare?.title || '未命名公开链接' }}</div>
        <div class="qr-box">
          <el-skeleton v-if="qrGenerating" :rows="5" animated />
          <img v-else-if="qrDataUrl" class="qr-image" :src="qrDataUrl" alt="公开链接二维码" />
          <el-empty v-else description="二维码生成失败" :image-size="80" />
        </div>
        <div class="qr-storage">
          <el-tag v-if="qrUploading" size="small" type="info" effect="light">
            正在上传二维码 {{ qrUploadPercent }}%
          </el-tag>
          <el-tag v-else-if="qrStorageUrl" size="small" type="success" effect="light">
            已上传到存储
          </el-tag>
          <el-tag v-else-if="qrUploadError" size="small" type="warning" effect="light">
            {{ qrUploadError }}
          </el-tag>
        </div>
        <div class="qr-link-group">
          <div class="qr-link-label">扫码链接</div>
          <button class="qr-link" type="button" @click="copyLink(currentQrLink)">
            {{ currentQrLink }}
          </button>
        </div>
        <div v-if="backendQrLink && backendQrLink !== currentQrLink" class="qr-link-group">
          <div class="qr-link-label">后端链接</div>
          <button class="qr-link" type="button" @click="copyLink(backendQrLink)">
            {{ backendQrLink }}
          </button>
        </div>
        <button v-if="qrStorageUrl" class="qr-link" type="button" @click="copyLink(qrStorageUrl)">
          {{ qrStorageUrl }}
        </button>
        <div v-if="qrStorageRef" class="qr-ref">文件引用：{{ qrStorageRef }}</div>
      </div>

      <template #footer>
        <div class="qr-footer-actions">
          <el-button @click="copyLink(currentQrLink)">复制链接</el-button>
          <el-button :disabled="!qrStorageUrl" @click="copyLink(qrStorageUrl)">复制图片地址</el-button>
          <el-button :disabled="!qrDataUrl" @click="downloadQrCode">下载二维码</el-button>
        </div>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import QRCode from 'qrcode'
import { Refresh, Search } from '@element-plus/icons-vue'
import { notifyUploadComplete, uploadFile } from '@/architecture/presentation/context/uploadContext'
import {
  createPublicShare,
  disablePublicShare,
  listPublicShares,
  type PublicShareItem,
} from '@/architecture/presentation/context/api/publicShare'
import type { FunctionDetail, ServiceTree } from '@/architecture/domain/types'

const props = defineProps<{
  functionDetail: FunctionDetail | null
  functionNode: ServiceTree | null
}>()

const loading = ref(false)
const creating = ref(false)
const disablingId = ref('')
const shares = ref<PublicShareItem[]>([])
const dialogVisible = ref(false)
const qrDialogVisible = ref(false)
const qrShare = ref<PublicShareItem | null>(null)
const qrDataUrl = ref('')
const qrGenerating = ref(false)
const qrUploading = ref(false)
const qrUploadPercent = ref(0)
const qrStorageUrl = ref('')
const qrStorageRef = ref('')
const qrUploadError = ref('')
const qrStorageCache = new Map<string, { url: string; ref: string }>()
const expireMode = ref<'never' | 'custom'>('never')
const limitMode = ref<'unlimited' | 'limited'>('unlimited')
const customExpiresAt = ref('')
const maxUses = ref(100)
const createForm = reactive({
  title: '',
  description: '',
})
const filters = reactive({
  keyword: '',
  createdBy: '',
  status: '',
})

const createDialogWidth = computed(() => 'min(520px, calc(100vw - 32px))')
const qrDialogWidth = computed(() => 'min(420px, calc(100vw - 32px))')

const fullCodePath = computed(() => {
  return props.functionNode?.full_code_path || props.functionDetail?.full_code_path || props.functionDetail?.router || ''
})

function pageURL(shareId: string) {
  return `${window.location.origin}/public/s/${shareId}`
}

function normalizePublicURL(value: string) {
  if (!value) {
    return ''
  }
  return new URL(value, window.location.origin).toString()
}

function publicLink(row: PublicShareItem) {
  if (import.meta.env.DEV) {
    return pageURL(row.share_id)
  }
  return officialPublicLink(row) || pageURL(row.share_id)
}

function officialPublicLink(row: PublicShareItem) {
  return normalizePublicURL(row.public_url || '')
}

async function load() {
  if (!fullCodePath.value) {
    return
  }
  loading.value = true
  try {
    const resp = await listPublicShares({
      full_code_path: fullCodePath.value,
      keyword: filters.keyword,
      created_by: filters.createdBy,
      status: filters.status,
    })
    shares.value = resp.items || []
  } finally {
    loading.value = false
  }
}

function resetFilters() {
  filters.keyword = ''
  filters.createdBy = ''
  filters.status = ''
  load()
}

function openCreateDialog() {
  createForm.title = ''
  createForm.description = ''
  expireMode.value = 'never'
  limitMode.value = 'unlimited'
  customExpiresAt.value = ''
  maxUses.value = 100
  dialogVisible.value = true
}

async function createShare() {
  if (!fullCodePath.value) {
    ElMessage.warning('当前表单路径未加载完成')
    return
  }
  if (expireMode.value === 'custom' && !customExpiresAt.value) {
    ElMessage.warning('请选择过期时间')
    return
  }
  creating.value = true
  try {
    const share = await createPublicShare({
      full_code_path: fullCodePath.value,
      title: createForm.title.trim(),
      description: createForm.description.trim(),
      expires_at: expireMode.value === 'custom' ? customExpiresAt.value : null,
      max_uses: limitMode.value === 'limited' ? maxUses.value : 0,
    })
    shares.value = [share, ...shares.value]
    dialogVisible.value = false
    await copyLink(publicLink(share))
    await openQrDialog(share)
    ElMessage.success('公开链接已创建，可扫码提交表单')
  } finally {
    creating.value = false
  }
}

async function disableShare(shareId: string) {
  disablingId.value = shareId
  try {
    await disablePublicShare(shareId)
    await load()
    ElMessage.success('公开链接已关闭')
  } finally {
    disablingId.value = ''
  }
}

async function copyLink(link: string) {
  if (!link) {
    return
  }
  await navigator.clipboard.writeText(link)
  ElMessage.success('链接已复制')
}

const currentQrLink = computed(() => {
  return qrShare.value ? publicLink(qrShare.value) : ''
})

const backendQrLink = computed(() => {
  return qrShare.value ? officialPublicLink(qrShare.value) : ''
})

async function openQrDialog(share: PublicShareItem) {
  qrShare.value = share
  qrDialogVisible.value = true
  qrDataUrl.value = ''
  qrGenerating.value = true
  resetQrStorageState()
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
    const cached = qrStorageCache.get(share.share_id)
    if (cached) {
      qrStorageUrl.value = cached.url
      qrStorageRef.value = cached.ref
    } else {
      await uploadQrCode(share, qrDataUrl.value)
    }
  } catch (error) {
    ElMessage.error('二维码生成失败')
  } finally {
    qrGenerating.value = false
  }
}

function resetQrStorageState() {
  qrUploading.value = false
  qrUploadPercent.value = 0
  qrStorageUrl.value = ''
  qrStorageRef.value = ''
  qrUploadError.value = ''
}

async function uploadQrCode(share: PublicShareItem, dataUrl: string) {
  if (!dataUrl || !fullCodePath.value) {
    return
  }

  qrUploading.value = true
  qrUploadPercent.value = 0
  qrUploadError.value = ''

  try {
    const blob = dataUrlToBlob(dataUrl)
    const file = new File([blob], qrFileName(share), { type: 'image/png' })
    const uploadResult = await uploadFile(fullCodePath.value, file, (progress) => {
      qrUploadPercent.value = progress.percent
    })
    const fileInfo = uploadResult.fileInfo
    if (!fileInfo) {
      throw new Error('二维码上传结果缺少文件信息')
    }
    const complete = await notifyUploadComplete({
      key: fileInfo.key,
      bucket: fileInfo.bucket,
      success: true,
      router: fileInfo.router,
      file_name: fileInfo.file_name,
      description: `公开链接二维码：${share.title || share.share_id}`,
      file_size: fileInfo.file_size,
      content_type: fileInfo.content_type,
      hash: fileInfo.hash,
      storage: uploadResult.storage,
    })

    if (!complete?.download_url) {
      throw new Error('二维码上传完成但未返回图片地址')
    }

    qrStorageUrl.value = complete.download_url
    qrStorageRef.value = complete.ref || uploadResult.ref || ''
    qrStorageCache.set(share.share_id, {
      url: qrStorageUrl.value,
      ref: qrStorageRef.value,
    })
  } catch (error) {
    qrUploadError.value = '上传存储失败，可先下载二维码'
  } finally {
    qrUploading.value = false
  }
}

function dataUrlToBlob(dataUrl: string) {
  const [header, base64Data] = dataUrl.split(',')
  if (!header || !base64Data) {
    throw new Error('无效的二维码图片数据')
  }
  const mimeMatch = header.match(/^data:(.*?);base64$/)
  const mime = mimeMatch?.[1] || 'image/png'
  const binary = window.atob(base64Data)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i)
  }
  return new Blob([bytes], { type: mime })
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
  if (!row.enabled) return '已关闭'
  if (isExpired(row.expires_at)) return '已过期'
  return '启用中'
}

function statusTagType(row: PublicShareItem) {
  if (!row.enabled) return 'info'
  if (isExpired(row.expires_at)) return 'warning'
  return 'success'
}

function expiryHint(value: string) {
  const diff = new Date(value).getTime() - Date.now()
  if (diff <= 0) {
    return '已过期'
  }
  const days = Math.ceil(diff / (24 * 60 * 60 * 1000))
  return `${days} 天后过期`
}

function formatDate(value: string) {
  return new Date(value).toLocaleString()
}

onMounted(load)
watch(fullCodePath, load)
</script>

<style scoped lang="scss">
.public-share-panel {
  height: 100%;
  min-height: 0;
  padding: 0;
}

.history-card {
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: var(--el-bg-color);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.04);
}

.section-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px 14px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  background: var(--el-fill-color-blank);
}

.section-heading {
  min-width: 0;
}

.section-title {
  font-size: 15px;
  font-weight: 700;
  line-height: 1.4;
  color: var(--el-text-color-primary);
}

.section-subtitle {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}

.form-history-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  background: var(--el-fill-color-lighter);
}

.history-search {
  flex: 1 1 320px;
  min-width: 220px;
}

.history-user-select {
  width: 180px;
}

.history-action-select {
  width: 140px;
}

.history-table {
  width: 100%;
}

.mobile-share-list {
  display: none;
}

.history-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.history-table :deep(.el-table__header th.el-table__cell) {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-weight: 700;
}

.history-table :deep(.el-table__cell) {
  vertical-align: top;
}

.history-table :deep(.cell) {
  padding-top: 5px;
  padding-bottom: 5px;
}

.title-cell {
  min-width: 0;
}

.title-name {
  line-height: 1.25;
  color: var(--el-text-color-primary);
  font-weight: 600;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.url-cell {
  display: block;
  max-width: 100%;
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--el-color-primary);
  font: inherit;
  font-size: 13px;
  line-height: 1.25;
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}

.link-description,
.muted-text,
.count-cell span,
.time-cell span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.link-description {
  margin-top: 2px;
  line-height: 1.25;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.count-cell,
.time-cell {
  line-height: 1.28;
  font-size: 13px;
}

.action-cell {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
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

.qr-storage {
  min-height: 24px;
}

.qr-ref {
  width: 100%;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
  text-align: center;
  word-break: break-all;
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

.custom-expire-picker,
.max-uses-input {
  display: block;
  width: 100%;
  margin-top: 12px;
}

@media (max-width: 820px) {
  .public-share-panel {
    overflow-x: hidden;
  }

  .history-card {
    border-radius: 8px;
  }

  .section-header,
  .form-history-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .history-search,
  .history-user-select,
  .history-action-select {
    width: 100%;
  }

  .section-header {
    gap: 12px;
    padding: 14px;
  }

  .section-header .el-button {
    width: 100%;
  }

  .form-history-toolbar {
    padding: 12px;
  }

  .form-history-toolbar > .el-button {
    width: 100%;
  }

  .history-table {
    display: none;
  }

  .mobile-share-list {
    display: grid;
    gap: 10px;
    padding: 12px;
    background: var(--el-fill-color-lighter);
  }

  .mobile-share-card {
    min-width: 0;
    padding: 12px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    background: var(--el-bg-color);
  }

  .mobile-share-head,
  .mobile-share-foot {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 10px;
  }

  .mobile-share-title {
    min-width: 0;
    flex: 1;
  }

  .mobile-share-url {
    display: block;
    width: 100%;
    margin: 10px 0 0;
    padding: 8px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background: var(--el-fill-color-light);
    color: var(--el-color-primary);
    font: inherit;
    font-size: 12px;
    line-height: 1.45;
    text-align: left;
    word-break: break-all;
    cursor: pointer;
  }

  .mobile-share-meta {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    gap: 8px;
    margin-top: 10px;
  }

  .mobile-share-meta > div {
    min-width: 0;
    padding: 8px;
    border-radius: 6px;
    background: var(--el-fill-color-lighter);
  }

  .mobile-share-meta span,
  .mobile-share-meta em,
  .mobile-share-foot > span {
    display: block;
    color: var(--el-text-color-secondary);
    font-size: 11px;
    font-style: normal;
    line-height: 1.4;
    overflow-wrap: anywhere;
  }

  .mobile-share-meta strong {
    display: block;
    margin: 2px 0;
    color: var(--el-text-color-primary);
    font-size: 13px;
    line-height: 1.4;
    overflow-wrap: anywhere;
  }

  .mobile-share-foot {
    align-items: center;
    margin-top: 10px;
    padding-top: 8px;
    border-top: 1px solid var(--el-border-color-extra-light);
  }

  .mobile-share-actions {
    display: flex;
    justify-content: flex-end;
    flex-wrap: wrap;
    flex: 0 0 auto;
  }

  .mobile-share-actions :deep(.el-button + .el-button) {
    margin-left: 0;
  }

  .public-share-dialog :deep(.el-dialog__body),
  .public-share-qr-dialog :deep(.el-dialog__body) {
    padding: 14px 16px;
  }

  .public-share-dialog :deep(.el-dialog__footer),
  .public-share-qr-dialog :deep(.el-dialog__footer) {
    padding: 0 16px 16px;
  }

  .public-share-dialog :deep(.el-dialog__footer .el-button) {
    width: 100%;
    margin-left: 0;
  }

  .public-share-dialog :deep(.el-dialog__footer .el-button + .el-button) {
    margin-top: 8px;
  }

  .qr-box {
    width: min(288px, calc(100vw - 80px));
    min-height: min(288px, calc(100vw - 80px));
    padding: 12px;
  }

  .qr-image {
    width: min(256px, calc(100vw - 104px));
    height: min(256px, calc(100vw - 104px));
  }

  .qr-footer-actions {
    justify-content: stretch;
  }

  .qr-footer-actions .el-button {
    width: 100%;
  }
}
</style>
