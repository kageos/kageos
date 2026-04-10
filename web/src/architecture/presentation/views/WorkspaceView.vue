<!--
  WorkspaceView - 工作空间视图
  🔥 新架构的展示层组件
  
  职责：
  - 纯 UI 展示，不包含业务逻辑
  - 通过事件与 Application Layer 通信
  - 从 StateManager 获取状态并渲染
-->

<template>
  <div class="workspace-container">
    <!-- 顶部导航栏：工作空间切换 + 应用中心 同一行 -->
    <WorkspaceHeader
      ref="workspaceHeaderRef"
      :current-app="currentApp"
      :app-list="appList"
      :loading-apps="loadingApps"
      :service-tree="serviceTree"
      @switch-app="handleSwitchApp"
      @create-app="showCreateAppDialog"
      @update-app="handleUpdateApp"
      @delete-app="handleDeleteApp"
      @load-apps="loadAppList"
    />

    <div class="workspace-view">
      <!-- 左下角：隐藏/显示目录按钮 -->
      <div class="sidebar-toggle-bottom-left">
        <el-button
          link
          @click="toggleLeftSidebar"
          class="sidebar-toggle-btn"
          :title="showLeftSidebar ? '隐藏目录' : '显示目录'"
        >
          <el-icon>
            <ArrowLeft v-if="showLeftSidebar" />
            <ArrowRight v-else />
          </el-icon>
        </el-button>
      </div>

      <!-- 左侧：目录树 -->
      <div class="left-sidebar" :class="{ 'sidebar-collapsed': !showLeftSidebar }">
        <div class="left-sidebar-tree">
          <ServiceTreePanel
            ref="serviceTreePanelRef"
            :tree-data="serviceTree"
            :loading="loading"
            :current-node-id="currentFunction?.id || null"
            :current-function="currentFunction"
            :expanded-keys="expandedKeys"
            @node-click="handleNodeClick"
            @create-directory="handleCreateDirectory"
            @create-docs="handleCreateDocs"
            @create-board="handleCreateBoard"
            @delete-doc="handleDeleteDoc"
            @delete-board="handleDeleteBoard"
            @delete-function="handleDeleteFunction"
            @delete-directory="handleDeleteDirectory"
            @import-go-files="handleImportGoFiles"
            @publish-to-hub="handlePublishToHub"
            @push-to-hub="handlePushToHub"
            @pull-from-hub="openPullFromHubDialog"
            @refresh-tree="handleRefreshTree"
            @update-history="handleUpdateHistory"
          />
        </div>
      </div>

      <!-- 中间函数渲染区域 -->
      <div class="function-renderer">
        <!-- 右侧边栏控制按钮：工作台会话 -->
        <div class="sidebar-controls" v-if="workstationContext">
          <div class="right-controls">
            <el-button
              v-if="!showRightSidebar"
              link
              @click="toggleRightSidebar"
              class="sidebar-toggle"
              title="显示工作台会话"
            >
              <el-icon><ArrowLeft /></el-icon>
              <el-badge :value="rightSidebarRunningCount" :hidden="rightSidebarRunningCount === 0" :max="99" :offset="[6, -2]">
                工作台会话
              </el-badge>
            </el-button>
            <el-button
              v-if="showRightSidebar"
              link
              @click="toggleRightSidebar"
              class="sidebar-toggle"
              title="隐藏工作台会话"
            >
              <el-icon><ArrowRight /></el-icon>
              隐藏会话
            </el-button>
          </div>
        </div>
        <!-- 面包屑导航（只在显示函数详情时显示） -->
        <FunctionBreadcrumb
          v-if="currentFunction && currentFunction.type === 'function'"
          :current-node="currentFunction"
          :service-tree="serviceTree"
          @node-click="handleBreadcrumbNodeClick"
        />
        
        <!-- 🔥 Create/Edit 模式：根据 queryTab 显示独立页面 -->
        <template v-if="queryTab === 'create' && currentFunction && currentFunctionDetail">
          <div class="form-page">
            <div class="form-page-header">
              <el-button @click="backToList" :icon="ArrowLeft">返回列表</el-button>
              <h2 class="form-page-title">新增数据</h2>
            </div>
            <div class="form-page-content">
              <FormView
                v-if="currentFunctionDetail.template_type === TEMPLATE_TYPE.FORM"
                :key="`form-create-${currentFunction.id}`"
                :function-detail="currentFunctionDetail"
              />
              <div v-else class="empty-state">
                <p>该函数不支持新增操作</p>
              </div>
            </div>
            <div class="form-page-footer">
              <el-button @click="backToList">取消</el-button>
              <el-button type="primary" @click="handleCreateSubmit">提交</el-button>
            </div>
          </div>
        </template>
        
        <template v-else-if="queryTab === 'edit' && currentFunction && currentFunctionDetail">
          <div class="form-page">
            <div class="form-page-header">
              <el-button @click="backToList" :icon="ArrowLeft">返回列表</el-button>
              <h2 class="form-page-title">编辑数据</h2>
            </div>
            <div class="form-page-content">
              <FormView
                v-if="currentFunctionDetail.template_type === TEMPLATE_TYPE.FORM"
                :key="`form-edit-${currentFunction.id}-${editRowId}`"
                :function-detail="editFunctionDetail || undefined"
                :initial-data="editInitialData"
              />
              <div v-else class="empty-state">
                <p>该函数不支持编辑操作</p>
              </div>
            </div>
            <div class="form-page-footer">
              <el-button @click="backToList">取消</el-button>
              <el-button type="primary" @click="handleEditSubmit">保存</el-button>
            </div>
          </div>
        </template>
        
        <!-- 🔥 Detail 模式：显示详情抽屉（通过 URL 参数打开） -->
        <!-- 注意：detail 模式使用抽屉显示，不需要单独的页面 -->
        
        <!-- 🔥 文档详情页面（可滚动） -->
        <div v-if="currentFunction && currentFunction.type === 'docs'" class="main-content-scroll">
          <DocView
            :node="currentFunction"
            @deleted="handleDocDeleted"
          />
        </div>

        <!-- 🔥 版块/讨论区页面（可滚动） -->
        <div v-else-if="currentFunction && currentFunction.type === 'board'" class="main-content-scroll">
          <BoardView
            :node="currentFunction"
          />
        </div>

        <!-- 🔥 服务目录详情页面（可滚动） -->
        <div v-else-if="currentFunction && currentFunction.type === 'package'" class="main-content-scroll">
          <PackageDetailView
            :package-node="currentFunction"
            @refresh="handleRefreshTree"
          />
        </div>
        
        <!-- 函数详情区域（正常模式 - 函数节点） -->
        <div v-else-if="currentFunction && currentFunction.type === 'function'" class="function-content-wrapper">
          <div class="function-content">
            <div v-if="showFunctionTabsWrapper" class="function-tabs-wrapper">
              <div class="function-tabs-shell">
                <el-tabs
                  v-model="functionActiveTab"
                  class="function-detail-tabs"
                  @tab-change="handleFunctionTabChange"
                >
                  <el-tab-pane name="content">
                    <template #label>
                      <span>函数内容</span>
                    </template>
                    <div class="tab-content">
                <!-- ⭐ 如果函数详情已加载，显示对应的视图 -->
                <!-- ⚠️ 重要：只有当 currentFunctionDetail 的 id 或 router 与 currentFunction 匹配时才显示 -->
                <template v-if="currentFunctionDetail && 
                               currentFunction && 
                               (currentFunctionDetail.id === currentFunction.ref_id || 
                                currentFunctionDetail.router === currentFunction.full_code_path)">
                  <!-- 🔥 移除 keep-alive，每次切换函数时重新渲染，保证数据一致性 -->
                  <!-- 🔥 使用 full_code_path 作为 key，确保函数切换时组件正确重建 -->
                  <FormView
                    v-if="currentFunctionDetail.template_type === TEMPLATE_TYPE.FORM"
                    ref="functionFormViewRef"
                    :key="`form-${currentFunction.full_code_path || currentFunction.id}`"
                    :function-detail="currentFunctionDetail"
                  />
                  <TableView
                    v-else-if="currentFunctionDetail.template_type === TEMPLATE_TYPE.TABLE"
                    :key="`table-${currentFunction.full_code_path || currentFunction.id}`"
                    :function-detail="currentFunctionDetail"
                  />
                  <ChartView
                    v-else-if="currentFunctionDetail.template_type === TEMPLATE_TYPE.CHART"
                    :key="`chart-${currentFunction.full_code_path || currentFunction.id}`"
                    :function-detail="currentFunctionDetail"
                  />
                  <div v-else :key="`empty-${currentFunction.full_code_path || currentFunction.id}`" class="function-loading">
                    <el-skeleton :rows="8" animated />
                  </div>
                </template>
                <!-- 如果函数详情未加载且有权限错误，显示权限错误组件 -->
                <PermissionDeniedView
                  v-else-if="hasPermissionError"
                  :key="`permission-denied-${currentFunction.full_code_path || currentFunction.id}`"
                />
                <!-- 如果函数详情未加载且没有权限错误，显示骨架屏 -->
                <div v-else :key="`loading-${currentFunction.full_code_path || currentFunction.id}`" class="function-loading">
                  <el-skeleton :rows="8" animated />
                </div>
                    </div>
                  </el-tab-pane>

                  <el-tab-pane v-if="showFunctionPermissionRequestTab" name="permission">
                    <template #label>
                      <el-badge :value="currentFunction?.pending_count || 0" :hidden="!currentFunction?.pending_count || currentFunction.pending_count === 0" :max="99">
                        <span>权限</span>
                      </el-badge>
                    </template>
                    <div class="tab-content">
                      <el-tabs
                        v-model="functionPermissionTab"
                        class="permission-detail-tabs"
                        @tab-change="handleFunctionPermissionTabChange"
                      >
                        <el-tab-pane name="request">
                          <template #label>
                            <el-badge :value="currentFunction?.pending_count || 0" :hidden="!currentFunction?.pending_count || currentFunction.pending_count === 0" :max="99">
                              <span>审批流</span>
                            </el-badge>
                          </template>
                          <div class="permission-tab-panel">
                            <PermissionRequestList
                              ref="functionPermissionRequestListRef"
                              :resource-path="currentFunction?.full_code_path"
                              resource-type="function"
                              :template-type="currentFunctionDetail?.template_type"
                              :auto-load="functionActiveTab === 'permission' && functionPermissionTab === 'request'"
                            />
                          </div>
                        </el-tab-pane>
                        <el-tab-pane name="manage" label="权限管理">
                          <div class="permission-tab-panel">
                            <PermissionManageList
                              ref="functionPermissionManageListRef"
                              :resource-path="currentFunction?.full_code_path"
                              resource-type="function"
                              :template-type="currentFunctionDetail?.template_type"
                              :auto-load="functionActiveTab === 'permission' && functionPermissionTab === 'manage'"
                            />
                          </div>
                        </el-tab-pane>
                      </el-tabs>
                    </div>
                  </el-tab-pane>

                  <el-tab-pane v-if="showFormOperateLogTab" name="operateLog" label="执行记录">
                    <div class="tab-content">
                      <FormOperateLogSection
                        ref="formOperateLogSectionRef"
                        :full-code-path="currentFunction?.full_code_path || ''"
                        :function-detail="currentFunctionDetail"
                        :auto-load="functionActiveTab === 'operateLog'"
                        @apply-log="handleApplyFormOperateLog"
                      />
                    </div>
                  </el-tab-pane>

                  <el-tab-pane v-if="showScheduledTaskTab" name="scheduledTask" label="定时任务">
                    <div class="tab-content">
                      <ScheduledTaskList
                        :resource-path="currentFunction?.full_code_path"
                        :auto-load="functionActiveTab === 'scheduledTask'"
                        @total-change="onScheduledTaskTotalChange"
                        @open-function-operate-log="openFunctionOperateLog"
                      />
                    </div>
                  </el-tab-pane>
                </el-tabs>
              </div>
            </div>

            <!-- 没有函数 tabs 时，直接显示内容 -->
            <div v-else>
              <!-- ⭐ 如果函数详情已加载，显示对应的视图 -->
              <!-- ⚠️ 重要：只有当 currentFunctionDetail 的 id 或 router 与 currentFunction 匹配时才显示 -->
              <template v-if="currentFunctionDetail && 
                             currentFunction && 
                             (currentFunctionDetail.id === currentFunction.ref_id || 
                              currentFunctionDetail.router === currentFunction.full_code_path)">
                <!-- 🔥 移除 keep-alive，每次切换函数时重新渲染，保证数据一致性 -->
                <!-- 🔥 使用 full_code_path 作为 key，确保函数切换时组件正确重建 -->
                <FormView
                  v-if="currentFunctionDetail.template_type === TEMPLATE_TYPE.FORM"
                  :key="`form-${currentFunction.full_code_path || currentFunction.id}`"
                  :function-detail="currentFunctionDetail"
                />
                <TableView
                  v-else-if="currentFunctionDetail.template_type === TEMPLATE_TYPE.TABLE"
                  :key="`table-${currentFunction.full_code_path || currentFunction.id}`"
                  :function-detail="currentFunctionDetail"
                />
                <ChartView
                  v-else-if="currentFunctionDetail.template_type === TEMPLATE_TYPE.CHART"
                  :key="`chart-${currentFunction.full_code_path || currentFunction.id}`"
                  :function-detail="currentFunctionDetail"
                />
                <div v-else :key="`empty-${currentFunction.full_code_path || currentFunction.id}`" class="function-loading">
                  <el-skeleton :rows="8" animated />
                </div>
              </template>
              <!-- 如果函数详情未加载且有权限错误，显示权限错误组件 -->
              <PermissionDeniedView
                v-else-if="hasPermissionError"
                :key="`permission-denied-${currentFunction.full_code_path || currentFunction.id}`"
              />
              <!-- 如果函数详情未加载且没有权限错误，显示骨架屏 -->
              <div v-else :key="`loading-${currentFunction.full_code_path || currentFunction.id}`" class="function-loading">
                <el-skeleton :rows="8" animated />
              </div>
            </div>
          </div>
        </div>
        <div v-else class="empty-state">
          <p>请在左侧选择功能或目录</p>
        </div>
      </div>

      <!-- 右侧面板：工作台会话（仅当前节点） -->
      <div
        v-if="workstationContext && showRightSidebar"
        class="right-sidebar"
      >
        <div class="right-sidebar-session-panel">
          <div class="right-session-header">
            <el-icon :size="16" color="var(--el-color-primary)"><FolderOpened /></el-icon>
            <span class="right-session-dir">{{ workstationContext.dirName }}</span>
          </div>
          <div class="right-session-tabs">
            <div :class="['right-tab', { active: rightTab === 'all' }]" @click="rightTab = 'all'">
              全部
            </div>
            <div :class="['right-tab', { active: rightTab === 'running' }]" @click="rightTab = 'running'">
              执行中
              <span v-if="rightSidebarRunningCount > 0" class="right-tab-badge">{{ rightSidebarRunningCount }}</span>
            </div>
            <div :class="['right-tab', { active: rightTab === 'finished' }]" @click="rightTab = 'finished'">
              已结束
            </div>
          </div>

          <el-input
            v-model="rightSessionSearchKeyword"
            class="right-session-search"
            placeholder="搜索会话…"
            clearable
            :prefix-icon="Search"
          />

          <div class="right-session-list" v-loading="rightSidebarSessionsLoading">
            <div
              v-for="s in filteredRightSessions"
              :key="s.session_id"
              :class="['right-session-card', { generating: s.status === 'generating' }]"
              @click="openSessionInMini(s)"
            >
              <div class="right-session-card-head">
                <el-icon v-if="s.status === 'generating'" class="is-loading" :size="12" color="var(--el-color-primary)"><Loading /></el-icon>
                <span class="right-session-card-title">{{ s.title || '未命名会话' }}</span>
              </div>
              <div v-if="s.user" class="right-session-card-user">
                <UserDisplay :username="s.user" mode="simple" size="small" />
              </div>
              <div class="right-session-card-meta">
                <el-tag v-if="s.status === 'generating'" type="primary" size="small" effect="light">执行中</el-tag>
                <el-tag v-else-if="s.status === 'done'" type="success" size="small" effect="plain">已完成</el-tag>
                <el-tag v-else-if="s.status === 'cancelled'" type="info" size="small" effect="plain">已取消</el-tag>
                <span class="right-session-time">{{ formatRelativeTime(s.updated_at) }}</span>
              </div>
              <div v-if="s.status === 'generating'" class="right-session-card-actions">
                <el-button size="small" link type="danger" @click.stop="handleCancelTask(s)" :loading="cancellingTaskId === s.session_id">停止</el-button>
              </div>
            </div>
            <div v-if="filteredRightSessions.length === 0 && !rightSidebarSessionsLoading" class="right-session-empty">
              <el-empty :description="rightSessionSearchKeyword ? '无匹配会话' : (rightTab === 'running' ? '暂无执行中的会话' : rightTab === 'finished' ? '暂无已结束的会话' : '暂无会话记录')" :image-size="48" />
            </div>
          </div>

          <div class="right-session-footer">
            <el-button type="primary" @click="openNewMiniWs()" :icon="ChatDotRound" class="right-new-session-btn">
              新增会话
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <WorkspaceCreateAppDialog
      v-model:visible="createAppDialogVisible"
      :form="createAppForm"
      :creating="creatingApp"
      :admins-field="createAppAdminsField"
      :admins-field-value="createAppAdminsFieldValue"
      @update-admins="handleCreateAppAdminsChange"
      @submit="submitCreateApp"
      @close="resetCreateAppForm"
    />

    <!-- 详情抽屉 -->
    <TableRowDetailDrawer
      v-model:visible="detailDrawerVisible"
      v-model:mode="detailDrawerMode"
      :title="detailDrawerTitle"
      :fields="detailFields"
      :row-data="detailRowData"
      :table-data="detailTableData"
      :current-index="currentDetailIndex"
      :can-edit="(currentFunctionDetail?.callbacks?.includes('OnTableUpdateRow') || false) && canUpdateTable"
      :edit-function-detail="editFunctionDetail"
      :current-function-detail="currentFunctionDetail"
      :user-info-map="detailUserInfoMap"
      :submitting="drawerSubmitting"
      :current-function="currentFunction"
      ref="detailDrawerRef"
      @navigate="handleNavigateDetail"
      @submit="(formRendererRef) => submitDrawerEdit(formRendererRef)"
      @close="handleDetailDrawerClose"
    />

    <!-- 创建文档节点对话框 -->
    <el-dialog
      v-model="createDocsDialogVisible"
      :title="currentDocsParentNode ? `在「${currentDocsParentNode.name || currentDocsParentNode.code}」下创建文档` : '创建文档'"
      width="600px"
      :close-on-click-modal="false"
      @close="handleCloseCreateDocsDialog"
    >
      <el-form :model="createDocsForm" label-width="120px">
        <el-form-item label="文档名称" required>
          <el-input
            v-model="createDocsForm.name"
            placeholder="请输入文档名称"
            maxlength="100"
            show-word-limit
            clearable
          />
        </el-form-item>
        <el-form-item label="文档代码" required>
          <el-input
            v-model="createDocsForm.code"
            placeholder="英文，如 readme"
            maxlength="50"
            show-word-limit
            clearable
            @input="createDocsForm.code = createDocsForm.code.toLowerCase().replace(/[^a-z0-9_]/g, '')"
          >
            <template #suffix>
              <span class="create-docs-code-suffix">.docs</span>
            </template>
          </el-input>
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            只能包含小写字母、数字和下划线，保存后自动带后缀 .docs
          </div>
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="createDocsForm.description"
            type="textarea"
            placeholder="请输入文档描述（可选）"
            :rows="3"
            maxlength="500"
            show-word-limit
          />
        </el-form-item>
        <el-form-item label="标签">
          <el-input
            v-model="createDocsForm.tags"
            placeholder="请输入标签，多个标签用逗号分隔（可选）"
            maxlength="200"
            clearable
          />
        </el-form-item>
        <el-form-item label="文档内容" required>
          <el-input
            v-model="createDocsForm.content"
            type="textarea"
            placeholder="请输入文档内容（支持 Markdown 格式）"
            :rows="15"
            maxlength="50000"
            show-word-limit
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            支持 Markdown 格式，可以使用标题、列表、代码块、链接等语法
          </div>
        </el-form-item>
        <el-form-item label="文档摘要">
          <el-input
            v-model="createDocsForm.summary"
            type="textarea"
            placeholder="请输入文档摘要（可选）"
            :rows="2"
            maxlength="500"
            show-word-limit
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createDocsDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmitCreateDocs" :loading="creatingDocs">
            创建
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 创建讨论区（版块）对话框 - 封装组件 -->
    <CreateBoardDialog
      v-if="createBoardDialogVisible"
      v-model="createBoardDialogVisible"
      :current-app="currentApp"
      :parent-node="currentBoardParentNode"
      @success="afterCreateBoard"
    />

    <WorkspaceCreateDirectoryDialog
      v-model:visible="createDirectoryDialogVisible"
      :parent-node="currentParentNode"
      :form="createDirectoryForm"
      :creating="creatingDirectory"
      :admins-field="adminsField"
      :admins-field-value="adminsFieldValue"
      @update-admins="handleAdminsChange"
      @submit="handleSubmitCreateDirectory"
      @close="handleCloseCreateDirectoryDialog"
    />

    <!-- 发布到应用中心对话框 -->
    <PublishToHubDialog
      v-if="publishToHubDialogVisible"
      v-model="publishToHubDialogVisible"
      :selected-node="publishSelectedNode"
      :current-app="currentApp || undefined"
      @success="handlePublishSuccess"
    />
    <PushToHubDialog
      v-if="pushToHubDialogVisible"
      v-model="pushToHubDialogVisible"
      :selected-node="pushSelectedNode"
      :current-app="currentApp || undefined"
      @success="handlePushSuccess"
    />
    <PullFromHubDialog
      v-if="pullFromHubDialogVisible"
      v-model="pullFromHubDialogVisible"
      :current-app="currentApp || undefined"
      :initial-hub-link="pastedHubLink"
      :initial-target-path="pullFromHubTargetPath"
      :initial-target-name="pullFromHubTargetName"
      @success="handlePullSuccess"
    />

    <!-- 变更记录对话框 -->
    <DirectoryUpdateHistoryDialog
      v-if="updateHistoryDialogVisible"
      v-model="updateHistoryDialogVisible"
      :mode="updateHistoryMode"
      :app-id="updateHistoryAppId"
      :app-version="updateHistoryAppVersion"
      :full-code-path="updateHistoryFullCodePath"
    />

    <!-- 导入 Go 文件：隐藏的 file input，选中的 .go 会写入当前目录 -->
    <input
      ref="importGoFileInputRef"
      type="file"
      accept=".go"
      multiple
      class="hidden"
      @change="onImportGoFilesSelected"
    />


    <!-- 多个 Mini 浮动工作台 -->
    <MiniWorkstation
      v-for="mini in miniWsList"
      :key="mini.id"
      :visible="mini.visible"
      :full-code-path="mini.fullCodePath"
      :dir-name="mini.dirName"
      :initial-session-id="mini.initialSessionId"
      :initial-offset="mini.offset"
      :initial-position="mini.initialPosition"
      :initial-maximized="mini.initialMaximized"
      @minimize="handleMiniMinimize(mini.id)"
      @close="handleMiniRemove(mini.id)"
      @maximize-change="handleMiniMaximizeChange"
      @tool-call-ok="handleWorkstationToolCallOk"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox, ElNotification, ElDialog, ElForm, ElFormItem, ElInput, ElButton, ElIcon, ElSkeleton } from 'element-plus'
import { InfoFilled, ArrowLeft, ArrowRight, ChatDotRound, Loading, FolderOpened, Search } from '@element-plus/icons-vue'
import { serviceFactory } from '../../infrastructure/factories'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import ServiceTreePanel from '@/architecture/presentation/components/ServiceTreePanel.vue'
import WorkspaceHeader from '../components/WorkspaceHeader.vue'
import FunctionBreadcrumb from '../components/FunctionBreadcrumb.vue'
import TableRowDetailDrawer from '../components/TableRowDetailDrawer.vue'
import WorkspaceCreateAppDialog from '../components/WorkspaceCreateAppDialog.vue'
import WorkspaceCreateDirectoryDialog from '../components/WorkspaceCreateDirectoryDialog.vue'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import PermissionRequestList from '@/shared/components/permission/PermissionRequestList.vue'
import PermissionManageList from '@/shared/components/permission/PermissionManageList.vue'
import ScheduledTaskList from '../components/ScheduledTaskList.vue'
import type { App } from '../../domain/services/WorkspaceDomainService'
import type { FieldConfig, FieldValue, FunctionDetail } from '@/architecture/domain/types'
import { WidgetType } from '@/core/constants/widget'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/types'
// 🔥 导入 Composable
import { useWorkspaceRouting } from '../composables/useWorkspaceRouting'
import { useWorkspaceDetail } from '../composables/useWorkspaceDetail'
import { useWorkspaceApp } from '../composables/useWorkspaceApp'
import { useWorkspaceServiceTree } from '../composables/useWorkspaceServiceTree'
import { useWorkspaceFunctionTabs } from '../composables/useWorkspaceFunctionTabs'
import { useWorkspaceSidebarSessions } from '../composables/useWorkspaceSidebarSessions'
import { useWorkspaceMiniWorkstations } from '../composables/useWorkspaceMiniWorkstations'
import { useWorkspaceNodeActions } from '../composables/useWorkspaceNodeActions'
import { useWorkspaceNodeNavigation } from '../composables/useWorkspaceNodeNavigation'
import { useWorkspaceNodeToolActions } from '../composables/useWorkspaceNodeToolActions'
import { useWorkspaceUiEffects } from '../composables/useWorkspaceUiEffects'
import { useWorkspaceViewLifecycle } from '../composables/useWorkspaceViewLifecycle'
import { findNodeByPath, findNodeById } from '../utils/workspaceUtils'
import { getScopedFieldQueryValue } from '@/utils/queryFieldNamespace'
import { useAfterCreateNode } from '../composables/useAfterCreateNode'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { hasPermission, TablePermission } from '@/utils/permission'
import { usePermissionErrorStore } from '@/stores/permissionError'
import { createStringFieldValue, createWidgetFieldConfig, extractStringFieldRaw } from '@/utils/widgetFieldHelpers'

const route = useRoute()
const router = useRouter()
const FormView = defineAsyncComponent(() => import('./FormView.vue'))
const TableView = defineAsyncComponent(() => import('./TableView.vue'))
const ChartView = defineAsyncComponent(() => import('./ChartView.vue'))
const DocView = defineAsyncComponent(() => import('../components/DocView.vue'))
const BoardView = defineAsyncComponent(() => import('../components/BoardView.vue'))
const PackageDetailView = defineAsyncComponent(() => import('../components/PackageDetailView.vue'))
const PermissionDeniedView = defineAsyncComponent(() => import('../components/PermissionDeniedView.vue'))
const FormOperateLogSection = defineAsyncComponent(() => import('../components/FormOperateLogSection.vue'))
const MiniWorkstation = defineAsyncComponent(() => import('../components/MiniWorkstation.vue'))
const CreateBoardDialog = defineAsyncComponent(() => import('../components/CreateBoardDialog.vue'))
const PublishToHubDialog = defineAsyncComponent(() => import('@/shared/components/PublishToHubDialog.vue'))
const PushToHubDialog = defineAsyncComponent(() => import('@/shared/components/PushToHubDialog.vue'))
const PullFromHubDialog = defineAsyncComponent(() => import('@/shared/components/PullFromHubDialog.vue'))
const DirectoryUpdateHistoryDialog = defineAsyncComponent(() => import('@/shared/components/DirectoryUpdateHistoryDialog.vue'))

// 依赖注入（使用 IServiceProvider 接口，遵循依赖倒置原则）
const serviceProvider: IServiceProvider = serviceFactory
const applicationService = serviceProvider.getWorkspaceApplicationService()
const domainService = serviceProvider.getWorkspaceDomainService()

// 从状态管理器获取状态
const serviceTree = computed(() => domainService.getServiceTree())
const currentFunction = computed(() => domainService.getCurrentFunction())
const currentAppFromState = computed(() => domainService.getCurrentApp())

// ⭐ 需要自动展开的节点ID列表（从后端返回）
const expandedKeys = ref<number[]>([])

// 🔥 不再使用 Tab 功能，简化系统

function normalizeApp(app: Partial<AppType> & Pick<AppType, 'id' | 'user' | 'code' | 'name'>): AppType {
  return {
    id: app.id,
    user: app.user,
    code: app.code,
    name: app.name,
    nats_id: app.nats_id ?? 0,
    host_id: app.host_id ?? 0,
    status: app.status ?? 'enabled',
    type: app.type,
    version: app.version ?? '',
    is_public: app.is_public ?? false,
    admins: app.admins ?? '',
    show_only_permitted: app.show_only_permitted,
    created_at: app.created_at ?? '',
    updated_at: app.updated_at ?? ''
  }
}

function getCurrentAppForTreeLoad(): App | null {
  return currentApp.value ? normalizeApp(currentApp.value) : null
}

const currentApp = computed<AppType | null>(() => {
  const app = currentAppFromState.value
  if (!app) return null
  // 从 appList 中查找对应的应用（确保使用最新的应用数据）
  const foundApp = appList.value.find((a: AppType) => a.id === app.id || (a.user === app.user && a.code === app.code))
  return foundApp ? normalizeApp(foundApp) : normalizeApp(app)
})

const {
  appList,
  loadingApps,
  createAppDialogVisible,
  creatingApp,
  createAppForm,
  loadAppList,
  handleSwitchApp: appHandleSwitchApp,
  showCreateAppDialog,
  resetCreateAppForm,
  submitCreateApp: appSubmitCreateApp,
  handleUpdateApp,
  handleDeleteApp: appHandleDeleteApp
} = useWorkspaceApp()

const createAppAdminsField = createWidgetFieldConfig({
  code: 'create_app_admins',
  name: '管理员',
  widgetType: WidgetType.USERS
})

const createAppAdminsFieldValue = computed(() =>
  createStringFieldValue(createAppAdminsField, createAppForm.value.admins, {
    display: (createAppForm.value.admins || '').split(',').map(s => s.trim()).filter(Boolean).join(', ')
  })
)

function handleCreateAppAdminsChange(value: FieldValue) {
  createAppForm.value.admins = extractStringFieldRaw(value)
}

const {
  createDirectoryDialogVisible,
  creatingDirectory,
  currentParentNode,
  createDirectoryForm,
  handleCreateDirectory: serviceTreeHandleCreateDirectory,
  resetCreateDirectoryForm,
  handleSubmitCreateDirectory: serviceTreeHandleSubmitCreateDirectory,
  expandCurrentRoutePath: serviceTreeExpandCurrentRoutePath,
} = useWorkspaceServiceTree()

const adminsField = createWidgetFieldConfig({
  code: 'admins',
  name: '管理员',
  widgetType: WidgetType.USERS
})

const adminsFieldValue = computed(() =>
  createStringFieldValue(adminsField, createDirectoryForm.value.admins, {
    display: (createDirectoryForm.value.admins || '').split(',').map(s => s.trim()).filter(Boolean).join(', ')
  })
)

// 处理管理员字段变化
function handleAdminsChange(value: FieldValue) {
  createDirectoryForm.value.admins = extractStringFieldRaw(value)
}

// 🔥 移除缓存后，通过事件获取函数详情
const currentFunctionDetail = ref<FunctionDetail | null>(null)

const {
  buildWorkspacePath,
  handleNodeClick,
  handleBreadcrumbNodeClick,
  backToList
} = useWorkspaceNodeNavigation({
  route,
  currentFunction: () => currentFunction.value,
  triggerNodeClick: (node) => applicationService.triggerNodeClick(node)
})

const {
  detailDrawerVisible,
  detailDrawerTitle,
  detailRowData,
  detailFields,
  detailOriginalRow,
  detailDrawerMode,
  drawerSubmitting,
  detailUserInfoMap,
  detailTableData,
  currentDetailIndex,
  editFunctionDetail,
  toggleDrawerMode,
  handleNavigateDetail,
  submitDrawerEdit,
  handleDetailDrawerClose,
  openDetailDrawer,
  setupUrlWatch
} = useWorkspaceDetail({
  currentFunctionDetail: () => currentFunctionDetail.value,
  currentFunction: () => currentFunction.value
})

const {
  syncRouteToTab,
  loadAppFromRoute: routingLoadAppFromRoute,
  setupRouteWatch
} = useWorkspaceRouting({
  serviceTree: () => serviceTree.value,
  currentApp: () => currentApp.value,
  appList: () => appList.value,
  loadAppList,
  findNodeByPath,
  expandCurrentRoutePath: () => serviceTreeExpandCurrentRoutePath(
    () => serviceTree.value,
    () => serviceTreePanelRef.value,
    () => currentApp.value
  )
})

// 🔥 Tab 点击处理已移除（直接使用 v-model，避免双重触发）
// const handleTabClick = tabsHandleTabClick


// 🔥 queryTab：当前激活的Tab模式（用于路由查询参数，控制 create/edit 等模式）
const queryTab = computed(() => (route.query._tab as string) || 'run')

// 🔥 编辑模式相关
const editRowId = computed(() => {
  const id = route.query.id || route.query._id
  return id ? Number(id) : null
})

// 🔥 编辑模式的初始数据（从 URL 参数提取）
const editInitialData = computed(() => {
  const initialData: Record<string, any> = {}
  const query = route.query
  
  // 如果有 id 参数，添加到 initialData
  if (editRowId.value) {
    const idField = currentFunctionDetail.value?.request?.find((f: FieldConfig) => 
      f.code.toLowerCase() === 'id' || f.widget?.type === 'number'
    )
    if (idField) {
      initialData[idField.code] = editRowId.value
    }
  }
  
  // 遍历所有查询参数，如果字段在 request 中，添加到 initialData
  if (currentFunctionDetail.value?.request) {
    currentFunctionDetail.value.request.forEach((field: FieldConfig) => {
      const fieldCode = field.code
      // 跳过 _ 开头的参数（系统参数）
      if (fieldCode.startsWith('_')) return
      
      const value = getScopedFieldQueryValue(query, fieldCode, 'form')
      if (value !== undefined && value !== null && value !== '') {
        // 🔥 类型转换：根据字段类型转换值
        if (field.data?.type === 'int' || field.data?.type === 'integer') {
          const intValue = parseInt(String(value), 10)
          if (!isNaN(intValue)) {
            initialData[fieldCode] = intValue
          }
        } else if (field.data?.type === 'float' || field.data?.type === 'number') {
          const floatValue = parseFloat(String(value))
          if (!isNaN(floatValue)) {
            initialData[fieldCode] = floatValue
          }
        } else if (field.data?.type === 'bool' || field.data?.type === 'boolean') {
          const strValue = String(value)
          const numValue = typeof value === 'number' ? value : Number(strValue)
          const boolValue = typeof value === 'boolean' ? value : false
          initialData[fieldCode] = strValue === 'true' || strValue === '1' || numValue === 1 || boolValue
        } else {
          initialData[fieldCode] = value
        }
      }
    })
  }
  
  return initialData
})


// ServiceTreePanel 引用（用于展开路径）
const serviceTreePanelRef = ref<InstanceType<typeof ServiceTreePanel> | null>(null)
const workspaceHeaderRef = ref<InstanceType<typeof WorkspaceHeader> | null>(null)

// 左侧服务目录树显示状态
const showLeftSidebar = ref(true)

// 右侧会话面板显示状态
const showRightSidebar = ref(true)

const {
  functionActiveTab,
  functionPermissionTab,
  functionFormViewRef,
  functionPermissionRequestListRef,
  functionPermissionManageListRef,
  formOperateLogSectionRef,
  showScheduledTaskTab,
  showFunctionPermissionRequestTab,
  showFormOperateLogTab,
  showFunctionTabsWrapper,
  handleFunctionTabChange,
  handleFunctionPermissionTabChange,
  handleApplyFormOperateLog,
  openFunctionOperateLog,
  onScheduledTaskTotalChange,
  syncFunctionTabQuery,
  activateScheduledTaskTab
} = useWorkspaceFunctionTabs({
  route,
  router,
  currentFunction,
  currentFunctionDetail
})

useWorkspaceViewLifecycle({
  route,
  router,
  currentFunction: () => currentFunction.value,
  currentFunctionDetail: () => currentFunctionDetail.value,
  setCurrentFunctionDetail: (detail) => {
    currentFunctionDetail.value = detail
  },
  clearPermissionError: () => permissionErrorStore.clearError(),
  expandedKeys,
  currentApp: () => currentApp.value,
  serviceTree: () => serviceTree.value,
  loadAppFromRoute: routingLoadAppFromRoute,
  setupRouteWatch,
  activateScheduledTaskTab,
  expandCurrentRoutePath: () => serviceTreeExpandCurrentRoutePath(
    () => serviceTree.value,
    () => serviceTreePanelRef.value,
    () => currentApp.value
  ),
  queryTab: () => queryTab.value,
  loadNodeDetail: (node) => applicationService.handleNodeClick(node),
  updateAppInfo: (app) => {
    const index = appList.value.findIndex((item: AppType) => item.code === app.code)
    if (index !== -1) {
      appList.value[index] = { ...appList.value[index], ...app }
    }
  },
  findNodeByPath,
  openWorkspaceListDialog: () => workspaceHeaderRef.value?.openWorkspaceListDialog(true)
})

// 切换左侧边栏显示
const toggleLeftSidebar = () => {
  showLeftSidebar.value = !showLeftSidebar.value
  // 保存到 localStorage 持久化
  localStorage.setItem('workspace-left-sidebar', String(showLeftSidebar.value))
}

// 切换右侧边栏显示
const toggleRightSidebar = () => {
  showRightSidebar.value = !showRightSidebar.value
  // 保存到 localStorage 持久化
  localStorage.setItem('workspace-right-sidebar', String(showRightSidebar.value))
}

/** 工作台上下文：点击什么节点就用什么节点的 full_code_path */
const workstationContext = computed(() => {
  const node = currentFunction.value
  if (!node?.full_code_path) return null
  const path = (node.full_code_path || '').replace(/\/+$/g, '')
  if (!path) return null
  const name = node.name || path.split('/').pop() || '工作台'
  return { fullCodePath: path, dirName: name }
})

const {
  miniWsList,
  openNewMiniWs,
  handleMiniMinimize,
  handleMiniRemove,
  handleMiniMaximizeChange,
  handleWorkspaceOpenWorkstation,
  initializeFromRoute: initializeMiniWorkstationsFromRoute,
} = useWorkspaceMiniWorkstations({
  route,
  router,
  workstationContext,
  buildWorkspacePath: (fullCodePath: string) => buildWorkspacePath(fullCodePath),
})

const {
  sessions: rightSidebarSessions,
  sessionsLoading: rightSidebarSessionsLoading,
  activeTab: rightTab,
  sessionSearchKeyword: rightSessionSearchKeyword,
  cancellingTaskId,
  runningCount: rightSidebarRunningCount,
  filteredSessions: filteredRightSessions,
  openSession: openSessionInMini,
  formatRelativeTime,
  handleCancelTask
} = useWorkspaceSidebarSessions({
  workstationContext,
  sidebarVisible: showRightSidebar,
  onOpenSession: (session) => {
    openNewMiniWs(session.session_id, session.full_code_path)
  }
})

// ⭐ 权限检查：是否有表格更新权限
const canUpdateTable = computed(() => {
  const node = currentFunction.value
  if (!node) return true  // 如果没有节点信息，默认允许（向后兼容）
  return hasPermission(node, TablePermission.update)
})

// ⭐ 权限错误状态
const permissionErrorStore = usePermissionErrorStore()
const hasPermissionError = computed(() => {
  return permissionErrorStore.currentError !== null
})



// 转换 loadingTree 为 boolean (避免 computed 类型问题)
const loading = computed(() => domainService.isLoading())
// 🔥 处理创建目录（使用 Composable）
const handleCreateDirectory = (parentNode?: ServiceTreeType) => {
  serviceTreeHandleCreateDirectory(parentNode || null, () => currentApp.value)
}

const handleSubmitCreateDirectory = async () => {
  await serviceTreeHandleSubmitCreateDirectory(() => currentApp.value)
}

// 处理关闭创建目录对话框
const handleCloseCreateDirectoryDialog = () => {
  resetCreateDirectoryForm(() => currentApp.value)
}


// 处理刷新服务树（复制粘贴后需要刷新）
const handleRefreshTree = async () => {
  const app = getCurrentAppForTreeLoad()
  if (app) {
    await domainService.loadServiceTree(app)
  }
}

// 创建节点后的统一处理：刷新树 + 选中新节点（文档/讨论区复用，需在 handleRefreshTree 之后定义）
const afterCreateNode = useAfterCreateNode({
  handleRefreshTree,
  serviceTree: () => serviceTree.value,
  findNodeById,
  handleNodeClick
})
const afterCreateBoard = afterCreateNode

const {
  createDocsDialogVisible,
  creatingDocs,
  currentDocsParentNode,
  createDocsForm,
  createBoardDialogVisible,
  currentBoardParentNode,
  handleCreateDocs,
  handleSubmitCreateDocs,
  handleCloseCreateDocsDialog,
  handleCreateBoard,
  handleDeleteBoard,
  handleDeleteDoc,
  handleDocDeleted,
  handleDeleteDirectory,
  handleDeleteFunction
} = useWorkspaceNodeActions({
  route,
  router,
  currentApp,
  currentFunction,
  domainService,
  handleRefreshTree,
  afterCreateNode
})

const {
  publishToHubDialogVisible,
  publishSelectedNode,
  pushToHubDialogVisible,
  pushSelectedNode,
  pullFromHubDialogVisible,
  pastedHubLink,
  pullFromHubTargetPath,
  pullFromHubTargetName,
  updateHistoryDialogVisible,
  updateHistoryMode,
  updateHistoryAppId,
  updateHistoryAppVersion,
  updateHistoryFullCodePath,
  importGoFileInputRef,
  handleImportGoFiles,
  onImportGoFilesSelected,
  handlePublishToHub,
  handlePushToHub,
  openPullFromHubDialog,
  handleUpdateHistory,
  handlePublishSuccess,
  handlePushSuccess,
  handlePullSuccess
} = useWorkspaceNodeToolActions({
  currentApp,
  handleRefreshTree
})

// 会改变服务目录结构的工具名（创建目录、写文档、写代码、编译工作空间）
const TREE_AFFECTING_TOOLS = ['create_directory', 'write_doc', 'write_go_file', 'build_workspace']

// 工作台工具调用成功时：若为改树工具则刷新服务树
const handleWorkstationToolCallOk = (payload: { name: string }) => {
  if (payload?.name && TREE_AFFECTING_TOOLS.includes(payload.name)) {
    handleRefreshTree()
  }
}
// 🔥 返回列表（从 create/edit 模式返回）
// 🔥 处理新增提交（通过 FormView 的提交按钮，这里只是占位）
const handleCreateSubmit = async () => {
  // FormView 内部已经有提交逻辑，这里不需要额外处理
  // 如果需要，可以通过 ref 或事件总线来触发 FormView 的提交
  ElNotification.info({
    title: '提示',
    message: '请使用表单内的提交按钮提交数据'
  })
}

// 🔥 处理编辑提交（通过 FormView 的提交按钮，这里只是占位）
const handleEditSubmit = async () => {
  // FormView 内部已经有提交逻辑，这里不需要额外处理
  // 如果需要，可以通过 ref 或事件总线来触发 FormView 的提交
  ElNotification.info({
    title: '提示',
    message: '请使用表单内的提交按钮提交数据'
  })
}

// 🔥 切换应用（使用 Composable）
const handleSwitchApp = async (app: AppType): Promise<void> => {
  await appHandleSwitchApp(app, () => currentApp.value)
}

// 🔥 提交创建应用（使用 Composable）
const submitCreateApp = async (): Promise<void> => {
  await appSubmitCreateApp(() => currentApp.value)
}

// 🔥 删除应用（使用 Composable）
const handleDeleteApp = async (app: AppType): Promise<void> => {
  await appHandleDeleteApp(app, () => currentApp.value)
}

useWorkspaceUiEffects({
  currentApp: () => currentApp.value,
  showLeftSidebar,
  showRightSidebar,
  openPullFromHubDialog,
  openDetailDrawer,
  setupUrlWatch,
  handleWorkspaceOpenWorkstation,
  initializeMiniWorkstationsFromRoute
})

</script>

<style scoped lang="scss">
.hidden {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
  pointer-events: none;
}

.create-docs-code-suffix {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  padding-right: 4px;
}

.workspace-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.workspace-view {
  display: flex;
  flex: 1;
  overflow: hidden; /* 防止双滚动条 */
  position: relative;
}

.function-content-wrapper {
  flex: 1;
  overflow: hidden; /* 🔥 外层容器隐藏溢出，内层处理滚动 */
  display: flex;
  flex-direction: column;
  min-height: 0; /* 🔥 关键：允许 flex 子元素缩小 */
}

.function-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0; /* 🔥 关键：允许 flex 子元素缩小 */
  overflow: hidden; /* 🔥 外层容器隐藏溢出，内层处理滚动 */
  
  // 当有 tab 结构时，需要特殊处理
  .function-tabs-wrapper {
    flex: 1;
    min-height: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }
  
  // 当没有 tab 结构时，直接显示内容（允许滚动）
  > div:not(.function-tabs-wrapper) {
    flex: 1;
    min-height: 0;
    overflow-y: auto !important;
    overflow-x: hidden;
    -webkit-overflow-scrolling: touch;
    height: 0; /* 🔥 关键：配合 flex: 1 和 min-height: 0，让滚动容器正确计算高度 */
  }
}

// 函数 tab 包装器（已在 function-content 中定义，这里不需要重复）

.function-tabs-wrapper {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0 16px 16px;
}

.function-tabs-shell {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0 16px 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 16px;
  background: var(--el-bg-color);
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.05);
}

.function-detail-tabs,
.permission-detail-tabs {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.function-detail-tabs :deep(.el-tabs__header) {
  margin: 14px 0 12px;
  flex-shrink: 0;
}

.permission-detail-tabs :deep(.el-tabs__header) {
  margin: 2px 0 14px;
  flex-shrink: 0;
}

.function-detail-tabs :deep(.el-tabs__nav-wrap::after),
.permission-detail-tabs :deep(.el-tabs__nav-wrap::after) {
  background-color: var(--el-border-color-extra-light);
}

.function-detail-tabs :deep(.el-tabs__item.is-active) {
  font-weight: 600;
}

.function-detail-tabs :deep(.el-tabs__content) {
  background: transparent;
}

.function-detail-tabs :deep(.el-tabs__item),
.permission-detail-tabs :deep(.el-tabs__item) {
  font-size: 14px;
}

.permission-detail-tabs :deep(.el-tabs__item) {
  font-size: 13px;
}

.function-detail-tabs :deep(.el-tabs__content),
.permission-detail-tabs :deep(.el-tabs__content) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.function-detail-tabs :deep(.el-tab-pane),
.permission-detail-tabs :deep(.el-tab-pane) {
  height: 100%;
}

.function-detail-tabs :deep(.el-badge),
.permission-detail-tabs :deep(.el-badge) {
  position: relative;
  display: inline-block;
}

.function-detail-tabs :deep(.el-badge__content),
.permission-detail-tabs :deep(.el-badge__content) {
  font-size: 11px;
  height: 16px;
  line-height: 16px;
  min-width: 16px;
  padding: 0 5px;
  border-radius: 8px;
}

.permission-tab-panel {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

/* 保留旧的类名以兼容（如果还有地方使用） */
.tabs-content-wrapper {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.tab-content {
  flex: 1;
  overflow-y: auto !important;
  overflow-x: hidden;
  min-height: 0;
  height: 100%;
  -webkit-overflow-scrolling: touch;
}

/* 左下角：隐藏/显示目录按钮 */
.sidebar-toggle-bottom-left {
  position: absolute;
  bottom: 12px;
  left: 12px;
  z-index: 100;
}

.sidebar-toggle-bottom-left .sidebar-toggle-btn {
  width: 32px;
  height: 32px;
  min-width: 32px;
  padding: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  transition: all 0.2s;

  &:hover {
    color: var(--el-color-primary);
    border-color: var(--el-color-primary);
    background: var(--el-fill-color-light);
  }

  .el-icon {
    font-size: 16px;
  }
}

.left-sidebar {
  width: 300px;
  min-width: 300px;
  border-right: 1px solid var(--el-border-color);
  transition: all 0.3s ease;
  overflow: hidden;
  display: flex;
  flex-direction: column;

  &.sidebar-collapsed {
    width: 0;
    min-width: 0;
    overflow: hidden;
    border-right: none;
  }
}

.left-sidebar-tree {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
}

.function-renderer {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
  position: relative;
}

/* 讨论区/文档/目录主内容区：可滚动；右侧留白避免被「板块说明」按钮挡住 */
.main-content-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  -webkit-overflow-scrolling: touch;
  padding-right: 130px; /* 为右上角「板块说明」按钮留出空间，避免挡住发帖等操作 */
}

// 右侧边栏控制按钮
.sidebar-controls {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 10;
  
  .right-controls {
    display: flex;
    gap: 8px;
  }
  
  .sidebar-toggle {
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    
    &:hover {
      background: var(--el-fill-color-light);
      border-color: var(--el-color-primary);
    }
  }
}

// 右侧面板：工作台会话
.right-sidebar {
  width: 280px;
  min-width: 280px;
  background-color: var(--el-bg-color);
  border-left: 1px solid var(--el-border-color-light);
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.right-sidebar-session-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}
.right-session-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  flex-shrink: 0;
}
.right-session-dir {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.right-session-tabs {
  display: flex;
  align-items: center;
  gap: 0;
  padding: 0 6px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  flex-shrink: 0;
}
.right-tab {
  padding: 8px 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
  white-space: nowrap;
  &:hover { color: var(--el-color-primary); }
  &.active {
    color: var(--el-color-primary);
    font-weight: 500;
    border-bottom-color: var(--el-color-primary);
  }
}
.right-tab-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  margin-left: 4px;
  font-size: 10px;
  line-height: 1;
  color: #fff;
  background: var(--el-color-danger);
  border-radius: 8px;
}
.right-session-search {
  flex-shrink: 0;
  padding: 6px 8px 4px;
}
.right-session-search :deep(.el-input__wrapper) {
  border-radius: 6px;
}
.right-session-card-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 4px;
}
.right-session-footer {
  flex-shrink: 0;
  padding: 10px 12px;
  border-top: 1px solid var(--el-border-color-extra-light);
}
.right-new-session-btn {
  width: 100%;
}
.right-session-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}
.right-session-card {
  padding: 10px 12px;
  margin-bottom: 6px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  background: var(--el-bg-color);
  cursor: pointer;
  transition: all 0.15s;
  &:hover {
    border-color: var(--el-color-primary);
    background: var(--el-fill-color-lighter);
  }
  &.generating {
    border-left: 3px solid var(--el-color-primary);
  }
}
.right-session-card-head {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 6px;
}
.right-session-card-user {
  margin-bottom: 4px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.right-session-card-user :deep(.user-display-wrapper) {
  display: inline-flex;
}
.right-session-card-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}
.right-session-card-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--el-text-color-placeholder);
}
.right-session-time {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
}
.right-session-empty {
  padding: 24px 8px;
  text-align: center;
}

.ai-chat-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

/* 工作台抽屉：右侧滑出，可折叠为窄条 */
.workstation-drawer .el-drawer__header {
  margin-bottom: 0;
  padding: 4px 12px;
}
.workstation-drawer-header {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.workstation-drawer-header--compact {
  justify-content: flex-end;
}
.workstation-drawer-header .drawer-actions {
  flex-shrink: 0;
}
.workstation-drawer-body {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.workstation-drawer-body .workstation-chat {
  flex: 1;
  min-height: 0;
}
.workstation-drawer .el-drawer__body {
  padding-top: 0;
}
.workstation-drawer--collapsed .el-drawer__body {
  padding: 0;
  overflow: hidden;
}
.workstation-drawer-strip {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 200px;
}
.workstation-drawer-strip .strip-text {
  writing-mode: vertical-rl;
  letter-spacing: 0.2em;
  font-size: 14px;
  color: var(--el-text-color-regular);
}

/* 新增/编辑页面样式 */
.form-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
  overflow-y: auto;
}

.form-page-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.form-page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.form-page-content {
  flex: 1;
  min-height: 0;
}

.form-page-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

/* 函数加载骨架屏样式 */
.function-loading {
  padding: 24px;
  width: 100%;
}
</style>
