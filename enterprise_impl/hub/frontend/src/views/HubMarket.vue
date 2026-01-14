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

    <!-- 搜索和筛选栏 -->
    <div class="filter-section">
      <div class="filter-content">
        <div class="search-bar">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索目录名称或描述..."
            clearable
            size="large"
            @clear="handleSearch"
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
            <template #append>
              <el-button @click="handleSearch" type="primary">搜索</el-button>
            </template>
          </el-input>
        </div>
        <div class="filter-controls">
          <el-select
            v-model="selectedCategory"
            placeholder="全部分类"
            clearable
            size="large"
            style="width: 200px"
            @change="handleFilterChange"
          >
            <el-option label="全部分类" value="" />
            <el-option label="工具" value="工具" />
            <el-option label="业务系统" value="业务系统" />
            <el-option label="数据管理" value="数据管理" />
            <el-option label="工作流" value="工作流" />
            <el-option label="报表" value="报表" />
          </el-select>
          <el-text type="info" size="large" style="margin-left: 16px">
            共 {{ total }} 个目录
          </el-text>
        </div>
      </div>
    </div>

    <!-- 目录列表 -->
    <div class="main-content">
      <div class="directory-content">
        <div v-loading="loading" class="directory-list">
          <div v-if="directories.length === 0 && !loading" class="empty-state">
            <el-empty description="暂无目录" :image-size="120">
              <el-button type="primary" @click="handleSearch">重新搜索</el-button>
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
                  <el-tag v-if="directory.category" type="info" size="small">
                    {{ directory.category }}
                  </el-tag>
                  <el-tag
                    v-if="directory.service_fee_personal > 0"
                    type="warning"
                    size="small"
                  >
                    ¥{{ directory.service_fee_personal }}
                  </el-tag>
                  <el-tag v-else type="success" size="small">免费</el-tag>
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
                  <div class="meta-item">
                    <UserDisplay 
                      :username="directory.publisher_username" 
                      layout="horizontal" 
                      size="small"
                    />
                  </div>
                  <div class="meta-item">
                    <el-icon class="meta-icon"><Download /></el-icon>
                    <el-text type="info" size="small">{{ directory.download_count }} 次克隆</el-text>
                  </div>
                  <div class="meta-item">
                    <el-icon class="meta-icon"><Files /></el-icon>
                    <el-text type="info" size="small">
                      {{ directory.directory_count }} 目录 · {{ directory.file_count }} 文件 · {{ directory.function_count }} 函数
                    </el-text>
                  </div>
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
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Search, User, Download, Files } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getHubDirectoryList, type HubDirectoryInfo } from '@/api/hub'
import UserDisplay from '@/components/UserDisplay.vue'
import { useUserInfoStore } from '@/stores/userInfo'

const router = useRouter()
const userInfoStore = useUserInfoStore()

// 搜索和筛选
const searchKeyword = ref('')
const selectedCategory = ref('')
const loading = ref(false)

// 分页
const currentPage = ref(1)
const pageSize = ref(24)
const total = ref(0)

// 目录列表
const directories = ref<HubDirectoryInfo[]>([])

// 加载目录列表
const loadDirectoryList = async () => {
  loading.value = true
  try {
    const response = await getHubDirectoryList({
      page: currentPage.value,
      page_size: pageSize.value,
      search: searchKeyword.value || undefined,
      category: selectedCategory.value || undefined
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
    ElMessage.error(`加载目录列表失败: ${error.message || '未知错误'}`)
    console.error('加载目录列表失败:', error)
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

// 点击目录卡片
const handleDirectoryClick = (directory: HubDirectoryInfo) => {
  router.push({
    name: 'hub-directory-detail',
    params: { id: directory.id }
  })
}

// 跳转到我的目录管理页面
const handleGoToManage = () => {
  router.push({ name: 'hub-directory-manage' })
}

// 监听路由变化，重新加载
watch(() => router.currentRoute.value.query, () => {
  const query = router.currentRoute.value.query
  if (query.search) {
    searchKeyword.value = query.search as string
  }
  if (query.category) {
    selectedCategory.value = query.category as string
  }
  loadDirectoryList()
}, { immediate: true })
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

  // 搜索和筛选区域
  .filter-section {
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
    padding: 24px 40px;

    .filter-content {
      max-width: 1400px;
      margin: 0 auto;
      display: flex;
      flex-direction: column;
      gap: 16px;

      .search-bar {
        flex: 1;
        max-width: 600px;
      }

      .filter-controls {
        display: flex;
        align-items: center;
      }
    }
  }

  // 主要内容区域
  .main-content {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;

    .directory-content {
      flex: 1;
      min-height: 0;
      overflow-y: auto;
      padding: 32px 40px;
      min-width: 0;
      width: 100%;
      max-width: 1400px;
      margin: 0 auto;

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
              gap: 8px;
              padding-top: 12px;
              border-top: 1px solid var(--el-border-color-lighter);
              margin-top: auto;

              .meta-item {
                display: flex;
                align-items: center;
                gap: 6px;

                .meta-icon {
                  font-size: 14px;
                  color: var(--el-text-color-secondary);
                }
                
                // UserDisplay 组件样式调整
                :deep(.user-display-wrapper) {
                  .user-name {
                    font-size: 13px;
                    color: var(--el-text-color-secondary);
                  }
                }
              }
            }
          }
        }
      }
    }
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

    .filter-section {
      padding: 20px;

      .filter-content {
        .filter-controls {
          flex-direction: column;
          align-items: stretch;
          gap: 12px;
        }
      }
    }

    .main-content {
      .directory-content {
        padding: 24px 20px;

        .directory-grid {
          grid-template-columns: 1fr;
        }
      }
    }

    .pagination-section {
      padding: 20px;
    }
  }
}
</style>
