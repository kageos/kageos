<template>
  <div class="permission-request-list" v-loading="loading">
    <!-- 筛选条件 -->
    <div class="filter-section">
      <el-form :inline="true" :model="filterForm" class="filter-form">
        <el-form-item label="状态">
          <el-select v-model="filterForm.status" placeholder="全部" clearable @change="handleFilterChange" style="width: 150px">
            <el-option label="全部" value="" />
            <el-option label="待审批" value="pending" />
            <el-option label="已通过" value="approved" />
            <el-option label="已驳回" value="rejected" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadRequests">刷新</el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 申请列表 -->
    <div class="request-list">
      <el-empty v-if="!loading && requests.length === 0" description="暂无权限申请" />
      <div v-else class="request-items">
        <div
          v-for="request in requests"
          :key="request.id"
          class="request-item"
          :class="{
            'status-pending': request.status === 'pending',
            'status-approved': request.status === 'approved',
            'status-rejected': request.status === 'rejected'
          }"
        >
          <div class="request-header">
            <div class="request-info">
              <span class="request-id">申请 #{{ request.id }}</span>
              <el-tag
                :type="getStatusTagType(request.status)"
                size="small"
                class="status-tag"
              >
                {{ getStatusText(request.status) }}
              </el-tag>
            </div>
            <div class="request-actions" v-if="request.status === 'pending' && canApprove">
              <el-button
                type="success"
                size="small"
                @click="handleApprove(request)"
                :loading="approvingId === request.id"
              >
                同意
              </el-button>
              <el-button
                type="danger"
                size="small"
                @click="handleReject(request)"
                :loading="rejectingId === request.id"
              >
                驳回
              </el-button>
            </div>
          </div>

          <div class="request-body">
            <div class="request-field">
              <span class="field-label">申请人：</span>
              <span class="field-value">{{ request.applicant_username }}</span>
            </div>
            <div class="request-field">
              <span class="field-label">权限主体：</span>
              <span class="field-value">
                {{ request.subject_type === 'user' ? '用户' : '组织架构' }}: {{ request.subject }}
              </span>
            </div>
            <div class="request-field">
              <span class="field-label">资源路径：</span>
              <span class="field-value">{{ request.resource_path }}</span>
            </div>
            <div class="request-field">
              <span class="field-label">操作类型：</span>
              <span class="field-value">{{ request.action }}</span>
            </div>
            <div class="request-field" v-if="request.reason">
              <span class="field-label">申请原因：</span>
              <span class="field-value">{{ request.reason }}</span>
            </div>
            <div class="request-field">
              <span class="field-label">有效期：</span>
              <span class="field-value">
                {{ request.start_time }} 至 {{ request.end_time || '永久' }}
              </span>
            </div>
            <div class="request-field" v-if="request.status === 'approved' && request.approved_by">
              <span class="field-label">审批人：</span>
              <span class="field-value">{{ request.approved_by }} ({{ request.approved_at }})</span>
            </div>
            <div class="request-field" v-if="request.status === 'rejected' && request.rejected_by">
              <span class="field-label">驳回人：</span>
              <span class="field-value">{{ request.rejected_by }} ({{ request.rejected_at }})</span>
            </div>
            <div class="request-field" v-if="request.status === 'rejected' && request.reject_reason">
              <span class="field-label">驳回原因：</span>
              <span class="field-value">{{ request.reject_reason }}</span>
            </div>
            <div class="request-field">
              <span class="field-label">申请时间：</span>
              <span class="field-value">{{ request.created_at }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 分页 -->
    <div class="pagination-section" v-if="total > 0">
      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handlePageChange"
      />
    </div>

    <!-- 驳回原因对话框 -->
    <el-dialog
      v-model="rejectDialogVisible"
      title="驳回原因"
      width="500px"
    >
      <el-form :model="rejectForm" label-width="100px">
        <el-form-item label="驳回原因" required>
          <el-input
            v-model="rejectForm.reason"
            type="textarea"
            :rows="4"
            placeholder="请输入驳回原因"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="rejectDialogVisible = false">取消</el-button>
        <el-button type="danger" @click="confirmReject" :loading="rejectingId !== null">
          确认驳回
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getPermissionRequests, approvePermissionRequest, rejectPermissionRequest } from '@/api/permission'
import { useAuthStore } from '@/stores/auth'

interface Props {
  resourcePath?: string  // 资源路径（可选，如果提供则只显示该资源的申请）
  autoLoad?: boolean  // 是否自动加载
}

const props = withDefaults(defineProps<Props>(), {
  autoLoad: true
})

// 权限申请信息接口
interface PermissionRequest {
  id: number
  app_id: number
  applicant_username: string
  subject_type: string
  subject: string
  resource_path: string
  action: string
  start_time: string
  end_time: string
  reason: string
  status: string
  approved_at?: string
  approved_by?: string
  rejected_at?: string
  rejected_by?: string
  reject_reason?: string
  created_at: string
}

// 状态
const loading = ref(false)
const requests = ref<PermissionRequest[]>([])
const total = ref(0)
const approvingId = ref<number | null>(null)
const rejectingId = ref<number | null>(null)

// 筛选条件
const filterForm = ref({
  status: '' as string
})

// 分页
const pagination = ref({
  page: 1,
  pageSize: 20
})

// 驳回对话框
const rejectDialogVisible = ref(false)
const rejectForm = ref({
  reason: ''
})
const currentRejectRequest = ref<PermissionRequest | null>(null)

// 获取当前用户
const authStore = useAuthStore()
const canApprove = computed(() => {
  // 只有管理员可以审批，这里可以根据实际情况判断
  return true  // TODO: 根据实际权限判断
})

// 加载申请列表
const loadRequests = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.value.page,
      page_size: pagination.value.pageSize
    }
    
    if (filterForm.value.status) {
      params.status = filterForm.value.status
    }
    
    if (props.resourcePath) {
      params.resource_path = props.resourcePath
    }
    
    const response = await getPermissionRequests(params)
    requests.value = response.records || []
    total.value = response.total || 0
  } catch (error: any) {
    ElMessage.error('加载权限申请列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 处理筛选变化
const handleFilterChange = () => {
  pagination.value.page = 1
  loadRequests()
}

// 处理分页变化
const handlePageChange = (page: number) => {
  pagination.value.page = page
  loadRequests()
}

const handleSizeChange = (size: number) => {
  pagination.value.pageSize = size
  pagination.value.page = 1
  loadRequests()
}

// 获取状态标签类型
const getStatusTagType = (status: string) => {
  switch (status) {
    case 'pending':
      return 'warning'
    case 'approved':
      return 'success'
    case 'rejected':
      return 'danger'
    default:
      return 'info'
  }
}

// 获取状态文本
const getStatusText = (status: string) => {
  switch (status) {
    case 'pending':
      return '待审批'
    case 'approved':
      return '已通过'
    case 'rejected':
      return '已驳回'
    default:
      return status
  }
}

// 处理同意
const handleApprove = async (request: PermissionRequest) => {
  try {
    await ElMessageBox.confirm(
      `确定要同意申请 #${request.id} 吗？`,
      '确认审批',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    approvingId.value = request.id
    await approvePermissionRequest({ request_id: request.id })
    ElMessage.success('审批通过成功')
    await loadRequests()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('审批失败: ' + (error.message || '未知错误'))
    }
  } finally {
    approvingId.value = null
  }
}

// 处理驳回
const handleReject = (request: PermissionRequest) => {
  currentRejectRequest.value = request
  rejectForm.value.reason = ''
  rejectDialogVisible.value = true
}

// 确认驳回
const confirmReject = async () => {
  if (!currentRejectRequest.value) {
    return
  }
  
  if (!rejectForm.value.reason.trim()) {
    ElMessage.warning('请输入驳回原因')
    return
  }
  
  try {
    rejectingId.value = currentRejectRequest.value.id
    await rejectPermissionRequest({
      request_id: currentRejectRequest.value.id,
      reason: rejectForm.value.reason
    })
    ElMessage.success('驳回成功')
    rejectDialogVisible.value = false
    await loadRequests()
  } catch (error: any) {
    ElMessage.error('驳回失败: ' + (error.message || '未知错误'))
  } finally {
    rejectingId.value = null
    currentRejectRequest.value = null
  }
}

// 监听 resourcePath 变化
watch(() => props.resourcePath, () => {
  if (props.autoLoad) {
    pagination.value.page = 1
    loadRequests()
  }
}, { immediate: false })

// 初始化
onMounted(() => {
  if (props.autoLoad) {
    loadRequests()
  }
})

// 暴露方法供外部调用
defineExpose({
  loadRequests
})
</script>

<style scoped lang="scss">
.permission-request-list {
  padding: 16px;
  
  .filter-section {
    margin-bottom: 16px;
    
    .filter-form {
      margin: 0;
    }
  }
  
  .request-list {
    min-height: 200px;
    
    .request-items {
      display: flex;
      flex-direction: column;
      gap: 16px;
    }
    
    .request-item {
      border: 1px solid var(--el-border-color);
      border-radius: 4px;
      padding: 16px;
      background-color: var(--el-bg-color);
      transition: all 0.2s;
      
      &:hover {
        box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
      }
      
      &.status-pending {
        border-left: 4px solid var(--el-color-warning);
      }
      
      &.status-approved {
        border-left: 4px solid var(--el-color-success);
      }
      
      &.status-rejected {
        border-left: 4px solid var(--el-color-danger);
      }
      
      .request-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;
        
        .request-info {
          display: flex;
          align-items: center;
          gap: 12px;
          
          .request-id {
            font-weight: 600;
            color: var(--el-text-color-primary);
          }
          
          .status-tag {
            margin: 0;
          }
        }
        
        .request-actions {
          display: flex;
          gap: 8px;
        }
      }
      
      .request-body {
        display: flex;
        flex-direction: column;
        gap: 8px;
        
        .request-field {
          display: flex;
          font-size: 14px;
          
          .field-label {
            min-width: 100px;
            color: var(--el-text-color-secondary);
            font-weight: 500;
          }
          
          .field-value {
            color: var(--el-text-color-primary);
            word-break: break-all;
          }
        }
      }
    }
  }
  
  .pagination-section {
    margin-top: 16px;
    display: flex;
    justify-content: flex-end;
  }
}
</style>
