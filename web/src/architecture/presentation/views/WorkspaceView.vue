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
            @pull-from-hub="handlePullFromHub"
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
                    :function-detail="asRenderableFunctionDetail(currentFunctionDetail)"
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

                <el-tab-pane v-if="hasScheduledTasksForCurrentPath" name="scheduledTask" label="定时任务">
                  <div class="tab-content">
                    <ScheduledTaskList
                      ref="scheduledTaskListRef"
                      :resource-path="currentFunction?.full_code_path"
                      :auto-load="functionActiveTab === 'scheduledTask'"
                      @total-change="onScheduledTaskTotalChange"
                    />
                  </div>
                </el-tab-pane>
              </el-tabs>
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
                  :function-detail="asRenderableFunctionDetail(currentFunctionDetail)"
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

    <!-- 创建工作空间对话框 -->
    <el-dialog
      v-model="createAppDialogVisible"
      title="创建新工作空间"
      width="800px"
      :close-on-click-modal="false"
      @close="resetCreateAppForm"
    >
      <el-form :model="createAppForm" label-width="120px">
        <el-form-item label="名称" required>
          <el-input
            v-model="createAppForm.name"
            placeholder="请输入名称（如：清北大学、首都市政府、xxx图书馆、xxx医院、xxx银行、xxx科技公司）"
            maxlength="100"
            show-word-limit
            clearable
          />
        </el-form-item>
        <el-form-item label="英文标识" required>
          <el-tooltip
            content="以小写字母开头，只能包含小写字母、数字和下划线，长度 2-50 个字符"
            placement="top"
          >
            <el-input
              v-model="createAppForm.code"
              placeholder="请输入英文标识（如：tsinghua、pku_gsm）"
              maxlength="50"
              show-word-limit
              clearable
              @input="createAppForm.code = createAppForm.code.toLowerCase()"
            />
          </el-tooltip>
        </el-form-item>
        <el-form-item label="公开">
          <el-tooltip
            content="公开的工作空间可以被其他用户搜索到，关闭则仅自己可见"
            placement="top"
          >
            <el-switch v-model="createAppForm.is_public" />
          </el-tooltip>
        </el-form-item>
        <el-form-item label="仅展示有权限">
          <el-tooltip
            content="开启后，非管理员用户进入该工作空间时，左侧目录只展示其有权限的节点（适合按区/街道/商户划分的 SaaS 场景）"
            placement="top"
          >
            <el-switch v-model="createAppForm.show_only_permitted" />
          </el-tooltip>
        </el-form-item>
        <el-form-item label="管理员">
          <UsersWidget
            :field="createAppAdminsField"
            :value="createAppAdminsFieldValue"
            :field-path="createAppAdminsField.code"
            mode="edit"
            @update:modelValue="handleCreateAppAdminsChange"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            可以设置多个管理员，用逗号分隔。管理员拥有工作空间的管理权限
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createAppDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitCreateApp" :loading="creatingApp">
            创建
          </el-button>
        </span>
      </template>
    </el-dialog>

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

    <!-- 创建服务目录对话框 -->
    <el-dialog
      v-model="createDirectoryDialogVisible"
      :title="currentParentNode ? `在「${currentParentNode.name || currentParentNode.code}」下创建服务目录` : '创建服务目录'"
      width="520px"
      :close-on-click-modal="false"
      @close="handleCloseCreateDirectoryDialog"
    >
      <el-form :model="createDirectoryForm" label-width="120px">
        <el-form-item label="目录名称" required>
          <el-input
            v-model="createDirectoryForm.name"
            placeholder="请输入目录名称（如：用户管理）"
            maxlength="50"
            show-word-limit
            clearable
          />
        </el-form-item>
        <el-form-item label="目录代码" required>
          <el-input
            v-model="createDirectoryForm.code"
            placeholder="请输入目录代码，如：user"
            maxlength="50"
            show-word-limit
            clearable
            @input="createDirectoryForm.code = createDirectoryForm.code.toLowerCase()"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            目录代码只能包含小写字母、数字和下划线
          </div>
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="createDirectoryForm.description"
            type="textarea"
            :rows="3"
            placeholder="请输入目录描述（可选）"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>
        <el-form-item label="标签">
          <el-input
            v-model="createDirectoryForm.tags"
            placeholder="请输入标签，多个标签用逗号分隔（可选）"
            maxlength="100"
            clearable
          />
        </el-form-item>
        <el-form-item label="管理员">
          <UsersWidget
            :field="adminsField"
            :value="adminsFieldValue"
            :field-path="adminsField.code"
            mode="edit"
            @update:modelValue="handleAdminsChange"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            默认当前用户为管理员，可以添加其他用户
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createDirectoryDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmitCreateDirectory" :loading="creatingDirectory">
            创建
          </el-button>
        </span>
      </template>
    </el-dialog>

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
import { computed, defineAsyncComponent, onMounted, onUnmounted, watch, ref, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElNotification, ElDialog, ElForm, ElFormItem, ElInput, ElButton, ElIcon, ElSwitch, ElSkeleton } from 'element-plus'
import { InfoFilled, ArrowLeft, ArrowRight, ChatDotRound, Loading, FolderOpened, Search } from '@element-plus/icons-vue'
import { eventBus, WorkspaceEvent, RouteEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import { RouteManager } from '../../infrastructure/routeManager'
import { useAuthStore } from '@/stores/auth'
import ServiceTreePanel from '@/architecture/presentation/components/ServiceTreePanel.vue'
import WorkspaceHeader from '../components/WorkspaceHeader.vue'
import FunctionBreadcrumb from '../components/FunctionBreadcrumb.vue'
import TableRowDetailDrawer from '../components/TableRowDetailDrawer.vue'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import UsersWidget from '@/shared/components/UsersWidget.vue'
import PermissionRequestList from '@/shared/components/permission/PermissionRequestList.vue'
import PermissionManageList from '@/shared/components/permission/PermissionManageList.vue'
import ScheduledTaskList from '../components/ScheduledTaskList.vue'
import type { ServiceTree, App } from '../../domain/services/WorkspaceDomainService'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import { WidgetType } from '@/core/constants/widget'
import type { FunctionDetail } from '../../domain/interfaces/IFunctionLoader'
import type { FunctionDetail as RenderableFunctionDetail } from '@/core/types/field'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/types'
import type { LocationQueryValue } from 'vue-router'
// 🔥 导入 Composable
import { useWorkspaceRouting } from '../composables/useWorkspaceRouting'
import { RouteSource } from '@/utils/routeSource'
import { useWorkspaceDetail } from '../composables/useWorkspaceDetail'
import { useWorkspaceApp } from '../composables/useWorkspaceApp'
import { useWorkspaceServiceTree } from '../composables/useWorkspaceServiceTree'
import { addFunctionsToDirectory, createDocs, deleteBoard, deleteDocs, deletePackage, deleteServiceTreeFunction } from '@/api/service-tree'
import { findNodeByPath, findNodeById } from '../utils/workspaceUtils'
import { getScopedFieldQueryValue } from '@/utils/queryFieldNamespace'
import { isRootNode as isRootTreeNode } from '@/utils/tree-utils'
import { useAfterCreateNode } from '../composables/useAfterCreateNode'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { resolveWorkspaceUrl, extractWorkspacePath } from '@/utils/route'
import { isLinkNavigation as checkLinkNavigation, LINK_TYPE_QUERY_KEY } from '@/utils/linkNavigation'
import { getWorkspaceSessions, cancelWorkspaceChat, type WorkspaceSessionItem } from '@/api/workspace'
import { listScheduledTasks } from '@/api/scheduledTask'
import { hasPermission, TablePermission, buildPermissionApplyURL } from '@/utils/permission'
import { usePermissionErrorStore } from '@/stores/permissionError'
import { isServiceTreeNodeAdmin } from '@/utils/permissionActors'
import { createStringFieldValue, createWidgetFieldConfig, extractStringFieldRaw } from '@/utils/widgetFieldHelpers'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
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

function normalizeQueryTab(tab: LocationQueryValue | LocationQueryValue[] | undefined): string | null {
  if (Array.isArray(tab)) {
    return tab[0] ?? null
  }

  return typeof tab === 'string' ? tab : null
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

function asRenderableFunctionDetail(detail: FunctionDetail): RenderableFunctionDetail {
  return detail as RenderableFunctionDetail
}

// 🔥 移除缓存后，通过事件获取函数详情
const currentFunctionDetail = ref<FunctionDetail | null>(null)

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


// Fork 函数组相关
// 发布到应用中心对话框
const publishToHubDialogVisible = ref(false)
const publishSelectedNode = ref<ServiceTreeType | null>(null)
const pushToHubDialogVisible = ref(false)
const pushSelectedNode = ref<ServiceTreeType | null>(null)
const importGoFileInputRef = ref<HTMLInputElement | null>(null)
const importGoTargetNode = ref<ServiceTreeType | null>(null)
const importGoLoading = ref(false)
const pullFromHubDialogVisible = ref(false)
const pastedHubLink = ref('')  // 粘贴的 Hub 链接

// 变更记录对话框状态
const updateHistoryDialogVisible = ref(false)
const updateHistoryMode = ref<'app' | 'directory'>('app')
const updateHistoryAppId = ref(0)
const updateHistoryAppVersion = ref('')
const updateHistoryFullCodePath = ref('')

// ServiceTreePanel 引用（用于展开路径）
const serviceTreePanelRef = ref<InstanceType<typeof ServiceTreePanel> | null>(null)
const workspaceHeaderRef = ref<InstanceType<typeof WorkspaceHeader> | null>(null)

// 左侧服务目录树显示状态
const showLeftSidebar = ref(true)

// 右侧会话面板显示状态
const showRightSidebar = ref(true)

// ─── 右侧会话列表 ───
const rightSidebarSessions = ref<WorkspaceSessionItem[]>([])
const rightSidebarSessionsLoading = ref(false)
let rightSidebarPollTimer: ReturnType<typeof setInterval> | null = null

const rightSidebarRunningCount = computed(() =>
  rightSidebarSessions.value.filter((s: WorkspaceSessionItem) => s.status === 'generating').length
)

async function loadRightSidebarSessions() {
  const ctx = workstationContext.value
  if (!ctx) { rightSidebarSessions.value = []; return }
  rightSidebarSessionsLoading.value = true
  try {
    const res = await getWorkspaceSessions({ full_code_path: ctx.fullCodePath })
    rightSidebarSessions.value = res.sessions || []
  } catch {
    rightSidebarSessions.value = []
  } finally {
    rightSidebarSessionsLoading.value = false
  }
}

function startRightSidebarPoll() {
  stopRightSidebarPoll()
  rightSidebarPollTimer = setInterval(() => {
    if (rightSidebarSessions.value.some((s: WorkspaceSessionItem) => s.status === 'generating')) loadRightSidebarSessions()
  }, 5000)
}
function stopRightSidebarPoll() {
  if (rightSidebarPollTimer) { clearInterval(rightSidebarPollTimer); rightSidebarPollTimer = null }
}

function openSessionInMini(session: WorkspaceSessionItem) {
  openNewMiniWs(session.session_id, session.full_code_path)
}

function formatRelativeTime(timeStr: string): string {
  const time = new Date(timeStr)
  const now = new Date()
  const diff = now.getTime() - time.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`
  return time.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

// ─── 右侧面板 tab（仅筛选当前节点会话） ───
const rightTab = ref<'all' | 'running' | 'finished'>('all')
const rightSessionSearchKeyword = ref('')
const cancellingTaskId = ref<string | null>(null)

const filteredRightSessions = computed(() => {
  let list = rightSidebarSessions.value
  if (rightTab.value === 'running') list = list.filter((s: WorkspaceSessionItem) => s.status === 'generating')
  else if (rightTab.value === 'finished') list = list.filter((s: WorkspaceSessionItem) => s.status === 'done' || s.status === 'cancelled')
  const k = rightSessionSearchKeyword.value.trim().toLowerCase()
  if (!k) return list
  return list.filter((s: WorkspaceSessionItem) => {
    const title = (s.title || '').toLowerCase()
    const user = (s.user || '').toLowerCase()
    return title.includes(k) || user.includes(k)
  })
})

async function handleCancelTask(task: WorkspaceSessionItem) {
  cancellingTaskId.value = task.session_id
  try {
    await cancelWorkspaceChat(task.session_id)
    ElMessage.success('已停止该任务')
    loadRightSidebarSessions()
  } catch (e: any) {
    ElMessage.error(e?.message || '停止失败')
  } finally {
    cancellingTaskId.value = null
  }
}

// 函数详情 tab 相关
const functionActiveTab = ref('content')
const functionPermissionTab = ref('request')
const functionFormViewRef = ref<{
  applyOperateLog: (payload: {
    requestBody?: Record<string, any> | null
    responseBody?: Record<string, any> | null
    responseMetadata?: Record<string, any> | null
  }) => Promise<void>
} | null>(null)
const functionPermissionRequestListRef = ref<InstanceType<typeof PermissionRequestList> | null>(null)
const functionPermissionManageListRef = ref<InstanceType<typeof PermissionManageList> | null>(null)
const scheduledTaskListRef = ref<InstanceType<typeof ScheduledTaskList> | null>(null)
const formOperateLogSectionRef = ref<{ loadLogs: (options?: { page?: number }) => void } | null>(null)
/** 当前函数是否有定时任务（无则不显示「定时任务」tab） */
const hasScheduledTasksForCurrentPath = ref(false)

// ⭐ 判断是否显示函数权限申请 tab
// 条件：1. 节点类型是 function  2. 用户是管理员
const showFunctionPermissionRequestTab = computed(() => {
  if (!currentFunction.value) {
    return false
  }
  
  // 必须是 function 类型
  if (currentFunction.value.type !== 'function') {
    return false
  }
  
  return isServiceTreeNodeAdmin(currentFunction.value, authStore.user?.username)
})

const showFormOperateLogTab = computed(() => {
  return currentFunction.value?.type === 'function' && currentFunctionDetail.value?.template_type === TEMPLATE_TYPE.FORM
})

const showFunctionTabsWrapper = computed(() => {
  return showFunctionPermissionRequestTab.value || showFormOperateLogTab.value
})

const loadCurrentFunctionPermissionTab = () => {
  if (functionPermissionTab.value === 'manage') {
    nextTick(() => {
      functionPermissionManageListRef.value?.loadPermissions()
    })
    return
  }

  nextTick(() => {
    functionPermissionRequestListRef.value?.loadRequests()
  })
}

// 处理函数 tab 切换
const handleFunctionTabChange = (tabName: string) => {
  functionActiveTab.value = tabName
  if (tabName === 'permission') {
    loadCurrentFunctionPermissionTab()
  } else if (tabName === 'operateLog' && formOperateLogSectionRef.value) {
    nextTick(() => {
      formOperateLogSectionRef.value?.loadLogs({ page: 1 })
    })
  }
}

const handleFunctionPermissionTabChange = (tabName: string) => {
  functionPermissionTab.value = tabName === 'manage' ? 'manage' : 'request'
  if (functionActiveTab.value === 'permission') {
    loadCurrentFunctionPermissionTab()
  }
}

const handleApplyFormOperateLog = async (payload: {
  requestBody?: Record<string, any> | null
  responseBody?: Record<string, any> | null
  responseMetadata?: Record<string, any> | null
}) => {
  functionActiveTab.value = 'content'
  await nextTick()

  if (!functionFormViewRef.value) {
    ElMessage.warning('当前表单尚未加载完成，请稍后重试')
    return
  }

  try {
    await functionFormViewRef.value.applyOperateLog(payload)
    ElMessage.success('已将执行记录回填到表单')
  } catch (error: any) {
    ElMessage.error(error?.message || '回填执行记录失败')
  }
}

/** 定时任务列表 total 变化：无任务时隐藏 tab 并切回内容 */
function onScheduledTaskTotalChange(total: number) {
  hasScheduledTasksForCurrentPath.value = total > 0
  if (total === 0 && functionActiveTab.value === 'scheduledTask') {
    functionActiveTab.value = 'content'
  }
}

/** 根据当前函数路径拉取定时任务数量，决定是否显示「定时任务」tab；若无任务且当前在定时任务 tab 则切回内容避免空白 */
async function refreshScheduledTasksCountForCurrentPath() {
  const path = currentFunction.value?.full_code_path
  if (!path || !showFunctionPermissionRequestTab.value) {
    hasScheduledTasksForCurrentPath.value = false
    if (functionActiveTab.value === 'scheduledTask') {
      functionActiveTab.value = 'content'
    }
    return
  }
  try {
    const res = await listScheduledTasks({ full_code_path: path, page: 1, page_size: 1 })
    const hasAny = (res.total ?? 0) > 0
    hasScheduledTasksForCurrentPath.value = hasAny
    if (!hasAny && functionActiveTab.value === 'scheduledTask') {
      functionActiveTab.value = 'content'
    }
  } catch {
    hasScheduledTasksForCurrentPath.value = false
    if (functionActiveTab.value === 'scheduledTask') {
      functionActiveTab.value = 'content'
    }
  }
}

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

/** 多 Mini 工作台实例 */
interface MiniWsInstance {
  id: string
  fullCodePath: string
  dirName: string
  initialSessionId: string
  visible: boolean
  offset: number
  initialPosition?: 'center'
  initialMaximized?: boolean
}
const miniWsList = ref<MiniWsInstance[]>([])
let miniIdCounter = 0

/** 工作台上下文：点击什么节点就用什么节点的 full_code_path */
const workstationContext = computed(() => {
  const node = currentFunction.value
  if (!node?.full_code_path) return null
  const path = (node.full_code_path || '').replace(/\/+$/g, '')
  if (!path) return null
  const name = node.name || path.split('/').pop() || '工作台'
  return { fullCodePath: path, dirName: name }
})

// 当目录变化或侧边栏打开时加载右侧会话列表
watch(
  [() => workstationContext.value?.fullCodePath, showRightSidebar],
  ([path, visible]) => {
    stopRightSidebarPoll()
    if (path && visible) {
      loadRightSidebarSessions()
      startRightSidebarPoll()
    }
  },
  { immediate: true }
)

function openNewMiniWs(initialSessionId?: string, overridePath?: string, overrideName?: string, initialMaximized = false) {
  const ctx = workstationContext.value
  const fcp = overridePath || ctx?.fullCodePath
  if (!fcp) return
  const dirName = overrideName || ctx?.dirName || fcp.split('/').filter(Boolean).pop() || '工作台'
  const existing = miniWsList.value.find(
    (mini: MiniWsInstance) => mini.fullCodePath === fcp && mini.initialSessionId === (initialSessionId || '')
  )
  if (existing) {
    existing.visible = true
    return
  }
  const offset = miniWsList.value.filter((m: MiniWsInstance) => m.visible).length * 40
  miniWsList.value.push({
    id: String(++miniIdCounter),
    fullCodePath: fcp,
    dirName,
    initialSessionId: initialSessionId || '',
    visible: true,
    offset: initialMaximized ? 0 : offset,
    initialPosition: initialMaximized ? undefined : 'center',
    initialMaximized,
  })
}

function handleMiniMinimize(id: string) {
  const mini = miniWsList.value.find((m: MiniWsInstance) => m.id === id)
  if (mini) mini.visible = false
  syncMiniWsQueryParam(false)
}

function handleMiniRemove(id: string) {
  miniWsList.value = miniWsList.value.filter((m: MiniWsInstance) => m.id !== id)
  syncMiniWsQueryParam(false)
}

function handleMiniMaximizeChange(payload: { maximized: boolean; sessionId?: string }) {
  if (payload.maximized) {
    syncMiniWsQueryParam(true, payload.sessionId)
  } else {
    syncMiniWsQueryParam(false)
  }
}

function syncMiniWsQueryParam(open: boolean, sid?: string) {
  const query = { ...route.query }
  if (open) {
    query.mws = 'open'
    if (sid) { query.mws_sid = sid } else { delete query.mws_sid }
    const ctx = workstationContext.value
    if (ctx) {
      query.mws_path = ctx.fullCodePath
      query.mws_name = ctx.dirName
    }
  } else {
    delete query.mws
    delete query.mws_sid
    delete query.mws_path
    delete query.mws_name
  }
  router.replace({ path: route.path, query })
}

function restoreMiniWorkstation(options?: {
  fullCodePath?: string
  dirName?: string
  sessionId?: string
  initialMaximized?: boolean
}) {
  const restore = (fullCodePath: string, dirName: string) => {
    openNewMiniWs(
      options?.sessionId || undefined,
      fullCodePath,
      dirName,
      !!options?.initialMaximized
    )
  }

  if (options?.fullCodePath) {
    const dirName = options.dirName || options.fullCodePath.split('/').filter(Boolean).pop() || '工作台'
    nextTick(() => restore(options.fullCodePath!, dirName))
    return
  }

  const stopRestore = watch(workstationContext, (ctx: { fullCodePath: string; dirName: string } | null) => {
    if (ctx?.fullCodePath) {
      restore(ctx.fullCodePath, ctx.dirName)
      stopRestore()
    }
  }, { immediate: true })

  setTimeout(() => stopRestore(), 10000)
}

/** 服务树「打开工作台」事件：统一打开 Mini；任务面板「查看」仍可指定最大化 Mini。 */
function handleWorkspaceOpenWorkstation(payload: { full_code_path?: string; session_id?: string; open_as_mini?: boolean }) {
  const fullCodePath = (payload?.full_code_path || '').trim()
  if (!fullCodePath) return
  const targetPath = buildWorkspacePath(fullCodePath)
  const dirName = fullCodePath.split('/').filter(Boolean).pop() || '工作台'
  const openMini = () => {
    if (payload.open_as_mini) {
      openMiniWsForTask(fullCodePath, payload.session_id || '')
      return
    }
    openNewMiniWs(payload.session_id || undefined, fullCodePath, dirName)
  }

  if (route.path !== targetPath) {
    const query = { ...route.query }
    delete query.ws
    delete query.ws_sid
    router.push({ path: targetPath, query }).then(() => {
      nextTick(() => openMini())
    })
  } else {
    nextTick(() => openMini())
  }
}

/** 任务面板「查看」：打开 Mini 并定位到该会话 */
function openMiniWsForTask(fullCodePath: string, sessionId: string) {
  const dirName = fullCodePath.split('/').filter(Boolean).pop() || '工作台'
  const existing = miniWsList.value.find(
    (m: MiniWsInstance) => m.fullCodePath === fullCodePath && m.initialSessionId === sessionId
  )
  if (existing) {
    existing.visible = true
    return
  }
  miniWsList.value.push({
    id: String(++miniIdCounter),
    fullCodePath,
    dirName,
    initialSessionId: sessionId,
    visible: true,
    offset: 0,
    initialPosition: 'center',
    initialMaximized: true,
  })
}

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

// 🔥 全局粘贴监听：检测 Hub 链接并自动打开安装对话框
const handleGlobalPaste = async (event: ClipboardEvent) => {
  // 如果当前焦点在输入框、文本域等可编辑元素上，不处理（让默认行为生效）
  const target = event.target as HTMLElement
  if (target && (
    target.tagName === 'INPUT' ||
    target.tagName === 'TEXTAREA' ||
    target.isContentEditable ||
    // 检查是否是 TipTap 编辑器（TipTap 编辑器有特定的类名）
    target.closest('.ProseMirror') ||
    target.closest('.rich-text-widget') ||
    target.closest('.editor-container')
  )) {
    return
  }

  const pastedText = event.clipboardData?.getData('text')
  if (pastedText && pastedText.trim().startsWith('hub://')) {
    // 阻止默认粘贴行为
    event.preventDefault()
    
    // 检查是否有当前应用
    if (!currentApp.value) {
      ElMessage.warning('请先选择应用')
      return
    }

    // 设置粘贴的 Hub 链接
    pastedHubLink.value = pastedText.trim()
    
    // 打开安装对话框
    pullFromHubDialogVisible.value = true
    
    ElMessage.info('检测到 Hub 链接，已打开安装对话框')
  }
}

onMounted(() => {
  // 从 localStorage 恢复左侧边栏状态
  const savedLeft = localStorage.getItem('workspace-left-sidebar')
  if (savedLeft !== null) {
    showLeftSidebar.value = savedLeft === 'true'
  }
  
  // 从 localStorage 恢复右侧边栏状态
  const savedRight = localStorage.getItem('workspace-right-sidebar')
  if (savedRight !== null) {
    showRightSidebar.value = savedRight === 'true'
  }

  
  // 🔥 监听表格详情事件（使用 Composable）
  unsubscribeTableDetailRow = eventBus.on('table:detail-row', async (payload: { row: Record<string, any>, index?: number, tableData?: any[], initialMode?: 'read' | 'edit' }) => {
    const { row, index, tableData, initialMode = 'read' } = payload
    await openDetailDrawer(row, index, tableData, initialMode)
  })
  
  // 🔥 Tab 功能已删除，相关事件监听已移除
  
  // 🔥 设置 URL 监听（使用 Composable）
  unsubscribeDetailUrlWatch = setupUrlWatch()
  
  // 🔥 添加全局粘贴监听
  document.addEventListener('paste', handleGlobalPaste)

  // 🔥 服务目录树「打开工作台」：统一打开 Mini 并定位到对应目录
  unsubscribeWorkspaceOpenWorkstation = eventBus.on('workspace:open-workstation', handleWorkspaceOpenWorkstation)

  // 🔥 兼容旧链接：?ws=open 时改为恢复 Mini 工作台
  if (route.query.ws === 'open') {
    const sid = typeof route.query.ws_sid === 'string' ? route.query.ws_sid : ''
    restoreMiniWorkstation({
      sessionId: sid || undefined,
      initialMaximized: false,
    })
  }

  // 🔥 URL 参数恢复：?mws=open 时恢复 Mini 工作台
  if (route.query.mws === 'open') {
    const mwsSid = typeof route.query.mws_sid === 'string' ? route.query.mws_sid : ''
    const mwsPath = typeof route.query.mws_path === 'string' ? route.query.mws_path : ''
    const mwsName = typeof route.query.mws_name === 'string' ? route.query.mws_name : ''
    restoreMiniWorkstation({
      fullCodePath: mwsPath || undefined,
      dirName: mwsName || undefined,
      sessionId: mwsSid || undefined,
      initialMaximized: true,
    })
  }
})

onUnmounted(() => {
  // 🔥 移除全局粘贴监听
  document.removeEventListener('paste', handleGlobalPaste)
  if (unsubscribeTableDetailRow) {
    unsubscribeTableDetailRow()
  }
  if (unsubscribeDetailUrlWatch) {
    unsubscribeDetailUrlWatch()
  }
  if (unsubscribeWorkspaceOpenWorkstation) {
    unsubscribeWorkspaceOpenWorkstation()
  }
  stopRightSidebarPoll()
})



// 转换 loadingTree 为 boolean (避免 computed 类型问题)
const loading = computed(() => domainService.isLoading())

/**
 * 构建工作空间路径
 */
const buildWorkspacePath = (fullCodePath: string): string => {
  return resolveWorkspaceUrl(fullCodePath.startsWith('/') ? fullCodePath : `/${fullCodePath}`)
}

/**
 * 判断是否是 table 函数
 */
const isTableFunction = (node: ServiceTree): boolean => {
  return node.template_type === TEMPLATE_TYPE.TABLE
}

/**
 * 判断是否是 link 跳转
 */
const isLinkNavigation = (): boolean => {
  return checkLinkNavigation(route.query as Record<string, any>)
}

/**
 * 构建 link 跳转的查询参数（保留所有参数，除了 _link_type）
 */
const buildLinkNavigationQuery = (): Record<string, string | string[]> => {
  const preservedQuery: Record<string, string | string[]> = {}
  Object.keys(route.query).forEach(key => {
    if (key !== LINK_TYPE_QUERY_KEY) {
      const value = route.query[key]
      if (value !== null && value !== undefined) {
        preservedQuery[key] = Array.isArray(value) 
          ? value.filter(v => v !== null).map(v => String(v))
          : String(value)
      }
    }
  })
  return preservedQuery
}

/**
 * 处理函数节点的路由更新
 * 🔥 切换函数时清空所有查询参数，避免参数污染
 */
const handleFunctionNodeRoute = (node: ServiceTree, source: string): void => {
  if (!node.full_code_path) {
    return
  }
  
  const targetPath = buildWorkspacePath(node.full_code_path)
  
  if (route.path === targetPath) {
    // 路由已匹配，直接触发节点点击加载详情（避免路由更新循环）
    applicationService.triggerNodeClick(node)
    return
  }
  
  const isLink = isLinkNavigation()
  
  // 🔥 构建查询参数
  // 只有 link 跳转时才保留参数，普通切换函数时清空所有参数
  const preservedQuery: Record<string, string | string[]> = isLink
    ? buildLinkNavigationQuery()  // link 跳转：保留所有参数（除了 _link_type）
    : {}                           // 普通切换函数：清空所有查询参数，避免参数污染
  
  const preserveParams = {
    table: false,      // 🔥 不再保留 table 参数
    search: false,     // 🔥 不再保留搜索参数
    state: false,      // 🔥 不再保留状态参数
    linkNavigation: isLink  // 只有 link 跳转时才保留参数
  }
  
  // 发出路由更新请求事件
  eventBus.emit(RouteEvent.updateRequested, {
    path: targetPath,
    query: preservedQuery,
    replace: true,
    preserveParams,
    source: source as any
  })
}

/**
 * 处理目录节点的路由更新
 * ⭐ 优化：不再在这里调用 triggerNodeClick，因为已经在 handleNodeClick 中调用过了
 */
const handlePackageNodeRoute = (node: ServiceTree, source: string, customQuery?: Record<string, any>): void => {
  if (!node.full_code_path) return
  
  const targetPath = buildWorkspacePath(node.full_code_path)
  // ⭐ 如果路由已匹配，不需要更新路由（节点点击已经在 handleNodeClick 中处理了）
  // 但如果 source 是 approve-permission-click，需要更新 query 参数
  if (route.path === targetPath && !customQuery) {
    return
  }
  
  eventBus.emit(RouteEvent.updateRequested, {
    path: targetPath,
    query: customQuery || {},
    replace: true,
    preserveParams: {
      table: false,
      search: false,
      state: false,
      linkNavigation: false
    },
    source: source as any
  })
}

// 事件处理
const handleNodeClick = (node: ServiceTreeType) => {
  // 转换为新架构的 ServiceTree 类型
  const serviceTree: ServiceTree = node as any
  
  if (serviceTree.type === 'function') {
    handleFunctionNodeRoute(serviceTree, RouteSource.WORKSPACE_NODE_CLICK)
  } else if (serviceTree.type === 'package') {
    // ⭐ 优化：先检查路由是否已匹配，避免重复调用
    const targetPath = buildWorkspacePath(serviceTree.full_code_path || '')
    if (route.path === targetPath) {
      // 路由已匹配，直接触发节点点击（只调用一次）
      applicationService.triggerNodeClick(serviceTree)
    } else {
      // 路由未匹配，先触发节点点击，然后更新路由
      applicationService.triggerNodeClick(serviceTree)
      handlePackageNodeRoute(serviceTree, RouteSource.WORKSPACE_NODE_CLICK_PACKAGE)
    }
  } else if (serviceTree.type === 'docs') {
    // ⭐ docs 类型节点，也需要更新路由
    const targetPath = buildWorkspacePath(serviceTree.full_code_path || '')
    if (route.path === targetPath) {
      applicationService.triggerNodeClick(serviceTree)
    } else {
      applicationService.triggerNodeClick(serviceTree)
      handlePackageNodeRoute(serviceTree, 'workspace-node-click-docs')
    }
  } else if (serviceTree.type === 'board') {
    // ⭐ board 类型节点，更新路由
    const targetPath = buildWorkspacePath(serviceTree.full_code_path || '')
    if (route.path === targetPath) {
      applicationService.triggerNodeClick(serviceTree)
    } else {
      applicationService.triggerNodeClick(serviceTree)
      handlePackageNodeRoute(serviceTree, 'workspace-node-click-board')
    }
  } else {
    // 其他类型节点，只设置当前函数
    applicationService.triggerNodeClick(serviceTree)
  }
}

// ⭐ 处理审批权限申请（从 ServiceTreePanel 调用）
const handleApprovePermission = (node: ServiceTreeType) => {
  const serviceTree: ServiceTree = node as any
  if (!serviceTree.full_code_path) return
  
  // 先触发节点点击，确保节点详情已加载
  applicationService.triggerNodeClick(serviceTree)
  
  // 然后更新路由，添加 tab 参数
  handlePackageNodeRoute(serviceTree, 'approve-permission-click', {
    tab: 'permissionRequest'
  })
}

/**
 * 处理面包屑节点点击
 */
const handleBreadcrumbNodeClick = (node: ServiceTree) => {
  if (node.type === 'function') {
    handleFunctionNodeRoute(node, RouteSource.WORKSPACE_NODE_CLICK)
  } else if (node.type === 'package') {
    handlePackageNodeRoute(node, RouteSource.WORKSPACE_NODE_CLICK_PACKAGE)
  } else if (node.type === 'docs') {
    handlePackageNodeRoute(node, 'breadcrumb-node-click-docs')
  } else if (node.type === 'board') {
    handlePackageNodeRoute(node, 'breadcrumb-node-click-board')
  } else {
    applicationService.triggerNodeClick(node)
  }
}


// 🔥 处理创建目录（使用 Composable）
const handleCreateDirectory = (parentNode?: ServiceTreeType) => {
  serviceTreeHandleCreateDirectory(parentNode || null, () => currentApp.value)
}

const handleSubmitCreateDirectory = async () => {
  await serviceTreeHandleSubmitCreateDirectory(() => currentApp.value)
}

// 创建文档对话框相关状态
const createDocsDialogVisible = ref(false)
const creatingDocs = ref(false)
const currentDocsParentNode = ref<ServiceTreeType | null>(null)
const createDocsForm = ref({
  name: '',
  code: '',
  description: '',
  tags: '',
  content: '',    // ⭐ 文档内容
  summary: ''     // ⭐ 文档摘要
})

// 创建讨论区（版块）对话框 - 仅保留可见性与父节点，逻辑在 CreateBoardDialog 内
const createBoardDialogVisible = ref(false)
const currentBoardParentNode = ref<ServiceTreeType | null>(null)

// 处理创建文档节点（打开对话框）
const handleCreateDocs = (parentNode?: ServiceTreeType) => {
  if (!currentApp.value) {
    ElMessage.warning('请先选择应用')
    return
  }
  currentDocsParentNode.value = parentNode || null
  createDocsForm.value = {
    name: '',
    code: '',
    description: '',
    tags: '',
    content: '',
    summary: ''
  }
  createDocsDialogVisible.value = true
}

// 提交创建文档
const handleSubmitCreateDocs = async () => {
  if (!currentApp.value) {
    ElMessage.warning('请先选择应用')
    return
  }

  if (!createDocsForm.value.name.trim()) {
    ElMessage.warning('请输入文档名称')
    return
  }

  if (!createDocsForm.value.code.trim()) {
    ElMessage.warning('请输入文档代码')
    return
  }

  // 验证代码格式（只能包含小写字母、数字和下划线）
  const codePattern = /^[a-z0-9_]+$/
  if (!codePattern.test(createDocsForm.value.code)) {
    ElMessage.warning('文档代码只能包含小写字母、数字和下划线')
    return
  }

  // 自动补全 type 后缀（与 form/table/chart 一致）
  let code = createDocsForm.value.code.trim()
  if (!code.endsWith('.docs')) code = code + '.docs'

  // ⭐ 验证文档内容
  if (!createDocsForm.value.content.trim()) {
    ElMessage.warning('请输入文档内容')
    return
  }

  creatingDocs.value = true
  try {
    const parentFullCodePath = currentDocsParentNode.value?.full_code_path || ''
    // ⭐ 使用新的分离接口
    const response = await createDocs({
      user: currentApp.value.user,
      app: currentApp.value.code,
      name: createDocsForm.value.name.trim(),
      code,
      parent_full_code_path: parentFullCodePath,
      description: createDocsForm.value.description.trim() || '',
      tags: createDocsForm.value.tags.trim() || '',
      content: createDocsForm.value.content.trim(),  // ⭐ 文档内容
      format: 'markdown',  // ⭐ 文档格式
      summary: createDocsForm.value.summary.trim() || ''  // ⭐ 文档摘要
    })

    if (response && response.id) {
      ElMessage.success('文档节点创建成功')
      createDocsDialogVisible.value = false
      await afterCreateNode(response)
    } else {
      ElMessage.warning('创建文档节点成功，但未返回节点信息')
      createDocsDialogVisible.value = false
    }
  } catch (error: any) {
    ElMessage.error('创建文档节点失败: ' + (error.message || '未知错误'))
  } finally {
    creatingDocs.value = false
  }
}

// 处理关闭创建文档对话框
const handleCloseCreateDocsDialog = () => {
  createDocsForm.value = {
    name: '',
    code: '',
    description: '',
    tags: '',
    content: '',
    summary: ''
  }
  currentDocsParentNode.value = null
}

// 处理创建讨论区（仅打开对话框）
const handleCreateBoard = (parentNode?: ServiceTreeType) => {
  if (!currentApp.value) {
    ElMessage.warning('请先选择应用')
    return
  }
  currentBoardParentNode.value = parentNode ?? null
  createBoardDialogVisible.value = true
}

// 处理删除讨论区
const handleDeleteBoard = async (node: ServiceTreeType) => {
  if (node.type !== 'board') {
    ElMessage.warning('只能删除讨论区节点')
    return
  }
  try {
    await ElMessageBox.confirm(
      `确定要删除讨论区 "${node.name}" 吗？将同时删除该版块下全部帖子，且无法恢复。`,
      '确认删除',
      { confirmButtonText: '确定', cancelButtonText: '取消', type: 'warning' }
    )
    await deleteBoard(node.id)
    ElMessage.success('讨论区已删除')
    await handleRefreshTree()
    if (currentFunction.value && currentFunction.value.id === node.id) {
      const parentPath = node.full_code_path?.split('/').slice(0, -1).join('/') || ''
      if (parentPath) router.push(`/workspace${parentPath}`)
    }
  } catch (error: any) {
    if (error !== 'cancel') ElMessage.error('删除讨论区失败: ' + (error.message || '未知错误'))
  }
}

// 处理关闭创建目录对话框
const handleCloseCreateDirectoryDialog = () => {
  resetCreateDirectoryForm(() => currentApp.value)
}

// 处理删除文档
const handleDeleteDoc = async (node: ServiceTreeType) => {
  if (node.type !== 'docs') {
    ElMessage.warning('只能删除文档节点')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除文档 "${node.name}" 吗？此操作将删除文档内容和文档节点，且无法恢复。`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await deleteDocs(node.id)
    
    ElMessage.success('文档删除成功')
    
    // 刷新服务树
    await handleRefreshTree()
    
    // 如果当前选中的是已删除的文档，清空选中状态
    if (currentFunction.value && currentFunction.value.id === node.id) {
      // 可以跳转到父节点或清空选中
      const parentPath = node.full_code_path?.split('/').slice(0, -1).join('/') || ''
      if (parentPath) {
        const targetPath = `/workspace${parentPath}`
        router.push(targetPath)
      }
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error('删除文档失败: ' + (error.message || '未知错误'))
    }
  }
}

// 处理文档删除后（从 DocView 组件触发）
const handleDocDeleted = async () => {
  // 刷新服务树
  await handleRefreshTree()
  
  // 如果当前选中的是已删除的文档，清空选中状态或跳转到父节点
  if (currentFunction.value && currentFunction.value.type === 'docs') {
    const parentPath = currentFunction.value.full_code_path?.split('/').slice(0, -1).join('/') || ''
    if (parentPath) {
      const targetPath = `/workspace${parentPath}`
      router.push(targetPath)
    }
  }
}

// 处理 Fork 函数组
// 导入 Go 文件：打开文件选择，选中后按 add_functions 写入目录
const handleImportGoFiles = (node: ServiceTreeType) => {
  const fullCodePath = node.full_code_path
  if (!fullCodePath) {
    ElMessage.warning('该目录无完整路径')
    return
  }
  importGoTargetNode.value = node
  importGoFileInputRef.value?.click()
}

async function doImportGoFiles(files: FileList | File[], fullCodePath: string) {
  importGoLoading.value = true
  let ok = 0
  let fail = 0
  try {
    const fileArray = Array.from(files)
    for (let i = 0; i < fileArray.length; i++) {
      const file = fileArray[i]
      if (!file || !file.name.toLowerCase().endsWith('.go')) continue
      const content = await readFileAsText(file)
      const fileName = file.name.endsWith('.go') ? file.name : file.name + '.go'
      try {
        const res = await addFunctionsToDirectory({
          full_code_path: fullCodePath,
          file_name: fileName,
          source_code: content,
          skip_build: true
        })
        if (res?.success !== false) ok++
        else { fail++; console.warn('add_functions failed:', res?.error) }
      } catch (err: any) {
        fail++
        console.warn('add_functions error:', err)
        ElMessage.warning(`${file.name}: ${err?.message || err?.response?.data?.msg || '写入失败'}`)
      }
    }
    if (ok > 0) {
      ElMessage.success(`已导入 ${ok} 个 Go 文件到目录，可在工作台执行编译以生效。`)
      await handleRefreshTree()
    }
    if (fail > 0 && ok === 0) ElMessage.error('导入失败')
  } finally {
    importGoLoading.value = false
  }
}

const onImportGoFilesSelected = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const files = input.files
  if (!files?.length || !importGoTargetNode.value) {
    input.value = ''
    return
  }
  const fullCodePath = importGoTargetNode.value.full_code_path!
  importGoTargetNode.value = null
  input.value = ''
  await doImportGoFiles(files, fullCodePath)
}

function readFileAsText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error)
    reader.readAsText(file, 'utf-8')
  })
}

// 处理发布到应用中心
const handlePublishToHub = (node: ServiceTreeType) => {
  publishSelectedNode.value = node
  publishToHubDialogVisible.value = true
}

// 处理推送到应用中心
const handlePushToHub = (node: ServiceTreeType) => {
  pushSelectedNode.value = node
  pushToHubDialogVisible.value = true
}

// 从应用中心安装：预填链接与目标目录（目标目录默认当前选中目录，显示用 name）
const pullFromHubTargetPath = ref('')
const pullFromHubTargetName = ref('')
const handlePullFromHub = (initialLink?: string, targetFullCodePath?: string, targetName?: string) => {
  pastedHubLink.value = initialLink ?? ''
  pullFromHubTargetPath.value = targetFullCodePath ?? ''
  pullFromHubTargetName.value = targetName ?? ''
  pullFromHubDialogVisible.value = true
}

// 处理删除目录（非根 package）
const handleDeleteDirectory = async (node: ServiceTreeType) => {
  if (node.type !== 'package') {
    ElMessage.warning('只能删除目录节点')
    return
  }
  if (isRootTreeNode(node as ServiceTreeType)) {
    ElMessage.warning('不能删除工作空间根目录')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除目录 "${node.name}" 吗？此操作将删除该目录及其下所有子目录、函数和文档，且无法恢复。`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await deletePackage(node.id)

    ElMessage.success('目录删除成功')

    // 如果当前选中的是该目录或其子节点，清空或跳转到父级
        const deletedPath = node.full_code_path || ''
    if (currentFunction.value) {
      const currentPath = currentFunction.value.full_code_path || ''
      if (currentPath === deletedPath || currentPath.startsWith(deletedPath + '/')) {
        domainService.setCurrentFunction(null)
        const parentPath = deletedPath.split('/').slice(0, -1).join('/') || ''
        if (parentPath) {
          router.replace({ path: `/workspace${parentPath}`, query: { ...route.query } })
        } else {
          router.replace({ path: route.path, query: { ...route.query, _id: undefined, _tab: undefined } })
        }
      }
    }

    await handleRefreshTree()
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') {
      const errorMessage = error?.response?.data?.msg || error?.message || '删除目录失败'
      ElMessage.error(errorMessage)
    }
  }
}

// 处理删除函数
const handleDeleteFunction = async (node: ServiceTreeType) => {
  if (node.type !== 'function') {
    ElMessage.warning('只能删除函数节点')
    return
  }

  try {
    await ElMessageBox.confirm(
      `确定要删除函数 "${node.name}" 吗？此操作不可恢复。`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    await deleteServiceTreeFunction(node.id)

    ElMessage.success('删除成功')

    // 如果删除的是当前选中的函数，清空选中状态
    if (currentFunction.value && currentFunction.value.id === node.id) {
      domainService.setCurrentFunction(null)
      // 清空 URL 参数
      router.replace({
        path: route.path,
        query: { ...route.query, _id: undefined, _tab: undefined }
      })
    }

    // 刷新服务树
    await handleRefreshTree()
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') {
      const errorMessage = error?.response?.data?.msg || error?.message || '删除失败'
      ElMessage.error(errorMessage)
    }
  }
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

// 会改变服务目录结构的工具名（创建目录、写文档、写代码、编译工作空间）
const TREE_AFFECTING_TOOLS = ['create_directory', 'write_doc', 'write_go_file', 'build_workspace']

// 工作台工具调用成功时：若为改树工具则刷新服务树
const handleWorkstationToolCallOk = (payload: { name: string }) => {
  if (payload?.name && TREE_AFFECTING_TOOLS.includes(payload.name)) {
    handleRefreshTree()
  }
}

// 处理变更记录
const handleUpdateHistory = (node?: ServiceTreeType) => {
  if (!currentApp.value) {
    ElMessage.warning('请先选择应用')
    return
  }
  
  // 🔥 修复：检查 appId 是否有效
  const appId = currentApp.value.id
  if (!appId || appId === 0) {
    console.error('[WorkspaceView] handleUpdateHistory: appId 无效', {
      currentApp: currentApp.value,
      appId
    })
    ElMessage.error('应用ID无效，无法加载变更记录。请刷新页面后重试。')
    return
  }
  
  if (node) {
    // 目录视角：显示指定目录的变更记录
    updateHistoryMode.value = 'directory'
    updateHistoryAppId.value = appId
    updateHistoryFullCodePath.value = node.full_code_path || ''
    updateHistoryAppVersion.value = ''
  } else {
    // App视角：显示工作空间的变更记录
    updateHistoryMode.value = 'app'
    updateHistoryAppId.value = appId
    updateHistoryAppVersion.value = '' // 空表示返回所有版本
    updateHistoryFullCodePath.value = ''
  }
  
  updateHistoryDialogVisible.value = true
}

// 发布成功后的回调
const handlePublishSuccess = async () => {
  // 刷新服务目录树
  const app = getCurrentAppForTreeLoad()
  if (app) {
    await domainService.loadServiceTree(app)
  }
}

// 推送成功后的回调
const handlePushSuccess = async () => {
  // 刷新服务目录树
  const app = getCurrentAppForTreeLoad()
  if (app) {
    await domainService.loadServiceTree(app)
  }
}

// 拉取成功后的回调
const handlePullSuccess = async () => {
  pastedHubLink.value = ''
  pullFromHubTargetPath.value = ''
  pullFromHubTargetName.value = ''
  // 刷新服务目录树
  const app = getCurrentAppForTreeLoad()
  if (app) {
    await domainService.loadServiceTree(app)
  }
}


// 🔥 展开当前路由对应的路径（使用 Composable）
const expandCurrentRoutePath = () => {
  serviceTreeExpandCurrentRoutePath(
    () => serviceTree.value,
    () => serviceTreePanelRef.value,
    () => currentApp.value
  )
}

// 🔥 返回列表（从 create/edit 模式返回）
// 🔥 阶段4：改为事件驱动，通过 RouteManager 统一处理路由更新
const backToList = () => {
  if (!currentFunction.value) return
  
  // 移除系统参数，保留其他参数
  const query: Record<string, string | string[]> = {}
  Object.keys(route.query).forEach(key => {
    if (key !== '_tab' && key !== '_id') {
      const value = route.query[key]
      if (value !== null && value !== undefined) {
        query[key] = Array.isArray(value) 
          ? value.filter(v => v !== null).map(v => String(v))
          : String(value)
      }
    }
  })
  
  const path = currentFunction.value.full_code_path 
    ? buildWorkspacePath(currentFunction.value.full_code_path)
    : ''
  
  // 🔥 发出路由更新请求事件
  eventBus.emit(RouteEvent.updateRequested, {
    path,
    query,
    replace: false,  // 返回列表使用 push，保留历史记录
    preserveParams: {
      state: true  // 保留状态参数
    },
    source: RouteSource.BACK_TO_LIST
  })
}


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


// 生命周期
let unsubscribeFunctionLoaded: (() => void) | null = null
let unsubscribeServiceTreeLoaded: (() => void) | null = null
let unsubscribeAppSwitched: (() => void) | null = null
let unsubscribeScheduledTaskCreated: (() => void) | null = null
let unsubscribeAppInfoUpdated: (() => void) | null = null
let unsubscribeTableDetailRow: (() => void) | null = null
let unsubscribeDetailUrlWatch: (() => void) | null = null
let unsubscribeWorkspaceOpenWorkstation: (() => void) | null = null

// 🔥 重新关联 tabs 的 node 信息（使用 Composable）
// 🔥 不再使用 Tab，删除 restoreTabsNodes 函数

// 🔥 初始化 RouteManager（路由管理器）
let routeManager: RouteManager | null = null

onMounted(async () => {
  // 🔥 如果已存在 routeManager，先销毁（避免热更新时重复创建）
  if (routeManager) {
    routeManager.destroy()
    routeManager = null
  }
  
  // 🔥 初始化 RouteManager（不再使用 Tab）
  routeManager = new RouteManager(router, route, eventBus)
  
  // 监听函数加载完成事件
  // 🔥 监听函数加载完成事件，更新 currentFunctionDetail
  unsubscribeFunctionLoaded = eventBus.on(WorkspaceEvent.functionLoaded, (payload: { node: any, detail: FunctionDetail }) => {
    // 只有当加载的函数是当前函数时，才更新 currentFunctionDetail
    if (currentFunction.value && 
        (currentFunction.value.id === payload.node.id || 
         currentFunction.value.full_code_path === payload.node.full_code_path)) {
      currentFunctionDetail.value = payload.detail
      // 清除权限错误（因为函数已成功加载）
      permissionErrorStore.clearError()
    }
  })

  // 监听服务树加载完成事件
  unsubscribeServiceTreeLoaded = eventBus.on(WorkspaceEvent.serviceTreeLoaded, async (payload: { app: any, tree: any[], expandedKeys?: number[] }) => {
    console.log('[WorkspaceView] serviceTreeLoaded 事件触发，expandedKeys:', payload.expandedKeys)
    // 状态已通过 StateManager 自动更新
    // ⭐ 更新 expandedKeys（如果后端返回了）
    // 注意：使用展开运算符创建新数组，避免引用共享导致的响应式问题
    if (payload.expandedKeys && payload.expandedKeys.length > 0) {
      console.log('[WorkspaceView] 更新 expandedKeys.value 为:', payload.expandedKeys)
      expandedKeys.value = [...payload.expandedKeys]
    } else {
      console.log('[WorkspaceView] 清空 expandedKeys.value')
      expandedKeys.value = []
    }
    
    // 🔥 切换工作空间后，如果是根路径，自动选中根节点显示详情
    await nextTick()
    const fullPath = extractWorkspacePath(route.path)
    if (fullPath) {
      const pathSegments = fullPath.split('/').filter(Boolean)
      // 如果是根路径（只有 user/app），自动选中根节点
      if (pathSegments.length === 2 && payload.tree && payload.tree.length > 0) {
        const rootPath = '/' + pathSegments.join('/')
        const rootNode = findNodeByPath(payload.tree, rootPath)
        if (rootNode && rootNode.type === 'package') {
          console.log('[WorkspaceView] 切换工作空间后，自动选中根节点:', rootNode.name, rootPath)
          // 触发节点点击，显示根目录详情
          applicationService.handleNodeClick(rootNode as any)
        }
      }
    }
  })
  
  // 监听应用切换事件，开始加载服务树
  unsubscribeAppSwitched = eventBus.on(WorkspaceEvent.appSwitched, (payload: { app: any }) => {
    // 应用切换事件处理
  })

  unsubscribeScheduledTaskCreated = eventBus.on(WorkspaceEvent.scheduledTaskCreated, () => {
    hasScheduledTasksForCurrentPath.value = true
    functionActiveTab.value = 'scheduledTask'
  })

  // 监听应用信息更新事件（用于更新应用列表中的 app.id）
  unsubscribeAppInfoUpdated = eventBus.on('workspace:app-info-updated' as any, (payload: { app: AppType }) => {
    // 更新应用列表中的 app 信息
    const index = appList.value.findIndex((a: AppType) => a.code === payload.app.code)
    if (index !== -1) {
      appList.value[index] = { ...appList.value[index], ...payload.app }
    }
  })

  // 从路由加载应用
  // 优化：如果路由中有应用信息，直接使用合并接口获取，不需要先加载整个应用列表
  await routingLoadAppFromRoute()
  
  // 注意：应用列表在用户点击应用切换器时才加载（AppSwitcher 的 handleVisibleChange 会触发 load-apps 事件）
  
  // 🔥 设置路由监听
  setupRouteWatch()
  
  // 进入 /workspace/:user（仅 username、无 app）时自动弹出「选择工作空间」，且必须选一个或创建，不可关闭
  if (route.name === 'workspace-user') {
    nextTick(() => workspaceHeaderRef.value?.openWorkspaceListDialog(true))
  }
})

// 🔥 监听服务树变化，展开目录树
watch(() => serviceTree.value.length, (newLength: number) => {
  if (newLength > 0 && currentApp.value) {
    console.log('[WorkspaceView] serviceTree 变化，准备展开目录树')
    // 展开目录树
    expandCurrentRoutePath()
  }
}, { immediate: true })


// 🔥 监听当前函数变化，清除旧的函数详情和权限错误
watch(() => currentFunction.value?.id, (newId: number | undefined, oldId: number | undefined) => {
  // 当切换函数时，先清空旧的函数详情，避免显示上一个函数的详情
  if (newId !== oldId && oldId !== undefined) {
    // ⭐ 清空旧的函数详情，这样如果新函数加载失败，不会显示旧函数的详情
    currentFunctionDetail.value = null
    // 清除旧的权限错误（新的权限错误会在加载失败时重新设置）
    permissionErrorStore.clearError()
  }
})

// 当前函数路径或 tabs 可见性变化时，刷新状态并修正当前 tab
watch(
  () => [currentFunction.value?.full_code_path, showFunctionPermissionRequestTab.value, showFormOperateLogTab.value] as const,
  () => {
    if (!showFunctionPermissionRequestTab.value && functionActiveTab.value === 'permission') {
      functionActiveTab.value = 'content'
    }
    if (!showFormOperateLogTab.value && functionActiveTab.value === 'operateLog') {
      functionActiveTab.value = 'content'
    }
    refreshScheduledTasksCountForCurrentPath()
  },
  { immediate: true }
)

// 🔥 监听 queryTab 变化，处理 create/edit/detail 模式
watch(queryTab, async (newTab: string, oldTab: string) => {
  if (newTab === 'create' || newTab === 'edit') {
    // create/edit 模式需要确保函数详情已加载
    if (!currentFunction.value) {
      return
    }
    
    // 如果函数详情未加载，触发加载
    if (!currentFunctionDetail.value) {
      await applicationService.handleNodeClick(currentFunction.value)
    }
  } else if (newTab === 'detail') {
    // detail 模式需要确保函数详情已加载，并且表格数据已加载
    if (!currentFunction.value) {
      return
    }
    
    // 如果函数详情未加载，触发加载
    if (!currentFunctionDetail.value) {
      await applicationService.handleNodeClick(currentFunction.value)
    }
    
    // detail 模式会在另一个 watch 中处理（监听 route.query.id）
  }
}, { immediate: false })

// ⭐ 监听路由 query 参数，支持通过 tab 参数指定要打开的函数 tab
watch(
  () => route.query.tab,
  (tab) => {
    const normalizedTab = normalizeQueryTab(tab)

    if (normalizedTab === 'permissionRequest' && showFunctionPermissionRequestTab.value) {
      functionActiveTab.value = 'permission'
      functionPermissionTab.value = 'request'
      loadCurrentFunctionPermissionTab()
    } else if (normalizedTab === 'permissionManage' && showFunctionPermissionRequestTab.value) {
      functionActiveTab.value = 'permission'
      functionPermissionTab.value = 'manage'
      loadCurrentFunctionPermissionTab()
    } else if (normalizedTab === 'permission' && showFunctionPermissionRequestTab.value) {
      functionActiveTab.value = 'permission'
      loadCurrentFunctionPermissionTab()
    } else if (normalizedTab === 'operateLog' && showFormOperateLogTab.value) {
      functionActiveTab.value = 'operateLog'
      nextTick(() => {
        formOperateLogSectionRef.value?.loadLogs({ page: 1 })
      })
    } else if (functionActiveTab.value !== 'scheduledTask') {
      functionActiveTab.value = 'content'
    }
  },
  { immediate: true }
)

// 🔥 监听路由 query 变化，处理 _tab 参数
watch(() => route.query._tab, async (newTab: any) => {
  if (newTab === 'create' || newTab === 'edit') {
    // 确保当前函数已加载
    if (!currentFunction.value) {
      return
    }
    
    // 🔥 移除缓存后，切换函数时总是重新加载函数详情
    if (currentFunction.value && currentFunction.value.type === 'function') {
      await applicationService.handleNodeClick(currentFunction.value)
    }
  } else if (newTab === 'detail') {
    // detail 模式会在另一个 watch 中处理（监听 route.query.id）
    // 这里只需要确保函数详情已加载
    if (!currentFunction.value) {
      return
    }
    
    // 🔥 移除缓存后，切换函数时总是重新加载函数详情
    if (currentFunction.value && currentFunction.value.type === 'function') {
      await applicationService.handleNodeClick(currentFunction.value)
    }
  }
}, { immediate: false })


onUnmounted(() => {
  // 清理函数详情
  currentFunctionDetail.value = null
  
  if (unsubscribeFunctionLoaded) {
    unsubscribeFunctionLoaded()
  }
  if (unsubscribeServiceTreeLoaded) {
    unsubscribeServiceTreeLoaded()
  }
  if (unsubscribeAppSwitched) {
    unsubscribeAppSwitched()
  }
  if (unsubscribeScheduledTaskCreated) {
    unsubscribeScheduledTaskCreated()
  }
  if (unsubscribeAppInfoUpdated) {
    unsubscribeAppInfoUpdated()
  }
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
  background: var(--el-bg-color);
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
