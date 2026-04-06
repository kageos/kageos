<!--
  角色管理页面

  目标：
  - 让用户先理解“角色是权限模板”，再去编辑或分配
  - 把角色模板库、权限编辑、角色分配三件事拆开表达
  - 保留现有 API 和权限语义，不改变后端交互
-->

<template>
  <div class="role-management-page">
    <section class="page-header">
      <div class="page-header-copy">
        <h1>角色管理</h1>
      </div>

      <div class="page-header-actions">
        <el-button type="primary" :icon="Plus" @click="handleCreateRole">新建角色</el-button>
        <el-button :icon="Refresh" @click="loadRoles">刷新</el-button>
      </div>
    </section>

    <section class="page-body">
      <aside class="tree-column">
        <section class="panel-card tree-panel-card">
          <div class="tree-panel-header">
            <div class="panel-copy">
              <h2>角色目录</h2>
              <p>左边按服务目录结构选资源，右边上方切角色，下面直接看这个角色会影响哪些节点。</p>
            </div>
            <el-button type="primary" :icon="Plus" @click="handleCreateRoleForResourceType(selectedTreeResourceType)">
              新建
            </el-button>
          </div>

          <el-input
            v-model="roleTreeKeyword"
            class="role-tree-search"
            placeholder="搜索目录节点…"
            clearable
            :prefix-icon="Search"
          />

          <el-tree
            ref="roleTreeRef"
            v-loading="loading"
            :data="roleTreeData"
            node-key="id"
            :default-expand-all="true"
            :expand-on-click-node="false"
            :highlight-current="true"
            :current-node-key="selectedRoleTreeKey"
            :filter-node-method="filterRoleTreeNode"
            class="role-tree"
            @node-click="handleRoleTreeNodeClick"
          >
            <template #default="{ data }">
              <div class="role-tree-node" :class="`role-tree-node-${data.kind}`">
                <div class="role-tree-node-main">
                  <img
                    v-if="isImageResourceIcon(data.resourceType)"
                    :src="getResourceTypeIconSrc(data.resourceType)"
                    :alt="getResourceTypeIconAlt(data.resourceType)"
                    class="node-icon"
                    :class="getResourceTypeIconClass(data.resourceType)"
                  />
                  <component
                    :is="getResourceTypeIconComponent(data.resourceType)"
                    v-else
                    class="node-icon"
                    :class="getResourceTypeIconClass(data.resourceType)"
                  />
                  <span class="role-tree-label">{{ data.label }}</span>
                </div>

                <div class="role-tree-node-side">
                  <el-tag size="small" effect="plain">{{ data.roleCount }} 个角色</el-tag>
                </div>
              </div>
            </template>
          </el-tree>
        </section>
      </aside>

      <section class="detail-column">
        <section class="panel-card detail-panel-card">
          <div v-if="selectedResourceRoles.length > 0" class="role-switcher-shell">
            <div class="role-switcher-header">
              <div>
                <span class="section-label">角色切换</span>
                <h3>点击切换当前预览角色</h3>
              </div>
              <span class="role-switcher-hint">当前选中的角色会直接影响下面的权限树</span>
            </div>

            <div class="role-switcher">
              <button
                v-for="role in selectedResourceRoles"
                :key="role.id"
                type="button"
                class="role-switcher-item"
                :class="[`role-switcher-item-${getRoleVisual(role).tone}`, { 'is-active': focusedRole?.id === role.id }]"
                @click="handleSelectRoleFromDetail(role)"
              >
                <div class="role-switcher-item-head">
                  <span class="role-icon-badge" :class="`role-icon-badge-${getRoleVisual(role).tone}`">
                    <el-icon>
                      <component :is="getRoleVisual(role).icon" />
                    </el-icon>
                  </span>
                  <div class="role-switcher-item-copy">
                    <strong>{{ role.name }}</strong>
                    <span>{{ getRoleVisual(role).label }}</span>
                  </div>
                </div>

                <div class="role-switcher-item-footer">
                  <span class="role-switcher-item-meta">{{ getRolePermissionCount(role) }} 个权限点</span>
                  <span class="role-switcher-state" :class="{ 'is-active': focusedRole?.id === role.id }">
                    <el-icon v-if="focusedRole?.id === role.id"><CircleCheck /></el-icon>
                    {{ focusedRole?.id === role.id ? '当前选中' : '点击切换' }}
                  </span>
                </div>

                <div v-if="focusedRole?.id === role.id" class="role-switcher-item-expanded">
                  <p>{{ role.description || getRoleFallbackDescription(role) }}</p>
                  <div class="role-switcher-item-actions">
                    <el-button size="small" type="primary" @click.stop="handleEditRole(role)">编辑模板</el-button>
                    <el-button size="small" text @click.stop="handleAssignRole(role)">分配</el-button>
                    <el-button
                      v-if="!role.is_system"
                      size="small"
                      text
                      type="danger"
                      @click.stop="handleDeleteRole(role)"
                    >
                      删除
                    </el-button>
                  </div>
                </div>
              </button>
            </div>
          </div>

          <div class="detail-header">
            <div class="detail-header-main">
              <div>
                <span class="section-label">当前资源类型</span>
                <h2>{{ getResourceTypeLabel(selectedTreeResourceType) }}</h2>
                <p>{{ getResourceTypeDescription(selectedTreeResourceType) }}</p>
              </div>
            </div>
            <div class="detail-header-actions">
              <el-button type="primary" plain :icon="Plus" @click="handleCreateRoleForResourceType(selectedTreeResourceType)">
                新建{{ getResourceTypeLabel(selectedTreeResourceType) }}角色
              </el-button>
            </div>
          </div>

          <div v-if="focusedRole" class="panel-card hierarchy-card">
            <div class="hierarchy-header">
              <div>
                <h3>角色继承树</h3>
                <p>按服务目录结构看这个角色会影响哪些节点，以及每个节点拿到什么动作。</p>
              </div>
              <el-tag round effect="plain">
                {{ getResourceTypeLabel(focusedRolePrimaryResourceType) }}
              </el-tag>
            </div>

            <div class="hierarchy-tree-shell">
              <el-tree
                :data="focusedRoleTreeData"
                node-key="id"
                :expand-on-click-node="false"
                :default-expand-all="true"
                :indent="22"
                class="role-hierarchy-tree"
              >
                <template #default="{ data }">
                  <div class="role-hierarchy-node">
                    <div class="role-hierarchy-node-main">
                      <img
                        v-if="isImageResourceIcon(data.resourceType)"
                        :src="getResourceTypeIconSrc(data.resourceType)"
                        :alt="getResourceTypeIconAlt(data.resourceType)"
                        class="node-icon"
                        :class="getResourceTypeIconClass(data.resourceType)"
                      />
                      <component
                        :is="getResourceTypeIconComponent(data.resourceType)"
                        v-else
                        class="node-icon"
                        :class="getResourceTypeIconClass(data.resourceType)"
                      />

                      <div class="role-hierarchy-copy">
                        <div class="role-hierarchy-node-head">
                          <strong>{{ data.title }}</strong>
                          <el-tag v-if="data.status" size="small" :type="data.statusType" effect="plain">
                            {{ data.status }}
                          </el-tag>
                        </div>
                        <p>{{ data.description }}</p>
                      </div>
                    </div>
                    <div class="role-hierarchy-actions">
                      <div
                        v-for="actionState in getNodeActionStates(data)"
                        :key="`${data.id}-${actionState.value}`"
                        class="role-hierarchy-action-item"
                        :class="{
                          'is-enabled': actionState.enabled,
                          'is-disabled': !actionState.enabled,
                        }"
                      >
                        <el-icon class="role-hierarchy-action-icon">
                          <component :is="actionState.enabled ? CircleCheck : CircleClose" />
                        </el-icon>
                        <div class="role-hierarchy-action-copy">
                          <span>{{ actionState.label }}</span>
                          <small v-if="actionState.description">{{ actionState.description }}</small>
                        </div>
                      </div>
                    </div>
                  </div>
                </template>
              </el-tree>
            </div>
          </div>

          <div v-else class="hierarchy-empty">
            <el-empty :description="`${getResourceTypeLabel(selectedTreeResourceType)}下还没有角色，先创建一个`" :image-size="84">
              <el-button type="primary" :icon="Plus" @click="handleCreateRoleForResourceType(selectedTreeResourceType)">
                创建{{ getResourceTypeLabel(selectedTreeResourceType) }}角色
              </el-button>
            </el-empty>
          </div>
        </section>

        <section class="panel-card hierarchy-card">
          <div class="hierarchy-header">
            <div>
              <h3>使用方式</h3>
              <p>左边按资源类型找角色，右边统一看权限和继承逻辑，不再靠卡片猜当前选中了谁。</p>
            </div>
          </div>

          <div class="usage-list">
            <article class="usage-item">
              <strong>1. 点左侧资源类型</strong>
              <span>比如目录、表格、文档，右边马上切到这类角色。</span>
            </article>
            <article class="usage-item">
              <strong>2. 右侧切角色</strong>
              <span>查看者、开发者、管理员在右边切，不需要在左边一堆卡片里找选中态。</span>
            </article>
            <article class="usage-item">
              <strong>3. 直接看树</strong>
              <span>目录角色会拆成目录本身和子资源继承；非目录角色就只显示自身作用范围。</span>
            </article>
          </div>
        </section>
      </section>
    </section>

    <el-dialog
      v-model="roleDialogVisible"
      class="role-dialog"
      width="920px"
      :close-on-click-modal="false"
    >
      <template #header>
        <div class="dialog-header">
          <span class="dialog-kicker">Role Template</span>
          <h3>{{ roleDialogTitle }}</h3>
          <p>{{ roleDialogDescription }}</p>
        </div>
      </template>

      <div class="role-dialog-body">
        <section class="dialog-card">
          <div class="section-heading">
            <div>
              <h4>基础信息</h4>
              <p>先定义这个角色主要服务什么资源，再补充名称和说明。</p>
            </div>
            <el-tag type="primary" round>{{ currentRolePrimaryResourceLabel }}</el-tag>
          </div>

          <div class="type-selector-block">
            <span class="field-label">适用资源</span>
            <p class="field-help">角色会优先出现在对应资源的权限申请中。</p>

            <div v-if="roleForm.id" class="locked-type">
              <span>已有角色的适用资源类型暂不支持修改。</span>
            </div>

            <el-radio-group
              v-else
              v-model="roleFormPrimaryResourceType"
              class="type-choice-grid"
            >
              <el-radio-button
                v-for="resourceType in resourceTypes"
                :key="resourceType"
                :label="resourceType"
              >
                {{ getResourceTypeLabel(resourceType) }}
              </el-radio-button>
            </el-radio-group>
          </div>

          <el-form
            ref="roleFormRef"
            :model="roleForm"
            :rules="roleFormRules"
            label-width="96px"
            class="role-form"
          >
            <el-form-item label="角色名称" prop="name">
              <el-input v-model="roleForm.name" placeholder="例如：目录开发者、表格编辑员" />
            </el-form-item>
            <el-form-item label="角色代码" prop="code">
              <el-input
                v-model="roleForm.code"
                placeholder="请输入稳定的英文代码，例如：directory_editor"
                :disabled="!!roleForm.id"
              />
            </el-form-item>
            <el-form-item label="角色描述">
              <el-input
                v-model="roleForm.description"
                type="textarea"
                :rows="3"
                placeholder="用一句话说明这个角色适合谁、能做什么"
              />
            </el-form-item>
            <el-form-item v-if="roleForm.id" label="默认推荐">
              <div class="default-switch-row">
                <el-switch v-model="roleForm.is_default" active-text="是" inactive-text="否" />
                <span class="field-help">设为默认角色后，在权限申请时会优先推荐。</span>
              </div>
            </el-form-item>
          </el-form>
        </section>

        <section class="dialog-card">
          <div class="section-heading">
            <div>
              <h4>权限编辑</h4>
              <p>左边按服务目录选节点，右边只编辑当前节点的动作。目录角色可以决定子资源是否一起继承。</p>
            </div>

            <div class="permission-heading-side">
              <span class="permission-count-pill">{{ roleSelectedPermissionCount }} 个权限点</span>
            </div>
          </div>

          <el-form :model="roleForm" label-width="0" class="permission-form-shell">
            <el-form-item prop="permissions">
              <div class="role-editor-layout">
                <aside class="role-editor-sidebar">
                  <div v-if="isDirectoryRole" class="role-editor-sidebar-toggle">
                    <div>
                      <strong>目录子资源继承</strong>
                      <p>打开后，目录下面的表格、表单、图表、文档、讨论区都可以单独配置动作。</p>
                    </div>
                    <el-switch v-model="roleFormCrossResourceEnabled" />
                  </div>

                  <el-tree
                    ref="roleEditorTreeRef"
                    :data="roleEditorTreeData"
                    node-key="id"
                    :default-expand-all="true"
                    :expand-on-click-node="false"
                    :highlight-current="true"
                    :current-node-key="selectedRoleEditorNodeKey"
                    class="role-editor-tree"
                    @node-click="handleRoleEditorNodeClick"
                  >
                    <template #default="{ data }">
                      <div class="role-editor-node" :class="{ 'is-disabled': data.disabled }">
                        <div class="role-editor-node-main">
                          <img
                            v-if="isImageResourceIcon(data.resourceType)"
                            :src="getResourceTypeIconSrc(data.resourceType)"
                            :alt="getResourceTypeIconAlt(data.resourceType)"
                            class="node-icon"
                            :class="getResourceTypeIconClass(data.resourceType)"
                          />
                          <component
                            :is="getResourceTypeIconComponent(data.resourceType)"
                            v-else
                            class="node-icon"
                            :class="getResourceTypeIconClass(data.resourceType)"
                          />
                          <span class="role-editor-node-label">{{ data.label }}</span>
                        </div>
                        <el-tag size="small" effect="plain" :type="data.statusType">
                          {{ data.status }}
                        </el-tag>
                      </div>
                    </template>
                  </el-tree>
                </aside>

                <section class="role-editor-detail">
                  <div class="role-editor-detail-header">
                    <div>
                      <span class="section-label">当前编辑节点</span>
                      <h4>{{ getResourceTypeLabel(selectedRoleEditorResourceType) }}</h4>
                      <p>{{ currentRoleEditorDescription }}</p>
                    </div>
                    <div class="role-editor-detail-actions">
                      <span class="permission-count-pill">
                        {{ currentRoleEditorEnabledActionCount }}/{{ currentRoleEditorActionOptions.length }} 已开启
                      </span>
                      <el-button
                        link
                        type="primary"
                        :disabled="currentRoleEditorDisabled"
                        @click="setCurrentEditorActions(true)"
                      >
                        全开
                      </el-button>
                      <el-button
                        link
                        type="primary"
                        :disabled="currentRoleEditorDisabled"
                        @click="setCurrentEditorActions(false)"
                      >
                        清空
                      </el-button>
                    </div>
                  </div>

                  <div v-if="currentRoleEditorDisabled" class="role-editor-disabled">
                    <el-icon><CircleClose /></el-icon>
                    <div>
                      <strong>当前节点暂未开启编辑</strong>
                      <p>先打开“目录子资源继承”，再为这个子资源配置动作。</p>
                    </div>
                  </div>

                  <div v-else class="role-editor-action-list">
                    <article
                      v-for="action in currentRoleEditorActionOptions"
                      :key="action.value"
                      class="role-editor-action-card"
                      :class="{ 'is-enabled': hasRoleFormAction(selectedRoleEditorResourceType, action.value) }"
                    >
                      <div class="role-editor-action-copy">
                        <strong>{{ action.label }}</strong>
                        <p v-if="action.description" class="role-editor-action-description">{{ action.description }}</p>
                        <span>{{ action.value }}</span>
                      </div>
                      <el-switch
                        :model-value="hasRoleFormAction(selectedRoleEditorResourceType, action.value)"
                        @change="setRoleFormAction(selectedRoleEditorResourceType, action.value, $event)"
                      />
                    </article>
                  </div>
                </section>
              </div>
            </el-form-item>
          </el-form>
        </section>
      </div>

      <template #footer>
        <div class="dialog-footer-actions">
          <el-button @click="roleDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="roleSubmitting" @click="handleSubmitRole">
            {{ roleForm.id ? '保存角色' : '创建角色' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog
      v-model="assignDialogVisible"
      class="assign-dialog"
      width="720px"
      :close-on-click-modal="false"
    >
      <template #header>
        <div class="dialog-header">
          <span class="dialog-kicker">Role Assignment</span>
          <h3>分配角色</h3>
          <p>{{ assignDialogDescription }}</p>
        </div>
      </template>

      <div v-if="currentAssignRole" class="assignment-role-summary">
        <div>
          <span class="summary-label">当前角色</span>
          <strong>{{ currentAssignRole.name }}</strong>
        </div>
        <el-tag round>{{ getResourceTypeLabel(currentAssignRole.resource_type || 'directory') }}</el-tag>
      </div>

      <el-form
        ref="assignFormRef"
        :model="assignForm"
        :rules="assignFormRules"
        label-width="110px"
        class="assign-form"
      >
        <el-form-item label="赋给谁" prop="subject_type">
          <el-radio-group v-model="assignForm.subject_type">
            <el-radio label="user">用户</el-radio>
            <el-radio label="department">组织架构</el-radio>
          </el-radio-group>
        </el-form-item>

        <el-form-item
          v-if="assignForm.subject_type === 'user'"
          label="目标用户"
          prop="username"
        >
          <UserWidget
            :field="assignUserField"
            :value="assignUserFieldValue"
            :field-path="assignUserField.code"
            mode="edit"
            @update:modelValue="handleAssignUserChange"
          />
        </el-form-item>

        <el-form-item
          v-if="assignForm.subject_type === 'department'"
          label="目标组织"
          prop="department_path"
        >
          <DepartmentSelector
            v-model="assignForm.department_path"
            placeholder="请选择组织架构"
            style="width: 100%"
          />
        </el-form-item>

        <div class="form-section-divider">
          <span>作用范围</span>
        </div>

        <el-form-item label="工作空间用户" prop="user">
          <el-input
            v-model="assignForm.user"
            placeholder="例如：beiluo"
          />
        </el-form-item>

        <el-form-item label="工作空间代码" prop="app">
          <el-input
            v-model="assignForm.app"
            placeholder="例如：crm"
          />
        </el-form-item>

        <el-form-item label="覆盖范围">
          <div class="scope-mode-field">
            <el-radio-group v-model="assignScopeMode">
              <el-radio-button label="workspace">仅当前工作空间</el-radio-button>
              <el-radio-button label="workspace_children">包含子资源</el-radio-button>
              <el-radio-button label="custom">自定义路径</el-radio-button>
            </el-radio-group>
            <div class="path-preview">
              <span class="summary-label">当前资源路径</span>
              <code>{{ assignForm.resource_path || '请先填写工作空间信息' }}</code>
            </div>
          </div>
        </el-form-item>

        <el-form-item v-if="assignScopeMode === 'custom'" label="自定义路径" prop="resource_path">
          <el-input
            v-model="assignForm.resource_path"
            :placeholder="assignCustomPathPlaceholder"
          />
          <div class="path-tip">
            你可以填具体文档、目录或通配路径，例如：{{ assignCustomPathPlaceholder }}
          </div>
        </el-form-item>

        <el-form-item v-else label="生效路径">
          <el-input :model-value="assignForm.resource_path" readonly />
        </el-form-item>

        <el-form-item label="有效期">
          <el-checkbox v-model="assignForm.isPermanent">永久有效</el-checkbox>
        </el-form-item>

        <el-form-item
          v-if="!assignForm.isPermanent"
          label="开始时间"
          prop="start_time"
        >
          <el-date-picker
            v-model="assignForm.start_time"
            type="datetime"
            placeholder="选择开始时间"
            style="width: 100%"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
          />
        </el-form-item>

        <el-form-item
          v-if="!assignForm.isPermanent"
          label="结束时间"
          prop="end_time"
        >
          <el-date-picker
            v-model="assignForm.end_time"
            type="datetime"
            placeholder="选择结束时间"
            style="width: 100%"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer-actions">
          <el-button @click="assignDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="assignSubmitting" @click="handleSubmitAssign">
            确认分配
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules, type TreeInstance } from 'element-plus'
import { CircleCheck, CircleClose, EditPen, Key, Plus, Refresh, Search, User, View } from '@element-plus/icons-vue'
import {
  getRoles,
  getRole,
  createRole,
  updateRole,
  deleteRole,
  assignRoleToUser,
  assignRoleToDepartment,
  type Role,
  type CreateRoleReq,
  type UpdateRoleReq,
  type AssignRoleToUserReq,
  type AssignRoleToDepartmentReq,
} from '@/api/role'
import UserWidget from '@/shared/components/UserWidget.vue'
import DepartmentSelector from '@/shared/components/DepartmentSelector.vue'
import ChartIcon from '@/shared/components/icons/ChartIcon.vue'
import TableIcon from '@/shared/components/icons/TableIcon.vue'
import type { FieldValue } from '@/core/types/field'
import { WidgetType } from '@/core/constants/widget'
import { buildAppResourcePath, parseResourcePath } from '@/utils/resourcePath'
import { createStringFieldValue, createWidgetFieldConfig, extractStringFieldRaw } from '@/utils/widgetFieldHelpers'

type ResourceType = 'directory' | 'table' | 'form' | 'chart' | 'docs' | 'board' | 'app'
type AssignScopeMode = 'workspace' | 'workspace_children' | 'custom'
type RoleHierarchyNode = {
  id: string
  title: string
  description: string
  resourceType: ResourceType
  status?: string
  statusType?: 'primary' | 'success' | 'info' | 'warning'
  actions: string[]
  children?: RoleHierarchyNode[]
}
type RoleVisualTone = 'view' | 'edit' | 'admin' | 'other'
type RoleTreeNavNode = {
  id: string
  kind: 'resource'
  label: string
  resourceType: ResourceType
  roleCount?: number
  children?: RoleTreeNavNode[]
}
type RoleEditorNode = {
  id: string
  label: string
  resourceType: ResourceType
  status: string
  statusType?: 'primary' | 'success' | 'info' | 'warning'
  disabled?: boolean
  children?: RoleEditorNode[]
}

const defaultPrimaryResourceType: ResourceType = 'directory'
const resourceTypes: ResourceType[] = ['directory', 'table', 'form', 'chart', 'docs', 'board', 'app']
const directoryInheritedResourceTypes: ResourceType[] = ['table', 'form', 'chart', 'docs', 'board']
const directoryRoleConfigResourceTypes: ResourceType[] = ['directory', ...directoryInheritedResourceTypes]

const resourceTypeLabels: Record<ResourceType, string> = {
  directory: '目录',
  table: '表格函数',
  form: '表单函数',
  chart: '图表函数',
  docs: '文档',
  board: '讨论区',
  app: '工作空间',
}

const resourceTypeDescriptions: Record<ResourceType, string> = {
  directory: '适合目录、服务树、包目录等结构化资源，也能向下给表格、表单、图表、文档、讨论区扩展动作。',
  table: '适合围绕表格记录的查看、编辑和删除动作。',
  form: '适合负责表单填写、提交和管理的人。',
  chart: '适合只读分析或图表维护场景。',
  docs: '适合知识库、文档和说明资料的协作。',
  board: '适合讨论区、帖子和社区互动场景。',
  app: '适合工作空间级别的整体访问和管理。',
}

const permissionConfig: Record<ResourceType, Array<{ value: string; label: string; description?: string }>> = {
  directory: [
    { value: 'directory:read', label: '查看目录' },
    { value: 'directory:write', label: '写入目录' },
    { value: 'directory:update', label: '更新目录' },
    { value: 'directory:delete', label: '删除目录' },
    { value: 'directory:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
  table: [
    { value: 'table:read', label: '查看表格' },
    { value: 'table:write', label: '新增记录' },
    { value: 'table:update', label: '更新记录' },
    { value: 'table:delete', label: '删除记录' },
    { value: 'table:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
  form: [
    { value: 'form:read', label: '查看表单' },
    { value: 'form:write', label: '提交表单' },
    { value: 'form:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
  chart: [
    { value: 'chart:read', label: '查看图表' },
    { value: 'chart:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
  docs: [
    { value: 'docs:read', label: '查看文档' },
    { value: 'docs:write', label: '编辑文档' },
    { value: 'docs:delete', label: '删除文档' },
    { value: 'docs:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
  board: [
    { value: 'board:read', label: '查看帖子' },
    { value: 'board:write', label: '发帖' },
    { value: 'board:update', label: '更新帖子' },
    { value: 'board:delete', label: '删除帖子' },
    { value: 'board:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
  app: [
    { value: 'app:read', label: '查看工作空间' },
    { value: 'app:create', label: '创建工作空间' },
    { value: 'app:update', label: '更新工作空间' },
    { value: 'app:delete', label: '删除工作空间' },
    { value: 'app:admin', label: '所有权', description: '可分配权限、完整管理、支持迭代。' },
  ],
}

const loading = ref(false)
const roleSubmitting = ref(false)
const assignSubmitting = ref(false)

const roleList = ref<Role[]>([])
const focusedRoleId = ref<number | null>(null)
const roleTreeRef = ref<TreeInstance>()
const roleTreeKeyword = ref('')
const selectedTreeResourceType = ref<ResourceType>(defaultPrimaryResourceType)
const selectedRoleTreeKey = ref(`resource-${defaultPrimaryResourceType}`)

const roleDialogVisible = ref(false)
const roleFormRef = ref<FormInstance>()
const roleEditorTreeRef = ref<TreeInstance>()
const roleFormPrimaryResourceType = ref<ResourceType>(defaultPrimaryResourceType)
const roleFormCrossResourceEnabled = ref(false)
const selectedRoleEditorResourceType = ref<ResourceType>(defaultPrimaryResourceType)
const selectedRoleEditorNodeKey = ref(getRoleTreeNodeId(defaultPrimaryResourceType))
const roleForm = reactive<{
  id?: number
  name: string
  code: string
  description: string
  is_default: boolean
  permissions: Record<string, string[]>
}>({
  name: '',
  code: '',
  description: '',
  is_default: false,
  permissions: {},
})

const assignDialogVisible = ref(false)
const assignFormRef = ref<FormInstance>()
const assignScopeMode = ref<AssignScopeMode>('workspace')
const currentAssignRole = ref<Role | null>(null)
const assignForm = reactive<{
  subject_type: 'user' | 'department'
  username: string
  department_path: string
  user: string
  app: string
  resource_path: string
  isPermanent: boolean
  start_time?: string
  end_time?: string
}>({
  subject_type: 'user',
  username: '',
  department_path: '',
  user: '',
  app: '',
  resource_path: '',
  isPermanent: true,
})

const assignUserField = createWidgetFieldConfig({
  code: 'assign_username',
  name: '目标用户',
  widgetType: WidgetType.USER
})

const assignUserFieldValue = computed(() => createStringFieldValue(assignUserField, assignForm.username))

const roleFormRules: FormRules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  code: [
    { required: true, message: '请输入角色代码', trigger: 'blur' },
    {
      pattern: /^[a-z][a-z0-9_]*$/,
      message: '角色代码只能包含小写字母、数字和下划线，且必须以字母开头',
      trigger: 'blur'
    },
  ],
  permissions: [
    {
      validator: (_rule, value, callback) => {
        const permissionGroups = (value ?? {}) as Record<string, string[]>
        const hasPermissions = hasConfiguredPermissions(permissionGroups)
        if (!hasPermissions) {
          callback(new Error('请至少配置一个权限'))
        } else {
          callback()
        }
      },
      trigger: 'change',
    },
  ],
}

const assignFormRules: FormRules = {
  username: [
    {
      validator: (_rule, value, callback) => {
        if (assignForm.subject_type === 'user' && !value) {
          callback(new Error('请选择目标用户'))
        } else {
          callback()
        }
      },
      trigger: 'change',
    },
  ],
  department_path: [
    {
      validator: (_rule, value, callback) => {
        if (assignForm.subject_type === 'department' && !value) {
          callback(new Error('请选择组织架构'))
        } else {
          callback()
        }
      },
      trigger: 'change',
    },
  ],
  user: [
    {
      validator: (_rule, value, callback) => {
        if (!String(value || '').trim()) {
          callback(new Error('请输入工作空间用户'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
  app: [
    {
      validator: (_rule, value, callback) => {
        if (!String(value || '').trim()) {
          callback(new Error('请输入工作空间代码'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
  resource_path: [{ required: true, message: '请补充资源路径', trigger: 'blur' }],
}

function createEmptyPermissions(): Record<string, string[]> {
  return Object.fromEntries(resourceTypes.map((resourceType) => [resourceType, []])) as Record<string, string[]>
}

function normalizePermissions(source?: Record<string, string[]>): Record<string, string[]> {
  const nextPermissions = createEmptyPermissions()
  for (const resourceType of resourceTypes) {
    const actions = source?.[resourceType]
    nextPermissions[resourceType] = Array.isArray(actions) ? [...actions] : []
  }
  return nextPermissions
}

function hasConfiguredPermissions(source?: Record<string, string[]>): boolean {
  return availableRoleConfigResourceTypes.value.some((resourceType) => {
    const actions = source?.[resourceType]
    return Array.isArray(actions) && actions.length > 0
  })
}

function getResourceTypeLabel(resourceType: string): string {
  return resourceTypeLabels[resourceType as ResourceType] || resourceType
}

function getResourceTypeDescription(resourceType: string): string {
  return resourceTypeDescriptions[resourceType as ResourceType] || '用于当前资源的权限模板。'
}

function getAvailableActions(resourceType: string) {
  return permissionConfig[resourceType as ResourceType] || []
}

function getPermissionLabel(actionValue: string): string {
  const action = Object.values(permissionConfig)
    .flat()
    .find((item) => item.value === actionValue)
  return action?.label || actionValue
}

function getPermissionDescription(actionValue: string): string {
  const action = Object.values(permissionConfig)
    .flat()
    .find((item) => item.value === actionValue)
  return action?.description || ''
}

function getActionVerb(actionValue: string): string {
  return actionValue.split(':').at(-1) || actionValue
}

function getRolePermissions(role: Role): Record<string, string[]> {
  if (!role.permissions || role.permissions.length === 0) {
    return {}
  }

  const grouped: Record<string, string[]> = {}
  for (const permission of role.permissions) {
    const actions = grouped[permission.resource_type] ?? (grouped[permission.resource_type] = [])
    actions.push(permission.action)
  }
  return grouped
}

function getRolePermissionPreview(role: Role, limit: number = 6): string[] {
  return (role.permissions || [])
    .map((permission) => getPermissionLabel(permission.action))
    .slice(0, limit)
}

function getRolePermissionCount(role: Role): number {
  return role.permissions?.length || 0
}

function getRoleVisual(role: Role) {
  const actions = (role.permissions || []).map((permission) => permission.action)

  if (actions.some((action) => getActionVerb(action) === 'admin')) {
    return {
      icon: Key,
      tone: 'admin' as RoleVisualTone,
      label: '管理员角色',
    }
  }

  if (actions.some((action) => ['write', 'update', 'delete', 'create'].includes(getActionVerb(action)))) {
    return {
      icon: EditPen,
      tone: 'edit' as RoleVisualTone,
      label: '可编辑角色',
    }
  }

  if (actions.some((action) => getActionVerb(action) === 'read')) {
    return {
      icon: View,
      tone: 'view' as RoleVisualTone,
      label: '只读角色',
    }
  }

  return {
    icon: User,
    tone: 'other' as RoleVisualTone,
    label: '自定义角色',
  }
}

function getRoleFallbackDescription(role: Role): string {
  const resourceLabel = getResourceTypeLabel(role.resource_type || defaultPrimaryResourceType)
  return `适用于${resourceLabel}场景的权限模板。`
}

function inferPrimaryResourceType(role: Role): ResourceType {
  if (role.resource_type && resourceTypes.includes(role.resource_type as ResourceType)) {
    return role.resource_type as ResourceType
  }

  const firstPermission = role.permissions?.[0]
  if (firstPermission && resourceTypes.includes(firstPermission.resource_type as ResourceType)) {
    return firstPermission.resource_type as ResourceType
  }

  return defaultPrimaryResourceType
}

function getRoleTreeNodeId(resourceType: ResourceType): string {
  return `resource-${resourceType}`
}

function getRoleCountForResourceType(resourceType: ResourceType): number {
  return roleList.value.filter((role) => inferPrimaryResourceType(role) === resourceType).length
}

const roleTreeData = computed<RoleTreeNavNode[]>(() => [
  {
    id: getRoleTreeNodeId('app'),
    kind: 'resource',
    label: getResourceTypeLabel('app'),
    resourceType: 'app',
    roleCount: getRoleCountForResourceType('app'),
    children: [
      {
        id: getRoleTreeNodeId('directory'),
        kind: 'resource',
        label: getResourceTypeLabel('directory'),
        resourceType: 'directory',
        roleCount: getRoleCountForResourceType('directory'),
        children: directoryInheritedResourceTypes.map((resourceType) => ({
          id: getRoleTreeNodeId(resourceType),
          kind: 'resource',
          label: getResourceTypeLabel(resourceType),
          resourceType,
          roleCount: getRoleCountForResourceType(resourceType),
        })),
      },
    ],
  },
])

const selectedResourceRoles = computed(() =>
  roleList.value.filter((role) => inferPrimaryResourceType(role) === selectedTreeResourceType.value)
)

function isImageResourceIcon(resourceType: ResourceType): boolean {
  return resourceType === 'directory'
    || resourceType === 'form'
    || resourceType === 'docs'
    || resourceType === 'board'
    || resourceType === 'app'
}

function getResourceTypeIconSrc(resourceType: ResourceType): string {
  switch (resourceType) {
    case 'directory':
      return '/service-tree/custom-folder.svg'
    case 'form':
      return '/service-tree/编辑.svg'
    case 'docs':
      return '/文档.svg'
    case 'board':
      return '/讨论区.svg'
    case 'app':
      return '/service-tree/custom-folder.svg'
    default:
      return '/service-tree/custom-folder.svg'
  }
}

function getResourceTypeIconComponent(resourceType: ResourceType) {
  switch (resourceType) {
    case 'table':
      return TableIcon
    case 'chart':
      return ChartIcon
    default:
      return TableIcon
  }
}

function getResourceTypeIconClass(resourceType: ResourceType): string {
  switch (resourceType) {
    case 'directory':
      return 'package-icon-img'
    case 'form':
      return 'form-icon-img'
    case 'docs':
      return 'docs-icon-img'
    case 'board':
      return 'board-icon-img'
    case 'app':
      return 'app-icon-img'
    case 'table':
      return 'table-icon'
    case 'chart':
      return 'chart-icon'
    default:
      return ''
  }
}

function getResourceTypeIconAlt(resourceType: ResourceType): string {
  return getResourceTypeLabel(resourceType)
}

function filterRoleTreeNode(keyword: string, data: RoleTreeNavNode): boolean {
  const normalized = keyword.trim().toLowerCase()
  if (!normalized) {
    return true
  }

  if (data.label.toLowerCase().includes(normalized)) {
    return true
  }

  return (data.children || []).some((child) => filterRoleTreeNode(keyword, child))
}

const focusedRole = computed<Role | null>(() => {
  if (focusedRoleId.value != null) {
    const matchedRole = selectedResourceRoles.value.find((role) => role.id === focusedRoleId.value)
    if (matchedRole) {
      return matchedRole
    }
  }

  return selectedResourceRoles.value[0] ?? null
})

const focusedRolePrimaryResourceType = computed<ResourceType>(() => {
  if (!focusedRole.value) {
    return defaultPrimaryResourceType
  }
  return inferPrimaryResourceType(focusedRole.value)
})

function getActionLabelsFromRole(role: Role, resourceType: ResourceType): string[] {
  const groupedPermissions = getRolePermissions(role)
  return (groupedPermissions[resourceType] || []).map((action) => getPermissionLabel(action))
}

function getNodeActionStates(node: RoleHierarchyNode) {
  const availableActions = getAvailableActions(node.resourceType)
  const enabledActions = new Set(node.actions)
  return availableActions.map((action) => ({
    value: action.value,
    label: action.label,
    description: action.description || '',
    enabled: enabledActions.has(action.label),
  }))
}

function buildRoleHierarchyTree(role: Role): RoleHierarchyNode[] {
  const primaryResourceType = inferPrimaryResourceType(role)
  const roleName = role.name

  if (primaryResourceType === 'directory') {
    const directoryActions = getActionLabelsFromRole(role, 'directory')
    return [{
      id: `role-${role.id}-directory-root`,
      title: '目录',
      description: `角色「${roleName}」直接作用在目录节点上，下面这些资源会按配置继续继承。`,
      resourceType: 'directory',
      status: directoryActions.length > 0 ? '直接作用' : '未配置',
      statusType: directoryActions.length > 0 ? 'primary' : 'info',
      actions: directoryActions,
      children: directoryInheritedResourceTypes.map((resourceType) => {
        const actionLabels = getActionLabelsFromRole(role, resourceType)
        return {
          id: `role-${role.id}-${resourceType}`,
          title: getResourceTypeLabel(resourceType),
          description: actionLabels.length > 0
            ? `目录下的${getResourceTypeLabel(resourceType)}会继承这些动作。`
            : `当前没有给${getResourceTypeLabel(resourceType)}配置继承动作。`,
          resourceType,
          status: actionLabels.length > 0 ? '继承中' : '未配置',
          statusType: actionLabels.length > 0 ? 'success' : 'info',
          actions: actionLabels,
        } satisfies RoleHierarchyNode
      }),
    }]
  }

  const directActions = getActionLabelsFromRole(role, primaryResourceType)
  return [{
    id: `role-${role.id}-${primaryResourceType}`,
    title: getResourceTypeLabel(primaryResourceType),
    description: `角色「${roleName}」直接作用在${getResourceTypeLabel(primaryResourceType)}资源上。`,
    resourceType: primaryResourceType,
    status: directActions.length > 0 ? '直接作用' : '未配置',
    statusType: directActions.length > 0 ? 'primary' : 'info',
    actions: directActions,
  }]
}

const focusedRoleTreeData = computed<RoleHierarchyNode[]>(() => {
  if (!focusedRole.value) {
    return []
  }
  return buildRoleHierarchyTree(focusedRole.value)
})

const currentRolePrimaryResourceType = computed<ResourceType>(() => {
  if (roleForm.id) {
    const currentRole = roleList.value.find((role) => role.id === roleForm.id)
    if (currentRole) {
      return inferPrimaryResourceType(currentRole)
    }
  }

  return roleFormPrimaryResourceType.value
})

const currentRolePrimaryResourceLabel = computed(() => getResourceTypeLabel(currentRolePrimaryResourceType.value))
const isDirectoryRole = computed(() => currentRolePrimaryResourceType.value === 'directory')

const availableRoleConfigResourceTypes = computed<ResourceType[]>(() => {
  if (isDirectoryRole.value && roleFormCrossResourceEnabled.value) {
    return directoryRoleConfigResourceTypes
  }

  return [currentRolePrimaryResourceType.value]
})

function isRoleEditorResourceDisabled(resourceType: ResourceType): boolean {
  return isDirectoryRole.value
    && resourceType !== 'directory'
    && !roleFormCrossResourceEnabled.value
}

const roleEditorTreeData = computed<RoleEditorNode[]>(() => {
  if (currentRolePrimaryResourceType.value === 'directory') {
    return [{
      id: getRoleTreeNodeId('directory'),
      label: '目录',
      resourceType: 'directory',
      status: '直接作用',
      statusType: 'primary',
      children: directoryInheritedResourceTypes.map((resourceType) => ({
        id: getRoleTreeNodeId(resourceType),
        label: getResourceTypeLabel(resourceType),
        resourceType,
        status: isRoleEditorResourceDisabled(resourceType) ? '未开启' : '可编辑',
        statusType: isRoleEditorResourceDisabled(resourceType) ? 'info' : 'success',
        disabled: isRoleEditorResourceDisabled(resourceType),
      })),
    }]
  }

  return [{
    id: getRoleTreeNodeId(currentRolePrimaryResourceType.value),
    label: getResourceTypeLabel(currentRolePrimaryResourceType.value),
    resourceType: currentRolePrimaryResourceType.value,
    status: '直接作用',
    statusType: 'primary',
  }]
})

const currentRoleEditorActionOptions = computed(() =>
  getAvailableActions(selectedRoleEditorResourceType.value)
)

const currentRoleEditorDisabled = computed(() =>
  isRoleEditorResourceDisabled(selectedRoleEditorResourceType.value)
)

const currentRoleEditorEnabledActionCount = computed(() =>
  (roleForm.permissions[selectedRoleEditorResourceType.value] || []).length
)

const currentRoleEditorDescription = computed(() => {
  if (currentRoleEditorDisabled.value) {
    return `先开启目录子资源继承，才能给${getResourceTypeLabel(selectedRoleEditorResourceType.value)}单独配置动作。`
  }

  if (selectedRoleEditorResourceType.value === 'directory') {
    return '这里编辑目录节点本身的动作。目录角色是否继续影响下面的子资源，由左边的继承开关决定。'
  }

  return `这里编辑${getResourceTypeLabel(selectedRoleEditorResourceType.value)}节点会拿到的动作。`
})

function getActionLabelsForResourceType(resourceType: ResourceType): string[] {
  return (roleForm.permissions[resourceType] || []).map((action) => getPermissionLabel(action))
}

const previewRoleTitle = computed(() => {
  const trimmedName = roleForm.name.trim()
  return trimmedName || `${currentRolePrimaryResourceLabel.value}角色模板`
})

const directoryDirectActionLabels = computed(() => getActionLabelsForResourceType('directory'))

const directoryInheritancePreviewNodes = computed(() =>
  directoryInheritedResourceTypes.map((resourceType) => {
    const label = getResourceTypeLabel(resourceType)
    const actionLabels = getActionLabelsForResourceType(resourceType)

    if (!roleFormCrossResourceEnabled.value) {
      return {
        resourceType,
        label,
        actionLabels,
        active: false,
        statusLabel: '未开启',
        description: `开启子资源继承后，才能给${label}配置动作。`,
      }
    }

    if (actionLabels.length === 0) {
      return {
        resourceType,
        label,
        actionLabels,
        active: false,
        statusLabel: '未配置',
        description: `当前没有给${label}配置任何继承动作。`,
      }
    }

    return {
      resourceType,
      label,
      actionLabels,
      active: true,
      statusLabel: '继承中',
      description: `目录下的${label}会继承这些动作。`,
    }
  })
)

const inheritancePreviewTitle = computed(() => (isDirectoryRole.value ? '继承关系树' : '作用范围预览'))

const inheritancePreviewSummary = computed(() => {
  if (!isDirectoryRole.value) {
    return `${currentRolePrimaryResourceLabel.value}角色会直接作用在当前资源类型上。`
  }

  const activeChildren = directoryInheritancePreviewNodes.value.filter((node) => node.actionLabels.length > 0).length
  if (!roleFormCrossResourceEnabled.value) {
    return '当前只配置目录本身权限，目录下的表格、表单、图表、文档和讨论区不会自动继承。'
  }
  if (activeChildren === 0) {
    return '你已经开启子资源继承，但还没有给任何子资源配置动作。'
  }
  return `当前有 ${activeChildren} 类子资源会跟着目录角色一起继承权限。`
})

const inheritancePreviewDescription = computed(() => {
  if (!isDirectoryRole.value) {
    return `${currentRolePrimaryResourceLabel.value}角色只直接作用在当前资源类型上，不会再向下展开。`
  }
  return '用树形结构先看目录本身，再看哪些子资源会继承动作，避免把目录角色配成一团看不懂的 checkbox。'
})

const inheritancePreviewBadgeText = computed(() => {
  if (!isDirectoryRole.value) {
    return '直接作用'
  }
  return roleFormCrossResourceEnabled.value ? '目录 -> 子资源' : '仅目录本身'
})

const singleScopeActionLabels = computed(() => getActionLabelsForResourceType(currentRolePrimaryResourceType.value))

const singleScopePreviewDescription = computed(() => {
  if (currentRolePrimaryResourceType.value === 'app') {
    return '工作空间角色直接作用在工作空间本身，不通过目录继续向下展开。'
  }
  return `${currentRolePrimaryResourceLabel.value}角色直接作用在${currentRolePrimaryResourceLabel.value}资源上，不通过目录继续向下继承。`
})

const roleDialogTitle = computed(() => (roleForm.id ? '编辑角色模板' : '新建角色模板'))

const roleDialogDescription = computed(() => {
  if (roleForm.id) {
    return `调整「${roleForm.name}」能做什么。先保证模板边界清楚，再考虑要不要继续分配给更多人。`
  }

  return '先选角色适用的资源，再配置它允许执行的动作。'
})

const roleSelectedPermissionCount = computed(() =>
  Object.values(roleForm.permissions).reduce((total, actions) => total + actions.length, 0)
)

const assignDialogDescription = computed(() => {
  if (!currentAssignRole.value) {
    return '把角色模板分配给用户或组织架构。'
  }

  return `把「${currentAssignRole.value.name}」分配给用户或组织架构，并指定它在哪个资源范围内生效。`
})

const assignRootResourcePath = computed(() => buildAppResourcePath(assignForm.user, assignForm.app))

const assignCustomPathPlaceholder = computed(() => {
  if (!assignRootResourcePath.value) {
    return '/user/app/docs/guide'
  }
  return `${assignRootResourcePath.value}/docs/guide`
})

watch(roleFormPrimaryResourceType, (nextType) => {
  if (roleForm.id) {
    return
  }

  if (nextType !== 'directory') {
    roleFormCrossResourceEnabled.value = false
  }

  clearPermissionsOutsideVisibleTypes()
  handlePermissionChange()
})

watch(currentRolePrimaryResourceType, (nextType) => {
  selectedRoleEditorResourceType.value = nextType
  selectedRoleEditorNodeKey.value = getRoleTreeNodeId(nextType)
})

watch(roleTreeKeyword, (keyword) => {
  roleTreeRef.value?.filter(keyword)
})

watch(
  roleList,
  (nextRoles) => {
    if (nextRoles.length === 0) {
      focusedRoleId.value = null
      selectedRoleTreeKey.value = getRoleTreeNodeId(defaultPrimaryResourceType)
      return
    }

    selectedRoleTreeKey.value = getRoleTreeNodeId(selectedTreeResourceType.value)
  },
  { immediate: true }
)

watch(
  selectedResourceRoles,
  (nextRoles) => {
    if (nextRoles.length === 0) {
      focusedRoleId.value = null
      if (selectedRoleTreeKey.value !== getRoleTreeNodeId(selectedTreeResourceType.value)) {
        selectedRoleTreeKey.value = getRoleTreeNodeId(selectedTreeResourceType.value)
      }
      return
    }

    if (!nextRoles.some((role) => role.id === focusedRoleId.value)) {
      focusedRoleId.value = nextRoles[0]?.id ?? null
    }

    if (!selectedRoleTreeKey.value) {
      selectedRoleTreeKey.value = getRoleTreeNodeId(selectedTreeResourceType.value)
    }
  },
  { immediate: true }
)

watch(roleFormCrossResourceEnabled, (enabled) => {
  if (!enabled) {
    if (selectedRoleEditorResourceType.value !== 'directory' && currentRolePrimaryResourceType.value === 'directory') {
      selectedRoleEditorResourceType.value = 'directory'
      selectedRoleEditorNodeKey.value = getRoleTreeNodeId('directory')
    }
    clearPermissionsOutsideVisibleTypes()
    handlePermissionChange()
  }
})

watch(
  [() => assignForm.user, () => assignForm.app, assignScopeMode],
  ([user, app, scopeMode]) => {
    if (scopeMode === 'custom') {
      return
    }

    const rootPath = buildAppResourcePath(user, app)
    const nextPath = scopeMode === 'workspace_children'
      ? (rootPath ? `${rootPath}/*` : '')
      : rootPath

    if (assignForm.resource_path !== nextPath) {
      assignForm.resource_path = nextPath
    }
  }
)

watch(
  () => assignForm.resource_path,
  (resourcePath) => {
    if (assignScopeMode.value !== 'custom') {
      return
    }

    const parsed = parseResourcePath(resourcePath)
    if (!parsed) {
      return
    }

    if (assignForm.user !== parsed.user) {
      assignForm.user = parsed.user
    }
    if (assignForm.app !== parsed.app) {
      assignForm.app = parsed.app
    }
  }
)

function clearPermissionsOutsideVisibleTypes() {
  const nextPermissions = normalizePermissions(roleForm.permissions)
  const visibleTypes = new Set(availableRoleConfigResourceTypes.value)

  for (const resourceType of resourceTypes) {
    if (!visibleTypes.has(resourceType)) {
      nextPermissions[resourceType] = []
    }
  }

  roleForm.permissions = nextPermissions
}

function hasRoleFormAction(resourceType: ResourceType, actionValue: string): boolean {
  return (roleForm.permissions[resourceType] || []).includes(actionValue)
}

function setRoleFormAction(resourceType: ResourceType, actionValue: string, enabled: boolean | string | number) {
  const nextPermissions = normalizePermissions(roleForm.permissions)
  const currentActions = new Set(nextPermissions[resourceType] || [])

  if (enabled === true) {
    currentActions.add(actionValue)
  } else {
    currentActions.delete(actionValue)
  }

  nextPermissions[resourceType] = Array.from(currentActions)
  roleForm.permissions = nextPermissions
  handlePermissionChange()
}

function setCurrentEditorActions(enabled: boolean) {
  if (currentRoleEditorDisabled.value) {
    return
  }

  const nextPermissions = normalizePermissions(roleForm.permissions)
  nextPermissions[selectedRoleEditorResourceType.value] = enabled
    ? currentRoleEditorActionOptions.value.map((action) => action.value)
    : []
  roleForm.permissions = nextPermissions
  handlePermissionChange()
}

function handlePermissionChange() {
  roleFormRef.value?.validateField('permissions').catch(() => undefined)
}

function isAllSelected(resourceType: string): boolean {
  const selected = roleForm.permissions[resourceType] || []
  const available = getAvailableActions(resourceType)
  return available.length > 0 && selected.length === available.length
}

function isIndeterminate(resourceType: string): boolean {
  const selected = roleForm.permissions[resourceType] || []
  const available = getAvailableActions(resourceType)
  return selected.length > 0 && selected.length < available.length
}

function handleSelectAll(resourceType: string, checked: boolean | string | number) {
  const enabled = checked === true
  const nextPermissions = normalizePermissions(roleForm.permissions)
  nextPermissions[resourceType] = enabled
    ? getAvailableActions(resourceType).map((action) => action.value)
    : []
  roleForm.permissions = nextPermissions
  handlePermissionChange()
}

function handleSelectAllChange(resourceType: string, checked: boolean | string | number) {
  handleSelectAll(resourceType, checked)
}

function formatDateTime(dateStr: string): string {
  if (!dateStr) return ''

  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function resetRoleForm(primaryResourceType: ResourceType = defaultPrimaryResourceType) {
  Object.assign(roleForm, {
    id: undefined,
    name: '',
    code: '',
    description: '',
    is_default: false,
    permissions: createEmptyPermissions(),
  })

  roleFormPrimaryResourceType.value = primaryResourceType
  roleFormCrossResourceEnabled.value = false
  selectedRoleEditorResourceType.value = primaryResourceType
  selectedRoleEditorNodeKey.value = getRoleTreeNodeId(primaryResourceType)
}

function handleRoleTreeNodeClick(data: RoleTreeNavNode) {
  selectedTreeResourceType.value = data.resourceType
  selectedRoleTreeKey.value = data.id

  if (!selectedResourceRoles.value.some((role) => role.id === focusedRoleId.value)) {
    focusedRoleId.value = selectedResourceRoles.value[0]?.id ?? null
  }
}

function handleRoleEditorNodeClick(data: RoleEditorNode) {
  if (data.disabled) {
    return
  }

  selectedRoleEditorResourceType.value = data.resourceType
  selectedRoleEditorNodeKey.value = data.id
}

function handleSelectRoleFromDetail(role: Role) {
  const resourceType = inferPrimaryResourceType(role)
  selectedTreeResourceType.value = resourceType
  focusedRoleId.value = role.id
  selectedRoleTreeKey.value = getRoleTreeNodeId(resourceType)
}

async function loadRoles() {
  try {
    loading.value = true
    const response = await getRoles()
    roleList.value = Array.isArray(response?.roles) ? response.roles : []
  } catch (error: any) {
    roleList.value = []
    ElMessage.error(`加载角色列表失败: ${error.message || '未知错误'}`)
  } finally {
    loading.value = false
  }
}

function handleCreateRole() {
  resetRoleForm(selectedTreeResourceType.value)
  roleDialogVisible.value = true
}

function handleCreateRoleForResourceType(resourceType: string) {
  resetRoleForm(resourceType as ResourceType)
  const firstAction = getAvailableActions(resourceType)[0]
  if (firstAction) {
    roleForm.permissions[resourceType] = [firstAction.value]
  }
  roleDialogVisible.value = true
}

async function handleEditRole(role: Role) {
  try {
    loading.value = true
    const response = await getRole(role.id)
    const roleData = response.role
    const primaryType = inferPrimaryResourceType(roleData)

    resetRoleForm(primaryType)

    Object.assign(roleForm, {
      id: roleData.id,
      name: roleData.name,
      code: roleData.code,
      description: roleData.description || '',
      is_default: roleData.is_default || false,
      permissions: createEmptyPermissions(),
    })

    const filledPermissions = createEmptyPermissions()
    for (const permission of roleData.permissions || []) {
      const actions = filledPermissions[permission.resource_type] ?? (filledPermissions[permission.resource_type] = [])
      actions.push(permission.action)
    }

    roleForm.permissions = filledPermissions
    roleFormCrossResourceEnabled.value = primaryType === 'directory'
      && Object.entries(filledPermissions).some(([resourceType, actions]) => resourceType !== 'directory' && actions.length > 0)
    clearPermissionsOutsideVisibleTypes()

    roleDialogVisible.value = true
  } catch (error: any) {
    ElMessage.error(`加载角色详情失败: ${error.message || '未知错误'}`)
  } finally {
    loading.value = false
  }
}

async function handleSubmitRole() {
  if (!roleFormRef.value) {
    return
  }

  try {
    await roleFormRef.value.validate()
    roleSubmitting.value = true

    const permissions: Record<string, string[]> = {}
    for (const resourceType of availableRoleConfigResourceTypes.value) {
      const actions = roleForm.permissions[resourceType] ?? []
      if (actions.length > 0) {
        permissions[resourceType] = actions
      }
    }

    if (roleForm.id) {
      const request: UpdateRoleReq = {
        name: roleForm.name,
        description: roleForm.description,
        is_default: roleForm.is_default,
        permissions,
      }
      await updateRole(roleForm.id, request)
      ElMessage.success('更新角色成功')
    } else {
      const request: CreateRoleReq = {
        name: roleForm.name,
        code: roleForm.code,
        description: roleForm.description,
        permissions,
      }
      await createRole(request)
      ElMessage.success('创建角色成功')
    }

    roleDialogVisible.value = false
    await loadRoles()
  } catch (error: any) {
    if (error?.message && !error.message.includes('验证')) {
      ElMessage.error(`操作失败: ${error.message || '未知错误'}`)
    }
  } finally {
    roleSubmitting.value = false
  }
}

async function handleDeleteRole(role: Role) {
  try {
    await ElMessageBox.confirm(
      `确定要删除角色 "${role.name}" 吗？此操作不可恢复。`,
      '确认删除',
      { type: 'warning' }
    )

    await deleteRole(role.id)
    ElMessage.success('删除角色成功')
    await loadRoles()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(`删除角色失败: ${error.message || '未知错误'}`)
    }
  }
}

function handleAssignUserChange(value: FieldValue) {
  assignForm.username = extractStringFieldRaw(value)
}

function handleAssignRole(role: Role) {
  currentAssignRole.value = role

  Object.assign(assignForm, {
    subject_type: 'user',
    username: '',
    department_path: '',
    user: '',
    app: '',
    resource_path: '',
    isPermanent: true,
    start_time: undefined,
    end_time: undefined,
  })

  assignScopeMode.value = 'workspace'
  assignDialogVisible.value = true
}

async function handleSubmitAssign() {
  if (!assignFormRef.value || !currentAssignRole.value) {
    return
  }

  try {
    await assignFormRef.value.validate()
    assignSubmitting.value = true

    const parsedResourcePath = parseResourcePath(assignForm.resource_path)
    if (!parsedResourcePath) {
      ElMessage.error('资源路径格式错误，至少需要 /user/app')
      return
    }

    if (assignForm.subject_type === 'user') {
      const request: AssignRoleToUserReq = {
        user: assignForm.user,
        app: assignForm.app,
        username: assignForm.username,
        role_code: currentAssignRole.value.code,
        resource_path: assignForm.resource_path,
        start_time: assignForm.isPermanent ? undefined : assignForm.start_time,
        end_time: assignForm.isPermanent ? undefined : assignForm.end_time,
      }
      await assignRoleToUser(request)
    } else {
      const request: AssignRoleToDepartmentReq = {
        user: assignForm.user,
        app: assignForm.app,
        department_path: assignForm.department_path,
        role_code: currentAssignRole.value.code,
        resource_path: assignForm.resource_path,
        start_time: assignForm.isPermanent ? undefined : assignForm.start_time,
        end_time: assignForm.isPermanent ? undefined : assignForm.end_time,
      }
      await assignRoleToDepartment(request)
    }

    ElMessage.success('分配角色成功')
    assignDialogVisible.value = false
  } catch (error: any) {
    if (error?.message && !error.message.includes('验证')) {
      ElMessage.error(`分配角色失败: ${error.message || '未知错误'}`)
    }
  } finally {
    assignSubmitting.value = false
  }
}

onMounted(() => {
  loadRoles()
})
</script>

<style scoped lang="scss">
.role-management-page {
  --role-ink: var(--text-primary);
  --role-muted: var(--text-secondary);
  --role-line: color-mix(in srgb, var(--border-base) 80%, var(--color-primary) 20%);
  --role-card: var(--bg-primary);
  --role-page: var(--bg-page);
  --role-shadow: var(--box-shadow-sm);
  --role-soft: color-mix(in srgb, var(--color-primary) 10%, transparent);
  min-height: 100%;
  padding: 24px;
  background: var(--role-page);
  color: var(--role-ink);
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.page-header-copy h1 {
  margin: 0;
  font-size: 28px;
  font-weight: 700;
}

.page-header-copy p {
  max-width: 760px;
  margin: 8px 0 0;
  color: var(--role-muted);
  line-height: 1.7;
}

.page-header-actions {
  display: flex;
  gap: 10px;
  flex-shrink: 0;
}

.page-body {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  gap: 18px;
}

.tree-column,
.detail-column {
  min-width: 0;
}

.detail-column {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.tree-panel-card,
.detail-panel-card {
  height: 100%;
}

.tree-panel-card {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.tree-panel-header,
.detail-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.detail-header {
  margin-bottom: 18px;
}

.detail-header-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.detail-header h2 {
  margin: 6px 0 0;
  font-size: 24px;
}

.detail-header p {
  margin: 8px 0 0;
  color: var(--role-muted);
  line-height: 1.7;
}

.detail-header-actions {
  flex-shrink: 0;
}

.active-role-inline {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  border-radius: 14px;
  background: color-mix(in srgb, var(--bg-primary) 92%, var(--color-primary) 8%);
  border: 1px solid var(--role-line);
}

.active-role-inline-main {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.active-role-inline h4 {
  margin: 6px 0 0;
  font-size: 18px;
}

.active-role-inline p {
  margin: 8px 0 0;
  color: var(--role-muted);
  font-size: 13px;
  line-height: 1.6;
}

.active-role-inline-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 12px;
}

.role-tree-search {
  margin-bottom: 4px;
}

.role-tree {
  min-height: 420px;
  background: transparent;
}

.role-tree :deep(.el-tree-node__content) {
  height: auto;
  margin-bottom: 2px;
  padding: 0 8px;
  display: flex;
  align-items: center;
  border-radius: 0;
}

.role-tree :deep(.el-tree-node__content:hover) {
  background: var(--el-fill-color-light);
}

.role-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
  background: rgba(99, 102, 241, 0.15);
  border-left: 2px solid #6366f1;
}

.role-tree :deep(.el-tree-node.is-current .el-tree-node__children .el-tree-node__content) {
  background: transparent;
  border-left: none;
}

.role-tree :deep(.el-tree-node__expand-icon) {
  padding: 6px;
  color: var(--el-text-color-secondary);
  border-radius: 2px;
  transition: all 0.2s ease;
}

.role-tree :deep(.el-tree-node__expand-icon:hover) {
  background: var(--el-fill-color);
}

.role-tree :deep(.el-tree-node.is-expanded > .el-tree-node__content .el-tree-node__expand-icon) {
  transform: rotate(90deg);
}

.role-tree :deep(.el-tree-node__expand-icon.is-leaf) {
  color: transparent;
}

.role-tree :deep(.el-tree-node__children) {
  overflow: visible;
}

.role-tree-node {
  width: 100%;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
}

.role-tree-node-main,
.role-tree-node-side {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.role-tree-node-resource .role-tree-label {
  font-weight: 600;
}

.role-tree-node-role .role-tree-label {
  font-size: 13px;
}

.role-tree-icon {
  flex-shrink: 0;
  color: var(--color-primary);
}

.node-icon {
  width: 16px;
  height: 16px;
  margin-right: 8px;
  flex-shrink: 0;
  opacity: 0.88;
  color: #6366f1;
}

.node-icon.app-icon-img,
.node-icon.package-icon-img,
.node-icon.form-icon-img,
.node-icon.docs-icon-img,
.node-icon.board-icon-img {
  object-fit: contain;
}

.node-icon.chart-icon {
  color: #9377e0;
}

.node-icon.table-icon {
  color: #553cce;
}

.role-tree-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.role-switcher-shell {
  margin-bottom: 16px;
}

.role-switcher-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.role-switcher-header h3 {
  margin: 6px 0 0;
  font-size: 16px;
}

.role-switcher-hint {
  color: var(--role-muted);
  font-size: 12px;
  line-height: 1.6;
}

.role-switcher {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.role-switcher-item {
  position: relative;
  min-width: 180px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
  padding: 14px;
  border-radius: 14px;
  border: 1px solid var(--role-line);
  background: var(--bg-primary);
  color: var(--role-ink);
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}

.role-switcher-item-head {
  display: flex;
  align-items: center;
  gap: 10px;
}

.role-switcher-item-copy {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
}

.role-switcher-item-copy span,
.role-switcher-item-meta {
  font-size: 12px;
  color: var(--role-muted);
}

.role-switcher-item-footer {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.role-switcher-state {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 8px;
  border-radius: 999px;
  background: var(--bg-secondary);
  color: var(--role-muted);
  font-size: 12px;
  white-space: nowrap;
}

.role-switcher-state.is-active {
  background: color-mix(in srgb, var(--color-primary) 12%, transparent);
  color: var(--color-primary);
  font-weight: 600;
}

.role-switcher-item:hover {
  border-color: color-mix(in srgb, var(--color-primary) 42%, var(--border-base) 58%);
  box-shadow: var(--box-shadow-sm);
  transform: translateY(-1px);
}

.role-switcher-item.is-active {
  border-color: color-mix(in srgb, var(--color-primary) 76%, var(--border-base) 24%);
  background: color-mix(in srgb, var(--bg-primary) 84%, var(--color-primary) 16%);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-primary) 22%, transparent);
  transform: translateY(-1px);
}

.role-switcher-item.is-active::after {
  content: '';
  position: absolute;
  top: 10px;
  right: 10px;
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: var(--color-primary);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--color-primary) 18%, transparent);
}

.role-switcher-item strong {
  font-size: 14px;
}

.role-switcher-item-view {
  background: color-mix(in srgb, var(--bg-primary) 96%, #3b82f6 4%);
}

.role-switcher-item-edit {
  background: color-mix(in srgb, var(--bg-primary) 96%, #f59e0b 4%);
}

.role-switcher-item-admin {
  background: color-mix(in srgb, var(--bg-primary) 94%, #ef4444 6%);
}

.role-switcher-item-other {
  background: color-mix(in srgb, var(--bg-primary) 96%, var(--color-primary) 4%);
}

.role-icon-badge {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  flex-shrink: 0;
}

.role-icon-badge-lg {
  width: 36px;
  height: 36px;
  border-radius: 12px;
}

.role-icon-badge-view {
  background: rgba(59, 130, 246, 0.12);
  color: #2563eb;
}

.role-icon-badge-edit {
  background: rgba(245, 158, 11, 0.14);
  color: #d97706;
}

.role-icon-badge-admin {
  background: rgba(239, 68, 68, 0.12);
  color: #dc2626;
}

.role-icon-badge-other {
  background: color-mix(in srgb, var(--color-primary) 12%, transparent);
  color: var(--color-primary);
}

.detail-role-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 4px;
}

.usage-list {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.usage-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  border-radius: 14px;
  border: 1px solid var(--role-line);
  background: var(--bg-secondary);
}

.usage-item strong {
  font-size: 14px;
}

.usage-item span {
  color: var(--role-muted);
  font-size: 13px;
  line-height: 1.6;
}

.panel-card {
  padding: 18px;
  border-radius: 18px;
  background: var(--role-card);
  border: 1px solid var(--border-base);
  box-shadow: var(--role-shadow);
}

.panel-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.panel-copy h2 {
  margin: 0;
  font-size: 20px;
}

.panel-copy p {
  margin: 6px 0 0;
  color: var(--role-muted);
  line-height: 1.6;
}

.resource-filter {
  flex-wrap: wrap;
}

.role-group-list {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.role-group {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.role-group-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.role-group-title {
  display: flex;
  align-items: center;
  gap: 10px;
}

.role-group-title h3 {
  margin: 0;
  font-size: 18px;
}

.role-group-header p {
  margin: 6px 0 0;
  color: var(--role-muted);
  font-size: 13px;
}

.role-card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 14px;
}

.role-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px;
  border-radius: 16px;
  border: 1px solid var(--role-line);
  background: color-mix(in srgb, var(--bg-primary) 94%, var(--color-primary) 6%);
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}

.role-card:hover {
  border-color: color-mix(in srgb, var(--color-primary) 42%, var(--border-base) 58%);
  box-shadow: var(--box-shadow-sm);
  transform: translateY(-1px);
}

.role-card.is-active {
  border-color: color-mix(in srgb, var(--color-primary) 58%, var(--border-base) 42%);
  background: color-mix(in srgb, var(--bg-primary) 88%, var(--color-primary) 12%);
}

.role-card-title {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.role-card-title h4 {
  margin: 0;
  font-size: 18px;
  line-height: 1.2;
}

.role-card-copy p {
  margin: 8px 0 0;
  color: var(--role-muted);
  line-height: 1.6;
  font-size: 13px;
}

.role-tags {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.role-meta-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.meta-chip {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px;
  border-radius: 12px;
  background: var(--bg-secondary);
}

.meta-label,
.summary-label,
.field-label,
.section-label {
  font-size: 12px;
  color: var(--role-muted);
}

.meta-chip strong {
  font-size: 13px;
}

.meta-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  word-break: break-word;
}

.role-capabilities {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.capability-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.capability-tag {
  margin: 0;
}

.subtle-text {
  margin: 0;
  color: var(--role-muted);
  font-size: 13px;
}

.role-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--role-line);
}

.footer-summary {
  font-size: 13px;
  color: var(--role-muted);
}

.role-card-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.preview-panel {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.hierarchy-card h3 {
  margin: 0 0 10px;
  font-size: 18px;
}

.hierarchy-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.hierarchy-header p {
  margin: 0;
  color: var(--role-muted);
  line-height: 1.7;
}

.hierarchy-tree-shell {
  border-radius: 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-base);
  padding: 14px;
}

.role-hierarchy-tree {
  background: transparent;
}

.role-hierarchy-tree :deep(.el-tree-node__content) {
  height: auto;
  align-items: flex-start;
  padding: 2px 8px;
}

.role-hierarchy-tree :deep(.el-tree-node__content:hover) {
  background: transparent;
}

.role-hierarchy-tree :deep(.el-tree-node__children) {
  overflow: visible;
}

.role-hierarchy-tree :deep(.el-tree-node__expand-icon) {
  padding: 6px;
  color: var(--el-text-color-secondary);
  border-radius: 2px;
}

.role-hierarchy-tree :deep(.el-tree-node__expand-icon:hover) {
  background: var(--el-fill-color);
}

.role-hierarchy-tree :deep(.el-tree-node.is-expanded > .el-tree-node__content .el-tree-node__expand-icon) {
  transform: rotate(90deg);
}

.role-hierarchy-node {
  width: 100%;
  padding: 10px 0 10px 4px;
  border-bottom: 1px solid color-mix(in srgb, var(--border-base) 84%, transparent);
}

.role-hierarchy-node-main {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
}

.role-hierarchy-copy {
  min-width: 0;
}

.role-hierarchy-node-head {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 10px;
  flex-wrap: wrap;
}

.role-hierarchy-node-head strong {
  font-size: 14px;
}

.role-hierarchy-copy p {
  margin: 8px 0 0;
  color: var(--role-muted);
  font-size: 13px;
  line-height: 1.6;
}

.role-hierarchy-actions {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 8px;
  margin-top: 12px;
}

.role-hierarchy-action-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 8px 10px;
  border-radius: 10px;
  background: var(--bg-primary);
  border: 1px solid var(--border-base);
  font-size: 12px;
}

.role-hierarchy-action-item.is-enabled {
  border-color: rgba(34, 197, 94, 0.22);
  background: rgba(34, 197, 94, 0.08);
  color: #15803d;
}

.role-hierarchy-action-item.is-disabled {
  color: var(--role-muted);
  background: color-mix(in srgb, var(--bg-primary) 96%, var(--text-secondary) 4%);
}

.role-hierarchy-action-icon {
  flex-shrink: 0;
}

.role-hierarchy-action-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.role-hierarchy-action-copy span {
  font-size: 12px;
  font-weight: 600;
}

.role-hierarchy-action-copy small {
  color: var(--role-muted);
  font-size: 11px;
  line-height: 1.5;
}

.hierarchy-empty {
  padding: 12px 0 4px;
}

.empty-state-shell,
.group-empty-shell {
  padding: 12px 0;
}

.dialog-header {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.dialog-kicker {
  display: inline-flex;
  align-items: center;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  font-size: 11px;
  font-weight: 700;
  color: color-mix(in srgb, var(--color-primary) 68%, var(--text-secondary) 32%);
}

.dialog-header h3 {
  margin: 0;
  font-size: 22px;
}

.dialog-header p {
  margin: 0;
  color: var(--role-muted);
  line-height: 1.7;
  font-size: 13px;
}

.role-dialog-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.dialog-card {
  padding: 16px;
  border-radius: 16px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-base);
}

.section-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.section-heading h4 {
  margin: 0;
  font-size: 18px;
}

.section-heading p {
  margin: 6px 0 0;
  color: var(--role-muted);
  font-size: 13px;
  line-height: 1.6;
}

.type-selector-block {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 18px;
}

.field-help {
  color: var(--role-muted);
  font-size: 12px;
  line-height: 1.6;
}

.locked-type {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--bg-primary) 90%, var(--color-primary) 10%);
  color: var(--role-muted);
}

.type-choice-grid {
  display: flex;
  flex-wrap: wrap;
}

.role-form :deep(.el-form-item:last-child) {
  margin-bottom: 0;
}

.default-switch-row {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}

.permission-heading-side {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.permission-count-pill {
  display: inline-flex;
  align-items: center;
  padding: 6px 10px;
  border-radius: 999px;
  background: var(--role-soft);
  color: var(--color-primary);
  font-size: 12px;
  font-weight: 600;
}

.cross-resource-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--role-muted);
  font-size: 13px;
}

.permission-form-shell :deep(.el-form-item) {
  margin-bottom: 0;
}

.role-editor-layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 16px;
  width: 100%;
}

.role-editor-sidebar {
  padding: 14px;
  border-radius: 14px;
  background: var(--bg-primary);
  border: 1px solid var(--role-line);
}

.role-editor-sidebar-toggle {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--role-line);
}

.role-editor-sidebar-toggle strong {
  display: block;
  font-size: 14px;
}

.role-editor-sidebar-toggle p {
  margin: 6px 0 0;
  color: var(--role-muted);
  font-size: 12px;
  line-height: 1.6;
}

.role-editor-tree {
  background: transparent;
}

.role-editor-tree :deep(.el-tree-node__content) {
  height: auto;
  padding: 0 8px;
  margin-bottom: 2px;
  display: flex;
  align-items: center;
}

.role-editor-tree :deep(.el-tree-node__content:hover) {
  background: var(--el-fill-color-light);
}

.role-editor-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
  background: rgba(99, 102, 241, 0.15);
  border-left: 2px solid #6366f1;
}

.role-editor-tree :deep(.el-tree-node.is-current .el-tree-node__children .el-tree-node__content) {
  background: transparent;
  border-left: none;
}

.role-editor-tree :deep(.el-tree-node__expand-icon) {
  padding: 6px;
  color: var(--el-text-color-secondary);
}

.role-editor-tree :deep(.el-tree-node__expand-icon.is-leaf) {
  color: transparent;
}

.role-editor-node {
  width: 100%;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
}

.role-editor-node.is-disabled {
  opacity: 0.56;
}

.role-editor-node-main {
  display: flex;
  align-items: center;
  min-width: 0;
}

.role-editor-node-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.role-editor-detail {
  padding: 16px;
  border-radius: 14px;
  background: var(--bg-primary);
  border: 1px solid var(--role-line);
}

.role-editor-detail-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.role-editor-detail-header h4 {
  margin: 6px 0 0;
  font-size: 20px;
}

.role-editor-detail-header p {
  margin: 8px 0 0;
  color: var(--role-muted);
  font-size: 13px;
  line-height: 1.7;
}

.role-editor-detail-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

.role-editor-disabled {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 14px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--bg-primary) 94%, #ef4444 6%);
  color: #b91c1c;
}

.role-editor-disabled strong {
  display: block;
  font-size: 14px;
}

.role-editor-disabled p {
  margin: 6px 0 0;
  color: var(--role-muted);
  font-size: 13px;
  line-height: 1.6;
}

.role-editor-action-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}

.role-editor-action-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px;
  border-radius: 14px;
  background: var(--bg-secondary);
  border: 1px solid var(--border-base);
  transition: border-color 0.2s ease, background 0.2s ease;
}

.role-editor-action-card.is-enabled {
  border-color: color-mix(in srgb, var(--color-primary) 45%, var(--border-base) 55%);
  background: color-mix(in srgb, var(--bg-primary) 90%, var(--color-primary) 10%);
}

.role-editor-action-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.role-editor-action-copy strong {
  font-size: 14px;
}

.role-editor-action-description {
  margin: 0;
  color: var(--role-muted);
  font-size: 12px;
  line-height: 1.6;
}

.role-editor-action-copy span {
  color: var(--role-muted);
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  word-break: break-word;
}

.permission-sections {
  display: flex;
  flex-direction: column;
  gap: 14px;
  width: 100%;
}

.permission-section {
  padding: 14px;
  border-radius: 14px;
  background: var(--bg-primary);
  border: 1px solid var(--role-line);
}

.permission-section-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.permission-section-header h5 {
  margin: 0;
  font-size: 15px;
}

.permission-section-header p {
  margin: 6px 0 0;
  color: var(--role-muted);
  font-size: 12px;
  line-height: 1.6;
}

.permission-choice-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 10px;
}

.permission-choice {
  margin: 0;
  padding: 12px;
  border-radius: 12px;
  border: 1px solid var(--border-base);
  background: var(--bg-primary);
}

.permission-choice :deep(.el-checkbox__label) {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.permission-choice-label {
  color: var(--role-ink);
  font-size: 13px;
  font-weight: 600;
}

.permission-choice-code {
  color: var(--role-muted);
  font-size: 12px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

.preview-card {
  background: color-mix(in srgb, var(--bg-secondary) 90%, var(--color-primary) 10%);
}

.inheritance-tree,
.scope-preview {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.tree-node {
  position: relative;
  padding: 14px 16px;
  border-radius: 14px;
  background: var(--bg-primary);
  border: 1px solid var(--role-line);
}

.tree-node-root {
  background: color-mix(in srgb, var(--bg-primary) 90%, var(--color-primary) 10%);
}

.tree-node-kicker {
  display: inline-flex;
  align-items: center;
  margin-bottom: 8px;
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--role-muted);
}

.tree-node-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}

.tree-node-title-row strong {
  font-size: 15px;
}

.tree-node p {
  margin: 8px 0 0;
  color: var(--role-muted);
  font-size: 13px;
  line-height: 1.6;
}

.tree-children {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding-left: 22px;
}

.tree-children::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: 8px;
  width: 1px;
  background: var(--role-line);
}

.tree-node-direct::before,
.tree-node-branch::before,
.tree-node-leaf::before {
  content: '';
  position: absolute;
  top: 22px;
  left: -14px;
  width: 14px;
  height: 1px;
  background: var(--role-line);
}

.tree-grandchildren {
  position: relative;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
  gap: 10px;
  margin-top: 14px;
  padding-left: 22px;
}

.tree-grandchildren::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: 8px;
  width: 1px;
  background: var(--role-line);
}

.tree-node-leaf.is-inactive {
  background: color-mix(in srgb, var(--bg-primary) 96%, var(--text-secondary) 4%);
}

.tree-node-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 8px;
  border-radius: 999px;
  background: var(--bg-secondary);
  color: var(--role-muted);
  font-size: 12px;
}

.tree-node-badge-active {
  background: var(--role-soft);
  color: var(--color-primary);
}

.tree-node-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.tree-node-tag {
  margin: 0;
}

.assignment-role-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
  padding: 14px 16px;
  border-radius: 14px;
  background: color-mix(in srgb, var(--bg-primary) 92%, var(--color-primary) 8%);
  border: 1px solid var(--role-line);
}

.assignment-role-summary strong {
  display: block;
  margin-top: 6px;
  font-size: 16px;
}

.assign-form .el-form-item {
  margin-bottom: 18px;
}

.form-section-divider {
  margin: 4px 0 18px;
  padding-top: 6px;
  border-top: 1px solid var(--role-line);
  font-size: 14px;
  font-weight: 600;
}

.scope-mode-field {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.path-preview {
  padding: 12px 14px;
  border-radius: 12px;
  background: var(--bg-secondary);
}

.path-preview code {
  display: block;
  margin-top: 6px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  word-break: break-all;
}

.path-tip {
  margin-top: 8px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--role-muted);
}

.dialog-footer-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.role-dialog),
:deep(.assign-dialog) {
  border-radius: 24px;
  overflow: hidden;
}

:deep(.role-dialog .el-dialog__header),
:deep(.assign-dialog .el-dialog__header) {
  padding: 24px 24px 12px;
}

:deep(.role-dialog .el-dialog__body),
:deep(.assign-dialog .el-dialog__body) {
  padding: 0 24px 12px;
}

:deep(.role-dialog .el-dialog__footer),
:deep(.assign-dialog .el-dialog__footer) {
  padding: 12px 24px 24px;
}

@media (max-width: 1380px) {
  .page-body {
    grid-template-columns: 1fr;
  }

  .tree-column {
    order: -1;
  }
}

@media (max-width: 960px) {
  .role-management-page {
    padding: 16px;
  }

  .page-header,
  .tree-panel-header,
  .detail-header,
  .panel-toolbar,
  .role-group-header,
  .section-heading,
  .role-card-footer,
  .hierarchy-header,
  .active-role-inline {
    flex-direction: column;
    align-items: flex-start;
  }

  .role-meta-grid,
  .role-editor-layout,
  .role-editor-action-list,
  .permission-choice-grid,
  .tree-grandchildren,
  .usage-list {
    grid-template-columns: 1fr;
  }

  .page-header-actions,
  .resource-filter,
  .type-choice-grid,
  .active-role-inline-side {
    width: 100%;
  }
}
</style>
