<template>
  <section class="permission-request-records">
    <div class="request-records-head">
      <div>
        <h3>{{ t('access.resourceRequestRecords') }}</h3>
        <p>{{ resourcePath }}</p>
      </div>
      <el-button size="small" :loading="loading" @click="loadRequests">{{ t('common.refresh') }}</el-button>
    </div>

    <el-tabs v-model="activeTab" class="request-record-tabs">
      <el-tab-pane name="pending">
        <template #label>
          <span class="request-tab-label">
            {{ t('access.pendingTab') }}
            <span v-if="pendingRequests.length > 0" class="request-tab-count is-review">{{ pendingRequests.length }}</span>
          </span>
        </template>
      </el-tab-pane>
      <el-tab-pane name="mine">
        <template #label>
          <span class="request-tab-label">
            {{ t('access.myRequestsTab') }}
            <span v-if="myPendingCount > 0" class="request-tab-count">{{ myPendingCount }}</span>
          </span>
        </template>
      </el-tab-pane>
      <el-tab-pane name="history" :label="t('access.reviewHistoryTab')" />
    </el-tabs>

    <el-table
      v-loading="loading"
      :data="activeRequests"
      row-key="id"
      size="small"
      :empty-text="t('access.noRequests')"
    >
      <el-table-column v-if="activeTab !== 'mine'" :label="t('access.requester')" min-width="150">
        <template #default="{ row }">
          <UsersWidget
            :value="principalUserValue(row.requester)"
            :field="memberUsersField"
            mode="response"
            :field-path="`resourcePermissionRequester:${row.id}`"
          />
        </template>
      </el-table-column>
      <el-table-column :label="t('access.role')" width="110">
        <template #default="{ row }">
          <el-tag size="small" :type="roleTagType(row.requested_role)">{{ roleLabel(row.requested_role) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('access.requestReason')" min-width="210" prop="reason" show-overflow-tooltip />
      <el-table-column :label="t('access.status')" width="105">
        <template #default="{ row }">
          <el-tag size="small" :type="requestStatusTagType(row.status)">{{ requestStatusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('access.requestedAt')" width="168">
        <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column :label="t('access.reviewedBy')" min-width="210">
        <template #default="{ row }">
          <div class="review-result-cell">
            <template v-if="row.status === 'pending' && row.approvers?.length">
              <div
                v-for="approver in row.approvers"
                :key="approverRowKey(row, approver)"
                class="pending-approver"
              >
                <UsersWidget
                  v-if="approver.principal_type === 'user'"
                  :value="principalUserValue(approver.principal_key)"
                  :field="memberUsersField"
                  mode="response"
                  :field-path="`resourcePermissionApprover:${approverRowKey(row, approver)}`"
                />
                <DepartmentDisplay
                  v-else
                  :full-code-path="approver.principal_key"
                  :display-name="departmentPrincipalLabel(approver.principal_key)"
                  mode="simple"
                  size="small"
                />
                <small>{{ roleLabel(approver.role_code) }}</small>
              </div>
            </template>
            <template v-else>
              <UsersWidget
                v-if="row.reviewed_by"
                :value="principalUserValue(row.reviewed_by)"
                :field="memberUsersField"
                mode="response"
                :field-path="`resourcePermissionReviewer:${row.id}`"
              />
              <small v-if="row.review_comment">{{ row.review_comment }}</small>
              <span v-if="!row.reviewed_by && !row.review_comment">-</span>
            </template>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.operation')" width="150" fixed="right">
        <template #default="{ row }">
          <template v-if="activeTab === 'pending'">
            <el-button type="success" link :loading="reviewingRequestID === row.id" @click="approveRequest(row)">
              {{ t('access.approve') }}
            </el-button>
            <el-button type="danger" link :loading="reviewingRequestID === row.id" @click="rejectRequest(row)">
              {{ t('access.reject') }}
            </el-button>
          </template>
          <el-button
            v-else-if="activeTab === 'mine' && row.status === 'pending'"
            type="danger"
            link
            :loading="reviewingRequestID === row.id"
            @click="cancelRequest(row)"
          >
            {{ t('access.cancelRequest') }}
          </el-button>
          <span v-else class="request-record-finished">{{ t('access.requestHandled') }}</span>
        </template>
      </el-table-column>
    </el-table>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { AccessRoleCode } from '@/architecture/domain/types'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'
import { WidgetType } from '@/architecture/domain/constants/widget'
import {
  approvePermissionRequest,
  cancelPermissionRequest,
  listMyPermissionRequests,
  listPendingPermissionRequests,
  listPermissionRequestHistory,
  rejectPermissionRequest,
  type PermissionApprover,
  type PermissionRequest,
  type PermissionRequestStatus,
} from '@/architecture/presentation/context/api/permission'
import { eventBus } from '@/architecture/presentation/context/eventBusContext'
import DepartmentDisplay from '@/architecture/presentation/shared/components/DepartmentDisplay.vue'
import UsersWidget from '@/architecture/presentation/shared/components/UsersWidget.vue'
import { createStringFieldValue } from '@/architecture/domain/utils/widgetFieldHelpers'
import { getErrorMessage } from '@/architecture/shared/apiError'
import { getPermissionRequestWorkspaceRoot } from '@/architecture/presentation/features/access/utils/permissionRequestSummary'

type RequestRecordTab = 'pending' | 'mine' | 'history'

const props = defineProps<{
  resourcePath: string
}>()

const emit = defineEmits<{
  (e: 'changed'): void
}>()

const { t } = useI18n()
const activeTab = ref<RequestRecordTab>('mine')
const loading = ref(false)
const reviewingRequestID = ref<number | null>(null)
const myRequests = ref<PermissionRequest[]>([])
const pendingRequests = ref<PermissionRequest[]>([])
const reviewHistory = ref<PermissionRequest[]>([])
let loadSequence = 0

const memberUsersField = computed<FieldConfig>(() => ({
  code: 'resourcePermissionRequestUsers',
  name: t('access.member'),
  desc: t('access.member'),
  widget: {
    type: WidgetType.USERS,
    config: { max_display_count: 1 },
  },
  data: { type: 'string' },
}))

const myPendingCount = computed(() => myRequests.value.filter(request => request.status === 'pending').length)
const activeRequests = computed(() => {
  if (activeTab.value === 'pending') return pendingRequests.value
  if (activeTab.value === 'history') return reviewHistory.value
  return myRequests.value
})

async function loadRequests() {
  const resourcePath = props.resourcePath
  const root = getPermissionRequestWorkspaceRoot(resourcePath)
  const sequence = ++loadSequence
  if (!resourcePath || !root) {
    myRequests.value = []
    pendingRequests.value = []
    reviewHistory.value = []
    return
  }

  loading.value = true
  try {
    const [mineResult, pendingResult, historyResult] = await Promise.allSettled([
      listMyPermissionRequests(root),
      listPendingPermissionRequests(root),
      listPermissionRequestHistory(root),
    ])
    if (sequence !== loadSequence) return
    const forCurrentResource = (requests: PermissionRequest[]) => requests.filter(request => (
      request.resource_path === resourcePath
    ))
    myRequests.value = mineResult.status === 'fulfilled'
      ? forCurrentResource(mineResult.value.requests || [])
      : []
    pendingRequests.value = pendingResult.status === 'fulfilled'
      ? forCurrentResource(pendingResult.value.requests || [])
      : []
    reviewHistory.value = historyResult.status === 'fulfilled'
      ? forCurrentResource(historyResult.value.requests || [])
      : []
    if (pendingRequests.value.length > 0 && activeTab.value === 'mine' && myRequests.value.length === 0) {
      activeTab.value = 'pending'
    }
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

async function approveRequest(request: PermissionRequest) {
  try {
    const { value } = await ElMessageBox.prompt(
      t('access.approveCommentPrompt'),
      t('access.approveRequestTitle'),
      {
        confirmButtonText: t('access.approve'),
        cancelButtonText: t('common.cancel'),
        inputPlaceholder: t('access.reviewCommentOptional'),
      },
    )
    reviewingRequestID.value = request.id
    await approvePermissionRequest(request.id, String(value || '').trim())
    ElMessage.success(t('access.requestApproved'))
    notifyChanged()
  } catch (error: any) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(getErrorMessage(error, t('access.reviewFailed')))
  } finally {
    reviewingRequestID.value = null
  }
}

async function rejectRequest(request: PermissionRequest) {
  try {
    const { value } = await ElMessageBox.prompt(
      t('access.rejectReasonPrompt'),
      t('access.rejectRequestTitle'),
      {
        confirmButtonText: t('access.reject'),
        cancelButtonText: t('common.cancel'),
        inputPlaceholder: t('access.rejectReasonPlaceholder'),
        inputPattern: /\S+/,
        inputErrorMessage: t('access.rejectReasonRequired'),
      },
    )
    reviewingRequestID.value = request.id
    await rejectPermissionRequest(request.id, String(value || '').trim())
    ElMessage.success(t('access.requestRejected'))
    notifyChanged()
  } catch (error: any) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(getErrorMessage(error, t('access.reviewFailed')))
  } finally {
    reviewingRequestID.value = null
  }
}

async function cancelRequest(request: PermissionRequest) {
  try {
    await ElMessageBox.confirm(
      t('access.cancelRequestConfirm'),
      t('access.cancelRequestTitle'),
      {
        confirmButtonText: t('access.cancelRequest'),
        cancelButtonText: t('common.cancel'),
        type: 'warning',
      },
    )
    reviewingRequestID.value = request.id
    await cancelPermissionRequest(request.id)
    ElMessage.success(t('access.requestCancelled'))
    notifyChanged()
  } catch (error: any) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(getErrorMessage(error, t('access.cancelRequestFailed')))
  } finally {
    reviewingRequestID.value = null
  }
}

function notifyChanged() {
  eventBus.emit('permission-request:changed', { resource_paths: [props.resourcePath] })
  emit('changed')
}

function principalUserValue(username: string): FieldValue {
  return createStringFieldValue(memberUsersField.value, username || '', { emptyRaw: '' })
}

function approverRowKey(request: PermissionRequest, approver: PermissionApprover): string {
  return `${request.id}:${approver.principal_type}:${approver.principal_key}:${approver.resource_path}`
}

function departmentPrincipalLabel(path: string): string | null {
  return path === '/org' ? t('access.allMembers') : null
}

function roleLabel(role: AccessRoleCode): string {
  return t(`access.role${role.charAt(0).toUpperCase()}${role.slice(1)}Title`)
}

function roleTagType(role: AccessRoleCode): 'danger' | 'warning' | 'success' | 'info' {
  if (role === 'owner') return 'danger'
  if (role === 'admin') return 'warning'
  if (role === 'member') return 'success'
  return 'info'
}

function requestStatusLabel(status: PermissionRequestStatus): string {
  return t(`access.requestStatus.${status}`)
}

function requestStatusTagType(status: PermissionRequestStatus): 'success' | 'warning' | 'danger' | 'info' {
  if (status === 'approved') return 'success'
  if (status === 'pending') return 'warning'
  if (status === 'rejected') return 'danger'
  return 'info'
}

function formatDateTime(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

watch(() => props.resourcePath, () => {
  activeTab.value = 'mine'
  void loadRequests()
}, { immediate: true })

const unsubscribe = eventBus.on<{ resource_paths?: string[] }>('permission-request:changed', (payload) => {
  const paths = payload?.resource_paths || []
  if (paths.length === 0 || paths.includes(props.resourcePath)) {
    void loadRequests()
  }
})

onBeforeUnmount(unsubscribe)

defineExpose({ loadRequests })
</script>

<style scoped>
.permission-request-records {
  min-width: 0;
}

.request-records-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 4px;
}

.request-records-head h3,
.request-records-head p {
  margin: 0;
}

.request-records-head h3 {
  font-size: 14px;
}

.request-records-head p {
  margin-top: 3px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.request-tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.request-tab-count {
  min-width: 16px;
  height: 16px;
  padding: 0 5px;
  border-radius: 999px;
  background: #f59e0b;
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  line-height: 16px;
  text-align: center;
}

.request-tab-count.is-review {
  background: #ef4444;
}

.review-result-cell {
  display: grid;
  gap: 3px;
}

.pending-approver {
  display: flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
}

.pending-approver small {
  flex: 0 0 auto;
}

.review-result-cell small,
.request-record-finished {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
