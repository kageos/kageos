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
          <div class="global-search-kicker">RESOURCE RADAR</div>
          <div class="global-search-title">全站资源搜索</div>
          <div class="global-search-subtitle">搜索目录、函数和文档，图标与目录树保持一致</div>
        </div>
      </div>
    </template>

    <div class="global-search-body">
      <div class="global-search-console">
        <div class="console-prefix">QUERY</div>
        <el-input
          ref="searchInputRef"
          v-model="searchKeyword"
          size="large"
          class="global-search-input"
          placeholder="输入关键字，例如：客户、报表、审批、SDK..."
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
        <span v-if="hasSearched && !loading">命中 {{ total }} 个资源节点</span>
        <span v-else-if="loading">扫描资源索引中...</span>
        <span v-else>等待输入关键字，自动扫描可见资源</span>
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
                :title="`调用量：${item.run_count || 0}`"
              >
                热度 {{ formatHeatCount(item.run_count || 0) }}
              </span>
            </span>
            <span class="result-path" :title="item.full_code_path || '路径缺失'">
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
          description="没有匹配的资源"
          :image-size="96"
        />

        <div v-if="!loading && !hasSearched" class="global-search-placeholder">
          <span class="placeholder-orbit">
            <el-icon><Search /></el-icon>
          </span>
          <span>输入关键字，快速跳转到全站资源</span>
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
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

const resourceTabs: Array<{ label: string; value: SearchResourceType }> = [
  { label: '全部', value: 'all' },
  { label: '目录', value: 'package' },
  { label: '函数', value: 'function' },
  { label: '文档', value: 'docs' }
]

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
    ElMessage.error(error?.message || '搜索失败')
  } finally {
    if (currentSeq === searchSeq) {
      loading.value = false
    }
  }
}

function getTypeLabel(item: ResourceSearchResult) {
  if (item.type === 'package') return '目录'
  if (item.type === 'function') {
    if (item.template_type === 'table') return '表格函数'
    if (item.template_type === 'form') return '表单函数'
    if (item.template_type === 'chart') return '图表函数'
    return '函数'
  }
  if (item.type === 'docs') return '文档'
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
    || '未命名资源'
}

function getResultSnippet(item: ResourceSearchResult) {
  return item.snippet?.trim()
    || item.description?.trim()
    || item.tags?.trim()
    || '暂无描述，点击打开查看资源详情'
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
  return truncateDisplayText(item.full_code_path || '路径缺失', 112)
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
  --radar-bg: #06131f;
  --radar-panel: rgba(8, 23, 38, 0.94);
  --radar-panel-soft: rgba(13, 42, 62, 0.72);
  --radar-cyan: #36f4ff;
  --radar-cyan-dim: rgba(54, 244, 255, 0.28);
  --radar-green: #7cffc4;
  --radar-amber: #ffd166;
  --radar-text: #d9fbff;
  --radar-muted: #81a8b8;

  border-radius: 24px;
  position: relative;
  max-width: calc(100vw - 40px);
  overflow: hidden;
  border: 1px solid var(--radar-cyan-dim);
  background:
    radial-gradient(circle at 12% 8%, rgba(54, 244, 255, 0.2), transparent 28%),
    radial-gradient(circle at 86% 14%, rgba(124, 255, 196, 0.12), transparent 28%),
    linear-gradient(145deg, rgba(3, 10, 18, 0.98), var(--radar-bg));
  box-shadow:
    0 0 0 1px rgba(54, 244, 255, 0.12),
    0 22px 70px rgba(0, 0, 0, 0.48),
    0 0 42px rgba(54, 244, 255, 0.16);
}

:global(.global-resource-search-dialog::before) {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background:
    linear-gradient(rgba(54, 244, 255, 0.055) 1px, transparent 1px),
    linear-gradient(90deg, rgba(54, 244, 255, 0.045) 1px, transparent 1px);
  background-size: 24px 24px;
  mask-image: linear-gradient(180deg, rgba(0, 0, 0, 0.7), transparent 80%);
}

:global(.global-resource-search-dialog .el-dialog__header) {
  position: relative;
  z-index: 1;
  padding: 24px 28px 18px;
  border-bottom: 1px solid rgba(54, 244, 255, 0.16);
  background: linear-gradient(90deg, rgba(54, 244, 255, 0.1), rgba(124, 255, 196, 0.03));
}

:global(.global-resource-search-dialog .el-dialog__body) {
  position: relative;
  z-index: 1;
  padding: 20px 28px 26px;
}

:global(.global-resource-search-dialog .el-dialog__headerbtn .el-dialog__close) {
  color: var(--radar-cyan);
  filter: drop-shadow(0 0 8px rgba(54, 244, 255, 0.5));
}

.global-search-header {
  display: flex;
  align-items: center;
  gap: 18px;
}

.global-search-mark {
  width: 58px;
  height: 58px;
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.mark-core {
  width: 38px;
  height: 38px;
  border-radius: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--radar-bg);
  background: linear-gradient(135deg, var(--radar-cyan), var(--radar-green));
  box-shadow: 0 0 24px rgba(54, 244, 255, 0.48);
  z-index: 2;
}

.mark-ring {
  position: absolute;
  border: 1px solid rgba(54, 244, 255, 0.38);
  border-radius: 999px;
  animation: radar-pulse 2.6s ease-in-out infinite;
}

.mark-ring-a {
  inset: 8px;
}

.mark-ring-b {
  inset: 0;
  animation-delay: 0.55s;
  border-color: rgba(124, 255, 196, 0.24);
}

.global-search-copy {
  min-width: 0;
}

.global-search-kicker {
  color: var(--radar-green);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
  letter-spacing: 0.18em;
  margin-bottom: 5px;
  text-shadow: 0 0 14px rgba(124, 255, 196, 0.42);
}

.global-search-title {
  color: var(--radar-text);
  font-size: 22px;
  font-weight: 800;
  letter-spacing: 0.03em;
  text-shadow: 0 0 18px rgba(54, 244, 255, 0.35);
}

.global-search-subtitle {
  font-size: 13px;
  color: var(--radar-muted);
  line-height: 1.5;
}

.global-search-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.global-search-console {
  display: grid;
  grid-template-columns: 72px 1fr;
  align-items: center;
  gap: 10px;
  padding: 10px;
  border-radius: 18px;
  border: 1px solid rgba(54, 244, 255, 0.22);
  background: linear-gradient(90deg, rgba(54, 244, 255, 0.08), rgba(5, 18, 29, 0.62));
  box-shadow: inset 0 0 22px rgba(54, 244, 255, 0.06);
}

.console-prefix {
  color: var(--radar-green);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  letter-spacing: 0.12em;
  text-align: center;
}

.global-search-input :deep(.el-input__wrapper) {
  min-height: 46px;
  border-radius: 14px;
  border: 1px solid rgba(54, 244, 255, 0.22);
  background: rgba(4, 12, 20, 0.78);
  box-shadow: inset 0 0 18px rgba(54, 244, 255, 0.08), 0 0 0 1px rgba(54, 244, 255, 0.04);
}

.global-search-input :deep(.el-input__inner) {
  color: var(--radar-text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.global-search-input :deep(.el-input__prefix),
.global-search-input :deep(.el-input__suffix) {
  color: var(--radar-cyan);
}

.global-search-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.global-search-tab {
  border: 1px solid rgba(54, 244, 255, 0.18);
  border-radius: 999px;
  background: rgba(7, 24, 38, 0.78);
  color: var(--radar-muted);
  cursor: pointer;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
  line-height: 1;
  padding: 9px 15px;
  transition: all 0.18s ease;

  &:hover,
  &.active {
    border-color: rgba(54, 244, 255, 0.7);
    background: rgba(54, 244, 255, 0.12);
    color: var(--radar-cyan);
    box-shadow: 0 0 18px rgba(54, 244, 255, 0.16);
  }
}

.global-search-result-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 20px;
  color: var(--radar-muted);
  font-size: 13px;
}

.meta-dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  background: var(--radar-green);
  box-shadow: 0 0 12px rgba(124, 255, 196, 0.8);
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
  background: rgba(54, 244, 255, 0.22);
}

.global-search-result {
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr) minmax(420px, 520px);
  align-items: stretch;
  gap: 18px;
  width: 100%;
  box-sizing: border-box;
  min-height: 112px;
  border: 1px solid rgba(54, 244, 255, 0.16);
  border-radius: 18px;
  background:
    linear-gradient(90deg, rgba(54, 244, 255, 0.09), transparent 42%),
    rgba(7, 20, 32, 0.78);
  cursor: pointer;
  padding: 18px;
  position: relative;
  text-align: left;
  transition: all 0.18s ease;
  overflow: hidden;

  &::after {
    content: '';
    position: absolute;
    inset: 0;
    transform: translateX(-120%);
    background: linear-gradient(90deg, transparent, rgba(54, 244, 255, 0.14), transparent);
    transition: transform 0.42s ease;
  }

  &:hover {
    border-color: rgba(54, 244, 255, 0.58);
    background:
      linear-gradient(90deg, rgba(54, 244, 255, 0.16), rgba(124, 255, 196, 0.07)),
      rgba(8, 28, 44, 0.92);
    transform: translateY(-2px);
    box-shadow: 0 12px 30px rgba(0, 0, 0, 0.18), 0 0 24px rgba(54, 244, 255, 0.12);
  }

  &:hover::after {
    transform: translateX(120%);
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
  width: 56px;
  height: 56px;
  border-radius: 18px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border: 1px solid rgba(54, 244, 255, 0.18);
  background: rgba(54, 244, 255, 0.08);
  color: var(--radar-cyan);
  box-shadow: inset 0 0 18px rgba(54, 244, 255, 0.08);
  position: relative;
  z-index: 1;
  align-self: center;

  &.type-function {
    background: rgba(16, 185, 129, 0.12);
    color: #36f4a6;
  }

  &.type-docs {
    background: rgba(59, 130, 246, 0.13);
    color: #72a8ff;
  }

  .el-icon {
    font-size: 18px;
  }
}

.result-icon-img {
  width: 26px;
  height: 26px;
  object-fit: contain;
  filter: drop-shadow(0 0 8px rgba(54, 244, 255, 0.24));
}

.result-icon.type-docs .result-icon-img {
  width: 24px;
  height: 24px;
}

.result-main {
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  display: flex;
  flex: 1;
  flex-direction: column;
  justify-content: center;
  gap: 10px;
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
  color: var(--radar-text);
  font-size: 16px;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  overflow-wrap: anywhere;
}

.result-type {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  background: rgba(54, 244, 255, 0.1);
  border: 1px solid rgba(54, 244, 255, 0.16);
  color: var(--radar-cyan);
  flex-shrink: 0;
  font-size: 12px;
  padding: 3px 8px;
}

.result-heat {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  border-radius: 999px;
  border: 1px solid rgba(255, 209, 102, 0.22);
  background:
    linear-gradient(90deg, rgba(255, 209, 102, 0.18), rgba(255, 107, 53, 0.12));
  color: #ffd98a;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1;
  padding: 4px 9px;
  box-shadow: 0 0 14px rgba(255, 209, 102, 0.12);
}

.result-path {
  color: var(--radar-muted);
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
  border-left: 1px solid rgba(54, 244, 255, 0.16);
  border-radius: 14px;
  background: rgba(2, 12, 21, 0.36);
  box-shadow: inset 0 0 18px rgba(54, 244, 255, 0.04);
}

.side-description {
  color: rgba(217, 251, 255, 0.76);
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
  color: var(--radar-muted);
  border: 1px dashed rgba(54, 244, 255, 0.18);
  border-radius: 20px;
  background: rgba(5, 18, 29, 0.42);
}

.placeholder-orbit {
  width: 54px;
  height: 54px;
  border: 1px solid rgba(54, 244, 255, 0.34);
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--radar-cyan);
  box-shadow: 0 0 24px rgba(54, 244, 255, 0.16);
}

@keyframes radar-pulse {
  0%,
  100% {
    opacity: 0.48;
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
