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
              <el-button
                v-if="canExport"
                text
                :icon="Download"
                @click="handleExportJson"
                class="path-export-btn"
                size="small"
                title="导出 JSON"
              >
                导出 JSON
              </el-button>
              <el-button
                v-if="canImportCapability"
                text
                :icon="Upload"
                @click="requestCapabilityJsonImport"
                class="path-import-btn"
                size="small"
                title="导入 JSON"
              >
                导入 JSON
              </el-button>
            </p>
          </div>
        </div>
      </div>
    </div>
    <input
      ref="capabilityImportInputRef"
      type="file"
      accept=".json,application/json"
      class="capability-import-input"
      @change="handleCapabilityImportFileChange"
    />

    <!-- 主要内容区域 -->
    <div class="main-content">
      <PackageDetailContent
        :package-node="packageNode || null"
        :total-run-count="totalRunCount"
        :has-no-directory-permissions="hasNoDirectoryPermissions"
        :show-permission-request-tab="showPermissionRequestTab"
        :can-edit="canEdit"
        :active-tab="activeTab"
        :is-import-go-dragging="isImportGoDragging"
        :resource-type="resourceType"
        @update:active-tab="activeTab = $event"
        @apply-permission="handleApplyPermission"
        @select-child="handleChildClick"
        @import-go-drop="onImportGoDrop"
        @set-import-go-dragging="isImportGoDragging = $event"
        @open-session="$emit('open-session', $event)"
      />
    </div>

    <PackageDetailEditDialog
      v-model:visible="editDialogVisible"
      :form="editForm"
      :submitting="editSubmitting"
      :admins-field="adminsField"
      :admins-field-value="editAdminsFieldValue"
      @update-admins="handleEditAdminsChange"
      @submit="handleSubmitEdit"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import type { LocationQueryValue } from 'vue-router'
import { ArrowLeft, Folder, CopyDocument, Link, Edit, Download, Upload } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { ServiceTree } from '@/types'
import { extractWorkspacePath } from '@/utils/route'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import { buildPermissionApplyURL, DirectoryPermission, hasPermission } from '@/utils/permission'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import { WidgetType } from '@/core/constants/widget'
import { useAuthStore } from '@/stores/auth'
import { updatePackage, addFunctionsToDirectory, exportCapabilityBundle, installCapabilityBundle } from '@/api/service-tree'
import { downloadCapabilityBundleFile, parseCapabilityBundleJson } from '@/utils/directoryBundleFile'
import { isServiceTreeNodeAdmin } from '@/utils/permissionActors'
import { Logger } from '@/core/utils/logger'
import PackageDetailContent from './PackageDetailContent.vue'
import PackageDetailEditDialog from './PackageDetailEditDialog.vue'
import type { WorkspaceSessionItem } from '@/api/workspace'

type DetailTabName = 'info' | 'import' | 'permissionRequest' | 'permissionManage' | 'scheduledAgentTask'

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
const authStore = useAuthStore() // ⭐ 必须在 showPermissionRequestTab 之前初始化

// Tab 相关
const activeTab = ref<DetailTabName>('info')
const isImportGoDragging = ref(false)
const importGoLoading = ref(false)
const capabilityImportInputRef = ref<HTMLInputElement | null>(null)

// ⭐ 判断是否显示权限申请 tab
// 条件：1. 节点类型是 package 或 app  2. 用户是管理员
const showPermissionRequestTab = computed(() => {
  if (!props.packageNode) {
    return false
  }
  
  // 必须是 package 类型
  if (props.packageNode.type !== 'package') {
    return false
  }
  
  return isServiceTreeNodeAdmin(props.packageNode, authStore.user?.username)
})

// ⭐ 计算资源类型（用于权限组件）
// ⭐ 所有 package 类型统一使用 directory 资源类型（包括根目录/工作空间）
const resourceType = computed<'directory'>(() => {
  return 'directory'
})

const showScheduledAgentTaskTab = computed(() => {
  return props.packageNode?.type === 'package' && !!props.packageNode.full_code_path
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
    } else if (normalizedTab === 'permissionRequest' && showPermissionRequestTab.value) {
      activeTab.value = 'permissionRequest'
    } else if (normalizedTab === 'permissionManage' && showPermissionRequestTab.value) {
      activeTab.value = 'permissionManage'
    }
  },
  { immediate: true }
)

// 编辑对话框
const editDialogVisible = ref(false)
const editSubmitting = ref(false)
const editForm = ref({
  name: '',
  admins: ''
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

const canExport = computed(() => {
  return props.packageNode?.type === 'package'
    && !!props.packageNode.full_code_path
    && hasPermission(props.packageNode, DirectoryPermission.read)
})

const canImportCapability = computed(() => {
  return props.packageNode?.type === 'package'
    && !!props.packageNode.full_code_path
    && hasPermission(props.packageNode, DirectoryPermission.write)
})

watch(
  () => [
    props.packageNode?.full_code_path,
    showPermissionRequestTab.value,
    showScheduledAgentTaskTab.value,
    canEdit.value
  ] as const,
  () => {
    if ((activeTab.value === 'permissionRequest' || activeTab.value === 'permissionManage') && !showPermissionRequestTab.value) {
      activeTab.value = showScheduledAgentTaskTab.value ? 'scheduledAgentTask' : 'info'
      return
    }
    if (activeTab.value === 'scheduledAgentTask' && !showScheduledAgentTaskTab.value) {
      activeTab.value = 'info'
      return
    }
    if (activeTab.value === 'import' && (!canEdit.value || !props.packageNode?.full_code_path)) {
      activeTab.value = 'info'
    }
  },
  { immediate: true }
)

// 管理员字段配置（用于 UsersWidget）
const adminsField = computed<FieldConfig>(() => ({
  code: 'admins',
  name: '管理员',
  widget: {
    type: WidgetType.USERS,
    config: {}
  }
}))

// ⭐ 检查是否没有任何权限（根据节点类型检查对应的权限）
const hasNoDirectoryPermissions = computed(() => {
  if (!props.packageNode) {
    return false
  }
  
  // 直接使用节点上的权限信息（后端返回的最新数据，已包含继承）
  const permissions = props.packageNode.permissions
  
  // 🔥 修复：如果没有权限信息或权限为空对象，返回 false（不显示权限不足）
  // 避免空 map 导致的无限循环问题
  if (!permissions || Object.keys(permissions).length === 0) {
    return false
  }
  
  // ⭐ 所有 package 类型统一检查 directory 权限（包括根目录/工作空间）
  const permissionsToCheck: string[] = [
    DirectoryPermission.read,
    DirectoryPermission.write,
    DirectoryPermission.update,
    DirectoryPermission.delete,
    DirectoryPermission.admin
  ]
  
  // 如果所有权限都是 false，则显示权限不足
  const hasNoPerms = permissionsToCheck.every(perm => {
    // 如果权限字段不存在，也视为 false
    return permissions[perm] === false || permissions[perm] === undefined
  })
  
  return hasNoPerms
})

// 处理权限申请
function handleApplyPermission() {
  if (!props.packageNode?.full_code_path) {
    ElMessage.warning('路径信息不可用')
    return
  }
  
  // ⭐ 所有 package 类型统一申请 directory:read 权限（包括根目录/工作空间）
  const defaultAction = DirectoryPermission.read
  
  // 跳转到权限申请页面
  const applyURL = buildPermissionApplyURL(props.packageNode.full_code_path, defaultAction, undefined)
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

async function handleExportJson() {
  const fullCodePath = props.packageNode?.full_code_path
  if (!fullCodePath) {
    ElMessage.warning('路径信息不可用')
    return
  }

  try {
    const bundle = await exportCapabilityBundle({
      source_directory_path: fullCodePath,
      name: props.packageNode?.name || props.packageNode?.code
    })
    downloadCapabilityBundleFile(bundle, fullCodePath)
    ElMessage.success('已开始下载 JSON 文件')
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || '导出失败'
    ElMessage.error(message)
  }
}

function requestCapabilityJsonImport() {
  if (!props.packageNode?.full_code_path) {
    ElMessage.warning('路径信息不可用')
    return
  }
  if (capabilityImportInputRef.value) {
    capabilityImportInputRef.value.value = ''
    capabilityImportInputRef.value.click()
  }
}

async function handleCapabilityImportFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  const fullCodePath = props.packageNode?.full_code_path
  if (!file || !fullCodePath) {
    return
  }

  try {
    const bundle = parseCapabilityBundleJson(await readFileAsText(file))
    await ElMessageBox.confirm(
      `将能力包「${bundle.name || file.name}」导入到 ${fullCodePath}，同名文件会被覆盖。`,
      '导入 JSON',
      {
        confirmButtonText: '覆盖导入',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    const resp = await installCapabilityBundle({
      target_directory_path: fullCodePath,
      overwrite: true,
      force_diff: true,
      bundle
    })
    ElMessage.success(resp.message || '导入成功')
    emit('refresh')
  } catch (error: any) {
    if (error === 'cancel' || error === 'close') {
      return
    }
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || '导入失败'
    ElMessage.error(message)
  }
}

function readFileAsText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error)
    reader.readAsText(file, 'utf-8')
  })
}

async function onImportGoDrop(e: DragEvent) {
  isImportGoDragging.value = false
  const fullCodePath = props.packageNode?.full_code_path
  if (!fullCodePath) return
  const files = e.dataTransfer?.files
  if (!files?.length) return
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
        else {
          fail++
          Logger.warn('[PackageDetailView]', '导入 Go 文件失败', {
            fullCodePath,
            fileName,
            error: res?.error
          })
        }
      } catch (err: any) {
        fail++
        ElMessage.warning(`${file.name}: ${err?.message || err?.response?.data?.msg || '写入失败'}`)
      }
    }
    if (ok > 0) {
      ElMessage.success(`已导入 ${ok} 个 Go 文件，可在工作台执行编译以生效。`)
      emit('refresh')
    }
    if (fail > 0 && ok === 0) ElMessage.error('导入失败')
  } finally {
    importGoLoading.value = false
  }
}

// 编辑表单的管理员字段值
const editAdminsFieldValue = computed<FieldValue>(() => {
  if (!editForm.value.admins || !editForm.value.admins.trim()) {
    return {
      raw: null,
      display: '',
      meta: {}
    }
  }
  
  const admins = editForm.value.admins.split(',').map((s: string) => s.trim()).filter((s: string) => Boolean(s))
  return {
    raw: admins.join(','),
    display: admins.join(', '),
    meta: {}
  }
})

// 处理编辑表单中管理员字段的变化
function handleEditAdminsChange(value: FieldValue): void {
  if (value.raw) {
    editForm.value.admins = typeof value.raw === 'string' ? value.raw : String(value.raw)
  } else {
    editForm.value.admins = ''
  }
}

// 处理编辑按钮点击
function handleEdit(): void {
  if (!props.packageNode) {
    return
  }
  
  // 初始化编辑表单
  editForm.value = {
    name: props.packageNode.name || '',
    admins: props.packageNode.admins || ''
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
      name: editForm.value.name.trim(),
      admins: editForm.value.admins.trim()
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

  // 导入 Go 文件 tab 内
  .import-tab-content {
    padding: 24px 0;
  }
  .import-go-drop-zone {
    padding: 24px 16px;
    border: 1px dashed var(--el-border-color);
    border-radius: 8px;
    font-size: 14px;
    color: var(--el-color-primary);
    text-align: center;
    transition: border-color 0.2s, background 0.2s;
    background: var(--el-fill-color-lighter);
  }
  .import-go-drop-zone--dragover {
    border-color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  .import-tab-hint {
    margin: 12px 0 0;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

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
            .path-edit-btn,
            .path-export-btn,
            .path-import-btn {
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

.capability-import-input {
  display: none;
}
</style>
