<template>
  <main class="permission-page">
    <div class="access-shell">
      <el-alert
        v-if="invalidResource"
        class="page-alert"
        type="warning"
        :title="t('access.invalidResource')"
        :closable="false"
        show-icon
      />

      <el-alert
        v-else-if="loadError && !treeAccessDenied"
        class="page-alert"
        type="error"
        :title="loadError"
        :closable="false"
        show-icon
      />

      <section v-else-if="treeAccessDenied" v-loading="pageLoading" class="access-request-layout">
        <section class="access-request-card">
          <div class="access-request-icon">
            <el-icon><Key /></el-icon>
          </div>
          <span class="access-request-kicker">{{ t('access.requestAccessKicker') }}</span>
          <h2>{{ t('access.requestAccessTitle') }}</h2>
          <p>{{ treeAccessDeniedMessage || t('access.requestAccessDesc') }}</p>

          <div class="access-request-resource">
            <span>{{ t('access.resource') }}</span>
            <code>{{ activeResourcePath }}</code>
          </div>

          <div class="access-request-compact-form">
            <el-radio-group v-model="grantRole" @change="handleRequestRoleChange">
              <el-radio-button label="viewer">{{ roleLabel('viewer') }}</el-radio-button>
              <el-radio-button label="member">{{ roleLabel('member') }}</el-radio-button>
              <el-radio-button label="admin">{{ roleLabel('admin') }}</el-radio-button>
            </el-radio-group>
            <el-alert
              v-if="grantRole === 'admin'"
              type="warning"
              :title="t('access.adminRequestWarningTitle')"
              :description="t('access.adminRequestWarningDescription')"
              :closable="false"
              show-icon
            />
            <el-input
              v-model="requestReason"
              type="textarea"
              :rows="3"
              maxlength="1000"
              show-word-limit
              :placeholder="t('access.requestReasonPlaceholder')"
            />
            <p v-if="approvers.length > 0" class="access-request-note">
              {{ t('access.approverCount', { count: approvers.length }) }}
            </p>
            <p v-else class="access-request-note">{{ t('access.noApprover') }}</p>
          </div>

          <div class="access-request-actions">
            <el-button
              type="primary"
              :icon="Key"
              :loading="requestSubmitting"
              :disabled="!canSubmitRequest"
              @click="submitAccessRequest"
            >
              {{ hasPendingRequest(activeResourcePath) ? t('access.requestPending') : t('access.submitRequest') }}
            </el-button>
            <el-button :icon="Document" @click="copyResourcePath">{{ t('access.copyResourcePath') }}</el-button>
            <el-button @click="goBack">
              {{ t('access.backToWorkspace') }}
            </el-button>
          </div>
        </section>
      </section>

      <section v-else v-loading="pageLoading" class="permission-workflow">
        <nav class="workflow-tabs" :aria-label="t('access.workflowTitle')">
          <button
            v-if="canManageActiveResource"
            type="button"
            :class="{ 'is-active': workflowTab === 'grant' }"
            @click="setWorkflowTab('grant')"
          >
            {{ t('access.grantTab') }}
          </button>
          <button
            type="button"
            :class="{ 'is-active': workflowTab === 'request' }"
            @click="setWorkflowTab('request')"
          >
            {{ t('access.requestTab') }}
          </button>
          <button
            v-if="canReviewRequests || pendingRequestCount > 0"
            type="button"
            :class="{ 'is-active': workflowTab === 'pending' }"
            @click="setWorkflowTab('pending')"
          >
            {{ t('access.pendingTab') }}
            <span v-if="pendingRequestCount > 0" class="workflow-tab-badge">{{ pendingRequestCount }}</span>
          </button>
          <button
            type="button"
            :class="{ 'is-active': workflowTab === 'mine' }"
            @click="setWorkflowTab('mine')"
          >
            {{ t('access.myRequestsTab') }}
          </button>
          <button
            v-if="canReviewRequests"
            type="button"
            :class="{ 'is-active': workflowTab === 'history' }"
            @click="setWorkflowTab('history')"
          >
            {{ t('access.reviewHistoryTab') }}
          </button>
        </nav>

        <div
          v-if="workflowTab === 'grant' || workflowTab === 'request'"
          class="apply-layout"
          :class="{ 'is-request-mode': workflowTab === 'request' }"
        >
          <aside class="apply-sidebar">
          <section class="tree-card">
            <div class="panel-toolbar">
              <div class="panel-title-copy">
                <h3>{{ t('access.selectResource') }}</h3>
                <p>{{ t('access.selectResourceDesc') }}</p>
              </div>
              <div class="resource-tools">
                <el-tag size="small" effect="plain">
                  {{ t('common.selectedCount', { count: displayedCheckedResourceCount }) }}
                </el-tag>
                <el-tooltip v-if="canRead(activeResource)" :content="t('access.viewExisting')" placement="bottom">
                  <el-button
                    size="small"
                    :icon="UserFilled"
                    :loading="assignmentsLoading && assignmentsDialogVisible"
                    @click="openAssignmentsDialog"
                  >
                    {{ t('access.existing') }}
                  </el-button>
                </el-tooltip>
              </div>
            </div>

            <div class="tree-container">
              <el-tree
                ref="treeRef"
                class="resource-tree"
                :data="treeData"
                :props="treeProps"
                node-key="full_code_path"
                :show-checkbox="true"
                :check-strictly="true"
                :default-expand-all="true"
                :expand-on-click-node="false"
                :check-on-click-node="true"
                :highlight-current="true"
                :current-node-key="activeResourcePath"
                @node-click="handleResourceClick"
                @check="handleResourceCheck"
              >
                <template #default="{ node: treeNode, data }">
                  <span class="tree-node" :class="{ 'is-selected': activeResourcePath === data.full_code_path }">
                    <img
                      v-if="data.type === 'package'"
                      src="/service-tree/custom-folder.svg"
                      :alt="t('packageDetail.detail')"
                      class="node-icon package-icon-img"
                      :class="getNodeIconClass(data)"
                    />
                    <template v-else-if="data.type === 'function'">
                      <img
                        v-if="data.template_type === TEMPLATE_TYPE.FORM"
                        src="/service-tree/编辑.svg"
                        :alt="t('packageDetail.form')"
                        class="node-icon form-icon-img"
                        :class="getNodeIconClass(data)"
                      />
                      <component
                        v-else
                        :is="getFunctionIcon(data)"
                        class="node-icon"
                        :class="getNodeIconClass(data)"
                      />
                    </template>
                    <img
                      v-else-if="data.type === 'docs'"
                      src="/文档.svg"
                      :alt="t('packageDetail.docs')"
                      class="node-icon docs-icon-img"
                      :class="getNodeIconClass(data)"
                    />
                    <span v-else class="node-icon fx-icon" :class="getNodeIconClass(data)">fx</span>
                    <span class="node-label">{{ treeNode.label }}</span>
                    <span class="resource-node-status">
                      <span
                        v-if="workflowTab === 'request' && hasPendingRequest(data.full_code_path)"
                        class="resource-pending-role"
                      >
                        <el-icon><Clock /></el-icon>
                        {{ t('access.pendingRole', { role: pendingRequestRoleLabel(data.full_code_path) }) }}
                      </span>
                      <span
                        v-if="workflowTab === 'request' && getEffectiveResourceRole(data)"
                        class="resource-current-role"
                        :class="`tone-${getEffectiveResourceRole(data)}`"
                      >
                        {{ data.inherited_from
                          ? t('access.inheritedCurrentRole', { role: effectiveResourceRoleLabel(data) })
                          : t('access.currentRole', { role: effectiveResourceRoleLabel(data) }) }}
                      </span>
                      <span
                        v-if="workflowTab === 'request'
                          && !isResourceCoveredForRequestedRole(data.full_code_path)
                          && !hasPendingRequest(data.full_code_path)
                          && getRequestInheritanceSource(data.full_code_path)"
                        class="resource-inheritance-state"
                        :title="t('access.inheritedRequestSource', { source: getRequestInheritanceSource(data.full_code_path) })"
                      >
                        <el-icon><Connection /></el-icon>
                        {{ isPendingRequestInheritance(data.full_code_path)
                          ? t('access.inheritedFromPendingRequest')
                          : t('access.inheritedFromSelectedParent') }}
                      </span>
                      <el-icon
                        v-if="!canRead(data) && !hasPendingRequest(data.full_code_path)"
                        class="resource-lock-icon"
                        :title="t('access.noReadAccess')"
                      >
                        <Lock />
                      </el-icon>
                    </span>
                  </span>
                </template>
              </el-tree>
            </div>
          </section>
          </aside>

        <section class="apply-main">
          <div class="scope-header-main">
            <div class="scope-title-main">
              <span class="scope-icon">
                <img
                  v-if="currentResourceIconSrc"
                  :src="currentResourceIconSrc"
                  :alt="currentResourceTypeLabel"
                  class="scope-icon-img"
                  :class="currentResourceIconClass"
                />
                <component
                  v-else
                  :is="currentResourceIconComponent"
                  class="scope-icon-component"
                  :class="currentResourceIconClass"
                />
              </span>
              <span class="scope-name-main">{{ activeResource?.name || appName || t('common.currentResource') }}</span>
              <el-tag size="small" :type="currentResourceTagType">{{ currentResourceTypeLabel }}</el-tag>
            </div>
            <div class="scope-path-main">
              <code>{{ activeResourcePath }}</code>
            </div>
          </div>

          <div class="role-selection-section">
            <div class="role-selection-header">
              <div class="role-selection-copy">
                <span class="role-selection-kicker">{{ t('access.roleSwitch') }}</span>
                <h4 class="role-selection-title">
                  <el-icon><UserFilled /></el-icon>
                  {{ t('access.selectRole') }}
                </h4>
                <div class="role-selection-meta">
                  <div class="role-selection-resource-pill">
                    <img
                      v-if="currentResourceIconSrc"
                      :src="currentResourceIconSrc"
                      :alt="currentResourceTypeLabel"
                      class="role-selection-resource-icon"
                      :class="currentResourceIconClass"
                    />
                    <component
                      v-else
                      :is="currentResourceIconComponent"
                      class="role-selection-resource-icon"
                      :class="currentResourceIconClass"
                    />
                    <span>{{ currentResourceTypeLabel }}</span>
                  </div>
                </div>
                <p class="role-selection-desc">{{ t('access.roleCardDesc') }}</p>
              </div>
              <span
                class="role-selection-hint is-selected-role"
                :class="`tone-${selectedRoleOption?.tone || 'edit'}`"
              >
                <el-icon><CircleCheck /></el-icon>
                {{ t('access.selectedRole', { role: roleLabel(grantRole) }) }}
              </span>
            </div>

            <div class="role-cards">
              <button
                v-for="role in visibleRoleOptions"
                :key="role.value"
                class="role-card"
                :class="[`tone-${role.tone}`, {
                  'is-selected': grantRole === role.value,
                  'is-unavailable': isRequestRoleUnavailable(role.value),
                }]"
                :aria-pressed="grantRole === role.value"
                :disabled="isRequestRoleUnavailable(role.value)"
                type="button"
                @click="selectRole(role.value)"
              >
                <span class="role-card-aside">
                  <span class="role-card-identity">
                    <span class="role-icon-badge" :class="`role-icon-badge-${role.tone}`">
                      <el-icon><component :is="role.icon" /></el-icon>
                    </span>
                    <span class="role-card-copy">
                      <span class="role-card-title-row">
                        <strong>{{ role.title }}</strong>
                      </span>
                      <span class="role-card-subtitle">{{ role.subtitle }}</span>
                      <span class="role-card-meta-text">{{ role.codeLabel }} · {{ rolePermissionPointCount(role) }} {{ t('access.permissionPoints') }}</span>
                    </span>
                  </span>
                  <span class="role-card-state" :class="{ 'is-selected': grantRole === role.value }">
                    <el-icon>
                      <component :is="grantRole === role.value || isRequestRoleUnavailable(role.value) ? CircleCheck : CircleClose" />
                    </el-icon>
                    {{ isRequestRoleUnavailable(role.value)
                      ? t('access.roleAlreadyCovered')
                      : (grantRole === role.value ? t('access.roleSelected') : t('access.roleSelect')) }}
                  </span>
                </span>
                <span class="role-action-groups">
                  <span
                    v-for="group in roleActionGroups(role)"
                    :key="`${role.value}-${group.kind}`"
                    class="role-action-group"
                  >
                    <span class="role-action-group-title">
                      <img
                        v-if="group.iconSrc"
                        :src="group.iconSrc"
                        :alt="group.label"
                        class="role-action-group-icon"
                        :class="group.iconClass"
                      />
                      <component
                        v-else
                        :is="group.iconComponent"
                        class="role-action-group-icon"
                        :class="group.iconClass"
                      />
                      <span>{{ group.label }}</span>
                    </span>
                    <span class="role-action-grid">
                      <span
                        v-for="actionState in group.actions"
                        :key="`${role.value}-${group.kind}-${actionState.value}`"
                        class="role-action-item"
                        :class="{
                          'is-enabled': actionState.enabled,
                          'is-disabled': !actionState.enabled,
                        }"
                      >
                        <el-icon class="role-action-icon">
                          <component :is="actionState.enabled ? CircleCheck : CircleClose" />
                        </el-icon>
                        <span class="role-action-copy">
                          <span>{{ actionState.label }}</span>
                          <small v-if="actionState.description">{{ actionState.description }}</small>
                        </span>
                      </span>
                    </span>
                  </span>
                </span>
              </button>
            </div>
            <el-alert
              v-if="workflowTab === 'request' && grantRole === 'admin'"
              class="admin-request-warning"
              type="warning"
              :title="t('access.adminRequestWarningTitle')"
              :description="t('access.adminRequestWarningDescription')"
              :closable="false"
              show-icon
            />
          </div>
        </section>

        <aside class="apply-sidebar-right">
          <section class="form-card">
            <div class="panel-toolbar">
              <div class="panel-title-copy">
                <h3>{{ workflowTab === 'grant' ? t('access.grantInfo') : t('access.requestInfo') }}</h3>
                <p>{{ workflowTab === 'grant' ? t('access.grantInfoDesc') : t('access.requestInfoDesc') }}</p>
              </div>
            </div>

            <el-form v-if="workflowTab === 'grant'" label-position="top" class="grant-form" @submit.prevent>
              <el-form-item :label="t('access.principalType')">
                <el-radio-group v-model="grantPrincipalType">
                  <el-radio-button label="department">{{ t('access.organization') }}</el-radio-button>
                  <el-radio-button label="user">{{ t('access.users') }}</el-radio-button>
                </el-radio-group>
              </el-form-item>

              <el-form-item v-if="grantPrincipalType === 'department'" :label="t('access.grantPrincipal')">
                <el-tree-select
                  v-model="grantDepartmentPath"
                  class="principal-selector"
                  :data="departmentOptions"
                  :loading="departmentsLoading"
                  value-key="value"
                  check-strictly
                  filterable
                  default-expand-all
                  :placeholder="t('access.selectOrganization')"
                />
                <el-alert
                  v-if="grantDepartmentPath === '/org'"
                  class="all-members-alert"
                  type="info"
                  :title="t('access.allMembersGrantTip')"
                  :closable="false"
                  show-icon
                />
              </el-form-item>

              <el-form-item v-else :label="t('access.grantPrincipal')">
                <UsersWidget
                  :value="grantUsersValue"
                  :field="grantUsersField"
                  mode="edit"
                  field-path="permissionPageUsers"
                  @update:modelValue="handleGrantUsersChange"
                />
              </el-form-item>

              <el-form-item :label="t('access.expires')" :error="grantExpiryError">
                <el-radio-group v-model="grantPermanent">
                  <el-radio :label="true">{{ t('access.permanent') }}</el-radio>
                  <el-radio :label="false">{{ t('access.customTime') }}</el-radio>
                </el-radio-group>
                <el-date-picker
                  v-if="!grantPermanent"
                  v-model="grantExpiresAt"
                  class="expires-picker"
                  type="datetime"
                  :placeholder="t('access.expiresPlaceholder')"
                  :disabled-date="disablePastPermissionDate"
                  clearable
                />
              </el-form-item>

              <div class="grant-preview">
                <div class="preview-row">
                  <span>{{ t('access.resource') }}</span>
                  <strong>{{ t('common.resourceCount', { count: selectedResourcePaths.length }) }}</strong>
                </div>
                <div class="preview-row">
                  <span>{{ t('access.principal') }}</span>
                  <strong>{{ principalPreview }}</strong>
                </div>
                <div class="preview-row">
                  <span>{{ t('access.role') }}</span>
                  <strong>{{ roleLabel(grantRole) }}</strong>
                </div>
              </div>

              <el-button
                class="submit-button"
                type="primary"
                :icon="Plus"
                :loading="submitting"
                :disabled="!canSubmitGrant"
                @click="submitGrant"
              >
                {{ t('access.submitGrant') }}
              </el-button>
              <el-button class="cancel-button" @click="goBack">
                {{ t('common.cancel') }}
              </el-button>
            </el-form>

            <el-form v-else label-position="top" class="grant-form request-form" @submit.prevent>
              <el-form-item :label="t('access.requester')">
                <strong>{{ currentUsername || t('access.currentAccount') }}</strong>
              </el-form-item>

              <el-form-item :label="t('access.approvers')">
                <div v-loading="approversLoading" class="approver-list">
                  <div v-for="approver in approvers" :key="approverKey(approver)" class="approver-item">
                    <UsersWidget
                      v-if="approver.principal_type === 'user'"
                      :value="principalUserValue(approver.principal_key)"
                      :field="memberUsersField"
                      mode="response"
                      :field-path="`permissionApprover:${approverKey(approver)}`"
                    />
                    <DepartmentDisplay
                      v-else
                      :full-code-path="approver.principal_key"
                      :display-name="departmentPrincipalLabel(approver.principal_key)"
                      mode="simple"
                      size="small"
                    />
                    <small>{{ roleLabel(approver.role_code) }} · {{ approver.inherited ? t('access.inherited') : t('access.currentResource') }}</small>
                  </div>
                  <el-empty v-if="!approversLoading && approvers.length === 0" :image-size="44" :description="t('access.noApprover')" />
                </div>
              </el-form-item>

              <el-form-item :label="t('access.requestReason')" required>
                <el-input
                  v-model="requestReason"
                  type="textarea"
                  :rows="4"
                  maxlength="1000"
                  show-word-limit
                  :placeholder="t('access.requestReasonPlaceholder')"
                />
              </el-form-item>

              <el-form-item :label="t('access.expires')" :error="requestExpiryError">
                <el-radio-group v-model="requestPermanent">
                  <el-radio :label="true">{{ t('access.permanent') }}</el-radio>
                  <el-radio :label="false">{{ t('access.customTime') }}</el-radio>
                </el-radio-group>
                <el-date-picker
                  v-if="!requestPermanent"
                  v-model="requestExpiresAt"
                  class="expires-picker"
                  type="datetime"
                  :placeholder="t('access.expiresPlaceholder')"
                  :disabled-date="disablePastPermissionDate"
                  clearable
                />
              </el-form-item>

              <div class="grant-preview">
                <div class="preview-row">
                  <span>{{ t('access.resource') }}</span>
                  <strong>{{ t('common.resourceCount', { count: requestTargetPaths.length }) }}</strong>
                </div>
                <div v-if="requestInheritedResourcePaths.length > 0" class="preview-row">
                  <span>{{ t('access.automaticInheritance') }}</span>
                  <strong>{{ t('common.resourceCount', { count: requestInheritedResourcePaths.length }) }}</strong>
                </div>
                <div class="preview-row">
                  <span>{{ t('access.role') }}</span>
                  <strong>{{ roleLabel(grantRole) }}</strong>
                </div>
              </div>

              <el-button
                class="submit-button"
                type="primary"
                :icon="Key"
                :loading="requestSubmitting"
                :disabled="!canSubmitRequest"
                @click="submitAccessRequest"
              >
                {{ t('access.submitRequest') }}
              </el-button>
              <el-button class="cancel-button" @click="goBack">
                {{ t('common.cancel') }}
              </el-button>
            </el-form>
          </section>
        </aside>
        </div>

        <section v-else class="request-records-card">
          <div class="request-records-header">
            <div>
              <h2>{{ workflowRecordTitle }}</h2>
              <p>{{ workflowRecordDescription }}</p>
            </div>
            <el-button :icon="Refresh" :loading="workflowLoading" @click="loadActiveWorkflowRecords">
              {{ t('common.refresh') }}
            </el-button>
          </div>

          <el-table
            v-loading="workflowLoading"
            :data="activeWorkflowRequests"
            row-key="id"
            class="request-records-table"
            :empty-text="t('access.noRequests')"
          >
            <el-table-column v-if="workflowTab !== 'mine'" :label="t('access.requester')" width="150">
              <template #default="{ row }">
                <UsersWidget
                  :value="principalUserValue(row.requester)"
                  :field="memberUsersField"
                  mode="response"
                  :field-path="`permissionRequester:${row.id}`"
                />
              </template>
            </el-table-column>
            <el-table-column :label="t('access.resource')" min-width="260">
              <template #default="{ row }"><code class="request-table-path">{{ row.resource_path }}</code></template>
            </el-table-column>
            <el-table-column :label="t('access.role')" width="110">
              <template #default="{ row }">
                <el-tag size="small" :type="roleTagType(row.requested_role)">{{ roleLabel(row.requested_role) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('access.requestReason')" min-width="220" prop="reason" show-overflow-tooltip />
            <el-table-column :label="t('access.status')" width="110">
              <template #default="{ row }">
                <el-tag size="small" :type="requestStatusTagType(row.status)">{{ requestStatusLabel(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('access.requestedAt')" width="170">
              <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
            </el-table-column>
            <el-table-column v-if="workflowTab === 'history'" :label="t('access.reviewedBy')" min-width="170">
              <template #default="{ row }">
                <div class="review-result-cell">
                  <UsersWidget
                    v-if="row.reviewed_by"
                    :value="principalUserValue(row.reviewed_by)"
                    :field="memberUsersField"
                    mode="response"
                    :field-path="`permissionReviewer:${row.id}`"
                  />
                  <small v-if="row.review_comment">{{ row.review_comment }}</small>
                </div>
              </template>
            </el-table-column>
            <el-table-column :label="t('common.operation')" width="170" fixed="right">
              <template #default="{ row }">
                <template v-if="workflowTab === 'pending'">
                  <el-button type="success" plain size="small" :loading="reviewingRequestID === row.id" @click="approveRequest(row)">
                    {{ t('access.approve') }}
                  </el-button>
                  <el-button type="danger" plain size="small" :loading="reviewingRequestID === row.id" @click="rejectRequest(row)">
                    {{ t('access.reject') }}
                  </el-button>
                </template>
                <el-button
                  v-else-if="workflowTab === 'mine' && row.status === 'pending'"
                  type="danger"
                  plain
                  size="small"
                  :loading="reviewingRequestID === row.id"
                  @click="cancelRequest(row)"
                >
                  {{ t('access.cancelRequest') }}
                </el-button>
                <el-button v-else link @click="openRequestResource(row)">{{ t('access.openResource') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </section>
      </section>
    </div>

    <el-dialog
      v-model="assignmentsDialogVisible"
      class="access-members-dialog"
      width="960px"
      :title="t('access.existing')"
      destroy-on-close
    >
      <div class="members-dialog-header">
        <div>
          <h3>{{ activeResource?.name || t('common.currentResource') }}</h3>
          <p>{{ activeResourcePath }}</p>
        </div>
        <el-button size="small" :icon="Refresh" :loading="assignmentsLoading" @click="loadAssignments">
          {{ t('common.refresh') }}
        </el-button>
      </div>

      <el-tabs v-model="activeTab" class="access-tabs">
        <el-tab-pane :label="t('access.currentDirectoryCount', { count: currentAssignments.length })" name="current" />
        <el-tab-pane :label="t('access.inheritedParentCount', { count: inheritedAssignments.length })" name="inherited" />
      </el-tabs>

      <el-table
        v-loading="assignmentsLoading"
        :data="visibleAssignments"
        class="access-table"
        :row-key="assignmentRowKey"
        size="small"
        :empty-text="t('access.empty')"
      >
        <el-table-column :label="t('access.principal')" min-width="240">
          <template #default="{ row }">
            <div v-if="row.principal_type === 'user'" class="member-users-cell">
              <UsersWidget
                :value="principalUserValue(row.principal_key)"
                :field="memberUsersField"
                mode="response"
                :field-path="`permissionPagePrincipal:${assignmentRowKey(row)}`"
              />
            </div>
            <DepartmentDisplay
              v-else
              :full-code-path="row.principal_key"
              :display-name="departmentPrincipalLabel(row.principal_key)"
              mode="simple"
              size="small"
            />
          </template>
        </el-table-column>
        <el-table-column :label="t('access.role')" width="116">
          <template #default="{ row }">
            <el-tag size="small" :type="roleTagType(row.role_code)">
              {{ roleLabel(row.role_code) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('functionTabs.permission')" min-width="220">
          <template #default="{ row }">
            <div class="permission-tags">
              <el-tag
                v-for="permission in permissionLabels(row.permissions)"
                :key="permission"
                size="small"
                effect="plain"
              >
                {{ permission }}
              </el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="t('access.source')" min-width="190">
          <template #default="{ row }">
            <span v-if="row.direct" class="source-current">{{ t('access.currentDirectory') }}</span>
            <span v-else class="source-inherited">{{ row.inherited_from || row.resource_path }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('access.expiresColumn')" width="150">
          <template #default="{ row }">
            {{ formatExpiresAt(row.expires_at) }}
          </template>
        </el-table-column>
        <el-table-column v-if="canManageActiveResource" :label="t('common.operation')" width="92" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.direct"
              type="danger"
              link
              size="small"
              :icon="Delete"
              :loading="removingKey === assignmentRowKey(row)"
              @click="removeAssignment(row)"
            >
              {{ t('common.remove') }}
            </el-button>
            <el-tooltip v-else :content="t('access.inheritedRemoveTip')" placement="top">
              <span class="disabled-action">{{ t('access.inherited') }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </main>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  CircleCheck,
  CircleClose,
  Clock,
  Connection,
  Delete,
  Document,
  EditPen,
  Key,
  Lock,
  Plus,
  Refresh,
  User,
  UserFilled,
  View
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { Component } from 'vue'
import type { AccessPermissions, AccessRoleCode, App, ServiceTree } from '@/architecture/domain/types'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types/field'
import { TEMPLATE_TYPE } from '@/architecture/domain/constants/functionTypes'
import { WidgetType } from '@/architecture/domain/constants/widget'
import { getAppWithServiceTree } from '@/architecture/presentation/context/api/app'
import {
  approvePermissionRequest,
  batchGrantRoles,
  cancelPermissionRequest,
  createPermissionRequest,
  listMyPermissionRequests,
  listPendingPermissionRequests,
  listPermissionApprovers,
  listPermissionRequestHistory,
  listPermissionAssignments,
  rejectPermissionRequest,
  revokeRole,
  type PermissionApprover,
  type PermissionPrincipal,
  type PermissionPrincipalType,
  type PermissionRequest,
  type PermissionRequestStatus,
  type RoleAssignment
} from '@/architecture/presentation/context/api/permission'
import {
  getDepartmentTree,
  type Department
} from '@/architecture/presentation/context/api/department'
import ChartIcon from '@/architecture/presentation/shared/components/icons/ChartIcon.vue'
import FormIcon from '@/architecture/presentation/shared/components/icons/FormIcon.vue'
import TableIcon from '@/architecture/presentation/shared/components/icons/TableIcon.vue'
import DepartmentDisplay from '@/architecture/presentation/shared/components/DepartmentDisplay.vue'
import UsersWidget from '@/architecture/presentation/shared/components/UsersWidget.vue'
import { createStringFieldValue, extractStringFieldRaw } from '@/architecture/domain/utils/widgetFieldHelpers'
import { buildAppResourcePath, normalizeResourcePath, parseResourcePath } from '@/architecture/shared/resourcePath'
import { resolveWorkspaceUrl } from '@/architecture/shared/routing/route'
import { getErrorMessage, isWorkspaceForbiddenError } from '@/architecture/shared/apiError'
import { canAdmin, canRead } from '@/architecture/presentation/composables/useAccessControl'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'
import { eventBus } from '@/architecture/presentation/context/eventBusContext'
import {
  findNearestPermissionRequestAncestor,
  getPermissionRequestCheckedPaths,
  getPermissionRequestTargetPaths,
  isDescendantResourcePath,
} from '@/architecture/presentation/features/access/utils/permissionRequestSelection'
import {
  getEffectiveAccessRole,
  getRecommendedPermissionRequestRole,
  permissionRequestRoleCovers,
  permissionSetCoversRequestRole,
  type PermissionRequestRole,
} from '@/architecture/presentation/features/access/utils/permissionRequestRole'
import {
  disablePastPermissionDate,
  isPermissionExpiryValid,
} from '@/architecture/presentation/features/access/utils/permissionExpiry'
import {
  getPermissionRequestSummaryState,
  loadPermissionRequestSummary,
} from '@/architecture/presentation/features/access/utils/permissionRequestSummaryStore'

type AccessTab = 'current' | 'inherited'
type RoleTone = 'view' | 'edit' | 'admin' | 'owner'
type ResourceKind = 'app' | 'directory' | 'table' | 'form' | 'chart' | 'docs'
type WorkflowTab = 'grant' | 'request' | 'pending' | 'mine' | 'history'

interface RoleOption {
  value: AccessRoleCode
  title: string
  codeLabel: string
  subtitle: string
  description: string
  permissions: string[]
  tone: RoleTone
  icon: Component
}

interface RoleActionConfig {
  value: AccessPermissionsKey
  label: string
  description?: string
}

interface RoleActionState extends RoleActionConfig {
  enabled: boolean
}

interface RoleActionGroup {
  kind: ResourceKind
  label: string
  iconSrc: string
  iconComponent: Component
  iconClass: string
  actions: RoleActionState[]
}

interface DepartmentOption {
  value: string
  label: string
  children?: DepartmentOption[]
}

type PermissionTreeNode = ServiceTree & {
  permission_request_disabled?: boolean
}

type AccessPermissionsKey = keyof AccessPermissions

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()

const permissionLabelMap = {
  read: 'read',
  write: 'write',
  update: 'update',
  delete: 'delete',
  admin: 'admin',
  owner: 'owner',
} satisfies Record<string, string>

const treeRef = ref()
const treeData = ref<ServiceTree[]>([])
const appName = ref('')
const pageLoading = ref(false)
const assignmentsLoading = ref(false)
const departmentsLoading = ref(false)
const submitting = ref(false)
const removingKey = ref('')
const loadError = ref('')
const treeAccessDenied = ref(false)
const treeAccessDeniedMessage = ref('')
const activeTab = ref<AccessTab>('current')
const activeResourcePath = ref('')
const selectedResourcePaths = ref<string[]>([])
const assignments = ref<RoleAssignment[]>([])
const assignmentsDialogVisible = ref(false)
const departmentOptions = ref<DepartmentOption[]>([])
const grantPrincipalType = ref<PermissionPrincipalType>('department')
const grantDepartmentPath = ref('/org')
const grantRole = ref<AccessRoleCode>('member')
const grantPermanent = ref(true)
const grantExpiresAt = ref<Date | null>(null)
const workflowTab = ref<WorkflowTab>('request')
const requestReason = ref('')
const requestPermanent = ref(true)
const requestExpiresAt = ref<Date | null>(null)
const requestSubmitting = ref(false)
const approversLoading = ref(false)
const workflowLoading = ref(false)
const reviewingRequestID = ref<number | null>(null)
const approvers = ref<PermissionApprover[]>([])
const approverReadyResourcePaths = ref<string[]>([])
const myRequests = ref<PermissionRequest[]>([])
const pendingRequests = ref<PermissionRequest[]>([])
const requestHistory = ref<PermissionRequest[]>([])
const pendingRequestCount = ref(0)
let approverLoadSequence = 0
const loadedWorkflowTabs = new Set<WorkflowTab>()

const grantUsersField = computed<FieldConfig>(() => ({
  code: 'permissionPageUsers',
  name: t('access.users'),
  desc: t('access.memberPickerDesc'),
  widget: {
    type: WidgetType.USERS,
    config: {
      max_count: 50
    }
  },
  data: {
    type: 'string'
  }
}))

const memberUsersField = computed<FieldConfig>(() => ({
  code: 'permissionPageMemberUsers',
  name: t('access.member'),
  desc: t('access.member'),
  widget: {
    type: WidgetType.USERS,
    config: {
      max_display_count: 1
    }
  },
  data: {
    type: 'string'
  }
}))

const grantUsersValue = ref<FieldValue>(createStringFieldValue(grantUsersField.value, '', { emptyRaw: '' }))

const treeProps = {
  children: 'children',
  label: 'name',
  disabled: 'permission_request_disabled',
}

const directoryRoleActionKinds: ResourceKind[] = ['directory', 'table', 'form', 'chart', 'docs']
const appRoleActionKinds: ResourceKind[] = ['app', ...directoryRoleActionKinds]

const roleOptions = computed<RoleOption[]>(() => [
  {
    value: 'viewer',
    title: t('access.roleViewerTitle'),
    codeLabel: 'Viewer',
    subtitle: t('access.roleViewerSubtitle'),
    description: t('access.roleViewerDescription'),
    permissions: ['read'],
    tone: 'view',
    icon: View,
  },
  {
    value: 'member',
    title: t('access.roleMemberTitle'),
    codeLabel: 'Member',
    subtitle: t('access.roleMemberSubtitle'),
    description: t('access.roleMemberDescription'),
    permissions: ['read', 'write', 'update'],
    tone: 'edit',
    icon: EditPen,
  },
  {
    value: 'admin',
    title: t('access.roleAdminTitle'),
    codeLabel: 'Admin',
    subtitle: t('access.roleAdminSubtitle'),
    description: t('access.roleAdminDescription'),
    permissions: ['read', 'write', 'update', 'delete', 'admin'],
    tone: 'admin',
    icon: Key,
  },
  {
    value: 'owner',
    title: t('access.roleOwnerTitle'),
    codeLabel: 'Owner',
    subtitle: t('access.roleOwnerSubtitle'),
    description: t('access.roleOwnerDescription'),
    permissions: ['read', 'write', 'update', 'delete', 'admin', 'owner'],
    tone: 'owner',
    icon: User,
  },
])

const requestedResourcePath = computed(() => {
  const raw = Array.isArray(route.query.resource) ? route.query.resource[0] : route.query.resource
  return normalizeResourcePath(String(raw || ''))
})

const requestedWorkflowMode = computed(() => {
  const raw = Array.isArray(route.query.mode) ? route.query.mode[0] : route.query.mode
  return raw === 'request' ? 'request' : ''
})

const parsedResource = computed(() => parseResourcePath(requestedResourcePath.value))
const invalidResource = computed(() => !parsedResource.value)

const activeResource = computed(() => {
  if (!activeResourcePath.value) return null
  return findNodeByPath(treeData.value, activeResourcePath.value)
})

const currentUsername = computed(() => authStore.user?.username || '')
const workspaceRootPath = computed(() => {
  const parsed = parsedResource.value
  return parsed ? buildAppResourcePath(parsed.user, parsed.app) : ''
})
const canManageActiveResource = computed(() => canAdmin(activeResource.value))
const canReviewRequests = computed(() => treeContainsAdminResource(treeData.value))
const visibleRoleOptions = computed(() => {
  if (workflowTab.value === 'request') {
    return roleOptions.value.filter(role => role.value !== 'owner')
  }
  return roleOptions.value
})
const selectedRoleOption = computed(() => roleOptions.value.find(role => role.value === grantRole.value))
const pendingRequestByPath = computed(() => new Map(
  myRequests.value
    .filter(item => item.status === 'pending')
    .map(item => [item.resource_path, item] as const),
))
const pendingRequestPaths = computed(() => new Set(pendingRequestByPath.value.keys()))
const inheritingPendingRequestPaths = computed(() => new Set(
  grantRole.value === 'owner'
    ? []
    : [...pendingRequestByPath.value.values()]
        .filter(request => permissionRequestRoleCovers(
          request.requested_role,
          grantRole.value as PermissionRequestRole,
        ))
        .map(request => request.resource_path),
))
const requestedRoleCoveredResourcePaths = computed(() => new Set(
  grantRole.value === 'owner'
    ? []
    : collectRoleCoveredResourcePaths(treeData.value, grantRole.value as PermissionRequestRole),
))
const requestTargetPaths = computed(() => getPermissionRequestTargetPaths(
  selectedResourcePaths.value,
  requestedRoleCoveredResourcePaths.value,
  pendingRequestPaths.value,
  inheritingPendingRequestPaths.value,
))
const requestCheckedResourcePaths = computed(() => getPermissionRequestCheckedPaths(
  requestTargetPaths.value,
  requestedRoleCoveredResourcePaths.value,
  pendingRequestPaths.value,
))
const requestInheritanceSourcePaths = computed(() => [
  ...inheritingPendingRequestPaths.value,
  ...requestTargetPaths.value,
])
const requestInheritedResourcePaths = computed(() => collectAllResourcePaths(treeData.value).filter(path => (
  Boolean(findNearestPermissionRequestAncestor(path, requestInheritanceSourcePaths.value))
  && !requestedRoleCoveredResourcePaths.value.has(path)
  && !pendingRequestPaths.value.has(path)
)))
const grantExpiryValid = computed(() => isPermissionExpiryValid(grantPermanent.value, grantExpiresAt.value))
const requestExpiryValid = computed(() => isPermissionExpiryValid(requestPermanent.value, requestExpiresAt.value))
const grantExpiryError = computed(() => {
  if (grantExpiryValid.value) return ''
  return grantExpiresAt.value ? t('access.expiresFutureRequired') : t('access.expiresRequired')
})
const requestExpiryError = computed(() => {
  if (requestExpiryValid.value) return ''
  return requestExpiresAt.value ? t('access.expiresFutureRequired') : t('access.expiresRequired')
})
const displayedCheckedResourceCount = computed(() => (
  workflowTab.value === 'request'
    ? requestTargetPaths.value.length
    : selectedResourcePaths.value.length
))
const canSubmitRequest = computed(() => {
  return Boolean(requestTargetPaths.value.length > 0 && requestReason.value.trim() && approvers.value.length > 0)
    && (grantRole.value === 'viewer' || grantRole.value === 'member' || grantRole.value === 'admin')
    && requestTargetPaths.value.every(path => approverReadyResourcePaths.value.includes(path))
    && requestExpiryValid.value
})
const activeWorkflowRequests = computed(() => {
  if (workflowTab.value === 'pending') return pendingRequests.value
  if (workflowTab.value === 'history') return requestHistory.value
  return myRequests.value
})
const workflowRecordTitle = computed(() => {
  if (workflowTab.value === 'pending') return t('access.pendingTitle')
  if (workflowTab.value === 'history') return t('access.reviewHistoryTitle')
  return t('access.myRequestsTitle')
})
const workflowRecordDescription = computed(() => {
  if (workflowTab.value === 'pending') return t('access.pendingDescription')
  if (workflowTab.value === 'history') return t('access.reviewHistoryDescription')
  return t('access.myRequestsDescription')
})

const currentResourceKind = computed<ResourceKind>(() => {
  const node = activeResource.value
  if (!node) {
    const depth = activeResourcePath.value.split('/').filter(Boolean).length
    return depth <= 2 ? 'app' : 'directory'
  }
  if (node.type === 'docs') return 'docs'
  if (node.type === 'function') {
    if (node.template_type === TEMPLATE_TYPE.FORM) return 'form'
    if (node.template_type === TEMPLATE_TYPE.CHART) return 'chart'
    return 'table'
  }
  const depth = node.full_code_path?.split('/').filter(Boolean).length || 0
  return depth <= 2 ? 'app' : 'directory'
})

const currentResourceTypeLabel = computed(() => getResourceKindLabel(currentResourceKind.value))

const currentResourceTagType = computed((): 'primary' | 'success' | 'warning' | 'info' => {
  if (currentResourceKind.value === 'table') return 'primary'
  if (currentResourceKind.value === 'form' || currentResourceKind.value === 'directory') return 'success'
  if (currentResourceKind.value === 'chart') return 'warning'
  return 'info'
})

const currentResourceIconSrc = computed(() => getResourceIconSrc(currentResourceKind.value))

const currentResourceIconComponent = computed<Component>(() => getResourceIconComponent(currentResourceKind.value))

const currentResourceIconClass = computed(() => getResourceIconClass(currentResourceKind.value))

const selectedUsernames = computed(() => {
  return extractStringFieldRaw(grantUsersValue.value)
    .split(',')
    .map(item => item.trim())
    .filter(Boolean)
})

const selectedPrincipals = computed<PermissionPrincipal[]>(() => {
  if (grantPrincipalType.value === 'department') {
    return grantDepartmentPath.value
      ? [{ type: 'department', key: grantDepartmentPath.value }]
      : []
  }
  return selectedUsernames.value.map(username => ({ type: 'user', key: username }))
})

const principalPreview = computed(() => {
  if (grantPrincipalType.value === 'department') {
    return departmentPrincipalLabel(grantDepartmentPath.value) || grantDepartmentPath.value
  }
  return t('common.userCount', { count: selectedUsernames.value.length })
})

const canSubmitGrant = computed(() => {
  return selectedResourcePaths.value.length > 0
    && selectedPrincipals.value.length > 0
    && Boolean(grantRole.value)
    && grantExpiryValid.value
})

const currentAssignments = computed(() => {
  return assignments.value.filter(assignment => assignment.direct !== false && assignment.source !== 'inherited')
})

const inheritedAssignments = computed(() => {
  return assignments.value.filter(assignment => assignment.direct === false || assignment.source === 'inherited')
})

const visibleAssignments = computed(() => {
  return activeTab.value === 'current' ? currentAssignments.value : inheritedAssignments.value
})

const roleActionConfigs = computed<Record<ResourceKind, RoleActionConfig[]>>(() => ({
  app: [
    { value: 'read', label: t('access.actionAppRead') },
    { value: 'write', label: t('access.actionAppCreate') },
    { value: 'update', label: t('access.actionAppUpdate') },
    { value: 'delete', label: t('access.actionAppDelete') },
    { value: 'admin', label: t('access.actionAdmin'), description: t('access.actionAdminDesc') },
    { value: 'owner', label: t('access.actionOwnership'), description: t('access.actionOwnershipDesc') },
  ],
  directory: [
    { value: 'read', label: t('access.actionDirectoryRead') },
    { value: 'write', label: t('access.actionDirectoryWrite') },
    { value: 'update', label: t('access.actionDirectoryUpdate') },
    { value: 'delete', label: t('access.actionDirectoryDelete') },
    { value: 'admin', label: t('access.actionAdmin'), description: t('access.actionAdminDesc') },
    { value: 'owner', label: t('access.actionOwnership'), description: t('access.actionOwnershipDesc') },
  ],
  table: [
    { value: 'read', label: t('access.actionTableRead') },
    { value: 'write', label: t('access.actionTableWrite') },
    { value: 'update', label: t('access.actionTableUpdate') },
    { value: 'delete', label: t('access.actionTableDelete') },
    { value: 'admin', label: t('access.actionAdmin'), description: t('access.actionAdminDesc') },
    { value: 'owner', label: t('access.actionOwnership'), description: t('access.actionOwnershipDesc') },
  ],
  form: [
    { value: 'read', label: t('access.actionFormRead') },
    { value: 'write', label: t('access.actionFormSubmit') },
    { value: 'admin', label: t('access.actionAdmin'), description: t('access.actionAdminDesc') },
    { value: 'owner', label: t('access.actionOwnership'), description: t('access.actionOwnershipDesc') },
  ],
  chart: [
    { value: 'read', label: t('access.actionChartRead') },
    { value: 'admin', label: t('access.actionAdmin'), description: t('access.actionAdminDesc') },
    { value: 'owner', label: t('access.actionOwnership'), description: t('access.actionOwnershipDesc') },
  ],
  docs: [
    { value: 'read', label: t('access.actionDocsRead') },
    { value: 'write', label: t('access.actionDocsWrite') },
    { value: 'update', label: t('access.actionDocsUpdate') },
    { value: 'delete', label: t('access.actionDocsDelete') },
    { value: 'admin', label: t('access.actionAdmin'), description: t('access.actionAdminDesc') },
    { value: 'owner', label: t('access.actionOwnership'), description: t('access.actionOwnershipDesc') },
  ],
}))

watch([requestedResourcePath, requestedWorkflowMode], () => {
  void reloadPage()
}, { immediate: true })

void loadDepartmentOptions()

async function reloadPage() {
  const parsed = parsedResource.value
  loadError.value = ''
  treeAccessDenied.value = false
  treeAccessDeniedMessage.value = ''
  loadedWorkflowTabs.clear()

  if (!parsed) {
    appName.value = ''
    treeData.value = []
    activeResourcePath.value = ''
    selectedResourcePaths.value = []
    assignments.value = []
    return
  }

  pageLoading.value = true
  try {
    const appResourcePath = buildAppResourcePath(parsed.user, parsed.app)
    const resp = await getAppWithServiceTree(appResourcePath)
    appName.value = resp.app?.name || parsed.app
    treeData.value = buildTreeData(resp.app, resp.service_tree || [])

    const initialPath = findNodeByPath(treeData.value, parsed.resourcePath)?.full_code_path || parsed.resourcePath
    activeTab.value = 'current'
    activeResourcePath.value = initialPath
    workflowTab.value = requestedWorkflowMode.value === 'request' || !canAdmin(findNodeByPath(treeData.value, initialPath))
      ? 'request'
      : 'grant'
    resetGrantForm()
    selectedResourcePaths.value = []
    if (workflowTab.value === 'grant') {
      selectedResourcePaths.value = collectSelectionWithDescendants(initialPath)
    } else {
      setRecommendedRequestRole(findNodeByPath(treeData.value, initialPath))
      selectedResourcePaths.value = isRequestableResourcePath(initialPath) ? [initialPath] : []
    }
    refreshRequestNodeDisabledState()

    await nextTick()
    syncVisibleTreeChecks()
    treeRef.value?.setCurrentKey?.(initialPath)
    await loadPermissionWorkflow(true)
    refreshRequestNodeDisabledState()
    await nextTick()
    syncVisibleTreeChecks()
    if (workflowTab.value === 'request') {
      await loadApprovers()
    }
  } catch (error: any) {
    if (isWorkspaceForbiddenError(error)) {
      showAccessRequestFallback(error)
      await loadPermissionWorkflow(true)
      await loadApprovers()
      return
    }
    const message = getErrorMessage(error, t('access.loadTreeFailed'))
    loadError.value = message
    ElMessage.error(message)
  } finally {
    pageLoading.value = false
  }
}

async function loadDepartmentOptions() {
  departmentsLoading.value = true
  try {
    const response = await getDepartmentTree()
    const root = findDepartmentByPath(response.departments || [], '/org')
    departmentOptions.value = [{
      value: '/org',
      label: t('access.allMembers'),
      children: (root?.children || response.departments || [])
        .filter(department => department.full_code_path !== '/org')
        .map(toDepartmentOption),
    }]
  } catch {
    // /org is a built-in permission principal even when the organization
    // management endpoint is temporarily unavailable.
    departmentOptions.value = [{
      value: '/org',
      label: t('access.allMembers'),
    }]
  } finally {
    departmentsLoading.value = false
  }
}

function findDepartmentByPath(departments: Department[], path: string): Department | null {
  for (const department of departments) {
    if (department.full_code_path === path) return department
    const child = findDepartmentByPath(department.children || [], path)
    if (child) return child
  }
  return null
}

function toDepartmentOption(department: Department): DepartmentOption {
  const children = (department.children || []).map(toDepartmentOption)
  return {
    value: department.full_code_path,
    label: department.name,
    ...(children.length > 0 ? { children } : {}),
  }
}

function showAccessRequestFallback(error: unknown) {
  const parsed = parsedResource.value
  appName.value = parsed?.app || ''
  treeData.value = []
  activeResourcePath.value = requestedResourcePath.value
  workflowTab.value = 'request'
  assignments.value = []
  treeAccessDenied.value = true
  treeAccessDeniedMessage.value = getErrorMessage(error, '')
  resetGrantForm()
  selectedResourcePaths.value = requestedResourcePath.value ? [requestedResourcePath.value] : []
}

function buildTreeData(app: App, serviceTree: ServiceTree[]): ServiceTree[] {
  const appResourcePath = buildAppResourcePath(app.user, app.code)
  const root = serviceTree.find(node => node.full_code_path === appResourcePath)
  if (root) {
    return serviceTree
  }

  return [{
    id: app.id,
    name: app.name || app.code,
    code: app.code,
    type: 'package',
    description: '',
    tags: '',
    app_id: app.id,
    ref_id: app.id,
    full_code_path: appResourcePath,
    created_at: app.created_at,
    updated_at: app.updated_at,
    children: serviceTree
  }]
}

async function loadAssignments() {
  const path = activeResourcePath.value
  if (!path) {
    assignments.value = []
    return
  }

  assignmentsLoading.value = true
  try {
    const resp = await listPermissionAssignments(path)
    assignments.value = resp.assignments || []
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || t('access.loadFailed')
    ElMessage.error(message)
  } finally {
    assignmentsLoading.value = false
  }
}

async function loadApprovers() {
  const sequence = ++approverLoadSequence
  const paths = workflowTab.value === 'request'
    ? [...requestTargetPaths.value]
    : (activeResourcePath.value ? [activeResourcePath.value] : [])
  if (paths.length === 0) {
    approvers.value = []
    approverReadyResourcePaths.value = []
    approversLoading.value = false
    return
  }
  approversLoading.value = true
  try {
    const results = await Promise.allSettled(paths.map(async path => ({
      path,
      response: await listPermissionApprovers(path),
    })))
    if (sequence !== approverLoadSequence) return

    const uniqueApprovers = new Map<string, PermissionApprover>()
    const readyPaths: string[] = []
    let firstError: unknown = null
    for (const result of results) {
      if (result.status === 'rejected') {
        firstError ||= result.reason
        continue
      }
      const resourceApprovers = result.value.response.approvers || []
      if (resourceApprovers.length > 0) {
        readyPaths.push(result.value.path)
      }
      for (const approver of resourceApprovers) {
        uniqueApprovers.set(approverKey(approver), approver)
      }
    }
    approvers.value = [...uniqueApprovers.values()]
    approverReadyResourcePaths.value = readyPaths
    if (firstError) {
      ElMessage.error(getErrorMessage(firstError, t('access.loadApproversFailed')))
    }
  } finally {
    if (sequence === approverLoadSequence) {
      approversLoading.value = false
    }
  }
}

async function loadPermissionWorkflow(force = false) {
  const root = workspaceRootPath.value
  if (!root) return
  const tab = workflowTab.value
  if (force) loadedWorkflowTabs.clear()
  workflowLoading.value = true
  try {
    await loadPermissionRequestSummary(root, { force })
    pendingRequestCount.value = getPermissionRequestSummaryState(root).reviewPendingCount
    if ((tab === 'grant') || (!force && loadedWorkflowTabs.has(tab))) return

    if (tab === 'request') {
      myRequests.value = (await listMyPermissionRequests(root, 'pending')).requests || []
    } else if (tab === 'mine') {
      myRequests.value = (await listMyPermissionRequests(root)).requests || []
    } else if (tab === 'pending') {
      pendingRequests.value = (await listPendingPermissionRequests(root)).requests || []
      pendingRequestCount.value = pendingRequests.value.length
    } else if (tab === 'history') {
      requestHistory.value = (await listPermissionRequestHistory(root)).requests || []
    }
    loadedWorkflowTabs.add(tab)
  } finally {
    workflowLoading.value = false
  }
}

async function loadActiveWorkflowRecords() {
  await loadPermissionWorkflow(true)
}

function setWorkflowTab(tab: WorkflowTab) {
  if (tab === 'grant' && !canManageActiveResource.value) return
  workflowTab.value = tab
  if (tab === 'grant') {
    grantRole.value = 'member'
    selectedResourcePaths.value = collectSelectionWithDescendants(activeResourcePath.value)
    refreshRequestNodeDisabledState()
    void nextTick(syncVisibleTreeChecks)
  } else if (tab === 'request') {
    selectedResourcePaths.value = []
    setRecommendedRequestRole(activeResource.value)
    selectedResourcePaths.value = isRequestableResourcePath(activeResourcePath.value)
      ? [activeResourcePath.value]
      : []
    refreshRequestNodeDisabledState()
    void nextTick(() => {
      syncVisibleTreeChecks()
      void loadApprovers()
    })
  } else {
    void loadActiveWorkflowRecords()
  }
}

function setRecommendedRequestRole(node: ServiceTree | null) {
  grantRole.value = getRecommendedPermissionRequestRole(node?.permissions) || 'admin'
}

function selectRole(role: AccessRoleCode) {
  if (isRequestRoleUnavailable(role)) return
  grantRole.value = role
  handleRequestRoleChange()
}

function handleRequestRoleChange() {
  if (workflowTab.value !== 'request') return
  refreshRequestNodeDisabledState()
  void nextTick(() => {
    syncVisibleTreeChecks()
    void loadApprovers()
  })
}

function isRequestRoleUnavailable(role: AccessRoleCode): boolean {
  if (workflowTab.value !== 'request') return false
  if (role === 'owner') return true
  const paths = selectedResourcePaths.value.length > 0
    ? selectedResourcePaths.value
    : (activeResourcePath.value ? [activeResourcePath.value] : [])
  if (paths.length === 0 || treeAccessDenied.value) return false

  return paths.every(path => {
    const node = findNodeByPath(treeData.value, path)
    return Boolean(node && permissionSetCoversRequestRole(node.permissions, role))
  })
}

async function submitAccessRequest() {
  if (!canSubmitRequest.value) {
    if (hasPendingRequest(activeResourcePath.value)) {
      ElMessage.warning(t('access.requestAlreadyPending'))
    } else {
      ElMessage.warning(t('access.completeRequestForm'))
    }
    return
  }
  const targetPaths = [...requestTargetPaths.value]
  requestSubmitting.value = true
  try {
    const results = await Promise.allSettled(targetPaths.map(resourcePath => createPermissionRequest({
      resource_path: resourcePath,
      role_code: grantRole.value as PermissionRequestRole,
      reason: requestReason.value.trim(),
      expires_at: requestPermanent.value ? null : (requestExpiresAt.value?.toISOString() || null),
    })))
    const successfulCount = results.filter(result => result.status === 'fulfilled').length
    const failedResults = results.filter(result => result.status === 'rejected') as PromiseRejectedResult[]

    if (failedResults.length === 0) {
      ElMessage.success(t('access.requestResourcesSubmitted', { count: successfulCount }))
      requestReason.value = ''
      requestPermanent.value = true
      requestExpiresAt.value = null
    } else if (successfulCount > 0) {
      ElMessage.warning(t('access.requestResourcesPartiallySubmitted', {
        success: successfulCount,
        failed: failedResults.length,
      }))
    } else {
      ElMessage.error(getErrorMessage(failedResults[0]?.reason, t('access.requestSubmitFailed')))
    }

    if (successfulCount > 0) {
      eventBus.emit('permission-request:changed', { resource_paths: targetPaths })
    }
    await loadPermissionWorkflow(true)
    if (failedResults.length === 0 && targetPaths[0]) {
      await router.push({
        path: resolveWorkspaceUrl(targetPaths[0]),
        query: { _panel: 'permission' },
      })
    } else if (!treeAccessDenied.value) {
      selectedResourcePaths.value = getPermissionRequestTargetPaths(
        targetPaths,
        requestedRoleCoveredResourcePaths.value,
        pendingRequestPaths.value,
        inheritingPendingRequestPaths.value,
      )
      refreshRequestNodeDisabledState()
      await nextTick()
      syncVisibleTreeChecks()
      await loadApprovers()
    }
  } catch (error: any) {
    ElMessage.error(getErrorMessage(error, t('access.requestSubmitFailed')))
  } finally {
    requestSubmitting.value = false
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
      }
    )
    reviewingRequestID.value = request.id
    await approvePermissionRequest(request.id, String(value || '').trim())
    ElMessage.success(t('access.requestApproved'))
    eventBus.emit('permission-request:changed', { resource_paths: [request.resource_path] })
    await loadPermissionWorkflow(true)
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
      }
    )
    reviewingRequestID.value = request.id
    await rejectPermissionRequest(request.id, String(value || '').trim())
    ElMessage.success(t('access.requestRejected'))
    eventBus.emit('permission-request:changed', { resource_paths: [request.resource_path] })
    await loadPermissionWorkflow(true)
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
        type: 'warning'
      }
    )
    reviewingRequestID.value = request.id
    await cancelPermissionRequest(request.id)
    ElMessage.success(t('access.requestCancelled'))
    eventBus.emit('permission-request:changed', { resource_paths: [request.resource_path] })
    await loadPermissionWorkflow(true)
  } catch (error: any) {
    if (error === 'cancel' || error === 'close') return
    ElMessage.error(getErrorMessage(error, t('access.cancelRequestFailed')))
  } finally {
    reviewingRequestID.value = null
  }
}

function openRequestResource(request: PermissionRequest) {
  void router.push(resolveWorkspaceUrl(request.resource_path))
}

function hasPendingRequest(resourcePath?: string): boolean {
  return Boolean(resourcePath && pendingRequestPaths.value.has(resourcePath))
}

function pendingRequestRoleLabel(resourcePath?: string): string {
  if (!resourcePath) return ''
  const request = pendingRequestByPath.value.get(resourcePath)
  return request ? roleLabel(request.requested_role) : ''
}

function approverKey(approver: PermissionApprover): string {
  return `${approver.principal_type}:${approver.principal_key}:${approver.resource_path}`
}

function treeContainsAdminResource(nodes: ServiceTree[]): boolean {
  return nodes.some(node => canAdmin(node) || treeContainsAdminResource(node.children || []))
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

function handleResourceClick(data: ServiceTree) {
  if (!data.full_code_path) return
  activeTab.value = 'current'
  activeResourcePath.value = data.full_code_path
  if (workflowTab.value === 'grant' && !canAdmin(data)) {
    workflowTab.value = 'request'
    selectedResourcePaths.value = []
    setRecommendedRequestRole(data)
    selectedResourcePaths.value = isRequestableResourcePath(data.full_code_path)
      ? [data.full_code_path]
      : []
    refreshRequestNodeDisabledState()
    void nextTick(() => {
      syncVisibleTreeChecks()
      treeRef.value?.setCurrentKey?.(data.full_code_path)
      void loadApprovers()
    })
    return
  }
  void nextTick(() => treeRef.value?.setCurrentKey?.(data.full_code_path))
}

function handleResourceCheck(data: ServiceTree) {
  if (workflowTab.value === 'request') {
    if (!isRequestableResourcePath(data?.full_code_path)) {
      syncVisibleTreeChecks()
      return
    }
    applyRequestSelection(data, isResourceChecked(data))
    return
  }
  if (workflowTab.value !== 'grant') return
  if (data?.full_code_path) {
    applyResourceSelectionCascade(data, isResourceChecked(data))
  } else {
    syncCheckedResourcePaths()
  }
}

function syncCheckedResourcePaths() {
  const checkedKeys = treeRef.value?.getCheckedKeys?.() as string[] | undefined
  selectedResourcePaths.value = normalizeResourcePathList(checkedKeys || [])
  if (selectedResourcePaths.value.length > 0 && !selectedResourcePaths.value.includes(activeResourcePath.value)) {
    activeResourcePath.value = selectedResourcePaths.value[0] || ''
    treeRef.value?.setCurrentKey?.(activeResourcePath.value)
  }
}

function isResourceChecked(node: ServiceTree): boolean {
  if (!node.full_code_path) return false
  const checkedKeys = treeRef.value?.getCheckedKeys?.() as string[] | undefined
  return new Set(checkedKeys || []).has(node.full_code_path)
}

function applyResourceSelectionCascade(node: ServiceTree, checked: boolean) {
  const checkedKeys = new Set((treeRef.value?.getCheckedKeys?.() as string[] | undefined) || [])
  for (const path of collectNodeAndDescendantPaths(node)) {
    if (checked) {
      checkedKeys.add(path)
    } else {
      checkedKeys.delete(path)
    }
  }
  treeRef.value?.setCheckedKeys?.([...checkedKeys])
  syncCheckedResourcePaths()
}

function applyRequestSelection(node: ServiceTree, checked: boolean) {
  const selectedPaths = new Set(requestTargetPaths.value)
  if (checked) {
    selectedPaths.add(node.full_code_path)
    for (const path of [...selectedPaths]) {
      if (isDescendantResourcePath(path, node.full_code_path)) {
        selectedPaths.delete(path)
      }
    }
  } else {
    selectedPaths.delete(node.full_code_path)
  }
  selectedResourcePaths.value = getPermissionRequestTargetPaths(
    selectedPaths,
    requestedRoleCoveredResourcePaths.value,
    pendingRequestPaths.value,
    inheritingPendingRequestPaths.value,
  )
  refreshRequestNodeDisabledState()
  syncVisibleTreeChecks()
  void loadApprovers()
}

function isRequestableResourcePath(resourcePath?: string): boolean {
  return Boolean(
    resourcePath
    && !requestedRoleCoveredResourcePaths.value.has(resourcePath)
    && !pendingRequestPaths.value.has(resourcePath)
    && !getRequestInheritanceSource(resourcePath),
  )
}

function isResourceCoveredForRequestedRole(resourcePath?: string): boolean {
  return Boolean(resourcePath && requestedRoleCoveredResourcePaths.value.has(resourcePath))
}

function getEffectiveResourceRole(node: ServiceTree): AccessRoleCode | null {
  return getEffectiveAccessRole(node.permissions)
}

function effectiveResourceRoleLabel(node: ServiceTree): string {
  const role = getEffectiveResourceRole(node)
  return role ? roleLabel(role) : ''
}

function getRequestInheritanceSource(resourcePath?: string): string {
  if (!resourcePath) return ''
  return findNearestPermissionRequestAncestor(resourcePath, requestInheritanceSourcePaths.value) || ''
}

function isPendingRequestInheritance(resourcePath?: string): boolean {
  const source = getRequestInheritanceSource(resourcePath)
  return Boolean(source && inheritingPendingRequestPaths.value.has(source))
}

function syncVisibleTreeChecks() {
  const checkedPaths = workflowTab.value === 'request'
    ? requestCheckedResourcePaths.value
    : selectedResourcePaths.value
  treeRef.value?.setCheckedKeys?.(checkedPaths)
}

function refreshRequestNodeDisabledState(nodes: ServiceTree[] = treeData.value) {
  for (const node of nodes) {
    const permissionNode = node as PermissionTreeNode
    permissionNode.permission_request_disabled = workflowTab.value === 'request'
      && !isRequestableResourcePath(node.full_code_path)
    refreshRequestNodeDisabledState(node.children || [])
  }
}

function collectRoleCoveredResourcePaths(nodes: ServiceTree[], role: PermissionRequestRole): string[] {
  const paths: string[] = []
  for (const node of nodes) {
    if (node.full_code_path && permissionSetCoversRequestRole(node.permissions, role)) {
      paths.push(node.full_code_path)
    }
    paths.push(...collectRoleCoveredResourcePaths(node.children || [], role))
  }
  return normalizeResourcePathList(paths)
}

function collectAllResourcePaths(nodes: ServiceTree[]): string[] {
  const paths: string[] = []
  for (const node of nodes) {
    if (node.full_code_path) {
      paths.push(node.full_code_path)
    }
    paths.push(...collectAllResourcePaths(node.children || []))
  }
  return normalizeResourcePathList(paths)
}

function handleGrantUsersChange(value: FieldValue) {
  grantUsersValue.value = value
}

function resetGrantForm() {
  grantPrincipalType.value = 'department'
  grantDepartmentPath.value = '/org'
  grantUsersValue.value = createStringFieldValue(grantUsersField.value, '', { emptyRaw: '' })
  grantRole.value = 'member'
  grantPermanent.value = true
  grantExpiresAt.value = null
}

async function copyResourcePath() {
  const path = activeResourcePath.value || requestedResourcePath.value
  if (!path) return

  try {
    await navigator.clipboard.writeText(path)
    ElMessage.success(t('access.copyResourcePathSuccess'))
  } catch {
    ElMessage.warning(t('access.copyResourcePathFailed'))
  }
}

function goBack() {
  const target = requestedResourcePath.value ? resolveWorkspaceUrl(requestedResourcePath.value) : '/workspace'
  void router.push(target)
}

async function openAssignmentsDialog() {
  if (!canRead(activeResource.value)) return
  assignmentsDialogVisible.value = true
  await loadAssignments()
}

function getFunctionIcon(data: ServiceTree): Component {
  if (data.template_type === TEMPLATE_TYPE.TABLE) return TableIcon
  if (data.template_type === TEMPLATE_TYPE.FORM) return FormIcon
  if (data.template_type === TEMPLATE_TYPE.CHART) return ChartIcon
  return Document
}

function getNodeIconClass(data: ServiceTree): string {
  if (data.type === 'package') return 'package-icon'
  if (data.type === 'function') {
    if (data.template_type === TEMPLATE_TYPE.TABLE) return 'table-icon'
    if (data.template_type === TEMPLATE_TYPE.FORM) return 'form-icon'
    if (data.template_type === TEMPLATE_TYPE.CHART) return 'chart-icon'
    return 'function-icon'
  }
  if (data.type === 'docs') return 'docs-icon'
  return 'function-icon'
}

async function submitGrant() {
  if (!canManageActiveResource.value || !canSubmitGrant.value) {
    ElMessage.warning(t('access.selectResourceUserRole'))
    return
  }

  submitting.value = true
  try {
    await batchGrantRoles({
      resource_paths: selectedResourcePaths.value,
      principals: selectedPrincipals.value,
      role_codes: [grantRole.value],
      expires_at: grantPermanent.value ? null : (grantExpiresAt.value ? grantExpiresAt.value.toISOString() : null)
    })
    ElMessage.success(t('access.grantResourcesSuccess', {
      principals: selectedPrincipals.value.length,
      resources: selectedResourcePaths.value.length
    }))
    if (grantPrincipalType.value === 'user') {
      grantUsersValue.value = createStringFieldValue(grantUsersField.value, '', { emptyRaw: '' })
    }
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || t('access.grantFailed')
    ElMessage.error(message)
  } finally {
    submitting.value = false
  }
}

async function removeAssignment(assignment: RoleAssignment) {
  const key = assignmentRowKey(assignment)
  const principal = principalLabel(assignment)
  try {
    await ElMessageBox.confirm(
      t('access.removeDirectoryConfirm', { principal, role: roleLabel(assignment.role_code) }),
      t('access.removeConfirmTitle'),
      {
        confirmButtonText: t('common.remove'),
        cancelButtonText: t('common.cancel'),
        type: 'warning',
      }
    )
  } catch {
    return
  }

  removingKey.value = key
  try {
    await revokeRole({
      resource_path: assignment.resource_path,
      principal: {
        type: assignment.principal_type,
        key: assignment.principal_key,
      },
      role_code: assignment.role_code
    })
    ElMessage.success(t('access.removeSuccess'))
    await loadAssignments()
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || t('access.removeFailed')
    ElMessage.error(message)
  } finally {
    removingKey.value = ''
  }
}

function findNodeByPath(nodes: ServiceTree[], path: string): ServiceTree | null {
  for (const item of nodes) {
    if (item.full_code_path === path) return item
    const child = item.children?.length ? findNodeByPath(item.children, path) : null
    if (child) return child
  }
  return null
}

function collectNodeAndDescendantPaths(node: ServiceTree): string[] {
  const paths: string[] = []
  const walk = (current: ServiceTree) => {
    if (current.full_code_path) {
      paths.push(current.full_code_path)
    }
    for (const child of current.children || []) {
      walk(child)
    }
  }
  walk(node)
  return paths
}

function collectSelectionWithDescendants(path: string): string[] {
  const node = findNodeByPath(treeData.value, path)
  return node ? normalizeResourcePathList(collectNodeAndDescendantPaths(node)) : [path]
}

function normalizeResourcePathList(paths: string[]): string[] {
  return [...new Set(paths.map(item => String(item).trim()).filter(Boolean))]
}

function assignmentRowKey(assignment: RoleAssignment): string {
  return `${assignment.principal_type}:${assignment.principal_key}:${assignment.resource_path}:${assignment.role_code}`
}

function roleLabel(role: AccessRoleCode): string {
  const option = roleOptions.value.find(item => item.value === role)
  return option?.title || role
}

function roleActionGroups(role: RoleOption): RoleActionGroup[] {
  return getRoleActionKinds()
    .map(kind => ({
      kind,
      label: getResourceKindLabel(kind),
      iconSrc: getResourceIconSrc(kind),
      iconComponent: getResourceIconComponent(kind),
      iconClass: getResourceIconClass(kind),
      actions: roleActionStatesForKind(role, kind),
    }))
    .filter(group => group.actions.length > 0)
}

function getRoleActionKinds(): ResourceKind[] {
  if (currentResourceKind.value === 'app') return appRoleActionKinds
  if (currentResourceKind.value === 'directory') return directoryRoleActionKinds
  return [currentResourceKind.value]
}

function roleActionStatesForKind(role: RoleOption, kind: ResourceKind): RoleActionState[] {
  const actionConfigs = (roleActionConfigs.value[kind] || []).filter(action => shouldShowRoleAction(role, action.value))
  const enabledActions = new Set(role.permissions)
  const hasAdminCoverage = enabledActions.has('admin')
  const hasOwnerCoverage = enabledActions.has('owner')
  return actionConfigs.map(action => ({
    ...action,
    enabled: hasOwnerCoverage
      || (action.value !== 'owner' && hasAdminCoverage)
      || enabledActions.has(action.value)
  }))
}

function shouldShowRoleAction(role: RoleOption, action: AccessPermissionsKey): boolean {
  if (action === 'admin') return role.value === 'admin' || role.value === 'owner'
  if (action === 'owner') return role.value === 'owner'
  return true
}

function rolePermissionPointCount(role: RoleOption): number {
  return roleActionGroups(role).reduce((total, group) => {
    return total + group.actions.filter(action => action.enabled).length
  }, 0)
}

function getResourceKindLabel(kind: ResourceKind): string {
  const labels: Record<ResourceKind, string> = {
    app: t('access.resourceApp'),
    directory: t('access.resourceDirectory'),
    table: t('packageDetail.table'),
    form: t('packageDetail.form'),
    chart: t('packageDetail.chart'),
    docs: t('packageDetail.docs'),
  }
  return labels[kind]
}

function getResourceIconSrc(kind: ResourceKind): string {
  if (kind === 'app' || kind === 'directory') return '/service-tree/custom-folder.svg'
  if (kind === 'form') return '/service-tree/编辑.svg'
  if (kind === 'docs') return '/文档.svg'
  return ''
}

function getResourceIconComponent(kind: ResourceKind): Component {
  if (kind === 'chart') return ChartIcon
  return TableIcon
}

function getResourceIconClass(kind: ResourceKind): string {
  if (kind === 'app') return 'app-icon-img'
  if (kind === 'directory') return 'package-icon-img'
  if (kind === 'form') return 'form-icon-img'
  if (kind === 'docs') return 'docs-icon-img'
  if (kind === 'chart') return 'chart-icon'
  return 'table-icon'
}

function principalUserValue(username: string): FieldValue {
  return createStringFieldValue(memberUsersField.value, username || '', { emptyRaw: '' })
}

function departmentPrincipalLabel(path: string): string | null {
  if (path === '/org') return t('access.allMembers')
  const findLabel = (options: DepartmentOption[]): string | null => {
    for (const option of options) {
      if (option.value === path) return option.label
      const childLabel = option.children?.length ? findLabel(option.children) : null
      if (childLabel) return childLabel
    }
    return null
  }
  return findLabel(departmentOptions.value)
}

function principalLabel(assignment: RoleAssignment): string {
  if (assignment.principal_type === 'department') {
    return departmentPrincipalLabel(assignment.principal_key) || assignment.principal_key
  }
  return assignment.principal_key
}

function roleTagType(role: AccessRoleCode): 'danger' | 'warning' | 'success' | 'info' {
  if (role === 'owner') return 'danger'
  if (role === 'admin') return 'warning'
  if (role === 'member') return 'success'
  return 'info'
}

function permissionLabels(permissions: AccessPermissions | null | undefined): string[] {
  if (!permissions) return []
  const labels = Object.entries(permissionLabelMap)
    .filter(([key]) => permissions[key as keyof AccessPermissions])
    .map(([, label]) => label)
  return labels.length > 0 ? labels : [t('access.noPermission')]
}

function formatExpiresAt(value?: string): string {
  if (!value) return t('access.permanent')
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}
</script>

<style scoped lang="scss">
.permission-page {
  min-height: 100vh;
  padding: 18px 22px 24px;
  background: var(--el-bg-color-page);
  color: var(--el-text-color-primary);
}

.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin: 0 auto 16px;
  max-width: 1440px;
}

.page-title-area {
  min-width: 0;

  h1 {
    margin: 4px 0 0;
    font-size: 24px;
    line-height: 1.25;
    letter-spacing: 0;
  }

  p {
    max-width: min(860px, calc(100vw - 380px));
    margin: 6px 0 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    word-break: break-all;
  }
}

.back-button {
  padding-left: 0;
  margin-bottom: 4px;
}

.page-kicker {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.page-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: 0 0 auto;
}

.page-alert,
.access-page-body {
  max-width: 1440px;
  margin: 0 auto;
}

.access-page-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 0;
}

.grant-flow {
  display: grid;
  grid-template-columns: minmax(280px, 0.9fr) minmax(360px, 1.1fr) minmax(300px, 0.95fr);
  gap: 14px;
  min-height: 0;
  align-items: start;
}

.resource-panel,
.role-panel,
.grant-panel,
.members-panel {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.resource-panel,
.role-panel,
.grant-panel {
  max-height: min(690px, calc(100vh - 132px));
  min-height: 0;
  padding: 14px;
}

.resource-panel {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.role-panel,
.grant-panel {
  overflow-y: auto;
}

.panel-header,
.members-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;

  h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  p {
    margin: 4px 0 0;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}

.tree-container {
  flex: 1 1 auto;
  min-height: 0;
  height: min(590px, calc(100vh - 216px));
  overflow-y: auto;
  padding-bottom: 8px;
}

.resource-tree {
  :deep(.el-tree-node__content) {
    height: 32px;
    padding: 0 8px;
    display: flex;
    align-items: center;

    &:hover {
      background-color: var(--el-fill-color-light);
    }
  }

  :deep(.el-tree-node__content:hover) {
    background-color: var(--el-fill-color-light);
  }

  :deep(.el-tree-node__expand-icon) {
    padding: 6px;
    color: var(--el-text-color-secondary);
    border-radius: 2px;
    cursor: pointer;
  }

  :deep(.el-tree-node__expand-icon:hover) {
    background-color: var(--el-fill-color);
  }

  :deep(.el-tree-node.is-expanded > .el-tree-node__content .el-tree-node__expand-icon) {
    transform: rotate(90deg);
  }

  :deep(.el-tree-node__expand-icon.is-leaf) {
    color: transparent;
  }

  :deep(.el-tree-node.is-current > .el-tree-node__content) {
    background-color: rgba(var(--el-color-primary-rgb), 0.12) !important;
    border-left: 2px solid var(--el-color-primary);
  }

  :deep(.el-tree-node.is-current .el-tree-node__children .el-tree-node__content) {
    background-color: transparent;
    border-left: none;
  }
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  width: 100%;
  min-width: 0;

  .node-icon {
    width: 16px;
    height: 16px;
    margin-right: 8px;
    color: #6366f1;
    opacity: 0.86;
    flex-shrink: 0;
  }

  .package-icon-img,
  .form-icon-img,
  .docs-icon-img {
    object-fit: contain;
  }

  .table-icon {
    color: #10b981;
  }

  .form-icon {
    color: #3b82f6;
  }

  .chart-icon {
    color: #f59e0b;
  }

  .docs-icon {
    color: #9b42f8;
  }

  .fx-icon {
    font-size: 12px;
    font-weight: 600;
    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
    font-style: italic;
  }

  .node-label {
    font-size: 14px;
    color: var(--el-text-color-primary);
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.resource-summary {
  padding: 10px 12px;
  margin-bottom: 12px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.resource-title {
  font-size: 14px;
  font-weight: 700;
}

.resource-path {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}

.role-cards {
  display: grid;
  gap: 10px;
}

.role-card {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
  padding: 12px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.14s ease, background-color 0.14s ease, box-shadow 0.14s ease;
  overflow: hidden;

  &::before {
    content: '';
    position: absolute;
    inset: 0 auto 0 0;
    width: 4px;
    background: transparent;
    transition: background-color 0.16s ease;
  }

  &:hover {
    border-color: rgba(var(--el-color-primary-rgb), 0.5);
    background: color-mix(in srgb, var(--el-bg-color) 94%, var(--el-color-primary) 6%);
  }

  &.is-selected {
    border-color: var(--el-color-primary);
    background: color-mix(in srgb, var(--el-bg-color) 90%, var(--el-color-primary) 10%);
    box-shadow: inset 0 0 0 1px rgba(var(--el-color-primary-rgb), 0.12);

    &::before {
      background: var(--el-color-primary);
    }

    .role-icon-badge {
      background: rgba(var(--el-color-primary-rgb), 0.16);
      color: var(--el-color-primary);
    }

    .role-card-copy strong {
      color: var(--el-color-primary);
      font-weight: 700;
    }

    .role-action-item {
      border-color: rgba(var(--el-color-primary-rgb), 0.18);
      background: rgba(var(--el-color-primary-rgb), 0.08);
      color: var(--el-color-primary);
    }
  }
}

.role-card-head {
  display: flex;
  align-items: center;
  gap: 10px;
}

.role-icon-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: rgba(var(--el-color-primary-rgb), 0.1);
  color: var(--el-color-primary);
  transition: background-color 0.16s ease, color 0.16s ease, box-shadow 0.16s ease;
}

.role-card-copy {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;

  strong {
    font-size: 14px;
  }

  em {
    font-style: normal;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
}

.role-state {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 999px;
  border: 1px solid var(--el-border-color);
  background: transparent;
  color: var(--el-text-color-placeholder);
  flex: 0 0 22px;
  transition: all 0.16s ease;
}

.role-state.is-selected {
  border-color: var(--el-color-primary);
  background: rgba(var(--el-color-primary-rgb), 0.1);
  color: var(--el-color-primary);
}

.role-empty-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: var(--el-border-color);
}

.role-description {
  font-size: 12px;
  line-height: 1.55;
  color: var(--el-text-color-secondary);
}

.role-action-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.role-action-item {
  padding: 3px 7px;
  border-radius: 999px;
  border: 1px solid transparent;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 12px;
}

.grant-form {
  :deep(.el-form-item) {
    margin-bottom: 14px;
  }
}

.principal-selector {
  width: 100%;
}

.all-members-alert {
  margin-top: 10px;
}

.expires-picker {
  width: 100%;
  margin-top: 10px;
}

.grant-preview {
  display: grid;
  gap: 8px;
  padding: 12px;
  margin: 2px 0 14px;
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.preview-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 13px;

  span {
    color: var(--el-text-color-secondary);
  }
}

.submit-button {
  width: 100%;
}

.members-panel {
  padding: 14px;
}

.members-header p {
  max-width: min(900px, calc(100vw - 260px));
  word-break: break-all;
}

.access-tabs {
  :deep(.el-tabs__header) {
    margin: 0 0 4px;
  }
}

.access-table {
  width: 100%;
}

.member-users-cell {
  display: inline-flex;
  align-items: center;
  min-width: 0;

  :deep(.users-response) {
    width: auto;
  }

  :deep(.users-list-horizontal) {
    gap: 6px;
  }

  :deep(.user-display-card-trigger) {
    max-width: 180px;
  }

  :deep(.user-name) {
    max-width: 128px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.permission-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.source-current {
  color: var(--el-text-color-primary);
}

.source-inherited {
  color: var(--el-text-color-secondary);
  word-break: break-all;
}

.disabled-action {
  color: var(--el-text-color-placeholder);
  cursor: not-allowed;
}

@media (max-width: 1180px) {
  .page-header {
    align-items: stretch;
    flex-direction: column;
  }

  .page-title-area p {
    max-width: 100%;
  }

  .grant-flow {
    grid-template-columns: 1fr;
  }

  .resource-panel,
  .role-panel,
  .grant-panel {
    max-height: none;
  }

  .tree-container {
    height: min(460px, 52vh);
  }
}

@media (max-width: 640px) {
  .permission-page {
    padding: 14px 12px 18px;
  }

  .page-actions {
    width: 100%;

    .el-button {
      flex: 1;
    }
  }
}

.permission-page {
  box-sizing: border-box;
  height: 100vh;
  padding: 12px 10px;
  overflow: hidden;
}

.access-shell {
  width: 100%;
  max-width: none;
  height: 100%;
  margin: 0;
}

.apply-layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr) 300px;
  gap: 12px;
  height: 100%;
  min-height: 0;
  align-items: stretch;
  overflow: hidden;
}

.apply-layout.is-request-mode {
  grid-template-columns: 360px minmax(0, 1fr) 320px;
}

.access-request-layout {
  height: 100%;
  min-height: 0;
  display: grid;
  place-items: center;
  padding: 24px;
}

.access-request-card {
  width: min(620px, 100%);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 42px 36px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
  text-align: center;
  box-shadow: var(--el-box-shadow-light);
}

.access-request-icon {
  width: 68px;
  height: 68px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 18px;
  border-radius: 8px;
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
  font-size: 32px;
}

.access-request-kicker {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 700;
}

.access-request-card h2 {
  margin: 10px 0 0;
  color: var(--el-text-color-primary);
  font-size: 24px;
  line-height: 1.3;
  letter-spacing: 0;
}

.access-request-card p {
  max-width: 520px;
  margin: 12px 0 0;
  color: var(--el-text-color-regular);
  font-size: 14px;
  line-height: 1.7;
}

.access-request-resource {
  width: 100%;
  margin-top: 22px;
  padding: 12px 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
  text-align: left;
}

.access-request-resource span {
  display: block;
  margin-bottom: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 700;
}

.access-request-resource code {
  display: block;
  color: var(--el-text-color-primary);
  font-size: 13px;
  line-height: 1.5;
  white-space: normal;
  overflow-wrap: anywhere;
}

.access-request-note {
  color: var(--el-text-color-secondary) !important;
}

.access-request-compact-form {
  width: 100%;
  display: grid;
  gap: 12px;
  margin-top: 16px;
  text-align: left;
}

.access-request-compact-form .access-request-note {
  margin: 0;
  font-size: 12px;
}

.access-request-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
  margin-top: 24px;
}

.apply-sidebar,
.apply-sidebar-right,
.apply-main {
  min-height: 0;
  height: 100%;
}

.tree-card,
.form-card {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-bg-color);
  overflow: hidden;
}

.apply-main {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  max-height: 100%;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding-right: 4px;
  scrollbar-gutter: stable;
}

.panel-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  padding: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-lighter);
}

.panel-title-copy {
  min-width: 0;

  h3 {
    margin: 0;
    font-size: 15px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  p {
    margin: 5px 0 0;
    font-size: 12px;
    line-height: 1.5;
    color: var(--el-text-color-secondary);
  }
}

.resource-tools {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.tree-container {
  height: auto;
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  padding: 12px 10px 18px;
}

.resource-tree {
  :deep(.el-tree-node__content) {
    height: 34px;
    padding: 0 8px;
    border-radius: 6px;
  }

  :deep(.el-tree-node.is-current > .el-tree-node__content) {
    border-left: none;
    box-shadow: inset 2px 0 0 var(--el-color-primary);
  }
}

.scope-header-main {
  padding: 0 0 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.scope-title-main {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.scope-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
}

.scope-icon-img,
.scope-icon-component,
.role-selection-resource-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.scope-name-main {
  min-width: 0;
  max-width: 100%;
  color: var(--el-text-color-primary);
  font-size: 18px;
  font-weight: 700;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scope-path-main {
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  word-break: break-all;

  code {
    padding: 2px 6px;
    border-radius: 4px;
    background: var(--el-fill-color-lighter);
    color: var(--el-text-color-secondary);
  }
}

.role-selection-section {
  padding: 14px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  overflow: visible;
}

.role-selection-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.role-selection-copy {
  min-width: 0;
}

.role-selection-kicker {
  display: inline-block;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.role-selection-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.role-selection-meta {
  margin-top: 10px;
}

.role-selection-resource-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 7px 12px;
  border-radius: 999px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  font-size: 12px;
  color: var(--el-text-color-primary);
}

.role-selection-desc {
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 1.55;
  color: var(--el-text-color-secondary);
}

.role-selection-hint {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
  padding: 8px 12px;
  border-radius: 999px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
}

.role-selection-hint.is-selected-role {
  border-color: rgba(var(--el-color-primary-rgb), 0.5);
  background: rgba(var(--el-color-primary-rgb), 0.12);
  color: var(--el-color-primary);
  font-weight: 800;
  box-shadow: 0 0 0 3px rgba(var(--el-color-primary-rgb), 0.07);
}

.role-selection-hint.is-selected-role.tone-admin {
  border-color: rgba(217, 119, 6, 0.48);
  background: rgba(245, 158, 11, 0.14);
  color: #b45309;
  box-shadow: 0 0 0 3px rgba(245, 158, 11, 0.08);
}

.role-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding-bottom: 10px;
}

.role-card {
  display: grid;
  grid-template-columns: 150px minmax(0, 1fr);
  align-items: stretch;
  gap: 12px;
  min-height: 0;
  padding: 10px;
  border-radius: 8px;
  background: var(--el-bg-color);
  transition: border-color 0.18s ease, background-color 0.18s ease, box-shadow 0.18s ease, transform 0.18s ease;
}

.role-card:hover,
.role-card:focus-visible {
  border-color: color-mix(in srgb, var(--el-color-primary) 35%, var(--el-border-color) 65%);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08);
  transform: translateY(-1px);
  outline: none;
}

.role-card.is-selected {
  padding: 9px;
  border: 2px solid var(--el-color-primary);
  background: color-mix(in srgb, var(--el-bg-color) 88%, var(--el-color-primary) 12%);
  box-shadow: 0 12px 28px rgba(var(--el-color-primary-rgb), 0.16), inset 0 0 0 1px rgba(var(--el-color-primary-rgb), 0.1);
}

.role-card.tone-admin.is-selected {
  border-color: #d97706;
  background: color-mix(in srgb, var(--el-bg-color) 88%, #f59e0b 12%);
  box-shadow: 0 12px 28px rgba(217, 119, 6, 0.16), inset 0 0 0 1px rgba(217, 119, 6, 0.1);
}

.role-card.is-unavailable {
  cursor: not-allowed;
  opacity: 0.58;
}

.role-card.is-unavailable:hover,
.role-card.is-unavailable:focus-visible {
  border-color: var(--el-border-color-light);
  box-shadow: none;
  transform: none;
}

.admin-request-warning {
  margin-top: 2px;
}

.role-card-aside {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
  padding: 2px 10px 2px 0;
  border-right: 1px solid var(--el-border-color-lighter);
}

.role-card-identity {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
}

.role-icon-badge {
  width: 32px;
  height: 32px;
  flex: 0 0 32px;
  font-size: 16px;
}

.role-icon-badge-view {
  color: #2563eb;
  background: rgba(37, 99, 235, 0.12);
}

.role-icon-badge-edit {
  color: #0f766e;
  background: rgba(15, 118, 110, 0.12);
}

.role-icon-badge-admin {
  color: #b45309;
  background: rgba(245, 158, 11, 0.14);
}

.role-icon-badge-owner {
  color: #be123c;
  background: rgba(244, 63, 94, 0.12);
}

.role-card-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.role-card-title-row strong {
  font-size: 15px;
  line-height: 1.25;
  color: var(--el-text-color-primary);
}

.role-card-subtitle,
.role-card-meta-text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.role-card-state {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  width: fit-content;
  padding: 4px 8px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  white-space: nowrap;
}

.role-card-state.is-selected {
  background: rgba(34, 197, 94, 0.12);
  color: #15803d;
}

.role-description {
  display: none;
}

.role-action-groups {
  align-self: stretch;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  min-width: 0;
}

.role-action-group {
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
  padding: 6px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: color-mix(in srgb, var(--el-bg-color) 94%, var(--el-fill-color) 6%);
}

.role-action-group-title {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 11px;
  font-weight: 700;

  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.role-action-group-icon {
  width: 13px;
  height: 13px;
  flex-shrink: 0;
  object-fit: contain;
}

.role-action-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(112px, 1fr));
  gap: 6px;
}

.role-action-group .role-action-grid {
  grid-template-columns: repeat(auto-fit, minmax(72px, 1fr));
  gap: 4px;
}

.role-action-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 7px 8px;
  border-radius: 8px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  font-size: 12px;
  font-weight: 600;
  min-width: 0;
}

.role-action-group .role-action-item {
  min-height: 26px;
  gap: 5px;
  padding: 4px 6px;
  border-radius: 7px;
  font-size: 11px;
}

.role-action-group .role-action-icon {
  font-size: 12px;
}

.role-action-group .role-action-copy span {
  overflow: visible;
  text-overflow: clip;
  white-space: normal;
  line-height: 1.25;
}

.role-action-item.is-enabled {
  border-color: rgba(34, 197, 94, 0.22);
  background: rgba(34, 197, 94, 0.08);
  color: #15803d;
}

.role-action-item.is-disabled {
  color: var(--el-text-color-secondary);
  background: color-mix(in srgb, var(--el-bg-color) 96%, var(--el-text-color-secondary) 4%);
}

.role-action-icon {
  flex-shrink: 0;
}

.role-action-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;

  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  small {
    color: var(--el-text-color-secondary);
    font-size: 11px;
    font-weight: 400;
    line-height: 1.35;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.role-action-group .role-action-copy small {
  display: none;
}

.role-card.is-selected .role-card-title-row strong {
  color: var(--el-text-color-primary);
}

.role-card.is-selected .role-icon-badge-view {
  color: #2563eb;
  background: rgba(37, 99, 235, 0.12);
}

.role-card.is-selected .role-icon-badge-edit {
  color: #0f766e;
  background: rgba(15, 118, 110, 0.12);
}

.role-card.is-selected .role-icon-badge-admin {
  color: #b45309;
  background: rgba(245, 158, 11, 0.14);
}

.role-card.is-selected .role-icon-badge-owner {
  color: #be123c;
  background: rgba(244, 63, 94, 0.12);
}

.role-card.is-selected .role-action-item.is-enabled {
  border-color: rgba(34, 197, 94, 0.24);
  background: rgba(34, 197, 94, 0.08);
  color: #15803d;
}

.role-card.is-selected .role-action-item.is-disabled {
  border-color: var(--el-border-color-lighter);
  background: color-mix(in srgb, var(--el-bg-color) 96%, var(--el-text-color-secondary) 4%);
  color: var(--el-text-color-secondary);
}

.grant-form {
  flex: 1 1 auto;
  min-height: 0;
  padding: 12px;
  overflow-y: auto;

  :deep(.el-form-item) {
    margin-bottom: 18px;
  }
}

.grant-preview {
  padding: 14px;
  margin: 2px 0 16px;
}

.submit-button,
.cancel-button {
  width: 100%;
}

.cancel-button {
  margin: 12px 0 0;
}

.permission-workflow {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  gap: 10px;
}

.permission-workflow > .apply-layout {
  flex: 1 1 auto;
  height: auto;
}

.workflow-tabs {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 4px;
  min-height: 42px;
  padding: 4px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 9px;
  background: var(--el-bg-color);
}

.workflow-tabs button {
  display: inline-flex;
  min-height: 32px;
  align-items: center;
  gap: 7px;
  padding: 0 13px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}

.workflow-tabs button:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}

.workflow-tabs button.is-active {
  background: rgba(var(--el-color-primary-rgb), 0.12);
  color: var(--el-color-primary);
  font-weight: 700;
}

.workflow-tab-badge {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 999px;
  background: var(--el-color-danger);
  color: #fff;
  font-size: 11px;
  line-height: 18px;
  text-align: center;
}

.resource-lock-icon {
  flex: 0 0 auto;
  color: var(--el-color-danger);
  font-size: 14px;
}

.resource-node-status {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 5px;
}

.resource-current-role {
  display: inline-flex;
  align-items: center;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 10px;
  font-weight: 700;
  line-height: 17px;
  white-space: nowrap;
}

.resource-pending-role {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 6px;
  border-radius: 999px;
  background: rgba(245, 158, 11, 0.14);
  color: #b45309;
  font-size: 10px;
  font-weight: 700;
  line-height: 17px;
  white-space: nowrap;
}

.resource-current-role.tone-member {
  background: rgba(15, 118, 110, 0.12);
  color: #0f766e;
}

.resource-current-role.tone-admin,
.resource-current-role.tone-owner {
  background: rgba(245, 158, 11, 0.14);
  color: #b45309;
}

.resource-current-role.tone-viewer {
  background: rgba(37, 99, 235, 0.1);
  color: #2563eb;
}

.resource-inheritance-state {
  display: inline-flex;
  max-width: 108px;
  align-items: center;
  gap: 3px;
  overflow: hidden;
  color: var(--el-color-primary);
  font-size: 11px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.resource-inheritance-state .el-icon {
  flex: 0 0 auto;
}

.resource-tree .tree-node.is-selected .node-label {
  color: var(--el-color-primary);
  font-weight: 700;
}

.approver-list {
  width: 100%;
  min-height: 52px;
  display: grid;
  gap: 7px;
}

.approver-item {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.approver-item small {
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.request-records-card {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  padding: 18px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 9px;
  background: var(--el-bg-color);
}

.request-records-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.request-records-header h2 {
  margin: 0;
  font-size: 18px;
}

.request-records-header p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.request-records-table {
  width: 100%;
}

.request-table-path {
  color: var(--el-text-color-regular);
  overflow-wrap: anywhere;
}

.review-result-cell {
  display: grid;
  gap: 5px;
}

.review-result-cell small {
  color: var(--el-text-color-secondary);
}

.members-dialog-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 10px;

  h3 {
    margin: 0;
    font-size: 15px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  p {
    margin: 5px 0 0;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    word-break: break-all;
  }
}

@media (max-width: 1500px) {
  .apply-layout {
    grid-template-columns: 260px minmax(0, 1fr) 280px;
    gap: 10px;
  }

  .apply-layout.is-request-mode {
    grid-template-columns: 330px minmax(0, 1fr) 300px;
  }
}

@media (max-width: 1180px) {
  .permission-page {
    height: auto;
    min-height: 100vh;
    overflow: auto;
  }

  .access-shell {
    height: auto;
  }

  .apply-layout {
    grid-template-columns: 1fr;
    height: auto;
  }

  .apply-layout.is-request-mode {
    grid-template-columns: 1fr;
  }

  .tree-card,
  .form-card {
    height: auto;
    max-height: none;
  }

  .role-selection-header {
    flex-direction: column;
  }

  .role-card {
    grid-template-columns: 1fr;
  }

  .role-card-aside {
    flex-direction: row;
    align-items: center;
    border-right: 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
    padding: 0 0 8px;
  }

  .role-card-identity {
    flex-direction: row;
    align-items: center;
  }

  .role-action-groups {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .resource-tools {
    align-items: flex-end;
    flex-direction: column;
  }

  .role-action-groups,
  .role-action-grid {
    grid-template-columns: 1fr;
  }
}
</style>
