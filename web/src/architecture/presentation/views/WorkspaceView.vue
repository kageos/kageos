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
    <!-- 顶部导航栏 -->
    <WorkspaceHeader />

    <div class="workspace-view">
      <!-- 左侧服务目录树 -->
      <div class="left-sidebar" :class="{ 'sidebar-collapsed': !showLeftSidebar }">
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
          @delete-doc="handleDeleteDoc"
          @fork-group="handleForkGroup"
          @copy-link="handleCopyLink"
          @delete-function="handleDeleteFunction"
          @publish-to-hub="handlePublishToHub"
          @push-to-hub="handlePushToHub"
          @pull-from-hub="handlePullFromHub"
          @refresh-tree="handleRefreshTree"
          @update-history="handleUpdateHistory"
        />
      </div>

      <!-- 左侧边栏控制按钮 -->
      <div class="left-sidebar-controls">
        <el-button
          v-if="!showLeftSidebar"
          link
          @click="toggleLeftSidebar"
          class="sidebar-toggle"
          title="显示服务目录"
        >
          <el-icon><ArrowRight /></el-icon>
          显示目录
        </el-button>
        
        <el-button
          v-if="showLeftSidebar"
          link
          @click="toggleLeftSidebar"
          class="sidebar-toggle"
          title="隐藏服务目录"
        >
          <el-icon><ArrowLeft /></el-icon>
          隐藏目录
        </el-button>
      </div>

      <!-- 中间函数渲染区域 -->
      <div class="function-renderer">
        <!-- 右侧边栏控制按钮 -->
        <div class="sidebar-controls" v-if="currentFunction && currentFunction.type === 'function'">
          <div class="right-controls">
            <el-button
              v-if="!showRightSidebar"
              link
              @click="toggleRightSidebar"
              class="sidebar-toggle"
              title="显示函数信息"
            >
              <el-icon><ArrowLeft /></el-icon>
              显示函数信息
            </el-button>
            
            <el-button
              v-if="showRightSidebar"
              link
              @click="toggleRightSidebar"
              class="sidebar-toggle"
              title="隐藏函数信息"
            >
              <el-icon><ArrowRight /></el-icon>
              隐藏函数信息
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
                :function-detail="editFunctionDetail"
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
        
        <!-- 🔥 文档详情页面 -->
        <DocView
          v-if="currentFunction && currentFunction.type === 'docs'"
          :node="currentFunction"
          @deleted="handleDocDeleted"
        />
        
        <!-- 🔥 服务目录详情页面（包括 app 根节点和 package 节点） -->
        <PackageDetailView
          v-else-if="currentFunction && (currentFunction.type === 'package' || currentFunction.type === 'app') && !selectedAgent"
          :package-node="currentFunction"
          @generate-system="handlePackageGenerateSystem"
          @refresh="handleRefreshTree"
        />
        
        <!-- 🔥 点击目录节点时根据选择的智能体显示不同的聊天面板 -->
        <div v-else-if="currentFunction && currentFunction.type === 'package' && selectedAgent" class="ai-chat-wrapper">
          <!-- 根据 chat_type 选择不同的渲染方式 -->
          <AIChatPanel
            v-if="selectedAgent.chat_type === 'function_gen'"
            ref="aiChatPanelRef"
            :agent-id="selectedAgent.id"
            :tree-id="currentFunction.id"
            :package="currentFunction.code"
            :current-node-name="currentFunction.name"
            :existing-files="existingFilesInPackage"
            @close="handleCloseAIChat"
          />
          <!-- 可以在这里添加其他 chat_type 的渲染组件 -->
          <!-- 例如：<TaskChatPanel v-else-if="selectedAgent.chat_type === 'chat-task'" ... /> -->
        </div>
        
        <!-- 函数详情区域（正常模式 - 函数节点） -->
        <div v-else-if="currentFunction && currentFunction.type === 'function'" class="function-content-wrapper">
          <div class="function-content">
            <!-- ⭐ 权限申请 tab（仅管理员可见） -->
            <div v-if="showFunctionPermissionRequestTab" class="function-tabs-wrapper">
              <el-tabs v-model="functionActiveTab" type="card" @tab-change="handleFunctionTabChange" class="function-detail-tabs">
                <!-- 函数内容 tab -->
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

                <!-- 权限申请 tab -->
                <el-tab-pane name="permissionRequest">
                  <template #label>
                    <el-badge :value="currentFunction?.pending_count || 0" :hidden="!currentFunction?.pending_count || currentFunction.pending_count === 0" :max="99">
                      <span>权限申请</span>
                    </el-badge>
                  </template>
                  <div class="tab-content">
                    <PermissionRequestList
                      ref="functionPermissionRequestListRef"
                      :resource-path="currentFunction?.full_code_path"
                      :auto-load="functionActiveTab === 'permissionRequest'"
                    />
                  </div>
                </el-tab-pane>

                <!-- 权限管理 tab -->
                <el-tab-pane name="permissionManage">
                  <template #label>
                    <span>权限管理</span>
                  </template>
                  <div class="tab-content">
                    <PermissionManageList
                      ref="functionPermissionManageListRef"
                      :resource-path="currentFunction?.full_code_path"
                      :user="currentApp?.user"
                      :app="currentApp?.code"
                      :auto-load="functionActiveTab === 'permissionManage'"
                    />
                  </div>
                </el-tab-pane>
              </el-tabs>
            </div>

            <!-- 非管理员或没有权限申请 tab 时，显示原来的内容 -->
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

      <!-- 右侧函数信息面板 -->
      <div 
        v-if="currentFunction && currentFunction.type === 'function' && showRightSidebar" 
        class="right-sidebar"
        :class="{ 'sidebar-collapsed': !showRightSidebar }"
      >
        <FunctionInfoPanel 
          :function-data="currentFunctionDetail" 
          :function-node="currentFunction"
        />
      </div>
    </div>

    <!-- 智能体选择对话框 -->
    <AgentSelectDialog
      v-model="agentSelectDialogVisible"
      :tree-id="currentFunction?.id || null"
      :package="currentFunction?.code || ''"
      :current-node-name="currentFunction?.name || ''"
      @confirm="handleAgentSelect"
    />

    <!-- 应用切换器（底部固定） -->
    <!-- 始终显示，即使应用列表为空，让用户可以创建应用 -->
    <AppSwitcher
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

    <!-- 创建工作空间对话框 -->
    <el-dialog
      v-model="createAppDialogVisible"
      title="创建新工作空间"
      width="800px"
      :close-on-click-modal="false"
      @close="resetCreateAppForm"
    >
      <el-form :model="createAppForm" label-width="90px">
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
          <el-input
            v-model="createAppForm.code"
            placeholder="请输入英文标识（如：tsinghua、pku_gsm）"
            maxlength="50"
            show-word-limit
            clearable
            @input="createAppForm.code = createAppForm.code.toLowerCase()"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            英文标识只能包含小写字母、数字和下划线，长度 2-50 个字符
          </div>
        </el-form-item>
        <el-form-item label="是否公开">
          <el-switch
            v-model="createAppForm.is_public"
            active-text="公开"
            inactive-text="私有"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            公开的工作空间可以被其他用户搜索到，私有的工作空间只有您自己可以看到
          </div>
        </el-form-item>
        <el-form-item label="管理员">
          <UserSearchInput
            v-model="adminsArray"
            placeholder="搜索并选择管理员（可多选）"
            :multiple="true"
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
      <el-form :model="createDocsForm" label-width="90px">
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
            placeholder="请输入文档代码（英文，用于URL）"
            maxlength="50"
            show-word-limit
            clearable
            @input="createDocsForm.code = createDocsForm.code.toLowerCase().replace(/[^a-z0-9_]/g, '')"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            文档代码只能包含小写字母、数字和下划线
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

    <!-- 创建服务目录对话框 -->
    <el-dialog
      v-model="createDirectoryDialogVisible"
      :title="currentParentNode ? `在「${currentParentNode.name || currentParentNode.code}」下创建服务目录` : '创建服务目录'"
      width="520px"
      :close-on-click-modal="false"
      @close="handleCloseCreateDirectoryDialog"
    >
      <el-form :model="createDirectoryForm" label-width="90px">
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
          <el-button type="primary" @click="() => handleSubmitCreateDirectory(() => currentApp.value)" :loading="creatingDirectory">
            创建
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- Fork 函数组对话框 -->
    <FunctionForkDialog
      v-model="forkDialogVisible"
      :source-full-group-code="forkSourceGroupCode || undefined"
      :source-group-name="forkSourceGroupName || undefined"
      :current-app="currentApp || undefined"
      @success="handleForkSuccess"
    />

    <!-- 发布到应用中心对话框 -->
    <PublishToHubDialog
      v-model="publishToHubDialogVisible"
      :selected-node="publishSelectedNode"
      :current-app="currentApp || undefined"
      @success="handlePublishSuccess"
    />
    <PushToHubDialog
      v-model="pushToHubDialogVisible"
      :selected-node="pushSelectedNode"
      :current-app="currentApp || undefined"
      @success="handlePushSuccess"
    />
    <PullFromHubDialog
      v-model="pullFromHubDialogVisible"
      :current-app="currentApp || undefined"
      :initial-hub-link="pastedHubLink"
      @success="handlePullSuccess"
    />

    <!-- 变更记录对话框 -->
    <DirectoryUpdateHistoryDialog
      v-model="updateHistoryDialogVisible"
      :mode="updateHistoryMode"
      :app-id="updateHistoryAppId"
      :app-version="updateHistoryAppVersion"
      :full-code-path="updateHistoryFullCodePath"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch, ref, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElNotification, ElDialog, ElForm, ElFormItem, ElInput, ElButton, ElIcon, ElSwitch, ElSkeleton } from 'element-plus'
import { InfoFilled, ArrowLeft, ArrowRight } from '@element-plus/icons-vue'
import { eventBus, WorkspaceEvent, RouteEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import { RouteManager } from '../../infrastructure/routeManager'
import { useAuthStore } from '@/stores/auth'
import ServiceTreePanel from '@/components/ServiceTreePanel.vue'
import AppSwitcher from '@/components/AppSwitcher.vue'
import FunctionForkDialog from '@/components/FunctionForkDialog.vue'
import PublishToHubDialog from '@/components/PublishToHubDialog.vue'
import PushToHubDialog from '@/components/PushToHubDialog.vue'
import PullFromHubDialog from '@/components/PullFromHubDialog.vue'
import DirectoryUpdateHistoryDialog from '@/components/DirectoryUpdateHistoryDialog.vue'
import FormView from './FormView.vue'
import TableView from './TableView.vue'
import ChartView from './ChartView.vue'
import WorkspaceHeader from '../components/WorkspaceHeader.vue'
import FunctionBreadcrumb from '../components/FunctionBreadcrumb.vue'
import TableRowDetailDrawer from '../components/TableRowDetailDrawer.vue'
import PermissionDeniedView from '../components/PermissionDeniedView.vue'
import AIChatPanel from '../components/AIChatPanel.vue'
import AgentSelectDialog from '@/components/Agent/AgentSelectDialog.vue'
import PackageDetailView from '../components/PackageDetailView.vue'
import DocView from '../components/DocView.vue'
import FunctionInfoPanel from '../components/FunctionInfoPanel.vue'
import UserSearchInput from '@/components/UserSearchInput.vue'
import UsersWidget from '../widgets/UsersWidget.vue'
import PermissionRequestList from '@/components/Permission/PermissionRequestList.vue'
import PermissionManageList from '@/components/Permission/PermissionManageList.vue'
import type { ServiceTree, App } from '../../domain/services/WorkspaceDomainService'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import { WidgetType } from '@/core/constants/widget'
import type { FunctionDetail } from '../../domain/interfaces/IFunctionLoader'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/types'
// 🔥 导入 Composable
import { useWorkspaceRouting } from '../composables/useWorkspaceRouting'
import { RouteSource } from '@/utils/routeSource'
import { useWorkspaceDetail } from '../composables/useWorkspaceDetail'
import { useWorkspaceApp } from '../composables/useWorkspaceApp'
import { useWorkspaceServiceTree } from '../composables/useWorkspaceServiceTree'
import { findNodeByPath, findNodeById, getDirectChildFunctionCodes } from '../utils/workspaceUtils'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { resolveWorkspaceUrl } from '@/utils/route'
import { getAgentList, type AgentInfo } from '@/api/agent'
import { isLinkNavigation as checkLinkNavigation, LINK_TYPE_QUERY_KEY } from '@/utils/linkNavigation'
import { hasPermission, TablePermissions, buildPermissionApplyURL } from '@/utils/permission'
import { usePermissionErrorStore } from '@/stores/permissionError'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

// 依赖注入（使用 IServiceProvider 接口，遵循依赖倒置原则）
const serviceProvider: IServiceProvider = serviceFactory
const stateManager = serviceProvider.getWorkspaceStateManager()
const applicationService = serviceProvider.getWorkspaceApplicationService()
const domainService = serviceProvider.getWorkspaceDomainService()

// 从状态管理器获取状态
const serviceTree = computed(() => stateManager.getServiceTree())
const currentFunction = computed(() => stateManager.getCurrentFunction())
const currentAppFromState = computed(() => stateManager.getCurrentApp())

// ⭐ 需要自动展开的节点ID列表（从后端返回）
const expandedKeys = ref<number[]>([])

// 🔥 不再使用 Tab 功能，简化系统

const currentApp = computed<AppType | null>(() => {
  const app = currentAppFromState.value
  if (!app) return null
  // 从 appList 中查找对应的应用（确保使用最新的应用数据）
  const foundApp = appList.value.find((a: AppType) => a.id === app.id || (a.user === app.user && a.code === app.code))
  return foundApp || {
    id: app.id,
    user: app.user,
    code: app.code,
    name: app.name,
    nats_id: app.nats_id || 0,
    host_id: app.host_id || 0,
    status: (app.status || 'enabled') as 'enabled' | 'disabled',
    version: app.version || '',
    created_at: app.created_at || '',
    updated_at: app.updated_at || '',
    admins: app.admins || '' // ⭐ 包含 admins 字段
  }
})

const {
  appList,
  loadingApps,
  createAppDialogVisible,
  creatingApp,
  createAppForm,
  adminsArray,
  loadAppList,
  handleSwitchApp: appHandleSwitchApp,
  showCreateAppDialog,
  resetCreateAppForm,
  submitCreateApp: appSubmitCreateApp,
  handleUpdateApp,
  handleDeleteApp: appHandleDeleteApp
} = useWorkspaceApp()

const {
  createDirectoryDialogVisible,
  creatingDirectory,
  currentParentNode,
  createDirectoryForm,
  handleCreateDirectory: serviceTreeHandleCreateDirectory,
  resetCreateDirectoryForm,
  handleSubmitCreateDirectory: serviceTreeHandleSubmitCreateDirectory,
  expandCurrentRoutePath: serviceTreeExpandCurrentRoutePath,
  checkAndExpandForkedPaths: serviceTreeCheckAndExpandForkedPaths,
  handleCopyLink
} = useWorkspaceServiceTree()

// 管理员字段配置（用于 UsersWidget）
const adminsField = computed<FieldConfig>(() => ({
  code: 'admins',
  name: '管理员',
  widget: {
    type: WidgetType.USERS,
    config: {}
  }
}))

// 管理员字段值（用于 UsersWidget）
const adminsFieldValue = computed<FieldValue>(() => {
  if (!createDirectoryForm.value.admins || !createDirectoryForm.value.admins.trim()) {
    return {
      raw: null,
      display: '',
      meta: {}
    }
  }
  
  const admins = createDirectoryForm.value.admins.split(',').map(s => s.trim()).filter(s => s)
  return {
    raw: admins.join(','),
    display: admins.join(', '),
    meta: {}
  }
})

// 处理管理员字段变化
function handleAdminsChange(value: FieldValue) {
  createDirectoryForm.value.admins = value.raw || ''
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
  detailFormRendererRef,
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
  checkAndExpandForkedPaths: () => serviceTreeCheckAndExpandForkedPaths(
    () => serviceTree.value,
    () => serviceTreePanelRef.value,
    () => currentApp.value
  ),
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
      
      if (query[fieldCode] !== undefined && query[fieldCode] !== null && query[fieldCode] !== '') {
        const value = query[fieldCode]
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
const forkDialogVisible = ref(false)
const forkSourceGroupCode = ref('')
const forkSourceGroupName = ref('')

// 发布到应用中心对话框
const publishToHubDialogVisible = ref(false)
const publishSelectedNode = ref<ServiceTreeType | null>(null)
const pushToHubDialogVisible = ref(false)
const pushSelectedNode = ref<ServiceTreeType | null>(null)
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

// 左侧服务目录树显示状态
const showLeftSidebar = ref(true)

// 右侧函数信息面板显示状态
const showRightSidebar = ref(true)

// 函数详情 tab 相关
const functionActiveTab = ref('content')
const functionPermissionRequestListRef = ref<InstanceType<typeof PermissionRequestList> | null>(null)
const functionPermissionManageListRef = ref<InstanceType<typeof PermissionManageList> | null>(null)

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
  
  // 检查是否是管理员
  if (!currentFunction.value.admins || !authStore.user?.username) {
    return false
  }
  
  const admins = currentFunction.value.admins.split(',').map((a: string) => a.trim()).filter(Boolean)
  return admins.includes(authStore.user.username)
})

// 处理函数 tab 切换
const handleFunctionTabChange = (tabName: string) => {
  if (tabName === 'permissionRequest' && functionPermissionRequestListRef.value) {
    // 切换到权限申请 tab 时，触发加载
    nextTick(() => {
      functionPermissionRequestListRef.value?.loadRequests()
    })
  } else if (tabName === 'permissionManage' && functionPermissionManageListRef.value) {
    // 切换到权限管理 tab 时，触发加载
    nextTick(() => {
      functionPermissionManageListRef.value?.loadPermissions()
    })
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

// AI 对话框相关
const agentSelectDialogVisible = ref(false)
const selectedAgent = ref<AgentInfo | null>(null)
const aiChatPanelRef = ref<InstanceType<typeof AIChatPanel> | null>(null)

// 处理智能体选择
function handleAgentSelect(agent: AgentInfo) {
  selectedAgent.value = agent
  agentSelectDialogVisible.value = false
  
  // 选择智能体后，通知 AIChatPanel 创建新会话
  // 使用 nextTick 确保组件已渲染
  nextTick(() => {
    if (aiChatPanelRef.value && typeof (aiChatPanelRef.value as any).handleAgentSelect === 'function') {
      (aiChatPanelRef.value as any).handleAgentSelect(agent)
    }
  })
  
  // 如果路由不匹配，更新路由
  if (currentFunction.value?.full_code_path && currentApp.value) {
    const targetPath = buildWorkspacePath(currentFunction.value.full_code_path)
    if (route.path !== targetPath) {
      eventBus.emit(RouteEvent.updateRequested, {
        path: targetPath,
        query: {},
        replace: true,
        preserveParams: {
          state: false,
          table: false,
          search: false
        },
        source: RouteSource.AGENT_SELECT
      })
    }
  }
}

// 处理服务目录的生成系统按钮点击
function handlePackageGenerateSystem(agent: AgentInfo) {
  selectedAgent.value = agent
  // 设置当前函数（确保 AIChatPanel 能正确显示）
  if (currentFunction.value && currentFunction.value.type === 'package') {
    applicationService.triggerNodeClick(currentFunction.value)
  }
  // 触发 AIChatPanel 新建会话（使用 nextTick 确保组件已渲染）
  nextTick(() => {
    if (aiChatPanelRef.value && typeof (aiChatPanelRef.value as any).handleAgentSelect === 'function') {
      // 调用 handleAgentSelect 会创建新会话（清空 sessionId，显示欢迎消息）
      (aiChatPanelRef.value as any).handleAgentSelect(agent)
    }
  })
}

// 关闭 AI 聊天面板
function handleCloseAIChat() {
  selectedAgent.value = null
  // 如果当前是目录节点，清除当前函数选择
  if (currentFunction.value?.type === 'package') {
    applicationService.triggerNodeClick(null as any)
  }
}

// 获取当前 package 下的子节点文件名（用于确保生成的文件名唯一）
const existingFilesInPackage = computed(() => {
  if (!currentFunction.value || currentFunction.value.type !== 'package') {
    return []
  }
  
  // 从 serviceTree 中查找当前节点
  const currentNode = findNodeById(serviceTree.value, currentFunction.value.id)
  if (!currentNode) {
    return []
  }
  
  // 获取直接子节点（只收集一级子节点，type 为 'function' 的）
  return getDirectChildFunctionCodes(currentNode)
})

// ⭐ 权限检查：是否有表格更新权限
const canUpdateTable = computed(() => {
  const node = currentFunction.value
  if (!node) return true  // 如果没有节点信息，默认允许（向后兼容）
  return hasPermission(node, TablePermissions.update)
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
  eventBus.on('table:detail-row', async ({ row, index, tableData }: { row: Record<string, any>, index?: number, tableData?: any[] }) => {
    await openDetailDrawer(row, index, tableData)
  })
  
  // 🔥 Tab 功能已删除，相关事件监听已移除
  
  // 🔥 设置 URL 监听（使用 Composable）
  setupUrlWatch()
  
  // 🔥 添加全局粘贴监听
  document.addEventListener('paste', handleGlobalPaste)
})

onUnmounted(() => {
  // 🔥 移除全局粘贴监听
  document.removeEventListener('paste', handleGlobalPaste)
})



// 转换 loadingTree 为 boolean (避免 computed 类型问题)
const loading = computed(() => stateManager.isLoading())

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
      // 路由已匹配，直接触发节点点击
      applicationService.triggerNodeClick(serviceTree)
    } else {
      // 路由未匹配，先触发节点点击，然后更新路由
      applicationService.triggerNodeClick(serviceTree)
      handlePackageNodeRoute(serviceTree, 'workspace-node-click-docs')
    }
  } else if (serviceTree.type === 'app') {
    // ⭐ app 类型节点（工作空间根节点），更新路由并触发节点点击
    const targetPath = buildWorkspacePath(serviceTree.full_code_path || '')
    if (route.path === targetPath) {
      // 路由已匹配，直接触发节点点击
      applicationService.triggerNodeClick(serviceTree)
    } else {
      // 路由未匹配，先触发节点点击，然后更新路由
      applicationService.triggerNodeClick(serviceTree)
      handlePackageNodeRoute(serviceTree, 'workspace-node-click-app')
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
    // ⭐ docs 类型节点，也需要更新路由
    handlePackageNodeRoute(node, 'breadcrumb-node-click-docs')
  } else if (node.type === 'app') {
    // ⭐ app 类型节点（工作空间根节点），更新路由
    handlePackageNodeRoute(node, 'breadcrumb-node-click-app')
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

  // ⭐ 验证文档内容
  if (!createDocsForm.value.content.trim()) {
    ElMessage.warning('请输入文档内容')
    return
  }

  creatingDocs.value = true
  try {
    const { createServiceTree } = await import('@/api/service-tree')
    const parentId = currentDocsParentNode.value?.id || 0
    const response = await createServiceTree({
      user: currentApp.value.user,
      app: currentApp.value.code,
      name: createDocsForm.value.name.trim(),
      code: createDocsForm.value.code.trim(),
      parent_id: parentId,
      type: 'docs',
      description: createDocsForm.value.description.trim() || '',
      tags: createDocsForm.value.tags.trim() || '',
      doc_title: createDocsForm.value.name.trim(),  // ⭐ 使用名称作为文档标题
      doc_content: createDocsForm.value.content.trim(),  // ⭐ 文档内容
      doc_format: 'markdown',  // ⭐ 文档格式
      doc_summary: createDocsForm.value.summary.trim() || ''  // ⭐ 文档摘要
    })

    // ⭐ 响应拦截器已经处理了，成功时返回的是 data 对象（ServiceTree），不是 { data: ServiceTree }
    if (response && response.id) {
      ElMessage.success('文档节点创建成功')
      // ⭐ 立即关闭弹窗，不等待后续操作
      createDocsDialogVisible.value = false
      
      // 刷新服务树（异步执行，不阻塞弹窗关闭）
      handleRefreshTree().then(() => {
        // 点击新创建的节点
        if (response.id) {
          const newNode = findNodeById(serviceTree.value, response.id)
          if (newNode) {
            handleNodeClick(newNode)
          }
        }
      }).catch((err) => {
        console.error('刷新服务树失败:', err)
        // 即使刷新失败，也尝试点击新创建的节点
        if (response.id) {
          const newNode = findNodeById(serviceTree.value, response.id)
          if (newNode) {
            handleNodeClick(newNode)
          }
        }
      })
    } else {
      // 如果响应数据为空，也关闭弹窗并提示
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

    const { deleteDoc } = await import('@/api/service-tree')
    await deleteDoc(node.id)
    
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
const handleForkGroup = (node: ServiceTreeType | null) => {
  // 如果传入了节点，使用它；否则打开对话框让用户选择
  if (node) {
    if (!node.full_group_code) {
      ElNotification.warning({
        title: '提示',
        message: '该节点没有函数组代码，无法克隆'
      })
      return
    }
    forkSourceGroupCode.value = node.full_group_code
    forkSourceGroupName.value = node.group_name || node.name || ''
  } else {
    // 没有传入节点，清空预设值，让用户在对话框中选择
    forkSourceGroupCode.value = ''
    forkSourceGroupName.value = ''
  }
  forkDialogVisible.value = true
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

// 处理从应用中心拉取
const handlePullFromHub = () => {
  pastedHubLink.value = ''  // 清空之前的链接（手动打开对话框时）
  pullFromHubDialogVisible.value = true
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

    // 调用删除 API
    const { deleteFunction } = await import('@/api/function')
    await deleteFunction(node.id)

    ElMessage.success('删除成功')

    // 如果删除的是当前选中的函数，清空选中状态
    if (currentFunction.value && currentFunction.value.id === node.id) {
      currentFunction.value = null
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
  if (currentApp.value) {
    const app: App = {
      id: currentApp.value.id,
      user: currentApp.value.user,
      code: currentApp.value.code,
      name: currentApp.value.name
    }
    await domainService.loadServiceTree(app)
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
  
  console.log('[WorkspaceView] 打开变更记录对话框', {
    mode: updateHistoryMode.value,
    appId: updateHistoryAppId.value,
    appVersion: updateHistoryAppVersion.value,
    fullCodePath: updateHistoryFullCodePath.value
  })
  
  updateHistoryDialogVisible.value = true
}

// 发布成功后的回调
const handlePublishSuccess = async () => {
  // 刷新服务目录树
  if (currentApp.value) {
    const app: App = {
      id: currentApp.value.id,
      user: currentApp.value.user,
      code: currentApp.value.code,
      name: currentApp.value.name
    }
    await domainService.loadServiceTree(app)
  }
}

// 推送成功后的回调
const handlePushSuccess = async () => {
  // 刷新服务目录树
  if (currentApp.value) {
    const app: App = {
      id: currentApp.value.id,
      user: currentApp.value.user,
      code: currentApp.value.code,
      name: currentApp.value.name
    }
    await domainService.loadServiceTree(app)
  }
}

// 拉取成功后的回调
const handlePullSuccess = async () => {
  // 清空粘贴的链接
  pastedHubLink.value = ''
  // 刷新服务目录树
  if (currentApp.value) {
    const app: App = {
      id: currentApp.value.id,
      user: currentApp.value.user,
      code: currentApp.value.code,
      name: currentApp.value.name
    }
    await domainService.loadServiceTree(app)
  }
}

// Fork 成功后的回调
const handleForkSuccess = () => {
  // 刷新服务目录树
  if (currentApp.value) {
    const appForService: App = {
      id: currentApp.value.id,
      user: currentApp.value.user,
      code: currentApp.value.code,
      name: currentApp.value.name,
      nats_id: currentApp.value.nats_id || 0,
      host_id: currentApp.value.host_id || 0,
      status: currentApp.value.status || 'enabled',
      version: currentApp.value.version || '',
      created_at: currentApp.value.created_at || '',
      updated_at: currentApp.value.updated_at || ''
    }
    applicationService.triggerAppSwitch(appForService)
  }
  ElNotification.success({
    title: '成功',
    message: '克隆完成！请刷新页面查看新功能'
  })
}

// 🔥 展开当前路由对应的路径（使用 Composable）
const expandCurrentRoutePath = () => {
  serviceTreeExpandCurrentRoutePath(
    () => serviceTree.value,
    () => serviceTreePanelRef.value,
    () => currentApp.value
  )
}

// 🔥 检查并展开 forked 路径（使用 Composable）
const checkAndExpandForkedPaths = () => {
  serviceTreeCheckAndExpandForkedPaths(
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
let unsubscribeAppInfoUpdated: (() => void) | null = null

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
  routeManager = new RouteManager(
    router,
    route,
    eventBus,
    () => null  // 🔥 Tab 功能已删除
  )
  
  // 🔥 开发环境下启用调试日志
  if (import.meta.env.DEV) {
    routeManager.setDebugLog(true)
  }
  
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
  unsubscribeServiceTreeLoaded = eventBus.on(WorkspaceEvent.serviceTreeLoaded, (payload: { app: any, tree: any[], expandedKeys?: number[] }) => {
    // 状态已通过 StateManager 自动更新
    // ⭐ 更新 expandedKeys（如果后端返回了）
    console.log('[WorkspaceView] serviceTreeLoaded 事件收到:', {
      treeLength: payload.tree?.length || 0,
      expandedKeysLength: payload.expandedKeys?.length || 0,
      expandedKeys: payload.expandedKeys
    })
    if (payload.expandedKeys && payload.expandedKeys.length > 0) {
      expandedKeys.value = payload.expandedKeys
      console.log('[WorkspaceView] ✅ 已更新 expandedKeys:', expandedKeys.value)
    } else {
      expandedKeys.value = []
      console.log('[WorkspaceView] expandedKeys 为空，清空')
    }
  })
  
  // 监听应用切换事件，开始加载服务树
  unsubscribeAppSwitched = eventBus.on(WorkspaceEvent.appSwitched, (payload: { app: any }) => {
    // 应用切换事件处理
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
  // 智能体列表在目录（package）节点时才加载（PackageDetailView 中处理）
  
  // 🔥 设置路由监听
  setupRouteWatch()
})

// 🔥 监听服务树变化，展开目录树
watch(() => serviceTree.value.length, (newLength: number) => {
  if (newLength > 0 && currentApp.value) {
    // 展开目录树
    if (route.query._forked) {
    checkAndExpandForkedPaths()
    } else {
      expandCurrentRoutePath()
  }
  }
}, { immediate: true })

// 🔥 监听当前应用变化，检查 _forked 参数
watch(currentApp, () => {
  if (serviceTree.value.length > 0 && currentApp.value && route.query._forked) {
    nextTick(() => {
      checkAndExpandForkedPaths()
    })
  }
})

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
  (tab: string | string[] | null) => {
    if (tab === 'permissionRequest' && showFunctionPermissionRequestTab.value) {
      functionActiveTab.value = 'permissionRequest'
      // 切换 tab 时触发加载
      nextTick(() => {
        if (functionPermissionRequestListRef.value) {
          functionPermissionRequestListRef.value.loadRequests()
        }
      })
    } else if (tab === 'permissionManage' && showFunctionPermissionRequestTab.value) {
      functionActiveTab.value = 'permissionManage'
      // 切换 tab 时触发加载
      nextTick(() => {
        if (functionPermissionManageListRef.value) {
          functionPermissionManageListRef.value.loadPermissions()
        }
      })
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
  if (unsubscribeAppInfoUpdated) {
    unsubscribeAppInfoUpdated()
  }
})
</script>

<style scoped lang="scss">
.workspace-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

.workspace-view {
  display: flex;
  flex: 1;
  overflow: hidden; /* 防止双滚动条 */
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

// 函数详情 tab 样式（参考旧版本的 card 样式）
.function-detail-tabs {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;

  :deep(.el-tabs__header) {
    margin-top: 20px; /* 与面包屑保持距离 */
    margin-bottom: 20px;
    flex-shrink: 0;
    position: relative;
    z-index: 1; /* 确保 tab header 在面包屑之上 */
    overflow: visible; /* 确保 badge 不被裁剪 */
  }

  :deep(.el-tabs__nav-wrap) {
    overflow: visible !important; /* 确保 badge 不被裁剪 */
  }

  :deep(.el-tabs__nav-scroll) {
    overflow: visible !important; /* 确保 badge 不被裁剪 */
  }

  :deep(.el-tabs__nav) {
    border: none;
    overflow: visible; /* 确保 badge 不被裁剪 */
  }

  :deep(.el-tabs__item) {
    height: 40px;
    line-height: 40px;
    font-size: 14px;
    color: var(--el-text-color-regular);
    border: none;
    background: var(--el-bg-color-overlay);
    margin-right: 4px;
    border-radius: 4px 4px 0 0;
    transition: all 0.3s;
    padding: 0 20px;
    overflow: visible; /* 确保 badge 不被裁剪 */

    &:hover {
      color: var(--el-color-primary);
      opacity: 0.8;
    }

    &.is-active {
      color: var(--el-color-primary);
      background: var(--el-bg-color);
      font-weight: 500;
      opacity: 1;
    }
  }

  :deep(.el-tabs__active-bar) {
    display: none; /* card 类型不需要 active-bar */
  }

  :deep(.el-tabs__content) {
    flex: 1;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    min-height: 0;
  }

  :deep(.el-tab-pane) {
    height: 100%;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    min-height: 0;
  }

  // Badge 样式
  :deep(.el-badge) {
    position: relative;
    display: inline-block;
    
    .el-badge__content {
      font-size: 11px;
      height: 18px;
      line-height: 18px;
      padding: 0 6px;
      min-width: 18px;
      border-radius: 9px;
      z-index: 10; /* 确保 badge 在最上层 */
      box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1); /* 添加阴影，增强可见性 */
    }
  }
}

.function-tabs-wrapper .tab-content {
  padding: 0;
  flex: 1;
  overflow-y: auto;
  min-height: 0;
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
  height: 0;
  -webkit-overflow-scrolling: touch;
}

.left-sidebar {
  width: 300px;
  min-width: 300px;
  border-right: 1px solid var(--el-border-color);
  transition: all 0.3s ease;
  overflow: hidden;
  
  &.sidebar-collapsed {
    width: 0;
    min-width: 0;
    overflow: hidden;
    border-right: none;
  }
}

// 左侧边栏控制按钮
.left-sidebar-controls {
  position: absolute;
  top: 16px;
  left: 16px;
  z-index: 10;
  transition: left 0.3s ease;
  
  // 当左侧边栏收起时，按钮位置保持不变
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

.function-renderer {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
  position: relative;
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

// 右侧函数信息面板
.right-sidebar {
  width: 350px;
  min-width: 350px;
  background-color: var(--el-bg-color);
  border-left: 1px solid var(--el-border-color-light);
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  
  &.sidebar-collapsed {
    width: 0;
    min-width: 0;
    overflow: hidden;
    border-left: none;
  }
}

.ai-chat-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
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
