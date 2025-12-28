<!--
  PackageDetailView - 服务目录详情页面

  职责：
  - 显示服务目录信息
  - 提供"生成系统"按钮，点击后打开智能体选择对话框
-->
<template>
  <div class="package-detail-view">
    <!-- 顶部横幅区域 -->
    <div class="hero-section">
      <div class="hero-content">
        <el-button
          @click="handleBack"
          :icon="ArrowLeft"
          circle
          class="back-button"
          size="large"
        />
        <div class="hero-info">
          <div class="hero-icon-wrapper">
            <img
              v-if="packageNode?.type === 'package'"
              src="/service-tree/custom-folder.svg"
              alt="目录"
              class="hero-icon-img"
            />
            <el-icon v-else class="hero-icon"><Folder /></el-icon>
          </div>
          <div class="hero-text">
            <h1 class="hero-title">{{ packageNode?.name || '服务目录' }}</h1>
            <p class="hero-subtitle" v-if="packageNode?.full_code_path">
              <el-icon class="path-icon"><Link /></el-icon>
              <span class="path-text">{{ packageNode.full_code_path }}</span>
              <el-button
                text
                :icon="CopyDocument"
                @click="handleCopyPath"
                class="path-copy-btn"
                size="small"
                title="复制路径"
              />
              <el-button
                text
                :icon="Clock"
                @click="handleShowUpdateHistory"
                class="path-history-btn"
                size="small"
                title="查看变更记录"
              >
                变更记录
              </el-button>
            </p>
            <p class="hero-description" v-if="packageNode?.description">
              {{ packageNode.description }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- 主要内容区域：左右分栏 -->
    <div class="main-content">
      <!-- 左侧：智能体列表 -->
      <div class="agent-sidebar">
        <div class="sidebar-header">
          <h3 class="sidebar-title">
            <el-icon class="sidebar-icon"><MagicStick /></el-icon>
            选择智能体
          </h3>
        </div>
        <div v-loading="agentLoading" class="agent-list">
          <div
            v-for="agent in agentList"
            :key="agent.id"
            class="agent-card"
            @click="handleAgentClick(agent)"
          >
            <div class="agent-card-header">
              <el-avatar
                :size="48"
                :src="getAgentLogo(agent)"
                class="agent-avatar"
              >
                <span class="agent-avatar-text">{{ getAgentLogoText(agent) }}</span>
              </el-avatar>
              <div class="agent-card-title">
                <div class="agent-name">{{ agent.name }}</div>
                <div class="agent-tags">
                  <el-tag
                    :type="agent.agent_type === 'plugin' ? 'warning' : 'success'"
                    size="small"
                  >
                    {{ agent.agent_type === 'plugin' ? '插件' : agent.agent_type === 'knowledge_only' ? '知识库' : agent.agent_type }}
                  </el-tag>
                  <el-tag
                    type="info"
                    size="small"
                    style="margin-left: 4px;"
                  >
                    {{ getChatTypeLabel(agent.chat_type) }}
                  </el-tag>
                </div>
              </div>
            </div>
            <div class="agent-description" v-if="agent.description">
              {{ agent.description }}
            </div>
          </div>
          <el-empty
            v-if="!agentLoading && agentList.length === 0"
            description="暂无可用智能体"
            :image-size="80"
          />
        </div>
      </div>

      <!-- 右侧：目录详情内容 -->
      <div class="detail-content">
      <!-- ⭐ 权限不足提示：当目录没有任何权限时显示 -->
      <div v-if="hasNoDirectoryPermissions" class="permission-error-wrapper">
        <el-card class="permission-error-card" shadow="hover">
          <template #header>
            <div class="permission-error-header">
              <el-icon class="permission-error-icon"><Lock /></el-icon>
              <span class="permission-error-title">权限不足</span>
            </div>
          </template>
          <div class="permission-error-content">
            <div class="permission-error-message">
              <p class="error-message-text">
                您没有 <strong>访问该目录</strong> 的权限
              </p>
            </div>
            <div v-if="packageNode?.full_code_path" class="permission-error-info">
              <el-icon><Document /></el-icon>
              <span class="info-label">资源路径：</span>
              <span class="info-value">{{ packageNode.full_code_path }}</span>
            </div>
            <div class="permission-error-info">
              <el-icon><Key /></el-icon>
              <span class="info-label">缺少权限：</span>
              <span class="info-value">目录查看</span>
            </div>
            <div class="permission-error-actions">
              <el-button
                type="primary"
                size="default"
                @click="handleApplyPermission"
                :icon="Lock"
              >
                立即申请权限
              </el-button>
            </div>
          </div>
        </el-card>
      </div>

      <!-- 信息概览卡片 -->
      <div v-else-if="packageNode" class="overview-section">
        <div class="overview-card">
          <div class="overview-item">
            <div class="overview-icon-wrapper name-icon">
              <el-icon class="overview-icon"><Document /></el-icon>
            </div>
            <div class="overview-content">
              <div class="overview-label">目录名称</div>
              <div class="overview-value">{{ packageNode.name }}</div>
            </div>
          </div>

          <div class="overview-divider"></div>

          <div class="overview-item">
            <div class="overview-icon-wrapper code-icon">
              <el-icon class="overview-icon"><Key /></el-icon>
            </div>
            <div class="overview-content">
              <div class="overview-label">目录代码</div>
              <div class="overview-value code-text">{{ packageNode.code }}</div>
            </div>
          </div>

          <div class="overview-divider"></div>

          <div class="overview-item">
            <div class="overview-icon-wrapper count-icon">
              <el-icon class="overview-icon"><Files /></el-icon>
            </div>
            <div class="overview-content">
              <div class="overview-label">子项数量</div>
              <div class="overview-value">
                {{ packageNode?.children?.length || 0 }} 项
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 子目录和函数列表 -->
      <div class="children-section" v-if="!hasNoDirectoryPermissions && packageNode?.children && packageNode.children.length > 0">
        <div class="section-header">
          <h3 class="section-title">
            <el-icon class="section-icon"><Files /></el-icon>
            子目录和函数
          </h3>
          <el-tag class="section-badge" type="primary" size="small">
            {{ packageNode.children.length }}
          </el-tag>
        </div>

        <div class="children-grid">
          <div
            v-for="child in packageNode.children"
            :key="child.id"
            class="child-card"
            @click="handleChildClick(child)"
          >
            <div class="child-card-header">
              <div class="child-icon-wrapper" :class="child.type === 'package' ? 'package-type' : 'function-type'">
                <!-- package 类型：使用自定义文件夹图标 -->
                <img
                  v-if="child.type === 'package'"
                  src="/service-tree/custom-folder.svg"
                  alt="目录"
                  class="child-icon-img"
                />
                <!-- function 类型：根据 template_type 显示不同图标 -->
                <template v-else-if="child.type === 'function'">
                  <!-- 表单类型：使用编辑图标 -->
                  <img
                    v-if="child.template_type === TEMPLATE_TYPE.FORM"
                    src="/service-tree/编辑.svg"
                    alt="表单"
                    class="child-icon-img"
                  />
                  <!-- 其他类型：使用组件图标 -->
                  <el-icon v-else class="child-icon">
                    <component :is="getChildFunctionIcon(child)" />
                  </el-icon>
                </template>
                <!-- 默认图标 -->
                <el-icon v-else class="child-icon">
                  <Document />
                </el-icon>
              </div>
              <el-tag
                v-if="child.type === 'function'"
                size="small"
                :type="getTemplateTypeTag(child.template_type)"
                class="child-type-tag"
              >
                {{ getTemplateTypeText(child.template_type) }}
              </el-tag>
            </div>
            <div class="child-card-body">
              <div class="child-name">{{ child.name }}</div>
              <div class="child-description" v-if="child.description">
                {{ child.description }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <el-empty
        v-else
        description="该目录下暂无子目录或函数"
        :image-size="120"
        class="empty-state"
      />
      </div>
    </div>

    <!-- 变更记录对话框 -->
    <DirectoryUpdateHistoryDialog
      v-model="updateHistoryDialogVisible"
      mode="directory"
      :app-id="packageNode?.app_id || 0"
      :full-code-path="packageNode?.full_code_path || ''"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ArrowLeft, MagicStick, Folder, Document, CopyDocument, Key, Link, Files, Clock, Lock } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { ServiceTree } from '@/types'
import type { AgentInfo, AgentListReq } from '@/api/agent'
import { getAgentList } from '@/api/agent'
import { extractWorkspacePath } from '@/utils/route'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import ChartIcon from '@/components/icons/ChartIcon.vue'
import TableIcon from '@/components/icons/TableIcon.vue'
import FormIcon from '@/components/icons/FormIcon.vue'
import DirectoryUpdateHistoryDialog from '@/components/DirectoryUpdateHistoryDialog.vue'
import { buildPermissionApplyURL } from '@/utils/permission'
import { useNodePermissionsStore } from '@/stores/nodePermissions'

interface Props {
  packageNode?: ServiceTree | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'generate-system': [agent: AgentInfo]
}>()

const router = useRouter()
const route = useRoute()

// 智能体列表相关
const agentLoading = ref(false)
const agentList = ref<AgentInfo[]>([])

// 变更记录对话框
const updateHistoryDialogVisible = ref(false)

// ⭐ 权限 Store
const permissionStore = useNodePermissionsStore()

// ⭐ 检查是否没有任何目录权限
const hasNoDirectoryPermissions = computed(() => {
  if (!props.packageNode) {
    return false
  }
  
  // 优先从 store 中获取权限信息
  let permissions: Record<string, boolean> | null = null
  if (props.packageNode) {
    permissions = permissionStore.getPermissions(props.packageNode)
  }
  
  // 如果 store 中没有，尝试从节点本身的 permissions 字段获取
  if (!permissions && props.packageNode.permissions) {
    permissions = props.packageNode.permissions
  }
  
  // ⭐ 如果没有权限信息，也检查一下是否应该显示权限不足
  // 如果节点有 full_code_path，说明这是一个有效的目录节点，应该检查权限
  // 如果权限信息不存在，可能是还没有加载，暂时不显示权限不足（避免闪烁）
  if (!permissions) {
    // 如果节点有 full_code_path，说明这是一个有效的目录节点
    // 但是权限信息可能还没有加载，暂时返回 false（不显示权限不足）
    // 等权限信息加载后，computed 会自动重新计算
    return false
  }
  
  const directoryPermissions = [
    'directory:read',
    'directory:write',
    'directory:update',
    'directory:delete',
    'directory:manage'
  ]
  
  // 如果所有目录权限都是 false，则显示权限不足
  const hasNoPerms = directoryPermissions.every(perm => {
    // 如果权限字段不存在，也视为 false
    return permissions![perm] === false || permissions![perm] === undefined
  })
  
  return hasNoPerms
})

// 处理权限申请
function handleApplyPermission() {
  if (!props.packageNode?.full_code_path) {
    ElMessage.warning('路径信息不可用')
    return
  }
  
  // 跳转到权限申请页面，默认申请目录查看权限
  const applyURL = buildPermissionApplyURL(props.packageNode.full_code_path, 'directory:read', undefined)
  router.push(applyURL)
}

// 返回上一级
function handleBack() {
  // 获取当前路径，去掉最后一段
  const currentPath = extractWorkspacePath(route.path)
  if (currentPath) {
    const pathSegments = currentPath.split('/').filter(Boolean)
    if (pathSegments.length > 2) {
      // 至少是 user/app/package，去掉最后一段
      pathSegments.pop()
      const parentPath = `/workspace/${pathSegments.join('/')}`
      router.push(parentPath)
    } else {
      // 回到根目录
      router.push('/workspace')
    }
  } else {
    router.push('/workspace')
  }
}

// 加载智能体列表
async function loadAgents() {
  agentLoading.value = true
  try {
    const params: AgentListReq = {
      enabled: true,
      scope: 'market', // 显示市场中的公开智能体
      page: 1,
      page_size: 1000
    }
    const res = await getAgentList(params)
    // 响应拦截器已返回 data 部分，所以 res 就是 { agents, total }
    agentList.value = (res as any).agents || []
  } catch (error: any) {
    console.error('加载智能体列表失败:', error)
    ElMessage.error(error.message || '加载智能体列表失败')
    agentList.value = []
  } finally {
    agentLoading.value = false
  }
}

// 获取聊天类型标签
function getChatTypeLabel(chatType: string): string {
  const labels: Record<string, string> = {
    function_gen: '函数生成',
    'chat-task': '任务对话'
  }
  return labels[chatType] || chatType
}

// 获取智能体 Logo（如果有则使用，否则使用默认生成的）
function getAgentLogo(agent: AgentInfo): string {
  if (agent.logo) {
    return agent.logo
  }
  // 生成默认 Logo（使用智能体 ID 生成唯一颜色）
  return generateDefaultLogo(agent.id, agent.name)
}

// 生成默认 Logo URL（使用智能体 ID 生成唯一颜色）
function generateDefaultLogo(agentId: number, agentName: string): string {
  // 使用智能体 ID 生成一个稳定的颜色
  const colors = [
    '#409EFF', '#67C23A', '#E6A23C', '#F56C6C', '#909399',
    '#606266', '#303133', '#409EFF', '#67C23A', '#E6A23C'
  ]
  const colorIndex = agentId % colors.length
  const color = colors[colorIndex]

  // 生成 SVG data URL
  const svg = `
    <svg width="48" height="48" xmlns="http://www.w3.org/2000/svg">
      <rect width="48" height="48" fill="${color}" rx="8"/>
      <text x="24" y="32" font-family="Arial, sans-serif" font-size="20" font-weight="bold" fill="white" text-anchor="middle">${getAgentLogoText({ id: agentId, name: agentName } as AgentInfo)}</text>
    </svg>
  `.trim()

  return `data:image/svg+xml;base64,${btoa(unescape(encodeURIComponent(svg)))}`
}

// 获取智能体 Logo 文本（取名称首字符）
function getAgentLogoText(agent: AgentInfo): string {
  if (!agent.name) return 'A'
  // 取第一个字符（支持中文）
  const firstChar = agent.name.charAt(0)
  return firstChar.toUpperCase()
}

// 点击智能体（直接触发生成系统）
function handleAgentClick(agent: AgentInfo) {
  // 触发生成系统事件，让父组件处理
  emit('generate-system', agent)
}

// 复制完整路径
async function handleCopyPath() {
  if (!props.packageNode?.full_code_path) {
    ElMessage.warning('路径信息不可用')
    return
  }

  try {
    await navigator.clipboard.writeText(props.packageNode.full_code_path)
    ElMessage.success('路径已复制到剪贴板')
  } catch (error) {
    // 降级方案：使用传统方法
    const textArea = document.createElement('textarea')
    textArea.value = props.packageNode.full_code_path
    textArea.style.position = 'fixed'
    textArea.style.opacity = '0'
    document.body.appendChild(textArea)
    textArea.select()
    try {
      document.execCommand('copy')
      ElMessage.success('路径已复制到剪贴板')
    } catch (err) {
      ElMessage.error('复制失败，请手动复制')
    }
    document.body.removeChild(textArea)
  }
}

// 获取模板类型标签类型
function getTemplateTypeTag(templateType: string): string {
  const typeMap: Record<string, string> = {
    'table': 'success',
    'form': 'primary',
    'chart': 'warning'
  }
  return typeMap[templateType] || 'info'
}

// 获取模板类型文本
function getTemplateTypeText(templateType: string): string {
  const typeMap: Record<string, string> = {
    'table': '表格',
    'form': '表单',
    'chart': '图表'
  }
  return typeMap[templateType] || '函数'
}

// 获取子项函数图标组件（与左侧目录树保持一致）
function getChildFunctionIcon(child: ServiceTree) {
  if (child.template_type === TEMPLATE_TYPE.TABLE) {
    return TableIcon
  } else if (child.template_type === TEMPLATE_TYPE.FORM) {
    return FormIcon
  } else if (child.template_type === TEMPLATE_TYPE.CHART) {
    return ChartIcon
  }
  // 默认使用 Document 图标
  return Document
}

// 处理显示变更记录
function handleShowUpdateHistory(): void {
  emit('update-history', props.packageNode)
}

// 组件挂载时加载智能体列表
onMounted(() => {
  loadAgents()
})

// 处理子项点击（跳转到对应的目录或函数）
function handleChildClick(child: ServiceTree): void {
  console.log('🔍 [PackageDetailView.handleChildClick] 开始处理子项点击', {
    childName: child.name,
    childType: child.type,
    fullCodePath: child.full_code_path,
    currentPath: route.path,
    currentQuery: route.query
  })
  
  const applicationService = serviceFactory.getWorkspaceApplicationService()

  if (child.type === 'function' && child.full_code_path) {
    // 函数节点：跳转到函数页面
    const targetPath = `/workspace${child.full_code_path}`
    console.log('🔍 [PackageDetailView.handleChildClick] 函数节点', {
      targetPath,
      currentPath: route.path,
      pathMatch: route.path === targetPath
    })
    
    if (route.path !== targetPath) {
      // 触发节点点击，加载函数详情
      applicationService.triggerNodeClick(child)

      const preserveParams = {
        table: false,
        search: false,
        state: false,
        linkNavigation: false
      }
      
      console.log('🔍 [PackageDetailView.handleChildClick] 发出路由更新请求（函数）', {
        path: targetPath,
        query: {},
        queryKeys: Object.keys({}),
        queryLength: Object.keys({}).length,
        preserveParams,
        source: 'package-detail-child-click'
      })

      // 更新路由
      eventBus.emit(RouteEvent.updateRequested, {
        path: targetPath,
        query: {},
        replace: true,
        preserveParams,
        source: 'package-detail-child-click'
      })
    } else {
      // 路由已匹配，直接触发节点点击加载详情
      console.log('🔍 [PackageDetailView.handleChildClick] 路由已匹配，直接触发节点点击')
      applicationService.triggerNodeClick(child)
    }
  } else if (child.type === 'package' && child.full_code_path) {
    // 目录节点：跳转到目录详情页面
    console.log('🔍 [PackageDetailView.handleChildClick] 目录节点', {
      fullCodePath: child.full_code_path
    })
    
    applicationService.triggerNodeClick(child)

    const targetPath = `/workspace${child.full_code_path}`
    if (route.path !== targetPath) {
      const preserveParams = {
        table: false,
        search: false,
        state: false,
        linkNavigation: false
      }
      
      console.log('🔍 [PackageDetailView.handleChildClick] 发出路由更新请求（目录）', {
        path: targetPath,
        query: {},
        queryKeys: Object.keys({}),
        queryLength: Object.keys({}).length,
        preserveParams,
        source: 'package-detail-child-click-package'
      })
      
      eventBus.emit(RouteEvent.updateRequested, {
        path: targetPath,
        query: {},
        replace: true,
        preserveParams,
        source: 'package-detail-child-click-package'
      })
    }
  }
}
</script>

<style scoped lang="scss">
.package-detail-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);

  // 顶部横幅区域
  .hero-section {
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
    padding: 32px 40px;

    .hero-content {
      max-width: 1400px;
      margin: 0 auto;
      display: flex;
      align-items: center;
      gap: 24px;

      .back-button {
        flex-shrink: 0;
        background: var(--el-bg-color);
        border-color: var(--el-border-color);
        color: var(--el-text-color-regular);

        &:hover {
          background: var(--el-color-primary-light-9);
          border-color: var(--el-color-primary);
          color: var(--el-color-primary);
        }
      }

      .hero-info {
        flex: 1;
        display: flex;
        align-items: center;
        gap: 20px;
        min-width: 0;

        .hero-icon-wrapper {
          flex-shrink: 0;
          display: flex;
          align-items: flex-start;
          justify-content: center;
          padding-top: 4px;

          .hero-icon {
            font-size: 48px;
            color: var(--el-color-primary);
          }

          .hero-icon-img {
            width: 48px;
            height: 48px;
            object-fit: contain;
          }
        }

        .hero-text {
          flex: 1;
          min-width: 0;

          .hero-title {
            margin: 0 0 8px 0;
            font-size: 28px;
            font-weight: 700;
            color: var(--el-text-color-primary);
            line-height: 1.2;
          }

          .hero-subtitle {
            margin: 0 0 8px 0;
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 14px;
            color: var(--el-text-color-secondary);

            .path-icon {
              font-size: 16px;
              color: var(--el-color-primary);
            }

            .path-text {
              flex: 1;
              font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
              color: var(--el-text-color-regular);
              word-break: break-all;
            }

            .path-copy-btn {
              flex-shrink: 0;
              color: var(--el-text-color-secondary);

              &:hover {
                color: var(--el-color-primary);
              }
            }
          }

          .hero-description {
            margin: 0;
            font-size: 15px;
            color: var(--el-text-color-regular);
            line-height: 1.6;
            padding: 12px 16px;
            background: var(--el-fill-color-lighter);
            border-radius: 8px;
            border-left: 3px solid var(--el-color-primary);
          }
        }
      }
    }
  }

  // 主要内容区域：左右分栏
  .main-content {
    flex: 1;
    display: flex;
    overflow: hidden;

    // 左侧：智能体列表
    .agent-sidebar {
      width: 320px;
      flex-shrink: 0;
      background: var(--el-bg-color);
      border-right: 1px solid var(--el-border-color-lighter);
      display: flex;
      flex-direction: column;

      .sidebar-header {
        padding: 20px;
        border-bottom: 1px solid var(--el-border-color-lighter);

        .sidebar-title {
          margin: 0;
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 16px;
          font-weight: 600;
          color: var(--el-text-color-primary);

          .sidebar-icon {
            font-size: 18px;
            color: var(--el-color-primary);
          }
        }
      }

      .agent-list {
        flex: 1;
        overflow-y: auto;
        padding: 16px;
        display: flex;
        flex-direction: column;
        gap: 12px;

        .agent-card {
          background: var(--el-bg-color);
          border: 2px solid var(--el-border-color-light);
          border-radius: 12px;
          padding: 16px;
          cursor: pointer;
          transition: all 0.3s ease;
          display: flex;
          flex-direction: column;
          gap: 12px;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.06);

          &:hover {
            border-color: var(--el-color-primary);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
            transform: translateY(-2px);
          }

          &:active {
            transform: translateY(0);
            box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
          }

          .agent-card-header {
            display: flex;
            align-items: center;
            gap: 12px;

            .agent-avatar {
              flex-shrink: 0;
              border: 2px solid var(--el-border-color-lighter);
              box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);

              .agent-avatar-text {
                font-size: 20px;
                font-weight: bold;
                color: white;
              }
            }

            .agent-card-title {
              flex: 1;
              min-width: 0;

              .agent-name {
                font-size: 16px;
                font-weight: 600;
                color: var(--el-text-color-primary);
                margin-bottom: 6px;
                overflow: hidden;
                text-overflow: ellipsis;
                white-space: nowrap;
                line-height: 1.4;
              }

              .agent-tags {
                display: flex;
                align-items: center;
                gap: 6px;
                flex-wrap: wrap;
              }
            }
          }

          .agent-description {
            font-size: 13px;
            color: var(--el-text-color-regular);
            line-height: 1.5;
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
            overflow: hidden;
            text-overflow: ellipsis;
            padding-top: 8px;
            border-top: 1px solid var(--el-border-color-lighter);
          }
        }
      }
    }

    // 右侧：目录详情内容
    .detail-content {
      flex: 1;
      overflow-y: auto;
      padding: 32px 40px;
      min-width: 0;
      width: 100%;

      // 信息概览卡片
      .overview-section {
        margin-bottom: 32px;

        .overview-card {
          display: flex;
          align-items: center;
          background: var(--el-bg-color);
          border: 1px solid var(--el-border-color-lighter);
          border-radius: 16px;
          padding: 24px;
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);

          .overview-item {
            flex: 1;
            display: flex;
            align-items: center;
            gap: 16px;

            .overview-icon-wrapper {
              flex-shrink: 0;
              display: flex;
              align-items: center;
              justify-content: center;
              width: 48px;
              height: 48px;
              border-radius: 12px;

              &.name-icon {
                background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));

                .overview-icon {
                  font-size: 24px;
                  color: var(--el-color-primary);
                }
              }

              &.code-icon {
                background: linear-gradient(135deg, var(--el-color-success-light-8), var(--el-color-success-light-9));

                .overview-icon {
                  font-size: 24px;
                  color: var(--el-color-success);
                }
              }

              &.count-icon {
                background: linear-gradient(135deg, var(--el-color-warning-light-8), var(--el-color-warning-light-9));

                .overview-icon {
                  font-size: 24px;
                  color: var(--el-color-warning);
                }
              }
            }

            .overview-content {
              flex: 1;
              min-width: 0;

              .overview-label {
                font-size: 13px;
                color: var(--el-text-color-secondary);
                margin-bottom: 4px;
                font-weight: 500;
              }

              .overview-value {
                font-size: 18px;
                font-weight: 600;
                color: var(--el-text-color-primary);

                &.code-text {
                  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
                  color: var(--el-color-success);
                  font-size: 16px;
                }
              }
            }
          }

          .overview-divider {
            width: 1px;
            height: 48px;
            background: var(--el-border-color-lighter);
            margin: 0 24px;
          }
        }
      }

      // 子目录和函数区域
      .children-section {
        margin-top: 32px;

        .section-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: 20px;

          .section-title {
            margin: 0;
            display: flex;
            align-items: center;
            gap: 10px;
            font-size: 20px;
            font-weight: 600;
            color: var(--el-text-color-primary);

            .section-icon {
              font-size: 22px;
              color: var(--el-color-primary);
            }
          }

          .section-badge {
            font-weight: 600;
            padding: 4px 12px;
          }
        }

        .children-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
          gap: 16px;
          width: 100%;

          .child-card {
            background: var(--el-bg-color);
            border: 1px solid var(--el-border-color-lighter);
            border-radius: 12px;
            padding: 20px;
            transition: all 0.3s ease;
            cursor: pointer;
            width: 100%;
            box-sizing: border-box;

            &:hover {
              border-color: var(--el-color-primary-light-7);
              box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
              transform: translateY(-2px);
            }

            .child-card-header {
              display: flex;
              align-items: center;
              justify-content: space-between;
              margin-bottom: 16px;

              .child-icon-wrapper {
                display: flex;
                align-items: center;
                justify-content: center;
                width: 48px;
                height: 48px;
                border-radius: 12px;
                flex-shrink: 0;

                &.package-type {
                  background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));

                  .child-icon-img {
                    width: 32px;
                    height: 32px;
                    object-fit: contain;
                  }
                }

                &.function-type {
                  background: linear-gradient(135deg, var(--el-color-success-light-8), var(--el-color-success-light-9));

                  .child-icon {
                    font-size: 24px;
                    color: var(--el-color-success);
                  }

                  .child-icon-img {
                    width: 32px;
                    height: 32px;
                    object-fit: contain;
                  }
                }
              }

              .child-type-tag {
                font-weight: 500;
              }
            }

            .child-card-body {
              .child-name {
                font-size: 16px;
                font-weight: 600;
                color: var(--el-text-color-primary);
                line-height: 1.5;
                word-break: break-word;
                margin-bottom: 8px;
              }

              .child-description {
                font-size: 13px;
                color: var(--el-text-color-secondary);
                line-height: 1.6;
                word-break: break-word;
                padding-top: 8px;
                border-top: 1px solid var(--el-border-color-lighter);
              }
            }
          }
        }
      }

      .empty-state {
        margin-top: 60px;
      }

      // ⭐ 权限不足提示样式
      .permission-error-wrapper {
        display: flex;
        justify-content: center;
        align-items: center;
        min-height: 400px;
        padding: 40px 20px;
      }

      .permission-error-card {
        max-width: 600px;
        width: 100%;
        border-radius: 16px;
        border: none;
        box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
        transition: all 0.3s ease;

        &:hover {
          box-shadow: 0 6px 24px rgba(0, 0, 0, 0.12);
          transform: translateY(-2px);
        }
      }

      .permission-error-header {
        display: flex;
        align-items: center;
        gap: 12px;
        font-size: 18px;
        font-weight: 600;
        color: var(--el-color-warning);
      }

      .permission-error-icon {
        font-size: 24px;
      }

      .permission-error-title {
        font-size: 18px;
      }

      .permission-error-content {
        padding: 8px 0;
      }

      .permission-error-message {
        margin-bottom: 24px;
        padding: 16px;
        background: linear-gradient(135deg, rgba(255, 193, 7, 0.1) 0%, rgba(255, 152, 0, 0.05) 100%);
        border-radius: 12px;
        border-left: 4px solid var(--el-color-warning);
      }

      .error-message-text {
        margin: 0;
        font-size: 15px;
        line-height: 1.6;
        color: var(--el-text-color-primary);

        strong {
          color: var(--el-color-warning);
          font-weight: 600;
        }
      }

      .permission-error-info {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 16px;
        padding: 12px 16px;
        background: var(--el-bg-color-page);
        border-radius: 10px;
        font-size: 14px;
        transition: all 0.2s ease;

        &:hover {
          background: var(--el-fill-color-light);
        }

        .el-icon {
          color: var(--el-color-info);
          font-size: 18px;
        }

        .info-label {
          color: var(--el-text-color-regular);
          font-weight: 500;
        }

        .info-value {
          color: var(--el-text-color-primary);
          font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
          font-size: 13px;
          word-break: break-all;
        }
      }

      .permission-error-actions {
        margin-top: 24px;
        display: flex;
        justify-content: center;
        padding-top: 16px;
        border-top: 1px solid var(--el-border-color-lighter);
      }
    }
  }
}

// 响应式设计
@media (max-width: 768px) {
  .package-detail-view {
    .hero-section {
      padding: 24px 20px;

      .hero-content {
        flex-direction: column;
        align-items: stretch;
        gap: 16px;

        .hero-info {
          flex-direction: column;
          align-items: flex-start;
          gap: 16px;
        }

        .action-button {
          width: 100%;
        }
      }
    }

    .main-content {
      flex-direction: column;

      .agent-sidebar {
        width: 100%;
        border-right: none;
        border-bottom: 1px solid var(--el-border-color-lighter);
        max-height: 300px;
      }

      .detail-content {
        padding: 24px 20px;

        .overview-section {
          .overview-card {
            flex-direction: column;
            gap: 20px;

            .overview-divider {
              width: 100%;
              height: 1px;
              margin: 0;
            }
          }
        }

        .children-section {
          .children-grid {
            grid-template-columns: 1fr;
          }
        }
      }
    }
  }
}
</style>

