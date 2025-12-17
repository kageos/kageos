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
    width="90%"
    :close-on-click-modal="false"
    class="directory-update-history-dialog"
    :style="{ maxWidth: '1400px' }"
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
            class="version-section"
          >
            <!-- 版本标题 -->
            <div class="section-header">
              <h3 class="section-title">
                <el-icon class="section-icon"><Clock /></el-icon>
                版本 {{ version.app_version }}
              </h3>
              <el-tag class="section-badge" type="primary" size="small">
                {{ version.directory_changes.length }} 个目录变更
              </el-tag>
            </div>
            
            <!-- 目录变更卡片列表 -->
            <div class="changes-grid">
              <div
                v-for="change in version.directory_changes"
                :key="`${change.full_code_path}-${change.dir_version}`"
                class="change-card"
              >
                <!-- 卡片头部 -->
                <div class="change-card-header">
                  <div class="change-icon-wrapper">
                    <el-icon class="change-icon"><Folder /></el-icon>
                  </div>
                  <div class="change-title-wrapper">
                    <el-link
                      type="primary"
                      :underline="false"
                      @click="handleViewDirectory(change.full_code_path)"
                      class="change-path"
                    >
                      {{ change.full_code_path }}
                    </el-link>
                    <el-tag size="small" type="info" class="change-version-tag">
                      v{{ change.dir_version_num }}
                    </el-tag>
                  </div>
                </div>
                
                <!-- 变更摘要 -->
                <div v-if="change.summary" class="change-summary">
                  {{ change.summary }}
                </div>
                
                <!-- 统计信息卡片 -->
                <div class="change-stats-card">
                  <div class="stat-item" v-if="change.added_count > 0">
                    <div class="stat-icon-wrapper added-icon">
                      <el-icon class="stat-icon"><Plus /></el-icon>
                    </div>
                    <div class="stat-content">
                      <div class="stat-label">新增</div>
                      <div class="stat-value">{{ change.added_count }}</div>
                    </div>
                  </div>
                  
                  <div class="stat-item" v-if="change.updated_count > 0">
                    <div class="stat-icon-wrapper updated-icon">
                      <el-icon class="stat-icon"><Edit /></el-icon>
                    </div>
                    <div class="stat-content">
                      <div class="stat-label">更新</div>
                      <div class="stat-value">{{ change.updated_count }}</div>
                    </div>
                  </div>
                  
                  <div class="stat-item" v-if="change.deleted_count > 0">
                    <div class="stat-icon-wrapper deleted-icon">
                      <el-icon class="stat-icon"><Delete /></el-icon>
                    </div>
                    <div class="stat-content">
                      <div class="stat-label">删除</div>
                      <div class="stat-value">{{ change.deleted_count }}</div>
                    </div>
                  </div>
                  
                  <div class="stat-item">
                    <div class="stat-icon-wrapper time-icon">
                      <el-icon class="stat-icon"><Clock /></el-icon>
                    </div>
                    <div class="stat-content">
                      <div class="stat-label">更新时间</div>
                      <div class="stat-value">{{ formatTime(change.created_at) }}</div>
                    </div>
                  </div>
                  
                  <div class="stat-item" v-if="change.updated_by">
                    <div class="stat-icon-wrapper user-icon">
                      <el-icon class="stat-icon"><User /></el-icon>
                    </div>
                    <div class="stat-content">
                      <div class="stat-label">操作人</div>
                      <div class="stat-value">{{ change.updated_by }}</div>
                    </div>
                  </div>
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
        
        <div v-else class="changes-grid">
          <div
            v-for="change in directoryHistory.directory_changes"
            :key="`${change.dir_version}`"
            class="change-card"
          >
            <!-- 卡片头部 -->
            <div class="change-card-header">
              <div class="change-icon-wrapper">
                <el-icon class="change-icon"><Clock /></el-icon>
              </div>
              <div class="change-title-wrapper">
                <div class="change-version">
                  <el-tag type="primary" size="large">v{{ change.dir_version_num }}</el-tag>
                </div>
                <div v-if="change.summary" class="change-summary">
                  {{ change.summary }}
                </div>
              </div>
            </div>
            
            <!-- 统计信息卡片 -->
            <div class="change-stats-card">
              <div class="stat-item" v-if="change.added_count > 0">
                <div class="stat-icon-wrapper added-icon">
                  <el-icon class="stat-icon"><Plus /></el-icon>
                </div>
                <div class="stat-content">
                  <div class="stat-label">新增</div>
                  <div class="stat-value">{{ change.added_count }}</div>
                </div>
              </div>
              
              <div class="stat-item" v-if="change.updated_count > 0">
                <div class="stat-icon-wrapper updated-icon">
                  <el-icon class="stat-icon"><Edit /></el-icon>
                </div>
                <div class="stat-content">
                  <div class="stat-label">更新</div>
                  <div class="stat-value">{{ change.updated_count }}</div>
                </div>
              </div>
              
              <div class="stat-item" v-if="change.deleted_count > 0">
                <div class="stat-icon-wrapper deleted-icon">
                  <el-icon class="stat-icon"><Delete /></el-icon>
                </div>
                <div class="stat-content">
                  <div class="stat-label">删除</div>
                  <div class="stat-value">{{ change.deleted_count }}</div>
                </div>
              </div>
              
              <div class="stat-item">
                <div class="stat-icon-wrapper time-icon">
                  <el-icon class="stat-icon"><Clock /></el-icon>
                </div>
                <div class="stat-content">
                  <div class="stat-label">更新时间</div>
                  <div class="stat-value">{{ formatTime(change.created_at) }}</div>
                </div>
              </div>
              
              <div class="stat-item" v-if="change.updated_by">
                <div class="stat-icon-wrapper user-icon">
                  <el-icon class="stat-icon"><User /></el-icon>
                </div>
                <div class="stat-content">
                  <div class="stat-label">操作人</div>
                  <div class="stat-value">{{ change.updated_by }}</div>
                </div>
              </div>
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
import { Clock, Folder, Plus, Edit, Delete, User } from '@element-plus/icons-vue'
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
    padding: 0;
    width: 100%;
  }
  
  .empty-state {
    padding: 40px 0;
    text-align: center;
  }
  
  // 版本列表样式
  .versions-list {
    width: 100%;
    
    .version-section {
      margin-bottom: 32px;
      width: 100%;
      
      .section-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 20px;
        
        .section-title {
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 18px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin: 0;
          
          .section-icon {
            font-size: 20px;
            color: var(--el-color-primary);
          }
        }
        
        .section-badge {
          font-weight: 500;
        }
      }
      
      .changes-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
        gap: 20px;
        width: 100%;
      }
    }
  }
  
  // 变更卡片样式（参考 PackageDetailView 的 overview-card）
  .changes-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
    gap: 20px;
    width: 100%;
  }
  
  .change-card {
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 16px;
    padding: 24px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
    transition: all 0.3s ease;
    
    &:hover {
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
      transform: translateY(-2px);
    }
    
    .change-card-header {
      display: flex;
      align-items: flex-start;
      gap: 16px;
      margin-bottom: 16px;
      
      .change-icon-wrapper {
        flex-shrink: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        width: 48px;
        height: 48px;
        border-radius: 12px;
        background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));
        
        .change-icon {
          font-size: 24px;
          color: var(--el-color-primary);
        }
      }
      
      .change-title-wrapper {
        flex: 1;
        min-width: 0;
        
        .change-path {
          display: block;
          font-size: 16px;
          font-weight: 600;
          color: var(--el-text-color-primary);
          margin-bottom: 8px;
          word-break: break-all;
        }
        
        .change-version-tag {
          margin-top: 4px;
        }
        
        .change-version {
          margin-bottom: 8px;
        }
        
        .change-summary {
          margin-top: 8px;
          font-size: 14px;
          color: var(--el-text-color-regular);
          line-height: 1.5;
        }
      }
    }
    
    .change-stats-card {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
      gap: 16px;
      margin-top: 16px;
      padding-top: 16px;
      border-top: 1px solid var(--el-border-color-lighter);
      
      .stat-item {
        display: flex;
        align-items: center;
        gap: 12px;
        
        .stat-icon-wrapper {
          flex-shrink: 0;
          display: flex;
          align-items: center;
          justify-content: center;
          width: 40px;
          height: 40px;
          border-radius: 10px;
          
          &.added-icon {
            background: linear-gradient(135deg, var(--el-color-success-light-8), var(--el-color-success-light-9));
            
            .stat-icon {
              font-size: 20px;
              color: var(--el-color-success);
            }
          }
          
          &.updated-icon {
            background: linear-gradient(135deg, var(--el-color-warning-light-8), var(--el-color-warning-light-9));
            
            .stat-icon {
              font-size: 20px;
              color: var(--el-color-warning);
            }
          }
          
          &.deleted-icon {
            background: linear-gradient(135deg, var(--el-color-danger-light-8), var(--el-color-danger-light-9));
            
            .stat-icon {
              font-size: 20px;
              color: var(--el-color-danger);
            }
          }
          
          &.time-icon {
            background: linear-gradient(135deg, var(--el-color-info-light-8), var(--el-color-info-light-9));
            
            .stat-icon {
              font-size: 20px;
              color: var(--el-color-info);
            }
          }
          
          &.user-icon {
            background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));
            
            .stat-icon {
              font-size: 20px;
              color: var(--el-color-primary);
            }
          }
        }
        
        .stat-content {
          flex: 1;
          min-width: 0;
          
          .stat-label {
            font-size: 12px;
            color: var(--el-text-color-secondary);
            margin-bottom: 4px;
            font-weight: 500;
          }
          
          .stat-value {
            font-size: 16px;
            font-weight: 600;
            color: var(--el-text-color-primary);
            word-break: break-all;
          }
        }
      }
    }
  }
  
  // API 变更详情样式
  .api-changes {
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid var(--el-border-color-lighter);
    
    :deep(.el-collapse-item__header) {
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
    
    .api-list {
      .api-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 12px 16px;
        margin-bottom: 8px;
        background: var(--el-bg-color-page);
        border-radius: 8px;
        border-left: 3px solid transparent;
        transition: all 0.2s ease;
        
        &:hover {
          background: var(--el-bg-color);
          transform: translateX(4px);
        }
        
        &.added {
          border-left-color: var(--el-color-success);
        }
        
        &.updated {
          border-left-color: var(--el-color-warning);
        }
        
        &.deleted {
          border-left-color: var(--el-color-danger);
        }
        
        .api-name {
          font-weight: 500;
          color: var(--el-text-color-primary);
          font-size: 14px;
        }
        
        .api-desc {
          color: var(--el-text-color-secondary);
          font-size: 12px;
          margin-left: 8px;
        }
        
        .api-router {
          margin-left: auto;
          color: var(--el-text-color-secondary);
          font-size: 12px;
          font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
        }
      }
    }
  }
  
  .pagination-wrapper {
    margin-top: 32px;
    display: flex;
    justify-content: center;
  }
  
  // 响应式设计
  @media (max-width: 768px) {
    .changes-grid {
      grid-template-columns: 1fr;
    }
    
    .change-card {
      .change-stats-card {
        grid-template-columns: 1fr;
      }
    }
  }
}
</style>

