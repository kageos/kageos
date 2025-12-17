<!--
  DirectoryUpdateHistoryDialog - 变更记录对话框组件
  
  职责：
  - 显示工作空间变更记录（App视角）或目录变更记录（目录视角）
  - 封装为可复用组件，避免代码重复
-->
<template>
  <el-dialog
    v-model="visible"
    :title="mode === 'app' ? '工作空间变更记录' : '目录变更记录'"
    width="80%"
    :close-on-click-modal="false"
    class="directory-update-history-dialog"
  >
    <div v-loading="loading" class="history-content">
      <!-- App视角：显示所有版本的变更 -->
      <template v-if="mode === 'app' && appHistory">
        <div v-if="appHistory.versions.length === 0" class="empty-state">
          <el-empty description="暂无变更记录" />
        </div>
        
        <div v-else class="versions-list">
          <div
            v-for="version in appHistory.versions"
            :key="version.app_version"
            class="version-item"
          >
            <div class="version-header">
              <el-tag type="primary" size="large">{{ version.app_version }}</el-tag>
              <span class="version-info">
                共 {{ version.directory_changes.length }} 个目录变更
              </span>
            </div>
            
            <div class="directory-changes">
              <div
                v-for="change in version.directory_changes"
                :key="`${change.full_code_path}-${change.dir_version}`"
                class="change-item"
              >
                <div class="change-header">
                  <el-link
                    type="primary"
                    :underline="false"
                    @click="handleViewDirectory(change.full_code_path)"
                  >
                    {{ change.full_code_path }}
                  </el-link>
                  <el-tag size="small" type="info">v{{ change.dir_version_num }}</el-tag>
                  <span class="change-summary" v-if="change.summary">{{ change.summary }}</span>
                </div>
                
                <div class="change-stats">
                  <el-tag v-if="change.added_count > 0" type="success" size="small">
                    +{{ change.added_count }} 新增
                  </el-tag>
                  <el-tag v-if="change.updated_count > 0" type="warning" size="small">
                    ~{{ change.updated_count }} 更新
                  </el-tag>
                  <el-tag v-if="change.deleted_count > 0" type="danger" size="small">
                    -{{ change.deleted_count }} 删除
                  </el-tag>
                  <span class="change-time">
                    {{ formatTime(change.created_at) }} · {{ change.updated_by }}
                  </span>
                </div>
                
                <!-- API 变更详情 -->
                <el-collapse v-if="hasApiChanges(change)" class="api-changes">
                  <el-collapse-item
                    v-if="getApiList(change.added_apis).length > 0"
                    title="新增的 API"
                    :name="`added-${change.full_code_path}`"
                  >
                    <div class="api-list">
                      <div
                        v-for="api in getApiList(change.added_apis)"
                        :key="api.code"
                        class="api-item added"
                      >
                        <el-tag type="success" size="small">{{ api.method }}</el-tag>
                        <span class="api-name">{{ api.name }}</span>
                        <span class="api-desc" v-if="api.desc">{{ api.desc }}</span>
                        <span class="api-router">{{ api.router }}</span>
                      </div>
                    </div>
                  </el-collapse-item>
                  
                  <el-collapse-item
                    v-if="getApiList(change.updated_apis).length > 0"
                    title="更新的 API"
                    :name="`updated-${change.full_code_path}`"
                  >
                    <div class="api-list">
                      <div
                        v-for="api in getApiList(change.updated_apis)"
                        :key="api.code"
                        class="api-item updated"
                      >
                        <el-tag type="warning" size="small">{{ api.method }}</el-tag>
                        <span class="api-name">{{ api.name }}</span>
                        <span class="api-desc" v-if="api.desc">{{ api.desc }}</span>
                        <span class="api-router">{{ api.router }}</span>
                      </div>
                    </div>
                  </el-collapse-item>
                  
                  <el-collapse-item
                    v-if="getApiList(change.deleted_apis).length > 0"
                    title="删除的 API"
                    :name="`deleted-${change.full_code_path}`"
                  >
                    <div class="api-list">
                      <div
                        v-for="api in getApiList(change.deleted_apis)"
                        :key="api.code"
                        class="api-item deleted"
                      >
                        <el-tag type="danger" size="small">{{ api.method }}</el-tag>
                        <span class="api-name">{{ api.name }}</span>
                        <span class="api-desc" v-if="api.desc">{{ api.desc }}</span>
                        <span class="api-router">{{ api.router }}</span>
                      </div>
                    </div>
                  </el-collapse-item>
                </el-collapse>
              </div>
            </div>
          </div>
        </div>
      </template>
      
      <!-- 目录视角：显示单个目录的变更历史 -->
      <template v-else-if="mode === 'directory' && directoryHistory">
        <div v-if="directoryHistory.directory_changes.length === 0" class="empty-state">
          <el-empty description="暂无变更记录" />
        </div>
        
        <div v-else>
          <div
            v-for="change in directoryHistory.directory_changes"
            :key="`${change.dir_version}`"
            class="change-item"
          >
            <div class="change-header">
              <el-tag type="primary" size="large">v{{ change.dir_version_num }}</el-tag>
              <span class="change-summary" v-if="change.summary">{{ change.summary }}</span>
            </div>
            
            <div class="change-stats">
              <el-tag v-if="change.added_count > 0" type="success" size="small">
                +{{ change.added_count }} 新增
              </el-tag>
              <el-tag v-if="change.updated_count > 0" type="warning" size="small">
                ~{{ change.updated_count }} 更新
              </el-tag>
              <el-tag v-if="change.deleted_count > 0" type="danger" size="small">
                -{{ change.deleted_count }} 删除
              </el-tag>
              <span class="change-time">
                {{ formatTime(change.created_at) }} · {{ change.updated_by }}
              </span>
            </div>
            
            <!-- API 变更详情 -->
            <el-collapse v-if="hasApiChanges(change)" class="api-changes">
              <el-collapse-item
                v-if="getApiList(change.added_apis).length > 0"
                title="新增的 API"
                :name="`added-${change.dir_version}`"
              >
                <div class="api-list">
                  <div
                    v-for="api in getApiList(change.added_apis)"
                    :key="api.code"
                    class="api-item added"
                  >
                    <el-tag type="success" size="small">{{ api.method }}</el-tag>
                    <span class="api-name">{{ api.name }}</span>
                    <span class="api-desc" v-if="api.desc">{{ api.desc }}</span>
                    <span class="api-router">{{ api.router }}</span>
                  </div>
                </div>
              </el-collapse-item>
              
              <el-collapse-item
                v-if="getApiList(change.updated_apis).length > 0"
                title="更新的 API"
                :name="`updated-${change.dir_version}`"
              >
                <div class="api-list">
                  <div
                    v-for="api in getApiList(change.updated_apis)"
                    :key="api.code"
                    class="api-item updated"
                  >
                    <el-tag type="warning" size="small">{{ api.method }}</el-tag>
                    <span class="api-name">{{ api.name }}</span>
                    <span class="api-desc" v-if="api.desc">{{ api.desc }}</span>
                    <span class="api-router">{{ api.router }}</span>
                  </div>
                </div>
              </el-collapse-item>
              
              <el-collapse-item
                v-if="getApiList(change.deleted_apis).length > 0"
                title="删除的 API"
                :name="`deleted-${change.dir_version}`"
              >
                <div class="api-list">
                  <div
                    v-for="api in getApiList(change.deleted_apis)"
                    :key="api.code"
                    class="api-item deleted"
                  >
                    <el-tag type="danger" size="small">{{ api.method }}</el-tag>
                    <span class="api-name">{{ api.name }}</span>
                    <span class="api-desc" v-if="api.desc">{{ api.desc }}</span>
                    <span class="api-router">{{ api.router }}</span>
                  </div>
                </div>
              </el-collapse-item>
            </el-collapse>
          </div>
          
          <!-- 分页 -->
          <div v-if="directoryHistory.paginated" class="pagination-wrapper">
            <el-pagination
              v-model:current-page="currentPage"
              v-model:page-size="pageSize"
              :total="directoryHistory.paginated.total_count"
              :page-sizes="[10, 20, 50, 100]"
              layout="total, sizes, prev, pager, next, jumper"
              @size-change="handleSizeChange"
              @current-change="handlePageChange"
            />
          </div>
        </div>
      </template>
    </div>
    
    <template #footer>
      <el-button @click="handleClose">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  getAppVersionUpdateHistory,
  getDirectoryUpdateHistory,
  type GetAppVersionUpdateHistoryResp,
  type GetDirectoryUpdateHistoryResp,
  type DirectoryChangeInfo,
  type ApiSummary
} from '@/api/directory-update-history'

interface Props {
  modelValue: boolean
  mode: 'app' | 'directory' // app: 工作空间视角, directory: 目录视角
  appId: number
  appVersion?: string // App视角时，可选指定版本
  fullCodePath?: string // 目录视角时，必填
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: false,
  mode: 'app',
  appVersion: '',
  fullCodePath: ''
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const router = useRouter()
const loading = ref(false)
const appHistory = ref<GetAppVersionUpdateHistoryResp | null>(null)
const directoryHistory = ref<GetDirectoryUpdateHistoryResp | null>(null)
const currentPage = ref(1)
const pageSize = ref(10)

const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// 解析 API 列表（处理 json.RawMessage）
const getApiList = (apis: any): ApiSummary[] => {
  if (!apis) return []
  if (Array.isArray(apis)) {
    return apis
  }
  // 如果是字符串，尝试解析 JSON
  if (typeof apis === 'string') {
    try {
      return JSON.parse(apis)
    } catch {
      return []
    }
  }
  return []
}

// 检查是否有API变更
const hasApiChanges = (change: DirectoryChangeInfo) => {
  return (
    getApiList(change.added_apis).length > 0 ||
    getApiList(change.updated_apis).length > 0 ||
    getApiList(change.deleted_apis).length > 0
  )
}

// 格式化时间
const formatTime = (time: string) => {
  if (!time) return ''
  const date = new Date(time)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 加载数据
const loadData = async () => {
  if (!props.appId) return
  
  loading.value = true
  try {
    if (props.mode === 'app') {
      const res = await getAppVersionUpdateHistory(props.appId, props.appVersion)
      appHistory.value = res
    } else {
      if (!props.fullCodePath) {
        ElMessage.warning('目录路径不能为空')
        return
      }
      const res = await getDirectoryUpdateHistory(
        props.appId,
        props.fullCodePath,
        currentPage.value,
        pageSize.value
      )
      directoryHistory.value = res
    }
  } catch (error: any) {
    ElMessage.error(error.message || '加载变更记录失败')
  } finally {
    loading.value = false
  }
}

// 分页处理
const handlePageChange = (page: number) => {
  currentPage.value = page
  loadData()
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
  loadData()
}

// 查看目录
const handleViewDirectory = (fullCodePath: string) => {
  // 跳转到目录详情页
  const pathParts = fullCodePath.split('/').filter(Boolean)
  if (pathParts.length >= 2) {
    const user = pathParts[0]
    const app = pathParts[1]
    const relativePath = pathParts.slice(2).join('/')
    router.push({
      name: 'PackageDetail',
      params: { user, app },
      query: { path: relativePath }
    })
    handleClose()
  }
}

// 关闭对话框
const handleClose = () => {
  visible.value = false
}

// 监听对话框打开
watch(visible, (newVal) => {
  if (newVal) {
    loadData()
  } else {
    // 关闭时重置数据
    appHistory.value = null
    directoryHistory.value = null
    currentPage.value = 1
  }
})

// 监听参数变化
watch([() => props.appId, () => props.appVersion, () => props.fullCodePath], () => {
  if (visible.value) {
    loadData()
  }
})
</script>

<style scoped lang="scss">
.directory-update-history-dialog {
  .history-content {
    min-height: 400px;
    max-height: 70vh;
    overflow-y: auto;
  }
  
  .empty-state {
    padding: 40px 0;
    text-align: center;
  }
  
  .versions-list {
    .version-item {
      margin-bottom: 24px;
      padding: 16px;
      background: var(--el-bg-color-page);
      border-radius: 8px;
      
      .version-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 16px;
        
        .version-info {
          color: var(--el-text-color-secondary);
          font-size: 14px;
        }
      }
      
      .directory-changes {
        .change-item {
          margin-bottom: 16px;
          padding: 12px;
          background: var(--el-bg-color);
          border-radius: 6px;
          border-left: 3px solid var(--el-border-color);
          
          .change-header {
            display: flex;
            align-items: center;
            gap: 12px;
            margin-bottom: 8px;
            
            .change-summary {
              color: var(--el-text-color-regular);
              font-size: 14px;
            }
          }
          
          .change-stats {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 12px;
            
            .change-time {
              margin-left: auto;
              color: var(--el-text-color-secondary);
              font-size: 12px;
            }
          }
        }
      }
    }
  }
  
  .change-item {
    margin-bottom: 16px;
    padding: 16px;
    background: var(--el-bg-color-page);
    border-radius: 8px;
    border-left: 3px solid var(--el-color-primary);
    
    .change-header {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 8px;
      
      .change-summary {
        color: var(--el-text-color-regular);
        font-size: 14px;
      }
    }
    
    .change-stats {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 12px;
      
      .change-time {
        margin-left: auto;
        color: var(--el-text-color-secondary);
        font-size: 12px;
      }
    }
  }
  
  .api-changes {
    margin-top: 12px;
    
    .api-list {
      .api-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 8px 12px;
        margin-bottom: 8px;
        background: var(--el-bg-color);
        border-radius: 4px;
        
        &.added {
          border-left: 3px solid var(--el-color-success);
        }
        
        &.updated {
          border-left: 3px solid var(--el-color-warning);
        }
        
        &.deleted {
          border-left: 3px solid var(--el-color-danger);
        }
        
        .api-name {
          font-weight: 500;
          color: var(--el-text-color-primary);
        }
        
        .api-desc {
          color: var(--el-text-color-secondary);
          font-size: 12px;
        }
        
        .api-router {
          margin-left: auto;
          color: var(--el-text-color-secondary);
          font-size: 12px;
          font-family: monospace;
        }
      }
    }
  }
  
  .pagination-wrapper {
    margin-top: 24px;
    display: flex;
    justify-content: center;
  }
}
</style>

