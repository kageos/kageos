<template>
  <el-dialog
    v-model="dialogVisible"
    class="global-resource-search-dialog"
    width="1180px"
    :append-to-body="true"
    :close-on-click-modal="true"
    @opened="handleOpened"
  >
    <template #header>
      <div class="global-search-header">
        <div class="global-search-mark">
          <span class="mark-core">
            <el-icon><Search /></el-icon>
          </span>
          <span class="mark-ring mark-ring-a"></span>
          <span class="mark-ring mark-ring-b"></span>
        </div>
        <div class="global-search-copy">
          <div class="global-search-kicker">{{ t('globalSearch.kicker') }}</div>
          <div class="global-search-title">{{ t('globalSearch.title') }}</div>
          <div class="global-search-subtitle">{{ t('globalSearch.subtitle') }}</div>
        </div>
      </div>
    </template>

    <div class="global-search-body">
      <div class="global-search-console">
        <div class="console-prefix">{{ t('globalSearch.query') }}</div>
        <el-input
          ref="searchInputRef"
          v-model="searchKeyword"
          size="large"
          class="global-search-input"
          :placeholder="t('globalSearch.placeholder')"
          clearable
          :prefix-icon="Search"
          @keyup.enter="runSearchNow"
          @clear="handleClear"
        />
      </div>

      <div class="global-search-tabs">
        <button
          v-for="tab in resourceTabs"
          :key="tab.value"
          type="button"
          :class="['global-search-tab', { active: activeType === tab.value }]"
          @click="activeType = tab.value"
        >
          {{ tab.label }}
        </button>
      </div>

      <div class="global-search-result-meta">
        <span class="meta-dot"></span>
        <span v-if="hasSearched && !loading">{{ t('globalSearch.hitCount', { count: total }) }}</span>
        <span v-else-if="loading">{{ t('globalSearch.scanning') }}</span>
        <span v-else>{{ t('globalSearch.waiting') }}</span>
      </div>

      <div v-loading="loading" class="global-search-results">
        <button
          v-for="item in results"
          :key="`${item.type}-${item.id}-${item.full_code_path}`"
          type="button"
          class="global-search-result"
          @click="handleSelect(item)"
        >
          <span :class="['result-icon', `type-${item.type}`]">
            <img
              v-if="getAssetIcon(item)"
              :src="getAssetIcon(item) || ''"
              :alt="getTypeLabel(item)"
              class="result-icon-img"
            />
            <el-icon v-else>
              <component :is="getComponentIcon(item)" />
            </el-icon>
          </span>
          <span class="result-main">
            <span class="result-title-row">
              <span class="result-title" :title="getResourceTitle(item)">{{ getDisplayTitle(item) }}</span>
              <span class="result-type">{{ getTypeLabel(item) }}</span>
              <span
                v-if="shouldShowHeat(item)"
                class="result-heat"
                :title="t('globalSearch.heatTitle', { count: item.run_count || 0 })"
              >
                {{ t('globalSearch.heat', { count: formatHeatCount(item.run_count || 0) }) }}
              </span>
            </span>
            <span class="result-path" :title="item.full_code_path || t('globalSearch.missingPath')">
              {{ getDisplayPath(item) }}
            </span>
          </span>
          <span class="result-side">
            <span class="side-description" :title="getResultSnippet(item)">
              {{ getDisplaySnippet(item) }}
            </span>
          </span>
        </button>

        <el-empty
          v-if="!loading && hasSearched && results.length === 0"
          :description="t('globalSearch.noResults')"
          :image-size="96"
        />

        <div v-if="!loading && !hasSearched" class="global-search-placeholder">
          <span class="placeholder-orbit">
            <el-icon><Search /></el-icon>
          </span>
          <span>{{ t('globalSearch.placeholderText') }}</span>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Document, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { resolveWorkspaceUrl } from '@/architecture/shared/routing/route'
import {
  searchResources,
  type ResourceSearchResult,
  type SearchResourceType
} from '@/architecture/presentation/context/api/service-tree'
import ChartIcon from '@/architecture/presentation/shared/components/icons/ChartIcon.vue'
import TableIcon from '@/architecture/presentation/shared/components/icons/TableIcon.vue'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
}>()

const router = useRouter()
const { t } = useI18n()
const searchInputRef = ref()
const searchKeyword = ref('')
const activeType = ref<SearchResourceType>('all')
const loading = ref(false)
const hasSearched = ref(false)
const results = ref<ResourceSearchResult[]>([])
const total = ref(0)
let searchTimer: ReturnType<typeof setTimeout> | null = null
let searchSeq = 0

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value)
})

const resourceTabs = computed<Array<{ label: string; value: SearchResourceType }>>(() => [
  { label: t('globalSearch.all'), value: 'all' },
  { label: t('globalSearch.directory'), value: 'package' },
  { label: t('globalSearch.function'), value: 'function' },
  { label: t('globalSearch.docs'), value: 'docs' }
])

function handleOpened() {
  nextTick(() => {
    searchInputRef.value?.focus?.()
  })
}

function handleClear() {
  results.value = []
  total.value = 0
  hasSearched.value = false
}

function scheduleSearch() {
  if (!dialogVisible.value) return
  if (searchTimer) {
    clearTimeout(searchTimer)
  }
  searchTimer = setTimeout(() => {
    runSearchNow()
  }, 280)
}

async function runSearchNow() {
  const keyword = searchKeyword.value.trim()
  if (!keyword) {
    handleClear()
    return
  }

  const currentSeq = ++searchSeq
  loading.value = true
  hasSearched.value = true
  try {
    const resp = await searchResources({
      keyword,
      resource_type: activeType.value,
      page: 1,
      page_size: 30
    })
    if (currentSeq !== searchSeq) return
    results.value = resp.items || []
    total.value = resp.total || 0
  } catch (error: any) {
    if (currentSeq !== searchSeq) return
    ElMessage.error(error?.message || t('globalSearch.searchFailed'))
  } finally {
    if (currentSeq === searchSeq) {
      loading.value = false
    }
  }
}

function getTypeLabel(item: ResourceSearchResult) {
  if (item.type === 'package') return t('globalSearch.directory')
  if (item.type === 'function') {
    if (item.template_type === 'table') return t('globalSearch.tableFunction')
    if (item.template_type === 'form') return t('globalSearch.formFunction')
    if (item.template_type === 'chart') return t('globalSearch.chartFunction')
    return t('globalSearch.function')
  }
  if (item.type === 'docs') return t('globalSearch.docs')
  return item.type
}

function shouldShowHeat(item: ResourceSearchResult) {
  return item.type === 'function'
}

function formatHeatCount(count: number) {
  if (count >= 10000) return `${(count / 10000).toFixed(count >= 100000 ? 0 : 1)}w`
  if (count >= 1000) return `${(count / 1000).toFixed(count >= 10000 ? 0 : 1)}k`
  return String(count)
}

function getAssetIcon(item: ResourceSearchResult): string | null {
  if (item.type === 'package') return '/service-tree/custom-folder.svg'
  if (item.type === 'docs') return '/文档.svg'
  if (item.type === 'function' && item.template_type === 'form') return '/service-tree/编辑.svg'
  return null
}

function getComponentIcon(item: ResourceSearchResult) {
  if (item.type === 'function' && item.template_type === 'table') return TableIcon
  if (item.type === 'function' && item.template_type === 'chart') return ChartIcon
  return Document
}

function getPathTail(path?: string) {
  const parts = String(path || '').split('/').filter(Boolean)
  return parts[parts.length - 1] || ''
}

function getResourceTitle(item: ResourceSearchResult) {
  return item.name?.trim()
    || item.code?.trim()
    || getPathTail(item.full_code_path)
    || t('globalSearch.unnamedResource')
}

function getResultSnippet(item: ResourceSearchResult) {
  return item.snippet?.trim()
    || item.description?.trim()
    || item.tags?.trim()
    || t('globalSearch.noDescription')
}

function normalizeDisplayText(text?: string) {
  return String(text || '').replace(/\s+/g, ' ').trim()
}

function truncateDisplayText(text: string, limit: number) {
  const normalized = normalizeDisplayText(text)
  if (normalized.length <= limit) return normalized
  return `${normalized.slice(0, limit)}...`
}

function getDisplayTitle(item: ResourceSearchResult) {
  return truncateDisplayText(getResourceTitle(item), 56)
}

function getDisplayPath(item: ResourceSearchResult) {
  return truncateDisplayText(item.full_code_path || t('globalSearch.missingPath'), 112)
}

function getDisplaySnippet(item: ResourceSearchResult) {
  return truncateDisplayText(getResultSnippet(item), 180)
}

async function handleSelect(item: ResourceSearchResult) {
  dialogVisible.value = false
  await router.push(resolveWorkspaceUrl(item.full_code_path))
}

watch([searchKeyword, activeType], scheduleSearch)

watch(dialogVisible, (visible) => {
  if (visible && searchKeyword.value.trim()) {
    scheduleSearch()
  }
})
</script>

<style scoped lang="scss">
:global(.global-resource-search-dialog) {
  --gs-ink: var(--el-text-color-primary);
  --gs-muted: var(--el-text-color-secondary);
  --gs-paper: var(--app-shell-panel-bg-strong, #f8fafc);
  --gs-tint: var(--app-shell-panel-muted-bg, #f1f5f9);
  --gs-line: var(--app-shell-panel-border, rgba(148, 163, 184, 0.15));
  --gs-accent: var(--el-color-primary);

  border-radius: 20px;
  position: relative;
  max-width: calc(100vw - 40px);
  overflow: hidden;
  border: 1px solid var(--gs-line);
  background: var(--el-bg-color, #fff);
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.16);
}

:global(.global-resource-search-dialog .el-dialog__header) {
  position: relative;
  z-index: 1;
  padding: 22px 26px 16px;
  border-bottom: 1px solid var(--gs-line);
  background: linear-gradient(90deg, color-mix(in srgb, var(--el-color-primary) 8%, transparent), transparent 70%);
}

:global(.global-resource-search-dialog .el-dialog__body) {
  position: relative;
  z-index: 1;
  padding: 20px 26px 24px;
}

:global(.global-resource-search-dialog .el-dialog__headerbtn .el-dialog__close) {
  color: var(--gs-muted);
}

.global-search-header {
  display: flex;
  align-items: center;
  gap: 16px;
}

.global-search-mark {
  width: 54px;
  height: 54px;
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.mark-core {
  width: 40px;
  height: 40px;
  border-radius: 13px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  background: linear-gradient(135deg, var(--el-color-primary), color-mix(in srgb, var(--el-color-primary) 55%, #a5b4fc));
  box-shadow: 0 8px 20px rgba(var(--el-color-primary-rgb), 0.3);
  z-index: 2;
}

.mark-ring {
  position: absolute;
  border: 1px solid rgba(var(--el-color-primary-rgb), 0.34);
  border-radius: 999px;
  animation: radar-pulse 2.6s ease-in-out infinite;
}

.mark-ring-a {
  inset: 6px;
}

.mark-ring-b {
  inset: 0;
  animation-delay: 0.55s;
  border-color: rgba(var(--el-color-primary-rgb), 0.18);
}

.global-search-copy {
  min-width: 0;
}

.global-search-kicker {
  color: var(--gs-accent);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.16em;
  margin-bottom: 5px;
}

.global-search-title {
  color: var(--gs-ink);
  font-size: 21px;
  font-weight: 800;
  letter-spacing: 0.01em;
}

.global-search-subtitle {
  margin-top: 2px;
  font-size: 13px;
  color: var(--gs-muted);
  line-height: 1.5;
}

.global-search-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.global-search-console {
  display: grid;
  grid-template-columns: 64px 1fr;
  align-items: center;
  gap: 10px;
  padding: 8px;
  border-radius: 14px;
  border: 1px solid var(--gs-line);
  background: var(--gs-tint);
}

.console-prefix {
  color: var(--gs-accent);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-align: center;
}

.global-search-input :deep(.el-input__wrapper) {
  min-height: 46px;
  border-radius: 12px;
  border: 1px solid var(--gs-line);
  background: var(--el-bg-color, #fff);
  box-shadow: none;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.global-search-input :deep(.el-input__wrapper.is-focus) {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px rgba(var(--el-color-primary-rgb), 0.14);
}

.global-search-input :deep(.el-input__inner) {
  color: var(--gs-ink);
}

.global-search-input :deep(.el-input__prefix),
.global-search-input :deep(.el-input__suffix) {
  color: var(--gs-accent);
}

.global-search-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.global-search-tab {
  border: 1px solid var(--gs-line);
  border-radius: 999px;
  background: var(--gs-paper);
  color: var(--gs-muted);
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  line-height: 1;
  padding: 8px 15px;
  transition: border-color 0.18s ease, background 0.18s ease, color 0.18s ease;

  &:hover {
    border-color: rgba(var(--el-color-primary-rgb), 0.32);
    color: var(--gs-accent);
  }

  &.active {
    border-color: rgba(var(--el-color-primary-rgb), 0.5);
    background: color-mix(in srgb, var(--el-color-primary) 10%, var(--gs-paper));
    color: var(--gs-accent);
  }
}

.global-search-result-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 20px;
  color: var(--gs-muted);
  font-size: 13px;
}

.meta-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: var(--el-color-success);
}

.global-search-results {
  min-height: 320px;
  max-height: 520px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding-right: 4px;
}

.global-search-results::-webkit-scrollbar {
  width: 8px;
}

.global-search-results::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(var(--el-color-primary-rgb), 0.22);
}

.global-search-result {
  display: grid;
  grid-template-columns: 52px minmax(0, 1fr) minmax(420px, 520px);
  align-items: stretch;
  gap: 16px;
  width: 100%;
  box-sizing: border-box;
  min-height: 104px;
  border: 1px solid var(--gs-line);
  border-radius: 14px;
  background: var(--gs-paper);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight, rgba(255, 255, 255, 0.7));
  cursor: pointer;
  padding: 16px;
  position: relative;
  text-align: left;
  transition: border-color 0.18s ease, background 0.18s ease, box-shadow 0.18s ease, transform 0.18s ease;
  overflow: hidden;

  &:hover {
    border-color: rgba(var(--el-color-primary-rgb), 0.32);
    background: color-mix(in srgb, var(--el-color-primary) 5%, var(--gs-paper));
    transform: translateY(-2px);
    box-shadow: 0 12px 30px rgba(15, 23, 42, 0.08);
  }
}

.global-search-result,
.global-search-result * {
  box-sizing: border-box;
}

.global-search-result > * {
  min-width: 0;
}

.result-icon {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 18%, transparent);
  background: color-mix(in srgb, var(--el-color-primary) 10%, var(--gs-paper));
  color: var(--gs-accent);
  position: relative;
  z-index: 1;
  align-self: center;

  &.type-function {
    border-color: color-mix(in srgb, var(--el-color-success) 22%, transparent);
    background: color-mix(in srgb, var(--el-color-success) 10%, var(--gs-paper));
    color: var(--el-color-success);
  }

  &.type-docs {
    border-color: color-mix(in srgb, #3b82f6 22%, transparent);
    background: color-mix(in srgb, #3b82f6 10%, var(--gs-paper));
    color: #2563eb;
  }

  .el-icon {
    font-size: 18px;
  }
}

.result-icon-img {
  width: 24px;
  height: 24px;
  object-fit: contain;
}

.result-main {
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  display: flex;
  flex: 1;
  flex-direction: column;
  justify-content: center;
  gap: 8px;
  position: relative;
  z-index: 1;
  padding-top: 2px;
}

.result-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
}

.result-title {
  display: block;
  flex: 1 1 auto;
  min-width: 0;
  color: var(--gs-ink);
  font-size: 15px;
  font-weight: 750;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  overflow-wrap: anywhere;
}

.result-type {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  background: color-mix(in srgb, var(--el-color-primary) 10%, var(--gs-paper));
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 20%, transparent);
  color: var(--gs-accent);
  flex-shrink: 0;
  font-size: 12px;
  padding: 3px 8px;
}

.result-heat {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  border-radius: 999px;
  border: 1px solid color-mix(in srgb, var(--el-color-warning) 30%, transparent);
  background: color-mix(in srgb, var(--el-color-warning) 12%, var(--gs-paper));
  color: #b45309;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1;
  padding: 4px 9px;
}

.result-path {
  color: var(--gs-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12.5px;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  overflow-wrap: anywhere;
}

.result-side {
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 8px;
  position: relative;
  z-index: 1;
  padding: 12px 14px;
  border-left: 1px solid var(--gs-line);
  border-radius: 12px;
  background: var(--gs-tint);
}

.side-description {
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.55;
  min-width: 0;
  max-width: 100%;
  max-height: 6.2em;
  overflow: hidden;
  text-overflow: ellipsis;
  word-break: break-word;
  overflow-wrap: anywhere;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 4;
}

.global-search-placeholder {
  min-height: 260px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 12px;
  color: var(--gs-muted);
  border: 1px dashed var(--gs-line);
  border-radius: 16px;
  background: color-mix(in srgb, var(--gs-paper) 72%, var(--gs-tint) 28%);
}

.placeholder-orbit {
  width: 54px;
  height: 54px;
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 30%, transparent);
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--gs-accent);
}

@keyframes radar-pulse {
  0%,
  100% {
    opacity: 0.5;
    transform: scale(0.96);
  }

  50% {
    opacity: 1;
    transform: scale(1.06);
  }
}

@media (max-width: 768px) {
  .global-search-header {
    align-items: flex-start;
  }

  .global-search-console {
    grid-template-columns: 1fr;
  }

  .console-prefix {
    text-align: left;
  }

  .global-search-results {
    max-height: 60vh;
  }

  .global-search-result {
    grid-template-columns: 56px minmax(0, 1fr);
    min-height: 150px;
  }

  .result-side {
    grid-column: 1 / -1;
    border-left: none;
    border-top: 1px solid rgba(54, 244, 255, 0.16);
  }

  .result-title-row {
    align-items: flex-start;
    flex-direction: column;
    gap: 4px;
  }
}
</style>
