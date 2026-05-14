<!--
  PackageDetailView - 服务目录详情页面

  职责：
  - 显示服务目录信息
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
                v-if="canEdit"
                text
                :icon="Edit"
                @click="handleEdit"
                class="path-edit-btn"
                size="small"
                title="编辑目录"
              >
                编辑
              </el-button>
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- 主要内容区域 -->
    <div class="main-content">
      <PackageDetailContent
        :package-node="packageNode || null"
        :total-run-count="totalRunCount"
        :active-tab="activeTab"
        @update:active-tab="activeTab = $event"
        @select-child="handleChildClick"
        @open-session="$emit('open-session', $event)"
      />
    </div>

    <PackageDetailEditDialog
      v-model:visible="editDialogVisible"
      :form="editForm"
      :submitting="editSubmitting"
      @submit="handleSubmitEdit"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import type { LocationQueryValue } from 'vue-router'
import { ArrowLeft, Folder, CopyDocument, Link, Edit } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { ServiceTree } from '@/architecture/domain/types'
import { extractWorkspacePath } from '@/architecture/runtime/utils/route'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import { useAuthStore } from '@/architecture/infrastructure/stores/auth'
import { updatePackage } from '@/architecture/infrastructure/api/service-tree'
import { Logger } from '@/architecture/runtime/utils/logger'
import PackageDetailContent from './PackageDetailContent.vue'
import PackageDetailEditDialog from './PackageDetailEditDialog.vue'
import type { WorkspaceSessionItem } from '@/architecture/infrastructure/api/workspace'
import { featureFlags } from '@/architecture/infrastructure/config/features'

type DetailTabName = 'info' | 'scheduledAgentTask'

interface Props {
  packageNode?: ServiceTree | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'refresh': []
  'open-session': [session: WorkspaceSessionItem]
}>()

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

// Tab 相关
const activeTab = ref<DetailTabName>('info')

const showScheduledAgentTaskTab = computed(() => {
  return featureFlags.scheduledTasks && props.packageNode?.type === 'package' && !!props.packageNode.full_code_path
})

// ⭐ 本目录下所有函数调用次数之和（仅统计直接子节点中的 function）
const totalRunCount = computed(() => {
  const children = props.packageNode?.children
  if (!children?.length) return 0
  return children
    .filter((c: ServiceTree) => c.type === 'function')
    .reduce((sum: number, c: ServiceTree) => sum + (c.run_count ?? 0), 0)
})

function normalizeTabQuery(tab: LocationQueryValue | LocationQueryValue[] | undefined): string | null {
  if (Array.isArray(tab)) {
    return tab[0] ?? null
  }

  return typeof tab === 'string' ? tab : null
}

// ⭐ 监听路由 query 参数，支持通过 _panel 参数指定要打开的 tab
watch(
  () => route.query._panel,
  (tab) => {
    const normalizedTab = normalizeTabQuery(tab)

    if (normalizedTab === 'scheduledAgentTask' && showScheduledAgentTaskTab.value) {
      activeTab.value = 'scheduledAgentTask'
    }
  },
  { immediate: true }
)

// 编辑对话框
const editDialogVisible = ref(false)
const editSubmitting = ref(false)
const editForm = ref({
  name: ''
})

// ⭐ 检查是否可以编辑（owner 或 admins 可以编辑）
const canEdit = computed(() => {
  if (!props.packageNode || !authStore.user?.username) {
    return false
  }
  
  const currentUser = authStore.user.username
  
  // 检查是否是 owner
  if (props.packageNode.owner && props.packageNode.owner.trim() === currentUser) {
    return true
  }
  
  // 检查是否是 admins 之一
  if (props.packageNode.admins && props.packageNode.admins.trim()) {
    const admins = props.packageNode.admins.split(',').map((s: string) => s.trim()).filter((s: string) => Boolean(s))
    if (admins.includes(currentUser)) {
      return true
    }
  }
  
  return false
})

watch(
  () => [
    props.packageNode?.full_code_path,
    showScheduledAgentTaskTab.value
  ] as const,
  () => {
    if (activeTab.value === 'scheduledAgentTask' && !showScheduledAgentTaskTab.value) {
      activeTab.value = 'info'
      return
    }
  },
  { immediate: true }
)

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

// 处理编辑按钮点击
function handleEdit(): void {
  if (!props.packageNode) {
    return
  }
  
  // 初始化编辑表单
  editForm.value = {
    name: props.packageNode.name || ''
  }
  
  editDialogVisible.value = true
}

// 提交编辑
async function handleSubmitEdit(editFormRef: any): Promise<void> {
  if (!props.packageNode) {
    return
  }
  
  // 表单验证
  if (!editFormRef.value) {
    return
  }
  
  try {
    await editFormRef.value.validate()
  } catch (error) {
    return
  }
  
  editSubmitting.value = true
  try {
    await updatePackage(props.packageNode.id, {
      name: editForm.value.name.trim()
    })
    
    ElMessage.success('更新成功')
    editDialogVisible.value = false
    
    // 触发刷新（通过 emit 事件或直接刷新）
    // 这里可以通过 emit 通知父组件刷新，或者直接刷新当前页面数据
    // 暂时先关闭对话框，父组件可以通过 watch packageNode 来刷新
    // 或者我们可以 emit 一个事件让父组件处理刷新
    emit('refresh')
  } catch (error: any) {
    Logger.error('[PackageDetailView]', '更新目录失败', {
      packageId: props.packageNode.id,
      error
    })
    ElMessage.error(error.message || '更新目录失败')
  } finally {
    editSubmitting.value = false
  }
}

// 处理子项点击（跳转到对应的目录或函数）
function handleChildClick(child: ServiceTree): void {
  const serviceProvider: IServiceProvider = serviceFactory
  const applicationService = serviceProvider.getWorkspaceApplicationService()

  if (child.type === 'function' && child.full_code_path) {
    // 函数节点：跳转到函数页面
    const targetPath = `/workspace${child.full_code_path}`

    if (route.path !== targetPath) {
      // 触发节点点击，加载函数详情
      applicationService.triggerNodeClick(child)

      const preserveParams = {
        table: false,
        search: false,
        state: false,
        linkNavigation: false
      }

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
      applicationService.triggerNodeClick(child)
    }
  } else if (child.type === 'package' && child.full_code_path) {
    // 目录节点：跳转到目录详情页面
    applicationService.triggerNodeClick(child)

    const targetPath = `/workspace${child.full_code_path}`
    if (route.path !== targetPath) {
      const preserveParams = {
        table: false,
        search: false,
        state: false,
        linkNavigation: false
      }

      eventBus.emit(RouteEvent.updateRequested, {
        path: targetPath,
        query: {},
        replace: true,
        preserveParams,
        source: 'package-detail-child-click-package'
      })
    }
  } else if ((child.type === 'board' || child.type === 'docs') && child.full_code_path) {
    // 讨论区/文档节点：跳转到对应页面
    applicationService.triggerNodeClick(child)
    const targetPath = `/workspace${child.full_code_path}`
    if (route.path !== targetPath) {
      eventBus.emit(RouteEvent.updateRequested, {
        path: targetPath,
        query: {},
        replace: true,
        preserveParams: { table: false, search: false, state: false, linkNavigation: false },
        source: 'package-detail-child-click-board-docs'
      })
    } else {
      applicationService.triggerNodeClick(child)
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
            flex-wrap: wrap;
            gap: 8px;
            font-size: 14px;
            color: var(--el-text-color-secondary);

            .path-icon {
              font-size: 16px;
              color: var(--el-color-primary);
            }

            .path-text {
              flex: 1 1 320px;
              min-width: 0;
              font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
              color: var(--el-text-color-regular);
              word-break: break-all;
            }

	            .path-copy-btn,
	            .path-edit-btn {
              flex-shrink: 0;
              color: var(--el-text-color-secondary);

              &:hover {
                color: var(--el-color-primary);
              }
            }
          }

        }
      }
    }
  }

  // 主要内容区域
.main-content {
  flex: 1;
  display: flex;
  overflow: hidden;
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

      .detail-content {
        padding: 24px 20px;
      }
    }
  }
}

</style>
