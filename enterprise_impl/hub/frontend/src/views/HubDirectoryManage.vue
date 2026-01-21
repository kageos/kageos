<template>
  <div class="hub-directory-manage" v-loading="loading">
    <!-- 顶部导航栏 -->
    <div class="hub-header">
      <div class="header-left">
        <el-button :icon="ArrowLeft" @click="handleBack" text>返回应用中心</el-button>
        <h1 class="logo">我的目录</h1>
      </div>
      <div class="header-right">
        <ThemeToggle />
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="manage-container">
      <!-- 搜索和筛选 -->
      <div class="filter-section">
        <el-input
          v-model="searchKeyword"
          placeholder="搜索目录名称或描述..."
          clearable
          style="width: 300px"
          @clear="handleSearch"
          @keyup.enter="handleSearch"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-select
          v-model="selectedCategory"
          placeholder="全部分类"
          clearable
          style="width: 150px; margin-left: 16px"
          @change="handleFilterChange"
        >
          <el-option label="全部分类" value="" />
          <el-option label="工具" value="工具" />
          <el-option label="业务系统" value="业务系统" />
          <el-option label="数据管理" value="数据管理" />
          <el-option label="工作流" value="工作流" />
          <el-option label="报表" value="报表" />
        </el-select>
      </div>

      <!-- 目录列表 -->
      <div class="directories-list">
        <el-table :data="directories" style="width: 100%" border>
          <el-table-column prop="name" label="目录名称" min-width="200">
            <template #default="{ row }">
              <el-link type="primary" @click="handleViewDetail(row.id)">
                {{ row.name }}
              </el-link>
            </template>
          </el-table-column>
          <el-table-column prop="category" label="分类" width="120" />
          <el-table-column prop="directory_count" label="子目录数" width="100" align="center" />
          <el-table-column prop="file_count" label="文件数" width="100" align="center" />
          <el-table-column prop="download_count" label="克隆次数" width="100" align="center" />
          <el-table-column prop="published_at" label="发布时间" width="180" />
          <el-table-column label="操作" width="120" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link @click="handleViewDetail(row.id)">
                查看
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- 分页 -->
      <div class="pagination" v-if="total > 0">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handlePageSizeChange"
          @current-change="handlePageChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getHubDirectoryList, type HubDirectoryInfo } from '@/api/hub'
import ThemeToggle from '@/components/ThemeToggle.vue'

const router = useRouter()

// 搜索和筛选
const searchKeyword = ref('')
const selectedCategory = ref('')
const loading = ref(false)

// 分页
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 目录列表
const directories = ref<HubDirectoryInfo[]>([])

// 获取当前用户（从 token 中解析，这里简化处理）
const getCurrentUsername = () => {
  // TODO: 从 token 或 API 获取当前用户名
  return ''
}

// 加载目录列表
const loadDirectoryList = async () => {
  loading.value = true
  try {
    const username = getCurrentUsername()
    const response = await getHubDirectoryList({
      page: currentPage.value,
      page_size: pageSize.value,
      search: searchKeyword.value || undefined,
      category: selectedCategory.value || undefined,
      publisher_username: username || undefined
    })

    directories.value = response.items || []
    total.value = response.total || 0
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

// 查看详情
const handleViewDetail = (directoryId: number) => {
  router.push({
    name: 'hub-directory-detail',
    params: { id: directoryId }
  })
}

// 返回
const handleBack = () => {
  router.push({ name: 'hub-market' })
}

onMounted(() => {
  loadDirectoryList()
})
</script>

<style scoped>
.hub-directory-manage {
  min-height: 100vh;
  background: var(--bg-page);
}

.hub-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background: var(--bg-card);
  border-bottom: 1px solid var(--border-color);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.logo {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}

.manage-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
}

.filter-section {
  display: flex;
  margin-bottom: 24px;
}

.directories-list {
  margin-bottom: 24px;
}

.pagination {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}
</style>

