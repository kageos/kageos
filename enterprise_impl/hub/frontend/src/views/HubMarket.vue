<!--
  HubMarket - Hub 应用中心市场页面
  
  参考目录详情页面的卡片样式
-->
<template>
  <div class="hub-market-view">
    <!-- 顶部横幅区域 -->
    <div class="hero-section">
      <div class="hero-content">
        <div class="hero-info">
          <div class="hero-icon-wrapper">
            <img
              src="/service-tree/custom-folder.svg"
              alt="应用中心"
              class="hero-icon-img"
            />
          </div>
          <div class="hero-text">
            <h1 class="hero-title">应用中心</h1>
            <p class="hero-subtitle">发现、分享、克隆优秀的目录和业务系统</p>
          </div>
        </div>
        <div class="hero-actions">
          <el-button type="primary" :icon="User" @click="handleGoToManage">
            我的目录
          </el-button>
        </div>
      </div>
    </div>

    <!-- 主体：左侧筛选 + 右侧列表 -->
    <div class="body-section">
      <!-- 左侧筛选栏 -->
      <aside class="filter-sidebar">
        <div class="filter-sidebar-header">
          <div class="filter-title-row">
            <el-icon class="filter-title-icon"><Filter /></el-icon>
            <h2 class="filter-title">应用筛选</h2>
          </div>
          <div class="filter-summary">
            <el-button link type="primary" class="clear-btn" :disabled="selectedFilterCount === 0" @click="handleClearFilters">
              清除
            </el-button>
            <span class="summary-text">已选 {{ selectedFilterCount }} 条件</span>
            <span class="summary-divider">|</span>
            <span class="summary-result">{{ total }} 结果</span>
          </div>
        </div>

        <div class="filter-block filter-block-search">
          <div class="filter-block-label">
            <el-icon><Search /></el-icon>
            <span>应用搜索</span>
          </div>
          <el-input
            v-model="searchKeyword"
            placeholder="搜索应用名称或描述"
            clearable
            size="default"
            class="filter-search-input"
            @clear="handleSearch"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-button type="primary" size="default" class="search-submit-btn" @click="handleSearch">搜索</el-button>
        </div>

        <div class="filter-block filter-block-sort">
          <div class="filter-block-label">
            <el-icon><Sort /></el-icon>
            <span>排序方式</span>
          </div>
          <div class="filter-options sort-options">
            <div
              v-for="opt in orderOptions"
              :key="opt.value"
              :class="['filter-option-item sort-option-item', { active: orderBy === opt.value }]"
              @click="orderBy = opt.value; handleFilterChange()"
            >
              <el-icon class="sort-option-icon"><component :is="opt.icon" /></el-icon>
              <span>{{ opt.label }}</span>
            </div>
          </div>
        </div>

        <div class="filter-block">
          <div class="filter-block-label">
            <el-icon><Money /></el-icon>
            <span>费用类型</span>
          </div>
          <div class="filter-options fee-options">
            <div
              v-for="opt in feeTypeOptions"
              :key="opt.value"
              :class="['filter-option-item', { active: feeTypeFilter === opt.value }]"
              @click="feeTypeFilter = opt.value; handleFilterChange()"
            >
              {{ opt.label }}
            </div>
          </div>
        </div>

        <div class="filter-block">
          <div class="filter-block-label">
            <el-icon><Folder /></el-icon>
            <span>所属分类</span>
          </div>
          <div class="filter-options category-options">
            <div
              :class="['filter-option-item', { active: selectedCategory === '' }]"
              @click="selectedCategory = ''; handleFilterChange()"
            >
              全部分类
            </div>
            <div
              v-for="cat in categoryOptions"
              :key="cat"
              :class="['filter-option-item', { active: selectedCategory === cat }]"
              @click="selectedCategory = cat; handleFilterChange()"
            >
              {{ cat }}
            </div>
          </div>
        </div>
      </aside>

      <!-- 右侧：列表 + 分页 -->
      <div class="right-area">
      <div class="main-content">
        <!-- Tab：全部 | 只看我的 -->
        <div class="list-scope-tabs">
          <el-radio-group v-model="listScope" size="default" @change="handleScopeChange">
            <el-radio-button label="all">全部</el-radio-button>
            <el-radio-button label="mine">只看我的</el-radio-button>
          </el-radio-group>
        </div>
        <div class="directory-content">
        <div v-loading="loading" class="directory-list">
          <div v-if="directories.length === 0 && !loading" class="empty-state">
            <el-empty
              :description="listScope === 'mine' ? '你还没有上传过应用，去「我的目录」发布吧' : '暂无目录'"
              :image-size="120"
            >
              <el-button v-if="listScope === 'mine'" type="primary" @click="handleGoToManage">去发布</el-button>
              <el-button v-else type="primary" @click="handleSearch">重新搜索</el-button>
            </el-empty>
          </div>

          <div v-else class="directory-grid">
            <div
              v-for="directory in directories"
              :key="directory.id"
              class="directory-card"
              @click="handleDirectoryClick(directory)"
            >
              <div class="directory-card-header">
                <div class="directory-icon-wrapper package-type">
                  <img
                    src="/service-tree/custom-folder.svg"
                    alt="目录"
                    class="directory-icon-img"
                  />
                </div>
                <div class="directory-header-badges">
                  <el-tag v-if="directory.category" type="info" size="small" class="badge-category">
                    {{ directory.category }}
                  </el-tag>
                  <el-tag
                    v-if="directory.service_fee_personal > 0"
                    type="warning"
                    size="small"
                    class="badge-fee paid"
                  >
                    ¥{{ directory.service_fee_personal }}
                  </el-tag>
                  <el-tag v-else type="success" size="small" class="badge-fee free">免费</el-tag>
                </div>
              </div>
              <div class="directory-card-body">
                <div class="directory-name">{{ directory.name }}</div>
                <div class="directory-description" v-if="directory.description">
                  <div
                    class="description-html"
                    v-html="directory.description"
                  />
                </div>
                <div class="directory-tags" v-if="directory.tags && directory.tags.length > 0">
                  <el-tag
                    v-for="tag in directory.tags"
                    :key="tag"
                    type="info"
                    size="small"
                    class="tag-item"
                  >
                    {{ tag }}
                  </el-tag>
                </div>
                <div class="directory-meta">
                  <div class="meta-value-row">
                    <div class="value-stat">
                      <el-icon class="value-stat-icon star"><Star /></el-icon>
                      <span class="value-stat-num">{{ directory.star_count ?? 0 }}</span>
                      <span class="value-stat-label">星</span>
                    </div>
                    <div class="value-stat">
                      <el-icon class="value-stat-icon copy"><CopyDocument /></el-icon>
                      <span class="value-stat-num">{{ directory.download_count ?? 0 }}</span>
                      <span class="value-stat-label">复制</span>
                    </div>
                  </div>
                  <div class="meta-item meta-publisher">
                    <UserDisplay 
                      :username="directory.publisher_username" 
                      layout="horizontal" 
                      size="small"
                    />
                  </div>
                </div>
                <div class="directory-card-actions" @click.stop>
                  <span class="actions-time" v-if="directory.published_at">
                    <el-icon class="meta-icon"><Clock /></el-icon>
                    发布时间 {{ formatDisplayTime(directory.published_at) }}
                  </span>
                  <el-button
                    link
                    type="primary"
                    size="small"
                    :icon="CopyDocument"
                    @click="handleCopyLink(directory)"
                  >
                    复制链接
                  </el-button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 分页 -->
    <div class="pagination-section" v-if="total > 0">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[12, 24, 48, 96]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handlePageSizeChange"
        @current-change="handlePageChange"
      />
    </div>
    </div>

    <!-- 右侧栏：热门 + 最新 -->
    <aside class="right-sidebar">
      <div class="sidebar-block">
        <div class="sidebar-block-head">
          <el-icon class="sidebar-block-icon hot"><Star /></el-icon>
          <h3 class="sidebar-title">热门应用</h3>
        </div>
        <div v-loading="hotLoading" class="sidebar-list">
          <div
            v-for="(item, index) in hotList"
            :key="item.id"
            class="sidebar-card"
            @click="handleDirectoryClick(item)"
          >
            <span class="sidebar-rank" :class="{ top: index < 3 }">{{ index + 1 }}</span>
            <div class="sidebar-card-body">
              <span class="sidebar-card-name">{{ item.name }}</span>
              <div class="sidebar-card-meta">
                <span class="hot-score">热度 {{ hotScore(item) }}</span>
                <span><el-icon><Star /></el-icon> {{ item.star_count ?? 0 }}</span>
                <span><el-icon><Download /></el-icon> {{ item.download_count ?? 0 }}</span>
              </div>
            </div>
          </div>
          <div v-if="!hotLoading && hotList.length === 0" class="sidebar-empty">
            <el-icon class="empty-icon"><Document /></el-icon>
            <span>暂无热门应用</span>
          </div>
        </div>
      </div>
      <div class="sidebar-block sidebar-block-latest">
        <div class="sidebar-block-head">
          <el-icon class="sidebar-block-icon latest"><Clock /></el-icon>
          <h3 class="sidebar-title">最新上架</h3>
        </div>
        <div v-loading="latestLoading" class="sidebar-list">
          <div
            v-for="item in latestList"
            :key="item.id"
            class="sidebar-card sidebar-card-latest"
            @click="handleDirectoryClick(item)"
          >
            <div class="sidebar-card-body">
              <div class="sidebar-card-headline">
                <span class="sidebar-card-name">{{ item.name }}</span>
                <span v-if="isNewPublish(item.published_at)" class="new-dot">新</span>
              </div>
              <div class="sidebar-card-meta">
                <span><el-icon><Star /></el-icon> {{ item.star_count ?? 0 }}</span>
                <span><el-icon><Download /></el-icon> {{ item.download_count ?? 0 }}</span>
              </div>
              <div class="sidebar-card-footer">
                <span class="sidebar-card-time" v-if="item.published_at">
                  <el-icon><Clock /></el-icon> {{ formatDisplayTime(item.published_at) }}
                </span>
                <span v-if="item.category" class="category-dot">{{ item.category }}</span>
              </div>
              <div v-if="item.publisher_username" class="sidebar-card-publisher">
                <UserDisplay :username="item.publisher_username" layout="horizontal" size="small" />
              </div>
            </div>
          </div>
          <div v-if="!latestLoading && latestList.length === 0" class="sidebar-empty">
            <el-icon class="empty-icon"><Document /></el-icon>
            <span>暂无最新上架</span>
          </div>
        </div>
      </div>
    </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Search, User, Download, CopyDocument, Star, Clock, Document, Filter, Money, Folder, Sort, TrendCharts } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getHubDirectoryList, type HubDirectoryInfo, type FeeTypeFilter, type OrderByFilter } from '@/api/hub'
import UserDisplay from '@/components/UserDisplay.vue'
import { useUserInfoStore } from '@/stores/userInfo'

const router = useRouter()
const userInfoStore = useUserInfoStore()

// Tab：全部 | 只看我的（后端用 mine_only 参数按当前用户过滤）
const listScope = ref<'all' | 'mine'>('all')

// 搜索和筛选
const searchKeyword = ref('')
const selectedCategory = ref('')
const feeTypeFilter = ref<FeeTypeFilter>('')
const orderBy = ref<OrderByFilter>('hot')
const loading = ref(false)

const orderOptions: { label: string; value: OrderByFilter; icon: typeof Clock }[] = [
  { label: '最新上架', value: 'latest', icon: Clock },
  { label: '热门', value: 'hot', icon: TrendCharts },
  { label: '按星', value: 'stars', icon: Star },
  { label: '按复制', value: 'downloads', icon: CopyDocument }
]

// 已选条件数量（用于左侧「已选 X 条件」）
const selectedFilterCount = computed(() => {
  let n = 0
  if (searchKeyword.value.trim()) n++
  if (feeTypeFilter.value) n++
  if (selectedCategory.value) n++
  return n
})

const feeTypeOptions = [
  { label: '全部', value: '' as FeeTypeFilter },
  { label: '免费', value: 'free' as FeeTypeFilter },
  { label: '收费', value: 'paid' as FeeTypeFilter }
]

const categoryOptions = [
  '表格', '表单', '表单、表格、图表', '企业服务', '数据分析', '视频处理',
  '工具', '业务系统', '数据管理', '工作流', '报表'
]

function handleClearFilters () {
  searchKeyword.value = ''
  selectedCategory.value = ''
  feeTypeFilter.value = ''
  currentPage.value = 1
  loadDirectoryList()
}

// 时间展示：优先友好格式，支持 ISO 或 "YYYY-MM-DD HH:mm:ss"
function formatDisplayTime (raw: string): string {
  if (!raw || typeof raw !== 'string') return ''
  const s = raw.trim().replace(/^"|"$/g, '')
  const date = new Date(s)
  if (Number.isNaN(date.getTime())) return s
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const day = 24 * 60 * 60 * 1000
  if (diff < day) return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  if (diff < 7 * day) return `${Math.floor(diff / day)} 天前`
  return date.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

// 是否「新上架」：发布时间在 3 天内则显示「新」标签
function isNewPublish (publishedAt: string | undefined): boolean {
  if (!publishedAt || typeof publishedAt !== 'string') return false
  const s = publishedAt.trim().replace(/^"|"$/g, '')
  const date = new Date(s)
  if (Number.isNaN(date.getTime())) return false
  const diff = Date.now() - date.getTime()
  return diff < 3 * 24 * 60 * 60 * 1000
}

// 热度值（与后端排序公式一致：星星*2 + 下载量*1）
function hotScore (item: HubDirectoryInfo): number {
  const star = item.star_count ?? 0
  const download = item.download_count ?? 0
  return star * 2 + download * 1
}

// 分页
const currentPage = ref(1)
const pageSize = ref(24)
const total = ref(0)

// 目录列表
const directories = ref<HubDirectoryInfo[]>([])

// 右侧栏：热门 / 最新
const hotList = ref<HubDirectoryInfo[]>([])
const latestList = ref<HubDirectoryInfo[]>([])
const hotLoading = ref(false)
const latestLoading = ref(false)

const loadHotList = async () => {
  hotLoading.value = true
  try {
    const res = await getHubDirectoryList({ page: 1, page_size: 8, order_by: 'hot' })
    hotList.value = res.items || []
  } catch {
    hotList.value = []
  } finally {
    hotLoading.value = false
  }
}

const loadLatestList = async () => {
  latestLoading.value = true
  try {
    const res = await getHubDirectoryList({ page: 1, page_size: 6 })
    latestList.value = res.items || []
    const publisherUsernames = (res.items || [])
      .map(dir => dir.publisher_username)
      .filter(Boolean) as string[]
    if (publisherUsernames.length > 0) {
      userInfoStore.batchGetUserInfo(publisherUsernames).catch(() => {})
    }
  } catch {
    latestList.value = []
  } finally {
    latestLoading.value = false
  }
}

// 加载目录列表（「只看我的」传 mine_only=true，后端按当前用户过滤；未登录时后端返回 401）
const loadDirectoryList = async () => {
  loading.value = true
  try {
    const response = await getHubDirectoryList({
      page: currentPage.value,
      page_size: pageSize.value,
      search: searchKeyword.value || undefined,
      category: selectedCategory.value || undefined,
      fee_type: feeTypeFilter.value || undefined,
      order_by: orderBy.value || 'latest',
      mine_only: listScope.value === 'mine'
    })

    directories.value = response.items || []
    total.value = response.total || 0
    
    // 🔥 预加载所有发布者的用户信息（批量获取，使用缓存）
    const publisherUsernames = directories.value
      .map(dir => dir.publisher_username)
      .filter(Boolean) as string[]
    
    if (publisherUsernames.length > 0) {
      // 批量获取用户信息（store 会自动处理缓存）
      userInfoStore.batchGetUserInfo(publisherUsernames).catch(error => {
        console.warn('[HubMarket] 预加载用户信息失败:', error)
      })
    }
  } catch (error: any) {
    const status = error?.response?.status
    if (status === 401 && listScope.value === 'mine') {
      ElMessage.warning('请先登录后再查看「只看我的」')
      directories.value = []
      total.value = 0
    } else {
      ElMessage.error(`加载目录列表失败: ${error.message || '未知错误'}`)
      console.error('加载目录列表失败:', error)
    }
  } finally {
    loading.value = false
  }
}

// 搜索
const handleSearch = () => {
  currentPage.value = 1
  loadDirectoryList()
}

// 筛选变化
const handleFilterChange = () => {
  currentPage.value = 1
  loadDirectoryList()
}

// Tab 切换：全部 | 只看我的
const handleScopeChange = () => {
  currentPage.value = 1
  loadDirectoryList()
}

// 分页变化
const handlePageChange = (page: number) => {
  currentPage.value = page
  loadDirectoryList()
}

const handlePageSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
  loadDirectoryList()
}

// 点击目录卡片（路由使用 path 参数：/directory/:path+，用 full_code_path 跳转）
const handleDirectoryClick = (directory: HubDirectoryInfo) => {
  const pathParam = directory.full_code_path?.startsWith('/')
    ? directory.full_code_path.slice(1)
    : (directory.full_code_path ?? '')
  if (!pathParam) {
    console.warn('[HubMarket] directory missing full_code_path', directory)
    return
  }
  router.push({
    name: 'hub-directory-detail',
    params: { path: pathParam }
  })
}

// 复制 Hub 链接（用于在工作空间粘贴安装）
const handleCopyLink = async (directory: HubDirectoryInfo) => {
  const copyUrl = directory.copy_url
  if (!copyUrl && !directory.full_code_path) {
    ElMessage.warning('复制链接不可用')
    return
  }
  const textToCopy = copyUrl || `hub://${window.location.host}${directory.full_code_path || ''}@${directory.version || ''}`
  if (!textToCopy || !textToCopy.startsWith('hub://')) {
    ElMessage.warning('复制链接不可用')
    return
  }
  try {
    await navigator.clipboard.writeText(textToCopy)
    ElMessage.success('Hub 链接已复制，可在工作空间「从应用中心安装」中粘贴使用')
  } catch {
    const ta = document.createElement('textarea')
    ta.value = textToCopy
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    try {
      document.execCommand('copy')
      ElMessage.success('Hub 链接已复制')
    } catch {
      ElMessage.error('复制失败，请手动复制')
    }
    document.body.removeChild(ta)
  }
}

// 跳转到我的目录管理页面
const handleGoToManage = () => {
  router.push({ name: 'hub-directory-manage' })
}

// 监听路由变化，重新加载
watch(() => router.currentRoute.value.query, () => {
  const query = router.currentRoute.value.query
  if (query.search) searchKeyword.value = query.search as string
  if (query.category) selectedCategory.value = query.category as string
  if (query.fee_type === 'free' || query.fee_type === 'paid') feeTypeFilter.value = query.fee_type as FeeTypeFilter
  loadDirectoryList()
}, { immediate: true })

onMounted(() => {
  loadHotList()
  loadLatestList()
})
</script>

<style scoped lang="scss">
.hub-market-view {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);

  // 顶部横幅区域
  .hero-section {
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
    padding: 32px 40px;

    .hero-content {
      max-width: 1400px;
      margin: 0 auto;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 24px;

      .hero-info {
        flex: 1;
        display: flex;
        align-items: center;
        gap: 20px;
        min-width: 0;

        .hero-icon-wrapper {
          flex-shrink: 0;
          display: flex;
          align-items: flex-start;
          justify-content: center;
          padding-top: 4px;

          .hero-icon-img {
            width: 48px;
            height: 48px;
            object-fit: contain;
          }
        }

        .hero-text {
          flex: 1;
          min-width: 0;

          .hero-title {
            margin: 0 0 8px 0;
            font-size: 28px;
            font-weight: 700;
            color: var(--el-text-color-primary);
            line-height: 1.2;
          }

          .hero-subtitle {
            margin: 0;
            font-size: 15px;
            color: var(--el-text-color-regular);
            line-height: 1.6;
          }
        }
      }

      .hero-actions {
        flex-shrink: 0;
      }
    }
  }

  // 主体：左侧筛选 + 右侧列表
  .body-section {
    flex: 1;
    min-height: 0;
    display: flex;
    width: 100%;
    max-width: 1800px;
    margin: 0 auto;
    align-self: stretch;
  }

  .filter-sidebar {
    width: 280px;
    flex-shrink: 0;
    background: linear-gradient(180deg, var(--el-bg-color) 0%, var(--el-fill-color-blank) 100%);
    border-right: 1px solid var(--el-border-color-lighter);
    padding: 0;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 0;

    .filter-sidebar-header {
      padding: 20px 18px 18px;
      border-bottom: 1px solid var(--el-border-color-lighter);
      background: var(--el-bg-color);

      .filter-title-row {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 12px;

        .filter-title-icon {
          font-size: 20px;
          color: var(--el-color-primary);
        }

        .filter-title {
          margin: 0;
          font-size: 16px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          letter-spacing: 0.02em;
        }
      }

      .filter-summary {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 8px;
        font-size: 12px;
        color: var(--el-text-color-secondary);
        padding: 10px 12px;
        background: var(--el-fill-color-light);
        border-radius: 10px;
        border: 1px solid var(--el-border-color-extra-light);

        .clear-btn {
          padding: 0 4px;
          font-size: 12px;
        }

        .summary-divider {
          color: var(--el-border-color);
          margin: 0 2px;
        }

        .summary-result {
          color: var(--el-text-color-primary);
          font-weight: 600;
        }
      }
    }

    .filter-block {
      padding: 16px 18px;
      border-bottom: 1px solid var(--el-border-color-extra-light);
      background: var(--el-bg-color);

      &:last-child {
        border-bottom: none;
      }

      .filter-block-label {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 13px;
        font-weight: 600;
        color: var(--el-text-color-primary);
        margin-bottom: 10px;

        .el-icon {
          font-size: 15px;
          color: var(--el-color-primary);
        }
      }

      &.filter-block-search {
        .filter-search-input {
          margin-bottom: 10px;
          border-radius: 8px;

          :deep(.el-input__wrapper) {
            border-radius: 8px;
          }
        }

        .search-submit-btn {
          width: 100%;
          border-radius: 8px;
        }
      }

      &.filter-block-sort {
        .sort-option-item {
          display: flex;
          align-items: center;
          gap: 10px;

          .sort-option-icon {
            font-size: 16px;
            color: var(--el-text-color-secondary);
            flex-shrink: 0;
          }

          &.active .sort-option-icon {
            color: var(--el-color-primary);
          }
        }
      }

      .filter-options {
        display: flex;
        flex-direction: column;
        gap: 6px;
      }

      .filter-option-item {
        padding: 10px 12px;
        border-radius: 10px;
        font-size: 13px;
        color: var(--el-text-color-regular);
        cursor: pointer;
        transition: background 0.2s, color 0.2s, border-color 0.2s;
        border: 1px solid transparent;

        &:hover {
          background: var(--el-fill-color-light);
          color: var(--el-text-color-primary);
        }

        &.active {
          background: color-mix(in srgb, var(--el-color-primary) 14%, transparent);
          color: var(--el-color-primary);
          font-weight: 500;
          border-color: color-mix(in srgb, var(--el-color-primary) 35%, transparent);
        }
      }

      .fee-options .filter-option-item.active {
        background: color-mix(in srgb, var(--el-color-primary) 14%, transparent);
        color: var(--el-color-primary);
        border-color: color-mix(in srgb, var(--el-color-primary) 35%, transparent);
      }

      .category-options .filter-option-item.active {
        background: color-mix(in srgb, var(--el-color-primary) 14%, transparent);
        color: var(--el-color-primary);
        border-color: color-mix(in srgb, var(--el-color-primary) 35%, transparent);
      }
    }
  }

  .right-area {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    background: var(--el-bg-color-page);
  }

  .right-sidebar {
    width: 280px;
    flex-shrink: 0;
    background: var(--el-bg-color);
    border-left: 1px solid var(--el-border-color-lighter);
    padding: 24px 0;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 28px;

    .sidebar-block {
      padding: 0 16px;

      .sidebar-block-head {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 14px;
        padding-bottom: 10px;
        border-bottom: 1px solid var(--el-border-color-lighter);

        .sidebar-block-icon {
          font-size: 18px;
          flex-shrink: 0;

          &.hot {
            color: var(--el-color-warning);
          }

          &.latest {
            color: var(--el-color-primary);
          }
        }

        .sidebar-title {
          margin: 0;
          font-size: 15px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          letter-spacing: 0.02em;
        }
      }

      .sidebar-list {
        min-height: 60px;
        display: flex;
        flex-direction: column;
        gap: 8px;
      }

      .sidebar-card {
        display: flex;
        align-items: flex-start;
        gap: 10px;
        padding: 12px 14px;
        border-radius: 10px;
        border: 1px solid transparent;
        cursor: pointer;
        transition: all 0.2s ease;
        background: var(--el-fill-color-blank);

        &:hover {
          background: var(--el-fill-color-light);
          border-color: var(--el-border-color-lighter);
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
        }

        .sidebar-rank {
          flex-shrink: 0;
          width: 22px;
          height: 22px;
          line-height: 22px;
          text-align: center;
          font-size: 12px;
          font-weight: 600;
          color: var(--el-text-color-secondary);
          background: var(--el-fill-color);
          border-radius: 6px;

          &.top {
            color: var(--el-color-white);
            background: linear-gradient(135deg, var(--el-color-warning), var(--el-color-warning-light-3));
          }
        }

        .sidebar-card-body {
          flex: 1;
          min-width: 0;
          display: flex;
          flex-direction: column;
          gap: 6px;
        }

        .sidebar-card-name {
          font-size: 14px;
          font-weight: 500;
          color: var(--el-text-color-primary);
          line-height: 1.4;
          display: -webkit-box;
          -webkit-line-clamp: 2;
          -webkit-box-orient: vertical;
          overflow: hidden;
        }

        .sidebar-card-meta {
          display: flex;
          align-items: center;
          gap: 12px;
          font-size: 12px;
          color: var(--el-text-color-secondary);

          .hot-score {
            font-weight: 600;
            color: var(--el-color-warning);
          }

          .el-icon {
            font-size: 12px;
            margin-right: 2px;
            vertical-align: -0.15em;
          }
        }

        .sidebar-card-time {
          font-size: 12px;
          color: var(--el-text-color-placeholder);
          display: flex;
          align-items: center;
          gap: 4px;

          .el-icon {
            font-size: 12px;
            flex-shrink: 0;
          }
        }
      }

      // 最新上架卡片：与热门同一套样式，无排名；多一行时间+分类，以及星/下载量
      .sidebar-card-latest {
        .sidebar-card-headline {
          display: flex;
          align-items: flex-start;
          justify-content: space-between;
          gap: 8px;

          .sidebar-card-name {
            flex: 1;
            min-width: 0;
          }

          .new-dot {
            flex-shrink: 0;
            font-size: 11px;
            color: var(--el-color-success);
            background: var(--el-color-success-light-8);
            padding: 2px 6px;
            border-radius: 4px;
            line-height: 1.2;
          }
        }

        .sidebar-card-meta {
          margin-top: 2px;
        }

        .sidebar-card-footer {
          display: flex;
          align-items: center;
          flex-wrap: wrap;
          gap: 8px;
          margin-top: 4px;

          .sidebar-card-time {
            font-size: 12px;
            color: var(--el-text-color-secondary);
          }

          .category-dot {
            font-size: 11px;
            color: var(--el-text-color-secondary);
            padding: 0 6px;
            line-height: 20px;
          }
        }

        .sidebar-card-publisher {
          margin-top: 8px;
          padding-top: 8px;
          border-top: 1px solid var(--el-border-color-extra-light);

          :deep(.user-display-wrapper) {
            .user-name {
              font-size: 12px;
              color: var(--el-text-color-secondary);
            }
          }
        }
      }
    }

    .sidebar-block-latest .sidebar-list {
      .sidebar-card-body {
        gap: 6px;
      }
    }

    .sidebar-block:not(.sidebar-block-latest) .sidebar-card-body {
      .sidebar-card-meta {
        margin-top: 0;
      }
    }

    .sidebar-block {
      .sidebar-empty {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 8px;
        padding: 24px 16px;
        font-size: 13px;
        color: var(--el-text-color-placeholder);

        .empty-icon {
          font-size: 32px;
          opacity: 0.6;
        }
      }
    }
  }

  // 右侧列表区域
  .main-content {
    flex: 1;
    min-height: 0;
    min-width: 0;
    display: flex;
    flex-direction: column;
    background: var(--el-bg-color-page);

    .list-scope-tabs {
      flex-shrink: 0;
      padding: 16px 32px 0;
      margin-bottom: 8px;

      :deep(.el-radio-group) {
        .el-radio-button__inner {
          border-radius: 8px;
        }
        .el-radio-button:first-child .el-radio-button__inner {
          border-radius: 8px 0 0 8px;
        }
        .el-radio-button:last-child .el-radio-button__inner {
          border-radius: 0 8px 8px 0;
        }
      }
    }

    .directory-content {
      flex: 1;
      min-height: 0;
      overflow-y: auto;
      padding: 24px 32px;
      min-width: 0;
      width: 100%;

      .directory-list {
        min-height: 400px;
      }

      .empty-state {
        margin-top: 60px;
      }

      .directory-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
        gap: 24px;
        width: 100%;

        .directory-card {
          background: var(--el-bg-color);
          border: 1px solid var(--el-border-color-lighter);
          border-radius: 12px;
          padding: 20px;
          transition: all 0.3s ease;
          cursor: pointer;
          width: 100%;
          box-sizing: border-box;
          display: flex;
          flex-direction: column;
          gap: 16px;

          &:hover {
            border-color: var(--el-color-primary-light-7);
            box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
            transform: translateY(-2px);
          }

          .directory-card-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 8px;

            .directory-icon-wrapper {
              display: flex;
              align-items: center;
              justify-content: center;
              width: 48px;
              height: 48px;
              border-radius: 12px;
              flex-shrink: 0;

              &.package-type {
                background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));

                .directory-icon-img {
                  width: 32px;
                  height: 32px;
                  object-fit: contain;
                }
              }
            }

            .directory-header-badges {
              display: flex;
              gap: 8px;
              flex-wrap: wrap;
              justify-content: flex-end;
              align-items: center;

              .badge-category {
                font-size: 12px;
              }

              .badge-fee {
                font-weight: 600;
                &.free {
                  background: var(--el-color-success-light-9);
                  color: var(--el-color-success);
                  border-color: var(--el-color-success-light-5);
                }
                &.paid {
                  background: var(--el-color-warning-light-9);
                  color: var(--el-color-warning-dark-2);
                  border-color: var(--el-color-warning-light-5);
                }
              }
            }
          }

          .directory-card-body {
            flex: 1;
            display: flex;
            flex-direction: column;
            gap: 12px;

            .directory-name {
              font-size: 18px;
              font-weight: 600;
              color: var(--el-text-color-primary);
              line-height: 1.5;
              word-break: break-word;
            }

            .directory-description {
              font-size: 13px;
              color: var(--el-text-color-secondary);
              line-height: 1.6;
              word-break: break-word;
              display: -webkit-box;
              -webkit-line-clamp: 3;
              -webkit-box-orient: vertical;
              overflow: hidden;
              text-overflow: ellipsis;

              .description-html {
                :deep(p) {
                  margin: 0;
                  &:not(:last-child) {
                    margin-bottom: 8px;
                  }
                }
              }
            }

            .directory-tags {
              display: flex;
              flex-wrap: wrap;
              gap: 8px;
              min-height: 24px;

              .tag-item {
                font-size: 12px;
                padding: 4px 8px;
              }
            }

            .directory-meta {
              display: flex;
              flex-direction: column;
              gap: 10px;
              padding-top: 12px;
              border-top: 1px solid var(--el-border-color-lighter);
              margin-top: auto;

              .meta-value-row {
                display: flex;
                align-items: center;
                gap: 16px;
                padding: 10px 12px;
                background: var(--el-fill-color-light);
                border-radius: 8px;
              }

              .value-stat {
                display: flex;
                align-items: center;
                gap: 6px;

                .value-stat-icon {
                  font-size: 18px;
                  flex-shrink: 0;

                  &.star {
                    color: var(--el-color-warning);
                  }

                  &.copy {
                    color: var(--el-color-primary);
                  }
                }

                .value-stat-num {
                  font-size: 18px;
                  font-weight: 700;
                  color: var(--el-text-color-primary);
                  line-height: 1.2;
                }

                .value-stat-label {
                  font-size: 13px;
                  color: var(--el-text-color-secondary);
                }
              }

              .meta-row {
                display: flex;
                align-items: center;
                gap: 6px;
                flex-wrap: wrap;

                .meta-icon {
                  font-size: 14px;
                  color: var(--el-text-color-placeholder);
                  flex-shrink: 0;
                }
              }

              .meta-item {
                display: flex;
                align-items: center;
                gap: 6px;
              }

              .meta-publisher {
                :deep(.user-display-wrapper) {
                  .user-name {
                    font-size: 13px;
                    color: var(--el-text-color-secondary);
                  }
                }
              }
            }

            .directory-card-actions {
              margin-top: 8px;
              padding-top: 8px;
              border-top: 1px solid var(--el-border-color-lighter);
              display: flex;
              align-items: center;
              justify-content: space-between;
              gap: 8px;

              .actions-time {
                display: inline-flex;
                align-items: center;
                gap: 4px;
                font-size: 12px;
                color: var(--el-text-color-placeholder);

                .meta-icon {
                  font-size: 14px;
                }
              }
            }
          }
        }
      }
    }
  }

  .pagination-section {
    flex-shrink: 0;
  }

  // 分页区域
  .pagination-section {
    background: var(--el-bg-color);
    border-top: 1px solid var(--el-border-color-lighter);
    padding: 24px 40px;
    display: flex;
    justify-content: center;
  }
}

// 响应式设计
@media (max-width: 1200px) {
  .hub-market-view .body-section .right-sidebar {
    display: none;
  }
}

@media (max-width: 1024px) {
  .hub-market-view .body-section {
    flex-direction: column;

    .filter-sidebar {
      width: 100%;
      border-right: none;
      border-bottom: 1px solid var(--el-border-color-lighter);
      flex-direction: row;
      flex-wrap: wrap;
      gap: 16px;
      padding: 16px 20px;

      .filter-block {
        min-width: 160px;
      }

      .filter-block:first-of-type {
        flex: 1;
        min-width: 200px;
      }
    }
  }
}

@media (max-width: 768px) {
  .hub-market-view {
    .hero-section {
      padding: 24px 20px;

      .hero-content {
        flex-direction: column;
        align-items: stretch;
        gap: 16px;

        .hero-info {
          flex-direction: column;
          align-items: flex-start;
          gap: 16px;
        }
      }
    }

    .body-section .filter-sidebar {
      .filter-block-label {
        margin-bottom: 6px;
      }

      .filter-options .filter-option-item {
        padding: 6px 10px;
        font-size: 13px;
      }
    }

    .main-content .directory-content {
      padding: 24px 20px;

      .directory-grid {
        grid-template-columns: 1fr;
      }
    }

    .pagination-section {
      padding: 20px;
    }
  }
}
</style>
