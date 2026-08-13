<template>
  <section class="permission-request-records">
    <el-table
      v-loading="loading"
      :data="activeRequests"
      row-key="id"
      size="small"
      :empty-text="t('access.noRequests')"
    >
      <el-table-column v-if="view !== 'mine'" :label="t('access.requester')" min-width="150">
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
      <el-table-column
        :label="t('common.operation')"
        width="150"
        class-name="permission-request-action-column"
        label-class-name="permission-request-action-column"
      >
        <template #default="{ row }">
          <template v-if="view === 'pending'">
            <el-button type="success" plain size="small" :loading="reviewingRequestID === row.id" @click="approveRequest(row)">
              {{ t('access.approve') }}
            </el-button>
            <el-button type="danger" plain size="small" :loading="reviewingRequestID === row.id" @click="rejectRequest(row)">
              {{ t('access.reject') }}
            </el-button>
          </template>
          <el-button
            v-else-if="view === 'mine' && row.status === 'pending'"
            class="permission-cancel-request-button"
            type="danger"
            plain
            size="small"
            :loading="reviewingRequestID === row.id"
            data-testid="permission-cancel-request"
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
import { settlePermissionRequestSummary } from '@/architecture/presentation/features/access/utils/permissionRequestSummaryStore'

type RequestRecordView = 'pending' | 'mine' | 'history'

const props = defineProps<{
  resourcePath: string
  view: RequestRecordView
}>()

const emit = defineEmits<{
  (e: 'changed'): void
  (e: 'loading-change', loading: boolean): void
}>()

const { t } = useI18n()
const loading = ref(false)
const loadingView = ref<RequestRecordView | null>(null)
const reviewingRequestID = ref<number | null>(null)
const myRequests = ref<PermissionRequest[]>([])
const pendingRequests = ref<PermissionRequest[]>([])
const reviewHistory = ref<PermissionRequest[]>([])
const loadedViews = new Set<RequestRecordView>()
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

const activeRequests = computed(() => {
  if (props.view === 'pending') return pendingRequests.value
  if (props.view === 'history') return reviewHistory.value
  return myRequests.value
})

function clearRequests() {
  loadSequence += 1
  loading.value = false
  loadingView.value = null
  emit('loading-change', false)
  loadedViews.clear()
  myRequests.value = []
  pendingRequests.value = []
  reviewHistory.value = []
}

async function loadRequests(force = true) {
  const view = props.view
  const resourcePath = props.resourcePath
  const root = getPermissionRequestWorkspaceRoot(resourcePath)
  if ((!force && loadedViews.has(view)) || loadingView.value === view) {
    return
  }
  const sequence = ++loadSequence
  if (!resourcePath || !root) {
    clearRequests()
    return
  }

  loading.value = true
  loadingView.value = view
  emit('loading-change', true)
  try {
    let requests: PermissionRequest[] = []
    if (view === 'pending') {
      requests = (await listPendingPermissionRequests(root)).requests || []
    } else if (view === 'history') {
      requests = (await listPermissionRequestHistory(root)).requests || []
    } else {
      requests = (await listMyPermissionRequests(root)).requests || []
    }
    if (sequence !== loadSequence) return
    const currentRequests = requests.filter(request => (
      request.resource_path === resourcePath
    ))
    if (view === 'pending') {
      pendingRequests.value = currentRequests
    } else if (view === 'history') {
      reviewHistory.value = currentRequests
    } else {
      myRequests.value = currentRequests
    }
    loadedViews.add(view)
  } catch (error: any) {
    if (sequence === loadSequence) {
      ElMessage.error(getErrorMessage(error, t('access.loadFailed')))
    }
  } finally {
    if (sequence === loadSequence) {
      loading.value = false
      loadingView.value = null
      emit('loading-change', false)
    }
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
    notifyChanged(request, 'review')
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
    notifyChanged(request, 'review')
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
    notifyChanged(request, 'own')
  } catch (error: any) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(getErrorMessage(error, t('access.cancelRequestFailed')))
  } finally {
    reviewingRequestID.value = null
  }
}

function notifyChanged(request: PermissionRequest, kind: 'own' | 'review') {
  settlePermissionRequestSummary(
    getPermissionRequestWorkspaceRoot(request.resource_path),
    request.resource_path,
    kind,
  )
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
  clearRequests()
  void loadRequests(false)
}, { immediate: true })

watch(() => props.view, () => {
  void loadRequests(false)
})

const unsubscribe = eventBus.on<{ resource_paths?: string[] }>('permission-request:changed', (payload) => {
  const paths = payload?.resource_paths || []
  if (paths.length === 0 || paths.includes(props.resourcePath)) {
    void loadRequests()
  }
})

onBeforeUnmount(unsubscribe)

onBeforeUnmount(() => emit('loading-change', false))

defineExpose({ loadRequests })
</script>

<style scoped>
.permission-request-records {
  min-width: 0;
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

:deep(td.permission-request-action-column .cell) {
  opacity: 1 !important;
  visibility: visible !important;
}

:deep(td.permission-request-action-column) {
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color)) !important;
}

:deep(th.permission-request-action-column) {
  background: var(--el-fill-color-light) !important;
}

:deep(.el-table__body tr:hover > td.permission-request-action-column) {
  background: var(--el-fill-color-light) !important;
}

:deep(.permission-cancel-request-button) {
  opacity: 1 !important;
  visibility: visible !important;
  border-color: var(--el-color-danger) !important;
  background: color-mix(in srgb, var(--el-color-danger) 16%, var(--el-bg-color)) !important;
  color: var(--el-color-danger) !important;
  font-weight: 700;
}
</style>
