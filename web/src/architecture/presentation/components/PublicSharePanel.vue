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
            <button class="url-cell" type="button" @click="copyLink(pageURL(row.share_id))">
              {{ pageURL(row.share_id) }}
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

        <el-table-column label="操作" width="150" align="right" fixed="right">
          <template #default="{ row }">
            <div class="action-cell">
              <el-button text @click="copyLink(pageURL(row.share_id))">复制</el-button>
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
      width="520px"
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
        <el-button type="primary" :loading="creating" @click="createShare">创建并复制链接</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Refresh, Search } from '@element-plus/icons-vue'
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

const fullCodePath = computed(() => {
  return props.functionNode?.full_code_path || props.functionDetail?.full_code_path || props.functionDetail?.router || ''
})

function pageURL(shareId: string) {
  return `${window.location.origin}/public/s/${shareId}`
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
    await copyLink(pageURL(share.share_id))
    ElMessage.success('公开链接已创建')
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
}

.custom-expire-picker,
.max-uses-input {
  display: block;
  width: 100%;
  margin-top: 12px;
}

@media (max-width: 820px) {
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
}
</style>
